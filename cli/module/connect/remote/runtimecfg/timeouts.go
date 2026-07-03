package runtimecfg

import (
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

	"connect/connectsvc"
)

const (
	DefaultCommandTimeout = 30000 * time.Millisecond
	DefaultSCPTimeout     = 30000 * time.Millisecond
	metaLookupTimeout     = 5 * time.Second
	pluginKeyRemote       = "remote"
)

var (
	ExecCommandContext = exec.CommandContext
	OSExecutable       = os.Executable
)

type Timeouts struct {
	Exec time.Duration
	SCP  time.Duration
}

type pluginMetaResponse struct {
	Key  string         `json:"key"`
	Meta pluginMetaBody `json:"meta"`
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
	meta, err := loadRemotePluginMeta(flags)
	if err != nil || meta == nil {
		return timeouts
	}
	if value, ok := ParseTimeoutMS(meta.stringValue("exec_timeout")); ok {
		timeouts.Exec = value
	}
	if value, ok := ParseTimeoutMS(meta.stringValue("scp_timeout")); ok {
		timeouts.SCP = value
	}
	return timeouts
}

func loadRemotePluginMeta(flags map[string]string) (pluginMetaBody, error) {
	connectBin, prefix := resolveMetaLookupCommand(flags)
	if strings.TrimSpace(connectBin) == "" {
		return nil, errors.New("connect-bin is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), metaLookupTimeout)
	defer cancel()

	args := append([]string{}, prefix...)
	args = append(args, "meta-get", "--key", pluginKeyRemote)
	cmd := ExecCommandContext(ctx, connectBin, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var response pluginMetaResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, err
	}
	return response.Meta, nil
}

func resolveMetaLookupCommand(flags map[string]string) (string, []string) {
	explicit := strings.TrimSpace(connectsvc.FirstValue(flags, "connect-bin"))
	if explicit != "" {
		return normalizeBinaryPath(explicit), connectPrefixForBinary(explicit)
	}
	return "", nil
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

func connectPrefixForBinary(binary string) []string {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(binary)))
	if base == "integration" || base == "proxy" || strings.HasPrefix(base, "integration.") || strings.HasPrefix(base, "proxy.") {
		return []string{"connect"}
	}
	return nil
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
