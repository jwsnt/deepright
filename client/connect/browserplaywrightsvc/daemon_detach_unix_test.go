//go:build darwin || linux

package browserplaywrightsvc

import (
	"os/exec"
	"testing"
)

func TestConfigureDetachedDaemonCommandSetsSetsid(t *testing.T) {
	cmd := exec.Command("sh", "-c", "exit 0")

	configureDetachedDaemonCommand(cmd)

	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr should be configured for detached daemon start")
	}
	if !cmd.SysProcAttr.Setsid {
		t.Fatal("Setsid should be enabled for detached daemon start")
	}
}
