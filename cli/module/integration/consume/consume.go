package consume

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	queryTimeLayout          = "20060102-150405"
	queryTimeReadableLayout  = "2006-01-02 15:04:05"
	queryTimeReadableLayoutT = "2006-01-02T15:04:05"
	tableName                = "token_consume_log"
)

type Record struct {
	Thinking  int    `json:"thinking"`
	Input     int    `json:"input"`
	Total     int    `json:"total"`
	Cache     int    `json:"cache"`
	Model     string `json:"model"`
	AgentID   string `json:"agentId"`
	Function  string `json:"function"`
	Timestamp int64  `json:"timestamp"`
}

type Summary struct {
	Model    string `json:"model"`
	Thinking int    `json:"thinking"`
	Input    int    `json:"input"`
	Total    int    `json:"total"`
	Cache    int    `json:"cache"`
}

type QueryResult struct {
	Details []Record  `json:"details"`
	Summary []Summary `json:"summary"`
}

func EnsureSchema(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS ` + tableName + ` (
id INTEGER PRIMARY KEY AUTOINCREMENT,
thinking INTEGER NOT NULL DEFAULT 0,
input_tokens INTEGER NOT NULL DEFAULT 0,
total_tokens INTEGER NOT NULL DEFAULT 0,
cache_tokens INTEGER NOT NULL DEFAULT 0,
model TEXT NOT NULL,
agent_id TEXT NOT NULL,
function_name TEXT NOT NULL,
created_at INTEGER NOT NULL
)`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_token_consume_time_agent ON ` + tableName + `(created_at, agent_id)`); err != nil {
		return err
	}
	return nil
}

func NormalizeRecord(record Record, now time.Time) (Record, error) {
	record.Model = strings.TrimSpace(record.Model)
	record.AgentID = strings.TrimSpace(record.AgentID)
	record.Function = strings.TrimSpace(record.Function)
	if record.Model == "" {
		return Record{}, fmt.Errorf("model is required")
	}
	if record.AgentID == "" {
		return Record{}, fmt.Errorf("agentId is required")
	}
	if record.Function == "" {
		return Record{}, fmt.Errorf("function is required")
	}
	if record.Thinking < 0 || record.Input < 0 || record.Total < 0 || record.Cache < 0 {
		return Record{}, fmt.Errorf("token fields must be non-negative")
	}
	if record.Timestamp == 0 {
		record.Timestamp = now.UnixMilli()
	}
	if record.Timestamp < 0 {
		return Record{}, fmt.Errorf("timestamp must be non-negative")
	}
	return record, nil
}

func Insert(db *sql.DB, record Record) (Record, error) {
	if err := EnsureSchema(db); err != nil {
		return Record{}, err
	}
	normalized, err := NormalizeRecord(record, time.Now())
	if err != nil {
		return Record{}, err
	}
	if _, err := db.Exec(
		`INSERT INTO `+tableName+` (thinking, input_tokens, total_tokens, cache_tokens, model, agent_id, function_name, created_at) VALUES (?,?,?,?,?,?,?,?)`,
		normalized.Thinking,
		normalized.Input,
		normalized.Total,
		normalized.Cache,
		normalized.Model,
		normalized.AgentID,
		normalized.Function,
		normalized.Timestamp,
	); err != nil {
		return Record{}, err
	}
	return normalized, nil
}

func ParseQueryTime(value string, loc *time.Location) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("time is required")
	}
	if loc == nil {
		loc = time.Local
	}
	parsed, err := time.ParseInLocation(queryTimeLayout, value, loc)
	if err != nil {
		return 0, fmt.Errorf("time must use format yyyyMMdd-hhmmss")
	}
	return parsed.UnixMilli(), nil
}

func ParseFlexibleQueryTime(value string, loc *time.Location) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("time is required")
	}
	if loc == nil {
		loc = time.Local
	}
	for _, layout := range []string{queryTimeLayout, queryTimeReadableLayout, queryTimeReadableLayoutT, time.RFC3339} {
		parsed, err := time.ParseInLocation(layout, value, loc)
		if err == nil {
			return parsed.UnixMilli(), nil
		}
	}
	return 0, fmt.Errorf("time must use format yyyyMMdd-hhmmss or yyyy-MM-dd HH:mm:ss")
}

func validateQueryRange(startTimestamp, closeTimestamp int64, limit int) error {
	if startTimestamp < 0 {
		return fmt.Errorf("starTime must be non-negative")
	}
	if closeTimestamp < 0 {
		return fmt.Errorf("closeTime must be non-negative")
	}
	if closeTimestamp < startTimestamp {
		return fmt.Errorf("closeTime must be greater than or equal to starTime")
	}
	if limit < 0 {
		return fmt.Errorf("limit must be non-negative")
	}
	return nil
}

func queryRangeRecords(db *sql.DB, agentID string, startTimestamp, closeTimestamp int64, limit int, latestFirst bool) ([]Record, error) {
	if err := EnsureSchema(db); err != nil {
		return nil, err
	}
	if err := validateQueryRange(startTimestamp, closeTimestamp, limit); err != nil {
		return nil, err
	}
	agentID = strings.TrimSpace(agentID)

	query := `SELECT thinking, input_tokens, total_tokens, cache_tokens, model, agent_id, function_name, created_at FROM ` + tableName + ` WHERE created_at >= ? AND created_at <= ?`
	args := []interface{}{startTimestamp, closeTimestamp}
	if agentID != "" {
		query += ` AND agent_id = ?`
		args = append(args, agentID)
	}
	if latestFirst {
		query += ` ORDER BY created_at DESC, agent_id DESC, id DESC`
	} else {
		query += ` ORDER BY created_at ASC, agent_id ASC, id ASC`
	}
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	details := make([]Record, 0)
	for rows.Next() {
		var item Record
		if err := rows.Scan(&item.Thinking, &item.Input, &item.Total, &item.Cache, &item.Model, &item.AgentID, &item.Function, &item.Timestamp); err != nil {
			return nil, err
		}
		details = append(details, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return details, nil
}

func summarizeRecords(details []Record) QueryResult {
	result := QueryResult{
		Details: details,
		Summary: make([]Summary, 0),
	}
	byModel := make(map[string]*Summary)
	for _, item := range details {
		summary, ok := byModel[item.Model]
		if !ok {
			summary = &Summary{Model: item.Model}
			byModel[item.Model] = summary
		}
		summary.Thinking += item.Thinking
		summary.Input += item.Input
		summary.Total += item.Total
		summary.Cache += item.Cache
	}
	models := make([]string, 0, len(byModel))
	for model := range byModel {
		models = append(models, model)
	}
	sort.Strings(models)
	result.Summary = make([]Summary, 0, len(models))
	for _, model := range models {
		result.Summary = append(result.Summary, *byModel[model])
	}
	return result
}

func QueryRange(db *sql.DB, agentID string, startTimestamp, closeTimestamp int64, limit int) (QueryResult, error) {
	details, err := queryRangeRecords(db, agentID, startTimestamp, closeTimestamp, limit, false)
	if err != nil {
		return QueryResult{}, err
	}
	return summarizeRecords(details), nil
}

func QueryLatestRange(db *sql.DB, agentID string, startTimestamp, closeTimestamp int64, limit int) (QueryResult, error) {
	details, err := queryRangeRecords(db, agentID, startTimestamp, closeTimestamp, limit, true)
	if err != nil {
		return QueryResult{}, err
	}
	return summarizeRecords(details), nil
}

func Query(db *sql.DB, agentID string, lastTimestamp int64) (QueryResult, error) {
	return QueryRange(db, agentID, lastTimestamp+1, time.Now().UnixMilli(), 0)
}
