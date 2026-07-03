package standalone

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStateSetAndReset(t *testing.T) {
	state := New(false)

	initial := state.Snapshot()
	if initial.Enabled || initial.StartupEnabled || initial.Overridden {
		t.Fatalf("unexpected initial snapshot: %#v", initial)
	}

	enabled := state.Set(true)
	if !enabled.Enabled || !enabled.Overridden {
		t.Fatalf("unexpected enabled snapshot: %#v", enabled)
	}

	disabled := state.Set(false)
	if disabled.Enabled || !disabled.Overridden {
		t.Fatalf("unexpected disabled snapshot: %#v", disabled)
	}

	reset := state.Reset()
	if reset.Enabled || reset.Overridden {
		t.Fatalf("unexpected reset snapshot: %#v", reset)
	}
}

func TestClientSetAndReset(t *testing.T) {
	var seenMethod string
	var seenPath string
	var seenBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenMethod = r.Method
		seenPath = r.URL.Path
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		seenBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": 0,
				"data": Snapshot{
					Enabled:        false,
					StartupEnabled: false,
					Overridden:     false,
					RuntimeOnly:    true,
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": 0,
			"data": Snapshot{
				Enabled:        true,
				StartupEnabled: false,
				Overridden:     true,
				RuntimeOnly:    true,
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	snapshot, err := client.Set(context.Background(), true)
	if err != nil {
		t.Fatalf("client set: %v", err)
	}
	if seenMethod != http.MethodPost || seenPath != Endpoint+"=true" || !strings.Contains(seenBody, `"enabled":true`) {
		t.Fatalf("unexpected set request: method=%s path=%s body=%s", seenMethod, seenPath, seenBody)
	}
	if !snapshot.Enabled || !snapshot.Overridden {
		t.Fatalf("unexpected set snapshot: %#v", snapshot)
	}

	snapshot, err = client.Reset(context.Background())
	if err != nil {
		t.Fatalf("client reset: %v", err)
	}
	if seenMethod != http.MethodDelete || seenPath != Endpoint {
		t.Fatalf("unexpected reset request: method=%s path=%s", seenMethod, seenPath)
	}
	if snapshot.Enabled || snapshot.Overridden {
		t.Fatalf("unexpected reset snapshot: %#v", snapshot)
	}
}
