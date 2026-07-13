package main

import (
	"agent-scanner/agentcore"
	"bufio"
	"bytes"
	"compress/gzip"
	"connect/connectsvc"
	"connect/sandboxstate"
	"connect/sharedutil"
	"connect/skillstate"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	integrationknowledge "integration/knowledge"
	integrationmessageinsert "integration/messageinsert"
	integrationnotification "integration/notification"
	"io"
	"knowledge/knowledgecore"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"runtimepaths"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	tokenconsume "integration/consume"
	"skill-scanner/skillscore"
)

// ── Proxy test: metadata injection + SSE streaming ──

func ensureKnowledgeRequestLockTable(db *sql.DB) error {
	return sharedutil.EnsureKnowledgeRequestLockTable(db)
}

func gzipBase64String(input string) string {
	return sharedutil.GzipBase64String(input)
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

func integrationTestMacRuntimeBaseDir(home string) string {
	return runtimepaths.MacAppRuntimeBaseDir(home, integrationBundleID, integrationRuntimeAppDir)
}

func copyTestDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

func disableIntegrationNotificationsForTest(t *testing.T) {
	t.Helper()
	oldSupported := integrationNotificationSupportedFn
	oldSend := integrationSendNotificationFn
	t.Cleanup(func() {
		integrationNotificationSupportedFn = oldSupported
		integrationSendNotificationFn = oldSend
	})
	integrationNotificationSupportedFn = func() bool { return false }
	integrationSendNotificationFn = func(integrationnotification.Options) error { return nil }
}

func disableIntegrationManagedRuntimeForTest(t *testing.T) {
	t.Helper()
	oldGOOS := integrationRuntimeGOOS
	t.Cleanup(func() {
		integrationRuntimeGOOS = oldGOOS
	})
	integrationRuntimeGOOS = "windows"
}

func TestHandleHeartbeatIncludesCliGetRuntimeStatus(t *testing.T) {
	hbMu.Lock()
	oldTimestamp := hbLastTimestamp
	oldStatus := hbLastStatus
	oldError := hbLastError
	hbMu.Unlock()
	cliGetRuntimeMu.RLock()
	oldTaskQueue := cliGetTaskQueue
	oldPublishQueue := cliGetPublishQueue
	cliGetRuntimeMu.RUnlock()
	oldRunningWorkers := atomic.LoadInt64(&cliGetRunningWorkers)
	cliGetActivityMu.Lock()
	oldRecentActivity := append([]cliGetRuntimeEvent(nil), cliGetRecentActivity...)
	cliGetActivityMu.Unlock()
	t.Cleanup(func() {
		hbMu.Lock()
		hbLastTimestamp = oldTimestamp
		hbLastStatus = oldStatus
		hbLastError = oldError
		hbMu.Unlock()
		setCliGetRuntimeQueues(oldTaskQueue, oldPublishQueue)
		setCliGetRunningWorkers(oldRunningWorkers)
		cliGetActivityMu.Lock()
		cliGetRecentActivity = oldRecentActivity
		cliGetActivityMu.Unlock()
	})

	cliGetActivityMu.Lock()
	cliGetRecentActivity = []cliGetRuntimeEvent{
		{At: time.Now().Add(-2 * time.Second), cliGetRuntimeSnapshot: cliGetRuntimeSnapshot{PendingPublishes: 1}},
		{At: time.Now().Add(-1500 * time.Millisecond), cliGetRuntimeSnapshot: cliGetRuntimeSnapshot{RunningWorkers: 3}},
		{At: time.Now().Add(-1 * time.Second), cliGetRuntimeSnapshot: cliGetRuntimeSnapshot{PendingTasks: 2}},
	}
	cliGetActivityMu.Unlock()
	recordHeartbeat(2, "")

	req := httptest.NewRequest(http.MethodGet, "/api/heartbeat", nil)
	rec := httptest.NewRecorder()
	handleHeartbeat().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp struct {
		LastTimestamp    int64  `json:"lastTimestamp"`
		Status           int    `json:"status"`
		LastError        string `json:"lastError"`
		PendingPublishes int    `json:"pendingPublishes"`
		RunningWorkers   int    `json:"runningWorkers"`
		PendingTasks     int    `json:"pendingTasks"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode heartbeat response: %v", err)
	}
	if resp.LastTimestamp <= 0 {
		t.Fatalf("lastTimestamp = %d, want > 0", resp.LastTimestamp)
	}
	if resp.Status != 2 {
		t.Fatalf("status = %d, want 2", resp.Status)
	}
	if resp.LastError != "" {
		t.Fatalf("lastError = %q, want empty", resp.LastError)
	}
	if resp.PendingPublishes != 1 {
		t.Fatalf("pendingPublishes = %d, want 1", resp.PendingPublishes)
	}
	if resp.RunningWorkers != 3 {
		t.Fatalf("runningWorkers = %d, want 3", resp.RunningWorkers)
	}
	if resp.PendingTasks != 2 {
		t.Fatalf("pendingTasks = %d, want 2", resp.PendingTasks)
	}
}

func TestHandleHeartbeatDropsExpiredCliGetRuntimeEvents(t *testing.T) {
	hbMu.Lock()
	oldTimestamp := hbLastTimestamp
	oldStatus := hbLastStatus
	oldError := hbLastError
	hbMu.Unlock()
	cliGetRuntimeMu.RLock()
	oldTaskQueue := cliGetTaskQueue
	oldPublishQueue := cliGetPublishQueue
	cliGetRuntimeMu.RUnlock()
	oldRunningWorkers := atomic.LoadInt64(&cliGetRunningWorkers)
	cliGetActivityMu.Lock()
	oldRecentActivity := append([]cliGetRuntimeEvent(nil), cliGetRecentActivity...)
	cliGetActivityMu.Unlock()
	t.Cleanup(func() {
		hbMu.Lock()
		hbLastTimestamp = oldTimestamp
		hbLastStatus = oldStatus
		hbLastError = oldError
		hbMu.Unlock()
		setCliGetRuntimeQueues(oldTaskQueue, oldPublishQueue)
		setCliGetRunningWorkers(oldRunningWorkers)
		cliGetActivityMu.Lock()
		cliGetRecentActivity = oldRecentActivity
		cliGetActivityMu.Unlock()
	})

	cliGetActivityMu.Lock()
	cliGetRecentActivity = []cliGetRuntimeEvent{
		{At: time.Now().Add(-6 * time.Second), cliGetRuntimeSnapshot: cliGetRuntimeSnapshot{PendingPublishes: 9, RunningWorkers: 9, PendingTasks: 9}},
		{At: time.Now().Add(-1 * time.Second), cliGetRuntimeSnapshot: cliGetRuntimeSnapshot{PendingPublishes: 2, RunningWorkers: 1, PendingTasks: 3}},
	}
	cliGetActivityMu.Unlock()
	recordHeartbeat(2, "")

	req := httptest.NewRequest(http.MethodGet, "/api/heartbeat", nil)
	rec := httptest.NewRecorder()
	handleHeartbeat().ServeHTTP(rec, req)

	var resp struct {
		PendingPublishes int `json:"pendingPublishes"`
		RunningWorkers   int `json:"runningWorkers"`
		PendingTasks     int `json:"pendingTasks"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode heartbeat response: %v", err)
	}
	if resp.PendingPublishes != 2 || resp.RunningWorkers != 1 || resp.PendingTasks != 3 {
		t.Fatalf("recent activity accumulation = %+v, want pendingPublishes=2 runningWorkers=1 pendingTasks=3", resp)
	}
}

func TestHandleHeartbeatIncludesLastError(t *testing.T) {
	hbMu.Lock()
	oldTimestamp := hbLastTimestamp
	oldStatus := hbLastStatus
	oldError := hbLastError
	hbMu.Unlock()
	t.Cleanup(func() {
		hbMu.Lock()
		hbLastTimestamp = oldTimestamp
		hbLastStatus = oldStatus
		hbLastError = oldError
		hbMu.Unlock()
	})

	wantError := "cli-get: heartbeat error (host=https://www.deepright.cn): heartbeat HTTP 502"
	recordHeartbeat(1, wantError)

	req := httptest.NewRequest(http.MethodGet, "/api/heartbeat", nil)
	rec := httptest.NewRecorder()
	handleHeartbeat().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp struct {
		Status    int    `json:"status"`
		LastError string `json:"lastError"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode heartbeat response: %v", err)
	}
	if resp.Status != 1 {
		t.Fatalf("status = %d, want 1", resp.Status)
	}
	if resp.LastError != wantError {
		t.Fatalf("lastError = %q, want %q", resp.LastError, wantError)
	}
}

func stubIntegrationReadyCheckForPID(t *testing.T, pidFile string) {
	t.Helper()
	oldReadyCheck := integrationReadyCheck
	integrationReadyCheck = func(string) bool {
		_, ok := readRunningPID(pidFile)
		return ok
	}
	t.Cleanup(func() {
		integrationReadyCheck = oldReadyCheck
	})
}

func findAgentMetadataInList(t *testing.T, metadata map[string]interface{}, agentID string) map[string]interface{} {
	t.Helper()
	agents, ok := metadata["agents"].([]interface{})
	if !ok || len(agents) == 0 {
		t.Fatalf("metadata.agents = %#v, want non-empty agent list", metadata["agents"])
	}
	for _, item := range agents {
		agent, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if strings.TrimSpace(fmt.Sprint(agent["agentId"])) == strings.TrimSpace(agentID) {
			return agent
		}
	}
	t.Fatalf("metadata.agents missing agent %q: %#v", agentID, agents)
	return nil
}

func integrationSkillNamesFromMetadataAgent(t *testing.T, agent map[string]interface{}) []string {
	t.Helper()
	items, ok := agent["skills"].([]interface{})
	if !ok {
		t.Fatalf("agent.skills = %#v, want []interface{}", agent["skills"])
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		skill, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		name := strings.TrimSpace(fmt.Sprint(skill["name"]))
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func TestBuildCLIRequestMetadataMapReportsSeedreamOnlyWhenConfigured(t *testing.T) {
	flushAgentCache()
	tmp := t.TempDir()

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
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
	oldCronDB := cronDB
	cronDB = db
	defer func() { cronDB = oldCronDB }()

	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "knowledge"), 0o755); err != nil {
		t.Fatalf("mkdir knowledge: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "config.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	readSkills := func() []string {
		metadata, err := getAgentOutput(agentRoot, "test-dev-cli-seedream", 0)
		if err != nil {
			t.Fatalf("getAgentOutput: %v", err)
		}
		metaMap := buildCLIRequestMetadataMap(metadata)
		agent := findAgentMetadataInList(t, metaMap, "A")
		return integrationSkillNamesFromMetadataAgent(t, agent)
	}

	skills := readSkills()
	for _, name := range skills {
		if name == seedreamInternalSkillName {
			t.Fatalf("cli/get skills without config = %v, want %q absent", skills, seedreamInternalSkillName)
		}
	}

	if err := writeTokenStoreConfigs(db, map[string]tokenConfig{
		seedreamProviderName: {Token: "seedream-token"},
	}); err != nil {
		t.Fatalf("write seedream token config: %v", err)
	}

	skills = readSkills()
	found := false
	for _, name := range skills {
		if name == seedreamInternalSkillName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("cli/get skills with defaults = %v, want %q present", skills, seedreamInternalSkillName)
	}
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

func TestReadLiveAgentKnowledgePrefersUppercaseThenLowercase(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()

	if got, ok := readLiveAgentKnowledge(workspace); ok || got != "" {
		t.Fatalf("readLiveAgentKnowledge() = (%q, %v), want empty/false when file missing", got, ok)
	}

	if err := os.WriteFile(filepath.Join(workspace, "knowledge.md"), []byte("lowercase"), 0o644); err != nil {
		t.Fatalf("write knowledge.md: %v", err)
	}
	if got, ok := readLiveAgentKnowledge(workspace); !ok || got != "lowercase" {
		t.Fatalf("readLiveAgentKnowledge() with lowercase = (%q, %v), want lowercase/true", got, ok)
	}

	if err := os.WriteFile(filepath.Join(workspace, "Knowledge.md"), []byte("uppercase"), 0o644); err != nil {
		t.Fatalf("write Knowledge.md: %v", err)
	}
	if got, ok := readLiveAgentKnowledge(workspace); !ok || got != "uppercase" {
		t.Fatalf("readLiveAgentKnowledge() with both files = (%q, %v), want uppercase/true", got, ok)
	}
}

func TestProxyChatCompletions(t *testing.T) {
	disableIntegrationNotificationsForTest(t)
	fixtureAgentDir, err := filepath.Abs("../agent/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	agentDir := filepath.Join(tmp, "agent")
	if err := copyTestDir(fixtureAgentDir, agentDir); err != nil {
		t.Fatalf("copy agent fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "knowledge"), 0o755); err != nil {
		t.Fatalf("mkdir knowledge: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "a", "knowledge.md"), []byte("agent a knowledge"), 0o644); err != nil {
		t.Fatalf("write knowledge.md: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	var capturedBody map[string]interface{}
	var capturedHeaders http.Header

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		f, _ := w.(http.Flusher)
		for _, chunk := range []string{
			`data: {"choices":[{"delta":{"role":"assistant","content":""}}]}`,
			`data: {"choices":[{"delta":{"content":"你好"}}]}`,
			`data: {"choices":[{"delta":{"content":"。"}}]}`,
			`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}`,
			`data: [DONE]`,
		} {
			fmt.Fprintf(w, "%s\n\n", chunk)
			f.Flush()
		}
	}))
	defer upstream.Close()

	cfg := &Config{
		Host:         upstream.URL,
		AgentDir:     agentDir,
		Device:       "test-dev",
		AgentCacheMs: 120000,
	}
	proxyClient := &http.Client{Timeout: 10 * time.Second}

	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, proxyClient)))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"HELLO WORLD"}],"stream":true}`
	req, _ := http.NewRequest("POST", server.URL, strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-test")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Verify headers preserved
	if capturedHeaders.Get("Authorization") != "Bearer sk-test" {
		t.Errorf("Authorization not preserved")
	}

	// Verify model/message unified
	if capturedBody["model"] != "gpt-4" {
		t.Errorf("model modified")
	}
	msgs, ok := capturedBody["messages"].([]interface{})
	if !ok || len(msgs) != 1 {
		t.Fatalf("messages modified: %v", capturedBody["messages"])
	}
	msg := msgs[0].(map[string]interface{})
	if msg["role"] != "user" || msg["content"] != "HELLO WORLD" {
		t.Errorf("content modified: %v", msg)
	}
	if _, exists := capturedBody["message"]; exists {
		t.Fatalf("message should not be forwarded: %v", capturedBody["message"])
	}
	if capturedBody["stream"] != true {
		t.Fatalf("stream = %v, want true", capturedBody["stream"])
	}

	// Verify metadata injected
	meta := capturedBody["metadata"].(map[string]interface{})
	if meta["deviceId"] != "test-dev" {
		t.Errorf("deviceId = %v", meta["deviceId"])
	}
	knowledge, ok := meta["knowledge"].(map[string]interface{})
	if !ok {
		t.Fatalf("knowledge missing from metadata: %#v", meta["knowledge"])
	}
	wantKnowledgePath, err := filepath.EvalSymlinks(filepath.Join(tmp, "knowledge"))
	if err != nil {
		wantKnowledgePath = filepath.Join(tmp, "knowledge")
	}
	gotKnowledgePath, _ := knowledge["path"].(string)
	gotResolved, err := filepath.EvalSymlinks(gotKnowledgePath)
	if err != nil {
		gotResolved = gotKnowledgePath
	}
	if gotResolved != wantKnowledgePath {
		t.Fatalf("knowledge.path = %q, want %q", gotResolved, wantKnowledgePath)
	}
	if value, exists := knowledge["lastUpdate"]; exists {
		got, ok := value.(float64)
		if !ok || got < 0 {
			t.Fatalf("knowledge.lastUpdate = %v, want non-negative number when present", value)
		}
	}
	if _, exists := knowledge["knowledgeCommit"]; exists {
		t.Fatalf("knowledge.knowledgeCommit should not be forwarded: %#v", knowledge)
	}
	if meta["agents"] == nil {
		t.Error("agents missing")
	}
	agents := meta["agents"].([]interface{})
	if len(agents) == 0 {
		t.Fatal("agents missing or empty")
	}
	var firstAgent map[string]interface{}
	for _, item := range agents {
		agentMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if agentMap["agentId"] == "a" {
			firstAgent = agentMap
			break
		}
	}
	if firstAgent == nil {
		t.Fatalf("agent a metadata missing: %#v", agents)
	}
	if firstAgent["provider"] != "" {
		t.Errorf("provider = %#v, want empty string for default test agent", firstAgent["provider"])
	}
	if firstAgent["knowledge"] != "agent a knowledge" {
		t.Fatalf("agent knowledge = %#v, want %q", firstAgent["knowledge"], "agent a knowledge")
	}

	// Verify SSE streamed content
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
	if content != "你好。" {
		t.Errorf("streamed content = %q, want 你好。", content)
	}
}

func TestProxyChatCompletionsLogsAbnormalWhenUpstreamStreamBreaksMidResponse(t *testing.T) {
	disableIntegrationNotificationsForTest(t)
	fixtureAgentDir, err := filepath.Abs("../agent/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	agentDir := filepath.Join(tmp, "agent")
	if err := copyTestDir(fixtureAgentDir, agentDir); err != nil {
		t.Fatalf("copy agent fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "knowledge"), 0o755); err != nil {
		t.Fatalf("mkdir knowledge: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	initCronDB()
	if cronDB == nil {
		t.Fatalf("cronDB not initialized")
	}
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()
	if err := ensureTokenStore(cronDB); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(cronDB, map[string]string{"gpt-4": "Bearer stored-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
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

	cfg := &Config{
		Host:         upstream.URL,
		AgentDir:     agentDir,
		Device:       "test-dev",
		AgentCacheMs: 120000,
	}
	proxyClient := &http.Client{Timeout: 3 * time.Second}
	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, proxyClient)))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"HELLO MIDSTREAM"}],"stream":true,"metadata":{"agentId":"a","chat":"chat-midstream"}}`
	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(reqBody))
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

	deadline := time.Now().Add(2 * time.Second)
	var abnormalCount int
	for time.Now().Before(deadline) {
		if err := cronDB.QueryRow(`SELECT COUNT(*) FROM chat_log WHERE agent_id = ? AND chat_id = ? AND role = 'A' AND response_type = 'abnormal' AND content LIKE ?`,
			"a", "chat-midstream", "SSE stream interrupted:%").Scan(&abnormalCount); err != nil {
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

func TestProxyChatCompletionsLogsAbnormalWhenUpstreamEndsWithoutDoneMarker(t *testing.T) {
	disableIntegrationNotificationsForTest(t)
	fixtureAgentDir, err := filepath.Abs("../agent/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	agentDir := filepath.Join(tmp, "agent")
	if err := copyTestDir(fixtureAgentDir, agentDir); err != nil {
		t.Fatalf("copy agent fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "knowledge"), 0o755); err != nil {
		t.Fatalf("mkdir knowledge: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	initCronDB()
	if cronDB == nil {
		t.Fatalf("cronDB not initialized")
	}
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()
	if err := ensureTokenStore(cronDB); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(cronDB, map[string]string{"gpt-4": "Bearer stored-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n")
	}))
	defer upstream.Close()

	cfg := &Config{
		Host:         upstream.URL,
		AgentDir:     agentDir,
		Device:       "test-dev",
		AgentCacheMs: 120000,
	}
	proxyClient := &http.Client{Timeout: 3 * time.Second}
	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, proxyClient)))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"HELLO EOF"}],"stream":true,"metadata":{"agentId":"a","chat":"chat-eof"}}`
	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(reqBody))
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

	deadline := time.Now().Add(2 * time.Second)
	var abnormalCount int
	for time.Now().Before(deadline) {
		if err := cronDB.QueryRow(`SELECT COUNT(*) FROM chat_log WHERE agent_id = ? AND chat_id = ? AND role = 'A' AND response_type = 'abnormal' AND content LIKE ?`,
			"a", "chat-eof", "SSE stream interrupted:%").Scan(&abnormalCount); err != nil {
			t.Fatalf("count abnormal chat_log: %v", err)
		}
		if abnormalCount > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if abnormalCount == 0 {
		t.Fatalf("expected abnormal restore log after EOF without done marker")
	}
}

func TestProxyChatCompletionsInjectsLastResponseMetadata(t *testing.T) {
	disableIntegrationNotificationsForTest(t)
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

	initCronDB()
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	expectedAt := time.Date(2026, time.July, 3, 11, 22, 33, 444000000, time.Local)
	if _, err := cronDB.Exec(`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES (?,?,?,?,?)`,
		"A", "chat-last-response", "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n", logTypeChatCompletionResponse, expectedAt.Format(roundLogStorageTimeLayout)); err != nil {
		t.Fatalf("insert agent_message_log: %v", err)
	}

	var capturedBody map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	cfg := &Config{
		Host:         upstream.URL,
		AgentDir:     agentDir,
		Device:       "test-dev-last-response",
		AgentCacheMs: 120000,
	}
	proxyClient := &http.Client{Timeout: 10 * time.Second}
	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, proxyClient)))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"HELLO LAST RESPONSE"}],"metadata":{"agentId":"A","chat":"chat-last-response"}}`
	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	meta, ok := capturedBody["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata missing: %#v", capturedBody["metadata"])
	}
	gotLastResponse, ok := meta["lastResponse"].(float64)
	if !ok || int64(gotLastResponse) != expectedAt.UnixMilli() {
		t.Fatalf("metadata.lastResponse = %#v, want %d", meta["lastResponse"], expectedAt.UnixMilli())
	}
}

func TestProxyChatCompletionsSendsNotificationOnCompletion(t *testing.T) {
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

	oldSupported := integrationNotificationSupportedFn
	oldSend := integrationSendNotificationFn
	t.Cleanup(func() {
		integrationNotificationSupportedFn = oldSupported
		integrationSendNotificationFn = oldSend
	})
	integrationNotificationSupportedFn = func() bool { return true }

	var got []integrationnotification.Options
	integrationSendNotificationFn = func(opts integrationnotification.Options) error {
		got = append(got, opts)
		return nil
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	cfg := &Config{
		Host:         upstream.URL,
		AgentDir:     agentDir,
		Device:       "test-dev",
		AgentCacheMs: 120000,
	}
	proxyClient := &http.Client{Timeout: 10 * time.Second}
	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, proxyClient)))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"HELLO"}],"stream":true}`
	req, _ := http.NewRequest("POST", server.URL, strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-test")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if len(got) != 1 {
		t.Fatalf("notification count = %d, want 1", len(got))
	}
	if got[0].Title != integrationCompletionNotificationTitle {
		t.Fatalf("notification title = %q, want %q", got[0].Title, integrationCompletionNotificationTitle)
	}
	if got[0].Message != "HELLO" {
		t.Fatalf("notification message = %q, want %q", got[0].Message, "HELLO")
	}
}

func TestBuildSSECompletionNotificationMessageUsesPromptSnippet(t *testing.T) {
	message, ok := buildSSECompletionNotificationMessage(chatTypePageSession, "", "请帮我总结这个很长的问题内容", false)
	if !ok {
		t.Fatal("buildSSECompletionNotificationMessage() ok = false, want true")
	}
	if message != "请帮我总结这个很长的..." {
		t.Fatalf("message = %q, want %q", message, "请帮我总结这个很长的...")
	}
}

func TestBuildSSECompletionNotificationMessageUsesPromptSnippetWhenAbnormal(t *testing.T) {
	message, ok := buildSSECompletionNotificationMessage(chatTypePageSession, "", "请帮我总结这个很长的问题内容", true)
	if !ok {
		t.Fatal("buildSSECompletionNotificationMessage() ok = false, want true")
	}
	if message != "请帮我总结这个很长的..." {
		t.Fatalf("message = %q, want %q", message, "请帮我总结这个很长的...")
	}
}

func TestProxyChatCompletionsSendsPromptNotificationOnAbnormalCompletion(t *testing.T) {
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

	oldSupported := integrationNotificationSupportedFn
	oldSend := integrationSendNotificationFn
	t.Cleanup(func() {
		integrationNotificationSupportedFn = oldSupported
		integrationSendNotificationFn = oldSend
	})
	integrationNotificationSupportedFn = func() bool { return true }

	var got []integrationnotification.Options
	integrationSendNotificationFn = func(opts integrationnotification.Options) error {
		got = append(got, opts)
		return nil
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "data: {\"choices\":[{\"finish_reason\":\"error\",\"delta\":{\"content\":\"boom\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	cfg := &Config{
		Host:         upstream.URL,
		AgentDir:     agentDir,
		Device:       "test-dev",
		AgentCacheMs: 120000,
	}
	proxyClient := &http.Client{Timeout: 10 * time.Second}
	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, proxyClient)))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"HELLO ABNORMAL"}],"stream":true}`
	req, _ := http.NewRequest("POST", server.URL, strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-test")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if len(got) != 1 {
		t.Fatalf("notification count = %d, want 1", len(got))
	}
	if got[0].Title != integrationCompletionNotificationTitle {
		t.Fatalf("notification title = %q, want %q", got[0].Title, integrationCompletionNotificationTitle)
	}
	if got[0].Message != "HELLO ABNO..." {
		t.Fatalf("notification message = %q, want %q", got[0].Message, "HELLO ABNO...")
	}
}

func TestExtractCompletionNotificationPromptUsesLastUserMessage(t *testing.T) {
	reqData := map[string]interface{}{
		"messages": []interface{}{
			map[string]interface{}{"role": "system", "content": "system"},
			map[string]interface{}{"role": "user", "content": "第一条问题"},
			map[string]interface{}{"role": "assistant", "content": "assistant"},
			map[string]interface{}{"role": "user", "content": "最后一条用户问题已经超过十个字"},
		},
	}

	got := extractCompletionNotificationPrompt(reqData)
	if got != "最后一条用户问题已经..." {
		t.Fatalf("prompt = %q, want %q", got, "最后一条用户问题已经...")
	}
}

func TestCronExecuteOnceSendsNotificationOnCompletion(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	if cronDB != nil {
		_ = cronDB.Close()
		cronDB = nil
	}
	flushAgentCache()

	oldSupported := integrationNotificationSupportedFn
	oldSend := integrationSendNotificationFn
	t.Cleanup(func() {
		integrationNotificationSupportedFn = oldSupported
		integrationSendNotificationFn = oldSend
	})
	integrationNotificationSupportedFn = func() bool { return true }

	var got []integrationnotification.Options
	integrationSendNotificationFn = func(opts integrationnotification.Options) error {
		got = append(got, opts)
		return nil
	}

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

	initCronDB()
	if cronDB == nil {
		t.Fatal("cronDB should not be nil")
	}
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	if err := ensureTokenStore(cronDB); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(cronDB, map[string]string{"OpenAI": "Bearer scheduled-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	metaRes, err := cronDB.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, chat_id, task_type, model, thinking, cron, content) VALUES (0, ?, 'A', 'chat-cron', 'cron', 'OpenAI', 0, '', '普通备忘录')`,
		time.Now().Format("2006-01-02 15:04"))
	if err != nil {
		t.Fatalf("insert task_meta: %v", err)
	}
	metaID, _ := metaRes.LastInsertId()
	if _, err := cronDB.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, content, started) VALUES (?, ?, 'A', 'chat-cron', 'cron', 'OpenAI', 0, '普通备忘录', 0)`,
		metaID, time.Now().Unix()); err != nil {
		t.Fatalf("insert task_detail: %v", err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	cfg := &Config{
		Host:         upstream.URL,
		AgentDir:     agentRoot,
		Device:       "dev",
		AgentCacheMs: 120000,
	}
	proxyClient := &http.Client{Timeout: 5 * time.Second}
	connectSvc, err := newIntegrationConnectService(agentRoot)
	if err != nil {
		t.Fatalf("newIntegrationConnectService: %v", err)
	}
	defer connectSvc.Close()

	cronExecuteOnce(cfg, proxyClient, connectSvc)
	time.Sleep(300 * time.Millisecond)

	if len(got) != 1 {
		t.Fatalf("notification count = %d, want 1", len(got))
	}
	if got[0].Title != integrationCompletionNotificationTitle {
		t.Fatalf("notification title = %q, want %q", got[0].Title, integrationCompletionNotificationTitle)
	}
	if got[0].Message != "备忘录任务已完成" {
		t.Fatalf("notification message = %q, want %q", got[0].Message, "备忘录任务已完成")
	}
}

func TestProxyChatCompletionsFallsBackToStoredTokenWhenAuthorizationMasked(t *testing.T) {
	agentDir, err := filepath.Abs("../agent/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}
	t.Setenv("HOME", t.TempDir())
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	dbPath := resolveIntegrationDBPath()
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatalf("mkdir db dir: %v", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := writeTokenStore(db, map[string]string{"openai": "Bearer stored-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	var capturedHeaders http.Header
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	cfg := &Config{
		Host:         upstream.URL,
		AgentDir:     agentDir,
		Device:       "test-dev-masked-token",
		AgentCacheMs: 120000,
	}
	proxyClient := &http.Client{Timeout: 10 * time.Second}
	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, proxyClient)))
	defer server.Close()

	reqBody := `{"model":"openai","messages":[{"role":"user","content":"HELLO MASKED"}],"stream":false}`
	req, _ := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", maskedTokenValue)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := strings.TrimSpace(capturedHeaders.Get("Authorization")); got != "Bearer stored-token" {
		t.Fatalf("Authorization = %q, want Bearer stored-token", got)
	}
}

func TestNextHeartbeatBackoff(t *testing.T) {
	tests := []struct {
		name    string
		current time.Duration
		base    time.Duration
		max     time.Duration
		want    time.Duration
	}{
		{name: "start from base when current is zero", current: 0, base: 3 * time.Second, max: 15 * time.Second, want: 3 * time.Second},
		{name: "double from base", current: 3 * time.Second, base: 3 * time.Second, max: 15 * time.Second, want: 6 * time.Second},
		{name: "double again", current: 6 * time.Second, base: 3 * time.Second, max: 15 * time.Second, want: 12 * time.Second},
		{name: "cap at max", current: 12 * time.Second, base: 3 * time.Second, max: 15 * time.Second, want: 15 * time.Second},
		{name: "keep max once reached", current: 15 * time.Second, base: 3 * time.Second, max: 15 * time.Second, want: 15 * time.Second},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := nextHeartbeatBackoff(tc.current, tc.base, tc.max); got != tc.want {
				t.Fatalf("nextHeartbeatBackoff(%v, %v, %v) = %v, want %v", tc.current, tc.base, tc.max, got, tc.want)
			}
		})
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

	cfg := &Config{
		Host:         upstream.URL,
		AgentDir:     agentDir,
		Device:       "test-dev-bool",
		AgentCacheMs: 120000,
	}
	proxyClient := &http.Client{Timeout: 10 * time.Second}
	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, proxyClient)))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"HELLO BOOL"}],"stream":false,"thinking":false,"html":true,"router_disable":false,"metadata":{"agentId":"A","chat":"chat-bool","thinking":false,"html":true,"router_disable":false}}`
	req, _ := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(reqBody))
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

func TestIntegrationChatCompletionsInjectsSelectedAgentVersionAndSandbox(t *testing.T) {
	disableIntegrationNotificationsForTest(t)
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
	if err := os.WriteFile(filepath.Join(agentDir, "config.json"), []byte(`{"version":"cron-v1"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "config.json"), []byte(`{"version":"cron-v1"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "config.json"), []byte(`{"version":"2026.06.10"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if cronDB != nil {
		_ = cronDB.Close()
		cronDB = nil
	}
	initCronDB()
	if cronDB == nil {
		t.Fatal("cronDB should not be nil")
	}
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()
	if _, err := sandboxstate.SetWithDirectory(cronDB, "A", "chat-version", sandboxstate.ModeFilePickNet, "/Users/demo/Desktop"); err != nil {
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

	cfg := &Config{
		Host:         upstream.URL,
		AgentDir:     agentRoot,
		Device:       "test-dev-version",
		AgentCacheMs: 120000,
	}
	proxyClient := &http.Client{Timeout: 10 * time.Second}
	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, proxyClient)))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"HELLO VERSION"}],"metadata":{"agentId":"A","chat":"chat-version","sandbox_path":"/tmp/forged"}}`
	req, _ := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(reqBody))
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
	if _, exists := meta["agent"]; exists {
		t.Fatalf("metadata.agent should be absent: %#v", meta["agent"])
	}
	agent := findAgentMetadataInList(t, meta, "A")
	if agent["version"] != "2026.06.10" {
		t.Fatalf("metadata.agents[A].version = %#v, want 2026.06.10", agent["version"])
	}
	if agent["sandbox"] != sandboxstate.ModeFilePickNet {
		t.Fatalf("metadata.agents[A].sandbox = %#v, want %q", agent["sandbox"], sandboxstate.ModeFilePickNet)
	}
	if meta["sandbox_path"] != "/Users/demo/Desktop" {
		t.Fatalf("metadata.sandbox_path = %#v, want /Users/demo/Desktop", meta["sandbox_path"])
	}
}

func TestIntegrationChatCompletionsOmitsSandboxPathWithoutDirectory(t *testing.T) {
	disableIntegrationNotificationsForTest(t)
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
	if err := os.WriteFile(filepath.Join(agentDir, "config.json"), []byte(`{"version":"2026.07.07"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if cronDB != nil {
		_ = cronDB.Close()
		cronDB = nil
	}
	initCronDB()
	if cronDB == nil {
		t.Fatal("cronDB should not be nil")
	}
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()
	if _, err := sandboxstate.Set(cronDB, "A", "chat-net", sandboxstate.ModeNet); err != nil {
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

	cfg := &Config{
		Host:         upstream.URL,
		AgentDir:     agentRoot,
		Device:       "test-dev-version",
		AgentCacheMs: 120000,
	}
	proxyClient := &http.Client{Timeout: 10 * time.Second}
	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, proxyClient)))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"HELLO"}],"metadata":{"agentId":"A","chat":"chat-net","sandbox_path":"/tmp/forged"}}`
	req, _ := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(reqBody))
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
	if _, exists := meta["sandbox_path"]; exists {
		t.Fatalf("metadata.sandbox_path should be absent: %#v", meta["sandbox_path"])
	}
}

func TestIntegrationChatCompletionsKeepsEmptySelectedAgentVersionCachedUntilAgentCacheExpires(t *testing.T) {
	disableIntegrationNotificationsForTest(t)
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
			agent := findAgentMetadataInList(t, meta, "A")
			if value, ok := agent["version"].(string); ok {
				version = value
			}
		}
		versions = append(versions, version)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	cfg := &Config{
		Host:         upstream.URL,
		AgentDir:     agentRoot,
		Device:       "test-dev-version-cache",
		AgentCacheMs: 120000,
	}
	proxyClient := &http.Client{Timeout: 10 * time.Second}
	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, proxyClient)))
	defer server.Close()

	send := func() {
		reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"HELLO VERSION CACHE"}],"metadata":{"agentId":"A"}}`
		req, _ := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(reqBody))
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
		t.Fatalf("first metadata.agents[A].version = %#v, want empty string", versions)
	}

	if err := os.WriteFile(configPath, []byte(`{"version":"2026.06.10"}`), 0o644); err != nil {
		t.Fatalf("rewrite config: %v", err)
	}

	send()
	if len(versions) != 2 || versions[1] != "" {
		t.Fatalf("second metadata.agents[A].version = %#v, want cached empty string", versions)
	}
}

func TestIntegrationChatCompletionsRefreshesMediaWithoutWaitingForAgentCache(t *testing.T) {
	disableIntegrationNotificationsForTest(t)
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
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var capturedBody map[string]interface{}
		_ = json.Unmarshal(body, &capturedBody)

		cover := ""
		if meta, ok := capturedBody["metadata"].(map[string]interface{}); ok {
			if _, exists := meta["agent"]; exists {
				t.Fatalf("metadata.agent should be absent: %#v", meta["agent"])
			}
			agent := findAgentMetadataInList(t, meta, "A")
			if media, ok := agent["media"].(map[string]interface{}); ok {
				if value, ok := media["cover"].(string); ok {
					cover = value
				}
			}
		}
		covers = append(covers, cover)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	cfg := &Config{
		Host:         upstream.URL,
		AgentDir:     agentRoot,
		Device:       "test-dev-media-cache",
		AgentCacheMs: 120000,
	}
	proxyClient := &http.Client{Timeout: 10 * time.Second}
	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, proxyClient)))
	defer server.Close()

	send := func() {
		reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"HELLO MEDIA CACHE"}],"metadata":{"agentId":"A"}}`
		req, _ := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(reqBody))
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
	if len(covers) != 1 || covers[0] != "first.png" {
		t.Fatalf("first metadata.agents[A].media.cover = %#v, want first.png", covers)
	}

	if err := os.WriteFile(configPath, []byte(`{"media":{"cover":"second.png"}}`), 0o644); err != nil {
		t.Fatalf("rewrite config: %v", err)
	}

	send()
	if len(covers) != 2 || covers[1] != "second.png" {
		t.Fatalf("second metadata.agents[A].media.cover = %#v, want second.png", covers)
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

func TestStopIntegrationPluginsUsesConnectPluginStatusAsSingleSourceOfTruth(t *testing.T) {
	oldListMeta := runConnectListMeta
	oldStatus := runConnectPluginStatus
	oldAction := runConnectPluginAction
	defer func() {
		runConnectListMeta = oldListMeta
		runConnectPluginStatus = oldStatus
		runConnectPluginAction = oldAction
	}()

	runConnectListMeta = func() ([]connectMetaConfigListItem, error) {
		return []connectMetaConfigListItem{
			{
				Key:      "browser",
				Name:     "浏览器",
				Callback: "/tmp/test-release/plugins/browser",
			},
		}, nil
	}

	statusCalls := 0
	runConnectPluginStatus = func(target string, flags map[string]string) (*connectsvc.PluginStatus, error) {
		statusCalls++
		if target != "browser" {
			t.Fatalf("status target = %q", target)
		}
		if got := strings.TrimSpace(flags["connect-bin"]); got == "" {
			t.Fatalf("expected connect-bin for status lookup")
		}
		return &connectsvc.PluginStatus{
			Key:     "browser",
			Name:    "浏览器",
			Path:    "/tmp/test-release/plugins/browser",
			PIDFile: "/tmp/test-release/plugins/browser.pid",
			Started: true,
			PID:     5408,
		}, nil
	}

	actionCalls := 0
	runConnectPluginAction = func(target, action string, flags map[string]string) (*connectsvc.PluginActionResult, error) {
		actionCalls++
		if target != "browser" {
			t.Fatalf("action target = %q", target)
		}
		if action != "stop" {
			t.Fatalf("action = %q, want stop", action)
		}
		if got := strings.TrimSpace(flags["connect-bin"]); got == "" {
			t.Fatalf("expected connect-bin for stop action")
		}
		return &connectsvc.PluginActionResult{
			Path:    "/tmp/test-release/plugins/browser",
			Command: []string{"stop", "--connect-bin", flags["connect-bin"]},
		}, nil
	}

	var logBuf bytes.Buffer
	warnings := stopIntegrationPlugins(&logBuf)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %+v, want none", warnings)
	}
	if statusCalls != 1 {
		t.Fatalf("statusCalls = %d, want 1", statusCalls)
	}
	if actionCalls != 1 {
		t.Fatalf("actionCalls = %d, want 1", actionCalls)
	}
	logText := logBuf.String()
	if !strings.Contains(logText, "stopping plugin name=浏览器 target=browser") {
		t.Fatalf("missing stopping log: %s", logText)
	}
	if !strings.Contains(logText, "plugin stopped name=浏览器 target=browser") {
		t.Fatalf("missing stopped log: %s", logText)
	}
}

func TestProxyChatCompletionsKnowledgeLastUpdateRemovedWithinInterval(t *testing.T) {
	disableIntegrationNotificationsForTest(t)
	disableIntegrationManagedRuntimeForTest(t)
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

	cfg := &Config{
		Host:                      upstream.URL,
		AgentDir:                  agentDir,
		Device:                    "test-dev-knowledge-interval",
		AgentCacheMs:              120000,
		KnowledgeUpdateIntervalMs: 7200000,
		KnowledgeUpdateLockMs:     1800000,
	}
	proxyClient := &http.Client{Timeout: 10 * time.Second}

	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, proxyClient)))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"hello"}],"stream":true}`
	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(reqBody))
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
		t.Fatalf("knowledge missing from metadata: %#v", meta["knowledge"])
	}
	if _, exists := knowledge["lastUpdate"]; exists {
		t.Fatalf("knowledge.lastUpdate should be removed within interval: %#v", knowledge)
	}
}

func TestProxyChatCompletionsKnowledgeLastUpdateLockedOnSecondRequest(t *testing.T) {
	disableIntegrationNotificationsForTest(t)
	disableIntegrationManagedRuntimeForTest(t)
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

	defer func() { _ = knowledgecore.CloseSharedDB() }()
	db, err := knowledgecore.OpenSharedDB(tmp)
	if err != nil {
		t.Fatalf("open knowledge db: %v", err)
	}
	lastUpdate := time.Now().Add(-3 * time.Hour).UnixMilli()
	if err := knowledgecore.SetLastUpdate(db, lastUpdate); err != nil {
		t.Fatalf("set knowledge last update: %v", err)
	}

	var capturedBodies []map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var captured map[string]interface{}
		_ = json.Unmarshal(body, &captured)
		capturedBodies = append(capturedBodies, captured)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	cfg := &Config{
		Host:                      upstream.URL,
		AgentDir:                  agentDir,
		Device:                    "test-dev-knowledge-lock",
		AgentCacheMs:              120000,
		KnowledgeUpdateIntervalMs: 7200000,
		KnowledgeUpdateLockMs:     1800000,
	}
	proxyClient := &http.Client{Timeout: 10 * time.Second}

	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, proxyClient)))
	defer server.Close()

	for i := 0; i < 2; i++ {
		reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"hello"}],"stream":true}`
		req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(reqBody))
		if err != nil {
			t.Fatalf("new request %d: %v", i+1, err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request %d failed: %v", i+1, err)
		}
		_ = resp.Body.Close()
	}

	if len(capturedBodies) != 2 {
		t.Fatalf("captured request count = %d, want 2", len(capturedBodies))
	}

	firstMeta, ok := capturedBodies[0]["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("first metadata missing: %#v", capturedBodies[0]["metadata"])
	}
	firstKnowledge, ok := firstMeta["knowledge"].(map[string]interface{})
	if !ok {
		t.Fatalf("first knowledge missing: %#v", firstMeta["knowledge"])
	}
	if got, ok := firstKnowledge["lastUpdate"].(float64); !ok || int64(got) != lastUpdate {
		t.Fatalf("first knowledge.lastUpdate = %#v, want %d", firstKnowledge["lastUpdate"], lastUpdate)
	}

	secondMeta, ok := capturedBodies[1]["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("second metadata missing: %#v", capturedBodies[1]["metadata"])
	}
	secondKnowledge, ok := secondMeta["knowledge"].(map[string]interface{})
	if !ok {
		t.Fatalf("second knowledge missing: %#v", secondMeta["knowledge"])
	}
	if _, exists := secondKnowledge["lastUpdate"]; exists {
		t.Fatalf("second knowledge.lastUpdate should be removed by lock window: %#v", secondKnowledge)
	}

	checkDB, err := sql.Open("sqlite", filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("open sqlite for lock check: %v", err)
	}
	defer checkDB.Close()

	if err := ensureKnowledgeRequestLockTable(checkDB); err != nil {
		t.Fatalf("ensure knowledge lock table: %v", err)
	}
	var lockTs int64
	if err := checkDB.QueryRow(`SELECT last_requested_at FROM knowledge_update_lock WHERE id = 1`).Scan(&lockTs); err != nil {
		t.Fatalf("query knowledge_update_lock: %v", err)
	}
	if lockTs <= 0 {
		t.Fatalf("knowledge_update_lock.last_requested_at = %d, want > 0", lockTs)
	}
}

func TestProxyChatCompletionsKnowledgeLastUpdateKeptWithinIntervalWhenKnowledgeCommitEnabled(t *testing.T) {
	disableIntegrationNotificationsForTest(t)
	disableIntegrationManagedRuntimeForTest(t)
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

	defer func() { _ = knowledgecore.CloseSharedDB() }()
	db, err := knowledgecore.OpenSharedDB(tmp)
	if err != nil {
		t.Fatalf("open knowledge db: %v", err)
	}
	agentID := "A"
	lastUpdate := time.Now().Add(-5 * time.Minute).UnixMilli()
	if err := knowledgecore.SetLastUpdateForAgent(db, agentID, lastUpdate); err != nil {
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

	cfg := &Config{
		Host:                      upstream.URL,
		AgentDir:                  agentDir,
		Device:                    "test-dev-knowledge-commit-interval",
		AgentCacheMs:              120000,
		KnowledgeUpdateIntervalMs: 7200000,
		KnowledgeUpdateLockMs:     1800000,
	}
	proxyClient := &http.Client{Timeout: 10 * time.Second}

	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, proxyClient)))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"hello"}],"stream":true,"metadata":{"agentId":"A","knowledge_commit":true,"knowledge":{}}}`
	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(reqBody))
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
		t.Fatalf("knowledge missing from metadata: %#v", meta["knowledge"])
	}
	gotLastUpdate, ok := knowledge["lastUpdate"].(float64)
	if !ok || int64(gotLastUpdate) != lastUpdate {
		t.Fatalf("knowledge.lastUpdate = %#v, want %d", knowledge["lastUpdate"], lastUpdate)
	}
	if _, exists := knowledge["knowledgeCommit"]; exists {
		t.Fatalf("knowledge.knowledgeCommit should not be forwarded: %#v", knowledge)
	}
}

func TestProxyChatCompletionsKnowledgeLastUpdateKeptWithinLockWindowWhenKnowledgeCommitEnabled(t *testing.T) {
	disableIntegrationNotificationsForTest(t)
	disableIntegrationManagedRuntimeForTest(t)
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

	defer func() { _ = knowledgecore.CloseSharedDB() }()
	db, err := knowledgecore.OpenSharedDB(tmp)
	if err != nil {
		t.Fatalf("open knowledge db: %v", err)
	}
	agentID := "A"
	lastUpdate := time.Now().Add(-3 * time.Hour).UnixMilli()
	if err := knowledgecore.SetLastUpdateForAgent(db, agentID, lastUpdate); err != nil {
		t.Fatalf("set knowledge last update: %v", err)
	}
	if err := ensureKnowledgeRequestLockTable(db); err != nil {
		t.Fatalf("ensure knowledge lock table: %v", err)
	}
	if err := sharedutil.EnsureKnowledgeRequestLockRow(db, agentID); err != nil {
		t.Fatalf("ensure knowledge lock row: %v", err)
	}
	lockTs := time.Now().Add(-10 * time.Minute).UnixMilli()
	if _, err := db.Exec(`UPDATE knowledge_update_lock SET last_requested_at = ? WHERE agent_id = ?`, lockTs, agentID); err != nil {
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

	cfg := &Config{
		Host:                      upstream.URL,
		AgentDir:                  agentDir,
		Device:                    "test-dev-knowledge-commit-lock",
		AgentCacheMs:              120000,
		KnowledgeUpdateIntervalMs: 7200000,
		KnowledgeUpdateLockMs:     1800000,
	}
	proxyClient := &http.Client{Timeout: 10 * time.Second}

	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, proxyClient)))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"hello"}],"stream":true,"metadata":{"agentId":"A","knowledge_commit":true,"knowledge":{}}}`
	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(reqBody))
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
		t.Fatalf("knowledge missing from metadata: %#v", meta["knowledge"])
	}
	gotLastUpdate, ok := knowledge["lastUpdate"].(float64)
	if !ok || int64(gotLastUpdate) != lastUpdate {
		t.Fatalf("knowledge.lastUpdate = %#v, want %d", knowledge["lastUpdate"], lastUpdate)
	}
	if _, exists := knowledge["knowledgeCommit"]; exists {
		t.Fatalf("knowledge.knowledgeCommit should not be forwarded: %#v", knowledge)
	}
}

func TestProxyChatCompletionsKnowledgeManualUpdatesKnowledgeTimestampsAfterSSEDone(t *testing.T) {
	disableIntegrationNotificationsForTest(t)
	disableIntegrationManagedRuntimeForTest(t)
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

	cfg := &Config{
		Host:                      upstream.URL,
		AgentDir:                  agentDir,
		Device:                    "test-dev-knowledge-manual",
		AgentCacheMs:              120000,
		KnowledgeUpdateIntervalMs: 7200000,
		KnowledgeUpdateLockMs:     1800000,
	}
	proxyClient := &http.Client{Timeout: 10 * time.Second}

	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, proxyClient)))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"整理完成"}],"stream":true,"metadata":{"knowledge_commit":true}}`
	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(reqBody))
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

func TestProxyChatCompletionsKnowledgeManualFalseDoesNotUpdateKnowledgeLastUpdateAfterSSEDone(t *testing.T) {
	disableIntegrationNotificationsForTest(t)
	disableIntegrationManagedRuntimeForTest(t)
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

	cfg := &Config{
		Host:                      upstream.URL,
		AgentDir:                  agentDir,
		Device:                    "test-dev-knowledge-manual-false",
		AgentCacheMs:              120000,
		KnowledgeUpdateIntervalMs: 7200000,
		KnowledgeUpdateLockMs:     1800000,
	}
	proxyClient := &http.Client{Timeout: 10 * time.Second}

	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, proxyClient)))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"普通请求"}],"stream":true,"metadata":{"knowledge_commit":false}}`
	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(reqBody))
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
	disableIntegrationNotificationsForTest(t)
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
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	cfg := &Config{
		Host:         upstream.URL,
		AgentDir:     agentDir,
		Device:       "test-dev-query",
		AgentCacheMs: 120000,
	}
	proxyClient := &http.Client{Timeout: 10 * time.Second}

	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, proxyClient)))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"HELLO QUERY"}],"stream":true,"metadata":{"hello":"world","extract":"true","bodyOnly":"kept"}}`
	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(reqBody))
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

func TestProxyChatCompletionsPrunesRedundantTopLevelAgentFieldsFromRequestMetadata(t *testing.T) {
	disableIntegrationNotificationsForTest(t)
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
	if err := os.MkdirAll(filepath.Join(agentDir, "skills"), 0o755); err != nil {
		t.Fatalf("mkdir skills: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("real soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("real user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "config.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var capturedBody map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	cfg := &Config{
		Host:         upstream.URL,
		AgentDir:     agentRoot,
		Device:       "test-dev-prune",
		AgentCacheMs: 120000,
	}
	proxyClient := &http.Client{Timeout: 10 * time.Second}
	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, proxyClient)))
	defer server.Close()

	reqBody := `{"model":"gpt-4","messages":[{"role":"user","content":"HELLO PRUNE"}],"stream":true,"metadata":{"agentId":"A","chat":"chat-prune","hello":"world","workspace":"/tmp/fake-workspace","skills":[{"name":"fake-skill"}],"media":{"type":"fake"},"soul":"fake soul","user":"fake user","dir":"/tmp/fake-dir"}}`
	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
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
		t.Fatalf("metadata not injected: %#v", capturedBody["metadata"])
	}
	if meta["hello"] != "world" {
		t.Fatalf("metadata.hello = %#v, want %q", meta["hello"], "world")
	}
	for _, key := range []string{"workspace", "skills", "media", "soul", "user", "dir"} {
		if _, exists := meta[key]; exists {
			t.Fatalf("metadata.%s should be removed, got %#v", key, meta[key])
		}
	}
	if _, exists := meta["agent"]; exists {
		t.Fatalf("metadata.agent should be removed, got %#v", meta["agent"])
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

	db, err := sql.Open("sqlite", resolveIntegrationDBPath())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := writeTokenStoreConfigs(db, map[string]tokenConfig{
		"deepseek": {
			Token:            "Bearer test-token",
			BaseURL:          "https://provider.example/v1",
			ModelFast:        "deepseek-fast",
			ModelThinking:    "",
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

	cfg := &Config{
		Host:         upstream.URL,
		AgentDir:     agentDir,
		Device:       "test-dev-provider",
		AgentCacheMs: 120000,
	}
	proxyClient := &http.Client{Timeout: 10 * time.Second}

	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, proxyClient)))
	defer server.Close()

	reqBody := `{"model":"deepseek","messages":[{"role":"user","content":"HELLO PROVIDER"}],"stream":true,"metadata":{"hello":"world"}}`
	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(reqBody))
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
	server := httptest.NewServer(handleInstallApp())
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

func TestIntegrationDetectInstallAppInstalledFindsCommonMacUserBin(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, ".local", "node", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin dir: %v", err)
	}
	npmPath := filepath.Join(binDir, "npm")
	if err := os.WriteFile(npmPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write npm: %v", err)
	}

	oldPlatformFn := integrationInstallAppPlatformKeyFn
	oldLookPathFn := integrationInstallAppLookPathFn
	oldUserHomeDirFn := integrationInstallAppUserHomeDirFn
	integrationInstallAppPlatformKeyFn = func() string { return "mac" }
	integrationInstallAppLookPathFn = func(string) (string, error) { return "", exec.ErrNotFound }
	integrationInstallAppUserHomeDirFn = func() (string, error) { return homeDir, nil }
	defer func() {
		integrationInstallAppPlatformKeyFn = oldPlatformFn
		integrationInstallAppLookPathFn = oldLookPathFn
		integrationInstallAppUserHomeDirFn = oldUserHomeDirFn
	}()

	if !integrationDetectInstallAppInstalled("npm") {
		t.Fatalf("integrationDetectInstallAppInstalled(%q) = false, want true", "npm")
	}
}

func TestHandleInstallAppMergesConfiguredEntries(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	writeRuntimeConfig(map[string]interface{}{
		"install_app": "node,git,node,python,python3",
	})

	oldDetectorFn := integrationInstallAppDetectorFn
	oldNowFn := integrationInstallAppNowFn
	integrationInstallAppDetectorFn = func(string) bool { return false }
	integrationInstallAppNowFn = func() time.Time {
		return time.Unix(1710000000, 0)
	}
	integrationResetInstallAppAvailabilityCache()
	defer func() {
		integrationInstallAppDetectorFn = oldDetectorFn
		integrationInstallAppNowFn = oldNowFn
		integrationResetInstallAppAvailabilityCache()
	}()

	server := httptest.NewServer(handleInstallApp())
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
	writeRuntimeConfig(map[string]interface{}{
		"install_app": "python,git,python3",
	})

	oldPlatformFn := integrationInstallAppPlatformKeyFn
	oldDetectorFn := integrationInstallAppDetectorFn
	oldNowFn := integrationInstallAppNowFn
	integrationInstallAppPlatformKeyFn = func() string { return "linux" }
	integrationInstallAppDetectorFn = func(name string) bool {
		return strings.EqualFold(strings.TrimSpace(name), "node")
	}
	integrationInstallAppNowFn = func() time.Time {
		return time.Unix(1710000000, 0)
	}
	integrationResetInstallAppAvailabilityCache()
	defer func() {
		integrationInstallAppPlatformKeyFn = oldPlatformFn
		integrationInstallAppDetectorFn = oldDetectorFn
		integrationInstallAppNowFn = oldNowFn
		integrationResetInstallAppAvailabilityCache()
	}()

	server := httptest.NewServer(handleInstallApp())
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
		t.Fatalf("install_app = %#v, want missing runtime entry kept", apps)
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

func TestIntegrationInstallAppDetectionCacheTTL(t *testing.T) {
	oldPlatformFn := integrationInstallAppPlatformKeyFn
	oldDetectorFn := integrationInstallAppDetectorFn
	oldNowFn := integrationInstallAppNowFn
	integrationInstallAppPlatformKeyFn = func() string { return "linux" }
	now := time.Unix(1710000000, 0)
	integrationInstallAppNowFn = func() time.Time { return now }
	calls := 0
	integrationInstallAppDetectorFn = func(name string) bool {
		calls++
		return strings.EqualFold(strings.TrimSpace(name), "node")
	}
	integrationResetInstallAppAvailabilityCache()
	defer func() {
		integrationInstallAppPlatformKeyFn = oldPlatformFn
		integrationInstallAppDetectorFn = oldDetectorFn
		integrationInstallAppNowFn = oldNowFn
		integrationResetInstallAppAvailabilityCache()
	}()

	if !integrationIsInstallAppInstalled("node") {
		t.Fatal("first install detection = false, want true")
	}
	if !integrationIsInstallAppInstalled("node") {
		t.Fatal("second install detection = false, want true")
	}
	if calls != 1 {
		t.Fatalf("detector calls = %d, want 1 within cache ttl", calls)
	}

	now = now.Add(4 * time.Minute)
	if !integrationIsInstallAppInstalled("node") {
		t.Fatal("cached install detection after 4 minutes = false, want true")
	}
	if calls != 1 {
		t.Fatalf("detector calls after 4 minutes = %d, want 1", calls)
	}

	now = now.Add(2 * time.Minute)
	if !integrationIsInstallAppInstalled("node") {
		t.Fatal("install detection after cache expiry = false, want true")
	}
	if calls != 2 {
		t.Fatalf("detector calls after cache expiry = %d, want 2", calls)
	}
}

func TestDetectPluginKeysRequiresConfiguredAndStartedPlugins(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, nil)
	defer restore()

	workdir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	pluginsDir := filepath.Clean(filepath.Join(workdir, "..", "plugins"))

	agentRoot := filepath.Join(workdir, "agent")
	if err := os.MkdirAll(filepath.Join(agentRoot, "A"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(workdir, "data")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"deepseek": "Bearer test-token"}); err != nil {
		_ = db.Close()
		t.Fatalf("writeTokenStore: %v", err)
	}
	_ = db.Close()
	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   dbPath,
		AgentDir: agentRoot,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("new connect service: %v", err)
	}
	defer svc.Close()

	pluginPath := filepath.Join(pluginsDir, "browser")
	writeIntegrationPluginScript(t, pluginPath, `{"key":"browser","name":"浏览器"}`, pluginParamJSON("cookie_path"))

	if _, err := svc.CreateMeta(connectsvc.MetaInput{
		Key:      "browser",
		Meta:     `{"cookie_path":""}`,
		Callback: "./browser",
		AgentID:  "A",
		ChatID:   "chat-browser",
		Model:    "deepseek",
	}); err != nil {
		t.Fatalf("create meta: %v", err)
	}

	if got := detectPluginKeys(); got != nil {
		t.Fatalf("plugins without pid = %#v, want nil", got)
	}

	writeIntegrationPluginPID(t, filepath.Join(pluginsDir, "browser.pid"), os.Getpid())
	if got := detectPluginKeys(); len(got) != 1 || got[0] != "browser" {
		t.Fatalf("plugins with pid = %#v, want [browser]", got)
	}
}

func TestDetectPluginKeysIgnoresBrowserPlaywrightPIDFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, nil)
	defer restore()

	workdir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	pluginsDir := filepath.Clean(filepath.Join(workdir, "..", "plugins"))

	agentRoot := filepath.Join(workdir, "agent")
	if err := os.MkdirAll(filepath.Join(agentRoot, "A"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(pluginsDir, ".browser_playwright"), 0o755); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(workdir, "data")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"deepseek": "Bearer test-token"}); err != nil {
		_ = db.Close()
		t.Fatalf("writeTokenStore: %v", err)
	}
	_ = db.Close()
	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   dbPath,
		AgentDir: agentRoot,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("new connect service: %v", err)
	}
	defer svc.Close()

	pluginPath := filepath.Join(pluginsDir, "browser")
	writeIntegrationPluginScript(t, pluginPath, `{"key":"browser","name":"浏览器"}`, pluginParamJSON("cookie_path"))

	if _, err := svc.CreateMeta(connectsvc.MetaInput{
		Key:      "browser",
		Meta:     `{"cookie_path":""}`,
		Callback: "./browser",
		AgentID:  "A",
		ChatID:   "chat-browser",
		Model:    "deepseek",
	}); err != nil {
		t.Fatalf("create meta: %v", err)
	}

	writeIntegrationPluginPID(t, filepath.Join(pluginsDir, ".browser_playwright", "browser_playwright.pid"), os.Getpid())
	if got := detectPluginKeys(); len(got) != 0 {
		t.Fatalf("plugins with browser_playwright pid = %#v, want []", got)
	}
}

func TestIntegrationMetadataRefreshesSkillsWhileAgentCacheIsWarm(t *testing.T) {
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

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
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

	cfg := &Config{
		Host:         upstream.URL,
		AgentDir:     agentDir,
		Device:       "device",
		AgentCacheMs: 3600000,
	}
	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, &http.Client{Timeout: 10 * time.Second})))
	defer server.Close()

	request := func() string {
		req, _ := http.NewRequest("POST", server.URL, strings.NewReader(`{"model":"gpt-4","messages":[{"role":"user","content":"hi"}],"stream":true}`))
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

func TestIntegrationMetadataNormalizesSkillCompatibilitySequence(t *testing.T) {
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

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
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

	cfg := &Config{
		Host:         upstream.URL,
		AgentDir:     agentDir,
		Device:       "device",
		AgentCacheMs: 3600000,
	}
	server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, &http.Client{Timeout: 10 * time.Second})))
	defer server.Close()

	req, _ := http.NewRequest("POST", server.URL, strings.NewReader(`{"model":"gpt-4","messages":[{"role":"user","content":"hi"}],"stream":true}`))
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

func TestHandleRawSupportsAbsolutePath(t *testing.T) {
	testDir := t.TempDir()
	homeDir := filepath.Join(testDir, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("mkdir home dir: %v", err)
	}
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	if err := os.MkdirAll(filepath.Join(testDir, "agent-a", "docs"), 0755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "agent-a", "docs", "hello.txt"), []byte("HELLO WORLD"), 0644); err != nil {
		t.Fatalf("write workspace file: %v", err)
	}
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

	cfg := &Config{
		AgentDir:     testDir,
		Device:       "test-dev",
		AgentCacheMs: 1,
	}
	server := httptest.NewServer(handleRaw(cfg))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/raw?agentId=agent-a&path=docs/hello.txt")
	if err != nil {
		t.Fatalf("GET relative path failed: %v", err)
	}
	var rr RawResponse
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		t.Fatalf("decode relative response: %v", err)
	}
	resp.Body.Close()
	if rr.Status != 0 {
		t.Fatalf("relative path: status = %d, content = %q", rr.Status, rr.Content)
	}

	resp2, err := http.Get(server.URL + "/api/raw?agentId=agent-a&path=" + url.QueryEscape(filepath.Join(testDir, "abs raw", "case file.bin")))
	if err != nil {
		t.Fatalf("GET absolute path failed: %v", err)
	}
	var rr2 RawResponse
	if err := json.NewDecoder(resp2.Body).Decode(&rr2); err != nil {
		t.Fatalf("decode absolute response: %v", err)
	}
	resp2.Body.Close()
	if rr2.Status != 0 {
		t.Fatalf("absolute path: status = %d, content = %q", rr2.Status, rr2.Content)
	}
	decoded2, err := base64.StdEncoding.DecodeString(rr2.Content)
	if err != nil {
		t.Fatalf("decode absolute base64: %v", err)
	}
	if string(decoded2) != "ABSOLUTE DATA" {
		t.Errorf("absolute path content = %q, want ABSOLUTE DATA", string(decoded2))
	}

	respQuoted, err := http.Get(server.URL + "/api/raw?agentId=agent-a&path=" + url.QueryEscape(`"`+filepath.Join(testDir, "abs raw", "case file.bin")+`"`))
	if err != nil {
		t.Fatalf("GET quoted absolute path failed: %v", err)
	}
	var rrQuoted RawResponse
	if err := json.NewDecoder(respQuoted.Body).Decode(&rrQuoted); err != nil {
		t.Fatalf("decode quoted absolute response: %v", err)
	}
	respQuoted.Body.Close()
	if rrQuoted.Status != 0 {
		t.Fatalf("quoted absolute path: status = %d, content = %q", rrQuoted.Status, rrQuoted.Content)
	}
	decodedQuoted, err := base64.StdEncoding.DecodeString(rrQuoted.Content)
	if err != nil {
		t.Fatalf("decode quoted absolute base64: %v", err)
	}
	if string(decodedQuoted) != "ABSOLUTE DATA" {
		t.Errorf("quoted absolute path content = %q, want ABSOLUTE DATA", string(decodedQuoted))
	}

	resp3, err := http.Get(server.URL + "/api/raw?agentId=agent-a&path=" + url.QueryEscape("~/demo.txt"))
	if err != nil {
		t.Fatalf("GET tilde path failed: %v", err)
	}
	var rr3 RawResponse
	if err := json.NewDecoder(resp3.Body).Decode(&rr3); err != nil {
		t.Fatalf("decode tilde response: %v", err)
	}
	resp3.Body.Close()
	if rr3.Status != 0 {
		t.Fatalf("tilde path: status = %d, content = %q", rr3.Status, rr3.Content)
	}
	decodedTilde, err := base64.StdEncoding.DecodeString(rr3.Content)
	if err != nil {
		t.Fatalf("decode tilde base64: %v", err)
	}
	if string(decodedTilde) != "HOME DATA" {
		t.Errorf("tilde path content = %q, want HOME DATA", string(decodedTilde))
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

	cfg := &Config{
		AgentDir:     testDir,
		Device:       "test-dev",
		AgentCacheMs: 1,
	}
	server := httptest.NewServer(handleFolder(cfg))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/folder?path=" + url.QueryEscape("~/nested"))
	if err != nil {
		t.Fatalf("GET absolute folder path failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("folder status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if openedPath != targetDir {
		t.Fatalf("opened path = %q, want %q", openedPath, targetDir)
	}
}

type testOpenSystemTargetFileInfo struct{}

func (testOpenSystemTargetFileInfo) Name() string       { return "explorer.exe" }
func (testOpenSystemTargetFileInfo) Size() int64        { return 1 }
func (testOpenSystemTargetFileInfo) Mode() os.FileMode  { return 0o755 }
func (testOpenSystemTargetFileInfo) ModTime() time.Time { return time.Time{} }
func (testOpenSystemTargetFileInfo) IsDir() bool        { return false }
func (testOpenSystemTargetFileInfo) Sys() interface{}   { return nil }

func TestOpenSystemTargetCommandWSLUsesExplorer(t *testing.T) {
	prevGOOS := openSystemTargetGOOS
	prevIsWSLFn := openSystemTargetIsWSLFn
	prevStatFn := openSystemTargetStatFn
	prevLookPathFn := openSystemTargetLookPathFn
	prevOutputFn := openSystemTargetOutputFn
	openSystemTargetGOOS = "linux"
	openSystemTargetIsWSLFn = func() bool { return true }
	openSystemTargetStatFn = func(path string) (os.FileInfo, error) {
		if path == "/mnt/c/Windows/explorer.exe" {
			return testOpenSystemTargetFileInfo{}, nil
		}
		return nil, os.ErrNotExist
	}
	openSystemTargetLookPathFn = func(name string) (string, error) {
		return "", os.ErrNotExist
	}
	openSystemTargetOutputFn = func(name string, args ...string) ([]byte, error) {
		if name != "wslpath" {
			t.Fatalf("unexpected command = %q", name)
		}
		if len(args) != 2 || args[0] != "-w" || args[1] != "/home/deepright" {
			t.Fatalf("unexpected args = %#v", args)
		}
		return []byte("\\\\wsl.localhost\\deepright\\home\\deepright\n"), nil
	}
	defer func() {
		openSystemTargetGOOS = prevGOOS
		openSystemTargetIsWSLFn = prevIsWSLFn
		openSystemTargetStatFn = prevStatFn
		openSystemTargetLookPathFn = prevLookPathFn
		openSystemTargetOutputFn = prevOutputFn
	}()

	cmd, args, err := openSystemTargetCommand("/home/deepright")
	if err != nil {
		t.Fatalf("openSystemTargetCommand returned error: %v", err)
	}
	if cmd != "/mnt/c/Windows/explorer.exe" {
		t.Fatalf("command = %q, want explorer.exe path", cmd)
	}
	if len(args) != 1 || args[0] != "\\\\wsl.localhost\\deepright\\home\\deepright" {
		t.Fatalf("args = %#v, want converted WSL path", args)
	}
}

func TestOpenSystemTargetWSLPrefersXdgOpen(t *testing.T) {
	prevGOOS := openSystemTargetGOOS
	prevIsWSLFn := openSystemTargetIsWSLFn
	prevRunFn := openSystemTargetRunCommandFn
	prevForegroundFn := openSystemTargetForegroundWSLFn
	prevStartFn := openSystemTargetStartCommandFn
	openSystemTargetGOOS = "linux"
	openSystemTargetIsWSLFn = func() bool { return true }
	runCalls := 0
	startCalls := 0
	openSystemTargetRunCommandFn = func(name string, args ...string) error {
		runCalls++
		if name != "xdg-open" {
			t.Fatalf("unexpected run command = %q", name)
		}
		if len(args) != 1 || args[0] != "/home/deepright" {
			t.Fatalf("unexpected run args = %#v", args)
		}
		return nil
	}
	openSystemTargetStartCommandFn = func(name string, args ...string) error {
		startCalls++
		return nil
	}
	openSystemTargetForegroundWSLFn = func(target string) error {
		t.Fatalf("foreground fallback should not run when xdg-open succeeds")
		return nil
	}
	defer func() {
		openSystemTargetGOOS = prevGOOS
		openSystemTargetIsWSLFn = prevIsWSLFn
		openSystemTargetRunCommandFn = prevRunFn
		openSystemTargetForegroundWSLFn = prevForegroundFn
		openSystemTargetStartCommandFn = prevStartFn
	}()

	if err := openSystemTarget("/home/deepright"); err != nil {
		t.Fatalf("openSystemTarget returned error: %v", err)
	}
	if runCalls != 1 {
		t.Fatalf("runCalls = %d, want 1", runCalls)
	}
	if startCalls != 0 {
		t.Fatalf("startCalls = %d, want 0", startCalls)
	}
}

func TestOpenSystemTargetWSLFallsBackWhenXdgOpenFails(t *testing.T) {
	prevGOOS := openSystemTargetGOOS
	prevIsWSLFn := openSystemTargetIsWSLFn
	prevRunFn := openSystemTargetRunCommandFn
	prevForegroundFn := openSystemTargetForegroundWSLFn
	prevStartFn := openSystemTargetStartCommandFn
	prevStatFn := openSystemTargetStatFn
	prevLookPathFn := openSystemTargetLookPathFn
	prevOutputFn := openSystemTargetOutputFn
	openSystemTargetGOOS = "linux"
	openSystemTargetIsWSLFn = func() bool { return true }
	runCalls := 0
	foregroundCalls := 0
	startCalls := 0
	startCmd := ""
	var startArgs []string
	openSystemTargetRunCommandFn = func(name string, args ...string) error {
		runCalls++
		return errors.New("xdg-open failed")
	}
	openSystemTargetForegroundWSLFn = func(target string) error {
		foregroundCalls++
		return errors.New("foreground failed")
	}
	openSystemTargetStartCommandFn = func(name string, args ...string) error {
		startCalls++
		startCmd = name
		startArgs = append([]string(nil), args...)
		return nil
	}
	openSystemTargetStatFn = func(path string) (os.FileInfo, error) {
		if path == "/mnt/c/Windows/explorer.exe" {
			return testOpenSystemTargetFileInfo{}, nil
		}
		return nil, os.ErrNotExist
	}
	openSystemTargetLookPathFn = func(name string) (string, error) {
		return "", os.ErrNotExist
	}
	openSystemTargetOutputFn = func(name string, args ...string) ([]byte, error) {
		return []byte("\\\\wsl.localhost\\deepright\\home\\deepright\n"), nil
	}
	defer func() {
		openSystemTargetGOOS = prevGOOS
		openSystemTargetIsWSLFn = prevIsWSLFn
		openSystemTargetRunCommandFn = prevRunFn
		openSystemTargetForegroundWSLFn = prevForegroundFn
		openSystemTargetStartCommandFn = prevStartFn
		openSystemTargetStatFn = prevStatFn
		openSystemTargetLookPathFn = prevLookPathFn
		openSystemTargetOutputFn = prevOutputFn
	}()

	if err := openSystemTarget("/home/deepright"); err != nil {
		t.Fatalf("openSystemTarget returned error: %v", err)
	}
	if runCalls != 1 {
		t.Fatalf("runCalls = %d, want 1", runCalls)
	}
	if foregroundCalls != 1 {
		t.Fatalf("foregroundCalls = %d, want 1", foregroundCalls)
	}
	if startCalls != 1 {
		t.Fatalf("startCalls = %d, want 1", startCalls)
	}
	if startCmd != "/mnt/c/Windows/explorer.exe" {
		t.Fatalf("startCmd = %q, want explorer.exe path", startCmd)
	}
	if len(startArgs) != 1 || startArgs[0] != "\\\\wsl.localhost\\deepright\\home\\deepright" {
		t.Fatalf("startArgs = %#v, want converted WSL path", startArgs)
	}
}

func TestOpenSystemTargetWSLForegroundOpenPreventsFallback(t *testing.T) {
	prevGOOS := openSystemTargetGOOS
	prevIsWSLFn := openSystemTargetIsWSLFn
	prevRunFn := openSystemTargetRunCommandFn
	prevForegroundFn := openSystemTargetForegroundWSLFn
	prevStartFn := openSystemTargetStartCommandFn
	openSystemTargetGOOS = "linux"
	openSystemTargetIsWSLFn = func() bool { return true }
	runCalls := 0
	foregroundCalls := 0
	startCalls := 0
	openSystemTargetRunCommandFn = func(name string, args ...string) error {
		runCalls++
		return errors.New("xdg-open failed")
	}
	openSystemTargetForegroundWSLFn = func(target string) error {
		foregroundCalls++
		if target != "/home/deepright" {
			t.Fatalf("target = %q, want /home/deepright", target)
		}
		return nil
	}
	openSystemTargetStartCommandFn = func(name string, args ...string) error {
		startCalls++
		return nil
	}
	defer func() {
		openSystemTargetGOOS = prevGOOS
		openSystemTargetIsWSLFn = prevIsWSLFn
		openSystemTargetRunCommandFn = prevRunFn
		openSystemTargetForegroundWSLFn = prevForegroundFn
		openSystemTargetStartCommandFn = prevStartFn
	}()

	if err := openSystemTarget("/home/deepright"); err != nil {
		t.Fatalf("openSystemTarget returned error: %v", err)
	}
	if runCalls != 1 {
		t.Fatalf("runCalls = %d, want 1", runCalls)
	}
	if foregroundCalls != 1 {
		t.Fatalf("foregroundCalls = %d, want 1", foregroundCalls)
	}
	if startCalls != 0 {
		t.Fatalf("startCalls = %d, want 0", startCalls)
	}
}

func TestOpenSystemTargetBuildForegroundScriptUsesWindowHandleActivation(t *testing.T) {
	script := openSystemTargetBuildForegroundScript(`\\wsl.localhost\deepright\home\deepright`)
	for _, needle := range []string{"Shell.Application", "SetForegroundWindow", "ShowWindowAsync", "Document.Folder.Self.Path"} {
		if !strings.Contains(script, needle) {
			t.Fatalf("script missing %q", needle)
		}
	}
}

func TestResolveFileTargetAbsoluteAndRelative(t *testing.T) {
	flushAgentCache()
	testDir := t.TempDir()
	homeDir := filepath.Join(testDir, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("mkdir home dir: %v", err)
	}
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	workspace := filepath.Join(testDir, "agent-a")
	if err := os.MkdirAll(filepath.Join(workspace, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	relFile := filepath.Join(workspace, "docs", "hello.txt")
	if err := os.WriteFile(relFile, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write relative file: %v", err)
	}
	absDir := filepath.Join(testDir, "Abs Last Update")
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		t.Fatalf("mkdir abs dir: %v", err)
	}
	absFile := filepath.Join(absDir, "Case File.md")
	if err := os.WriteFile(absFile, []byte("abs"), 0o644); err != nil {
		t.Fatalf("write absolute file: %v", err)
	}
	homeFile := filepath.Join(homeDir, "demo-home.txt")
	if err := os.WriteFile(homeFile, []byte("home"), 0o644); err != nil {
		t.Fatalf("write home file: %v", err)
	}

	cfg := &Config{AgentDir: testDir, Device: "test-dev", AgentCacheMs: 1}

	resolvedRel, err := resolveFileTarget(cfg, "agent-a", "docs/hello.txt")
	if err != nil {
		t.Fatalf("resolve relative path: %v", err)
	}
	relResolvedInfo, err := os.Stat(resolvedRel)
	if err != nil {
		t.Fatalf("stat resolved relative path: %v", err)
	}
	relExpectedInfo, err := os.Stat(relFile)
	if err != nil {
		t.Fatalf("stat expected relative path: %v", err)
	}
	if !os.SameFile(relResolvedInfo, relExpectedInfo) {
		t.Fatalf("relative path resolved to %q, want same file as %q", resolvedRel, relFile)
	}

	resolvedQuotedRel, err := resolveFileTarget(cfg, "agent-a", `"docs/hello.txt"`)
	if err != nil {
		t.Fatalf("resolve quoted relative path: %v", err)
	}
	quotedRelInfo, err := os.Stat(resolvedQuotedRel)
	if err != nil {
		t.Fatalf("stat resolved quoted relative path: %v", err)
	}
	if !os.SameFile(quotedRelInfo, relExpectedInfo) {
		t.Fatalf("quoted relative path resolved to %q, want same file as %q", resolvedQuotedRel, relFile)
	}

	resolvedAbs, err := resolveFileTarget(cfg, "", filepath.Join(testDir, "abs last update", "case file.MD"))
	if err != nil {
		t.Fatalf("resolve absolute path: %v", err)
	}
	absResolvedInfo, err := os.Stat(resolvedAbs)
	if err != nil {
		t.Fatalf("stat resolved absolute path: %v", err)
	}
	absExpectedInfo, err := os.Stat(absFile)
	if err != nil {
		t.Fatalf("stat expected absolute path: %v", err)
	}
	if !os.SameFile(absResolvedInfo, absExpectedInfo) {
		t.Fatalf("absolute path resolved to %q, want same file as %q", resolvedAbs, absFile)
	}

	resolvedQuotedAbs, err := resolveFileTarget(cfg, "", `"`+filepath.Join(testDir, "abs last update", "case file.MD")+`"`)
	if err != nil {
		t.Fatalf("resolve quoted absolute path: %v", err)
	}
	quotedAbsInfo, err := os.Stat(resolvedQuotedAbs)
	if err != nil {
		t.Fatalf("stat resolved quoted absolute path: %v", err)
	}
	if !os.SameFile(quotedAbsInfo, absExpectedInfo) {
		t.Fatalf("quoted absolute path resolved to %q, want same file as %q", resolvedQuotedAbs, absFile)
	}

	resolvedTilde, err := resolveFileTarget(cfg, "", "~/demo-home.txt")
	if err != nil {
		t.Fatalf("resolve tilde path: %v", err)
	}
	tildeResolvedInfo, err := os.Stat(resolvedTilde)
	if err != nil {
		t.Fatalf("stat resolved tilde path: %v", err)
	}
	tildeExpectedInfo, err := os.Stat(homeFile)
	if err != nil {
		t.Fatalf("stat expected tilde path: %v", err)
	}
	if !os.SameFile(tildeResolvedInfo, tildeExpectedInfo) {
		t.Fatalf("tilde path resolved to %q, want same file as %q", resolvedTilde, homeFile)
	}
}

func TestResolveFileTargetRelativeRequiresAgentID(t *testing.T) {
	flushAgentCache()
	cfg := &Config{AgentDir: t.TempDir(), Device: "test-dev", AgentCacheMs: 1}

	if _, err := resolveFileTarget(cfg, "", "docs/hello.txt"); err == nil || !strings.Contains(err.Error(), "agentId is required for relative path") {
		t.Fatalf("relative path without agentId error = %v, want agentId requirement", err)
	}
	if _, err := resolveFileTarget(cfg, "", "../escape.txt"); err == nil || !strings.Contains(err.Error(), "path escapes workspace") {
		t.Fatalf("escape path error = %v, want workspace escape", err)
	}
}

func TestFileLastUpdateDuration(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "demo.txt")
	if err := os.WriteFile(filePath, []byte("demo"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	modTime := time.Now().Add(-3 * time.Second).Truncate(time.Second)
	if err := os.Chtimes(filePath, modTime, modTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	got, err := fileLastUpdateDuration(filePath, modTime.Add(3500*time.Millisecond))
	if err != nil {
		t.Fatalf("fileLastUpdateDuration file: %v", err)
	}
	if got != 3500 {
		t.Fatalf("duration = %d, want 3500", got)
	}

	dirPath := filepath.Join(filepath.Dir(filePath), "dir")
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		t.Fatalf("mkdir dir: %v", err)
	}
	dirModTime := modTime.Add(2 * time.Second)
	if err := os.Chtimes(dirPath, dirModTime, dirModTime); err != nil {
		t.Fatalf("chtimes dir: %v", err)
	}
	gotDir, err := fileLastUpdateDuration(dirPath, dirModTime.Add(4500*time.Millisecond))
	if err != nil {
		t.Fatalf("fileLastUpdateDuration dir: %v", err)
	}
	if gotDir != 4500 {
		t.Fatalf("directory duration = %d, want 4500", gotDir)
	}
}

func TestHandleFileLastUpdate(t *testing.T) {
	flushAgentCache()
	testDir := t.TempDir()
	workspace := filepath.Join(testDir, "agent-a")
	if err := os.MkdirAll(filepath.Join(workspace, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	targetFile := filepath.Join(workspace, "docs", "hello.txt")
	if err := os.WriteFile(targetFile, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}
	modTime := time.Now().Add(-2 * time.Second).Truncate(time.Second)
	if err := os.Chtimes(targetFile, modTime, modTime); err != nil {
		t.Fatalf("chtimes target file: %v", err)
	}
	targetDir := filepath.Join(workspace, "docs", "folder")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}

	cfg := &Config{AgentDir: testDir, Device: "test-dev", AgentCacheMs: 1}
	server := httptest.NewServer(handleFileLastUpdate(cfg))
	defer server.Close()

	resp, err := http.Get(server.URL + "/file/lastUpdate?agentId=agent-a&file=docs/hello.txt")
	if err != nil {
		t.Fatalf("GET relative file failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("relative status = %d, body = %s", resp.StatusCode, string(body))
	}
	value, err := strconv.ParseInt(strings.TrimSpace(string(body)), 10, 64)
	if err != nil {
		t.Fatalf("parse relative response: %v", err)
	}
	if value < 1500 || value > 10000 {
		t.Fatalf("relative value = %d, want reasonable milliseconds", value)
	}

	resp2, err := http.Get(server.URL + "/file/lastUpdate?file=" + url.QueryEscape(filepath.Join(testDir, "AGENT-A", "DOCS", "HELLO.TXT")))
	if err != nil {
		t.Fatalf("GET absolute file failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("absolute status = %d, body = %s", resp2.StatusCode, string(body2))
	}

	resp3, err := http.Get(server.URL + "/file/lastUpdate?file=docs/hello.txt")
	if err != nil {
		t.Fatalf("GET missing agent failed: %v", err)
	}
	body3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()
	if resp3.StatusCode != http.StatusBadRequest || !strings.Contains(string(body3), "agentId is required for relative path") {
		t.Fatalf("missing agent status/body = %d/%q", resp3.StatusCode, string(body3))
	}

	resp4, err := http.Get(server.URL + "/file/lastUpdate?agent=agent-a&file=docs/folder")
	if err != nil {
		t.Fatalf("GET directory failed: %v", err)
	}
	body4, _ := io.ReadAll(resp4.Body)
	resp4.Body.Close()
	if resp4.StatusCode != http.StatusOK {
		t.Fatalf("directory status = %d, body = %q", resp4.StatusCode, string(body4))
	}
	dirValue, err := strconv.ParseInt(strings.TrimSpace(string(body4)), 10, 64)
	if err != nil {
		t.Fatalf("parse directory response: %v", err)
	}
	if dirValue < 0 {
		t.Fatalf("directory value = %d, want >= 0", dirValue)
	}
}

func TestCleanupIntegrationAgentBackupsMovesAndDeletesFiles(t *testing.T) {
	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	agentRoot := filepath.Join(t.TempDir(), "agents")
	workspaceA := filepath.Join(agentRoot, "A")
	workspaceB := filepath.Join(agentRoot, "B")
	if err := os.MkdirAll(workspaceA, 0o755); err != nil {
		t.Fatalf("mkdir workspaceA: %v", err)
	}
	if err := os.MkdirAll(workspaceB, 0o755); err != nil {
		t.Fatalf("mkdir workspaceB: %v", err)
	}

	mustWriteWithModTime := func(path string, body string, modTime time.Time) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir parent for %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		if err := os.Chtimes(path, modTime, modTime); err != nil {
			t.Fatalf("chtimes %s: %v", path, err)
		}
	}

	mustWriteWithModTime(filepath.Join(workspaceA, "USER.md"), "current-user", now.Add(-2*time.Hour))
	mustWriteWithModTime(filepath.Join(workspaceA, "SOUL.md"), "current-soul", now.Add(-2*time.Hour))
	mustWriteWithModTime(filepath.Join(workspaceA, "USER.md.bak"), "stale-user-backup", now.Add(-30*time.Hour))
	mustWriteWithModTime(filepath.Join(workspaceA, "Soul-20260701-120000.md"), "stale-soul-backup", now.Add(-36*time.Hour))
	mustWriteWithModTime(filepath.Join(workspaceA, "USER_20260703.md"), "fresh-user-backup", now.Add(-3*time.Hour))
	mustWriteWithModTime(filepath.Join(workspaceA, "USER_GUIDE.md"), "guide", now.Add(-96*time.Hour))
	mustWriteWithModTime(filepath.Join(workspaceA, "bak", "USER.md.bak"), "existing-bak", now.Add(-10*time.Hour))
	mustWriteWithModTime(filepath.Join(workspaceA, "bak", "SOUL.md.20260628.bak"), "too-old-in-bak", now.Add(-80*time.Hour))
	mustWriteWithModTime(filepath.Join(workspaceA, "bak", "SOUL.md.20260702.bak"), "fresh-in-bak", now.Add(-40*time.Hour))

	mustWriteWithModTime(filepath.Join(workspaceB, "SOUL_20260628_120000.md"), "very-old-root-backup", now.Add(-90*time.Hour))

	summary, err := cleanupIntegrationAgentBackups(agentRoot, now, 24*time.Hour, 72*time.Hour)
	if err != nil {
		t.Fatalf("cleanupIntegrationAgentBackups: %v", err)
	}
	if summary.AgentCount != 2 {
		t.Fatalf("agentCount = %d, want 2", summary.AgentCount)
	}
	if len(summary.Archived) != 3 {
		t.Fatalf("archived count = %d, want 3; summary=%+v", len(summary.Archived), summary)
	}
	if len(summary.Deleted) != 2 {
		t.Fatalf("deleted count = %d, want 2; summary=%+v", len(summary.Deleted), summary)
	}

	if _, err := os.Stat(filepath.Join(workspaceA, "USER.md")); err != nil {
		t.Fatalf("USER.md should remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceA, "SOUL.md")); err != nil {
		t.Fatalf("SOUL.md should remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceA, "USER_20260703.md")); err != nil {
		t.Fatalf("fresh backup should remain in workspace root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceA, "USER_GUIDE.md")); err != nil {
		t.Fatalf("USER_GUIDE.md should not be treated as backup: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceA, "USER.md.bak")); !os.IsNotExist(err) {
		t.Fatalf("stale root USER.md.bak should have been moved, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceA, "Soul-20260701-120000.md")); !os.IsNotExist(err) {
		t.Fatalf("stale root Soul backup should have been moved, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceB, "SOUL_20260628_120000.md")); !os.IsNotExist(err) {
		t.Fatalf("very old root backup should be removed from workspace root, err=%v", err)
	}

	if _, err := os.Stat(filepath.Join(workspaceA, "bak", "USER.md.bak")); err != nil {
		t.Fatalf("existing bak file should remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceA, "bak", "USER.md.1.bak")); err != nil {
		t.Fatalf("moved bak file should use collision-safe name: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceA, "bak", "Soul-20260701-120000.md")); err != nil {
		t.Fatalf("stale soul backup should move into bak: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceA, "bak", "SOUL.md.20260628.bak")); !os.IsNotExist(err) {
		t.Fatalf("old bak file should be deleted, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceA, "bak", "SOUL.md.20260702.bak")); err != nil {
		t.Fatalf("fresh bak file should remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceB, "bak", "SOUL_20260628_120000.md")); !os.IsNotExist(err) {
		t.Fatalf("very old file should also be deleted from bak, err=%v", err)
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

	cfg := &Config{AgentDir: agentRoot, Device: "test-dev", AgentCacheMs: 1}
	req := httptest.NewRequest(http.MethodGet, "/api/agent/create?agentId=agent-a&name="+url.QueryEscape("docs/data")+"&type=0", nil)
	rec := httptest.NewRecorder()
	handleAgentCreate(cfg)(rec, req)

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

	cfg := &Config{AgentDir: agentRoot, Device: "test-dev", AgentCacheMs: 1}
	req := httptest.NewRequest(http.MethodGet, "/api/agent/create?agentId=agent-a&name="+url.QueryEscape("../escape")+"&type=0", nil)
	rec := httptest.NewRecorder()
	handleAgentCreate(cfg)(rec, req)

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

func TestIsIntegrationConnectSubcommand(t *testing.T) {
	if !isIntegrationConnectSubcommand("meta-create") {
		t.Fatal("meta-create should be treated as top-level connect subcommand")
	}
	if !isIntegrationConnectSubcommand("response-list") {
		t.Fatal("response-list should be treated as top-level connect subcommand")
	}
	if !isIntegrationConnectSubcommand("list-plugins") {
		t.Fatal("list-plugins should be treated as top-level connect subcommand")
	}
	if isIntegrationConnectSubcommand("cron") {
		t.Fatal("cron should not be treated as top-level connect subcommand")
	}
	if isIntegrationConnectSubcommand("serve") {
		t.Fatal("serve should not be treated as top-level connect subcommand")
	}
}

func TestReadRuntimeConfigFallsBackToExecutableDirectory(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	runtimePath := filepath.Join(configDir, "config.json")
	payload := []byte("{\"agent-dir\":\"/tmp/agent\",\"app\":\"" + filepath.Join(tempDir, "integration") + "\",\"app-dir\":\"" + tempDir + "\"}\n")
	if err := os.WriteFile(runtimePath, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(tempDir, "integration"), nil
	}

	cfg := readRuntimeConfig()
	if cfg == nil {
		t.Fatal("expected runtime config")
	}
	if cfg["agent-dir"] != "/tmp/agent" {
		t.Fatalf("agent-dir = %q", cfg["agent-dir"])
	}
	if cfg["app-dir"] != tempDir {
		t.Fatalf("app-dir = %q", cfg["app-dir"])
	}
}

func TestReadRuntimeConfigPrefersExecutableDirectory(t *testing.T) {
	exeDir := t.TempDir()
	cwdDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(exeDir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(cwdDir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(exeDir, "config", "config.json"), []byte("{\"app-dir\":\"/exe-dir\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cwdDir, "config", "config.json"), []byte("{\"app-dir\":\"/cwd-dir\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(cwdDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(exeDir, "integration"), nil
	}

	cfg := readRuntimeConfig()
	if cfg == nil {
		t.Fatal("expected runtime config")
	}
	if cfg["app-dir"] != "/exe-dir" {
		t.Fatalf("app-dir = %q, want /exe-dir", cfg["app-dir"])
	}
}

func TestReadRuntimeConfigPrefersBundledResourcesDirectory(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("bundle runtime layout is only used on darwin")
	}

	bundleRoot := filepath.Join(t.TempDir(), "integration.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	resourcesDir := filepath.Join(bundleRoot, "Contents", "Resources")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(resourcesDir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resourcesDir, "config", "config.json"), []byte("{\"app-dir\":\"/bundle-resources\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(macOSDir, "integration"), nil
	}

	cfg := readRuntimeConfig()
	if cfg == nil {
		t.Fatal("expected runtime config")
	}
	if cfg["app-dir"] != "/bundle-resources" {
		t.Fatalf("app-dir = %q, want /bundle-resources", cfg["app-dir"])
	}
}

func TestReadRuntimeConfigReturnsNilWhenBundledConfigMissing(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("bundle runtime layout is only used on darwin")
	}

	bundleRoot := filepath.Join(t.TempDir(), "integration.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatal(err)
	}

	originalExecutable := integrationExecutableFn
	defer func() {
		integrationExecutableFn = originalExecutable
	}()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(macOSDir, "integration"), nil
	}

	if cfg := readRuntimeConfig(); len(cfg) != 0 {
		t.Fatalf("expected empty config when bundled config/config.json is missing, got %#v", cfg)
	}
}

func TestIntegrationBundleRuntimeBaseDirUsesFixedAppSupportDirectory(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("bundle runtime layout is only used on darwin")
	}

	bundleRoot := filepath.Join(t.TempDir(), "integration.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatal(err)
	}

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(macOSDir, "integration"), nil
	}

	originalRuntimeEnv := os.Getenv(integrationRuntimeDirEnv)
	defer func() {
		_ = os.Setenv(integrationRuntimeDirEnv, originalRuntimeEnv)
	}()
	if err := os.Unsetenv(integrationRuntimeDirEnv); err != nil {
		t.Fatal(err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	got := integrationBundleRuntimeBaseDir()
	want := integrationTestMacRuntimeBaseDir(home)
	if got != want {
		t.Fatalf("integrationBundleRuntimeBaseDir() = %q, want %q", got, want)
	}
}

func TestIntegrationBundleRuntimeBaseDirUsesFixedWSLHomeDirectory(t *testing.T) {
	homeDir := filepath.Join(t.TempDir(), "home", "tester")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldRuntimeGOOS := integrationRuntimeGOOS
	oldHomeFn := integrationUserHomeFn
	oldGetenv := integrationBrowserGetenvFn
	defer func() {
		integrationRuntimeGOOS = oldRuntimeGOOS
		integrationUserHomeFn = oldHomeFn
		integrationBrowserGetenvFn = oldGetenv
	}()

	integrationRuntimeGOOS = "linux"
	integrationUserHomeFn = func() (string, error) { return homeDir, nil }
	integrationBrowserGetenvFn = os.Getenv
	t.Setenv("WSL_DISTRO_NAME", "Ubuntu")
	t.Setenv(integrationRuntimeDirEnv, "")

	got := integrationBundleRuntimeBaseDir()
	want := filepath.Join(homeDir, integrationRuntimeAppDir)
	if got != want {
		t.Fatalf("integrationBundleRuntimeBaseDir() = %q, want %q", got, want)
	}
}

func TestIntegrationResourcesDirUsesBundleResources(t *testing.T) {
	bundleRoot := filepath.Join(t.TempDir(), "integration.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatal(err)
	}

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(macOSDir, "integration"), nil
	}

	got := integrationResourcesDir()
	want := filepath.Join(bundleRoot, "Contents", "Resources")
	if got != want {
		t.Fatalf("integrationResourcesDir() = %q, want %q", got, want)
	}
}

func TestIntegrationDefaultAgentDirUsesWSLRuntimeDirectory(t *testing.T) {
	homeDir := filepath.Join(t.TempDir(), "home", "tester")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldRuntimeGOOS := integrationRuntimeGOOS
	oldHomeFn := integrationUserHomeFn
	oldGetenv := integrationBrowserGetenvFn
	defer func() {
		integrationRuntimeGOOS = oldRuntimeGOOS
		integrationUserHomeFn = oldHomeFn
		integrationBrowserGetenvFn = oldGetenv
	}()

	integrationRuntimeGOOS = "linux"
	integrationUserHomeFn = func() (string, error) { return homeDir, nil }
	integrationBrowserGetenvFn = os.Getenv
	t.Setenv("WSL_DISTRO_NAME", "Ubuntu")
	t.Setenv("AGENT_DIR", "")

	got := integrationDefaultAgentDir()
	want := filepath.Join(homeDir, integrationRuntimeAppDir, "agent")
	if got != want {
		t.Fatalf("integrationDefaultAgentDir() = %q, want %q", got, want)
	}
}

func TestPrepareIntegrationRuntimeLayoutUsesBundledPluginsDir(t *testing.T) {
	bundleRoot := filepath.Join(t.TempDir(), "integration.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	pluginsDir := filepath.Join(bundleRoot, "Contents", "Resources", "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginsDir, "email"), []byte("plugin"), 0o755); err != nil {
		t.Fatal(err)
	}

	homeDir := t.TempDir()
	runtimeDir := integrationTestMacRuntimeBaseDir(homeDir)

	originalExecutable := integrationExecutableFn
	originalHomeFn := integrationUserHomeFn
	defer func() { integrationExecutableFn = originalExecutable }()
	defer func() { integrationUserHomeFn = originalHomeFn }()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(macOSDir, "integration"), nil
	}
	integrationUserHomeFn = func() (string, error) { return homeDir, nil }

	originalRuntimeEnv := os.Getenv(integrationRuntimeDirEnv)
	originalPluginEnv := os.Getenv(integrationPluginDirEnv)
	defer func() {
		_ = os.Setenv(integrationRuntimeDirEnv, originalRuntimeEnv)
		_ = os.Setenv(integrationPluginDirEnv, originalPluginEnv)
	}()
	if err := os.Setenv(integrationRuntimeDirEnv, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if err := os.Unsetenv(integrationPluginDirEnv); err != nil {
		t.Fatal(err)
	}

	if err := prepareIntegrationRuntimeLayout(); err != nil {
		t.Fatalf("prepareIntegrationRuntimeLayout() error = %v", err)
	}

	wantPluginDir := filepath.Join(runtimeDir, "plugins")
	if got := os.Getenv(integrationPluginDirEnv); got != wantPluginDir {
		t.Fatalf("plugin dir env = %q, want %q", got, wantPluginDir)
	}
	if data, err := os.ReadFile(filepath.Join(runtimeDir, "plugins", "email")); err != nil || string(data) != "plugin" {
		t.Fatalf("runtime plugin copy mismatch, err=%v data=%q", err, string(data))
	}
}

func TestCopyReleaseAssetWithPermissionsSkipsIdenticalExecutable(t *testing.T) {
	srcPath := filepath.Join(t.TempDir(), "browser")
	dstDir := t.TempDir()
	dstPath := filepath.Join(dstDir, "browser")
	payload := []byte("#!/bin/sh\necho browser\n")
	if err := os.WriteFile(srcPath, payload, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dstPath, payload, 0o755); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)

	if err := copyReleaseAssetWithPermissions(srcPath, dstPath, 0o755); err != nil {
		t.Fatalf("copyReleaseAssetWithPermissions() error = %v", err)
	}

	after, err := os.Stat(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Fatalf("mod time changed for identical file: before=%v after=%v", before.ModTime(), after.ModTime())
	}
}

func TestCopyReleaseAssetWithPermissionsReplacesContentAtomically(t *testing.T) {
	srcPath := filepath.Join(t.TempDir(), "browser")
	dstDir := t.TempDir()
	dstPath := filepath.Join(dstDir, "browser")
	if err := os.WriteFile(srcPath, []byte("#!/bin/sh\necho new\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dstPath, []byte("old"), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := copyReleaseAssetWithPermissions(srcPath, dstPath, 0o755); err != nil {
		t.Fatalf("copyReleaseAssetWithPermissions() error = %v", err)
	}

	data, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "#!/bin/sh\necho new\n" {
		t.Fatalf("copied data = %q", string(data))
	}
	info, err := os.Stat(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("mode = %o, want 755", info.Mode().Perm())
	}
	matches, err := filepath.Glob(filepath.Join(dstDir, "browser.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files were not cleaned up: %v", matches)
	}
}

func TestResolveIntegrationDBPathUsesFixedBundledRuntimeDirectoryBeforeRuntimeConfigExists(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("bundle runtime layout is only used on darwin")
	}

	bundleRoot := filepath.Join(t.TempDir(), "integration.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatal(err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(macOSDir, "integration"), nil
	}

	originalRuntimeEnv := os.Getenv(integrationRuntimeDirEnv)
	defer func() {
		_ = os.Setenv(integrationRuntimeDirEnv, originalRuntimeEnv)
	}()
	if err := os.Unsetenv(integrationRuntimeDirEnv); err != nil {
		t.Fatal(err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	got := resolveIntegrationDBPath()
	want := filepath.Join(integrationTestMacRuntimeBaseDir(home), "data")
	if got != want {
		t.Fatalf("resolveIntegrationDBPath() = %q, want %q", got, want)
	}
}

func TestInitCronDBUsesResolvedIntegrationDBPath(t *testing.T) {
	runtimeDir := t.TempDir()
	exeDir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	useIntegrationExecutableDir(t, exeDir)
	t.Setenv(integrationRuntimeDirEnv, runtimeDir)

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	if cronDB != nil {
		_ = cronDB.Close()
		cronDB = nil
	}
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	initCronDB()
	if cronDB == nil {
		t.Fatal("cronDB should not be nil")
	}
	if err := writeTokenStore(cronDB, map[string]string{"deepright": "Bearer runtime-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}
	if err := cronDB.Close(); err != nil {
		t.Fatalf("close cronDB: %v", err)
	}
	cronDB = nil

	runtimeDB, err := sql.Open("sqlite", filepath.Join(runtimeDir, "data"))
	if err != nil {
		t.Fatal(err)
	}
	defer runtimeDB.Close()

	var token string
	if err := runtimeDB.QueryRow(`SELECT token FROM token_store WHERE model = ?`, "deepright").Scan(&token); err != nil {
		t.Fatalf("query runtime token: %v", err)
	}
	if token != "Bearer runtime-token" {
		t.Fatalf("token = %q, want Bearer runtime-token", token)
	}

	if _, err := os.Stat(filepath.Join(exeDir, "data")); !os.IsNotExist(err) {
		t.Fatalf("legacy executable db should not be created, stat err = %v", err)
	}
}

func TestResolveIntegrationDBPathPrefersRuntimeDirectoryOverPluginWorkingDirectory(t *testing.T) {
	oldRuntimeGOOS := integrationRuntimeGOOS
	integrationRuntimeGOOS = "linux"
	t.Setenv("WSL_DISTRO_NAME", "")
	t.Setenv(integrationRuntimeDirEnv, "")
	defer func() { integrationRuntimeGOOS = oldRuntimeGOOS }()

	tempDir := t.TempDir()
	releaseDir := filepath.Join(tempDir, "release")
	pluginsDir := filepath.Join(releaseDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configDir := filepath.Join(releaseDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	startupConfigPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(startupConfigPath, []byte("{\"port\":8080,\"app-dir\":\""+releaseDir+"\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(releaseDir, "data"), []byte("main-db"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginsDir, "data"), []byte("plugin-db"), 0o644); err != nil {
		t.Fatal(err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(pluginsDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(releaseDir, "integration"), nil
	}

	got := resolveIntegrationDBPath()
	want := filepath.Join(releaseDir, "data")
	if got != want {
		t.Fatalf("resolveIntegrationDBPath() = %q, want %q", got, want)
	}
}

func TestMergeIntegrationRuntimeFlagsPrefersStartupConfigOverRuntimeHost(t *testing.T) {
	tempDir := t.TempDir()
	releaseDir := filepath.Join(tempDir, "release")
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configDir := filepath.Join(releaseDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte("{\"port\":8081,\"host\":\"http://bundle.example.com\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(releaseDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(releaseDir, "integration"), nil
	}

	merged := mergeIntegrationRuntimeFlags(map[string]string{})
	if got := strings.TrimSpace(merged["port"]); got != "" {
		t.Fatalf("merged port = %q, want empty so command default stays 8080", got)
	}
	if got := strings.TrimSpace(merged["host"]); got != "http://bundle.example.com" {
		t.Fatalf("merged host = %q, want startup config host", got)
	}
}

func TestStartIntegrationProcessLogsToFileOnly(t *testing.T) {
	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "integration.pid")
	logFile := filepath.Join(tempDir, "integration.log")
	scriptPath := filepath.Join(tempDir, "fake-integration.sh")
	flags := map[string]string{
		"pid-file":  pidFile,
		"log-file":  logFile,
		"port":      "8080",
		"agent-dir": filepath.Join(tempDir, "agent"),
		"site":      filepath.Join(tempDir, "site"),
	}
	script := "#!/bin/sh\n" +
		"pid_file=''\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  if [ \"$1\" = \"--pid-file\" ]; then\n" +
		"    shift\n" +
		"    pid_file=\"$1\"\n" +
		"  fi\n" +
		"  shift\n" +
		"done\n" +
		"if [ -n \"$pid_file\" ]; then\n" +
		"  printf '%s' \"$$\" > \"$pid_file\"\n" +
		"fi\n" +
		"trap 'rm -f \"$pid_file\"; exit 0' TERM INT\n" +
		"while :; do sleep 1; done\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake integration script: %v", err)
	}

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	integrationExecutableFn = func() (string, error) {
		return scriptPath, nil
	}
	stubIntegrationReadyCheckForPID(t, pidFile)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := startIntegrationProcess(flags, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr=%s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	logContent, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	logText := string(logContent)
	if !strings.Contains(logText, "start requested") {
		t.Fatalf("log file missing start requested entry: %s", logText)
	}
	if !strings.Contains(logText, "service started") {
		t.Fatalf("log file missing started entry: %s", logText)
	}

	pid, ok := readRunningPID(pidFile)
	if !ok {
		t.Fatalf("expected running pid in %s", pidFile)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		t.Fatalf("find process: %v", err)
	}
	_ = proc.Signal(syscall.SIGTERM)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := readRunningPID(pidFile); !ok {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("process pid=%d did not exit in time", pid)
}

func TestStopIntegrationProcessLogsToFileOnlyWhenAlreadyStopped(t *testing.T) {
	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "integration.pid")
	logFile := filepath.Join(tempDir, "integration.log")
	flags := map[string]string{
		"pid-file": pidFile,
		"log-file": logFile,
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := stopIntegrationProcess(flags, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr=%s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	logContent, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	logText := string(logContent)
	if !strings.Contains(logText, "stop requested") {
		t.Fatalf("log file missing stop requested entry: %s", logText)
	}
	if !strings.Contains(logText, "service already stopped") {
		t.Fatalf("log file missing already stopped entry: %s", logText)
	}
}

func TestIntegrationDailyLogWriterRotatesAcrossDays(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "integration.log")
	currentTime := time.Date(2026, 7, 7, 23, 59, 0, 0, time.Local)

	writer, err := newIntegrationDailyLogWriterWithClock(logFile, integrationLogRetentionDays, func() time.Time {
		return currentTime
	})
	if err != nil {
		t.Fatalf("newIntegrationDailyLogWriterWithClock: %v", err)
	}
	defer writer.Close()

	if _, err := writer.Write([]byte("day-1\n")); err != nil {
		t.Fatalf("write day-1: %v", err)
	}

	currentTime = currentTime.Add(2 * time.Minute)
	if _, err := writer.Write([]byte("day-2\n")); err != nil {
		t.Fatalf("write day-2: %v", err)
	}

	currentData, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read current log: %v", err)
	}
	if got := string(currentData); got != "day-2\n" {
		t.Fatalf("current log = %q, want %q", got, "day-2\n")
	}

	archivePath := integrationArchivedLogPath(logFile, "2026-07-07")
	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read archive log: %v", err)
	}
	if got := string(archiveData); got != "day-1\n" {
		t.Fatalf("archive log = %q, want %q", got, "day-1\n")
	}
}

func TestIntegrationDailyLogWriterArchivesStaleBaseAndCleansOldArchives(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "integration.log")
	staleArchivePath := integrationArchivedLogPath(logFile, "2026-07-06")
	oldArchivePath := integrationArchivedLogPath(logFile, "2026-07-03")

	if err := os.WriteFile(staleArchivePath, []byte("archived-day-6\n"), 0o644); err != nil {
		t.Fatalf("write existing archive: %v", err)
	}
	if err := os.WriteFile(oldArchivePath, []byte("expired\n"), 0o644); err != nil {
		t.Fatalf("write old archive: %v", err)
	}
	if err := os.WriteFile(logFile, []byte("stale-base\n"), 0o644); err != nil {
		t.Fatalf("write stale base log: %v", err)
	}

	staleTime := time.Date(2026, 7, 6, 21, 0, 0, 0, time.Local)
	if err := os.Chtimes(logFile, staleTime, staleTime); err != nil {
		t.Fatalf("chtimes stale base log: %v", err)
	}

	writer, err := newIntegrationDailyLogWriterWithClock(logFile, integrationLogRetentionDays, func() time.Time {
		return time.Date(2026, 7, 7, 9, 0, 0, 0, time.Local)
	})
	if err != nil {
		t.Fatalf("newIntegrationDailyLogWriterWithClock: %v", err)
	}
	defer writer.Close()

	if _, err := writer.Write([]byte("today\n")); err != nil {
		t.Fatalf("write today log: %v", err)
	}

	currentData, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read current log: %v", err)
	}
	if got := string(currentData); got != "today\n" {
		t.Fatalf("current log = %q, want %q", got, "today\n")
	}

	staleArchiveData, err := os.ReadFile(staleArchivePath)
	if err != nil {
		t.Fatalf("read stale archive: %v", err)
	}
	if got := string(staleArchiveData); got != "archived-day-6\nstale-base\n" {
		t.Fatalf("stale archive = %q, want %q", got, "archived-day-6\nstale-base\n")
	}

	if _, err := os.Stat(oldArchivePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old archive stat err = %v, want not exist", err)
	}
}

func TestStopIntegrationProcessPreservesStartupConfigWhenAlreadyStopped(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	startupConfigPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(startupConfigPath, []byte("{\"host\":\"http://127.0.0.1:9998\"}\n"), 0o644); err != nil {
		t.Fatalf("write config/config.json: %v", err)
	}

	originalExecutable := integrationExecutableFn
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(tempDir, "integration"), nil
	}
	defer func() { integrationExecutableFn = originalExecutable }()

	pidFile := filepath.Join(tempDir, "integration.pid")
	logFile := filepath.Join(tempDir, "integration.log")
	flags := map[string]string{
		"pid-file": pidFile,
		"log-file": logFile,
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := stopIntegrationProcess(flags, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(startupConfigPath); err != nil {
		t.Fatalf("config/config.json should be preserved, stat err=%v", err)
	}
}

func TestStartIntegrationProcessCleansStartupPIDFilesBeforeLaunch(t *testing.T) {
	tempDir := t.TempDir()
	useIntegrationExecutableDir(t, tempDir)
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	pidFile := filepath.Join(tempDir, "integration.pid")
	logFile := filepath.Join(tempDir, "integration.log")
	scriptPath := filepath.Join(tempDir, "fake-integration.sh")
	stalePluginPID := filepath.Join(integrationStartupDir(pidFile), "browser.pid")
	if err := os.WriteFile(stalePluginPID, []byte("12345"), 0o644); err != nil {
		t.Fatalf("write stale pid: %v", err)
	}
	flags := map[string]string{
		"pid-file":  pidFile,
		"log-file":  logFile,
		"port":      strconv.Itoa(integrationServicePort),
		"agent-dir": filepath.Join(tempDir, "agent"),
		"site":      filepath.Join(tempDir, "site"),
	}
	script := "#!/bin/sh\n" +
		"pid_file=''\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  if [ \"$1\" = \"--pid-file\" ]; then\n" +
		"    shift\n" +
		"    pid_file=\"$1\"\n" +
		"  fi\n" +
		"  shift\n" +
		"done\n" +
		"if [ -n \"$pid_file\" ]; then\n" +
		"  printf '%s' \"$$\" > \"$pid_file\"\n" +
		"fi\n" +
		"trap 'rm -f \"$pid_file\"; exit 0' TERM INT\n" +
		"while :; do sleep 1; done\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake integration script: %v", err)
	}

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	integrationExecutableFn = func() (string, error) {
		return scriptPath, nil
	}
	stubIntegrationReadyCheckForPID(t, pidFile)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := startIntegrationProcess(flags, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(stalePluginPID); !os.IsNotExist(err) {
		t.Fatalf("stale plugin pid should be removed, stat err=%v", err)
	}
	if _, ok := readRunningPID(pidFile); !ok {
		t.Fatalf("expected running pid in %s", pidFile)
	}
	pid, _ := readRunningPID(pidFile)
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Signal(syscall.SIGTERM)
	}
}

func TestStartIntegrationProcessOpensBrowserWhenAlreadyRunning(t *testing.T) {
	oldTokenFn := integrationSiteVersionTokenFn
	integrationSiteVersionTokenFn = func() string { return "site-version-token" }
	defer func() { integrationSiteVersionTokenFn = oldTokenFn }()

	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "integration.pid")
	logFile := filepath.Join(tempDir, "integration.log")
	flags := map[string]string{
		"pid-file": pidFile,
		"log-file": logFile,
		"port":     strconv.Itoa(integrationServicePort),
	}
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}

	integrationReadyCheck = func(string) bool { return true }
	oldBrowserEntryReadyCheck := integrationBrowserEntryReadyCheck
	integrationBrowserEntryReadyCheck = func(int) bool { return true }
	defer func() { integrationBrowserEntryReadyCheck = oldBrowserEntryReadyCheck }()

	var opened string
	oldOpenBrowser := integrationOpenBrowserFn
	integrationOpenBrowserFn = func(target string) error {
		opened = target
		return nil
	}
	defer func() { integrationOpenBrowserFn = oldOpenBrowser }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := startIntegrationProcess(flags, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr=%s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if opened != integrationBrowserOpenURL(8080) {
		t.Fatalf("opened = %q, want browser url", opened)
	}

	logContent, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	logText := string(logContent)
	if !strings.Contains(logText, "start reused running service") {
		t.Fatalf("log file missing reused running service entry: %s", logText)
	}
}

func TestStartIntegrationProcessOpensBrowserWhenServiceReadyWithoutPIDFile(t *testing.T) {
	oldTokenFn := integrationSiteVersionTokenFn
	integrationSiteVersionTokenFn = func() string { return "site-version-token" }
	defer func() { integrationSiteVersionTokenFn = oldTokenFn }()

	tempDir := t.TempDir()
	useIntegrationExecutableDir(t, tempDir)
	pidFile := filepath.Join(tempDir, "integration.pid")
	logFile := filepath.Join(tempDir, "integration.log")
	flags := map[string]string{
		"pid-file": pidFile,
		"log-file": logFile,
		"port":     strconv.Itoa(integrationServicePort),
	}

	integrationReadyCheck = func(string) bool { return true }
	oldBrowserEntryReadyCheck := integrationBrowserEntryReadyCheck
	integrationBrowserEntryReadyCheck = func(int) bool { return true }
	defer func() { integrationBrowserEntryReadyCheck = oldBrowserEntryReadyCheck }()

	var opened string
	oldOpenBrowser := integrationOpenBrowserFn
	integrationOpenBrowserFn = func(target string) error {
		opened = target
		return nil
	}
	defer func() { integrationOpenBrowserFn = oldOpenBrowser }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := startIntegrationProcess(flags, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr=%s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if opened != integrationBrowserOpenURL(8080) {
		t.Fatalf("opened = %q, want browser url", opened)
	}

	logContent, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	logText := string(logContent)
	if !strings.Contains(logText, "start reused running service without pid-file") {
		t.Fatalf("log file missing reused running service without pid entry: %s", logText)
	}
}

func TestOpenExistingIntegrationBrowserIfReadyWaitsForBrowserEntry(t *testing.T) {
	oldTokenFn := integrationSiteVersionTokenFn
	integrationSiteVersionTokenFn = func() string { return "site-version-token" }
	defer func() { integrationSiteVersionTokenFn = oldTokenFn }()

	oldReadyCheck := integrationReadyCheck
	oldBrowserEntryReadyCheck := integrationBrowserEntryReadyCheck
	oldOpenBrowser := integrationOpenBrowserFn
	oldBrowserEntryWait := integrationBrowserEntryWait
	oldBrowserEntryPollInterval := integrationBrowserEntryPollInterval
	defer func() {
		integrationReadyCheck = oldReadyCheck
		integrationBrowserEntryReadyCheck = oldBrowserEntryReadyCheck
		integrationOpenBrowserFn = oldOpenBrowser
		integrationBrowserEntryWait = oldBrowserEntryWait
		integrationBrowserEntryPollInterval = oldBrowserEntryPollInterval
	}()

	integrationReadyCheck = func(string) bool { return true }
	var entryChecks int
	integrationBrowserEntryReadyCheck = func(int) bool {
		entryChecks++
		return entryChecks >= 3
	}
	var opened string
	integrationOpenBrowserFn = func(target string) error {
		opened = target
		return nil
	}
	integrationBrowserEntryWait = 100 * time.Millisecond
	integrationBrowserEntryPollInterval = time.Millisecond

	var stderr bytes.Buffer
	if ok := openExistingIntegrationBrowserIfReady(18084, nil, &stderr, ""); !ok {
		t.Fatal("openExistingIntegrationBrowserIfReady() = false, want true")
	}
	if opened != integrationBrowserOpenURL(18084) {
		t.Fatalf("opened = %q, want browser url", opened)
	}
	if entryChecks < 3 {
		t.Fatalf("entryChecks = %d, want at least 3", entryChecks)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestOpenExistingIntegrationBrowserIfReadyOpensWhenBrowserEntryWaitTimesOut(t *testing.T) {
	oldTokenFn := integrationSiteVersionTokenFn
	integrationSiteVersionTokenFn = func() string { return "site-version-token" }
	defer func() { integrationSiteVersionTokenFn = oldTokenFn }()

	oldReadyCheck := integrationReadyCheck
	oldBrowserEntryReadyCheck := integrationBrowserEntryReadyCheck
	oldOpenBrowser := integrationOpenBrowserFn
	oldBrowserEntryWait := integrationBrowserEntryWait
	oldBrowserEntryPollInterval := integrationBrowserEntryPollInterval
	defer func() {
		integrationReadyCheck = oldReadyCheck
		integrationBrowserEntryReadyCheck = oldBrowserEntryReadyCheck
		integrationOpenBrowserFn = oldOpenBrowser
		integrationBrowserEntryWait = oldBrowserEntryWait
		integrationBrowserEntryPollInterval = oldBrowserEntryPollInterval
	}()

	integrationReadyCheck = func(string) bool { return true }
	var entryChecks int
	integrationBrowserEntryReadyCheck = func(int) bool {
		entryChecks++
		return false
	}
	var opened string
	integrationOpenBrowserFn = func(target string) error {
		opened = target
		return nil
	}
	integrationBrowserEntryWait = 5 * time.Millisecond
	integrationBrowserEntryPollInterval = time.Millisecond

	var stderr bytes.Buffer
	if ok := openExistingIntegrationBrowserIfReady(18085, nil, &stderr, ""); !ok {
		t.Fatal("openExistingIntegrationBrowserIfReady() = false, want true")
	}
	if opened != integrationBrowserOpenURL(18085) {
		t.Fatalf("opened = %q, want browser url", opened)
	}
	if entryChecks == 0 {
		t.Fatal("expected browser entry readiness to be checked at least once")
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestStartIntegrationProcessOpensBrowserWhenAlreadyRunningEvenIfBrowserEntryNotReady(t *testing.T) {
	oldTokenFn := integrationSiteVersionTokenFn
	integrationSiteVersionTokenFn = func() string { return "site-version-token" }
	defer func() { integrationSiteVersionTokenFn = oldTokenFn }()

	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "integration.pid")
	logFile := filepath.Join(tempDir, "integration.log")
	flags := map[string]string{
		"pid-file": pidFile,
		"log-file": logFile,
		"port":     strconv.Itoa(integrationServicePort),
	}
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}

	oldReadyCheck := integrationReadyCheck
	oldBrowserEntryReadyCheck := integrationBrowserEntryReadyCheck
	oldBrowserEntryWait := integrationBrowserEntryWait
	oldBrowserEntryPollInterval := integrationBrowserEntryPollInterval
	oldOpenBrowser := integrationOpenBrowserFn
	defer func() {
		integrationReadyCheck = oldReadyCheck
		integrationBrowserEntryReadyCheck = oldBrowserEntryReadyCheck
		integrationBrowserEntryWait = oldBrowserEntryWait
		integrationBrowserEntryPollInterval = oldBrowserEntryPollInterval
		integrationOpenBrowserFn = oldOpenBrowser
	}()

	stubIntegrationReadyCheckForPID(t, pidFile)
	integrationBrowserEntryReadyCheck = func(int) bool { return false }
	integrationBrowserEntryWait = 5 * time.Millisecond
	integrationBrowserEntryPollInterval = time.Millisecond

	var opened string
	integrationOpenBrowserFn = func(target string) error {
		opened = target
		return nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := startIntegrationProcess(flags, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr=%s", code, stderr.String())
	}
	if opened != integrationBrowserOpenURL(8080) {
		t.Fatalf("opened = %q, want browser url", opened)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestStartIntegrationProcessOpensBrowserAfterFreshLaunch(t *testing.T) {
	oldTokenFn := integrationSiteVersionTokenFn
	integrationSiteVersionTokenFn = func() string { return "site-version-token" }
	defer func() { integrationSiteVersionTokenFn = oldTokenFn }()

	tempDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	pidFile := filepath.Join(tempDir, "integration.pid")
	logFile := filepath.Join(tempDir, "integration.log")
	scriptPath := filepath.Join(tempDir, "fake-integration.sh")
	flags := map[string]string{
		"pid-file":  pidFile,
		"log-file":  logFile,
		"port":      "8080",
		"agent-dir": filepath.Join(tempDir, "agent"),
		"site":      filepath.Join(tempDir, "site"),
	}
	script := "#!/bin/sh\n" +
		"pid_file=''\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  if [ \"$1\" = \"--pid-file\" ]; then\n" +
		"    shift\n" +
		"    pid_file=\"$1\"\n" +
		"  fi\n" +
		"  shift\n" +
		"done\n" +
		"if [ -n \"$pid_file\" ]; then\n" +
		"  printf '%s' \"$$\" > \"$pid_file\"\n" +
		"fi\n" +
		"trap 'rm -f \"$pid_file\"; exit 0' TERM INT\n" +
		"while :; do sleep 1; done\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake integration script: %v", err)
	}

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	integrationExecutableFn = func() (string, error) {
		return scriptPath, nil
	}

	oldReadyCheck := integrationReadyCheck
	integrationReadyCheck = func(string) bool {
		_, ok := readRunningPID(pidFile)
		return ok
	}
	defer func() { integrationReadyCheck = oldReadyCheck }()
	oldBrowserEntryReadyCheck := integrationBrowserEntryReadyCheck
	oldBrowserEntryWait := integrationBrowserEntryWait
	oldBrowserEntryPollInterval := integrationBrowserEntryPollInterval
	entryChecks := 0
	integrationBrowserEntryReadyCheck = func(int) bool {
		entryChecks++
		return entryChecks >= 3
	}
	integrationBrowserEntryWait = 100 * time.Millisecond
	integrationBrowserEntryPollInterval = time.Millisecond
	defer func() {
		integrationBrowserEntryReadyCheck = oldBrowserEntryReadyCheck
		integrationBrowserEntryWait = oldBrowserEntryWait
		integrationBrowserEntryPollInterval = oldBrowserEntryPollInterval
	}()

	var opened string
	oldOpenBrowser := integrationOpenBrowserFn
	integrationOpenBrowserFn = func(target string) error {
		opened = target
		return nil
	}
	defer func() { integrationOpenBrowserFn = oldOpenBrowser }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := startIntegrationProcess(flags, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr=%s", code, stderr.String())
	}
	if opened != integrationBrowserOpenURL(8080) {
		t.Fatalf("opened = %q, want browser url", opened)
	}
	if entryChecks < 3 {
		t.Fatalf("entryChecks = %d, want at least 3", entryChecks)
	}
	if _, ok := readRunningPID(pidFile); !ok {
		t.Fatalf("expected running pid in %s", pidFile)
	}
	pid, _ := readRunningPID(pidFile)
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Signal(syscall.SIGTERM)
	}
}

func TestStartIntegrationProcessUsesAbsoluteRuntimePIDFileForBundledApp(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("bundle runtime path behavior is only used on darwin")
	}

	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	appSupportDir := integrationTestMacRuntimeBaseDir(homeDir)
	if err := os.MkdirAll(appSupportDir, 0o755); err != nil {
		t.Fatal(err)
	}

	originalHomeFn := integrationUserHomeFn
	integrationUserHomeFn = func() (string, error) { return homeDir, nil }
	defer func() { integrationUserHomeFn = originalHomeFn }()

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	logFile := filepath.Join(appSupportDir, integrationDefaultLogFile)
	pidFile := filepath.Join(appSupportDir, integrationDefaultPIDFile)
	scriptPath := filepath.Join(tempDir, "fake-integration.sh")
	flags := map[string]string{
		"port":      "8080",
		"agent-dir": filepath.Join(tempDir, "agent"),
		"site":      filepath.Join(tempDir, "site"),
	}
	script := "#!/bin/sh\n" +
		"pid_file=''\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  if [ \"$1\" = \"--pid-file\" ]; then\n" +
		"    shift\n" +
		"    pid_file=\"$1\"\n" +
		"  fi\n" +
		"  shift\n" +
		"done\n" +
		"if [ -n \"$pid_file\" ]; then\n" +
		"  printf '%s' \"$$\" > \"$pid_file\"\n" +
		"fi\n" +
		"trap 'rm -f \"$pid_file\"; exit 0' TERM INT\n" +
		"while :; do sleep 1; done\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake integration script: %v", err)
	}

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	integrationExecutableFn = func() (string, error) {
		return scriptPath, nil
	}

	stubIntegrationReadyCheckForPID(t, pidFile)

	oldOpenBrowser := integrationOpenBrowserFn
	integrationOpenBrowserFn = func(string) error { return nil }
	defer func() { integrationOpenBrowserFn = oldOpenBrowser }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := startIntegrationProcess(flags, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr=%s", code, stderr.String())
	}

	if _, ok := readRunningPID(pidFile); !ok {
		t.Fatalf("expected bundled runtime pid in %s", pidFile)
	}
	if _, err := os.Stat(logFile); err != nil {
		t.Fatalf("expected bundled runtime log in %s: %v", logFile, err)
	}

	pid, _ := readRunningPID(pidFile)
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Signal(syscall.SIGTERM)
	}
}

func TestStopIntegrationProcessCleansStartupPIDFilesAfterShutdown(t *testing.T) {
	tempDir := t.TempDir()
	useIntegrationExecutableDir(t, tempDir)
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	pidFile := filepath.Join(tempDir, "integration.pid")
	logFile := filepath.Join(tempDir, "integration.log")
	scriptPath := filepath.Join(tempDir, "fake-integration.sh")
	flags := map[string]string{
		"pid-file":  pidFile,
		"log-file":  logFile,
		"port":      "8080",
		"agent-dir": filepath.Join(tempDir, "agent"),
		"site":      filepath.Join(tempDir, "site"),
	}
	script := "#!/bin/sh\n" +
		"pid_file=''\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  if [ \"$1\" = \"--pid-file\" ]; then\n" +
		"    shift\n" +
		"    pid_file=\"$1\"\n" +
		"  fi\n" +
		"  shift\n" +
		"done\n" +
		"if [ -n \"$pid_file\" ]; then\n" +
		"  printf '%s' \"$$\" > \"$pid_file\"\n" +
		"fi\n" +
		"trap 'rm -f \"$pid_file\"; exit 0' TERM INT\n" +
		"while :; do sleep 1; done\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake integration script: %v", err)
	}

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	integrationExecutableFn = func() (string, error) {
		return scriptPath, nil
	}
	stubIntegrationReadyCheckForPID(t, pidFile)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := startIntegrationProcess(flags, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("start code = %d, stderr=%s", code, stderr.String())
	}
	stalePluginPID := filepath.Join(integrationStartupDir(pidFile), "browser.pid")
	if err := os.WriteFile(stalePluginPID, []byte("12345"), 0o644); err != nil {
		t.Fatalf("write stale pid: %v", err)
	}
	code = stopIntegrationProcess(flags, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("stop code = %d, stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(stalePluginPID); !os.IsNotExist(err) {
		t.Fatalf("stale plugin pid should be removed after stop, stat err=%v", err)
	}
}

func TestStopIntegrationProcessStopsPluginsBeforeIntegration(t *testing.T) {
	tempDir := t.TempDir()
	pluginsDir := filepath.Join(tempDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pidFile := filepath.Join(tempDir, "integration.pid")
	logFile := filepath.Join(tempDir, "integration.log")
	scriptPath := filepath.Join(tempDir, "fake-integration.sh")
	orderFile := filepath.Join(tempDir, "order.log")
	flags := map[string]string{
		"pid-file": pidFile,
		"log-file": logFile,
	}

	script := "#!/bin/sh\n" +
		"pid_file=''\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  if [ \"$1\" = \"--pid-file\" ]; then\n" +
		"    shift\n" +
		"    pid_file=\"$1\"\n" +
		"  fi\n" +
		"  shift\n" +
		"done\n" +
		"if [ -n \"$pid_file\" ]; then\n" +
		"  printf '%s' \"$$\" > \"$pid_file\"\n" +
		"fi\n" +
		"trap 'printf \"process-stop\\n\" >> " + strconv.Quote(orderFile) + "; rm -f \"$pid_file\"; exit 0' TERM INT\n" +
		"while :; do sleep 1; done\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake integration script: %v", err)
	}

	originalExecutable := integrationExecutableFn
	originalReadyCheck := integrationReadyCheck
	originalRunConnectListMeta := runConnectListMeta
	originalRunConnectPluginStatus := runConnectPluginStatus
	originalRunConnectPluginAction := runConnectPluginAction
	defer func() {
		integrationExecutableFn = originalExecutable
		integrationReadyCheck = originalReadyCheck
		runConnectListMeta = originalRunConnectListMeta
		runConnectPluginStatus = originalRunConnectPluginStatus
		runConnectPluginAction = originalRunConnectPluginAction
	}()

	integrationExecutableFn = func() (string, error) {
		return scriptPath, nil
	}
	stubIntegrationReadyCheckForPID(t, pidFile)
	runConnectListMeta = func() ([]connectMetaConfigListItem, error) {
		return []connectMetaConfigListItem{
			{Key: "feishu", Name: "feishu", Callback: filepath.Join(pluginsDir, "feishu")},
			{Key: "email", Name: "email", Callback: filepath.Join(pluginsDir, "email")},
		}, nil
	}
	runConnectPluginStatus = func(target string, flags map[string]string) (*connectsvc.PluginStatus, error) {
		return &connectsvc.PluginStatus{Key: target, Name: target, Started: true}, nil
	}
	runConnectPluginAction = func(target, action string, flags map[string]string) (*connectsvc.PluginActionResult, error) {
		f, err := os.OpenFile(orderFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		fmt.Fprintf(f, "plugin-stop:%s:%s\n", action, filepath.Base(target))
		return &connectsvc.PluginActionResult{
			Path:    target,
			Command: []string{action},
			Output:  []byte(`{"status":"stopped"}`),
		}, nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := startIntegrationProcess(flags, &stdout, &stderr); code != 0 {
		t.Fatalf("start code = %d, stderr=%s", code, stderr.String())
	}
	writeIntegrationPluginPID(t, filepath.Join(pluginsDir, "feishu.pid"), os.Getpid())
	writeIntegrationPluginPID(t, filepath.Join(pluginsDir, "email.pid"), os.Getpid())

	stdout.Reset()
	stderr.Reset()
	if code := stopIntegrationProcess(flags, &stdout, &stderr); code != 0 {
		t.Fatalf("stop code = %d, stderr=%s", code, stderr.String())
	}

	data, err := os.ReadFile(orderFile)
	if err != nil {
		t.Fatalf("read order file: %v", err)
	}
	lines := strings.Fields(strings.TrimSpace(string(data)))
	if len(lines) != 3 {
		t.Fatalf("order lines = %v, want 3 entries", lines)
	}
	if lines[0] != "plugin-stop:stop:feishu" {
		t.Fatalf("first order = %q, want plugin-stop:stop:feishu", lines[0])
	}
	if lines[1] != "plugin-stop:stop:email" {
		t.Fatalf("second order = %q, want plugin-stop:stop:email", lines[1])
	}
	if lines[2] != "process-stop" {
		t.Fatalf("third order = %q, want process-stop", lines[2])
	}
}

func TestStopIntegrationProcessStillSignalsProcessWhenPluginStopFails(t *testing.T) {
	tempDir := t.TempDir()
	pluginsDir := filepath.Join(tempDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pidFile := filepath.Join(tempDir, "integration.pid")
	logFile := filepath.Join(tempDir, "integration.log")
	scriptPath := filepath.Join(tempDir, "fake-integration.sh")
	orderFile := filepath.Join(tempDir, "order.log")
	flags := map[string]string{
		"pid-file": pidFile,
		"log-file": logFile,
	}

	script := "#!/bin/sh\n" +
		"pid_file=''\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  if [ \"$1\" = \"--pid-file\" ]; then\n" +
		"    shift\n" +
		"    pid_file=\"$1\"\n" +
		"  fi\n" +
		"  shift\n" +
		"done\n" +
		"if [ -n \"$pid_file\" ]; then\n" +
		"  printf '%s' \"$$\" > \"$pid_file\"\n" +
		"fi\n" +
		"trap 'printf \"process-stop\\n\" >> " + strconv.Quote(orderFile) + "; rm -f \"$pid_file\"; exit 0' TERM INT\n" +
		"while :; do sleep 1; done\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake integration script: %v", err)
	}

	originalExecutable := integrationExecutableFn
	originalReadyCheck := integrationReadyCheck
	originalRunConnectListMeta := runConnectListMeta
	originalRunConnectPluginStatus := runConnectPluginStatus
	originalRunConnectPluginAction := runConnectPluginAction
	defer func() {
		integrationExecutableFn = originalExecutable
		integrationReadyCheck = originalReadyCheck
		runConnectListMeta = originalRunConnectListMeta
		runConnectPluginStatus = originalRunConnectPluginStatus
		runConnectPluginAction = originalRunConnectPluginAction
	}()

	integrationExecutableFn = func() (string, error) {
		return scriptPath, nil
	}
	stubIntegrationReadyCheckForPID(t, pidFile)
	runConnectListMeta = func() ([]connectMetaConfigListItem, error) {
		return []connectMetaConfigListItem{
			{Key: "feishu", Name: "feishu", Callback: filepath.Join(pluginsDir, "feishu")},
		}, nil
	}
	runConnectPluginStatus = func(target string, flags map[string]string) (*connectsvc.PluginStatus, error) {
		return &connectsvc.PluginStatus{Key: target, Name: target, Started: true}, nil
	}
	runConnectPluginAction = func(target, action string, flags map[string]string) (*connectsvc.PluginActionResult, error) {
		return nil, fmt.Errorf("plugin stop failed")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := startIntegrationProcess(flags, &stdout, &stderr); code != 0 {
		t.Fatalf("start code = %d, stderr=%s", code, stderr.String())
	}
	writeIntegrationPluginPID(t, filepath.Join(pluginsDir, "feishu.pid"), os.Getpid())

	stdout.Reset()
	stderr.Reset()
	if code := stopIntegrationProcess(flags, &stdout, &stderr); code != 0 {
		t.Fatalf("expected stop to continue on plugin failure, code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "plugin stop failed") {
		t.Fatalf("stderr = %q, want plugin stop failed", stderr.String())
	}
	if _, ok := readRunningPID(pidFile); ok {
		t.Fatal("integration process should be stopped even when plugin stop fails")
	}
	data, err := os.ReadFile(orderFile)
	if err != nil {
		t.Fatalf("read order file: %v", err)
	}
	lines := strings.Fields(strings.TrimSpace(string(data)))
	if len(lines) != 1 || lines[0] != "process-stop" {
		t.Fatalf("order lines = %v, want only process-stop", lines)
	}

	logContent, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	logText := string(logContent)
	if !strings.Contains(logText, "stop plugin feishu failed") {
		t.Fatalf("log file missing plugin failure warning: %s", logText)
	}
	if !strings.Contains(logText, "service stopped") {
		t.Fatalf("log file missing service stopped entry: %s", logText)
	}
}

func TestStopIntegrationProcessPreservesStartupConfigAfterShutdown(t *testing.T) {
	tempDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	startupConfigPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(startupConfigPath, []byte("{\"host\":\"http://127.0.0.1:9998\"}\n"), 0o644); err != nil {
		t.Fatalf("write config/config.json: %v", err)
	}

	pidFile := filepath.Join(tempDir, "integration.pid")
	logFile := filepath.Join(tempDir, "integration.log")
	scriptPath := filepath.Join(tempDir, "fake-integration.sh")
	flags := map[string]string{
		"pid-file":  pidFile,
		"log-file":  logFile,
		"port":      "8080",
		"agent-dir": filepath.Join(tempDir, "agent"),
		"site":      filepath.Join(tempDir, "site"),
	}
	script := "#!/bin/sh\n" +
		"pid_file=''\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  if [ \"$1\" = \"--pid-file\" ]; then\n" +
		"    shift\n" +
		"    pid_file=\"$1\"\n" +
		"  fi\n" +
		"  shift\n" +
		"done\n" +
		"if [ -n \"$pid_file\" ]; then\n" +
		"  printf '%s' \"$$\" > \"$pid_file\"\n" +
		"fi\n" +
		"trap 'rm -f \"$pid_file\"; exit 0' TERM INT\n" +
		"while :; do sleep 1; done\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake integration script: %v", err)
	}

	originalExecutable := integrationExecutableFn
	integrationExecutableFn = func() (string, error) {
		return scriptPath, nil
	}
	defer func() { integrationExecutableFn = originalExecutable }()

	stubIntegrationReadyCheckForPID(t, pidFile)

	oldOpenBrowser := integrationOpenBrowserFn
	integrationOpenBrowserFn = func(string) error { return nil }
	defer func() { integrationOpenBrowserFn = oldOpenBrowser }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := startIntegrationProcess(flags, &stdout, &stderr); code != 0 {
		t.Fatalf("start code = %d, stderr=%s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := stopIntegrationProcess(flags, &stdout, &stderr); code != 0 {
		t.Fatalf("stop code = %d, stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(startupConfigPath); err != nil {
		t.Fatalf("config/config.json should be preserved after stop, stat err=%v", err)
	}
}

func TestStopIntegrationProcessSkipsPluginsThatAreNotStarted(t *testing.T) {
	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "integration.pid")
	logFile := filepath.Join(tempDir, "integration.log")
	scriptPath := filepath.Join(tempDir, "fake-integration.sh")
	orderFile := filepath.Join(tempDir, "order.log")
	flags := map[string]string{
		"pid-file": pidFile,
		"log-file": logFile,
	}

	script := "#!/bin/sh\n" +
		"pid_file=''\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  if [ \"$1\" = \"--pid-file\" ]; then\n" +
		"    shift\n" +
		"    pid_file=\"$1\"\n" +
		"  fi\n" +
		"  shift\n" +
		"done\n" +
		"if [ -n \"$pid_file\" ]; then\n" +
		"  printf '%s' \"$$\" > \"$pid_file\"\n" +
		"fi\n" +
		"trap 'printf \"process-stop\\n\" >> " + strconv.Quote(orderFile) + "; rm -f \"$pid_file\"; exit 0' TERM INT\n" +
		"while :; do sleep 1; done\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake integration script: %v", err)
	}

	originalExecutable := integrationExecutableFn
	originalReadyCheck := integrationReadyCheck
	originalRunConnectListMeta := runConnectListMeta
	originalRunConnectPluginStatus := runConnectPluginStatus
	originalRunConnectPluginAction := runConnectPluginAction
	defer func() {
		integrationExecutableFn = originalExecutable
		integrationReadyCheck = originalReadyCheck
		runConnectListMeta = originalRunConnectListMeta
		runConnectPluginStatus = originalRunConnectPluginStatus
		runConnectPluginAction = originalRunConnectPluginAction
	}()

	integrationExecutableFn = func() (string, error) {
		return scriptPath, nil
	}
	stubIntegrationReadyCheckForPID(t, pidFile)
	runConnectListMeta = func() ([]connectMetaConfigListItem, error) {
		return []connectMetaConfigListItem{
			{Key: "browser", Name: "browser", Callback: filepath.Join(tempDir, "plugins", "browser")},
		}, nil
	}
	runConnectPluginAction = func(target, action string, flags map[string]string) (*connectsvc.PluginActionResult, error) {
		t.Fatalf("plugin stop should be skipped for not-started plugin: target=%s action=%s", target, action)
		return nil, nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := startIntegrationProcess(flags, &stdout, &stderr); code != 0 {
		t.Fatalf("start code = %d, stderr=%s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := stopIntegrationProcess(flags, &stdout, &stderr); code != 0 {
		t.Fatalf("stop code = %d, stderr=%s", code, stderr.String())
	}

	data, err := os.ReadFile(orderFile)
	if err != nil {
		t.Fatalf("read order file: %v", err)
	}
	lines := strings.Fields(strings.TrimSpace(string(data)))
	if len(lines) != 1 || lines[0] != "process-stop" {
		t.Fatalf("order lines = %v, want only process-stop", lines)
	}
}

func TestStopIntegrationProcessPassesConnectBinToPluginStop(t *testing.T) {
	tempDir := t.TempDir()
	pluginsDir := filepath.Join(tempDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pidFile := filepath.Join(tempDir, "integration.pid")
	logFile := filepath.Join(tempDir, "integration.log")
	scriptPath := filepath.Join(tempDir, "fake-integration.sh")
	flags := map[string]string{
		"pid-file": pidFile,
		"log-file": logFile,
	}

	script := "#!/bin/sh\n" +
		"pid_file=''\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  if [ \"$1\" = \"--pid-file\" ]; then\n" +
		"    shift\n" +
		"    pid_file=\"$1\"\n" +
		"  fi\n" +
		"  shift\n" +
		"done\n" +
		"if [ -n \"$pid_file\" ]; then\n" +
		"  printf '%s' \"$$\" > \"$pid_file\"\n" +
		"fi\n" +
		"trap 'rm -f \"$pid_file\"; exit 0' TERM INT\n" +
		"while :; do sleep 1; done\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake integration script: %v", err)
	}

	originalExecutable := integrationExecutableFn
	originalReadyCheck := integrationReadyCheck
	originalRunConnectListMeta := runConnectListMeta
	originalRunConnectPluginAction := runConnectPluginAction
	defer func() {
		integrationExecutableFn = originalExecutable
		integrationReadyCheck = originalReadyCheck
		runConnectListMeta = originalRunConnectListMeta
		runConnectPluginAction = originalRunConnectPluginAction
	}()

	integrationExecutableFn = func() (string, error) {
		return scriptPath, nil
	}
	stubIntegrationReadyCheckForPID(t, pidFile)
	runConnectListMeta = func() ([]connectMetaConfigListItem, error) {
		return []connectMetaConfigListItem{
			{Key: "browser", Name: "browser", Callback: filepath.Join(pluginsDir, "browser")},
		}, nil
	}
	runConnectPluginStatus = func(target string, flags map[string]string) (*connectsvc.PluginStatus, error) {
		return &connectsvc.PluginStatus{Key: target, Name: target, Started: true}, nil
	}

	called := false
	runConnectPluginAction = func(target, action string, flags map[string]string) (*connectsvc.PluginActionResult, error) {
		called = true
		if action != "stop" {
			t.Fatalf("action = %q, want stop", action)
		}
		if got := strings.TrimSpace(flags["connect-bin"]); got != scriptPath {
			t.Fatalf("connect-bin = %q, want %q", got, scriptPath)
		}
		return &connectsvc.PluginActionResult{
			Path:    target,
			Command: []string{action},
			Output:  []byte(`{"status":"stopped"}`),
		}, nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := startIntegrationProcess(flags, &stdout, &stderr); code != 0 {
		t.Fatalf("start code = %d, stderr=%s", code, stderr.String())
	}
	writeIntegrationPluginPID(t, filepath.Join(pluginsDir, "browser.pid"), os.Getpid())

	stdout.Reset()
	stderr.Reset()
	if code := stopIntegrationProcess(flags, &stdout, &stderr); code != 0 {
		t.Fatalf("stop code = %d, stderr=%s", code, stderr.String())
	}
	if !called {
		t.Fatal("expected plugin stop to be called")
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

func TestStartIntegrationProcessWaitsForHTTPReadiness(t *testing.T) {
	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "integration.pid")
	logFile := filepath.Join(tempDir, "integration.log")
	startupStatusFile := integrationStartupStatusFilePath(pidFile)
	flags := map[string]string{
		"pid-file": pidFile,
		"log-file": logFile,
		"port":     strconv.Itoa(integrationServicePort),
	}

	oldWait := integrationStartWait
	integrationStartWait = 1500 * time.Millisecond
	defer func() { integrationStartWait = oldWait }()

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	oldReadyCheck := integrationReadyCheck
	integrationReadyCheck = func(string) bool { return false }
	defer func() { integrationReadyCheck = oldReadyCheck }()

	scriptPath := filepath.Join(tempDir, "fake-integration.sh")
	script := "#!/bin/sh\n" +
		"pid_file=''\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  if [ \"$1\" = \"--pid-file\" ]; then\n" +
		"    shift\n" +
		"    pid_file=\"$1\"\n" +
		"  fi\n" +
		"  shift\n" +
		"done\n" +
		"if [ -n \"$pid_file\" ]; then\n" +
		"  printf '%s' \"$$\" > \"$pid_file\"\n" +
		"fi\n" +
		"trap 'rm -f \"$pid_file\"; exit 0' TERM INT\n" +
		"while :; do sleep 1; done\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake integration script: %v", err)
	}
	integrationExecutableFn = func() (string, error) {
		return scriptPath, nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := startIntegrationProcess(flags, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected start failure when HTTP is not ready")
	}
	if _, ok := readRunningPID(pidFile); ok {
		t.Fatalf("expected timed out child process to be cleaned up")
	}
	if _, err := os.Stat(startupStatusFile); !os.IsNotExist(err) {
		t.Fatalf("startup status file should be cleaned up on timeout, stat err=%v", err)
	}
}

func TestStartIntegrationProcessReportsStartupFailureMessage(t *testing.T) {
	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "integration.pid")
	logFile := filepath.Join(tempDir, "integration.log")
	flags := map[string]string{
		"pid-file": pidFile,
		"log-file": logFile,
		"port":     "8080",
	}

	oldWait := integrationStartWait
	integrationStartWait = 2 * time.Second
	defer func() { integrationStartWait = oldWait }()

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()

	scriptPath := filepath.Join(tempDir, "fake-integration.sh")
	failureMessage := "server listen failed: listen tcp :8080: bind: address already in use"
	script := "#!/bin/sh\n" +
		"status_file=''\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  if [ \"$1\" = \"--startup-status-file\" ]; then\n" +
		"    shift\n" +
		"    status_file=\"$1\"\n" +
		"  fi\n" +
		"  shift\n" +
		"done\n" +
		"printf '{\"status\":\"failed\",\"message\":%s}\\n' " + strconv.Quote(strconv.Quote(failureMessage)) + " > \"$status_file\"\n" +
		"exit 1\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake integration script: %v", err)
	}
	integrationExecutableFn = func() (string, error) {
		return scriptPath, nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := startIntegrationProcess(flags, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected start failure")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if got := strings.TrimSpace(stderr.String()); got != failureMessage {
		t.Fatalf("stderr = %q, want %q", got, failureMessage)
	}
	logContent, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	logText := string(logContent)
	if !strings.Contains(logText, "start failed: "+failureMessage) {
		t.Fatalf("log file missing startup failure message: %s", logText)
	}
}

func TestStartIntegrationProcessMergesRuntimeFlags(t *testing.T) {
	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "integration.pid")
	logFile := filepath.Join(tempDir, "integration.log")
	argsFile := filepath.Join(tempDir, "args.txt")
	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	startupConfigPath := filepath.Join(configDir, "config.json")
	runtimeJSON := fmt.Sprintf("{\"agent-dir\":%q,\"default-dir\":%q,\"site\":%q,\"host\":\"http://127.0.0.1:9998\",\"pid-file\":%q,\"log-file\":%q}\n",
		filepath.Join(tempDir, "agent-from-runtime"),
		filepath.Join(tempDir, "config-from-runtime"),
		filepath.Join(tempDir, "site-from-runtime"),
		pidFile,
		logFile,
	)
	if err := os.WriteFile(startupConfigPath, []byte(runtimeJSON), 0o644); err != nil {
		t.Fatalf("write config/config.json: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	scriptPath := filepath.Join(tempDir, "fake-integration.sh")
	script := "#!/bin/sh\n" +
		"args_file=''\n" +
		"pid_file=''\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  if [ \"$1\" = \"--pid-file\" ]; then\n" +
		"    shift\n" +
		"    pid_file=\"$1\"\n" +
		"    args_file=\"$args_file pid-file=$1\"\n" +
		"  elif [ \"$1\" = \"--log-file\" ]; then\n" +
		"    shift\n" +
		"    args_file=\"$args_file log-file=$1\"\n" +
		"  elif [ \"$1\" = \"--startup-status-file\" ]; then\n" +
		"    shift\n" +
		"    args_file=\"$args_file startup-status-file=$1\"\n" +
		"  elif [ \"$1\" = \"--agent-dir\" ]; then\n" +
		"    shift\n" +
		"    args_file=\"$args_file agent-dir=$1\"\n" +
		"  elif [ \"$1\" = \"--default-dir\" ]; then\n" +
		"    shift\n" +
		"    args_file=\"$args_file default-dir=$1\"\n" +
		"  elif [ \"$1\" = \"--site\" ]; then\n" +
		"    shift\n" +
		"    args_file=\"$args_file site=$1\"\n" +
		"  fi\n" +
		"  shift\n" +
		"done\n" +
		"printf '%s\\n' \"$args_file\" > " + strconv.Quote(argsFile) + "\n" +
		"if [ -n \"$pid_file\" ]; then\n" +
		"  printf '%s' \"$$\" > \"$pid_file\"\n" +
		"fi\n" +
		"trap 'rm -f \"$pid_file\"; exit 0' TERM INT\n" +
		"while :; do sleep 1; done\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake integration script: %v", err)
	}

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(tempDir, "integration"), nil
	}
	oldReadyCheck := integrationReadyCheck
	integrationReadyCheck = func(string) bool {
		_, ok := readRunningPID(pidFile)
		return ok
	}
	defer func() { integrationReadyCheck = oldReadyCheck }()

	originalExecPath := integrationExecutableFn
	integrationExecutableFn = func() (string, error) {
		return scriptPath, nil
	}
	defer func() { integrationExecutableFn = originalExecPath }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := startIntegrationProcess(map[string]string{}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr=%s", code, stderr.String())
	}

	argsText, err := os.ReadFile(argsFile)
	if err != nil {
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			time.Sleep(50 * time.Millisecond)
			argsText, err = os.ReadFile(argsFile)
			if err == nil {
				break
			}
		}
		if err != nil {
			t.Fatalf("read args file: %v", err)
		}
	}
	got := string(argsText)
	if !strings.Contains(got, "agent-dir="+filepath.Join(tempDir, "agent-from-runtime")) {
		t.Fatalf("agent-dir not inherited from runtime: %q", got)
	}
	if !strings.Contains(got, "default-dir="+filepath.Join(tempDir, "config-from-runtime")) {
		t.Fatalf("default-dir not inherited from config/config.json: %q", got)
	}
	if !strings.Contains(got, "site="+filepath.Join(tempDir, "site-from-runtime")) {
		t.Fatalf("site not inherited from config/config.json: %q", got)
	}
	if !strings.Contains(got, "pid-file="+pidFile) {
		t.Fatalf("pid-file not inherited from config/config.json: %q", got)
	}
	if !strings.Contains(got, "log-file="+logFile) {
		t.Fatalf("log-file not inherited from config/config.json: %q", got)
	}
	if !strings.Contains(got, "startup-status-file="+pidFile+".startup.json") {
		t.Fatalf("startup-status-file not derived from pid-file: %q", got)
	}

	if pid, ok := readRunningPID(pidFile); ok {
		if proc, err := os.FindProcess(pid); err == nil {
			_ = proc.Signal(syscall.SIGTERM)
		}
	}
}

func TestMergeIntegrationRuntimeFlagsRebasesMovedReleasePaths(t *testing.T) {
	oldReleaseDir := filepath.Join(t.TempDir(), "old-release")
	newReleaseDir := filepath.Join(t.TempDir(), "new-release")
	if err := os.MkdirAll(oldReleaseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(newReleaseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(newReleaseDir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	runtimeJSON := fmt.Sprintf("{\"app-dir\":%q,\"default-dir\":%q,\"site\":%q,\"pid-file\":%q,\"log-file\":%q}\n",
		oldReleaseDir,
		filepath.Join(oldReleaseDir, "config"),
		filepath.Join(oldReleaseDir, "site"),
		filepath.Join(oldReleaseDir, "integration.pid"),
		filepath.Join(oldReleaseDir, "integration.log"),
	)
	if err := os.WriteFile(filepath.Join(newReleaseDir, "config", "config.json"), []byte(runtimeJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(newReleaseDir, "integration"), nil
	}

	got := mergeIntegrationRuntimeFlags(map[string]string{})
	if got["default-dir"] != filepath.Join(newReleaseDir, "config") {
		t.Fatalf("default-dir = %q, want %q", got["default-dir"], filepath.Join(newReleaseDir, "config"))
	}
	if got["site"] != filepath.Join(newReleaseDir, "site") {
		t.Fatalf("site = %q, want %q", got["site"], filepath.Join(newReleaseDir, "site"))
	}
}

func TestMergeIntegrationRuntimeFlagsRebasesMovedBundleResourcePaths(t *testing.T) {
	bundleRoot := filepath.Join(t.TempDir(), "DeepRight.app")
	oldBundleRoot := filepath.Join(t.TempDir(), "OldDeepRight.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	configDir := filepath.Join(bundleRoot, "Contents", "Resources", "config")
	oldResourcesDir := filepath.Join(oldBundleRoot, "Contents", "Resources")
	newResourcesDir := filepath.Join(bundleRoot, "Contents", "Resources")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	configJSON := fmt.Sprintf("{\"resources-dir\":%q,\"default-dir\":%q,\"site\":%q}\n",
		oldResourcesDir,
		filepath.Join(oldResourcesDir, "config"),
		filepath.Join(oldResourcesDir, "site"),
	)
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(configJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(macOSDir, "integration"), nil
	}

	got := mergeIntegrationRuntimeFlags(map[string]string{})
	if got["default-dir"] != filepath.Join(newResourcesDir, "config") {
		t.Fatalf("default-dir = %q, want %q", got["default-dir"], filepath.Join(newResourcesDir, "config"))
	}
	if got["site"] != filepath.Join(newResourcesDir, "site") {
		t.Fatalf("site = %q, want %q", got["site"], filepath.Join(newResourcesDir, "site"))
	}
}

func TestMergeIntegrationRuntimeFlagsDoesNotFallbackHTTPSettingsFromRuntime(t *testing.T) {
	releaseDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(releaseDir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	startupConfigJSON := fmt.Sprintf("{\"app-dir\":%q,\"http_timeout\":\"61000\",\"http_connect_timeout\":\"62000\",\"http_socket_timeout\":\"63000\"}\n", releaseDir)
	if err := os.WriteFile(filepath.Join(releaseDir, "config", "config.json"), []byte(startupConfigJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(releaseDir, "integration"), nil
	}

	got := mergeIntegrationRuntimeFlags(map[string]string{})
	if got["http_timeout"] != "" {
		t.Fatalf("http_timeout = %q, want empty", got["http_timeout"])
	}
	if got["http_connect_timeout"] != "" {
		t.Fatalf("http_connect_timeout = %q, want empty", got["http_connect_timeout"])
	}
	if got["http_socket_timeout"] != "" {
		t.Fatalf("http_socket_timeout = %q, want empty", got["http_socket_timeout"])
	}
}

func TestWriteRuntimeConfigUsesStartupDirectorySemantics(t *testing.T) {
	tempDir := t.TempDir()
	useIntegrationExecutableDir(t, tempDir)

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	startupDir, err := filepath.Abs(tempDir)
	if err != nil {
		t.Fatalf("abs startup dir: %v", err)
	}
	writeRuntimeConfig(map[string]interface{}{
		"app":     "/tmp/fake/integration",
		"app-dir": startupDir,
		"db":      filepath.Join(startupDir, "data"),
	})

	data, err := os.ReadFile(filepath.Join(tempDir, "config", "config.json"))
	if err != nil {
		t.Fatalf("read config/config.json: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config/config.json: %v", err)
	}
	wantAppDir := startupDir
	if runtimeDir := strings.TrimSpace(integrationBundleRuntimeBaseDir()); runtimeDir != "" {
		wantAppDir = runtimeDir
	}
	if cfg["app-dir"] != wantAppDir {
		t.Fatalf("app-dir = %v, want %q", cfg["app-dir"], wantAppDir)
	}
	if cfg["db"] != filepath.Join(startupDir, "data") {
		t.Fatalf("db = %v, want %q", cfg["db"], filepath.Join(startupDir, "data"))
	}
}

func TestWriteRuntimeConfigStoresHTTPFieldsUnderConfigHTTP(t *testing.T) {
	tempDir := t.TempDir()
	useIntegrationExecutableDir(t, tempDir)

	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"host":"http://example.com","http":{"debug":false}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	writeRuntimeConfig(map[string]interface{}{
		"http_timeout":         45000,
		"http_connect_timeout": 15000,
		"http_socket_timeout":  47000,
		"http_debug":           true,
	})

	data, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if err != nil {
		t.Fatalf("read config/config.json: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config/config.json: %v", err)
	}
	if _, ok := cfg["http_timeout"]; ok {
		t.Fatalf("legacy top-level http_timeout should be absent: %#v", cfg)
	}
	httpCfg, ok := cfg["http"].(map[string]any)
	if !ok {
		t.Fatalf("http = %#v, want object", cfg["http"])
	}
	if httpCfg["http_timeout"] != float64(45000) {
		t.Fatalf("http.http_timeout = %v", httpCfg["http_timeout"])
	}
	if httpCfg["http_connect_timeout"] != float64(15000) {
		t.Fatalf("http.http_connect_timeout = %v", httpCfg["http_connect_timeout"])
	}
	if httpCfg["http_socket_timeout"] != float64(47000) {
		t.Fatalf("http.http_socket_timeout = %v", httpCfg["http_socket_timeout"])
	}
	if httpCfg["debug"] != true {
		t.Fatalf("http.debug = %v", httpCfg["debug"])
	}
	if cfg["host"] != "http://example.com" {
		t.Fatalf("host = %v, want preserved host", cfg["host"])
	}
}

func TestWriteRuntimeConfigStoresBundledResourcePathsAsRelative(t *testing.T) {
	bundleRoot := filepath.Join(t.TempDir(), "DeepRight.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	configDir := filepath.Join(bundleRoot, "Contents", "Resources", "config")
	siteDir := filepath.Join(bundleRoot, "Contents", "Resources", "site")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(siteDir, 0o755); err != nil {
		t.Fatal(err)
	}

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(macOSDir, "integration"), nil
	}

	writeRuntimeConfig(map[string]interface{}{
		"default-dir": configDir,
		"site":        siteDir,
	})

	data, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if err != nil {
		t.Fatalf("read config/config.json: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config/config.json: %v", err)
	}
	if cfg["default-dir"] != "config" {
		t.Fatalf("default-dir = %v, want %q", cfg["default-dir"], "config")
	}
	if cfg["site"] != "site" {
		t.Fatalf("site = %v, want %q", cfg["site"], "site")
	}
}

func TestIntegrationStartupCreatesKnowledgeDirectory(t *testing.T) {
	tempDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	_, err = knowledgecore.EnsureRuntime(tempDir)
	if err != nil {
		t.Fatalf("EnsureRuntime: %v", err)
	}
	if info, err := os.Stat(filepath.Join(tempDir, "knowledge")); err != nil || !info.IsDir() {
		t.Fatalf("knowledge dir not created: err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(tempDir, "data")); err != nil {
		t.Fatalf("data sqlite not created: %v", err)
	}
}

// ── Static test ──

func TestStaticFiles(t *testing.T) {
	mux := http.NewServeMux()
	registerStatic(mux, "../static/test-case")

	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		path, want string
	}{
		{"/site/hello.html", "HELLO WORLD"},
		{"/site/js/hello.js", "1+1"},
	}
	for _, tt := range tests {
		resp, err := http.Get(server.URL + tt.path)
		if err != nil {
			t.Fatalf("GET %s: %v", tt.path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("GET %s: %d", tt.path, resp.StatusCode)
		}
		if strings.TrimSpace(string(body)) != tt.want {
			t.Errorf("GET %s: %q, want %q", tt.path, strings.TrimSpace(string(body)), tt.want)
		}
	}
}

func TestAppStaticFiles(t *testing.T) {
	tmp := t.TempDir()
	agentRoot := filepath.Join(tmp, "agent")
	appRoot := filepath.Join(agentRoot, "demo-agent", "app")
	if err := os.MkdirAll(agentRoot, 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appRoot, "assets"), 0o755); err != nil {
		t.Fatalf("mkdir app assets: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appRoot, "demos", "nested"), 0o755); err != nil {
		t.Fatalf("mkdir nested demo dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appRoot, "index.html"), []byte("APP HOME\n"), 0o644); err != nil {
		t.Fatalf("write app index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appRoot, "assets", "hello.js"), []byte("console.log('hello app')\n"), 0o644); err != nil {
		t.Fatalf("write app asset: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appRoot, "demos", "nested", "index.html"), []byte("NESTED HOME\n"), 0o644); err != nil {
		t.Fatalf("write nested app index: %v", err)
	}

	mux := http.NewServeMux()
	registerAppStatic(mux, &Config{AgentDir: agentRoot})

	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		path string
		want string
	}{
		{"/mapping/demo-agent/", "APP HOME"},
		{"/mapping/demo-agent/index.html", "APP HOME"},
		{"/mapping/demo-agent/demos/nested/", "NESTED HOME"},
		{"/mapping/demo-agent/demos/nested/index.html", "NESTED HOME"},
		{"/mapping/demo-agent/assets/hello.js", "console.log('hello app')"},
	}
	for _, tt := range tests {
		resp, err := http.Get(server.URL + tt.path)
		if err != nil {
			t.Fatalf("GET %s: %v", tt.path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s: %d", tt.path, resp.StatusCode)
		}
		if strings.TrimSpace(string(body)) != tt.want {
			t.Errorf("GET %s: %q, want %q", tt.path, strings.TrimSpace(string(body)), tt.want)
		}
	}
}

func TestAppStaticExplicitIndexDoesNotRedirect(t *testing.T) {
	tmp := t.TempDir()
	agentRoot := filepath.Join(tmp, "agent")
	appRoot := filepath.Join(agentRoot, "demo-agent", "app")
	if err := os.MkdirAll(appRoot, 0o755); err != nil {
		t.Fatalf("mkdir app root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appRoot, "index.html"), []byte("APP HOME\n"), 0o644); err != nil {
		t.Fatalf("write app index: %v", err)
	}

	mux := http.NewServeMux()
	registerAppStatic(mux, &Config{AgentDir: agentRoot})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(server.URL + "/mapping/demo-agent/index.html")
	if err != nil {
		t.Fatalf("GET explicit index: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET explicit index status = %d, body = %s", resp.StatusCode, string(body))
	}
	if location := resp.Header.Get("Location"); strings.TrimSpace(location) != "" {
		t.Fatalf("GET explicit index should not redirect, Location = %q", location)
	}
	if strings.TrimSpace(string(body)) != "APP HOME" {
		t.Fatalf("GET explicit index body = %q", strings.TrimSpace(string(body)))
	}
}

func TestHandleAppStaticRejectsPathEscape(t *testing.T) {
	tmp := t.TempDir()
	agentRoot := filepath.Join(tmp, "agent")
	if err := os.MkdirAll(agentRoot, 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/mapping/demo-agent/%2e%2e/secret.txt", nil)
	rec := httptest.NewRecorder()

	handleAppStatic(&Config{AgentDir: agentRoot}).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("GET escape path status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "path escapes app root") {
		t.Fatalf("GET escape path body = %q", rec.Body.String())
	}
}

func TestHandleAppStaticRequiresAgentID(t *testing.T) {
	tmp := t.TempDir()
	agentRoot := filepath.Join(tmp, "agent")
	if err := os.MkdirAll(agentRoot, 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/mapping/", nil)
	rec := httptest.NewRecorder()

	handleAppStatic(&Config{AgentDir: agentRoot}).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("GET /mapping/ status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "agentId is required") {
		t.Fatalf("GET /mapping/ body = %q", rec.Body.String())
	}
}

func TestResolveServeAgentDirDefaultsAndCreatesDirectory(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if runtime.GOOS == "darwin" {
		t.Setenv("HOME", tmp)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)
	defaultDir := filepath.Join(tmp, "config")
	writeDefaultAgentTemplate(t, defaultDir)

	got, err := resolveServeAgentDir("", "")
	if err != nil {
		t.Fatalf("resolveServeAgentDir: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd after chdir: %v", err)
	}
	want := filepath.Join(cwd, "agent")
	if runtime.GOOS == "darwin" {
		want = filepath.Join(integrationTestMacRuntimeBaseDir(tmp), "agent")
	}
	if want, err = filepath.Abs(want); err != nil {
		t.Fatalf("abs agent dir: %v", err)
	}
	if got != want {
		t.Fatalf("agent dir = %q, want %q", got, want)
	}
	info, err := os.Stat(want)
	if err != nil {
		t.Fatalf("stat default agent dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("default agent dir should be a directory")
	}
	defAgentDir := filepath.Join(want, "DEF_AGENT")
	defSkillsDir := filepath.Join(defAgentDir, "skills")
	if info, err := os.Stat(defAgentDir); err != nil || !info.IsDir() {
		t.Fatalf("default DEF_AGENT dir missing: %v", err)
	}
	if info, err := os.Stat(defSkillsDir); err != nil || !info.IsDir() {
		t.Fatalf("default DEF_AGENT/skills dir missing: %v", err)
	}
	assertDefaultAgentTemplateCopied(t, defAgentDir)
}

func TestDefaultIntegrationStartupOptionsUsesMacAgentDir(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("mac-specific default agent dir")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AGENT_DIR", "")

	defaults := defaultIntegrationStartupOptions()
	want := filepath.Join(integrationTestMacRuntimeBaseDir(home), "agent")
	if defaults.Config.AgentDir != want {
		t.Fatalf("defaults.Config.AgentDir = %q, want %q", defaults.Config.AgentDir, want)
	}
}

func TestApplyIntegrationStartupConfigSupportsSkillExtractRound(t *testing.T) {
	opts := defaultIntegrationStartupOptions()
	if err := applyIntegrationStartupConfig(&opts, map[string]interface{}{
		"skill_extract": json.Number("12"),
	}); err != nil {
		t.Fatalf("applyIntegrationStartupConfig: %v", err)
	}
	if opts.Config.SkillExtractRound != 12 {
		t.Fatalf("SkillExtractRound = %d, want 12", opts.Config.SkillExtractRound)
	}
}

func TestIntegrationStartupConfigKeyAliasesSkillCreateToSkillExtract(t *testing.T) {
	got := integrationStartupConfigKeys[normalizeIntegrationStartupConfigKey("skill_create")]
	if got != "skill_extract" {
		t.Fatalf("alias = %q, want skill_extract", got)
	}
}

func TestResolveServeAgentDirScaffoldsExplicitEmptyDirectory(t *testing.T) {
	tmp := t.TempDir()
	agentDir := filepath.Join(tmp, "custom-agent")
	defaultDir := filepath.Join(tmp, "config")
	writeDefaultAgentTemplate(t, defaultDir)
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}

	got, err := resolveServeAgentDir(agentDir, defaultDir)
	if err != nil {
		t.Fatalf("resolveServeAgentDir: %v", err)
	}
	if got != agentDir {
		t.Fatalf("agent dir = %q, want %q", got, agentDir)
	}
	if info, err := os.Stat(filepath.Join(agentDir, "DEF_AGENT")); err != nil || !info.IsDir() {
		t.Fatalf("DEF_AGENT missing in explicit empty dir: %v", err)
	}
	if info, err := os.Stat(filepath.Join(agentDir, "DEF_AGENT", "skills")); err != nil || !info.IsDir() {
		t.Fatalf("DEF_AGENT/skills missing in explicit empty dir: %v", err)
	}
	assertDefaultAgentTemplateCopied(t, filepath.Join(agentDir, "DEF_AGENT"))
}

func TestResolveServeAgentDirFailsWhenDefaultDirUnavailableForEmptyDirectory(t *testing.T) {
	tmp := t.TempDir()
	agentDir := filepath.Join(tmp, "agent")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}

	if _, err := resolveServeAgentDir(agentDir, filepath.Join(tmp, "missing-config")); err == nil || !strings.Contains(err.Error(), "default-dir unavailable") {
		t.Fatalf("resolveServeAgentDir error = %v, want default-dir unavailable", err)
	}
	if _, err := os.Stat(filepath.Join(agentDir, "DEF_AGENT")); !os.IsNotExist(err) {
		t.Fatalf("DEF_AGENT should not remain after failed scaffold, err=%v", err)
	}
}

func TestResolveServeAgentDirKeepsExistingContentUntouched(t *testing.T) {
	tmp := t.TempDir()
	agentDir := filepath.Join(tmp, "agent")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}
	existing := filepath.Join(agentDir, "existing.txt")
	if err := os.WriteFile(existing, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	if _, err := resolveServeAgentDir(agentDir, filepath.Join(tmp, "missing-config")); err != nil {
		t.Fatalf("resolveServeAgentDir: %v", err)
	}
	if _, err := os.Stat(existing); err != nil {
		t.Fatalf("existing file should remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(agentDir, "DEF_AGENT")); !os.IsNotExist(err) {
		t.Fatalf("DEF_AGENT should not be created for non-empty dir, err=%v", err)
	}
}

func TestCopyDefaultAgentTemplateCreatesEmptyConfigJSON(t *testing.T) {
	tmp := t.TempDir()
	defaultDir := filepath.Join(tmp, "config")
	dst := filepath.Join(tmp, "agent")
	writeDefaultAgentTemplate(t, defaultDir)
	if err := os.WriteFile(filepath.Join(defaultDir, "config.json"), []byte("{\"host\":\"http://127.0.0.1:9998\",\"version\":\"1.0\"}\n"), 0o644); err != nil {
		t.Fatalf("write bundled config: %v", err)
	}

	if err := copyDefaultAgentTemplate(defaultDir, dst); err != nil {
		t.Fatalf("copyDefaultAgentTemplate: %v", err)
	}

	assertDefaultAgentTemplateCopied(t, dst)
	assertAgentConfigIsEmpty(t, dst)
}

func TestResolveServeAgentDirScaffoldCreatesEmptyConfigJSON(t *testing.T) {
	tmp := t.TempDir()
	agentDir := filepath.Join(tmp, "agent")
	defaultDir := filepath.Join(tmp, "config")
	writeDefaultAgentTemplate(t, defaultDir)
	if err := os.WriteFile(filepath.Join(defaultDir, "config.json"), []byte("{\"host\":\"http://127.0.0.1:9998\",\"version\":\"1.0\"}\n"), 0o644); err != nil {
		t.Fatalf("write bundled config: %v", err)
	}
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}

	got, err := resolveServeAgentDir(agentDir, defaultDir)
	if err != nil {
		t.Fatalf("resolveServeAgentDir: %v", err)
	}
	if got != agentDir {
		t.Fatalf("agent dir = %q, want %q", got, agentDir)
	}

	defAgentDir := filepath.Join(agentDir, "DEF_AGENT")
	assertDefaultAgentTemplateCopied(t, defAgentDir)
	assertAgentConfigIsEmpty(t, defAgentDir)
}

func TestHandleAgentInitCreatesEmptyConfigJSON(t *testing.T) {
	tmp := t.TempDir()
	agentDir := filepath.Join(tmp, "agent")
	defaultDir := filepath.Join(tmp, "config")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}
	writeDefaultAgentTemplate(t, defaultDir)
	if err := os.WriteFile(filepath.Join(defaultDir, "config.json"), []byte("{\"host\":\"http://127.0.0.1:9998\",\"version\":\"1.0\"}\n"), 0o644); err != nil {
		t.Fatalf("write bundled config: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agent/init?name=A", nil)
	rec := httptest.NewRecorder()
	handleAgentInit(&Config{AgentDir: agentDir, DefaultDir: defaultDir}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["status"] != float64(0) {
		t.Fatalf("response status = %#v, want 0", resp["status"])
	}

	newAgentDir := filepath.Join(agentDir, "A")
	assertDefaultAgentTemplateCopied(t, newAgentDir)
	assertAgentConfigIsEmpty(t, newAgentDir)
}

func writeDefaultAgentTemplate(t *testing.T, dir string) {
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

func assertDefaultAgentTemplateCopied(t *testing.T, dir string) {
	t.Helper()
	if data, err := os.ReadFile(filepath.Join(dir, "SOUL.md")); err != nil || string(data) != "soul" {
		t.Fatalf("SOUL.md copy mismatch, err=%v data=%q", err, string(data))
	}
	if data, err := os.ReadFile(filepath.Join(dir, "nested", "USER.md")); err != nil || string(data) != "user" {
		t.Fatalf("nested USER.md copy mismatch, err=%v data=%q", err, string(data))
	}
}

func assertAgentConfigIsEmpty(t *testing.T, dir string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("read config.json: %v", err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("decode config.json: %v", err)
	}
	if len(cfg) != 0 {
		t.Fatalf("config.json = %#v, want empty object", cfg)
	}
}

func TestResolveServeSiteDirDefaultsToExecutableSite(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	exeDir := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(exeDir, "integration"), nil
	}

	got, err := resolveServeSiteDir("")
	if err != nil {
		t.Fatalf("resolveServeSiteDir: %v", err)
	}

	want, err := filepath.Abs(filepath.Join(exeDir, "site"))
	if err != nil {
		t.Fatalf("abs site dir: %v", err)
	}
	if got != want {
		t.Fatalf("site dir = %q, want %q", got, want)
	}
}

func TestResolveServeSiteDirUsesBundleResources(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("bundle resources layout is only used on darwin")
	}

	bundleRoot := filepath.Join(t.TempDir(), "integration.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatalf("mkdir macOS dir: %v", err)
	}

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(macOSDir, "integration"), nil
	}

	got, err := resolveServeSiteDir("")
	if err != nil {
		t.Fatalf("resolveServeSiteDir: %v", err)
	}

	want := filepath.Join(bundleRoot, "Contents", "Resources", "site")
	if got != want {
		t.Fatalf("site dir = %q, want %q", got, want)
	}
}

func TestRunIntegrationForegroundDefaultsHostToWWWDeepright(t *testing.T) {
	fs := flag.NewFlagSet("integration-serve", flag.ContinueOnError)
	var cfg Config
	fs.StringVar(&cfg.Host, "host", defaultUpstreamHost, "upstream server URL")
	if err := fs.Parse(nil); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	if got := cfg.Host; got != defaultUpstreamHost {
		t.Fatalf("default host = %q, want %s", got, defaultUpstreamHost)
	}
}

func useIntegrationExecutableDir(t *testing.T, dir string) {
	t.Helper()
	originalExecutable := integrationExecutableFn
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(dir, "integration"), nil
	}
	t.Cleanup(func() {
		integrationExecutableFn = originalExecutable
	})
	t.Setenv(integrationRuntimeDirEnv, "")
}

func useBundledIntegrationExecutable(t *testing.T, bundleRoot string) {
	t.Helper()
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatalf("mkdir macOS dir: %v", err)
	}
	originalExecutable := integrationExecutableFn
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(macOSDir, "integration"), nil
	}
	t.Cleanup(func() {
		integrationExecutableFn = originalExecutable
	})
	t.Setenv(integrationRuntimeDirEnv, "")
}

func TestLoadIntegrationStartupOptionsReadsConfigDirectoryConfig(t *testing.T) {
	tempDir := t.TempDir()
	useIntegrationExecutableDir(t, tempDir)

	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{
  "host": "http://www.dr.cn",
  "agentDir": "agents",
  "default_dir": "templates",
  "site": "web",
  "connectTimeout": 2500,
  "pluginExecTimeout": 7000,
  "pid_file": "custom.pid",
  "logFile": "custom.log"
}
`), 0o644); err != nil {
		t.Fatalf("write config.json: %v", err)
	}

	opts, path, err := loadIntegrationStartupOptions()
	if err != nil {
		t.Fatalf("loadIntegrationStartupOptions: %v", err)
	}
	if path != configPath {
		t.Fatalf("path = %q, want %q", path, configPath)
	}
	if opts.Config.Host != "http://www.dr.cn" {
		t.Fatalf("host = %q, want http://www.dr.cn", opts.Config.Host)
	}
	if opts.Config.AgentDir != "agents" {
		t.Fatalf("agent-dir = %q, want agents", opts.Config.AgentDir)
	}
	if opts.Config.DefaultDir != "templates" {
		t.Fatalf("default-dir = %q, want templates", opts.Config.DefaultDir)
	}
	if opts.Config.Site != "web" {
		t.Fatalf("site = %q, want web", opts.Config.Site)
	}
	if opts.Config.ConnectTimeoutMs != 2500 {
		t.Fatalf("connect_timeout = %d, want 2500", opts.Config.ConnectTimeoutMs)
	}
	if opts.Config.PluginExecTimeout != 7000 {
		t.Fatalf("plugin_exec_timeout = %d, want 7000", opts.Config.PluginExecTimeout)
	}
	if opts.PIDFile != "custom.pid" {
		t.Fatalf("pid-file = %q, want custom.pid", opts.PIDFile)
	}
	if opts.LogFile != "custom.log" {
		t.Fatalf("log-file = %q, want custom.log", opts.LogFile)
	}
}

func TestLoadIntegrationStartupOptionsReadsNestedHTTPConfig(t *testing.T) {
	tempDir := t.TempDir()
	useIntegrationExecutableDir(t, tempDir)

	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{
  "http": {
    "http_timeout": 47000,
    "http_connect_timeout": 16000,
    "http_socket_timeout": 46000,
    "debug": true
  }
}
`), 0o644); err != nil {
		t.Fatalf("write config.json: %v", err)
	}

	opts, path, err := loadIntegrationStartupOptions()
	if err != nil {
		t.Fatalf("loadIntegrationStartupOptions: %v", err)
	}
	if path != configPath {
		t.Fatalf("path = %q, want %q", path, configPath)
	}
	if opts.Config.HTTPTimeout != 47000 {
		t.Fatalf("http_timeout = %d, want 47000", opts.Config.HTTPTimeout)
	}
	if opts.Config.HTTPConnTimeout != 16000 {
		t.Fatalf("http_connect_timeout = %d, want 16000", opts.Config.HTTPConnTimeout)
	}
	if opts.Config.HTTPSocketTimeout != 46000 {
		t.Fatalf("http_socket_timeout = %d, want 46000", opts.Config.HTTPSocketTimeout)
	}
	if !opts.Config.HTTPDebug {
		t.Fatal("http_debug = false, want true")
	}
}

func TestLoadIntegrationStartupOptionsIgnoresFlatHTTPConfigKeys(t *testing.T) {
	tempDir := t.TempDir()
	useIntegrationExecutableDir(t, tempDir)

	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{
  "http_timeout": 99999,
  "http_connect_timeout": 99998,
  "http_socket_timeout": 99997,
  "http_debug": true
}
`), 0o644); err != nil {
		t.Fatalf("write config.json: %v", err)
	}

	opts, path, err := loadIntegrationStartupOptions()
	if err != nil {
		t.Fatalf("loadIntegrationStartupOptions: %v", err)
	}
	if path != configPath {
		t.Fatalf("path = %q, want %q", path, configPath)
	}
	if opts.Config.HTTPTimeout != 45000 {
		t.Fatalf("http_timeout = %d, want default 45000", opts.Config.HTTPTimeout)
	}
	if opts.Config.HTTPConnTimeout != 15000 {
		t.Fatalf("http_connect_timeout = %d, want default 15000", opts.Config.HTTPConnTimeout)
	}
	if opts.Config.HTTPSocketTimeout != 45000 {
		t.Fatalf("http_socket_timeout = %d, want default 45000", opts.Config.HTTPSocketTimeout)
	}
	if opts.Config.HTTPDebug {
		t.Fatal("http_debug = true, want default false")
	}
}

func TestLoadIntegrationStartupOptionsIgnoresAdjacentLegacyConfig(t *testing.T) {
	tempDir := t.TempDir()
	useIntegrationExecutableDir(t, tempDir)

	if err := os.WriteFile(filepath.Join(tempDir, "config.json"), []byte(`{"host":"http://legacy.example.com"}`), 0o644); err != nil {
		t.Fatalf("write legacy config.json: %v", err)
	}
	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"host":"http://www.dr.cn"}`), 0o644); err != nil {
		t.Fatalf("write config/config.json: %v", err)
	}

	opts, path, err := loadIntegrationStartupOptions()
	if err != nil {
		t.Fatalf("loadIntegrationStartupOptions: %v", err)
	}
	if path != configPath {
		t.Fatalf("path = %q, want %q", path, configPath)
	}
	if opts.Config.Host != "http://www.dr.cn" {
		t.Fatalf("host = %q, want http://www.dr.cn", opts.Config.Host)
	}
}

func TestLoadIntegrationStartupOptionsReadsBundledResourcesConfig(t *testing.T) {
	bundleRoot := filepath.Join(t.TempDir(), "integration.app")
	useBundledIntegrationExecutable(t, bundleRoot)

	configDir := filepath.Join(bundleRoot, "Contents", "Resources", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"host":"http://bundle.example.com"}`), 0o644); err != nil {
		t.Fatalf("write bundled config/config.json: %v", err)
	}

	opts, path, err := loadIntegrationStartupOptions()
	if err != nil {
		t.Fatalf("loadIntegrationStartupOptions: %v", err)
	}
	if path != configPath {
		t.Fatalf("path = %q, want %q", path, configPath)
	}
	if opts.Config.Host != "http://bundle.example.com" {
		t.Fatalf("host = %q, want http://bundle.example.com", opts.Config.Host)
	}
}

func TestLoadIntegrationStartupOptionsRejectsUnsupportedPort(t *testing.T) {
	tempDir := t.TempDir()
	useIntegrationExecutableDir(t, tempDir)

	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"port":19090}`), 0o644); err != nil {
		t.Fatalf("write config.json: %v", err)
	}

	_, path, err := loadIntegrationStartupOptions()
	if err == nil {
		t.Fatalf("expected unsupported port error")
	}
	if path != configPath {
		t.Fatalf("path = %q, want %q", path, configPath)
	}
	if !strings.Contains(err.Error(), "--port only supports 8080") {
		t.Fatalf("err = %v", err)
	}
}

func TestStartIntegrationProcessFailsWhenStartupConfigMalformed(t *testing.T) {
	tempDir := t.TempDir()
	useIntegrationExecutableDir(t, tempDir)

	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"host":`), 0o644); err != nil {
		t.Fatalf("write malformed config.json: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := startIntegrationProcess(nil, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected startIntegrationProcess to fail when startup config is malformed")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	stderrText := stderr.String()
	if !strings.Contains(stderrText, configPath) {
		t.Fatalf("stderr = %q, want config path %q", stderrText, configPath)
	}
	if !strings.Contains(stderrText, "parse config/config.json") {
		t.Fatalf("stderr = %q, want parse failure", stderrText)
	}
}

func TestBindIntegrationServeFlagsCLIOverridesStartupConfig(t *testing.T) {
	tempDir := t.TempDir()
	useIntegrationExecutableDir(t, tempDir)

	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{
  "host": "http://www.dr.cn",
  "port": 8080,
  "thread": 9,
  "pid-file": "startup.pid",
  "log-file": "startup.log"
}
`), 0o644); err != nil {
		t.Fatalf("write config.json: %v", err)
	}

	defaults, _, err := loadIntegrationStartupOptions()
	if err != nil {
		t.Fatalf("loadIntegrationStartupOptions: %v", err)
	}

	fs := flag.NewFlagSet("integration-serve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var cfg Config
	pidFile := ""
	logFile := ""
	startupStatusFile := ""
	bindIntegrationServeFlags(fs, &cfg, &pidFile, &logFile, &startupStatusFile, defaults)

	if err := fs.Parse([]string{"--host", "http://www.deepright.cn", "--thread", "4"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	if cfg.Host != "http://www.deepright.cn" {
		t.Fatalf("host = %q, want CLI value", cfg.Host)
	}
	if cfg.Port != integrationServicePort {
		t.Fatalf("port = %d, want fixed port %d", cfg.Port, integrationServicePort)
	}
	if cfg.Thread != 4 {
		t.Fatalf("thread = %d, want CLI value 4", cfg.Thread)
	}
	if pidFile != "startup.pid" {
		t.Fatalf("pid-file = %q, want startup.pid", pidFile)
	}
	if logFile != "startup.log" {
		t.Fatalf("log-file = %q, want startup.log", logFile)
	}
}

func TestLifecyclePathsAndPortFallBackToStartupConfig(t *testing.T) {
	tempDir := t.TempDir()
	useIntegrationExecutableDir(t, tempDir)

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{
  "port": 8080,
  "pidFile": "custom.pid",
  "log_file": "custom.log"
}
`), 0o644); err != nil {
		t.Fatalf("write config.json: %v", err)
	}

	if got := resolveIntegrationLaunchPort(map[string]string{}); got != integrationServicePort {
		t.Fatalf("resolveIntegrationLaunchPort() = %d, want %d", got, integrationServicePort)
	}
	if got := integrationPIDFilePath(map[string]string{}); got != integrationResolveAppPath("custom.pid") {
		t.Fatalf("integrationPIDFilePath() = %q, want %q", got, integrationResolveAppPath("custom.pid"))
	}
	if got := integrationLogFilePath(map[string]string{}); got != integrationResolveAppPath("custom.log") {
		t.Fatalf("integrationLogFilePath() = %q, want %q", got, integrationResolveAppPath("custom.log"))
	}
}

func TestWSLDefaultRuntimePaths(t *testing.T) {
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home", "tester")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	useIntegrationExecutableDir(t, tempDir)

	oldRuntimeGOOS := integrationRuntimeGOOS
	oldHomeFn := integrationUserHomeFn
	oldGetenv := integrationBrowserGetenvFn
	defer func() {
		integrationRuntimeGOOS = oldRuntimeGOOS
		integrationUserHomeFn = oldHomeFn
		integrationBrowserGetenvFn = oldGetenv
	}()

	integrationRuntimeGOOS = "linux"
	integrationUserHomeFn = func() (string, error) { return homeDir, nil }
	integrationBrowserGetenvFn = os.Getenv
	t.Setenv("WSL_DISTRO_NAME", "Ubuntu")
	t.Setenv(integrationRuntimeDirEnv, "")

	runtimeDir, err := prepareIntegrationRuntimeBaseDir()
	if err != nil {
		t.Fatalf("prepareIntegrationRuntimeBaseDir: %v", err)
	}
	wantRuntimeDir := filepath.Join(homeDir, integrationRuntimeAppDir)
	if runtimeDir != wantRuntimeDir {
		t.Fatalf("runtimeDir = %q, want %q", runtimeDir, wantRuntimeDir)
	}
	if info, err := os.Stat(runtimeDir); err != nil || !info.IsDir() {
		t.Fatalf("runtime dir not created: err=%v", err)
	}

	pluginDir, err := resolveIntegrationRuntimePluginDirFromRuntimeBase(runtimeDir)
	if err != nil {
		t.Fatalf("resolveIntegrationRuntimePluginDirFromRuntimeBase: %v", err)
	}
	wantPluginDir := filepath.Join(wantRuntimeDir, "plugins")
	if pluginDir != wantPluginDir {
		t.Fatalf("pluginDir = %q, want %q", pluginDir, wantPluginDir)
	}
	if info, err := os.Stat(pluginDir); err != nil || !info.IsDir() {
		t.Fatalf("plugin dir not created: err=%v", err)
	}

	if got := integrationPIDFilePath(map[string]string{}); got != filepath.Join(wantRuntimeDir, integrationDefaultPIDFile) {
		t.Fatalf("integrationPIDFilePath() = %q, want %q", got, filepath.Join(wantRuntimeDir, integrationDefaultPIDFile))
	}
	if got := integrationLogFilePath(map[string]string{}); got != filepath.Join(wantRuntimeDir, integrationDefaultLogFile) {
		t.Fatalf("integrationLogFilePath() = %q, want %q", got, filepath.Join(wantRuntimeDir, integrationDefaultLogFile))
	}

	knowledgeDir, err := knowledgecore.KnowledgeDir(wantRuntimeDir)
	if err != nil {
		t.Fatalf("KnowledgeDir: %v", err)
	}
	wantKnowledgeDir := filepath.Join(wantRuntimeDir, "knowledge")
	if knowledgeDir != wantKnowledgeDir {
		t.Fatalf("knowledgeDir = %q, want %q", knowledgeDir, wantKnowledgeDir)
	}
	if _, err := knowledgecore.EnsureRuntime(wantRuntimeDir); err != nil {
		t.Fatalf("EnsureRuntime: %v", err)
	}
	if info, err := os.Stat(wantKnowledgeDir); err != nil || !info.IsDir() {
		t.Fatalf("knowledge dir not created: err=%v", err)
	}
}

func TestPrepareIntegrationRuntimeBaseDirDoesNotMigrateLegacyRuntimeData(t *testing.T) {
	runtimeDir := t.TempDir()
	exeDir := t.TempDir()
	useIntegrationExecutableDir(t, exeDir)
	t.Setenv(integrationRuntimeDirEnv, runtimeDir)
	oldRuntimeGOOS := integrationRuntimeGOOS
	oldGetenv := integrationBrowserGetenvFn
	defer func() {
		integrationRuntimeGOOS = oldRuntimeGOOS
		integrationBrowserGetenvFn = oldGetenv
	}()
	integrationRuntimeGOOS = "linux"
	integrationBrowserGetenvFn = os.Getenv
	t.Setenv("WSL_DISTRO_NAME", "")

	if err := os.WriteFile(filepath.Join(exeDir, "data"), []byte("legacy-db"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := prepareIntegrationRuntimeBaseDir()
	if err != nil {
		t.Fatalf("prepareIntegrationRuntimeBaseDir: %v", err)
	}
	if got != runtimeDir {
		t.Fatalf("runtimeDir = %q, want %q", got, runtimeDir)
	}
	if _, err := os.Stat(filepath.Join(runtimeDir, "data")); !os.IsNotExist(err) {
		t.Fatalf("runtime data should not be migrated, stat err = %v", err)
	}
}

func TestRunIntegrationForegroundWritesStartupFailureStatusOnListenError(t *testing.T) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", integrationServicePort))
	if err != nil {
		t.Skipf("listen on fixed port %d: %v", integrationServicePort, err)
	}
	defer listener.Close()
	tempDir := t.TempDir()
	startupStatusFile := filepath.Join(tempDir, "startup.json")
	pidFile := filepath.Join(tempDir, "integration.pid")
	agentDir := filepath.Join(tempDir, "agent")
	defaultDir := filepath.Join(tempDir, "config")
	siteDir := filepath.Join(tempDir, "site")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}
	writeDefaultAgentTemplate(t, defaultDir)
	if err := os.MkdirAll(siteDir, 0o755); err != nil {
		t.Fatalf("mkdir site dir: %v", err)
	}

	args := []string{
		"--port", strconv.Itoa(integrationServicePort),
		"--agent-dir", agentDir,
		"--default-dir", defaultDir,
		"--site", siteDir,
		"--pid-file", pidFile,
		"--startup-status-file", startupStatusFile,
	}

	var stderr bytes.Buffer
	code := runIntegrationForeground(args, &stderr)
	if code == 0 {
		t.Fatalf("expected foreground start to fail when port is occupied")
	}
	if !strings.Contains(stderr.String(), "server listen failed:") {
		t.Fatalf("stderr = %q, want listen failure", stderr.String())
	}
	status, err := readIntegrationStartupStatus(startupStatusFile)
	if err != nil {
		t.Fatalf("read startup status: %v", err)
	}
	if status.Status != "failed" {
		t.Fatalf("status = %q, want failed", status.Status)
	}
	if !strings.Contains(status.Message, "server listen failed:") {
		t.Fatalf("message = %q, want listen failure", status.Message)
	}
}

func TestEnsureIntegrationGitIdentityConfiguredSetsDefaultsWhenMissing(t *testing.T) {
	prevLookPathFn := integrationGitLookPathFn
	prevReadConfigFn := integrationGitReadConfigFn
	prevWriteConfigFn := integrationGitWriteConfigFn
	t.Cleanup(func() {
		integrationGitLookPathFn = prevLookPathFn
		integrationGitReadConfigFn = prevReadConfigFn
		integrationGitWriteConfigFn = prevWriteConfigFn
	})

	integrationGitLookPathFn = func(name string) (string, error) {
		if name != "git" {
			t.Fatalf("unexpected lookpath name = %q", name)
		}
		return "/usr/bin/git", nil
	}
	integrationGitReadConfigFn = func(key string) (string, error) {
		switch key {
		case "user.name", "user.email":
			return "", nil
		default:
			t.Fatalf("unexpected read key = %q", key)
			return "", nil
		}
	}

	var writes []string
	integrationGitWriteConfigFn = func(key, value string) error {
		writes = append(writes, key+"="+value)
		return nil
	}

	if err := ensureIntegrationGitIdentityConfigured(); err != nil {
		t.Fatalf("ensureIntegrationGitIdentityConfigured() error = %v", err)
	}
	want := []string{
		"user.email=" + integrationDefaultGitUserEmail,
		"user.name=" + integrationDefaultGitUserName,
	}
	if !reflect.DeepEqual(writes, want) {
		t.Fatalf("writes = %#v, want %#v", writes, want)
	}
}

func TestEnsureIntegrationGitIdentityConfiguredPreservesExistingValues(t *testing.T) {
	prevLookPathFn := integrationGitLookPathFn
	prevReadConfigFn := integrationGitReadConfigFn
	prevWriteConfigFn := integrationGitWriteConfigFn
	t.Cleanup(func() {
		integrationGitLookPathFn = prevLookPathFn
		integrationGitReadConfigFn = prevReadConfigFn
		integrationGitWriteConfigFn = prevWriteConfigFn
	})

	integrationGitLookPathFn = func(name string) (string, error) {
		if name != "git" {
			t.Fatalf("unexpected lookpath name = %q", name)
		}
		return "/usr/bin/git", nil
	}
	integrationGitReadConfigFn = func(key string) (string, error) {
		switch key {
		case "user.name":
			return "existing-user", nil
		case "user.email":
			return "existing@example.com", nil
		default:
			t.Fatalf("unexpected read key = %q", key)
			return "", nil
		}
	}
	integrationGitWriteConfigFn = func(key, value string) error {
		t.Fatalf("unexpected write %s=%s", key, value)
		return nil
	}

	if err := ensureIntegrationGitIdentityConfigured(); err != nil {
		t.Fatalf("ensureIntegrationGitIdentityConfigured() error = %v", err)
	}
}

func TestRunIntegrationForegroundInvokesGitIdentityEnsure(t *testing.T) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", integrationServicePort))
	if err != nil {
		t.Skipf("listen on fixed port %d: %v", integrationServicePort, err)
	}
	defer listener.Close()

	prevEnsureFn := integrationEnsureGitIdentityConfiguredFn
	t.Cleanup(func() {
		integrationEnsureGitIdentityConfiguredFn = prevEnsureFn
	})

	calls := 0
	integrationEnsureGitIdentityConfiguredFn = func() error {
		calls++
		return nil
	}

	tempDir := t.TempDir()
	startupStatusFile := filepath.Join(tempDir, "startup.json")
	pidFile := filepath.Join(tempDir, "integration.pid")
	agentDir := filepath.Join(tempDir, "agent")
	defaultDir := filepath.Join(tempDir, "config")
	siteDir := filepath.Join(tempDir, "site")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}
	writeDefaultAgentTemplate(t, defaultDir)
	if err := os.MkdirAll(siteDir, 0o755); err != nil {
		t.Fatalf("mkdir site dir: %v", err)
	}

	args := []string{
		"--port", strconv.Itoa(integrationServicePort),
		"--agent-dir", agentDir,
		"--default-dir", defaultDir,
		"--site", siteDir,
		"--pid-file", pidFile,
		"--startup-status-file", startupStatusFile,
	}

	var stderr bytes.Buffer
	if code := runIntegrationForeground(args, &stderr); code == 0 {
		t.Fatalf("expected foreground start to fail when port is occupied")
	}
	if calls != 1 {
		t.Fatalf("git identity ensure calls = %d, want 1", calls)
	}
}

func TestOpenIntegrationBrowserAfterEntryReady(t *testing.T) {
	oldOpenBrowser := integrationOpenBrowserFn
	oldBrowserEntryReadyCheck := integrationBrowserEntryReadyCheck
	oldBrowserEntryWait := integrationBrowserEntryWait
	oldBrowserEntryPollInterval := integrationBrowserEntryPollInterval
	t.Cleanup(func() {
		integrationOpenBrowserFn = oldOpenBrowser
		integrationBrowserEntryReadyCheck = oldBrowserEntryReadyCheck
		integrationBrowserEntryWait = oldBrowserEntryWait
		integrationBrowserEntryPollInterval = oldBrowserEntryPollInterval
	})
	port := 18086

	entryChecks := 0
	integrationBrowserEntryReadyCheck = func(int) bool {
		entryChecks++
		return entryChecks >= 3
	}
	integrationBrowserEntryWait = 200 * time.Millisecond
	integrationBrowserEntryPollInterval = 5 * time.Millisecond

	var opened string
	integrationOpenBrowserFn = func(target string) error {
		opened = target
		return nil
	}

	openIntegrationBrowserAfterEntryReady(port, nil)

	if opened != integrationBrowserOpenURL(port) {
		t.Fatalf("opened = %q, want %q", opened, integrationBrowserOpenURL(port))
	}
	if entryChecks < 3 {
		t.Fatalf("entryChecks = %d, want at least 3", entryChecks)
	}
}

func TestNormalizeIntegrationLaunchArgsDropsMacProcessSerialNumber(t *testing.T) {
	args := []string{"-psn_0_12345", "--port", "18080"}
	got := normalizeIntegrationLaunchArgs(args)
	want := []string{"--port", "18080"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeIntegrationLaunchArgs() = %#v, want %#v", got, want)
	}
}

func TestNormalizeIntegrationLaunchArgsPreservesCLICommands(t *testing.T) {
	args := []string{"serve", "--port", "18080"}
	got := normalizeIntegrationLaunchArgs(args)
	if !reflect.DeepEqual(got, args) {
		t.Fatalf("normalizeIntegrationLaunchArgs() = %#v, want %#v", got, args)
	}
}

func TestShouldHandleIntegrationBundleAppLaunchWithContext(t *testing.T) {
	layout := &integrationBundlePaths{BundleRoot: "/tmp/integration.app"}
	if !shouldHandleIntegrationBundleAppLaunchWithContext(nil, "cn.deepright.integration", layout) {
		t.Fatalf("expected bundled app launch to be handled")
	}
	if shouldHandleIntegrationBundleAppLaunchWithContext([]string{"serve"}, "cn.deepright.integration", layout) {
		t.Fatalf("explicit CLI args should not use bundled app launch mode")
	}
	if shouldHandleIntegrationBundleAppLaunchWithContext(nil, "", layout) {
		t.Fatalf("missing bundle identifier should not use bundled app launch mode")
	}
	if shouldHandleIntegrationBundleAppLaunchWithContext(nil, "cn.deepright.integration", nil) {
		t.Fatalf("missing bundle layout should not use bundled app launch mode")
	}
}

func TestIntegrationBundleLaunchAllowedAnyLocation(t *testing.T) {
	oldHome := integrationUserHomeFn
	oldEvalSymlinks := integrationEvalSymlinksFn
	defer func() {
		integrationUserHomeFn = oldHome
		integrationEvalSymlinksFn = oldEvalSymlinks
	}()

	integrationUserHomeFn = func() (string, error) { return "/Users/tester", nil }
	integrationEvalSymlinksFn = func(path string) (string, error) { return filepath.Clean(path), nil }

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "system applications", path: "/Applications/DeepRight.app", want: true},
		{name: "user applications", path: "/Users/tester/Applications/DeepRight.app", want: true},
		{name: "downloads", path: "/Users/tester/Downloads/DeepRight.app", want: true},
		{name: "desktop", path: "/Users/tester/Desktop/DeepRight.app", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout := &integrationBundlePaths{BundleRoot: tt.path}
			if got := integrationBundleLaunchAllowed(layout); got != tt.want {
				t.Fatalf("integrationBundleLaunchAllowed(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestResolveIntegrationLaunchPort(t *testing.T) {
	if got := resolveIntegrationLaunchPort(map[string]string{"port": "18081"}); got != integrationServicePort {
		t.Fatalf("resolveIntegrationLaunchPort() = %d, want %d", got, integrationServicePort)
	}
	if got := resolveIntegrationLaunchPort(map[string]string{"port": "invalid"}); got != integrationServicePort {
		t.Fatalf("resolveIntegrationLaunchPort() invalid = %d, want %d", got, integrationServicePort)
	}
	if got := resolveIntegrationLaunchPort(nil); got != integrationServicePort {
		t.Fatalf("resolveIntegrationLaunchPort() nil = %d, want %d", got, integrationServicePort)
	}
}

func TestFormatCliGetHeartbeatErrorIncludesHost(t *testing.T) {
	errText := formatCliGetHeartbeatError("http://127.0.0.1:9998", errors.New("heartbeat HTTP 500"))
	if errText != "cli-get: heartbeat error (host=http://127.0.0.1:9998): heartbeat HTTP 500" {
		t.Fatalf("formatCliGetHeartbeatError() = %q", errText)
	}
}

func TestShouldSleepAfterHeartbeat(t *testing.T) {
	task := &TaskContent{Tid: "task-1"}

	if shouldSleepAfterHeartbeat(nil, nil) {
		t.Fatal("shouldSleepAfterHeartbeat(nil, nil) = true, want false")
	}
	if shouldSleepAfterHeartbeat(task, nil) {
		t.Fatal("shouldSleepAfterHeartbeat(task, nil) = true, want false")
	}
	if !shouldSleepAfterHeartbeat(nil, io.EOF) {
		t.Fatal("shouldSleepAfterHeartbeat(nil, err) = false, want true")
	}
}

func TestIntegrationBrowserURL(t *testing.T) {
	oldTokenFn := integrationSiteVersionTokenFn
	integrationSiteVersionTokenFn = func() string { return "site-version-token" }
	defer func() { integrationSiteVersionTokenFn = oldTokenFn }()

	if got := integrationBrowserURL(8080); got != "http://localhost:8080/site/?v=site-version-token#app" {
		t.Fatalf("browser url = %q", got)
	}
	if got := integrationBrowserURL(0); got != "http://localhost:8080/site/?v=site-version-token#app" {
		t.Fatalf("default browser url = %q", got)
	}
}

func TestIntegrationBrowserOpenURL(t *testing.T) {
	if got := integrationBrowserOpenURL(8080); got != "http://localhost:8080/launch" {
		t.Fatalf("browser open url = %q", got)
	}
	if got := integrationBrowserOpenURL(0); got != "http://localhost:8080/launch" {
		t.Fatalf("default browser open url = %q", got)
	}
}

func TestHandleIntegrationBrowserLaunch(t *testing.T) {
	oldTokenFn := integrationSiteVersionTokenFn
	integrationSiteVersionTokenFn = func() string { return "site-version-token" }
	defer func() { integrationSiteVersionTokenFn = oldTokenFn }()

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:18080/launch", nil)
	rec := httptest.NewRecorder()

	handleIntegrationBrowserLaunch().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store, no-cache, must-revalidate" {
		t.Fatalf("Cache-Control = %q", got)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "window.location.replace(target)") {
		t.Fatalf("launch body missing redirect script: %q", body)
	}
	if !strings.Contains(body, "http://127.0.0.1:18080/site/?v=site-version-token#app") {
		t.Fatalf("launch body missing target site url: %q", body)
	}
}

func TestIntegrationShouldOpenBrowser(t *testing.T) {
	original := os.Getenv(integrationSkipBrowserEnv)
	defer func() {
		if original == "" {
			_ = os.Unsetenv(integrationSkipBrowserEnv)
			return
		}
		_ = os.Setenv(integrationSkipBrowserEnv, original)
	}()

	if err := os.Unsetenv(integrationSkipBrowserEnv); err != nil {
		t.Fatal(err)
	}
	if !integrationShouldOpenBrowser() {
		t.Fatal("expected browser open when skip env is unset")
	}
	if err := os.Setenv(integrationSkipBrowserEnv, "1"); err != nil {
		t.Fatal(err)
	}
	if integrationShouldOpenBrowser() {
		t.Fatal("expected browser open to be skipped when env is set")
	}
}

func TestMaybeOpenIntegrationNotificationSettingsWhenPermissionDenied(t *testing.T) {
	oldShould := integrationShouldOpenNotificationSettingsFn
	oldOpen := integrationOpenNotificationSettingsFn
	t.Cleanup(func() {
		integrationShouldOpenNotificationSettingsFn = oldShould
		integrationOpenNotificationSettingsFn = oldOpen
	})

	integrationShouldOpenNotificationSettingsFn = func(err error) bool {
		return err == integrationnotification.ErrNotificationsNotAllowed
	}

	var opened int
	integrationOpenNotificationSettingsFn = func() error {
		opened++
		return nil
	}

	maybeOpenIntegrationNotificationSettings(integrationnotification.ErrNotificationsNotAllowed)

	if opened != 1 {
		t.Fatalf("opened count = %d, want 1", opened)
	}
}

func TestMaybeOpenIntegrationNotificationSettingsIgnoresOtherErrors(t *testing.T) {
	oldShould := integrationShouldOpenNotificationSettingsFn
	oldOpen := integrationOpenNotificationSettingsFn
	t.Cleanup(func() {
		integrationShouldOpenNotificationSettingsFn = oldShould
		integrationOpenNotificationSettingsFn = oldOpen
	})

	integrationShouldOpenNotificationSettingsFn = func(error) bool { return false }

	var opened int
	integrationOpenNotificationSettingsFn = func() error {
		opened++
		return nil
	}

	maybeOpenIntegrationNotificationSettings(errors.New("other"))

	if opened != 0 {
		t.Fatalf("opened count = %d, want 0", opened)
	}
}

// ── cli-get test: heartbeat + execute + publish ──

func TestCliGetHeartbeatAndExecute(t *testing.T) {
	taskJSON := `{"timeout":5000,"suffix":"cmd","type":"cmd","tid":"t001","cmd":"echo integration_test"}`

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

	var capturedGet map[string]interface{}
	var capturedPub PubRequest

	mux := http.NewServeMux()
	mux.HandleFunc("/cli/get", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedGet)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ResponsePayload{
			Code: 200,
			Choices: []struct {
				Message struct {
					Role    string      `json:"role"`
					Content interface{} `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Role    string      `json:"role"`
					Content interface{} `json:"content"`
				}{Role: "assistant", Content: taskJSON}},
			},
		})
	})
	mux.HandleFunc("/cli/pub", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedPub)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ResponsePayload{Code: 200})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	workspace := filepath.Join(tmp, "agent-a")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "knowledge.md"), []byte("agent knowledge"), 0o644); err != nil {
		t.Fatalf("write knowledge.md: %v", err)
	}
	metadata := &AgentOutput{
		DeviceID: "d", Terminal: "/bin/zsh", Plugins: []string{"browser", "feishu"}, Git: detectGit(), Gateway: "g", Sys: "s",
		Knowledge: &agentcore.Knowledge{
			Path:       filepath.Join(tmp, "knowledge"),
			LastUpdate: 0,
		},
		Agents: []Agent{{Workspace: workspace, AgentID: "a", Skills: []Skill{}}},
	}

	// Test heartbeat
	task, err := heartbeat(client, server.URL, metadata, nil)
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if task == nil || task.Tid != "t001" {
		t.Fatalf("task: %v", task)
	}

	// Verify get request format
	getMessages, ok := capturedGet["messages"].([]interface{})
	if !ok || len(getMessages) != 1 {
		t.Fatalf("get messages = %#v, want one heartbeat message", capturedGet["messages"])
	}
	getMessage, ok := getMessages[0].(map[string]interface{})
	if !ok {
		t.Fatalf("heartbeat message malformed: %#v", getMessages[0])
	}
	if getMessage["role"] != "user" || getMessage["content"] != "" {
		t.Errorf("get message = %#v, want user/empty", getMessage)
	}
	if _, exists := capturedGet["message"]; exists {
		t.Errorf("get message should be absent: %v", capturedGet["message"])
	}
	if _, exists := capturedGet["stream"]; exists {
		t.Errorf("get stream should be absent: %v", capturedGet["stream"])
	}
	getMeta := capturedGet["metadata"].(map[string]interface{})
	if getMeta["deviceId"] != "d" {
		t.Errorf("get deviceId = %v", getMeta["deviceId"])
	}
	if getMeta["git"] != metadata.Git {
		t.Errorf("get git = %v, want %q", getMeta["git"], metadata.Git)
	}
	if gotPlugins, ok := getMeta["plugins"].([]interface{}); !ok || len(gotPlugins) != 2 || gotPlugins[0] != "browser" || gotPlugins[1] != "feishu" {
		t.Errorf("get plugins = %#v", getMeta["plugins"])
	}
	if getKnowledge, ok := getMeta["knowledge"].(map[string]interface{}); !ok {
		t.Fatalf("get knowledge = %#v", getMeta["knowledge"])
	} else {
		if getKnowledge["lastUpdate"] != float64(0) {
			t.Errorf("get knowledge.lastUpdate = %v, want 0", getKnowledge["lastUpdate"])
		}
		if _, exists := getKnowledge["knowledgeCommit"]; exists {
			t.Fatalf("get knowledge.knowledgeCommit should not be forwarded: %#v", getKnowledge)
		}
	}
	getAgents, ok := getMeta["agents"].([]interface{})
	if !ok || len(getAgents) != 1 {
		t.Fatalf("get agents = %#v", getMeta["agents"])
	}
	getAgent, ok := getAgents[0].(map[string]interface{})
	if !ok {
		t.Fatalf("get agent = %#v", getAgents[0])
	}
	if getAgent["knowledge"] != "agent knowledge" {
		t.Fatalf("get agent knowledge = %#v, want %q", getAgent["knowledge"], "agent knowledge")
	}

	// Test execute
	terminal := detectTerminal()
	if terminal == "" {
		t.Skip("no SHELL")
	}
	result := executeTask(task, terminal)
	if result.Status != 0 {
		t.Errorf("status = %d", result.Status)
	}
	if result.Suffix != "cmd" || result.Type != "cmd" || result.Tid != "t001" {
		t.Errorf("result fields wrong: %+v", result)
	}

	// Decode output
	decoded, _ := base64.StdEncoding.DecodeString(result.Cmd)
	gz, _ := gzip.NewReader(strings.NewReader(string(decoded)))
	output, _ := io.ReadAll(gz)
	if !strings.Contains(string(output), "integration_test") {
		t.Errorf("output = %q", string(output))
	}

	// Test publish
	if err := publishResult(client, server.URL, result, metadata, nil); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if capturedPub.Model != "" {
		t.Errorf("pub model = %q", capturedPub.Model)
	}
	if capturedPub.Metadata == nil {
		t.Fatal("pub metadata should not be nil")
	}
	if capturedPub.Metadata.DeviceID != "d" {
		t.Errorf("pub metadata deviceId = %q, want d", capturedPub.Metadata.DeviceID)
	}
	if capturedPub.Metadata.Knowledge == nil {
		t.Fatal("pub metadata knowledge should not be nil")
	}
	var pubResult ResultPayload
	json.Unmarshal([]byte(capturedPub.Messages[0].Content), &pubResult)
	if pubResult.Tid != "t001" || pubResult.Status != 0 {
		t.Errorf("pub result: %+v", pubResult)
	}
}

func TestHeartbeatInjectsSandboxPathForCurrentPageSession(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	if cronDB != nil {
		_ = cronDB.Close()
		cronDB = nil
	}
	initCronDB()
	if cronDB == nil {
		t.Fatal("cronDB should not be nil")
	}
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	createdAt := time.Now().Format(roundLogStorageTimeLayout)
	if _, err := cronDB.Exec(`INSERT INTO chat_log(agent_id, chat_id, chat_type, role, response_type, content, created_at) VALUES (?,?,?,?,?,?,?)`,
		"A", "chat-sandbox", chatTypePageSession, "A", "normal", "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n", createdAt); err != nil {
		t.Fatalf("insert chat_log: %v", err)
	}
	if _, err := sandboxstate.SetWithDirectory(cronDB, "A", "chat-sandbox", sandboxstate.ModeFilePick, "/Users/demo/Desktop"); err != nil {
		t.Fatalf("set sandbox: %v", err)
	}

	var capturedGet map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedGet)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ResponsePayload{
			Code: 200,
			Choices: []struct {
				Message struct {
					Role    string      `json:"role"`
					Content interface{} `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Role    string      `json:"role"`
					Content interface{} `json:"content"`
				}{Role: "assistant", Content: nil}},
			},
		})
	}))
	defer server.Close()

	metadata := &AgentOutput{
		DeviceID: "d",
		Terminal: "/bin/zsh",
		Agents:   []Agent{{AgentID: "A", Workspace: "/tmp/a", Skills: []Skill{}}},
	}
	if _, err := heartbeat(&http.Client{Timeout: 5 * time.Second}, server.URL, metadata, nil); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}

	meta, ok := capturedGet["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata missing: %#v", capturedGet["metadata"])
	}
	if meta["sandbox_path"] != "/Users/demo/Desktop" {
		t.Fatalf("metadata.sandbox_path = %#v, want /Users/demo/Desktop", meta["sandbox_path"])
	}
}

func TestHeartbeatRefreshesMediaWithoutWaitingForAgentCache(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	agentDir := filepath.Join(tmp, "agents", "a")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent: %v", err)
	}
	configPath := filepath.Join(agentDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"media":{"cover":"first.png"}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var covers []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var captured map[string]interface{}
		_ = json.Unmarshal(body, &captured)
		cover := ""
		if meta, ok := captured["metadata"].(map[string]interface{}); ok {
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
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ResponsePayload{
			Code: 200,
			Choices: []struct {
				Message struct {
					Role    string      `json:"role"`
					Content interface{} `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Role    string      `json:"role"`
					Content interface{} `json:"content"`
				}{Role: "assistant", Content: nil}},
			},
		})
	}))
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	metadata := &AgentOutput{
		DeviceID: "d",
		Agents:   []Agent{{AgentID: "a", Workspace: agentDir}},
	}

	if _, err := heartbeat(client, server.URL, metadata, nil); err != nil {
		t.Fatalf("heartbeat first: %v", err)
	}
	if len(covers) != 1 || covers[0] != "first.png" {
		t.Fatalf("first media covers = %#v, want first.png", covers)
	}

	if err := os.WriteFile(configPath, []byte(`{"media":{"cover":"second.png"}}`), 0o644); err != nil {
		t.Fatalf("rewrite config: %v", err)
	}
	if _, err := heartbeat(client, server.URL, metadata, nil); err != nil {
		t.Fatalf("heartbeat second: %v", err)
	}
	if len(covers) != 2 || covers[1] != "second.png" {
		t.Fatalf("second media covers = %#v, want second.png", covers)
	}
}

func TestHeartbeatInjectsLastResponseForLatestPageSessionChat(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	initCronDB()
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	expectedAt := time.Date(2026, time.July, 3, 12, 34, 56, 789000000, time.Local)
	createdAt := expectedAt.Format(roundLogStorageTimeLayout)
	if _, err := cronDB.Exec(`INSERT INTO chat_log (agent_id, chat_id, chat_type, role, response_type, content, created_at) VALUES (?,?,?,?,?,?,?)`,
		"A", "chat-current", chatTypePageSession, "A", "normal", "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n", createdAt); err != nil {
		t.Fatalf("insert chat_log: %v", err)
	}
	if _, err := cronDB.Exec(`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES (?,?,?,?,?)`,
		"A", "chat-current", "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n", logTypeChatCompletionResponse, createdAt); err != nil {
		t.Fatalf("insert agent_message_log: %v", err)
	}

	var capturedGet map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedGet)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ResponsePayload{
			Code: 200,
			Choices: []struct {
				Message struct {
					Role    string      `json:"role"`
					Content interface{} `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Role    string      `json:"role"`
					Content interface{} `json:"content"`
				}{Role: "assistant", Content: nil}},
			},
		})
	}))
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	metadata := &AgentOutput{
		DeviceID: "d",
		Agents:   []Agent{{AgentID: "A", Workspace: "/tmp/a"}},
	}

	if _, err := heartbeat(client, server.URL, metadata, nil); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}

	meta, ok := capturedGet["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata missing: %#v", capturedGet["metadata"])
	}
	gotLastResponse, ok := meta["lastResponse"].(float64)
	if !ok || int64(gotLastResponse) != expectedAt.UnixMilli() {
		t.Fatalf("metadata.lastResponse = %#v, want %d", meta["lastResponse"], expectedAt.UnixMilli())
	}
}

func TestCliGetWritesUnifiedEventLogs(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)
	db, err := sql.Open("sqlite", filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	cronDB = db
	if _, err := cronDB.Exec(`CREATE TABLE IF NOT EXISTS agent_message_log (id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL DEFAULT '', chat_id TEXT NOT NULL DEFAULT '', content TEXT NOT NULL, log_type INTEGER NOT NULL, created_at TEXT NOT NULL)`); err != nil {
		t.Fatalf("create agent_message_log: %v", err)
	}
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	taskJSON := `{"timeout":5000,"suffix":"cmd","type":"cmd","tid":"t002","cmd":"echo hi","agentId":"a","chat":"chat-1"}`
	mux := http.NewServeMux()
	mux.HandleFunc("/cli/get", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ResponsePayload{
			Code: 200,
			Choices: []struct {
				Message struct {
					Role    string      `json:"role"`
					Content interface{} `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Role    string      `json:"role"`
					Content interface{} `json:"content"`
				}{Role: "assistant", Content: taskJSON}},
			},
		})
	})
	mux.HandleFunc("/cli/pub", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ResponsePayload{Code: 200})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	meta := &AgentOutput{DeviceID: "d", Terminal: "/bin/zsh", Agents: []Agent{{AgentID: "a", Workspace: "/tmp/a"}}}
	task, err := heartbeat(client, server.URL, meta, nil)
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if task == nil {
		t.Fatal("expected task")
	}
	result := &ResultPayload{Status: 0, AgentId: "a", Chat: "chat-1", Suffix: "cmd", Type: "cmd", Cmd: gzipBase64String("hello from cli pub raw"), Tid: "t002"}
	if err := publishResult(client, server.URL, result, meta, nil); err != nil {
		t.Fatalf("publish: %v", err)
	}

	var getCount, pubCount int
	if err := cronDB.QueryRow(`SELECT COUNT(*) FROM agent_message_log WHERE agent_id = ? AND chat_id = ? AND log_type = ?`,
		"a", "chat-1", logTypeCLIGet).Scan(&getCount); err != nil {
		t.Fatalf("count cli/get logs: %v", err)
	}
	if err := cronDB.QueryRow(`SELECT COUNT(*) FROM agent_message_log WHERE agent_id = ? AND chat_id = ? AND log_type = ?`,
		"a", "chat-1", logTypeCLIPub).Scan(&pubCount); err != nil {
		t.Fatalf("count cli/pub logs: %v", err)
	}
	if getCount != 1 || pubCount != 1 {
		t.Fatalf("counts = (%d,%d), want (1,1)", getCount, pubCount)
	}
	var storedPub string
	if err := cronDB.QueryRow(`SELECT content FROM agent_message_log WHERE agent_id = ? AND chat_id = ? AND log_type = ? ORDER BY id DESC LIMIT 1`,
		"a", "chat-1", logTypeCLIPub).Scan(&storedPub); err != nil {
		t.Fatalf("select cli/pub content: %v", err)
	}
	if storedPub != "hello from cli pub raw" {
		t.Fatalf("cli/pub content = %q, want raw output", storedPub)
	}
}

func TestPublishResultIncludesPendingMessageInsertAndMarksPublished(t *testing.T) {
	oldCronDB := cronDB
	defer func() {
		if cronDB != nil && cronDB != oldCronDB {
			_ = cronDB.Close()
		}
		cronDB = oldCronDB
	}()

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	cronDB = db
	if err := integrationmessageinsert.EnsureSchema(cronDB); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	if _, err := integrationmessageinsert.UpsertPending(cronDB, "agent-a", "chat-1", "1710000000000", "first insert", time.Now()); err != nil {
		t.Fatalf("insert first pending: %v", err)
	}
	if _, err := integrationmessageinsert.UpsertPending(cronDB, "agent-a", "chat-1", "1710000005000", "second insert", time.Now().Add(time.Millisecond)); err != nil {
		t.Fatalf("insert second pending: %v", err)
	}

	var captured PubRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode publish request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ResponsePayload{Code: 200})
	}))
	defer server.Close()

	result := &ResultPayload{
		Status:  0,
		AgentId: "agent-a",
		Chat:    "chat-1",
		Suffix:  "cmd",
		Type:    "cmd",
		Cmd:     gzipBase64String("hello from cli pub"),
		Tid:     "tid-with-insert",
	}
	if err := publishResult(&http.Client{Timeout: time.Second}, server.URL, result, &AgentOutput{}, nil); err != nil {
		t.Fatalf("publishResult: %v", err)
	}

	if len(captured.Messages) != 1 {
		t.Fatalf("messages = %#v", captured.Messages)
	}
	var payload ResultPayload
	if err := json.Unmarshal([]byte(captured.Messages[0].Content), &payload); err != nil {
		t.Fatalf("decode published payload: %v", err)
	}
	if len(payload.Insert) != 2 {
		t.Fatalf("insert payload count = %d, want 2", len(payload.Insert))
	}
	if payload.Insert[0].Tid != "1710000000000" || payload.Insert[0].Message != "first insert" {
		t.Fatalf("first insert payload = %#v", payload.Insert[0])
	}
	if payload.Insert[1].Tid != "1710000005000" || payload.Insert[1].Message != "second insert" {
		t.Fatalf("second insert payload = %#v", payload.Insert[1])
	}

	statuses, err := integrationmessageinsert.StatusByTIDs(cronDB, "chat-1", []string{"1710000000000", "1710000005000"})
	if err != nil {
		t.Fatalf("status query: %v", err)
	}
	if statuses["1710000000000"] != integrationmessageinsert.StatusPending {
		t.Fatalf("status first = %d, want %d", statuses["1710000000000"], integrationmessageinsert.StatusPending)
	}
	if statuses["1710000005000"] != integrationmessageinsert.StatusPending {
		t.Fatalf("status second = %d, want %d", statuses["1710000005000"], integrationmessageinsert.StatusPending)
	}

	pending, err := integrationmessageinsert.ListPending(cronDB, "chat-1", 5)
	if err != nil {
		t.Fatalf("ListPending after publish: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending after publish = %#v, want none", pending)
	}
}

func TestMarkConfirmedMessageInsertUploadsFromSSE(t *testing.T) {
	oldCronDB := cronDB
	defer func() {
		if cronDB != nil && cronDB != oldCronDB {
			_ = cronDB.Close()
		}
		cronDB = oldCronDB
	}()

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	cronDB = db
	if err := integrationmessageinsert.EnsureSchema(cronDB); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	if _, err := integrationmessageinsert.UpsertPending(cronDB, "agent-a", "chat-1", "1710000000000", "first insert", time.Now()); err != nil {
		t.Fatalf("insert first pending: %v", err)
	}
	if _, err := integrationmessageinsert.UpsertPending(cronDB, "agent-a", "chat-1", "1710000005000", "second insert", time.Now().Add(time.Millisecond)); err != nil {
		t.Fatalf("insert second pending: %v", err)
	}
	if _, err := integrationmessageinsert.MarkPublished(cronDB, "chat-1", []string{"1710000000000", "1710000005000"}, time.Now().Add(2*time.Millisecond)); err != nil {
		t.Fatalf("MarkPublished: %v", err)
	}

	event := "data: {\"choices\":[{\"index\":0,\"metadata\":{\"__TID__\":\"1710000005000\",\"__PROCESS__\":\"rag_insert\"},\"delta\":{\"content\":\"[已加入引导内容] hello\",\"role\":\"assistant\"}}],\"workflow\":\"main\",\"object\":\"chat.completion\",\"model\":\"right\",\"created\":1783184455608,\"code\":200,\"biz\":\"main\",\"id\":\"15c859bd-8f0e-4e93-85b9-cecb1b3362d2\"}\n\n"
	markConfirmedMessageInsertUploads("chat-1", event)

	statuses, err := integrationmessageinsert.StatusByTIDs(cronDB, "chat-1", []string{"1710000000000", "1710000005000"})
	if err != nil {
		t.Fatalf("status query: %v", err)
	}
	if statuses["1710000000000"] != integrationmessageinsert.StatusPending {
		t.Fatalf("status first = %d, want %d", statuses["1710000000000"], integrationmessageinsert.StatusPending)
	}
	if statuses["1710000005000"] != integrationmessageinsert.StatusUploaded {
		t.Fatalf("status second = %d, want %d", statuses["1710000005000"], integrationmessageinsert.StatusUploaded)
	}
}

func TestMessageInsertHandlersLifecycle(t *testing.T) {
	oldCronDB := cronDB
	defer func() {
		if cronDB != nil && cronDB != oldCronDB {
			_ = cronDB.Close()
		}
		cronDB = oldCronDB
	}()

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	cronDB = db
	if err := integrationmessageinsert.EnsureSchema(cronDB); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}

	addReq := httptest.NewRequest(http.MethodPost, "/api/message_insert/add", strings.NewReader(`{"agentId":"agent-a","chatId":"chat-1","tid":1710000000000,"message":"hello"}`))
	addRec := httptest.NewRecorder()
	handleMessageInsertAdd().ServeHTTP(addRec, addReq)
	if addRec.Code != http.StatusOK {
		t.Fatalf("add status = %d body=%s", addRec.Code, addRec.Body.String())
	}

	var addResp struct {
		Status int `json:"status"`
		Data   struct {
			Tid string `json:"tid"`
		} `json:"data"`
	}
	if err := json.Unmarshal(addRec.Body.Bytes(), &addResp); err != nil {
		t.Fatalf("decode add response: %v", err)
	}
	if addResp.Status != 0 {
		t.Fatalf("add response status = %d", addResp.Status)
	}
	if addResp.Data.Tid != "1710000000000" {
		t.Fatalf("add tid = %q", addResp.Data.Tid)
	}

	delReq := httptest.NewRequest(http.MethodPost, "/api/message_insert/del", strings.NewReader(`{"chatId":"chat-1","tid":"1710000000000"}`))
	delRec := httptest.NewRecorder()
	handleMessageInsertDel().ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusOK {
		t.Fatalf("del status = %d body=%s", delRec.Code, delRec.Body.String())
	}

	statuses, err := integrationmessageinsert.StatusByTIDs(cronDB, "chat-1", []string{"1710000000000"})
	if err != nil {
		t.Fatalf("status query after del: %v", err)
	}
	if statuses["1710000000000"] != integrationmessageinsert.StatusCancelled {
		t.Fatalf("status after del = %d, want %d", statuses["1710000000000"], integrationmessageinsert.StatusCancelled)
	}
}

func TestMessageInsertDeleteHandlerRemovesActiveRows(t *testing.T) {
	oldCronDB := cronDB
	defer func() {
		if cronDB != nil && cronDB != oldCronDB {
			_ = cronDB.Close()
		}
		cronDB = oldCronDB
	}()

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	cronDB = db
	if err := integrationmessageinsert.EnsureSchema(cronDB); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	if _, err := integrationmessageinsert.UpsertPending(cronDB, "agent-a", "chat-1", "1710000000000", "first", time.Unix(1710000000, 0)); err != nil {
		t.Fatalf("UpsertPending first: %v", err)
	}
	if _, err := integrationmessageinsert.UpsertPending(cronDB, "agent-a", "chat-1", "1710000005000", "second", time.Unix(1710000010, 0)); err != nil {
		t.Fatalf("UpsertPending second: %v", err)
	}
	if _, err := integrationmessageinsert.MarkPublished(cronDB, "chat-1", []string{"1710000005000"}, time.Unix(1710000020, 0)); err != nil {
		t.Fatalf("MarkPublished: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/message_insert/delete", strings.NewReader(`{"chatId":"chat-1","tids":["1710000000000","1710000005000"]}`))
	rec := httptest.NewRecorder()
	handleMessageInsertDelete().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Status int `json:"status"`
		Data   struct {
			Affected int64    `json:"affected"`
			Tids     []string `json:"tids"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode delete response: %v", err)
	}
	if resp.Status != 0 {
		t.Fatalf("delete response status = %d", resp.Status)
	}
	if resp.Data.Affected != 2 {
		t.Fatalf("delete affected = %d, want 2", resp.Data.Affected)
	}
	if len(resp.Data.Tids) != 2 {
		t.Fatalf("delete tids = %#v", resp.Data.Tids)
	}

	statuses, err := integrationmessageinsert.StatusByTIDs(cronDB, "chat-1", []string{"1710000000000", "1710000005000"})
	if err != nil {
		t.Fatalf("status query after delete: %v", err)
	}
	if len(statuses) != 0 {
		t.Fatalf("statuses after delete = %#v, want empty", statuses)
	}

	items, err := integrationmessageinsert.ListActive(cronDB, "chat-1", 5)
	if err != nil {
		t.Fatalf("ListActive after delete: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("active items after delete = %#v, want empty", items)
	}
}

func TestMessageInsertListHandlerIncludesPublishedPendingItems(t *testing.T) {
	oldCronDB := cronDB
	defer func() {
		if cronDB != nil && cronDB != oldCronDB {
			_ = cronDB.Close()
		}
		cronDB = oldCronDB
	}()

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	cronDB = db
	if err := integrationmessageinsert.EnsureSchema(cronDB); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	if _, err := integrationmessageinsert.UpsertPending(cronDB, "agent-a", "chat-1", "1710000000000", "first", time.Unix(1710000000, 0)); err != nil {
		t.Fatalf("UpsertPending first: %v", err)
	}
	if _, err := integrationmessageinsert.UpsertPending(cronDB, "agent-a", "chat-1", "1710000005000", "second", time.Unix(1710000010, 0)); err != nil {
		t.Fatalf("UpsertPending second: %v", err)
	}
	if _, err := integrationmessageinsert.MarkPublished(cronDB, "chat-1", []string{"1710000000000"}, time.Unix(1710000020, 0)); err != nil {
		t.Fatalf("MarkPublished: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/message_insert/list?chatId=chat-1&limit=5", nil)
	rec := httptest.NewRecorder()
	handleMessageInsertList().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Status int `json:"status"`
		Data   []struct {
			Tid        string `json:"tid"`
			ReportedAt string `json:"reportedAt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if resp.Status != 0 {
		t.Fatalf("list response status = %d", resp.Status)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("list items = %d, want 2", len(resp.Data))
	}
	if resp.Data[0].Tid != "1710000000000" || resp.Data[1].Tid != "1710000005000" {
		t.Fatalf("list tids = %#v", resp.Data)
	}
	if strings.TrimSpace(resp.Data[0].ReportedAt) == "" {
		t.Fatalf("published item reportedAt is empty: %#v", resp.Data[0])
	}
}

func TestMessageInsertHandlersRejectMutationsAfterReported(t *testing.T) {
	oldCronDB := cronDB
	defer func() {
		if cronDB != nil && cronDB != oldCronDB {
			_ = cronDB.Close()
		}
		cronDB = oldCronDB
	}()

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	cronDB = db
	if err := integrationmessageinsert.EnsureSchema(cronDB); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	if _, err := integrationmessageinsert.UpsertPending(cronDB, "agent-a", "chat-1", "1710000000000", "hello", time.Unix(1710000000, 0)); err != nil {
		t.Fatalf("UpsertPending: %v", err)
	}
	if _, err := integrationmessageinsert.MarkPublished(cronDB, "chat-1", []string{"1710000000000"}, time.Unix(1710000010, 0)); err != nil {
		t.Fatalf("MarkPublished: %v", err)
	}

	addReq := httptest.NewRequest(http.MethodPost, "/api/message_insert/add", strings.NewReader(`{"agentId":"agent-a","chatId":"chat-1","tid":"1710000000000","message":"updated"}`))
	addRec := httptest.NewRecorder()
	handleMessageInsertAdd().ServeHTTP(addRec, addReq)
	if addRec.Code != http.StatusConflict {
		t.Fatalf("add after publish status = %d body=%s", addRec.Code, addRec.Body.String())
	}

	delReq := httptest.NewRequest(http.MethodPost, "/api/message_insert/del", strings.NewReader(`{"chatId":"chat-1","tid":"1710000000000"}`))
	delRec := httptest.NewRecorder()
	handleMessageInsertDel().ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusConflict {
		t.Fatalf("del after publish status = %d body=%s", delRec.Code, delRec.Body.String())
	}
}

func TestHeartbeatDebugLogsTimeoutAndTaskPayload(t *testing.T) {
	var logBuf bytes.Buffer
	originalWriter := log.Writer()
	originalFlags := log.Flags()
	log.SetOutput(&logBuf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(originalWriter)
		log.SetFlags(originalFlags)
	}()

	timeoutServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(120 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ResponsePayload{Code: 200})
	}))
	defer timeoutServer.Close()

	timeoutCfg := &Config{
		HTTPTimeout:       200,
		HTTPConnTimeout:   50,
		HTTPSocketTimeout: 50,
		HTTPDebug:         true,
		IdleTimeout:       1,
	}
	timeoutClient := createCliGetHTTPClient(timeoutCfg)
	defer closeHTTPClientIdleConnections(timeoutClient)

	if _, err := heartbeat(timeoutClient, timeoutServer.URL, &AgentOutput{}, timeoutCfg); err == nil {
		t.Fatal("expected timeout error")
	}
	timeoutLog := logBuf.String()
	if !strings.Contains(timeoutLog, "cli-get debug: /cli/get timeout") {
		t.Fatalf("timeout log missing: %s", timeoutLog)
	}
	if !strings.Contains(timeoutLog, "http_timeout=200") || !strings.Contains(timeoutLog, "http_socket_timeout=50") {
		t.Fatalf("timeout config missing: %s", timeoutLog)
	}
	if !strings.Contains(timeoutLog, "elapsed_ms=") || !strings.Contains(timeoutLog, "time=") {
		t.Fatalf("timeout timing missing: %s", timeoutLog)
	}

	logBuf.Reset()

	taskJSON := `{"timeout":5000,"suffix":"cmd","type":"cmd","tid":"task-debug","cmd":"echo hello","agentId":"a","chat":"chat-debug"}`
	taskServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ResponsePayload{
			Code: 200,
			Choices: []struct {
				Message struct {
					Role    string      `json:"role"`
					Content interface{} `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Role    string      `json:"role"`
					Content interface{} `json:"content"`
				}{Role: "assistant", Content: taskJSON}},
			},
		})
	}))
	defer taskServer.Close()

	taskClient := &http.Client{Timeout: time.Second}
	taskCfg := &Config{HTTPDebug: true}
	task, err := heartbeat(taskClient, taskServer.URL, &AgentOutput{}, taskCfg)
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if task == nil {
		t.Fatal("expected task")
	}
	taskLog := logBuf.String()
	if !strings.Contains(taskLog, "cli-get debug: /cli/get task") {
		t.Fatalf("task log missing: %s", taskLog)
	}
	if !strings.Contains(taskLog, "task-debug") || !strings.Contains(taskLog, taskJSON) {
		t.Fatalf("task payload missing: %s", taskLog)
	}
	if !strings.Contains(taskLog, "time=") {
		t.Fatalf("task time missing: %s", taskLog)
	}
}

func TestPublishResultDebugLogsStatusAndResult(t *testing.T) {
	var logBuf bytes.Buffer
	originalWriter := log.Writer()
	originalFlags := log.Flags()
	log.SetOutput(&logBuf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(originalWriter)
		log.SetFlags(originalFlags)
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ResponsePayload{Code: 200})
	}))
	defer server.Close()

	result := &ResultPayload{
		Status:  7,
		AgentId: "a",
		Chat:    "chat-debug",
		Suffix:  "cmd",
		Type:    "cmd",
		Cmd:     gzipBase64String("hello from debug log"),
		Tid:     "tid-debug",
	}
	cfg := &Config{HTTPDebug: true}
	if err := publishResult(&http.Client{Timeout: time.Second}, server.URL, result, &AgentOutput{}, cfg); err != nil {
		t.Fatalf("publish: %v", err)
	}
	text := logBuf.String()
	if !strings.Contains(text, "cli-get debug: /cli/pub result") {
		t.Fatalf("publish debug log missing: %s", text)
	}
	if !strings.Contains(text, "status=7") || !strings.Contains(text, "hello from debug log") {
		t.Fatalf("publish status/result missing: %s", text)
	}
	if !strings.Contains(text, "time=") {
		t.Fatalf("publish time missing: %s", text)
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
	db, err := sql.Open("sqlite", filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	cronDB = db
	if _, err := cronDB.Exec(`CREATE TABLE IF NOT EXISTS chat_log (id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL, chat_type TEXT NOT NULL DEFAULT 'page_session', role TEXT NOT NULL, response_type TEXT NOT NULL DEFAULT 'normal', content TEXT NOT NULL, created_at TEXT NOT NULL)`); err != nil {
		t.Fatalf("create chat_log: %v", err)
	}
	if _, err := cronDB.Exec(`CREATE TABLE IF NOT EXISTS agent_message_log (id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL DEFAULT '', chat_id TEXT NOT NULL DEFAULT '', content TEXT NOT NULL, log_type INTEGER NOT NULL, created_at TEXT NOT NULL)`); err != nil {
		t.Fatalf("create agent_message_log: %v", err)
	}
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	if _, err := cronDB.Exec(`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES
		('a','chat-r','{\"task\":\"run\"}',?, '2026-05-13T12:00:00.100'),
		('a','chat-r','{\"status\":0}',?, '2026-05-13T12:00:00.200')`,
		logTypeCLIGet, logTypeCLIPub); err != nil {
		t.Fatalf("seed event logs: %v", err)
	}

	handler := handleRestore()
	req := httptest.NewRequest(http.MethodPost, "/api/restore?agentId=a&chat=chat-r&timeline=2026-05-13T12:00:00", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp struct {
		Status int `json:"status"`
		Data   []struct {
			Role    string `json:"role"`
			LogType int    `json:"logType"`
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
	if resp.Data[0].Role != "cli/get" || resp.Data[0].LogType != logTypeCLIGet {
		t.Fatalf("first = %#v", resp.Data[0])
	}
	if resp.Data[1].Role != "cli/pub" || resp.Data[1].LogType != logTypeCLIPub {
		t.Fatalf("second = %#v", resp.Data[1])
	}
}

func TestHandleRestoreUsesChatAcrossAgentSwitches(t *testing.T) {
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
		t.Fatalf("open sqlite: %v", err)
	}
	cronDB = db
	if _, err := cronDB.Exec(`CREATE TABLE IF NOT EXISTS chat_log (id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL, chat_type TEXT NOT NULL DEFAULT 'page_session', role TEXT NOT NULL, response_type TEXT NOT NULL DEFAULT 'normal', content TEXT NOT NULL, created_at TEXT NOT NULL)`); err != nil {
		t.Fatalf("create chat_log: %v", err)
	}
	if _, err := cronDB.Exec(`CREATE TABLE IF NOT EXISTS agent_message_log (id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL DEFAULT '', chat_id TEXT NOT NULL DEFAULT '', content TEXT NOT NULL, log_type INTEGER NOT NULL, created_at TEXT NOT NULL)`); err != nil {
		t.Fatalf("create agent_message_log: %v", err)
	}
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	if _, err := cronDB.Exec(`INSERT INTO chat_log (agent_id, chat_id, chat_type, role, response_type, content, created_at) VALUES
		('agent-a','chat-switch','page_session','A','normal','from-agent-a','2026-05-13T12:00:00.100'),
		('agent-b','chat-switch','page_session','A','normal','from-agent-b','2026-05-13T12:00:00.300'),
		('agent-z','chat-other','page_session','A','normal','other-chat','2026-05-13T12:00:00.400')`); err != nil {
		t.Fatalf("seed chat_log: %v", err)
	}
	if _, err := cronDB.Exec(`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES
		('agent-a','chat-switch','{"task":"run-a"}',?, '2026-05-13T12:00:00.200'),
		('agent-b','chat-switch','{"status":0,"cmd":"run-b"}',?, '2026-05-13T12:00:00.400'),
		('agent-z','chat-other','{"task":"other"}',?, '2026-05-13T12:00:00.500')`,
		logTypeCLIGet, logTypeCLIPub, logTypeCLIGet); err != nil {
		t.Fatalf("seed event logs: %v", err)
	}

	handler := handleRestore()
	req := httptest.NewRequest(http.MethodPost, "/api/restore?chat=chat-switch&timeline=2026-05-13T12:00:00", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

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

func TestHandleRestoreHistoryReturnsLatestCompleteRound(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	db, err := sql.Open("sqlite", filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	cronDB = db
	if _, err := cronDB.Exec(`CREATE TABLE IF NOT EXISTS chat_log (id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL, chat_type TEXT NOT NULL DEFAULT 'page_session', role TEXT NOT NULL, response_type TEXT NOT NULL DEFAULT 'normal', content TEXT NOT NULL, created_at TEXT NOT NULL)`); err != nil {
		t.Fatalf("create chat_log: %v", err)
	}
	if _, err := cronDB.Exec(`CREATE TABLE IF NOT EXISTS agent_message_log (id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL DEFAULT '', chat_id TEXT NOT NULL DEFAULT '', content TEXT NOT NULL, log_type INTEGER NOT NULL, created_at TEXT NOT NULL)`); err != nil {
		t.Fatalf("create agent_message_log: %v", err)
	}
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	if _, err := cronDB.Exec(`INSERT INTO chat_log (agent_id, chat_id, chat_type, role, response_type, content, created_at) VALUES
		('a','chat-history','page_session','Q','normal','{"messages":[{"role":"user","content":"round-1"}]}','2026-05-13T10:00:00.000'),
		('a','chat-history','page_session','A','normal','data: {"choices":[{"delta":{"content":"old"}}]}' || char(10),'2026-05-13T10:00:01.000'),
		('a','chat-history','page_session','A','normal','data: [DONE]' || char(10),'2026-05-13T10:00:01.100'),
		('a','chat-history','page_session','Q','normal','{"messages":[{"role":"user","content":"round-2"}]}','2026-05-13T10:00:02.000'),
		('a','chat-history','page_session','A','normal','data: {"choices":[{"delta":{"content":"new"}}]}' || char(10),'2026-05-13T10:00:03.000'),
		('a','chat-history','page_session','A','normal','data: [DONE]' || char(10),'2026-05-13T10:00:04.000')`); err != nil {
		t.Fatalf("seed chat_log: %v", err)
	}
	if _, err := cronDB.Exec(`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES
		('a','chat-history','{"cmd":"echo second-round"}', ?, '2026-05-13T10:00:02.500'),
		('a','chat-history','{"status":0,"content":"ok"}', ?, '2026-05-13T10:00:03.500')`,
		logTypeCLIGet, logTypeCLIPub); err != nil {
		t.Fatalf("seed event logs: %v", err)
	}

	handler := handleRestore()
	req := httptest.NewRequest(http.MethodPost, "/api/restore?chat=chat-history&history=1&limit=2", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp struct {
		Status int `json:"status"`
		Data   []struct {
			Role      string `json:"role"`
			Content   string `json:"content"`
			CreatedAt string `json:"createdAt"`
		} `json:"data"`
		History struct {
			HasMore        bool   `json:"hasMore"`
			BeforeTimeline string `json:"beforeTimeline"`
			BeforeID       int    `json:"beforeId"`
		} `json:"history"`
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
	if resp.Data[0].Role != "Q" || !strings.Contains(resp.Data[0].Content, "round-2") {
		t.Fatalf("first record = %#v, want second round Q", resp.Data[0])
	}
	if resp.Data[1].Role != "cli/get" {
		t.Fatalf("second record role = %q, want cli/get", resp.Data[1].Role)
	}
	if resp.Data[2].Role != "A" || !strings.Contains(resp.Data[2].Content, `"new"`) {
		t.Fatalf("third record = %#v, want second round A", resp.Data[2])
	}
	if resp.Data[3].Role != "cli/pub" {
		t.Fatalf("fourth record role = %q, want cli/pub", resp.Data[3].Role)
	}
	if resp.Data[4].Role != "A" || resp.Data[4].Content != "data: [DONE]\n" {
		t.Fatalf("fifth record = %#v, want done marker", resp.Data[4])
	}
	if !resp.History.HasMore {
		t.Fatal("history.hasMore = false, want true")
	}
	if resp.History.BeforeTimeline != "2026-05-13T10:00:02.000" {
		t.Fatalf("history.beforeTimeline = %q, want second round Q timestamp", resp.History.BeforeTimeline)
	}
	if resp.History.BeforeID <= 0 {
		t.Fatalf("history.beforeId = %d, want positive", resp.History.BeforeID)
	}
}

func TestHandleRestoreHistoryBeforeCursorReturnsOlderRounds(t *testing.T) {
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
		t.Fatalf("open sqlite: %v", err)
	}
	cronDB = db
	if _, err := cronDB.Exec(`CREATE TABLE IF NOT EXISTS chat_log (id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL, chat_type TEXT NOT NULL DEFAULT 'page_session', role TEXT NOT NULL, response_type TEXT NOT NULL DEFAULT 'normal', content TEXT NOT NULL, created_at TEXT NOT NULL)`); err != nil {
		t.Fatalf("create chat_log: %v", err)
	}
	if _, err := cronDB.Exec(`CREATE TABLE IF NOT EXISTS agent_message_log (id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL DEFAULT '', chat_id TEXT NOT NULL DEFAULT '', content TEXT NOT NULL, log_type INTEGER NOT NULL, created_at TEXT NOT NULL)`); err != nil {
		t.Fatalf("create agent_message_log: %v", err)
	}
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	if _, err := cronDB.Exec(`INSERT INTO chat_log (agent_id, chat_id, chat_type, role, response_type, content, created_at) VALUES
		('a','chat-history-page','page_session','Q','normal','{"messages":[{"role":"user","content":"round-1"}]}','2026-05-13T10:00:00.000'),
		('a','chat-history-page','page_session','A','normal','data: {"choices":[{"delta":{"content":"one"}}]}' || char(10),'2026-05-13T10:00:01.000'),
		('a','chat-history-page','page_session','A','normal','data: [DONE]' || char(10),'2026-05-13T10:00:01.100'),
		('a','chat-history-page','page_session','Q','normal','{"messages":[{"role":"user","content":"round-2"}]}','2026-05-13T10:00:02.000'),
		('a','chat-history-page','page_session','A','normal','data: {"choices":[{"delta":{"content":"two"}}]}' || char(10),'2026-05-13T10:00:03.000'),
		('a','chat-history-page','page_session','A','normal','data: [DONE]' || char(10),'2026-05-13T10:00:03.100')`); err != nil {
		t.Fatalf("seed chat_log: %v", err)
	}

	handler := handleRestore()
	firstReq := httptest.NewRequest(http.MethodPost, "/api/restore?chat=chat-history-page&history=1&limit=2", nil)
	firstRec := httptest.NewRecorder()
	handler(firstRec, firstReq)

	var firstResp struct {
		Status  int `json:"status"`
		History struct {
			HasMore        bool   `json:"hasMore"`
			BeforeTimeline string `json:"beforeTimeline"`
			BeforeID       int    `json:"beforeId"`
		} `json:"history"`
	}
	if err := json.NewDecoder(firstRec.Body).Decode(&firstResp); err != nil {
		t.Fatalf("decode first: %v", err)
	}
	if firstResp.Status != 0 {
		t.Fatalf("first status body = %d, want 0", firstResp.Status)
	}
	if !firstResp.History.HasMore {
		t.Fatal("first history.hasMore = false, want true")
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/api/restore?chat=chat-history-page&history=1&limit=2&beforeTimeline="+url.QueryEscape(firstResp.History.BeforeTimeline)+"&beforeId="+strconv.Itoa(firstResp.History.BeforeID), nil)
	secondRec := httptest.NewRecorder()
	handler(secondRec, secondReq)

	if secondRec.Code != http.StatusOK {
		t.Fatalf("second status = %d, want 200", secondRec.Code)
	}

	var secondResp struct {
		Status int `json:"status"`
		Data   []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"data"`
		History struct {
			HasMore bool `json:"hasMore"`
		} `json:"history"`
	}
	if err := json.NewDecoder(secondRec.Body).Decode(&secondResp); err != nil {
		t.Fatalf("decode second: %v", err)
	}
	if secondResp.Status != 0 {
		t.Fatalf("second status body = %d, want 0", secondResp.Status)
	}
	if len(secondResp.Data) != 3 {
		t.Fatalf("len(second data) = %d, want 3", len(secondResp.Data))
	}
	if secondResp.Data[0].Role != "Q" || !strings.Contains(secondResp.Data[0].Content, "round-1") {
		t.Fatalf("first second-page record = %#v, want first round Q", secondResp.Data[0])
	}
	if secondResp.Data[1].Role != "A" || !strings.Contains(secondResp.Data[1].Content, `"one"`) {
		t.Fatalf("second second-page record = %#v, want first round A", secondResp.Data[1])
	}
	if secondResp.Data[2].Role != "A" || secondResp.Data[2].Content != "data: [DONE]\n" {
		t.Fatalf("third second-page record = %#v, want first round done marker", secondResp.Data[2])
	}
	if secondResp.History.HasMore {
		t.Fatal("second history.hasMore = true, want false")
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

	handler := handleCancel(&Config{})
	req := httptest.NewRequest(http.MethodPost, "/api/cancel?chat=chat-switch", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

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
	connMu.Lock()
	_, exists := connMap[connKey("chat-switch")]
	connMu.Unlock()
	if exists {
		t.Fatalf("connection should be removed after cancel")
	}
}

func TestHeartbeatStoresCLIGetTaskContentInUnifiedLog(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	initCronDB()
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	taskJSON := `{"timeout":5000,"suffix":"cmd","type":"cmd","tid":"task_001","cmd":"echo hello","agentId":"a","chat":"chat_001"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ResponsePayload{
			Code: 200,
			Choices: []struct {
				Message struct {
					Role    string      `json:"role"`
					Content interface{} `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Role    string      `json:"role"`
					Content interface{} `json:"content"`
				}{Role: "assistant", Content: taskJSON}},
			},
		})
	}))
	defer server.Close()

	meta := &AgentOutput{
		DeviceID: "test-device",
		Terminal: "/bin/zsh",
		Agents:   []Agent{{AgentID: "a", Workspace: "/tmp/a"}},
	}
	client := &http.Client{Timeout: 5 * time.Second}
	task, err := heartbeat(client, server.URL, meta, nil)
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if task == nil {
		t.Fatal("expected task")
	}

	var stored string
	if err := cronDB.QueryRow(`SELECT content FROM agent_message_log WHERE agent_id = ? AND chat_id = ? AND log_type = ? ORDER BY id DESC LIMIT 1`,
		"a", "chat_001", logTypeCLIGet).Scan(&stored); err != nil {
		t.Fatalf("select cli/get log: %v", err)
	}
	if stored != taskJSON {
		t.Fatalf("cli/get log content = %q, want %q", stored, taskJSON)
	}
}

func TestHandleLogRoundExportsMarkdownToAgentTmp(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	initCronDB()
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	agentRoot := filepath.Join(tmp, "agent")
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

	if _, err := cronDB.Exec(`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES
		('A','chat-z','req-1',?, '2026-05-13T10:00:00.000'),
		('A','chat-z','resp-a',?, '2026-05-13T10:00:01.000'),
		('A','chat-z','resp-b',?, '2026-05-13T10:00:02.000'),
		('A','chat-z','tool-get',?, '2026-05-13T10:00:03.000'),
		('A','chat-z','tool-pub',?, '2026-05-13T10:00:04.000')`,
		logTypeChatCompletionRequest, logTypeChatCompletionResponse, logTypeChatCompletionResponse, logTypeCLIGet, logTypeCLIPub); err != nil {
		t.Fatalf("seed event logs: %v", err)
	}

	cfg := &Config{AgentDir: agentRoot, Device: "dev-1", AgentCacheMs: 1000}
	req := httptest.NewRequest(http.MethodGet, "/log_round?agentId=A&chatId=chat-z&round=1", nil)
	rec := httptest.NewRecorder()
	handleLogRound(cfg)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp struct {
		Status int    `json:"status"`
		Path   string `json:"path"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != 0 {
		t.Fatalf("status body = %d, want 0", resp.Status)
	}
	data, err := os.ReadFile(resp.Path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "| 时间 | 类型 | 内容 |") {
		t.Fatalf("markdown header missing: %s", text)
	}
	if !strings.Contains(text, "工具请求（cli/get）") || !strings.Contains(text, "工具响应（cli/pub）") {
		t.Fatalf("tool rows missing: %s", text)
	}
	if !strings.Contains(text, "req-1") || !strings.Contains(text, "tool-get") || !strings.Contains(text, "tool-pub") {
		t.Fatalf("expected rows missing: %s", text)
	}
}

func TestHandleLogSkillExportsMarkdownToAgentTmp(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	initCronDB()
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()
	chatID := "chat-skill-" + filepath.Base(tmp)

	agentRoot := filepath.Join(tmp, "agent")
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

	if _, err := cronDB.Exec(`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES
		(?,?,'{"model":"deepseek","messages":[{"role":"user","content":"看下我的桌面"}],"stream":true,"metadata":{"agentId":"A","chat":"`+chatID+`","type":"page_session"}}',?, '2026-05-13T10:00:00.000'),
		(?,?,'data: {"choices":[{"delta":{"content":"正在加载长期记忆\n"}}]}\n\n',?, '2026-05-13T10:00:01.000'),
		(?,?,'data: {"choices":[{"delta":{"content":"查看桌面文件列表\n"}}]}\n\n',?, '2026-05-13T10:00:02.000'),
		(?,?,'{"cmd":"ls -la /Users/demo/Desktop","tid":"task_1"}',?, '2026-05-13T10:00:03.000'),
		(?,?,'{"messages":[{"role":"user","content":"桌面列表已返回"}]}',?, '2026-05-13T10:00:04.000')`,
		"A", chatID, logTypeChatCompletionRequest,
		"A", chatID, logTypeChatCompletionResponse,
		"A", chatID, logTypeChatCompletionResponse,
		"A", chatID, logTypeCLIGet,
		"A", chatID, logTypeCLIPub); err != nil {
		t.Fatalf("seed event logs: %v", err)
	}

	cfg := &Config{AgentDir: agentRoot, Device: "dev-1", AgentCacheMs: 1000}
	req := httptest.NewRequest(http.MethodGet, "/log_skill?agentId=stale-agent&chatId="+url.QueryEscape(chatID)+"&round=1", nil)
	rec := httptest.NewRecorder()
	handleLogSkill(cfg)(rec, req)

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
	if matched, _ := regexp.MatchString(`/A_`+regexp.QuoteMeta(chatID)+`_\d{14}\.log$`, filepath.ToSlash(resp.Path)); !matched {
		t.Fatalf("path = %q, want suffix /A_%s_YYYYMMDDHHMMSS.log", resp.Path, chatID)
	}
	data, err := os.ReadFile(resp.Path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "| 时间 | 类型 | 内容 |") {
		t.Fatalf("markdown header missing: %s", text)
	}
	if !strings.Contains(text, "工具请求（cli/get）") || !strings.Contains(text, "工具响应（cli/pub）") {
		t.Fatalf("tool rows missing: %s", text)
	}
	if strings.Contains(text, `"messages"`) || strings.Contains(text, `"choices"`) || strings.Contains(text, `data:`) {
		t.Fatalf("markdown should contain readable content instead of raw payloads: %s", text)
	}
	if !strings.Contains(text, "看下我的桌面") {
		t.Fatalf("request content missing: %s", text)
	}
	if !strings.Contains(text, "正在加载长期记忆<br>查看桌面文件列表") {
		t.Fatalf("merged SSE content missing: %s", text)
	}
	if !strings.Contains(text, "ls -la /Users/demo/Desktop") {
		t.Fatalf("cli/get cmd missing: %s", text)
	}
	if !strings.Contains(text, "桌面列表已返回") {
		t.Fatalf("cli/pub content missing: %s", text)
	}
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) != 6 {
		t.Fatalf("line count = %d, want 6, text=%s", len(lines), text)
	}
	if resp.SizeK <= 0 {
		t.Fatalf("sizeK = %v, want > 0", resp.SizeK)
	}
}

func TestFormatRoundLogContentByType(t *testing.T) {
	request := `{"messages":[{"role":"user","content":"看下我的桌面"}]}`
	if got := formatRoundLogContent(logTypeChatCompletionRequest, request); got != "看下我的桌面" {
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
	if got := formatRoundLogContent(logTypeChatCompletionResponse, response); got != "正在加载长期记忆\n正在读取知识库\n查看桌面文件列表\n" {
		t.Fatalf("response content = %q", got)
	}

	cliGetTask := `{"cmd":"ls -la /Users/demo/Desktop","tid":"t-1"}`
	if got := formatRoundLogContent(logTypeCLIGet, cliGetTask); got != "ls -la /Users/demo/Desktop" {
		t.Fatalf("cli/get task content = %q", got)
	}

	cliGetRequest := `{"messages":[{"role":"user","content":"{\"timeout\":5000,\"suffix\":\"cmd\",\"type\":\"cmd\",\"tid\":\"task_001\",\"cmd\":\"echo hello\"}"}]}`
	if got := formatRoundLogContent(logTypeCLIGet, cliGetRequest); got != "echo hello" {
		t.Fatalf("cli/get request content = %q", got)
	}

	cliGetRequestUnified := `{"message":"{\"timeout\":5000,\"suffix\":\"cmd\",\"type\":\"cmd\",\"tid\":\"task_001\",\"cmd\":\"echo hi unified\"}"}`
	if got := formatRoundLogContent(logTypeCLIGet, cliGetRequestUnified); got != "echo hi unified" {
		t.Fatalf("cli/get unified request content = %q", got)
	}

	cliPubRequest := `{"messages":[{"role":"user","content":"任务执行成功"}]}`
	if got := formatRoundLogContent(logTypeCLIPub, cliPubRequest); got != "任务执行成功" {
		t.Fatalf("cli/pub request content = %q", got)
	}

	if got := formatRoundLogContent(logTypeCLIPub, "raw cli pub output"); got != "raw cli pub output" {
		t.Fatalf("cli/pub raw content = %q", got)
	}
}

func TestHandleLogSkillStatusReturnsCLIPubState(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)
	chatID := "chat-1-" + filepath.Base(tmp)

	initCronDB()
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	if _, err := cronDB.Exec(`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES
		(?, ?, 'req', ?, '2026-05-13T10:00:00.000'),
		(?, ?, 'resp', ?, '2026-05-13T10:00:01.000'),
		(?, ?, 'pub', ?, '2026-05-13T10:00:02.000')`,
		"A", chatID, logTypeChatCompletionRequest,
		"A", chatID, logTypeChatCompletionResponse,
		"A", chatID, logTypeCLIPub); err != nil {
		t.Fatalf("seed event logs: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/log_skill_status?agentId=A&chatId="+url.QueryEscape(chatID), nil)
	rec := httptest.NewRecorder()
	handleLogSkillStatus(nil)(rec, req)

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
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)
	chatID := "chat-get-" + filepath.Base(tmp)

	initCronDB()
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	if _, err := cronDB.Exec(`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES
		(?, ?, 'req', ?, '2026-05-13T10:00:00.000'),
		(?, ?, 'resp', ?, '2026-05-13T10:00:00.500'),
		(?, ?, 'cli-get-only', ?, '2026-05-13T10:00:01.000')`,
		"A", chatID, logTypeChatCompletionRequest,
		"A", chatID, logTypeChatCompletionResponse,
		"A", chatID, logTypeCLIGet); err != nil {
		t.Fatalf("seed event logs: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/log_skill_status?agentId=A&chatId="+url.QueryEscape(chatID), nil)
	rec := httptest.NewRecorder()
	handleLogSkillStatus(nil)(rec, req)
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

func TestHandleLogSkillStatusAcceptsChatIDWithoutAgentID(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)
	chatID := "chat-only-" + filepath.Base(tmp)

	initCronDB()
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	if _, err := cronDB.Exec(`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES
		(?, ?, 'req', ?, '2026-05-13T10:00:00.000'),
		(?, ?, 'resp', ?, '2026-05-13T10:00:01.000')`,
		"A", chatID, logTypeChatCompletionRequest,
		"A", chatID, logTypeChatCompletionResponse); err != nil {
		t.Fatalf("seed request/response logs: %v", err)
	}
	for i := 0; i < 10; i++ {
		cliAt := fmt.Sprintf("2026-05-13T10:00:%02d.000", i+2)
		if _, err := cronDB.Exec(
			`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES (?, ?, ?, ?, ?)`,
			"A", chatID, fmt.Sprintf("cli-get-%d", i+1), logTypeCLIGet, cliAt,
		); err != nil {
			t.Fatalf("insert cli/get %d: %v", i+1, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/log_skill_status?chatId="+url.QueryEscape(chatID), nil)
	rec := httptest.NewRecorder()
	handleLogSkillStatus(nil)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp struct {
		Status   int    `json:"status"`
		AgentID  string `json:"agentId"`
		ChatID   string `json:"chatId"`
		HasSkill bool   `json:"hasSkill"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != 0 {
		t.Fatalf("status body = %d, want 0", resp.Status)
	}
	if resp.AgentID != "" {
		t.Fatalf("agentId = %q, want empty", resp.AgentID)
	}
	if resp.ChatID != chatID {
		t.Fatalf("chatId = %q, want %q", resp.ChatID, chatID)
	}
	if !resp.HasSkill {
		t.Fatalf("hasSkill = false, want true when chatId-only lookup reaches threshold")
	}
}

func TestHandleLogSkillStatusUsesConfiguredSkillExtractRoundWhenRoundMissing(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)
	chatID := "chat-threshold-" + filepath.Base(tmp)
	db, err := sql.Open("sqlite", filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	cronDB = db
	if _, err := cronDB.Exec(`CREATE TABLE IF NOT EXISTS agent_message_log (id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL DEFAULT '', chat_id TEXT NOT NULL DEFAULT '', content TEXT NOT NULL, log_type INTEGER NOT NULL, created_at TEXT NOT NULL)`); err != nil {
		t.Fatalf("create agent_message_log: %v", err)
	}
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	if _, err := cronDB.Exec(`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES
		(?, ?, 'req-old', ?, '2026-05-13T09:59:00.000'),
		(?, ?, 'resp-old', ?, '2026-05-13T09:59:01.000'),
		(?, ?, 'req-latest', ?, '2026-05-13T10:00:00.000'),
		(?, ?, 'resp-latest', ?, '2026-05-13T10:00:01.000')`,
		"A", chatID, logTypeChatCompletionRequest,
		"A", chatID, logTypeChatCompletionResponse,
		"A", chatID, logTypeChatCompletionRequest,
		"A", chatID, logTypeChatCompletionResponse); err != nil {
		t.Fatalf("seed request/response logs: %v", err)
	}
	for i := 0; i < 20; i++ {
		cliAt := fmt.Sprintf("2026-05-13T09:59:%02d.000", i+2)
		if _, err := cronDB.Exec(
			`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES (?, ?, ?, ?, ?)`,
			"A", chatID, fmt.Sprintf("cli-get-old-%d", i+1), logTypeCLIGet, cliAt,
		); err != nil {
			t.Fatalf("insert old cli/get %d: %v", i+1, err)
		}
	}
	for i := 0; i < 10; i++ {
		cliAt := fmt.Sprintf("2026-05-13T10:00:%02d.000", i+2)
		if _, err := cronDB.Exec(
			`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES (?, ?, ?, ?, ?)`,
			"A", chatID, fmt.Sprintf("cli-get-latest-%d", i+1), logTypeCLIGet, cliAt,
		); err != nil {
			t.Fatalf("insert latest cli/get %d: %v", i+1, err)
		}
	}

	assertHasSkill := func(name, rawURL string, cfg *Config, want bool, wantRound int) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, rawURL, nil)
		rec := httptest.NewRecorder()
		handleLogSkillStatus(cfg)(rec, req)
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

	assertHasSkill("default threshold", "/log_skill_status?chatId="+url.QueryEscape(chatID), nil, true, 10)
	assertHasSkill("configured threshold", "/log_skill_status?chatId="+url.QueryEscape(chatID), &Config{SkillExtractRound: 11}, false, 11)
	assertHasSkill("explicit threshold override", "/log_skill_status?chatId="+url.QueryEscape(chatID)+"&round=9", &Config{SkillExtractRound: 11}, true, 9)
}

// ── Proxy iteration: /api/agentId ──

func TestApiAgentId(t *testing.T) {
	cfg := &Config{
		AgentDir:     "../agent/test-case",
		Device:       "test-dev",
		AgentCacheMs: 120000,
	}

	server := httptest.NewServer(http.HandlerFunc(handleAgentIDs(cfg)))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/agentId")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	var ids []string
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		t.Fatalf("decode: %v", err)
	}

	hasA, hasB := false, false
	for _, id := range ids {
		if id == "a" {
			hasA = true
		}
		if id == "b" {
			hasB = true
		}
	}
	if !hasA || !hasB {
		t.Errorf("agent IDs = %v, want to contain a and b", ids)
	}
}

func TestApiAgentDeleteSucceedsWhenDirectoryAlreadyMissing(t *testing.T) {
	flushAgentCache()

	root := t.TempDir()
	agentDir := filepath.Join(root, "alpha")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cfg := &Config{
		AgentDir:     root,
		Device:       "test-dev",
		AgentCacheMs: 120000,
	}
	if _, err := getAgentOutput(cfg.AgentDir, cfg.Device, time.Duration(cfg.AgentCacheMs)*time.Millisecond); err != nil {
		t.Fatalf("warm agent cache: %v", err)
	}
	if err := os.RemoveAll(agentDir); err != nil {
		t.Fatalf("remove agent dir: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agent/delete?name=alpha", nil)
	handleAgentDelete(cfg)(rec, req)
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

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/agentId", nil)
	handleAgentIDs(cfg)(listRec, listReq)
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

func TestApiSwarmAgent(t *testing.T) {
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

	cfg := &Config{
		AgentDir:     root,
		Device:       "test-dev",
		AgentCacheMs: 120000,
	}

	server := httptest.NewServer(http.HandlerFunc(handleSwarmAgents(cfg)))
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

func TestApiSwarmAgentExcludesCurrentAgent(t *testing.T) {
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

	cfg := &Config{
		AgentDir:     root,
		Device:       "test-dev",
		AgentCacheMs: 120000,
	}

	server := httptest.NewServer(http.HandlerFunc(handleSwarmAgents(cfg)))
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

func TestApiSwarmAgentMethodNotAllowed(t *testing.T) {
	cfg := &Config{
		AgentDir:     t.TempDir(),
		Device:       "test-dev",
		AgentCacheMs: 120000,
	}

	server := httptest.NewServer(http.HandlerFunc(handleSwarmAgents(cfg)))
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

	if cronDB != nil {
		_ = cronDB.Close()
		cronDB = nil
	}
	flushAgentCache()
	configDir := filepath.Join(tmp, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	startupConfigPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(startupConfigPath, []byte("{\n  \"app\": \""+filepath.Join(tmp, "integration")+"\"\n}\n"), 0o644); err != nil {
		t.Fatalf("write config/config.json: %v", err)
	}

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

	initCronDB()
	if cronDB == nil {
		t.Fatal("cronDB should not be nil")
	}
	if err := writeTokenStore(cronDB, map[string]string{"openai": "sk-test"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	cfg := &Config{AgentDir: agentRoot, Device: "dev"}
	handler := handleCron(cfg)
	rawTime := time.Now().Add(2 * time.Minute).Format("2006-01-02 15:04")
	req := httptest.NewRequest(http.MethodPost, "/api/cron/create?agentId=alpha",
		strings.NewReader(fmt.Sprintf(`{"content":"我之前问了什么","model":"openai","thinking":false,"router_disable":true,"rawTime":"%s","cycle":0,"token":"sk-test","chatId":"chat-reuse-1"}`, rawTime)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler(rec, req)

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
	if err := cronDB.QueryRow(`SELECT chat_id FROM task_meta WHERE id = ?`, resp.ID).Scan(&metaChatID); err != nil {
		t.Fatalf("query task_meta: %v", err)
	}
	if err := cronDB.QueryRow(`SELECT chat_id FROM task_detail WHERE meta_id = ?`, resp.ID).Scan(&detailChatID); err != nil {
		t.Fatalf("query task_detail: %v", err)
	}
	if err := cronDB.QueryRow(`SELECT router_disable FROM task_meta WHERE id = ?`, resp.ID).Scan(&metaRouterDisable); err != nil {
		t.Fatalf("query task_meta router_disable: %v", err)
	}
	if err := cronDB.QueryRow(`SELECT router_disable FROM task_detail WHERE meta_id = ?`, resp.ID).Scan(&detailRouterDisable); err != nil {
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
	handleCronMetadata()(metaRec, metaReq)
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

	db, err := sql.Open("sqlite", "data")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

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

func TestInitCronDBCreatesChatScopedAgentMessageLogIndex(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	initCronDB()
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	var count int
	if err := cronDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_agent_message_log_chat_type_time'`).Scan(&count); err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	if count != 1 {
		t.Fatalf("idx_agent_message_log_chat_type_time count = %d, want 1", count)
	}
}

func TestDeleteCronMetasPreservesCompletedDetails(t *testing.T) {
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

	renameCronLogTables(db)
	db.Exec(`CREATE TABLE IF NOT EXISTS task_meta (id INTEGER PRIMARY KEY AUTOINCREMENT, cycle INTEGER NOT NULL, raw_time TEXT NOT NULL DEFAULT '', agent_id TEXT NOT NULL, model TEXT NOT NULL, thinking INTEGER NOT NULL DEFAULT 0, cron TEXT NOT NULL, content TEXT NOT NULL, chat_id TEXT NOT NULL DEFAULT '', task_type TEXT NOT NULL DEFAULT 'cron')`)
	db.Exec(`CREATE TABLE IF NOT EXISTS task_detail (id INTEGER PRIMARY KEY AUTOINCREMENT, meta_id INTEGER NOT NULL, exec_time INTEGER NOT NULL, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL DEFAULT '', task_type TEXT NOT NULL DEFAULT 'cron', model TEXT NOT NULL, thinking INTEGER NOT NULL DEFAULT 0, content TEXT NOT NULL, result_content TEXT NOT NULL DEFAULT '', replied_at TEXT NOT NULL DEFAULT '', started INTEGER NOT NULL DEFAULT 0)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS cron_meta_log (id INTEGER PRIMARY KEY AUTOINCREMENT, meta_id INTEGER NOT NULL, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL DEFAULT '', task_type TEXT NOT NULL DEFAULT 'cron', action TEXT NOT NULL, cycle INTEGER NOT NULL, raw_time TEXT NOT NULL DEFAULT '', model TEXT NOT NULL, thinking INTEGER NOT NULL DEFAULT 0, cron TEXT NOT NULL DEFAULT '', content TEXT NOT NULL, created_at TEXT NOT NULL)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS cron_detail_log (id INTEGER PRIMARY KEY AUTOINCREMENT, detail_id INTEGER NOT NULL, meta_id INTEGER NOT NULL, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL DEFAULT '', task_type TEXT NOT NULL DEFAULT 'cron', action TEXT NOT NULL, exec_time INTEGER NOT NULL, model TEXT NOT NULL, thinking INTEGER NOT NULL DEFAULT 0, content TEXT NOT NULL, started INTEGER NOT NULL DEFAULT 0, occurred_at TEXT NOT NULL)`)

	res, err := db.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, model, thinking, cron, content, chat_id, task_type) VALUES (0, '', 'alpha', 'openai', 0, '', 'memo', 'chat-1', 'cron')`)
	if err != nil {
		t.Fatalf("insert meta: %v", err)
	}
	metaID, _ := res.LastInsertId()
	if _, err := db.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, content, started) VALUES (?, ?, 'alpha', 'chat-1', 'cron', 'openai', 0, 'todo', 0), (?, ?, 'alpha', 'chat-1', 'cron', 'openai', 0, 'done', 3)`,
		metaID, time.Now().Unix(), metaID, time.Now().Add(time.Minute).Unix()); err != nil {
		t.Fatalf("insert detail: %v", err)
	}

	affected, _, err := deleteCronMetas(db, cronMetaFilter{}, fmt.Sprintf("%d", metaID))
	if err != nil {
		t.Fatalf("deleteCronMetas: %v", err)
	}
	if affected != 1 {
		t.Fatalf("affected = %d, want 1", affected)
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

func TestQueryCronDetailsIncludesCompletedDetailsWithoutMeta(t *testing.T) {
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

	renameCronLogTables(db)
	db.Exec(`CREATE TABLE IF NOT EXISTS task_meta (id INTEGER PRIMARY KEY AUTOINCREMENT, cycle INTEGER NOT NULL, raw_time TEXT NOT NULL DEFAULT '', agent_id TEXT NOT NULL, model TEXT NOT NULL, thinking INTEGER NOT NULL DEFAULT 0, cron TEXT NOT NULL, content TEXT NOT NULL, chat_id TEXT NOT NULL DEFAULT '', task_type TEXT NOT NULL DEFAULT 'cron')`)
	db.Exec(`CREATE TABLE IF NOT EXISTS task_detail (id INTEGER PRIMARY KEY AUTOINCREMENT, meta_id INTEGER NOT NULL, exec_time INTEGER NOT NULL, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL DEFAULT '', task_type TEXT NOT NULL DEFAULT 'cron', model TEXT NOT NULL, thinking INTEGER NOT NULL DEFAULT 0, content TEXT NOT NULL, result_content TEXT NOT NULL DEFAULT '', replied_at TEXT NOT NULL DEFAULT '', started INTEGER NOT NULL DEFAULT 0)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS cron_meta_log (id INTEGER PRIMARY KEY AUTOINCREMENT, meta_id INTEGER NOT NULL, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL DEFAULT '', task_type TEXT NOT NULL DEFAULT 'cron', action TEXT NOT NULL, cycle INTEGER NOT NULL, raw_time TEXT NOT NULL DEFAULT '', model TEXT NOT NULL, thinking INTEGER NOT NULL DEFAULT 0, cron TEXT NOT NULL DEFAULT '', content TEXT NOT NULL, created_at TEXT NOT NULL)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS cron_detail_log (id INTEGER PRIMARY KEY AUTOINCREMENT, detail_id INTEGER NOT NULL, meta_id INTEGER NOT NULL, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL DEFAULT '', task_type TEXT NOT NULL DEFAULT 'cron', action TEXT NOT NULL, exec_time INTEGER NOT NULL, model TEXT NOT NULL, thinking INTEGER NOT NULL DEFAULT 0, content TEXT NOT NULL, started INTEGER NOT NULL DEFAULT 0, occurred_at TEXT NOT NULL)`)

	res, err := db.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, model, thinking, cron, content, chat_id, task_type) VALUES (0, '', 'alpha', 'openai', 0, '', 'memo', 'chat-1', 'email')`)
	if err != nil {
		t.Fatalf("insert meta: %v", err)
	}
	metaID, _ := res.LastInsertId()
	execTime := time.Now().Add(-time.Minute).Unix()
	if _, err := db.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, content, result_content, replied_at, started) VALUES (?, ?, 'alpha', 'chat-1', 'email', 'openai', 0, 'done', 'done-result', '2026-06-14T10:00:00', 3)`,
		metaID, execTime); err != nil {
		t.Fatalf("insert detail: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM task_meta WHERE id = ?`, metaID); err != nil {
		t.Fatalf("delete meta: %v", err)
	}

	items, err := queryCronDetails(db, cronDetailFilter{AgentID: "alpha"})
	if err != nil {
		t.Fatalf("queryCronDetails: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].MetaID != int(metaID) {
		t.Fatalf("meta_id = %d, want %d", items[0].MetaID, metaID)
	}
	if items[0].Type != "email" {
		t.Fatalf("type = %q, want email", items[0].Type)
	}
	if items[0].Started != 3 {
		t.Fatalf("started = %d, want 3", items[0].Started)
	}
	if items[0].ResultContent != "done-result" {
		t.Fatalf("resultContent = %q, want done-result", items[0].ResultContent)
	}
	if items[0].RepliedAt != "2026-06-14T10:00:00" {
		t.Fatalf("repliedAt = %q, want 2026-06-14T10:00:00", items[0].RepliedAt)
	}
	if strings.TrimSpace(items[0].RawTime) == "" {
		t.Fatalf("rawTime should be populated for completed detail without meta")
	}
}

func TestCleanupDeletedAgentCronDataRemovesOnlyUnfinished(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	if cronDB != nil {
		_ = cronDB.Close()
		cronDB = nil
	}
	flushAgentCache()

	agentRoot := filepath.Join(tmp, "agents")
	if err := os.MkdirAll(filepath.Join(agentRoot, "beta"), 0o755); err != nil {
		t.Fatalf("mkdir beta: %v", err)
	}

	initCronDB()
	if cronDB == nil {
		t.Fatal("cronDB should not be nil")
	}

	res, err := cronDB.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, model, thinking, cron, content, chat_id, task_type) VALUES (0, '', 'alpha', 'openai', 0, '', 'alpha memo', 'chat-a', 'cron')`)
	if err != nil {
		t.Fatalf("insert meta alpha: %v", err)
	}
	alphaMetaID, _ := res.LastInsertId()
	if _, err := cronDB.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, content, started) VALUES (?, ?, 'alpha', 'chat-a', 'cron', 'openai', 0, 'todo', 0), (?, ?, 'alpha', 'chat-a', 'cron', 'openai', 0, 'done', 3)`,
		alphaMetaID, time.Now().Unix(), alphaMetaID, time.Now().Add(time.Minute).Unix()); err != nil {
		t.Fatalf("insert alpha detail: %v", err)
	}

	res, err = cronDB.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, model, thinking, cron, content, chat_id, task_type) VALUES (0, '', 'beta', 'openai', 0, '', 'beta memo', 'chat-b', 'cron')`)
	if err != nil {
		t.Fatalf("insert meta beta: %v", err)
	}
	betaMetaID, _ := res.LastInsertId()
	if _, err := cronDB.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, content, started) VALUES (?, ?, 'beta', 'chat-b', 'cron', 'openai', 0, 'todo', 0)`,
		betaMetaID, time.Now().Unix()); err != nil {
		t.Fatalf("insert beta detail: %v", err)
	}

	cfg := &Config{AgentDir: agentRoot, Device: "dev", AgentCacheMs: 120000}
	cleanupDeletedAgentCronData(cfg)

	var alphaMetaCount, alphaUnfinished, alphaCompleted, betaMetaCount int
	if err := cronDB.QueryRow(`SELECT COUNT(*) FROM task_meta WHERE agent_id = 'alpha'`).Scan(&alphaMetaCount); err != nil {
		t.Fatalf("count alpha meta: %v", err)
	}
	if err := cronDB.QueryRow(`SELECT COUNT(*) FROM task_detail WHERE agent_id = 'alpha' AND started != 3`).Scan(&alphaUnfinished); err != nil {
		t.Fatalf("count alpha unfinished: %v", err)
	}
	if err := cronDB.QueryRow(`SELECT COUNT(*) FROM task_detail WHERE agent_id = 'alpha' AND started = 3`).Scan(&alphaCompleted); err != nil {
		t.Fatalf("count alpha completed: %v", err)
	}
	if err := cronDB.QueryRow(`SELECT COUNT(*) FROM task_meta WHERE agent_id = 'beta'`).Scan(&betaMetaCount); err != nil {
		t.Fatalf("count beta meta: %v", err)
	}
	if alphaMetaCount != 0 || alphaUnfinished != 0 || alphaCompleted != 1 || betaMetaCount != 1 {
		t.Fatalf("alpha meta=%d unfinished=%d completed=%d beta meta=%d, want 0/0/1/1", alphaMetaCount, alphaUnfinished, alphaCompleted, betaMetaCount)
	}
}

func TestCleanupInvalidCronDataRemovesUnconfiguredModelOnlyUnfinished(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	if cronDB != nil {
		_ = cronDB.Close()
		cronDB = nil
	}
	flushAgentCache()

	agentRoot := filepath.Join(tmp, "agents")
	if err := os.MkdirAll(filepath.Join(agentRoot, "alpha"), 0o755); err != nil {
		t.Fatalf("mkdir alpha: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(agentRoot, "beta"), 0o755); err != nil {
		t.Fatalf("mkdir beta: %v", err)
	}

	initCronDB()
	if cronDB == nil {
		t.Fatal("cronDB should not be nil")
	}
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	if err := writeTokenStore(cronDB, map[string]string{"openai": "sk-test"}); err != nil {
		t.Fatalf("write token store: %v", err)
	}

	res, err := cronDB.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, model, thinking, cron, content, chat_id, task_type) VALUES (0, '', 'alpha', 'missing-model', 0, '', 'alpha memo', 'chat-a', 'cron')`)
	if err != nil {
		t.Fatalf("insert alpha meta: %v", err)
	}
	alphaMetaID, _ := res.LastInsertId()
	if _, err := cronDB.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, content, started) VALUES (?, ?, 'alpha', 'chat-a', 'cron', 'missing-model', 0, 'todo', 0), (?, ?, 'alpha', 'chat-a', 'cron', 'missing-model', 0, 'done', 3)`,
		alphaMetaID, time.Now().Unix(), alphaMetaID, time.Now().Add(time.Minute).Unix()); err != nil {
		t.Fatalf("insert alpha detail: %v", err)
	}

	res, err = cronDB.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, model, thinking, cron, content, chat_id, task_type) VALUES (0, '', 'beta', 'openai', 0, '', 'beta memo', 'chat-b', 'cron')`)
	if err != nil {
		t.Fatalf("insert beta meta: %v", err)
	}
	betaMetaID, _ := res.LastInsertId()
	if _, err := cronDB.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, content, started) VALUES (?, ?, 'beta', 'chat-b', 'cron', 'openai', 0, 'todo', 0)`,
		betaMetaID, time.Now().Unix()); err != nil {
		t.Fatalf("insert beta detail: %v", err)
	}

	cfg := &Config{AgentDir: agentRoot, Device: "dev", AgentCacheMs: 120000}
	cleanupInvalidCronData(cfg)

	var alphaMetaCount, alphaUnfinished, alphaCompleted, betaMetaCount int
	if err := cronDB.QueryRow(`SELECT COUNT(*) FROM task_meta WHERE agent_id = 'alpha'`).Scan(&alphaMetaCount); err != nil {
		t.Fatalf("count alpha meta: %v", err)
	}
	if err := cronDB.QueryRow(`SELECT COUNT(*) FROM task_detail WHERE agent_id = 'alpha' AND started != 3`).Scan(&alphaUnfinished); err != nil {
		t.Fatalf("count alpha unfinished: %v", err)
	}
	if err := cronDB.QueryRow(`SELECT COUNT(*) FROM task_detail WHERE agent_id = 'alpha' AND started = 3`).Scan(&alphaCompleted); err != nil {
		t.Fatalf("count alpha completed: %v", err)
	}
	if err := cronDB.QueryRow(`SELECT COUNT(*) FROM task_meta WHERE agent_id = 'beta'`).Scan(&betaMetaCount); err != nil {
		t.Fatalf("count beta meta: %v", err)
	}
	if alphaMetaCount != 0 || alphaUnfinished != 0 || alphaCompleted != 1 || betaMetaCount != 1 {
		t.Fatalf("alpha meta=%d unfinished=%d completed=%d beta meta=%d, want 0/0/1/1", alphaMetaCount, alphaUnfinished, alphaCompleted, betaMetaCount)
	}
}

func TestCronExecuteOnceSkipsTaskWithoutConfiguredModel(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	if cronDB != nil {
		_ = cronDB.Close()
		cronDB = nil
	}
	flushAgentCache()

	agentRoot := filepath.Join(tmp, "agents")
	if err := os.MkdirAll(filepath.Join(agentRoot, "alpha"), 0o755); err != nil {
		t.Fatalf("mkdir alpha: %v", err)
	}

	initCronDB()
	if cronDB == nil {
		t.Fatal("cronDB should not be nil")
	}
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	res, err := cronDB.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, model, thinking, cron, content, chat_id, task_type) VALUES (0, '', 'alpha', 'missing-model', 0, '', 'alpha memo', 'chat-a', 'cron')`)
	if err != nil {
		t.Fatalf("insert meta: %v", err)
	}
	metaID, _ := res.LastInsertId()
	res, err = cronDB.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, content, started) VALUES (?, ?, 'alpha', 'chat-a', 'cron', 'missing-model', 0, 'todo', 0)`,
		metaID, time.Now().Add(-time.Minute).Unix())
	if err != nil {
		t.Fatalf("insert detail: %v", err)
	}
	detailID, _ := res.LastInsertId()

	cfg := &Config{AgentDir: agentRoot, Device: "dev", AgentCacheMs: 120000, Host: "http://127.0.0.1:1"}
	proxyClient := &http.Client{Timeout: 500 * time.Millisecond}
	connectSvc, err := newIntegrationConnectService(agentRoot)
	if err != nil {
		t.Fatalf("newIntegrationConnectService: %v", err)
	}
	defer connectSvc.Close()
	cronExecuteOnce(cfg, proxyClient, connectSvc)

	var started int
	if err := cronDB.QueryRow(`SELECT started FROM task_detail WHERE id = ?`, detailID).Scan(&started); err != nil {
		t.Fatalf("query detail status: %v", err)
	}
	if started != 0 {
		t.Fatalf("started = %d, want 0", started)
	}

	var skipped int
	if err := cronDB.QueryRow(`SELECT COUNT(*) FROM cron_detail_log WHERE detail_id = ? AND action = 'skip_invalid_model'`, detailID).Scan(&skipped); err != nil {
		t.Fatalf("query detail log: %v", err)
	}
	if skipped == 0 {
		t.Fatal("expected skip_invalid_model log")
	}
}

func TestIntegrationConnectHTTPFlow(t *testing.T) {
	tmp := t.TempDir()
	prevWD, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(prevWD)
	flushAgentCache()
	agentRoot := filepath.Join(tmp, "agents")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatal(err)
	}

	initCronDB()
	defer func() {
		if cronDB != nil {
			cronDB.Close()
			cronDB = nil
		}
	}()
	if err := writeTokenStore(cronDB, map[string]string{"OpenAI": "sk-test"}); err != nil {
		t.Fatal(err)
	}

	connectSvc, err := newIntegrationConnectService(agentRoot)
	if err != nil {
		t.Fatal(err)
	}
	defer connectSvc.Close()

	mux := http.NewServeMux()
	connectHandler := connectsvc.NewDynamicHTTPHandler(func(_ *http.Request) (*connectsvc.Service, func(), error) {
		return connectSvc, nil, nil
	})
	mux.Handle("/api/connect/meta", connectHandler)
	mux.Handle("/api/connect/request", connectHandler)
	mux.Handle("/api/connect/response", connectHandler)
	server := httptest.NewServer(mux)
	defer server.Close()

	metaResp, err := http.Post(server.URL+"/api/connect/meta?key=feishu&meta=%7B%22token%22%3A%22abc%22%7D&stream=true&callback=.%2Ffeishu&agent=A&model=OpenAI", "application/json", nil)
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
		t.Fatalf("unexpected meta payload: %+v", metaPayload)
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
	if reqPayload.Data.Status != connectsvc.RequestStatusPending {
		t.Fatalf("unexpected request status: %+v", reqPayload)
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

	requestListResp, err := http.Get(server.URL + "/api/connect/request?key=feishu&status=4")
	if err != nil {
		t.Fatal(err)
	}
	defer requestListResp.Body.Close()
	var requestListPayload struct {
		Status int                  `json:"status"`
		Data   []connectsvc.Request `json:"data"`
	}
	if err := json.NewDecoder(requestListResp.Body).Decode(&requestListPayload); err != nil {
		t.Fatal(err)
	}
	if requestListPayload.Status != 0 || len(requestListPayload.Data) != 1 {
		t.Fatalf("unexpected request list payload: %+v", requestListPayload)
	}
	if requestListPayload.Data[0].Status != connectsvc.RequestStatusReplied {
		t.Fatalf("unexpected replied request payload: %+v", requestListPayload.Data[0])
	}

	listResp, err := http.Get(server.URL + "/api/connect/response?key=feishu&requestId=" + fmt.Sprint(reqPayload.Data.ID))
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()
	var listPayload struct {
		Status int                   `json:"status"`
		Data   []connectsvc.Response `json:"data"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&listPayload); err != nil {
		t.Fatal(err)
	}
	if listPayload.Status != 0 || len(listPayload.Data) != 1 {
		t.Fatalf("unexpected list payload: %+v", listPayload)
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
	if configPayload.Data[0].Key != "feishu" || configPayload.Data[0].Name != "飞书" || fmt.Sprint(configPayload.Data[0].Meta["token"]) != "abc" {
		t.Fatalf("unexpected config data: %+v", configPayload.Data[0])
	}
}

func TestCronExecuteOnceInjectsMetaIDAndCronTypeIntoRequestMetadata(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	if cronDB != nil {
		_ = cronDB.Close()
		cronDB = nil
	}
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

	initCronDB()
	if cronDB == nil {
		t.Fatal("cronDB should not be nil")
	}
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	if err := ensureTokenStore(cronDB); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(cronDB, map[string]string{"OpenAI": "Bearer scheduled-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}
	if _, err := sandboxstate.SetWithDirectory(cronDB, "A", "chat-a", sandboxstate.ModeFilePickNet, "/Users/demo/Documents/feishu"); err != nil {
		t.Fatalf("set sandbox: %v", err)
	}

	metaRes, err := cronDB.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, chat_id, task_type, model, thinking, cron, content) VALUES (0, ?, 'A', 'chat-a', 'feishu', 'OpenAI', 1, '', '聚合消息')`,
		time.Now().Format("2006-01-02 15:04"))
	if err != nil {
		t.Fatalf("insert task_meta: %v", err)
	}
	metaID, _ := metaRes.LastInsertId()
	detailRes, err := cronDB.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, meta_ref, task_type, model, thinking, content, started) VALUES (?, ?, 'A', 'chat-a', '11,12', 'feishu', 'OpenAI', 1, '聚合消息', 0)`,
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

	cfg := &Config{
		Host:         upstream.URL,
		AgentDir:     agentRoot,
		Device:       "dev",
		AgentCacheMs: 120000,
	}
	proxyClient := &http.Client{Timeout: 5 * time.Second}
	connectSvc, err := newIntegrationConnectService(agentRoot)
	if err != nil {
		t.Fatalf("newIntegrationConnectService: %v", err)
	}
	defer connectSvc.Close()
	cronExecuteOnce(cfg, proxyClient, connectSvc)
	time.Sleep(300 * time.Millisecond)

	var started int
	if err := cronDB.QueryRow(`SELECT started FROM task_detail WHERE id = ?`, detailID).Scan(&started); err != nil {
		t.Fatalf("query started: %v", err)
	}
	if started != 3 {
		t.Fatalf("started = %d, want 3", started)
	}

	metadata, ok := captured["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata missing: %+v", captured)
	}
	msgs, ok := captured["messages"].([]interface{})
	if !ok || len(msgs) != 1 {
		t.Fatalf("messages = %#v, want one forwarded message", captured["messages"])
	}
	msg := msgs[0].(map[string]interface{})
	if msg["role"] != "user" || msg["content"] != "聚合消息" {
		t.Fatalf("forwarded message = %#v, want user/聚合消息", msg)
	}
	if _, exists := captured["message"]; exists {
		t.Fatalf("message should be absent: %v", captured["message"])
	}
	if captured["stream"] != true {
		t.Fatalf("stream = %v, want true", captured["stream"])
	}
	if metadata["type"] != chatTypeScheduledTask {
		t.Fatalf("metadata type = %v, want %s", metadata["type"], chatTypeScheduledTask)
	}
	if metadata["cron_type"] != "feishu" {
		t.Fatalf("metadata cron_type = %v, want feishu", metadata["cron_type"])
	}
	if _, exists := metadata["agent"]; exists {
		t.Fatalf("metadata.agent should be absent: %+v", metadata["agent"])
	}
	agentMeta := findAgentMetadataInList(t, metadata, "A")
	if agentMeta["version"] != "cron-v1" {
		t.Fatalf("metadata.agents[A].version = %v, want cron-v1", agentMeta["version"])
	}
	if agentMeta["sandbox"] != sandboxstate.ModeFilePickNet {
		t.Fatalf("metadata.agents[A].sandbox = %v, want %q", agentMeta["sandbox"], sandboxstate.ModeFilePickNet)
	}
	if metadata["sandbox_path"] != "/Users/demo/Documents/feishu" {
		t.Fatalf("metadata sandbox_path = %v, want /Users/demo/Documents/feishu", metadata["sandbox_path"])
	}
	if metadata["META_ID"] != "12" {
		t.Fatalf("metadata META_ID = %v, want 12", metadata["META_ID"])
	}
}

func TestCronExecuteOnceInjectsDefaultCronTypeIntoRequestMetadata(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	if cronDB != nil {
		_ = cronDB.Close()
		cronDB = nil
	}
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

	initCronDB()
	if cronDB == nil {
		t.Fatal("cronDB should not be nil")
	}
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	if err := ensureTokenStore(cronDB); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(cronDB, map[string]string{"OpenAI": "Bearer scheduled-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	metaRes, err := cronDB.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, chat_id, task_type, model, thinking, cron, content) VALUES (0, ?, 'A', 'chat-cron', '', 'OpenAI', 0, '', '普通备忘录')`,
		time.Now().Format("2006-01-02 15:04"))
	if err != nil {
		t.Fatalf("insert task_meta: %v", err)
	}
	metaID, _ := metaRes.LastInsertId()
	detailRes, err := cronDB.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, meta_ref, task_type, model, thinking, content, started) VALUES (?, ?, 'A', 'chat-cron', '', '', 'OpenAI', 0, '普通备忘录', 0)`,
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

	cfg := &Config{
		Host:         upstream.URL,
		AgentDir:     agentRoot,
		Device:       "dev",
		AgentCacheMs: 120000,
	}
	proxyClient := &http.Client{Timeout: 5 * time.Second}
	connectSvc, err := newIntegrationConnectService(agentRoot)
	if err != nil {
		t.Fatalf("newIntegrationConnectService: %v", err)
	}
	defer connectSvc.Close()
	cronExecuteOnce(cfg, proxyClient, connectSvc)
	time.Sleep(300 * time.Millisecond)

	var started int
	if err := cronDB.QueryRow(`SELECT started FROM task_detail WHERE id = ?`, detailID).Scan(&started); err != nil {
		t.Fatalf("query started: %v", err)
	}
	if started != 3 {
		t.Fatalf("started = %d, want 3", started)
	}

	metadata, ok := captured["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata missing: %+v", captured)
	}
	msgs, ok := captured["messages"].([]interface{})
	if !ok || len(msgs) != 1 {
		t.Fatalf("messages = %#v, want one forwarded message", captured["messages"])
	}
	msg := msgs[0].(map[string]interface{})
	if msg["role"] != "user" || msg["content"] != "普通备忘录" {
		t.Fatalf("forwarded message = %#v, want user/普通备忘录", msg)
	}
	if _, exists := captured["message"]; exists {
		t.Fatalf("message should be absent: %v", captured["message"])
	}
	if captured["stream"] != true {
		t.Fatalf("stream = %v, want true", captured["stream"])
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

	if cronDB != nil {
		_ = cronDB.Close()
		cronDB = nil
	}
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

	initCronDB()
	if cronDB == nil {
		t.Fatal("cronDB should not be nil")
	}
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	if err := ensureTokenStore(cronDB); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(cronDB, map[string]string{"OpenAI": "Bearer scheduled-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	schema := `{"type":"object","properties":{"summary":{"type":"string"}},"required":["summary"]}`
	metaRes, err := cronDB.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, chat_id, task_type, model, thinking, cron, content, response_schema) VALUES (0, ?, 'A', 'chat-schema', 'cron', 'OpenAI', 0, '', '结构化备忘录', ?)`,
		time.Now().Format("2006-01-02 15:04"), schema)
	if err != nil {
		t.Fatalf("insert task_meta: %v", err)
	}
	metaID, _ := metaRes.LastInsertId()
	if _, err := cronDB.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, meta_ref, task_type, model, thinking, content, response_schema, started) VALUES (?, ?, 'A', 'chat-schema', '', 'cron', 'OpenAI', 0, '结构化备忘录', ?, 0)`,
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

	cfg := &Config{
		Host:           upstream.URL,
		AgentDir:       agentRoot,
		Device:         "dev",
		AgentCacheMs:   60000,
		ConnectCacheMs: 1000,
	}
	connectSvc, err := newIntegrationConnectService(agentRoot)
	if err != nil {
		t.Fatalf("newIntegrationConnectService: %v", err)
	}
	defer connectSvc.Close()

	cronExecuteOnce(cfg, &http.Client{Timeout: 5 * time.Second}, connectSvc)
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
	if msg["role"] != "user" || msg["content"] != "结构化备忘录" {
		t.Fatalf("forwarded message = %#v, want user/结构化备忘录", msg)
	}
	if _, exists := captured["message"]; exists {
		t.Fatalf("message should be absent: %v", captured["message"])
	}
	if captured["stream"] != true {
		t.Fatalf("stream = %v, want true", captured["stream"])
	}
	if metadata["response_schema"] != schema {
		t.Fatalf("metadata response_schema = %v, want %s", metadata["response_schema"], schema)
	}

	responseFormat, ok := captured["response_format"].(map[string]interface{})
	if !ok {
		t.Fatalf("response_format missing: %+v", captured)
	}
	if responseFormat["type"] != "json_schema" {
		t.Fatalf("response_format.type = %v, want json_schema", responseFormat["type"])
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

	if cronDB != nil {
		_ = cronDB.Close()
		cronDB = nil
	}
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
	if err := os.WriteFile(filepath.Join(agentDir, "config.json"), []byte(`{"description":"demo","router_disable":true}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	initCronDB()
	if cronDB == nil {
		t.Fatal("cronDB should not be nil")
	}
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	if err := ensureTokenStore(cronDB); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(cronDB, map[string]string{"OpenAI": "Bearer scheduled-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	metaRes, err := cronDB.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, chat_id, task_type, model, thinking, router_disable, cron, content) VALUES (0, ?, 'A', 'chat-router', 'cron', 'OpenAI', 0, 1, '', '覆盖 Agent 蜂群配置')`,
		time.Now().Format("2006-01-02 15:04"))
	if err != nil {
		t.Fatalf("insert task_meta: %v", err)
	}
	metaID, _ := metaRes.LastInsertId()
	if _, err := cronDB.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, meta_ref, task_type, model, thinking, router_disable, content, started) VALUES (?, ?, 'A', 'chat-router', '', 'cron', 'OpenAI', 0, 0, '覆盖 Agent 蜂群配置', 0)`,
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

	cfg := &Config{
		Host:           upstream.URL,
		AgentDir:       agentRoot,
		Device:         "dev",
		AgentCacheMs:   60000,
		ConnectCacheMs: 1000,
	}
	proxyClient := &http.Client{Timeout: 5 * time.Second}
	connectSvc, err := newIntegrationConnectService(agentRoot)
	if err != nil {
		t.Fatalf("newIntegrationConnectService: %v", err)
	}
	defer connectSvc.Close()

	cronExecuteOnce(cfg, proxyClient, connectSvc)
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

	if cronDB != nil {
		_ = cronDB.Close()
		cronDB = nil
	}
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

	initCronDB()
	if cronDB == nil {
		t.Fatal("cronDB should not be nil")
	}
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	if err := ensureTokenStore(cronDB); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(cronDB, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}
	var storedToken string
	if err := cronDB.QueryRow(`SELECT token FROM token_store WHERE model = ?`, "openai").Scan(&storedToken); err != nil {
		t.Fatalf("query token_store: %v", err)
	}
	if storedToken != "Bearer test-token" {
		t.Fatalf("stored token = %q", storedToken)
	}

	cfg := &Config{
		AgentDir:       agentRoot,
		Device:         "dev",
		AgentCacheMs:   120000,
		ConnectCacheMs: 1000,
		Reply:          resolveServeReply(""),
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
	metaRes, err := cronDB.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, chat_id, task_type, model, thinking, cron, content) VALUES (0, ?, 'A', 'chat-feishu', 'feishu', 'OpenAI', 1, '', '需要回复的消息')`,
		now.Format("2006-01-02 15:04"))
	if err != nil {
		t.Fatalf("insert task_meta: %v", err)
	}
	metaID, _ := metaRes.LastInsertId()
	detailRes, err := cronDB.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, meta_ref, task_type, model, thinking, content, result_content, started) VALUES (?, ?, 'A', 'chat-feishu', ?, 'feishu', 'OpenAI', 1, '需要回复的消息', '任务已完成，下面是结果', 3)`,
		metaID, now.Add(-time.Minute).Unix(), fmt.Sprintf("%d,%d", earlier.ID, target.ID))
	if err != nil {
		t.Fatalf("insert task_detail: %v", err)
	}
	detailID, _ := detailRes.LastInsertId()

	var notifiedPlugin string
	var notifiedFlags map[string]string
	originalRunConnectListMeta := runConnectListMeta
	originalRunConnectPluginSend := runConnectPluginSend
	runConnectListMeta = func() ([]connectMetaConfigListItem, error) {
		return []connectMetaConfigListItem{{Key: "feishu", Name: meta.Name, Callback: filepath.Join(tmp, "plugins", "feishu")}}, nil
	}
	runConnectPluginSend = func(callbackPath string, flags map[string]string) error {
		notifiedPlugin = callbackPath
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

	notifyCompletedConnectTasks(cfg, svc)

	wantCallback := filepath.Join(tmp, "plugins", "feishu")
	if notifiedPlugin != wantCallback {
		t.Fatalf("notified callback = %q, want %q", notifiedPlugin, wantCallback)
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
	if err := cronDB.QueryRow(`SELECT replied_at FROM task_detail WHERE id = ?`, detailID).Scan(&repliedAt); err != nil {
		t.Fatalf("query replied_at: %v", err)
	}
	if strings.TrimSpace(repliedAt) == "" {
		t.Fatal("replied_at is empty")
	}
}

func TestNotifyCompletedConnectTasksSkipsWhenMetaRefTargetNotStarted(t *testing.T) {
	tmp := t.TempDir()
	restore := withIntegrationPluginSandbox(t, map[string][2]string{
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

	if existing := cronDB; existing != nil {
		existing.Close()
		cronDB = nil
	}
	defer func() {
		if cronDB != nil {
			cronDB.Close()
			cronDB = nil
		}
	}()
	initCronDB()
	if cronDB == nil {
		t.Fatal("cronDB should not be nil")
	}
	if err := ensureTokenStore(cronDB); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(cronDB, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	cfg := &Config{
		AgentDir:       agentRoot,
		Device:         "dev",
		AgentCacheMs:   120000,
		ConnectCacheMs: 1000,
		Reply:          resolveServeReply(""),
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
	metaRes, err := cronDB.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, chat_id, task_type, model, thinking, cron, content) VALUES (0, ?, 'A', 'chat-feishu', 'feishu', 'OpenAI', 1, '', '原始目标消息')`,
		now.Format("2006-01-02 15:04"))
	if err != nil {
		t.Fatalf("insert task_meta: %v", err)
	}
	metaID, _ := metaRes.LastInsertId()
	detailRes, err := cronDB.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, meta_ref, task_type, model, thinking, content, result_content, started) VALUES (?, ?, 'A', 'chat-feishu', ?, 'feishu', 'OpenAI', 1, '原始目标消息', '任务完成结果', 3)`,
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

	notifyCompletedConnectTasks(cfg, svc)

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
	if err := cronDB.QueryRow(`SELECT replied_at FROM task_detail WHERE id = ?`, detailID).Scan(&repliedAt); err != nil {
		t.Fatalf("query replied_at: %v", err)
	}
	if strings.TrimSpace(repliedAt) != "" {
		t.Fatalf("replied_at = %q, want empty", repliedAt)
	}
}

func TestRunConnectPluginSendNormalizesJSONMarkdownContent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, nil)
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

func TestPluginsMetaHandler(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)
	dbPath := filepath.Join(root, "data")
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "config.json"), []byte(fmt.Sprintf("{\"db\":%q}\n", dbPath)), 0o644); err != nil {
		t.Fatal(err)
	}
	agentDir := filepath.Join(root, "agent")

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
		"mail":   {`"邮件"`, pluginParamJSON("token")},
	})
	defer restore()
	writeIntegrationPluginScriptWithScope(t, filepath.Join("..", "plugins", "feishu"), `{"key":"feishu","name":"飞书"}`, pluginParamJSON("appId", "appSecret"), `["reuse","agent","swarm"]`)

	seedIntegrationPluginMetaTestData(t, dbPath, agentDir, connectsvc.MetaInput{
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

	cfg := &Config{ConnectCacheMs: 1000, AgentDir: agentDir}
	server := httptest.NewServer(http.HandlerFunc(handlePluginsMeta(cfg)))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/plugins/meta")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var payload struct {
		Status int `json:"status"`
		Data   []struct {
			Key           string                       `json:"key"`
			Name          string                       `json:"name"`
			Param         connectsvc.PluginParamFields `json:"param"`
			Scope         []string                     `json:"scope"`
			Meta          map[string]any               `json:"meta"`
			AgentID       string                       `json:"agentId"`
			ChatID        string                       `json:"chatId"`
			Model         string                       `json:"model"`
			Thinking      bool                         `json:"thinking"`
			Stream        bool                         `json:"stream"`
			RouterDisable bool                         `json:"router_disable"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Status != 0 || len(payload.Data) != 2 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.Data[0].Name != "飞书" {
		t.Fatalf("plugin name = %q, want 飞书", payload.Data[0].Name)
	}
	if payload.Data[0].Key != "feishu" || payload.Data[1].Key != "mail" {
		t.Fatalf("unexpected plugin keys: %+v", payload.Data)
	}
	if !reflect.DeepEqual(payload.Data[0].Param, pluginParamFields("appId", "appSecret")) {
		t.Fatalf("unexpected plugin params: %+v", payload.Data[0].Param)
	}
	if strings.Join(payload.Data[0].Scope, ",") != "reuse,agent,swarm" {
		t.Fatalf("unexpected feishu scope: %+v", payload.Data[0].Scope)
	}
	if strings.Join(payload.Data[1].Scope, ",") != "reuse,agent,provider,thinking,swarm" {
		t.Fatalf("unexpected mail scope: %+v", payload.Data[1].Scope)
	}
	if got := strings.TrimSpace(fmt.Sprint(payload.Data[0].Meta["appId"])); got == "" {
		t.Fatalf("unexpected plugin meta: %+v", payload.Data[0].Meta)
	}
	if payload.Data[0].AgentID != "A" || strings.TrimSpace(payload.Data[0].ChatID) == "" {
		t.Fatalf("unexpected plugin runtime fields: %+v", payload.Data[0])
	}
	if !payload.Data[0].Stream {
		t.Fatalf("unexpected plugin flags: %+v", payload.Data[0])
	}
	if payload.Data[1].Name != "邮件" || payload.Data[1].Meta == nil || len(payload.Data[1].Meta) != 0 {
		t.Fatalf("unexpected empty meta payload: %+v", payload.Data[1])
	}
}

func TestPluginsMetaHandlerMethodNotAllowed(t *testing.T) {
	cfg := &Config{ConnectCacheMs: 1000}
	server := httptest.NewServer(http.HandlerFunc(handlePluginsMeta(cfg)))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/plugins/meta", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}

func TestPluginsMetaHandlerPreservesEmptyScopeArray(t *testing.T) {
	pluginDir := t.TempDir()
	writeIntegrationPluginScriptWithScope(t, filepath.Join(pluginDir, "remote"), `{"key":"remote","name":"远程"}`, pluginParamJSON("exec_timeout"), `[]`)
	restorePluginDir := connectsvc.SetPluginDirResolverForTest(func() (string, error) {
		return pluginDir, nil
	})
	originalLocalPluginDirResolver := localPluginDirResolver
	localPluginDirResolver = func() (string, error) {
		return pluginDir, nil
	}
	defer restorePluginDir()
	defer func() {
		localPluginDirResolver = originalLocalPluginDirResolver
	}()

	cfg := &Config{ConnectCacheMs: 1000, AgentDir: "agent"}
	server := httptest.NewServer(http.HandlerFunc(handlePluginsMeta(cfg)))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/plugins/meta")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200, raw=%s", resp.StatusCode, string(raw))
	}
	if !strings.Contains(string(raw), `"scope":[]`) {
		t.Fatalf("scope should be preserved as empty array: %s", string(raw))
	}

	var payload struct {
		Status int `json:"status"`
		Data   []struct {
			Key   string   `json:"key"`
			Scope []string `json:"scope"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Status != 0 {
		t.Fatalf("unexpected payload: %s", string(raw))
	}
	found := false
	for _, item := range payload.Data {
		if item.Key != "remote" {
			continue
		}
		found = true
		if item.Scope == nil {
			t.Fatalf("remote scope = nil, want []")
		}
		if len(item.Scope) != 0 {
			t.Fatalf("remote scope = %+v, want []", item.Scope)
		}
	}
	if !found {
		t.Fatalf("remote plugin not found: %s", string(raw))
	}
}

func TestPluginsMetaHandlerAllowsIPv6Loopback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId")},
	})
	defer restore()

	cfg := &Config{ConnectCacheMs: 1000, AgentDir: "agent"}
	req := httptest.NewRequest(http.MethodGet, "/api/plugins/meta", nil)
	req.RemoteAddr = "[::1]:4567"
	rec := httptest.NewRecorder()

	handlePluginsMeta(cfg).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", rec.Code, rec.Body.String())
	}
}

func TestPluginsMetaHandlerAllowsRemoteReadWithoutSensitiveConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)

	dbPath := filepath.Join(root, "data")
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "config.json"), []byte(fmt.Sprintf("{\"db\":%q}\n", dbPath)), 0o644); err != nil {
		t.Fatal(err)
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
		"feishu": {`{"key":"feishu","name":"飞书"}`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()

	agentDir := filepath.Join(root, "agent")
	if err := os.MkdirAll(filepath.Join(agentDir, "A"), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	_ = db.Close()

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   dbPath,
		AgentDir: agentDir,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	if _, err := connectsvc.UpsertPluginConfigWithService(svc, connectsvc.MetaInput{
		Key:           "feishu",
		Name:          "飞书",
		Meta:          `{"appId":"cli-app","appSecret":"cli-secret"}`,
		Stream:        true,
		Callback:      "./feishu",
		AgentID:       "A",
		ChatID:        "chat-001",
		Model:         "OpenAI",
		Thinking:      true,
		RouterDisable: false,
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/meta", nil)
	req.RemoteAddr = "203.0.113.10:4567"
	rec := httptest.NewRecorder()

	handlePluginsMeta(&Config{ConnectCacheMs: 1000, AgentDir: agentDir}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Status int `json:"status"`
		Data   []struct {
			Key           string                       `json:"key"`
			Name          string                       `json:"name"`
			Param         connectsvc.PluginParamFields `json:"param"`
			Meta          map[string]any               `json:"meta"`
			Stream        bool                         `json:"stream"`
			Callback      string                       `json:"callback"`
			AgentID       string                       `json:"agentId"`
			ChatID        string                       `json:"chatId"`
			Model         string                       `json:"model"`
			Thinking      bool                         `json:"thinking"`
			RouterDisable bool                         `json:"router_disable"`
			CreatedAt     string                       `json:"createdAt"`
			UpdatedAt     string                       `json:"updatedAt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rec.Body.String())
	}
	if payload.Status != 0 || len(payload.Data) != 1 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	item := payload.Data[0]
	if item.Key != "feishu" || item.Name != "飞书" {
		t.Fatalf("unexpected plugin identity: %+v", item)
	}
	if len(item.Param) != 0 {
		t.Fatalf("remote param should be hidden: %+v", item.Param)
	}
	if len(item.Meta) != 0 {
		t.Fatalf("remote meta should be hidden: %+v", item.Meta)
	}
	if item.Stream || item.Callback != "" || item.AgentID != "" || item.ChatID != "" || item.Model != "" || item.Thinking || item.CreatedAt != "" || item.UpdatedAt != "" {
		t.Fatalf("remote runtime config should be hidden: %+v", item)
	}
	if !item.RouterDisable {
		t.Fatalf("remote router_disable should default to true: %+v", item)
	}
}

func TestPluginsMetaHandlerReloadsPluginsImmediately(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId")},
	})
	defer restore()

	cfg := &Config{ConnectCacheMs: 1000, AgentDir: "agent"}
	server := httptest.NewServer(http.HandlerFunc(handlePluginsMeta(cfg)))
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

	writeIntegrationPluginScript(t, filepath.Join("..", "plugins", "feishu"), `{"key":"feishu","name":"飞书新版"}`, pluginParamJSON("appId"))

	secondNames := readNames()
	if len(secondNames) != 1 || secondNames[0] != "飞书新版" {
		t.Fatalf("plugins meta should refresh immediately, got %+v", secondNames)
	}
}

func TestPluginsMetaHandlerSkipsInvalidEntriesAndLogs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
		"feishu": {`{"key":"feishu","name":"飞书"}`, pluginParamJSON("appId")},
	})
	defer restore()

	if err := os.WriteFile(filepath.Join("..", "plugins", "broken"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write broken plugin: %v", err)
	}
	if err := os.WriteFile(filepath.Join("..", "plugins", "User_Data.zip"), []byte("zip"), 0o644); err != nil {
		t.Fatalf("write zip file: %v", err)
	}

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

	cfg := &Config{ConnectCacheMs: 1000, AgentDir: "agent"}
	server := httptest.NewServer(http.HandlerFunc(handlePluginsMeta(cfg)))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/plugins/meta")
	if err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}
	if payload.Status != 0 || len(payload.Data) != 1 || payload.Data[0].Key != "feishu" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if !strings.Contains(logBuf.String(), "plugins meta skip broken") {
		t.Fatalf("expected skip log, got %q", logBuf.String())
	}
}

func TestPluginsStatusHandler(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
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

	cfg := &Config{}
	server := httptest.NewServer(http.HandlerFunc(handlePluginsStatus(cfg)))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/plugins/status?key=feishu")
	if err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}
	if payload.Status != 0 || !payload.Data.Started {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.Data.PID != cmd.Process.Pid {
		t.Fatalf("pid = %d, want %d", payload.Data.PID, cmd.Process.Pid)
	}
	if payload.Data.Key != "feishu" || payload.Data.Name != "飞书" {
		t.Fatalf("unexpected status data: %+v", payload.Data)
	}
}

func TestPluginsStatusHandlerMissingName(t *testing.T) {
	cfg := &Config{}
	server := httptest.NewServer(http.HandlerFunc(handlePluginsStatus(cfg)))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/plugins/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestPluginsStatusHandlerAllowsRemoteRead(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId")},
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

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/status?key=feishu", nil)
	req.RemoteAddr = "203.0.113.10:4567"
	rec := httptest.NewRecorder()

	handlePluginsStatus(&Config{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Status int `json:"status"`
		Data   struct {
			Key     string `json:"key"`
			Name    string `json:"name"`
			PID     int    `json:"pid"`
			Started bool   `json:"started"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rec.Body.String())
	}
	if payload.Status != 0 || !payload.Data.Started {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.Data.Key != "feishu" || payload.Data.Name != "飞书" || payload.Data.PID != cmd.Process.Pid {
		t.Fatalf("unexpected status payload: %+v", payload.Data)
	}
}

func TestPluginsConfigHandlerCreatesAndUpdatesMeta(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()
	if err := os.MkdirAll(filepath.Join("agent", "A"), 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := resolveIntegrationDBPath()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	cronDB = db
	defer func() {
		cronDB = nil
		_ = db.Close()
	}()
	if err := writeTokenStore(db, map[string]string{"openai": "Bearer test-token"}); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{ConnectCacheMs: 1000, AgentDir: "agent"}
	server := httptest.NewServer(http.HandlerFunc(handlePluginsConfig(cfg)))
	defer server.Close()

	createResp, err := http.Post(server.URL+"/api/plugins/config?key=feishu&meta=%7B%22appId%22%3A%22cli-app%22%2C%22appSecret%22%3A%22cli-secret%22%7D&agentId=A&model=openai", "application/json", nil)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(createResp.Body)
		t.Fatalf("create status = %d, body=%s", createResp.StatusCode, string(body))
	}

	var created struct {
		Status int `json:"status"`
		Data   struct {
			Name          string `json:"name"`
			AgentID       string `json:"agentId"`
			Model         string `json:"model"`
			Callback      string `json:"callback"`
			ChatID        string `json:"chatId"`
			Stream        bool   `json:"stream"`
			Thinking      bool   `json:"thinking"`
			RouterDisable bool   `json:"router_disable"`
		} `json:"data"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Status != 0 {
		t.Fatalf("create payload status = %d", created.Status)
	}
	if created.Data.Name != "feishu" || created.Data.AgentID != "A" || created.Data.Model != "openai" {
		t.Fatalf("unexpected create payload: %+v", created.Data)
	}
	if created.Data.Callback == "" || !filepath.IsAbs(created.Data.Callback) {
		t.Fatalf("callback = %q, want absolute plugin path", created.Data.Callback)
	}
	if created.Data.ChatID != "feishu" {
		t.Fatalf("chatId = %q, want feishu", created.Data.ChatID)
	}
	if created.Data.Stream || created.Data.Thinking || !created.Data.RouterDisable {
		t.Fatalf("unexpected defaults: stream=%v thinking=%v router_disable=%v", created.Data.Stream, created.Data.Thinking, created.Data.RouterDisable)
	}

	updateResp, err := http.Post(server.URL+"/api/plugins/config?key=feishu&meta=%7B%22appId%22%3A%22new-app%22%7D&stream=true&agentId=A&chatId=chat-fixed&model=openai&thinking=true&router_disable=false", "application/json", nil)
	if err != nil {
		t.Fatalf("update request failed: %v", err)
	}
	defer updateResp.Body.Close()

	if updateResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(updateResp.Body)
		t.Fatalf("update status = %d, body=%s", updateResp.StatusCode, string(body))
	}

	var updated struct {
		Status int `json:"status"`
		Data   struct {
			Meta          string `json:"meta"`
			Stream        bool   `json:"stream"`
			Thinking      bool   `json:"thinking"`
			RouterDisable bool   `json:"router_disable"`
			ChatID        string `json:"chatId"`
		} `json:"data"`
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
	if !updated.Data.Stream || !updated.Data.Thinking || updated.Data.RouterDisable {
		t.Fatalf("updated booleans = stream:%v thinking:%v router_disable:%v", updated.Data.Stream, updated.Data.Thinking, updated.Data.RouterDisable)
	}
	if updated.Data.ChatID != "chat-fixed" {
		t.Fatalf("chatId = %q, want chat-fixed", updated.Data.ChatID)
	}
}

func TestPluginsMetaReflectsNewConfigImmediatelyAfterDisplayNameSave(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)
	dbPath := filepath.Join(root, "data")
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "config.json"), []byte(fmt.Sprintf("{\"db\":%q}\n", dbPath)), 0o644); err != nil {
		t.Fatal(err)
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
		"email": {`{"key":"email","name":"邮件"}`, `[{"email":"","email_pop3":"","email_smtp":"","email_password":"","email_whitelist":""}]`},
	})
	defer restore()
	agentDir := filepath.Join(root, "agent")
	if err := os.MkdirAll(filepath.Join(agentDir, "A"), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeTokenStore(db, map[string]string{"openai": "Bearer test-token"}); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	cronDB = db
	defer func() {
		cronDB = nil
		_ = db.Close()
	}()
	if token, err := lookupTokenByModel(cronDB, "openai"); err != nil || strings.TrimSpace(token) == "" {
		t.Fatalf("seed openai token failed: token=%q err=%v", token, err)
	}

	cfg := &Config{ConnectCacheMs: 1000, AgentDir: agentDir}
	metaServer := httptest.NewServer(http.HandlerFunc(handlePluginsMeta(cfg)))
	defer metaServer.Close()

	firstResp, err := http.Get(metaServer.URL + "/api/plugins/meta")
	if err != nil {
		t.Fatalf("initial GET failed: %v", err)
	}
	defer firstResp.Body.Close()
	if firstResp.StatusCode != http.StatusOK {
		t.Fatalf("initial status = %d, want 200", firstResp.StatusCode)
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
	if _, err := connectsvc.UpsertPluginConfigWithService(svc, connectsvc.MetaInput{
		Key:     "email",
		Name:    "email",
		Meta:    `{"email":"demo@example.com","email_pop3":"imap.example.com:993","email_smtp":"smtp.example.com:465","email_password":"secret","email_whitelist":""}`,
		AgentID: "A",
		Model:   "openai",
	}); err != nil {
		t.Fatalf("save config failed: %v", err)
	}

	secondResp, err := http.Get(metaServer.URL + "/api/plugins/meta")
	if err != nil {
		t.Fatalf("second GET failed: %v", err)
	}
	defer secondResp.Body.Close()
	if secondResp.StatusCode != http.StatusOK {
		t.Fatalf("second status = %d, want 200", secondResp.StatusCode)
	}

	var payload struct {
		Status int `json:"status"`
		Data   []struct {
			Key   string                       `json:"key"`
			Name  string                       `json:"name"`
			Param connectsvc.PluginParamFields `json:"param"`
			Meta  map[string]any               `json:"meta"`
		} `json:"data"`
	}
	if err := json.NewDecoder(secondResp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode second payload: %v", err)
	}
	if payload.Status != 0 || len(payload.Data) != 1 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.Data[0].Key != "email" || payload.Data[0].Name != "邮件" {
		t.Fatalf("unexpected plugin identity: %+v", payload.Data[0])
	}
	if got := fmt.Sprint(payload.Data[0].Meta["email"]); got != "demo@example.com" {
		t.Fatalf("unexpected email meta: %+v", payload.Data[0].Meta)
	}
	if got := fmt.Sprint(payload.Data[0].Meta["email_pop3"]); got != "imap.example.com:993" {
		t.Fatalf("unexpected email_pop3 meta: %+v", payload.Data[0].Meta)
	}
	if got := fmt.Sprint(payload.Data[0].Meta["email_smtp"]); got != "smtp.example.com:465" {
		t.Fatalf("unexpected email_smtp meta: %+v", payload.Data[0].Meta)
	}
	if got := fmt.Sprint(payload.Data[0].Meta["email_password"]); got != "secret" {
		t.Fatalf("unexpected email_password meta: %+v", payload.Data[0].Meta)
	}
}

func TestPluginsConfigHandlerReturnsValidationError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
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

	cfg := &Config{ConnectCacheMs: 1000, AgentDir: "agent"}
	server := httptest.NewServer(http.HandlerFunc(handlePluginsConfig(cfg)))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/plugins/config?key=feishu&meta=%7B%7D&model=OpenAI", "application/json", nil)
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
	if payload.Status != 1 {
		t.Fatalf("status payload = %d", payload.Status)
	}
	if !strings.Contains(payload.Content, "agent") {
		t.Fatalf("content = %q", payload.Content)
	}
}

func TestCronCheckOnceInheritsRouterDisableIntoFutureTaskDetails(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	if cronDB != nil {
		_ = cronDB.Close()
		cronDB = nil
	}
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

	initCronDB()
	if cronDB == nil {
		t.Fatal("cronDB should not be nil")
	}
	defer func() {
		if cronDB != nil {
			_ = cronDB.Close()
			cronDB = nil
		}
	}()

	if err := ensureTokenStore(cronDB); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if err := writeTokenStore(cronDB, map[string]string{"OpenAI": "Bearer scheduled-token"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	rawTime := time.Now().Add(2 * time.Hour).Format("2006-01-02 15:04")
	res, err := cronDB.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, chat_id, task_type, model, thinking, router_disable, cron, content) VALUES (2, ?, 'A', 'chat-future', 'cron', 'OpenAI', 0, 0, '', '后续明细继承蜂群')`,
		rawTime)
	if err != nil {
		t.Fatalf("insert task_meta: %v", err)
	}
	metaID, _ := res.LastInsertId()

	cfg := &Config{AgentDir: agentRoot, Device: "dev"}
	cronCheckOnce(cfg)

	var detailCount int
	if err := cronDB.QueryRow(`SELECT COUNT(*) FROM task_detail WHERE meta_id = ? AND router_disable = 0`, metaID).Scan(&detailCount); err != nil {
		t.Fatalf("count task_detail router_disable=0: %v", err)
	}
	if detailCount == 0 {
		t.Fatalf("expected future task_detail rows inheriting router_disable=0 for meta %d", metaID)
	}

	var wrongCount int
	if err := cronDB.QueryRow(`SELECT COUNT(*) FROM task_detail WHERE meta_id = ? AND router_disable != 0`, metaID).Scan(&wrongCount); err != nil {
		t.Fatalf("count task_detail router_disable!=0: %v", err)
	}
	if wrongCount != 0 {
		t.Fatalf("unexpected task_detail rows with mismatched router_disable for meta %d: %d", metaID, wrongCount)
	}
}

func TestCreateImmediateCronTaskFromConnectCarriesSwarm(t *testing.T) {
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

	renameCronLogTables(db)
	db.Exec(`CREATE TABLE IF NOT EXISTS task_meta (id INTEGER PRIMARY KEY AUTOINCREMENT, cycle INTEGER NOT NULL, raw_time TEXT NOT NULL DEFAULT '', agent_id TEXT NOT NULL, model TEXT NOT NULL, thinking INTEGER NOT NULL DEFAULT 0, router_disable INTEGER NOT NULL DEFAULT 1, cron TEXT NOT NULL, content TEXT NOT NULL, response_schema TEXT NOT NULL DEFAULT '', chat_id TEXT NOT NULL DEFAULT '', task_type TEXT NOT NULL DEFAULT 'cron')`)
	db.Exec(`CREATE TABLE IF NOT EXISTS task_detail (id INTEGER PRIMARY KEY AUTOINCREMENT, meta_id INTEGER NOT NULL, exec_time INTEGER NOT NULL, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL DEFAULT '', meta_ref TEXT NOT NULL DEFAULT '', task_type TEXT NOT NULL DEFAULT 'cron', model TEXT NOT NULL, thinking INTEGER NOT NULL DEFAULT 0, router_disable INTEGER NOT NULL DEFAULT 1, content TEXT NOT NULL, response_schema TEXT NOT NULL DEFAULT '', started INTEGER NOT NULL DEFAULT 0)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS cron_meta_log (id INTEGER PRIMARY KEY AUTOINCREMENT, meta_id INTEGER NOT NULL, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL DEFAULT '', task_type TEXT NOT NULL DEFAULT 'cron', action TEXT NOT NULL, cycle INTEGER NOT NULL, raw_time TEXT NOT NULL DEFAULT '', model TEXT NOT NULL, thinking INTEGER NOT NULL DEFAULT 0, router_disable INTEGER NOT NULL DEFAULT 1, cron TEXT NOT NULL DEFAULT '', content TEXT NOT NULL, response_schema TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS cron_detail_log (id INTEGER PRIMARY KEY AUTOINCREMENT, detail_id INTEGER NOT NULL, meta_id INTEGER NOT NULL, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL DEFAULT '', task_type TEXT NOT NULL DEFAULT 'cron', action TEXT NOT NULL, exec_time INTEGER NOT NULL, model TEXT NOT NULL, thinking INTEGER NOT NULL DEFAULT 0, router_disable INTEGER NOT NULL DEFAULT 1, content TEXT NOT NULL, response_schema TEXT NOT NULL DEFAULT '', started INTEGER NOT NULL DEFAULT 0, occurred_at TEXT NOT NULL)`)

	meta := &connectsvc.Meta{
		Key:           "feishu",
		Name:          "feishu",
		AgentID:       "A",
		ChatID:        "chat-feishu",
		Model:         "OpenAI",
		Thinking:      true,
		RouterDisable: true,
	}
	reqs := []connectsvc.Request{{
		ID:             7,
		Name:           "feishu",
		Request:        "hello",
		ResponseSchema: `{"type":"object"}`,
	}}

	lastRequestID, err := createImmediateCronTaskFromConnect(db, meta, reqs, 0)
	if err != nil {
		t.Fatalf("createImmediateCronTaskFromConnect: %v", err)
	}
	if lastRequestID != "7" {
		t.Fatalf("lastRequestID = %q, want 7", lastRequestID)
	}

	var metaRouterDisable, detailRouterDisable int
	if err := db.QueryRow(`SELECT router_disable FROM task_meta WHERE task_type = 'feishu' LIMIT 1`).Scan(&metaRouterDisable); err != nil {
		t.Fatalf("query task_meta router_disable: %v", err)
	}
	if err := db.QueryRow(`SELECT router_disable FROM task_detail WHERE task_type = 'feishu' LIMIT 1`).Scan(&detailRouterDisable); err != nil {
		t.Fatalf("query task_detail router_disable: %v", err)
	}
	if metaRouterDisable != 1 || detailRouterDisable != 1 {
		t.Fatalf("router_disable values = meta:%d detail:%d, want 1/1", metaRouterDisable, detailRouterDisable)
	}
}

func TestPluginsLogHandlerStreamsExistingAndAppendedLines(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()

	if err := os.WriteFile(filepath.Join("..", "plugins", "feishu.log"), []byte("old-1\nold-2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{}
	server := httptest.NewServer(http.HandlerFunc(handlePluginsLog(cfg)))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/plugins/log?key=feishu&last=2")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}

	linesCh := make(chan string, 8)
	errCh := make(chan error, 1)
	go collectIntegrationPluginLogEvents(resp.Body, linesCh, errCh)

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

func TestPluginsLogHandlerEmitsErrorWhenFileMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()

	logPath := filepath.Join("..", "plugins", "feishu.log")
	if err := os.WriteFile(logPath, []byte("before-delete\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{}
	server := httptest.NewServer(http.HandlerFunc(handlePluginsLog(cfg)))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/plugins/log?key=feishu&last=1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	linesCh := make(chan string, 8)
	errCh := make(chan error, 1)
	go collectIntegrationPluginLogEvents(resp.Body, linesCh, errCh)

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

func TestPluginsLogHandlerReturnsMissingFileMessageWhenLogDoesNotExistAtStart(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
		"a": {`"A"`, `[]`},
	})
	defer restore()

	cfg := &Config{}
	server := httptest.NewServer(http.HandlerFunc(handlePluginsLog(cfg)))
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

func TestPluginsLogHandlerWaitsForLogFileCreationAtStart(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()

	cfg := &Config{}
	server := httptest.NewServer(http.HandlerFunc(handlePluginsLog(cfg)))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/plugins/log?key=feishu&last=1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = os.WriteFile(filepath.Join("..", "plugins", "feishu.log"), []byte("created-late\n"), 0o644)
	}()

	linesCh := make(chan string, 8)
	errCh := make(chan error, 1)
	go collectIntegrationPluginLogEvents(resp.Body, linesCh, errCh)

	select {
	case line := <-linesCh:
		if line != "created-late" {
			t.Fatalf("line = %q, want created-late", line)
		}
	case err := <-errCh:
		t.Fatalf("read stream early: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting created log line")
	}
}

func TestPluginsStartHandlerRequiresKey(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()

	writeIntegrationPluginActionScript(t, filepath.Join("..", "plugins", "feishu"), `"飞书"`, pluginParamJSON("appId", "appSecret"), "subcommand")

	cfg := &Config{}
	server := httptest.NewServer(http.HandlerFunc(handlePluginsStart(cfg)))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/plugins/start?key=feishu&connect-bin=.%2Fconnect", "application/json", nil)
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
		t.Fatal(err)
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
}

func TestPluginsExecHandlerRunsNestedCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
		"browser": {`"浏览器"`, pluginParamJSON("agentId", "chatId")},
	})
	defer restore()

	writeIntegrationPluginExecScript(t, filepath.Join("..", "plugins", "browser"), `"浏览器"`, pluginParamJSON("agentId", "chatId"))

	cfg := &Config{}
	server := httptest.NewServer(http.HandlerFunc(handlePluginsExec(cfg)))
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
		t.Fatal(err)
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

func TestPluginsStartHandlerDefaultsConnectBinToCurrentIntegration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
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
		"  printf '%s\\n' '[\"appId\",\"appSecret\"]'\n" +
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

	cfg := &Config{}
	server := httptest.NewServer(http.HandlerFunc(handlePluginsStart(cfg)))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/plugins/start?key=feishu", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var payload struct {
		Status int `json:"status"`
		Data   struct {
			Output map[string]any `json:"output"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
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

func TestRemotePluginWriteEndpointsRejectNonLocalRequests(t *testing.T) {
	cfg := &Config{}
	cases := []struct {
		name    string
		method  string
		target  string
		handler http.HandlerFunc
	}{
		{name: "config", method: http.MethodPost, target: "/api/plugins/config", handler: handlePluginsConfig(cfg)},
		{name: "exec", method: http.MethodGet, target: "/api/plugins/exec", handler: handlePluginsExec(cfg)},
		{name: "start", method: http.MethodPost, target: "/api/plugins/start", handler: handlePluginsStart(cfg)},
		{name: "stop", method: http.MethodPost, target: "/api/plugins/stop", handler: handlePluginsStop(cfg)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.target, nil)
			req.RemoteAddr = "203.0.113.10:4567"
			rec := httptest.NewRecorder()

			tc.handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "remote access is limited to viewing plugin list, status, and logs") {
				t.Fatalf("unexpected body: %s", rec.Body.String())
			}
		})
	}
}

func TestHandlePluginsStatusPreservesBrowserStartedStateForRemoteHostAlias(t *testing.T) {
	originalRunConnectPluginStatus := runConnectPluginStatus
	defer func() {
		runConnectPluginStatus = originalRunConnectPluginStatus
	}()

	runConnectPluginStatus = func(target string, flags map[string]string) (*connectsvc.PluginStatus, error) {
		return &connectsvc.PluginStatus{
			Key:     "browser",
			Name:    "浏览器",
			Path:    "/tmp/browser",
			PIDFile: "/tmp/browser.pid",
			Started: true,
			PID:     4321,
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/status?key=browser", nil)
	req.RemoteAddr = "127.0.0.1:34567"
	req.Host = "remote.example.com:8080"
	rec := httptest.NewRecorder()

	handlePluginsStatus(&Config{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Status int `json:"status"`
		Data   struct {
			Key     string `json:"key"`
			Started bool   `json:"started"`
			PID     int    `json:"pid"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v body=%s", err, rec.Body.String())
	}
	if payload.Status != 0 {
		t.Fatalf("payload status = %d", payload.Status)
	}
	if payload.Data.Key != "browser" {
		t.Fatalf("key = %q, want browser", payload.Data.Key)
	}
	if !payload.Data.Started {
		t.Fatalf("started = false, want true for remote host alias")
	}
	if payload.Data.PID != 4321 {
		t.Fatalf("pid = %d, want 4321 for remote host alias", payload.Data.PID)
	}
}

func TestIntegrationPluginsStartCLI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()

	writeIntegrationPluginActionScript(t, filepath.Join("..", "plugins", "feishu"), `"飞书"`, pluginParamJSON("appId", "appSecret"), "subcommand")

	stdout := captureIntegrationStdout(t, func() {
		runIntegrationPluginCLI([]string{"start", "--key", "feishu"})
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
	if len(payload.Data.Command) == 0 || payload.Data.Command[0] != "start" {
		t.Fatalf("command = %v, want start", payload.Data.Command)
	}
	if got := fmt.Sprint(payload.Data.Output["mode"]); got != "subcommand" {
		t.Fatalf("output = %+v", payload.Data.Output)
	}
}

func TestIntegrationPluginsExecCLI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
		"browser": {`"浏览器"`, pluginParamJSON("agentId", "chatId")},
	})
	defer restore()

	writeIntegrationPluginExecScript(t, filepath.Join("..", "plugins", "browser"), `"浏览器"`, pluginParamJSON("agentId", "chatId"))

	stdout := captureIntegrationStdout(t, func() {
		runIntegrationPluginCLI([]string{"exec", "--key", "browser", "--command", "instance init", "--agentId", "a1", "--chatId", "b1"})
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

func TestIntegrationTopLevelListPluginsCLI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
		"feishu": {`{"key":"feishu","name":"飞书"}`, pluginParamJSON("appId", "appSecret")},
		"email":  {`{"key":"email","name":"邮件"}`, `[{"email":"","email_pop3":"","email_smtp":"","email_password":"","email_whitelist":""}]`},
	})
	defer restore()

	stdout := captureIntegrationStdout(t, func() {
		runIntegrationConnectCLI([]string{"list-plugins"})
	})

	var items []connectsvc.PluginInfo
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("decode list-plugins output: %v; raw=%s", err, stdout)
	}
	if len(items) != 2 {
		t.Fatalf("items = %+v, want 2 plugins", items)
	}
	if items[0].Key != "email" || items[1].Key != "feishu" {
		t.Fatalf("unexpected plugin keys: %+v", items)
	}
}

func TestIntegrationTopLevelListPluginsRecognizesScriptExtensions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, nil)
	defer restore()

	writeIntegrationPluginScript(t, filepath.Join("..", "plugins", "browser.py"), `{"key":"browser","name":"浏览器"}`, pluginParamJSON("cookie_path"))
	writeIntegrationPluginScript(t, filepath.Join("..", "plugins", "mail.js"), `{"key":"mail","name":"邮件"}`, pluginParamJSON("token"))
	writeIntegrationPluginScript(t, filepath.Join("..", "plugins", "calc.go"), `{"key":"calc","name":"计算器"}`, pluginParamJSON("expr"))
	if err := os.MkdirAll(filepath.Join("..", "plugins", "User_Data"), 0o755); err != nil {
		t.Fatalf("mkdir plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join("..", "plugins", "User_Data.zip"), []byte("zip"), 0o644); err != nil {
		t.Fatalf("write zip file: %v", err)
	}

	stdout := captureIntegrationStdout(t, func() {
		runIntegrationConnectCLI([]string{"list-plugins"})
	})

	var items []connectsvc.PluginInfo
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("decode list-plugins output: %v; raw=%s", err, stdout)
	}
	if len(items) != 3 {
		t.Fatalf("items = %+v, want 3 plugins", items)
	}
	if items[0].Key != "browser" || items[1].Key != "calc" || items[2].Key != "mail" {
		t.Fatalf("unexpected plugin keys: %+v", items)
	}
}

func TestIntegrationTopLevelMetaGetByKeyCLI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
		"feishu": {`{"key":"feishu","name":"飞书"}`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()

	seedIntegrationPluginMetaTestData(t, "data", "agent", connectsvc.MetaInput{
		Key:      "feishu",
		Meta:     `{"appId":"cli-app","appSecret":"cli-secret"}`,
		Stream:   true,
		Callback: "./feishu",
		AgentID:  "A",
		ChatID:   "chat-001",
		Model:    "OpenAI",
		Thinking: true,
	})

	stdout := captureIntegrationStdout(t, func() {
		runIntegrationConnectCLI([]string{"meta-get", "--key", "feishu"})
	})

	var item connectsvc.MetaConfig
	if err := json.Unmarshal([]byte(stdout), &item); err != nil {
		t.Fatalf("decode meta-get output: %v; raw=%s", err, stdout)
	}
	if item.Key != "feishu" || item.Name != "飞书" {
		t.Fatalf("unexpected meta-get item: %+v", item)
	}
	if got := fmt.Sprint(item.Meta["appId"]); got != "cli-app" {
		t.Fatalf("unexpected meta-get config: %+v", item.Meta)
	}
	if !item.Stream || !item.Thinking {
		t.Fatalf("unexpected meta-get runtime fields: %+v", item)
	}
}

func TestIntegrationTopLevelMetaCreateAndUpdateByKeyCLI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
		"feishu": {`{"key":"feishu","name":"飞书"}`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()
	t.Setenv("AGENT_DIR", "agent")
	if err := os.MkdirAll(filepath.Join("agent", "A"), 0o755); err != nil {
		t.Fatalf("mkdir agent: %v", err)
	}
	db, err := sql.Open("sqlite", "data")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := writeTokenStore(db, map[string]string{"OpenAI": "Bearer test-token"}); err != nil {
		_ = db.Close()
		t.Fatalf("writeTokenStore: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	stdout := captureIntegrationStdout(t, func() {
		runIntegrationConnectCLI([]string{
			"meta-create",
			"--key", "feishu",
			"--meta", `{"appId":"cli-app","appSecret":"cli-secret"}`,
			"--stream", "true",
			"--callback", "./ignored",
			"--agent", "A",
			"--chatId", "chat-001",
			"--model", "OpenAI",
			"--thinking", "true",
		})
	})

	var created connectsvc.Meta
	if err := json.Unmarshal([]byte(stdout), &created); err != nil {
		t.Fatalf("decode meta-create output: %v; raw=%s", err, stdout)
	}
	if created.Name != "feishu" {
		t.Fatalf("unexpected created meta: %+v", created)
	}
	if filepath.Base(created.Callback) != "feishu" {
		t.Fatalf("unexpected callback: %+v", created)
	}

	stdout = captureIntegrationStdout(t, func() {
		runIntegrationConnectCLI([]string{
			"meta-update",
			"--key", "feishu",
			"--meta", `{"appId":"new-app","appSecret":"new-secret"}`,
			"--callback", "./ignored-again",
			"--chatId", "chat-002",
		})
	})

	var updated connectsvc.Meta
	if err := json.Unmarshal([]byte(stdout), &updated); err != nil {
		t.Fatalf("decode meta-update output: %v; raw=%s", err, stdout)
	}
	if updated.Name != "feishu" {
		t.Fatalf("unexpected updated meta: %+v", updated)
	}
	if filepath.Base(updated.Callback) != "feishu" {
		t.Fatalf("unexpected updated callback: %+v", updated)
	}
	if updated.ChatID != "chat-002" {
		t.Fatalf("unexpected updated chat id: %+v", updated)
	}
}

func TestIntegrationCronCreateHelp(t *testing.T) {
	stdout := captureIntegrationStdout(t, func() {
		runIntegrationCronCLI([]string{"create", "--help"})
	})
	if !strings.Contains(stdout, "Usage:\n  integration cron create [options]") {
		t.Fatalf("missing create usage: %s", stdout)
	}
	if !strings.Contains(stdout, "--router_disable[=true|false]") {
		t.Fatalf("missing create router_disable flag: %s", stdout)
	}
	if !strings.Contains(stdout, "Examples:") || !strings.Contains(stdout, "integration cron create --content") {
		t.Fatalf("missing create examples: %s", stdout)
	}
}

func TestIntegrationCronCreateCronHelp(t *testing.T) {
	stdout := captureIntegrationStdout(t, func() {
		runIntegrationCronCLI([]string{"create-cron", "--help"})
	})
	if !strings.Contains(stdout, "Usage:\n  integration cron create-cron [options]") {
		t.Fatalf("missing create-cron usage: %s", stdout)
	}
	if !strings.Contains(stdout, "--router_disable[=true|false]") {
		t.Fatalf("missing create-cron router_disable flag: %s", stdout)
	}
	if !strings.Contains(stdout, "--cron EXPR") || !strings.Contains(stdout, "工作日中午整理异常日志") {
		t.Fatalf("missing create-cron details: %s", stdout)
	}
}

func TestIntegrationCronCreateCLIWithSwarm(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	agentRoot := filepath.Join(tmp, "agent")
	agentDir := filepath.Join(agentRoot, "alpha")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("soul"), 0o644); err != nil {
		t.Fatalf("write soul: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "USER.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	initCronDB()
	if cronDB == nil {
		t.Fatal("cronDB should not be nil")
	}
	defer func() {
		_ = cronDB.Close()
		cronDB = nil
	}()
	if err := writeTokenStore(cronDB, map[string]string{"openai": "sk-test"}); err != nil {
		t.Fatalf("writeTokenStore: %v", err)
	}

	rawTime := time.Now().Add(2 * time.Minute).Format("2006-01-02 15:04")
	stdout := captureIntegrationStdout(t, func() {
		runIntegrationCronCLI([]string{
			"create",
			"--agent-dir", agentRoot,
			"--content", "整理日报",
			"--model", "openai",
			"--rawTime", rawTime,
			"--cycle", "0",
			"--chatId", "chat-cli-1",
			"--agent", "alpha",
			"--router_disable", "true",
		})
	})

	var resp struct {
		Status int   `json:"status"`
		ID     int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("decode stdout: %v; raw=%s", err, stdout)
	}
	if resp.Status != 0 {
		t.Fatalf("status = %d; raw=%s", resp.Status, stdout)
	}

	var metaRouterDisable int
	if err := cronDB.QueryRow(`SELECT router_disable FROM task_meta WHERE id = ?`, resp.ID).Scan(&metaRouterDisable); err != nil {
		t.Fatalf("query task_meta router_disable: %v", err)
	}
	if metaRouterDisable != 1 {
		t.Fatalf("task_meta.router_disable = %d, want 1", metaRouterDisable)
	}
}

func TestIntegrationPluginsHelp(t *testing.T) {
	stdout := captureIntegrationStdout(t, func() {
		runIntegrationPluginCLI([]string{"--help"})
	})
	if !strings.Contains(stdout, "integration plugins start --key KEY") {
		t.Fatalf("missing plugins usage: %s", stdout)
	}
	if !strings.Contains(stdout, "Examples:") || !strings.Contains(stdout, "integration plugins status --key feishu") {
		t.Fatalf("missing plugins examples: %s", stdout)
	}
	if !strings.Contains(stdout, "integration plugins sync-bundled [--check]") {
		t.Fatalf("missing plugins sync-bundled usage: %s", stdout)
	}
}

func TestIntegrationCLIHelpIncludesSkillsWarning(t *testing.T) {
	stdout := captureIntegrationStdout(t, func() {
		printCLIHelp()
	})
	if !strings.Contains(stdout, "integration skills-warning [--refresh]") {
		t.Fatalf("missing skills-warning usage: %s", stdout)
	}
	if !strings.Contains(stdout, "skills-warning     Read current SKILL parse warnings from shared sqlite") {
		t.Fatalf("missing skills-warning command description: %s", stdout)
	}
}

func TestIntegrationCLIHelpIncludesFileLastUpdate(t *testing.T) {
	stdout := captureIntegrationStdout(t, func() {
		printCLIHelp()
	})
	if !strings.Contains(stdout, "integration file-last-update [options]") {
		t.Fatalf("missing file-last-update usage: %s", stdout)
	}
	if !strings.Contains(stdout, "file-last-update   Print milliseconds since the target file was last updated") {
		t.Fatalf("missing file-last-update command description: %s", stdout)
	}
}

func TestIntegrationCLIHelpIncludesBackupClean(t *testing.T) {
	stdout := captureIntegrationStdout(t, func() {
		printCLIHelp()
	})
	if !strings.Contains(stdout, "integration backup-clean [options]") {
		t.Fatalf("missing backup-clean usage: %s", stdout)
	}
	if !strings.Contains(stdout, "backup-clean       Move stale User/Soul backup files into bak and delete stale bak files") {
		t.Fatalf("missing backup-clean command description: %s", stdout)
	}
}

func TestIntegrationConnectHelp(t *testing.T) {
	stdout := captureIntegrationStdout(t, func() {
		printConnectHelp()
	})
	if !strings.Contains(stdout, "integration connect <subcommand> [options]") {
		t.Fatalf("missing connect usage: %s", stdout)
	}
	if !strings.Contains(stdout, "integration connect meta-create --key feishu") || !strings.Contains(stdout, "integration connect meta-list") {
		t.Fatalf("missing connect examples: %s", stdout)
	}
}

func TestIntegrationKnowledgeHelp(t *testing.T) {
	stdout := captureIntegrationStdout(t, func() {
		runIntegrationKnowledgeCLI([]string{"--help"})
	})
	if !strings.Contains(stdout, "integration knowledge update-time --timestamp N --agentId ID") {
		t.Fatalf("missing knowledge usage: %s", stdout)
	}
	if !strings.Contains(stdout, "integration knowledge update-time --timestamp 1715337600000 --agentId demo-agent") {
		t.Fatalf("missing knowledge examples: %s", stdout)
	}
}

func TestIntegrationCLIHelpIncludesHost(t *testing.T) {
	stdout := captureIntegrationStdout(t, func() {
		printCLIHelp()
	})
	if !strings.Contains(stdout, "integration host get [--addr URL]") {
		t.Fatalf("missing host usage: %s", stdout)
	}
	if !strings.Contains(stdout, "host              Read or update the in-memory upstream host of a running integration service") {
		t.Fatalf("missing host description: %s", stdout)
	}
}

func TestIntegrationCLIHelpIncludesStandalone(t *testing.T) {
	stdout := captureIntegrationStdout(t, func() {
		printCLIHelp()
	})
	if !strings.Contains(stdout, "integration standalone get [--addr URL]") {
		t.Fatalf("missing standalone usage: %s", stdout)
	}
	if !strings.Contains(stdout, "standalone        Read or update the in-memory localhost-only mode of a running integration service") {
		t.Fatalf("missing standalone description: %s", stdout)
	}
}

func TestRunIntegrationHostCLIRejectsUnsupportedPort(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runIntegrationHostCLI([]string{"get", "--port", "9090"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "--port only supports 8080") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunIntegrationStandaloneCLIRejectsUnsupportedPort(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runIntegrationStandaloneCLI([]string{"get", "--port", "9090"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "--port only supports 8080") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestHandleRuntimeHost(t *testing.T) {
	cfg := &Config{Host: "https://www.deepright.cn"}

	decode := func(t *testing.T, recorder *httptest.ResponseRecorder) struct {
		Status  int    `json:"status"`
		Content string `json:"content"`
		Data    struct {
			Host        string `json:"host"`
			StartupHost string `json:"startupHost"`
			Overridden  bool   `json:"overridden"`
			RuntimeOnly bool   `json:"runtimeOnly"`
		} `json:"data"`
	} {
		t.Helper()
		var payload struct {
			Status  int    `json:"status"`
			Content string `json:"content"`
			Data    struct {
				Host        string `json:"host"`
				StartupHost string `json:"startupHost"`
				Overridden  bool   `json:"overridden"`
				RuntimeOnly bool   `json:"runtimeOnly"`
			} `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
		}
		return payload
	}

	handler := handleRuntimeHost(cfg)

	getReq := httptest.NewRequest(http.MethodGet, "/api/host", nil)
	getReq.RemoteAddr = "127.0.0.1:34567"
	getReq.Host = "127.0.0.1:8080"
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	getPayload := decode(t, getRec)
	if getPayload.Status != 0 || getPayload.Data.Host != "https://www.deepright.cn" || getPayload.Data.StartupHost != "https://www.deepright.cn" || getPayload.Data.Overridden {
		t.Fatalf("unexpected initial payload: %#v", getPayload)
	}

	setReq := httptest.NewRequest(http.MethodPost, "/api/host", strings.NewReader(`{"host":"https://staging.deepright.cn/base"}`))
	setReq.RemoteAddr = "127.0.0.1:34567"
	setReq.Host = "127.0.0.1:8080"
	setReq.Header.Set("Content-Type", "application/json")
	setRec := httptest.NewRecorder()
	handler.ServeHTTP(setRec, setReq)
	setPayload := decode(t, setRec)
	if setPayload.Status != 0 || setPayload.Data.Host != "https://staging.deepright.cn/base" || !setPayload.Data.Overridden {
		t.Fatalf("unexpected set payload: %#v", setPayload)
	}
	if got := cfg.currentHost(); got != "https://staging.deepright.cn/base" {
		t.Fatalf("cfg.currentHost() = %q, want %q", got, "https://staging.deepright.cn/base")
	}

	resetReq := httptest.NewRequest(http.MethodDelete, "/api/host", nil)
	resetReq.RemoteAddr = "127.0.0.1:34567"
	resetReq.Host = "127.0.0.1:8080"
	resetRec := httptest.NewRecorder()
	handler.ServeHTTP(resetRec, resetReq)
	resetPayload := decode(t, resetRec)
	if resetPayload.Status != 0 || resetPayload.Data.Host != "https://www.deepright.cn" || resetPayload.Data.Overridden {
		t.Fatalf("unexpected reset payload: %#v", resetPayload)
	}
}

func TestHandleRuntimeHostRejectsRemoteRequest(t *testing.T) {
	cfg := &Config{Host: "https://www.deepright.cn"}
	req := httptest.NewRequest(http.MethodPost, "/api/host", strings.NewReader(`{"host":"https://staging.deepright.cn"}`))
	req.RemoteAddr = "203.0.113.10:4567"
	rec := httptest.NewRecorder()

	handleRuntimeHost(cfg).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if !strings.Contains(rec.Body.String(), "only localhost requests can modify runtime host") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestHandleRuntimeStandalone(t *testing.T) {
	cfg := &Config{}

	decode := func(t *testing.T, recorder *httptest.ResponseRecorder) struct {
		Status  int    `json:"status"`
		Content string `json:"content"`
		Data    struct {
			Enabled        bool `json:"enabled"`
			StartupEnabled bool `json:"startupEnabled"`
			Overridden     bool `json:"overridden"`
			RuntimeOnly    bool `json:"runtimeOnly"`
		} `json:"data"`
	} {
		t.Helper()
		var payload struct {
			Status  int    `json:"status"`
			Content string `json:"content"`
			Data    struct {
				Enabled        bool `json:"enabled"`
				StartupEnabled bool `json:"startupEnabled"`
				Overridden     bool `json:"overridden"`
				RuntimeOnly    bool `json:"runtimeOnly"`
			} `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
		}
		return payload
	}

	handler := handleRuntimeStandalone(cfg)

	getReq := httptest.NewRequest(http.MethodGet, "/api/standalone", nil)
	getReq.RemoteAddr = "127.0.0.1:34567"
	getReq.Host = "127.0.0.1:8080"
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	getPayload := decode(t, getRec)
	if getPayload.Status != 0 || getPayload.Data.Enabled || getPayload.Data.StartupEnabled || getPayload.Data.Overridden {
		t.Fatalf("unexpected initial payload: %#v", getPayload)
	}

	setReq := httptest.NewRequest(http.MethodPost, "/api/standalone=true", nil)
	setReq.RemoteAddr = "127.0.0.1:34567"
	setReq.Host = "127.0.0.1:8080"
	setRec := httptest.NewRecorder()
	handler.ServeHTTP(setRec, setReq)
	setPayload := decode(t, setRec)
	if setPayload.Status != 0 || !setPayload.Data.Enabled || !setPayload.Data.Overridden {
		t.Fatalf("unexpected set payload: %#v", setPayload)
	}
	if !cfg.standaloneEnabled() {
		t.Fatalf("cfg.standaloneEnabled() = false, want true")
	}

	resetReq := httptest.NewRequest(http.MethodDelete, "/api/standalone", nil)
	resetReq.RemoteAddr = "127.0.0.1:34567"
	resetReq.Host = "127.0.0.1:8080"
	resetRec := httptest.NewRecorder()
	handler.ServeHTTP(resetRec, resetReq)
	resetPayload := decode(t, resetRec)
	if resetPayload.Status != 0 || resetPayload.Data.Enabled || resetPayload.Data.Overridden {
		t.Fatalf("unexpected reset payload: %#v", resetPayload)
	}
}

func TestHandleRuntimeStandaloneRejectsRemoteRequest(t *testing.T) {
	cfg := &Config{}
	req := httptest.NewRequest(http.MethodPost, "/api/standalone=true", nil)
	req.RemoteAddr = "203.0.113.10:4567"
	rec := httptest.NewRecorder()

	handleRuntimeStandalone(cfg).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if !strings.Contains(rec.Body.String(), "only localhost/127.0.0.1/::1 requests can modify standalone mode") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestSelectPreferredLANAccessHost(t *testing.T) {
	host := selectPreferredLANAccessHost([]lanAccessCandidate{
		{Host: "172.17.0.1", InterfaceName: "docker0", InterfaceIndex: 7, Private: true},
		{Host: "192.168.31.9", InterfaceName: "en0", InterfaceIndex: 3, Private: true},
		{Host: "10.0.0.8", InterfaceName: "utun3", InterfaceIndex: 2, Private: true},
	})
	if host != "192.168.31.9" {
		t.Fatalf("host = %q, want %q", host, "192.168.31.9")
	}
}

func TestHandleLANAccessURLReturnsExpectedURL(t *testing.T) {
	cfg := &Config{Port: 9876}
	orig := detectLANAccessHost
	detectLANAccessHost = func() (string, error) {
		return "192.168.31.9", nil
	}
	defer func() {
		detectLANAccessHost = orig
	}()

	req := httptest.NewRequest(http.MethodGet, "/api/site/access", nil)
	req.RemoteAddr = "127.0.0.1:34567"
	req.Host = "127.0.0.1:9876"
	rec := httptest.NewRecorder()

	handleLANAccessURL(cfg).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var payload struct {
		Status  int    `json:"status"`
		Content string `json:"content"`
		Data    struct {
			Host string `json:"host"`
			URL  string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rec.Body.String())
	}
	if payload.Status != 0 {
		t.Fatalf("status payload = %d, want 0", payload.Status)
	}
	if payload.Data.Host != "192.168.31.9" {
		t.Fatalf("host = %q, want %q", payload.Data.Host, "192.168.31.9")
	}
	if payload.Data.URL != buildLANAccessURL("192.168.31.9", 9876) {
		t.Fatalf("url = %q, want %q", payload.Data.URL, buildLANAccessURL("192.168.31.9", 9876))
	}
}

func TestHandleLANAccessURLRejectsRemoteRequest(t *testing.T) {
	cfg := &Config{Port: 8080}
	req := httptest.NewRequest(http.MethodGet, "/api/site/access", nil)
	req.RemoteAddr = "203.0.113.10:4567"
	rec := httptest.NewRecorder()

	handleLANAccessURL(cfg).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if !strings.Contains(rec.Body.String(), "only localhost requests can query LAN access URL") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestHandleShutdownRejectsRemoteRequest(t *testing.T) {
	controller := newIntegrationShutdownController(func() {}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
	req.RemoteAddr = "203.0.113.10:4567"
	rec := httptest.NewRecorder()

	handleShutdown(controller).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if !strings.Contains(rec.Body.String(), "only localhost/127.0.0.1/::1 requests can shut down integration") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestIsLocalManagementRequestRejectsRemoteHostEvenWhenRemoteAddrIsLoopback(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/plugins/status?key=browser", nil)
	req.RemoteAddr = "127.0.0.1:34567"
	req.Host = "remote.example.com:8080"
	if isLocalManagementRequest(req) {
		t.Fatal("expected remote host alias to be treated as remote access")
	}
}

type standaloneHijackRecorder struct {
	*httptest.ResponseRecorder
	conn     net.Conn
	hijacked bool
}

func newStandaloneHijackRecorder(t *testing.T) (*standaloneHijackRecorder, net.Conn) {
	t.Helper()
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		_ = clientConn.Close()
	})
	return &standaloneHijackRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		conn:             serverConn,
	}, clientConn
}

func (r *standaloneHijackRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	r.hijacked = true
	return r.conn, bufio.NewReadWriter(bufio.NewReader(r.conn), bufio.NewWriter(r.conn)), nil
}

func TestWithStandaloneAPIProtectionBlocksRemoteStaticAndAPI(t *testing.T) {
	cfg := &Config{}
	cfg.setStandaloneEnabled(true)

	var reachedNext bool
	handler := withStandaloneAPIProtection(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reachedNext = true
		w.WriteHeader(http.StatusNoContent)
	}), cfg)

	staticReq := httptest.NewRequest(http.MethodGet, "/site/index.html", nil)
	staticReq.RemoteAddr = "203.0.113.10:4567"
	staticRec, staticPeer := newStandaloneHijackRecorder(t)
	handler.ServeHTTP(staticRec, staticReq)
	if reachedNext {
		t.Fatal("static request should not reach next handler")
	}
	if !staticRec.hijacked {
		t.Fatal("static request should hijack and close the connection")
	}
	_ = staticPeer.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var staticBuf [1]byte
	if n, err := staticPeer.Read(staticBuf[:]); n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("static connection should close without payload, n=%d err=%v", n, err)
	}

	apiReq := httptest.NewRequest(http.MethodGet, "/api/agentId", nil)
	apiReq.RemoteAddr = "203.0.113.10:4567"
	apiRec, apiPeer := newStandaloneHijackRecorder(t)
	handler.ServeHTTP(apiRec, apiReq)
	if reachedNext {
		t.Fatal("api request should not reach next handler")
	}
	if !apiRec.hijacked {
		t.Fatal("api request should hijack and close the connection")
	}
	_ = apiPeer.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var apiBuf [1]byte
	if n, err := apiPeer.Read(apiBuf[:]); n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("api connection should close without payload, n=%d err=%v", n, err)
	}

	optionsReq := httptest.NewRequest(http.MethodOptions, "/site/index.html", nil)
	optionsReq.RemoteAddr = "203.0.113.10:4567"
	optionsRec, optionsPeer := newStandaloneHijackRecorder(t)
	handler.ServeHTTP(optionsRec, optionsReq)
	if reachedNext {
		t.Fatal("options request should not reach next handler")
	}
	if !optionsRec.hijacked {
		t.Fatal("options request should hijack and close the connection")
	}
	_ = optionsPeer.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var optionsBuf [1]byte
	if n, err := optionsPeer.Read(optionsBuf[:]); n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("options connection should close without payload, n=%d err=%v", n, err)
	}

	localReq := httptest.NewRequest(http.MethodGet, "/site/index.html", nil)
	localReq.RemoteAddr = "127.0.0.1:4567"
	localReq.Host = "127.0.0.1:8080"
	localRec := httptest.NewRecorder()
	handler.ServeHTTP(localRec, localReq)
	if localRec.Code != http.StatusNoContent {
		t.Fatalf("local status = %d, want %d", localRec.Code, http.StatusNoContent)
	}
	if !reachedNext {
		t.Fatal("local request should reach next handler")
	}
}

func TestIntegrationCLIHelpIncludesToken(t *testing.T) {
	stdout := captureIntegrationStdout(t, func() {
		printCLIHelp()
	})
	if !strings.Contains(stdout, "integration token [--provider MODEL]") {
		t.Fatalf("missing token usage: %s", stdout)
	}
	if !strings.Contains(stdout, "integration token get [--n N]") {
		t.Fatalf("missing token get usage: %s", stdout)
	}
	if !strings.Contains(stdout, "token             Print saved model tokens, query token consume details, or record token consume details") {
		t.Fatalf("missing token command description: %s", stdout)
	}
}

func TestIntegrationTokenGetHelp(t *testing.T) {
	stdout := captureIntegrationStdout(t, func() {
		runIntegrationTokenCLI([]string{"get", "--help"})
	})
	if !strings.Contains(stdout, "integration token get [--n N]") {
		t.Fatalf("missing token get help usage: %s", stdout)
	}
	if !strings.Contains(stdout, "--n N            查询最新 N 条本地 token 用量记录") {
		t.Fatalf("missing token get help option: %s", stdout)
	}
}

func TestIntegrationCLIHelpIncludesSandbox(t *testing.T) {
	stdout := captureIntegrationStdout(t, func() {
		printCLIHelp()
	})
	if !strings.Contains(stdout, "integration sandbox --agentId ID --chatId ID [--sandbox off|filepick|net|filepick_net] [--dir DIR]") {
		t.Fatalf("missing sandbox usage: %s", stdout)
	}
	if !strings.Contains(stdout, "sandbox           Read or update sandbox mode for one AgentId + ChatId") {
		t.Fatalf("missing sandbox command description: %s", stdout)
	}
}

func TestIntegrationHandleSandboxPersistsByAgentAndChat(t *testing.T) {
	tmp := t.TempDir()

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
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
	oldCronDB := cronDB
	cronDB = db
	defer func() { cronDB = oldCronDB }()
	oldPrimeSandbox := integrationPrimeSandboxModeFn
	integrationPrimeSandboxModeFn = func(string) (string, error) { return "", nil }
	defer func() { integrationPrimeSandboxModeFn = oldPrimeSandbox }()

	readReq := httptest.NewRequest(http.MethodGet, "/api/sandbox_status?chatId=chat-1", nil)
	readRec := httptest.NewRecorder()
	handleSandboxStatus()(readRec, readReq)

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
	handleSandbox()(writeRec, writeReq)

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

	var stored string
	if err := cronDB.QueryRow(`SELECT sandbox_exe FROM cli_sandbox_state WHERE chat_id = ?`, "chat-1").Scan(&stored); err != nil {
		t.Fatalf("query sandbox state: %v", err)
	}
	if stored != sandboxstate.ModeFilePickNet {
		t.Fatalf("sandbox_exe = %q, want %q", stored, sandboxstate.ModeFilePickNet)
	}

	otherReq := httptest.NewRequest(http.MethodGet, "/api/sandbox_status?chatId=chat-2", nil)
	otherRec := httptest.NewRecorder()
	handleSandboxStatus()(otherRec, otherReq)

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
	handleSandbox()(disableRec, disableReq)

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
	if err := cronDB.QueryRow(`SELECT COUNT(1) FROM cli_sandbox_state WHERE chat_id = ?`, "chat-1").Scan(&count); err != nil {
		t.Fatalf("count sandbox state: %v", err)
	}
	if count != 0 {
		t.Fatalf("sandbox row count = %d, want 0", count)
	}
}

func TestIntegrationHandleSandboxStatusReadsWithoutWriting(t *testing.T) {
	tmp := t.TempDir()

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
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
	cronDB = db
	defer func() { cronDB = nil }()
	oldPrimeSandbox := integrationPrimeSandboxModeFn
	integrationPrimeSandboxModeFn = func(string) (string, error) { return "", nil }
	defer func() { integrationPrimeSandboxModeFn = oldPrimeSandbox }()

	writeReq := httptest.NewRequest(http.MethodPost, "/api/sandbox=filepick?agentId=agent-a&chatId=chat-1", nil)
	writeRec := httptest.NewRecorder()
	handleSandbox()(writeRec, writeReq)

	statusReq := httptest.NewRequest(http.MethodGet, "/api/sandbox_status?agentId=agent-b&chatId=chat-1", nil)
	statusRec := httptest.NewRecorder()
	handleSandboxStatus()(statusRec, statusReq)

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
	if payload.Status != 0 || payload.AgentID != "agent-b" || payload.ChatID != "chat-1" || payload.Sandbox != sandboxstate.ModeFilePick || !payload.Recorded || payload.UpdatedAt == "" {
		t.Fatalf("unexpected sandbox_status payload: %+v", payload)
	}

	var stored string
	if err := cronDB.QueryRow(`SELECT sandbox_exe FROM cli_sandbox_state WHERE chat_id = ?`, "chat-1").Scan(&stored); err != nil {
		t.Fatalf("query sandbox state: %v", err)
	}
	if stored != sandboxstate.ModeFilePick {
		t.Fatalf("sandbox_exe = %q, want %q", stored, sandboxstate.ModeFilePick)
	}
}

func TestIntegrationHandleSandboxReturnsPrimeError(t *testing.T) {
	tmp := t.TempDir()

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
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
	cronDB = db
	defer func() { cronDB = nil }()

	oldPrimeSandbox := integrationPrimeSandboxModeFn
	integrationPrimeSandboxModeFn = func(mode string) (string, error) {
		if mode != sandboxstate.ModeFilePick {
			t.Fatalf("mode = %q, want %q", mode, sandboxstate.ModeFilePick)
		}
		return "", errors.New("picker unavailable")
	}
	defer func() { integrationPrimeSandboxModeFn = oldPrimeSandbox }()

	req := httptest.NewRequest(http.MethodPost, "/api/sandbox=filepick?agentId=agent-a&chatId=chat-1", nil)
	rec := httptest.NewRecorder()
	handleSandbox()(rec, req)

	var payload struct {
		Status  int    `json:"status"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Status != 1 || payload.Content != "picker unavailable" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if err := sandboxstate.EnsureSchema(cronDB); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	var count int
	if err := cronDB.QueryRow(`SELECT COUNT(1) FROM cli_sandbox_state WHERE chat_id = ?`, "chat-1").Scan(&count); err != nil {
		t.Fatalf("count sandbox state: %v", err)
	}
	if count != 0 {
		t.Fatalf("sandbox row count = %d, want 0", count)
	}
}

func TestIntegrationHandleSandboxUsesManualDirectoryWhitelist(t *testing.T) {
	tmp := t.TempDir()

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
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
	cronDB = db
	defer func() { cronDB = nil }()

	oldPrimeSandboxDir := integrationPrimeSandboxDirectoryFn
	integrationPrimeSandboxDirectoryFn = func(mode, dir string) (string, error) {
		if mode != sandboxstate.ModeFilePick {
			t.Fatalf("mode = %q, want %q", mode, sandboxstate.ModeFilePick)
		}
		if dir != "/Users/demo/Desktop" {
			t.Fatalf("dir = %q, want /Users/demo/Desktop", dir)
		}
		return dir, nil
	}
	defer func() { integrationPrimeSandboxDirectoryFn = oldPrimeSandboxDir }()

	req := httptest.NewRequest(http.MethodPost, "/api/sandbox=filepick?agentId=agent-a&chatId=chat-1&dir=%2FUsers%2Fdemo%2FDesktop", nil)
	rec := httptest.NewRecorder()
	handleSandbox()(rec, req)

	var payload struct {
		Status     int    `json:"status"`
		Sandbox    string `json:"sandbox"`
		AllowedDir string `json:"allowedDir"`
		Recorded   bool   `json:"recorded"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Status != 0 || payload.Sandbox != sandboxstate.ModeFilePick || payload.AllowedDir != "/Users/demo/Desktop" || !payload.Recorded {
		t.Fatalf("unexpected payload: %+v", payload)
	}

	var storedDir string
	if err := cronDB.QueryRow(`SELECT allowed_dir FROM cli_sandbox_state WHERE chat_id = ?`, "chat-1").Scan(&storedDir); err != nil {
		t.Fatalf("query sandbox allowed_dir: %v", err)
	}
	if storedDir != "/Users/demo/Desktop" {
		t.Fatalf("allowed_dir = %q, want /Users/demo/Desktop", storedDir)
	}
}

func TestPrimeIntegrationSandboxModeUsesHelperOutsideBundleLayout(t *testing.T) {
	tmp := t.TempDir()
	argsPath := filepath.Join(tmp, "sandbox-args.txt")
	envPath := filepath.Join(tmp, "sandbox-env.txt")
	helperPath := filepath.Join(tmp, "CLI_SANDBOX")
	helperScript := "#!/bin/sh\n" +
		"printf '%s\\n' \"$@\" > \"" + argsPath + "\"\n" +
		"printf '%s\\n' \"$CLI_SANDBOX_FORCE_PICK\" > \"" + envPath + "\"\n"
	if err := os.WriteFile(helperPath, []byte(helperScript), 0o755); err != nil {
		t.Fatalf("write helper: %v", err)
	}

	originalSandboxPathFn := integrationSandboxCommandPathFn
	integrationSandboxCommandPathFn = func(mode string) (string, error) {
		if mode != sandboxstate.ModeFilePick {
			t.Fatalf("mode = %q, want %q", mode, sandboxstate.ModeFilePick)
		}
		return helperPath, nil
	}
	defer func() { integrationSandboxCommandPathFn = originalSandboxPathFn }()

	if _, err := primeIntegrationSandboxMode(sandboxstate.ModeFilePick); err != nil {
		t.Fatalf("primeIntegrationSandboxMode: %v", err)
	}

	argsData, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	if strings.TrimSpace(string(argsData)) != "" {
		t.Fatalf("args = %q, want empty for pick-only prime", strings.TrimSpace(string(argsData)))
	}

	envData, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read env: %v", err)
	}
	if strings.TrimSpace(string(envData)) != "1" {
		t.Fatalf("CLI_SANDBOX_FORCE_PICK = %q, want 1", strings.TrimSpace(string(envData)))
	}
}

func TestPrimeIntegrationSandboxDirectoryUsesHelperOutsideBundleLayout(t *testing.T) {
	tmp := t.TempDir()
	argsPath := filepath.Join(tmp, "sandbox-args.txt")
	helperPath := filepath.Join(tmp, "CLI_SANDBOX")
	helperScript := "#!/bin/sh\n" +
		"printf '%s\\n' \"$@\" > \"" + argsPath + "\"\n"
	if err := os.WriteFile(helperPath, []byte(helperScript), 0o755); err != nil {
		t.Fatalf("write helper: %v", err)
	}

	originalSandboxPathFn := integrationSandboxCommandPathFn
	integrationSandboxCommandPathFn = func(mode string) (string, error) {
		if mode != sandboxstate.ModeFilePickNet {
			t.Fatalf("mode = %q, want %q", mode, sandboxstate.ModeFilePickNet)
		}
		return helperPath, nil
	}
	defer func() { integrationSandboxCommandPathFn = originalSandboxPathFn }()

	allowedDir, err := primeIntegrationSandboxDirectory(sandboxstate.ModeFilePickNet, "/tmp/workspace")
	if err != nil {
		t.Fatalf("primeIntegrationSandboxDirectory: %v", err)
	}
	if allowedDir != "/tmp/workspace" {
		t.Fatalf("allowedDir = %q, want /tmp/workspace", allowedDir)
	}

	argsData, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	gotArgs := strings.Split(strings.TrimSpace(string(argsData)), "\n")
	wantArgs := []string{"--allowed-dir", "/tmp/workspace"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
}

func TestIntegrationHandleSandboxRejectsDirForNetMode(t *testing.T) {
	tmp := t.TempDir()

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
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
	cronDB = db
	defer func() { cronDB = nil }()

	req := httptest.NewRequest(http.MethodPost, "/api/sandbox=net?agentId=agent-a&chatId=chat-1&dir=%2FUsers%2Fdemo%2FDesktop", nil)
	rec := httptest.NewRecorder()
	handleSandbox()(rec, req)

	var payload struct {
		Status  int    `json:"status"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Status != 1 || !strings.Contains(payload.Content, "does not support manual directory whitelist") {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestIntegrationHandleCmdUsesSandboxHelperForEnabledSession(t *testing.T) {
	tmp := t.TempDir()
	flushAgentCache()

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
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
	cronDB = db
	defer func() { cronDB = nil }()

	if _, err := sandboxstate.SetWithDirectory(cronDB, "agent-a", "chat-1", sandboxstate.ModeFilePickNet, "/Users/demo/Documents/photos"); err != nil {
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
		"printf 'sandbox-executed\\n'\n"
	if err := os.WriteFile(helperPath, []byte(helperScript), 0o755); err != nil {
		t.Fatalf("write helper: %v", err)
	}

	originalSandboxPathFn := integrationSandboxCommandPathFn
	integrationSandboxCommandPathFn = func(mode string) (string, error) {
		if mode != sandboxstate.ModeFilePickNet {
			t.Fatalf("mode = %q, want %q", mode, sandboxstate.ModeFilePickNet)
		}
		return helperPath, nil
	}
	defer func() { integrationSandboxCommandPathFn = originalSandboxPathFn }()

	cfg := &Config{
		AgentDir:     agentRoot,
		Device:       "dev",
		AgentCacheMs: 120000,
	}

	body := `{"agentId":"agent-a","chatId":"chat-1","cmd":"ls -la /Users/shenjiawei/Downloads","timeout":4321}`
	req := httptest.NewRequest(http.MethodPost, "/api/cmd", strings.NewReader(body))
	req.RemoteAddr = "127.0.0.1:34567"
	req.Host = "127.0.0.1:8080"
	rec := httptest.NewRecorder()
	handleCmd(cfg)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload CmdResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Status != 0 {
		t.Fatalf("payload.Status = %d, body = %+v", payload.Status, payload)
	}
	if payload.Output != "sandbox-executed\n" {
		t.Fatalf("payload.Output = %q, want sandbox-executed", payload.Output)
	}

	argsData, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	gotArgs := strings.Split(strings.TrimSpace(string(argsData)), "\n")
	wantPrefix := []string{"--cmd", "ls -la /Users/shenjiawei/Downloads", "--allowed-dir", "/Users/demo/Documents/photos", "--shell", sharedutil.DefaultExecShell(), "--timeout", "4321"}
	if !reflect.DeepEqual(gotArgs, wantPrefix) {
		t.Fatalf("sandbox args = %#v, want %#v", gotArgs, wantPrefix)
	}
}

func TestExecuteTaskUsesSandboxHelperForEnabledSession(t *testing.T) {
	tmp := t.TempDir()

	db, err := sql.Open("sqlite", filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	cronDB = db
	defer func() { cronDB = nil }()

	if _, err := sandboxstate.SetWithDirectory(cronDB, "agent-a", "chat-1", sandboxstate.ModeFilePickNet, "/Users/demo/Documents/code"); err != nil {
		t.Fatalf("enable sandbox: %v", err)
	}

	argsPath := filepath.Join(tmp, "sandbox-args.txt")
	helperPath := filepath.Join(tmp, "CLI_SANDBOX")
	helperScript := "#!/bin/sh\n" +
		"printf '%s\\n' \"$@\" > \"" + argsPath + "\"\n" +
		"printf 'sandbox-cli-get\\n'\n"
	if err := os.WriteFile(helperPath, []byte(helperScript), 0o755); err != nil {
		t.Fatalf("write helper: %v", err)
	}

	originalSandboxPathFn := integrationSandboxCommandPathFn
	integrationSandboxCommandPathFn = func(mode string) (string, error) {
		if mode != sandboxstate.ModeFilePickNet {
			t.Fatalf("mode = %q, want %q", mode, sandboxstate.ModeFilePickNet)
		}
		return helperPath, nil
	}
	defer func() { integrationSandboxCommandPathFn = originalSandboxPathFn }()

	result := executeTask(&TaskContent{
		Timeout: 3210,
		Suffix:  "cmd",
		Type:    "cmd",
		Tid:     "task-1",
		Cmd:     "ls -la /Users/shenjiawei/Downloads",
		AgentId: "agent-a",
		Chat:    "chat-1",
	}, sharedutil.DefaultExecShell())

	if result == nil || result.Status != 0 {
		t.Fatalf("result = %#v", result)
	}

	rawOutput, err := base64.StdEncoding.DecodeString(result.Cmd)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	decoded, err := gzip.NewReader(bytes.NewReader(rawOutput))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	output, err := io.ReadAll(decoded)
	if err != nil {
		t.Fatalf("read gzip: %v", err)
	}
	_ = decoded.Close()
	if string(output) != "sandbox-cli-get\n" {
		t.Fatalf("output = %q, want sandbox-cli-get", string(output))
	}

	argsData, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	gotArgs := strings.Split(strings.TrimSpace(string(argsData)), "\n")
	wantPrefix := []string{"--cmd", "ls -la /Users/shenjiawei/Downloads", "--allowed-dir", "/Users/demo/Documents/code", "--shell", sharedutil.DefaultExecShell(), "--timeout", "3210"}
	if !reflect.DeepEqual(gotArgs, wantPrefix) {
		t.Fatalf("sandbox args = %#v, want %#v", gotArgs, wantPrefix)
	}
}

func TestResolveIntegrationSandboxCommandPathFromLinuxHelpersDir(t *testing.T) {
	root := t.TempDir()
	execPath := filepath.Join(root, "linux-release", "integration")
	sandboxExec := filepath.Join(root, "linux-release", "helpers", sandboxstate.ModeFilePick, "CLI_SANDBOX")

	if err := os.MkdirAll(filepath.Dir(execPath), 0o755); err != nil {
		t.Fatalf("mkdir exec dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(sandboxExec), 0o755); err != nil {
		t.Fatalf("mkdir sandbox dir: %v", err)
	}
	if err := os.WriteFile(execPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write exec: %v", err)
	}
	if err := os.WriteFile(sandboxExec, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write sandbox exec: %v", err)
	}

	originalExecutable := integrationExecutableFn
	integrationExecutableFn = func() (string, error) { return execPath, nil }
	defer func() { integrationExecutableFn = originalExecutable }()

	got, err := resolveIntegrationSandboxCommandPath(sandboxstate.ModeFilePick)
	if err != nil {
		t.Fatalf("resolveIntegrationSandboxCommandPath: %v", err)
	}
	if got != sandboxExec {
		t.Fatalf("got %q, want %q", got, sandboxExec)
	}
}

func TestExecuteTaskSkipsSandboxHelperForExemptedTask(t *testing.T) {
	tmp := t.TempDir()

	db, err := sql.Open("sqlite", filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	cronDB = db
	defer func() { cronDB = nil }()

	if _, err := sandboxstate.Set(cronDB, "agent-a", "chat-1", sandboxstate.ModeFilePickNet); err != nil {
		t.Fatalf("enable sandbox: %v", err)
	}

	argsPath := filepath.Join(tmp, "sandbox-args.txt")
	helperPath := filepath.Join(tmp, "CLI_SANDBOX")
	helperScript := "#!/bin/sh\n" +
		"printf '%s\\n' \"$@\" > \"" + argsPath + "\"\n" +
		"printf 'sandbox-cli-get\\n'\n"
	if err := os.WriteFile(helperPath, []byte(helperScript), 0o755); err != nil {
		t.Fatalf("write helper: %v", err)
	}

	originalSandboxPathFn := integrationSandboxCommandPathFn
	integrationSandboxCommandPathFn = func(mode string) (string, error) {
		return helperPath, nil
	}
	defer func() { integrationSandboxCommandPathFn = originalSandboxPathFn }()

	task := &TaskContent{
		Timeout: 3210,
		Suffix:  "cmd",
		Type:    "cmd",
		Tid:     "task-exempted",
		Cmd:     "printf direct-integration",
		AgentId: "agent-a",
		Chat:    "chat-1",
	}
	task.SubOps.Exempted = true

	result := executeTask(task, sharedutil.DefaultExecShell())
	if result == nil || result.Status != 0 {
		t.Fatalf("result = %#v", result)
	}

	rawOutput, err := base64.StdEncoding.DecodeString(result.Cmd)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	decoded, err := gzip.NewReader(bytes.NewReader(rawOutput))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	output, err := io.ReadAll(decoded)
	if err != nil {
		t.Fatalf("read gzip: %v", err)
	}
	_ = decoded.Close()
	if string(output) != "direct-integration" {
		t.Fatalf("output = %q, want direct-integration", string(output))
	}
	if _, err := os.Stat(argsPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("sandbox helper should not run, stat err=%v", err)
	}
}

func TestExecuteTaskSkipsSandboxHelperForInternalTokenCommand(t *testing.T) {
	tmp := t.TempDir()

	db, err := sql.Open("sqlite", filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	cronDB = db
	defer func() { cronDB = nil }()

	if _, err := sandboxstate.Set(cronDB, "agent-a", "chat-1", sandboxstate.ModeFilePickNet); err != nil {
		t.Fatalf("enable sandbox: %v", err)
	}

	originalExecutable := integrationExecutableFn
	integrationExecutableFn = func() (string, error) {
		return "/Applications/DeepRight.app/Contents/MacOS/integration", nil
	}
	defer func() { integrationExecutableFn = originalExecutable }()

	originalSandboxPathFn := integrationSandboxCommandPathFn
	integrationSandboxCommandPathFn = func(mode string) (string, error) {
		return "", errors.New("sandbox helper should not run")
	}
	defer func() { integrationSandboxCommandPathFn = originalSandboxPathFn }()

	result := executeTask(&TaskContent{
		Timeout: 5000,
		Suffix:  "cmd",
		Type:    "cmd",
		Tid:     "task-token",
		Cmd:     "/Applications/DeepRight.app/Contents/MacOS/integration token --agentId agent-a --function 对话 --thinking 1 --cache 2 --input 3 --total 4 --model demo",
		AgentId: "agent-a",
		Chat:    "chat-1",
	}, sharedutil.DefaultExecShell())
	if result == nil || result.Status != 0 {
		t.Fatalf("result = %#v", result)
	}
	rawOutput, err := base64.StdEncoding.DecodeString(result.Cmd)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	decoded, err := gzip.NewReader(bytes.NewReader(rawOutput))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	output, err := io.ReadAll(decoded)
	if err != nil {
		t.Fatalf("read gzip: %v", err)
	}
	_ = decoded.Close()
	if !strings.Contains(string(output), `"status":0`) {
		t.Fatalf("output = %q, want token json", string(output))
	}
}

func TestExecuteTaskSkipsSandboxHelperForInternalRuntimePaths(t *testing.T) {
	tmp := t.TempDir()

	db, err := sql.Open("sqlite", filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	cronDB = db
	defer func() { cronDB = nil }()

	if _, err := sandboxstate.Set(cronDB, "agent-a", "chat-1", sandboxstate.ModeFilePick); err != nil {
		t.Fatalf("enable sandbox: %v", err)
	}

	runtimeRoot := integrationTestMacRuntimeBaseDir(tmp)
	targetFile := filepath.Join(runtimeRoot, "agent", "agent-a", "USER.md")
	if err := os.MkdirAll(filepath.Dir(targetFile), 0o755); err != nil {
		t.Fatalf("mkdir runtime dir: %v", err)
	}
	if err := os.WriteFile(targetFile, []byte("internal-user"), 0o644); err != nil {
		t.Fatalf("write USER.md: %v", err)
	}

	oldHome := integrationUserHomeFn
	integrationUserHomeFn = func() (string, error) { return tmp, nil }
	defer func() { integrationUserHomeFn = oldHome }()

	originalSandboxPathFn := integrationSandboxCommandPathFn
	integrationSandboxCommandPathFn = func(mode string) (string, error) {
		return "", errors.New("sandbox helper should not run")
	}
	defer func() { integrationSandboxCommandPathFn = originalSandboxPathFn }()

	task := &TaskContent{
		Timeout: 5000,
		Suffix:  "md",
		Type:    "fun",
		Tid:     "task-runtime-read",
		Cmd:     "cat " + strconv.Quote(targetFile),
		AgentId: "agent-a",
		Chat:    "chat-1",
	}
	task.SubOps.R = []string{targetFile}

	result := executeTask(task, sharedutil.DefaultExecShell())
	if result == nil || result.Status != 0 {
		t.Fatalf("result = %#v", result)
	}
	rawOutput, err := base64.StdEncoding.DecodeString(result.Cmd)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	decoded, err := gzip.NewReader(bytes.NewReader(rawOutput))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	output, err := io.ReadAll(decoded)
	if err != nil {
		t.Fatalf("read gzip: %v", err)
	}
	_ = decoded.Close()
	if strings.TrimSpace(string(output)) != "internal-user" {
		t.Fatalf("output = %q, want internal-user", string(output))
	}
}

func TestIntegrationSandboxCLIReadsAndWritesSessionState(t *testing.T) {
	tmp := t.TempDir()
	homeDir := filepath.Join(tmp, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	originalHomeFn := integrationUserHomeFn
	integrationUserHomeFn = func() (string, error) { return homeDir, nil }
	defer func() { integrationUserHomeFn = originalHomeFn }()

	stdout := captureIntegrationStdout(t, func() {
		runIntegrationSandboxCLI([]string{"--chatId", "chat-1"})
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

	stdout = captureIntegrationStdout(t, func() {
		runIntegrationSandboxCLI([]string{"--agentId", "agent-a", "--chatId", "chat-1", "--sandbox", "net"})
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

	db, err := sql.Open("sqlite", resolveIntegrationDBPath())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	var stored string
	if err := db.QueryRow(`SELECT sandbox_exe FROM cli_sandbox_state WHERE chat_id = ?`, "chat-1").Scan(&stored); err != nil {
		t.Fatalf("query sandbox state: %v", err)
	}
	if stored != sandboxstate.ModeNet {
		t.Fatalf("sandbox_exe = %q, want %q", stored, sandboxstate.ModeNet)
	}
}

func TestIntegrationHandleTokenNormalizesLegacyModelAliases(t *testing.T) {
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
	cronDB = db
	defer func() { cronDB = nil }()

	if err := ensureTokenStore(cronDB); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if _, err := cronDB.Exec(`INSERT INTO token_store (model, token, __url, __model, __model_fast, __model_thinking, __model_multi_input, __model_multi_output, updated_at) VALUES (?,?,?,?,?,?,?,?,?)`, "aiohright", "legacy-token", "https://legacy.example", "base-a", "fast-a", "think-a", "multi-in-a", "multi-out-a", time.Now().Format(time.RFC3339)); err != nil {
		t.Fatalf("seed token_store: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/token", nil)
	req.RemoteAddr = "127.0.0.1:34567"
	req.Host = "127.0.0.1:8080"
	rec := httptest.NewRecorder()
	handleToken()(rec, req)

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
	if payload.Models["deepright"].Token != "legacy-token" {
		t.Fatalf("deepright token = %q", payload.Models["deepright"].Token)
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
		t.Fatalf("legacy alias leaked: %+v", payload.Models)
	}
}

func TestIntegrationHandleTokenMasksRemoteRead(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	db, err := sql.Open("sqlite", filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	cronDB = db
	defer func() { cronDB = nil }()

	if err := writeTokenStoreConfigs(cronDB, map[string]tokenConfig{
		"openai": {
			Token:            "Bearer remote-secret",
			BaseURL:          "https://api.openai.com/v1",
			ModelBase:        "gpt-4.1",
			ModelFast:        "gpt-4.1-mini",
			ModelThinking:    "o4-mini",
			ModelMultiInput:  "gpt-4.1",
			ModelMultiOutput: "gpt-image-1",
		},
	}); err != nil {
		t.Fatalf("writeTokenStoreConfigs: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/token", nil)
	req.RemoteAddr = "203.0.113.10:4567"
	rec := httptest.NewRecorder()
	handleToken()(rec, req)

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
	cfg, ok := payload.Models["openai"]
	if !ok {
		t.Fatalf("openai missing: %+v", payload.Models)
	}
	if cfg.Token != maskedTokenValue {
		t.Fatalf("remote token = %q, want %q", cfg.Token, maskedTokenValue)
	}
	if cfg.BaseURL != "https://api.openai.com/v1" || cfg.ModelBase != "gpt-4.1" || cfg.ModelFast != "gpt-4.1-mini" || cfg.ModelThinking != "o4-mini" || cfg.ModelMultiInput != "gpt-4.1" || cfg.ModelMultiOutput != "gpt-image-1" {
		t.Fatalf("masked config should keep model metadata: %+v", cfg)
	}
}

func TestIntegrationHandleTokenMasksRemoteHostAliasEvenWhenRemoteAddrIsLoopback(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	db, err := sql.Open("sqlite", filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	cronDB = db
	defer func() { cronDB = nil }()

	if err := writeTokenStoreConfigs(cronDB, map[string]tokenConfig{
		"openai": {Token: "Bearer remote-secret"},
	}); err != nil {
		t.Fatalf("writeTokenStoreConfigs: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/token", nil)
	req.RemoteAddr = "127.0.0.1:34567"
	req.Host = "remote.example.com:8080"
	rec := httptest.NewRecorder()
	handleToken()(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Status int                    `json:"status"`
		Models map[string]tokenConfig `json:"models"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v body=%s", err, rec.Body.String())
	}
	if payload.Models["openai"].Token != maskedTokenValue {
		t.Fatalf("token = %q, want %q", payload.Models["openai"].Token, maskedTokenValue)
	}
}

func TestIntegrationHandleTokenRenamedModelDeletesPreviousRecord(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	db, err := sql.Open("sqlite", filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	cronDB = db
	defer func() { cronDB = nil }()

	if err := ensureTokenStore(cronDB); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if _, err := cronDB.Exec(`INSERT INTO token_store (model, token, __url, __model, __model_fast, __model_thinking, __model_multi_input, __model_multi_output, updated_at) VALUES (?,?,?,?,?,?,?,?,?)`, "openai", "old-token", "https://api.openai.com/v1/chat/completions", "gpt-old", "gpt-fast-old", "gpt-think-old", "gpt-vision-old", "", time.Now().Format(time.RFC3339)); err != nil {
		t.Fatalf("seed token_store: %v", err)
	}

	reqBody := `{"models":{"deepright":{"token":"new-token"}},"removedModels":["openai"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/token", strings.NewReader(reqBody))
	req.RemoteAddr = "127.0.0.1:34567"
	req.Host = "127.0.0.1:8080"
	rec := httptest.NewRecorder()
	handleToken()(rec, req)

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
	if err := cronDB.QueryRow(`SELECT COUNT(*) FROM token_store WHERE model = ?`, "openai").Scan(&count); err != nil {
		t.Fatalf("count openai: %v", err)
	}
	if count != 0 {
		t.Fatalf("openai count = %d, want 0", count)
	}
	if err := cronDB.QueryRow(`SELECT COUNT(*) FROM token_store WHERE model = ?`, "deepright").Scan(&count); err != nil {
		t.Fatalf("count deepright: %v", err)
	}
	if count != 1 {
		t.Fatalf("deepright count = %d, want 1", count)
	}

	rows, err := cronDB.Query(`SELECT model, action FROM proxy_agent_provider_log ORDER BY id`)
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

func TestIntegrationHandleTokenRemotePostMasksResponseButKeepsStoredToken(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	db, err := sql.Open("sqlite", filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	cronDB = db
	defer func() { cronDB = nil }()

	if err := writeTokenStoreConfigs(cronDB, map[string]tokenConfig{
		"openai": {
			Token:            "Bearer old-remote-token",
			BaseURL:          "https://api.openai.com/v1",
			ModelBase:        "gpt-4.1",
			ModelFast:        "gpt-4.1-mini",
			ModelThinking:    "o4-mini",
			ModelMultiInput:  "gpt-4.1",
			ModelMultiOutput: "gpt-image-1",
		},
	}); err != nil {
		t.Fatalf("writeTokenStoreConfigs: %v", err)
	}

	reqBody := `{"models":{"openai":{"token":"Bearer new-remote-token","__url":"https://api.openai.com/v1"}}}`
	req := httptest.NewRequest(http.MethodPost, "/api/token", strings.NewReader(reqBody))
	req.RemoteAddr = "203.0.113.10:4567"
	rec := httptest.NewRecorder()
	handleToken()(rec, req)

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
	cfg, ok := payload.Models["openai"]
	if !ok {
		t.Fatalf("openai missing: %+v", payload.Models)
	}
	if cfg.Token != maskedTokenValue {
		t.Fatalf("post response token = %q, want %q", cfg.Token, maskedTokenValue)
	}
	if cfg.BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("post response __url = %q", cfg.BaseURL)
	}

	var stored string
	if err := cronDB.QueryRow(`SELECT token FROM token_store WHERE model = ?`, "openai").Scan(&stored); err != nil {
		t.Fatalf("query token_store: %v", err)
	}
	if stored != "Bearer new-remote-token" {
		t.Fatalf("stored token = %q, want Bearer new-remote-token", stored)
	}
}

func TestIntegrationHandleTokenRemotePostMaskedTokenPreservesStoredSecret(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	db, err := sql.Open("sqlite", filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	cronDB = db
	defer func() { cronDB = nil }()

	if err := writeTokenStoreConfigs(cronDB, map[string]tokenConfig{
		"openai": {
			Token:            "Bearer persisted-secret",
			BaseURL:          "https://api.openai.com/v1",
			ModelBase:        "gpt-4.1",
			ModelFast:        "gpt-4.1-mini",
			ModelThinking:    "o4-mini",
			ModelMultiInput:  "gpt-4.1",
			ModelMultiOutput: "gpt-image-1",
		},
	}); err != nil {
		t.Fatalf("writeTokenStoreConfigs: %v", err)
	}

	reqBody := `{"models":{"openai":{"token":"**********","__url":"https://gateway.example.com/v1","__model":"gpt-4.1-custom"}}}`
	req := httptest.NewRequest(http.MethodPost, "/api/token", strings.NewReader(reqBody))
	req.RemoteAddr = "203.0.113.10:4567"
	rec := httptest.NewRecorder()
	handleToken()(rec, req)

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
	cfg, ok := payload.Models["openai"]
	if !ok {
		t.Fatalf("openai missing: %+v", payload.Models)
	}
	if cfg.Token != maskedTokenValue {
		t.Fatalf("post response token = %q, want %q", cfg.Token, maskedTokenValue)
	}
	if cfg.BaseURL != "https://gateway.example.com/v1" || cfg.ModelBase != "gpt-4.1-custom" {
		t.Fatalf("post response should keep updated config: %+v", cfg)
	}

	var stored tokenConfig
	if err := cronDB.QueryRow(`SELECT token, __url, __model, __model_fast, __model_thinking, __model_multi_input, __model_multi_output, __custom_cleared FROM token_store WHERE model = ?`, "openai").
		Scan(&stored.Token, &stored.BaseURL, &stored.ModelBase, &stored.ModelFast, &stored.ModelThinking, &stored.ModelMultiInput, &stored.ModelMultiOutput, &stored.CustomCleared); err != nil {
		t.Fatalf("query token_store: %v", err)
	}
	if stored.Token != "Bearer persisted-secret" {
		t.Fatalf("stored token = %q, want Bearer persisted-secret", stored.Token)
	}
	if stored.BaseURL != "https://gateway.example.com/v1" || stored.ModelBase != "gpt-4.1-custom" {
		t.Fatalf("stored config should keep updated metadata: %+v", stored)
	}
}

func TestIntegrationHandleTokenRemotePostRejectsNewModel(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	db, err := sql.Open("sqlite", filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	cronDB = db
	defer func() { cronDB = nil }()

	reqBody := `{"models":{"openai":{"token":"Bearer new-remote-token","__url":"https://api.openai.com/v1"}}}`
	req := httptest.NewRequest(http.MethodPost, "/api/token", strings.NewReader(reqBody))
	req.RemoteAddr = "203.0.113.10:4567"
	rec := httptest.NewRecorder()
	handleToken()(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "only localhost/127.0.0.1/::1 requests can add models") {
		t.Fatalf("body = %s", rec.Body.String())
	}

	var count int
	if err := cronDB.QueryRow(`SELECT COUNT(*) FROM token_store`).Scan(&count); err != nil {
		t.Fatalf("count token_store: %v", err)
	}
	if count != 0 {
		t.Fatalf("token_store count = %d, want 0", count)
	}
}

func TestIntegrationHandleTokenDeleteModelRemovesLegacyCaseVariants(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	db, err := sql.Open("sqlite", filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	cronDB = db
	defer func() { cronDB = nil }()

	if err := ensureTokenStore(cronDB); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	now := time.Now().Format(time.RFC3339)
	if _, err := cronDB.Exec(`INSERT INTO token_store (model, token, updated_at) VALUES (?,?,?)`, "OpenAI", "legacy-token", now); err != nil {
		t.Fatalf("seed legacy token_store: %v", err)
	}
	if _, err := cronDB.Exec(`INSERT INTO token_store (model, token, updated_at) VALUES (?,?,?)`, "openai", "canonical-token", now); err != nil {
		t.Fatalf("seed canonical token_store: %v", err)
	}

	reqBody := `{"action":"delete_model","model":"openai"}`
	req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(reqBody))
	rec := httptest.NewRecorder()
	handleSwarm(&Config{})(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var count int
	if err := cronDB.QueryRow(`SELECT COUNT(*) FROM token_store WHERE model IN ('OpenAI', 'openai')`).Scan(&count); err != nil {
		t.Fatalf("count token_store variants: %v", err)
	}
	if count != 0 {
		t.Fatalf("token_store variant count = %d, want 0", count)
	}
}

func TestIntegrationHandleSwarmDeletesModelConfig(t *testing.T) {
	tmp := t.TempDir()

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
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
	cronDB = db
	defer func() { cronDB = nil }()

	if err := ensureTokenStore(cronDB); err != nil {
		t.Fatalf("ensureTokenStore: %v", err)
	}
	if _, err := cronDB.Exec(`INSERT INTO token_store (model, token, updated_at) VALUES (?,?,?)`, "deepseek", "Bearer deepseek-token", time.Now().Format(time.RFC3339)); err != nil {
		t.Fatalf("seed token_store: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(`{"action":"delete_model","model":"deepseek"}`))
	rec := httptest.NewRecorder()
	handleSwarm(&Config{})(rec, req)

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
	if err := cronDB.QueryRow(`SELECT COUNT(*) FROM token_store WHERE model = ?`, "deepseek").Scan(&count); err != nil {
		t.Fatalf("count token_store: %v", err)
	}
	if count != 0 {
		t.Fatalf("token_store count = %d, want 0", count)
	}

	var action string
	if err := cronDB.QueryRow(`SELECT action FROM proxy_agent_provider_log WHERE model = ? ORDER BY id DESC LIMIT 1`, "deepseek").Scan(&action); err != nil {
		t.Fatalf("query provider log: %v", err)
	}
	if action != "delete" {
		t.Fatalf("provider log action = %q, want delete", action)
	}
}

func TestIntegrationTokenCLIWritesConsumeRecord(t *testing.T) {
	tmp := t.TempDir()

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	timestamp := time.Date(2026, 5, 27, 10, 0, 0, 0, time.Local).UnixMilli()
	stdout := captureIntegrationStdout(t, func() {
		runIntegrationTokenCLI([]string{
			"--agentId", "agent-a",
			"--model", "deepseek-chat",
			"--function", "cli/get",
			"--thinking", "12",
			"--input", "24",
			"--total", "36",
			"--cache", "6",
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
	if payload.Record.AgentID != "agent-a" || payload.Record.Model != "deepseek-chat" || payload.Record.Total != 36 {
		t.Fatalf("unexpected record: %+v", payload.Record)
	}

	db, err := sql.Open("sqlite", resolveIntegrationDBPath())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	result, err := tokenconsume.Query(db, "agent-a", timestamp-1)
	if err != nil {
		t.Fatalf("query consume log: %v", err)
	}
	if len(result.Details) != 1 {
		t.Fatalf("len(details) = %d, want 1", len(result.Details))
	}
	if result.Details[0].Cache != 6 {
		t.Fatalf("stored detail = %+v", result.Details[0])
	}
}

func TestIntegrationTokenCLIQueriesLatestConsumeRecords(t *testing.T) {
	tmp := t.TempDir()

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	oldHomeFn := integrationUserHomeFn
	integrationUserHomeFn = func() (string, error) { return tmp, nil }
	defer func() { integrationUserHomeFn = oldHomeFn }()

	dbPath := filepath.Join(integrationTestMacRuntimeBaseDir(tmp), "data")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatalf("mkdir runtime dir: %v", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

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

	stdout := captureIntegrationStdout(t, func() {
		runIntegrationTokenCLI([]string{"get", "--agentId", "agent-a", "--n", "2"})
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

func TestIntegrationHandleConsumeAggregatesByModel(t *testing.T) {
	tmp := t.TempDir()

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
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
	cronDB = db
	defer func() { cronDB = nil }()

	base := time.Date(2026, 5, 27, 8, 0, 0, 0, time.Local)
	records := []tokenconsume.Record{
		{Thinking: 10, Input: 20, Total: 30, Cache: 5, Model: "model-a", AgentID: "agent-a", Function: "cli/get", Timestamp: base.Add(1 * time.Minute).UnixMilli()},
		{Thinking: 1, Input: 2, Total: 3, Cache: 1, Model: "model-a", AgentID: "agent-a", Function: "cron", Timestamp: base.Add(2 * time.Minute).UnixMilli()},
		{Thinking: 4, Input: 5, Total: 9, Cache: 0, Model: "model-b", AgentID: "agent-a", Function: "cli/pub", Timestamp: base.Add(3 * time.Minute).UnixMilli()},
		{Thinking: 7, Input: 8, Total: 15, Cache: 2, Model: "model-c", AgentID: "agent-b", Function: "cli/get", Timestamp: base.Add(4 * time.Minute).UnixMilli()},
	}
	for _, record := range records {
		if _, err := tokenconsume.Insert(cronDB, record); err != nil {
			t.Fatalf("insert consume record: %v", err)
		}
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/consume?agentId=agent-a&starTime="+base.Format("20060102-150405")+"&closeTime="+base.Add(3*time.Minute).Format("20060102-150405")+"&limit=2",
		nil,
	)
	rec := httptest.NewRecorder()
	handleConsume()(rec, req)

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

func TestIntegrationHandleChatSessionLogReturnsDescendingRawRows(t *testing.T) {
	tmp := t.TempDir()

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
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
	cronDB = db
	defer func() { cronDB = nil }()

	if _, err := cronDB.Exec(`
		CREATE TABLE IF NOT EXISTS agent_message_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id TEXT NOT NULL DEFAULT '',
			chat_id TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			log_type INTEGER NOT NULL,
			created_at TEXT NOT NULL
		);
	`); err != nil {
		t.Fatalf("create agent_message_log: %v", err)
	}

	rows := []struct {
		agentID   string
		chatID    string
		content   string
		logType   int
		createdAt string
	}{
		{agentID: "agent-a", chatID: "chat-1", content: `{"messages":[{"role":"user","content":"hello"}]}`, logType: logTypeChatCompletionRequest, createdAt: "2026-05-27T08:01:00.000"},
		{agentID: "agent-a", chatID: "chat-1", content: "data: {\"choices\":[{\"delta\":{\"content\":\"world\"}}]}\n\n", logType: logTypeChatCompletionResponse, createdAt: "2026-05-27T08:02:00.000"},
		{agentID: "agent-a", chatID: "chat-1", content: `{"cmd":"pwd"}`, logType: logTypeCLIGet, createdAt: "2026-05-27T08:02:30.000"},
		{agentID: "agent-b", chatID: "chat-9", content: "data: {\"choices\":[{\"delta\":{\"content\":\"skip\"}}]}\n\n", logType: logTypeChatCompletionResponse, createdAt: "2026-05-27T08:03:00.000"},
	}
	for _, row := range rows {
		if _, err := cronDB.Exec(`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES (?,?,?,?,?)`,
			row.agentID, row.chatID, row.content, row.logType, row.createdAt); err != nil {
			t.Fatalf("insert agent_message_log: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/chat_session_log?chatId=chat-1&start=2026-05-27T08:01:00&close=2026-05-27T08:02:00&limit=5", nil)
	rec := httptest.NewRecorder()
	handleChatSessionLog()(rec, req)

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
	if payload.Records[0].LogType != logTypeChatCompletionResponse || payload.Records[0].CreatedAt != "2026-05-27T08:02:00.000" {
		t.Fatalf("records[0] = %+v", payload.Records[0])
	}
	if payload.Records[1].LogType != logTypeChatCompletionRequest || payload.Records[1].CreatedAt != "2026-05-27T08:01:00.000" {
		t.Fatalf("records[1] = %+v", payload.Records[1])
	}
	if payload.Records[0].Content != "data: {\"choices\":[{\"delta\":{\"content\":\"world\"}}]}\n\n" {
		t.Fatalf("response content mismatch: %q", payload.Records[0].Content)
	}
}

func TestIntegrationResolveRouterDisableFormValueIgnoresLegacySwarm(t *testing.T) {
	form := url.Values{"swarm": []string{"true"}}
	if got := resolveRouterDisableFormValue(form, true); !got {
		t.Fatalf("router_disable from legacy swarm = %v, want true", got)
	}

	form.Set("router_disable", "false")
	if got := resolveRouterDisableFormValue(form, true); got {
		t.Fatalf("router_disable explicit value = %v, want false", got)
	}
}

func TestIntegrationCronCreateRequestIgnoresLegacySwarm(t *testing.T) {
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

func TestIntegrationHandleSwarmWritesRouterDisableConfig(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPost, "/api/config?agentId=A", strings.NewReader(`{"description":"demo","thinking":true,"router_disable":false}`))
	rec := httptest.NewRecorder()
	handleSwarm(&Config{AgentDir: "agent", AgentCacheMs: 1})(rec, req)

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

func TestIntegrationHandleSwarmWritesMediaConfig(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPost, "/api/config?agentId=A", strings.NewReader(`{"description":"demo","thinking":true,"router_disable":false,"media":{"cover":"/tmp/a.png","duration":12}}`))
	rec := httptest.NewRecorder()
	handleSwarm(&Config{AgentDir: "agent", AgentCacheMs: 1})(rec, req)

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

func TestIntegrationTokenCLIOutputSingleModel(t *testing.T) {
	var stdout bytes.Buffer
	err := writeIntegrationTokenCLIOutput(&stdout, map[string]tokenConfig{
		"deepseek": {Token: "aaa", BaseURL: "https://api.example", ModelMultiOutput: "gpt-image-1"},
		"kimi":     {Token: "bbb"},
	}, "deepseek")
	if err != nil {
		t.Fatalf("writeIntegrationTokenCLIOutput: %v", err)
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

func TestIntegrationTokenCLIUsesRuntimeDBPath(t *testing.T) {
	tmp := t.TempDir()
	appDir := filepath.Join(tmp, "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("mkdir app dir: %v", err)
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	configDir := filepath.Join(tmp, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	startupConfigPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(startupConfigPath, []byte(fmt.Sprintf("{\"app-dir\":%q}\n", appDir)), 0o644); err != nil {
		t.Fatalf("write config/config.json: %v", err)
	}

	db, err := sql.Open("sqlite", filepath.Join(appDir, "data"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	if err := writeTokenStore(db, map[string]string{"deepseek": "aaa", "kimi": "bbb"}); err != nil {
		t.Fatalf("write token store: %v", err)
	}

	stdout := captureIntegrationStdout(t, func() {
		runIntegrationTokenCLI([]string{"--provider", "deepseek"})
	})

	var payload map[string]tokenConfig
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("decode output: %v; raw=%s", err, stdout)
	}
	if len(payload) != 1 || payload["deepseek"].Token != "aaa" {
		t.Fatalf("payload = %+v, want only deepseek", payload)
	}
}

func TestIntegrationKnowledgeUpdateTimeCLI(t *testing.T) {
	disableIntegrationManagedRuntimeForTest(t)
	tmp := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)
	defer func() { _ = knowledgecore.CloseSharedDB() }()

	stdout := captureIntegrationStdout(t, func() {
		runIntegrationKnowledgeCLI([]string{"update-time", "--timestamp", "1715337600000", "--agentId", "agent-a"})
	})
	if !strings.Contains(stdout, filepath.Join(tmp, "knowledge", "agent-a")) {
		t.Fatalf("stdout = %q, want knowledge path", stdout)
	}

	db, err := knowledgecore.OpenSharedDB(tmp)
	if err != nil {
		t.Fatalf("OpenSharedDB: %v", err)
	}
	lastUpdate, err := knowledgecore.GetLastUpdateForAgent(db, "agent-a")
	if err != nil {
		t.Fatalf("GetLastUpdateForAgent: %v", err)
	}
	if lastUpdate != 1715337600000 {
		t.Fatalf("lastUpdate = %d, want 1715337600000", lastUpdate)
	}
}

func TestHandleKnowledgeListsTreeAndServesFiles(t *testing.T) {
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
	if err := os.MkdirAll(filepath.Join(tmp, "config"), 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "config", "config.json"), []byte(fmt.Sprintf("{\"agent-dir\":%q}\n", agentRoot)), 0o644); err != nil {
		t.Fatalf("write startup config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmp, "knowledge", "docs"), 0o755); err != nil {
		t.Fatalf("mkdir knowledge docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "knowledge", "README.md"), []byte("hello knowledge\n"), 0o644); err != nil {
		t.Fatalf("write knowledge root file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "knowledge", "docs", "guide.txt"), []byte("guide body"), 0o644); err != nil {
		t.Fatalf("write knowledge child file: %v", err)
	}

	server := httptest.NewServer(handleKnowledge(&Config{AgentDir: agentRoot}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/knowledge")
	if err != nil {
		t.Fatalf("GET /knowledge failed: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read /knowledge body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /knowledge status = %d, body = %s", resp.StatusCode, string(body))
	}
	tree := string(body)
	if !strings.Contains(tree, "knowledge/") {
		t.Fatalf("tree missing root: %q", tree)
	}
	if !strings.Contains(tree, "README.md") {
		t.Fatalf("tree missing README.md: %q", tree)
	}
	if !strings.Contains(tree, "docs/") {
		t.Fatalf("tree missing docs/: %q", tree)
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "text/plain") {
		t.Fatalf("GET /knowledge content-type = %q", resp.Header.Get("Content-Type"))
	}

	resp, err = http.Get(server.URL + "/knowledge/docs/guide.txt")
	if err != nil {
		t.Fatalf("GET /knowledge/docs/guide.txt failed: %v", err)
	}
	body, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read /knowledge/docs/guide.txt body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /knowledge/docs/guide.txt status = %d, body = %s", resp.StatusCode, string(body))
	}
	if string(body) != "guide body" {
		t.Fatalf("GET /knowledge/docs/guide.txt body = %q", string(body))
	}
}

func TestHandleKnowledgeRejectsPathEscape(t *testing.T) {
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
	if err := os.MkdirAll(filepath.Join(tmp, "config"), 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "config", "config.json"), []byte(fmt.Sprintf("{\"agent-dir\":%q}\n", agentRoot)), 0o644); err != nil {
		t.Fatalf("write startup config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmp, "knowledge"), 0o755); err != nil {
		t.Fatalf("mkdir knowledge: %v", err)
	}

	server := httptest.NewServer(handleKnowledge(&Config{AgentDir: agentRoot}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/knowledge/../secret.txt")
	if err != nil {
		t.Fatalf("GET escape path failed: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read escape path body: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("GET escape path status = %d, body = %s", resp.StatusCode, string(body))
	}
	if !strings.Contains(string(body), "path escapes knowledge root") {
		t.Fatalf("GET escape path body = %q", string(body))
	}
}

func TestHandleKnowledgeLastUpdateReturnsFormattedTimestamp(t *testing.T) {
	disableIntegrationManagedRuntimeForTest(t)
	tmp := t.TempDir()
	appDir := filepath.Join(tmp, "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("mkdir app dir: %v", err)
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
	if err := os.MkdirAll(filepath.Join(tmp, "config"), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "config", "config.json"), []byte(fmt.Sprintf("{\"app-dir\":%q}\n", appDir)), 0o644); err != nil {
		t.Fatalf("write startup config: %v", err)
	}

	db, err := knowledgecore.OpenSharedDB(appDir)
	if err != nil {
		t.Fatalf("open knowledge db: %v", err)
	}
	ts := time.Date(2026, time.May, 11, 9, 45, 0, 0, time.Local).UnixMilli()
	if err := knowledgecore.SetLastUpdateForAgent(db, "agent-a", ts); err != nil {
		t.Fatalf("set knowledge last update: %v", err)
	}

	server := httptest.NewServer(handleKnowledgeLastUpdate())
	defer server.Close()

	resp, err := http.Get(server.URL + "/knowledge_lastUpdate?agentId=" + url.QueryEscape("agent-a"))
	if err != nil {
		t.Fatalf("GET /knowledge_lastUpdate failed: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read /knowledge_lastUpdate body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /knowledge_lastUpdate status = %d, body = %s", resp.StatusCode, string(body))
	}
	if got, want := string(body), "2026-05-11 09:45"; got != want {
		t.Fatalf("GET /knowledge_lastUpdate body = %q, want %q", got, want)
	}
}

func TestHandleKnowledgeLastUpdateRejectsNonGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/knowledge_lastUpdate", nil)
	w := httptest.NewRecorder()

	handleKnowledgeLastUpdate().ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleKnowledgePath(t *testing.T) {
	disableIntegrationManagedRuntimeForTest(t)
	tmp := t.TempDir()
	appDir := filepath.Join(tmp, "app")
	agentRoot := filepath.Join(tmp, "agent")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("mkdir app dir: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)
	if err := os.MkdirAll(filepath.Join(tmp, "config"), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "config", "config.json"), []byte(fmt.Sprintf("{\"app-dir\":%q,\"agent-dir\":%q}\n", appDir, agentRoot)), 0o644); err != nil {
		t.Fatalf("write startup config: %v", err)
	}
	server := httptest.NewServer(handleKnowledgePath(&Config{AgentDir: agentRoot}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/knowledge_path?agentId=" + url.QueryEscape("agent-a"))
	if err != nil {
		t.Fatalf("GET /knowledge_path failed: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read /knowledge_path body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /knowledge_path status = %d, body = %s", resp.StatusCode, string(body))
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
		t.Fatalf("GET /knowledge_path body = %q (resolved %q), want %q (resolved %q)", got, gotResolved, want, wantResolved)
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
	req := httptest.NewRequest(http.MethodPost, "/knowledge_path", nil)
	w := httptest.NewRecorder()

	handleKnowledgePath(&Config{AgentDir: t.TempDir()}).ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestKnowledgeLastUpdateModuleFormatsSharedRuntime(t *testing.T) {
	disableIntegrationManagedRuntimeForTest(t)
	tmp := t.TempDir()
	defer func() { _ = knowledgecore.CloseSharedDB() }()

	db, err := knowledgecore.OpenSharedDB(tmp)
	if err != nil {
		t.Fatalf("open knowledge db: %v", err)
	}
	ts := time.Date(2026, time.May, 16, 14, 21, 0, 0, time.Local).UnixMilli()
	if err := knowledgecore.SetLastUpdate(db, ts); err != nil {
		t.Fatalf("set knowledge last update: %v", err)
	}

	got, err := integrationknowledge.LoadLastUpdateText(tmp, "", time.Local)
	if err != nil {
		t.Fatalf("LoadLastUpdateText failed: %v", err)
	}
	if want := "2026-05-16 14:21"; got != want {
		t.Fatalf("LoadLastUpdateText = %q, want %q", got, want)
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

	store, err := skillscore.OpenWarningStore(filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("OpenWarningStore failed: %v", err)
	}
	if err := store.Sync([]skillscore.SkillWarning{{Path: "/tmp/demo/SKILL.md", Reason: "broken", Time: 123}}); err != nil {
		t.Fatalf("store.Sync failed: %v", err)
	}

	cfg := &Config{AgentDir: filepath.Join(tmp, "agent")}
	server := httptest.NewServer(http.HandlerFunc(handleSkillsWarning(cfg)))
	defer server.Close()

	resp, err := http.Get(server.URL + "/skills_warning")
	if err != nil {
		t.Fatalf("GET /skills_warning failed: %v", err)
	}
	defer resp.Body.Close()

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
	if len(payload.Data) != 1 || payload.Data[0].Reason != "broken" {
		t.Fatalf("payload data = %+v, want one broken warning", payload.Data)
	}
}

func TestHandleSkillsIncludesInternalSkills(t *testing.T) {
	agentDir, err := filepath.Abs("../proxy/iteration/20260419_4/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}
	setupSkillsConfig := func(t *testing.T) {
		t.Helper()
		appRoot := t.TempDir()
		if err := os.MkdirAll(filepath.Join(appRoot, "config"), 0o755); err != nil {
			t.Fatalf("mkdir config: %v", err)
		}
		if err := os.WriteFile(filepath.Join(appRoot, "config", "config.json"), []byte(`{"skills":["__internal_cron","__internal_demo"]}`), 0o644); err != nil {
			t.Fatalf("write config.json: %v", err)
		}
		originalExecutable := integrationExecutableFn
		integrationExecutableFn = func() (string, error) {
			return filepath.Join(appRoot, "integration"), nil
		}
		t.Cleanup(func() {
			integrationExecutableFn = originalExecutable
		})
		db, err := sql.Open("sqlite", filepath.Join(appRoot, "data"))
		if err != nil {
			t.Fatalf("open isolated db: %v", err)
		}
		oldCronDB := cronDB
		cronDB = db
		t.Cleanup(func() {
			cronDB = oldCronDB
			_ = db.Close()
		})
	}

	t.Run("cron only", func(t *testing.T) {
		restore := withIntegrationPluginSandbox(t, nil)
		defer restore()
		setupSkillsConfig(t)

		cfg := &Config{
			AgentDir:     agentDir,
			Device:       "test-dev",
			AgentCacheMs: 120000,
		}

		server := httptest.NewServer(http.HandlerFunc(handleSkills(cfg)))
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
			t.Fatalf("skills = %v, want %v", names, want)
		}
	})

	t.Run("browser and remote plugins", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("plugin shell script test is not supported on windows")
		}

		restore := withIntegrationPluginSandbox(t, map[string][2]string{
			"browser": {`{"key":"browser","name":"浏览器"}`, pluginParamJSON("cookie_path")},
			"remote":  {`{"key":"remote","name":"远程"}`, pluginParamJSON("exec_timeout")},
		})
		defer restore()
		setupSkillsConfig(t)

		cfg := &Config{
			AgentDir:     agentDir,
			Device:       "test-dev",
			AgentCacheMs: 120000,
		}

		server := httptest.NewServer(http.HandlerFunc(handleSkills(cfg)))
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
		writeIntegrationPluginPID(t, filepath.Join("..", "plugins", "browser.pid"), cmd.Process.Pid)
		writeIntegrationPluginPID(t, filepath.Join("..", "plugins", "remote.pid"), cmd.Process.Pid)

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
	})
}

func TestHandleSkillsReportsSeedreamOnlyWhenConfigured(t *testing.T) {
	agentDir, err := filepath.Abs("../proxy/iteration/20260419_4/test-case")
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}

	tmp := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
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
	oldCronDB := cronDB
	cronDB = db
	defer func() { cronDB = oldCronDB }()

	cfg := &Config{
		AgentDir:     agentDir,
		Device:       "test-dev-seedream",
		AgentCacheMs: 120000,
	}
	server := httptest.NewServer(http.HandlerFunc(handleSkills(cfg)))
	defer server.Close()

	getSkills := func() []string {
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
		return names
	}

	names := getSkills()
	for _, name := range names {
		if name == seedreamInternalSkillName {
			t.Fatalf("skills without seedream config = %v, want %q absent", names, seedreamInternalSkillName)
		}
	}

	if err := writeTokenStoreConfigs(db, map[string]tokenConfig{
		seedreamProviderName: {Token: "seedream-token"},
	}); err != nil {
		t.Fatalf("write seedream token config: %v", err)
	}

	names = getSkills()
	found := false
	for _, name := range names {
		if name == seedreamInternalSkillName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("skills with defaults = %v, want %q present", names, seedreamInternalSkillName)
	}
}

func TestHandleSkillsRespectsChatSkillState(t *testing.T) {
	tmp := t.TempDir()

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
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
	oldCronDB := cronDB
	cronDB = db
	defer func() { cronDB = oldCronDB }()

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

	if _, err := skillstate.SetDisabled(db, "chat-1", alphaDir, true); err != nil {
		t.Fatalf("SetDisabled: %v", err)
	}

	cfg := &Config{
		AgentDir:     agentDir,
		Device:       "test-dev",
		AgentCacheMs: 0,
	}

	server := httptest.NewServer(http.HandlerFunc(handleSkills(cfg)))
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

func TestHandleSkillsWarningRefreshScansAgentDirSkillsOnly(t *testing.T) {
	tmp := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	agentRoot := filepath.Join(tmp, "agent")
	agentDir := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(filepath.Join(agentDir, "skills", "broken"), 0o755); err != nil {
		t.Fatalf("mkdir skills broken: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(agentDir, "tmp", "ignored"), 0o755); err != nil {
		t.Fatalf("mkdir tmp ignored: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "skills", "broken", "SKILL.md"), []byte("---\ndescription: broken\n---\n"), 0o644); err != nil {
		t.Fatalf("write broken skill: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "skills", "outside"), 0o755); err != nil {
		t.Fatalf("mkdir cwd skills outside: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "skills", "outside", "SKILL.md"), []byte("---\ndescription: outside\n---\n"), 0o644); err != nil {
		t.Fatalf("write cwd skills outside: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "tmp", "ignored", "SKILL.md"), []byte("---\ndescription: ignored\n---\n"), 0o644); err != nil {
		t.Fatalf("write ignored tmp skill: %v", err)
	}

	cfg := &Config{AgentDir: agentRoot}
	server := httptest.NewServer(http.HandlerFunc(handleSkillsWarning(cfg)))
	defer server.Close()

	resp, err := http.Get(server.URL + "/skills_warning?refresh=1")
	if err != nil {
		t.Fatalf("GET /skills_warning refresh failed: %v", err)
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
	if !strings.HasSuffix(payload.Data[0].Path, "/skills/broken/SKILL.md") {
		t.Fatalf("path = %q, want suffix /skills/broken/SKILL.md", payload.Data[0].Path)
	}
}

func TestHandleSkillsWarningRefreshScansAllAgentsSkills(t *testing.T) {
	tmp := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	agentRoot := filepath.Join(tmp, "agent")
	if err := os.MkdirAll(filepath.Join(agentRoot, "A", "skills", "broken-a"), 0o755); err != nil {
		t.Fatalf("mkdir agent A skill: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(agentRoot, "B", "skills", "broken-b"), 0o755); err != nil {
		t.Fatalf("mkdir agent B skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentRoot, "A", "skills", "broken-a", "SKILL.md"), []byte("---\ndescription: broken-a\n---\n"), 0o644); err != nil {
		t.Fatalf("write agent A broken skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentRoot, "B", "skills", "broken-b", "SKILL.md"), []byte("---\ndescription: broken-b\n---\n"), 0o644); err != nil {
		t.Fatalf("write agent B broken skill: %v", err)
	}

	cfg := &Config{AgentDir: agentRoot}
	server := httptest.NewServer(http.HandlerFunc(handleSkillsWarning(cfg)))
	defer server.Close()

	resp, err := http.Get(server.URL + "/skills_warning?refresh=1")
	if err != nil {
		t.Fatalf("GET /skills_warning refresh failed: %v", err)
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
	if len(payload.Data) != 2 {
		t.Fatalf("warnings len = %d, want 2", len(payload.Data))
	}
	if !strings.Contains(payload.Data[0].Path, "/A/skills/") && !strings.Contains(payload.Data[1].Path, "/A/skills/") {
		t.Fatalf("payload missing agent A warning: %+v", payload.Data)
	}
	if !strings.Contains(payload.Data[0].Path, "/B/skills/") && !strings.Contains(payload.Data[1].Path, "/B/skills/") {
		t.Fatalf("payload missing agent B warning: %+v", payload.Data)
	}
}

func TestPluginsStopHandlerRejectsFlagStyleLifecycleOnlyPlugin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()

	writeIntegrationPluginActionScript(t, filepath.Join("..", "plugins", "feishu"), `"飞书"`, pluginParamJSON("appId", "appSecret"), "flag")

	cfg := &Config{}
	server := httptest.NewServer(http.HandlerFunc(handlePluginsStop(cfg)))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/plugins/stop?key=feishu&pid-file=.%2Ffeishu.pid", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want %d body=%s", resp.StatusCode, http.StatusInternalServerError, string(body))
	}

	var payload struct {
		Status  int    `json:"status"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Status != 1 {
		t.Fatalf("payload status = %d, want 1", payload.Status)
	}
	if !strings.Contains(payload.Content, "unknown command") {
		t.Fatalf("content = %q, want unknown command", payload.Content)
	}
}

func TestPluginsStopHandlerPassesConnectBin(t *testing.T) {
	scriptPath := filepath.Join(t.TempDir(), "integration-bin")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake integration executable: %v", err)
	}

	originalExecutable := integrationExecutableFn
	originalRunConnectPluginAction := runConnectPluginAction
	defer func() {
		integrationExecutableFn = originalExecutable
		runConnectPluginAction = originalRunConnectPluginAction
	}()

	integrationExecutableFn = func() (string, error) {
		return scriptPath, nil
	}
	runConnectPluginAction = func(target, action string, flags map[string]string) (*connectsvc.PluginActionResult, error) {
		if target != "browser" {
			t.Fatalf("target = %q, want browser", target)
		}
		if action != "stop" {
			t.Fatalf("action = %q, want stop", action)
		}
		if got := strings.TrimSpace(flags["connect-bin"]); got != scriptPath {
			t.Fatalf("connect-bin = %q, want %q", got, scriptPath)
		}
		return &connectsvc.PluginActionResult{
			Path:    target,
			Command: []string{"stop"},
			Output:  []byte(`{"status":"stopped"}`),
		}, nil
	}

	cfg := &Config{}
	server := httptest.NewServer(http.HandlerFunc(handlePluginsStop(cfg)))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/plugins/stop?key=browser", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body=%s", resp.StatusCode, string(body))
	}
}

func TestShutdownHandlerSchedulesPluginStopBeforeCancel(t *testing.T) {
	originalRunConnectListMeta := runConnectListMeta
	originalRunConnectPluginStatus := runConnectPluginStatus
	originalRunConnectPluginAction := runConnectPluginAction
	defer func() {
		runConnectListMeta = originalRunConnectListMeta
		runConnectPluginStatus = originalRunConnectPluginStatus
		runConnectPluginAction = originalRunConnectPluginAction
	}()

	runConnectListMeta = func() ([]connectMetaConfigListItem, error) {
		return []connectMetaConfigListItem{
			{Key: "feishu", Name: "飞书"},
		}, nil
	}
	runConnectPluginStatus = func(target string, flags map[string]string) (*connectsvc.PluginStatus, error) {
		return &connectsvc.PluginStatus{Key: target, Name: target, Started: true}, nil
	}

	events := make(chan string, 4)
	runConnectPluginAction = func(target, action string, flags map[string]string) (*connectsvc.PluginActionResult, error) {
		if target != "feishu" {
			t.Fatalf("target = %q, want feishu", target)
		}
		if action != "stop" {
			t.Fatalf("action = %q, want stop", action)
		}
		events <- "plugin-stop"
		return &connectsvc.PluginActionResult{
			Path:    target,
			Command: []string{action},
			Output:  []byte(`{"status":"stopped"}`),
		}, nil
	}

	canceled := make(chan struct{})
	controller := newIntegrationShutdownController(func() {
		events <- "cancel"
		close(canceled)
	}, nil)
	controller.delay = 20 * time.Millisecond

	server := httptest.NewServer(http.HandlerFunc(handleShutdown(controller)))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/shutdown", "application/json", nil)
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
			Scheduled bool  `json:"scheduled"`
			DelayMS   int64 `json:"delayMs"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Status != 0 {
		t.Fatalf("status = %d, want 0", payload.Status)
	}
	if !payload.Data.Scheduled {
		t.Fatal("scheduled = false, want true")
	}
	if payload.Data.DelayMS != 20 {
		t.Fatalf("delayMs = %d, want 20", payload.Data.DelayMS)
	}

	select {
	case <-canceled:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for shutdown cancel")
	}

	got := []string{<-events, <-events}
	if !reflect.DeepEqual(got, []string{"plugin-stop", "cancel"}) {
		t.Fatalf("events = %v, want [plugin-stop cancel]", got)
	}
}

func TestShutdownHandlerOnlySchedulesOnce(t *testing.T) {
	controller := newIntegrationShutdownController(func() {}, nil)
	controller.delay = 20 * time.Millisecond

	server := httptest.NewServer(http.HandlerFunc(handleShutdown(controller)))
	defer server.Close()

	for i, wantScheduled := range []bool{true, false} {
		resp, err := http.Post(server.URL+"/api/shutdown", "application/json", nil)
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		var payload struct {
			Status int `json:"status"`
			Data   struct {
				Scheduled bool `json:"scheduled"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			resp.Body.Close()
			t.Fatalf("decode request %d failed: %v", i, err)
		}
		resp.Body.Close()
		if payload.Status != 0 {
			t.Fatalf("request %d status = %d, want 0", i, payload.Status)
		}
		if payload.Data.Scheduled != wantScheduled {
			t.Fatalf("request %d scheduled = %v, want %v", i, payload.Data.Scheduled, wantScheduled)
		}
	}
}

func TestPluginsStartHandlerWritesFailureToPluginLog(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin shell script test is not supported on windows")
	}

	restore := withIntegrationPluginSandbox(t, map[string][2]string{
		"feishu": {`"飞书"`, pluginParamJSON("appId", "appSecret")},
	})
	defer restore()

	originalRunConnectPluginAction := runConnectPluginAction
	defer func() {
		runConnectPluginAction = originalRunConnectPluginAction
	}()

	runConnectPluginAction = func(target, action string, flags map[string]string) (*connectsvc.PluginActionResult, error) {
		if target != "feishu" {
			t.Fatalf("target = %q, want feishu", target)
		}
		if action != "start" {
			t.Fatalf("action = %q, want start", action)
		}
		return nil, errors.New("model not registered: deepright")
	}

	cfg := &Config{}
	server := httptest.NewServer(http.HandlerFunc(handlePluginsStart(cfg)))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/plugins/start?key=feishu", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body=%s", resp.StatusCode, string(body))
	}

	data, err := os.ReadFile(filepath.Join("..", "plugins", "feishu.log"))
	if err != nil {
		t.Fatalf("read plugin log: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "做了：插件操作失败，原因：model not registered: deepright") {
		t.Fatalf("plugin log = %q", text)
	}
}

func TestIntegrationPluginCLIPassesConnectBinToStop(t *testing.T) {
	scriptPath := filepath.Join(t.TempDir(), "integration-bin")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake integration executable: %v", err)
	}

	originalExecutable := integrationExecutableFn
	originalRunConnectPluginAction := runConnectPluginAction
	defer func() {
		integrationExecutableFn = originalExecutable
		runConnectPluginAction = originalRunConnectPluginAction
	}()

	integrationExecutableFn = func() (string, error) {
		return scriptPath, nil
	}
	runConnectPluginAction = func(target, action string, flags map[string]string) (*connectsvc.PluginActionResult, error) {
		if target != "browser" {
			t.Fatalf("target = %q, want browser", target)
		}
		if action != "stop" {
			t.Fatalf("action = %q, want stop", action)
		}
		if got := strings.TrimSpace(flags["connect-bin"]); got != scriptPath {
			t.Fatalf("connect-bin = %q, want %q", got, scriptPath)
		}
		return &connectsvc.PluginActionResult{
			Path:    "/tmp/browser",
			Command: []string{"stop"},
			Output:  []byte(`{"status":"stopped"}`),
		}, nil
	}

	stdout := captureIntegrationStdout(t, func() {
		runIntegrationPluginCLI([]string{"stop", "--key", "browser"})
	})
	if !strings.Contains(stdout, `"status": 0`) {
		t.Fatalf("stdout = %q, want success json", stdout)
	}
}

func withIntegrationPluginSandbox(t *testing.T, plugins map[string][2]string) func() {
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
		writeIntegrationPluginScript(t, filepath.Join(pluginsDir, file), outputs[0], outputs[1])
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

func writeIntegrationPluginScript(t *testing.T, path, nameOutput, paramOutput string) {
	t.Helper()
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"name\" ]; then\n" +
		"  printf '%s\\n' '" + escapeIntegrationSingleQuotes(nameOutput) + "'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"param\" ]; then\n" +
		"  printf '%s\\n' '" + escapeIntegrationSingleQuotes(paramOutput) + "'\n" +
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

func writeIntegrationPluginScriptWithScope(t *testing.T, path, nameOutput, paramOutput, scopeOutput string) {
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

func writeIntegrationPluginPID(t *testing.T, path string, pid int) {
	t.Helper()
	if err := os.WriteFile(path, []byte(strconv.Itoa(pid)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func escapeSingleQuotes(input string) string {
	return strings.ReplaceAll(input, "'", `'"'"'`)
}

func writeIntegrationPluginActionScript(t *testing.T, path, nameOutput, paramOutput, mode string) {
	t.Helper()
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"name\" ]; then\n" +
		"  printf '%s\\n' '" + escapeIntegrationSingleQuotes(nameOutput) + "'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"param\" ]; then\n" +
		"  printf '%s\\n' '" + escapeIntegrationSingleQuotes(paramOutput) + "'\n" +
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
		t.Fatalf("unknown action mode: %s", mode)
	}
	content += "printf 'unknown command: %s\\n' \"$1\" >&2\n" +
		"exit 1\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeIntegrationPluginExecScript(t *testing.T, path, nameOutput, paramOutput string) {
	t.Helper()
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"name\" ]; then\n" +
		"  printf '%s\\n' '" + escapeIntegrationSingleQuotes(nameOutput) + "'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"param\" ]; then\n" +
		"  printf '%s\\n' '" + escapeIntegrationSingleQuotes(paramOutput) + "'\n" +
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

func captureIntegrationStdout(t *testing.T, fn func()) string {
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

func TestRunIntegrationNotifyCLIReportsSupportedFromNotificationPackage(t *testing.T) {
	oldSupported := integrationNotificationSupportedFn
	oldSend := integrationSendNotificationFn
	t.Cleanup(func() {
		integrationNotificationSupportedFn = oldSupported
		integrationSendNotificationFn = oldSend
	})
	integrationNotificationSupportedFn = func() bool { return true }

	var got integrationnotification.Options
	integrationSendNotificationFn = func(opts integrationnotification.Options) error {
		got = opts
		return nil
	}

	stdout := captureIntegrationStdout(t, func() {
		runIntegrationNotifyCLI([]string{"--title", " Windows通知 ", "--message", " 已完成 "})
	})

	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &payload); err != nil {
		t.Fatalf("unmarshal stdout: %v", err)
	}
	if payload["supported"] != true {
		t.Fatalf("supported = %v, want true", payload["supported"])
	}
	if payload["title"] != "Windows通知" {
		t.Fatalf("title = %v, want %q", payload["title"], "Windows通知")
	}
	if payload["message"] != "已完成" {
		t.Fatalf("message = %v, want %q", payload["message"], "已完成")
	}
	if got.Title != " Windows通知 " || got.Message != " 已完成 " {
		t.Fatalf("notify options = %#v, want original flag values", got)
	}
}

func TestRunIntegrationBackupCleanCLIOutputsJSON(t *testing.T) {
	agentRoot := filepath.Join(t.TempDir(), "agents")
	workspace := filepath.Join(agentRoot, "agent-a")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	target := filepath.Join(workspace, "USER.md.bak")
	if err := os.WriteFile(target, []byte("backup"), 0o644); err != nil {
		t.Fatalf("write backup file: %v", err)
	}
	oldTime := time.Now().Add(-30 * time.Hour).Truncate(time.Second)
	if err := os.Chtimes(target, oldTime, oldTime); err != nil {
		t.Fatalf("chtimes backup file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runIntegrationBackupCleanCLI([]string{"--agent-dir", agentRoot}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr should be empty, got %q", stderr.String())
	}

	var payload struct {
		Status      int                 `json:"status"`
		AgentDir    string              `json:"agentDir"`
		AgentCount  int                 `json:"agentCount"`
		ArchivedCnt int                 `json:"archivedCnt"`
		DeletedCnt  int                 `json:"deletedCnt"`
		Archived    []backupCleanupMove `json:"archived"`
		Deleted     []string            `json:"deleted"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode stdout: %v; raw=%s", err, stdout.String())
	}
	if payload.Status != 0 {
		t.Fatalf("status = %d; raw=%s", payload.Status, stdout.String())
	}
	if payload.AgentCount != 1 {
		t.Fatalf("agentCount = %d, want 1", payload.AgentCount)
	}
	if payload.ArchivedCnt != 1 || len(payload.Archived) != 1 {
		t.Fatalf("archived count = %d/%d, want 1/1; raw=%s", payload.ArchivedCnt, len(payload.Archived), stdout.String())
	}
	if payload.DeletedCnt != 0 || len(payload.Deleted) != 0 {
		t.Fatalf("deleted count = %d/%d, want 0/0; raw=%s", payload.DeletedCnt, len(payload.Deleted), stdout.String())
	}
	if _, err := os.Stat(filepath.Join(workspace, "bak", "USER.md.bak")); err != nil {
		t.Fatalf("backup file should be moved into bak: %v", err)
	}
}

func escapeIntegrationSingleQuotes(input string) string {
	return strings.ReplaceAll(input, "'", `'"'"'`)
}

func seedIntegrationPluginMetaTestData(t *testing.T, dbPath, agentDir string, input connectsvc.MetaInput) {
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

func collectIntegrationPluginLogEvents(body io.Reader, lines chan<- string, errs chan<- error) {
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

func TestIntegrationLaunchSplashLogoPathPrefersBundledSiteIcon(t *testing.T) {
	bundleRoot := filepath.Join(t.TempDir(), "integration.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	siteDir := filepath.Join(bundleRoot, "Contents", "Resources", "site")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(siteDir, 0o755); err != nil {
		t.Fatal(err)
	}
	iconPath := filepath.Join(siteDir, "icon.png")
	if err := os.WriteFile(iconPath, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}

	originalExecutable := integrationExecutableFn
	defer func() { integrationExecutableFn = originalExecutable }()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(macOSDir, "integration"), nil
	}

	if got := integrationLaunchSplashLogoPath(); got != iconPath {
		t.Fatalf("integrationLaunchSplashLogoPath() = %q, want %q", got, iconPath)
	}
}

func TestHandleIntegrationBundleAppLaunchStartsSplashBeforeOpen(t *testing.T) {
	oldSplash := integrationStartLaunchSplashFn
	oldReadyCheck := integrationReadyCheck
	oldBrowserEntryReadyCheck := integrationBrowserEntryReadyCheck
	oldOpenBrowser := integrationOpenBrowserFn
	oldExecutable := integrationExecutableFn
	oldHome := integrationUserHomeFn
	oldRuntimeEnv := os.Getenv(integrationRuntimeDirEnv)
	oldPluginEnv := os.Getenv(integrationPluginDirEnv)
	defer func() {
		integrationStartLaunchSplashFn = oldSplash
		integrationReadyCheck = oldReadyCheck
		integrationBrowserEntryReadyCheck = oldBrowserEntryReadyCheck
		integrationOpenBrowserFn = oldOpenBrowser
		integrationExecutableFn = oldExecutable
		integrationUserHomeFn = oldHome
		_ = os.Setenv(integrationRuntimeDirEnv, oldRuntimeEnv)
		_ = os.Setenv(integrationPluginDirEnv, oldPluginEnv)
	}()

	bundleRoot := filepath.Join(t.TempDir(), "DeepRight.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	pluginsDir := filepath.Join(bundleRoot, "Contents", "Resources", "plugins")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginsDir, "browser"), []byte("plugin"), 0o755); err != nil {
		t.Fatal(err)
	}

	runtimeRoot := t.TempDir()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(macOSDir, "integration"), nil
	}
	if runtime.GOOS == "darwin" {
		integrationUserHomeFn = func() (string, error) { return runtimeRoot, nil }
	} else {
		if err := os.Setenv(integrationRuntimeDirEnv, runtimeRoot); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Unsetenv(integrationPluginDirEnv); err != nil {
		t.Fatal(err)
	}

	var calls []string
	integrationStartLaunchSplashFn = func() error {
		calls = append(calls, "splash")
		return nil
	}
	integrationReadyCheck = func(port string) bool {
		calls = append(calls, "ready:"+port)
		return true
	}
	integrationBrowserEntryReadyCheck = func(int) bool { return true }
	integrationOpenBrowserFn = func(target string) error {
		calls = append(calls, "open:"+target)
		return nil
	}

	var stderr bytes.Buffer
	if code := handleIntegrationBundleAppLaunch(&stderr); code != 0 {
		t.Fatalf("handleIntegrationBundleAppLaunch() code = %d, stderr=%s", code, stderr.String())
	}
	if len(calls) != 3 {
		t.Fatalf("calls = %#v", calls)
	}
	if calls[0] != "splash" {
		t.Fatalf("first call = %q, want splash", calls[0])
	}
	if !strings.HasPrefix(calls[1], "ready:") {
		t.Fatalf("second call = %q, want readiness check", calls[1])
	}
	if !strings.HasPrefix(calls[2], "open:http://localhost:") || !strings.Contains(calls[2], "/site/") || !strings.HasSuffix(calls[2], "#app") {
		t.Fatalf("third call = %q, want browser open URL", calls[2])
	}
}

func TestHandleIntegrationBundleAppLaunchAllowsOutsideApplications(t *testing.T) {
	oldAlert := integrationBundleLaunchAlertFn
	oldExecutable := integrationExecutableFn
	oldHome := integrationUserHomeFn
	oldEvalSymlinks := integrationEvalSymlinksFn
	oldSplash := integrationStartLaunchSplashFn
	oldReadyCheck := integrationReadyCheck
	oldBrowserEntryReadyCheck := integrationBrowserEntryReadyCheck
	oldOpenBrowser := integrationOpenBrowserFn
	oldRuntimeEnv := os.Getenv(integrationRuntimeDirEnv)
	oldPluginEnv := os.Getenv(integrationPluginDirEnv)
	defer func() {
		integrationBundleLaunchAlertFn = oldAlert
		integrationExecutableFn = oldExecutable
		integrationUserHomeFn = oldHome
		integrationEvalSymlinksFn = oldEvalSymlinks
		integrationStartLaunchSplashFn = oldSplash
		integrationReadyCheck = oldReadyCheck
		integrationBrowserEntryReadyCheck = oldBrowserEntryReadyCheck
		integrationOpenBrowserFn = oldOpenBrowser
		_ = os.Setenv(integrationRuntimeDirEnv, oldRuntimeEnv)
		_ = os.Setenv(integrationPluginDirEnv, oldPluginEnv)
	}()

	bundleRoot := filepath.Join(t.TempDir(), "Downloads", "DeepRight.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	pluginsDir := filepath.Join(bundleRoot, "Contents", "Resources", "plugins")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	runtimeRoot := t.TempDir()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(macOSDir, "integration"), nil
	}
	if runtime.GOOS == "darwin" {
		integrationUserHomeFn = func() (string, error) { return runtimeRoot, nil }
	} else {
		integrationUserHomeFn = func() (string, error) { return runtimeRoot, nil }
		if err := os.Setenv(integrationRuntimeDirEnv, runtimeRoot); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Unsetenv(integrationPluginDirEnv); err != nil {
		t.Fatal(err)
	}
	integrationEvalSymlinksFn = func(path string) (string, error) { return filepath.Clean(path), nil }

	alerted := false
	integrationBundleLaunchAlertFn = func(layout *integrationBundlePaths) error {
		alerted = true
		if layout == nil || layout.BundleRoot != bundleRoot {
			t.Fatalf("alert layout = %#v, want bundle root %q", layout, bundleRoot)
		}
		return nil
	}
	integrationStartLaunchSplashFn = func() error {
		return nil
	}
	integrationReadyCheck = func(string) bool {
		return true
	}
	integrationBrowserEntryReadyCheck = func(int) bool {
		return true
	}
	integrationOpenBrowserFn = func(string) error {
		return nil
	}

	var stderr bytes.Buffer
	if code := handleIntegrationBundleAppLaunch(&stderr); code != 0 {
		t.Fatalf("handleIntegrationBundleAppLaunch() code = %d, want 0", code)
	}
	if alerted {
		t.Fatal("launch location alert should not be shown")
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestCloseActiveConnIsIdempotent(t *testing.T) {
	ch := make(chan chatMsg, 1)
	ac := &activeConn{ch: ch}

	closeActiveConn(ac)
	closeActiveConn(ac)

	if ac.ch != nil {
		t.Fatal("expected active connection channel to be cleared")
	}

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected closed channel")
		}
	default:
		t.Fatal("expected closed channel to be readable immediately")
	}
}
