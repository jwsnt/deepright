package main

import (
	"fmt"
	"time"
)

var (
	browserLifecycleProbeInterval = 5 * time.Second
	browserLifecycleProbeTimeout  = time.Minute
	browserSleepFn                = time.Sleep
)

func browserWaitForLifecycleCDPState(port int, wantReady bool) error {
	timeout := browserLifecycleProbeTimeout
	if timeout <= 0 {
		timeout = time.Minute
	}
	interval := browserLifecycleProbeInterval
	if interval <= 0 {
		interval = time.Second
	}
	deadline := browserNowFn().Add(timeout)
	var lastErr error
	for {
		_, err := browserResolveLiveCDPEndpoint(port)
		ready := err == nil
		if wantReady && ready {
			return nil
		}
		if !wantReady && !ready {
			return nil
		}
		lastErr = err
		if !browserNowFn().Before(deadline) {
			break
		}
		browserSleepFn(interval)
	}
	if wantReady {
		if lastErr != nil {
			return fmt.Errorf("timed out waiting for CDP on port %d: %w", port, lastErr)
		}
		return fmt.Errorf("timed out waiting for CDP on port %d", port)
	}
	if lastErr != nil {
		return fmt.Errorf("timed out waiting for CDP shutdown on port %d: %w", port, lastErr)
	}
	return fmt.Errorf("timed out waiting for CDP shutdown on port %d", port)
}
