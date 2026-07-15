package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBrowserResolveLauncherScriptPathRecreatesMissingLauncher(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	pluginDir := t.TempDir()
	browserExecutablePathFn = func() (string, error) {
		return filepath.Join(pluginDir, "browser"), nil
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "browser"), []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}

	scriptPath, err := browserResolveLauncherScriptPath(nil)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, "DEEPRIGHT_BROWSER_BIN") {
		t.Fatalf("launcher script missing browser env fallback: %s", text)
	}
	if !strings.Contains(text, `exec "$BROWSER_BIN" __wsl-instance acquire "$@"`) {
		t.Fatalf("launcher script missing acquire command: %s", text)
	}
}
