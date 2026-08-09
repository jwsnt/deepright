package instance

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"connect/connectsvc"
	"connect/remote/runtimecfg"
)

const (
	stateFileName          = "remote.json"
	logFileName            = "remote.log"
	runtimeDirName         = ".remote"
	daemonProtocolVersion  = "remote-daemon-v1"
	defaultRemotePort      = 22
	defaultStartupTimeout  = 12 * time.Second
	defaultSocketTimeout   = 5 * time.Second
	defaultCreateTimeout   = defaultStartupTimeout + defaultSocketTimeout
	defaultShutdownTimeout = 5 * time.Second
	defaultIdleTimeout     = 15 * time.Minute
	defaultIdleCheckEvery  = time.Minute
	maxUnixPathLength      = 96
	shortRuntimeRoot       = "/tmp"
)

type Record struct {
	AgentID string `json:"agentId"`
	ChatID  string `json:"chatId"`
	Port    int    `json:"port"`
	PID     int    `json:"pid"`
	SSH     string `json:"ssh"`
}

type stateRecord struct {
	AgentID      string `json:"agentId"`
	ChatID       string `json:"chatId"`
	Port         int    `json:"port"`
	PID          int    `json:"pid"`
	SSH          string `json:"ssh"`
	LastActiveAt string `json:"lastActiveAt,omitempty"`
}

type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type execFingerprint struct {
	Path            string `json:"path"`
	Size            int64  `json:"size"`
	ModTimeUnixNano int64  `json:"modTimeUnixNano"`
}

type daemonRequest struct {
	Action           string `json:"action"`
	Command          string `json:"command,omitempty"`
	TimeoutMillisecs int    `json:"timeoutMilliseconds,omitempty"`
}

type authConfig struct {
	Password        string
	CertificatePath string
}

type daemonResponse struct {
	OK          bool            `json:"ok"`
	Error       string          `json:"error,omitempty"`
	Stdout      string          `json:"stdout,omitempty"`
	Stderr      string          `json:"stderr,omitempty"`
	ExitCode    int             `json:"exitCode,omitempty"`
	PID         int             `json:"pid,omitempty"`
	Version     string          `json:"version,omitempty"`
	Fingerprint execFingerprint `json:"fingerprint,omitempty"`
	Record      Record          `json:"record,omitempty"`
}

var (
	executablePathFn      = os.Executable
	statFn                = os.Stat
	readFileFn            = os.ReadFile
	writeFileFn           = os.WriteFile
	removeFn              = os.Remove
	removeAllFn           = os.RemoveAll
	mkdirAllFn            = os.MkdirAll
	startDetachedDaemonFn = startDetachedDaemon
	processExistsFn       = processExists
	terminateProcessFn    = terminateProcess
	cleanupRuntimeFn      = cleanupRuntime
	pingDaemonFn          = pingDaemon
	daemonRequestFn       = daemonRequestCall
	checkSSHMasterFn      = checkSSHMaster
	exitSSHMasterFn       = exitSSHMaster
	nowFn                 = time.Now
)

func Create(flags map[string]string) (Record, error) {
	agentID, chatID, err := requiredIdentity(flags)
	if err != nil {
		return Record{}, err
	}
	sshTarget := normalizeRemoteTarget(connectsvc.FirstValue(flags, "remote"))
	if sshTarget == "" {
		return Record{}, errors.New("remote is required")
	}
	statePath, err := resolveStatePath(flags)
	if err != nil {
		return Record{}, err
	}
	items, err := loadStateRecords(statePath)
	if err != nil {
		return Record{}, err
	}
	items, err = filterLive(flags, items)
	if err != nil {
		return Record{}, err
	}
	if idx := findStateRecord(items, agentID, chatID, sshTarget); idx >= 0 {
		items[idx].touchActivity()
		if err := saveStateRecords(statePath, items); err != nil {
			return Record{}, err
		}
		return items[idx].apiRecord(), nil
	}
	auth, err := authFromFlags(flags)
	if err != nil {
		return Record{}, err
	}
	port, err := connectsvc.IntValue(flags, "port", defaultRemotePort)
	if err != nil {
		return Record{}, err
	}
	if port <= 0 {
		return Record{}, errors.New("port must be greater than zero")
	}

	socketPath, err := resolveSocketPath(flags, agentID, chatID, sshTarget)
	if err != nil {
		return Record{}, err
	}
	controlPath, err := resolveControlPath(flags, agentID, chatID, sshTarget)
	if err != nil {
		return Record{}, err
	}
	secretFile, err := writeSecretFile(flags, agentID, chatID, sshTarget, auth.Password)
	if err != nil {
		return Record{}, err
	}

	selfPath, err := resolveSelfPath(flags)
	if err != nil {
		return Record{}, err
	}
	logPath, err := resolveLogPath(flags)
	if err != nil {
		return Record{}, err
	}
	sshBinary, err := resolveSSHBinary(flags)
	if err != nil {
		return Record{}, err
	}

	pid, err := startDetachedDaemonFn(selfPath, detachedDaemonArgs(statePath, logPath, sshBinary, secretFile, socketPath, controlPath, agentID, chatID, sshTarget, port, auth.CertificatePath))
	if err != nil {
		return Record{}, err
	}

	record := stateRecord{
		AgentID:      agentID,
		ChatID:       chatID,
		Port:         port,
		PID:          pid,
		SSH:          sshTarget,
		LastActiveAt: formatActivityTime(nowFn()),
	}
	if err := waitForDaemon(flags, record.apiRecord()); err != nil {
		_ = cleanupRuntimeFn(flags, record.apiRecord(), false)
		return Record{}, err
	}

	items = append(items, record)
	sortStateRecords(items)
	if err := saveStateRecords(statePath, items); err != nil {
		_ = cleanupRuntimeFn(flags, record.apiRecord(), false)
		return Record{}, err
	}
	return record.apiRecord(), nil
}

func Shutdown(flags map[string]string) error {
	agentID, chatID, err := requiredIdentity(flags)
	if err != nil {
		return err
	}
	sshTarget := normalizeRemoteTarget(connectsvc.FirstValue(flags, "remote"))
	statePath, err := resolveStatePath(flags)
	if err != nil {
		return err
	}
	items, err := loadStateRecords(statePath)
	if err != nil {
		return err
	}
	matchIndexes := findStateRecordIndexes(items, agentID, chatID, sshTarget)
	if len(matchIndexes) == 0 {
		return fmt.Errorf("remote session not found: %s", describeSession(agentID, chatID, sshTarget))
	}
	remaining := make([]stateRecord, 0, len(items)-len(matchIndexes))
	toShutdown := make(map[int]struct{}, len(matchIndexes))
	for _, idx := range matchIndexes {
		toShutdown[idx] = struct{}{}
	}
	for idx, item := range items {
		if _, ok := toShutdown[idx]; ok {
			if err := cleanupRuntimeFn(flags, item.apiRecord(), true); err != nil {
				return err
			}
			continue
		}
		remaining = append(remaining, item)
	}
	return saveStateRecords(statePath, remaining)
}

func List(flags map[string]string) ([]Record, error) {
	statePath, err := resolveStatePath(flags)
	if err != nil {
		return nil, err
	}
	items, err := loadStateRecords(statePath)
	if err != nil {
		return nil, err
	}
	live, err := filterLive(flags, items)
	if err != nil {
		return nil, err
	}
	if err := saveStateRecords(statePath, live); err != nil {
		return nil, err
	}
	return apiRecords(live), nil
}

func Get(flags map[string]string) (Record, error) {
	agentID, chatID, err := requiredIdentity(flags)
	if err != nil {
		return Record{}, err
	}
	sshTarget := normalizeRemoteTarget(connectsvc.FirstValue(flags, "remote"))
	statePath, err := resolveStatePath(flags)
	if err != nil {
		return Record{}, err
	}
	items, err := loadStateRecords(statePath)
	if err != nil {
		return Record{}, err
	}
	live, err := filterLive(flags, items)
	if err != nil {
		return Record{}, err
	}
	matchIndexes := findStateRecordIndexes(live, agentID, chatID, sshTarget)
	if len(matchIndexes) == 0 {
		return Record{}, fmt.Errorf("remote session not found: %s", describeSession(agentID, chatID, sshTarget))
	}
	if len(matchIndexes) > 1 && sshTarget == "" {
		return Record{}, fmt.Errorf("multiple remote sessions found for %s; remote is required", describeSession(agentID, chatID, ""))
	}
	idx := matchIndexes[0]
	live[idx].touchActivity()
	if err := saveStateRecords(statePath, live); err != nil {
		return Record{}, err
	}
	return live[idx].apiRecord(), nil
}

func Exec(flags map[string]string, remoteCommand string) (CommandResult, error) {
	record, err := ResolveSessionRecord(flags)
	if err != nil {
		return CommandResult{}, err
	}
	socketPath, err := resolveSocketPath(flags, record.AgentID, record.ChatID, record.SSH)
	if err != nil {
		return CommandResult{}, err
	}
	timeout := resolveExecTimeout(flags)
	response, err := daemonRequestFn(socketPath, daemonRequest{
		Action:           "exec",
		Command:          remoteCommand,
		TimeoutMillisecs: int(timeout / time.Millisecond),
	}, timeout+defaultSocketTimeout)
	if err != nil {
		return CommandResult{}, err
	}
	if !response.OK {
		if response.Error == "" {
			response.Error = "remote exec failed"
		}
		return CommandResult{}, errors.New(response.Error)
	}
	return CommandResult{
		Stdout:   response.Stdout,
		Stderr:   response.Stderr,
		ExitCode: response.ExitCode,
	}, nil
}

func resolveExecTimeout(flags map[string]string) time.Duration {
	if value, ok := runtimecfg.ParseTimeoutMS(connectsvc.FirstValue(flags, "timeout")); ok {
		return value
	}
	if value, ok := runtimecfg.ParseTimeoutMS(connectsvc.FirstValue(flags, "exec-timeout-ms")); ok {
		return value
	}
	return runtimecfg.ResolvePluginTimeouts(flags).Exec
}

func ResolveSessionRecord(flags map[string]string) (Record, error) {
	session := normalizeSession(connectsvc.FirstValue(flags, "session"))
	if session == "" {
		return Record{}, errors.New("session is required")
	}
	parts := strings.SplitN(session, "@", 2)
	if len(parts) != 2 {
		return Record{}, errors.New("session must use agentId@chatId")
	}
	return Get(map[string]string{
		"agentId":     parts[0],
		"chatId":      parts[1],
		"remote":      connectsvc.FirstValue(flags, "remote"),
		"state":       connectsvc.FirstValue(flags, "state"),
		"log":         connectsvc.FirstValue(flags, "log"),
		"runtime-dir": connectsvc.FirstValue(flags, "runtime-dir"),
		"self":        connectsvc.FirstValue(flags, "self"),
		"ssh-bin":     connectsvc.FirstValue(flags, "ssh-bin"),
	})
}

func ResolveSessionControlPath(flags map[string]string) (Record, string, error) {
	record, err := ResolveSessionRecord(flags)
	if err != nil {
		return Record{}, "", err
	}
	controlPath, err := resolveControlPath(flags, record.AgentID, record.ChatID, record.SSH)
	if err != nil {
		return Record{}, "", err
	}
	if err := checkSSHMaster(resolveSSHBinaryNoFail(flags), controlPath, record.Port, record.SSH); err != nil {
		return Record{}, "", fmt.Errorf("remote session not available: %s@%s", record.AgentID, record.ChatID)
	}
	return record, controlPath, nil
}

func requiredIdentity(flags map[string]string) (string, string, error) {
	agentID := normalizeIdentityPart(connectsvc.FirstValue(flags, "agentId", "agent"))
	chatID := normalizeIdentityPart(connectsvc.FirstValue(flags, "chatId", "chat"))
	if agentID == "" {
		return "", "", errors.New("agentId is required")
	}
	if chatID == "" {
		return "", "", errors.New("chatId is required")
	}
	return agentID, chatID, nil
}

func normalizeIdentityPart(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeRemoteTarget(value string) string {
	return strings.TrimSpace(value)
}

func normalizeSession(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.SplitN(value, "@", 2)
	if len(parts) != 2 {
		return strings.ToLower(value)
	}
	agentID := normalizeIdentityPart(parts[0])
	chatID := normalizeIdentityPart(parts[1])
	if agentID == "" || chatID == "" {
		return strings.ToLower(value)
	}
	return agentID + "@" + chatID
}

func waitForDaemon(flags map[string]string, record Record) error {
	deadline := nowFn().Add(defaultStartupTimeout)
	for nowFn().Before(deadline) {
		if isRecordLive(flags, record) {
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	return fmt.Errorf("remote daemon start timeout for %s@%s; see remote.log", record.AgentID, record.ChatID)
}

func filterLive(flags map[string]string, items []stateRecord) ([]stateRecord, error) {
	live := make([]stateRecord, 0, len(items))
	for _, item := range items {
		if isRecordLive(flags, item.apiRecord()) {
			live = append(live, item)
			continue
		}
		_ = cleanupRuntimeFn(flags, item.apiRecord(), false)
	}
	sortStateRecords(live)
	return live, nil
}

func isRecordLive(flags map[string]string, record Record) bool {
	if record.PID <= 0 || !processExistsFn(record.PID) {
		return false
	}
	socketPath, err := resolveSocketPath(flags, record.AgentID, record.ChatID, record.SSH)
	if err != nil {
		return false
	}
	response, err := pingDaemonFn(socketPath, defaultSocketTimeout)
	if err != nil || !response.OK {
		return false
	}
	currentFingerprint, err := currentFingerprint(flags)
	if err != nil {
		return false
	}
	if response.Version != daemonProtocolVersion {
		return false
	}
	if response.PID != record.PID {
		return false
	}
	if response.Record.AgentID != record.AgentID || response.Record.ChatID != record.ChatID || response.Record.Port != record.Port || response.Record.SSH != record.SSH {
		return false
	}
	return response.Fingerprint == currentFingerprint
}

func findRecord(items []Record, agentID, chatID, sshTarget string) int {
	for i, item := range items {
		if item.AgentID == agentID && item.ChatID == chatID && (sshTarget == "" || normalizeRemoteTarget(item.SSH) == sshTarget) {
			return i
		}
	}
	return -1
}

func sortRecords(items []Record) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].AgentID != items[j].AgentID {
			return items[i].AgentID < items[j].AgentID
		}
		if items[i].ChatID != items[j].ChatID {
			return items[i].ChatID < items[j].ChatID
		}
		return items[i].SSH < items[j].SSH
	})
}

func findStateRecord(items []stateRecord, agentID, chatID, sshTarget string) int {
	matchIndexes := findStateRecordIndexes(items, agentID, chatID, sshTarget)
	if len(matchIndexes) == 0 {
		return -1
	}
	return matchIndexes[0]
}

func findStateRecordIndexes(items []stateRecord, agentID, chatID, sshTarget string) []int {
	matchIndexes := make([]int, 0, len(items))
	for i, item := range items {
		if item.AgentID != agentID || item.ChatID != chatID {
			continue
		}
		if sshTarget != "" && normalizeRemoteTarget(item.SSH) != sshTarget {
			continue
		}
		matchIndexes = append(matchIndexes, i)
	}
	return matchIndexes
}

func sortStateRecords(items []stateRecord) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].AgentID != items[j].AgentID {
			return items[i].AgentID < items[j].AgentID
		}
		if items[i].ChatID != items[j].ChatID {
			return items[i].ChatID < items[j].ChatID
		}
		return items[i].SSH < items[j].SSH
	})
}

func loadState(path string) ([]Record, error) {
	items, err := loadStateRecords(path)
	if err != nil {
		return nil, err
	}
	return apiRecords(items), nil
}

func loadStateRecords(path string) ([]stateRecord, error) {
	data, err := readFileFn(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []stateRecord{}, nil
		}
		return nil, err
	}
	if strings.TrimSpace(string(data)) == "" {
		return []stateRecord{}, nil
	}
	var items []stateRecord
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func saveState(path string, items []Record) error {
	stateItems := make([]stateRecord, 0, len(items))
	for _, item := range items {
		stateItems = append(stateItems, stateRecord{
			AgentID: item.AgentID,
			ChatID:  item.ChatID,
			Port:    item.Port,
			PID:     item.PID,
			SSH:     item.SSH,
		})
	}
	return saveStateRecords(path, stateItems)
}

func saveStateRecords(path string, items []stateRecord) error {
	if err := mkdirAllFn(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	sortStateRecords(items)
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return writeFileFn(path, append(data, '\n'), 0o644)
}

func apiRecords(items []stateRecord) []Record {
	out := make([]Record, 0, len(items))
	for _, item := range items {
		out = append(out, item.apiRecord())
	}
	return out
}

func currentFingerprint(flags map[string]string) (execFingerprint, error) {
	selfPath, err := resolveSelfPath(flags)
	if err != nil {
		return execFingerprint{}, err
	}
	info, err := statFn(selfPath)
	if err != nil {
		return execFingerprint{}, err
	}
	return execFingerprint{
		Path:            selfPath,
		Size:            info.Size(),
		ModTimeUnixNano: info.ModTime().UnixNano(),
	}, nil
}

func resolveSelfPath(flags map[string]string) (string, error) {
	if explicit := strings.TrimSpace(connectsvc.FirstValue(flags, "self")); explicit != "" {
		return filepath.Abs(explicit)
	}
	exe, err := executablePathFn()
	if err != nil {
		return "", err
	}
	return filepath.Abs(exe)
}

func resolveStatePath(flags map[string]string) (string, error) {
	if explicit := strings.TrimSpace(connectsvc.FirstValue(flags, "state")); explicit != "" {
		return filepath.Abs(explicit)
	}
	runtime, err := runtimePaths(flags)
	if err != nil {
		return "", err
	}
	return runtime.StatePath, nil
}

func resolveLogPath(flags map[string]string) (string, error) {
	if explicit := strings.TrimSpace(connectsvc.FirstValue(flags, "log")); explicit != "" {
		return filepath.Abs(explicit)
	}
	runtime, err := runtimePaths(flags)
	if err != nil {
		return "", err
	}
	return runtime.LogPath, nil
}

func resolveRuntimeDir(flags map[string]string) (string, error) {
	if explicit := strings.TrimSpace(connectsvc.FirstValue(flags, "runtime-dir")); explicit != "" {
		return filepath.Abs(explicit)
	}
	runtime, err := runtimePaths(flags)
	if err != nil {
		return "", err
	}
	return runtime.RuntimeDir, nil
}

func resolveSocketPath(flags map[string]string, agentID, chatID, sshTarget string) (string, error) {
	runtimeDir, err := resolveTransportRuntimeDir(flags)
	if err != nil {
		return "", err
	}
	return filepath.Join(runtimeDir, "sock-"+sessionFileKey(agentID, chatID, sshTarget)+".sock"), nil
}

func resolveControlPath(flags map[string]string, agentID, chatID, sshTarget string) (string, error) {
	runtimeDir, err := resolveTransportRuntimeDir(flags)
	if err != nil {
		return "", err
	}
	return filepath.Join(runtimeDir, "ctl-"+sessionFileKey(agentID, chatID, sshTarget)), nil
}

func resolveTransportRuntimeDir(flags map[string]string) (string, error) {
	runtimeDir, err := resolveRuntimeDir(flags)
	if err != nil {
		return "", err
	}

	candidates := []string{
		runtimeDir,
		filepath.Join(shortRuntimeRoot, "remote-"+shortHash(runtimeDir)),
		filepath.Join(os.TempDir(), "remote-"+shortHash(runtimeDir)),
	}
	for _, candidate := range candidates {
		if transportRuntimeDirFits(candidate) {
			return candidate, nil
		}
	}
	return candidates[1], nil
}

func transportRuntimeDirFits(runtimeDir string) bool {
	keyProbe := strings.Repeat("f", 16)
	socketProbe := filepath.Join(runtimeDir, "sock-"+keyProbe+".sock")
	controlProbe := filepath.Join(runtimeDir, "ctl-"+keyProbe+"."+keyProbe)
	return len(socketProbe) <= maxUnixPathLength && len(controlProbe) <= maxUnixPathLength
}

func resolveSSHBinary(flags map[string]string) (string, error) {
	if explicit := strings.TrimSpace(connectsvc.FirstValue(flags, "ssh-bin")); explicit != "" {
		return filepath.Abs(explicit)
	}
	return exec.LookPath("ssh")
}

func writeSecretFile(flags map[string]string, agentID, chatID, sshTarget, password string) (string, error) {
	runtimeDir, err := resolveRuntimeDir(flags)
	if err != nil {
		return "", err
	}
	if err := mkdirAllFn(runtimeDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(runtimeDir, "secret-"+sessionFileKey(agentID, chatID, sshTarget))
	return path, writeFileFn(path, []byte(password), 0o600)
}

func sessionFileKey(agentID, chatID, sshTarget string) string {
	return shortHash(agentID + "@" + chatID + "@" + normalizeRemoteTarget(sshTarget))
}

func shortHash(value string) string {
	const table = "0123456789abcdef"
	sum := 1469598103934665603 ^ uint64(len(value))
	for i := 0; i < len(value); i++ {
		sum ^= uint64(value[i])
		sum *= 1099511628211
	}
	out := make([]byte, 16)
	for i := range out {
		out[len(out)-1-i] = table[sum&0x0f]
		sum >>= 4
	}
	return string(out)
}

func detachedDaemonArgs(statePath, logPath, sshBinary, secretFile, socketPath, controlPath, agentID, chatID, sshTarget string, port int, certificatePath string) []string {
	return []string{
		"__daemon",
		"--agentId", agentID,
		"--chatId", chatID,
		"--remote", sshTarget,
		"--port", fmt.Sprintf("%d", port),
		"--state", statePath,
		"--log", logPath,
		"--ssh-bin", sshBinary,
		"--secret-file", secretFile,
		"--socket", socketPath,
		"--control", controlPath,
		"--certificate", certificatePath,
	}
}

func cleanupRuntime(flags map[string]string, record Record, requestShutdown bool) error {
	socketPath, err := resolveSocketPath(flags, record.AgentID, record.ChatID, record.SSH)
	if err == nil && requestShutdown {
		_, _ = daemonRequestFn(socketPath, daemonRequest{Action: "shutdown"}, defaultShutdownTimeout)
	}
	controlPath, controlErr := resolveControlPath(flags, record.AgentID, record.ChatID, record.SSH)
	if controlErr == nil {
		_ = exitSSHMasterFn(resolveSSHBinaryNoFail(flags), controlPath, record.Port, record.SSH)
		_ = removeFn(controlPath)
	}
	if err == nil {
		_ = removeFn(socketPath)
	}
	if processExistsFn(record.PID) {
		_ = terminateProcessFn(record.PID)
	}
	return nil
}

func describeSession(agentID, chatID, sshTarget string) string {
	if normalizeRemoteTarget(sshTarget) == "" {
		return agentID + "@" + chatID
	}
	return agentID + "@" + chatID + " (" + normalizeRemoteTarget(sshTarget) + ")"
}

func resolveSSHBinaryNoFail(flags map[string]string) string {
	sshBinary, err := resolveSSHBinary(flags)
	if err != nil {
		return "ssh"
	}
	return sshBinary
}

func daemonRequestCall(socketPath string, req daemonRequest, timeout time.Duration) (daemonResponse, error) {
	conn, err := net.DialTimeout("unix", socketPath, timeout)
	if err != nil {
		return daemonResponse{}, err
	}
	defer conn.Close()
	if timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(timeout))
	}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return daemonResponse{}, err
	}
	var response daemonResponse
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return daemonResponse{}, err
	}
	return response, nil
}

func pingDaemon(socketPath string, timeout time.Duration) (daemonResponse, error) {
	return daemonRequestCall(socketPath, daemonRequest{Action: "ping"}, timeout)
}

func checkSSHMaster(sshBinary, controlPath string, port int, sshTarget string) error {
	cmd := exec.Command(sshBinary, "-S", controlPath, "-p", fmt.Sprintf("%d", port), "-O", "check", sshTarget)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func exitSSHMaster(sshBinary, controlPath string, port int, sshTarget string) error {
	cmd := exec.Command(sshBinary, "-S", controlPath, "-p", fmt.Sprintf("%d", port), "-O", "exit", sshTarget)
	return cmd.Run()
}

func openLogWriter(path string) (io.WriteCloser, error) {
	if err := mkdirAllFn(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
}

func authFromFlags(flags map[string]string) (authConfig, error) {
	password := strings.TrimSpace(connectsvc.FirstValue(flags, "password"))
	certificatePath := strings.TrimSpace(connectsvc.FirstValue(flags, "certificate"))
	if password == "" && certificatePath == "" {
		return authConfig{}, errors.New("password or certificate is required")
	}
	if certificatePath != "" {
		abs, err := filepath.Abs(certificatePath)
		if err != nil {
			return authConfig{}, err
		}
		info, err := statFn(abs)
		if err != nil {
			return authConfig{}, err
		}
		if info.IsDir() {
			return authConfig{}, fmt.Errorf("certificate must be a file: %s", abs)
		}
		certificatePath = abs
	}
	return authConfig{
		Password:        password,
		CertificatePath: certificatePath,
	}, nil
}

func (item stateRecord) apiRecord() Record {
	return Record{
		AgentID: item.AgentID,
		ChatID:  item.ChatID,
		Port:    item.Port,
		PID:     item.PID,
		SSH:     item.SSH,
	}
}

func (item *stateRecord) touchActivity() {
	if item == nil {
		return
	}
	item.LastActiveAt = formatActivityTime(nowFn())
}

func formatActivityTime(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.UTC().Format(time.RFC3339Nano)
}

func parseActivityTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func pruneIdleSessions(flags map[string]string, idleTimeout time.Duration, logger *log.Logger) (int, error) {
	statePath, err := resolveStatePath(flags)
	if err != nil {
		return 0, err
	}
	items, err := loadStateRecords(statePath)
	if err != nil {
		return 0, err
	}
	live, err := filterLive(flags, items)
	if err != nil {
		return 0, err
	}
	now := nowFn()
	kept := make([]stateRecord, 0, len(live))
	closed := 0
	changed := false
	for _, item := range live {
		lastActive, ok := parseActivityTime(item.LastActiveAt)
		if !ok {
			item.LastActiveAt = formatActivityTime(now)
			kept = append(kept, item)
			changed = true
			continue
		}
		idleFor := now.Sub(lastActive)
		if idleFor < idleTimeout {
			kept = append(kept, item)
			continue
		}
		if err := cleanupRuntimeFn(flags, item.apiRecord(), true); err != nil && logger != nil {
			logger.Printf("用参数：Agent=%s；会话=%s；进程=%d；远程地址=%s；原因=%v，做了：空闲太久后尝试关闭远程连接失败", item.AgentID, item.ChatID, item.PID, item.SSH, err)
		} else if logger != nil {
			logger.Printf("用参数：Agent=%s；会话=%s；进程=%d；远程地址=%s；空闲时长=%s，做了：自动关闭了长时间没活动的远程连接", item.AgentID, item.ChatID, item.PID, item.SSH, idleFor.Truncate(time.Second))
		}
		closed++
		changed = true
	}
	if changed {
		if err := saveStateRecords(statePath, kept); err != nil {
			return closed, err
		}
	}
	return closed, nil
}
