package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserWSLInstanceSeedReservedProfileDirSeedsFromChromeDef(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	sourceDir := t.TempDir()
	profileDir := filepath.Join(t.TempDir(), "chrome_ab12")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for rel, body := range map[string]string{
		filepath.Join("Default", "Preferences"):                            "{}",
		filepath.Join("Default", "Local Storage", "leveldb", "000003.log"): "keep",
		filepath.Join("Default", "Network", "Cache", "index"):              "keep",
		"SingletonLock": "lock",
	} {
		path := filepath.Join(sourceDir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	profileWin := `C:\ProgramData\deepright\chrome_ab12`
	browserWSLPathUnixFn = func(path string) (string, error) {
		switch path {
		case browserWSLChromeDefRoot:
			return sourceDir, nil
		case profileWin:
			return profileDir, nil
		default:
			return "", errors.New("unexpected windows path")
		}
	}

	cleanupCalls := 0
	browserWSLCleanupChromeUserDataFn = func(path string) error {
		cleanupCalls++
		if path != profileWin {
			t.Fatalf("cleanup path = %q, want %q", path, profileWin)
		}
		return os.RemoveAll(filepath.Join(profileDir, "SingletonLock"))
	}

	if err := browserWSLInstanceSeedReservedProfileDir(profileWin, profileDir); err != nil {
		t.Fatal(err)
	}
	if cleanupCalls != 1 {
		t.Fatalf("cleanupCalls = %d, want 1", cleanupCalls)
	}
	if _, err := os.Stat(filepath.Join(profileDir, "Default", "Preferences")); err != nil {
		t.Fatalf("expected copied Preferences: %v", err)
	}
	if _, err := os.Stat(filepath.Join(profileDir, "Default", "Local Storage", "leveldb", "000003.log")); err != nil {
		t.Fatalf("expected copied login data: %v", err)
	}
	if _, err := os.Stat(filepath.Join(profileDir, "Default", "Network", "Cache", "index")); err != nil {
		t.Fatalf("expected Network directory to be copied, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(profileDir, "SingletonLock")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected lock file cleanup, err=%v", err)
	}
}

func TestBrowserWSLInstanceSeedReservedProfileDirFallsBackWhenChromeDefMissing(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	profileDir := filepath.Join(t.TempDir(), "chrome_missing")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}

	profileWin := `C:\ProgramData\deepright\chrome_missing`
	missingSource := filepath.Join(t.TempDir(), "missing_chrome_def")
	browserWSLPathUnixFn = func(path string) (string, error) {
		switch path {
		case browserWSLChromeDefRoot:
			return missingSource, nil
		case profileWin:
			return profileDir, nil
		default:
			return "", errors.New("unexpected windows path")
		}
	}

	cleanupCalls := 0
	browserWSLCleanupChromeUserDataFn = func(path string) error {
		cleanupCalls++
		return nil
	}

	if err := browserWSLInstanceSeedReservedProfileDir(profileWin, profileDir); err != nil {
		t.Fatal(err)
	}
	if cleanupCalls != 0 {
		t.Fatalf("cleanupCalls = %d, want 0", cleanupCalls)
	}
	entries, err := os.ReadDir(profileDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty fallback profile dir, got %d entries", len(entries))
	}
}

func TestBrowserWSLInstanceSeedReservedProfileDirResetsPartialCopyFailure(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	sourceDir := t.TempDir()
	profileDir := filepath.Join(t.TempDir(), "chrome_copy_fail")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "Preferences"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	profileWin := `C:\ProgramData\deepright\chrome_copy_fail`
	browserWSLPathUnixFn = func(path string) (string, error) {
		switch path {
		case browserWSLChromeDefRoot:
			return sourceDir, nil
		case profileWin:
			return profileDir, nil
		default:
			return "", errors.New("unexpected windows path")
		}
	}

	oldCopyFn := browserWSLInstanceCopyDirectoryFn
	browserWSLInstanceCopyDirectoryFn = func(sourceDir, targetDir string, progressFn func(browserArchiveProgress), skipFn func(string, error)) (browserArchiveProgress, error) {
		if err := os.WriteFile(filepath.Join(targetDir, "partial.txt"), []byte("partial"), 0o644); err != nil {
			return browserArchiveProgress{}, err
		}
		return browserArchiveProgress{}, errors.New("copy failed")
	}
	defer func() {
		browserWSLInstanceCopyDirectoryFn = oldCopyFn
	}()

	cleanupCalls := 0
	browserWSLCleanupChromeUserDataFn = func(path string) error {
		cleanupCalls++
		return nil
	}

	if err := browserWSLInstanceSeedReservedProfileDir(profileWin, profileDir); err != nil {
		t.Fatal(err)
	}
	if cleanupCalls != 0 {
		t.Fatalf("cleanupCalls = %d, want 0", cleanupCalls)
	}
	entries, err := os.ReadDir(profileDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected reset empty profile dir after copy failure, got %d entries", len(entries))
	}
}
