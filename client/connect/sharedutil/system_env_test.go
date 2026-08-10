package sharedutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestApplySystemPathMergesUnixShellAndSystemPaths(t *testing.T) {
	resetSystemPathTestHooks(t)

	t.Setenv("PATH", "/usr/bin")
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("HOME", "/Users/tester")

	systemPathRuntimeGOOS = "darwin"
	systemPathCommandFn = func(name string, args ...string) ([]byte, error) {
		switch {
		case name == "/bin/zsh":
			return []byte("/usr/bin:/Users/tester/.local/node/bin:/opt/homebrew/bin"), nil
		case name == "/usr/libexec/path_helper":
			return []byte(`PATH="/usr/bin:/bin:/usr/sbin:/sbin"; export PATH;`), nil
		case name == "/bin/launchctl":
			return []byte("/usr/bin:/bin:/usr/sbin:/sbin\n"), nil
		default:
			return nil, fmt.Errorf("unexpected command %s %v", name, args)
		}
	}

	ApplySystemPath()

	got := os.Getenv("PATH")
	parts := splitPathEntries(got)
	if len(parts) == 0 {
		t.Fatalf("PATH is empty after ApplySystemPath")
	}
	if !containsPathEntry(parts, "/Users/tester/.local/node/bin") {
		t.Fatalf("PATH = %q, want user shell bin included", got)
	}
	if !containsPathEntry(parts, "/opt/homebrew/bin") {
		t.Fatalf("PATH = %q, want Homebrew bin included", got)
	}
	if countPathEntry(parts, "/usr/bin") != 1 {
		t.Fatalf("PATH = %q, want /usr/bin deduped", got)
	}
}

func TestApplySystemPathIncludesDarwinPythonRuntimeBins(t *testing.T) {
	resetSystemPathTestHooks(t)

	t.Setenv("PATH", "/usr/bin")
	t.Setenv("VIRTUAL_ENV", "/Users/tester/.virtualenvs/active")
	t.Setenv("PYENV_ROOT", "/Users/tester/.pyenv")
	systemPathRuntimeGOOS = "darwin"
	systemPathUserHomeFn = func() (string, error) { return "/Users/tester", nil }
	systemPathGlobFn = func(pattern string) ([]string, error) {
		switch pattern {
		case "/Users/tester/Library/Python/*/bin":
			return []string{
				"/Users/tester/Library/Python/3.9/bin",
				"/Users/tester/Library/Python/3.13/bin",
			}, nil
		case "/Users/tester/.pyenv/versions/*/bin":
			return []string{"/Users/tester/.pyenv/versions/3.12/bin"}, nil
		case "/Library/Frameworks/Python.framework/Versions/*/bin":
			return []string{"/Library/Frameworks/Python.framework/Versions/3.12/bin"}, nil
		default:
			return nil, nil
		}
	}
	systemPathCommandFn = func(string, ...string) ([]byte, error) { return nil, nil }

	ApplySystemPath()

	parts := splitPathEntries(os.Getenv("PATH"))
	for _, want := range []string{
		"/Users/tester/.virtualenvs/active/bin",
		"/Users/tester/Library/Python/3.9/bin",
		"/Users/tester/Library/Python/3.13/bin",
		"/Users/tester/.pyenv/versions/3.12/bin",
		"/Library/Frameworks/Python.framework/Versions/3.12/bin",
	} {
		if !containsPathEntry(parts, want) {
			t.Fatalf("PATH = %q, want %q", os.Getenv("PATH"), want)
		}
	}
}

func TestApplySystemPathMergesWindowsUserAndMachinePath(t *testing.T) {
	resetSystemPathTestHooks(t)

	t.Setenv("Path", `C:\Windows\System32`)
	t.Setenv("SystemRoot", `C:\Windows`)
	t.Setenv("USERPROFILE", `C:\Users\tester`)
	t.Setenv("LocalAppData", `C:\Users\tester\AppData\Local`)
	t.Setenv("AppData", `C:\Users\tester\AppData\Roaming`)

	systemPathRuntimeGOOS = "windows"
	systemPathCommandFn = func(name string, args ...string) ([]byte, error) {
		if strings.HasSuffix(strings.ToLower(name), `powershell.exe`) || strings.EqualFold(name, "powershell.exe") {
			return []byte("C:\\Program Files\\nodejs\r\nC:\\Users\\tester\\AppData\\Roaming\\npm\r\n"), nil
		}
		return nil, fmt.Errorf("unexpected command %s %v", name, args)
	}

	ApplySystemPath()

	gotPath := os.Getenv("Path")
	gotPATH := os.Getenv("PATH")
	if gotPath == "" || gotPATH == "" {
		t.Fatalf("Path/PATH should both be populated, got Path=%q PATH=%q", gotPath, gotPATH)
	}
	if gotPath != gotPATH {
		t.Fatalf("Path and PATH should match, got Path=%q PATH=%q", gotPath, gotPATH)
	}
	parts := splitPathEntries(gotPath)
	if !containsPathEntry(parts, `C:\Program Files\nodejs`) {
		t.Fatalf("Path = %q, want machine Node path included", gotPath)
	}
	if !containsPathEntry(parts, `C:\Users\tester\AppData\Roaming\npm`) {
		t.Fatalf("Path = %q, want user npm path included", gotPath)
	}
	if countPathEntry(parts, `C:\Windows\System32`) != 1 {
		t.Fatalf("Path = %q, want System32 deduped", gotPath)
	}
}

func resetSystemPathTestHooks(t *testing.T) {
	t.Helper()

	oldGOOS := systemPathRuntimeGOOS
	oldLookupEnv := systemPathLookupEnvFn
	oldSetenv := systemPathSetenvFn
	oldUserHome := systemPathUserHomeFn
	oldGlob := systemPathGlobFn
	oldCommandFn := systemPathCommandFn

	systemPathApplyOnce = sync.Once{}
	systemPathRuntimeGOOS = oldGOOS
	systemPathLookupEnvFn = os.LookupEnv
	systemPathSetenvFn = os.Setenv
	systemPathUserHomeFn = os.UserHomeDir
	systemPathGlobFn = filepath.Glob
	systemPathCommandFn = func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("unexpected command %s %v", name, args)
	}

	t.Cleanup(func() {
		systemPathApplyOnce = sync.Once{}
		systemPathRuntimeGOOS = oldGOOS
		systemPathLookupEnvFn = oldLookupEnv
		systemPathSetenvFn = oldSetenv
		systemPathUserHomeFn = oldUserHome
		systemPathGlobFn = oldGlob
		systemPathCommandFn = oldCommandFn
	})
}

func containsPathEntry(entries []string, target string) bool {
	for _, entry := range entries {
		if entry == target {
			return true
		}
	}
	return false
}

func countPathEntry(entries []string, target string) int {
	count := 0
	for _, entry := range entries {
		if entry == target {
			count++
		}
	}
	return count
}
