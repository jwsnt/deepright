package browserplaywrightsvc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const defaultDaemonTimeoutKillThreshold = 3

var forceStopDaemonProcessFn = forceStopDaemonProcess
var daemonStopWaitInterval = 25 * time.Millisecond
var processAliveFn = processAlive

var supportedCommands = []string{
	"command", "help", "name", "param",
	"start", "serve", "stop",
	"open", "attach", "create", "goto", "go-back", "go-forward", "reload", "close",
	"tab-list", "tab-new", "tab-close", "tab-select",
	"click", "dblclick", "fill", "type", "hover", "select", "check", "uncheck",
	"drag", "upload",
	"press", "keydown", "keyup",
	"mousemove", "mousedown", "mouseup", "mousewheel",
	"snapshot", "screenshot", "pdf", "eval", "resize",
	"state-save", "state-load",
	"cookie-list", "cookie-get", "cookie-set", "cookie-delete", "cookie-clear",
	"localstorage-list", "localstorage-get", "localstorage-set", "localstorage-delete", "localstorage-clear",
	"sessionstorage-list", "sessionstorage-get", "sessionstorage-set", "sessionstorage-delete", "sessionstorage-clear",
	"console", "requests", "request", "dialog-accept", "dialog-dismiss",
	"list", "close-all", "kill-all",
}

func RunCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || isHelpCommand(args[0]) {
		printHelp(stdout)
		return 0
	}

	command, rest := normalizeArgs(args)
	flags, positionals, err := parseArgs(rest)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	positionals, flags = normalizeEvalCodeArgs(command, positionals, flags)

	switch command {
	case "name":
		writeJSON(stdout, NameInfo{Key: DefaultName, Name: DefaultDisplayName})
		return 0
	case "param":
		writeJSON(stdout, []string{"session", "browser", "cdp", "profile", "headed", "persistent", "width", "height", "filename", "agentId", "chatId", "instance-bin", "browser-timeout", "browser_retry"})
		return 0
	case "command":
		writeJSON(stdout, supportedCommands)
		return 0
	case "help":
		printHelp(stdout)
		return 0
	case "serve", "__daemon":
		opts, err := optionsFromFlags(flags)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		if err := ServeForeground(opts, stderr); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		return 0
	}

	opts, err := optionsFromFlags(flags)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	switch command {
	case "start":
		result, err := StartDaemon(opts, nil)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		writeJSON(stdout, result)
		return 0
	case "stop":
		result, err := StopDaemon(opts)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		writeJSON(stdout, result)
		return 0
	}

	session := resolveSessionName(flags)
	result, err := runDaemonCommand(opts, CommandRequest{
		Session: session,
		Command: command,
		Args:    positionals,
		Flags:   flags,
	})
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	writeJSON(stdout, result)
	if result.Message != "" && command == "snapshot" {
		fmt.Fprintln(stdout, result.Message)
	}
	return 0
}

func resolveSessionName(flags map[string]string) string {
	flags = normalizeManagedIdentityFlags(flags)
	if session := strings.TrimSpace(flags["session"]); session != "" {
		return session
	}
	agentID := firstIdentityValue(flags["agentId"], flags["agent"])
	chatID := firstIdentityValue(flags["chatId"], flags["chat"])
	if agentID != "" && chatID != "" {
		return instanceSessionName(agentID, chatID)
	}
	return "default"
}

func StartDaemon(opts Options, launchPrefix []string) (*StartResult, error) {
	normalized, err := normalizeOptions(opts)
	if err != nil {
		return nil, err
	}
	if daemonOwnedByCurrentRuntime(normalized) && waitForDaemonReady(normalized.Addr, 300*time.Millisecond) {
		pid := 0
		if value, err := readPIDFile(normalized.PIDFile); err == nil {
			pid = value
		}
		return &StartResult{
			Status:  "already_running",
			PID:     pid,
			PIDFile: normalized.PIDFile,
			LogFile: normalized.LogFile,
			Addr:    normalized.Addr,
		}, nil
	}
	if pid, err := readPIDFile(normalized.PIDFile); err == nil && processAliveFn(pid) && waitForDaemonReady(normalized.Addr, 500*time.Millisecond) {
		return &StartResult{
			Status:  "already_running",
			PID:     pid,
			PIDFile: normalized.PIDFile,
			LogFile: normalized.LogFile,
			Addr:    normalized.Addr,
		}, nil
	}
	if waitForDaemonReady(normalized.Addr, 300*time.Millisecond) {
		_ = forceStopDaemonByAddr(normalized.Addr)
		time.Sleep(150 * time.Millisecond)
	}

	writer, err := logWriter(normalized.LogFile)
	if err == nil {
		_ = writer.Close()
	}

	exe := strings.TrimSpace(normalized.ExecutablePath)
	if exe == "" {
		var err error
		exe, err = os.Executable()
		if err != nil {
			return nil, err
		}
	}
	args := append([]string{}, launchPrefix...)
	args = append(args, "__daemon")
	args = append(args, "--state-dir", normalized.StateDir, "--addr", normalized.Addr, "--pid-file", normalized.PIDFile, "--log-file", normalized.LogFile)
	cmd := exec.Command(exe, args...)
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return nil, err
	}
	defer devNull.Close()
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	configureDetachedDaemonCommand(cmd)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()

	if !waitForDaemonReady(normalized.Addr, normalized.BrowserTimeout) {
		_ = forceStopDaemonProcess(normalized.PIDFile)
		return nil, fmt.Errorf("browser_playwright start timed out after %s waiting for daemon at %s", normalized.BrowserTimeout, normalized.Addr)
	}
	return &StartResult{
		Status:  "started",
		PID:     pid,
		PIDFile: normalized.PIDFile,
		LogFile: normalized.LogFile,
		Addr:    normalized.Addr,
	}, nil
}

func daemonOwnedByCurrentRuntime(opts Options) bool {
	if strings.TrimSpace(opts.PIDFile) == "" || strings.TrimSpace(opts.StateDir) == "" {
		return false
	}
	pid, err := readPIDFile(opts.PIDFile)
	if err != nil || !processAliveFn(pid) {
		return false
	}
	var meta daemonMeta
	if err := readJSONFile(metaPath(opts.StateDir), &meta); err != nil {
		return false
	}
	return meta.PID == pid && strings.TrimSpace(meta.Addr) == strings.TrimSpace(opts.Addr)
}

func forceStopDaemonByAddr(addr string) error {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil
	}
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(strings.TrimSpace(portText))
	if err != nil || port <= 0 {
		return fmt.Errorf("invalid daemon addr: %s", addr)
	}
	if strings.TrimSpace(host) == "" {
		host = "127.0.0.1"
	}
	out, err := exec.Command("lsof", "-nP", "-iTCP:"+strconv.Itoa(port), "-sTCP:LISTEN", "-t").Output()
	if err != nil {
		return nil
	}
	lines := strings.Fields(string(out))
	for _, line := range lines {
		pid, convErr := strconv.Atoi(strings.TrimSpace(line))
		if convErr != nil || pid <= 0 {
			continue
		}
		proc, findErr := os.FindProcess(pid)
		if findErr != nil {
			continue
		}
		_ = proc.Signal(syscall.SIGTERM)
		deadline := time.Now().Add(500 * time.Millisecond)
		for time.Now().Before(deadline) {
			if !processAliveFn(pid) {
				break
			}
			time.Sleep(25 * time.Millisecond)
		}
		if processAliveFn(pid) {
			_ = proc.Signal(syscall.SIGKILL)
		}
	}
	return nil
}

func StopDaemon(opts Options) (*StartResult, error) {
	normalized, err := normalizeOptions(opts)
	if err != nil {
		return nil, err
	}
	pid, err := readPIDFile(normalized.PIDFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &StartResult{Status: "stopped", PIDFile: normalized.PIDFile, LogFile: normalized.LogFile, Addr: normalized.Addr}, nil
		}
		return nil, err
	}
	if !processAliveFn(pid) {
		_ = os.Remove(normalized.PIDFile)
		return &StartResult{Status: "stopped", PID: pid, PIDFile: normalized.PIDFile, LogFile: normalized.LogFile, Addr: normalized.Addr}, nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		_ = os.Remove(normalized.PIDFile)
		return &StartResult{Status: "stopped", PID: pid, PIDFile: normalized.PIDFile, LogFile: normalized.LogFile, Addr: normalized.Addr}, nil
	}
	_ = proc.Signal(syscall.SIGTERM)
	if waitForProcessExit(pid, normalized.BrowserTimeout) {
		_ = os.Remove(normalized.PIDFile)
		return &StartResult{Status: "stopped", PID: pid, PIDFile: normalized.PIDFile, LogFile: normalized.LogFile, Addr: normalized.Addr}, nil
	}
	_ = proc.Signal(syscall.SIGKILL)
	if waitForProcessExit(pid, 500*time.Millisecond) {
		_ = os.Remove(normalized.PIDFile)
		return &StartResult{Status: "stopped", PID: pid, PIDFile: normalized.PIDFile, LogFile: normalized.LogFile, Addr: normalized.Addr}, nil
	}
	return nil, fmt.Errorf("browser_playwright stop timed out after %s waiting for pid=%d", normalized.BrowserTimeout+500*time.Millisecond, pid)
}

func ServeForeground(opts Options, stderr io.Writer) error {
	normalized, err := normalizeOptions(opts)
	if err != nil {
		return err
	}
	writer, err := logWriter(normalized.LogFile)
	if err != nil {
		return err
	}
	defer writer.Close()
	logger := log.New(io.MultiWriter(stderr, writer), "[browser_playwright] ", log.LstdFlags)
	logger.Printf("用参数：进程=%d；父进程=%d；地址=%s；状态目录=%s，做了：启动浏览器后台服务", os.Getpid(), os.Getppid(), normalized.Addr, normalized.StateDir)
	defer func() {
		if r := recover(); r != nil {
			logger.Printf("用参数：进程=%d；异常=%v，做了：浏览器后台服务发生了异常", os.Getpid(), r)
			panic(r)
		}
		logger.Printf("用参数：进程=%d，做了：浏览器后台服务退出", os.Getpid())
	}()
	if err := writePIDFile(normalized.PIDFile, os.Getpid()); err != nil {
		return err
	}
	defer os.Remove(normalized.PIDFile)
	if err := writeJSONFile(metaPath(normalized.StateDir), daemonMeta{
		Addr:      normalized.Addr,
		PID:       os.Getpid(),
		UpdatedAt: time.Now(),
	}); err != nil {
		return err
	}

	svc := newDaemonService(normalized, logger)
	server := &http.Server{
		Addr:    normalized.Addr,
		Handler: http.HandlerFunc(svc.serveHTTP),
	}

	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	defer signal.Stop(signalCh)
	go func() {
		for sig := range signalCh {
			logger.Printf("用参数：进程=%d；信号=%s，做了：浏览器后台服务收到了退出信号", os.Getpid(), sig.String())
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		logger.Printf("用参数：进程=%d；原因=%v，做了：开始关闭浏览器后台服务", os.Getpid(), ctx.Err())
		if err := svc.shutdown(); err != nil {
			logger.Printf("用参数：进程=%d；原因=%v，做了：关闭浏览器后台服务时清理资源失败", os.Getpid(), err)
		}
		_ = server.Shutdown(context.Background())
	}()
	logger.Printf("用参数：地址=%s，做了：浏览器后台服务开始监听请求", normalized.Addr)
	err = server.ListenAndServe()
	logger.Printf("用参数：进程=%d；结果=%v，做了：浏览器后台服务结束监听", os.Getpid(), err)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func runDaemonCommand(opts Options, req CommandRequest) (*CommandResult, error) {
	normalized, err := normalizeOptions(opts)
	if err != nil {
		return nil, err
	}
	if _, err := StartDaemon(normalized, nil); err != nil {
		return nil, err
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), normalized.BrowserTimeout)
	defer cancel()

	resp, err := doDaemonCommandWithRetry(ctx, normalized, data)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, handleDaemonCommandTimeout(normalized, req.Command)
		}
		_ = clearDaemonCommandTimeoutState(normalized.StateDir)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		_ = clearDaemonCommandTimeoutState(normalized.StateDir)
		var item map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&item)
		return nil, fmt.Errorf("%v", item["error"])
	}
	var result CommandResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, handleDaemonCommandTimeout(normalized, req.Command)
		}
		_ = clearDaemonCommandTimeoutState(normalized.StateDir)
		return nil, err
	}
	if err := clearDaemonCommandTimeoutState(normalized.StateDir); err != nil {
		return nil, err
	}
	return &result, nil
}

func doDaemonCommandWithRetry(ctx context.Context, opts Options, data []byte) (*http.Response, error) {
	var lastErr error
	attempt := 0
	for {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+opts.Addr+"/command", bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(httpReq)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !shouldRetryDaemonCommand(err) || ctx.Err() != nil {
			break
		}
		attempt++
		delay := daemonCommandRetryDelay(attempt)
		if deadline, ok := ctx.Deadline(); ok {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				break
			}
			if delay >= remaining {
				delay = minDuration(remaining/4, 250*time.Millisecond)
			}
			if delay <= 0 {
				break
			}
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, lastErr
		case <-timer.C:
		}
	}
	return nil, lastErr
}

func daemonCommandRetryDelay(attempt int) time.Duration {
	if attempt <= 1 {
		return 100 * time.Millisecond
	}
	delay := time.Duration(attempt) * 150 * time.Millisecond
	if delay > time.Second {
		return time.Second
	}
	return delay
}

func minDuration(a, b time.Duration) time.Duration {
	if a <= 0 {
		return b
	}
	if b <= 0 {
		return a
	}
	if a < b {
		return a
	}
	return b
}

func shouldRetryDaemonCommand(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "eof") ||
		strings.Contains(text, "connection refused") ||
		strings.Contains(text, "connection reset by peer") ||
		strings.Contains(text, "broken pipe")
}

type HelpManualOptions struct {
	BinaryName         string
	DaemonInvocation   string
	DaemonLabelPrefix  string
	IncludeMetadata    bool
	IncludeUsageHeader bool
}

func printHelp(w io.Writer) {
	WriteHelpManual(w, HelpManualOptions{
		BinaryName:         "browser_playwright",
		DaemonInvocation:   "browser_playwright",
		IncludeMetadata:    true,
		IncludeUsageHeader: true,
	})
}

func WriteHelpManual(w io.Writer, opts HelpManualOptions) {
	binaryName := strings.TrimSpace(opts.BinaryName)
	if binaryName == "" {
		binaryName = "browser_playwright"
	}
	daemonInvocation := strings.TrimSpace(opts.DaemonInvocation)
	if daemonInvocation == "" {
		daemonInvocation = binaryName
	}
	daemonLabelPrefix := opts.DaemonLabelPrefix

	if opts.IncludeUsageHeader {
		fmt.Fprintln(w, "Usage:")
		fmt.Fprintf(w, "  %s help\n", binaryName)
		fmt.Fprintf(w, "  %s start [--state-dir DIR] [--addr HOST:PORT]\n", daemonInvocation)
		fmt.Fprintf(w, "  %s stop [--state-dir DIR]\n", daemonInvocation)
		fmt.Fprintf(w, "  %s [--session NAME] open [url] [--headed] [--persistent]\n", binaryName)
		fmt.Fprintf(w, "  %s [--session NAME] attach --cdp=chrome\n", binaryName)
		fmt.Fprintf(w, "  %s create --agentId AGENT --chatId CHAT\n", binaryName)
		fmt.Fprintf(w, "  %s [--session NAME] attach --cdp=http://127.0.0.1:9222\n", binaryName)
		fmt.Fprintf(w, "  %s [--session NAME] <command> [args] [--flags]\n", binaryName)
		fmt.Fprintln(w, "")
	}

	fmt.Fprintln(w, "Manual:")
	fmt.Fprintln(w, "  browser_playwright is a Go CLI built on playwright-go. It auto-starts a local daemon")
	fmt.Fprintln(w, "  so sequential commands share one live browser session, matching the Playwright CLI flow.")
	fmt.Fprintln(w, "  It also auto-loads Chrome cookies through the cookie module and injects cookies")
	fmt.Fprintln(w, "  matching the current page domain for navigation commands.")
	fmt.Fprintln(w, "")
	if opts.IncludeMetadata {
		fmt.Fprintln(w, "Metadata:")
		fmt.Fprintln(w, "  name       print plugin key and display name")
		fmt.Fprintln(w, "  param      print common runtime parameter keys")
		fmt.Fprintln(w, "  command    print supported command list")
		fmt.Fprintln(w, "  help       print full manual")
		fmt.Fprintln(w, "")
	}
	fmt.Fprintln(w, "Daemon:")
	fmt.Fprintf(w, "  %sstart      start local browser_playwright daemon in background\n", daemonLabelPrefix)
	fmt.Fprintf(w, "  %sstop       stop local browser_playwright daemon\n", daemonLabelPrefix)
	fmt.Fprintf(w, "  %sserve      run daemon in foreground\n", daemonLabelPrefix)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Core:")
	fmt.Fprintln(w, "  open [url]                 open browser session and optionally navigate")
	fmt.Fprintln(w, "  attach                     attach to existing Chromium via CDP")
	fmt.Fprintln(w, "  create                     create/reuse CDP via browser_instance then attach")
	fmt.Fprintln(w, "  goto <url>                 navigate active tab")
	fmt.Fprintln(w, "  go-back | go-forward       browser navigation")
	fmt.Fprintln(w, "  reload                     reload active tab")
	fmt.Fprintln(w, "  close                      close current session")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Tabs:")
	fmt.Fprintln(w, "  tab-list                   list tabs")
	fmt.Fprintln(w, "  tab-new [url]              create a new tab")
	fmt.Fprintln(w, "  tab-close [index]          close tab")
	fmt.Fprintln(w, "  tab-select <index>         switch active tab")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Actions:")
	fmt.Fprintln(w, "  click <ref|selector|text>")
	fmt.Fprintln(w, "  dblclick <ref|selector|text>")
	fmt.Fprintln(w, "  fill <ref|selector|text> <value>")
	fmt.Fprintln(w, "  type <text>                type into current focus")
	fmt.Fprintln(w, "  hover <ref|selector|text>")
	fmt.Fprintln(w, "  select <ref|selector|text> <value>")
	fmt.Fprintln(w, "  check <ref|selector|text>")
	fmt.Fprintln(w, "  uncheck <ref|selector|text>")
	fmt.Fprintln(w, "  drag <startRef> <endRef>")
	fmt.Fprintln(w, "  upload <file...> [--selector CSS]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Keyboard / Mouse:")
	fmt.Fprintln(w, "  press <key>")
	fmt.Fprintln(w, "  keydown <key>")
	fmt.Fprintln(w, "  keyup <key>")
	fmt.Fprintln(w, "  mousemove <x> <y>")
	fmt.Fprintln(w, "  mousedown")
	fmt.Fprintln(w, "  mouseup")
	fmt.Fprintln(w, "  mousewheel <dx> <dy>")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Capture / State:")
	fmt.Fprintln(w, "  snapshot [target] [--filename FILE]")
	fmt.Fprintln(w, "  screenshot [target] [--filename FILE]")
	fmt.Fprintln(w, "  pdf [--filename FILE]")
	fmt.Fprintln(w, "  eval <js> [target]")
	fmt.Fprintln(w, "  eval --code <js> [target]")
	fmt.Fprintln(w, "  resize <w> <h>")
	fmt.Fprintln(w, "  state-save [file]")
	fmt.Fprintln(w, "  state-load <file>")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Storage:")
	fmt.Fprintln(w, "  cookie-list | cookie-get <name> | cookie-set <name> <value> | cookie-delete <name> | cookie-clear")
	fmt.Fprintln(w, "  localstorage-list | localstorage-get <key> | localstorage-set <key> <value> | localstorage-delete <key> | localstorage-clear")
	fmt.Fprintln(w, "  sessionstorage-list | sessionstorage-get <key> | sessionstorage-set <key> <value> | sessionstorage-delete <key> | sessionstorage-clear")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Logs:")
	fmt.Fprintln(w, "  console [level]")
	fmt.Fprintln(w, "  requests")
	fmt.Fprintln(w, "  request <index>")
	fmt.Fprintln(w, "  dialog-accept [prompt]")
	fmt.Fprintln(w, "  dialog-dismiss")
	fmt.Fprintln(w, "  list | close-all | kill-all")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --session NAME             session name, default default")
	fmt.Fprintln(w, "  --state-dir DIR            state root, default ./.browser_playwright")
	fmt.Fprintln(w, "  --addr HOST:PORT           daemon address, default 127.0.0.1:18333")
	fmt.Fprintln(w, "  --driver-dir DIR           playwright driver directory, default ./playwright/driver beside executable")
	fmt.Fprintln(w, "  --headed                   launch headed browser")
	fmt.Fprintln(w, "  --persistent               keep persistent profile directory")
	fmt.Fprintln(w, "  --browser NAME             chromium|firefox|webkit, default chromium")
	fmt.Fprintln(w, "  --cdp TARGET               attach over CDP, use chrome or a CDP URL")
	fmt.Fprintln(w, "  --agentId AGENT            create command agent identity")
	fmt.Fprintln(w, "  --chatId CHAT              create command chat identity; create session is always Agent@Chat")
	fmt.Fprintln(w, "  --instance-bin PATH        override sibling ../instance/browser_instance path")
	fmt.Fprintln(w, "  --timeout MS               Playwright action timeout in milliseconds, default 5000")
	fmt.Fprintln(w, "  --navigation-timeout MS    Playwright navigation timeout in milliseconds, default 60000")
	fmt.Fprintln(w, "  --browser-timeout VALUE    wall-clock timeout, default 120s; plain numbers mean seconds")
	fmt.Fprintln(w, "  --browser_retry N          consecutive timeout count before daemon is killed, default 3")
	fmt.Fprintln(w, "  --profile PATH             custom profile dir")
	fmt.Fprintln(w, "  --width N --height N       viewport size, default 2560x1440")
	fmt.Fprintln(w, "  --filename FILE            output file for snapshot/screenshot/pdf")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintf(w, "  ./%s open https://example.com --headed\n", binaryName)
	fmt.Fprintf(w, "  ./%s attach --cdp=chrome\n", binaryName)
	fmt.Fprintf(w, "  ./%s create --agentId demo-agent --chatId chat-001\n", binaryName)
	fmt.Fprintf(w, "  ./%s --session demo-agent@chat-001 goto https://example.com\n", binaryName)
	fmt.Fprintf(w, "  ./%s attach --cdp=http://127.0.0.1:9222\n", binaryName)
	fmt.Fprintf(w, "  ./%s snapshot\n", binaryName)
	fmt.Fprintf(w, "  ./%s click e12\n", binaryName)
	fmt.Fprintf(w, "  ./%s fill e18 \"hello world\"\n", binaryName)
	fmt.Fprintf(w, "  ./%s --session demo-agent@chat-001 --timeout 15000 --browser-timeout 30s eval 'new Promise(resolve => setTimeout(resolve, 10000))'\n", binaryName)
	fmt.Fprintf(w, "  ./%s --session demo-agent@chat-001 eval --code 'document.title'\n", binaryName)
	fmt.Fprintf(w, "  ./%s screenshot --filename page.png\n", binaryName)
	fmt.Fprintf(w, "  ./%s tab-new https://playwright.dev\n", binaryName)
	fmt.Fprintf(w, "  ./%s cookie-list\n", binaryName)
	fmt.Fprintf(w, "  ./%s localstorage-set token abc\n", binaryName)
	fmt.Fprintf(w, "  ./%s requests\n", binaryName)
}

func optionsFromFlags(flags map[string]string) (Options, error) {
	timeout, err := browserTimeoutFromFlags(flags)
	if err != nil {
		return Options{}, err
	}
	return Options{
		StateDir:       strings.TrimSpace(flags["state-dir"]),
		Addr:           strings.TrimSpace(flags["addr"]),
		LogFile:        strings.TrimSpace(flags["log-file"]),
		PIDFile:        strings.TrimSpace(flags["pid-file"]),
		DriverDir:      strings.TrimSpace(flags["driver-dir"]),
		BrowserTimeout: timeout,
		BrowserRetry:   browserRetryFromFlags(flags),
	}, nil
}

func browserRetryFromFlags(flags map[string]string) int {
	value, ok := cliIntFlagValue(flags, "browser_retry")
	if !ok || value <= 0 {
		return defaultDaemonTimeoutKillThreshold
	}
	return value
}

func cliIntFlagValue(flags map[string]string, key string) (int, bool) {
	raw, ok := flags[key]
	if !ok || strings.TrimSpace(raw) == "" {
		return 0, false
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, false
	}
	return value, true
}

func browserTimeoutFromFlags(flags map[string]string) (time.Duration, error) {
	raw := strings.TrimSpace(flags["browser-timeout"])
	if raw == "" {
		return defaultBrowserTimeout, nil
	}
	if seconds, err := strconv.ParseFloat(raw, 64); err == nil {
		if seconds <= 0 {
			return 0, fmt.Errorf("--browser-timeout must be positive")
		}
		return time.Duration(seconds * float64(time.Second)), nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid --browser-timeout %q", raw)
	}
	if value <= 0 {
		return 0, fmt.Errorf("--browser-timeout must be positive")
	}
	return value, nil
}

func parseArgs(args []string) (map[string]string, []string, error) {
	flags := map[string]string{}
	positionals := make([]string, 0)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--") {
			key := strings.TrimPrefix(arg, "--")
			value := "true"
			if parts := strings.SplitN(key, "=", 2); len(parts) == 2 {
				key = parts[0]
				value = parts[1]
			} else if isValueFlag(key) && i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				value = args[i+1]
				i++
			}
			flags[key] = value
			continue
		}
		positionals = append(positionals, arg)
	}
	return normalizeManagedIdentityFlags(flags), positionals, nil
}

func normalizeEvalCodeArgs(command string, positionals []string, flags map[string]string) ([]string, map[string]string) {
	if strings.TrimSpace(command) != "eval" {
		return positionals, flags
	}
	if len(positionals) > 0 {
		return positionals, flags
	}
	value, ok := flags["code"]
	if !ok {
		return positionals, flags
	}
	nextFlags := make(map[string]string, len(flags))
	for key, item := range flags {
		nextFlags[key] = item
	}
	delete(nextFlags, "code")
	return []string{value}, nextFlags
}

func normalizeArgs(args []string) (string, []string) {
	if len(args) == 0 {
		return "help", nil
	}
	rest := append([]string{}, args...)
	commandIndex := -1
	for i := 0; i < len(rest); i++ {
		arg := rest[i]
		if strings.HasPrefix(arg, "--") {
			key := strings.TrimPrefix(arg, "--")
			if parts := strings.SplitN(key, "=", 2); len(parts) == 2 {
				continue
			}
			if isValueFlag(key) {
				if i+1 < len(rest) && !strings.HasPrefix(rest[i+1], "--") {
					i++
				}
			}
			continue
		}
		commandIndex = i
		break
	}
	if commandIndex == -1 {
		return "help", rest
	}
	command := rest[commandIndex]
	rest = append(rest[:commandIndex], rest[commandIndex+1:]...)
	return strings.TrimSpace(command), rest
}

func isHelpCommand(arg string) bool {
	switch strings.TrimSpace(arg) {
	case "help", "--help", "-h":
		return true
	default:
		return false
	}
}

func isValueFlag(key string) bool {
	switch strings.TrimSpace(key) {
	case "session", "state-dir", "addr", "browser", "cdp", "profile", "width", "height", "filename", "selector", "timeout", "navigation-timeout", "channel", "pid-file", "log-file", "driver-dir", "user-agent", "agentId", "chatId", "agent", "chat", "instance-bin", "instance-state", "instance-obscura", "instance-monitor", "instance-monitor-ms", "browser-timeout", "browser_retry", "code":
		return true
	default:
		return false
	}
}

func forceStopDaemonProcess(pidFile string) error {
	pid, err := readPIDFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	_ = proc.Signal(syscall.SIGTERM)
	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		if !processAliveFn(pid) {
			_ = os.Remove(pidFile)
			return nil
		}
		time.Sleep(25 * time.Millisecond)
	}
	if processAliveFn(pid) {
		_ = proc.Signal(syscall.SIGKILL)
	}
	_ = os.Remove(pidFile)
	return nil
}

func waitForProcessExit(pid int, timeout time.Duration) bool {
	if pid <= 0 {
		return true
	}
	if timeout <= 0 {
		return !processAliveFn(pid)
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processAliveFn(pid) {
			return true
		}
		time.Sleep(daemonStopWaitInterval)
	}
	return !processAliveFn(pid)
}

type daemonCommandTimeoutState struct {
	ConsecutiveTimeouts int       `json:"consecutiveTimeouts"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

func handleDaemonCommandTimeout(opts Options, command string) error {
	count, err := recordDaemonCommandTimeout(opts.StateDir)
	if err != nil {
		return fmt.Errorf("browser_playwright %s timed out after %s (also failed to update timeout state: %v)", command, opts.BrowserTimeout, err)
	}
	threshold := opts.BrowserRetry
	if threshold <= 0 {
		threshold = defaultDaemonTimeoutKillThreshold
	}
	if count >= threshold {
		_ = forceStopDaemonProcessFn(opts.PIDFile)
		_ = clearDaemonCommandTimeoutState(opts.StateDir)
		return fmt.Errorf("browser_playwright %s timed out after %s; daemon stopped after %d consecutive timeouts", command, opts.BrowserTimeout, count)
	}
	return fmt.Errorf("browser_playwright %s timed out after %s (consecutive timeouts: %d/%d)", command, opts.BrowserTimeout, count, threshold)
}

func recordDaemonCommandTimeout(stateDir string) (int, error) {
	path := timeoutStatePath(stateDir)
	var state daemonCommandTimeoutState
	if err := readJSONFile(path, &state); err != nil && !os.IsNotExist(err) {
		return 0, err
	}
	state.ConsecutiveTimeouts++
	state.UpdatedAt = time.Now()
	if err := writeJSONFile(path, state); err != nil {
		return 0, err
	}
	return state.ConsecutiveTimeouts, nil
}

func clearDaemonCommandTimeoutState(stateDir string) error {
	path := timeoutStatePath(stateDir)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func writeJSON(w io.Writer, value any) {
	data, _ := json.MarshalIndent(value, "", "  ")
	fmt.Fprintln(w, string(data))
}
