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
	connectBin, prefix, err := browserResolveMetaLookupCommand(flags, allowInfer)
	if err != nil {
		return browserCookieMetaResponse{}, false, err
	}
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

func browserResolveMetaLookupCommand(flags map[string]string, allowInfer bool) (string, []string, error) {
	if connectBin := strings.TrimSpace(flags["connect-bin"]); connectBin != "" {
		connectBin, err := browserEnsureExecutablePath(connectBin)
		if err != nil {
			return "", nil, err
		}
		return browserNormalizeBinaryPath(connectBin), browserConnectPrefixForBinary(connectBin), nil
	}
	if connectBin, ok, err := browserReadRecordedConnectBin(); err != nil {
		if !browserIgnoresInvalidRuntimeRecord(flags) {
			return "", nil, err
		}
	} else if ok {
		return browserNormalizeBinaryPath(connectBin), browserConnectPrefixForBinary(connectBin), nil
	}
	if !allowInfer {
		return "", nil, nil
	}
	connectBin, err := browserInferMetaLookupBinary(flags)
	if err != nil {
		return "", nil, err
	}
	if strings.TrimSpace(connectBin) == "" {
		return "", nil, nil
	}
	return browserNormalizeBinaryPath(connectBin), browserConnectPrefixForBinary(connectBin), nil
}

func browserInferMetaLookupBinary(flags map[string]string) (string, error) {
	runtimePath, ok, err := browserResolveRuntimeConfigPath(flags)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", nil
	}
	cfg, err := browserReadRuntimeConfig(runtimePath)
	if err != nil {
		return "", err
	}
	appDir, err := browserResolveIntegrationAppDir(runtimePath, cfg)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(appDir) == "" {
		return "", nil
	}
	for _, name := range []string{"integration", "integration.exe", "proxy", "proxy.exe"} {
		candidate := filepath.Join(appDir, name)
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", nil
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
