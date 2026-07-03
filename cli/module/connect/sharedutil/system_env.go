package sharedutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var (
	systemPathApplyOnce sync.Once

	systemPathRuntimeGOOS = runtime.GOOS
	systemPathLookupEnvFn = os.LookupEnv
	systemPathSetenvFn    = os.Setenv
	systemPathUserHomeFn  = os.UserHomeDir
	systemPathCommandFn   = func(name string, args ...string) ([]byte, error) {
		return exec.Command(name, args...).Output()
	}
)

// ApplySystemPath normalizes PATH/Path once per process so command execution
// and runtime probing resolve binaries from the same environment.
func ApplySystemPath() {
	systemPathApplyOnce.Do(func() {
		values := append([]string{currentProcessPath()}, discoveredSystemPathValues()...)
		merged := mergeSystemPathValues(values...)
		if strings.TrimSpace(merged) == "" {
			return
		}
		setProcessPath(merged)
	})
}

// CurrentEnvironmentWithSystemPath returns the current environment after the
// process PATH has been normalized once.
func CurrentEnvironmentWithSystemPath() []string {
	ApplySystemPath()
	return os.Environ()
}

func discoveredSystemPathValues() []string {
	values := make([]string, 0, 8)
	values = append(values, platformSystemPathValues(systemPathRuntimeGOOS)...)
	values = append(values, platformCommonPathValue(systemPathRuntimeGOOS))
	return values
}

func platformSystemPathValues(goos string) []string {
	switch strings.TrimSpace(goos) {
	case "darwin":
		return []string{
			runFirstPathProbe(unixShellPathCandidates(goos), "-lc", `printf '%s' "$PATH"`),
			parsePathHelperOutput(runPathProbe("/usr/libexec/path_helper", "-s")),
			runPathProbe("/bin/launchctl", "getenv", "PATH"),
		}
	case "linux":
		return []string{
			runFirstPathProbe(unixShellPathCandidates(goos), "-lc", `printf '%s' "$PATH"`),
		}
	case "windows":
		return []string{
			normalizeWindowsPathProbe(runFirstPathProbe(windowsPowerShellCandidates(), "-NoLogo", "-NoProfile", "-NonInteractive", "-Command",
				`[Environment]::GetEnvironmentVariable('Path','Machine'); [Environment]::GetEnvironmentVariable('Path','User')`)),
		}
	default:
		return nil
	}
}

func unixShellPathCandidates(goos string) []string {
	candidates := make([]string, 0, 4)
	if shell, ok := systemPathLookupEnvFn("SHELL"); ok && strings.TrimSpace(shell) != "" {
		candidates = append(candidates, strings.TrimSpace(shell))
	}
	switch goos {
	case "darwin":
		candidates = append(candidates, "/bin/zsh", "/bin/bash", "/bin/sh")
	default:
		candidates = append(candidates, "/bin/bash", "/bin/sh")
	}
	return uniquePathEntries(candidates, false)
}

func windowsPowerShellCandidates() []string {
	candidates := make([]string, 0, 4)
	systemRoot := firstNonEmptyLookup("SystemRoot", "SYSTEMROOT")
	if strings.TrimSpace(systemRoot) != "" {
		candidates = append(candidates,
			filepath.Join(systemRoot, "System32", "WindowsPowerShell", "v1.0", "powershell.exe"),
			filepath.Join(systemRoot, "System32", "WindowsPowerShell", "v1.0", "pwsh.exe"),
		)
	}
	candidates = append(candidates, "powershell.exe", "pwsh.exe")
	return uniquePathEntries(candidates, true)
}

func firstNonEmptyLookup(keys ...string) string {
	for _, key := range keys {
		if value, ok := systemPathLookupEnvFn(key); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func runFirstPathProbe(commands []string, args ...string) string {
	for _, command := range commands {
		if output := runPathProbe(command, args...); strings.TrimSpace(output) != "" {
			return output
		}
	}
	return ""
}

func runPathProbe(command string, args ...string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}
	out, err := systemPathCommandFn(command, args...)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func parsePathHelperOutput(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(part, "PATH=") {
			continue
		}
		return strings.Trim(strings.TrimPrefix(part, "PATH="), `"'`)
	}
	return ""
}

func normalizeWindowsPathProbe(raw string) string {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "\r", ""))
	if raw == "" {
		return ""
	}
	lines := strings.Split(raw, "\n")
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			parts = append(parts, line)
		}
	}
	return strings.Join(parts, pathListSeparator("windows"))
}

func platformCommonPathValue(goos string) string {
	paths := make([]string, 0, 16)
	homeDir, _ := systemPathUserHomeFn()
	homeDir = strings.TrimSpace(homeDir)

	switch strings.TrimSpace(goos) {
	case "darwin":
		paths = append(paths,
			"/opt/homebrew/bin",
			"/usr/local/bin",
			"/usr/bin",
			"/bin",
			"/usr/sbin",
			"/sbin",
		)
		if homeDir != "" {
			paths = append(paths,
				filepath.Join(homeDir, ".local", "bin"),
				filepath.Join(homeDir, ".local", "node", "bin"),
				filepath.Join(homeDir, ".npm-global", "bin"),
				filepath.Join(homeDir, ".volta", "bin"),
				filepath.Join(homeDir, ".fnm", "current", "bin"),
				filepath.Join(homeDir, ".n", "bin"),
				filepath.Join(homeDir, "bin"),
			)
		}
	case "linux":
		paths = append(paths,
			"/usr/local/sbin",
			"/usr/local/bin",
			"/usr/sbin",
			"/usr/bin",
			"/sbin",
			"/bin",
		)
		if homeDir != "" {
			paths = append(paths,
				filepath.Join(homeDir, ".local", "bin"),
				filepath.Join(homeDir, ".local", "node", "bin"),
				filepath.Join(homeDir, ".npm-global", "bin"),
				filepath.Join(homeDir, ".volta", "bin"),
				filepath.Join(homeDir, ".fnm", "current", "bin"),
				filepath.Join(homeDir, ".n", "bin"),
				filepath.Join(homeDir, "bin"),
			)
		}
	case "windows":
		systemRoot := firstNonEmptyLookup("SystemRoot", "SYSTEMROOT")
		localAppData := firstNonEmptyLookup("LocalAppData", "LOCALAPPDATA")
		appData := firstNonEmptyLookup("AppData", "APPDATA")
		programFiles := firstNonEmptyLookup("ProgramFiles")
		userProfile := firstNonEmptyLookup("USERPROFILE")
		if systemRoot != "" {
			paths = append(paths,
				filepath.Join(systemRoot, "System32"),
				filepath.Join(systemRoot, "System32", "Wbem"),
				filepath.Join(systemRoot, "System32", "WindowsPowerShell", "v1.0"),
				systemRoot,
			)
		}
		if programFiles != "" {
			paths = append(paths, filepath.Join(programFiles, "nodejs"))
		}
		if localAppData != "" {
			paths = append(paths,
				filepath.Join(localAppData, "Programs", "nodejs"),
				filepath.Join(localAppData, "Microsoft", "WindowsApps"),
			)
		}
		if appData != "" {
			paths = append(paths, filepath.Join(appData, "npm"))
		}
		if userProfile != "" {
			paths = append(paths,
				filepath.Join(userProfile, "scoop", "shims"),
				filepath.Join(userProfile, ".volta", "bin"),
			)
		}
	}

	return strings.Join(uniquePathEntries(paths, strings.TrimSpace(goos) == "windows"), pathListSeparator(goos))
}

func currentProcessPath() string {
	_, value := pathEnvKeyAndValue()
	return value
}

func pathEnvKeyAndValue() (string, string) {
	if strings.TrimSpace(systemPathRuntimeGOOS) == "windows" {
		if value, ok := systemPathLookupEnvFn("Path"); ok {
			return "Path", value
		}
		if value, ok := systemPathLookupEnvFn("PATH"); ok {
			return "PATH", value
		}
		return "Path", ""
	}
	if value, ok := systemPathLookupEnvFn("PATH"); ok {
		return "PATH", value
	}
	return "PATH", ""
}

func setProcessPath(value string) {
	key, _ := pathEnvKeyAndValue()
	_ = systemPathSetenvFn(key, value)
	if strings.TrimSpace(systemPathRuntimeGOOS) == "windows" {
		other := "PATH"
		if key == "PATH" {
			other = "Path"
		}
		_ = systemPathSetenvFn(other, value)
	}
}

func mergeSystemPathValues(values ...string) string {
	entries := make([]string, 0, 32)
	caseInsensitive := strings.TrimSpace(systemPathRuntimeGOOS) == "windows"
	for _, value := range values {
		entries = append(entries, splitPathEntries(value)...)
	}
	return strings.Join(uniquePathEntries(entries, caseInsensitive), pathListSeparator(systemPathRuntimeGOOS))
}

func splitPathEntries(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, pathListSeparator(systemPathRuntimeGOOS))
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func uniquePathEntries(entries []string, caseInsensitive bool) []string {
	out := make([]string, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		key := entry
		if caseInsensitive {
			key = strings.ToLower(key)
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, entry)
	}
	return out
}

func pathListSeparator(goos string) string {
	if strings.TrimSpace(goos) == "windows" {
		return ";"
	}
	return ":"
}

func debugSystemPath() string {
	key, value := pathEnvKeyAndValue()
	return fmt.Sprintf("%s=%s", key, value)
}
