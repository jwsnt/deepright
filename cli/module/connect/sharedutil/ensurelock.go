package sharedutil

import (
	"database/sql"
	"fmt"
	"strings"
)

// EnsureKnowledgeRequestLockTable creates the knowledge_update_lock table
// and ensures the singleton row exists. This is shared between proxy and integration.
func EnsureKnowledgeRequestLockTable(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("nil sqlite db")
	}
	if err := ensureKnowledgeRequestLockTableSchema(db); err != nil {
		return err
	}
	return EnsureKnowledgeRequestLockRow(db, "")
}

// EnsureKnowledgeRequestLockTableTx creates the lock table within a transaction.
func EnsureKnowledgeRequestLockTableTx(tx *sql.Tx) error {
	if tx == nil {
		return fmt.Errorf("nil sqlite tx")
	}
	if err := ensureKnowledgeRequestLockTableSchemaTx(tx); err != nil {
		return err
	}
	return EnsureKnowledgeRequestLockRowTx(tx, "")
}

func EnsureKnowledgeRequestLockRow(db *sql.DB, agentID string) error {
	if db == nil {
		return fmt.Errorf("nil sqlite db")
	}
	if err := ensureKnowledgeRequestLockTableSchema(db); err != nil {
		return err
	}
	if _, err := db.Exec(`
		INSERT OR IGNORE INTO knowledge_update_lock(agent_id, last_requested_at) VALUES (?, 0)
	`, normalizeKnowledgeLockAgentID(agentID)); err != nil {
		return fmt.Errorf("init knowledge_update_lock row: %w", err)
	}
	return nil
}

func EnsureKnowledgeRequestLockRowTx(tx *sql.Tx, agentID string) error {
	if tx == nil {
		return fmt.Errorf("nil sqlite tx")
	}
	if err := ensureKnowledgeRequestLockTableSchemaTx(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT OR IGNORE INTO knowledge_update_lock(agent_id, last_requested_at) VALUES (?, 0)
	`, normalizeKnowledgeLockAgentID(agentID)); err != nil {
		return fmt.Errorf("init knowledge_update_lock row: %w", err)
	}
	return nil
}

func normalizeKnowledgeLockAgentID(agentID string) string {
	return strings.TrimSpace(agentID)
}

func ensureKnowledgeRequestLockTableSchema(db schemaExecer) error {
	hasTable, err := schemaTableExists(db, "knowledge_update_lock")
	if err != nil {
		return err
	}
	if !hasTable {
		if err := createKnowledgeRequestLockTable(db); err != nil {
			return err
		}
		return nil
	}
	hasAgentID, err := schemaTableHasColumn(db, "knowledge_update_lock", "agent_id")
	if err != nil {
		return err
	}
	if hasAgentID {
		return nil
	}
	return migrateKnowledgeRequestLockTable(db)
}

func ensureKnowledgeRequestLockTableSchemaTx(tx *sql.Tx) error {
	return ensureKnowledgeRequestLockTableSchema(tx)
}

type schemaExecer interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

func createKnowledgeRequestLockTable(db schemaExecer) error {
	if _, err := db.Exec(`
		CREATE TABLE knowledge_update_lock (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id TEXT NOT NULL UNIQUE,
			last_requested_at INTEGER NOT NULL DEFAULT 0
		)
	`); err != nil {
		return fmt.Errorf("create knowledge_update_lock: %w", err)
	}
	return nil
}

func migrateKnowledgeRequestLockTable(db schemaExecer) error {
	if err := createKnowledgeRequestLockTableWithName(db, "knowledge_update_lock_migrated"); err != nil {
		return err
	}
	if _, err := db.Exec(`
		INSERT INTO knowledge_update_lock_migrated(agent_id, last_requested_at)
		SELECT '', COALESCE(last_requested_at, 0) FROM knowledge_update_lock
	`); err != nil {
		return fmt.Errorf("migrate knowledge_update_lock data: %w", err)
	}
	if _, err := db.Exec(`DROP TABLE knowledge_update_lock`); err != nil {
		return fmt.Errorf("drop legacy knowledge_update_lock: %w", err)
	}
	if _, err := db.Exec(`ALTER TABLE knowledge_update_lock_migrated RENAME TO knowledge_update_lock`); err != nil {
		return fmt.Errorf("rename migrated knowledge_update_lock: %w", err)
	}
	return nil
}

func createKnowledgeRequestLockTableWithName(db schemaExecer, tableName string) error {
	if _, err := db.Exec(fmt.Sprintf(`
		CREATE TABLE %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id TEXT NOT NULL UNIQUE,
			last_requested_at INTEGER NOT NULL DEFAULT 0
		)
	`, tableName)); err != nil {
		return fmt.Errorf("create %s: %w", tableName, err)
	}
	return nil
}

func schemaTableExists(db schemaExecer, table string) (bool, error) {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count); err != nil {
		return false, fmt.Errorf("query sqlite_master for %s: %w", table, err)
	}
	return count > 0, nil
}

func schemaTableHasColumn(db schemaExecer, table string, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return false, fmt.Errorf("query table_info for %s: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			colType    string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultVal, &pk); err != nil {
			return false, fmt.Errorf("scan table_info for %s: %w", table, err)
		}
		if strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(column)) {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate table_info for %s: %w", table, err)
	}
	return false, nil
}
