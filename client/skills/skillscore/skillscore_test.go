package skillscore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFrontMatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantOK  bool
		want    string
	}{
		{"no front matter", "plain content", false, ""},
		{"empty front matter (---)", "---\n---\ncontent", false, ""},
		{"empty front matter (-----)", "-----\n-----\ncontent", false, ""},
		{"with metadata", "---\nname: test\n---\ncontent", true, "name: test"},
		{"with long dashes", "-----\nname: test\n-----\ncontent", true, "name: test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseFrontMatter(tt.content)
			if ok != tt.wantOK {
				t.Errorf("parseFrontMatter() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("parseFrontMatter() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateSkill(t *testing.T) {
	tests := []struct {
		name    string
		skill   *Skill
		wantErr bool
	}{
		{"valid", &Skill{Name: "test_skill_1", Description: "A valid skill"}, false},
		{"empty name", &Skill{Name: "", Description: "desc"}, true},
		{"name too long", &Skill{Name: string(make([]byte, 65)), Description: "desc"}, true},
		{"name invalid chars", &Skill{Name: "test skill!", Description: "desc"}, true},
		{"empty description", &Skill{Name: "test", Description: ""}, true},
		{"description too long", &Skill{Name: "test", Description: string(make([]byte, 1025))}, true},
		{"compatibility too long", &Skill{Name: "test", Description: "desc", Compatibility: compatibilityValue(string(make([]byte, 501)))}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSkill(tt.skill)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSkill() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseSkill(t *testing.T) {
	tmpDir := t.TempDir()
	skillFile := filepath.Join(tmpDir, "SKILL.md")

	content := `---
name: my-skill
description: A test skill
---
# My Skill
Some content here`

	if err := os.WriteFile(skillFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	skill, err := parseSkill(skillFile)
	if err != nil {
		t.Fatalf("parseSkill() error: %v", err)
	}
	if skill.Name != "my-skill" {
		t.Errorf("Name = %q, want %q", skill.Name, "my-skill")
	}
	if skill.Description != "A test skill" {
		t.Errorf("Description = %q, want %q", skill.Description, "A test skill")
	}
	if skill.Location == "" {
		t.Error("Location should not be empty")
	}
}

func TestParseSkillNoFrontMatter(t *testing.T) {
	tmpDir := t.TempDir()
	skillFile := filepath.Join(tmpDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte("no front matter"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := parseSkill(skillFile)
	if err == nil {
		t.Error("parseSkill() expected error for missing front matter")
	}
}

func TestParseSkillInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	skillFile := filepath.Join(tmpDir, "SKILL.md")
	content := "---\ninvalid: [unclosed\n---\ncontent"
	if err := os.WriteFile(skillFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := parseSkill(skillFile)
	if err == nil {
		t.Error("parseSkill() expected error for invalid YAML")
	}
}

func TestFlushCache(t *testing.T) {
	FlushCache()
}

func TestScanNonExistentDir(t *testing.T) {
	skills, err := Scan("/nonexistent/path")
	if err == nil {
		t.Log("Scan() returned no error for non-existent dir")
	}
	_ = skills
}

func TestScanEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	skills, err := Scan(tmpDir)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("Scan() returned %d skills, want 0", len(skills))
	}
}

func TestScanWarningsNonExistentDir(t *testing.T) {
	warnings, err := ScanWarnings("/nonexistent")
	if err != nil {
		t.Logf("ScanWarnings() error: %v (expected)", err)
	}
	_ = warnings
}

func TestGetOutputJSON(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := GetOutputJSON(tmpDir, 0)
	if err != nil {
		t.Logf("GetOutputJSON() error: %v (may be expected)", err)
	}
}

func TestScanAgentSkills(t *testing.T) {
	tmpDir := t.TempDir()
	skills, err := ScanAgentSkills(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if skills == nil {
		t.Error("ScanAgentSkills() returned nil, want empty slice")
	}
}

func TestScanWithWarningsEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	skills, warnings, err := ScanWithWarnings(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 0 {
		t.Errorf("ScanWithWarnings() returned %d skills, want 0", len(skills))
	}
	// warnings may be nil or empty
	if warnings == nil {
		t.Log("ScanWithWarnings() warnings is nil for empty dir")
	}
}

func TestNameRegex(t *testing.T) {
	tests := []struct {
		name  string
		input string
		match bool
	}{
		{"valid underscore", "test_skill", true},
		{"valid alphanumeric", "skill1", true},
		{"with dash in middle", "my-skill", true},
		{"with dash at end", "skill-", false},
		{"starts with number", "1skill", true},
		{"with space", "my skill", false},
		{"too short empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nameRe.MatchString(tt.input)
			if got != tt.match {
				t.Errorf("nameRe.MatchString(%q) = %v, want %v", tt.input, got, tt.match)
			}
		})
	}
}
