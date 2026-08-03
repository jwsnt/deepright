package main

// RVM video-matting tasks use the official Robust Video Matting inference
// script.  Keeping this queue separate from rembg means a long video never
// blocks image-body extraction.

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
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	rvmTaskQueued       = "queued"
	rvmTaskRunning      = "running"
	rvmTaskCompleted    = "completed"
	rvmTaskCancelled    = "cancelled"
	rvmTaskFailed       = "failed"
	rvmTaskLogLimit     = 128 * 1024
	rvmProbeTimeout     = time.Minute
	rvmScenarioStandard = "standard"
	rvmScenarioQuality  = "quality"
	rvmScenarioFast     = "fast"
	// A domestic mirror for the checkpoint expected by the unmodified RVM
	// script. The install request includes the digest so the installer can
	// reject a changed community copy before placing it in the workspace.
	rvmDomesticCheckpointURL    = "https://hf-mirror.com/DJ02-a/video_matting_RVM/resolve/main/rvm_mobilenetv3.pth"
	rvmDomesticCheckpointSHA256 = "3c7c1d92033f7c38d6577c481d13a195d7d80a159b960f4f3119ac7b534cf4f8"
)

type rvmCheckConfig struct {
	CacheFor time.Duration
	Install  string
}
type rvmCheckResponse struct {
	Available bool   `json:"available"`
	Install   string `json:"install,omitempty"`
	Content   string `json:"content,omitempty"`
	AgentID   string `json:"agentId,omitempty"`
	Status    int    `json:"status"`
}
type rvmRuntime struct{ Python, Script, Checkpoint string }
type rvmScenario struct {
	Name            string
	DownsampleRatio string
	VideoMbps       string
	SequenceChunk   string
}

// rvmCompatibilityBootstrap runs the unmodified upstream script while fixing
// the PyAV and PyTorch API changes in Integration's own process boundary. In
// particular, it never writes to the Agent's downloaded RVM directory.
const rvmCompatibilityBootstrap = `
import os
import runpy
import sys
from fractions import Fraction

# PyTorch 2.6 changed torch.load's default to weights_only=True. RVM's
# official checkpoints use the conventional state-dict wrapper; retain the
# old behavior only when the installed torch supports this argument.
import inspect
import torch
try:
    _deepright_supports_weights_only = 'weights_only' in inspect.signature(torch.load).parameters
except (TypeError, ValueError):
    _deepright_supports_weights_only = False
if _deepright_supports_weights_only:
    _deepright_torch_load = torch.load
    def _deepright_compatible_torch_load(*args, **kwargs):
        kwargs.setdefault('weights_only', False)
        return _deepright_torch_load(*args, **kwargs)
    torch.load = _deepright_compatible_torch_load

script = os.path.abspath(sys.argv[1])
script_dir = os.path.dirname(script)
if script_dir not in sys.path:
    sys.path.insert(0, script_dir)

import inference_utils

def _deepright_video_writer_init(self, path, frame_rate, bit_rate=1000000):
    self.container = inference_utils.av.open(path, mode='w')
    try:
        self.stream = self.container.add_stream('h264', rate=Fraction(str(frame_rate)).limit_denominator(100000))
    except TypeError:
        self.stream = self.container.add_stream('h264', rate=frame_rate)
    self.stream.pix_fmt = 'yuv420p'
    self.stream.bit_rate = bit_rate

inference_utils.VideoWriter.__init__ = _deepright_video_writer_init
sys.argv = [script] + sys.argv[2:]
runpy.run_path(script, run_name='__main__')
`

type rvmCachedRuntime struct {
	checkedAt time.Time
	runtime   rvmRuntime
}
type rvmTask struct {
	ID          int64  `json:"id"`
	AgentID     string `json:"agentId"`
	Scenario    string `json:"scenario"`
	SourcePath  string `json:"sourcePath"`
	OutputPath  string `json:"outputPath,omitempty"`
	SavedAs     string `json:"savedAs,omitempty"`
	Status      string `json:"status"`
	Progress    int    `json:"progress"`
	StartedAt   string `json:"startedAt,omitempty"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	Logs        string `json:"logs,omitempty"`
	CancelAsked bool   `json:"cancelRequested"`
}
type rvmTaskPage struct {
	Tasks                 []rvmTask
	Total, Page, PageSize int
	Status                string
}
type rvmTaskCreateRequest struct {
	AgentID string              `json:"agentId"`
	Paths   []string            `json:"paths,omitempty"`
	Tasks   []rvmTaskCreateItem `json:"tasks,omitempty"`
}
type rvmTaskCreateItem struct {
	Path     string `json:"path"`
	Scenario string `json:"scenario"`
}
type rvmTaskActionRequest struct {
	AgentID string `json:"agentId"`
	ID      int64  `json:"id"`
}
type rvmTaskManager struct {
	cfg          *Config
	db           *sql.DB
	createMu, mu sync.Mutex
	runningID    int64
	cancel       context.CancelFunc
	wake         chan struct{}
}

var (
	rvmTasks          *rvmTaskManager
	rvmLookPath       = exec.LookPath
	rvmCommandContext = exec.CommandContext
	rvmNow            = time.Now
	rvmCheckCache     struct {
		sync.Mutex
		entries map[string]rvmCachedRuntime
	}
)

func rvmScenarioFor(value string) (string, rvmScenario, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = rvmScenarioStandard
	}
	values := map[string]rvmScenario{
		rvmScenarioStandard: {Name: "标准主体提取", VideoMbps: "4", SequenceChunk: "1"},
		rvmScenarioQuality:  {Name: "精细边缘提取", DownsampleRatio: "1", VideoMbps: "8", SequenceChunk: "1"},
		rvmScenarioFast:     {Name: "快速主体提取", DownsampleRatio: "0.25", VideoMbps: "2", SequenceChunk: "1"},
	}
	info, ok := values[value]
	return value, info, ok
}

func rvmTimestamp() string { return rvmNow().Format(time.RFC3339Nano) }

func parseRVMCheckConfig(raw map[string]interface{}) (rvmCheckConfig, error) {
	value, ok := raw["rvm"]
	if !ok {
		return rvmCheckConfig{}, errors.New("config/config.json.rvm 配置缺失")
	}
	section, ok := value.(map[string]interface{})
	if !ok || section == nil {
		return rvmCheckConfig{}, errors.New("config/config.json.rvm 必须是对象")
	}
	hours, ok := ffmpegCheckPositiveHours(section["check"])
	if !ok {
		return rvmCheckConfig{}, errors.New("config/config.json.rvm.check 必须是正整数（小时）")
	}
	install, ok := section["install"].(string)
	install = strings.TrimSpace(install)
	if !ok || install == "" {
		return rvmCheckConfig{}, errors.New("config/config.json.rvm.install 必须是非空字符串")
	}
	return rvmCheckConfig{CacheFor: time.Duration(hours) * time.Hour, Install: install}, nil
}

func rvmRuntimeFor(cacheFor time.Duration, workspace string) (rvmRuntime, bool) {
	for _, candidate := range rvmRuntimeCandidates(workspace) {
		key := candidate.Script + "\x00" + candidate.Checkpoint
		now := rvmNow()
		rvmCheckCache.Lock()
		cached, ok := rvmCheckCache.entries[key]
		rvmCheckCache.Unlock()
		if ok && cached.runtime.Python != "" && now.Before(cached.checkedAt.Add(cacheFor)) {
			return cached.runtime, true
		}
		for _, python := range rvmPythonCandidates() {
			probe, cancel := context.WithTimeout(context.Background(), rvmProbeTimeout)
			// The probe must exercise the same bootstrap as a real task. Otherwise
			// a current PyAV/PyTorch runtime can look unavailable even though the
			// compatibility wrapper can execute the upstream script safely.
			err := rvmCommandContext(probe, python, rvmProbeArgs(candidate.Script)...).Run()
			timedOut := probe.Err() != nil
			cancel()
			if err != nil || timedOut {
				continue
			}
			value := rvmRuntime{Python: python, Script: candidate.Script, Checkpoint: candidate.Checkpoint}
			rvmCheckCache.Lock()
			if rvmCheckCache.entries == nil {
				rvmCheckCache.entries = make(map[string]rvmCachedRuntime)
			}
			rvmCheckCache.entries[key] = rvmCachedRuntime{checkedAt: now, runtime: value}
			rvmCheckCache.Unlock()
			return value, true
		}
	}
	return rvmRuntime{}, false
}

func rvmRuntimeCandidates(workspace string) []rvmRuntime {
	values := make([]rvmRuntime, 0, 1)
	add := func(home, checkpoint string) {
		home, checkpoint = strings.TrimSpace(home), strings.TrimSpace(checkpoint)
		if home == "" || checkpoint == "" {
			return
		}
		script, err := filepath.Abs(filepath.Join(home, "inference.py"))
		if err != nil {
			return
		}
		checkpoint, err = filepath.Abs(checkpoint)
		if err != nil {
			return
		}
		scriptInfo, scriptErr := os.Stat(script)
		checkpointInfo, checkpointErr := os.Stat(checkpoint)
		if scriptErr != nil || scriptInfo.IsDir() || checkpointErr != nil || checkpointInfo.IsDir() || checkpointInfo.Size() == 0 {
			return
		}
		for _, value := range values {
			if value.Script == script && value.Checkpoint == checkpoint {
				return
			}
		}
		values = append(values, rvmRuntime{Script: script, Checkpoint: checkpoint})
	}
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return values
	}
	home := filepath.Join(workspace, "rvm")
	// Installation, checking and task execution use this same fixed layout.
	add(home, filepath.Join(home, "weights", "rvm_mobilenetv3.pth"))
	return values
}

func rvmRequiredPaths(workspace string) string {
	home := filepath.Join(workspace, "rvm")
	return filepath.Join(home, "inference.py") + " 与 " + filepath.Join(home, "weights", "rvm_mobilenetv3.pth")
}

func rvmPythonCandidates() []string {
	values := make([]string, 0, 4)
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		for _, existing := range values {
			if existing == value {
				return
			}
		}
		values = append(values, value)
	}
	if python, err := rvmLookPath("python3"); err == nil {
		add(python)
	}
	if runtime.GOOS == "darwin" {
		if candidates, err := filepath.Glob("/Library/Frameworks/Python.framework/Versions/*/bin/python3"); err == nil {
			for index := len(candidates) - 1; index >= 0; index-- {
				add(candidates[index])
			}
		}
	}
	return values
}

func rvmRuntimeForStartup(cacheFor time.Duration, cfg *Config) (rvmRuntime, bool) {
	if cfg == nil {
		return rvmRuntime{}, false
	}
	root, err := resolveIntegrationAgentDir(cfg.AgentDir)
	if err != nil {
		return rvmRuntime{}, false
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return rvmRuntime{}, false
	}
	for _, entry := range entries {
		if !entry.IsDir() || !isValidMediaPreviewAgentID(entry.Name()) {
			continue
		}
		workspace, workspaceErr := getWorkspaceByAgentID(cfg, entry.Name())
		if workspaceErr != nil {
			continue
		}
		if value, ok := rvmRuntimeFor(cacheFor, workspace); ok {
			return value, true
		}
	}
	return rvmRuntime{}, false
}

func rvmInstallRequest(template, workspace string) string {
	request := strings.ReplaceAll(template, "$workspace", workspace)
	checkpoint := filepath.Join(workspace, "rvm", "weights", "rvm_mobilenetv3.pth")
	return request + "\n\n模型下载要求：先按配置中的官方权重地址下载 `rvm_mobilenetv3.pth` 到 `" + checkpoint + "`。官方源失败后回退国内源 `" + rvmDomesticCheckpointURL + "`。下载连接超时最多 30 秒，连续 90 秒无字节进度必须终止并报告。完成后校验 SHA-256 必须为 `" + rvmDomesticCheckpointSHA256 + "`。"
}

func rvmWorkspaceForRequest(cfg *Config, agentID string) (string, string, error) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return "", "", errors.New("agentId is required")
	}
	workspace, err := getWorkspaceByAgentID(cfg, agentID)
	if err != nil {
		return "", "", err
	}
	return agentID, workspace, nil
}

func checkRVMDependency(integrationConfig *Config, agentID string) (int, rvmCheckResponse) {
	raw, _, err := readIntegrationStartupConfigRaw()
	if err != nil {
		return http.StatusInternalServerError, rvmCheckResponse{Content: "读取 config/config.json 失败: " + err.Error(), Status: 1}
	}
	checkConfig, err := parseRVMCheckConfig(raw)
	if err != nil {
		return http.StatusBadRequest, rvmCheckResponse{Content: err.Error(), Status: 1}
	}
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		_, available := rvmRuntimeForStartup(checkConfig.CacheFor, integrationConfig)
		if available {
			return http.StatusOK, rvmCheckResponse{Available: true, Status: 0}
		}
		return http.StatusOK, rvmCheckResponse{Content: "未检测到已安装的 Robust Video Matting", Status: 0}
	}
	agentID, workspace, err := rvmWorkspaceForRequest(integrationConfig, agentID)
	if err != nil {
		return http.StatusNotFound, rvmCheckResponse{Content: "Agent 工作目录不存在", Status: 1}
	}
	if _, available := rvmRuntimeFor(checkConfig.CacheFor, workspace); !available {
		return http.StatusOK, rvmCheckResponse{
			Install: rvmInstallRequest(checkConfig.Install, workspace),
			Content: "未检测到可用的 Robust Video Matting（需要 " + rvmRequiredPaths(workspace) + "）",
			AgentID: agentID,
			Status:  0,
		}
	}
	return http.StatusOK, rvmCheckResponse{Available: true, AgentID: agentID, Status: 0}
}
func writeRVMJSON(w http.ResponseWriter, code int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
func handleRVMCheck(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeRVMJSON(w, http.StatusMethodNotAllowed, rvmCheckResponse{Content: "仅支持 GET 请求", Status: 1})
			return
		}
		status, result := checkRVMDependency(cfg, r.URL.Query().Get("agentId"))
		writeRVMJSON(w, status, result)
	}
}

func ensureRVMTaskSchema(db *sql.DB) error {
	if db == nil {
		return errors.New("rvm task database is not ready")
	}
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS rvm_task (
		id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL, source_path TEXT NOT NULL, output_path TEXT NOT NULL DEFAULT '', saved_as TEXT NOT NULL DEFAULT '',
		scenario TEXT NOT NULL DEFAULT 'standard', status TEXT NOT NULL DEFAULT 'queued', progress INTEGER NOT NULL DEFAULT 0, started_at TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL, logs TEXT NOT NULL DEFAULT '', cancel_requested INTEGER NOT NULL DEFAULT 0
	); CREATE INDEX IF NOT EXISTS idx_rvm_task_agent_created ON rvm_task(agent_id, id DESC); CREATE INDEX IF NOT EXISTS idx_rvm_task_queue ON rvm_task(status, cancel_requested, id);`)
	if err != nil {
		return err
	}
	rows, err := db.Query(`PRAGMA table_info(rvm_task)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	hasScenario := false
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue interface{}
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		if name == "scenario" {
			hasScenario = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !hasScenario {
		_, err = db.Exec(`ALTER TABLE rvm_task ADD COLUMN scenario TEXT NOT NULL DEFAULT 'standard'`)
	}
	return err
}
func startRVMTaskManager(ctx context.Context, cfg *Config) {
	if cronDB == nil {
		log.Printf("[rvm] task queue unavailable: database is not ready")
		return
	}
	if err := ensureRVMTaskSchema(cronDB); err != nil {
		log.Printf("[rvm] ensure task schema failed: %v", err)
		return
	}
	m := &rvmTaskManager{cfg: cfg, db: cronDB, wake: make(chan struct{}, 1)}
	if err := m.recover(); err != nil {
		log.Printf("[rvm] recover interrupted tasks failed: %v", err)
	}
	rvmTasks = m
	go m.run(ctx)
}
func (m *rvmTaskManager) recover() error {
	now := rvmTimestamp()
	if _, err := m.db.Exec(`UPDATE rvm_task SET status=?,progress=0,started_at='',updated_at=?,logs=CASE WHEN length(logs)>? THEN substr(logs,-?) ELSE logs END || ? WHERE status=? AND cancel_requested=0`, rvmTaskQueued, now, rvmTaskLogLimit-2048, rvmTaskLogLimit/2, "\n[服务重启] 上次执行被中断，任务已恢复为排队中。\n", rvmTaskRunning); err != nil {
		return err
	}
	_, err := m.db.Exec(`UPDATE rvm_task SET status=?,updated_at=? WHERE status=? AND cancel_requested=1`, rvmTaskCancelled, now, rvmTaskRunning)
	return err
}
func (m *rvmTaskManager) signal() {
	select {
	case m.wake <- struct{}{}:
	default:
	}
	notifyIntegrationSleepConditionChanged(m.cfg)
}
func (m *rvmTaskManager) run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			m.cancelRunning()
			return
		}
		slotHeld, err := reserveModelTaskSlot(ctx, m.cfg, m.db, "rvm_task")
		if err != nil {
			if errors.Is(err, context.Canceled) {
				m.cancelRunning()
				return
			}
			log.Printf("[rvm] reserve shared task slot failed: %v", err)
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
			log.Printf("[rvm] claim task failed: %v", err)
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
func (m *rvmTaskManager) claim() (*rvmTask, error) {
	tx, err := m.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	task, err := scanRVMTask(tx.QueryRow(`SELECT id,agent_id,COALESCE(NULLIF(scenario,''),'standard'),source_path,output_path,saved_as,status,progress,started_at,created_at,updated_at,logs,cancel_requested FROM rvm_task WHERE status=? AND cancel_requested=0 ORDER BY id LIMIT 1`, rvmTaskQueued))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, tx.Commit()
	}
	if err != nil {
		return nil, err
	}
	now := rvmTimestamp()
	result, err := tx.Exec(`UPDATE rvm_task SET status=?,started_at=?,updated_at=?,progress=0 WHERE id=? AND status=? AND cancel_requested=0`, rvmTaskRunning, now, now, task.ID, rvmTaskQueued)
	if err != nil {
		return nil, err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return nil, tx.Commit()
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	task.Status, task.StartedAt, task.UpdatedAt, task.Progress = rvmTaskRunning, now, now, 0
	return &task, nil
}
func (m *rvmTaskManager) execute(parent context.Context, task rvmTask) {
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
	_, scenario, validScenario := rvmScenarioFor(task.Scenario)
	if !validScenario {
		_, scenario, _ = rvmScenarioFor(rvmScenarioStandard)
	}
	m.appendLog(task.ID, "开始使用 RVM 提取视频主体（"+scenario.Name+"）。优先使用 GPU，失败会自动回退 CPU。")
	m.appendLog(task.ID, "任务目的：提取视频主体，生成透明 MOV 和黑底 MP4 预览。")
	if err := m.extract(ctx, task); err != nil {
		if errors.Is(ctx.Err(), context.Canceled) || m.cancelled(task.ID) {
			m.finish(task.ID, rvmTaskCancelled, 0, "任务已取消。")
		} else {
			m.finish(task.ID, rvmTaskFailed, 0, "任务失败，具体原因："+err.Error())
		}
		return
	}
	m.finish(task.ID, rvmTaskCompleted, 100, "视频主体提取完成。")
}
func isRVMVideo(path string) bool {
	mime, ok := mediaPreviewMIMEType(path)
	return ok && strings.HasPrefix(mime, "video/")
}

// File names are useful for choosing the browser picker, but they are not a
// trustworthy media type boundary. ffprobe confirms that a client cannot put
// an arbitrary renamed file into the RVM queue.
func validateRVMVideo(ctx context.Context, path string) error {
	if !isRVMVideo(path) {
		return errors.New("仅支持视频文件")
	}
	ffprobePath, err := videoTrimLookPathFn("ffprobe")
	if err != nil {
		return errors.New("未检测到 FFprobe")
	}
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	output, err := videoTrimCommandContextFn(probeCtx, ffprobePath, "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=codec_type", "-of", "csv=p=0", path).CombinedOutput()
	if err != nil || probeCtx.Err() != nil || strings.TrimSpace(string(output)) != "video" {
		return errors.New("仅支持包含视频流的视频文件")
	}
	return nil
}
func (m *rvmTaskManager) extract(ctx context.Context, task rvmTask) error {
	status, ffmpeg := checkFFmpegDependency()
	if status != http.StatusOK || !ffmpeg.Available {
		return errors.New("FFmpeg 或 FFprobe 未安装")
	}
	source, _, status, msg := resolveMediaPreviewFile(m.cfg, task.AgentID, task.SourcePath)
	if status != 0 {
		return errors.New(msg)
	}
	if err := validateRVMVideo(ctx, source); err != nil {
		return err
	}
	workspace, err := getWorkspaceByAgentID(m.cfg, task.AgentID)
	if err != nil {
		return errors.New("Agent 工作目录不存在")
	}
	runtimeValue, ok := rvmRuntimeFor(time.Hour, workspace)
	if !ok {
		return errors.New("未检测到可用的 Robust Video Matting")
	}
	m.appendLog(task.ID, taskKnownPythonEnvironment(runtimeValue.Python, runtimeValue.Script))
	target, err := resolveCaseInsensitiveUnderRoot(workspace, filepath.FromSlash(task.OutputPath))
	if err != nil || !ensureWritablePathWithinRoot(workspace, target) {
		return errors.New("无法创建视频输出路径")
	}
	if target, err = m.ensureAvailableOutputPath(&task, workspace, target); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return errors.New("无法创建视频输出目录")
	}
	m.appendLog(task.ID, "原始视频绝对路径："+source)
	m.appendLog(task.ID, "输出透明 MOV 绝对路径："+target)
	m.appendLog(task.ID, "输出 MP4 预览绝对路径："+rvmPreviewMP4Path(target))
	m.appendLog(task.ID, "RVM 推理脚本："+runtimeValue.Script)
	m.appendLog(task.ID, "RVM 模型权重："+runtimeValue.Checkpoint)
	tmp, err := os.MkdirTemp(filepath.Dir(target), ".rvm-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	foreground := filepath.Join(tmp, "foreground.mov")
	alpha := filepath.Join(tmp, "alpha.mov")
	devices := rvmPreferredDevices()
	var lastErr error
	for index, device := range devices {
		if index > 0 {
			m.appendLog(task.ID, "GPU/MPS 推理失败，已清理临时输出并回退 CPU："+lastErr.Error())
			_ = os.Remove(foreground)
			_ = os.Remove(alpha)
		}
		m.appendLog(task.ID, "正在使用 "+device+" 执行 RVM 推理。")
		m.appendLog(task.ID, "实际 RVM 模型参数："+strings.Join(rvmInvocationArgs(runtimeValue, task.Scenario, device, source, foreground, alpha)[2:], " "))
		err = m.runRVM(ctx, task.ID, runtimeValue, task.Scenario, device, source, foreground, alpha)
		if err == nil {
			lastErr = nil
			break
		}
		if errors.Is(err, context.Canceled) {
			return err
		}
		lastErr = err
	}
	if lastErr != nil {
		return lastErr
	}
	ffmpegPath, err := videoTrimLookPathFn("ffmpeg")
	if err != nil {
		return errors.New("未检测到 FFmpeg")
	}
	target, err = m.composeTransparentMOV(ctx, task.ID, ffmpegPath, &task, workspace, target, foreground, alpha)
	if err != nil {
		return err
	}
	info, err := os.Stat(target)
	if err != nil || info.Size() == 0 {
		return errors.New("未生成有效的透明 MOV")
	}
	preview := rvmPreviewMP4Path(target)
	if err := m.composePreviewMP4(ctx, task.ID, ffmpegPath, target, preview); err != nil {
		return err
	}
	previewInfo, previewErr := os.Stat(preview)
	if previewErr != nil || previewInfo.Size() == 0 {
		return errors.New("未生成有效的 MP4 预览副本")
	}
	m.appendLog(task.ID, "最终输出透明 MOV 绝对路径："+target)
	m.appendLog(task.ID, "最终输出 MP4 预览绝对路径："+preview)
	m.progress(task.ID, 100)
	return nil
}
func rvmPreferredDevices() []string {
	if runtime.GOOS == "darwin" {
		return []string{"mps", "cpu"}
	}
	return []string{"cuda", "cpu"}
}

func rvmInvocationArgs(r rvmRuntime, scenarioValue, device, source, foreground, alpha string) []string {
	_, scenario, validScenario := rvmScenarioFor(scenarioValue)
	if !validScenario {
		_, scenario, _ = rvmScenarioFor(rvmScenarioStandard)
	}
	args := []string{"-c", rvmCompatibilityBootstrap, r.Script, "--variant", "mobilenetv3", "--checkpoint", r.Checkpoint, "--device", device, "--input-source", source, "--output-type", "video", "--output-foreground", foreground, "--output-alpha", alpha, "--output-video-mbps", scenario.VideoMbps, "--seq-chunk", scenario.SequenceChunk}
	if scenario.DownsampleRatio != "" {
		args = append(args, "--downsample-ratio", scenario.DownsampleRatio)
	}
	return args
}

func rvmProbeArgs(script string) []string {
	return []string{"-c", rvmCompatibilityBootstrap, script, "--help"}
}

func (m *rvmTaskManager) runRVM(ctx context.Context, id int64, r rvmRuntime, scenario, device, source, foreground, alpha string) error {
	cmd := rvmCommandContext(ctx, r.Python, rvmInvocationArgs(r, scenario, device, source, foreground, alpha)...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err = cmd.Start(); err != nil {
		return err
	}
	var captured strings.Builder
	var capturedMu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); m.capture(id, stdout, &captured, &capturedMu) }()
	go func() { defer wg.Done(); m.capture(id, stderr, &captured, &capturedMu) }()
	wg.Wait()
	if err = cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return context.Canceled
		}
		capturedMu.Lock()
		diagnostics := captured.String()
		capturedMu.Unlock()
		return taskExecutionError("RVM（"+device+"）", err, diagnostics)
	}
	for _, file := range []string{foreground, alpha} {
		info, statErr := os.Stat(file)
		if statErr != nil || info.Size() == 0 {
			return errors.New("RVM 未生成有效视频输出")
		}
	}
	return nil
}
func (m *rvmTaskManager) capture(id int64, r io.Reader, captured *strings.Builder, capturedMu *sync.Mutex) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 128*1024)
	scanner.Split(splitRembgOutput)
	for scanner.Scan() {
		line := scanner.Text()
		m.appendLog(id, line)
		if captured != nil && capturedMu != nil {
			capturedMu.Lock()
			if captured.Len() < 128*1024 {
				captured.WriteString(line + "\n")
			}
			capturedMu.Unlock()
		}
	}
	if err := scanner.Err(); err != nil {
		m.appendLog(id, "读取 RVM 输出失败："+err.Error())
	}
}
func (m *rvmTaskManager) progress(id int64, p int) {
	_, _ = m.db.Exec(`UPDATE rvm_task SET progress=CASE WHEN progress>? THEN progress ELSE ? END,updated_at=? WHERE id=? AND status=?`, p, p, rvmTimestamp(), id, rvmTaskRunning)
}
func (m *rvmTaskManager) appendLog(id int64, message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	if len(message) > 4096 {
		message = message[:4096] + "…"
	}
	now := rvmTimestamp()
	_, _ = m.db.Exec(`UPDATE rvm_task SET logs=CASE WHEN length(logs)>? THEN substr(logs,-?) ELSE logs END || ?,updated_at=? WHERE id=?`, rvmTaskLogLimit-8192, rvmTaskLogLimit/2, "["+now+"] "+message+"\n", now, id)
}
func (m *rvmTaskManager) finish(id int64, status string, p int, message string) {
	m.appendLog(id, message)
	_, _ = m.db.Exec(`UPDATE rvm_task SET status=?,progress=?,updated_at=? WHERE id=?`, status, p, rvmTimestamp(), id)
	notifyIntegrationSleepConditionChanged(m.cfg)
}
func (m *rvmTaskManager) cancelled(id int64) bool {
	var value int
	return m.db.QueryRow(`SELECT cancel_requested FROM rvm_task WHERE id=?`, id).Scan(&value) != nil || value != 0
}
func (m *rvmTaskManager) cancelRunning() {
	m.mu.Lock()
	cancel := m.cancel
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (m *rvmTaskManager) allocate(agentID, workspace, source string, reserved map[string]bool, except int64) (string, string, error) {
	stem := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
	stamp := rvmNow().Format("20060102_150405")
	for attempt := 0; attempt < 1000; attempt++ {
		name := stem + "_subject.mov"
		if attempt > 0 {
			name = stem + "_subject_" + stamp
			if attempt > 1 {
				name += "_" + strconv.Itoa(attempt)
			}
			name += ".mov"
		}
		relative := filepath.Join("videos", name)
		key := filepath.ToSlash(relative)
		if reserved[key] {
			continue
		}
		absolute, err := resolveCaseInsensitiveUnderRoot(workspace, relative)
		if err != nil || !ensureWritablePathWithinRoot(workspace, absolute) {
			return "", "", errors.New("无法创建视频输出路径")
		}
		var count int
		if err := m.db.QueryRow(`SELECT COUNT(*) FROM rvm_task WHERE agent_id=? AND output_path=? AND id<>?`, agentID, key, except).Scan(&count); err != nil {
			return "", "", err
		}
		if count != 0 {
			continue
		}
		preview := rvmPreviewMP4Path(absolute)
		if _, err := os.Lstat(absolute); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", "", err
		}
		if _, err := os.Lstat(preview); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", "", err
		}
		reserved[key] = true
		return relative, absolute, nil
	}
	return "", "", errors.New("无法分配视频输出文件名")
}

func rvmPreviewMP4Path(movPath string) string {
	return strings.TrimSuffix(movPath, filepath.Ext(movPath)) + ".mp4"
}

// rvmPreviewFFmpegArgs flattens the alpha-bearing MOV onto a black canvas.
// H.264/MP4 has no alpha channel, so mapping the MOV stream directly would
// discard alpha and leave its hidden RGB values visible in the preview.
func rvmPreviewFFmpegArgs(movPath, previewPath string) []string {
	return rvmPreviewFFmpegArgsWithCodec(movPath, previewPath, "libx264")
}

// rvmPreviewFFmpegArgsWithCodec keeps the filter graph stable while selecting
// a video encoder known to the local FFmpeg build. h264_videotoolbox receives
// an explicit bitrate because libx264's CRF/preset options do not apply to it.
// libx264 is the CPU fallback and mpeg4 supports minimal FFmpeg builds.
func rvmPreviewFFmpegArgsWithCodec(movPath, previewPath, codec string) []string {
	args := []string{
		"-n", "-i", movPath,
		"-filter_complex", "[0:v:0]format=rgba[foreground];color=c=black:s=16x16,format=rgba[background];[background][foreground]scale2ref[background][foreground];[background][foreground]overlay=shortest=1:format=auto,format=yuv420p[preview]",
		"-map", "[preview]",
	}
	if codec == "h264_videotoolbox" {
		args = append(args, "-c:v", codec, "-b:v", "8M")
	} else if codec == "mpeg4" {
		args = append(args, "-c:v", "mpeg4", "-q:v", "2")
	} else {
		args = append(args, "-c:v", "libx264", "-crf", "18", "-preset", "medium")
	}
	return append(args, "-pix_fmt", "yuv420p", "-movflags", "+faststart", previewPath)
}

func rvmTransparentMOVFFmpegArgs(foreground, alpha, target, codec string) []string {
	args := []string{"-n", "-i", foreground, "-i", alpha, "-filter_complex", "[0:v][1:v]alphamerge[v]", "-map", "[v]"}
	if codec == "qtrle" {
		return append(args, "-c:v", "qtrle", "-pix_fmt", "argb", target)
	}
	if codec == "prores_videotoolbox" {
		return append(args, "-c:v", codec, "-profile:v", "4", "-pix_fmt", "bgra", target)
	}
	return append(args, "-c:v", "prores_ks", "-profile:v", "4", target)
}

func (m *rvmTaskManager) composeTransparentMOV(ctx context.Context, taskID int64, ffmpegPath string, task *rvmTask, workspace, target, foreground, alpha string) (string, error) {
	codecs := []string{"prores_ks", "qtrle"}
	if ffmpegHardwareEncoderAvailable(ffmpegPath, "prores_videotoolbox") {
		codecs = append([]string{"prores_videotoolbox"}, codecs...)
	}
	for attempt := 0; attempt < 4; attempt++ {
		for index, codec := range codecs {
			if codec == "prores_videotoolbox" {
				m.appendLog(taskID, "正在使用 GPU 编码器 prores_videotoolbox 合成透明 MOV。")
				log.Printf("[ffmpeg] RVM transparent MOV: using GPU encoder %s", codec)
			} else {
				m.appendLog(taskID, "正在使用 CPU 编码器 "+codec+" 合成透明 MOV。")
				log.Printf("[ffmpeg] RVM transparent MOV: using CPU encoder %s", codec)
			}
			_, targetStatErr := os.Lstat(target)
			targetExisted := targetStatErr == nil
			if targetStatErr != nil && !errors.Is(targetStatErr, os.ErrNotExist) {
				return "", errors.New("无法检查透明 MOV 输出路径")
			}
			output, err := videoTrimCommandContextFn(ctx, ffmpegPath, rvmTransparentMOVFFmpegArgs(foreground, alpha, target, codec)...).CombinedOutput()
			if err == nil {
				return target, nil
			}
			if targetExisted {
				var allocateErr error
				target, allocateErr = m.ensureAvailableOutputPath(task, workspace, target)
				if allocateErr != nil {
					return "", allocateErr
				}
				break
			}
			// Failed encoders sometimes leave an empty or partial destination.
			// It belongs to this attempt, not to another task, so clear it before
			// trying the fallback codec with the same collision-safe name.
			_ = os.Remove(target)
			if index+1 < len(codecs) {
				if codec == "prores_videotoolbox" {
					m.appendLog(taskID, "GPU 编码器 prores_videotoolbox 失败，已回退到 CPU 编码器 "+codecs[index+1]+"。")
					log.Printf("[ffmpeg] RVM transparent MOV: GPU encoder %s failed: %s; falling back to CPU encoder %s", codec, strings.TrimSpace(string(output)), codecs[index+1])
				} else {
					m.appendLog(taskID, "当前 FFmpeg 不支持 "+codec+" 透明编码，已回退到 "+codecs[index+1]+"。")
					log.Printf("[ffmpeg] RVM transparent MOV: CPU encoder %s failed: %s; falling back to %s", codec, strings.TrimSpace(string(output)), codecs[index+1])
				}
				continue
			}
			return "", fmt.Errorf("合成透明 MOV 失败: %s", strings.TrimSpace(string(output)))
		}
	}
	return "", errors.New("无法分配透明 MOV 输出路径")
}

func (m *rvmTaskManager) composePreviewMP4(ctx context.Context, taskID int64, ffmpegPath, source, preview string) error {
	codecs := []string{"libx264", "mpeg4"}
	if ffmpegHardwareEncoderAvailable(ffmpegPath, "h264_videotoolbox") {
		codecs = append([]string{"h264_videotoolbox"}, codecs...)
	}
	for index, codec := range codecs {
		if codec == "h264_videotoolbox" {
			m.appendLog(taskID, "正在使用 GPU 编码器 h264_videotoolbox 生成 MP4 预览副本。")
			log.Printf("[ffmpeg] RVM MP4 preview: using GPU encoder %s", codec)
		} else {
			m.appendLog(taskID, "正在使用 CPU 编码器 "+codec+" 生成 MP4 预览副本。")
			log.Printf("[ffmpeg] RVM MP4 preview: using CPU encoder %s", codec)
		}
		if _, statErr := os.Lstat(preview); statErr == nil {
			return errors.New("MP4 预览输出已存在，已避免覆盖")
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return errors.New("无法检查 MP4 预览输出路径")
		}
		output, err := videoTrimCommandContextFn(ctx, ffmpegPath, rvmPreviewFFmpegArgsWithCodec(source, preview, codec)...).CombinedOutput()
		if err == nil {
			return nil
		}
		_ = os.Remove(preview)
		if index+1 < len(codecs) {
			if codec == "h264_videotoolbox" {
				m.appendLog(taskID, "GPU 编码器 h264_videotoolbox 失败，已回退到 CPU 编码器 "+codecs[index+1]+"。")
				log.Printf("[ffmpeg] RVM MP4 preview: GPU encoder %s failed: %s; falling back to CPU encoder %s", codec, strings.TrimSpace(string(output)), codecs[index+1])
			} else {
				m.appendLog(taskID, "当前 FFmpeg 不支持 "+codec+"，已回退到 "+codecs[index+1]+" 生成预览。")
				log.Printf("[ffmpeg] RVM MP4 preview: CPU encoder %s failed: %s; falling back to %s", codec, strings.TrimSpace(string(output)), codecs[index+1])
			}
			continue
		}
		return fmt.Errorf("生成 MP4 预览副本失败: %s", strings.TrimSpace(string(output)))
	}
	return errors.New("无法生成 MP4 预览副本")
}

// ensureAvailableOutputPath makes the output collision rule hold until the
// final FFmpeg write.  A file may appear after a task was queued, so it must
// never be overwritten merely because the originally allocated name was free.
func (m *rvmTaskManager) ensureAvailableOutputPath(task *rvmTask, workspace, target string) (string, error) {
	preview := rvmPreviewMP4Path(target)
	_, targetErr := os.Lstat(target)
	_, previewErr := os.Lstat(preview)
	if errors.Is(targetErr, os.ErrNotExist) && errors.Is(previewErr, os.ErrNotExist) {
		return target, nil
	}
	if (targetErr != nil && !errors.Is(targetErr, os.ErrNotExist)) || (previewErr != nil && !errors.Is(previewErr, os.ErrNotExist)) {
		return "", errors.New("无法检查视频输出路径")
	}
	relative, absolute, err := m.allocate(task.AgentID, workspace, task.SourcePath, map[string]bool{}, task.ID)
	if err != nil {
		return "", err
	}
	result, err := m.db.Exec(`UPDATE rvm_task SET output_path=?,saved_as=?,updated_at=? WHERE id=? AND agent_id=?`, filepath.ToSlash(relative), absolute, rvmTimestamp(), task.ID, task.AgentID)
	if err != nil {
		return "", err
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		return "", errors.New("无法更新视频输出路径")
	}
	task.OutputPath, task.SavedAs = filepath.ToSlash(relative), absolute
	m.appendLog(task.ID, "检测到同名输出，已改用带时间戳的文件名。")
	return absolute, nil
}
func (m *rvmTaskManager) create(request rvmTaskCreateRequest) ([]rvmTask, error) {
	if m == nil || m.cfg == nil || m.db == nil {
		return nil, errors.New("视频主体任务服务未就绪")
	}
	request.AgentID = strings.TrimSpace(request.AgentID)
	if len(request.Tasks) == 0 {
		request.Tasks = make([]rvmTaskCreateItem, 0, len(request.Paths))
		for _, path := range request.Paths {
			request.Tasks = append(request.Tasks, rvmTaskCreateItem{Path: path, Scenario: rvmScenarioStandard})
		}
	}
	if !isValidMediaPreviewAgentID(request.AgentID) || len(request.Tasks) == 0 || len(request.Tasks) > 64 {
		return nil, errors.New("agentId 和 1 至 64 个视频文件不能为空")
	}
	m.createMu.Lock()
	defer m.createMu.Unlock()
	workspace, err := getWorkspaceByAgentID(m.cfg, request.AgentID)
	if err != nil {
		return nil, errors.New("Agent 不存在")
	}
	type source struct{ relative, output, saved, scenario string }
	sources := []source{}
	seen, reserved := map[string]bool{}, map[string]bool{}
	for _, item := range request.Tasks {
		path := normalizeQuotedPathArg(item.Path)
		if seen[path] {
			continue
		}
		seen[path] = true
		scenarioID, _, validScenario := rvmScenarioFor(item.Scenario)
		if !validScenario {
			return nil, errors.New("视频主体提取场景无效")
		}
		absolute, _, status, msg := resolveMediaPreviewFile(m.cfg, request.AgentID, path)
		if status != 0 {
			return nil, errors.New(msg)
		}
		if err := validateRVMVideo(context.Background(), absolute); err != nil {
			return nil, err
		}
		relative, err := filepath.Rel(workspace, absolute)
		if err != nil || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return nil, errors.New("视频文件必须位于当前 Agent 工作目录")
		}
		output, saved, err := m.allocate(request.AgentID, workspace, relative, reserved, 0)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source{filepath.ToSlash(relative), filepath.ToSlash(output), saved, scenarioID})
	}
	if len(sources) == 0 {
		return nil, errors.New("未找到可添加的视频文件")
	}
	tx, err := m.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	now := rvmTimestamp()
	created := make([]rvmTask, 0, len(sources))
	for _, source := range sources {
		result, err := tx.Exec(`INSERT INTO rvm_task(agent_id,scenario,source_path,output_path,saved_as,status,progress,created_at,updated_at,logs,cancel_requested) VALUES(?,?,?,?,?,?,?,?,?,?,0)`, request.AgentID, source.scenario, source.relative, source.output, source.saved, rvmTaskQueued, 0, now, now, "")
		if err != nil {
			return nil, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}
		created = append(created, rvmTask{ID: id, AgentID: request.AgentID, Scenario: source.scenario, SourcePath: source.relative, OutputPath: source.output, SavedAs: source.saved, Status: rvmTaskQueued, CreatedAt: now, UpdatedAt: now})
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	m.signal()
	return created, nil
}
func validRVMStatus(value string) bool {
	switch value {
	case "", rvmTaskQueued, rvmTaskRunning, rvmTaskCompleted, rvmTaskCancelled, rvmTaskFailed:
		return true
	}
	return false
}
func (m *rvmTaskManager) list(agentID, status string, page int, logs bool) (rvmTaskPage, error) {
	result := rvmTaskPage{Tasks: []rvmTask{}, PageSize: 5}
	agentID = strings.TrimSpace(agentID)
	status = strings.TrimSpace(status)
	if !isValidMediaPreviewAgentID(agentID) || !validRVMStatus(status) {
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
	if err := m.db.QueryRow(`SELECT COUNT(*) FROM rvm_task WHERE `+where, args...).Scan(&result.Total); err != nil {
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
	rows, err := m.db.Query(`SELECT id,agent_id,COALESCE(NULLIF(scenario,''),'standard'),source_path,output_path,saved_as,status,progress,started_at,created_at,updated_at,`+column+`,cancel_requested FROM rvm_task WHERE `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		task, err := scanRVMTask(rows)
		if err != nil {
			return result, err
		}
		result.Tasks = append(result.Tasks, task)
	}
	return result, rows.Err()
}
func (m *rvmTaskManager) log(agentID string, id int64) (rvmTask, error) {
	if id <= 0 || !isValidMediaPreviewAgentID(strings.TrimSpace(agentID)) {
		return rvmTask{}, errors.New("任务参数无效")
	}
	return scanRVMTask(m.db.QueryRow(`SELECT id,agent_id,COALESCE(NULLIF(scenario,''),'standard'),source_path,output_path,saved_as,status,progress,started_at,created_at,updated_at,logs,cancel_requested FROM rvm_task WHERE id=? AND agent_id=?`, id, strings.TrimSpace(agentID)))
}
func (m *rvmTaskManager) cancelTask(agentID string, id int64) error {
	now := rvmTimestamp()
	result, err := m.db.Exec(`UPDATE rvm_task SET cancel_requested=1,status=CASE WHEN status=? THEN ? ELSE status END,updated_at=?,logs=CASE WHEN length(logs)>? THEN substr(logs,-?) ELSE logs END || ? WHERE id=? AND agent_id=? AND status IN (?,?)`, rvmTaskQueued, rvmTaskCancelled, now, rvmTaskLogLimit-4096, rvmTaskLogLimit/2, "["+now+"] 已请求取消任务。\n", id, strings.TrimSpace(agentID), rvmTaskQueued, rvmTaskRunning)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return errors.New("仅可取消排队中或执行中的任务")
	}
	m.mu.Lock()
	running, cancel := m.runningID == id, m.cancel
	m.mu.Unlock()
	if running && cancel != nil {
		cancel()
	}
	notifyIntegrationSleepConditionChanged(m.cfg)
	return nil
}
func (m *rvmTaskManager) restart(agentID string, id int64) error {
	agentID = strings.TrimSpace(agentID)
	var source, scenario string
	if err := m.db.QueryRow(`SELECT source_path,COALESCE(NULLIF(scenario,''),'standard') FROM rvm_task WHERE id=? AND agent_id=? AND status IN (?, ?)`, id, agentID, rvmTaskFailed, rvmTaskCancelled).Scan(&source, &scenario); err != nil {
		return errors.New("仅可重新开始失败或已取消任务")
	}
	workspace, err := getWorkspaceByAgentID(m.cfg, agentID)
	if err != nil {
		return errors.New("Agent 不存在")
	}
	output, saved, err := m.allocate(agentID, workspace, source, map[string]bool{}, id)
	if err != nil {
		return err
	}
	now := rvmTimestamp()
	result, err := m.db.Exec(`UPDATE rvm_task SET scenario=?,output_path=?,saved_as=?,status=?,progress=0,started_at='',cancel_requested=0,updated_at=?,logs='' WHERE id=? AND agent_id=? AND status IN (?, ?)`, scenario, filepath.ToSlash(output), saved, rvmTaskQueued, now, id, agentID, rvmTaskFailed, rvmTaskCancelled)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return errors.New("重新开始任务失败")
	}
	m.signal()
	return nil
}
func (m *rvmTaskManager) deleteFailedOrCancelled(agentID string, id int64) error {
	result, err := m.db.Exec(`DELETE FROM rvm_task WHERE id=? AND agent_id=? AND status IN (?, ?)`, id, strings.TrimSpace(agentID), rvmTaskFailed, rvmTaskCancelled)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return errors.New("仅可删除失败或已取消任务")
	}
	return nil
}

type rvmScanner interface{ Scan(...interface{}) error }

func scanRVMTask(scanner rvmScanner) (rvmTask, error) {
	var task rvmTask
	var cancelled int
	err := scanner.Scan(&task.ID, &task.AgentID, &task.Scenario, &task.SourcePath, &task.OutputPath, &task.SavedAs, &task.Status, &task.Progress, &task.StartedAt, &task.CreatedAt, &task.UpdatedAt, &task.Logs, &cancelled)
	task.CancelAsked = cancelled != 0
	return task, err
}
func readRVMAction(w http.ResponseWriter, r *http.Request) (rvmTaskActionRequest, error) {
	var request rvmTaskActionRequest
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
func handleRVMTasks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if rvmTasks == nil {
			writeRVMJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": 1, "content": "视频主体任务服务未就绪"})
			return
		}
		switch r.Method {
		case http.MethodGet:
			page := 1
			if raw := strings.TrimSpace(r.URL.Query().Get("page")); raw != "" {
				value, err := strconv.Atoi(raw)
				if err != nil || value < 1 {
					writeRVMJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "页码必须是正整数"})
					return
				}
				page = value
			}
			value, err := rvmTasks.list(r.URL.Query().Get("agentId"), r.URL.Query().Get("status"), page, false)
			if err != nil {
				writeRVMJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
				return
			}
			writeRVMJSON(w, http.StatusOK, map[string]interface{}{"status": 0, "tasks": value.Tasks, "total": value.Total, "page": value.Page, "pageSize": value.PageSize, "taskStatus": value.Status})
		case http.MethodPost:
			var request rvmTaskCreateRequest
			decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
				writeRVMJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "请求内容无效"})
				return
			}
			value, err := rvmTasks.create(request)
			if err != nil {
				writeRVMJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
				return
			}
			writeRVMJSON(w, http.StatusOK, map[string]interface{}{"status": 0, "tasks": value})
		default:
			w.Header().Set("Allow", "GET, POST")
			writeRVMJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"status": 1, "content": "仅支持 GET 或 POST 请求"})
		}
	}
}
func rvmActionHandler(action func(*rvmTaskManager, rvmTaskActionRequest) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeRVMJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"status": 1, "content": "仅支持 POST 请求"})
			return
		}
		if rvmTasks == nil {
			writeRVMJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": 1, "content": "视频主体任务服务未就绪"})
			return
		}
		request, err := readRVMAction(w, r)
		if err == nil {
			err = action(rvmTasks, request)
		}
		if err != nil {
			writeRVMJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		writeRVMJSON(w, http.StatusOK, map[string]interface{}{"status": 0})
	}
}
func handleRVMTaskCancel() http.HandlerFunc {
	return rvmActionHandler(func(m *rvmTaskManager, r rvmTaskActionRequest) error { return m.cancelTask(r.AgentID, r.ID) })
}
func handleRVMTaskRestart() http.HandlerFunc {
	return rvmActionHandler(func(m *rvmTaskManager, r rvmTaskActionRequest) error { return m.restart(r.AgentID, r.ID) })
}
func handleRVMTaskDelete() http.HandlerFunc {
	return rvmActionHandler(func(m *rvmTaskManager, r rvmTaskActionRequest) error {
		return m.deleteFailedOrCancelled(r.AgentID, r.ID)
	})
}
func handleRVMTaskLog() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeRVMJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"status": 1, "content": "仅支持 GET 请求"})
			return
		}
		if rvmTasks == nil {
			writeRVMJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": 1, "content": "视频主体任务服务未就绪"})
			return
		}
		id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
		if err == nil {
			task, lookupErr := rvmTasks.log(r.URL.Query().Get("agentId"), id)
			if lookupErr == nil {
				writeRVMJSON(w, http.StatusOK, map[string]interface{}{"status": 0, "task": task})
				return
			}
		}
		writeRVMJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "任务参数无效"})
	}
}
