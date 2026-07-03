package pluginlog

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestNormalizeOptions(t *testing.T) {
	tests := []struct {
		name string
		opts Options
		want Options
	}{
		{"zero value", Options{}, Options{LastLines: DefaultLastLines, PollInterval: DefaultPollInterval}},
		{"custom last lines", Options{LastLines: 5}, Options{LastLines: 5, PollInterval: DefaultPollInterval}},
		{"negative last lines", Options{LastLines: -1}, Options{LastLines: DefaultLastLines, PollInterval: DefaultPollInterval}},
		{"custom poll interval", Options{PollInterval: 1 * time.Second}, Options{LastLines: 0, PollInterval: 1 * time.Second}},
		{"both custom", Options{LastLines: 3, PollInterval: 500 * time.Millisecond}, Options{LastLines: 3, PollInterval: 500 * time.Millisecond}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeOptions(tt.opts)
			if got.LastLines != tt.want.LastLines {
				t.Errorf("LastLines = %d, want %d", got.LastLines, tt.want.LastLines)
			}
			if got.PollInterval != tt.want.PollInterval {
				t.Errorf("PollInterval = %v, want %v", got.PollInterval, tt.want.PollInterval)
			}
		})
	}
}

func TestResolveLogPath(t *testing.T) {
	tests := []struct {
		name    string
		plugin  string
		wantErr bool
	}{
		{"empty plugin", "", true},
		{"simple plugin name (non-absolute)", "myplugin", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveLogPath(tt.plugin)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveLogPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResolveLogPathPrefersEnvPluginDirOnDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only plugin dir behavior")
	}

	pluginDir := t.TempDir()
	if err := os.Setenv("DEEPRIGHT_PLUGIN_DIR", pluginDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Unsetenv("DEEPRIGHT_PLUGIN_DIR") }()

	got, err := ResolveLogPath("feishu")
	if err != nil {
		t.Fatalf("ResolveLogPath failed: %v", err)
	}
	want := filepath.Join(pluginDir, "feishu.log")
	if got != want {
		t.Fatalf("ResolveLogPath = %q, want %q", got, want)
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{"empty", "", nil},
		{"single line", "hello", []string{"hello"}},
		{"multiple lines", "line1\nline2\nline3", []string{"line1", "line2", "line3"}},
		{"trailing newline", "line1\nline2\n", []string{"line1", "line2"}},
		{"with carriage return", "line1\r\nline2\r\n", []string{"line1", "line2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitLines(tt.text)
			if len(got) != len(tt.want) {
				t.Errorf("splitLines() = %v (len=%d), want %v (len=%d)", got, len(got), tt.want, len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitLines()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestOpenAndReadTail(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	content := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(logFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		lastLines int
		wantLines int
	}{
		{"all lines", 0, 5},
		{"last 3", 3, 3},
		{"more than available", 10, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, lines, offset, err := openAndReadTail(logFile, tt.lastLines)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()

			if len(lines) != tt.wantLines {
				t.Errorf("got %d lines, want %d: %v", len(lines), tt.wantLines, lines)
			}
			if offset != int64(len(content)) {
				t.Errorf("offset = %d, want %d", offset, len(content))
			}
		})
	}
}

func TestEmitBufferedLines(t *testing.T) {
	var emitted []string
	emit := func(e Event) error {
		emitted = append(emitted, e.Data)
		return nil
	}

	tests := []struct {
		name        string
		text        string
		wantPending string
		wantEvents  int
	}{
		{"single line no newline", "hello", "hello", 0},
		{"two lines", "hello\nworld\n", "", 2},
		{"partial last line", "hello\nworld", "world", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emitted = nil
			pending, err := emitBufferedLines(tt.text, emit)
			if err != nil {
				t.Fatal(err)
			}
			if pending != tt.wantPending {
				t.Errorf("pending = %q, want %q", pending, tt.wantPending)
			}
			if len(emitted) != tt.wantEvents {
				t.Errorf("emitted %d events, want %d", len(emitted), tt.wantEvents)
			}
		})
	}
}

func TestTailFileWaitsForCreate(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "delayed.log")

	go func() {
		time.Sleep(40 * time.Millisecond)
		_ = os.WriteFile(logPath, []byte("created\n"), 0o644)
	}()

	stopErr := errors.New("stop")
	var lines []string
	err := TailFile(context.Background(), logPath, Options{
		LastLines:     10,
		PollInterval:  10 * time.Millisecond,
		WaitForCreate: 300 * time.Millisecond,
	}, func(event Event) error {
		lines = append(lines, event.Data)
		return stopErr
	})
	if !errors.Is(err, stopErr) {
		t.Fatalf("TailFile() error = %v, want %v", err, stopErr)
	}
	if len(lines) != 1 || lines[0] != "created" {
		t.Fatalf("lines = %v, want [created]", lines)
	}
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	tests := []struct {
		name string
		path string
		want string
	}{
		{"tilde only", "~", home},
		{"tilde with path", "~/test", home + "/test"},
		{"no tilde", "/tmp/test", "/tmp/test"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandHome(tt.path)
			if got != tt.want {
				t.Errorf("expandHome(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
