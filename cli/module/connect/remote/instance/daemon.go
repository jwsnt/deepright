package instance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"connect/connectsvc"
	"connect/remote/runtimecfg"
)

func RunSessionDaemon(flags map[string]string) error {
	agentID, chatID, err := requiredIdentity(flags)
	if err != nil {
		return err
	}
	sshTarget := strings.TrimSpace(connectsvc.FirstValue(flags, "remote"))
	if sshTarget == "" {
		return errors.New("remote is required")
	}
	port, err := connectsvc.IntValue(flags, "port", defaultRemotePort)
	if err != nil {
		return err
	}
	socketPath := strings.TrimSpace(connectsvc.FirstValue(flags, "socket"))
	if socketPath == "" {
		return errors.New("socket is required")
	}
	controlPath := strings.TrimSpace(connectsvc.FirstValue(flags, "control"))
	if controlPath == "" {
		return errors.New("control is required")
	}
	sshBinary, err := resolveSSHBinary(flags)
	if err != nil {
		return err
	}
	logPath, err := resolveLogPath(flags)
	if err != nil {
		return err
	}
	loggerWriter, err := openLogWriter(logPath)
	if err != nil {
		return err
	}
	defer loggerWriter.Close()
	logger := log.New(loggerWriter, "remote-daemon ", log.LstdFlags)

	fingerprint, err := currentFingerprint(flags)
	if err != nil {
		return err
	}

	auth := authConfig{
		CertificatePath: strings.TrimSpace(connectsvc.FirstValue(flags, "certificate")),
	}
	password := ""
	if auth.CertificatePath == "" {
		password, err = readSecret(flags)
		if err != nil {
			return err
		}
	}
	auth.Password = password
	askpass := ""
	if password != "" {
		askpass, err = writeAskpass(flags, agentID, chatID, sshTarget)
		if err != nil {
			return err
		}
		defer removeFn(askpass)
	}

	if err := mkdirAllFn(filepath.Dir(socketPath), 0o755); err != nil {
		return err
	}
	_ = removeFn(socketPath)
	_ = removeFn(controlPath)

	sshCmd := exec.Command(sshBinary, sshMasterArgs(controlPath, port, sshTarget, auth)...)
	sshCmd.Env = append(os.Environ(), "DISPLAY=remote-daemon:0")
	if password != "" {
		sshCmd.Env = append(sshCmd.Env,
			"SSH_ASKPASS="+askpass,
			"SSH_ASKPASS_REQUIRE=force",
			"REMOTE_SSH_PASSWORD="+password,
		)
	}
	sshCmd.Stdin = nil
	sshCmd.Stdout = loggerWriter
	sshCmd.Stderr = loggerWriter
	if err := sshCmd.Start(); err != nil {
		return err
	}
	password = ""
	logger.Printf("用参数：Agent=%s；会话=%s；进程=%d；远程地址=%s；端口=%d，做了：建立远程连接", agentID, chatID, sshCmd.Process.Pid, sshTarget, port)

	if err := waitForSSHMaster(sshBinary, controlPath, port, sshTarget, sshCmd.Process.Pid, logger); err != nil {
		_ = sshCmd.Process.Kill()
		return err
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		_ = sshCmd.Process.Kill()
		return err
	}
	defer func() {
		_ = listener.Close()
		_ = removeFn(socketPath)
		_ = removeFn(controlPath)
	}()

	exitCh := make(chan error, 1)
	go func() {
		exitCh <- sshCmd.Wait()
	}()

	var once sync.Once
	stopSSH := func() {
		once.Do(func() {
			_ = exitSSHMasterFn(sshBinary, controlPath, port, sshTarget)
			if sshCmd.Process != nil {
				_ = sshCmd.Process.Signal(syscall.SIGTERM)
			}
		})
	}

	signalCh := make(chan os.Signal, 1)
	signalNotify(signalCh)
	defer signalStop(signalCh)
	go func() {
		<-signalCh
		logger.Printf("用参数：Agent=%s；会话=%s，做了：收到退出信号，准备关闭远程连接", agentID, chatID)
		stopSSH()
		_ = listener.Close()
	}()

	record := Record{
		AgentID: agentID,
		ChatID:  chatID,
		Port:    port,
		PID:     os.Getpid(),
		SSH:     sshTarget,
	}

	for {
		if unixListener, ok := listener.(*net.UnixListener); ok {
			_ = unixListener.SetDeadline(time.Now().Add(time.Second))
		}
		conn, err := listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				select {
				case waitErr := <-exitCh:
					if waitErr != nil {
						logger.Printf("用参数：原因=%v，做了：远程连接异常退出", waitErr)
					} else {
						logger.Printf("用参数：无，做了：远程连接正常退出")
					}
					return nil
				default:
					continue
				}
			}
			select {
			case waitErr := <-exitCh:
				if waitErr != nil {
					logger.Printf("用参数：原因=%v，做了：远程连接异常退出", waitErr)
				}
				return nil
			default:
				return err
			}
		}
		go func(c net.Conn) {
			defer c.Close()
			var req daemonRequest
			if err := json.NewDecoder(c).Decode(&req); err != nil {
				_ = json.NewEncoder(c).Encode(daemonResponse{OK: false, Error: err.Error()})
				return
			}
			switch req.Action {
			case "ping":
				_ = json.NewEncoder(c).Encode(daemonResponse{
					OK:          true,
					PID:         os.Getpid(),
					Version:     daemonProtocolVersion,
					Fingerprint: fingerprint,
					Record:      record,
				})
			case "exec":
				timeout := runtimecfg.DefaultCommandTimeout
				if req.TimeoutMillisecs > 0 {
					timeout = time.Duration(req.TimeoutMillisecs) * time.Millisecond
				}
				result, err := execRemoteCommand(sshBinary, controlPath, port, sshTarget, req.Command, timeout)
				if err != nil {
					_ = json.NewEncoder(c).Encode(daemonResponse{OK: false, Error: err.Error()})
					return
				}
				_ = json.NewEncoder(c).Encode(daemonResponse{
					OK:       true,
					Stdout:   result.Stdout,
					Stderr:   result.Stderr,
					ExitCode: result.ExitCode,
				})
			case "shutdown":
				_ = json.NewEncoder(c).Encode(daemonResponse{OK: true})
				stopSSH()
				_ = listener.Close()
			default:
				_ = json.NewEncoder(c).Encode(daemonResponse{OK: false, Error: fmt.Sprintf("unknown action: %s", req.Action)})
			}
		}(conn)
	}
}

func readSecret(flags map[string]string) (string, error) {
	path := strings.TrimSpace(connectsvc.FirstValue(flags, "secret-file"))
	if path == "" {
		return "", errors.New("secret-file is required")
	}
	data, err := readFileFn(path)
	if err != nil {
		return "", err
	}
	_ = removeFn(path)
	return strings.TrimSpace(string(data)), nil
}

func writeAskpass(flags map[string]string, agentID, chatID, sshTarget string) (string, error) {
	runtimeDir, err := resolveRuntimeDir(flags)
	if err != nil {
		return "", err
	}
	if err := mkdirAllFn(runtimeDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(runtimeDir, "askpass-"+sessionFileKey(agentID, chatID, sshTarget)+".sh")
	content := "#!/bin/sh\nprintf '%s\\n' \"$REMOTE_SSH_PASSWORD\"\n"
	if err := writeFileFn(path, []byte(content), 0o700); err != nil {
		return "", err
	}
	return path, nil
}

func sshMasterArgs(controlPath string, port int, sshTarget string, auth authConfig) []string {
	args := []string{
		"-M",
		"-N",
		"-S", controlPath,
		"-p", fmt.Sprintf("%d", port),
		"-o", "ControlMaster=yes",
		"-o", "ControlPersist=no",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=3",
		"-o", "ConnectTimeout=10",
	}
	if auth.CertificatePath != "" {
		args = append(args,
			"-i", auth.CertificatePath,
			"-o", "PreferredAuthentications=publickey",
			"-o", "PubkeyAuthentication=yes",
			"-o", "PasswordAuthentication=no",
		)
	} else {
		args = append(args,
			"-o", "PreferredAuthentications=password",
			"-o", "PubkeyAuthentication=no",
			"-o", "NumberOfPasswordPrompts=1",
		)
	}
	args = append(args, sshTarget)
	return args
}

func waitForSSHMaster(sshBinary, controlPath string, port int, sshTarget string, pid int, logger *log.Logger) error {
	deadline := time.Now().Add(defaultStartupTimeout)
	for time.Now().Before(deadline) {
		if !processExistsFn(pid) {
			return errors.New("ssh master exited during startup")
		}
		if err := checkSSHMasterFn(sshBinary, controlPath, port, sshTarget); err == nil {
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	if logger != nil {
		logger.Printf("用参数：控制文件=%s；远程地址=%s，做了：等待远程连接就绪超时", controlPath, sshTarget)
	}
	return errors.New("ssh master not ready before timeout")
}

func execRemoteCommand(sshBinary, controlPath string, port int, sshTarget, remoteCommand string, timeout time.Duration) (CommandResult, error) {
	ctx := context.Background()
	cancel := func() {}
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
	}
	defer cancel()

	cmd := exec.CommandContext(ctx, sshBinary,
		"-S", controlPath,
		"-p", fmt.Sprintf("%d", port),
		"-o", "BatchMode=yes",
		sshTarget,
		remoteCommand,
	)
	output, err := cmd.CombinedOutput()
	result := CommandResult{
		Stdout:   string(output),
		Stderr:   "",
		ExitCode: 0,
	}
	if err == nil {
		return result, nil
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return CommandResult{}, fmt.Errorf("remote exec timed out after %dms", int(timeout/time.Millisecond))
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
		result.Stdout = ""
		result.Stderr = string(output)
		return result, nil
	}
	return CommandResult{}, err
}
