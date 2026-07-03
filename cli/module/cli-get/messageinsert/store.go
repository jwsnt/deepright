package messageinsert

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	StatusPending   = 0
	StatusUploaded  = 1
	StatusCancelled = 2
)

type Item struct {
	AgentID   string `json:"agentId"`
	ChatID    string `json:"chatId"`
	Mid       string `json:"mid"`
	Message   string `json:"message"`
	Status    int    `json:"status"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func EnsureSchema(db *sql.DB) error {
	if db == nil {
		return nil
	}
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS message_insert (
			chat_id TEXT NOT NULL,
			mid TEXT NOT NULL,
			agent_id TEXT NOT NULL DEFAULT '',
			message TEXT NOT NULL DEFAULT '',
			status INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (chat_id, mid)
		);
		CREATE INDEX IF NOT EXISTS idx_message_insert_chat_status_time
			ON message_insert(chat_id, status, created_at);
		CREATE INDEX IF NOT EXISTS idx_message_insert_chat_updated_time
			ON message_insert(chat_id, updated_at);
	`)
	return err
}

func UpsertPending(db *sql.DB, agentID, chatID, mid, message string, now time.Time) (Item, error) {
	agentID = strings.TrimSpace(agentID)
	chatID = strings.TrimSpace(chatID)
	mid = strings.TrimSpace(mid)
	message = strings.TrimSpace(message)
	if chatID == "" {
		return Item{}, fmt.Errorf("chatId is required")
	}
	if mid == "" {
		return Item{}, fmt.Errorf("mid is required")
	}
	if message == "" {
		return Item{}, fmt.Errorf("message is required")
	}
	if err := EnsureSchema(db); err != nil {
		return Item{}, err
	}
	ts := nowOrUTC(now)
	if _, err := db.Exec(`INSERT INTO message_insert (chat_id, mid, agent_id, message, status, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?)
		ON CONFLICT(chat_id, mid) DO UPDATE SET
			agent_id = excluded.agent_id,
			message = excluded.message,
			status = excluded.status,
			updated_at = excluded.updated_at`,
		chatID, mid, agentID, message, StatusPending, ts, ts); err != nil {
		return Item{}, err
	}
	var item Item
	err := db.QueryRow(`SELECT agent_id, chat_id, mid, message, status, created_at, updated_at
		FROM message_insert WHERE chat_id = ? AND mid = ?`,
		chatID, mid,
	).Scan(&item.AgentID, &item.ChatID, &item.Mid, &item.Message, &item.Status, &item.CreatedAt, &item.UpdatedAt)
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
	rows, err := db.Query(`SELECT agent_id, chat_id, mid, message, status, created_at, updated_at
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
		if err := rows.Scan(&item.AgentID, &item.ChatID, &item.Mid, &item.Message, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func MarkUploaded(db *sql.DB, chatID string, mids []string, now time.Time) (int64, error) {
	chatID = strings.TrimSpace(chatID)
	mids = uniqueMIDs(mids)
	if chatID == "" || len(mids) == 0 {
		return 0, nil
	}
	if err := EnsureSchema(db); err != nil {
		return 0, err
	}
	args := make([]interface{}, 0, len(mids)+3)
	args = append(args, StatusUploaded, nowOrUTC(now), chatID)
	for _, mid := range mids {
		args = append(args, mid)
	}
	res, err := db.Exec(`UPDATE message_insert
		SET status = ?, updated_at = ?
		WHERE chat_id = ? AND status = `+fmt.Sprintf("%d", StatusPending)+` AND mid IN (`+placeholders(len(mids))+`)`, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func uniqueMIDs(mids []string) []string {
	if len(mids) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(mids))
	result := make([]string, 0, len(mids))
	for _, mid := range mids {
		mid = strings.TrimSpace(mid)
		if mid == "" {
			continue
		}
		if _, ok := seen[mid]; ok {
			continue
		}
		seen[mid] = struct{}{}
		result = append(result, mid)
	}
	return result
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
