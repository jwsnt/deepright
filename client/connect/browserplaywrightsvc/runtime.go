package browserplaywrightsvc

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func normalizeOptions(opts Options) (Options, error) {
	stateDir := strings.TrimSpace(opts.StateDir)
	if stateDir == "" {
		stateDir = defaultStateDir
	}
	absState, err := filepath.Abs(stateDir)
	if err != nil {
		return Options{}, err
	}
	if err := os.MkdirAll(absState, 0o755); err != nil {
		return Options{}, err
	}

	addr := strings.TrimSpace(opts.Addr)
	if addr == "" {
		addr = defaultAddr
	}
	logFile := strings.TrimSpace(opts.LogFile)
	if logFile == "" {
		defaultPath, err := defaultLogFilePath(absState, opts.ExecutablePath)
		if err != nil {
			return Options{}, err
		}
		logFile = defaultPath
	}
	pidFile := strings.TrimSpace(opts.PIDFile)
	if pidFile == "" {
		pidFile = filepath.Join(absState, defaultPIDFileName)
	}
	driverDir := strings.TrimSpace(opts.DriverDir)
	if driverDir == "" {
		defaultDriverDir, err := defaultDriverDirectory(opts.ExecutablePath)
		if err != nil {
			return Options{}, err
		}
		driverDir = defaultDriverDir
	} else {
		driverDir, err = filepath.Abs(driverDir)
		if err != nil {
			return Options{}, err
		}
	}
	return Options{
		StateDir:       absState,
		Addr:           addr,
		LogFile:        logFile,
		PIDFile:        pidFile,
		DriverDir:      driverDir,
		ExecutablePath: strings.TrimSpace(opts.ExecutablePath),
		BrowserTimeout: normalizeBrowserTimeout(opts.BrowserTimeout),
	}, nil
}

func normalizeBrowserTimeout(value time.Duration) time.Duration {
	if value <= 0 {
		return defaultBrowserTimeout
	}
	return value
}

func metaPath(stateDir string) string {
	return filepath.Join(stateDir, "daemon.json")
}

func timeoutStatePath(stateDir string) string {
	return filepath.Join(stateDir, "command_timeout_state.json")
}

func shutdownReasonPath(stateDir string) string {
	return filepath.Join(stateDir, "shutdown_reason.json")
}

func executableRoot(explicit string) (string, error) {
	exe := strings.TrimSpace(explicit)
	if exe == "" {
		var err error
		exe, err = browserPlaywrightExecutablePathFn()
		if err != nil {
			return "", err
		}
	}
	if abs, err := filepath.Abs(exe); err == nil {
		exe = abs
	}
	return filepath.Dir(exe), nil
}

func defaultLogFilePath(stateDir string, explicitExecutable string) (string, error) {
	root, err := executableRoot(explicitExecutable)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(root) == "" {
		root = stateDir
	}
	name := defaultExecutableLogFileName(explicitExecutable)
	return filepath.Join(root, defaultLogDirName, name), nil
}

func defaultExecutableLogFileName(explicitExecutable string) string {
	exe := strings.TrimSpace(explicitExecutable)
	if exe == "" {
		if resolved, err := browserPlaywrightExecutablePathFn(); err == nil {
			exe = strings.TrimSpace(resolved)
		}
	}
	base := strings.TrimSpace(filepath.Base(exe))
	if base == "" || base == "." || base == string(filepath.Separator) {
		base = DefaultName
	}
	return base + ".log"
}

func defaultDriverDirectory(explicitExecutable string) (string, error) {
	root, err := executableRoot(explicitExecutable)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("resolve executable directory for playwright driver")
	}
	return filepath.Join(root, "playwright", "driver"), nil
}

func sessionRoot(stateDir string) string {
	return filepath.Join(stateDir, "sessions")
}

func sessionDir(stateDir, name string) string {
	return filepath.Join(sessionRoot(stateDir), sanitizeName(name))
}

func sessionMetaPath(stateDir, name string) string {
	return filepath.Join(sessionDir(stateDir, name), "session.json")
}

func snapshotPath(stateDir, name string) string {
	return filepath.Join(sessionDir(stateDir, name), "snapshot.json")
}

func sanitizeName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "default"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_")
	value = replacer.Replace(value)
	return strings.ToLower(value)
}

type daemonMeta struct {
	Addr      string    `json:"addr"`
	PID       int       `json:"pid"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type daemonShutdownReason struct {
	Reason         string            `json:"reason"`
	TargetPID      int               `json:"targetPid,omitempty"`
	RequestedAt    time.Time         `json:"requestedAt"`
	RequestedByPID int               `json:"requestedByPid,omitempty"`
	Details        map[string]string `json:"details,omitempty"`
}

func writeJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func writeDaemonShutdownReason(stateDir, reason string, targetPID int, details map[string]string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil
	}
	copied := map[string]string{}
	for key, value := range details {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		copied[key] = value
	}
	if len(copied) == 0 {
		copied = nil
	}
	return writeJSONFile(shutdownReasonPath(stateDir), daemonShutdownReason{
		Reason:         reason,
		TargetPID:      targetPID,
		RequestedAt:    time.Now(),
		RequestedByPID: os.Getpid(),
		Details:        copied,
	})
}

func consumeDaemonShutdownReason(stateDir string, currentPID int) (daemonShutdownReason, error) {
	path := shutdownReasonPath(stateDir)
	var item daemonShutdownReason
	if err := readJSONFile(path, &item); err != nil {
		return daemonShutdownReason{}, err
	}
	if item.TargetPID > 0 && currentPID > 0 && item.TargetPID != currentPID {
		return daemonShutdownReason{}, os.ErrNotExist
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return daemonShutdownReason{}, err
	}
	return item, nil
}

func clearDaemonShutdownReason(stateDir string) error {
	err := os.Remove(shutdownReasonPath(stateDir))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func daemonShutdownReasonLogFields(item daemonShutdownReason) string {
	parts := []string{}
	if reason := strings.TrimSpace(item.Reason); reason != "" {
		parts = append(parts, "原因="+reason)
	}
	if item.RequestedByPID > 0 {
		parts = append(parts, "来源进程="+strconv.Itoa(item.RequestedByPID))
	}
	if item.TargetPID > 0 {
		parts = append(parts, "目标进程="+strconv.Itoa(item.TargetPID))
	}
	if !item.RequestedAt.IsZero() {
		parts = append(parts, "请求时间="+item.RequestedAt.Format(time.RFC3339Nano))
	}
	if len(item.Details) > 0 {
		keys := make([]string, 0, len(item.Details))
		for key := range item.Details {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		detailParts := make([]string, 0, len(keys))
		for _, key := range keys {
			detailParts = append(detailParts, key+"="+item.Details[key])
		}
		parts = append(parts, "详情="+strings.Join(detailParts, ","))
	}
	return strings.Join(parts, "；")
}

func readJSONFile(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func writePIDFile(path string, pid int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0o644)
}

func readPIDFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}
	return pid, nil
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

func waitForAddr(addr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func waitForDaemonReady(addr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 300 * time.Millisecond}
	for time.Now().Before(deadline) {
		if daemonHealthOK(client, addr) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func daemonHealthOK(client *http.Client, addr string) bool {
	resp, err := client.Get("http://" + addr + "/healthz")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode == http.StatusOK
}

func ensureParentDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

func absChild(root, child string) string {
	child = strings.TrimSpace(child)
	if child == "" {
		return root
	}
	if filepath.IsAbs(child) {
		return child
	}
	return filepath.Join(root, child)
}

func sameSiteString(value any) string {
	switch v := value.(type) {
	case *string:
		if v == nil {
			return ""
		}
		return *v
	case string:
		return v
	default:
		return fmt.Sprint(value)
	}
}
