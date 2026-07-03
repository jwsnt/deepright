package main

import (
	"database/sql"
	"os"
	"testing"
	"time"
)

func newTestSvc(t *testing.T) *CronService {
	t.Helper()
	path := t.TempDir() + "/cron_test.db"
	svc, err := NewCronService(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { svc.db.Close(); os.Remove(path) })
	return svc
}

func TestBuildCron(t *testing.T) {
	tm := time.Date(2026, 4, 30, 12, 10, 0, 0, time.Local)
	tests := []struct {
		cycle int
		want  string
	}{
		{0, "0 10 12 30 4 ? 2026"},
		{1, "0 10 12 * * 1-5"},
		{2, "0 10 12 * * ?"},
	}
	for _, tt := range tests {
		got := buildCron(tt.cycle, tm)
		if got != tt.want {
			t.Errorf("buildCron(%d) = %q, want %q", tt.cycle, got, tt.want)
		}
	}
}

func TestSubmitOnce(t *testing.T) {
	svc := newTestSvc(t)
	meta, err := svc.Submit(0, "2026-04-30 12:10", "A", "", "OpenAI", "查看天气", true)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Cron != "0 10 12 30 4 ? 2026" {
		t.Errorf("cron = %q", meta.Cron)
	}
	details := svc.GetDetails()
	if len(details) != 1 {
		t.Fatalf("expected 1 detail, got %d", len(details))
	}
	if details[0].AgentID != "A" || details[0].Content != "查看天气" || !details[0].Thinking {
		t.Errorf("detail mismatch: %+v", details[0])
	}
	if !meta.RouterDisable {
		t.Errorf("meta.RouterDisable = false, want true")
	}
	if !details[0].RouterDisable {
		t.Errorf("detail.RouterDisable = false, want true")
	}
	if details[0].ChatID != "" {
		t.Errorf("expected empty chatId, got %q", details[0].ChatID)
	}
}

func TestSubmitWorkday(t *testing.T) {
	svc := newTestSvc(t)
	meta, err := svc.Submit(1, "2026-04-30 12:10", "A", "", "OpenAI", "查看天气", true)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Cron != "0 10 12 * * 1-5" {
		t.Errorf("cron = %q", meta.Cron)
	}
	if len(svc.GetDetails()) < 1 {
		t.Error("expected at least 1 detail")
	}
}

func TestSubmitDaily(t *testing.T) {
	svc := newTestSvc(t)
	meta, err := svc.Submit(2, "2026-04-30 12:10", "A", "", "OpenAI", "查看天气", true)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Cron != "0 10 12 * * ?" {
		t.Errorf("cron = %q", meta.Cron)
	}
	if len(svc.GetDetails()) < 1 {
		t.Error("expected at least 1 detail")
	}
}

func TestSubmitCronExpr(t *testing.T) {
	svc := newTestSvc(t)
	// Use every-minute cron so there's always a next occurrence within 5 hours
	meta, err := svc.SubmitCron("* * * * *", "A", "", "OpenAI", "查看天气", true)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Cron != "* * * * *" {
		t.Errorf("cron = %q", meta.Cron)
	}
	if meta.Cycle != -1 {
		t.Errorf("cycle = %d, want -1", meta.Cycle)
	}
	// No immediate detail creation for custom cron
	details := svc.GetDetails()
	if len(details) != 0 {
		t.Errorf("expected 0 details for custom cron submit, got %d", len(details))
	}
	// After check, should create details
	svc.CheckAndCreate()
	details = svc.GetDetails()
	if len(details) < 1 {
		t.Error("expected at least 1 detail after check")
	}
}

func TestDuplicateDetail(t *testing.T) {
	svc := newTestSvc(t)
	svc.Submit(0, "2026-04-30 12:10", "A", "", "OpenAI", "查看天气", true)
	svc.CheckAndCreate()
	details := svc.GetDetails()
	if len(details) != 1 {
		t.Errorf("expected 1 detail (no duplicate), got %d", len(details))
	}
}

func TestIsWorkday(t *testing.T) {
	mon := time.Date(2026, 4, 27, 0, 0, 0, 0, time.Local)
	if !isWorkday(mon) {
		t.Error("Monday should be workday")
	}
	sun := time.Date(2026, 4, 26, 0, 0, 0, 0, time.Local)
	if isWorkday(sun) {
		t.Error("Sunday should not be workday")
	}
}

func TestPersistence(t *testing.T) {
	path := t.TempDir() + "/cron_persist.db"
	svc, _ := NewCronService(path)
	svc.Submit(0, "2026-04-30 12:10", "A", "", "OpenAI", "查看天气", true)
	svc.db.Close()

	svc2, _ := NewCronService(path)
	defer svc2.db.Close()
	if len(svc2.GetMetas()) != 1 {
		t.Error("expected 1 meta after reopen")
	}
	if len(svc2.GetDetails()) != 1 {
		t.Error("expected 1 detail after reopen")
	}
}

func TestSubmitWithChatID(t *testing.T) {
	svc := newTestSvc(t)
	meta, err := svc.Submit(0, "2026-04-30 12:10", "A", "chat-001", "OpenAI", "查看天气", true)
	if err != nil {
		t.Fatal(err)
	}
	if meta.ChatID != "chat-001" {
		t.Fatalf("meta.ChatID = %q, want chat-001", meta.ChatID)
	}
	details := svc.GetDetails()
	if len(details) != 1 {
		t.Fatalf("expected 1 detail, got %d", len(details))
	}
	if details[0].ChatID != "chat-001" {
		t.Fatalf("detail.ChatID = %q, want chat-001", details[0].ChatID)
	}
	if details[0].Type != defaultTaskType {
		t.Fatalf("detail.Type = %q, want %q", details[0].Type, defaultTaskType)
	}
}

func TestTaskMetaPersistsChatID(t *testing.T) {
	svc := newTestSvc(t)
	_, err := svc.Submit(0, "2026-04-30 12:10", "A", "chat-meta-1", "OpenAI", "查看天气", true)
	if err != nil {
		t.Fatal(err)
	}
	metas := svc.GetMetas()
	if len(metas) != 1 {
		t.Fatalf("expected 1 meta, got %d", len(metas))
	}
	if metas[0].ChatID != "chat-meta-1" {
		t.Fatalf("meta.ChatID = %q, want chat-meta-1", metas[0].ChatID)
	}
}

func TestSubmitHourlyCreatesWindow(t *testing.T) {
	svc := newTestSvc(t)
	_, err := svc.Submit(3, "2026-04-30 12:10", "A", "", "OpenAI", "查看天气", true)
	if err != nil {
		t.Fatal(err)
	}
	details := svc.GetDetails()
	if len(details) < 24 {
		t.Fatalf("expected many hourly details, got %d", len(details))
	}
}

func TestSubmitEvery15CreatesWindow(t *testing.T) {
	svc := newTestSvc(t)
	_, err := svc.Submit(4, "2026-04-30 12:10", "A", "", "OpenAI", "查看天气", true)
	if err != nil {
		t.Fatal(err)
	}
	details := svc.GetDetails()
	if len(details) < 96 {
		t.Fatalf("expected many 15-min details, got %d", len(details))
	}
}

func TestSubmitWithCustomType(t *testing.T) {
	svc := newTestSvc(t)
	meta, err := svc.SubmitWithType(0, "2026-04-30 12:10", "A", "chat-001", "FEISHU", "OpenAI", "查看天气", true)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Type != "FEISHU" {
		t.Fatalf("meta.Type = %q, want FEISHU", meta.Type)
	}
	details := svc.GetDetails()
	if len(details) != 1 {
		t.Fatalf("expected 1 detail, got %d", len(details))
	}
	if details[0].Type != "FEISHU" {
		t.Fatalf("detail.Type = %q, want FEISHU", details[0].Type)
	}
}

func TestResponseSchemaPersistsAndPropagates(t *testing.T) {
	svc := newTestSvc(t)
	schema := `{"type":"object","properties":{"answer":{"type":"string"}}}`
	meta, err := svc.SubmitWithTypeAndSchema(0, "2026-04-30 12:10", "A", "chat-001", "FEISHU", "OpenAI", "查看天气", schema, true)
	if err != nil {
		t.Fatal(err)
	}
	if meta.ResponseSchema != schema {
		t.Fatalf("meta.ResponseSchema = %q, want %q", meta.ResponseSchema, schema)
	}
	metas := svc.GetMetas()
	if len(metas) != 1 || metas[0].ResponseSchema != schema {
		t.Fatalf("metas response schema mismatch: %+v", metas)
	}
	details := svc.GetDetails()
	if len(details) != 1 {
		t.Fatalf("expected 1 detail, got %d", len(details))
	}
	if details[0].ResponseSchema != schema {
		t.Fatalf("detail.ResponseSchema = %q, want %q", details[0].ResponseSchema, schema)
	}
}

func TestRouterDisablePersistsAndPropagates(t *testing.T) {
	svc := newTestSvc(t)
	meta, err := svc.SubmitWithTypeSchemaAndRouterDisable(0, "2026-04-30 12:10", "A", "chat-001", "FEISHU", "OpenAI", "查看天气", "", false, true)
	if err != nil {
		t.Fatal(err)
	}
	if meta.RouterDisable {
		t.Fatalf("meta.RouterDisable = true, want false")
	}
	metas := svc.GetMetas()
	if len(metas) != 1 || metas[0].RouterDisable {
		t.Fatalf("metas router_disable mismatch: %+v", metas)
	}
	details := svc.GetDetails()
	if len(details) != 1 {
		t.Fatalf("expected 1 detail, got %d", len(details))
	}
	if details[0].RouterDisable {
		t.Fatalf("detail.RouterDisable = true, want false")
	}
}

func TestCustomCronRouterDisablePersistsAfterPeriodicSplit(t *testing.T) {
	svc := newTestSvc(t)
	meta, err := svc.SubmitCronWithTypeSchemaAndRouterDisable("* * * * *", "A", "chat-001", "FEISHU", "OpenAI", "查看天气", "", false, true)
	if err != nil {
		t.Fatal(err)
	}
	if meta.RouterDisable {
		t.Fatalf("meta.RouterDisable = true, want false")
	}
	svc.CheckAndCreate()
	details := svc.GetDetails()
	if len(details) == 0 {
		t.Fatal("expected details after periodic split")
	}
	if details[0].RouterDisable {
		t.Fatalf("detail.RouterDisable = true, want false")
	}
}

func TestParseArgsSupportsRouterDisable(t *testing.T) {
	routerDisable := true
	thinking := false
	if err := parseArgs([]string{"--router_disable", "false", "--thinking", "false"}, nil, nil, nil, nil, nil, nil, nil, nil, &thinking, &routerDisable, nil, nil); err != nil {
		t.Fatal(err)
	}
	if routerDisable {
		t.Fatalf("routerDisable = true, want false")
	}
	if thinking {
		t.Fatalf("thinking = true, want false")
	}
}

func TestSchemaMigrationAddsTaskTypeAndRebuildsIndex(t *testing.T) {
	path := t.TempDir() + "/cron_migration.db"
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		CREATE TABLE task_meta (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cycle INTEGER NOT NULL,
			raw_time TEXT NOT NULL DEFAULT '',
			agent_id TEXT NOT NULL,
			chat_id TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL,
			thinking INTEGER NOT NULL DEFAULT 0,
			cron TEXT NOT NULL,
			content TEXT NOT NULL
		);
		CREATE TABLE task_detail (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			meta_id INTEGER NOT NULL,
			exec_time INTEGER NOT NULL,
			agent_id TEXT NOT NULL,
			chat_id TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL,
			thinking INTEGER NOT NULL DEFAULT 0,
			content TEXT NOT NULL,
			started INTEGER NOT NULL DEFAULT 0,
			UNIQUE(meta_id, exec_time)
		);
		CREATE INDEX idx_detail_agent_time ON task_detail(agent_id, exec_time);
	`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	db.Close()

	svc, err := NewCronService(path)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.db.Close()

	var taskType string
	if err := svc.db.QueryRow(`SELECT name FROM pragma_table_info('task_detail') WHERE name = 'task_type'`).Scan(&taskType); err != nil {
		t.Fatalf("expected task_detail.task_type column: %v", err)
	}
	if taskType != "task_type" {
		t.Fatalf("pragma task_type column name = %q, want task_type", taskType)
	}

	var metaRouterDisableColumn string
	if err := svc.db.QueryRow(`SELECT name FROM pragma_table_info('task_meta') WHERE name = 'router_disable'`).Scan(&metaRouterDisableColumn); err != nil {
		t.Fatalf("expected task_meta.router_disable column: %v", err)
	}
	if metaRouterDisableColumn != "router_disable" {
		t.Fatalf("pragma task_meta router_disable column name = %q, want router_disable", metaRouterDisableColumn)
	}

	var detailRouterDisableColumn string
	if err := svc.db.QueryRow(`SELECT name FROM pragma_table_info('task_detail') WHERE name = 'router_disable'`).Scan(&detailRouterDisableColumn); err != nil {
		t.Fatalf("expected task_detail.router_disable column: %v", err)
	}
	if detailRouterDisableColumn != "router_disable" {
		t.Fatalf("pragma task_detail router_disable column name = %q, want router_disable", detailRouterDisableColumn)
	}

	var oldIndexCount int
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_detail_agent_time'`).Scan(&oldIndexCount); err != nil {
		t.Fatal(err)
	}
	if oldIndexCount != 0 {
		t.Fatalf("old index still exists: %d", oldIndexCount)
	}

	var agentStatusIndexCount int
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_detail_agent_exec_status'`).Scan(&agentStatusIndexCount); err != nil {
		t.Fatal(err)
	}
	if agentStatusIndexCount != 1 {
		t.Fatalf("agent status index count = %d, want 1", agentStatusIndexCount)
	}

	var execStatusIndexCount int
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_detail_exec_status'`).Scan(&execStatusIndexCount); err != nil {
		t.Fatal(err)
	}
	if execStatusIndexCount != 1 {
		t.Fatalf("exec status index count = %d, want 1", execStatusIndexCount)
	}
}
