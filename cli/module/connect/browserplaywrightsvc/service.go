package browserplaywrightsvc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
	"gopkg.in/natefinch/lumberjack.v2"
)

type daemonService struct {
	opts     Options
	logger   *log.Logger
	mu       sync.Mutex
	pw       *playwright.Playwright
	sessions map[string]*liveSession
}

type liveSession struct {
	config     SessionConfig
	browser    playwright.Browser
	context    playwright.BrowserContext
	pages      []playwright.Page
	active     int
	lastURL    string
	lastTitle  string
	updatedAt  time.Time
	consoleLog []ConsoleEntry
	requests   []RequestInfo
	dialog     playwright.Dialog
	injected   map[string]struct{}
}

type fingerprintProfile struct {
	UserAgent      string
	Platform       string
	Locale         string
	Languages      []string
	TimezoneID     string
	ViewportWidth  int
	ViewportHeight int
	ScreenWidth    int
	ScreenHeight   int
	MaxTouchPoints int
	WebGLVendor    string
	WebGLRenderer  string
}

var (
	systemLocaleValuesFn = detectSystemLocaleValues
	systemTimezoneIDFn   = detectSystemTimezoneID
)

var (
	playwrightRunFn = func(opts *playwright.RunOptions) (*playwright.Playwright, error) {
		return playwright.Run(opts)
	}
	playwrightInstallFn = func(opts *playwright.RunOptions) error {
		return playwright.Install(opts)
	}
	playwrightStopRuntimeFn = func(pw *playwright.Playwright) error {
		return pw.Stop()
	}
)

func newDaemonService(opts Options, logger *log.Logger) *daemonService {
	return &daemonService{
		opts:     opts,
		logger:   logger,
		sessions: map[string]*liveSession{},
	}
}

func (d *daemonService) ensurePlaywright() error {
	if d.pw != nil {
		return nil
	}
	runOptions := d.playwrightRunOptions()
	driverReady, err := playwrightDriverReady(strings.TrimSpace(d.opts.DriverDir))
	if err != nil {
		return err
	}
	if !driverReady {
		if d.logger != nil {
			d.logger.Printf("用参数：驱动目录=%s，做了：发现浏览器驱动缺失，开始自动安装", strings.TrimSpace(d.opts.DriverDir))
		}
		if err := InstallPlaywrightDriver(strings.TrimSpace(d.opts.DriverDir)); err != nil {
			return err
		}
	}
	if _, err := ensurePlaywrightNodeEnvironment(strings.TrimSpace(d.opts.DriverDir)); err != nil {
		return err
	}
	pw, err := playwrightRunFn(runOptions)
	if err != nil {
		if !driverReady && errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("playwright driver install incomplete for driverDir=%s: %w", strings.TrimSpace(d.opts.DriverDir), err)
		}
		return err
	}
	d.pw = pw
	return nil
}

func (d *daemonService) playwrightRunOptions() *playwright.RunOptions {
	return &playwright.RunOptions{
		DriverDirectory: strings.TrimSpace(d.opts.DriverDir),
		// We only need the Playwright driver runtime here. Browser binaries come
		// from the managed Chrome/CDP flow or the local system browser.
		SkipInstallBrowsers: true,
	}
}

func (d *daemonService) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if d.logger != nil {
		d.logger.Printf("用参数：请求方法=%s；请求路径=%s；来源=%s，做了：开始处理浏览器后台请求", r.Method, r.URL.Path, r.RemoteAddr)
	}
	switch r.URL.Path {
	case "/healthz":
		writeHTTPJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	case "/command":
	default:
		writeHTTPJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	if r.Method != http.MethodPost {
		writeHTTPJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeHTTPJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	result, err := d.execute(req)
	if err != nil {
		writeHTTPJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeHTTPJSON(w, http.StatusOK, result)
}

func (d *daemonService) execute(req CommandRequest) (*CommandResult, error) {
	startedAt := time.Now()
	sessionName := sanitizeName(req.Session)
	if sessionName == "" {
		sessionName = "default"
	}
	d.logCommandDiagnostic("start", sessionName, req, nil, nil, 0)

	d.mu.Lock()
	defer d.mu.Unlock()

	switch req.Command {
	case "list":
		items, err := d.listSessions()
		if err != nil {
			d.logCommandDiagnostic("error", sessionName, req, nil, err, time.Since(startedAt))
			return nil, err
		}
		result := &CommandResult{Value: items, Command: req.Command}
		d.logCommandDiagnostic("finish", sessionName, req, result, nil, time.Since(startedAt))
		return result, nil
	case "close-all":
		for name := range d.sessions {
			_ = d.closeSessionWithReason(name, "close_all_command")
		}
		result := &CommandResult{Command: req.Command, Message: "closed all sessions"}
		d.logCommandDiagnostic("finish", sessionName, req, result, nil, time.Since(startedAt))
		return result, nil
	case "kill-all":
		for name := range d.sessions {
			_ = d.closeSessionWithReason(name, "kill_all_command")
		}
		result := &CommandResult{Command: req.Command, Message: "killed all sessions"}
		d.logCommandDiagnostic("finish", sessionName, req, result, nil, time.Since(startedAt))
		return result, nil
	}

	if req.Command == "create" {
		flags, err := prepareCreateFlags(req.Flags)
		if err != nil {
			d.logCommandDiagnostic("error", sessionName, req, nil, err, time.Since(startedAt))
			return nil, err
		}
		req.Flags = flags
		req.Session = flags["session"]
	} else {
		flags, managed, err := prepareManagedFlags(req.Flags)
		if err != nil {
			d.logCommandDiagnostic("error", sessionName, req, nil, err, time.Since(startedAt))
			return nil, err
		}
		if managed {
			req.Flags = flags
			req.Session = flags["session"]
		}
	}
	sessionName = sanitizeName(req.Session)
	if sessionName == "" {
		sessionName = "default"
	}
	if err := d.ensurePlaywright(); err != nil {
		d.logCommandDiagnostic("error", sessionName, req, nil, err, time.Since(startedAt))
		return nil, err
	}
	if req.Command == "attach" || req.Command == "create" {
		closeReason := "session_replace_before_" + strings.TrimSpace(req.Command)
		if err := d.closeSessionWithReason(sessionName, closeReason); err != nil {
			d.logCommandDiagnostic("error", sessionName, req, nil, err, time.Since(startedAt))
			return nil, err
		}
	}
	sess, err := d.ensureSession(sessionName, req.Command, req.Flags)
	if err != nil {
		d.logCommandDiagnostic("error", sessionName, req, nil, err, time.Since(startedAt))
		return nil, err
	}
	result, err := d.runSessionCommand(sessionName, sess, req)
	if err != nil {
		d.logCommandDiagnostic("error", sessionName, req, nil, err, time.Since(startedAt))
		return nil, err
	}
	if err := d.persistSessionSummary(sessionName, sess); err != nil {
		d.logCommandDiagnostic("error", sessionName, req, nil, err, time.Since(startedAt))
		return nil, err
	}
	d.logCommandDiagnostic("finish", sessionName, req, result, nil, time.Since(startedAt))
	return result, nil
}

func (d *daemonService) ensureSession(name, command string, flags map[string]string) (*liveSession, error) {
	if existing := d.sessions[name]; existing != nil {
		return existing, nil
	}
	config, err := d.loadOrCreateConfig(name, flags)
	if err != nil {
		return nil, err
	}
	if (command == "attach" || command == "create") && strings.TrimSpace(config.CDP) == "" {
		return nil, fmt.Errorf("attach requires --cdp=chrome or --cdp=<url>")
	}
	if d.logger != nil {
		d.logger.Printf("用参数：会话=%s；命令=%s；阶段=ensure_session_launch，做了：开始附着或创建浏览器会话", name, command)
	}
	fp := fingerprintProfileFromConfig(config)
	ctx, browser, err := d.launchContext(config)
	if err != nil {
		return nil, err
	}
	if d.logger != nil {
		d.logger.Printf("用参数：会话=%s；命令=%s；阶段=ensure_session_context_ready；页面数=%d，做了：已拿到浏览器上下文", name, command, len(ctx.Pages()))
	}
	pages := ctx.Pages()
	if len(pages) == 0 {
		if d.logger != nil {
			d.logger.Printf("用参数：会话=%s；命令=%s；阶段=ensure_session_new_page，做了：当前上下文没有页面，准备新建页面", name, command)
		}
		page, err := ctx.NewPage()
		if err != nil {
			if browser != nil {
				_ = browser.Close()
			} else {
				_ = ctx.Close()
			}
			return nil, err
		}
		if strings.TrimSpace(config.UserAgent) != "" {
			if err := d.applyFingerprintOverride(ctx, page, fp, strings.TrimSpace(config.CDP) != ""); err != nil {
				if browser != nil {
					_ = browser.Close()
				} else {
					_ = ctx.Close()
				}
				return nil, err
			}
		}
		pages = append(pages, page)
	}
	sess := &liveSession{
		config:    config,
		browser:   browser,
		context:   ctx,
		pages:     pages,
		active:    0,
		updatedAt: time.Now(),
		injected:  map[string]struct{}{},
	}
	d.attachSessionHooks(sess)
	if d.logger != nil {
		d.logger.Printf("用参数：会话=%s；命令=%s；阶段=ensure_session_sync_pages，做了：开始同步页面信息", name, command)
	}
	d.syncPages(sess)
	if d.logger != nil {
		d.logger.Printf("用参数：会话=%s；命令=%s；阶段=ensure_session_ready；页面数=%d；活动页=%d，做了：浏览器会话已经可用", name, command, len(sess.pages), sess.active)
	}
	d.sessions[name] = sess
	return sess, nil
}

func (d *daemonService) attachSessionHooks(sess *liveSession) {
	sess.context.OnPage(func(page playwright.Page) {
		sess.pages = sess.context.Pages()
	})
	sess.context.OnConsole(func(msg playwright.ConsoleMessage) {
		entry := ConsoleEntry{
			Type:      msg.Type(),
			Text:      msg.Text(),
			CreatedAt: time.Now(),
		}
		if loc := msg.Location(); loc != nil {
			entry.URL = loc.URL
			entry.Line = loc.LineNumber
			entry.Column = loc.ColumnNumber
		}
		sess.consoleLog = append(sess.consoleLog, entry)
		if len(sess.consoleLog) > 200 {
			sess.consoleLog = sess.consoleLog[len(sess.consoleLog)-200:]
		}
	})
	sess.context.OnDialog(func(dialog playwright.Dialog) {
		sess.dialog = dialog
	})
	sess.context.OnRequest(func(req playwright.Request) {
		info := RequestInfo{
			Index:    len(sess.requests) + 1,
			URL:      req.URL(),
			Method:   req.Method(),
			Resource: req.ResourceType(),
			Headers:  req.Headers(),
		}
		sess.requests = append(sess.requests, info)
		if len(sess.requests) > 200 {
			sess.requests = sess.requests[len(sess.requests)-200:]
		}
	})
	sess.context.OnResponse(func(resp playwright.Response) {
		for i := len(sess.requests) - 1; i >= 0; i-- {
			if sess.requests[i].URL == resp.URL() {
				sess.requests[i].Status = resp.Status()
				sess.requests[i].OK = resp.Ok()
				break
			}
		}
	})
	sess.context.OnRequestFailed(func(req playwright.Request) {
		for i := len(sess.requests) - 1; i >= 0; i-- {
			if sess.requests[i].URL == req.URL() {
				if failure := req.Failure(); failure != nil {
					sess.requests[i].Failure = failure.Error()
				}
				break
			}
		}
	})
}

func (d *daemonService) syncPages(sess *liveSession) {
	sess.pages = sess.context.Pages()
	if sess.active >= len(sess.pages) {
		sess.active = len(sess.pages) - 1
	}
	if sess.active < 0 {
		sess.active = 0
	}
	if len(sess.pages) > 0 {
		page := sess.pages[sess.active]
		sess.lastURL = page.URL()
		title, _ := page.Title()
		sess.lastTitle = title
	}
	sess.updatedAt = time.Now()
}

func (d *daemonService) activePage(sess *liveSession) (playwright.Page, error) {
	d.syncPages(sess)
	if len(sess.pages) == 0 {
		return nil, fmt.Errorf("no open tabs")
	}
	return sess.pages[sess.active], nil
}

func shouldUseFreshNavigationPage(raw string) bool {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.HasPrefix(value, "chrome://"),
		strings.HasPrefix(value, "chrome-untrusted://"),
		strings.HasPrefix(value, "edge://"),
		strings.HasPrefix(value, "devtools://"):
		return true
	default:
		return false
	}
}

func (d *daemonService) ensureNavigationPage(sess *liveSession, page playwright.Page) (playwright.Page, error) {
	if page == nil {
		return nil, fmt.Errorf("no open tabs")
	}
	if !shouldUseFreshNavigationPage(page.URL()) {
		return page, nil
	}
	next, err := sess.context.NewPage()
	if err != nil {
		return nil, err
	}
	if err := next.BringToFront(); err != nil {
		_ = next.Close()
		return nil, err
	}
	d.syncPages(sess)
	sess.active = len(sess.pages) - 1
	return next, nil
}

func (d *daemonService) runSessionCommand(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	switch req.Command {
	case "open":
		return d.cmdOpen(name, sess, req)
	case "attach":
		return d.cmdAttach(name, sess, req)
	case "create":
		return d.cmdCreate(name, sess, req)
	case "goto":
		return d.cmdGoto(name, sess, req)
	case "go-back":
		return d.cmdNav(name, sess, req, "back")
	case "go-forward":
		return d.cmdNav(name, sess, req, "forward")
	case "reload":
		return d.cmdReload(name, sess, req)
	case "close":
		return d.cmdClose(name)
	case "tab-list":
		return d.cmdTabList(name, sess)
	case "tab-new":
		return d.cmdTabNew(name, sess, req)
	case "tab-close":
		return d.cmdTabClose(name, sess, req)
	case "tab-select":
		return d.cmdTabSelect(name, sess, req)
	case "click", "dblclick", "fill", "type", "hover", "check", "uncheck", "select":
		return d.cmdLocatorAction(name, sess, req)
	case "press":
		return d.cmdKeyboard(name, sess, req, "press")
	case "keydown":
		return d.cmdKeyboard(name, sess, req, "down")
	case "keyup":
		return d.cmdKeyboard(name, sess, req, "up")
	case "mousemove", "mousedown", "mouseup", "mousewheel":
		return d.cmdMouse(name, sess, req)
	case "drag":
		return d.cmdDrag(name, sess, req)
	case "upload":
		return d.cmdUpload(name, sess, req)
	case "snapshot":
		return d.cmdSnapshot(name, sess, req)
	case "screenshot":
		return d.cmdScreenshot(name, sess, req)
	case "pdf":
		return d.cmdPDF(name, sess, req)
	case "eval":
		return d.cmdEval(name, sess, req)
	case "resize":
		return d.cmdResize(name, sess, req)
	case "state-save":
		return d.cmdStateSave(name, sess, req)
	case "state-load":
		return d.cmdStateLoad(name, sess, req)
	case "cookie-list":
		return d.cmdCookieList(name, sess, req)
	case "cookie-get":
		return d.cmdCookieGet(name, sess, req)
	case "cookie-set":
		return d.cmdCookieSet(name, sess, req)
	case "cookie-delete":
		return d.cmdCookieDelete(name, sess, req)
	case "cookie-clear":
		return d.cmdCookieClear(name, sess)
	case "localstorage-list", "sessionstorage-list":
		return d.cmdStorageList(name, sess, req)
	case "localstorage-get", "sessionstorage-get":
		return d.cmdStorageGet(name, sess, req)
	case "localstorage-set", "sessionstorage-set":
		return d.cmdStorageSet(name, sess, req)
	case "localstorage-delete", "sessionstorage-delete":
		return d.cmdStorageDelete(name, sess, req)
	case "localstorage-clear", "sessionstorage-clear":
		return d.cmdStorageClear(name, sess, req)
	case "requests":
		return d.cmdRequests(name, sess)
	case "request":
		return d.cmdRequest(name, sess, req)
	case "console":
		return d.cmdConsole(name, sess, req)
	case "dialog-accept":
		return d.cmdDialog(name, sess, req, true)
	case "dialog-dismiss":
		return d.cmdDialog(name, sess, req, false)
	default:
		return nil, fmt.Errorf("unsupported command: %s", req.Command)
	}
}

func (d *daemonService) injectChromeCookies(sess *liveSession, page playwright.Page) error {
	return nil
}

func (d *daemonService) injectChromeCookiesForURL(sess *liveSession, raw string) error {
	return nil
}

func (d *daemonService) injectChromeCookiesForTarget(sess *liveSession, target *url.URL) error {
	return nil
}

func (d *daemonService) loadOrCreateConfig(name string, flags map[string]string) (SessionConfig, error) {
	path := sessionMetaPath(d.opts.StateDir, name)
	config := SessionConfig{Name: name}
	var stored persistedSession
	if err := readJSONFile(path, &stored); err == nil {
		config = stored.SessionConfig
	}
	config = d.applyFlagsToConfig(name, config, flags)
	if err := d.writeSessionState(config, "", "", time.Now()); err != nil {
		return SessionConfig{}, err
	}
	return config, nil
}

func (d *daemonService) launchContext(config SessionConfig) (playwright.BrowserContext, playwright.Browser, error) {
	if err := os.MkdirAll(sessionDir(d.opts.StateDir, config.Name), 0o755); err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(config.ProfileDir, 0o755); err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(config.DownloadDir, 0o755); err != nil {
		return nil, nil, err
	}
	if strings.TrimSpace(config.CDP) != "" {
		switch strings.ToLower(strings.TrimSpace(config.Browser)) {
		case "", "chromium", "chrome":
		default:
			return nil, nil, fmt.Errorf("cdp attach only supports chromium/chrome, got %s", config.Browser)
		}
		endpoint, err := resolveCDPEndpoint(config.CDP)
		if err != nil {
			return nil, nil, err
		}
		if d.logger != nil {
			d.logger.Printf("用参数：会话=%s；阶段=connect_cdp_begin；端点=%s，做了：开始通过 CDP 连接浏览器", config.Name, endpoint)
		}
		browser, err := d.pw.Chromium.ConnectOverCDP(endpoint)
		if err != nil {
			return nil, nil, err
		}
		if d.logger != nil {
			d.logger.Printf("用参数：会话=%s；阶段=connect_cdp_ready；上下文数=%d，做了：已经通过 CDP 连接浏览器", config.Name, len(browser.Contexts()))
		}
		contexts := browser.Contexts()
		var ctx playwright.BrowserContext
		if len(contexts) > 0 {
			ctx = contexts[0]
		} else {
			ctx, err = browser.NewContext(browserNewContextOptions(config))
			if err != nil {
				_ = browser.Close()
				return nil, nil, err
			}
		}
		if d.logger != nil {
			d.logger.Printf("用参数：会话=%s；阶段=apply_fingerprint_begin；页面数=%d；cdp=true，做了：开始应用浏览器指纹覆盖", config.Name, len(ctx.Pages()))
		}
		if err := d.applyFingerprintOverrides(ctx, config, false); err != nil {
			_ = browser.Close()
			return nil, nil, err
		}
		if d.logger != nil {
			d.logger.Printf("用参数：会话=%s；阶段=apply_fingerprint_ready；cdp=true，做了：浏览器指纹覆盖已经完成", config.Name)
		}
		ctx.SetDefaultTimeout(config.TimeoutMS)
		ctx.SetDefaultNavigationTimeout(config.NavTimeout)
		return ctx, browser, nil
	}
	headless := config.Headless
	acceptDownloads := true
	opts := playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless:        &headless,
		AcceptDownloads: &acceptDownloads,
		DownloadsPath:   stringPtr(config.DownloadDir),
		Viewport: &playwright.Size{
			Width:  config.Width,
			Height: config.Height,
		},
		Screen: &playwright.Size{
			Width:  config.Width,
			Height: config.Height,
		},
		Locale:     stringPtr(config.Locale),
		TimezoneId: stringPtr(config.TimezoneID),
		HasTouch:   boolPtr(config.MaxTouchPoints > 0),
	}
	if config.Channel != "" {
		opts.Channel = &config.Channel
	}
	if config.UserAgent != "" {
		opts.UserAgent = &config.UserAgent
	}
	if len(config.Languages) > 0 {
		opts.ExtraHttpHeaders = map[string]string{
			"Accept-Language": strings.Join(config.Languages, ","),
		}
	}

	var bt playwright.BrowserType
	switch strings.ToLower(strings.TrimSpace(config.Browser)) {
	case "", "chromium", "chrome":
		bt = d.pw.Chromium
	case "firefox":
		bt = d.pw.Firefox
	case "webkit":
		bt = d.pw.WebKit
	default:
		return nil, nil, fmt.Errorf("unsupported browser: %s", config.Browser)
	}
	ctx, err := bt.LaunchPersistentContext(config.ProfileDir, opts)
	if err != nil {
		return nil, nil, err
	}
	if err := d.applyFingerprintOverrides(ctx, config, false); err != nil {
		_ = ctx.Close()
		return nil, nil, err
	}
	ctx.SetDefaultTimeout(config.TimeoutMS)
	ctx.SetDefaultNavigationTimeout(config.NavTimeout)
	return ctx, nil, nil
}

func (d *daemonService) persistSessionSummary(name string, sess *liveSession) error {
	d.syncPages(sess)
	return d.writeSessionState(sess.config, sess.lastURL, sess.lastTitle, sess.updatedAt)
}

func (d *daemonService) listSessions() ([]SessionSummary, error) {
	root := sessionRoot(d.opts.StateDir)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []SessionSummary{}, nil
		}
		return nil, err
	}
	out := make([]SessionSummary, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		var item SessionSummary
		if err := readJSONFile(filepath.Join(root, entry.Name(), "session.json"), &item); err == nil {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (d *daemonService) closeSession(name string) error {
	return d.closeSessionWithReason(name, "unspecified")
}

func (d *daemonService) closeSessionWithReason(name, reason string) error {
	name = sanitizeName(name)
	sess := d.sessions[name]
	if sess == nil {
		d.logDiagnostic(map[string]any{
			"event":   "session_close",
			"stage":   "skip_missing",
			"session": name,
			"reason":  strings.TrimSpace(reason),
		})
		return nil
	}
	var err error
	if sess.browser != nil {
		err = sess.browser.Close()
	} else {
		err = sess.context.Close()
	}
	delete(d.sessions, name)
	fields := map[string]any{
		"event":   "session_close",
		"stage":   "finish",
		"session": name,
		"reason":  strings.TrimSpace(reason),
	}
	if sess.browser != nil {
		fields["mode"] = "browser"
	} else {
		fields["mode"] = "context"
	}
	fields["remainingSessionCount"] = len(d.sessions)
	if err != nil {
		fields["error"] = err.Error()
	}
	d.logDiagnostic(fields)
	return err
}

func (d *daemonService) shutdown() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	var shutdownErr error
	for name := range d.sessions {
		if err := d.closeSessionWithReason(name, "daemon_shutdown"); err != nil {
			shutdownErr = errors.Join(shutdownErr, err)
		}
	}
	if d.pw != nil {
		if err := playwrightStopRuntimeFn(d.pw); err != nil {
			shutdownErr = errors.Join(shutdownErr, err)
		}
		d.pw = nil
	}
	return shutdownErr
}

func (d *daemonService) activeSessionNames() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.activeSessionNamesLocked()
}

func (d *daemonService) activeSessionNamesLocked() []string {
	names := make([]string, 0, len(d.sessions))
	for name := range d.sessions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

type persistedSession struct {
	SessionConfig
	Headed    bool      `json:"headed"`
	LastURL   string    `json:"lastUrl,omitempty"`
	LastTitle string    `json:"lastTitle,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (d *daemonService) applyFlagsToConfig(name string, config SessionConfig, flags map[string]string) SessionConfig {
	config.Name = name
	if browser := strings.TrimSpace(flags["browser"]); browser != "" {
		config.Browser = browser
	}
	if cdp, ok := flags["cdp"]; ok {
		config.CDP = strings.TrimSpace(cdp)
		if strings.TrimSpace(flags["browser"]) == "" {
			config.Browser = defaultBrowserName
		}
	}
	if strings.TrimSpace(config.Browser) == "" {
		config.Browser = defaultBrowserName
	}
	if channel := strings.TrimSpace(flags["channel"]); channel != "" {
		config.Channel = channel
	}
	if boolFlag(flags, "headed") {
		config.Headless = false
	} else if config.Width == 0 && config.Height == 0 && config.TimeoutMS == 0 && config.NavTimeout == 0 && config.ProfileDir == "" && config.DownloadDir == "" && !config.Persistent {
		config.Headless = true
	}
	if boolFlag(flags, "persistent") {
		config.Persistent = true
	}
	if width, ok := intFlagValue(flags, "width"); ok {
		config.Width = width
	}
	if height, ok := intFlagValue(flags, "height"); ok {
		config.Height = height
	}
	if config.Width <= 0 {
		config.Width = DefaultViewportWidth
	}
	if config.Height <= 0 {
		config.Height = DefaultViewportHeight
	}
	if ua := strings.TrimSpace(flags["user-agent"]); ua != "" {
		config.UserAgent = ua
	}
	if strings.TrimSpace(config.UserAgent) == "" {
		config.UserAgent = DefaultChromeUserAgent
	}
	if timeout, ok := intFlagValue(flags, "timeout"); ok {
		config.TimeoutMS = float64(timeout)
	}
	if navTimeout, ok := intFlagValue(flags, "navigation-timeout"); ok {
		config.NavTimeout = float64(navTimeout)
	}
	if config.TimeoutMS <= 0 {
		config.TimeoutMS = defaultActionMS
	}
	if config.NavTimeout <= 0 {
		config.NavTimeout = defaultNavMS
	}
	config.Locale, config.Languages = resolvedSystemLocaleValues()
	config.TimezoneID = resolvedSystemTimezoneID()
	config.MaxTouchPoints = DefaultMaxTouchPoints
	config.WebGLVendor = DefaultWebGLVendor
	config.WebGLRenderer = DefaultWebGLRenderer

	root := sessionDir(d.opts.StateDir, name)
	if profile := strings.TrimSpace(flags["profile"]); profile != "" {
		config.ProfileDir = absChild(root, profile)
	}
	if strings.TrimSpace(config.ProfileDir) == "" {
		config.ProfileDir = absChild(root, "profile")
	}
	if download := strings.TrimSpace(flags["downloads"]); download != "" {
		config.DownloadDir = absChild(root, download)
	}
	if strings.TrimSpace(config.DownloadDir) == "" {
		config.DownloadDir = absChild(root, "downloads")
	}
	return config
}

func (d *daemonService) writeSessionState(config SessionConfig, lastURL, lastTitle string, updatedAt time.Time) error {
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	return writeJSONFile(sessionMetaPath(d.opts.StateDir, config.Name), persistedSession{
		SessionConfig: config,
		Headed:        !config.Headless,
		LastURL:       lastURL,
		LastTitle:     lastTitle,
		UpdatedAt:     updatedAt,
	})
}

func resolveCDPEndpoint(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("empty cdp target")
	}
	if strings.EqualFold(value, "chrome") {
		return defaultChromeCDP, nil
	}
	if strings.Contains(value, "://") {
		return value, nil
	}
	return "http://" + value, nil
}

func (d *daemonService) applyFingerprintOverrides(ctx playwright.BrowserContext, config SessionConfig, setHeaders bool) error {
	fp := fingerprintProfileFromConfig(config)
	if strings.TrimSpace(fp.UserAgent) == "" {
		return nil
	}
	if setHeaders {
		headers := map[string]string{"User-Agent": fp.UserAgent}
		if len(fp.Languages) > 0 {
			headers["Accept-Language"] = strings.Join(fp.Languages, ",")
		}
		if err := ctx.SetExtraHTTPHeaders(headers); err != nil {
			return err
		}
	}
	if err := applyFingerprintInitScript(ctx, fp); err != nil {
		return err
	}
	for _, page := range ctx.Pages() {
		if shouldUseFreshNavigationPage(page.URL()) {
			continue
		}
		if err := d.applyFingerprintOverride(ctx, page, fp, setHeaders); err != nil {
			return err
		}
	}
	return nil
}

func (d *daemonService) applyFingerprintOverride(ctx playwright.BrowserContext, page playwright.Page, fp fingerprintProfile, setHeaders bool) error {
	fp = normalizeFingerprintProfile(fp)
	if ctx == nil || page == nil || fp.UserAgent == "" {
		return nil
	}
	if shouldUseFreshNavigationPage(page.URL()) {
		return nil
	}
	if _, err := page.Evaluate(`(fp) => {
		const ua = String(fp && fp.userAgent || "");
		const platform = String(fp && fp.platform || "");
		const locale = String(fp && fp.locale || "");
		const languages = Array.isArray(fp && fp.languages) ? fp.languages.map(item => String(item)) : [];
		const maxTouchPoints = Number(fp && fp.maxTouchPoints || 0);
		const screenWidth = Number(fp && fp.screenWidth || 0);
		const screenHeight = Number(fp && fp.screenHeight || 0);
		const webglVendor = String(fp && fp.webglVendor || "");
		const webglRenderer = String(fp && fp.webglRenderer || "");
		const brands = [
			{ brand: "Chromium", version: "131" },
			{ brand: "Google Chrome", version: "131" },
			{ brand: "Not_A Brand", version: "24" },
		];
		const fullVersionList = [
			{ brand: "Chromium", version: "131.0.0.0" },
			{ brand: "Google Chrome", version: "131.0.0.0" },
			{ brand: "Not_A Brand", version: "24.0.0.0" },
		];
		const proto = Object.getPrototypeOf(navigator);
		const define = target => {
			if (!target) {
				return;
			}
			try {
				Object.defineProperty(target, "userAgent", {
					configurable: true,
					get: () => ua,
				});
			} catch (e) {}
			try {
				Object.defineProperty(target, "platform", {
					configurable: true,
					get: () => platform,
				});
			} catch (e) {}
			try {
				Object.defineProperty(target, "language", {
					configurable: true,
					get: () => locale,
				});
			} catch (e) {}
			try {
				Object.defineProperty(target, "languages", {
					configurable: true,
					get: () => languages.slice(),
				});
			} catch (e) {}
			try {
				Object.defineProperty(target, "maxTouchPoints", {
					configurable: true,
					get: () => maxTouchPoints,
				});
			} catch (e) {}
			try {
				Object.defineProperty(target, "userAgentData", {
					configurable: true,
					get: () => ({
						brands,
						mobile: false,
						platform: "macOS",
						getHighEntropyValues: async hints => {
							const result = {
								architecture: "arm",
								bitness: "64",
								formFactor: "Desktop",
								fullVersionList,
								mobile: false,
								model: "",
								platform: "macOS",
								platformVersion: "14.0.0",
								uaFullVersion: "131.0.0.0",
								wow64: false,
							};
							if (!Array.isArray(hints)) {
								return result;
							}
							return hints.reduce((acc, key) => {
								if (Object.prototype.hasOwnProperty.call(result, key)) {
									acc[key] = result[key];
								}
								return acc;
							}, {
								brands,
								mobile: false,
								platform: "macOS",
							});
						},
						toJSON: () => ({ brands, mobile: false, platform: "macOS" }),
					}),
				});
			} catch (e) {}
			try {
				Object.defineProperty(target, "webdriver", {
					configurable: true,
					get: () => undefined,
				});
			} catch (e) {}
		};
		const defineReadonly = (target, key, value) => {
			if (!target) {
				return;
			}
			try {
				Object.defineProperty(target, key, {
					configurable: true,
					get: () => value,
				});
			} catch (e) {}
		};
		const buildScreenOverride = () => {
			const base = window.screen;
			if (!base) {
				return null;
			}
			const copy = Object.create(base);
			defineReadonly(copy, "width", screenWidth || base.width);
			defineReadonly(copy, "height", screenHeight || base.height);
			defineReadonly(copy, "availWidth", screenWidth || base.availWidth || base.width);
			defineReadonly(copy, "availHeight", screenHeight || base.availHeight || base.height);
			return copy;
		};
		define(proto);
		define(navigator);
		const screenOverride = buildScreenOverride();
		if (screenOverride) {
			defineReadonly(window, "screen", screenOverride);
		}
		const patchWebGL = protoLike => {
			if (!protoLike || !protoLike.getParameter) {
				return;
			}
			const originalGetParameter = protoLike.getParameter;
			const originalGetExtension = protoLike.getExtension;
			if (originalGetParameter && originalGetParameter.__browserFingerprintPatched) {
				return;
			}
			const patched = function(parameter) {
				if (parameter === 37445) {
					return webglVendor;
				}
				if (parameter === 37446) {
					return webglRenderer;
				}
				return originalGetParameter.call(this, parameter);
			};
			Object.defineProperty(patched, "__browserFingerprintPatched", {
				value: true,
				configurable: false,
				enumerable: false,
				writable: false,
			});
			try {
				protoLike.getParameter = patched;
			} catch (e) {}
			if (typeof originalGetExtension === "function") {
				const patchedExtension = function(name) {
					if (String(name || "").toUpperCase() === "WEBGL_DEBUG_RENDERER_INFO") {
						return {
							UNMASKED_VENDOR_WEBGL: 37445,
							UNMASKED_RENDERER_WEBGL: 37446,
						};
					}
					return originalGetExtension.call(this, name);
				};
				try {
					protoLike.getExtension = patchedExtension;
				} catch (e) {}
			}
		};
		patchWebGL(window.WebGLRenderingContext && window.WebGLRenderingContext.prototype);
		patchWebGL(window.WebGL2RenderingContext && window.WebGL2RenderingContext.prototype);
	}`, map[string]any{
		"userAgent":      fp.UserAgent,
		"platform":       fp.Platform,
		"locale":         fp.Locale,
		"languages":      fp.Languages,
		"maxTouchPoints": fp.MaxTouchPoints,
		"screenWidth":    fp.ScreenWidth,
		"screenHeight":   fp.ScreenHeight,
		"webglVendor":    fp.WebGLVendor,
		"webglRenderer":  fp.WebGLRenderer,
	}); err != nil {
		return err
	}
	if !setHeaders {
		return nil
	}
	headers := map[string]string{"User-Agent": fp.UserAgent}
	if len(fp.Languages) > 0 {
		headers["Accept-Language"] = strings.Join(fp.Languages, ",")
	}
	return page.SetExtraHTTPHeaders(headers)
}

func applyFingerprintInitScript(ctx playwright.BrowserContext, fp fingerprintProfile) error {
	fp = normalizeFingerprintProfile(fp)
	if ctx == nil || fp.UserAgent == "" {
		return nil
	}
	script := navigatorUserAgentInitScript(fp)
	return ctx.AddInitScript(playwright.Script{Content: &script})
}

func navigatorUserAgentInitScript(fp fingerprintProfile) string {
	fp = normalizeFingerprintProfile(fp)
	languagesJSON, _ := json.Marshal(fp.Languages)
	return fmt.Sprintf(`(() => {
		const ua = %q;
		const platform = %q;
		const locale = %q;
		const languages = %s;
		const maxTouchPoints = %d;
		const screenWidth = %d;
		const screenHeight = %d;
		const webglVendor = %q;
		const webglRenderer = %q;
		const brands = [
			{ brand: "Chromium", version: "131" },
			{ brand: "Google Chrome", version: "131" },
			{ brand: "Not_A Brand", version: "24" },
		];
		const fullVersionList = [
			{ brand: "Chromium", version: "131.0.0.0" },
			{ brand: "Google Chrome", version: "131.0.0.0" },
			{ brand: "Not_A Brand", version: "24.0.0.0" },
		];
		const proto = Object.getPrototypeOf(navigator);
		const define = target => {
			if (!target) {
				return;
			}
			try {
				Object.defineProperty(target, "userAgent", {
					configurable: true,
					get: () => ua,
				});
			} catch (e) {}
			try {
				Object.defineProperty(target, "platform", {
					configurable: true,
					get: () => platform,
				});
			} catch (e) {}
			try {
				Object.defineProperty(target, "language", {
					configurable: true,
					get: () => locale,
				});
			} catch (e) {}
			try {
				Object.defineProperty(target, "languages", {
					configurable: true,
					get: () => languages.slice(),
				});
			} catch (e) {}
			try {
				Object.defineProperty(target, "maxTouchPoints", {
					configurable: true,
					get: () => maxTouchPoints,
				});
			} catch (e) {}
			try {
				Object.defineProperty(target, "userAgentData", {
					configurable: true,
					get: () => ({
						brands,
						mobile: false,
						platform: "macOS",
						getHighEntropyValues: async hints => {
							const result = {
								architecture: "arm",
								bitness: "64",
								formFactor: "Desktop",
								fullVersionList,
								mobile: false,
								model: "",
								platform: "macOS",
								platformVersion: "14.0.0",
								uaFullVersion: "131.0.0.0",
								wow64: false,
							};
							if (!Array.isArray(hints)) {
								return result;
							}
							return hints.reduce((acc, key) => {
								if (Object.prototype.hasOwnProperty.call(result, key)) {
									acc[key] = result[key];
								}
								return acc;
							}, {
								brands,
								mobile: false,
								platform: "macOS",
							});
						},
						toJSON: () => ({ brands, mobile: false, platform: "macOS" }),
					}),
				});
			} catch (e) {}
			try {
				Object.defineProperty(target, "webdriver", {
					configurable: true,
					get: () => undefined,
				});
			} catch (e) {}
		};
		const defineReadonly = (target, key, value) => {
			if (!target) {
				return;
			}
			try {
				Object.defineProperty(target, key, {
					configurable: true,
					get: () => value,
				});
			} catch (e) {}
		};
		const buildScreenOverride = () => {
			const base = window.screen;
			if (!base) {
				return null;
			}
			const copy = Object.create(base);
			defineReadonly(copy, "width", screenWidth || base.width);
			defineReadonly(copy, "height", screenHeight || base.height);
			defineReadonly(copy, "availWidth", screenWidth || base.availWidth || base.width);
			defineReadonly(copy, "availHeight", screenHeight || base.availHeight || base.height);
			return copy;
		};
		define(proto);
		define(navigator);
		const screenOverride = buildScreenOverride();
		if (screenOverride) {
			defineReadonly(window, "screen", screenOverride);
		}
		const patchWebGL = protoLike => {
			if (!protoLike || !protoLike.getParameter) {
				return;
			}
			const originalGetParameter = protoLike.getParameter;
			const originalGetExtension = protoLike.getExtension;
			if (originalGetParameter && originalGetParameter.__browserFingerprintPatched) {
				return;
			}
			const patched = function(parameter) {
				if (parameter === 37445) {
					return webglVendor;
				}
				if (parameter === 37446) {
					return webglRenderer;
				}
				return originalGetParameter.call(this, parameter);
			};
			Object.defineProperty(patched, "__browserFingerprintPatched", {
				value: true,
				configurable: false,
				enumerable: false,
				writable: false,
			});
			try {
				protoLike.getParameter = patched;
			} catch (e) {}
			if (typeof originalGetExtension === "function") {
				const patchedExtension = function(name) {
					if (String(name || "").toUpperCase() === "WEBGL_DEBUG_RENDERER_INFO") {
						return {
							UNMASKED_VENDOR_WEBGL: 37445,
							UNMASKED_RENDERER_WEBGL: 37446,
						};
					}
					return originalGetExtension.call(this, name);
				};
				try {
					protoLike.getExtension = patchedExtension;
				} catch (e) {}
			}
		};
		patchWebGL(window.WebGLRenderingContext && window.WebGLRenderingContext.prototype);
		patchWebGL(window.WebGL2RenderingContext && window.WebGL2RenderingContext.prototype);
	})();`, fp.UserAgent, fp.Platform, fp.Locale, string(languagesJSON), fp.MaxTouchPoints, fp.ScreenWidth, fp.ScreenHeight, fp.WebGLVendor, fp.WebGLRenderer)
}

func fingerprintProfileFromConfig(config SessionConfig) fingerprintProfile {
	return normalizeFingerprintProfile(fingerprintProfile{
		UserAgent:      config.UserAgent,
		Platform:       DefaultNavigatorPlatform,
		Locale:         config.Locale,
		Languages:      append([]string(nil), config.Languages...),
		TimezoneID:     config.TimezoneID,
		ViewportWidth:  config.Width,
		ViewportHeight: config.Height,
		ScreenWidth:    config.Width,
		ScreenHeight:   config.Height,
		MaxTouchPoints: config.MaxTouchPoints,
		WebGLVendor:    config.WebGLVendor,
		WebGLRenderer:  config.WebGLRenderer,
	})
}

func normalizeFingerprintProfile(fp fingerprintProfile) fingerprintProfile {
	fp.UserAgent = strings.TrimSpace(fp.UserAgent)
	fp.Platform = strings.TrimSpace(fp.Platform)
	if fp.Platform == "" {
		fp.Platform = DefaultNavigatorPlatform
	}
	fp.Locale = strings.TrimSpace(fp.Locale)
	if fp.Locale == "" {
		fp.Locale = resolvedSystemPrimaryLocale()
	}
	if len(fp.Languages) == 0 {
		_, fp.Languages = resolvedSystemLocaleValues()
	}
	fp.TimezoneID = strings.TrimSpace(fp.TimezoneID)
	if fp.TimezoneID == "" {
		fp.TimezoneID = resolvedSystemTimezoneID()
	}
	if fp.ViewportWidth <= 0 {
		fp.ViewportWidth = DefaultViewportWidth
	}
	if fp.ViewportHeight <= 0 {
		fp.ViewportHeight = DefaultViewportHeight
	}
	if fp.ScreenWidth <= 0 {
		fp.ScreenWidth = DefaultScreenWidth
	}
	if fp.ScreenHeight <= 0 {
		fp.ScreenHeight = DefaultScreenHeight
	}
	if fp.MaxTouchPoints <= 0 {
		fp.MaxTouchPoints = DefaultMaxTouchPoints
	}
	fp.WebGLVendor = strings.TrimSpace(fp.WebGLVendor)
	if fp.WebGLVendor == "" {
		fp.WebGLVendor = DefaultWebGLVendor
	}
	fp.WebGLRenderer = strings.TrimSpace(fp.WebGLRenderer)
	if fp.WebGLRenderer == "" {
		fp.WebGLRenderer = DefaultWebGLRenderer
	}
	return fp
}

func resolvedSystemPrimaryLocale() string {
	locale, _ := resolvedSystemLocaleValues()
	return locale
}

func resolvedSystemLocaleValues() (string, []string) {
	primary, languages := systemLocaleValuesFn()
	primary = normalizeLocaleCode(primary)
	out := normalizeLanguageList(languages)
	if primary == "" && len(out) > 0 {
		primary = out[0]
	}
	if primary == "" {
		primary = DefaultLocale
	}
	if len(out) == 0 {
		out = []string{primary}
	}
	if out[0] != primary {
		items := []string{primary}
		for _, item := range out {
			if item != primary {
				items = append(items, item)
			}
		}
		out = items
	}
	return primary, out
}

func resolvedSystemTimezoneID() string {
	value := normalizeTimezoneID(systemTimezoneIDFn())
	if value == "" {
		return DefaultTimezoneID
	}
	return value
}

func detectSystemLocaleValues() (string, []string) {
	if runtime.GOOS == "darwin" {
		if langs := normalizeLanguageList(parseAppleLanguages(defaultsReadGlobal("AppleLanguages"))); len(langs) > 0 {
			primary := langs[0]
			if locale := normalizeLocaleCode(defaultsReadGlobal("AppleLocale")); locale != "" {
				primary = locale
				if langs[0] != primary {
					langs = append([]string{primary}, filterLanguages(langs, primary)...)
				}
			}
			return primary, langs
		}
		if locale := normalizeLocaleCode(defaultsReadGlobal("AppleLocale")); locale != "" {
			return locale, []string{locale}
		}
	}
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if locale := normalizeLocaleCode(os.Getenv(key)); locale != "" {
			return locale, []string{locale}
		}
	}
	return DefaultLocale, []string{DefaultLocale}
}

func detectSystemTimezoneID() string {
	if value := normalizeTimezoneID(os.Getenv("TZ")); value != "" {
		return value
	}
	for _, path := range []string{"/etc/localtime", "/var/db/timezone/zoneinfo"} {
		if value := timezoneIDFromSymlink(path); value != "" {
			return value
		}
	}
	if runtime.GOOS == "darwin" {
		if location, err := time.LoadLocation("Local"); err == nil {
			if value := normalizeTimezoneID(location.String()); value != "" && value != "Local" {
				return value
			}
		}
	}
	return DefaultTimezoneID
}

func defaultsReadGlobal(key string) string {
	out, err := exec.Command("defaults", "read", "-g", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func parseAppleLanguages(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(line, ","), ";"))
		line = strings.Trim(line, "()\" ")
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func filterLanguages(items []string, skip string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item != skip {
			out = append(out, item)
		}
	}
	return out
}

func normalizeLanguageList(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		normalized := normalizeLocaleCode(item)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func normalizeLocaleCode(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if idx := strings.Index(value, "."); idx >= 0 {
		value = value[:idx]
	}
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.Trim(value, "\" ")
	if strings.EqualFold(value, "c") || strings.EqualFold(value, "posix") {
		return ""
	}
	parts := strings.Split(value, "-")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		switch {
		case i == 0:
			parts[i] = strings.ToLower(part)
		case len(part) == 2:
			parts[i] = strings.ToUpper(part)
		case len(part) == 4:
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		default:
			parts[i] = part
		}
	}
	return strings.Join(parts, "-")
}

func normalizeTimezoneID(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, ":")
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "/") {
		if idx := strings.Index(value, "/zoneinfo/"); idx >= 0 {
			value = value[idx+len("/zoneinfo/"):]
		}
	}
	return value
}

func timezoneIDFromSymlink(path string) string {
	target, err := os.Readlink(path)
	if err != nil {
		return ""
	}
	return normalizeTimezoneID(target)
}

func browserNewContextOptions(config SessionConfig) playwright.BrowserNewContextOptions {
	return playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{
			Width:  config.Width,
			Height: config.Height,
		},
		Screen: &playwright.Size{
			Width:  config.Width,
			Height: config.Height,
		},
		UserAgent:  stringPtr(config.UserAgent),
		Locale:     stringPtr(config.Locale),
		TimezoneId: stringPtr(config.TimezoneID),
		HasTouch:   boolPtr(config.MaxTouchPoints > 0),
		ExtraHttpHeaders: map[string]string{
			"Accept-Language": strings.Join(config.Languages, ","),
		},
	}
}

func writeHTTPJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}

func boolFlag(flags map[string]string, key string) bool {
	raw, ok := flags[key]
	if !ok {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func intFlag(flags map[string]string, key string, fallback int) int {
	raw, ok := flags[key]
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return value
}

func intFlagValue(flags map[string]string, key string) (int, bool) {
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

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func pageInfo(page playwright.Page, index int) *PageInfo {
	title, _ := page.Title()
	return &PageInfo{
		URL:   page.URL(),
		Title: title,
		Index: index,
	}
}

func resultWithPage(name, command string, sess *liveSession) (*CommandResult, error) {
	if len(sess.pages) == 0 {
		return &CommandResult{Session: name, Command: command, Message: "no active page"}, nil
	}
	page := sess.pages[sess.active]
	return &CommandResult{
		Session: commandSession(name),
		Command: command,
		Page:    pageInfo(page, sess.active),
		Tabs:    tabsFromSession(sess),
	}, nil
}

func commandSession(name string) string {
	if name == "" {
		return "default"
	}
	return name
}

func tabsFromSession(sess *liveSession) []TabInfo {
	out := make([]TabInfo, 0, len(sess.pages))
	for i, page := range sess.pages {
		title, _ := page.Title()
		out = append(out, TabInfo{
			Index:  i,
			URL:    page.URL(),
			Title:  title,
			Active: i == sess.active,
		})
	}
	return out
}

func writeTextFile(path, value string) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(value), 0o644)
}

func logWriter(path string) (io.WriteCloser, error) {
	if err := ensureParentDir(path); err != nil {
		return nil, err
	}
	return &lumberjack.Logger{
		Filename:   path,
		MaxSize:    defaultLogMaxSizeMB,
		MaxBackups: defaultLogMaxFiles - 1,
		LocalTime:  true,
		Compress:   false,
	}, nil
}

func (d *daemonService) logNavigationDiagnostic(sess *liveSession, command string, target string, elapsed time.Duration) {
	d.logDiagnostic(map[string]any{
		"event":         "navigation",
		"session":       sess.config.Name,
		"command":       command,
		"target":        strings.TrimSpace(target),
		"gotoCostMs":    elapsed.Milliseconds(),
		"gotoCostExact": elapsed.String(),
	})
}

func (d *daemonService) logCommandDiagnostic(stage, session string, req CommandRequest, result *CommandResult, err error, elapsed time.Duration) {
	fields := map[string]any{
		"event":   "command",
		"stage":   strings.TrimSpace(stage),
		"session": strings.TrimSpace(session),
		"command": strings.TrimSpace(req.Command),
		"args":    append([]string{}, req.Args...),
		"flags":   cloneLogStringMap(req.Flags),
	}
	if elapsed > 0 {
		fields["elapsedMs"] = elapsed.Milliseconds()
		fields["elapsed"] = elapsed.String()
	}
	if result != nil {
		fields["result"] = commandResultSummary(result)
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	d.logDiagnostic(fields)
}

func cloneLogStringMap(value map[string]string) map[string]string {
	if len(value) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(value))
	for key, item := range value {
		out[key] = item
	}
	return out
}

func commandResultSummary(result *CommandResult) map[string]any {
	if result == nil {
		return map[string]any{}
	}
	out := map[string]any{}
	if strings.TrimSpace(result.Session) != "" {
		out["session"] = result.Session
	}
	if strings.TrimSpace(result.Command) != "" {
		out["command"] = result.Command
	}
	if strings.TrimSpace(result.Message) != "" {
		out["message"] = result.Message
	}
	if result.Page != nil {
		out["page"] = result.Page
	}
	if len(result.Tabs) > 0 {
		out["tabCount"] = len(result.Tabs)
	}
	if len(result.Cookies) > 0 {
		out["cookieCount"] = len(result.Cookies)
	}
	if len(result.Storage) > 0 {
		out["storageCount"] = len(result.Storage)
	}
	if len(result.OutputPaths) > 0 {
		out["outputPaths"] = append([]string{}, result.OutputPaths...)
	}
	if strings.TrimSpace(result.OutputPath) != "" {
		out["outputPath"] = result.OutputPath
	}
	if strings.TrimSpace(result.StatePath) != "" {
		out["statePath"] = result.StatePath
	}
	if result.Snapshot != nil {
		out["snapshotPath"] = result.Snapshot.Path
	}
	if len(result.Requests) > 0 {
		out["requestCount"] = len(result.Requests)
	}
	if len(result.Console) > 0 {
		out["consoleCount"] = len(result.Console)
	}
	if result.Value != nil {
		out["hasValue"] = true
	}
	return out
}

func (d *daemonService) logDiagnostic(fields map[string]any) {
	if d == nil || d.logger == nil || len(fields) == 0 {
		return
	}
	data, err := json.Marshal(fields)
	if err != nil {
		d.logger.Printf("diagnostic marshal error: %v", err)
		return
	}
	d.logger.Printf("%s", data)
}
