package main

import (
	"bufio"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseRembgCheckConfig(t *testing.T) {
	cfg, err := parseRembgCheckConfig(map[string]interface{}{"rembg": map[string]interface{}{"check": float64(24), "install": "请安装 rembg"}})
	if err != nil || cfg.CacheFor != 24*time.Hour || cfg.Install != "请安装 rembg" {
		t.Fatalf("parse valid rembg config = %#v, %v", cfg, err)
	}
	if _, err := parseRembgCheckConfig(map[string]interface{}{"rembg": map[string]interface{}{"check": 0, "install": "x"}}); err == nil {
		t.Fatal("zero cache duration must be rejected")
	}
}

func TestNormalizeRembgModel(t *testing.T) {
	if model, ok := normalizeRembgModel(""); !ok || model != rembgDefaultModel {
		t.Fatalf("empty model = %q, %v", model, ok)
	}
	if model, ok := normalizeRembgModel("isnet-general-use"); !ok || model != "isnet-general-use" {
		t.Fatalf("known model = %q, %v", model, ok)
	}
	if _, ok := normalizeRembgModel("--output /tmp/untrusted"); ok {
		t.Fatal("unsupported model must be rejected")
	}
}

func TestSplitRembgOutputAcceptsCarriageReturnProgress(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("下载 0%\r下载 50%\r下载 100%\n完成"))
	scanner.Split(splitRembgOutput)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(lines, "|")
	if got != "下载 0%|下载 50%|下载 100%|完成" {
		t.Fatalf("split output = %q", got)
	}
}

func TestEnsureRembgTaskSchemaMigratesModelOptions(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE rembg_task (
		id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL, source_path TEXT NOT NULL,
		output_path TEXT NOT NULL DEFAULT '', saved_as TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'queued',
		progress INTEGER NOT NULL DEFAULT 0, started_at TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL, logs TEXT NOT NULL DEFAULT '', cancel_requested INTEGER NOT NULL DEFAULT 0
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO rembg_task(agent_id,source_path,created_at,updated_at) VALUES('agent-a','tmp/source.png','now','now')`); err != nil {
		t.Fatal(err)
	}
	if err := ensureRembgTaskSchema(db); err != nil {
		t.Fatal(err)
	}
	var model string
	var alpha int
	if err := db.QueryRow(`SELECT model,alpha_matting FROM rembg_task`).Scan(&model, &alpha); err != nil {
		t.Fatal(err)
	}
	if model != rembgDefaultModel || alpha != 0 {
		t.Fatalf("migrated model options = %q, %d", model, alpha)
	}
}

func TestRembgExecutableFindsMacOSPythonUserBin(t *testing.T) {
	root := t.TempDir()
	rembg := filepath.Join(root, "Library", "Python", "3.12", "bin", "rembg")
	if err := os.MkdirAll(filepath.Dir(rembg), 0o755); err != nil {
		t.Fatalf("create Python user bin: %v", err)
	}
	if err := os.WriteFile(rembg, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write rembg command: %v", err)
	}
	originalLookPath := rembgLookPath
	originalHomeDir := rembgUserHomeDir
	originalGlob := rembgGlob
	rembgLookPath = func(string) (string, error) { return "", os.ErrNotExist }
	rembgUserHomeDir = func() (string, error) { return root, nil }
	rembgGlob = func(pattern string) []string {
		if strings.HasPrefix(pattern, filepath.Join(root, "Library", "Python")) {
			return []string{rembg}
		}
		return nil
	}
	defer func() {
		rembgLookPath = originalLookPath
		rembgUserHomeDir = originalHomeDir
		rembgGlob = originalGlob
	}()
	rembgCheckCache.Lock()
	originalCheckedAt := rembgCheckCache.checkedAt
	originalExecutable := rembgCheckCache.executable
	rembgCheckCache.checkedAt = time.Time{}
	rembgCheckCache.executable = ""
	rembgCheckCache.Unlock()
	defer func() {
		rembgCheckCache.Lock()
		rembgCheckCache.checkedAt = originalCheckedAt
		rembgCheckCache.executable = originalExecutable
		rembgCheckCache.Unlock()
	}()

	path, available := rembgExecutable(time.Hour)
	if !available || path != rembg {
		t.Fatalf("resolved rembg path = %q, available=%v", path, available)
	}
}

func TestRembgExecutableRequiresRunnableCommandAndInvalidatesMissingCache(t *testing.T) {
	root := t.TempDir()
	rembg := filepath.Join(root, "rembg")
	if err := os.WriteFile(rembg, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write unavailable rembg command: %v", err)
	}
	originalLookPath := rembgLookPath
	originalHomeDir := rembgUserHomeDir
	originalGlob := rembgGlob
	rembgLookPath = func(string) (string, error) { return rembg, nil }
	rembgUserHomeDir = func() (string, error) { return root, nil }
	rembgGlob = func(string) []string { return nil }
	defer func() {
		rembgLookPath = originalLookPath
		rembgUserHomeDir = originalHomeDir
		rembgGlob = originalGlob
	}()
	rembgCheckCache.Lock()
	originalCheckedAt := rembgCheckCache.checkedAt
	originalExecutable := rembgCheckCache.executable
	rembgCheckCache.checkedAt = time.Time{}
	rembgCheckCache.executable = ""
	rembgCheckCache.Unlock()
	defer func() {
		rembgCheckCache.Lock()
		rembgCheckCache.checkedAt = originalCheckedAt
		rembgCheckCache.executable = originalExecutable
		rembgCheckCache.Unlock()
	}()

	if _, available := rembgExecutable(time.Hour); available {
		t.Fatal("non-runnable rembg command must not be treated as available")
	}
	if err := os.WriteFile(rembg, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write runnable rembg command: %v", err)
	}
	if _, available := rembgExecutable(time.Hour); !available {
		t.Fatal("runnable rembg command should be available")
	}
	rembgLookPath = func(string) (string, error) { return "", os.ErrNotExist }
	if _, available := rembgExecutable(time.Hour); available {
		t.Fatal("a cached result must not hide a removed rembg command")
	}
}

func TestRembgTaskListCancelAndDelete(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ensureRembgTaskSchema(db); err != nil {
		t.Fatal(err)
	}
	manager := &rembgTaskManager{db: db, wake: make(chan struct{}, 1)}
	for i, status := range []string{rembgTaskQueued, rembgTaskCompleted, rembgTaskFailed, rembgTaskQueued, rembgTaskQueued, rembgTaskQueued} {
		if _, err := db.Exec(`INSERT INTO rembg_task(agent_id,source_path,output_path,status,created_at,updated_at) VALUES(?,?,?,?, 'now','now')`, "agent-a", "tmp/image"+string(rune('a'+i))+".png", "images/out.png", status); err != nil {
			t.Fatal(err)
		}
	}
	page, err := manager.list("agent-a", "", 99, false)
	if err != nil || page.Page != 2 || page.Total != 6 || len(page.Tasks) != 1 {
		t.Fatalf("unexpected paged list: %#v, %v", page, err)
	}
	if page.Tasks[0].Model != rembgDefaultModel || page.Tasks[0].AlphaMatting {
		t.Fatalf("default persisted task options = %#v", page.Tasks[0])
	}
	var queuedID, failedID int64
	if err := db.QueryRow(`SELECT id FROM rembg_task WHERE status=? ORDER BY id LIMIT 1`, rembgTaskQueued).Scan(&queuedID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT id FROM rembg_task WHERE status=?`, rembgTaskFailed).Scan(&failedID); err != nil {
		t.Fatal(err)
	}
	if err := manager.cancelTask("agent-a", queuedID); err != nil {
		t.Fatal(err)
	}
	var status string
	var requested int
	var logs string
	if err := db.QueryRow(`SELECT status,cancel_requested,logs FROM rembg_task WHERE id=?`, queuedID).Scan(&status, &requested, &logs); err != nil {
		t.Fatal(err)
	}
	if status != rembgTaskCancelled || requested != 1 || !strings.Contains(logs, "已请求取消任务") {
		t.Fatalf("unexpected cancellation: %q %d %q", status, requested, logs)
	}
	if err := manager.deleteFailed("agent-a", failedID); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM rembg_task WHERE id=?`, failedID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("failed task remains: %d %v", count, err)
	}
}

func TestRembgTaskOutputNamesAreAgentScoped(t *testing.T) {
	root := t.TempDir()
	for _, id := range []string{"agent-a", "agent-b"} {
		if err := os.MkdirAll(filepath.Join(root, id, "images"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ensureRembgTaskSchema(db); err != nil {
		t.Fatal(err)
	}
	manager := &rembgTaskManager{db: db}
	if _, err := db.Exec(`INSERT INTO rembg_task(agent_id,source_path,output_path,status,created_at,updated_at) VALUES('agent-a','tmp/a.png','images/photo_subject.png','completed','now','now')`); err != nil {
		t.Fatal(err)
	}
	output, _, err := manager.allocate("agent-b", filepath.Join(root, "agent-b"), "tmp/photo.jpg", map[string]bool{}, 0)
	if err != nil || filepath.ToSlash(output) != "images/photo_subject.png" {
		t.Fatalf("agent-b should be allowed to use its own output name: %q, %v", output, err)
	}
}
