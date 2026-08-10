//go:build linux

package notification

import (
	"os"
	"strings"
)

var notificationReadFileFn = os.ReadFile

func supported() bool {
	return runningUnderWSL() && powerShellNotificationSupported()
}

func settingsSupported() bool {
	return false
}

func notify(opts Options) error {
	if !supported() {
		return nil
	}
	return notifyViaPowerShell(opts)
}

func runningUnderWSL() bool {
	if strings.TrimSpace(os.Getenv("WSL_INTEROP")) != "" || strings.TrimSpace(os.Getenv("WSL_DISTRO_NAME")) != "" {
		return true
	}
	for _, candidate := range []string{"/proc/sys/kernel/osrelease", "/proc/version"} {
		data, err := notificationReadFileFn(candidate)
		if err == nil && looksLikeWSLKernel(data) {
			return true
		}
	}
	return false
}

func openSettings() error {
	return nil
}
