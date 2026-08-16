package connectsvc

import (
	"bytes"
	"connect/sharedutil"
	"context"
	"encoding/json"
	"errors"
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

const localPluginMetadataOutputLimit = 64 * 1024

var localPluginProbes = struct {
	sync.Mutex
	running map[string]*localPluginProbe
}{
	running: make(map[string]*localPluginProbe),
}

type localPluginProbe struct {
	done chan struct{}

	mu       sync.Mutex
	output   []byte
	err      error
	timedOut bool
}

type localPluginCommandTimeoutError struct {
	Command string
	Timeout time.Duration
}

func (e *localPluginCommandTimeoutError) Error() string {
	return fmt.Sprintf("plugin metadata command timed out after %s; process was left running (command=%s)", e.Timeout, e.Command)
}

type localPluginCommandInProgressError struct {
	Command string
}

func (e *localPluginCommandInProgressError) Error() string {
	return fmt.Sprintf("plugin metadata command is still running after an earlier timeout (command=%s)", e.Command)
}

type localPluginCommandExitError struct {
	Command string
	Err     error
	Stderr  string
}

func (e *localPluginCommandExitError) Error() string {
	state := strings.TrimSpace(e.Err.Error())
	if exitErr, ok := e.Err.(*exec.ExitError); ok && exitErr.ProcessState != nil {
		state = strings.TrimSpace(exitErr.ProcessState.String())
	}
	if strings.Contains(strings.ToLower(state), "signal:") {
		return fmt.Sprintf("plugin metadata command exited due to %s (command=%s)", state, e.Command)
	}
	if stderr := strings.TrimSpace(e.Stderr); stderr != "" {
		return fmt.Sprintf("plugin metadata command failed: %s (command=%s): %s", state, e.Command, stderr)
	}
	return fmt.Sprintf("plugin metadata command failed: %s (command=%s)", state, e.Command)
}

func (e *localPluginCommandExitError) Unwrap() error {
	return e.Err
}

type localPluginOutputBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *localPluginOutputBuffer) Write(p []byte) (int, error) {
	originalLen := len(p)
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := localPluginMetadataOutputLimit - b.buf.Len()
	if remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		_, _ = b.buf.Write(p)
	}
	return originalLen, nil
}

func (b *localPluginOutputBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.buf.Bytes()...)
}

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
			var inProgress *localPluginCommandInProgressError
			if errors.As(err, &inProgress) {
				// The original timeout has already been recorded. Avoid turning a
				// temporarily stuck probe into a high-frequency log storm.
				continue
			}
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
	startedAt := time.Now()
	command := strings.Join(args, " ")
	key := path + "\x00" + command

	localPluginProbes.Lock()
	if _, exists := localPluginProbes.running[key]; exists {
		localPluginProbes.Unlock()
		return nil, &localPluginCommandInProgressError{Command: command}
	}
	probe := &localPluginProbe{done: make(chan struct{})}
	localPluginProbes.running[key] = probe
	localPluginProbes.Unlock()

	cmd, err := buildLocalPluginCommand(path, args...)
	if err != nil {
		removeLocalPluginProbe(key, probe)
		return nil, err
	}
	var stdout, stderr localPluginOutputBuffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		removeLocalPluginProbe(key, probe)
		return nil, err
	}

	go waitForLocalPluginProbe(key, probe, cmd, command, filepath.Base(path), &stdout, &stderr)

	select {
	case <-probe.done:
		probe.mu.Lock()
		defer probe.mu.Unlock()
		return append([]byte(nil), probe.output...), probe.err
	case <-ctx.Done():
		select {
		case <-probe.done:
			probe.mu.Lock()
			defer probe.mu.Unlock()
			return append([]byte(nil), probe.output...), probe.err
		default:
		}
		probe.mu.Lock()
		probe.timedOut = true
		probe.mu.Unlock()
		return nil, &localPluginCommandTimeoutError{Command: command, Timeout: time.Since(startedAt).Round(time.Millisecond)}
	}
}

func waitForLocalPluginProbe(key string, probe *localPluginProbe, cmd *exec.Cmd, command, plugin string, stdout, stderr *localPluginOutputBuffer) {
	err := cmd.Wait()
	if err != nil {
		err = &localPluginCommandExitError{Command: command, Err: err, Stderr: string(stderr.Bytes())}
	}

	probe.mu.Lock()
	probe.output = stdout.Bytes()
	probe.err = err
	timedOut := probe.timedOut
	probe.mu.Unlock()
	close(probe.done)
	removeLocalPluginProbe(key, probe)

	if !timedOut {
		return
	}
	if err != nil {
		log.Printf("plugins meta probe completed after timeout plugin=%s command=%s: %v", plugin, command, err)
		return
	}
	log.Printf("plugins meta probe completed after timeout plugin=%s command=%s: success", plugin, command)
}

func removeLocalPluginProbe(key string, probe *localPluginProbe) {
	localPluginProbes.Lock()
	defer localPluginProbes.Unlock()
	if localPluginProbes.running[key] == probe {
		delete(localPluginProbes.running, key)
	}
}

func buildLocalPluginCommand(path string, args ...string) (*exec.Cmd, error) {
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
			cmd = exec.Command(path, args...)
			break
		}
		pythonBin, err := resolveLocalInterpreter("python3", "python")
		if err != nil {
			return nil, err
		}
		cmd = exec.Command(pythonBin, append([]string{path}, args...)...)
	case ".js":
		if info.Mode()&0o111 != 0 {
			cmd = exec.Command(path, args...)
			break
		}
		nodeBin, err := resolveLocalInterpreter("node")
		if err != nil {
			return nil, err
		}
		cmd = exec.Command(nodeBin, append([]string{path}, args...)...)
	case ".go":
		if info.Mode()&0o111 != 0 {
			cmd = exec.Command(path, args...)
			break
		}
		goBin, err := resolveLocalInterpreter("go")
		if err != nil {
			return nil, err
		}
		cmd = exec.Command(goBin, append([]string{"run", path}, args...)...)
	default:
		cmd = exec.Command(path, args...)
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
