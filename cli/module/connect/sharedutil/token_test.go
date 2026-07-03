package sharedutil

import (
	"testing"
)

func TestNormalizeTokenModelName(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  string
	}{
		{"empty", "", ""},
		{"already normalized", "deepright", "deepright"},
		{"alias aioright", "aioright", "deepright"},
		{"alias AioRight", "AioRight", "deepright"},
		{"alias aiorhight", "aiorhight", "deepright"},
		{"alias aiohright", "aiohright", "deepright"},
		{"alias OpenAI", "OpenAI", "openai"},
		{"alias OPENAI", "OPENAI", "openai"},
		{"unknown model", "gpt-4", "gpt-4"},
		{"with spaces", "  deepright ", "deepright"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeTokenModelName(tt.model)
			if got != tt.want {
				t.Errorf("NormalizeTokenModelName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeTaskType(t *testing.T) {
	tests := []struct {
		name     string
		taskType string
		want     string
	}{
		{"empty", "", DefaultTaskType},
		{"blank spaces", "  ", DefaultTaskType},
		{"custom value", "daily", "daily"},
		{"with spaces", "  cron  ", "cron"},
		{"default const", DefaultTaskType, DefaultTaskType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeTaskType(tt.taskType)
			if got != tt.want {
				t.Errorf("NormalizeTaskType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefaultTaskTypeConst(t *testing.T) {
	if DefaultTaskType != "cron" {
		t.Errorf("DefaultTaskType = %q, want %q", DefaultTaskType, "cron")
	}
}
