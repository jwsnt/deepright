package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBrowserParseProfileCleanupConfig(t *testing.T) {
	config, err := browserParseProfileCleanupConfig([]byte(`{"browser":{"clear":72,"scan":2}}`))
	if err != nil {
		t.Fatal(err)
	}
	if config.ClearAfter != 72*time.Hour {
		t.Fatalf("clear duration = %s, want 72h", config.ClearAfter)
	}
	if config.ScanEvery != 2*time.Hour {
		t.Fatalf("scan duration = %s, want 2h", config.ScanEvery)
	}

	for name, data := range map[string]string{
		"missing browser": `{"other":true}`,
		"missing clear":   `{"browser":{"scan":2}}`,
		"missing scan":    `{"browser":{"clear":72}}`,
		"invalid clear":   `{"browser":{"clear":0,"scan":2}}`,
		"invalid scan":    `{"browser":{"clear":72,"scan":"2"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := browserParseProfileCleanupConfig([]byte(data)); err == nil {
				t.Fatalf("browserParseProfileCleanupConfig(%s) unexpectedly succeeded", data)
			}
		})
	}
}

func TestBrowserScanAgentProfileDirectoriesRemovesOnlyExpiredChromeDirectories(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	now := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	browserNowFn = func() time.Time { return now }
	agentRoot := t.TempDir()
	workspace := filepath.Join(agentRoot, "agent-a")
	oldProfile := filepath.Join(workspace, "ChRoMe_old")
	newProfile := filepath.Join(workspace, "chrome_new")
	nonProfile := filepath.Join(workspace, "not-chrome")
	for _, path := range []string{oldProfile, newProfile, nonProfile} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chtimes(oldProfile, now.Add(-73*time.Hour), now.Add(-73*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newProfile, now.Add(-72*time.Hour), now.Add(-72*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "chrome_file"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := browserProfileCleanupResult{Platform: "macos", Root: agentRoot}
	browserScanAgentProfileDirectories(agentRoot, 72*time.Hour, &result)
	if result.Scanned != 2 || result.Removed != 1 || result.Skipped != 1 {
		t.Fatalf("scan result = %+v, want scanned=2 removed=1 skipped=1", result)
	}
	for _, path := range []string{newProfile, nonProfile, filepath.Join(workspace, "chrome_file")} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("%s should remain: %v", path, err)
		}
	}
	if _, err := os.Stat(oldProfile); !os.IsNotExist(err) {
		t.Fatalf("expired profile stat err = %v, want not exist", err)
	}
}

func TestBrowserResolveProfileCleanupTargetUsesWSLChatProfileDirectory(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	wslRoot := t.TempDir()
	now := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	browserNowFn = func() time.Time { return now }
	browserWSLDetectFn = func() (bool, error) { return true, nil }
	browserWSLPathUnixFn = func(path string) (string, error) {
		if path != browserWSLProfileCleanupRoot {
			t.Fatalf("Windows path = %q, want %q", path, browserWSLProfileCleanupRoot)
		}
		return wslRoot, nil
	}

	target, ok, err := browserResolveProfileCleanupTarget()
	if err != nil {
		t.Fatal(err)
	}
	if !ok || target.Platform != "wsl" || target.Root != wslRoot || target.AgentDir || !target.ChatDir {
		t.Fatalf("target = %+v, ok = %v", target, ok)
	}

	expired := filepath.Join(wslRoot, "chat-expired")
	current := filepath.Join(wslRoot, "chat-current")
	for _, path := range []string{expired, current} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chtimes(expired, now.Add(-73*time.Hour), now.Add(-73*time.Hour)); err != nil {
		t.Fatal(err)
	}
	browserRunProfileCleanupScan(browserProfileCleanupConfig{ClearAfter: 72 * time.Hour, ScanEvery: time.Hour})
	if _, err := os.Stat(expired); !os.IsNotExist(err) {
		t.Fatalf("expired WSL profile stat err = %v, want not exist", err)
	}
	if _, err := os.Stat(current); err != nil {
		t.Fatalf("current WSL profile should remain: %v", err)
	}
}

func TestBrowserResolveProfileCleanupTargetSkipsNativeLinux(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	browserRuntimeGOOSFn = func() string { return "linux" }
	browserWSLDetectFn = func() (bool, error) { return false, nil }
	_, ok, err := browserResolveProfileCleanupTarget()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("native Linux must not have a profile cleanup target")
	}
}

func TestPluginLifecycleStartsAndStopsProfileCleanupWorker(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	var starts, stops int
	browserProfileCleanupStartFn = func() error {
		starts++
		return nil
	}
	browserProfileCleanupStopFn = func() error {
		stops++
		return nil
	}

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	executableDir := t.TempDir()
	browserExecutablePathFn = func() (string, error) {
		return writeBrowserBinaryFixture(t, executableDir), nil
	}
	connectBin, _ := writeBundledConnectBinFixture(t, homeDir, map[string]string{
		"app-dir": filepath.Join(t.TempDir(), "runtime-app"),
	})

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := runCLI([]string{"start", "--connect-bin", connectBin}, stdout, stderr); code != 0 {
		t.Fatalf("start exit code = %d, stderr = %s", code, stderr.String())
	}
	if code := runCLI([]string{"stop"}, stdout, stderr); code != 0 {
		t.Fatalf("stop exit code = %d, stderr = %s", code, stderr.String())
	}
	if starts != 1 || stops != 1 {
		t.Fatalf("cleanup worker starts=%d stops=%d, want 1,1", starts, stops)
	}
	if strings.Count(stdout.String(), "OK") != 2 {
		t.Fatalf("lifecycle output = %q", stdout.String())
	}
}
