package main

import (
	"context"
	"encoding/json"
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
	"connect/emailsvc"
)

const (
	defaultPIDFileLocal      = "email.pid"
	defaultPOP3ScanSecsLocal = 300
)

func runLocalLifecycleCommand(args []string, stdout, stderr io.Writer) (bool, int) {
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
			return true, runForegroundLocal(flags, stderr)
		}
		result, err := startBackgroundLocal(flags)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return true, 1
		}
		connectsvc.WriteJSON(stdout, result)
		return true, 0
	case "serve":
		return true, runForegroundLocal(flags, stderr)
	case "stop":
		result, err := stopProcessLocal(flags)
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

func runForegroundLocal(flags map[string]string, stderr io.Writer) int {
	pidFile := pidFilePathLocal(flags)
	if err := ensureParentDirLocal(pidFile); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	messageFile, runtimeFile, logger, err := openLocalMessageAndRuntimeLogs(flags, stderr)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	defer messageFile.Close()
	defer runtimeFile.Close()

	configClient, err := newLocalConfigClient(flags, logger, messageFile)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if err := ensureInitialScanStateForFlagsLocal(flags, configClient); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	if err := writePIDFileLocal(pidFile); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	defer os.Remove(pidFile)

	service, err := buildLocalService(flags, logger, messageFile)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	writeStartupConnectMessageLogLocal(messageFile, logger, flags)

	ctx, stop := signalContextLocal()
	defer stop()

	if err := service.Run(ctx); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return 0
}

func startBackgroundLocal(flags map[string]string) (*emailsvc.StartResult, error) {
	if err := validateBackgroundStartupLocal(flags); err != nil {
		return nil, err
	}

	pidFile := pidFilePathLocal(flags)
	if _, ok := runningPIDLocal(pidFile); ok {
		if _, err := stopProcessLocal(flags); err != nil {
			return nil, fmt.Errorf("restart stop failed: %w", err)
		}
	}
	if err := ensureParentDirLocal(pidFile); err != nil {
		return nil, err
	}

	logFile := strings.TrimSpace(connectsvc.FirstValue(flags, "log-file"))
	if logFile == "" {
		logFile = defaultLogFile
	}
	if err := ensureParentDirLocal(logFile); err != nil {
		return nil, err
	}
	runtimeLog := runtimeLogPathForLocal(logFile)
	if err := ensureParentDirLocal(runtimeLog); err != nil {
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
	args = append(args, rebuildFlagsLocal(flags, map[string]struct{}{
		"foreground": {},
		"log-file":   {},
		"pid-file":   {},
	})...)
	args = append(args, "--pid-file", pidFile, "--log-file", logFile, "--foreground", "true")

	cmd := execCommandLocal(exe, args...)
	cmd.Stdout = writer
	cmd.Stderr = writer
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	childPID := cmd.Process.Pid
	_ = cmd.Process.Release()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if pid, ok := runningPIDLocal(pidFile); ok {
			return &emailsvc.StartResult{
				Status:  "started",
				PID:     pid,
				PIDFile: pidFile,
				LogFile: logFile,
				Name:    firstNonEmptyLocal(connectsvc.FirstValue(flags, "name"), emailsvc.DefaultName),
			}, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !processExistsLocal(childPID) {
		return nil, fmt.Errorf("start process exited before pid file was ready; runtime log: %s", runtimeLog)
	}
	return nil, fmt.Errorf("start timeout waiting for pid file; runtime log: %s", runtimeLog)
}

func validateBackgroundStartupLocal(flags map[string]string) error {
	configClient, err := newLocalConfigClient(flags, log.New(io.Discard, "", 0), io.Discard)
	if err != nil {
		return err
	}
	if err := ensureInitialScanStateForFlagsLocal(flags, configClient); err != nil {
		return err
	}
	service, err := buildLocalService(flags, log.New(io.Discard, "", 0), io.Discard)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return service.ValidateStartup(ctx)
}

func stopProcessLocal(flags map[string]string) (*emailsvc.StartResult, error) {
	pidFile := pidFilePathLocal(flags)
	pid, ok := runningPIDLocal(pidFile)
	if !ok {
		if err := os.Remove(pidFile); err == nil || errors.Is(err, os.ErrNotExist) {
			return &emailsvc.StartResult{Status: "stopped", PIDFile: pidFile}, nil
		}
		return nil, fmt.Errorf("email not running: %s", pidFile)
	}
	proc, err := findProcessLocal(pid)
	if err != nil {
		return nil, err
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, stillRunning := runningPIDLocal(pidFile); !stillRunning {
			return &emailsvc.StartResult{
				Status:  "stopped",
				PID:     pid,
				PIDFile: pidFile,
				Name:    firstNonEmptyLocal(connectsvc.FirstValue(flags, "name"), emailsvc.DefaultName),
			}, nil
		}
		if isZombiePIDLocal(pid) {
			_ = os.Remove(pidFile)
			return &emailsvc.StartResult{
				Status:  "stopped",
				PID:     pid,
				PIDFile: pidFile,
				Name:    firstNonEmptyLocal(connectsvc.FirstValue(flags, "name"), emailsvc.DefaultName),
			}, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("stop timeout waiting pid=%d", pid)
}

func pidFilePathLocal(flags map[string]string) string {
	path := strings.TrimSpace(connectsvc.FirstValue(flags, "pid-file"))
	if path == "" {
		path = defaultPIDFileLocal
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func writePIDFileLocal(path string) error {
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o644)
}

func runningPIDLocal(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, false
	}
	proc, err := findProcessLocal(pid)
	if err != nil {
		return 0, false
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return 0, false
	}
	return pid, true
}

func processExistsLocal(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := findProcessLocal(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func isZombiePIDLocal(pid int) bool {
	if pid <= 0 {
		return false
	}
	output, err := execCommandLocal("ps", "-o", "stat=", "-p", strconv.Itoa(pid)).CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(strings.TrimSpace(string(output)), "Z")
}

type processHandleLocal interface {
	Signal(sig os.Signal) error
}

var execCommandLocal = func(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

var findProcessLocal = func(pid int) (processHandleLocal, error) {
	return os.FindProcess(pid)
}

func rebuildFlagsLocal(flags map[string]string, skip map[string]struct{}) []string {
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

func printHelpLocal(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  email command")
	fmt.Fprintln(w, "  email schema")
	fmt.Fprintln(w, "  email param")
	fmt.Fprintln(w, "  email scope")
	fmt.Fprintln(w, "  email name")
	fmt.Fprintln(w, "  email sender [options]")
	fmt.Fprintln(w, "  email search [--query TEXT] [--sender EMAIL] [options]")
	fmt.Fprintln(w, "  email send [options]")
	fmt.Fprintln(w, "  email init [options]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Fixed output commands:")
	fmt.Fprintln(w, "  command                         print supported commands as a json string array")
	fmt.Fprintln(w, "  help                            print plugin user guide")
	fmt.Fprintln(w, "  schema                          print plugin response json schema")
	fmt.Fprintln(w, "  param                           print meta schema examples: [{\"email\":\"邮箱地址，如hello_world@gmail.com\",\"email_pop3\":\"邮箱的pop3地址，如pop.gmail.com\",\"email_smtp\":\"邮箱的smtp地址，如smtp.gmail.com\",\"email_password\":\"邮箱的密码\",\"email_whitelist\":\"以逗号分隔的收件人白名单，如a@gmail.com,b@gmail.com\",\"email_pop3_interval\":\"每次扫描待处理邮件的间隔秒数，默认300\"}]")
	fmt.Fprintln(w, "  scope                           print supported config scopes: [\"reuse\",\"agent\",\"provider\",\"thinking\",\"swarm\"]")
	fmt.Fprintln(w, "  name                            print display name: \"邮件\"")
	fmt.Fprintln(w, "  sender                          list unique sender email addresses and their last message time in the configured window")
	fmt.Fprintln(w, "  search                          search normalized subject and body text in the configured window")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Message snapshot queries:")
	fmt.Fprintln(w, "  Both commands query the local Integration/Connect message snapshot only; they never connect to POP3/SMTP or read email.log.")
	fmt.Fprintln(w, "  The window comes from integration config/config.json: email.lastMessage (positive integer, hours).")
	fmt.Fprintln(w, "  Missing/invalid config causes an immediate error; no default window is used.")
	fmt.Fprintln(w, "  sender output: [{\"sender\":\"alice@example.com\",\"lastMessageAt\":\"2026-07-19T10:30:00Z\"}] (newest first)")
	fmt.Fprintln(w, "  search output: {\"total\":1,\"limit\":50,\"offset\":0,\"items\":[{\"messageId\":\"<mail@example.com>\",\"sender\":\"alice@example.com\",\"content\":\"...\",\"sentAt\":\"2026-07-19T10:30:00Z\"}]}")
	fmt.Fprintln(w, "  search --query TEXT            optional; whitespace terms are AND, quoted text is one phrase, case-insensitive contains")
	fmt.Fprintln(w, "  search --sender EMAIL          optional exact sender filter; normalized to lowercase and combined with --query using AND")
	fmt.Fprintln(w, "  search --limit N               result count, default 50, maximum 200")
	fmt.Fprintln(w, "  search --offset N              zero-based result offset, default 0")
	fmt.Fprintln(w, "  --connect-bin PATH             Integration/Connect binary used for the query and config/config.json lookup")
	fmt.Fprintln(w, "  Examples:")
	fmt.Fprintln(w, "    email sender --connect-bin ./integration")
	fmt.Fprintln(w, "    email search --limit 20 --offset 0 --connect-bin ./integration")
	fmt.Fprintln(w, "    email search --query \"退款 已处理\" --limit 20 --offset 0 --connect-bin ./integration")
	fmt.Fprintln(w, "    email search --query '\"退款申请\" 已处理' --connect-bin ./integration")
	fmt.Fprintln(w, "    email search --sender alice@example.com --limit 20 --connect-bin ./integration")
	fmt.Fprintln(w, "  send                            reply using rawRequest, or send a new mail with --to and --subject")
	fmt.Fprintln(w, "  init                            send the initialization message using the same arguments and flow as send")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  --message JSON                  optional add-request request json; rawRequest sends a reply")
	fmt.Fprintln(w, "  --to ADDRESSES                  comma separated recipients for a new mail without rawRequest")
	fmt.Fprintln(w, "  --subject TEXT                  subject for a new mail without rawRequest")
	fmt.Fprintln(w, "  --content TEXT                  plain text body content, or a JSON object matching `email schema`")
	fmt.Fprintln(w, "  --image PATHS                   comma separated image attachment paths")
	fmt.Fprintln(w, "  --file PATHS                    comma separated file attachment paths")
	fmt.Fprintln(w, "  --name NAME                     connect meta key/name, default email")
	fmt.Fprintln(w, "  --connect-bin PATH              connect or integration binary path")
	fmt.Fprintln(w, "  --addr HOST:PORT                connect service address")
	fmt.Fprintln(w, "  --port PORT                     connect service port shorthand")
	fmt.Fprintln(w, "  --connect-cache MS              connect cli cache ttl, default 10000")
	fmt.Fprintln(w, "  --scan-seconds N                unread mail scan interval, default 300")
	fmt.Fprintln(w, "  --pid-file PATH                 pid file, default ./email.pid")
	fmt.Fprintln(w, "  --log-file PATH                 message log file, default ./email.log")
	fmt.Fprintln(w, "  --foreground true               run in foreground")
}

func printSnapshotCommandHelpLocal(w io.Writer, command string) {
	if command == "sender" {
		fmt.Fprintln(w, "Usage: email sender [--connect-bin PATH]")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "List unique sender email addresses from the local Integration/Connect message snapshot.")
		fmt.Fprintln(w, "The lastMessageAt value is each sender's latest mail time in config/config.json email.lastMessage hours.")
		fmt.Fprintln(w, "No POP3/SMTP server or email.log is queried. Missing/invalid email.lastMessage fails immediately.")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "Example:")
		fmt.Fprintln(w, "  email sender --connect-bin ./integration")
		return
	}
	fmt.Fprintln(w, "Usage: email search [--query TEXT] [--sender EMAIL] [--limit N] [--offset N] [--connect-bin PATH]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Search normalized email subject and body text in the local Integration/Connect message snapshot.")
	fmt.Fprintln(w, "Terms separated by whitespace are AND; quoted text is one phrase; matching is case-insensitive contains.")
	fmt.Fprintln(w, "Without --query, all text messages in the configured window are listed. --sender is an exact lowercase email filter and combines with --query using AND.")
	fmt.Fprintln(w, "Attachments are excluded. --limit defaults to 50 and is capped at 200; --offset defaults to 0.")
	fmt.Fprintln(w, "The time window is config/config.json email.lastMessage positive integer hours; invalid config fails immediately.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  email search --limit 20 --offset 0 --connect-bin ./integration")
	fmt.Fprintln(w, "  email search --query \"退款 已处理\" --limit 20 --offset 0 --connect-bin ./integration")
	fmt.Fprintln(w, "  email search --query '\"退款申请\" 已处理' --connect-bin ./integration")
	fmt.Fprintln(w, "  email search --sender alice@example.com --limit 20 --connect-bin ./integration")
}

func signalContextLocal() (context.Context, context.CancelFunc) {
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

func writeStartupConnectMessageLogLocal(messageLog io.Writer, logger *log.Logger, flags map[string]string) {
	line, ok := startupConnectLogLineLocal(flags)
	if !ok {
		return
	}
	writeTimestampedLocalMessageLog(messageLog, logger, time.Now().Format(time.RFC3339), line)
}

func startupConnectLogLineLocal(flags map[string]string) (string, bool) {
	baseArgs, err := buildConnectBaseArgsLocal(flags)
	if err != nil {
		return "", false
	}
	name := firstNonEmptyLocal(connectsvc.FirstValue(flags, "name"), emailsvc.DefaultName)
	args := append([]string{}, resolveConnectPrefixLocal(flags)...)
	args = append(args, "meta-get", "--key", name)
	args = append(args, baseArgs...)
	return fmt.Sprintf("stage=startup-connect name=%s command=%s", name, formatCommandLocal(resolveConnectBinaryLocal(flags), args)), true
}

type localMetaConfigCompat struct {
	Email             string `json:"email"`
	EmailPOP3         string `json:"email_pop3"`
	EmailSMTP         string `json:"email_smtp"`
	EmailPassword     string `json:"email_password"`
	EmailWhitelist    string `json:"email_whitelist"`
	EmailPOP3Interval any    `json:"email_pop3_interval"`
	EmailPOP3Seconds  any    `json:"email_pop3_seconds"`
	ScanSeconds       int    `json:"scanSeconds"`
}

func buildLocalService(flags map[string]string, logger *log.Logger, messageLog io.Writer) (*emailsvc.Service, error) {
	configClient, err := newLocalConfigClient(flags, logger, messageLog)
	if err != nil {
		return nil, err
	}
	return emailsvc.NewService(emailsvc.Options{
		Name:        connectsvc.FirstValue(flags, "name"),
		Client:      configClient,
		Fetcher:     newLocalPersistentPOP3Fetcher(downloadDirForLogLocal(connectsvc.FirstValue(flags, "log-file"))),
		Logger:      logger,
		MessageLog:  messageLog,
		DownloadDir: downloadDirForLogLocal(connectsvc.FirstValue(flags, "log-file")),
		StateFile:   stateFileForLogLocal(connectsvc.FirstValue(flags, "log-file")),
		ScanEvery:   time.Duration(defaultPOP3ScanSecsLocal) * time.Second,
	})
}

func newLocalConfigClient(flags map[string]string, logger *log.Logger, messageLog io.Writer) (localConfigCompatClient, error) {
	baseArgs, err := buildConnectBaseArgsLocal(flags)
	if err != nil {
		return localConfigCompatClient{}, err
	}
	cliClient := &emailsvc.CLIConnectClient{
		Binary:   resolveConnectBinaryLocal(flags),
		Prefix:   resolveConnectPrefixLocal(flags),
		BaseArgs: baseArgs,
	}
	client := localSchemaConnectClient{inner: cliClient}
	configClient := localConfigCompatClient{
		inner:      client,
		binary:     cliClient.Binary,
		prefix:     append([]string{}, cliClient.Prefix...),
		baseArgs:   append([]string{}, cliClient.BaseArgs...),
		runner:     cliClient.Runner,
		logger:     logger,
		messageLog: messageLog,
	}
	return configClient, nil
}

type localConfigCompatClient struct {
	inner interface {
		GetMeta(context.Context, string) (*connectsvc.Meta, error)
		AddRequest(context.Context, connectsvc.RequestInput) (*connectsvc.Request, error)
	}
	binary     string
	prefix     []string
	baseArgs   []string
	runner     emailsvc.CommandRunner
	logger     *log.Logger
	messageLog io.Writer
}

func (c localConfigCompatClient) GetMeta(ctx context.Context, name string) (*connectsvc.Meta, error) {
	if c.inner == nil {
		return nil, errors.New("connect client is required")
	}
	meta, err := c.inner.GetMeta(ctx, name)
	if err != nil {
		return nil, err
	}
	if meta, err = c.ensureDefaultChatID(ctx, meta); err != nil {
		return nil, err
	}
	normalized, err := normalizeLocalMetaConfig(meta)
	if err != nil {
		return nil, err
	}
	c.logResolvedMetaConfig(normalized)
	return normalized, nil
}

func (c localConfigCompatClient) AddRequest(ctx context.Context, input connectsvc.RequestInput) (*connectsvc.Request, error) {
	if c.inner == nil {
		return nil, errors.New("connect client is required")
	}
	return c.inner.AddRequest(ctx, input)
}

func (c localConfigCompatClient) ensureDefaultChatID(ctx context.Context, meta *connectsvc.Meta) (*connectsvc.Meta, error) {
	if meta == nil {
		return nil, errors.New("meta is required")
	}
	if strings.TrimSpace(meta.ChatID) != "" {
		return meta, nil
	}

	key := strings.TrimSpace(meta.Key)
	if key == "" {
		key = emailsvc.DefaultName
	}
	update := []string{
		"meta-update",
		"--key", key,
		"--meta", firstNonEmptyLocal(strings.TrimSpace(meta.Meta), "{}"),
		"--stream", strconv.FormatBool(meta.Stream),
		"--callback", strings.TrimSpace(meta.Callback),
		"--agent", strings.TrimSpace(meta.AgentID),
		"--chatId", emailsvc.DefaultName,
		"--model", strings.TrimSpace(meta.Model),
		"--thinking", strconv.FormatBool(meta.Thinking),
		"--router_disable", strconv.FormatBool(meta.RouterDisable),
	}

	var updated connectsvc.Meta
	if err := c.runJSON(ctx, &updated, update...); err != nil {
		return nil, fmt.Errorf("sync default email chatId: %w", err)
	}
	c.logf("stage=meta-update reason=empty-chatid key=%s chat_id=%s agent_id=%s", quoteLogValueLocalConfig(key), quoteLogValueLocalConfig(updated.ChatID), quoteLogValueLocalConfig(updated.AgentID))
	return &updated, nil
}

func normalizeLocalMetaConfig(meta *connectsvc.Meta) (*connectsvc.Meta, error) {
	if meta == nil {
		return nil, errors.New("meta is required")
	}
	var cfg localMetaConfigCompat
	if err := json.Unmarshal([]byte(meta.Meta), &cfg); err != nil {
		return meta, nil
	}
	cfg.Email = strings.TrimSpace(cfg.Email)
	cfg.EmailPOP3 = strings.TrimSpace(cfg.EmailPOP3)
	cfg.EmailSMTP = strings.TrimSpace(cfg.EmailSMTP)
	cfg.EmailPassword = strings.TrimSpace(cfg.EmailPassword)
	cfg.EmailWhitelist = strings.TrimSpace(cfg.EmailWhitelist)
	if cfg.ScanSeconds <= 0 {
		cfg.ScanSeconds = parseLocalPOP3ScanSeconds(cfg.EmailPOP3Interval, cfg.EmailPOP3Seconds)
	}
	if cfg.ScanSeconds <= 0 {
		cfg.ScanSeconds = defaultPOP3ScanSecsLocal
	}
	body, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	copyMeta := *meta
	copyMeta.Meta = string(body)
	return &copyMeta, nil
}

func (c localConfigCompatClient) runJSON(ctx context.Context, dest any, args ...string) error {
	bin := strings.TrimSpace(c.binary)
	if bin == "" {
		bin = "connect"
	}
	runner := c.runner
	if runner == nil {
		runner = emailsvc.ExecCommandRunner{}
	}
	fullArgs := append([]string{}, c.prefix...)
	fullArgs = append(fullArgs, args...)
	fullArgs = append(fullArgs, c.baseArgs...)
	output, err := runner.Run(ctx, bin, fullArgs...)
	if err != nil {
		return fmt.Errorf("run %s %s failed: %w", bin, formatCommandLocal("", fullArgs), err)
	}
	if err := json.Unmarshal(output, dest); err != nil {
		return fmt.Errorf("decode connect cli output: %w", err)
	}
	return nil
}

func (c localConfigCompatClient) logResolvedMetaConfig(meta *connectsvc.Meta) {
	if meta == nil {
		return
	}
	var cfg localMetaConfigCompat
	if err := json.Unmarshal([]byte(meta.Meta), &cfg); err != nil {
		c.logf("stage=load-config key=%s agent_id=%s chat_id=%s meta=%s", quoteLogValueLocalConfig(firstNonEmptyLocal(meta.Key, emailsvc.DefaultName)), quoteLogValueLocalConfig(meta.AgentID), quoteLogValueLocalConfig(meta.ChatID), quoteLogValueLocalConfig(meta.Meta))
		return
	}
	line := fmt.Sprintf(
		"stage=load-config key=%s agent_id=%s chat_id=%s email_pop3_interval=%s email_pop3_seconds=%s scan_seconds=%d",
		quoteLogValueLocalConfig(firstNonEmptyLocal(meta.Key, emailsvc.DefaultName)),
		quoteLogValueLocalConfig(meta.AgentID),
		quoteLogValueLocalConfig(meta.ChatID),
		quoteLogValueLocalConfig(anyToStringLocal(cfg.EmailPOP3Interval)),
		quoteLogValueLocalConfig(anyToStringLocal(cfg.EmailPOP3Seconds)),
		cfg.ScanSeconds,
	)
	c.logf("%s", line)
	writeTimestampedLocalMessageLog(c.messageLog, c.logger, time.Now().Format(time.RFC3339), line)
}

func (c localConfigCompatClient) logf(format string, args ...any) {
	if c.logger != nil {
		c.logger.Printf(format, args...)
	}
}

func anyToStringLocal(value any) string {
	switch current := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(current)
	case float64:
		if current == float64(int64(current)) {
			return strconv.FormatInt(int64(current), 10)
		}
		return strconv.FormatFloat(current, 'f', -1, 64)
	case int:
		return strconv.Itoa(current)
	case int64:
		return strconv.FormatInt(current, 10)
	default:
		return strings.TrimSpace(fmt.Sprint(current))
	}
}

func quoteLogValueLocalConfig(value string) string {
	return strconv.Quote(strings.TrimSpace(value))
}

func parseLocalPOP3ScanSeconds(values ...any) int {
	for _, raw := range values {
		switch value := raw.(type) {
		case string:
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				continue
			}
			n, err := strconv.Atoi(trimmed)
			if err == nil && n > 0 {
				return n
			}
		case float64:
			if value > 0 && value == float64(int(value)) {
				return int(value)
			}
		case int:
			if value > 0 {
				return value
			}
		}
	}
	return 0
}

func formatCommandLocal(name string, args []string) string {
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
