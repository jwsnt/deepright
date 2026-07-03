package sharedutil

import (
	"testing"
)

func TestGenerateCmdTID(t *testing.T) {
	id1 := GenerateCmdTID()
	id2 := GenerateCmdTID()

	if len(id1) != 8 {
		t.Errorf("GenerateCmdTID() length = %d, want 8", len(id1))
	}
	if id1 == id2 {
		t.Errorf("GenerateCmdTID() returned same value: %s", id1)
	}
}

func TestActiveCmdKey(t *testing.T) {
	key := ActiveCmdKey("agent1", "chat1", "tid1", "ls -la")
	if key != "agent1|chat1|tid1|ls -la" {
		t.Errorf("ActiveCmdKey() = %q, want %q", key, "agent1|chat1|tid1|ls -la")
	}
}

func TestRegisterAndUnregisterActiveCmd(t *testing.T) {
	cmd := &ActiveCmd{
		AgentID: "test-agent",
		ChatID:  "test-chat",
		Tid:     "test-tid",
		RawCmd:  "echo hello",
	}

	key := RegisterActiveCmd(cmd)
	if key == "" {
		t.Fatal("RegisterActiveCmd() returned empty key")
	}

	// Find it back by exact match
	found, foundKey := FindActiveCmd("test-agent", "test-chat", "test-tid", "echo hello")
	if found == nil {
		t.Fatal("FindActiveCmd() returned nil for registered command")
	}
	if found.AgentID != "test-agent" {
		t.Errorf("FindActiveCmd() AgentID = %q", found.AgentID)
	}
	if foundKey != key {
		t.Errorf("FindActiveCmd() key = %q, want %q", foundKey, key)
	}

	// Find by fallback (without tid)
	found2, _ := FindActiveCmd("test-agent", "test-chat", "wrong-tid", "echo hello")
	if found2 == nil {
		t.Fatal("FindActiveCmd() fallback returned nil")
	}

	// Unregister
	UnregisterActiveCmd(key)
	found3, _ := FindActiveCmd("test-agent", "test-chat", "test-tid", "echo hello")
	if found3 != nil {
		t.Fatal("FindActiveCmd() should return nil after unregister")
	}
}

func TestDefaultExecShell(t *testing.T) {
	shell := DefaultExecShell()
	if shell == "" {
		t.Error("DefaultExecShell() returned empty string")
	}
}

func TestIntPtrValue(t *testing.T) {
	tests := []struct {
		name  string
		value *int
		want  int
	}{
		{"nil", nil, 0},
		{"zero", intPtr(0), 0},
		{"positive", intPtr(42), 42},
		{"negative", intPtr(-1), -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IntPtrValue(tt.value); got != tt.want {
				t.Errorf("IntPtrValue() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGzipBase64String(t *testing.T) {
	input := "hello world"
	encoded := GzipBase64String(input)
	if encoded == "" {
		t.Fatal("GzipBase64String() returned empty")
	}
	if encoded == input {
		t.Error("GzipBase64String() returned the same string (no compression)")
	}

	// Verify it's valid base64
	if len(encoded) < 10 {
		t.Errorf("GzipBase64String() result too short: %q", encoded)
	}
}

func intPtr(v int) *int { return &v }
