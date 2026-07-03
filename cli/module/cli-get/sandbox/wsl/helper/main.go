package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	sandboxModeFilePick    = "filepick"
	sandboxModeNet         = "net"
	sandboxModeFilePickNet = "filepick_net"

	sandboxAllowedDirEnv = "CLI_SANDBOX_ALLOWED_DIR"
	sandboxForcePickEnv  = "CLI_SANDBOX_FORCE_PICK"
	sandboxStateDirName  = "CLI_SANDBOX"

	defaultTaskTimeout = 180 * time.Second
	pickerTimeout      = 60 * time.Second
)

var defaultMode string

var helperLookPathFn = exec.LookPath
var helperCommandContextFn = exec.CommandContext
var helperUserConfigDirFn = os.UserConfigDir
var helperUserHomeDirFn = os.UserHomeDir
var helperStatFn = os.Stat
var helperMkdirAllFn = os.MkdirAll
var helperReadFileFn = os.ReadFile
var helperWriteFileFn = os.WriteFile
var helperReadProcVersionFn = func() ([]byte, error) { return os.ReadFile("/proc/version") }

var sandboxPermissionDeniedMarkers = []string{
	"permission denied",
	"operation not permitted",
	"not permitted",
	"sandbox denied",
	"forbidden by sandbox",
	"权限拒绝",
	"权限不足",
	"没有权限",
	"不允许的操作",
}

func main() {
	var directCmd string
	var directTimeout int
	var shell string
	var logFile string
	var mode string
	var allowedDir string

	flag.StringVar(&shell, "shell", defaultShell(), "shell used to execute delegated commands")
	flag.StringVar(&logFile, "log-file", "sandbox.log", "sandbox log file path")
	flag.StringVar(&mode, "mode", defaultMode, "sandbox mode: filepick, net, filepick_net")
	flag.StringVar(&allowedDir, "allowed-dir", "", "cache an allowed directory for filepick-based modes")
	flag.StringVar(&directCmd, "cmd", "", "execute a single command and print its output")
	flag.IntVar(&directTimeout, "timeout", 0, "command timeout in ms; 0 uses the default timeout")
	flag.Parse()

	mode = normalizeSandboxMode(mode)
	forcePick := envTruthy(sandboxForcePickEnv)
	if strings.TrimSpace(allowedDir) != "" {
		normalized, err := setPickedDirectory(allowedDir)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			os.Exit(1)
		}
		if strings.TrimSpace(directCmd) == "" {
			fmt.Fprint(os.Stdout, normalized)
			return
		}
	}

	if strings.TrimSpace(directCmd) == "" && requiresPickedDirectory(mode) && forcePick {
		ctx, cancel := context.WithTimeout(context.Background(), pickerTimeout)
		defer cancel()
		normalized, err := resolvePickedDirectory(ctx, true)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			os.Exit(1)
		}
		fmt.Fprint(os.Stdout, normalized)
		return
	}

	if strings.TrimSpace(directCmd) == "" {
		fmt.Fprintln(os.Stderr, "CLI_SANDBOX requires --cmd or --allowed-dir")
		os.Exit(1)
	}

	result := runCommandWithMode(directCmd, shell, directTimeout, mode, forcePick)
	if result.Output != "" {
		if result.Status == 0 {
			fmt.Fprint(os.Stdout, result.Output)
		} else {
			fmt.Fprint(os.Stderr, result.Output)
		}
	}
	os.Exit(result.Status)
}

type commandResult struct {
	Status int
	Output string
}

func defaultShell() string {
	if shell := strings.TrimSpace(os.Getenv("SHELL")); shell != "" {
		return shell
	}
	return "/bin/sh"
}

func normalizeSandboxMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case sandboxModeFilePick:
		return sandboxModeFilePick
	case sandboxModeNet:
		return sandboxModeNet
	case sandboxModeFilePickNet:
		return sandboxModeFilePickNet
	default:
		return ""
	}
}

func requiresPickedDirectory(mode string) bool {
	switch normalizeSandboxMode(mode) {
	case sandboxModeFilePick, sandboxModeFilePickNet:
		return true
	default:
		return false
	}
}

func runCommandWithMode(cmdText, shell string, timeoutMS int, mode string, forcePick bool) commandResult {
	mode = normalizeSandboxMode(mode)
	if mode == "" {
		return commandResult{Status: 1, Output: "CLI_SANDBOX当前系统不支持该模式"}
	}
	bwrapPath, err := helperLookPathFn("bwrap")
	if err != nil || strings.TrimSpace(bwrapPath) == "" {
		return commandResult{Status: 1, Output: "CLI_SANDBOX当前系统未安装bubblewrap"}
	}

	timeout := time.Duration(timeoutMS) * time.Millisecond
	if timeoutMS <= 0 {
		timeout = defaultTaskTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if strings.TrimSpace(shell) == "" {
		shell = defaultShell()
	}
	shellPath, err := resolveShellPath(shell)
	if err != nil {
		return commandResult{Status: 1, Output: err.Error()}
	}

	pickedDir := ""
	if requiresPickedDirectory(mode) {
		pickedDir, err = resolvePickedDirectory(ctx, forcePick)
		if err != nil {
			return commandResult{Status: 1, Output: err.Error()}
		}
	}

	args, err := buildBubblewrapArgs(shellPath, cmdText, mode, pickedDir)
	if err != nil {
		return commandResult{Status: 1, Output: err.Error()}
	}
	cmd := helperCommandContextFn(ctx, bwrapPath, args...)
	output, permissionDenied, err := runCommandWithEarlyPermissionDetection(ctx, cancel, cmd)
	if permissionDenied {
		if strings.TrimSpace(output) == "" {
			output = "CLI_SANDBOX权限拒绝"
		}
		return commandResult{Status: 1, Output: output}
	}
	if ctx.Err() == context.DeadlineExceeded {
		return commandResult{Status: 1, Output: "命令执行超时"}
	}
	if ctx.Err() == context.Canceled {
		return commandResult{Status: 1, Output: "命令被终止"}
	}
	if err != nil {
		if strings.TrimSpace(output) != "" {
			return commandResult{Status: 1, Output: output}
		}
		return commandResult{Status: 1, Output: err.Error()}
	}
	return commandResult{Status: 0, Output: output}
}

func resolveShellPath(shell string) (string, error) {
	shell = strings.TrimSpace(shell)
	if shell == "" {
		return "", fmt.Errorf("shell is required")
	}
	if filepath.IsAbs(shell) {
		info, err := helperStatFn(shell)
		if err != nil {
			return "", err
		}
		if info.IsDir() {
			return "", fmt.Errorf("shell is a directory: %s", shell)
		}
		return filepath.Clean(shell), nil
	}
	path, err := helperLookPathFn(shell)
	if err != nil {
		return "", err
	}
	return filepath.Clean(path), nil
}

func resolvePickedDirectory(ctx context.Context, forcePick bool) (string, error) {
	if override := strings.TrimSpace(os.Getenv(sandboxAllowedDirEnv)); override != "" {
		return normalizeDirectoryPath(override)
	}
	if !forcePick {
		if cached, ok := readCachedPickedDirectory(); ok {
			return cached, nil
		}
	}
	pickerCtx, cancel := withPickerTimeout(ctx)
	defer cancel()
	picked, err := pickAllowedDirectory(pickerCtx)
	if err != nil {
		return "", err
	}
	return setPickedDirectory(picked)
}

func withPickerTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining > 0 && remaining <= pickerTimeout {
			return ctx, func() {}
		}
	}
	return context.WithTimeout(ctx, pickerTimeout)
}

func setPickedDirectory(path string) (string, error) {
	normalized, err := normalizeDirectoryPath(path)
	if err != nil {
		return "", err
	}
	if err := writeCachedPickedDirectory(normalized); err != nil {
		return "", err
	}
	return normalized, nil
}

func normalizeDirectoryPath(raw string) (string, error) {
	path := stripWrappedQuotes(strings.TrimSpace(raw))
	if path == "" {
		return "", fmt.Errorf("empty directory path")
	}
	converted, err := maybeConvertWindowsPath(path)
	if err != nil {
		return "", err
	}
	if converted != "" {
		path = converted
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	path = filepath.Clean(path)
	info, err := helperStatFn(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", path)
	}
	return path, nil
}

func maybeConvertWindowsPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	isWindowsPath := strings.Contains(path, `\`) ||
		strings.HasPrefix(path, `\\wsl$\`) ||
		(len(path) >= 2 && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) && path[1] == ':')
	if !isWindowsPath {
		return "", nil
	}
	wslpath, err := helperLookPathFn("wslpath")
	if err != nil {
		return "", fmt.Errorf("无法转换Windows目录到WSL路径: %s", path)
	}
	out, err := exec.Command(wslpath, "-u", path).CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(out))
		if text == "" {
			text = err.Error()
		}
		return "", fmt.Errorf("无法转换Windows目录到WSL路径: %s", text)
	}
	return strings.TrimSpace(string(out)), nil
}

func stripWrappedQuotes(path string) string {
	path = strings.TrimSpace(path)
	if len(path) >= 2 {
		first := path[0]
		last := path[len(path)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return strings.TrimSpace(path[1 : len(path)-1])
		}
	}
	return path
}

func readCachedPickedDirectory() (string, bool) {
	statePath, err := sandboxSelectedDirPath()
	if err != nil {
		return "", false
	}
	data, err := helperReadFileFn(statePath)
	if err != nil {
		return "", false
	}
	path, err := normalizeDirectoryPath(string(data))
	if err != nil {
		return "", false
	}
	return path, true
}

func writeCachedPickedDirectory(path string) error {
	statePath, err := sandboxSelectedDirPath()
	if err != nil {
		return err
	}
	return helperWriteFileFn(statePath, []byte(path+"\n"), 0o644)
}

func sandboxSelectedDirPath() (string, error) {
	stateDir, err := sandboxStateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(stateDir, "selected-dir.txt"), nil
}

func sandboxStateDir() (string, error) {
	configDir, err := helperUserConfigDirFn()
	if err != nil {
		return "", err
	}
	stateDir := filepath.Join(configDir, sandboxStateDirName)
	if err := helperMkdirAllFn(stateDir, 0o755); err != nil {
		return "", err
	}
	return stateDir, nil
}

func pickAllowedDirectory(ctx context.Context) (string, error) {
	if path, ok := pickDirectoryViaPowerShell(ctx); ok {
		return path, nil
	}
	if path, ok := pickDirectoryViaZenity(ctx); ok {
		return path, nil
	}
	if _, ok := readCachedPickedDirectory(); ok {
		return "", fmt.Errorf("CLI_SANDBOX权限拒绝")
	}
	return "", fmt.Errorf("未找到已授权目录，请先通过 CLI_SANDBOX 完成目录授权或显式传入 --allowed-dir")
}

func pickDirectoryViaPowerShell(ctx context.Context) (string, bool) {
	if !helperIsWSL() {
		return "", false
	}
	powershellPath := ""
	for _, candidate := range []string{"powershell.exe", "pwsh.exe"} {
		path, err := helperLookPathFn(candidate)
		if err == nil && strings.TrimSpace(path) != "" {
			powershellPath = path
			break
		}
	}
	if powershellPath == "" {
		return "", false
	}
	script := strings.Join([]string{
		"Add-Type -AssemblyName System.Windows.Forms | Out-Null",
		"$dialog = New-Object System.Windows.Forms.FolderBrowserDialog",
		"$dialog.Description = '请选择允许 CLI_SANDBOX 访问的目录'",
		"$dialog.ShowNewFolderButton = $false",
		"if ($dialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {",
		"  [Console]::OutputEncoding = [System.Text.Encoding]::UTF8",
		"  Write-Output $dialog.SelectedPath",
		"  exit 0",
		"}",
		"exit 1",
	}, "; ")
	cmd := helperCommandContextFn(ctx, powershellPath, "-NoProfile", "-STA", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", false
	}
	path := strings.TrimSpace(string(output))
	if path == "" {
		return "", false
	}
	converted, err := maybeConvertWindowsPath(path)
	if err != nil {
		return "", false
	}
	if converted == "" {
		converted = path
	}
	return converted, true
}

func pickDirectoryViaZenity(ctx context.Context) (string, bool) {
	zenityPath, err := helperLookPathFn("zenity")
	if err != nil || strings.TrimSpace(zenityPath) == "" {
		return "", false
	}
	cmd := helperCommandContextFn(ctx, zenityPath, "--file-selection", "--directory", "--title=选择允许 CLI_SANDBOX 访问的目录")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", false
	}
	path := strings.TrimSpace(string(output))
	if path == "" {
		return "", false
	}
	return path, true
}

func helperIsWSL() bool {
	if strings.TrimSpace(os.Getenv("WSL_DISTRO_NAME")) != "" {
		return true
	}
	if strings.TrimSpace(os.Getenv("WSL_INTEROP")) != "" {
		return true
	}
	data, err := helperReadProcVersionFn()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(data)), "microsoft")
}

func buildBubblewrapArgs(shellPath, cmdText, mode, pickedDir string) ([]string, error) {
	mode = normalizeSandboxMode(mode)
	if mode == "" {
		return nil, fmt.Errorf("sandbox mode is required")
	}
	if strings.TrimSpace(shellPath) == "" {
		return nil, fmt.Errorf("shell is required")
	}
	stateDir, err := sandboxStateDir()
	if err != nil {
		return nil, err
	}
	scratchHome := "/home/sandbox"
	chdir := "/tmp"
	if requiresPickedDirectory(mode) {
		chdir = pickedDir
	}

	args := []string{
		"--die-with-parent",
		"--new-session",
		"--clearenv",
		"--unshare-all",
		"--proc", "/proc",
		"--dev", "/dev",
		"--tmpfs", "/tmp",
	}
	if mode == sandboxModeFilePick {
		args = append(args, "--share-net")
	}

	createdDirs := make(map[string]struct{})
	createDir := func(path string) {
		for _, dir := range sandboxDirChain(path) {
			if dir == "/" {
				continue
			}
			if _, ok := createdDirs[dir]; ok {
				continue
			}
			args = append(args, "--dir", dir)
			createdDirs[dir] = struct{}{}
		}
	}
	addBind := func(flagName, src, dest string) {
		src = strings.TrimSpace(src)
		dest = strings.TrimSpace(dest)
		if src == "" || dest == "" {
			return
		}
		if info, err := helperStatFn(src); err != nil || !info.IsDir() {
			return
		}
		createDir(dest)
		args = append(args, flagName, src, dest)
	}

	for _, path := range []string{
		"/usr",
		"/bin",
		"/sbin",
		"/lib",
		"/lib64",
		"/etc",
		"/run/current-system/sw",
		"/nix/store",
	} {
		addBind("--ro-bind", path, path)
	}

	createDir(scratchHome)
	createDir(filepath.Join(scratchHome, ".config"))
	createDir(filepath.Join(scratchHome, ".cache"))
	createDir(filepath.Join(scratchHome, ".local", "state"))

	addBind("--bind", stateDir, stateDir)
	if deeprightRuntimeDir := helperDeepRightRuntimeDir(); deeprightRuntimeDir != "" {
		addBind("--bind", deeprightRuntimeDir, deeprightRuntimeDir)
	}
	addBind("--bind", "/var/tmp", "/var/tmp")
	for _, variant := range pickedDirectoryVariants(pickedDir) {
		addBind("--bind", variant, variant)
	}

	args = append(args,
		"--chdir", chdir,
		"--setenv", "HOME", scratchHome,
		"--setenv", "XDG_CONFIG_HOME", filepath.Join(scratchHome, ".config"),
		"--setenv", "XDG_CACHE_HOME", filepath.Join(scratchHome, ".cache"),
		"--setenv", "XDG_STATE_HOME", filepath.Join(scratchHome, ".local", "state"),
		"--setenv", "ZDOTDIR", scratchHome,
		"--setenv", "TMPDIR", "/tmp",
		"--setenv", "PATH", "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"--setenv", "SHELL", shellPath,
		shellPath, "-c", cmdText,
	)
	return args, nil
}

func helperDeepRightRuntimeDir() string {
	home, err := helperUserHomeDirFn()
	if err != nil {
		return ""
	}
	home = strings.TrimSpace(home)
	if home == "" {
		return ""
	}
	path := filepath.Join(home, "deepright")
	info, err := helperStatFn(path)
	if err != nil || !info.IsDir() {
		return ""
	}
	return filepath.Clean(path)
}

func pickedDirectoryVariants(path string) []string {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	path = filepath.Clean(path)
	seen := map[string]struct{}{path: {}}
	paths := []string{path}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		resolved = filepath.Clean(resolved)
		if _, ok := seen[resolved]; !ok {
			paths = append(paths, resolved)
		}
	}
	return paths
}

func sandboxDirChain(path string) []string {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" || path == "." || path == "/" {
		return nil
	}
	var dirs []string
	current := path
	for current != "" && current != "." && current != "/" {
		dirs = append(dirs, current)
		current = filepath.Dir(current)
	}
	for i, j := 0, len(dirs)-1; i < j; i, j = i+1, j-1 {
		dirs[i], dirs[j] = dirs[j], dirs[i]
	}
	return dirs
}

func runCommandWithEarlyPermissionDetection(ctx context.Context, cancel context.CancelFunc, cmd *exec.Cmd) (string, bool, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", false, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", false, err
	}
	if err := cmd.Start(); err != nil {
		return err.Error(), isSandboxPermissionDenied(err.Error()), err
	}

	var (
		buf            bytes.Buffer
		bufMu          sync.Mutex
		permissionFlag atomic.Bool
		permissionOnce sync.Once
	)

	stopForPermission := func() {
		permissionOnce.Do(func() {
			permissionFlag.Store(true)
			cancel()
			if cmd.Process != nil && runtime.GOOS != "windows" {
				_ = cmd.Process.Signal(syscall.SIGKILL)
			}
		})
	}

	readPipe := func(r io.Reader) error {
		chunk := make([]byte, 4096)
		for {
			n, readErr := r.Read(chunk)
			if n > 0 {
				text := string(chunk[:n])
				bufMu.Lock()
				_, _ = buf.WriteString(text)
				current := buf.String()
				bufMu.Unlock()
				if isSandboxPermissionDenied(current) {
					stopForPermission()
				}
			}
			if readErr != nil {
				if errors.Is(readErr, io.EOF) {
					return nil
				}
				return readErr
			}
		}
	}

	streamErrCh := make(chan error, 2)
	go func() { streamErrCh <- readPipe(stdout) }()
	go func() { streamErrCh <- readPipe(stderr) }()

	waitErr := cmd.Wait()
	streamErr1 := <-streamErrCh
	streamErr2 := <-streamErrCh

	bufMu.Lock()
	output := buf.String()
	bufMu.Unlock()

	if permissionFlag.Load() {
		return output, true, nil
	}
	if streamErr1 != nil {
		return output, false, streamErr1
	}
	if streamErr2 != nil {
		return output, false, streamErr2
	}
	return output, false, waitErr
}

func isSandboxPermissionDenied(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	for _, marker := range sandboxPermissionDeniedMarkers {
		if strings.Contains(normalized, strings.ToLower(marker)) {
			return true
		}
	}
	return false
}

func envTruthy(name string) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	switch value {
	case "", "0", "false", "no", "off":
		return false
	default:
		return true
	}
}
