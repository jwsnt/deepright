package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestBrowserCleanupWSLProgramDataChromeDirsRemovesChromePrefixedDirectories(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	root := t.TempDir()
	browserWSLDetectFn = func() (bool, error) {
		return true, nil
	}
	browserReadDirFn = func(path string) ([]os.DirEntry, error) {
		if path != browserWSLInstanceProfileRoot {
			t.Fatalf("read dir path = %q, want %q", path, browserWSLInstanceProfileRoot)
		}
		return os.ReadDir(root)
	}

	for _, name := range []string{"chrome_def", "chrome_1234", "other"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	removed := []string{}
	browserRemoveAllFn = func(path string) error {
		removed = append(removed, filepath.Base(path))
		return os.RemoveAll(filepath.Join(root, filepath.Base(path)))
	}

	if err := browserCleanupWSLProgramDataChromeDirs(); err != nil {
		t.Fatal(err)
	}

	sort.Strings(removed)
	want := []string{"chrome_1234", "chrome_def"}
	if len(removed) != len(want) {
		t.Fatalf("removed = %v, want %v", removed, want)
	}
	for idx, item := range want {
		if removed[idx] != item {
			t.Fatalf("removed = %v, want %v", removed, want)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "other")); err != nil {
		t.Fatalf("other directory should stay: %v", err)
	}
}

func TestBrowserCleanupWSLProgramDataChromeDirsLogsFailuresButKeepsGoing(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	root := t.TempDir()
	browserWSLDetectFn = func() (bool, error) {
		return true, nil
	}
	browserReadDirFn = func(path string) ([]os.DirEntry, error) {
		if path != browserWSLInstanceProfileRoot {
			t.Fatalf("read dir path = %q, want %q", path, browserWSLInstanceProfileRoot)
		}
		return os.ReadDir(root)
	}

	for _, name := range []string{"chrome_a", "chrome_b"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	attempted := []string{}
	browserRemoveAllFn = func(path string) error {
		attempted = append(attempted, filepath.Base(path))
		if filepath.Base(path) == "chrome_a" {
			return fs.ErrPermission
		}
		return os.RemoveAll(filepath.Join(root, filepath.Base(path)))
	}

	if err := browserCleanupWSLProgramDataChromeDirs(); err != nil {
		t.Fatal(err)
	}

	sort.Strings(attempted)
	want := []string{"chrome_a", "chrome_b"}
	if len(attempted) != len(want) {
		t.Fatalf("attempted = %v, want %v", attempted, want)
	}
	for idx, item := range want {
		if attempted[idx] != item {
			t.Fatalf("attempted = %v, want %v", attempted, want)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "chrome_a")); err != nil {
		t.Fatalf("chrome_a should remain after failed delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "chrome_b")); !os.IsNotExist(err) {
		t.Fatalf("chrome_b should be removed, stat err = %v", err)
	}
}
