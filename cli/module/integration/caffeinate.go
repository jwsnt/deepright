package main

import (
	"os/exec"
	"sync"
)

const integrationCaffeinateBinary = "caffeinate"

var integrationCaffeinateCommandFn = exec.Command

type integrationSleepAssertion interface {
	Stop() error
}

type integrationCaffeinateProcess struct {
	cmd      *exec.Cmd
	done     chan struct{}
	stopOnce sync.Once
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
		_ = p.cmd.Process.Kill()
		<-p.done
	})
	return nil
}
