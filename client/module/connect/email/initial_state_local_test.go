package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connect/connectsvc"
)

type stubLocalMetaGetter struct {
	meta  *connectsvc.Meta
	err   error
	calls int
}

func (s *stubLocalMetaGetter) GetMeta(_ context.Context, _ string) (*connectsvc.Meta, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	if s.meta == nil {
		return nil, nil
	}
	copyMeta := *s.meta
	return &copyMeta, nil
}

func TestEnsureInitialScanStateLocalUsesPOP3IntervalTimesTwo(t *testing.T) {
	now := time.Date(2026, 6, 12, 18, 0, 0, 0, time.FixedZone("CST", 8*3600))
	stateFile := filepath.Join(t.TempDir(), "email.state.json")
	getter := &stubLocalMetaGetter{
		meta: &connectsvc.Meta{
			Key:  "email",
			Meta: `{"email_pop3_interval":"420"}`,
		},
	}

	if err := ensureInitialScanStateLocal(context.Background(), getter, "email", stateFile, func() time.Time { return now }); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	var state localInitialState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("decode state: %v raw=%s", err, string(data))
	}
	want := now.Add(-840 * time.Second).Unix()
	if state.LastTimelineUnix != want {
		t.Fatalf("lastTimelineUnix = %d, want %d", state.LastTimelineUnix, want)
	}
	if len(state.Processed) != 0 {
		t.Fatalf("processed = %+v, want empty", state.Processed)
	}
	if getter.calls != 1 {
		t.Fatalf("getter calls = %d, want 1", getter.calls)
	}
}

func TestEnsureInitialScanStateLocalFallsBackToDefaultWindow(t *testing.T) {
	now := time.Date(2026, 6, 12, 18, 0, 0, 0, time.FixedZone("CST", 8*3600))
	stateFile := filepath.Join(t.TempDir(), "email.state.json")
	getter := &stubLocalMetaGetter{
		meta: &connectsvc.Meta{
			Key:  "email",
			Meta: `{"email":"demo@example.com"}`,
		},
	}

	if err := ensureInitialScanStateLocal(context.Background(), getter, "email", stateFile, func() time.Time { return now }); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	var state localInitialState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("decode state: %v raw=%s", err, string(data))
	}
	want := now.Add(-time.Duration(defaultPOP3ScanSecsLocal*2) * time.Second).Unix()
	if state.LastTimelineUnix != want {
		t.Fatalf("lastTimelineUnix = %d, want %d", state.LastTimelineUnix, want)
	}
}

func TestEnsureInitialScanStateLocalDoesNotOverwriteExistingState(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "email.state.json")
	original := `{"lastTimelineUnix":123,"processed":{"keep":456}}`
	if err := os.WriteFile(stateFile, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	getter := &stubLocalMetaGetter{
		meta: &connectsvc.Meta{
			Key:  "email",
			Meta: `{"email_pop3_interval":"420"}`,
		},
	}

	if err := ensureInitialScanStateLocal(context.Background(), getter, "email", stateFile, time.Now); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Fatalf("state file overwritten: %s", string(data))
	}
	if getter.calls != 0 {
		t.Fatalf("getter calls = %d, want 0", getter.calls)
	}
}
