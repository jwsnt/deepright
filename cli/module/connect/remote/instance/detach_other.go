//go:build !darwin && !linux

package instance

import (
	"errors"
)

func startDetachedDaemon(binary string, args []string) (int, error) {
	return 0, errors.New("remote daemon detach is only implemented for darwin and linux")
}

func processExists(pid int) bool {
	return false
}

func terminateProcess(pid int) error {
	return nil
}
