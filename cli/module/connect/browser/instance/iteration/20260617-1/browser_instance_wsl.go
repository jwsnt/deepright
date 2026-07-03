//go:build ignore

package main

import (
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

const (
	browserInstanceWSLChromePath   = "/mnt/c/Program Files/Google/Chrome/Application/chrome.exe"
	browserInstanceWSLDBName       = "browser_data"
	browserInstanceWSLTableName    = "browser_instance_wsl"
	browserInstanceWSLProfileRoot  = "/mnt/c/temp"
	browserInstanceWSLProfileRootW = `C:\temp`
	browserInstanceWSLPollInterval = 5 * time.Second
	browserInstanceWSLTimeout      = 30 * time.Second
)

type browserInstanceWSLRecord struct {
	AgentID     string
	ChatID      string
	PID         int
	Port        int
	WS          string
	HTTP        string
	UserDataDir string
}

type browserInstanceWSLResponse struct {
	Status      int    `json:"status"`
	PID         int    `json:"pid,omitempty"`
	Port        int    `json:"port,omitempty"`
	WS          string `json:"ws,omitempty"`
	HTTP        string `json:"http,omitempty"`
	UserDataDir string `json:"user-data-dir,omitempty"`
	Message     string `json:"message,omitempty"`
}

type browserInstanceWSLVersion struct {
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

func main() {
	os.Exit(runBrowserInstanceWSL(os.Args[1:]))
}

func runBrowserInstanceWSL(args []string) int {
	fs := flag.NewFlagSet("browser_instance_wsl", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		agentID    string
		chatID     string
		headless   bool
		chromePath string
	)
	fs.StringVar(&agentID, "agentId", "", "managed browser agentId")
	fs.StringVar(&chatID, "chatId", "", "managed browser chatId")
	fs.BoolVar(&headless, "headless", true, "launch chrome in headless mode")
	fs.StringVar(&chromePath, "chrome", browserInstanceWSLChromePath, "chrome executable path")

	if err := fs.Parse(args); err != nil {
		return browserInstanceWSLWriteFailure(err)
	}

	record, err := browserInstanceWSLAcquire(agentID, chatID, headless, chromePath)
	if err != nil {
		return browserInstanceWSLWriteFailure(err)
	}

	_ = json.NewEncoder(os.Stdout).Encode(browserInstanceWSLResponse{
		Status:      0,
		PID:         record.PID,
		Port:        record.Port,
		WS:          record.WS,
		HTTP:        record.HTTP,
		UserDataDir: record.UserDataDir,
	})
	return 0
}

func browserInstanceWSLWriteFailure(err error) int {
	message := "unknown error"
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		message = strings.TrimSpace(err.Error())
	}
	_ = json.NewEncoder(os.Stdout).Encode(browserInstanceWSLResponse{
		Status:  1,
		Message: message,
	})
	return 1
}

func browserInstanceWSLAcquire(agentID, chatID string, headless bool, chromePath string) (browserInstanceWSLRecord, error) {
	if ok, err := browserInstanceWSLIsWSL(); err != nil {
		return browserInstanceWSLRecord{}, err
	} else if !ok {
		return browserInstanceWSLRecord{}, errors.New("browser_instance_wsl must run inside WSL")
	}

	agentID = browserInstanceWSLNormalizeIdentity(agentID)
	chatID = browserInstanceWSLNormalizeIdentity(chatID)
	if agentID == "" {
		return browserInstanceWSLRecord{}, errors.New("agentId is required")
	}
	if chatID == "" {
		return browserInstanceWSLRecord{}, errors.New("chatId is required")
	}

	chromePath = browserInstanceWSLResolveChromePath(chromePath)
	if _, err := os.Stat(chromePath); err != nil {
		return browserInstanceWSLRecord{}, fmt.Errorf("chrome not found: %s", chromePath)
	}

	db, err := browserInstanceWSLOpenDB()
	if err != nil {
		return browserInstanceWSLRecord{}, err
	}
	defer db.Close()

	existing, found, err := browserInstanceWSLGetRecord(db, agentID, chatID)
	if err != nil {
		return browserInstanceWSLRecord{}, err
	}
	if found {
		liveRecord, live, probeErr := browserInstanceWSLProbeRecord(existing)
		if probeErr == nil && live {
			if err := browserInstanceWSLUpsertRecord(db, liveRecord); err != nil {
				return browserInstanceWSLRecord{}, err
			}
			return liveRecord, nil
		}
		reusedRecord, reused, reuseErr := browserInstanceWSLRestartStaleRecord(agentID, chatID, existing, headless, chromePath)
		if reuseErr == nil && reused {
			if err := browserInstanceWSLUpsertRecord(db, reusedRecord); err != nil {
				return browserInstanceWSLRecord{}, err
			}
			return reusedRecord, nil
		}
	}

	profileWin, profileUnix, err := browserInstanceWSLReserveProfileDir()
	if err != nil {
		return browserInstanceWSLRecord{}, err
	}

	launchPID, err := browserInstanceWSLStartChrome(chromePath, 0, profileWin, headless)
	if err != nil {
		_ = browserInstanceWSLRemoveProfile(profileWin)
		return browserInstanceWSLRecord{}, err
	}

	record, err := browserInstanceWSLWaitForReady(agentID, chatID, 0, profileWin, profileUnix, launchPID)
	if err != nil {
		_ = browserInstanceWSLRemoveProfile(profileWin)
		return browserInstanceWSLRecord{}, err
	}
	if err := browserInstanceWSLUpsertRecord(db, record); err != nil {
		return browserInstanceWSLRecord{}, err
	}
	return record, nil
}

func browserInstanceWSLRestartStaleRecord(agentID, chatID string, item browserInstanceWSLRecord, headless bool, chromePath string) (browserInstanceWSLRecord, bool, error) {
	if item.Port <= 0 {
		return browserInstanceWSLRecord{}, false, errors.New("stored port is invalid")
	}
	profileWin := strings.TrimSpace(item.UserDataDir)
	if profileWin == "" {
		return browserInstanceWSLRecord{}, false, errors.New("stored user-data-dir is empty")
	}
	profileUnix, err := browserInstanceWSLWindowsToUnixPath(profileWin)
	if err != nil {
		return browserInstanceWSLRecord{}, false, err
	}
	if _, err := os.Stat(profileUnix); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return browserInstanceWSLRecord{}, false, nil
		}
		return browserInstanceWSLRecord{}, false, err
	}
	if err := browserInstanceWSLCleanupProfileLocks(profileUnix); err != nil {
		return browserInstanceWSLRecord{}, false, err
	}
	if err := browserInstanceWSLWaitForPortRelease(item.Port, 5*time.Second); err != nil {
		return browserInstanceWSLRecord{}, false, err
	}
	launchPID, err := browserInstanceWSLStartChrome(chromePath, item.Port, profileWin, headless)
	if err != nil {
		return browserInstanceWSLRecord{}, false, err
	}
	record, err := browserInstanceWSLWaitForReady(agentID, chatID, item.Port, profileWin, profileUnix, launchPID)
	if err != nil {
		return browserInstanceWSLRecord{}, false, err
	}
	record.Port = item.Port
	return record, true, nil
}

func browserInstanceWSLOpenDB() (*sql.DB, error) {
	dbPath, err := browserInstanceWSLDBPath()
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open browser_data: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if _, err := db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set browser_data busy_timeout: %w", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ` + browserInstanceWSLTableName + ` (
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

func browserInstanceWSLDBPath() (string, error) {
	if cwd, err := os.Getwd(); err == nil {
		if browserInstanceWSLLooksLikeBrowserDir(cwd) {
			return filepath.Join(cwd, browserInstanceWSLDBName), nil
		}
	}
	_, filePath, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("resolve browser_instance_wsl directory failed")
	}
	return filepath.Join(filepath.Dir(filePath), browserInstanceWSLDBName), nil
}

func browserInstanceWSLLooksLikeBrowserDir(dir string) bool {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return false
	}
	for _, name := range []string{"main.go", "browser_instance_wsl.go", "build.sh"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func browserInstanceWSLGetRecord(db *sql.DB, agentID, chatID string) (browserInstanceWSLRecord, bool, error) {
	var item browserInstanceWSLRecord
	err := db.QueryRow(
		`SELECT agent_id, chat_id, pid, port, ws, http, user_data_dir FROM `+browserInstanceWSLTableName+` WHERE agent_id = ? AND chat_id = ?`,
		agentID,
		chatID,
	).Scan(&item.AgentID, &item.ChatID, &item.PID, &item.Port, &item.WS, &item.HTTP, &item.UserDataDir)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return browserInstanceWSLRecord{}, false, nil
		}
		return browserInstanceWSLRecord{}, false, fmt.Errorf("query browser_data failed: %w", err)
	}
	return item, true, nil
}

func browserInstanceWSLDeleteRecord(db *sql.DB, agentID, chatID string) error {
	if _, err := db.Exec(
		`DELETE FROM `+browserInstanceWSLTableName+` WHERE agent_id = ? AND chat_id = ?`,
		agentID,
		chatID,
	); err != nil {
		return fmt.Errorf("delete stale browser_data record failed: %w", err)
	}
	return nil
}

func browserInstanceWSLUpsertRecord(db *sql.DB, item browserInstanceWSLRecord) error {
	if _, err := db.Exec(
		`INSERT INTO `+browserInstanceWSLTableName+` (agent_id, chat_id, pid, port, ws, http, user_data_dir, updated_at)
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

func browserInstanceWSLProbeRecord(item browserInstanceWSLRecord) (browserInstanceWSLRecord, bool, error) {
	if item.Port <= 0 {
		return browserInstanceWSLRecord{}, false, errors.New("stored port is invalid")
	}
	ws, httpBase, err := browserInstanceWSLProbeCDP(item.Port)
	if err != nil {
		return browserInstanceWSLRecord{}, false, err
	}
	pid := item.PID
	if livePID, live := browserInstanceWSLLookupPIDByPort(item.Port); live {
		pid = livePID
	}
	item.PID = pid
	item.WS = ws
	item.HTTP = httpBase
	if strings.TrimSpace(item.UserDataDir) == "" {
		if userDataDir, ok := browserInstanceWSLLookupUserDataDirByPID(pid); ok {
			item.UserDataDir = userDataDir
		}
	}
	return item, true, nil
}

func browserInstanceWSLReserveProfileDir() (string, string, error) {
	if err := os.MkdirAll(browserInstanceWSLProfileRoot, 0o755); err != nil {
		return "", "", fmt.Errorf("create profile root failed: %w", err)
	}

	for attempt := 0; attempt < 256; attempt++ {
		suffix, err := browserInstanceWSLRandomSuffix(4)
		if err != nil {
			return "", "", err
		}
		dirName := "chrome_" + suffix
		profileUnix := filepath.Join(browserInstanceWSLProfileRoot, dirName)
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
		return browserInstanceWSLProfileRootW + `\` + dirName, profileUnix, nil
	}
	return "", "", errors.New("allocate unique chrome user-data-dir failed")
}

func browserInstanceWSLStartChrome(chromePath string, port int, profileWin string, headless bool) (int, error) {
	chromePath = browserInstanceWSLResolveChromePath(chromePath)
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

func browserInstanceWSLResolveChromePath(chromePath string) string {
	chromePath = strings.TrimSpace(chromePath)
	if chromePath == "" {
		return browserInstanceWSLChromePath
	}
	return chromePath
}

func browserInstanceWSLWaitForReady(agentID, chatID string, expectedPort int, profileWin, profileUnix string, launchPID int) (browserInstanceWSLRecord, error) {
	deadline := time.Now().Add(browserInstanceWSLTimeout)
	var lastErr error

	for {
		port := expectedPort
		if port <= 0 {
			var err error
			port, err = browserInstanceWSLReadDevToolsPort(profileUnix)
			if err != nil {
				lastErr = err
				if time.Now().After(deadline) {
					break
				}
				time.Sleep(browserInstanceWSLPollInterval)
				continue
			}
		}
		if port > 0 {
			ws, httpBase, probeErr := browserInstanceWSLProbeCDP(port)
			if probeErr == nil {
				pid := launchPID
				if livePID, ok := browserInstanceWSLLookupPIDByPort(port); ok {
					pid = livePID
				}
				return browserInstanceWSLRecord{
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
		time.Sleep(browserInstanceWSLPollInterval)
	}

	if lastErr == nil {
		lastErr = errors.New("cdp not ready after 30s")
	}
	return browserInstanceWSLRecord{}, lastErr
}

func browserInstanceWSLWaitForPortRelease(port int, timeout time.Duration) error {
	if port <= 0 {
		return nil
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	deadline := time.Now().Add(timeout)
	for {
		if _, ok := browserInstanceWSLLookupPIDByPort(port); !ok {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("port %d did not release before restart", port)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func browserInstanceWSLReadDevToolsPort(profileUnix string) (int, error) {
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

func browserInstanceWSLProbeCDP(port int) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("http://localhost:%d/json/version", port)
	cmd := exec.CommandContext(ctx, "curl", "-s", url)
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("curl check failed for port %d: %w", port, err)
	}

	var version browserInstanceWSLVersion
	if err := json.Unmarshal(output, &version); err != nil {
		return "", "", fmt.Errorf("parse /json/version failed for port %d: %w", port, err)
	}
	ws := strings.TrimSpace(version.WebSocketDebuggerURL)
	if ws == "" {
		return "", "", fmt.Errorf("empty webSocketDebuggerUrl for port %d", port)
	}
	return ws, fmt.Sprintf("http://localhost:%d", port), nil
}

func browserInstanceWSLLookupPIDByPort(port int) (int, bool) {
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

func browserInstanceWSLLookupUserDataDirByPID(pid int) (string, bool) {
	if pid <= 0 {
		return "", false
	}

	if userDataDir, ok := browserInstanceWSLLookupUserDataDirFromProc(pid); ok {
		return userDataDir, true
	}
	if userDataDir, ok := browserInstanceWSLLookupUserDataDirFromPowerShell(pid); ok {
		return userDataDir, true
	}
	return "", false
}

func browserInstanceWSLLookupUserDataDirFromProc(pid int) (string, bool) {
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

func browserInstanceWSLLookupUserDataDirFromPowerShell(pid int) (string, bool) {
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
		"/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe",
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

func browserInstanceWSLCleanupProfileLocks(profileUnix string) error {
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
		if info == nil {
			return nil
		}
		if path == profileUnix {
			return nil
		}
		if !browserInstanceWSLShouldRemoveLockEntry(info.Name()) {
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

func browserInstanceWSLShouldRemoveLockEntry(name string) bool {
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

func browserInstanceWSLRemoveProfile(profileWin string) error {
	profileWin = strings.TrimSpace(profileWin)
	if profileWin == "" {
		return nil
	}
	profileUnix, err := browserInstanceWSLWindowsToUnixPath(profileWin)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(profileUnix); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func browserInstanceWSLWindowsToUnixPath(path string) (string, error) {
	output, err := exec.Command("wslpath", "-u", path).Output()
	if err != nil {
		return "", fmt.Errorf("wslpath -u %q failed: %w", path, err)
	}
	converted := strings.TrimSpace(string(output))
	if converted == "" {
		return "", fmt.Errorf("wslpath -u %q returned empty path", path)
	}
	return converted, nil
}

func browserInstanceWSLNormalizeIdentity(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func browserInstanceWSLRandomSuffix(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("random suffix length must be positive")
	}
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	data := make([]byte, length)
	for idx := range data {
		value, err := browserInstanceWSLRandomInt(len(alphabet))
		if err != nil {
			return "", fmt.Errorf("generate random suffix failed: %w", err)
		}
		data[idx] = alphabet[value]
	}
	return string(data), nil
}

func browserInstanceWSLRandomInt(max int) (int, error) {
	if max <= 0 {
		return 0, fmt.Errorf("invalid random max: %d", max)
	}
	value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(value.Int64()), nil
}

func browserInstanceWSLIsWSL() (bool, error) {
	if runtime.GOOS != "linux" {
		return false, nil
	}
	for _, key := range []string{"WSL_DISTRO_NAME", "WSL_INTEROP"} {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return true, nil
		}
	}
	for _, candidate := range []string{"/proc/sys/kernel/osrelease", "/proc/version"} {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		text := strings.ToLower(strings.TrimSpace(string(data)))
		if strings.Contains(text, "microsoft") || strings.Contains(text, "wsl") {
			return true, nil
		}
	}
	return false, nil
}
