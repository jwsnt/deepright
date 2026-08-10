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
	AgentID    string `json:"agentId"`
	ChatID     string `json:"chatId"`
	Sandbox    string `json:"sandbox"`
	AllowedDir string `json:"allowedDir"`
	UpdatedAt  string `json:"updatedAt"`
}

type columnInfo struct {
	Name string
	Type string
	PK   int
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
			PK:   pk,
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
			chat_id TEXT NOT NULL DEFAULT '',
			sandbox_exe TEXT NOT NULL DEFAULT '',
			allowed_dir TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL,
			PRIMARY KEY (chat_id)
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
	if len(cols) != 4 {
		return true
	}
	expected := map[string]string{
		"chat_id":     "TEXT",
		"sandbox_exe": "TEXT",
		"allowed_dir": "TEXT",
		"updated_at":  "TEXT",
	}
	for _, col := range cols {
		wantType, ok := expected[col.Name]
		if !ok || col.Type != wantType {
			return true
		}
		if col.Name == "chat_id" && col.PK != 1 {
			return true
		}
	}
	return false
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
	if chatID == "" {
		return State{}, false, fmt.Errorf("chatId is required")
	}
	if err := EnsureSchema(db); err != nil {
		return State{}, false, err
	}

	var value string
	var allowedDir string
	var updatedAt string
	err := db.QueryRow(
		`SELECT sandbox_exe, allowed_dir, updated_at FROM cli_sandbox_state WHERE chat_id = ?`,
		chatID,
	).Scan(&value, &allowedDir, &updatedAt)
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
		AgentID:    agentID,
		ChatID:     chatID,
		Sandbox:    mode,
		AllowedDir: strings.TrimSpace(allowedDir),
		UpdatedAt:  updatedAt,
	}, true, nil
}

func Set(db *sql.DB, agentID, chatID, mode string) (State, error) {
	return setAt(db, agentID, chatID, mode, "", time.Now())
}

func SetWithDirectory(db *sql.DB, agentID, chatID, mode, allowedDir string) (State, error) {
	return setAt(db, agentID, chatID, mode, allowedDir, time.Now())
}

func setAt(db *sql.DB, agentID, chatID, mode, allowedDir string, now time.Time) (State, error) {
	agentID = strings.TrimSpace(agentID)
	chatID = strings.TrimSpace(chatID)
	allowedDir = strings.TrimSpace(allowedDir)
	if chatID == "" {
		return State{}, fmt.Errorf("chatId is required")
	}
	if err := EnsureSchema(db); err != nil {
		return State{}, err
	}
	rawMode := strings.TrimSpace(mode)
	mode = NormalizeMode(mode)
	if rawMode != "" && mode == "" {
		return State{}, fmt.Errorf("sandbox must be one of filepick,net,filepick_net")
	}
	if mode != ModeFilePick && mode != ModeFilePickNet {
		allowedDir = ""
	}
	if mode == "" {
		if err := Delete(db, agentID, chatID); err != nil {
			return State{}, err
		}
		return State{AgentID: agentID, ChatID: chatID}, nil
	}
	updatedAt := now.Format(timeLayout)
	_, err := db.Exec(
		`INSERT INTO cli_sandbox_state (chat_id, sandbox_exe, allowed_dir, updated_at)
		 VALUES (?,?,?,?)
		 ON CONFLICT(chat_id) DO UPDATE SET sandbox_exe = excluded.sandbox_exe, allowed_dir = excluded.allowed_dir, updated_at = excluded.updated_at`,
		chatID,
		mode,
		allowedDir,
		updatedAt,
	)
	if err != nil {
		return State{}, err
	}
	return State{
		AgentID:    agentID,
		ChatID:     chatID,
		Sandbox:    mode,
		AllowedDir: allowedDir,
		UpdatedAt:  updatedAt,
	}, nil
}

func Delete(db *sql.DB, agentID, chatID string) error {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return fmt.Errorf("chatId is required")
	}
	if err := EnsureSchema(db); err != nil {
		return err
	}
	_, err := db.Exec(`DELETE FROM cli_sandbox_state WHERE chat_id = ?`, chatID)
	return err
}
