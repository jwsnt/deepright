package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRegisterSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "test.html"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	if err := Register(mux, tmpDir); err != nil {
		t.Fatalf("Register() returned error: %v", err)
	}
}

func TestRegisterInvalidPath(t *testing.T) {
	mux := http.NewServeMux()
	if err := Register(mux, "/nonexistent/path/that/does/not/exist"); err == nil {
		t.Fatal("Register() expected error for invalid path, got nil")
	}
}

func TestRegisterFileNotDir(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(tmpFile, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	if err := Register(mux, tmpFile); err == nil {
		t.Fatal("Register() expected error for a file path, got nil")
	}
}

func TestRegisterServesSitePrefix(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "hello.txt"), []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	if err := Register(mux, tmpDir); err != nil {
		t.Fatalf("Register() returned error: %v", err)
	}

	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/site/hello.txt")
	if err != nil {
		t.Fatalf("GET /site/hello.txt: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /site/hello.txt status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
