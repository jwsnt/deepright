package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestParseWhisperCheckConfig(t *testing.T) {
	config, err := parseWhisperCheckConfig(map[string]interface{}{
		"whisper": map[string]interface{}{"check": float64(12), "install": "请安装 openai-whisper"},
	})
	if err != nil {
		t.Fatalf("parse valid config: %v", err)
	}
	if config.CacheFor.Hours() != 12 || config.Install != "请安装 openai-whisper" {
		t.Fatalf("unexpected config: %#v", config)
	}
	if _, err := parseWhisperCheckConfig(map[string]interface{}{"whisper": map[string]interface{}{"check": 0, "install": "x"}}); err == nil {
		t.Fatal("zero check must be rejected")
	}
	if _, err := parseWhisperCheckConfig(map[string]interface{}{"whisper": map[string]interface{}{"check": 1}}); err == nil {
		t.Fatal("missing install request must be rejected")
	}
}

func TestWhisperDependencyAvailableRequiresRunnableCommand(t *testing.T) {
	root := t.TempDir()
	whisper := filepath.Join(root, "whisper-test")
	if err := os.WriteFile(whisper, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write unavailable whisper command: %v", err)
	}
	originalLookPath := whisperTaskLookPath
	originalHomeDir := whisperTaskUserHomeDir
	whisperTaskLookPath = func(name string) (string, error) {
		if name == "whisper" {
			return whisper, nil
		}
		return originalLookPath(name)
	}
	whisperTaskUserHomeDir = func() (string, error) { return root, nil }
	defer func() {
		whisperTaskLookPath = originalLookPath
		whisperTaskUserHomeDir = originalHomeDir
	}()
	whisperCheckCache.Lock()
	originalAvailableAt := whisperCheckCache.availableAt
	originalExecutable := whisperCheckCache.executable
	whisperCheckCache.availableAt = time.Time{}
	whisperCheckCache.executable = ""
	whisperCheckCache.Unlock()
	defer func() {
		whisperCheckCache.Lock()
		whisperCheckCache.availableAt = originalAvailableAt
		whisperCheckCache.executable = originalExecutable
		whisperCheckCache.Unlock()
	}()
	if whisperDependencyAvailable(time.Hour) {
		t.Fatal("non-runnable whisper command must not be treated as available")
	}
	if err := os.WriteFile(whisper, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write runnable whisper command: %v", err)
	}
	if !whisperDependencyAvailable(time.Hour) {
		t.Fatal("runnable whisper command should be available")
	}
	whisperTaskLookPath = func(string) (string, error) { return "", os.ErrNotExist }
	if whisperDependencyAvailable(time.Hour) {
		t.Fatal("a cached result must not hide a removed whisper command")
	}
}

func TestWhisperDependencyAvailableFindsMacOSPythonUserBin(t *testing.T) {
	root := t.TempDir()
	whisper := filepath.Join(root, "Library", "Python", "3.9", "bin", "whisper")
	if err := os.MkdirAll(filepath.Dir(whisper), 0o755); err != nil {
		t.Fatalf("create Python user bin: %v", err)
	}
	if err := os.WriteFile(whisper, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write whisper command: %v", err)
	}
	originalLookPath := whisperTaskLookPath
	originalHomeDir := whisperTaskUserHomeDir
	whisperTaskLookPath = func(string) (string, error) { return "", os.ErrNotExist }
	whisperTaskUserHomeDir = func() (string, error) { return root, nil }
	defer func() {
		whisperTaskLookPath = originalLookPath
		whisperTaskUserHomeDir = originalHomeDir
	}()
	whisperCheckCache.Lock()
	originalAvailableAt := whisperCheckCache.availableAt
	originalExecutable := whisperCheckCache.executable
	whisperCheckCache.availableAt = time.Time{}
	whisperCheckCache.executable = ""
	whisperCheckCache.Unlock()
	defer func() {
		whisperCheckCache.Lock()
		whisperCheckCache.availableAt = originalAvailableAt
		whisperCheckCache.executable = originalExecutable
		whisperCheckCache.Unlock()
	}()
	if !whisperDependencyAvailable(time.Hour) {
		t.Fatal("macOS Python user bin whisper command should be available")
	}
	path, available := whisperExecutablePath(time.Hour)
	if !available || path != whisper {
		t.Fatalf("resolved whisper path = %q, available=%v", path, available)
	}
}

func TestScanWhisperOutputSplitsProgressCarriageReturns(t *testing.T) {
	input := []byte(" 42%|████\r100%|████\n")
	advance, token, err := scanWhisperOutput(input, false)
	if err != nil || advance == 0 || string(token) != " 42%|████" {
		t.Fatalf("first token = %q, advance=%d, err=%v", token, advance, err)
	}
	advance, token, err = scanWhisperOutput(input[advance:], false)
	if err != nil || advance == 0 || string(token) != "100%|████" {
		t.Fatalf("second token = %q, advance=%d, err=%v", token, advance, err)
	}
}

func TestWhisperTaskCancelQueuedTask(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ensureWhisperTaskSchema(db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	result, err := db.Exec(`INSERT INTO whisper_task (agent_id, source_path, output_path, status, created_at, updated_at) VALUES ('agent-a', 'audios/a.mp3', 'audios/a.txt', 'queued', 'now', 'now')`)
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("task id: %v", err)
	}
	manager := newWhisperTaskManager(nil, db)
	if err := manager.cancelTask("agent-a", id); err != nil {
		t.Fatalf("cancel task: %v", err)
	}
	var status string
	var cancelled int
	var logs string
	if err := db.QueryRow(`SELECT status, cancel_requested, logs FROM whisper_task WHERE id = ?`, id).Scan(&status, &cancelled, &logs); err != nil {
		t.Fatalf("read task: %v", err)
	}
	if status != whisperTaskStatusCancelled || cancelled != 1 || !strings.Contains(logs, "已请求取消任务") {
		t.Fatalf("unexpected cancelled task: status=%q cancelled=%d logs=%q", status, cancelled, logs)
	}
}

func TestWhisperTaskListPageFiltersAndPaginates(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ensureWhisperTaskSchema(db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	statuses := []string{
		whisperTaskStatusQueued,
		whisperTaskStatusRunning,
		whisperTaskStatusCompleted,
		whisperTaskStatusCancelled,
		whisperTaskStatusFailed,
		whisperTaskStatusQueued,
	}
	for index, status := range statuses {
		if _, err := db.Exec(`INSERT INTO whisper_task (agent_id, source_path, output_path, status, created_at, updated_at) VALUES (?, ?, ?, ?, 'now', 'now')`, "agent-a", "audios/"+strconv.Itoa(index)+".mp3", "audios/"+strconv.Itoa(index)+".txt", status); err != nil {
			t.Fatalf("insert task %d: %v", index, err)
		}
	}
	manager := newWhisperTaskManager(nil, db)
	all, err := manager.listTasksPage("agent-a", "", 1, false)
	if err != nil {
		t.Fatalf("list all tasks: %v", err)
	}
	if all.Total != 6 || all.Page != 1 || all.PageSize != 5 || len(all.Tasks) != 5 {
		t.Fatalf("unexpected first page: %#v", all)
	}
	second, err := manager.listTasksPage("agent-a", "all", 2, false)
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if second.Total != 6 || second.Page != 2 || len(second.Tasks) != 1 {
		t.Fatalf("unexpected second page: %#v", second)
	}
	queued, err := manager.listTasksPage("agent-a", whisperTaskStatusQueued, 1, false)
	if err != nil {
		t.Fatalf("filter queued tasks: %v", err)
	}
	if queued.Total != 2 || len(queued.Tasks) != 2 || queued.Status != whisperTaskStatusQueued {
		t.Fatalf("unexpected queued page: %#v", queued)
	}
	for _, task := range queued.Tasks {
		if task.Status != whisperTaskStatusQueued {
			t.Fatalf("unexpected filtered task: %#v", task)
		}
	}
	last, err := manager.listTasksPage("agent-a", "", 99, false)
	if err != nil {
		t.Fatalf("list clamped page: %v", err)
	}
	if last.Page != 2 || len(last.Tasks) != 1 {
		t.Fatalf("unexpected clamped page: %#v", last)
	}
	if _, err := manager.listTasksPage("agent-a", "invalid", 1, false); err == nil {
		t.Fatal("invalid status filter must be rejected")
	}
}

func TestWhisperTaskAllocateOutputPathAppendsTimestamp(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "agent-a")
	if err := os.MkdirAll(filepath.Join(workspace, "whisper"), 0o755); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "whisper", "demo.txt"), []byte("existing"), 0o644); err != nil {
		t.Fatalf("write existing output: %v", err)
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ensureWhisperTaskSchema(db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	originalNow := whisperTaskNow
	whisperTaskNow = func() time.Time { return time.Date(2026, 8, 1, 16, 2, 3, 0, time.Local) }
	defer func() { whisperTaskNow = originalNow }()
	manager := newWhisperTaskManager(&Config{AgentDir: root}, db)
	relative, absolute, err := manager.allocateOutputPath("agent-a", workspace, "tmp/demo.wav", map[string]bool{})
	if err != nil {
		t.Fatalf("allocate output: %v", err)
	}
	if got, want := filepath.ToSlash(relative), "whisper/demo_20260801_160203.txt"; got != want {
		t.Fatalf("relative output = %q, want %q", got, want)
	}
	if got, want := absolute, filepath.Join(workspace, "whisper", "demo_20260801_160203.txt"); got != want {
		t.Fatalf("absolute output = %q, want %q", got, want)
	}
}

func TestWhisperTaskRestartCancelledTaskUsesNewOutputPath(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "agent-a")
	if err := os.MkdirAll(filepath.Join(workspace, "tmp"), 0o755); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(workspace, "whisper"), 0o755); err != nil {
		t.Fatalf("create whisper directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "whisper", "demo.txt"), []byte("existing transcription"), 0o644); err != nil {
		t.Fatalf("write existing whisper output: %v", err)
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ensureWhisperTaskSchema(db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	result, err := db.Exec(`INSERT INTO whisper_task (agent_id, source_path, output_path, saved_as, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, 'now', 'now')`, "agent-a", "tmp/demo.wav", "tmp/demo.txt", filepath.Join(workspace, "tmp", "demo.txt"), whisperTaskStatusCancelled)
	if err != nil {
		t.Fatalf("insert cancelled task: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("task id: %v", err)
	}
	originalNow := whisperTaskNow
	whisperTaskNow = func() time.Time { return time.Date(2026, 8, 1, 16, 2, 3, 0, time.Local) }
	defer func() { whisperTaskNow = originalNow }()
	manager := newWhisperTaskManager(&Config{AgentDir: root}, db)
	if err := manager.restartTask("agent-a", id); err != nil {
		t.Fatalf("restart cancelled task: %v", err)
	}
	var status, outputPath string
	var cancelRequested int
	if err := db.QueryRow(`SELECT status, output_path, cancel_requested FROM whisper_task WHERE id = ?`, id).Scan(&status, &outputPath, &cancelRequested); err != nil {
		t.Fatalf("read restarted task: %v", err)
	}
	if status != whisperTaskStatusQueued || cancelRequested != 0 || outputPath != "whisper/demo_20260801_160203.txt" {
		t.Fatalf("unexpected restarted task: status=%q output=%q cancelled=%d", status, outputPath, cancelRequested)
	}
}

func TestWhisperTaskRestartFailureWritesTaskLog(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ensureWhisperTaskSchema(db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	result, err := db.Exec(`INSERT INTO whisper_task (agent_id, source_path, output_path, status, created_at, updated_at) VALUES ('agent-a', 'tmp/demo.wav', 'tmp/demo.txt', ?, 'now', 'now')`, whisperTaskStatusCancelled)
	if err != nil {
		t.Fatalf("insert cancelled task: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("task id: %v", err)
	}
	manager := newWhisperTaskManager(&Config{AgentDir: filepath.Join(t.TempDir(), "missing")}, db)
	if err := manager.restartTask("agent-a", id); err == nil {
		t.Fatal("restart with a missing Agent directory must fail")
	}
	var logs string
	if err := db.QueryRow(`SELECT logs FROM whisper_task WHERE id = ?`, id).Scan(&logs); err != nil {
		t.Fatalf("read task log: %v", err)
	}
	if !strings.Contains(logs, "重新开始任务失败：Agent 不存在") {
		t.Fatalf("missing restart failure log: %q", logs)
	}
}

func TestWhisperTaskDeleteFailedTaskOnlyDeletesMatchingFailure(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ensureWhisperTaskSchema(db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	result, err := db.Exec(`INSERT INTO whisper_task (agent_id, source_path, output_path, status, created_at, updated_at) VALUES ('agent-a', 'audios/a.wav', 'whisper/a.txt', ?, 'now', 'now')`, whisperTaskStatusFailed)
	if err != nil {
		t.Fatalf("insert failed task: %v", err)
	}
	failedID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed task id: %v", err)
	}
	completed, err := db.Exec(`INSERT INTO whisper_task (agent_id, source_path, output_path, status, created_at, updated_at) VALUES ('agent-a', 'audios/b.wav', 'whisper/b.txt', ?, 'now', 'now')`, whisperTaskStatusCompleted)
	if err != nil {
		t.Fatalf("insert completed task: %v", err)
	}
	completedID, _ := completed.LastInsertId()
	manager := newWhisperTaskManager(&Config{}, db)
	if err := manager.deleteFailedTask("agent-a", failedID); err != nil {
		t.Fatalf("delete failed task: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM whisper_task WHERE id = ?`, failedID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("failed task still exists: count=%d err=%v", count, err)
	}
	if err := manager.deleteFailedTask("agent-a", completedID); err == nil {
		t.Fatal("completed task must not be deletable through failed-task endpoint")
	}
}

func TestWhisperTaskTranscribeWritesTextFileToWhisperDirectory(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "agent-a")
	audioDir := filepath.Join(workspace, "audios")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	source := filepath.Join(audioDir, "demo.mp3")
	if err := os.WriteFile(source, []byte("not decoded by the test command"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	whisper := filepath.Join(root, "whisper-test")
	script := "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then exit 0; fi\nsource=\"$1\"\nshift\nout=\"\"\nwhile [ \"$#\" -gt 0 ]; do\n  if [ \"$1\" = \"--output_dir\" ]; then out=\"$2\"; shift 2; else shift; fi\ndone\nbase=$(basename \"$source\")\nbase=${base%.*}\nprintf 'test transcription\\n' > \"$out/$base.txt\"\nprintf '100%%\\n' >&2\n"
	if err := os.WriteFile(whisper, []byte(script), 0o755); err != nil {
		t.Fatalf("write whisper script: %v", err)
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ensureWhisperTaskSchema(db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	manager := newWhisperTaskManager(&Config{AgentDir: root}, db)
	originalLookPath := whisperTaskLookPath
	whisperTaskLookPath = func(name string) (string, error) {
		if name == "whisper" {
			return whisper, nil
		}
		return originalLookPath(name)
	}
	defer func() { whisperTaskLookPath = originalLookPath }()
	task := whisperTask{ID: 1, AgentID: "agent-a", SourcePath: "audios/demo.mp3", OutputPath: "whisper/demo.txt", SavedAs: filepath.Join(workspace, "whisper", "demo.txt")}
	if err := manager.transcribe(context.Background(), task); err != nil {
		t.Fatalf("transcribe: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(workspace, "whisper", "demo.txt"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(content) != "test transcription\n" {
		t.Fatalf("unexpected output: %q", content)
	}
}

func TestWhisperTaskTranscribeFallsBackFromMPS(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "agent-a")
	audioDir := filepath.Join(workspace, "audios")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(audioDir, "demo.wav"), []byte("audio"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	whisper := filepath.Join(root, "whisper-test")
	script := "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then exit 0; fi\nsource=\"$1\"\nshift\nout=\"\"\ndevice=\"\"\nwhile [ \"$#\" -gt 0 ]; do\n  if [ \"$1\" = \"--output_dir\" ]; then out=\"$2\"; shift 2; elif [ \"$1\" = \"--device\" ]; then device=\"$2\"; shift 2; else shift; fi\ndone\nif [ \"$device\" = \"mps\" ]; then printf 'MPS backend is unsupported\\n' >&2; exit 1; fi\nbase=$(basename \"$source\")\nbase=${base%.*}\nprintf 'CPU fallback transcription\\n' > \"$out/$base.txt\"\n"
	if err := os.WriteFile(whisper, []byte(script), 0o755); err != nil {
		t.Fatalf("write whisper script: %v", err)
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ensureWhisperTaskSchema(db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO whisper_task (id, agent_id, source_path, output_path, saved_as, status, created_at, updated_at) VALUES (1, 'agent-a', 'audios/demo.wav', 'whisper/demo.txt', ?, ?, 'now', 'now')`, filepath.Join(workspace, "whisper", "demo.txt"), whisperTaskStatusRunning); err != nil {
		t.Fatalf("insert task: %v", err)
	}
	originalLookPath := whisperTaskLookPath
	originalPreferredDevice := whisperTaskPreferredDevice
	whisperTaskLookPath = func(name string) (string, error) {
		if name == "whisper" {
			return whisper, nil
		}
		return originalLookPath(name)
	}
	whisperTaskPreferredDevice = func(context.Context, string) (string, string) {
		return "mps", "检测到 Apple MPS GPU，优先使用 GPU。"
	}
	defer func() {
		whisperTaskLookPath = originalLookPath
		whisperTaskPreferredDevice = originalPreferredDevice
	}()
	whisperCheckCache.Lock()
	originalAvailableAt := whisperCheckCache.availableAt
	originalExecutable := whisperCheckCache.executable
	whisperCheckCache.availableAt = time.Time{}
	whisperCheckCache.executable = ""
	whisperCheckCache.Unlock()
	defer func() {
		whisperCheckCache.Lock()
		whisperCheckCache.availableAt = originalAvailableAt
		whisperCheckCache.executable = originalExecutable
		whisperCheckCache.Unlock()
	}()
	manager := newWhisperTaskManager(&Config{AgentDir: root}, db)
	task := whisperTask{ID: 1, AgentID: "agent-a", SourcePath: "audios/demo.wav", OutputPath: "whisper/demo.txt", SavedAs: filepath.Join(workspace, "whisper", "demo.txt"), Status: whisperTaskStatusRunning}
	if err := manager.transcribe(context.Background(), task); err != nil {
		var logs string
		_ = db.QueryRow(`SELECT logs FROM whisper_task WHERE id = 1`).Scan(&logs)
		t.Fatalf("transcribe with MPS fallback: %v; logs=%q", err, logs)
	}
	content, err := os.ReadFile(filepath.Join(workspace, "whisper", "demo.txt"))
	if err != nil || string(content) != "CPU fallback transcription\n" {
		t.Fatalf("unexpected fallback output: %q, err=%v", content, err)
	}
	var logs string
	if err := db.QueryRow(`SELECT logs FROM whisper_task WHERE id = 1`).Scan(&logs); err != nil {
		t.Fatalf("read task logs: %v", err)
	}
	if !strings.Contains(logs, "自动回退到 CPU") || !strings.Contains(logs, "已使用 CPU 回退完成") {
		t.Fatalf("missing CPU fallback logs: %q", logs)
	}
}

func TestWhisperTaskTranscribeReallocatesExistingLegacyOutput(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "agent-a")
	audioDir := filepath.Join(workspace, "tmp")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	source := filepath.Join(audioDir, "demo.wav")
	if err := os.WriteFile(source, []byte("audio"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	legacyOutput := filepath.Join(audioDir, "demo.txt")
	if err := os.WriteFile(legacyOutput, []byte("existing transcription"), 0o644); err != nil {
		t.Fatalf("write legacy output: %v", err)
	}
	whisper := filepath.Join(root, "whisper-test")
	script := "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then exit 0; fi\nsource=\"$1\"\nshift\nout=\"\"\nwhile [ \"$#\" -gt 0 ]; do\n  if [ \"$1\" = \"--output_dir\" ]; then out=\"$2\"; shift 2; else shift; fi\ndone\nbase=$(basename \"$source\")\nbase=${base%.*}\nprintf 'new transcription\\n' > \"$out/$base.txt\"\n"
	if err := os.WriteFile(whisper, []byte(script), 0o755); err != nil {
		t.Fatalf("write whisper script: %v", err)
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ensureWhisperTaskSchema(db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO whisper_task (id, agent_id, source_path, output_path, saved_as, status, created_at, updated_at) VALUES (1, 'agent-a', 'tmp/demo.wav', 'tmp/demo.txt', ?, ?, 'now', 'now')`, legacyOutput, whisperTaskStatusRunning); err != nil {
		t.Fatalf("insert legacy task: %v", err)
	}
	originalLookPath := whisperTaskLookPath
	whisperTaskLookPath = func(name string) (string, error) {
		if name == "whisper" {
			return whisper, nil
		}
		return originalLookPath(name)
	}
	defer func() { whisperTaskLookPath = originalLookPath }()
	originalNow := whisperTaskNow
	whisperTaskNow = func() time.Time { return time.Date(2026, 8, 1, 16, 2, 3, 0, time.Local) }
	defer func() { whisperTaskNow = originalNow }()
	manager := newWhisperTaskManager(&Config{AgentDir: root}, db)
	task := whisperTask{ID: 1, AgentID: "agent-a", SourcePath: "tmp/demo.wav", OutputPath: "tmp/demo.txt", SavedAs: legacyOutput, Status: whisperTaskStatusRunning}
	if err := manager.transcribe(context.Background(), task); err != nil {
		t.Fatalf("transcribe legacy task: %v", err)
	}
	newOutput := filepath.Join(workspace, "whisper", "demo.txt")
	content, err := os.ReadFile(newOutput)
	if err != nil {
		t.Fatalf("read reallocated output: %v", err)
	}
	if string(content) != "new transcription\n" {
		t.Fatalf("unexpected reallocated output: %q", content)
	}
	legacyContent, err := os.ReadFile(legacyOutput)
	if err != nil || string(legacyContent) != "existing transcription" {
		t.Fatalf("legacy output was overwritten: content=%q err=%v", legacyContent, err)
	}
	var outputPath, logs string
	if err := db.QueryRow(`SELECT output_path, logs FROM whisper_task WHERE id = 1`).Scan(&outputPath, &logs); err != nil {
		t.Fatalf("read reallocated task: %v", err)
	}
	if outputPath != "whisper/demo.txt" || !strings.Contains(logs, "已自动改为") {
		t.Fatalf("unexpected task update: output=%q logs=%q", outputPath, logs)
	}
}
