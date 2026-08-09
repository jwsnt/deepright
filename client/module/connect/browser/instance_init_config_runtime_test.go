package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBrowserParseInstanceInitTimeout(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		want    time.Duration
		wantErr string
	}{
		{name: "missing browser", config: `{}`, want: browserDefaultInstanceInitTimeout},
		{name: "missing field", config: `{"browser":{}}`, want: browserDefaultInstanceInitTimeout},
		{name: "configured", config: `{"browser":{"init_timeout":17}}`, want: 17 * time.Second},
		{name: "invalid json", config: `{`, wantErr: "unexpected end"},
		{name: "null browser", config: `{"browser":null}`, wantErr: "browser must be an object"},
		{name: "string value", config: `{"browser":{"init_timeout":"300"}}`, wantErr: "positive integer"},
		{name: "zero value", config: `{"browser":{"init_timeout":0}}`, wantErr: "positive integer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := browserParseInstanceInitTimeout([]byte(tt.config))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("timeout = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestBrowserResolveInstanceInitRuntimeConfigRequiresBrowserStart(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()
	browserResolveInstanceInitRuntimeConfigFn = browserResolveInstanceInitRuntimeConfig

	pluginDir := t.TempDir()
	browserExecutablePathFn = func() (string, error) {
		return writeBrowserBinaryFixture(t, pluginDir), nil
	}

	_, err := browserResolveInstanceInitRuntimeConfig()
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "browser start") {
		t.Fatalf("err = %v, want browser start requirement", err)
	}
}

func TestBrowserResolveInstanceInitRuntimeConfigFailsWhenRecordedConfigIsMissing(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()
	browserResolveInstanceInitRuntimeConfigFn = browserResolveInstanceInitRuntimeConfig

	runtimeRoot := t.TempDir()
	pluginDir := filepath.Join(runtimeRoot, "plugins")
	browserExecutablePathFn = func() (string, error) {
		return writeBrowserBinaryFixture(t, pluginDir), nil
	}
	connectBin := filepath.Join(runtimeRoot, "integration")
	if err := os.WriteFile(connectBin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := browserWriteRecordedConnectBin(connectBin); err != nil {
		t.Fatal(err)
	}

	_, err := browserResolveInstanceInitRuntimeConfig()
	if err == nil || !strings.Contains(err.Error(), "config/config.json not found") {
		t.Fatalf("err = %v, want missing config error", err)
	}
}

func TestBrowserInitConfigFailureDoesNotShutdownExistingInstance(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	browserResolveInstanceInitRuntimeConfigFn = func() (browserInstanceInitRuntimeConfig, error) {
		return browserInstanceInitRuntimeConfig{}, errors.New("read integration config failed")
	}
	instanceGetFn = func(map[string]string) (browserInstanceRecord, error) {
		t.Fatal("instance get must not run before init config validation")
		return browserInstanceRecord{}, nil
	}
	instanceDestroyFn = func(map[string]string) error {
		t.Fatal("instance shutdown must not run before init config validation")
		return nil
	}

	_, err := browserInitInstance(map[string]string{
		"state":   filepath.Join(t.TempDir(), "browser_instance.json"),
		"agentId": "agent-a",
		"chatId":  "chat-001",
	})
	if err == nil || !strings.Contains(err.Error(), "integration config") {
		t.Fatalf("err = %v, want config failure", err)
	}
}

func TestBrowserInitUsesConfiguredTotalDeadlineForNativeChrome(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	pluginDir := t.TempDir()
	chromePath := filepath.Join(pluginDir, "chrome")
	if err := os.WriteFile(chromePath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	browserResolveInstanceInitRuntimeConfigFn = func() (browserInstanceInitRuntimeConfig, error) {
		return browserInstanceInitRuntimeConfig{Timeout: 9 * time.Second, ConfigPath: "test-config"}, nil
	}
	instanceGetFn = func(map[string]string) (browserInstanceRecord, error) {
		return browserInstanceRecord{}, errors.New("instance not found: agentId=agent-a chatId=chat-001")
	}
	instanceDestroyFn = func(map[string]string) error { return nil }
	browserExecutablePathFn = func() (string, error) { return filepath.Join(pluginDir, "browser"), nil }
	browserWSLDetectFn = func() (bool, error) { return false, nil }
	browserAttachedCommandStartProcessFn = func(string, []string, string) (int, func() error, error) {
		return 9001, func() error { return nil }, nil
	}
	var capturedTimeout time.Duration
	browserWaitForPortFn = func(_ int, _ int, timeout time.Duration) error {
		capturedTimeout = timeout
		return nil
	}
	browserCDPVersionFn = func(port int) (browserCDPVersion, error) {
		return browserCDPVersion{WebSocketDebuggerURL: instanceCDPEndpoint(port)}, nil
	}

	_, err := browserInitInstance(map[string]string{
		"state":          filepath.Join(pluginDir, "browser_instance.json"),
		"chrome":         chromePath,
		"agentId":        "agent-a",
		"chatId":         "chat-001",
		"user-data-mode": browserUserDataModeDirect,
	})
	if err != nil {
		t.Fatal(err)
	}
	if capturedTimeout <= 8*time.Second || capturedTimeout > 9*time.Second {
		t.Fatalf("chrome ready timeout = %s, want remaining configured 9s deadline", capturedTimeout)
	}
}
