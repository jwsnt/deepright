package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeDirectoryPathAcceptsQuotedPath(t *testing.T) {
	root := t.TempDir()
	allowedDir := filepath.Join(root, "picked")
	if err := os.MkdirAll(allowedDir, 0o755); err != nil {
		t.Fatalf("mkdir picked: %v", err)
	}

	got, err := normalizeDirectoryPath(`"` + allowedDir + `"`)
	if err != nil {
		t.Fatalf("normalizeDirectoryPath quoted: %v", err)
	}
	if got != filepath.Clean(allowedDir) {
		t.Fatalf("path = %q, want %q", got, filepath.Clean(allowedDir))
	}
}

func TestDefaultMacPickerDirectoryUsesDocuments(t *testing.T) {
	home := t.TempDir()
	documentsDir := filepath.Join(home, "Documents")
	if err := os.MkdirAll(documentsDir, 0o755); err != nil {
		t.Fatalf("mkdir Documents: %v", err)
	}
	t.Setenv("HOME", home)

	got := defaultMacPickerDirectory()
	if got != filepath.Clean(documentsDir) {
		t.Fatalf("defaultMacPickerDirectory = %q, want %q", got, filepath.Clean(documentsDir))
	}
}
