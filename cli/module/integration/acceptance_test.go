package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestAcceptanceChatFixtures(t *testing.T) {
	cases := []struct {
		name           string
		fixture        string
		model          string
		content        string
		stream         bool
		html           bool
		thinking       bool
		routerDisable  bool
		agentProvider  string
		agentThinking  bool
		agentRouterOff bool
	}{
		{
			name:           "chat case1",
			fixture:        "iteration/20260528-1/test/chat-test-case1.json",
			model:          "deepseek",
			content:        "1+1",
			stream:         true,
			html:           false,
			thinking:       false,
			routerDisable:  true,
			agentProvider:  "bigmodel",
			agentThinking:  true,
			agentRouterOff: false,
		},
		{
			name:           "chat case2 swarm enabled keeps metadata router_disable false",
			fixture:        "iteration/20260528-1/test/chat-test-case2.json",
			model:          "bigmodel",
			content:        "1+1",
			stream:         true,
			html:           true,
			thinking:       true,
			routerDisable:  false,
			agentProvider:  "deepseek",
			agentThinking:  true,
			agentRouterOff: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
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

			agentDir := filepath.Join(tmp, "agent")
			writeAcceptanceAgent(t, agentDir, tc.agentProvider, tc.agentThinking, tc.agentRouterOff)
			flushAgentCache()

			var captured map[string]any
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(body, &captured)
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, "data: [DONE]\n\n")
			}))
			defer upstream.Close()

			cfg := &Config{
				Host:         upstream.URL,
				AgentDir:     agentDir,
				Device:       "fixture-device",
				AgentCacheMs: 120000,
			}
			server := httptest.NewServer(http.HandlerFunc(handleChatCompletions(cfg, &http.Client{Timeout: 5 * time.Second})))
			defer server.Close()

			reqBody := fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":%q}],"stream":%t,"thinking":%t,"html":%t,"router_disable":%t,"metadata":{"agentId":"A","chat":"98@98","type":"page_session"}}`,
				tc.model, tc.content, tc.stream, tc.thinking, tc.html, tc.routerDisable)
			req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(reqBody))
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			resp.Body.Close()

			expected := loadAcceptanceFixture(t, tc.fixture)
			assertAcceptanceChatPayload(t, captured, expected)
		})
	}
}

func TestAcceptanceCliGetFixtures(t *testing.T) {
	cases := []struct {
		name           string
		fixture        string
		agentProvider  string
		agentThinking  bool
		agentRouterOff bool
	}{
		{
			name:           "cli get case1",
			fixture:        "iteration/20260528-1/test/cli-get-test-case1.json",
			agentProvider:  "deepseek",
			agentThinking:  false,
			agentRouterOff: true,
		},
		{
			name:           "cli get case2",
			fixture:        "iteration/20260528-1/test/cli-get-test-case2.json",
			agentProvider:  "bigmodel",
			agentThinking:  true,
			agentRouterOff: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
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

			agentDir := filepath.Join(tmp, "agent")
			writeAcceptanceAgent(t, agentDir, tc.agentProvider, tc.agentThinking, tc.agentRouterOff)
			flushAgentCache()
			metadata, err := getAgentOutput(agentDir, "fixture-device", time.Minute)
			if err != nil {
				t.Fatalf("getAgentOutput: %v", err)
			}

			var captured map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(body, &captured)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"code":200,"choices":[{"message":{"role":"assistant","content":""}}]}`)
			}))
			defer server.Close()

			if _, err := heartbeat(&http.Client{Timeout: 5 * time.Second}, server.URL, metadata, nil); err != nil {
				t.Fatalf("heartbeat: %v", err)
			}

			expected := loadAcceptanceFixture(t, tc.fixture)
			assertAcceptanceCLIGetPayload(t, captured, expected)
		})
	}
}

func writeAcceptanceAgent(t *testing.T, agentRoot, provider string, thinking, routerDisable bool) {
	t.Helper()
	agentPath := filepath.Join(agentRoot, "A")
	if err := os.MkdirAll(filepath.Join(agentPath, "skills", "skill-creator"), 0o755); err != nil {
		t.Fatalf("mkdir agent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentPath, "SOUL.md"), []byte("fixture soul"), 0o644); err != nil {
		t.Fatalf("write SOUL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentPath, "USER.md"), []byte("fixture user"), 0o644); err != nil {
		t.Fatalf("write USER.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentPath, "skills", "skill-creator", "SKILL.md"), []byte("# skill-creator\n\nfixture skill"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	configBody := fmt.Sprintf(`{"provider":%q,"thinking":%t,"router_disable":%t}`, provider, thinking, routerDisable)
	if err := os.WriteFile(filepath.Join(agentPath, "config.json"), []byte(configBody), 0o644); err != nil {
		t.Fatalf("write config.json: %v", err)
	}
}

func loadAcceptanceFixture(t *testing.T, relativePath string) map[string]any {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve acceptance fixture base path")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(filename), relativePath))
	if err != nil {
		t.Fatalf("read fixture %s: %v", relativePath, err)
	}
	replaced := strings.NewReplacer(
		": SOUL.md的内容", `: "SOUL.md的内容"`,
		": USER.md的内容", `: "USER.md的内容"`,
	).Replace(string(body))
	var payload map[string]any
	if err := json.Unmarshal([]byte(replaced), &payload); err != nil {
		t.Fatalf("unmarshal fixture %s: %v", relativePath, err)
	}
	return payload
}

func assertAcceptanceChatPayload(t *testing.T, actual, expected map[string]any) {
	t.Helper()
	if actual["model"] != expected["model"] {
		t.Fatalf("model = %v, want %v", actual["model"], expected["model"])
	}
	if actual["stream"] != expected["stream"] {
		t.Fatalf("stream = %v, want %v", actual["stream"], expected["stream"])
	}
	assertAcceptanceMessages(t, actual["messages"], expected["messages"])
	if _, exists := actual["message"]; exists {
		t.Fatalf("message should be absent: %v", actual["message"])
	}
	if _, exists := actual["thinking"]; exists {
		t.Fatalf("top-level thinking should be absent: %v", actual["thinking"])
	}
	if _, exists := actual["html"]; exists {
		t.Fatalf("top-level html should be absent: %v", actual["html"])
	}
	if _, exists := actual["router_disable"]; exists {
		t.Fatalf("top-level router_disable should be absent: %v", actual["router_disable"])
	}
	assertAcceptanceMetadataFlags(t, actual["metadata"], expected["metadata"], true)
}

func assertAcceptanceCLIGetPayload(t *testing.T, actual, expected map[string]any) {
	t.Helper()
	assertAcceptanceMessages(t, actual["messages"], expected["messages"])
	if _, exists := actual["message"]; exists {
		t.Fatalf("message should be absent: %v", actual["message"])
	}
	if _, exists := actual["stream"]; exists {
		t.Fatalf("stream should be absent: %v", actual["stream"])
	}
	assertAcceptanceMetadataFlags(t, actual["metadata"], expected["metadata"], false)
}

func assertAcceptanceMessages(t *testing.T, actualRaw, expectedRaw any) {
	t.Helper()
	actual, ok := actualRaw.([]any)
	if !ok || len(actual) != 1 {
		t.Fatalf("messages = %#v, want one message", actualRaw)
	}
	expected, ok := expectedRaw.([]any)
	if !ok || len(expected) != 1 {
		t.Fatalf("fixture messages = %#v, want one message", expectedRaw)
	}
	actualMsg, ok := actual[0].(map[string]any)
	if !ok {
		t.Fatalf("actual message malformed: %#v", actual[0])
	}
	expectedMsg, ok := expected[0].(map[string]any)
	if !ok {
		t.Fatalf("expected message malformed: %#v", expected[0])
	}
	if actualMsg["role"] != expectedMsg["role"] || actualMsg["content"] != expectedMsg["content"] {
		t.Fatalf("message = %#v, want %#v", actualMsg, expectedMsg)
	}
}

func assertAcceptanceMetadataFlags(t *testing.T, actualRaw, expectedRaw any, includeChatFlags bool) {
	t.Helper()
	actual, ok := actualRaw.(map[string]any)
	if !ok {
		t.Fatalf("metadata malformed: %#v", actualRaw)
	}
	expected, ok := expectedRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected metadata malformed: %#v", expectedRaw)
	}
	actualAgents, ok := actual["agents"].([]any)
	if !ok || len(actualAgents) == 0 {
		t.Fatalf("metadata.agents = %#v, want at least one agent", actual["agents"])
	}
	expectedAgents, ok := expected["agents"].([]any)
	if !ok || len(expectedAgents) == 0 {
		t.Fatalf("fixture metadata.agents = %#v, want at least one agent", expected["agents"])
	}
	actualAgent, ok := actualAgents[0].(map[string]any)
	if !ok {
		t.Fatalf("actual agent malformed: %#v", actualAgents[0])
	}
	expectedAgent, ok := expectedAgents[0].(map[string]any)
	if !ok {
		t.Fatalf("expected agent malformed: %#v", expectedAgents[0])
	}
	for _, key := range []string{"provider", "thinking", "router_disable"} {
		if actualAgent[key] != expectedAgent[key] {
			t.Fatalf("agent %s = %#v, want %#v", key, actualAgent[key], expectedAgent[key])
		}
	}
	if includeChatFlags {
		for _, key := range []string{"thinking", "html", "router_disable"} {
			if actual[key] != expected[key] {
				if key == "router_disable" {
					t.Fatalf("metadata.router_disable = %#v, want %#v (SWARM 开启时必须为 false，关闭时必须为 true)", actual[key], expected[key])
				}
				t.Fatalf("metadata %s = %#v, want %#v", key, actual[key], expected[key])
			}
		}
	}
}
