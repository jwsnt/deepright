//go:build darwin || linux

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func startDetachedIntegrationProcess(binary string, args []string, logPath string, skipOpenBrowser bool) (int, error) {
	if err := ensureParentDir(logPath); err != nil {
		return 0, err
	}
	logWriter, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer logWriter.Close()

	cmd := exec.Command(binary, args...)
	cmd.Dir = filepath.Dir(binary)
	cmd.Stdin = nil
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	if skipOpenBrowser {
		cmd.Env = append(os.Environ(), integrationSkipBrowserEnv+"=1")
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()
	return pid, nil
}
