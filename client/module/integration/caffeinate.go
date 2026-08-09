package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

const integrationCaffeinateBinary = "caffeinate"
const integrationCaffeinateStopTimeout = 2 * time.Second

var integrationCaffeinateCommandFn = exec.Command

type integrationSleepAssertion interface {
	Stop() error
}

type integrationCaffeinateProcess struct {
	cmd      *exec.Cmd
	done     chan struct{}
	stopOnce sync.Once
	stopErr  error
}

func startIntegrationSleepAssertion() (integrationSleepAssertion, error) {
	if integrationRuntimeGOOS != "darwin" {
		return nil, nil
	}

	cmd := integrationCaffeinateCommandFn(integrationCaffeinateBinary, "-d", "-i", "-m", "-s")
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	process := &integrationCaffeinateProcess{
		cmd:  cmd,
		done: make(chan struct{}),
	}
	go func() {
		_ = cmd.Wait()
		close(process.done)
	}()
	return process, nil
}

func (p *integrationCaffeinateProcess) Stop() error {
	if p == nil {
		return nil
	}
	p.stopOnce.Do(func() {
		if p.cmd == nil || p.cmd.Process == nil {
			return
		}
		select {
		case <-p.done:
			return
		default:
		}
		if err := p.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			p.stopErr = fmt.Errorf("terminate caffeinate: %w", err)
			return
		}
		timer := time.NewTimer(integrationCaffeinateStopTimeout)
		defer timer.Stop()
		select {
		case <-p.done:
		case <-timer.C:
			p.stopErr = fmt.Errorf("wait for caffeinate termination timed out after %s", integrationCaffeinateStopTimeout)
		}
	})
	return p.stopErr
}

func (p *integrationCaffeinateProcess) Running() bool {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return false
	}
	select {
	case <-p.done:
		return false
	default:
		return true
	}
}
