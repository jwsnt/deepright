//go:build darwin || linux

package instance

import (
	"os"
	"os/exec"
	"syscall"
	"time"
)

func startDetachedDaemon(binary string, args []string) (int, error) {
	logPath := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--log" && i+1 < len(args) {
			logPath = args[i+1]
			break
		}
	}
	logWriter, err := openLogWriter(logPath)
	if err != nil {
		return 0, err
	}
	defer logWriter.Close()

	cmd := exec.Command(binary, args...)
	cmd.Stdin = nil
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()
	return pid, nil
}

func processExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

func terminateProcess(pid int) error {
	if pid <= 0 {
		return nil
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := process.Signal(syscall.SIGTERM); err != nil && !errorsIsNoProcess(err) {
		return err
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !processExists(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err := process.Signal(syscall.SIGKILL); err != nil && !errorsIsNoProcess(err) {
		return err
	}
	return nil
}
