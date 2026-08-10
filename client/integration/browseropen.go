package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
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
	integrationBrowserAppleScriptFn = func(script string) (string, error) {
		cmd := exec.Command("osascript")
		cmd.Stdin = strings.NewReader(script)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			message := strings.TrimSpace(stderr.String())
			if message != "" {
				return "", fmt.Errorf("run osascript: %w: %s", err, message)
			}
			return "", fmt.Errorf("run osascript: %w", err)
		}
		return strings.TrimSpace(stdout.String()), nil
	}
)

type integrationBrowserProcessInfo struct {
	CommandLine string
}

type integrationMacBrowserSpec struct {
	AppPath                string
	AppleScriptName        string
	ProcessMatch           string
	SupportsRemoteDebugCDP bool
}

func integrationMacBrowserSpecs() []integrationMacBrowserSpec {
	return []integrationMacBrowserSpec{
		{
			AppPath:                "/Applications/Google Chrome.app",
			AppleScriptName:        "Google Chrome",
			ProcessMatch:           "google chrome.app/contents/macos/google chrome",
			SupportsRemoteDebugCDP: true,
		},
		{
			AppPath:                "/Applications/Google Chrome for Testing.app",
			AppleScriptName:        "Google Chrome for Testing",
			ProcessMatch:           "google chrome for testing.app/contents/macos/google chrome for testing",
			SupportsRemoteDebugCDP: true,
		},
		{
			AppPath:                "/Applications/Microsoft Edge.app",
			AppleScriptName:        "Microsoft Edge",
			ProcessMatch:           "microsoft edge.app/contents/macos/microsoft edge",
			SupportsRemoteDebugCDP: true,
		},
		{
			AppPath:                "/Applications/Brave Browser.app",
			AppleScriptName:        "Brave Browser",
			ProcessMatch:           "brave browser.app/contents/macos/brave browser",
			SupportsRemoteDebugCDP: true,
		},
		{
			AppPath:                "/Applications/Chromium.app",
			AppleScriptName:        "Chromium",
			ProcessMatch:           "chromium.app/contents/macos/chromium",
			SupportsRemoteDebugCDP: true,
		},
		{
			AppPath:                "/Applications/Safari.app",
			AppleScriptName:        "Safari",
			ProcessMatch:           "/applications/safari.app/contents/macos/safari",
			SupportsRemoteDebugCDP: false,
		},
	}
}

func openIntegrationBrowserMaximized(target string) error {
	return openIntegrationBrowser(target, true)
}

func openIntegrationBrowserWithoutActivation(target string) error {
	return openIntegrationBrowser(target, false)
}

func openIntegrationBrowser(target string, allowActivation bool) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return errors.New("browser target is empty")
	}
	if allowActivation && runtime.GOOS == "darwin" {
		handled, err := integrationBrowserOpenOrActivateExistingMacTab(target)
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
		if appPath, ok := integrationBrowserFirstExistingApp(integrationMacPreferredAppPaths()...); ok {
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
	if path, ok := integrationBrowserFirstExistingFile("/mnt/c/Windows/System32/cmd.exe"); ok {
		return path, []string{"/c", "start", "/max", target}, nil
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

func integrationMacPreferredAppPaths() []string {
	specs := integrationMacBrowserSpecs()
	paths := make([]string, 0, len(specs)-1)
	for _, spec := range specs {
		if strings.EqualFold(strings.TrimSpace(spec.AppleScriptName), "Safari") {
			continue
		}
		paths = append(paths, spec.AppPath)
	}
	return paths
}

func integrationBrowserOpenOrActivateExistingMacTab(target string) (bool, error) {
	spec, ok := integrationBrowserResolveRunningMacBrowser()
	if !ok {
		return false, nil
	}
	result, err := integrationBrowserAppleScriptFn(integrationBrowserMacAppleScript(spec, target))
	if err != nil {
		return false, nil
	}
	if strings.EqualFold(strings.TrimSpace(result), "activated") {
		return true, nil
	}
	return false, nil
}

func integrationBrowserResolveRunningMacBrowser() (integrationMacBrowserSpec, bool) {
	processes, err := integrationBrowserListProcessesFn()
	if err != nil {
		return integrationMacBrowserSpec{}, false
	}
	for _, spec := range integrationMacBrowserSpecs() {
		if _, ok := integrationBrowserFirstExistingApp(spec.AppPath); !ok {
			continue
		}
		if integrationBrowserProcessesContainInteractiveApp(processes, spec.ProcessMatch) {
			return spec, true
		}
	}
	return integrationMacBrowserSpec{}, false
}

func integrationBrowserProcessesContainRemoteDebug(processes []integrationBrowserProcessInfo, processMatch string) bool {
	processMatch = strings.ToLower(strings.TrimSpace(processMatch))
	if processMatch == "" {
		return false
	}
	for _, process := range processes {
		commandLine := strings.ToLower(strings.TrimSpace(process.CommandLine))
		if commandLine == "" {
			continue
		}
		if strings.Contains(commandLine, processMatch) && strings.Contains(commandLine, "--remote-debugging-port=") {
			return true
		}
	}
	return false
}

func integrationBrowserProcessesContainApp(processes []integrationBrowserProcessInfo, processMatch string) bool {
	processMatch = strings.ToLower(strings.TrimSpace(processMatch))
	if processMatch == "" {
		return false
	}
	for _, process := range processes {
		commandLine := strings.ToLower(strings.TrimSpace(process.CommandLine))
		if commandLine == "" {
			continue
		}
		if strings.Contains(commandLine, processMatch) {
			return true
		}
	}
	return false
}

func integrationBrowserProcessesContainInteractiveApp(processes []integrationBrowserProcessInfo, processMatch string) bool {
	processMatch = strings.ToLower(strings.TrimSpace(processMatch))
	if processMatch == "" {
		return false
	}
	for _, process := range processes {
		commandLine := strings.ToLower(strings.TrimSpace(process.CommandLine))
		if commandLine == "" {
			continue
		}
		if !strings.Contains(commandLine, processMatch) {
			continue
		}
		if strings.Contains(commandLine, "--headless") {
			continue
		}
		return true
	}
	return false
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

func integrationBrowserMacAppleScript(spec integrationMacBrowserSpec, target string) string {
	exactURLs, prefixURLs := integrationBrowserChromeTabMatchURLs(target)
	if strings.EqualFold(strings.TrimSpace(spec.AppleScriptName), "Safari") {
		return fmt.Sprintf(`
set exactUrls to %s
set prefixUrls to %s

tell application %s
	if it is not running then
		return "miss"
	end if

	repeat with w in windows
		repeat with t in tabs of w
			set currentUrl to ""
			try
				set currentUrl to (URL of t) as text
			end try
			if my deeprightUrlMatches(currentUrl, exactUrls, prefixUrls) then
				set current tab of w to t
				set index of w to 1
				activate
				return "activated"
			end if
		end repeat
	end repeat

	return "miss"
end tell

on deeprightUrlMatches(currentUrl, exactUrls, prefixUrls)
	if currentUrl is missing value then
		return false
	end if
	set normalizedUrl to currentUrl as text
	repeat with candidate in exactUrls
		if normalizedUrl is (candidate as text) then
			return true
		end if
	end repeat
	repeat with prefixValue in prefixUrls
		if normalizedUrl starts with (prefixValue as text) then
			return true
		end if
	end repeat
	return false
end deeprightUrlMatches
`, integrationBrowserAppleScriptStringList(exactURLs), integrationBrowserAppleScriptStringList(prefixURLs), strconv.Quote(spec.AppleScriptName))
	}
	return fmt.Sprintf(`
set exactUrls to %s
set prefixUrls to %s

tell application %s
	if it is not running then
		return "miss"
	end if

	repeat with w in windows
		set tabIndex to 0
		repeat with t in tabs of w
			set tabIndex to tabIndex + 1
			set currentUrl to ""
			try
				set currentUrl to (URL of t) as text
			end try
			if my deeprightUrlMatches(currentUrl, exactUrls, prefixUrls) then
				set active tab index of w to tabIndex
				set index of w to 1
				activate
				return "activated"
			end if
		end repeat
	end repeat

	return "miss"
end tell

on deeprightUrlMatches(currentUrl, exactUrls, prefixUrls)
	if currentUrl is missing value then
		return false
	end if
	set normalizedUrl to currentUrl as text
	repeat with candidate in exactUrls
		if normalizedUrl is (candidate as text) then
			return true
		end if
	end repeat
	repeat with prefixValue in prefixUrls
		if normalizedUrl starts with (prefixValue as text) then
			return true
		end if
	end repeat
	return false
end deeprightUrlMatches
`, integrationBrowserAppleScriptStringList(exactURLs), integrationBrowserAppleScriptStringList(prefixURLs), strconv.Quote(spec.AppleScriptName))
}

func integrationBrowserChromeTabMatchURLs(target string) ([]string, []string) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, nil
	}

	exactURLs := []string{target}
	parsed, err := url.Parse(target)
	if err != nil {
		return dedupeStrings(exactURLs), nil
	}

	scheme := strings.TrimSpace(parsed.Scheme)
	host := strings.TrimSpace(parsed.Hostname())
	if scheme == "" || host == "" || !integrationBrowserIsLoopbackHost(host) {
		return dedupeStrings(exactURLs), nil
	}

	port := strings.TrimSpace(parsed.Port())
	path := strings.TrimSpace(parsed.EscapedPath())
	if path == "" {
		path = "/"
	}
	rawQuery := strings.TrimSpace(parsed.RawQuery)
	fragment := strings.TrimSpace(parsed.Fragment)

	hosts := integrationBrowserLoopbackHostVariants(host)
	for _, candidateHost := range hosts {
		exactURLs = append(exactURLs, integrationBrowserBuildURLVariant(scheme, candidateHost, port, path, rawQuery, fragment))
	}

	var prefixURLs []string
	if integrationBrowserShouldMatchSitePrefix(path) {
		for _, candidateHost := range hosts {
			prefixURLs = append(prefixURLs, integrationBrowserBuildURLVariant(scheme, candidateHost, port, "/site/", "", ""))
		}
	}

	return dedupeStrings(exactURLs), dedupeStrings(prefixURLs)
}

func integrationBrowserShouldMatchSitePrefix(path string) bool {
	path = strings.TrimSpace(path)
	switch {
	case path == "/launch":
		return true
	case path == "/site", path == "/site/":
		return true
	case strings.HasPrefix(path, "/site/"):
		return true
	default:
		return false
	}
}

func integrationBrowserIsLoopbackHost(host string) bool {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "localhost", "127.0.0.1", "::1", "[::1]":
		return true
	default:
		return false
	}
}

func integrationBrowserLoopbackHostVariants(host string) []string {
	host = strings.ToLower(strings.TrimSpace(host))
	hosts := []string{host}
	switch host {
	case "localhost":
		hosts = append(hosts, "127.0.0.1")
	case "127.0.0.1":
		hosts = append(hosts, "localhost")
	case "::1", "[::1]":
		hosts = append(hosts, "localhost", "127.0.0.1")
	}
	return dedupeStrings(hosts)
}

func integrationBrowserBuildURLVariant(scheme, host, port, path, rawQuery, fragment string) string {
	u := url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   path,
	}
	if strings.TrimSpace(port) != "" {
		u.Host = host + ":" + port
	}
	u.RawQuery = rawQuery
	u.Fragment = fragment
	return u.String()
}

func integrationBrowserAppleScriptStringList(values []string) string {
	if len(values) == 0 {
		return "{}"
	}
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, strconv.Quote(value))
	}
	return "{" + strings.Join(quoted, ", ") + "}"
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
