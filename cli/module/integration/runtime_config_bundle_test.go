package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteRuntimeConfigStoresBundledRuntimeConfigOutsideSignedBundle(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("bundle runtime layout is only used on darwin")
	}

	bundleRoot := filepath.Join(t.TempDir(), "DeepRight.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	resourcesDir := filepath.Join(bundleRoot, "Contents", "Resources")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(resourcesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	originalExecutable := integrationExecutableFn
	defer func() {
		integrationExecutableFn = originalExecutable
	}()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(macOSDir, "integration"), nil
	}

	writeRuntimeConfig(map[string]interface{}{
		"app":           filepath.Join(macOSDir, "integration"),
		"app-dir":       resourcesDir,
		"resources-dir": resourcesDir,
		"db":            filepath.Join(resourcesDir, "data"),
	})

	configPath := filepath.Join(resourcesDir, "config", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config/config.json: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config/config.json: %v", err)
	}
	wantAppDir := resourcesDir
	if runtimeDir := strings.TrimSpace(integrationBundleRuntimeBaseDir()); runtimeDir != "" {
		wantAppDir = runtimeDir
	}
	if cfg["app-dir"] != wantAppDir {
		t.Fatalf("app-dir = %v, want %q", cfg["app-dir"], wantAppDir)
	}
	if cfg["resources-dir"] != resourcesDir {
		t.Fatalf("resources-dir = %v, want %q", cfg["resources-dir"], resourcesDir)
	}
}

func TestReadRuntimeConfigUsesBundledConfigDirectory(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("bundle runtime layout is only used on darwin")
	}

	bundleRoot := filepath.Join(t.TempDir(), "DeepRight.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	resourcesDir := filepath.Join(bundleRoot, "Contents", "Resources")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(resourcesDir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resourcesDir, "config", "config.json"), []byte("{\"app-dir\":\"/runtime\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	originalExecutable := integrationExecutableFn
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		integrationExecutableFn = originalExecutable
		_ = os.Chdir(originalWD)
	}()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(macOSDir, "integration"), nil
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}

	cfg := readRuntimeConfig()
	if cfg == nil {
		t.Fatal("expected runtime config")
	}
	if cfg["app-dir"] != "/runtime" {
		t.Fatalf("app-dir = %q, want /runtime", cfg["app-dir"])
	}
}
