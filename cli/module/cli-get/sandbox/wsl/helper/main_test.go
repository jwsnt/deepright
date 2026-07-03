package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeSandboxMode(t *testing.T) {
	if got := normalizeSandboxMode(" FILEPICK_NET "); got != sandboxModeFilePickNet {
		t.Fatalf("mode = %q, want %q", got, sandboxModeFilePickNet)
	}
	if got := normalizeSandboxMode("unknown"); got != "" {
		t.Fatalf("mode = %q, want empty", got)
	}
}

func TestSetPickedDirectoryWritesCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	allowedDir := filepath.Join(home, "picked")
	if err := os.MkdirAll(allowedDir, 0o755); err != nil {
		t.Fatalf("mkdir picked: %v", err)
	}

	got, err := setPickedDirectory(allowedDir)
	if err != nil {
		t.Fatalf("setPickedDirectory: %v", err)
	}
	if got != filepath.Clean(allowedDir) {
		t.Fatalf("path = %q, want %q", got, filepath.Clean(allowedDir))
	}

	cached, ok := readCachedPickedDirectory()
	if !ok {
		t.Fatal("readCachedPickedDirectory should return cache hit")
	}
	if cached != filepath.Clean(allowedDir) {
		t.Fatalf("cached = %q, want %q", cached, filepath.Clean(allowedDir))
	}
}

func TestResolvePickedDirectoryReturnsHelpfulErrorWithoutCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, err := resolvePickedDirectory(t.Context(), false)
	if err == nil {
		t.Fatal("resolvePickedDirectory err = nil, want authorization error")
	}
	if !strings.Contains(err.Error(), "显式传入 --allowed-dir") {
		t.Fatalf("err = %q, want missing authorization message", err)
	}
}

func TestBuildBubblewrapArgsFilePickNet(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	picked := filepath.Join(home, "workspace")
	if err := os.MkdirAll(picked, 0o755); err != nil {
		t.Fatalf("mkdir picked: %v", err)
	}

	args, err := buildBubblewrapArgs("/bin/sh", "pwd", sandboxModeFilePickNet, picked)
	if err != nil {
		t.Fatalf("buildBubblewrapArgs: %v", err)
	}
	text := strings.Join(args, "\n")
	if strings.Contains(text, "--share-net") {
		t.Fatalf("filepick_net should not share net: %v", args)
	}
	if !strings.Contains(text, "--chdir\n"+picked) {
		t.Fatalf("args should chdir into picked directory: %v", args)
	}
	if !strings.Contains(text, "--bind\n"+picked+"\n"+picked) {
		t.Fatalf("args should bind picked directory: %v", args)
	}
}

func TestRunCommandWithModeUsesBubblewrapBinary(t *testing.T) {
	tmp := t.TempDir()
	argsPath := filepath.Join(tmp, "args.txt")
	bwrapPath := filepath.Join(tmp, "bwrap")
	script := strings.Join([]string{
		"#!/bin/sh",
		"printf '%s\\n' \"$@\" > " + shellQuote(argsPath),
		"target_dir=''",
		"arg1=''",
		"arg2=''",
		"arg3=''",
		"while [ \"$#\" -gt 0 ]; do",
		"  if [ \"$1\" = '--chdir' ]; then",
		"    shift",
		"    target_dir=\"$1\"",
		"  fi",
		"  arg1=\"$arg2\"",
		"  arg2=\"$arg3\"",
		"  arg3=\"$1\"",
		"  shift",
		"done",
		"if [ -n \"$target_dir\" ]; then",
		"  cd \"$target_dir\" || exit 1",
		"fi",
		"exec \"$arg1\" \"$arg2\" \"$arg3\"",
	}, "\n")
	if err := os.WriteFile(bwrapPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write bwrap: %v", err)
	}

	oldLookPath := helperLookPathFn
	helperLookPathFn = func(name string) (string, error) {
		if name == "bwrap" {
			return bwrapPath, nil
		}
		return oldLookPath(name)
	}
	defer func() { helperLookPathFn = oldLookPath }()

	result := runCommandWithMode("printf sandbox_ok", "/bin/sh", 5000, sandboxModeNet, false)
	if result.Status != 0 {
		t.Fatalf("status = %d, output=%q", result.Status, result.Output)
	}
	if result.Output != "sandbox_ok" {
		t.Fatalf("output = %q, want sandbox_ok", result.Output)
	}

	argsData, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	got := strings.Split(strings.TrimSpace(string(argsData)), "\n")
	wantPrefix := []string{
		"--die-with-parent",
		"--new-session",
		"--clearenv",
		"--unshare-all",
		"--proc",
		"/proc",
		"--dev",
		"/dev",
		"--tmpfs",
		"/tmp",
	}
	if !reflect.DeepEqual(got[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("args prefix = %#v, want %#v", got[:len(wantPrefix)], wantPrefix)
	}
}

func shellQuote(path string) string {
	return "'" + strings.ReplaceAll(path, "'", "'\\''") + "'"
}
