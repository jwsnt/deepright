package logretention

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	DefaultRetentionHours = 30 * 24
	DefaultMessage        = "正在清理过期日志，请稍后"
	timeLayout            = "2006-01-02T15:04:05.000"
)

var cleanupDeleteBatchSize = 500

type Result struct {
	RetentionHours            int
	Cutoff                    string
	DeletedAgentMessageLog    int64
	DeletedChatLog            int64
	DeletedCmdLog             int64
	DeletedChatHistoryMessage int64
}

type Status struct {
	Checked                   bool
	Running                   bool
	Message                   string
	RetentionHours            int
	Cutoff                    string
	StartedAt                 string
	FinishedAt                string
	DeletedAgentMessageLog    int64
	DeletedChatLog            int64
	DeletedCmdLog             int64
	DeletedChatHistoryMessage int64
	Error                     string
}

type Manager struct {
	mu      sync.RWMutex
	started bool
	status  Status
}

func NewManager() *Manager {
	return &Manager{
		status: Status{
			Message:        DefaultMessage,
			RetentionHours: DefaultRetentionHours,
		},
	}
}

func (m *Manager) Snapshot() Status {
	if m == nil {
		return Status{
			Checked:        true,
			Message:        DefaultMessage,
			RetentionHours: DefaultRetentionHours,
			Error:          "log retention manager is nil",
		}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func (m *Manager) MarkChecked(err error) {
	m.MarkCheckedWithRetentionHours(DefaultRetentionHours, err)
}

func (m *Manager) MarkCheckedWithRetentionHours(retentionHours int, err error) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = true
	m.status.Checked = true
	m.status.Running = false
	m.status.Message = DefaultMessage
	m.status.RetentionHours = retentionHours
	m.status.FinishedAt = time.Now().Format(timeLayout)
	if err != nil {
		m.status.Error = strings.TrimSpace(err.Error())
	}
}

func (m *Manager) startAsyncRunner(
	ctx context.Context,
	retentionHours int,
	logf func(string, ...any),
	run func(context.Context, time.Time, int) (Result, error),
) bool {
	if m == nil {
		return false
	}
	if retentionHours <= 0 {
		m.MarkCheckedWithRetentionHours(retentionHours, fmt.Errorf("log retention hours must be positive"))
		return false
	}
	if run == nil {
		m.MarkCheckedWithRetentionHours(retentionHours, fmt.Errorf("log retention runner is nil"))
		return false
	}

	startedAt := time.Now()
	cutoff := startedAt.Add(-time.Duration(retentionHours) * time.Hour).Format(timeLayout)

	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return false
	}
	m.started = true
	m.status = Status{
		Checked:        false,
		Running:        true,
		Message:        DefaultMessage,
		RetentionHours: retentionHours,
		Cutoff:         cutoff,
		StartedAt:      startedAt.Format(timeLayout),
	}
	m.mu.Unlock()

	go func() {
		result, err := run(ctx, startedAt, retentionHours)
		finishedAt := time.Now().Format(timeLayout)

		m.mu.Lock()
		m.status.Checked = true
		m.status.Running = false
		m.status.Message = DefaultMessage
		m.status.RetentionHours = retentionHours
		m.status.Cutoff = result.Cutoff
		if m.status.Cutoff == "" {
			m.status.Cutoff = cutoff
		}
		m.status.FinishedAt = finishedAt
		m.status.DeletedAgentMessageLog = result.DeletedAgentMessageLog
		m.status.DeletedChatLog = result.DeletedChatLog
		m.status.DeletedCmdLog = result.DeletedCmdLog
		m.status.DeletedChatHistoryMessage = result.DeletedChatHistoryMessage
		if err != nil {
			m.status.Error = strings.TrimSpace(err.Error())
		}
		m.mu.Unlock()

		if logf == nil {
			return
		}
		if err != nil {
			logf("[log-retention] cleanup failed: %v", err)
			return
		}
		logf(
			"[log-retention] cleanup finished cutoff=%s retention_hours=%d agent_message_log=%d chat_log=%d cmd_log=%d chat_history_message=%d",
			result.Cutoff,
			result.RetentionHours,
			result.DeletedAgentMessageLog,
			result.DeletedChatLog,
			result.DeletedCmdLog,
			result.DeletedChatHistoryMessage,
		)
	}()
	return true
}

func (m *Manager) StartAsync(ctx context.Context, db *sql.DB, retentionHours int, logf func(string, ...any)) bool {
	if db == nil {
		m.MarkCheckedWithRetentionHours(retentionHours, fmt.Errorf("log retention db is nil"))
		return false
	}
	return m.startAsyncRunner(ctx, retentionHours, logf, func(ctx context.Context, startedAt time.Time, retentionHours int) (Result, error) {
		return CleanupExpiredLogs(ctx, db, startedAt, retentionHours)
	})
}

func (m *Manager) StartAsyncWithDBOpener(ctx context.Context, retentionHours int, openDB func() (*sql.DB, error), logf func(string, ...any)) bool {
	if openDB == nil {
		m.MarkCheckedWithRetentionHours(retentionHours, fmt.Errorf("log retention db opener is nil"))
		return false
	}
	return m.startAsyncRunner(ctx, retentionHours, logf, func(ctx context.Context, startedAt time.Time, retentionHours int) (Result, error) {
		db, err := openDB()
		if err != nil {
			return Result{
				RetentionHours: retentionHours,
				Cutoff:         startedAt.Add(-time.Duration(retentionHours) * time.Hour).Format(timeLayout),
			}, err
		}
		if db == nil {
			return Result{
				RetentionHours: retentionHours,
				Cutoff:         startedAt.Add(-time.Duration(retentionHours) * time.Hour).Format(timeLayout),
			}, fmt.Errorf("log retention db is nil")
		}
		defer db.Close()
		return CleanupExpiredLogs(ctx, db, startedAt, retentionHours)
	})
}

func CleanupExpiredLogs(ctx context.Context, db *sql.DB, now time.Time, retentionHours int) (Result, error) {
	if db == nil {
		return Result{}, fmt.Errorf("log retention db is nil")
	}
	if retentionHours <= 0 {
		return Result{}, fmt.Errorf("log retention hours must be positive")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	result := Result{
		RetentionHours: retentionHours,
		Cutoff:         now.Add(-time.Duration(retentionHours) * time.Hour).Format(timeLayout),
	}

	if exists, err := tableExists(ctx, db, "agent_message_log"); err != nil {
		return result, err
	} else if exists {
		deleted, err := deleteBeforeCutoffBatched(ctx, db, "agent_message_log", "created_at", result.Cutoff)
		if err != nil {
			return result, err
		}
		result.DeletedAgentMessageLog = deleted
	}

	if exists, err := tableExists(ctx, db, "chat_log"); err != nil {
		return result, err
	} else if exists {
		deleted, err := deleteBeforeCutoffBatched(ctx, db, "chat_log", "created_at", result.Cutoff)
		if err != nil {
			return result, err
		}
		result.DeletedChatLog = deleted
	}

	if exists, err := tableExists(ctx, db, "cmd_log"); err != nil {
		return result, err
	} else if exists {
		deleted, err := deleteBeforeCutoffBatched(ctx, db, "cmd_log", "received_at", result.Cutoff)
		if err != nil {
			return result, err
		}
		result.DeletedCmdLog = deleted
	}

	if exists, err := tableExists(ctx, db, "chat_history_message"); err != nil {
		return result, err
	} else if exists {
		deleted, err := deleteBeforeCutoffBatched(ctx, db, "chat_history_message", "created_at", result.Cutoff)
		if err != nil {
			return result, err
		}
		result.DeletedChatHistoryMessage = deleted
	}
	return result, nil
}

func tableExists(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var count int
	if err := db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = ?`,
		strings.TrimSpace(name),
	).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func deleteBeforeCutoff(ctx context.Context, tx *sql.Tx, tableName, timeColumn, cutoff string) (int64, error) {
	stmt := fmt.Sprintf(`DELETE FROM %s WHERE %s != '' AND %s < ?`, tableName, timeColumn, timeColumn)
	res, err := tx.ExecContext(ctx, stmt, cutoff)
	if err != nil {
		return 0, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rows, nil
}

func deleteBeforeCutoffBatched(ctx context.Context, db *sql.DB, tableName, timeColumn, cutoff string) (int64, error) {
	batchSize := cleanupDeleteBatchSize
	if batchSize <= 0 {
		batchSize = 500
	}

	var total int64
	for {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return total, err
		}

		deleted, err := deleteBeforeCutoffBatch(ctx, tx, tableName, timeColumn, cutoff, batchSize)
		if err != nil {
			_ = tx.Rollback()
			return total, err
		}
		if err := tx.Commit(); err != nil {
			return total, err
		}

		total += deleted
		if deleted < int64(batchSize) {
			return total, nil
		}
	}
}

func deleteBeforeCutoffBatch(ctx context.Context, tx *sql.Tx, tableName, timeColumn, cutoff string, limit int) (int64, error) {
	stmt := fmt.Sprintf(`DELETE FROM %s WHERE rowid IN (SELECT rowid FROM %s WHERE %s != '' AND %s < ? LIMIT ?)`, tableName, tableName, timeColumn, timeColumn)
	res, err := tx.ExecContext(ctx, stmt, cutoff, limit)
	if err != nil {
		return 0, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rows, nil
}
