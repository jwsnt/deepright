package connectsvc

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	messageSearchDefaultLimit = 50
	messageSearchMaximumLimit = 200
)

// MessageSnapshot is a source-owned, normalized message collection. Connect
// stores and queries it generically; each plugin owns source-specific parsing.
type MessageSnapshot struct {
	Source   string                  `json:"source"`
	Messages []MessageSnapshotRecord `json:"messages"`
}

type MessageSnapshotRecord struct {
	MessageID   string `json:"messageId"`
	SenderID    string `json:"senderId"`
	Content     string `json:"content,omitempty"`
	MessageType string `json:"messageType"`
	SentAt      int64  `json:"sentAt"`
}

type MessageSender struct {
	SenderID      string `json:"senderId"`
	LastMessageAt string `json:"lastMessageAt"`
}

type MessageSnapshotItem struct {
	MessageID string `json:"messageId"`
	SenderID  string `json:"senderId"`
	Content   string `json:"content"`
	SentAt    string `json:"sentAt"`
}

type MessageSnapshotPage struct {
	Total  int                   `json:"total"`
	Limit  int                   `json:"limit"`
	Offset int                   `json:"offset"`
	Items  []MessageSnapshotItem `json:"items"`
}

type MessageSnapshotSearch struct {
	Source   string
	SenderID string
	Query    string
	Limit    int
	Offset   int
}

type sqlExecer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func (s *Service) ListMessageSenders(source string, window time.Duration, now time.Time) ([]MessageSender, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, errors.New("source is required")
	}
	if window <= 0 {
		return nil, errors.New("window-hours must be a positive integer")
	}
	if now.IsZero() {
		now = time.Now()
	}
	cutoff := now.Add(-window).UnixMilli()
	rows, err := s.db.Query(`
		SELECT sender_id, MAX(sent_at)
		FROM connect_message_snapshot
		WHERE source = ? AND sent_at >= ? AND sender_id != ''
		GROUP BY sender_id
		ORDER BY MAX(sent_at) DESC, sender_id ASC
	`, source, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MessageSender, 0)
	for rows.Next() {
		var item MessageSender
		var sentAt int64
		if err := rows.Scan(&item.SenderID, &sentAt); err != nil {
			return nil, err
		}
		item.LastMessageAt = formatMessageSnapshotTime(sentAt)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) SearchMessageSnapshots(window time.Duration, input MessageSnapshotSearch, now time.Time) (*MessageSnapshotPage, error) {
	input.Source = strings.TrimSpace(input.Source)
	if input.Source == "" {
		return nil, errors.New("source is required")
	}
	if window <= 0 {
		return nil, errors.New("window-hours must be a positive integer")
	}
	terms := []string(nil)
	if strings.TrimSpace(input.Query) != "" {
		var err error
		terms, err = ParseMessageSnapshotSearchTerms(input.Query)
		if err != nil {
			return nil, err
		}
	}
	input.SenderID = strings.TrimSpace(input.SenderID)
	if input.Limit == 0 {
		input.Limit = messageSearchDefaultLimit
	}
	if input.Limit < 0 {
		return nil, errors.New("limit must be positive")
	}
	if input.Limit > messageSearchMaximumLimit {
		return nil, fmt.Errorf("limit must not exceed %d", messageSearchMaximumLimit)
	}
	if input.Offset < 0 {
		return nil, errors.New("offset must not be negative")
	}
	if now.IsZero() {
		now = time.Now()
	}

	from, where, args := buildMessageSnapshotSearchSQL(input.Source, input.SenderID, terms, now.Add(-window).UnixMilli())
	countQuery := "SELECT COUNT(*) " + from + " WHERE " + strings.Join(where, " AND ")
	var total int
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	query := "SELECT m.message_id, m.sender_id, m.content, m.sent_at " + from +
		" WHERE " + strings.Join(where, " AND ") +
		" ORDER BY m.sent_at DESC, m.message_id DESC LIMIT ? OFFSET ?"
	queryArgs := append(append([]any{}, args...), input.Limit, input.Offset)
	rows, err := s.db.Query(query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MessageSnapshotItem, 0)
	for rows.Next() {
		var item MessageSnapshotItem
		var sentAt int64
		if err := rows.Scan(&item.MessageID, &item.SenderID, &item.Content, &sentAt); err != nil {
			return nil, err
		}
		item.SentAt = formatMessageSnapshotTime(sentAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &MessageSnapshotPage{Total: total, Limit: input.Limit, Offset: input.Offset, Items: items}, nil
}

func buildMessageSnapshotSearchSQL(source, senderID string, terms []string, cutoff int64) (string, []string, []any) {
	from := "FROM connect_message_snapshot m"
	where := []string{"m.source = ?", "m.sent_at >= ?", "m.message_type = 'text'"}
	args := []any{source, cutoff}
	if senderID = strings.TrimSpace(senderID); senderID != "" {
		where = append(where, "m.sender_id = ?")
		args = append(args, senderID)
	}
	if shouldUseMessageSnapshotFTS(terms) {
		from += " JOIN connect_message_snapshot_fts f ON f.source = m.source AND f.message_id = m.message_id"
		where = append(where, "f.content MATCH ?")
		args = append(args, messageSnapshotFTSQuery(terms))
	}
	for _, term := range terms {
		where = append(where, "instr(m.content_folded, ?) > 0")
		args = append(args, strings.ToLower(term))
	}
	return from, where, args
}

func ParseMessageSnapshotSearchTerms(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("query is required")
	}
	terms := make([]string, 0)
	var current strings.Builder
	inQuote := false
	flush := func() {
		term := strings.TrimSpace(current.String())
		if term != "" {
			terms = append(terms, term)
		}
		current.Reset()
	}
	for _, r := range raw {
		switch {
		case r == '"':
			if inQuote {
				flush()
			}
			inQuote = !inQuote
		case !inQuote && (r == ' ' || r == '\t' || r == '\n' || r == '\r'):
			flush()
		default:
			current.WriteRune(r)
		}
	}
	if inQuote {
		return nil, errors.New("query contains an unclosed quote")
	}
	flush()
	if len(terms) == 0 {
		return nil, errors.New("query is required")
	}
	return terms, nil
}

func shouldUseMessageSnapshotFTS(terms []string) bool {
	for _, term := range terms {
		if utf8.RuneCountInString(term) < 3 {
			return false
		}
	}
	return len(terms) > 0
}

func messageSnapshotFTSQuery(terms []string) string {
	parts := make([]string, 0, len(terms))
	for _, term := range terms {
		parts = append(parts, `"`+strings.ReplaceAll(strings.ToLower(term), `"`, `""`)+`"`)
	}
	return strings.Join(parts, " AND ")
}

func formatMessageSnapshotTime(unixMS int64) string {
	return time.UnixMilli(unixMS).UTC().Format(time.RFC3339)
}

func syncMessageSnapshot(exec sqlExecer, raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if exec == nil {
		return errors.New("message snapshot database is required")
	}
	var snapshot MessageSnapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return fmt.Errorf("parse message-snapshot: %w", err)
	}
	snapshot.Source = strings.TrimSpace(snapshot.Source)
	if snapshot.Source == "" {
		return errors.New("message-snapshot source is required")
	}
	for _, item := range snapshot.Messages {
		item.MessageID = strings.TrimSpace(item.MessageID)
		item.SenderID = strings.TrimSpace(item.SenderID)
		item.MessageType = strings.ToLower(strings.TrimSpace(item.MessageType))
		if item.MessageID == "" || item.SenderID == "" || item.MessageType == "" || item.SentAt <= 0 {
			return errors.New("message-snapshot messages require messageId, senderId, messageType, and sentAt")
		}
		if _, err := exec.Exec(`
			INSERT INTO connect_message_snapshot (source, message_id, sender_id, content, content_folded, message_type, sent_at)
			VALUES (?,?,?,?,?,?,?)
			ON CONFLICT(source, message_id) DO UPDATE SET
				sender_id = excluded.sender_id,
				content = excluded.content,
				content_folded = excluded.content_folded,
				message_type = excluded.message_type,
				sent_at = excluded.sent_at
		`, snapshot.Source, item.MessageID, item.SenderID, strings.TrimSpace(item.Content), strings.ToLower(strings.TrimSpace(item.Content)), item.MessageType, item.SentAt); err != nil {
			return err
		}
		if _, err := exec.Exec(`DELETE FROM connect_message_snapshot_fts WHERE source = ? AND message_id = ?`, snapshot.Source, item.MessageID); err != nil {
			return err
		}
		if _, err := exec.Exec(`INSERT INTO connect_message_snapshot_fts (source, message_id, content) VALUES (?, ?, ?)`, snapshot.Source, item.MessageID, strings.ToLower(strings.TrimSpace(item.Content))); err != nil {
			return err
		}
	}
	return nil
}

func ParseMessageSnapshotWindowHours(raw string) (time.Duration, error) {
	hours, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || hours <= 0 {
		return 0, errors.New("window-hours must be a positive integer")
	}
	return time.Duration(hours) * time.Hour, nil
}
