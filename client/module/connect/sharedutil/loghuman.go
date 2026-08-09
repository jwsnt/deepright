package sharedutil

import (
	"strconv"
	"strings"
)

// HumanLogLine formats a user-facing log line in plain language.
func HumanLogLine(at string, params []string, action string) string {
	parts := make([]string, 0, 3)
	at = strings.TrimSpace(at)
	if at != "" {
		parts = append(parts, at)
	}
	filtered := make([]string, 0, len(params))
	for _, item := range params {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		filtered = append(filtered, item)
	}
	if len(filtered) > 0 {
		parts = append(parts, "用参数："+strings.Join(filtered, "；"))
	}
	action = strings.TrimSpace(action)
	if action != "" {
		parts = append(parts, "做了："+action)
	}
	return strings.Join(parts, "，")
}

// HumanParam returns a labeled parameter when value is not empty.
func HumanParam(label, value string) string {
	label = strings.TrimSpace(label)
	value = SummarizeLogText(value, 120)
	if label == "" || value == "" {
		return ""
	}
	return label + "=" + value
}

// SummarizeLogText trims and shortens free-form text for human-readable logs.
func SummarizeLogText(value string, max int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.Join(strings.Fields(value), " ")
	if max > 0 && len([]rune(value)) > max {
		runes := []rune(value)
		return strings.TrimSpace(string(runes[:max])) + "..."
	}
	return value
}

// ParseStructuredLogFields parses key=value style log content.
func ParseStructuredLogFields(text string) map[string]string {
	fields := make(map[string]string)
	for _, token := range splitStructuredLogTokens(text) {
		idx := strings.IndexByte(token, '=')
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(token[:idx])
		value := strings.TrimSpace(token[idx+1:])
		if key == "" {
			continue
		}
		if unquoted, err := strconv.Unquote(value); err == nil {
			value = unquoted
		}
		fields[key] = strings.TrimSpace(value)
	}
	return fields
}

func splitStructuredLogTokens(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	tokens := make([]string, 0, 8)
	var current strings.Builder
	var quote byte
	escaped := false
	flush := func() {
		if current.Len() == 0 {
			return
		}
		tokens = append(tokens, current.String())
		current.Reset()
	}
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if escaped {
			current.WriteByte(ch)
			escaped = false
			continue
		}
		if quote != 0 {
			current.WriteByte(ch)
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		switch ch {
		case '"', '\'':
			quote = ch
			current.WriteByte(ch)
		case ' ', '\t', '\n', '\r':
			flush()
		default:
			current.WriteByte(ch)
		}
	}
	flush()
	return tokens
}
