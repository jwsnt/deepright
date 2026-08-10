//go:build !darwin && !linux

package browserplaywrightsvc

import "os/exec"

func configureDetachedDaemonCommand(cmd *exec.Cmd) {
}
