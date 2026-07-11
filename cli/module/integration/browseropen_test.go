package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegrationBrowserOpenCommandLinuxPrefersChromeFamily(t *testing.T) {
	oldLookPath := integrationBrowserLookPathFn
	oldStat := integrationBrowserStatFn
	oldGetenv := integrationBrowserGetenvFn
	oldReadFile := integrationBrowserReadFileFn
	defer func() {
		integrationBrowserLookPathFn = oldLookPath
		integrationBrowserStatFn = oldStat
		integrationBrowserGetenvFn = oldGetenv
		integrationBrowserReadFileFn = oldReadFile
	}()

	integrationBrowserLookPathFn = func(name string) (string, error) {
		if name == "google-chrome" {
			return "/usr/bin/google-chrome", nil
		}
		return "", errors.New("missing")
	}
	integrationBrowserStatFn = os.Stat
	integrationBrowserGetenvFn = func(string) string { return "" }
	integrationBrowserReadFileFn = func(string) ([]byte, error) { return []byte("Linux version"), nil }

	cmd, args, err := integrationBrowserOpenCommand("linux", "http://localhost:8080/site/#app")
	if err != nil {
		t.Fatalf("integrationBrowserOpenCommand: %v", err)
	}
	if cmd != "/usr/bin/google-chrome" {
		t.Fatalf("cmd = %q", cmd)
	}
	if len(args) != 2 || args[0] != "--start-maximized" || args[1] != "http://localhost:8080/site/#app" {
		t.Fatalf("args = %#v", args)
	}
}

func TestIntegrationBrowserOpenCommandWSLPrefersWindowsChrome(t *testing.T) {
	oldLookPath := integrationBrowserLookPathFn
	oldStat := integrationBrowserStatFn
	oldGetenv := integrationBrowserGetenvFn
	oldReadFile := integrationBrowserReadFileFn
	defer func() {
		integrationBrowserLookPathFn = oldLookPath
		integrationBrowserStatFn = oldStat
		integrationBrowserGetenvFn = oldGetenv
		integrationBrowserReadFileFn = oldReadFile
	}()

	root := t.TempDir()
	chromePath := filepath.Join(root, "Program Files", "Google", "Chrome", "Application", "chrome.exe")
	if err := os.MkdirAll(filepath.Dir(chromePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chromePath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}

	integrationBrowserLookPathFn = func(string) (string, error) {
		return "", errors.New("missing")
	}
	integrationBrowserStatFn = func(path string) (os.FileInfo, error) {
		if path == "/mnt/c/Program Files/Google/Chrome/Application/chrome.exe" {
			return os.Stat(chromePath)
		}
		return nil, os.ErrNotExist
	}
	integrationBrowserGetenvFn = func(string) string { return "" }
	integrationBrowserReadFileFn = func(string) ([]byte, error) {
		return []byte("Linux version 5.15.0 microsoft-standard-WSL2"), nil
	}

	cmd, args, err := integrationBrowserOpenCommand("linux", "http://localhost:8080/site/#app")
	if err != nil {
		t.Fatalf("integrationBrowserOpenCommand: %v", err)
	}
	if cmd != "/mnt/c/Program Files/Google/Chrome/Application/chrome.exe" {
		t.Fatalf("cmd = %q", cmd)
	}
	if len(args) != 2 || args[0] != "--start-maximized" || args[1] != "http://localhost:8080/site/#app" {
		t.Fatalf("args = %#v", args)
	}
}

func TestIntegrationBrowserOpenCommandDarwinFallsBackToOpen(t *testing.T) {
	oldStat := integrationBrowserStatFn
	defer func() { integrationBrowserStatFn = oldStat }()

	integrationBrowserStatFn = func(string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}

	cmd, args, err := integrationBrowserOpenCommand("darwin", "http://localhost:8080/site/#app")
	if err != nil {
		t.Fatalf("integrationBrowserOpenCommand: %v", err)
	}
	if cmd != "open" {
		t.Fatalf("cmd = %q", cmd)
	}
	if len(args) != 1 || args[0] != "http://localhost:8080/site/#app" {
		t.Fatalf("args = %#v", args)
	}
}

func TestIntegrationBrowserOpenCommandDarwinPrefersOpenWithChromeApp(t *testing.T) {
	oldStat := integrationBrowserStatFn
	defer func() { integrationBrowserStatFn = oldStat }()

	root := t.TempDir()
	chromeApp := filepath.Join(root, "Google Chrome.app")
	if err := os.MkdirAll(chromeApp, 0o755); err != nil {
		t.Fatal(err)
	}

	integrationBrowserStatFn = func(path string) (os.FileInfo, error) {
		if path == "/Applications/Google Chrome.app" {
			return os.Stat(chromeApp)
		}
		return nil, os.ErrNotExist
	}

	cmd, args, err := integrationBrowserOpenCommand("darwin", "http://localhost:8080/site/#app")
	if err != nil {
		t.Fatalf("integrationBrowserOpenCommand: %v", err)
	}
	if cmd != "open" {
		t.Fatalf("cmd = %q", cmd)
	}
	if len(args) != 3 || args[0] != "-a" || args[1] != "/Applications/Google Chrome.app" || args[2] != "http://localhost:8080/site/#app" {
		t.Fatalf("args = %#v", args)
	}
}

func TestIntegrationBrowserOpenCommandWindowsFallsBackToCmdStart(t *testing.T) {
	oldLookPath := integrationBrowserLookPathFn
	oldStat := integrationBrowserStatFn
	oldGetenv := integrationBrowserGetenvFn
	defer func() {
		integrationBrowserLookPathFn = oldLookPath
		integrationBrowserStatFn = oldStat
		integrationBrowserGetenvFn = oldGetenv
	}()

	integrationBrowserLookPathFn = func(string) (string, error) {
		return "", errors.New("missing")
	}
	integrationBrowserStatFn = func(string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}
	integrationBrowserGetenvFn = func(string) string { return "" }

	cmd, args, err := integrationBrowserOpenCommand("windows", "http://localhost:8080/site/#app")
	if err != nil {
		t.Fatalf("integrationBrowserOpenCommand: %v", err)
	}
	if cmd != "cmd" {
		t.Fatalf("cmd = %q", cmd)
	}
	if len(args) != 4 || args[0] != "/c" || args[1] != "start" || args[2] != "/max" || args[3] != "http://localhost:8080/site/#app" {
		t.Fatalf("args = %#v", args)
	}
}

func TestIntegrationBrowserOpenCommandUnsupportedOS(t *testing.T) {
	if _, _, err := integrationBrowserOpenCommand("plan9", "http://localhost:8080/site/#app"); err == nil || err.Error() != "unsupported OS" {
		t.Fatalf("err = %v", err)
	}
}

func TestOpenIntegrationBrowserMaximizedDarwinActivatesExistingChromeTab(t *testing.T) {
	oldStat := integrationBrowserStatFn
	oldAppleScript := integrationBrowserAppleScriptFn
	oldListProcesses := integrationBrowserListProcessesFn
	defer func() {
		integrationBrowserStatFn = oldStat
		integrationBrowserAppleScriptFn = oldAppleScript
		integrationBrowserListProcessesFn = oldListProcesses
	}()

	root := t.TempDir()
	chromeApp := filepath.Join(root, "Google Chrome.app")
	if err := os.MkdirAll(chromeApp, 0o755); err != nil {
		t.Fatal(err)
	}

	var gotScript string
	integrationBrowserStatFn = func(path string) (os.FileInfo, error) {
		if path == "/Applications/Google Chrome.app" {
			return os.Stat(chromeApp)
		}
		return nil, os.ErrNotExist
	}
	integrationBrowserListProcessesFn = func() ([]integrationBrowserProcessInfo, error) {
		return []integrationBrowserProcessInfo{{
			CommandLine: "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		}}, nil
	}
	integrationBrowserAppleScriptFn = func(script string) (string, error) {
		gotScript = script
		return "activated", nil
	}

	handled, err := integrationBrowserOpenOrActivateChromeTab("http://localhost:8080/site/#app")
	if err != nil {
		t.Fatalf("integrationBrowserOpenOrActivateChromeTab: %v", err)
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
	if strings.Contains(gotScript, `open location targetUrl`) {
		t.Fatalf("script should not open new locations directly: %q", gotScript)
	}
}

func TestOpenIntegrationBrowserMaximizedDarwinSkipsChromeTabRestoreForCDPProcess(t *testing.T) {
	oldStat := integrationBrowserStatFn
	oldAppleScript := integrationBrowserAppleScriptFn
	oldListProcesses := integrationBrowserListProcessesFn
	defer func() {
		integrationBrowserStatFn = oldStat
		integrationBrowserAppleScriptFn = oldAppleScript
		integrationBrowserListProcessesFn = oldListProcesses
	}()

	root := t.TempDir()
	chromeApp := filepath.Join(root, "Google Chrome.app")
	if err := os.MkdirAll(chromeApp, 0o755); err != nil {
		t.Fatal(err)
	}

	integrationBrowserStatFn = func(path string) (os.FileInfo, error) {
		if path == "/Applications/Google Chrome.app" {
			return os.Stat(chromeApp)
		}
		return nil, os.ErrNotExist
	}
	integrationBrowserListProcessesFn = func() ([]integrationBrowserProcessInfo, error) {
		return []integrationBrowserProcessInfo{{
			CommandLine: "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome --remote-debugging-port=20001 --user-data-dir=/tmp/chrome_20001",
		}}, nil
	}
	integrationBrowserAppleScriptFn = func(string) (string, error) {
		t.Fatal("AppleScript should be skipped when a Chrome CDP process is running")
		return "", nil
	}

	handled, err := integrationBrowserOpenOrActivateChromeTab("http://localhost:8080/site/#app")
	if err != nil {
		t.Fatalf("integrationBrowserOpenOrActivateChromeTab: %v", err)
	}
	if handled {
		t.Fatal("expected helper to fall back when Chrome CDP process is running")
	}
}

func TestOpenIntegrationBrowserMaximizedDarwinFallsBackWhenChromeMissing(t *testing.T) {
	oldStat := integrationBrowserStatFn
	defer func() {
		integrationBrowserStatFn = oldStat
	}()

	integrationBrowserStatFn = func(string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}

	handled, err := integrationBrowserOpenOrActivateChromeTab("http://localhost:8080/site/#app")
	if err != nil {
		t.Fatalf("integrationBrowserOpenOrActivateChromeTab: %v", err)
	}
	if handled {
		t.Fatal("expected helper to fall back when Chrome app is missing")
	}
}

func TestOpenIntegrationBrowserMaximizedDarwinFallsBackWhenChromeNotRunning(t *testing.T) {
	oldStat := integrationBrowserStatFn
	oldAppleScript := integrationBrowserAppleScriptFn
	oldListProcesses := integrationBrowserListProcessesFn
	defer func() {
		integrationBrowserStatFn = oldStat
		integrationBrowserAppleScriptFn = oldAppleScript
		integrationBrowserListProcessesFn = oldListProcesses
	}()

	root := t.TempDir()
	chromeApp := filepath.Join(root, "Google Chrome.app")
	if err := os.MkdirAll(chromeApp, 0o755); err != nil {
		t.Fatal(err)
	}

	integrationBrowserStatFn = func(path string) (os.FileInfo, error) {
		if path == "/Applications/Google Chrome.app" {
			return os.Stat(chromeApp)
		}
		return nil, os.ErrNotExist
	}
	integrationBrowserListProcessesFn = func() ([]integrationBrowserProcessInfo, error) {
		return nil, nil
	}
	integrationBrowserAppleScriptFn = func(string) (string, error) {
		t.Fatal("AppleScript should be skipped when Chrome is not running")
		return "", nil
	}

	handled, err := integrationBrowserOpenOrActivateChromeTab("http://localhost:8080/site/#app")
	if err != nil {
		t.Fatalf("integrationBrowserOpenOrActivateChromeTab: %v", err)
	}
	if handled {
		t.Fatal("expected helper to fall back when Chrome is not running")
	}
}

func TestOpenIntegrationBrowserMaximizedDarwinFallsBackWhenMatchingTabMissing(t *testing.T) {
	oldStat := integrationBrowserStatFn
	oldAppleScript := integrationBrowserAppleScriptFn
	oldListProcesses := integrationBrowserListProcessesFn
	defer func() {
		integrationBrowserStatFn = oldStat
		integrationBrowserAppleScriptFn = oldAppleScript
		integrationBrowserListProcessesFn = oldListProcesses
	}()

	root := t.TempDir()
	chromeApp := filepath.Join(root, "Google Chrome.app")
	if err := os.MkdirAll(chromeApp, 0o755); err != nil {
		t.Fatal(err)
	}

	integrationBrowserStatFn = func(path string) (os.FileInfo, error) {
		if path == "/Applications/Google Chrome.app" {
			return os.Stat(chromeApp)
		}
		return nil, os.ErrNotExist
	}
	integrationBrowserListProcessesFn = func() ([]integrationBrowserProcessInfo, error) {
		return []integrationBrowserProcessInfo{{
			CommandLine: "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		}}, nil
	}
	integrationBrowserAppleScriptFn = func(string) (string, error) {
		return "miss", nil
	}

	handled, err := integrationBrowserOpenOrActivateChromeTab("http://localhost:8080/site/#app")
	if err != nil {
		t.Fatalf("integrationBrowserOpenOrActivateChromeTab: %v", err)
	}
	if handled {
		t.Fatal("expected helper to fall back when matching tab is missing")
	}
}
