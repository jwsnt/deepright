//go:build windows

package main

import (
	"os/exec"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

const browserWindowsStillActive = 259

func browserPrepareDetachedCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS,
		HideWindow:    true,
	}
}

func browserPrepareAttachedCommand(cmd *exec.Cmd) {
	browserPrepareDetachedCommand(cmd)
}

func browserProcessExistsByPID(pid int) bool {
	if pid <= 0 {
		return false
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)
	var exitCode uint32
	if err := windows.GetExitCodeProcess(handle, &exitCode); err != nil {
		return false
	}
	return exitCode == browserWindowsStillActive
}

func browserTerminateProcessByPID(pid int) error {
	if pid <= 0 || !browserProcessExistsFn(pid) {
		return nil
	}
	handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	if err := windows.TerminateProcess(handle, 1); err != nil {
		return err
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !browserProcessExistsFn(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}
