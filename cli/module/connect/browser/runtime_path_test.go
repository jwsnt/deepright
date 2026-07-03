package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserResolveIntegrationRuntimePathPrefersBundledConfigDirectory(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	bundleRoot := filepath.Join(t.TempDir(), "DeepRight.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	runtimeDir := filepath.Join(homeDir, "Library", "Application Support", "deepright", "config")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	connectBin := filepath.Join(macOSDir, "integration")
	if err := os.WriteFile(connectBin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	runtimePath := filepath.Join(runtimeDir, "config.json")
	if err := os.WriteFile(runtimePath, []byte("{\"app-dir\":\"/runtime\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, ok := browserResolveIntegrationRuntimePath(connectBin)
	if !ok {
		t.Fatal("expected config/config.json path")
	}
	if got != runtimePath {
		t.Fatalf("runtime path = %q, want %q", got, runtimePath)
	}
}
