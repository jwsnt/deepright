package pluginlog

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var pluginLogExecutable = os.Executable
var pluginLogGetwd = os.Getwd

const (
	DefaultLastLines    = 10
	DefaultPollInterval = 300 * time.Millisecond
)

type Options struct {
	LastLines     int
	PollInterval  time.Duration
	WaitForCreate time.Duration
}

type Event struct {
	Type string
	Data string
}

func ResolveLogPath(plugin string) (string, error) {
	plugin = strings.TrimSpace(plugin)
	if plugin == "" {
		return "", fmt.Errorf("plugin is required")
	}

	resolved, err := resolvePluginPath(plugin)
	if err != nil {
		return "", err
	}
	if strings.HasSuffix(strings.ToLower(resolved), ".log") {
		return filepath.Abs(resolved)
	}
	return filepath.Abs(resolved + ".log")
}

func ServeSSE(ctx context.Context, w http.ResponseWriter, logPath string, opts Options) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("response writer does not support streaming")
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	err := TailFile(ctx, logPath, opts, func(event Event) error {
		return writeSSEEvent(w, flusher, event.Type, event.Data)
	})
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if os.IsNotExist(err) {
		_ = writeSSEEvent(w, flusher, "error", "log file not found: "+logPath)
		return err
	}
	_ = writeSSEEvent(w, flusher, "error", err.Error())
	return err
}

func TailFile(ctx context.Context, logPath string, opts Options, emit func(Event) error) error {
	opts = normalizeOptions(opts)

	file, lines, offset, err := openAndReadTail(logPath, opts.LastLines)
	if err != nil {
		if os.IsNotExist(err) && opts.WaitForCreate > 0 {
			file, lines, offset, err = waitForLogFile(ctx, logPath, opts)
		}
	}
	if err != nil {
		return err
	}
	defer file.Close()

	for _, line := range lines {
		if err := emit(Event{Type: "log", Data: line}); err != nil {
			return err
		}
	}

	pending := ""
	ticker := time.NewTicker(opts.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		info, err := os.Stat(logPath)
		if err != nil {
			return err
		}

		if info.Size() < offset {
			if err := file.Close(); err != nil {
				return err
			}
			file, _, offset, err = openAndReadTail(logPath, 0)
			if err != nil {
				return err
			}
			pending = ""
		}

		if info.Size() == offset {
			continue
		}

		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			return err
		}
		data, err := io.ReadAll(file)
		if err != nil {
			return err
		}
		offset += int64(len(data))

		pending, err = emitBufferedLines(pending+string(data), emit)
		if err != nil {
			return err
		}
	}
}

func normalizeOptions(opts Options) Options {
	if opts.LastLines < 0 {
		opts.LastLines = 0
	}
	if opts.WaitForCreate < 0 {
		opts.WaitForCreate = 0
	}
	if opts.LastLines == 0 && opts.PollInterval == 0 {
		opts.LastLines = DefaultLastLines
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = DefaultPollInterval
	}
	return opts
}

func waitForLogFile(ctx context.Context, logPath string, opts Options) (*os.File, []string, int64, error) {
	deadline := time.Now().Add(opts.WaitForCreate)
	for {
		file, lines, offset, err := openAndReadTail(logPath, opts.LastLines)
		if err == nil {
			return file, lines, offset, nil
		}
		if !os.IsNotExist(err) {
			return nil, nil, 0, err
		}
		if ctx.Err() != nil {
			return nil, nil, 0, ctx.Err()
		}
		if time.Now().After(deadline) {
			return nil, nil, 0, err
		}
		timer := time.NewTimer(opts.PollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, nil, 0, ctx.Err()
		case <-timer.C:
		}
	}
}

func openAndReadTail(path string, lastLines int) (*os.File, []string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, 0, err
	}

	data, err := io.ReadAll(file)
	if err != nil {
		file.Close()
		return nil, nil, 0, err
	}

	lines := splitLines(string(data))
	if lastLines > 0 && len(lines) > lastLines {
		lines = lines[len(lines)-lastLines:]
	}
	return file, lines, int64(len(data)), nil
}

func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	parts := strings.Split(text, "\n")
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	for i := range parts {
		parts[i] = strings.TrimSuffix(parts[i], "\r")
	}
	return parts
}

func emitBufferedLines(text string, emit func(Event) error) (string, error) {
	parts := strings.Split(text, "\n")
	if len(parts) == 1 {
		return text, nil
	}
	for _, part := range parts[:len(parts)-1] {
		if err := emit(Event{Type: "log", Data: strings.TrimSuffix(part, "\r")}); err != nil {
			return "", err
		}
	}
	return parts[len(parts)-1], nil
}

func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType, data string) error {
	if eventType == "" {
		eventType = "message"
	}
	if _, err := fmt.Fprintf(w, "event: %s\n", eventType); err != nil {
		return err
	}
	lines := strings.Split(data, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	for _, line := range lines {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(w, "\n"); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

func resolvePluginPath(plugin string) (string, error) {
	plugin = expandHome(plugin)
	if filepath.IsAbs(plugin) || strings.HasPrefix(plugin, ".") || strings.Contains(plugin, "/") || strings.Contains(plugin, string(filepath.Separator)) {
		return filepath.Clean(plugin), nil
	}

	dir, err := defaultPluginDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, plugin), nil
}

func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func defaultPluginDir() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("DEEPRIGHT_PLUGIN_DIR")); explicit != "" {
		return filepath.Clean(explicit), nil
	}
	if runtime.GOOS == "darwin" {
		exe, err := pluginLogExecutable()
		if err != nil {
			return "", fmt.Errorf("resolve plugins dir: %w", err)
		}
		exe = strings.TrimSpace(exe)
		if exe == "" {
			return "", fmt.Errorf("resolve plugins dir")
		}
		if abs, err := filepath.Abs(exe); err == nil {
			exe = abs
		}
		macOSDir := filepath.Dir(exe)
		contentsDir := filepath.Dir(macOSDir)
		if filepath.Base(macOSDir) == "MacOS" &&
			filepath.Base(contentsDir) == "Contents" &&
			strings.HasSuffix(strings.ToLower(filepath.Dir(contentsDir)), ".app") {
			return filepath.Join(contentsDir, "Resources", "plugins"), nil
		}
		return filepath.Join(filepath.Dir(exe), "plugins"), nil
	}
	candidates := make([]string, 0, 3)
	if exe, err := pluginLogExecutable(); err == nil && strings.TrimSpace(exe) != "" {
		candidates = append(candidates, filepath.Clean(filepath.Join(filepath.Dir(exe), "plugins")))
	}
	if cwd, err := pluginLogGetwd(); err == nil && strings.TrimSpace(cwd) != "" {
		candidates = append(candidates,
			filepath.Clean(filepath.Join(cwd, "plugins")),
			filepath.Clean(filepath.Join(cwd, "..", "plugins")),
		)
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, nil
		}
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("resolve plugins dir")
	}
	return candidates[0], nil
}
