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
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
)

func writePlaywrightDriverFixture(t *testing.T, driverDir, nodeFile string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(driverDir, "package"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{
		filepath.Join(driverDir, nodeFile),
		filepath.Join(driverDir, "package", "cli.js"),
		filepath.Join(driverDir, "package", "package.json"),
	} {
		if err := os.WriteFile(file, []byte("fake"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResolveCDPEndpoint(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "chrome alias", raw: "chrome", want: defaultChromeCDP},
		{name: "http passthrough", raw: "http://127.0.0.1:9222", want: "http://127.0.0.1:9222"},
		{name: "ws passthrough", raw: "ws://127.0.0.1:9222/devtools/browser/abc", want: "ws://127.0.0.1:9222/devtools/browser/abc"},
		{name: "host port", raw: "127.0.0.1:9222", want: "http://127.0.0.1:9222"},
	}

	for _, tc := range tests {
		got, err := resolveCDPEndpoint(tc.raw)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
		if got != tc.want {
			t.Fatalf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}

func TestNavigatorUserAgentInitScriptEmbedsQuotedUserAgent(t *testing.T) {
	got := navigatorUserAgentInitScript(fingerprintProfile{
		UserAgent:      `Test UA "quoted" value`,
		Platform:       DefaultNavigatorPlatform,
		Locale:         DefaultLocale,
		Languages:      []string{DefaultLocale, "en"},
		MaxTouchPoints: DefaultMaxTouchPoints,
		ScreenWidth:    DefaultScreenWidth,
		ScreenHeight:   DefaultScreenHeight,
		WebGLVendor:    DefaultWebGLVendor,
		WebGLRenderer:  DefaultWebGLRenderer,
	})
	if !strings.Contains(got, `"userAgent"`) {
		t.Fatalf("script missing userAgent override: %s", got)
	}
	if !strings.Contains(got, `Test UA \"quoted\" value`) {
		t.Fatalf("script missing quoted user agent: %s", got)
	}
	if !strings.Contains(got, `"platform"`) {
		t.Fatalf("script missing platform override: %s", got)
	}
	if !strings.Contains(got, DefaultNavigatorPlatform) {
		t.Fatalf("script missing default navigator platform: %s", got)
	}
	if !strings.Contains(got, `"webdriver"`) {
		t.Fatalf("script missing webdriver override: %s", got)
	}
	if !strings.Contains(got, "undefined") {
		t.Fatalf("script missing webdriver undefined getter: %s", got)
	}
	for _, want := range []string{`"userAgentData"`, `"maxTouchPoints"`, `"screen"`, `Apple M1 Pro`, `Apple Inc.`, `en-US`} {
		if !strings.Contains(got, want) {
			t.Fatalf("script missing %q: %s", want, got)
		}
	}
	for _, want := range []string{`WEBGL_DEBUG_RENDERER_INFO`, `UNMASKED_RENDERER_WEBGL`, `buildScreenOverride`} {
		if !strings.Contains(got, want) {
			t.Fatalf("script missing %q: %s", want, got)
		}
	}
}

func TestLoadOrCreateConfigPersistsCDP(t *testing.T) {
	svc := newDaemonService(Options{StateDir: t.TempDir()}, log.New(io.Discard, "", 0))

	first, err := svc.loadOrCreateConfig("demo", map[string]string{"cdp": "chrome"})
	if err != nil {
		t.Fatalf("first load config: %v", err)
	}
	if first.CDP != "chrome" {
		t.Fatalf("first config cdp = %q", first.CDP)
	}
	if first.Browser != defaultBrowserName {
		t.Fatalf("first config browser = %q", first.Browser)
	}
	if first.UserAgent != DefaultChromeUserAgent {
		t.Fatalf("first config userAgent = %q", first.UserAgent)
	}

	second, err := svc.loadOrCreateConfig("demo", map[string]string{})
	if err != nil {
		t.Fatalf("second load config: %v", err)
	}
	if second.CDP != "chrome" {
		t.Fatalf("second config cdp = %q", second.CDP)
	}
	if second.Browser != defaultBrowserName {
		t.Fatalf("second config browser = %q", second.Browser)
	}
	if second.UserAgent != DefaultChromeUserAgent {
		t.Fatalf("second config userAgent = %q", second.UserAgent)
	}
}

func TestLoadOrCreateConfigPreservesExplicitUserAgent(t *testing.T) {
	svc := newDaemonService(Options{StateDir: t.TempDir()}, log.New(io.Discard, "", 0))

	config, err := svc.loadOrCreateConfig("demo", map[string]string{"user-agent": "custom-ua"})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if config.UserAgent != "custom-ua" {
		t.Fatalf("config userAgent = %q", config.UserAgent)
	}
}

func TestApplyFlagsToConfigDefaultsChromeUserAgentForNonCDP(t *testing.T) {
	svc := newDaemonService(Options{StateDir: t.TempDir()}, log.New(io.Discard, "", 0))
	originalLocaleFn := systemLocaleValuesFn
	originalTimezoneFn := systemTimezoneIDFn
	defer func() {
		systemLocaleValuesFn = originalLocaleFn
		systemTimezoneIDFn = originalTimezoneFn
	}()
	systemLocaleValuesFn = func() (string, []string) {
		return "zh-Hans-CN", []string{"zh-Hans-CN", "zh-CN"}
	}
	systemTimezoneIDFn = func() string {
		return "Asia/Shanghai"
	}

	config := svc.applyFlagsToConfig("demo", SessionConfig{}, map[string]string{})
	if config.UserAgent != DefaultChromeUserAgent {
		t.Fatalf("config userAgent = %q", config.UserAgent)
	}
	if config.Width != DefaultViewportWidth || config.Height != DefaultViewportHeight {
		t.Fatalf("viewport = %dx%d", config.Width, config.Height)
	}
	if config.Locale != "zh-Hans-CN" || config.TimezoneID != "Asia/Shanghai" {
		t.Fatalf("locale/timezone = %q %q", config.Locale, config.TimezoneID)
	}
	if len(config.Languages) != 2 || config.Languages[0] != "zh-Hans-CN" || config.Languages[1] != "zh-CN" {
		t.Fatalf("languages = %+v", config.Languages)
	}
	if config.MaxTouchPoints != DefaultMaxTouchPoints {
		t.Fatalf("maxTouchPoints = %d", config.MaxTouchPoints)
	}
}

func TestBrowserNewContextOptionsCarryFingerprintDefaults(t *testing.T) {
	opts := browserNewContextOptions(SessionConfig{
		UserAgent:      DefaultChromeUserAgent,
		Width:          DefaultViewportWidth,
		Height:         DefaultViewportHeight,
		Locale:         "zh-Hans-CN",
		Languages:      []string{"zh-Hans-CN", "zh-CN"},
		TimezoneID:     "Asia/Shanghai",
		MaxTouchPoints: DefaultMaxTouchPoints,
	})
	if opts.UserAgent == nil || *opts.UserAgent != DefaultChromeUserAgent {
		t.Fatalf("userAgent = %+v", opts.UserAgent)
	}
	if opts.Locale == nil || *opts.Locale != "zh-Hans-CN" {
		t.Fatalf("locale = %+v", opts.Locale)
	}
	if opts.TimezoneId == nil || *opts.TimezoneId != "Asia/Shanghai" {
		t.Fatalf("timezone = %+v", opts.TimezoneId)
	}
	if opts.HasTouch == nil || !*opts.HasTouch {
		t.Fatalf("hasTouch = %+v", opts.HasTouch)
	}
	if opts.Screen == nil || opts.Screen.Width != DefaultViewportWidth || opts.Screen.Height != DefaultViewportHeight {
		t.Fatalf("screen = %+v", opts.Screen)
	}
	if opts.ExtraHttpHeaders["Accept-Language"] != "zh-Hans-CN,zh-CN" {
		t.Fatalf("accept-language = %q", opts.ExtraHttpHeaders["Accept-Language"])
	}
}

func TestResolvedSystemLocaleValuesNormalizesAndDeduplicates(t *testing.T) {
	originalLocaleFn := systemLocaleValuesFn
	defer func() {
		systemLocaleValuesFn = originalLocaleFn
	}()
	systemLocaleValuesFn = func() (string, []string) {
		return "zh_CN", []string{"zh-Hans-CN", "zh_CN", "zh-Hans-CN"}
	}
	primary, languages := resolvedSystemLocaleValues()
	if primary != "zh-CN" {
		t.Fatalf("primary = %q", primary)
	}
	if len(languages) != 2 || languages[0] != "zh-CN" || languages[1] != "zh-Hans-CN" {
		t.Fatalf("languages = %+v", languages)
	}
}

func TestShouldUseFreshNavigationPage(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{name: "chrome new tab", url: "chrome://newtab/", want: true},
		{name: "chrome untrusted", url: "chrome-untrusted://new-tab-page/one-google-bar", want: true},
		{name: "edge new tab", url: "edge://newtab/", want: true},
		{name: "devtools", url: "devtools://devtools/bundled", want: true},
		{name: "about blank", url: "about:blank", want: false},
		{name: "http page", url: "https://example.com", want: false},
	}

	for _, tc := range tests {
		if got := shouldUseFreshNavigationPage(tc.url); got != tc.want {
			t.Fatalf("%s: shouldUseFreshNavigationPage(%q) = %v, want %v", tc.name, tc.url, got, tc.want)
		}
	}
}

func TestSessionRequiresCDPReconnect(t *testing.T) {
	tests := []struct {
		name       string
		currentCDP string
		flags      map[string]string
		want       bool
	}{
		{
			name:       "no cdp supplied",
			currentCDP: "ws://127.0.0.1:9222/devtools/browser/first",
			flags:      map[string]string{},
			want:       false,
		},
		{
			name:       "same cdp",
			currentCDP: "ws://127.0.0.1:9222/devtools/browser/first",
			flags: map[string]string{
				"cdp": "ws://127.0.0.1:9222/devtools/browser/first",
			},
			want: false,
		},
		{
			name:       "new browser cdp",
			currentCDP: "ws://127.0.0.1:9222/devtools/browser/first",
			flags: map[string]string{
				"cdp": "ws://127.0.0.1:9222/devtools/browser/restarted",
			},
			want: true,
		},
		{
			name:       "attach cdp to a local session",
			currentCDP: "",
			flags: map[string]string{
				"cdp": "ws://127.0.0.1:9222/devtools/browser/restarted",
			},
			want: true,
		},
	}

	for _, tc := range tests {
		if got := sessionRequiresCDPReconnect(tc.currentCDP, tc.flags); got != tc.want {
			t.Fatalf("%s: sessionRequiresCDPReconnect(%q, %+v) = %v, want %v", tc.name, tc.currentCDP, tc.flags, got, tc.want)
		}
	}
}

func TestAttachWithoutCDPRequiresFlagOrSavedConfig(t *testing.T) {
	svc := newDaemonService(Options{StateDir: t.TempDir()}, log.New(io.Discard, "", 0))

	_, err := svc.loadOrCreateConfig("demo", map[string]string{})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	_, err = svc.ensureSession("demo", "attach", map[string]string{})
	if err == nil || err.Error() != "attach requires --cdp=chrome or --cdp=<url>" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareCreateFlagsRequiresAgentAndChat(t *testing.T) {
	if _, err := prepareCreateFlags(map[string]string{"agentId": "agent-a"}); err == nil || err.Error() != "create requires --agentId and --chatId" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareCreateFlagsUsesSessionIdentityWhenPresent(t *testing.T) {
	originalExecutablePathFn := browserPlaywrightExecutablePathFn
	originalGetFn := browserInstanceGetFn
	defer func() {
		browserPlaywrightExecutablePathFn = originalExecutablePathFn
		browserInstanceGetFn = originalGetFn
	}()

	root := t.TempDir()
	playwrightDir := filepath.Join(root, "browser", "playwright")
	instanceDir := filepath.Join(root, "browser", "instance")
	if err := os.MkdirAll(playwrightDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(instanceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	playwrightBinary := filepath.Join(playwrightDir, "browser_playwright")
	instanceBinary := filepath.Join(instanceDir, "browser_instance")
	if err := os.WriteFile(playwrightBinary, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(instanceBinary, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	browserPlaywrightExecutablePathFn = func() (string, error) {
		return playwrightBinary, nil
	}

	var gotAgent string
	var gotChat string
	browserInstanceGetFn = func(path, agentID, chatID string, flags map[string]string, timeout time.Duration) (instanceCreateResult, error) {
		gotAgent = agentID
		gotChat = chatID
		return instanceCreateResult{AgentID: agentID, ChatID: chatID, Port: 20001, CDP: instanceCDPEndpoint(20001)}, nil
	}

	flags, err := prepareCreateFlags(map[string]string{
		"session": "agent-a@ctrip-home",
		"agentId": "agent-b",
		"chatId":  "other-chat",
	})
	if err != nil {
		t.Fatalf("prepare create flags: %v", err)
	}
	if gotAgent != "agent-a" || gotChat != "ctrip-home" {
		t.Fatalf("identity = %s %s, want session-derived agent-a ctrip-home", gotAgent, gotChat)
	}
	if flags["session"] != "agent-a@ctrip-home" {
		t.Fatalf("session = %q", flags["session"])
	}
	if flags["agentId"] != "agent-a" || flags["chatId"] != "ctrip-home" {
		t.Fatalf("flags agent/chat not normalized: %+v", flags)
	}
}

func TestPrepareCreateFlagsResolvesInstanceAndInjectsCDP(t *testing.T) {
	originalExecutablePathFn := browserPlaywrightExecutablePathFn
	originalGetFn := browserInstanceGetFn
	originalCreateFn := browserInstanceCreateFn
	defer func() {
		browserPlaywrightExecutablePathFn = originalExecutablePathFn
		browserInstanceGetFn = originalGetFn
		browserInstanceCreateFn = originalCreateFn
	}()

	root := t.TempDir()
	playwrightDir := filepath.Join(root, "browser", "playwright")
	instanceDir := filepath.Join(root, "browser", "instance")
	if err := os.MkdirAll(playwrightDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(instanceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	playwrightBinary := filepath.Join(playwrightDir, "browser_playwright")
	instanceBinary := filepath.Join(instanceDir, "browser_instance")
	if err := os.WriteFile(playwrightBinary, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(instanceBinary, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}

	browserPlaywrightExecutablePathFn = func() (string, error) {
		return playwrightBinary, nil
	}

	var gotPath string
	var gotAgent string
	var gotChat string
	var gotTimeout time.Duration
	getCalls := 0
	browserInstanceGetFn = func(path, agentID, chatID string, flags map[string]string, timeout time.Duration) (instanceCreateResult, error) {
		getCalls++
		return instanceCreateResult{}, errBrowserInstanceNotFound
	}
	browserInstanceCreateFn = func(path, agentID, chatID string, flags map[string]string, timeout time.Duration) (instanceCreateResult, error) {
		gotPath = path
		gotAgent = agentID
		gotChat = chatID
		gotTimeout = timeout
		return instanceCreateResult{AgentID: agentID, ChatID: chatID, Port: 10086, PID: 9527}, nil
	}

	flags, err := prepareCreateFlags(map[string]string{"agentId": "agent-a", "chatId": "chat-001", "browser-timeout": "12s"})
	if err != nil {
		t.Fatalf("prepare create flags: %v", err)
	}
	if gotPath != instanceBinary {
		t.Fatalf("instance path = %q, want %q", gotPath, instanceBinary)
	}
	if gotAgent != "agent-a" || gotChat != "chat-001" {
		t.Fatalf("unexpected instance identity: %s %s", gotAgent, gotChat)
	}
	if gotTimeout != 12*time.Second {
		t.Fatalf("timeout = %s", gotTimeout)
	}
	if getCalls != 1 {
		t.Fatalf("getCalls = %d, want 1", getCalls)
	}
	if flags["cdp"] != "ws://127.0.0.1:10086/devtools/browser" {
		t.Fatalf("cdp = %q", flags["cdp"])
	}
	if flags["session"] != "agent-a@chat-001" {
		t.Fatalf("session = %q", flags["session"])
	}
}

func TestNormalizeOptionsKeepsExplicitExecutablePath(t *testing.T) {
	root := t.TempDir()
	got, err := normalizeOptions(Options{
		StateDir:       filepath.Join(root, "state"),
		ExecutablePath: filepath.Join(root, "plugins", "browser"),
	})
	if err != nil {
		t.Fatalf("normalizeOptions: %v", err)
	}
	if got.ExecutablePath != filepath.Join(root, "plugins", "browser") {
		t.Fatalf("executable path = %q", got.ExecutablePath)
	}
}

func TestWrapEvalExpressionWithTimeoutUsesDefaultActionTimeout(t *testing.T) {
	script, timeout := wrapEvalExpressionWithTimeout("1+1", 0)
	if timeout != defaultActionMS {
		t.Fatalf("timeout = %v", timeout)
	}
	if !strings.Contains(script, "Promise.race") {
		t.Fatalf("script missing Promise.race: %s", script)
	}
	if !strings.Contains(script, "globalThis.eval") {
		t.Fatalf("script missing globalThis.eval: %s", script)
	}
}

func TestWrapEvalExpressionWithTimeoutPreservesExplicitTimeout(t *testing.T) {
	_, timeout := wrapEvalExpressionWithTimeout("1+1", 1234)
	if timeout != 1234 {
		t.Fatalf("timeout = %v", timeout)
	}
}

func TestPrepareManagedFlagsUsesExistingInstanceWithoutCreate(t *testing.T) {
	originalExecutablePathFn := browserPlaywrightExecutablePathFn
	originalGetFn := browserInstanceGetFn
	originalCreateFn := browserInstanceCreateFn
	defer func() {
		browserPlaywrightExecutablePathFn = originalExecutablePathFn
		browserInstanceGetFn = originalGetFn
		browserInstanceCreateFn = originalCreateFn
	}()

	root := t.TempDir()
	playwrightDir := filepath.Join(root, "browser", "playwright")
	instanceDir := filepath.Join(root, "browser", "instance")
	if err := os.MkdirAll(playwrightDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(instanceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	playwrightBinary := filepath.Join(playwrightDir, "browser_playwright")
	instanceBinary := filepath.Join(instanceDir, "browser_instance")
	if err := os.WriteFile(playwrightBinary, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(instanceBinary, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	browserPlaywrightExecutablePathFn = func() (string, error) {
		return playwrightBinary, nil
	}

	getCalls := 0
	createCalls := 0
	browserInstanceGetFn = func(path, agentID, chatID string, flags map[string]string, timeout time.Duration) (instanceCreateResult, error) {
		getCalls++
		if path != instanceBinary {
			t.Fatalf("path = %q, want %q", path, instanceBinary)
		}
		return instanceCreateResult{AgentID: agentID, ChatID: chatID, Port: 21001, CDP: instanceCDPEndpoint(21001)}, nil
	}
	browserInstanceCreateFn = func(path, agentID, chatID string, flags map[string]string, timeout time.Duration) (instanceCreateResult, error) {
		createCalls++
		return instanceCreateResult{}, nil
	}

	flags, managed, err := prepareManagedFlags(map[string]string{
		"agentId": "agent-a",
		"chatId":  "chat-001",
	})
	if err != nil {
		t.Fatalf("prepareManagedFlags: %v", err)
	}
	if !managed {
		t.Fatalf("expected managed=true")
	}
	if getCalls != 1 || createCalls != 0 {
		t.Fatalf("getCalls=%d createCalls=%d", getCalls, createCalls)
	}
	if flags["session"] != "agent-a@chat-001" {
		t.Fatalf("session = %q", flags["session"])
	}
	if flags["cdp"] != instanceCDPEndpoint(21001) {
		t.Fatalf("cdp = %q", flags["cdp"])
	}
}

func TestStopDaemonTreatsStalePIDFileAsStopped(t *testing.T) {
	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "browser.pid")
	logFile := filepath.Join(tempDir, "browser.log")
	if err := os.WriteFile(pidFile, []byte("999999"), 0o644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}

	result, err := StopDaemon(Options{
		StateDir: tempDir,
		PIDFile:  pidFile,
		LogFile:  logFile,
	})
	if err != nil {
		t.Fatalf("StopDaemon: %v", err)
	}
	if result.Status != "stopped" {
		t.Fatalf("status = %q, want stopped", result.Status)
	}
	if result.Reason != "stale_pid_file" {
		t.Fatalf("reason = %q, want stale_pid_file", result.Reason)
	}
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Fatalf("pid file should be removed, stat err=%v", err)
	}
}

func TestConsumeDaemonShutdownReasonRemovesMarkerFile(t *testing.T) {
	stateDir := t.TempDir()
	if err := writeDaemonShutdownReason(stateDir, "explicit_stop_command", 1001, map[string]string{
		"addr": "127.0.0.1:18333",
		"pid":  "1001",
	}); err != nil {
		t.Fatalf("write shutdown reason: %v", err)
	}

	item, err := consumeDaemonShutdownReason(stateDir, 1001)
	if err != nil {
		t.Fatalf("consume shutdown reason: %v", err)
	}
	if item.Reason != "explicit_stop_command" {
		t.Fatalf("reason = %q, want explicit_stop_command", item.Reason)
	}
	if item.TargetPID != 1001 {
		t.Fatalf("targetPid = %d, want 1001", item.TargetPID)
	}
	if item.RequestedByPID <= 0 {
		t.Fatalf("requestedByPid = %d, want positive", item.RequestedByPID)
	}
	if item.Details["addr"] != "127.0.0.1:18333" {
		t.Fatalf("addr detail = %q", item.Details["addr"])
	}
	if _, err := os.Stat(shutdownReasonPath(stateDir)); !os.IsNotExist(err) {
		t.Fatalf("shutdown reason marker should be removed, stat err=%v", err)
	}
}

func TestConsumeDaemonShutdownReasonSkipsMismatchedTargetPID(t *testing.T) {
	stateDir := t.TempDir()
	if err := writeDaemonShutdownReason(stateDir, "explicit_stop_command", 1001, map[string]string{
		"addr": "127.0.0.1:18333",
	}); err != nil {
		t.Fatalf("write shutdown reason: %v", err)
	}

	_, err := consumeDaemonShutdownReason(stateDir, 2002)
	if !os.IsNotExist(err) {
		t.Fatalf("consume mismatch err = %v, want not exist", err)
	}
	if _, err := os.Stat(shutdownReasonPath(stateDir)); err != nil {
		t.Fatalf("shutdown reason marker should remain for other pid, stat err=%v", err)
	}
}

func TestDaemonServiceShutdownStopsPlaywrightRuntime(t *testing.T) {
	svc := newDaemonService(Options{StateDir: t.TempDir()}, log.New(io.Discard, "", 0))
	pw := &playwright.Playwright{}
	svc.pw = pw

	originalStopFn := playwrightStopRuntimeFn
	defer func() {
		playwrightStopRuntimeFn = originalStopFn
	}()

	called := 0
	playwrightStopRuntimeFn = func(got *playwright.Playwright) error {
		called++
		if got != pw {
			t.Fatalf("playwright runtime = %p, want %p", got, pw)
		}
		return nil
	}

	if err := svc.shutdown(); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if called != 1 {
		t.Fatalf("playwright stop calls = %d, want 1", called)
	}
	if svc.pw != nil {
		t.Fatal("expected playwright runtime to be cleared")
	}
}

func TestStopDaemonWaitsForProcessExit(t *testing.T) {
	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "browser.pid")
	logFile := filepath.Join(tempDir, "browser.log")
	cmd := exec.Command("sh", "-c", "trap 'exit 0' TERM; while true; do sleep 0.1; done")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	}()
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}
	done := make(chan struct{})
	go func() {
		_, _ = cmd.Process.Wait()
		close(done)
	}()

	originalProcessAliveFn := processAliveFn
	defer func() {
		processAliveFn = originalProcessAliveFn
	}()
	processAliveFn = func(pid int) bool {
		if pid != cmd.Process.Pid {
			return originalProcessAliveFn(pid)
		}
		select {
		case <-done:
			return false
		default:
			return true
		}
	}

	result, err := StopDaemon(Options{
		StateDir:       tempDir,
		PIDFile:        pidFile,
		LogFile:        logFile,
		BrowserTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("StopDaemon: %v", err)
	}
	if result.Status != "stopped" {
		t.Fatalf("status = %q, want stopped", result.Status)
	}
	if result.Reason != "explicit_stop_command" {
		t.Fatalf("reason = %q, want explicit_stop_command", result.Reason)
	}
	select {
	case <-done:
	default:
		t.Fatalf("process %d should be stopped", cmd.Process.Pid)
	}
	if processAliveFn(cmd.Process.Pid) {
		t.Fatalf("process %d should be stopped", cmd.Process.Pid)
	}
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Fatalf("pid file should be removed, stat err=%v", err)
	}
	cmd.Process = nil
}

func TestPrepareManagedFlagsFallsBackToCreateWhenGetMisses(t *testing.T) {
	originalExecutablePathFn := browserPlaywrightExecutablePathFn
	originalGetFn := browserInstanceGetFn
	originalCreateFn := browserInstanceCreateFn
	defer func() {
		browserPlaywrightExecutablePathFn = originalExecutablePathFn
		browserInstanceGetFn = originalGetFn
		browserInstanceCreateFn = originalCreateFn
	}()

	root := t.TempDir()
	playwrightDir := filepath.Join(root, "browser", "playwright")
	instanceDir := filepath.Join(root, "browser", "instance")
	if err := os.MkdirAll(playwrightDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(instanceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	playwrightBinary := filepath.Join(playwrightDir, "browser_playwright")
	instanceBinary := filepath.Join(instanceDir, "browser_instance")
	if err := os.WriteFile(playwrightBinary, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(instanceBinary, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	browserPlaywrightExecutablePathFn = func() (string, error) {
		return playwrightBinary, nil
	}

	getCalls := 0
	createCalls := 0
	browserInstanceGetFn = func(path, agentID, chatID string, flags map[string]string, timeout time.Duration) (instanceCreateResult, error) {
		getCalls++
		return instanceCreateResult{}, errBrowserInstanceNotFound
	}
	browserInstanceCreateFn = func(path, agentID, chatID string, flags map[string]string, timeout time.Duration) (instanceCreateResult, error) {
		createCalls++
		if path != instanceBinary {
			t.Fatalf("path = %q, want %q", path, instanceBinary)
		}
		return instanceCreateResult{AgentID: agentID, ChatID: chatID, Port: 22001}, nil
	}

	flags, managed, err := prepareManagedFlags(map[string]string{
		"session": "agent-a@chat-001",
		"agentId": "agent-b",
		"chatId":  "other",
	})
	if err != nil {
		t.Fatalf("prepareManagedFlags: %v", err)
	}
	if !managed {
		t.Fatalf("expected managed=true")
	}
	if getCalls != 1 || createCalls != 1 {
		t.Fatalf("getCalls=%d createCalls=%d", getCalls, createCalls)
	}
	if flags["session"] != "agent-a@chat-001" {
		t.Fatalf("session = %q", flags["session"])
	}
	if flags["agentId"] != "agent-a" || flags["chatId"] != "chat-001" {
		t.Fatalf("identity flags = %+v", flags)
	}
	if flags["cdp"] != instanceCDPEndpoint(22001) {
		t.Fatalf("cdp = %q", flags["cdp"])
	}
}

func TestPrepareManagedFlagsSkipsWhenCDPAlreadyProvided(t *testing.T) {
	flags, managed, err := prepareManagedFlags(map[string]string{
		"session": "agent-a@chat-001",
		"cdp":     "http://127.0.0.1:9222",
	})
	if err != nil {
		t.Fatalf("prepareManagedFlags: %v", err)
	}
	if managed {
		t.Fatalf("expected managed=false when cdp is explicit")
	}
	if flags["cdp"] != "http://127.0.0.1:9222" {
		t.Fatalf("cdp = %q", flags["cdp"])
	}
}

func TestResolveManagedInstanceIdentityPrefersSession(t *testing.T) {
	item, ok := resolveManagedInstanceIdentity(map[string]string{
		"session": "agent-a@ctrip-home",
		"agentId": "agent-b",
		"chatId":  "other",
	})
	if !ok {
		t.Fatalf("expected managed identity")
	}
	if item.AgentID != "agent-a" || item.ChatID != "ctrip-home" || item.Session != "agent-a@ctrip-home" {
		t.Fatalf("identity = %+v", item)
	}
}

func TestResolveSessionNameUsesAgentAndChatWhenSessionMissing(t *testing.T) {
	got := resolveSessionName(map[string]string{
		"agentId": "agent-a",
		"chatId":  "chat-001",
	})
	if got != "agent-a@chat-001" {
		t.Fatalf("session = %q", got)
	}
}

func TestResolveSessionNamePrefersExplicitSession(t *testing.T) {
	got := resolveSessionName(map[string]string{
		"session": "agent-a@ctrip-home",
		"agentId": "agent-b",
		"chatId":  "ctrip-home",
	})
	if got != "agent-a@ctrip-home" {
		t.Fatalf("session = %q", got)
	}
}

func TestResolveSessionNameNormalizesUppercaseSession(t *testing.T) {
	got := resolveSessionName(map[string]string{
		"session": "AGENT-A@CTRIP-HOME",
	})
	if got != "agent-a@ctrip-home" {
		t.Fatalf("session = %q", got)
	}
}

func TestResolveSessionNameNormalizesUppercaseIdentity(t *testing.T) {
	got := resolveSessionName(map[string]string{
		"agentId": "AGENT-A",
		"chatId":  "CHAT-001",
	})
	if got != "agent-a@chat-001" {
		t.Fatalf("session = %q", got)
	}
}

func TestPrepareManagedFlagsNormalizesUppercaseIdentity(t *testing.T) {
	originalExecutablePathFn := browserPlaywrightExecutablePathFn
	originalGetFn := browserInstanceGetFn
	defer func() {
		browserPlaywrightExecutablePathFn = originalExecutablePathFn
		browserInstanceGetFn = originalGetFn
	}()

	root := t.TempDir()
	playwrightDir := filepath.Join(root, "browser", "playwright")
	instanceDir := filepath.Join(root, "browser", "instance")
	if err := os.MkdirAll(playwrightDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(instanceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	playwrightBinary := filepath.Join(playwrightDir, "browser_playwright")
	instanceBinary := filepath.Join(instanceDir, "browser_instance")
	if err := os.WriteFile(playwrightBinary, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(instanceBinary, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	browserPlaywrightExecutablePathFn = func() (string, error) {
		return playwrightBinary, nil
	}

	browserInstanceGetFn = func(path, agentID, chatID string, flags map[string]string, timeout time.Duration) (instanceCreateResult, error) {
		if agentID != "agent-a" || chatID != "chat-001" {
			t.Fatalf("unexpected normalized identity: %s %s", agentID, chatID)
		}
		if flags["agentId"] != "agent-a" || flags["chatId"] != "chat-001" {
			t.Fatalf("unexpected normalized flags: %+v", flags)
		}
		return instanceCreateResult{AgentID: agentID, ChatID: chatID, Port: 21001, CDP: instanceCDPEndpoint(21001)}, nil
	}

	flags, managed, err := prepareManagedFlags(map[string]string{
		"agentId": "AGENT-A",
		"chatId":  "CHAT-001",
	})
	if err != nil {
		t.Fatalf("prepareManagedFlags: %v", err)
	}
	if !managed {
		t.Fatalf("expected managed=true")
	}
	if flags["agentId"] != "agent-a" || flags["chatId"] != "chat-001" || flags["session"] != "agent-a@chat-001" {
		t.Fatalf("unexpected normalized flags: %+v", flags)
	}
}

func TestBrowserTimeoutFromFlagsDefaultsTo120Seconds(t *testing.T) {
	got, err := browserTimeoutFromFlags(map[string]string{})
	if err != nil {
		t.Fatalf("browserTimeoutFromFlags: %v", err)
	}
	if got != 120*time.Second {
		t.Fatalf("timeout = %s, want %s", got, 120*time.Second)
	}
}

func TestNormalizeOptionsDefaultsLogBesideExecutableAndDriverBesideExecutable(t *testing.T) {
	originalExecutablePathFn := browserPlaywrightExecutablePathFn
	defer func() {
		browserPlaywrightExecutablePathFn = originalExecutablePathFn
	}()

	root := t.TempDir()
	exeDir := filepath.Join(root, "browser", "playwright")
	if err := os.MkdirAll(exeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(exeDir, "playwright", "driver"), 0o755); err != nil {
		t.Fatal(err)
	}
	exePath := filepath.Join(exeDir, "browser_playwright")
	if err := os.WriteFile(exePath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	browserPlaywrightExecutablePathFn = func() (string, error) {
		return exePath, nil
	}

	got, err := normalizeOptions(Options{StateDir: filepath.Join(root, "state")})
	if err != nil {
		t.Fatalf("normalizeOptions: %v", err)
	}
	want := filepath.Join(exeDir, defaultLogDirName, defaultExecutableLogFileName(exePath))
	if got.LogFile != want {
		t.Fatalf("logFile = %q, want %q", got.LogFile, want)
	}
	if !filepath.IsAbs(got.DriverDir) {
		t.Fatalf("driverDir should be absolute, got %q", got.DriverDir)
	}
	if !strings.HasSuffix(filepath.ToSlash(got.DriverDir), "/browser/playwright/playwright/driver") {
		t.Fatalf("driverDir = %q, want suffix /browser/playwright/playwright/driver", got.DriverDir)
	}
}

func TestEnsurePlaywrightSkipsInstallWhenDriverReady(t *testing.T) {
	originalRunFn := playwrightRunFn
	originalInstallFn := playwrightInstallFn
	originalDriverUsableFn := playwrightDriverExecutableUsableFn
	defer func() {
		playwrightRunFn = originalRunFn
		playwrightInstallFn = originalInstallFn
		playwrightDriverExecutableUsableFn = originalDriverUsableFn
	}()

	driverDir := filepath.Join(t.TempDir(), "playwright", "driver")
	writePlaywrightDriverFixture(t, driverDir, playwrightDriverNodeFileName())
	playwrightDriverExecutableUsableFn = func(path string) bool {
		return strings.TrimSpace(path) == filepath.Join(driverDir, playwrightDriverNodeFileName())
	}

	installCalls := 0
	runCalls := 0
	playwrightInstallFn = func(opts *playwright.RunOptions) error {
		installCalls++
		return nil
	}
	playwrightRunFn = func(opts *playwright.RunOptions) (*playwright.Playwright, error) {
		runCalls++
		if opts.DriverDirectory != driverDir {
			t.Fatalf("driver directory = %q", opts.DriverDirectory)
		}
		return &playwright.Playwright{}, nil
	}

	svc := newDaemonService(Options{StateDir: t.TempDir(), DriverDir: driverDir}, log.New(io.Discard, "", 0))
	if err := svc.ensurePlaywright(); err != nil {
		t.Fatalf("ensurePlaywright: %v", err)
	}
	if installCalls != 0 {
		t.Fatalf("install should not be called, got %d", installCalls)
	}
	if runCalls != 1 {
		t.Fatalf("run should be called once, got %d", runCalls)
	}
}

func TestEnsurePlaywrightInstallsOnlyWhenDriverMissing(t *testing.T) {
	originalRunFn := playwrightRunFn
	originalInstallFn := playwrightInstallFn
	originalLookupEnvFn := playwrightDriverLookupEnvFn
	originalLookPathFn := playwrightDriverLookPathFn
	originalDriverUsableFn := playwrightDriverExecutableUsableFn
	defer func() {
		playwrightRunFn = originalRunFn
		playwrightInstallFn = originalInstallFn
		playwrightDriverLookupEnvFn = originalLookupEnvFn
		playwrightDriverLookPathFn = originalLookPathFn
		playwrightDriverExecutableUsableFn = originalDriverUsableFn
	}()

	driverDir := filepath.Join(t.TempDir(), "playwright", "driver")
	playwrightDriverLookupEnvFn = func(string) (string, bool) { return "", false }
	playwrightDriverLookPathFn = func(file string) (string, error) {
		if file != "node" {
			t.Fatalf("unexpected lookpath %q", file)
		}
		return "/usr/local/bin/node", nil
	}
	playwrightDriverExecutableUsableFn = func(path string) bool {
		return strings.TrimSpace(path) == "/usr/local/bin/node"
	}
	installCalls := 0
	runCalls := 0
	playwrightInstallFn = func(opts *playwright.RunOptions) error {
		installCalls++
		if opts.DriverDirectory != driverDir {
			t.Fatalf("driver directory = %q", opts.DriverDirectory)
		}
		if !opts.SkipInstallBrowsers {
			t.Fatal("install should skip Playwright browser downloads")
		}
		return nil
	}
	playwrightRunFn = func(opts *playwright.RunOptions) (*playwright.Playwright, error) {
		runCalls++
		if opts.DriverDirectory != driverDir {
			t.Fatalf("driver directory = %q", opts.DriverDirectory)
		}
		if !opts.SkipInstallBrowsers {
			t.Fatal("run should skip Playwright browser downloads")
		}
		return &playwright.Playwright{}, nil
	}

	svc := newDaemonService(Options{StateDir: t.TempDir(), DriverDir: driverDir}, log.New(io.Discard, "", 0))
	if err := svc.ensurePlaywright(); err != nil {
		t.Fatalf("ensurePlaywright: %v", err)
	}
	if installCalls != 1 {
		t.Fatalf("install should be called once, got %d", installCalls)
	}
	if runCalls != 1 {
		t.Fatalf("run should be called once, got %d", runCalls)
	}
}

func TestPlaywrightRunOptionsSkipBrowserInstall(t *testing.T) {
	svc := newDaemonService(Options{StateDir: t.TempDir(), DriverDir: filepath.Join(t.TempDir(), "driver")}, log.New(io.Discard, "", 0))
	opts := svc.playwrightRunOptions()
	if !opts.SkipInstallBrowsers {
		t.Fatal("SkipInstallBrowsers should be true")
	}
}

func TestPlaywrightDriverReadyUsesCurrentPlatformNodeName(t *testing.T) {
	originalGOOSFn := playwrightDriverRuntimeGOOSFn
	originalDriverUsableFn := playwrightDriverExecutableUsableFn
	defer func() {
		playwrightDriverRuntimeGOOSFn = originalGOOSFn
		playwrightDriverExecutableUsableFn = originalDriverUsableFn
	}()

	for _, tc := range []struct {
		name      string
		goos      string
		nodeFile  string
		otherFile string
	}{
		{name: "unix-like", goos: "darwin", nodeFile: "node", otherFile: "node.exe"},
		{name: "windows", goos: "windows", nodeFile: "node.exe", otherFile: "node"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			playwrightDriverRuntimeGOOSFn = func() string { return tc.goos }
			driverDir := filepath.Join(t.TempDir(), "playwright", "driver")
			writePlaywrightDriverFixture(t, driverDir, tc.nodeFile)
			playwrightDriverExecutableUsableFn = func(path string) bool {
				return strings.TrimSpace(path) == filepath.Join(driverDir, tc.nodeFile)
			}

			ready, err := PlaywrightDriverReady(driverDir)
			if err != nil {
				t.Fatalf("PlaywrightDriverReady: %v", err)
			}
			if !ready {
				t.Fatal("expected driver to be ready for current platform")
			}

			mismatchDir := filepath.Join(t.TempDir(), "playwright", "driver")
			writePlaywrightDriverFixture(t, mismatchDir, tc.otherFile)
			ready, err = PlaywrightDriverReady(mismatchDir)
			if err != nil {
				t.Fatalf("PlaywrightDriverReady mismatch: %v", err)
			}
			if ready {
				t.Fatal("expected driver with mismatched platform executable name to be rejected")
			}
		})
	}
}

func TestWaitForDaemonReadyUsesHealthEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	if !waitForDaemonReady(addr, time.Second) {
		t.Fatalf("expected daemon health check to succeed")
	}
}

func TestDoDaemonCommandWithRetryRetriesEOF(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/command" {
			http.NotFound(w, r)
			return
		}
		attempts++
		if attempts == 1 {
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatalf("response writer does not support hijack")
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Fatalf("hijack: %v", err)
			}
			_ = conn.Close()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"session":"demo","command":"goto"}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := doDaemonCommandWithRetry(ctx, Options{
		Addr: strings.TrimPrefix(server.URL, "http://"),
	}, []byte(`{"session":"demo","command":"goto"}`))
	if err != nil {
		t.Fatalf("doDaemonCommandWithRetry: %v", err)
	}
	defer resp.Body.Close()

	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestDoDaemonCommandWithRetryUsesDeadlineWindow(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/command" {
			http.NotFound(w, r)
			return
		}
		attempts++
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatalf("response writer does not support hijack")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Fatalf("hijack: %v", err)
		}
		_ = conn.Close()
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 750*time.Millisecond)
	defer cancel()

	_, err := doDaemonCommandWithRetry(ctx, Options{
		Addr: strings.TrimPrefix(server.URL, "http://"),
	}, []byte(`{"session":"demo","command":"goto"}`))
	if err == nil {
		t.Fatalf("expected retry loop to end with error")
	}
	if attempts < 4 {
		t.Fatalf("attempts = %d, want at least 4 within deadline window", attempts)
	}
}

func TestShouldRetryDaemonCommand(t *testing.T) {
	if !shouldRetryDaemonCommand(io.EOF) {
		t.Fatalf("expected EOF to be retried")
	}
	if !shouldRetryDaemonCommand(&net.OpError{Err: io.EOF}) {
		t.Fatalf("expected wrapped EOF to be retried")
	}
	if !shouldRetryDaemonCommand(&net.OpError{Err: errors.New("connect: connection refused")}) {
		t.Fatalf("expected connection refused to be retried")
	}
	if shouldRetryDaemonCommand(context.DeadlineExceeded) {
		t.Fatalf("deadline exceeded should not be retried")
	}
}

func TestRecordDaemonCommandTimeoutAndClear(t *testing.T) {
	stateDir := t.TempDir()

	count, err := recordDaemonCommandTimeout(stateDir)
	if err != nil {
		t.Fatalf("record first timeout: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}

	count, err = recordDaemonCommandTimeout(stateDir)
	if err != nil {
		t.Fatalf("record second timeout: %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}

	if err := clearDaemonCommandTimeoutState(stateDir); err != nil {
		t.Fatalf("clear timeout state: %v", err)
	}

	count, err = recordDaemonCommandTimeout(stateDir)
	if err != nil {
		t.Fatalf("record after clear: %v", err)
	}
	if count != 1 {
		t.Fatalf("count after clear = %d, want 1", count)
	}
}

func TestHandleDaemonCommandTimeoutKillsOnThirdTimeout(t *testing.T) {
	stateDir := t.TempDir()
	pidFile := filepath.Join(stateDir, "browser_playwright.pid")
	if err := os.WriteFile(pidFile, []byte("99999"), 0o644); err != nil {
		t.Fatal(err)
	}

	originalForceStopFn := forceStopDaemonProcessFn
	killed := 0
	forceStopDaemonProcessFn = func(string) error {
		killed++
		return nil
	}
	defer func() {
		forceStopDaemonProcessFn = originalForceStopFn
	}()

	opts := Options{StateDir: stateDir, PIDFile: pidFile, BrowserTimeout: 5 * time.Second}
	err := handleDaemonCommandTimeout(opts, "goto")
	if err == nil || !strings.Contains(err.Error(), "1/3") {
		t.Fatalf("first timeout err = %v", err)
	}
	err = handleDaemonCommandTimeout(opts, "goto")
	if err == nil || !strings.Contains(err.Error(), "2/3") {
		t.Fatalf("second timeout err = %v", err)
	}
	err = handleDaemonCommandTimeout(opts, "goto")
	if err == nil || !strings.Contains(err.Error(), "daemon stopped after 3 consecutive timeouts") {
		t.Fatalf("third timeout err = %v", err)
	}
	if killed != 1 {
		t.Fatalf("killed = %d, want 1", killed)
	}
}

func TestHandleDaemonCommandTimeoutUsesConfiguredThreshold(t *testing.T) {
	stateDir := t.TempDir()

	originalForceStopFn := forceStopDaemonProcessFn
	killed := 0
	forceStopDaemonProcessFn = func(string) error {
		killed++
		return nil
	}
	defer func() {
		forceStopDaemonProcessFn = originalForceStopFn
	}()

	opts := Options{StateDir: stateDir, PIDFile: filepath.Join(stateDir, "browser_playwright.pid"), BrowserTimeout: 5 * time.Second, BrowserRetry: 2}
	err := handleDaemonCommandTimeout(opts, "goto")
	if err == nil || !strings.Contains(err.Error(), "1/2") {
		t.Fatalf("first timeout err = %v", err)
	}
	err = handleDaemonCommandTimeout(opts, "goto")
	if err == nil || !strings.Contains(err.Error(), "daemon stopped after 2 consecutive timeouts") {
		t.Fatalf("second timeout err = %v", err)
	}
	if killed != 1 {
		t.Fatalf("killed = %d, want 1", killed)
	}
}

func TestPlaywrightDriverReadyReportsMissingEntries(t *testing.T) {
	driverDir := filepath.Join(t.TempDir(), "playwright", "driver")
	if err := os.MkdirAll(filepath.Join(driverDir, "package"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(driverDir, "node"), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	ready, err := playwrightDriverReady(driverDir)
	if err != nil {
		t.Fatalf("playwrightDriverReady error: %v", err)
	}
	if ready {
		t.Fatalf("expected driverDir to be incomplete")
	}
}

func TestEnsurePlaywrightReturnsRunErrorWhenDriverReady(t *testing.T) {
	originalRunFn := playwrightRunFn
	originalInstallFn := playwrightInstallFn
	originalDriverUsableFn := playwrightDriverExecutableUsableFn
	defer func() {
		playwrightRunFn = originalRunFn
		playwrightInstallFn = originalInstallFn
		playwrightDriverExecutableUsableFn = originalDriverUsableFn
	}()

	driverDir := filepath.Join(t.TempDir(), "playwright", "driver")
	writePlaywrightDriverFixture(t, driverDir, playwrightDriverNodeFileName())
	playwrightDriverExecutableUsableFn = func(path string) bool {
		return strings.TrimSpace(path) == filepath.Join(driverDir, playwrightDriverNodeFileName())
	}

	wantErr := errors.New("run failed")
	playwrightInstallFn = func(opts *playwright.RunOptions) error {
		t.Fatal("install should not be called when driver is ready")
		return nil
	}
	playwrightRunFn = func(opts *playwright.RunOptions) (*playwright.Playwright, error) {
		return nil, wantErr
	}

	svc := newDaemonService(Options{StateDir: t.TempDir(), DriverDir: driverDir}, log.New(io.Discard, "", 0))
	err := svc.ensurePlaywright()
	if !errors.Is(err, wantErr) {
		t.Fatalf("ensurePlaywright error = %v, want %v", err, wantErr)
	}
}

func TestLogNavigationDiagnostic(t *testing.T) {
	buf := &bytes.Buffer{}
	svc := newDaemonService(Options{StateDir: t.TempDir()}, log.New(buf, "", 0))
	sess := &liveSession{config: SessionConfig{Name: "demo"}}

	svc.logNavigationDiagnostic(sess, "goto", "https://example.com", 1500*time.Millisecond)

	line := strings.TrimSpace(buf.String())
	if line == "" {
		t.Fatalf("expected log output")
	}
	var item map[string]any
	if err := json.Unmarshal([]byte(line), &item); err != nil {
		t.Fatalf("unmarshal log line: %v raw=%s", err, line)
	}
	if item["event"] != "navigation" || item["command"] != "goto" {
		t.Fatalf("unexpected log payload: %+v", item)
	}
	if item["gotoCostMs"] != float64(1500) {
		t.Fatalf("gotoCostMs = %v", item["gotoCostMs"])
	}
}

func TestRewriteNavigationErrorTooManyRedirects(t *testing.T) {
	err := rewriteNavigationError(
		"https://vacations.ctrip.com/tour/detail/p-f2--ta162--s2--o0--t0--g0--l0.html",
		fmt.Errorf("Frame.Goto https://vacations.ctrip.com/tour/detail/p-f2--ta162--s2--o0--t0--g0--l0.html: playwright: Protocol error (Page.navigate): Network error: Too many redirects: https://vacations.ctrip.com/tour/detail/p-f2--ta162--s2--o0--t0--g0--l0.html"),
	)
	if err == nil {
		t.Fatalf("expected rewritten error")
	}
	if !strings.Contains(err.Error(), "redirect loop") {
		t.Fatalf("expected redirect loop hint, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "Too many redirects") {
		t.Fatalf("expected original error detail, got %q", err.Error())
	}
}

func TestRewriteNavigationErrorPassThrough(t *testing.T) {
	original := fmt.Errorf("playwright: timeout: Timeout 30000ms exceeded")
	got := rewriteNavigationError("https://example.com", original)
	if got != original {
		t.Fatalf("expected original error passthrough, got %v", got)
	}
}

func TestLogCommandDiagnostic(t *testing.T) {
	buf := &bytes.Buffer{}
	svc := newDaemonService(Options{StateDir: t.TempDir()}, log.New(buf, "", 0))

	svc.logCommandDiagnostic("finish", "demo", CommandRequest{
		Session: "demo",
		Command: "goto",
		Args:    []string{"https://example.com"},
		Flags:   map[string]string{"headed": "true"},
	}, &CommandResult{
		Session: "demo",
		Command: "goto",
		Message: "ok",
		Page:    &PageInfo{URL: "https://example.com", Title: "Example", Index: 0},
	}, nil, 1500*time.Millisecond)

	line := strings.TrimSpace(buf.String())
	if line == "" {
		t.Fatalf("expected log output")
	}
	var item map[string]any
	if err := json.Unmarshal([]byte(line), &item); err != nil {
		t.Fatalf("unmarshal log line: %v raw=%s", err, line)
	}
	if item["event"] != "command" || item["stage"] != "finish" || item["command"] != "goto" {
		t.Fatalf("unexpected log payload: %+v", item)
	}
	if item["elapsedMs"] != float64(1500) {
		t.Fatalf("elapsedMs = %v", item["elapsedMs"])
	}
	result, ok := item["result"].(map[string]any)
	if !ok {
		t.Fatalf("result = %#v", item["result"])
	}
	if result["message"] != "ok" {
		t.Fatalf("result.message = %v", result["message"])
	}
}
