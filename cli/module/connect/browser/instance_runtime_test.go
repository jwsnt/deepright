package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtimepaths"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestBrowserAllocatePortsUseStableIdentityHash(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	portsChecked := []int{}
	browserPortAvailableFn = func(port int) bool {
		portsChecked = append(portsChecked, port)
		return true
	}

	createPort, err := browserAllocateCreatePort("Agent-A", "Chat-001", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	initPort, err := browserAllocateInitPort("Agent-A", "Chat-001", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	wantPort := browserHashedPort("Agent-A", "Chat-001")
	if createPort != wantPort {
		t.Fatalf("createPort = %d, want %d", createPort, wantPort)
	}
	if initPort != wantPort {
		t.Fatalf("initPort = %d, want %d", initPort, wantPort)
	}
	if len(portsChecked) != 2 || portsChecked[0] != wantPort || portsChecked[1] != wantPort {
		t.Fatalf("portsChecked = %v, want [%d %d]", portsChecked, wantPort, wantPort)
	}
}

func TestBrowserAllocateCreatePortFailsWhenStablePortIsTracked(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	targetPort := browserHashedPort("Agent-A", "Chat-001")
	browserPortAvailableFn = func(port int) bool {
		t.Fatalf("browserPortAvailableFn should not be called when port is already tracked, got %d", port)
		return true
	}

	_, err := browserAllocateCreatePort("Agent-A", "Chat-001", []browserInstanceRecord{{AgentID: "other", ChatID: "chat-002", Port: targetPort}}, nil)
	if err == nil || !strings.Contains(err.Error(), "stable browser instance port occupied") {
		t.Fatalf("err = %v, want occupied stable port error", err)
	}
}

func TestBrowserAllocateCreatePortFailsWhenStablePortIsUnavailable(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	targetPort := browserHashedPort("Agent-A", "Chat-001")
	portsChecked := []int{}
	browserPortAvailableFn = func(port int) bool {
		portsChecked = append(portsChecked, port)
		return false
	}

	_, err := browserAllocateCreatePort("Agent-A", "Chat-001", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "stable browser instance port unavailable") {
		t.Fatalf("err = %v, want unavailable stable port error", err)
	}
	if len(portsChecked) != 1 || portsChecked[0] != targetPort {
		t.Fatalf("portsChecked = %v, want [%d]", portsChecked, targetPort)
	}
}

func TestBrowserCreateInstancePersistsStateAndCopiesUserData(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	pluginDir := t.TempDir()
	chromePath := filepath.Join(pluginDir, "chrome")
	if err := os.WriteFile(chromePath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	sourceDir := filepath.Join(pluginDir, "source-profile")
	if err := os.MkdirAll(filepath.Join(sourceDir, "Default"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "Default", "Preferences"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	browserExecutablePathFn = func() (string, error) {
		return filepath.Join(pluginDir, "browser"), nil
	}
	browserRuntimeGOOSFn = func() string { return "darwin" }
	browserResolveSystemChromeUserDataDirFn = func(flags map[string]string) (string, error) {
		return sourceDir, nil
	}
	browserStartChromeFn = func(chromePath string, port int, profileDir, headlessMode, logPath string) (int, error) {
		return 9001, nil
	}
	browserWaitForPortFn = func(pid, port int, timeoutDuration time.Duration) error {
		return nil
	}
	browserCDPVersionFn = func(port int) (browserCDPVersion, error) {
		return browserCDPVersion{WebSocketDebuggerURL: instanceCDPEndpoint(port)}, nil
	}

	flags := map[string]string{
		"state":   filepath.Join(pluginDir, "browser_instance.json"),
		"chrome":  chromePath,
		"agentId": "Agent-A",
		"chatId":  "Chat-001",
	}
	item, err := browserCreateInstance(flags)
	if err != nil {
		t.Fatal(err)
	}
	if item.AgentID != "agent-a" || item.ChatID != "chat-001" {
		t.Fatalf("unexpected item: %+v", item)
	}

	profileDir := filepath.Join(pluginDir, "agent-a", "chrome_"+strconv.Itoa(item.Port))
	if item.ProfileDir != profileDir {
		t.Fatalf("item.ProfileDir = %q, want %q", item.ProfileDir, profileDir)
	}
	if _, err := os.Stat(filepath.Join(profileDir, "Default", "Preferences")); err != nil {
		t.Fatalf("copied profile missing: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(pluginDir, "browser_instance.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"agentId": "agent-a"`) {
		t.Fatalf("state file = %s", string(data))
	}
}

func TestBrowserInitInstanceForcesHeadedAndRunsDestroyBeforeCreate(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	pluginDir := t.TempDir()
	chromePath := filepath.Join(pluginDir, "chrome")
	if err := os.WriteFile(chromePath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	sourceDir := filepath.Join(pluginDir, "source-profile")
	if err := os.MkdirAll(filepath.Join(sourceDir, "Default"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "Default", "Preferences"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	browserExecutablePathFn = func() (string, error) {
		return filepath.Join(pluginDir, "browser"), nil
	}
	browserResolveSystemChromeUserDataDirFn = func(flags map[string]string) (string, error) {
		return sourceDir, nil
	}
	instanceGetFn = func(flags map[string]string) (browserInstanceRecord, error) {
		return browserInstanceRecord{}, fmt.Errorf("instance not found: agentId=%s chatId=%s", flags["agentId"], flags["chatId"])
	}
	destroyCalled := false
	instanceDestroyFn = func(flags map[string]string) error {
		destroyCalled = true
		return nil
	}
	browserWSLCommandStartProcessFn = func(chromePath string, args []string) (int, func() error, error) {
		t.Fatal("mac init should not use WSL launcher")
		return 0, nil, nil
	}
	waitCalled := false
	capturedArgs := []string{}
	browserAttachedCommandStartProcessFn = func(chromePath string, args []string, logPath string) (int, func() error, error) {
		capturedArgs = append([]string{}, args...)
		return 9003, func() error {
			waitCalled = true
			return nil
		}, nil
	}
	browserAttachedCommandWaitForExitFn = browserWaitForChromeProcessExit
	browserWaitForPortFn = func(pid, port int, timeoutDuration time.Duration) error {
		return nil
	}
	browserCDPVersionFn = func(port int) (browserCDPVersion, error) {
		return browserCDPVersion{WebSocketDebuggerURL: instanceCDPEndpoint(port)}, nil
	}

	item, err := browserInitInstance(map[string]string{
		"state":   filepath.Join(pluginDir, "browser_instance.json"),
		"chrome":  chromePath,
		"agentId": "Agent-A",
		"chatId":  "Chat-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.AgentID != "agent-a" || item.ChatID != "chat-001" {
		t.Fatalf("unexpected item: %+v", item)
	}
	if strings.Contains(strings.Join(capturedArgs, " "), "--headless=") {
		t.Fatalf("captured args should not include headless flag: %+v", capturedArgs)
	}
	if !strings.Contains(strings.Join(capturedArgs, " "), "--remote-debugging-address=0.0.0.0") {
		t.Fatalf("captured args missing sync init address: %+v", capturedArgs)
	}
	if !destroyCalled {
		t.Fatal("destroy should still run so init can clear stale state or ports before create")
	}
	if waitCalled {
		t.Fatal("mac init should return without waiting for the attached chrome process to exit")
	}
	got, err := browserGetInstance(map[string]string{
		"state":   filepath.Join(pluginDir, "browser_instance.json"),
		"agentId": "Agent-A",
		"chatId":  "Chat-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Port != item.Port || got.PID != item.PID || got.CDP != item.CDP {
		t.Fatalf("got = %+v, want %+v", got, item)
	}
}

func TestBrowserInitInstanceDestroysExistingInstanceBeforeCreate(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	pluginDir := t.TempDir()
	chromePath := filepath.Join(pluginDir, "chrome")
	if err := os.WriteFile(chromePath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	sourceDir := filepath.Join(pluginDir, "source-profile")
	if err := os.MkdirAll(filepath.Join(sourceDir, "Default"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "Default", "Preferences"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	browserExecutablePathFn = func() (string, error) {
		return filepath.Join(pluginDir, "browser"), nil
	}
	browserRuntimeGOOSFn = func() string { return "darwin" }
	browserResolveSystemChromeUserDataDirFn = func(flags map[string]string) (string, error) {
		return sourceDir, nil
	}
	order := []string{}
	instanceGetFn = func(flags map[string]string) (browserInstanceRecord, error) {
		order = append(order, "get")
		return browserInstanceRecord{AgentID: "agent-a", ChatID: "chat-001", Port: 22001, PID: 7001, CDP: instanceCDPEndpoint(22001)}, nil
	}
	instanceDestroyFn = func(flags map[string]string) error {
		order = append(order, "destroy")
		return nil
	}
	browserWSLCommandStartProcessFn = func(chromePath string, args []string) (int, func() error, error) {
		t.Fatal("mac init should not use WSL launcher")
		return 0, nil, nil
	}
	browserAttachedCommandStartProcessFn = func(chromePath string, args []string, logPath string) (int, func() error, error) {
		order = append(order, "create")
		if strings.Contains(strings.Join(args, " "), "--headless=") {
			t.Fatalf("init args should not include headless flag: %+v", args)
		}
		return 9004, func() error {
			order = append(order, "wait")
			return nil
		}, nil
	}
	browserAttachedCommandWaitForExitFn = browserWaitForChromeProcessExit
	browserWaitForPortFn = func(pid, port int, timeoutDuration time.Duration) error {
		return nil
	}
	browserCDPVersionFn = func(port int) (browserCDPVersion, error) {
		return browserCDPVersion{WebSocketDebuggerURL: instanceCDPEndpoint(port)}, nil
	}

	_, err := browserInitInstance(map[string]string{
		"state":   filepath.Join(pluginDir, "browser_instance.json"),
		"chrome":  chromePath,
		"agentId": "Agent-A",
		"chatId":  "Chat-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(order, ",") != "get,destroy,create" {
		t.Fatalf("order = %q, want get,destroy,create", strings.Join(order, ","))
	}
}

func TestBrowserShouldSkipClonedChromeUserDataPathSkipsRuntimeMarkers(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "RunningChromeVersion", want: "skip chrome runtime version marker"},
		{path: "SingletonLock", want: "skip chrome runtime lock path"},
		{path: "SingletonCookie", want: "skip chrome runtime lock path"},
		{path: "SingletonSocket", want: "skip chrome runtime lock path"},
		{path: "DevToolsActivePort", want: "skip chrome runtime lock path"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, reason := browserShouldSkipClonedChromeUserDataPath(tt.path)
			if !got {
				t.Fatalf("browserShouldSkipClonedChromeUserDataPath(%q) = false, want true", tt.path)
			}
			if reason != tt.want {
				t.Fatalf("reason = %q, want %q", reason, tt.want)
			}
		})
	}
}

func TestBrowserShouldSkipClonedChromeUserDataPathKeepsRegularProfileData(t *testing.T) {
	got, reason := browserShouldSkipClonedChromeUserDataPath(filepath.Join("Default", "Preferences"))
	if got {
		t.Fatalf("browserShouldSkipClonedChromeUserDataPath(Default/Preferences) = true, reason=%q", reason)
	}
}

func TestBrowserCreateInstanceReusesChatScopedRecordAcrossAgentChanges(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	statePath := filepath.Join(t.TempDir(), "browser_instance.json")
	browserProcessExistsFn = func(pid int) bool {
		return pid == 9011
	}
	browserCDPVersionFn = func(port int) (browserCDPVersion, error) {
		if port != 22001 {
			t.Fatalf("unexpected port: %d", port)
		}
		return browserCDPVersion{WebSocketDebuggerURL: instanceCDPEndpoint(port)}, nil
	}
	if err := browserSaveInstances(statePath, []browserInstanceStateRecord{{
		AgentID:      "agent-a",
		ChatID:       "chat-001",
		Port:         22001,
		PID:          9011,
		CDP:          instanceCDPEndpoint(22001),
		LastActiveAt: time.Now().Add(-time.Minute).Format(time.RFC3339),
	}}); err != nil {
		t.Fatal(err)
	}

	item, err := browserCreateInstance(map[string]string{
		"state":   statePath,
		"agentId": "Agent-B",
		"chatId":  "Chat-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.AgentID != "agent-a" || item.ChatID != "chat-001" {
		t.Fatalf("unexpected item: %+v", item)
	}
	if item.Port != 22001 || item.PID != 9011 {
		t.Fatalf("unexpected item: %+v", item)
	}
}

func TestBrowserGetInstanceFallsBackToChatScopedRecord(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	statePath := filepath.Join(t.TempDir(), "browser_instance.json")
	browserProcessExistsFn = func(pid int) bool {
		return pid == 9011
	}
	browserCDPVersionFn = func(port int) (browserCDPVersion, error) {
		if port != 22001 {
			t.Fatalf("unexpected port: %d", port)
		}
		return browserCDPVersion{WebSocketDebuggerURL: instanceCDPEndpoint(port)}, nil
	}
	if err := browserSaveInstances(statePath, []browserInstanceStateRecord{{
		AgentID:      "agent-a",
		ChatID:       "chat-001",
		Port:         22001,
		PID:          9011,
		CDP:          instanceCDPEndpoint(22001),
		LastActiveAt: time.Now().Add(-time.Minute).Format(time.RFC3339),
	}}); err != nil {
		t.Fatal(err)
	}

	item, err := browserGetInstance(map[string]string{
		"state":   statePath,
		"agentId": "Agent-B",
		"chatId":  "Chat-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.AgentID != "agent-a" || item.ChatID != "chat-001" {
		t.Fatalf("unexpected item: %+v", item)
	}
}

func TestBrowserDestroyInstanceSupportsChatScopedShutdownWithoutAgent(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	statePath := filepath.Join(t.TempDir(), "browser_instance.json")
	browserProcessExistsFn = func(pid int) bool {
		return pid == 9011
	}
	browserCDPVersionFn = func(port int) (browserCDPVersion, error) {
		if port != 22001 {
			t.Fatalf("unexpected port: %d", port)
		}
		return browserCDPVersion{WebSocketDebuggerURL: instanceCDPEndpoint(port)}, nil
	}
	if err := browserSaveInstances(statePath, []browserInstanceStateRecord{{
		AgentID:      "agent-a",
		ChatID:       "chat-001",
		Port:         22001,
		PID:          9011,
		CDP:          instanceCDPEndpoint(22001),
		LastActiveAt: time.Now().Add(-time.Minute).Format(time.RFC3339),
	}}); err != nil {
		t.Fatal(err)
	}

	terminated := []browserInstanceRecord{}
	browserTerminateManagedInstanceFn = func(item browserInstanceRecord) error {
		terminated = append(terminated, item)
		return nil
	}

	if err := browserDestroyInstance(map[string]string{
		"state":  statePath,
		"chatId": "Chat-001",
	}); err != nil {
		t.Fatal(err)
	}
	if len(terminated) != 1 {
		t.Fatalf("terminated = %v, want one record", terminated)
	}
	if terminated[0].AgentID != "agent-a" || terminated[0].ChatID != "chat-001" {
		t.Fatalf("unexpected terminated record: %+v", terminated[0])
	}
	if _, err := os.Stat(statePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("state file should be removed, err = %v", err)
	}
}

func TestBrowserResolveChromeHeadlessModeFromPluginMetaTrue(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	exeDir := t.TempDir()
	browserExecutablePathFn = func() (string, error) {
		return writeBrowserBinaryFixture(t, exeDir), nil
	}
	connectBin := filepath.Join(t.TempDir(), "integration")
	if err := os.WriteFile(connectBin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeBrowserRuntimeRecordFixture(t, connectBin)

	browserCookieCommandContextFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "cat <<'EOF'\n{\"key\":\"browser\",\"meta\":{\"headless\":\"TRUE\"}}\nEOF\n")
	}

	mode, ok, err := browserResolveChromeHeadlessModeFromPluginMeta(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || mode != browserDefaultHeadlessMode {
		t.Fatalf("mode=%q ok=%v", mode, ok)
	}
}

func TestBrowserResolveChromeHeadlessModeFromPluginMetaFalse(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	exeDir := t.TempDir()
	browserExecutablePathFn = func() (string, error) {
		return writeBrowserBinaryFixture(t, exeDir), nil
	}
	connectBin := filepath.Join(t.TempDir(), "integration")
	if err := os.WriteFile(connectBin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeBrowserRuntimeRecordFixture(t, connectBin)

	browserCookieCommandContextFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "cat <<'EOF'\n{\"key\":\"browser\",\"meta\":{\"headless\":\"FALSE\"}}\nEOF\n")
	}

	mode, ok, err := browserResolveChromeHeadlessModeFromPluginMeta(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || mode != "none" {
		t.Fatalf("mode=%q ok=%v", mode, ok)
	}
}

func TestBrowserResolveChromeHeadlessModeFromPluginMetaInvalidDefaultsHeadless(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	exeDir := t.TempDir()
	browserExecutablePathFn = func() (string, error) {
		return writeBrowserBinaryFixture(t, exeDir), nil
	}
	connectBin := filepath.Join(t.TempDir(), "integration")
	if err := os.WriteFile(connectBin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeBrowserRuntimeRecordFixture(t, connectBin)

	browserCookieCommandContextFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "cat <<'EOF'\n{\"key\":\"browser\",\"meta\":{\"headless\":\"maybe\"}}\nEOF\n")
	}

	mode, ok, err := browserResolveChromeHeadlessModeFromPluginMeta(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || mode != browserDefaultHeadlessMode {
		t.Fatalf("mode=%q ok=%v", mode, ok)
	}
}

func TestBrowserResolveChromeHeadlessModeFromPluginMetaEmptyFallsBack(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	exeDir := t.TempDir()
	browserExecutablePathFn = func() (string, error) {
		return writeBrowserBinaryFixture(t, exeDir), nil
	}
	connectBin := filepath.Join(t.TempDir(), "integration")
	if err := os.WriteFile(connectBin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeBrowserRuntimeRecordFixture(t, connectBin)

	browserCookieCommandContextFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "cat <<'EOF'\n{\"key\":\"browser\",\"meta\":{\"headless\":\"\"}}\nEOF\n")
	}

	mode, ok, err := browserResolveChromeHeadlessModeFromPluginMeta(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if ok || mode != "" {
		t.Fatalf("mode=%q ok=%v", mode, ok)
	}
}

func TestBrowserResolveWSLBaseUserDataDirUsesDefaultPath(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	expected := filepath.Join(t.TempDir(), "chrome_base")
	browserWSLBaseUserDataDirFn = func() string {
		return expected
	}

	got, err := browserResolveWSLBaseUserDataDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != expected {
		t.Fatalf("base dir = %q, want %q", got, expected)
	}
}

func TestBrowserDefaultWSLPathsUseHomeDirectory(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	homeDir := filepath.Join(t.TempDir(), "home", "tester")
	browserUserHomeDirFn = func() (string, error) {
		return homeDir, nil
	}

	wantRoot := filepath.Join(homeDir, "deepright")
	if got := browserDefaultWSLRuntimeRoot(); got != wantRoot {
		t.Fatalf("runtime root = %q, want %q", got, wantRoot)
	}
	if got := browserWSLAgentRootDirFn(); got != filepath.Join(wantRoot, "agent") {
		t.Fatalf("agent root = %q, want %q", got, filepath.Join(wantRoot, "agent"))
	}
	if got := browserWSLBaseUserDataDirFn(); got != filepath.Join(wantRoot, "chrome_base") {
		t.Fatalf("chrome base = %q, want %q", got, filepath.Join(wantRoot, "chrome_base"))
	}
}

func TestBrowserResolveWSLBaseUserDataDirRejectsRelativePath(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	browserWSLBaseUserDataDirFn = func() string {
		return "relative/chrome_base"
	}

	if _, err := browserResolveWSLBaseUserDataDir(); err == nil {
		t.Fatal("expected relative WSL base dir to fail")
	}
}

func TestBrowserCreateInstanceUsesWSLAgentWorkspaceAndSkipsProfileClone(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	pluginDir := t.TempDir()
	chromePath := filepath.Join(pluginDir, "chrome.exe")
	if err := os.WriteFile(chromePath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	launcherArgsPath := filepath.Join(pluginDir, "launcher.args")
	launcherPath := filepath.Join(pluginDir, "browser_launcher.sh")
	launcherScript := fmt.Sprintf(`#!/bin/bash
printf '%%s\n' "$@" > %q
printf '%%s\n' '{"status":0,"pid":9002,"port":23091,"ws":"ws://127.0.0.1:23091/devtools/browser/test","http":"http://localhost:23091","user-data-dir":"C:\\ProgramData\\deepright\\chrome_ab12","message":""}'
`, launcherArgsPath)
	if err := os.WriteFile(launcherPath, []byte(launcherScript), 0o755); err != nil {
		t.Fatal(err)
	}

	browserExecutablePathFn = func() (string, error) {
		return filepath.Join(pluginDir, "browser"), nil
	}
	browserWSLDetectFn = func() (bool, error) {
		return true, nil
	}
	browserResolveSystemChromeUserDataDirFn = func(flags map[string]string) (string, error) {
		t.Fatal("WSL create should not clone the system Chrome user-data-dir")
		return "", nil
	}
	browserResolveLauncherScriptPathFn = func(flags map[string]string) (string, error) {
		return launcherPath, nil
	}

	item, err := browserCreateInstance(map[string]string{
		"state":    filepath.Join(pluginDir, "browser_instance.json"),
		"chrome":   chromePath,
		"agentId":  "Agent-A",
		"chatId":   "Chat-001",
		"headless": "none",
	})
	if err != nil {
		t.Fatal(err)
	}

	expectedProfileDir := `C:\ProgramData\deepright\chrome_ab12`
	if item.ProfileDir != expectedProfileDir {
		t.Fatalf("item.ProfileDir = %q, want %q", item.ProfileDir, expectedProfileDir)
	}
	if item.PID != 9002 {
		t.Fatalf("item.PID = %d, want 9002", item.PID)
	}
	if item.Port != 23091 {
		t.Fatalf("item.Port = %d, want 23091", item.Port)
	}
	argsData, err := os.ReadFile(launcherArgsPath)
	if err != nil {
		t.Fatal(err)
	}
	gotArgs := strings.Split(strings.TrimSpace(string(argsData)), "\n")
	wantArgs := []string{"--agentId", "agent-a", "--chatId", "chat-001", "--headless", "false", "--chrome", chromePath}
	if strings.Join(gotArgs, ",") != strings.Join(wantArgs, ",") {
		t.Fatalf("launcher args = %v, want %v", gotArgs, wantArgs)
	}
}

func TestBrowserInitInstanceUsesWindowsStyleWSLUserDataDir(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	pluginDir := t.TempDir()
	chromePath := filepath.Join(pluginDir, "chrome.exe")
	if err := os.WriteFile(chromePath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	launcherArgsPath := filepath.Join(pluginDir, "launcher.args")
	launcherForceRecreatePath := filepath.Join(pluginDir, "launcher.force-recreate")
	launcherPath := filepath.Join(pluginDir, "browser_launcher.sh")
	launcherScript := fmt.Sprintf(`#!/bin/bash
printf '%%s\n' "$@" > %q
printf '%%s\n' "$DEEPRIGHT_BROWSER_WSL_FORCE_RECREATE" > %q
printf '%%s\n' '{"status":0,"pid":9003,"port":23092,"ws":"ws://127.0.0.1:23092/devtools/browser/test","http":"http://localhost:23092","user-data-dir":"C:\\ProgramData\\deepright\\chrome_xy99","message":""}'
`, launcherArgsPath, launcherForceRecreatePath)
	if err := os.WriteFile(launcherPath, []byte(launcherScript), 0o755); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(pluginDir, "browser_instance.json")

	browserExecutablePathFn = func() (string, error) {
		return filepath.Join(pluginDir, "browser"), nil
	}
	browserWSLDetectFn = func() (bool, error) {
		return true, nil
	}
	browserResolveLauncherScriptPathFn = func(flags map[string]string) (string, error) {
		return launcherPath, nil
	}
	instanceGetFn = func(flags map[string]string) (browserInstanceRecord, error) {
		return browserInstanceRecord{}, fmt.Errorf("instance not found: agentId=%s chatId=%s", flags["agentId"], flags["chatId"])
	}
	browserCDPVersionFn = func(port int) (browserCDPVersion, error) {
		if port != 23092 {
			t.Fatalf("unexpected port: %d", port)
		}
		return browserCDPVersion{WebSocketDebuggerURL: instanceCDPEndpoint(port)}, nil
	}

	item, err := browserInitInstance(map[string]string{
		"state":   statePath,
		"chrome":  chromePath,
		"agentId": "Agent-A",
		"chatId":  "Chat-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	expectedProfileDir := `C:\ProgramData\deepright\chrome_xy99`
	if item.ProfileDir != expectedProfileDir {
		t.Fatalf("item.ProfileDir = %q, want %q", item.ProfileDir, expectedProfileDir)
	}
	if item.PID != 9003 {
		t.Fatalf("item.PID = %d, want 9003", item.PID)
	}
	if item.Port != 23092 {
		t.Fatalf("item.Port = %d, want 23092", item.Port)
	}
	argsData, err := os.ReadFile(launcherArgsPath)
	if err != nil {
		t.Fatal(err)
	}
	gotArgs := strings.Split(strings.TrimSpace(string(argsData)), "\n")
	wantArgs := []string{"--agentId", "agent-a", "--chatId", "chat-001", "--headless", "false", "--chrome", chromePath}
	if strings.Join(gotArgs, ",") != strings.Join(wantArgs, ",") {
		t.Fatalf("launcher args = %v, want %v", gotArgs, wantArgs)
	}
	forceRecreateData, err := os.ReadFile(launcherForceRecreatePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(forceRecreateData)) != "1" {
		t.Fatalf("force recreate environment = %q, want 1", forceRecreateData)
	}
	got, err := browserGetInstance(map[string]string{
		"state":   statePath,
		"agentId": "Agent-A",
		"chatId":  "Chat-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Port != item.Port || got.PID != item.PID {
		t.Fatalf("got = %+v, want port=%d pid=%d", got, item.Port, item.PID)
	}
	if strings.TrimSpace(got.CDP) == "" {
		t.Fatalf("got empty cdp: %+v", got)
	}
}

func TestBrowserInitInstanceOnWSLStaysVisibleUntilShutdown(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	pluginDir := t.TempDir()
	chromePath := filepath.Join(pluginDir, "chrome.exe")
	if err := os.WriteFile(chromePath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	launcherPath := filepath.Join(pluginDir, "browser_launcher.sh")
	launcherScript := `#!/bin/bash
printf '%s\n' '{"status":0,"pid":9003,"port":23092,"ws":"ws://127.0.0.1:23092/devtools/browser/test","http":"http://localhost:23092","user-data-dir":"C:\\ProgramData\\deepright\\chrome_xy99","message":""}'
`
	if err := os.WriteFile(launcherPath, []byte(launcherScript), 0o755); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(pluginDir, "browser_instance.json")

	browserExecutablePathFn = func() (string, error) {
		return filepath.Join(pluginDir, "browser"), nil
	}
	browserWSLDetectFn = func() (bool, error) {
		return true, nil
	}
	browserResolveLauncherScriptPathFn = func(flags map[string]string) (string, error) {
		return launcherPath, nil
	}
	instanceGetFn = func(flags map[string]string) (browserInstanceRecord, error) {
		return browserInstanceRecord{}, fmt.Errorf("instance not found: agentId=%s chatId=%s", flags["agentId"], flags["chatId"])
	}

	live := true
	browserCDPVersionFn = func(port int) (browserCDPVersion, error) {
		if port != 23092 {
			t.Fatalf("unexpected port: %d", port)
		}
		if !live {
			return browserCDPVersion{}, errors.New("cdp closed")
		}
		return browserCDPVersion{WebSocketDebuggerURL: instanceCDPEndpoint(port)}, nil
	}
	browserWSLTerminateProcessFn = func(pid, port int) error {
		if pid != 9003 || port != 23092 {
			t.Fatalf("unexpected shutdown target pid=%d port=%d", pid, port)
		}
		live = false
		return nil
	}
	browserLifecycleProbeInterval = 10 * time.Millisecond
	item, err := browserInitInstance(map[string]string{
		"state":   statePath,
		"chrome":  chromePath,
		"agentId": "Agent-A",
		"chatId":  "Chat-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.Port != 23092 || item.PID != 9003 {
		t.Fatalf("unexpected init item: %+v", item)
	}

	item, err = browserGetInstance(map[string]string{
		"state":   statePath,
		"agentId": "Agent-A",
		"chatId":  "Chat-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.Port != 23092 || item.PID != 9003 {
		t.Fatalf("unexpected get item: %+v", item)
	}

	items, err := browserListInstances(map[string]string{"state": statePath})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].AgentID != "agent-a" || items[0].ChatID != "chat-001" {
		t.Fatalf("unexpected list item: %+v", items[0])
	}

	if err := browserShutdownInstance(map[string]string{
		"state":   statePath,
		"agentId": "Agent-A",
		"chatId":  "Chat-001",
	}); err != nil {
		t.Fatal(err)
	}

	items, err = browserListInstances(map[string]string{"state": statePath})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(items))
	}
	if _, err := browserGetInstance(map[string]string{
		"state":   statePath,
		"agentId": "Agent-A",
		"chatId":  "Chat-001",
	}); err == nil || !strings.Contains(err.Error(), "instance not found") {
		t.Fatalf("err = %v, want instance not found", err)
	}
}

func TestBrowserInitInstanceOnWSLDoesNotRetryDifferentPortsAfterLauncherFailure(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	pluginDir := t.TempDir()
	chromePath := filepath.Join(pluginDir, "chrome.exe")
	if err := os.WriteFile(chromePath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	launcherArgsPath := filepath.Join(pluginDir, "launcher.args")
	_ = launcherArgsPath
	launcherPath := filepath.Join(pluginDir, "browser_launcher.sh")
	launcherScript := "#!/bin/bash\nprintf '{\"status\":1,\"pid\":\"\",\"port\":\"%s\",\"url\":\"\",\"message\":\"failed once\"}\\n' \"$1\"\n"
	if err := os.WriteFile(launcherPath, []byte(launcherScript), 0o755); err != nil {
		t.Fatal(err)
	}

	browserExecutablePathFn = func() (string, error) {
		return filepath.Join(pluginDir, "browser"), nil
	}
	browserWSLDetectFn = func() (bool, error) {
		return true, nil
	}
	browserWSLPathUnixFn = func(path string) (string, error) {
		return filepath.Join(pluginDir, strings.TrimPrefix(path, `C:\ProgramData\deepright\`)), nil
	}
	browserResolveLauncherScriptPathFn = func(flags map[string]string) (string, error) {
		return launcherPath, nil
	}
	instanceGetFn = func(flags map[string]string) (browserInstanceRecord, error) {
		return browserInstanceRecord{}, fmt.Errorf("instance not found: agentId=%s chatId=%s", flags["agentId"], flags["chatId"])
	}
	_, err := browserInitInstance(map[string]string{
		"state":   filepath.Join(pluginDir, "browser_instance.json"),
		"chrome":  chromePath,
		"agentId": "Agent-A",
		"chatId":  "Chat-001",
	})
	if err == nil || !strings.Contains(err.Error(), "failed once") {
		t.Fatalf("err = %v, want launcher failure", err)
	}
}

func TestBrowserPrepareChromeUserDataDirOnWSLCleansLocksBeforeLaunch(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	profileDir := filepath.Join(t.TempDir(), "chrome_58354")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}

	browserWSLDetectFn = func() (bool, error) {
		return true, nil
	}
	cleanedProfiles := []string{}
	browserWSLCleanupChromeUserDataFn = func(profileDir string) error {
		cleanedProfiles = append(cleanedProfiles, profileDir)
		return nil
	}

	if err := browserPrepareChromeUserDataDir(map[string]string{}, profileDir); err != nil {
		t.Fatal(err)
	}
	if len(cleanedProfiles) != 1 || cleanedProfiles[0] != profileDir {
		t.Fatalf("cleaned profiles = %v, want [%q]", cleanedProfiles, profileDir)
	}
}

func TestBrowserPrepareChromeUserDataDirOnWSLCleanupFailureStopsInit(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	profileDir := filepath.Join(t.TempDir(), "chrome_58354")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}

	browserWSLDetectFn = func() (bool, error) {
		return true, nil
	}
	wantErr := errors.New("cleanup failed")
	browserWSLCleanupChromeUserDataFn = func(profileDir string) error {
		return wantErr
	}

	err := browserPrepareChromeUserDataDir(map[string]string{}, profileDir)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}

func TestBrowserIsPortAvailableOnWSLAlsoChecksWindowsHostPort(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	browserWSLDetectFn = func() (bool, error) {
		return true, nil
	}
	browserWSLWindowsPortFreeFn = func(port int) (bool, error) {
		if port != 23456 {
			t.Fatalf("unexpected port: %d", port)
		}
		return false, nil
	}

	if browserIsPortAvailable(23456) {
		t.Fatal("expected port to be treated as unavailable when Windows host reports busy")
	}
}

func TestBrowserGetAndListInstancesIncludeProfileDir(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	pluginDir := t.TempDir()
	statePath := filepath.Join(pluginDir, "browser_instance.json")
	state := []byte("[\n  {\n    \"agentId\": \"agent-a\",\n    \"chatId\": \"chat-001\",\n    \"port\": 23001,\n    \"pid\": 9001,\n    \"cdp\": \"" + instanceCDPEndpoint(23001) + "\",\n    \"lastActiveAt\": \"" + browserFormatActivityTime(time.Now()) + "\"\n  }\n]\n")
	if err := os.WriteFile(statePath, state, 0o644); err != nil {
		t.Fatal(err)
	}

	browserExecutablePathFn = func() (string, error) {
		return filepath.Join(pluginDir, "browser"), nil
	}
	browserProcessExistsFn = func(pid int) bool {
		return pid == 9001
	}
	browserCDPVersionFn = func(port int) (browserCDPVersion, error) {
		return browserCDPVersion{WebSocketDebuggerURL: instanceCDPEndpoint(port)}, nil
	}

	got, err := browserGetInstance(map[string]string{
		"state":   statePath,
		"agentId": "Agent-A",
		"chatId":  "Chat-001",
	})
	if err != nil {
		t.Fatal(err)
	}

	expectedProfileDir := filepath.Join(pluginDir, "agent-a", "chrome_23001")
	if got.ProfileDir != expectedProfileDir {
		t.Fatalf("get ProfileDir = %q, want %q", got.ProfileDir, expectedProfileDir)
	}

	items, err := browserListInstances(map[string]string{"state": statePath})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ProfileDir != expectedProfileDir {
		t.Fatalf("list ProfileDir = %q, want %q", items[0].ProfileDir, expectedProfileDir)
	}
}

func TestBrowserCopyDirectoryWithProgressSkipsCacheStorageAndOptGuideModel(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := filepath.Join(t.TempDir(), "target")

	keepPaths := []string{
		filepath.Join("Default", "WebStorage", "433", "IndexedDB", "state.leveldb"),
		filepath.Join("Default", "IndexedDB", "login.leveldb"),
		filepath.Join("Default", "Local Storage", "leveldb", "000003.log"),
	}
	skipPaths := []string{
		filepath.Join("Default", "Service Worker", "CacheStorage", "cache", "index"),
		filepath.Join("Default", "WebStorage", "433", "CacheStorage", "cache", "index"),
		filepath.Join("OptGuideOnDeviceModel", "2025.8.21.1028", "weights.bin"),
	}

	for _, rel := range append(append([]string{}, keepPaths...), skipPaths...) {
		fullPath := filepath.Join(sourceDir, rel)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(rel), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := browserCopyDirectoryWithProgress(sourceDir, targetDir, nil, nil); err != nil {
		t.Fatal(err)
	}

	for _, rel := range keepPaths {
		if _, err := os.Stat(filepath.Join(targetDir, rel)); err != nil {
			t.Fatalf("expected copied path missing %q: %v", rel, err)
		}
	}
	for _, rel := range skipPaths {
		if _, err := os.Stat(filepath.Join(targetDir, rel)); !os.IsNotExist(err) {
			t.Fatalf("expected skipped path %q to be absent, err=%v", rel, err)
		}
	}
}

func TestBrowserDestroyInstanceRemovesStateAndUserData(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	pluginDir := t.TempDir()
	statePath := filepath.Join(pluginDir, "browser_instance.json")
	profileDir := filepath.Join(pluginDir, "agent-a", "chrome_23001")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, "Preferences"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	state := []byte("[\n  {\n    \"agentId\": \"agent-a\",\n    \"chatId\": \"chat-001\",\n    \"port\": 23001,\n    \"pid\": 9001,\n    \"cdp\": \"" + instanceCDPEndpoint(23001) + "\",\n    \"lastActiveAt\": \"" + browserFormatActivityTime(time.Now()) + "\"\n  }\n]\n")
	if err := os.WriteFile(statePath, state, 0o644); err != nil {
		t.Fatal(err)
	}

	browserExecutablePathFn = func() (string, error) {
		return filepath.Join(pluginDir, "browser"), nil
	}
	browserPortCDPCheckFn = func(port int) (bool, error) {
		if port != 23001 {
			t.Fatalf("unexpected cdp check port: %d", port)
		}
		return true, nil
	}
	browserCDPVersionFn = func(port int) (browserCDPVersion, error) {
		if port != 23001 {
			t.Fatalf("unexpected version probe port: %d", port)
		}
		return browserCDPVersion{WebSocketDebuggerURL: instanceCDPEndpoint(port)}, nil
	}
	terminated := 0
	browserTerminateManagedInstanceFn = func(item browserInstanceRecord) error {
		terminated++
		if item.PID != 9001 || item.Port != 23001 {
			t.Fatalf("unexpected terminated item: %+v", item)
		}
		return nil
	}

	err := browserDestroyInstance(map[string]string{
		"state":             statePath,
		"agentId":           "agent-a",
		"chatId":            "chat-001",
		"cleanup-user-data": "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	if terminated != 1 {
		t.Fatalf("terminated = %d, want 1", terminated)
	}
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		data, _ := os.ReadFile(statePath)
		t.Fatalf("state file still exists: %v, data=%s", err, string(data))
	}
	if _, err := os.Stat(profileDir); !os.IsNotExist(err) {
		t.Fatalf("profile dir still exists: %v", err)
	}
}

func TestBrowserDestroyInstanceWithoutIdentityUsesBootstrapPort(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	statePath := filepath.Join(t.TempDir(), "browser_instance.json")
	terminated := 0
	browserPortPIDLookupFn = func(port int) (int, error) {
		if port != browserWSLBootstrapPort {
			t.Fatalf("unexpected pid lookup port: %d", port)
		}
		return 9011, nil
	}
	browserCDPVersionFn = func(port int) (browserCDPVersion, error) {
		if port != browserWSLBootstrapPort {
			t.Fatalf("unexpected port: %d", port)
		}
		return browserCDPVersion{WebSocketDebuggerURL: instanceCDPEndpoint(port)}, nil
	}
	browserTerminateManagedInstanceFn = func(item browserInstanceRecord) error {
		terminated++
		if item.Port != browserWSLBootstrapPort || item.PID != 9011 {
			t.Fatalf("unexpected terminated bootstrap item: %+v", item)
		}
		return nil
	}

	if err := browserDestroyInstance(map[string]string{"state": statePath}); err != nil {
		t.Fatal(err)
	}
	if terminated != 1 {
		t.Fatalf("terminated = %d, want 1", terminated)
	}
}

func TestBrowserDestroyInstanceCleansMissingStateFallbackCDP(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	statePath := filepath.Join(t.TempDir(), "browser_instance.json")
	instanceGetFn = func(flags map[string]string) (browserInstanceRecord, error) {
		return browserInstanceRecord{}, fmt.Errorf("instance not found: agentId=%s chatId=%s", flags["agentId"], flags["chatId"])
	}
	targetPort := browserHashedPort("agent-a", "chat-001")
	cdpLive := true
	browserCDPVersionFn = func(port int) (browserCDPVersion, error) {
		if port != targetPort {
			t.Fatalf("unexpected port: %d", port)
		}
		if !cdpLive {
			return browserCDPVersion{}, errors.New("cdp closed")
		}
		return browserCDPVersion{WebSocketDebuggerURL: instanceCDPEndpoint(port)}, nil
	}
	lookupCalls := 0
	browserPortPIDLookupFn = func(port int) (int, error) {
		if port != targetPort {
			t.Fatalf("unexpected lookup port: %d", port)
		}
		lookupCalls++
		if lookupCalls == 1 {
			return 9911, nil
		}
		return 0, nil
	}
	terminated := []int{}
	browserTerminateProcessFn = func(pid int) error {
		terminated = append(terminated, pid)
		if pid != 9911 {
			t.Fatalf("unexpected pid: %d", pid)
		}
		cdpLive = false
		return nil
	}

	err := browserDestroyInstance(map[string]string{
		"state":   statePath,
		"agentId": "Agent-A",
		"chatId":  "Chat-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(terminated) != 1 || terminated[0] != 9911 {
		t.Fatalf("terminated = %v, want [9911]", terminated)
	}
}

func TestBrowserDestroyInstanceReturnsNotFoundWhenMissingStateHasNoLiveFallback(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	statePath := filepath.Join(t.TempDir(), "browser_instance.json")
	instanceGetFn = func(flags map[string]string) (browserInstanceRecord, error) {
		return browserInstanceRecord{}, fmt.Errorf("instance not found: agentId=%s chatId=%s", flags["agentId"], flags["chatId"])
	}
	targetPort := browserHashedPort("agent-a", "chat-001")
	browserCDPVersionFn = func(port int) (browserCDPVersion, error) {
		if port != targetPort {
			t.Fatalf("unexpected port: %d", port)
		}
		return browserCDPVersion{}, errors.New("cdp closed")
	}
	browserTerminateProcessFn = func(pid int) error {
		t.Fatalf("terminate should not run for missing fallback pid=%d", pid)
		return nil
	}

	err := browserDestroyInstance(map[string]string{
		"state":   statePath,
		"agentId": "Agent-A",
		"chatId":  "Chat-001",
	})
	if err == nil || !strings.Contains(err.Error(), "instance not found") {
		t.Fatalf("err = %v, want instance not found", err)
	}
}

func TestBrowserDestroyMissingStateFallbackItemsOnWSLOnlyUsesBrowserData(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	browserWSLDetectFn = func() (bool, error) {
		return true, nil
	}
	browserWSLInstanceLookupRecordFn = func(agentID, chatID string) (browserWSLInstanceRecord, bool) {
		if agentID != "agent-a" || chatID != "chat-001" {
			t.Fatalf("unexpected identity %s/%s", agentID, chatID)
		}
		return browserWSLInstanceRecord{
			AgentID:     "agent-a",
			ChatID:      "chat-001",
			PID:         9123,
			Port:        23092,
			WS:          instanceCDPEndpoint(23092),
			UserDataDir: `C:\ProgramData\deepright\chrome_xy99`,
		}, true
	}

	items := browserDestroyMissingStateFallbackItems("Agent-A", "Chat-001")
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Port != 23092 || items[0].PID != 9123 {
		t.Fatalf("unexpected item: %+v", items[0])
	}
	if items[0].ProfileDir != `C:\ProgramData\deepright\chrome_xy99` {
		t.Fatalf("profileDir = %q, want %q", items[0].ProfileDir, `C:\ProgramData\deepright\chrome_xy99`)
	}
}

func TestBrowserTerminateManagedInstanceOnWSLUsesWindowsHostForceKillAndIgnoresLockCleanupFailure(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	browserWSLDetectFn = func() (bool, error) {
		return true, nil
	}
	forceCalls := 0
	browserWSLTerminateProcessFn = func(pid, port int) error {
		forceCalls++
		if pid != 9012 || port != 23012 {
			t.Fatalf("unexpected force kill target pid=%d port=%d", pid, port)
		}
		return nil
	}
	cleanupCalls := 0
	browserWSLCleanupChromeUserDataFn = func(profileDir string) error {
		cleanupCalls++
		if profileDir != `C:\ProgramData\deepright\chrome_ab12` {
			t.Fatalf("unexpected profileDir: %q", profileDir)
		}
		return errors.New("lock cleanup failed")
	}

	err := browserTerminateManagedInstance(browserInstanceRecord{
		AgentID:    "agent-a",
		ChatID:     "chat-001",
		PID:        9012,
		Port:       23012,
		ProfileDir: `C:\ProgramData\deepright\chrome_ab12`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if forceCalls != 1 {
		t.Fatalf("forceCalls = %d, want 1", forceCalls)
	}
	if cleanupCalls != 1 {
		t.Fatalf("cleanupCalls = %d, want 1", cleanupCalls)
	}
}

func TestBrowserCleanupStopUserDataOnWSLDoesNotRemoveManagedChromeDirs(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	root := t.TempDir()
	targetProfileDir := filepath.Join(root, "chrome_29876")
	keepDir := filepath.Join(root, "data")
	if err := os.MkdirAll(targetProfileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(keepDir, 0o755); err != nil {
		t.Fatal(err)
	}
	browserExecutablePathFn = func() (string, error) {
		return filepath.Join(root, "browser"), nil
	}
	if err := os.WriteFile(filepath.Join(root, "browser"), []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	browserWSLDetectFn = func() (bool, error) {
		return true, nil
	}

	if err := browserCleanupStopUserData(nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(targetProfileDir); err != nil {
		t.Fatalf("managed profile dir missing: %v", err)
	}
	if _, err := os.Stat(keepDir); err != nil {
		t.Fatalf("non-profile dir missing: %v", err)
	}
}

func TestBrowserDestroyCleanupUserDataDirOnWSLDoesNotRemoveProfile(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	profileDir := filepath.Join(t.TempDir(), "chrome_ab12")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, "Preferences"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	browserWSLDetectFn = func() (bool, error) {
		return true, nil
	}

	removed, err := browserDestroyCleanupUserDataDir(nil, browserInstanceRecord{
		AgentID:    "agent-a",
		ChatID:     "chat-001",
		Port:       23001,
		ProfileDir: profileDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if removed != "" {
		t.Fatalf("removed = %q, want empty", removed)
	}
	if _, err := os.Stat(profileDir); err != nil {
		t.Fatalf("profile dir missing: %v", err)
	}
}

func TestBrowserCleanupStopUserDataRemovesManagedChromeDirsUnderAgentRoot(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	root := t.TempDir()
	appDir := filepath.Join(root, "integration.app", "Contents")
	macosDir := filepath.Join(appDir, "MacOS")
	resourcesDir := filepath.Join(appDir, "Resources")
	agentRoot := filepath.Join(root, "agent")
	defAgentDir := filepath.Join(agentRoot, "DEF_AGENT")
	targetProfileDir := filepath.Join(defAgentDir, "chrome_53142")
	keepDir := filepath.Join(defAgentDir, "data")

	if err := os.MkdirAll(targetProfileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(keepDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(macosDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(resourcesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetProfileDir, "Preferences"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	homeDir := filepath.Join(root, "home")
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	runtimeDir := filepath.Join(runtimepaths.MacAppRuntimeBaseDir(homeDir, runtimepaths.DeepRightMacBundleIdentifier, runtimepaths.DeepRightAppName), "config")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	connectBin := filepath.Join(macosDir, "integration")
	if err := os.WriteFile(connectBin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	runtimeJSON := fmt.Sprintf("{\"agent-dir\":%q,\"app-dir\":%q}\n", agentRoot, resourcesDir)
	if err := os.WriteFile(filepath.Join(runtimeDir, "config.json"), []byte(runtimeJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	browserWSLDetectFn = func() (bool, error) {
		return false, nil
	}

	exeDir := t.TempDir()
	browserExecutablePathFn = func() (string, error) {
		return writeBrowserBinaryFixture(t, exeDir), nil
	}
	writeBrowserRuntimeRecordFixture(t, connectBin)

	if err := browserCleanupStopUserData(map[string]string{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(targetProfileDir); !os.IsNotExist(err) {
		t.Fatalf("managed profile dir still exists: %v", err)
	}
	if _, err := os.Stat(defAgentDir); err != nil {
		t.Fatalf("agent workspace missing: %v", err)
	}
	if _, err := os.Stat(keepDir); err != nil {
		t.Fatalf("non-profile dir missing: %v", err)
	}
}

func TestBrowserCDPProbeHostsOnlyUsesLocalHosts(t *testing.T) {
	hosts := browserCDPProbeHosts()
	if len(hosts) != 2 || hosts[0] != "127.0.0.1" || hosts[1] != "::1" {
		t.Fatalf("hosts = %+v", hosts)
	}
}
