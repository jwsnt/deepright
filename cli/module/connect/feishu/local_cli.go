package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"connect/connectsvc"
	"connect/feishusvc"
)

func runLocalLifecycleCommandFeishu(args []string, stdout, stderr io.Writer) (bool, int) {
	if len(args) == 0 {
		return false, 0
	}

	command := strings.TrimSpace(args[0])
	if command != "start" && command != "serve" && command != "stop" {
		return false, 0
	}

	flags, err := connectsvc.ParseFlags(args[1:])
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return true, 1
	}

	switch command {
	case "start":
		if connectsvc.BoolValue(flags, "foreground", false) {
			return true, runForegroundLocalFeishu(flags, stderr)
		}
		result, err := startBackgroundLocalFeishu(flags)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return true, 1
		}
		connectsvc.WriteJSON(stdout, result)
		return true, 0
	case "serve":
		return true, runForegroundLocalFeishu(flags, stderr)
	case "stop":
		result, err := stopProcessLocalFeishu(flags)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return true, 1
		}
		connectsvc.WriteJSON(stdout, result)
		return true, 0
	default:
		return false, 0
	}
}

func runForegroundLocalFeishu(flags map[string]string, stderr io.Writer) int {
	pidFile := pidFilePathLocalFeishu(flags)
	if err := ensureParentDirLocalFeishu(pidFile); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	messageFile, runtimeFile, logger, err := openLocalMessageAndRuntimeLogsFeishu(flags, stderr)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	defer messageFile.Close()
	defer runtimeFile.Close()

	if err := writePIDFileLocalFeishu(pidFile); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	defer os.Remove(pidFile)

	service, err := newLocalFeishuService(flags, logger, messageFile)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	writeStartupConnectMessageLogLocalFeishu(messageFile, logger, flags)

	ctx, stop := signalContextLocalFeishuFn()
	defer stop()

	if err := service.Run(ctx); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return 0
}

func startBackgroundLocalFeishu(flags map[string]string) (*feishusvc.StartResult, error) {
	if err := validateBackgroundStartupLocalFeishu(flags); err != nil {
		writeStartupValidationFailureLocalFeishu(flags, err)
		return nil, err
	}

	pidFile := pidFilePathLocalFeishu(flags)
	if _, ok := runningPIDLocalFeishu(pidFile); ok {
		if _, err := stopProcessLocalFeishu(flags); err != nil {
			return nil, fmt.Errorf("restart stop failed: %w", err)
		}
	}
	if err := ensureParentDirLocalFeishu(pidFile); err != nil {
		return nil, err
	}

	logFile := strings.TrimSpace(connectsvc.FirstValue(flags, "log-file"))
	if logFile == "" {
		logFile = defaultFeishuLogFile
	}
	if err := ensureParentDirLocalFeishu(logFile); err != nil {
		return nil, err
	}
	runtimeLog := runtimeLogPathForLocalFeishu(logFile)
	if err := ensureParentDirLocalFeishu(runtimeLog); err != nil {
		return nil, err
	}
	writer, err := os.OpenFile(runtimeLog, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	defer writer.Close()

	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	args := []string{"start"}
	args = append(args, rebuildFlagsLocalFeishu(flags, map[string]struct{}{
		"foreground": {},
		"log-file":   {},
		"pid-file":   {},
	})...)
	args = append(args, "--pid-file", pidFile, "--log-file", logFile, "--foreground", "true")

	cmd := exec.Command(exe, args...)
	cmd.Stdout = writer
	cmd.Stderr = writer
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	childPID := cmd.Process.Pid
	_ = cmd.Process.Release()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if pid, ok := runningPIDLocalFeishu(pidFile); ok {
			return &feishusvc.StartResult{
				Status:  "started",
				PID:     pid,
				PIDFile: pidFile,
				LogFile: logFile,
				Name:    firstNonEmptyLocalFeishu(connectsvc.FirstValue(flags, "name"), feishusvc.DefaultName),
			}, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !processExistsLocalFeishu(childPID) {
		return nil, fmt.Errorf("start process exited before pid file was ready; runtime log: %s", runtimeLog)
	}
	return nil, fmt.Errorf("start timeout waiting for pid file; runtime log: %s", runtimeLog)
}

func validateBackgroundStartupLocalFeishu(flags map[string]string) error {
	service, err := newLocalFeishuService(flags, log.New(io.Discard, "", 0), io.Discard)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return service.ValidateStartup(ctx)
}

func writeStartupValidationFailureLocalFeishu(flags map[string]string, startupErr error) {
	if startupErr == nil {
		return
	}
	logFile := strings.TrimSpace(connectsvc.FirstValue(flags, "log-file"))
	if logFile == "" {
		logFile = defaultFeishuLogFile
	}
	runtimeLog := runtimeLogPathForLocalFeishu(logFile)
	if err := ensureParentDirLocalFeishu(runtimeLog); err != nil {
		return
	}
	file, err := os.OpenFile(runtimeLog, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	logger := log.New(file, "[feishu] ", log.LstdFlags)
	logger.Printf("stage=startup-validation-failed name=%s err=%s", firstNonEmptyLocalFeishu(connectsvc.FirstValue(flags, "name"), feishusvc.DefaultName), quoteLogValueLocalFeishu(startupErr.Error()))
}

func stopProcessLocalFeishu(flags map[string]string) (*feishusvc.StartResult, error) {
	pidFile := pidFilePathLocalFeishu(flags)
	pid, ok := runningPIDLocalFeishu(pidFile)
	if !ok {
		if err := os.Remove(pidFile); err == nil || errors.Is(err, os.ErrNotExist) {
			return &feishusvc.StartResult{Status: "stopped", PIDFile: pidFile}, nil
		}
		return nil, fmt.Errorf("feishu not running: %s", pidFile)
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
		if _, stillRunning := runningPIDLocalFeishu(pidFile); !stillRunning {
			return &feishusvc.StartResult{
				Status:  "stopped",
				PID:     pid,
				PIDFile: pidFile,
				Name:    firstNonEmptyLocalFeishu(connectsvc.FirstValue(flags, "name"), feishusvc.DefaultName),
			}, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("stop timeout waiting pid=%d", pid)
}

func pidFilePathLocalFeishu(flags map[string]string) string {
	path := strings.TrimSpace(connectsvc.FirstValue(flags, "pid-file"))
	if path == "" {
		path = defaultFeishuPIDFile
	}
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

func ensureParentDirLocalFeishu(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

func writePIDFileLocalFeishu(path string) error {
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o644)
}

func runningPIDLocalFeishu(path string) (int, bool) {
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

func processExistsLocalFeishu(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func rebuildFlagsLocalFeishu(flags map[string]string, skip map[string]struct{}) []string {
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

func openLocalMessageAndRuntimeLogsFeishu(flags map[string]string, stderr io.Writer) (io.WriteCloser, io.WriteCloser, *log.Logger, error) {
	messageLogPath := strings.TrimSpace(connectsvc.FirstValue(flags, "log-file"))
	if messageLogPath == "" {
		messageLogPath = defaultFeishuLogFile
	}
	if err := ensureParentDirLocalFeishu(messageLogPath); err != nil {
		return nil, nil, nil, err
	}
	messageFile := newRotatingLocalFeishuLogWriter(messageLogPath, 10*1024*1024, 5)

	runtimeLogPath := runtimeLogPathForLocalFeishu(messageLogPath)
	if err := ensureParentDirLocalFeishu(runtimeLogPath); err != nil {
		messageFile.Close()
		return nil, nil, nil, err
	}
	runtimeFile, err := os.OpenFile(runtimeLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		messageFile.Close()
		return nil, nil, nil, err
	}
	logger := log.New(io.MultiWriter(stderr, runtimeFile), "[feishu] ", log.LstdFlags)
	return messageFile, runtimeFile, logger, nil
}

func writeStartupConnectMessageLogLocalFeishu(messageLog io.Writer, logger *log.Logger, flags map[string]string) {
	line, ok := startupConnectLogLineLocalFeishu(flags)
	if !ok {
		return
	}
	writeTimestampedMessageLogLocalFeishu(messageLog, logger, time.Now().Format(time.RFC3339), line)
}

func startupConnectLogLineLocalFeishu(flags map[string]string) (string, bool) {
	baseArgs, err := buildConnectBaseArgsLocalFeishu(flags)
	if err != nil {
		return "", false
	}
	name := firstNonEmptyLocalFeishu(connectsvc.FirstValue(flags, "name"), feishusvc.DefaultName)
	args := append([]string{}, resolveConnectPrefixLocalFeishu(flags)...)
	args = append(args, "meta-get", "--key", name)
	args = append(args, baseArgs...)
	return fmt.Sprintf("stage=startup-connect name=%s command=%s", name, formatCommandLocalFeishu(resolveConnectBinaryLocalFeishu(flags), args)), true
}

func formatCommandLocalFeishu(name string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	appendPart := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if strings.ContainsAny(value, " \t\r\n\"'") {
			parts = append(parts, strconv.Quote(value))
			return
		}
		parts = append(parts, value)
	}
	appendPart(name)
	for _, arg := range args {
		appendPart(arg)
	}
	return strings.Join(parts, " ")
}

var signalContextLocalFeishuFn = signalContextLocalFeishu

func signalContextLocalFeishu() (context.Context, context.CancelFunc) {
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
