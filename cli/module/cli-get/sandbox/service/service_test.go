package service

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestDefaultShellReturnsValue(t *testing.T) {
	if strings.TrimSpace(DefaultShell()) == "" {
		t.Fatal("DefaultShell returned empty string")
	}
}

func TestExecuteTaskReturnsPlaintext(t *testing.T) {
	task := &TaskContent{
		Timeout: 5000,
		Suffix:  "cmd",
		Type:    "cmd",
		Tid:     "t-plain",
		Cmd:     "echo sandbox_plain",
	}
	result := ExecuteTask(task, DefaultShell())
	if result.Status != 0 {
		t.Fatalf("status = %d, want 0", result.Status)
	}
	if !strings.Contains(result.Cmd, "sandbox_plain") {
		t.Fatalf("cmd = %q", result.Cmd)
	}
}

func TestRunCommandReturnsPlaintext(t *testing.T) {
	result := RunCommand("echo sandbox_run", DefaultShell(), 5000)
	if result.Status != 0 {
		t.Fatalf("status = %d, want 0", result.Status)
	}
	if !strings.Contains(result.Output, "sandbox_run") {
		t.Fatalf("output = %q", result.Output)
	}
}

func TestExecuteTaskReturnsTimeoutMessage(t *testing.T) {
	task := &TaskContent{
		Timeout: 10,
		Suffix:  "cmd",
		Type:    "cmd",
		Tid:     "t-timeout",
		Cmd:     "sleep 1",
	}
	result := ExecuteTask(task, DefaultShell())
	if result.Status != 1 {
		t.Fatalf("status = %d, want 1", result.Status)
	}
	if result.Cmd != "命令执行超时" {
		t.Fatalf("cmd = %q, want 命令执行超时", result.Cmd)
	}
}

func TestExecuteTaskReturnsPermissionDeniedImmediately(t *testing.T) {
	start := time.Now()
	task := &TaskContent{
		Timeout: 10000,
		Suffix:  "cmd",
		Type:    "cmd",
		Tid:     "t-permission",
		Cmd:     "printf 'operation not permitted\\n' >&2; sleep 5",
	}

	result := ExecuteTask(task, DefaultShell())

	if result.Status != 1 {
		t.Fatalf("status = %d, want 1", result.Status)
	}
	if !strings.Contains(strings.ToLower(result.Cmd), "operation not permitted") {
		t.Fatalf("cmd = %q, want permission denied output", result.Cmd)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("elapsed = %s, want immediate return", elapsed)
	}
}

func TestNormalizeSandboxMode(t *testing.T) {
	if got := NormalizeSandboxMode(" FILEPICK_NET "); got != SandboxModeFilePickNet {
		t.Fatalf("mode = %q, want %q", got, SandboxModeFilePickNet)
	}
	if got := NormalizeSandboxMode("unknown"); got != "" {
		t.Fatalf("mode = %q, want empty", got)
	}
}

func TestRunCommandWithModeFilePickUsesSelectedDirectory(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only sandbox test")
	}
	root := t.TempDir()
	allowedDir := filepath.Join(root, "allowed")
	deniedDir := filepath.Join(root, "denied")
	if err := os.MkdirAll(allowedDir, 0o755); err != nil {
		t.Fatalf("mkdir allowed: %v", err)
	}
	if err := os.MkdirAll(deniedDir, 0o755); err != nil {
		t.Fatalf("mkdir denied: %v", err)
	}
	if err := os.WriteFile(filepath.Join(allowedDir, "ok.txt"), []byte("picked_ok"), 0o644); err != nil {
		t.Fatalf("write allowed file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(deniedDir, "blocked.txt"), []byte("blocked"), 0o644); err != nil {
		t.Fatalf("write denied file: %v", err)
	}
	t.Setenv(sandboxAllowedDirEnv, allowedDir)

	allowedResult := RunCommandWithMode("cat ok.txt", "/bin/sh", 5000, SandboxModeFilePick)
	if allowedResult.Status != 0 {
		t.Fatalf("allowed status = %d, want 0, output=%q", allowedResult.Status, allowedResult.Output)
	}
	if !strings.Contains(allowedResult.Output, "picked_ok") {
		t.Fatalf("allowed output = %q", allowedResult.Output)
	}

	deniedResult := RunCommandWithMode("cat ../denied/blocked.txt", "/bin/sh", 5000, SandboxModeFilePick)
	if deniedResult.Status != 1 {
		t.Fatalf("denied status = %d, want 1", deniedResult.Status)
	}
	if !strings.Contains(strings.ToLower(deniedResult.Output), "operation not permitted") {
		t.Fatalf("denied output = %q, want operation not permitted", deniedResult.Output)
	}
}

func TestBuildSandboxProfileAllowsDeepRightRuntimePath(t *testing.T) {
	profile := buildSandboxProfile(SandboxModeFilePick, "/tmp/picked")
	if !strings.Contains(profile, "/Library/Application Support/deepright") {
		t.Fatalf("profile should allow deepright runtime path, got %q", profile)
	}
	if !strings.Contains(profile, "selected-dir.txt") && !strings.Contains(profile, "CLI_SANDBOX") {
		t.Fatalf("profile should include sandbox state path, got %q", profile)
	}
}

func TestSetPickedDirectoryWritesCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	allowedDir := filepath.Join(home, "picked")
	if err := os.MkdirAll(allowedDir, 0o755); err != nil {
		t.Fatalf("mkdir picked: %v", err)
	}

	got, err := SetPickedDirectory(allowedDir)
	if err != nil {
		t.Fatalf("SetPickedDirectory: %v", err)
	}
	if got != filepath.Clean(allowedDir) {
		t.Fatalf("path = %q, want %q", got, filepath.Clean(allowedDir))
	}

	cached, ok := readCachedPickedDirectory()
	if !ok {
		t.Fatal("readCachedPickedDirectory should return cache hit")
	}
	if cached != filepath.Clean(allowedDir) {
		t.Fatalf("cached = %q, want %q", cached, filepath.Clean(allowedDir))
	}
}

func TestSetPickedDirectoryAcceptsQuotedPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	allowedDir := filepath.Join(home, "picked")
	if err := os.MkdirAll(allowedDir, 0o755); err != nil {
		t.Fatalf("mkdir picked: %v", err)
	}

	got, err := SetPickedDirectory(`"` + allowedDir + `"`)
	if err != nil {
		t.Fatalf("SetPickedDirectory quoted: %v", err)
	}
	if got != filepath.Clean(allowedDir) {
		t.Fatalf("path = %q, want %q", got, filepath.Clean(allowedDir))
	}
}

func TestResolvePickedDirectoryReturnsCachedDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	allowedDir := filepath.Join(home, "picked")
	if err := os.MkdirAll(allowedDir, 0o755); err != nil {
		t.Fatalf("mkdir picked: %v", err)
	}
	statePath, err := sandboxSelectedDirPath()
	if err != nil {
		t.Fatalf("sandboxSelectedDirPath: %v", err)
	}
	if err := os.WriteFile(statePath, []byte(allowedDir+"\n"), 0o644); err != nil {
		t.Fatalf("write selected-dir cache: %v", err)
	}

	got, err := resolvePickedDirectory()
	if err != nil {
		t.Fatalf("resolvePickedDirectory err = %v", err)
	}
	if got != filepath.Clean(allowedDir) {
		t.Fatalf("path = %q, want %q", got, filepath.Clean(allowedDir))
	}
}

func TestResolvePickedDirectoryReturnsHelpfulErrorWithoutCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, err := resolvePickedDirectory()
	if err == nil {
		t.Fatal("resolvePickedDirectory err = nil, want authorization error")
	}
	if !strings.Contains(err.Error(), "请先通过 CLI_SANDBOX.app 完成目录授权") {
		t.Fatalf("err = %q, want missing authorization message", err)
	}
}

func TestRunCommandWithModeReturnsMissingAuthorizationMessage(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only sandbox test")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)

	result := RunCommandWithMode("pwd", "/bin/sh", 5000, SandboxModeFilePick)
	if result.Status != 1 {
		t.Fatalf("status = %d, want 1", result.Status)
	}
	if !strings.Contains(result.Output, "请先通过 CLI_SANDBOX.app 完成目录授权") {
		t.Fatalf("output = %q, want missing authorization message", result.Output)
	}
}
