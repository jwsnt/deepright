package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIntegrationCommandLookPathPrefersUserEnvironment(t *testing.T) {
	directory := t.TempDir()
	command := filepath.Join(directory, "voxcpm")
	if err := os.WriteFile(command, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	previousPath, previousLookPath := integrationUserEnvironmentPathFn, integrationServiceLookPathFn
	integrationUserEnvironmentPathFn = func() string { return directory }
	integrationServiceLookPathFn = func(string) (string, error) { return "", exec.ErrNotFound }
	t.Cleanup(func() {
		integrationUserEnvironmentPathFn, integrationServiceLookPathFn = previousPath, previousLookPath
	})

	path, err := integrationCommandLookPath("voxcpm")
	if err != nil || path != command {
		t.Fatalf("user-environment command = %q, %v, want %q, nil", path, err, command)
	}
}

func TestIntegrationCommandLookPathFallsBackToServiceEnvironment(t *testing.T) {
	previousPath, previousLookPath := integrationUserEnvironmentPathFn, integrationServiceLookPathFn
	integrationUserEnvironmentPathFn = func() string { return "" }
	integrationServiceLookPathFn = func(name string) (string, error) {
		if name != "ffmpeg" {
			t.Fatalf("service lookup name = %q", name)
		}
		return "/usr/local/bin/ffmpeg", nil
	}
	t.Cleanup(func() {
		integrationUserEnvironmentPathFn, integrationServiceLookPathFn = previousPath, previousLookPath
	})

	path, err := integrationCommandLookPath("ffmpeg")
	if err != nil || path != "/usr/local/bin/ffmpeg" {
		t.Fatalf("service-environment command = %q, %v", path, err)
	}
}
