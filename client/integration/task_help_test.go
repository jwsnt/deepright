package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestTaskHelpFunctionNames(t *testing.T) {
	for taskType, want := range map[string]string{
		taskHelpTypeVoxCPM:  "文字转语音",
		taskHelpTypeWhisper: "提取音频文字",
		taskHelpTypeRembg:   "提取图片主体",
		taskHelpTypeWav2Lip: "人物视频对口型",
		taskHelpTypeRVM:     "视频提取主体",
	} {
		if got, ok := taskHelpFunctionName(taskType); !ok || got != want {
			t.Fatalf("taskHelpFunctionName(%q) = %q, %v; want %q, true", taskType, got, ok, want)
		}
	}
	if _, ok := taskHelpFunctionName("untrusted"); ok {
		t.Fatal("unknown task type must be rejected")
	}
}

func TestTaskHelpTemplateRequiresBothPlaceholders(t *testing.T) {
	if _, err := taskHelpTemplate(map[string]interface{}{"seekHelp": "我在使用 $function：$log"}); err != nil {
		t.Fatalf("valid seekHelp template: %v", err)
	}
	for _, value := range []interface{}{"", "only $function", "only $log", 1} {
		if _, err := taskHelpTemplate(map[string]interface{}{"seekHelp": value}); err == nil {
			t.Fatalf("invalid seekHelp template %v was accepted", value)
		}
	}
}

func TestWriteTaskHelpLogFileUsesTimestampAndDoesNotOverwrite(t *testing.T) {
	workspace := t.TempDir()
	originalNow := taskHelpNow
	taskHelpNow = func() time.Time { return time.Date(2026, 8, 3, 14, 30, 25, 0, time.Local) }
	defer func() { taskHelpNow = originalNow }()

	first, err := writeTaskHelpLogFile(workspace, "提取图片主体", "first log")
	if err != nil {
		t.Fatalf("write first log: %v", err)
	}
	second, err := writeTaskHelpLogFile(workspace, "提取图片主体", "second log")
	if err != nil {
		t.Fatalf("write second log: %v", err)
	}
	if got, want := filepath.Base(first), "提取图片主体_20260803_143025.log"; got != want {
		t.Fatalf("first filename = %q, want %q", got, want)
	}
	if got, want := filepath.Base(second), "提取图片主体_20260803_143025_2.log"; got != want {
		t.Fatalf("second filename = %q, want %q", got, want)
	}
	if !strings.HasPrefix(first, filepath.Join(workspace, "tmp")+string(os.PathSeparator)) || !strings.HasPrefix(second, filepath.Join(workspace, "tmp")+string(os.PathSeparator)) {
		t.Fatalf("logs must be written below workspace tmp: %q %q", first, second)
	}
	if data, err := os.ReadFile(first); err != nil || string(data) != "first log" {
		t.Fatalf("first log = %q, %v", data, err)
	}
	if data, err := os.ReadFile(second); err != nil || string(data) != "second log" {
		t.Fatalf("second log = %q, %v", data, err)
	}
}

func TestHandleTaskHelpWritesCurrentTaskLogAndReturnsFileBubble(t *testing.T) {
	root := t.TempDir()
	agentRoot := filepath.Join(root, "agents")
	workspace := filepath.Join(agentRoot, "agent-a")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "config.json"), []byte(`{"seekHelp":"我正在使用 $function，请解释：$log"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	originalExecutable := integrationExecutableFn
	integrationExecutableFn = func() (string, error) { return filepath.Join(root, "integration"), nil }
	defer func() { integrationExecutableFn = originalExecutable }()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ensureWhisperTaskSchema(db); err != nil {
		t.Fatal(err)
	}
	result, err := db.Exec(`INSERT INTO whisper_task(agent_id,source_path,output_path,status,logs,created_at,updated_at) VALUES(?,?,?,?,?,'now','now')`, "agent-a", "tmp/source.wav", "whisper/source.txt", whisperTaskStatusFailed, "current task log")
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	originalTasks := whisperTasks
	whisperTasks = newWhisperTaskManager(&Config{AgentDir: agentRoot}, db)
	defer func() { whisperTasks = originalTasks }()
	originalNow := taskHelpNow
	taskHelpNow = func() time.Time { return time.Date(2026, 8, 3, 14, 30, 25, 0, time.Local) }
	defer func() { taskHelpNow = originalNow }()

	request := httptest.NewRequest(http.MethodPost, "/api/task_help", strings.NewReader(`{"agentId":"agent-a","taskType":"whisper","id":`+strconv.FormatInt(id, 10)+`}`))
	recorder := httptest.NewRecorder()
	handleTaskHelp(&Config{AgentDir: agentRoot})(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Status  int    `json:"status"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(workspace, "tmp", "提取音频文字_20260803_143025.log")
	if response.Status != 0 || !strings.Contains(response.Content, "我正在使用 提取音频文字") || !strings.Contains(response.Content, "[FILE:"+logPath+"]") {
		t.Fatalf("response = %#v", response)
	}
	if data, err := os.ReadFile(logPath); err != nil || string(data) != "current task log" {
		t.Fatalf("saved log = %q, %v", data, err)
	}
}
