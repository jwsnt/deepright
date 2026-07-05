package messageinsert

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	StatusPending   = 0
	StatusUploaded  = 1
	StatusCancelled = 2
)

var (
	ErrAlreadyReported = errors.New("message insert already reported")
	ErrImmutable       = errors.New("message insert is immutable")
)

type Item struct {
	AgentID    string `json:"agentId"`
	ChatID     string `json:"chatId"`
	Tid        string `json:"tid"`
	Message    string `json:"message"`
	Status     int    `json:"status"`
	ReportedAt string `json:"reportedAt,omitempty"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

func EnsureSchema(db *sql.DB) error {
	if db == nil {
		return nil
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS message_insert (
			chat_id TEXT NOT NULL,
			mid TEXT NOT NULL,
			agent_id TEXT NOT NULL DEFAULT '',
			message TEXT NOT NULL DEFAULT '',
			status INTEGER NOT NULL DEFAULT 0,
			reported_at TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (chat_id, mid)
		)
	`); err != nil {
		return err
	}
	if err := ensureMessageInsertColumn(db, "reported_at", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	_, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_message_insert_chat_status_time
			ON message_insert(chat_id, status, created_at);
		CREATE INDEX IF NOT EXISTS idx_message_insert_chat_updated_time
			ON message_insert(chat_id, updated_at);
		CREATE INDEX IF NOT EXISTS idx_message_insert_chat_reported_time
			ON message_insert(chat_id, status, reported_at, created_at);
	`)
	return err
}

func UpsertPending(db *sql.DB, agentID, chatID, tid, message string, now time.Time) (Item, error) {
	agentID = strings.TrimSpace(agentID)
	chatID = strings.TrimSpace(chatID)
	tid = strings.TrimSpace(tid)
	message = strings.TrimSpace(message)
	if chatID == "" {
		return Item{}, fmt.Errorf("chatId is required")
	}
	if tid == "" {
		return Item{}, fmt.Errorf("tid is required")
	}
	if message == "" {
		return Item{}, fmt.Errorf("message is required")
	}
	if err := EnsureSchema(db); err != nil {
		return Item{}, err
	}
	ts := nowOrUTC(now)
	res, err := db.Exec(`INSERT INTO message_insert (chat_id, mid, agent_id, message, status, reported_at, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?)
		ON CONFLICT(chat_id, mid) DO UPDATE SET
			agent_id = excluded.agent_id,
			message = excluded.message,
			status = excluded.status,
			reported_at = excluded.reported_at,
			updated_at = excluded.updated_at
		WHERE message_insert.status = `+fmt.Sprintf("%d", StatusPending)+` AND message_insert.reported_at = ''`,
		chatID, tid, agentID, message, StatusPending, "", ts, ts)
	if err != nil {
		return Item{}, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return Item{}, err
	}
	if rows == 0 {
		existing, getErr := Get(db, chatID, tid)
		if getErr == nil {
			if strings.TrimSpace(existing.ReportedAt) != "" {
				return Item{}, ErrAlreadyReported
			}
			if existing.Status != StatusPending {
				return Item{}, ErrImmutable
			}
		}
		if getErr != nil && !errors.Is(getErr, sql.ErrNoRows) {
			return Item{}, getErr
		}
		return Item{}, ErrImmutable
	}
	return Get(db, chatID, tid)
}

func Cancel(db *sql.DB, chatID, tid string, now time.Time) (bool, error) {
	chatID = strings.TrimSpace(chatID)
	tid = strings.TrimSpace(tid)
	if chatID == "" {
		return false, fmt.Errorf("chatId is required")
	}
	if tid == "" {
		return false, fmt.Errorf("tid is required")
	}
	if err := EnsureSchema(db); err != nil {
		return false, err
	}
	res, err := db.Exec(`UPDATE message_insert
		SET status = ?, updated_at = ?
		WHERE chat_id = ? AND mid = ? AND status = ? AND reported_at = ''`,
		StatusCancelled, nowOrUTC(now), chatID, tid, StatusPending)
	if err != nil {
		return false, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if rows == 0 {
		existing, getErr := Get(db, chatID, tid)
		if getErr == nil {
			if strings.TrimSpace(existing.ReportedAt) != "" {
				return false, ErrAlreadyReported
			}
			if existing.Status != StatusPending {
				return false, ErrImmutable
			}
			return false, nil
		}
		if errors.Is(getErr, sql.ErrNoRows) {
			return false, nil
		}
		return false, getErr
	}
	return rows > 0, nil
}

func Get(db *sql.DB, chatID, tid string) (Item, error) {
	if err := EnsureSchema(db); err != nil {
		return Item{}, err
	}
	var item Item
	err := db.QueryRow(`SELECT agent_id, chat_id, mid, message, status, reported_at, created_at, updated_at
		FROM message_insert WHERE chat_id = ? AND mid = ?`,
		strings.TrimSpace(chatID), strings.TrimSpace(tid),
	).Scan(&item.AgentID, &item.ChatID, &item.Tid, &item.Message, &item.Status, &item.ReportedAt, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func ListPending(db *sql.DB, chatID string, limit int) ([]Item, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return nil, fmt.Errorf("chatId is required")
	}
	if limit <= 0 {
		limit = 5
	}
	if err := EnsureSchema(db); err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT agent_id, chat_id, mid, message, status, reported_at, created_at, updated_at
		FROM message_insert
		WHERE chat_id = ? AND status = ? AND reported_at = ''
		ORDER BY created_at, mid
		LIMIT ?`,
		chatID, StatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Item, 0, limit)
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.AgentID, &item.ChatID, &item.Tid, &item.Message, &item.Status, &item.ReportedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func ListActive(db *sql.DB, chatID string, limit int) ([]Item, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return nil, fmt.Errorf("chatId is required")
	}
	if limit <= 0 {
		limit = 20
	}
	if err := EnsureSchema(db); err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT agent_id, chat_id, mid, message, status, reported_at, created_at, updated_at
		FROM message_insert
		WHERE chat_id = ? AND status = ?
		ORDER BY created_at, mid
		LIMIT ?`,
		chatID, StatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Item, 0, limit)
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.AgentID, &item.ChatID, &item.Tid, &item.Message, &item.Status, &item.ReportedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func StatusByTIDs(db *sql.DB, chatID string, tids []string) (map[string]int, error) {
	chatID = strings.TrimSpace(chatID)
	tids = uniqueTIDs(tids)
	if chatID == "" {
		return nil, fmt.Errorf("chatId is required")
	}
	if len(tids) == 0 {
		return map[string]int{}, nil
	}
	if err := EnsureSchema(db); err != nil {
		return nil, err
	}
	args := make([]interface{}, 0, len(tids)+1)
	args = append(args, chatID)
	for _, tid := range tids {
		args = append(args, tid)
	}
	rows, err := db.Query(`SELECT mid, status FROM message_insert
		WHERE chat_id = ? AND mid IN (`+placeholders(len(tids))+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	statuses := make(map[string]int, len(tids))
	for rows.Next() {
		var tid string
		var status int
		if err := rows.Scan(&tid, &status); err != nil {
			return nil, err
		}
		statuses[strings.TrimSpace(tid)] = status
	}
	return statuses, rows.Err()
}

func MarkPublished(db *sql.DB, chatID string, tids []string, now time.Time) (int64, error) {
	chatID = strings.TrimSpace(chatID)
	tids = uniqueTIDs(tids)
	if chatID == "" || len(tids) == 0 {
		return 0, nil
	}
	if err := EnsureSchema(db); err != nil {
		return 0, err
	}
	ts := nowOrUTC(now)
	args := make([]interface{}, 0, len(tids)+3)
	args = append(args, ts, ts, chatID)
	for _, tid := range tids {
		args = append(args, tid)
	}
	res, err := db.Exec(`UPDATE message_insert
		SET reported_at = ?, updated_at = ?
		WHERE chat_id = ? AND status = `+fmt.Sprintf("%d", StatusPending)+` AND reported_at = '' AND mid IN (`+placeholders(len(tids))+`)`,
		args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func MarkUploaded(db *sql.DB, chatID string, tids []string, now time.Time) (int64, error) {
	chatID = strings.TrimSpace(chatID)
	tids = uniqueTIDs(tids)
	if chatID == "" || len(tids) == 0 {
		return 0, nil
	}
	if err := EnsureSchema(db); err != nil {
		return 0, err
	}
	args := make([]interface{}, 0, len(tids)+3)
	args = append(args, StatusUploaded, nowOrUTC(now), chatID)
	for _, tid := range tids {
		args = append(args, tid)
	}
	res, err := db.Exec(`UPDATE message_insert
		SET status = ?, updated_at = ?
		WHERE chat_id = ? AND status = `+fmt.Sprintf("%d", StatusPending)+` AND mid IN (`+placeholders(len(tids))+`)`, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func uniqueTIDs(tids []string) []string {
	if len(tids) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(tids))
	result := make([]string, 0, len(tids))
	for _, tid := range tids {
		tid = strings.TrimSpace(tid)
		if tid == "" {
			continue
		}
		if _, ok := seen[tid]; ok {
			continue
		}
		seen[tid] = struct{}{}
		result = append(result, tid)
	}
	return result
}

func ensureMessageInsertColumn(db *sql.DB, name, definition string) error {
	rows, err := db.Query(`PRAGMA table_info(message_insert)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var columnName, columnType string
		var notNull int
		var defaultValue interface{}
		var pk int
		if err := rows.Scan(&cid, &columnName, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if strings.EqualFold(strings.TrimSpace(columnName), strings.TrimSpace(name)) {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec(`ALTER TABLE message_insert ADD COLUMN ` + name + ` ` + definition)
	return err
}

func placeholders(count int) string {
	if count <= 0 {
		return ""
	}
	parts := make([]string, count)
	for i := 0; i < count; i++ {
		parts[i] = "?"
	}
	return strings.Join(parts, ",")
}

func nowOrUTC(now time.Time) string {
	if now.IsZero() {
		now = time.Now()
	}
	return now.UTC().Format(time.RFC3339Nano)
}
