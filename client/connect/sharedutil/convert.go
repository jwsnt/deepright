package sharedutil

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ToInt64Value attempts to convert an interface{} to int64.
func ToInt64Value(raw interface{}) (int64, bool) {
	switch v := raw.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case float32:
		return int64(v), true
	case float64:
		return int64(v), true
	case json.Number:
		n, err := v.Int64()
		if err == nil {
			return n, true
		}
		f, err := v.Float64()
		if err != nil {
			return 0, false
		}
		return int64(f), true
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return 0, false
		}
		var n int64
		if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

// ToBoolValue attempts to convert an interface{} to bool.
func ToBoolValue(raw interface{}) (bool, bool) {
	switch v := raw.(type) {
	case bool:
		return v, true
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "y", "on":
			return true, true
		case "0", "false", "no", "n", "off":
			return false, true
		default:
			return false, false
		}
	case float64:
		return v != 0, true
	case float32:
		return v != 0, true
	case int:
		return v != 0, true
	case int64:
		return v != 0, true
	default:
		return false, false
	}
}

// MergeMetadataFields merges raw metadata into dst.
func MergeMetadataFields(dst map[string]interface{}, raw interface{}) {
	if dst == nil || raw == nil {
		return
	}
	rawBytes, err := json.Marshal(raw)
	if err != nil {
		return
	}
	var existing map[string]interface{}
	if err := json.Unmarshal(rawBytes, &existing); err != nil {
		return
	}
	for k, v := range existing {
		dst[k] = v
	}
}
