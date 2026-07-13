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
	DefaultRetentionDays = 30
	DefaultMessage       = "正在清理过期日志，请稍后"
	timeLayout           = "2006-01-02T15:04:05.000"
)

var cleanupDeleteBatchSize = 500

type Result struct {
	RetentionDays          int
	Cutoff                 string
	DeletedAgentMessageLog int64
	DeletedChatLog         int64
}

type Status struct {
	Checked                bool
	Running                bool
	Message                string
	RetentionDays          int
	Cutoff                 string
	StartedAt              string
	FinishedAt             string
	DeletedAgentMessageLog int64
	DeletedChatLog         int64
	Error                  string
}

type Manager struct {
	mu      sync.RWMutex
	started bool
	status  Status
}

func NewManager() *Manager {
	return &Manager{
		status: Status{
			Message:       DefaultMessage,
			RetentionDays: DefaultRetentionDays,
		},
	}
}

func (m *Manager) Snapshot() Status {
	if m == nil {
		return Status{
			Checked:       true,
			Message:       DefaultMessage,
			RetentionDays: DefaultRetentionDays,
			Error:         "log retention manager is nil",
		}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func (m *Manager) MarkChecked(err error) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = true
	m.status.Checked = true
	m.status.Running = false
	m.status.Message = DefaultMessage
	m.status.FinishedAt = time.Now().Format(timeLayout)
	if err != nil {
		m.status.Error = strings.TrimSpace(err.Error())
	}
}

func (m *Manager) startAsyncRunner(
	ctx context.Context,
	retentionDays int,
	logf func(string, ...any),
	run func(context.Context, time.Time, int) (Result, error),
) bool {
	if m == nil {
		return false
	}
	if retentionDays <= 0 {
		retentionDays = DefaultRetentionDays
	}
	if run == nil {
		m.MarkChecked(fmt.Errorf("log retention runner is nil"))
		return false
	}

	startedAt := time.Now()
	cutoff := startedAt.AddDate(0, 0, -retentionDays).Format(timeLayout)

	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return false
	}
	m.started = true
	m.status = Status{
		Checked:       false,
		Running:       true,
		Message:       DefaultMessage,
		RetentionDays: retentionDays,
		Cutoff:        cutoff,
		StartedAt:     startedAt.Format(timeLayout),
	}
	m.mu.Unlock()

	go func() {
		result, err := run(ctx, startedAt, retentionDays)
		finishedAt := time.Now().Format(timeLayout)

		m.mu.Lock()
		m.status.Checked = true
		m.status.Running = false
		m.status.Message = DefaultMessage
		m.status.RetentionDays = retentionDays
		m.status.Cutoff = result.Cutoff
		if m.status.Cutoff == "" {
			m.status.Cutoff = cutoff
		}
		m.status.FinishedAt = finishedAt
		m.status.DeletedAgentMessageLog = result.DeletedAgentMessageLog
		m.status.DeletedChatLog = result.DeletedChatLog
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
			"[log-retention] cleanup finished cutoff=%s retention_days=%d agent_message_log=%d chat_log=%d",
			result.Cutoff,
			result.RetentionDays,
			result.DeletedAgentMessageLog,
			result.DeletedChatLog,
		)
	}()
	return true
}

func (m *Manager) StartAsync(ctx context.Context, db *sql.DB, retentionDays int, logf func(string, ...any)) bool {
	if db == nil {
		m.MarkChecked(fmt.Errorf("log retention db is nil"))
		return false
	}
	return m.startAsyncRunner(ctx, retentionDays, logf, func(ctx context.Context, startedAt time.Time, retentionDays int) (Result, error) {
		return CleanupExpiredLogs(ctx, db, startedAt, retentionDays)
	})
}

func (m *Manager) StartAsyncWithDBOpener(ctx context.Context, retentionDays int, openDB func() (*sql.DB, error), logf func(string, ...any)) bool {
	if openDB == nil {
		m.MarkChecked(fmt.Errorf("log retention db opener is nil"))
		return false
	}
	return m.startAsyncRunner(ctx, retentionDays, logf, func(ctx context.Context, startedAt time.Time, retentionDays int) (Result, error) {
		db, err := openDB()
		if err != nil {
			return Result{
				RetentionDays: retentionDays,
				Cutoff:        startedAt.AddDate(0, 0, -retentionDays).Format(timeLayout),
			}, err
		}
		if db == nil {
			return Result{
				RetentionDays: retentionDays,
				Cutoff:        startedAt.AddDate(0, 0, -retentionDays).Format(timeLayout),
			}, fmt.Errorf("log retention db is nil")
		}
		defer db.Close()
		return CleanupExpiredLogs(ctx, db, startedAt, retentionDays)
	})
}

func CleanupExpiredLogs(ctx context.Context, db *sql.DB, now time.Time, retentionDays int) (Result, error) {
	if db == nil {
		return Result{}, fmt.Errorf("log retention db is nil")
	}
	if retentionDays <= 0 {
		retentionDays = DefaultRetentionDays
	}
	if ctx == nil {
		ctx = context.Background()
	}

	result := Result{
		RetentionDays: retentionDays,
		Cutoff:        now.AddDate(0, 0, -retentionDays).Format(timeLayout),
	}

	if exists, err := tableExists(ctx, db, "agent_message_log"); err != nil {
		return result, err
	} else if exists {
		deleted, err := deleteBeforeCutoffBatched(ctx, db, "agent_message_log", result.Cutoff)
		if err != nil {
			return result, err
		}
		result.DeletedAgentMessageLog = deleted
	}

	if exists, err := tableExists(ctx, db, "chat_log"); err != nil {
		return result, err
	} else if exists {
		deleted, err := deleteBeforeCutoffBatched(ctx, db, "chat_log", result.Cutoff)
		if err != nil {
			return result, err
		}
		result.DeletedChatLog = deleted
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

func deleteBeforeCutoff(ctx context.Context, tx *sql.Tx, tableName, cutoff string) (int64, error) {
	stmt := fmt.Sprintf(`DELETE FROM %s WHERE created_at != '' AND created_at < ?`, tableName)
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

func deleteBeforeCutoffBatched(ctx context.Context, db *sql.DB, tableName, cutoff string) (int64, error) {
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

		deleted, err := deleteBeforeCutoffBatch(ctx, tx, tableName, cutoff, batchSize)
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

func deleteBeforeCutoffBatch(ctx context.Context, tx *sql.Tx, tableName, cutoff string, limit int) (int64, error) {
	stmt := fmt.Sprintf(`DELETE FROM %s WHERE rowid IN (SELECT rowid FROM %s WHERE created_at != '' AND created_at < ? LIMIT ?)`, tableName, tableName)
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
