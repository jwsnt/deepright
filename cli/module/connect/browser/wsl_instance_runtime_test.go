package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserWSLInstanceReserveProfileDirCreatesEmptyDirectory(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	root := t.TempDir()
	profileWin, profileUnix, err := browserWSLInstanceReserveProfileDirAt(root, `C:\ProgramData\deepright`)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(profileUnix) != root {
		t.Fatalf("profile directory = %q, want child of %q", profileUnix, root)
	}
	entries, err := os.ReadDir(profileUnix)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("new WSL profile must be empty, got %d entries", len(entries))
	}
	if filepath.Base(profileWin) == "" {
		t.Fatalf("profile directory is empty: %q", profileWin)
	}
}
