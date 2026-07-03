package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"connect/browserplaywrightsvc"

	"github.com/playwright-community/playwright-go"
)

const browserPlaywrightDriverPreflightTimeout = 90 * time.Second

type browserPlaywrightDriverPreflightResult struct {
	DriverDir string
	Status    string
	Output    string
	Err       error
	Elapsed   time.Duration
}

var (
	browserPlaywrightInstallFn = func(opts *playwright.RunOptions) error {
		return playwright.Install(opts)
	}
	browserPlaywrightInstallCommandContextFn = exec.CommandContext
	browserEnsurePlaywrightDriverForStartFn  = browserEnsurePlaywrightDriverForStart
)

func browserEnsurePlaywrightDriverForStart(flags map[string]string, opts browserplaywrightsvc.Options) browserPlaywrightDriverPreflightResult {
	startedAt := browserNowFn()
	result := browserPlaywrightDriverPreflightResult{DriverDir: strings.TrimSpace(opts.DriverDir)}
	if ready, err := browserplaywrightsvc.PlaywrightDriverReady(result.DriverDir); err != nil {
		result.Status = "check_error"
		result.Err = err
		result.Elapsed = browserNowFn().Sub(startedAt)
		return result
	} else if ready {
		result.Status = "ready"
		result.Elapsed = browserNowFn().Sub(startedAt)
		return result
	}

	selfPath, err := browserResolveSelfPath(flags)
	if err != nil {
		result.Status = "resolve_error"
		result.Err = err
		result.Elapsed = browserNowFn().Sub(startedAt)
		return result
	}

	ctx, cancel := context.WithTimeout(context.Background(), browserPlaywrightDriverPreflightTimeout)
	defer cancel()

	cmd := browserPlaywrightInstallCommandContextFn(ctx, selfPath, "__install-playwright-driver", "--driver-dir", result.DriverDir)
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	if err := cmd.Run(); err != nil {
		result.Output = strings.TrimSpace(combined.String())
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			result.Status = "timeout"
			result.Err = ctx.Err()
		} else {
			result.Status = "error"
			result.Err = err
		}
		result.Elapsed = browserNowFn().Sub(startedAt)
		return result
	}

	result.Output = strings.TrimSpace(combined.String())
	if ready, err := browserplaywrightsvc.PlaywrightDriverReady(result.DriverDir); err != nil {
		result.Status = "check_error"
		result.Err = err
	} else if !ready {
		result.Status = "incomplete"
		result.Err = fmt.Errorf("playwright driver install incomplete: %s", result.DriverDir)
	} else {
		result.Status = "installed"
	}
	result.Elapsed = browserNowFn().Sub(startedAt)
	return result
}

func runPlaywrightDriverInstallCommand(args []string, stdout, stderr io.Writer) int {
	flags, positionals, err := parseCLIArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if len(positionals) > 0 {
		fmt.Fprintln(stderr, "__install-playwright-driver does not accept positional arguments")
		return 1
	}
	driverDir := strings.TrimSpace(flags["driver-dir"])
	if driverDir == "" {
		fmt.Fprintln(stderr, "driver-dir is required")
		return 1
	}
	if err := browserInstallPlaywrightDriver(driverDir); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	fmt.Fprintln(stdout, "OK")
	return 0
}

func browserInstallPlaywrightDriver(driverDir string) error {
	driverDir = strings.TrimSpace(driverDir)
	if driverDir == "" {
		return fmt.Errorf("playwright driver directory is empty")
	}
	absDriverDir, err := filepath.Abs(driverDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(absDriverDir, 0o755); err != nil {
		return err
	}
	if ready, err := browserplaywrightsvc.PlaywrightDriverReady(absDriverDir); err != nil {
		return err
	} else if ready {
		return nil
	}
	if err := browserPlaywrightInstallFn(&playwright.RunOptions{
		DriverDirectory:     absDriverDir,
		SkipInstallBrowsers: true,
	}); err != nil {
		return err
	}
	ready, err := browserplaywrightsvc.PlaywrightDriverReady(absDriverDir)
	if err != nil {
		return err
	}
	if !ready {
		return fmt.Errorf("playwright driver install incomplete: %s", absDriverDir)
	}
	return nil
}

func browserLogPlaywrightDriverPreflightEvent(result browserPlaywrightDriverPreflightResult) {
	payload := map[string]any{
		"event":     "browser_playwright_driver_preflight",
		"timestamp": browserNowFn().Format(time.RFC3339Nano),
		"driverDir": strings.TrimSpace(result.DriverDir),
		"stage":     strings.TrimSpace(result.Status),
	}
	if result.Elapsed > 0 {
		payload["elapsed"] = result.Elapsed.String()
		payload["elapsedMs"] = result.Elapsed.Milliseconds()
	}
	if strings.TrimSpace(result.Output) != "" {
		payload["output"] = strings.TrimSpace(result.Output)
	}
	if result.Err != nil {
		payload["error"] = result.Err.Error()
	}
	browserAppendLogJSON(payload)
}
