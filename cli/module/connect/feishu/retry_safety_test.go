package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connect/connectsvc"
)

func TestAcknowledgementRetryBudgetSurvivesRestart(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "feishu.pending.json")
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	messageLog := &bytes.Buffer{}
	attempts := 0
	acknowledger := localFeishuAcknowledgementSenderFunc(func(_ context.Context, _ connectsvc.Request, _ string) error {
		attempts++
		return errors.New("feishu unavailable")
	})

	newService := func() *localFeishuService {
		return &localFeishuService{
			name:         "feishu",
			logger:       log.New(io.Discard, "", 0),
			messageLog:   messageLog,
			stateFile:    stateFile,
			scanInterval: localFeishuScanInterval,
			expireWindow: localFeishuExpireWindow,
			nowFn:        func() time.Time { return now },
			acknowledger: acknowledger,
			state: localFeishuState{
				Pending:       map[string]localFeishuPendingMessage{},
				Pushed:        map[string]int64{"text-1": now.Unix()},
				Acknowledging: map[string]localFeishuPendingAcknowledgement{},
				Acknowledged:  map[string]int64{},
			},
		}
	}

	first := newService()
	first.state.Acknowledging["text-1"] = localFeishuPendingAcknowledgement{
		Request:        connectsvc.Request{ExternalID: "text-1"},
		ExternalIDs:    []string{"text-1"},
		StoredAt:       now.Unix(),
		IdempotencyKey: "feishu-ack-test",
	}
	if err := first.flushAcknowledgements(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("attempts after first flush = %d, want 1", attempts)
	}

	restarted := newService()
	if err := restarted.loadState(); err != nil {
		t.Fatal(err)
	}
	for range 4 {
		if err := restarted.flushAcknowledgements(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
	}

	if attempts != localFeishuMaxAcknowledgementAttempts {
		t.Fatalf("acknowledgement attempts = %d, want %d", attempts, localFeishuMaxAcknowledgementAttempts)
	}
	if len(restarted.state.Acknowledging) != 0 {
		t.Fatalf("acknowledgement should be terminal: %+v", restarted.state.Acknowledging)
	}
	failed, ok := restarted.state.AcknowledgementFailures["text-1"]
	if !ok {
		t.Fatalf("missing terminal acknowledgement state: %+v", restarted.state.AcknowledgementFailures)
	}
	if failed.Attempts != localFeishuMaxAcknowledgementAttempts {
		t.Fatalf("persisted attempts = %d, want %d", failed.Attempts, localFeishuMaxAcknowledgementAttempts)
	}
	if !strings.Contains(messageLog.String(), "stage=received-ack-terminate") {
		t.Fatalf("missing terminal retry log: %s", messageLog.String())
	}
}
