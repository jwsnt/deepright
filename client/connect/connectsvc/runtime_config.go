package connectsvc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtimepaths"
	"strings"
)

func ResolveBundledRuntimeConfigPath(binaryPath string) string {
	binaryPath = strings.TrimSpace(binaryPath)
	if binaryPath == "" {
		return ""
	}
	if abs, err := filepath.Abs(binaryPath); err == nil {
		binaryPath = abs
	}
	macOSDir := filepath.Dir(binaryPath)
	if filepath.Base(macOSDir) != "MacOS" {
		return ""
	}
	contentsDir := filepath.Dir(macOSDir)
	if filepath.Base(contentsDir) != "Contents" {
		return ""
	}
	bundleRoot := filepath.Dir(contentsDir)
	if !strings.HasSuffix(strings.ToLower(bundleRoot), ".app") {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ""
	}
	paths := runtimepaths.MacRuntimeConfigCandidates(home, runtimepaths.DeepRightMacBundleIdentifier, runtimepaths.DeepRightAppName)
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func ResolveRuntimeConfigPathFromConnectBin(connectBin string) (string, bool) {
	connectBin = strings.TrimSpace(connectBin)
	if connectBin == "" {
		return "", false
	}
	if abs, err := filepath.Abs(connectBin); err == nil {
		connectBin = abs
	}
	if bundledPath := ResolveBundledRuntimeConfigPath(connectBin); bundledPath != "" {
		configPaths := []string{bundledPath}
		if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
			configPaths = runtimepaths.MacRuntimeConfigCandidates(home, runtimepaths.DeepRightMacBundleIdentifier, runtimepaths.DeepRightAppName)
		}
		for _, configPath := range configPaths {
			if info, err := os.Stat(configPath); err == nil && !info.IsDir() {
				return configPath, true
			}
		}
	}
	return resolveRuntimeConfigPathCandidates(filepath.Dir(connectBin), []string{
		filepath.Join("config", "config.json"),
		filepath.Join("..", "config", "config.json"),
		filepath.Join("Resources", "config", "config.json"),
		filepath.Join("..", "Resources", "config", "config.json"),
	})
}

func ResolveRuntimeConfigPathNearBinary(binaryPath string) (string, bool) {
	binaryPath = strings.TrimSpace(binaryPath)
	if binaryPath == "" {
		return "", false
	}
	if abs, err := filepath.Abs(binaryPath); err == nil {
		binaryPath = abs
	}
	if bundledPath := ResolveBundledRuntimeConfigPath(binaryPath); bundledPath != "" {
		configPaths := []string{bundledPath}
		if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
			configPaths = runtimepaths.MacRuntimeConfigCandidates(home, runtimepaths.DeepRightMacBundleIdentifier, runtimepaths.DeepRightAppName)
		}
		for _, configPath := range configPaths {
			if info, err := os.Stat(configPath); err == nil && !info.IsDir() {
				return configPath, true
			}
		}
	}
	return resolveRuntimeConfigPathCandidates(filepath.Dir(binaryPath), []string{
		filepath.Join("config", "config.json"),
		filepath.Join("..", "config", "config.json"),
		filepath.Join("..", "..", "config", "config.json"),
	})
}

func ReadRuntimeConfig(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(raw))
	for key, value := range raw {
		switch v := value.(type) {
		case string:
			out[key] = v
		}
	}
	return out, nil
}

func ResolveRuntimePathValue(configPath, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if filepath.IsAbs(raw) {
		return raw
	}
	return filepath.Clean(filepath.Join(filepath.Dir(configPath), raw))
}

func resolveRuntimeConfigPathCandidates(baseDir string, candidates []string) (string, bool) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		return "", false
	}
	for _, candidate := range candidates {
		path := filepath.Join(baseDir, candidate)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, true
		}
	}
	return "", false
}
