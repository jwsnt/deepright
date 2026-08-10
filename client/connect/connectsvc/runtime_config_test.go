package connectsvc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRuntimeConfigPathFromConnectBinPrefersConfigJSON(t *testing.T) {
	root := t.TempDir()
	connectBin := filepath.Join(root, "integration")
	if err := os.WriteFile(connectBin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "config", "config.json")
	if err := os.WriteFile(configPath, []byte(`{"app-dir":"/config"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, ok := ResolveRuntimeConfigPathFromConnectBin(connectBin)
	if !ok {
		t.Fatal("expected runtime config path")
	}
	if got != configPath {
		t.Fatalf("path = %q, want %q", got, configPath)
	}
}

func TestResolveRuntimeConfigPathFromConnectBinIgnoresLegacyRuntimeJSON(t *testing.T) {
	root := t.TempDir()
	connectBin := filepath.Join(root, "integration")
	if err := os.WriteFile(connectBin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "runtime.json"), []byte(`{"app-dir":"/legacy"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, ok := ResolveRuntimeConfigPathFromConnectBin(connectBin)
	if ok {
		t.Fatalf("path = %q, want no match", got)
	}
}

func TestResolveRuntimeConfigPathNearBinaryPrefersConfigJSON(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "plugins")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	binaryPath := filepath.Join(pluginDir, "browser")
	if err := os.WriteFile(binaryPath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "config", "config.json")
	if err := os.WriteFile(configPath, []byte(`{"app-dir":"/config"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, ok := ResolveRuntimeConfigPathNearBinary(binaryPath)
	if !ok {
		t.Fatal("expected nearby runtime config path")
	}
	if got != configPath {
		t.Fatalf("path = %q, want %q", got, configPath)
	}
}
