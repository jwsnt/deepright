package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	_ "github.com/glebarez/go-sqlite"

	"connect/connectsvc"
)

func TestConnectHelpIncludesPluginManual(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	if code := connectsvc.RunCLI([]string{"help"}, stdout, stderr); code != 0 {
		t.Fatalf("help exit code = %d, stderr = %s", code, stderr.String())
	}

	text := stdout.String()
	for _, want := range []string{
		"Manual:",
		"Prefer integration top-level commands",
		"Plugin workflow:",
		"meta-get --key feishu",
		"../plugins/feishu start --connect-bin ./connect",
		"Plugin notes:",
		"shared SQLite connection pool",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("help output missing %q\n%s", want, text)
		}
	}
}

func TestServiceMetaRequestResponseFlow(t *testing.T) {
	agentDir, dbPath, artifactPath := testPaths(t)
	seedModelToken(t, dbPath, "OpenAI", "sk-test")

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   dbPath,
		AgentDir: agentDir,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	meta, err := svc.CreateMeta(connectsvc.MetaInput{
		Key:           "feishu",
		Meta:          `{"token":"abc"}`,
		Stream:        true,
		Callback:      "./feishu",
		AgentID:       "A",
		ChatID:        "chat-001",
		Model:         "OpenAI",
		Thinking:      true,
		RouterDisable: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if meta.Name != "feishu" || !meta.Stream || meta.ChatID != "chat-001" || meta.RouterDisable {
		t.Fatalf("unexpected meta: %+v", meta)
	}

	updatedMetaJSON := `{"token":"xyz"}`
	updatedCallback := "./new-feishu"
	stream := false
	updated, err := svc.UpdateMeta("feishu", connectsvc.MetaUpdate{
		Meta:     &updatedMetaJSON,
		Callback: &updatedCallback,
		Stream:   &stream,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Meta != updatedMetaJSON || updated.Stream {
		t.Fatalf("unexpected updated meta: %+v", updated)
	}
	if filepath.Base(updated.Callback) != "feishu" {
		t.Fatalf("unexpected normalized callback: %+v", updated)
	}

	req, err := svc.AddRequest(connectsvc.RequestInput{
		ExternalID:     "ext-1",
		Key:            "feishu",
		Request:        "HELLO WORLD",
		Artifacts:      artifactPath,
		RawRequest:     `{"text":"HELLO WORLD"}`,
		ResponseSchema: `{"type":"object","properties":{"reply":{"type":"string"}}}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if req.ID == 0 || !strings.Contains(req.Artifacts, "artifact.txt") {
		t.Fatalf("unexpected request: %+v", req)
	}
	if req.ExternalID != "ext-1" || req.RawRequest != `{"text":"HELLO WORLD"}` {
		t.Fatalf("request fields not persisted: %+v", req)
	}
	if req.ResponseSchema == "" {
		t.Fatalf("request response schema not persisted: %+v", req)
	}
	if req.Status != connectsvc.RequestStatusPending {
		t.Fatalf("request status = %d, want pending", req.Status)
	}

	resp, err := svc.AddResponse(connectsvc.ResponseInput{
		Key:       "feishu",
		RequestID: req.ID,
		Response:  "HELLO BACK",
		Artifacts: artifactPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.RequestID != req.ID || resp.Response != "HELLO BACK" {
		t.Fatalf("unexpected response: %+v", resp)
	}

	requests, err := svc.ListRequests(connectsvc.RequestFilter{Key: "feishu", AfterID: 0, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 || requests[0].ID != req.ID {
		t.Fatalf("unexpected requests: %+v", requests)
	}
	if requests[0].ExternalID != "ext-1" || requests[0].RawRequest != `{"text":"HELLO WORLD"}` {
		t.Fatalf("request fields missing from list: %+v", requests[0])
	}
	if requests[0].ResponseSchema == "" {
		t.Fatalf("request schema missing from list: %+v", requests[0])
	}
	if requests[0].Status != connectsvc.RequestStatusReplied {
		t.Fatalf("request status after response = %d, want replied", requests[0].Status)
	}

	responses, err := svc.ListResponses(connectsvc.ResponseFilter{Key: "feishu", RequestID: req.ID, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(responses) != 1 || responses[0].ID != resp.ID {
		t.Fatalf("unexpected responses: %+v", responses)
	}
}

func TestDeleteMetaBlocksNewTraffic(t *testing.T) {
	agentDir, dbPath, _ := testPaths(t)
	seedModelToken(t, dbPath, "OpenAI", "sk-test")

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   dbPath,
		AgentDir: agentDir,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	_, err = svc.CreateMeta(connectsvc.MetaInput{
		Key:      "wechat",
		Meta:     `{"token":"abc"}`,
		Callback: "./wechat",
		AgentID:  "A",
		Model:    "OpenAI",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DeleteMeta("wechat"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddRequest(connectsvc.RequestInput{Key: "wechat", Request: "HELLO"}); err == nil {
		t.Fatal("expected add request to fail after delete")
	}
}

func TestRequestExternalIDIsUniquePerName(t *testing.T) {
	agentDir, dbPath, _ := testPaths(t)
	seedModelToken(t, dbPath, "OpenAI", "sk-test")

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   dbPath,
		AgentDir: agentDir,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	_, err = svc.CreateMeta(connectsvc.MetaInput{
		Key:      "dup",
		Meta:     `{"token":"abc"}`,
		Callback: "./dup",
		AgentID:  "A",
		Model:    "OpenAI",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.AddRequest(connectsvc.RequestInput{
		ExternalID: "msg-1",
		Key:        "dup",
		Request:    "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.AddRequest(connectsvc.RequestInput{
		ExternalID: "msg-1",
		Key:        "dup",
		Request:    "hello again",
	})
	if err == nil || !strings.Contains(err.Error(), "connect request already exists") {
		t.Fatalf("unexpected duplicate error: %v", err)
	}
}

func TestRequestStatusFilterAndExplicitStatus(t *testing.T) {
	agentDir, dbPath, _ := testPaths(t)
	seedModelToken(t, dbPath, "OpenAI", "sk-test")

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   dbPath,
		AgentDir: agentDir,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	_, err = svc.CreateMeta(connectsvc.MetaInput{
		Key:      "status-demo",
		Meta:     `{"token":"abc"}`,
		Callback: "./status-demo",
		AgentID:  "A",
		Model:    "OpenAI",
	})
	if err != nil {
		t.Fatal(err)
	}

	started := connectsvc.RequestStatusStarted
	startedReq, err := svc.AddRequest(connectsvc.RequestInput{
		Key:     "status-demo",
		Request: "started request",
		Status:  &started,
	})
	if err != nil {
		t.Fatal(err)
	}
	if startedReq.Status != connectsvc.RequestStatusStarted {
		t.Fatalf("started request status = %d", startedReq.Status)
	}

	pendingReq, err := svc.AddRequest(connectsvc.RequestInput{
		Key:     "status-demo",
		Request: "pending request",
	})
	if err != nil {
		t.Fatal(err)
	}
	if pendingReq.Status != connectsvc.RequestStatusPending {
		t.Fatalf("pending request status = %d", pendingReq.Status)
	}

	filtered, err := svc.ListRequests(connectsvc.RequestFilter{
		Key:    "status-demo",
		Limit:  10,
		Status: &started,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 1 || filtered[0].ID != startedReq.ID {
		t.Fatalf("unexpected filtered requests: %+v", filtered)
	}

	replied := connectsvc.RequestStatusReplied
	repliedReq, err := svc.AddRequest(connectsvc.RequestInput{
		Key:     "status-demo",
		Request: "replied request",
		Status:  &replied,
	})
	if err != nil {
		t.Fatal(err)
	}
	if repliedReq.Status != connectsvc.RequestStatusReplied {
		t.Fatalf("replied request status = %d", repliedReq.Status)
	}

	repliedFiltered, err := svc.ListRequests(connectsvc.RequestFilter{
		Key:    "status-demo",
		Limit:  10,
		Status: &replied,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(repliedFiltered) != 1 || repliedFiltered[0].ID != repliedReq.ID {
		t.Fatalf("unexpected replied filtered requests: %+v", repliedFiltered)
	}

	expired := connectsvc.RequestStatusExpired
	expiredReq, err := svc.AddRequest(connectsvc.RequestInput{
		Key:     "status-demo",
		Request: "expired request",
		Status:  &expired,
	})
	if err != nil {
		t.Fatal(err)
	}
	if expiredReq.Status != connectsvc.RequestStatusExpired {
		t.Fatalf("expired request status = %d", expiredReq.Status)
	}

	expiredFiltered, err := svc.ListRequests(connectsvc.RequestFilter{
		Key:    "status-demo",
		Limit:  10,
		Status: &expired,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(expiredFiltered) != 1 || expiredFiltered[0].ID != expiredReq.ID {
		t.Fatalf("unexpected expired filtered requests: %+v", expiredFiltered)
	}
}

func TestRunCLIMetaCreateAndRequestList(t *testing.T) {
	agentDir, dbPath, artifactPath := testPaths(t)
	seedModelToken(t, dbPath, "OpenAI", "sk-test")

	server := startTestConnectServer(t, agentDir, dbPath)
	defer server.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := connectsvc.RunCLI([]string{
		"meta-create",
		"--addr", server.URL,
		"--db", dbPath,
		"--agent-dir", agentDir,
		"--key", "lark",
		"--meta", `{"token":"abc"}`,
		"--stream", "true",
		"--callback", "./lark",
		"--agent", "A",
		"--chatId", "chat-001",
		"--model", "OpenAI",
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("meta-create failed: %s", stderr.String())
	}
	var meta connectsvc.Meta
	if err := json.Unmarshal(stdout.Bytes(), &meta); err != nil {
		t.Fatal(err)
	}
	if meta.Name != "lark" || meta.Model != "OpenAI" {
		t.Fatalf("unexpected meta: %+v", meta)
	}

	stdout.Reset()
	stderr.Reset()
	code = connectsvc.RunCLI([]string{
		"add-request",
		"--addr", server.URL,
		"--db", dbPath,
		"--agent-dir", agentDir,
		"--key", "lark",
		"--external-id", "msg-1",
		"--request", "HELLO WORLD",
		"--raw-request", `{"text":"HELLO WORLD"}`,
		"--artifacts", artifactPath,
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("add-request failed: %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = connectsvc.RunCLI([]string{
		"request-list",
		"--addr", server.URL,
		"--db", dbPath,
		"--agent-dir", agentDir,
		"--key", "lark",
		"--limit", "5",
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("request-list failed: %s", stderr.String())
	}
	var requests []connectsvc.Request
	if err := json.Unmarshal(stdout.Bytes(), &requests); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 || requests[0].Name != "lark" {
		t.Fatalf("unexpected requests: %+v", requests)
	}
	if requests[0].ExternalID != "msg-1" || requests[0].RawRequest != `{"text":"HELLO WORLD"}` {
		t.Fatalf("unexpected request extra fields: %+v", requests[0])
	}
	if requests[0].Status != connectsvc.RequestStatusPending {
		t.Fatalf("unexpected request status: %+v", requests[0])
	}
}

func TestListMetaReturnsParsedConfiguredMeta(t *testing.T) {
	agentDir, dbPath, _ := testPaths(t)
	seedModelToken(t, dbPath, "OpenAI", "sk-test")

	server := startTestConnectServer(t, agentDir, dbPath)
	defer server.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := connectsvc.RunCLI([]string{
		"meta-create",
		"--addr", server.URL,
		"--key", "feishu",
		"--meta", `{"appId":"cli-app","appSecret":"cli-secret"}`,
		"--stream", "true",
		"--callback", "./feishu",
		"--agent", "A",
		"--chatId", "chat-001",
		"--model", "OpenAI",
		"--thinking", "true",
		"--router_disable", "false",
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("meta-create failed: %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = connectsvc.RunCLI([]string{
		"list-meta",
		"--addr", server.URL,
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("list-meta failed: %s", stderr.String())
	}

	var items []connectsvc.MetaConfig
	if err := json.Unmarshal(stdout.Bytes(), &items); err != nil {
		t.Fatalf("decode list-meta output: %v; raw=%s", err, stdout.String())
	}
	if len(items) != 1 {
		t.Fatalf("unexpected list-meta size: %+v", items)
	}
	if items[0].Key != "feishu" {
		t.Fatalf("unexpected meta config key: %+v", items[0])
	}
	if items[0].Name != "feishu" || !items[0].Stream || !filepath.IsAbs(items[0].Callback) {
		t.Fatalf("unexpected meta config header: %+v", items[0])
	}
	if items[0].AgentID != "A" || items[0].ChatID != "chat-001" || items[0].Model != "OpenAI" || !items[0].Thinking || items[0].RouterDisable {
		t.Fatalf("unexpected meta config runtime fields: %+v", items[0])
	}
	if got := fmt.Sprint(items[0].Meta["appId"]); got != "cli-app" {
		t.Fatalf("unexpected parsed meta: %+v", items[0].Meta)
	}
	if got := fmt.Sprint(items[0].Meta["appSecret"]); got != "cli-secret" {
		t.Fatalf("unexpected parsed meta secret: %+v", items[0].Meta)
	}
}

func TestMetaGetByKeyReturnsConfiguredPluginView(t *testing.T) {
	agentDir, dbPath, _ := testPaths(t)
	seedModelToken(t, dbPath, "OpenAI", "sk-test")

	server := startTestConnectServer(t, agentDir, dbPath)
	defer server.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := connectsvc.RunCLI([]string{
		"meta-create",
		"--addr", server.URL,
		"--key", "feishu",
		"--meta", `{"appId":"cli-app","appSecret":"cli-secret"}`,
		"--stream", "true",
		"--callback", "./feishu",
		"--agent", "A",
		"--chatId", "chat-001",
		"--model", "OpenAI",
		"--thinking", "true",
		"--router_disable", "false",
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("meta-create failed: %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = connectsvc.RunCLI([]string{
		"meta-get",
		"--addr", server.URL,
		"--key", "feishu",
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("meta-get --key failed: %s", stderr.String())
	}

	var item connectsvc.MetaConfig
	if err := json.Unmarshal(stdout.Bytes(), &item); err != nil {
		t.Fatalf("decode meta-get output: %v; raw=%s", err, stdout.String())
	}
	if item.Key != "feishu" || item.Name != "feishu" {
		t.Fatalf("unexpected meta-get identity: %+v", item)
	}
	if got := fmt.Sprint(item.Meta["appId"]); got != "cli-app" {
		t.Fatalf("unexpected meta-get config: %+v", item.Meta)
	}
	if !item.Stream || !item.Thinking || item.RouterDisable {
		t.Fatalf("unexpected meta-get runtime flags: %+v", item)
	}
}

func TestAddRequestResolvesMetaByPluginKey(t *testing.T) {
	agentDir, dbPath, artifactPath := testPaths(t)
	seedModelToken(t, dbPath, "OpenAI", "sk-test")

	pluginDir := t.TempDir()
	pluginPath := filepath.Join(pluginDir, "feishu")
	writeLocalPluginScript(t, pluginPath, `{"key":"feishu","name":"飞书"}`, `["appId","appSecret"]`)

	server := startTestConnectServer(t, agentDir, dbPath)
	defer server.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := connectsvc.RunCLI([]string{
		"meta-create",
		"--addr", server.URL,
		"--db", dbPath,
		"--agent-dir", agentDir,
		"--key", "feishu",
		"--meta", `{"appId":"cli-app","appSecret":"cli-secret"}`,
		"--stream", "true",
		"--callback", pluginPath,
		"--agent", "A",
		"--chatId", "chat-001",
		"--model", "OpenAI",
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("meta-create failed: %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = connectsvc.RunCLI([]string{
		"add-request",
		"--addr", server.URL,
		"--db", dbPath,
		"--agent-dir", agentDir,
		"--key", "feishu",
		"--external-id", "msg-key-1",
		"--content", "HELLO WORLD",
		"--original", `{"text":"HELLO WORLD"}`,
		"--artifacts", artifactPath,
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("add-request by key failed: %s", stderr.String())
	}

	var req connectsvc.Request
	if err := json.Unmarshal(stdout.Bytes(), &req); err != nil {
		t.Fatalf("decode add-request: %v raw=%s", err, stdout.String())
	}
	if req.Key != "feishu" {
		t.Fatalf("request key = %q, want feishu", req.Key)
	}
	if req.Name != "feishu" {
		t.Fatalf("request name = %q, want feishu", req.Name)
	}
}

func TestAddRequestRejectsPluginDisplayNameInKeyFlag(t *testing.T) {
	agentDir, dbPath, artifactPath := testPaths(t)
	seedModelToken(t, dbPath, "OpenAI", "sk-test")

	pluginDir := t.TempDir()
	pluginPath := filepath.Join(pluginDir, "feishu")
	writeLocalPluginScript(t, pluginPath, `{"key":"feishu","name":"飞书"}`, `["appId","appSecret"]`)

	server := startTestConnectServer(t, agentDir, dbPath)
	defer server.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := connectsvc.RunCLI([]string{
		"meta-create",
		"--addr", server.URL,
		"--db", dbPath,
		"--agent-dir", agentDir,
		"--key", "feishu",
		"--meta", `{"appId":"cli-app","appSecret":"cli-secret"}`,
		"--stream", "true",
		"--callback", pluginPath,
		"--agent", "A",
		"--chatId", "chat-001",
		"--model", "OpenAI",
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("meta-create failed: %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = connectsvc.RunCLI([]string{
		"add-request",
		"--addr", server.URL,
		"--db", dbPath,
		"--agent-dir", agentDir,
		"--key", "飞书",
		"--external-id", "msg-display-key-1",
		"--content", "HELLO WORLD",
		"--original", `{"text":"HELLO WORLD"}`,
		"--artifacts", artifactPath,
	}, stdout, stderr)
	if code == 0 {
		t.Fatalf("add-request by display-name key unexpectedly succeeded: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "connect meta not found") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestMetaCreateNormalizesCallbackToRuntimePluginsDir(t *testing.T) {
	agentDir, dbPath, _ := testPaths(t)
	seedModelToken(t, dbPath, "OpenAI", "sk-test")

	pluginDir := t.TempDir()
	pluginPath := filepath.Join(pluginDir, "feishu")
	writeLocalPluginScript(t, pluginPath, `{"key":"feishu","name":"飞书"}`, `["appId","appSecret"]`)
	restore := connectsvcTestOverridePluginDir(pluginDir)
	defer restore()

	server := startTestConnectServer(t, agentDir, dbPath)
	defer server.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := connectsvc.RunCLI([]string{
		"meta-create",
		"--addr", server.URL,
		"--db", dbPath,
		"--agent-dir", agentDir,
		"--key", "feishu",
		"--meta", `{"appId":"cli-app","appSecret":"cli-secret"}`,
		"--stream", "true",
		"--callback", "./legacy/feishu",
		"--agent", "A",
		"--chatId", "chat-001",
		"--model", "OpenAI",
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("meta-create failed: %s", stderr.String())
	}

	var item connectsvc.Meta
	if err := json.Unmarshal(stdout.Bytes(), &item); err != nil {
		t.Fatalf("decode meta-create output: %v raw=%s", err, stdout.String())
	}
	wantCallback := filepath.Join(pluginDir, "feishu")
	if item.Callback != wantCallback {
		t.Fatalf("callback = %q, want %q", item.Callback, wantCallback)
	}
	if item.Name != "feishu" {
		t.Fatalf("name = %q, want feishu", item.Name)
	}
}

func TestMetaUpdateNormalizesCallbackToRuntimePluginsDir(t *testing.T) {
	agentDir, dbPath, _ := testPaths(t)
	seedModelToken(t, dbPath, "OpenAI", "sk-test")

	pluginDir := t.TempDir()
	pluginPath := filepath.Join(pluginDir, "feishu")
	writeLocalPluginScript(t, pluginPath, `{"key":"feishu","name":"飞书"}`, `["appId","appSecret"]`)
	restore := connectsvcTestOverridePluginDir(pluginDir)
	defer restore()

	server := startTestConnectServer(t, agentDir, dbPath)
	defer server.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := connectsvc.RunCLI([]string{
		"meta-create",
		"--addr", server.URL,
		"--db", dbPath,
		"--agent-dir", agentDir,
		"--key", "feishu",
		"--meta", `{"appId":"cli-app","appSecret":"cli-secret"}`,
		"--stream", "true",
		"--callback", "./old/feishu",
		"--agent", "A",
		"--chatId", "chat-001",
		"--model", "OpenAI",
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("meta-create failed: %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = connectsvc.RunCLI([]string{
		"meta-update",
		"--addr", server.URL,
		"--key", "feishu",
		"--meta", `{"appId":"new-app","appSecret":"new-secret"}`,
		"--callback", "./another/path/feishu",
		"--agent", "A",
		"--chatId", "chat-002",
		"--model", "OpenAI",
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("meta-update failed: %s", stderr.String())
	}

	var item connectsvc.Meta
	if err := json.Unmarshal(stdout.Bytes(), &item); err != nil {
		t.Fatalf("decode meta-update output: %v raw=%s", err, stdout.String())
	}
	wantCallback := filepath.Join(pluginDir, "feishu")
	if item.Callback != wantCallback {
		t.Fatalf("callback = %q, want %q", item.Callback, wantCallback)
	}
}

func TestMetaUpdateRequiresKey(t *testing.T) {
	agentDir, dbPath, _ := testPaths(t)
	seedModelToken(t, dbPath, "OpenAI", "sk-test")

	server := startTestConnectServer(t, agentDir, dbPath)
	defer server.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := connectsvc.RunCLI([]string{
		"meta-update",
		"--addr", server.URL,
		"--meta", `{"appId":"new-app"}`,
	}, stdout, stderr)
	if code == 0 {
		t.Fatal("meta-update should fail without key")
	}
	if !strings.Contains(stderr.String(), "key is required") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func writeLocalPluginScript(t *testing.T, path, nameOutput, paramOutput string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Fatal("plugin shell script test is not supported on windows")
	}
	paramOutput = normalizeLocalPluginParamOutput(t, paramOutput)
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"name\" ]; then\n" +
		"  printf '%s\\n' '" + escapeLocalSingleQuotes(nameOutput) + "'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"param\" ]; then\n" +
		"  printf '%s\\n' '" + escapeLocalSingleQuotes(paramOutput) + "'\n" +
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

func normalizeLocalPluginParamOutput(t *testing.T, raw string) string {
	t.Helper()
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}

	var stringItems []string
	if err := json.Unmarshal([]byte(raw), &stringItems); err != nil {
		return raw
	}

	items := make([]map[string]string, 0, len(stringItems))
	for _, item := range stringItems {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		items = append(items, map[string]string{item: ""})
	}
	normalized, err := json.Marshal(items)
	if err != nil {
		t.Fatalf("normalize plugin param output: %v", err)
	}
	return string(normalized)
}

func escapeLocalSingleQuotes(input string) string {
	return strings.ReplaceAll(input, "'", `'"'"'`)
}

func connectsvcTestOverridePluginDir(dir string) func() {
	restoreDir := connectsvc.SetPluginDirResolverForTest(func() (string, error) { return dir, nil })
	restoreExe := connectsvc.SetOSExecutableForTest(func() (string, error) { return filepath.Join(dir, "..", "connect"), nil })
	return func() {
		restoreExe()
		restoreDir()
	}
}

func TestRunCLIFailsWhenServiceNotRunning(t *testing.T) {
	agentDir, dbPath, _ := testPaths(t)
	seedModelToken(t, dbPath, "OpenAI", "sk-test")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := connectsvc.RunCLI([]string{
		"meta-list",
		"--addr", "http://127.0.0.1:18999",
		"--db", dbPath,
		"--agent-dir", agentDir,
	}, stdout, stderr)
	if code == 0 {
		t.Fatal("expected command to fail when service is not running")
	}
	if !strings.Contains(stderr.String(), "start it first with ./connect start") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunCLIHTTPResponseFlow(t *testing.T) {
	agentDir, dbPath, artifactPath := testPaths(t)
	seedModelToken(t, dbPath, "OpenAI", "sk-test")
	server := startTestConnectServer(t, agentDir, dbPath)
	defer server.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := connectsvc.RunCLI([]string{
		"meta-create",
		"--addr", server.URL,
		"--key", "feishu",
		"--meta", `{"token":"abc"}`,
		"--stream", "true",
		"--callback", "./feishu",
		"--agent", "A",
		"--model", "OpenAI",
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("meta-create failed: %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = connectsvc.RunCLI([]string{
		"add-request",
		"--addr", server.URL,
		"--key", "feishu",
		"--request", "HELLO WORLD",
		"--artifacts", artifactPath,
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("add-request failed: %s", stderr.String())
	}
	var req connectsvc.Request
	if err := json.Unmarshal(stdout.Bytes(), &req); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	code = connectsvc.RunCLI([]string{
		"add-response",
		"--addr", server.URL,
		"--key", "feishu",
		"--request-id", fmt.Sprint(req.ID),
		"--response", "HELLO BACK",
		"--artifacts", artifactPath,
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("add-response failed: %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = connectsvc.RunCLI([]string{
		"response-list",
		"--addr", server.URL,
		"--key", "feishu",
		"--request-id", fmt.Sprint(req.ID),
		"--limit", "10",
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("response-list failed: %s", stderr.String())
	}
	var responses []connectsvc.Response
	if err := json.Unmarshal(stdout.Bytes(), &responses); err != nil {
		t.Fatal(err)
	}
	if len(responses) != 1 || responses[0].RequestID != req.ID {
		t.Fatalf("unexpected responses: %+v", responses)
	}
}

func TestMetaRequiresRegisteredModel(t *testing.T) {
	agentDir, dbPath, _ := testPaths(t)

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   dbPath,
		AgentDir: agentDir,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	_, err = svc.CreateMeta(connectsvc.MetaInput{
		Key:      "bad",
		Meta:     `{"token":"abc"}`,
		Callback: "./bad",
		AgentID:  "A",
		Model:    "MissingModel",
	})
	if err == nil || !strings.Contains(err.Error(), "model not registered") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateRequestStatus(t *testing.T) {
	agentDir, dbPath, _ := testPaths(t)
	seedModelToken(t, dbPath, "OpenAI", "sk-test")

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   dbPath,
		AgentDir: agentDir,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	_, err = svc.CreateMeta(connectsvc.MetaInput{
		Key:      "status-meta",
		Meta:     `{"token":"abc"}`,
		Callback: "./status-meta",
		AgentID:  "A",
		Model:    "OpenAI",
	})
	if err != nil {
		t.Fatal(err)
	}

	req, err := svc.AddRequest(connectsvc.RequestInput{
		Key:     "status-meta",
		Request: "HELLO",
	})
	if err != nil {
		t.Fatal(err)
	}

	from := connectsvc.RequestStatusPending
	updated, err := svc.UpdateRequestStatus(connectsvc.RequestStatusUpdate{
		ID:     req.ID,
		Key:    "status-meta",
		From:   &from,
		To:     connectsvc.RequestStatusStarted,
		Strict: true,
	})
	if err != nil {
		t.Fatalf("update status: %v", err)
	}
	if updated.Status != connectsvc.RequestStatusStarted {
		t.Fatalf("status = %d, want started", updated.Status)
	}

	if _, err := svc.UpdateRequestStatus(connectsvc.RequestStatusUpdate{
		ID:     req.ID,
		Key:    "status-meta",
		From:   &from,
		To:     connectsvc.RequestStatusCompleted,
		Strict: true,
	}); err == nil {
		t.Fatal("expected strict update to fail on status mismatch")
	}
}

func TestServiceMigratesLegacyRepliedStatusToNewCode(t *testing.T) {
	agentDir, dbPath, _ := testPaths(t)
	seedModelToken(t, dbPath, "OpenAI", "sk-test")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS connect_meta (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			meta TEXT NOT NULL,
			stream INTEGER NOT NULL DEFAULT 0,
			callback TEXT NOT NULL,
			agent_id TEXT NOT NULL,
			chat_id TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL,
			thinking INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			deleted_at TEXT NOT NULL DEFAULT ''
		)
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS connect_request (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			external_id TEXT NOT NULL DEFAULT '',
			request TEXT NOT NULL,
			artifacts TEXT NOT NULL DEFAULT '',
			raw_request TEXT NOT NULL DEFAULT '',
			response_schema TEXT NOT NULL DEFAULT '',
			status INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS connect_response (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			request_id INTEGER NOT NULL,
			response TEXT NOT NULL,
			artifacts TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)
	`); err != nil {
		t.Fatal(err)
	}

	now := time.Now().Format(time.RFC3339)
	res, err := db.Exec(`INSERT INTO connect_request (name, request, status, created_at) VALUES (?,?,?,?)`, "legacy", "hello", 3, now)
	if err != nil {
		t.Fatal(err)
	}
	reqID64, _ := res.LastInsertId()
	reqID := int(reqID64)
	if _, err := db.Exec(`INSERT INTO connect_response (name, request_id, response, created_at) VALUES (?,?,?,?)`, "legacy", reqID, "world", now); err != nil {
		t.Fatal(err)
	}

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   dbPath,
		AgentDir: agentDir,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	requests, err := svc.ListRequests(connectsvc.RequestFilter{Key: "legacy", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 {
		t.Fatalf("unexpected requests: %+v", requests)
	}
	if requests[0].Status != connectsvc.RequestStatusReplied {
		t.Fatalf("legacy migrated status = %d, want replied(%d)", requests[0].Status, connectsvc.RequestStatusReplied)
	}
}

func testPaths(t *testing.T) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	agentDir := filepath.Join(root, "agents")
	if err := os.MkdirAll(filepath.Join(agentDir, "A"), 0o755); err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(root, "artifact.txt")
	if err := os.WriteFile(artifactPath, []byte("artifact"), 0o644); err != nil {
		t.Fatal(err)
	}
	return agentDir, filepath.Join(root, "data"), artifactPath
}

func seedModelToken(t *testing.T, dbPath, model, token string) {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS token_store (model TEXT PRIMARY KEY, token TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO token_store (model, token, updated_at) VALUES (?,?,?) ON CONFLICT(model) DO UPDATE SET token=excluded.token, updated_at=excluded.updated_at`, model, token, time.Now().Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
}

func startTestConnectServer(t *testing.T, agentDir, dbPath string) *httptest.Server {
	t.Helper()
	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   dbPath,
		AgentDir: agentDir,
		CacheTTL: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(connectsvc.NewHTTPHandler(svc))
	t.Cleanup(func() {
		server.Close()
		_ = svc.Close()
	})
	return server
}
