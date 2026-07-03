package instance

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"connect/connectsvc"
	"connect/remote/runtimecfg"
)

const (
	managerProtocolVersion = "remote-manager-v1"
	managerSocketFileName  = "manager.sock"
	pidFileName            = "remote.pid"
)

type managerRequest struct {
	Action           string            `json:"action"`
	Flags            map[string]string `json:"flags,omitempty"`
	Command          string            `json:"command,omitempty"`
	TimeoutMillisecs int               `json:"timeoutMilliseconds,omitempty"`
}

type managerResponse struct {
	OK          bool     `json:"ok"`
	Error       string   `json:"error,omitempty"`
	Item        Record   `json:"item,omitempty"`
	Items       []Record `json:"items,omitempty"`
	Stdout      string   `json:"stdout,omitempty"`
	Stderr      string   `json:"stderr,omitempty"`
	ExitCode    int      `json:"exitCode,omitempty"`
	PID         int      `json:"pid,omitempty"`
	Version     string   `json:"version,omitempty"`
	Fingerprint string   `json:"fingerprint,omitempty"`
}

var (
	managerRequestFn     = managerRequestCall
	startManagerDaemonFn = startDetachedManager
	readPIDFileFn        = os.ReadFile
	managerExecutableFn  = os.Executable
	managerListenFn      = net.Listen
	managerDialTimeoutFn = net.DialTimeout
)

func StartManager(flags map[string]string) error {
	runtime, err := runtimePaths(flags)
	if err != nil {
		return err
	}
	logger, closeLogger := commandLogger(runtime.LogPath, "remote-cli ")
	if closeLogger != nil {
		defer closeLogger()
	}
	if logger != nil {
		logger.Printf("用参数：插件=remote，做了：准备启动远程连接管理服务")
	}
	if err := mkdirAllFn(runtime.RootDir, 0o755); err != nil {
		if logger != nil {
			logger.Printf("用参数：插件=remote；原因=%v，做了：启动远程连接管理服务失败", err)
		}
		return err
	}
	if err := ensureStopped(flags); err != nil {
		if logger != nil {
			logger.Printf("用参数：插件=remote；原因=%v，做了：启动远程连接管理服务失败", err)
		}
		return err
	}
	if err := saveState(runtime.StatePath, []Record{}); err != nil {
		if logger != nil {
			logger.Printf("用参数：插件=remote；原因=%v，做了：启动远程连接管理服务失败", err)
		}
		return err
	}
	selfPath, err := resolveSelfPath(flags)
	if err != nil {
		if logger != nil {
			logger.Printf("用参数：插件=remote；原因=%v，做了：启动远程连接管理服务失败", err)
		}
		return err
	}
	pid, err := startManagerDaemonFn(selfPath, []string{
		"__manager",
		"--state", runtime.StatePath,
		"--log", runtime.LogPath,
		"--runtime-dir", runtime.RuntimeDir,
		"--pid-file", runtime.PIDFile,
		"--manager-socket", runtime.ManagerSocket,
		"--self", selfPath,
		"--ssh-bin", connectsvc.FirstValue(flags, "ssh-bin"),
	})
	if err != nil {
		if logger != nil {
			logger.Printf("用参数：插件=remote；原因=%v，做了：启动远程连接管理服务失败", err)
		}
		return err
	}
	deadline := time.Now().Add(defaultStartupTimeout)
	for time.Now().Before(deadline) {
		if ok := isManagerLive(flags); ok {
			if logger != nil {
				logger.Printf("用参数：进程=%d，做了：远程连接管理服务已启动", pid)
			}
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	_ = terminateProcessFn(pid)
	if logger != nil {
		logger.Printf("用参数：进程=%d，做了：远程连接管理服务启动超时", pid)
	}
	return fmt.Errorf("remote manager start timeout; see %s", runtime.LogPath)
}

func StopManager(flags map[string]string) error {
	runtime, err := runtimePaths(flags)
	if err != nil {
		return err
	}
	logger, closeLogger := commandLogger(runtime.LogPath, "remote-cli ")
	if closeLogger != nil {
		defer closeLogger()
	}
	if logger != nil {
		logger.Printf("用参数：插件=remote，做了：准备停止远程连接管理服务")
	}
	_, _ = managerRequestFn(runtime.ManagerSocket, managerRequest{Action: "shutdown"}, defaultShutdownTimeout)
	if pid, ok := readPID(runtime.PIDFile); ok {
		_ = terminateProcessFn(pid)
	}
	items, err := loadState(runtime.StatePath)
	if err == nil {
		for _, item := range items {
			_ = cleanupRuntime(withRuntimeFlags(flags, runtime), item, true)
		}
	}
	_ = removeFn(runtime.PIDFile)
	_ = removeFn(runtime.ManagerSocket)
	if err := saveState(runtime.StatePath, []Record{}); err != nil {
		if logger != nil {
			logger.Printf("用参数：插件=remote；原因=%v，做了：停止远程连接管理服务失败", err)
		}
		return err
	}
	if logger != nil {
		logger.Printf("用参数：插件=remote，做了：远程连接管理服务已停止")
	}
	return nil
}

func CreateViaManager(flags map[string]string) (Record, error) {
	response, err := callManager(flags, managerRequest{Action: "create", Flags: sanitizeUserFlags(flags)}, defaultCreateTimeout)
	if err != nil {
		return Record{}, err
	}
	if !response.OK {
		return Record{}, errors.New(response.Error)
	}
	return response.Item, nil
}

func ShutdownViaManager(flags map[string]string) error {
	response, err := callManager(flags, managerRequest{Action: "shutdown-session", Flags: sanitizeUserFlags(flags)}, defaultSocketTimeout)
	if err != nil {
		return err
	}
	if !response.OK {
		return errors.New(response.Error)
	}
	return nil
}

func ListViaManager(flags map[string]string) ([]Record, error) {
	response, err := callManager(flags, managerRequest{Action: "list", Flags: sanitizeUserFlags(flags)}, defaultSocketTimeout)
	if err != nil {
		return nil, err
	}
	if !response.OK {
		return nil, errors.New(response.Error)
	}
	return response.Items, nil
}

func GetViaManager(flags map[string]string) (Record, error) {
	response, err := callManager(flags, managerRequest{Action: "get", Flags: sanitizeUserFlags(flags)}, defaultSocketTimeout)
	if err != nil {
		return Record{}, err
	}
	if !response.OK {
		return Record{}, errors.New(response.Error)
	}
	return response.Item, nil
}

func ExecViaManager(flags map[string]string, remoteCommand string) (CommandResult, error) {
	timeout := runtimecfg.ResolvePluginTimeouts(flags).Exec
	response, err := callManager(flags, managerRequest{
		Action:           "exec",
		Flags:            sanitizeUserFlags(flags),
		Command:          remoteCommand,
		TimeoutMillisecs: int(timeout / time.Millisecond),
	}, timeout+defaultSocketTimeout)
	if err != nil {
		return CommandResult{}, err
	}
	if !response.OK {
		return CommandResult{}, errors.New(response.Error)
	}
	return CommandResult{
		Stdout:   response.Stdout,
		Stderr:   response.Stderr,
		ExitCode: response.ExitCode,
	}, nil
}

func callManager(flags map[string]string, req managerRequest, timeout time.Duration) (managerResponse, error) {
	if !isManagerLive(flags) {
		if err := StartManager(flags); err != nil {
			return managerResponse{}, err
		}
	}
	runtime, err := runtimePaths(flags)
	if err != nil {
		return managerResponse{}, err
	}
	if timeout <= 0 {
		timeout = defaultSocketTimeout
	}
	response, err := managerRequestFn(runtime.ManagerSocket, req, timeout)
	if err != nil {
		return managerResponse{}, err
	}
	return response, nil
}

func isManagerLive(flags map[string]string) bool {
	runtime, err := runtimePaths(flags)
	if err != nil {
		return false
	}
	pid, ok := readPID(runtime.PIDFile)
	if !ok || !processExistsFn(pid) {
		return false
	}
	response, err := managerRequestFn(runtime.ManagerSocket, managerRequest{Action: "ping"}, defaultSocketTimeout)
	if err != nil {
		return false
	}
	return response.OK && response.PID == pid && response.Version == managerProtocolVersion && response.Fingerprint == fingerprintString(flags)
}

func fingerprintString(flags map[string]string) string {
	value, err := currentFingerprint(flags)
	if err != nil {
		return ""
	}
	data, _ := json.Marshal(value)
	return string(data)
}

func managerRequestCall(socketPath string, req managerRequest, timeout time.Duration) (managerResponse, error) {
	conn, err := managerDialTimeoutFn("unix", socketPath, timeout)
	if err != nil {
		return managerResponse{}, err
	}
	defer conn.Close()
	if timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(timeout))
	}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return managerResponse{}, err
	}
	var response managerResponse
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return managerResponse{}, err
	}
	return response, nil
}

func readPID(path string) (int, bool) {
	data, err := readPIDFileFn(path)
	if err != nil {
		return 0, false
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return 0, false
	}
	value, err := strconv.Atoi(text)
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}

func runtimePaths(flags map[string]string) (Paths, error) {
	root, binaryPath, err := runtimeRoot(flags)
	if err != nil {
		return Paths{}, err
	}
	managerSocketRoot := filepath.Join(root, runtimeDirName)
	managerSocketPath := filepath.Join(managerSocketRoot, managerSocketFileName)
	if len(managerSocketPath) > maxUnixPathLength {
		managerSocketRoot = filepath.Join(os.TempDir(), "remote-manager-"+shortHash(root))
		managerSocketPath = filepath.Join(managerSocketRoot, managerSocketFileName)
	}
	return Paths{
		RootDir:       root,
		BinaryPath:    binaryPath,
		LogPath:       filepath.Join(root, "remote.log"),
		StatePath:     filepath.Join(root, "remote.json"),
		PIDFile:       filepath.Join(root, pidFileName),
		RuntimeDir:    filepath.Join(root, runtimeDirName),
		ManagerSocket: managerSocketPath,
	}, nil
}

type Paths struct {
	RootDir       string
	BinaryPath    string
	LogPath       string
	StatePath     string
	PIDFile       string
	RuntimeDir    string
	ManagerSocket string
}

func runtimeRoot(flags map[string]string) (string, string, error) {
	selfPath, err := resolveSelfPath(flags)
	if err != nil {
		return "", "", err
	}
	if root, ok := integrationPluginsRoot(flags); ok {
		return root, filepath.Join(root, "remote"), nil
	}
	return filepath.Dir(selfPath), selfPath, nil
}

func integrationPluginsRoot(flags map[string]string) (string, bool) {
	connectBin := strings.TrimSpace(connectsvc.FirstValue(flags, "connect-bin"))
	if connectBin == "" {
		return "", false
	}
	runtimePath, ok := resolveIntegrationRuntimePath(connectBin)
	if !ok {
		return "", false
	}
	cfg, err := readRuntimeConfig(runtimePath)
	if err != nil {
		return "", false
	}
	appDir := strings.TrimSpace(cfg["app-dir"])
	if appDir == "" {
		appPath := strings.TrimSpace(cfg["app"])
		if appPath != "" {
			appDir = filepath.Dir(appPath)
		}
	}
	if appDir == "" {
		return "", false
	}
	if abs, err := filepath.Abs(appDir); err == nil {
		appDir = abs
	}
	return filepath.Join(appDir, "plugins"), true
}

func resolveBundledIntegrationRuntimePath(connectBin string) string {
	return connectsvc.ResolveBundledRuntimeConfigPath(connectBin)
}

func resolveIntegrationRuntimePath(connectBin string) (string, bool) {
	return connectsvc.ResolveRuntimeConfigPathFromConnectBin(connectBin)
}

func readRuntimeConfig(path string) (map[string]string, error) {
	return connectsvc.ReadRuntimeConfig(path)
}

func sanitizeUserFlags(flags map[string]string) map[string]string {
	next := map[string]string{}
	for key, value := range flags {
		switch key {
		case "agentId", "agent", "chatId", "chat", "session", "remote", "password", "certificate", "port", "ssh-bin", "connect-bin":
			next[key] = value
		}
	}
	return next
}

func withRuntimeFlags(flags map[string]string, runtime Paths) map[string]string {
	next := sanitizeUserFlags(flags)
	next["state"] = runtime.StatePath
	next["log"] = runtime.LogPath
	next["runtime-dir"] = runtime.RuntimeDir
	next["manager-socket"] = runtime.ManagerSocket
	next["pid-file"] = runtime.PIDFile
	return next
}

func ensureStopped(flags map[string]string) error {
	_ = StopManager(flags)
	return nil
}

func startDetachedManager(binary string, args []string) (int, error) {
	return startDetachedDaemon(binary, args)
}

func commandLogger(path, prefix string) (*log.Logger, func()) {
	writer, err := openLogWriter(path)
	if err != nil {
		return nil, nil
	}
	return log.New(writer, prefix, log.LstdFlags), func() {
		_ = writer.Close()
	}
}
