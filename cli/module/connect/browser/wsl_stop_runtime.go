package main

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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

	entries, err := browserReadDirFn(rootUnix)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			browserLogWSLLifecycleEvent("browser_stop_wsl_programdata", "complete", map[string]any{
				"rootDir":      rootWin,
				"removedCount": 0,
			}, nil)
			return nil
		}
		return err
	}

	targets := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if !strings.HasPrefix(strings.ToLower(name), "chrome") {
			continue
		}
		targets = append(targets, name)
	}
	sort.Strings(targets)

	fields := map[string]any{
		"rootDir":     rootWin,
		"targetCount": len(targets),
	}
	if len(targets) > 0 {
		paths := make([]string, 0, len(targets))
		for _, name := range targets {
			paths = append(paths, filepath.Join(rootWin, name))
		}
		fields["targets"] = paths
	}
	browserLogWSLLifecycleEvent("browser_stop_wsl_programdata", "begin", fields, nil)

	if len(targets) == 0 {
		browserLogWSLLifecycleEvent("browser_stop_wsl_programdata", "complete", map[string]any{
			"rootDir":      rootWin,
			"removedCount": 0,
		}, nil)
		return nil
	}

	removed := make([]string, 0, len(targets))
	removedMu := sync.Mutex{}
	wg := sync.WaitGroup{}
	for _, name := range targets {
		name := name
		wg.Add(1)
		go func() {
			defer wg.Done()

			targetUnix := filepath.Join(rootUnix, name)
			targetWin := filepath.Join(rootWin, name)
			browserLogWSLLifecycleEvent("browser_stop_wsl_programdata", "remove_begin", map[string]any{
				"rootDir":    rootWin,
				"targetDir":  targetWin,
				"targetName": name,
			}, nil)
			if err := browserRemoveAllFn(targetUnix); err != nil && !errors.Is(err, os.ErrNotExist) {
				browserLogWSLLifecycleEvent("browser_stop_wsl_programdata", "remove_error", map[string]any{
					"rootDir":    rootWin,
					"targetDir":  targetWin,
					"targetName": name,
				}, err)
				return
			}
			removedMu.Lock()
			removed = append(removed, targetWin)
			removedMu.Unlock()
			browserLogWSLLifecycleEvent("browser_stop_wsl_programdata", "remove_ok", map[string]any{
				"rootDir":    rootWin,
				"targetDir":  targetWin,
				"targetName": name,
			}, nil)
		}()
	}
	wg.Wait()
	sort.Strings(removed)

	completeFields := map[string]any{
		"rootDir":      rootWin,
		"targetCount":  len(targets),
		"removedCount": len(removed),
		"failedCount":  len(targets) - len(removed),
	}
	if len(removed) > 0 {
		completeFields["removed"] = removed
	}
	browserLogWSLLifecycleEvent("browser_stop_wsl_programdata", "complete", completeFields, nil)
	return nil
}
