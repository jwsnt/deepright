package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"connect/browserplaywrightsvc"
)

func stubBrowserRuntime() func() {
	oldPlaywrightRunCLIFn := playwrightRunCLIFn
	oldPlaywrightStartFn := playwrightStartFn
	oldPlaywrightStopFn := playwrightStopFn
	oldBrowserExecutablePathFn := browserExecutablePathFn
	oldBrowserRuntimeGOOSFn := browserRuntimeGOOSFn
	oldBrowserReadFileFn := browserReadFileFn
	oldBrowserWriteFileFn := browserWriteFileFn
	oldBrowserMkdirAllFn := browserMkdirAllFn
	oldBrowserReadDirFn := browserReadDirFn
	oldBrowserRemoveAllFn := browserRemoveAllFn
	oldBrowserLookPathFn := browserLookPathFn
	oldInstanceCreateFn := instanceCreateFn
	oldInstanceInitFn := instanceInitFn
	oldInstanceDestroyFn := instanceDestroyFn
	oldInstanceShutdownFn := instanceShutdownFn
	oldInstanceGetFn := instanceGetFn
	oldInstanceListFn := instanceListFn
	oldInstanceRestartFn := instanceRestartFn
	oldBrowserNowFn := browserNowFn
	oldBrowserResolveSystemChromeUserDataDirFn := browserResolveSystemChromeUserDataDirFn
	oldBrowserStartChromeFn := browserStartChromeFn
	oldBrowserWaitForPortFn := browserWaitForPortFn
	oldBrowserPortCDPCheckFn := browserPortCDPCheckFn
	oldBrowserCDPVersionFn := browserCDPVersionFn
	oldBrowserTerminateProcessFn := browserTerminateProcessFn
	oldBrowserTerminateManagedInstanceFn := browserTerminateManagedInstanceFn
	oldBrowserWaitForCDPShutdownFn := browserWaitForCDPShutdownFn
	oldBrowserCookieCommandContextFn := browserCookieCommandContextFn
	oldBrowserPortAvailableFn := browserPortAvailableFn
	oldBrowserWSLDetectFn := browserWSLDetectFn
	oldBrowserWSLBootstrapLifecycleFn := browserWSLBootstrapLifecycleFn
	oldBrowserWSLAgentRootDirFn := browserWSLAgentRootDirFn
	oldBrowserWSLBaseUserDataDirFn := browserWSLBaseUserDataDirFn
	oldBrowserWSLPathWindowsFn := browserWSLPathWindowsFn
	oldBrowserWSLPathUnixFn := browserWSLPathUnixFn
	oldBrowserWSLProcessExistsFn := browserWSLProcessExistsFn
	oldBrowserWSLWindowsPortFreeFn := browserWSLWindowsPortFreeFn
	oldBrowserWSLTerminateProcessFn := browserWSLTerminateProcessFn
	oldBrowserWSLCommandStartProcessFn := browserWSLCommandStartProcessFn
	oldBrowserWSLCommandWaitForExitFn := browserWSLCommandWaitForExitFn
	oldBrowserWSLCleanupChromeUserDataFn := browserWSLCleanupChromeUserDataFn
	oldBrowserWSLInstanceLookupRecordFn := browserWSLInstanceLookupRecordFn
	oldBrowserAttachedCommandStartProcessFn := browserAttachedCommandStartProcessFn
	oldBrowserAttachedCommandWaitForExitFn := browserAttachedCommandWaitForExitFn
	oldBrowserAllocateCreatePortFn := browserAllocateCreatePortFn
	oldBrowserAllocateInitPortFn := browserAllocateInitPortFn
	oldBrowserAllocatePortFn := browserAllocatePortFn
	oldBrowserRandomIntnFn := browserRandomIntnFn
	oldBrowserResolveLauncherScriptPathFn := browserResolveLauncherScriptPathFn
	oldBrowserLauncherCommandContextFn := browserLauncherCommandContextFn
	oldBrowserLifecycleProbeInterval := browserLifecycleProbeInterval
	oldBrowserLifecycleProbeTimeout := browserLifecycleProbeTimeout
	oldBrowserSleepFn := browserSleepFn
	oldBrowserPrepareWSLChromeDefForStartFn := browserPrepareWSLChromeDefForStartFn
	oldBrowserRestartIntegrationAfterWSLStartFn := browserRestartIntegrationAfterWSLStartFn
	oldBrowserWSLCommandCombinedOutputFn := browserWSLCommandCombinedOutputFn
	oldBrowserWSLIntegrationStartFn := browserWSLIntegrationStartFn
	oldBrowserMaybeCleanupWSLProgramDataChromeDirsFn := browserMaybeCleanupWSLProgramDataChromeDirsFn

	playwrightRunCLIFn = func(args []string, stdout, stderr io.Writer) int {
		switch len(args) {
		case 1:
			switch args[0] {
			case "command":
				_, _ = io.WriteString(stdout, `["goto","eval","attach"]`)
				return 0
			case "param":
				_, _ = io.WriteString(stdout, `["timeout"]`)
				return 0
			}
		}
		return 0
	}
	playwrightStartFn = func(opts browserplaywrightsvc.Options, launchPrefix []string) (*browserplaywrightsvc.StartResult, error) {
		return &browserplaywrightsvc.StartResult{Status: "started", PIDFile: opts.PIDFile, LogFile: opts.LogFile, Addr: "127.0.0.1:18333"}, nil
	}
	playwrightStopFn = func(opts browserplaywrightsvc.Options) (*browserplaywrightsvc.StartResult, error) {
		return &browserplaywrightsvc.StartResult{Status: "stopped", PIDFile: opts.PIDFile, LogFile: opts.LogFile}, nil
	}
	browserExecutablePathFn = os.Executable
	browserRuntimeGOOSFn = func() string { return runtime.GOOS }
	browserReadFileFn = os.ReadFile
	browserWriteFileFn = os.WriteFile
	browserMkdirAllFn = os.MkdirAll
	browserReadDirFn = os.ReadDir
	browserRemoveAllFn = os.RemoveAll
	browserLookPathFn = func(path string) (string, error) {
		return path, nil
	}
	instanceCreateFn = browserCreateInstance
	instanceInitFn = browserInitInstance
	instanceDestroyFn = browserDestroyInstance
	instanceShutdownFn = browserShutdownInstance
	instanceGetFn = browserGetInstance
	instanceListFn = browserListInstances
	instanceRestartFn = browserRestartInstances
	browserNowFn = browserNowFn
	browserResolveSystemChromeUserDataDirFn = browserResolveSystemChromeUserDataDir
	browserStartChromeFn = browserStartChromeProcess
	browserWaitForPortFn = browserWaitForPort
	browserPortCDPCheckFn = browserIsCDPPort
	browserCDPVersionFn = browserFetchCDPVersion
	browserTerminateProcessFn = browserTerminateProcess
	browserTerminateManagedInstanceFn = browserTerminateManagedInstance
	browserWaitForCDPShutdownFn = browserWaitForCDPShutdown
	browserCookieCommandContextFn = exec.CommandContext
	browserPortAvailableFn = browserIsPortAvailable
	browserWSLDetectFn = browserIsWSLSystem
	browserWSLBootstrapLifecycleFn = browserRunWSLBootstrapLifecycle
	browserWSLAgentRootDirFn = browserDefaultWSLAgentRoot
	browserWSLBaseUserDataDirFn = browserDefaultWSLChromeBase
	browserWSLPathWindowsFn = browserWSLPathToWindows
	browserWSLPathUnixFn = browserWSLPathToUnix
	browserWSLProcessExistsFn = browserWSLWindowsProcessExists
	browserWSLWindowsPortFreeFn = browserIsWSLWindowsPortFree
	browserWSLTerminateProcessFn = browserTerminateWSLWindowsProcess
	browserWSLCommandStartProcessFn = browserStartAttachedChromeProcess
	browserWSLCommandWaitForExitFn = browserWaitForChromeProcessExit
	browserWSLCleanupChromeUserDataFn = browserCleanupWSLChromeUserData
	browserWSLInstanceLookupRecordFn = browserWSLInstanceLookupRecord
	browserAttachedCommandStartProcessFn = browserStartAttachedChromeProcessDirect
	browserAttachedCommandWaitForExitFn = browserWaitForChromeProcessExit
	browserAllocateCreatePortFn = browserAllocateCreatePort
	browserAllocateInitPortFn = browserAllocateInitPort
	browserAllocatePortFn = browserAllocatePort
	browserRandomIntnFn = browserRandomIntn
	browserResolveLauncherScriptPathFn = browserResolveLauncherScriptPath
	browserLauncherCommandContextFn = exec.CommandContext
	browserLifecycleProbeInterval = 5 * time.Second
	browserLifecycleProbeTimeout = time.Minute
	browserSleepFn = time.Sleep
	browserPrepareWSLChromeDefForStartFn = browserPrepareWSLChromeDefForStart
	browserRestartIntegrationAfterWSLStartFn = browserRestartIntegrationAfterWSLStart
	browserWSLCommandCombinedOutputFn = browserRunWSLCommandCombinedOutput
	browserWSLIntegrationStartFn = browserRunIntegrationStart
	browserMaybeCleanupWSLProgramDataChromeDirsFn = browserMaybeCleanupWSLProgramDataChromeDirs

	return func() {
		playwrightRunCLIFn = oldPlaywrightRunCLIFn
		playwrightStartFn = oldPlaywrightStartFn
		playwrightStopFn = oldPlaywrightStopFn
		browserExecutablePathFn = oldBrowserExecutablePathFn
		browserRuntimeGOOSFn = oldBrowserRuntimeGOOSFn
		browserReadFileFn = oldBrowserReadFileFn
		browserWriteFileFn = oldBrowserWriteFileFn
		browserMkdirAllFn = oldBrowserMkdirAllFn
		browserReadDirFn = oldBrowserReadDirFn
		browserRemoveAllFn = oldBrowserRemoveAllFn
		browserLookPathFn = oldBrowserLookPathFn
		instanceCreateFn = oldInstanceCreateFn
		instanceInitFn = oldInstanceInitFn
		instanceDestroyFn = oldInstanceDestroyFn
		instanceShutdownFn = oldInstanceShutdownFn
		instanceGetFn = oldInstanceGetFn
		instanceListFn = oldInstanceListFn
		instanceRestartFn = oldInstanceRestartFn
		browserNowFn = oldBrowserNowFn
		browserResolveSystemChromeUserDataDirFn = oldBrowserResolveSystemChromeUserDataDirFn
		browserStartChromeFn = oldBrowserStartChromeFn
		browserWaitForPortFn = oldBrowserWaitForPortFn
		browserPortCDPCheckFn = oldBrowserPortCDPCheckFn
		browserCDPVersionFn = oldBrowserCDPVersionFn
		browserTerminateProcessFn = oldBrowserTerminateProcessFn
		browserTerminateManagedInstanceFn = oldBrowserTerminateManagedInstanceFn
		browserWaitForCDPShutdownFn = oldBrowserWaitForCDPShutdownFn
		browserCookieCommandContextFn = oldBrowserCookieCommandContextFn
		browserPortAvailableFn = oldBrowserPortAvailableFn
		browserWSLDetectFn = oldBrowserWSLDetectFn
		browserWSLBootstrapLifecycleFn = oldBrowserWSLBootstrapLifecycleFn
		browserWSLAgentRootDirFn = oldBrowserWSLAgentRootDirFn
		browserWSLBaseUserDataDirFn = oldBrowserWSLBaseUserDataDirFn
		browserWSLPathWindowsFn = oldBrowserWSLPathWindowsFn
		browserWSLPathUnixFn = oldBrowserWSLPathUnixFn
		browserWSLProcessExistsFn = oldBrowserWSLProcessExistsFn
		browserWSLWindowsPortFreeFn = oldBrowserWSLWindowsPortFreeFn
		browserWSLTerminateProcessFn = oldBrowserWSLTerminateProcessFn
		browserWSLCommandStartProcessFn = oldBrowserWSLCommandStartProcessFn
		browserWSLCommandWaitForExitFn = oldBrowserWSLCommandWaitForExitFn
		browserWSLCleanupChromeUserDataFn = oldBrowserWSLCleanupChromeUserDataFn
		browserWSLInstanceLookupRecordFn = oldBrowserWSLInstanceLookupRecordFn
		browserAttachedCommandStartProcessFn = oldBrowserAttachedCommandStartProcessFn
		browserAttachedCommandWaitForExitFn = oldBrowserAttachedCommandWaitForExitFn
		browserAllocateCreatePortFn = oldBrowserAllocateCreatePortFn
		browserAllocateInitPortFn = oldBrowserAllocateInitPortFn
		browserAllocatePortFn = oldBrowserAllocatePortFn
		browserRandomIntnFn = oldBrowserRandomIntnFn
		browserResolveLauncherScriptPathFn = oldBrowserResolveLauncherScriptPathFn
		browserLauncherCommandContextFn = oldBrowserLauncherCommandContextFn
		browserLifecycleProbeInterval = oldBrowserLifecycleProbeInterval
		browserLifecycleProbeTimeout = oldBrowserLifecycleProbeTimeout
		browserSleepFn = oldBrowserSleepFn
		browserPrepareWSLChromeDefForStartFn = oldBrowserPrepareWSLChromeDefForStartFn
		browserRestartIntegrationAfterWSLStartFn = oldBrowserRestartIntegrationAfterWSLStartFn
		browserWSLCommandCombinedOutputFn = oldBrowserWSLCommandCombinedOutputFn
		browserWSLIntegrationStartFn = oldBrowserWSLIntegrationStartFn
		browserMaybeCleanupWSLProgramDataChromeDirsFn = oldBrowserMaybeCleanupWSLProgramDataChromeDirsFn
	}
}

func TestHelpOmitsLegacyWSLBootstrapDetails(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := runCLI([]string{"help"}, stdout, stderr); code != 0 {
		t.Fatalf("help exit code = %d, stderr = %s", code, stderr.String())
	}

	text := stdout.String()
	for _, wanted := range []string{
		"before restarting the plugin daemon.",
		"./browser instance shutdown --agentId agent-a --chatId chat-001",
		"./browser shutdown --agentId agent-a --chatId chat-001",
		"/mnt/c/Program Files/Google/Chrome/Application/chrome.exe",
		"C:\\ProgramData\\deepright\\chrome_def",
		"browser start best-effort reruns `integration start`",
		"browser stop finishes by concurrently deleting every C:\\ProgramData\\deepright\\chrome* directory",
	} {
		if !strings.Contains(text, wanted) {
			t.Fatalf("help output missing %q\n%s", wanted, text)
		}
	}
	for _, hidden := range []string{
		"localhost:29876",
		"browser start logs the full fixed-port Chrome launch command",
		"browser start also runs one fixed Chrome CDP",
	} {
		if strings.Contains(text, hidden) {
			t.Fatalf("help output should not expose %q\n%s", hidden, text)
		}
	}
}

func TestBrowserRuntimeRootUsesBundledRuntimePluginDir(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	bundleRoot := filepath.Join(t.TempDir(), "DeepRight.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	runtimeDir := filepath.Join(homeDir, "Library", "Application Support", "deepright", "config")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	connectBin := filepath.Join(macOSDir, "integration")
	if err := os.WriteFile(connectBin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	appDir := filepath.Dir(runtimeDir)
	if err := os.WriteFile(filepath.Join(runtimeDir, "config.json"), []byte(fmt.Sprintf("{\"app-dir\":%q}\n", appDir)), 0o644); err != nil {
		t.Fatal(err)
	}

	root, browserPath, err := browserRuntimeRoot(map[string]string{"connect-bin": connectBin})
	if err != nil {
		t.Fatalf("browserRuntimeRoot() error = %v", err)
	}
	wantRoot := filepath.Join(appDir, "plugins")
	if root != wantRoot {
		t.Fatalf("root = %q, want %q", root, wantRoot)
	}
	if browserPath != filepath.Join(wantRoot, "browser") {
		t.Fatalf("browser path = %q, want %q", browserPath, filepath.Join(wantRoot, "browser"))
	}
}

func TestRunCLIRejectsTopLevelInitCommand(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := runCLI([]string{"init"}, stdout, stderr); code == 0 {
		t.Fatal("expected init to fail")
	}
	if !strings.Contains(stderr.String(), "unknown command: init") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunInstanceHelpIncludesInitCommand(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := runInstanceCommand([]string{"help"}, stdout, stderr); code != 0 {
		t.Fatalf("instance help exit code = %d, stderr = %s", code, stderr.String())
	}
	text := stdout.String()
	for _, wanted := range []string{
		"browser instance init --agentId AGENT --chatId CHAT",
		"init always launches headed Chrome, registers the new instance in get/list, and returns once ready",
		"on WSL, create/init call browser_launcher.sh beside the plugin; profileDir still stays in the normal CLI response",
		"C:\\ProgramData\\deepright\\chrome_${suffix}",
	} {
		if !strings.Contains(text, wanted) {
			t.Fatalf("instance help missing %q\n%s", wanted, text)
		}
	}
}

func TestRunCLIRejectsRemovedDestroyCommand(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := runCLI([]string{"destroy"}, stdout, stderr); code == 0 {
		t.Fatal("expected destroy to fail")
	}
	if !strings.Contains(stderr.String(), "browser instance destroy has been renamed to `browser instance shutdown`") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunCLIShutdownInvokesShutdownHandler(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	called := false
	instanceShutdownFn = func(flags map[string]string) error {
		called = true
		if flags["agentId"] != "agent-a" || flags["chatId"] != "chat-1" {
			t.Fatalf("unexpected flags: %+v", flags)
		}
		return nil
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := runCLI([]string{"shutdown", "--agentId", "Agent-A", "--chatId", "Chat-1"}, stdout, stderr); code != 0 {
		t.Fatalf("shutdown exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "OK" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !called {
		t.Fatal("expected shutdown handler to be invoked")
	}
}

func TestRunInstanceCommandStopInvokesShutdownHandler(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	called := false
	instanceShutdownFn = func(flags map[string]string) error {
		called = true
		if flags["agentId"] != "agent-a" || flags["chatId"] != "chat-1" {
			t.Fatalf("unexpected flags: %+v", flags)
		}
		return nil
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := runInstanceCommand([]string{"stop", "--agentId", "Agent-A", "--chatId", "Chat-1"}, stdout, stderr); code != 0 {
		t.Fatalf("stop exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "OK" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !called {
		t.Fatal("expected stop handler to be invoked")
	}
}

func TestRunInstanceCommandInitInvokesInitHandler(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	called := false
	instanceInitFn = func(flags map[string]string) (browserInstanceRecord, error) {
		called = true
		if flags["agentId"] != "agent-a" || flags["chatId"] != "chat-1" {
			t.Fatalf("unexpected flags: %+v", flags)
		}
		return browserInstanceRecord{AgentID: "agent-a", ChatID: "chat-1", Port: 22001, PID: 77, CDP: instanceCDPEndpoint(22001)}, nil
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := runInstanceCommand([]string{"init", "--agentId", "Agent-A", "--chatId", "Chat-1"}, stdout, stderr); code != 0 {
		t.Fatalf("init exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"agentId": "agent-a"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !called {
		t.Fatal("expected init handler to be invoked")
	}
}

func TestRunInstanceCommandRejectsRemovedDestroyCommand(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := runInstanceCommand([]string{"destroy", "--agentId", "agent-a", "--chatId", "chat-1"}, stdout, stderr); code == 0 {
		t.Fatal("expected destroy to fail")
	}
	if !strings.Contains(stderr.String(), "browser instance destroy has been renamed to `browser instance shutdown`") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunInstanceCommandShutdownInvokesDestroyHandler(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	called := false
	instanceDestroyFn = func(flags map[string]string) error {
		called = true
		if flags["agentId"] != "agent-a" || flags["chatId"] != "chat-1" {
			t.Fatalf("unexpected flags: %+v", flags)
		}
		return nil
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := runInstanceCommand([]string{"shutdown", "--agentId", "Agent-A", "--chatId", "Chat-1"}, stdout, stderr); code != 0 {
		t.Fatalf("shutdown exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "OK" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !called {
		t.Fatal("expected shutdown handler to be invoked")
	}
}

func TestRunInstanceCommandShutdownIgnoresMissingInstance(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	called := false
	instanceDestroyFn = func(flags map[string]string) error {
		called = true
		return fmt.Errorf("instance not found: agentId=%s chatId=%s", flags["agentId"], flags["chatId"])
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := runInstanceCommand([]string{"shutdown", "--agentId", "Agent-A", "--chatId", "Chat-1"}, stdout, stderr); code != 0 {
		t.Fatalf("shutdown exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "OK" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !called {
		t.Fatal("expected shutdown handler to be invoked")
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunInstanceCommandShutdownSupportsDefaultBootstrapTarget(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	called := false
	instanceDestroyFn = func(flags map[string]string) error {
		called = true
		if _, ok := flags["agentId"]; ok {
			t.Fatalf("unexpected agentId: %+v", flags)
		}
		if _, ok := flags["chatId"]; ok {
			t.Fatalf("unexpected chatId: %+v", flags)
		}
		return nil
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := runInstanceCommand([]string{"shutdown"}, stdout, stderr); code != 0 {
		t.Fatalf("shutdown exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "OK" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !called {
		t.Fatal("expected default bootstrap shutdown to be invoked")
	}
}

func TestMergedParamsExposeSupportedOverrides(t *testing.T) {
	got := mergedParams()
	want := []map[string]string{
		{"headless": "选填。默认为true，使用无头浏览器静默访问，也可切换为false开启可视化访问"},
		{"chrome": "选填。Chrome浏览器地址，默认使用系统路径"},
	}
	if len(got) != len(want) {
		t.Fatalf("params len = %d, want %d", len(got), len(want))
	}
	for idx := range want {
		if len(got[idx]) != 1 {
			t.Fatalf("params[%d] = %+v", idx, got[idx])
		}
		for key, value := range want[idx] {
			if got[idx][key] != value {
				t.Fatalf("params[%d][%q] = %q, want %q", idx, key, got[idx][key], value)
			}
		}
	}
}

func TestRunCLIParamOutputsFixedDescriptions(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := runCLI([]string{"param"}, stdout, stderr); code != 0 {
		t.Fatalf("param exit code = %d, stderr = %s", code, stderr.String())
	}

	var got []map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode param output: %v\n%s", err, stdout.String())
	}
	want := []map[string]string{
		{"headless": "选填。默认为true，使用无头浏览器静默访问，也可切换为false开启可视化访问"},
		{"chrome": "选填。Chrome浏览器地址，默认使用系统路径"},
	}
	if len(got) != len(want) {
		t.Fatalf("params len = %d, want %d", len(got), len(want))
	}
	for idx := range want {
		for key, value := range want[idx] {
			if got[idx][key] != value {
				t.Fatalf("params[%d][%q] = %q, want %q", idx, key, got[idx][key], value)
			}
		}
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestPluginLifecycleUsesUnifiedRestartFlow(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	exeDir := t.TempDir()
	browserExecutablePathFn = func() (string, error) {
		return filepath.Join(exeDir, "browser"), nil
	}

	restartCalls := 0
	listCalls := 0
	shutdownCalls := 0
	wslCalls := 0
	instanceRestartFn = func(flags map[string]string) error {
		restartCalls++
		return nil
	}
	instanceDestroyFn = func(flags map[string]string) error {
		shutdownCalls++
		if flags["agentId"] == "" || flags["chatId"] == "" {
			t.Fatalf("shutdown flags missing identity: %+v", flags)
		}
		return nil
	}
	browserWSLBootstrapLifecycleFn = func(flags map[string]string) error {
		wslCalls++
		return nil
	}
	instanceListFn = func(flags map[string]string) ([]browserInstanceRecord, error) {
		listCalls++
		return []browserInstanceRecord{{AgentID: "agent-a", ChatID: "chat-1", Port: 22001, PID: 77, CDP: instanceCDPEndpoint(22001)}}, nil
	}

	for _, command := range []string{"start", "stop"} {
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		if code := runCLI([]string{command}, stdout, stderr); code != 0 {
			t.Fatalf("%s exit code = %d, stderr = %s", command, code, stderr.String())
		}
		if strings.TrimSpace(stdout.String()) != "OK" {
			t.Fatalf("%s stdout = %q", command, stdout.String())
		}
	}

	if restartCalls != 1 {
		t.Fatalf("restart calls = %d, want 1", restartCalls)
	}
	if shutdownCalls != 1 {
		t.Fatalf("shutdown calls = %d, want 1", shutdownCalls)
	}
	if wslCalls != 0 {
		t.Fatalf("wsl bootstrap calls = %d, want 0", wslCalls)
	}
}

func TestPluginLifecycleStartOnWSLRefreshSuccessRestartsIntegration(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	exeDir := t.TempDir()
	browserExecutablePathFn = func() (string, error) {
		return filepath.Join(exeDir, "browser"), nil
	}

	refreshCalls := 0
	reopenCalls := 0
	restartCalls := 0
	browserPrepareWSLChromeDefForStartFn = func(flags map[string]string) (bool, error) {
		refreshCalls++
		return true, nil
	}
	browserRestartIntegrationAfterWSLStartFn = func(flags map[string]string) error {
		reopenCalls++
		return nil
	}
	instanceRestartFn = func(flags map[string]string) error {
		restartCalls++
		return nil
	}
	instanceListFn = func(flags map[string]string) ([]browserInstanceRecord, error) {
		return nil, nil
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := runCLI([]string{"start"}, stdout, stderr); code != 0 {
		t.Fatalf("start exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "OK" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if refreshCalls != 1 {
		t.Fatalf("refresh calls = %d, want 1", refreshCalls)
	}
	if restartCalls != 1 {
		t.Fatalf("restart calls = %d, want 1", restartCalls)
	}
	if reopenCalls != 1 {
		t.Fatalf("integration restart calls = %d, want 1", reopenCalls)
	}
}

func TestPluginLifecycleStartOnWSLRefreshFailureDoesNotBlockStart(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	exeDir := t.TempDir()
	browserExecutablePathFn = func() (string, error) {
		return filepath.Join(exeDir, "browser"), nil
	}

	refreshCalls := 0
	reopenCalls := 0
	restartCalls := 0
	browserPrepareWSLChromeDefForStartFn = func(flags map[string]string) (bool, error) {
		refreshCalls++
		return false, errors.New("copy failed")
	}
	browserRestartIntegrationAfterWSLStartFn = func(flags map[string]string) error {
		reopenCalls++
		return nil
	}
	instanceRestartFn = func(flags map[string]string) error {
		restartCalls++
		return nil
	}
	instanceListFn = func(flags map[string]string) ([]browserInstanceRecord, error) {
		return nil, nil
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := runCLI([]string{"start"}, stdout, stderr); code != 0 {
		t.Fatalf("start exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "OK" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if refreshCalls != 1 {
		t.Fatalf("refresh calls = %d, want 1", refreshCalls)
	}
	if restartCalls != 1 {
		t.Fatalf("restart calls = %d, want 1", restartCalls)
	}
	if reopenCalls != 0 {
		t.Fatalf("integration restart calls = %d, want 0", reopenCalls)
	}
}

func TestPluginStopDoesNotBlockOnShutdownErrors(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	exeDir := t.TempDir()
	browserExecutablePathFn = func() (string, error) {
		return filepath.Join(exeDir, "browser"), nil
	}
	instanceListFn = func(flags map[string]string) ([]browserInstanceRecord, error) {
		return []browserInstanceRecord{
			{AgentID: "agent-a", ChatID: "chat-1", Port: 22001, PID: 77, CDP: instanceCDPEndpoint(22001)},
			{AgentID: "agent-b", ChatID: "chat-2", Port: 22002, PID: 88, CDP: instanceCDPEndpoint(22002)},
		}, nil
	}
	shutdownCalls := 0
	instanceDestroyFn = func(flags map[string]string) error {
		shutdownCalls++
		if flags["agentId"] == "agent-a" {
			return os.ErrPermission
		}
		return nil
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := runCLI([]string{"stop"}, stdout, stderr); code != 0 {
		t.Fatalf("stop exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "OK" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if shutdownCalls != 2 {
		t.Fatalf("shutdown calls = %d, want 2", shutdownCalls)
	}
}

func TestPluginStopRunsWSLProgramDataCleanupAfterStopFlow(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	exeDir := t.TempDir()
	browserExecutablePathFn = func() (string, error) {
		return filepath.Join(exeDir, "browser"), nil
	}

	order := []string{}
	playwrightStopFn = func(opts browserplaywrightsvc.Options) (*browserplaywrightsvc.StartResult, error) {
		order = append(order, "playwright-stop")
		return &browserplaywrightsvc.StartResult{Status: "stopped"}, nil
	}
	instanceListFn = func(flags map[string]string) ([]browserInstanceRecord, error) {
		order = append(order, "instance-list")
		return nil, nil
	}
	browserMaybeCleanupWSLProgramDataChromeDirsFn = func() {
		order = append(order, "wsl-programdata-cleanup")
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := runCLI([]string{"stop"}, stdout, stderr); code != 0 {
		t.Fatalf("stop exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "OK" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if len(order) < 3 {
		t.Fatalf("order = %v", order)
	}
	if order[len(order)-1] != "wsl-programdata-cleanup" {
		t.Fatalf("cleanup should run last, order = %v", order)
	}
}
