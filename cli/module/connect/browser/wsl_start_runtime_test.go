package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBrowserPrepareWSLChromeDefForStartCopiesFilteredWindowsUserData(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	sourceUnix := filepath.Join(t.TempDir(), "chrome-user-data")
	targetUnix := filepath.Join(t.TempDir(), "chrome-def")

	writeFile := func(rel, content string) {
		path := filepath.Join(sourceUnix, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	writeFile("Default/WebStorage/keep.txt", "keep")
	writeFile("Default/Cache/remove.txt", "remove")
	writeFile("Default/Network/Cookies", "keep")
	writeFile("Default/Network/Cache/index", "keep")
	writeFile("Default/Service Worker/CacheStorage/remove.txt", "remove")
	writeFile("SingletonLock", "locked")

	browserWSLDetectFn = func() (bool, error) {
		return true, nil
	}
	browserWSLCommandCombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name != browserWSLPowerShellPath {
			t.Fatalf("unexpected command: %s %v", name, args)
		}
		script := args[len(args)-1]
		switch {
		case strings.Contains(script, "Get-Process chrome"):
			return []byte("2\n"), nil
		case strings.Contains(script, "GetFolderPath('LocalApplicationData')"):
			return []byte("C:\\Users\\tester\\AppData\\Local\\Google\\Chrome\\User Data\n"), nil
		default:
			t.Fatalf("unexpected powershell script: %s", script)
			return nil, nil
		}
	}
	browserWSLPathUnixFn = func(path string) (string, error) {
		switch path {
		case `C:\Users\tester\AppData\Local\Google\Chrome\User Data`:
			return sourceUnix, nil
		case browserWSLChromeDefRoot:
			return targetUnix, nil
		default:
			t.Fatalf("unexpected windows path: %q", path)
			return "", nil
		}
	}
	cleanupCalls := 0
	browserWSLCleanupChromeUserDataFn = func(profileDir string) error {
		cleanupCalls++
		if profileDir != browserWSLChromeDefRoot {
			t.Fatalf("cleanup profile dir = %q, want %q", profileDir, browserWSLChromeDefRoot)
		}
		return os.RemoveAll(filepath.Join(targetUnix, "SingletonLock"))
	}

	ok, err := browserPrepareWSLChromeDefForStart(nil)
	if err != nil {
		t.Fatalf("prepare error = %v", err)
	}
	if !ok {
		t.Fatal("expected prepare to succeed")
	}
	if cleanupCalls != 1 {
		t.Fatalf("cleanup calls = %d, want 1", cleanupCalls)
	}

	if _, err := os.Stat(filepath.Join(targetUnix, "Default", "WebStorage", "keep.txt")); err != nil {
		t.Fatalf("expected kept file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetUnix, "Default", "Cache", "remove.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected cache file removed, stat err=%v", err)
	}
	for _, rel := range []string{
		filepath.Join("Default", "Network", "Cookies"),
		filepath.Join("Default", "Network", "Cache", "index"),
		filepath.Join("Default", "Service Worker", "CacheStorage", "remove.txt"),
	} {
		if _, err := os.Stat(filepath.Join(targetUnix, rel)); err != nil {
			t.Fatalf("expected preserved login-state file %q: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(targetUnix, "SingletonLock")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected lock removed, stat err=%v", err)
	}
}

func TestBrowserShouldSkipWSLChromeUserDataPathPreservesLoginState(t *testing.T) {
	paths := []string{
		filepath.Join("Default", "Network", "Cookies"),
		filepath.Join("Default", "Network", "Cache", "index"),
		filepath.Join("Default", "Login Data"),
		filepath.Join("Default", "Login Data For Account"),
		filepath.Join("Default", "Web Data"),
		filepath.Join("Default", "Local Storage", "leveldb", "000003.log"),
		filepath.Join("Default", "Session Storage", "000003.log"),
		filepath.Join("Default", "IndexedDB", "https_example.com_0.indexeddb.leveldb", "000003.log"),
		filepath.Join("Default", "WebStorage", "keep"),
		filepath.Join("Default", "Service Worker", "CacheStorage", "keep"),
		filepath.Join("Profile 1", "Network", "Cookies"),
	}
	for _, path := range paths {
		if skip, reason := browserShouldSkipWSLChromeUserDataPath(path); skip {
			t.Fatalf("WSL copy skipped login-state path %q: %s", path, reason)
		}
	}
	if skip, _ := browserShouldSkipWSLChromeUserDataPath(filepath.Join("Default", "Cache", "data")); !skip {
		t.Fatal("WSL copy should still skip the Chrome cache directory")
	}
}
