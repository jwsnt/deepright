package sharedschemasvc

import (
	"encoding/json"
	"testing"
)

func TestResponseSchemaJSON(t *testing.T) {
	schema := ResponseSchemaJSON()
	if schema == "" {
		t.Fatal("ResponseSchemaJSON() returned empty string")
	}

	// Verify it's valid JSON
	var parsed map[string]any
	if err := json.Unmarshal([]byte(schema), &parsed); err != nil {
		t.Fatalf("ResponseSchemaJSON() returned invalid JSON: %v", err)
	}

	// Verify required fields exist
	props, ok := parsed["properties"].(map[string]any)
	if !ok {
		t.Fatal("ResponseSchemaJSON() missing 'properties' field")
	}

	for _, field := range []string{"content", "artifacts", "why_do_this"} {
		if _, exists := props[field]; !exists {
			t.Errorf("ResponseSchemaJSON() missing property %q", field)
		}
	}
}

func TestResponseSchemaDefinition(t *testing.T) {
	schema := ResponseSchemaDefinition()
	if schema == nil {
		t.Fatal("ResponseSchemaDefinition() returned nil")
	}

	if _, ok := schema["type"]; !ok {
		t.Error("ResponseSchemaDefinition() missing 'type' field")
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("ResponseSchemaDefinition() missing 'properties' field")
	}

	if _, exists := props["content"]; !exists {
		t.Error("ResponseSchemaDefinition() missing property 'content'")
	}

	required, ok := schema["required"].([]any)
	if !ok {
		t.Fatalf("ResponseSchemaDefinition() required type = %T", schema["required"])
	}
	if len(required) != 1 || required[0] != "content" {
		t.Fatalf("ResponseSchemaDefinition() required = %#v", required)
	}
}

func TestResponseSchemaDefinitionReturnsCopy(t *testing.T) {
	// Verify that multiple calls return the same structure (copy)
	s1 := ResponseSchemaDefinition()
	s2 := ResponseSchemaDefinition()

	if s1 == nil || s2 == nil {
		t.Fatal("ResponseSchemaDefinition() returned nil")
	}

	s1["extra"] = "modified"
	if _, exists := s2["extra"]; exists {
		t.Error("ResponseSchemaDefinition() returned the same map reference, not a copy")
	}
}
