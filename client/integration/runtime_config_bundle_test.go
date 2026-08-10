package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBundledConfigIsAlwaysReadInsteadOfLegacyRuntimeConfig(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("bundle runtime layout is only used on darwin")
	}

	bundleRoot := filepath.Join(t.TempDir(), "DeepRight.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	resourcesDir := filepath.Join(bundleRoot, "Contents", "Resources")
	bundledConfigPath := filepath.Join(resourcesDir, "config", "config.json")
	if err := os.MkdirAll(filepath.Dir(bundledConfigPath), 0o755); err != nil {
		t.Fatal(err)
	}
	bundledConfig := []byte(`{"version":"2","miniapp":{"function":"new"},"host":"https://bundle.example.com"}` + "\n")
	if err := os.WriteFile(bundledConfigPath, bundledConfig, 0o644); err != nil {
		t.Fatal(err)
	}

	originalExecutable := integrationExecutableFn
	originalHome := integrationUserHomeFn
	defer func() {
		integrationExecutableFn = originalExecutable
		integrationUserHomeFn = originalHome
	}()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(macOSDir, "integration"), nil
	}
	home := t.TempDir()
	integrationUserHomeFn = func() (string, error) { return home, nil }

	legacyRuntimePath := filepath.Join(integrationTestMacRuntimeBaseDir(home), "config", "config.json")
	if err := os.MkdirAll(filepath.Dir(legacyRuntimePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyRuntimePath, []byte(`{"version":"1","miniapp":{"function":"stale"},"host":"https://stale.example.com"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	raw, path, err := readIntegrationStartupConfigRaw()
	if err != nil {
		t.Fatalf("read bundled config: %v", err)
	}
	if path != bundledConfigPath || raw["version"] != "2" {
		t.Fatalf("read config = %#v from %q, want bundled config", raw, path)
	}
	if got := readRuntimeConfig(); got["host"] != "https://bundle.example.com" {
		t.Fatalf("host = %q, want bundled value", got["host"])
	}
	if got, err := os.ReadFile(bundledConfigPath); err != nil || !bytes.Equal(got, bundledConfig) {
		t.Fatalf("bundled config changed: data=%q err=%v", got, err)
	}
}

func TestStartupConfigReadDoesNotCreateRuntimeConfig(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("bundle runtime layout is only used on darwin")
	}

	bundleRoot := filepath.Join(t.TempDir(), "DeepRight.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	resourcesDir := filepath.Join(bundleRoot, "Contents", "Resources")
	if err := os.MkdirAll(filepath.Join(resourcesDir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resourcesDir, "config", "config.json"), []byte(`{"host":"https://bundle.example.com"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	originalExecutable := integrationExecutableFn
	originalHome := integrationUserHomeFn
	defer func() {
		integrationExecutableFn = originalExecutable
		integrationUserHomeFn = originalHome
	}()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(macOSDir, "integration"), nil
	}
	home := t.TempDir()
	integrationUserHomeFn = func() (string, error) { return home, nil }

	if _, _, err := loadIntegrationStartupOptions(); err != nil {
		t.Fatalf("load startup config: %v", err)
	}
	if _, err := os.Stat(filepath.Join(integrationTestMacRuntimeBaseDir(home), "config", "config.json")); !os.IsNotExist(err) {
		t.Fatalf("runtime config must not be created, stat err = %v", err)
	}
}
