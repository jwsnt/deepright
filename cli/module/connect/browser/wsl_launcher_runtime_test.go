package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBrowserResolveLauncherScriptPathRecreatesMissingLauncher(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	pluginDir := t.TempDir()
	browserExecutablePathFn = func() (string, error) {
		return filepath.Join(pluginDir, "browser"), nil
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "browser"), []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}

	scriptPath, err := browserResolveLauncherScriptPath(nil)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, "DEEPRIGHT_BROWSER_BIN") {
		t.Fatalf("launcher script missing browser env fallback: %s", text)
	}
	if !strings.Contains(text, `exec "$BROWSER_BIN" __wsl-instance acquire "$@"`) {
		t.Fatalf("launcher script missing acquire command: %s", text)
	}
}

func TestBrowserResolveInstanceInitRuntimeConfigUsesRecordedIntegrationConfig(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()
	browserResolveInstanceInitRuntimeConfigFn = browserResolveInstanceInitRuntimeConfig

	runtimeRoot := t.TempDir()
	pluginDir := filepath.Join(runtimeRoot, "plugins")
	browserPath := writeBrowserBinaryFixture(t, pluginDir)
	browserExecutablePathFn = func() (string, error) {
		return browserPath, nil
	}
	runtimePath := filepath.Join(runtimeRoot, "config", "config.json")
	if err := os.MkdirAll(filepath.Dir(runtimePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runtimePath, []byte(`{"browser":{"init_timeout":300}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	connectBin := filepath.Join(runtimeRoot, "integration")
	if err := os.WriteFile(connectBin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := browserWriteRecordedConnectBin(connectBin); err != nil {
		t.Fatal(err)
	}

	config, err := browserResolveInstanceInitRuntimeConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Timeout != 5*time.Minute {
		t.Fatalf("timeout = %s, want %s", config.Timeout, 5*time.Minute)
	}
}

func TestBrowserResolveInstanceInitRuntimeConfigDefaultsOnlyWhenFieldIsMissing(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()
	browserResolveInstanceInitRuntimeConfigFn = browserResolveInstanceInitRuntimeConfig

	runtimeRoot := t.TempDir()
	pluginDir := filepath.Join(runtimeRoot, "plugins")
	browserPath := writeBrowserBinaryFixture(t, pluginDir)
	browserExecutablePathFn = func() (string, error) {
		return browserPath, nil
	}
	runtimePath := filepath.Join(runtimeRoot, "config", "config.json")
	if err := os.MkdirAll(filepath.Dir(runtimePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runtimePath, []byte(`{"browser":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	connectBin := filepath.Join(runtimeRoot, "integration")
	if err := os.WriteFile(connectBin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := browserWriteRecordedConnectBin(connectBin); err != nil {
		t.Fatal(err)
	}

	config, err := browserResolveInstanceInitRuntimeConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Timeout != browserDefaultInstanceInitTimeout {
		t.Fatalf("timeout = %s, want %s", config.Timeout, browserDefaultInstanceInitTimeout)
	}
}
