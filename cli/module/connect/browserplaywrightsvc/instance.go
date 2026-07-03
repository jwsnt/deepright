package browserplaywrightsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultInstanceBinaryName  = "browser_instance"
	defaultInstanceCDPHostname = "127.0.0.1"
	instanceCreateTimeout      = 15 * time.Second
)

type instanceCreateResult struct {
	AgentID string `json:"agentId"`
	ChatID  string `json:"chatId"`
	Port    int    `json:"port"`
	PID     int    `json:"pid"`
	CDP     string `json:"cdp"`
}

type managedInstanceIdentity struct {
	AgentID string
	ChatID  string
	Session string
}

var (
	browserPlaywrightExecutablePathFn = os.Executable
	managedInstanceProvider           ManagedInstanceProvider
	browserInstanceCreateFn           = runBrowserInstanceCreate
	browserInstanceGetFn              = runBrowserInstanceGet
)

var errBrowserInstanceNotFound = errors.New("browser_instance not found")

func prepareCreateFlags(flags map[string]string) (map[string]string, error) {
	next := normalizeManagedIdentityFlags(flags)
	timeout, err := browserTimeoutFromFlags(next)
	if err != nil {
		return nil, err
	}
	identity, err := requiredManagedInstanceIdentity(next)
	if err != nil {
		return nil, err
	}
	item, err := ensureBrowserInstance(next, identity, timeout)
	if err != nil {
		return nil, err
	}
	next["cdp"] = firstNonEmptyString(strings.TrimSpace(item.CDP), instanceCDPEndpoint(item.Port))
	next["session"] = identity.Session
	next["agentId"] = identity.AgentID
	next["chatId"] = identity.ChatID
	return next, nil
}

func prepareManagedFlags(flags map[string]string) (map[string]string, bool, error) {
	next := normalizeManagedIdentityFlags(flags)
	if strings.TrimSpace(next["cdp"]) != "" {
		return next, false, nil
	}
	timeout, err := browserTimeoutFromFlags(next)
	if err != nil {
		return nil, false, err
	}
	identity, ok := resolveManagedInstanceIdentity(next)
	if !ok {
		return next, false, nil
	}
	item, err := ensureBrowserInstance(next, identity, timeout)
	if err != nil {
		return nil, false, err
	}
	next["cdp"] = firstNonEmptyString(strings.TrimSpace(item.CDP), instanceCDPEndpoint(item.Port))
	next["session"] = identity.Session
	next["agentId"] = identity.AgentID
	next["chatId"] = identity.ChatID
	return next, true, nil
}

func ensureBrowserInstance(flags map[string]string, identity managedInstanceIdentity, timeout time.Duration) (instanceCreateResult, error) {
	if managedInstanceProvider != nil {
		item, err := managedInstanceProvider.GetManagedInstance(normalizeManagedInstanceProviderFlags(flags, identity), identity.AgentID, identity.ChatID)
		if err == nil {
			return normalizeInstanceIdentity(managedInstanceRecordToCreateResult(item), identity), nil
		}
		if !isBrowserInstanceNotFound(err) {
			return instanceCreateResult{}, err
		}
		item, err = managedInstanceProvider.CreateManagedInstance(normalizeManagedInstanceProviderFlags(flags, identity), identity.AgentID, identity.ChatID)
		if err != nil {
			return instanceCreateResult{}, err
		}
		return normalizeInstanceIdentity(managedInstanceRecordToCreateResult(item), identity), nil
	}

	path, err := resolveBrowserInstanceBinary(flags)
	if err != nil {
		return instanceCreateResult{}, err
	}
	item, err := browserInstanceGetFn(path, identity.AgentID, identity.ChatID, flags, timeout)
	if err == nil {
		return normalizeInstanceIdentity(item, identity), nil
	}
	if !isBrowserInstanceNotFound(err) {
		return instanceCreateResult{}, err
	}
	item, err = browserInstanceCreateFn(path, identity.AgentID, identity.ChatID, flags, timeout)
	if err != nil {
		return instanceCreateResult{}, err
	}
	return normalizeInstanceIdentity(item, identity), nil
}

func resolveBrowserInstanceBinary(flags map[string]string) (string, error) {
	if custom := strings.TrimSpace(flags["instance-bin"]); custom != "" {
		return ensureExecutablePath(custom, "browser_instance")
	}
	exe, err := browserPlaywrightExecutablePathFn()
	if err != nil {
		return "", err
	}
	candidate := filepath.Join(filepath.Dir(exe), "..", "instance", defaultInstanceBinaryName)
	return ensureExecutablePath(candidate, "browser_instance")
}

func ensureExecutablePath(path, label string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("%s path is empty", label)
	}
	absPath, err := filepath.Abs(path)
	if err == nil {
		path = absPath
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%s not found: %s", label, path)
		}
		return "", err
	}
	if info.IsDir() || info.Mode()&0o111 == 0 {
		return "", fmt.Errorf("%s is not executable: %s", label, path)
	}
	return path, nil
}

func runBrowserInstanceCreate(path, agentID, chatID string, flags map[string]string, timeout time.Duration) (instanceCreateResult, error) {
	return runBrowserInstanceCommand(path, "create", agentID, chatID, flags, timeout)
}

func runBrowserInstanceGet(path, agentID, chatID string, flags map[string]string, timeout time.Duration) (instanceCreateResult, error) {
	return runBrowserInstanceCommand(path, "get", agentID, chatID, flags, timeout)
}

func runBrowserInstanceCommand(path, command, agentID, chatID string, flags map[string]string, timeout time.Duration) (instanceCreateResult, error) {
	timeout = normalizeBrowserTimeout(timeout)
	if timeout > instanceCreateTimeout {
		timeout = instanceCreateTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := []string{command, "--agentId", agentID, "--chatId", chatID}
	if value := strings.TrimSpace(flags["instance-state"]); value != "" {
		args = append(args, "--state", value)
	}
	if value := strings.TrimSpace(flags["instance-obscura"]); value != "" {
		args = append(args, "--obscura", value)
	}
	if value, ok := flags["instance-monitor"]; ok {
		value = strings.TrimSpace(value)
		if value == "" {
			value = "true"
		}
		args = append(args, "--monitor="+value)
	}
	if value := strings.TrimSpace(flags["instance-monitor-ms"]); value != "" {
		args = append(args, "--monitor-ms", value)
	}

	cmd := exec.CommandContext(ctx, path, args...)
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return instanceCreateResult{}, fmt.Errorf("browser_instance %s timed out after %s", command, timeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			message := strings.TrimSpace(string(exitErr.Stderr))
			if message != "" {
				if command == "get" && strings.Contains(strings.ToLower(message), "instance not found") {
					return instanceCreateResult{}, fmt.Errorf("%w: %s", errBrowserInstanceNotFound, message)
				}
				return instanceCreateResult{}, fmt.Errorf("browser_instance %s failed: %s", command, message)
			}
		}
		return instanceCreateResult{}, fmt.Errorf("browser_instance %s failed: %w", command, err)
	}

	var item instanceCreateResult
	if err := json.Unmarshal(out, &item); err != nil {
		return instanceCreateResult{}, fmt.Errorf("decode browser_instance %s output: %w", command, err)
	}
	if item.Port <= 0 {
		return instanceCreateResult{}, fmt.Errorf("browser_instance %s returned invalid port %d", command, item.Port)
	}
	return item, nil
}

func resolveManagedInstanceIdentity(flags map[string]string) (managedInstanceIdentity, bool) {
	flags = normalizeManagedIdentityFlags(flags)
	if agentID, chatID, ok := splitManagedSession(flags["session"]); ok {
		return managedInstanceIdentity{
			AgentID: agentID,
			ChatID:  chatID,
			Session: instanceSessionName(agentID, chatID),
		}, true
	}
	agentID := firstIdentityValue(flags["agentId"], flags["agent"])
	chatID := firstIdentityValue(flags["chatId"], flags["chat"])
	if agentID == "" || chatID == "" {
		return managedInstanceIdentity{}, false
	}
	return managedInstanceIdentity{
		AgentID: agentID,
		ChatID:  chatID,
		Session: instanceSessionName(agentID, chatID),
	}, true
}

func requiredManagedInstanceIdentity(flags map[string]string) (managedInstanceIdentity, error) {
	identity, ok := resolveManagedInstanceIdentity(flags)
	if !ok {
		return managedInstanceIdentity{}, fmt.Errorf("create requires --agentId and --chatId")
	}
	return identity, nil
}

func normalizeInstanceIdentity(item instanceCreateResult, identity managedInstanceIdentity) instanceCreateResult {
	if strings.TrimSpace(item.AgentID) == "" {
		item.AgentID = identity.AgentID
	}
	if strings.TrimSpace(item.ChatID) == "" {
		item.ChatID = identity.ChatID
	}
	if strings.TrimSpace(item.CDP) == "" && item.Port > 0 {
		item.CDP = instanceCDPEndpoint(item.Port)
	}
	return item
}

func SetManagedInstanceProvider(provider ManagedInstanceProvider) {
	managedInstanceProvider = provider
}

func normalizeManagedInstanceProviderFlags(flags map[string]string, identity managedInstanceIdentity) map[string]string {
	next := normalizeManagedIdentityFlags(flags)
	if next == nil {
		next = map[string]string{}
	}
	next["agentId"] = identity.AgentID
	next["chatId"] = identity.ChatID
	next["session"] = identity.Session
	if value, ok := next["instance-state"]; ok && strings.TrimSpace(value) != "" {
		next["state"] = value
	}
	if value, ok := next["instance-obscura"]; ok && strings.TrimSpace(value) != "" {
		next["obscura"] = value
	}
	if value, ok := next["instance-monitor"]; ok {
		next["monitor"] = value
	}
	if value, ok := next["instance-monitor-ms"]; ok && strings.TrimSpace(value) != "" {
		next["monitor-ms"] = value
	}
	if value, ok := next["instance-browser_expired"]; ok && strings.TrimSpace(value) != "" {
		next["browser_expired"] = value
	}
	return next
}

func managedInstanceRecordToCreateResult(item ManagedInstanceRecord) instanceCreateResult {
	return instanceCreateResult{
		AgentID: item.AgentID,
		ChatID:  item.ChatID,
		Port:    item.Port,
		PID:     item.PID,
		CDP:     item.CDP,
	}
}

func isBrowserInstanceNotFound(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, errBrowserInstanceNotFound) || strings.Contains(strings.ToLower(err.Error()), "instance not found")
}

func instanceCDPEndpoint(port int) string {
	return fmt.Sprintf("ws://%s:%d/devtools/browser", defaultInstanceCDPHostname, port)
}

func instanceSessionName(agentID, chatID string) string {
	return fmt.Sprintf("%s@%s", normalizeManagedIdentityPart(agentID), normalizeManagedIdentityPart(chatID))
}

func cloneFlags(flags map[string]string) map[string]string {
	if len(flags) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(flags))
	for key, value := range flags {
		out[key] = value
	}
	return out
}

func firstIdentityValue(values ...string) string {
	for _, value := range values {
		value = normalizeManagedIdentityPart(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func splitManagedSession(session string) (string, string, bool) {
	session = normalizeManagedSessionValue(session)
	if session == "" {
		return "", "", false
	}
	parts := strings.SplitN(session, "@", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	agentID := strings.TrimSpace(parts[0])
	chatID := strings.TrimSpace(parts[1])
	if agentID == "" || chatID == "" {
		return "", "", false
	}
	return agentID, chatID, true
}

func normalizeManagedIdentityPart(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeManagedSessionValue(session string) string {
	session = strings.TrimSpace(session)
	if session == "" {
		return ""
	}
	parts := strings.SplitN(session, "@", 2)
	if len(parts) != 2 {
		return strings.ToLower(session)
	}
	agentID := normalizeManagedIdentityPart(parts[0])
	chatID := normalizeManagedIdentityPart(parts[1])
	if agentID == "" || chatID == "" {
		return strings.ToLower(session)
	}
	return instanceSessionName(agentID, chatID)
}

func normalizeManagedIdentityFlags(flags map[string]string) map[string]string {
	next := cloneFlags(flags)
	for _, key := range []string{"agentId", "agent", "chatId", "chat"} {
		if value, ok := next[key]; ok {
			next[key] = normalizeManagedIdentityPart(value)
		}
	}
	if value, ok := next["session"]; ok {
		next["session"] = normalizeManagedSessionValue(value)
	}
	return next
}
