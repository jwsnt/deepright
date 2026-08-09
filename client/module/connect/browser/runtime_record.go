package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const browserRuntimeFileName = "browser_runtime.json"

type browserRuntimeRecord struct {
	ConnectBin string `json:"connectBin"`
}

func browserRuntimeFilePath() (string, error) {
	browserPath, err := browserExecutablePathFn()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(browserPath); err == nil {
		browserPath = resolved
	}
	if abs, err := filepath.Abs(browserPath); err == nil {
		browserPath = abs
	}
	root := strings.TrimSpace(filepath.Dir(browserPath))
	if root == "" {
		return "", fmt.Errorf("resolve browser executable directory")
	}
	return filepath.Join(root, browserRuntimeFileName), nil
}

func browserReadRecordedConnectBin() (string, bool, error) {
	runtimeFilePath, err := browserRuntimeFilePath()
	if err != nil {
		return "", false, err
	}
	data, err := browserReadFileFn(runtimeFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	var record browserRuntimeRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return "", true, fmt.Errorf("decode %s: %w", runtimeFilePath, err)
	}
	connectBin, err := browserEnsureExecutablePath(record.ConnectBin)
	if err != nil {
		return "", true, fmt.Errorf("resolve %s connectBin: %w", runtimeFilePath, err)
	}
	return connectBin, true, nil
}

func browserResolveRecordedRuntimeConfigPath() (string, bool, error) {
	connectBin, ok, err := browserReadRecordedConnectBin()
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}
	runtimePath, ok := browserResolveIntegrationRuntimePath(connectBin)
	if !ok {
		return "", false, fmt.Errorf("resolve runtime config from %s", browserRuntimeFileName)
	}
	return runtimePath, true, nil
}

func browserWriteRecordedConnectBin(connectBin string) error {
	connectBin, err := browserEnsureExecutablePath(connectBin)
	if err != nil {
		return err
	}
	runtimeFilePath, err := browserRuntimeFilePath()
	if err != nil {
		return err
	}
	if err := browserMkdirAllFn(filepath.Dir(runtimeFilePath), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(browserRuntimeRecord{ConnectBin: connectBin}, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return browserWriteFileFn(runtimeFilePath, payload, 0o644)
}

func browserRemoveRecordedConnectBin() error {
	runtimeFilePath, err := browserRuntimeFilePath()
	if err != nil {
		return err
	}
	if err := browserRemoveAllFn(runtimeFilePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func browserMaybeRemoveRecordedConnectBin() {
	if err := browserRemoveRecordedConnectBin(); err != nil {
		browserLogAsyncLifecycleEvent("browser_runtime_record", "remove_error", []string{browserRuntimeFileName}, 0, err)
		return
	}
	browserLogAsyncLifecycleEvent("browser_runtime_record", "remove_ok", []string{browserRuntimeFileName}, 0, nil)
}
