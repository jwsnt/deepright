package main

import (
	"context"
	"sync"
	"testing"
)

func TestRunIntegrationDependencyChecksRunsAllChecks(t *testing.T) {
	var mu sync.Mutex
	ran := map[string]bool{}
	checks := make([]integrationDependencyCheck, 0, 5)
	for _, name := range []string{"ffmpeg", "whisper", "voxcpm", "rembg", "rvm"} {
		name := name
		checks = append(checks, integrationDependencyCheck{
			name: name,
			check: func() (int, bool, string) {
				mu.Lock()
				ran[name] = true
				mu.Unlock()
				return 200, true, ""
			},
		})
	}

	runIntegrationDependencyChecks(context.Background(), checks)
	for _, name := range []string{"ffmpeg", "whisper", "voxcpm", "rembg", "rvm"} {
		if !ran[name] {
			t.Fatalf("%s check was not run", name)
		}
	}
}

func TestRunIntegrationDependencyChecksSkipsCancelledStartup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ran := false
	runIntegrationDependencyChecks(ctx, []integrationDependencyCheck{{
		name: "ffmpeg",
		check: func() (int, bool, string) {
			ran = true
			return 200, true, ""
		},
	}})
	if ran {
		t.Fatal("cancelled startup must not run dependency checks")
	}
}
