package sharedutil

import (
	"database/sql"
	"testing"

	_ "github.com/glebarez/go-sqlite"
)

func TestEnsureKnowledgeRequestLockTable(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := EnsureKnowledgeRequestLockTable(db); err != nil {
		t.Fatalf("EnsureKnowledgeRequestLockTable() error: %v", err)
	}

	// Call again to ensure idempotent
	if err := EnsureKnowledgeRequestLockTable(db); err != nil {
		t.Fatalf("EnsureKnowledgeRequestLockTable() second call error: %v", err)
	}

	// Verify the row exists
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM knowledge_update_lock`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}
}

func TestEnsureKnowledgeRequestLockTableNilDB(t *testing.T) {
	if err := EnsureKnowledgeRequestLockTable(nil); err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestEnsureKnowledgeRequestLockTableTx(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	if err := EnsureKnowledgeRequestLockTableTx(tx); err != nil {
		t.Fatalf("EnsureKnowledgeRequestLockTableTx() error: %v", err)
	}
}

func TestEnsureKnowledgeRequestLockTableTxNil(t *testing.T) {
	if err := EnsureKnowledgeRequestLockTableTx(nil); err == nil {
		t.Fatal("expected error for nil tx")
	}
}
