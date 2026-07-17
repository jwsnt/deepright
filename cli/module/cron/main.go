package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"knowledge/knowledgecore"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/glebarez/go-sqlite"
	"github.com/robfig/cron/v3"
)

// ===== Task Metadata =====

type TaskMeta struct {
	ID             int    `json:"id"`
	Cycle          int    `json:"cycle"`   // 0=once, 1=workday, 2=daily, 3=hourly, 4=every15, 5=every30, -1=custom cron
	RawTime        string `json:"rawTime"` // yyyy-MM-dd hh:mm (empty if custom cron)
	AgentID        string `json:"agentId"`
	ChatID         string `json:"chatId,omitempty"`
	Type           string `json:"type"`
	Model          string `json:"model"`
	Thinking       bool   `json:"thinking"`
	RouterDisable  bool   `json:"router_disable"`
	Cron           string `json:"cron"`
	Content        string `json:"content"`
	ResponseSchema string `json:"responseSchema"`
}

// ===== Task Detail =====

type TaskDetail struct {
	ID             int    `json:"id"`
	MetaID         int    `json:"metaId"`
	ExecTime       int64  `json:"execTime"`
	AgentID        string `json:"agentId"`
	ChatID         string `json:"chatId,omitempty"`
	Type           string `json:"type"`
	Model          string `json:"model"`
	Thinking       bool   `json:"thinking"`
	RouterDisable  bool   `json:"router_disable"`
	Content        string `json:"content"`
	ResponseSchema string `json:"responseSchema"`
	Started        int    `json:"started"` // 0=未启动, 1=已启动, 2=无需启动, 3=已完成
}

// ===== Cron Service =====

type CronService struct {
	mu            sync.Mutex
	db            *sql.DB
	stopCh        chan struct{}
	LastCheckTime time.Time
}

type cronMetaLogEntry struct {
	MetaID         int
	AgentID        string
	ChatID         string
	TaskType       string
	Action         string
	Cycle          int
	RawTime        string
	Model          string
	Thinking       bool
	RouterDisable  bool
	Cron           string
	Content        string
	ResponseSchema string
	CreatedAt      string
}

type cronDetailLogEntry struct {
	DetailID       int
	MetaID         int
	AgentID        string
	ChatID         string
	TaskType       string
	Action         string
	ExecTime       int64
	Model          string
	Thinking       bool
	RouterDisable  bool
	Content        string
	ResponseSchema string
	Started        int
	OccurredAt     string
}

const defaultTaskType = "cron"

func normalizeTaskType(taskType string) string {
	taskType = strings.TrimSpace(taskType)
	if taskType == "" {
		return defaultTaskType
	}
	return taskType
}

func NewCronService(dbPath string) (*CronService, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	svc := &CronService{db: db, stopCh: make(chan struct{})}
	if err := svc.initDB(); err != nil {
		return nil, err
	}
	return svc, nil
}

func (s *CronService) initDB() error {
	s.db.Exec(`PRAGMA journal_mode=WAL`)
	return s.ensureSchema()
}

func (s *CronService) ensureSchema() error {
	s.renameCronLogTables()
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS task_meta (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cycle INTEGER NOT NULL,
			raw_time TEXT NOT NULL DEFAULT '',
			agent_id TEXT NOT NULL,
			chat_id TEXT NOT NULL DEFAULT '',
			task_type TEXT NOT NULL DEFAULT 'cron',
			model TEXT NOT NULL,
			thinking INTEGER NOT NULL DEFAULT 0,
			router_disable INTEGER NOT NULL DEFAULT 1,
			cron TEXT NOT NULL,
			content TEXT NOT NULL,
			response_schema TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE IF NOT EXISTS task_detail (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			meta_id INTEGER NOT NULL,
			exec_time INTEGER NOT NULL,
			agent_id TEXT NOT NULL,
			chat_id TEXT NOT NULL DEFAULT '',
			task_type TEXT NOT NULL DEFAULT 'cron',
			model TEXT NOT NULL,
			thinking INTEGER NOT NULL DEFAULT 0,
			router_disable INTEGER NOT NULL DEFAULT 1,
			content TEXT NOT NULL,
			response_schema TEXT NOT NULL DEFAULT '',
			started INTEGER NOT NULL DEFAULT 0,
			UNIQUE(meta_id, exec_time)
		);
		CREATE TABLE IF NOT EXISTS cron_meta_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			meta_id INTEGER NOT NULL,
			agent_id TEXT NOT NULL,
			chat_id TEXT NOT NULL DEFAULT '',
			task_type TEXT NOT NULL DEFAULT 'cron',
			action TEXT NOT NULL,
			cycle INTEGER NOT NULL,
			raw_time TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL,
			thinking INTEGER NOT NULL DEFAULT 0,
			router_disable INTEGER NOT NULL DEFAULT 1,
			cron TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			response_schema TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS cron_detail_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			detail_id INTEGER NOT NULL,
			meta_id INTEGER NOT NULL,
			agent_id TEXT NOT NULL,
			chat_id TEXT NOT NULL DEFAULT '',
			task_type TEXT NOT NULL DEFAULT 'cron',
			action TEXT NOT NULL,
			exec_time INTEGER NOT NULL,
			model TEXT NOT NULL,
			thinking INTEGER NOT NULL DEFAULT 0,
			router_disable INTEGER NOT NULL DEFAULT 1,
			content TEXT NOT NULL,
			response_schema TEXT NOT NULL DEFAULT '',
			started INTEGER NOT NULL DEFAULT 0,
			occurred_at TEXT NOT NULL
		);
	`)
	for _, stmt := range []string{
		`ALTER TABLE task_meta ADD COLUMN chat_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE task_meta ADD COLUMN task_type TEXT NOT NULL DEFAULT 'cron'`,
		`ALTER TABLE task_meta ADD COLUMN response_schema TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE task_meta ADD COLUMN router_disable INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE task_detail ADD COLUMN chat_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE task_detail ADD COLUMN started INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE task_detail ADD COLUMN task_type TEXT NOT NULL DEFAULT 'cron'`,
		`ALTER TABLE task_detail ADD COLUMN response_schema TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE task_detail ADD COLUMN router_disable INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE cron_meta_log ADD COLUMN chat_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE cron_meta_log ADD COLUMN task_type TEXT NOT NULL DEFAULT 'cron'`,
		`ALTER TABLE cron_meta_log ADD COLUMN response_schema TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE cron_meta_log ADD COLUMN router_disable INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE cron_detail_log ADD COLUMN chat_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE cron_detail_log ADD COLUMN started INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE cron_detail_log ADD COLUMN task_type TEXT NOT NULL DEFAULT 'cron'`,
		`ALTER TABLE cron_detail_log ADD COLUMN response_schema TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE cron_detail_log ADD COLUMN router_disable INTEGER NOT NULL DEFAULT 1`,
		`DROP INDEX IF EXISTS idx_detail_agent_time`,
		`DROP INDEX IF EXISTS idx_detail_agent_chat_time`,
		`DROP INDEX IF EXISTS idx_detail_agent_chat_time_type`,
		`DROP INDEX IF EXISTS idx_detail_agent_exec_status`,
		`DROP INDEX IF EXISTS idx_detail_exec_status`,
		`CREATE INDEX IF NOT EXISTS idx_detail_agent_exec_status ON task_detail(agent_id, exec_time, started)`,
		`CREATE INDEX IF NOT EXISTS idx_detail_exec_status ON task_detail(exec_time, started)`,
		`CREATE INDEX IF NOT EXISTS idx_meta_agent ON task_meta(agent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cron_meta_log_agent_chat_time ON cron_meta_log(agent_id, chat_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_cron_meta_log_meta_time ON cron_meta_log(meta_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_cron_detail_log_agent_chat_time ON cron_detail_log(agent_id, chat_id, occurred_at)`,
		`CREATE INDEX IF NOT EXISTS idx_cron_detail_log_meta_time ON cron_detail_log(meta_id, occurred_at)`,
		`CREATE INDEX IF NOT EXISTS idx_cron_detail_log_detail_time ON cron_detail_log(detail_id, occurred_at)`,
	} {
		if _, execErr := s.db.Exec(stmt); execErr != nil {
			lower := strings.ToLower(execErr.Error())
			if !strings.Contains(lower, "duplicate column name") {
				log.Printf("[cron] schema statement failed: %s: %v", stmt, execErr)
			}
		}
	}
	for _, table := range []string{"task_meta", "task_detail", "cron_meta_log", "cron_detail_log"} {
		s.ensureRouterDisableColumn(table)
	}
	return err
}

func (s *CronService) ensureRouterDisableColumn(table string) {
	if s == nil || s.db == nil || hasTableColumn(s.db, table, "router_disable") {
		return
	}
	_, _ = s.db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN router_disable INTEGER NOT NULL DEFAULT 1`, table))
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

func routerDisableSelect(db *sql.DB, table string) string {
	if hasTableColumn(db, table, "router_disable") {
		return "router_disable"
	}
	return "1 AS router_disable"
}

func (s *CronService) renameCronLogTables() {
	rows, err := s.db.Query(`SELECT name FROM sqlite_master WHERE type='table'`)
	if err != nil {
		return
	}
	defer rows.Close()

	foundMeta := false
	foundDetail := false
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		switch name {
		case "cron_meta_log":
			foundMeta = true
		case "cron_detail_log":
			foundDetail = true
		case "task_meta_log":
			if !foundMeta {
				s.db.Exec(`ALTER TABLE task_meta_log RENAME TO cron_meta_log`)
				foundMeta = true
			}
		case "task_detail_log":
			if !foundDetail {
				s.db.Exec(`ALTER TABLE task_detail_log RENAME TO cron_detail_log`)
				foundDetail = true
			}
		}
	}
}

func cronLogTimestamp() string {
	return time.Now().Format("2006-01-02T15:04:05.000")
}

func (s *CronService) appendMetaLog(entry cronMetaLogEntry) {
	_, err := s.db.Exec(`INSERT INTO cron_meta_log (meta_id, agent_id, chat_id, task_type, action, cycle, raw_time, model, thinking, router_disable, cron, content, response_schema, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		entry.MetaID, entry.AgentID, entry.ChatID, entry.TaskType, entry.Action, entry.Cycle, entry.RawTime, entry.Model, boolToInt(entry.Thinking), boolToInt(entry.RouterDisable), entry.Cron, entry.Content, entry.ResponseSchema, entry.CreatedAt)
	if err != nil {
		log.Printf("[cron] append meta log failed: %v", err)
	}
}

func (s *CronService) appendDetailLog(entry cronDetailLogEntry) {
	_, err := s.db.Exec(`INSERT INTO cron_detail_log (detail_id, meta_id, agent_id, chat_id, task_type, action, exec_time, model, thinking, router_disable, content, response_schema, started, occurred_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		entry.DetailID, entry.MetaID, entry.AgentID, entry.ChatID, entry.TaskType, entry.Action, entry.ExecTime, entry.Model, boolToInt(entry.Thinking), boolToInt(entry.RouterDisable), entry.Content, entry.ResponseSchema, entry.Started, entry.OccurredAt)
	if err != nil {
		log.Printf("[cron] append detail log failed: %v", err)
	}
}

// Submit with cycle + rawTime (auto-generate cron)
func (s *CronService) Submit(cycle int, rawTime, agentID, chatID, model, content string, thinking bool) (*TaskMeta, error) {
	return s.SubmitWithTypeSchemaAndRouterDisable(cycle, rawTime, agentID, chatID, defaultTaskType, model, content, "", true, thinking)
}

func (s *CronService) SubmitWithType(cycle int, rawTime, agentID, chatID, taskType, model, content string, thinking bool) (*TaskMeta, error) {
	return s.SubmitWithTypeSchemaAndRouterDisable(cycle, rawTime, agentID, chatID, taskType, model, content, "", true, thinking)
}

func (s *CronService) SubmitWithTypeAndSchema(cycle int, rawTime, agentID, chatID, taskType, model, content, responseSchema string, thinking bool) (*TaskMeta, error) {
	return s.SubmitWithTypeSchemaAndRouterDisable(cycle, rawTime, agentID, chatID, taskType, model, content, responseSchema, true, thinking)
}

func (s *CronService) SubmitWithTypeSchemaAndRouterDisable(cycle int, rawTime, agentID, chatID, taskType, model, content, responseSchema string, routerDisable, thinking bool) (*TaskMeta, error) {
	t, err := time.ParseInLocation("2006-01-02 15:04", rawTime, time.Local)
	if err != nil {
		return nil, fmt.Errorf("invalid time format: %w", err)
	}
	cronExpr := buildCron(cycle, t)
	if cronExpr == "" {
		return nil, fmt.Errorf("unsupported cycle: %d", cycle)
	}
	return s.insert(cycle, rawTime, agentID, chatID, taskType, model, content, responseSchema, cronExpr, routerDisable, thinking, true)
}

// SubmitCron with explicit cron expression (no immediate detail creation)
func (s *CronService) SubmitCron(cronExpr, agentID, chatID, model, content string, thinking bool) (*TaskMeta, error) {
	return s.SubmitCronWithTypeSchemaAndRouterDisable(cronExpr, agentID, chatID, defaultTaskType, model, content, "", true, thinking)
}

func (s *CronService) SubmitCronWithType(cronExpr, agentID, chatID, taskType, model, content string, thinking bool) (*TaskMeta, error) {
	return s.SubmitCronWithTypeSchemaAndRouterDisable(cronExpr, agentID, chatID, taskType, model, content, "", true, thinking)
}

func (s *CronService) SubmitCronWithTypeAndSchema(cronExpr, agentID, chatID, taskType, model, content, responseSchema string, thinking bool) (*TaskMeta, error) {
	return s.SubmitCronWithTypeSchemaAndRouterDisable(cronExpr, agentID, chatID, taskType, model, content, responseSchema, true, thinking)
}

func (s *CronService) SubmitCronWithTypeSchemaAndRouterDisable(cronExpr, agentID, chatID, taskType, model, content, responseSchema string, routerDisable, thinking bool) (*TaskMeta, error) {
	cronExpr = strings.TrimSpace(cronExpr)
	if cronExpr == "" {
		return nil, fmt.Errorf("cron expression is required")
	}
	return s.insert(-1, "", agentID, chatID, taskType, model, content, responseSchema, cronExpr, routerDisable, thinking, false)
}

func (s *CronService) insert(cycle int, rawTime, agentID, chatID, taskType, model, content, responseSchema, cronExpr string, routerDisable, thinking, createFirst bool) (*TaskMeta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	chatID = strings.TrimSpace(chatID)
	taskType = normalizeTaskType(taskType)
	responseSchema = strings.TrimSpace(responseSchema)
	res, err := s.db.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, chat_id, task_type, model, thinking, router_disable, cron, content, response_schema) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		cycle, rawTime, agentID, chatID, taskType, model, boolToInt(thinking), boolToInt(routerDisable), cronExpr, content, responseSchema)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	meta := &TaskMeta{ID: int(id), Cycle: cycle, RawTime: rawTime, AgentID: agentID, ChatID: chatID, Type: taskType, Model: model, Thinking: thinking, RouterDisable: routerDisable, Cron: cronExpr, Content: content, ResponseSchema: responseSchema}
	s.appendMetaLog(cronMetaLogEntry{
		MetaID:         meta.ID,
		AgentID:        meta.AgentID,
		ChatID:         meta.ChatID,
		TaskType:       meta.Type,
		Action:         "insert",
		Cycle:          meta.Cycle,
		RawTime:        meta.RawTime,
		Model:          meta.Model,
		Thinking:       meta.Thinking,
		RouterDisable:  meta.RouterDisable,
		Cron:           meta.Cron,
		Content:        meta.Content,
		ResponseSchema: meta.ResponseSchema,
		CreatedAt:      cronLogTimestamp(),
	})

	if createFirst {
		t, _ := time.ParseInLocation("2006-01-02 15:04", rawTime, time.Local)
		s.createInitialDetails(meta, t)
	}
	return meta, nil
}

func (s *CronService) createDetail(meta *TaskMeta, execTime time.Time) bool {
	ts := execTime.Unix()
	res, err := s.db.Exec(`INSERT OR IGNORE INTO task_detail (meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, router_disable, content, response_schema, started) VALUES (?,?,?,?,?,?,?,?,?,?,0)`,
		meta.ID, ts, meta.AgentID, meta.ChatID, meta.Type, meta.Model, boolToInt(meta.Thinking), boolToInt(meta.RouterDisable), meta.Content, meta.ResponseSchema)
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		detailID, _ := res.LastInsertId()
		s.appendDetailLog(cronDetailLogEntry{
			DetailID:       int(detailID),
			MetaID:         meta.ID,
			AgentID:        meta.AgentID,
			ChatID:         meta.ChatID,
			TaskType:       meta.Type,
			Action:         "insert",
			ExecTime:       ts,
			Model:          meta.Model,
			Thinking:       meta.Thinking,
			RouterDisable:  meta.RouterDisable,
			Content:        meta.Content,
			ResponseSchema: meta.ResponseSchema,
			Started:        0,
			OccurredAt:     cronLogTimestamp(),
		})
		log.Printf("[cron] detail created: metaId=%d, execTime=%s", meta.ID, execTime.Format("2006-01-02 15:04"))
		return true
	}
	return false
}

func (s *CronService) createInitialDetails(meta *TaskMeta, base time.Time) {
	switch meta.Cycle {
	case 0:
		s.createDetail(meta, base)
	case 1:
		s.createRange(meta, base, base, base.Add(5*24*time.Hour), true)
	case 2:
		s.createRange(meta, base, base, base.Add(5*24*time.Hour), false)
	case 3, 4, 5:
		interval := cycleInterval(meta.Cycle)
		if interval <= 0 {
			return
		}
		start := base.Truncate(time.Minute)
		if start.Before(base) {
			start = start.Add(interval)
		}
		s.createIntervalRange(meta, base, start, start.Add(5*24*time.Hour), interval)
	}
}

func (s *CronService) CheckAndCreate() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	end := now.Add(5 * 24 * time.Hour)
	s.LastCheckTime = now

	rows, err := s.db.Query(fmt.Sprintf(`SELECT id, cycle, raw_time, agent_id, chat_id, task_type, model, thinking, %s, cron, content, response_schema FROM task_meta`, routerDisableSelect(s.db, "task_meta")))
	if err != nil {
		log.Printf("[cron] check error: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var m TaskMeta
		var th, routerDisable int
		rows.Scan(&m.ID, &m.Cycle, &m.RawTime, &m.AgentID, &m.ChatID, &m.Type, &m.Model, &th, &routerDisable, &m.Cron, &m.Content, &m.ResponseSchema)
		m.Type = normalizeTaskType(m.Type)
		m.Thinking = th != 0
		m.RouterDisable = routerDisable != 0
		s.appendMetaLog(cronMetaLogEntry{
			MetaID:         m.ID,
			AgentID:        m.AgentID,
			ChatID:         m.ChatID,
			TaskType:       m.Type,
			Action:         "select",
			Cycle:          m.Cycle,
			RawTime:        m.RawTime,
			Model:          m.Model,
			Thinking:       m.Thinking,
			RouterDisable:  m.RouterDisable,
			Cron:           m.Cron,
			Content:        m.Content,
			ResponseSchema: m.ResponseSchema,
			CreatedAt:      cronLogTimestamp(),
		})

		if m.Cycle == -1 || m.RawTime == "" {
			// Custom cron: parse expression and find next times
			s.createFromCron(&m, now, end)
		} else {
			t, err := time.ParseInLocation("2006-01-02 15:04", m.RawTime, time.Local)
			if err != nil {
				continue
			}
			switch m.Cycle {
			case 0:
				if !t.Before(now) && t.Before(end) {
					s.createDetail(&m, t)
				}
			case 1:
				s.createRange(&m, t, now, end, true)
			case 2:
				s.createRange(&m, t, now, end, false)
			case 3, 4, 5:
				interval := cycleInterval(m.Cycle)
				if interval > 0 {
					s.createIntervalRange(&m, t, now, end, interval)
				}
			}
		}
	}
}

func (s *CronService) createFromCron(meta *TaskMeta, now, end time.Time) {
	expr := strings.ReplaceAll(meta.Cron, "?", "*")
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sched, err := parser.Parse(expr)
	if err != nil {
		return
	}
	next := sched.Next(now.Add(-1 * time.Second))
	for !next.After(end) && !next.IsZero() {
		s.createDetail(meta, next)
		next = sched.Next(next)
	}
}

func (s *CronService) createRange(meta *TaskMeta, baseTime, now, end time.Time, workdayOnly bool) {
	start := now
	if baseTime.After(now) {
		start = baseTime
	}
	day := time.Date(start.Year(), start.Month(), start.Day(), baseTime.Hour(), baseTime.Minute(), 0, 0, time.Local)
	for !day.After(end) {
		if !day.Before(now) {
			if !workdayOnly || isWorkday(day) {
				s.createDetail(meta, day)
			}
		}
		day = day.AddDate(0, 0, 1)
	}
}

func (s *CronService) createIntervalRange(meta *TaskMeta, anchor, now, end time.Time, interval time.Duration) {
	if interval <= 0 {
		return
	}
	tick := anchor
	if tick.Before(now) {
		steps := now.Sub(anchor) / interval
		tick = anchor.Add(steps * interval)
		if tick.Before(now) {
			tick = tick.Add(interval)
		}
	}
	for !tick.After(end) {
		s.createDetail(meta, tick)
		tick = tick.Add(interval)
	}
}

func (s *CronService) Start() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		s.CheckAndCreate()
		for {
			select {
			case <-ticker.C:
				s.CheckAndCreate()
			case <-s.stopCh:
				return
			}
		}
	}()
	log.Println("[cron] periodic check started (every 1 minute)")
}

func (s *CronService) Stop() {
	close(s.stopCh)
	s.db.Close()
}

func (s *CronService) GetMetas() []TaskMeta {
	s.mu.Lock()
	defer s.mu.Unlock()
	var metas []TaskMeta
	rows, _ := s.db.Query(fmt.Sprintf(`SELECT id, cycle, raw_time, agent_id, chat_id, task_type, model, thinking, %s, cron, content, response_schema FROM task_meta ORDER BY id`, routerDisableSelect(s.db, "task_meta")))
	if rows == nil {
		return metas
	}
	defer rows.Close()
	for rows.Next() {
		var m TaskMeta
		var th, routerDisable int
		rows.Scan(&m.ID, &m.Cycle, &m.RawTime, &m.AgentID, &m.ChatID, &m.Type, &m.Model, &th, &routerDisable, &m.Cron, &m.Content, &m.ResponseSchema)
		m.Type = normalizeTaskType(m.Type)
		m.Thinking = th != 0
		m.RouterDisable = routerDisable != 0
		s.appendMetaLog(cronMetaLogEntry{
			MetaID:         m.ID,
			AgentID:        m.AgentID,
			ChatID:         m.ChatID,
			TaskType:       m.Type,
			Action:         "select",
			Cycle:          m.Cycle,
			RawTime:        m.RawTime,
			Model:          m.Model,
			Thinking:       m.Thinking,
			RouterDisable:  m.RouterDisable,
			Cron:           m.Cron,
			Content:        m.Content,
			ResponseSchema: m.ResponseSchema,
			CreatedAt:      cronLogTimestamp(),
		})
		metas = append(metas, m)
	}
	return metas
}

func (s *CronService) GetDetails() []TaskDetail {
	s.mu.Lock()
	defer s.mu.Unlock()
	var details []TaskDetail
	rows, _ := s.db.Query(fmt.Sprintf(`SELECT id, meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, %s, content, response_schema, started FROM task_detail ORDER BY exec_time`, routerDisableSelect(s.db, "task_detail")))
	if rows == nil {
		return details
	}
	defer rows.Close()
	for rows.Next() {
		var d TaskDetail
		var th, routerDisable int
		rows.Scan(&d.ID, &d.MetaID, &d.ExecTime, &d.AgentID, &d.ChatID, &d.Type, &d.Model, &th, &routerDisable, &d.Content, &d.ResponseSchema, &d.Started)
		d.Type = normalizeTaskType(d.Type)
		d.Thinking = th != 0
		d.RouterDisable = routerDisable != 0
		s.appendDetailLog(cronDetailLogEntry{
			DetailID:       d.ID,
			MetaID:         d.MetaID,
			AgentID:        d.AgentID,
			ChatID:         d.ChatID,
			TaskType:       d.Type,
			Action:         "select",
			ExecTime:       d.ExecTime,
			Model:          d.Model,
			Thinking:       d.Thinking,
			RouterDisable:  d.RouterDisable,
			Content:        d.Content,
			ResponseSchema: d.ResponseSchema,
			Started:        d.Started,
			OccurredAt:     cronLogTimestamp(),
		})
		details = append(details, d)
	}
	return details
}

// ===== Helpers =====

func buildCron(cycle int, t time.Time) string {
	min := t.Minute()
	hour := t.Hour()
	switch cycle {
	case 0:
		return fmt.Sprintf("0 %d %d %d %d ? %d", min, hour, t.Day(), t.Month(), t.Year())
	case 1:
		return fmt.Sprintf("0 %d %d * * 1-5", min, hour)
	case 2:
		return fmt.Sprintf("0 %d %d * * ?", min, hour)
	case 3:
		return fmt.Sprintf("%d * * * *", min)
	case 4:
		return "*/15 * * * *"
	case 5:
		return "*/30 * * * *"
	}
	return ""
}

func cycleInterval(cycle int) time.Duration {
	switch cycle {
	case 3:
		return time.Hour
	case 4:
		return 15 * time.Minute
	case 5:
		return 30 * time.Minute
	default:
		return 0
	}
}

func isWorkday(t time.Time) bool {
	wd := t.Weekday()
	return wd >= time.Monday && wd <= time.Friday
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ===== CLI Entry =====

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}
	if os.Args[1] == "--help" || os.Args[1] == "-h" || os.Args[1] == "help" {
		printHelp()
		os.Exit(0)
	}

	dbPath, err := knowledgecore.DBPath(".")
	if err != nil {
		log.Fatal(err)
	}
	svc, err := NewCronService(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer svc.Stop()

	switch os.Args[1] {
	case "start":
		svc.Start()
		select {} // block forever

	case "create", "submit":
		cycle, rawTime, agentID, chatID, taskType, model, content, responseSchema, agentDir := 0, "", "", "", "", "", "", "", ""
		thinking, routerDisable := false, true
		cronExpr := ""
		if err := parseArgs(os.Args[2:], &cycle, &rawTime, &agentID, &chatID, &taskType, &model, &content, &responseSchema, &thinking, &routerDisable, &cronExpr, &agentDir); err != nil {
			log.Fatal(err)
		}
		if err := validateCreateArgs(cycle, rawTime, agentID, model, content); err != nil {
			log.Fatal(err)
		}
		resolvedAgentDir, err := resolveAgentDir(agentDir)
		if err != nil {
			log.Fatal(err)
		}
		if err := validateAgentExists(resolvedAgentDir, agentID); err != nil {
			log.Fatal(err)
		}
		if err := validateRegisteredModel(model); err != nil {
			log.Fatal(err)
		}
		meta, err := svc.SubmitWithTypeSchemaAndRouterDisable(cycle, rawTime, agentID, chatID, taskType, model, content, responseSchema, routerDisable, thinking)
		if err != nil {
			log.Fatal(err)
		}
		out, _ := json.MarshalIndent(meta, "", "  ")
		fmt.Println(string(out))

	case "create-cron", "submit-cron":
		cronExpr, agentID, chatID, taskType, model, content, responseSchema, agentDir := "", "", "", "", "", "", "", ""
		thinking, routerDisable := false, true
		if err := parseArgs(os.Args[2:], nil, nil, &agentID, &chatID, &taskType, &model, &content, &responseSchema, &thinking, &routerDisable, &cronExpr, &agentDir); err != nil {
			log.Fatal(err)
		}
		if err := validateCreateCronArgs(cronExpr, agentID, model, content); err != nil {
			log.Fatal(err)
		}
		resolvedAgentDir, err := resolveAgentDir(agentDir)
		if err != nil {
			log.Fatal(err)
		}
		if err := validateAgentExists(resolvedAgentDir, agentID); err != nil {
			log.Fatal(err)
		}
		if err := validateRegisteredModel(model); err != nil {
			log.Fatal(err)
		}
		meta, err := svc.SubmitCronWithTypeSchemaAndRouterDisable(cronExpr, agentID, chatID, taskType, model, content, responseSchema, routerDisable, thinking)
		if err != nil {
			log.Fatal(err)
		}
		out, _ := json.MarshalIndent(meta, "", "  ")
		fmt.Println(string(out))

	case "list":
		fmt.Println("=== Metas ===")
		out, _ := json.MarshalIndent(svc.GetMetas(), "", "  ")
		fmt.Println(string(out))
		fmt.Println("=== Details ===")
		out, _ = json.MarshalIndent(svc.GetDetails(), "", "  ")
		fmt.Println(string(out))

	case "check":
		svc.CheckAndCreate()
		fmt.Println("Check complete")
		out, _ := json.MarshalIndent(svc.GetDetails(), "", "  ")
		fmt.Println(string(out))

	default:
		fmt.Println("Unknown command:", os.Args[1])
		printHelp()
		os.Exit(1)
	}
}

func parseArgs(args []string, cycle *int, rawTime, agentID, chatID, taskType, model, content, responseSchema *string, thinking, routerDisable *bool, cronExpr, agentDir *string) error {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		key := strings.TrimPrefix(arg, "--")
		value := ""
		hasValue := false
		if parts := strings.SplitN(key, "=", 2); len(parts) == 2 {
			key = parts[0]
			value = parts[1]
			hasValue = true
		} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			value = args[i+1]
			hasValue = true
			i++
		}

		switch key {
		case "cycle":
			if cycle == nil {
				continue
			}
			if !hasValue {
				return fmt.Errorf("cycle requires a value")
			}
			if _, err := fmt.Sscanf(value, "%d", cycle); err != nil {
				return fmt.Errorf("invalid cycle: %s", value)
			}
		case "time", "rawTime":
			if rawTime != nil && hasValue {
				*rawTime = value
			}
		case "agent", "agentId":
			if agentID != nil && hasValue {
				*agentID = value
			}
		case "agent-dir":
			if agentDir != nil && hasValue {
				*agentDir = value
			}
		case "chat", "chatId":
			if chatID != nil && hasValue {
				*chatID = value
			}
		case "type":
			if taskType != nil && hasValue {
				*taskType = value
			}
		case "model":
			if model != nil && hasValue {
				*model = value
			}
		case "content":
			if content != nil && hasValue {
				*content = value
			}
		case "schema":
			if responseSchema != nil && hasValue {
				*responseSchema = value
			}
		case "cron":
			if cronExpr != nil && hasValue {
				*cronExpr = value
			}
		case "thinking":
			if thinking == nil {
				continue
			}
			if !hasValue {
				*thinking = true
				continue
			}
			parsed, err := parseBoolValue(value)
			if err != nil {
				return err
			}
			*thinking = parsed
		case "router_disable":
			if routerDisable == nil {
				continue
			}
			if !hasValue {
				*routerDisable = true
				continue
			}
			parsed, err := parseBoolValue(value)
			if err != nil {
				return fmt.Errorf("invalid router_disable value: %s", value)
			}
			*routerDisable = parsed
		}
	}
	return nil
}

func resolveAgentDir(agentDir string) (string, error) {
	agentDir = strings.TrimSpace(agentDir)
	if agentDir == "" {
		agentDir = strings.TrimSpace(os.Getenv("AGENT_DIR"))
	}
	if agentDir == "" {
		if info, err := os.Stat("agents"); err == nil && info.IsDir() {
			agentDir = "agents"
		}
	}
	if agentDir == "" {
		return "", fmt.Errorf("agent-dir is required for agent validation")
	}
	absPath, err := filepath.Abs(agentDir)
	if err != nil {
		return "", fmt.Errorf("cannot resolve agent-dir: %w", err)
	}
	return absPath, nil
}

func validateAgentExists(agentDir, agentID string) error {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return fmt.Errorf("agent is required")
	}
	cleanDir := filepath.Clean(agentDir)
	if filepath.Base(cleanDir) == agentID {
		if info, err := os.Stat(cleanDir); err == nil && info.IsDir() {
			return nil
		}
	}
	info, err := os.Stat(filepath.Join(agentDir, agentID))
	if err != nil || !info.IsDir() {
		return fmt.Errorf("agent not found: %s", agentID)
	}
	return nil
}

func validateRegisteredModel(model string) error {
	model = strings.TrimSpace(model)
	if model == "" {
		return fmt.Errorf("model is required")
	}
	dbPath, err := knowledgecore.DBPath(".")
	if err != nil {
		return fmt.Errorf("db path error: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("db error: %w", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS token_store (model TEXT PRIMARY KEY, token TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("ensure token_store failed: %w", err)
	}
	var token string
	err = db.QueryRow(`SELECT token FROM token_store WHERE model = ?`, model).Scan(&token)
	if err == sql.ErrNoRows {
		return fmt.Errorf("model not registered: %s", model)
	}
	if err != nil {
		return fmt.Errorf("query token failed: %w", err)
	}
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("model token is empty: %s", model)
	}
	return nil
}

func parseBoolValue(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "1", "true", "yes", "y", "on":
		return true, nil
	case "0", "false", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid thinking value: %s", value)
	}
}

func validateCreateArgs(cycle int, rawTime, agentID, model, content string) error {
	if cycle < 0 || cycle > 5 {
		return fmt.Errorf("cycle must be one of 0,1,2,3,4,5")
	}
	if strings.TrimSpace(rawTime) == "" {
		return fmt.Errorf("time is required, format: yyyy-MM-dd hh:mm")
	}
	if _, err := time.ParseInLocation("2006-01-02 15:04", rawTime, time.Local); err != nil {
		return fmt.Errorf("invalid time format: %w", err)
	}
	if strings.TrimSpace(agentID) == "" {
		return fmt.Errorf("agent is required")
	}
	if strings.TrimSpace(model) == "" {
		return fmt.Errorf("model is required")
	}
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content is required")
	}
	return nil
}

func validateCreateCronArgs(cronExpr, agentID, model, content string) error {
	if strings.TrimSpace(cronExpr) == "" {
		return fmt.Errorf("cron is required")
	}
	if strings.TrimSpace(agentID) == "" {
		return fmt.Errorf("agent is required")
	}
	if strings.TrimSpace(model) == "" {
		return fmt.Errorf("model is required")
	}
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content is required")
	}
	return nil
}

func printHelp() {
	fmt.Println("Usage:")
	fmt.Println("  cron <command> [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  start         Start periodic check loop (every 1 minute)")
	fmt.Println("  create        Create task metadata from cycle + first start time")
	fmt.Println("  create-cron   Create task metadata from explicit cron expression")
	fmt.Println("  list          Print all task metas and task details")
	fmt.Println("  check         Run periodic check once")
	fmt.Println("  help          Show this help")
	fmt.Println("")
	fmt.Println("Create options:")
	fmt.Println("  --cycle=INT / --cycle INT        0=仅一次, 1=工作日, 2=自然日, 3=每小时, 4=每15分钟, 5=每30分钟")
	fmt.Println("  --time='YYYY-MM-DD HH:MM'        首次开始时间，必填；也支持 --rawTime")
	fmt.Println("  --agent=ID / --agentId ID        绑定的 AgentId，必填")
	fmt.Println("  --agent-dir=DIR / --agent-dir DIR  Agent 根目录；也可用 AGENT_DIR 或当前目录 ./agents")
	fmt.Println("  --chat=ID / --chatId ID          会话 ID（CHAT_ID），可选")
	fmt.Println("  --type=NAME / --type NAME        任务类型，默认 cron；Connect 创建任务时可传具体模块名如 FEISHU")
	fmt.Println("  --model=NAME / --model NAME      模型名称，必填")
	fmt.Println("  --thinking[=true|false]          是否深度思考；也支持 --thinking true")
	fmt.Println("  --router_disable[=true|false]    直接控制路由关闭状态，默认 true")
	fmt.Println("  --content='TEXT'                 任务内容，必填")
	fmt.Println("  --schema='JSON STRING'           LLM Response JSON Schema，可选")
	fmt.Println("")
	fmt.Println("Create-cron options:")
	fmt.Println("  --cron='EXPR' / --cron EXPR      Cron 表达式，必填")
	fmt.Println("  --agent=ID / --agentId ID        绑定的 AgentId，必填")
	fmt.Println("  --agent-dir=DIR / --agent-dir DIR  Agent 根目录；也可用 AGENT_DIR 或当前目录 ./agents")
	fmt.Println("  --chat=ID / --chatId ID          会话 ID（CHAT_ID），可选")
	fmt.Println("  --type=NAME / --type NAME        任务类型，默认 cron；Connect 创建任务时可传具体模块名如 FEISHU")
	fmt.Println("  --model=NAME / --model NAME      模型名称，必填")
	fmt.Println("  --thinking[=true|false]          是否深度思考；也支持 --thinking true")
	fmt.Println("  --router_disable[=true|false]    直接控制路由关闭状态，默认 true")
	fmt.Println("  --content='TEXT'                 任务内容，必填")
	fmt.Println("  --schema='JSON STRING'           LLM Response JSON Schema，可选")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  cron create --agent-dir ../agent/test-case --cycle=0 --time='2026-04-30 12:10' --agent=A --model=OpenAI --thinking --content='查看天气'")
	fmt.Println("  cron create --agent-dir ../agent/test-case --content '每15分钟检查一次上游接口健康' --model OpenAI --thinking true --router_disable false --rawTime '2026-05-03 10:00' --cycle 4 --chatId chat-001 --agent A")
	fmt.Println("  cron create --agent-dir ../agent/test-case --cycle=1 --time='2026-04-30 12:10' --agent=A --chat=chat-001 --model=OpenAI --thinking --content='查看天气'")
	fmt.Println("  cron create-cron --agent-dir ../agent/test-case --cron='10 12 * * 1-5' --agent=A --chat=chat-001 --model=OpenAI --router_disable=false --content='查看天气'")
}
