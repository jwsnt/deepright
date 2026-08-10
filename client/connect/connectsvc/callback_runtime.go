package connectsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DefaultAutoReply = "<开始执行>可通过新消息更新任务内容"

type PluginCallbackTarget struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Callback string `json:"callback"`
}

type PluginActionResponse struct {
	Path    string   `json:"path"`
	Command []string `json:"command"`
	Output  any      `json:"output,omitempty"`
}

type PluginStatusResponse struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	PID     int    `json:"pid"`
	PIDFile string `json:"pidFile"`
	Started bool   `json:"started"`
}

func ResolveAutoReply(reply string) string {
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return DefaultAutoReply
	}
	return reply
}

func BuildPluginCallbackMap(items []PluginCallbackTarget) map[string]string {
	if len(items) == 0 {
		return nil
	}
	callbacks := make(map[string]string, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.Key)
		callback := strings.TrimSpace(item.Callback)
		if key == "" || callback == "" {
			continue
		}
		callbacks[key] = callback
	}
	if len(callbacks) == 0 {
		return nil
	}
	return callbacks
}

func ResolvePluginCallback(callbacks map[string]string, pluginKey string) (string, error) {
	key := strings.TrimSpace(pluginKey)
	if key == "" {
		return "", fmt.Errorf("plugin key is required")
	}
	callback := strings.TrimSpace(callbacks[key])
	if callback == "" {
		return "", fmt.Errorf("plugin callback is empty: %s", key)
	}
	return callback, nil
}

func BuildPluginCallbackMapFromMeta(items []MetaConfig, normalizeKey func(string) string) map[string]string {
	if len(items) == 0 {
		return nil
	}
	targets := make([]PluginCallbackTarget, 0, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.Key)
		if normalizeKey != nil {
			key = strings.TrimSpace(normalizeKey(key))
		}
		targets = append(targets, PluginCallbackTarget{
			Key:      key,
			Name:     strings.TrimSpace(item.Name),
			Callback: strings.TrimSpace(item.Callback),
		})
	}
	return BuildPluginCallbackMap(targets)
}

func NormalizePluginCallbackKey(pluginKey string, normalizeKey func(string) string) string {
	pluginKey = strings.TrimSpace(pluginKey)
	if normalizeKey != nil {
		pluginKey = strings.TrimSpace(normalizeKey(pluginKey))
	}
	return pluginKey
}

func EnsureConnectBinary(flags map[string]string, executable string) {
	if flags == nil {
		return
	}
	if strings.TrimSpace(flags["connect-bin"]) != "" {
		return
	}
	if strings.TrimSpace(executable) != "" {
		flags["connect-bin"] = strings.TrimSpace(executable)
	}
}

func NormalizePluginActionOutput(raw []byte) any {
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return nil
	}
	var value any
	if err := json.Unmarshal([]byte(text), &value); err == nil {
		return value
	}
	return text
}

func BuildPluginActionResponse(result *PluginActionResult) PluginActionResponse {
	response := PluginActionResponse{}
	if result == nil {
		return response
	}
	response.Path = result.Path
	response.Command = append([]string{}, result.Command...)
	response.Output = NormalizePluginActionOutput(result.Output)
	return response
}

func BuildPluginStatusResponse(result *PluginStatus) PluginStatusResponse {
	response := PluginStatusResponse{}
	if result == nil {
		return response
	}
	response.Key = result.Key
	response.Name = result.Name
	response.Path = result.Path
	response.PID = result.PID
	response.PIDFile = result.PIDFile
	response.Started = result.Started
	return response
}

func listPluginCallbackCommands(callbackPath string) ([]string, error) {
	callbackPath = strings.TrimSpace(callbackPath)
	if callbackPath == "" {
		return nil, fmt.Errorf("plugin callback path is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), pluginCommandTimeout)
	defer cancel()

	output, err := runPluginCommand(ctx, callbackPath, "command")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("callback executable not found: %s", callbackPath)
		}
		trimmed := strings.TrimSpace(err.Error())
		if trimmed != "" {
			return nil, fmt.Errorf("callback command check failed: %s", trimmed)
		}
		return nil, fmt.Errorf("callback command check failed: %w", err)
	}

	var commands []string
	if err := json.Unmarshal(output, &commands); err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return nil, fmt.Errorf("callback command returned empty output")
		}
		return nil, fmt.Errorf("callback command returned invalid json: %s", trimmed)
	}
	return commands, nil
}

func pluginCallbackCommandSupported(commands []string, action string) bool {
	action = strings.TrimSpace(strings.ToLower(action))
	for _, command := range commands {
		if strings.EqualFold(strings.TrimSpace(command), action) {
			return true
		}
	}
	return false
}

func detectPluginCallbackActionSupport(callbackPath, action string) error {
	callbackPath = strings.TrimSpace(callbackPath)
	if callbackPath == "" {
		return fmt.Errorf("plugin callback path is required")
	}
	action = strings.TrimSpace(strings.ToLower(action))
	if action == "" {
		return fmt.Errorf("plugin callback action is required")
	}
	switch action {
	case "init", "send":
	default:
		return fmt.Errorf("unsupported plugin callback action: %s", action)
	}

	commands, err := listPluginCallbackCommands(callbackPath)
	if err == nil {
		if pluginCallbackCommandSupported(commands, action) {
			return nil
		}
		return fmt.Errorf("callback command does not expose %s", action)
	}
	return err
}

func RunPluginCallbackAction(callbackPath, action string, flags map[string]string) error {
	callbackPath = strings.TrimSpace(callbackPath)
	if callbackPath == "" {
		return fmt.Errorf("plugin callback path is required")
	}
	action = strings.TrimSpace(strings.ToLower(action))
	if action == "" {
		return fmt.Errorf("plugin callback action is required")
	}
	if absPath, err := filepath.Abs(callbackPath); err == nil {
		callbackPath = absPath
	}
	if err := detectPluginCallbackActionSupport(callbackPath, action); err != nil {
		return err
	}
	args := []string{action}
	args = append(args, buildPluginActionArgs(flags)...)
	_, err := runPluginActionCommand(callbackPath, args...)
	return err
}
