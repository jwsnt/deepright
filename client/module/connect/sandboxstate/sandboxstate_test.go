package sandboxstate

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func TestSetAndGet(t *testing.T) {
	db := openTestDB(t)
	now := time.Date(2026, time.June, 8, 12, 34, 56, 0, time.UTC)

	state, err := setAt(db, "agent-a", "chat-1", ModeFilePickNet, "/Users/demo/Pictures", now)
	if err != nil {
		t.Fatalf("setAt: %v", err)
	}
	if state.Sandbox != ModeFilePickNet || state.AgentID != "agent-a" || state.ChatID != "chat-1" || state.AllowedDir != "/Users/demo/Pictures" {
		t.Fatalf("unexpected state: %+v", state)
	}

	got, found, err := Get(db, "agent-b", "chat-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("Get should return found=true")
	}
	if got.ChatID != "chat-1" || got.Sandbox != ModeFilePickNet || got.AllowedDir != "/Users/demo/Pictures" || got.UpdatedAt != state.UpdatedAt {
		t.Fatalf("Get = %+v, want chat-scoped state", got)
	}
	if got.AgentID != "agent-b" {
		t.Fatalf("Get agent echo = %q, want agent-b", got.AgentID)
	}
}

func TestGetMissingState(t *testing.T) {
	db := openTestDB(t)

	got, found, err := Get(db, "", "chat-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if found {
		t.Fatal("Get should return found=false")
	}
	if got.AgentID != "" || got.ChatID != "chat-1" || got.Sandbox != "" {
		t.Fatalf("unexpected missing state: %+v", got)
	}
}

func TestSetRequiresChatID(t *testing.T) {
	db := openTestDB(t)

	if _, err := Set(db, "agent-a", "   ", ModeNet); err == nil {
		t.Fatal("Set should require chatId")
	}
}

func TestSetEmptyModeDeletesState(t *testing.T) {
	db := openTestDB(t)

	if _, err := Set(db, "agent-a", "chat-1", ModeNet); err != nil {
		t.Fatalf("Set enable: %v", err)
	}
	state, err := Set(db, "agent-a", "chat-1", "")
	if err != nil {
		t.Fatalf("Set disable: %v", err)
	}
	if state.Sandbox != "" || state.UpdatedAt != "" {
		t.Fatalf("unexpected deleted state: %+v", state)
	}
	got, found, err := Get(db, "", "chat-1")
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if found || got.Sandbox != "" {
		t.Fatalf("expected empty state after delete, got %+v found=%t", got, found)
	}
}

func TestSetClearsAllowedDirForNetMode(t *testing.T) {
	db := openTestDB(t)

	state, err := SetWithDirectory(db, "agent-a", "chat-1", ModeNet, "/Users/demo/Desktop")
	if err != nil {
		t.Fatalf("SetWithDirectory: %v", err)
	}
	if state.AllowedDir != "" {
		t.Fatalf("state.AllowedDir = %q, want empty", state.AllowedDir)
	}

	got, found, err := Get(db, "", "chat-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("Get should return found=true")
	}
	if got.AllowedDir != "" {
		t.Fatalf("got.AllowedDir = %q, want empty", got.AllowedDir)
	}
}

func TestGetRequiresChatID(t *testing.T) {
	db := openTestDB(t)

	if _, _, err := Get(db, "agent-a", ""); err == nil {
		t.Fatal("Get should require chatId")
	}
}
