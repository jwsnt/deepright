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

func TestParseFFmpegCheckConfig(t *testing.T) {
	config, err := parseFFmpegCheckConfig(map[string]interface{}{
		"ffmpeg": map[string]interface{}{"check": float64(120), "install": " 请安装 ffmpeg 和 ffprobe "},
	})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if config.CacheFor != 120*time.Hour {
		t.Fatalf("cache duration = %s, want 120h", config.CacheFor)
	}
	if config.Install != "请安装 ffmpeg 和 ffprobe" {
		t.Fatalf("install = %q", config.Install)
	}

	for name, raw := range map[string]map[string]interface{}{
		"missing ffmpeg": {},
		"zero hours":     {"ffmpeg": map[string]interface{}{"check": float64(0), "install": "install"}},
		"fraction hours": {"ffmpeg": map[string]interface{}{"check": float64(1.5), "install": "install"}},
		"empty install":  {"ffmpeg": map[string]interface{}{"check": float64(1), "install": "  "}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseFFmpegCheckConfig(raw); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
}

func TestHandleFFmpegCheckCachesOnlySuccessfulDependencies(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "deepright", "integration")
	configDir := filepath.Join(filepath.Dir(executable), "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"ffmpeg":{"check":1,"install":"请安装 ffmpeg 和 ffprobe"}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	previousExecutable := integrationExecutableFn
	previousLookPath := ffmpegCheckLookPathFn
	previousNow := ffmpegCheckNowFn
	ffmpegCheckCache.Lock()
	previousAvailableAt := ffmpegCheckCache.availableAt
	ffmpegCheckCache.availableAt = time.Time{}
	ffmpegCheckCache.Unlock()
	t.Cleanup(func() {
		integrationExecutableFn = previousExecutable
		ffmpegCheckLookPathFn = previousLookPath
		ffmpegCheckNowFn = previousNow
		ffmpegCheckCache.Lock()
		ffmpegCheckCache.availableAt = previousAvailableAt
		ffmpegCheckCache.Unlock()
	})
	integrationExecutableFn = func() (string, error) { return executable, nil }

	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	ffmpegCheckNowFn = func() time.Time { return now }
	installed := true
	lookups := map[string]int{}
	ffmpegCheckLookPathFn = func(name string) (string, error) {
		lookups[name]++
		if !installed {
			return "", errors.New("not installed")
		}
		return "/usr/local/bin/" + name, nil
	}

	request := func() FFmpegCheckResponse {
		recorder := httptest.NewRecorder()
		handleFFmpegCheck().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/ffmpeg/check", nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}
		var response FFmpegCheckResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		return response
	}

	if response := request(); response.Status != 0 || !response.Available {
		t.Fatalf("first response = %+v, want available", response)
	}
	if lookups["ffmpeg"] != 1 || lookups["ffprobe"] != 1 {
		t.Fatalf("lookups after first check = %#v", lookups)
	}

	now = now.Add(30 * time.Minute)
	if response := request(); response.Status != 0 || !response.Available {
		t.Fatalf("cached response = %+v, want available", response)
	}
	if lookups["ffmpeg"] != 1 || lookups["ffprobe"] != 1 {
		t.Fatalf("cached check unexpectedly looked up dependencies: %#v", lookups)
	}

	now = now.Add(time.Hour)
	installed = false
	if response := request(); response.Status != 0 || response.Available || response.Install != "请安装 ffmpeg 和 ffprobe" {
		t.Fatalf("missing response = %+v", response)
	}
	failedLookups := lookups["ffmpeg"]
	if failedLookups != 2 {
		t.Fatalf("failed lookup count = %d, want 2", failedLookups)
	}

	installed = true
	if response := request(); response.Status != 0 || !response.Available {
		t.Fatalf("recovery response = %+v, want available", response)
	}
	if lookups["ffmpeg"] != failedLookups+1 || lookups["ffprobe"] != 2 {
		t.Fatalf("failed result was cached: %#v", lookups)
	}
}

func TestHandleFFmpegCheckReportsInvalidConfiguration(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "deepright", "integration")
	configDir := filepath.Join(filepath.Dir(executable), "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"ffmpeg":{"check":0,"install":"install"}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	previousExecutable := integrationExecutableFn
	t.Cleanup(func() { integrationExecutableFn = previousExecutable })
	integrationExecutableFn = func() (string, error) { return executable, nil }

	recorder := httptest.NewRecorder()
	handleFFmpegCheck().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/ffmpeg/check", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	var response FFmpegCheckResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != 1 || response.Content == "" {
		t.Fatalf("response = %+v, want visible configuration error", response)
	}
}
