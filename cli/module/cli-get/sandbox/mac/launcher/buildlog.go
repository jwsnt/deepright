package launcher

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

type pooledDB struct {
	mu   sync.Mutex
	path string
	db   *sql.DB
}

var sharedBuildLogDB pooledDB

func appendBuildLog(baseDir, action, status, appPath, identity, detail string) {
	db, err := getBuildLogDB(baseDir)
	if err != nil {
		return
	}
	_, _ = db.Exec(
		`INSERT INTO mac_build_log (action, status, app_path, identity, detail, created_at) VALUES (?,?,?,?,?,?)`,
		action,
		status,
		appPath,
		identity,
		detail,
		time.Now().Format("2006-01-02T15:04:05.000"),
	)
}

func getBuildLogDB(baseDir string) (*sql.DB, error) {
	if baseDir == "" {
		baseDir = "."
	}
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(absBase, "data")

	sharedBuildLogDB.mu.Lock()
	defer sharedBuildLogDB.mu.Unlock()

	if sharedBuildLogDB.db != nil && sharedBuildLogDB.path == path {
		return sharedBuildLogDB.db, nil
	}
	if sharedBuildLogDB.db != nil {
		_ = sharedBuildLogDB.db.Close()
		sharedBuildLogDB.db = nil
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(0)
	if err := ensureBuildLogSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	sharedBuildLogDB.path = path
	sharedBuildLogDB.db = db
	return db, nil
}

func ensureBuildLogSchema(db *sql.DB) error {
	if db == nil {
		return nil
	}
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS mac_build_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			action TEXT NOT NULL,
			status TEXT NOT NULL,
			app_path TEXT NOT NULL DEFAULT '',
			identity TEXT NOT NULL DEFAULT '',
			detail TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_mac_build_log_action_time
			ON mac_build_log(action, created_at);
	`)
	return err
}

func resetBuildLogDBForTest() {
	sharedBuildLogDB.mu.Lock()
	defer sharedBuildLogDB.mu.Unlock()
	if sharedBuildLogDB.db != nil {
		_ = sharedBuildLogDB.db.Close()
	}
	sharedBuildLogDB.db = nil
	sharedBuildLogDB.path = ""
}

func buildLogCount(baseDir string) (int, error) {
	db, err := getBuildLogDB(baseDir)
	if err != nil {
		return 0, err
	}
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM mac_build_log`).Scan(&count)
	return count, err
}

func buildLogLatest(baseDir string) (string, string, error) {
	db, err := getBuildLogDB(baseDir)
	if err != nil {
		return "", "", err
	}
	var action string
	var status string
	err = db.QueryRow(`SELECT action, status FROM mac_build_log ORDER BY id DESC LIMIT 1`).Scan(&action, &status)
	return action, status, err
}

func ensureBaseDir(baseDir string) error {
	if baseDir == "" {
		baseDir = "."
	}
	return os.MkdirAll(baseDir, 0o755)
}
