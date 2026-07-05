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
	if second.Tid != "102" {
		t.Fatalf("second tid = %q", second.Tid)
	}

	items, err := ListPending(db, "chat-1", 5)
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("pending items = %d, want 2", len(items))
	}
	if items[0].Tid != "101" || items[1].Tid != "102" {
		t.Fatalf("pending order = %#v", items)
	}

	statuses, err := StatusByTIDs(db, "chat-1", []string{"101", "102"})
	if err != nil {
		t.Fatalf("StatusByTIDs before: %v", err)
	}
	if statuses["101"] != StatusPending || statuses["102"] != StatusPending {
		t.Fatalf("pending statuses = %#v", statuses)
	}

	if _, err := MarkPublished(db, "chat-1", []string{"101"}, time.Unix(1710000015, 0)); err != nil {
		t.Fatalf("MarkPublished: %v", err)
	}
	activeItems, err := ListActive(db, "chat-1", 5)
	if err != nil {
		t.Fatalf("ListActive after publish: %v", err)
	}
	if len(activeItems) != 2 {
		t.Fatalf("active items after publish = %d, want 2", len(activeItems))
	}
	if activeItems[0].Tid != "101" || activeItems[1].Tid != "102" {
		t.Fatalf("active order after publish = %#v", activeItems)
	}
	if activeItems[0].ReportedAt == "" {
		t.Fatalf("reported_at for published item is empty: %#v", activeItems[0])
	}
	items, err = ListPending(db, "chat-1", 5)
	if err != nil {
		t.Fatalf("ListPending after publish: %v", err)
	}
	if len(items) != 1 || items[0].Tid != "102" {
		t.Fatalf("pending after publish = %#v", items)
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

	statuses, err = StatusByTIDs(db, "chat-1", []string{"101", "102"})
	if err != nil {
		t.Fatalf("StatusByTIDs after: %v", err)
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

	activeItems, err = ListActive(db, "chat-1", 5)
	if err != nil {
		t.Fatalf("ListActive after transitions: %v", err)
	}
	if len(activeItems) != 0 {
		t.Fatalf("active items after transitions = %d, want 0", len(activeItems))
	}
}
