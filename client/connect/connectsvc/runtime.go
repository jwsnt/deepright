package connectsvc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	defaultPIDFile = "connect.pid"
	defaultLogFile = "connect.log"
)

type StartResult struct {
	Status  string `json:"status"`
	PID     int    `json:"pid,omitempty"`
	PIDFile string `json:"pidFile,omitempty"`
	LogFile string `json:"logFile,omitempty"`
	Addr    string `json:"addr,omitempty"`
}

func runServiceCommand(command string, flags map[string]string, stdout, stderr io.Writer) int {
	switch command {
	case "start":
		if BoolValue(flags, "foreground", false) {
			return runForeground(flags, stderr)
		}
		result, err := startBackground(flags)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, result)
		return 0
	case "serve":
		return runForeground(flags, stderr)
	case "stop":
		result, err := stopProcess(flags)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, result)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown service command: %s\n", command)
		return 1
	}
}

func runForeground(flags map[string]string, stderr io.Writer) int {
	pidFile := pidFilePath(flags)
	if err := ensureParentDir(pidFile); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	logWriter := stderr
	logFilePath := strings.TrimSpace(FirstValue(flags, "log-file"))
	if logFilePath == "" {
		logFilePath = defaultLogFile
	}
	if err := ensureParentDir(logFilePath); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	defer file.Close()
	logWriter = io.MultiWriter(stderr, file)
	logger := log.New(logWriter, "[connect] ", log.LstdFlags)

	svc, err := buildService(flags)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	defer svc.Close()

	addr := ListenAddrFromFlags(flags)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	defer listener.Close()

	if err := writePIDFile(pidFile); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	defer os.Remove(pidFile)

	server := &http.Server{Handler: NewHTTPHandler(svc)}
	ctx, stop := signalContextFn()
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	logger.Printf("connect service started addr=%s db=%s agent-dir=%s", listener.Addr().String(), svc.dbPathValue(), svc.agentDir)
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return 0
}

func buildService(flags map[string]string) (*Service, error) {
	cacheMS, err := IntValue(flags, "connect-cache", 10000)
	if err != nil {
		return nil, err
	}
	return NewService(Options{
		DBPath:   FirstValue(flags, "db"),
		AgentDir: FirstValue(flags, "agent-dir"),
		CacheTTL: time.Duration(cacheMS) * time.Millisecond,
	})
}

func startBackground(flags map[string]string) (*StartResult, error) {
	pidFile := pidFilePath(flags)
	if pid, ok := runningPID(pidFile); ok {
		return nil, fmt.Errorf("connect already running: pid=%d", pid)
	}
	if err := ensureParentDir(pidFile); err != nil {
		return nil, err
	}

	logFile := strings.TrimSpace(FirstValue(flags, "log-file"))
	if logFile == "" {
		logFile = defaultLogFile
	}
	if err := ensureParentDir(logFile); err != nil {
		return nil, err
	}
	writer, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	defer writer.Close()

	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	args := []string{"serve"}
	args = append(args, rebuildFlags(flags, map[string]struct{}{
		"foreground": {},
		"log-file":   {},
		"pid-file":   {},
	})...)
	args = append(args, "--pid-file", pidFile, "--log-file", logFile)

	cmd := execCommand(exe, args...)
	cmd.Stdout = writer
	cmd.Stderr = writer
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	_ = cmd.Process.Release()

	client := NewAPIClient(ServiceBaseURLFromFlags(flags))
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if pid, ok := runningPID(pidFile); ok {
			if err := client.Health(); err == nil {
				return &StartResult{
					Status:  "started",
					PID:     pid,
					PIDFile: pidFile,
					LogFile: logFile,
					Addr:    ListenAddrFromFlags(flags),
				}, nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, errors.New("start timeout waiting for connect service")
}

func stopProcess(flags map[string]string) (*StartResult, error) {
	pidFile := pidFilePath(flags)
	pid, ok := runningPID(pidFile)
	if !ok {
		if err := os.Remove(pidFile); err == nil || errors.Is(err, os.ErrNotExist) {
			return &StartResult{Status: "stopped", PIDFile: pidFile, Addr: ListenAddrFromFlags(flags)}, nil
		}
		return nil, fmt.Errorf("connect not running: %s", pidFile)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil, err
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, stillRunning := runningPID(pidFile); !stillRunning {
			return &StartResult{
				Status:  "stopped",
				PID:     pid,
				PIDFile: pidFile,
				Addr:    ListenAddrFromFlags(flags),
			}, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("stop timeout waiting pid=%d", pid)
}

func pidFilePath(flags map[string]string) string {
	path := strings.TrimSpace(FirstValue(flags, "pid-file"))
	if path == "" {
		path = defaultPIDFile
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0o755)
}

func writePIDFile(path string) error {
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o644)
}

func runningPID(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return 0, false
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return 0, false
	}
	return pid, true
}

func rebuildFlags(flags map[string]string, skip map[string]struct{}) []string {
	keys := make([]string, 0, len(flags))
	for key := range flags {
		if _, ok := skip[key]; ok {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys)*2)
	for _, key := range keys {
		out = append(out, "--"+key, flags[key])
	}
	return out
}

func signalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-signals:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, func() {
		signal.Stop(signals)
		cancel()
	}
}

var execCommand = func(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

var signalContextFn = signalContext
