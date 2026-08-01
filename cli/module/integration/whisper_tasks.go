package main

import (
	"bufio"
	"context"
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
		raw, _, err := readIntegrationStartupConfigRaw()
		if err != nil {
			writeWhisperCheckResp(w, http.StatusInternalServerError, WhisperCheckResponse{Content: "读取 config/config.json 失败: " + err.Error(), Status: 1})
			return
		}
		config, err := parseWhisperCheckConfig(raw)
		if err != nil {
			writeWhisperCheckResp(w, http.StatusBadRequest, WhisperCheckResponse{Content: err.Error(), Status: 1})
			return
		}
		if !whisperDependencyAvailable(config.CacheFor) {
			writeWhisperCheckResp(w, http.StatusOK, WhisperCheckResponse{Install: config.Install, Content: "未检测到可用的 Whisper", Status: 0})
			return
		}
		writeWhisperCheckResp(w, http.StatusOK, WhisperCheckResponse{Available: true, Status: 0})
	}
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
		task, err := manager.claimNextTask()
		if err != nil {
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
			select {
			case <-ctx.Done():
				manager.cancelRunning()
				return
			case <-manager.wake:
			}
			continue
		}
		manager.executeTask(ctx, *task)
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
	if err := manager.transcribe(ctx, task); err != nil {
		if manager.isCancelled(task.ID) || errors.Is(ctx.Err(), context.Canceled) {
			manager.finishTask(task.ID, whisperTaskStatusCancelled, 0, "", "任务已取消。")
		} else {
			manager.finishTask(task.ID, whisperTaskStatusFailed, 0, "", "提取失败："+err.Error())
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
	if err := manager.runWhisperTranscription(ctx, task.ID, whisperPath, sourcePath, temporaryDirectory, device, scenario); err != nil {
		if ctx.Err() != nil {
			return context.Canceled
		}
		if device == "cuda" || device == "mps" {
			manager.appendLog(task.ID, "Whisper 无法使用 "+strings.ToUpper(device)+" 完成转写，已自动回退到 CPU 重新尝试。")
			if retryErr := manager.runWhisperTranscription(ctx, task.ID, whisperPath, sourcePath, temporaryDirectory, "cpu", scenario); retryErr == nil {
				manager.appendLog(task.ID, "Whisper 已使用 CPU 回退完成。")
			} else {
				if ctx.Err() != nil {
					return context.Canceled
				}
				return retryErr
			}
		} else {
			return err
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

func (manager *whisperTaskManager) runWhisperTranscription(ctx context.Context, taskID int64, whisperPath, sourcePath, temporaryDirectory, device string, scenario whisperScenario) error {
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
	manager.appendLog(taskID, "本次转写参数："+whisperScenarioLogParameters(scenario, device)+"。")
	command := whisperTaskCommandContext(ctx, whisperPath, args...)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return fmt.Errorf("无法读取 Whisper 输出: %w", err)
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return fmt.Errorf("无法读取 Whisper 日志: %w", err)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("无法启动 Whisper: %w", err)
	}
	var output sync.WaitGroup
	output.Add(2)
	go func() { defer output.Done(); manager.captureWhisperOutput(taskID, stdout) }()
	go func() { defer output.Done(); manager.captureWhisperOutput(taskID, stderr) }()
	// StdoutPipe/StderrPipe readers must finish before Wait. Wait closes these
	// pipes itself, which otherwise races with Scanner and produces a misleading
	// "file already closed" message after a successful transcription.
	output.Wait()
	err = command.Wait()
	if err != nil {
		if ctx.Err() != nil {
			return context.Canceled
		}
		return fmt.Errorf("Whisper 执行失败: %w", err)
	}
	return nil
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

func (manager *whisperTaskManager) captureWhisperOutput(taskID int64, reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 128*1024)
	scanner.Split(scanWhisperOutput)
	for scanner.Scan() {
		line := strings.TrimSpace(stripWhisperANSI(scanner.Text()))
		if line == "" {
			continue
		}
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
	if err := manager.db.QueryRow(`SELECT source_path FROM whisper_task WHERE id = ? AND agent_id = ? AND status = ?`, taskID, agentID, whisperTaskStatusCancelled).Scan(&sourcePath); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("未找到可重新开始的已取消任务")
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
	result, err := manager.db.Exec(`UPDATE whisper_task SET output_path = ?, saved_as = ?, status = ?, progress = 0, started_at = '', cancel_requested = 0, updated_at = ?, logs = CASE WHEN length(logs) > ? THEN substr(logs, -?) ELSE logs END || ? WHERE id = ? AND agent_id = ? AND status = ?`, filepath.ToSlash(outputPath), savedAs, whisperTaskStatusQueued, now, whisperTaskLogLimit-4096, whisperTaskLogLimit/2, "["+now+"] 已重新加入 Whisper 转写队列。\n", taskID, agentID, whisperTaskStatusCancelled)
	if err != nil {
		return fail(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fail(err)
	}
	if affected != 1 {
		return fail(errors.New("未找到可重新开始的已取消任务"))
	}
	manager.signal()
	return nil
}

func (manager *whisperTaskManager) deleteFailedTask(agentID string, taskID int64) error {
	if manager == nil || manager.db == nil || !isValidMediaPreviewAgentID(agentID) || taskID <= 0 {
		return errors.New("任务参数无效")
	}
	result, err := manager.db.Exec(`DELETE FROM whisper_task WHERE id = ? AND agent_id = ? AND status = ?`, taskID, strings.TrimSpace(agentID), whisperTaskStatusFailed)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return errors.New("未找到可删除的失败任务")
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
		if err := whisperTasks.deleteFailedTask(strings.TrimSpace(request.AgentID), request.ID); err != nil {
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
