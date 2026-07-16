package instance

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func stubRuntime(t *testing.T) {
	t.Helper()
	oldExec := executablePathFn
	oldStat := statFn
	oldStart := startDetachedDaemonFn
	oldExists := processExistsFn
	oldPing := pingDaemonFn
	oldTerminate := terminateProcessFn
	oldExitMaster := exitSSHMasterFn
	oldCleanup := cleanupRuntimeFn
	oldReadPID := readPIDFileFn
	t.Cleanup(func() {
		executablePathFn = oldExec
		statFn = oldStat
		startDetachedDaemonFn = oldStart
		processExistsFn = oldExists
		pingDaemonFn = oldPing
		terminateProcessFn = oldTerminate
		exitSSHMasterFn = oldExitMaster
		cleanupRuntimeFn = oldCleanup
		readPIDFileFn = oldReadPID
	})
}

func TestCreateReusesMatchingLiveDaemon(t *testing.T) {
	stubRuntime(t)

	tempDir := t.TempDir()
	selfPath := filepath.Join(tempDir, "remote")
	if err := writeFileFn(selfPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	executablePathFn = func() (string, error) { return selfPath, nil }
	info, err := os.Stat(selfPath)
	if err != nil {
		t.Fatal(err)
	}
	statFn = func(name string) (os.FileInfo, error) { return info, nil }
	processExistsFn = func(pid int) bool { return pid == 101 || pid == 202 }
	pingDaemonFn = func(socketPath string, timeout time.Duration) (daemonResponse, error) {
		record := Record{AgentID: "agent-a", ChatID: "chat-1", Port: 22, PID: 101, SSH: "ubuntu@1.2.3.4"}
		if strings.Contains(socketPath, sessionFileKey("agent-a", "chat-1", "ubuntu@1.2.3.5")) {
			record = Record{AgentID: "agent-a", ChatID: "chat-1", Port: 22, PID: 202, SSH: "ubuntu@1.2.3.5"}
		}
		return daemonResponse{
			OK:      true,
			PID:     record.PID,
			Version: daemonProtocolVersion,
			Fingerprint: execFingerprint{
				Path:            selfPath,
				Size:            info.Size(),
				ModTimeUnixNano: info.ModTime().UnixNano(),
			},
			Record: record,
		}, nil
	}
	startDetachedDaemonFn = func(binary string, args []string) (int, error) {
		t.Fatal("startDetachedDaemon should not be called")
		return 0, nil
	}

	statePath := filepath.Join(tempDir, "remote.json")
	if err := saveState(statePath, []Record{{AgentID: "agent-a", ChatID: "chat-1", Port: 22, PID: 101, SSH: "ubuntu@1.2.3.4"}}); err != nil {
		t.Fatal(err)
	}

	item, err := Create(map[string]string{
		"agentId": "AGENT-A",
		"chatId":  "CHAT-1",
		"remote":  "ubuntu@1.2.3.4",
		"state":   statePath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.AgentID != "agent-a" || item.ChatID != "chat-1" || item.PID != 101 {
		t.Fatalf("unexpected item: %+v", item)
	}
}

func TestCreateDoesNotReuseDifferentRemote(t *testing.T) {
	stubRuntime(t)

	tempDir := t.TempDir()
	selfPath := filepath.Join(tempDir, "remote")
	if err := os.WriteFile(selfPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	executablePathFn = func() (string, error) { return selfPath, nil }
	info, err := os.Stat(selfPath)
	if err != nil {
		t.Fatal(err)
	}
	statFn = func(name string) (os.FileInfo, error) { return info, nil }
	processExistsFn = func(pid int) bool { return pid == 101 || pid == 202 }
	pingDaemonFn = func(socketPath string, timeout time.Duration) (daemonResponse, error) {
		record := Record{AgentID: "agent-a", ChatID: "chat-1", Port: 22, PID: 101, SSH: "ubuntu@1.2.3.4"}
		if strings.Contains(socketPath, sessionFileKey("agent-a", "chat-1", "ubuntu@1.2.3.5")) {
			record = Record{AgentID: "agent-a", ChatID: "chat-1", Port: 22, PID: 202, SSH: "ubuntu@1.2.3.5"}
		}
		return daemonResponse{
			OK:      true,
			PID:     record.PID,
			Version: daemonProtocolVersion,
			Fingerprint: execFingerprint{
				Path:            selfPath,
				Size:            info.Size(),
				ModTimeUnixNano: info.ModTime().UnixNano(),
			},
			Record: record,
		}, nil
	}

	startCalls := 0
	startDetachedDaemonFn = func(binary string, args []string) (int, error) {
		startCalls++
		return 202, nil
	}

	statePath := filepath.Join(tempDir, "remote.json")
	if err := saveState(statePath, []Record{{AgentID: "agent-a", ChatID: "chat-1", Port: 22, PID: 101, SSH: "ubuntu@1.2.3.4"}}); err != nil {
		t.Fatal(err)
	}

	item, err := Create(map[string]string{
		"agentId":  "agent-a",
		"chatId":   "chat-1",
		"remote":   "ubuntu@1.2.3.5",
		"password": "secret",
		"state":    statePath,
		"self":     selfPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.SSH != "ubuntu@1.2.3.5" || item.PID != 202 {
		t.Fatalf("unexpected item: %+v", item)
	}
	if startCalls != 1 {
		t.Fatalf("start calls = %d", startCalls)
	}

	persisted, err := loadStateRecords(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(persisted) != 2 {
		t.Fatalf("persisted len = %d", len(persisted))
	}
}

func TestGetRefreshesLastActiveAt(t *testing.T) {
	stubRuntime(t)

	tempDir := t.TempDir()
	selfPath := filepath.Join(tempDir, "remote")
	if err := os.WriteFile(selfPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	executablePathFn = func() (string, error) { return selfPath, nil }
	info, err := os.Stat(selfPath)
	if err != nil {
		t.Fatal(err)
	}
	statFn = func(name string) (os.FileInfo, error) { return info, nil }
	processExistsFn = func(pid int) bool { return pid == 701 }
	pingDaemonFn = func(socketPath string, timeout time.Duration) (daemonResponse, error) {
		return daemonResponse{
			OK:      true,
			PID:     701,
			Version: daemonProtocolVersion,
			Fingerprint: execFingerprint{
				Path:            selfPath,
				Size:            info.Size(),
				ModTimeUnixNano: info.ModTime().UnixNano(),
			},
			Record: Record{AgentID: "agent-a", ChatID: "chat-1", Port: 22, PID: 701, SSH: "ubuntu@1.2.3.4"},
		}, nil
	}

	statePath := filepath.Join(tempDir, "remote.json")
	oldActivity := formatActivityTime(time.Now().Add(-20 * time.Minute))
	if err := saveStateRecords(statePath, []stateRecord{{
		AgentID:      "agent-a",
		ChatID:       "chat-1",
		Port:         22,
		PID:          701,
		SSH:          "ubuntu@1.2.3.4",
		LastActiveAt: oldActivity,
	}}); err != nil {
		t.Fatal(err)
	}

	item, err := Get(map[string]string{
		"agentId": "agent-a",
		"chatId":  "chat-1",
		"state":   statePath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.PID != 701 {
		t.Fatalf("unexpected item: %+v", item)
	}

	persisted, err := loadStateRecords(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(persisted) != 1 {
		t.Fatalf("persisted len = %d", len(persisted))
	}
	if persisted[0].LastActiveAt == oldActivity {
		t.Fatalf("expected lastActiveAt refresh, got %q", persisted[0].LastActiveAt)
	}
}

func TestGetRequiresRemoteWhenMultipleSessionsShareIdentity(t *testing.T) {
	stubRuntime(t)

	tempDir := t.TempDir()
	selfPath := filepath.Join(tempDir, "remote")
	if err := os.WriteFile(selfPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	executablePathFn = func() (string, error) { return selfPath, nil }
	info, err := os.Stat(selfPath)
	if err != nil {
		t.Fatal(err)
	}
	statFn = func(name string) (os.FileInfo, error) { return info, nil }
	processExistsFn = func(pid int) bool { return pid == 701 || pid == 702 }
	pingDaemonFn = func(socketPath string, timeout time.Duration) (daemonResponse, error) {
		record := Record{AgentID: "agent-a", ChatID: "chat-1", Port: 22, PID: 701, SSH: "ubuntu@1.2.3.4"}
		if strings.Contains(socketPath, sessionFileKey("agent-a", "chat-1", "ubuntu@1.2.3.5")) {
			record = Record{AgentID: "agent-a", ChatID: "chat-1", Port: 22, PID: 702, SSH: "ubuntu@1.2.3.5"}
		}
		return daemonResponse{
			OK:      true,
			PID:     record.PID,
			Version: daemonProtocolVersion,
			Fingerprint: execFingerprint{
				Path:            selfPath,
				Size:            info.Size(),
				ModTimeUnixNano: info.ModTime().UnixNano(),
			},
			Record: record,
		}, nil
	}

	statePath := filepath.Join(tempDir, "remote.json")
	if err := saveState(statePath, []Record{
		{AgentID: "agent-a", ChatID: "chat-1", Port: 22, PID: 701, SSH: "ubuntu@1.2.3.4"},
		{AgentID: "agent-a", ChatID: "chat-1", Port: 22, PID: 702, SSH: "ubuntu@1.2.3.5"},
	}); err != nil {
		t.Fatal(err)
	}

	_, err = Get(map[string]string{
		"agentId": "agent-a",
		"chatId":  "chat-1",
		"state":   statePath,
	})
	if err == nil || !strings.Contains(err.Error(), "remote is required") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestShutdownWithoutRemoteClosesAllMatchingSessions(t *testing.T) {
	stubRuntime(t)

	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "remote.json")
	if err := saveState(statePath, []Record{
		{AgentID: "agent-a", ChatID: "chat-1", Port: 22, PID: 701, SSH: "ubuntu@1.2.3.4"},
		{AgentID: "agent-a", ChatID: "chat-1", Port: 22, PID: 702, SSH: "ubuntu@1.2.3.5"},
		{AgentID: "agent-b", ChatID: "chat-2", Port: 22, PID: 703, SSH: "ubuntu@1.2.3.6"},
	}); err != nil {
		t.Fatal(err)
	}

	closed := []string{}
	cleanupRuntimeFn = func(flags map[string]string, record Record, requestShutdown bool) error {
		closed = append(closed, record.SSH)
		return nil
	}

	if err := Shutdown(map[string]string{
		"agentId": "agent-a",
		"chatId":  "chat-1",
		"state":   statePath,
	}); err != nil {
		t.Fatal(err)
	}

	if len(closed) != 2 {
		t.Fatalf("closed len = %d", len(closed))
	}

	persisted, err := loadStateRecords(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(persisted) != 1 || persisted[0].AgentID != "agent-b" {
		t.Fatalf("persisted = %+v", persisted)
	}
}

func TestPruneIdleSessionsClosesAndLogs(t *testing.T) {
	stubRuntime(t)

	tempDir := t.TempDir()
	selfPath := filepath.Join(tempDir, "remote")
	if err := os.WriteFile(selfPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	executablePathFn = func() (string, error) { return selfPath, nil }
	info, err := os.Stat(selfPath)
	if err != nil {
		t.Fatal(err)
	}
	statFn = func(name string) (os.FileInfo, error) { return info, nil }
	processExistsFn = func(pid int) bool { return pid == 801 || pid == 802 }
	pingDaemonFn = func(socketPath string, timeout time.Duration) (daemonResponse, error) {
		pid := 801
		agentID := "agent-old"
		chatID := "chat-old"
		if strings.Contains(socketPath, sessionFileKey("agent-new", "chat-new", "ubuntu@1.2.3.4")) {
			pid = 802
			agentID = "agent-new"
			chatID = "chat-new"
		}
		return daemonResponse{
			OK:      true,
			PID:     pid,
			Version: daemonProtocolVersion,
			Fingerprint: execFingerprint{
				Path:            selfPath,
				Size:            info.Size(),
				ModTimeUnixNano: info.ModTime().UnixNano(),
			},
			Record: Record{AgentID: agentID, ChatID: chatID, Port: 22, PID: pid, SSH: "ubuntu@1.2.3.4"},
		}, nil
	}

	closed := []string{}
	cleanupRuntimeFn = func(flags map[string]string, record Record, requestShutdown bool) error {
		closed = append(closed, record.AgentID+"@"+record.ChatID)
		return nil
	}

	statePath := filepath.Join(tempDir, "remote.json")
	if err := saveStateRecords(statePath, []stateRecord{
		{
			AgentID:      "agent-old",
			ChatID:       "chat-old",
			Port:         22,
			PID:          801,
			SSH:          "ubuntu@1.2.3.4",
			LastActiveAt: formatActivityTime(time.Now().Add(-16 * time.Minute)),
		},
		{
			AgentID:      "agent-new",
			ChatID:       "chat-new",
			Port:         22,
			PID:          802,
			SSH:          "ubuntu@1.2.3.4",
			LastActiveAt: formatActivityTime(time.Now().Add(-3 * time.Minute)),
		},
	}); err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(tempDir, "remote.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	logger := log.New(file, "test ", 0)

	count, err := pruneIdleSessions(map[string]string{
		"state": statePath,
		"log":   logPath,
	}, 15*time.Minute, logger)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("closed count = %d, want 1", count)
	}
	if len(closed) != 1 || closed[0] != "agent-old@chat-old" {
		t.Fatalf("closed = %+v", closed)
	}

	persisted, err := loadStateRecords(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(persisted) != 1 || persisted[0].AgentID != "agent-new" {
		t.Fatalf("persisted = %+v", persisted)
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(logData), "做了：自动关闭了长时间没活动的远程连接") {
		t.Fatalf("expected idle close log, got %s", string(logData))
	}
}

func TestExecViaManagerPassesConfiguredTimeout(t *testing.T) {
	oldRequest := managerRequestFn
	oldStart := startManagerDaemonFn
	t.Cleanup(func() {
		managerRequestFn = oldRequest
		startManagerDaemonFn = oldStart
	})

	tempDir := t.TempDir()
	selfPath := filepath.Join(tempDir, "remote")
	if err := os.WriteFile(selfPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	connectBin := filepath.Join(tempDir, "integration")
	if err := os.WriteFile(connectBin, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tempDir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "config", "config.json"), []byte(`{"remote":{"exec_timeout":9}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	executablePathFn = func() (string, error) { return selfPath, nil }
	readPIDFileFn = func(string) ([]byte, error) { return []byte("321\n"), nil }
	processExistsFn = func(pid int) bool { return pid == 321 }
	startManagerDaemonFn = func(binary string, args []string) (int, error) {
		t.Fatal("startManagerDaemon should not be called")
		return 0, nil
	}

	managerRequestFn = func(socketPath string, req managerRequest, timeout time.Duration) (managerResponse, error) {
		switch req.Action {
		case "ping":
			return managerResponse{
				OK:          true,
				PID:         321,
				Version:     managerProtocolVersion,
				Fingerprint: fingerprintString(map[string]string{"self": selfPath, "connect-bin": connectBin}),
			}, nil
		case "exec":
			if req.TimeoutMillisecs != 9000 {
				t.Fatalf("timeout milliseconds = %d", req.TimeoutMillisecs)
			}
			if timeout != 14*time.Second {
				t.Fatalf("manager timeout = %v", timeout)
			}
			return managerResponse{OK: true, Stdout: "ok", ExitCode: 0}, nil
		default:
			t.Fatalf("unexpected action: %s", req.Action)
			return managerResponse{}, nil
		}
	}

	result, err := ExecViaManager(map[string]string{"self": selfPath, "connect-bin": connectBin}, "echo ok")
	if err != nil {
		t.Fatal(err)
	}
	if result.Stdout != "ok" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestExecViaManagerPrefersExplicitTimeoutFlag(t *testing.T) {
	oldRequest := managerRequestFn
	oldStart := startManagerDaemonFn
	t.Cleanup(func() {
		managerRequestFn = oldRequest
		startManagerDaemonFn = oldStart
	})

	tempDir := t.TempDir()
	selfPath := filepath.Join(tempDir, "remote")
	if err := os.WriteFile(selfPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	executablePathFn = func() (string, error) { return selfPath, nil }
	readPIDFileFn = func(string) ([]byte, error) { return []byte("321\n"), nil }
	processExistsFn = func(pid int) bool { return pid == 321 }
	startManagerDaemonFn = func(binary string, args []string) (int, error) {
		t.Fatal("startManagerDaemon should not be called")
		return 0, nil
	}

	managerRequestFn = func(socketPath string, req managerRequest, timeout time.Duration) (managerResponse, error) {
		switch req.Action {
		case "ping":
			return managerResponse{
				OK:          true,
				PID:         321,
				Version:     managerProtocolVersion,
				Fingerprint: fingerprintString(map[string]string{"self": selfPath}),
			}, nil
		case "exec":
			if req.TimeoutMillisecs != 1234 {
				t.Fatalf("timeout milliseconds = %d", req.TimeoutMillisecs)
			}
			if timeout != 6234*time.Millisecond {
				t.Fatalf("manager timeout = %v", timeout)
			}
			return managerResponse{OK: true, Stdout: "ok", ExitCode: 0}, nil
		default:
			t.Fatalf("unexpected action: %s", req.Action)
			return managerResponse{}, nil
		}
	}

	result, err := ExecViaManager(map[string]string{
		"self":        selfPath,
		"connect-bin": "/tmp/integration",
		"timeout":     "1234",
	}, "echo ok")
	if err != nil {
		t.Fatal(err)
	}
	if result.Stdout != "ok" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCreateViaManagerUsesCreateTimeout(t *testing.T) {
	oldRequest := managerRequestFn
	oldStart := startManagerDaemonFn
	t.Cleanup(func() {
		managerRequestFn = oldRequest
		startManagerDaemonFn = oldStart
	})

	tempDir := t.TempDir()
	selfPath := filepath.Join(tempDir, "remote")
	if err := os.WriteFile(selfPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	executablePathFn = func() (string, error) { return selfPath, nil }
	readPIDFileFn = func(string) ([]byte, error) { return []byte("321\n"), nil }
	processExistsFn = func(pid int) bool { return pid == 321 }
	startManagerDaemonFn = func(binary string, args []string) (int, error) {
		t.Fatal("startManagerDaemon should not be called")
		return 0, nil
	}

	managerRequestFn = func(socketPath string, req managerRequest, timeout time.Duration) (managerResponse, error) {
		switch req.Action {
		case "ping":
			return managerResponse{
				OK:          true,
				PID:         321,
				Version:     managerProtocolVersion,
				Fingerprint: fingerprintString(map[string]string{"self": selfPath}),
			}, nil
		case "create":
			if timeout != defaultCreateTimeout {
				t.Fatalf("manager timeout = %v", timeout)
			}
			return managerResponse{OK: true, Item: Record{AgentID: "agent-a", ChatID: "chat-a", SSH: "ubuntu@1.2.3.4", PID: 1001, Port: 22}}, nil
		default:
			t.Fatalf("unexpected action: %s", req.Action)
			return managerResponse{}, nil
		}
	}

	item, err := CreateViaManager(map[string]string{"self": selfPath})
	if err != nil {
		t.Fatal(err)
	}
	if item.PID != 1001 {
		t.Fatalf("unexpected item: %+v", item)
	}
}

func TestResolveTransportRuntimeDirFallsBackWhenSSHControlPathWouldBeTooLong(t *testing.T) {
	flags := map[string]string{
		"runtime-dir": filepath.Join(t.TempDir(), strings.Repeat("very-long-segment-", 5), ".remote"),
	}

	runtimeDir, err := resolveTransportRuntimeDir(flags)
	if err != nil {
		t.Fatal(err)
	}
	if runtimeDir == flags["runtime-dir"] {
		t.Fatalf("expected shorter transport runtime dir, got %s", runtimeDir)
	}
	if !transportRuntimeDirFits(runtimeDir) {
		t.Fatalf("transport runtime dir still too long: %s", runtimeDir)
	}

	controlPath, err := resolveControlPath(flags, "agent-a", "chat-a", "ubuntu@1.2.3.4")
	if err != nil {
		t.Fatal(err)
	}
	if len(controlPath+"."+strings.Repeat("f", 16)) > maxUnixPathLength {
		t.Fatalf("control path too long: %s", controlPath)
	}
}

func TestStartAndStopManagerWriteLifecycleLogs(t *testing.T) {
	stubRuntime(t)

	tempDir := t.TempDir()
	selfPath := filepath.Join(tempDir, "remote")
	if err := os.WriteFile(selfPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	executablePathFn = func() (string, error) { return selfPath, nil }

	oldStart := startManagerDaemonFn
	oldManagerReq := managerRequestFn
	oldReadPID := readPIDFileFn
	oldProcessExists := processExistsFn
	t.Cleanup(func() {
		startManagerDaemonFn = oldStart
		managerRequestFn = oldManagerReq
		readPIDFileFn = oldReadPID
		processExistsFn = oldProcessExists
	})

	logPath := filepath.Join(tempDir, "remote.log")
	pidPath := filepath.Join(tempDir, "remote.pid")
	managerSocket := filepath.Join(tempDir, ".remote", "manager.sock")

	startManagerDaemonFn = func(binary string, args []string) (int, error) {
		if err := os.MkdirAll(filepath.Dir(pidPath), 0o755); err != nil {
			return 0, err
		}
		if err := os.WriteFile(pidPath, []byte("777\n"), 0o644); err != nil {
			return 0, err
		}
		return 777, nil
	}
	processExistsFn = func(pid int) bool { return pid == 777 }
	readPIDFileFn = func(path string) ([]byte, error) {
		if path == pidPath {
			return []byte("777\n"), nil
		}
		return os.ReadFile(path)
	}
	managerRequestFn = func(socketPath string, req managerRequest, timeout time.Duration) (managerResponse, error) {
		switch req.Action {
		case "ping":
			return managerResponse{
				OK:          true,
				PID:         777,
				Version:     managerProtocolVersion,
				Fingerprint: fingerprintString(map[string]string{"self": selfPath}),
			}, nil
		case "shutdown":
			return managerResponse{OK: true}, nil
		default:
			return managerResponse{}, nil
		}
	}

	flags := map[string]string{
		"self":           selfPath,
		"log":            logPath,
		"pid-file":       pidPath,
		"manager-socket": managerSocket,
		"runtime-dir":    filepath.Join(tempDir, ".remote"),
		"state":          filepath.Join(tempDir, "remote.json"),
	}

	if err := StartManager(flags); err != nil {
		t.Fatal(err)
	}
	if err := StopManager(flags); err != nil {
		t.Fatal(err)
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(logData)
	for _, want := range []string{"准备启动远程连接管理服务", "远程连接管理服务已启动", "准备停止远程连接管理服务", "远程连接管理服务已停止"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected log to contain %q, got %s", want, text)
		}
	}
}

func TestAuthFromFlagsSupportsCertificate(t *testing.T) {
	stubRuntime(t)

	tempDir := t.TempDir()
	certPath := filepath.Join(tempDir, "id.pem")
	if err := os.WriteFile(certPath, []byte("pem"), 0o600); err != nil {
		t.Fatal(err)
	}

	auth, err := authFromFlags(map[string]string{
		"certificate": certPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if auth.CertificatePath != certPath {
		t.Fatalf("certificate path = %q, want %q", auth.CertificatePath, certPath)
	}
	if auth.Password != "" {
		t.Fatalf("password should be empty, got %q", auth.Password)
	}
}
