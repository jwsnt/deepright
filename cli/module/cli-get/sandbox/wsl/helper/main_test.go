package main

import (
	"context"
	"os"
	"os/exec"
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

func TestSetPickedDirectoryReturnsNormalizedDirectory(t *testing.T) {
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
}

func TestResolvePickedDirectoryUsesEnvOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	allowedDir := filepath.Join(home, "picked")
	if err := os.MkdirAll(allowedDir, 0o755); err != nil {
		t.Fatalf("mkdir picked: %v", err)
	}
	t.Setenv(sandboxAllowedDirEnv, allowedDir)

	got, err := resolvePickedDirectory(t.Context(), false)
	if err != nil {
		t.Fatalf("resolvePickedDirectory err = %v", err)
	}
	if got != filepath.Clean(allowedDir) {
		t.Fatalf("path = %q, want %q", got, filepath.Clean(allowedDir))
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

func TestPickDirectoryViaPowerShellFallsBackToKnownWindowsPaths(t *testing.T) {
	t.Setenv("WSL_DISTRO_NAME", "Ubuntu")
	tmp := t.TempDir()
	wslpathPath := filepath.Join(tmp, "wslpath")
	wslpathScript := "#!/bin/sh\nprintf '/mnt/c/Users/demo/Desktop'\n"
	if err := os.WriteFile(wslpathPath, []byte(wslpathScript), 0o755); err != nil {
		t.Fatalf("write wslpath: %v", err)
	}

	originalLookPathFn := helperLookPathFn
	helperLookPathFn = func(name string) (string, error) {
		if name == "wslpath" {
			return wslpathPath, nil
		}
		return "", os.ErrNotExist
	}
	defer func() { helperLookPathFn = originalLookPathFn }()

	var invoked string
	originalCommandContextFn := helperCommandContextFn
	helperCommandContextFn = func(_ context.Context, name string, args ...string) *exec.Cmd {
		invoked = name
		return exec.Command("/bin/sh", "-c", "printf 'C:\\\\Users\\\\demo\\\\Desktop'")
	}
	defer func() { helperCommandContextFn = originalCommandContextFn }()

	path, ok, canceled := pickDirectoryViaPowerShell(t.Context())
	if !ok {
		t.Fatal("pickDirectoryViaPowerShell should succeed")
	}
	if canceled {
		t.Fatal("pickDirectoryViaPowerShell should not be canceled")
	}
	if invoked != "/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe" {
		t.Fatalf("powershell path = %q, want hardcoded Windows PowerShell path", invoked)
	}
	if path != "/mnt/c/Users/demo/Desktop" {
		t.Fatalf("path = %q, want %q", path, "/mnt/c/Users/demo/Desktop")
	}
}

func TestHelperWindowsPickerCandidatesPreferSiblingExecutable(t *testing.T) {
	tmp := t.TempDir()
	helperPath := filepath.Join(tmp, "CLI_SANDBOX")
	pickerPath := filepath.Join(tmp, "CLI_SANDBOX_PICKER_LAUNCHER")

	originalExecutableFn := helperExecutableFn
	helperExecutableFn = func() (string, error) { return helperPath, nil }
	defer func() { helperExecutableFn = originalExecutableFn }()

	originalLookPathFn := helperLookPathFn
	helperLookPathFn = func(name string) (string, error) {
		if name == "CLI_SANDBOX_PICKER_LAUNCHER" {
			return filepath.Join("/other", "CLI_SANDBOX_PICKER_LAUNCHER"), nil
		}
		return "", os.ErrNotExist
	}
	defer func() { helperLookPathFn = originalLookPathFn }()

	got := helperWindowsPickerCandidates()
	want := []string{pickerPath, filepath.Join("/other", "CLI_SANDBOX_PICKER_LAUNCHER")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("candidates = %#v, want %#v", got, want)
	}
}

func TestIsPickerCanceledTreatsCanceledOutputAsCancellation(t *testing.T) {
	err := exec.Command("/bin/sh", "-c", "exit 1").Run()
	if !isPickerCanceled(err, []byte("canceled\n")) {
		t.Fatal("isPickerCanceled should treat canceled output as cancellation")
	}
	if !isPickerCanceled(err, []byte("picker canceled\n")) {
		t.Fatal("isPickerCanceled should treat picker canceled output as cancellation")
	}
	if isPickerCanceled(err, []byte("other failure\n")) {
		t.Fatal("isPickerCanceled should not treat arbitrary output as cancellation")
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

func TestBuildBubblewrapArgsSpecialPathsAcrossModes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	picked := filepath.Join(home, "workspace")
	if err := os.MkdirAll(picked, 0o755); err != nil {
		t.Fatalf("mkdir picked: %v", err)
	}

	for _, tc := range []struct {
		mode             string
		wantPickedBind   bool
		wantShareNetwork bool
		wantChdir        string
	}{
		{mode: sandboxModeFilePick, wantPickedBind: true, wantShareNetwork: true, wantChdir: picked},
		{mode: sandboxModeNet, wantPickedBind: false, wantShareNetwork: false, wantChdir: "/tmp"},
		{mode: sandboxModeFilePickNet, wantPickedBind: true, wantShareNetwork: false, wantChdir: picked},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			args, err := buildBubblewrapArgs("/bin/sh", "pwd", tc.mode, picked)
			if err != nil {
				t.Fatalf("buildBubblewrapArgs: %v", err)
			}
			text := strings.Join(args, "\n")
			for _, path := range sandboxSystemReadOnlyPaths {
				if _, err := os.Stat(path); err != nil {
					continue
				}
				if !strings.Contains(text, "--ro-bind\n"+path+"\n"+path) {
					t.Fatalf("system path %s should be read-only mounted: %v", path, args)
				}
				if strings.Contains(text, "--bind\n"+path+"\n"+path) {
					t.Fatalf("system path %s must not be writable mounted: %v", path, args)
				}
			}
			if strings.Contains(text, "--bind\n/var/tmp\n/var/tmp") {
				t.Fatalf("host /var/tmp must not be mounted: %v", args)
			}
			if !strings.Contains(text, "--dir\n/var/tmp") {
				t.Fatalf("sandbox-private /var/tmp should be created: %v", args)
			}
			if !strings.Contains(text, "--setenv\nTMPDIR\n/tmp") {
				t.Fatalf("TMPDIR should use private /tmp: %v", args)
			}
			if !strings.Contains(text, "--setenv\nPATH\n"+sandboxCommandPath) {
				t.Fatalf("PATH should match mounted standard tool roots: %v", args)
			}
			if got := strings.Contains(text, "--bind\n"+picked+"\n"+picked); got != tc.wantPickedBind {
				t.Fatalf("picked bind = %t, want %t: %v", got, tc.wantPickedBind, args)
			}
			if got := strings.Contains(text, "--share-net"); got != tc.wantShareNetwork {
				t.Fatalf("share network = %t, want %t: %v", got, tc.wantShareNetwork, args)
			}
			if !strings.Contains(text, "--chdir\n"+tc.wantChdir) {
				t.Fatalf("chdir should be %s: %v", tc.wantChdir, args)
			}
		})
	}
}

func TestBuildBubblewrapArgsIncludesUserPythonRuntimeReadOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	bin := filepath.Join(home, ".local", "bin")
	site := filepath.Join(home, ".local", "lib", "python3.12", "site-packages")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatalf("create user bin: %v", err)
	}
	if err := os.MkdirAll(site, 0o755); err != nil {
		t.Fatalf("create user site packages: %v", err)
	}

	args, err := buildBubblewrapArgs("/bin/sh", "command -v whisper", sandboxModeNet, "")
	if err != nil {
		t.Fatalf("buildBubblewrapArgs: %v", err)
	}
	text := strings.Join(args, "\n")
	for _, path := range []string{bin, site} {
		if !strings.Contains(text, "--ro-bind\n"+path+"\n"+path) {
			t.Fatalf("user Python runtime %s should be read-only mounted: %v", path, args)
		}
		if strings.Contains(text, "--bind\n"+path+"\n"+path) {
			t.Fatalf("user Python runtime %s must not be writable mounted: %v", path, args)
		}
	}
	if !strings.Contains(text, "--setenv\nPATH\n"+bin+":"+sandboxCommandPath) {
		t.Fatalf("PATH should include user Python bin: %v", args)
	}
	if !strings.Contains(text, "--setenv\nPYTHONPATH\n"+site) {
		t.Fatalf("PYTHONPATH should include user site packages: %v", args)
	}
}

func TestBuildBubblewrapArgsRejectsSelectedSystemToolRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	_, err := buildBubblewrapArgs("/bin/sh", "pwd", sandboxModeFilePick, "/usr")
	if err == nil || !strings.Contains(err.Error(), "系统工具路径") {
		t.Fatalf("selected system tool root error = %v, want a system tool path rejection", err)
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

func TestTimeoutOutput(t *testing.T) {
	if got := timeoutOutput(""); got != "[Warning: Command execution timed out.]" {
		t.Fatalf("timeoutOutput(empty) = %q", got)
	}
	if got := timeoutOutput("partial output"); got != "partial output[Warning: Command execution timed out, the returned content may be incomplete.]" {
		t.Fatalf("timeoutOutput(partial) = %q", got)
	}
	if got := timeoutOutput("\n\t "); got != "[Warning: Command execution timed out.]" {
		t.Fatalf("timeoutOutput(whitespace) = %q", got)
	}
}
