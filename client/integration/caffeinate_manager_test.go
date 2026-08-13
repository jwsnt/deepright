package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

func drainIntegrationSSEActivitySignals() {
	for {
		select {
		case <-integrationSSEActivity.changed:
		default:
			return
		}
	}
}

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

func TestIntegrationCaffeinatePendingTaskUsesQueuedAndRunningMediaTasks(t *testing.T) {
	cases := []struct {
		name   string
		table  string
		status string
		active bool
	}{
		{name: "wav2lip queued", table: "wav2lip", status: wav2lipTaskQueued, active: true},
		{name: "wav2lip running", table: "wav2lip", status: wav2lipTaskRunning, active: true},
		{name: "rembg queued", table: "rembg", status: rembgTaskQueued, active: true},
		{name: "rembg running", table: "rembg", status: rembgTaskRunning, active: true},
		{name: "voxcpm queued", table: "voxcpm", status: voxcpmTaskQueued, active: true},
		{name: "voxcpm running", table: "voxcpm", status: voxcpmTaskRunning, active: true},
		{name: "rvm queued", table: "rvm", status: rvmTaskQueued, active: true},
		{name: "rvm running", table: "rvm", status: rvmTaskRunning, active: true},
		{name: "terminal task", table: "wav2lip", status: wav2lipTaskCompleted, active: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			for _, ensure := range []func(*sql.DB) error{ensureWav2LipTaskSchema, ensureRembgTaskSchema, ensureVoxCPMTaskSchema, ensureRVMTaskSchema, ensureSAM2TaskSchema} {
				if err := ensure(db); err != nil {
					t.Fatal(err)
				}
			}

			oldCronDB := cronDB
			cronDB = db
			t.Cleanup(func() { cronDB = oldCronDB })
			now := "2026-08-02T10:00:00Z"
			switch tc.table {
			case "wav2lip":
				_, err = db.Exec(`INSERT INTO wav2lip_task(agent_id,video_path,audio_path,status,created_at,updated_at) VALUES ('agent-a','video.mp4','audio.wav',?,?,?)`, tc.status, now, now)
			case "rembg":
				_, err = db.Exec(`INSERT INTO rembg_task(agent_id,source_path,status,created_at,updated_at) VALUES ('agent-a','photo.png',?,?,?)`, tc.status, now, now)
			case "voxcpm":
				_, err = db.Exec(`INSERT INTO voxcpm_task(agent_id,text_path,text_saved_as,reference_audio_path,reference_audio_saved_as,requested_name,output_path,saved_as,status,created_at,updated_at) VALUES ('agent-a','text.txt','/text.txt','','','voice','voice.wav','/voice.wav',?,?,?)`, tc.status, now, now)
			case "rvm":
				_, err = db.Exec(`INSERT INTO rvm_task(agent_id,source_path,status,created_at,updated_at) VALUES ('agent-a','video.mp4',?,?,?)`, tc.status, now, now)
			default:
				t.Fatalf("unsupported task table %q", tc.table)
			}
			if err != nil {
				t.Fatal(err)
			}

			active, err := integrationCaffeinatePendingTask()
			if err != nil {
				t.Fatal(err)
			}
			if active != tc.active {
				t.Fatalf("active = %t, want %t", active, tc.active)
			}
		})
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

func TestTrackIntegrationSSEWakesIdleHeartbeatAndIsIdempotent(t *testing.T) {
	previous := integrationSSEActivity.active.Swap(0)
	drainIntegrationSSEActivitySignals()
	t.Cleanup(func() {
		integrationSSEActivity.active.Store(previous)
		drainIntegrationSSEActivitySignals()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	waited := make(chan bool, 1)
	go func() {
		waited <- waitForCliGetHeartbeat(ctx, time.Minute)
	}()

	finish := trackIntegrationSSE(nil)
	select {
	case ok := <-waited:
		if !ok {
			t.Fatal("idle heartbeat wait stopped unexpectedly")
		}
	case <-time.After(time.Second):
		t.Fatal("active SSE did not wake idle heartbeat wait")
	}
	if !integrationHasActiveSSE() {
		t.Fatal("expected active SSE after tracking starts")
	}

	finish()
	finish()
	if integrationHasActiveSSE() {
		t.Fatal("expected SSE count to return to zero after cleanup")
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
