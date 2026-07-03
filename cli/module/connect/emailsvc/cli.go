package emailsvc

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
)

const (
	defaultPIDFile         = "email.pid"
	defaultLogFile         = "email.log"
	defaultRunLog          = "email.runtime.log"
	defaultScanSecs        = 60
	startupValidateTimeout = 30 * time.Second
)

type StartResult struct {
	Status  string `json:"status"`
	PID     int    `json:"pid,omitempty"`
	PIDFile string `json:"pidFile,omitempty"`
	LogFile string `json:"logFile,omitempty"`
	Name    string `json:"name,omitempty"`
}

var fixedMetaParams = []string{"email", "email_pop3", "email_smtp", "email_password", "email_whitelist"}
var supportedCommands = SupportedCommands()

func RunCLI(args []string, stdout, stderr io.Writer) int {
	return runCLI(args, stdout, stderr, nil)
}

func RunCLIWithPrefix(args []string, stdout, stderr io.Writer, launchPrefix []string) int {
	return runCLI(args, stdout, stderr, launchPrefix)
}

func runCLI(args []string, stdout, stderr io.Writer, launchPrefix []string) int {
	if len(args) == 0 || connectsvc.IsHelpCommand(args[0]) {
		printHelp(stdout)
		return 0
	}

	command := strings.TrimSpace(args[0])
	rest := args[1:]
	flags, err := connectsvc.ParseFlags(rest)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	switch command {
	case "command":
		connectsvc.WriteJSON(stdout, supportedCommands)
		return 0
	case "help":
		printHelp(stdout)
		return 0
	case "param":
		connectsvc.WriteJSON(stdout, fixedMetaParams)
		return 0
	case "scope":
		connectsvc.WriteJSON(stdout, SupportedScopes())
		return 0
	case "schema":
		connectsvc.WriteJSON(stdout, ResponseSchemaDefinition())
		return 0
	case "name":
		connectsvc.WriteJSON(stdout, map[string]string{
			"key":  DefaultName,
			"name": DefaultDisplayName,
		})
		return 0
	case "send":
		result, err := runSend("send", flags, stderr)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		connectsvc.WriteJSON(stdout, result)
		return 0
	case "init":
		result, err := runSend("init", flags, stderr)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		connectsvc.WriteJSON(stdout, result)
		return 0
	case "start":
		if connectsvc.BoolValue(flags, "foreground", false) {
			return runForeground(flags, stderr)
		}
		result, err := startBackground(flags, launchPrefix)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		connectsvc.WriteJSON(stdout, result)
		return 0
	case "serve":
		return runForeground(flags, stderr)
	case "stop":
		result, err := stopProcess(flags)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		connectsvc.WriteJSON(stdout, result)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", command)
		printHelp(stderr)
		return 1
	}
}

func runForeground(flags map[string]string, stderr io.Writer) int {
	pidFile := pidFilePath(flags)
	if err := ensureParentDir(pidFile); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	messageLogPath := strings.TrimSpace(connectsvc.FirstValue(flags, "log-file"))
	if messageLogPath == "" {
		messageLogPath = defaultLogFile
	}
	if err := ensureParentDir(messageLogPath); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	messageFile, err := os.OpenFile(messageLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	defer messageFile.Close()

	runtimeLogPath := runtimeLogPathFor(messageLogPath)
	if err := ensureParentDir(runtimeLogPath); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	runtimeFile, err := os.OpenFile(runtimeLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	defer runtimeFile.Close()

	logger := log.New(io.MultiWriter(stderr, runtimeFile), "[email] ", log.LstdFlags)
	if err := writePIDFile(pidFile); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	defer os.Remove(pidFile)

	service, err := newServiceBuilder(flags, logger, messageFile)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	writeStartupConnectMessageLog(messageFile, logger, flags)

	ctx, stop := signalContextFn()
	defer stop()

	if err := service.Run(ctx); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return 0
}

func buildService(flags map[string]string, logger *log.Logger, messageLog io.Writer) (*Service, error) {
	baseArgs, err := buildConnectBaseArgs(flags)
	if err != nil {
		return nil, err
	}

	scanSeconds, err := connectsvc.IntValue(flags, "scan-seconds", defaultScanSecs)
	if err != nil {
		return nil, err
	}
	if scanSeconds <= 0 {
		scanSeconds = defaultScanSecs
	}

	client := &CLIConnectClient{
		Binary:   resolveConnectBinary(flags),
		Prefix:   resolveConnectPrefix(flags),
		BaseArgs: baseArgs,
	}

	return NewService(Options{
		Name:        connectsvc.FirstValue(flags, "name"),
		Client:      client,
		Logger:      logger,
		MessageLog:  messageLog,
		DownloadDir: downloadDirForLog(connectsvc.FirstValue(flags, "log-file")),
		StateFile:   stateFileForLog(connectsvc.FirstValue(flags, "log-file")),
		ScanEvery:   time.Duration(scanSeconds) * time.Second,
	})
}

func startBackground(flags map[string]string, launchPrefix []string) (*StartResult, error) {
	if err := validateBackgroundStartup(flags); err != nil {
		return nil, err
	}

	pidFile := pidFilePath(flags)
	if _, ok := runningPID(pidFile); ok {
		if _, err := stopProcess(flags); err != nil {
			return nil, fmt.Errorf("restart stop failed: %w", err)
		}
	}
	if err := ensureParentDir(pidFile); err != nil {
		return nil, err
	}

	logFile := strings.TrimSpace(connectsvc.FirstValue(flags, "log-file"))
	if logFile == "" {
		logFile = defaultLogFile
	}
	if err := ensureParentDir(logFile); err != nil {
		return nil, err
	}
	runtimeLog := runtimeLogPathFor(logFile)
	if err := ensureParentDir(runtimeLog); err != nil {
		return nil, err
	}
	writer, err := os.OpenFile(runtimeLog, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	defer writer.Close()

	exe, err := osExecutableFn()
	if err != nil {
		return nil, err
	}
	args := append([]string{}, launchPrefix...)
	args = append(args, "start")
	args = append(args, rebuildFlags(flags, map[string]struct{}{
		"foreground": {},
		"log-file":   {},
		"pid-file":   {},
	})...)
	args = append(args, "--pid-file", pidFile, "--log-file", logFile, "--foreground", "true")

	cmd := execCommand(exe, args...)
	cmd.Stdout = writer
	cmd.Stderr = writer
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	childPID := cmd.Process.Pid
	_ = cmd.Process.Release()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if pid, ok := runningPID(pidFile); ok {
			return &StartResult{
				Status:  "started",
				PID:     pid,
				PIDFile: pidFile,
				LogFile: logFile,
				Name:    firstNonEmpty(connectsvc.FirstValue(flags, "name"), DefaultName),
			}, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !processExists(childPID) {
		return nil, fmt.Errorf("start process exited before pid file was ready; runtime log: %s", runtimeLog)
	}
	return nil, fmt.Errorf("start timeout waiting for pid file; runtime log: %s", runtimeLog)
}

func validateBackgroundStartup(flags map[string]string) error {
	service, err := newServiceBuilder(flags, log.New(io.Discard, "", 0), io.Discard)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), startupValidateTimeout)
	defer cancel()
	return service.ValidateStartup(ctx)
}

func stopProcess(flags map[string]string) (*StartResult, error) {
	pidFile := pidFilePath(flags)
	pid, ok := runningPID(pidFile)
	if !ok {
		if err := os.Remove(pidFile); err == nil || errors.Is(err, os.ErrNotExist) {
			return &StartResult{Status: "stopped", PIDFile: pidFile}, nil
		}
		return nil, fmt.Errorf("email not running: %s", pidFile)
	}
	proc, err := findProcess(pid)
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
				Name:    firstNonEmpty(connectsvc.FirstValue(flags, "name"), DefaultName),
			}, nil
		}
		if isZombiePID(pid) {
			_ = os.Remove(pidFile)
			return &StartResult{
				Status:  "stopped",
				PID:     pid,
				PIDFile: pidFile,
				Name:    firstNonEmpty(connectsvc.FirstValue(flags, "name"), DefaultName),
			}, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("stop timeout waiting pid=%d", pid)
}

func pidFilePath(flags map[string]string) string {
	path := strings.TrimSpace(connectsvc.FirstValue(flags, "pid-file"))
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
	proc, err := findProcess(pid)
	if err != nil {
		return 0, false
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return 0, false
	}
	return pid, true
}

func processExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := findProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func isZombiePID(pid int) bool {
	if pid <= 0 {
		return false
	}
	output, err := execCommand("ps", "-o", "stat=", "-p", strconv.Itoa(pid)).CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(strings.TrimSpace(string(output)), "Z")
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

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  email command")
	fmt.Fprintln(w, "  email schema")
	fmt.Fprintln(w, "  email param")
	fmt.Fprintln(w, "  email scope")
	fmt.Fprintln(w, "  email name")
	fmt.Fprintln(w, "  email send [options]")
	fmt.Fprintln(w, "  email init [options]")
	fmt.Fprintln(w, "  email start [options]")
	fmt.Fprintln(w, "  email stop [options]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Fixed output commands:")
	fmt.Fprintln(w, "  command                         print supported commands: [\"command\",\"help\",\"name\",\"param\",\"scope\",\"schema\",\"init\",\"send\",\"start\",\"stop\"]")
	fmt.Fprintln(w, "  help                            print plugin user guide")
	fmt.Fprintln(w, "  schema                          print plugin response json schema")
	fmt.Fprintln(w, "  param                           print meta schema keys: [\"email\",\"email_pop3\",\"email_smtp\",\"email_password\",\"email_whitelist\"]")
	fmt.Fprintln(w, "  scope                           print supported config scopes: [\"reuse\",\"agent\",\"provider\",\"thinking\",\"swarm\"]")
	fmt.Fprintln(w, "  name                            print display name: \"邮件\"")
	fmt.Fprintln(w, "  send                            reply with html text and attachments using the add-request request json")
	fmt.Fprintln(w, "  init                            send the initialization message using the same arguments and flow as send")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  --message JSON                  required, add-request request json with latest rawRequest envelope")
	fmt.Fprintln(w, "  --content TEXT                  html/plain text body content")
	fmt.Fprintln(w, "  --image PATHS                   comma separated image attachment paths")
	fmt.Fprintln(w, "  --file PATHS                    comma separated file attachment paths")
	fmt.Fprintln(w, "  --name NAME                     connect meta key/name, default email")
	fmt.Fprintln(w, "  --connect-bin PATH              connect or integration binary path")
	fmt.Fprintln(w, "  --addr HOST:PORT                connect service address")
	fmt.Fprintln(w, "  --port PORT                     connect service port shorthand")
	fmt.Fprintln(w, "  --connect-cache MS              connect cli cache ttl, default 10000")
	fmt.Fprintln(w, "  --scan-seconds N                unread mail scan interval, default 60")
	fmt.Fprintln(w, "  --pid-file PATH                 pid file, default ./email.pid")
	fmt.Fprintln(w, "  --log-file PATH                 message log file, default ./email.log")
	fmt.Fprintln(w, "  --foreground true               run in foreground")
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

var osExecutableFn = os.Executable

type processHandle interface {
	Signal(sig os.Signal) error
}

type osProcessHandle struct {
	*os.Process
}

func (p osProcessHandle) Signal(sig os.Signal) error {
	return p.Process.Signal(sig)
}

var findProcess = func(pid int) (processHandle, error) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil, err
	}
	return osProcessHandle{Process: proc}, nil
}

func runtimeLogPathFor(messageLogPath string) string {
	messageLogPath = strings.TrimSpace(messageLogPath)
	if messageLogPath == "" {
		messageLogPath = defaultLogFile
	}
	dir := filepath.Dir(messageLogPath)
	return filepath.Join(dir, defaultRunLog)
}

func openMessageAndRuntimeLogs(flags map[string]string, stderr io.Writer) (io.WriteCloser, io.WriteCloser, *log.Logger, error) {
	messageLogPath := strings.TrimSpace(connectsvc.FirstValue(flags, "log-file"))
	if messageLogPath == "" {
		messageLogPath = defaultLogFile
	}
	if err := ensureParentDir(messageLogPath); err != nil {
		return nil, nil, nil, err
	}
	messageFile, err := os.OpenFile(messageLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, nil, err
	}

	runtimeLogPath := runtimeLogPathFor(messageLogPath)
	if err := ensureParentDir(runtimeLogPath); err != nil {
		messageFile.Close()
		return nil, nil, nil, err
	}
	runtimeFile, err := os.OpenFile(runtimeLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		messageFile.Close()
		return nil, nil, nil, err
	}
	logger := log.New(io.MultiWriter(stderr, runtimeFile), "[email] ", log.LstdFlags)
	return messageFile, runtimeFile, logger, nil
}

func downloadDirForLog(messageLogPath string) string {
	messageLogPath = strings.TrimSpace(messageLogPath)
	if messageLogPath == "" {
		messageLogPath = defaultLogFile
	}
	dir := filepath.Dir(messageLogPath)
	return filepath.Join(dir, "email_artifacts")
}

func stateFileForLog(messageLogPath string) string {
	messageLogPath = strings.TrimSpace(messageLogPath)
	if messageLogPath == "" {
		messageLogPath = defaultLogFile
	}
	dir := filepath.Dir(messageLogPath)
	return filepath.Join(dir, stateFileName)
}

func buildConnectBaseArgs(flags map[string]string) ([]string, error) {
	cacheMS, err := connectsvc.IntValue(flags, "connect-cache", 10000)
	if err != nil {
		return nil, err
	}
	baseArgs := make([]string, 0, 6)
	if addr := connectsvc.FirstValue(flags, "addr"); addr != "" {
		baseArgs = append(baseArgs, "--addr", addr)
	}
	if port := connectsvc.FirstValue(flags, "port"); port != "" {
		baseArgs = append(baseArgs, "--port", port)
	}
	baseArgs = append(baseArgs, "--connect-cache", strconv.Itoa(cacheMS))
	return baseArgs, nil
}

func logStartupConnectParams(logger *log.Logger, flags map[string]string) {
	line, ok := startupConnectLogLine(flags)
	if !ok || logger == nil {
		return
	}
	logger.Print(line)
}

func writeStartupConnectMessageLog(messageLog io.Writer, logger *log.Logger, flags map[string]string) {
	line, ok := startupConnectLogLine(flags)
	if !ok {
		return
	}
	writeTimestampedMessageLog(messageLog, logger, time.Now().Format(time.RFC3339), line)
}

func startupConnectLogLine(flags map[string]string) (string, bool) {
	baseArgs, err := buildConnectBaseArgs(flags)
	if err != nil {
		return "", false
	}
	name := firstNonEmpty(connectsvc.FirstValue(flags, "name"), DefaultName)
	args := append([]string{}, resolveConnectPrefix(flags)...)
	args = append(args, "meta-get", "--key", normalizeMetaKey(name))
	args = append(args, baseArgs...)
	return fmt.Sprintf("stage=startup-connect name=%s command=%s", name, formatCommand(resolveConnectBinary(flags), args)), true
}

func writeTimestampedMessageLog(messageLog io.Writer, logger *log.Logger, at, content string) {
	if messageLog == nil {
		return
	}
	line := HumanizeLogEntry(strings.TrimSpace(at), strings.TrimSpace(content)) + "\n"
	if _, err := io.WriteString(messageLog, line); err != nil && logger != nil {
		logger.Printf("stage=message-log err=%v", err)
	}
}

func runSend(action string, flags map[string]string, stderr io.Writer) (*SendResult, error) {
	messageFile, runtimeFile, logger, err := openMessageAndRuntimeLogs(flags, stderr)
	if err != nil {
		return nil, err
	}
	defer messageFile.Close()
	defer runtimeFile.Close()

	service, err := newServiceBuilder(flags, logger, messageFile)
	if err != nil {
		writeTimestampedMessageLog(messageFile, logger, time.Now().Format(time.RFC3339), formatSendRequestLog(action, SendInput{
			Action:  action,
			Message: connectsvc.FirstValue(flags, "message"),
			Content: connectsvc.FirstValue(flags, "content"),
			Images:  splitCSVFlag(connectsvc.FirstValue(flags, "image", "images")),
			Files:   splitCSVFlag(connectsvc.FirstValue(flags, "file", "files")),
		}))
		writeTimestampedMessageLog(messageFile, logger, time.Now().Format(time.RFC3339), formatSendFailureLog(action, "build-service", nil, err))
		return nil, err
	}
	sender := newSender(service, logger, messageFile)
	result, err := sender.Send(context.Background(), SendInput{
		Action:  action,
		Message: connectsvc.FirstValue(flags, "message"),
		Content: connectsvc.FirstValue(flags, "content"),
		Images:  splitCSVFlag(connectsvc.FirstValue(flags, "image", "images")),
		Files:   splitCSVFlag(connectsvc.FirstValue(flags, "file", "files")),
	})
	if err != nil {
		return nil, err
	}
	action = strings.TrimSpace(strings.ToLower(action))
	if action == "" {
		action = "send"
	}
	return result, nil
}

func formatCommand(name string, args []string) string {
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

func resolveConnectBinary(flags map[string]string) string {
	if explicit := strings.TrimSpace(connectsvc.FirstValue(flags, "connect-bin")); explicit != "" {
		if filepath.IsAbs(explicit) {
			return explicit
		}
		if strings.HasPrefix(explicit, ".") || strings.Contains(explicit, "/") || strings.Contains(explicit, string(filepath.Separator)) {
			if abs, err := filepath.Abs(explicit); err == nil {
				return abs
			}
		}
		return explicit
	}

	for _, candidate := range connectBinaryCandidates() {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return candidate
		}
	}
	return "connect"
}

func resolveConnectPrefix(flags map[string]string) []string {
	explicit := strings.TrimSpace(connectsvc.FirstValue(flags, "connect-bin"))
	if explicit != "" {
		base := strings.ToLower(filepath.Base(explicit))
		if base == "proxy" || base == "integration" || strings.HasPrefix(base, "proxy.") || strings.HasPrefix(base, "integration.") {
			return []string{"connect"}
		}
		return nil
	}

	resolved := resolveConnectBinary(flags)
	base := strings.ToLower(filepath.Base(resolved))
	if base == "proxy" || base == "integration" || strings.HasPrefix(base, "proxy.") || strings.HasPrefix(base, "integration.") {
		return []string{"connect"}
	}
	return nil
}

var newSender = NewSender

func connectBinaryCandidates() []string {
	seen := make(map[string]struct{})
	add := func(list *[]string, path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		clean := filepath.Clean(path)
		if _, ok := seen[clean]; ok {
			return
		}
		seen[clean] = struct{}{}
		*list = append(*list, clean)
	}

	var candidates []string
	if exe, err := osExecutableFn(); err == nil && strings.TrimSpace(exe) != "" {
		exeDir := filepath.Dir(exe)
		add(&candidates, filepath.Join(exeDir, "connect"))
		add(&candidates, filepath.Join(exeDir, "..", "connect", "connect"))
		add(&candidates, filepath.Join(exeDir, "..", "proxy", "proxy"))
		add(&candidates, filepath.Join(exeDir, "..", "integration", "integration"))
		add(&candidates, filepath.Join(exeDir, "..", "module", "connect", "connect"))
		add(&candidates, filepath.Join(exeDir, "..", "module", "proxy", "proxy"))
		add(&candidates, filepath.Join(exeDir, "..", "module", "integration", "integration"))
	}
	if cwd, err := os.Getwd(); err == nil && strings.TrimSpace(cwd) != "" {
		add(&candidates, filepath.Join(cwd, "connect"))
		add(&candidates, filepath.Join(cwd, "..", "connect", "connect"))
		add(&candidates, filepath.Join(cwd, "..", "proxy", "proxy"))
		add(&candidates, filepath.Join(cwd, "..", "integration", "integration"))
		add(&candidates, filepath.Join(cwd, "..", "..", "connect", "connect"))
		add(&candidates, filepath.Join(cwd, "..", "..", "proxy", "proxy"))
		add(&candidates, filepath.Join(cwd, "..", "..", "integration", "integration"))
	}
	return candidates
}

var newServiceBuilder = buildService
var signalContextFn = signalContext
