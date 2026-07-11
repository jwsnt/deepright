package skillscore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

const warningTableDDL = `
CREATE TABLE IF NOT EXISTS skills_warning (
	path TEXT PRIMARY KEY,
	reason TEXT NOT NULL,
	time INTEGER NOT NULL
);`

type WarningStore struct {
	db *sql.DB
}

var (
	storeMu    sync.Mutex
	storeCache = map[string]*WarningStore{}
)

func OpenWarningStore(dbPath string) (*WarningStore, error) {
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, fmt.Errorf("解析数据库路径失败: %w", err)
	}

	storeMu.Lock()
	defer storeMu.Unlock()

	if store := storeCache[absPath]; store != nil {
		return store, nil
	}

	db, err := sql.Open("sqlite", absPath)
	if err != nil {
		return nil, fmt.Errorf("打开 sqlite 失败: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(0)

	if _, err := db.Exec(warningTableDDL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("初始化 skills_warning 表失败: %w", err)
	}

	store := &WarningStore{db: db}
	storeCache[absPath] = store
	return store, nil
}

func (s *WarningStore) Sync(warnings []SkillWarning) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(`DELETE FROM skills_warning`); err != nil {
		return fmt.Errorf("清空旧告警失败: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT INTO skills_warning(path, reason, time) VALUES(?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("准备写入告警失败: %w", err)
	}
	defer stmt.Close()

	for _, warning := range warnings {
		ts := warning.Time
		if ts <= 0 {
			ts = time.Now().Unix()
		}
		if _, err = stmt.Exec(warning.Path, warning.Reason, ts); err != nil {
			return fmt.Errorf("写入告警失败: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}

func (s *WarningStore) List() ([]SkillWarning, error) {
	rows, err := s.db.Query(`SELECT path, reason, time FROM skills_warning ORDER BY path`)
	if err != nil {
		return nil, fmt.Errorf("查询告警失败: %w", err)
	}
	defer rows.Close()

	var warnings []SkillWarning
	for rows.Next() {
		var warning SkillWarning
		if err := rows.Scan(&warning.Path, &warning.Reason, &warning.Time); err != nil {
			return nil, fmt.Errorf("读取告警失败: %w", err)
		}
		warnings = append(warnings, warning)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历告警失败: %w", err)
	}
	return warnings, nil
}

func (s *WarningStore) ListJSON() ([]byte, error) {
	warnings, err := s.List()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(warnings, "", "  ")
}

func ScanAndSyncWarnings(root, dbPath string) ([]SkillWarning, error) {
	_, warnings, err := ScanWithWarnings(root)
	if err != nil {
		return nil, err
	}
	store, err := OpenWarningStore(dbPath)
	if err != nil {
		return nil, err
	}
	if err := store.Sync(warnings); err != nil {
		return nil, err
	}
	return warnings, nil
}
