package main

// Persistent background-removal queue.  It deliberately mirrors the Whisper
// queue's lifecycle, while keeping its table and worker independent so an
// image job can never interfere with audio transcription.

import (
	"bufio"
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	rembgDefaultModel  = "u2net"
	rembgTaskQueued    = "queued"
	rembgTaskRunning   = "running"
	rembgTaskCompleted = "completed"
	rembgTaskCancelled = "cancelled"
	rembgTaskFailed    = "failed"
	rembgTaskLogLimit  = 128 * 1024
	// Loading rembg imports its Python image/ML dependencies and can exceed the
	// generic CLI probe timeout on a cold start.
	rembgProbeTimeout        = time.Minute
	rembgModelMirrorBaseURL  = "https://ghproxy.net/"
	rembgReleaseAssetBaseURL = "https://github.com/danielgatis/rembg/releases/download/v0.0.0/"
	rembgModelHeaderTimeout  = 30 * time.Second
	rembgModelIdleTimeout    = 90 * time.Second
)

type rembgModelArtifact struct {
	Filename string
	MD5      string
}

// These checksums are the hashes rembg itself uses for the models exposed by
// this queue. The mirror is only allowed to populate the cache after the
// downloaded bytes match the upstream artifact exactly.
var rembgModelArtifacts = map[string]rembgModelArtifact{
	"u2net":             {Filename: "u2net.onnx", MD5: "60024c5c889badc19c04ad937298a77b"},
	"u2net_human_seg":   {Filename: "u2net_human_seg.onnx", MD5: "c09ddc2e0104f800e3e1bb4652583d1f"},
	"u2netp":            {Filename: "u2netp.onnx", MD5: "8e83ca70e441ab06c318d82300c84806"},
	"u2net_cloth_seg":   {Filename: "u2net_cloth_seg.onnx", MD5: "2434d1f3cb744e0e49386c906e5a08bb"},
	"silueta":           {Filename: "silueta.onnx", MD5: "55e59e0d8062d2f5d013f4725ee84782"},
	"isnet-general-use": {Filename: "isnet-general-use.onnx", MD5: "fc16ebd8b0c10d971d3513d564d01e29"},
	"isnet-anime":       {Filename: "isnet-anime.onnx", MD5: "6f184e756bb3bd901c8849220a83e38e"},
}

type rembgCheckConfig struct {
	CacheFor time.Duration
	Install  string
}
type rembgCheckResponse struct {
	Available bool   `json:"available"`
	Install   string `json:"install,omitempty"`
	Content   string `json:"content,omitempty"`
	Status    int    `json:"status"`
}
type rembgTask struct {
	ID           int64  `json:"id"`
	AgentID      string `json:"agentId"`
	SourcePath   string `json:"sourcePath"`
	OutputPath   string `json:"outputPath,omitempty"`
	SavedAs      string `json:"savedAs,omitempty"`
	Model        string `json:"model"`
	AlphaMatting bool   `json:"alphaMatting"`
	Status       string `json:"status"`
	Progress     int    `json:"progress"`
	StartedAt    string `json:"startedAt,omitempty"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
	Logs         string `json:"logs,omitempty"`
	CancelAsked  bool   `json:"cancelRequested"`
}
type rembgTaskPage struct {
	Tasks    []rembgTask `json:"tasks"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
	Status   string      `json:"taskStatus"`
}
type rembgTaskCreateRequest struct {
	AgentID string                `json:"agentId"`
	Tasks   []rembgTaskCreateItem `json:"tasks,omitempty"`
	// Paths and Model keep the original API usable by older desktop clients.
	Paths        []string `json:"paths,omitempty"`
	Model        string   `json:"model,omitempty"`
	AlphaMatting bool     `json:"alphaMatting,omitempty"`
}
type rembgTaskCreateItem struct {
	Path         string `json:"path"`
	Model        string `json:"model,omitempty"`
	AlphaMatting bool   `json:"alphaMatting,omitempty"`
}
type rembgTaskActionRequest struct {
	AgentID string `json:"agentId"`
	ID      int64  `json:"id"`
}
type rembgTaskManager struct {
	cfg       *Config
	db        *sql.DB
	createMu  sync.Mutex
	mu        sync.Mutex
	runningID int64
	cancel    context.CancelFunc
	wake      chan struct{}
}

var (
	rembgTasks          *rembgTaskManager
	rembgLookPath       = exec.LookPath
	rembgCommandContext = exec.CommandContext
	rembgNow            = time.Now
	rembgUserHomeDir    = os.UserHomeDir
	rembgGlob           = mustGlob
	rembgHTTPClient     = &http.Client{Transport: &http.Transport{ResponseHeaderTimeout: rembgModelHeaderTimeout}}
	rembgCheckCache     struct {
		sync.Mutex
		checkedAt  time.Time
		executable string
	}
)

func rembgTimestamp() string { return rembgNow().Format(time.RFC3339Nano) }

func ensureRembgTaskSchema(db *sql.DB) error {
	if db == nil {
		return errors.New("rembg task database is not ready")
	}
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS rembg_task (
		id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL, source_path TEXT NOT NULL,
		output_path TEXT NOT NULL DEFAULT '', saved_as TEXT NOT NULL DEFAULT '', model TEXT NOT NULL DEFAULT 'u2net', alpha_matting INTEGER NOT NULL DEFAULT 0, status TEXT NOT NULL DEFAULT 'queued',
		progress INTEGER NOT NULL DEFAULT 0, started_at TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL, logs TEXT NOT NULL DEFAULT '', cancel_requested INTEGER NOT NULL DEFAULT 0
	); CREATE INDEX IF NOT EXISTS idx_rembg_task_agent_created ON rembg_task(agent_id, id DESC);
	CREATE INDEX IF NOT EXISTS idx_rembg_task_queue ON rembg_task(status, cancel_requested, id);`)
	if err != nil {
		return err
	}
	// The first release of the queue did not persist model parameters. Migrate
	// existing application databases without invalidating their queued jobs.
	columns := map[string]bool{}
	rows, err := db.Query(`PRAGMA table_info(rembg_task)`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var defaultValue interface{}
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			_ = rows.Close()
			return err
		}
		columns[name] = true
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if !columns["model"] {
		if _, err := db.Exec(`ALTER TABLE rembg_task ADD COLUMN model TEXT NOT NULL DEFAULT 'u2net'`); err != nil {
			return err
		}
	}
	if !columns["alpha_matting"] {
		if _, err := db.Exec(`ALTER TABLE rembg_task ADD COLUMN alpha_matting INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	_, err = db.Exec(`UPDATE rembg_task SET model=? WHERE trim(model)=''`, rembgDefaultModel)
	return err
}

func normalizeRembgModel(raw string) (string, bool) {
	model := strings.TrimSpace(raw)
	if model == "" {
		return rembgDefaultModel, true
	}
	switch model {
	case "u2net_human_seg", "u2net", "u2netp", "u2net_cloth_seg", "silueta", "isnet-general-use", "isnet-anime":
		return model, true
	default:
		return "", false
	}
}

func parseRembgCheckConfig(raw map[string]interface{}) (rembgCheckConfig, error) {
	value, ok := raw["rembg"]
	if !ok {
		return rembgCheckConfig{}, errors.New("config/config.json.rembg 配置缺失")
	}
	section, ok := value.(map[string]interface{})
	if !ok || section == nil {
		return rembgCheckConfig{}, errors.New("config/config.json.rembg 必须是对象")
	}
	hours, ok := ffmpegCheckPositiveHours(section["check"])
	if !ok {
		return rembgCheckConfig{}, errors.New("config/config.json.rembg.check 必须是正整数（小时）")
	}
	install, ok := section["install"].(string)
	install = strings.TrimSpace(install)
	if !ok || install == "" {
		return rembgCheckConfig{}, errors.New("config/config.json.rembg.install 必须是非空字符串")
	}
	return rembgCheckConfig{CacheFor: time.Duration(hours) * time.Hour, Install: install}, nil
}

// rembgExecutableCandidates discovers console scripts that can be installed by
// Python without being present in the desktop application's PATH. In
// particular, macOS GUI apps do not load shell startup files, so a successful
// `pip install --user rembg` must not depend on the user having exported its
// Python bin directory in the app's inherited environment.
func rembgExecutableCandidates() []string {
	candidates := make([]string, 0, 8)
	seen := make(map[string]bool)
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		absolute, err := filepath.Abs(path)
		if err != nil || seen[absolute] {
			return
		}
		info, err := os.Stat(absolute)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
			return
		}
		seen[absolute] = true
		candidates = append(candidates, absolute)
	}
	if path, err := rembgLookPath("rembg"); err == nil {
		add(path)
	}
	if home, err := rembgUserHomeDir(); err == nil {
		// Standard locations for `pip install --user` on Linux and macOS.
		add(filepath.Join(home, ".local", "bin", "rembg"))
		for _, path := range rembgGlob(filepath.Join(home, "Library", "Python", "*", "bin", "rembg")) {
			add(path)
		}
	}
	// Python.org's macOS installer puts console scripts under this framework
	// directory. It is intentionally checked explicitly because it is normally
	// absent from the environment of an application launched from Finder.
	for _, path := range rembgGlob(filepath.Join(string(filepath.Separator), "Library", "Frameworks", "Python.framework", "Versions", "*", "bin", "rembg")) {
		add(path)
	}
	return candidates
}

func rembgExecutable(cacheFor time.Duration) (string, bool) {
	now := rembgNow()
	candidates := rembgExecutableCandidates()
	if len(candidates) == 0 {
		return "", false
	}
	rembgCheckCache.Lock()
	cachedAt, cached := rembgCheckCache.checkedAt, rembgCheckCache.executable
	rembgCheckCache.Unlock()
	if cached != "" && now.Before(cachedAt.Add(cacheFor)) {
		for _, path := range candidates {
			if path == cached {
				return path, true
			}
		}
	}
	// Verify the precise executable that will later run the task. This avoids a
	// stale Python wrapper making the UI report that rembg is available.
	for _, path := range candidates {
		probeCtx, cancel := context.WithTimeout(context.Background(), rembgProbeTimeout)
		err := rembgCommandContext(probeCtx, path, "--help").Run()
		timedOut := probeCtx.Err() != nil
		cancel()
		if err != nil || timedOut {
			continue
		}
		rembgCheckCache.Lock()
		rembgCheckCache.checkedAt, rembgCheckCache.executable = now, path
		rembgCheckCache.Unlock()
		return path, true
	}
	return "", false
}

func writeRembgCheckResponse(w http.ResponseWriter, code int, response rembgCheckResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(response)
}
func handleRembgCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeRembgCheckResponse(w, http.StatusMethodNotAllowed, rembgCheckResponse{Content: "仅支持 GET 请求", Status: 1})
			return
		}
		status, response := checkRembgDependency()
		writeRembgCheckResponse(w, status, response)
	}
}

// checkRembgDependency contains the complete availability rule shared by the
// HTTP endpoint and the asynchronous startup preflight.
func checkRembgDependency() (int, rembgCheckResponse) {
	raw, _, err := readIntegrationStartupConfigRaw()
	if err != nil {
		return http.StatusInternalServerError, rembgCheckResponse{Content: "读取 config/config.json 失败: " + err.Error(), Status: 1}
	}
	cfg, err := parseRembgCheckConfig(raw)
	if err != nil {
		return http.StatusBadRequest, rembgCheckResponse{Content: err.Error(), Status: 1}
	}
	if _, ok := rembgExecutable(cfg.CacheFor); !ok {
		return http.StatusOK, rembgCheckResponse{Install: cfg.Install, Content: "未检测到可用的 rembg", Status: 0}
	}
	return http.StatusOK, rembgCheckResponse{Available: true, Status: 0}
}

func startRembgTaskManager(ctx context.Context, cfg *Config) {
	if cronDB == nil {
		log.Printf("[rembg] task queue unavailable: database is not ready")
		return
	}
	if err := ensureRembgTaskSchema(cronDB); err != nil {
		log.Printf("[rembg] ensure task schema failed: %v", err)
		return
	}
	m := &rembgTaskManager{cfg: cfg, db: cronDB, wake: make(chan struct{}, 1)}
	if err := m.recover(); err != nil {
		log.Printf("[rembg] recover interrupted tasks failed: %v", err)
	}
	rembgTasks = m
	go m.run(ctx)
}
func (m *rembgTaskManager) recover() error {
	now := rembgTimestamp()
	_, err := m.db.Exec(`UPDATE rembg_task SET status=?,progress=0,started_at='',updated_at=?,logs=CASE WHEN length(logs)>? THEN substr(logs,-?) ELSE logs END || ? WHERE status=? AND cancel_requested=0`, rembgTaskQueued, now, rembgTaskLogLimit-2048, rembgTaskLogLimit/2, "\n[服务重启] 上次执行被中断，任务已恢复为排队中。\n", rembgTaskRunning)
	if err != nil {
		return err
	}
	_, err = m.db.Exec(`UPDATE rembg_task SET status=?,updated_at=? WHERE status=? AND cancel_requested=1`, rembgTaskCancelled, now, rembgTaskRunning)
	return err
}
func (m *rembgTaskManager) signal() {
	select {
	case m.wake <- struct{}{}:
	default:
	}
	notifyIntegrationSleepConditionChanged(m.cfg)
}
func (m *rembgTaskManager) run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			m.cancelRunning()
			return
		}
		task, err := m.claim()
		if err != nil {
			log.Printf("[rembg] claim task failed: %v", err)
			select {
			case <-ctx.Done():
				m.cancelRunning()
				return
			case <-time.After(time.Second):
			}
			continue
		}
		if task == nil {
			select {
			case <-ctx.Done():
				m.cancelRunning()
				return
			case <-m.wake:
			}
			continue
		}
		m.execute(ctx, *task)
	}
}
func (m *rembgTaskManager) claim() (*rembgTask, error) {
	tx, err := m.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	row := tx.QueryRow(`SELECT id,agent_id,source_path,output_path,saved_as,model,alpha_matting,status,progress,started_at,created_at,updated_at,logs,cancel_requested FROM rembg_task WHERE status=? AND cancel_requested=0 ORDER BY id LIMIT 1`, rembgTaskQueued)
	task, err := scanRembgTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, tx.Commit()
	}
	if err != nil {
		return nil, err
	}
	now := rembgTimestamp()
	result, err := tx.Exec(`UPDATE rembg_task SET status=?,started_at=?,updated_at=?,progress=0 WHERE id=? AND status=? AND cancel_requested=0`, rembgTaskRunning, now, now, task.ID, rembgTaskQueued)
	if err != nil {
		return nil, err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return nil, tx.Commit()
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	task.Status, task.StartedAt, task.UpdatedAt, task.Progress = rembgTaskRunning, now, now, 0
	return &task, nil
}
func (m *rembgTaskManager) execute(parent context.Context, task rembgTask) {
	ctx, cancel := context.WithCancel(parent)
	m.mu.Lock()
	m.runningID, m.cancel = task.ID, cancel
	m.mu.Unlock()
	defer func() {
		cancel()
		m.mu.Lock()
		if m.runningID == task.ID {
			m.runningID, m.cancel = 0, nil
		}
		m.mu.Unlock()
	}()
	model, valid := normalizeRembgModel(task.Model)
	if !valid {
		m.finish(task.ID, rembgTaskFailed, 0, "提取失败：任务模型无效。")
		return
	}
	task.Model = model
	alphaNote := ""
	if task.AlphaMatting {
		alphaNote = "已启用 alpha matting，可改善复杂边缘的透明度。"
	}
	m.appendLog(task.ID, "开始使用 rembg（模型："+task.Model+"）提取图片主体；若模型尚未缓存，rembg 将在当前任务中自动下载。"+alphaNote)
	err := m.extract(ctx, task)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) || m.cancelled(task.ID) {
			m.finish(task.ID, rembgTaskCancelled, 0, "任务已取消。")
		} else {
			m.finish(task.ID, rembgTaskFailed, 0, "提取失败："+err.Error())
		}
		return
	}
	m.finish(task.ID, rembgTaskCompleted, 100, "图片主体提取完成。")
}
func isRembgImage(path string) bool {
	declared, allowed := mediaPreviewMIMEType(path)
	if !allowed || !strings.HasPrefix(declared, "image/") {
		return false
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	buffer := make([]byte, 512)
	n, _ := file.Read(buffer)
	if n == 0 {
		return false
	}
	actual := http.DetectContentType(buffer[:n])
	if strings.HasPrefix(actual, "image/") {
		return true
	}
	// Go does not consistently label TIFF/BMP. Validate their signatures rather
	// than accepting a file merely because it has an image extension.
	extension := strings.ToLower(filepath.Ext(path))
	if extension == ".bmp" {
		return n >= 2 && buffer[0] == 'B' && buffer[1] == 'M'
	}
	return (extension == ".tif" || extension == ".tiff") && n >= 4 && ((buffer[0] == 'I' && buffer[1] == 'I' && buffer[2] == 42 && buffer[3] == 0) || (buffer[0] == 'M' && buffer[1] == 'M' && buffer[2] == 0 && buffer[3] == 42))
}
func (m *rembgTaskManager) extract(ctx context.Context, task rembgTask) error {
	path, ok := rembgExecutable(time.Hour)
	if !ok {
		return errors.New("未检测到 rembg")
	}
	source, _, status, msg := resolveMediaPreviewFile(m.cfg, task.AgentID, task.SourcePath)
	if status != 0 {
		return errors.New(msg)
	}
	if !isRembgImage(source) {
		return errors.New("仅支持图片文件")
	}
	workspace, err := getWorkspaceByAgentID(m.cfg, task.AgentID)
	if err != nil {
		return errors.New("Agent 工作目录不存在")
	}
	target, err := resolveCaseInsensitiveUnderRoot(workspace, filepath.FromSlash(task.OutputPath))
	if err != nil || !ensureWritablePathWithinRoot(workspace, target) {
		return errors.New("无法创建图片输出路径")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return errors.New("无法创建图片输出目录")
	}
	completed := false
	defer func() {
		if !completed {
			_ = os.Remove(target)
		}
	}()
	model, valid := normalizeRembgModel(task.Model)
	if !valid {
		return errors.New("任务模型无效")
	}
	cached, err := rembgModelCached(model)
	if err != nil {
		return err
	}
	if !cached {
		m.appendLog(task.ID, "未发现已校验的 rembg 模型缓存，正在从国内镜像下载。")
		if mirrorErr := m.downloadRembgModelFromMirror(ctx, model, task.ID); mirrorErr != nil {
			return fmt.Errorf("国内镜像下载 rembg 模型失败：%w", mirrorErr)
		}
		m.appendLog(task.ID, "rembg 国内镜像模型下载并校验成功，开始执行主体提取。")
	}
	args := rembgInvocationArgs(model, task.AlphaMatting, true, source, target)
	diagnostics, runErr := m.runRembg(ctx, task.ID, path, args)
	if runErr != nil && rembgModelDownloadFailure(diagnostics) {
		m.appendLog(task.ID, "官方下载模型失败，正在通过国内镜像重试下载。")
		if mirrorErr := m.downloadRembgModelFromMirror(ctx, model, task.ID); mirrorErr != nil {
			return fmt.Errorf("%w；国内镜像下载也失败：%v", runErr, mirrorErr)
		}
		m.appendLog(task.ID, "国内镜像下载并校验模型成功，正在重新执行 rembg。")
		_ = os.Remove(target)
		diagnostics, runErr = m.runRembg(ctx, task.ID, path, args)
	}
	if runErr != nil && rembgCLIArgumentError(diagnostics) {
		// rembg has kept its core `i input output` command stable, but older
		// versions may not expose newer quality/model flags. Retry only parser
		// failures, never an inference failure, so the fallback cannot conceal a
		// corrupt input or a broken runtime.
		if task.AlphaMatting {
			m.appendLog(task.ID, "当前 rembg 不支持边缘优化参数，已回退到基础去背景模式。")
			_ = os.Remove(target)
			diagnostics, runErr = m.runRembg(ctx, task.ID, path, rembgInvocationArgs(model, false, true, source, target))
		}
		if runErr != nil && rembgCLIArgumentError(diagnostics) && model != rembgDefaultModel {
			m.appendLog(task.ID, "当前 rembg 不支持所选模型参数，已回退到默认 u2net 模型。")
			_ = os.Remove(target)
			diagnostics, runErr = m.runRembg(ctx, task.ID, path, rembgInvocationArgs(rembgDefaultModel, false, false, source, target))
		}
	}
	if runErr != nil {
		return runErr
	}
	info, err := os.Stat(target)
	if err != nil || info.IsDir() || info.Size() == 0 {
		return errors.New("rembg 未生成有效的 PNG 图片")
	}
	m.progress(task.ID, 100)
	completed = true
	return nil
}

func rembgInvocationArgs(model string, alphaMatting, includeModel bool, source, target string) []string {
	args := []string{"i"}
	if includeModel {
		args = append(args, "-m", model)
	}
	if alphaMatting {
		args = append(args, "--alpha-matting")
	}
	return append(args, source, target)
}

func rembgCLIArgumentError(diagnostics string) bool {
	value := strings.ToLower(diagnostics)
	return strings.Contains(value, "no such option") ||
		strings.Contains(value, "unrecognized option") ||
		strings.Contains(value, "unrecognized arguments") ||
		strings.Contains(value, "unknown option") ||
		strings.Contains(value, "no such command") ||
		strings.Contains(value, "usage: rembg")
}

// rembgModelDownloadFailure is intentionally narrower than a general rembg
// failure. We only use a third-party mirror after rembg/pooch reports that its
// GitHub model download failed; inference, image, and argument errors keep
// their original failure behavior.
func rembgModelDownloadFailure(diagnostics string) bool {
	value := strings.ToLower(diagnostics)
	if strings.Contains(value, "downloading data from") && strings.Contains(value, "github") {
		return true
	}
	if !strings.Contains(value, "pooch") {
		return false
	}
	return strings.Contains(value, "github.com") ||
		strings.Contains(value, "httpsconnectionpool") ||
		strings.Contains(value, "read timed out") ||
		strings.Contains(value, "connection timed out") ||
		strings.Contains(value, "hash of downloaded file")
}

func rembgModelCacheDir() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("U2NET_HOME")); configured != "" {
		if configured == "~" || strings.HasPrefix(configured, "~/") {
			home, err := rembgUserHomeDir()
			if err != nil {
				return "", err
			}
			if configured == "~" {
				return home, nil
			}
			return filepath.Join(home, configured[2:]), nil
		}
		return configured, nil
	}
	dataHome := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
	if dataHome == "" || dataHome == "~" {
		home, err := rembgUserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".u2net"), nil
	}
	return filepath.Join(dataHome, ".u2net"), nil
}

func rembgModelCached(model string) (bool, error) {
	artifact, ok := rembgModelArtifacts[model]
	if !ok {
		return false, errors.New("所选 rembg 模型不支持国内镜像下载")
	}
	directory, err := rembgModelCacheDir()
	if err != nil {
		return false, fmt.Errorf("无法确定 rembg 模型缓存目录: %w", err)
	}
	file, err := os.Open(filepath.Join(directory, artifact.Filename))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("无法读取 rembg 模型缓存: %w", err)
	}
	defer file.Close()
	hash := md5.New() // #nosec G401 -- rembg upstream publishes immutable artifact MD5 values.
	if _, err := io.Copy(hash, file); err != nil {
		return false, fmt.Errorf("无法校验 rembg 模型缓存: %w", err)
	}
	return fmt.Sprintf("%x", hash.Sum(nil)) == artifact.MD5, nil
}

func rembgModelMirrorURL(model string) (string, bool) {
	artifact, ok := rembgModelArtifacts[model]
	if !ok {
		return "", false
	}
	return rembgModelMirrorBaseURL + rembgReleaseAssetBaseURL + artifact.Filename, true
}

func (m *rembgTaskManager) downloadRembgModelFromMirror(ctx context.Context, model string, taskID ...int64) error {
	artifact, ok := rembgModelArtifacts[model]
	if !ok {
		return errors.New("所选 rembg 模型不支持镜像下载")
	}
	url, _ := rembgModelMirrorURL(model)
	cacheDir, err := rembgModelCacheDir()
	if err != nil {
		return fmt.Errorf("无法确定模型缓存目录: %w", err)
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return fmt.Errorf("无法创建模型缓存目录: %w", err)
	}
	downloadCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	idleTimer := time.AfterFunc(rembgModelIdleTimeout, cancel)
	defer idleTimer.Stop()
	req, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("无法创建镜像下载请求: %w", err)
	}
	response, err := rembgHTTPClient.Do(req)
	if err != nil {
		if ctx.Err() == nil && downloadCtx.Err() != nil {
			return fmt.Errorf("国内镜像下载超过 %s 没有进度", rembgModelIdleTimeout)
		}
		return fmt.Errorf("请求镜像失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("镜像返回 HTTP %d", response.StatusCode)
	}

	temporary, err := os.CreateTemp(cacheDir, "."+artifact.Filename+".mirror-*")
	if err != nil {
		return fmt.Errorf("无法创建模型临时文件: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	hash := md5.New() // #nosec G401 -- rembg upstream publishes an MD5 for each immutable model artifact.
	buffer := make([]byte, 1024*1024)
	var copied int64
	nextProgress := int64(10)
	for {
		count, readErr := response.Body.Read(buffer)
		if count > 0 {
			if !idleTimer.Stop() && downloadCtx.Err() != nil && ctx.Err() == nil {
				_ = temporary.Close()
				return fmt.Errorf("国内镜像下载超过 %s 没有进度", rembgModelIdleTimeout)
			}
			idleTimer.Reset(rembgModelIdleTimeout)
			if _, err := hash.Write(buffer[:count]); err != nil {
				_ = temporary.Close()
				return fmt.Errorf("校验镜像模型失败: %w", err)
			}
			if _, err := temporary.Write(buffer[:count]); err != nil {
				_ = temporary.Close()
				return fmt.Errorf("保存镜像模型失败: %w", err)
			}
			copied += int64(count)
			if len(taskID) > 0 && response.ContentLength > 0 && copied*100 >= response.ContentLength*nextProgress {
				m.appendLog(taskID[0], fmt.Sprintf("rembg 国内镜像模型已下载 %d%%。", nextProgress))
				nextProgress += 10
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = temporary.Close()
			if ctx.Err() == nil && downloadCtx.Err() != nil {
				return fmt.Errorf("国内镜像下载超过 %s 没有进度", rembgModelIdleTimeout)
			}
			return fmt.Errorf("读取镜像模型失败: %w", readErr)
		}
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("保存镜像模型失败: %w", err)
	}
	if actual := fmt.Sprintf("%x", hash.Sum(nil)); actual != artifact.MD5 {
		return fmt.Errorf("镜像模型校验失败（MD5 %s）", actual)
	}
	if err := os.Rename(temporaryPath, filepath.Join(cacheDir, artifact.Filename)); err != nil {
		return fmt.Errorf("无法写入模型缓存: %w", err)
	}
	return nil
}

func (m *rembgTaskManager) runRembg(ctx context.Context, taskID int64, executable string, args []string) (string, error) {
	cmd := rembgCommandContext(ctx, executable, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("无法启动 rembg: %w", err)
	}
	var diagnostics strings.Builder
	var diagnosticsMu sync.Mutex
	var out sync.WaitGroup
	out.Add(2)
	go func() { defer out.Done(); m.capture(taskID, stdout, &diagnostics, &diagnosticsMu) }()
	go func() { defer out.Done(); m.capture(taskID, stderr, &diagnostics, &diagnosticsMu) }()
	out.Wait()
	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return diagnostics.String(), context.Canceled
		}
		return diagnostics.String(), fmt.Errorf("rembg 执行失败: %w", err)
	}
	return diagnostics.String(), nil
}
func (m *rembgTaskManager) capture(id int64, r io.Reader, diagnostics *strings.Builder, diagnosticsMu *sync.Mutex) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 128*1024)
	// rembg's model downloader uses a terminal-style progress bar: it rewrites
	// one line with carriage returns instead of emitting newline-delimited
	// output. Treat both delimiters as a log boundary so live task logs show
	// genuine download progress rather than appearing frozen.
	scanner.Split(splitRembgOutput)
	previous := ""
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			diagnosticsMu.Lock()
			if diagnostics.Len() < 128*1024 {
				diagnostics.WriteString(line + "\n")
			}
			diagnosticsMu.Unlock()
			if line != previous {
				m.appendLog(id, line)
				previous = line
			}
		}
	}
	if err := scanner.Err(); err != nil {
		m.appendLog(id, "读取 rembg 输出失败："+err.Error())
	}
}

func splitRembgOutput(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for index, value := range data {
		if value == '\n' || value == '\r' {
			return index + 1, data[:index], nil
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}
func (m *rembgTaskManager) progress(id int64, progress int) {
	_, _ = m.db.Exec(`UPDATE rembg_task SET progress=CASE WHEN progress>? THEN progress ELSE ? END,updated_at=? WHERE id=? AND status=?`, progress, progress, rembgTimestamp(), id, rembgTaskRunning)
}
func (m *rembgTaskManager) appendLog(id int64, message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	if len(message) > 4096 {
		message = message[:4096] + "…"
	}
	now := rembgTimestamp()
	_, _ = m.db.Exec(`UPDATE rembg_task SET logs=CASE WHEN length(logs)>? THEN substr(logs,-?) ELSE logs END || ?,updated_at=? WHERE id=?`, rembgTaskLogLimit-8192, rembgTaskLogLimit/2, "["+now+"] "+message+"\n", now, id)
}
func (m *rembgTaskManager) finish(id int64, status string, progress int, message string) {
	m.appendLog(id, message)
	_, _ = m.db.Exec(`UPDATE rembg_task SET status=?,progress=?,updated_at=? WHERE id=?`, status, progress, rembgTimestamp(), id)
	notifyIntegrationSleepConditionChanged(m.cfg)
}
func (m *rembgTaskManager) cancelled(id int64) bool {
	var cancelled int
	return m.db.QueryRow(`SELECT cancel_requested FROM rembg_task WHERE id=?`, id).Scan(&cancelled) != nil || cancelled != 0
}
func (m *rembgTaskManager) cancelRunning() {
	m.mu.Lock()
	cancel := m.cancel
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (m *rembgTaskManager) allocate(agentID, workspace, source string, reserved map[string]bool, except int64) (string, string, error) {
	stem := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
	stamp := rembgNow().Format("20060102_150405")
	for attempt := 0; attempt < 1000; attempt++ {
		name := stem + "_subject.png"
		if attempt > 0 {
			name = stem + "_subject_" + stamp
			if attempt > 1 {
				name += "_" + strconv.Itoa(attempt)
			}
			name += ".png"
		}
		relative := filepath.Join("images", name)
		key := filepath.ToSlash(relative)
		if reserved[key] {
			continue
		}
		absolute, err := resolveCaseInsensitiveUnderRoot(workspace, relative)
		if err != nil || !ensureWritablePathWithinRoot(workspace, absolute) {
			return "", "", errors.New("无法创建图片输出路径")
		}
		var existing int
		if err := m.db.QueryRow(`SELECT COUNT(*) FROM rembg_task WHERE agent_id=? AND output_path=? AND id<>?`, agentID, key, except).Scan(&existing); err != nil {
			return "", "", err
		}
		if existing != 0 {
			continue
		}
		if _, err := os.Lstat(absolute); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", "", err
		}
		reserved[key] = true
		return relative, absolute, nil
	}
	return "", "", errors.New("无法分配图片输出文件名")
}
func (m *rembgTaskManager) create(request rembgTaskCreateRequest) ([]rembgTask, error) {
	if m == nil || m.cfg == nil || m.db == nil {
		return nil, errors.New("图片主体任务服务未就绪")
	}
	request.AgentID = strings.TrimSpace(request.AgentID)
	items := request.Tasks
	if len(items) == 0 {
		items = make([]rembgTaskCreateItem, 0, len(request.Paths))
		for _, path := range request.Paths {
			items = append(items, rembgTaskCreateItem{Path: path, Model: request.Model, AlphaMatting: request.AlphaMatting})
		}
	}
	if !isValidMediaPreviewAgentID(request.AgentID) || len(items) == 0 || len(items) > 64 {
		return nil, errors.New("agentId 和 1 至 64 个图片文件不能为空")
	}
	m.createMu.Lock()
	defer m.createMu.Unlock()
	workspace, err := getWorkspaceByAgentID(m.cfg, request.AgentID)
	if err != nil {
		return nil, errors.New("Agent 不存在")
	}
	type source struct {
		relative, output, absolute, model string
		alpha                             bool
	}
	sources := []source{}
	seen, reserved := map[string]bool{}, map[string]bool{}
	for _, item := range items {
		relative := normalizeQuotedPathArg(item.Path)
		if seen[relative] {
			continue
		}
		seen[relative] = true
		model, valid := normalizeRembgModel(item.Model)
		if !valid {
			return nil, errors.New("不支持的 rembg 模型")
		}
		absolute, _, status, msg := resolveMediaPreviewFile(m.cfg, request.AgentID, relative)
		if status != 0 {
			return nil, errors.New(msg)
		}
		if !isRembgImage(absolute) {
			return nil, errors.New("仅支持图片文件")
		}
		rel, err := filepath.Rel(workspace, absolute)
		if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, errors.New("图片文件必须位于当前 Agent 工作目录")
		}
		output, saved, err := m.allocate(request.AgentID, workspace, rel, reserved, 0)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source{relative: filepath.ToSlash(rel), output: filepath.ToSlash(output), absolute: saved, model: model, alpha: item.AlphaMatting})
	}
	if len(sources) == 0 {
		return nil, errors.New("未找到可添加的图片文件")
	}
	tx, err := m.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	now := rembgTimestamp()
	created := make([]rembgTask, 0, len(sources))
	for _, source := range sources {
		logMessage := "[" + now + "] 已加入图片主体提取队列（模型：" + source.model + "）。\n"
		if source.alpha {
			logMessage = "[" + now + "] 已加入图片主体提取队列（模型：" + source.model + "，已启用 alpha matting）。\n"
		}
		result, err := tx.Exec(`INSERT INTO rembg_task(agent_id,source_path,output_path,saved_as,model,alpha_matting,status,progress,created_at,updated_at,logs,cancel_requested) VALUES(?,?,?,?,?,?,?,?,?,?,?,0)`, request.AgentID, source.relative, source.output, source.absolute, source.model, boolToSQLite(source.alpha), rembgTaskQueued, 0, now, now, logMessage)
		if err != nil {
			return nil, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}
		created = append(created, rembgTask{ID: id, AgentID: request.AgentID, SourcePath: source.relative, OutputPath: source.output, SavedAs: source.absolute, Model: source.model, AlphaMatting: source.alpha, Status: rembgTaskQueued, CreatedAt: now, UpdatedAt: now})
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	m.signal()
	return created, nil
}

func boolToSQLite(value bool) int {
	if value {
		return 1
	}
	return 0
}

func validRembgStatus(status string) bool {
	switch status {
	case "", rembgTaskQueued, rembgTaskRunning, rembgTaskCompleted, rembgTaskCancelled, rembgTaskFailed:
		return true
	}
	return false
}
func (m *rembgTaskManager) list(agentID, status string, page int, logs bool) (rembgTaskPage, error) {
	result := rembgTaskPage{Tasks: []rembgTask{}, PageSize: 5}
	agentID = strings.TrimSpace(agentID)
	status = strings.TrimSpace(status)
	if !isValidMediaPreviewAgentID(agentID) || !validRembgStatus(status) {
		return result, errors.New("任务参数无效")
	}
	if page < 1 {
		page = 1
	}
	where, args := "agent_id=?", []interface{}{agentID}
	if status != "" {
		where += " AND status=?"
		args = append(args, status)
	}
	if err := m.db.QueryRow(`SELECT COUNT(*) FROM rembg_task WHERE `+where, args...).Scan(&result.Total); err != nil {
		return result, err
	}
	lastPage := (result.Total + result.PageSize - 1) / result.PageSize
	if lastPage < 1 {
		lastPage = 1
	}
	if page > lastPage {
		page = lastPage
	}
	result.Page, result.Status = page, status
	column := "''"
	if logs {
		column = "logs"
	}
	args = append(args, result.PageSize, (page-1)*result.PageSize)
	rows, err := m.db.Query(`SELECT id,agent_id,source_path,output_path,saved_as,model,alpha_matting,status,progress,started_at,created_at,updated_at,`+column+`,cancel_requested FROM rembg_task WHERE `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		task, err := scanRembgTask(rows)
		if err != nil {
			return result, err
		}
		result.Tasks = append(result.Tasks, task)
	}
	return result, rows.Err()
}
func (m *rembgTaskManager) log(agentID string, id int64) (rembgTask, error) {
	if id <= 0 || !isValidMediaPreviewAgentID(strings.TrimSpace(agentID)) {
		return rembgTask{}, errors.New("任务参数无效")
	}
	return scanRembgTask(m.db.QueryRow(`SELECT id,agent_id,source_path,output_path,saved_as,model,alpha_matting,status,progress,started_at,created_at,updated_at,logs,cancel_requested FROM rembg_task WHERE id=? AND agent_id=?`, id, strings.TrimSpace(agentID)))
}
func (m *rembgTaskManager) cancelTask(agentID string, id int64) error {
	now := rembgTimestamp()
	result, err := m.db.Exec(`UPDATE rembg_task SET cancel_requested=1,status=CASE WHEN status=? THEN ? ELSE status END,updated_at=?,logs=CASE WHEN length(logs)>? THEN substr(logs,-?) ELSE logs END || ? WHERE id=? AND agent_id=? AND status IN (?,?)`, rembgTaskQueued, rembgTaskCancelled, now, rembgTaskLogLimit-4096, rembgTaskLogLimit/2, "["+now+"] 已请求取消任务。\n", id, strings.TrimSpace(agentID), rembgTaskQueued, rembgTaskRunning)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return errors.New("仅可取消排队中或执行中的任务")
	}
	m.mu.Lock()
	shouldCancel := m.runningID == id
	cancel := m.cancel
	m.mu.Unlock()
	if shouldCancel && cancel != nil {
		cancel()
	}
	notifyIntegrationSleepConditionChanged(m.cfg)
	return nil
}
func (m *rembgTaskManager) restart(agentID string, id int64) error {
	agentID = strings.TrimSpace(agentID)
	var source string
	if err := m.db.QueryRow(`SELECT source_path FROM rembg_task WHERE id=? AND agent_id=? AND status=?`, id, agentID, rembgTaskCancelled).Scan(&source); err != nil {
		return errors.New("仅可重新开始已取消任务")
	}
	workspace, err := getWorkspaceByAgentID(m.cfg, agentID)
	if err != nil {
		return errors.New("Agent 不存在")
	}
	output, saved, err := m.allocate(agentID, workspace, source, map[string]bool{}, id)
	if err != nil {
		return err
	}
	now := rembgTimestamp()
	result, err := m.db.Exec(`UPDATE rembg_task SET output_path=?,saved_as=?,status=?,progress=0,started_at='',cancel_requested=0,updated_at=?,logs=CASE WHEN length(logs)>? THEN substr(logs,-?) ELSE logs END || ? WHERE id=? AND agent_id=? AND status=?`, filepath.ToSlash(output), saved, rembgTaskQueued, now, rembgTaskLogLimit-4096, rembgTaskLogLimit/2, "["+now+"] 已重新加入图片主体提取队列。\n", id, agentID, rembgTaskCancelled)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return errors.New("重新开始任务失败")
	}
	m.signal()
	return nil
}
func (m *rembgTaskManager) deleteFailedOrCancelled(agentID string, id int64) error {
	result, err := m.db.Exec(`DELETE FROM rembg_task WHERE id=? AND agent_id=? AND status IN (?, ?)`, id, strings.TrimSpace(agentID), rembgTaskFailed, rembgTaskCancelled)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return errors.New("仅可删除失败或已取消任务")
	}
	return nil
}

type rembgScanner interface{ Scan(...interface{}) error }

func scanRembgTask(scanner rembgScanner) (rembgTask, error) {
	var task rembgTask
	var cancelled int
	var alphaMatting int
	err := scanner.Scan(&task.ID, &task.AgentID, &task.SourcePath, &task.OutputPath, &task.SavedAs, &task.Model, &alphaMatting, &task.Status, &task.Progress, &task.StartedAt, &task.CreatedAt, &task.UpdatedAt, &task.Logs, &cancelled)
	if strings.TrimSpace(task.Model) == "" {
		task.Model = rembgDefaultModel
	}
	task.AlphaMatting = alphaMatting != 0
	task.CancelAsked = cancelled != 0
	return task, err
}

func writeRembgJSON(w http.ResponseWriter, code int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
func readRembgAction(w http.ResponseWriter, r *http.Request) (rembgTaskActionRequest, error) {
	var request rembgTaskActionRequest
	if r.Method != http.MethodPost {
		return request, errors.New("仅支持 POST 请求")
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return request, errors.New("请求内容无效")
	}
	return request, nil
}
func handleRembgTasks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if rembgTasks == nil {
			writeRembgJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": 1, "content": "图片主体任务服务未就绪"})
			return
		}
		switch r.Method {
		case http.MethodGet:
			page := 1
			if raw := strings.TrimSpace(r.URL.Query().Get("page")); raw != "" {
				parsed, err := strconv.Atoi(raw)
				if err != nil || parsed < 1 {
					writeRembgJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "页码必须是正整数"})
					return
				}
				page = parsed
			}
			value, err := rembgTasks.list(r.URL.Query().Get("agentId"), r.URL.Query().Get("status"), page, false)
			if err != nil {
				writeRembgJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
				return
			}
			writeRembgJSON(w, http.StatusOK, map[string]interface{}{"status": 0, "tasks": value.Tasks, "total": value.Total, "page": value.Page, "pageSize": value.PageSize, "taskStatus": value.Status})
		case http.MethodPost:
			var request rembgTaskCreateRequest
			decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
				writeRembgJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "请求内容无效"})
				return
			}
			value, err := rembgTasks.create(request)
			if err != nil {
				writeRembgJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
				return
			}
			writeRembgJSON(w, http.StatusOK, map[string]interface{}{"status": 0, "tasks": value})
		default:
			w.Header().Set("Allow", "GET, POST")
			writeRembgJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"status": 1, "content": "仅支持 GET 或 POST 请求"})
		}
	}
}
func handleRembgTaskCancel() http.HandlerFunc {
	return rembgActionHandler(func(m *rembgTaskManager, r rembgTaskActionRequest) error { return m.cancelTask(r.AgentID, r.ID) })
}
func handleRembgTaskRestart() http.HandlerFunc {
	return rembgActionHandler(func(m *rembgTaskManager, r rembgTaskActionRequest) error { return m.restart(r.AgentID, r.ID) })
}
func handleRembgTaskDelete() http.HandlerFunc {
	return rembgActionHandler(func(m *rembgTaskManager, r rembgTaskActionRequest) error {
		return m.deleteFailedOrCancelled(r.AgentID, r.ID)
	})
}
func rembgActionHandler(action func(*rembgTaskManager, rembgTaskActionRequest) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeRembgJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"status": 1, "content": "仅支持 POST 请求"})
			return
		}
		if rembgTasks == nil {
			writeRembgJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": 1, "content": "图片主体任务服务未就绪"})
			return
		}
		request, err := readRembgAction(w, r)
		if err == nil {
			err = action(rembgTasks, request)
		}
		if err != nil {
			writeRembgJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		writeRembgJSON(w, http.StatusOK, map[string]interface{}{"status": 0})
	}
}
func handleRembgTaskLog() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeRembgJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"status": 1, "content": "仅支持 GET 请求"})
			return
		}
		if rembgTasks == nil {
			writeRembgJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": 1, "content": "图片主体任务服务未就绪"})
			return
		}
		id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
		if err == nil {
			var task rembgTask
			task, err = rembgTasks.log(r.URL.Query().Get("agentId"), id)
			if err == nil {
				writeRembgJSON(w, http.StatusOK, map[string]interface{}{"status": 0, "task": task})
				return
			}
		}
		writeRembgJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "任务参数无效"})
	}
}
