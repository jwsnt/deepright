package main

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
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	whisperScenarioChineseMeeting  = "chinese_meeting"
	whisperScenarioChineseAccurate = "chinese_accurate"
	whisperScenarioRealtime        = "realtime"
	whisperScenarioBatch           = "batch"
	whisperScenarioCPU             = "cpu"
	whisperScenarioMixedTechnical  = "mixed_technical"

	whisperTaskStatusQueued    = "queued"
	whisperTaskStatusRunning   = "running"
	whisperTaskStatusCompleted = "completed"
	whisperTaskStatusCancelled = "cancelled"
	whisperTaskStatusFailed    = "failed"
	whisperTaskLogLimit        = 128 * 1024
	// Loading Whisper may initialize native Python/ML dependencies on a cold
	// start, so dependency detection allows one minute before declaring it absent.
	whisperProbeTimeout = time.Minute
	// ModelScope's iic/Whisper-large-v3 checkpoint has the same SHA-256 as
	// OpenAI's official large-v3.pt. It is the domestic fallback for every
	// supported Whisper profile when the original checkpoint transfer fails.
	whisperDefaultModelScopeLargeV3URL = "https://modelscope.cn/models/iic/whisper-large-v3/resolve/master/large-v3.pt"
	whisperDefaultOfficialLargeV3URL   = "https://openaipublic.azureedge.net/main/whisper/models/e5b1a55b89c1367dacf97e3e19bfd829a01529dbfdeefa8caeb59b3f1b81dadb/large-v3.pt"
	whisperDefaultLargeV3SHA256        = "e5b1a55b89c1367dacf97e3e19bfd829a01529dbfdeefa8caeb59b3f1b81dadb"
	whisperDefaultLargeV3Size          = int64(3087371615)
	whisperModelHeaderTimeout          = 30 * time.Second
	whisperModelIdleTimeout            = 90 * time.Second
)

// WhisperCheckResponse intentionally mirrors the narrow FFmpeg check response:
// the browser can decide whether to show an installation request, but never
// receives a command line or executes an installer itself.
type WhisperCheckResponse struct {
	Available bool   `json:"available"`
	Install   string `json:"install,omitempty"`
	Content   string `json:"content,omitempty"`
	Status    int    `json:"status"`
}

type whisperCheckConfig struct {
	CacheFor time.Duration
	Install  string
}

type whisperTask struct {
	ID          int64  `json:"id"`
	AgentID     string `json:"agentId"`
	SourcePath  string `json:"sourcePath"`
	OutputPath  string `json:"outputPath,omitempty"`
	SavedAs     string `json:"savedAs,omitempty"`
	Scenario    string `json:"scenario"`
	Status      string `json:"status"`
	Progress    int    `json:"progress"`
	StartedAt   string `json:"startedAt,omitempty"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	Logs        string `json:"logs,omitempty"`
	CancelAsked bool   `json:"cancelRequested"`
}

type whisperTaskCreateRequest struct {
	AgentID string                  `json:"agentId"`
	Tasks   []whisperTaskCreateItem `json:"tasks,omitempty"`
	// Paths and Scenario preserve compatibility with clients from before
	// per-file scenarios were introduced.
	Paths    []string `json:"paths,omitempty"`
	Scenario string   `json:"scenario,omitempty"`
}

type whisperTaskCreateItem struct {
	Path     string `json:"path"`
	Scenario string `json:"scenario,omitempty"`
}

type whisperScenario struct {
	ID                      string
	Name                    string
	Model                   string
	Language                string
	Task                    string
	Temperature             float64
	BeamSize                int
	ConditionOnPreviousText bool
	InitialPrompt           string
	ForceCPU                bool
}

type whisperRuntimeModelArtifact struct {
	OfficialURL string
	BackupURL   string
	SHA256      string
}

var whisperRuntimeModelArtifacts = map[string]whisperRuntimeModelArtifact{
	"base": {
		OfficialURL: "https://openaipublic.azureedge.net/main/whisper/models/ed3a0b6b1c0edf879ad9b11b1af5a0e6ab5db9205f891f668f8b0e6c6326e34e/base.pt",
		BackupURL:   "https://modelscope.cn/models/iic/whisper-base/resolve/master/base.pt",
		SHA256:      "ed3a0b6b1c0edf879ad9b11b1af5a0e6ab5db9205f891f668f8b0e6c6326e34e",
	},
	"small": {
		OfficialURL: "https://openaipublic.azureedge.net/main/whisper/models/9ecf779972d90ba49c06d968637d720dd632c55bbf19d441fb42bf17a411e794/small.pt",
		BackupURL:   "https://modelscope.cn/models/iic/whisper-small/resolve/master/small.pt",
		SHA256:      "9ecf779972d90ba49c06d968637d720dd632c55bbf19d441fb42bf17a411e794",
	},
	"medium": {
		OfficialURL: "https://openaipublic.azureedge.net/main/whisper/models/345ae4da62f9b3d59415adc60127b97c714f32e89e936602e85993674d08dcb1/medium.pt",
		BackupURL:   "https://modelscope.cn/models/iic/whisper-medium/resolve/master/medium.pt",
		SHA256:      "345ae4da62f9b3d59415adc60127b97c714f32e89e936602e85993674d08dcb1",
	},
	"large-v3": {
		OfficialURL: whisperDefaultOfficialLargeV3URL,
		BackupURL:   whisperDefaultModelScopeLargeV3URL,
		SHA256:      whisperDefaultLargeV3SHA256,
	},
}

func whisperScenarioFor(raw string) (whisperScenario, bool) {
	switch strings.TrimSpace(raw) {
	case "", whisperScenarioChineseMeeting:
		return whisperScenario{ID: whisperScenarioChineseMeeting, Name: "中文会议", Model: "small", Language: "zh", Task: "transcribe", Temperature: 0, BeamSize: 5, ConditionOnPreviousText: true}, true
	case whisperScenarioChineseAccurate:
		return whisperScenario{ID: whisperScenarioChineseAccurate, Name: "专业转写", Model: "large-v3", Language: "zh", Task: "transcribe", Temperature: 0, BeamSize: 5, ConditionOnPreviousText: true, InitialPrompt: "以下内容为中文专业访谈，请准确保留人名、产品名、英文缩写及专业术语。"}, true
	case whisperScenarioRealtime:
		return whisperScenario{ID: whisperScenarioRealtime, Name: "低延迟", Model: "base", Language: "zh", Task: "transcribe", Temperature: 0, BeamSize: 1, ConditionOnPreviousText: false}, true
	case whisperScenarioBatch:
		return whisperScenario{ID: whisperScenarioBatch, Name: "批量处理", Model: "small", Language: "zh", Task: "transcribe", Temperature: 0, BeamSize: 3, ConditionOnPreviousText: true}, true
	case whisperScenarioCPU:
		return whisperScenario{ID: whisperScenarioCPU, Name: "CPU轻量", Model: "base", Language: "zh", Task: "transcribe", Temperature: 0, BeamSize: 1, ConditionOnPreviousText: true, ForceCPU: true}, true
	case whisperScenarioMixedTechnical:
		return whisperScenario{ID: whisperScenarioMixedTechnical, Name: "中英技术", Model: "medium", Language: "zh", Task: "transcribe", Temperature: 0, BeamSize: 5, ConditionOnPreviousText: true, InitialPrompt: "以下是中英混合的技术讨论。请保留英文产品名、代码名、API、缩写和数字，例如 GPT、Python、Docker、API。"}, true
	default:
		return whisperScenario{}, false
	}
}

// Whisper's command-line parser accepts title-cased boolean literals only.
func whisperCLIBoolean(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

func whisperScenarioLogParameters(scenario whisperScenario, device string) string {
	parameters := []string{
		"model=" + scenario.Model,
		"language=" + scenario.Language,
		"task=" + scenario.Task,
		"temperature=" + strconv.FormatFloat(scenario.Temperature, 'f', -1, 64),
		"beam_size=" + strconv.Itoa(scenario.BeamSize),
		"condition_on_previous_text=" + whisperCLIBoolean(scenario.ConditionOnPreviousText),
		"fp16=" + whisperCLIBoolean(device != "cpu"),
	}
	if scenario.InitialPrompt != "" {
		parameters = append(parameters, "initial_prompt=已设置")
	}
	return strings.Join(parameters, ", ")
}

type whisperTaskCancelRequest struct {
	AgentID string `json:"agentId"`
	ID      int64  `json:"id"`
}

type whisperTaskRestartRequest struct {
	AgentID string `json:"agentId"`
	ID      int64  `json:"id"`
}

type whisperTaskDeleteRequest struct {
	AgentID string `json:"agentId"`
	ID      int64  `json:"id"`
}

type whisperTaskPage struct {
	Tasks    []whisperTask `json:"tasks"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
	Status   string        `json:"taskStatus"`
}

type whisperTaskManager struct {
	cfg *Config
	db  *sql.DB

	createMu  sync.Mutex
	mu        sync.Mutex
	runningID int64
	cancel    context.CancelFunc
	wake      chan struct{}
}

var (
	whisperModelHTTPClient      = &http.Client{Transport: &http.Transport{ResponseHeaderTimeout: whisperModelHeaderTimeout}}
	whisperModelUserCacheDir    = os.UserCacheDir
	whisperModelMirrorMu        sync.Mutex
	whisperModelScopeLargeV3URL = whisperDefaultModelScopeLargeV3URL
	whisperOfficialLargeV3URL   = whisperDefaultOfficialLargeV3URL
	whisperLargeV3SHA256        = whisperDefaultLargeV3SHA256
	whisperLargeV3Size          = whisperDefaultLargeV3Size
)

var (
	whisperTaskLookPath        = exec.LookPath
	whisperTaskCommandContext  = exec.CommandContext
	whisperTaskNow             = time.Now
	whisperTaskUserHomeDir     = os.UserHomeDir
	whisperTaskPreferredDevice = whisperPreferredDevice
	whisperTasks               *whisperTaskManager
	whisperCheckCache          struct {
		sync.Mutex
		availableAt time.Time
		executable  string
	}
	whisperProgressPattern = regexp.MustCompile(`(?i)(?:^|[^0-9])(100|[1-9]?\d)\s*%`)
)

func whisperTaskTimestamp() string {
	return whisperTaskNow().Format(time.RFC3339Nano)
}

func ensureWhisperTaskSchema(db *sql.DB) error {
	if db == nil {
		return errors.New("whisper task database is not ready")
	}
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS whisper_task (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id TEXT NOT NULL,
			source_path TEXT NOT NULL,
			output_path TEXT NOT NULL DEFAULT '',
			saved_as TEXT NOT NULL DEFAULT '',
			scenario TEXT NOT NULL DEFAULT 'chinese_meeting',
			status TEXT NOT NULL DEFAULT 'queued',
			progress INTEGER NOT NULL DEFAULT 0,
			started_at TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			logs TEXT NOT NULL DEFAULT '',
			cancel_requested INTEGER NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_whisper_task_agent_created ON whisper_task(agent_id, id DESC);
		CREATE INDEX IF NOT EXISTS idx_whisper_task_queue ON whisper_task(status, cancel_requested, id);
	`)
	if err != nil {
		return err
	}
	// The original table did not retain task-specific transcription options.
	// Add the scenario column for existing databases without discarding queued
	// tasks; they continue with the default Chinese meeting profile.
	columns := map[string]bool{}
	rows, err := db.Query(`PRAGMA table_info(whisper_task)`)
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
		if _, err := db.Exec(`ALTER TABLE whisper_task ADD COLUMN scenario TEXT NOT NULL DEFAULT 'chinese_meeting'`); err != nil {
			return err
		}
	}
	_, err = db.Exec(`UPDATE whisper_task SET scenario = ? WHERE trim(scenario) = ''`, whisperScenarioChineseMeeting)
	return err
}

func parseWhisperCheckConfig(raw map[string]interface{}) (whisperCheckConfig, error) {
	if raw == nil {
		return whisperCheckConfig{}, errors.New("config/config.json.whisper 配置缺失")
	}
	value, ok := raw["whisper"]
	if !ok {
		return whisperCheckConfig{}, errors.New("config/config.json.whisper 配置缺失")
	}
	section, ok := value.(map[string]interface{})
	if !ok || section == nil {
		return whisperCheckConfig{}, errors.New("config/config.json.whisper 必须是对象")
	}
	hours, ok := ffmpegCheckPositiveHours(section["check"])
	if !ok {
		return whisperCheckConfig{}, errors.New("config/config.json.whisper.check 必须是正整数（小时）")
	}
	install, ok := section["install"].(string)
	install = strings.TrimSpace(install)
	if !ok || install == "" {
		return whisperCheckConfig{}, errors.New("config/config.json.whisper.install 必须是非空字符串")
	}
	return whisperCheckConfig{CacheFor: time.Duration(hours) * time.Hour, Install: install}, nil
}

func whisperExecutableCandidates() []string {
	candidates := make([]string, 0, 4)
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
	if path, err := whisperTaskLookPath("whisper"); err == nil {
		add(path)
	}
	// macOS `pip install --user` normally places console scripts here. The
	// desktop app does not inherit shell startup PATH additions, so discover
	// this standard Python user bin directory explicitly.
	if home, err := whisperTaskUserHomeDir(); err == nil {
		add(filepath.Join(home, ".local", "bin", "whisper"))
		for _, path := range mustGlob(filepath.Join(home, "Library", "Python", "*", "bin", "whisper")) {
			add(path)
		}
	}
	return candidates
}

func mustGlob(pattern string) []string {
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	return paths
}

func whisperExecutablePath(cacheFor time.Duration) (string, bool) {
	now := whisperTaskNow()
	candidates := whisperExecutableCandidates()
	if len(candidates) == 0 {
		return "", false
	}
	whisperCheckCache.Lock()
	availableAt := whisperCheckCache.availableAt
	cachedPath := whisperCheckCache.executable
	whisperCheckCache.Unlock()
	if !availableAt.IsZero() && now.Before(availableAt.Add(cacheFor)) && cachedPath != "" {
		for _, path := range candidates {
			if path == cachedPath {
				return path, true
			}
		}
	}
	// A PATH entry alone is not sufficient. In particular, an old virtualenv
	// wrapper may still be found after its Python package has been removed.
	// Verify that the same command which will run a task can start normally,
	// so the UI can offer installation before it opens the task dialog.
	for _, whisperPath := range candidates {
		probeCtx, cancel := context.WithTimeout(context.Background(), whisperProbeTimeout)
		err := whisperTaskCommandContext(probeCtx, whisperPath, "--help").Run()
		timedOut := probeCtx.Err() != nil
		cancel()
		if err != nil || timedOut {
			continue
		}
		whisperCheckCache.Lock()
		whisperCheckCache.availableAt = now
		whisperCheckCache.executable = whisperPath
		whisperCheckCache.Unlock()
		return whisperPath, true
	}
	return "", false
}

func whisperDependencyAvailable(cacheFor time.Duration) bool {
	_, available := whisperExecutablePath(cacheFor)
	return available
}

func writeWhisperCheckResp(w http.ResponseWriter, httpStatus int, resp WhisperCheckResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(resp)
}

func handleWhisperCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeWhisperCheckResp(w, http.StatusMethodNotAllowed, WhisperCheckResponse{Content: "仅支持 GET 请求", Status: 1})
			return
		}
		status, response := checkWhisperDependency()
		writeWhisperCheckResp(w, status, response)
	}
}

// checkWhisperDependency contains the complete availability rule shared by
// the HTTP endpoint and the asynchronous startup preflight.
func checkWhisperDependency() (int, WhisperCheckResponse) {
	raw, _, err := readIntegrationStartupConfigRaw()
	if err != nil {
		return http.StatusInternalServerError, WhisperCheckResponse{Content: "读取 config/config.json 失败: " + err.Error(), Status: 1}
	}
	config, err := parseWhisperCheckConfig(raw)
	if err != nil {
		return http.StatusBadRequest, WhisperCheckResponse{Content: err.Error(), Status: 1}
	}
	if !whisperDependencyAvailable(config.CacheFor) {
		return http.StatusOK, WhisperCheckResponse{Install: config.Install, Content: "未检测到可用的 Whisper", Status: 0}
	}
	return http.StatusOK, WhisperCheckResponse{Available: true, Status: 0}
}

func newWhisperTaskManager(cfg *Config, db *sql.DB) *whisperTaskManager {
	return &whisperTaskManager{cfg: cfg, db: db, wake: make(chan struct{}, 1)}
}

func startWhisperTaskManager(ctx context.Context, cfg *Config) {
	if cronDB == nil {
		log.Printf("[whisper] task queue unavailable: database is not ready")
		return
	}
	if err := ensureWhisperTaskSchema(cronDB); err != nil {
		log.Printf("[whisper] ensure task schema failed: %v", err)
		return
	}
	manager := newWhisperTaskManager(cfg, cronDB)
	if err := manager.recoverInterruptedTasks(); err != nil {
		log.Printf("[whisper] recover interrupted tasks failed: %v", err)
	}
	whisperTasks = manager
	go manager.run(ctx)
}

func (manager *whisperTaskManager) recoverInterruptedTasks() error {
	if manager == nil || manager.db == nil {
		return errors.New("task manager is not ready")
	}
	now := whisperTaskTimestamp()
	_, err := manager.db.Exec(`UPDATE whisper_task
		SET status = ?, progress = 0, started_at = '', updated_at = ?,
			logs = CASE WHEN length(logs) > ? THEN substr(logs, -?) ELSE logs END || ?
		WHERE status = ? AND cancel_requested = 0`,
		whisperTaskStatusQueued, now, whisperTaskLogLimit-2048, whisperTaskLogLimit/2,
		"\n[服务重启] 上次执行被中断，任务已恢复为排队中。\n", whisperTaskStatusRunning)
	if err != nil {
		return err
	}
	_, err = manager.db.Exec(`UPDATE whisper_task SET status = ?, updated_at = ? WHERE status = ? AND cancel_requested = 1`, whisperTaskStatusCancelled, now, whisperTaskStatusRunning)
	return err
}

func (manager *whisperTaskManager) signal() {
	if manager == nil {
		return
	}
	select {
	case manager.wake <- struct{}{}:
	default:
	}
}

func (manager *whisperTaskManager) run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			manager.cancelRunning()
			return
		}
		slotHeld, err := reserveModelTaskSlot(ctx, manager.cfg, manager.db, "whisper_task")
		if err != nil {
			if errors.Is(err, context.Canceled) {
				manager.cancelRunning()
				return
			}
			log.Printf("[whisper] reserve shared task slot failed: %v", err)
			select {
			case <-ctx.Done():
				manager.cancelRunning()
				return
			case <-time.After(time.Second):
			}
			continue
		}
		if !slotHeld {
			select {
			case <-ctx.Done():
				manager.cancelRunning()
				return
			case <-manager.wake:
			}
			continue
		}
		task, err := manager.claimNextTask()
		if err != nil {
			releaseModelTaskSlot(manager.cfg)
			log.Printf("[whisper] claim task failed: %v", err)
			select {
			case <-ctx.Done():
				manager.cancelRunning()
				return
			case <-time.After(time.Second):
			}
			continue
		}
		if task == nil {
			releaseModelTaskSlot(manager.cfg)
			select {
			case <-ctx.Done():
				manager.cancelRunning()
				return
			case <-manager.wake:
			}
			continue
		}
		manager.executeTask(ctx, *task)
		releaseModelTaskSlot(manager.cfg)
	}
}

func (manager *whisperTaskManager) claimNextTask() (*whisperTask, error) {
	if manager == nil || manager.db == nil {
		return nil, errors.New("task manager is not ready")
	}
	tx, err := manager.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	row := tx.QueryRow(`SELECT id, agent_id, source_path, output_path, saved_as, scenario, status, progress, started_at, created_at, updated_at, logs, cancel_requested
		FROM whisper_task WHERE status = ? AND cancel_requested = 0 ORDER BY id LIMIT 1`, whisperTaskStatusQueued)
	task, err := scanWhisperTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, tx.Commit()
	}
	if err != nil {
		return nil, err
	}
	now := whisperTaskTimestamp()
	result, err := tx.Exec(`UPDATE whisper_task SET status = ?, started_at = ?, updated_at = ?, progress = 0 WHERE id = ? AND status = ? AND cancel_requested = 0`, whisperTaskStatusRunning, now, now, task.ID, whisperTaskStatusQueued)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected != 1 {
		return nil, tx.Commit()
	}
	task.Status = whisperTaskStatusRunning
	task.StartedAt = now
	task.UpdatedAt = now
	task.Progress = 0
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &task, nil
}

func (manager *whisperTaskManager) executeTask(parent context.Context, task whisperTask) {
	ctx, cancel := context.WithCancel(parent)
	manager.mu.Lock()
	manager.runningID = task.ID
	manager.cancel = cancel
	manager.mu.Unlock()
	defer func() {
		cancel()
		manager.mu.Lock()
		if manager.runningID == task.ID {
			manager.runningID = 0
			manager.cancel = nil
		}
		manager.mu.Unlock()
	}()

	scenario, valid := whisperScenarioFor(task.Scenario)
	if !valid {
		manager.finishTask(task.ID, whisperTaskStatusFailed, 0, "", "提取失败：任务场景无效。")
		return
	}
	task.Scenario = scenario.ID
	manager.appendLog(task.ID, "开始使用 Whisper 转写。场景："+scenario.Name+"。")
	manager.appendLog(task.ID, "任务目的：从音频提取文字。")
	if err := manager.transcribe(ctx, task); err != nil {
		if manager.isCancelled(task.ID) || errors.Is(ctx.Err(), context.Canceled) {
			manager.finishTask(task.ID, whisperTaskStatusCancelled, 0, "", "任务已取消。")
		} else {
			manager.finishTask(task.ID, whisperTaskStatusFailed, 0, "", "任务失败，具体原因："+err.Error())
		}
		return
	}
	manager.finishTask(task.ID, whisperTaskStatusCompleted, 100, task.SavedAs, "文字提取完成。")
}

func (manager *whisperTaskManager) transcribe(ctx context.Context, task whisperTask) error {
	if manager == nil || manager.cfg == nil {
		return errors.New("服务配置不可用")
	}
	scenario, valid := whisperScenarioFor(task.Scenario)
	if !valid {
		return errors.New("任务场景无效")
	}
	whisperPath, available := whisperExecutablePath(time.Hour)
	if !available {
		return errors.New("未检测到 Whisper")
	}
	manager.appendLog(task.ID, taskPythonExecutionEnvironment(whisperPath))
	sourcePath, _, status, message := resolveMediaPreviewFile(manager.cfg, task.AgentID, task.SourcePath)
	if status != 0 {
		return errors.New(message)
	}
	mimeType, _ := mediaPreviewMIMEType(sourcePath)
	if !strings.HasPrefix(mimeType, "audio/") {
		return errors.New("仅支持音频文件")
	}
	workspace, err := getWorkspaceByAgentID(manager.cfg, task.AgentID)
	if err != nil {
		return errors.New("Agent 工作目录不存在")
	}
	targetPath, err := resolveCaseInsensitiveUnderRoot(workspace, filepath.FromSlash(task.OutputPath))
	if err != nil || !ensureWritablePathWithinRoot(workspace, targetPath) {
		return errors.New("无法创建文字输出路径")
	}
	if _, err := os.Lstat(targetPath); err == nil {
		outputPath, outputAbsolute, allocateErr := manager.allocateOutputPathExcept(task.AgentID, workspace, filepath.FromSlash(task.SourcePath), map[string]bool{}, task.ID)
		if allocateErr != nil {
			return allocateErr
		}
		if _, updateErr := manager.db.Exec(`UPDATE whisper_task SET output_path = ?, saved_as = ?, updated_at = ? WHERE id = ? AND status = ?`, filepath.ToSlash(outputPath), outputAbsolute, whisperTaskTimestamp(), task.ID, whisperTaskStatusRunning); updateErr != nil {
			return fmt.Errorf("无法更新文字输出路径: %w", updateErr)
		}
		task.OutputPath = filepath.ToSlash(outputPath)
		task.SavedAs = outputAbsolute
		targetPath = outputAbsolute
		manager.appendLog(task.ID, "同名文字文件已存在，已自动改为："+task.OutputPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.New("无法检查文字输出路径")
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return errors.New("无法创建文字输出目录")
	}
	manager.appendLog(task.ID, "原始音频绝对路径："+sourcePath)
	manager.appendLog(task.ID, "输出文字绝对路径："+targetPath)
	manager.appendLog(task.ID, "Whisper 可执行文件："+whisperPath)
	temporaryDirectory, err := os.MkdirTemp(filepath.Dir(targetPath), ".whisper-")
	if err != nil {
		return errors.New("无法创建 Whisper 临时目录")
	}
	defer os.RemoveAll(temporaryDirectory)

	device, deviceReason := "cpu", ""
	if scenario.ForceCPU {
		deviceReason = "场景“CPU轻量”指定使用 CPU。"
	} else {
		device, deviceReason = whisperTaskPreferredDevice(ctx, whisperPath)
	}
	if deviceReason != "" {
		manager.appendLog(task.ID, deviceReason)
	}
	modelPath, err := manager.prepareWhisperRuntimeModel(ctx, task.ID, scenario.Model)
	if err != nil {
		return err
	}
	scenario.Model = modelPath
	usedScenario, runErr := manager.runWhisperWithModelFallback(ctx, task.ID, whisperPath, sourcePath, temporaryDirectory, device, scenario)
	if runErr != nil {
		if ctx.Err() != nil {
			return context.Canceled
		}
		if device == "cuda" || device == "mps" {
			manager.appendLog(task.ID, "Whisper 无法使用 "+strings.ToUpper(device)+" 完成转写，已自动回退到 CPU 重新尝试。")
			if _, retryErr := manager.runWhisperWithModelFallback(ctx, task.ID, whisperPath, sourcePath, temporaryDirectory, "cpu", usedScenario); retryErr == nil {
				manager.appendLog(task.ID, "Whisper 已使用 CPU 回退完成。")
			} else {
				if ctx.Err() != nil {
					return context.Canceled
				}
				return retryErr
			}
		} else {
			return runErr
		}
	}
	temporaryOutput := filepath.Join(temporaryDirectory, strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))+".txt")
	info, err := os.Stat(temporaryOutput)
	if err != nil || info.IsDir() || info.Size() == 0 {
		return errors.New("Whisper 未生成有效的文字文件")
	}
	if err := os.Link(temporaryOutput, targetPath); err != nil {
		if errors.Is(err, os.ErrExist) {
			return errors.New("同名文字文件已存在")
		}
		return fmt.Errorf("保存文字文件失败: %w", err)
	}
	return nil
}

func (manager *whisperTaskManager) runWhisperWithModelFallback(ctx context.Context, taskID int64, whisperPath, sourcePath, temporaryDirectory, device string, scenario whisperScenario) (whisperScenario, error) {
	diagnostics, err := manager.runWhisperTranscription(ctx, taskID, whisperPath, sourcePath, temporaryDirectory, device, scenario)
	if err == nil {
		return scenario, nil
	}
	if whisperModelDownloadFailure(diagnostics) {
		manager.appendLog(taskID, "Whisper 原始模型下载失败，正在从 ModelScope 国内源下载已校验的 large-v3 模型。下载源："+whisperModelScopeLargeV3URL)
		modelPath, mirrorErr := manager.downloadWhisperLargeV3FromModelScope(ctx, taskID)
		if mirrorErr != nil {
			return scenario, fmt.Errorf("%w；ModelScope 国内源下载也失败：%v", err, mirrorErr)
		}
		mirrored := scenario
		mirrored.Model = modelPath
		manager.appendLog(taskID, "ModelScope Whisper large-v3 下载并校验成功，正在使用本地模型重新转写。")
		_, err = manager.runWhisperTranscription(ctx, taskID, whisperPath, sourcePath, temporaryDirectory, device, mirrored)
		return mirrored, err
	}
	if !whisperModelSelectionError(scenario, diagnostics) {
		return scenario, err
	}
	// large-v3 was introduced after the long-standing `large` checkpoint. A
	// stable older Whisper CLI can still complete the quality profile with that
	// model, whereas retrying the same unsupported name just fails again.
	legacy := scenario
	legacy.Model = "large"
	manager.appendLog(taskID, "当前 Whisper 不支持模型 large-v3，已回退到兼容的 large 模型重新尝试。")
	_, err = manager.runWhisperTranscription(ctx, taskID, whisperPath, sourcePath, temporaryDirectory, device, legacy)
	return legacy, err
}

func whisperModelDownloadFailure(diagnostics string) bool {
	value := strings.ToLower(diagnostics)
	for _, marker := range []string{
		"openaipublic.azureedge", "download the model", "downloaded but the sha256",
		"urllib.error", "urlopen error", "connection timed out", "read timed out",
		"network is unreachable", "temporary failure in name resolution", "http error",
	} {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func whisperModelSelectionError(scenario whisperScenario, diagnostics string) bool {
	if scenario.Model != "large-v3" {
		return false
	}
	value := strings.ToLower(diagnostics)
	return strings.Contains(value, "large-v3") && (strings.Contains(value, "not found") ||
		strings.Contains(value, "unknown model") ||
		strings.Contains(value, "available models") ||
		strings.Contains(value, "invalid choice"))
}

func (manager *whisperTaskManager) runWhisperTranscription(ctx context.Context, taskID int64, whisperPath, sourcePath, temporaryDirectory, device string, scenario whisperScenario) (string, error) {
	modelDirectory, err := whisperModelCacheDirectory()
	if err != nil {
		return "", err
	}
	args := []string{sourcePath, "--model", scenario.Model, "--output_dir", temporaryDirectory, "--output_format", "txt", "--device", device,
		"--language", scenario.Language, "--task", scenario.Task,
		"--temperature", strconv.FormatFloat(scenario.Temperature, 'f', -1, 64),
		"--beam_size", strconv.Itoa(scenario.BeamSize),
		"--condition_on_previous_text", whisperCLIBoolean(scenario.ConditionOnPreviousText),
	}
	if scenario.InitialPrompt != "" {
		args = append(args, "--initial_prompt", scenario.InitialPrompt)
	}
	if device == "cpu" {
		args = append(args, "--fp16", "False")
	}
	args = append(args, "--model_dir", modelDirectory)
	manager.appendLog(taskID, "本次转写参数："+whisperScenarioLogParameters(scenario, device)+"。")
	manager.appendLog(taskID, "实际 Whisper 参数："+strings.Join(args, " "))
	manager.appendLog(taskID, "Whisper 使用已校验的本地模型："+taskLogAbsolutePath(scenario.Model)+"。")
	command := whisperTaskCommandContext(ctx, whisperPath, args...)
	timeoutPath, cleanupTimeoutPath, err := whisperSocketTimeoutSitePackage()
	if err != nil {
		return "", err
	}
	defer cleanupTimeoutPath()
	pythonPath := timeoutPath
	if inherited := strings.TrimSpace(os.Getenv("PYTHONPATH")); inherited != "" {
		pythonPath += string(os.PathListSeparator) + inherited
	}
	command.Env = append(os.Environ(), "PYTHONPATH="+pythonPath)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("无法读取 Whisper 输出: %w", err)
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("无法读取 Whisper 日志: %w", err)
	}
	if err := command.Start(); err != nil {
		return "", fmt.Errorf("无法启动 Whisper: %w", err)
	}
	var diagnostics strings.Builder
	var diagnosticsMu sync.Mutex
	var output sync.WaitGroup
	output.Add(2)
	go func() {
		defer output.Done()
		manager.captureWhisperOutput(taskID, stdout, &diagnostics, &diagnosticsMu)
	}()
	go func() {
		defer output.Done()
		manager.captureWhisperOutput(taskID, stderr, &diagnostics, &diagnosticsMu)
	}()
	// StdoutPipe/StderrPipe readers must finish before Wait. Wait closes these
	// pipes itself, which otherwise races with Scanner and produces a misleading
	// "file already closed" message after a successful transcription.
	output.Wait()
	err = command.Wait()
	if err != nil {
		if ctx.Err() != nil {
			return diagnostics.String(), context.Canceled
		}
		return diagnostics.String(), taskExecutionError("Whisper", err, diagnostics.String())
	}
	return diagnostics.String(), nil
}

func whisperModelCacheDirectory() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("DEEPRIGHT_WHISPER_MODEL_CACHE")); configured != "" {
		if err := os.MkdirAll(configured, 0o755); err != nil {
			return "", fmt.Errorf("无法创建 Whisper 模型缓存目录: %w", err)
		}
		return configured, nil
	}
	cache, err := whisperModelUserCacheDir()
	if err != nil {
		return "", fmt.Errorf("无法确定 Whisper 模型缓存目录: %w", err)
	}
	directory := filepath.Join(cache, "deepright", "whisper")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", fmt.Errorf("无法创建 Whisper 模型缓存目录: %w", err)
	}
	return directory, nil
}

// openai-whisper downloads through urllib rather than a configurable Hub
// client. sitecustomize gives every socket in just this child process a
// bounded idle timeout without altering the user's Python installation.
func whisperSocketTimeoutSitePackage() (string, func(), error) {
	directory, err := os.MkdirTemp("", "deepright-whisper-timeout-")
	if err != nil {
		return "", nil, fmt.Errorf("无法创建 Whisper 下载超时配置: %w", err)
	}
	content := "import socket\nsocket.setdefaulttimeout(90)\n"
	if err := os.WriteFile(filepath.Join(directory, "sitecustomize.py"), []byte(content), 0o600); err != nil {
		_ = os.RemoveAll(directory)
		return "", nil, fmt.Errorf("无法写入 Whisper 下载超时配置: %w", err)
	}
	return directory, func() { _ = os.RemoveAll(directory) }, nil
}

// prepareWhisperRuntimeModel prevents the Whisper CLI from starting an
// unmanaged download. All configured task models are cached through the same
// resumable downloader used by the other media tasks, then passed to Whisper
// as an absolute local checkpoint path.
func (manager *whisperTaskManager) prepareWhisperRuntimeModel(ctx context.Context, taskID int64, name string) (string, error) {
	artifact, ok := whisperRuntimeModelArtifacts[name]
	if !ok {
		return "", errors.New("不支持的 Whisper 运行时模型：" + name)
	}
	whisperModelMirrorMu.Lock()
	defer whisperModelMirrorMu.Unlock()
	directory, err := whisperModelCacheDirectory()
	if err != nil {
		return "", err
	}
	target := filepath.Join(directory, name+".pt")
	manager.appendLog(taskID, "Whisper 模型缓存目录："+taskLogAbsolutePath(directory))
	manager.appendLog(taskID, "Whisper 模型下载目标绝对路径："+taskLogAbsolutePath(target))
	if whisperModelFileMatchesHash(target, artifact.SHA256) {
		manager.appendLog(taskID, "检测到已校验的 Whisper "+name+" 本地缓存，跳过重复下载。")
		return target, nil
	}
	manager.appendLog(taskID, "Whisper "+name+" 将使用多线程分段下载并保留断点；任务取消或网络中断后重新开始会继续未完成分段。")
	temporaryPath := target + ".deepright.part"
	lastProgress := int64(-1)
	_, err = downloadModelWithFallback(ctx, resumableModelDownloadConfig{
		Client:      whisperModelHTTPClient,
		URL:         artifact.OfficialURL,
		BackupURL:   artifact.BackupURL,
		PartPath:    temporaryPath,
		IdleTimeout: modelTaskDownloadReadTimeout(manager.cfg),
		Workers:     modelTaskDownloadWorkers(manager.cfg),
		Retries:     modelTaskDownloadRetries(manager.cfg),
		Progress: func(copied, total int64) {
			if total <= 0 {
				return
			}
			progress := copied * 100 / total
			if progress >= 100 || progress/10 > lastProgress/10 {
				lastProgress = progress
				manager.appendLog(taskID, fmt.Sprintf("Whisper %s 已下载 %d%%。", name, progress))
			}
		},
		PartProgress: func(worker, part, parts int, copied, total int64) {
			if total <= 0 {
				return
			}
			manager.appendLog(taskID, fmt.Sprintf("Whisper %s 下载线程 %d，分段 %d/%d 已下载 %d%%。", name, worker, part, parts, copied*100/total))
		},
		Retry: func(worker, part, parts, retry, retries int, err error) {
			manager.appendLog(taskID, "Whisper "+name+" "+modelDownloadRetryMessage(worker, part, parts, retry, retries, err))
		},
	}, func(message string) { manager.appendLog(taskID, "Whisper "+name+"："+message) })
	if err != nil {
		return "", fmt.Errorf("下载 Whisper %s 模型失败: %w", name, err)
	}
	if !whisperModelFileMatchesHash(temporaryPath, artifact.SHA256) {
		_ = os.Remove(temporaryPath)
		removeModelDownloadState(temporaryPath)
		return "", errors.New("Whisper " + name + " 模型 SHA-256 校验失败")
	}
	if err := os.Rename(temporaryPath, target); err != nil {
		return "", fmt.Errorf("写入 Whisper 本地模型失败: %w", err)
	}
	removeModelDownloadState(temporaryPath)
	return target, nil
}

func (manager *whisperTaskManager) downloadWhisperLargeV3FromModelScope(ctx context.Context, taskID int64) (string, error) {
	whisperModelMirrorMu.Lock()
	defer whisperModelMirrorMu.Unlock()
	directory, err := whisperModelCacheDirectory()
	if err != nil {
		return "", err
	}
	target := filepath.Join(directory, "large-v3.pt")
	manager.appendLog(taskID, "ModelScope Whisper large-v3 下载目标绝对路径："+taskLogAbsolutePath(target))
	if whisperModelFileMatches(target) {
		manager.appendLog(taskID, "检测到已校验的 ModelScope Whisper large-v3 本地缓存，跳过重复下载。")
		return target, nil
	}
	temporaryPath := target + ".deepright.part"
	manager.appendLog(taskID, "ModelScope Whisper large-v3 将使用多线程分段下载并保留断点；任务取消或网络中断后重新开始会继续未完成分段。")
	lastProgress := int64(-1)
	_, err = downloadModelWithFallback(ctx, resumableModelDownloadConfig{
		Client:       whisperModelHTTPClient,
		URL:          whisperModelScopeLargeV3URL,
		BackupURL:    whisperOfficialLargeV3URL,
		PartPath:     temporaryPath,
		ExpectedSize: whisperLargeV3Size,
		IdleTimeout:  modelTaskDownloadReadTimeout(manager.cfg),
		Workers:      modelTaskDownloadWorkers(manager.cfg),
		Retries:      modelTaskDownloadRetries(manager.cfg),
		Progress: func(copied, total int64) {
			if total <= 0 {
				return
			}
			progress := copied * 100 / total
			if progress >= 100 || progress/10 > lastProgress/10 {
				lastProgress = progress
				manager.appendLog(taskID, fmt.Sprintf("ModelScope Whisper large-v3 已下载 %d%%。", progress))
			}
		},
		PartProgress: func(worker, part, parts int, copied, total int64) {
			if total <= 0 {
				return
			}
			manager.appendLog(taskID, fmt.Sprintf("ModelScope Whisper large-v3 下载线程 %d，分段 %d/%d 已下载 %d%%。", worker, part, parts, copied*100/total))
		},
		Retry: func(worker, part, parts, retry, retries int, err error) {
			manager.appendLog(taskID, "ModelScope Whisper large-v3 "+modelDownloadRetryMessage(worker, part, parts, retry, retries, err))
		},
	}, func(message string) { manager.appendLog(taskID, message) })
	if err != nil {
		return "", fmt.Errorf("下载 ModelScope Whisper 模型失败: %w", err)
	}
	file, err := os.Open(temporaryPath)
	if err != nil {
		return "", fmt.Errorf("无法读取已下载 Whisper 模型: %w", err)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("校验 Whisper 模型失败: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("关闭 Whisper 模型文件失败: %w", err)
	}
	if actual := fmt.Sprintf("%x", hash.Sum(nil)); actual != whisperLargeV3SHA256 {
		_ = os.Remove(temporaryPath)
		removeModelDownloadState(temporaryPath)
		return "", errors.New("ModelScope Whisper 模型 SHA-256 校验失败")
	}
	if err := os.Rename(temporaryPath, target); err != nil {
		return "", fmt.Errorf("写入 Whisper 本地模型失败: %w", err)
	}
	removeModelDownloadState(temporaryPath)
	return target, nil
}

func whisperModelFileMatches(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Size() != whisperLargeV3Size {
		return false
	}
	return whisperModelFileMatchesHash(path, whisperLargeV3SHA256)
}

func whisperModelFileMatchesHash(path, expected string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false
	}
	return fmt.Sprintf("%x", hash.Sum(nil)) == expected
}

func whisperPreferredDevice(ctx context.Context, whisperPath string) (string, string) {
	interpreter := whisperPythonInterpreter(whisperPath)
	if interpreter == "" {
		return "cpu", "未能验证 Whisper 的 GPU 环境，已使用 CPU。"
	}
	probeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	command := whisperTaskCommandContext(probeCtx, interpreter, "-c", "import torch; print('cuda' if torch.cuda.is_available() else ('mps' if getattr(torch.backends, 'mps', None) and torch.backends.mps.is_available() else 'cpu'))")
	result, err := command.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(stripWhisperANSI(string(result)))
		if detail != "" {
			detail = strings.ReplaceAll(detail, "\n", " ")
			if len(detail) > 240 {
				detail = detail[:240] + "…"
			}
			return "cpu", "Whisper GPU 探测失败（" + filepath.Base(interpreter) + "）：" + detail + "；已使用 CPU。"
		}
		return "cpu", "Whisper GPU 探测失败（" + filepath.Base(interpreter) + "）：" + err.Error() + "；已使用 CPU。"
	}
	switch strings.TrimSpace(string(result)) {
	case "cuda":
		return "cuda", "检测到 CUDA GPU，优先使用 GPU。"
	case "mps":
		return "mps", "检测到 Apple MPS GPU，优先使用 GPU。"
	default:
		return "cpu", "未检测到 Whisper 可用 GPU，已使用 CPU。"
	}
}

func whisperPythonInterpreter(whisperPath string) string {
	data, err := os.ReadFile(whisperPath)
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
		resolved, err := whisperTaskLookPath(candidate)
		if err != nil {
			return ""
		}
		return resolved
	}
	return parts[0]
}

func (manager *whisperTaskManager) captureWhisperOutput(taskID int64, reader io.Reader, diagnostics *strings.Builder, diagnosticsMu *sync.Mutex) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 128*1024)
	scanner.Split(scanWhisperOutput)
	for scanner.Scan() {
		line := strings.TrimSpace(stripWhisperANSI(scanner.Text()))
		if line == "" {
			continue
		}
		diagnosticsMu.Lock()
		if diagnostics.Len() < 128*1024 {
			diagnostics.WriteString(line + "\n")
		}
		diagnosticsMu.Unlock()
		manager.appendLog(taskID, line)
		if matches := whisperProgressPattern.FindStringSubmatch(line); len(matches) == 2 {
			if progress, err := strconv.Atoi(matches[1]); err == nil && progress >= 0 && progress <= 100 {
				manager.updateProgress(taskID, progress)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		manager.appendLog(taskID, "读取 Whisper 输出失败："+err.Error())
	}
}

func scanWhisperOutput(data []byte, atEOF bool) (advance int, token []byte, err error) {
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

var whisperANSIPattern = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)

func stripWhisperANSI(value string) string {
	return whisperANSIPattern.ReplaceAllString(value, "")
}

func (manager *whisperTaskManager) updateProgress(taskID int64, progress int) {
	if manager == nil || manager.db == nil {
		return
	}
	_, _ = manager.db.Exec(`UPDATE whisper_task SET progress = CASE WHEN progress > ? THEN progress ELSE ? END, updated_at = ? WHERE id = ? AND status = ?`, progress, progress, whisperTaskTimestamp(), taskID, whisperTaskStatusRunning)
}

func (manager *whisperTaskManager) appendLog(taskID int64, message string) {
	if manager == nil || manager.db == nil {
		return
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	if len(message) > 4096 {
		message = message[:4096] + "…"
	}
	entry := "[" + whisperTaskTimestamp() + "] " + message + "\n"
	_, _ = manager.db.Exec(`UPDATE whisper_task SET logs = CASE WHEN length(logs) > ? THEN substr(logs, -?) ELSE logs END || ?, updated_at = ? WHERE id = ?`, whisperTaskLogLimit-8192, whisperTaskLogLimit/2, entry, whisperTaskTimestamp(), taskID)
}

func (manager *whisperTaskManager) finishTask(taskID int64, status string, progress int, savedAs, message string) {
	manager.appendLog(taskID, message)
	if manager == nil || manager.db == nil {
		return
	}
	// saved_as is recorded when a task is created or its collision-safe output
	// path is reallocated. Do not overwrite it here with a stale task snapshot.
	_ = savedAs
	_, _ = manager.db.Exec(`UPDATE whisper_task SET status = ?, progress = ?, updated_at = ? WHERE id = ?`, status, progress, whisperTaskTimestamp(), taskID)
}

func (manager *whisperTaskManager) isCancelled(taskID int64) bool {
	if manager == nil || manager.db == nil {
		return true
	}
	var cancelRequested int
	err := manager.db.QueryRow(`SELECT cancel_requested FROM whisper_task WHERE id = ?`, taskID).Scan(&cancelRequested)
	return err != nil || cancelRequested != 0
}

func (manager *whisperTaskManager) cancelRunning() {
	if manager == nil {
		return
	}
	manager.mu.Lock()
	cancel := manager.cancel
	manager.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (manager *whisperTaskManager) createTasks(request whisperTaskCreateRequest) ([]whisperTask, error) {
	if manager == nil || manager.cfg == nil || manager.db == nil {
		return nil, errors.New("Whisper 任务服务未就绪")
	}
	request.AgentID = strings.TrimSpace(request.AgentID)
	items := request.Tasks
	if len(items) == 0 {
		items = make([]whisperTaskCreateItem, 0, len(request.Paths))
		for _, path := range request.Paths {
			items = append(items, whisperTaskCreateItem{Path: path, Scenario: request.Scenario})
		}
	}
	if !isValidMediaPreviewAgentID(request.AgentID) || len(items) == 0 || len(items) > 64 {
		return nil, errors.New("agentId 和 1 至 64 个音频文件不能为空")
	}
	manager.createMu.Lock()
	defer manager.createMu.Unlock()
	workspace, err := getWorkspaceByAgentID(manager.cfg, request.AgentID)
	if err != nil {
		return nil, errors.New("Agent 不存在")
	}
	type source struct{ relative, output, absolute, scenario string }
	sources := make([]source, 0, len(items))
	seen := make(map[string]bool)
	reservedOutputs := make(map[string]bool)
	for _, item := range items {
		relative := normalizeQuotedPathArg(item.Path)
		if seen[relative] {
			continue
		}
		seen[relative] = true
		scenario, valid := whisperScenarioFor(item.Scenario)
		if !valid {
			return nil, errors.New("不支持的 Whisper 转写场景")
		}
		absolute, _, status, message := resolveMediaPreviewFile(manager.cfg, request.AgentID, relative)
		if status != 0 {
			return nil, errors.New(message)
		}
		mimeType, _ := mediaPreviewMIMEType(absolute)
		if !strings.HasPrefix(mimeType, "audio/") {
			return nil, errors.New("仅支持音频文件")
		}
		relativePath, err := filepath.Rel(workspace, absolute)
		if err != nil || relativePath == "." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
			return nil, errors.New("音频文件必须位于当前 Agent 工作目录")
		}
		outputRelative, outputAbsolute, err := manager.allocateOutputPath(request.AgentID, workspace, relativePath, reservedOutputs)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source{relative: filepath.ToSlash(relativePath), output: filepath.ToSlash(outputRelative), absolute: outputAbsolute, scenario: scenario.ID})
	}
	if len(sources) == 0 {
		return nil, errors.New("未找到可添加的音频文件")
	}
	tx, err := manager.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	created := make([]whisperTask, 0, len(sources))
	now := whisperTaskTimestamp()
	for _, source := range sources {
		scenario, _ := whisperScenarioFor(source.scenario)
		result, err := tx.Exec(`INSERT INTO whisper_task (agent_id, source_path, output_path, saved_as, scenario, status, progress, created_at, updated_at, logs, cancel_requested) VALUES (?,?,?,?,?,?,?,?,?,?,0)`, request.AgentID, source.relative, source.output, source.absolute, source.scenario, whisperTaskStatusQueued, 0, now, now, "["+now+"] 已加入 Whisper 转写队列。场景："+scenario.Name+"。\n")
		if err != nil {
			return nil, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}
		created = append(created, whisperTask{ID: id, AgentID: request.AgentID, SourcePath: source.relative, OutputPath: source.output, SavedAs: source.absolute, Scenario: source.scenario, Status: whisperTaskStatusQueued, CreatedAt: now, UpdatedAt: now})
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	manager.signal()
	return created, nil
}

func (manager *whisperTaskManager) allocateOutputPath(agentID, workspace, sourceRelative string, reserved map[string]bool) (string, string, error) {
	return manager.allocateOutputPathExcept(agentID, workspace, sourceRelative, reserved, 0)
}

func (manager *whisperTaskManager) allocateOutputPathExcept(agentID, workspace, sourceRelative string, reserved map[string]bool, exceptTaskID int64) (string, string, error) {
	// Keep every transcription in the Agent workspace's dedicated whisper
	// directory. The source path only contributes the file stem, never the
	// destination directory.
	directory := "whisper"
	stem := strings.TrimSuffix(filepath.Base(sourceRelative), filepath.Ext(sourceRelative))
	stamp := whisperTaskNow().Format("20060102_150405")
	for attempt := 0; attempt < 1000; attempt++ {
		name := stem + ".txt"
		if attempt > 0 {
			suffix := "_" + stamp
			if attempt > 1 {
				suffix += "_" + strconv.Itoa(attempt)
			}
			name = stem + suffix + ".txt"
		}
		relative := filepath.Join(directory, name)
		key := filepath.ToSlash(relative)
		if reserved[key] {
			continue
		}
		absolute, err := resolveCaseInsensitiveUnderRoot(workspace, relative)
		if err != nil || !ensureWritablePathWithinRoot(workspace, absolute) {
			return "", "", errors.New("无法创建文字输出路径")
		}
		if _, err := os.Lstat(absolute); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", "", errors.New("无法检查文字输出路径")
		}
		var existing int
		if err := manager.db.QueryRow(`SELECT COUNT(*) FROM whisper_task WHERE agent_id = ? AND output_path = ? AND id <> ?`, agentID, key, exceptTaskID).Scan(&existing); err != nil {
			return "", "", err
		}
		if existing != 0 {
			continue
		}
		reserved[key] = true
		return relative, absolute, nil
	}
	return "", "", errors.New("无法生成唯一的文字输出路径")
}

func normalizeWhisperTaskStatusFilter(value string) (string, error) {
	status := strings.TrimSpace(strings.ToLower(value))
	switch status {
	case "", "all":
		return "", nil
	case whisperTaskStatusQueued, whisperTaskStatusRunning, whisperTaskStatusCompleted, whisperTaskStatusCancelled, whisperTaskStatusFailed:
		return status, nil
	default:
		return "", errors.New("任务状态筛选无效")
	}
}

func (manager *whisperTaskManager) listTasksPage(agentID, status string, page int, includeLogs bool) (whisperTaskPage, error) {
	result := whisperTaskPage{Tasks: []whisperTask{}, PageSize: 5}
	if manager == nil || manager.db == nil {
		return result, errors.New("Whisper 任务服务未就绪")
	}
	agentID = strings.TrimSpace(agentID)
	if !isValidMediaPreviewAgentID(agentID) {
		return result, errors.New("agentId 无效")
	}
	status, err := normalizeWhisperTaskStatusFilter(status)
	if err != nil {
		return result, err
	}
	if page < 1 {
		page = 1
	}
	result.Page = page
	result.Status = status
	logs := "''"
	if includeLogs {
		logs = "logs"
	}
	where := "agent_id = ?"
	args := []interface{}{agentID}
	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}
	if err := manager.db.QueryRow(`SELECT COUNT(*) FROM whisper_task WHERE `+where, args...).Scan(&result.Total); err != nil {
		return result, err
	}
	if lastPage := (result.Total + result.PageSize - 1) / result.PageSize; lastPage > 0 && result.Page > lastPage {
		result.Page = lastPage
	}
	args = append(args, result.PageSize, (result.Page-1)*result.PageSize)
	rows, err := manager.db.Query(`SELECT id, agent_id, source_path, output_path, saved_as, scenario, status, progress, started_at, created_at, updated_at, `+logs+`, cancel_requested FROM whisper_task WHERE `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		task, err := scanWhisperTask(rows)
		if err != nil {
			return result, err
		}
		result.Tasks = append(result.Tasks, task)
	}
	return result, rows.Err()
}

func (manager *whisperTaskManager) taskLog(agentID string, taskID int64) (whisperTask, error) {
	if manager == nil || manager.db == nil || !isValidMediaPreviewAgentID(agentID) || taskID <= 0 {
		return whisperTask{}, errors.New("任务参数无效")
	}
	return scanWhisperTask(manager.db.QueryRow(`SELECT id, agent_id, source_path, output_path, saved_as, scenario, status, progress, started_at, created_at, updated_at, logs, cancel_requested FROM whisper_task WHERE id = ? AND agent_id = ?`, taskID, strings.TrimSpace(agentID)))
}

func (manager *whisperTaskManager) cancelTask(agentID string, taskID int64) error {
	if manager == nil || manager.db == nil || !isValidMediaPreviewAgentID(agentID) || taskID <= 0 {
		return errors.New("任务参数无效")
	}
	now := whisperTaskTimestamp()
	result, err := manager.db.Exec(`UPDATE whisper_task SET cancel_requested = 1, status = CASE WHEN status = ? THEN ? ELSE status END, updated_at = ?, logs = CASE WHEN length(logs) > ? THEN substr(logs, -?) ELSE logs END || ? WHERE id = ? AND agent_id = ? AND status IN (?, ?)`, whisperTaskStatusQueued, whisperTaskStatusCancelled, now, whisperTaskLogLimit-4096, whisperTaskLogLimit/2, "["+now+"] 已请求取消任务。\n", taskID, agentID, whisperTaskStatusQueued, whisperTaskStatusRunning)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return errors.New("未找到可取消的任务")
	}
	manager.mu.Lock()
	cancel := manager.cancel
	running := manager.runningID == taskID
	manager.mu.Unlock()
	if running && cancel != nil {
		cancel()
	}
	manager.signal()
	return nil
}

func (manager *whisperTaskManager) restartTask(agentID string, taskID int64) error {
	if manager == nil || manager.cfg == nil || manager.db == nil || !isValidMediaPreviewAgentID(agentID) || taskID <= 0 {
		return errors.New("任务参数无效")
	}
	agentID = strings.TrimSpace(agentID)
	manager.createMu.Lock()
	defer manager.createMu.Unlock()
	var sourcePath string
	if err := manager.db.QueryRow(`SELECT source_path FROM whisper_task WHERE id = ? AND agent_id = ? AND status IN (?, ?)`, taskID, agentID, whisperTaskStatusFailed, whisperTaskStatusCancelled).Scan(&sourcePath); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("未找到可重新开始的失败或已取消任务")
		}
		return err
	}
	fail := func(err error) error {
		manager.appendLog(taskID, "重新开始任务失败："+err.Error())
		return err
	}
	workspace, err := getWorkspaceByAgentID(manager.cfg, agentID)
	if err != nil {
		return fail(errors.New("Agent 不存在"))
	}
	outputPath, savedAs, err := manager.allocateOutputPath(agentID, workspace, filepath.FromSlash(sourcePath), map[string]bool{})
	if err != nil {
		return fail(err)
	}
	now := whisperTaskTimestamp()
	result, err := manager.db.Exec(`UPDATE whisper_task SET output_path = ?, saved_as = ?, status = ?, progress = 0, started_at = '', cancel_requested = 0, updated_at = ?, logs = '' WHERE id = ? AND agent_id = ? AND status IN (?, ?)`, filepath.ToSlash(outputPath), savedAs, whisperTaskStatusQueued, now, taskID, agentID, whisperTaskStatusFailed, whisperTaskStatusCancelled)
	if err != nil {
		return fail(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fail(err)
	}
	if affected != 1 {
		return fail(errors.New("未找到可重新开始的失败或已取消任务"))
	}
	manager.signal()
	return nil
}

func (manager *whisperTaskManager) deleteFailedOrCancelledTask(agentID string, taskID int64) error {
	if manager == nil || manager.db == nil || !isValidMediaPreviewAgentID(agentID) || taskID <= 0 {
		return errors.New("任务参数无效")
	}
	result, err := manager.db.Exec(`DELETE FROM whisper_task WHERE id = ? AND agent_id = ? AND status IN (?, ?)`, taskID, strings.TrimSpace(agentID), whisperTaskStatusFailed, whisperTaskStatusCancelled)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return errors.New("仅可删除失败或已取消任务")
	}
	return nil
}

type whisperTaskScanner interface {
	Scan(dest ...interface{}) error
}

func scanWhisperTask(scanner whisperTaskScanner) (whisperTask, error) {
	var task whisperTask
	var cancelRequested int
	err := scanner.Scan(&task.ID, &task.AgentID, &task.SourcePath, &task.OutputPath, &task.SavedAs, &task.Scenario, &task.Status, &task.Progress, &task.StartedAt, &task.CreatedAt, &task.UpdatedAt, &task.Logs, &cancelRequested)
	task.CancelAsked = cancelRequested != 0
	return task, err
}

func writeWhisperTaskJSON(w http.ResponseWriter, httpStatus int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(payload)
}

func handleWhisperTasks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if whisperTasks == nil {
			writeWhisperTaskJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": 1, "content": "Whisper 任务服务未就绪"})
			return
		}
		switch r.Method {
		case http.MethodGet:
			page := 1
			if raw := strings.TrimSpace(r.URL.Query().Get("page")); raw != "" {
				parsed, parseErr := strconv.Atoi(raw)
				if parseErr != nil || parsed < 1 {
					writeWhisperTaskJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "页码必须是正整数"})
					return
				}
				page = parsed
			}
			result, err := whisperTasks.listTasksPage(r.URL.Query().Get("agentId"), r.URL.Query().Get("status"), page, false)
			if err != nil {
				writeWhisperTaskJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
				return
			}
			writeWhisperTaskJSON(w, http.StatusOK, map[string]interface{}{"status": 0, "tasks": result.Tasks, "total": result.Total, "page": result.Page, "pageSize": result.PageSize, "taskStatus": result.Status})
		case http.MethodPost:
			r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
			defer r.Body.Close()
			var request whisperTaskCreateRequest
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
				writeWhisperTaskJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "请求体格式错误"})
				return
			}
			tasks, err := whisperTasks.createTasks(request)
			if err != nil {
				writeWhisperTaskJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
				return
			}
			writeWhisperTaskJSON(w, http.StatusOK, map[string]interface{}{"status": 0, "tasks": tasks})
		default:
			w.Header().Set("Allow", "GET, POST")
			writeWhisperTaskJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"status": 1, "content": "仅支持 GET 或 POST 请求"})
		}
	}
}

func handleWhisperTaskCancel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeWhisperTaskJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"status": 1, "content": "仅支持 POST 请求"})
			return
		}
		if whisperTasks == nil {
			writeWhisperTaskJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": 1, "content": "Whisper 任务服务未就绪"})
			return
		}
		var request whisperTaskCancelRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
			writeWhisperTaskJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "请求体格式错误"})
			return
		}
		if err := whisperTasks.cancelTask(strings.TrimSpace(request.AgentID), request.ID); err != nil {
			writeWhisperTaskJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		writeWhisperTaskJSON(w, http.StatusOK, map[string]interface{}{"status": 0})
	}
}

func handleWhisperTaskRestart() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeWhisperTaskJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"status": 1, "content": "仅支持 POST 请求"})
			return
		}
		if whisperTasks == nil {
			writeWhisperTaskJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": 1, "content": "Whisper 任务服务未就绪"})
			return
		}
		var request whisperTaskRestartRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
			writeWhisperTaskJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "请求体格式错误"})
			return
		}
		if err := whisperTasks.restartTask(strings.TrimSpace(request.AgentID), request.ID); err != nil {
			writeWhisperTaskJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		writeWhisperTaskJSON(w, http.StatusOK, map[string]interface{}{"status": 0})
	}
}

func handleWhisperTaskDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeWhisperTaskJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"status": 1, "content": "仅支持 POST 请求"})
			return
		}
		if whisperTasks == nil {
			writeWhisperTaskJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": 1, "content": "Whisper 任务服务未就绪"})
			return
		}
		var request whisperTaskDeleteRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
			writeWhisperTaskJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "请求体格式错误"})
			return
		}
		if err := whisperTasks.deleteFailedOrCancelledTask(strings.TrimSpace(request.AgentID), request.ID); err != nil {
			writeWhisperTaskJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		writeWhisperTaskJSON(w, http.StatusOK, map[string]interface{}{"status": 0})
	}
}

func handleWhisperTaskLog() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeWhisperTaskJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"status": 1, "content": "仅支持 GET 请求"})
			return
		}
		if whisperTasks == nil {
			writeWhisperTaskJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": 1, "content": "Whisper 任务服务未就绪"})
			return
		}
		id, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("id")), 10, 64)
		if err != nil || id <= 0 {
			writeWhisperTaskJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "任务 ID 无效"})
			return
		}
		task, err := whisperTasks.taskLog(r.URL.Query().Get("agentId"), id)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, sql.ErrNoRows) {
				status = http.StatusNotFound
			}
			writeWhisperTaskJSON(w, status, map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		writeWhisperTaskJSON(w, http.StatusOK, map[string]interface{}{"status": 0, "task": task})
	}
}
