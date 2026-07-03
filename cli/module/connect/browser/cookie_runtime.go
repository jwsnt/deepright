package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type browserCookieFileConfig struct {
	Path       string
	Configured bool
	FromMeta   bool
}

func runCookieCommand(command string, args []string, stdout, stderr io.Writer) int {
	flags, positionals, err := parseCLIArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if len(positionals) > 0 {
		fmt.Fprintf(stderr, "%s does not accept positional arguments\n", command)
		return 1
	}
	if err := browserRejectRemovedFlags(flags); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	switch command {
	case "fetch":
		if err := browserRunCookieFetch(flags, stdout); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		return 0
	case "store":
		if err := browserRunCookieStore(flags, stdout); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		return 0
	default:
		fmt.Fprintf(stderr, "unknown cookie command: %s\n", command)
		return 1
	}
}

func browserValidateStartCookieSupport(flags map[string]string) error {
	if _, err := browserEnsureCookieFile(flags); err != nil {
		return fmt.Errorf("browser cookie store validation failed: %w", err)
	}
	if _, err := browserReadCookieFile(flags); err != nil {
		return fmt.Errorf("browser cookie fetch validation failed: %w", err)
	}
	return nil
}

func browserRunCookieStore(flags map[string]string, stdout io.Writer) error {
	if _, err := browserEnsureCookieFile(flags); err != nil {
		return err
	}
	_, err := io.WriteString(stdout, "OK\n")
	return err
}

func browserRunCookieFetch(flags map[string]string, stdout io.Writer) error {
	data, err := browserReadCookieFile(flags)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		data = []byte("[]")
	}
	if data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	_, err = stdout.Write(data)
	return err
}

func browserEnsureCookieFile(flags map[string]string) (browserCookieFileConfig, error) {
	cfg, err := browserResolveCookieFile(flags)
	if err != nil {
		return browserCookieFileConfig{}, err
	}
	if !cfg.Configured {
		return cfg, nil
	}
	info, err := os.Stat(cfg.Path)
	if err == nil {
		if info.IsDir() {
			return browserCookieFileConfig{}, fmt.Errorf("cookie_path must be a file, got directory: %s", cfg.Path)
		}
		if _, err := browserReadCookieFile(flags); err != nil {
			return browserCookieFileConfig{}, err
		}
		return cfg, nil
	}
	if !os.IsNotExist(err) {
		return browserCookieFileConfig{}, err
	}
	if err := os.MkdirAll(filepath.Dir(cfg.Path), 0o755); err != nil {
		return browserCookieFileConfig{}, err
	}
	if err := os.WriteFile(cfg.Path, []byte("[]\n"), 0o644); err != nil {
		return browserCookieFileConfig{}, err
	}
	return cfg, nil
}

func browserReadCookieFile(flags map[string]string) ([]byte, error) {
	cfg, err := browserResolveCookieFile(flags)
	if err != nil {
		return nil, err
	}
	if !cfg.Configured {
		return []byte("[]"), nil
	}
	info, err := os.Stat(cfg.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("cookie_path file not found: %s", cfg.Path)
		}
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("cookie_path must be a file, got directory: %s", cfg.Path)
	}
	data, err := os.ReadFile(cfg.Path)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return []byte("[]"), nil
	}
	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return nil, fmt.Errorf("decode cookie_path file %s: %w", cfg.Path, err)
	}
	return []byte(trimmed), nil
}

func browserResolveCookieFile(flags map[string]string) (browserCookieFileConfig, error) {
	raw := strings.TrimSpace(flags["cookie_path"])
	fromMeta := false
	if raw == "" {
		value, ok, err := browserLookupCookiePathFromPluginMeta(flags)
		if err != nil {
			return browserCookieFileConfig{}, err
		}
		if ok {
			raw = value
			fromMeta = true
		}
	}
	if raw == "" {
		return browserCookieFileConfig{}, nil
	}
	root, _, err := browserRuntimeRoot(flags)
	if err != nil {
		return browserCookieFileConfig{}, err
	}
	if !filepath.IsAbs(raw) {
		raw = filepath.Join(root, raw)
	}
	path, err := filepath.Abs(raw)
	if err != nil {
		return browserCookieFileConfig{}, err
	}
	return browserCookieFileConfig{
		Path:       path,
		Configured: true,
		FromMeta:   fromMeta,
	}, nil
}

func browserLookupCookiePathFromPluginMeta(flags map[string]string) (string, bool, error) {
	response, ok, err := browserLookupPluginMeta(flags, browserKey, true)
	if err != nil || !ok {
		return "", false, nil
	}
	value, _ := response.Meta["cookie_path"].(string)
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false, nil
	}
	return value, true, nil
}

func browserLogCookiePreflightEvent(flags map[string]string, command string, err error) {
	payload := map[string]any{
		"event":     "browser_cookie_preflight",
		"command":   strings.TrimSpace(command),
		"timestamp": browserNowFn().Format(time.RFC3339Nano),
	}
	if cfg, resolveErr := browserResolveCookieFile(flags); resolveErr == nil && cfg.Configured {
		payload["cookiePath"] = cfg.Path
		payload["fromMeta"] = cfg.FromMeta
	}
	if err != nil {
		payload["stage"] = "error"
		payload["error"] = err.Error()
	} else {
		payload["stage"] = "ok"
	}
	browserAppendLogJSON(payload)
}
