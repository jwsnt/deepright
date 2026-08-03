package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIntegrationAPIRembgCreateAndActionCLI(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch requests {
		case 1:
			if r.Method != http.MethodPost || r.URL.Path != "/api/rembg/tasks" {
				t.Fatalf("create request = %s %s", r.Method, r.URL.Path)
			}
			var request rembgTaskCreateRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode create request: %v", err)
			}
			if request.AgentID != "demo-agent" || request.Model != "isnet-general-use" || !request.AlphaMatting || len(request.Paths) != 2 || request.Paths[0] != "images/a.jpg" || request.Paths[1] != "images/b.png" {
				t.Fatalf("unexpected create request: %#v", request)
			}
		case 2:
			if r.Method != http.MethodPost || r.URL.Path != "/api/rembg/tasks/cancel" {
				t.Fatalf("cancel request = %s %s", r.Method, r.URL.Path)
			}
			var request rembgTaskActionRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode cancel request: %v", err)
			}
			if request.AgentID != "demo-agent" || request.ID != 12 {
				t.Fatalf("unexpected cancel request: %#v", request)
			}
		default:
			t.Fatalf("unexpected request %d", requests)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":0}`))
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	if code := runIntegrationAPIRembgCLI([]string{"create", "--addr", server.URL, "--agentId", "demo-agent", "--path", "images/a.jpg", "--path", "images/b.png", "--model", "isnet-general-use", "--alpha-matting"}, &stdout, &stderr); code != 0 {
		t.Fatalf("create exit code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runIntegrationAPIRembgCLI([]string{"cancel", "--addr", server.URL, "--agentId", "demo-agent", "--id", "12"}, &stdout, &stderr); code != 0 {
		t.Fatalf("cancel exit code = %d, stderr = %s", code, stderr.String())
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func TestIntegrationAPIRembgHelpListsOperations(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runIntegrationAPIRembgCLI([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("help exit code = %d", code)
	}
	for _, command := range []string{"check", "list", "create", "cancel", "restart", "delete", "log", "--alpha-matting"} {
		if !strings.Contains(stdout.String(), command) {
			t.Fatalf("help does not contain %q: %s", command, stdout.String())
		}
	}
}

func TestIntegrationAPIVoxCPMBatchCreateCLI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/voxcpm/tasks" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		var request voxcpmTaskCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.AgentID != "demo-agent" || request.OutputName != "narration.wav" || len(request.Tasks) != 2 {
			t.Fatalf("unexpected request: %#v", request)
		}
		first, second := request.Tasks[0], request.Tasks[1]
		if first.TextPath != "scripts/a.txt" || second.TextPath != "scripts/b.txt" || first.ReferenceAudioPath != "audios/voice.wav" || second.ReferenceAudioPath != "audios/voice.wav" || first.Scenario != voxcpmScenarioWarmNarration || second.Scenario != voxcpmScenarioLively || first.Control != "温和、平稳" || second.Control != "轻快、有活力" {
			t.Fatalf("unexpected task values: %#v", request.Tasks)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":0,"tasks":[]}`))
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := runIntegrationAPIVoxCPMCLI([]string{
		"create", "--addr", server.URL, "--agentId", "demo-agent", "--textPath", "scripts/a.txt", "--text-path", "scripts/b.txt", "--referenceAudioPath", "audios/voice.wav", "--outputName", "narration.wav", "--scenario", voxcpmScenarioWarmNarration, "--scenario", voxcpmScenarioLively, "--control", "温和、平稳", "--control", "轻快、有活力",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status": 0`) {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestIntegrationAPIVoxCPMHelpListsOperations(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runIntegrationAPIVoxCPMCLI([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("help exit code = %d", code)
	}
	for _, command := range []string{"check", "list", "create", "cancel", "restart", "delete", "log", "--textPath"} {
		if !strings.Contains(stdout.String(), command) {
			t.Fatalf("help does not contain %q: %s", command, stdout.String())
		}
	}
}

func TestIntegrationAPIRVMCreateAndRestartCLI(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch requests {
		case 1:
			if r.Method != http.MethodPost || r.URL.Path != "/api/rvm/tasks" {
				t.Fatalf("create request = %s %s", r.Method, r.URL.Path)
			}
			var request rvmTaskCreateRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode create request: %v", err)
			}
			if request.AgentID != "demo-agent" || len(request.Tasks) != 2 || request.Tasks[0].Path != "videos/a.mp4" || request.Tasks[0].Scenario != rvmScenarioQuality || request.Tasks[1].Path != "videos/b.mov" || request.Tasks[1].Scenario != rvmScenarioFast {
				t.Fatalf("unexpected RVM create request: %#v", request)
			}
		case 2:
			if r.Method != http.MethodPost || r.URL.Path != "/api/rvm/tasks/restart" {
				t.Fatalf("restart request = %s %s", r.Method, r.URL.Path)
			}
			var request rvmTaskActionRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode restart request: %v", err)
			}
			if request.AgentID != "demo-agent" || request.ID != 12 {
				t.Fatalf("unexpected RVM restart request: %#v", request)
			}
		default:
			t.Fatalf("unexpected request %d", requests)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":0,"tasks":[]}`))
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	if code := runIntegrationAPICLI([]string{"rvm", "create", "--addr", server.URL, "--agentId", "demo-agent", "--path", "videos/a.mp4", "--path", "videos/b.mov", "--scenario", rvmScenarioQuality, "--scenario", rvmScenarioFast}, &stdout, &stderr); code != 0 {
		t.Fatalf("create exit code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runIntegrationAPICLI([]string{"rvm", "restart", "--addr", server.URL, "--agentId", "demo-agent", "--id", "12"}, &stdout, &stderr); code != 0 {
		t.Fatalf("restart exit code = %d, stderr = %s", code, stderr.String())
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	stdout.Reset()
	if code := runIntegrationAPIRVMCLI([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("help exit code = %d", code)
	}
	for _, text := range []string{"check", "list", "create", "cancel", "restart", "delete", "log", "--scenario"} {
		if !strings.Contains(stdout.String(), text) {
			t.Fatalf("help does not contain %q: %s", text, stdout.String())
		}
	}
}

func TestIntegrationAPIWav2LipCreateAndHelpCLI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/wav2lip/tasks" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		var request wav2lipTaskCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.AgentID != "demo-agent" || len(request.Tasks) != 2 || request.Tasks[0].VideoPath != "videos/a.mp4" || request.Tasks[0].AudioPath != "audios/a.wav" || request.Tasks[1].VideoPath != "videos/b.mov" || request.Tasks[1].AudioPath != "audios/b.mp3" {
			t.Fatalf("unexpected Wav2Lip create request: %#v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":0,"tasks":[]}`))
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	if code := runIntegrationAPICLI([]string{"wav2lip", "create", "--addr", server.URL, "--agentId", "demo-agent", "--videoPath", "videos/a.mp4", "--audioPath", "audios/a.wav", "--video-path", "videos/b.mov", "--audio-path", "audios/b.mp3"}, &stdout, &stderr); code != 0 {
		t.Fatalf("create exit code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	if code := runIntegrationAPIWav2LipCLI([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("help exit code = %d", code)
	}
	for _, text := range []string{"check", "list", "create", "cancel", "restart", "delete", "log", "--videoPath", "--audioPath"} {
		if !strings.Contains(stdout.String(), text) {
			t.Fatalf("help does not contain %q: %s", text, stdout.String())
		}
	}
}
