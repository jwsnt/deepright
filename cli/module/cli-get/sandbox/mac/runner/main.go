package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type runnerConfig struct {
	BundleID string `json:"bundleId"`
	Mode     string `json:"mode"`
}

const (
	runnerSandboxForcePickEnv = "CLI_SANDBOX_FORCE_PICK"
	runnerChooserTimeout      = 60 * time.Second
)

func main() {
	cfg, err := loadRunnerConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	var shell string
	var logFile string
	var directCmd string
	var directTimeout int
	var allowedDir string

	flag.StringVar(&shell, "shell", defaultShell(), "shell used by CLI_SANDBOX")
	flag.StringVar(&logFile, "log-file", "", "absolute log file path; defaults to the app sandbox container")
	flag.StringVar(&allowedDir, "allowed-dir", "", "provide an allowed directory for filepick-based modes")
	flag.StringVar(&directCmd, "cmd", "", "execute a single command and print its output")
	flag.IntVar(&directTimeout, "timeout", 0, "command timeout in ms; 0 uses the default timeout")
	flag.Parse()

	helperPath, err := embeddedHelperPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if logFile == "" {
		logFile = defaultLogFile(cfg.BundleID)
	}
	logDir := filepath.Dir(logFile)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create log directory failed: %v\n", err)
		os.Exit(1)
	}

	if requiresPickedDirectory(cfg.Mode) {
		allowedDir, err = resolveAllowedDirectory(allowedDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}

	forcePickOnly := strings.TrimSpace(os.Getenv(runnerSandboxForcePickEnv)) != "" && strings.TrimSpace(directCmd) == "" && strings.TrimSpace(allowedDir) != ""
	if forcePickOnly {
		fmt.Fprint(os.Stdout, allowedDir)
		return
	}

	args := []string{
		"--mode", cfg.Mode,
		"--shell", shell,
		"--log-file", logFile,
	}
	if strings.TrimSpace(allowedDir) != "" {
		args = append(args, "--allowed-dir", allowedDir)
	}
	if strings.TrimSpace(directCmd) != "" {
		args = append(args,
			"--cmd", directCmd,
			"--timeout", fmt.Sprintf("%d", directTimeout),
		)
	}
	if strings.TrimSpace(directCmd) == "" && strings.TrimSpace(allowedDir) == "" {
		fmt.Fprintln(os.Stderr, "CLI_SANDBOX requires --cmd or --allowed-dir")
		os.Exit(1)
	}

	cmd := exec.Command(helperPath, args...)
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	cmd.Stdin = os.Stdin
	cmd.Env = forwardedEnvironment(strings.TrimSpace(allowedDir) != "")
	err = cmd.Run()
	if stdoutText := stdoutBuf.String(); stdoutText != "" {
		fmt.Fprint(os.Stdout, stdoutText)
	}
	if stderrText := sanitizeRunnerStderr(stderrBuf.String()); stderrText != "" {
		fmt.Fprint(os.Stderr, stderrText)
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func loadRunnerConfig() (*runnerConfig, error) {
	configPath, err := runnerConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read runner config: %w", err)
	}
	var cfg runnerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse runner config: %w", err)
	}
	if strings.TrimSpace(cfg.BundleID) == "" {
		cfg.BundleID = "cn.deepright.cli-sandbox"
	}
	return &cfg, nil
}

func runnerConfigPath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate executable: %w", err)
	}
	contentsDir := filepath.Dir(filepath.Dir(exePath))
	return filepath.Join(contentsDir, "Resources", "runner-config.json"), nil
}

func embeddedHelperPath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate executable: %w", err)
	}
	contentsDir := filepath.Dir(filepath.Dir(exePath))
	return filepath.Join(contentsDir, "Helpers", "CLI_SANDBOX"), nil
}

func defaultShell() string {
	if shell := strings.TrimSpace(os.Getenv("SHELL")); shell != "" {
		return shell
	}
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	return "/bin/sh"
}

func defaultLogFile(bundleID string) string {
	if homeBase := containerHomeBase(bundleID); homeBase != "" {
		logDir := filepath.Join(homeBase, "Library", "Logs", "CLI_SANDBOX")
		return filepath.Join(logDir, "sandbox.log")
	}
	return filepath.Join(os.TempDir(), "CLI_SANDBOX", "sandbox.log")
}

func containerHomeBase(bundleID string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	home = filepath.Clean(strings.TrimSpace(home))
	if home == "" {
		return ""
	}
	containerSuffix := filepath.Join("Library", "Containers", strings.TrimSpace(bundleID), "Data")
	if strings.TrimSpace(bundleID) != "" && strings.HasSuffix(home, containerSuffix) {
		return home
	}
	if strings.TrimSpace(bundleID) != "" {
		return filepath.Join(home, "Library", "Containers", bundleID, "Data")
	}
	return home
}

func sanitizeRunnerStderr(raw string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(raw, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if isSandboxSystemWarningLine(trimmed) {
			continue
		}
		kept = append(kept, line)
	}
	if len(kept) == 0 {
		return ""
	}
	return strings.Join(kept, "\n")
}

func isSandboxSystemWarningLine(line string) bool {
	if strings.Contains(line, "Failure on line 686 in function id scheduleApplicationNotification(") {
		return true
	}
	if strings.Contains(line, "_LSModifyNotification(") {
		return true
	}
	return false
}

func requiresPickedDirectory(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "filepick", "filepick_net":
		return true
	default:
		return false
	}
}

func resolveAllowedDirectory(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return normalizeDirectoryPath(explicit)
	}
	picked, err := sandboxChooseFolderWithTimeout(runnerChooserTimeout)
	if err != nil {
		return "", err
	}
	picked, err = normalizeDirectoryPath(picked)
	if err != nil {
		return "", err
	}
	return picked, nil
}

func normalizeDirectoryPath(raw string) (string, error) {
	path := stripWrappedQuotes(strings.TrimSpace(raw))
	if path == "" {
		return "", fmt.Errorf("empty directory path")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", absPath)
	}
	return filepath.Clean(absPath), nil
}

func stripWrappedQuotes(path string) string {
	path = strings.TrimSpace(path)
	if len(path) >= 2 {
		if (path[0] == '"' && path[len(path)-1] == '"') || (path[0] == '\'' && path[len(path)-1] == '\'') {
			return strings.TrimSpace(path[1 : len(path)-1])
		}
	}
	return path
}

func forwardedEnvironment(stripForcePick bool) []string {
	if !stripForcePick {
		return os.Environ()
	}
	out := make([]string, 0, len(os.Environ()))
	for _, item := range os.Environ() {
		if strings.HasPrefix(item, runnerSandboxForcePickEnv+"=") {
			continue
		}
		out = append(out, item)
	}
	return out
}
