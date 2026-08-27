package logretention

import (
	"context"
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
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func TestCleanupExpiredLogsDeletesOnlyExpiredRows(t *testing.T) {
	db := openTestDB(t)
	if _, err := db.Exec(`
		CREATE TABLE agent_message_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id TEXT NOT NULL DEFAULT '',
			chat_id TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			log_type INTEGER NOT NULL,
			created_at TEXT NOT NULL
		);
		CREATE TABLE chat_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id TEXT NOT NULL,
			chat_id TEXT NOT NULL,
			chat_type TEXT NOT NULL DEFAULT 'page_session',
			role TEXT NOT NULL,
			response_type TEXT NOT NULL DEFAULT 'normal',
			content TEXT NOT NULL,
			created_at TEXT NOT NULL
		);
		CREATE TABLE cmd_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id TEXT NOT NULL,
			chat_id TEXT NOT NULL,
			tid TEXT NOT NULL,
			cmd TEXT NOT NULL,
			result TEXT NOT NULL DEFAULT '',
			status INTEGER NOT NULL DEFAULT -1,
			received_at TEXT NOT NULL,
			completed_at TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE chat_history_message (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id TEXT NOT NULL DEFAULT '',
			chat_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content_json TEXT NOT NULL,
			created INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	expired := now.AddDate(0, 0, -31).Format(timeLayout)
	kept := now.AddDate(0, 0, -29).Format(timeLayout)

	if _, err := db.Exec(
		`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES
			('a', 'chat-expired', 'old', 1, ?),
			('a', 'chat-kept', 'new', 1, ?);
		  INSERT INTO chat_log (agent_id, chat_id, chat_type, role, response_type, content, created_at) VALUES
			('a', 'chat-expired', 'page_session', 'A', 'normal', 'old', ?),
			('a', 'chat-kept', 'page_session', 'A', 'normal', 'new', ?);`,
		expired, kept, expired, kept,
	); err != nil {
		t.Fatalf("seed logs: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO cmd_log (agent_id, chat_id, tid, cmd, result, status, received_at, completed_at) VALUES
			('a', 'chat-expired', 'expired', 'old', 'old output', 0, ?, ?),
			('a', 'chat-kept', 'kept', 'new', 'new output', 0, ?, ?);`,
		expired, expired, kept, kept,
	); err != nil {
		t.Fatalf("seed cmd logs: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO chat_history_message (agent_id, chat_id, role, content_json, created, created_at) VALUES
			('a', 'chat-expired', 'user', '"old"', 1, ?),
			('a', 'chat-kept', 'assistant', '"new"', 2, ?);`,
		expired, kept,
	); err != nil {
		t.Fatalf("seed chat history: %v", err)
	}

	result, err := CleanupExpiredLogs(context.Background(), db, now, 30*24)
	if err != nil {
		t.Fatalf("CleanupExpiredLogs: %v", err)
	}
	if result.DeletedAgentMessageLog != 1 {
		t.Fatalf("DeletedAgentMessageLog = %d, want 1", result.DeletedAgentMessageLog)
	}
	if result.DeletedChatLog != 1 {
		t.Fatalf("DeletedChatLog = %d, want 1", result.DeletedChatLog)
	}
	if result.DeletedCmdLog != 1 {
		t.Fatalf("DeletedCmdLog = %d, want 1", result.DeletedCmdLog)
	}
	if result.DeletedChatHistoryMessage != 1 {
		t.Fatalf("DeletedChatHistoryMessage = %d, want 1", result.DeletedChatHistoryMessage)
	}

	var agentCount, chatCount, cmdCount, historyCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM agent_message_log`).Scan(&agentCount); err != nil {
		t.Fatalf("count agent_message_log: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM chat_log`).Scan(&chatCount); err != nil {
		t.Fatalf("count chat_log: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM cmd_log`).Scan(&cmdCount); err != nil {
		t.Fatalf("count cmd_log: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM chat_history_message`).Scan(&historyCount); err != nil {
		t.Fatalf("count chat_history_message: %v", err)
	}
	if agentCount != 1 {
		t.Fatalf("agent_message_log rows = %d, want 1", agentCount)
	}
	if chatCount != 1 {
		t.Fatalf("chat_log rows = %d, want 1", chatCount)
	}
	if cmdCount != 1 {
		t.Fatalf("cmd_log rows = %d, want 1", cmdCount)
	}
	if historyCount != 1 {
		t.Fatalf("chat_history_message rows = %d, want 1", historyCount)
	}
}

func TestManagerStartAsyncMarksCompletion(t *testing.T) {
	db := openTestDB(t)
	if _, err := db.Exec(`CREATE TABLE agent_message_log (id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL DEFAULT '', chat_id TEXT NOT NULL DEFAULT '', content TEXT NOT NULL, log_type INTEGER NOT NULL, created_at TEXT NOT NULL)`); err != nil {
		t.Fatalf("create agent_message_log: %v", err)
	}

	manager := NewManager()
	if !manager.StartAsync(context.Background(), db, 168, nil) {
		t.Fatalf("StartAsync returned false")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		status := manager.Snapshot()
		if status.Checked && !status.Running {
			if status.RetentionHours != 168 {
				t.Fatalf("RetentionHours = %d, want 168", status.RetentionHours)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("manager did not finish in time: %#v", manager.Snapshot())
}

func TestManagerStartAsyncWithDBOpenerMarksCompletion(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "data")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE chat_log (id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL, chat_type TEXT NOT NULL DEFAULT 'page_session', role TEXT NOT NULL, response_type TEXT NOT NULL DEFAULT 'normal', content TEXT NOT NULL, created_at TEXT NOT NULL)`); err != nil {
		_ = db.Close()
		t.Fatalf("create chat_log: %v", err)
	}
	_ = db.Close()

	manager := NewManager()
	if !manager.StartAsyncWithDBOpener(context.Background(), 168, func() (*sql.DB, error) {
		opened, err := sql.Open("sqlite", dbPath)
		if err != nil {
			return nil, err
		}
		opened.SetMaxOpenConns(1)
		opened.SetMaxIdleConns(1)
		return opened, nil
	}, nil) {
		t.Fatalf("StartAsyncWithDBOpener returned false")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		status := manager.Snapshot()
		if status.Checked && !status.Running {
			if status.RetentionHours != 168 {
				t.Fatalf("RetentionHours = %d, want 168", status.RetentionHours)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("manager did not finish in time: %#v", manager.Snapshot())
}

func TestCleanupExpiredLogsDeletesInBatches(t *testing.T) {
	db := openTestDB(t)
	if _, err := db.Exec(`CREATE TABLE agent_message_log (id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL DEFAULT '', chat_id TEXT NOT NULL DEFAULT '', content TEXT NOT NULL, log_type INTEGER NOT NULL, created_at TEXT NOT NULL)`); err != nil {
		t.Fatalf("create agent_message_log: %v", err)
	}

	oldBatchSize := cleanupDeleteBatchSize
	cleanupDeleteBatchSize = 2
	defer func() {
		cleanupDeleteBatchSize = oldBatchSize
	}()

	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	expired := now.AddDate(0, 0, -31).Format(timeLayout)
	fresh := now.AddDate(0, 0, -1).Format(timeLayout)

	for i := 0; i < 5; i++ {
		if _, err := db.Exec(`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES ('a', 'expired', 'old', 1, ?)`, expired); err != nil {
			t.Fatalf("insert expired row %d: %v", i, err)
		}
	}
	if _, err := db.Exec(`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES ('a', 'fresh', 'new', 1, ?)`, fresh); err != nil {
		t.Fatalf("insert fresh row: %v", err)
	}

	result, err := CleanupExpiredLogs(context.Background(), db, now, 30*24)
	if err != nil {
		t.Fatalf("CleanupExpiredLogs: %v", err)
	}
	if result.DeletedAgentMessageLog != 5 {
		t.Fatalf("DeletedAgentMessageLog = %d, want 5", result.DeletedAgentMessageLog)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM agent_message_log`).Scan(&count); err != nil {
		t.Fatalf("count agent_message_log: %v", err)
	}
	if count != 1 {
		t.Fatalf("agent_message_log rows = %d, want 1", count)
	}
}
