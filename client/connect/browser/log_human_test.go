package main

import (
	"strings"
	"testing"
)

func TestBrowserRenderLogLineKeepsFullCommandText(t *testing.T) {
	longArg := strings.Repeat("x", 180)
	command := `"/mnt/c/Program Files/Google/Chrome/Application/chrome.exe" --remote-debugging-port=28443 --user-data-dir="C:\temp\chrome_28443" --flag=` + longArg

	line := browserRenderLogLine(map[string]any{
		"timestamp": "2026-06-16T12:00:00Z",
		"event":     "browser_create_trace",
		"command":   command,
		"stage":     "instance.init.launch",
	})

	if !strings.Contains(line, "命令="+command) {
		t.Fatalf("log line should contain full command, got: %s", line)
	}
	if strings.Contains(line, "...") {
		t.Fatalf("log line should not truncate command, got: %s", line)
	}
}

func TestBrowserRenderLogLineKeepsFullLauncherRequestAndResponse(t *testing.T) {
	request := `/bin/bash /plugin/browser_launcher.sh 60556 false "/mnt/c/Program Files/Google/Chrome/Application/chrome.exe"`
	response := `{"status":1,"pid":"","port":"60556","url":"","message":"CDP not ready after 30s"}`
	stderr := strings.Repeat("e", 180)

	line := browserRenderLogLine(map[string]any{
		"timestamp": "2026-06-16T12:00:00Z",
		"event":     "browser_create_trace",
		"stage":     "instance.wsl.launcher.response",
		"request":   request,
		"response":  response,
		"stderr":    stderr,
	})

	for _, wanted := range []string{
		"请求=" + request,
		"响应=" + response,
		"标准错误=" + stderr,
	} {
		if !strings.Contains(line, wanted) {
			t.Fatalf("log line should contain %q, got: %s", wanted, line)
		}
	}
	if strings.Contains(line, "...") {
		t.Fatalf("log line should not truncate request/response, got: %s", line)
	}
}

func TestBrowserRenderLogLineIncludesCopySourceAndTarget(t *testing.T) {
	line := browserRenderLogLine(map[string]any{
		"timestamp": "2026-07-15T12:00:00Z",
		"event":     "browser_create_trace",
		"stage":     "copy_begin",
		"sourceDir": `/Users/test/Library/Application Support/Google/Chrome`,
		"targetDir": `/tmp/chrome_12345`,
	})

	for _, wanted := range []string{
		`复制来源=/Users/test/Library/Application Support/Google/Chrome`,
		`复制目标=/tmp/chrome_12345`,
	} {
		if !strings.Contains(line, wanted) {
			t.Fatalf("log line should contain %q, got: %s", wanted, line)
		}
	}
}

func TestBrowserRenderLogLineUsesProfileDirAsCopyTarget(t *testing.T) {
	line := browserRenderLogLine(map[string]any{
		"timestamp":  "2026-07-15T12:00:00Z",
		"event":      "browser_create_trace",
		"stage":      "instance.user_data.copy.begin",
		"sourceDir":  `/Users/test/Chrome`,
		"profileDir": `/tmp/chrome_ab12`,
	})

	for _, wanted := range []string{
		`复制来源=/Users/test/Chrome`,
		`复制目标=/tmp/chrome_ab12`,
	} {
		if !strings.Contains(line, wanted) {
			t.Fatalf("log line should contain %q, got: %s", wanted, line)
		}
	}
}
