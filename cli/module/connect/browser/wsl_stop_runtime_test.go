package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserCleanupWSLProgramDataChromeDirsRemovesOnlyChromeDef(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	root := t.TempDir()
	browserWSLDetectFn = func() (bool, error) {
		return true, nil
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

	if len(removed) != 1 || removed[0] != "chrome_def" {
		t.Fatalf("removed = %v, want [chrome_def]", removed)
	}
	for _, name := range []string{"chrome_1234", "other"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatalf("%s directory should stay: %v", name, err)
		}
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

	for _, name := range []string{"chrome_def", "chrome_a"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	attempted := []string{}
	browserRemoveAllFn = func(path string) error {
		attempted = append(attempted, filepath.Base(path))
		if filepath.Base(path) == "chrome_def" {
			return fs.ErrPermission
		}
		return os.RemoveAll(filepath.Join(root, filepath.Base(path)))
	}

	if err := browserCleanupWSLProgramDataChromeDirs(); err != nil {
		t.Fatal(err)
	}

	if len(attempted) != 1 || attempted[0] != "chrome_def" {
		t.Fatalf("attempted = %v, want [chrome_def]", attempted)
	}
	if _, err := os.Stat(filepath.Join(root, "chrome_def")); err != nil {
		t.Fatalf("chrome_def should remain after failed delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "chrome_a")); err != nil {
		t.Fatalf("chrome_a should not be touched, stat err = %v", err)
	}
}
