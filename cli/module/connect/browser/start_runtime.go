package main

import (
	"strings"
	"time"
)

func browserMaybeCleanupStopUserData(flags map[string]string) {
	if err := browserCleanupStopUserData(flags); err != nil {
		browserLogAsyncLifecycleEvent("browser_stop_cleanup_user_data", "cleanup_error", nil, 0, err)
	}
}

func browserCleanupStopUserData(flags map[string]string) error {
	if isWSL, err := browserWSLDetectFn(); err == nil && isWSL {
		return nil
	}
	_, err := browserDestroyCleanupAllAgentUserDataDirs(flags)
	return err
}

func browserMaybeShutdownDefaultBootstrapCDP(flags map[string]string) {
	if err := browserShutdownDefaultBootstrapCDP(flags); err != nil {
		browserLogAsyncLifecycleEvent("browser_stop_bootstrap_cdp", "cleanup_error", nil, 0, err)
	}
}

func browserLogAsyncLifecycleEvent(event, stage string, args []string, pid int, err error) {
	fields := map[string]any{
		"event":     strings.TrimSpace(event),
		"stage":     strings.TrimSpace(stage),
		"timestamp": browserNowFn().Format(time.RFC3339Nano),
	}
	if len(args) > 0 {
		fields["args"] = append([]string{}, args...)
	}
	if pid > 0 {
		fields["pid"] = pid
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	browserAppendLogJSON(fields)
}

func browserAppendLogJSON(payload map[string]any) {
	logPath, err := browserDefaultLogPath()
	if err != nil {
		return
	}
	writer, err := browserLogWriter(logPath)
	if err != nil {
		return
	}
	defer writer.Close()

	_, _ = writer.Write([]byte(browserRenderLogLine(payload) + "\n"))
}
