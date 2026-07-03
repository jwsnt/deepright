package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBrowserFormatCommandLineKeepsWindowsUserDataDirUnescaped(t *testing.T) {
	got := browserFormatCommandLine(
		"/mnt/c/Program Files/Google/Chrome/Application/chrome.exe",
		[]string{
			"--remote-debugging-port=12345",
			"--remote-debugging-address=0.0.0.0",
			`--user-data-dir=C:\temp\chrome_12345`,
			"--no-first-run",
		},
	)

	want := `"/mnt/c/Program Files/Google/Chrome/Application/chrome.exe" --remote-debugging-port=12345 --remote-debugging-address=0.0.0.0 --user-data-dir="C:\\temp\chrome_12345" --no-first-run`
	if got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
}

func TestBrowserBuildWSLExecCommandKeepsFullChromeCommand(t *testing.T) {
	got := browserBuildWSLExecCommand(
		"/mnt/c/Program Files/Google/Chrome/Application/chrome.exe",
		[]string{
			"--remote-debugging-port=12345",
			"--remote-debugging-address=127.0.0.1",
			`--user-data-dir=C:\temp\chrome_12345`,
			"--no-first-run",
		},
	)

	want := `exec "/mnt/c/Program Files/Google/Chrome/Application/chrome.exe" --remote-debugging-port=12345 --remote-debugging-address=127.0.0.1 --user-data-dir="C:\\temp\chrome_12345" --no-first-run`
	if got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
}

func TestBrowserBuildWSLLaunchScriptStartsChromeWithPowerShellLauncher(t *testing.T) {
	got := browserBuildWSLLaunchScript()
	for _, want := range []string{
		`@echo off`,
		`set "LOG_PATH=%~1"`,
		`set "CHROME_PATH=%~2"`,
		`set "ARG_FILE=%~3"`,
		`-EncodedCommand `,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("script missing %q:\n%s", want, got)
		}
	}
}

func TestBrowserBuildWSLLaunchPowerShellScriptStartsChromeFromArgsFile(t *testing.T) {
	got := browserBuildWSLLaunchPowerShellScript()
	for _, want := range []string{
		"$args = @(Get-Content -LiteralPath $argFile)",
		"$null = & $cmdPath '/d' '/c' 'start' '' $chrome @args",
		"Get-CimInstance Win32_Process -Filter \"Name = 'chrome.exe'\" |",
		"[Console]::Out.WriteLine($proc.ProcessId)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("powershell missing %q:\n%s", want, got)
		}
	}
}

func TestBrowserResolveWSLLaunchScriptPathsUsesWindowsTempRoot(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	browserWSLPathUnixFn = func(path string) (string, error) {
		if path != `C:\temp\browser.cmd` {
			t.Fatalf("unexpected windows script path: %q", path)
		}
		return filepath.Join("/mnt/c/temp", "browser.cmd"), nil
	}

	windowsPath, unixPath, err := browserResolveWSLLaunchScriptPaths()
	if err != nil {
		t.Fatal(err)
	}
	if windowsPath != `C:\temp\browser.cmd` {
		t.Fatalf("windows script path = %q, want %q", windowsPath, `C:\temp\browser.cmd`)
	}
	if unixPath != filepath.Join("/mnt/c/temp", "browser.cmd") {
		t.Fatalf("unix script path = %q, want %q", unixPath, filepath.Join("/mnt/c/temp", "browser.cmd"))
	}
}

func TestBrowserEnsureWSLLaunchScriptWritesScriptFile(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	tempRoot := t.TempDir()
	browserWSLPathUnixFn = func(path string) (string, error) {
		if path != `C:\temp\browser.cmd` {
			t.Fatalf("unexpected windows script path: %q", path)
		}
		return filepath.Join(tempRoot, "browser.cmd"), nil
	}

	scriptPath, err := browserEnsureWSLLaunchScript()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != browserBuildWSLLaunchScript() {
		t.Fatalf("script content = %q, want %q", string(data), browserBuildWSLLaunchScript())
	}
	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("script mode = %o, want 755", info.Mode().Perm())
	}
}

func TestBrowserWriteWSLLaunchArgsFileWritesWindowsStyleArgs(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	tempRoot := t.TempDir()
	browserWSLPathUnixFn = func(path string) (string, error) {
		if path != `C:\temp\browser_args_12345.txt` {
			t.Fatalf("unexpected windows args path: %q", path)
		}
		return filepath.Join(tempRoot, "browser_args_12345.txt"), nil
	}

	windowsPath, err := browserWriteWSLLaunchArgsFile(12345, []string{
		"--remote-debugging-port=12345",
		"--remote-debugging-address=127.0.0.1",
		`--user-data-dir=C:\temp\chrome_12345`,
		"--no-first-run",
	})
	if err != nil {
		t.Fatal(err)
	}
	if windowsPath != `C:\temp\browser_args_12345.txt` {
		t.Fatalf("windows args path = %q, want %q", windowsPath, `C:\temp\browser_args_12345.txt`)
	}
	data, err := os.ReadFile(filepath.Join(tempRoot, "browser_args_12345.txt"))
	if err != nil {
		t.Fatal(err)
	}
	want := "--remote-debugging-port=12345\r\n--remote-debugging-address=127.0.0.1\r\n--user-data-dir=C:\\temp\\chrome_12345\r\n--no-first-run\r\n"
	if string(data) != want {
		t.Fatalf("args file = %q, want %q", string(data), want)
	}
}

func TestBrowserResolveWSLLaunchLogFilePathsUsesWindowsTempRoot(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	browserWSLPathUnixFn = func(path string) (string, error) {
		if path != `C:\temp\browser_stderr_12345.log` {
			t.Fatalf("unexpected windows log path: %q", path)
		}
		return filepath.Join("/mnt/c/temp", "browser_stderr_12345.log"), nil
	}

	windowsPath, unixPath, err := browserResolveWSLLaunchLogFilePaths(12345)
	if err != nil {
		t.Fatal(err)
	}
	if windowsPath != `C:\temp\browser_stderr_12345.log` {
		t.Fatalf("windows log path = %q, want %q", windowsPath, `C:\temp\browser_stderr_12345.log`)
	}
	if unixPath != filepath.Join("/mnt/c/temp", "browser_stderr_12345.log") {
		t.Fatalf("unix log path = %q, want %q", unixPath, filepath.Join("/mnt/c/temp", "browser_stderr_12345.log"))
	}
}

func TestBrowserExtractWSLRemoteDebuggingPort(t *testing.T) {
	got, err := browserExtractWSLRemoteDebuggingPort([]string{
		"--remote-debugging-port=12345",
		"--remote-debugging-address=127.0.0.1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != 12345 {
		t.Fatalf("port = %d, want 12345", got)
	}
}

func TestBrowserParseWSLBackgroundPID(t *testing.T) {
	got, err := browserParseWSLBackgroundPID([]byte("25255\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got != 25255 {
		t.Fatalf("pid = %d, want 25255", got)
	}
}

func TestBrowserWaitForAttachedWSLProcessExitUsesWindowsProcessCheckOnWSL(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	browserWSLDetectFn = func() (bool, error) {
		return true, nil
	}
	probes := 0
	browserWSLProcessExistsFn = func(pid int) (bool, error) {
		if pid != 25255 {
			t.Fatalf("pid = %d, want 25255", pid)
		}
		probes++
		return probes < 3, nil
	}
	browserProcessExistsFn = func(pid int) bool {
		t.Fatalf("unix process probe should not run for WSL pid %d", pid)
		return false
	}
	sleepCalls := 0
	browserSleepFn = func(dur time.Duration) {
		sleepCalls++
	}

	if err := browserWaitForAttachedWSLProcessExit(25255); err != nil {
		t.Fatal(err)
	}
	if probes != 3 {
		t.Fatalf("windows probes = %d, want 3", probes)
	}
	if sleepCalls != 2 {
		t.Fatalf("sleep calls = %d, want 2", sleepCalls)
	}
}

func TestBrowserWaitForAttachedWSLProcessExitFallsBackToUnixProbeOutsideWSL(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	browserWSLDetectFn = func() (bool, error) {
		return false, nil
	}
	browserWSLProcessExistsFn = func(pid int) (bool, error) {
		t.Fatalf("wsl process probe should not run outside WSL for pid %d", pid)
		return false, nil
	}
	probes := 0
	browserProcessExistsFn = func(pid int) bool {
		if pid != 9003 {
			t.Fatalf("pid = %d, want 9003", pid)
		}
		probes++
		return probes < 3
	}
	sleepCalls := 0
	browserSleepFn = func(dur time.Duration) {
		sleepCalls++
	}

	if err := browserWaitForAttachedWSLProcessExit(9003); err != nil {
		t.Fatal(err)
	}
	if probes != 3 {
		t.Fatalf("unix probes = %d, want 3", probes)
	}
	if sleepCalls != 2 {
		t.Fatalf("sleep calls = %d, want 2", sleepCalls)
	}
}

func TestBrowserCleanupWSLChromeUserDataAcceptsWindowsStyleProfileDir(t *testing.T) {
	got, err := browserResolveWSLChromeProfileCleanupDir(`C:\temp\chrome_12345`)
	if err != nil {
		t.Fatal(err)
	}
	if got != `C:\temp\chrome_12345` {
		t.Fatalf("cleanup dir = %q, want %q", got, `C:\temp\chrome_12345`)
	}
}

func TestBrowserCleanupWSLChromeUserDataConvertsUnixProfileDir(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	browserWSLPathWindowsFn = func(path string) (string, error) {
		if path != filepath.Join("/mnt/c/temp", "chrome_12345") {
			t.Fatalf("unexpected unix profile dir: %q", path)
		}
		return `C:\temp\chrome_12345`, nil
	}

	got, err := browserResolveWSLChromeProfileCleanupDir(filepath.Join("/mnt/c/temp", "chrome_12345"))
	if err != nil {
		t.Fatal(err)
	}
	if got != `C:\temp\chrome_12345` {
		t.Fatalf("cleanup dir = %q, want %q", got, `C:\temp\chrome_12345`)
	}
}

func TestBrowserRunWSLBootstrapLifecycleStopsWhenCleanupFails(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	pluginDir := t.TempDir()
	chromePath := filepath.Join(pluginDir, "chrome.exe")
	if err := os.WriteFile(chromePath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}

	browserWSLDetectFn = func() (bool, error) {
		return true, nil
	}
	browserWSLPathUnixFn = func(path string) (string, error) {
		return filepath.Join(pluginDir, "chrome_29876"), nil
	}
	browserPortAvailableFn = func(port int) bool {
		return true
	}
	browserCDPVersionFn = func(port int) (browserCDPVersion, error) {
		return browserCDPVersion{}, errors.New("not ready")
	}
	wantErr := errors.New("cleanup failed")
	browserWSLCleanupChromeUserDataFn = func(profileDir string) error {
		if profileDir != `C:\temp\chrome_29876` {
			t.Fatalf("cleanup profile dir = %q, want %q", profileDir, `C:\temp\chrome_29876`)
		}
		return wantErr
	}

	err := browserRunWSLBootstrapLifecycle(map[string]string{"chrome": chromePath})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}
