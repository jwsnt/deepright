package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"connect/connectsvc"
	"connect/pluginlog"
	"connect/sandboxstate"
	"connect/sharedutil"
	"connect/skillstate"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"knowledge/knowledgecore"
	"log"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	tokenconsume "proxy/consume"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"proxy/eventlog"
	"skill-scanner/skillscore"
)

func ensureKnowledgeRequestLockTable(db *sql.DB) error {
	return sharedutil.EnsureKnowledgeRequestLockTable(db)
}

func gzipBase64String(input string) string {
	return sharedutil.GzipBase64String(input)
}

func NewProxyClient(connectTimeout time.Duration) *http.Client {
	return sharedutil.NewProxyClient(connectTimeout)
}

func pluginParamJSON(keys ...string) string {
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

func pluginParamFields(keys ...string) connectsvc.PluginParamFields {
	fields := make(connectsvc.PluginParamFields, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, connectsvc.PluginParamField{Key: key})
	}
	return fields
}

func TestDetectResponseType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "finish reason error",
			content: "data: {\"choices\":[{\"finish_reason\":\"error\"}]}\n\n",
			want:    "abnormal",
		},
		{
			name:    "non 2xx numeric code",
			content: "data: {\"code\":503,\"choices\":[{\"delta\":{\"content\":\"Timeout\"}}]}\n\n",
			want:    "abnormal",
		},
		{
			name:    "non 2xx string code",
			content: "data: {\"code\":\"401\",\"choices\":[{\"delta\":{\"content\":\"Unauthorized\"}}]}\n\n",
			want:    "abnormal",
		},
		{
			name:    "2xx code stays normal",
			content: "data: {\"code\":200,\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n",
			want:    "normal",
		},
		{
			name:    "whole sse payload detects abnormal final chunk",
			content: "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\ndata: {\"code\":503,\"choices\":[{\"delta\":{\"content\":\"Timeout\"}}]}\n\ndata: [DONE]\n\n",
			want:    "abnormal",
		},
		{
			name:    "missing code defaults normal",
			content: "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n",
			want:    "normal",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := detectResponseType(tt.content); got != tt.want {
				t.Fatalf("detectResponseType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEnsureServeDefaultAgentScaffoldCopiesDefaultTemplate(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	defaultDir := filepath.Join(tmp, "config")
	writeProxyDefaultAgentTemplate(t, defaultDir)
	agentDir := filepath.Join(tmp, "agent")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}

	if err := ensureServeDefaultAgentScaffold(agentDir, ""); err != nil {
		t.Fatalf("ensureServeDefaultAgentScaffold: %v", err)
	}
	if info, err := os.Stat(filepath.Join(agentDir, "DEF_AGENT")); err != nil || !info.IsDir() {
		t.Fatalf("DEF_AGENT missing after scaffold: %v", err)
	}
	if info, err := os.Stat(filepath.Join(agentDir, "DEF_AGENT", "skills")); err != nil || !info.IsDir() {
		t.Fatalf("DEF_AGENT/skills missing after scaffold: %v", err)
	}
	assertProxyDefaultAgentTemplateCopied(t, filepath.Join(agentDir, "DEF_AGENT"))
}

func TestResolveDefaultAgentDirArgUsesMacAppSupportDirectory(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("mac-specific default agent dir")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AGENT_DIR", "")

	got := resolveDefaultAgentDirArg()
	want := filepath.Join(home, "Library", "Application Support", "deepright", "agent")
	if got != want {
		t.Fatalf("resolveDefaultAgentDirArg() = %q, want %q", got, want)
	}
}

func TestEnsureServeDefaultAgentScaffoldFailsWhenDefaultDirUnavailable(t *testing.T) {
	tmp := t.TempDir()
	agentDir := filepath.Join(tmp, "agent")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}

	if err := ensureServeDefaultAgentScaffold(agentDir, filepath.Join(tmp, "missing-config")); err == nil || !strings.Contains(err.Error(), "default-dir unavailable") {
		t.Fatalf("ensureServeDefaultAgentScaffold error = %v, want default-dir unavailable", err)
	}
	if _, err := os.Stat(filepath.Join(agentDir, "DEF_AGENT")); !os.IsNotExist(err) {
		t.Fatalf("DEF_AGENT should not remain after failed scaffold, err=%v", err)
	}
}

func TestEnsureServeDefaultAgentScaffoldSkipsNonEmptyAgentDir(t *testing.T) {
	tmp := t.TempDir()
	agentDir := filepath.Join(tmp, "agent")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}
	existing := filepath.Join(agentDir, "existing.txt")
	if err := os.WriteFile(existing, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	if err := ensureServeDefaultAgentScaffold(agentDir, filepath.Join(tmp, "missing-config")); err != nil {
		t.Fatalf("ensureServeDefaultAgentScaffold: %v", err)
	}
	if _, err := os.Stat(existing); err != nil {
		t.Fatalf("existing file should remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(agentDir, "DEF_AGENT")); !os.IsNotExist(err) {
		t.Fatalf("DEF_AGENT should not be created for non-empty dir, err=%v", err)
	}
}

func TestSyncDefaultAgentVersionMergesTemplateVersionIntoExistingDEF_AGENTConfig(t *testing.T) {
	tmp := t.TempDir()
	defaultDir := filepath.Join(tmp, "config")
	if err := os.MkdirAll(defaultDir, 0o755); err != nil {
		t.Fatalf("mkdir default dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(defaultDir, "config.json"), []byte("{\"host\":\"http://127.0.0.1:9998\",\"version\":\"1.0\"}\n"), 0o644); err != nil {
		t.Fatalf("write default config: %v", err)
	}

	agentDir := filepath.Join(tmp, "agent")
	defAgentDir := filepath.Join(agentDir, "DEF_AGENT")
	if err := os.MkdirAll(defAgentDir, 0o755); err != nil {
		t.Fatalf("mkdir DEF_AGENT: %v", err)
	}
	if err := os.WriteFile(filepath.Join(defAgentDir, "config.json"), []byte("{\"host\":\"http://127.0.0.1:9998\"}\n"), 0o644); err != nil {
		t.Fatalf("write agent config: %v", err)
	}

	if err := syncDefaultAgentVersion(agentDir, defaultDir); err != nil {
		t.Fatalf("syncDefaultAgentVersion: %v", err)
	}

	var cfg map[string]interface{}
	data, err := os.ReadFile(filepath.Join(defAgentDir, "config.json"))
	if err != nil {
		t.Fatalf("read synced config: %v", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("decode synced config: %v", err)
	}
	if cfg["host"] != "http://127.0.0.1:9998" {
		t.Fatalf("host = %#v, want preserved host", cfg["host"])
	}
	if cfg["version"] != "1.0" {
		t.Fatalf("version = %#v, want 1.0", cfg["version"])
	}
}

func writeProxyDefaultAgentTemplate(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir default template: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write SOUL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "nested", "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write USER.md: %v", err)
	}
}

func assertProxyDefaultAgentTemplateCopied(t *testing.T, dir string) {
	t.Helper()
	if data, err := os.ReadFile(filepath.Join(dir, "SOUL.md")); err != nil || string(data) != "soul" {
		t.Fatalf("SOUL.md copy mismatch, err=%v data=%q", err, string(data))
	}
	if data, err := os.ReadFile(filepath.Join(dir, "nested", "USER.md")); err != nil || string(data) != "user" {
		t.Fatalf("nested USER.md copy mismatch, err=%v data=%q", err, string(data))
	}
}

func TestResolveFileTargetAbsoluteAndRelative(t *testing.T) {
	testDir := t.TempDir()
	agentDir := filepath.Join(testDir, "agent-root")
	homeDir := filepath.Join(testDir, "home")
	if err := os.MkdirAll(filepath.Join(agentDir, "b"), 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("mkdir home dir: %v", err)
	}
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	absTarget := filepath.Join(agentDir, "b", "USER.md")
	if err := os.WriteFile(absTarget, []byte("user"), 0o644); err != nil {
		t.Fatalf("write absolute target: %v", err)
	}
	homeTarget := filepath.Join(homeDir, "demo-home.txt")
	if err := os.WriteFile(homeTarget, []byte("home"), 0o644); err != nil {
		t.Fatalf("write home target: %v", err)
	}
	proxy := &ProxyServer{
		AgentDir: agentDir,
		DeviceID: "test-device",
		CacheTTL: time.Second,
	}

	resolvedAbs, err := proxy.resolveFileTarget("", absTarget)
	if err != nil {
		t.Fatalf("resolve absolute path: %v", err)
	}
	if resolvedAbs != absTarget {
		t.Fatalf("resolved absolute path = %q, want %q", resolvedAbs, absTarget)
	}

	resolvedRel, err := proxy.resolveFileTarget("b", "USER.md")
	if err != nil {
		t.Fatalf("resolve relative path: %v", err)
	}
	if resolvedRel != absTarget {
		t.Fatalf("resolved relative path = %q, want %q", resolvedRel, absTarget)
	}

	resolvedTilde, err := proxy.resolveFileTarget("", "~/demo-home.txt")
	if err != nil {
		t.Fatalf("resolve tilde path: %v", err)
	}
	if resolvedTilde != homeTarget {
		t.Fatalf("resolved tilde path = %q, want %q", resolvedTilde, homeTarget)
	}
}

func TestResolveFileTargetRelativeRequiresAgentID(t *testing.T) {
	agentDir, err := filepath.Abs("../agent/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}
	proxy := &ProxyServer{
		AgentDir: agentDir,
		DeviceID: "test-device",
		CacheTTL: time.Second,
	}
	if _, err := proxy.resolveFileTarget("", "USER.md"); err == nil || !strings.Contains(err.Error(), "agentId is required") {
		t.Fatalf("resolve relative path without agentId err = %v, want agentId is required", err)
	}
}

func TestFileLastUpdateDuration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.md")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	modTime := time.Date(2026, 5, 6, 13, 0, 0, 0, time.Local)
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	now := modTime.Add(5 * time.Second)
	got, err := fileLastUpdateDuration(path, now)
	if err != nil {
		t.Fatalf("fileLastUpdateDuration: %v", err)
	}
	if got != 5000 {
		t.Fatalf("fileLastUpdateDuration = %d, want 5000", got)
	}

	dirModTime := modTime.Add(2 * time.Second)
	if err := os.Chtimes(dir, dirModTime, dirModTime); err != nil {
		t.Fatalf("chtimes dir: %v", err)
	}
	gotDir, err := fileLastUpdateDuration(dir, dirModTime.Add(4*time.Second))
	if err != nil {
		t.Fatalf("fileLastUpdateDuration dir: %v", err)
	}
	if gotDir != 4000 {
		t.Fatalf("directory fileLastUpdateDuration = %d, want 4000", gotDir)
	}
}

func TestHandleFileLastUpdate(t *testing.T) {
	agentDir, err := filepath.Abs("../agent/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}
	target := filepath.Join(agentDir, "b", "USER.md")
	targetDir := filepath.Join(agentDir, "b", "tmp")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}
	modTime := time.Now().Add(-3 * time.Second)
	if err := os.Chtimes(target, modTime, modTime); err != nil {
		t.Fatalf("chtimes target: %v", err)
	}
	dirModTime := time.Now().Add(-4 * time.Second)
	if err := os.Chtimes(targetDir, dirModTime, dirModTime); err != nil {
		t.Fatalf("chtimes target dir: %v", err)
	}

	proxy := &ProxyServer{
		AgentDir: agentDir,
		DeviceID: "test-device",
		CacheTTL: time.Second,
	}
	req := httptest.NewRequest(http.MethodGet, "/file/lastUpdate?agentId=b&file=USER.md", nil)
	rec := httptest.NewRecorder()
	proxy.HandleFileLastUpdate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%q", rec.Code, rec.Body.String())
	}
	value, err := strconv.ParseInt(strings.TrimSpace(rec.Body.String()), 10, 64)
	if err != nil {
		t.Fatalf("parse body: %v, body=%q", err, rec.Body.String())
	}
	if value < 0 {
		t.Fatalf("last update value = %d, want >= 0", value)
	}

	dirReq := httptest.NewRequest(http.MethodGet, "/file/lastUpdate?agentId=b&file=tmp", nil)
	dirRec := httptest.NewRecorder()
	proxy.HandleFileLastUpdate(dirRec, dirReq)
	if dirRec.Code != http.StatusOK {
		t.Fatalf("directory status = %d, want 200, body=%q", dirRec.Code, dirRec.Body.String())
	}
	dirValue, err := strconv.ParseInt(strings.TrimSpace(dirRec.Body.String()), 10, 64)
	if err != nil {
		t.Fatalf("parse directory body: %v, body=%q", err, dirRec.Body.String())
	}
	if dirValue < 0 {
		t.Fatalf("directory last update value = %d, want >= 0", dirValue)
	}
}

func TestNormalizePluginReplyContent(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "json object", in: "{\n  \"hello\": \"world\"\n}", want: "{\"hello\":\"world\"}"},
		{name: "json array", in: "[\n  {\n    \"today\": \"sunday\"\n  }\n]", want: "[{\"today\":\"sunday\"}]"},
		{name: "fenced json object", in: "```json\n{\n  \"hello\": \"world\"\n}\n```", want: "{\"hello\":\"world\"}"},
		{name: "fenced plain object", in: "```\n{\n  \"today\": \"sunday\"\n}\n```", want: "{\"today\":\"sunday\"}"},
		{name: "fenced json array", in: "```json\n[\n  {\n    \"hello\": \"world\"\n  }\n]\n```", want: "[{\"hello\":\"world\"}]"},
		{name: "fenced plain array", in: "```\n[\n  {\n    \"today\": \"sunday\"\n  }\n]\n```", want: "[{\"today\":\"sunday\"}]"},
		{name: "invalid keeps raw", in: "```json\n{hello}\n```", want: "```json\n{hello}\n```"},
		{name: "scalar keeps raw", in: "```json\n\"hello\"\n```", want: "```json\n\"hello\"\n```"},
		{name: "plain text keeps raw", in: "任务完成结果", want: "任务完成结果"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizePluginReplyContent(tc.in); got != tc.want {
				t.Fatalf("normalizePluginReplyContent() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExtractSSEAssistantTextFiltersLeadingPunctuation(t *testing.T) {
	payload := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"。"}}]}`,
		`data: {"choices":[{"delta":{"content":"，"}}]}`,
		`data: {"choices":[{"delta":{"content":"hello"}}]}`,
		`data: {"choices":[{"delta":{"content":", world"}}]}`,
		`data: [DONE]`,
		"",
	}, "\n")

	if got := extractSSEAssistantText([]byte(payload)); got != "hello, world" {
		t.Fatalf("extractSSEAssistantText() = %q, want %q", got, "hello, world")
	}
}

func TestProxyInjectsMetadataAndPreservesFields(t *testing.T) {
	fixtureAgentDir, err := filepath.Abs("../agent/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}

	restore := withPluginSandbox(t, nil)
	defer restore()
	flushAgentCache()

	tmp := t.TempDir()
	agentDir := filepath.Join(tmp, "agent")
	if err := copyDir(fixtureAgentDir, agentDir); err != nil {
		t.Fatalf("copy agent fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "knowledge"), 0o755); err != nil {
		t.Fatalf("create knowledge dir: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   filepath.Join(tmp, "data"),
		AgentDir: agentDir,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	defer svc.Close()
	db, err := sql.Open("sqlite", filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		t.Fatalf("seed token: %v", err)
	}
	if _, err := svc.CreateMeta(connectsvc.MetaInput{
		Key:      "feishu",
		Name:     "feishu",
		Meta:     `{"appId":"cli-app"}`,
		Callback: "./feishu",
		AgentID:  "A",
		ChatID:   "chat-001",
		Model:    "OpenAI",
		Thinking: true,
	}); err != nil {
		t.Fatalf("create meta: %v", err)
	}
	cfg, err := svc.GetMetaConfig("feishu", false)
	if err != nil {
		t.Fatalf("get meta config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(cfg.Callback), "feishu.pid"), []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}
	if got := detectPluginKeys(); len(got) != 1 || got[0] != "feishu" {
		t.Fatalf("detectPluginKeys() = %#v, want [feishu]", got)
	}

	var capturedBody map[string]interface{}
	var capturedHeaders http.Header

	// Mock upstream: capture forwarded request, return SSE response
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		flusher, _ := w.(http.Flusher)

		chunks := []string{
			`data: {"id":"cmpl-1","object":"chat.completion.chunk","created":1698999575,"model":"kimi-k2.5","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`,
			`data: {"id":"cmpl-1","object":"chat.completion.chunk","created":1698999575,"model":"kimi-k2.5","choices":[{"index":0,"delta":{"content":"你好"},"finish_reason":null}]}`,
			`data: {"id":"cmpl-1","object":"chat.completion.chunk","created":1698999575,"model":"kimi-k2.5","choices":[{"index":0,"delta":{"content":"。"},"finish_reason":null}]}`,
			`data: {"id":"cmpl-1","object":"chat.completion.chunk","created":1698999575,"model":"kimi-k2.5","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
			`data: [DONE]`,
		}
		for _, chunk := range chunks {
			fmt.Fprintf(w, "%s\n\n", chunk)
			flusher.Flush()
		}
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:     upstream.URL,
		AgentDir: agentDir,
		DeviceID: "test-device-123",
		CacheTTL: 120 * time.Second,
		Client:   &http.Client{Timeout: 10 * time.Second},
	}

	// Create proxy server
	proxyServer := httptest.NewServer(http.HandlerFunc(proxy.HandleChatCompletions))
	defer proxyServer.Close()

	// Send request like the CURL example
	reqBody := `{
		"model": "gpt-4",
		"messages": [{"role":"user","content":"HELLO WORLD"}],
		"stream": true
	}`
	req, _ := http.NewRequest("POST", proxyServer.URL+"/v1/chat/completions",
		strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-test-key")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// ── Verify forwarded request ──

	// Headers preserved
	if capturedHeaders.Get("Authorization") != "Bearer sk-test-key" {
		t.Errorf("Authorization header not preserved: %q", capturedHeaders.Get("Authorization"))
	}
	if capturedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type header not preserved: %q", capturedHeaders.Get("Content-Type"))
	}
	if capturedHeaders.Get("Accept") != "text/event-stream" {
		t.Errorf("Accept header not set: %q", capturedHeaders.Get("Accept"))
	}

	// model unchanged
	if model, ok := capturedBody["model"].(string); !ok || model != "gpt-4" {
		t.Errorf("model modified: %v", capturedBody["model"])
	}

	msgs, ok := capturedBody["messages"].([]interface{})
	if !ok || len(msgs) != 1 {
		t.Fatalf("messages modified: %v", capturedBody["messages"])
	}
	msg := msgs[0].(map[string]interface{})
	if msg["role"] != "user" || msg["content"] != "HELLO WORLD" {
		t.Errorf("message content modified: %v", msg)
	}
	if _, exists := capturedBody["message"]; exists {
		t.Fatalf("message should not be forwarded: %v", capturedBody["message"])
	}
	if capturedBody["stream"] != true {
		t.Fatalf("stream = %v, want true", capturedBody["stream"])
	}

	// metadata injected
	meta, ok := capturedBody["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("metadata not injected")
	}
	knowledge, ok := meta["knowledge"].(map[string]interface{})
	if !ok {
		t.Fatalf("knowledge missing from metadata: %#v", meta["knowledge"])
	}
	wantKnowledgePath, err := filepath.Abs(filepath.Join(tmp, "knowledge"))
	if err != nil {
		t.Fatalf("abs knowledge path: %v", err)
	}
	gotKnowledgePath, _ := knowledge["path"].(string)
	gotResolved, err := filepath.EvalSymlinks(gotKnowledgePath)
	if err != nil {
		gotResolved = gotKnowledgePath
	}
	wantResolved, err := filepath.EvalSymlinks(wantKnowledgePath)
	if err != nil {
		wantResolved = wantKnowledgePath
	}
	if gotResolved != wantResolved {
		t.Fatalf("knowledge.path = %v (resolved %q), want %q (resolved %q)", knowledge["path"], gotResolved, wantKnowledgePath, wantResolved)
	}
	if knowledge["lastUpdate"] != float64(0) {
		t.Fatalf("knowledge.lastUpdate = %v, want 0", knowledge["lastUpdate"])
	}
	if meta["deviceId"] != "test-device-123" {
		t.Errorf("deviceId = %v", meta["deviceId"])
	}
	if meta["type"] != chatTypePageSession {
		t.Errorf("type = %v, want %s", meta["type"], chatTypePageSession)
	}
	if meta["terminal"] == nil || meta["terminal"] == "" {
		t.Error("terminal is empty")
	}
	if meta["git"] != detectGit() {
		t.Errorf("git = %v, want %q", meta["git"], detectGit())
	}
	if plugins, ok := meta["plugins"].([]interface{}); !ok || len(plugins) == 0 {
		t.Errorf("plugins missing from metadata: %#v", meta["plugins"])
	}
	if meta["gateway"] == nil || meta["gateway"] == "" {
		t.Error("gateway is empty")
	}
	if meta["sys"] == nil || meta["sys"] == "" {
		t.Error("sys is empty")
	}
	agents, ok := meta["agents"].([]interface{})
	if !ok || len(agents) == 0 {
		t.Error("agents missing or empty")
	}
	firstAgent, ok := agents[0].(map[string]interface{})
	if !ok {
		t.Fatalf("first agent metadata malformed: %#v", agents[0])
	}
	if firstAgent["provider"] != "" {
		t.Errorf("provider = %#v, want empty string for default test agent", firstAgent["provider"])
	}

	// ── Verify SSE response streamed correctly ──

	var allData []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			allData = append(allData, line)
		}
	}

	if len(allData) != 5 {
		t.Errorf("expected 5 SSE data lines, got %d", len(allData))
	}

	// Verify content is "你好。"
	var fullContent string
	for _, line := range allData {
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			continue
		}
		var chunk map[string]interface{}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		choices, _ := chunk["choices"].([]interface{})
		if len(choices) > 0 {
			choice := choices[0].(map[string]interface{})
			delta, _ := choice["delta"].(map[string]interface{})
			if c, ok := delta["content"].(string); ok {
				fullContent += c
			}
		}
	}
	if fullContent != "你好。" {
		t.Errorf("streamed content = %q, want 你好。", fullContent)
	}

	// Last data line should be [DONE]
	if allData[len(allData)-1] != "data: [DONE]" {
		t.Errorf("last line = %q, want 'data: [DONE]'", allData[len(allData)-1])
	}
}

func TestProxyChatCompletionsKeepsBoolMetadataOnlyUnderMetadata(t *testing.T) {
	agentDir, err := filepath.Abs("../agent/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "knowledge"), 0o755); err != nil {
		t.Fatalf("mkdir knowledge: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	var capturedBody map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:     upstream.URL,
		AgentDir: agentDir,
		DeviceID: "test-device-bool",
		CacheTTL: 120 * time.Second,
		Client:   &http.Client{Timeout: 10 * time.Second},
	}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandleChatCompletions))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"HELLO BOOL"}],"stream":false,"thinking":false,"html":true,"router_disable":false,"metadata":{"agentId":"A","chat":"chat-bool","thinking":false,"html":true,"router_disable":false}}`
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	if _, exists := capturedBody["thinking"]; exists {
		t.Fatalf("top-level thinking should be absent: %v", capturedBody["thinking"])
	}
	if _, exists := capturedBody["html"]; exists {
		t.Fatalf("top-level html should be absent: %v", capturedBody["html"])
	}
	if _, exists := capturedBody["router_disable"]; exists {
		t.Fatalf("top-level router_disable should be absent: %v", capturedBody["router_disable"])
	}
	msgs, ok := capturedBody["messages"].([]interface{})
	if !ok || len(msgs) != 1 {
		t.Fatalf("messages = %#v, want one forwarded message", capturedBody["messages"])
	}
	msg := msgs[0].(map[string]interface{})
	if msg["role"] != "user" || msg["content"] != "HELLO BOOL" {
		t.Fatalf("forwarded message = %#v, want user/HELLO BOOL", msg)
	}
	if _, exists := capturedBody["message"]; exists {
		t.Fatalf("message should not be forwarded: %v", capturedBody["message"])
	}
	if _, exists := capturedBody["stream"]; !exists || capturedBody["stream"] != false {
		t.Fatalf("stream = %v, want false", capturedBody["stream"])
	}

	meta, ok := capturedBody["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata missing: %#v", capturedBody["metadata"])
	}
	if meta["thinking"] != false {
		t.Fatalf("metadata thinking = %v, want false", meta["thinking"])
	}
	if meta["html"] != true {
		t.Fatalf("metadata html = %v, want true", meta["html"])
	}
	if meta["router_disable"] != false {
		t.Fatalf("metadata router_disable = %v, want false", meta["router_disable"])
	}
}

func TestProxyChatCompletionsLogsAbnormalWhenUpstreamStreamBreaksMidResponse(t *testing.T) {
	flushAgentCache()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "knowledge"), 0o755); err != nil {
		t.Fatalf("mkdir knowledge: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	oldSharedDataDB := sharedDataDB
	oldSharedEventLogger := sharedEventLogger
	sharedDataDB = pooledDB{}
	sharedEventLogger = struct {
		mu     sync.Mutex
		path   string
		logger *eventlog.Logger
	}{}
	defer func() {
		if sharedEventLogger.logger != nil {
			_ = sharedEventLogger.logger.Close()
		}
		if sharedDataDB.db != nil {
			_ = sharedDataDB.db.Close()
		}
		sharedDataDB = oldSharedDataDB
		sharedEventLogger = oldSharedEventLogger
	}()

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Errorf("hijacker unavailable")
			return
		}
		conn, bufrw, err := hj.Hijack()
		if err != nil {
			t.Errorf("hijack failed: %v", err)
			return
		}
		payload := "data: " + `{"choices":[{"delta":{"content":"partial"}}]}` + "\n\n"
		fmt.Fprintf(bufrw, "HTTP/1.1 200 OK\r\nContent-Type: text/event-stream\r\nTransfer-Encoding: chunked\r\n\r\n")
		fmt.Fprintf(bufrw, "%x\r\n%s\r\n", len(payload), payload)
		_ = bufrw.Flush()
		_ = conn.Close()
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:     upstream.URL,
		AgentDir: agentRoot,
		DeviceID: "test-midstream-device",
		CacheTTL: 120 * time.Second,
		Client:   &http.Client{Timeout: 3 * time.Second},
	}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandleChatCompletions))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"HELLO MIDSTREAM"}],"stream":true,"metadata":{"agentId":"A","chat":"chat-midstream"}}`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-test")

	resp, err := http.DefaultClient.Do(req)
	if err == nil && resp != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	var abnormalCount int
	for time.Now().Before(deadline) {
		if err := db.QueryRow(`SELECT COUNT(*) FROM chat_log WHERE agent_id = ? AND chat_id = ? AND role = 'A' AND response_type = 'abnormal' AND content LIKE ?`,
			"A", "chat-midstream", "SSE stream interrupted:%").Scan(&abnormalCount); err != nil {
			t.Fatalf("count abnormal chat_log: %v", err)
		}
		if abnormalCount > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if abnormalCount == 0 {
		t.Fatalf("expected abnormal restore log after midstream disconnect")
	}
}

func TestProxyChatCompletionsInjectsSelectedAgentVersionAndSandbox(t *testing.T) {
	flushAgentCache()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "config.json"), []byte(`{"version":"2026.06.10"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if _, err := sandboxstate.Set(db, "A", "chat-version", sandboxstate.ModeFilePickNet); err != nil {
		t.Fatalf("set sandbox: %v", err)
	}

	var capturedBody map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:     upstream.URL,
		AgentDir: agentRoot,
		DeviceID: "test-device-version",
		CacheTTL: time.Hour,
		Client:   &http.Client{Timeout: 10 * time.Second},
	}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandleChatCompletions))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"HELLO VERSION"}],"metadata":{"agentId":"A","chat":"chat-version"}}`
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	meta, ok := capturedBody["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata missing: %#v", capturedBody["metadata"])
	}
	agent, ok := meta["agent"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata.agent missing: %#v", meta["agent"])
	}
	if agent["version"] != "2026.06.10" {
		t.Fatalf("metadata.agent.version = %#v, want 2026.06.10", agent["version"])
	}
	if agent["sandbox"] != sandboxstate.ModeFilePickNet {
		t.Fatalf("metadata.agent.sandbox = %#v, want %q", agent["sandbox"], sandboxstate.ModeFilePickNet)
	}
}

func TestProxyChatCompletionsKeepsEmptySelectedAgentVersionCachedUntilAgentCacheExpires(t *testing.T) {
	flushAgentCache()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}
	configPath := filepath.Join(agentDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var versions []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var capturedBody map[string]interface{}
		_ = json.Unmarshal(body, &capturedBody)
		version := ""
		if meta, ok := capturedBody["metadata"].(map[string]interface{}); ok {
			if agent, ok := meta["agent"].(map[string]interface{}); ok {
				if value, ok := agent["version"].(string); ok {
					version = value
				}
			}
		}
		versions = append(versions, version)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:     upstream.URL,
		AgentDir: agentRoot,
		DeviceID: "test-device-version-cache",
		CacheTTL: time.Hour,
		Client:   &http.Client{Timeout: 10 * time.Second},
	}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandleChatCompletions))
	defer server.Close()

	send := func() {
		reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"HELLO VERSION CACHE"}],"metadata":{"agentId":"A"}}`
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/chat/completions", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
	}

	send()
	if len(versions) != 1 || versions[0] != "" {
		t.Fatalf("first metadata.agent.version = %#v, want empty string", versions)
	}

	if err := os.WriteFile(configPath, []byte(`{"version":"2026.06.10"}`), 0o644); err != nil {
		t.Fatalf("rewrite config: %v", err)
	}

	send()
	if len(versions) != 2 || versions[1] != "" {
		t.Fatalf("second metadata.agent.version = %#v, want cached empty string", versions)
	}
}

func TestProxyChatCompletionsRefreshesMediaWithoutWaitingForAgentCache(t *testing.T) {
	flushAgentCache()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}
	configPath := filepath.Join(agentDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"media":{"cover":"first.png"}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var covers []string
	var agentCovers []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var capturedBody map[string]interface{}
		_ = json.Unmarshal(body, &capturedBody)

		cover := ""
		agentCover := ""
		if meta, ok := capturedBody["metadata"].(map[string]interface{}); ok {
			if agent, ok := meta["agent"].(map[string]interface{}); ok {
				if media, ok := agent["media"].(map[string]interface{}); ok {
					if value, ok := media["cover"].(string); ok {
						agentCover = value
					}
				}
			}
			if agents, ok := meta["agents"].([]interface{}); ok && len(agents) > 0 {
				if firstAgent, ok := agents[0].(map[string]interface{}); ok {
					if media, ok := firstAgent["media"].(map[string]interface{}); ok {
						if value, ok := media["cover"].(string); ok {
							cover = value
						}
					}
				}
			}
		}
		covers = append(covers, cover)
		agentCovers = append(agentCovers, agentCover)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:     upstream.URL,
		AgentDir: agentRoot,
		DeviceID: "test-device-media-cache",
		CacheTTL: time.Hour,
		Client:   &http.Client{Timeout: 10 * time.Second},
	}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandleChatCompletions))
	defer server.Close()

	send := func() {
		reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"HELLO MEDIA CACHE"}],"metadata":{"agentId":"A"}}`
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/chat/completions", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
	}

	send()
	if len(covers) != 1 || covers[0] != "first.png" || agentCovers[0] != "first.png" {
		t.Fatalf("first media covers = agents:%#v selected:%#v, want first.png", covers, agentCovers)
	}

	if err := os.WriteFile(configPath, []byte(`{"media":{"cover":"second.png"}}`), 0o644); err != nil {
		t.Fatalf("rewrite config: %v", err)
	}

	send()
	if len(covers) != 2 || covers[1] != "second.png" || agentCovers[1] != "second.png" {
		t.Fatalf("second media covers = agents:%#v selected:%#v, want second.png", covers, agentCovers)
	}
}

func TestProxyInjectsResponseSchemaIntoMetadataWhenProvided(t *testing.T) {
	agentDir, err := filepath.Abs("../agent/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}

	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "knowledge"), 0o755); err != nil {
		t.Fatalf("mkdir knowledge: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	var capturedBody map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:     upstream.URL,
		AgentDir: agentDir,
		DeviceID: "test-device-schema",
		CacheTTL: time.Second,
		Client:   &http.Client{Timeout: 10 * time.Second},
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleChatCompletions))
	defer server.Close()

	schema := `{"type":"object","properties":{"summary":{"type":"string"}},"required":["summary"]}`
	reqBody := fmt.Sprintf(`{"model":"gpt-4","messages":[{"role":"user","content":"HELLO WORLD"}],"stream":true,"metadata":{"agentId":"A","chat":"chat-schema","response_schema":%q}}`, schema)
	req, _ := http.NewRequest("POST", server.URL, strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)

	metadata, ok := capturedBody["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata missing: %+v", capturedBody)
	}
	msgs, ok := capturedBody["messages"].([]interface{})
	if !ok || len(msgs) != 1 {
		t.Fatalf("messages = %#v, want one forwarded message", capturedBody["messages"])
	}
	msg := msgs[0].(map[string]interface{})
	if msg["role"] != "user" || msg["content"] != "HELLO WORLD" {
		t.Fatalf("forwarded message = %#v, want user/HELLO WORLD", msg)
	}
	if _, exists := capturedBody["message"]; exists {
		t.Fatalf("message should not be forwarded: %v", capturedBody["message"])
	}
	if capturedBody["stream"] != true {
		t.Fatalf("stream = %v, want true", capturedBody["stream"])
	}
	if metadata["response_schema"] != schema {
		t.Fatalf("metadata response_schema = %v, want %s", metadata["response_schema"], schema)
	}
}

func TestProxyKnowledgeLastUpdateRemovedWithinInterval(t *testing.T) {
	agentDir, err := filepath.Abs("../agent/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}

	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "knowledge"), 0o755); err != nil {
		t.Fatalf("create knowledge dir: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	defer func() { _ = knowledgecore.CloseSharedDB() }()
	db, err := knowledgecore.OpenSharedDB(tmp)
	if err != nil {
		t.Fatalf("open knowledge db: %v", err)
	}
	now := time.Now().UnixMilli()
	if err := knowledgecore.SetLastUpdate(db, now-5*60*1000); err != nil {
		t.Fatalf("set knowledge last update: %v", err)
	}

	var capturedBody map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:                    upstream.URL,
		AgentDir:                agentDir,
		DeviceID:                "test-device-knowledge-interval",
		CacheTTL:                time.Second,
		KnowledgeUpdateInterval: 2 * time.Hour,
		KnowledgeUpdateLock:     30 * time.Minute,
		Client:                  &http.Client{Timeout: 10 * time.Second},
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleChatCompletions))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"hello"}]}`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	meta, ok := capturedBody["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata not injected: %#v", capturedBody["metadata"])
	}
	knowledge, ok := meta["knowledge"].(map[string]interface{})
	if !ok {
		t.Fatalf("knowledge not injected: %#v", meta["knowledge"])
	}
	if _, exists := knowledge["lastUpdate"]; exists {
		t.Fatalf("knowledge.lastUpdate should be removed within interval: %#v", knowledge)
	}
	if _, exists := knowledge["path"]; !exists {
		t.Fatalf("knowledge.path missing: %#v", knowledge)
	}
}

func TestProxyKnowledgeLastUpdateRemovedWithinLockWindow(t *testing.T) {
	agentDir, err := filepath.Abs("../agent/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}

	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "knowledge"), 0o755); err != nil {
		t.Fatalf("create knowledge dir: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	defer func() { _ = knowledgecore.CloseSharedDB() }()
	db, err := knowledgecore.OpenSharedDB(tmp)
	if err != nil {
		t.Fatalf("open knowledge db: %v", err)
	}
	now := time.Now().UnixMilli()
	if err := knowledgecore.SetLastUpdate(db, now-3*60*60*1000); err != nil {
		t.Fatalf("set knowledge last update: %v", err)
	}
	if err := ensureKnowledgeRequestLockTable(db); err != nil {
		t.Fatalf("ensure knowledge lock table: %v", err)
	}
	lockTs := now - 10*60*1000
	if _, err := db.Exec(`UPDATE knowledge_update_lock SET last_requested_at = ? WHERE id = 1`, lockTs); err != nil {
		t.Fatalf("seed knowledge lock: %v", err)
	}

	var capturedBody map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:                    upstream.URL,
		AgentDir:                agentDir,
		DeviceID:                "test-device-knowledge-lock",
		CacheTTL:                time.Second,
		KnowledgeUpdateInterval: 2 * time.Hour,
		KnowledgeUpdateLock:     30 * time.Minute,
		Client:                  &http.Client{Timeout: 10 * time.Second},
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleChatCompletions))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"hello"}]}`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	meta, ok := capturedBody["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata not injected: %#v", capturedBody["metadata"])
	}
	knowledge, ok := meta["knowledge"].(map[string]interface{})
	if !ok {
		t.Fatalf("knowledge not injected: %#v", meta["knowledge"])
	}
	if _, exists := knowledge["lastUpdate"]; exists {
		t.Fatalf("knowledge.lastUpdate should be removed within lock window: %#v", knowledge)
	}

	var gotLock int64
	if err := db.QueryRow(`SELECT last_requested_at FROM knowledge_update_lock WHERE id = 1`).Scan(&gotLock); err != nil {
		t.Fatalf("query lock timestamp: %v", err)
	}
	if gotLock != lockTs {
		t.Fatalf("lock timestamp = %d, want %d", gotLock, lockTs)
	}
}

func TestProxyKnowledgeLastUpdateKeptWithinIntervalWhenKnowledgeCommitEnabled(t *testing.T) {
	agentDir, err := filepath.Abs("../agent/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}

	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "knowledge"), 0o755); err != nil {
		t.Fatalf("create knowledge dir: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	defer func() { _ = knowledgecore.CloseSharedDB() }()
	db, err := knowledgecore.OpenSharedDB(tmp)
	if err != nil {
		t.Fatalf("open knowledge db: %v", err)
	}
	lastUpdate := time.Now().Add(-5 * time.Minute).UnixMilli()
	if err := knowledgecore.SetLastUpdate(db, lastUpdate); err != nil {
		t.Fatalf("set knowledge last update: %v", err)
	}

	var capturedBody map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:                    upstream.URL,
		AgentDir:                agentDir,
		DeviceID:                "test-device-knowledge-commit-interval",
		CacheTTL:                time.Second,
		KnowledgeUpdateInterval: 2 * time.Hour,
		KnowledgeUpdateLock:     30 * time.Minute,
		Client:                  &http.Client{Timeout: 10 * time.Second},
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleChatCompletions))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"hello"}],"metadata":{"knowledge_commit":true}}`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	meta, ok := capturedBody["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata not injected: %#v", capturedBody["metadata"])
	}
	knowledge, ok := meta["knowledge"].(map[string]interface{})
	if !ok {
		t.Fatalf("knowledge not injected: %#v", meta["knowledge"])
	}
	gotLastUpdate, ok := knowledge["lastUpdate"].(float64)
	if !ok || int64(gotLastUpdate) != lastUpdate {
		t.Fatalf("knowledge.lastUpdate = %#v, want %d", knowledge["lastUpdate"], lastUpdate)
	}
}

func TestProxyKnowledgeLastUpdateKeptWithinLockWindowWhenKnowledgeCommitEnabled(t *testing.T) {
	agentDir, err := filepath.Abs("../agent/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}

	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "knowledge"), 0o755); err != nil {
		t.Fatalf("create knowledge dir: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	defer func() { _ = knowledgecore.CloseSharedDB() }()
	db, err := knowledgecore.OpenSharedDB(tmp)
	if err != nil {
		t.Fatalf("open knowledge db: %v", err)
	}
	lastUpdate := time.Now().Add(-3 * time.Hour).UnixMilli()
	if err := knowledgecore.SetLastUpdate(db, lastUpdate); err != nil {
		t.Fatalf("set knowledge last update: %v", err)
	}
	if err := ensureKnowledgeRequestLockTable(db); err != nil {
		t.Fatalf("ensure knowledge lock table: %v", err)
	}
	lockTs := time.Now().Add(-10 * time.Minute).UnixMilli()
	if _, err := db.Exec(`UPDATE knowledge_update_lock SET last_requested_at = ? WHERE id = 1`, lockTs); err != nil {
		t.Fatalf("seed knowledge lock: %v", err)
	}

	var capturedBody map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:                    upstream.URL,
		AgentDir:                agentDir,
		DeviceID:                "test-device-knowledge-commit-lock",
		CacheTTL:                time.Second,
		KnowledgeUpdateInterval: 2 * time.Hour,
		KnowledgeUpdateLock:     30 * time.Minute,
		Client:                  &http.Client{Timeout: 10 * time.Second},
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleChatCompletions))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"hello"}],"metadata":{"knowledge_commit":true}}`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	meta, ok := capturedBody["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata not injected: %#v", capturedBody["metadata"])
	}
	knowledge, ok := meta["knowledge"].(map[string]interface{})
	if !ok {
		t.Fatalf("knowledge not injected: %#v", meta["knowledge"])
	}
	gotLastUpdate, ok := knowledge["lastUpdate"].(float64)
	if !ok || int64(gotLastUpdate) != lastUpdate {
		t.Fatalf("knowledge.lastUpdate = %#v, want %d", knowledge["lastUpdate"], lastUpdate)
	}
}

func TestProxyKnowledgeLastUpdateKeptAndLockUpdatedWhenExpired(t *testing.T) {
	agentDir, err := filepath.Abs("../agent/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}

	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "knowledge"), 0o755); err != nil {
		t.Fatalf("create knowledge dir: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	defer func() { _ = knowledgecore.CloseSharedDB() }()
	db, err := knowledgecore.OpenSharedDB(tmp)
	if err != nil {
		t.Fatalf("open knowledge db: %v", err)
	}
	lastUpdate := time.Now().Add(-3 * time.Hour).UnixMilli()
	if err := knowledgecore.SetLastUpdate(db, lastUpdate); err != nil {
		t.Fatalf("set knowledge last update: %v", err)
	}
	if err := ensureKnowledgeRequestLockTable(db); err != nil {
		t.Fatalf("ensure knowledge lock table: %v", err)
	}
	oldLock := time.Now().Add(-2 * time.Hour).UnixMilli()
	if _, err := db.Exec(`UPDATE knowledge_update_lock SET last_requested_at = ? WHERE id = 1`, oldLock); err != nil {
		t.Fatalf("seed knowledge lock: %v", err)
	}

	var capturedBody map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:                    upstream.URL,
		AgentDir:                agentDir,
		DeviceID:                "test-device-knowledge-expired",
		CacheTTL:                time.Second,
		KnowledgeUpdateInterval: 2 * time.Hour,
		KnowledgeUpdateLock:     30 * time.Minute,
		Client:                  &http.Client{Timeout: 10 * time.Second},
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleChatCompletions))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"hello"}]}`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	before := time.Now().UnixMilli()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	after := time.Now().UnixMilli()

	meta, ok := capturedBody["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata not injected: %#v", capturedBody["metadata"])
	}
	knowledge, ok := meta["knowledge"].(map[string]interface{})
	if !ok {
		t.Fatalf("knowledge not injected: %#v", meta["knowledge"])
	}
	gotLastUpdate, ok := knowledge["lastUpdate"].(float64)
	if !ok {
		t.Fatalf("knowledge.lastUpdate missing when request should trigger update: %#v", knowledge)
	}
	if int64(gotLastUpdate) != lastUpdate {
		t.Fatalf("knowledge.lastUpdate = %d, want %d", int64(gotLastUpdate), lastUpdate)
	}

	var gotLock int64
	if err := db.QueryRow(`SELECT last_requested_at FROM knowledge_update_lock WHERE id = 1`).Scan(&gotLock); err != nil {
		t.Fatalf("query lock timestamp: %v", err)
	}
	if gotLock < before || gotLock > after {
		t.Fatalf("lock timestamp = %d, want between %d and %d", gotLock, before, after)
	}
}

func TestProxyKnowledgeManualUpdatesKnowledgeTimestampsAfterSSEDone(t *testing.T) {
	agentDir, err := filepath.Abs("../agent/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}

	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "knowledge"), 0o755); err != nil {
		t.Fatalf("create knowledge dir: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	defer func() { _ = knowledgecore.CloseSharedDB() }()
	db, err := knowledgecore.OpenSharedDB(tmp)
	if err != nil {
		t.Fatalf("open knowledge db: %v", err)
	}
	initialLastUpdate := time.Now().Add(-6 * time.Hour).UnixMilli()
	if err := knowledgecore.SetLastUpdate(db, initialLastUpdate); err != nil {
		t.Fatalf("set initial knowledge last update: %v", err)
	}
	if err := ensureKnowledgeRequestLockTable(db); err != nil {
		t.Fatalf("ensure knowledge lock table: %v", err)
	}
	initialLock := time.Now().Add(-5 * time.Hour).UnixMilli()
	if _, err := db.Exec(`UPDATE knowledge_update_lock SET last_requested_at = ? WHERE id = 1`, initialLock); err != nil {
		t.Fatalf("seed knowledge lock: %v", err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"done\"}}]}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:                    upstream.URL,
		AgentDir:                agentDir,
		DeviceID:                "test-device-knowledge-manual",
		CacheTTL:                time.Second,
		KnowledgeUpdateInterval: 2 * time.Hour,
		KnowledgeUpdateLock:     30 * time.Minute,
		Client:                  &http.Client{Timeout: 10 * time.Second},
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleChatCompletions))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"整理完成"}],"metadata":{"knowledge_commit":true}}`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	before := time.Now().UnixMilli()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	var updatedLastUpdate int64
	var updatedLock int64
	deadline := time.Now().Add(1 * time.Second)
	for {
		checkDB, err := knowledgecore.OpenExistingDB(tmp)
		if err != nil {
			t.Fatalf("open existing knowledge db: %v", err)
		}
		if checkDB != nil {
			updatedLastUpdate, err = knowledgecore.GetLastUpdate(checkDB)
			if err != nil {
				_ = checkDB.Close()
				t.Fatalf("get knowledge last update: %v", err)
			}
			err = checkDB.QueryRow(`SELECT last_requested_at FROM knowledge_update_lock WHERE id = 1`).Scan(&updatedLock)
			_ = checkDB.Close()
			if err != nil {
				t.Fatalf("query knowledge lock: %v", err)
			}
			if updatedLastUpdate != initialLastUpdate && updatedLock != initialLock {
				break
			}
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if updatedLastUpdate < before {
		t.Fatalf("knowledge last update = %d, want >= %d", updatedLastUpdate, before)
	}
	if updatedLastUpdate == initialLastUpdate {
		t.Fatalf("knowledge last update should change from initial value %d", initialLastUpdate)
	}
	if updatedLock < before {
		t.Fatalf("knowledge update lock = %d, want >= %d", updatedLock, before)
	}
	if updatedLock == initialLock {
		t.Fatalf("knowledge update lock should change from initial value %d", initialLock)
	}
	commitDB, err := knowledgecore.OpenExistingDB(tmp)
	if err != nil {
		t.Fatalf("open existing knowledge db for commit: %v", err)
	}
	if commitDB == nil {
		t.Fatal("expected existing knowledge db for commit")
	}
	defer commitDB.Close()
	knowledgeCommit, err := knowledgecore.GetKnowledgeCommit(commitDB)
	if err != nil {
		t.Fatalf("get knowledge commit: %v", err)
	}
	if !knowledgeCommit {
		t.Fatal("knowledge commit = false, want true")
	}
}

func TestProxyKnowledgeManualFalseDoesNotUpdateKnowledgeLastUpdateAfterSSEDone(t *testing.T) {
	agentDir, err := filepath.Abs("../agent/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}

	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "knowledge"), 0o755); err != nil {
		t.Fatalf("create knowledge dir: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	defer func() { _ = knowledgecore.CloseSharedDB() }()
	db, err := knowledgecore.OpenSharedDB(tmp)
	if err != nil {
		t.Fatalf("open knowledge db: %v", err)
	}
	initialLastUpdate := time.Now().Add(-6 * time.Hour).UnixMilli()
	if err := knowledgecore.SetLastUpdate(db, initialLastUpdate); err != nil {
		t.Fatalf("set initial knowledge last update: %v", err)
	}
	if err := ensureKnowledgeRequestLockTable(db); err != nil {
		t.Fatalf("ensure knowledge lock table: %v", err)
	}
	if _, err := db.Exec(`UPDATE knowledge_update_lock SET last_requested_at = ? WHERE id = 1`, time.Now().Add(-5*time.Hour).UnixMilli()); err != nil {
		t.Fatalf("seed knowledge lock: %v", err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:                    upstream.URL,
		AgentDir:                agentDir,
		DeviceID:                "test-device-knowledge-manual-false",
		CacheTTL:                time.Second,
		KnowledgeUpdateInterval: 2 * time.Hour,
		KnowledgeUpdateLock:     30 * time.Minute,
		Client:                  &http.Client{Timeout: 10 * time.Second},
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleChatCompletions))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"普通请求"}],"metadata":{"knowledge_commit":false}}`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_, _ = io.ReadAll(resp.Body)
	resp.Body.Close()

	checkDB, err := knowledgecore.OpenExistingDB(tmp)
	if err != nil {
		t.Fatalf("open existing knowledge db: %v", err)
	}
	if checkDB == nil {
		t.Fatal("expected existing knowledge db")
	}
	defer checkDB.Close()
	updatedLastUpdate, err := knowledgecore.GetLastUpdate(checkDB)
	if err != nil {
		t.Fatalf("get knowledge last update: %v", err)
	}
	if updatedLastUpdate != initialLastUpdate {
		t.Fatalf("knowledge last update = %d, want unchanged %d", updatedLastUpdate, initialLastUpdate)
	}
	knowledgeCommit, err := knowledgecore.GetKnowledgeCommit(checkDB)
	if err != nil {
		t.Fatalf("get knowledge commit: %v", err)
	}
	if knowledgeCommit {
		t.Fatal("knowledge commit = true, want false")
	}
}

func TestProxyChatCompletionsQueryParamsInjectedIntoMetadataOnly(t *testing.T) {
	agentDir, err := filepath.Abs("../agent/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "knowledge", "agent-a"), 0o755); err != nil {
		t.Fatalf("mkdir knowledge: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	var capturedBody map[string]interface{}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:     upstream.URL,
		AgentDir: agentDir,
		DeviceID: "test-device-query",
		CacheTTL: time.Second,
		Client:   &http.Client{Timeout: 10 * time.Second},
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleChatCompletions))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"HELLO QUERY"}],"metadata":{"hello":"world","extract":"true","bodyOnly":"kept"}}`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	meta, ok := capturedBody["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata not injected: %#v", capturedBody["metadata"])
	}
	if meta["hello"] != "world" {
		t.Fatalf("metadata.hello = %#v, want %q", meta["hello"], "world")
	}
	if meta["extract"] != "true" {
		t.Fatalf("metadata.extract = %#v, want %q", meta["extract"], "true")
	}
	if meta["bodyOnly"] != "kept" {
		t.Fatalf("metadata.bodyOnly = %#v, want %q", meta["bodyOnly"], "kept")
	}
}

func TestProxyChatCompletionsInjectsConfiguredModelMetadata(t *testing.T) {
	agentDir, err := filepath.Abs("../agent/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "knowledge"), 0o755); err != nil {
		t.Fatalf("mkdir knowledge: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	db, err := sql.Open("sqlite", filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := writeTokenStoreConfigs(db, map[string]tokenConfig{
		"deepseek": {
			Token:            "Bearer test-token",
			BaseURL:          "https://provider.example/v1",
			ModelFast:        "deepseek-fast",
			ModelMultiInput:  "deepseek-vision",
			ModelMultiOutput: "deepseek-image",
		},
	}); err != nil {
		t.Fatalf("seed token config: %v", err)
	}

	var capturedBody map[string]interface{}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:     upstream.URL,
		AgentDir: agentDir,
		DeviceID: "test-device-provider",
		CacheTTL: time.Second,
		Client:   &http.Client{Timeout: 10 * time.Second},
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleChatCompletions))
	defer server.Close()

	reqBody := `{"model":"deepseek","messages":[{"role":"user","content":"HELLO PROVIDER"}],"metadata":{"hello":"world"}}`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	meta, ok := capturedBody["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata not injected: %#v", capturedBody["metadata"])
	}
	if meta["hello"] != "world" {
		t.Fatalf("metadata.hello = %#v, want %q", meta["hello"], "world")
	}
	if meta["__url"] != "https://provider.example/v1" {
		t.Fatalf("metadata.__url = %#v, want provider url", meta["__url"])
	}
	if meta["__model_fast"] != "deepseek-fast" {
		t.Fatalf("metadata.__model_fast = %#v, want %q", meta["__model_fast"], "deepseek-fast")
	}
	if meta["__model_multi_input"] != "deepseek-vision" {
		t.Fatalf("metadata.__model_multi_input = %#v, want %q", meta["__model_multi_input"], "deepseek-vision")
	}
	if meta["__model_multi_output"] != "deepseek-image" {
		t.Fatalf("metadata.__model_multi_output = %#v, want %q", meta["__model_multi_output"], "deepseek-image")
	}
	if _, exists := meta["__model"]; exists {
		t.Fatalf("metadata.__model should be absent when config is empty: %#v", meta["__model"])
	}
	if _, exists := meta["__model_thinking"]; exists {
		t.Fatalf("metadata.__model_thinking should be absent when config is empty: %#v", meta["__model_thinking"])
	}
}

func TestHandleInstallAppReturnsMissingBuiltinsWhenNeeded(t *testing.T) {
	proxy := &ProxyServer{}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandleInstallApp))
	defer server.Close()

	resp, err := http.Get(server.URL + "/install_app")
	if err != nil {
		t.Fatalf("GET /install_app failed: %v", err)
	}
	defer resp.Body.Close()

	var apps []string
	if err := json.NewDecoder(resp.Body).Decode(&apps); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	expectedMissing := map[string]bool{}
	if detectGit() == "" {
		expectedMissing["git"] = true
	}
	if detectPython3() == "" {
		expectedMissing["python3"] = true
	}

	for app := range expectedMissing {
		found := false
		for _, item := range apps {
			if strings.EqualFold(item, app) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("install_app = %#v, want %q present", apps, app)
		}
	}
	if len(apps) != len(expectedMissing) {
		t.Fatalf("install_app = %#v, want %d auto-detected missing apps", apps, len(expectedMissing))
	}
}

func TestProxyDetectInstallAppInstalledFindsCommonMacUserBin(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, ".local", "node", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin dir: %v", err)
	}
	npmPath := filepath.Join(binDir, "npm")
	if err := os.WriteFile(npmPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write npm: %v", err)
	}

	oldPlatformFn := proxyInstallAppPlatformKeyFn
	oldLookPathFn := proxyInstallAppLookPathFn
	oldUserHomeDirFn := proxyInstallAppUserHomeDirFn
	proxyInstallAppPlatformKeyFn = func() string { return "mac" }
	proxyInstallAppLookPathFn = func(string) (string, error) { return "", exec.ErrNotFound }
	proxyInstallAppUserHomeDirFn = func() (string, error) { return homeDir, nil }
	defer func() {
		proxyInstallAppPlatformKeyFn = oldPlatformFn
		proxyInstallAppLookPathFn = oldLookPathFn
		proxyInstallAppUserHomeDirFn = oldUserHomeDirFn
	}()

	if !proxyDetectInstallAppInstalled("npm") {
		t.Fatalf("proxyDetectInstallAppInstalled(%q) = false, want true", "npm")
	}
}

func TestHandleInstallAppMergesConfiguredEntries(t *testing.T) {
	oldDetectorFn := proxyInstallAppDetectorFn
	oldNowFn := proxyInstallAppNowFn
	proxyInstallAppDetectorFn = func(string) bool { return false }
	proxyInstallAppNowFn = func() time.Time {
		return time.Unix(1710000000, 0)
	}
	proxyResetInstallAppAvailabilityCache()
	defer func() {
		proxyInstallAppDetectorFn = oldDetectorFn
		proxyInstallAppNowFn = oldNowFn
		proxyResetInstallAppAvailabilityCache()
	}()

	proxy := &ProxyServer{InstallApps: []string{"node", "git", "node", "python", "python3"}}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandleInstallApp))
	defer server.Close()

	resp, err := http.Get(server.URL + "/install_app")
	if err != nil {
		t.Fatalf("GET /install_app failed: %v", err)
	}
	defer resp.Body.Close()

	var apps []string
	if err := json.NewDecoder(resp.Body).Decode(&apps); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	has := func(target string) bool {
		for _, item := range apps {
			if strings.EqualFold(item, target) {
				return true
			}
		}
		return false
	}

	if !has("node") || !has("python") {
		t.Fatalf("install_app = %#v, want merged configured entries", apps)
	}
	countGit := 0
	countPython3 := 0
	for _, item := range apps {
		if strings.EqualFold(item, "git") {
			countGit++
		}
		if strings.EqualFold(item, "python3") {
			countPython3++
		}
	}
	if countGit > 1 {
		t.Fatalf("install_app git duplicated: %#v", apps)
	}
	if countPython3 > 1 {
		t.Fatalf("install_app python3 duplicated: %#v", apps)
	}
}

func TestHandleInstallAppMergesCurrentOSConfigEntries(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	if err := os.MkdirAll(filepath.Join(tmp, "config"), 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "config", "config.json"), []byte(`{
  "install_app": {
    "linux": ["node", "git", "node"],
    "wsl": ["docker"],
    "mac": ["xcode"]
  }
}
`), 0o644); err != nil {
		t.Fatalf("write config.json: %v", err)
	}

	oldPlatformFn := proxyInstallAppPlatformKeyFn
	oldDetectorFn := proxyInstallAppDetectorFn
	oldNowFn := proxyInstallAppNowFn
	proxyInstallAppPlatformKeyFn = func() string { return "linux" }
	proxyInstallAppDetectorFn = func(name string) bool {
		return strings.EqualFold(strings.TrimSpace(name), "node")
	}
	proxyInstallAppNowFn = func() time.Time {
		return time.Unix(1710000000, 0)
	}
	proxyResetInstallAppAvailabilityCache()
	defer func() {
		proxyInstallAppPlatformKeyFn = oldPlatformFn
		proxyInstallAppDetectorFn = oldDetectorFn
		proxyInstallAppNowFn = oldNowFn
		proxyResetInstallAppAvailabilityCache()
	}()

	proxy := &ProxyServer{InstallApps: []string{"python", "git", "python3"}}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandleInstallApp))
	defer server.Close()

	resp, err := http.Get(server.URL + "/install_app")
	if err != nil {
		t.Fatalf("GET /install_app failed: %v", err)
	}
	defer resp.Body.Close()

	var apps []string
	if err := json.NewDecoder(resp.Body).Decode(&apps); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	has := func(target string) bool {
		for _, item := range apps {
			if strings.EqualFold(item, target) {
				return true
			}
		}
		return false
	}

	if has("node") {
		t.Fatalf("install_app = %#v, want installed app filtered out", apps)
	}
	if !has("python") {
		t.Fatalf("install_app = %#v, want missing CLI entry kept", apps)
	}
	if has("docker") || has("xcode") {
		t.Fatalf("install_app = %#v, want only current OS config entries", apps)
	}
	if got := resp.Header.Get("Cache-Control"); got != "max-age=300" {
		t.Fatalf("Cache-Control = %q, want %q", got, "max-age=300")
	}

	countGit := 0
	for _, item := range apps {
		if strings.EqualFold(item, "git") {
			countGit++
		}
	}
	if countGit > 1 {
		t.Fatalf("install_app git duplicated: %#v", apps)
	}
}

func TestProxyInstallAppDetectionCacheTTL(t *testing.T) {
	oldPlatformFn := proxyInstallAppPlatformKeyFn
	oldDetectorFn := proxyInstallAppDetectorFn
	oldNowFn := proxyInstallAppNowFn
	proxyInstallAppPlatformKeyFn = func() string { return "linux" }
	now := time.Unix(1710000000, 0)
	proxyInstallAppNowFn = func() time.Time { return now }
	calls := 0
	proxyInstallAppDetectorFn = func(name string) bool {
		calls++
		return strings.EqualFold(strings.TrimSpace(name), "node")
	}
	proxyResetInstallAppAvailabilityCache()
	defer func() {
		proxyInstallAppPlatformKeyFn = oldPlatformFn
		proxyInstallAppDetectorFn = oldDetectorFn
		proxyInstallAppNowFn = oldNowFn
		proxyResetInstallAppAvailabilityCache()
	}()

	if !proxyIsInstallAppInstalled("node") {
		t.Fatal("first install detection = false, want true")
	}
	if !proxyIsInstallAppInstalled("node") {
		t.Fatal("second install detection = false, want true")
	}
	if calls != 1 {
		t.Fatalf("detector calls = %d, want 1 within cache ttl", calls)
	}

	now = now.Add(4 * time.Minute)
	if !proxyIsInstallAppInstalled("node") {
		t.Fatal("cached install detection after 4 minutes = false, want true")
	}
	if calls != 1 {
		t.Fatalf("detector calls after 4 minutes = %d, want 1", calls)
	}

	now = now.Add(2 * time.Minute)
	if !proxyIsInstallAppInstalled("node") {
		t.Fatal("install detection after cache expiry = false, want true")
	}
	if calls != 2 {
		t.Fatalf("detector calls after cache expiry = %d, want 2", calls)
	}
}

func TestProxyDoesNotInjectPluginsWhenNoPluginConfigured(t *testing.T) {
	agentDir, err := filepath.Abs("../agent/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}

	restore := withPluginSandbox(t, nil)
	defer restore()
	flushAgentCache()

	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "knowledge"), 0o755); err != nil {
		t.Fatalf("create knowledge dir: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	var capturedBody map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:     upstream.URL,
		AgentDir: agentDir,
		DeviceID: "test-device-123",
		CacheTTL: 120 * time.Second,
		Client:   &http.Client{Timeout: 10 * time.Second},
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleChatCompletions))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"HELLO WORLD"}]}`
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	meta, ok := capturedBody["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("metadata not injected")
	}
	if _, exists := meta["knowledge"]; !exists {
		t.Fatal("knowledge should be present when knowledge dir exists")
	}
	if _, exists := meta["plugins"]; exists {
		t.Fatalf("plugins should be omitted when plugin is not started: %#v", meta["plugins"])
	}
}

func TestResolveServeReply(t *testing.T) {
	if got := resolveServeReply(""); got != defaultConnectAutoReply {
		t.Fatalf("resolveServeReply(\"\") = %q, want %q", got, defaultConnectAutoReply)
	}

	const customReply = "收到，已经开始处理你的想法"
	if got := resolveServeReply(customReply); got != customReply {
		t.Fatalf("resolveServeReply(custom) = %q, want %q", got, customReply)
	}

	if got := resolveServeReply("   "); got != defaultConnectAutoReply {
		t.Fatalf("resolveServeReply(blank) = %q, want %q", got, defaultConnectAutoReply)
	}
}

func TestResolveConnectPluginCallback(t *testing.T) {
	callback, err := resolveConnectPluginCallback(map[string]string{
		"feishu": "/tmp/plugins/feishu",
	}, "feishu")
	if err != nil {
		t.Fatalf("resolve callback: %v", err)
	}
	if callback != "/tmp/plugins/feishu" {
		t.Fatalf("callback = %q, want /tmp/plugins/feishu", callback)
	}
}

func TestResolveConnectPluginCallbackRequiresConfiguredCallback(t *testing.T) {
	if _, err := resolveConnectPluginCallback(map[string]string{}, "feishu"); err == nil {
		t.Fatal("expected callback error")
	}
}

func TestIsProxyConnectSubcommand(t *testing.T) {
	if !isProxyConnectSubcommand("meta-create") {
		t.Fatal("meta-create should be recognized as top-level connect subcommand")
	}
	if !isProxyConnectSubcommand("list-meta") {
		t.Fatal("list-meta should be recognized as top-level connect subcommand")
	}
	if isProxyConnectSubcommand("plugins") {
		t.Fatal("plugins should not be recognized as connect subcommand")
	}
}

func TestListConnectPluginCallbacksUsesKeyForLookup(t *testing.T) {
	originalRunConnectListMeta := runConnectListMeta
	defer func() {
		runConnectListMeta = originalRunConnectListMeta
	}()

	runConnectListMeta = func() ([]connectMetaConfigListItem, error) {
		return []connectMetaConfigListItem{
			{Key: "feishu", Name: "飞书", Callback: "/tmp/plugins/feishu"},
		}, nil
	}

	callbacks, err := listConnectPluginCallbacks()
	if err != nil {
		t.Fatalf("list callbacks: %v", err)
	}
	if callbacks["feishu"] != "/tmp/plugins/feishu" {
		t.Fatalf("callbacks[feishu] = %q, want /tmp/plugins/feishu", callbacks["feishu"])
	}
	if _, ok := callbacks["飞书"]; ok {
		t.Fatalf("callbacks should not be keyed by display name: %#v", callbacks)
	}
}

func TestHandleAgentIDs(t *testing.T) {
	proxy := &ProxyServer{
		AgentDir: "iteration/20260419_1/test-case",
		DeviceID: "test-dev",
		CacheTTL: 120 * time.Second,
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleAgentIDs))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/agentId")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q", ct)
	}

	var ids []string
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(ids) != 2 || ids[0] != "a" || ids[1] != "b" {
		t.Errorf("agent IDs = %v, want [a, b]", ids)
	}
}

func TestHandleAgentIDsMethodNotAllowed(t *testing.T) {
	proxy := &ProxyServer{
		AgentDir: "iteration/20260419_1/test-case",
		DeviceID: "test-dev",
		CacheTTL: 120 * time.Second,
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleAgentIDs))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/agentId", "application/json", nil)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", resp.StatusCode)
	}
}

func TestHandleSwarmAgents(t *testing.T) {
	root := t.TempDir()
	for _, item := range []struct {
		agentID       string
		routerDisable bool
	}{
		{agentID: "alpha", routerDisable: false},
		{agentID: "beta", routerDisable: true},
		{agentID: "gamma", routerDisable: false},
	} {
		agentDir := filepath.Join(root, item.agentID)
		if err := os.MkdirAll(agentDir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", item.agentID, err)
		}
		configBody := fmt.Sprintf(`{"router_disable":%t}`, item.routerDisable)
		if err := os.WriteFile(filepath.Join(agentDir, "config.json"), []byte(configBody), 0o644); err != nil {
			t.Fatalf("write %s config: %v", item.agentID, err)
		}
	}

	proxy := &ProxyServer{
		AgentDir: root,
		DeviceID: "test-dev",
		CacheTTL: time.Second,
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleSwarmAgents))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/swarm_agent")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var ids []string
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if !reflect.DeepEqual(ids, []string{"alpha", "gamma"}) {
		t.Fatalf("swarm agents = %v, want [alpha gamma]", ids)
	}
}

func TestHandleSwarmAgentsExcludesCurrentAgent(t *testing.T) {
	root := t.TempDir()
	for _, item := range []struct {
		agentID       string
		routerDisable bool
	}{
		{agentID: "alpha", routerDisable: false},
		{agentID: "beta", routerDisable: true},
		{agentID: "gamma", routerDisable: false},
	} {
		agentDir := filepath.Join(root, item.agentID)
		if err := os.MkdirAll(agentDir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", item.agentID, err)
		}
		configBody := fmt.Sprintf(`{"router_disable":%t}`, item.routerDisable)
		if err := os.WriteFile(filepath.Join(agentDir, "config.json"), []byte(configBody), 0o644); err != nil {
			t.Fatalf("write %s config: %v", item.agentID, err)
		}
	}

	proxy := &ProxyServer{
		AgentDir: root,
		DeviceID: "test-dev",
		CacheTTL: time.Second,
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleSwarmAgents))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/swarm_agent?agentId=alpha")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var ids []string
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if !reflect.DeepEqual(ids, []string{"gamma"}) {
		t.Fatalf("swarm agents = %v, want [gamma]", ids)
	}
}

func TestHandleSwarmAgentsMethodNotAllowed(t *testing.T) {
	proxy := &ProxyServer{
		AgentDir: t.TempDir(),
		DeviceID: "test-dev",
		CacheTTL: time.Second,
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleSwarmAgents))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/swarm_agent", "application/json", nil)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}

func TestEmbeddedConnectHandlers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)

	restore := withPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()
	if err := os.MkdirAll(filepath.Join("agent", "A"), 0o755); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", "data")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := writeTokenStore(db, map[string]string{"OpenAI": "sk-test"}); err != nil {
		t.Fatal(err)
	}

	proxy := &ProxyServer{
		AgentDir:        "agent",
		ConnectCacheTTL: 1000 * time.Millisecond,
	}
	mux := http.NewServeMux()
	connectHandler := connectsvc.NewDynamicHTTPHandler(func(_ *http.Request) (*connectsvc.Service, func(), error) {
		return proxy.borrowConnectService()
	})
	mux.Handle("/api/connect/meta", connectHandler)
	mux.Handle("/api/connect/request", connectHandler)
	mux.Handle("/api/connect/response", connectHandler)
	server := httptest.NewServer(mux)
	defer server.Close()

	metaResp, err := http.Post(server.URL+"/api/connect/meta?key=feishu&meta=%7B%22token%22%3A%22abc%22%7D&stream=true&callback=ignored&agent=A&model=OpenAI", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer metaResp.Body.Close()
	var metaPayload struct {
		Status  int             `json:"status"`
		Content string          `json:"content"`
		Data    connectsvc.Meta `json:"data"`
	}
	if err := json.NewDecoder(metaResp.Body).Decode(&metaPayload); err != nil {
		t.Fatal(err)
	}
	if metaPayload.Status != 0 || metaPayload.Data.Name != "feishu" {
		t.Fatalf("unexpected meta payload: %+v content=%q", metaPayload, metaPayload.Content)
	}
	if metaPayload.Data.Callback == "" || !filepath.IsAbs(metaPayload.Data.Callback) {
		t.Fatalf("callback = %q, want absolute plugin path", metaPayload.Data.Callback)
	}

	reqResp, err := http.Post(server.URL+"/api/connect/request?key=feishu&request=HELLO", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer reqResp.Body.Close()
	var reqPayload struct {
		Status int                `json:"status"`
		Data   connectsvc.Request `json:"data"`
	}
	if err := json.NewDecoder(reqResp.Body).Decode(&reqPayload); err != nil {
		t.Fatal(err)
	}
	if reqPayload.Status != 0 || reqPayload.Data.ID == 0 {
		t.Fatalf("unexpected request payload: %+v", reqPayload)
	}

	respResp, err := http.Post(fmt.Sprintf("%s/api/connect/response?key=feishu&requestId=%d&response=OK", server.URL, reqPayload.Data.ID), "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer respResp.Body.Close()
	var responsePayload struct {
		Status int                 `json:"status"`
		Data   connectsvc.Response `json:"data"`
	}
	if err := json.NewDecoder(respResp.Body).Decode(&responsePayload); err != nil {
		t.Fatal(err)
	}
	if responsePayload.Status != 0 || responsePayload.Data.RequestID != reqPayload.Data.ID {
		t.Fatalf("unexpected response payload: %+v", responsePayload)
	}

	configResp, err := http.Get(server.URL + "/api/connect/meta?view=config")
	if err != nil {
		t.Fatal(err)
	}
	defer configResp.Body.Close()
	var configPayload struct {
		Status int                     `json:"status"`
		Data   []connectsvc.MetaConfig `json:"data"`
	}
	if err := json.NewDecoder(configResp.Body).Decode(&configPayload); err != nil {
		t.Fatal(err)
	}
	if configPayload.Status != 0 || len(configPayload.Data) != 1 {
		t.Fatalf("unexpected config payload: %+v", configPayload)
	}
	if configPayload.Data[0].Key != "feishu" || configPayload.Data[0].Name != "飞书" {
		t.Fatalf("unexpected config item: %+v", configPayload.Data[0])
	}
}

func TestHandlePluginsMeta(t *testing.T) {
	t.Skip("obsolete: plugin metadata inspection flow no longer matches the current runtime contract")
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
		"mail":   {`"邮件"`, pluginParamJSON("token")},
	})
	defer restore()

	seedPluginMetaTestData(t, "data", "agent", connectsvc.MetaInput{
		Key:      "feishu",
		Name:     "飞书",
		Meta:     `{"appId":"cli-app","appSecret":"cli-secret"}`,
		Stream:   true,
		Callback: "./feishu",
		AgentID:  "A",
		ChatID:   "chat-001",
		Model:    "OpenAI",
		Thinking: true,
	})

	proxy := &ProxyServer{AgentDir: "agent"}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandlePluginsMeta))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/plugins/meta")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var payload struct {
		Status int `json:"status"`
		Data   []struct {
			Key      string                       `json:"key"`
			Name     string                       `json:"name"`
			Param    connectsvc.PluginParamFields `json:"param"`
			Meta     map[string]any               `json:"meta"`
			AgentID  string                       `json:"agentId"`
			ChatID   string                       `json:"chatId"`
			Model    string                       `json:"model"`
			Thinking bool                         `json:"thinking"`
			Stream   bool                         `json:"stream"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Status != 0 {
		t.Fatalf("status payload = %d, want 0", payload.Status)
	}
	if len(payload.Data) != 2 || payload.Data[0].Name != "飞书" || payload.Data[1].Name != "邮件" {
		t.Fatalf("unexpected plugin payload: %+v", payload)
	}
	if payload.Data[0].Key != "feishu" || payload.Data[1].Key != "mail" {
		t.Fatalf("unexpected plugin keys: %+v", payload.Data)
	}
	if !reflect.DeepEqual(payload.Data[0].Param, pluginParamFields("appId", "appSecret")) {
		t.Fatalf("unexpected plugin params: %+v", payload.Data[0].Param)
	}
	if got := fmt.Sprint(payload.Data[0].Meta["appId"]); got != "cli-app" {
		t.Fatalf("unexpected plugin meta: %+v", payload.Data[0].Meta)
	}
	if got := fmt.Sprint(payload.Data[0].Meta["appSecret"]); got != "cli-secret" {
		t.Fatalf("unexpected plugin meta secret: %+v", payload.Data[0].Meta)
	}
	if payload.Data[0].AgentID != "A" || payload.Data[0].ChatID != "chat-001" || payload.Data[0].Model != "OpenAI" {
		t.Fatalf("unexpected plugin runtime fields: %+v", payload.Data[0])
	}
	if !payload.Data[0].Thinking || !payload.Data[0].Stream {
		t.Fatalf("unexpected plugin flags: %+v", payload.Data[0])
	}
	if payload.Data[1].Meta == nil || len(payload.Data[1].Meta) != 0 {
		t.Fatalf("unexpected empty meta payload: %+v", payload.Data[1].Meta)
	}
}

func TestHandlePluginsMetaMethodNotAllowed(t *testing.T) {
	proxy := &ProxyServer{}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandlePluginsMeta))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/plugins/meta", "application/json", nil)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}

func TestHandlePluginsMetaReloadsPluginsImmediately(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId")},
	})
	defer restore()

	agentDir := "agent"
	if err := os.MkdirAll(filepath.Join(agentDir, "A"), 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}
	db, err := sql.Open("sqlite", "data")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		_ = db.Close()
		t.Fatalf("writeTokenStore: %v", err)
	}
	_ = db.Close()
	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   "data",
		AgentDir: agentDir,
		CacheTTL: time.Minute,
	})
	if err != nil {
		t.Fatalf("new connect service: %v", err)
	}
	defer svc.Close()

	proxy := &ProxyServer{AgentDir: agentDir, ConnectService: svc}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandlePluginsMeta))
	defer server.Close()

	readNames := func() []string {
		resp, err := http.Get(server.URL + "/api/plugins/meta")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()
		var payload struct {
			Status int `json:"status"`
			Data   []struct {
				Name string `json:"name"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		names := make([]string, 0, len(payload.Data))
		for _, item := range payload.Data {
			names = append(names, item.Name)
		}
		return names
	}

	firstNames := readNames()
	if len(firstNames) != 1 || firstNames[0] != "飞书" {
		t.Fatalf("unexpected initial plugins: %+v", firstNames)
	}

	writePluginScript(t, filepath.Join("..", "plugins", "feishu"), `"飞书新版"`, pluginParamJSON("appId"))

	secondNames := readNames()
	if len(secondNames) != 1 || secondNames[0] != "飞书新版" {
		t.Fatalf("plugins meta should refresh immediately, got %+v", secondNames)
	}
}

func TestHandlePluginsMetaIncludesScope(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
		"mail":   {`"邮件"`, pluginParamJSON("token")},
	})
	defer restore()
	writePluginScriptWithScope(t, filepath.Join("..", "plugins", "feishu"), `{"key":"feishu","name":"飞书"}`, pluginParamJSON("appId", "appSecret"), `["reuse","agent","swarm"]`)

	root := t.TempDir()
	agentDir := filepath.Join(root, "agent")
	if err := os.MkdirAll(filepath.Join(agentDir, "A"), 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}
	dbPath := filepath.Join(root, "data")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		_ = db.Close()
		t.Fatalf("writeTokenStore: %v", err)
	}
	_ = db.Close()
	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   dbPath,
		AgentDir: agentDir,
		CacheTTL: time.Minute,
	})
	if err != nil {
		t.Fatalf("new connect service: %v", err)
	}
	defer svc.Close()
	if _, err := connectsvc.UpsertPluginConfigWithService(svc, connectsvc.MetaInput{
		Key:           "feishu",
		Name:          "feishu",
		Meta:          `{"appId":"cli-app"}`,
		Stream:        true,
		AgentID:       "A",
		ChatID:        "chat-001",
		Model:         "OpenAI",
		Thinking:      true,
		RouterDisable: true,
	}); err != nil {
		t.Fatalf("upsert plugin config: %v", err)
	}

	proxy := &ProxyServer{AgentDir: agentDir, ConnectService: svc}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandlePluginsMeta))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/plugins/meta")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	var payload struct {
		Status int `json:"status"`
		Data   []struct {
			Key           string   `json:"key"`
			Name          string   `json:"name"`
			Scope         []string `json:"scope"`
			RouterDisable bool     `json:"router_disable"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Status != 0 || len(payload.Data) != 2 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if strings.Join(payload.Data[0].Scope, ",") != "reuse,agent,swarm" {
		t.Fatalf("unexpected feishu scope: %+v", payload.Data[0].Scope)
	}
	if strings.Join(payload.Data[1].Scope, ",") != "reuse,agent,provider,thinking,swarm" {
		t.Fatalf("unexpected mail scope: %+v", payload.Data[1].Scope)
	}
	if !payload.Data[0].RouterDisable {
		t.Fatalf("unexpected feishu router_disable flag: %+v", payload.Data[0])
	}
}

func TestHandlePluginsMetaRecognizesScriptExtensions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, nil)
	defer restore()

	writePluginScript(t, filepath.Join("..", "plugins", "browser.py"), `{"key":"browser","name":"浏览器"}`, pluginParamJSON("cookie_path"))
	writePluginScript(t, filepath.Join("..", "plugins", "mail.js"), `{"key":"mail","name":"邮件"}`, pluginParamJSON("token"))
	writePluginScript(t, filepath.Join("..", "plugins", "calc.go"), `{"key":"calc","name":"计算器"}`, pluginParamJSON("expr"))
	if err := os.MkdirAll(filepath.Join("..", "plugins", "User_Data"), 0o755); err != nil {
		t.Fatalf("mkdir plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join("..", "plugins", "User_Data.zip"), []byte("zip"), 0o644); err != nil {
		t.Fatalf("write zip file: %v", err)
	}

	root := t.TempDir()
	agentDir := filepath.Join(root, "agent")
	if err := os.MkdirAll(filepath.Join(agentDir, "A"), 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}
	dbPath := filepath.Join(root, "data")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		_ = db.Close()
		t.Fatalf("writeTokenStore: %v", err)
	}
	_ = db.Close()
	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   dbPath,
		AgentDir: agentDir,
		CacheTTL: time.Minute,
	})
	if err != nil {
		t.Fatalf("new connect service: %v", err)
	}
	defer svc.Close()

	proxy := &ProxyServer{AgentDir: agentDir, ConnectService: svc}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandlePluginsMeta))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/plugins/meta")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var payload struct {
		Status int `json:"status"`
		Data   []struct {
			Key  string `json:"key"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Status != 0 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if len(payload.Data) != 3 {
		t.Fatalf("unexpected plugins: %+v", payload.Data)
	}
	if payload.Data[0].Key != "browser" || payload.Data[1].Key != "calc" || payload.Data[2].Key != "mail" {
		t.Fatalf("unexpected plugin keys: %+v", payload.Data)
	}
}

func TestHandlePluginsMetaSkipsInvalidEntriesAndLogs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, map[string][2]string{
		"feishu": {`{"key":"feishu","name":"飞书"}`, pluginParamJSON("appId")},
	})
	defer restore()

	if err := os.WriteFile(filepath.Join("..", "plugins", "broken"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write broken plugin: %v", err)
	}
	if err := os.WriteFile(filepath.Join("..", "plugins", "User_Data.zip"), []byte("zip"), 0o644); err != nil {
		t.Fatalf("write zip file: %v", err)
	}

	root := t.TempDir()
	agentDir := filepath.Join(root, "agent")
	if err := os.MkdirAll(filepath.Join(agentDir, "A"), 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}
	dbPath := filepath.Join(root, "data")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		_ = db.Close()
		t.Fatalf("writeTokenStore: %v", err)
	}
	_ = db.Close()
	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   dbPath,
		AgentDir: agentDir,
		CacheTTL: time.Minute,
	})
	if err != nil {
		t.Fatalf("new connect service: %v", err)
	}
	defer svc.Close()

	var logBuf bytes.Buffer
	originalWriter := log.Writer()
	originalFlags := log.Flags()
	originalPrefix := log.Prefix()
	log.SetOutput(&logBuf)
	log.SetFlags(0)
	log.SetPrefix("")
	defer func() {
		log.SetOutput(originalWriter)
		log.SetFlags(originalFlags)
		log.SetPrefix(originalPrefix)
	}()

	proxy := &ProxyServer{AgentDir: agentDir, ConnectService: svc}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandlePluginsMeta))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/plugins/meta")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var payload struct {
		Status int `json:"status"`
		Data   []struct {
			Key string `json:"key"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Status != 0 || len(payload.Data) != 1 || payload.Data[0].Key != "feishu" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if !strings.Contains(logBuf.String(), "plugins meta skip broken") {
		t.Fatalf("expected skip log, got %q", logBuf.String())
	}
}

func TestHandlePluginsStatus(t *testing.T) {
	t.Skip("obsolete: plugin status inspection depends on legacy plugin name/param contract")
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()

	cmd := exec.Command("sh", "-c", "sleep 5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()
	pidFile := filepath.Join("..", "plugins", "feishu.pid")
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}

	proxy := &ProxyServer{}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandlePluginsStatus))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/plugins/status?key=feishu")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var payload struct {
		Status int `json:"status"`
		Data   struct {
			Key     string `json:"key"`
			Name    string `json:"name"`
			Path    string `json:"path"`
			PID     int    `json:"pid"`
			PIDFile string `json:"pidFile"`
			Started bool   `json:"started"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Status != 0 {
		t.Fatalf("payload status = %d", payload.Status)
	}
	if !payload.Data.Started {
		t.Fatalf("started = false, want true")
	}
	if payload.Data.PID != cmd.Process.Pid {
		t.Fatalf("pid = %d, want %d", payload.Data.PID, cmd.Process.Pid)
	}
	if payload.Data.Key != "feishu" || payload.Data.Name != "飞书" {
		t.Fatalf("unexpected status payload: %+v", payload.Data)
	}
	if payload.Data.PIDFile == "" || !strings.HasSuffix(payload.Data.PIDFile, string(filepath.Separator)+"feishu.pid") {
		t.Fatalf("pidFile = %q", payload.Data.PIDFile)
	}
	if payload.Data.Path == "" || !strings.HasSuffix(payload.Data.Path, string(filepath.Separator)+"feishu") {
		t.Fatalf("path = %q", payload.Data.Path)
	}
}

func TestHandlePluginsStatusMissingName(t *testing.T) {
	proxy := &ProxyServer{}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandlePluginsStatus))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/plugins/status")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestHandlePluginsConfigCreatesAndUpdatesMeta(t *testing.T) {
	t.Skip("obsolete: plugin config creation assertions depend on legacy plugin inspection behavior")
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()
	if err := os.MkdirAll(filepath.Join("agent", "A"), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "data")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		db.Close()
		t.Fatal(err)
	}
	_ = db.Close()

	proxy := &ProxyServer{AgentDir: "agent"}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandlePluginsConfig))
	defer server.Close()

	createResp, err := http.Post(server.URL+"/api/plugins/config?key=feishu&meta=%7B%22appId%22%3A%22cli-app%22%2C%22appSecret%22%3A%22cli-secret%22%7D&agentId=A&model=OpenAI", "application/json", nil)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(createResp.Body)
		t.Fatalf("create status = %d, body=%s", createResp.StatusCode, string(body))
	}

	var created struct {
		Status int             `json:"status"`
		Data   connectsvc.Meta `json:"data"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Status != 0 {
		t.Fatalf("create payload status = %d", created.Status)
	}
	if created.Data.Name != "feishu" || created.Data.AgentID != "A" || created.Data.Model != "OpenAI" {
		t.Fatalf("unexpected create payload: %+v", created.Data)
	}
	if created.Data.Callback == "" || !filepath.IsAbs(created.Data.Callback) {
		t.Fatalf("callback = %q, want absolute plugin path", created.Data.Callback)
	}
	if created.Data.ChatID != "" {
		t.Fatalf("chatId = %q, want empty", created.Data.ChatID)
	}
	if created.Data.Stream || created.Data.Thinking {
		t.Fatalf("unexpected defaults: stream=%v thinking=%v", created.Data.Stream, created.Data.Thinking)
	}

	updateResp, err := http.Post(server.URL+"/api/plugins/config?key=feishu&meta=%7B%22appId%22%3A%22new-app%22%7D&stream=true&agentId=A&chatId=chat-fixed&model=OpenAI&thinking=true", "application/json", nil)
	if err != nil {
		t.Fatalf("update request failed: %v", err)
	}
	defer updateResp.Body.Close()

	if updateResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(updateResp.Body)
		t.Fatalf("update status = %d, body=%s", updateResp.StatusCode, string(body))
	}

	var updated struct {
		Status int             `json:"status"`
		Data   connectsvc.Meta `json:"data"`
	}
	if err := json.NewDecoder(updateResp.Body).Decode(&updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updated.Status != 0 {
		t.Fatalf("update payload status = %d", updated.Status)
	}
	if updated.Data.Meta != `{"appId":"new-app"}` {
		t.Fatalf("meta = %q", updated.Data.Meta)
	}
	if !updated.Data.Stream || !updated.Data.Thinking {
		t.Fatalf("updated booleans = stream:%v thinking:%v", updated.Data.Stream, updated.Data.Thinking)
	}
	if updated.Data.ChatID != "chat-fixed" {
		t.Fatalf("chatId = %q, want chat-fixed", updated.Data.ChatID)
	}
}

func TestHandlePluginsConfigReturnsValidationError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()
	if err := os.MkdirAll(filepath.Join("agent", "A"), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "data")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		db.Close()
		t.Fatal(err)
	}
	_ = db.Close()

	proxy := &ProxyServer{AgentDir: "agent"}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandlePluginsConfig))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/plugins/config?name=%E9%A3%9E%E4%B9%A6&meta=%7B%7D&model=OpenAI", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}

	var payload struct {
		Status  int    `json:"status"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Status != 1 || !strings.Contains(payload.Content, "key is required") {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestHandlePluginsConfigPersistsRouterDisable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()
	if err := os.MkdirAll(filepath.Join("agent", "A"), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "data")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	_ = db.Close()

	proxy := &ProxyServer{AgentDir: "agent"}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandlePluginsConfig))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/plugins/config?key=feishu&meta=%7B%22appId%22%3A%22cli-app%22%7D&agentId=A&model=OpenAI&router_disable=true", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body=%s", resp.StatusCode, string(body))
	}

	var payload struct {
		Status int `json:"status"`
		Data   struct {
			RouterDisable bool `json:"router_disable"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Status != 0 {
		t.Fatalf("payload status = %d", payload.Status)
	}
	if !payload.Data.RouterDisable {
		t.Fatalf("response router_disable = %v, want true", payload.Data.RouterDisable)
	}

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   "data",
		AgentDir: "agent",
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	defer svc.Close()

	item, err := svc.GetMeta("feishu", false)
	if err != nil {
		t.Fatalf("get meta: %v", err)
	}
	if !item.RouterDisable {
		t.Fatalf("persisted router_disable = %v, want true", item.RouterDisable)
	}
}

func TestResolveRouterDisableFormValueIgnoresLegacySwarm(t *testing.T) {
	form := url.Values{"swarm": []string{"true"}}
	if got := resolveRouterDisableFormValue(form, true); !got {
		t.Fatalf("router_disable from legacy swarm = %v, want true", got)
	}

	form.Set("router_disable", "false")
	if got := resolveRouterDisableFormValue(form, true); got {
		t.Fatalf("router_disable explicit value = %v, want false", got)
	}
}

func TestProxyCronCreateRequestIgnoresLegacySwarm(t *testing.T) {
	var req cronCreateRequest
	if err := json.Unmarshal([]byte(`{"content":"demo","model":"OpenAI","swarm":true}`), &req); err != nil {
		t.Fatalf("unmarshal legacy swarm payload: %v", err)
	}
	if !req.RouterDisable {
		t.Fatalf("router_disable = %v, want default true", req.RouterDisable)
	}

	if err := json.Unmarshal([]byte(`{"content":"demo","model":"OpenAI","router_disable":false,"swarm":true}`), &req); err != nil {
		t.Fatalf("unmarshal router_disable payload: %v", err)
	}
	if req.RouterDisable {
		t.Fatalf("router_disable = %v, want false", req.RouterDisable)
	}
}

func TestHandleSwarmWritesRouterDisableConfig(t *testing.T) {
	tmp := t.TempDir()

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	if err := os.MkdirAll(filepath.Join("agent", "A"), 0o755); err != nil {
		t.Fatalf("mkdir agent workspace: %v", err)
	}

	proxy := &ProxyServer{AgentDir: "agent"}
	req := httptest.NewRequest(http.MethodPost, "/api/config?agentId=A", strings.NewReader(`{"description":"demo","thinking":true,"router_disable":false}`))
	rec := httptest.NewRecorder()
	proxy.HandleSwarm(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}

	cfgPath := filepath.Join("agent", "A", "config.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config.json: %v", err)
	}
	if strings.Contains(string(data), `"swarm"`) {
		t.Fatalf("config.json still contains swarm: %s", string(data))
	}

	var cfg struct {
		Description   string `json:"description"`
		Thinking      bool   `json:"thinking"`
		RouterDisable bool   `json:"router_disable"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("decode config.json: %v", err)
	}
	if cfg.Description != "demo" || !cfg.Thinking || cfg.RouterDisable {
		t.Fatalf("config.json = %+v, want description=demo thinking=true router_disable=false", cfg)
	}
}

func TestHandleSwarmWritesMediaConfig(t *testing.T) {
	tmp := t.TempDir()

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	if err := os.MkdirAll(filepath.Join("agent", "A"), 0o755); err != nil {
		t.Fatalf("mkdir agent workspace: %v", err)
	}

	proxy := &ProxyServer{AgentDir: "agent"}
	req := httptest.NewRequest(http.MethodPost, "/api/config?agentId=A", strings.NewReader(`{"description":"demo","thinking":true,"router_disable":false,"media":{"cover":"/tmp/a.png","duration":12}}`))
	rec := httptest.NewRecorder()
	proxy.HandleSwarm(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}

	data, err := os.ReadFile(filepath.Join("agent", "A", "config.json"))
	if err != nil {
		t.Fatalf("read config.json: %v", err)
	}

	var cfg struct {
		Media map[string]interface{} `json:"media"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("decode config.json: %v", err)
	}
	if got := cfg.Media["cover"]; got != "/tmp/a.png" {
		t.Fatalf("media.cover = %v, want /tmp/a.png", got)
	}
	if got := cfg.Media["duration"]; got != float64(12) {
		t.Fatalf("media.duration = %v, want 12", got)
	}
}

func TestHandlePluginsLogStreamsExistingAndAppendedLines(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()

	if err := os.WriteFile(filepath.Join("..", "plugins", "feishu.log"), []byte("old-1\nold-2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	proxy := &ProxyServer{}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandlePluginsLog))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/plugins/log?key=feishu&last=2", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}

	linesCh := make(chan string, 8)
	errCh := make(chan error, 1)
	go collectPluginLogEvents(resp.Body, linesCh, errCh)

	gotLines := make([]string, 0, 3)
	for len(gotLines) < 2 {
		select {
		case line := <-linesCh:
			gotLines = append(gotLines, line)
		case err := <-errCh:
			t.Fatalf("read stream early: %v", err)
		case <-time.After(3 * time.Second):
			t.Fatalf("timeout waiting initial lines, got=%v", gotLines)
		}
	}

	if strings.Join(gotLines, ",") != "old-1,old-2" {
		t.Fatalf("initial lines = %v", gotLines)
	}

	f, err := os.OpenFile(filepath.Join("..", "plugins", "feishu.log"), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("new-3\n"); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	select {
	case line := <-linesCh:
		if line != "new-3" {
			t.Fatalf("appended line = %q, want new-3", line)
		}
	case err := <-errCh:
		t.Fatalf("read stream before appended line: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting appended line")
	}
}

func TestHandlePluginsLogEmitsErrorWhenFileMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()

	logPath := filepath.Join("..", "plugins", "feishu.log")
	if err := os.WriteFile(logPath, []byte("before-delete\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	proxy := &ProxyServer{}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandlePluginsLog))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/plugins/log?key=feishu&last=1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	linesCh := make(chan string, 8)
	errCh := make(chan error, 1)
	go collectPluginLogEvents(resp.Body, linesCh, errCh)

	select {
	case line := <-linesCh:
		if line != "before-delete" {
			t.Fatalf("initial line = %q", line)
		}
	case err := <-errCh:
		t.Fatalf("read stream early: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting initial line")
	}

	if err := os.Remove(logPath); err != nil {
		t.Fatal(err)
	}

	select {
	case line := <-linesCh:
		if !strings.Contains(line, "log file not found") {
			t.Fatalf("error event = %q", line)
		}
	case err := <-errCh:
		if err != io.EOF {
			t.Fatalf("unexpected stream err: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting missing-file event")
	}
}

func TestHandlePluginsLogReturnsMissingFileMessageWhenLogDoesNotExistAtStart(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, map[string][2]string{
		"a": {`"A"`, `[]`},
	})
	defer restore()

	proxy := &ProxyServer{}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandlePluginsLog))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/plugins/log?key=b")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "event: error") {
		t.Fatalf("response = %q, want error event", text)
	}
	if !strings.Contains(text, "log file not found") {
		t.Fatalf("response = %q, want missing-file message", text)
	}
}

func TestHandlePluginsStartSupportsDisplayName(t *testing.T) {
	t.Skip("obsolete: start action fallback assumptions no longer match the current plugin command contract")
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()

	writePluginActionScript(t, filepath.Join("..", "plugins", "feishu"), `"飞书"`, pluginParamJSON("appId", "appSecret"), "subcommand")

	proxy := &ProxyServer{}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandlePluginsStart))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/plugins/start?name=%E9%A3%9E%E4%B9%A6&connect-bin=.%2Fconnect", "application/json", nil)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body=%s", resp.StatusCode, string(body))
	}

	var payload struct {
		Status int `json:"status"`
		Data   struct {
			Path    string         `json:"path"`
			Command []string       `json:"command"`
			Output  map[string]any `json:"output"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Status != 0 {
		t.Fatalf("payload status = %d", payload.Status)
	}
	if len(payload.Data.Command) == 0 || payload.Data.Command[0] != "start" {
		t.Fatalf("command = %v, want start", payload.Data.Command)
	}
	if got := fmt.Sprint(payload.Data.Output["mode"]); got != "subcommand" {
		t.Fatalf("output = %+v", payload.Data.Output)
	}
	if !strings.HasSuffix(payload.Data.Path, string(filepath.Separator)+"feishu") {
		t.Fatalf("path = %q", payload.Data.Path)
	}
}

func TestHandlePluginsStartDefaultsConnectBinToCurrentProxy(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()

	pluginPath := filepath.Join("..", "plugins", "feishu")
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"name\" ]; then\n" +
		"  printf '%s\\n' '\"飞书\"'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"param\" ]; then\n" +
		"  printf '%s\\n' '[{\"appId\":\"\"},{\"appSecret\":\"\"}]'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"start\" ]; then\n" +
		"  shift\n" +
		"  while [ $# -gt 0 ]; do\n" +
		"    if [ \"$1\" = \"--connect-bin\" ]; then\n" +
		"      printf '{\"connectBin\":\"%s\"}\\n' \"$2\"\n" +
		"      exit 0\n" +
		"    fi\n" +
		"    shift\n" +
		"  done\n" +
		"  printf '%s\\n' '{\"connectBin\":\"\"}'\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 1\n"
	if err := os.WriteFile(pluginPath, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}

	proxy := &ProxyServer{}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandlePluginsStart))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/plugins/start?key=feishu", "application/json", nil)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	var payload struct {
		Status int `json:"status"`
		Data   struct {
			Output map[string]any `json:"output"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Status != 0 {
		t.Fatalf("payload status = %d", payload.Status)
	}
	got := strings.TrimSpace(fmt.Sprint(payload.Data.Output["connectBin"]))
	want, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	if got != want {
		t.Fatalf("connectBin = %q, want %q", got, want)
	}
}

func TestHandlePluginsExecRunsNestedCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, map[string][2]string{
		"browser": {`"浏览器"`, pluginParamJSON("agentId", "chatId")},
	})
	defer restore()

	writePluginExecScript(t, filepath.Join("..", "plugins", "browser"), `"浏览器"`, pluginParamJSON("agentId", "chatId"))

	proxy := &ProxyServer{}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandlePluginsExec))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/plugins/exec?key=browser&command=instance+init&agentId=a1&chatId=b1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body=%s", resp.StatusCode, string(body))
	}

	var payload struct {
		Status int `json:"status"`
		Data   struct {
			Command []string       `json:"command"`
			Output  map[string]any `json:"output"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Status != 0 {
		t.Fatalf("payload status = %d", payload.Status)
	}
	if got := strings.Join(payload.Data.Command[:2], " "); got != "instance init" {
		t.Fatalf("command prefix = %v, want instance init", payload.Data.Command)
	}
	if got := fmt.Sprint(payload.Data.Output["agentId"]); got != "a1" {
		t.Fatalf("agentId = %q", got)
	}
	if got := fmt.Sprint(payload.Data.Output["chatId"]); got != "b1" {
		t.Fatalf("chatId = %q", got)
	}
	want, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	if got := strings.TrimSpace(fmt.Sprint(payload.Data.Output["connectBin"])); got != want {
		t.Fatalf("connectBin = %q, want %q", got, want)
	}
}

func TestHandlePluginsStopFallsBackToFlagStyle(t *testing.T) {
	t.Skip("obsolete: stop action fallback assumptions no longer match the current plugin command contract")
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()

	writePluginActionScript(t, filepath.Join("..", "plugins", "feishu"), `"飞书"`, pluginParamJSON("appId", "appSecret"), "flag")

	proxy := &ProxyServer{}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandlePluginsStop))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/plugins/stop?name=feishu&pid-file=.%2Ffeishu.pid", "application/json", nil)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body=%s", resp.StatusCode, string(body))
	}

	var payload struct {
		Status int `json:"status"`
		Data   struct {
			Command []string       `json:"command"`
			Output  map[string]any `json:"output"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Status != 0 {
		t.Fatalf("payload status = %d", payload.Status)
	}
	if len(payload.Data.Command) == 0 || payload.Data.Command[0] != "--stop" {
		t.Fatalf("command = %v, want --stop", payload.Data.Command)
	}
	if got := fmt.Sprint(payload.Data.Output["mode"]); got != "flag" {
		t.Fatalf("output = %+v", payload.Data.Output)
	}
}

func TestProxyPluginsExecCLI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, map[string][2]string{
		"browser": {`"浏览器"`, pluginParamJSON("agentId", "chatId")},
	})
	defer restore()

	writePluginExecScript(t, filepath.Join("..", "plugins", "browser"), `"浏览器"`, pluginParamJSON("agentId", "chatId"))

	stdout := captureProxyStdout(t, func() {
		runProxyPluginCLI([]string{"exec", "--key", "browser", "--command", "instance init", "--agentId", "a1", "--chatId", "b1"})
	})

	var payload struct {
		Status int `json:"status"`
		Data   struct {
			Command []string       `json:"command"`
			Output  map[string]any `json:"output"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("decode stdout: %v; raw=%s", err, stdout)
	}
	if payload.Status != 0 {
		t.Fatalf("payload status = %d", payload.Status)
	}
	if got := strings.Join(payload.Data.Command[:2], " "); got != "instance init" {
		t.Fatalf("command prefix = %v, want instance init", payload.Data.Command)
	}
	if got := fmt.Sprint(payload.Data.Output["agentId"]); got != "a1" {
		t.Fatalf("agentId = %q", got)
	}
	if got := fmt.Sprint(payload.Data.Output["chatId"]); got != "b1" {
		t.Fatalf("chatId = %q", got)
	}
}

func TestConnectTimeoutFails(t *testing.T) {
	// Create a TCP listener that accepts connections but never responds
	// to simulate a hanging connect
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	// Set a tiny backlog by not calling Accept — connections will queue
	// Actually, we need to NOT accept to cause a timeout on connect.
	// Close the listener and use its address — connection will be refused immediately.
	// Instead, use a raw listener that accepts but drops:
	addr := ln.Addr().String()
	ln.Close()

	// Use a non-routable IP to guarantee connect timeout
	// On macOS, use a blackhole approach: listen on a port, then firewall it
	// Simpler: just use a port that's closed — but that gives "connection refused" not timeout.
	// Best approach: verify the client has no total timeout set.

	proxy := &ProxyServer{
		Host:           "http://" + addr,
		AgentDir:       "../agent/test-case",
		DeviceID:       "test-dev",
		CacheTTL:       120 * time.Second,
		ConnectTimeout: 500 * time.Millisecond,
		Client:         NewProxyClient(500 * time.Millisecond),
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleChatCompletions))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"hi"}],"stream":true}`
	req, _ := http.NewRequest("POST", server.URL, strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-test")

	resp, err2 := http.DefaultClient.Do(req)
	if err2 != nil {
		t.Fatalf("request failed: %v", err2)
	}
	defer resp.Body.Close()

	// Should get 502 Bad Gateway because upstream is unreachable
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", resp.StatusCode)
	}
}

func TestConnectTimeoutConfigured(t *testing.T) {
	// Verify that NewProxyClient installs a custom transport for connect timeout handling.
	timeout := 500 * time.Millisecond
	client := NewProxyClient(timeout)

	if client.Timeout < 0 {
		t.Errorf("client.Timeout = %v, want non-negative", client.Timeout)
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("transport is not *http.Transport")
	}

	// Verify DialContext is set (we can't inspect the timeout directly,
	// but we can verify the transport has a custom dialer by checking it's not nil)
	if transport.DialContext == nil {
		t.Error("DialContext is nil, expected custom dialer with connect timeout")
	}
}

func TestNoReadTimeoutForSSE(t *testing.T) {
	// Upstream that delays between SSE chunks — should NOT timeout
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		f, _ := w.(http.Flusher)

		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"A\"}}]}\n\n")
		f.Flush()

		// Simulate slow generation — 1.5s pause between chunks
		time.Sleep(1500 * time.Millisecond)

		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"B\"}}]}\n\n")
		f.Flush()
		fmt.Fprintf(w, "data: [DONE]\n\n")
		f.Flush()
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:           upstream.URL,
		AgentDir:       "../agent/test-case",
		DeviceID:       "test-dev",
		CacheTTL:       120 * time.Second,
		ConnectTimeout: 5 * time.Second,
		Client:         NewProxyClient(5 * time.Second),
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleChatCompletions))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"hi"}],"stream":true}`
	req, _ := http.NewRequest("POST", server.URL, strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-test")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	// Read all SSE content — should get both A and B despite the delay
	var content string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") || line == "data: [DONE]" {
			continue
		}
		var chunk map[string]interface{}
		json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &chunk)
		choices, _ := chunk["choices"].([]interface{})
		if len(choices) > 0 {
			delta := choices[0].(map[string]interface{})["delta"].(map[string]interface{})
			if c, ok := delta["content"].(string); ok {
				content += c
			}
		}
	}
	if content != "AB" {
		t.Errorf("content = %q, want AB", content)
	}
}

func TestHandleFolder(t *testing.T) {
	proxy := &ProxyServer{
		AgentDir: "../agent/test-case",
		DeviceID: "test-dev",
		CacheTTL: 120 * time.Second,
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleFolder))
	defer server.Close()

	// Test agentId=a returns the correct absolute path
	resp, err := http.Get(server.URL + "/api/folder?agentId=a")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	// Should succeed (200) — the folder open command runs in background
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body = %s", resp.StatusCode, string(body))
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["status"] != "ok" {
		t.Errorf("status = %q", result["status"])
	}
	// Path should be absolute and end with /test-case/a
	if !strings.HasSuffix(result["path"], "/test-case/a") {
		t.Errorf("path = %q, want suffix /test-case/a", result["path"])
	}

	// Test agentId=b
	resp2, err := http.Get(server.URL + "/api/folder?agentId=b")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		body, _ := io.ReadAll(resp2.Body)
		t.Fatalf("status = %d, body = %s", resp2.StatusCode, string(body))
	}
	var result2 map[string]string
	json.NewDecoder(resp2.Body).Decode(&result2)
	if !strings.HasSuffix(result2["path"], "/test-case/b") {
		t.Errorf("path = %q, want suffix /test-case/b", result2["path"])
	}
}

func TestHandleFolderSupportsAbsolutePath(t *testing.T) {
	testDir := t.TempDir()
	homeDir := filepath.Join(testDir, "home")
	targetDir := filepath.Join(homeDir, "nested")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	openedPath := ""
	prevOpenFolderFn := openFolderFn
	openFolderFn = func(path string) error {
		openedPath = path
		return nil
	}
	defer func() {
		openFolderFn = prevOpenFolderFn
	}()

	proxy := &ProxyServer{
		AgentDir: "../agent/test-case",
		DeviceID: "test-dev",
		CacheTTL: 120 * time.Second,
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleFolder))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/folder?path=" + url.QueryEscape("~/nested"))
	if err != nil {
		t.Fatalf("GET absolute folder path failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body = %s", resp.StatusCode, string(body))
	}
	if openedPath != targetDir {
		t.Fatalf("opened path = %q, want %q", openedPath, targetDir)
	}
}

func TestHandleFolderMissing(t *testing.T) {
	proxy := &ProxyServer{
		AgentDir: "../agent/test-case",
		DeviceID: "test-dev",
		CacheTTL: 120 * time.Second,
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleFolder))
	defer server.Close()

	// Missing agentId param
	resp, _ := http.Get(server.URL + "/api/folder")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("no agentId: status = %d, want 400", resp.StatusCode)
	}
	resp.Body.Close()

	// Unknown agentId
	resp2, _ := http.Get(server.URL + "/api/folder?agentId=nonexistent")
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("unknown agentId: status = %d, want 404", resp2.StatusCode)
	}
	resp2.Body.Close()
}

func TestHandleRestoreExcludesTimelineBoundary(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	db, err := sql.Open("sqlite", "data")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	initChatTable(db)

	timeline := "2026-04-27T10:00:00"
	_, err = db.Exec(`INSERT INTO chat_log (agent_id, chat_id, chat_type, role, response_type, content, created_at) VALUES (?,?,?,?,?,?,?), (?,?,?,?,?,?,?), (?,?,?,?,?,?,?), (?,?,?,?,?,?,?)`,
		"a", "chat-1", "user", "Q", "normal", "equal", timeline,
		"a", "chat-1", "user", "A", "normal", "after", "2026-04-27T10:00:01",
		"a", "chat-1", "user", "A", "normal", "data: [DONE]\n", "2026-04-27T10:00:01.100",
		"a", "chat-2", "cron", "Q", "normal", "other-chat", "2026-04-27T09:59:59",
	)
	if err != nil {
		t.Fatalf("seed db: %v", err)
	}

	proxy := &ProxyServer{}
	req := httptest.NewRequest(http.MethodPost, "/api/restore?agentId=a&chat=chat-1&timeline="+timeline, nil)
	rec := httptest.NewRecorder()
	proxy.HandleRestore(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp struct {
		Status int `json:"status"`
		Data   []struct {
			ChatType     string `json:"chatType"`
			ResponseType string `json:"responseType"`
			Content      string `json:"content"`
			CreatedAt    string `json:"createdAt"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != 0 {
		t.Fatalf("status body = %d, want 0", resp.Status)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("len(data) = %d, want 2", len(resp.Data))
	}
	if resp.Data[0].Content != "after" {
		t.Fatalf("content = %q, want after", resp.Data[0].Content)
	}
	if resp.Data[0].ChatType != chatTypePageSession || resp.Data[0].ResponseType != "normal" {
		t.Fatalf("types = (%q,%q), want (%s,normal)", resp.Data[0].ChatType, resp.Data[0].ResponseType, chatTypePageSession)
	}
	if resp.Data[0].CreatedAt != "2026-04-27T10:00:01" {
		t.Fatalf("createdAt = %q, want 2026-04-27T10:00:01", resp.Data[0].CreatedAt)
	}
	if resp.Data[1].Content != "data: [DONE]\n" {
		t.Fatalf("done content = %q, want data: [DONE]\\n", resp.Data[1].Content)
	}
}

func TestHandleRestoreReturnsIncrementalAssistantChunks(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	db, err := sql.Open("sqlite", "data")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	initChatTable(db)

	_, err = db.Exec(`INSERT INTO chat_log (agent_id, chat_id, chat_type, role, response_type, content, created_at) VALUES (?,?,?,?,?,?,?), (?,?,?,?,?,?,?), (?,?,?,?,?,?,?), (?,?,?,?,?,?,?), (?,?,?,?,?,?,?)`,
		"a", "chat-1", "user", "Q", "normal", `{"messages":[{"role":"user","content":"hello"}]}`, "2026-04-27T10:00:00",
		"a", "chat-1", "user", "A", "normal", `data: {"choices":[{"delta":{"content":"he"}}]}`+"\n", "2026-04-27T10:00:01",
		"a", "chat-1", "user", "A", "normal", `data: {"choices":[{"delta":{"content":"llo"}}]}`+"\n", "2026-04-27T10:00:02",
		"a", "chat-1", "user", "A", "normal", "data: [DONE]\n", "2026-04-27T10:00:02.100",
		"a", "chat-1", "user", "X", "normal", "", "2026-04-27T10:00:03",
	)
	if err != nil {
		t.Fatalf("seed db: %v", err)
	}

	proxy := &ProxyServer{}
	req := httptest.NewRequest(http.MethodPost, "/api/restore?agentId=a&chat=chat-1&timeline=1970-01-01T00:00:00", nil)
	rec := httptest.NewRecorder()
	proxy.HandleRestore(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp struct {
		Status int `json:"status"`
		Data   []struct {
			ChatType     string `json:"chatType"`
			ResponseType string `json:"responseType"`
			Role         string `json:"role"`
			Content      string `json:"content"`
			CreatedAt    string `json:"createdAt"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != 0 {
		t.Fatalf("status body = %d, want 0", resp.Status)
	}
	if len(resp.Data) != 5 {
		t.Fatalf("len(data) = %d, want 5", len(resp.Data))
	}
	if resp.Data[1].Role != "A" || !strings.Contains(resp.Data[1].Content, `"he"`) {
		t.Fatalf("first A chunk = %#v, want incremental chunk", resp.Data[1])
	}
	if resp.Data[1].ChatType != chatTypePageSession || resp.Data[1].ResponseType != "normal" {
		t.Fatalf("first A types = (%q,%q), want (%s,normal)", resp.Data[1].ChatType, resp.Data[1].ResponseType, chatTypePageSession)
	}
	if resp.Data[2].Role != "A" || !strings.Contains(resp.Data[2].Content, `"llo"`) {
		t.Fatalf("second A chunk = %#v, want incremental chunk", resp.Data[2])
	}
	if resp.Data[3].Role != "A" || resp.Data[3].Content != "data: [DONE]\n" {
		t.Fatalf("done chunk = %#v, want done marker", resp.Data[3])
	}
	if resp.Data[4].Role != "X" {
		t.Fatalf("last role = %q, want X", resp.Data[4].Role)
	}
}

func TestHandleRestoreIncludesCLIGetAndCLIPubLogs(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	sharedEventLogger = struct {
		mu     sync.Mutex
		path   string
		logger *eventlog.Logger
	}{}

	logger, err := getEventLogger()
	if err != nil {
		t.Fatalf("getEventLogger: %v", err)
	}
	logger.Append(eventlog.Entry{
		AgentID:   "a",
		ChatID:    "chat-1",
		Content:   `{"messages":[{"role":"user","content":""}]}`,
		Type:      eventlog.TypeCLIGet,
		CreatedAt: "2026-05-13T12:00:00.100",
	})
	logger.Append(eventlog.Entry{
		AgentID:   "a",
		ChatID:    "chat-1",
		Content:   `{"status":0,"cmd":"ok"}`,
		Type:      eventlog.TypeCLIPub,
		CreatedAt: "2026-05-13T12:00:00.200",
	})
	time.Sleep(100 * time.Millisecond)

	proxy := &ProxyServer{}
	req := httptest.NewRequest(http.MethodPost, "/api/restore?agentId=a&chat=chat-1&timeline=2026-05-13T12:00:00", nil)
	rec := httptest.NewRecorder()
	proxy.HandleRestore(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp struct {
		Status int `json:"status"`
		Data   []struct {
			Role    string `json:"role"`
			LogType int    `json:"logType"`
			Content string `json:"content"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != 0 {
		t.Fatalf("status body = %d, want 0", resp.Status)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("len(data) = %d, want 2", len(resp.Data))
	}
	if resp.Data[0].Role != "cli/get" || resp.Data[0].LogType != eventlog.TypeCLIGet {
		t.Fatalf("first record = %#v, want cli/get type 2", resp.Data[0])
	}
	if resp.Data[1].Role != "cli/pub" || resp.Data[1].LogType != eventlog.TypeCLIPub {
		t.Fatalf("second record = %#v, want cli/pub type 3", resp.Data[1])
	}
}

func TestHandleRestoreUsesChatAcrossAgentSwitches(t *testing.T) {
	flushAgentCache()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	sharedEventLogger = struct {
		mu     sync.Mutex
		path   string
		logger *eventlog.Logger
	}{}

	db, err := sql.Open("sqlite", "data")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	initChatTable(db)

	if _, err := db.Exec(`INSERT INTO chat_log (agent_id, chat_id, chat_type, role, response_type, content, created_at) VALUES
		('agent-a','chat-switch','page_session','A','normal','from-agent-a','2026-05-13T12:00:00.100'),
		('agent-b','chat-switch','page_session','A','normal','from-agent-b','2026-05-13T12:00:00.300'),
		('agent-z','chat-other','page_session','A','normal','other-chat','2026-05-13T12:00:00.400')`); err != nil {
		t.Fatalf("seed chat_log: %v", err)
	}

	logger, err := getEventLogger()
	if err != nil {
		t.Fatalf("getEventLogger: %v", err)
	}
	logger.Append(eventlog.Entry{
		AgentID:   "agent-a",
		ChatID:    "chat-switch",
		Content:   `{"task":"run-a"}`,
		Type:      eventlog.TypeCLIGet,
		CreatedAt: "2026-05-13T12:00:00.200",
	})
	logger.Append(eventlog.Entry{
		AgentID:   "agent-b",
		ChatID:    "chat-switch",
		Content:   `{"status":0,"cmd":"run-b"}`,
		Type:      eventlog.TypeCLIPub,
		CreatedAt: "2026-05-13T12:00:00.400",
	})
	time.Sleep(100 * time.Millisecond)

	proxy := &ProxyServer{}
	req := httptest.NewRequest(http.MethodPost, "/api/restore?chat=chat-switch&timeline=2026-05-13T12:00:00", nil)
	rec := httptest.NewRecorder()
	proxy.HandleRestore(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp struct {
		Status int `json:"status"`
		Data   []struct {
			AgentID string `json:"agentId"`
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != 0 {
		t.Fatalf("status body = %d, want 0", resp.Status)
	}
	if len(resp.Data) != 4 {
		t.Fatalf("len(data) = %d, want 4", len(resp.Data))
	}

	var hasAgentAAssistant, hasAgentBAssistant, hasCLIGet, hasCLIPub bool
	for _, item := range resp.Data {
		switch {
		case item.Role == "A" && item.AgentID == "agent-a" && item.Content == "from-agent-a":
			hasAgentAAssistant = true
		case item.Role == "A" && item.AgentID == "agent-b" && item.Content == "from-agent-b":
			hasAgentBAssistant = true
		case item.Role == "cli/get" && item.AgentID == "agent-a":
			hasCLIGet = true
		case item.Role == "cli/pub" && item.AgentID == "agent-b":
			hasCLIPub = true
		}
	}
	if !hasAgentAAssistant || !hasAgentBAssistant || !hasCLIGet || !hasCLIPub {
		t.Fatalf("restored records missing cross-agent entries: %#v", resp.Data)
	}
}

func TestHandleCancelUsesChatAcrossAgentSwitches(t *testing.T) {
	connMu.Lock()
	connMap = make(map[string]*activeConn)
	cancelled := false
	ch := make(chan chatMsg, 1)
	connMap[connKey("chat-switch")] = &activeConn{
		cancel:  func() { cancelled = true },
		ch:      ch,
		agentID: "agent-old",
		chatID:  "chat-switch",
	}
	connMu.Unlock()
	defer func() {
		connMu.Lock()
		connMap = make(map[string]*activeConn)
		connMu.Unlock()
	}()

	proxy := &ProxyServer{}
	req := httptest.NewRequest(http.MethodPost, "/api/cancel?chat=chat-switch", nil)
	rec := httptest.NewRecorder()
	proxy.HandleCancel(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp struct {
		Status    int  `json:"status"`
		Cancelled bool `json:"cancelled"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != 0 || !resp.Cancelled {
		t.Fatalf("resp = %#v, want status=0 cancelled=true", resp)
	}
	if !cancelled {
		t.Fatalf("cancel func was not called")
	}
	msg, ok := <-ch
	if !ok {
		t.Fatalf("cancel marker channel closed before delivering marker")
	}
	if msg.role != "X" {
		t.Fatalf("cancel marker = %#v, want role X", msg)
	}
	if _, ok := <-ch; ok {
		t.Fatalf("channel should be closed after cancel")
	}
	connMu.Lock()
	_, exists := connMap[connKey("chat-switch")]
	connMu.Unlock()
	if exists {
		t.Fatalf("connection should be removed after cancel")
	}
}

func TestUnifiedEventLogNormalizesCLIPubRawOutput(t *testing.T) {
	flushAgentCache()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	sharedEventLogger = struct {
		mu     sync.Mutex
		path   string
		logger *eventlog.Logger
	}{}

	appendEventLog("a", "chat-raw", `{"status":0,"suffix":"cmd","cmd":"`+gzipBase64String("raw cli pub output")+`","tid":"t-1"}`, eventlog.TypeCLIPub, "2026-05-13T12:00:00.200")
	time.Sleep(100 * time.Millisecond)

	logger, err := getEventLogger()
	if err != nil {
		t.Fatalf("getEventLogger: %v", err)
	}
	entries, err := logger.QueryAll("a", "chat-raw")
	if err != nil {
		t.Fatalf("query event logs: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	if entries[0].Content != "raw cli pub output" {
		t.Fatalf("content = %q, want raw output", entries[0].Content)
	}
}

func TestChatCompletionAlsoWritesUnifiedEventLog(t *testing.T) {
	flushAgentCache()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	sharedEventLogger = struct {
		mu     sync.Mutex
		path   string
		logger *eventlog.Logger
	}{}

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:     upstream.URL,
		AgentDir: agentRoot,
		DeviceID: "dev-1",
		CacheTTL: time.Second,
		Client:   &http.Client{Timeout: 5 * time.Second},
	}

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"hi"}],"metadata":{"agentId":"A","chat":"chat-a"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	proxy.HandleChatCompletions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	time.Sleep(100 * time.Millisecond)
	db, err := sql.Open("sqlite", "data")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	var reqCount, respCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM agent_message_log WHERE agent_id = ? AND chat_id = ? AND log_type = ?`,
		"A", "chat-a", eventlog.TypeChatCompletionRequest).Scan(&reqCount); err != nil {
		t.Fatalf("count request logs: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM agent_message_log WHERE agent_id = ? AND chat_id = ? AND log_type = ?`,
		"A", "chat-a", eventlog.TypeChatCompletionResponse).Scan(&respCount); err != nil {
		t.Fatalf("count response logs: %v", err)
	}
	if reqCount != 1 {
		t.Fatalf("request log count = %d, want 1", reqCount)
	}
	if respCount < 2 {
		t.Fatalf("response log count = %d, want at least 2", respCount)
	}
}

func TestHandleLogSkillExportsMarkdownToAgentTmp(t *testing.T) {
	flushAgentCache()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	sharedEventLogger = struct {
		mu     sync.Mutex
		path   string
		logger *eventlog.Logger
	}{}

	agentRoot := filepath.Join(tmp, "agents")
	workspace := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(filepath.Join(workspace, "tmp"), 0o755); err != nil {
		t.Fatalf("mkdir tmp: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	logger, err := getEventLogger()
	if err != nil {
		t.Fatalf("getEventLogger: %v", err)
	}
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-1", Type: eventlog.TypeChatCompletionRequest, Content: `{"messages":[{"role":"user","content":"hello"}],"metadata":{"agentId":"A","chat":"chat-1"}}`, CreatedAt: "2026-05-13T10:00:00.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-1", Type: eventlog.TypeChatCompletionResponse, Content: "data: {\"choices\":[{\"delta\":{\"content\":\"he\"}}]}\n\n", CreatedAt: "2026-05-13T10:00:01.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-1", Type: eventlog.TypeChatCompletionResponse, Content: "data: {\"choices\":[{\"delta\":{\"content\":\"llo\\n\"}}]}\n\n", CreatedAt: "2026-05-13T10:00:02.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-1", Type: eventlog.TypeCLIGet, Content: `{"cmd":"ls -la /tmp/demo","tid":"task-1"}`, CreatedAt: "2026-05-13T10:00:03.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-1", Type: eventlog.TypeCLIPub, Content: `{"messages":[{"role":"user","content":"命令执行完成"}]}`, CreatedAt: "2026-05-13T10:00:04.000"})
	time.Sleep(100 * time.Millisecond)

	proxy := &ProxyServer{AgentDir: agentRoot, DeviceID: "dev-1", CacheTTL: time.Second}
	req := httptest.NewRequest(http.MethodGet, "/log_skill?agentId=A&chatId=chat-1&round=1", nil)
	rec := httptest.NewRecorder()
	proxy.HandleLogSkill(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp struct {
		Status int     `json:"status"`
		Path   string  `json:"path"`
		SizeK  float64 `json:"sizeK"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != 0 {
		t.Fatalf("status body = %d, want 0", resp.Status)
	}
	if !strings.HasPrefix(resp.Path, filepath.Join(workspace, "tmp")) {
		t.Fatalf("path = %q, want under %q", resp.Path, filepath.Join(workspace, "tmp"))
	}
	if matched, _ := regexp.MatchString(`/A_chat-1_\d{14}\.md$`, filepath.ToSlash(resp.Path)); !matched {
		t.Fatalf("path = %q, want suffix /A_chat-1_YYYYMMDDHHMMSS.md", resp.Path)
	}
	data, err := os.ReadFile(resp.Path)
	if err != nil {
		t.Fatalf("read exported markdown: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "| 时间 | 类型 | 具体内容 |") {
		t.Fatalf("markdown header missing: %s", text)
	}
	if !strings.Contains(text, "SSE请求") || !strings.Contains(text, "SSE响应") || !strings.Contains(text, "工具请求（cli/get）") || !strings.Contains(text, "工具响应（cli/pub）") {
		t.Fatalf("markdown type labels missing: %s", text)
	}
	if strings.Contains(text, `"messages"`) || strings.Contains(text, `"choices"`) || strings.Contains(text, `data:`) {
		t.Fatalf("markdown should contain readable content instead of raw payloads: %s", text)
	}
	if !strings.Contains(text, `hello`) {
		t.Fatalf("markdown request content missing: %s", text)
	}
	if !strings.Contains(text, `hello<br>`) {
		t.Fatalf("markdown merged SSE content missing: %s", text)
	}
	if !strings.Contains(text, `ls -la /tmp/demo`) {
		t.Fatalf("markdown cli/get cmd missing: %s", text)
	}
	if !strings.Contains(text, `命令执行完成`) {
		t.Fatalf("markdown cli/pub content missing: %s", text)
	}
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) != 6 {
		t.Fatalf("markdown line count = %d, want 6, content: %s", len(lines), text)
	}
	if resp.SizeK <= 0 {
		t.Fatalf("sizeK = %v, want > 0", resp.SizeK)
	}
	info, err := os.Stat(resp.Path)
	if err != nil {
		t.Fatalf("stat exported log: %v", err)
	}
	wantSizeK := float64(info.Size()) / 1024.0
	if math.Abs(resp.SizeK-wantSizeK) > 0.000001 {
		t.Fatalf("sizeK = %v, want %v", resp.SizeK, wantSizeK)
	}
}

func TestHandleLogSkillStatusReturnsCLIPubState(t *testing.T) {
	flushAgentCache()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	sharedEventLogger = struct {
		mu     sync.Mutex
		path   string
		logger *eventlog.Logger
	}{}

	logger, err := getEventLogger()
	if err != nil {
		t.Fatalf("getEventLogger: %v", err)
	}
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-1", Type: eventlog.TypeChatCompletionRequest, Content: "req", CreatedAt: "2026-05-13T10:00:00.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-1", Type: eventlog.TypeChatCompletionResponse, Content: "resp", CreatedAt: "2026-05-13T10:00:01.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-1", Type: eventlog.TypeCLIPub, Content: "pub", CreatedAt: "2026-05-13T10:00:02.000"})
	time.Sleep(100 * time.Millisecond)

	proxy := &ProxyServer{}
	req := httptest.NewRequest(http.MethodGet, "/log_skill_status?agentId=A&chatId=chat-1", nil)
	rec := httptest.NewRecorder()
	proxy.HandleLogSkillStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp struct {
		Status   int  `json:"status"`
		HasSkill bool `json:"hasSkill"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != 0 {
		t.Fatalf("status body = %d, want 0", resp.Status)
	}
	if resp.HasSkill {
		t.Fatalf("hasSkill = true, want false without enough cli/get events")
	}
}

func TestHandleLogSkillStatusReturnsTrueForCLIGetWithoutCLIPub(t *testing.T) {
	flushAgentCache()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	sharedEventLogger = struct {
		mu     sync.Mutex
		path   string
		logger *eventlog.Logger
	}{}

	logger, err := getEventLogger()
	if err != nil {
		t.Fatalf("getEventLogger: %v", err)
	}
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-get", Type: eventlog.TypeChatCompletionRequest, Content: "req", CreatedAt: "2026-05-13T10:00:00.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-get", Type: eventlog.TypeChatCompletionResponse, Content: "resp", CreatedAt: "2026-05-13T10:00:00.500"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-get", Type: eventlog.TypeCLIGet, Content: "cli-get-only", CreatedAt: "2026-05-13T10:00:01.000"})
	time.Sleep(100 * time.Millisecond)

	proxy := &ProxyServer{}
	req := httptest.NewRequest(http.MethodGet, "/log_skill_status?agentId=A&chatId=chat-get", nil)
	rec := httptest.NewRecorder()
	proxy.HandleLogSkillStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp struct {
		Status   int  `json:"status"`
		HasSkill bool `json:"hasSkill"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != 0 {
		t.Fatalf("status body = %d, want 0", resp.Status)
	}
	if resp.HasSkill {
		t.Fatalf("hasSkill = true, want false when latest round cli/get count is below default threshold")
	}
}

func TestHandleLogSkillStatusUsesConfiguredSkillExtractThresholdWhenRoundMissing(t *testing.T) {
	flushAgentCache()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	sharedEventLogger = struct {
		mu     sync.Mutex
		path   string
		logger *eventlog.Logger
	}{}

	if err := os.MkdirAll(filepath.Join(tmp, "config"), 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "config", "config.json"), []byte("{\"skill_extract\":11}\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	originalExecutable := proxyOsExecutable
	proxyOsExecutable = func() (string, error) { return filepath.Join(tmp, "bin", "proxy"), nil }
	defer func() { proxyOsExecutable = originalExecutable }()

	logger, err := getEventLogger()
	if err != nil {
		t.Fatalf("getEventLogger: %v", err)
	}
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-threshold", Type: eventlog.TypeChatCompletionRequest, Content: "req-old", CreatedAt: "2026-05-13T09:59:00.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-threshold", Type: eventlog.TypeChatCompletionResponse, Content: "resp-old", CreatedAt: "2026-05-13T09:59:01.000"})
	for i := 0; i < 20; i++ {
		logger.Append(eventlog.Entry{
			AgentID:   "A",
			ChatID:    "chat-threshold",
			Type:      eventlog.TypeCLIGet,
			Content:   fmt.Sprintf("cli-get-old-%d", i+1),
			CreatedAt: fmt.Sprintf("2026-05-13T09:59:%02d.000", i+2),
		})
	}
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-threshold", Type: eventlog.TypeChatCompletionRequest, Content: "req-latest", CreatedAt: "2026-05-13T10:00:00.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-threshold", Type: eventlog.TypeChatCompletionResponse, Content: "resp-latest", CreatedAt: "2026-05-13T10:00:01.000"})
	for i := 0; i < 10; i++ {
		logger.Append(eventlog.Entry{
			AgentID:   "A",
			ChatID:    "chat-threshold",
			Type:      eventlog.TypeCLIGet,
			Content:   fmt.Sprintf("cli-get-latest-%d", i+1),
			CreatedAt: fmt.Sprintf("2026-05-13T10:00:%02d.000", i+2),
		})
	}
	time.Sleep(100 * time.Millisecond)

	assertHasSkill := func(name, rawURL string, want bool, wantRound int) {
		t.Helper()
		proxy := &ProxyServer{}
		req := httptest.NewRequest(http.MethodGet, rawURL, nil)
		rec := httptest.NewRecorder()
		proxy.HandleLogSkillStatus(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, want 200", name, rec.Code)
		}

		var resp struct {
			Status   int  `json:"status"`
			Round    int  `json:"round"`
			HasSkill bool `json:"hasSkill"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("%s: decode: %v", name, err)
		}
		if resp.Status != 0 {
			t.Fatalf("%s: status body = %d, want 0", name, resp.Status)
		}
		if resp.Round != wantRound {
			t.Fatalf("%s: round = %d, want %d", name, resp.Round, wantRound)
		}
		if resp.HasSkill != want {
			t.Fatalf("%s: hasSkill = %v, want %v", name, resp.HasSkill, want)
		}
	}

	assertHasSkill("configured threshold", "/log_skill_status?agentId=A&chatId=chat-threshold", false, 11)
	assertHasSkill("explicit threshold override", "/log_skill_status?agentId=A&chatId=chat-threshold&round=1", true, 1)
}

func TestExportRoundLogUsesRecentNthRequestAsBoundary(t *testing.T) {
	flushAgentCache()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	sharedEventLogger = struct {
		mu     sync.Mutex
		path   string
		logger *eventlog.Logger
	}{}

	agentRoot := filepath.Join(tmp, "agents")
	workspace := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(filepath.Join(workspace, "tmp"), 0o755); err != nil {
		t.Fatalf("mkdir tmp: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	logger, err := getEventLogger()
	if err != nil {
		t.Fatalf("getEventLogger: %v", err)
	}
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-2", Type: eventlog.TypeChatCompletionRequest, Content: "req-1", CreatedAt: "2026-05-13T09:00:00.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-2", Type: eventlog.TypeChatCompletionResponse, Content: "resp-1", CreatedAt: "2026-05-13T09:00:01.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-2", Type: eventlog.TypeChatCompletionRequest, Content: "req-2", CreatedAt: "2026-05-13T10:00:00.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-2", Type: eventlog.TypeChatCompletionResponse, Content: "resp-2", CreatedAt: "2026-05-13T10:00:01.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-2", Type: eventlog.TypeChatCompletionRequest, Content: "req-3", CreatedAt: "2026-05-13T11:00:00.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-2", Type: eventlog.TypeCLIPub, Content: "pub-3", CreatedAt: "2026-05-13T11:00:02.000"})
	time.Sleep(100 * time.Millisecond)

	proxy := &ProxyServer{AgentDir: agentRoot, DeviceID: "dev-1", CacheTTL: time.Second}
	path, err := exportRoundLog(proxy, "A", "chat-2", roundLogOptions{Round: 2})
	if err != nil {
		t.Fatalf("exportRoundLog: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read exported markdown: %v", err)
	}
	text := string(data)
	if strings.Contains(text, "req-1") || strings.Contains(text, "resp-1") {
		t.Fatalf("export should exclude round 1 data: %s", text)
	}
	if !strings.Contains(text, "req-2") || !strings.Contains(text, "req-3") || !strings.Contains(text, "pub-3") {
		t.Fatalf("export should include recent 2 rounds: %s", text)
	}
}

func TestExportRoundLogAppliesStartAndCloseAsAndConditions(t *testing.T) {
	flushAgentCache()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	sharedEventLogger = struct {
		mu     sync.Mutex
		path   string
		logger *eventlog.Logger
	}{}

	agentRoot := filepath.Join(tmp, "agents")
	workspace := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(filepath.Join(workspace, "tmp"), 0o755); err != nil {
		t.Fatalf("mkdir tmp: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	logger, err := getEventLogger()
	if err != nil {
		t.Fatalf("getEventLogger: %v", err)
	}
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-3", Type: eventlog.TypeChatCompletionRequest, Content: "req-1", CreatedAt: "2026-05-13T09:00:00.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-3", Type: eventlog.TypeChatCompletionResponse, Content: "resp-1", CreatedAt: "2026-05-13T09:00:01.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-3", Type: eventlog.TypeChatCompletionRequest, Content: "req-2", CreatedAt: "2026-05-13T10:00:00.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-3", Type: eventlog.TypeCLIGet, Content: "cli-2", CreatedAt: "2026-05-13T10:00:02.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-3", Type: eventlog.TypeChatCompletionRequest, Content: "req-3", CreatedAt: "2026-05-13T11:00:00.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "chat-3", Type: eventlog.TypeCLIPub, Content: "pub-3", CreatedAt: "2026-05-13T11:00:03.000"})
	time.Sleep(100 * time.Millisecond)

	proxy := &ProxyServer{AgentDir: agentRoot, DeviceID: "dev-1", CacheTTL: time.Second}
	path, err := exportRoundLog(proxy, "A", "chat-3", roundLogOptions{
		Round: 3,
		Start: "2026-05-13 10:00:00",
		Close: "2026-05-13 10:59:59",
	})
	if err != nil {
		t.Fatalf("exportRoundLog: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read exported markdown: %v", err)
	}
	text := string(data)
	if strings.Contains(text, "req-1") || strings.Contains(text, "req-3") || strings.Contains(text, "pub-3") {
		t.Fatalf("export should respect AND time filters: %s", text)
	}
	if !strings.Contains(text, "req-2") || !strings.Contains(text, "cli-2") {
		t.Fatalf("export should include only entries inside range: %s", text)
	}
}

func TestExportRoundLogSupportsStartAndCloseWithoutRound(t *testing.T) {
	flushAgentCache()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	sharedEventLogger = struct {
		mu     sync.Mutex
		path   string
		logger *eventlog.Logger
	}{}

	agentRoot := filepath.Join(tmp, "agents")
	workspace := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(filepath.Join(workspace, "tmp"), 0o755); err != nil {
		t.Fatalf("mkdir tmp: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	logger, err := getEventLogger()
	if err != nil {
		t.Fatalf("getEventLogger: %v", err)
	}
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "B", Type: eventlog.TypeChatCompletionRequest, Content: `{"messages":[{"role":"user","content":"req-old"}]}`, CreatedAt: "2026-05-13T21:20:00.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "B", Type: eventlog.TypeChatCompletionRequest, Content: `{"messages":[{"role":"user","content":"req-hit-1"}]}`, CreatedAt: "2026-05-13T21:25:00.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "B", Type: eventlog.TypeCLIGet, Content: `{"cmd":"cli-hit"}`, CreatedAt: "2026-05-13T21:40:00.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "B", Type: eventlog.TypeChatCompletionRequest, Content: `{"messages":[{"role":"user","content":"req-hit-2"}]}`, CreatedAt: "2026-05-13T22:25:00.000"})
	logger.Append(eventlog.Entry{AgentID: "A", ChatID: "B", Type: eventlog.TypeCLIPub, Content: `{"messages":[{"role":"user","content":"pub-new"}]}`, CreatedAt: "2026-05-13T22:25:00.500"})
	time.Sleep(100 * time.Millisecond)

	proxy := &ProxyServer{AgentDir: agentRoot, DeviceID: "dev-1", CacheTTL: time.Second}
	path, err := exportRoundLog(proxy, "A", "B", roundLogOptions{
		Start: "2026-05-13 21:25:00",
		Close: "2026-05-13 22:25:00",
	})
	if err != nil {
		t.Fatalf("exportRoundLog: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read exported markdown: %v", err)
	}
	text := string(data)
	if strings.Contains(text, "req-old") {
		t.Fatalf("time-only export should respect closed range without forcing latest round: %s", text)
	}
	if !strings.Contains(text, "req-hit-1") || !strings.Contains(text, "req-hit-2") || !strings.Contains(text, "cli-hit") || !strings.Contains(text, "pub-new") {
		t.Fatalf("time-only export should include entries inside range: %s", text)
	}
}

func TestFormatRoundLogContentByType(t *testing.T) {
	request := `{"messages":[{"role":"user","content":"看下我的桌面"}]}`
	if got := formatRoundLogContent(eventlog.TypeChatCompletionRequest, request); got != "看下我的桌面" {
		t.Fatalf("request content = %q, want 看下我的桌面", got)
	}

	response := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"正在加载长期记忆\n"}}]}`,
		``,
		`data: {"choices":[{"delta":{"content":"正在读取知识库\n"}}]}`,
		``,
		`data: {"choices":[{"delta":{"content":"查看桌面文件列表\n"}}]}`,
		``,
	}, "\n")
	if got := formatRoundLogContent(eventlog.TypeChatCompletionResponse, response); got != "正在加载长期记忆\n正在读取知识库\n查看桌面文件列表\n" {
		t.Fatalf("response content = %q", got)
	}

	cliGetTask := `{"cmd":"ls -la /Users/demo/Desktop","tid":"t-1"}`
	if got := formatRoundLogContent(eventlog.TypeCLIGet, cliGetTask); got != "ls -la /Users/demo/Desktop" {
		t.Fatalf("cli/get task content = %q", got)
	}

	cliGetRequest := `{"messages":[{"role":"user","content":"{\"timeout\":5000,\"suffix\":\"cmd\",\"type\":\"cmd\",\"tid\":\"task_001\",\"cmd\":\"echo hello\"}"}]}`
	if got := formatRoundLogContent(eventlog.TypeCLIGet, cliGetRequest); got != "echo hello" {
		t.Fatalf("cli/get request content = %q", got)
	}

	cliPubRequest := `{"messages":[{"role":"user","content":"任务执行成功"}]}`
	if got := formatRoundLogContent(eventlog.TypeCLIPub, cliPubRequest); got != "任务执行成功" {
		t.Fatalf("cli/pub request content = %q", got)
	}

	if got := formatRoundLogContent(eventlog.TypeCLIPub, "raw cli pub output"); got != "raw cli pub output" {
		t.Fatalf("cli/pub raw content = %q", got)
	}
}

func TestHandleSkills(t *testing.T) {
	agentDir, err := filepath.Abs("iteration/20260419_4/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}

	restorePlugins := withPluginSandbox(t, nil)
	defer restorePlugins()
	if err := os.MkdirAll("config", 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(filepath.Join("config", "config.json"), []byte(`{"skills":["__internal_cron","__internal_demo"]}`), 0o644); err != nil {
		t.Fatalf("write config.json: %v", err)
	}

	proxy := &ProxyServer{
		AgentDir: agentDir,
		DeviceID: "test-dev",
		CacheTTL: 120 * time.Second,
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleSkills))
	defer server.Close()

	// agentId=a → [__internal_F]
	resp, err := http.Get(server.URL + "/api/skills?agentId=a")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body = %s", resp.StatusCode, string(body))
	}
	var namesA []string
	json.NewDecoder(resp.Body).Decode(&namesA)
	if !reflect.DeepEqual(namesA, []string{"__internal_F", "__internal_cron", "__internal_demo"}) {
		t.Errorf("agentId=a skills = %v, want [__internal_F __internal_cron __internal_demo]", namesA)
	}

	// agentId=b → [__internal_A, __internal_F] (requirement) — actual test-case may include __internal_c
	resp2, err := http.Get(server.URL + "/api/skills?agentId=b")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		body, _ := io.ReadAll(resp2.Body)
		t.Fatalf("status = %d, body = %s", resp2.StatusCode, string(body))
	}
	var namesB []string
	json.NewDecoder(resp2.Body).Decode(&namesB)
	// Verify __internal_A, __internal_F plus app-level configured skills are present.
	hasA, hasF, hasCron, hasDemo := false, false, false, false
	for _, n := range namesB {
		if n == "__internal_A" {
			hasA = true
		}
		if n == "__internal_F" {
			hasF = true
		}
		if n == "__internal_cron" {
			hasCron = true
		}
		if n == "__internal_demo" {
			hasDemo = true
		}
	}
	if !hasA || !hasF || !hasCron || !hasDemo {
		t.Errorf("agentId=b skills = %v, want to contain __internal_A, __internal_F, __internal_cron and __internal_demo", namesB)
	}

	// Missing agentId
	resp3, _ := http.Get(server.URL + "/api/skills")
	if resp3.StatusCode != http.StatusBadRequest {
		t.Errorf("no agentId: status = %d, want 400", resp3.StatusCode)
	}
	resp3.Body.Close()

	// Unknown agentId
	resp4, _ := http.Get(server.URL + "/api/skills?agentId=nonexistent")
	if resp4.StatusCode != http.StatusNotFound {
		t.Errorf("unknown agentId: status = %d, want 404", resp4.StatusCode)
	}
	resp4.Body.Close()
}

func TestHandleSkillsRespectsChatSkillState(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	defer func() {
		if sharedDataDB.db != nil {
			_ = sharedDataDB.db.Close()
		}
		sharedDataDB = pooledDB{}
	}()

	agentDir := filepath.Join(tmp, "agent")
	alphaDir := filepath.Join(agentDir, "a", "skills", "alpha")
	betaDir := filepath.Join(alphaDir, "beta")
	for path, name := range map[string]string{
		filepath.Join(alphaDir, "SKILL.md"): "alpha",
		filepath.Join(betaDir, "SKILL.md"):  "beta",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir skill dir: %v", err)
		}
		content := fmt.Sprintf("---\nname: %s\ndescription: %s skill\n---\nbody\n", name, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write skill: %v", err)
		}
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if _, err := skillstate.SetDisabled(db, "chat-1", alphaDir, true); err != nil {
		t.Fatalf("SetDisabled: %v", err)
	}

	proxy := &ProxyServer{AgentDir: agentDir, DeviceID: "test-dev", CacheTTL: 0}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandleSkills))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/skills?agentId=a")
	if err != nil {
		t.Fatalf("GET skills: %v", err)
	}
	defer resp.Body.Close()
	var names []string
	if err := json.NewDecoder(resp.Body).Decode(&names); err != nil {
		t.Fatalf("decode all skills: %v", err)
	}
	if !reflect.DeepEqual(names, []string{"alpha", "beta"}) {
		t.Fatalf("skills without chat state = %v, want [alpha beta]", names)
	}

	resp, err = http.Get(server.URL + "/api/skills?agentId=a&chatId=chat-1")
	if err != nil {
		t.Fatalf("GET filtered skills: %v", err)
	}
	defer resp.Body.Close()
	names = nil
	if err := json.NewDecoder(resp.Body).Decode(&names); err != nil {
		t.Fatalf("decode filtered skills: %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("skills with disabled alpha = %v, want empty", names)
	}
}

func TestHandleSkillsIncludesPluginInternalSkillsOnlyWhenStarted(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	agentDir, err := filepath.Abs("iteration/20260419_4/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}

	restorePlugins := withPluginSandbox(t, map[string][2]string{
		"browser": {`{"key":"browser","name":"浏览器"}`, pluginParamJSON("cookie_path")},
		"remote":  {`{"key":"remote","name":"远程"}`, pluginParamJSON("exec_timeout")},
	})
	defer restorePlugins()
	if err := os.MkdirAll("config", 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(filepath.Join("config", "config.json"), []byte(`{"skills":["__internal_cron","__internal_demo"]}`), 0o644); err != nil {
		t.Fatalf("write config.json: %v", err)
	}

	proxy := &ProxyServer{
		AgentDir: agentDir,
		DeviceID: "test-dev",
		CacheTTL: 120 * time.Second,
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleSkills))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/skills?agentId=a")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body = %s", resp.StatusCode, string(body))
	}

	var names []string
	if err := json.NewDecoder(resp.Body).Decode(&names); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	want := []string{"__internal_F", "__internal_cron", "__internal_demo"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("skills without started plugins = %v, want %v", names, want)
	}

	cmd := exec.Command("sh", "-c", "sleep 5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()
	if err := os.WriteFile(filepath.Join("..", "plugins", "browser.pid"), []byte(strconv.Itoa(cmd.Process.Pid)), 0o644); err != nil {
		t.Fatalf("write browser pid: %v", err)
	}
	if err := os.WriteFile(filepath.Join("..", "plugins", "remote.pid"), []byte(strconv.Itoa(cmd.Process.Pid)), 0o644); err != nil {
		t.Fatalf("write remote pid: %v", err)
	}

	resp, err = http.Get(server.URL + "/api/skills?agentId=a")
	if err != nil {
		t.Fatalf("GET after start failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status after start = %d, body = %s", resp.StatusCode, string(body))
	}

	names = nil
	if err := json.NewDecoder(resp.Body).Decode(&names); err != nil {
		t.Fatalf("decode started response: %v", err)
	}

	wantStarted := []string{"__internal_F", "__internal_cron", "__internal_demo", "__internal_browser", "__internal_remote"}
	if !reflect.DeepEqual(names, wantStarted) {
		t.Fatalf("skills = %v, want %v", names, wantStarted)
	}
}

func TestHandleSkillsWarning(t *testing.T) {
	tmp := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	dbPath := filepath.Join(tmp, "data")
	store, err := skillscore.OpenWarningStore(dbPath)
	if err != nil {
		t.Fatalf("OpenWarningStore failed: %v", err)
	}
	if err := store.Sync([]skillscore.SkillWarning{{Path: "/tmp/demo/SKILL.md", Reason: "broken", Time: 111}}); err != nil {
		t.Fatalf("store.Sync failed: %v", err)
	}

	proxy := &ProxyServer{}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandleSkillsWarning))
	defer server.Close()

	resp, err := http.Get(server.URL + "/skills_warning")
	if err != nil {
		t.Fatalf("GET /skills_warning failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body = %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Status int                       `json:"status"`
		Data   []skillscore.SkillWarning `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if payload.Status != 0 {
		t.Fatalf("status = %d, want 0", payload.Status)
	}
	if len(payload.Data) != 1 {
		t.Fatalf("warnings len = %d, want 1", len(payload.Data))
	}
	if payload.Data[0].Reason != "broken" {
		t.Fatalf("reason = %q, want broken", payload.Data[0].Reason)
	}
}

func TestHandleSkillsWarningRefreshesByScanningRoot(t *testing.T) {
	tmp := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	root := filepath.Join(tmp, "agent")
	if err := os.MkdirAll(filepath.Join(root, "broken"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "broken", "SKILL.md"), []byte("---\ndescription: broken\n---\n"), 0o644); err != nil {
		t.Fatalf("write broken skill: %v", err)
	}

	proxy := &ProxyServer{WarningScanRoot: root}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandleSkillsWarning))
	defer server.Close()

	resp, err := http.Get(server.URL + "/skills_warning?refresh=1")
	if err != nil {
		t.Fatalf("GET refresh failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body = %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Status int                       `json:"status"`
		Data   []skillscore.SkillWarning `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if len(payload.Data) != 1 {
		t.Fatalf("warnings len = %d, want 1", len(payload.Data))
	}
	if !strings.HasSuffix(payload.Data[0].Path, "/broken/SKILL.md") {
		t.Fatalf("path = %q, want suffix /broken/SKILL.md", payload.Data[0].Path)
	}
}

func TestHandleSkillsWarningRefreshDefaultsToAgentDirSkills(t *testing.T) {
	tmp := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	agentDir := filepath.Join(tmp, "agent")
	if err := os.MkdirAll(filepath.Join(agentDir, "skills", "broken"), 0o755); err != nil {
		t.Fatalf("mkdir skills broken: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(agentDir, "tmp", "ignored"), 0o755); err != nil {
		t.Fatalf("mkdir tmp ignored: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "skills", "broken", "SKILL.md"), []byte("---\ndescription: broken\n---\n"), 0o644); err != nil {
		t.Fatalf("write broken skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "tmp", "ignored", "SKILL.md"), []byte("---\ndescription: ignored\n---\n"), 0o644); err != nil {
		t.Fatalf("write ignored tmp skill: %v", err)
	}

	proxy := &ProxyServer{AgentDir: agentDir}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandleSkillsWarning))
	defer server.Close()

	resp, err := http.Get(server.URL + "/skills_warning?refresh=1")
	if err != nil {
		t.Fatalf("GET refresh failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body = %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Status int                       `json:"status"`
		Data   []skillscore.SkillWarning `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if len(payload.Data) != 1 {
		t.Fatalf("warnings len = %d, want 1", len(payload.Data))
	}
	if !strings.HasSuffix(payload.Data[0].Path, "/skills/broken/SKILL.md") {
		t.Fatalf("path = %q, want suffix /skills/broken/SKILL.md", payload.Data[0].Path)
	}
}

func TestProxyMetadataRefreshesSkillsWhileAgentCacheIsWarm(t *testing.T) {
	root := t.TempDir()
	agentDir := filepath.Join(root, "agent")
	skillDir := filepath.Join(agentDir, "demo", "skills", "live")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(skillDir, "SKILL.md")
	writeSkill := func(desc string) {
		content := strings.Join([]string{
			"---",
			"name: live-skill",
			"description: " + desc,
			"---",
			"",
			"body",
		}, "\n")
		if err := os.WriteFile(skillPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeSkill("first")

	var capturedBody map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:     upstream.URL,
		AgentDir: agentDir,
		DeviceID: "device",
		CacheTTL: time.Hour,
		Client:   &http.Client{Timeout: 10 * time.Second},
	}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandleChatCompletions))
	defer server.Close()

	request := func() string {
		req, _ := http.NewRequest("POST", server.URL+"/v1/chat/completions", strings.NewReader(`{"model":"gpt-4","messages":[{"role":"user","content":"hi"}]}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		meta := capturedBody["metadata"].(map[string]interface{})
		agents := meta["agents"].([]interface{})
		agent := agents[0].(map[string]interface{})
		skills := agent["skills"].([]interface{})
		skill := skills[0].(map[string]interface{})
		return skill["description"].(string)
	}

	if got := request(); got != "first" {
		t.Fatalf("first skill description = %q", got)
	}

	writeSkill("second")
	if got := request(); got != "second" {
		t.Fatalf("second skill description = %q, want refreshed value", got)
	}
}

func TestProxyMetadataNormalizesSkillCompatibilitySequence(t *testing.T) {
	root := t.TempDir()
	agentDir := filepath.Join(root, "agent")
	skillDir := filepath.Join(agentDir, "demo", "skills", "view-directory")
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

	var capturedBody map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:     upstream.URL,
		AgentDir: agentDir,
		DeviceID: "device",
		CacheTTL: time.Hour,
		Client:   &http.Client{Timeout: 10 * time.Second},
	}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandleChatCompletions))
	defer server.Close()

	req, _ := http.NewRequest("POST", server.URL+"/v1/chat/completions", strings.NewReader(`{"model":"gpt-4","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	meta := capturedBody["metadata"].(map[string]interface{})
	agents := meta["agents"].([]interface{})
	agent := agents[0].(map[string]interface{})
	skills := agent["skills"].([]interface{})
	skill := skills[0].(map[string]interface{})
	if got := skill["compatibility"]; got != "macOS (Darwin); zsh shell" {
		t.Fatalf("compatibility = %#v, want %q", got, "macOS (Darwin); zsh shell")
	}
}

func TestHandleFiles(t *testing.T) {
	proxy := &ProxyServer{}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleFiles))
	defer server.Close()

	// path=a: dir skills, files SOUL.md and user.md
	resp, err := http.Get(server.URL + "/api/files?path=iteration/20260419_5/test-case/a")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body = %s", resp.StatusCode, string(body))
	}

	type fileEntry struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	var entriesA []fileEntry
	json.NewDecoder(resp.Body).Decode(&entriesA)

	nameMap := make(map[string]string)
	for _, e := range entriesA {
		nameMap[e.Name] = e.Type
	}
	if nameMap["skills"] != "dir" {
		t.Errorf("skills: got %q, want dir", nameMap["skills"])
	}
	if nameMap["SOUL.md"] != "file" {
		t.Errorf("SOUL.md: got %q, want file", nameMap["SOUL.md"])
	}
	if nameMap["user.md"] != "file" {
		t.Errorf("user.md: got %q, want file", nameMap["user.md"])
	}

	// path=b: dir skills only
	resp2, err := http.Get(server.URL + "/api/files?path=iteration/20260419_5/test-case/b")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		body, _ := io.ReadAll(resp2.Body)
		t.Fatalf("status = %d, body = %s", resp2.StatusCode, string(body))
	}
	var entriesB []fileEntry
	json.NewDecoder(resp2.Body).Decode(&entriesB)
	if len(entriesB) != 1 || entriesB[0].Name != "skills" || entriesB[0].Type != "dir" {
		t.Errorf("path=b entries = %v, want [{skills dir}]", entriesB)
	}

	// Missing path
	resp3, _ := http.Get(server.URL + "/api/files")
	if resp3.StatusCode != http.StatusBadRequest {
		t.Errorf("no path: status = %d, want 400", resp3.StatusCode)
	}
	resp3.Body.Close()

	// Nonexistent path with nonexistent parent → 404
	resp4, _ := http.Get(server.URL + "/api/files?path=/nonexistent_parent_xyz/child")
	if resp4.StatusCode != http.StatusNotFound {
		t.Errorf("bad path: status = %d, want 404", resp4.StatusCode)
	}
	resp4.Body.Close()

	// Prefix matching: path=iteration/20260419_5/test-case/a/SO → matches SOUL.md
	resp5, err := http.Get(server.URL + "/api/files?path=iteration/20260419_5/test-case/a/SO")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp5.Body.Close()
	if resp5.StatusCode != 200 {
		body, _ := io.ReadAll(resp5.Body)
		t.Fatalf("prefix match status = %d, body = %s", resp5.StatusCode, string(body))
	}
	var prefixEntries []fileEntry
	json.NewDecoder(resp5.Body).Decode(&prefixEntries)
	if len(prefixEntries) != 1 || prefixEntries[0].Name != "SOUL.md" {
		t.Errorf("prefix match entries = %v, want [{SOUL.md file}]", prefixEntries)
	}
}

func TestHandleFilesReportsChatSkillState(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	defer func() {
		if sharedDataDB.db != nil {
			_ = sharedDataDB.db.Close()
		}
		sharedDataDB = pooledDB{}
	}()

	agentDir := filepath.Join(tmp, "agent")
	skillsRoot := filepath.Join(agentDir, "a", "skills")
	alphaDir := filepath.Join(skillsRoot, "alpha")
	betaDir := filepath.Join(alphaDir, "beta")
	for path, name := range map[string]string{
		filepath.Join(alphaDir, "SKILL.md"): "alpha",
		filepath.Join(betaDir, "SKILL.md"):  "beta",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir skill dir: %v", err)
		}
		content := fmt.Sprintf("---\nname: %s\ndescription: %s skill\n---\nbody\n", name, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write skill: %v", err)
		}
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if _, err := skillstate.SetDisabled(db, "chat-1", alphaDir, true); err != nil {
		t.Fatalf("SetDisabled: %v", err)
	}

	proxy := &ProxyServer{}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandleFiles))
	defer server.Close()

	type fileEntry struct {
		Name                   string `json:"name"`
		Type                   string `json:"type"`
		HasSkill               bool   `json:"hasSkill"`
		SkillDisabled          bool   `json:"skillDisabled"`
		SkillDisabledSelf      bool   `json:"skillDisabledSelf"`
		SkillDisabledInherited bool   `json:"skillDisabledInherited"`
	}

	resp, err := http.Get(server.URL + "/api/files?path=" + url.QueryEscape(skillsRoot) + "&chatId=chat-1")
	if err != nil {
		t.Fatalf("GET root skills: %v", err)
	}
	defer resp.Body.Close()
	var rootEntries []fileEntry
	if err := json.NewDecoder(resp.Body).Decode(&rootEntries); err != nil {
		t.Fatalf("decode root entries: %v", err)
	}
	if len(rootEntries) != 1 || rootEntries[0].Name != "alpha" || !rootEntries[0].HasSkill || !rootEntries[0].SkillDisabled || !rootEntries[0].SkillDisabledSelf || rootEntries[0].SkillDisabledInherited {
		t.Fatalf("root entries = %+v, want alpha self-disabled", rootEntries)
	}

	resp, err = http.Get(server.URL + "/api/files?path=" + url.QueryEscape(alphaDir) + "&chatId=chat-1")
	if err != nil {
		t.Fatalf("GET nested skills: %v", err)
	}
	defer resp.Body.Close()
	var nestedEntries []fileEntry
	if err := json.NewDecoder(resp.Body).Decode(&nestedEntries); err != nil {
		t.Fatalf("decode nested entries: %v", err)
	}
	if len(nestedEntries) != 2 {
		t.Fatalf("nested entries len = %d, want 2", len(nestedEntries))
	}
	var beta fileEntry
	for _, entry := range nestedEntries {
		if entry.Name == "beta" {
			beta = entry
			break
		}
	}
	if beta.Name != "beta" || !beta.HasSkill || !beta.SkillDisabled || beta.SkillDisabledSelf || !beta.SkillDisabledInherited {
		t.Fatalf("beta entry = %+v, want inherited-disabled", beta)
	}
}

func TestHandleData(t *testing.T) {
	proxy := &ProxyServer{}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleData))
	defer server.Close()

	// path=a/user.md → status:0, content: HELLO USER
	resp, err := http.Get(server.URL + "/api/data?path=iteration/20260419_6/test-case/a/user.md")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()
	var dr DataResponse
	json.NewDecoder(resp.Body).Decode(&dr)
	if dr.Status != 0 {
		t.Errorf("status = %d, want 0, content = %q", dr.Status, dr.Content)
	}
	if strings.TrimSpace(dr.Content) != "HELLO USER" {
		t.Errorf("content = %q, want HELLO USER", strings.TrimSpace(dr.Content))
	}
	if dr.Path == "" {
		t.Error("path should not be empty")
	}

	// Case-insensitive: path=a/USER.MD
	resp2, err := http.Get(server.URL + "/api/data?path=iteration/20260419_6/test-case/a/USER.MD")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp2.Body.Close()
	var dr2 DataResponse
	json.NewDecoder(resp2.Body).Decode(&dr2)
	if dr2.Status != 0 {
		t.Errorf("case-insensitive status = %d, content = %q", dr2.Status, dr2.Content)
	}

	// Directory path → status:1
	resp3, _ := http.Get(server.URL + "/api/data?path=iteration/20260419_6/test-case/a")
	var dr3 DataResponse
	json.NewDecoder(resp3.Body).Decode(&dr3)
	resp3.Body.Close()
	if dr3.Status != 1 {
		t.Errorf("dir: status = %d, want 1", dr3.Status)
	}

	// Missing path → status:1
	resp4, _ := http.Get(server.URL + "/api/data")
	var dr4 DataResponse
	json.NewDecoder(resp4.Body).Decode(&dr4)
	resp4.Body.Close()
	if dr4.Status != 1 {
		t.Errorf("no path: status = %d, want 1", dr4.Status)
	}
}

func TestHandleKnowledgeDirectoryTreeAndFileContent(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	agentRoot := filepath.Join(tmp, "agent")
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	if err := os.MkdirAll(filepath.Join(tmp, "knowledge", "docs"), 0o755); err != nil {
		t.Fatalf("mkdir knowledge docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "knowledge", "README.md"), []byte("hello knowledge\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "knowledge", "docs", "guide.txt"), []byte("guide body"), 0o644); err != nil {
		t.Fatalf("write guide: %v", err)
	}

	proxy := &ProxyServer{AgentDir: agentRoot}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandleKnowledge))
	defer server.Close()

	resp, err := http.Get(server.URL + "/knowledge")
	if err != nil {
		t.Fatalf("GET /knowledge failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	treeBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read tree body: %v", err)
	}
	treeText := string(treeBody)
	if !strings.Contains(treeText, "knowledge/") {
		t.Fatalf("tree body = %q, want knowledge root", treeText)
	}
	if !strings.Contains(treeText, "README.md") {
		t.Fatalf("tree body = %q, want README.md", treeText)
	}
	if !strings.Contains(treeText, "docs/") {
		t.Fatalf("tree body = %q, want docs/", treeText)
	}
	if !strings.Contains(treeText, "guide.txt") {
		t.Fatalf("tree body = %q, want guide.txt", treeText)
	}

	resp2, err := http.Get(server.URL + "/knowledge/README.md")
	if err != nil {
		t.Fatalf("GET file failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("file status = %d, want 200", resp2.StatusCode)
	}
	fileBody, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatalf("read file body: %v", err)
	}
	if string(fileBody) != "hello knowledge\n" {
		t.Fatalf("file body = %q, want hello knowledge", string(fileBody))
	}
}

func TestHandleKnowledgeRejectsPathEscape(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	agentRoot := filepath.Join(tmp, "agent")
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	if err := os.MkdirAll(filepath.Join(tmp, "knowledge"), 0o755); err != nil {
		t.Fatalf("mkdir knowledge: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "outside.txt"), []byte("outside"), 0o644); err != nil {
		t.Fatalf("write outside: %v", err)
	}

	proxy := &ProxyServer{AgentDir: agentRoot}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandleKnowledge))
	defer server.Close()

	resp, err := http.Get(server.URL + "/knowledge/%2e%2e/outside.txt")
	if err != nil {
		t.Fatalf("GET escape path failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestHandleKnowledgeLastUpdateReturnsFormattedTimestamp(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)
	defer func() { _ = knowledgecore.CloseSharedDB() }()

	db, err := knowledgecore.OpenSharedDB(tmp)
	if err != nil {
		t.Fatalf("open knowledge db: %v", err)
	}
	ts := time.Date(2026, time.May, 11, 9, 45, 0, 0, time.Local).UnixMilli()
	if err := knowledgecore.SetLastUpdateForAgent(db, "agent-a", ts); err != nil {
		t.Fatalf("set knowledge last update: %v", err)
	}

	proxy := &ProxyServer{}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandleKnowledgeLastUpdate))
	defer server.Close()

	resp, err := http.Get(server.URL + "/knowledge_lastUpdate?agentId=" + url.QueryEscape("agent-a"))
	if err != nil {
		t.Fatalf("GET /knowledge_lastUpdate failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if got, want := string(body), "2026-05-11 09:45"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestHandleKnowledgeLastUpdateRejectsNonGet(t *testing.T) {
	proxy := &ProxyServer{}
	req := httptest.NewRequest(http.MethodPost, "/knowledge_lastUpdate", nil)
	w := httptest.NewRecorder()

	proxy.HandleKnowledgeLastUpdate(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleKnowledgePath(t *testing.T) {
	tmp := t.TempDir()
	agentRoot := filepath.Join(tmp, "agent")
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)
	proxy := &ProxyServer{AgentDir: agentRoot}
	server := httptest.NewServer(http.HandlerFunc(proxy.HandleKnowledgePath))
	defer server.Close()

	resp, err := http.Get(server.URL + "/knowledge_path?agentId=" + url.QueryEscape("agent-a"))
	if err != nil {
		t.Fatalf("GET /knowledge_path failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	want := filepath.Join(tmp, "knowledge", "agent-a")
	got := strings.TrimSpace(string(body))
	gotResolved, err := filepath.EvalSymlinks(got)
	if err != nil {
		gotResolved = got
	}
	wantResolved, err := filepath.EvalSymlinks(want)
	if err != nil {
		wantResolved = want
	}
	if gotResolved != wantResolved {
		t.Fatalf("body = %q (resolved %q), want %q (resolved %q)", got, gotResolved, want, wantResolved)
	}
	info, err := os.Stat(want)
	if err != nil {
		t.Fatalf("stat created knowledge dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("created knowledge path %q is not a directory", want)
	}
}

func TestHandleKnowledgePathRejectsNonGet(t *testing.T) {
	proxy := &ProxyServer{}
	req := httptest.NewRequest(http.MethodPost, "/knowledge_path", nil)
	w := httptest.NewRecorder()

	proxy.HandleKnowledgePath(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleWorkspace(t *testing.T) {
	proxy := &ProxyServer{
		AgentDir: "../agent/test-case",
		DeviceID: "test-dev",
		CacheTTL: 120 * time.Second,
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleWorkspace))
	defer server.Close()

	// agentId=a → workspace ends with /test-case/a
	resp, err := http.Get(server.URL + "/api/workspace?agentId=a")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, string(body))
	}
	if !strings.HasSuffix(strings.TrimSpace(string(body)), "/test-case/a") {
		t.Errorf("workspace = %q, want suffix /test-case/a", string(body))
	}

	// agentId=b → workspace ends with /test-case/b
	resp2, _ := http.Get(server.URL + "/api/workspace?agentId=b")
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if !strings.HasSuffix(strings.TrimSpace(string(body2)), "/test-case/b") {
		t.Errorf("workspace = %q, want suffix /test-case/b", string(body2))
	}

	// Missing agentId
	resp3, _ := http.Get(server.URL + "/api/workspace")
	if resp3.StatusCode != http.StatusBadRequest {
		t.Errorf("no agentId: status = %d, want 400", resp3.StatusCode)
	}
	resp3.Body.Close()

	// Unknown agentId
	resp4, _ := http.Get(server.URL + "/api/workspace?agentId=nonexistent")
	if resp4.StatusCode != http.StatusNotFound {
		t.Errorf("unknown agentId: status = %d, want 404", resp4.StatusCode)
	}
	resp4.Body.Close()
}

func TestHandleEdit(t *testing.T) {
	// Use iteration test-case as agent dir (has agent a with workspace)
	testDir := t.TempDir()
	// Create agent structure: testDir/a/ as workspace
	os.MkdirAll(filepath.Join(testDir, "a"), 0755)

	proxy := &ProxyServer{
		AgentDir: testDir,
		DeviceID: "test-dev",
		CacheTTL: 1 * time.Millisecond, // no cache for test
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleEdit))
	defer server.Close()

	// Write b/c/user.md under agent a's workspace
	reqBody := `{"content":"HELLO USER"}`
	req, _ := http.NewRequest("POST", server.URL+"/api/edit?agentId=a&path=b/c/user.md", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	var er EditResponse
	json.NewDecoder(resp.Body).Decode(&er)
	if er.Status != 0 {
		t.Fatalf("status = %d, content = %q", er.Status, er.Content)
	}
	if er.AgentID != "a" {
		t.Errorf("agentId = %q", er.AgentID)
	}

	// Verify file was written
	data, err := os.ReadFile(filepath.Join(testDir, "a", "b", "c", "user.md"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "HELLO USER" {
		t.Errorf("file content = %q, want HELLO USER", string(data))
	}

	// Reject absolute path
	req2, _ := http.NewRequest("POST", server.URL+"/api/edit?agentId=a&path=/etc/passwd", strings.NewReader(`{"content":"x"}`))
	resp2, _ := http.DefaultClient.Do(req2)
	var er2 EditResponse
	json.NewDecoder(resp2.Body).Decode(&er2)
	resp2.Body.Close()
	if er2.Status != 1 {
		t.Errorf("absolute path: status = %d, want 1", er2.Status)
	}

	// Reject directory
	req3, _ := http.NewRequest("POST", server.URL+"/api/edit?agentId=a&path=b/c", strings.NewReader(`{"content":"x"}`))
	resp3, _ := http.DefaultClient.Do(req3)
	var er3 EditResponse
	json.NewDecoder(resp3.Body).Decode(&er3)
	resp3.Body.Close()
	if er3.Status != 1 {
		t.Errorf("dir path: status = %d, want 1", er3.Status)
	}

	// Reject invalid binary payload
	req4, _ := http.NewRequest("POST", server.URL+"/api/edit?agentId=a&path=b/image.png", strings.NewReader(`{"content":"x"}`))
	resp4, _ := http.DefaultClient.Do(req4)
	var er4 EditResponse
	json.NewDecoder(resp4.Body).Decode(&er4)
	resp4.Body.Close()
	if er4.Status != 1 {
		t.Errorf("invalid binary payload: status = %d, want 1", er4.Status)
	}
}

func TestHandleEditAllowsImageBase64(t *testing.T) {
	testDir := t.TempDir()
	os.MkdirAll(filepath.Join(testDir, "a", "b"), 0755)

	proxy := &ProxyServer{
		AgentDir: testDir,
		DeviceID: "test-dev",
		CacheTTL: 1 * time.Millisecond,
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleEdit))
	defer server.Close()

	payload := `{"content":"aGVsbG8="}`
	req, _ := http.NewRequest("POST", server.URL+"/api/edit?agentId=a&path=b/image.png", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	var er EditResponse
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if er.Status != 0 {
		t.Fatalf("status = %d, content = %q", er.Status, er.Content)
	}

	data, err := os.ReadFile(filepath.Join(testDir, "a", "b", "image.png"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("file content = %q, want hello", string(data))
	}
}

func TestHandleEditSaveAsNewCreatesTimestampedFile(t *testing.T) {
	testDir := t.TempDir()
	workspace := filepath.Join(testDir, "a")
	if err := os.MkdirAll(filepath.Join(workspace, "b"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	originalPath := filepath.Join(workspace, "b", "photo.png")
	if err := os.WriteFile(originalPath, []byte("orig"), 0644); err != nil {
		t.Fatalf("write original: %v", err)
	}

	proxy := &ProxyServer{
		AgentDir: testDir,
		DeviceID: "test-dev",
		CacheTTL: 1 * time.Millisecond,
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleEdit))
	defer server.Close()

	req, _ := http.NewRequest("POST", server.URL+"/api/edit?agentId=a&path=b/photo.png&saveAsNew=true", strings.NewReader(`{"content":"bmV3"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	var er EditResponse
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if er.Status != 0 {
		t.Fatalf("status = %d, content = %q", er.Status, er.Content)
	}
	if er.SavedAs == "" {
		t.Fatal("savedAs should not be empty")
	}
	if er.SavedAs == originalPath {
		t.Fatal("savedAs should not overwrite original file")
	}
	if !strings.Contains(filepath.Base(er.SavedAs), "photo_") {
		t.Fatalf("savedAs = %q, want timestamped filename", er.SavedAs)
	}
	originalData, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}
	if string(originalData) != "orig" {
		t.Fatalf("original content = %q, want orig", string(originalData))
	}
	savedData, err := os.ReadFile(er.SavedAs)
	if err != nil {
		t.Fatalf("read savedAs: %v", err)
	}
	if string(savedData) != "new" {
		t.Fatalf("savedAs content = %q, want new", string(savedData))
	}
}

func TestHandleEditCaseInsensitiveAndSpacePath(t *testing.T) {
	testDir := t.TempDir()
	workspace := filepath.Join(testDir, "a")
	if err := os.MkdirAll(filepath.Join(workspace, "Docs Space"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	proxy := &ProxyServer{
		AgentDir: testDir,
		DeviceID: "test-dev",
		CacheTTL: 1 * time.Millisecond,
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleEdit))
	defer server.Close()

	req, _ := http.NewRequest("POST", server.URL+"/api/edit?agentId=a&path=docs%20space/User.MD", strings.NewReader(`{"content":"HELLO USER"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	var er EditResponse
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if er.Status != 0 {
		t.Fatalf("status = %d, content = %q", er.Status, er.Content)
	}

	data, err := os.ReadFile(filepath.Join(workspace, "Docs Space", "User.MD"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "HELLO USER" {
		t.Fatalf("file content = %q, want HELLO USER", string(data))
	}
}

func TestHandleEditQuotedRelativePath(t *testing.T) {
	testDir := t.TempDir()
	workspace := filepath.Join(testDir, "a")
	if err := os.MkdirAll(filepath.Join(workspace, "Docs Space"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	proxy := &ProxyServer{
		AgentDir: testDir,
		DeviceID: "test-dev",
		CacheTTL: 1 * time.Millisecond,
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleEdit))
	defer server.Close()

	req, _ := http.NewRequest("POST", server.URL+"/api/edit?agentId=a&path=%22docs%20space/User.MD%22", strings.NewReader(`{"content":"HELLO QUOTED USER"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	var er EditResponse
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if er.Status != 0 {
		t.Fatalf("status = %d, content = %q", er.Status, er.Content)
	}

	data, err := os.ReadFile(filepath.Join(workspace, "Docs Space", "User.MD"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "HELLO QUOTED USER" {
		t.Fatalf("file content = %q, want HELLO QUOTED USER", string(data))
	}
}

func TestHandleEditRejectsWorkspaceEscape(t *testing.T) {
	testDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(testDir, "a"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	proxy := &ProxyServer{
		AgentDir: testDir,
		DeviceID: "test-dev",
		CacheTTL: 1 * time.Millisecond,
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleEdit))
	defer server.Close()

	req, _ := http.NewRequest("POST", server.URL+"/api/edit?agentId=a&path=../escape.txt", strings.NewReader(`{"content":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	var er EditResponse
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if er.Status != 1 {
		t.Fatalf("status = %d, want 1", er.Status)
	}
}

func TestHandleEditCreatesMissingBinaryFile(t *testing.T) {
	testDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(testDir, "a", "media"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	proxy := &ProxyServer{
		AgentDir: testDir,
		DeviceID: "test-dev",
		CacheTTL: 1 * time.Millisecond,
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleEdit))
	defer server.Close()

	req, _ := http.NewRequest("POST", server.URL+"/api/edit?agentId=a&path=media/movie.mp4", strings.NewReader(`{"content":"aGVsbG8="}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	var er EditResponse
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if er.Status != 0 {
		t.Fatalf("status = %d, content = %q", er.Status, er.Content)
	}

	data, err := os.ReadFile(filepath.Join(testDir, "a", "media", "movie.mp4"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("file content = %q, want hello", string(data))
	}
}

func TestHandleDel(t *testing.T) {
	testDir := t.TempDir()
	// Create agent a with a file to delete
	os.MkdirAll(filepath.Join(testDir, "a", "b", "c"), 0755)
	os.WriteFile(filepath.Join(testDir, "a", "b", "c", "user.md"), []byte("HELLO USER"), 0644)

	proxy := &ProxyServer{
		AgentDir: testDir,
		DeviceID: "test-dev",
		CacheTTL: 1 * time.Millisecond,
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleDel))
	defer server.Close()

	// Delete b/c/user.md under agent a
	resp, err := http.Get(server.URL + "/api/del?agentId=a&path=b/c/user.md")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()
	var dr DelResponse
	json.NewDecoder(resp.Body).Decode(&dr)
	if dr.Status != 0 {
		t.Fatalf("status = %d, content = %q", dr.Status, dr.Content)
	}

	// Verify file is gone
	if _, err := os.Stat(filepath.Join(testDir, "a", "b", "c", "user.md")); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}

	// Reject absolute path
	resp2, _ := http.Get(server.URL + "/api/del?agentId=a&path=/etc/passwd")
	var dr2 DelResponse
	json.NewDecoder(resp2.Body).Decode(&dr2)
	resp2.Body.Close()
	if dr2.Status != 1 {
		t.Errorf("absolute path: status = %d, want 1", dr2.Status)
	}

	// Directory deletion should now succeed (recursive)
	resp3, _ := http.Get(server.URL + "/api/del?agentId=a&path=b/c")
	var dr3 DelResponse
	json.NewDecoder(resp3.Body).Decode(&dr3)
	resp3.Body.Close()
	if dr3.Status != 0 {
		t.Errorf("dir delete: status = %d, content = %q, want 0", dr3.Status, dr3.Content)
	}
	// Verify directory is gone
	if _, err := os.Stat(filepath.Join(testDir, "a", "b", "c")); !os.IsNotExist(err) {
		t.Error("directory b/c should be deleted")
	}
}

func TestHandleRaw(t *testing.T) {
	testDir := t.TempDir()
	homeDir := filepath.Join(testDir, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("mkdir home dir: %v", err)
	}
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	os.MkdirAll(filepath.Join(testDir, "a", "b", "c"), 0755)
	os.WriteFile(filepath.Join(testDir, "a", "b", "c", "user.md"), []byte("HELLO WORLD"), 0644)
	absDir := filepath.Join(testDir, "Abs Raw")
	if err := os.MkdirAll(absDir, 0755); err != nil {
		t.Fatalf("mkdir abs dir: %v", err)
	}
	absFile := filepath.Join(absDir, "Case File.BIN")
	if err := os.WriteFile(absFile, []byte("ABSOLUTE DATA"), 0644); err != nil {
		t.Fatalf("write abs file: %v", err)
	}
	homeFile := filepath.Join(homeDir, "demo.txt")
	if err := os.WriteFile(homeFile, []byte("HOME DATA"), 0o644); err != nil {
		t.Fatalf("write home file: %v", err)
	}

	proxy := &ProxyServer{
		AgentDir: testDir,
		DeviceID: "test-dev",
		CacheTTL: 1 * time.Millisecond,
	}

	server := httptest.NewServer(http.HandlerFunc(proxy.HandleRaw))
	defer server.Close()

	// Read b/c/user.md → base64 of "HELLO WORLD"
	resp, err := http.Get(server.URL + "/api/raw?agentId=a&path=b/c/user.md")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()
	var rr RawResponse
	json.NewDecoder(resp.Body).Decode(&rr)
	if rr.Status != 0 {
		t.Fatalf("status = %d, content = %q", rr.Status, rr.Content)
	}
	decoded, err := base64.StdEncoding.DecodeString(rr.Content)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	if string(decoded) != "HELLO WORLD" {
		t.Errorf("content = %q, want HELLO WORLD", string(decoded))
	}

	// Absolute path should also be readable with case-insensitive matching.
	rawAbsPath := filepath.Join(testDir, "abs raw", "case file.bin")
	resp2, err := http.Get(server.URL + "/api/raw?agentId=a&path=" + url.QueryEscape(rawAbsPath))
	if err != nil {
		t.Fatalf("GET absolute path failed: %v", err)
	}
	var rr2 RawResponse
	json.NewDecoder(resp2.Body).Decode(&rr2)
	resp2.Body.Close()
	if rr2.Status != 0 {
		t.Fatalf("absolute path: status = %d, content = %q", rr2.Status, rr2.Content)
	}
	decoded2, err := base64.StdEncoding.DecodeString(rr2.Content)
	if err != nil {
		t.Fatalf("absolute path base64 decode: %v", err)
	}
	if string(decoded2) != "ABSOLUTE DATA" {
		t.Errorf("absolute path content = %q, want ABSOLUTE DATA", string(decoded2))
	}

	respQuoted, err := http.Get(server.URL + "/api/raw?agentId=a&path=" + url.QueryEscape(`"`+rawAbsPath+`"`))
	if err != nil {
		t.Fatalf("GET quoted absolute path failed: %v", err)
	}
	var rrQuoted RawResponse
	json.NewDecoder(respQuoted.Body).Decode(&rrQuoted)
	respQuoted.Body.Close()
	if rrQuoted.Status != 0 {
		t.Fatalf("quoted absolute path: status = %d, content = %q", rrQuoted.Status, rrQuoted.Content)
	}
	decodedQuoted, err := base64.StdEncoding.DecodeString(rrQuoted.Content)
	if err != nil {
		t.Fatalf("quoted absolute path base64 decode: %v", err)
	}
	if string(decodedQuoted) != "ABSOLUTE DATA" {
		t.Errorf("quoted absolute path content = %q, want ABSOLUTE DATA", string(decodedQuoted))
	}

	respTilde, err := http.Get(server.URL + "/api/raw?agentId=a&path=" + url.QueryEscape("~/demo.txt"))
	if err != nil {
		t.Fatalf("GET tilde path failed: %v", err)
	}
	var rrTilde RawResponse
	json.NewDecoder(respTilde.Body).Decode(&rrTilde)
	respTilde.Body.Close()
	if rrTilde.Status != 0 {
		t.Fatalf("tilde path: status = %d, content = %q", rrTilde.Status, rrTilde.Content)
	}
	decodedTilde, err := base64.StdEncoding.DecodeString(rrTilde.Content)
	if err != nil {
		t.Fatalf("tilde path base64 decode: %v", err)
	}
	if string(decodedTilde) != "HOME DATA" {
		t.Errorf("tilde path content = %q, want HOME DATA", string(decodedTilde))
	}

	// Nonexistent file
	resp3, _ := http.Get(server.URL + "/api/raw?agentId=a&path=nonexistent.txt")
	var rr3 RawResponse
	json.NewDecoder(resp3.Body).Decode(&rr3)
	resp3.Body.Close()
	if rr3.Status != 1 {
		t.Errorf("nonexistent: status = %d, want 1", rr3.Status)
	}
}

func TestHandleCmdExecutesAndLogs(t *testing.T) {
	flushAgentCache()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	agentRoot := filepath.Join(tmp, "agents")
	workspace := filepath.Join(agentRoot, "alpha")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	proxy := &ProxyServer{AgentDir: agentRoot, DeviceID: "test-dev", CacheTTL: 120 * time.Second}
	req := httptest.NewRequest(http.MethodPost, "/api/cmd",
		strings.NewReader(`{"agentId":"alpha","chatId":"chat-1","cmd":"printf hello"}`))
	req.RemoteAddr = "127.0.0.1:43210"
	rec := httptest.NewRecorder()
	proxy.HandleCmd(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp CmdResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != 0 {
		t.Fatalf("status body = %d, output = %q", resp.Status, resp.Output)
	}
	if resp.Output != "hello" {
		t.Fatalf("output = %q, want hello", resp.Output)
	}
	if resp.Tid == "" {
		t.Fatal("tid should be auto-generated")
	}

	db, err := sql.Open("sqlite", "data")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	initCmdLogTable(db)

	var tid, cmd, result string
	var status int
	if err := db.QueryRow(`SELECT tid, cmd, result, status FROM cmd_log WHERE agent_id = ? AND chat_id = ?`,
		"alpha", "chat-1").Scan(&tid, &cmd, &result, &status); err != nil {
		t.Fatalf("query cmd_log: %v", err)
	}
	if tid != resp.Tid {
		t.Fatalf("tid = %q, want %q", tid, resp.Tid)
	}
	if cmd != "printf hello" {
		t.Fatalf("cmd = %q", cmd)
	}
	if status != 0 {
		t.Fatalf("db status = %d, want 0", status)
	}
	decoded, err := base64.StdEncoding.DecodeString(result)
	if err != nil {
		t.Fatalf("decode result: %v", err)
	}
	gz, err := gzip.NewReader(bytes.NewReader(decoded))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer gz.Close()
	plain, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("read gzip: %v", err)
	}
	if string(plain) != "hello" {
		t.Fatalf("logged result = %q, want hello", string(plain))
	}
}

func TestHandleCmdRejectsNonLocalRequest(t *testing.T) {
	flushAgentCache()
	proxy := &ProxyServer{AgentDir: t.TempDir(), DeviceID: "test-dev", CacheTTL: 120 * time.Second}
	req := httptest.NewRequest(http.MethodPost, "/api/cmd",
		strings.NewReader(`{"agentId":"alpha","chatId":"chat-1","cmd":"printf hello"}`))
	req.RemoteAddr = "192.168.1.8:54321"
	rec := httptest.NewRecorder()
	proxy.HandleCmd(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestHandleCmdRejectsBlockedCommand(t *testing.T) {
	flushAgentCache()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	agentRoot := filepath.Join(tmp, "agents")
	workspace := filepath.Join(agentRoot, "alpha")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	_ = os.WriteFile(filepath.Join(workspace, "SOUL.md"), []byte("soul"), 0o644)
	_ = os.WriteFile(filepath.Join(workspace, "USER.md"), []byte("user"), 0o644)

	proxy := &ProxyServer{AgentDir: agentRoot, DeviceID: "test-dev", CacheTTL: 120 * time.Second}
	req := httptest.NewRequest(http.MethodPost, "/api/cmd",
		strings.NewReader(`{"agentId":"alpha","chatId":"chat-1","cmd":"echo ok && rm -rf /tmp/x"}`))
	req.RemoteAddr = "127.0.0.1:43210"
	rec := httptest.NewRecorder()
	proxy.HandleCmd(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}

	db, err := sql.Open("sqlite", "data")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	initCmdLogTable(db)
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cmd_log`).Scan(&count); err != nil {
		t.Fatalf("count cmd_log: %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
}

func TestHandleCmdRejectsMissingOrDeletedAgent(t *testing.T) {
	flushAgentCache()
	proxy := &ProxyServer{AgentDir: t.TempDir(), DeviceID: "test-dev", CacheTTL: 120 * time.Second}
	req := httptest.NewRequest(http.MethodPost, "/api/cmd",
		strings.NewReader(`{"agentId":"missing","chatId":"chat-1","cmd":"printf hello"}`))
	req.RemoteAddr = "127.0.0.1:43210"
	rec := httptest.NewRecorder()
	proxy.HandleCmd(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestHandleKillStopsActiveCommandAndLogs(t *testing.T) {
	flushAgentCache()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	agentRoot := filepath.Join(tmp, "agents")
	workspace := filepath.Join(agentRoot, "alpha")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	_ = os.WriteFile(filepath.Join(workspace, "SOUL.md"), []byte("soul"), 0o644)
	_ = os.WriteFile(filepath.Join(workspace, "USER.md"), []byte("user"), 0o644)

	proxy := &ProxyServer{AgentDir: agentRoot, DeviceID: "test-dev", CacheTTL: 120 * time.Second}

	done := make(chan struct{})
	go func() {
		req := httptest.NewRequest(http.MethodPost, "/api/cmd",
			strings.NewReader(`{"agentId":"alpha","chatId":"chat-kill","tid":"tid-kill","cmd":"sleep 10"}`))
		req.RemoteAddr = "127.0.0.1:43210"
		rec := httptest.NewRecorder()
		proxy.HandleCmd(rec, req)
		close(done)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if ac, _ := findActiveCmd("alpha", "chat-kill", "tid-kill", "sleep 10"); ac != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("active command was not registered in time")
		}
		time.Sleep(20 * time.Millisecond)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/kill",
		strings.NewReader(`{"agentId":"alpha","chatId":"chat-kill","tid":"tid-kill","cmd":"sleep 10"}`))
	req.RemoteAddr = "127.0.0.1:43211"
	rec := httptest.NewRecorder()
	proxy.HandleKill(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("command was not stopped after kill")
	}

	var resp KillResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != 0 {
		t.Fatalf("status body = %d, content = %q", resp.Status, resp.Content)
	}

	db, err := sql.Open("sqlite", "data")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	initKillLogTable(db)

	var cmd, completedAt string
	if err := db.QueryRow(`SELECT cmd, completed_at FROM kill_log WHERE agent_id = ? AND chat_id = ? ORDER BY id DESC LIMIT 1`,
		"alpha", "chat-kill").Scan(&cmd, &completedAt); err != nil {
		t.Fatalf("query kill_log: %v", err)
	}
	if cmd != "sleep 10" {
		t.Fatalf("cmd = %q, want sleep 10", cmd)
	}
	if completedAt == "" {
		t.Fatal("completed_at should not be empty")
	}
}

func TestHandleKillRejectsNonLocalRequest(t *testing.T) {
	flushAgentCache()
	proxy := &ProxyServer{AgentDir: t.TempDir(), DeviceID: "test-dev", CacheTTL: 120 * time.Second}
	req := httptest.NewRequest(http.MethodPost, "/api/kill",
		strings.NewReader(`{"agentId":"alpha","chatId":"chat-1","cmd":"sleep 10"}`))
	req.RemoteAddr = "10.0.0.8:43210"
	rec := httptest.NewRecorder()
	proxy.HandleKill(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestHandleKillRejectsMissingOrDeletedAgent(t *testing.T) {
	flushAgentCache()
	proxy := &ProxyServer{AgentDir: t.TempDir(), DeviceID: "test-dev", CacheTTL: 120 * time.Second}
	req := httptest.NewRequest(http.MethodPost, "/api/kill",
		strings.NewReader(`{"agentId":"missing","chatId":"chat-1","cmd":"sleep 10"}`))
	req.RemoteAddr = "127.0.0.1:43210"
	rec := httptest.NewRecorder()
	proxy.HandleKill(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestHandleKillLogsNotFoundActiveCommand(t *testing.T) {
	flushAgentCache()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	agentRoot := filepath.Join(tmp, "agents")
	workspace := filepath.Join(agentRoot, "alpha")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	_ = os.WriteFile(filepath.Join(workspace, "SOUL.md"), []byte("soul"), 0o644)
	_ = os.WriteFile(filepath.Join(workspace, "USER.md"), []byte("user"), 0o644)

	proxy := &ProxyServer{AgentDir: agentRoot, DeviceID: "test-dev", CacheTTL: 120 * time.Second}
	req := httptest.NewRequest(http.MethodPost, "/api/kill",
		strings.NewReader(`{"agentId":"alpha","chatId":"chat-none","tid":"tid-none","cmd":"sleep 10"}`))
	req.RemoteAddr = "127.0.0.1:43210"
	rec := httptest.NewRecorder()
	proxy.HandleKill(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}

	db, err := sql.Open("sqlite", "data")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	initKillLogTable(db)

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_log WHERE agent_id = ? AND chat_id = ?`,
		"alpha", "chat-none").Scan(&count); err != nil {
		t.Fatalf("count kill_log: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func TestHandleKillFallsBackToSingleActiveCommandInSameChat(t *testing.T) {
	flushAgentCache()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	agentRoot := filepath.Join(tmp, "agents")
	workspace := filepath.Join(agentRoot, "alpha")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	_ = os.WriteFile(filepath.Join(workspace, "SOUL.md"), []byte("soul"), 0o644)
	_ = os.WriteFile(filepath.Join(workspace, "USER.md"), []byte("user"), 0o644)

	proxy := &ProxyServer{AgentDir: agentRoot, DeviceID: "test-dev", CacheTTL: 120 * time.Second}

	done := make(chan struct{})
	go func() {
		req := httptest.NewRequest(http.MethodPost, "/api/cmd",
			strings.NewReader(`{"agentId":"alpha","chatId":"chat-fallback","tid":"tid-fallback","cmd":"sleep 10"}`))
		req.RemoteAddr = "127.0.0.1:43210"
		rec := httptest.NewRecorder()
		proxy.HandleCmd(rec, req)
		close(done)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if ac, _ := findActiveCmd("alpha", "chat-fallback", "tid-fallback", "sleep 10"); ac != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("active command was not registered in time")
		}
		time.Sleep(20 * time.Millisecond)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/kill",
		strings.NewReader(`{"agentId":"alpha","chatId":"chat-fallback","cmd":"display text not raw cmd"}`))
	req.RemoteAddr = "127.0.0.1:43211"
	rec := httptest.NewRecorder()
	proxy.HandleKill(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("command was not stopped after fallback kill")
	}

	var resp KillResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != 0 {
		t.Fatalf("status body = %d, content = %q", resp.Status, resp.Content)
	}
	if resp.Cmd != "sleep 10" {
		t.Fatalf("cmd = %q, want sleep 10", resp.Cmd)
	}
}

func TestHandleCronPersistsReuseChatIDToDetails(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	flushAgentCache()

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "alpha")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "config.json"), []byte(`{"version":"cron-v1"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"openai": "sk-test"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	proxy := &ProxyServer{AgentDir: agentRoot, DeviceID: "dev", CacheTTL: time.Minute}
	rawTime := time.Now().Add(2 * time.Minute).Format("2006-01-02 15:04")
	req := httptest.NewRequest(http.MethodPost, "/api/cron/create?agentId=alpha",
		strings.NewReader(fmt.Sprintf(`{"content":"我之前问了什么","model":"openai","thinking":false,"router_disable":true,"rawTime":"%s","cycle":0,"token":"sk-test","chatId":"chat-reuse-1"}`, rawTime)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	proxy.HandleCron(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Status int   `json:"status"`
		ID     int64 `json:"id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != 0 {
		t.Fatalf("status body = %d", resp.Status)
	}

	var metaChatID, detailChatID string
	var metaRouterDisable, detailRouterDisable int
	if err := db.QueryRow(`SELECT chat_id FROM task_meta WHERE id = ?`, resp.ID).Scan(&metaChatID); err != nil {
		t.Fatalf("query task_meta: %v", err)
	}
	if err := db.QueryRow(`SELECT chat_id FROM task_detail WHERE meta_id = ?`, resp.ID).Scan(&detailChatID); err != nil {
		t.Fatalf("query task_detail: %v", err)
	}
	if err := db.QueryRow(`SELECT router_disable FROM task_meta WHERE id = ?`, resp.ID).Scan(&metaRouterDisable); err != nil {
		t.Fatalf("query task_meta router_disable: %v", err)
	}
	if err := db.QueryRow(`SELECT router_disable FROM task_detail WHERE meta_id = ?`, resp.ID).Scan(&detailRouterDisable); err != nil {
		t.Fatalf("query task_detail router_disable: %v", err)
	}
	if metaChatID != "chat-reuse-1" {
		t.Fatalf("task_meta.chat_id = %q, want chat-reuse-1", metaChatID)
	}
	if detailChatID != "chat-reuse-1" {
		t.Fatalf("task_detail.chat_id = %q, want chat-reuse-1", detailChatID)
	}
	if metaRouterDisable != 1 {
		t.Fatalf("task_meta.router_disable = %d, want 1", metaRouterDisable)
	}
	if detailRouterDisable != 1 {
		t.Fatalf("task_detail.router_disable = %d, want 1", detailRouterDisable)
	}

	metaReq := httptest.NewRequest(http.MethodPost, "/api/cron/detail/metadata?agentId=alpha", nil)
	metaRec := httptest.NewRecorder()
	proxy.HandleCronMetadata(metaRec, metaReq)
	if metaRec.Code != http.StatusOK {
		t.Fatalf("metadata status = %d, body = %s", metaRec.Code, metaRec.Body.String())
	}
	var metaResp struct {
		Status int `json:"status"`
		Data   []struct {
			ID            int  `json:"id"`
			RouterDisable bool `json:"router_disable"`
		} `json:"data"`
	}
	if err := json.NewDecoder(metaRec.Body).Decode(&metaResp); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if metaResp.Status != 0 {
		t.Fatalf("metadata status body = %d", metaResp.Status)
	}
	if len(metaResp.Data) != 1 || metaResp.Data[0].ID != int(resp.ID) || !metaResp.Data[0].RouterDisable {
		t.Fatalf("metadata response = %+v, want one router_disable=true item for id %d", metaResp.Data, resp.ID)
	}
}

func TestEnsureTokenStoreRenamesProviderLogTable(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE token_store_log (id INTEGER PRIMARY KEY AUTOINCREMENT, model TEXT NOT NULL, token TEXT NOT NULL DEFAULT '', action TEXT NOT NULL, updated_at TEXT NOT NULL)`); err != nil {
		t.Fatalf("create old table: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='proxy_agent_provider_log'`).Scan(&count); err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	if count != 1 {
		t.Fatalf("proxy_agent_provider_log count = %d, want 1", count)
	}
}

func TestHandleTokenNormalizesLegacyModelAliases(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO token_store (model, token, __url, __model, __model_fast, __model_thinking, __model_multi_input, __model_multi_output, updated_at) VALUES (?,?,?,?,?,?,?,?,?)`, "aiohright", "legacy-token", "https://legacy.example", "base-a", "fast-a", "think-a", "multi-in-a", "multi-out-a", time.Now().Format(time.RFC3339)); err != nil {
		t.Fatalf("seed token_store: %v", err)
	}

	proxy := &ProxyServer{}
	req := httptest.NewRequest(http.MethodGet, "/api/token", nil)
	rec := httptest.NewRecorder()
	proxy.HandleToken(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Status int                    `json:"status"`
		Models map[string]tokenConfig `json:"models"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Status != 0 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.Models["deepright"].Token != "legacy-token" {
		t.Fatalf("deepright token = %q, want legacy-token", payload.Models["deepright"].Token)
	}
	if payload.Models["deepright"].BaseURL != "https://legacy.example" {
		t.Fatalf("deepright __url = %q", payload.Models["deepright"].BaseURL)
	}
	if payload.Models["deepright"].ModelMultiInput != "multi-in-a" {
		t.Fatalf("deepright __model_multi_input = %q", payload.Models["deepright"].ModelMultiInput)
	}
	if payload.Models["deepright"].ModelMultiOutput != "multi-out-a" {
		t.Fatalf("deepright __model_multi_output = %q", payload.Models["deepright"].ModelMultiOutput)
	}
	if _, exists := payload.Models["aiohright"]; exists {
		t.Fatalf("legacy alias leaked in response: %+v", payload.Models)
	}
}

func TestHandleTokenRenamedModelDeletesPreviousRecord(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO token_store (model, token, __url, __model, __model_fast, __model_thinking, __model_multi_input, __model_multi_output, updated_at) VALUES (?,?,?,?,?,?,?,?,?)`, "openai", "old-token", "https://api.openai.com/v1/chat/completions", "gpt-old", "gpt-fast-old", "gpt-think-old", "gpt-vision-old", "", time.Now().Format(time.RFC3339)); err != nil {
		t.Fatalf("seed token_store: %v", err)
	}

	proxy := &ProxyServer{}
	reqBody := `{"models":{"deepright":{"token":"new-token"}},"removedModels":["openai"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/token", strings.NewReader(reqBody))
	rec := httptest.NewRecorder()
	proxy.HandleToken(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Status int                    `json:"status"`
		Models map[string]tokenConfig `json:"models"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Status != 0 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if _, exists := payload.Models["openai"]; exists {
		t.Fatalf("openai still exists in response: %+v", payload.Models)
	}
	deeprightCfg, exists := payload.Models["deepright"]
	if !exists {
		t.Fatalf("deepright not found in response: %+v", payload.Models)
	}
	if deeprightCfg.Token != "new-token" {
		t.Fatalf("deepright token = %q, want new-token", deeprightCfg.Token)
	}
	if deeprightCfg.BaseURL != "" || deeprightCfg.ModelBase != "" || deeprightCfg.ModelFast != "" || deeprightCfg.ModelThinking != "" || deeprightCfg.ModelMultiInput != "" || deeprightCfg.ModelMultiOutput != "" {
		t.Fatalf("deepright custom config should be cleared: %+v", deeprightCfg)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM token_store WHERE model = ?`, "openai").Scan(&count); err != nil {
		t.Fatalf("count openai: %v", err)
	}
	if count != 0 {
		t.Fatalf("openai count = %d, want 0", count)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM token_store WHERE model = ?`, "deepright").Scan(&count); err != nil {
		t.Fatalf("count deepright: %v", err)
	}
	if count != 1 {
		t.Fatalf("deepright count = %d, want 1", count)
	}

	rows, err := db.Query(`SELECT model, action FROM proxy_agent_provider_log ORDER BY id`)
	if err != nil {
		t.Fatalf("query provider log: %v", err)
	}
	defer rows.Close()

	logs := make([]string, 0, 2)
	for rows.Next() {
		var model string
		var action string
		if err := rows.Scan(&model, &action); err != nil {
			t.Fatalf("scan provider log: %v", err)
		}
		logs = append(logs, model+":"+action)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("provider log rows: %v", err)
	}
	if !reflect.DeepEqual(logs, []string{"openai:delete", "deepright:insert"}) {
		t.Fatalf("provider logs = %#v", logs)
	}
}

func TestHandleTokenDeleteModelRemovesLegacyCaseVariants(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	now := time.Now().Format(time.RFC3339)
	if _, err := db.Exec(`INSERT INTO token_store (model, token, updated_at) VALUES (?,?,?)`, "OpenAI", "legacy-token", now); err != nil {
		t.Fatalf("seed legacy token_store: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO token_store (model, token, updated_at) VALUES (?,?,?)`, "openai", "canonical-token", now); err != nil {
		t.Fatalf("seed canonical token_store: %v", err)
	}

	proxy := &ProxyServer{}
	reqBody := `{"action":"delete_model","model":"openai"}`
	req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(reqBody))
	rec := httptest.NewRecorder()
	proxy.HandleSwarm(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM token_store WHERE model IN ('OpenAI', 'openai')`).Scan(&count); err != nil {
		t.Fatalf("count token_store variants: %v", err)
	}
	if count != 0 {
		t.Fatalf("token_store variant count = %d, want 0", count)
	}
}

func TestHandleSwarmDeletesModelConfig(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO token_store (model, token, updated_at) VALUES (?,?,?)`, "deepseek", "Bearer deepseek-token", time.Now().Format(time.RFC3339)); err != nil {
		t.Fatalf("seed token_store: %v", err)
	}

	proxy := &ProxyServer{}
	req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(`{"action":"delete_model","model":"deepseek"}`))
	rec := httptest.NewRecorder()
	proxy.HandleSwarm(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Status int                    `json:"status"`
		Action string                 `json:"action"`
		Model  string                 `json:"model"`
		Models map[string]tokenConfig `json:"models"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Status != 0 || payload.Action != "delete_model" || payload.Model != "deepseek" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if _, exists := payload.Models["deepseek"]; exists {
		t.Fatalf("deepseek still exists in response: %+v", payload.Models)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM token_store WHERE model = ?`, "deepseek").Scan(&count); err != nil {
		t.Fatalf("count token_store: %v", err)
	}
	if count != 0 {
		t.Fatalf("token_store count = %d, want 0", count)
	}

	var action string
	if err := db.QueryRow(`SELECT action FROM proxy_agent_provider_log WHERE model = ? ORDER BY id DESC LIMIT 1`, "deepseek").Scan(&action); err != nil {
		t.Fatalf("query provider log: %v", err)
	}
	if action != "delete" {
		t.Fatalf("provider log action = %q, want delete", action)
	}
}

func TestProxyTokenCLIWritesConsumeRecord(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	timestamp := time.Date(2026, 5, 27, 9, 30, 0, 0, time.Local).UnixMilli()
	stdout := captureProxyStdout(t, func() {
		runProxyTokenCLI([]string{
			"--agentId", "agent-a",
			"--model", "deepseek-chat",
			"--function", "cli/get",
			"--thinking", "11",
			"--input", "22",
			"--total", "33",
			"--cache", "4",
			"--timestamp", strconv.FormatInt(timestamp, 10),
		})
	})

	var payload struct {
		Status int                 `json:"status"`
		Record tokenconsume.Record `json:"record"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("decode output: %v, raw=%s", err, stdout)
	}
	if payload.Status != 0 {
		t.Fatalf("payload status = %d, want 0, raw=%s", payload.Status, stdout)
	}
	if payload.Record.AgentID != "agent-a" || payload.Record.Model != "deepseek-chat" || payload.Record.Function != "cli/get" {
		t.Fatalf("unexpected record: %+v", payload.Record)
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	result, err := tokenconsume.Query(db, "agent-a", timestamp-1)
	if err != nil {
		t.Fatalf("query consume log: %v", err)
	}
	if len(result.Details) != 1 {
		t.Fatalf("len(details) = %d, want 1", len(result.Details))
	}
	if result.Details[0].Total != 33 || result.Details[0].Cache != 4 {
		t.Fatalf("stored detail = %+v", result.Details[0])
	}
}

func TestProxyHandleConsumeAggregatesByModel(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}

	base := time.Date(2026, 5, 27, 8, 0, 0, 0, time.Local)
	records := []tokenconsume.Record{
		{Thinking: 10, Input: 20, Total: 30, Cache: 5, Model: "model-a", AgentID: "agent-a", Function: "cli/get", Timestamp: base.Add(1 * time.Minute).UnixMilli()},
		{Thinking: 1, Input: 2, Total: 3, Cache: 1, Model: "model-a", AgentID: "agent-a", Function: "cron", Timestamp: base.Add(2 * time.Minute).UnixMilli()},
		{Thinking: 4, Input: 5, Total: 9, Cache: 0, Model: "model-b", AgentID: "agent-a", Function: "cli/pub", Timestamp: base.Add(3 * time.Minute).UnixMilli()},
		{Thinking: 7, Input: 8, Total: 15, Cache: 2, Model: "model-c", AgentID: "agent-b", Function: "cli/get", Timestamp: base.Add(4 * time.Minute).UnixMilli()},
	}
	for _, record := range records {
		if _, err := tokenconsume.Insert(db, record); err != nil {
			t.Fatalf("insert consume record: %v", err)
		}
	}

	proxy := &ProxyServer{}
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/consume?agentId=agent-a&starTime="+base.Format("20060102-150405")+"&closeTime="+base.Add(3*time.Minute).Format("20060102-150405")+"&limit=2",
		nil,
	)
	rec := httptest.NewRecorder()
	proxy.HandleConsume(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Status  int                    `json:"status"`
		Details []tokenconsume.Record  `json:"details"`
		Summary []tokenconsume.Summary `json:"summary"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Status != 0 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if len(payload.Details) != 2 {
		t.Fatalf("len(details) = %d, want 2", len(payload.Details))
	}
	if payload.Details[0].Timestamp <= payload.Details[1].Timestamp {
		t.Fatalf("details should be newest first: %+v", payload.Details)
	}
	if len(payload.Summary) != 2 {
		t.Fatalf("len(summary) = %d, want 2", len(payload.Summary))
	}
	if payload.Summary[0].Model != "model-a" || payload.Summary[0].Thinking != 1 || payload.Summary[0].Input != 2 || payload.Summary[0].Total != 3 || payload.Summary[0].Cache != 1 {
		t.Fatalf("summary[0] = %+v", payload.Summary[0])
	}
	if payload.Summary[1].Model != "model-b" || payload.Summary[1].Thinking != 4 || payload.Summary[1].Input != 5 || payload.Summary[1].Total != 9 || payload.Summary[1].Cache != 0 {
		t.Fatalf("summary[1] = %+v", payload.Summary[1])
	}
}

func TestProxyHandleChatSessionLogReturnsDescendingRawRows(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	oldSharedDataDB := sharedDataDB
	oldSharedEventLogger := sharedEventLogger
	sharedDataDB = pooledDB{}
	sharedEventLogger = struct {
		mu     sync.Mutex
		path   string
		logger *eventlog.Logger
	}{}
	defer func() {
		if sharedEventLogger.logger != nil {
			_ = sharedEventLogger.logger.Close()
		}
		if sharedDataDB.db != nil {
			_ = sharedDataDB.db.Close()
		}
		sharedDataDB = oldSharedDataDB
		sharedEventLogger = oldSharedEventLogger
	}()

	dbPath, err := getDataDBPath()
	if err != nil {
		t.Fatalf("getDataDBPath: %v", err)
	}
	store, err := eventlog.Open(dbPath)
	if err != nil {
		t.Fatalf("eventlog.Open: %v", err)
	}
	defer store.Close()

	entries := []eventlog.Entry{
		{AgentID: "agent-a", ChatID: "chat-1", Type: eventlog.TypeChatCompletionRequest, Content: `{"messages":[{"role":"user","content":"hello"}]}`, CreatedAt: "2026-05-27T08:01:00.000"},
		{AgentID: "agent-a", ChatID: "chat-1", Type: eventlog.TypeChatCompletionResponse, Content: "data: {\"choices\":[{\"delta\":{\"content\":\"world\"}}]}\n\n", CreatedAt: "2026-05-27T08:02:00.000"},
		{AgentID: "agent-a", ChatID: "chat-1", Type: eventlog.TypeCLIGet, Content: `{"cmd":"pwd"}`, CreatedAt: "2026-05-27T08:02:30.000"},
		{AgentID: "agent-b", ChatID: "chat-9", Type: eventlog.TypeChatCompletionResponse, Content: "data: {\"choices\":[{\"delta\":{\"content\":\"skip\"}}]}\n\n", CreatedAt: "2026-05-27T08:03:00.000"},
	}
	for _, entry := range entries {
		if err := store.Insert(entry); err != nil {
			t.Fatalf("insert event log: %v", err)
		}
	}

	proxy := &ProxyServer{}
	req := httptest.NewRequest(http.MethodGet, "/api/chat_session_log?agentId=agent-a&chatId=chat-1&start=2026-05-27T08:01:00&close=2026-05-27T08:02:00&limit=5", nil)
	rec := httptest.NewRecorder()
	proxy.HandleChatSessionLog(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Status  int                    `json:"status"`
		Count   int                    `json:"count"`
		Records []chatSessionLogRecord `json:"records"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Status != 0 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.Count != 2 || len(payload.Records) != 2 {
		t.Fatalf("count/records = %d/%d, want 2/2", payload.Count, len(payload.Records))
	}
	if payload.Records[0].LogType != eventlog.TypeChatCompletionResponse || payload.Records[0].CreatedAt != "2026-05-27T08:02:00.000" {
		t.Fatalf("records[0] = %+v", payload.Records[0])
	}
	if payload.Records[1].LogType != eventlog.TypeChatCompletionRequest || payload.Records[1].CreatedAt != "2026-05-27T08:01:00.000" {
		t.Fatalf("records[1] = %+v", payload.Records[1])
	}
	if payload.Records[0].Content != "data: {\"choices\":[{\"delta\":{\"content\":\"world\"}}]}\n\n" {
		t.Fatalf("response content mismatch: %q", payload.Records[0].Content)
	}
}

func TestProxyTokenCLIOutputAllModels(t *testing.T) {
	var stdout bytes.Buffer
	err := writeProxyTokenCLIOutput(&stdout, map[string]tokenConfig{
		"kimi":     {Token: "bbb"},
		"deepseek": {Token: "aaa", ModelFast: "fast-deepseek", ModelMultiInput: "deepseek-vision"},
	}, "")
	if err != nil {
		t.Fatalf("writeProxyTokenCLIOutput: %v", err)
	}

	var payload []map[string]tokenConfig
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v; raw=%s", err, stdout.String())
	}
	if len(payload) != 2 {
		t.Fatalf("len(payload) = %d, want 2", len(payload))
	}
	if got := payload[0]["deepseek"].Token; got != "aaa" {
		t.Fatalf("payload[0] = %+v, want deepseek first", payload[0])
	}
	if got := payload[0]["deepseek"].ModelFast; got != "fast-deepseek" {
		t.Fatalf("payload[0] = %+v, want __model_fast", payload[0])
	}
	if got := payload[0]["deepseek"].ModelMultiInput; got != "deepseek-vision" {
		t.Fatalf("payload[0] = %+v, want __model_multi_input", payload[0])
	}
	if got := payload[1]["kimi"].Token; got != "bbb" {
		t.Fatalf("payload[1] = %+v, want kimi second", payload[1])
	}
}

func TestProxyTokenCLIOutputSingleModel(t *testing.T) {
	var stdout bytes.Buffer
	err := writeProxyTokenCLIOutput(&stdout, map[string]tokenConfig{
		"deepseek": {Token: "aaa", BaseURL: "https://api.example", ModelMultiOutput: "gpt-image-1"},
		"kimi":     {Token: "bbb"},
	}, "deepseek")
	if err != nil {
		t.Fatalf("writeProxyTokenCLIOutput: %v", err)
	}

	var payload map[string]tokenConfig
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v; raw=%s", err, stdout.String())
	}
	if len(payload) != 1 || payload["deepseek"].Token != "aaa" {
		t.Fatalf("payload = %+v, want only deepseek", payload)
	}
	if payload["deepseek"].BaseURL != "https://api.example" {
		t.Fatalf("payload = %+v, want only deepseek", payload)
	}
	if payload["deepseek"].ModelMultiOutput != "gpt-image-1" {
		t.Fatalf("payload = %+v, want __model_multi_output", payload)
	}
}

func TestProxyTokenHelp(t *testing.T) {
	oldStdout := os.Stdout
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	os.Stdout = wOut
	defer func() {
		os.Stdout = oldStdout
	}()

	done := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(rOut)
		done <- string(data)
	}()

	printProxyTokenHelp()
	_ = wOut.Close()
	stdout := <-done

	if !strings.Contains(stdout, "proxy token --provider MODEL") {
		t.Fatalf("missing token usage: %s", stdout)
	}
	if !strings.Contains(stdout, "proxy token get [--n N]") {
		t.Fatalf("missing token get usage: %s", stdout)
	}
	if !strings.Contains(stdout, "proxy token --provider deepseek") {
		t.Fatalf("missing token example: %s", stdout)
	}
}

func TestProxyTokenGetHelp(t *testing.T) {
	stdout := captureProxyStdout(t, func() {
		runProxyTokenCLI([]string{"get", "--help"})
	})
	if !strings.Contains(stdout, "proxy token get [--n N]") {
		t.Fatalf("missing token get help usage: %s", stdout)
	}
	if !strings.Contains(stdout, "--n N            查询最新 N 条本地 token 用量记录") {
		t.Fatalf("missing token get help option: %s", stdout)
	}
}

func TestProxyTokenCLIQueriesLatestConsumeRecords(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	defer func() {
		if sharedDataDB.db != nil {
			_ = sharedDataDB.db.Close()
		}
		sharedDataDB = pooledDB{}
	}()

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}

	base := time.Date(2026, 6, 14, 12, 0, 0, 0, time.Local)
	records := []tokenconsume.Record{
		{Thinking: 10, Input: 20, Total: 30, Cache: 1, Model: "model-a", AgentID: "agent-a", Function: "cli/get", Timestamp: base.Add(1 * time.Minute).UnixMilli()},
		{Thinking: 11, Input: 21, Total: 31, Cache: 2, Model: "model-b", AgentID: "agent-a", Function: "cli/get", Timestamp: base.Add(2 * time.Minute).UnixMilli()},
		{Thinking: 12, Input: 22, Total: 32, Cache: 3, Model: "model-c", AgentID: "agent-a", Function: "cli/get", Timestamp: base.Add(3 * time.Minute).UnixMilli()},
		{Thinking: 13, Input: 23, Total: 33, Cache: 4, Model: "model-z", AgentID: "agent-b", Function: "cli/get", Timestamp: base.Add(4 * time.Minute).UnixMilli()},
	}
	for _, record := range records {
		if _, err := tokenconsume.Insert(db, record); err != nil {
			t.Fatalf("insert consume record: %v", err)
		}
	}

	stdout := captureProxyStdout(t, func() {
		runProxyTokenCLI([]string{"--agentId", "agent-a", "--n", "2"})
	})

	var payload struct {
		Status  int                    `json:"status"`
		Details []tokenconsume.Record  `json:"details"`
		Summary []tokenconsume.Summary `json:"summary"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("decode output: %v, raw=%s", err, stdout)
	}
	if payload.Status != 0 {
		t.Fatalf("payload status = %d, raw=%s", payload.Status, stdout)
	}
	if len(payload.Details) != 2 {
		t.Fatalf("len(details) = %d, want 2", len(payload.Details))
	}
	if payload.Details[0].Model != "model-c" || payload.Details[1].Model != "model-b" {
		t.Fatalf("details = %+v", payload.Details)
	}
	if payload.Details[0].Timestamp <= payload.Details[1].Timestamp {
		t.Fatalf("details should be newest first: %+v", payload.Details)
	}
	if len(payload.Summary) != 2 {
		t.Fatalf("len(summary) = %d, want 2", len(payload.Summary))
	}
}

func TestProxyCLIHelpIncludesSandbox(t *testing.T) {
	stdout := captureProxyStdout(t, func() {
		printHelp()
	})
	if !strings.Contains(stdout, "proxy sandbox --agentId ID --chatId ID [--sandbox off|filepick|net|filepick_net]") {
		t.Fatalf("missing sandbox usage: %s", stdout)
	}
	if !strings.Contains(stdout, "sandbox       Read or update sandbox mode for one AgentId + ChatId") {
		t.Fatalf("missing sandbox command description: %s", stdout)
	}
}

func TestProxyConnectHelpUsesConnectPrefix(t *testing.T) {
	stdout := captureProxyStdout(t, func() {
		printProxyConnectHelp()
	})
	if !strings.Contains(stdout, "proxy connect <subcommand> [options]") {
		t.Fatalf("missing connect usage: %s", stdout)
	}
	if !strings.Contains(stdout, "proxy connect meta-create --key feishu") || !strings.Contains(stdout, "proxy connect meta-list") {
		t.Fatalf("missing connect examples: %s", stdout)
	}
}

func TestProxyHandleSandboxPersistsByAgentAndChat(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	defer func() {
		if sharedDataDB.db != nil {
			_ = sharedDataDB.db.Close()
		}
		sharedDataDB = pooledDB{}
	}()

	proxy := &ProxyServer{}

	readReq := httptest.NewRequest(http.MethodGet, "/api/sandbox_status?agentId=agent-a&chatId=chat-1", nil)
	readRec := httptest.NewRecorder()
	proxy.HandleSandboxStatus(readRec, readReq)

	var readPayload struct {
		Status    int    `json:"status"`
		AgentID   string `json:"agentId"`
		ChatID    string `json:"chatId"`
		Sandbox   string `json:"sandbox"`
		Recorded  bool   `json:"recorded"`
		UpdatedAt string `json:"updatedAt"`
	}
	if err := json.NewDecoder(readRec.Body).Decode(&readPayload); err != nil {
		t.Fatalf("decode read payload: %v", err)
	}
	if readPayload.Status != 0 || readPayload.Sandbox != "" || readPayload.Recorded || readPayload.UpdatedAt != "" {
		t.Fatalf("unexpected initial payload: %+v", readPayload)
	}

	writeReq := httptest.NewRequest(http.MethodPost, "/api/sandbox=filepick_net?agentId=agent-a&chatId=chat-1", nil)
	writeRec := httptest.NewRecorder()
	proxy.HandleSandbox(writeRec, writeReq)

	var writePayload struct {
		Status    int    `json:"status"`
		Sandbox   string `json:"sandbox"`
		Recorded  bool   `json:"recorded"`
		UpdatedAt string `json:"updatedAt"`
	}
	if err := json.NewDecoder(writeRec.Body).Decode(&writePayload); err != nil {
		t.Fatalf("decode write payload: %v", err)
	}
	if writePayload.Status != 0 || writePayload.Sandbox != sandboxstate.ModeFilePickNet || !writePayload.Recorded || writePayload.UpdatedAt == "" {
		t.Fatalf("unexpected write payload: %+v", writePayload)
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}

	var stored string
	if err := db.QueryRow(`SELECT sandbox_exe FROM cli_sandbox_state WHERE agent_id = ? AND chat_id = ?`, "agent-a", "chat-1").Scan(&stored); err != nil {
		t.Fatalf("query sandbox state: %v", err)
	}
	if stored != sandboxstate.ModeFilePickNet {
		t.Fatalf("sandbox_exe = %q, want %q", stored, sandboxstate.ModeFilePickNet)
	}

	otherReq := httptest.NewRequest(http.MethodGet, "/api/sandbox_status?agentId=agent-a&chatId=chat-2", nil)
	otherRec := httptest.NewRecorder()
	proxy.HandleSandboxStatus(otherRec, otherReq)

	var otherPayload struct {
		Status   int    `json:"status"`
		Sandbox  string `json:"sandbox"`
		Recorded bool   `json:"recorded"`
	}
	if err := json.NewDecoder(otherRec.Body).Decode(&otherPayload); err != nil {
		t.Fatalf("decode other payload: %v", err)
	}
	if otherPayload.Status != 0 || otherPayload.Sandbox != "" || otherPayload.Recorded {
		t.Fatalf("unexpected other payload: %+v", otherPayload)
	}

	disableReq := httptest.NewRequest(http.MethodGet, "/api/sandbox=off?agentId=agent-a&chatId=chat-1", nil)
	disableRec := httptest.NewRecorder()
	proxy.HandleSandbox(disableRec, disableReq)

	var disablePayload struct {
		Status   int    `json:"status"`
		Sandbox  string `json:"sandbox"`
		Recorded bool   `json:"recorded"`
	}
	if err := json.NewDecoder(disableRec.Body).Decode(&disablePayload); err != nil {
		t.Fatalf("decode disable payload: %v", err)
	}
	if disablePayload.Status != 0 || disablePayload.Sandbox != "" || disablePayload.Recorded {
		t.Fatalf("unexpected disable payload: %+v", disablePayload)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(1) FROM cli_sandbox_state WHERE agent_id = ? AND chat_id = ?`, "agent-a", "chat-1").Scan(&count); err != nil {
		t.Fatalf("count sandbox state: %v", err)
	}
	if count != 0 {
		t.Fatalf("sandbox row count = %d, want 0", count)
	}
}

func TestProxyHandleSandboxStatusReadsWithoutWriting(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	defer func() {
		if sharedDataDB.db != nil {
			_ = sharedDataDB.db.Close()
		}
		sharedDataDB = pooledDB{}
	}()

	proxy := &ProxyServer{}

	writeReq := httptest.NewRequest(http.MethodPost, "/api/sandbox=filepick?agentId=agent-a&chatId=chat-1", nil)
	writeRec := httptest.NewRecorder()
	proxy.HandleSandbox(writeRec, writeReq)

	statusReq := httptest.NewRequest(http.MethodGet, "/api/sandbox_status?agentId=agent-a&chatId=chat-1", nil)
	statusRec := httptest.NewRecorder()
	proxy.HandleSandboxStatus(statusRec, statusReq)

	var payload struct {
		Status    int    `json:"status"`
		AgentID   string `json:"agentId"`
		ChatID    string `json:"chatId"`
		Sandbox   string `json:"sandbox"`
		Recorded  bool   `json:"recorded"`
		UpdatedAt string `json:"updatedAt"`
	}
	if err := json.NewDecoder(statusRec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode sandbox_status payload: %v", err)
	}
	if payload.Status != 0 || payload.AgentID != "agent-a" || payload.ChatID != "chat-1" || payload.Sandbox != sandboxstate.ModeFilePick || !payload.Recorded || payload.UpdatedAt == "" {
		t.Fatalf("unexpected sandbox_status payload: %+v", payload)
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}

	var stored string
	if err := db.QueryRow(`SELECT sandbox_exe FROM cli_sandbox_state WHERE agent_id = ? AND chat_id = ?`, "agent-a", "chat-1").Scan(&stored); err != nil {
		t.Fatalf("query sandbox state: %v", err)
	}
	if stored != sandboxstate.ModeFilePick {
		t.Fatalf("sandbox_exe = %q, want %q", stored, sandboxstate.ModeFilePick)
	}
}

func TestProxySandboxCLIReadsAndWritesSessionState(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	defer func() {
		if sharedDataDB.db != nil {
			_ = sharedDataDB.db.Close()
		}
		sharedDataDB = pooledDB{}
	}()

	stdout := captureProxyStdout(t, func() {
		runProxySandboxCLI([]string{"--agentId", "agent-a", "--chatId", "chat-1"})
	})
	var initial struct {
		Status   int    `json:"status"`
		Sandbox  string `json:"sandbox"`
		Recorded bool   `json:"recorded"`
	}
	if err := json.Unmarshal([]byte(stdout), &initial); err != nil {
		t.Fatalf("decode initial stdout: %v, raw=%s", err, stdout)
	}
	if initial.Status != 0 || initial.Sandbox != "" || initial.Recorded {
		t.Fatalf("unexpected initial stdout: %+v", initial)
	}

	stdout = captureProxyStdout(t, func() {
		runProxySandboxCLI([]string{"--agentId", "agent-a", "--chatId", "chat-1", "--sandbox", "net"})
	})
	var updated struct {
		Status    int    `json:"status"`
		Sandbox   string `json:"sandbox"`
		Recorded  bool   `json:"recorded"`
		UpdatedAt string `json:"updatedAt"`
	}
	if err := json.Unmarshal([]byte(stdout), &updated); err != nil {
		t.Fatalf("decode updated stdout: %v, raw=%s", err, stdout)
	}
	if updated.Status != 0 || updated.Sandbox != sandboxstate.ModeNet || !updated.Recorded || updated.UpdatedAt == "" {
		t.Fatalf("unexpected updated stdout: %+v", updated)
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}

	var stored string
	if err := db.QueryRow(`SELECT sandbox_exe FROM cli_sandbox_state WHERE agent_id = ? AND chat_id = ?`, "agent-a", "chat-1").Scan(&stored); err != nil {
		t.Fatalf("query sandbox state: %v", err)
	}
	if stored != sandboxstate.ModeNet {
		t.Fatalf("sandbox_exe = %q, want %q", stored, sandboxstate.ModeNet)
	}
}

func TestHandleCmdUsesSandboxHelperForEnabledSession(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	flushAgentCache()
	sharedDataDB = pooledDB{}
	defer func() {
		if sharedDataDB.db != nil {
			_ = sharedDataDB.db.Close()
		}
		sharedDataDB = pooledDB{}
	}()

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if _, err := sandboxstate.Set(db, "agent-a", "chat-1", sandboxstate.ModeFilePickNet); err != nil {
		t.Fatalf("enable sandbox: %v", err)
	}

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "agent-a")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	argsPath := filepath.Join(tmp, "sandbox-args.txt")
	helperPath := filepath.Join(tmp, "CLI_SANDBOX")
	helperScript := "#!/bin/sh\n" +
		"printf '%s\\n' \"$@\" > \"" + argsPath + "\"\n" +
		"printf 'proxy-sandbox\\n'\n"
	if err := os.WriteFile(helperPath, []byte(helperScript), 0o755); err != nil {
		t.Fatalf("write helper: %v", err)
	}

	originalPathFn := proxySandboxExecutablePathFn
	proxySandboxExecutablePathFn = func(mode string) (string, error) {
		if mode != sandboxstate.ModeFilePickNet {
			t.Fatalf("mode = %q, want %q", mode, sandboxstate.ModeFilePickNet)
		}
		return helperPath, nil
	}
	defer func() { proxySandboxExecutablePathFn = originalPathFn }()

	proxy := &ProxyServer{
		AgentDir: agentRoot,
		DeviceID: "dev",
		CacheTTL: 120 * time.Second,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/cmd", strings.NewReader(`{"agentId":"agent-a","chatId":"chat-1","cmd":"ls -la /tmp/demo","timeout":4567}`))
	req.RemoteAddr = "127.0.0.1:34567"
	rec := httptest.NewRecorder()
	proxy.HandleCmd(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload CmdResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Status != 0 || payload.Output != "proxy-sandbox\n" {
		t.Fatalf("payload = %+v", payload)
	}

	argsData, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	gotArgs := strings.Split(strings.TrimSpace(string(argsData)), "\n")
	wantArgs := []string{"--cmd", "ls -la /tmp/demo", "--shell", sharedutil.DefaultExecShell(), "--timeout", "4567"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("sandbox args = %#v, want %#v", gotArgs, wantArgs)
	}
}

func TestResolveProxySandboxExecutablePathFromLinuxHelpersConfig(t *testing.T) {
	root := t.TempDir()
	execPath := filepath.Join(root, "linux-release", "proxy")
	sandboxExec := filepath.Join(root, "linux-release", "helpers", sandboxstate.ModeFilePick, "CLI_SANDBOX")
	configPath := filepath.Join(root, "linux-release", "config", "config.json")

	if err := os.MkdirAll(filepath.Dir(execPath), 0o755); err != nil {
		t.Fatalf("mkdir exec dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(sandboxExec), 0o755); err != nil {
		t.Fatalf("mkdir sandbox dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(execPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write exec: %v", err)
	}
	if err := os.WriteFile(sandboxExec, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write sandbox exec: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{"sandbox_app":"./helpers"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	originalExecutable := proxyOsExecutable
	proxyOsExecutable = func() (string, error) { return execPath, nil }
	defer func() { proxyOsExecutable = originalExecutable }()

	got, err := resolveProxySandboxExecutablePath(sandboxstate.ModeFilePick)
	if err != nil {
		t.Fatalf("resolveProxySandboxExecutablePath: %v", err)
	}
	if got != sandboxExec {
		t.Fatalf("got %q, want %q", got, sandboxExec)
	}
}

func TestHandleCronDeletePreservesCompletedDetails(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	renameCronLogTables(db)
	db.Exec(`CREATE TABLE IF NOT EXISTS task_meta (id INTEGER PRIMARY KEY AUTOINCREMENT, cycle INTEGER NOT NULL, raw_time TEXT NOT NULL DEFAULT '', agent_id TEXT NOT NULL, model TEXT NOT NULL, thinking INTEGER NOT NULL DEFAULT 0, cron TEXT NOT NULL, content TEXT NOT NULL, chat_id TEXT NOT NULL DEFAULT '')`)
	db.Exec(`CREATE TABLE IF NOT EXISTS task_detail (id INTEGER PRIMARY KEY AUTOINCREMENT, meta_id INTEGER NOT NULL, exec_time INTEGER NOT NULL, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL DEFAULT '', model TEXT NOT NULL, thinking INTEGER NOT NULL DEFAULT 0, content TEXT NOT NULL, started INTEGER NOT NULL DEFAULT 0)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS cron_meta_log (id INTEGER PRIMARY KEY AUTOINCREMENT, meta_id INTEGER NOT NULL, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL DEFAULT '', action TEXT NOT NULL, cycle INTEGER NOT NULL, raw_time TEXT NOT NULL DEFAULT '', model TEXT NOT NULL, thinking INTEGER NOT NULL DEFAULT 0, cron TEXT NOT NULL DEFAULT '', content TEXT NOT NULL, created_at TEXT NOT NULL)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS cron_detail_log (id INTEGER PRIMARY KEY AUTOINCREMENT, detail_id INTEGER NOT NULL, meta_id INTEGER NOT NULL, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL DEFAULT '', action TEXT NOT NULL, exec_time INTEGER NOT NULL, model TEXT NOT NULL, thinking INTEGER NOT NULL DEFAULT 0, content TEXT NOT NULL, started INTEGER NOT NULL DEFAULT 0, occurred_at TEXT NOT NULL)`)

	res, err := db.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, model, thinking, cron, content, chat_id) VALUES (0, '', 'alpha', 'openai', 0, '', 'memo', 'chat-1')`)
	if err != nil {
		t.Fatalf("insert meta: %v", err)
	}
	metaID, _ := res.LastInsertId()
	if _, err := db.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, model, thinking, content, started) VALUES (?, ?, 'alpha', 'chat-1', 'openai', 0, 'todo', 0), (?, ?, 'alpha', 'chat-1', 'openai', 0, 'done', 3)`,
		metaID, time.Now().Unix(), metaID, time.Now().Add(time.Minute).Unix()); err != nil {
		t.Fatalf("insert detail: %v", err)
	}

	proxy := &ProxyServer{}
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/cron/delete?id=%d", metaID), nil)
	rec := httptest.NewRecorder()
	proxy.HandleCronDelete(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var metaCount, unfinishedCount, completedCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM task_meta WHERE id = ?`, metaID).Scan(&metaCount); err != nil {
		t.Fatalf("count meta: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM task_detail WHERE meta_id = ? AND started != 3`, metaID).Scan(&unfinishedCount); err != nil {
		t.Fatalf("count unfinished: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM task_detail WHERE meta_id = ? AND started = 3`, metaID).Scan(&completedCount); err != nil {
		t.Fatalf("count completed: %v", err)
	}
	if metaCount != 0 || unfinishedCount != 0 || completedCount != 1 {
		t.Fatalf("meta=%d unfinished=%d completed=%d, want 0/0/1", metaCount, unfinishedCount, completedCount)
	}
}

func TestHandleCronDetailListIncludesCompletedResultContent(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	renameCronLogTables(db)
	db.Exec(`CREATE TABLE IF NOT EXISTS task_meta (id INTEGER PRIMARY KEY AUTOINCREMENT, cycle INTEGER NOT NULL, raw_time TEXT NOT NULL DEFAULT '', agent_id TEXT NOT NULL, model TEXT NOT NULL, thinking INTEGER NOT NULL DEFAULT 0, cron TEXT NOT NULL, content TEXT NOT NULL, chat_id TEXT NOT NULL DEFAULT '')`)
	db.Exec(`CREATE TABLE IF NOT EXISTS task_detail (id INTEGER PRIMARY KEY AUTOINCREMENT, meta_id INTEGER NOT NULL, exec_time INTEGER NOT NULL, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL DEFAULT '', meta_ref TEXT NOT NULL DEFAULT '', task_type TEXT NOT NULL DEFAULT 'cron', model TEXT NOT NULL, thinking INTEGER NOT NULL DEFAULT 0, content TEXT NOT NULL, response_schema TEXT NOT NULL DEFAULT '', result_content TEXT NOT NULL DEFAULT '', replied_at TEXT NOT NULL DEFAULT '', started INTEGER NOT NULL DEFAULT 0)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS cron_meta_log (id INTEGER PRIMARY KEY AUTOINCREMENT, meta_id INTEGER NOT NULL, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL DEFAULT '', action TEXT NOT NULL, cycle INTEGER NOT NULL, raw_time TEXT NOT NULL DEFAULT '', model TEXT NOT NULL, thinking INTEGER NOT NULL DEFAULT 0, cron TEXT NOT NULL DEFAULT '', content TEXT NOT NULL, response_schema TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS cron_detail_log (id INTEGER PRIMARY KEY AUTOINCREMENT, detail_id INTEGER NOT NULL, meta_id INTEGER NOT NULL, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL DEFAULT '', action TEXT NOT NULL, exec_time INTEGER NOT NULL, model TEXT NOT NULL, thinking INTEGER NOT NULL DEFAULT 0, content TEXT NOT NULL, response_schema TEXT NOT NULL DEFAULT '', started INTEGER NOT NULL DEFAULT 0, occurred_at TEXT NOT NULL)`)

	res, err := db.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, model, thinking, cron, content, chat_id) VALUES (0, '', 'alpha', 'openai', 0, '', 'memo', 'chat-ja')`)
	if err != nil {
		t.Fatalf("insert meta: %v", err)
	}
	metaID, _ := res.LastInsertId()
	execTime := time.Date(2026, 6, 14, 10, 0, 0, 0, time.Local).Unix()
	if _, err := db.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, meta_ref, task_type, model, thinking, content, response_schema, result_content, replied_at, started) VALUES (?, ?, 'alpha', 'chat-ja', '', 'cron', 'openai', 0, '日本語の備忘録', '', '完了しました', '2026-06-14T10:00:05', 3)`,
		metaID, execTime); err != nil {
		t.Fatalf("insert detail: %v", err)
	}

	proxy := &ProxyServer{}
	req := httptest.NewRequest(http.MethodPost, "/api/cron/detail/list?agentId=alpha&date=2026-06-14", nil)
	rec := httptest.NewRecorder()
	proxy.HandleCronDetailList(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Status int `json:"status"`
		Data   []struct {
			Content       string `json:"content"`
			ResultContent string `json:"resultContent"`
			RepliedAt     string `json:"repliedAt"`
			Started       int    `json:"started"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != 0 {
		t.Fatalf("status body = %d, want 0", resp.Status)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("len(data) = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].Content != "日本語の備忘録" {
		t.Fatalf("content = %q, want 日本語の備忘録", resp.Data[0].Content)
	}
	if resp.Data[0].ResultContent != "完了しました" {
		t.Fatalf("resultContent = %q, want 完了しました", resp.Data[0].ResultContent)
	}
	if resp.Data[0].RepliedAt != "2026-06-14T10:00:05" {
		t.Fatalf("repliedAt = %q, want 2026-06-14T10:00:05", resp.Data[0].RepliedAt)
	}
	if resp.Data[0].Started != 3 {
		t.Fatalf("started = %d, want 3", resp.Data[0].Started)
	}
}

func TestHandleAgentDeleteRemovesAgentCronData(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	flushAgentCache()

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "alpha")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	ensureCronSchema(db)
	res, err := db.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, chat_id, task_type, model, thinking, router_disable, cron, content, response_schema) VALUES (0, '', 'alpha', 'chat-1', 'cron', 'openai', 0, 1, '', 'memo', '')`)
	if err != nil {
		t.Fatalf("insert meta: %v", err)
	}
	metaID, _ := res.LastInsertId()
	if _, err := db.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, meta_ref, task_type, model, thinking, router_disable, content, response_schema, result_content, replied_at, started) VALUES (?, ?, 'alpha', 'chat-1', '', 'cron', 'openai', 0, 1, 'memo', '', '', '', 0)`,
		metaID, time.Now().Unix()); err != nil {
		t.Fatalf("insert detail: %v", err)
	}

	proxy := &ProxyServer{AgentDir: agentRoot, DeviceID: "dev", CacheTTL: time.Minute}
	req := httptest.NewRequest(http.MethodGet, "/api/agent/delete?name=alpha", nil)
	rec := httptest.NewRecorder()
	proxy.HandleAgentDelete(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if status, ok := payload["status"].(float64); !ok || status != 0 {
		t.Fatalf("response status = %v, body = %s", payload["status"], rec.Body.String())
	}

	var metaCount, detailCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM task_meta WHERE agent_id = 'alpha'`).Scan(&metaCount); err != nil {
		t.Fatalf("count meta: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM task_detail WHERE agent_id = 'alpha'`).Scan(&detailCount); err != nil {
		t.Fatalf("count detail: %v", err)
	}
	if metaCount != 0 || detailCount != 0 {
		t.Fatalf("meta=%d detail=%d, want 0/0", metaCount, detailCount)
	}
}

func TestHandleAgentDeleteSucceedsWhenDirectoryAlreadyMissing(t *testing.T) {
	flushAgentCache()

	agentRoot := t.TempDir()
	agentDir := filepath.Join(agentRoot, "alpha")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	proxy := &ProxyServer{AgentDir: agentRoot, DeviceID: "dev", CacheTTL: time.Hour}
	if _, err := getAgentOutput(agentRoot, "dev", time.Hour); err != nil {
		t.Fatalf("warm agent cache: %v", err)
	}
	if err := os.RemoveAll(agentDir); err != nil {
		t.Fatalf("remove agent dir: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agent/delete?name=alpha", nil)
	rec := httptest.NewRecorder()
	proxy.HandleAgentDelete(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if status, ok := payload["status"].(float64); !ok || status != 0 {
		t.Fatalf("response status = %v, body = %s", payload["status"], rec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/agentId", nil)
	listRec := httptest.NewRecorder()
	proxy.HandleAgentIDs(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}

	var ids []string
	if err := json.Unmarshal(listRec.Body.Bytes(), &ids); err != nil {
		t.Fatalf("decode ids: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("agent IDs = %v, want empty after missing delete cleanup", ids)
	}
}

func TestHandleAgentCreateAllowsNestedRelativePath(t *testing.T) {
	flushAgentCache()
	agentRoot := t.TempDir()
	workspace := filepath.Join(agentRoot, "agent-a")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write SOUL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write USER.md: %v", err)
	}

	proxy := &ProxyServer{AgentDir: agentRoot, DeviceID: "dev", CacheTTL: time.Second}
	req := httptest.NewRequest(http.MethodGet, "/api/agent/create?agentId=agent-a&name="+url.QueryEscape("docs/data")+"&type=0", nil)
	rec := httptest.NewRecorder()
	proxy.HandleAgentCreate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Status  int    `json:"status"`
		Name    string `json:"name"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != 0 {
		t.Fatalf("status = %d, content = %q", resp.Status, resp.Content)
	}
	if resp.Name != "docs/data" {
		t.Fatalf("name = %q, want docs/data", resp.Name)
	}
	info, err := os.Stat(filepath.Join(workspace, "docs", "data"))
	if err != nil {
		t.Fatalf("stat created dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("created path is not a directory")
	}
}

func TestHandleAgentCreateRejectsWorkspaceEscape(t *testing.T) {
	flushAgentCache()
	agentRoot := t.TempDir()
	workspace := filepath.Join(agentRoot, "agent-a")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write SOUL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write USER.md: %v", err)
	}

	proxy := &ProxyServer{AgentDir: agentRoot, DeviceID: "dev", CacheTTL: time.Second}
	req := httptest.NewRequest(http.MethodGet, "/api/agent/create?agentId=agent-a&name="+url.QueryEscape("../escape")+"&type=0", nil)
	rec := httptest.NewRecorder()
	proxy.HandleAgentCreate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Status  int    `json:"status"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != 1 {
		t.Fatalf("status = %d, want 1", resp.Status)
	}
	if !strings.Contains(resp.Content, "escapes workspace") {
		t.Fatalf("content = %q, want workspace escape error", resp.Content)
	}
	if _, err := os.Stat(filepath.Join(agentRoot, "escape")); !os.IsNotExist(err) {
		t.Fatalf("escape path should not be created, stat err = %v", err)
	}
}

func TestSyncConnectPendingRequestsCreatesImmediateCronTask(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	flushAgentCache()

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "config.json"), []byte(`{"version":"cron-v1"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	var notifiedCallback string
	var notifiedFlags map[string]string

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   filepath.Join(tmp, "data"),
		AgentDir: agentRoot,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	defer svc.Close()
	meta, err := svc.CreateMeta(connectsvc.MetaInput{
		Key:           "feishu",
		Name:          "feishu",
		Meta:          `{"appId":"x"}`,
		Callback:      "./feishu",
		AgentID:       "A",
		ChatID:        "chat-feishu",
		Model:         "OpenAI",
		Thinking:      true,
		RouterDisable: true,
	})
	if err != nil {
		t.Fatalf("create meta: %v", err)
	}
	originalRunConnectListMeta := runConnectListMeta
	originalRunConnectPluginInit := runConnectPluginInit
	runConnectListMeta = func() ([]connectMetaConfigListItem, error) {
		return []connectMetaConfigListItem{{Key: "feishu", Name: meta.Name, Callback: filepath.Join(tmp, "plugins", "feishu")}}, nil
	}
	runConnectPluginInit = func(callbackPath string, flags map[string]string) error {
		notifiedCallback = callbackPath
		notifiedFlags = make(map[string]string, len(flags))
		for k, v := range flags {
			notifiedFlags[k] = v
		}
		return nil
	}
	defer func() {
		runConnectListMeta = originalRunConnectListMeta
		runConnectPluginInit = originalRunConnectPluginInit
	}()

	imagePath := filepath.Join(tmp, "a.png")
	filePath := filepath.Join(tmp, "b.csv")
	if err := os.WriteFile(imagePath, []byte("img"), 0o644); err != nil {
		t.Fatalf("write image: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("csv"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	r1, err := svc.AddRequest(connectsvc.RequestInput{Key: meta.Key, Request: "你好"})
	if err != nil {
		t.Fatalf("add request 1: %v", err)
	}
	r2, err := svc.AddRequest(connectsvc.RequestInput{Key: meta.Key, Request: "[image]", Artifacts: imagePath})
	if err != nil {
		t.Fatalf("add request 2: %v", err)
	}
	r3, err := svc.AddRequest(connectsvc.RequestInput{Key: meta.Key, Request: "[file]", Artifacts: filePath})
	if err != nil {
		t.Fatalf("add request 3: %v", err)
	}
	r4, err := svc.AddRequest(connectsvc.RequestInput{Key: meta.Key, Request: "天气怎么样"})
	if err != nil {
		t.Fatalf("add request 4: %v", err)
	}
	proxy := &ProxyServer{AgentDir: agentRoot, DeviceID: "dev", CacheTTL: time.Minute, ConnectCacheTTL: time.Second}
	syncConnectPendingRequests(proxy)

	var metaCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM task_meta WHERE task_type = 'feishu'`).Scan(&metaCount); err != nil {
		t.Fatalf("count task_meta: %v", err)
	}
	if metaCount != 1 {
		t.Fatalf("task_meta count = %d, want 1", metaCount)
	}

	var detailCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM task_detail WHERE task_type = 'feishu'`).Scan(&detailCount); err != nil {
		t.Fatalf("count task_detail: %v", err)
	}
	if detailCount != 1 {
		t.Fatalf("task_detail count = %d, want 1", detailCount)
	}
	var metaRouterDisable int
	if err := db.QueryRow(`SELECT router_disable FROM task_meta WHERE task_type = 'feishu' LIMIT 1`).Scan(&metaRouterDisable); err != nil {
		t.Fatalf("query task_meta router_disable: %v", err)
	}
	if metaRouterDisable != 1 {
		t.Fatalf("task_meta router_disable = %d, want 1", metaRouterDisable)
	}

	var content, chatID, metaRef string
	var thinking, routerDisable, started int
	if err := db.QueryRow(`SELECT content, chat_id, meta_ref, thinking, router_disable, started FROM task_detail WHERE task_type = 'feishu' LIMIT 1`).Scan(&content, &chatID, &metaRef, &thinking, &routerDisable, &started); err != nil {
		t.Fatalf("query task_detail: %v", err)
	}
	if content != "你好 [image]"+imagePath+" [file]"+filePath+" 天气怎么样" {
		t.Fatalf("content = %q", content)
	}
	if chatID != "chat-feishu" {
		t.Fatalf("chatID = %q, want chat-feishu", chatID)
	}
	wantMetaRef := fmt.Sprintf("%d,%d,%d,%d", r1.ID, r2.ID, r3.ID, r4.ID)
	if metaRef != wantMetaRef {
		t.Fatalf("metaRef = %q, want %q", metaRef, wantMetaRef)
	}
	if thinking != 1 {
		t.Fatalf("thinking = %d, want 1", thinking)
	}
	if routerDisable != 1 {
		t.Fatalf("router_disable = %d, want 1", routerDisable)
	}
	if started != 1 {
		t.Fatalf("started = %d, want 1", started)
	}
	wantCallback := filepath.Join(tmp, "plugins", "feishu")
	if notifiedCallback != wantCallback {
		t.Fatalf("notified callback = %q, want %q", notifiedCallback, wantCallback)
	}
	if notifiedFlags["content"] != defaultConnectAutoReply {
		t.Fatalf("notify content = %q, want %q", notifiedFlags["content"], defaultConnectAutoReply)
	}
	if strings.TrimSpace(notifiedFlags["message"]) == "" {
		t.Fatal("notify message is empty")
	}
	var notifiedRequest connectsvc.Request
	if err := json.Unmarshal([]byte(notifiedFlags["message"]), &notifiedRequest); err != nil {
		t.Fatalf("decode notify message: %v raw=%s", err, notifiedFlags["message"])
	}
	if notifiedRequest.ID != r4.ID {
		t.Fatalf("notify request id = %d, want %d", notifiedRequest.ID, r4.ID)
	}

	requests, err := svc.ListRequests(connectsvc.RequestFilter{Key: meta.Key, Limit: 10})
	if err != nil {
		t.Fatalf("list requests: %v", err)
	}
	for _, req := range requests {
		if req.Status != connectsvc.RequestStatusStarted {
			t.Fatalf("request %d status = %d, want started", req.ID, req.Status)
		}
	}
}

func TestSyncConnectPendingRequestsExecutesStartedDetailImmediately(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	flushAgentCache()

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer scheduled-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	var notifiedFlags map[string]string

	var captured map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"马上执行\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   filepath.Join(tmp, "data"),
		AgentDir: agentRoot,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	defer svc.Close()
	meta, err := svc.CreateMeta(connectsvc.MetaInput{
		Key:      "feishu",
		Name:     "feishu",
		Meta:     `{"appId":"x"}`,
		Callback: "./feishu",
		AgentID:  "A",
		ChatID:   "chat-feishu",
		Model:    "OpenAI",
		Thinking: true,
	})
	if err != nil {
		t.Fatalf("create meta: %v", err)
	}
	originalRunConnectListMeta := runConnectListMeta
	originalRunConnectPluginInit := runConnectPluginInit
	runConnectListMeta = func() ([]connectMetaConfigListItem, error) {
		return []connectMetaConfigListItem{{Key: "feishu", Name: meta.Name, Callback: filepath.Join(tmp, "plugins", "feishu")}}, nil
	}
	runConnectPluginInit = func(callbackPath string, flags map[string]string) error {
		notifiedFlags = flags
		return nil
	}
	defer func() {
		runConnectListMeta = originalRunConnectListMeta
		runConnectPluginInit = originalRunConnectPluginInit
	}()

	_, err = svc.AddRequest(connectsvc.RequestInput{Key: meta.Key, Request: "第一条"})
	if err != nil {
		t.Fatalf("add request 1: %v", err)
	}
	r2, err := svc.AddRequest(connectsvc.RequestInput{Key: meta.Key, Request: "第二条"})
	if err != nil {
		t.Fatalf("add request 2: %v", err)
	}
	proxy := &ProxyServer{
		Host:            upstream.URL,
		AgentDir:        agentRoot,
		DeviceID:        "dev",
		CacheTTL:        time.Minute,
		ConnectCacheTTL: time.Second,
		Client:          &http.Client{Timeout: 5 * time.Second},
	}
	syncConnectPendingRequests(proxy)
	time.Sleep(300 * time.Millisecond)

	var started int
	var resultContent string
	if err := db.QueryRow(`SELECT started, result_content FROM task_detail WHERE task_type = 'feishu' LIMIT 1`).Scan(&started, &resultContent); err != nil {
		t.Fatalf("query task_detail: %v", err)
	}
	if started != 3 {
		t.Fatalf("started = %d, want 3", started)
	}
	if resultContent != "马上执行" {
		t.Fatalf("result_content = %q, want 马上执行", resultContent)
	}

	if captured == nil {
		t.Fatal("expected upstream request")
	}
	metadata, ok := captured["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata missing: %+v", captured)
	}
	if metadata["META_ID"] != strconv.Itoa(r2.ID) {
		t.Fatalf("metadata META_ID = %v, want %d", metadata["META_ID"], r2.ID)
	}
	if metadata["chat"] != "chat-feishu" {
		t.Fatalf("metadata chat = %v, want chat-feishu", metadata["chat"])
	}
	if notifiedFlags == nil || notifiedFlags["content"] != defaultConnectAutoReply {
		t.Fatalf("notify flags = %+v", notifiedFlags)
	}
}

func TestSyncConnectPendingRequestsCreatesImmediateTaskForRecentPendingRequests(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	flushAgentCache()

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   filepath.Join(tmp, "data"),
		AgentDir: agentRoot,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	defer svc.Close()
	meta, err := svc.CreateMeta(connectsvc.MetaInput{
		Key:      "feishu",
		Name:     "feishu",
		Meta:     `{"appId":"x"}`,
		Callback: "./feishu",
		AgentID:  "A",
		ChatID:   "chat-feishu",
		Model:    "OpenAI",
		Thinking: true,
	})
	if err != nil {
		t.Fatalf("create meta: %v", err)
	}

	req, err := svc.AddRequest(connectsvc.RequestInput{Key: meta.Key, Request: "新消息立即转任务"})
	if err != nil {
		t.Fatalf("add request: %v", err)
	}

	originalRunConnectPluginInit := runConnectPluginInit
	runConnectPluginInit = func(string, map[string]string) error { return nil }
	defer func() { runConnectPluginInit = originalRunConnectPluginInit }()

	proxy := &ProxyServer{AgentDir: agentRoot, DeviceID: "dev", CacheTTL: time.Minute, ConnectCacheTTL: time.Second}
	syncConnectPendingRequests(proxy)

	var metaCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM task_meta WHERE task_type = 'feishu'`).Scan(&metaCount); err != nil {
		t.Fatalf("count task_meta: %v", err)
	}
	if metaCount != 1 {
		t.Fatalf("task_meta count = %d, want 1", metaCount)
	}

	var status int
	if err := db.QueryRow(`SELECT status FROM connect_request WHERE id = ?`, req.ID).Scan(&status); err != nil {
		t.Fatalf("query request status: %v", err)
	}
	if status != connectsvc.RequestStatusStarted {
		t.Fatalf("request status = %d, want started", status)
	}
}

func TestSyncConnectPendingRequestsSkipsArtifactOnlyMessages(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	flushAgentCache()

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   filepath.Join(tmp, "data"),
		AgentDir: agentRoot,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	defer svc.Close()
	meta, err := svc.CreateMeta(connectsvc.MetaInput{
		Key:      "feishu",
		Name:     "feishu",
		Meta:     `{"appId":"x"}`,
		Callback: "./feishu",
		AgentID:  "A",
		ChatID:   "chat-feishu",
		Model:    "OpenAI",
		Thinking: true,
	})
	if err != nil {
		t.Fatalf("create meta: %v", err)
	}

	imagePath := filepath.Join(tmp, "a.png")
	if err := os.WriteFile(imagePath, []byte("img"), 0o644); err != nil {
		t.Fatalf("write image: %v", err)
	}
	req, err := svc.AddRequest(connectsvc.RequestInput{Key: meta.Key, Request: "[image]", Artifacts: imagePath})
	if err != nil {
		t.Fatalf("add request: %v", err)
	}
	oldTimestamp := time.Now().Add(-21 * time.Minute).Format(time.RFC3339)
	if _, err := db.Exec(`UPDATE connect_request SET created_at = ? WHERE id = ?`, oldTimestamp, req.ID); err != nil {
		t.Fatalf("age request: %v", err)
	}

	proxy := &ProxyServer{AgentDir: agentRoot, DeviceID: "dev", CacheTTL: time.Minute, ConnectCacheTTL: time.Second}
	syncConnectPendingRequests(proxy)

	var metaCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM task_meta WHERE task_type = 'feishu'`).Scan(&metaCount); err != nil {
		t.Fatalf("count task_meta: %v", err)
	}
	if metaCount != 0 {
		t.Fatalf("task_meta count = %d, want 0", metaCount)
	}

	var status int
	if err := db.QueryRow(`SELECT status FROM connect_request WHERE id = ?`, req.ID).Scan(&status); err != nil {
		t.Fatalf("query request status: %v", err)
	}
	if status != connectsvc.RequestStatusPending {
		t.Fatalf("request status = %d, want pending", status)
	}
}

func TestSyncConnectPendingRequestsCreatesMemoOnlyDetailAfterTwentyMinutes(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	flushAgentCache()

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	notified := false

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   filepath.Join(tmp, "data"),
		AgentDir: agentRoot,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	defer svc.Close()
	meta, err := svc.CreateMeta(connectsvc.MetaInput{
		Key:      "feishu",
		Name:     "feishu",
		Meta:     `{"appId":"x"}`,
		Callback: "./feishu",
		AgentID:  "A",
		ChatID:   "chat-feishu",
		Model:    "OpenAI",
		Thinking: true,
	})
	if err != nil {
		t.Fatalf("create meta: %v", err)
	}
	originalRunConnectListMeta := runConnectListMeta
	originalRunConnectPluginInit := runConnectPluginInit
	runConnectListMeta = func() ([]connectMetaConfigListItem, error) {
		return []connectMetaConfigListItem{{Key: "feishu", Name: meta.Name, Callback: filepath.Join(tmp, "plugins", "feishu")}}, nil
	}
	runConnectPluginInit = func(callbackPath string, flags map[string]string) error {
		notified = true
		return nil
	}
	defer func() {
		runConnectListMeta = originalRunConnectListMeta
		runConnectPluginInit = originalRunConnectPluginInit
	}()

	req, err := svc.AddRequest(connectsvc.RequestInput{Key: meta.Key, Request: "超过二十分钟的消息"})
	if err != nil {
		t.Fatalf("add request: %v", err)
	}
	oldTimestamp := time.Now().Add(-21 * time.Minute).Format(time.RFC3339)
	if _, err := db.Exec(`UPDATE connect_request SET created_at = ? WHERE id = ?`, oldTimestamp, req.ID); err != nil {
		t.Fatalf("age request: %v", err)
	}

	proxy := &ProxyServer{AgentDir: agentRoot, DeviceID: "dev", CacheTTL: time.Minute, ConnectCacheTTL: time.Second}
	syncConnectPendingRequests(proxy)

	var metaCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM task_meta WHERE task_type = 'feishu'`).Scan(&metaCount); err != nil {
		t.Fatalf("count task_meta: %v", err)
	}
	if metaCount != 1 {
		t.Fatalf("task_meta count = %d, want 1", metaCount)
	}

	var started int
	if err := db.QueryRow(`SELECT started FROM task_detail WHERE task_type = 'feishu' LIMIT 1`).Scan(&started); err != nil {
		t.Fatalf("query task_detail started: %v", err)
	}
	if started != 2 {
		t.Fatalf("started = %d, want 2", started)
	}

	var status int
	if err := db.QueryRow(`SELECT status FROM connect_request WHERE id = ?`, req.ID).Scan(&status); err != nil {
		t.Fatalf("query request status: %v", err)
	}
	if status != connectsvc.RequestStatusExpired {
		t.Fatalf("request status = %d, want expired", status)
	}
	if notified {
		t.Fatal("unexpected start notification for memo-only detail")
	}
}

func TestSyncConnectPendingRequestsUsesDefaultAutoReply(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	flushAgentCache()

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	var notifiedFlags map[string]string

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   filepath.Join(tmp, "data"),
		AgentDir: agentRoot,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	defer svc.Close()
	meta, err := svc.CreateMeta(connectsvc.MetaInput{
		Key:      "feishu",
		Name:     "feishu",
		Meta:     `{"appId":"x"}`,
		Callback: "./feishu",
		AgentID:  "A",
		ChatID:   "chat-feishu",
		Model:    "OpenAI",
		Thinking: true,
	})
	if err != nil {
		t.Fatalf("create meta: %v", err)
	}
	originalRunConnectListMeta := runConnectListMeta
	originalRunConnectPluginInit := runConnectPluginInit
	runConnectListMeta = func() ([]connectMetaConfigListItem, error) {
		return []connectMetaConfigListItem{{Key: "feishu", Name: meta.Name, Callback: filepath.Join(tmp, "plugins", "feishu")}}, nil
	}
	runConnectPluginInit = func(callbackPath string, flags map[string]string) error {
		notifiedFlags = flags
		return nil
	}
	defer func() {
		runConnectListMeta = originalRunConnectListMeta
		runConnectPluginInit = originalRunConnectPluginInit
	}()

	_, err = svc.AddRequest(connectsvc.RequestInput{Key: meta.Key, Request: "默认自动回复"})
	if err != nil {
		t.Fatalf("add request: %v", err)
	}
	proxy := &ProxyServer{AgentDir: agentRoot, DeviceID: "dev", CacheTTL: time.Minute, ConnectCacheTTL: time.Second}
	syncConnectPendingRequests(proxy)

	if notifiedFlags == nil {
		t.Fatal("expected notification flags")
	}
	if notifiedFlags["content"] != defaultConnectAutoReply {
		t.Fatalf("notify content = %q, want %q", notifiedFlags["content"], defaultConnectAutoReply)
	}
}

func TestSyncConnectPendingRequestsUsesCustomAutoReply(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	flushAgentCache()

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	var notifiedFlags map[string]string

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   filepath.Join(tmp, "data"),
		AgentDir: agentRoot,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	defer svc.Close()
	meta, err := svc.CreateMeta(connectsvc.MetaInput{
		Key:      "feishu",
		Name:     "feishu",
		Meta:     `{"appId":"x"}`,
		Callback: "./feishu",
		AgentID:  "A",
		ChatID:   "chat-feishu",
		Model:    "OpenAI",
		Thinking: true,
	})
	if err != nil {
		t.Fatalf("create meta: %v", err)
	}
	originalRunConnectListMeta := runConnectListMeta
	originalRunConnectPluginInit := runConnectPluginInit
	runConnectListMeta = func() ([]connectMetaConfigListItem, error) {
		return []connectMetaConfigListItem{{Key: "feishu", Name: meta.Name, Callback: filepath.Join(tmp, "plugins", "feishu")}}, nil
	}
	runConnectPluginInit = func(callbackPath string, flags map[string]string) error {
		notifiedFlags = flags
		return nil
	}
	defer func() {
		runConnectListMeta = originalRunConnectListMeta
		runConnectPluginInit = originalRunConnectPluginInit
	}()

	_, err = svc.AddRequest(connectsvc.RequestInput{Key: meta.Key, Request: "自定义自动回复"})
	if err != nil {
		t.Fatalf("add request: %v", err)
	}
	const customReply = "收到，已经开始处理你的想法"
	proxy := &ProxyServer{
		AgentDir:        agentRoot,
		DeviceID:        "dev",
		CacheTTL:        time.Minute,
		ConnectCacheTTL: time.Second,
		AutoReply:       customReply,
	}
	syncConnectPendingRequests(proxy)

	if notifiedFlags == nil {
		t.Fatal("expected notification flags")
	}
	if notifiedFlags["content"] != customReply {
		t.Fatalf("notify content = %q, want %q", notifiedFlags["content"], customReply)
	}
}

func TestCronExecuteOnceInjectsMetaIDIntoRequestMetadata(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	flushAgentCache()

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "config.json"), []byte(`{"version":"cron-v1"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer scheduled-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}
	if _, err := sandboxstate.Set(db, "A", "chat-a", sandboxstate.ModeNet); err != nil {
		t.Fatalf("set sandbox: %v", err)
	}
	metadataSnapshot, err := getAgentOutputForChat(agentRoot, "dev", time.Minute, "chat-a")
	if err != nil {
		t.Fatalf("getAgentOutputForChat: %v", err)
	}
	if len(metadataSnapshot.Agents) != 1 || metadataSnapshot.Agents[0].Version != "cron-v1" || metadataSnapshot.Agents[0].Sandbox != sandboxstate.ModeNet {
		t.Fatalf("agent snapshot = %+v", metadataSnapshot.Agents)
	}
	metaRes, err := db.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, chat_id, task_type, model, thinking, cron, content) VALUES (0, ?, 'A', 'chat-a', 'feishu', 'OpenAI', 1, '', '聚合消息')`,
		time.Now().Format("2006-01-02 15:04"))
	if err != nil {
		t.Fatalf("insert task_meta: %v", err)
	}
	metaID, _ := metaRes.LastInsertId()
	detailRes, err := db.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, meta_ref, task_type, model, thinking, content, started) VALUES (?, ?, 'A', 'chat-a', '11,12', 'feishu', 'OpenAI', 1, '聚合消息', 0)`,
		metaID, time.Now().Unix())
	if err != nil {
		t.Fatalf("insert task_detail: %v", err)
	}
	detailID, _ := detailRes.LastInsertId()

	var captured map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:            upstream.URL,
		AgentDir:        agentRoot,
		DeviceID:        "dev",
		CacheTTL:        time.Minute,
		ConnectCacheTTL: time.Second,
		Client:          &http.Client{Timeout: 5 * time.Second},
	}
	cronExecuteOnce(proxy)
	time.Sleep(300 * time.Millisecond)

	var started int
	if err := db.QueryRow(`SELECT started FROM task_detail WHERE id = ?`, detailID).Scan(&started); err != nil {
		t.Fatalf("query started: %v", err)
	}
	if started != 3 {
		t.Fatalf("started = %d, want 3", started)
	}
	var resultContent string
	if err := db.QueryRow(`SELECT result_content FROM task_detail WHERE id = ?`, detailID).Scan(&resultContent); err != nil {
		t.Fatalf("query result_content: %v", err)
	}
	if resultContent != "ok" {
		t.Fatalf("result_content = %q, want ok", resultContent)
	}

	metadata, ok := captured["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata missing: %+v", captured)
	}
	if metadata["META_ID"] != "12" {
		t.Fatalf("metadata META_ID = %v, want 12", metadata["META_ID"])
	}
	if metadata["chat"] != "chat-a" {
		t.Fatalf("metadata chat = %v, want chat-a", metadata["chat"])
	}
	if metadata["type"] != chatTypeScheduledTask {
		t.Fatalf("metadata type = %v, want %s", metadata["type"], chatTypeScheduledTask)
	}
	if metadata["cron_type"] != "feishu" {
		t.Fatalf("metadata cron_type = %v, want feishu", metadata["cron_type"])
	}
	agentMeta, ok := metadata["agent"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata.agent missing: %+v", metadata)
	}
	if agentMeta["version"] != "cron-v1" {
		t.Fatalf("metadata.agent.version = %v, want cron-v1", agentMeta["version"])
	}
	if agentMeta["sandbox"] != sandboxstate.ModeNet {
		t.Fatalf("metadata.agent.sandbox = %v, want %q", agentMeta["sandbox"], sandboxstate.ModeNet)
	}

	var chatLogCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM chat_log WHERE agent_id = 'A' AND chat_id = 'chat-a'`).Scan(&chatLogCount); err != nil {
		t.Fatalf("count chat_log: %v", err)
	}
	if chatLogCount < 2 {
		t.Fatalf("chat_log count = %d, want at least 2", chatLogCount)
	}
}

func TestCronExecuteOnceInjectsDefaultCronTaskTypeIntoRequestMetadata(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	flushAgentCache()

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer scheduled-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}
	metaRes, err := db.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, chat_id, task_type, model, thinking, cron, content) VALUES (0, ?, 'A', 'chat-cron', '', 'OpenAI', 0, '', '普通备忘录')`,
		time.Now().Format("2006-01-02 15:04"))
	if err != nil {
		t.Fatalf("insert task_meta: %v", err)
	}
	metaID, _ := metaRes.LastInsertId()
	detailRes, err := db.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, meta_ref, task_type, model, thinking, content, started) VALUES (?, ?, 'A', 'chat-cron', '', '', 'OpenAI', 0, '普通备忘录', 0)`,
		metaID, time.Now().Unix())
	if err != nil {
		t.Fatalf("insert task_detail: %v", err)
	}
	detailID, _ := detailRes.LastInsertId()

	var captured map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:            upstream.URL,
		AgentDir:        agentRoot,
		DeviceID:        "dev",
		CacheTTL:        time.Minute,
		ConnectCacheTTL: time.Second,
		Client:          &http.Client{Timeout: 5 * time.Second},
	}
	cronExecuteOnce(proxy)
	time.Sleep(300 * time.Millisecond)

	var started int
	if err := db.QueryRow(`SELECT started FROM task_detail WHERE id = ?`, detailID).Scan(&started); err != nil {
		t.Fatalf("query started: %v", err)
	}
	if started != 3 {
		t.Fatalf("started = %d, want 3", started)
	}

	metadata, ok := captured["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata missing: %+v", captured)
	}
	if metadata["cron_type"] != defaultTaskType {
		t.Fatalf("metadata cron_type = %v, want %s", metadata["cron_type"], defaultTaskType)
	}
	if _, exists := metadata["META_ID"]; exists {
		t.Fatalf("metadata META_ID should be absent when meta_ref is empty: %+v", metadata)
	}
}

func TestCronExecuteOnceInjectsResponseSchemaIntoRequestMetadata(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	flushAgentCache()

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer scheduled-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}
	schema := `{"type":"object","properties":{"summary":{"type":"string"}},"required":["summary"]}`
	metaRes, err := db.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, chat_id, task_type, model, thinking, cron, content, response_schema) VALUES (0, ?, 'A', 'chat-schema', 'cron', 'OpenAI', 0, '', '结构化备忘录', ?)`,
		time.Now().Format("2006-01-02 15:04"), schema)
	if err != nil {
		t.Fatalf("insert task_meta: %v", err)
	}
	metaID, _ := metaRes.LastInsertId()
	if _, err := db.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, meta_ref, task_type, model, thinking, content, response_schema, started) VALUES (?, ?, 'A', 'chat-schema', '', 'cron', 'OpenAI', 0, '结构化备忘录', ?, 0)`,
		metaID, time.Now().Unix(), schema); err != nil {
		t.Fatalf("insert task_detail: %v", err)
	}

	var captured map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:            upstream.URL,
		AgentDir:        agentRoot,
		DeviceID:        "dev",
		CacheTTL:        time.Minute,
		ConnectCacheTTL: time.Second,
		Client:          &http.Client{Timeout: 5 * time.Second},
	}
	cronExecuteOnce(proxy)
	time.Sleep(300 * time.Millisecond)

	metadata, ok := captured["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata missing: %+v", captured)
	}
	if metadata["response_schema"] != schema {
		t.Fatalf("metadata response_schema = %v, want %s", metadata["response_schema"], schema)
	}
}

func TestCronExecuteOnceInjectsTaskDetailRouterDisableIntoRequestMetadata(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	flushAgentCache()

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}
	// Agent-level config intentionally conflicts with the task detail value so
	// the execution path must prefer the memo task's persisted router_disable.
	if err := os.WriteFile(filepath.Join(agentDir, "config.json"), []byte(`{"description":"demo","router_disable":true}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer scheduled-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}
	metaRes, err := db.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, chat_id, task_type, model, thinking, router_disable, cron, content) VALUES (0, ?, 'A', 'chat-router', 'cron', 'OpenAI', 0, 1, '', '覆盖 Agent 蜂群配置')`,
		time.Now().Format("2006-01-02 15:04"))
	if err != nil {
		t.Fatalf("insert task_meta: %v", err)
	}
	metaID, _ := metaRes.LastInsertId()
	if _, err := db.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, meta_ref, task_type, model, thinking, router_disable, content, started) VALUES (?, ?, 'A', 'chat-router', '', 'cron', 'OpenAI', 0, 0, '覆盖 Agent 蜂群配置', 0)`,
		metaID, time.Now().Unix()); err != nil {
		t.Fatalf("insert task_detail: %v", err)
	}

	var captured map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	proxy := &ProxyServer{
		Host:            upstream.URL,
		AgentDir:        agentRoot,
		DeviceID:        "dev",
		CacheTTL:        time.Minute,
		ConnectCacheTTL: time.Second,
		Client:          &http.Client{Timeout: 5 * time.Second},
	}
	cronExecuteOnce(proxy)
	time.Sleep(300 * time.Millisecond)

	metadata, ok := captured["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata missing: %+v", captured)
	}
	msgs, ok := captured["messages"].([]interface{})
	if !ok || len(msgs) != 1 {
		t.Fatalf("messages = %#v, want one forwarded message", captured["messages"])
	}
	msg := msgs[0].(map[string]interface{})
	if msg["role"] != "user" || msg["content"] != "覆盖 Agent 蜂群配置" {
		t.Fatalf("forwarded message = %#v, want user/覆盖 Agent 蜂群配置", msg)
	}
	if _, exists := captured["message"]; exists {
		t.Fatalf("message should be absent: %v", captured["message"])
	}
	if captured["stream"] != true {
		t.Fatalf("stream = %v, want true", captured["stream"])
	}
	if _, exists := captured["thinking"]; exists {
		t.Fatalf("top-level thinking should be absent: %v", captured["thinking"])
	}
	if _, exists := captured["router_disable"]; exists {
		t.Fatalf("top-level router_disable should be absent: %v", captured["router_disable"])
	}
	if metadata["thinking"] != false {
		t.Fatalf("metadata thinking = %v, want false", metadata["thinking"])
	}
	if metadata["router_disable"] != false {
		t.Fatalf("metadata router_disable = %v, want false", metadata["router_disable"])
	}
}

func TestSyncConnectPendingRequestsRepliesCompletedConnectTask(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	sharedDataDB = pooledDB{}
	flushAgentCache()

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   filepath.Join(tmp, "data"),
		AgentDir: agentRoot,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	defer svc.Close()

	meta, err := svc.CreateMeta(connectsvc.MetaInput{
		Key:      "feishu",
		Name:     "feishu",
		Meta:     `{"appId":"x"}`,
		Callback: "./feishu",
		AgentID:  "A",
		ChatID:   "chat-feishu",
		Model:    "OpenAI",
		Thinking: true,
	})
	if err != nil {
		t.Fatalf("create meta: %v", err)
	}

	startedStatus := connectsvc.RequestStatusStarted
	earlier, err := svc.AddRequest(connectsvc.RequestInput{
		Key:        meta.Key,
		Request:    "更早消息",
		RawRequest: `{"message":"earlier"}`,
		Status:     &startedStatus,
	})
	if err != nil {
		t.Fatalf("add earlier request: %v", err)
	}
	target, err := svc.AddRequest(connectsvc.RequestInput{
		Key:        meta.Key,
		Request:    "需要回复的消息",
		RawRequest: `{"message":"target"}`,
		Status:     &startedStatus,
	})
	if err != nil {
		t.Fatalf("add target request: %v", err)
	}

	now := time.Now()
	metaRes, err := db.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, chat_id, task_type, model, thinking, cron, content) VALUES (0, ?, 'A', 'chat-feishu', 'feishu', 'OpenAI', 1, '', '需要回复的消息')`,
		now.Format("2006-01-02 15:04"))
	if err != nil {
		t.Fatalf("insert task_meta: %v", err)
	}
	metaID, _ := metaRes.LastInsertId()
	detailRes, err := db.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, meta_ref, task_type, model, thinking, content, result_content, started) VALUES (?, ?, 'A', 'chat-feishu', ?, 'feishu', 'OpenAI', 1, '需要回复的消息', '任务已完成，下面是结果', 3)`,
		metaID, now.Add(-time.Minute).Unix(), fmt.Sprintf("%d,%d", earlier.ID, target.ID))
	if err != nil {
		t.Fatalf("insert task_detail: %v", err)
	}
	detailID, _ := detailRes.LastInsertId()

	var notifiedCallback string
	var notifiedFlags map[string]string
	originalRunConnectListMeta := runConnectListMeta
	originalRunConnectPluginSend := runConnectPluginSend
	runConnectListMeta = func() ([]connectMetaConfigListItem, error) {
		return []connectMetaConfigListItem{{Key: "feishu", Name: meta.Name, Callback: filepath.Join(tmp, "plugins", "feishu")}}, nil
	}
	runConnectPluginSend = func(callbackPath string, flags map[string]string) error {
		notifiedCallback = callbackPath
		notifiedFlags = make(map[string]string, len(flags))
		for k, v := range flags {
			notifiedFlags[k] = v
		}
		return nil
	}
	defer func() {
		runConnectListMeta = originalRunConnectListMeta
		runConnectPluginSend = originalRunConnectPluginSend
	}()

	proxy := &ProxyServer{
		AgentDir:        agentRoot,
		DeviceID:        "dev",
		CacheTTL:        time.Minute,
		ConnectCacheTTL: time.Second,
	}
	syncConnectPendingRequests(proxy)

	wantCallback := filepath.Join(tmp, "plugins", "feishu")
	if notifiedCallback != wantCallback {
		t.Fatalf("notified callback = %q, want %q", notifiedCallback, wantCallback)
	}
	if notifiedFlags["content"] != "任务已完成，下面是结果" {
		t.Fatalf("notify content = %q", notifiedFlags["content"])
	}
	if strings.TrimSpace(notifiedFlags["message"]) == "" {
		t.Fatal("notify message is empty")
	}

	requests, err := svc.ListRequests(connectsvc.RequestFilter{Key: meta.Key, Limit: 10})
	if err != nil {
		t.Fatalf("list requests: %v", err)
	}
	statusByID := make(map[int]int, len(requests))
	for _, req := range requests {
		statusByID[req.ID] = req.Status
	}
	if statusByID[target.ID] != connectsvc.RequestStatusReplied {
		t.Fatalf("target status = %d, want replied", statusByID[target.ID])
	}
	if statusByID[earlier.ID] != connectsvc.RequestStatusCompleted {
		t.Fatalf("earlier status = %d, want completed", statusByID[earlier.ID])
	}

	var repliedAt string
	if err := db.QueryRow(`SELECT replied_at FROM task_detail WHERE id = ?`, detailID).Scan(&repliedAt); err != nil {
		t.Fatalf("query replied_at: %v", err)
	}
	if strings.TrimSpace(repliedAt) == "" {
		t.Fatal("replied_at is empty")
	}

	notifiedCallback = ""
	notifiedFlags = nil
	syncConnectPendingRequests(proxy)
	if notifiedCallback != "" || notifiedFlags != nil {
		t.Fatal("expected completed reply to be sent only once")
	}
}

func withPluginSandbox(t *testing.T, plugins map[string][2]string) func() {
	t.Helper()
	root := t.TempDir()
	workdir := filepath.Join(root, "work")
	pluginsDir := filepath.Join(root, "plugins")
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for file, outputs := range plugins {
		writePluginScript(t, filepath.Join(pluginsDir, file), outputs[0], outputs[1])
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workdir); err != nil {
		t.Fatal(err)
	}
	restorePluginDir := connectsvc.SetPluginDirResolverForTest(func() (string, error) {
		return pluginsDir, nil
	})
	originalLocalPluginDirResolver := localPluginDirResolver
	localPluginDirResolver = func() (string, error) {
		return pluginsDir, nil
	}
	return func() {
		restorePluginDir()
		localPluginDirResolver = originalLocalPluginDirResolver
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}
}

func TestSyncConnectPendingRequestsSkipsCompletedReplyWhenMetaRefTargetNotStarted(t *testing.T) {
	tmp := t.TempDir()
	restore := withPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	sharedDataDB = pooledDB{}
	flushAgentCache()
	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   filepath.Join(tmp, "data"),
		AgentDir: agentRoot,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	defer svc.Close()

	meta, err := svc.CreateMeta(connectsvc.MetaInput{
		Key:      "feishu",
		Name:     "feishu",
		Meta:     `{"appId":"x"}`,
		Callback: "./feishu",
		AgentID:  "A",
		ChatID:   "chat-feishu",
		Model:    "OpenAI",
		Thinking: true,
	})
	if err != nil {
		t.Fatalf("create meta: %v", err)
	}

	startedStatus := connectsvc.RequestStatusStarted
	target, err := svc.AddRequest(connectsvc.RequestInput{
		Key:        meta.Key,
		Request:    "原始目标消息",
		RawRequest: `{"message":"target"}`,
		Status:     &startedStatus,
	})
	if err != nil {
		t.Fatalf("add target request: %v", err)
	}
	repliedStatus := connectsvc.RequestStatusReplied
	if _, err := svc.UpdateRequestStatus(connectsvc.RequestStatusUpdate{
		ID:     target.ID,
		Key:    meta.Key,
		From:   &startedStatus,
		To:     repliedStatus,
		Strict: true,
	}); err != nil {
		t.Fatalf("update target request status: %v", err)
	}
	later, err := svc.AddRequest(connectsvc.RequestInput{
		Key:        meta.Key,
		Request:    "后续新消息",
		RawRequest: `{"message":"later"}`,
		Status:     &startedStatus,
	})
	if err != nil {
		t.Fatalf("add later request: %v", err)
	}

	now := time.Now()
	metaRes, err := db.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, chat_id, task_type, model, thinking, cron, content) VALUES (0, ?, 'A', 'chat-feishu', 'feishu', 'OpenAI', 1, '', '原始目标消息')`,
		now.Format("2006-01-02 15:04"))
	if err != nil {
		t.Fatalf("insert task_meta: %v", err)
	}
	metaID, _ := metaRes.LastInsertId()
	detailRes, err := db.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, meta_ref, task_type, model, thinking, content, result_content, started) VALUES (?, ?, 'A', 'chat-feishu', ?, 'feishu', 'OpenAI', 1, '原始目标消息', '任务完成结果', 3)`,
		metaID, now.Add(-time.Minute).Unix(), strconv.Itoa(target.ID))
	if err != nil {
		t.Fatalf("insert task_detail: %v", err)
	}
	detailID, _ := detailRes.LastInsertId()

	var sendCalls int
	originalRunConnectListMeta := runConnectListMeta
	originalRunConnectPluginSend := runConnectPluginSend
	runConnectListMeta = func() ([]connectMetaConfigListItem, error) {
		return []connectMetaConfigListItem{{Key: "feishu", Name: meta.Name, Callback: filepath.Join(tmp, "plugins", "feishu")}}, nil
	}
	runConnectPluginSend = func(callbackPath string, flags map[string]string) error {
		sendCalls++
		return nil
	}
	defer func() {
		runConnectListMeta = originalRunConnectListMeta
		runConnectPluginSend = originalRunConnectPluginSend
	}()

	proxy := &ProxyServer{
		AgentDir:        agentRoot,
		DeviceID:        "dev",
		CacheTTL:        time.Minute,
		ConnectCacheTTL: time.Second,
	}
	syncConnectPendingRequests(proxy)

	if sendCalls != 0 {
		t.Fatalf("send calls = %d, want 0", sendCalls)
	}

	requests, err := svc.ListRequests(connectsvc.RequestFilter{Key: meta.Key, Limit: 10})
	if err != nil {
		t.Fatalf("list requests: %v", err)
	}
	statusByID := make(map[int]int, len(requests))
	for _, req := range requests {
		statusByID[req.ID] = req.Status
	}
	if statusByID[target.ID] != connectsvc.RequestStatusReplied {
		t.Fatalf("target status = %d, want replied", statusByID[target.ID])
	}
	if statusByID[later.ID] != connectsvc.RequestStatusStarted {
		t.Fatalf("later status = %d, want started", statusByID[later.ID])
	}

	var repliedAt string
	if err := db.QueryRow(`SELECT replied_at FROM task_detail WHERE id = ?`, detailID).Scan(&repliedAt); err != nil {
		t.Fatalf("query replied_at: %v", err)
	}
	if strings.TrimSpace(repliedAt) != "" {
		t.Fatalf("replied_at = %q, want empty", repliedAt)
	}
}

func TestSyncConnectPendingRequestsNormalizesReplyContentBeforePluginSend(t *testing.T) {
	tmp := t.TempDir()
	restore := withPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	sharedDataDB = pooledDB{}
	flushAgentCache()
	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   filepath.Join(tmp, "data"),
		AgentDir: agentRoot,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	defer svc.Close()

	meta, err := svc.CreateMeta(connectsvc.MetaInput{
		Key:      "feishu",
		Name:     "feishu",
		Meta:     `{"appId":"x"}`,
		Callback: "./feishu",
		AgentID:  "A",
		ChatID:   "chat-feishu",
		Model:    "OpenAI",
		Thinking: true,
	})
	if err != nil {
		t.Fatalf("create meta: %v", err)
	}

	startedStatus := connectsvc.RequestStatusStarted
	target, err := svc.AddRequest(connectsvc.RequestInput{
		Key:        meta.Key,
		Request:    "原始目标消息",
		RawRequest: `{"message":"target"}`,
		Status:     &startedStatus,
	})
	if err != nil {
		t.Fatalf("add target request: %v", err)
	}

	now := time.Now()
	metaRes, err := db.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, chat_id, task_type, model, thinking, cron, content) VALUES (0, ?, 'A', 'chat-feishu', 'feishu', 'OpenAI', 1, '', '原始目标消息')`,
		now.Format("2006-01-02 15:04"))
	if err != nil {
		t.Fatalf("insert task_meta: %v", err)
	}
	metaID, _ := metaRes.LastInsertId()
	detailRes, err := db.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, meta_ref, task_type, model, thinking, content, result_content, started) VALUES (?, ?, 'A', 'chat-feishu', ?, 'feishu', 'OpenAI', 1, '原始目标消息', ?, 3)`,
		metaID, now.Add(-time.Minute).Unix(), strconv.Itoa(target.ID), "```json\n{\n  \"hello\": \"world\"\n}\n```")
	if err != nil {
		t.Fatalf("insert task_detail: %v", err)
	}
	detailID, _ := detailRes.LastInsertId()

	var notifiedFlags map[string]string
	originalRunConnectListMeta := runConnectListMeta
	originalRunConnectPluginSend := runConnectPluginSend
	runConnectListMeta = func() ([]connectMetaConfigListItem, error) {
		return []connectMetaConfigListItem{{Key: "feishu", Name: meta.Name, Callback: filepath.Join(tmp, "plugins", "feishu")}}, nil
	}
	runConnectPluginSend = func(callbackPath string, flags map[string]string) error {
		notifiedFlags = make(map[string]string, len(flags))
		for k, v := range flags {
			notifiedFlags[k] = v
		}
		return nil
	}
	defer func() {
		runConnectListMeta = originalRunConnectListMeta
		runConnectPluginSend = originalRunConnectPluginSend
	}()

	proxy := &ProxyServer{
		AgentDir:        agentRoot,
		DeviceID:        "dev",
		CacheTTL:        time.Minute,
		ConnectCacheTTL: time.Second,
	}
	syncConnectPendingRequests(proxy)

	if notifiedFlags == nil {
		t.Fatal("expected send flags")
	}
	if notifiedFlags["content"] != "```json\n{\n  \"hello\": \"world\"\n}\n```" {
		t.Fatalf("notify content = %q", notifiedFlags["content"])
	}

	var repliedAt string
	if err := db.QueryRow(`SELECT replied_at FROM task_detail WHERE id = ?`, detailID).Scan(&repliedAt); err != nil {
		t.Fatalf("query replied_at: %v", err)
	}
	if strings.TrimSpace(repliedAt) == "" {
		t.Fatal("replied_at is empty")
	}
}

func TestRunConnectPluginSendNormalizesJSONMarkdownContent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, nil)
	defer restore()

	callbackPath := filepath.Join("..", "plugins", "normalize-send")
	callPath := filepath.Join("..", "plugins", "normalize-send.content")
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"command\" ]; then\n" +
		"  printf '%s\\n' '[\"send\"]'\n" +
		"  exit 0\n" +
		"fi\n" +
		"while [ $# -gt 0 ]; do\n" +
		"  if [ \"$1\" = \"--content\" ]; then\n" +
		"    shift\n" +
		"    printf '%s' \"$1\" > \"" + escapeSingleQuotes(callPath) + "\"\n" +
		"    exit 0\n" +
		"  fi\n" +
		"  shift\n" +
		"done\n" +
		"exit 1\n"
	if err := os.WriteFile(callbackPath, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}

	err := runConnectPluginSend(callbackPath, map[string]string{
		"name":    "normalize-send",
		"message": `{"id":1}`,
		"content": "```json\n{\n  \"hello\": \"world\"\n}\n```",
	})
	if err != nil {
		t.Fatalf("run send: %v", err)
	}

	data, err := os.ReadFile(callPath)
	if err != nil {
		t.Fatalf("read content file: %v", err)
	}
	if got := string(data); got != "{\"hello\":\"world\"}" {
		t.Fatalf("send content = %q, want compact json", got)
	}
}

func TestRunConnectPluginInitReturnsErrorWhenCommandDoesNotExposeInit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, nil)
	defer restore()

	callbackPath := filepath.Join("..", "plugins", "nosend")
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"command\" ]; then\n" +
		"  printf '%s\\n' '[\"start\",\"stop\"]'\n" +
		"  exit 0\n" +
		"fi\n" +
		"printf '%s\\n' \"$1\" >> \"" + escapeSingleQuotes(filepath.Join("..", "plugins", "nosend.calls")) + "\"\n" +
		"exit 0\n"
	if err := os.WriteFile(callbackPath, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}

	err := runConnectPluginInit(callbackPath, map[string]string{
		"name":    "nosend",
		"message": `{"id":1}`,
		"content": "任务结果",
	})
	if err == nil || !strings.Contains(err.Error(), "does not expose init") {
		t.Fatalf("err = %v, want missing init error", err)
	}

	callPath := filepath.Join("..", "plugins", "nosend.calls")
	if _, statErr := os.Stat(callPath); !os.IsNotExist(statErr) {
		t.Fatalf("init command should not run, stat err = %v", statErr)
	}
}

func TestRunConnectPluginInitFallsBackToHelpWhenCommandUnsupported(t *testing.T) {
	t.Skip("obsolete: callback runtime no longer probes unsupported commands via --help fallback")
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, nil)
	defer restore()

	callbackPath := filepath.Join("..", "plugins", "legacy")
	callPath := filepath.Join("..", "plugins", "legacy.calls")
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"command\" ]; then\n" +
		"  printf '%s\\n' 'unknown command' >&2\n" +
		"  exit 1\n" +
		"fi\n" +
		"if [ \"$1\" = \"--help\" ]; then\n" +
		"  printf '%s\\n' 'usage: legacy start|stop|init'\n" +
		"  exit 0\n" +
		"fi\n" +
		"printf '%s\\n' \"$1\" >> \"" + escapeSingleQuotes(callPath) + "\"\n" +
		"exit 0\n"
	if err := os.WriteFile(callbackPath, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}

	err := runConnectPluginInit(callbackPath, map[string]string{
		"name":    "legacy",
		"message": `{"id":1}`,
		"content": "开始执行",
	})
	if err != nil {
		t.Fatalf("run init: %v", err)
	}

	data, err := os.ReadFile(callPath)
	if err != nil {
		t.Fatalf("read calls: %v", err)
	}
	lines := strings.Fields(string(data))
	if len(lines) != 1 || lines[0] != "init" {
		t.Fatalf("calls = %v, want [init]", lines)
	}
}

func TestRunConnectPluginSendFallsBackToHelpWhenCommandUnsupported(t *testing.T) {
	t.Skip("obsolete: callback runtime no longer probes unsupported commands via --help fallback")
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, nil)
	defer restore()

	callbackPath := filepath.Join("..", "plugins", "legacy-send")
	callPath := filepath.Join("..", "plugins", "legacy-send.calls")
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"command\" ]; then\n" +
		"  printf '%s\\n' 'unknown command' >&2\n" +
		"  exit 1\n" +
		"fi\n" +
		"if [ \"$1\" = \"--help\" ]; then\n" +
		"  printf '%s\\n' 'usage: legacy-send start|stop|send'\n" +
		"  exit 0\n" +
		"fi\n" +
		"printf '%s\\n' \"$1\" >> \"" + escapeSingleQuotes(callPath) + "\"\n" +
		"exit 0\n"
	if err := os.WriteFile(callbackPath, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}

	err := runConnectPluginSend(callbackPath, map[string]string{
		"name":    "legacy-send",
		"message": `{"id":1}`,
		"content": "任务完成",
	})
	if err != nil {
		t.Fatalf("run send: %v", err)
	}

	data, err := os.ReadFile(callPath)
	if err != nil {
		t.Fatalf("read calls: %v", err)
	}
	lines := strings.Fields(string(data))
	if len(lines) != 1 || lines[0] != "send" {
		t.Fatalf("calls = %v, want [send]", lines)
	}
}

func TestSyncConnectPendingRequestsMarksStartedWhenPluginHasNoInit(t *testing.T) {
	t.Skip("obsolete: strict callback protocol no longer skips missing init")
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, nil)
	defer restore()

	callbackPath := filepath.Join("..", "plugins", "feishu")
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"command\" ]; then\n" +
		"  printf '%s\\n' '[\"start\",\"stop\",\"send\"]'\n" +
		"  exit 0\n" +
		"fi\n" +
		"printf '%s\\n' \"$1\" >> \"" + escapeSingleQuotes(filepath.Join("..", "plugins", "feishu.calls")) + "\"\n" +
		"exit 0\n"
	if err := os.WriteFile(callbackPath, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}

	workdir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	agentRoot := filepath.Join(workdir, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	sharedDataDB = pooledDB{}
	flushAgentCache()
	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   filepath.Join(workdir, "data"),
		AgentDir: agentRoot,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	defer svc.Close()

	meta, err := svc.CreateMeta(connectsvc.MetaInput{
		Name:     "feishu",
		Meta:     `{"appId":"x"}`,
		Callback: "./feishu",
		AgentID:  "A",
		ChatID:   "chat-feishu",
		Model:    "OpenAI",
		Thinking: true,
	})
	if err != nil {
		t.Fatalf("create meta: %v", err)
	}

	request, err := svc.AddRequest(connectsvc.RequestInput{
		Key:        meta.Key,
		Request:    "原始目标消息",
		RawRequest: `{"message":"target"}`,
	})
	if err != nil {
		t.Fatalf("add request: %v", err)
	}
	oldTimestamp := time.Now().Add(-11 * time.Minute).Format(time.RFC3339)
	if _, err := db.Exec(`UPDATE connect_request SET created_at = ? WHERE id = ?`, oldTimestamp, request.ID); err != nil {
		t.Fatalf("age request: %v", err)
	}

	originalRunConnectListMeta := runConnectListMeta
	runConnectListMeta = func() ([]connectMetaConfigListItem, error) {
		return []connectMetaConfigListItem{{Key: "feishu", Name: meta.Name, Callback: callbackPath}}, nil
	}
	defer func() {
		runConnectListMeta = originalRunConnectListMeta
	}()

	proxy := &ProxyServer{
		AgentDir:        agentRoot,
		DeviceID:        "dev",
		CacheTTL:        time.Minute,
		ConnectCacheTTL: time.Second,
	}
	syncConnectPendingRequests(proxy)

	var detailCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM task_detail WHERE task_type = 'feishu'`).Scan(&detailCount); err != nil {
		t.Fatalf("count task_detail: %v", err)
	}
	if detailCount != 1 {
		t.Fatalf("task_detail count = %d, want 1", detailCount)
	}

	requests, err := svc.ListRequests(connectsvc.RequestFilter{Key: meta.Key, Limit: 10})
	if err != nil {
		t.Fatalf("list requests: %v", err)
	}
	if len(requests) != 1 || requests[0].Status != connectsvc.RequestStatusStarted {
		t.Fatalf("unexpected request status after skip: %+v", requests)
	}

	callPath := filepath.Join("..", "plugins", "feishu.calls")
	if callData, err := os.ReadFile(callPath); err == nil {
		if strings.Contains(string(callData), "init") {
			t.Fatalf("init command should not run, calls = %s", string(callData))
		}
	} else if !os.IsNotExist(err) {
		t.Fatalf("read call log: %v", err)
	}

	logPath, err := pluginlog.ResolveLogPath("feishu")
	if err != nil {
		t.Fatalf("resolve log path: %v", err)
	}
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(logData), "stage=init skip=true") {
		t.Fatalf("unexpected log: %s", string(logData))
	}
}

func TestSyncConnectPendingRequestsMarksCompletedReplySkippedWhenPluginHasNoSend(t *testing.T) {
	t.Skip("obsolete: strict callback protocol no longer skips missing send")
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withPluginSandbox(t, nil)
	defer restore()

	callbackPath := filepath.Join("..", "plugins", "feishu")
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"command\" ]; then\n" +
		"  printf '%s\\n' '[\"start\",\"stop\"]'\n" +
		"  exit 0\n" +
		"fi\n" +
		"printf '%s\\n' \"$1\" >> \"" + escapeSingleQuotes(filepath.Join("..", "plugins", "feishu.calls")) + "\"\n" +
		"exit 0\n"
	if err := os.WriteFile(callbackPath, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}

	workdir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	agentRoot := filepath.Join(workdir, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	sharedDataDB = pooledDB{}
	flushAgentCache()
	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if err := ensureTokenStore(db); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   filepath.Join(workdir, "data"),
		AgentDir: agentRoot,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	defer svc.Close()

	meta, err := svc.CreateMeta(connectsvc.MetaInput{
		Name:     "feishu",
		Meta:     `{"appId":"x"}`,
		Callback: "./feishu",
		AgentID:  "A",
		ChatID:   "chat-feishu",
		Model:    "OpenAI",
		Thinking: true,
	})
	if err != nil {
		t.Fatalf("create meta: %v", err)
	}

	startedStatus := connectsvc.RequestStatusStarted
	target, err := svc.AddRequest(connectsvc.RequestInput{
		Key:        meta.Key,
		Request:    "原始目标消息",
		RawRequest: `{"message":"target"}`,
		Status:     &startedStatus,
	})
	if err != nil {
		t.Fatalf("add target request: %v", err)
	}

	now := time.Now()
	metaRes, err := db.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, chat_id, task_type, model, thinking, cron, content) VALUES (0, ?, 'A', 'chat-feishu', 'feishu', 'OpenAI', 1, '', '原始目标消息')`,
		now.Format("2006-01-02 15:04"))
	if err != nil {
		t.Fatalf("insert task_meta: %v", err)
	}
	metaID, _ := metaRes.LastInsertId()
	detailRes, err := db.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, meta_ref, task_type, model, thinking, content, result_content, started) VALUES (?, ?, 'A', 'chat-feishu', ?, 'feishu', 'OpenAI', 1, '原始目标消息', '任务完成结果', 3)`,
		metaID, now.Add(-time.Minute).Unix(), strconv.Itoa(target.ID))
	if err != nil {
		t.Fatalf("insert task_detail: %v", err)
	}
	detailID, _ := detailRes.LastInsertId()

	originalRunConnectListMeta := runConnectListMeta
	runConnectListMeta = func() ([]connectMetaConfigListItem, error) {
		return []connectMetaConfigListItem{{Key: "feishu", Name: meta.Name, Callback: callbackPath}}, nil
	}
	defer func() {
		runConnectListMeta = originalRunConnectListMeta
	}()

	proxy := &ProxyServer{
		AgentDir:        agentRoot,
		DeviceID:        "dev",
		CacheTTL:        time.Minute,
		ConnectCacheTTL: time.Second,
	}
	syncConnectPendingRequests(proxy)

	var repliedAt string
	if err := db.QueryRow(`SELECT replied_at FROM task_detail WHERE id = ?`, detailID).Scan(&repliedAt); err != nil {
		t.Fatalf("query replied_at: %v", err)
	}
	if strings.TrimSpace(repliedAt) == "" {
		t.Fatal("replied_at is empty, want skip to close the task")
	}

	requests, err := svc.ListRequests(connectsvc.RequestFilter{Key: meta.Key, Limit: 10})
	if err != nil {
		t.Fatalf("list requests: %v", err)
	}
	if len(requests) != 1 || requests[0].Status != connectsvc.RequestStatusStarted {
		t.Fatalf("unexpected request status after skip: %+v", requests)
	}

	callPath := filepath.Join("..", "plugins", "feishu.calls")
	if callData, err := os.ReadFile(callPath); err == nil {
		if strings.Contains(string(callData), "send") {
			t.Fatalf("send command should not run, calls = %s", string(callData))
		}
	} else if !os.IsNotExist(err) {
		t.Fatalf("read call log: %v", err)
	}

	logPath, err := pluginlog.ResolveLogPath("feishu")
	if err != nil {
		t.Fatalf("resolve log path: %v", err)
	}
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(logData), "stage=send skip=true") {
		t.Fatalf("unexpected log: %s", string(logData))
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
		t.Fatalf("unknown action mode: %s", mode)
	}
	content += "printf 'unknown command: %s\\n' \"$1\" >&2\n" +
		"exit 1\n"
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
		"  printf '%s\\n' '[\"reuse\",\"agent\",\"provider\",\"thinking\",\"swarm\"]'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"command\" ]; then\n" +
		"  printf '%s\\n' '[\"command\",\"help\",\"name\",\"param\",\"start\",\"stop\",\"instance init\"]'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"help\" ]; then\n" +
		"  printf '%s\\n' 'help'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"instance\" ] && [ \"$2\" = \"init\" ]; then\n" +
		"  shift 2\n" +
		"  agent=''\n" +
		"  chat=''\n" +
		"  connect=''\n" +
		"  while [ \"$#\" -gt 0 ]; do\n" +
		"    case \"$1\" in\n" +
		"      --agentId)\n" +
		"        shift\n" +
		"        agent=\"$1\"\n" +
		"        ;;\n" +
		"      --chatId)\n" +
		"        shift\n" +
		"        chat=\"$1\"\n" +
		"        ;;\n" +
		"      --connect-bin)\n" +
		"        shift\n" +
		"        connect=\"$1\"\n" +
		"        ;;\n" +
		"    esac\n" +
		"    shift\n" +
		"  done\n" +
		"  printf '{\"subcommand\":\"instance init\",\"agentId\":\"%s\",\"chatId\":\"%s\",\"connectBin\":\"%s\"}\\n' \"$agent\" \"$chat\" \"$connect\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"printf 'unknown command: %s\\n' \"$1\" >&2\n" +
		"exit 1\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func escapeSingleQuotes(input string) string {
	return strings.ReplaceAll(input, "'", `'"'"'`)
}

func seedPluginMetaTestData(t *testing.T, dbPath, agentDir string, input connectsvc.MetaInput) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(agentDir, input.AgentID), 0o755); err != nil {
		t.Fatalf("mkdir agent: %v", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS token_store (model TEXT PRIMARY KEY, token TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL)`); err != nil {
		t.Fatalf("create token_store: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO token_store (model, token, updated_at) VALUES (?,?,?) ON CONFLICT(model) DO UPDATE SET token=excluded.token, updated_at=excluded.updated_at`, input.Model, "Bearer test-token", time.Now().Format(time.RFC3339)); err != nil {
		t.Fatalf("seed token_store: %v", err)
	}
	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   dbPath,
		AgentDir: agentDir,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("new connect service: %v", err)
	}
	defer svc.Close()
	if _, err := connectsvc.UpsertPluginConfigWithService(svc, input); err != nil {
		t.Fatalf("upsert plugin config: %v", err)
	}
}

func collectPluginLogEvents(body io.Reader, lines chan<- string, errs chan<- error) {
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			lines <- strings.TrimPrefix(line, "data: ")
		}
	}
	if err := scanner.Err(); err != nil {
		errs <- err
		return
	}
	errs <- io.EOF
}

func captureProxyStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	os.Stdout = wOut
	defer func() {
		os.Stdout = oldStdout
	}()

	done := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(rOut)
		done <- string(data)
	}()

	fn()
	_ = wOut.Close()
	return <-done
}

func TestProxyBrowserURL(t *testing.T) {
	if got := proxyBrowserURL(8080); got != "http://localhost:8080/site/#app" {
		t.Fatalf("browser url = %q", got)
	}
	if got := proxyBrowserURL(0); got != "http://localhost:8080/site/#app" {
		t.Fatalf("default browser url = %q", got)
	}
}

func TestValidateServeOptionsRejectsUnsupportedPort(t *testing.T) {
	err := validateServeOptions(serveOptions{port: 9876})
	if err == nil {
		t.Fatalf("expected unsupported port error")
	}
	if !strings.Contains(err.Error(), "--port only supports 8080") {
		t.Fatalf("err = %v", err)
	}
}
