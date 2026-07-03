package sharedutil

import (
	"encoding/json"
	"net/http"
	"strings"
)

// ═══════════════════════════════════════════════════════════════════════════
// Request / Response types shared by proxy and integration
// ═══════════════════════════════════════════════════════════════════════════

// CmdRequest is the JSON body for a command execution request.
type CmdRequest struct {
	AgentID string `json:"agentId"`
	ChatID  string `json:"chatId"`
	Cmd     string `json:"cmd"`
	Tid     string `json:"tid,omitempty"`
	Timeout int64  `json:"timeout,omitempty"`
}

// KillRequest is the JSON body for a kill request.
type KillRequest struct {
	AgentID string `json:"agentId"`
	ChatID  string `json:"chatId"`
	Cmd     string `json:"cmd"`
	Tid     string `json:"tid,omitempty"`
}

// CmdResponse is the JSON response for command execution.
type CmdResponse struct {
	Status      int    `json:"status"`
	AgentID     string `json:"agentId,omitempty"`
	ChatID      string `json:"chatId,omitempty"`
	Tid         string `json:"tid,omitempty"`
	Cmd         string `json:"cmd,omitempty"`
	Output      string `json:"output,omitempty"`
	Content     string `json:"content,omitempty"`
	ReceivedAt  string `json:"receivedAt,omitempty"`
	CompletedAt string `json:"completedAt,omitempty"`
}

// KillResponse is the JSON response for kill operations.
type KillResponse struct {
	Status      int    `json:"status"`
	AgentID     string `json:"agentId,omitempty"`
	ChatID      string `json:"chatId,omitempty"`
	Tid         string `json:"tid,omitempty"`
	Cmd         string `json:"cmd,omitempty"`
	Content     string `json:"content,omitempty"`
	ReceivedAt  string `json:"receivedAt,omitempty"`
	CompletedAt string `json:"completedAt,omitempty"`
}

// WriteCmdResponse writes a JSON CmdResponse with the given status code.
func WriteCmdResponse(w http.ResponseWriter, code int, resp CmdResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(resp)
}

// WriteKillResponse writes a JSON KillResponse with the given status code.
func WriteKillResponse(w http.ResponseWriter, code int, resp KillResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(resp)
}

// NormalizeCmdRequest trims whitespace from all CmdRequest fields.
func NormalizeCmdRequest(req *CmdRequest) {
	if req != nil {
		req.AgentID = strings.TrimSpace(req.AgentID)
		req.ChatID = strings.TrimSpace(req.ChatID)
		req.Cmd = strings.TrimSpace(req.Cmd)
		req.Tid = strings.TrimSpace(req.Tid)
	}
}

// NormalizeKillRequest trims whitespace from all KillRequest fields.
func NormalizeKillRequest(req *KillRequest) {
	if req != nil {
		req.AgentID = strings.TrimSpace(req.AgentID)
		req.ChatID = strings.TrimSpace(req.ChatID)
		req.Cmd = strings.TrimSpace(req.Cmd)
		req.Tid = strings.TrimSpace(req.Tid)
	}
}

// FindAgent searches an Agent slice for one matching agentID.
func FindAgent(agents []Agent, agentID string) *Agent {
	for i := range agents {
		if agents[i].AgentID == agentID {
			return &agents[i]
		}
	}
	return nil
}
type Agent struct {
	AgentID     string `json:"agentId"`
	Description string `json:"description"`
	Provider    string `json:"provider"`
	Swarm       bool   `json:"swarm"`
	Thinking    bool   `json:"thinking"`
	Workspace   string `json:"workspace"`
}
