package service

import (
	"fmt"
	"os"
	"path/filepath"
	"runtimepaths"
	"strings"
)

const (
	SandboxModeFilePick    = "filepick"
	SandboxModeNet         = "net"
	SandboxModeFilePickNet = "filepick_net"
)

const sandboxAllowedDirEnv = "CLI_SANDBOX_ALLOWED_DIR"
const SandboxAllowedDirEnv = sandboxAllowedDirEnv
const sandboxStateDirName = "CLI_SANDBOX"

var Debugf func(format string, args ...any)

func NormalizeSandboxMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case SandboxModeFilePick:
		return SandboxModeFilePick
	case SandboxModeNet:
		return SandboxModeNet
	case SandboxModeFilePickNet:
		return SandboxModeFilePickNet
	default:
		return ""
	}
}

func requiresPickedDirectory(mode string) bool {
	switch NormalizeSandboxMode(mode) {
	case SandboxModeFilePick, SandboxModeFilePickNet:
		return true
	default:
		return false
	}
}

func resolvePickedDirectory() (string, error) {
	if override := strings.TrimSpace(os.Getenv(sandboxAllowedDirEnv)); override != "" {
		debugf("picked directory using override path=%q", override)
		return normalizeDirectoryPath(override)
	}
	return "", fmt.Errorf("未找到当前命令的已授权目录，请显式传入 --allowed-dir")
}

func SetPickedDirectory(path string) (string, error) {
	normalized, err := normalizeDirectoryPath(path)
	if err != nil {
		return "", err
	}
	debugf("picked directory accepted path=%q", normalized)
	return normalized, nil
}

func debugf(format string, args ...any) {
	if Debugf != nil {
		Debugf(format, args...)
	}
}

func normalizeDirectoryPath(raw string) (string, error) {
	path := stripWrappedQuotes(strings.TrimSpace(raw))
	if path == "" {
		return "", fmt.Errorf("empty directory path")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", absPath)
	}
	return filepath.Clean(absPath), nil
}

func stripWrappedQuotes(path string) string {
	path = strings.TrimSpace(path)
	if len(path) >= 2 {
		if (path[0] == '"' && path[len(path)-1] == '"') || (path[0] == '\'' && path[len(path)-1] == '\'') {
			return strings.TrimSpace(path[1 : len(path)-1])
		}
	}
	return path
}

func pickedDirectoryVariants(path string) []string {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	path = filepath.Clean(path)
	seen := map[string]struct{}{path: {}}
	paths := []string{path}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		resolved = filepath.Clean(resolved)
		if _, ok := seen[resolved]; !ok {
			paths = append(paths, resolved)
		}
	}
	return paths
}

func buildSandboxProfile(mode, pickedDir string) string {
	mode = NormalizeSandboxMode(mode)
	var builder strings.Builder
	builder.WriteString("(version 1)\n")
	builder.WriteString("(allow default)\n")
	if mode == SandboxModeNet || mode == SandboxModeFilePickNet {
		builder.WriteString("(deny network*)\n")
	}
	if requiresPickedDirectory(mode) {
		builder.WriteString("(deny file-read* file-write* (subpath \"/Users\") (subpath \"/Volumes\") (subpath \"/private\"))\n")
		builder.WriteString("(allow file-read* file-write*\n")
		for _, variant := range pickedDirectoryVariants(pickedDir) {
			builder.WriteString("  (subpath ")
			builder.WriteString(quoteSandboxString(variant))
			builder.WriteString(")\n")
		}
		for _, path := range sandboxAlwaysAllowedUserPaths() {
			builder.WriteString("  (subpath ")
			builder.WriteString(quoteSandboxString(path))
			builder.WriteString(")\n")
		}
		for _, path := range []string{
			"/private/etc",
			"/private/dev",
			"/private/var/select",
			"/private/var/run",
			"/private/var/db",
			"/private/tmp",
			"/tmp",
		} {
			builder.WriteString("  (subpath ")
			builder.WriteString(quoteSandboxString(path))
			builder.WriteString(")\n")
		}
		builder.WriteString(")\n")
	}
	return builder.String()
}

func quoteSandboxString(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return `"` + replacer.Replace(value) + `"`
}

func sandboxStateDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	stateDir := filepath.Join(configDir, sandboxStateDirName)
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return "", err
	}
	return stateDir, nil
}

func sandboxAlwaysAllowedUserPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	home = filepath.Clean(strings.TrimSpace(home))
	if home == "" {
		return nil
	}

	paths := []string{
		runtimepaths.MacAppRuntimeBaseDir(home, runtimepaths.DeepRightMacBundleIdentifier, runtimepaths.DeepRightAppName),
	}
	if stateDir, err := sandboxStateDir(); err == nil {
		paths = append(paths, stateDir)
	}

	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		path = filepath.Clean(strings.TrimSpace(path))
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		result = append(result, path)
	}
	return result
}
