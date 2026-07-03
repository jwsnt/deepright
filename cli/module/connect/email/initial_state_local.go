package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connect/connectsvc"
	"connect/emailsvc"
)

const initialTimelineScanMultiplierLocal = 2

type localInitialState struct {
	LastTimelineUnix int64            `json:"lastTimelineUnix"`
	Processed        map[string]int64 `json:"processed"`
	ProcessedUIDs    map[string]int64 `json:"processedUIDs,omitempty"`
}

type localMetaGetter interface {
	GetMeta(context.Context, string) (*connectsvc.Meta, error)
}

func ensureInitialScanStateForFlagsLocal(flags map[string]string, getter localMetaGetter) error {
	stateFile := stateFileForLogLocal(connectsvc.FirstValue(flags, "log-file"))
	if info, err := os.Stat(stateFile); err == nil {
		if info.IsDir() {
			return errors.New("state file path is a directory")
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return ensureInitialScanStateLocal(ctx, getter, connectsvc.FirstValue(flags, "name"), stateFile, time.Now)
}

func ensureInitialScanStateLocal(ctx context.Context, getter localMetaGetter, name, stateFile string, nowFn func() time.Time) error {
	stateFile = strings.TrimSpace(stateFile)
	if stateFile == "" {
		return errors.New("state file is required")
	}
	if getter == nil {
		return errors.New("meta getter is required")
	}
	if nowFn == nil {
		nowFn = time.Now
	}
	if info, err := os.Stat(stateFile); err == nil {
		if info.IsDir() {
			return errors.New("state file path is a directory")
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	scanSeconds, err := loadInitialScanSecondsLocal(ctx, getter, name)
	if err != nil {
		return err
	}
	state := localInitialState{
		LastTimelineUnix: nowFn().Add(-initialTimelineWindowLocal(scanSeconds)).Unix(),
		Processed:        map[string]int64{},
	}
	if err := os.MkdirAll(filepath.Dir(stateFile), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmpFile := stateFile + ".tmp"
	if err := os.WriteFile(tmpFile, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpFile, stateFile)
}

func loadInitialScanSecondsLocal(ctx context.Context, getter localMetaGetter, name string) (int, error) {
	name = firstNonEmptyLocal(strings.TrimSpace(name), emailsvc.DefaultName)
	meta, err := getter.GetMeta(ctx, name)
	if err != nil {
		return 0, err
	}
	return initialScanSecondsFromMetaLocal(meta), nil
}

func initialScanSecondsFromMetaLocal(meta *connectsvc.Meta) int {
	if meta == nil {
		return defaultPOP3ScanSecsLocal
	}
	var cfg localMetaConfigCompat
	if err := json.Unmarshal([]byte(meta.Meta), &cfg); err != nil {
		return defaultPOP3ScanSecsLocal
	}
	scanSeconds := cfg.ScanSeconds
	if scanSeconds <= 0 {
		scanSeconds = parseLocalPOP3ScanSeconds(cfg.EmailPOP3Interval, cfg.EmailPOP3Seconds)
	}
	if scanSeconds <= 0 {
		scanSeconds = defaultPOP3ScanSecsLocal
	}
	return scanSeconds
}

func initialTimelineWindowLocal(scanSeconds int) time.Duration {
	if scanSeconds <= 0 {
		scanSeconds = defaultPOP3ScanSecsLocal
	}
	return time.Duration(scanSeconds*initialTimelineScanMultiplierLocal) * time.Second
}
