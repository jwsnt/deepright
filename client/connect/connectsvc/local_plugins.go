package connectsvc

import (
	"connect/sharedutil"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const localPluginCommandTimeout = 5 * time.Second

func init() {
	sharedutil.ApplySystemPath()
}

type LocalPluginOptions struct {
	ResolveDir func() (string, error)
}

func DefaultLocalPluginDir() (string, error) {
	if cwd, err := os.Getwd(); err == nil && strings.TrimSpace(cwd) != "" {
		return filepath.Clean(filepath.Join(cwd, "plugins")), nil
	}
	return "", fmt.Errorf("resolve startup cwd for plugins dir")
}

func ListLocalPlugins(opts LocalPluginOptions) ([]PluginInfo, error) {
	pluginDir, err := resolveLocalPluginDir(opts)
	if err != nil {
		return nil, err
	}
	return scanLocalPlugins(pluginDir)
}

func ListLocalPluginMetaWithService(svc *Service, opts LocalPluginOptions) ([]PluginMetaInfo, error) {
	if svc == nil {
		return nil, fmt.Errorf("connect service is required")
	}

	items, err := ListLocalPlugins(opts)
	if err != nil {
		return nil, err
	}
	return listPluginMetaItems(svc, items)
}

func resolveLocalPluginDir(opts LocalPluginOptions) (string, error) {
	resolver := opts.ResolveDir
	if resolver == nil {
		return "", fmt.Errorf("resolve local plugin dir")
	}
	return resolver()
}

func scanLocalPlugins(pluginDir string) ([]PluginInfo, error) {
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
		if name == "" || strings.HasPrefix(name, ".") || entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			log.Printf("plugins meta skip %s: read file info failed: %v", name, err)
			continue
		}
		if !shouldInspectLocalPluginEntry(name, info) {
			continue
		}
		path := filepath.Join(pluginDir, name)
		item, err := inspectLocalPlugin(path)
		if err != nil {
			log.Printf("plugins meta skip %s: %v", name, err)
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func shouldInspectLocalPluginEntry(name string, info os.FileInfo) bool {
	if info == nil || info.IsDir() {
		return false
	}
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(name))) {
	case "":
		return info.Mode()&0o111 != 0
	case ".py", ".js", ".go":
		return true
	default:
		return false
	}
}

func inspectLocalPlugin(path string) (PluginInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), localPluginCommandTimeout)
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
		nameOut, nameErr = runLocalPluginCommand(ctx, path, "name")
	}()
	go func() {
		defer wg.Done()
		paramOut, paramErr = runLocalPluginCommand(ctx, path, "param")
	}()
	go func() {
		defer wg.Done()
		scopeOut, scopeErr = runLocalPluginCommand(ctx, path, "scope")
	}()
	go func() {
		defer wg.Done()
		commandOut, commandErr = runLocalPluginCommand(ctx, path, "command")
	}()
	go func() {
		defer wg.Done()
		_, helpErr = runLocalPluginCommand(ctx, path, "help")
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

	key, displayName, err := parseLocalPluginName(nameOut, path)
	if err != nil {
		return PluginInfo{}, err
	}
	params, err := parseLocalPluginParam(paramOut)
	if err != nil {
		return PluginInfo{}, err
	}
	commands, err := parseLocalPluginCommands(commandOut)
	if err != nil {
		return PluginInfo{}, err
	}
	scope, err := parseLocalPluginScope(scopeOut, scopeErr)
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

func runLocalPluginCommand(ctx context.Context, path string, args ...string) ([]byte, error) {
	cmd, err := buildLocalPluginCommand(ctx, path, args...)
	if err != nil {
		return nil, err
	}
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, err
	}
	return out, nil
}

func buildLocalPluginCommand(ctx context.Context, path string, args ...string) (*exec.Cmd, error) {
	dir := filepath.Dir(path)
	if strings.TrimSpace(dir) == "" {
		dir = "."
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	var cmd *exec.Cmd
	switch strings.ToLower(filepath.Ext(path)) {
	case ".py":
		if info.Mode()&0o111 != 0 {
			cmd = exec.CommandContext(ctx, path, args...)
			break
		}
		pythonBin, err := resolveLocalInterpreter("python3", "python")
		if err != nil {
			return nil, err
		}
		cmd = exec.CommandContext(ctx, pythonBin, append([]string{path}, args...)...)
	case ".js":
		if info.Mode()&0o111 != 0 {
			cmd = exec.CommandContext(ctx, path, args...)
			break
		}
		nodeBin, err := resolveLocalInterpreter("node")
		if err != nil {
			return nil, err
		}
		cmd = exec.CommandContext(ctx, nodeBin, append([]string{path}, args...)...)
	case ".go":
		if info.Mode()&0o111 != 0 {
			cmd = exec.CommandContext(ctx, path, args...)
			break
		}
		goBin, err := resolveLocalInterpreter("go")
		if err != nil {
			return nil, err
		}
		cmd = exec.CommandContext(ctx, goBin, append([]string{"run", path}, args...)...)
	default:
		cmd = exec.CommandContext(ctx, path, args...)
	}

	cmd.Dir = dir
	return cmd, nil
}

func resolveLocalInterpreter(candidates ...string) (string, error) {
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		path, err := exec.LookPath(candidate)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("required runtime not found")
}

func parseLocalPluginName(out []byte, path string) (string, string, error) {
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return "", "", fmt.Errorf("empty name output")
	}

	defaultKey := defaultLocalPluginKey(path)
	var value string
	if err := json.Unmarshal([]byte(raw), &value); err == nil {
		value = strings.TrimSpace(value)
		if value == "" {
			return "", "", fmt.Errorf("empty name output")
		}
		return defaultKey, value, nil
	}

	var object struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(raw), &object); err == nil {
		object.Key = strings.TrimSpace(object.Key)
		object.Name = strings.TrimSpace(object.Name)
		if object.Name == "" {
			return "", "", fmt.Errorf("empty name output")
		}
		if object.Key == "" {
			object.Key = defaultKey
		}
		return object.Key, object.Name, nil
	}
	return "", "", fmt.Errorf("invalid name output: %s", raw)
}

func defaultLocalPluginKey(path string) string {
	base := strings.TrimSpace(filepath.Base(path))
	switch strings.ToLower(filepath.Ext(base)) {
	case ".py", ".js", ".go":
		return strings.TrimSpace(strings.TrimSuffix(base, filepath.Ext(base)))
	default:
		return base
	}
}

func parseLocalPluginParam(out []byte) (PluginParamFields, error) {
	return parsePluginParamFields(out)
}

func parseLocalPluginScope(out []byte, cmdErr error) ([]string, error) {
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

func parseLocalPluginCommands(out []byte) ([]string, error) {
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
