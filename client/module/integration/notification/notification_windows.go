//go:build windows

package notification

func supported() bool {
	return powerShellNotificationSupported()
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

func openSettings() error {
	return nil
}
