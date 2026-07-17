package agentcore

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseInstallApps(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"empty", "", nil},
		{"single", "git", []string{"git"}},
		{"multiple", "git,curl, vim ", []string{"git", "curl", "vim"}},
		{"with spaces", "  git , curl ", []string{"git", "curl"}},
		{"dedup", "git, git, curl", []string{"git", "curl"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseInstallApps(tt.raw)
			if len(got) != len(tt.want) {
				t.Errorf("ParseInstallApps() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ParseInstallApps()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSkillNames(t *testing.T) {
	got := SkillNames([]Skill{
		{Name: "alpha"},
		{Name: "  "},
		{Name: "beta"},
	})
	want := []string{"alpha", "beta"}
	if len(got) != len(want) {
		t.Fatalf("SkillNames() len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SkillNames()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestMergeInstallApps(t *testing.T) {
	tests := []struct {
		name   string
		base   []string
		extras []string
		want   []string
	}{
		{"empty base", nil, nil, nil},
		{"base only", []string{"git"}, nil, []string{"git"}},
		{"merge with extras", []string{"git"}, []string{"curl"}, []string{"git", "curl"}},
		{"dedup across base and extras", []string{"git"}, []string{"git"}, []string{"git"}},
		{"with spaces in base", []string{"  git "}, []string{}, []string{"git"}},
		{"empty strings filtered", []string{"", "git"}, []string{}, []string{"git"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeInstallApps(tt.base, tt.extras...)
			if len(got) != len(tt.want) {
				t.Errorf("MergeInstallApps() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("MergeInstallApps()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestResolveExecutablePath(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"empty", ""},
		{"simple name", "git"},
		{"with quotes", `"/usr/bin/git"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveExecutablePath(tt.path)
			if tt.path == "" {
				if got != "" {
					t.Errorf("ResolveExecutablePath('') = %q, want ''", got)
				}
			}
		})
	}
}

func TestFlushCache(t *testing.T) {
	// Just verify FlushCache does not panic
	FlushCache()
	FlushCache()
}

func TestListAgentIDsReadsImmediateSortedDirectoriesOnly(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"zeta", "alpha", "nested"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "nested", "child"), 0o755); err != nil {
		t.Fatalf("mkdir nested child: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("not an agent"), 0o644); err != nil {
		t.Fatalf("write root file: %v", err)
	}

	ids, err := ListAgentIDs(root)
	if err != nil {
		t.Fatalf("ListAgentIDs: %v", err)
	}
	want := []string{"alpha", "nested", "zeta"}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("agent IDs = %#v, want %#v", ids, want)
	}
}

func TestGenerateDeviceID(t *testing.T) {
	id1 := GenerateDeviceID()
	id2 := GenerateDeviceID()

	if id1 == "" {
		t.Fatal("GenerateDeviceID() returned empty string")
	}
	if id1 != id2 {
		t.Errorf("GenerateDeviceID() not deterministic: %q vs %q", id1, id2)
	}
}

func TestDetectTerminal(t *testing.T) {
	term := DetectTerminal()
	if term == "" {
		t.Log("DetectTerminal() returned empty (expected if SHELL not set)")
	}
}

func TestPluginPIDFiles(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name             string
		defaultPluginDir string
		key              string
		callback         string
	}{
		{"basic", tmpDir, "myplugin", ""},
		{"browser with callback", tmpDir, "browser", tmpDir + "/callback.sh"},
		{"empty key", tmpDir, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := PluginPIDFiles(tt.defaultPluginDir, tt.key, tt.callback)
			if len(files) == 0 {
				t.Error("expected at least one pid file")
			}
		})
	}
}

func TestPluginStarted(t *testing.T) {
	// With non-existent plugin dir, should return false
	got := PluginStarted("/nonexistent", "test-plugin", "")
	if got {
		t.Error("PluginStarted() = true, want false for nonexistent plugin")
	}
}

func TestDetectRequiredInstallApps(t *testing.T) {
	apps := DetectRequiredInstallApps()
	// This depends on whether git is installed on the system
	// Just verify it returns a slice and doesn't panic
	_ = apps
}

func TestDetectSys(t *testing.T) {
	sys := DetectSys()
	if sys == "" {
		t.Error("DetectSys() returned empty string")
	}
}

func TestDetectGateway(t *testing.T) {
	// This may or may not work depending on environment
	_ = DetectGateway()
}

func TestDetectGit(t *testing.T) {
	// Git detection may fail in some environments
	_ = DetectGit()
}

func TestDetectPython3(t *testing.T) {
	_ = DetectPython3()
}

func TestReadPluginPID(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	// Write a valid PID
	if err := os.WriteFile(pidFile, []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}

	pid, err := readPluginPID(pidFile)
	if err != nil {
		t.Fatalf("readPluginPID() error: %v", err)
	}
	if pid != 12345 {
		t.Errorf("readPluginPID() = %d, want 12345", pid)
	}

	// Invalid PID
	if err := os.WriteFile(pidFile, []byte("invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = readPluginPID(pidFile)
	if err == nil {
		t.Error("readPluginPID() expected error for invalid PID")
	}

	// Non-existent file
	_, err = readPluginPID("/nonexistent/pid")
	if err == nil {
		t.Error("readPluginPID() expected error for non-existent file")
	}
}

func TestNormalizeAppDir(t *testing.T) {
	tests := []struct {
		name   string
		appDir string
	}{
		{"empty", ""},
		{"with spaces", "  /tmp "},
		{"relative", "."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeAppDir(tt.appDir)
			if tt.appDir == "" {
				if result != "" {
					t.Errorf("normalizeAppDir('') = %q, want ''", result)
				}
			}
		})
	}
}

func TestReadFileContent(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")

	if err := os.WriteFile(file, []byte("  hello world  \n"), 0o644); err != nil {
		t.Fatal(err)
	}

	content := readFileContent(file)
	if content != "hello world" {
		t.Errorf("readFileContent() = %q, want %q", content, "hello world")
	}

	empty := readFileContent("/nonexistent")
	if empty != "" {
		t.Errorf("readFileContent() for non-existent = %q, want ''", empty)
	}
}

func TestReadAgentConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	// Non-existent config
	cfg := readAgentConfig(tmpDir)
	if cfg.Description != "" || !cfg.RouterDisable {
		t.Errorf("expected empty config, got %+v", cfg)
	}

	// Valid config
	configContent := `{"description":"test agent","provider":"openai","router_disable":false,"thinking":false,"version":"v1.2.3"}`
	if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg = readAgentConfig(tmpDir)
	if cfg.Description != "test agent" || cfg.Provider != "openai" || cfg.RouterDisable || cfg.Thinking || cfg.Version != "v1.2.3" {
		t.Errorf("readAgentConfig() = %+v", cfg)
	}

	// Legacy swarm config remains readable and is inverted into router_disable
	if err := os.WriteFile(configFile, []byte(`{"swarm":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg = readAgentConfig(tmpDir)
	if cfg.RouterDisable {
		t.Errorf("legacy swarm mapping failed: %+v", cfg)
	}

	// Invalid JSON config
	// Write invalid JSON
	if err := os.WriteFile(configFile, []byte("{invalid}"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg = readAgentConfig(tmpDir)
	if cfg.Description != "" || !cfg.RouterDisable {
		t.Errorf("expected empty config for invalid JSON, got %+v", cfg)
	}
}

func TestCloneOutputNil(t *testing.T) {
	if cloned := cloneOutput(nil); cloned != nil {
		t.Error("cloneOutput(nil) should return nil")
	}
}
