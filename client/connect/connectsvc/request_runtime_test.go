package connectsvc

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func newCompletedReplyTestService(t *testing.T) *Service {
	t.Helper()
	svc, err := NewService(Options{
		DBPath:   filepath.Join(t.TempDir(), "connect.db"),
		AgentDir: t.TempDir(),
		CacheTTL: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new connect service: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })
	if _, err := svc.db.Exec(`
		CREATE TABLE task_detail (
			id INTEGER PRIMARY KEY,
			task_type TEXT NOT NULL,
			meta_ref TEXT NOT NULL DEFAULT '',
			result_content TEXT NOT NULL DEFAULT '',
			replied_at TEXT NOT NULL DEFAULT '',
			reply_state TEXT NOT NULL DEFAULT '',
			reply_uuid TEXT NOT NULL DEFAULT '',
			reply_started_at INTEGER NOT NULL DEFAULT 0,
			started INTEGER NOT NULL DEFAULT 0,
			exec_time INTEGER NOT NULL DEFAULT 0
		)
	`); err != nil {
		t.Fatalf("create task_detail: %v", err)
	}
	return svc
}

func insertCompletedReplyTestRequest(t *testing.T, svc *Service, status int) int {
	t.Helper()
	result, err := svc.db.Exec(`
		INSERT INTO connect_request (name, request, artifacts, raw_request, response_schema, status, created_at)
		VALUES ('feishu', 'question', '', '', '', ?, ?)
	`, status, time.Now().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("insert request: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("request id: %v", err)
	}
	return int(id)
}

func insertCompletedReplyTestDetail(t *testing.T, svc *Service, id, requestID int) {
	t.Helper()
	if _, err := svc.db.Exec(`
		INSERT INTO task_detail (id, task_type, meta_ref, result_content, reply_state, started, exec_time)
		VALUES (?, 'feishu', ?, 'answer', 'pending', 3, ?)
	`, id, requestID, time.Now().Unix()); err != nil {
		t.Fatalf("insert detail: %v", err)
	}
}

func completedReplyDispatchOptions(svc *Service, send func(map[string]string) error) CompletedReplyDispatchOptions {
	return CompletedReplyDispatchOptions{
		QueryDB:         svc.db,
		Service:         svc,
		DefaultTaskType: "cron",
		SinceUnix:       0,
		ResolveCallback: func(string) (string, error) { return "feishu", nil },
		SendReply: func(_ string, flags map[string]string) error {
			return send(flags)
		},
		BuildFlags: func(string, Request, string) (map[string]string, error) {
			return map[string]string{"content": "answer"}, nil
		},
		MarkDetailReplied: func(detailID int) error {
			_, err := svc.db.Exec(`
				UPDATE task_detail
				SET replied_at = ?, reply_state = 'sent'
				WHERE id = ? AND reply_state = 'sending'
			`, time.Now().Format(time.RFC3339), detailID)
			return err
		},
	}
}

func completedReplyState(t *testing.T, svc *Service, detailID int) string {
	t.Helper()
	var state string
	if err := svc.db.QueryRow(`SELECT reply_state FROM task_detail WHERE id = ?`, detailID).Scan(&state); err != nil {
		t.Fatalf("read detail state: %v", err)
	}
	return state
}

func TestDispatchCompletedRepliesSendsDetailOnlyOnce(t *testing.T) {
	svc := newCompletedReplyTestService(t)
	requestID := insertCompletedReplyTestRequest(t, svc, RequestStatusStarted)
	insertCompletedReplyTestDetail(t, svc, 41, requestID)

	calls := 0
	DispatchCompletedReplies(completedReplyDispatchOptions(svc, func(flags map[string]string) error {
		calls++
		if got, want := flags["idempotency-key"], completedReplyUUID(41, "feishu"); got != want {
			t.Fatalf("idempotency key = %q, want %q", got, want)
		}
		return nil
	}))
	DispatchCompletedReplies(completedReplyDispatchOptions(svc, func(map[string]string) error {
		calls++
		return nil
	}))
	if calls != 1 {
		t.Fatalf("send calls = %d, want 1", calls)
	}
	if got := completedReplyState(t, svc, 41); got != "sent" {
		t.Fatalf("reply state = %q, want sent", got)
	}
}

func TestDispatchCompletedRepliesQuarantinesUnknownResults(t *testing.T) {
	tests := []struct {
		name string
		send func(t *testing.T, svc *Service, requestID int) func(map[string]string) error
	}{
		{
			name: "plugin send failure",
			send: func(*testing.T, *Service, int) func(map[string]string) error {
				return func(map[string]string) error { return errors.New("send timeout") }
			},
		},
		{
			name: "request status write failure after send",
			send: func(t *testing.T, svc *Service, requestID int) func(map[string]string) error {
				return func(map[string]string) error {
					if _, err := svc.db.Exec(`UPDATE connect_request SET status = ? WHERE id = ?`, RequestStatusCompleted, requestID); err != nil {
						t.Fatalf("race request status: %v", err)
					}
					return nil
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newCompletedReplyTestService(t)
			requestID := insertCompletedReplyTestRequest(t, svc, RequestStatusStarted)
			insertCompletedReplyTestDetail(t, svc, 42, requestID)
			calls := 0
			send := tt.send(t, svc, requestID)
			opts := completedReplyDispatchOptions(svc, func(flags map[string]string) error {
				calls++
				return send(flags)
			})
			DispatchCompletedReplies(opts)
			DispatchCompletedReplies(opts)
			if calls != 1 {
				t.Fatalf("send calls = %d, want 1", calls)
			}
			if got := completedReplyState(t, svc, 42); got != "unknown" {
				t.Fatalf("reply state = %q, want unknown", got)
			}
		})
	}
}

func TestDispatchCompletedRepliesClaimsOneSenderAcrossConcurrentScans(t *testing.T) {
	svc := newCompletedReplyTestService(t)
	requestID := insertCompletedReplyTestRequest(t, svc, RequestStatusStarted)
	insertCompletedReplyTestDetail(t, svc, 43, requestID)

	var mu sync.Mutex
	calls := 0
	opts := completedReplyDispatchOptions(svc, func(map[string]string) error {
		mu.Lock()
		calls++
		mu.Unlock()
		return nil
	})
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			DispatchCompletedReplies(opts)
		}()
	}
	wg.Wait()
	if calls != 1 {
		t.Fatalf("concurrent send calls = %d, want 1", calls)
	}
	if got := completedReplyState(t, svc, 43); got != "sent" {
		t.Fatalf("reply state = %q, want sent", got)
	}
}

func TestCompletedReplyUUIDDiffersForDifferentDetails(t *testing.T) {
	if first, second := completedReplyUUID(51, "feishu"), completedReplyUUID(52, "feishu"); first == second || first == "" || second == "" {
		t.Fatalf("detail UUIDs must be non-empty and unique: %q, %q", first, second)
	}
}
