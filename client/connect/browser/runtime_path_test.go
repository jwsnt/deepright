package main

import (
	"os"
	"path/filepath"
	"runtimepaths"
	"testing"
)

func TestBrowserResolveIntegrationRuntimePathPrefersBundledConfigDirectory(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	bundleRoot := filepath.Join(t.TempDir(), "DeepRight.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	runtimeDir := filepath.Join(runtimepaths.MacAppRuntimeBaseDir(homeDir, runtimepaths.DeepRightMacBundleIdentifier, runtimepaths.DeepRightAppName), "config")
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

func TestBrowserResolveRuntimeConfigPathUsesRecordedConnectBin(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	exeDir := t.TempDir()
	browserExecutablePathFn = func() (string, error) {
		return writeBrowserBinaryFixture(t, exeDir), nil
	}
	connectBin, runtimePath := writeBundledConnectBinFixture(t, homeDir, map[string]string{
		"app-dir": filepath.Join(t.TempDir(), "runtime-app"),
	})
	writeBrowserRuntimeRecordFixture(t, connectBin)

	got, ok, err := browserResolveRuntimeConfigPath(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected recorded runtime config path")
	}
	if got != runtimePath {
		t.Fatalf("runtime path = %q, want %q", got, runtimePath)
	}
}

func TestBrowserResolveRuntimeConfigPathFallsBackToBrowserBinaryWhenRuntimeRecordMissing(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

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
	if err := os.WriteFile(runtimePath, []byte("{\"app-dir\":\"/runtime\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, ok, err := browserResolveRuntimeConfigPath(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected inferred runtime config path")
	}
	want := runtimePath
	if resolved, err := filepath.EvalSymlinks(runtimePath); err == nil {
		want = resolved
	}
	if got != want {
		t.Fatalf("runtime path = %q, want %q", got, want)
	}
}
