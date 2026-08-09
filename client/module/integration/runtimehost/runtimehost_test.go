package runtimehost

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
	state := New("https://www.deepright.cn")

	initial := state.Snapshot()
	if initial.Host != "https://www.deepright.cn" || initial.StartupHost != "https://www.deepright.cn" || initial.Overridden {
		t.Fatalf("unexpected initial snapshot: %#v", initial)
	}

	updated, err := state.Set(" https://staging.deepright.cn/base/ ")
	if err != nil {
		t.Fatalf("set runtime host: %v", err)
	}
	if updated.Host != "https://staging.deepright.cn/base" {
		t.Fatalf("updated host = %q, want %q", updated.Host, "https://staging.deepright.cn/base")
	}
	if !updated.Overridden {
		t.Fatalf("updated snapshot should be overridden: %#v", updated)
	}

	reset := state.Reset()
	if reset.Host != "https://www.deepright.cn" || reset.Overridden {
		t.Fatalf("unexpected reset snapshot: %#v", reset)
	}
}

func TestValidateRejectsInvalidURL(t *testing.T) {
	invalid := []string{
		"",
		"deepright.cn",
		"ftp://deepright.cn",
		"https://www.deepright.cn?x=1",
		"https://",
	}
	for _, input := range invalid {
		if _, err := Validate(input); err == nil {
			t.Fatalf("Validate(%q) expected error", input)
		}
	}
}

func TestClientSetAndReset(t *testing.T) {
	var seenMethod string
	var seenBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != Endpoint {
			t.Fatalf("path = %q, want %q", r.URL.Path, Endpoint)
		}
		seenMethod = r.Method
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
					Host:        "https://www.deepright.cn",
					StartupHost: "https://www.deepright.cn",
					Overridden:  false,
					RuntimeOnly: true,
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": 0,
			"data": Snapshot{
				Host:        "https://staging.deepright.cn",
				StartupHost: "https://www.deepright.cn",
				Overridden:  true,
				RuntimeOnly: true,
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	snapshot, err := client.Set(context.Background(), "https://staging.deepright.cn")
	if err != nil {
		t.Fatalf("client set: %v", err)
	}
	if seenMethod != http.MethodPost || !strings.Contains(seenBody, "staging.deepright.cn") {
		t.Fatalf("unexpected set request: method=%s body=%s", seenMethod, seenBody)
	}
	if snapshot.Host != "https://staging.deepright.cn" || !snapshot.Overridden {
		t.Fatalf("unexpected set snapshot: %#v", snapshot)
	}

	snapshot, err = client.Reset(context.Background())
	if err != nil {
		t.Fatalf("client reset: %v", err)
	}
	if seenMethod != http.MethodDelete {
		t.Fatalf("reset method = %s, want DELETE", seenMethod)
	}
	if snapshot.Host != "https://www.deepright.cn" || snapshot.Overridden {
		t.Fatalf("unexpected reset snapshot: %#v", snapshot)
	}
}
