package main

import (
	"path/filepath"
	"strings"
)

var (
	browserMaybeCleanupWSLProgramDataChromeDirsFn = browserMaybeCleanupWSLProgramDataChromeDirs
)

func browserMaybeCleanupWSLProgramDataChromeDirs() {
	if err := browserCleanupWSLProgramDataChromeDirs(); err != nil {
		browserLogWSLLifecycleEvent("browser_stop_wsl_programdata", "cleanup_error", nil, err)
	}
}

func browserCleanupWSLProgramDataChromeDirs() error {
	isWSL, err := browserWSLDetectFn()
	if err != nil {
		return err
	}
	if !isWSL {
		return nil
	}

	rootUnix := strings.TrimSpace(browserWSLInstanceProfileRoot)
	rootWin := strings.TrimSpace(browserWSLInstanceProfileRootW)
	if rootUnix == "" || rootWin == "" {
		return nil
	}

	const targetName = "chrome_def"
	targetUnix := filepath.Join(rootUnix, targetName)
	targetWin := filepath.Join(rootWin, targetName)
	fields := map[string]any{
		"rootDir":    rootWin,
		"targetDir":  targetWin,
		"targetName": targetName,
	}
	browserLogWSLLifecycleEvent("browser_stop_wsl_programdata", "begin", fields, nil)

	browserLogWSLLifecycleEvent("browser_stop_wsl_programdata", "remove_begin", fields, nil)
	if err := browserRemoveAllFn(targetUnix); err != nil {
		browserLogWSLLifecycleEvent("browser_stop_wsl_programdata", "remove_error", fields, err)
		return nil
	}
	browserLogWSLLifecycleEvent("browser_stop_wsl_programdata", "remove_ok", fields, nil)
	browserLogWSLLifecycleEvent("browser_stop_wsl_programdata", "complete", map[string]any{
		"rootDir":        rootWin,
		"targetDir":      targetWin,
		"targetName":     targetName,
		"removedCount":   1,
		"preservedGlobs": []string{"chrome_*"},
	}, nil)
	return nil
}
