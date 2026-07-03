package messageinsert

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	return db
}

func TestMessageInsertLifecycle(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	first, err := UpsertPending(db, "agent-a", "chat-1", "101", "hello", time.Unix(1710000000, 0))
	if err != nil {
		t.Fatalf("UpsertPending first: %v", err)
	}
	if first.Status != StatusPending {
		t.Fatalf("first status = %d, want %d", first.Status, StatusPending)
	}

	second, err := UpsertPending(db, "agent-a", "chat-1", "102", "world", time.Unix(1710000010, 0))
	if err != nil {
		t.Fatalf("UpsertPending second: %v", err)
	}
	if second.Mid != "102" {
		t.Fatalf("second mid = %q", second.Mid)
	}

	items, err := ListPending(db, "chat-1", 5)
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("pending items = %d, want 2", len(items))
	}
	if items[0].Mid != "101" || items[1].Mid != "102" {
		t.Fatalf("pending order = %#v", items)
	}

	statuses, err := StatusByMIDs(db, "chat-1", []string{"101", "102"})
	if err != nil {
		t.Fatalf("StatusByMIDs before: %v", err)
	}
	if statuses["101"] != StatusPending || statuses["102"] != StatusPending {
		t.Fatalf("pending statuses = %#v", statuses)
	}

	if _, err := MarkUploaded(db, "chat-1", []string{"101"}, time.Unix(1710000020, 0)); err != nil {
		t.Fatalf("MarkUploaded: %v", err)
	}
	affected, err := Cancel(db, "chat-1", "102", time.Unix(1710000030, 0))
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if !affected {
		t.Fatal("Cancel affected = false, want true")
	}

	statuses, err = StatusByMIDs(db, "chat-1", []string{"101", "102"})
	if err != nil {
		t.Fatalf("StatusByMIDs after: %v", err)
	}
	if statuses["101"] != StatusUploaded {
		t.Fatalf("status 101 = %d, want %d", statuses["101"], StatusUploaded)
	}
	if statuses["102"] != StatusCancelled {
		t.Fatalf("status 102 = %d, want %d", statuses["102"], StatusCancelled)
	}

	items, err = ListPending(db, "chat-1", 5)
	if err != nil {
		t.Fatalf("ListPending after transitions: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("pending items after transitions = %d, want 0", len(items))
	}
}
