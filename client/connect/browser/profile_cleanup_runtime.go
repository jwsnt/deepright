package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	browserProfileCleanupCommand     = "__profile-cleaner"
	browserProfileCleanupPIDFileName = "browser_profile_cleanup.pid"
	browserWSLProfileCleanupRoot     = `C:\ProgramData\deepright\profiles\chats`
)

type browserProfileCleanupConfig struct {
	ClearAfter time.Duration
	ScanEvery  time.Duration
}

type browserProfileCleanupTarget struct {
	Platform string
	Root     string
	AgentDir bool
	ChatDir  bool
}

type browserProfileCleanupResult struct {
	Platform string
	Root     string
	Scanned  int
	Removed  int
	Skipped  int
	Errors   int
}

var (
	browserProfileCleanupStartFn = browserStartProfileCleanupWorker
	browserProfileCleanupStopFn  = browserStopProfileCleanupWorker
	browserProfileCleanupScanFn  = browserRunProfileCleanupScan
	browserProfileCleanupExecFn  = exec.Command
)

func runBrowserProfileCleanupCommand(args []string, stderr io.Writer) int {
	if len(args) > 0 {
		fmt.Fprintln(stderr, "browser profile cleanup worker does not accept arguments")
		return 1
	}
	browserRunProfileCleanupWorker()
	return 0
}

func browserStartProfileCleanupWorker() error {
	if err := browserStopProfileCleanupWorker(); err != nil {
		return fmt.Errorf("stop existing profile cleanup worker: %w", err)
	}
	executable, err := browserExecutablePathFn()
	if err != nil {
		return err
	}
	if resolved, resolveErr := filepath.EvalSymlinks(executable); resolveErr == nil {
		executable = resolved
	}
	cmd := browserProfileCleanupExecFn(executable, browserProfileCleanupCommand)
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer devNull.Close()
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	browserPrepareDetachedCommand(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	pid := cmd.Process.Pid
	if err := cmd.Process.Release(); err != nil {
		_ = browserTerminateProcessFn(pid)
		return err
	}
	if err := browserWriteProfileCleanupPID(pid); err != nil {
		_ = browserTerminateProcessFn(pid)
		return err
	}
	browserLogProfileCleanupEvent("worker_started", browserProfileCleanupResult{}, pid, nil)
	return nil
}

func browserStopProfileCleanupWorker() error {
	pid, ok, err := browserReadProfileCleanupPID()
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if browserProcessExistsFn(pid) {
		if err := browserTerminateProcessFn(pid); err != nil {
			return err
		}
	}
	if err := browserRemoveProfileCleanupPID(); err != nil {
		return err
	}
	browserLogProfileCleanupEvent("worker_stopped", browserProfileCleanupResult{}, pid, nil)
	return nil
}

func browserRunProfileCleanupWorker() {
	pid := os.Getpid()
	defer browserRemoveOwnProfileCleanupPID(pid)

	config, err := browserLoadProfileCleanupConfig()
	if err != nil {
		browserLogProfileCleanupEvent("config_skip", browserProfileCleanupResult{}, pid, err)
		return
	}
	browserLogProfileCleanupEvent("worker_ready", browserProfileCleanupResult{}, pid, nil)
	browserProfileCleanupScanFn(config)

	ticker := time.NewTicker(config.ScanEvery)
	defer ticker.Stop()
	for range ticker.C {
		browserProfileCleanupScanFn(config)
	}
}

func browserLoadProfileCleanupConfig() (browserProfileCleanupConfig, error) {
	runtimePath, ok, err := browserResolveRecordedRuntimeConfigPath()
	if err != nil {
		return browserProfileCleanupConfig{}, fmt.Errorf("resolve main application config: %w", err)
	}
	if !ok {
		return browserProfileCleanupConfig{}, errors.New("main application config is unavailable")
	}
	data, err := browserReadFileFn(runtimePath)
	if err != nil {
		return browserProfileCleanupConfig{}, fmt.Errorf("read main application config %s: %w", runtimePath, err)
	}
	config, err := browserParseProfileCleanupConfig(data)
	if err != nil {
		return browserProfileCleanupConfig{}, fmt.Errorf("parse main application config %s: %w", runtimePath, err)
	}
	return config, nil
}

func browserParseProfileCleanupConfig(data []byte) (browserProfileCleanupConfig, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return browserProfileCleanupConfig{}, err
	}
	if root == nil {
		return browserProfileCleanupConfig{}, errors.New("config root must be an object")
	}
	browserRaw, ok := root["browser"]
	if !ok {
		return browserProfileCleanupConfig{}, errors.New("browser configuration is missing")
	}
	var browserConfig map[string]json.RawMessage
	if err := json.Unmarshal(browserRaw, &browserConfig); err != nil || browserConfig == nil {
		return browserProfileCleanupConfig{}, errors.New("browser configuration must be an object")
	}
	clearAfter, err := browserProfileCleanupHours(browserConfig, "clear")
	if err != nil {
		return browserProfileCleanupConfig{}, err
	}
	scanEvery, err := browserProfileCleanupHours(browserConfig, "scan")
	if err != nil {
		return browserProfileCleanupConfig{}, err
	}
	return browserProfileCleanupConfig{ClearAfter: clearAfter, ScanEvery: scanEvery}, nil
}

func browserProfileCleanupHours(config map[string]json.RawMessage, key string) (time.Duration, error) {
	raw, ok := config[key]
	if !ok {
		return 0, fmt.Errorf("browser.%s is missing", key)
	}
	var hours int64
	if err := json.Unmarshal(raw, &hours); err != nil || hours <= 0 {
		return 0, fmt.Errorf("browser.%s must be a positive integer number of hours", key)
	}
	if hours > int64((time.Duration(1<<63-1))/time.Hour) {
		return 0, fmt.Errorf("browser.%s is too large", key)
	}
	return time.Duration(hours) * time.Hour, nil
}

func browserRunProfileCleanupScan(config browserProfileCleanupConfig) {
	target, ok, err := browserResolveProfileCleanupTarget()
	if err != nil {
		browserLogProfileCleanupEvent("scan_target_error", browserProfileCleanupResult{}, 0, err)
		return
	}
	if !ok {
		browserLogProfileCleanupEvent("scan_unsupported_platform", browserProfileCleanupResult{}, 0, nil)
		return
	}
	result := browserProfileCleanupResult{Platform: target.Platform, Root: target.Root}
	browserLogProfileCleanupEvent("scan_started", result, 0, nil)
	if target.AgentDir {
		browserScanAgentProfileDirectories(target.Root, config.ClearAfter, &result)
	} else if target.ChatDir {
		browserScanChatProfileDirectories(target.Root, config.ClearAfter, &result)
	} else {
		browserScanProfileDirectories(target.Root, config.ClearAfter, &result)
	}
	browserLogProfileCleanupEvent("scan_finished", result, 0, nil)
}

func browserResolveProfileCleanupTarget() (browserProfileCleanupTarget, bool, error) {
	isWSL, err := browserWSLDetectFn()
	if err != nil {
		return browserProfileCleanupTarget{}, false, err
	}
	if isWSL {
		root, err := browserWSLPathUnixFn(browserWSLProfileCleanupRoot)
		if err != nil {
			return browserProfileCleanupTarget{}, false, fmt.Errorf("resolve WSL cleanup root %s: %w", browserWSLProfileCleanupRoot, err)
		}
		root = strings.TrimSpace(root)
		if root == "" {
			return browserProfileCleanupTarget{}, false, fmt.Errorf("resolve WSL cleanup root %s: empty path", browserWSLProfileCleanupRoot)
		}
		return browserProfileCleanupTarget{Platform: "wsl", Root: root, ChatDir: true}, true, nil
	}
	if browserRuntimeGOOSFn() != "darwin" {
		return browserProfileCleanupTarget{}, false, nil
	}
	root, err := browserResolveAgentRoot(map[string]string{})
	if err != nil {
		return browserProfileCleanupTarget{}, false, fmt.Errorf("resolve macOS agent directory: %w", err)
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return browserProfileCleanupTarget{}, false, errors.New("resolve macOS agent directory: empty path")
	}
	return browserProfileCleanupTarget{Platform: "macos", Root: root, AgentDir: true}, true, nil
}

func browserScanAgentProfileDirectories(agentRoot string, clearAfter time.Duration, result *browserProfileCleanupResult) {
	entries, err := browserReadDirFn(agentRoot)
	if err != nil {
		browserRecordProfileCleanupError(result, agentRoot, err)
		return
	}
	for _, entry := range entries {
		if entry == nil || entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() {
			continue
		}
		browserScanProfileDirectories(filepath.Join(agentRoot, entry.Name()), clearAfter, result)
	}
}

func browserScanProfileDirectories(root string, clearAfter time.Duration, result *browserProfileCleanupResult) {
	browserScanProfileDirectoriesMatching(root, clearAfter, result, browserIsChromeProfileDirectory)
}

func browserScanChatProfileDirectories(root string, clearAfter time.Duration, result *browserProfileCleanupResult) {
	browserScanProfileDirectoriesMatching(root, clearAfter, result, func(string) bool { return true })
}

func browserScanProfileDirectoriesMatching(root string, clearAfter time.Duration, result *browserProfileCleanupResult, matches func(string) bool) {
	entries, err := browserReadDirFn(root)
	if err != nil {
		browserRecordProfileCleanupError(result, root, err)
		return
	}
	cutoff := browserNowFn().Add(-clearAfter)
	for _, entry := range entries {
		if entry == nil || entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() || !matches(entry.Name()) {
			continue
		}
		path := filepath.Join(root, entry.Name())
		info, err := entry.Info()
		if err != nil {
			browserRecordProfileCleanupError(result, path, err)
			continue
		}
		result.Scanned++
		if !info.ModTime().Before(cutoff) {
			result.Skipped++
			continue
		}
		if err := browserRemoveAllFn(path); err != nil {
			browserRecordProfileCleanupError(result, path, err)
			continue
		}
		result.Removed++
		browserLogProfileCleanupEvent("directory_removed", browserProfileCleanupResult{
			Platform: result.Platform,
			Root:     path,
			Removed:  1,
		}, 0, nil)
	}
}

func browserIsChromeProfileDirectory(name string) bool {
	name = strings.TrimSpace(name)
	return len(name) > len("chrome_") && strings.HasPrefix(strings.ToLower(name), "chrome_")
}

func browserRecordProfileCleanupError(result *browserProfileCleanupResult, path string, err error) {
	logResult := browserProfileCleanupResult{Root: path}
	if result != nil {
		result.Errors++
		logResult.Platform = result.Platform
	}
	browserLogProfileCleanupEvent("scan_error", logResult, 0, err)
}

func browserLogProfileCleanupEvent(stage string, result browserProfileCleanupResult, pid int, err error) {
	payload := map[string]any{
		"event":     "browser_profile_cleanup",
		"stage":     strings.TrimSpace(stage),
		"timestamp": browserNowFn().Format(time.RFC3339Nano),
	}
	if result.Platform != "" {
		payload["platform"] = result.Platform
	}
	if result.Root != "" {
		payload["root"] = result.Root
	}
	if result.Scanned > 0 {
		payload["scanned"] = result.Scanned
	}
	if result.Removed > 0 {
		payload["removed"] = result.Removed
	}
	if result.Skipped > 0 {
		payload["skipped"] = result.Skipped
	}
	if result.Errors > 0 {
		payload["errors"] = result.Errors
	}
	if pid > 0 {
		payload["pid"] = pid
	}
	if err != nil {
		payload["error"] = err.Error()
	}
	browserAppendLogJSON(payload)
}

func browserProfileCleanupPIDPath() (string, error) {
	runtimePath, err := browserRuntimeFilePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(runtimePath), browserProfileCleanupPIDFileName), nil
}

func browserReadProfileCleanupPID() (int, bool, error) {
	path, err := browserProfileCleanupPIDPath()
	if err != nil {
		return 0, false, err
	}
	data, err := browserReadFileFn(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, false, nil
		}
		return 0, false, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		if removeErr := browserRemoveAllFn(path); removeErr != nil && !os.IsNotExist(removeErr) {
			return 0, false, removeErr
		}
		return 0, false, nil
	}
	return pid, true, nil
}

func browserWriteProfileCleanupPID(pid int) error {
	if pid <= 0 {
		return errors.New("profile cleanup worker pid is invalid")
	}
	path, err := browserProfileCleanupPIDPath()
	if err != nil {
		return err
	}
	return browserWriteFileFn(path, []byte(strconv.Itoa(pid)+"\n"), 0o644)
}

func browserRemoveProfileCleanupPID() error {
	path, err := browserProfileCleanupPIDPath()
	if err != nil {
		return err
	}
	if err := browserRemoveAllFn(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func browserRemoveOwnProfileCleanupPID(pid int) {
	storedPID, ok, err := browserReadProfileCleanupPID()
	if err != nil || !ok || storedPID != pid {
		return
	}
	_ = browserRemoveProfileCleanupPID()
}
