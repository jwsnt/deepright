package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const browserMetaLookupTimeout = 5 * time.Second

var browserCookieCommandContextFn = exec.CommandContext

type browserCookieMetaResponse struct {
	Key  string         `json:"key"`
	Meta map[string]any `json:"meta"`
}

func browserLookupPluginMeta(flags map[string]string, key string, allowInfer bool) (browserCookieMetaResponse, bool, error) {
	connectBin, prefix := browserResolveMetaLookupCommand(flags, allowInfer)
	if strings.TrimSpace(connectBin) == "" {
		return browserCookieMetaResponse{}, false, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), browserMetaLookupTimeout)
	defer cancel()

	args := append([]string{}, prefix...)
	args = append(args, "meta-get", "--key", strings.TrimSpace(key))
	cmd := browserCookieCommandContextFn(ctx, connectBin, args...)
	output, err := cmd.Output()
	if err != nil {
		return browserCookieMetaResponse{}, false, fmt.Errorf("run %s %s failed: %w", connectBin, strings.Join(args, " "), err)
	}

	var response browserCookieMetaResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return browserCookieMetaResponse{}, false, fmt.Errorf("decode %s meta-get output: %w", key, err)
	}
	return response, true, nil
}

func browserResolveMetaLookupCommand(flags map[string]string, allowInfer bool) (string, []string) {
	explicit := strings.TrimSpace(flags["connect-bin"])
	if explicit == "" && allowInfer {
		explicit = browserInferMetaLookupBinary(flags)
	}
	if explicit == "" {
		return "", nil
	}
	return browserNormalizeBinaryPath(explicit), browserConnectPrefixForBinary(explicit)
}

func browserInferMetaLookupBinary(flags map[string]string) string {
	runtimePath, ok := browserResolveRuntimeConfigPath(flags)
	if !ok {
		return ""
	}
	cfg, err := browserReadRuntimeConfig(runtimePath)
	if err != nil {
		return ""
	}
	if appPath := browserResolveRuntimePathValue(runtimePath, cfg["app"]); appPath != "" {
		return appPath
	}
	baseDir := filepath.Dir(runtimePath)
	for _, name := range []string{"integration", "integration.exe", "proxy", "proxy.exe"} {
		candidate := filepath.Join(baseDir, name)
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func browserNormalizeBinaryPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if filepath.IsAbs(value) {
		return value
	}
	if strings.HasPrefix(value, ".") || strings.Contains(value, "/") || strings.Contains(value, string(filepath.Separator)) {
		if abs, err := filepath.Abs(value); err == nil {
			return abs
		}
	}
	return value
}

func browserConnectPrefixForBinary(binary string) []string {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(binary)))
	if base == "integration" || base == "proxy" || strings.HasPrefix(base, "integration.") || strings.HasPrefix(base, "proxy.") {
		return []string{"connect"}
	}
	return nil
}
