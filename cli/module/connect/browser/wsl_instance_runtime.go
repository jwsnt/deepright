package main

import (
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"connect/connectsvc"

	_ "github.com/glebarez/go-sqlite"
)

const (
	browserWSLInstanceDBName       = "browser_data"
	browserWSLInstanceTableName    = "browser_instance_wsl"
	browserWSLInstanceProfileRoot  = "/mnt/c/ProgramData/deepright"
	browserWSLInstanceProfileRootW = `C:\ProgramData\deepright`
	browserWSLInstancePollInterval = 5 * time.Second
	browserWSLInstanceTimeout      = 30 * time.Second
)

type browserWSLInstanceRecord struct {
	AgentID     string
	ChatID      string
	PID         int
	Port        int
	WS          string
	HTTP        string
	UserDataDir string
}

type browserWSLInstanceResponse struct {
	Status      int    `json:"status"`
	PID         int    `json:"pid,omitempty"`
	Port        int    `json:"port,omitempty"`
	WS          string `json:"ws,omitempty"`
	HTTP        string `json:"http,omitempty"`
	UserDataDir string `json:"user-data-dir,omitempty"`
	Message     string `json:"message,omitempty"`
}

type browserWSLInstanceVersion struct {
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

var (
	browserWSLInstanceLookupRecordFn  = browserWSLInstanceLookupRecord
	browserWSLInstanceCopyDirectoryFn = browserCopyWSLChromeUserDataWithProgress
)

func browserWSLInstanceCommand(args []string) int {
	if len(args) == 0 || strings.TrimSpace(args[0]) != "acquire" {
		return browserWSLInstanceWriteFailure(errors.New("unknown wsl instance command"))
	}
	flags, err := connectsvc.ParseFlags(args[1:])
	if err != nil {
		return browserWSLInstanceWriteFailure(err)
	}

	agentID := strings.TrimSpace(connectsvc.FirstValue(flags, "agentId", "agent"))
	chatID := strings.TrimSpace(connectsvc.FirstValue(flags, "chatId", "chat"))
	headless := connectsvc.BoolValue(flags, "headless", true)
	forceRecreate := browserWSLForceRecreateRequested()
	chromePath := strings.TrimSpace(connectsvc.FirstValue(flags, "chrome"))

	timeout := browserWSLInstanceTimeout
	if forceRecreate {
		timeout = browserWSLInstanceInitTimeoutFromEnvironment()
	}
	record, err := browserWSLInstanceAcquire(agentID, chatID, headless, forceRecreate, chromePath, timeout)
	if err != nil {
		return browserWSLInstanceWriteFailure(err)
	}
	_ = json.NewEncoder(os.Stdout).Encode(browserWSLInstanceResponse{
		Status:      0,
		PID:         record.PID,
		Port:        record.Port,
		WS:          record.WS,
		HTTP:        record.HTTP,
		UserDataDir: record.UserDataDir,
	})
	return 0
}

func browserWSLForceRecreateRequested() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(browserWSLForceRecreateEnv))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func browserWSLInstanceWriteFailure(err error) int {
	message := "unknown error"
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		message = strings.TrimSpace(err.Error())
	}
	_ = json.NewEncoder(os.Stdout).Encode(browserWSLInstanceResponse{
		Status:  1,
		Message: message,
	})
	return 1
}

func browserWSLInstanceAcquire(agentID, chatID string, headless, forceRecreate bool, chromePath string, timeout time.Duration) (browserWSLInstanceRecord, error) {
	if timeout <= 0 {
		timeout = browserWSLInstanceTimeout
	}
	if ok, err := browserWSLDetectFn(); err != nil {
		return browserWSLInstanceRecord{}, err
	} else if !ok {
		return browserWSLInstanceRecord{}, errors.New("browser_instance_wsl must run inside WSL")
	}

	agentID = browserNormalizeIdentityPart(agentID)
	chatID = browserNormalizeIdentityPart(chatID)
	if agentID == "" {
		return browserWSLInstanceRecord{}, errors.New("agentId is required")
	}
	if chatID == "" {
		return browserWSLInstanceRecord{}, errors.New("chatId is required")
	}

	chromePath = browserWSLInstanceResolveChromePath(chromePath)
	if _, err := os.Stat(chromePath); err != nil {
		return browserWSLInstanceRecord{}, fmt.Errorf("chrome not found: %s", chromePath)
	}

	db, err := browserWSLInstanceOpenDB()
	if err != nil {
		return browserWSLInstanceRecord{}, err
	}
	defer db.Close()

	existing, found, err := browserWSLInstanceGetRecord(db, agentID, chatID)
	if err != nil {
		return browserWSLInstanceRecord{}, err
	}
	if found {
		if forceRecreate {
			if err := browserWSLInstanceStopForRecreate(existing); err != nil {
				return browserWSLInstanceRecord{}, err
			}
			recreatedRecord, recreated, recreateErr := browserWSLInstanceRestartStaleRecord(agentID, chatID, existing, headless, chromePath, timeout)
			if recreateErr == nil && recreated {
				if err := browserWSLInstanceUpsertRecord(db, recreatedRecord); err != nil {
					return browserWSLInstanceRecord{}, err
				}
				return recreatedRecord, nil
			}
			if err := browserWSLInstanceDeleteRecord(db, agentID, chatID); err != nil {
				return browserWSLInstanceRecord{}, err
			}
		} else {
			liveRecord, live, probeErr := browserWSLInstanceProbeRecord(existing)
			if probeErr == nil && live {
				if err := browserWSLInstanceUpsertRecord(db, liveRecord); err != nil {
					return browserWSLInstanceRecord{}, err
				}
				return liveRecord, nil
			}
			reusedRecord, reused, reuseErr := browserWSLInstanceRestartStaleRecord(agentID, chatID, existing, headless, chromePath, timeout)
			if reuseErr == nil && reused {
				if err := browserWSLInstanceUpsertRecord(db, reusedRecord); err != nil {
					return browserWSLInstanceRecord{}, err
				}
				return reusedRecord, nil
			}
			if err := browserWSLInstanceDeleteRecord(db, agentID, chatID); err != nil {
				return browserWSLInstanceRecord{}, err
			}
		}
	}

	profileWin, profileUnix, err := browserWSLInstanceReserveProfileDir()
	if err != nil {
		return browserWSLInstanceRecord{}, err
	}

	launchPID, err := browserWSLInstanceStartChrome(chromePath, 0, profileWin, headless)
	if err != nil {
		_ = browserWSLInstanceRemoveProfile(profileWin)
		return browserWSLInstanceRecord{}, err
	}

	record, err := browserWSLInstanceWaitForReady(agentID, chatID, 0, profileWin, profileUnix, launchPID, timeout)
	if err != nil {
		_ = browserWSLTerminateProcessFn(launchPID, 0)
		return browserWSLInstanceRecord{}, err
	}
	if err := browserWSLInstanceUpsertRecord(db, record); err != nil {
		return browserWSLInstanceRecord{}, err
	}
	return record, nil
}

func browserWSLInstanceStopForRecreate(item browserWSLInstanceRecord) error {
	if item.Port <= 0 {
		return nil
	}
	if pid, found := browserWSLInstanceLookupPIDByPort(item.Port); found {
		return browserWSLTerminateProcessFn(pid, item.Port)
	}
	return browserWSLInstanceWaitForPortRelease(item.Port, 5*time.Second)
}

func browserWSLInstanceLookupUserDataDir(agentID, chatID string) (string, bool) {
	db, err := browserWSLInstanceOpenDB()
	if err != nil {
		return "", false
	}
	defer db.Close()

	record, found, err := browserWSLInstanceGetRecord(db, browserNormalizeIdentityPart(agentID), browserNormalizeIdentityPart(chatID))
	if err != nil || !found {
		return "", false
	}
	value := strings.TrimSpace(record.UserDataDir)
	return value, value != ""
}

func browserWSLInstanceLookupRecord(agentID, chatID string) (browserWSLInstanceRecord, bool) {
	db, err := browserWSLInstanceOpenDB()
	if err != nil {
		return browserWSLInstanceRecord{}, false
	}
	defer db.Close()

	record, found, err := browserWSLInstanceGetRecord(db, browserNormalizeIdentityPart(agentID), browserNormalizeIdentityPart(chatID))
	if err != nil || !found {
		return browserWSLInstanceRecord{}, false
	}
	return record, true
}

func browserWSLInstanceRestartStaleRecord(agentID, chatID string, item browserWSLInstanceRecord, headless bool, chromePath string, timeout time.Duration) (browserWSLInstanceRecord, bool, error) {
	if item.Port <= 0 {
		return browserWSLInstanceRecord{}, false, errors.New("stored port is invalid")
	}
	profileWin := strings.TrimSpace(item.UserDataDir)
	if profileWin == "" {
		return browserWSLInstanceRecord{}, false, errors.New("stored user-data-dir is empty")
	}
	profileUnix, err := browserWSLInstanceWindowsToUnixPath(profileWin)
	if err != nil {
		return browserWSLInstanceRecord{}, false, err
	}
	if _, err := os.Stat(profileUnix); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return browserWSLInstanceRecord{}, false, nil
		}
		return browserWSLInstanceRecord{}, false, err
	}
	if err := browserWSLInstanceCleanupProfileLocks(profileUnix); err != nil {
		return browserWSLInstanceRecord{}, false, err
	}
	if err := browserWSLInstanceWaitForPortRelease(item.Port, 5*time.Second); err != nil {
		return browserWSLInstanceRecord{}, false, err
	}
	launchPID, err := browserWSLInstanceStartChrome(chromePath, item.Port, profileWin, headless)
	if err != nil {
		return browserWSLInstanceRecord{}, false, err
	}
	record, err := browserWSLInstanceWaitForReady(agentID, chatID, item.Port, profileWin, profileUnix, launchPID, timeout)
	if err != nil {
		_ = browserWSLTerminateProcessFn(launchPID, item.Port)
		return browserWSLInstanceRecord{}, false, err
	}
	record.Port = item.Port
	return record, true, nil
}

func browserWSLInstanceOpenDB() (*sql.DB, error) {
	dbPath, err := browserWSLInstanceDBPath()
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open browser_data: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	if _, err := db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set browser_data busy_timeout: %w", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ` + browserWSLInstanceTableName + ` (
			agent_id TEXT NOT NULL,
			chat_id TEXT NOT NULL,
			pid INTEGER NOT NULL DEFAULT 0,
			port INTEGER NOT NULL DEFAULT 0,
			ws TEXT NOT NULL DEFAULT '',
			http TEXT NOT NULL DEFAULT '',
			user_data_dir TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL,
			PRIMARY KEY (agent_id, chat_id)
		)
	`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create browser_data table: %w", err)
	}
	return db, nil
}

func browserWSLInstanceDBPath() (string, error) {
	if cwd, err := os.Getwd(); err == nil && browserWSLInstanceLooksLikeBrowserDir(cwd) {
		return filepath.Join(cwd, browserWSLInstanceDBName), nil
	}
	root, _, err := browserRuntimeRoot(map[string]string{})
	if err != nil {
		return "", err
	}
	return filepath.Join(root, browserWSLInstanceDBName), nil
}

func browserWSLInstanceLooksLikeBrowserDir(dir string) bool {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return false
	}
	for _, name := range []string{"browser", "browser.exe", "browser_launcher.sh", "build.sh", "main.go"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func browserWSLInstanceGetRecord(db *sql.DB, agentID, chatID string) (browserWSLInstanceRecord, bool, error) {
	var item browserWSLInstanceRecord
	err := db.QueryRow(
		`SELECT agent_id, chat_id, pid, port, ws, http, user_data_dir FROM `+browserWSLInstanceTableName+` WHERE agent_id = ? AND chat_id = ?`,
		agentID,
		chatID,
	).Scan(&item.AgentID, &item.ChatID, &item.PID, &item.Port, &item.WS, &item.HTTP, &item.UserDataDir)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return browserWSLInstanceRecord{}, false, nil
		}
		return browserWSLInstanceRecord{}, false, fmt.Errorf("query browser_data failed: %w", err)
	}
	return item, true, nil
}

func browserWSLInstanceDeleteRecord(db *sql.DB, agentID, chatID string) error {
	if _, err := db.Exec(
		`DELETE FROM `+browserWSLInstanceTableName+` WHERE agent_id = ? AND chat_id = ?`,
		agentID,
		chatID,
	); err != nil {
		return fmt.Errorf("delete stale browser_data record failed: %w", err)
	}
	return nil
}

func browserWSLInstanceUpsertRecord(db *sql.DB, item browserWSLInstanceRecord) error {
	if _, err := db.Exec(
		`INSERT INTO `+browserWSLInstanceTableName+` (agent_id, chat_id, pid, port, ws, http, user_data_dir, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(agent_id, chat_id) DO UPDATE SET
			pid = excluded.pid,
			port = excluded.port,
			ws = excluded.ws,
			http = excluded.http,
			user_data_dir = excluded.user_data_dir,
			updated_at = excluded.updated_at`,
		item.AgentID,
		item.ChatID,
		item.PID,
		item.Port,
		item.WS,
		item.HTTP,
		item.UserDataDir,
		time.Now().Format(time.RFC3339),
	); err != nil {
		return fmt.Errorf("upsert browser_data record failed: %w", err)
	}
	return nil
}

func browserWSLInstanceProbeRecord(item browserWSLInstanceRecord) (browserWSLInstanceRecord, bool, error) {
	if item.Port <= 0 {
		return browserWSLInstanceRecord{}, false, errors.New("stored port is invalid")
	}
	ws, httpBase, err := browserWSLInstanceProbeCDP(item.Port)
	if err != nil {
		return browserWSLInstanceRecord{}, false, err
	}
	pid := item.PID
	if livePID, live := browserWSLInstanceLookupPIDByPort(item.Port); live {
		pid = livePID
	}
	item.PID = pid
	item.WS = ws
	item.HTTP = httpBase
	if strings.TrimSpace(item.UserDataDir) == "" {
		if userDataDir, ok := browserWSLInstanceLookupUserDataDirByPID(pid); ok {
			item.UserDataDir = userDataDir
		}
	}
	return item, true, nil
}

func browserWSLInstanceReserveProfileDir() (string, string, error) {
	if err := os.MkdirAll(browserWSLInstanceProfileRoot, 0o755); err != nil {
		return "", "", fmt.Errorf("create profile root failed: %w", err)
	}

	for attempt := 0; attempt < 256; attempt++ {
		suffix, err := browserWSLInstanceRandomSuffix(4)
		if err != nil {
			return "", "", err
		}
		dirName := "chrome_" + suffix
		profileUnix := filepath.Join(browserWSLInstanceProfileRoot, dirName)
		if _, err := os.Stat(profileUnix); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", "", fmt.Errorf("check profile dir failed: %w", err)
		}
		if err := os.Mkdir(profileUnix, 0o755); err != nil {
			if errors.Is(err, os.ErrExist) {
				continue
			}
			return "", "", fmt.Errorf("reserve profile dir failed: %w", err)
		}
		profileWin := browserWSLInstanceProfileRootW + `\` + dirName
		if err := browserWSLInstanceSeedReservedProfileDir(profileWin, profileUnix); err != nil {
			_ = os.RemoveAll(profileUnix)
			return "", "", err
		}
		return profileWin, profileUnix, nil
	}
	return "", "", errors.New("allocate unique chrome user-data-dir failed")
}

func browserWSLInstanceSeedReservedProfileDir(profileWin, profileUnix string) error {
	sourceWin := browserWSLChromeDefRoot
	fields := map[string]any{
		"profileDir": profileWin,
		"sourceDir":  sourceWin,
	}

	sourceUnix, err := browserWSLInstanceWindowsToUnixPath(sourceWin)
	if err != nil {
		browserWSLInstanceTraceSeedSkip(fields, "resolve_source_error", err)
		return browserWSLInstanceResetReservedProfileDir(profileUnix)
	}
	fields["sourceDir"] = sourceWin

	sourceInfo, err := os.Stat(sourceUnix)
	switch {
	case errors.Is(err, os.ErrNotExist):
		browserWSLInstanceTraceSeedSkip(fields, "source_missing", err)
		return browserWSLInstanceResetReservedProfileDir(profileUnix)
	case err != nil:
		browserWSLInstanceTraceSeedSkip(fields, "source_stat_error", err)
		return browserWSLInstanceResetReservedProfileDir(profileUnix)
	case !sourceInfo.IsDir():
		browserWSLInstanceTraceSeedSkip(fields, "source_not_dir", fmt.Errorf("chrome_def is not a directory: %s", sourceWin))
		return browserWSLInstanceResetReservedProfileDir(profileUnix)
	}

	browserCreateTrace("instance.wsl.user_data.seed.begin", fields)
	copyStats, err := browserWSLInstanceCopyDirectoryFn(
		sourceUnix,
		profileUnix,
		browserInstanceUserDataProgressTracer("instance.wsl.user_data.seed.progress", fields),
		browserInstanceUserDataSkipTracer("instance.wsl.user_data.seed.skip", fields),
	)
	if err != nil {
		browserWSLInstanceTraceSeedSkip(fields, "copy_error", err)
		return browserWSLInstanceResetReservedProfileDir(profileUnix)
	}

	if err := browserWSLCleanupChromeUserDataFn(profileWin); err != nil {
		cleanupFields := map[string]any{
			"profileDir": profileWin,
			"sourceDir":  sourceWin,
			"error":      err.Error(),
		}
		browserCreateTrace("instance.wsl.user_data.seed.cleanup.warn", cleanupFields)
	}

	readyFields := map[string]any{
		"profileDir": profileWin,
		"sourceDir":  sourceWin,
		"files":      copyStats.Files,
		"dirs":       copyStats.Dirs,
		"bytes":      copyStats.Bytes,
	}
	browserCreateTrace("instance.wsl.user_data.seed.ok", readyFields)
	return nil
}

func browserWSLInstanceResetReservedProfileDir(profileUnix string) error {
	profileUnix = strings.TrimSpace(profileUnix)
	if profileUnix == "" {
		return errors.New("profile dir is required")
	}
	if err := os.RemoveAll(profileUnix); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reset reserved profile dir failed: %w", err)
	}
	if err := os.MkdirAll(profileUnix, 0o755); err != nil {
		return fmt.Errorf("create reserved profile dir failed: %w", err)
	}
	return nil
}

func browserWSLInstanceTraceSeedSkip(fields map[string]any, reason string, err error) {
	traceFields := map[string]any{
		"reason": reason,
	}
	for key, value := range fields {
		traceFields[key] = value
	}
	if err != nil {
		traceFields["error"] = err.Error()
	}
	browserCreateTrace("instance.wsl.user_data.seed.skip", traceFields)
}

func browserWSLInstanceStartChrome(chromePath string, port int, profileWin string, headless bool) (int, error) {
	chromePath = browserWSLInstanceResolveChromePath(chromePath)
	portArg := "--remote-debugging-port=0"
	if port > 0 {
		portArg = "--remote-debugging-port=" + strconv.Itoa(port)
	}
	args := []string{
		portArg,
		"--remote-debugging-address=0.0.0.0",
		"--user-data-dir=" + profileWin,
		"--no-first-run",
	}
	if headless {
		args = append(args, "--headless=new")
	}

	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return 0, fmt.Errorf("open %s failed: %w", os.DevNull, err)
	}

	cmd := exec.Command(chromePath, args...)
	cmd.Stdin = devNull
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	cmd.Dir = filepath.Dir(chromePath)

	if err := cmd.Start(); err != nil {
		_ = devNull.Close()
		return 0, fmt.Errorf("start chrome failed: %w", err)
	}
	go func() {
		defer func() {
			_ = devNull.Close()
		}()
		_ = cmd.Wait()
	}()
	return cmd.Process.Pid, nil
}

func browserWSLInstanceResolveChromePath(chromePath string) string {
	chromePath = strings.TrimSpace(chromePath)
	if chromePath == "" {
		return browserWSLDefaultChromePath
	}
	return chromePath
}

func browserWSLInstanceWaitForReady(agentID, chatID string, expectedPort int, profileWin, profileUnix string, launchPID int, timeout time.Duration) (browserWSLInstanceRecord, error) {
	if timeout <= 0 {
		timeout = browserWSLInstanceTimeout
	}
	deadline := time.Now().Add(timeout)
	var lastErr error

	for {
		port := expectedPort
		if port <= 0 {
			var err error
			port, err = browserWSLInstanceReadDevToolsPort(profileUnix)
			if err != nil {
				lastErr = err
				if time.Now().After(deadline) {
					break
				}
				time.Sleep(browserWSLInstancePollInterval)
				continue
			}
		}
		if port > 0 {
			ws, httpBase, probeErr := browserWSLInstanceProbeCDP(port)
			if probeErr == nil {
				pid := launchPID
				if livePID, ok := browserWSLInstanceLookupPIDByPort(port); ok {
					pid = livePID
				}
				return browserWSLInstanceRecord{
					AgentID:     agentID,
					ChatID:      chatID,
					PID:         pid,
					Port:        port,
					WS:          ws,
					HTTP:        httpBase,
					UserDataDir: profileWin,
				}, nil
			}
			lastErr = probeErr
		}

		if time.Now().After(deadline) {
			break
		}
		time.Sleep(browserWSLInstancePollInterval)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("cdp not ready after %s", timeout)
	}
	return browserWSLInstanceRecord{}, lastErr
}

func browserWSLInstanceInitTimeoutFromEnvironment() time.Duration {
	raw := strings.TrimSpace(os.Getenv(browserWSLInitTimeoutEnv))
	if raw == "" {
		return browserWSLInstanceTimeout
	}
	seconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || seconds <= 0 {
		return browserWSLInstanceTimeout
	}
	return time.Duration(seconds) * time.Second
}

func browserWSLInstanceWaitForPortRelease(port int, timeout time.Duration) error {
	if port <= 0 {
		return nil
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	deadline := time.Now().Add(timeout)
	for {
		if _, ok := browserWSLInstanceLookupPIDByPort(port); !ok {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("port %d did not release before restart", port)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func browserWSLInstanceReadDevToolsPort(profileUnix string) (int, error) {
	data, err := os.ReadFile(filepath.Join(profileUnix, "DevToolsActivePort"))
	if err != nil {
		return 0, err
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return 0, errors.New("DevToolsActivePort is empty")
	}
	port, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil || port <= 0 {
		return 0, fmt.Errorf("invalid DevToolsActivePort value: %q", strings.TrimSpace(lines[0]))
	}
	return port, nil
}

func browserWSLInstanceProbeCDP(port int) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("http://localhost:%d/json/version", port)
	cmd := exec.CommandContext(ctx, "curl", "-s", url)
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("curl check failed for port %d: %w", port, err)
	}

	var version browserWSLInstanceVersion
	if err := json.Unmarshal(output, &version); err != nil {
		return "", "", fmt.Errorf("parse /json/version failed for port %d: %w", port, err)
	}
	ws := strings.TrimSpace(version.WebSocketDebuggerURL)
	if ws == "" {
		return "", "", fmt.Errorf("empty webSocketDebuggerUrl for port %d", port)
	}
	return ws, fmt.Sprintf("http://localhost:%d", port), nil
}

func browserWSLInstanceLookupPIDByPort(port int) (int, bool) {
	output, err := exec.Command("/mnt/c/Windows/System32/netstat.exe", "-ano").Output()
	if err != nil {
		return 0, false
	}

	for _, line := range strings.Split(strings.ReplaceAll(string(output), "\r", ""), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 || !strings.EqualFold(fields[0], "TCP") {
			continue
		}
		localAddr := fields[1]
		pidText := fields[len(fields)-1]
		if !strings.HasSuffix(localAddr, ":"+strconv.Itoa(port)) {
			continue
		}
		pid, err := strconv.Atoi(pidText)
		if err != nil || pid <= 0 {
			continue
		}
		return pid, true
	}
	return 0, false
}

func browserWSLInstanceLookupUserDataDirByPID(pid int) (string, bool) {
	if pid <= 0 {
		return "", false
	}
	if userDataDir, ok := browserWSLInstanceLookupUserDataDirFromProc(pid); ok {
		return userDataDir, true
	}
	if userDataDir, ok := browserWSLInstanceLookupUserDataDirFromPowerShell(pid); ok {
		return userDataDir, true
	}
	return "", false
}

func browserWSLInstanceLookupUserDataDirFromProc(pid int) (string, bool) {
	cmdlinePath := filepath.Join("/proc", strconv.Itoa(pid), "cmdline")
	data, err := os.ReadFile(cmdlinePath)
	if err != nil || len(data) == 0 {
		return "", false
	}
	for _, arg := range strings.Split(string(data), "\x00") {
		const prefix = "--user-data-dir="
		if strings.HasPrefix(strings.TrimSpace(arg), prefix) {
			value := strings.TrimSpace(strings.TrimPrefix(arg, prefix))
			if value != "" {
				return value, true
			}
		}
	}
	return "", false
}

func browserWSLInstanceLookupUserDataDirFromPowerShell(pid int) (string, bool) {
	script := fmt.Sprintf(`$proc = Get-CimInstance Win32_Process -Filter "ProcessId = %d"
if ($null -eq $proc -or [string]::IsNullOrWhiteSpace($proc.CommandLine)) { exit 1 }
$match = [regex]::Match($proc.CommandLine, '--user-data-dir=("([^"]+)"|(\S+))')
if (-not $match.Success) { exit 1 }
if (-not [string]::IsNullOrWhiteSpace($match.Groups[2].Value)) {
  [Console]::Out.Write($match.Groups[2].Value)
  exit 0
}
[Console]::Out.Write($match.Groups[3].Value)
`, pid)
	output, err := exec.Command(
		browserWSLPowerShellPath,
		"-NoProfile",
		"-NonInteractive",
		"-Command",
		script,
	).Output()
	if err != nil {
		return "", false
	}
	value := strings.TrimSpace(strings.ReplaceAll(string(output), "\r", ""))
	if value == "" {
		return "", false
	}
	return value, true
}

func browserWSLInstanceCleanupProfileLocks(profileUnix string) error {
	profileUnix = strings.TrimSpace(profileUnix)
	if profileUnix == "" {
		return nil
	}
	if _, err := os.Stat(profileUnix); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return filepath.Walk(profileUnix, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if info == nil || path == profileUnix {
			return nil
		}
		if !browserWSLInstanceShouldRemoveLockEntry(info.Name()) {
			return nil
		}
		if err := os.RemoveAll(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if info.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})
}

func browserWSLInstanceShouldRemoveLockEntry(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	switch strings.ToUpper(name) {
	case "LOCK", "LOCKFILE", "SINGLETONLOCK", "SINGLETONCOOKIE", "SINGLETONSOCKET", "DEVTOOLSACTIVEPORT":
		return true
	}
	lowerName := strings.ToLower(name)
	return strings.HasSuffix(lowerName, ".lock") || strings.HasSuffix(lowerName, "-journal")
}

func browserWSLInstanceRemoveProfile(profileWin string) error {
	profileWin = strings.TrimSpace(profileWin)
	if profileWin == "" {
		return nil
	}
	profileUnix, err := browserWSLInstanceWindowsToUnixPath(profileWin)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(profileUnix); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func browserWSLInstanceWindowsToUnixPath(path string) (string, error) {
	return browserWSLPathUnixFn(path)
}

func browserWSLInstanceRandomSuffix(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("random suffix length must be positive")
	}
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	data := make([]byte, length)
	for idx := range data {
		value, err := browserWSLInstanceRandomInt(len(alphabet))
		if err != nil {
			return "", fmt.Errorf("generate random suffix failed: %w", err)
		}
		data[idx] = alphabet[value]
	}
	return string(data), nil
}

func browserWSLInstanceRandomInt(max int) (int, error) {
	if max <= 0 {
		return 0, fmt.Errorf("invalid random max: %d", max)
	}
	value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(value.Int64()), nil
}
