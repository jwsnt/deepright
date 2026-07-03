package main

import (
	"connect/sandboxstate"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestGetAgentOutputIncludesGitField(t *testing.T) {
	root := filepath.Join(".", "test-case")
	data, err := GetAgentOutput(root, "test-device", 0*time.Millisecond)
	if err != nil {
		t.Fatalf("GetAgentOutput failed: %v", err)
	}

	var out AgentOutput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}

	if out.DeviceID != "test-device" {
		t.Fatalf("deviceId = %q, want test-device", out.DeviceID)
	}
	if out.Git != detectGit() {
		t.Fatalf("git = %q, want %q", out.Git, detectGit())
	}
	if out.Knowledge != nil {
		t.Fatalf("knowledge = %#v, want nil when app dir has no knowledge", out.Knowledge)
	}
	if len(out.Agents) == 0 {
		t.Fatal("agents should not be empty")
	}
}

func TestGetAgentOutputIncludesKnowledgeWhenPresent(t *testing.T) {
	appDir := t.TempDir()
	agentDir := filepath.Join(appDir, "agent")
	if err := os.MkdirAll(filepath.Join(agentDir, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "demo", "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(appDir, "data")
	db, err := os.Create(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	data, err := GetAgentOutputForApp(agentDir, appDir, "test-device", 0)
	if err != nil {
		t.Fatalf("GetAgentOutputForApp failed: %v", err)
	}

	var out AgentOutput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}

	if out.Knowledge == nil {
		t.Fatal("knowledge should not be nil")
	}
	if out.Knowledge.Path != filepath.Join(appDir, "knowledge") {
		t.Fatalf("knowledge.path = %q", out.Knowledge.Path)
	}
	if out.Knowledge.LastUpdate != 0 {
		t.Fatalf("knowledge.lastUpdate = %d, want 0", out.Knowledge.LastUpdate)
	}
}

func TestGetAgentOutputIncludesProviderFromConfig(t *testing.T) {
	root := t.TempDir()
	agentDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "config.json"), []byte(`{"description":"demo","provider":"deepseek","thinking":true,"router_disable":true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	FlushCache()
	data, err := GetAgentOutput(root, "device", 0)
	if err != nil {
		t.Fatalf("GetAgentOutput failed: %v", err)
	}

	var out AgentOutput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if len(out.Agents) != 1 {
		t.Fatalf("agents len = %d, want 1", len(out.Agents))
	}
	if got := out.Agents[0].Provider; got != "deepseek" {
		t.Fatalf("provider = %q, want %q", got, "deepseek")
	}
	if !out.Agents[0].RouterDisable {
		t.Fatal("router_disable = false, want true")
	}
}

func TestGetAgentOutputDefaultsProviderWhenConfigMissing(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}

	FlushCache()
	data, err := GetAgentOutput(root, "device", 0)
	if err != nil {
		t.Fatalf("GetAgentOutput failed: %v", err)
	}

	var out AgentOutput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if len(out.Agents) != 1 {
		t.Fatalf("agents len = %d, want 1", len(out.Agents))
	}
	if got := out.Agents[0].Provider; got != "" {
		t.Fatalf("provider = %q, want empty string", got)
	}
}

func TestDetectPluginsOnlyReturnsTopLevelExecutables(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows shell script execution is not covered in this unix-style test")
	}

	root := t.TempDir()
	appPath := filepath.Join(root, "agent-scanner")
	script := "#!/bin/sh\nif [ \"$1\" = \"list-meta\" ]; then\n  printf '%s\\n' '[{\"key\":\"feishu\",\"name\":\"飞书\"},{\"key\":\"browser\",\"name\":\"浏览器\"}]'\n  exit 0\nfi\nexit 1\n"
	if err := os.WriteFile(appPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	pluginDir := filepath.Join(root, "plugins")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}

	switch runtime.GOOS {
	case "windows":
		if err := os.WriteFile(filepath.Join(pluginDir, "alpha.exe"), []byte("fake"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pluginDir, "beta"), []byte("fake"), 0o644); err != nil {
			t.Fatal(err)
		}
	default:
		if err := os.WriteFile(filepath.Join(pluginDir, "alpha"), []byte("fake"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pluginDir, "beta.sh"), []byte("fake"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pluginDir, "gamma"), []byte("fake"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.MkdirAll(filepath.Join(pluginDir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "nested", "inner"), []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(pluginDir, "browser.pid"), []byte(itoa(os.Getpid())), 0o644); err != nil {
		t.Fatal(err)
	}

	got := detectPlugins(appPath)
	if !slices.Equal(got, []string{"browser"}) {
		t.Fatalf("plugins = %#v", got)
	}
}

func TestDetectPluginsReturnsConfiguredPluginsFromListMeta(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows shell script execution is not covered in this unix-style test")
	}

	root := t.TempDir()
	appPath := filepath.Join(root, "agent-scanner")
	script := "#!/bin/sh\nif [ \"$1\" = \"list-meta\" ]; then\n  printf '%s\\n' '[{\"key\":\"feishu\",\"name\":\"飞书\"}]'\n  exit 0\nfi\nexit 1\n"
	if err := os.WriteFile(appPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	pluginDir := filepath.Join(root, "plugins")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "feishu"), []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "browser"), []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "feishu.pid"), []byte(itoa(os.Getpid())), 0o644); err != nil {
		t.Fatal(err)
	}

	got := detectPlugins(appPath)
	if !slices.Equal(got, []string{"feishu"}) {
		t.Fatalf("plugins = %#v, want [feishu]", got)
	}
}

func TestAgentSkillsAreRefreshedEvenWhenCacheIsWarm(t *testing.T) {
	root := t.TempDir()
	agentDir := filepath.Join(root, "demo")
	skillsDir := filepath.Join(agentDir, "skills", "live")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(skillsDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(strings.Join([]string{
		"---",
		"name: live-skill",
		"description: first version",
		"---",
		"",
		"body",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	FlushCache()
	data1, err := GetAgentOutput(root, "device", time.Hour)
	if err != nil {
		t.Fatalf("GetAgentOutput first failed: %v", err)
	}
	var out1 AgentOutput
	if err := json.Unmarshal(data1, &out1); err != nil {
		t.Fatalf("unmarshal first output failed: %v", err)
	}
	if got := out1.Agents[0].Skills[0].Description; got != "first version" {
		t.Fatalf("first description = %q", got)
	}

	if err := os.WriteFile(skillPath, []byte(strings.Join([]string{
		"---",
		"name: live-skill",
		"description: second version",
		"---",
		"",
		"body",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	data2, err := GetAgentOutput(root, "device", time.Hour)
	if err != nil {
		t.Fatalf("GetAgentOutput second failed: %v", err)
	}
	var out2 AgentOutput
	if err := json.Unmarshal(data2, &out2); err != nil {
		t.Fatalf("unmarshal second output failed: %v", err)
	}
	if got := out2.Agents[0].Skills[0].Description; got != "second version" {
		t.Fatalf("second description = %q, want refreshed result", got)
	}
}

func TestAgentSkillsCompatibilitySequenceNormalizesToString(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "demo", "skills", "view-directory")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(skillDir, "SKILL.md")
	content := strings.Join([]string{
		"---",
		"name: view-directory",
		"description: list files",
		"compatibility:",
		"  - macOS (Darwin)",
		"  - zsh shell",
		"---",
		"",
		"body",
	}, "\n")
	if err := os.WriteFile(skillPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	FlushCache()
	data, err := GetAgentOutput(root, "device", time.Hour)
	if err != nil {
		t.Fatalf("GetAgentOutput failed: %v", err)
	}

	var out AgentOutput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if len(out.Agents) != 1 {
		t.Fatalf("agents len = %d, want 1", len(out.Agents))
	}
	if len(out.Agents[0].Skills) != 1 {
		t.Fatalf("skills len = %d, want 1", len(out.Agents[0].Skills))
	}
	if got := string(out.Agents[0].Skills[0].Compatibility); got != "macOS (Darwin); zsh shell" {
		t.Fatalf("compatibility = %q, want %q", got, "macOS (Darwin); zsh shell")
	}
}

func TestGitPathIsRefreshedEvenWhenCacheIsWarm(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PATH-based fake git test is unix-only")
	}

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}

	binA := filepath.Join(t.TempDir(), "git")
	if err := os.WriteFile(binA, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	binB := filepath.Join(t.TempDir(), "git")
	if err := os.WriteFile(binB, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	FlushCache()
	if err := os.Setenv("PATH", filepath.Dir(binA)); err != nil {
		t.Fatal(err)
	}
	data1, err := GetAgentOutput(root, "device", time.Hour)
	if err != nil {
		t.Fatalf("GetAgentOutput first failed: %v", err)
	}
	var out1 AgentOutput
	if err := json.Unmarshal(data1, &out1); err != nil {
		t.Fatalf("unmarshal first output failed: %v", err)
	}
	if out1.Git != binA {
		t.Fatalf("first git = %q, want %q", out1.Git, binA)
	}

	if err := os.Setenv("PATH", filepath.Dir(binB)); err != nil {
		t.Fatal(err)
	}
	data2, err := GetAgentOutput(root, "device", time.Hour)
	if err != nil {
		t.Fatalf("GetAgentOutput second failed: %v", err)
	}
	var out2 AgentOutput
	if err := json.Unmarshal(data2, &out2); err != nil {
		t.Fatalf("unmarshal second output failed: %v", err)
	}
	if out2.Git != binB {
		t.Fatalf("second git = %q, want %q", out2.Git, binB)
	}
}

func TestAgentConfigFieldsAreRefreshedEvenWhenCacheIsWarm(t *testing.T) {
	root := t.TempDir()
	agentDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(agentDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"description":"first","provider":"deepseek","thinking":true,"router_disable":true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	FlushCache()
	data1, err := GetAgentOutput(root, "device", time.Hour)
	if err != nil {
		t.Fatalf("GetAgentOutput first failed: %v", err)
	}
	var out1 AgentOutput
	if err := json.Unmarshal(data1, &out1); err != nil {
		t.Fatalf("unmarshal first output failed: %v", err)
	}
	if got := out1.Agents[0].Description; got != "first" {
		t.Fatalf("first description = %q", got)
	}
	if got := out1.Agents[0].Provider; got != "deepseek" {
		t.Fatalf("first provider = %q", got)
	}
	if !out1.Agents[0].Thinking {
		t.Fatal("first thinking = false, want true")
	}
	if !out1.Agents[0].RouterDisable {
		t.Fatal("first router_disable = false, want true")
	}

	if err := os.WriteFile(configPath, []byte(`{"description":"second","provider":"kimi","thinking":false,"router_disable":false}`), 0o644); err != nil {
		t.Fatal(err)
	}

	data2, err := GetAgentOutput(root, "device", time.Hour)
	if err != nil {
		t.Fatalf("GetAgentOutput second failed: %v", err)
	}
	var out2 AgentOutput
	if err := json.Unmarshal(data2, &out2); err != nil {
		t.Fatalf("unmarshal second output failed: %v", err)
	}
	if got := out2.Agents[0].Description; got != "second" {
		t.Fatalf("second description = %q, want refreshed result", got)
	}
	if got := out2.Agents[0].Provider; got != "kimi" {
		t.Fatalf("second provider = %q, want refreshed result", got)
	}
	if out2.Agents[0].Thinking {
		t.Fatal("second thinking = true, want false")
	}
	if out2.Agents[0].RouterDisable {
		t.Fatal("second router_disable = true, want false")
	}
}

func TestGetAgentOutputRefreshesAgentListWhenDirectoriesChange(t *testing.T) {
	root := t.TempDir()
	alphaDir := filepath.Join(root, "alpha")
	if err := os.MkdirAll(alphaDir, 0o755); err != nil {
		t.Fatal(err)
	}

	FlushCache()
	data1, err := GetAgentOutput(root, "device", time.Hour)
	if err != nil {
		t.Fatalf("GetAgentOutput first failed: %v", err)
	}
	var out1 AgentOutput
	if err := json.Unmarshal(data1, &out1); err != nil {
		t.Fatalf("unmarshal first output failed: %v", err)
	}
	if len(out1.Agents) != 1 || out1.Agents[0].AgentID != "alpha" {
		t.Fatalf("first agents = %#v, want [alpha]", out1.Agents)
	}

	if err := os.RemoveAll(alphaDir); err != nil {
		t.Fatal(err)
	}
	betaDir := filepath.Join(root, "beta")
	if err := os.MkdirAll(betaDir, 0o755); err != nil {
		t.Fatal(err)
	}

	data2, err := GetAgentOutput(root, "device", time.Hour)
	if err != nil {
		t.Fatalf("GetAgentOutput second failed: %v", err)
	}
	var out2 AgentOutput
	if err := json.Unmarshal(data2, &out2); err != nil {
		t.Fatalf("unmarshal second output failed: %v", err)
	}
	if len(out2.Agents) != 1 || out2.Agents[0].AgentID != "beta" {
		t.Fatalf("second agents = %#v, want [beta]", out2.Agents)
	}
}

func TestGetAgentOutputCachesVersionButRefreshesSandboxByChat(t *testing.T) {
	root := t.TempDir()
	agentDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "config.json"), []byte(`{"version":"v1"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	appDir := t.TempDir()
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(appDir, "data"))
	if err != nil {
		t.Fatalf("open sandbox db: %v", err)
	}
	defer db.Close()

	if _, err := sandboxstate.Set(db, "demo", "chat-1", sandboxstate.ModeFilePick); err != nil {
		t.Fatalf("seed sandbox state: %v", err)
	}

	FlushCache()
	data1, err := GetAgentOutputForAppAndChat(root, appDir, "device", time.Hour, "chat-1")
	if err != nil {
		t.Fatalf("GetAgentOutputForAppAndChat first failed: %v", err)
	}
	var out1 AgentOutput
	if err := json.Unmarshal(data1, &out1); err != nil {
		t.Fatalf("unmarshal first output failed: %v", err)
	}
	if got := out1.Agents[0].Version; got != "v1" {
		t.Fatalf("first version = %q, want v1", got)
	}
	if got := out1.Agents[0].Sandbox; got != sandboxstate.ModeFilePick {
		t.Fatalf("first sandbox = %q, want %q", got, sandboxstate.ModeFilePick)
	}

	if err := os.WriteFile(filepath.Join(agentDir, "config.json"), []byte(`{"version":"v2"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := sandboxstate.Set(db, "demo", "chat-1", sandboxstate.ModeNet); err != nil {
		t.Fatalf("update sandbox state: %v", err)
	}

	data2, err := GetAgentOutputForAppAndChat(root, appDir, "device", time.Hour, "chat-1")
	if err != nil {
		t.Fatalf("GetAgentOutputForAppAndChat second failed: %v", err)
	}
	var out2 AgentOutput
	if err := json.Unmarshal(data2, &out2); err != nil {
		t.Fatalf("unmarshal second output failed: %v", err)
	}
	if got := out2.Agents[0].Version; got != "v1" {
		t.Fatalf("second version = %q, want cached v1", got)
	}
	if got := out2.Agents[0].Sandbox; got != sandboxstate.ModeNet {
		t.Fatalf("second sandbox = %q, want %q", got, sandboxstate.ModeNet)
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	buf := [32]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
