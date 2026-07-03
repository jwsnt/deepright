package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

var (
	integrationBrowserLookPathFn      = exec.LookPath
	integrationBrowserStatFn          = os.Stat
	integrationBrowserGetenvFn        = os.Getenv
	integrationBrowserReadFileFn      = os.ReadFile
	integrationBrowserListProcessesFn = integrationBrowserListProcesses
	integrationBrowserStartFn         = func(cmdPath string, args ...string) error {
		return exec.Command(cmdPath, args...).Start()
	}
	integrationBrowserAppleScriptFn = func(script string) error {
		cmd := exec.Command("osascript")
		cmd.Stdin = strings.NewReader(script)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			message := strings.TrimSpace(stderr.String())
			if message != "" {
				return fmt.Errorf("run osascript: %w: %s", err, message)
			}
			return fmt.Errorf("run osascript: %w", err)
		}
		return nil
	}
)

type integrationBrowserProcessInfo struct {
	CommandLine string
}

func openIntegrationBrowserMaximized(target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return errors.New("browser target is empty")
	}
	if runtime.GOOS == "darwin" {
		handled, err := integrationBrowserOpenOrActivateChromeTab(target)
		if handled {
			return err
		}
	}
	cmdPath, args, err := integrationBrowserOpenCommand(runtime.GOOS, target)
	if err != nil {
		return err
	}
	return integrationBrowserStartFn(cmdPath, args...)
}

func integrationBrowserOpenCommand(goos, target string) (string, []string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", nil, errors.New("browser target is empty")
	}

	goos = strings.TrimSpace(goos)
	switch goos {
	case "darwin":
		if appPath, ok := integrationBrowserFirstExistingApp(
			"/Applications/Google Chrome.app",
			"/Applications/Google Chrome for Testing.app",
			"/Applications/Microsoft Edge.app",
			"/Applications/Brave Browser.app",
			"/Applications/Chromium.app",
		); ok {
			return "open", []string{"-a", appPath, target}, nil
		}
		return "open", []string{target}, nil
	case "linux":
		if integrationBrowserIsWSL() {
			return integrationBrowserOpenCommandWSL(target)
		}
		if path, ok := integrationBrowserFirstLookPath(
			"google-chrome",
			"google-chrome-stable",
			"chromium-browser",
			"chromium",
			"microsoft-edge",
			"microsoft-edge-stable",
			"brave-browser",
		); ok {
			return path, []string{"--start-maximized", target}, nil
		}
		return "xdg-open", []string{target}, nil
	case "windows":
		return integrationBrowserOpenCommandWindows(target)
	default:
		return "", nil, errors.New("unsupported OS")
	}
}

func integrationBrowserOpenCommandWSL(target string) (string, []string, error) {
	if path, ok := integrationBrowserFirstExistingFile(
		"/mnt/c/Program Files/Google/Chrome/Application/chrome.exe",
		"/mnt/c/Program Files (x86)/Google/Chrome/Application/chrome.exe",
		"/mnt/c/Program Files/Google/Chrome for Testing/Application/chrome.exe",
		"/mnt/c/Program Files (x86)/Google/Chrome for Testing/Application/chrome.exe",
		"/mnt/c/Program Files/Microsoft/Edge/Application/msedge.exe",
		"/mnt/c/Program Files (x86)/Microsoft/Edge/Application/msedge.exe",
		"/mnt/c/Program Files/Chromium/Application/chrome.exe",
		"/mnt/c/Program Files (x86)/Chromium/Application/chrome.exe",
	); ok {
		return path, []string{"--start-maximized", target}, nil
	}
	if path, ok := integrationBrowserFirstLookPath(
		"chrome",
		"chrome.exe",
		"msedge",
		"msedge.exe",
		"chromium",
		"chromium.exe",
		"brave",
		"brave.exe",
	); ok {
		return path, []string{"--start-maximized", target}, nil
	}
	return "cmd.exe", []string{"/c", "start", "/max", target}, nil
}

func integrationBrowserOpenCommandWindows(target string) (string, []string, error) {
	if path, ok := integrationBrowserFirstExistingFile(
		filepath.Join(integrationBrowserGetenvFn("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(integrationBrowserGetenvFn("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(integrationBrowserGetenvFn("LocalAppData"), "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(integrationBrowserGetenvFn("ProgramFiles"), "Google", "Chrome for Testing", "Application", "chrome.exe"),
		filepath.Join(integrationBrowserGetenvFn("ProgramFiles(x86)"), "Google", "Chrome for Testing", "Application", "chrome.exe"),
		filepath.Join(integrationBrowserGetenvFn("LocalAppData"), "Google", "Chrome for Testing", "Application", "chrome.exe"),
		filepath.Join(integrationBrowserGetenvFn("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(integrationBrowserGetenvFn("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(integrationBrowserGetenvFn("LocalAppData"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(integrationBrowserGetenvFn("ProgramFiles"), "Chromium", "Application", "chrome.exe"),
		filepath.Join(integrationBrowserGetenvFn("ProgramFiles(x86)"), "Chromium", "Application", "chrome.exe"),
		filepath.Join(integrationBrowserGetenvFn("LocalAppData"), "Chromium", "Application", "chrome.exe"),
	); ok {
		return path, []string{"--start-maximized", target}, nil
	}
	if path, ok := integrationBrowserFirstLookPath("chrome", "msedge", "chromium", "brave"); ok {
		return path, []string{"--start-maximized", target}, nil
	}
	return "cmd", []string{"/c", "start", "/max", target}, nil
}

func integrationBrowserIsWSL() bool {
	if strings.TrimSpace(integrationBrowserGetenvFn("WSL_DISTRO_NAME")) != "" {
		return true
	}
	if strings.TrimSpace(integrationBrowserGetenvFn("WSL_INTEROP")) != "" {
		return true
	}
	data, err := integrationBrowserReadFileFn("/proc/version")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(data)), "microsoft")
}

func integrationBrowserFirstLookPath(names ...string) (string, bool) {
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		path, err := integrationBrowserLookPathFn(name)
		if err == nil && strings.TrimSpace(path) != "" {
			return path, true
		}
	}
	return "", false
}

func integrationBrowserFirstExistingFile(paths ...string) (string, bool) {
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		info, err := integrationBrowserStatFn(path)
		if err == nil && !info.IsDir() {
			return path, true
		}
	}
	return "", false
}

func integrationBrowserFirstExistingApp(paths ...string) (string, bool) {
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		info, err := integrationBrowserStatFn(path)
		if err == nil && info.IsDir() {
			return path, true
		}
	}
	return "", false
}

func integrationBrowserOpenOrActivateChromeTab(target string) (bool, error) {
	if _, ok := integrationBrowserFirstExistingApp("/Applications/Google Chrome.app"); !ok {
		return false, nil
	}
	if integrationBrowserHasRemoteDebuggingChromeProcess() {
		return false, nil
	}
	if err := integrationBrowserAppleScriptFn(integrationBrowserChromeAppleScript(target)); err != nil {
		return false, nil
	}
	return true, nil
}

func integrationBrowserHasRemoteDebuggingChromeProcess() bool {
	processes, err := integrationBrowserListProcessesFn()
	if err != nil {
		return false
	}
	for _, process := range processes {
		if integrationBrowserProcessHasRemoteDebuggingChrome(process.CommandLine) {
			return true
		}
	}
	return false
}

func integrationBrowserProcessHasRemoteDebuggingChrome(commandLine string) bool {
	commandLine = strings.ToLower(strings.TrimSpace(commandLine))
	if commandLine == "" {
		return false
	}
	if !strings.Contains(commandLine, "--remote-debugging-port=") {
		return false
	}
	return strings.Contains(commandLine, "google chrome.app/contents/macos/google chrome")
}

func integrationBrowserListProcesses() ([]integrationBrowserProcessInfo, error) {
	if runtime.GOOS == "windows" {
		return nil, nil
	}
	out, err := exec.Command("ps", "-axo", "command=").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(out), "\n")
	processes := make([]integrationBrowserProcessInfo, 0, len(lines))
	for _, line := range lines {
		commandLine := strings.TrimSpace(line)
		if commandLine == "" {
			continue
		}
		processes = append(processes, integrationBrowserProcessInfo{CommandLine: commandLine})
	}
	return processes, nil
}

func integrationBrowserChromeAppleScript(target string) string {
	return fmt.Sprintf(`
set targetUrl to %s

tell application "Google Chrome"
	if it is not running then
		activate
		open location targetUrl
		return "opened"
	end if

	repeat with w in windows
		set tabIndex to 0
		repeat with t in tabs of w
			set tabIndex to tabIndex + 1
			if (URL of t) is targetUrl then
				set active tab index of w to tabIndex
				set index of w to 1
				activate
				return "activated"
			end if
		end repeat
	end repeat

	activate
	open location targetUrl
	return "opened"
end tell
`, strconv.Quote(target))
}
