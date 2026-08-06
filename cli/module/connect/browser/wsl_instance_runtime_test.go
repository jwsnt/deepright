package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserWSLInstanceEnsureChatProfileDirCreatesEmptyDirectory(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	root := t.TempDir()
	profileWin, profileUnix, err := browserWSLInstanceEnsureChatProfileDirAt("chat-001", root, `C:\ProgramData\deepright`)
	if err != nil {
		t.Fatal(err)
	}
	wantUnix := filepath.Join(root, "profiles", "chats", "chat-001")
	if profileUnix != wantUnix {
		t.Fatalf("profile directory = %q, want %q", profileUnix, wantUnix)
	}
	entries, err := os.ReadDir(profileUnix)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("new WSL profile must be empty, got %d entries", len(entries))
	}
	if profileWin != `C:\ProgramData\deepright\profiles\chats\chat-001` {
		t.Fatalf("profile directory = %q", profileWin)
	}
}

func TestBrowserWSLInstanceRecordsAreSharedByChat(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "browser_data"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE browser_instance_wsl (
		agent_id TEXT NOT NULL,
		chat_id TEXT NOT NULL,
		pid INTEGER NOT NULL DEFAULT 0,
		port INTEGER NOT NULL DEFAULT 0,
		ws TEXT NOT NULL DEFAULT '',
		http TEXT NOT NULL DEFAULT '',
		user_data_dir TEXT NOT NULL DEFAULT '',
		updated_at TEXT NOT NULL,
		PRIMARY KEY (agent_id, chat_id)
	)`); err != nil {
		t.Fatal(err)
	}

	first := browserWSLInstanceRecord{AgentID: "agent-a", ChatID: "chat-001", PID: 101, Port: 9222, UserDataDir: `C:\ProgramData\deepright\profiles\chats\chat-001`}
	if err := browserWSLInstanceUpsertRecord(db, first); err != nil {
		t.Fatal(err)
	}
	got, found, err := browserWSLInstanceGetRecord(db, "agent-b", "chat-001")
	if err != nil || !found || got.PID != first.PID {
		t.Fatalf("shared chat record = %+v, found=%v, err=%v", got, found, err)
	}

	second := first
	second.AgentID = "agent-b"
	second.PID = 202
	if err := browserWSLInstanceUpsertRecord(db, second); err != nil {
		t.Fatal(err)
	}
	got, found, err = browserWSLInstanceGetRecord(db, "agent-a", "chat-001")
	if err != nil || !found || got.PID != second.PID || got.AgentID != "agent-b" {
		t.Fatalf("replaced shared chat record = %+v, found=%v, err=%v", got, found, err)
	}
	if err := browserWSLInstanceDeleteRecord(db, "agent-a", "chat-001"); err != nil {
		t.Fatal(err)
	}
	if _, found, err := browserWSLInstanceGetRecord(db, "agent-b", "chat-001"); err != nil || found {
		t.Fatalf("deleted shared chat record found=%v, err=%v", found, err)
	}
}
