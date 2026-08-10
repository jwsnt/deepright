package knowledgecore

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtimepaths"
	"strings"
	"sync"

	_ "github.com/glebarez/go-sqlite"
)

const (
	defaultKnowledgeDirName = "knowledge"
	defaultSQLiteFileName   = "data"
	defaultRuntimeAppName   = "deepright"
	defaultRuntimeBundleID  = "cn.deepright.integration"
)

var (
	knowledgeExecutableFn = os.Executable
	knowledgeUserHomeFn   = os.UserHomeDir
)

type State struct {
	Path            string `json:"path"`
	LastUpdate      int64  `json:"lastUpdate"`
	KnowledgeCommit bool   `json:"knowledgeCommit"`
}

func normalizeKnowledgeAgentID(agentID string) string {
	return strings.TrimSpace(agentID)
}

var sharedDB struct {
	mu   sync.Mutex
	path string
	db   *sql.DB
}

func normalizeAppDir(appDir string) (string, error) {
	appDir = strings.TrimSpace(appDir)
	if appDir == "" {
		appDir = "."
	}
	return filepath.Abs(appDir)
}

func knowledgeExecutablePath() string {
	exe, err := knowledgeExecutableFn()
	if err != nil {
		return ""
	}
	exe = strings.TrimSpace(exe)
	if exe == "" || strings.HasSuffix(exe, ".test") {
		return ""
	}
	if abs, err := filepath.Abs(exe); err == nil {
		return abs
	}
	return filepath.Clean(exe)
}

func knowledgeMacRuntimeBaseDir() string {
	if runtime.GOOS != "darwin" {
		return ""
	}
	if knowledgeExecutablePath() == "" {
		return ""
	}
	home, err := knowledgeUserHomeFn()
	if err != nil {
		return ""
	}
	home = strings.TrimSpace(home)
	if home == "" {
		return ""
	}
	return runtimepaths.MacAppRuntimeBaseDir(home, defaultRuntimeBundleID, defaultRuntimeAppName)
}

func shouldUseKnowledgeRuntimeBase(absAppDir string) bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	if knowledgeMacRuntimeBaseDir() == "" {
		return false
	}
	if cwd, err := os.Getwd(); err == nil {
		if absCwd, err := filepath.Abs(cwd); err == nil && absCwd == absAppDir {
			return true
		}
	}
	if exe := knowledgeExecutablePath(); exe != "" && filepath.Dir(exe) == absAppDir {
		return true
	}
	lower := strings.ToLower(filepath.ToSlash(absAppDir))
	return strings.Contains(lower, ".app/contents/")
}

func resolveRuntimeAppDir(appDir string) (string, error) {
	absAppDir, err := normalizeAppDir(appDir)
	if err != nil {
		return "", err
	}
	if shouldUseKnowledgeRuntimeBase(absAppDir) {
		runtimeBase := knowledgeMacRuntimeBaseDir()
		if runtimeBase == "" {
			return "", fmt.Errorf("resolve mac runtime dir")
		}
		return runtimeBase, nil
	}
	return absAppDir, nil
}

func KnowledgeDir(appDir string) (string, error) {
	absAppDir, err := resolveRuntimeAppDir(appDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(absAppDir, defaultKnowledgeDirName), nil
}

func KnowledgeDirForAgent(appDir string, agentID string) (string, error) {
	root, err := KnowledgeDir(appDir)
	if err != nil {
		return "", err
	}
	agentID = normalizeKnowledgeAgentID(agentID)
	if agentID == "" {
		return root, nil
	}
	return filepath.Join(root, agentID), nil
}

func EnsureKnowledgeDir(appDir string) (string, error) {
	dir, err := KnowledgeDir(appDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create knowledge dir: %w", err)
	}
	return dir, nil
}

func EnsureKnowledgeDirForAgent(appDir string, agentID string) (string, error) {
	dir, err := KnowledgeDirForAgent(appDir, agentID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create agent knowledge dir: %w", err)
	}
	return dir, nil
}

func DBPath(appDir string) (string, error) {
	absAppDir, err := resolveRuntimeAppDir(appDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(absAppDir, defaultSQLiteFileName), nil
}

func OpenSharedDB(appDir string) (*sql.DB, error) {
	dbPath, err := DBPath(appDir)
	if err != nil {
		return nil, err
	}

	sharedDB.mu.Lock()
	defer sharedDB.mu.Unlock()

	if sharedDB.db != nil && sharedDB.path == dbPath {
		if err := EnsureSchema(sharedDB.db); err != nil {
			return nil, err
		}
		return sharedDB.db, nil
	}

	if sharedDB.db != nil {
		_ = sharedDB.db.Close()
		sharedDB.db = nil
		sharedDB.path = ""
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(0)

	if err := EnsureSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	sharedDB.db = db
	sharedDB.path = dbPath
	return db, nil
}

func OpenExistingDB(appDir string) (*sql.DB, error) {
	dbPath, err := DBPath(appDir)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(dbPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat sqlite: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("sqlite path is a directory: %s", dbPath)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open existing sqlite: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(0)
	return db, nil
}

func CloseSharedDB() error {
	sharedDB.mu.Lock()
	defer sharedDB.mu.Unlock()

	if sharedDB.db == nil {
		return nil
	}
	err := sharedDB.db.Close()
	sharedDB.db = nil
	sharedDB.path = ""
	return err
}

func EnsureSchema(db *sql.DB) error {
	if db == nil {
		return errors.New("nil sqlite db")
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		return fmt.Errorf("set sqlite wal: %w", err)
	}
	if err := ensureKnowledgeRuntimeSchema(db); err != nil {
		return err
	}
	if _, err := db.Exec(`
		INSERT OR IGNORE INTO knowledge_runtime(agent_id, last_update, knowledge_commit) VALUES ('', 0, 0)
	`); err != nil {
		return fmt.Errorf("init default knowledge_runtime row: %w", err)
	}
	return nil
}

func ensureKnowledgeRuntimeSchema(db *sql.DB) error {
	hasTable, err := tableExists(db, "knowledge_runtime")
	if err != nil {
		return err
	}
	if !hasTable {
		if _, err := db.Exec(`
			CREATE TABLE knowledge_runtime (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				agent_id TEXT NOT NULL UNIQUE,
				last_update INTEGER NOT NULL DEFAULT 0,
				knowledge_commit INTEGER NOT NULL DEFAULT 0
			)
		`); err != nil {
			return fmt.Errorf("create knowledge_runtime: %w", err)
		}
		return nil
	}
	hasAgentID, err := tableHasColumn(db, "knowledge_runtime", "agent_id")
	if err != nil {
		return err
	}
	hasKnowledgeCommit, err := tableHasColumn(db, "knowledge_runtime", "knowledge_commit")
	if err != nil {
		return err
	}
	if hasAgentID && hasKnowledgeCommit {
		return nil
	}
	if hasAgentID && !hasKnowledgeCommit {
		if _, err := db.Exec(`ALTER TABLE knowledge_runtime ADD COLUMN knowledge_commit INTEGER NOT NULL DEFAULT 0`); err != nil {
			return fmt.Errorf("add knowledge_commit to knowledge_runtime: %w", err)
		}
		return nil
	}
	if _, err := db.Exec(`
		CREATE TABLE knowledge_runtime_migrated (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id TEXT NOT NULL UNIQUE,
			last_update INTEGER NOT NULL DEFAULT 0,
			knowledge_commit INTEGER NOT NULL DEFAULT 0
		)
	`); err != nil {
		return fmt.Errorf("create migrated knowledge_runtime: %w", err)
	}
	if _, err := db.Exec(`
		INSERT INTO knowledge_runtime_migrated(agent_id, last_update, knowledge_commit)
		SELECT '', COALESCE(last_update, 0), 0 FROM knowledge_runtime
	`); err != nil {
		return fmt.Errorf("migrate knowledge_runtime data: %w", err)
	}
	if _, err := db.Exec(`DROP TABLE knowledge_runtime`); err != nil {
		return fmt.Errorf("drop legacy knowledge_runtime: %w", err)
	}
	if _, err := db.Exec(`ALTER TABLE knowledge_runtime_migrated RENAME TO knowledge_runtime`); err != nil {
		return fmt.Errorf("rename migrated knowledge_runtime: %w", err)
	}
	return nil
}

func tableExists(db *sql.DB, table string) (bool, error) {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count); err != nil {
		return false, fmt.Errorf("query sqlite_master for %s: %w", table, err)
	}
	return count > 0, nil
}

func tableHasColumn(db *sql.DB, table string, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return false, fmt.Errorf("query table_info for %s: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			colType    string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultVal, &pk); err != nil {
			return false, fmt.Errorf("scan table_info for %s: %w", table, err)
		}
		if strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(column)) {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate table_info for %s: %w", table, err)
	}
	return false, nil
}

func ensureKnowledgeRuntimeRow(db *sql.DB, agentID string) error {
	if err := EnsureSchema(db); err != nil {
		return err
	}
	if _, err := db.Exec(`
		INSERT OR IGNORE INTO knowledge_runtime(agent_id, last_update, knowledge_commit) VALUES (?, 0, 0)
	`, normalizeKnowledgeAgentID(agentID)); err != nil {
		return fmt.Errorf("init knowledge_runtime row: %w", err)
	}
	return nil
}

func GetLastUpdate(db *sql.DB) (int64, error) {
	return GetLastUpdateForAgent(db, "")
}

func GetLastUpdateForAgent(db *sql.DB, agentID string) (int64, error) {
	if err := EnsureSchema(db); err != nil {
		return 0, err
	}
	if err := ensureKnowledgeRuntimeRow(db, agentID); err != nil {
		return 0, err
	}
	var lastUpdate int64
	if err := db.QueryRow(`SELECT last_update FROM knowledge_runtime WHERE agent_id = ?`, normalizeKnowledgeAgentID(agentID)).Scan(&lastUpdate); err != nil {
		return 0, fmt.Errorf("query last_update: %w", err)
	}
	return lastUpdate, nil
}

func SetLastUpdate(db *sql.DB, ts int64) error {
	return SetLastUpdateForAgent(db, "", ts)
}

func SetLastUpdateForAgent(db *sql.DB, agentID string, ts int64) error {
	if ts < 0 {
		return fmt.Errorf("invalid last update timestamp: %d", ts)
	}
	if err := EnsureSchema(db); err != nil {
		return err
	}
	if _, err := db.Exec(`
		INSERT INTO knowledge_runtime(agent_id, last_update, knowledge_commit) VALUES (?, ?, 0)
		ON CONFLICT(agent_id) DO UPDATE SET last_update = excluded.last_update
	`, normalizeKnowledgeAgentID(agentID), ts); err != nil {
		return fmt.Errorf("update knowledge last_update: %w", err)
	}
	return nil
}

func GetKnowledgeCommit(db *sql.DB) (bool, error) {
	return GetKnowledgeCommitForAgent(db, "")
}

func GetKnowledgeCommitForAgent(db *sql.DB, agentID string) (bool, error) {
	if err := EnsureSchema(db); err != nil {
		return false, err
	}
	if err := ensureKnowledgeRuntimeRow(db, agentID); err != nil {
		return false, err
	}
	var raw int64
	if err := db.QueryRow(`SELECT knowledge_commit FROM knowledge_runtime WHERE agent_id = ?`, normalizeKnowledgeAgentID(agentID)).Scan(&raw); err != nil {
		return false, fmt.Errorf("query knowledge_commit: %w", err)
	}
	return raw != 0, nil
}

func SetKnowledgeCommit(db *sql.DB, value bool) error {
	return SetKnowledgeCommitForAgent(db, "", value)
}

func SetKnowledgeCommitForAgent(db *sql.DB, agentID string, value bool) error {
	if err := EnsureSchema(db); err != nil {
		return err
	}
	commitValue := int64(0)
	if value {
		commitValue = 1
	}
	if _, err := db.Exec(`
		INSERT INTO knowledge_runtime(agent_id, last_update, knowledge_commit) VALUES (?, 0, ?)
		ON CONFLICT(agent_id) DO UPDATE SET knowledge_commit = excluded.knowledge_commit
	`, normalizeKnowledgeAgentID(agentID), commitValue); err != nil {
		return fmt.Errorf("update knowledge knowledge_commit: %w", err)
	}
	return nil
}

func EnsureRuntime(appDir string) (*State, error) {
	return EnsureRuntimeForAgent(appDir, "")
}

func EnsureRuntimeForAgent(appDir string, agentID string) (*State, error) {
	knowledgeDir, err := EnsureKnowledgeDirForAgent(appDir, agentID)
	if err != nil {
		return nil, err
	}
	db, err := OpenSharedDB(appDir)
	if err != nil {
		return nil, err
	}
	lastUpdate, err := GetLastUpdateForAgent(db, agentID)
	if err != nil {
		return nil, err
	}
	knowledgeCommit, err := GetKnowledgeCommitForAgent(db, agentID)
	if err != nil {
		return nil, err
	}
	return &State{
		Path:            knowledgeDir,
		LastUpdate:      lastUpdate,
		KnowledgeCommit: knowledgeCommit,
	}, nil
}

func LookupRuntime(appDir string) (*State, error) {
	return LookupRuntimeForAgent(appDir, "")
}

func LookupRuntimeForAgent(appDir string, agentID string) (*State, error) {
	knowledgeDir, err := KnowledgeDirForAgent(appDir, agentID)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(knowledgeDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat knowledge dir: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("knowledge path is not a directory: %s", knowledgeDir)
	}

	db, err := OpenExistingDB(appDir)
	if err != nil {
		return nil, err
	}
	if db == nil {
		return &State{
			Path:            knowledgeDir,
			LastUpdate:      0,
			KnowledgeCommit: false,
		}, nil
	}
	defer db.Close()

	lastUpdate, err := GetLastUpdateForAgent(db, agentID)
	if err != nil {
		return &State{
			Path:            knowledgeDir,
			LastUpdate:      0,
			KnowledgeCommit: false,
		}, nil
	}
	knowledgeCommit, err := GetKnowledgeCommitForAgent(db, agentID)
	if err != nil {
		return &State{
			Path:            knowledgeDir,
			LastUpdate:      lastUpdate,
			KnowledgeCommit: false,
		}, nil
	}
	return &State{
		Path:            knowledgeDir,
		LastUpdate:      lastUpdate,
		KnowledgeCommit: knowledgeCommit,
	}, nil
}

func Metadata(appDir string) (map[string]any, error) {
	return MetadataForAgent(appDir, "")
}

func MetadataForAgent(appDir string, agentID string) (map[string]any, error) {
	state, err := EnsureRuntimeForAgent(appDir, agentID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"knowledge": state,
	}, nil
}

func MetadataIfExists(appDir string) (map[string]any, error) {
	return MetadataIfExistsForAgent(appDir, "")
}

func MetadataIfExistsForAgent(appDir string, agentID string) (map[string]any, error) {
	state, err := LookupRuntimeForAgent(appDir, agentID)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, nil
	}
	return map[string]any{
		"knowledge": state,
	}, nil
}

func MergeMetadata(base map[string]any, appDir string) (map[string]any, error) {
	extra, err := MetadataForAgent(appDir, metadataAgentID(base))
	if err != nil {
		return nil, err
	}
	merged := make(map[string]any, len(base)+len(extra))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range extra {
		merged[k] = v
	}
	return merged, nil
}

func MergeMetadataIfExists(base map[string]any, appDir string) (map[string]any, error) {
	extra, err := MetadataIfExistsForAgent(appDir, metadataAgentID(base))
	if err != nil {
		return nil, err
	}
	merged := make(map[string]any, len(base)+1)
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range extra {
		merged[k] = v
	}
	return merged, nil
}

func MarshalState(appDir string) ([]byte, error) {
	return MarshalStateForAgent(appDir, "")
}

func MarshalStateForAgent(appDir string, agentID string) ([]byte, error) {
	state, err := EnsureRuntimeForAgent(appDir, agentID)
	if err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal knowledge state: %w", err)
	}
	return data, nil
}

func metadataAgentID(base map[string]any) string {
	if len(base) == 0 {
		return ""
	}
	raw, ok := base["agentId"]
	if !ok || raw == nil {
		return ""
	}
	if agentID, ok := raw.(string); ok {
		return normalizeKnowledgeAgentID(agentID)
	}
	return ""
}
