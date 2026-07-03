package taskexec

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

const DefaultTaskTimeout = 180 * time.Second

var permissionDeniedMarkers = []string{
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

type ActiveCmd struct {
	Cancel    context.CancelFunc
	Cmd       *exec.Cmd
	AgentID   string
	ChatID    string
	Tid       string
	RawCmd    string
	StartedAt string
}

type Options struct {
	Shell                   string
	TimeoutMs               int
	AgentID                 string
	ChatID                  string
	Tid                     string
	OnStart                 func(*ActiveCmd)
	DetectPermissionDenied  bool
	PermissionDeniedMatcher func(string) bool
}

type Result struct {
	Status           int
	Output           string
	PermissionDenied bool
	TimedOut         bool
	Cancelled        bool
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

func IsPermissionDenied(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	for _, marker := range permissionDeniedMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func Execute(rawCmd string, opts Options) Result {
	timeout := time.Duration(opts.TimeoutMs) * time.Millisecond
	if opts.TimeoutMs <= 0 {
		timeout = DefaultTaskTimeout
	}

	shell := strings.TrimSpace(opts.Shell)
	if shell == "" {
		shell = DefaultShell()
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, shell, "-c", rawCmd)
	cmd.Env = os.Environ()

	if opts.OnStart != nil {
		opts.OnStart(&ActiveCmd{
			Cancel:    cancel,
			Cmd:       cmd,
			AgentID:   strings.TrimSpace(opts.AgentID),
			ChatID:    strings.TrimSpace(opts.ChatID),
			Tid:       strings.TrimSpace(opts.Tid),
			RawCmd:    strings.TrimSpace(rawCmd),
			StartedAt: time.Now().Format("2006-01-02T15:04:05.000"),
		})
	}

	matcher := opts.PermissionDeniedMatcher
	if matcher == nil {
		matcher = IsPermissionDenied
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Result{Status: 1, Output: err.Error()}
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return Result{Status: 1, Output: err.Error()}
	}

	if err := cmd.Start(); err != nil {
		return Result{
			Status:           1,
			Output:           err.Error(),
			PermissionDenied: opts.DetectPermissionDenied && matcher(err.Error()),
		}
	}

	var (
		buf            bytes.Buffer
		bufMu          sync.Mutex
		permissionFlag bool
		permissionOnce sync.Once
	)

	stopForPermission := func() {
		permissionOnce.Do(func() {
			permissionFlag = true
			cancel()
			if cmd.Process == nil {
				return
			}
			if runtime.GOOS == "windows" {
				_ = cmd.Process.Kill()
				return
			}
			_ = cmd.Process.Signal(syscall.SIGKILL)
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
				if opts.DetectPermissionDenied && matcher(current) {
					stopForPermission()
				}
			}
			if readErr != nil {
				if readErr == io.EOF {
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

	if permissionFlag {
		return Result{Status: 1, Output: output, PermissionDenied: true}
	}
	if streamErr1 != nil {
		return Result{Status: 1, Output: streamErr1.Error()}
	}
	if streamErr2 != nil {
		return Result{Status: 1, Output: streamErr2.Error()}
	}
	if ctx.Err() == context.DeadlineExceeded {
		return Result{Status: 1, Output: "命令执行超时", TimedOut: true}
	}
	if ctx.Err() == context.Canceled {
		return Result{Status: 1, Output: "命令被终止", Cancelled: true}
	}
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok && exitErr.ExitCode() == -1 {
			return Result{Status: 1, Output: "命令被终止", Cancelled: true}
		}
		if strings.TrimSpace(output) != "" {
			return Result{Status: 1, Output: output}
		}
		return Result{Status: 1, Output: waitErr.Error()}
	}
	return Result{Status: 0, Output: output}
}
