package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIntegrationAPIWhisperCreateCLI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/whisper/tasks" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		var request whisperTaskCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.AgentID != "demo-agent" || request.Scenario != whisperScenarioRealtime || len(request.Paths) != 2 || request.Paths[0] != "audios/one.mp3" || request.Paths[1] != "audios/two.m4a" {
			t.Fatalf("unexpected request: %#v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":0,"tasks":[]}`))
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := runIntegrationAPIWhisperCLI([]string{
		"create", "--addr", server.URL, "--agentId", "demo-agent", "--path", "audios/one.mp3", "--path", "audios/two.m4a", "--scenario", whisperScenarioRealtime,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status": 0`) {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestIntegrationAPIWhisperListAndTaskActionCLI(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch requests {
		case 1:
			if r.Method != http.MethodGet || r.URL.Path != "/api/whisper/tasks" {
				t.Fatalf("list request = %s %s", r.Method, r.URL.Path)
			}
			if got := r.URL.Query(); got.Get("agentId") != "demo-agent" || got.Get("status") != "completed" || got.Get("page") != "2" {
				t.Fatalf("list query = %v", got)
			}
			_, _ = w.Write([]byte(`{"status":0,"tasks":[],"total":0,"page":2,"pageSize":5}`))
		case 2:
			if r.Method != http.MethodPost || r.URL.Path != "/api/whisper/tasks/cancel" {
				t.Fatalf("cancel request = %s %s", r.Method, r.URL.Path)
			}
			var request whisperTaskCancelRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode cancel request: %v", err)
			}
			if request.AgentID != "demo-agent" || request.ID != 12 {
				t.Fatalf("cancel request = %#v", request)
			}
			_, _ = w.Write([]byte(`{"status":0}`))
		default:
			t.Fatalf("unexpected request %d", requests)
		}
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	if code := runIntegrationAPIWhisperCLI([]string{"list", "--addr", server.URL, "--agent", "demo-agent", "--status", "completed", "--page", "2"}, &stdout, &stderr); code != 0 {
		t.Fatalf("list exit code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runIntegrationAPIWhisperCLI([]string{"cancel", "--addr", server.URL, "--agentId", "demo-agent", "--id", "12"}, &stdout, &stderr); code != 0 {
		t.Fatalf("cancel exit code = %d, stderr = %s", code, stderr.String())
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func TestIntegrationAPIWhisperHelpListsOperations(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runIntegrationAPIWhisperCLI([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("help exit code = %d", code)
	}
	for _, command := range []string{"check", "list", "create", "cancel", "restart", "delete", "log"} {
		if !strings.Contains(stdout.String(), command) {
			t.Fatalf("help does not contain %q: %s", command, stdout.String())
		}
	}
}
