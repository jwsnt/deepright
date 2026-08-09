package runtimecfg

import (
	"path/filepath"
	"testing"
	"time"
)

func TestParseTimeoutMS(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		want   time.Duration
		wantOK bool
	}{
		{"empty", "", 0, false},
		{"valid ms", "30000", 30000 * time.Millisecond, true},
		{"zero", "0", 0, false},
		{"negative", "-1", 0, false},
		{"invalid string", "abc", 0, false},
		{"with spaces", "  5000  ", 5000 * time.Millisecond, true},
		{"large value", "3600000", 3600000 * time.Millisecond, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseTimeoutMS(tt.raw)
			if ok != tt.wantOK {
				t.Errorf("ParseTimeoutMS() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("ParseTimeoutMS() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolvePluginTimeoutsDefaults(t *testing.T) {
	timeouts := ResolvePluginTimeouts(map[string]string{})
	if timeouts.Exec != DefaultCommandTimeout {
		t.Errorf("Exec = %v, want %v", timeouts.Exec, DefaultCommandTimeout)
	}
	if timeouts.SCP != DefaultSCPTimeout {
		t.Errorf("SCP = %v, want %v", timeouts.SCP, DefaultSCPTimeout)
	}
}

func TestResolvePluginTimeoutsWithTimeoutFlag(t *testing.T) {
	timeouts := ResolvePluginTimeouts(map[string]string{"timeout": "10000"})
	want := 10000 * time.Millisecond
	if timeouts.Exec != want {
		t.Errorf("Exec = %v, want %v", timeouts.Exec, want)
	}
	if timeouts.SCP != want {
		t.Errorf("SCP = %v, want %v", timeouts.SCP, want)
	}
}

func TestNormalizeBinaryPath(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"empty", ""},
		{"absolute path", "/usr/bin/integration"},
		{"simple name", "integration"},
		{"relative with dot", "./relative/path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeBinaryPath(tt.value)
			if tt.value == "" {
				if got != "" {
					t.Errorf("normalizeBinaryPath(%q) = %q, want ''", tt.value, got)
				}
				return
			}
			if got == "" {
				t.Errorf("normalizeBinaryPath(%q) = ''", tt.value)
			}
		})
	}
}

func TestNormalizeBinaryPathExpandsRelative(t *testing.T) {
	got := normalizeBinaryPath("./test/foo")
	if !filepath.IsAbs(got) {
		t.Errorf("normalizeBinaryPath('./test/foo') = %q, expected absolute path", got)
	}
}

func TestParseTimeoutSeconds(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		want   time.Duration
		wantOK bool
	}{
		{"valid seconds", "300", 300 * time.Second, true},
		{"empty", "", 0, false},
		{"zero", "0", 0, false},
		{"negative", "-1", 0, false},
		{"invalid", "three hundred", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseTimeoutSeconds(tt.raw)
			if ok != tt.wantOK {
				t.Errorf("ParseTimeoutSeconds() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("ParseTimeoutSeconds() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPluginMetaBodyStringValue(t *testing.T) {
	tests := []struct {
		name string
		body pluginMetaBody
		key  string
		want string
	}{
		{"nil body", nil, "key", ""},
		{"string value", pluginMetaBody{"key": "value"}, "key", "value"},
		{"int value", pluginMetaBody{"key": 42}, "key", "42"},
		{"float value", pluginMetaBody{"key": 3.14}, "key", "3.14"},
		{"missing key", pluginMetaBody{"other": "value"}, "key", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.body.stringValue(tt.key); got != tt.want {
				t.Errorf("stringValue() = %q, want %q", got, tt.want)
			}
		})
	}
}
