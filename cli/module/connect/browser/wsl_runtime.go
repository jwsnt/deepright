package main

import (
	"connect/connectsvc"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

const (
	browserWSLBootstrapPort      = 29876
	browserWSLCommandPromptPath  = "/mnt/c/Windows/System32/cmd.exe"
	browserWSLDefaultChromePath  = "/mnt/c/Program Files/Google/Chrome/Application/chrome.exe"
	browserWSLLaunchLogFileName  = "browser_stderr_%d.log"
	browserWSLWindowsPowerShell  = `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`
	browserWSLPowerShellPath     = "/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe"
	browserWSLLaunchArgsFileName = "browser_args_%d.txt"
	browserWSLLaunchScriptName   = "browser.cmd"
	browserWSLRuntimeDirName     = "deepright"
	browserWSLBootstrapProbeWait = 5 * time.Second
	browserWSLManagedProfileRoot = `C:\temp`
)

var (
	browserWSLDetectFn                = browserIsWSLSystem
	browserWSLPathWindowsFn           = browserWSLPathToWindows
	browserWSLPathUnixFn              = browserWSLPathToUnix
	browserWSLProcessExistsFn         = browserWSLWindowsProcessExists
	browserWSLWindowsPortFreeFn       = browserIsWSLWindowsPortFree
	browserWSLBootstrapLifecycleFn    = browserRunWSLBootstrapLifecycle
	browserWSLAgentRootDirFn          = browserDefaultWSLAgentRoot
	browserWSLBaseUserDataDirFn       = browserDefaultWSLChromeBase
	browserWSLTerminateProcessFn      = browserTerminateWSLWindowsProcess
	browserWSLCommandStartProcessFn   = browserStartAttachedChromeProcess
	browserWSLCommandWaitForExitFn    = browserWaitForChromeProcessExit
	browserWSLCleanupChromeUserDataFn = browserCleanupWSLChromeUserData
)

func browserDefaultWSLRuntimeRoot() string {
	home, err := browserUserHomeDirFn()
	if err != nil || strings.TrimSpace(home) == "" {
		return ""
	}
	return filepath.Join(home, browserWSLRuntimeDirName)
}

func browserDefaultWSLChromeBase() string {
	root := strings.TrimSpace(browserDefaultWSLRuntimeRoot())
	if root == "" {
		return ""
	}
	return filepath.Join(root, "chrome_base")
}

func browserDefaultWSLAgentRoot() string {
	root := strings.TrimSpace(browserDefaultWSLRuntimeRoot())
	if root == "" {
		return ""
	}
	return filepath.Join(root, "agent")
}

func browserDestroyUsesDefaultStartPort(flags map[string]string) bool {
	agentID := strings.TrimSpace(connectsvc.FirstValue(flags, "agentId", "agent"))
	chatID := strings.TrimSpace(connectsvc.FirstValue(flags, "chatId", "chat"))
	return agentID == "" && chatID == ""
}

func browserRunWSLBootstrapLifecycle(flags map[string]string) error {
	isWSL, err := browserWSLDetectFn()
	if err != nil || !isWSL {
		return err
	}

	chromePath, err := browserResolveChromePath(flags)
	if err != nil {
		browserLogWSLLifecycleEvent("browser_start_wsl_cdp", "resolve_chrome_error", nil, err)
		return err
	}
	profilePaths, err := browserResolveWSLChromeProfileDir(browserWSLBootstrapPort)
	if err != nil {
		browserLogWSLLifecycleEvent("browser_start_wsl_cdp", "resolve_profile_error", map[string]any{
			"chromePath": chromePath,
		}, err)
		return err
	}
	profileDir := profilePaths.Local
	launchProfileDir := profilePaths.Launch
	if err := browserMkdirAllFn(profileDir, 0o755); err != nil {
		browserLogWSLLifecycleEvent("browser_start_wsl_cdp", "mkdir_profile_error", map[string]any{
			"chromePath": chromePath,
			"profileDir": profileDir,
		}, err)
		return err
	}

	args := browserWSLBootstrapLaunchArgs(launchProfileDir)
	commandLine := browserFormatCommandLine(chromePath, args)
	if cdp, cdpErr := browserResolveLiveCDPEndpoint(browserWSLBootstrapPort); cdpErr == nil && strings.TrimSpace(cdp) != "" {
		browserLogWSLLifecycleEvent("browser_start_wsl_cdp", "already_ready", map[string]any{
			"chromePath":       chromePath,
			"profileDir":       profileDir,
			"launchProfileDir": launchProfileDir,
			"port":             browserWSLBootstrapPort,
			"cdp":              cdp,
			"command":          commandLine,
		}, nil)
		return browserWaitForLifecycleCDPExit(browserWSLBootstrapPort)
	}
	if !browserPortAvailableFn(browserWSLBootstrapPort) {
		err := fmt.Errorf("port %d is already in use by a non-CDP process", browserWSLBootstrapPort)
		browserLogWSLLifecycleEvent("browser_start_wsl_cdp", "port_busy", map[string]any{
			"chromePath":       chromePath,
			"profileDir":       profileDir,
			"launchProfileDir": launchProfileDir,
			"port":             browserWSLBootstrapPort,
			"command":          commandLine,
		}, err)
		return err
	}
	if err := browserWSLCleanupChromeUserDataFn(launchProfileDir); err != nil {
		browserLogWSLLifecycleEvent("browser_start_wsl_cdp", "cleanup_error", map[string]any{
			"chromePath":       chromePath,
			"profileDir":       profileDir,
			"launchProfileDir": launchProfileDir,
			"port":             browserWSLBootstrapPort,
			"command":          commandLine,
		}, err)
		return err
	}

	browserLogWSLLifecycleEvent("browser_start_wsl_cdp", "launch_command", map[string]any{
		"chromePath":       chromePath,
		"profileDir":       profileDir,
		"launchProfileDir": launchProfileDir,
		"port":             browserWSLBootstrapPort,
		"command":          commandLine,
	}, nil)
	pid, waitFn, err := browserWSLCommandStartProcessFn(chromePath, args)
	if err != nil {
		browserLogWSLLifecycleEvent("browser_start_wsl_cdp", "start_error", map[string]any{
			"chromePath":       chromePath,
			"profileDir":       profileDir,
			"launchProfileDir": launchProfileDir,
			"port":             browserWSLBootstrapPort,
			"command":          commandLine,
		}, err)
		return err
	}
	if waitFn == nil {
		waitFn = func() error { return nil }
	}
	if err := browserWaitForPortFn(pid, browserWSLBootstrapPort, browserWSLBootstrapProbeWait); err != nil {
		_ = browserTerminateProcessFn(pid)
		_ = waitFn()
		browserLogWSLLifecycleEvent("browser_start_wsl_cdp", "probe_error", map[string]any{
			"chromePath":       chromePath,
			"profileDir":       profileDir,
			"launchProfileDir": launchProfileDir,
			"port":             browserWSLBootstrapPort,
			"pid":              pid,
			"command":          commandLine,
		}, err)
		return err
	}
	cdp, err := browserResolveLiveCDPEndpoint(browserWSLBootstrapPort)
	if err != nil {
		_ = browserTerminateProcessFn(pid)
		_ = waitFn()
		browserLogWSLLifecycleEvent("browser_start_wsl_cdp", "resolve_cdp_error", map[string]any{
			"chromePath":       chromePath,
			"profileDir":       profileDir,
			"launchProfileDir": launchProfileDir,
			"port":             browserWSLBootstrapPort,
			"pid":              pid,
			"command":          commandLine,
		}, err)
		return err
	}
	browserLogWSLLifecycleEvent("browser_start_wsl_cdp", "ready", map[string]any{
		"chromePath":       chromePath,
		"profileDir":       profileDir,
		"launchProfileDir": launchProfileDir,
		"port":             browserWSLBootstrapPort,
		"pid":              pid,
		"cdp":              cdp,
		"command":          commandLine,
	}, nil)
	waitErr := browserWSLCommandWaitForExitFn(waitFn)
	browserLogWSLLifecycleEvent("browser_start_wsl_cdp", "closed", map[string]any{
		"chromePath":       chromePath,
		"profileDir":       profileDir,
		"launchProfileDir": launchProfileDir,
		"port":             browserWSLBootstrapPort,
		"pid":              pid,
		"cdp":              cdp,
		"command":          commandLine,
	}, waitErr)
	return nil
}

func browserShutdownDefaultBootstrapCDP(flags map[string]string) error {
	statePath, err := browserResolveStatePath(flags)
	if err != nil {
		statePath = ""
	}
	cdp, resolveErr := browserResolveLiveCDPEndpoint(browserWSLBootstrapPort)
	if resolveErr != nil || strings.TrimSpace(cdp) == "" {
		if statePath != "" {
			_ = browserRemoveInstancesByPort(statePath, browserWSLBootstrapPort)
		}
		return nil
	}

	item := browserInstanceRecord{
		Port: browserWSLBootstrapPort,
		CDP:  cdp,
	}
	if pid, pidErr := browserPortPIDLookupFn(browserWSLBootstrapPort); pidErr == nil {
		item.PID = pid
	}
	if err := browserTerminateManagedInstanceFn(item); err != nil {
		browserLogWSLLifecycleEvent("browser_shutdown_wsl_cdp", "terminate_error", map[string]any{
			"port": browserWSLBootstrapPort,
			"pid":  item.PID,
			"cdp":  cdp,
		}, err)
		return err
	}
	if statePath != "" {
		if err := browserRemoveInstancesByPort(statePath, browserWSLBootstrapPort); err != nil && !os.IsNotExist(err) {
			browserLogWSLLifecycleEvent("browser_shutdown_wsl_cdp", "cleanup_state_error", map[string]any{
				"port":      browserWSLBootstrapPort,
				"pid":       item.PID,
				"cdp":       cdp,
				"statePath": statePath,
			}, err)
			return err
		}
	}
	browserLogWSLLifecycleEvent("browser_shutdown_wsl_cdp", "closed", map[string]any{
		"port":      browserWSLBootstrapPort,
		"pid":       item.PID,
		"cdp":       cdp,
		"statePath": statePath,
	}, nil)
	return nil
}

func browserResolveWSLBaseUserDataDir() (string, error) {
	raw := strings.TrimSpace(browserWSLBaseUserDataDirFn())
	if raw == "" {
		return "", errors.New("wsl chrome base dir is required")
	}
	if !filepath.IsAbs(raw) {
		return "", fmt.Errorf("wsl chrome base dir must be an absolute path: %q", raw)
	}
	return raw, nil
}

func browserResolveWSLCreateSourceDir() (string, string, error) {
	sourceDir, err := browserResolveWSLBaseUserDataDir()
	if err != nil {
		return "", "", err
	}
	info, err := os.Stat(sourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "wsl_base_missing", nil
		}
		return "", "", err
	}
	if !info.IsDir() {
		return "", "", fmt.Errorf("wsl chrome base dir exists and is not a directory: %s", sourceDir)
	}
	entries, err := browserReadDirFn(sourceDir)
	if err != nil {
		return "", "", err
	}
	if len(entries) == 0 {
		return "", "wsl_base_empty", nil
	}
	return sourceDir, "wsl_base", nil
}

func browserResolveWSLChromeProfileDir(port int) (browserChromeProfilePaths, error) {
	if port <= 0 {
		return browserChromeProfilePaths{}, fmt.Errorf("invalid WSL profile port: %d", port)
	}
	launchProfileDir := browserWSLManagedChromeProfileDir(port)
	profileDir, err := browserWSLPathUnixFn(launchProfileDir)
	if err != nil {
		return browserChromeProfilePaths{}, err
	}
	return browserChromeProfilePaths{
		Local:  profileDir,
		Launch: launchProfileDir,
	}, nil
}

func browserCleanupWSLAgentUserData() error {
	cleanupRoot, err := browserResolveWSLManagedProfileCleanupRoot()
	if err != nil {
		return err
	}
	if cleanupRoot == "" {
		return nil
	}
	_, err = browserDestroyCleanupManagedChromeProfileDirs(cleanupRoot)
	return err
}

func browserWSLBootstrapLaunchArgs(launchProfileDir string) []string {
	return []string{
		"--remote-debugging-port=" + strconv.Itoa(browserWSLBootstrapPort),
		"--remote-debugging-address=127.0.0.1",
		"--user-data-dir=" + launchProfileDir,
		"--no-first-run",
		"about:blank",
	}
}

func browserStartAttachedChromeProcess(chromePath string, args []string) (int, func() error, error) {
	port, err := browserExtractWSLRemoteDebuggingPort(args)
	if err != nil {
		return 0, nil, err
	}
	windowsScriptPath, scriptPath, err := browserResolveWSLLaunchScriptPaths()
	if err != nil {
		return 0, nil, err
	}
	if err := browserWriteWSLLaunchScript(scriptPath); err != nil {
		return 0, nil, err
	}
	argsFilePath, err := browserWriteWSLLaunchArgsFile(port, args)
	if err != nil {
		return 0, nil, err
	}
	windowsLogPath, _, err := browserResolveWSLLaunchLogFilePaths(port)
	if err != nil {
		return 0, nil, err
	}
	windowsChromePath, err := browserResolveWSLWindowsPath(chromePath)
	if err != nil {
		return 0, nil, err
	}
	cmd := exec.Command(browserWSLCommandPromptPath, "/C", windowsScriptPath, windowsLogPath, windowsChromePath, argsFilePath)
	cmd.Dir = filepath.Dir(scriptPath)
	output, err := cmd.CombinedOutput()
	waitFn := func() error {
		return nil
	}
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return 0, waitFn, err
		}
		return 0, waitFn, fmt.Errorf("%w: %s", err, message)
	}
	pid, err := browserParseWSLBackgroundPID(output)
	if err != nil {
		return 0, waitFn, err
	}
	waitFn = func() error {
		return browserWaitForAttachedWSLProcessExit(pid)
	}
	return pid, waitFn, nil
}

func browserBuildWSLExecCommand(chromePath string, args []string) string {
	return "exec " + browserFormatCommandLine(chromePath, args)
}

func browserResolveWSLWindowsPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("wsl path is required")
	}
	if len(path) >= 3 && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
		return strings.ReplaceAll(path, "/", `\`), nil
	}
	return browserWSLPathWindowsFn(path)
}

func browserResolveWSLLaunchScriptPaths() (string, string, error) {
	return browserResolveWSLManagedWindowsFile(browserWSLLaunchScriptName)
}

func browserResolveWSLLaunchArgsFilePaths(port int) (string, string, error) {
	if port <= 0 {
		return "", "", fmt.Errorf("invalid wsl launch port: %d", port)
	}
	return browserResolveWSLManagedWindowsFile(fmt.Sprintf(browserWSLLaunchArgsFileName, port))
}

func browserResolveWSLLaunchLogFilePaths(port int) (string, string, error) {
	if port <= 0 {
		return "", "", fmt.Errorf("invalid wsl launch port: %d", port)
	}
	return browserResolveWSLManagedWindowsFile(fmt.Sprintf(browserWSLLaunchLogFileName, port))
}

func browserResolveWSLManagedWindowsFile(name string) (string, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", fmt.Errorf("wsl managed file name is required")
	}
	windowsPath := browserWSLManagedProfileRoot + `\` + name
	unixPath, err := browserWSLPathUnixFn(windowsPath)
	if err != nil {
		return "", "", err
	}
	return windowsPath, unixPath, nil
}

func browserBuildWSLLaunchScript() string {
	encoded := browserWSLEncodePowerShellCommand(browserBuildWSLLaunchPowerShellScript())
	return strings.Join([]string{
		"@echo off",
		"setlocal EnableExtensions",
		`if "%~3"=="" exit /b 1`,
		`set "LOG_PATH=%~1"`,
		`set "CHROME_PATH=%~2"`,
		`set "ARG_FILE=%~3"`,
		browserWSLWindowsPowerShell + " -NoProfile -NonInteractive -EncodedCommand " + encoded,
		"set \"CODE=%ERRORLEVEL%\"",
		"endlocal & exit /b %CODE%",
		"",
	}, "\r\n")
}

func browserBuildWSLLaunchPowerShellScript() string {
	return strings.Join([]string{
		"$ErrorActionPreference = 'Stop'",
		"$chrome = $env:CHROME_PATH",
		"$argFile = $env:ARG_FILE",
		"$args = @(Get-Content -LiteralPath $argFile)",
		"$portArg = $args | Where-Object { $_ -like '--remote-debugging-port=*' } | Select-Object -First 1",
		"$userDataArg = $args | Where-Object { $_ -like '--user-data-dir=*' } | Select-Object -First 1",
		"if ([string]::IsNullOrWhiteSpace($portArg) -or [string]::IsNullOrWhiteSpace($userDataArg)) { exit 1 }",
		"$cmdPath = $env:ComSpec",
		"if ([string]::IsNullOrWhiteSpace($cmdPath)) { $cmdPath = 'C:\\Windows\\System32\\cmd.exe' }",
		"$null = & $cmdPath '/d' '/c' 'start' '' $chrome @args",
		"$deadline = (Get-Date).AddSeconds(10)",
		"do {",
		"  $proc = Get-CimInstance Win32_Process -Filter \"Name = 'chrome.exe'\" |",
		"    Where-Object { $_.CommandLine -like ('*' + $portArg + '*') -and $_.CommandLine -like ('*' + $userDataArg + '*') } |",
		"    Sort-Object CreationDate -Descending |",
		"    Select-Object -First 1",
		"  if ($null -ne $proc) {",
		"    [Console]::Out.WriteLine($proc.ProcessId)",
		"    exit 0",
		"  }",
		"  Start-Sleep -Milliseconds 250",
		"} while ((Get-Date) -lt $deadline)",
		"exit 1",
		"",
	}, "\n")
}

func browserEnsureWSLLaunchScript() (string, error) {
	_, scriptPath, err := browserResolveWSLLaunchScriptPaths()
	if err != nil {
		return "", err
	}
	if err := browserWriteWSLLaunchScript(scriptPath); err != nil {
		return "", err
	}
	return scriptPath, nil
}

func browserWriteWSLLaunchScript(scriptPath string) error {
	if err := browserMkdirAllFn(filepath.Dir(scriptPath), 0o755); err != nil {
		return err
	}
	if err := browserWriteFileFn(scriptPath, []byte(browserBuildWSLLaunchScript()), 0o755); err != nil {
		return err
	}
	return nil
}

func browserWriteWSLLaunchArgsFile(port int, args []string) (string, error) {
	windowsPath, unixPath, err := browserResolveWSLLaunchArgsFilePaths(port)
	if err != nil {
		return "", err
	}
	if err := browserMkdirAllFn(filepath.Dir(unixPath), 0o755); err != nil {
		return "", err
	}
	data := strings.Join(args, "\r\n")
	if data != "" {
		data += "\r\n"
	}
	if err := browserWriteFileFn(unixPath, []byte(data), 0o644); err != nil {
		return "", err
	}
	return windowsPath, nil
}

func browserExtractWSLRemoteDebuggingPort(args []string) (int, error) {
	for _, arg := range args {
		const prefix = "--remote-debugging-port="
		if !strings.HasPrefix(strings.TrimSpace(arg), prefix) {
			continue
		}
		port, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(arg, prefix)))
		if err != nil || port <= 0 {
			return 0, fmt.Errorf("invalid remote debugging port argument: %q", arg)
		}
		return port, nil
	}
	return 0, fmt.Errorf("missing remote debugging port argument")
}

func browserParseWSLBackgroundPID(output []byte) (int, error) {
	fields := strings.Fields(strings.TrimSpace(string(output)))
	if len(fields) == 0 {
		return 0, fmt.Errorf("wsl attached chrome launch returned empty pid output")
	}
	pid, err := strconv.Atoi(fields[len(fields)-1])
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("invalid wsl attached chrome pid output: %q", strings.TrimSpace(string(output)))
	}
	return pid, nil
}

func browserWaitForAttachedWSLProcessExit(pid int) error {
	if pid <= 0 {
		return nil
	}
	if isWSL, err := browserWSLDetectFn(); err == nil && isWSL {
		for {
			alive, existsErr := browserWSLProcessExistsFn(pid)
			if existsErr != nil {
				return existsErr
			}
			if !alive {
				return nil
			}
			browserSleepFn(time.Second)
		}
	}
	for browserProcessExistsFn(pid) {
		browserSleepFn(time.Second)
	}
	return nil
}

func browserStartAttachedChromeProcessDirect(chromePath string, args []string, logPath string) (int, func() error, error) {
	cmd := exec.Command(chromePath, args...)
	cmd.Dir = filepath.Dir(chromePath)
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return 0, nil, err
	}
	logWriter, cleanup, err := browserStartChromeLogFilterFn(logPath)
	if err == nil {
		cmd.Stdout = logWriter
		cmd.Stderr = logWriter
	} else {
		logFile, fileErr := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if fileErr != nil {
			devNull.Close()
			return 0, nil, fileErr
		}
		logWriter = logFile
		cleanup = func() {
			_ = logFile.Close()
		}
	}
	cmd.Stdin = devNull
	browserPrepareAttachedCommand(cmd)
	if err := cmd.Start(); err != nil {
		cleanup()
		_ = devNull.Close()
		return 0, nil, err
	}
	waitFn := func() error {
		defer cleanup()
		defer devNull.Close()
		return cmd.Wait()
	}
	return cmd.Process.Pid, waitFn, nil
}

func browserWaitForChromeProcessExit(waitFn func() error) error {
	if waitFn == nil {
		return nil
	}
	return waitFn()
}

func browserWaitForLifecycleCDPExit(port int) error {
	interval := browserLifecycleProbeInterval
	if interval <= 0 {
		interval = time.Second
	}
	for {
		if _, err := browserResolveLiveCDPEndpoint(port); err != nil {
			return nil
		}
		browserSleepFn(interval)
	}
}

func browserFormatCommandLine(executable string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	if trimmed := strings.TrimSpace(executable); trimmed != "" {
		parts = append(parts, browserQuoteCommandPart(trimmed))
	}
	for _, arg := range args {
		parts = append(parts, browserQuoteCommandPart(arg))
	}
	return strings.Join(parts, " ")
}

func browserQuoteCommandPart(part string) string {
	part = strings.TrimSpace(part)
	if part == "" {
		return `""`
	}
	if formatted, ok := browserFormatSpecialCommandPart(part); ok {
		return formatted
	}
	if !strings.ContainsAny(part, " \t\n\r\"'") {
		return part
	}
	return strconv.Quote(part)
}

func browserFormatSpecialCommandPart(part string) (string, bool) {
	const userDataPrefix = "--user-data-dir="
	if !strings.HasPrefix(part, userDataPrefix) {
		return "", false
	}
	path := strings.TrimSpace(strings.TrimPrefix(part, userDataPrefix))
	if path == "" {
		return "", false
	}
	return userDataPrefix + `"` + browserFormatWindowsCommandPath(path) + `"`, true
}

func browserFormatWindowsCommandPath(path string) string {
	path = strings.TrimSpace(path)
	if len(path) >= 3 && path[1] == ':' && path[2] == '\\' {
		return path[:3] + `\` + path[3:]
	}
	return path
}

func browserIsWSLSystem() (bool, error) {
	if runtime.GOOS != "linux" {
		return false, nil
	}
	for _, key := range []string{"WSL_DISTRO_NAME", "WSL_INTEROP"} {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return true, nil
		}
	}
	for _, candidate := range []string{"/proc/sys/kernel/osrelease", "/proc/version"} {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		text := strings.ToLower(strings.TrimSpace(string(data)))
		if strings.Contains(text, "microsoft") || strings.Contains(text, "wsl") {
			return true, nil
		}
	}
	return false, nil
}

func browserWSLPathToWindows(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("wsl path is empty")
	}
	output, err := exec.Command("wslpath", "-w", path).Output()
	if err != nil {
		return "", fmt.Errorf("wslpath -w %q: %w", path, err)
	}
	windowsPath := strings.TrimSpace(string(output))
	if windowsPath == "" {
		return "", fmt.Errorf("wslpath returned empty path for %s", path)
	}
	return windowsPath, nil
}

func browserWSLPathToUnix(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("windows path is empty")
	}
	output, err := exec.Command("wslpath", "-u", path).Output()
	if err != nil {
		return "", fmt.Errorf("wslpath -u %q: %w", path, err)
	}
	unixPath := strings.TrimSpace(string(output))
	if unixPath == "" {
		return "", fmt.Errorf("wslpath returned empty path for %s", path)
	}
	return unixPath, nil
}

func browserWSLManagedChromeProfileDir(port int) string {
	return browserWSLManagedProfileRoot + `\chrome_` + strconv.Itoa(port)
}

func browserCleanupWSLChromeUserData(profileDir string) error {
	windowsProfileDir, err := browserResolveWSLChromeProfileCleanupDir(profileDir)
	if err != nil {
		return err
	}
	script := browserWSLChromeLockCleanupScript(windowsProfileDir)
	output, err := exec.Command(
		browserWSLPowerShellPath,
		"-NoProfile",
		"-NonInteractive",
		"-EncodedCommand",
		browserWSLEncodePowerShellCommand(script),
	).CombinedOutput()
	if err == nil {
		return nil
	}
	message := strings.TrimSpace(string(output))
	if message == "" {
		return fmt.Errorf("cleanup chrome lock files failed for %s: %w", windowsProfileDir, err)
	}
	return fmt.Errorf("cleanup chrome lock files failed for %s: %w: %s", windowsProfileDir, err, message)
}

func browserResolveWSLChromeProfileCleanupDir(profileDir string) (string, error) {
	profileDir = strings.TrimSpace(profileDir)
	if profileDir == "" {
		return "", fmt.Errorf("wsl chrome profile dir is required")
	}
	if len(profileDir) >= 3 && profileDir[1] == ':' && (profileDir[2] == '\\' || profileDir[2] == '/') {
		return strings.ReplaceAll(profileDir, "/", `\`), nil
	}
	windowsProfileDir, err := browserWSLPathWindowsFn(profileDir)
	if err != nil {
		return "", err
	}
	windowsProfileDir = strings.TrimSpace(windowsProfileDir)
	if windowsProfileDir == "" {
		return "", fmt.Errorf("resolved empty windows profile dir for %s", profileDir)
	}
	return strings.ReplaceAll(windowsProfileDir, "/", `\`), nil
}

func browserWSLChromeLockCleanupScript(profileDir string) string {
	lockNames := browserChromeUserDataLockNames("windows")
	quotedNames := make([]string, 0, len(lockNames))
	for _, name := range lockNames {
		quotedNames = append(quotedNames, "'"+strings.ReplaceAll(strings.TrimSpace(name), "'", "''")+"'")
	}
	return fmt.Sprintf(`$ErrorActionPreference = 'Stop'
$profileDir = '%s'
if (-not (Test-Path -LiteralPath $profileDir)) { exit 0 }
$lockNames = @(%s)
$targets = Get-ChildItem -LiteralPath $profileDir -Force -Recurse -ErrorAction SilentlyContinue |
  Where-Object {
    $lockNames -contains $_.Name -or
    $_.Name -like '*.lock' -or
    $_.Name -like '*-journal'
  } |
  Select-Object -ExpandProperty FullName -Unique |
  Sort-Object Length -Descending
foreach ($target in $targets) {
  Remove-Item -LiteralPath $target -Recurse -Force -ErrorAction Stop
  if (Test-Path -LiteralPath $target) {
    throw "failed to remove lock entry: $target"
  }
}
`, strings.ReplaceAll(profileDir, "'", "''"), strings.Join(quotedNames, ", "))
}

func browserWSLEncodePowerShellCommand(script string) string {
	encoded := utf16.Encode([]rune(script))
	bytes := make([]byte, 0, len(encoded)*2)
	for _, value := range encoded {
		bytes = append(bytes, byte(value), byte(value>>8))
	}
	return base64.StdEncoding.EncodeToString(bytes)
}

func browserResolveWSLManagedProfileCleanupRoot() (string, error) {
	return browserWSLPathUnixFn(browserWSLManagedProfileRoot)
}

func browserWSLWindowsProcessExists(pid int) (bool, error) {
	if pid <= 0 {
		return false, nil
	}
	script := fmt.Sprintf(`$proc = Get-CimInstance Win32_Process -Filter "ProcessId = %d" -ErrorAction SilentlyContinue
if ($null -eq $proc) {
  [Console]::Out.Write("missing")
  exit 0
}
[Console]::Out.Write("running")
`, pid)
	output, err := exec.Command(
		browserWSLPowerShellPath,
		"-NoProfile",
		"-NonInteractive",
		"-Command",
		script,
	).Output()
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(strings.ReplaceAll(string(output), "\r", ""))) {
	case "", "missing":
		return false, nil
	case "running":
		return true, nil
	default:
		return false, fmt.Errorf("unexpected wsl process check output for pid %d: %q", pid, strings.TrimSpace(string(output)))
	}
}

func browserTerminateWSLManagedInstance(item browserInstanceRecord) error {
	pid := item.PID
	if pid <= 0 && item.Port > 0 {
		if livePID, ok := browserWSLInstanceLookupPIDByPort(item.Port); ok {
			pid = livePID
		}
	}
	if pid <= 0 {
		if item.Port > 0 {
			if free, err := browserWSLWindowsPortFreeFn(item.Port); err == nil && free {
				browserWSLBestEffortCleanupProfileLocks(item)
				return nil
			}
		}
		return fmt.Errorf("wsl instance pid not found for port %d", item.Port)
	}
	if err := browserWSLTerminateProcessFn(pid, item.Port); err != nil {
		return err
	}
	browserWSLBestEffortCleanupProfileLocks(item)
	return nil
}

func browserTerminateWSLWindowsProcess(pid, port int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid wsl windows pid: %d", pid)
	}
	script := fmt.Sprintf(`$ErrorActionPreference = 'Stop'
try {
  Stop-Process -Id %d -Force -ErrorAction Stop
} catch {
  if ($_.FullyQualifiedErrorId -like '*NoProcessFoundForGivenId*') { exit 0 }
  throw
}
`, pid)
	output, err := exec.Command(
		browserWSLPowerShellPath,
		"-NoProfile",
		"-NonInteractive",
		"-Command",
		script,
	).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(strings.ReplaceAll(string(output), "\r", ""))
		if message == "" {
			return fmt.Errorf("wsl Stop-Process -Id %d -Force failed: %w", pid, err)
		}
		return fmt.Errorf("wsl Stop-Process -Id %d -Force failed: %w: %s", pid, err, message)
	}
	return browserWaitForWSLWindowsProcessClosed(pid, port, 5*time.Second)
}

func browserWaitForWSLWindowsProcessClosed(pid, port int, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	deadline := time.Now().Add(timeout)
	for {
		pidAlive, pidErr := browserWSLNetstatContainsPID(pid)
		if pidErr != nil {
			return pidErr
		}
		portBusy := false
		if port > 0 {
			free, freeErr := browserWSLWindowsPortFreeFn(port)
			if freeErr != nil {
				return freeErr
			}
			portBusy = !free
		}
		if !pidAlive && !portBusy {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("wsl process or port still active after Stop-Process: pid=%d port=%d", pid, port)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func browserWSLNetstatContainsPID(pid int) (bool, error) {
	if pid <= 0 {
		return false, nil
	}
	cmdText := fmt.Sprintf("netstat -ano | findstr %d", pid)
	output, err := exec.Command(browserWSLCommandPromptPath, "/C", cmdText).CombinedOutput()
	text := strings.TrimSpace(strings.ReplaceAll(string(output), "\r", ""))
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return false, nil
		}
		if text == "" {
			return false, fmt.Errorf("cmd /C %q failed: %w", cmdText, err)
		}
		return false, fmt.Errorf("cmd /C %q failed: %w: %s", cmdText, err, text)
	}
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		if fields[len(fields)-1] == strconv.Itoa(pid) {
			return true, nil
		}
	}
	return false, nil
}

func browserWSLBestEffortCleanupProfileLocks(item browserInstanceRecord) {
	profileDir := strings.TrimSpace(item.ProfileDir)
	if profileDir == "" {
		if value, ok := browserWSLInstanceLookupUserDataDir(item.AgentID, item.ChatID); ok {
			profileDir = value
		}
	}
	if profileDir == "" {
		return
	}
	if err := browserWSLCleanupChromeUserDataFn(profileDir); err != nil {
		browserShutdownTrace("instance.shutdown.lock_cleanup.warn", map[string]any{
			"agentId":    item.AgentID,
			"chatId":     item.ChatID,
			"pid":        item.PID,
			"port":       item.Port,
			"profileDir": profileDir,
			"error":      err.Error(),
		})
		return
	}
	browserShutdownTrace("instance.shutdown.lock_cleanup.ok", map[string]any{
		"agentId":    item.AgentID,
		"chatId":     item.ChatID,
		"pid":        item.PID,
		"port":       item.Port,
		"profileDir": profileDir,
	})
}

func browserIsWSLWindowsPortFree(port int) (bool, error) {
	if port <= 0 {
		return false, fmt.Errorf("invalid port: %d", port)
	}
	script := fmt.Sprintf(`$items = @(Get-NetTCPConnection -State Listen -LocalPort %d -ErrorAction SilentlyContinue); if ($items.Count -gt 0) { "busy" } else { "free" }`, port)
	output, err := exec.Command(browserWSLPowerShellPath, "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(string(output))) {
	case "free":
		return true, nil
	case "busy":
		return false, nil
	default:
		return false, fmt.Errorf("unexpected powershell port check output for %d: %q", port, strings.TrimSpace(string(output)))
	}
}

func browserLogWSLLifecycleEvent(event, stage string, fields map[string]any, err error) {
	payload := map[string]any{
		"event":     strings.TrimSpace(event),
		"stage":     strings.TrimSpace(stage),
		"timestamp": browserNowFn().Format(time.RFC3339Nano),
	}
	for key, value := range fields {
		payload[key] = value
	}
	if err != nil {
		payload["error"] = err.Error()
	}
	browserAppendLogJSON(payload)
}
