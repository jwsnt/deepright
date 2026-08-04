package main

import (
	messageinsert "cli-get/messageinsert"
	"compress/gzip"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

func TestCreateHTTPClientForcesHTTP11AndDoesNotFollowRedirects(t *testing.T) {
	client := createHTTPClient(15*time.Second, 45*time.Second, 45*time.Second, 90*time.Second)
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T", client.Transport)
	}
	if transport.ForceAttemptHTTP2 {
		t.Fatalf("ForceAttemptHTTP2 = true, want false")
	}
	if len(transport.TLSNextProto) != 0 {
		t.Fatalf("TLSNextProto length = %d, want 0", len(transport.TLSNextProto))
	}
	if client.CheckRedirect == nil {
		t.Fatal("CheckRedirect should not be nil")
	}
}

func resetGlobalStateForTest() {
	loggerState.mu.Lock()
	oldLogger := loggerState.logger
	loggerState.path = ""
	loggerState.logger = nil
	loggerState.mu.Unlock()
	if oldLogger != nil {
		close(oldLogger.ch)
	}

	sharedDataDB.mu.Lock()
	if sharedDataDB.db != nil {
		_ = sharedDataDB.db.Close()
		sharedDataDB.db = nil
	}
	sharedDataDB.path = ""
	sharedDataDB.mu.Unlock()

	activeCmdMu.Lock()
	activeCmdMap = make(map[string]*activeCmd)
	activeCmdMu.Unlock()
}

func TestHeartbeatRequestFormat(t *testing.T) {
	var captured GetRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cli/get" {
			t.Errorf("expected path /cli/get, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &captured)

		// Return no-task response
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
		DeviceID: "test-device",
		Terminal: "/bin/zsh",
		Plugins:  []string{"browser", "feishu"},
		Gateway:  "aa:bb:cc:dd:ee:ff",
		Sys:      "Darwin 23.4.0",
		Agents: []Agent{
			{
				Workspace: "/tmp/test/a",
				AgentID:   "a",
				Provider:  "deepseek",
				Soul:      "HELLO SOUL",
				User:      "HELLO USER",
				Skills:    []Skill{{Name: "__internal_E", Description: "技能E"}},
			},
		},
	}

	client := &http.Client{Timeout: 5 * time.Second}
	task, err := Heartbeat(client, server.URL, metadata)
	if err != nil {
		t.Fatalf("heartbeat error: %v", err)
	}
	if task != nil {
		t.Error("expected nil task for empty content")
	}

	// Verify request format
	if captured.Model != "" {
		t.Errorf("model should be empty, got %q", captured.Model)
	}
	if len(captured.Messages) != 1 || captured.Messages[0].Role != "user" || captured.Messages[0].Content != "" {
		t.Errorf("messages format incorrect: %+v", captured.Messages)
	}
	if captured.Metadata == nil {
		t.Fatal("metadata should not be nil")
	}
	if captured.Metadata.DeviceID != "test-device" {
		t.Errorf("deviceId = %q, want test-device", captured.Metadata.DeviceID)
	}
	if captured.Metadata.Terminal != "/bin/zsh" {
		t.Errorf("terminal = %q, want /bin/zsh", captured.Metadata.Terminal)
	}
	if len(captured.Metadata.Plugins) != 2 || captured.Metadata.Plugins[0] != "browser" || captured.Metadata.Plugins[1] != "feishu" {
		t.Errorf("plugins = %#v", captured.Metadata.Plugins)
	}
	if captured.Metadata.Gateway != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("gateway = %q", captured.Metadata.Gateway)
	}
	if captured.Metadata.Sys != "Darwin 23.4.0" {
		t.Errorf("sys = %q", captured.Metadata.Sys)
	}
	if len(captured.Metadata.Agents) != 1 || captured.Metadata.Agents[0].AgentID != "a" {
		t.Errorf("agents incorrect: %+v", captured.Metadata.Agents)
	}
	if captured.Metadata.Agents[0].Provider != "deepseek" {
		t.Errorf("provider = %q, want deepseek", captured.Metadata.Agents[0].Provider)
	}
}

func TestApplicationDataDirUsesAgentRootParent(t *testing.T) {
	root := t.TempDir()
	agentRoot := filepath.Join(root, "agent")
	if got := applicationDataDir(agentRoot); got != root {
		t.Fatalf("applicationDataDir(%q) = %q, want %q", agentRoot, got, root)
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

func TestHeartbeatWithNoTaskDoesNotWriteLog(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	resetGlobalStateForTest()

	tests := []struct {
		name    string
		content interface{}
	}{
		{name: "null content", content: nil},
		{name: "empty string content", content: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
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
						}{Role: "assistant", Content: tc.content}},
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
			task, err := Heartbeat(client, server.URL, meta)
			if err != nil {
				t.Fatalf("Heartbeat: %v", err)
			}
			if task != nil {
				t.Fatalf("task = %#v, want nil", task)
			}

			db, err := getDataDB()
			if err != nil {
				t.Fatalf("getDataDB: %v", err)
			}
			var count int
			if err := db.QueryRow(`SELECT COUNT(*) FROM agent_message_log WHERE log_type = ?`, logTypeCLIGet).Scan(&count); err != nil {
				t.Fatalf("count logs: %v", err)
			}
			if count != 0 {
				t.Fatalf("cli/get log count = %d, want 0", count)
			}
		})
	}
}

func TestHeartbeatWithTask(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	resetGlobalStateForTest()

	taskJSON := `{"timeout":5000,"suffix":"cmd","type":"cmd","tid":"task_001","cmd":"echo hello"}`

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

	metadata := &AgentOutput{DeviceID: "d", Terminal: "t", Gateway: "g", Sys: "s", Agents: []Agent{}}
	client := &http.Client{Timeout: 5 * time.Second}

	task, err := Heartbeat(client, server.URL, metadata)
	if err != nil {
		t.Fatalf("heartbeat error: %v", err)
	}
	if task == nil {
		t.Fatal("expected a task")
	}
	if task.Tid != "task_001" {
		t.Errorf("tid = %q, want task_001", task.Tid)
	}
	if task.Cmd != "echo hello" {
		t.Errorf("cmd = %q", task.Cmd)
	}
	if task.Suffix != "cmd" {
		t.Errorf("suffix = %q", task.Suffix)
	}
	if task.Type != "cmd" {
		t.Errorf("type = %q", task.Type)
	}
	if task.Timeout != 5000 {
		t.Errorf("timeout = %d", task.Timeout)
	}

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	var stored string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		err = db.QueryRow(`SELECT content FROM agent_message_log WHERE log_type = ? ORDER BY id DESC LIMIT 1`, logTypeCLIGet).Scan(&stored)
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("select cli/get log: %v", err)
	}
	if stored != taskJSON {
		t.Fatalf("cli/get log content = %q, want %q", stored, taskJSON)
	}
}

func TestHeartbeatDoesNotFollowRedirect(t *testing.T) {
	var firstMethod string
	redirectHits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cli/get":
			firstMethod = r.Method
			w.Header().Set("Location", "/redirected")
			w.WriteHeader(http.StatusMovedPermanently)
		case "/redirected":
			redirectHits++
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := createHTTPClient(15*time.Second, 45*time.Second, 45*time.Second, 90*time.Second)
	_, err := Heartbeat(client, server.URL, &AgentOutput{})
	if err == nil {
		t.Fatal("Heartbeat error = nil, want redirect error")
	}
	if !strings.Contains(err.Error(), "heartbeat redirect HTTP 301") {
		t.Fatalf("Heartbeat error = %q, want redirect error", err.Error())
	}
	if firstMethod != http.MethodPost {
		t.Fatalf("first method = %s, want POST", firstMethod)
	}
	if redirectHits != 0 {
		t.Fatalf("redirected request count = %d, want 0", redirectHits)
	}
}

func TestExecuteAndPublish(t *testing.T) {
	task := &TaskContent{
		Timeout: 5000,
		Suffix:  "cmd",
		Type:    "cmd",
		Tid:     "task_002",
		Cmd:     "echo test_output",
	}

	terminal := detectTerminal()
	if terminal == "" {
		t.Skip("no SHELL detected")
	}

	result := ExecuteTask(task, terminal)

	// Verify result format
	if result.Status != 0 {
		t.Errorf("status = %d, want 0", result.Status)
	}
	if result.Suffix != "cmd" {
		t.Errorf("suffix = %q", result.Suffix)
	}
	if result.Type != "cmd" {
		t.Errorf("type = %q", result.Type)
	}
	if result.Tid != "task_002" {
		t.Errorf("tid = %q", result.Tid)
	}

	// Decode GZIP+Base64 cmd
	decoded, err := base64.StdEncoding.DecodeString(result.Cmd)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	gz, err := gzip.NewReader(strings.NewReader(string(decoded)))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	output, _ := io.ReadAll(gz)
	gz.Close()
	if !strings.Contains(string(output), "test_output") {
		t.Errorf("cmd output = %q, want to contain test_output", string(output))
	}

	// Test publish format
	var capturedPub PubRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cli/pub" {
			t.Errorf("expected path /cli/pub, got %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedPub)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ResponsePayload{Code: 200})
	}))
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	pubMeta := &AgentOutput{
		DeviceID: "test-device",
		Terminal: "/bin/zsh",
		Plugins:  []string{"browser"},
		Gateway:  "aa:bb:cc:dd:ee:ff",
		Sys:      "Darwin 23.4.0",
		Agents:   []Agent{{Workspace: "/tmp", AgentID: "a", Provider: "deepseek", Skills: []Skill{}}},
	}
	if err := PublishResult(client, server.URL, result, pubMeta); err != nil {
		t.Fatalf("publish error: %v", err)
	}

	// Verify pub request format
	if capturedPub.Model != "" {
		t.Errorf("model should be empty, got %q", capturedPub.Model)
	}
	if len(capturedPub.Messages) != 1 || capturedPub.Messages[0].Role != "user" {
		t.Errorf("messages format incorrect: %+v", capturedPub.Messages)
	}
	// Verify metadata in pub request
	if capturedPub.Metadata == nil {
		t.Fatal("pub metadata should not be nil")
	}
	if capturedPub.Metadata.DeviceID != "test-device" {
		t.Errorf("pub metadata deviceId = %q, want test-device", capturedPub.Metadata.DeviceID)
	}
	if len(capturedPub.Metadata.Plugins) != 1 || capturedPub.Metadata.Plugins[0] != "browser" {
		t.Errorf("pub metadata plugins incorrect: %+v", capturedPub.Metadata.Plugins)
	}
	if len(capturedPub.Metadata.Agents) != 1 || capturedPub.Metadata.Agents[0].AgentID != "a" {
		t.Errorf("pub metadata agents incorrect: %+v", capturedPub.Metadata.Agents)
	}
	if capturedPub.Metadata.Agents[0].Provider != "deepseek" {
		t.Errorf("pub metadata provider = %q, want deepseek", capturedPub.Metadata.Agents[0].Provider)
	}

	// Parse content as JSON and verify schema
	var pubResult ResultPayload
	if err := json.Unmarshal([]byte(capturedPub.Messages[0].Content), &pubResult); err != nil {
		t.Fatalf("parse pub content: %v", err)
	}
	if pubResult.Status != 0 {
		t.Errorf("pub status = %d", pubResult.Status)
	}
	if pubResult.Suffix != "cmd" {
		t.Errorf("pub suffix = %q", pubResult.Suffix)
	}
	if pubResult.Type != "cmd" {
		t.Errorf("pub type = %q", pubResult.Type)
	}
	if pubResult.Tid != "task_002" {
		t.Errorf("pub tid = %q", pubResult.Tid)
	}
	if pubResult.Cmd == "" {
		t.Error("pub cmd should not be empty (GZIP+Base64)")
	}
}

func TestPublishResultDoesNotFollowRedirect(t *testing.T) {
	var firstMethod string
	redirectHits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cli/pub":
			firstMethod = r.Method
			w.Header().Set("Location", "/redirected")
			w.WriteHeader(http.StatusMovedPermanently)
		case "/redirected":
			redirectHits++
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := createHTTPClient(15*time.Second, 45*time.Second, 45*time.Second, 90*time.Second)
	err := PublishResult(client, server.URL, &ResultPayload{Status: 0, Cmd: "H4sI"}, &AgentOutput{})
	if err == nil {
		t.Fatal("PublishResult error = nil, want redirect error")
	}
	if !strings.Contains(err.Error(), "publish redirect HTTP 301") {
		t.Fatalf("PublishResult error = %q, want redirect error", err.Error())
	}
	if firstMethod != http.MethodPost {
		t.Fatalf("first method = %s, want POST", firstMethod)
	}
	if redirectHits != 0 {
		t.Fatalf("redirected request count = %d, want 0", redirectHits)
	}
}

func TestPublishResultIncludesPendingMessageInsertAndMarksUploaded(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	resetGlobalStateForTest()

	db, err := getDataDB()
	if err != nil {
		t.Fatalf("getDataDB: %v", err)
	}
	if _, err := messageinsert.UpsertPending(db, "agent-a", "chat-1", "1710000000000", "first insert", time.Now()); err != nil {
		t.Fatalf("insert first pending: %v", err)
	}
	if _, err := messageinsert.UpsertPending(db, "agent-a", "chat-1", "1710000005000", "second insert", time.Now().Add(time.Millisecond)); err != nil {
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
	if err := PublishResult(&http.Client{Timeout: time.Second}, server.URL, result, &AgentOutput{}); err != nil {
		t.Fatalf("PublishResult: %v", err)
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

	rows, err := db.Query(`SELECT mid, status, reported_at FROM message_insert WHERE chat_id = ? ORDER BY mid`, "chat-1")
	if err != nil {
		t.Fatalf("query statuses: %v", err)
	}
	defer rows.Close()
	statuses := map[string]int{}
	reported := map[string]string{}
	for rows.Next() {
		var mid string
		var status int
		var reportedAt string
		if err := rows.Scan(&mid, &status, &reportedAt); err != nil {
			t.Fatalf("scan status: %v", err)
		}
		statuses[mid] = status
		reported[mid] = reportedAt
	}
	if statuses["1710000000000"] != messageinsert.StatusPending {
		t.Fatalf("status first = %d, want %d", statuses["1710000000000"], messageinsert.StatusPending)
	}
	if statuses["1710000005000"] != messageinsert.StatusPending {
		t.Fatalf("status second = %d, want %d", statuses["1710000005000"], messageinsert.StatusPending)
	}
	if reported["1710000000000"] == "" || reported["1710000005000"] == "" {
		t.Fatalf("reported_at not updated: %#v", reported)
	}

	pending, err := messageinsert.ListPending(db, "chat-1", 5)
	if err != nil {
		t.Fatalf("ListPending after publish: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending after publish = %#v, want none", pending)
	}
}

func TestHeartbeatAndPublishWriteUnifiedLogs(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	resetGlobalStateForTest()

	taskJSON := `{"timeout":5000,"suffix":"cmd","type":"cmd","tid":"task_001","cmd":"echo hello","agentId":"a","chat":"chat_001"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/cli/get" {
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
			return
		}
		json.NewEncoder(w).Encode(ResponsePayload{Code: 200})
	}))
	defer server.Close()

	meta := &AgentOutput{
		DeviceID: "test-device",
		Terminal: "/bin/zsh",
		Agents:   []Agent{{AgentID: "a", Workspace: "/tmp/a"}},
	}
	client := &http.Client{Timeout: 5 * time.Second}
	task, err := Heartbeat(client, server.URL, meta)
	if err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if task == nil {
		t.Fatal("expected task")
	}
	result := &ResultPayload{
		Status:  0,
		AgentId: "a",
		Chat:    "chat_001",
		Suffix:  "cmd",
		Type:    "cmd",
		Cmd:     "H4sI",
		Tid:     "task_001",
	}
	if err := PublishResult(client, server.URL, result, meta); err != nil {
		t.Fatalf("PublishResult: %v", err)
	}

	db, err := sql.Open("sqlite", filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if _, err := getDataDB(); err != nil {
		t.Fatalf("getDataDB: %v", err)
	}

	var getCount, pubCount int
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := db.QueryRow(`SELECT COUNT(*) FROM agent_message_log WHERE agent_id = ? AND chat_id = ? AND log_type = ?`,
			"a", "chat_001", logTypeCLIGet).Scan(&getCount); err != nil {
			t.Fatalf("count cli/get logs: %v", err)
		}
		if err := db.QueryRow(`SELECT COUNT(*) FROM agent_message_log WHERE agent_id = ? AND chat_id = ? AND log_type = ?`,
			"a", "chat_001", logTypeCLIPub).Scan(&pubCount); err != nil {
			t.Fatalf("count cli/pub logs: %v", err)
		}
		if getCount == 1 && pubCount == 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if getCount != 1 {
		t.Fatalf("cli/get log count = %d, want 1", getCount)
	}
	if pubCount != 1 {
		t.Fatalf("cli/pub log count = %d, want 1", pubCount)
	}
	var storedGet string
	if err := db.QueryRow(`SELECT content FROM agent_message_log WHERE agent_id = ? AND chat_id = ? AND log_type = ? ORDER BY id DESC LIMIT 1`,
		"a", "chat_001", logTypeCLIGet).Scan(&storedGet); err != nil {
		t.Fatalf("select cli/get log: %v", err)
	}
	if storedGet != taskJSON {
		t.Fatalf("cli/get content = %q, want %q", storedGet, taskJSON)
	}
}

func TestExecuteTaskReturnsKilledMessageWhenCancelled(t *testing.T) {
	terminal := detectTerminal()
	if terminal == "" {
		t.Skip("no SHELL detected")
	}

	task := &TaskContent{
		Timeout: 10000,
		Suffix:  "cmd",
		Type:    "cmd",
		Tid:     "task-kill",
		Cmd:     "sleep 5",
		AgentId: "a",
		Chat:    "chat-kill",
	}

	done := make(chan *ResultPayload, 1)
	go func() {
		done <- ExecuteTask(task, terminal)
	}()

	deadline := time.Now().Add(2 * time.Second)
	var key string
	for time.Now().Before(deadline) {
		activeCmdMu.Lock()
		for currentKey, item := range activeCmdMap {
			if item.agent == "a" && item.chat == "chat-kill" && item.tid == "task-kill" {
				key = currentKey
				if item.cancel != nil {
					item.cancel()
				}
				break
			}
		}
		activeCmdMu.Unlock()
		if key != "" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if key == "" {
		t.Fatal("active command not registered")
	}

	select {
	case result := <-done:
		if result.Status != 1 {
			t.Fatalf("status = %d, want 1", result.Status)
		}
		decoded, err := base64.StdEncoding.DecodeString(result.Cmd)
		if err != nil {
			t.Fatalf("base64 decode: %v", err)
		}
		gz, err := gzip.NewReader(strings.NewReader(string(decoded)))
		if err != nil {
			t.Fatalf("gzip reader: %v", err)
		}
		body, err := io.ReadAll(gz)
		if err != nil {
			t.Fatalf("read gzip: %v", err)
		}
		_ = gz.Close()
		if string(body) != "命令被终止" {
			t.Fatalf("body = %q, want 命令被终止", string(body))
		}
	case <-time.After(3 * time.Second):
		t.Fatal("ExecuteTask did not return after cancel")
	}
}

func TestDetectPluginKeysReturnsConfiguredPluginsFromListMeta(t *testing.T) {
	root := t.TempDir()
	appPath := filepath.Join(root, "cli-get")
	if err := os.WriteFile(appPath, []byte("#!/bin/sh\nif [ \"$1\" = \"list-meta\" ]; then\n  printf '%s\\n' '[{\"key\":\"feishu\",\"name\":\"飞书\"}]'\n  exit 0\nfi\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	pluginDir := filepath.Join(root, "plugins")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "feishu.pid"), []byte(itoa(os.Getpid())), 0o644); err != nil {
		t.Fatal(err)
	}

	restoreExec := osExecutable
	osExecutable = func() (string, error) { return appPath, nil }
	defer func() { osExecutable = restoreExec }()

	got := detectPluginKeys()
	if len(got) != 1 || got[0] != "feishu" {
		t.Fatalf("plugins = %#v, want [feishu]", got)
	}
}

func TestSandboxStateForTaskReadsSessionStateOnly(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	resetGlobalStateForTest()

	if err := SetSandboxMode("agent-a", "chat-a", sandboxModeFilePickNet); err != nil {
		t.Fatalf("SetSandboxMode: %v", err)
	}

	task := &TaskContent{
		AgentId: "agent-a",
		Chat:    "chat-a",
	}

	state, err := sandboxStateForTask(nil, task)
	if err != nil {
		t.Fatalf("sandboxStateForTask: %v", err)
	}
	if state.Mode != sandboxModeFilePickNet {
		t.Fatalf("state.Mode = %q, want %q", state.Mode, sandboxModeFilePickNet)
	}
}

func TestSandboxStateForTaskUsesChatScopedStateAcrossAgents(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	resetGlobalStateForTest()

	if err := SetSandboxMode("agent-a", "chat-a", sandboxModeNet); err != nil {
		t.Fatalf("SetSandboxMode: %v", err)
	}

	task := &TaskContent{
		AgentId: "agent-b",
		Chat:    "chat-a",
	}

	state, err := sandboxStateForTask(nil, task)
	if err != nil {
		t.Fatalf("sandboxStateForTask: %v", err)
	}
	if state.Mode != sandboxModeNet {
		t.Fatalf("state.Mode = %q, want %q", state.Mode, sandboxModeNet)
	}
}

func TestBuildTaskResultUsesDirectExecutionWhenChatHasNoSandboxState(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	resetGlobalStateForTest()

	if err := SetSandboxMode("agent-a", "chat-b", sandboxModeNet); err != nil {
		t.Fatalf("SetSandboxMode: %v", err)
	}

	task := &TaskContent{
		Timeout: 5000,
		Suffix:  "cmd",
		Type:    "cmd",
		Tid:     "direct-task",
		Cmd:     "printf direct-run",
		AgentId: "agent-a",
		Chat:    "chat-a",
	}

	result, err := BuildTaskResult(task, "/bin/sh", nil, filepath.Join(tmp, "unused", "CLI_SANDBOX.app"))
	if err != nil {
		t.Fatalf("BuildTaskResult: %v", err)
	}
	if result.Status != 0 {
		t.Fatalf("status = %d, want 0", result.Status)
	}
	if body := decodeGzipBase64(t, result.Cmd); body != "direct-run" {
		t.Fatalf("decoded cmd = %q, want direct-run", body)
	}
}

func TestSandboxStateForTaskSkipsExemptedTask(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	resetGlobalStateForTest()

	if err := SetSandboxMode("agent-a", "chat-a", sandboxModeFilePickNet); err != nil {
		t.Fatalf("SetSandboxMode: %v", err)
	}

	task := &TaskContent{
		AgentId: "agent-a",
		Chat:    "chat-a",
	}
	task.SubOps.Exempted = true

	state, err := sandboxStateForTask(nil, task)
	if err != nil {
		t.Fatalf("sandboxStateForTask: %v", err)
	}
	if state.Mode != "" || state.AllowedDir != "" {
		t.Fatalf("state = %+v, want empty", state)
	}
}

func TestSandboxStateSettingKeepsAllowedDirPerChat(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	resetGlobalStateForTest()

	if err := SetSandboxModeWithDirectory("agent-a", "chat-a", sandboxModeFilePick, "/Users/demo/Pictures"); err != nil {
		t.Fatalf("SetSandboxModeWithDirectory chat-a: %v", err)
	}
	if err := SetSandboxModeWithDirectory("agent-b", "chat-b", sandboxModeFilePickNet, "/Users/demo/code"); err != nil {
		t.Fatalf("SetSandboxModeWithDirectory chat-b: %v", err)
	}

	stateA, foundA, err := sandboxStateSetting("chat-a")
	if err != nil {
		t.Fatalf("sandboxStateSetting chat-a: %v", err)
	}
	if !foundA {
		t.Fatal("expected chat-a sandbox state")
	}
	if stateA.Mode != sandboxModeFilePick || stateA.AllowedDir != "/Users/demo/Pictures" {
		t.Fatalf("stateA = %+v", stateA)
	}

	stateB, foundB, err := sandboxStateSetting("chat-b")
	if err != nil {
		t.Fatalf("sandboxStateSetting chat-b: %v", err)
	}
	if !foundB {
		t.Fatal("expected chat-b sandbox state")
	}
	if stateB.Mode != sandboxModeFilePickNet || stateB.AllowedDir != "/Users/demo/code" {
		t.Fatalf("stateB = %+v", stateB)
	}
}

func TestHeartbeatParsesExemptedFlagFromTaskContent(t *testing.T) {
	taskJSON := `{"timeout":5000,"suffix":"cmd","type":"cmd","tid":"t-exempted","cmd":"echo hi","agentId":"a","chat":"chat-1","subOps":{"exempted":true}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ResponsePayload{
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

	task, err := Heartbeat(&http.Client{Timeout: 5 * time.Second}, server.URL, &AgentOutput{})
	if err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if task == nil {
		t.Fatal("expected task")
	}
	if !task.SubOps.Exempted {
		t.Fatal("expected task.SubOps.Exempted = true")
	}
}

func TestResolveSandboxExecutablePathFromConfig(t *testing.T) {
	root := t.TempDir()
	execPath := filepath.Join(root, "integration.app", "Contents", "MacOS", "integration")
	sandboxExec := filepath.Join(root, "integration.app", "Contents", "Helpers", sandboxModeFilePick, "CLI_SANDBOX.app", "Contents", "MacOS", "CLI_SANDBOX")
	configPath := filepath.Join(root, "integration.app", "Contents", "Resources", "config", "config.json")

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
	if err := os.WriteFile(configPath, []byte(`{"sandbox_app":"../Helpers/CLI_SANDBOX.app"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	restoreExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	defer func() { osExecutable = restoreExec }()

	got, err := resolveSandboxExecutablePath("", sandboxModeFilePick)
	if err != nil {
		t.Fatalf("resolveSandboxExecutablePath: %v", err)
	}
	if got != sandboxExec {
		t.Fatalf("got %q, want %q", got, sandboxExec)
	}
}

func TestResolveSandboxExecutablePathFromLinuxHelpersConfig(t *testing.T) {
	root := t.TempDir()
	execPath := filepath.Join(root, "linux-release", "integration")
	sandboxExec := filepath.Join(root, "linux-release", "helpers", sandboxModeFilePick, "CLI_SANDBOX")
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

	restoreExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	defer func() { osExecutable = restoreExec }()

	got, err := resolveSandboxExecutablePath("", sandboxModeFilePick)
	if err != nil {
		t.Fatalf("resolveSandboxExecutablePath: %v", err)
	}
	if got != sandboxExec {
		t.Fatalf("got %q, want %q", got, sandboxExec)
	}
}

func TestConfiguredHTTPDebugFromNestedConfig(t *testing.T) {
	root := t.TempDir()
	execPath := filepath.Join(root, "integration.app", "Contents", "MacOS", "integration")
	configPath := filepath.Join(root, "integration.app", "Contents", "Resources", "config", "config.json")

	if err := os.MkdirAll(filepath.Dir(execPath), 0o755); err != nil {
		t.Fatalf("mkdir exec dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(execPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write exec: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{"http":{"heartbeat":10,"debug":true}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	restoreExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	defer func() { osExecutable = restoreExec }()

	if !configuredHTTPDebug() {
		t.Fatal("configuredHTTPDebug() = false, want true")
	}
}

func TestConfiguredHeartbeatIntervalsFromNestedConfig(t *testing.T) {
	sleepMs, awaitMs := configuredHeartbeatIntervalsFromRaw(map[string]interface{}{
		"get": map[string]interface{}{
			"sleep": json.Number("15000"),
			"await": json.Number("30000"),
		},
	})
	if sleepMs != 15000 {
		t.Fatalf("sleep = %d, want 15000", sleepMs)
	}
	if awaitMs != 30000 {
		t.Fatalf("await = %d, want 30000", awaitMs)
	}
}

func TestConfiguredHeartbeatIntervalsRejectFractionalValues(t *testing.T) {
	sleepMs, awaitMs := configuredHeartbeatIntervalsFromRaw(map[string]interface{}{
		"get": map[string]interface{}{
			"sleep": json.Number("1.5"),
			"await": json.Number("30000.5"),
		},
	})
	if sleepMs != defaultCliGetSleepMs {
		t.Fatalf("sleep = %d, want default %d", sleepMs, defaultCliGetSleepMs)
	}
	if awaitMs != defaultCliGetAwaitMs {
		t.Fatalf("await = %d, want default %d", awaitMs, defaultCliGetAwaitMs)
	}
}

func TestConfiguredHTTPDebugIgnoresLegacyFlatKey(t *testing.T) {
	root := t.TempDir()
	execPath := filepath.Join(root, "integration.app", "Contents", "MacOS", "integration")
	configPath := filepath.Join(root, "integration.app", "Contents", "Resources", "config", "config.json")

	if err := os.MkdirAll(filepath.Dir(execPath), 0o755); err != nil {
		t.Fatalf("mkdir exec dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(execPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write exec: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{"http_debug":true}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	restoreExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	defer func() { osExecutable = restoreExec }()

	if configuredHTTPDebug() {
		t.Fatal("configuredHTTPDebug() = true, want false")
	}
}

func TestProcessTaskUsesSandboxAppWhenEnabled(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	resetGlobalStateForTest()

	sandboxExec := filepath.Join(tmp, sandboxModeFilePickNet, "CLI_SANDBOX.app", "Contents", "MacOS", "CLI_SANDBOX")
	script := "#!/bin/sh\ncmd=''\ndir=''\ntimeout=''\nwhile [ \"$#\" -gt 0 ]; do\n  case \"$1\" in\n    --cmd)\n      shift\n      cmd=\"$1\"\n      ;;\n    --allowed-dir)\n      shift\n      dir=\"$1\"\n      ;;\n    --timeout)\n      shift\n      timeout=\"$1\"\n      ;;\n  esac\n  shift\ndone\nprintf 'sandbox:%s\\ndir:%s\\ntimeout:%s\\n' \"$cmd\" \"$dir\" \"$timeout\"\n"
	if err := os.MkdirAll(filepath.Dir(sandboxExec), 0o755); err != nil {
		t.Fatalf("mkdir sandbox dir: %v", err)
	}
	if err := os.WriteFile(sandboxExec, []byte(script), 0o755); err != nil {
		t.Fatalf("write sandbox script: %v", err)
	}

	if err := SetSandboxModeWithDirectory("agent-a", "chat-a", sandboxModeFilePickNet, "/Users/demo/Pictures"); err != nil {
		t.Fatalf("SetSandboxModeWithDirectory: %v", err)
	}

	var captured PubRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cli/pub" {
			t.Fatalf("path = %s, want /cli/pub", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode publish request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ResponsePayload{Code: 200})
	}))
	defer server.Close()

	task := &TaskContent{
		Timeout: 5000,
		Suffix:  "cmd",
		Type:    "cmd",
		Tid:     "sandbox-task",
		Cmd:     "echo hello",
		AgentId: "agent-a",
		Chat:    "chat-a",
	}
	meta := &AgentOutput{
		DeviceID: "device-1",
		Agents:   []Agent{{AgentID: "agent-a", Workspace: filepath.Join(tmp, "workspace")}},
	}

	if err := ProcessTask(&http.Client{Timeout: 5 * time.Second}, server.URL, task, "/bin/sh", meta, filepath.Join(tmp, "CLI_SANDBOX.app")); err != nil {
		t.Fatalf("ProcessTask: %v", err)
	}

	if len(captured.Messages) != 1 {
		t.Fatalf("messages = %#v", captured.Messages)
	}
	var published ResultPayload
	if err := json.Unmarshal([]byte(captured.Messages[0].Content), &published); err != nil {
		t.Fatalf("parse published result: %v", err)
	}
	if published.Status != 0 || published.Tid != "sandbox-task" {
		t.Fatalf("published = %#v", published)
	}
	if body := decodeGzipBase64(t, published.Cmd); body != "sandbox:echo hello\ndir:/Users/demo/Pictures\ntimeout:5000\n" {
		t.Fatalf("decoded cmd = %q, want sandbox output with allowed dir", body)
	}
}

func TestProcessTaskSkipsSandboxForExemptedTask(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	resetGlobalStateForTest()

	sandboxExec := filepath.Join(tmp, sandboxModeNet, "CLI_SANDBOX.app", "Contents", "MacOS", "CLI_SANDBOX")
	script := "#!/bin/sh\nprintf 'sandbox should not run\\n' >&2\nexit 99\n"
	if err := os.MkdirAll(filepath.Dir(sandboxExec), 0o755); err != nil {
		t.Fatalf("mkdir sandbox dir: %v", err)
	}
	if err := os.WriteFile(sandboxExec, []byte(script), 0o755); err != nil {
		t.Fatalf("write sandbox script: %v", err)
	}

	if err := SetSandboxMode("agent-a", "chat-a", sandboxModeNet); err != nil {
		t.Fatalf("SetSandboxMode: %v", err)
	}

	var captured PubRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cli/pub" {
			t.Fatalf("path = %s, want /cli/pub", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode publish request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ResponsePayload{Code: 200})
	}))
	defer server.Close()

	task := &TaskContent{
		Timeout: 5000,
		Suffix:  "cmd",
		Type:    "cmd",
		Tid:     "sandbox-skip",
		Cmd:     "printf direct-run",
		AgentId: "agent-a",
		Chat:    "chat-a",
	}
	task.SubOps.Exempted = true
	meta := &AgentOutput{
		DeviceID: "device-1",
		Agents:   []Agent{{AgentID: "agent-a", Workspace: filepath.Join(tmp, "workspace")}},
	}

	if err := ProcessTask(&http.Client{Timeout: 5 * time.Second}, server.URL, task, "/bin/sh", meta, filepath.Join(tmp, "CLI_SANDBOX.app")); err != nil {
		t.Fatalf("ProcessTask: %v", err)
	}

	var published ResultPayload
	if err := json.Unmarshal([]byte(captured.Messages[0].Content), &published); err != nil {
		t.Fatalf("parse published result: %v", err)
	}
	if published.Status != 0 || published.Tid != "sandbox-skip" {
		t.Fatalf("published = %#v", published)
	}
	if body := decodeGzipBase64(t, published.Cmd); body != "direct-run" {
		t.Fatalf("decoded cmd = %q, want direct-run", body)
	}
}

func TestSetSandboxModeEmptyDeletesSessionState(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	resetGlobalStateForTest()

	if err := SetSandboxMode("agent-a", "chat-a", sandboxModeNet); err != nil {
		t.Fatalf("SetSandboxMode enable: %v", err)
	}
	if err := SetSandboxMode("agent-a", "chat-a", ""); err != nil {
		t.Fatalf("SetSandboxMode disable: %v", err)
	}

	state, found, err := sandboxStateSetting("chat-a")
	if err != nil {
		t.Fatalf("sandboxStateSetting: %v", err)
	}
	if found || state.Mode != "" || state.AllowedDir != "" {
		t.Fatalf("state=%+v found=%t, want empty false", state, found)
	}
}

func TestSetSandboxModeRequiresChatID(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	resetGlobalStateForTest()

	if err := SetSandboxMode("agent-a", "", sandboxModeNet); err == nil {
		t.Fatal("SetSandboxMode should require chatId")
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

func decodeGzipBase64(t *testing.T, raw string) string {
	t.Helper()
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	gz, err := gzip.NewReader(strings.NewReader(string(decoded)))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	body, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("read gzip: %v", err)
	}
	_ = gz.Close()
	return string(body)
}

func TestIsTaskExpired(t *testing.T) {
	now := time.UnixMilli(1_717_000_000_000)
	if isTaskExpired(&TaskContent{Ddl: now.Add(-time.Millisecond).UnixMilli()}, now) != true {
		t.Fatal("expired task should return true")
	}
	if isTaskExpired(&TaskContent{Ddl: now.UnixMilli()}, now) {
		t.Fatal("task at ddl boundary should not be expired")
	}
	if isTaskExpired(&TaskContent{Ddl: now.Add(time.Millisecond).UnixMilli()}, now) {
		t.Fatal("future ddl should not be expired")
	}
	if isTaskExpired(&TaskContent{}, now) {
		t.Fatal("zero ddl should not be expired")
	}
}

func TestPublishWithRetryRetriesOnce(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			http.Error(w, "temporary", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ResponsePayload{Code: 200})
	}))
	defer server.Close()

	result := &ResultPayload{Status: 0, Suffix: "cmd", Type: "cmd", Cmd: gzipBase64String("ok"), Tid: "retry-once"}
	if err := publishWithRetry(server.Client(), server.URL, result, &AgentOutput{}, time.Millisecond, 1); err != nil {
		t.Fatalf("publishWithRetry: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestPublishWithRetryStopsAfterRetryLimit(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		http.Error(w, "temporary", http.StatusInternalServerError)
	}))
	defer server.Close()

	result := &ResultPayload{Status: 0, Suffix: "cmd", Type: "cmd", Cmd: gzipBase64String("fail"), Tid: "retry-stop"}
	if err := publishWithRetry(server.Client(), server.URL, result, &AgentOutput{}, time.Millisecond, 1); err == nil {
		t.Fatal("publishWithRetry error = nil, want failure")
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestTimeoutOutput(t *testing.T) {
	if got := timeoutOutput(""); got != "[Warning: Command execution timed out.]" {
		t.Fatalf("timeoutOutput(empty) = %q", got)
	}
	if got := timeoutOutput("partial output"); got != "partial output[Warning: Command execution timed out, the returned content may be incomplete.]" {
		t.Fatalf("timeoutOutput(partial) = %q", got)
	}
	if got := timeoutOutput("\n\t "); got != "[Warning: Command execution timed out.]" {
		t.Fatalf("timeoutOutput(whitespace) = %q", got)
	}
}
