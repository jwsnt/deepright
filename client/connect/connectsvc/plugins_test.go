package connectsvc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func testParamJSON(keys ...string) string {
	items := make([]map[string]string, 0, len(keys))
	for _, key := range keys {
		items = append(items, map[string]string{key: ""})
	}
	data, err := json.Marshal(items)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func testParamFields(keys ...string) PluginParamFields {
	fields := make(PluginParamFields, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, PluginParamField{Key: key})
	}
	return fields
}

func assertParamKeys(t *testing.T, got PluginParamFields, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("param len = %d, want %d: %+v", len(got), len(want), got)
	}
	for idx, key := range want {
		if got[idx].Key != key {
			t.Fatalf("param[%d].Key = %q, want %q", idx, got[idx].Key, key)
		}
	}
}

func TestListPluginsReadsTopLevelExecutables(t *testing.T) {
	pluginDir := t.TempDir()
	writePluginScript(t, filepath.Join(pluginDir, "a"), `"IM"`, testParamJSON("key"))
	writePluginScript(t, filepath.Join(pluginDir, "b"), `"APP"`, `[{"token":"","ticket":""}]`)
	if err := os.WriteFile(filepath.Join(pluginDir, "README.txt"), []byte("skip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(pluginDir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	writePluginScript(t, filepath.Join(pluginDir, "nested", "c"), `"NESTED"`, testParamJSON("ignored"))

	restore := overridePluginDir(pluginDir)
	defer restore()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := RunCLI([]string{"list-plugins", "--connect-cache", "10000"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("list-plugins failed: %s", stderr.String())
	}

	var items []PluginInfo
	if err := json.Unmarshal(stdout.Bytes(), &items); err != nil {
		t.Fatalf("decode list-plugins output: %v; raw=%s", err, stdout.String())
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 plugins, got %d: %+v", len(items), items)
	}
	if items[0].Name != "IM" {
		t.Fatalf("unexpected first plugin: %+v", items[0])
	}
	assertParamKeys(t, items[0].Param, "key")
	if items[1].Name != "APP" {
		t.Fatalf("unexpected second plugin: %+v", items[1])
	}
	assertParamKeys(t, items[1].Param, "token", "ticket")
}

func TestListPluginsSkipsRuntimeArtifactsEvenWhenExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	pluginDir := t.TempDir()
	writePluginScript(t, filepath.Join(pluginDir, "browser"), `{"key":"browser","name":"浏览器"}`, `[]`)
	if err := os.WriteFile(filepath.Join(pluginDir, "browser.log"), []byte("not a plugin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "browser.pid"), []byte("123"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "browser_instance.json"), []byte("{}"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "User_Data.zip"), []byte("not a plugin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "data"), []byte("SQLite format 3\x00"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "data-wal"), []byte("not a plugin"), 0o755); err != nil {
		t.Fatal(err)
	}

	restore := overridePluginDir(pluginDir)
	defer restore()

	items, err := ListPlugins(map[string]string{"connect-cache": "0"})
	if err != nil {
		t.Fatalf("ListPlugins failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 plugin, got %+v", items)
	}
	if items[0].Key != "browser" || items[0].Name != "浏览器" {
		t.Fatalf("unexpected plugin: %+v", items[0])
	}
}

func TestListPluginsUsesCacheAcrossExecutions(t *testing.T) {
	pluginDir := t.TempDir()
	pluginPath := filepath.Join(pluginDir, "a")
	writePluginScript(t, pluginPath, `"IM"`, testParamJSON("key"))

	restore := overridePluginDir(pluginDir)
	defer restore()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := RunCLI([]string{"list-plugins", "--connect-cache", "60000"}, stdout, stderr); code != 0 {
		t.Fatalf("first list-plugins failed: %s", stderr.String())
	}

	writePluginScript(t, pluginPath, `"UPDATED"`, testParamJSON("new"))

	stdout.Reset()
	stderr.Reset()
	if code := RunCLI([]string{"list-plugins", "--connect-cache", "60000"}, stdout, stderr); code != 0 {
		t.Fatalf("second list-plugins failed: %s", stderr.String())
	}
	var cached []PluginInfo
	if err := json.Unmarshal(stdout.Bytes(), &cached); err != nil {
		t.Fatalf("decode cached output: %v; raw=%s", err, stdout.String())
	}
	if len(cached) != 1 || cached[0].Name != "IM" {
		t.Fatalf("expected cached plugin result, got %+v", cached)
	}
	assertParamKeys(t, cached[0].Param, "key")

	time.Sleep(5 * time.Millisecond)
	stdout.Reset()
	stderr.Reset()
	if code := RunCLI([]string{"list-plugins", "--connect-cache", "1"}, stdout, stderr); code != 0 {
		t.Fatalf("third list-plugins failed: %s", stderr.String())
	}
	var refreshed []PluginInfo
	if err := json.Unmarshal(stdout.Bytes(), &refreshed); err != nil {
		t.Fatalf("decode refreshed output: %v; raw=%s", err, stdout.String())
	}
	if len(refreshed) != 1 || refreshed[0].Name != "UPDATED" {
		t.Fatalf("expected refreshed plugin result, got %+v", refreshed)
	}
	assertParamKeys(t, refreshed[0].Param, "new")
}

func TestListPluginsReturnsScanErrorEvenWhenCacheExists(t *testing.T) {
	pluginDir := t.TempDir()
	restore := overridePluginDir(pluginDir)
	defer restore()

	cached := []PluginInfo{{
		Key:   "feishu",
		Name:  "飞书",
		Param: testParamFields("appId", "appSecret"),
	}}
	if err := writePluginCache(pluginDir, cached); err != nil {
		t.Fatalf("write cache: %v", err)
	}

	pluginPath := filepath.Join(pluginDir, "feishu")
	if err := os.WriteFile(pluginPath, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	items, err := ListPlugins(map[string]string{"connect-cache": "0"})
	if err == nil {
		t.Fatalf("ListPlugins unexpectedly succeeded with stale cache: %+v", items)
	}
	if !strings.Contains(err.Error(), "inspect plugin feishu") {
		t.Fatalf("err = %v, want inspect plugin feishu", err)
	}
}

func TestMetaToConfigDoesNotSynthesizeDisplayNameFallback(t *testing.T) {
	cfg, err := metaToConfig(Meta{
		Key:  "feishu",
		Name: "feishu",
		Meta: `{"token":"abc"}`,
	})
	if err != nil {
		t.Fatalf("metaToConfig failed: %v", err)
	}
	if cfg.Key != "feishu" {
		t.Fatalf("cfg.Key = %q, want feishu", cfg.Key)
	}
	if cfg.Name != "feishu" {
		t.Fatalf("cfg.Name = %q, want stored runtime name without fallback", cfg.Name)
	}
}

func TestListPluginMetaMergesConfiguredMeta(t *testing.T) {
	pluginDir := t.TempDir()
	writePluginScriptWithScope(t, filepath.Join(pluginDir, "feishu"), `{"key":"feishu","name":"飞书"}`, `[{"appId":"","appSecret":""}]`, `["reuse","agent","swarm"]`)
	writePluginScript(t, filepath.Join(pluginDir, "mail"), `"邮件"`, testParamJSON("token"))

	restore := overridePluginDir(pluginDir)
	defer restore()

	agentDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(agentDir, "A"), 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(t.TempDir(), "data")
	seedPluginMetaServiceData(t, dbPath, agentDir, MetaInput{
		Key:           "feishu",
		Meta:          `{"appId":"cli-app","appSecret":"cli-secret"}`,
		Stream:        true,
		Callback:      "./feishu",
		AgentID:       "A",
		ChatID:        "chat-001",
		Model:         "OpenAI",
		Thinking:      true,
		RouterDisable: true,
	})

	items, err := ListPluginMeta(map[string]string{
		"db":        dbPath,
		"agent-dir": agentDir,
	})
	if err != nil {
		t.Fatalf("ListPluginMeta failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d: %+v", len(items), items)
	}
	if items[0].Key != "feishu" || items[1].Key != "mail" {
		t.Fatalf("unexpected plugin keys: %+v", items)
	}
	if items[0].Name != "飞书" {
		t.Fatalf("first plugin = %+v", items[0])
	}
	if strings.Join(items[0].Scope, ",") != "reuse,agent,swarm" {
		t.Fatalf("unexpected scope: %+v", items[0].Scope)
	}
	if got := fmt.Sprint(items[0].Meta["appId"]); got != "cli-app" {
		t.Fatalf("unexpected merged meta: %+v", items[0].Meta)
	}
	if !items[0].RouterDisable {
		t.Fatalf("swarm = %v, want true", items[0].RouterDisable)
	}
	if strings.Join(items[1].Scope, ",") != "reuse,agent,provider,thinking,swarm" {
		t.Fatalf("unexpected default scope: %+v", items[1].Scope)
	}
	if items[1].Name != "邮件" || items[1].Meta == nil || len(items[1].Meta) != 0 {
		t.Fatalf("unexpected empty meta plugin: %+v", items[1])
	}
	if !items[1].RouterDisable {
		t.Fatalf("default router_disable = %v, want true", items[1].RouterDisable)
	}
}

func TestListPluginsReadsScopeWhenPluginExposesIt(t *testing.T) {
	pluginDir := t.TempDir()
	writePluginScriptWithScope(t, filepath.Join(pluginDir, "a"), `"IM"`, testParamJSON("key"), `["reuse","provider"]`)

	restore := overridePluginDir(pluginDir)
	defer restore()

	items, err := ListPlugins(map[string]string{"connect-cache": "0"})
	if err != nil {
		t.Fatalf("ListPlugins failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 plugin, got %+v", items)
	}
	if strings.Join(items[0].Scope, ",") != "reuse,provider" {
		t.Fatalf("scope = %+v, want [reuse provider]", items[0].Scope)
	}
}

func TestListPluginsSkipsMissingScopeCommand(t *testing.T) {
	pluginDir := t.TempDir()
	path := filepath.Join(pluginDir, "a")
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"name\" ]; then\n" +
		"  printf '%s\\n' '\"IM\"'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"param\" ]; then\n" +
		"  printf '%s\\n' '[{\"key\":\"\"}]'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"command\" ]; then\n" +
		"  printf '%s\\n' '[\"command\",\"help\",\"name\",\"param\",\"start\",\"stop\"]'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"help\" ]; then\n" +
		"  printf '%s\\n' 'help'\n" +
		"  exit 0\n" +
		"fi\n" +
		"printf 'unknown command: %s\\n' \"$1\" >&2\n" +
		"exit 1\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}

	restore := overridePluginDir(pluginDir)
	defer restore()

	items, err := ListPlugins(map[string]string{"connect-cache": "0"})
	if err != nil {
		t.Fatalf("ListPlugins failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 plugin, got %+v", items)
	}
	if items[0].Name != "IM" {
		t.Fatalf("unexpected plugin: %+v", items[0])
	}
	if items[0].Scope != nil {
		t.Fatalf("scope = %+v, want nil when scope command is missing", items[0].Scope)
	}
	raw, err := json.Marshal(items[0])
	if err != nil {
		t.Fatalf("marshal plugin: %v", err)
	}
	if strings.Contains(string(raw), "\"scope\"") {
		t.Fatalf("scope should be omitted from json when command is missing: %s", string(raw))
	}
}

func TestListPluginsKeepsEmptyScopeInJSON(t *testing.T) {
	pluginDir := t.TempDir()
	writePluginScriptWithScope(t, filepath.Join(pluginDir, "a"), `"IM"`, testParamJSON("key"), `[]`)

	restore := overridePluginDir(pluginDir)
	defer restore()

	items, err := ListPlugins(map[string]string{"connect-cache": "0"})
	if err != nil {
		t.Fatalf("ListPlugins failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 plugin, got %+v", items)
	}
	if items[0].Scope == nil {
		t.Fatalf("scope = nil, want empty slice")
	}
	if len(items[0].Scope) != 0 {
		t.Fatalf("scope = %+v, want []", items[0].Scope)
	}
	raw, err := json.Marshal(items[0])
	if err != nil {
		t.Fatalf("marshal plugin: %v", err)
	}
	if !strings.Contains(string(raw), `"scope":[]`) {
		t.Fatalf("scope should be preserved as empty array: %s", string(raw))
	}
}

func TestListPluginsReturnsErrorWhenScopeOutputIsInvalid(t *testing.T) {
	pluginDir := t.TempDir()
	writePluginScriptWithScope(t, filepath.Join(pluginDir, "a"), `"IM"`, testParamJSON("key"), `Usage:
  a command
  a param`)

	restore := overridePluginDir(pluginDir)
	defer restore()

	_, err := ListPlugins(map[string]string{"connect-cache": "0"})
	if err == nil || !strings.Contains(err.Error(), "invalid scope output") {
		t.Fatalf("ListPlugins err = %v, want invalid scope output", err)
	}
}

func TestUpsertPluginConfigCreatesWithResolvedCallbackAndDefaultsFeishuChatID(t *testing.T) {
	pluginDir := t.TempDir()
	writePluginScript(t, filepath.Join(pluginDir, "feishu"), `{"key":"feishu","name":"飞书"}`, `[{"appId":"","appSecret":""}]`)

	restore := overridePluginDir(pluginDir)
	defer restore()

	agentDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(agentDir, "A"), 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(t.TempDir(), "data")
	seedTokenStore(t, dbPath, agentDir, "OpenAI")

	item, err := UpsertPluginConfig(map[string]string{
		"db":        dbPath,
		"agent-dir": agentDir,
	}, MetaInput{
		Key:     "feishu",
		Meta:    `{"appId":"cli-app","appSecret":"cli-secret"}`,
		AgentID: "A",
		Model:   "OpenAI",
	})
	if err != nil {
		t.Fatalf("UpsertPluginConfig failed: %v", err)
	}

	wantCallback, err := filepath.Abs(filepath.Join(pluginDir, "feishu"))
	if err != nil {
		t.Fatalf("abs callback: %v", err)
	}
	if item.Callback != wantCallback {
		t.Fatalf("callback = %q, want %q", item.Callback, wantCallback)
	}
	if item.ChatID != "feishu" {
		t.Fatalf("chatId = %q, want feishu", item.ChatID)
	}
	if item.Stream {
		t.Fatalf("stream = %v, want false", item.Stream)
	}
	if item.Thinking {
		t.Fatalf("thinking = %v, want false", item.Thinking)
	}
	if item.RouterDisable {
		t.Fatalf("swarm = %v, want false", item.RouterDisable)
	}
}

func TestUpsertPluginConfigUpdatesExistingFeishuMetaWithDefaultChatIDWhenInputIsEmpty(t *testing.T) {
	pluginDir := t.TempDir()
	writePluginScript(t, filepath.Join(pluginDir, "feishu"), `{"key":"feishu","name":"飞书"}`, `[{"appId":"","appSecret":""}]`)

	restore := overridePluginDir(pluginDir)
	defer restore()

	agentDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(agentDir, "A"), 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(t.TempDir(), "data")
	seedPluginMetaServiceData(t, dbPath, agentDir, MetaInput{
		Key:           "feishu",
		Meta:          `{"appId":"old","appSecret":"old"}`,
		Stream:        false,
		Callback:      "./legacy-feishu",
		AgentID:       "A",
		ChatID:        "",
		Model:         "OpenAI",
		Thinking:      false,
		RouterDisable: false,
	})

	item, err := UpsertPluginConfig(map[string]string{
		"db":        dbPath,
		"agent-dir": agentDir,
	}, MetaInput{
		Key:     "feishu",
		Meta:    `{"appId":"new","appSecret":"new"}`,
		AgentID: "A",
		Model:   "OpenAI",
	})
	if err != nil {
		t.Fatalf("UpsertPluginConfig update failed: %v", err)
	}
	if item.ChatID != "feishu" {
		t.Fatalf("chatId = %q, want feishu", item.ChatID)
	}
}

func TestUpsertPluginConfigUpdatesExistingMeta(t *testing.T) {
	pluginDir := t.TempDir()
	writePluginScript(t, filepath.Join(pluginDir, "feishu"), `{"key":"feishu","name":"飞书"}`, `[{"appId":"","appSecret":""}]`)

	restore := overridePluginDir(pluginDir)
	defer restore()

	agentDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(agentDir, "A"), 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(t.TempDir(), "data")
	seedPluginMetaServiceData(t, dbPath, agentDir, MetaInput{
		Key:           "feishu",
		Meta:          `{"appId":"old","appSecret":"old"}`,
		Stream:        false,
		Callback:      "./legacy-feishu",
		AgentID:       "A",
		ChatID:        "chat-old",
		Model:         "OpenAI",
		Thinking:      false,
		RouterDisable: false,
	})

	item, err := UpsertPluginConfig(map[string]string{
		"db":        dbPath,
		"agent-dir": agentDir,
	}, MetaInput{
		Key:           "feishu",
		Meta:          `{"appId":"new","appSecret":"new"}`,
		Stream:        true,
		AgentID:       "A",
		ChatID:        "chat-fixed",
		Model:         "OpenAI",
		Thinking:      true,
		RouterDisable: true,
	})
	if err != nil {
		t.Fatalf("UpsertPluginConfig update failed: %v", err)
	}

	if item.Meta != `{"appId":"new","appSecret":"new"}` {
		t.Fatalf("meta = %q", item.Meta)
	}
	if !item.Stream {
		t.Fatalf("stream = %v, want true", item.Stream)
	}
	if item.ChatID != "chat-fixed" {
		t.Fatalf("chatId = %q, want chat-fixed", item.ChatID)
	}
	if !item.Thinking {
		t.Fatalf("thinking = %v, want true", item.Thinking)
	}
	if !item.RouterDisable {
		t.Fatalf("swarm = %v, want true", item.RouterDisable)
	}
	wantCallback, err := filepath.Abs(filepath.Join(pluginDir, "feishu"))
	if err != nil {
		t.Fatalf("abs callback: %v", err)
	}
	if item.Callback != wantCallback {
		t.Fatalf("callback = %q, want %q", item.Callback, wantCallback)
	}
}

func TestUpsertPluginConfigAllowsEmptyAgentAndModelWhenScopeIsEmpty(t *testing.T) {
	pluginDir := t.TempDir()
	writePluginScriptWithScope(t, filepath.Join(pluginDir, "browser"), `{"key":"browser","name":"浏览器"}`, `[]`, `[]`)

	restore := overridePluginDir(pluginDir)
	defer restore()

	agentDir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "data")

	item, err := UpsertPluginConfig(map[string]string{
		"db":        dbPath,
		"agent-dir": agentDir,
	}, MetaInput{
		Key:  "browser",
		Meta: `{}`,
	})
	if err != nil {
		t.Fatalf("UpsertPluginConfig failed: %v", err)
	}
	if item.AgentID != "" {
		t.Fatalf("agentId = %q, want empty", item.AgentID)
	}
	if item.Model != "" {
		t.Fatalf("model = %q, want empty", item.Model)
	}
	if item.ChatID != "" {
		t.Fatalf("chatId = %q, want empty", item.ChatID)
	}
}

func TestAddRequestSkipsAgentAndModelValidationWhenPluginScopeIsEmpty(t *testing.T) {
	pluginDir := t.TempDir()
	writePluginScriptWithScope(t, filepath.Join(pluginDir, "browser"), `{"key":"browser","name":"浏览器"}`, `[]`, `[]`)

	restore := overridePluginDir(pluginDir)
	defer restore()

	agentDir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "data")

	if _, err := UpsertPluginConfig(map[string]string{
		"db":        dbPath,
		"agent-dir": agentDir,
	}, MetaInput{
		Key:  "browser",
		Meta: `{}`,
	}); err != nil {
		t.Fatalf("UpsertPluginConfig failed: %v", err)
	}

	svc, err := NewService(Options{
		DBPath:   dbPath,
		AgentDir: agentDir,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	defer svc.Close()

	request, err := svc.AddRequest(RequestInput{
		Key:     "browser",
		Content: "ping",
	})
	if err != nil {
		t.Fatalf("AddRequest failed: %v", err)
	}
	if request == nil || request.ID <= 0 {
		t.Fatalf("unexpected request: %+v", request)
	}
}

func TestUpsertPluginConfigByKeyMergesBackIntoPluginMeta(t *testing.T) {
	pluginDir := t.TempDir()
	writePluginScript(t, filepath.Join(pluginDir, "email"), `{"key":"email","name":"邮件"}`, `[{"email":"","email_pop3":"","email_smtp":"","email_password":"","email_whitelist":""}]`)

	restore := overridePluginDir(pluginDir)
	defer restore()

	agentDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(agentDir, "A"), 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(t.TempDir(), "data")
	seedTokenStore(t, dbPath, agentDir, "OpenAI")

	if _, err := UpsertPluginConfig(map[string]string{
		"db":        dbPath,
		"agent-dir": agentDir,
	}, MetaInput{
		Key:     "email",
		Meta:    `{"email":"demo@example.com","email_pop3":"pop.example.com:995","email_smtp":"smtp.example.com:465","email_password":"secret","email_whitelist":"sender@example.com"}`,
		AgentID: "A",
		Model:   "OpenAI",
	}); err != nil {
		t.Fatalf("UpsertPluginConfig by key failed: %v", err)
	}

	items, err := ListPluginMeta(map[string]string{
		"db":        dbPath,
		"agent-dir": agentDir,
	})
	if err != nil {
		t.Fatalf("ListPluginMeta failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d: %+v", len(items), items)
	}
	if items[0].Key != "email" || items[0].Name != "邮件" {
		t.Fatalf("unexpected plugin identity: %+v", items[0])
	}
	if got := fmt.Sprint(items[0].Meta["email"]); got != "demo@example.com" {
		t.Fatalf("unexpected email meta: %+v", items[0].Meta)
	}
	if got := fmt.Sprint(items[0].Meta["email_pop3"]); got != "pop.example.com:995" {
		t.Fatalf("unexpected email_pop3 meta: %+v", items[0].Meta)
	}
	if got := fmt.Sprint(items[0].Meta["email_smtp"]); got != "smtp.example.com:465" {
		t.Fatalf("unexpected email_smtp meta: %+v", items[0].Meta)
	}
	if got := fmt.Sprint(items[0].Meta["email_password"]); got != "secret" {
		t.Fatalf("unexpected email_password meta: %+v", items[0].Meta)
	}
	if got := fmt.Sprint(items[0].Meta["email_whitelist"]); got != "sender@example.com" {
		t.Fatalf("unexpected email_whitelist meta: %+v", items[0].Meta)
	}
}

func TestUpsertPluginConfigRequiresPluginKey(t *testing.T) {
	pluginDir := t.TempDir()
	writePluginScript(t, filepath.Join(pluginDir, "feishu"), `{"key":"feishu","name":"飞书"}`, `[{"appId":"","appSecret":""}]`)

	restore := overridePluginDir(pluginDir)
	defer restore()

	agentDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(agentDir, "A"), 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(t.TempDir(), "data")
	seedTokenStore(t, dbPath, agentDir, "OpenAI")

	_, err := UpsertPluginConfig(map[string]string{
		"db":        dbPath,
		"agent-dir": agentDir,
	}, MetaInput{
		Key:     "missing",
		Meta:    `{"appId":"x"}`,
		AgentID: "A",
		Model:   "OpenAI",
	})
	if err == nil || !strings.Contains(err.Error(), "plugin not found") {
		t.Fatalf("err = %v, want plugin not found", err)
	}
}

func TestResolvePluginBinaryRejectsDisplayNameAndSupportsKey(t *testing.T) {
	pluginDir := t.TempDir()
	pluginPath := filepath.Join(pluginDir, "feishu")
	writePluginScript(t, pluginPath, `{"key":"feishu","name":"飞书"}`, `[{"appId":"","appSecret":""}]`)

	restore := overridePluginDir(pluginDir)
	defer restore()

	if _, err := ResolvePluginBinary("飞书"); err == nil {
		t.Fatal("ResolvePluginBinary unexpectedly accepted display name")
	}
	byKey, err := ResolvePluginBinary("feishu")
	if err != nil {
		t.Fatalf("ResolvePluginBinary by key failed: %v", err)
	}
	wantPath, err := filepath.Abs(pluginPath)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	if byKey.Path != wantPath {
		t.Fatalf("resolved path = %q, want %q", byKey.Path, wantPath)
	}
	if byKey.Name != "飞书" {
		t.Fatalf("resolved name = %q", byKey.Name)
	}
}

func TestResolvePluginBinaryByKeySkipsRuntimeArtifacts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	pluginDir := t.TempDir()
	pluginPath := filepath.Join(pluginDir, "remote")
	writePluginScript(t, pluginPath, `{"key":"remote","name":"远程"}`, `[]`)
	if err := os.WriteFile(filepath.Join(pluginDir, "browser.log"), []byte("not a plugin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "User_Data.zip"), []byte("not a plugin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "data"), []byte("SQLite format 3\x00"), 0o755); err != nil {
		t.Fatal(err)
	}

	restore := overridePluginDir(pluginDir)
	defer restore()

	binary, err := ResolvePluginBinaryByKey("remote")
	if err != nil {
		t.Fatalf("ResolvePluginBinaryByKey failed: %v", err)
	}
	wantPath, err := filepath.Abs(pluginPath)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	if binary.Path != wantPath {
		t.Fatalf("binary.Path = %q, want %q", binary.Path, wantPath)
	}
	if binary.Key != "remote" || binary.Name != "远程" {
		t.Fatalf("unexpected binary metadata: %+v", binary)
	}
}

func TestResolveDirectPluginBinaryByExecutableNameSkipsInspect(t *testing.T) {
	pluginDir := t.TempDir()
	pluginPath := filepath.Join(pluginDir, "feishu")
	if err := os.WriteFile(pluginPath, []byte("#!/bin/sh\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writePluginCache(pluginDir, []PluginInfo{{
		Key:   "feishu",
		Name:  "飞书",
		Param: testParamFields("appId", "appSecret"),
	}}); err != nil {
		t.Fatalf("write cache: %v", err)
	}

	restore := overridePluginDir(pluginDir)
	defer restore()

	binary, err := resolveDirectPluginBinary("feishu")
	if err != nil {
		t.Fatalf("resolveDirectPluginBinary by executable name failed: %v", err)
	}
	wantPath, err := filepath.Abs(pluginPath)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	if binary.Path != wantPath {
		t.Fatalf("binary.Path = %q, want %q", binary.Path, wantPath)
	}
	if binary.Key != "feishu" || binary.Name != "飞书" {
		t.Fatalf("unexpected binary metadata: %+v", binary)
	}
}

func TestRunPluginActionSupportsSubcommandStyle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	pluginDir := t.TempDir()
	writePluginActionScript(t, filepath.Join(pluginDir, "feishu"), `{"key":"feishu","name":"飞书"}`, `[{"appId":"","appSecret":""}]`, "subcommand")

	restore := overridePluginDir(pluginDir)
	defer restore()

	result, err := RunPluginAction("feishu", "start", map[string]string{
		"connect-bin": "./connect",
		"foreground":  "true",
	})
	if err != nil {
		t.Fatalf("RunPluginAction failed: %v", err)
	}
	if len(result.Command) == 0 || result.Command[0] != "start" {
		t.Fatalf("command = %v, want subcommand start", result.Command)
	}
	if got := strings.TrimSpace(string(result.Output)); got != `{"status":"started","mode":"subcommand"}` {
		t.Fatalf("output = %q", got)
	}
}

func TestRunPluginActionRejectsFlagStyleLifecycleOnlyPlugin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	pluginDir := t.TempDir()
	writePluginActionScript(t, filepath.Join(pluginDir, "feishu"), `{"key":"feishu","name":"飞书"}`, `[{"appId":"","appSecret":""}]`, "flag")

	restore := overridePluginDir(pluginDir)
	defer restore()

	result, err := RunPluginAction("feishu", "stop", map[string]string{
		"pid-file": "./feishu.pid",
	})
	if err == nil {
		t.Fatalf("RunPluginAction unexpectedly succeeded: %+v", result)
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("err = %v, want unknown command", err)
	}
}

func TestRunPluginActionUsesPluginDirectoryAsWorkingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	pluginDir := t.TempDir()
	pluginPath := filepath.Join(pluginDir, "feishu")
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"name\" ]; then\n" +
		"  printf '%s\\n' '{\"key\":\"feishu\",\"name\":\"飞书\"}'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"param\" ]; then\n" +
		"  printf '%s\\n' '[\"appId\",\"appSecret\"]'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"start\" ]; then\n" +
		"  printf '%s\\n' started > relative.log\n" +
		"  printf '%s\\n' '{\"status\":\"started\",\"mode\":\"cwd\"}'\n" +
		"  exit 0\n" +
		"fi\n" +
		"printf 'unknown command: %s\\n' \"$1\" >&2\n" +
		"exit 1\n"
	if err := os.WriteFile(pluginPath, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}

	restore := overridePluginDir(pluginDir)
	defer restore()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	outsideLog := filepath.Join(cwd, "relative.log")
	_ = os.Remove(outsideLog)
	t.Cleanup(func() { _ = os.Remove(outsideLog) })

	result, err := RunPluginAction("feishu", "start", nil)
	if err != nil {
		t.Fatalf("RunPluginAction failed: %v", err)
	}
	if len(result.Command) == 0 || result.Command[0] != "start" {
		t.Fatalf("command = %v, want start", result.Command)
	}

	pluginLog := filepath.Join(pluginDir, "relative.log")
	data, err := os.ReadFile(pluginLog)
	if err != nil {
		t.Fatalf("read plugin relative log: %v", err)
	}
	if strings.TrimSpace(string(data)) != "started" {
		t.Fatalf("plugin relative log = %q", string(data))
	}
	if _, err := os.Stat(outsideLog); !os.IsNotExist(err) {
		t.Fatalf("unexpected relative log outside plugin dir: %v", err)
	}
}

func TestRunPluginActionDoesNotFallbackOnRealStartFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	pluginDir := t.TempDir()
	pluginPath := filepath.Join(pluginDir, "email")
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"name\" ]; then\n" +
		"  printf '%s\\n' '{\"key\":\"email\",\"name\":\"邮件\"}'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"param\" ]; then\n" +
		"  printf '%s\\n' '[\"email\"]'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"start\" ]; then\n" +
		"  printf '%s\\n' 'start timeout waiting for pid file' >&2\n" +
		"  exit 1\n" +
		"fi\n" +
		"printf 'unknown command: %s\\n' \"$1\" >&2\n" +
		"exit 1\n"
	if err := os.WriteFile(pluginPath, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}

	restore := overridePluginDir(pluginDir)
	defer restore()

	_, err := RunPluginAction("email", "start", nil)
	if err == nil || !strings.Contains(err.Error(), "start timeout waiting for pid file") {
		t.Fatalf("err = %v, want real start failure", err)
	}
	if _, statErr := os.Stat(filepath.Join(pluginDir, "fallback.log")); !os.IsNotExist(statErr) {
		t.Fatalf("fallback unexpectedly executed: %v", statErr)
	}
}

func TestRunPluginActionDoesNotFallbackOnUnknownCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	pluginDir := t.TempDir()
	writePluginActionScript(t, filepath.Join(pluginDir, "feishu"), `{"key":"feishu","name":"飞书"}`, `[{"appId":"","appSecret":""}]`, "flag")

	restore := overridePluginDir(pluginDir)
	defer restore()

	result, err := RunPluginAction("feishu", "start", nil)
	if err == nil {
		t.Fatalf("RunPluginAction unexpectedly succeeded: %+v", result)
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("err = %v, want unknown command", err)
	}
}

func TestPluginStatusByKeyRunningAndStopped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	pluginDir := t.TempDir()
	pluginPath := filepath.Join(pluginDir, "feishu")
	writePluginScript(t, pluginPath, `{"key":"feishu","name":"飞书"}`, `[{"appId":"","appSecret":""}]`)

	restore := overridePluginDir(pluginDir)
	defer restore()

	pidFile := filepath.Join(pluginDir, "feishu.pid")

	result, err := PluginStatusByKey("feishu", nil)
	if err != nil {
		t.Fatalf("PluginStatusByKey failed: %v", err)
	}
	if result.Started {
		t.Fatalf("started = true, want false")
	}
	if result.PID != 0 {
		t.Fatalf("pid = %d, want 0", result.PID)
	}
	if result.PIDFile != pidFile {
		t.Fatalf("pidFile = %q, want %q", result.PIDFile, pidFile)
	}

	cmd := exec.Command("sh", "-c", "sleep 5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}

	runningResult, err := PluginStatusByKey("feishu", nil)
	if err != nil {
		t.Fatalf("PluginStatusByKey failed: %v", err)
	}
	if !runningResult.Started {
		t.Fatalf("started = false, want true")
	}
	if runningResult.PID != cmd.Process.Pid {
		t.Fatalf("pid = %d, want %d", runningResult.PID, cmd.Process.Pid)
	}
}

func TestGetPluginStatusUsesDefaultPidFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	pluginDir := t.TempDir()
	pluginPath := filepath.Join(pluginDir, "feishu")
	writePluginScript(t, pluginPath, `{"key":"feishu","name":"飞书"}`, `[{"appId":"","appSecret":""}]`)

	restore := overridePluginDir(pluginDir)
	defer restore()

	pidFile := filepath.Join(pluginDir, "feishu.pid")
	cmd := execShellCommand(t, fmt.Sprintf("echo $$ > %q; sleep 30", pidFile))
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
		_ = os.Remove(pidFile)
	})
	waitForFile(t, pidFile)

	status, err := GetPluginStatus("feishu", nil)
	if err != nil {
		t.Fatalf("GetPluginStatus failed: %v", err)
	}
	if !status.Started {
		t.Fatalf("status.Started = %v, want true", status.Started)
	}
	if status.PID <= 0 {
		t.Fatalf("status.PID = %d, want positive pid", status.PID)
	}
	if status.PIDFile != pidFile {
		t.Fatalf("status.PIDFile = %q, want %q", status.PIDFile, pidFile)
	}
	if status.Key != "feishu" || status.Name != "飞书" {
		t.Fatalf("unexpected status metadata: %+v", status)
	}
}

func TestGetPluginStatusSupportsCustomPidFileAndStoppedState(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	pluginDir := t.TempDir()
	writePluginScript(t, filepath.Join(pluginDir, "feishu"), `{"key":"feishu","name":"飞书"}`, `[{"appId":"","appSecret":""}]`)

	restore := overridePluginDir(pluginDir)
	defer restore()

	customPIDFile := filepath.Join(t.TempDir(), "custom.pid")
	status, err := GetPluginStatus("feishu", map[string]string{"pid-file": customPIDFile})
	if err != nil {
		t.Fatalf("GetPluginStatus failed: %v", err)
	}
	if status.Started {
		t.Fatalf("status.Started = %v, want false", status.Started)
	}
	if status.PID != 0 {
		t.Fatalf("status.PID = %d, want 0", status.PID)
	}
	if status.PIDFile != customPIDFile {
		t.Fatalf("status.PIDFile = %q, want %q", status.PIDFile, customPIDFile)
	}
}

func execShellCommand(t *testing.T, script string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command("sh", "-c", script)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start shell command: %v", err)
	}
	return cmd
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timeout waiting file: %s", path)
}

func TestDefaultPluginDirUsesRuntimeConfigAppDirPlugins(t *testing.T) {
	root := t.TempDir()
	exePath := filepath.Join(root, "integration")
	configPath := filepath.Join(root, "config", "config.json")
	runtimeDir := filepath.Join(root, "runtime")
	if err := os.WriteFile(exePath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(fmt.Sprintf("{\"app-dir\":%q}\n", runtimeDir)), 0o644); err != nil {
		t.Fatal(err)
	}

	restoreExe := SetOSExecutableForTest(func() (string, error) { return exePath, nil })
	defer restoreExe()
	t.Setenv("DEEPRIGHT_PLUGIN_DIR", "")

	got, err := defaultPluginDir()
	if err != nil {
		t.Fatalf("defaultPluginDir failed: %v", err)
	}
	want := filepath.Join(runtimeDir, "plugins")
	if got != want {
		t.Fatalf("defaultPluginDir = %q, want %q", got, want)
	}
}

func TestDefaultPluginDirFailsWithoutRuntimePluginLocation(t *testing.T) {
	root := t.TempDir()
	exePath := filepath.Join(root, "integration")
	if err := os.WriteFile(exePath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}

	restoreExe := SetOSExecutableForTest(func() (string, error) { return exePath, nil })
	defer restoreExe()
	t.Setenv("DEEPRIGHT_PLUGIN_DIR", "")

	if got, err := defaultPluginDir(); err == nil {
		t.Fatalf("defaultPluginDir unexpectedly succeeded: %q", got)
	}
}

func TestSplitPluginExecCommandHandlesQuotesEscapesAndEmptyArgs(t *testing.T) {
	args, err := SplitPluginExecCommand(`instance init "chat room" '' escaped\ value --flag "a\"b"`)
	if err != nil {
		t.Fatalf("SplitPluginExecCommand failed: %v", err)
	}
	want := []string{"instance", "init", "chat room", "", "escaped value", "--flag", `a"b`}
	if len(args) != len(want) {
		t.Fatalf("args len = %d, want %d (%v)", len(args), len(want), args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("args[%d] = %q, want %q; full=%v", i, args[i], want[i], args)
		}
	}
}

func TestDefaultPluginDirPrefersEnvOnDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only plugin dir behavior")
	}

	want := t.TempDir()
	if err := os.Setenv("DEEPRIGHT_PLUGIN_DIR", want); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Unsetenv("DEEPRIGHT_PLUGIN_DIR") }()

	got, err := defaultPluginDir()
	if err != nil {
		t.Fatalf("defaultPluginDir failed: %v", err)
	}
	if got != want {
		t.Fatalf("defaultPluginDir = %q, want %q", got, want)
	}
}

func TestSplitPluginExecCommandRejectsBrokenQuoting(t *testing.T) {
	tests := []struct {
		name    string
		command string
		wantErr string
	}{
		{name: "dangling escape", command: `instance init foo\`, wantErr: "dangling escape"},
		{name: "unclosed quote", command: `instance init "foo`, wantErr: "unclosed quote"},
		{name: "empty", command: `   `, wantErr: "command is required"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := SplitPluginExecCommand(tc.command)
			if err == nil || err.Error() == "" || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestParsePluginExecRequestCollectsFlags(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/plugins/exec?key=browser&command=instance+init&agentId=a1&chatId=b1&connect-bin=%2Ftmp%2Fintegration", nil)
	key, command, flags, err := ParsePluginExecRequest(req)
	if err != nil {
		t.Fatalf("ParsePluginExecRequest failed: %v", err)
	}
	if key != "browser" || command != "instance init" {
		t.Fatalf("unexpected request parse result: key=%q command=%q", key, command)
	}
	if flags["agentId"] != "a1" || flags["chatId"] != "b1" || flags["connect-bin"] != "/tmp/integration" {
		t.Fatalf("unexpected flags: %+v", flags)
	}
}

func TestPluginActionStatusCodeClassifiesLookupFailures(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{err: errors.New("key is required"), want: http.StatusBadRequest},
		{err: errors.New("command is required"), want: http.StatusBadRequest},
		{err: errors.New("plugin not found: browser"), want: http.StatusBadRequest},
		{err: errors.New("plugin is not executable: /tmp/browser"), want: http.StatusBadRequest},
		{err: errors.New("run plugin browser start failed: exit status 1"), want: http.StatusInternalServerError},
	}

	for _, tc := range tests {
		if got := PluginActionStatusCode(tc.err); got != tc.want {
			t.Fatalf("PluginActionStatusCode(%q) = %d, want %d", tc.err, got, tc.want)
		}
	}
}

func TestRunPluginExecCommandPreservesSortedFlags(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	pluginDir := t.TempDir()
	writePluginExecScript(t, pluginDir+"/browser", `{"key":"browser","name":"浏览器"}`, `[{"agentId":"","chatId":""}]`)

	restore := overridePluginDir(pluginDir)
	defer restore()

	result, err := RunPluginExecCommand("browser", `instance init`, map[string]string{
		"chatId":      "chat-1",
		"agentId":     "agent-a",
		"connect-bin": "/tmp/connect",
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("RunPluginExecCommand failed: %v", err)
	}
	want := []string{"instance", "init", "--agentId", "agent-a", "--chatId", "chat-1", "--connect-bin", "/tmp/connect"}
	if len(result.Command) != len(want) {
		t.Fatalf("command len = %d, want %d (%v)", len(result.Command), len(want), result.Command)
	}
	for i := range want {
		if result.Command[i] != want[i] {
			t.Fatalf("command[%d] = %q, want %q; full=%v", i, result.Command[i], want[i], result.Command)
		}
	}
}

func writePluginScript(t *testing.T, path, nameOutput, paramOutput string) {
	t.Helper()
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"name\" ]; then\n" +
		"  printf '%s\\n' '" + escapeSingleQuotes(nameOutput) + "'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"param\" ]; then\n" +
		"  printf '%s\\n' '" + escapeSingleQuotes(paramOutput) + "'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"scope\" ]; then\n" +
		"  printf '%s\\n' '[\"reuse\",\"agent\",\"provider\",\"thinking\",\"swarm\"]'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"command\" ]; then\n" +
		"  printf '%s\\n' '[\"command\",\"help\",\"name\",\"param\",\"start\",\"stop\"]'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"help\" ]; then\n" +
		"  printf '%s\\n' 'help'\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 1\n"
	if runtime.GOOS == "windows" {
		t.Fatal("plugin script test is not supported on windows")
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func writePluginExecScript(t *testing.T, path, nameOutput, paramOutput string) {
	t.Helper()
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"name\" ]; then\n" +
		"  printf '%s\\n' '" + escapeSingleQuotes(nameOutput) + "'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"param\" ]; then\n" +
		"  printf '%s\\n' '" + escapeSingleQuotes(paramOutput) + "'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"scope\" ]; then\n" +
		"  printf '%s\\n' '[\"agent\",\"provider\"]'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"command\" ]; then\n" +
		"  printf '%s\\n' '[\"command\",\"help\",\"name\",\"param\",\"start\",\"stop\",\"instance\"]'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"help\" ]; then\n" +
		"  printf '%s\\n' 'help'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"instance\" ] && [ \"$2\" = \"init\" ]; then\n" +
		"  printf '%s\\n' '{\"ok\":true}'\n" +
		"  exit 0\n" +
		"fi\n" +
		"printf 'unknown command: %s\\n' \"$1\" >&2\n" +
		"exit 1\n"
	if runtime.GOOS == "windows" {
		t.Fatal("plugin script test is not supported on windows")
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func writePluginScriptWithScope(t *testing.T, path, nameOutput, paramOutput, scopeOutput string) {
	t.Helper()
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"name\" ]; then\n" +
		"  printf '%s\\n' '" + escapeSingleQuotes(nameOutput) + "'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"param\" ]; then\n" +
		"  printf '%s\\n' '" + escapeSingleQuotes(paramOutput) + "'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"scope\" ]; then\n" +
		"  printf '%s\\n' '" + escapeSingleQuotes(scopeOutput) + "'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"command\" ]; then\n" +
		"  printf '%s\\n' '[\"command\",\"help\",\"name\",\"param\",\"start\",\"stop\"]'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"help\" ]; then\n" +
		"  printf '%s\\n' 'help'\n" +
		"  exit 0\n" +
		"fi\n" +
		"printf 'unknown command: %s\\n' \"$1\" >&2\n" +
		"exit 1\n"
	if runtime.GOOS == "windows" {
		t.Fatal("plugin script test is not supported on windows")
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func writePluginActionScript(t *testing.T, path, nameOutput, paramOutput, mode string) {
	t.Helper()
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"name\" ]; then\n" +
		"  printf '%s\\n' '" + escapeSingleQuotes(nameOutput) + "'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"param\" ]; then\n" +
		"  printf '%s\\n' '" + escapeSingleQuotes(paramOutput) + "'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"command\" ]; then\n" +
		"  printf '%s\\n' '[\"command\",\"help\",\"name\",\"param\",\"start\",\"stop\"]'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"help\" ]; then\n" +
		"  printf '%s\\n' 'help'\n" +
		"  exit 0\n" +
		"fi\n"
	switch mode {
	case "subcommand":
		content += "if [ \"$1\" = \"start\" ]; then\n" +
			"  printf '%s\\n' '{\"status\":\"started\",\"mode\":\"subcommand\"}'\n" +
			"  exit 0\n" +
			"fi\n" +
			"if [ \"$1\" = \"stop\" ]; then\n" +
			"  printf '%s\\n' '{\"status\":\"stopped\",\"mode\":\"subcommand\"}'\n" +
			"  exit 0\n" +
			"fi\n"
	case "flag":
		content += "if [ \"$1\" = \"--start\" ]; then\n" +
			"  printf '%s\\n' '{\"status\":\"started\",\"mode\":\"flag\"}'\n" +
			"  exit 0\n" +
			"fi\n" +
			"if [ \"$1\" = \"--stop\" ]; then\n" +
			"  printf '%s\\n' '{\"status\":\"stopped\",\"mode\":\"flag\"}'\n" +
			"  exit 0\n" +
			"fi\n"
	default:
		t.Fatalf("unknown action script mode: %s", mode)
	}
	content += "printf 'unknown command: %s\\n' \"$1\" >&2\n" +
		"exit 1\n"
	if runtime.GOOS == "windows" {
		t.Fatal("plugin shell script test is not supported on windows")
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func escapeSingleQuotes(input string) string {
	return strings.ReplaceAll(input, "'", `'"'"'`)
}

func overridePluginDir(dir string) func() {
	originalDirResolver := pluginDirResolver
	pluginDirResolver = func() (string, error) { return dir, nil }
	return func() {
		pluginDirResolver = originalDirResolver
	}
}

func seedPluginMetaServiceData(t *testing.T, dbPath, agentDir string, input MetaInput) {
	t.Helper()
	db, err := os.OpenFile(dbPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	svc, err := NewService(Options{
		DBPath:   dbPath,
		AgentDir: agentDir,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	defer svc.Close()
	if _, err := svc.db.Exec(`INSERT INTO token_store (model, token, updated_at) VALUES (?,?,?) ON CONFLICT(model) DO UPDATE SET token=excluded.token, updated_at=excluded.updated_at`, input.Model, "Bearer test-token", time.Now().Format(time.RFC3339)); err != nil {
		t.Fatalf("seed token: %v", err)
	}
	if _, err := svc.CreateMeta(input); err != nil {
		t.Fatalf("create meta: %v", err)
	}
}

func seedTokenStore(t *testing.T, dbPath, agentDir, model string) {
	t.Helper()
	db, err := os.OpenFile(dbPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	svc, err := NewService(Options{
		DBPath:   dbPath,
		AgentDir: agentDir,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	defer svc.Close()
	if _, err := svc.db.Exec(`INSERT INTO token_store (model, token, updated_at) VALUES (?,?,?) ON CONFLICT(model) DO UPDATE SET token=excluded.token, updated_at=excluded.updated_at`, model, "Bearer test-token", time.Now().Format(time.RFC3339)); err != nil {
		t.Fatalf("seed token: %v", err)
	}
}
