//go:build darwin && !cgo

package notification

func supported() bool {
	return false
}

func settingsSupported() bool {
	return false
}

func notify(Options) error {
	return nil
}

func openSettings() error {
	return nil
}
