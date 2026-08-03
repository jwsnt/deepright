package main

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

func TestParseModelTaskConcurrence(t *testing.T) {
	concurrence, err := parseModelTaskConcurrence(map[string]interface{}{
		"modelTask": map[string]interface{}{"concurrence": 2},
	})
	if err != nil || concurrence != 2 {
		t.Fatalf("parse concurrence = %d, %v", concurrence, err)
	}
	for _, raw := range []map[string]interface{}{
		{"modelTask": map[string]interface{}{}},
		{"modelTask": map[string]interface{}{"concurrence": 0}},
		{"modelTask": map[string]interface{}{"concurrence": "1"}},
		{"modelTask": 1},
	} {
		if _, err := parseModelTaskConcurrence(raw); err == nil {
			t.Fatalf("invalid modelTask config accepted: %#v", raw)
		}
	}
}

func TestSharedModelTaskQueueKeepsOtherTaskQueuedUntilSlotIsAvailable(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	if err := ensureWhisperTaskSchema(db); err != nil {
		t.Fatalf("ensure whisper schema: %v", err)
	}
	if err := ensureRembgTaskSchema(db); err != nil {
		t.Fatalf("ensure rembg schema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO whisper_task (agent_id, source_path, output_path, status, created_at, updated_at) VALUES ('agent-a', 'tmp/a.wav', 'whisper/a.txt', 'queued', 'now', 'now')`); err != nil {
		t.Fatalf("insert whisper task: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO rembg_task (agent_id, source_path, output_path, status, created_at, updated_at) VALUES ('agent-a', 'tmp/a.png', 'images/a.png', 'queued', 'now', 'now')`); err != nil {
		t.Fatalf("insert rembg task: %v", err)
	}
	cfg := &Config{modelTaskQueue: newModelTaskQueue(1)}
	held, err := reserveModelTaskSlot(context.Background(), cfg, db, "whisper_task")
	if err != nil || !held {
		t.Fatalf("reserve whisper slot = %t, %v", held, err)
	}
	secondDone := make(chan struct{})
	var secondHeld bool
	var secondErr error
	go func() {
		secondHeld, secondErr = reserveModelTaskSlot(context.Background(), cfg, db, "rembg_task")
		close(secondDone)
	}()
	select {
	case <-secondDone:
		t.Fatal("second task acquired the only shared slot")
	case <-time.After(40 * time.Millisecond):
	}
	var status string
	if err := db.QueryRow(`SELECT status FROM rembg_task LIMIT 1`).Scan(&status); err != nil {
		t.Fatalf("read waiting task: %v", err)
	}
	if status != rembgTaskQueued {
		t.Fatalf("waiting task status = %q, want %q", status, rembgTaskQueued)
	}
	releaseModelTaskSlot(cfg)
	select {
	case <-secondDone:
	case <-time.After(time.Second):
		t.Fatal("second task did not acquire released shared slot")
	}
	if secondErr != nil || !secondHeld {
		t.Fatalf("reserve rembg slot = %t, %v", secondHeld, secondErr)
	}
	releaseModelTaskSlot(cfg)
}
