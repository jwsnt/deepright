package eventlog

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

const (
	TypeChatCompletionRequest  = 0
	TypeChatCompletionResponse = 1
	TypeCLIGet                 = 2
	TypeCLIPub                 = 3
)

type Entry struct {
	ID        int
	AgentID   string
	ChatID    string
	Content   string
	Type      int
	CreatedAt string
}

type Logger struct {
	store *Store
	ch    chan Entry
	wg    sync.WaitGroup
}

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(0)

	store := &Store{db: db}
	if err := store.ensureSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func NewLogger(path string, buffer int) (*Logger, error) {
	if buffer <= 0 {
		buffer = 256
	}
	store, err := Open(path)
	if err != nil {
		return nil, err
	}
	logger := &Logger{
		store: store,
		ch:    make(chan Entry, buffer),
	}
	logger.wg.Add(1)
	go func() {
		defer logger.wg.Done()
		for entry := range logger.ch {
			_ = logger.store.Insert(entry)
		}
	}()
	return logger, nil
}

func (l *Logger) Append(entry Entry) {
	if l == nil {
		return
	}
	entry.Content = NormalizeContent(entry.Type, entry.Content)
	if strings.TrimSpace(entry.CreatedAt) == "" {
		entry.CreatedAt = nowString()
	}
	select {
	case l.ch <- entry:
	default:
		go func() {
			_ = l.store.Insert(entry)
		}()
	}
}

func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	close(l.ch)
	l.wg.Wait()
	return l.store.Close()
}

func (l *Logger) Query(agentID, chatID, timeline string, types ...int) ([]Entry, error) {
	if l == nil {
		return nil, nil
	}
	return l.store.Query(agentID, chatID, timeline, types...)
}

func (l *Logger) QueryAll(agentID, chatID string) ([]Entry, error) {
	if l == nil {
		return nil, nil
	}
	return l.store.QueryAll(agentID, chatID)
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Insert(entry Entry) error {
	if s == nil || s.db == nil {
		return nil
	}
	entry.Content = NormalizeContent(entry.Type, entry.Content)
	if strings.TrimSpace(entry.CreatedAt) == "" {
		entry.CreatedAt = nowString()
	}
	_, err := s.db.Exec(
		`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES (?,?,?,?,?)`,
		strings.TrimSpace(entry.AgentID),
		strings.TrimSpace(entry.ChatID),
		entry.Content,
		entry.Type,
		entry.CreatedAt,
	)
	return err
}

func (s *Store) Query(agentID, chatID, timeline string, types ...int) ([]Entry, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if len(types) == 0 {
		return []Entry{}, nil
	}
	args := make([]any, 0, 3+len(types))
	trimmedAgentID := strings.TrimSpace(agentID)
	trimmedChatID := strings.TrimSpace(chatID)
	args = append(args, trimmedChatID, strings.TrimSpace(timeline))
	holders := make([]string, 0, len(types))
	for _, item := range types {
		holders = append(holders, "?")
		args = append(args, item)
	}
	query := `SELECT id, agent_id, chat_id, content, log_type, created_at
		 FROM agent_message_log
		 WHERE chat_id = ? AND created_at > ?
		   AND log_type IN (` + strings.Join(holders, ",") + `)
		 ORDER BY id`
	if trimmedAgentID != "" {
		args = append(args[:0], trimmedAgentID, trimmedChatID, strings.TrimSpace(timeline))
		for _, item := range types {
			args = append(args, item)
		}
		query = `SELECT id, agent_id, chat_id, content, log_type, created_at
		 FROM agent_message_log
		 WHERE agent_id = ? AND chat_id = ? AND created_at > ?
		   AND log_type IN (` + strings.Join(holders, ",") + `)
		 ORDER BY id`
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]Entry, 0)
	for rows.Next() {
		var item Entry
		if err := rows.Scan(&item.ID, &item.AgentID, &item.ChatID, &item.Content, &item.Type, &item.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, item)
	}
	return entries, rows.Err()
}

func (s *Store) QueryAll(agentID, chatID string) ([]Entry, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	trimmedAgentID := strings.TrimSpace(agentID)
	trimmedChatID := strings.TrimSpace(chatID)
	query := `SELECT id, agent_id, chat_id, content, log_type, created_at
		 FROM agent_message_log
		 WHERE chat_id = ?
		 ORDER BY created_at, id`
	args := []any{trimmedChatID}
	if trimmedAgentID != "" {
		query = `SELECT id, agent_id, chat_id, content, log_type, created_at
		 FROM agent_message_log
		 WHERE agent_id = ? AND chat_id = ?
		 ORDER BY created_at, id`
		args = []any{trimmedAgentID, trimmedChatID}
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]Entry, 0)
	for rows.Next() {
		var item Entry
		if err := rows.Scan(&item.ID, &item.AgentID, &item.ChatID, &item.Content, &item.Type, &item.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, item)
	}
	return entries, rows.Err()
}

func (s *Store) ensureSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS agent_message_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id TEXT NOT NULL DEFAULT '',
			chat_id TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			log_type INTEGER NOT NULL,
			created_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_agent_message_log_agent_chat_type_time
			ON agent_message_log(agent_id, chat_id, log_type, created_at);
		CREATE INDEX IF NOT EXISTS idx_agent_message_log_agent_chat_time
			ON agent_message_log(agent_id, chat_id, created_at);
	`)
	return err
}

func nowString() string {
	return time.Now().Format("2006-01-02T15:04:05.000")
}

func NormalizeContent(logType int, content string) string {
	if logType != TypeCLIPub {
		return content
	}
	normalized, ok := normalizeCLIPubContent(content)
	if !ok {
		return content
	}
	return normalized
}

type cliPubPayload struct {
	Status int32  `json:"status"`
	Suffix string `json:"suffix,omitempty"`
	Cmd    string `json:"cmd"`
	Tid    string `json:"tid,omitempty"`
}

func normalizeCLIPubContent(content string) (string, bool) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
		return "", false
	}
	var payload cliPubPayload
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return "", false
	}
	decoded, err := decodeGzipBase64(payload.Cmd)
	if err != nil {
		return "", false
	}
	return decoded, true
}

func decodeGzipBase64(encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return "", err
	}
	gzr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	defer gzr.Close()
	data, err := io.ReadAll(gzr)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
