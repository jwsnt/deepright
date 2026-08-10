//go:build !darwin && !linux

package instance

import "os"

func signalNotify(ch chan os.Signal) {
}

func signalStop(ch chan os.Signal) {
}

func errorsIsNoProcess(err error) bool {
	return false
}
