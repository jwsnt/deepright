package connectsvc

import (
	"connect/sharedutil"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

const (
	RequestStatusPending = iota
	RequestStatusStarted
	RequestStatusCompleted
	RequestStatusExpired
	RequestStatusReplied
)

type Options struct {
	DBPath   string
	AgentDir string
	CacheTTL time.Duration
}

type Service struct {
	db       *sql.DB
	dbPath   string
	agentDir string
	cacheTTL time.Duration
	closeFn  func() error

	cacheMu sync.Mutex
	cache   map[string]cacheEntry
}

type cacheEntry struct {
	payload []byte
	expires time.Time
}

type sharedDBEntry struct {
	db   *sql.DB
	refs int
}

var (
	sharedDBMu      sync.Mutex
	sharedDBEntries = map[string]*sharedDBEntry{}
)

type Meta struct {
	ID            int    `json:"id"`
	Key           string `json:"key,omitempty"`
	Name          string `json:"name"`
	Meta          string `json:"meta"`
	Stream        bool   `json:"stream"`
	Callback      string `json:"callback"`
	AgentID       string `json:"agentId"`
	ChatID        string `json:"chatId"`
	Model         string `json:"model"`
	Thinking      bool   `json:"thinking"`
	Verify        bool   `json:"verify"`
	RouterDisable bool   `json:"router_disable"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
	DeletedAt     string `json:"deletedAt,omitempty"`
}

func (m *Meta) UnmarshalJSON(data []byte) error {
	type metaAlias struct {
		ID            int             `json:"id"`
		Key           string          `json:"key,omitempty"`
		Name          string          `json:"name"`
		Meta          json.RawMessage `json:"meta"`
		Stream        bool            `json:"stream"`
		Callback      string          `json:"callback"`
		AgentID       string          `json:"agentId"`
		ChatID        string          `json:"chatId"`
		Model         string          `json:"model"`
		Thinking      bool            `json:"thinking"`
		Verify        *bool           `json:"verify"`
		RouterDisable *bool           `json:"router_disable"`
		CreatedAt     string          `json:"createdAt"`
		UpdatedAt     string          `json:"updatedAt"`
		DeletedAt     string          `json:"deletedAt,omitempty"`
	}

	var raw metaAlias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	m.ID = raw.ID
	m.Key = strings.TrimSpace(raw.Key)
	m.Name = raw.Name
	m.Stream = raw.Stream
	m.Callback = raw.Callback
	m.AgentID = raw.AgentID
	m.ChatID = raw.ChatID
	m.Model = raw.Model
	m.Thinking = raw.Thinking
	if raw.Verify != nil {
		m.Verify = *raw.Verify
	}
	if raw.RouterDisable != nil {
		m.RouterDisable = *raw.RouterDisable
	} else {
		m.RouterDisable = true
	}
	m.CreatedAt = raw.CreatedAt
	m.UpdatedAt = raw.UpdatedAt
	m.DeletedAt = raw.DeletedAt

	if len(raw.Meta) == 0 || string(raw.Meta) == "null" {
		m.fillRuntimeKey()
		m.Meta = ""
		return nil
	}

	var metaText string
	if err := json.Unmarshal(raw.Meta, &metaText); err == nil {
		m.Meta = metaText
		m.fillRuntimeKey()
		return nil
	}

	m.Meta = string(raw.Meta)
	m.fillRuntimeKey()
	return nil
}

type MetaConfig struct {
	Key           string         `json:"key"`
	Name          string         `json:"name"`
	Meta          map[string]any `json:"meta"`
	Stream        bool           `json:"stream"`
	Callback      string         `json:"callback"`
	AgentID       string         `json:"agentId"`
	ChatID        string         `json:"chatId"`
	Model         string         `json:"model"`
	Thinking      bool           `json:"thinking"`
	Verify        bool           `json:"verify"`
	RouterDisable bool           `json:"router_disable"`
	CreatedAt     string         `json:"createdAt"`
	UpdatedAt     string         `json:"updatedAt"`
}

type MetaInput struct {
	Key           string
	Name          string
	Meta          string
	Stream        bool
	Callback      string
	AgentID       string
	ChatID        string
	Model         string
	Thinking      bool
	Verify        bool
	RouterDisable bool
}

type MetaUpdate struct {
	Meta          *string
	Stream        *bool
	Callback      *string
	AgentID       *string
	ChatID        *string
	Model         *string
	Thinking      *bool
	Verify        *bool
	RouterDisable *bool
}

type Request struct {
	ID             int    `json:"id"`
	Key            string `json:"key"`
	Name           string `json:"name,omitempty"`
	ExternalID     string `json:"externalId,omitempty"`
	Content        string `json:"content"`
	Request        string `json:"request,omitempty"`
	Artifacts      string `json:"artifacts,omitempty"`
	Original       string `json:"original,omitempty"`
	RawRequest     string `json:"rawRequest,omitempty"`
	ResponseSchema string `json:"responseSchema,omitempty"`
	Status         int    `json:"status"`
	CreatedAt      string `json:"createdAt"`
}

type RequestInput struct {
	Key             string
	Name            string
	ExternalID      string
	Content         string
	Request         string
	Artifacts       string
	Original        string
	RawRequest      string
	ResponseSchema  string
	MessageSnapshot string
	Status          *int
	CreatedAt       string
}

type RequestFilter struct {
	Key      string
	Name     string
	AfterID  int
	BeforeID int
	Limit    int
	Status   *int
}

type RequestStatusUpdate struct {
	ID     int
	Key    string
	Name   string
	From   *int
	To     int
	Strict bool
}

type Response struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	RequestID int    `json:"requestId"`
	Response  string `json:"response"`
	Artifacts string `json:"artifacts,omitempty"`
	CreatedAt string `json:"createdAt"`
}

type ResponseInput struct {
	Key       string
	Name      string
	RequestID int
	Response  string
	Artifacts string
}

type ResponseFilter struct {
	Key       string
	Name      string
	RequestID int
	AfterID   int
	Limit     int
}

func NewService(opts Options) (*Service, error) {
	dbPath := strings.TrimSpace(opts.DBPath)
	if dbPath == "" {
		dbPath = DefaultDBPath()
	}
	agentDir, err := ResolveAgentDir(opts.AgentDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	absDBPath, err := filepath.Abs(dbPath)
	if err == nil {
		dbPath = absDBPath
	}
	db, closeFn, err := acquireSharedDB(dbPath)
	if err != nil {
		return nil, err
	}

	svc := &Service{
		db:       db,
		dbPath:   dbPath,
		agentDir: agentDir,
		cacheTTL: opts.CacheTTL,
		closeFn:  closeFn,
		cache:    make(map[string]cacheEntry),
	}
	if svc.cacheTTL <= 0 {
		svc.cacheTTL = 10 * time.Second
	}
	if err := svc.initDB(); err != nil {
		_ = closeFn()
		return nil, err
	}
	return svc, nil
}

func (s *Service) Close() error {
	if s == nil || s.closeFn == nil {
		return nil
	}
	closeFn := s.closeFn
	s.closeFn = nil
	return closeFn()
}

func acquireSharedDB(dbPath string) (*sql.DB, func() error, error) {
	sharedDBMu.Lock()
	defer sharedDBMu.Unlock()

	if entry, ok := sharedDBEntries[dbPath]; ok && entry != nil && entry.db != nil {
		entry.refs++
		return entry.db, releaseSharedDB(dbPath), nil
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(30 * time.Minute)

	sharedDBEntries[dbPath] = &sharedDBEntry{
		db:   db,
		refs: 1,
	}
	return db, releaseSharedDB(dbPath), nil
}

func releaseSharedDB(dbPath string) func() error {
	return func() error {
		sharedDBMu.Lock()
		entry, ok := sharedDBEntries[dbPath]
		if !ok || entry == nil || entry.db == nil {
			sharedDBMu.Unlock()
			return nil
		}
		entry.refs--
		if entry.refs > 0 {
			sharedDBMu.Unlock()
			return nil
		}
		delete(sharedDBEntries, dbPath)
		db := entry.db
		sharedDBMu.Unlock()
		return db.Close()
	}
}

func DefaultDBPath() string {
	if env := strings.TrimSpace(os.Getenv("CONNECT_DB")); env != "" {
		return absOrOriginal(env)
	}
	if _, err := os.Stat("../cron"); err == nil {
		return absOrOriginal(filepath.Join("..", "cron", "data"))
	}
	return absOrOriginal("data")
}

func ResolveAgentDir(raw string) (string, error) {
	agentDir := strings.TrimSpace(raw)
	if agentDir == "" {
		agentDir = strings.TrimSpace(os.Getenv("AGENT_DIR"))
	}
	if agentDir == "" {
		if info, err := os.Stat("agent"); err == nil && info.IsDir() {
			agentDir = "agent"
		}
	}
	if agentDir == "" {
		return "", fmt.Errorf("agent-dir is required")
	}
	return filepath.Abs(agentDir)
}

func (s *Service) dbPathValue() string {
	if s == nil {
		return ""
	}
	return s.dbPath
}

func (s *Service) CreateMeta(input MetaInput) (*Meta, error) {
	var err error
	input, err = s.normalizeMetaInput(input)
	if err != nil {
		return nil, err
	}
	if err := s.validateMetaInput(input); err != nil {
		return nil, err
	}
	now := timestamp()
	res, err := s.db.Exec(`
		INSERT INTO connect_meta (name, meta, stream, callback, agent_id, chat_id, model, thinking, verify, router_disable, created_at, updated_at, deleted_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
	`,
		strings.TrimSpace(input.Key),
		strings.TrimSpace(input.Meta),
		boolToInt(input.Stream),
		strings.TrimSpace(input.Callback),
		strings.TrimSpace(input.AgentID),
		strings.TrimSpace(input.ChatID),
		strings.TrimSpace(input.Model),
		boolToInt(input.Thinking),
		boolToInt(input.Verify),
		boolToInt(input.RouterDisable),
		now,
		now,
		"",
	)
	if err != nil {
		if isUniqueErr(err) {
			return nil, fmt.Errorf("connect meta already exists: %s", strings.TrimSpace(input.Key))
		}
		return nil, err
	}
	id, _ := res.LastInsertId()
	s.invalidateCache()
	return s.getMetaByID(int(id))
}

func (s *Service) UpdateMeta(key string, patch MetaUpdate) (*Meta, error) {
	meta, err := s.mustActiveMeta(key)
	if err != nil {
		return nil, err
	}
	next := MetaInput{
		Key:           meta.Key,
		Name:          meta.Name,
		Meta:          meta.Meta,
		Stream:        meta.Stream,
		Callback:      meta.Callback,
		AgentID:       meta.AgentID,
		ChatID:        meta.ChatID,
		Model:         meta.Model,
		Thinking:      meta.Thinking,
		Verify:        meta.Verify,
		RouterDisable: meta.RouterDisable,
	}
	if patch.Meta != nil {
		next.Meta = strings.TrimSpace(*patch.Meta)
	}
	if patch.Stream != nil {
		next.Stream = *patch.Stream
	}
	if patch.Callback != nil {
		next.Callback = strings.TrimSpace(*patch.Callback)
	}
	if patch.AgentID != nil {
		next.AgentID = strings.TrimSpace(*patch.AgentID)
	}
	if patch.ChatID != nil {
		next.ChatID = strings.TrimSpace(*patch.ChatID)
	}
	if patch.Model != nil {
		next.Model = strings.TrimSpace(*patch.Model)
	}
	if patch.Thinking != nil {
		next.Thinking = *patch.Thinking
	}
	if patch.Verify != nil {
		next.Verify = *patch.Verify
	}
	if patch.RouterDisable != nil {
		next.RouterDisable = *patch.RouterDisable
	}
	next, err = s.normalizeMetaInput(next)
	if err != nil {
		return nil, err
	}
	if err := s.validateMetaInput(next); err != nil {
		return nil, err
	}
	_, err = s.db.Exec(`
		UPDATE connect_meta
		SET meta = ?, stream = ?, callback = ?, agent_id = ?, chat_id = ?, model = ?, thinking = ?, verify = ?, router_disable = ?, updated_at = ?
		WHERE id = ? AND deleted_at = ''
	`,
		next.Meta,
		boolToInt(next.Stream),
		next.Callback,
		next.AgentID,
		next.ChatID,
		next.Model,
		boolToInt(next.Thinking),
		boolToInt(next.Verify),
		boolToInt(next.RouterDisable),
		timestamp(),
		meta.ID,
	)
	if err != nil {
		return nil, err
	}
	s.invalidateCache()
	return s.getMetaByID(meta.ID)
}

func (s *Service) DeleteMeta(key string) (*Meta, error) {
	meta, err := s.mustActiveMeta(key)
	if err != nil {
		return nil, err
	}
	now := timestamp()
	_, err = s.db.Exec(`UPDATE connect_meta SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at = ''`, now, now, meta.ID)
	if err != nil {
		return nil, err
	}
	s.invalidateCache()
	return s.getMetaByID(meta.ID)
}

func (s *Service) GetMeta(key string, includeDeleted bool) (*Meta, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}
	cacheKey := fmt.Sprintf("meta:get:%s:%t", key, includeDeleted)
	var meta Meta
	err := s.cachedJSON(cacheKey, &meta, func() (any, error) {
		var row *sql.Row
		if includeDeleted {
			row = s.db.QueryRow(fmt.Sprintf(`
				SELECT id, name, meta, stream, callback, agent_id, chat_id, model, thinking, %s, %s, created_at, updated_at, deleted_at
				FROM connect_meta WHERE name = ? ORDER BY id DESC LIMIT 1
			`, connectMetaVerifySelect(s.db), connectMetaRouterDisableSelect(s.db)), key)
		} else {
			row = s.db.QueryRow(fmt.Sprintf(`
				SELECT id, name, meta, stream, callback, agent_id, chat_id, model, thinking, %s, %s, created_at, updated_at, deleted_at
				FROM connect_meta WHERE name = ? AND deleted_at = '' ORDER BY id DESC LIMIT 1
			`, connectMetaVerifySelect(s.db), connectMetaRouterDisableSelect(s.db)), key)
		}
		item, err := scanMetaRow(row)
		if err != nil {
			return nil, err
		}
		return item, nil
	})
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

func (s *Service) GetMetaConfig(key string, includeDeleted bool) (*MetaConfig, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	configs, err := s.ListMetaConfig(includeDeleted)
	if err != nil {
		return nil, err
	}
	for _, item := range configs {
		if strings.EqualFold(strings.TrimSpace(item.Key), key) {
			cfg := item
			return &cfg, nil
		}
	}
	return nil, fmt.Errorf("connect meta not found: %s", key)
}

func (s *Service) ListMeta(includeDeleted bool) ([]Meta, error) {
	key := fmt.Sprintf("meta:list:%t", includeDeleted)
	var metas []Meta
	err := s.cachedJSON(key, &metas, func() (any, error) {
		query := fmt.Sprintf(`
			SELECT id, name, meta, stream, callback, agent_id, chat_id, model, thinking, %s, %s, created_at, updated_at, deleted_at
			FROM connect_meta
		`, connectMetaVerifySelect(s.db), connectMetaRouterDisableSelect(s.db))
		if !includeDeleted {
			query += ` WHERE deleted_at = ''`
		}
		query += ` ORDER BY id`
		rows, err := s.db.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		items := make([]Meta, 0)
		for rows.Next() {
			item, err := scanMetaRows(rows)
			if err != nil {
				return nil, err
			}
			items = append(items, *item)
		}
		return items, rows.Err()
	})
	return metas, err
}

func (s *Service) ListMetaConfig(includeDeleted bool) ([]MetaConfig, error) {
	key := fmt.Sprintf("meta:config:list:%t", includeDeleted)
	var items []MetaConfig
	err := s.cachedJSON(key, &items, func() (any, error) {
		metas, err := s.ListMeta(includeDeleted)
		if err != nil {
			return nil, err
		}
		out := make([]MetaConfig, 0, len(metas))
		for _, item := range metas {
			cfg, err := metaToConfig(item)
			if err != nil {
				return nil, err
			}
			out = append(out, cfg)
		}
		return out, nil
	})
	return items, err
}

func (s *Service) AddRequest(input RequestInput) (*Request, error) {
	key := strings.TrimSpace(input.Key)
	if key == "" {
		return nil, errors.New("key is required")
	}
	meta, err := s.mustActiveMeta(key)
	if err != nil {
		return nil, err
	}
	if err := s.validateMetaRuntime(meta); err != nil {
		return nil, err
	}
	content := strings.TrimSpace(input.Content)
	if content == "" {
		content = strings.TrimSpace(input.Request)
	}
	if content == "" {
		return nil, errors.New("content is required")
	}
	externalID := strings.TrimSpace(input.ExternalID)
	artifacts, err := normalizeArtifacts(input.Artifacts)
	if err != nil {
		return nil, err
	}
	original := strings.TrimSpace(input.Original)
	if original == "" {
		original = strings.TrimSpace(input.RawRequest)
	}
	responseSchema := strings.TrimSpace(input.ResponseSchema)
	status := RequestStatusPending
	if input.Status != nil {
		status = *input.Status
	}
	if err := validateRequestStatus(status); err != nil {
		return nil, err
	}
	createdAt, err := normalizeRequestCreatedAt(input.CreatedAt)
	if err != nil {
		return nil, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`
		INSERT INTO connect_request (name, external_id, request, artifacts, raw_request, response_schema, status, created_at)
		VALUES (?,?,?,?,?,?,?,?)
	`, meta.runtimeKey(), externalID, content, artifacts, original, responseSchema, status, createdAt)
	if err != nil {
		if isUniqueErr(err) && externalID != "" {
			return nil, fmt.Errorf("connect request already exists: %s/%s", meta.runtimeKey(), externalID)
		}
		return nil, err
	}
	if err := syncMessageSnapshot(tx, input.MessageSnapshot); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	s.invalidateCache()
	item, err := s.getRequestByID(int(id))
	if err != nil {
		return nil, err
	}
	if item != nil {
		item.Key = meta.runtimeKey()
	}
	return item, nil
}

func (s *Service) ListRequests(filter RequestFilter) ([]Request, error) {
	if strings.TrimSpace(filter.Name) != "" {
		return nil, errors.New("name filter is not supported; use key")
	}
	if filter.Limit <= 0 {
		filter.Limit = 100
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000
	}
	if filter.Status != nil {
		if err := validateRequestStatus(*filter.Status); err != nil {
			return nil, err
		}
	}
	statusKey := -1
	if filter.Status != nil {
		statusKey = *filter.Status
	}
	key := fmt.Sprintf("request:list:%s:%d:%d:%d:%d", strings.TrimSpace(filter.Key), filter.AfterID, filter.BeforeID, filter.Limit, statusKey)
	var items []Request
	err := s.cachedJSON(key, &items, func() (any, error) {
		parts := []string{`SELECT id, name, external_id, request, artifacts, raw_request, response_schema, status, created_at FROM connect_request WHERE 1 = 1`}
		args := make([]any, 0)
		if key := strings.TrimSpace(filter.Key); key != "" {
			parts = append(parts, `AND name = ?`)
			args = append(args, key)
		}
		if filter.Status != nil {
			parts = append(parts, `AND status = ?`)
			args = append(args, *filter.Status)
		}
		if filter.AfterID > 0 {
			parts = append(parts, `AND id > ?`)
			args = append(args, filter.AfterID)
		}
		if filter.BeforeID > 0 {
			parts = append(parts, `AND id < ?`)
			args = append(args, filter.BeforeID)
		}
		parts = append(parts, `ORDER BY id DESC LIMIT ?`)
		args = append(args, filter.Limit)
		rows, err := s.db.Query(strings.Join(parts, " "), args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := make([]Request, 0)
		for rows.Next() {
			var item Request
			if err := rows.Scan(&item.ID, &item.Name, &item.ExternalID, &item.Request, &item.Artifacts, &item.RawRequest, &item.ResponseSchema, &item.Status, &item.CreatedAt); err != nil {
				return nil, err
			}
			item.fillDerivedFields()
			out = append(out, item)
		}
		return out, rows.Err()
	})
	return items, err
}

func (s *Service) UpdateRequestStatus(input RequestStatusUpdate) (*Request, error) {
	if input.ID <= 0 {
		return nil, errors.New("request id is required")
	}
	if strings.TrimSpace(input.Name) != "" {
		return nil, errors.New("name is not supported; use key")
	}
	if err := validateRequestStatus(input.To); err != nil {
		return nil, err
	}

	request, err := s.getRequestByID(input.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("connect request not found: %d", input.ID)
		}
		return nil, err
	}
	if key := strings.TrimSpace(input.Key); key != "" && request.Name != key {
		return nil, fmt.Errorf("request %d does not belong to %s", input.ID, key)
	}
	if input.From != nil {
		if err := validateRequestStatus(*input.From); err != nil {
			return nil, err
		}
	}

	res, err := s.updateRequestStatusRow(input)
	if err != nil {
		return nil, err
	}
	affected, _ := res.RowsAffected()
	if input.Strict && affected == 0 {
		if input.From != nil {
			return nil, fmt.Errorf("connect request %d status mismatch: expected %d", input.ID, *input.From)
		}
		return nil, fmt.Errorf("connect request %d status unchanged", input.ID)
	}
	if affected > 0 {
		s.invalidateCache()
	}
	return s.getRequestByID(input.ID)
}

func (s *Service) updateRequestStatusRow(input RequestStatusUpdate) (sql.Result, error) {
	query := `UPDATE connect_request SET status = ? WHERE id = ?`
	args := []interface{}{input.To, input.ID}
	if key := strings.TrimSpace(input.Key); key != "" {
		query += ` AND name = ?`
		args = append(args, key)
	}
	if input.From != nil {
		query += ` AND status = ?`
		args = append(args, *input.From)
	}
	return s.db.Exec(query, args...)
}

func (s *Service) AddResponse(input ResponseInput) (*Response, error) {
	key := strings.TrimSpace(input.Key)
	if key == "" {
		return nil, errors.New("key is required")
	}
	meta, err := s.mustActiveMeta(key)
	if err != nil {
		return nil, err
	}
	if err := s.validateMetaRuntime(meta); err != nil {
		return nil, err
	}
	if input.RequestID <= 0 {
		return nil, errors.New("request-id is required")
	}
	content := strings.TrimSpace(input.Response)
	if content == "" {
		return nil, errors.New("response is required")
	}
	artifacts, err := normalizeArtifacts(input.Artifacts)
	if err != nil {
		return nil, err
	}
	var requestName string
	err = s.db.QueryRow(`SELECT name FROM connect_request WHERE id = ?`, input.RequestID).Scan(&requestName)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("request not found: %d", input.RequestID)
	}
	if err != nil {
		return nil, err
	}
	if requestName != meta.runtimeKey() {
		return nil, fmt.Errorf("request %d does not belong to %s", input.RequestID, meta.runtimeKey())
	}
	res, err := s.db.Exec(`
		INSERT INTO connect_response (name, request_id, response, artifacts, created_at)
		VALUES (?,?,?,?,?)
	`, meta.runtimeKey(), input.RequestID, content, artifacts, timestamp())
	if err != nil {
		return nil, err
	}
	if _, err := s.db.Exec(`UPDATE connect_request SET status = ? WHERE id = ?`, RequestStatusReplied, input.RequestID); err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	s.invalidateCache()
	return s.getResponseByID(int(id))
}

func (s *Service) ListResponses(filter ResponseFilter) ([]Response, error) {
	if strings.TrimSpace(filter.Name) != "" {
		return nil, errors.New("name filter is not supported; use key")
	}
	if filter.Limit <= 0 {
		filter.Limit = 100
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000
	}
	key := fmt.Sprintf("response:list:%s:%d:%d:%d", strings.TrimSpace(filter.Key), filter.RequestID, filter.AfterID, filter.Limit)
	var items []Response
	err := s.cachedJSON(key, &items, func() (any, error) {
		parts := []string{`SELECT id, name, request_id, response, artifacts, created_at FROM connect_response WHERE 1 = 1`}
		args := make([]any, 0)
		if key := strings.TrimSpace(filter.Key); key != "" {
			parts = append(parts, `AND name = ?`)
			args = append(args, key)
		}
		if filter.RequestID > 0 {
			parts = append(parts, `AND request_id = ?`)
			args = append(args, filter.RequestID)
		}
		if filter.AfterID > 0 {
			parts = append(parts, `AND id > ?`)
			args = append(args, filter.AfterID)
		}
		parts = append(parts, `ORDER BY id LIMIT ?`)
		args = append(args, filter.Limit)
		rows, err := s.db.Query(strings.Join(parts, " "), args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := make([]Response, 0)
		for rows.Next() {
			var item Response
			if err := rows.Scan(&item.ID, &item.Name, &item.RequestID, &item.Response, &item.Artifacts, &item.CreatedAt); err != nil {
				return nil, err
			}
			out = append(out, item)
		}
		return out, rows.Err()
	})
	return items, err
}

func (s *Service) initDB() error {
	statements := []string{
		`PRAGMA journal_mode=WAL`,
		`CREATE TABLE IF NOT EXISTS token_store (model TEXT PRIMARY KEY, token TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS token_store_log (id INTEGER PRIMARY KEY AUTOINCREMENT, model TEXT NOT NULL, token TEXT NOT NULL DEFAULT '', action TEXT NOT NULL, updated_at TEXT NOT NULL)`,
		`
			CREATE TABLE IF NOT EXISTS connect_meta (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				meta TEXT NOT NULL,
				stream INTEGER NOT NULL DEFAULT 0,
				callback TEXT NOT NULL,
				agent_id TEXT NOT NULL,
				chat_id TEXT NOT NULL DEFAULT '',
				model TEXT NOT NULL,
				thinking INTEGER NOT NULL DEFAULT 0,
				verify INTEGER NOT NULL DEFAULT 0,
				router_disable INTEGER NOT NULL DEFAULT 1,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL,
				deleted_at TEXT NOT NULL DEFAULT ''
			)
		`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_connect_meta_name_active ON connect_meta(name) WHERE deleted_at = ''`,
		`
			CREATE TABLE IF NOT EXISTS connect_request (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				external_id TEXT NOT NULL DEFAULT '',
				request TEXT NOT NULL,
				artifacts TEXT NOT NULL DEFAULT '',
				raw_request TEXT NOT NULL DEFAULT '',
				response_schema TEXT NOT NULL DEFAULT '',
				status INTEGER NOT NULL DEFAULT 0,
				created_at TEXT NOT NULL
			)
		`,
		`CREATE INDEX IF NOT EXISTS idx_connect_request_name_id ON connect_request(name, id)`,
		`CREATE INDEX IF NOT EXISTS idx_connect_request_name_created_at ON connect_request(name, created_at)`,
		`
			CREATE TABLE IF NOT EXISTS connect_message_snapshot (
				source TEXT NOT NULL,
				message_id TEXT NOT NULL,
				sender_id TEXT NOT NULL,
				content TEXT NOT NULL DEFAULT '',
				content_folded TEXT NOT NULL DEFAULT '',
				message_type TEXT NOT NULL DEFAULT '',
				sent_at INTEGER NOT NULL,
				PRIMARY KEY (source, message_id)
			)
		`,
		`CREATE INDEX IF NOT EXISTS idx_connect_message_snapshot_source_sent_at ON connect_message_snapshot(source, sent_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_connect_message_snapshot_source_sender_sent_at ON connect_message_snapshot(source, sender_id, sent_at DESC)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS connect_message_snapshot_fts USING fts5(source UNINDEXED, message_id UNINDEXED, content, tokenize = 'trigram case_sensitive 0')`,
		`
			CREATE TABLE IF NOT EXISTS connect_response (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				request_id INTEGER NOT NULL,
				response TEXT NOT NULL,
				artifacts TEXT NOT NULL DEFAULT '',
				created_at TEXT NOT NULL
			)
		`,
		`CREATE INDEX IF NOT EXISTS idx_connect_response_name_id ON connect_response(name, id)`,
		`CREATE INDEX IF NOT EXISTS idx_connect_response_request_id ON connect_response(request_id, id)`,
	}
	for _, stmt := range statements {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	_, _ = s.db.Exec(`ALTER TABLE connect_request ADD COLUMN external_id TEXT NOT NULL DEFAULT ''`)
	_, _ = s.db.Exec(`ALTER TABLE connect_request ADD COLUMN raw_request TEXT NOT NULL DEFAULT ''`)
	_, _ = s.db.Exec(`ALTER TABLE connect_request ADD COLUMN response_schema TEXT NOT NULL DEFAULT ''`)
	_, _ = s.db.Exec(`ALTER TABLE connect_request ADD COLUMN status INTEGER NOT NULL DEFAULT 0`)
	ensureConnectMetaVerifyColumn(s.db)
	ensureConnectMetaRouterDisableColumn(s.db)
	_, _ = s.db.Exec(`UPDATE connect_request SET status = 0 WHERE status IS NULL`)
	_, _ = s.db.Exec(`UPDATE connect_request SET status = 4 WHERE status = 3 AND EXISTS (SELECT 1 FROM connect_response WHERE connect_response.request_id = connect_request.id)`)
	_, _ = s.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_connect_request_name_external_id ON connect_request(name, external_id) WHERE external_id != ''`)
	return nil
}

func ensureConnectMetaVerifyColumn(db *sql.DB) {
	if db == nil || hasTableColumn(db, "connect_meta", "verify") {
		return
	}
	_, _ = db.Exec(`ALTER TABLE connect_meta ADD COLUMN verify INTEGER NOT NULL DEFAULT 0`)
}

func ensureConnectMetaRouterDisableColumn(db *sql.DB) {
	if db == nil || hasTableColumn(db, "connect_meta", "router_disable") {
		return
	}
	_, _ = db.Exec(`ALTER TABLE connect_meta ADD COLUMN router_disable INTEGER NOT NULL DEFAULT 1`)
}

func connectMetaVerifySelect(db *sql.DB) string {
	if hasTableColumn(db, "connect_meta", "verify") {
		return "verify"
	}
	return "0 AS verify"
}

func hasTableColumn(db *sql.DB, table, column string) bool {
	if db == nil {
		return false
	}
	rows, err := db.Query(fmt.Sprintf(`SELECT name FROM pragma_table_info('%s')`, strings.TrimSpace(table)))
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil && strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(column)) {
			return true
		}
	}
	return false
}

func connectMetaRouterDisableSelect(db *sql.DB) string {
	if hasTableColumn(db, "connect_meta", "router_disable") {
		return "router_disable"
	}
	return "1 AS router_disable"
}

func (s *Service) validateMetaInput(input MetaInput) error {
	input.Key = strings.TrimSpace(input.Key)
	input.Name = strings.TrimSpace(input.Name)
	input.Meta = strings.TrimSpace(input.Meta)
	input.Callback = strings.TrimSpace(input.Callback)
	input.AgentID = strings.TrimSpace(input.AgentID)
	input.ChatID = strings.TrimSpace(input.ChatID)
	input.Model = strings.TrimSpace(input.Model)

	if input.Key == "" {
		return errors.New("key is required")
	}
	if input.Meta == "" {
		return errors.New("meta is required")
	}
	if !json.Valid([]byte(input.Meta)) {
		return errors.New("meta must be valid json")
	}
	if input.Callback == "" {
		return errors.New("callback is required")
	}
	scope, hasPluginScope, err := resolvePluginScopeByKey(input.Key)
	if err != nil {
		return err
	}
	if !hasPluginScope || pluginScopeEnabled(scope, "agent") {
		if err := s.validateAgentExists(input.AgentID); err != nil {
			return err
		}
	}
	if !hasPluginScope || pluginScopeEnabled(scope, "provider") {
		if err := s.validateRegisteredModel(input.Model); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) normalizeMetaInput(input MetaInput) (MetaInput, error) {
	input.Key = strings.TrimSpace(input.Key)
	input.Name = strings.TrimSpace(input.Name)
	input.Meta = strings.TrimSpace(input.Meta)
	input.Callback = strings.TrimSpace(input.Callback)
	input.AgentID = strings.TrimSpace(input.AgentID)
	input.ChatID = strings.TrimSpace(input.ChatID)
	input.Model = strings.TrimSpace(input.Model)

	if input.Key == "" {
		return input, nil
	}
	input.Name = input.Key
	input.Callback = defaultCallbackPathForKey(input.Key)
	return input, nil
}

func defaultCallbackPathForKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if pluginDir, err := pluginDirResolver(); err == nil && strings.TrimSpace(pluginDir) != "" {
		return filepath.Clean(filepath.Join(pluginDir, key))
	}
	return filepath.Join("plugins", key)
}

func (s *Service) validateMetaRuntime(meta *Meta) error {
	if meta == nil {
		return errors.New("connect meta is required")
	}
	if meta.DeletedAt != "" {
		return fmt.Errorf("connect meta deleted: %s", meta.Name)
	}
	scope, hasPluginScope, err := resolvePluginScopeByKey(meta.runtimeKey())
	if err != nil {
		return err
	}
	if !hasPluginScope || pluginScopeEnabled(scope, "agent") {
		if err := s.validateAgentExists(meta.AgentID); err != nil {
			return err
		}
	}
	if !hasPluginScope || pluginScopeEnabled(scope, "provider") {
		return s.validateRegisteredModel(meta.Model)
	}
	return nil
}

func (s *Service) validateAgentExists(agentID string) error {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return errors.New("agent is required")
	}
	cleanDir := filepath.Clean(s.agentDir)
	if filepath.Base(cleanDir) == agentID {
		if info, err := os.Stat(cleanDir); err == nil && info.IsDir() {
			return nil
		}
	}
	info, err := os.Stat(filepath.Join(s.agentDir, agentID))
	if err != nil || !info.IsDir() {
		return fmt.Errorf("agent not found: %s", agentID)
	}
	return nil
}

func (s *Service) validateRegisteredModel(model string) error {
	model = sharedutil.NormalizeTokenModelName(model)
	if model == "" {
		return errors.New("model is required")
	}
	token, found, err := s.lookupRegisteredModelToken(model)
	if err != nil {
		return fmt.Errorf("query token failed: %w", err)
	}
	if !found {
		return fmt.Errorf("model not registered: %s", model)
	}
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("model token is empty: %s", model)
	}
	return nil
}

func (s *Service) lookupRegisteredModelToken(model string) (string, bool, error) {
	model = sharedutil.NormalizeTokenModelName(model)
	if model == "" {
		return "", false, nil
	}
	rows, err := s.db.Query(`SELECT model, token FROM token_store`)
	if err != nil {
		return "", false, err
	}
	defer rows.Close()

	var fallback string
	for rows.Next() {
		var storedModel string
		var token string
		if err := rows.Scan(&storedModel, &token); err != nil {
			return "", false, err
		}
		if sharedutil.NormalizeTokenModelName(storedModel) != model {
			continue
		}
		if storedModel == model {
			return token, true, rows.Err()
		}
		if fallback == "" {
			fallback = token
		}
	}
	if err := rows.Err(); err != nil {
		return "", false, err
	}
	if fallback != "" {
		return fallback, true, nil
	}
	return "", false, nil
}

func (s *Service) mustActiveMeta(key string) (*Meta, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}
	item, err := s.GetMeta(key, false)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("connect meta not found: %s", key)
		}
		return nil, err
	}
	item.fillRuntimeKey()
	return item, nil
}

func (s *Service) getMetaByID(id int) (*Meta, error) {
	row := s.db.QueryRow(fmt.Sprintf(`
		SELECT id, name, meta, stream, callback, agent_id, chat_id, model, thinking, %s, %s, created_at, updated_at, deleted_at
		FROM connect_meta WHERE id = ?
	`, connectMetaVerifySelect(s.db), connectMetaRouterDisableSelect(s.db)), id)
	return scanMetaRow(row)
}

func (s *Service) getRequestByID(id int) (*Request, error) {
	row := s.db.QueryRow(`SELECT id, name, external_id, request, artifacts, raw_request, response_schema, status, created_at FROM connect_request WHERE id = ?`, id)
	var item Request
	if err := row.Scan(&item.ID, &item.Name, &item.ExternalID, &item.Request, &item.Artifacts, &item.RawRequest, &item.ResponseSchema, &item.Status, &item.CreatedAt); err != nil {
		return nil, err
	}
	item.fillDerivedFields()
	return &item, nil
}

func validateRequestStatus(status int) error {
	switch status {
	case RequestStatusPending, RequestStatusStarted, RequestStatusCompleted, RequestStatusExpired, RequestStatusReplied:
		return nil
	default:
		return fmt.Errorf("status must be one of 0,1,2,3,4")
	}
}

func (s *Service) getResponseByID(id int) (*Response, error) {
	row := s.db.QueryRow(`SELECT id, name, request_id, response, artifacts, created_at FROM connect_response WHERE id = ?`, id)
	var item Response
	if err := row.Scan(&item.ID, &item.Name, &item.RequestID, &item.Response, &item.Artifacts, &item.CreatedAt); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) cachedJSON(key string, dest any, load func() (any, error)) error {
	now := time.Now()
	s.cacheMu.Lock()
	entry, ok := s.cache[key]
	if ok && entry.expires.After(now) {
		s.cacheMu.Unlock()
		return json.Unmarshal(entry.payload, dest)
	}
	s.cacheMu.Unlock()

	value, err := load()
	if err != nil {
		return err
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	s.cacheMu.Lock()
	s.cache[key] = cacheEntry{payload: payload, expires: now.Add(s.cacheTTL)}
	s.cacheMu.Unlock()
	return json.Unmarshal(payload, dest)
}

func (s *Service) invalidateCache() {
	s.cacheMu.Lock()
	s.cache = make(map[string]cacheEntry)
	s.cacheMu.Unlock()
}

func scanMetaRow(row *sql.Row) (*Meta, error) {
	var item Meta
	var stream int
	var thinking int
	var verify int
	var routerDisable int
	if err := row.Scan(
		&item.ID,
		&item.Name,
		&item.Meta,
		&stream,
		&item.Callback,
		&item.AgentID,
		&item.ChatID,
		&item.Model,
		&thinking,
		&verify,
		&routerDisable,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.DeletedAt,
	); err != nil {
		return nil, err
	}
	item.Stream = stream != 0
	item.Thinking = thinking != 0
	item.Verify = verify != 0
	item.RouterDisable = routerDisable != 0
	item.fillRuntimeKey()
	return &item, nil
}

func scanMetaRows(rows *sql.Rows) (*Meta, error) {
	var item Meta
	var stream int
	var thinking int
	var verify int
	var routerDisable int
	if err := rows.Scan(
		&item.ID,
		&item.Name,
		&item.Meta,
		&stream,
		&item.Callback,
		&item.AgentID,
		&item.ChatID,
		&item.Model,
		&thinking,
		&verify,
		&routerDisable,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.DeletedAt,
	); err != nil {
		return nil, err
	}
	item.Stream = stream != 0
	item.Thinking = thinking != 0
	item.Verify = verify != 0
	item.RouterDisable = routerDisable != 0
	item.fillRuntimeKey()
	return &item, nil
}

func normalizeArtifacts(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{})
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			return "", errors.New("artifacts contains empty path")
		}
		if !strings.Contains(item, "://") {
			abs, err := filepath.Abs(item)
			if err == nil {
				item = abs
			}
			if _, err := os.Stat(item); err != nil {
				return "", fmt.Errorf("artifact not found: %s", item)
			}
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return strings.Join(out, ","), nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func timestamp() string {
	return time.Now().Format(time.RFC3339)
}

func normalizeRequestCreatedAt(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return timestamp(), nil
	}
	if unixMS, err := strconv.ParseInt(raw, 10, 64); err == nil {
		switch {
		case len(raw) >= 16:
			return time.Unix(0, unixMS).UTC().Format(time.RFC3339), nil
		case len(raw) >= 13:
			return time.UnixMilli(unixMS).UTC().Format(time.RFC3339), nil
		default:
			return time.Unix(unixMS, 0).UTC().Format(time.RFC3339), nil
		}
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return "", fmt.Errorf("created must be unix timestamp or RFC3339 time")
	}
	return parsed.Format(time.RFC3339), nil
}

func isUniqueErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "unique")
}

func absOrOriginal(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func metaToConfig(item Meta) (MetaConfig, error) {
	metaMap := map[string]any{}
	if strings.TrimSpace(item.Meta) != "" {
		if err := json.Unmarshal([]byte(item.Meta), &metaMap); err != nil {
			return MetaConfig{}, fmt.Errorf("decode meta for %s: %w", item.Name, err)
		}
	}
	callback := strings.TrimSpace(item.Callback)
	if callback != "" {
		callback = absOrOriginal(callback)
	}
	key := item.runtimeKey()
	displayName := strings.TrimSpace(item.Name)
	if callback != "" {
		if base := strings.TrimSpace(filepath.Base(callback)); base != "" && base != "." && base != string(filepath.Separator) {
			key = base
		}
		if plugin, err := inspectPlugin(callback); err == nil {
			if strings.TrimSpace(plugin.Key) != "" {
				key = strings.TrimSpace(plugin.Key)
			}
			if strings.TrimSpace(plugin.Name) != "" {
				displayName = strings.TrimSpace(plugin.Name)
			}
		}
	}
	return MetaConfig{
		Key:           key,
		Name:          displayName,
		Meta:          metaMap,
		Stream:        item.Stream,
		Callback:      callback,
		AgentID:       item.AgentID,
		ChatID:        item.ChatID,
		Model:         item.Model,
		Thinking:      item.Thinking,
		Verify:        item.Verify,
		RouterDisable: item.RouterDisable,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}, nil
}

func (m *Meta) runtimeKey() string {
	if m == nil {
		return ""
	}
	if key := strings.TrimSpace(m.Key); key != "" {
		return key
	}
	return strings.TrimSpace(m.Name)
}

func (m *Meta) fillRuntimeKey() {
	if m == nil {
		return
	}
	if strings.TrimSpace(m.Key) != "" {
		return
	}
	m.Key = strings.TrimSpace(m.Name)
}

func (r *Request) fillDerivedFields() {
	if r == nil {
		return
	}
	if strings.TrimSpace(r.Key) == "" {
		r.Key = r.Name
	}
	r.Content = r.Request
	r.Original = r.RawRequest
}
