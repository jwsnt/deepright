package sharedutil

import (
	"encoding/json"
	"testing"
)

func TestToInt64Value(t *testing.T) {
	tests := []struct {
		name  string
		raw   interface{}
		want  int64
		wantOK bool
	}{
		{"int", int(42), 42, true},
		{"int8", int8(8), 8, true},
		{"int16", int16(16), 16, true},
		{"int32", int32(32), 32, true},
		{"int64", int64(64), 64, true},
		{"float32", float32(3.14), 3, true},
		{"float64", float64(3.99), 3, true},
		{"json number", json.Number("42"), 42, true},
		{"json float number", json.Number("3.14"), 3, true},
		{"string", "42", 42, true},
		{"string empty", "", 0, false},
		{"string invalid", "abc", 0, false},
		{"nil", nil, 0, false},
		{"bool", true, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ToInt64Value(tt.raw)
			if ok != tt.wantOK {
				t.Errorf("ToInt64Value() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("ToInt64Value() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestToBoolValue(t *testing.T) {
	tests := []struct {
		name  string
		raw   interface{}
		want  bool
		wantOK bool
	}{
		{"bool true", true, true, true},
		{"bool false", false, false, true},
		{"string 1", "1", true, true},
		{"string true", "true", true, true},
		{"string yes", "yes", true, true},
		{"string y", "y", true, true},
		{"string on", "on", true, true},
		{"string 0", "0", false, true},
		{"string false", "false", false, true},
		{"string no", "no", false, true},
		{"string off", "off", false, true},
		{"string invalid", "maybe", false, false},
		{"float64 nonzero", float64(1), true, true},
		{"float64 zero", float64(0), false, true},
		{"int nonzero", 1, true, true},
		{"int zero", 0, false, true},
		{"nil", nil, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ToBoolValue(tt.raw)
			if ok != tt.wantOK {
				t.Errorf("ToBoolValue() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("ToBoolValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeMetadataFields(t *testing.T) {
	tests := []struct {
		name string
		dst  map[string]interface{}
		raw  interface{}
		want map[string]interface{}
	}{
		{"nil dst", nil, map[string]interface{}{"a": 1}, nil},
		{"nil raw", map[string]interface{}{}, nil, map[string]interface{}{}},
		{"merge", map[string]interface{}{"existing": "v1"}, map[string]interface{}{"new": "v2"},
			map[string]interface{}{"existing": "v1", "new": "v2"}},
		{"override", map[string]interface{}{"key": "old"}, map[string]interface{}{"key": "new"},
			map[string]interface{}{"key": "new"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			MergeMetadataFields(tt.dst, tt.raw)
			if tt.dst == nil {
				return
			}
			for k, v := range tt.want {
				if got, ok := tt.dst[k]; !ok || got != v {
					t.Errorf("MergeMetadataFields() dst[%q] = %v, want %v", k, got, v)
				}
			}
		})
	}
}
