package main

// VoxCPM voice-cloning tasks deliberately follow the same durable queue
// semantics as Whisper and rembg.  The request only supplies workspace
// relative source paths; this package owns all command and output arguments.

import (
	"bufio"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	voxcpmTaskQueued    = "queued"
	voxcpmTaskRunning   = "running"
	voxcpmTaskCompleted = "completed"
	voxcpmTaskCancelled = "cancelled"
	voxcpmTaskFailed    = "failed"
	voxcpmTaskLogLimit  = 128 * 1024
	voxcpmTextLimit     = 1024 * 1024
	voxcpmControlLimit  = 240
	voxcpmProbeTimeout  = time.Minute
	// VoxCPM2 is published by OpenBMB on ModelScope as an official domestic
	// mirror. The URLs stay overridable in tests, while the production values
	// intentionally point at the publisher's namespace rather than a proxy.
	voxcpmDefaultModelScopeFilesURL     = "https://modelscope.cn/api/v1/models/OpenBMB/VoxCPM2/repo/files?Revision=master&Root="
	voxcpmDefaultModelScopeResolveBase  = "https://modelscope.cn/models/OpenBMB/VoxCPM2/resolve/master/"
	voxcpmDefaultHuggingFaceResolveBase = "https://huggingface.co/openbmb/VoxCPM2/resolve/main/"
	voxcpmModelHeaderTimeout            = 30 * time.Second
	voxcpmModelIdleTimeout              = 90 * time.Second

	voxcpmScenarioBalanced      = "balanced"
	voxcpmScenarioQuality       = "quality"
	voxcpmScenarioFast          = "fast"
	voxcpmScenarioClean         = "clean"
	voxcpmScenarioWarmNarration = "warm_narration"
	voxcpmScenarioLively        = "lively"
)

type voxcpmScenario struct {
	ID, Name, Description, Control string
	CFGValue                       float64
	InferenceTimesteps             int
	Normalize, Denoise             bool
}

func voxcpmScenarioFor(raw string) (voxcpmScenario, bool) {
	switch strings.TrimSpace(raw) {
	case "", voxcpmScenarioBalanced:
		return voxcpmScenario{ID: voxcpmScenarioBalanced, Name: "均衡克隆", Description: "速度、自然度与资源占用均衡。", CFGValue: 2, InferenceTimesteps: 10}, true
	case voxcpmScenarioQuality:
		return voxcpmScenario{ID: voxcpmScenarioQuality, Name: "质量优先", Description: "更高推理步数与参考音频增强，适合正式成品。", CFGValue: 2.5, InferenceTimesteps: 20, Normalize: true, Denoise: true}, true
	case voxcpmScenarioFast:
		return voxcpmScenario{ID: voxcpmScenarioFast, Name: "快速预览", Description: "低推理步数，优先快速试听。", CFGValue: 1.5, InferenceTimesteps: 4, Normalize: true}, true
	case voxcpmScenarioClean:
		return voxcpmScenario{ID: voxcpmScenarioClean, Name: "清晰增强", Description: "规范化文字并增强参考音频，适合有底噪的素材。", CFGValue: 2, InferenceTimesteps: 12, Normalize: true, Denoise: true}, true
	case voxcpmScenarioWarmNarration:
		return voxcpmScenario{ID: voxcpmScenarioWarmNarration, Name: "温暖叙述", Description: "温和、清晰、平稳的叙述风格。", Control: "温和、清晰、平稳、自然的叙述语气", CFGValue: 2.5, InferenceTimesteps: 15, Normalize: true, Denoise: true}, true
	case voxcpmScenarioLively:
		return voxcpmScenario{ID: voxcpmScenarioLively, Name: "轻快表达", Description: "语速稍快、自然有活力的表达风格。", Control: "语速稍快、轻松愉悦、自然有活力", CFGValue: 2.5, InferenceTimesteps: 15, Normalize: true}, true
	default:
		return voxcpmScenario{}, false
	}
}

func voxcpmScenarioLogParameters(scenario voxcpmScenario, control string, cloning bool) string {
	parts := []string{
		"场景“" + scenario.Name + "”参数",
		"cfg_value=" + strconv.FormatFloat(scenario.CFGValue, 'f', -1, 64),
		"inference_timesteps=" + strconv.Itoa(scenario.InferenceTimesteps),
		"normalize=" + strconv.FormatBool(scenario.Normalize),
		"denoise=" + strconv.FormatBool(scenario.Denoise && cloning),
	}
	if control != "" {
		parts = append(parts, "--control="+control)
	} else {
		parts = append(parts, "--control=未设置")
	}
	return strings.Join(parts, "，") + "。"
}

func voxcpmControlText(raw string) (string, error) {
	control := strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	if !utf8.ValidString(control) || utf8.RuneCountInString(control) > voxcpmControlLimit {
		return "", fmt.Errorf("表达风格最多支持 %d 个字符", voxcpmControlLimit)
	}
	return control, nil
}

func voxcpmEffectiveControl(scenario voxcpmScenario, custom string, cloning bool) string {
	if cloning {
		return ""
	}
	if strings.TrimSpace(custom) != "" {
		return custom
	}
	return scenario.Control
}

type voxcpmCheckConfig struct {
	CacheFor time.Duration
	Install  string
}
type voxcpmCheckResponse struct {
	Available bool   `json:"available"`
	Install   string `json:"install,omitempty"`
	Content   string `json:"content,omitempty"`
	Status    int    `json:"status"`
}
type voxcpmTask struct {
	ID                    int64  `json:"id"`
	AgentID               string `json:"agentId"`
	TextPath              string `json:"textPath"`
	TextSavedAs           string `json:"textSavedAs,omitempty"`
	ReferenceAudioPath    string `json:"referenceAudioPath"`
	ReferenceAudioSavedAs string `json:"referenceAudioSavedAs,omitempty"`
	Scenario              string `json:"scenario"`
	Control               string `json:"control,omitempty"`
	RequestedName         string `json:"requestedName"`
	OutputPath            string `json:"outputPath,omitempty"`
	SavedAs               string `json:"savedAs,omitempty"`
	Status                string `json:"status"`
	Progress              int    `json:"progress"`
	StartedAt             string `json:"startedAt,omitempty"`
	CreatedAt             string `json:"createdAt"`
	UpdatedAt             string `json:"updatedAt"`
	Logs                  string `json:"logs,omitempty"`
	CancelAsked           bool   `json:"cancelRequested"`
}
type voxcpmTaskCreateItem struct {
	TextPath           string `json:"textPath"`
	ReferenceAudioPath string `json:"referenceAudioPath"`
	Scenario           string `json:"scenario,omitempty"`
	Control            string `json:"control,omitempty"`
}
type voxcpmTaskCreateRequest struct {
	AgentID    string                 `json:"agentId"`
	OutputName string                 `json:"outputName"`
	Tasks      []voxcpmTaskCreateItem `json:"tasks"`
}
type voxcpmTaskActionRequest struct {
	AgentID string `json:"agentId"`
	ID      int64  `json:"id"`
}
type voxcpmTaskPage struct {
	Tasks                 []voxcpmTask
	Total, Page, PageSize int
	Status                string
}
type voxcpmTaskManager struct {
	cfg       *Config
	db        *sql.DB
	createMu  sync.Mutex
	mu        sync.Mutex
	runningID int64
	cancel    context.CancelFunc
	wake      chan struct{}
}

var (
	voxcpmTasks                  *voxcpmTaskManager
	voxcpmLookPath               = exec.LookPath
	voxcpmCommandContext         = exec.CommandContext
	voxcpmTaskLookPath           = exec.LookPath
	voxcpmTaskPreferredDevice    = voxcpmPreferredDevice
	voxcpmNow                    = time.Now
	voxcpmHTTPClient             = &http.Client{Transport: &http.Transport{ResponseHeaderTimeout: voxcpmModelHeaderTimeout}}
	voxcpmUserCacheDir           = os.UserCacheDir
	voxcpmModelMirrorMu          sync.Mutex
	voxcpmModelScopeFilesURL     = voxcpmDefaultModelScopeFilesURL
	voxcpmModelScopeResolveBase  = voxcpmDefaultModelScopeResolveBase
	voxcpmHuggingFaceResolveBase = voxcpmDefaultHuggingFaceResolveBase
	// Kept injectable so task-flow tests can use a tiny local snapshot instead
	// of downloading the real multi-gigabyte model.
	voxcpmPrepareLocalModel = func(manager *voxcpmTaskManager, ctx context.Context, taskID int64) (string, error) {
		return manager.downloadVoxCPM2FromModelScope(ctx, taskID)
	}
	voxcpmCheckCache struct {
		sync.Mutex
		checkedAt  time.Time
		executable string
	}
)

func voxcpmTimestamp() string { return voxcpmNow().Format(time.RFC3339Nano) }

func ensureVoxCPMTaskSchema(db *sql.DB) error {
	if db == nil {
		return errors.New("voxcpm task database is not ready")
	}
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS voxcpm_task (
		id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL,
		text_path TEXT NOT NULL, text_saved_as TEXT NOT NULL,
		reference_audio_path TEXT NOT NULL, reference_audio_saved_as TEXT NOT NULL,
		requested_name TEXT NOT NULL, output_path TEXT NOT NULL, saved_as TEXT NOT NULL,
		scenario TEXT NOT NULL DEFAULT 'balanced', control TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'queued', progress INTEGER NOT NULL DEFAULT 0,
		started_at TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
		logs TEXT NOT NULL DEFAULT '', cancel_requested INTEGER NOT NULL DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_voxcpm_task_agent_created ON voxcpm_task(agent_id,id DESC);
	CREATE INDEX IF NOT EXISTS idx_voxcpm_task_queue ON voxcpm_task(status,cancel_requested,id);`)
	if err != nil {
		return err
	}
	// The first release did not persist generation settings. Add the scenario
	// column without invalidating existing queued tasks; they use the balanced
	// profile that matches the former CLI defaults.
	columns := map[string]bool{}
	rows, err := db.Query(`PRAGMA table_info(voxcpm_task)`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
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
	if !columns["scenario"] {
		if _, err := db.Exec(`ALTER TABLE voxcpm_task ADD COLUMN scenario TEXT NOT NULL DEFAULT 'balanced'`); err != nil {
			return err
		}
	}
	if !columns["control"] {
		if _, err := db.Exec(`ALTER TABLE voxcpm_task ADD COLUMN control TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	return nil
}

func parseVoxCPMCheckConfig(raw map[string]interface{}) (voxcpmCheckConfig, error) {
	if raw == nil {
		return voxcpmCheckConfig{}, errors.New("config/config.json.voxcpm 配置缺失")
	}
	value, ok := raw["voxcpm"]
	if !ok {
		return voxcpmCheckConfig{}, errors.New("config/config.json.voxcpm 配置缺失")
	}
	section, ok := value.(map[string]interface{})
	if !ok || section == nil {
		return voxcpmCheckConfig{}, errors.New("config/config.json.voxcpm 必须是对象")
	}
	hours, ok := ffmpegCheckPositiveHours(section["check"])
	if !ok {
		return voxcpmCheckConfig{}, errors.New("config/config.json.voxcpm.check 必须是正整数（小时）")
	}
	install, ok := section["install"].(string)
	install = strings.TrimSpace(install)
	if !ok || install == "" {
		return voxcpmCheckConfig{}, errors.New("config/config.json.voxcpm.install 必须是非空字符串")
	}
	return voxcpmCheckConfig{CacheFor: time.Duration(hours) * time.Hour, Install: install}, nil
}

func voxcpmExecutable(cacheFor time.Duration) (string, bool) {
	now := voxcpmNow()
	path, err := voxcpmLookPath("voxcpm")
	if err != nil || strings.TrimSpace(path) == "" {
		return "", false
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", false
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return "", false
	}
	voxcpmCheckCache.Lock()
	cachedAt, cached := voxcpmCheckCache.checkedAt, voxcpmCheckCache.executable
	voxcpmCheckCache.Unlock()
	if cached == path && !cachedAt.IsZero() && now.Before(cachedAt.Add(cacheFor)) {
		return path, true
	}
	ctx, cancel := context.WithTimeout(context.Background(), voxcpmProbeTimeout)
	err = voxcpmCommandContext(ctx, path, "--help").Run()
	timedOut := ctx.Err() != nil
	cancel()
	if err != nil || timedOut {
		return "", false
	}
	voxcpmCheckCache.Lock()
	voxcpmCheckCache.checkedAt, voxcpmCheckCache.executable = now, path
	voxcpmCheckCache.Unlock()
	return path, true
}

func writeVoxCPMJSON(w http.ResponseWriter, code int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}

func handleVoxCPMCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeVoxCPMJSON(w, http.StatusMethodNotAllowed, voxcpmCheckResponse{Content: "仅支持 GET 请求", Status: 1})
			return
		}
		status, response := checkVoxCPMDependency()
		writeVoxCPMJSON(w, status, response)
	}
}

// checkVoxCPMDependency contains the complete availability rule shared by
// the HTTP endpoint and the asynchronous startup preflight.
func checkVoxCPMDependency() (int, voxcpmCheckResponse) {
	raw, _, err := readIntegrationStartupConfigRaw()
	if err != nil {
		return http.StatusInternalServerError, voxcpmCheckResponse{Content: "读取 config/config.json 失败: " + err.Error(), Status: 1}
	}
	config, err := parseVoxCPMCheckConfig(raw)
	if err != nil {
		return http.StatusBadRequest, voxcpmCheckResponse{Content: err.Error(), Status: 1}
	}
	if _, ok := voxcpmExecutable(config.CacheFor); !ok {
		return http.StatusOK, voxcpmCheckResponse{Install: config.Install, Content: "未检测到可用的 voxcpm", Status: 0}
	}
	return http.StatusOK, voxcpmCheckResponse{Available: true, Status: 0}
}

func startVoxCPMTaskManager(ctx context.Context, cfg *Config) {
	if cronDB == nil {
		return
	}
	if err := ensureVoxCPMTaskSchema(cronDB); err != nil {
		return
	}
	m := &voxcpmTaskManager{cfg: cfg, db: cronDB, wake: make(chan struct{}, 1)}
	if err := m.recover(); err != nil {
		return
	}
	voxcpmTasks = m
	go m.run(ctx)
}
func (m *voxcpmTaskManager) recover() error {
	now := voxcpmTimestamp()
	_, err := m.db.Exec(`UPDATE voxcpm_task SET status=?,progress=0,started_at='',updated_at=?,logs=CASE WHEN length(logs)>? THEN substr(logs,-?) ELSE logs END || ? WHERE status=? AND cancel_requested=0`, voxcpmTaskQueued, now, voxcpmTaskLogLimit-2048, voxcpmTaskLogLimit/2, "\n[服务重启] 上次执行被中断，任务已恢复为排队中。\n", voxcpmTaskRunning)
	if err != nil {
		return err
	}
	_, err = m.db.Exec(`UPDATE voxcpm_task SET status=?,updated_at=? WHERE status=? AND cancel_requested=1`, voxcpmTaskCancelled, now, voxcpmTaskRunning)
	return err
}
func (m *voxcpmTaskManager) signal() {
	select {
	case m.wake <- struct{}{}:
	default:
	}
	notifyIntegrationSleepConditionChanged(m.cfg)
}
func (m *voxcpmTaskManager) run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			m.cancelRunning()
			return
		}
		slotHeld, err := reserveModelTaskSlot(ctx, m.cfg, m.db, "voxcpm_task")
		if err != nil {
			if errors.Is(err, context.Canceled) {
				m.cancelRunning()
				return
			}
			log.Printf("[voxcpm] reserve shared task slot failed: %v", err)
			select {
			case <-ctx.Done():
				m.cancelRunning()
				return
			case <-time.After(time.Second):
			}
			continue
		}
		if !slotHeld {
			select {
			case <-ctx.Done():
				m.cancelRunning()
				return
			case <-m.wake:
			}
			continue
		}
		task, err := m.claim()
		if err != nil {
			releaseModelTaskSlot(m.cfg)
			select {
			case <-ctx.Done():
				m.cancelRunning()
				return
			case <-time.After(time.Second):
			}
			continue
		}
		if task == nil {
			releaseModelTaskSlot(m.cfg)
			select {
			case <-ctx.Done():
				m.cancelRunning()
				return
			case <-m.wake:
			}
			continue
		}
		m.execute(ctx, *task)
		releaseModelTaskSlot(m.cfg)
	}
}
func (m *voxcpmTaskManager) claim() (*voxcpmTask, error) {
	tx, err := m.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	task, err := scanVoxCPMTask(tx.QueryRow(voxcpmTaskColumns+`logs,cancel_requested FROM voxcpm_task WHERE status=? AND cancel_requested=0 ORDER BY id LIMIT 1`, voxcpmTaskQueued))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, tx.Commit()
	}
	if err != nil {
		return nil, err
	}
	now := voxcpmTimestamp()
	result, err := tx.Exec(`UPDATE voxcpm_task SET status=?,started_at=?,updated_at=?,progress=0 WHERE id=? AND status=? AND cancel_requested=0`, voxcpmTaskRunning, now, now, task.ID, voxcpmTaskQueued)
	if err != nil {
		return nil, err
	}
	changed, _ := result.RowsAffected()
	if changed != 1 {
		return nil, tx.Commit()
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	task.Status, task.StartedAt, task.UpdatedAt, task.Progress = voxcpmTaskRunning, now, now, 0
	return &task, nil
}
func (m *voxcpmTaskManager) execute(parent context.Context, task voxcpmTask) {
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
	if strings.TrimSpace(task.ReferenceAudioPath) == "" {
		m.appendLog(task.ID, "未选择参考音色，开始使用 VoxCPM design 自由生成声音。")
	} else {
		m.appendLog(task.ID, "已选择参考音色，开始使用 VoxCPM 克隆参考音色生成语音。")
	}
	m.appendLog(task.ID, "任务目的：将 UTF-8 文字合成为 WAV 语音。")
	if err := m.generate(ctx, task); err != nil {
		if errors.Is(ctx.Err(), context.Canceled) || m.cancelled(task.ID) {
			m.finish(task.ID, voxcpmTaskCancelled, 0, "任务已取消。")
		} else {
			m.finish(task.ID, voxcpmTaskFailed, 0, "任务失败，具体原因："+err.Error())
		}
		return
	}
	m.finish(task.ID, voxcpmTaskCompleted, 100, "文字转语音完成。")
}

func voxcpmResolveText(cfg *Config, agentID, raw string) (string, string, string, error) {
	relative := normalizeQuotedPathArg(raw)
	if relative == "" {
		return "", "", "", errors.New("文字文件不能为空")
	}
	clean := filepath.Clean(relative)
	if filepath.IsAbs(clean) || strings.HasPrefix(relative, "~") || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", "", "", errors.New("文字文件必须位于当前 Agent 工作目录")
	}
	workspace, err := getWorkspaceByAgentID(cfg, agentID)
	if err != nil {
		return "", "", "", errors.New("Agent 不存在")
	}
	path, err := resolveCaseInsensitiveUnderRoot(workspace, clean)
	if err != nil || !ensurePathWithinRoot(workspace, path) {
		return "", "", "", errors.New("文字文件必须位于当前 Agent 工作目录")
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil || !ensurePathWithinRoot(workspace, path) {
		return "", "", "", errors.New("无法读取文字文件")
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || !info.Mode().IsRegular() {
		return "", "", "", errors.New("文字文件无效")
	}
	if info.Size() <= 0 || info.Size() > voxcpmTextLimit {
		return "", "", "", errors.New("文字文件不能为空且不能超过 1 MB")
	}
	data, err := os.ReadFile(path)
	if err != nil || !utf8.Valid(data) {
		return "", "", "", errors.New("文字文件必须是 UTF-8 文本")
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return "", "", "", errors.New("文字文件不能为空")
	}
	rel, err := filepath.Rel(workspace, path)
	if err != nil {
		return "", "", "", errors.New("文字文件必须位于当前 Agent 工作目录")
	}
	return filepath.ToSlash(rel), path, text, nil
}
func voxcpmResolveReference(cfg *Config, agentID, raw string) (string, string, error) {
	if strings.TrimSpace(normalizeQuotedPathArg(raw)) == "" {
		return "", "", nil
	}
	path, _, status, message := resolveMediaPreviewFile(cfg, agentID, raw)
	if status != 0 {
		return "", "", errors.New(message)
	}
	mime, ok := mediaPreviewMIMEType(path)
	if !ok || !strings.HasPrefix(mime, "audio/") {
		return "", "", errors.New("参考文件必须是音频")
	}
	workspace, err := getWorkspaceByAgentID(cfg, agentID)
	if err != nil {
		return "", "", errors.New("Agent 不存在")
	}
	rel, err := filepath.Rel(workspace, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", errors.New("参考音频必须位于当前 Agent 工作目录")
	}
	return filepath.ToSlash(rel), path, nil
}

func (m *voxcpmTaskManager) generate(ctx context.Context, task voxcpmTask) error {
	executable, ok := voxcpmExecutable(time.Hour)
	if !ok {
		return errors.New("未检测到可用的 voxcpm")
	}
	m.appendLog(task.ID, taskPythonExecutionEnvironment(executable))
	scenario, ok := voxcpmScenarioFor(task.Scenario)
	if !ok {
		return errors.New("配音场景无效")
	}
	_, textPath, text, err := voxcpmResolveText(m.cfg, task.AgentID, task.TextPath)
	if err != nil {
		return err
	}
	_, reference, err := voxcpmResolveReference(m.cfg, task.AgentID, task.ReferenceAudioPath)
	if err != nil {
		return err
	}
	workspace, err := getWorkspaceByAgentID(m.cfg, task.AgentID)
	if err != nil {
		return errors.New("Agent 不存在")
	}
	target, err := resolveCaseInsensitiveUnderRoot(workspace, filepath.FromSlash(task.OutputPath))
	if err != nil || !ensureWritablePathWithinRoot(workspace, target) {
		return errors.New("无法创建音频输出路径")
	}
	if _, err := os.Lstat(target); err == nil {
		return errors.New("输出 WAV 已存在")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return errors.New("无法创建 audios 目录")
	}
	m.appendLog(task.ID, "原始文字文件绝对路径："+textPath)
	if reference == "" {
		m.appendLog(task.ID, "参考音频绝对路径：未使用（design 模式）。")
	} else {
		m.appendLog(task.ID, "参考音频绝对路径："+reference)
	}
	m.appendLog(task.ID, "输出 WAV 绝对路径："+target)
	m.appendLog(task.ID, "VoxCPM 可执行文件："+executable)
	temporary := target + ".voxcpm-" + strconv.FormatInt(task.ID, 10) + ".part.wav"
	temporaryFile, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return errors.New("无法创建配音临时文件")
	}
	if err := temporaryFile.Close(); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	completed := false
	defer func() {
		if !completed {
			_ = os.Remove(temporary)
		}
	}()
	device, deviceReason := voxcpmTaskPreferredDevice(ctx, executable)
	if deviceReason != "" {
		m.appendLog(task.ID, deviceReason)
	}
	cloning := reference != ""
	control := voxcpmEffectiveControl(scenario, task.Control, cloning)
	m.appendLog(task.ID, voxcpmScenarioLogParameters(scenario, control, cloning))
	if cloning {
		m.appendLog(task.ID, "参考音色：已选择，使用 clone 模式。")
		m.appendLog(task.ID, "表达风格仅适用于自由生成，当前不传 --control。")
	} else {
		m.appendLog(task.ID, "参考音色：未选择，使用 design 模式自由生成声音。")
		if task.Control != "" {
			m.appendLog(task.ID, "已使用任务填写的表达风格："+task.Control)
		} else if control != "" {
			m.appendLog(task.ID, "未填写表达风格，已使用场景内置表达风格："+control)
		} else {
			m.appendLog(task.ID, "未填写表达风格，当前场景不传 --control。")
		}
	}
	m.progress(task.ID, 10)
	// Do not let VoxCPM invoke snapshot_download itself: that path can keep a
	// child process at "Loading VoxCPM model..." without useful progress. The
	// controlled ModelScope downloader has a header timeout, a no-byte-progress
	// timeout, task logs, and SHA-256 verification for every file.
	m.appendLog(task.ID, "正在从 ModelScope 国内源检查并准备 VoxCPM2 模型（下载过程会显示进度）。模型清单源："+voxcpmModelScopeFilesURL+"；模型文件源："+voxcpmModelScopeResolveBase)
	modelPath, mirrorErr := voxcpmPrepareLocalModel(m, ctx, task.ID)
	if mirrorErr != nil {
		return fmt.Errorf("ModelScope 国内源准备 VoxCPM2 模型失败：%w", mirrorErr)
	}
	m.appendLog(task.ID, "ModelScope 国内源模型已校验，开始从本地加载 VoxCPM2。")
	diagnostics, err := m.runGenerate(ctx, task.ID, executable, text, reference, temporary, device, scenario, control, modelPath)
	if err != nil {
		if ctx.Err() != nil {
			return context.Canceled
		}
		if voxcpmCLIArgumentError(diagnostics) {
			if cloning {
				m.appendLog(task.ID, "检测到 VoxCPM CLI 参数接口不兼容；旧版克隆接口还需要参考文本，当前任务未提供该文本，无法安全回退。")
				return errors.New("当前 VoxCPM CLI 不兼容，且参考音色克隆需要参考文本才能回退")
			}
			m.appendLog(task.ID, "检测到 VoxCPM CLI 参数接口不兼容，已回退到兼容的直接合成模式（CPU、忽略表达风格）重新尝试。")
			if _, retryErr := m.runGenerateFlat(ctx, task.ID, executable, text, temporary, scenario, modelPath); retryErr != nil {
				if ctx.Err() != nil {
					return context.Canceled
				}
				return retryErr
			}
			m.appendLog(task.ID, "VoxCPM 已使用兼容直接合成模式完成。")
		} else if voxcpmGPUDevice(device) {
			m.appendLog(task.ID, "VoxCPM 使用 "+strings.ToUpper(device)+" 配音失败，已自动回退到 CPU 重新尝试一次。")
			cpuDiagnostics, retryErr := m.runGenerate(ctx, task.ID, executable, text, reference, temporary, "cpu", scenario, control, modelPath)
			if retryErr != nil {
				if ctx.Err() != nil {
					return context.Canceled
				}
				if voxcpmCLIArgumentError(cpuDiagnostics) && !cloning {
					m.appendLog(task.ID, "CPU 回退检测到 VoxCPM CLI 参数接口不兼容，已继续回退到兼容的直接合成模式（忽略表达风格）重新尝试。")
					if _, flatErr := m.runGenerateFlat(ctx, task.ID, executable, text, temporary, scenario, modelPath); flatErr == nil {
						m.appendLog(task.ID, "VoxCPM 已使用兼容直接合成模式完成。")
					} else {
						if ctx.Err() != nil {
							return context.Canceled
						}
						return flatErr
					}
				} else if voxcpmCLIArgumentError(cpuDiagnostics) && cloning {
					m.appendLog(task.ID, "CPU 回退检测到 VoxCPM CLI 参数接口不兼容；旧版克隆接口还需要参考文本，当前任务未提供该文本，无法安全回退。")
					return errors.New("当前 VoxCPM CLI 不兼容，且参考音色克隆需要参考文本才能回退")
				} else {
					return retryErr
				}
			}
			m.appendLog(task.ID, "VoxCPM 已使用 CPU 回退完成。")
		} else {
			return err
		}
	}
	info, err := os.Stat(temporary)
	if err != nil || info.IsDir() || info.Size() == 0 {
		return errors.New("voxcpm 未生成有效 WAV")
	}
	if err := os.Link(temporary, target); err != nil {
		if errors.Is(err, os.ErrExist) {
			return errors.New("输出 WAV 已存在")
		}
		return fmt.Errorf("保存 WAV 失败: %w", err)
	}
	if err := os.Remove(temporary); err != nil {
		return err
	}
	completed = true
	m.progress(task.ID, 100)
	return nil
}

func (m *voxcpmTaskManager) runGenerate(ctx context.Context, taskID int64, executable, text, reference, output, device string, scenario voxcpmScenario, control, modelPath string) (string, error) {
	cloning := strings.TrimSpace(reference) != ""
	command := "design"
	if cloning {
		command = "clone"
	}
	args := []string{command, "--text", text, "--output", output,
		"--cfg-value", strconv.FormatFloat(scenario.CFGValue, 'f', -1, 64),
		"--inference-timesteps", strconv.Itoa(scenario.InferenceTimesteps)}
	if cloning {
		args = append(args, "--reference-audio", reference)
	}
	if control != "" {
		args = append(args, "--control", control)
	}
	if scenario.Normalize {
		args = append(args, "--normalize")
	}
	if scenario.Denoise && cloning {
		args = append(args, "--denoise")
	} else {
		// The CLI enables ZipEnhancer by default. A task that has not requested
		// denoising must disable it explicitly, otherwise it starts an unrelated
		// lazy ModelScope download while reporting only "Loading VoxCPM model...".
		args = append(args, "--no-denoiser")
	}
	if strings.TrimSpace(device) != "" {
		args = append(args, "--device", strings.TrimSpace(device))
	}
	if strings.TrimSpace(modelPath) != "" {
		args = append(args, "--model-path", modelPath)
	}
	return m.runGenerateArgs(ctx, taskID, executable, output, args)
}

// runGenerateFlat targets the older VoxCPM single-parser CLI. That CLI has no
// design subcommand, no expression-control flag, and no device selector; it
// chooses CPU automatically when CUDA is unavailable.
func (m *voxcpmTaskManager) runGenerateFlat(ctx context.Context, taskID int64, executable, text, output string, scenario voxcpmScenario, modelPath string) (string, error) {
	args := []string{"--text", text, "--output", output,
		"--cfg-value", strconv.FormatFloat(scenario.CFGValue, 'f', -1, 64),
		"--inference-timesteps", strconv.Itoa(scenario.InferenceTimesteps)}
	if scenario.Normalize {
		args = append(args, "--normalize")
	}
	// Flat-mode generation has no reference input, so denoising cannot be part
	// of the requested operation either.
	args = append(args, "--no-denoiser")
	if strings.TrimSpace(modelPath) != "" {
		args = append(args, "--model-path", modelPath)
	}
	return m.runGenerateArgs(ctx, taskID, executable, output, args)
}

func (m *voxcpmTaskManager) runGenerateArgs(ctx context.Context, taskID int64, executable, output string, args []string) (string, error) {
	// A failed backend may leave partial audio in place. Start each fallback
	// attempt from an empty task-private temporary file.
	if err := os.Truncate(output, 0); err != nil {
		return "", fmt.Errorf("无法重置配音临时文件: %w", err)
	}
	m.appendLog(taskID, "实际 VoxCPM 参数："+strings.Join(voxcpmLoggedArgs(args), " "))
	cmd := voxcpmCommandContext(ctx, executable, args...)
	// The command always receives --model-path. Make an unexpected Hub request
	// fail immediately rather than silently creating an uncontrolled download.
	cmd.Env = append(os.Environ(), "HF_HUB_OFFLINE=1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("无法启动 voxcpm: %w", err)
	}
	var copiedStdout, copiedStderr strings.Builder
	var captures sync.WaitGroup
	captures.Add(2)
	go func() { defer captures.Done(); m.capture(taskID, stdout, &copiedStdout) }()
	go func() { defer captures.Done(); m.capture(taskID, stderr, &copiedStderr) }()
	// Wait closes its pipe readers. Drain both streams before calling it so a
	// failed process cannot turn its real error into a misleading closed-pipe log.
	captures.Wait()
	err = cmd.Wait()
	diagnostics := copiedStdout.String() + "\n" + copiedStderr.String()
	if err != nil {
		if ctx.Err() != nil {
			return diagnostics, context.Canceled
		}
		return diagnostics, taskExecutionError("VoxCPM", err, diagnostics)
	}
	return diagnostics, nil
}

// The text is already traceable through the controlled input file path. Keep
// the parameter name in the reproducibility log without duplicating a user's
// potentially sensitive prose into the task record.
func voxcpmLoggedArgs(args []string) []string {
	logged := append([]string(nil), args...)
	for index := 0; index+1 < len(logged); index++ {
		if logged[index] == "--text" {
			logged[index+1] = "<由原始文字文件读取>"
		}
	}
	return logged
}

func voxcpmCLIArgumentError(diagnostics string) bool {
	value := strings.ToLower(diagnostics)
	return strings.Contains(value, "unrecognized arguments") ||
		strings.Contains(value, "unknown option") ||
		strings.Contains(value, "unknown command") ||
		strings.Contains(value, "invalid choice") ||
		strings.Contains(value, "usage: voxcpm")
}

type voxcpmModelScopeFile struct {
	Path   string `json:"Path"`
	SHA256 string `json:"Sha256"`
	Size   int64  `json:"Size"`
	Type   string `json:"Type"`
}

type voxcpmModelScopeFilesResponse struct {
	Code    int    `json:"Code"`
	Message string `json:"Message"`
	Data    struct {
		Files []voxcpmModelScopeFile `json:"Files"`
	} `json:"Data"`
}

func voxcpmModelMirrorCacheRoot() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("DEEPRIGHT_VOXCPM_MODEL_CACHE")); configured != "" {
		return configured, nil
	}
	cache, err := voxcpmUserCacheDir()
	if err != nil {
		return "", fmt.Errorf("无法确定用户模型缓存目录: %w", err)
	}
	return filepath.Join(cache, "deepright", "voxcpm"), nil
}

func voxcpmModelScopeManifest(ctx context.Context) ([]voxcpmModelScopeFile, error) {
	// The manifest is small, so give it a bounded total lifetime as well as the
	// transport's response-header timeout. Otherwise a peer that sends headers
	// and then stops before the JSON body could leave a task waiting forever.
	manifestCtx, cancel := context.WithTimeout(ctx, voxcpmModelHeaderTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(manifestCtx, http.MethodGet, voxcpmModelScopeFilesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("无法创建 ModelScope 清单请求: %w", err)
	}
	response, err := voxcpmHTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("请求 ModelScope 清单失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("ModelScope 清单返回 HTTP %d", response.StatusCode)
	}
	var payload voxcpmModelScopeFilesResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("读取 ModelScope 清单失败: %w", err)
	}
	if payload.Code != http.StatusOK {
		return nil, fmt.Errorf("ModelScope 清单错误: %s", strings.TrimSpace(payload.Message))
	}
	files := make([]voxcpmModelScopeFile, 0, len(payload.Data.Files))
	contains := map[string]bool{}
	for _, file := range payload.Data.Files {
		if strings.ToLower(strings.TrimSpace(file.Type)) != "blob" {
			continue
		}
		if !voxcpmModelRelativePath(file.Path) || file.Size < 0 || len(strings.TrimSpace(file.SHA256)) != 64 {
			return nil, errors.New("ModelScope 返回了无效的 VoxCPM2 模型清单")
		}
		file.Path = filepath.ToSlash(file.Path)
		file.SHA256 = strings.ToLower(strings.TrimSpace(file.SHA256))
		files = append(files, file)
		contains[file.Path] = true
	}
	if len(files) == 0 || !contains["config.json"] || !contains["model.safetensors"] || !contains["audiovae.pth"] {
		return nil, errors.New("ModelScope VoxCPM2 清单缺少必要模型文件")
	}
	return files, nil
}

func voxcpmModelRelativePath(raw string) bool {
	path := filepath.ToSlash(strings.TrimSpace(raw))
	if path == "" || strings.HasPrefix(path, "/") || strings.Contains(path, "\x00") {
		return false
	}
	for _, part := range strings.Split(path, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func voxcpmModelScopeFileURL(relative string) string {
	return voxcpmModelFileURL(voxcpmModelScopeResolveBase, relative)
}

func voxcpmModelBackupFileURL(relative string) string {
	return voxcpmModelFileURL(voxcpmHuggingFaceResolveBase, relative)
}

func voxcpmModelFileURL(base, relative string) string {
	parts := strings.Split(filepath.ToSlash(relative), "/")
	for index, part := range parts {
		parts[index] = url.PathEscape(part)
	}
	return strings.TrimRight(base, "/") + "/" + strings.Join(parts, "/")
}

func voxcpmModelDirectoryAvailable(directory string, files []voxcpmModelScopeFile) bool {
	for _, file := range files {
		path := filepath.Join(directory, filepath.FromSlash(file.Path))
		info, err := os.Stat(path)
		if err != nil || info.IsDir() || info.Size() != file.Size {
			return false
		}
		input, err := os.Open(path)
		if err != nil {
			return false
		}
		hash := sha256.New()
		_, copyErr := io.Copy(hash, input)
		closeErr := input.Close()
		if copyErr != nil || closeErr != nil || fmt.Sprintf("%x", hash.Sum(nil)) != file.SHA256 {
			return false
		}
	}
	return true
}

// downloadVoxCPM2FromModelScope obtains the exact snapshot listed by the
// OpenBMB ModelScope repository. Every file is SHA-256 verified before the
// staging directory is made visible to VoxCPM.
func (m *voxcpmTaskManager) downloadVoxCPM2FromModelScope(ctx context.Context, taskID int64) (string, error) {
	voxcpmModelMirrorMu.Lock()
	defer voxcpmModelMirrorMu.Unlock()

	files, err := voxcpmModelScopeManifest(ctx)
	if err != nil {
		return "", err
	}
	cacheRoot, err := voxcpmModelMirrorCacheRoot()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
		return "", fmt.Errorf("无法创建 VoxCPM2 模型缓存目录: %w", err)
	}
	destination := filepath.Join(cacheRoot, "OpenBMB--VoxCPM2")
	if voxcpmModelDirectoryAvailable(destination, files) {
		m.appendLog(taskID, "检测到已校验的 ModelScope VoxCPM2 本地缓存，跳过重复下载。")
		return destination, nil
	}
	staging := destination + ".deepright.partial"
	if _, err := os.Lstat(destination); err == nil {
		// Preserve an existing incomplete user cache rather than overwriting it.
		// The deterministic staging path survives a cancelled task and is reused
		// by a retry, so completed model-file parts are never discarded.
		if _, stagingErr := os.Stat(staging); errors.Is(stagingErr, os.ErrNotExist) {
			preserved := destination + ".incomplete-" + strconv.FormatInt(time.Now().UnixNano(), 10)
			if err := os.Rename(destination, preserved); err != nil {
				return "", fmt.Errorf("无法保留不完整的 VoxCPM2 模型缓存: %w", err)
			}
			m.appendLog(taskID, "已保留未校验的旧 VoxCPM2 缓存："+taskLogAbsolutePath(preserved))
		}
	}
	m.appendLog(taskID, "VoxCPM2 模型下载目标绝对目录："+taskLogAbsolutePath(destination))
	m.appendLog(taskID, "VoxCPM2 模型将使用多线程分段下载并保留断点；任务取消或网络中断后重新开始会继续未完成分段。")
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return "", fmt.Errorf("无法创建 VoxCPM2 模型临时目录: %w", err)
	}
	for index, file := range files {
		m.appendLog(taskID, fmt.Sprintf("正在从 ModelScope 下载模型文件（%d/%d）：%s；下载源：%s；最终目标绝对路径：%s", index+1, len(files), file.Path, voxcpmModelScopeFileURL(file.Path), taskLogAbsolutePath(filepath.Join(destination, filepath.FromSlash(file.Path)))))
		if err := m.downloadVoxCPM2ModelFile(ctx, taskID, staging, file); err != nil {
			return "", err
		}
	}
	if !voxcpmModelDirectoryAvailable(staging, files) {
		return "", errors.New("VoxCPM2 模型整体校验失败")
	}
	if err := os.Rename(staging, destination); err != nil {
		return "", fmt.Errorf("无法发布已下载的 VoxCPM2 模型: %w", err)
	}
	return destination, nil
}

func (m *voxcpmTaskManager) downloadVoxCPM2ModelFile(ctx context.Context, taskID int64, directory string, file voxcpmModelScopeFile) error {
	target := filepath.Join(directory, filepath.FromSlash(file.Path))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("无法创建模型文件目录: %w", err)
	}
	temporaryPath := target + ".deepright.part"
	lastProgress := int64(-1)
	_, err := downloadModelWithFallback(ctx, resumableModelDownloadConfig{
		Client:       voxcpmHTTPClient,
		URL:          voxcpmModelScopeFileURL(file.Path),
		BackupURL:    voxcpmModelBackupFileURL(file.Path),
		PartPath:     temporaryPath,
		ExpectedSize: file.Size,
		IdleTimeout:  voxcpmModelIdleTimeout,
		Workers:      modelTaskDownloadWorkers(m.cfg),
		Retries:      modelTaskDownloadRetries(m.cfg),
		Progress: func(copied, total int64) {
			if total < 10*1024*1024 {
				return
			}
			progress := copied * 100 / total
			if progress >= 100 || progress/10 > lastProgress/10 {
				lastProgress = progress
				m.appendLog(taskID, fmt.Sprintf("ModelScope 文件 %s 已下载 %d%%。", file.Path, progress))
			}
		},
		PartProgress: func(worker, part, parts int, copied, total int64) {
			if total <= 0 {
				return
			}
			m.appendLog(taskID, fmt.Sprintf("ModelScope 文件 %s 下载线程 %d，分段 %d/%d 已下载 %d%%。", file.Path, worker, part, parts, copied*100/total))
		},
		Retry: func(worker, part, parts, retry, retries int, err error) {
			m.appendLog(taskID, "ModelScope 文件 "+file.Path+" "+modelDownloadRetryMessage(worker, part, parts, retry, retries, err))
		},
	}, func(message string) { m.appendLog(taskID, "ModelScope 文件 "+file.Path+"："+message) })
	if err != nil {
		return fmt.Errorf("下载 ModelScope 文件 %s 失败: %w", file.Path, err)
	}
	input, err := os.Open(temporaryPath)
	if err != nil {
		return fmt.Errorf("无法读取模型文件 %s: %w", file.Path, err)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, input); err != nil {
		_ = input.Close()
		return fmt.Errorf("校验模型文件 %s 失败: %w", file.Path, err)
	}
	if err := input.Close(); err != nil {
		return fmt.Errorf("关闭模型文件 %s 失败: %w", file.Path, err)
	}
	if actual := fmt.Sprintf("%x", hash.Sum(nil)); actual != file.SHA256 {
		_ = os.Remove(temporaryPath)
		removeModelDownloadState(temporaryPath)
		return fmt.Errorf("ModelScope 文件 %s SHA-256 校验失败", file.Path)
	}
	if err := os.Rename(temporaryPath, target); err != nil {
		return fmt.Errorf("写入模型文件 %s 失败: %w", file.Path, err)
	}
	removeModelDownloadState(temporaryPath)
	return nil
}

func voxcpmGPUDevice(device string) bool {
	switch strings.ToLower(strings.TrimSpace(device)) {
	case "cuda", "mps":
		return true
	default:
		return false
	}
}

// voxcpmPreferredDevice probes the Python interpreter used by the installed
// voxcpm command, then passes the selected device explicitly to VoxCPM. This
// lets the queue distinguish a GPU execution failure from a CPU execution
// failure and guarantees that CPU fallback is attempted at most once.
func voxcpmPreferredDevice(ctx context.Context, voxcpmPath string) (string, string) {
	interpreter := voxcpmPythonInterpreter(voxcpmPath)
	if interpreter == "" {
		return "cpu", "未能验证 VoxCPM 的 GPU 环境，已使用 CPU。"
	}
	probeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	command := voxcpmCommandContext(probeCtx, interpreter, "-c", "import torch; print('cuda' if torch.cuda.is_available() else ('mps' if getattr(torch.backends, 'mps', None) and torch.backends.mps.is_available() else 'cpu'))")
	result, err := command.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(result))
		detail = strings.ReplaceAll(detail, "\n", " ")
		if len(detail) > 240 {
			detail = detail[:240] + "…"
		}
		if detail != "" {
			return "cpu", "VoxCPM GPU 探测失败（" + filepath.Base(interpreter) + "）：" + detail + "；已使用 CPU。"
		}
		return "cpu", "VoxCPM GPU 探测失败（" + filepath.Base(interpreter) + "）：" + err.Error() + "；已使用 CPU。"
	}
	switch strings.TrimSpace(string(result)) {
	case "cuda":
		return "cuda", "检测到 CUDA GPU，VoxCPM 优先使用 GPU。"
	case "mps":
		return "mps", "检测到 Apple MPS GPU，VoxCPM 优先使用 GPU。"
	default:
		return "cpu", "未检测到 VoxCPM 可用 GPU，已使用 CPU。"
	}
}

func voxcpmPythonInterpreter(voxcpmPath string) string {
	data, err := os.ReadFile(voxcpmPath)
	if err != nil {
		return ""
	}
	firstLine := strings.TrimSpace(strings.SplitN(string(data), "\n", 2)[0])
	if !strings.HasPrefix(firstLine, "#!") {
		return ""
	}
	parts := strings.Fields(strings.TrimSpace(strings.TrimPrefix(firstLine, "#!")))
	if len(parts) == 0 {
		return ""
	}
	if filepath.Base(parts[0]) == "env" {
		if len(parts) < 2 {
			return ""
		}
		candidate := parts[len(parts)-1]
		if candidate == "-S" || strings.HasPrefix(candidate, "-") {
			return ""
		}
		resolved, err := voxcpmTaskLookPath(candidate)
		if err != nil {
			return ""
		}
		return resolved
	}
	return parts[0]
}

func (m *voxcpmTaskManager) capture(id int64, reader io.Reader, copied *strings.Builder) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 128*1024)
	// huggingface_hub updates tqdm in place with carriage returns. Splitting on
	// them avoids a task that appears frozen at “0%” while bytes are arriving.
	scanner.Split(splitRembgOutput)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			m.appendLog(id, line)
			if copied != nil && copied.Len() < 128*1024 {
				remaining := 128*1024 - copied.Len()
				if len(line) > remaining {
					line = line[:remaining]
				}
				copied.WriteString(line)
				copied.WriteByte('\n')
			}
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, os.ErrClosed) && !strings.Contains(strings.ToLower(err.Error()), "file already closed") {
		m.appendLog(id, "读取 voxcpm 输出失败："+err.Error())
	}
}
func (m *voxcpmTaskManager) progress(id int64, value int) {
	_, _ = m.db.Exec(`UPDATE voxcpm_task SET progress=CASE WHEN progress>? THEN progress ELSE ? END,updated_at=? WHERE id=? AND status=?`, value, value, voxcpmTimestamp(), id, voxcpmTaskRunning)
}
func (m *voxcpmTaskManager) appendLog(id int64, message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	if len(message) > 4096 {
		message = message[:4096] + "…"
	}
	now := voxcpmTimestamp()
	_, _ = m.db.Exec(`UPDATE voxcpm_task SET logs=CASE WHEN length(logs)>? THEN substr(logs,-?) ELSE logs END || ?,updated_at=? WHERE id=?`, voxcpmTaskLogLimit-8192, voxcpmTaskLogLimit/2, "["+now+"] "+message+"\n", now, id)
}
func (m *voxcpmTaskManager) finish(id int64, status string, progress int, message string) {
	m.appendLog(id, message)
	_, _ = m.db.Exec(`UPDATE voxcpm_task SET status=?,progress=?,updated_at=? WHERE id=?`, status, progress, voxcpmTimestamp(), id)
	notifyIntegrationSleepConditionChanged(m.cfg)
}
func (m *voxcpmTaskManager) cancelled(id int64) bool {
	var requested int
	return m.db.QueryRow(`SELECT cancel_requested FROM voxcpm_task WHERE id=?`, id).Scan(&requested) != nil || requested != 0
}
func (m *voxcpmTaskManager) cancelRunning() {
	m.mu.Lock()
	cancel := m.cancel
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func voxcpmOutputName(name string) (string, bool) {
	name = strings.TrimSpace(normalizeQuotedPathArg(name))
	if !strings.EqualFold(filepath.Ext(name), ".wav") {
		name += ".wav"
	}
	return name, isSafeAudioMixOutputName(name)
}
func (m *voxcpmTaskManager) allocate(agentID, workspace, requested string, forceTimestamp bool, reserved map[string]bool, except int64) (string, string, error) {
	requested, valid := voxcpmOutputName(requested)
	if !valid {
		return "", "", errors.New("配音名称只支持 .wav")
	}
	stem := strings.TrimSuffix(requested, filepath.Ext(requested))
	stamp := voxcpmNow().Format("20060102_150405.000000000")
	for attempt := 0; attempt < 1000; attempt++ {
		name := requested
		if forceTimestamp || attempt > 0 {
			name = stem + "_" + stamp
			sequence := attempt
			if !forceTimestamp {
				sequence--
			}
			if sequence > 0 {
				name += "_" + strconv.Itoa(sequence)
			}
			name += ".wav"
		}
		relative := filepath.ToSlash(filepath.Join("audios", name))
		if reserved[relative] {
			continue
		}
		absolute, err := resolveCaseInsensitiveUnderRoot(workspace, filepath.FromSlash(relative))
		if err != nil || !ensureWritablePathWithinRoot(workspace, absolute) {
			return "", "", errors.New("无法创建 WAV 输出路径")
		}
		var used int
		if err := m.db.QueryRow(`SELECT COUNT(*) FROM voxcpm_task WHERE agent_id=? AND output_path=? AND id<>?`, agentID, relative, except).Scan(&used); err != nil {
			return "", "", err
		}
		if used != 0 {
			continue
		}
		if _, err := os.Lstat(absolute); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", "", err
		}
		reserved[relative] = true
		return relative, absolute, nil
	}
	return "", "", errors.New("无法分配不重复的 WAV 文件名")
}
func (m *voxcpmTaskManager) create(request voxcpmTaskCreateRequest) ([]voxcpmTask, error) {
	if m == nil || m.cfg == nil || m.db == nil {
		return nil, errors.New("配音任务服务未就绪")
	}
	request.AgentID = strings.TrimSpace(request.AgentID)
	requested, valid := voxcpmOutputName(request.OutputName)
	if !isValidMediaPreviewAgentID(request.AgentID) || !valid || len(request.Tasks) == 0 || len(request.Tasks) > 5 {
		return nil, errors.New("agentId、WAV 名称和 1 至 5 条语音任务不能为空")
	}
	if _, ok := voxcpmExecutable(time.Hour); !ok {
		return nil, errors.New("未检测到可用的 voxcpm")
	}
	m.createMu.Lock()
	defer m.createMu.Unlock()
	workspace, err := getWorkspaceByAgentID(m.cfg, request.AgentID)
	if err != nil {
		return nil, errors.New("Agent 不存在")
	}
	type source struct{ textPath, textSavedAs, referencePath, referenceSavedAs, output, saved, scenario, control string }
	sources := make([]source, 0, len(request.Tasks))
	reserved := map[string]bool{}
	for _, item := range request.Tasks {
		scenario, ok := voxcpmScenarioFor(item.Scenario)
		if !ok {
			return nil, errors.New("配音场景无效")
		}
		textPath, textSavedAs, _, err := voxcpmResolveText(m.cfg, request.AgentID, item.TextPath)
		if err != nil {
			return nil, err
		}
		referencePath, referenceSavedAs, err := voxcpmResolveReference(m.cfg, request.AgentID, item.ReferenceAudioPath)
		if err != nil {
			return nil, err
		}
		control := ""
		if referencePath == "" {
			control, err = voxcpmControlText(item.Control)
			if err != nil {
				return nil, err
			}
		}
		output, saved, err := m.allocate(request.AgentID, workspace, requested, len(request.Tasks) > 1, reserved, 0)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source{textPath, textSavedAs, referencePath, referenceSavedAs, output, saved, scenario.ID, control})
	}
	tx, err := m.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	now := voxcpmTimestamp()
	created := make([]voxcpmTask, 0, len(sources))
	for _, source := range sources {
		scenario, _ := voxcpmScenarioFor(source.scenario)
		mode := "VoxCPM design 自由生成"
		if source.referencePath != "" {
			mode = "VoxCPM 参考音色克隆"
		}
		logs := "[" + now + "] 已加入文字转语音队列（" + mode + "，场景：" + scenario.Name + "）。\n"
		if source.referencePath == "" {
			logs += "[" + now + "] 未选择参考音色，执行时将使用 design 模式自由生成声音。\n"
		} else {
			logs += "[" + now + "] 已选择参考音色，执行时将使用 clone 模式。\n"
		}
		if source.control != "" {
			logs += "[" + now + "] 已填写表达风格：" + source.control + "。\n"
		}
		result, err := tx.Exec(`INSERT INTO voxcpm_task(agent_id,text_path,text_saved_as,reference_audio_path,reference_audio_saved_as,requested_name,output_path,saved_as,scenario,control,status,progress,created_at,updated_at,logs,cancel_requested) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,0)`, request.AgentID, source.textPath, source.textSavedAs, source.referencePath, source.referenceSavedAs, requested, source.output, source.saved, source.scenario, source.control, voxcpmTaskQueued, 0, now, now, logs)
		if err != nil {
			return nil, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}
		created = append(created, voxcpmTask{ID: id, AgentID: request.AgentID, TextPath: source.textPath, TextSavedAs: source.textSavedAs, ReferenceAudioPath: source.referencePath, ReferenceAudioSavedAs: source.referenceSavedAs, RequestedName: requested, OutputPath: source.output, SavedAs: source.saved, Scenario: source.scenario, Control: source.control, Status: voxcpmTaskQueued, CreatedAt: now, UpdatedAt: now})
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	m.signal()
	return created, nil
}

const voxcpmTaskColumns = `SELECT id,agent_id,text_path,text_saved_as,reference_audio_path,reference_audio_saved_as,requested_name,output_path,saved_as,scenario,control,status,progress,started_at,created_at,updated_at,`

type voxcpmScanner interface{ Scan(...interface{}) error }

func scanVoxCPMTask(scanner voxcpmScanner) (voxcpmTask, error) {
	var task voxcpmTask
	var cancelled int
	err := scanner.Scan(&task.ID, &task.AgentID, &task.TextPath, &task.TextSavedAs, &task.ReferenceAudioPath, &task.ReferenceAudioSavedAs, &task.RequestedName, &task.OutputPath, &task.SavedAs, &task.Scenario, &task.Control, &task.Status, &task.Progress, &task.StartedAt, &task.CreatedAt, &task.UpdatedAt, &task.Logs, &cancelled)
	if task.Scenario == "" {
		task.Scenario = voxcpmScenarioBalanced
	}
	task.CancelAsked = cancelled != 0
	return task, err
}
func validVoxCPMStatus(status string) bool {
	switch status {
	case "", voxcpmTaskQueued, voxcpmTaskRunning, voxcpmTaskCompleted, voxcpmTaskCancelled, voxcpmTaskFailed:
		return true
	}
	return false
}
func (m *voxcpmTaskManager) list(agentID, status string, page int, logs bool) (voxcpmTaskPage, error) {
	result := voxcpmTaskPage{Tasks: []voxcpmTask{}, PageSize: 5}
	agentID = strings.TrimSpace(agentID)
	status = strings.TrimSpace(status)
	if !isValidMediaPreviewAgentID(agentID) || !validVoxCPMStatus(status) {
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
	if err := m.db.QueryRow(`SELECT COUNT(*) FROM voxcpm_task WHERE `+where, args...).Scan(&result.Total); err != nil {
		return result, err
	}
	last := (result.Total + result.PageSize - 1) / result.PageSize
	if last < 1 {
		last = 1
	}
	if page > last {
		page = last
	}
	result.Page, result.Status = page, status
	column := "''"
	if logs {
		column = "logs"
	}
	args = append(args, result.PageSize, (page-1)*result.PageSize)
	rows, err := m.db.Query(voxcpmTaskColumns+column+`,cancel_requested FROM voxcpm_task WHERE `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		task, err := scanVoxCPMTask(rows)
		if err != nil {
			return result, err
		}
		result.Tasks = append(result.Tasks, task)
	}
	return result, rows.Err()
}
func (m *voxcpmTaskManager) log(agentID string, id int64) (voxcpmTask, error) {
	if id <= 0 || !isValidMediaPreviewAgentID(strings.TrimSpace(agentID)) {
		return voxcpmTask{}, errors.New("任务参数无效")
	}
	return scanVoxCPMTask(m.db.QueryRow(voxcpmTaskColumns+`logs,cancel_requested FROM voxcpm_task WHERE id=? AND agent_id=?`, id, strings.TrimSpace(agentID)))
}
func (m *voxcpmTaskManager) cancelTask(agentID string, id int64) error {
	now := voxcpmTimestamp()
	result, err := m.db.Exec(`UPDATE voxcpm_task SET cancel_requested=1,status=CASE WHEN status=? THEN ? ELSE status END,updated_at=?,logs=CASE WHEN length(logs)>? THEN substr(logs,-?) ELSE logs END || ? WHERE id=? AND agent_id=? AND status IN (?,?)`, voxcpmTaskQueued, voxcpmTaskCancelled, now, voxcpmTaskLogLimit-4096, voxcpmTaskLogLimit/2, "["+now+"] 已请求取消任务。\n", id, strings.TrimSpace(agentID), voxcpmTaskQueued, voxcpmTaskRunning)
	if err != nil {
		return err
	}
	changed, _ := result.RowsAffected()
	if changed != 1 {
		return errors.New("仅可取消排队中或执行中的任务")
	}
	m.mu.Lock()
	should := m.runningID == id
	cancel := m.cancel
	m.mu.Unlock()
	if should && cancel != nil {
		cancel()
	}
	notifyIntegrationSleepConditionChanged(m.cfg)
	return nil
}
func (m *voxcpmTaskManager) restart(agentID string, id int64) error {
	agentID = strings.TrimSpace(agentID)
	var textPath, referencePath, requested string
	if err := m.db.QueryRow(`SELECT text_path,reference_audio_path,requested_name FROM voxcpm_task WHERE id=? AND agent_id=? AND status IN (?, ?)`, id, agentID, voxcpmTaskFailed, voxcpmTaskCancelled).Scan(&textPath, &referencePath, &requested); err != nil {
		return errors.New("仅可重新开始失败或已取消任务")
	}
	workspace, err := getWorkspaceByAgentID(m.cfg, agentID)
	if err != nil {
		return errors.New("Agent 不存在")
	}
	output, saved, err := m.allocate(agentID, workspace, requested, true, map[string]bool{}, id)
	if err != nil {
		return err
	}
	now := voxcpmTimestamp()
	result, err := m.db.Exec(`UPDATE voxcpm_task SET output_path=?,saved_as=?,status=?,progress=0,started_at='',cancel_requested=0,updated_at=?,logs='' WHERE id=? AND agent_id=? AND status IN (?, ?)`, output, saved, voxcpmTaskQueued, now, id, agentID, voxcpmTaskFailed, voxcpmTaskCancelled)
	if err != nil {
		return err
	}
	changed, _ := result.RowsAffected()
	if changed != 1 {
		return errors.New("重新开始任务失败")
	}
	m.signal()
	return nil
}
func (m *voxcpmTaskManager) deleteFailedOrCancelled(agentID string, id int64) error {
	result, err := m.db.Exec(`DELETE FROM voxcpm_task WHERE id=? AND agent_id=? AND status IN (?, ?)`, id, strings.TrimSpace(agentID), voxcpmTaskFailed, voxcpmTaskCancelled)
	if err != nil {
		return err
	}
	changed, _ := result.RowsAffected()
	if changed != 1 {
		return errors.New("仅可删除失败或已取消任务")
	}
	return nil
}

func readVoxCPMAction(w http.ResponseWriter, r *http.Request) (voxcpmTaskActionRequest, error) {
	var request voxcpmTaskActionRequest
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
func handleVoxCPMTasks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if voxcpmTasks == nil {
			writeVoxCPMJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": 1, "content": "文字转语音任务服务未就绪"})
			return
		}
		switch r.Method {
		case http.MethodGet:
			page := 1
			if raw := strings.TrimSpace(r.URL.Query().Get("page")); raw != "" {
				parsed, err := strconv.Atoi(raw)
				if err != nil || parsed < 1 {
					writeVoxCPMJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "页码必须是正整数"})
					return
				}
				page = parsed
			}
			value, err := voxcpmTasks.list(r.URL.Query().Get("agentId"), r.URL.Query().Get("status"), page, false)
			if err != nil {
				writeVoxCPMJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
				return
			}
			writeVoxCPMJSON(w, http.StatusOK, map[string]interface{}{"status": 0, "tasks": value.Tasks, "total": value.Total, "page": value.Page, "pageSize": value.PageSize, "taskStatus": value.Status})
		case http.MethodPost:
			var request voxcpmTaskCreateRequest
			decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
				writeVoxCPMJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "请求内容无效"})
				return
			}
			value, err := voxcpmTasks.create(request)
			if err != nil {
				writeVoxCPMJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
				return
			}
			writeVoxCPMJSON(w, http.StatusOK, map[string]interface{}{"status": 0, "tasks": value})
		default:
			w.Header().Set("Allow", "GET, POST")
			writeVoxCPMJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"status": 1, "content": "仅支持 GET 或 POST 请求"})
		}
	}
}
func voxcpmActionHandler(action func(*voxcpmTaskManager, voxcpmTaskActionRequest) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeVoxCPMJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"status": 1, "content": "仅支持 POST 请求"})
			return
		}
		if voxcpmTasks == nil {
			writeVoxCPMJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": 1, "content": "文字转语音任务服务未就绪"})
			return
		}
		request, err := readVoxCPMAction(w, r)
		if err == nil {
			err = action(voxcpmTasks, request)
		}
		if err != nil {
			writeVoxCPMJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		writeVoxCPMJSON(w, http.StatusOK, map[string]interface{}{"status": 0})
	}
}
func handleVoxCPMTaskCancel() http.HandlerFunc {
	return voxcpmActionHandler(func(m *voxcpmTaskManager, r voxcpmTaskActionRequest) error { return m.cancelTask(r.AgentID, r.ID) })
}
func handleVoxCPMTaskRestart() http.HandlerFunc {
	return voxcpmActionHandler(func(m *voxcpmTaskManager, r voxcpmTaskActionRequest) error { return m.restart(r.AgentID, r.ID) })
}
func handleVoxCPMTaskDelete() http.HandlerFunc {
	return voxcpmActionHandler(func(m *voxcpmTaskManager, r voxcpmTaskActionRequest) error {
		return m.deleteFailedOrCancelled(r.AgentID, r.ID)
	})
}
func handleVoxCPMTaskLog() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeVoxCPMJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"status": 1, "content": "仅支持 GET 请求"})
			return
		}
		if voxcpmTasks == nil {
			writeVoxCPMJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": 1, "content": "文字转语音任务服务未就绪"})
			return
		}
		id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
		if err == nil {
			var task voxcpmTask
			task, err = voxcpmTasks.log(r.URL.Query().Get("agentId"), id)
			if err == nil {
				writeVoxCPMJSON(w, http.StatusOK, map[string]interface{}{"status": 0, "task": task})
				return
			}
		}
		writeVoxCPMJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "任务参数无效"})
	}
}
