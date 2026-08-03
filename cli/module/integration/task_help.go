package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	taskHelpTypeVoxCPM  = "voxcpm"
	taskHelpTypeWhisper = "whisper"
	taskHelpTypeRembg   = "rembg"
	taskHelpTypeWav2Lip = "wav2lip"
	taskHelpTypeRVM     = "rvm"
)

var taskHelpNow = time.Now

type taskHelpRequest struct {
	AgentID  string `json:"agentId"`
	TaskType string `json:"taskType"`
	ID       int64  `json:"id"`
}

func taskHelpFunctionName(taskType string) (string, bool) {
	switch strings.TrimSpace(taskType) {
	case taskHelpTypeVoxCPM:
		return "文字转语音", true
	case taskHelpTypeWhisper:
		return "提取音频文字", true
	case taskHelpTypeRembg:
		return "提取图片主体", true
	case taskHelpTypeWav2Lip:
		return "人物视频对口型", true
	case taskHelpTypeRVM:
		return "视频提取主体", true
	default:
		return "", false
	}
}

func taskHelpLog(agentID, taskType string, taskID int64) (string, error) {
	agentID = strings.TrimSpace(agentID)
	if !isValidMediaPreviewAgentID(agentID) || taskID <= 0 {
		return "", errors.New("任务参数无效")
	}
	switch strings.TrimSpace(taskType) {
	case taskHelpTypeVoxCPM:
		if voxcpmTasks == nil {
			return "", errors.New("文字转语音任务服务未就绪")
		}
		task, err := voxcpmTasks.log(agentID, taskID)
		return task.Logs, err
	case taskHelpTypeWhisper:
		if whisperTasks == nil {
			return "", errors.New("音频文字提取任务服务未就绪")
		}
		task, err := whisperTasks.taskLog(agentID, taskID)
		return task.Logs, err
	case taskHelpTypeRembg:
		if rembgTasks == nil {
			return "", errors.New("图片主体任务服务未就绪")
		}
		task, err := rembgTasks.log(agentID, taskID)
		return task.Logs, err
	case taskHelpTypeWav2Lip:
		if wav2lipTasks == nil {
			return "", errors.New("人物视频对口型任务服务未就绪")
		}
		task, err := wav2lipTasks.log(agentID, taskID)
		return task.Logs, err
	case taskHelpTypeRVM:
		if rvmTasks == nil {
			return "", errors.New("视频主体任务服务未就绪")
		}
		task, err := rvmTasks.log(agentID, taskID)
		return task.Logs, err
	default:
		return "", errors.New("任务类型无效")
	}
}

func taskHelpTemplate(raw map[string]interface{}) (string, error) {
	value, ok := raw["seekHelp"].(string)
	if !ok || strings.TrimSpace(value) == "" || !strings.Contains(value, "$function") || !strings.Contains(value, "$log") {
		return "", errors.New("config/config.json.seekHelp 必须是包含 $function 和 $log 的非空字符串")
	}
	return value, nil
}

func writeTaskHelpLogFile(workspace, functionName, logs string) (string, error) {
	workspace, err := filepath.Abs(strings.TrimSpace(workspace))
	if err != nil || strings.TrimSpace(workspace) == "" {
		return "", errors.New("无法确定 Agent 工作目录")
	}
	directory := filepath.Join(workspace, "tmp")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", fmt.Errorf("无法创建日志目录: %w", err)
	}
	if !ensureWritablePathWithinRoot(workspace, directory) {
		return "", errors.New("日志目录不在 Agent 工作目录内")
	}
	base := functionName + "_" + taskHelpNow().Format("20060102_150405")
	for number := 0; number < 1000; number++ {
		name := base + ".log"
		if number > 0 {
			name = fmt.Sprintf("%s_%d.log", base, number+1)
		}
		path := filepath.Join(directory, name)
		if !ensureWritablePathWithinRoot(workspace, path) {
			return "", errors.New("日志文件路径不在 Agent 工作目录内")
		}
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("无法创建日志文件: %w", err)
		}
		if _, err := io.WriteString(file, logs); err != nil {
			_ = file.Close()
			_ = os.Remove(path)
			return "", fmt.Errorf("无法写入日志文件: %w", err)
		}
		if err := file.Close(); err != nil {
			_ = os.Remove(path)
			return "", fmt.Errorf("无法保存日志文件: %w", err)
		}
		return path, nil
	}
	return "", errors.New("无法分配唯一日志文件名")
}

func buildTaskHelpPrompt(workspace, taskType, logs string) (string, error) {
	functionName, ok := taskHelpFunctionName(taskType)
	if !ok {
		return "", errors.New("任务类型无效")
	}
	raw, _, err := readIntegrationStartupConfigRaw()
	if err != nil {
		return "", fmt.Errorf("读取 config/config.json 失败: %w", err)
	}
	template, err := taskHelpTemplate(raw)
	if err != nil {
		return "", err
	}
	logPath, err := writeTaskHelpLogFile(workspace, functionName, logs)
	if err != nil {
		return "", err
	}
	prompt := strings.ReplaceAll(template, "$function", functionName)
	prompt = strings.ReplaceAll(prompt, "$log", "[FILE:"+logPath+"]")
	return prompt, nil
}

func writeTaskHelpJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func handleTaskHelp(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeTaskHelpJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"status": 1, "content": "仅支持 POST 请求"})
			return
		}
		var request taskHelpRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
			writeTaskHelpJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "请求体格式错误"})
			return
		}
		functionName, ok := taskHelpFunctionName(request.TaskType)
		if !ok || !isValidMediaPreviewAgentID(strings.TrimSpace(request.AgentID)) || request.ID <= 0 {
			writeTaskHelpJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "任务参数无效"})
			return
		}
		logs, err := taskHelpLog(request.AgentID, request.TaskType, request.ID)
		if err != nil {
			writeTaskHelpJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		workspace, err := getWorkspaceByAgentID(cfg, strings.TrimSpace(request.AgentID))
		if err != nil {
			writeTaskHelpJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "Agent 工作目录不存在"})
			return
		}
		prompt, err := buildTaskHelpPrompt(workspace, request.TaskType, logs)
		if err != nil {
			writeTaskHelpJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		writeTaskHelpJSON(w, http.StatusOK, map[string]interface{}{"status": 0, "content": prompt, "function": functionName})
	}
}
