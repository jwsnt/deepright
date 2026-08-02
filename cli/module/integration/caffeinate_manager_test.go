package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

type integrationSleepAssertionStub struct {
	mu      sync.Mutex
	running bool
	stops   int
	stopErr error
}

func (s *integrationSleepAssertionStub) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stops++
	s.running = false
	return s.stopErr
}

func (s *integrationSleepAssertionStub) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *integrationSleepAssertionStub) stopCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stops
}

func newIntegrationSleepManagerForTest(conditions *integrationSleepConditions, assertion *integrationSleepAssertionStub) *integrationSleepManager {
	manager := &integrationSleepManager{
		interval: time.Minute,
		evaluateConditions: func(activeSSE int) (integrationSleepConditions, error) {
			current := *conditions
			current.SSE = current.SSE || activeSSE > 0
			return current, nil
		},
		startAssertion: func() (integrationSleepAssertion, error) {
			assertion.running = true
			return assertion, nil
		},
		logf:           func(string, ...interface{}) {},
		evaluateSignal: make(chan struct{}, 1),
	}
	manager.evaluateNow = manager.evaluate
	return manager
}

func TestParseIntegrationCaffeinateInterval(t *testing.T) {
	cases := []struct {
		name string
		raw  map[string]interface{}
		want time.Duration
		err  bool
	}{
		{name: "valid", raw: map[string]interface{}{"caffeinate": json.Number("15")}, want: 15 * time.Minute},
		{name: "missing", raw: map[string]interface{}{}, err: true},
		{name: "fraction", raw: map[string]interface{}{"caffeinate": json.Number("1.5")}, err: true},
		{name: "zero", raw: map[string]interface{}{"caffeinate": json.Number("0")}, err: true},
		{name: "text", raw: map[string]interface{}{"caffeinate": "15"}, err: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseIntegrationCaffeinateInterval(tc.raw)
			if tc.err {
				if err == nil {
					t.Fatal("expected validation error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("interval = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestIntegrationCaffeinatePendingMemoUsesFuture24HourWindow(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	oldCronDB := cronDB
	cronDB = db
	t.Cleanup(func() { cronDB = oldCronDB })
	ensureCronSchema(db)

	meta, err := db.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, model, cron, content) VALUES (0, '', 'agent-a', 'openai', '', 'memo')`)
	if err != nil {
		t.Fatal(err)
	}
	metaID, err := meta.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	if _, err := db.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, task_type, model, content, started) VALUES (?, ?, 'agent-a', 'cron', 'openai', 'future memo', 0)`, metaID, now.Add(2*time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}

	pending, err := integrationCaffeinatePendingMemo(now)
	if err != nil {
		t.Fatal(err)
	}
	if !pending {
		t.Fatal("expected a pending memo in the next 24 hours")
	}

	if _, err := db.Exec(`UPDATE task_detail SET started = 3`); err != nil {
		t.Fatal(err)
	}
	pending, err = integrationCaffeinatePendingMemo(now)
	if err != nil {
		t.Fatal(err)
	}
	if pending {
		t.Fatal("completed memo must not keep the sleep assertion")
	}
}

func TestIntegrationSleepManagerStartsAndStopsForConditionChanges(t *testing.T) {
	conditions := integrationSleepConditions{Memo: true}
	assertion := &integrationSleepAssertionStub{}
	manager := newIntegrationSleepManagerForTest(&conditions, assertion)

	manager.evaluate()
	if !assertion.Running() {
		t.Fatal("expected assertion to start for pending memo")
	}

	conditions = integrationSleepConditions{}
	manager.evaluate()
	if assertion.Running() {
		t.Fatal("expected assertion to stop after every condition clears")
	}
	if got := assertion.stopCount(); got != 1 {
		t.Fatalf("stop count = %d, want 1", got)
	}
}

func TestIntegrationSleepManagerTracksSSEUntilAbnormalTermination(t *testing.T) {
	conditions := integrationSleepConditions{}
	assertion := &integrationSleepAssertionStub{}
	manager := newIntegrationSleepManagerForTest(&conditions, assertion)
	manager.started = true

	finish := manager.TrackSSE()
	manager.evaluate()
	if !assertion.Running() {
		t.Fatal("expected assertion to start while SSE is active")
	}

	finish() // The same cleanup path is used for EOF, upstream errors, and client disconnects.
	finish()
	manager.evaluate()
	if assertion.Running() {
		t.Fatal("expected assertion to stop when the SSE terminates")
	}
	if got := assertion.stopCount(); got != 1 {
		t.Fatalf("stop count = %d, want one idempotent stop", got)
	}
}

func TestIntegrationSleepManagerCloseContinuesWhenAssertionStopFails(t *testing.T) {
	conditions := integrationSleepConditions{Plugin: true}
	assertion := &integrationSleepAssertionStub{stopErr: errors.New("kill failed")}
	manager := newIntegrationSleepManagerForTest(&conditions, assertion)
	manager.evaluate()

	manager.Close()
	if got := assertion.stopCount(); got != 1 {
		t.Fatalf("stop count = %d, want 1", got)
	}
	manager.mu.Lock()
	closed := manager.closed
	manager.mu.Unlock()
	if !closed {
		t.Fatal("manager should remain closed after stop failure")
	}
}

func TestIntegrationSleepManagerDoesNotDuplicateAssertionAfterStopFailure(t *testing.T) {
	conditions := integrationSleepConditions{Memo: true}
	assertion := &integrationSleepAssertionStub{stopErr: errors.New("kill failed")}
	manager := newIntegrationSleepManagerForTest(&conditions, assertion)
	starts := 0
	manager.startAssertion = func() (integrationSleepAssertion, error) {
		starts++
		assertion.running = true
		return assertion, nil
	}

	manager.evaluate()
	conditions = integrationSleepConditions{}
	manager.evaluate()
	conditions = integrationSleepConditions{Plugin: true}
	manager.evaluate()

	if starts != 1 {
		t.Fatalf("assertion starts = %d, want 1 after stop failure", starts)
	}
}
