package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseBubblewrapCheckConfig(t *testing.T) {
	config, err := parseBubblewrapCheckConfig(map[string]interface{}{
		"bubblewrap": map[string]interface{}{"check": float64(24), "install": " 安装 bubblewrap "},
	})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if config.CacheFor != 24*time.Hour || config.Install != "安装 bubblewrap" {
		t.Fatalf("config = %+v, want 24h and trimmed install request", config)
	}

	for name, raw := range map[string]map[string]interface{}{
		"missing section": {},
		"zero hours":      {"bubblewrap": map[string]interface{}{"check": float64(0), "install": "install"}},
		"empty install":   {"bubblewrap": map[string]interface{}{"check": float64(1), "install": "  "}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseBubblewrapCheckConfig(raw); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
}

func TestHandleBubblewrapCheckCachesLinuxAvailabilityAndSkipsOtherPlatforms(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "deepright", "integration")
	configDir := filepath.Join(filepath.Dir(executable), "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"bubblewrap":{"check":1,"install":"安装 bubblewrap"}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	previousExecutable := integrationExecutableFn
	previousLookPath := bubblewrapCheckLookPathFn
	previousNow := bubblewrapCheckNowFn
	previousGOOS := integrationRuntimeGOOS
	bubblewrapCheckCache.Lock()
	previousAvailableAt := bubblewrapCheckCache.availableAt
	bubblewrapCheckCache.availableAt = time.Time{}
	bubblewrapCheckCache.Unlock()
	t.Cleanup(func() {
		integrationExecutableFn = previousExecutable
		bubblewrapCheckLookPathFn = previousLookPath
		bubblewrapCheckNowFn = previousNow
		integrationRuntimeGOOS = previousGOOS
		bubblewrapCheckCache.Lock()
		bubblewrapCheckCache.availableAt = previousAvailableAt
		bubblewrapCheckCache.Unlock()
	})
	integrationExecutableFn = func() (string, error) { return executable, nil }
	integrationRuntimeGOOS = "linux"

	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	bubblewrapCheckNowFn = func() time.Time { return now }
	installed := true
	lookups := 0
	bubblewrapCheckLookPathFn = func(name string) (string, error) {
		if name != "bwrap" {
			t.Fatalf("lookup = %q, want bwrap", name)
		}
		lookups++
		if !installed {
			return "", errors.New("not installed")
		}
		return "/usr/bin/bwrap", nil
	}

	request := func() BubblewrapCheckResponse {
		recorder := httptest.NewRecorder()
		handleBubblewrapCheck().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/bubblewrap/check", nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}
		var response BubblewrapCheckResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		return response
	}

	if response := request(); response.Status != 0 || !response.Required || !response.Available || lookups != 1 {
		t.Fatalf("first response = %+v, lookups = %d", response, lookups)
	}
	now = now.Add(30 * time.Minute)
	if response := request(); !response.Available || lookups != 1 {
		t.Fatalf("cached response = %+v, lookups = %d", response, lookups)
	}
	now = now.Add(time.Hour)
	installed = false
	if response := request(); response.Available || response.Install != "安装 bubblewrap" || lookups != 2 {
		t.Fatalf("missing response = %+v, lookups = %d", response, lookups)
	}

	integrationRuntimeGOOS = "darwin"
	if response := request(); response.Required || !response.Available || lookups != 2 {
		t.Fatalf("non-Linux response = %+v, lookups = %d", response, lookups)
	}
}

func TestHandleBubblewrapCheckReportsInvalidLinuxConfiguration(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "deepright", "integration")
	configDir := filepath.Join(filepath.Dir(executable), "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"bubblewrap":{"check":0,"install":"install"}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	previousExecutable := integrationExecutableFn
	previousGOOS := integrationRuntimeGOOS
	t.Cleanup(func() {
		integrationExecutableFn = previousExecutable
		integrationRuntimeGOOS = previousGOOS
	})
	integrationExecutableFn = func() (string, error) { return executable, nil }
	integrationRuntimeGOOS = "linux"

	recorder := httptest.NewRecorder()
	handleBubblewrapCheck().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/bubblewrap/check", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	var response BubblewrapCheckResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != 1 || !response.Required || response.Content == "" {
		t.Fatalf("response = %+v, want visible Linux configuration error", response)
	}
}
