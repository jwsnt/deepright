package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"connect/browserplaywrightsvc"
	"connect/connectsvc"
	"runtimepaths"
)

const (
	browserKey                            = "browser"
	browserDisplayName                    = "浏览器"
	defaultChromeUserAgent                = browserplaywrightsvc.DefaultChromeUserAgent
	browserIgnoreInvalidRuntimeRecordFlag = "__browser-ignore-invalid-runtime-record"
)

var (
	playwrightRunCLIFn = browserplaywrightsvc.RunCLI
	playwrightStartFn  = browserplaywrightsvc.StartDaemon
	playwrightStopFn   = browserplaywrightsvc.StopDaemon

	browserReadFileFn  = os.ReadFile
	browserWriteFileFn = os.WriteFile
	browserMkdirAllFn  = os.MkdirAll
	browserReadDirFn   = os.ReadDir
	browserRemoveAllFn = os.RemoveAll
	browserLookPathFn  = exec.LookPath

	instanceCreateFn   = browserCreateInstance
	instanceInitFn     = browserInitInstance
	instanceDestroyFn  = browserDestroyInstance
	instanceShutdownFn = browserShutdownInstance
	instanceGetFn      = browserGetInstance
	instanceListFn     = browserListInstances
	instanceRestartFn  = browserRestartInstances
)

func main() {
	configureBrowserPlaywrightRuntime()
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}

func init() {
	configureBrowserPlaywrightRuntime()
}

func runCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || connectsvc.IsHelpCommand(args[0]) {
		printHelp(stdout)
		return 0
	}

	command, rest := normalizeCLIArgs(args)
	switch command {
	case "help":
		printHelp(stdout)
		return 0
	case "name":
		connectsvc.WriteJSON(stdout, map[string]string{"key": browserKey, "name": browserDisplayName})
		return 0
	case "param":
		connectsvc.WriteJSON(stdout, mergedParams())
		return 0
	case "scope":
		connectsvc.WriteJSON(stdout, pluginScopes())
		return 0
	case "command":
		connectsvc.WriteJSON(stdout, mergedCommands())
		return 0
	case "fetch", "store":
		return runCookieCommand(command, rest, stdout, stderr)
	case "start", "stop":
		return runPluginLifecycleCommand(command, rest, stdout, stderr)
	case "shutdown":
		flags, err := connectsvc.ParseFlags(rest)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		if err := browserRejectConnectBinFlag(flags, false); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		if err := browserRejectRemovedFlags(flags); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		normalizedFlags := normalizeSharedInstanceFlags(flags)
		browserLogInstanceShutdownRequest("top_level_shutdown_command", normalizedFlags, nil)
		if err := instanceShutdownFn(normalizedFlags); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		fmt.Fprintln(stdout, "OK")
		return 0
	case "daemon":
		return runDaemonCommand(rest, stdout, stderr)
	case "__daemon":
		return playwrightRunCLIFn(withDefaultBrowserUserAgent(append([]string{"__daemon"}, rest...)), stdout, stderr)
	case browserProfileCleanupCommand:
		return runBrowserProfileCleanupCommand(rest, stderr)
	case "__log-filter":
		flags, err := connectsvc.ParseFlags(rest)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		if err := browserRunLogFilter(os.Stdin, strings.TrimSpace(flags["log-file"])); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		return 0
	case "__wsl-instance":
		return browserWSLInstanceCommand(rest)
	case "__install-playwright-driver":
		return runPlaywrightDriverInstallCommand(rest, stdout, stderr)
	case "instance":
		return runInstanceCommand(rest, stdout, stderr)
	case "init":
		fmt.Fprintln(stderr, "unknown command: init")
		return 1
	case "destroy":
		fmt.Fprintln(stderr, "browser instance destroy has been renamed to `browser instance shutdown`")
		return 1
	case "create", "get", "list", "restart":
		fmt.Fprintf(stderr, "%s is only available as `browser instance %s`\n", command, command)
		return 1
	default:
		flags, _, err := parseCLIArgs(args)
		if err == nil {
			if err := browserRejectConnectBinFlag(flags, false); err != nil {
				fmt.Fprintln(stderr, err.Error())
				return 1
			}
			if err := browserRejectRemovedFlags(flags); err != nil {
				fmt.Fprintln(stderr, err.Error())
				return 1
			}
		}
		refreshInstanceFromSessionArgs(args)
		nextArgs, err := normalizeBrowserCommandArgs(args)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		return playwrightRunCLIFn(withDefaultBrowserUserAgent(nextArgs), stdout, stderr)
	}
}

func configureBrowserPlaywrightRuntime() {
	browserplaywrightsvc.SetManagedInstanceProvider(browserManagedInstanceProvider{})
}

type browserManagedInstanceProvider struct{}

func (browserManagedInstanceProvider) GetManagedInstance(flags map[string]string, agentID, chatID string) (browserplaywrightsvc.ManagedInstanceRecord, error) {
	next := normalizeSharedInstanceFlags(flags)
	next["agentId"] = normalizeBrowserIdentityPart(agentID)
	next["chatId"] = normalizeBrowserIdentityPart(chatID)
	item, err := instanceGetFn(next)
	if err != nil {
		return browserplaywrightsvc.ManagedInstanceRecord{}, err
	}
	return browserplaywrightsvc.ManagedInstanceRecord{
		AgentID: item.AgentID,
		ChatID:  item.ChatID,
		Port:    item.Port,
		PID:     item.PID,
		CDP:     item.CDP,
	}, nil
}

func (browserManagedInstanceProvider) CreateManagedInstance(flags map[string]string, agentID, chatID string) (browserplaywrightsvc.ManagedInstanceRecord, error) {
	next := normalizeSharedInstanceFlags(flags)
	next["agentId"] = normalizeBrowserIdentityPart(agentID)
	next["chatId"] = normalizeBrowserIdentityPart(chatID)
	item, err := instanceCreateFn(next)
	if err != nil {
		return browserplaywrightsvc.ManagedInstanceRecord{}, err
	}
	return browserplaywrightsvc.ManagedInstanceRecord{
		AgentID: item.AgentID,
		ChatID:  item.ChatID,
		Port:    item.Port,
		PID:     item.PID,
		CDP:     item.CDP,
	}, nil
}

func normalizeBrowserIdentityPart(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeBrowserManagedSession(session string) string {
	session = strings.TrimSpace(session)
	if session == "" {
		return ""
	}
	parts := strings.SplitN(session, "@", 2)
	if len(parts) != 2 {
		return strings.ToLower(session)
	}
	agentID := normalizeBrowserIdentityPart(parts[0])
	chatID := normalizeBrowserIdentityPart(parts[1])
	if agentID == "" || chatID == "" {
		return strings.ToLower(session)
	}
	return browserInstanceSessionName(agentID, chatID)
}

func normalizeBrowserIdentityFlags(flags map[string]string) map[string]string {
	next := cloneFlags(flags)
	for _, key := range []string{"agentId", "agent", "chatId", "chat"} {
		if value, ok := next[key]; ok {
			next[key] = normalizeBrowserIdentityPart(value)
		}
	}
	if value, ok := next["session"]; ok {
		next["session"] = normalizeBrowserManagedSession(value)
	}
	return next
}

func runInstanceCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || connectsvc.IsHelpCommand(args[0]) {
		printInstanceHelp(stdout)
		return 0
	}

	command := strings.TrimSpace(args[0])
	flags, err := connectsvc.ParseFlags(args[1:])
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if err := browserRejectConnectBinFlag(flags, false); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if err := browserRejectRemovedFlags(flags); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	switch command {
	case "create":
		item, err := instanceCreateFn(normalizeSharedInstanceFlags(flags))
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		connectsvc.WriteJSON(stdout, item)
		return 0
	case "init":
		item, err := instanceInitFn(normalizeSharedInstanceFlags(flags))
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		connectsvc.WriteJSON(stdout, item)
		return 0
	case "restart":
		if err := instanceShutdownFn(normalizeSharedInstanceFlags(flags)); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		item, err := instanceCreateFn(normalizeSharedInstanceFlags(flags))
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		connectsvc.WriteJSON(stdout, item)
		return 0
	case "stop":
		normalizedFlags := normalizeSharedInstanceFlags(flags)
		browserLogInstanceShutdownRequest("instance_stop_command", normalizedFlags, nil)
		if err := instanceShutdownFn(normalizedFlags); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		fmt.Fprintln(stdout, "OK")
		return 0
	case "shutdown":
		normalizedFlags := normalizeSharedInstanceFlags(flags)
		browserLogInstanceShutdownRequest("instance_shutdown_command", normalizedFlags, nil)
		if err := browserInvokeInstanceShutdown(normalizedFlags); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		fmt.Fprintln(stdout, "OK")
		return 0
	case "destroy":
		fmt.Fprintln(stderr, "browser instance destroy has been renamed to `browser instance shutdown`")
		return 1
	case "list":
		items, err := instanceListFn(normalizeSharedInstanceFlags(flags))
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		connectsvc.WriteJSON(stdout, items)
		return 0
	case "get":
		item, err := instanceGetFn(normalizeSharedInstanceFlags(flags))
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		connectsvc.WriteJSON(stdout, item)
		return 0
	case "help":
		printInstanceHelp(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown instance command: %s\n", command)
		printInstanceHelp(stderr)
		return 1
	}
}

func runPluginLifecycleCommand(command string, args []string, stdout, stderr io.Writer) int {
	flags, positionals, err := parseCLIArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if err := browserRejectConnectBinFlag(flags, command == "start"); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if len(positionals) > 0 {
		fmt.Fprintf(stderr, "%s does not accept positional arguments\n", command)
		return 1
	}
	if err := browserRejectRemovedFlags(flags); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if command == "start" {
		flags = cloneFlags(flags)
		flags[browserIgnoreInvalidRuntimeRecordFlag] = "true"
	}
	if err := browserEnsurePluginLogFile(); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	pluginOpts, err := browserPluginDaemonOptions(flags)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if command == "start" {
		browserLogPlaywrightDriverPreflightEvent(browserEnsurePlaywrightDriverForStartFn(flags, pluginOpts))
		if err := browserValidateStartCookieSupport(flags); err != nil {
			browserLogCookiePreflightEvent(flags, "start", err)
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		browserLogCookiePreflightEvent(flags, "start", nil)
	}
	normalizedFlags := normalizeSharedInstanceFlags(flags)
	beforeItems, err := instanceListFn(normalizedFlags)
	if err != nil {
		if command == "stop" {
			browserLogAsyncLifecycleEvent("browser_stop_instances", "before_list_error", nil, 0, err)
			beforeItems = nil
		} else {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
	}
	browserLogInstanceListEvent(command, "before", beforeItems)
	switch command {
	case "start":
		if err := instanceRestartFn(normalizedFlags); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
	}
	switch command {
	case "start":
		result, err := playwrightStartFn(pluginOpts, nil)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		if connectBin := strings.TrimSpace(flags["connect-bin"]); connectBin != "" {
			if err := browserWriteRecordedConnectBin(connectBin); err != nil {
				fmt.Fprintln(stderr, err.Error())
				return 1
			}
		}
		if err := browserProfileCleanupStartFn(); err != nil {
			browserLogAsyncLifecycleEvent("browser_profile_cleanup", "worker_start_error", nil, 0, err)
		}
		browserLogPluginDaemonEvent(command, result)
	case "stop":
		if err := browserProfileCleanupStopFn(); err != nil {
			browserLogAsyncLifecycleEvent("browser_profile_cleanup", "worker_stop_error", nil, 0, err)
		}
		result, err := playwrightStopFn(pluginOpts)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		browserLogPluginDaemonEvent(command, result)
		browserGracefulStopPluginInstances(normalizedFlags)
		browserMaybeCleanupStopUserData(flags)
		browserMaybeShutdownDefaultBootstrapCDP(flags)
	}
	afterItems, err := instanceListFn(normalizedFlags)
	if err != nil {
		if command == "stop" {
			browserLogAsyncLifecycleEvent("browser_stop_instances", "after_list_error", nil, 0, err)
			afterItems = nil
		} else {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
	}
	browserLogInstanceListEvent(command, "after", afterItems)
	if command == "stop" {
		browserMaybeRemoveRecordedConnectBin()
	}
	fmt.Fprintln(stdout, "OK")
	return 0
}

func runDaemonCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || connectsvc.IsHelpCommand(args[0]) {
		printDaemonHelp(stdout)
		return 0
	}

	command := strings.TrimSpace(args[0])
	switch command {
	case "help":
		printDaemonHelp(stdout)
		return 0
	case "start", "stop", "serve":
		flags, err := connectsvc.ParseFlags(args[1:])
		if err == nil {
			if err := browserRejectConnectBinFlag(flags, false); err != nil {
				fmt.Fprintln(stderr, err.Error())
				return 1
			}
			if command != "stop" {
				if err := browserRejectRemovedFlags(flags); err != nil {
					fmt.Fprintln(stderr, err.Error())
					return 1
				}
			}
		}
		nextArgs, err := normalizeBrowserDaemonCLIArgs(command, args[1:])
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		return playwrightRunCLIFn(withDefaultBrowserUserAgent(nextArgs), stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown daemon command: %s\n", command)
		printDaemonHelp(stderr)
		return 1
	}
}

func browserPluginDaemonOptions(flags map[string]string) (browserplaywrightsvc.Options, error) {
	root, browserPath, err := browserRuntimeRoot(flags)
	if err != nil {
		return browserplaywrightsvc.Options{}, err
	}

	opts, err := browserPlaywrightOptionsFromFlags(flags)
	if err != nil {
		return browserplaywrightsvc.Options{}, err
	}
	if strings.TrimSpace(opts.StateDir) == "" {
		opts.StateDir = filepath.Join(root, ".browser_playwright")
	}
	if strings.TrimSpace(opts.PIDFile) == "" {
		opts.PIDFile = filepath.Join(root, "browser.pid")
	}
	if strings.TrimSpace(opts.LogFile) == "" {
		opts.LogFile = filepath.Join(root, filepath.Base(browserPath)+".log")
	}
	if strings.TrimSpace(opts.DriverDir) == "" {
		opts.DriverDir = filepath.Join(root, "playwright", "driver")
	}
	opts.ExecutablePath = browserPath
	return opts, nil
}

func browserPlaywrightOptionsFromFlags(flags map[string]string) (browserplaywrightsvc.Options, error) {
	timeout, err := browserPlaywrightTimeoutFromFlags(flags)
	if err != nil {
		return browserplaywrightsvc.Options{}, err
	}
	return browserplaywrightsvc.Options{
		StateDir:       strings.TrimSpace(flags["state-dir"]),
		Addr:           strings.TrimSpace(flags["addr"]),
		LogFile:        strings.TrimSpace(flags["log-file"]),
		PIDFile:        strings.TrimSpace(flags["pid-file"]),
		DriverDir:      strings.TrimSpace(flags["driver-dir"]),
		BrowserTimeout: timeout,
		BrowserRetry:   browserRetryFromFlags(flags),
	}, nil
}

func browserRejectRemovedFlags(flags map[string]string) error {
	for _, key := range []string{
		"browser_cookie_cache",
		"monitor",
		"instance-monitor",
		"monitor-ms",
		"instance-monitor-ms",
		"monitor-state",
		"monitor-bin",
	} {
		if _, ok := flags[key]; ok {
			switch key {
			case "browser_cookie_cache":
				return fmt.Errorf("%s is no longer supported: browser cookie functionality has been removed", key)
			default:
				return fmt.Errorf("%s is no longer supported: browser instance monitor has been removed", key)
			}
		}
	}
	return nil
}

func browserRetryFromFlags(flags map[string]string) int {
	value, ok := browserIntFlagValue(flags, "browser_retry")
	if !ok || value <= 0 {
		return 0
	}
	return value
}

func browserRejectConnectBinFlag(flags map[string]string, allowed bool) error {
	if allowed {
		return nil
	}
	if _, ok := flags["connect-bin"]; ok {
		return fmt.Errorf("--connect-bin is only supported by `browser start`")
	}
	return nil
}

func browserIntFlagValue(flags map[string]string, key string) (int, bool) {
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

func browserRuntimeRoot(flags map[string]string) (string, string, error) {
	browserPath, err := browserExecutablePathFn()
	if err != nil {
		return "", "", err
	}
	if resolved, err := filepath.EvalSymlinks(browserPath); err == nil {
		browserPath = resolved
	}
	if abs, err := filepath.Abs(browserPath); err == nil {
		browserPath = abs
	}
	if root, ok, err := browserIntegrationPluginsRoot(flags); err != nil {
		return "", "", err
	} else if ok {
		return root, filepath.Join(root, "browser"), nil
	}
	root := filepath.Dir(browserPath)
	if strings.TrimSpace(root) == "" {
		return "", "", fmt.Errorf("resolve browser executable directory")
	}
	return root, browserPath, nil
}

func browserIntegrationPluginsRoot(flags map[string]string) (string, bool, error) {
	if connectBin := strings.TrimSpace(flags["connect-bin"]); connectBin != "" {
		connectBin, err := browserEnsureExecutablePath(connectBin)
		if err != nil {
			return "", false, err
		}
		return browserIntegrationPluginsRootFromConnectBin(connectBin)
	}
	connectBin, ok, err := browserReadRecordedConnectBin()
	if err != nil {
		if browserIgnoresInvalidRuntimeRecord(flags) {
			return "", false, nil
		}
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}
	root, ok, err := browserIntegrationPluginsRootFromConnectBin(connectBin)
	if err != nil && browserIgnoresInvalidRuntimeRecord(flags) {
		return "", false, nil
	}
	return root, ok, err
}

func browserIgnoresInvalidRuntimeRecord(flags map[string]string) bool {
	return strings.EqualFold(strings.TrimSpace(flags[browserIgnoreInvalidRuntimeRecordFlag]), "true")
}

func browserIntegrationPluginsRootFromConnectBin(connectBin string) (string, bool, error) {
	runtimePath, ok := browserResolveIntegrationRuntimePath(connectBin)
	if !ok {
		return "", false, fmt.Errorf("resolve runtime config from %s", browserRuntimeFileName)
	}
	cfg, err := browserReadRuntimeConfig(runtimePath)
	if err != nil {
		return "", false, err
	}
	appDir, err := browserResolveIntegrationAppDir(runtimePath, cfg)
	if err != nil {
		return "", false, err
	}
	if appDir == "" {
		return "", false, fmt.Errorf("resolve integration app directory from %s", runtimePath)
	}
	return filepath.Join(appDir, "plugins"), true, nil
}

func browserResolveIntegrationAppDir(runtimePath string, _ map[string]string) (string, error) {
	homeDir, homeErr := browserUserHomeDirFn()
	switch {
	case browserRuntimeGOOSFn() == "darwin":
		if homeErr != nil {
			return "", fmt.Errorf("resolve macOS Browser runtime directory: %w", homeErr)
		}
		if strings.TrimSpace(homeDir) == "" {
			return "", fmt.Errorf("resolve macOS Browser runtime directory: home directory is empty")
		}
		return runtimepaths.MacAppRuntimeBaseDir(homeDir, runtimepaths.DeepRightMacBundleIdentifier, runtimepaths.DeepRightAppName), nil
	case browserRuntimeGOOSFn() == "linux":
		isWSL, err := browserWSLDetectFn()
		if err != nil {
			return "", err
		}
		if isWSL {
			if homeErr != nil {
				return "", fmt.Errorf("resolve WSL Browser runtime directory: %w", homeErr)
			}
			if strings.TrimSpace(homeDir) == "" {
				return "", fmt.Errorf("resolve WSL Browser runtime directory: home directory is empty")
			}
			return filepath.Join(homeDir, "deepright"), nil
		}
	}

	// Directory-style releases keep config/config.json beside the application.
	// This fallback does not apply on macOS or WSL, whose runtime locations are
	// fixed and intentionally separate from the signed/static configuration.
	runtimeDir := filepath.Dir(strings.TrimSpace(runtimePath))
	if filepath.Base(runtimeDir) == "config" {
		runtimeDir = filepath.Dir(runtimeDir)
	}
	if runtimeDir == "" || runtimeDir == "." {
		return "", nil
	}
	if abs, err := filepath.Abs(runtimeDir); err == nil {
		return abs, nil
	}
	return filepath.Clean(runtimeDir), nil
}

func browserResolveBundledIntegrationRuntimePath(connectBin string) string {
	return connectsvc.ResolveBundledRuntimeConfigPath(connectBin)
}

func browserResolveIntegrationRuntimePath(connectBin string) (string, bool) {
	return connectsvc.ResolveRuntimeConfigPathFromConnectBin(connectBin)
}

func browserReadRuntimeConfig(path string) (map[string]string, error) {
	return connectsvc.ReadRuntimeConfig(path)
}

func normalizeBrowserDaemonCLIArgs(command string, args []string) ([]string, error) {
	flags, positionals, err := parseCLIArgs(args)
	if err != nil {
		return nil, err
	}
	if len(positionals) > 0 {
		return nil, fmt.Errorf("%s does not accept positional arguments", command)
	}
	opts, err := browserPluginDaemonOptions(flags)
	if err != nil {
		return nil, err
	}
	next := cloneFlags(flags)
	if strings.TrimSpace(next["state-dir"]) == "" {
		next["state-dir"] = opts.StateDir
	}
	if strings.TrimSpace(next["pid-file"]) == "" {
		next["pid-file"] = opts.PIDFile
	}
	if strings.TrimSpace(next["log-file"]) == "" {
		next["log-file"] = opts.LogFile
	}
	return buildBrowserCLIArgs(command, next, nil), nil
}

func normalizeBrowserCommandArgs(args []string) ([]string, error) {
	flags, _, err := parseCLIArgs(args)
	if err != nil {
		return nil, err
	}
	command, rest := normalizeCLIArgs(args)
	if strings.TrimSpace(command) == "" {
		return append([]string{}, args...), nil
	}
	_, positionals, err := parseCLIArgs(rest)
	if err != nil {
		return nil, err
	}
	positionals, flags = normalizeEvalCodeArgs(command, positionals, flags)
	opts, err := browserPluginDaemonOptions(flags)
	if err != nil {
		return nil, err
	}
	next := cloneFlags(flags)
	if strings.TrimSpace(next["state-dir"]) == "" {
		next["state-dir"] = opts.StateDir
	}
	if strings.TrimSpace(next["pid-file"]) == "" {
		next["pid-file"] = opts.PIDFile
	}
	if strings.TrimSpace(next["log-file"]) == "" {
		next["log-file"] = opts.LogFile
	}
	return withDefaultBrowserUserAgent(buildBrowserCLIArgs(command, next, positionals)), nil
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
	nextFlags := cloneFlags(flags)
	delete(nextFlags, "code")
	return []string{value}, nextFlags
}

func buildBrowserCLIArgs(command string, flags map[string]string, positionals []string) []string {
	keys := make([]string, 0, len(flags))
	for key := range flags {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]string, 0, len(keys)*2+1+len(positionals))
	for _, key := range keys {
		out = appendFlagArgs(out, key, flags[key], browserFlagTakesValue)
	}
	out = append(out, command)
	out = append(out, positionals...)
	return out
}

func buildBrowserInstanceCLIArgs(command string, flags map[string]string) []string {
	keys := make([]string, 0, len(flags))
	for key := range flags {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := []string{"instance", command}
	for _, key := range keys {
		out = appendFlagArgs(out, key, flags[key], browserFlagTakesValue)
	}
	return out
}

func browserPlaywrightTimeoutFromFlags(flags map[string]string) (time.Duration, error) {
	raw := strings.TrimSpace(flags["browser-timeout"])
	if raw == "" {
		return 0, nil
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

func buildAttachArgs(flags map[string]string, positionals []string, item browserInstanceRecord) []string {
	next := normalizeBrowserIdentityFlags(flags)
	cdp := strings.TrimSpace(item.CDP)
	if cdp == "" {
		cdp = instanceCDPEndpoint(item.Port)
	}
	next["cdp"] = cdp
	next["session"] = browserInstanceSessionName(item.AgentID, item.ChatID)

	keys := make([]string, 0, len(next))
	for key := range next {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	args := make([]string, 0, len(keys)*2+2+len(positionals))
	for _, key := range keys {
		args = appendFlagArgs(args, key, next[key], browserFlagTakesValue)
	}
	args = append(args, "attach")
	args = append(args, positionals...)
	return args
}

func withDefaultBrowserUserAgent(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	hasUserAgent := false
	next := make([]string, 0, len(args)+2)
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		next = append(next, args[i])
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		key := normalizeBrowserCLIFlagKey(strings.TrimPrefix(arg, "--"))
		if parts := strings.SplitN(key, "=", 2); len(parts) == 2 {
			key = normalizeBrowserCLIFlagKey(parts[0])
		} else if browserFlagTakesValue(key) && i+1 < len(args) && !strings.HasPrefix(strings.TrimSpace(args[i+1]), "--") {
			i++
			next = append(next, args[i])
		}
		if key == "user-agent" {
			hasUserAgent = true
		}
	}
	if hasUserAgent {
		return next
	}
	return append([]string{"--user-agent", defaultChromeUserAgent}, next...)
}

func appendFlagArgs(out []string, key, value string, takesValue func(string) bool) []string {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if key == "" {
		return out
	}
	out = append(out, "--"+key)
	if takesValue != nil && !takesValue(key) && strings.EqualFold(value, "true") {
		return out
	}
	return append(out, value)
}

func normalizeSharedInstanceFlags(flags map[string]string) map[string]string {
	next := normalizeBrowserIdentityFlags(flags)
	copyAliasValue(next, "state", flags, "state", "instance-state")
	copyAliasValue(next, "chrome", flags, "chrome", "instance-chrome", "obscura", "instance-obscura")
	copyAliasValue(next, "browser_expired", flags, "browser_expired", "instance-browser_expired")
	copyAliasValue(next, "headless", flags, "headless", "instance-headless")
	return browserApplyMetaChromeFlag(next)
}

func browserApplyMetaChromeFlag(flags map[string]string) map[string]string {
	next := cloneFlags(flags)
	if path, ok := browserLookupChromePathFromPluginMeta(next); ok {
		next["chrome"] = path
	}
	return next
}

func browserLookupChromePathFromPluginMeta(flags map[string]string) (string, bool) {
	response, ok, err := browserLookupPluginMeta(flags, browserKey, true)
	if err != nil || !ok {
		return "", false
	}
	value, _ := response.Meta["chrome"].(string)
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	return value, true
}

func copyAliasValue(target map[string]string, targetKey string, source map[string]string, keys ...string) {
	for _, key := range keys {
		if value, ok := source[key]; ok {
			target[targetKey] = value
			return
		}
	}
}

func mergedCommands() []string {
	seen := map[string]struct{}{
		"command":  {},
		"daemon":   {},
		"fetch":    {},
		"help":     {},
		"name":     {},
		"param":    {},
		"scope":    {},
		"shutdown": {},
		"start":    {},
		"store":    {},
		"stop":     {},
		"instance": {},
	}
	items := []string{"command", "daemon", "fetch", "help", "instance", "name", "param", "scope", "shutdown", "start", "stop", "store"}

	for _, item := range playwrightCommands() {
		if !browserCommandVisible(item) {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		items = append(items, item)
	}
	sort.Strings(items)
	return items
}

func browserCommandVisible(command string) bool {
	switch strings.TrimSpace(command) {
	case "", "create", "destroy", "get", "list", "restart", "serve", "shutdown", "__daemon", "__log-filter":
		return false
	default:
		return true
	}
}

func mergedParams() []map[string]string {
	return []map[string]string{
		{"headless": "选填。默认为true，使用无头浏览器静默访问，也可切换为false开启可视化访问"},
		{"chrome": "选填。Chrome浏览器地址，默认使用系统路径"},
	}
}

func pluginScopes() []string {
	return []string{}
}

func playwrightCommands() []string {
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	if playwrightRunCLIFn([]string{"command"}, stdout, stderr) != 0 {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(stdout.String()), &items); err != nil {
		return nil
	}
	return items
}

func playwrightParams() []string {
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	if playwrightRunCLIFn([]string{"param"}, stdout, stderr) != 0 {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(stdout.String()), &items); err != nil {
		return nil
	}
	return items
}

func parseCLIArgs(args []string) (map[string]string, []string, error) {
	flags := map[string]string{}
	positionals := make([]string, 0)
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "" {
			continue
		}
		if !strings.HasPrefix(arg, "--") {
			positionals = append(positionals, arg)
			continue
		}

		key := normalizeBrowserCLIFlagKey(strings.TrimPrefix(arg, "--"))
		value := "true"
		if parts := strings.SplitN(key, "=", 2); len(parts) == 2 {
			key = normalizeBrowserCLIFlagKey(parts[0])
			value = parts[1]
		} else if browserFlagTakesValue(key) && i+1 < len(args) && !strings.HasPrefix(strings.TrimSpace(args[i+1]), "--") {
			value = args[i+1]
			i++
		}
		flags[key] = value
	}
	return normalizeBrowserIdentityFlags(flags), positionals, nil
}

func normalizeCLIArgs(args []string) (string, []string) {
	if len(args) == 0 {
		return "help", nil
	}

	rest := append([]string{}, args...)
	commandIndex := -1
	for i := 0; i < len(rest); i++ {
		arg := strings.TrimSpace(rest[i])
		if arg == "" {
			continue
		}
		if strings.HasPrefix(arg, "--") {
			key := normalizeBrowserCLIFlagKey(strings.TrimPrefix(arg, "--"))
			if parts := strings.SplitN(key, "=", 2); len(parts) == 2 {
				continue
			}
			if browserFlagTakesValue(key) && i+1 < len(rest) && !strings.HasPrefix(strings.TrimSpace(rest[i+1]), "--") {
				i++
			}
			continue
		}
		commandIndex = i
		break
	}
	if commandIndex == -1 {
		return "help", rest
	}
	command := strings.TrimSpace(rest[commandIndex])
	rest = append(rest[:commandIndex], rest[commandIndex+1:]...)
	return command, rest
}

func cloneFlags(flags map[string]string) map[string]string {
	if len(flags) == 0 {
		return map[string]string{}
	}
	next := make(map[string]string, len(flags))
	for key, value := range flags {
		next[key] = value
	}
	return next
}

func browserInstanceSessionName(agentID, chatID string) string {
	return normalizeBrowserIdentityPart(agentID) + "@" + normalizeBrowserIdentityPart(chatID)
}

func refreshInstanceFromSessionArgs(args []string) {
	flags, _, err := parseCLIArgs(args)
	if err != nil {
		return
	}
	session := strings.TrimSpace(flags["session"])
	agentID, chatID, ok := splitManagedSession(session)
	if !ok {
		return
	}
	refreshManagedInstance(agentID, chatID, normalizeSharedInstanceFlags(flags))
}

func refreshManagedInstance(agentID, chatID string, flags map[string]string) {
	agentID = normalizeBrowserIdentityPart(agentID)
	chatID = normalizeBrowserIdentityPart(chatID)
	if agentID == "" || chatID == "" {
		return
	}
	next := normalizeSharedInstanceFlags(flags)
	next["agentId"] = agentID
	next["chatId"] = chatID
	_, _ = instanceGetFn(next)
}

func splitManagedSession(session string) (string, string, bool) {
	session = normalizeBrowserManagedSession(session)
	if session == "" {
		return "", "", false
	}
	parts := strings.SplitN(session, "@", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	agentID := strings.TrimSpace(parts[0])
	chatID := strings.TrimSpace(parts[1])
	if agentID == "" || chatID == "" {
		return "", "", false
	}
	return agentID, chatID, true
}

func browserFlagTakesValue(key string) bool {
	switch strings.TrimSpace(key) {
	case "session", "state-dir", "addr", "browser", "cdp", "profile", "width", "height", "filename", "path", "selector", "timeout", "navigation-timeout", "channel", "pid-file", "log-file", "user-agent", "agentId", "chatId", "agent", "chat", "connect-bin", "instance-bin", "instance-state", "instance-chrome", "instance-obscura", "instance-browser_expired", "instance-headless", "browser-timeout", "browser_retry", "driver-dir", "state", "chrome", "obscura", "browser_expired", "headless", "cookie_path", "code":
		return true
	default:
		return false
	}
}

func normalizeBrowserCLIFlagKey(key string) string {
	switch strings.TrimSpace(key) {
	case "path":
		return "filename"
	default:
		return strings.TrimSpace(key)
	}
}

func intFlag(flags map[string]string, key string, fallback int) (int, error) {
	return connectsvc.IntValue(flags, key, fallback)
}

func browserFlagEnabled(flags map[string]string, key string) bool {
	return connectsvc.BoolValue(flags, key, false)
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  browser help")
	fmt.Fprintln(w, "  browser fetch [--cookie_path PATH]")
	fmt.Fprintln(w, "  browser store [--cookie_path PATH]")
	fmt.Fprintln(w, "  browser --session AGENT@CHAT goto <url>")
	fmt.Fprintln(w, "  browser --session AGENT@CHAT eval <javascript>")
	fmt.Fprintln(w, "  browser --session AGENT@CHAT eval --code <javascript>")
	fmt.Fprintln(w, "  browser daemon <command> [flags]")
	fmt.Fprintln(w, "  browser shutdown --agentId AGENT --chatId CHAT")
	fmt.Fprintln(w, "  browser instance <create|restart|shutdown|list|get> [flags]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Overview:")
	fmt.Fprintln(w, "  browser is the browser plugin binary and the single end-user entry for browser automation,")
	fmt.Fprintln(w, "  and CDP instance lifecycle. All browser_playwright commands remain")
	fmt.Fprintln(w, "  available directly on browser, while plugin lifecycle reset now restarts managed CDP state")
	fmt.Fprintln(w, "  before restarting the plugin daemon.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Core:")
	fmt.Fprintln(w, "  fetch     validate and print the configured browser cookie file")
	fmt.Fprintln(w, "  daemon    manage the underlying browser_playwright daemon lifecycle")
	fmt.Fprintln(w, "  instance  manage raw CDP instances directly")
	fmt.Fprintln(w, "  store     validate or initialize the configured browser cookie file")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Chrome CDP Runtime:")
	fmt.Fprintln(w, "  - browser resolves the local Chrome executable automatically, or you can override it with --chrome")
	fmt.Fprintln(w, "  - browser instance create starts or reuses one Chrome-backed CDP endpoint")
	fmt.Fprintln(w, "  - browser instance lifecycle commands manage one AgentId + ChatId Chrome instance directly; on WSL, all agents inside the same chat share one managed instance")
	fmt.Fprintln(w, "  - managed instance state is persisted in ./browser_instance.json beside the browser binary")
	fmt.Fprintln(w, "  - instance create prepares one managed --user-data-dir before launch")
	fmt.Fprintln(w, "    using <agent workspace>/chrome_${port} on macOS/Linux and C:\\ProgramData\\deepright\\profiles\\chats\\<chatId> on WSL")
	fmt.Fprintln(w, "  - on WSL, browser resolves Chrome from browser meta chrome first, then falls back to /mnt/c/Program Files/Google/Chrome/Application/chrome.exe")
	fmt.Fprintln(w, "  - on WSL, instance create uses browser_launcher.sh beside the plugin for browser_instance_wsl launch/reuse logic; if the packaged plugin is missing that script, browser recreates it automatically before launch")
	fmt.Fprintln(w, "  - on WSL, browser_instance_wsl reuses one persistent Chat-scoped profile under C:\\ProgramData\\deepright\\profiles\\chats\\<chatId>; it does not copy system Chrome data or clean profile locks")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Shared Flags:")
	fmt.Fprintln(w, "  --session NAME             Playwright session name for direct commands")
	fmt.Fprintln(w, "  --agentId AGENT            managed instance identity")
	fmt.Fprintln(w, "  --chatId CHAT              managed instance identity")
	fmt.Fprintln(w, "  --timeout MS               Playwright action timeout in milliseconds, default 5000")
	fmt.Fprintln(w, "  --navigation-timeout MS    Playwright navigation timeout in milliseconds, default 60000; mainly for open/goto/reload")
	fmt.Fprintln(w, "  --browser-timeout VALUE    Playwright wall-clock timeout, default 120s")
	fmt.Fprintln(w, "  --browser_retry N          consecutive timeout count before Playwright daemon is killed, default 3")
	fmt.Fprintln(w, "  --cookie_path PATH         browser cookie file path; can also come from plugin meta")
	fmt.Fprintln(w, "  --state-dir DIR            Playwright state root, default ./.browser_playwright")
	fmt.Fprintln(w, "  --instance-state PATH      override browser_instance.json path for browser instance commands")
	fmt.Fprintln(w, "  --chrome PATH              override the Chrome executable used by instance create")
	fmt.Fprintln(w, "  --headless MODE            standalone Chrome headless mode override for instance create: new (default) or none")
	fmt.Fprintln(w, "  --instance-browser_expired N browser instance idle release timeout in minutes, default 10")
	fmt.Fprintln(w, "  --browser_expired N        shorthand for browser instance idle release timeout in minutes")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  ./browser fetch --cookie_path ./cookies.json")
	fmt.Fprintln(w, "  ./browser store --cookie_path ./cookies.json")
	fmt.Fprintln(w, "  ./browser --session agent-a@ctrip-home goto https://www.ctrip.com")
	fmt.Fprintln(w, "  ./browser --session agent-a@ctrip-home --navigation-timeout 120000 --browser-timeout 120s goto https://m.ctrip.com")
	fmt.Fprintln(w, "  ./browser --session agent-a@ctrip-home eval 'document.body ? document.body.innerText.slice(0, 1000) : \"\"'")
	fmt.Fprintln(w, "  ./browser --session agent-a@ctrip-home eval --code 'document.title'")
	fmt.Fprintln(w, "  ./browser --session agent-a@ctrip-home --timeout 15000 --browser-timeout 30s eval 'new Promise(resolve => setTimeout(resolve, 10000))'")
	fmt.Fprintln(w, "  ./browser shutdown --agentId agent-a --chatId chat-001")
	fmt.Fprintln(w, "  ./browser instance create --agentId agent-a --chatId chat-001")
	fmt.Fprintln(w, "  ./browser instance restart --agentId agent-a --chatId chat-001")
	fmt.Fprintln(w, "  ./browser instance shutdown --agentId agent-a --chatId chat-001")
	fmt.Fprintln(w, "  ./browser instance list")
	fmt.Fprintln(w, "  ./browser instance get --agentId agent-a --chatId ctrip-home")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Notes:")
	fmt.Fprintln(w, "  - browser logs are written to ./browser.log beside the browser executable")
	fmt.Fprintln(w, "  - managed Agent@Chat sessions refresh browser instance activity and idle instances are cleaned when instance state is reloaded")
	fmt.Fprintln(w, "  - browser shutdown force-stops one managed Chrome pid after resolving it from instance state")
	fmt.Fprintln(w, "  - supported platforms: macOS, Linux, and Windows")
	fmt.Fprintln(w, "  - outside WSL, instance create uses a managed chrome_${port} directory as the Chrome user-data-dir")
	fmt.Fprintln(w, "  - run `browser daemon help` or `browser instance help` for subcommand help")
	fmt.Fprintln(w, "")
}

func printDaemonHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  browser daemon help")
	fmt.Fprintln(w, "  browser daemon serve [flags]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Notes:")
	fmt.Fprintln(w, "  - daemon commands proxy the underlying browser_playwright service")
	fmt.Fprintln(w, "  - use Integration plugin lifecycle control for whole-plugin management")
}

func printInstanceHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  browser instance help")
	fmt.Fprintln(w, "  browser instance create --agentId AGENT --chatId CHAT")
	fmt.Fprintln(w, "  browser instance restart --agentId AGENT --chatId CHAT")
	fmt.Fprintln(w, "  browser instance shutdown [--agentId AGENT --chatId CHAT]")
	fmt.Fprintln(w, "  browser instance list")
	fmt.Fprintln(w, "  browser instance get --agentId AGENT --chatId CHAT")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Behavior:")
	fmt.Fprintln(w, "  - state file defaults to ./browser_instance.json beside the browser executable")
	fmt.Fprintln(w, "  - Chrome is resolved from the local system, or overridden with --chrome PATH")
	fmt.Fprintln(w, "  - on WSL, Chrome resolves browser meta chrome first and otherwise falls back to /mnt/c/Program Files/Google/Chrome/Application/chrome.exe")
	fmt.Fprintln(w, "  - create prefers browser meta headless from the recorded browser_runtime.json integration runtime: false => headed, true/invalid => --headless new, empty => fallback to --headless")
	fmt.Fprintln(w, "  - without a recorded integration runtime, create still accepts --headless none for standalone headed mode")
	fmt.Fprintln(w, "  - outside WSL, create reuses the managed chrome_${port} directory when it already exists")
	fmt.Fprintln(w, "  - on WSL, create calls browser_launcher.sh beside the plugin; profileDir still stays in the normal CLI response, and its value comes from the launcher-returned user-data-dir")
	fmt.Fprintln(w, "  - on WSL, browser_instance_wsl keeps one persistent Chat-scoped profile at C:\\ProgramData\\deepright\\profiles\\chats\\<chatId>; all agents inside that chat share it and it never copies system Chrome data")
	fmt.Fprintln(w, "  - on macOS/Linux/Windows, create clones a filtered copy of the current-system Chrome User Data root when chrome_${port} does not exist")
	fmt.Fprintln(w, "    filtering CacheStorage, OptGuideOnDeviceModel, and other volatile cache paths while keeping login storage such as WebStorage/IndexedDB/Local Storage")
	fmt.Fprintln(w, "  - create/get/list JSON also include profileDir so the resolved managed user-data-dir is visible")
	fmt.Fprintln(w, "  - restart replaces one managed instance with a fresh managed port")
	fmt.Fprintln(w, "  - shutdown also force-terminates that instance process and removes its saved state")
	fmt.Fprintln(w, "  - get/list/create/restart reload instance state and clean managed CDP instances that are already dead or expired")
}
