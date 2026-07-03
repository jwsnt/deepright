package sharedutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBrowserOpenCommandLinuxPrefersChromeFamily(t *testing.T) {
	oldLookPath := browserOpenLookPathFn
	oldStat := browserOpenStatFn
	defer func() {
		browserOpenLookPathFn = oldLookPath
		browserOpenStatFn = oldStat
	}()

	browserOpenLookPathFn = func(name string) (string, error) {
		if name == "google-chrome" {
			return "/usr/bin/google-chrome", nil
		}
		return "", errors.New("missing")
	}
	browserOpenStatFn = os.Stat

	cmd, args, err := browserOpenCommand("linux", "http://localhost:8080/site/#app")
	if err != nil {
		t.Fatalf("browserOpenCommand: %v", err)
	}
	if cmd != "/usr/bin/google-chrome" {
		t.Fatalf("cmd = %q", cmd)
	}
	if len(args) != 2 || args[0] != "--start-maximized" || args[1] != "http://localhost:8080/site/#app" {
		t.Fatalf("args = %#v", args)
	}
}

func TestBrowserOpenCommandDarwinFallsBackToOpen(t *testing.T) {
	oldStat := browserOpenStatFn
	defer func() { browserOpenStatFn = oldStat }()

	browserOpenStatFn = func(string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}

	cmd, args, err := browserOpenCommand("darwin", "http://localhost:8080/site/#app")
	if err != nil {
		t.Fatalf("browserOpenCommand: %v", err)
	}
	if cmd != "open" {
		t.Fatalf("cmd = %q", cmd)
	}
	if len(args) != 1 || args[0] != "http://localhost:8080/site/#app" {
		t.Fatalf("args = %#v", args)
	}
}

func TestBrowserOpenCommandWindowsPrefersInstalledExecutable(t *testing.T) {
	oldLookPath := browserOpenLookPathFn
	oldStat := browserOpenStatFn
	oldProgramFiles := os.Getenv("ProgramFiles")
	defer func() {
		browserOpenLookPathFn = oldLookPath
		browserOpenStatFn = oldStat
		_ = os.Setenv("ProgramFiles", oldProgramFiles)
	}()

	root := t.TempDir()
	chromePath := filepath.Join(root, "Google", "Chrome", "Application", "chrome.exe")
	if err := os.MkdirAll(filepath.Dir(chromePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chromePath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("ProgramFiles", root); err != nil {
		t.Fatal(err)
	}

	browserOpenLookPathFn = func(name string) (string, error) {
		return "", errors.New("missing")
	}
	browserOpenStatFn = os.Stat

	cmd, args, err := browserOpenCommand("windows", "http://localhost:8080/site/#app")
	if err != nil {
		t.Fatalf("browserOpenCommand: %v", err)
	}
	if cmd != chromePath {
		t.Fatalf("cmd = %q, want %q", cmd, chromePath)
	}
	if len(args) != 2 || args[0] != "--start-maximized" || args[1] != "http://localhost:8080/site/#app" {
		t.Fatalf("args = %#v", args)
	}
}

func TestOpenBrowserMaximizedDarwinActivatesExistingChromeTab(t *testing.T) {
	oldStat := browserOpenStatFn
	oldAppleScript := browserOpenAppleScriptFn
	oldListProcesses := browserOpenListProcessesFn
	defer func() {
		browserOpenStatFn = oldStat
		browserOpenAppleScriptFn = oldAppleScript
		browserOpenListProcessesFn = oldListProcesses
	}()

	root := t.TempDir()
	chromeApp := filepath.Join(root, "Google Chrome.app")
	if err := os.MkdirAll(chromeApp, 0o755); err != nil {
		t.Fatal(err)
	}

	var gotScript string
	browserOpenStatFn = func(path string) (os.FileInfo, error) {
		if path == "/Applications/Google Chrome.app" {
			return os.Stat(chromeApp)
		}
		return nil, os.ErrNotExist
	}
	browserOpenListProcessesFn = func() ([]browserOpenProcessInfo, error) {
		return nil, nil
	}
	browserOpenAppleScriptFn = func(script string) error {
		gotScript = script
		return nil
	}

	handled, err := browserOpenOrActivateChromeTab("http://localhost:8080/site/#app")
	if err != nil {
		t.Fatalf("browserOpenOrActivateChromeTab: %v", err)
	}
	if !handled {
		t.Fatal("expected helper to handle Chrome activation path")
	}
	if !strings.Contains(gotScript, `set targetUrl to "http://localhost:8080/site/#app"`) {
		t.Fatalf("script missing target URL: %q", gotScript)
	}
	if !strings.Contains(gotScript, `if (URL of t) is targetUrl then`) {
		t.Fatalf("script does not scan existing tabs: %q", gotScript)
	}
}

func TestOpenBrowserMaximizedDarwinSkipsChromeTabRestoreForCDPProcess(t *testing.T) {
	oldStat := browserOpenStatFn
	oldAppleScript := browserOpenAppleScriptFn
	oldListProcesses := browserOpenListProcessesFn
	defer func() {
		browserOpenStatFn = oldStat
		browserOpenAppleScriptFn = oldAppleScript
		browserOpenListProcessesFn = oldListProcesses
	}()

	root := t.TempDir()
	chromeApp := filepath.Join(root, "Google Chrome.app")
	if err := os.MkdirAll(chromeApp, 0o755); err != nil {
		t.Fatal(err)
	}

	browserOpenStatFn = func(path string) (os.FileInfo, error) {
		if path == "/Applications/Google Chrome.app" {
			return os.Stat(chromeApp)
		}
		return nil, os.ErrNotExist
	}
	browserOpenListProcessesFn = func() ([]browserOpenProcessInfo, error) {
		return []browserOpenProcessInfo{{
			CommandLine: "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome --remote-debugging-port=20001 --user-data-dir=/tmp/chrome_20001",
		}}, nil
	}
	browserOpenAppleScriptFn = func(string) error {
		t.Fatal("AppleScript should be skipped when a Chrome CDP process is running")
		return nil
	}

	handled, err := browserOpenOrActivateChromeTab("http://localhost:8080/site/#app")
	if err != nil {
		t.Fatalf("browserOpenOrActivateChromeTab: %v", err)
	}
	if handled {
		t.Fatal("expected helper to fall back when Chrome CDP process is running")
	}
}

func TestOpenBrowserMaximizedDarwinFallsBackWhenChromeMissing(t *testing.T) {
	oldStat := browserOpenStatFn
	defer func() {
		browserOpenStatFn = oldStat
	}()

	browserOpenStatFn = func(string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}

	handled, err := browserOpenOrActivateChromeTab("http://localhost:8080/site/#app")
	if err != nil {
		t.Fatalf("browserOpenOrActivateChromeTab: %v", err)
	}
	if handled {
		t.Fatal("expected helper to fall back when Chrome app is missing")
	}
}
