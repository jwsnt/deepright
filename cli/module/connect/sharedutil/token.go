package sharedutil

import "strings"

// TokenModelAliases normalizes common misspellings of model provider names.
var TokenModelAliases = map[string]string{
	"aioright":  "deepright",
	"aiorhight": "deepright",
	"aiohright": "deepright",
	"openai":    "openai",
}

// NormalizeTokenModelName returns the canonical name for a token model.
func NormalizeTokenModelName(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return ""
	}
	if normalized, ok := TokenModelAliases[strings.ToLower(model)]; ok {
		return normalized
	}
	return model
}

// DefaultTaskType is the fallback value when no task type is specified.
const DefaultTaskType = "cron"

// NormalizeTaskType returns a non-empty default if taskType is blank.
func NormalizeTaskType(taskType string) string {
	if strings.TrimSpace(taskType) == "" {
		return DefaultTaskType
	}
	return strings.TrimSpace(taskType)
}
