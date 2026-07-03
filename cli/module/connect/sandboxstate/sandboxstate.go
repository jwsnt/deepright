package sandboxstate

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const timeLayout = "2006-01-02T15:04:05.000"

const (
	ModeFilePick    = "filepick"
	ModeNet         = "net"
	ModeFilePickNet = "filepick_net"
)

type State struct {
	AgentID   string `json:"agentId"`
	ChatID    string `json:"chatId"`
	Sandbox   string `json:"sandbox"`
	UpdatedAt string `json:"updatedAt"`
}

type columnInfo struct {
	Name string
	Type string
}

func EnsureSchema(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	rows, err := db.Query(`PRAGMA table_info(cli_sandbox_state)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var cols []columnInfo
	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultV   sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultV, &pk); err != nil {
			return err
		}
		cols = append(cols, columnInfo{
			Name: strings.TrimSpace(name),
			Type: strings.ToUpper(strings.TrimSpace(columnType)),
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if needsRebuild(cols) {
		if _, err := db.Exec(`DROP TABLE IF EXISTS cli_sandbox_state`); err != nil {
			return err
		}
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS cli_sandbox_state (
			agent_id TEXT NOT NULL,
			chat_id TEXT NOT NULL DEFAULT '',
			sandbox_exe TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL,
			PRIMARY KEY (agent_id, chat_id)
		);
		CREATE INDEX IF NOT EXISTS idx_cli_sandbox_state_updated_at
			ON cli_sandbox_state(updated_at);
	`)
	return err
}

func needsRebuild(cols []columnInfo) bool {
	if len(cols) == 0 {
		return false
	}
	for _, col := range cols {
		if col.Name == "sandbox_exe" {
			return col.Type != "TEXT"
		}
	}
	return true
}

func NormalizeMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case ModeFilePick:
		return ModeFilePick
	case ModeNet:
		return ModeNet
	case ModeFilePickNet:
		return ModeFilePickNet
	default:
		return ""
	}
}

func ValidModes() []string {
	return []string{ModeFilePick, ModeNet, ModeFilePickNet}
}

func Get(db *sql.DB, agentID, chatID string) (State, bool, error) {
	agentID = strings.TrimSpace(agentID)
	chatID = strings.TrimSpace(chatID)
	if agentID == "" {
		return State{}, false, fmt.Errorf("agentId is required")
	}
	if err := EnsureSchema(db); err != nil {
		return State{}, false, err
	}

	var value string
	var updatedAt string
	err := db.QueryRow(
		`SELECT sandbox_exe, updated_at FROM cli_sandbox_state WHERE agent_id = ? AND chat_id = ?`,
		agentID,
		chatID,
	).Scan(&value, &updatedAt)
	if err == sql.ErrNoRows {
		return State{AgentID: agentID, ChatID: chatID}, false, nil
	}
	if err != nil {
		return State{}, false, err
	}
	mode := NormalizeMode(value)
	if mode == "" {
		return State{AgentID: agentID, ChatID: chatID}, false, nil
	}
	return State{
		AgentID:   agentID,
		ChatID:    chatID,
		Sandbox:   mode,
		UpdatedAt: updatedAt,
	}, true, nil
}

func Set(db *sql.DB, agentID, chatID, mode string) (State, error) {
	return setAt(db, agentID, chatID, mode, time.Now())
}

func setAt(db *sql.DB, agentID, chatID, mode string, now time.Time) (State, error) {
	agentID = strings.TrimSpace(agentID)
	chatID = strings.TrimSpace(chatID)
	if agentID == "" {
		return State{}, fmt.Errorf("agentId is required")
	}
	if err := EnsureSchema(db); err != nil {
		return State{}, err
	}
	rawMode := strings.TrimSpace(mode)
	mode = NormalizeMode(mode)
	if rawMode != "" && mode == "" {
		return State{}, fmt.Errorf("sandbox must be one of filepick,net,filepick_net")
	}
	if mode == "" {
		if err := Delete(db, agentID, chatID); err != nil {
			return State{}, err
		}
		return State{AgentID: agentID, ChatID: chatID}, nil
	}
	updatedAt := now.Format(timeLayout)
	_, err := db.Exec(
		`INSERT INTO cli_sandbox_state (agent_id, chat_id, sandbox_exe, updated_at)
		 VALUES (?,?,?,?)
		 ON CONFLICT(agent_id, chat_id) DO UPDATE SET sandbox_exe = excluded.sandbox_exe, updated_at = excluded.updated_at`,
		agentID,
		chatID,
		mode,
		updatedAt,
	)
	if err != nil {
		return State{}, err
	}
	return State{
		AgentID:   agentID,
		ChatID:    chatID,
		Sandbox:   mode,
		UpdatedAt: updatedAt,
	}, nil
}

func Delete(db *sql.DB, agentID, chatID string) error {
	agentID = strings.TrimSpace(agentID)
	chatID = strings.TrimSpace(chatID)
	if agentID == "" {
		return fmt.Errorf("agentId is required")
	}
	if err := EnsureSchema(db); err != nil {
		return err
	}
	_, err := db.Exec(`DELETE FROM cli_sandbox_state WHERE agent_id = ? AND chat_id = ?`, agentID, chatID)
	return err
}
