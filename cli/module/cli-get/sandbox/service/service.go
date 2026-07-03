package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	defaultTaskTimeout = 180 * time.Second
)

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

type TaskContent struct {
	Timeout  int             `json:"timeout"`
	Suffix   string          `json:"suffix"`
	Type     string          `json:"type"`
	Tid      string          `json:"tid"`
	Cmd      string          `json:"cmd"`
	AgentID  string          `json:"agentId,omitempty"`
	Chat     string          `json:"chat,omitempty"`
	Model    string          `json:"model,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type ResultPayload struct {
	Status  int    `json:"status"`
	AgentID string `json:"agentId,omitempty"`
	Suffix  string `json:"suffix"`
	Chat    string `json:"chat,omitempty"`
	Type    string `json:"type"`
	Cmd     string `json:"cmd"`
	Tid     string `json:"tid"`
}

type CommandResult struct {
	Status int
	Output string
}

func DefaultShell() string {
	if shell := strings.TrimSpace(os.Getenv("SHELL")); shell != "" {
		return shell
	}
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	return "/bin/sh"
}

func ExecuteTask(task *TaskContent, shell string) *ResultPayload {
	status, output := executeShellCommand(shell, task, "")
	return &ResultPayload{
		Status:  status,
		AgentID: task.AgentID,
		Suffix:  task.Suffix,
		Chat:    task.Chat,
		Type:    task.Type,
		Cmd:     output,
		Tid:     task.Tid,
	}
}

func RunCommand(cmdText, shell string, timeoutMS int) CommandResult {
	return RunCommandWithMode(cmdText, shell, timeoutMS, "")
}

func RunCommandWithMode(cmdText, shell string, timeoutMS int, mode string) CommandResult {
	task := &TaskContent{
		Timeout: timeoutMS,
		Suffix:  "cmd",
		Type:    "cmd",
		Tid:     "direct",
		Cmd:     cmdText,
	}
	if strings.TrimSpace(shell) == "" {
		shell = DefaultShell()
	}
	status, output := executeShellCommand(shell, task, mode)
	return CommandResult{
		Status: status,
		Output: output,
	}
}

func executeShellCommand(shell string, task *TaskContent, mode string) (int, string) {
	timeout := time.Duration(task.Timeout) * time.Millisecond
	if task.Timeout <= 0 {
		timeout = defaultTaskTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if strings.TrimSpace(shell) == "" {
		shell = DefaultShell()
	}
	mode = NormalizeSandboxMode(mode)

	var (
		cmd       *exec.Cmd
		extraEnv  []string
		pickedDir string
	)
	if mode == "" {
		cmd = exec.CommandContext(ctx, shell, "-c", task.Cmd)
	} else {
		if runtime.GOOS != "darwin" {
			return 1, "CLI_SANDBOX当前系统不支持该模式"
		}
		if _, err := exec.LookPath("sandbox-exec"); err != nil {
			return 1, "CLI_SANDBOX当前系统不支持该模式"
		}
		if requiresPickedDirectory(mode) {
			var err error
			pickedDir, err = resolvePickedDirectory()
			if err != nil {
				if strings.TrimSpace(err.Error()) != "" {
					return 1, err.Error()
				}
				return 1, "CLI_SANDBOX权限拒绝"
			}
			extraEnv = append(extraEnv, "ZDOTDIR="+pickedDir)
		}
		cmd = exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", buildSandboxProfile(mode, pickedDir), shell, "-c", task.Cmd)
		if pickedDir != "" {
			cmd.Dir = pickedDir
		}
	}
	cmd.Env = os.Environ()
	if len(extraEnv) > 0 {
		cmd.Env = append(cmd.Env, extraEnv...)
	}

	output, permissionDenied, err := runCommandWithEarlyPermissionDetection(ctx, cancel, cmd)

	if permissionDenied {
		if strings.TrimSpace(output) != "" {
			return 1, output
		}
		return 1, "CLI_SANDBOX权限拒绝"
	}
	if ctx.Err() == context.DeadlineExceeded {
		return 1, "命令执行超时"
	}
	if ctx.Err() == context.Canceled {
		return 1, "命令被终止"
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == -1 {
			return 1, "命令被终止"
		}
		if strings.TrimSpace(output) != "" {
			return 1, output
		}
		return 1, err.Error()
	}
	return 0, output
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
