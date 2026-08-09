//go:build linux

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var errPickerCanceled = errors.New("picker canceled")

func main() {
	path, err := pickWindowsDirectory(context.Background())
	if err != nil {
		if errors.Is(err, errPickerCanceled) {
			os.Exit(1)
		}
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
	fmt.Fprint(os.Stdout, path)
}

func pickWindowsDirectory(ctx context.Context) (string, error) {
	if path, ok, canceled, detail := pickViaNativePicker(ctx); ok {
		return path, nil
	} else if canceled {
		return "", errPickerCanceled
	} else if detail != "" {
		return "", errors.New(detail)
	}
	if path, ok, canceled, detail := pickViaPowerShell(ctx); ok {
		return path, nil
	} else if canceled {
		return "", errPickerCanceled
	} else if detail != "" {
		return "", errors.New(detail)
	}
	return "", errors.New("no picker available")
}

func pickViaNativePicker(ctx context.Context) (string, bool, bool, string) {
	var failures []string
	for _, candidate := range nativePickerCandidates() {
		output, err := runInteropCommand(ctx, candidate)
		if err != nil {
			if isCanceled(err, output) {
				return "", false, true, ""
			}
			failures = append(failures, fmt.Sprintf("native picker %q failed: %v output=%q", candidate, err, strings.TrimSpace(string(output))))
			continue
		}
		path := strings.TrimSpace(string(output))
		if path != "" {
			return path, true, false, ""
		}
		failures = append(failures, fmt.Sprintf("native picker %q returned empty output", candidate))
	}
	return "", false, false, strings.Join(failures, "; ")
}

func pickViaPowerShell(ctx context.Context) (string, bool, bool, string) {
	script := windowsPowerShellPickerScript(defaultWindowsPickerDirectory())
	var failures []string
	for _, candidate := range powerShellCandidates() {
		output, err := runInteropCommand(ctx, candidate, "-NoProfile", "-STA", "-Command", script)
		if err != nil {
			if isCanceled(err, output) {
				return "", false, true, ""
			}
			failures = append(failures, fmt.Sprintf("powershell %q failed: %v output=%q", candidate, err, strings.TrimSpace(string(output))))
			continue
		}
		path := strings.TrimSpace(string(output))
		if path != "" {
			return path, true, false, ""
		}
		failures = append(failures, fmt.Sprintf("powershell %q returned empty output", candidate))
	}
	return "", false, false, strings.Join(failures, "; ")
}

func nativePickerCandidates() []string {
	var candidates []string
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		for _, existing := range candidates {
			if existing == path {
				return
			}
		}
		candidates = append(candidates, path)
	}
	if executable, err := os.Executable(); err == nil && strings.TrimSpace(executable) != "" {
		add(filepath.Join(filepath.Dir(executable), "CLI_SANDBOX_PICKER.exe"))
	}
	if path, err := exec.LookPath("CLI_SANDBOX_PICKER.exe"); err == nil {
		add(path)
	}
	return candidates
}

func powerShellCandidates() []string {
	var candidates []string
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		for _, existing := range candidates {
			if existing == path {
				return
			}
		}
		candidates = append(candidates, path)
	}
	for _, candidate := range []string{"powershell.exe", "pwsh.exe"} {
		if path, err := exec.LookPath(candidate); err == nil {
			add(path)
		}
	}
	add("/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe")
	add("/mnt/c/Program Files/PowerShell/7/pwsh.exe")
	add("/mnt/c/Program Files (x86)/PowerShell/7/pwsh.exe")
	return candidates
}

func runInteropCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	stdoutFile, err := os.CreateTemp("", "cli-sandbox-interop-stdout-*")
	if err != nil {
		return nil, err
	}
	stdoutPath := stdoutFile.Name()
	defer os.Remove(stdoutPath)
	defer stdoutFile.Close()

	stderrFile, err := os.CreateTemp("", "cli-sandbox-interop-stderr-*")
	if err != nil {
		return nil, err
	}
	stderrPath := stderrFile.Name()
	defer os.Remove(stderrPath)
	defer stderrFile.Close()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile
	runErr := cmd.Run()

	stdoutFile.Sync()
	stderrFile.Sync()
	stdoutData, _ := os.ReadFile(stdoutPath)
	stderrData, _ := os.ReadFile(stderrPath)
	if len(stderrData) > 0 {
		if len(stdoutData) == 0 {
			stdoutData = stderrData
		} else {
			stdoutData = append(append(stdoutData, '\n'), stderrData...)
		}
	}
	return stdoutData, runErr
}

func isCanceled(err error, output []byte) bool {
	if err == nil {
		return false
	}
	if strings.TrimSpace(string(output)) != "" {
		return false
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}
	return exitErr.ExitCode() == 1
}
