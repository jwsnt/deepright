package runtimecfg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"connect/connectsvc"
)

const (
	DefaultCommandTimeout = 300 * time.Second
	DefaultSCPTimeout     = 300 * time.Second
)

var (
	OSExecutable = os.Executable
)

type Timeouts struct {
	Exec time.Duration
	SCP  time.Duration
}

type pluginMetaBody map[string]any

func (m pluginMetaBody) stringValue(key string) string {
	if m == nil {
		return ""
	}
	value, ok := m[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strings.TrimSpace(strconv.FormatFloat(typed, 'f', -1, 64))
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case json.Number:
		return strings.TrimSpace(typed.String())
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func ResolvePluginTimeouts(flags map[string]string) Timeouts {
	timeouts := Timeouts{
		Exec: DefaultCommandTimeout,
		SCP:  DefaultSCPTimeout,
	}
	if value, ok := ParseTimeoutMS(connectsvc.FirstValue(flags, "timeout")); ok {
		timeouts.Exec = value
		timeouts.SCP = value
		return timeouts
	}
	config, err := loadRemoteTimeoutConfig(flags)
	if err != nil || config == nil {
		return timeouts
	}
	if value, ok := ParseTimeoutSeconds(config.stringValue("exec_timeout")); ok {
		timeouts.Exec = value
	}
	if value, ok := ParseTimeoutSeconds(config.stringValue("scp_timeout")); ok {
		timeouts.SCP = value
	}
	return timeouts
}

func loadRemoteTimeoutConfig(flags map[string]string) (pluginMetaBody, error) {
	configPath, ok := resolveRemoteConfigPath(flags)
	if !ok {
		return nil, fmt.Errorf("resolve config/config.json")
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	var raw map[string]any
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}
	remote, ok := raw["remote"].(map[string]any)
	if !ok || remote == nil {
		return nil, fmt.Errorf("remote config is missing")
	}
	return pluginMetaBody(remote), nil
}

func resolveRemoteConfigPath(flags map[string]string) (string, bool) {
	if connectBin := strings.TrimSpace(connectsvc.FirstValue(flags, "connect-bin")); connectBin != "" {
		if path, ok := connectsvc.ResolveRuntimeConfigPathFromConnectBin(normalizeBinaryPath(connectBin)); ok {
			return path, true
		}
	}
	if executable, err := OSExecutable(); err == nil && strings.TrimSpace(executable) != "" {
		return connectsvc.ResolveRuntimeConfigPathNearBinary(executable)
	}
	return "", false
}

func normalizeBinaryPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if filepath.IsAbs(value) {
		return value
	}
	if strings.HasPrefix(value, ".") || strings.Contains(value, "/") || strings.Contains(value, string(filepath.Separator)) {
		if abs, err := filepath.Abs(value); err == nil {
			return abs
		}
	}
	return value
}

func ParseTimeoutMS(raw string) (time.Duration, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	milliseconds, err := strconv.Atoi(raw)
	if err != nil || milliseconds <= 0 {
		return 0, false
	}
	return time.Duration(milliseconds) * time.Millisecond, true
}

func ParseTimeoutSeconds(raw string) (time.Duration, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	seconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || seconds <= 0 || seconds > int64((1<<63-1)/int64(time.Second)) {
		return 0, false
	}
	return time.Duration(seconds) * time.Second, true
}
