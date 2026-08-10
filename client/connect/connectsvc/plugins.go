package connectsvc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultPluginCacheMS = 10000
	pluginCommandTimeout = 5 * time.Second
	pluginActionTimeout  = 180 * time.Second
	pluginCacheFileName  = ".connect-plugin-cache.json"
)

type PluginInfo struct {
	Key     string            `json:"key"`
	Name    string            `json:"name"`
	Param   PluginParamFields `json:"param"`
	Scope   []string          `json:"scope,omitempty"`
	Command []string          `json:"command,omitempty"`
}

type PluginBinary struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Path string `json:"path"`
}

type PluginMetaInfo struct {
	Key           string            `json:"key"`
	Name          string            `json:"name"`
	Param         PluginParamFields `json:"param"`
	Scope         []string          `json:"scope,omitempty"`
	Meta          map[string]any    `json:"meta"`
	Stream        bool              `json:"stream"`
	Callback      string            `json:"callback"`
	AgentID       string            `json:"agentId"`
	ChatID        string            `json:"chatId"`
	Model         string            `json:"model"`
	Thinking      bool              `json:"thinking"`
	Verify        bool              `json:"verify"`
	RouterDisable bool              `json:"router_disable"`
	CreatedAt     string            `json:"createdAt"`
	UpdatedAt     string            `json:"updatedAt"`
}

type pluginInfoJSON struct {
	Key     string            `json:"key"`
	Name    string            `json:"name"`
	Param   PluginParamFields `json:"param"`
	Scope   *[]string         `json:"scope,omitempty"`
	Command []string          `json:"command,omitempty"`
}

type pluginMetaInfoJSON struct {
	Key           string            `json:"key"`
	Name          string            `json:"name"`
	Param         PluginParamFields `json:"param"`
	Scope         *[]string         `json:"scope,omitempty"`
	Meta          map[string]any    `json:"meta"`
	Stream        bool              `json:"stream"`
	Callback      string            `json:"callback"`
	AgentID       string            `json:"agentId"`
	ChatID        string            `json:"chatId"`
	Model         string            `json:"model"`
	Thinking      bool              `json:"thinking"`
	Verify        bool              `json:"verify"`
	RouterDisable bool              `json:"router_disable"`
	CreatedAt     string            `json:"createdAt"`
	UpdatedAt     string            `json:"updatedAt"`
}

type PluginParamField struct {
	Key         string
	Placeholder string
}

type PluginParamFields []PluginParamField

func (p PluginParamFields) MarshalJSON() ([]byte, error) {
	items := make([]map[string]string, 0, len(p))
	for _, field := range p {
		key := strings.TrimSpace(field.Key)
		if key == "" {
			continue
		}
		items = append(items, map[string]string{
			key: field.Placeholder,
		})
	}
	return json.Marshal(items)
}

func (p *PluginParamFields) UnmarshalJSON(data []byte) error {
	fields, err := parsePluginParamFields(data)
	if err != nil {
		return err
	}
	*p = fields
	return nil
}

func (p PluginInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(pluginInfoJSON{
		Key:     p.Key,
		Name:    p.Name,
		Param:   p.Param,
		Scope:   scopeJSONField(p.Scope),
		Command: p.Command,
	})
}

func (p PluginMetaInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(pluginMetaInfoJSON{
		Key:           p.Key,
		Name:          p.Name,
		Param:         p.Param,
		Scope:         scopeJSONField(p.Scope),
		Meta:          p.Meta,
		Stream:        p.Stream,
		Callback:      p.Callback,
		AgentID:       p.AgentID,
		ChatID:        p.ChatID,
		Model:         p.Model,
		Thinking:      p.Thinking,
		Verify:        p.Verify,
		RouterDisable: p.RouterDisable,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	})
}

func scopeJSONField(scope []string) *[]string {
	if scope == nil {
		return nil
	}
	next := make([]string, len(scope))
	copy(next, scope)
	return &next
}

type PluginActionResult struct {
	Path    string   `json:"path"`
	Command []string `json:"command"`
	Output  []byte   `json:"-"`
}

type PluginStatus struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	PIDFile string `json:"pidFile"`
	Started bool   `json:"started"`
	PID     int    `json:"pid,omitempty"`
}

type pluginCache struct {
	UpdatedAtUnixMilli int64        `json:"updatedAtUnixMilli"`
	Items              []PluginInfo `json:"items"`
}

func ParsePluginExecRequest(r *http.Request) (string, string, map[string]string, error) {
	query := r.URL.Query()
	key := strings.TrimSpace(query.Get("key"))
	if key == "" {
		return "", "", nil, fmt.Errorf("key is required")
	}
	command := strings.TrimSpace(query.Get("command"))
	if command == "" {
		return "", "", nil, fmt.Errorf("command is required")
	}
	flags := make(map[string]string)
	for key, values := range query {
		if key == "key" || key == "command" || len(values) == 0 {
			continue
		}
		flags[key] = strings.TrimSpace(values[len(values)-1])
	}
	return key, command, flags, nil
}

func SplitPluginExecCommand(command string) ([]string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil, fmt.Errorf("command is required")
	}

	var (
		args      []string
		current   strings.Builder
		quote     rune
		escaping  bool
		hasQuoted bool
	)

	flush := func(force bool) {
		if current.Len() == 0 && !force {
			return
		}
		args = append(args, current.String())
		current.Reset()
		hasQuoted = false
	}

	for _, r := range command {
		if escaping {
			// 转义只影响紧随其后的一个字符，保留原字符内容。
			current.WriteRune(r)
			escaping = false
			hasQuoted = true
			continue
		}
		switch {
		case r == '\\':
			// 遇到反斜杠时延迟到下一轮处理，避免提前拆词。
			escaping = true
		case quote != 0:
			if r == quote {
				// 只有遇到同类型引号才结束当前引号段。
				quote = 0
				hasQuoted = true
				continue
			}
			current.WriteRune(r)
		case r == '\'' || r == '"':
			// 未进入引号段时，单引号和双引号都作为新的分支入口。
			quote = r
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			// 只有在非引号态下，空白才作为参数分隔符生效。
			flush(hasQuoted)
		default:
			current.WriteRune(r)
		}
	}

	if escaping {
		return nil, fmt.Errorf("command has dangling escape")
	}
	if quote != 0 {
		return nil, fmt.Errorf("command has unclosed quote")
	}
	flush(hasQuoted)
	if len(args) == 0 {
		return nil, fmt.Errorf("command is required")
	}
	return args, nil
}

func RunPluginExecCommand(target, command string, flags map[string]string, timeout time.Duration) (*PluginActionResult, error) {
	binary, err := ResolvePluginBinary(target)
	if err != nil {
		return nil, err
	}

	commandArgs, err := SplitPluginExecCommand(command)
	if err != nil {
		return nil, err
	}
	fullArgs := append(commandArgs, buildPluginActionArgs(flags)...)
	output, err := RunPluginCLIWithTimeout(binary.Path, timeout, fullArgs...)
	if err != nil {
		return nil, fmt.Errorf("run plugin %s %s failed: %w", target, command, err)
	}
	return &PluginActionResult{
		Path:    binary.Path,
		Command: fullArgs,
		Output:  output,
	}, nil
}

func WritePluginActionError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  1,
		"content": err.Error(),
	})
}

func PluginActionStatusCode(err error) int {
	lower := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(lower, "key is required"),
		strings.Contains(lower, "command is required"),
		strings.Contains(lower, "plugin not found"),
		strings.Contains(lower, "plugin is not executable"):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func ListPlugins(flags map[string]string) ([]PluginInfo, error) {
	cacheMS, err := IntValue(flags, "connect-cache", defaultPluginCacheMS)
	if err != nil {
		return nil, err
	}
	pluginDir, err := pluginDirResolver()
	if err != nil {
		return nil, err
	}
	ttl := time.Duration(cacheMS) * time.Millisecond
	if ttl > 0 {
		if items, ok := readPluginCache(pluginDir, ttl); ok {
			return items, nil
		}
	}
	items, err := scanPlugins(pluginDir)
	if err != nil {
		return nil, err
	}
	if ttl > 0 {
		_ = writePluginCache(pluginDir, items)
	}
	return items, nil
}

func ListPluginMeta(flags map[string]string) ([]PluginMetaInfo, error) {
	svc, err := buildService(flags)
	if err != nil {
		return nil, err
	}
	defer svc.Close()
	return ListPluginMetaWithService(svc, flags)
}

func ListPluginMetaWithService(svc *Service, flags map[string]string) ([]PluginMetaInfo, error) {
	if svc == nil {
		return nil, fmt.Errorf("connect service is required")
	}

	items, err := ListPlugins(flags)
	if err != nil {
		return nil, err
	}
	return listPluginMetaItems(svc, items)
}

func listPluginMetaItems(svc *Service, items []PluginInfo) ([]PluginMetaInfo, error) {
	configs, err := svc.ListMetaConfig(false)
	if err != nil {
		return nil, err
	}

	byKey := make(map[string]MetaConfig, len(configs))
	for _, item := range configs {
		if item.Meta == nil {
			item.Meta = map[string]any{}
		}
		if key := strings.TrimSpace(item.Key); key != "" {
			byKey[key] = item
		}
	}

	out := make([]PluginMetaInfo, 0, len(items))
	for _, item := range items {
		config, ok := byKey[strings.TrimSpace(item.Key)]
		if !ok {
			config = MetaConfig{Meta: map[string]any{}, RouterDisable: true}
		}
		if config.Meta == nil {
			config.Meta = map[string]any{}
		}
		out = append(out, PluginMetaInfo{
			Key:           item.Key,
			Name:          item.Name,
			Param:         item.Param,
			Scope:         item.Scope,
			Meta:          config.Meta,
			Stream:        config.Stream,
			Callback:      config.Callback,
			AgentID:       config.AgentID,
			ChatID:        config.ChatID,
			Model:         config.Model,
			Thinking:      config.Thinking,
			Verify:        config.Verify,
			RouterDisable: config.RouterDisable,
			CreatedAt:     config.CreatedAt,
			UpdatedAt:     config.UpdatedAt,
		})
	}
	return out, nil
}

func ResolvePluginBinaryByKey(key string) (*PluginBinary, error) {
	targetKey := strings.TrimSpace(key)
	if targetKey == "" {
		return nil, fmt.Errorf("key is required")
	}

	pluginDir, err := pluginDirResolver()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("plugin not found: %s", targetKey)
		}
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name())
		if shouldSkipPluginEntry(name) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		path := filepath.Join(pluginDir, name)
		if !shouldInspectPluginEntry(path, info) {
			continue
		}
		item, err := inspectPlugin(path)
		if err != nil {
			return nil, fmt.Errorf("inspect plugin %s: %w", name, err)
		}
		if !strings.EqualFold(strings.TrimSpace(item.Key), targetKey) {
			continue
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path
		}
		return &PluginBinary{
			Key:  item.Key,
			Name: item.Name,
			Path: absPath,
		}, nil
	}
	return nil, fmt.Errorf("plugin not found: %s", targetKey)
}

func ResolvePluginBinary(target string) (*PluginBinary, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("key is required")
	}
	if isPluginPathLike(target) {
		return resolvePluginBinaryFromPath(target)
	}

	pluginDir, err := pluginDirResolver()
	if err != nil {
		return nil, err
	}
	if binary, err := resolvePluginBinaryFromPath(filepath.Join(pluginDir, target)); err == nil {
		return binary, nil
	}
	return ResolvePluginBinaryByKey(target)
}

func RunPluginAction(target, action string, extraFlags map[string]string) (*PluginActionResult, error) {
	target = strings.TrimSpace(target)
	action = strings.ToLower(strings.TrimSpace(action))
	if target == "" {
		return nil, fmt.Errorf("key is required")
	}
	switch action {
	case "start", "stop":
	default:
		return nil, fmt.Errorf("unsupported plugin action: %s", action)
	}

	binary, err := resolvePluginBinaryForAction(target)
	if err != nil {
		return nil, err
	}

	args := buildPluginActionArgs(extraFlags)
	subcommandArgs := append([]string{action}, args...)
	output, err := runPluginActionCommand(binary.Path, subcommandArgs...)
	if err != nil {
		return nil, fmt.Errorf("run plugin %s %s failed: %w", target, action, err)
	}
	return &PluginActionResult{
		Path:    binary.Path,
		Command: subcommandArgs,
		Output:  output,
	}, nil
}

func RunPluginCLI(path string, args ...string) ([]byte, error) {
	return RunPluginCLIWithTimeout(path, pluginActionTimeout, args...)
}

func RunPluginCLIWithTimeout(path string, timeout time.Duration, args ...string) ([]byte, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("plugin path is required")
	}
	absPath, err := filepath.Abs(path)
	if err == nil {
		path = absPath
	}
	if timeout <= 0 {
		timeout = pluginActionTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := pluginCommand(ctx, path, args...)
	out, runErr := cmd.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		timeoutErr := fmt.Errorf("plugin command timed out after %s", timeout)
		if len(out) == 0 {
			return nil, timeoutErr
		}
		return out, fmt.Errorf("%w: %s", timeoutErr, strings.TrimSpace(string(out)))
	}
	if runErr != nil {
		if len(out) == 0 {
			return nil, runErr
		}
		return out, fmt.Errorf("%w: %s", runErr, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func resolvePluginBinaryForAction(target string) (*PluginBinary, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("key is required")
	}
	if binary, err := resolveDirectPluginBinary(target); err == nil {
		return binary, nil
	}
	return ResolvePluginBinary(target)
}

func resolveDirectPluginBinary(target string) (*PluginBinary, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("key is required")
	}
	if isPluginPathLike(target) {
		info, err := os.Stat(target)
		if err != nil {
			return nil, err
		}
		if !isExecutablePlugin(info) {
			return nil, fmt.Errorf("plugin is not executable: %s", strings.TrimSpace(target))
		}
		absPath, err := filepath.Abs(target)
		if err != nil {
			absPath = target
		}
		return &PluginBinary{
			Key:  filepath.Base(absPath),
			Name: filepath.Base(absPath),
			Path: absPath,
		}, nil
	}

	pluginDir, err := pluginDirResolver()
	if err != nil {
		return nil, err
	}
	if binary, err := ResolvePluginBinaryByKey(target); err == nil {
		return binary, nil
	}
	directPath := filepath.Join(pluginDir, target)
	info, err := os.Stat(directPath)
	if err != nil {
		return nil, err
	}
	if !isExecutablePlugin(info) {
		return nil, fmt.Errorf("plugin is not executable: %s", strings.TrimSpace(directPath))
	}
	absPath, err := filepath.Abs(directPath)
	if err != nil {
		absPath = directPath
	}
	binary := &PluginBinary{
		Key:  filepath.Base(absPath),
		Name: filepath.Base(absPath),
		Path: absPath,
	}
	if cached, ok := readPluginCache(pluginDir, 365*24*time.Hour); ok {
		for _, item := range cached {
			if strings.EqualFold(strings.TrimSpace(item.Key), target) {
				if strings.TrimSpace(item.Key) != "" {
					binary.Key = strings.TrimSpace(item.Key)
				}
				if strings.TrimSpace(item.Name) != "" {
					binary.Name = strings.TrimSpace(item.Name)
				}
				break
			}
		}
	}
	return binary, nil
}

func GetPluginStatus(target string, flags map[string]string) (*PluginStatus, error) {
	binary, err := ResolvePluginBinary(target)
	if err != nil {
		return nil, err
	}
	pidFile, pid, started := runningPluginPID(binary, flags)
	return &PluginStatus{
		Key:     binary.Key,
		Name:    binary.Name,
		Path:    binary.Path,
		PIDFile: pidFile,
		Started: started,
		PID:     pid,
	}, nil
}

func PluginStatusByKey(target string, flags map[string]string) (*PluginStatus, error) {
	return GetPluginStatus(target, flags)
}

func runningPluginPID(binary *PluginBinary, flags map[string]string) (string, int, bool) {
	for _, pidFile := range pluginPIDFiles(binary, flags) {
		pid, started := runningPID(pidFile)
		if started {
			return pidFile, pid, true
		}
	}
	primary := pluginPIDFile(binary, flags)
	return primary, 0, false
}

func UpsertPluginConfig(flags map[string]string, input MetaInput) (*Meta, error) {
	svc, err := buildService(flags)
	if err != nil {
		return nil, err
	}
	defer svc.Close()

	return UpsertPluginConfigWithService(svc, input)
}

func UpsertPluginConfigWithService(svc *Service, input MetaInput) (*Meta, error) {
	if svc == nil {
		return nil, fmt.Errorf("connect service is required")
	}

	pluginKey := strings.TrimSpace(input.Key)
	pluginBinary, err := ResolvePluginBinaryByKey(pluginKey)
	if err != nil {
		return nil, err
	}
	pluginInfo, err := inspectPlugin(pluginBinary.Path)
	if err != nil {
		return nil, err
	}
	input.Key = pluginBinary.Key
	normalized, err := normalizePluginConfigInput(input, pluginInfo.Scope)
	if err != nil {
		return nil, err
	}
	normalized.Key = pluginBinary.Key
	normalized.Name = pluginBinary.Name
	normalized.Callback = pluginBinary.Path

	item, err := svc.CreateMeta(normalized)
	if err == nil {
		return item, nil
	}
	if !strings.Contains(strings.ToLower(err.Error()), "already exists") {
		return nil, err
	}

	metaValue := normalized.Meta
	modelValue := normalized.Model
	agentIDValue := normalized.AgentID
	chatIDValue := normalized.ChatID
	callbackValue := normalized.Callback
	streamValue := normalized.Stream
	thinkingValue := normalized.Thinking
	verifyValue := normalized.Verify
	routerDisableValue := normalized.RouterDisable

	item, err = svc.UpdateMeta(normalized.Key, MetaUpdate{
		Meta:          &metaValue,
		Stream:        &streamValue,
		Callback:      &callbackValue,
		AgentID:       &agentIDValue,
		ChatID:        &chatIDValue,
		Model:         &modelValue,
		Thinking:      &thinkingValue,
		Verify:        &verifyValue,
		RouterDisable: &routerDisableValue,
	})
	if err != nil {
		return nil, err
	}
	return item, nil
}

func normalizePluginConfigInput(input MetaInput, scope []string) (MetaInput, error) {
	input.Key = strings.TrimSpace(input.Key)
	input.Name = strings.TrimSpace(input.Name)
	input.Meta = strings.TrimSpace(input.Meta)
	input.AgentID = strings.TrimSpace(input.AgentID)
	input.ChatID = strings.TrimSpace(input.ChatID)
	input.Model = strings.TrimSpace(input.Model)
	if input.Key == "" {
		return MetaInput{}, fmt.Errorf("key is required")
	}
	if !pluginScopeEnabled(scope, "reuse") {
		input.ChatID = ""
	}
	if !pluginScopeEnabled(scope, "agent") {
		input.AgentID = ""
	}
	if !pluginScopeEnabled(scope, "provider") {
		input.Model = ""
	}
	if !pluginScopeEnabled(scope, "thinking") {
		input.Thinking = false
		input.Verify = false
	}
	if !input.Thinking {
		input.Verify = false
	}
	if strings.EqualFold(input.Key, "feishu") && input.ChatID == "" {
		input.ChatID = "feishu"
	}
	if pluginScopeEnabled(scope, "agent") && input.AgentID == "" {
		return MetaInput{}, fmt.Errorf("agentId is required")
	}
	if pluginScopeEnabled(scope, "provider") && input.Model == "" {
		return MetaInput{}, fmt.Errorf("model is required")
	}
	if input.Meta == "" {
		input.Meta = "{}"
	}
	return input, nil
}

func ResolvePluginInfoByKey(key string) (*PluginInfo, error) {
	binary, err := ResolvePluginBinaryByKey(key)
	if err != nil {
		return nil, err
	}
	item, err := inspectPlugin(binary.Path)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func resolvePluginScopeByKey(key string) ([]string, bool, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, false, nil
	}
	info, err := ResolvePluginInfoByKey(key)
	if err != nil {
		if isPluginLookupMiss(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return info.Scope, true, nil
}

func isPluginLookupMiss(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(strings.TrimSpace(err.Error()))
	return lower != "" && strings.Contains(lower, "plugin not found")
}

func pluginScopeEnabled(scope []string, key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "" {
		return false
	}
	if scope == nil {
		return true
	}
	for _, item := range scope {
		if strings.EqualFold(strings.TrimSpace(item), key) {
			return true
		}
	}
	return false
}

func scanPlugins(pluginDir string) ([]PluginInfo, error) {
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []PluginInfo{}, nil
		}
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	items := make([]PluginInfo, 0, len(entries))
	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name())
		if shouldSkipPluginEntry(name) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		path := filepath.Join(pluginDir, name)
		if !shouldInspectPluginEntry(path, info) {
			continue
		}
		item, err := inspectPlugin(path)
		if err != nil {
			return nil, fmt.Errorf("inspect plugin %s: %w", name, err)
		}
		items = append(items, item)
	}
	return items, nil
}

func shouldSkipPluginEntry(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || strings.HasPrefix(name, ".") {
		return true
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".json", ".log", ".pid", ".zip":
		return true
	default:
		return false
	}
}

func shouldInspectPluginEntry(path string, info os.FileInfo) bool {
	if info == nil || info.IsDir() || info.Mode()&0o111 == 0 {
		return false
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	header := make([]byte, 4)
	n, err := io.ReadFull(file, header)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return false
	}
	header = header[:n]
	if len(header) >= 2 && header[0] == '#' && header[1] == '!' {
		return true
	}
	if len(header) >= 4 {
		if header[0] == 0x7f && header[1] == 'E' && header[2] == 'L' && header[3] == 'F' {
			return true
		}
		switch {
		case header[0] == 0xfe && header[1] == 0xed && header[2] == 0xfa && (header[3] == 0xce || header[3] == 0xcf):
			return true
		case (header[0] == 0xce || header[0] == 0xcf) && header[1] == 0xfa && header[2] == 0xed && header[3] == 0xfe:
			return true
		}
	}
	if len(header) >= 2 && header[0] == 'M' && header[1] == 'Z' {
		return true
	}
	return false
}

func inspectPlugin(path string) (PluginInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pluginCommandTimeout)
	defer cancel()

	var (
		nameOut    []byte
		paramOut   []byte
		scopeOut   []byte
		commandOut []byte
		nameErr    error
		paramErr   error
		scopeErr   error
		commandErr error
		helpErr    error
	)
	var wg sync.WaitGroup
	wg.Add(5)
	go func() {
		defer wg.Done()
		nameOut, nameErr = runPluginCommand(ctx, path, "name")
	}()
	go func() {
		defer wg.Done()
		paramOut, paramErr = runPluginCommand(ctx, path, "param")
	}()
	go func() {
		defer wg.Done()
		scopeOut, scopeErr = runPluginCommand(ctx, path, "scope")
	}()
	go func() {
		defer wg.Done()
		commandOut, commandErr = runPluginCommand(ctx, path, "command")
	}()
	go func() {
		defer wg.Done()
		_, helpErr = runPluginCommand(ctx, path, "help")
	}()
	wg.Wait()

	if nameErr != nil {
		return PluginInfo{}, nameErr
	}
	if paramErr != nil {
		return PluginInfo{}, paramErr
	}
	if commandErr != nil {
		return PluginInfo{}, commandErr
	}
	if helpErr != nil {
		return PluginInfo{}, helpErr
	}

	key, displayName, err := parsePluginName(nameOut, path)
	if err != nil {
		return PluginInfo{}, err
	}
	params, err := parsePluginParam(paramOut)
	if err != nil {
		return PluginInfo{}, err
	}
	commands, err := parsePluginCommands(commandOut)
	if err != nil {
		return PluginInfo{}, err
	}
	scope, err := parsePluginScope(scopeOut, scopeErr)
	if err != nil {
		return PluginInfo{}, err
	}
	if err := validatePluginCommands(commands); err != nil {
		return PluginInfo{}, err
	}

	return PluginInfo{
		Key:     key,
		Name:    displayName,
		Param:   params,
		Scope:   scope,
		Command: commands,
	}, nil
}

func runPluginActionCommand(path string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pluginActionTimeout)
	defer cancel()

	cmd := pluginCommand(ctx, path, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(out) == 0 {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func runPluginCommand(ctx context.Context, path string, args ...string) ([]byte, error) {
	cmd := pluginCommand(ctx, path, args...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, err
	}
	return out, nil
}

func pluginCommand(ctx context.Context, path string, args ...string) *exec.Cmd {
	cmd := pluginExecCommand(ctx, path, args...)
	dir := filepath.Dir(path)
	if strings.TrimSpace(dir) == "" {
		dir = "."
	}
	cmd.Dir = dir
	return cmd
}

func parsePluginName(out []byte, path string) (string, string, error) {
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return "", "", fmt.Errorf("empty name output")
	}
	defaultKey := strings.TrimSpace(filepath.Base(path))
	var value string
	if err := json.Unmarshal([]byte(raw), &value); err == nil {
		value = strings.TrimSpace(value)
		if value == "" {
			return "", "", fmt.Errorf("empty name output")
		}
		return defaultKey, value, nil
	}
	var object struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(raw), &object); err == nil {
		object.Name = strings.TrimSpace(object.Name)
		if object.Name == "" {
			return "", "", fmt.Errorf("empty name output")
		}
		return defaultKey, object.Name, nil
	}
	return "", "", fmt.Errorf("invalid name output: %s", raw)
}

func parsePluginParam(out []byte) (PluginParamFields, error) {
	return parsePluginParamFields(out)
}

func parsePluginParamFields(out []byte) (PluginParamFields, error) {
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return PluginParamFields{}, nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, fmt.Errorf("invalid param output: %w", err)
	}
	fields := make(PluginParamFields, 0)
	for _, item := range items {
		parsed, err := parsePluginParamObject(item)
		if err != nil {
			return nil, fmt.Errorf("invalid param output: %w", err)
		}
		fields = append(fields, parsed...)
	}
	return fields, nil
}

func parsePluginParamObject(raw []byte) (PluginParamFields, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '{' {
		return nil, fmt.Errorf("param item must be an object")
	}

	fields := make(PluginParamFields, 0)
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyToken.(string)
		if !ok {
			return nil, fmt.Errorf("param key must be a string")
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("param key is required")
		}

		var placeholder string
		if err := decoder.Decode(&placeholder); err != nil {
			return nil, err
		}
		fields = append(fields, PluginParamField{
			Key:         key,
			Placeholder: placeholder,
		})
	}

	endToken, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	endDelim, ok := endToken.(json.Delim)
	if !ok || endDelim != '}' {
		return nil, fmt.Errorf("param item must be an object")
	}
	return fields, nil
}

func parsePluginScope(out []byte, cmdErr error) ([]string, error) {
	if cmdErr != nil {
		if isMissingScopeCommandError(cmdErr) {
			return nil, nil
		}
		return nil, cmdErr
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return []string{}, nil
	}
	var value []string
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, fmt.Errorf("invalid scope output: %w", err)
	}
	return normalizePluginScope(value)
}

func isMissingScopeCommandError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(strings.TrimSpace(err.Error()))
	if lower == "" {
		return false
	}
	return strings.Contains(lower, "scope") &&
		(strings.Contains(lower, "unknown command") ||
			strings.Contains(lower, "unknown subcommand") ||
			strings.Contains(lower, "unsupported command") ||
			strings.Contains(lower, "invalid command") ||
			strings.Contains(lower, "not implemented"))
}

func normalizePluginScope(scope []string) ([]string, error) {
	if len(scope) == 0 {
		return []string{}, nil
	}
	allowed := map[string]struct{}{
		"reuse":    {},
		"agent":    {},
		"provider": {},
		"thinking": {},
		"swarm":    {},
	}
	out := make([]string, 0, len(scope))
	seen := make(map[string]struct{}, len(scope))
	for _, item := range scope {
		key := strings.ToLower(strings.TrimSpace(item))
		if key == "" {
			continue
		}
		if _, ok := allowed[key]; !ok {
			return nil, fmt.Errorf("unsupported scope entry: %s", item)
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out, nil
}

func parsePluginCommands(out []byte) ([]string, error) {
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, fmt.Errorf("empty command output")
	}
	var value []string
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, fmt.Errorf("invalid command output: %w", err)
	}
	return value, nil
}

func validatePluginCommands(commands []string) error {
	required := []string{"command", "help", "name", "param", "start", "stop"}
	for _, requiredCommand := range required {
		found := false
		for _, command := range commands {
			if strings.EqualFold(strings.TrimSpace(command), requiredCommand) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("command output missing %s", requiredCommand)
		}
	}
	return nil
}

func resolvePluginBinaryFromPath(path string) (*PluginBinary, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("plugin not found: %s", strings.TrimSpace(path))
		}
		return nil, err
	}
	if !isExecutablePlugin(info) {
		return nil, fmt.Errorf("plugin is not executable: %s", strings.TrimSpace(path))
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	binary := &PluginBinary{
		Key:  filepath.Base(absPath),
		Name: filepath.Base(absPath),
		Path: absPath,
	}
	if item, err := inspectPlugin(absPath); err == nil {
		if strings.TrimSpace(item.Key) != "" {
			binary.Key = item.Key
		}
		if strings.TrimSpace(item.Name) != "" {
			binary.Name = item.Name
		}
	}
	return binary, nil
}

func isExecutablePlugin(info os.FileInfo) bool {
	return info != nil && !info.IsDir() && info.Mode()&0o111 != 0
}

func isPluginPathLike(target string) bool {
	return filepath.IsAbs(target) || strings.HasPrefix(target, ".") || strings.Contains(target, "/") || strings.Contains(target, string(filepath.Separator))
}

func buildPluginActionArgs(flags map[string]string) []string {
	if len(flags) == 0 {
		return nil
	}
	keys := make([]string, 0, len(flags))
	for key := range flags {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		keys = append(keys, trimmed)
	}
	sort.Strings(keys)

	args := make([]string, 0, len(keys)*2)
	for _, key := range keys {
		args = append(args, "--"+key)
		if value := strings.TrimSpace(flags[key]); value != "" {
			args = append(args, value)
		}
	}
	return args
}

func pluginPIDFile(binary *PluginBinary, flags map[string]string) string {
	if path := strings.TrimSpace(FirstValue(flags, "pid-file")); path != "" {
		if abs, err := filepath.Abs(path); err == nil {
			return abs
		}
		return path
	}
	base := strings.TrimSpace(binary.Key)
	if base == "" {
		base = strings.TrimSpace(filepath.Base(binary.Path))
	}
	if base == "" {
		base = "plugin"
	}
	path := filepath.Join(filepath.Dir(binary.Path), base+".pid")
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

func pluginPIDFiles(binary *PluginBinary, flags map[string]string) []string {
	primary := pluginPIDFile(binary, flags)
	return []string{primary}
}

func readPluginCache(pluginDir string, ttl time.Duration) ([]PluginInfo, bool) {
	if ttl <= 0 {
		return nil, false
	}
	path := filepath.Join(pluginDir, pluginCacheFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var cached pluginCache
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, false
	}
	if cached.UpdatedAtUnixMilli <= 0 {
		return nil, false
	}
	updatedAt := time.UnixMilli(cached.UpdatedAtUnixMilli)
	if pluginNow().After(updatedAt.Add(ttl)) {
		return nil, false
	}
	return cached.Items, true
}

func writePluginCache(pluginDir string, items []PluginInfo) error {
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(pluginDir, pluginCacheFileName)
	tempPath := path + ".tmp"
	payload, err := json.MarshalIndent(pluginCache{
		UpdatedAtUnixMilli: pluginNow().UnixMilli(),
		Items:              items,
	}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tempPath, payload, 0o644); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func defaultPluginDir() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("DEEPRIGHT_PLUGIN_DIR")); explicit != "" {
		if !filepath.IsAbs(explicit) {
			if abs, err := filepath.Abs(explicit); err == nil {
				explicit = abs
			}
		}
		return filepath.Clean(explicit), nil
	}

	exe, err := pluginExecutableResolver()
	if err != nil {
		return "", fmt.Errorf("resolve runtime plugins dir: %w", err)
	}
	exe = strings.TrimSpace(exe)
	if exe == "" {
		return "", fmt.Errorf("resolve runtime plugins dir")
	}
	if abs, err := filepath.Abs(exe); err == nil {
		exe = abs
	}
	runtimeConfigPath, ok := resolveStrictRuntimeConfigPathNearBinary(exe)
	if !ok {
		return "", fmt.Errorf("resolve runtime plugins dir from executable: %s", exe)
	}
	cfg, err := ReadRuntimeConfig(runtimeConfigPath)
	if err != nil {
		return "", fmt.Errorf("read runtime config %s: %w", runtimeConfigPath, err)
	}
	appDir := strings.TrimSpace(ResolveRuntimePathValue(runtimeConfigPath, cfg["app-dir"]))
	if appDir == "" {
		return "", fmt.Errorf("resolve app-dir from runtime config: %s", runtimeConfigPath)
	}
	return filepath.Clean(filepath.Join(appDir, "plugins")), nil
}

func resolveStrictRuntimeConfigPathNearBinary(binaryPath string) (string, bool) {
	binaryPath = strings.TrimSpace(binaryPath)
	if binaryPath == "" {
		return "", false
	}
	if abs, err := filepath.Abs(binaryPath); err == nil {
		binaryPath = abs
	}
	if bundledPath := ResolveBundledRuntimeConfigPath(binaryPath); bundledPath != "" {
		if info, err := os.Stat(bundledPath); err == nil && !info.IsDir() {
			return bundledPath, true
		}
		return "", false
	}
	return resolveRuntimeConfigPathCandidates(filepath.Dir(binaryPath), []string{
		filepath.Join("config", "config.json"),
		filepath.Join("..", "config", "config.json"),
		filepath.Join("..", "..", "config", "config.json"),
	})
}

var pluginDirResolver = defaultPluginDir
var pluginExecCommand = exec.CommandContext
var pluginNow = time.Now
var pluginExecutableResolver = os.Executable

func SetPluginDirResolverForTest(fn func() (string, error)) func() {
	original := pluginDirResolver
	if fn == nil {
		pluginDirResolver = defaultPluginDir
	} else {
		pluginDirResolver = fn
	}
	return func() {
		pluginDirResolver = original
	}
}

func SetOSExecutableForTest(fn func() (string, error)) func() {
	original := pluginExecutableResolver
	if fn == nil {
		pluginExecutableResolver = os.Executable
	} else {
		pluginExecutableResolver = fn
	}
	return func() {
		pluginExecutableResolver = original
	}
}
