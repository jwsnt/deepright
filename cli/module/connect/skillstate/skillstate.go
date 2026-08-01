package skillstate

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const timeLayout = "2006-01-02T15:04:05.000"

// SkillDirState stores one explicit skill directory state bound to a chat session.
type SkillDirState struct {
	ChatID    string `json:"chatId"`
	Path      string `json:"path"`
	Disabled  bool   `json:"disabled"`
	UpdatedAt string `json:"updatedAt"`
}

func EnsureSchema(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS chat_skill_dir_state (
			chat_id TEXT NOT NULL,
			path TEXT NOT NULL,
			disabled INTEGER NOT NULL DEFAULT 1,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (chat_id, path)
		);
		CREATE INDEX IF NOT EXISTS idx_chat_skill_dir_state_updated_at
			ON chat_skill_dir_state(updated_at);
	`)
	if err != nil {
		return err
	}
	rows, err := db.Query(`PRAGMA table_info(chat_skill_dir_state)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	hasDisabled := false
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue interface{}
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		if name == "disabled" {
			hasDisabled = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if hasDisabled {
		return nil
	}
	// Older installations only recorded disabled directories. Preserve those
	// entries as disabled while adding explicit enabled-state support.
	_, err = db.Exec(`ALTER TABLE chat_skill_dir_state ADD COLUMN disabled INTEGER NOT NULL DEFAULT 1`)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
		return nil
	}
	return err
}

func NormalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(path)
}

func IsSameOrDescendant(base, target string) bool {
	base = NormalizePath(base)
	target = NormalizePath(target)
	if base == "" || target == "" {
		return false
	}
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func normalizePathList(paths []string) []string {
	normalized := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		path = NormalizePath(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		normalized = append(normalized, path)
	}
	sort.Slice(normalized, func(i, j int) bool {
		if len(normalized[i]) != len(normalized[j]) {
			return len(normalized[i]) < len(normalized[j])
		}
		return normalized[i] < normalized[j]
	})
	pruned := make([]string, 0, len(normalized))
	for _, path := range normalized {
		inherited := false
		for _, kept := range pruned {
			if IsSameOrDescendant(kept, path) {
				inherited = true
				break
			}
		}
		if inherited {
			continue
		}
		pruned = append(pruned, path)
	}
	sort.Strings(pruned)
	return pruned
}

func normalizeUniquePaths(paths []string) []string {
	normalized := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		path = NormalizePath(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		normalized = append(normalized, path)
	}
	sort.Strings(normalized)
	return normalized
}

func DisabledStatus(paths []string, path string) (self bool, inherited bool) {
	path = NormalizePath(path)
	if path == "" {
		return false, false
	}
	for _, disabledPath := range normalizePathList(paths) {
		if disabledPath == path {
			return true, false
		}
		if IsSameOrDescendant(disabledPath, path) {
			return false, true
		}
	}
	return false, false
}

func ApplyDisabledPaths(current []string, path string, disabled bool) []string {
	path = NormalizePath(path)
	if path == "" {
		return normalizePathList(current)
	}

	current = normalizePathList(current)
	next := make([]string, 0, len(current)+1)
	hasAncestor := false
	for _, existing := range current {
		if existing == path || IsSameOrDescendant(path, existing) {
			continue
		}
		if IsSameOrDescendant(existing, path) {
			hasAncestor = true
		}
		next = append(next, existing)
	}
	if disabled && !hasAncestor {
		next = append(next, path)
	}
	return normalizePathList(next)
}

func ListDisabledPaths(db *sql.DB, chatID string) ([]string, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return nil, nil
	}
	if err := EnsureSchema(db); err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT path FROM chat_skill_dir_state WHERE chat_id = ? AND disabled != 0 ORDER BY path`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]string, 0)
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		items = append(items, path)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return normalizePathList(items), nil
}

// EnsureDefaultDisabled records each newly discovered skill directory as
// disabled without changing a state the user has already explicitly chosen.
func EnsureDefaultDisabled(db *sql.DB, chatID string, paths []string) ([]string, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return nil, nil
	}
	paths = normalizeUniquePaths(paths)
	if err := EnsureSchema(db); err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return ListDisabledPaths(db, chatID)
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	updatedAt := time.Now().Format(timeLayout)
	for _, path := range paths {
		if _, err := tx.Exec(`
			INSERT INTO chat_skill_dir_state (chat_id, path, disabled, updated_at)
			VALUES (?, ?, 1, ?)
			ON CONFLICT(chat_id, path) DO NOTHING
		`, chatID, path, updatedAt); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return ListDisabledPaths(db, chatID)
}

func SetDisabled(db *sql.DB, chatID, path string, disabled bool) ([]string, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return nil, fmt.Errorf("chatId is required")
	}
	path = NormalizePath(path)
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if err := EnsureSchema(db); err != nil {
		return nil, err
	}

	updatedAt := time.Now().Format(timeLayout)
	state := 0
	if disabled {
		state = 1
	}
	if _, err := db.Exec(`
		INSERT INTO chat_skill_dir_state (chat_id, path, disabled, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(chat_id, path) DO UPDATE SET
			disabled = excluded.disabled,
			updated_at = excluded.updated_at
	`, chatID, path, state, updatedAt); err != nil {
		return nil, err
	}
	return ListDisabledPaths(db, chatID)
}
