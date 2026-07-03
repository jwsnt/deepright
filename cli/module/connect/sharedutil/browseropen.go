package sharedutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

var (
	browserOpenLookPathFn      = exec.LookPath
	browserOpenStatFn          = os.Stat
	browserOpenListProcessesFn = browserOpenListProcesses
	browserOpenStartFn         = func(cmdPath string, args ...string) error {
		return exec.Command(cmdPath, args...).Start()
	}
	browserOpenAppleScriptFn = func(script string) error {
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

type browserOpenProcessInfo struct {
	CommandLine string
}

func OpenBrowserMaximized(target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("browser target is empty")
	}
	if runtime.GOOS == "darwin" {
		handled, err := browserOpenOrActivateChromeTab(target)
		if handled {
			return err
		}
	}
	cmdPath, args, err := browserOpenCommand(runtime.GOOS, target)
	if err != nil {
		return err
	}
	return browserOpenStartFn(cmdPath, args...)
}

func browserOpenCommand(goos, target string) (string, []string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", nil, fmt.Errorf("browser target is empty")
	}

	switch strings.TrimSpace(goos) {
	case "darwin":
		if path, ok := browserOpenFirstExistingFile(
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		); ok {
			return path, []string{"--start-maximized", target}, nil
		}
		return "open", []string{target}, nil
	case "linux":
		if path, ok := browserOpenFirstLookPath(
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
		if path, ok := browserOpenFirstExistingFile(
			filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("LocalAppData"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(os.Getenv("LocalAppData"), "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "Chromium", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Chromium", "Application", "chrome.exe"),
		); ok {
			return path, []string{"--start-maximized", target}, nil
		}
		if path, ok := browserOpenFirstLookPath("chrome", "msedge", "chromium", "brave"); ok {
			return path, []string{"--start-maximized", target}, nil
		}
		return "cmd", []string{"/c", "start", "", "/max", target}, nil
	default:
		return "", nil, fmt.Errorf("unsupported OS: %s", goos)
	}
}

func browserOpenFirstLookPath(names ...string) (string, bool) {
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		path, err := browserOpenLookPathFn(name)
		if err == nil && strings.TrimSpace(path) != "" {
			return path, true
		}
	}
	return "", false
}

func browserOpenFirstExistingFile(paths ...string) (string, bool) {
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		info, err := browserOpenStatFn(path)
		if err == nil && !info.IsDir() {
			return path, true
		}
	}
	return "", false
}

func browserOpenFirstExistingApp(paths ...string) (string, bool) {
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		info, err := browserOpenStatFn(path)
		if err == nil && info.IsDir() {
			return path, true
		}
	}
	return "", false
}

func browserOpenOrActivateChromeTab(target string) (bool, error) {
	if _, ok := browserOpenFirstExistingApp("/Applications/Google Chrome.app"); !ok {
		return false, nil
	}
	if browserOpenHasRemoteDebuggingChromeProcess() {
		return false, nil
	}
	if err := browserOpenAppleScriptFn(browserOpenChromeAppleScript(target)); err != nil {
		return false, nil
	}
	return true, nil
}

func browserOpenHasRemoteDebuggingChromeProcess() bool {
	processes, err := browserOpenListProcessesFn()
	if err != nil {
		return false
	}
	for _, process := range processes {
		if browserOpenProcessHasRemoteDebuggingChrome(process.CommandLine) {
			return true
		}
	}
	return false
}

func browserOpenProcessHasRemoteDebuggingChrome(commandLine string) bool {
	commandLine = strings.ToLower(strings.TrimSpace(commandLine))
	if commandLine == "" {
		return false
	}
	if !strings.Contains(commandLine, "--remote-debugging-port=") {
		return false
	}
	return strings.Contains(commandLine, "google chrome.app/contents/macos/google chrome")
}

func browserOpenListProcesses() ([]browserOpenProcessInfo, error) {
	if runtime.GOOS == "windows" {
		return nil, nil
	}
	out, err := exec.Command("ps", "-axo", "command=").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(out), "\n")
	processes := make([]browserOpenProcessInfo, 0, len(lines))
	for _, line := range lines {
		commandLine := strings.TrimSpace(line)
		if commandLine == "" {
			continue
		}
		processes = append(processes, browserOpenProcessInfo{CommandLine: commandLine})
	}
	return processes, nil
}

func browserOpenChromeAppleScript(target string) string {
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
