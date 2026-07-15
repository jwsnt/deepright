package main

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	browserWSLChromeDefRoot = `C:\ProgramData\deepright\chrome_def`
)

var (
	browserPrepareWSLChromeDefForStartFn     = browserPrepareWSLChromeDefForStart
	browserRestartIntegrationAfterWSLStartFn = browserRestartIntegrationAfterWSLStart
	browserWSLCommandCombinedOutputFn        = browserRunWSLCommandCombinedOutput
	browserWSLIntegrationStartFn             = browserRunIntegrationStart
)

func browserPrepareWSLChromeDefForStart(flags map[string]string) (bool, error) {
	isWSL, err := browserWSLDetectFn()
	if err != nil {
		browserLogWSLLifecycleEvent("browser_start_wsl_chrome_def", "detect_error", nil, err)
		return false, err
	}
	if !isWSL {
		return false, nil
	}

	browserLogWSLLifecycleEvent("browser_start_wsl_chrome_def", "begin", nil, nil)

	terminated, err := browserTerminateAllWSLChromeProcesses()
	if err != nil {
		browserLogWSLLifecycleEvent("browser_start_wsl_chrome_def", "terminate_all_error", nil, err)
		return false, err
	}

	sourceWin, sourceUnix, err := browserResolveWSLSystemChromeUserDataDir()
	if err != nil {
		browserLogWSLLifecycleEvent("browser_start_wsl_chrome_def", "resolve_source_error", map[string]any{
			"terminated": terminated,
		}, err)
		return false, err
	}

	targetWin, targetUnix, err := browserResolveWSLChromeDefTargetDir()
	if err != nil {
		browserLogWSLLifecycleEvent("browser_start_wsl_chrome_def", "resolve_target_error", map[string]any{
			"terminated": terminated,
			"sourceDir":  sourceWin,
		}, err)
		return false, err
	}

	if err := browserRemoveAllFn(targetUnix); err != nil {
		browserLogWSLLifecycleEvent("browser_start_wsl_chrome_def", "reset_target_error", map[string]any{
			"terminated": terminated,
			"sourceDir":  sourceWin,
			"targetDir":  targetWin,
		}, err)
		return false, err
	}
	if err := browserMkdirAllFn(filepath.Dir(targetUnix), 0o755); err != nil {
		browserLogWSLLifecycleEvent("browser_start_wsl_chrome_def", "mkdir_parent_error", map[string]any{
			"terminated": terminated,
			"sourceDir":  sourceWin,
			"targetDir":  targetWin,
		}, err)
		return false, err
	}

	copyStats, err := browserCopyDirectoryWithProgress(
		sourceUnix,
		targetUnix,
		browserInstanceUserDataProgressTracer("browser_start_wsl_chrome_def.copy.progress", map[string]any{
			"sourceDir": sourceWin,
			"targetDir": targetWin,
		}),
		browserInstanceUserDataSkipTracer("browser_start_wsl_chrome_def.copy.skip", map[string]any{
			"sourceDir": sourceWin,
			"targetDir": targetWin,
		}),
	)
	if err != nil {
		browserLogWSLLifecycleEvent("browser_start_wsl_chrome_def", "copy_error", map[string]any{
			"terminated": terminated,
			"sourceDir":  sourceWin,
			"targetDir":  targetWin,
		}, err)
		return false, err
	}

	if err := browserWSLCleanupChromeUserDataFn(targetWin); err != nil {
		browserLogWSLLifecycleEvent("browser_start_wsl_chrome_def", "cleanup_error", map[string]any{
			"terminated": terminated,
			"sourceDir":  sourceWin,
			"targetDir":  targetWin,
			"files":      copyStats.Files,
			"dirs":       copyStats.Dirs,
			"bytes":      copyStats.Bytes,
		}, err)
		return false, err
	}

	browserLogWSLLifecycleEvent("browser_start_wsl_chrome_def", "ready", map[string]any{
		"terminated": terminated,
		"sourceDir":  sourceWin,
		"targetDir":  targetWin,
		"files":      copyStats.Files,
		"dirs":       copyStats.Dirs,
		"bytes":      copyStats.Bytes,
	}, nil)
	return true, nil
}

func browserRestartIntegrationAfterWSLStart(flags map[string]string) error {
	isWSL, err := browserWSLDetectFn()
	if err != nil {
		browserLogWSLLifecycleEvent("browser_start_wsl_integration", "detect_error", nil, err)
		return err
	}
	if !isWSL {
		return nil
	}

	browserLogWSLLifecycleEvent("browser_start_wsl_integration", "begin", nil, nil)
	if err := browserWSLIntegrationStartFn(flags); err != nil {
		browserLogWSLLifecycleEvent("browser_start_wsl_integration", "restart_error", nil, err)
		return err
	}
	browserLogWSLLifecycleEvent("browser_start_wsl_integration", "ready", nil, nil)
	return nil
}

func browserTerminateAllWSLChromeProcesses() (int, error) {
	script := `$ErrorActionPreference = 'Stop'
$items = @(Get-Process chrome -ErrorAction SilentlyContinue)
if ($items.Count -eq 0) {
  [Console]::Out.WriteLine('0')
  exit 0
}
$ids = @($items | Select-Object -ExpandProperty Id)
Stop-Process -Id $ids -Force -ErrorAction Stop
$deadline = (Get-Date).AddSeconds(10)
do {
  $alive = @()
  foreach ($id in $ids) {
    if ($null -ne (Get-Process -Id $id -ErrorAction SilentlyContinue)) {
      $alive += $id
    }
  }
  if ($alive.Count -eq 0) {
    [Console]::Out.WriteLine($ids.Count)
    exit 0
  }
  Start-Sleep -Milliseconds 250
} while ((Get-Date) -lt $deadline)
throw ('chrome processes still running: ' + ($alive -join ','))
`
	output, err := browserWSLCommandCombinedOutputFn(
		browserWSLPowerShellPath,
		"-NoProfile",
		"-NonInteractive",
		"-Command",
		script,
	)
	if err != nil {
		message := strings.TrimSpace(strings.ReplaceAll(string(output), "\r", ""))
		if message == "" {
			return 0, fmt.Errorf("terminate all chrome processes failed: %w", err)
		}
		return 0, fmt.Errorf("terminate all chrome processes failed: %w: %s", err, message)
	}
	text := strings.TrimSpace(strings.ReplaceAll(string(output), "\r", ""))
	if text == "" {
		return 0, nil
	}
	count, err := strconv.Atoi(text)
	if err != nil {
		return 0, nil
	}
	return count, nil
}

func browserResolveWSLSystemChromeUserDataDir() (string, string, error) {
	script := `$ErrorActionPreference = 'Stop'
$localAppData = [Environment]::GetFolderPath('LocalApplicationData')
$target = Join-Path $localAppData 'Google\Chrome\User Data'
if (-not (Test-Path -LiteralPath $target -PathType Container)) {
  throw 'windows chrome user data dir not found'
}
[Console]::Out.WriteLine($target)
`
	output, err := browserWSLCommandCombinedOutputFn(
		browserWSLPowerShellPath,
		"-NoProfile",
		"-NonInteractive",
		"-Command",
		script,
	)
	if err != nil {
		message := strings.TrimSpace(strings.ReplaceAll(string(output), "\r", ""))
		if message == "" {
			return "", "", fmt.Errorf("resolve windows chrome user data dir failed: %w", err)
		}
		return "", "", fmt.Errorf("resolve windows chrome user data dir failed: %w: %s", err, message)
	}
	windowsPath := strings.TrimSpace(strings.ReplaceAll(string(output), "\r", ""))
	if windowsPath == "" {
		return "", "", errors.New("windows chrome user data dir is empty")
	}
	unixPath, err := browserWSLPathUnixFn(windowsPath)
	if err != nil {
		return "", "", err
	}
	return strings.ReplaceAll(windowsPath, "/", `\`), strings.TrimSpace(unixPath), nil
}

func browserResolveWSLChromeDefTargetDir() (string, string, error) {
	unixPath, err := browserWSLPathUnixFn(browserWSLChromeDefRoot)
	if err != nil {
		return "", "", err
	}
	return browserWSLChromeDefRoot, strings.TrimSpace(unixPath), nil
}

func browserRunIntegrationStart(flags map[string]string) error {
	exe, err := browserResolveIntegrationStartBinary(flags)
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "start")
	cmd.Dir = filepath.Dir(exe)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(strings.ReplaceAll(string(output), "\r", ""))
		if message == "" {
			return fmt.Errorf("run integration start failed: %w", err)
		}
		return fmt.Errorf("run integration start failed: %w: %s", err, message)
	}
	return nil
}

func browserResolveIntegrationStartBinary(flags map[string]string) (string, error) {
	if connectBin, ok, err := browserReadRecordedConnectBin(); err != nil {
		return "", err
	} else if ok {
		return connectBin, nil
	}

	browserPath, err := browserExecutablePathFn()
	if err != nil {
		return "", err
	}
	if absPath, absErr := filepath.Abs(browserPath); absErr == nil {
		browserPath = absPath
	}

	for _, candidate := range []string{
		filepath.Join(filepath.Dir(browserPath), "..", "integration"),
		filepath.Join(filepath.Dir(browserPath), "integration"),
	} {
		if resolved, ensureErr := browserEnsureExecutablePath(candidate); ensureErr == nil {
			return resolved, nil
		}
	}
	return "", errors.New("integration binary not found; start browser with --connect-bin first or run beside integration")
}

func browserRunWSLCommandCombinedOutput(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}
