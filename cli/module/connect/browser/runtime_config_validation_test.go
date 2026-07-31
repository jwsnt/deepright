package main

import (
	"os"
	"path/filepath"
	"runtimepaths"
	"testing"
)

func TestBrowserIntegrationPluginsRootUsesManagedMacRuntimeDirWhenAppDirIsMissing(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	browserRuntimeGOOSFn = func() string { return "darwin" }
	browserUserHomeDirFn = func() (string, error) { return homeDir, nil }

	connectBin, _ := writeBundledConnectBinFixture(t, homeDir, map[string]string{})
	root, ok, err := browserIntegrationPluginsRootFromConnectBin(connectBin)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected a plugins root")
	}
	want := filepath.Join(runtimepaths.MacAppRuntimeBaseDir(homeDir, runtimepaths.DeepRightMacBundleIdentifier, runtimepaths.DeepRightAppName), "plugins")
	if root != want {
		t.Fatalf("plugins root = %q, want %q", root, want)
	}
}

func TestBrowserIntegrationPluginsRootUsesManagedWSLRuntimeDirWhenAppDirIsMissing(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	homeDir := t.TempDir()
	browserRuntimeGOOSFn = func() string { return "linux" }
	browserUserHomeDirFn = func() (string, error) { return homeDir, nil }
	browserWSLDetectFn = func() (bool, error) { return true, nil }

	appDir := t.TempDir()
	connectBin := filepath.Join(appDir, "integration")
	configPath := filepath.Join(appDir, "config", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(connectBin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	root, ok, err := browserIntegrationPluginsRootFromConnectBin(connectBin)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected a plugins root")
	}
	want := filepath.Join(homeDir, "deepright", "plugins")
	if root != want {
		t.Fatalf("plugins root = %q, want %q", root, want)
	}
}
