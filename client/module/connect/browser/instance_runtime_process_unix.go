//go:build !windows

package main

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func browserPrepareDetachedCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func browserPrepareAttachedCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

func browserProcessExistsByPID(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}

func browserTerminateProcessByPID(pid int) error {
	if pid <= 0 || !browserProcessExistsFn(pid) {
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !browserProcessExistsFn(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err := proc.Signal(syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}
