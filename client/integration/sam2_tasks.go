package main

// SAM2 object extraction tasks intentionally use a separate durable queue from
// RVM: SAM2 needs a user prompt on the first frame and tracks that exact object.

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"image/png"
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
	sam2ModelTiny  = "tiny"
	sam2ModelSmall = "small"
	sam2ModelBase  = "base"
	sam2ModelLarge = "large"
)

type sam2Model struct{ ID, Name, Filename, Config, OfficialURL, BackupURL string }

var sam2Models = map[string]sam2Model{
	sam2ModelTiny:  {sam2ModelTiny, "速度优先（tiny）", "sam2_hiera_tiny.pt", "sam2_hiera_t.yaml", "https://dl.fbaipublicfiles.com/segment_anything_2/072824/sam2_hiera_tiny.pt", "https://hf-mirror.com/facebook/sam2-hiera-tiny/resolve/main/sam2_hiera_tiny.pt"},
	sam2ModelSmall: {sam2ModelSmall, "兼顾精度（small）", "sam2_hiera_small.pt", "sam2_hiera_s.yaml", "https://dl.fbaipublicfiles.com/segment_anything_2/072824/sam2_hiera_small.pt", "https://hf-mirror.com/facebook/sam2-hiera-small/resolve/main/sam2_hiera_small.pt"},
	sam2ModelBase:  {sam2ModelBase, "综合均衡（base）", "sam2_hiera_base_plus.pt", "sam2_hiera_b+.yaml", "https://dl.fbaipublicfiles.com/segment_anything_2/072824/sam2_hiera_base_plus.pt", "https://hf-mirror.com/facebook/sam2-hiera-base-plus/resolve/main/sam2_hiera_base_plus.pt"},
	sam2ModelLarge: {sam2ModelLarge, "高精度分割（large）", "sam2_hiera_large.pt", "sam2_hiera_l.yaml", "https://dl.fbaipublicfiles.com/segment_anything_2/072824/sam2_hiera_large.pt", "https://hf-mirror.com/facebook/sam2-hiera-large/resolve/main/sam2_hiera_large.pt"},
}

type sam2Point struct {
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Foreground bool    `json:"foreground"`
}
type sam2Box struct {
	XMin float64 `json:"xMin"`
	YMin float64 `json:"yMin"`
	XMax float64 `json:"xMax"`
	YMax float64 `json:"yMax"`
}
type sam2Prompt struct {
	Points   []sam2Point `json:"points,omitempty"`
	Boxes    []sam2Box   `json:"boxes,omitempty"`
	MaskPath string      `json:"maskPath,omitempty"`
	MaskMode string      `json:"maskMode,omitempty"`
}
type sam2Task struct {
	ID          int64      `json:"id"`
	AgentID     string     `json:"agentId"`
	Model       string     `json:"model"`
	ModelName   string     `json:"modelName"`
	SourcePath  string     `json:"sourcePath"`
	Prompt      sam2Prompt `json:"prompt"`
	OutputPath  string     `json:"outputPath,omitempty"`
	SavedAs     string     `json:"savedAs,omitempty"`
	Status      string     `json:"status"`
	Progress    int        `json:"progress"`
	StartedAt   string     `json:"startedAt,omitempty"`
	CreatedAt   string     `json:"createdAt"`
	UpdatedAt   string     `json:"updatedAt"`
	Logs        string     `json:"logs,omitempty"`
	CancelAsked bool       `json:"cancelRequested"`
}
type sam2TaskRequest struct {
	AgentID    string     `json:"agentId"`
	SourcePath string     `json:"sourcePath"`
	Model      string     `json:"model"`
	Prompt     sam2Prompt `json:"prompt"`
}

// sam2PreviewRequest deliberately has the same inference inputs as a task, but
// never enters the durable queue: it segments only frame zero for confirmation.
type sam2PreviewRequest struct {
	AgentID    string     `json:"agentId"`
	SourcePath string     `json:"sourcePath"`
	Model      string     `json:"model"`
	Prompt     sam2Prompt `json:"prompt"`
}
type sam2PreviewResult struct {
	MaskData string  `json:"maskData"`
	Coverage float64 `json:"coverage"`
}
type sam2ActionRequest struct {
	AgentID string `json:"agentId"`
	ID      int64  `json:"id"`
}
type sam2Page struct {
	Tasks                 []sam2Task
	Total, Page, PageSize int
	Status                string
}
type sam2CheckConfig struct {
	CacheFor time.Duration
	Install  string
}
type sam2CheckResponse struct {
	Available bool   `json:"available"`
	Install   string `json:"install,omitempty"`
	Content   string `json:"content,omitempty"`
	AgentID   string `json:"agentId,omitempty"`
	Status    int    `json:"status"`
}
type sam2TaskManager struct {
	cfg          *Config
	db           *sql.DB
	createMu, mu sync.Mutex
	runningID    int64
	cancel       context.CancelFunc
	wake         chan struct{}
}

var sam2Tasks *sam2TaskManager

func sam2ModelFor(value string) (sam2Model, bool) {
	if strings.TrimSpace(value) == "" {
		value = sam2ModelTiny
	}
	m, ok := sam2Models[strings.TrimSpace(value)]
	return m, ok
}
func sam2ModelName(value string) string {
	if m, ok := sam2ModelFor(value); ok {
		return m.Name
	}
	return sam2Models[sam2ModelTiny].Name
}

func parseSAM2CheckConfig(raw map[string]interface{}) (sam2CheckConfig, error) {
	v, ok := raw["sam2"]
	if !ok {
		return sam2CheckConfig{}, errors.New("config/config.json.sam2 配置缺失")
	}
	section, ok := v.(map[string]interface{})
	if !ok {
		return sam2CheckConfig{}, errors.New("config/config.json.sam2 必须是对象")
	}
	hours, ok := ffmpegCheckPositiveHours(section["check"])
	if !ok {
		return sam2CheckConfig{}, errors.New("config/config.json.sam2.check 必须是正整数（小时）")
	}
	install, ok := section["install"].(string)
	install = strings.TrimSpace(install)
	if !ok || install == "" {
		return sam2CheckConfig{}, errors.New("config/config.json.sam2.install 必须是非空字符串")
	}
	return sam2CheckConfig{time.Duration(hours) * time.Hour, install}, nil
}

func sam2Home(workspace string) string { return filepath.Join(workspace, "sam2") }
func sam2Checkpoint(workspace string, model sam2Model) string {
	return filepath.Join(sam2Home(workspace), "checkpoints", model.Filename)
}
func sam2ConfigPath(workspace string, model sam2Model) string {
	return filepath.Join(sam2Home(workspace), "sam2", "configs", "sam2", model.Config)
}
func sam2ConfigName(model sam2Model) string {
	return filepath.ToSlash(filepath.Join("configs", "sam2", model.Config))
}
func sam2Python(workspace string) (string, bool) {
	if info, err := os.Stat(sam2Home(workspace)); err != nil || !info.IsDir() {
		return "", false
	}
	candidates := make([]string, 0, 4)
	add := func(python string) {
		python = strings.TrimSpace(python)
		if python == "" {
			return
		}
		for _, existing := range candidates {
			if existing == python {
				return
			}
		}
		candidates = append(candidates, python)
	}
	// Prefer the framework interpreter: it is the validated runtime on macOS and
	// avoids a pyenv/virtualenv shim making an installed source tree look absent.
	if runtime.GOOS == "darwin" {
		if paths, err := filepath.Glob("/Library/Frameworks/Python.framework/Versions/*/bin/python3"); err == nil {
			for index := len(paths) - 1; index >= 0; index-- {
				add(paths[index])
			}
		}
	}
	for _, python := range rvmPythonCandidates() {
		add(python)
	}
	env := sam2CommandEnvironment(workspace)
	for _, python := range candidates {
		ctx, cancel := context.WithTimeout(context.Background(), rvmProbeTimeout)
		cmd := rvmCommandContext(ctx, python, "-c", "import sam2; from sam2.build_sam import build_sam2_video_predictor; from sam2.modeling.backbones.hieradet import Hiera")
		cmd.Env = env
		output, err := cmd.CombinedOutput()
		cancel()
		if err == nil {
			return python, true
		}
		log.Printf("[sam2] Python probe failed for %s: %s", python, strings.TrimSpace(string(output)))
	}
	return "", false
}

func sam2CommandEnvironment(workspace string) []string {
	env := make([]string, 0, len(os.Environ())+1)
	for _, item := range os.Environ() {
		if !strings.HasPrefix(item, "PYTHONPATH=") {
			env = append(env, item)
		}
	}
	return append(env, "PYTHONPATH="+sam2Home(workspace))
}
func sam2InstallRequest(template, workspace string) string {
	return strings.ReplaceAll(template, "$workspace", workspace)
}
func checkSAM2Dependency(cfg *Config, agentID string) (int, sam2CheckResponse) {
	raw, _, err := readIntegrationStartupConfigRaw()
	if err != nil {
		return http.StatusInternalServerError, sam2CheckResponse{Status: 1, Content: "读取 config/config.json 失败: " + err.Error()}
	}
	check, err := parseSAM2CheckConfig(raw)
	if err != nil {
		return http.StatusBadRequest, sam2CheckResponse{Status: 1, Content: err.Error()}
	}
	agentID = strings.TrimSpace(agentID)
	if !isValidMediaPreviewAgentID(agentID) {
		return http.StatusBadRequest, sam2CheckResponse{Status: 1, Content: "Agent 工作目录不存在"}
	}
	workspace, err := getWorkspaceByAgentID(cfg, agentID)
	if err != nil {
		return http.StatusNotFound, sam2CheckResponse{Status: 1, Content: "Agent 工作目录不存在"}
	}
	tiny := sam2Models[sam2ModelTiny]
	if _, ok := sam2Python(workspace); !ok || !nonEmptyFile(sam2Checkpoint(workspace, tiny)) || !nonEmptyFile(sam2ConfigPath(workspace, tiny)) {
		return http.StatusOK, sam2CheckResponse{Status: 0, AgentID: agentID, Install: sam2InstallRequest(check.Install, workspace), Content: "未检测到可用的 SAM2 或基础 tiny 模型"}
	}
	_ = check
	return http.StatusOK, sam2CheckResponse{Status: 0, AgentID: agentID, Available: true}
}
func nonEmptyFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > 0
}
func writeSAM2JSON(w http.ResponseWriter, code int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
func handleSAM2Check(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeSAM2JSON(w, http.StatusMethodNotAllowed, sam2CheckResponse{Status: 1, Content: "仅支持 GET 请求"})
			return
		}
		status, result := checkSAM2Dependency(cfg, r.URL.Query().Get("agentId"))
		writeSAM2JSON(w, status, result)
	}
}

func ensureSAM2TaskSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS sam2_task (id INTEGER PRIMARY KEY AUTOINCREMENT,agent_id TEXT NOT NULL,model TEXT NOT NULL DEFAULT 'tiny',source_path TEXT NOT NULL,prompt_json TEXT NOT NULL DEFAULT '{}',output_path TEXT NOT NULL DEFAULT '',saved_as TEXT NOT NULL DEFAULT '',status TEXT NOT NULL DEFAULT 'queued',progress INTEGER NOT NULL DEFAULT 0,started_at TEXT NOT NULL DEFAULT '',created_at TEXT NOT NULL,updated_at TEXT NOT NULL,logs TEXT NOT NULL DEFAULT '',cancel_requested INTEGER NOT NULL DEFAULT 0); CREATE INDEX IF NOT EXISTS idx_sam2_task_agent_created ON sam2_task(agent_id,id DESC); CREATE INDEX IF NOT EXISTS idx_sam2_task_queue ON sam2_task(status,cancel_requested,id);`)
	return err
}
func startSAM2TaskManager(ctx context.Context, cfg *Config) {
	if cronDB == nil {
		return
	}
	if err := ensureSAM2TaskSchema(cronDB); err != nil {
		log.Printf("[sam2] ensure task schema failed: %v", err)
		return
	}
	m := &sam2TaskManager{cfg: cfg, db: cronDB, wake: make(chan struct{}, 1)}
	if err := m.recover(); err != nil {
		log.Printf("[sam2] recover failed: %v", err)
	}
	sam2Tasks = m
	go m.run(ctx)
}
func (m *sam2TaskManager) recover() error {
	now := rvmTimestamp()
	_, err := m.db.Exec(`UPDATE sam2_task SET status=?,progress=0,started_at='',updated_at=?,logs=logs || ? WHERE status=? AND cancel_requested=0`, rvmTaskQueued, now, "\n[服务重启] 上次执行被中断，任务已恢复为排队中。\n", rvmTaskRunning)
	return err
}
func (m *sam2TaskManager) signal() {
	select {
	case m.wake <- struct{}{}:
	default:
	}
	notifyIntegrationSleepConditionChanged(m.cfg)
}
func (m *sam2TaskManager) run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			m.cancelRunning()
			return
		}
		held, err := reserveModelTaskSlot(ctx, m.cfg, m.db, "sam2_task")
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			time.Sleep(time.Second)
			continue
		}
		if !held {
			select {
			case <-ctx.Done():
				return
			case <-m.wake:
			}
			continue
		}
		task, err := m.claim()
		if err != nil {
			releaseModelTaskSlot(m.cfg)
			time.Sleep(time.Second)
			continue
		}
		if task == nil {
			releaseModelTaskSlot(m.cfg)
			select {
			case <-ctx.Done():
				return
			case <-m.wake:
			}
			continue
		}
		m.execute(ctx, *task)
		releaseModelTaskSlot(m.cfg)
	}
}
func (m *sam2TaskManager) claim() (*sam2Task, error) {
	tx, err := m.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	task, err := scanSAM2Task(tx.QueryRow(`SELECT id,agent_id,model,source_path,prompt_json,output_path,saved_as,status,progress,started_at,created_at,updated_at,logs,cancel_requested FROM sam2_task WHERE status=? AND cancel_requested=0 ORDER BY id LIMIT 1`, rvmTaskQueued))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, tx.Commit()
	}
	if err != nil {
		return nil, err
	}
	now := rvmTimestamp()
	res, err := tx.Exec(`UPDATE sam2_task SET status=?,started_at=?,updated_at=? WHERE id=? AND status=?`, rvmTaskRunning, now, now, task.ID, rvmTaskQueued)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return nil, tx.Commit()
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	task.Status = rvmTaskRunning
	task.StartedAt = now
	return &task, nil
}
func (m *sam2TaskManager) execute(parent context.Context, task sam2Task) {
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
	model, _ := sam2ModelFor(task.Model)
	m.appendLog(task.ID, "开始使用 SAM2 提取视频物体，选择模型："+model.Name+"（"+model.Filename+"）。")
	m.appendLog(task.ID, "首帧提示："+sam2PromptDescription(task.Prompt))
	if err := m.extract(ctx, task); err != nil {
		if errors.Is(ctx.Err(), context.Canceled) || m.cancelled(task.ID) {
			m.finish(task.ID, rvmTaskCancelled, 0, "任务已取消。")
		} else {
			m.finish(task.ID, rvmTaskFailed, 0, "任务失败，具体原因："+err.Error())
		}
		return
	}
	m.finish(task.ID, rvmTaskCompleted, 100, "视频物体提取完成，模型："+model.Name+"。")
}
func sam2PromptDescription(p sam2Prompt) string {
	values := []string{}
	if len(p.Points) > 0 {
		values = append(values, fmt.Sprintf("点击点 %d 个", len(p.Points)))
	}
	if len(p.Boxes) > 0 {
		values = append(values, fmt.Sprintf("框选 %d 个", len(p.Boxes)))
	}
	if p.MaskPath != "" {
		values = append(values, "掩码图（"+p.MaskMode+"）")
	}
	if len(values) == 0 {
		return "自动提取首帧最显著主体"
	}
	return strings.Join(values, "、")
}

func sam2ValidatePrompt(p sam2Prompt) error {
	if len(p.Points) > 64 || len(p.Boxes) > 1 {
		return errors.New("首帧提示数量超出限制")
	}
	for _, v := range p.Points {
		if v.X < 0 || v.Y < 0 {
			return errors.New("点击点坐标无效")
		}
	}
	for _, b := range p.Boxes {
		if b.XMin < 0 || b.YMin < 0 || b.XMax <= b.XMin || b.YMax <= b.YMin {
			return errors.New("框选区域无效")
		}
	}
	if p.MaskPath != "" {
		switch p.MaskMode {
		case "green", "white", "black", "transparent":
		default:
			return errors.New("掩码图模式无效")
		}
	}
	if len(p.Points) == 0 && len(p.Boxes) == 0 && p.MaskPath == "" {
		return nil // 无提示时由 SAM2 自动选择首帧最显著主体。
	}
	foreground := len(p.Boxes) > 0 || p.MaskPath != ""
	for _, point := range p.Points {
		foreground = foreground || point.Foreground
	}
	if !foreground {
		return errors.New("首帧提示必须包含目标点击点、框选区域或掩码图")
	}
	return nil
}
func (m *sam2TaskManager) create(req sam2TaskRequest) (sam2Task, error) {
	if m == nil || m.cfg == nil {
		return sam2Task{}, errors.New("视频提取物体任务服务未就绪")
	}
	req.AgentID = strings.TrimSpace(req.AgentID)
	model, ok := sam2ModelFor(req.Model)
	if !ok {
		return sam2Task{}, errors.New("SAM2 模型无效")
	}
	if !isValidMediaPreviewAgentID(req.AgentID) || strings.TrimSpace(req.SourcePath) == "" {
		return sam2Task{}, errors.New("agentId 和视频文件不能为空")
	}
	if err := sam2ValidatePrompt(req.Prompt); err != nil {
		return sam2Task{}, err
	}
	m.createMu.Lock()
	defer m.createMu.Unlock()
	workspace, err := getWorkspaceByAgentID(m.cfg, req.AgentID)
	if err != nil {
		return sam2Task{}, errors.New("Agent 不存在")
	}
	source, _, status, msg := resolveMediaPreviewFile(m.cfg, req.AgentID, normalizeQuotedPathArg(req.SourcePath))
	if status != 0 {
		return sam2Task{}, errors.New(msg)
	}
	if err := validateRVMVideo(context.Background(), source); err != nil {
		return sam2Task{}, err
	}
	relative, err := filepath.Rel(workspace, source)
	if err != nil || strings.HasPrefix(relative, "..") {
		return sam2Task{}, errors.New("视频文件必须位于当前 Agent 工作目录")
	}
	if req.Prompt.MaskPath != "" {
		mask, err := sam2MaskPath(workspace, req.Prompt.MaskPath)
		if err != nil {
			return sam2Task{}, err
		}
		rel, err := filepath.Rel(workspace, mask)
		if err != nil {
			return sam2Task{}, errors.New("掩码图必须位于当前 Agent 工作目录")
		}
		req.Prompt.MaskPath = filepath.ToSlash(rel)
	}
	output, saved, err := m.allocate(req.AgentID, workspace, relative, 0)
	if err != nil {
		return sam2Task{}, err
	}
	promptJSON, err := json.Marshal(req.Prompt)
	if err != nil {
		return sam2Task{}, err
	}
	now := rvmTimestamp()
	res, err := m.db.Exec(`INSERT INTO sam2_task(agent_id,model,source_path,prompt_json,output_path,saved_as,status,progress,created_at,updated_at,logs) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, req.AgentID, model.ID, filepath.ToSlash(relative), string(promptJSON), output, saved, rvmTaskQueued, 0, now, now, "")
	if err != nil {
		return sam2Task{}, err
	}
	id, _ := res.LastInsertId()
	task := sam2Task{ID: id, AgentID: req.AgentID, Model: model.ID, ModelName: model.Name, SourcePath: filepath.ToSlash(relative), Prompt: req.Prompt, OutputPath: output, SavedAs: saved, Status: rvmTaskQueued, CreatedAt: now, UpdatedAt: now}
	m.appendLog(id, "任务已创建，模型："+model.Name+"；提示："+sam2PromptDescription(req.Prompt)+"。")
	m.signal()
	return task, nil
}
func sam2MaskPath(workspace, input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", nil
	}
	var path string
	if filepath.IsAbs(input) {
		rel, err := filepath.Rel(workspace, input)
		if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
			return "", errors.New("掩码图必须位于当前 Agent 工作目录")
		}
		path, err = resolveCaseInsensitiveUnderRoot(workspace, rel)
		if err != nil {
			return "", errors.New("掩码图路径无效")
		}
	} else {
		var err error
		path, err = resolveCaseInsensitiveUnderRoot(workspace, input)
		if err != nil {
			return "", errors.New("掩码图路径无效")
		}
	}
	if !nonEmptyFile(path) {
		return "", errors.New("掩码图不存在或为空")
	}
	return path, nil
}

// previewFirstFrame runs the exact first-frame branch used by video extraction
// (including automatic salient-subject selection), without propagating through
// the rest of the video or creating a task record.
func (m *sam2TaskManager) previewFirstFrame(ctx context.Context, req sam2PreviewRequest) (sam2PreviewResult, error) {
	if m == nil || m.cfg == nil {
		return sam2PreviewResult{}, errors.New("视频提取物体任务服务未就绪")
	}
	req.AgentID = strings.TrimSpace(req.AgentID)
	model, ok := sam2ModelFor(req.Model)
	if !ok {
		return sam2PreviewResult{}, errors.New("SAM2 模型无效")
	}
	if !isValidMediaPreviewAgentID(req.AgentID) || strings.TrimSpace(req.SourcePath) == "" {
		return sam2PreviewResult{}, errors.New("agentId 和视频文件不能为空")
	}
	if err := sam2ValidatePrompt(req.Prompt); err != nil {
		return sam2PreviewResult{}, err
	}
	if status, ff := checkFFmpegDependency(); status != http.StatusOK || !ff.Available {
		return sam2PreviewResult{}, errors.New("FFmpeg 或 FFprobe 未安装")
	}
	workspace, err := getWorkspaceByAgentID(m.cfg, req.AgentID)
	if err != nil {
		return sam2PreviewResult{}, errors.New("Agent 不存在")
	}
	source, _, status, message := resolveMediaPreviewFile(m.cfg, req.AgentID, normalizeQuotedPathArg(req.SourcePath))
	if status != 0 {
		return sam2PreviewResult{}, errors.New(message)
	}
	if err := validateRVMVideo(ctx, source); err != nil {
		return sam2PreviewResult{}, err
	}
	python, ok := sam2Python(workspace)
	if !ok {
		return sam2PreviewResult{}, errors.New("未检测到可用的 SAM2 Python 运行环境")
	}
	if !nonEmptyFile(sam2Checkpoint(workspace, model)) {
		return sam2PreviewResult{}, errors.New("未找到所选 SAM2 模型，请先安装或完成一次对应模型的任务")
	}
	if !nonEmptyFile(sam2ConfigPath(workspace, model)) {
		return sam2PreviewResult{}, errors.New("未找到 SAM2 模型配置：" + sam2ConfigPath(workspace, model))
	}
	if req.Prompt.MaskPath != "" {
		mask, err := sam2MaskPath(workspace, req.Prompt.MaskPath)
		if err != nil {
			return sam2PreviewResult{}, err
		}
		req.Prompt.MaskPath = mask
	}
	tmp, err := os.MkdirTemp("", "deepright-sam2-preview-")
	if err != nil {
		return sam2PreviewResult{}, err
	}
	defer os.RemoveAll(tmp)
	frames, masks := filepath.Join(tmp, "frames"), filepath.Join(tmp, "masks")
	if err := os.MkdirAll(frames, 0o755); err != nil {
		return sam2PreviewResult{}, err
	}
	ffmpeg, err := videoTrimLookPathFn("ffmpeg")
	if err != nil {
		return sam2PreviewResult{}, errors.New("未检测到 FFmpeg")
	}
	output, err := videoTrimCommandContextFn(ctx, ffmpeg, "-y", "-i", source, "-map", "0:v:0", "-frames:v", "1", "-q:v", "2", "-start_number", "0", filepath.Join(frames, "%06d.jpg")).CombinedOutput()
	if err != nil {
		return sam2PreviewResult{}, fmt.Errorf("提取首帧失败: %s", strings.TrimSpace(string(output)))
	}
	promptPath := filepath.Join(tmp, "prompt.json")
	body, _ := json.Marshal(req.Prompt)
	if err := os.WriteFile(promptPath, body, 0o600); err != nil {
		return sam2PreviewResult{}, err
	}
	var inferErr error
	for _, device := range rvmPreferredDevices() {
		_ = os.RemoveAll(masks)
		if err := os.MkdirAll(masks, 0o755); err != nil {
			return sam2PreviewResult{}, err
		}
		cmd := rvmCommandContext(ctx, python, "-c", sam2InferenceScript, "--frames", frames, "--checkpoint", sam2Checkpoint(workspace, model), "--config", sam2ConfigName(model), "--prompt", promptPath, "--masks", masks, "--device", device)
		cmd.Env = sam2CommandEnvironment(workspace)
		output, err = cmd.CombinedOutput()
		if err == nil {
			inferErr = nil
			break
		}
		if ctx.Err() != nil {
			return sam2PreviewResult{}, ctx.Err()
		}
		inferErr = fmt.Errorf("SAM2 %s 预览失败: %s", device, strings.TrimSpace(string(output)))
	}
	if inferErr != nil {
		return sam2PreviewResult{}, inferErr
	}
	mask, err := os.ReadFile(filepath.Join(masks, "mask_000000.png"))
	if err != nil || len(mask) == 0 {
		return sam2PreviewResult{}, errors.New("SAM2 未生成首帧掩码")
	}
	coverage, err := sam2MaskCoverage(filepath.Join(masks, "mask_000000.png"))
	if err != nil {
		return sam2PreviewResult{}, err
	}
	return sam2PreviewResult{MaskData: "data:image/png;base64," + base64.StdEncoding.EncodeToString(mask), Coverage: coverage}, nil
}

func sam2MaskCoverage(path string) (float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	decoded, err := png.Decode(f)
	if err != nil {
		return 0, fmt.Errorf("读取 SAM2 掩码失败: %w", err)
	}
	bounds := decoded.Bounds()
	total := bounds.Dx() * bounds.Dy()
	if total <= 0 {
		return 0, errors.New("SAM2 掩码尺寸无效")
	}
	foreground := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray := color.GrayModel.Convert(decoded.At(x, y)).(color.Gray)
			if gray.Y >= 128 {
				foreground++
			}
		}
	}
	return float64(foreground) / float64(total), nil
}
func (m *sam2TaskManager) allocate(agent, workspace, source string, except int64) (string, string, error) {
	stem := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
	for i := 0; i < 1000; i++ {
		suffix := ""
		if i > 0 {
			suffix = "_" + rvmNow().Format("20060102_150405")
			if i > 1 {
				suffix += "_" + strconv.Itoa(i)
			}
		}
		rel := filepath.ToSlash(filepath.Join("videos", stem+"_object"+suffix+".mov"))
		var count int
		if err := m.db.QueryRow(`SELECT COUNT(*) FROM sam2_task WHERE agent_id=? AND output_path=? AND id<>?`, agent, rel, except).Scan(&count); err != nil {
			return "", "", err
		}
		abs, err := resolveCaseInsensitiveUnderRoot(workspace, filepath.FromSlash(rel))
		if err != nil || count > 0 || !ensureWritablePathWithinRoot(workspace, abs) {
			continue
		}
		if _, err = os.Lstat(abs); err == nil {
			continue
		}
		if _, err = os.Lstat(rvmPreviewMP4Path(abs)); err == nil {
			continue
		}
		return rel, abs, nil
	}
	return "", "", errors.New("无法分配视频输出文件名")
}

const sam2InferenceScript = `
import argparse, gc, json, os, sys
import numpy as np
from PIL import Image
import torch
from sam2.build_sam import build_sam2, build_sam2_video_predictor
from sam2.automatic_mask_generator import SAM2AutomaticMaskGenerator

def load_mask(path, mode, width, height):
    image = Image.open(path)
    if image.size != (width, height): raise ValueError('掩码图尺寸必须与视频首帧一致')
    rgba = np.asarray(image.convert('RGBA'))
    if mode == 'transparent': return rgba[:, :, 3] < 128
    rgb = rgba[:, :, :3].astype(np.int16)
    color = {'green': np.array([0,255,0]), 'white': np.array([255,255,255]), 'black': np.array([0,0,0])}[mode]
    background = np.max(np.abs(rgb - color), axis=2) <= 48
    return ~background

p=argparse.ArgumentParser();p.add_argument('--frames',required=True);p.add_argument('--checkpoint',required=True);p.add_argument('--config',required=True);p.add_argument('--prompt',required=True);p.add_argument('--masks',required=True);p.add_argument('--device',required=True);a=p.parse_args()
prompt=json.load(open(a.prompt,encoding='utf-8'))
os.makedirs(a.masks,exist_ok=True)
device=a.device
points=prompt.get('points',[]); boxes=prompt.get('boxes',[])
auto_mask=None
if not points and not boxes and not prompt.get('maskPath'):
    first_name=sorted(os.listdir(a.frames), key=lambda n: int(os.path.splitext(n)[0]))[0]
    first_frame=np.asarray(Image.open(os.path.join(a.frames, first_name)).convert('RGB'))
    auto_model=build_sam2(a.config,a.checkpoint,device=device)
    generated=SAM2AutomaticMaskGenerator(auto_model,points_per_side=16,points_per_batch=32,pred_iou_thresh=0.75,stability_score_thresh=0.9,output_mode='binary_mask').generate(first_frame)
    height,width=first_frame.shape[:2]; total=height*width
    candidates=[item for item in generated if total*0.001 <= item['area'] <= total*0.9]
    if not candidates: candidates=generated
    if not candidates: raise ValueError('未能自动识别首帧主体，请添加点击点、框选区域或掩码图')
    chosen=max(candidates,key=lambda item:item['area']*((item.get('predicted_iou',0.0)+item.get('stability_score',0.0))/2.0))
    auto_mask=np.asarray(chosen['segmentation'],dtype=bool)
    print('SAM2 自动选择首帧最显著主体（候选 %d 个，面积 %d 像素）。'%(len(generated),int(chosen['area'])),flush=True)
    del generated,auto_model
    gc.collect()
    if device=='mps': torch.mps.empty_cache()
predictor=build_sam2_video_predictor(a.config,a.checkpoint,device=device,apply_postprocessing=True)
# Keep the MPS memory profile aligned with the validated successful workflow.
state=predictor.init_state(video_path=a.frames,offload_video_to_cpu=True,offload_state_to_cpu=True)
if points or boxes:
    coords=np.array([[x['x'],x['y']] for x in points],dtype=np.float32) if points else None
    labels=np.array([1 if x.get('foreground',True) else 0 for x in points],dtype=np.int32) if points else None
    box=None
    if boxes:
        b=boxes[0];box=np.array([b['xMin'],b['yMin'],b['xMax'],b['yMax']],dtype=np.float32)
    predictor.add_new_points_or_box(state,frame_idx=0,obj_id=1,points=coords,labels=labels,box=box)
if prompt.get('maskPath'):
    first_frame=Image.open(os.path.join(a.frames, sorted(os.listdir(a.frames), key=lambda n: int(os.path.splitext(n)[0]))[0]))
    w,h=first_frame.size
    mask=load_mask(prompt['maskPath'],prompt['maskMode'],w,h)
    if not np.any(mask): raise ValueError('掩码图没有目标区域')
    predictor.add_new_mask(state,frame_idx=0,obj_id=1,mask=mask)
if auto_mask is not None:
    predictor.add_new_mask(state,frame_idx=0,obj_id=1,mask=auto_mask)
for frame_idx,obj_ids,mask_logits in predictor.propagate_in_video(state):
    mask=(mask_logits[0,0] > 0).detach().cpu().numpy().astype(np.uint8)*255
    Image.fromarray(mask).save(os.path.join(a.masks,'mask_%06d.png'%frame_idx))
    print('SAM2 frame %d' % frame_idx,flush=True)
`

func (m *sam2TaskManager) extract(ctx context.Context, task sam2Task) error {
	if status, ff := checkFFmpegDependency(); status != http.StatusOK || !ff.Available {
		return errors.New("FFmpeg 或 FFprobe 未安装")
	}
	workspace, err := getWorkspaceByAgentID(m.cfg, task.AgentID)
	if err != nil {
		return errors.New("Agent 工作目录不存在")
	}
	source, _, status, msg := resolveMediaPreviewFile(m.cfg, task.AgentID, task.SourcePath)
	if status != 0 {
		return errors.New(msg)
	}
	model, _ := sam2ModelFor(task.Model)
	python, ok := sam2Python(workspace)
	if !ok {
		return errors.New("未检测到可用的 SAM2 Python 运行环境")
	}
	if err := m.ensureModel(ctx, task.ID, workspace, model); err != nil {
		return err
	}
	config := sam2ConfigPath(workspace, model)
	if !nonEmptyFile(config) {
		return errors.New("未找到 SAM2 模型配置：" + config)
	}
	target := task.SavedAs
	if target == "" {
		target = filepath.Join(workspace, filepath.FromSlash(task.OutputPath))
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(filepath.Dir(target), ".sam2-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	prompt := task.Prompt
	if prompt.MaskPath != "" {
		prompt.MaskPath = filepath.Join(workspace, filepath.FromSlash(prompt.MaskPath))
	}
	promptPath := filepath.Join(tmp, "prompt.json")
	body, _ := json.Marshal(prompt)
	if err := os.WriteFile(promptPath, body, 0o600); err != nil {
		return err
	}
	masks := filepath.Join(tmp, "masks")
	frames := filepath.Join(tmp, "frames")
	if err := os.MkdirAll(frames, 0o755); err != nil {
		return err
	}
	ffmpeg, err := videoTrimLookPathFn("ffmpeg")
	if err != nil {
		return errors.New("未检测到 FFmpeg")
	}
	m.appendLog(task.ID, "正在使用 FFmpeg 提取视频帧供 SAM2 分割（无需 decord）。")
	frameOutput, err := videoTrimCommandContextFn(ctx, ffmpeg, "-y", "-i", source, "-map", "0:v:0", "-vsync", "0", "-q:v", "2", "-start_number", "0", filepath.Join(frames, "%06d.jpg")).CombinedOutput()
	if err != nil {
		return fmt.Errorf("提取 SAM2 视频帧失败: %s", strings.TrimSpace(string(frameOutput)))
	}
	frameFiles, err := filepath.Glob(filepath.Join(frames, "*.jpg"))
	if err != nil || len(frameFiles) == 0 {
		return errors.New("FFmpeg 未提取到可供 SAM2 使用的视频帧")
	}
	m.appendLog(task.ID, fmt.Sprintf("已提取 %d 帧，开始 SAM2 分割。", len(frameFiles)))
	m.progress(task.ID, 8)
	m.appendLog(task.ID, "SAM2 模型："+model.Name+"；权重："+sam2Checkpoint(workspace, model))
	m.appendLog(task.ID, "SAM2 配置："+config)
	var inferErr error
	for _, device := range rvmPreferredDevices() {
		if err := os.RemoveAll(masks); err != nil {
			return err
		}
		if err := os.MkdirAll(masks, 0o755); err != nil {
			return err
		}
		m.appendLog(task.ID, "正在使用 "+device+" 执行 SAM2 视频分割（模型推理优先使用 GPU；仅将缓存帧与状态卸载到内存以降低显存压力）。")
		cmd := rvmCommandContext(ctx, python, "-c", sam2InferenceScript, "--frames", frames, "--checkpoint", sam2Checkpoint(workspace, model), "--config", sam2ConfigName(model), "--prompt", promptPath, "--masks", masks, "--device", device)
		cmd.Env = sam2CommandEnvironment(workspace)
		out, err := m.runInference(cmd, task.ID, len(frameFiles))
		if len(out) > 0 {
			m.appendLog(task.ID, string(out))
		}
		if err == nil {
			inferErr = nil
			break
		}
		if ctx.Err() != nil {
			return context.Canceled
		}
		inferErr = err
		m.appendLog(task.ID, "SAM2 "+device+" 推理失败："+err.Error())
	}
	if inferErr != nil {
		return inferErr
	}
	coverage, err := sam2ValidateMaskSeries(masks, len(frameFiles))
	if err != nil {
		return err
	}
	m.progress(task.ID, 92)
	m.appendLog(task.ID, fmt.Sprintf("SAM2 分割完成；首帧/中帧/末帧主体覆盖率：%.2f%% / %.2f%% / %.2f%%。", coverage[0]*100, coverage[1]*100, coverage[2]*100))
	m.appendLog(task.ID, "正在合成带原音频的透明 ProRes 4444 MOV。")
	frameRate, err := sam2VideoFrameRate(ctx, source)
	if err != nil {
		return err
	}
	args := []string{"-n", "-i", source, "-framerate", frameRate, "-start_number", "0", "-i", filepath.Join(masks, "mask_%06d.png"), "-filter_complex", "[0:v][1:v]alphamerge,format=yuva444p10le[video]", "-map", "[video]", "-map", "0:a?", "-c:v", "prores_ks", "-profile:v", "4", "-pix_fmt", "yuva444p10le", "-c:a", "aac", "-movflags", "+faststart", "-shortest", target}
	output, err := videoTrimCommandContextFn(ctx, ffmpeg, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("合成透明 MOV 失败: %s", strings.TrimSpace(string(output)))
	}
	if !nonEmptyFile(target) {
		return errors.New("未生成有效透明 MOV")
	}
	m.progress(task.ID, 96)
	m.appendLog(task.ID, "透明 ProRes 4444 MOV 已生成，正在生成黑底 MP4 预览。")
	preview := rvmPreviewMP4Path(target)
	if err := m.preview(ctx, ffmpeg, target, preview); err != nil {
		return err
	}
	m.progress(task.ID, 100)
	m.appendLog(task.ID, "最终输出透明 ProRes 4444 MOV（含原音频）："+target)
	m.appendLog(task.ID, "最终输出黑底 MP4 预览："+preview)
	return nil
}

// sam2ValidateMaskSeries catches the two most damaging prompt failures before
// packaging: selecting nearly all background, or losing the subject entirely.
func sam2ValidateMaskSeries(masks string, frameTotal int) ([3]float64, error) {
	var coverage [3]float64
	if frameTotal < 1 {
		return coverage, errors.New("SAM2 没有可验证的掩码帧")
	}
	indexes := [3]int{0, frameTotal / 2, frameTotal - 1}
	for i, index := range indexes {
		value, err := sam2MaskCoverage(filepath.Join(masks, fmt.Sprintf("mask_%06d.png", index)))
		if err != nil {
			return coverage, fmt.Errorf("读取 SAM2 抽样掩码失败: %w", err)
		}
		coverage[i] = value
	}
	for _, value := range coverage {
		if value < 0.0005 {
			return coverage, errors.New("SAM2 在抽样帧中丢失主体；请调整首帧标注后重试")
		}
		if value > 0.85 {
			return coverage, errors.New("SAM2 在抽样帧中选择了过大的区域，可能是背景；请调整首帧标注后重试")
		}
	}
	return coverage, nil
}

// runInference streams SAM2 stdout. The inference script emits one line per
// completed frame, so task progress and milestone logs remain visible while a
// long video is still being segmented instead of appearing only after exit.
func (m *sam2TaskManager) runInference(cmd *exec.Cmd, taskID int64, frameTotal int) (string, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}
	var wg sync.WaitGroup
	var output strings.Builder
	var outputMu sync.Mutex
	appendOutput := func(line string) {
		outputMu.Lock()
		if output.Len() < rvmTaskLogLimit {
			output.WriteString(line)
			output.WriteByte('\n')
		}
		outputMu.Unlock()
	}
	wg.Add(2)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 32*1024), 1024*1024)
		lastMilestone := -1
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			if frame, ok := sam2OutputFrame(line); ok && frameTotal > 0 {
				completed := frame + 1
				progress := 8 + completed*84/frameTotal
				if progress > 92 {
					progress = 92
				}
				m.progress(taskID, progress)
				milestone := progress / 5 * 5
				if milestone > lastMilestone || completed == frameTotal {
					m.appendLog(taskID, fmt.Sprintf("SAM2 分割进度：%d%%（%d/%d 帧）。", progress, completed, frameTotal))
					lastMilestone = milestone
				}
				continue
			}
			appendOutput(line)
			m.appendLog(taskID, line)
		}
	}()
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		scanner.Buffer(make([]byte, 32*1024), 1024*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				appendOutput(line)
			}
		}
	}()
	err = cmd.Wait()
	wg.Wait()
	outputMu.Lock()
	result := output.String()
	outputMu.Unlock()
	return result, err
}

func sam2OutputFrame(line string) (int, bool) {
	var frame int
	if _, err := fmt.Sscanf(strings.TrimSpace(line), "SAM2 frame %d", &frame); err == nil && frame >= 0 {
		return frame, true
	}
	return 0, false
}

func sam2VideoFrameRate(ctx context.Context, source string) (string, error) {
	ffprobe, err := videoTrimLookPathFn("ffprobe")
	if err != nil {
		return "", errors.New("未检测到 FFprobe")
	}
	output, err := videoTrimCommandContextFn(ctx, ffprobe, "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=r_frame_rate", "-of", "default=noprint_wrappers=1:nokey=1", source).CombinedOutput()
	value := strings.TrimSpace(string(output))
	if err != nil || value == "" || !isSafeSAM2FrameRate(value) {
		return "", errors.New("无法读取视频帧率")
	}
	return value, nil
}

func isSafeSAM2FrameRate(value string) bool {
	parts := strings.Split(value, "/")
	if len(parts) > 2 || len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		if _, err := strconv.ParseUint(part, 10, 64); err != nil {
			return false
		}
	}
	return value != "0" && value != "0/0"
}
func (m *sam2TaskManager) ensureModel(ctx context.Context, id int64, workspace string, model sam2Model) error {
	path := sam2Checkpoint(workspace, model)
	if nonEmptyFile(path) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	part := path + ".deepright.part"
	m.appendLog(id, "未检测到 "+model.Name+"，正在下载："+path)
	_, err := downloadModelWithFallback(ctx, resumableModelDownloadConfig{Client: rvmHTTPClient, URL: model.OfficialURL, BackupURL: model.BackupURL, PartPath: part, SourceID: "meta:sam2", Revision: "072824", ArtifactPath: model.Filename, BackupSourceID: "hf-mirror:facebook/sam2", BackupRevision: "main", BackupArtifactPath: model.Filename, IdleTimeout: modelTaskDownloadReadTimeout(m.cfg), Workers: modelTaskDownloadWorkers(m.cfg), Retries: modelTaskDownloadRetries(m.cfg), Progress: func(done, total int64) {
		if total > 0 {
			m.appendLog(id, fmt.Sprintf("%s 下载 %d%%", model.Name, done*100/total))
		}
	}, Retry: func(worker, partNo, parts, retry, retries int, e error) {
		m.appendLog(id, model.Name+modelDownloadRetryMessage(worker, partNo, parts, retry, retries, e))
	}}, func(msg string) { m.appendLog(id, model.Name+"："+msg) })
	if err != nil {
		return fmt.Errorf("%s 下载失败: %w", model.Name, err)
	}
	if !nonEmptyFile(part) {
		return errors.New(model.Name + " 下载文件为空")
	}
	if err := os.Rename(part, path); err != nil {
		return err
	}
	removeModelDownloadState(part)
	m.appendLog(id, model.Name+" 下载完成。")
	return nil
}
func (m *sam2TaskManager) preview(ctx context.Context, ffmpeg, source, preview string) error {
	output, err := videoTrimCommandContextFn(ctx, ffmpeg, rvmPreviewFFmpegArgs(source, preview)...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("生成 MP4 预览失败: %s", strings.TrimSpace(string(output)))
	}
	if !nonEmptyFile(preview) {
		return errors.New("未生成有效 MP4 预览")
	}
	return nil
}
func (m *sam2TaskManager) progress(id int64, p int) {
	_, _ = m.db.Exec(`UPDATE sam2_task SET progress=CASE WHEN progress>? THEN progress ELSE ? END,updated_at=? WHERE id=?`, p, p, rvmTimestamp(), id)
}
func (m *sam2TaskManager) appendLog(id int64, message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	if len(message) > 4096 {
		message = message[:4096] + "…"
	}
	_, _ = m.db.Exec(`UPDATE sam2_task SET logs=CASE WHEN length(logs)>? THEN substr(logs,-?) ELSE logs END || ?,updated_at=? WHERE id=?`, rvmTaskLogLimit-8192, rvmTaskLogLimit/2, "["+rvmTimestamp()+"] "+message+"\n", rvmTimestamp(), id)
}
func (m *sam2TaskManager) finish(id int64, status string, p int, message string) {
	m.appendLog(id, message)
	_, _ = m.db.Exec(`UPDATE sam2_task SET status=?,progress=?,updated_at=? WHERE id=?`, status, p, rvmTimestamp(), id)
	notifyIntegrationSleepConditionChanged(m.cfg)
}
func (m *sam2TaskManager) cancelled(id int64) bool {
	var v int
	return m.db.QueryRow(`SELECT cancel_requested FROM sam2_task WHERE id=?`, id).Scan(&v) != nil || v != 0
}
func (m *sam2TaskManager) cancelRunning() {
	m.mu.Lock()
	c := m.cancel
	m.mu.Unlock()
	if c != nil {
		c()
	}
}

func validSAM2Status(v string) bool {
	switch v {
	case "", rvmTaskQueued, rvmTaskRunning, rvmTaskCompleted, rvmTaskCancelled, rvmTaskFailed:
		return true
	}
	return false
}
func (m *sam2TaskManager) list(agent, status string, page int, logs bool) (sam2Page, error) {
	var result sam2Page
	agent = strings.TrimSpace(agent)
	if !isValidMediaPreviewAgentID(agent) || !validSAM2Status(status) {
		return result, errors.New("任务参数无效")
	}
	result.PageSize = 5
	if page < 1 {
		page = 1
	}
	where, args := "agent_id=?", []interface{}{agent}
	if status != "" {
		where += " AND status=?"
		args = append(args, status)
	}
	if err := m.db.QueryRow(`SELECT COUNT(*) FROM sam2_task WHERE `+where, args...).Scan(&result.Total); err != nil {
		return result, err
	}
	last := (result.Total + 4) / 5
	if last < 1 {
		last = 1
	}
	if page > last {
		page = last
	}
	column := "''"
	if logs {
		column = "logs"
	}
	args = append(args, 5, (page-1)*5)
	rows, err := m.db.Query(`SELECT id,agent_id,model,source_path,prompt_json,output_path,saved_as,status,progress,started_at,created_at,updated_at,`+column+`,cancel_requested FROM sam2_task WHERE `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		t, e := scanSAM2Task(rows)
		if e != nil {
			return result, e
		}
		result.Tasks = append(result.Tasks, t)
	}
	result.Page, result.Status = page, status
	return result, rows.Err()
}
func scanSAM2Task(s interface{ Scan(...interface{}) error }) (sam2Task, error) {
	var t sam2Task
	var prompt string
	var cancel int
	err := s.Scan(&t.ID, &t.AgentID, &t.Model, &t.SourcePath, &prompt, &t.OutputPath, &t.SavedAs, &t.Status, &t.Progress, &t.StartedAt, &t.CreatedAt, &t.UpdatedAt, &t.Logs, &cancel)
	if err == nil {
		_ = json.Unmarshal([]byte(prompt), &t.Prompt)
		t.ModelName = sam2ModelName(t.Model)
		t.CancelAsked = cancel != 0
	}
	return t, err
}
func (m *sam2TaskManager) log(agent string, id int64) (sam2Task, error) {
	return scanSAM2Task(m.db.QueryRow(`SELECT id,agent_id,model,source_path,prompt_json,output_path,saved_as,status,progress,started_at,created_at,updated_at,logs,cancel_requested FROM sam2_task WHERE id=? AND agent_id=?`, id, strings.TrimSpace(agent)))
}
func (m *sam2TaskManager) cancelTask(agent string, id int64) error {
	now := rvmTimestamp()
	res, err := m.db.Exec(`UPDATE sam2_task SET cancel_requested=1,status=CASE WHEN status=? THEN ? ELSE status END,updated_at=?,logs=logs || ? WHERE id=? AND agent_id=? AND status IN (?,?)`, rvmTaskQueued, rvmTaskCancelled, now, "["+now+"] 已请求取消任务。\n", id, strings.TrimSpace(agent), rvmTaskQueued, rvmTaskRunning)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return errors.New("仅可取消排队中或执行中的任务")
	}
	m.mu.Lock()
	running, c := m.runningID == id, m.cancel
	m.mu.Unlock()
	if running && c != nil {
		c()
	}
	m.signal()
	return nil
}
func (m *sam2TaskManager) restart(agent string, id int64) error {
	var source string
	if err := m.db.QueryRow(`SELECT source_path FROM sam2_task WHERE id=? AND agent_id=? AND status IN (?,?)`, id, strings.TrimSpace(agent), rvmTaskFailed, rvmTaskCancelled).Scan(&source); err != nil {
		return errors.New("仅可重新开始失败或已取消任务")
	}
	workspace, err := getWorkspaceByAgentID(m.cfg, strings.TrimSpace(agent))
	if err != nil {
		return errors.New("Agent 不存在")
	}
	out, saved, err := m.allocate(agent, workspace, source, id)
	if err != nil {
		return err
	}
	_, err = m.db.Exec(`UPDATE sam2_task SET output_path=?,saved_as=?,status=?,progress=0,started_at='',cancel_requested=0,updated_at=?,logs='' WHERE id=? AND agent_id=?`, out, saved, rvmTaskQueued, rvmTimestamp(), id, strings.TrimSpace(agent))
	if err == nil {
		m.signal()
	}
	return err
}
func (m *sam2TaskManager) deleteTask(agent string, id int64) error {
	res, err := m.db.Exec(`DELETE FROM sam2_task WHERE id=? AND agent_id=? AND status IN (?,?)`, id, strings.TrimSpace(agent), rvmTaskFailed, rvmTaskCancelled)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return errors.New("仅可删除失败或已取消任务")
	}
	return nil
}

func readSAM2Action(w http.ResponseWriter, r *http.Request) (sam2ActionRequest, error) {
	var req sam2ActionRequest
	if r.Method != http.MethodPost {
		return req, errors.New("仅支持 POST 请求")
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	d.DisallowUnknownFields()
	if err := d.Decode(&req); err != nil {
		return req, errors.New("请求内容无效")
	}
	return req, nil
}
func handleSAM2Tasks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if sam2Tasks == nil {
			writeSAM2JSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": 1, "content": "视频提取物体任务服务未就绪"})
			return
		}
		if r.Method == http.MethodGet {
			page, _ := strconv.Atoi(r.URL.Query().Get("page"))
			if page < 1 {
				page = 1
			}
			v, err := sam2Tasks.list(r.URL.Query().Get("agentId"), r.URL.Query().Get("status"), page, false)
			if err != nil {
				writeSAM2JSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
				return
			}
			writeSAM2JSON(w, http.StatusOK, map[string]interface{}{"status": 0, "tasks": v.Tasks, "total": v.Total, "page": v.Page, "pageSize": v.PageSize, "taskStatus": v.Status})
			return
		}
		if r.Method == http.MethodPost {
			var req sam2TaskRequest
			d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
			d.DisallowUnknownFields()
			if err := d.Decode(&req); err != nil {
				writeSAM2JSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "请求内容无效"})
				return
			}
			v, err := sam2Tasks.create(req)
			if err != nil {
				writeSAM2JSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
				return
			}
			writeSAM2JSON(w, http.StatusOK, map[string]interface{}{"status": 0, "task": v})
			return
		}
		w.Header().Set("Allow", "GET, POST")
		writeSAM2JSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"status": 1, "content": "仅支持 GET 或 POST 请求"})
	}
}
func handleSAM2Preview() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeSAM2JSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"status": 1, "content": "仅支持 POST 请求"})
			return
		}
		if sam2Tasks == nil {
			writeSAM2JSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": 1, "content": "视频提取物体任务服务未就绪"})
			return
		}
		var req sam2PreviewRequest
		d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
		d.DisallowUnknownFields()
		if err := d.Decode(&req); err != nil {
			writeSAM2JSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "请求内容无效"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()
		preview, err := sam2Tasks.previewFirstFrame(ctx, req)
		if err != nil {
			writeSAM2JSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		writeSAM2JSON(w, http.StatusOK, map[string]interface{}{"status": 0, "maskData": preview.MaskData, "coverage": preview.Coverage})
	}
}
func sam2Action(fn func(*sam2TaskManager, string, int64) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if sam2Tasks == nil {
			writeSAM2JSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": 1})
			return
		}
		req, err := readSAM2Action(w, r)
		if err == nil {
			err = fn(sam2Tasks, req.AgentID, req.ID)
		}
		if err != nil {
			writeSAM2JSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		writeSAM2JSON(w, http.StatusOK, map[string]interface{}{"status": 0})
	}
}
func handleSAM2TaskCancel() http.HandlerFunc {
	return sam2Action(func(m *sam2TaskManager, a string, id int64) error { return m.cancelTask(a, id) })
}
func handleSAM2TaskRestart() http.HandlerFunc {
	return sam2Action(func(m *sam2TaskManager, a string, id int64) error { return m.restart(a, id) })
}
func handleSAM2TaskDelete() http.HandlerFunc {
	return sam2Action(func(m *sam2TaskManager, a string, id int64) error { return m.deleteTask(a, id) })
}
func handleSAM2TaskLog() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || sam2Tasks == nil {
			writeSAM2JSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "任务参数无效"})
			return
		}
		id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
		if err != nil {
			writeSAM2JSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "任务参数无效"})
			return
		}
		task, err := sam2Tasks.log(r.URL.Query().Get("agentId"), id)
		if err != nil {
			writeSAM2JSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "任务参数无效"})
			return
		}
		writeSAM2JSON(w, http.StatusOK, map[string]interface{}{"status": 0, "task": task})
	}
}
