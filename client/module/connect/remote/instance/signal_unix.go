//go:build darwin || linux

package instance

import (
	"errors"
	"os"
	"os/signal"
	"syscall"
)

func signalNotify(ch chan os.Signal) {
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
}

func signalStop(ch chan os.Signal) {
	signal.Stop(ch)
}

func errorsIsNoProcess(err error) bool {
	return errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH)
}
