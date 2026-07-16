package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	browserLauncherScriptName     = "browser_launcher.sh"
	browserLauncherScriptTimeout  = 45 * time.Second
	browserWSLForceRecreateEnv    = "DEEPRIGHT_BROWSER_WSL_FORCE_RECREATE"
	browserEmbeddedLauncherScript = `#!/bin/sh

set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname "$0")" && pwd)

BROWSER_BIN="${DEEPRIGHT_BROWSER_BIN:-}"
if [ -z "$BROWSER_BIN" ]; then
  for candidate in "$SCRIPT_DIR/browser" "$SCRIPT_DIR/../browser"; do
    if [ -x "$candidate" ]; then
      BROWSER_BIN="$candidate"
      break
    fi
  done
fi

if [ -z "$BROWSER_BIN" ]; then
  printf '{"status":1,"message":"browser binary not found beside launcher"}\n'
  exit 1
fi

exec "$BROWSER_BIN" __wsl-instance acquire "$@"`
)

type browserWSLLauncherResponse struct {
	Status      int    `json:"status"`
	PID         any    `json:"pid"`
	Port        any    `json:"port"`
	URL         string `json:"url"`
	WS          string `json:"ws"`
	HTTP        string `json:"http"`
	UserDataDir string `json:"user-data-dir"`
	Message     string `json:"message"`
}

type browserWSLLauncherRunResult struct {
	Response browserWSLLauncherResponse
	Request  string
	Stdout   string
	Stderr   string
}

type browserWSLInitRuntimeConfig struct {
	Browser struct {
		InitTimeout *int `json:"init_timeout"`
	} `json:"browser"`
}

var (
	browserResolveLauncherScriptPathFn = browserResolveLauncherScriptPath
	browserLauncherCommandContextFn    = exec.CommandContext
)

func browserResolveLauncherScriptPath(flags map[string]string) (string, error) {
	root, _, err := browserRuntimeRoot(flags)
	if err != nil {
		return "", err
	}
	candidates := []string{
		filepath.Join(root, browserLauncherScriptName),
		filepath.Join(root, "instance", browserLauncherScriptName),
	}
	for _, candidate := range candidates {
		info, statErr := os.Stat(candidate)
		if statErr != nil || info.IsDir() {
			continue
		}
		if absPath, absErr := filepath.Abs(candidate); absErr == nil {
			return absPath, nil
		}
		return candidate, nil
	}
	materialized := filepath.Join(root, browserLauncherScriptName)
	if err := browserWriteFileFn(materialized, []byte(browserEmbeddedLauncherScript+"\n"), 0o755); err != nil {
		return "", fmt.Errorf("browser launcher script not found and could not be recreated: %w", err)
	}
	if absPath, absErr := filepath.Abs(materialized); absErr == nil {
		return absPath, nil
	}
	return materialized, nil
}

func browserRunWSLLauncher(flags map[string]string, agentID, chatID string, headless, forceRecreate bool, chromePath string) (browserWSLLauncherRunResult, error) {
	scriptPath, err := browserResolveLauncherScriptPathFn(flags)
	if err != nil {
		return browserWSLLauncherRunResult{}, err
	}
	timeout := browserLauncherScriptTimeout
	if forceRecreate {
		timeout = browserWSLInitTimeoutFromRuntimeConfig(flags, timeout)
	}
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := []string{
		scriptPath,
		"--agentId", agentID,
		"--chatId", chatID,
		"--headless", strconv.FormatBool(headless),
	}
	if trimmedChromePath := strings.TrimSpace(chromePath); trimmedChromePath != "" {
		args = append(args, "--chrome", trimmedChromePath)
	}
	requestText := browserFormatCommandLine("/bin/bash", args)
	browserCreateTrace("instance.wsl.launcher.request", map[string]any{
		"agentId":       agentID,
		"chatId":        chatID,
		"request":       requestText,
		"scriptPath":    scriptPath,
		"headless":      headless,
		"forceRecreate": forceRecreate,
		"timeout":       timeout.String(),
	})

	cmd := browserLauncherCommandContextFn(ctx, "/bin/bash", args...)
	cmd.Dir = filepath.Dir(scriptPath)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = browserWSLLauncherEnvironment(forceRecreate)
	if root, _, rootErr := browserRuntimeRoot(flags); rootErr == nil {
		if browserBin := browserResolveLauncherBinaryPath(root); browserBin != "" {
			cmd.Env = append(cmd.Env, "DEEPRIGHT_BROWSER_BIN="+browserBin)
		}
	}

	runErr := cmd.Run()
	stdoutText := strings.TrimSpace(stdout.String())
	stderrText := strings.TrimSpace(stderr.String())
	payload := bytes.TrimSpace(stdout.Bytes())
	if len(payload) == 0 {
		browserCreateTrace("instance.wsl.launcher.response", map[string]any{
			"agentId":  agentID,
			"chatId":   chatID,
			"request":  requestText,
			"response": stdoutText,
			"stderr":   stderrText,
			"error":    browserWSLLauncherRunErrorText(runErr),
		})
		switch {
		case stderrText != "":
			return browserWSLLauncherRunResult{}, fmt.Errorf("browser launcher returned no JSON response: %s", stderrText)
		case runErr != nil:
			return browserWSLLauncherRunResult{}, fmt.Errorf("browser launcher returned no JSON response: %w", runErr)
		default:
			return browserWSLLauncherRunResult{}, errors.New("browser launcher returned no JSON response")
		}
	}

	var response browserWSLLauncherResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		browserCreateTrace("instance.wsl.launcher.response", map[string]any{
			"agentId":  agentID,
			"chatId":   chatID,
			"request":  requestText,
			"response": stdoutText,
			"stderr":   stderrText,
			"error":    err.Error(),
		})
		return browserWSLLauncherRunResult{}, fmt.Errorf("parse browser launcher response: %w", err)
	}
	browserCreateTrace("instance.wsl.launcher.response", map[string]any{
		"agentId":  agentID,
		"chatId":   chatID,
		"request":  requestText,
		"response": stdoutText,
		"stderr":   stderrText,
		"error":    browserWSLLauncherRunErrorText(runErr),
		"status":   response.Status,
		"pid":      browserWSLLauncherValueText(response.PID),
		"port":     browserWSLLauncherValueText(response.Port),
		"cdp":      browserWSLLauncherResponseCDP(response),
	})
	if response.Status != 0 {
		message := strings.TrimSpace(response.Message)
		if message == "" {
			if runErr != nil {
				message = runErr.Error()
			} else {
				message = "browser launcher failed"
			}
		}
		return browserWSLLauncherRunResult{
			Response: response,
			Request:  requestText,
			Stdout:   stdoutText,
			Stderr:   stderrText,
		}, errors.New(message)
	}
	if runErr != nil {
		return browserWSLLauncherRunResult{
			Response: response,
			Request:  requestText,
			Stdout:   stdoutText,
			Stderr:   stderrText,
		}, fmt.Errorf("browser launcher exited unexpectedly: %w", runErr)
	}
	return browserWSLLauncherRunResult{
		Response: response,
		Request:  requestText,
		Stdout:   stdoutText,
		Stderr:   stderrText,
	}, nil
}

// browserWSLInitTimeoutFromRuntimeConfig reads browser.init_timeout from the
// integration config.json. The value is a positive number of seconds.
func browserWSLInitTimeoutFromRuntimeConfig(flags map[string]string, fallback time.Duration) time.Duration {
	if fallback <= 0 {
		fallback = 45 * time.Second
	}
	runtimePath, ok, err := browserResolveRuntimeConfigPath(flags)
	if err != nil || !ok {
		return fallback
	}
	data, err := browserReadFileFn(runtimePath)
	if err != nil {
		return fallback
	}
	var cfg browserWSLInitRuntimeConfig
	if err := json.Unmarshal(data, &cfg); err != nil || cfg.Browser.InitTimeout == nil || *cfg.Browser.InitTimeout <= 0 {
		return fallback
	}
	return time.Duration(*cfg.Browser.InitTimeout) * time.Second
}

func browserWSLLauncherEnvironment(forceRecreate bool) []string {
	env := make([]string, 0, len(os.Environ())+1)
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, browserWSLForceRecreateEnv+"=") {
			env = append(env, value)
		}
	}
	if forceRecreate {
		env = append(env, browserWSLForceRecreateEnv+"=1")
	}
	return env
}

func browserResolveLauncherBinaryPath(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return ""
	}
	for _, candidate := range []string{
		filepath.Join(root, "browser"),
		filepath.Join(root, "browser.exe"),
		filepath.Join(root, "..", "browser"),
		filepath.Join(root, "..", "browser.exe"),
	} {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		if absPath, absErr := filepath.Abs(candidate); absErr == nil {
			return absPath
		}
		return candidate
	}
	return ""
}

func browserAcquireWSLManagedInstanceWithLauncher(flags map[string]string, agentID, chatID string, headless, forceRecreate bool, chromePath string) (browserWSLInstanceRecord, error) {
	result, err := browserRunWSLLauncher(flags, agentID, chatID, headless, forceRecreate, chromePath)
	if err != nil {
		return browserWSLInstanceRecord{}, err
	}
	response := result.Response
	port, err := browserWSLLauncherValueInt(response.Port)
	if err != nil {
		return browserWSLInstanceRecord{}, fmt.Errorf("invalid browser launcher port: %w", err)
	}
	pid, err := browserWSLLauncherValueInt(response.PID)
	if err != nil {
		return browserWSLInstanceRecord{}, fmt.Errorf("invalid browser launcher pid: %w", err)
	}
	if pid <= 0 {
		if lookupPID, ok := browserWSLInstanceLookupPIDByPort(port); ok {
			pid = lookupPID
		}
	}
	cdp := browserWSLLauncherResponseCDP(response)
	if cdp == "" {
		return browserWSLInstanceRecord{}, errors.New("browser launcher returned empty cdp url")
	}
	httpBase := strings.TrimSpace(response.HTTP)
	if httpBase == "" && port > 0 {
		httpBase = fmt.Sprintf("http://localhost:%d", port)
	}
	return browserWSLInstanceRecord{
		AgentID:     browserNormalizeIdentityPart(agentID),
		ChatID:      browserNormalizeIdentityPart(chatID),
		PID:         pid,
		Port:        port,
		WS:          cdp,
		HTTP:        httpBase,
		UserDataDir: strings.TrimSpace(response.UserDataDir),
	}, nil
}

func browserWSLLauncherRunErrorText(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}

func browserWSLLauncherResponseCDP(response browserWSLLauncherResponse) string {
	if ws := strings.TrimSpace(response.WS); ws != "" {
		return ws
	}
	return strings.TrimSpace(response.URL)
}

func browserWSLLauncherValueText(value any) string {
	switch item := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(item)
	case float64:
		return strconv.Itoa(int(item))
	default:
		return strings.TrimSpace(fmt.Sprint(item))
	}
}

func browserWSLLauncherValueInt(value any) (int, error) {
	text := browserWSLLauncherValueText(value)
	if text == "" {
		return 0, nil
	}
	number, err := strconv.Atoi(text)
	if err != nil {
		return 0, err
	}
	return number, nil
}
