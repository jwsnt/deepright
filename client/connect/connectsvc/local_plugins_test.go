package connectsvc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunLocalPluginCommandTimeoutLeavesProcessRunning(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	marker := filepath.Join(t.TempDir(), "completed")
	script := filepath.Join(t.TempDir(), "slow-plugin")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 0.08\nprintf survived > \"$2\"\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := runLocalPluginCommand(ctx, script, "name", marker)
	var timeoutErr *localPluginCommandTimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("runLocalPluginCommand() error = %v, want timeout error", err)
	}
	if !strings.Contains(timeoutErr.Error(), "process was left running") {
		t.Fatalf("timeout error = %q, want left-running reason", timeoutErr)
	}

	deadline := time.Now().Add(time.Second)
	for {
		if content, readErr := os.ReadFile(marker); readErr == nil {
			if string(content) != "survived" {
				t.Fatalf("marker = %q, want survived", content)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out probe did not continue running")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestRunLocalPluginCommandRecordsExitSignal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	script := filepath.Join(t.TempDir(), "signal-plugin")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nkill -TERM $$\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := runLocalPluginCommand(ctx, script, "name")
	var exitErr *localPluginCommandExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("runLocalPluginCommand() error = %v, want exit error", err)
	}
	if !strings.Contains(exitErr.Error(), "signal") || !strings.Contains(exitErr.Error(), "command=name") {
		t.Fatalf("exit error = %q, want signal and command reason", exitErr)
	}
}

func TestDefaultLocalPluginDir(t *testing.T) {
	dir, err := DefaultLocalPluginDir()
	if err != nil {
		t.Fatalf("DefaultLocalPluginDir() error = %v", err)
	}
	if !strings.HasSuffix(dir, string(filepath.Separator)+"plugins") {
		t.Fatalf("DefaultLocalPluginDir() = %q, want suffix /plugins", dir)
	}
}

func TestListLocalPluginsSupportsExecutableScriptExtensions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	pluginDir := t.TempDir()
	writePluginScriptWithScope(t, filepath.Join(pluginDir, "remote.py"), `{"name":"远程"}`, `[{"exec_timeout":""}]`, `[]`)

	items, err := ListLocalPlugins(LocalPluginOptions{
		ResolveDir: func() (string, error) { return pluginDir, nil },
	})
	if err != nil {
		t.Fatalf("ListLocalPlugins() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("ListLocalPlugins() len = %d, want 1", len(items))
	}
	if items[0].Key != "remote" {
		t.Fatalf("plugin key = %q, want remote", items[0].Key)
	}
	if items[0].Name != "远程" {
		t.Fatalf("plugin name = %q, want 远程", items[0].Name)
	}
	if items[0].Scope == nil {
		t.Fatal("plugin scope = nil, want empty slice")
	}
	if len(items[0].Scope) != 0 {
		t.Fatalf("plugin scope = %+v, want []", items[0].Scope)
	}
}

func TestListLocalPluginMetaWithServiceMergesSavedConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	pluginDir := t.TempDir()
	writePluginScriptWithScope(t, filepath.Join(pluginDir, "remote.py"), `{"name":"远程"}`, `[{"exec_timeout":""}]`, `[]`)
	restorePluginDir := SetPluginDirResolverForTest(func() (string, error) { return pluginDir, nil })
	defer restorePluginDir()

	agentDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(agentDir, "A"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	dbPath := filepath.Join(t.TempDir(), "data")
	seedTokenStore(t, dbPath, agentDir, "OpenAI")
	svc, err := NewService(Options{
		DBPath:   dbPath,
		AgentDir: agentDir,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer svc.Close()

	_, err = svc.CreateMeta(MetaInput{
		Key:           "remote",
		Meta:          `{"host":"127.0.0.1"}`,
		Callback:      "plugins/remote",
		AgentID:       "A",
		Model:         "OpenAI",
		Thinking:      true,
		RouterDisable: false,
	})
	if err != nil {
		t.Fatalf("CreateMeta() error = %v", err)
	}

	items, err := ListLocalPluginMetaWithService(svc, LocalPluginOptions{
		ResolveDir: func() (string, error) { return pluginDir, nil },
	})
	if err != nil {
		t.Fatalf("ListLocalPluginMetaWithService() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("ListLocalPluginMetaWithService() len = %d, want 1", len(items))
	}
	if items[0].Key != "remote" {
		t.Fatalf("plugin key = %q, want remote", items[0].Key)
	}
	if items[0].Meta == nil || items[0].Meta["host"] != "127.0.0.1" {
		t.Fatalf("plugin meta = %+v, want host", items[0].Meta)
	}
	if !strings.HasSuffix(items[0].Callback, string(filepath.Separator)+"remote") && items[0].Callback != "remote" {
		t.Fatalf("plugin callback = %q, want remote callback path", items[0].Callback)
	}
	if items[0].RouterDisable {
		t.Fatal("plugin router_disable = true, want false")
	}
	if !items[0].Thinking {
		t.Fatal("plugin thinking = false, want true")
	}
}
