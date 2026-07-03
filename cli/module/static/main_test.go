package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"static-server/server"
)

func TestStaticFileServing(t *testing.T) {
	mux := http.NewServeMux()
	if err := server.Register(mux, "test-case"); err != nil {
		t.Fatalf("Register: %v", err)
	}

	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		path string
		want string
	}{
		{"/site/hello.html", "HELLO WORLD"},
		{"/site/js/hello.js", "1+1"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			resp, err := http.Get(server.URL + tt.path)
			if err != nil {
				t.Fatalf("GET %s: %v", tt.path, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				t.Fatalf("GET %s: status %d", tt.path, resp.StatusCode)
			}

			body, _ := io.ReadAll(resp.Body)
			content := strings.TrimSpace(string(body))
			if content != tt.want {
				t.Errorf("GET %s: body = %q, want %q", tt.path, content, tt.want)
			}
		})
	}
}
