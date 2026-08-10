package main

import (
	"context"
	"log"
	"sync"
)

type integrationDependencyCheck struct {
	name  string
	check func() (status int, available bool, content string)
}

// startIntegrationDependencyChecks warms the same success caches used by the
// page-triggered dependency checks. It is deliberately asynchronous so a
// slow Python probe never delays application startup or browser readiness.
func startIntegrationDependencyChecks(ctx context.Context, cfg *Config) {
	checks := []integrationDependencyCheck{
		{
			name: "bubblewrap",
			check: func() (int, bool, string) {
				status, response := checkBubblewrapDependency()
				return status, response.Available, response.Content
			},
		},
		{
			name: "ffmpeg",
			check: func() (int, bool, string) {
				status, response := checkFFmpegDependency()
				return status, response.Available, response.Content
			},
		},
		{
			name: "whisper",
			check: func() (int, bool, string) {
				status, response := checkWhisperDependency()
				return status, response.Available, response.Content
			},
		},
		{
			name: "voxcpm",
			check: func() (int, bool, string) {
				status, response := checkVoxCPMDependency()
				return status, response.Available, response.Content
			},
		},
		{
			name: "rembg",
			check: func() (int, bool, string) {
				status, response := checkRembgDependency()
				return status, response.Available, response.Content
			},
		},
		{
			name: "rvm",
			check: func() (int, bool, string) {
				status, response := checkRVMDependency(cfg, "")
				return status, response.Available, response.Content
			},
		},
		{
			name: "wav2lip",
			check: func() (int, bool, string) {
				status, response := checkWav2LipDependency(cfg, "")
				return status, response.Available, response.Content
			},
		},
	}
	go runIntegrationDependencyChecks(ctx, checks)
}

func runIntegrationDependencyChecks(ctx context.Context, checks []integrationDependencyCheck) {
	var workers sync.WaitGroup
	for _, check := range checks {
		check := check
		workers.Add(1)
		go func() {
			defer workers.Done()
			select {
			case <-ctx.Done():
				return
			default:
			}
			status, available, content := check.check()
			if status == 200 && available {
				log.Printf("dependency preflight: %s is available", check.name)
				return
			}
			if content == "" {
				content = "检查失败"
			}
			log.Printf("dependency preflight: %s is unavailable: %s", check.name, content)
		}()
	}
	workers.Wait()
}
