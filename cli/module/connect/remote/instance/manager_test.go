package instance

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveIntegrationRuntimePathPrefersBundledConfigDirectory(t *testing.T) {
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

	got, ok := resolveIntegrationRuntimePath(connectBin)
	if !ok {
		t.Fatal("expected config/config.json path")
	}
	if got != runtimePath {
		t.Fatalf("runtime path = %q, want %q", got, runtimePath)
	}
}

func TestRuntimePathsUseBundledRuntimePluginDir(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	bundleRoot := filepath.Join(t.TempDir(), "DeepRight.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	runtimeDir := filepath.Join(homeDir, "Library", "Application Support", "deepright", "config")
	selfPath := filepath.Join(t.TempDir(), "remote")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(selfPath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	connectBin := filepath.Join(macOSDir, "integration")
	if err := os.WriteFile(connectBin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	appDir := filepath.Dir(runtimeDir)
	if err := os.WriteFile(filepath.Join(runtimeDir, "config.json"), []byte(fmt.Sprintf("{\"app-dir\":%q}\n", appDir)), 0o644); err != nil {
		t.Fatal(err)
	}

	oldExec := executablePathFn
	t.Cleanup(func() { executablePathFn = oldExec })
	executablePathFn = func() (string, error) { return selfPath, nil }

	got, err := runtimePaths(map[string]string{"connect-bin": connectBin})
	if err != nil {
		t.Fatalf("runtimePaths() error = %v", err)
	}
	wantRoot := filepath.Join(appDir, "plugins")
	if got.RootDir != wantRoot {
		t.Fatalf("root dir = %q, want %q", got.RootDir, wantRoot)
	}
	if got.BinaryPath != filepath.Join(wantRoot, "remote") {
		t.Fatalf("binary path = %q, want %q", got.BinaryPath, filepath.Join(wantRoot, "remote"))
	}
}
