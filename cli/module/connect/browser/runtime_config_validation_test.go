package main

import (
	"os"
	"path/filepath"
	"runtimepaths"
	"testing"
)

func TestBrowserIntegrationPluginsRootUsesFixedMacRuntimeDirIgnoringConfiguredAppDirs(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	browserRuntimeGOOSFn = func() string { return "darwin" }
	browserUserHomeDirFn = func() (string, error) { return homeDir, nil }

	connectBin, _ := writeBundledConnectBinFixture(t, homeDir, map[string]string{
		"app-dir": filepath.Join(t.TempDir(), "ignored-app-dir"),
		"app":     filepath.Join(t.TempDir(), "ignored-app"),
	})
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

func TestBrowserIntegrationPluginsRootUsesFixedWSLRuntimeDirIgnoringConfiguredAppDirs(t *testing.T) {
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
	configJSON := `{"app-dir":"/ignored/app-dir","app":"/ignored/app"}` + "\n"
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
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

func TestBrowserIntegrationPluginsRootUsesFixedDirectoryReleaseIgnoringConfiguredAppDirs(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	browserRuntimeGOOSFn = func() string { return "linux" }
	browserWSLDetectFn = func() (bool, error) { return false, nil }

	appDir := t.TempDir()
	connectBin := filepath.Join(appDir, "integration")
	configPath := filepath.Join(appDir, "config", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(connectBin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	configJSON := `{"app-dir":"/ignored/app-dir","app":"/ignored/app"}` + "\n"
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	root, ok, err := browserIntegrationPluginsRootFromConnectBin(connectBin)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected a plugins root")
	}
	want := filepath.Join(appDir, "plugins")
	if root != want {
		t.Fatalf("plugins root = %q, want %q", root, want)
	}
}

func TestBrowserInferMetaLookupBinaryUsesFixedDirectoryIgnoringConfiguredApp(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	browserRuntimeGOOSFn = func() string { return "linux" }
	browserWSLDetectFn = func() (bool, error) { return false, nil }

	appDir := t.TempDir()
	browserPath := filepath.Join(appDir, "plugins", "browser")
	configPath := filepath.Join(appDir, "config", "config.json")
	fixedIntegration := filepath.Join(appDir, "integration")
	ignoredIntegration := filepath.Join(t.TempDir(), "integration")
	if err := os.MkdirAll(filepath.Dir(browserPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{browserPath, fixedIntegration, ignoredIntegration} {
		if err := os.WriteFile(path, []byte("fake"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	configJSON := `{"app":"` + ignoredIntegration + `"}` + "\n"
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	browserExecutablePathFn = func() (string, error) { return browserPath, nil }

	got, err := browserInferMetaLookupBinary(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	want := fixedIntegration
	if resolved, resolveErr := filepath.EvalSymlinks(fixedIntegration); resolveErr == nil {
		want = resolved
	}
	if got != want {
		t.Fatalf("meta lookup binary = %q, want fixed path %q", got, want)
	}
}
