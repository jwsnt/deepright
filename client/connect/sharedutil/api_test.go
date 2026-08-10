package sharedutil

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestWriteCmdResponse(t *testing.T) {
	w := httptest.NewRecorder()
	resp := CmdResponse{Status: 0, AgentID: "test", Output: "ok"}
	WriteCmdResponse(w, http.StatusOK, resp)

	result := w.Result()
	if result.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", result.StatusCode, http.StatusOK)
	}
}

func TestWriteKillResponse(t *testing.T) {
	w := httptest.NewRecorder()
	resp := KillResponse{Status: 1, Content: "killed"}
	WriteKillResponse(w, http.StatusOK, resp)

	result := w.Result()
	if result.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", result.StatusCode, http.StatusOK)
	}
}

func TestNormalizeCmdRequest(t *testing.T) {
	tests := []struct {
		name string
		req  *CmdRequest
		want *CmdRequest
	}{
		{"nil", nil, nil},
		{"empty", &CmdRequest{}, &CmdRequest{}},
		{"trim spaces", &CmdRequest{AgentID: "  agent1 ", ChatID: " chat1 ", Cmd: " ls ", Tid: " tid "},
			&CmdRequest{AgentID: "agent1", ChatID: "chat1", Cmd: "ls", Tid: "tid"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NormalizeCmdRequest(tt.req)
			if tt.req == nil {
				return
			}
			if !reflect.DeepEqual(tt.req, tt.want) {
				t.Errorf("NormalizeCmdRequest() = %+v, want %+v", tt.req, tt.want)
			}
		})
	}
}

func TestNormalizeKillRequest(t *testing.T) {
	tests := []struct {
		name string
		req  *KillRequest
		want *KillRequest
	}{
		{"nil", nil, nil},
		{"empty", &KillRequest{}, &KillRequest{}},
		{"trim spaces", &KillRequest{AgentID: "  agent1 ", Cmd: " kill "},
			&KillRequest{AgentID: "agent1", Cmd: "kill"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NormalizeKillRequest(tt.req)
			if tt.req == nil {
				return
			}
			if !reflect.DeepEqual(tt.req, tt.want) {
				t.Errorf("NormalizeKillRequest() = %+v, want %+v", tt.req, tt.want)
			}
		})
	}
}

func TestFindAgent(t *testing.T) {
	agents := []Agent{
		{AgentID: "a1", Description: "first"},
		{AgentID: "a2", Description: "second"},
	}

	tests := []struct {
		name    string
		agentID string
		want    *Agent
	}{
		{"found", "a1", &agents[0]},
		{"found second", "a2", &agents[1]},
		{"not found", "a3", nil},
		{"empty", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindAgent(agents, tt.agentID)
			if tt.want == nil {
				if got != nil {
					t.Errorf("FindAgent() = %+v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("FindAgent() = nil, want %+v", tt.want)
			}
			if got.AgentID != tt.want.AgentID {
				t.Errorf("FindAgent() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestFindAgentEmptySlice(t *testing.T) {
	if got := FindAgent(nil, "a1"); got != nil {
		t.Errorf("FindAgent(nil) = %+v, want nil", got)
	}
}
