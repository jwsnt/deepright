//go:build darwin || linux

package browserplaywrightsvc

import (
	"os/exec"
	"syscall"
)

func configureDetachedDaemonCommand(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
