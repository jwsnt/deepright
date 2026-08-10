package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"knowledge/knowledgecore"
)

func TestUpdateTimeAcceptsPositionalTimestamp(t *testing.T) {
	appDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(appDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWD) }()
	defer func() { _ = knowledgecore.CloseSharedDB() }()

	stdout := os.Stdout
	stderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = stdout
		os.Stderr = stderr
	}()

	runUpdateTime([]string{"1715337600000"})

	_ = w.Close()
	var out bytes.Buffer
	if _, err := io.Copy(&out, r); err != nil {
		t.Fatalf("copy output: %v", err)
	}

	db, err := knowledgecore.OpenSharedDB(appDir)
	if err != nil {
		t.Fatalf("OpenSharedDB: %v", err)
	}
	lastUpdate, err := knowledgecore.GetLastUpdate(db)
	if err != nil {
		t.Fatalf("GetLastUpdate: %v", err)
	}
	if lastUpdate != 1715337600000 {
		t.Fatalf("lastUpdate = %d, want 1715337600000", lastUpdate)
	}
	if !strings.Contains(out.String(), filepath.Join(appDir, "knowledge")) {
		t.Fatalf("output = %q, want knowledge path", out.String())
	}
}
