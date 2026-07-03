package connectsvc

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

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
