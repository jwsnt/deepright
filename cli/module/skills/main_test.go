package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/glebarez/go-sqlite"

	"skill-scanner/skillscore"
)

func TestGetSkillsOutputJSON(t *testing.T) {
	root := filepath.Join(".", "test-case")
	data, err := GetSkillsOutputJSON(root, 0)
	if err != nil {
		t.Fatalf("GetSkillsOutputJSON failed: %v", err)
	}

	var skills []Skill
	if err := json.Unmarshal(data, &skills); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if len(skills) != 3 {
		t.Fatalf("skills len = %d, want 3", len(skills))
	}

	byName := make(map[string]Skill, len(skills))
	for _, skill := range skills {
		byName[skill.Name] = skill
	}

	a := byName["__internal_A"]
	if a.Description != "技能A" {
		t.Fatalf("__internal_A description = %q", a.Description)
	}
	if a.Compatibility != "测试内容" {
		t.Fatalf("__internal_A compatibility = %q", a.Compatibility)
	}
	wantA, _ := filepath.Abs(filepath.Join(root, "a", "SKILL.md"))
	if a.Location != wantA {
		t.Fatalf("__internal_A location = %q, want %q", a.Location, wantA)
	}

	c := byName["__internal_c"]
	if c.Description != "技能C" {
		t.Fatalf("__internal_c description = %q", c.Description)
	}
	if got := c.Metadata["os"]; got != "darwin" {
		t.Fatalf("__internal_c metadata.os = %v", got)
	}

	f := byName["__internal_F"]
	if f.Description != "技能F" {
		t.Fatalf("__internal_F description = %q", f.Description)
	}
	if f.License != "F授权" {
		t.Fatalf("__internal_F license = %q", f.License)
	}
	if got := f.Metadata["os"]; got != "win" {
		t.Fatalf("__internal_F metadata.os = %v", got)
	}
}

func TestScanWarningsAndSyncClearsRecoveredEntry(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "skills")
	if err := os.MkdirAll(filepath.Join(root, "ok"), 0755); err != nil {
		t.Fatalf("mkdir ok: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "bad"), 0755); err != nil {
		t.Fatalf("mkdir bad: %v", err)
	}

	okSkill := "---\nname: test_ok\ndescription: ok\n---\nbody\n"
	badSkill := "---\ndescription: broken\n---\nbody\n"
	if err := os.WriteFile(filepath.Join(root, "ok", "SKILL.md"), []byte(okSkill), 0644); err != nil {
		t.Fatalf("write ok skill: %v", err)
	}
	badPath := filepath.Join(root, "bad", "SKILL.md")
	if err := os.WriteFile(badPath, []byte(badSkill), 0644); err != nil {
		t.Fatalf("write bad skill: %v", err)
	}

	dbPath := filepath.Join(tmp, "data")
	warnings, err := skillscore.ScanAndSyncWarnings(root, dbPath)
	if err != nil {
		t.Fatalf("ScanAndSyncWarnings failed: %v", err)
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings len = %d, want 1", len(warnings))
	}
	if warnings[0].Reason == "" {
		t.Fatal("warning reason should not be empty")
	}
	if warnings[0].Time <= 0 {
		t.Fatalf("warning time = %d, want > 0", warnings[0].Time)
	}

	store, err := skillscore.OpenWarningStore(dbPath)
	if err != nil {
		t.Fatalf("OpenWarningStore failed: %v", err)
	}
	list, err := store.List()
	if err != nil {
		t.Fatalf("store.List failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("store list len = %d, want 1", len(list))
	}

	fixedSkill := "---\nname: test_bad\ndescription: fixed\n---\nbody\n"
	if err := os.WriteFile(badPath, []byte(fixedSkill), 0644); err != nil {
		t.Fatalf("rewrite bad skill: %v", err)
	}
	warnings, err = skillscore.ScanAndSyncWarnings(root, dbPath)
	if err != nil {
		t.Fatalf("ScanAndSyncWarnings after fix failed: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings len after fix = %d, want 0", len(warnings))
	}

	list, err = store.List()
	if err != nil {
		t.Fatalf("store.List after fix failed: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("store list len after fix = %d, want 0", len(list))
	}
}

func TestScanCompatibilitySequenceNormalizesToString(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "skills")
	if err := os.MkdirAll(filepath.Join(root, "view-directory"), 0755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}

	content := "---\nname: view-directory\ndescription: list files\ncompatibility:\n  - macOS (Darwin)\n  - zsh shell\n---\nbody\n"
	if err := os.WriteFile(filepath.Join(root, "view-directory", "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	skills, warnings, err := skillscore.ScanWithWarnings(root)
	if err != nil {
		t.Fatalf("ScanWithWarnings failed: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings len = %d, want 0", len(warnings))
	}
	if len(skills) != 1 {
		t.Fatalf("skills len = %d, want 1", len(skills))
	}
	if got := string(skills[0].Compatibility); got != "macOS (Darwin); zsh shell" {
		t.Fatalf("compatibility = %q, want %q", got, "macOS (Darwin); zsh shell")
	}
}

func TestWarningListJSON(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "data")
	store, err := skillscore.OpenWarningStore(dbPath)
	if err != nil {
		t.Fatalf("OpenWarningStore failed: %v", err)
	}
	if err := store.Sync([]skillscore.SkillWarning{{Path: "/tmp/a/SKILL.md", Reason: "broken", Time: 123}}); err != nil {
		t.Fatalf("store.Sync failed: %v", err)
	}
	out, err := store.ListJSON()
	if err != nil {
		t.Fatalf("store.ListJSON failed: %v", err)
	}
	var warnings []skillscore.SkillWarning
	if err := json.Unmarshal(out, &warnings); err != nil {
		t.Fatalf("unmarshal warnings failed: %v", err)
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings len = %d, want 1", len(warnings))
	}
	if warnings[0].Path != "/tmp/a/SKILL.md" {
		t.Fatalf("warning path = %q, want /tmp/a/SKILL.md", warnings[0].Path)
	}
}

func TestWarningStoreUsesSQLiteDataFile(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "data")
	store, err := skillscore.OpenWarningStore(dbPath)
	if err != nil {
		t.Fatalf("OpenWarningStore failed: %v", err)
	}
	if err := store.Sync([]skillscore.SkillWarning{{Path: "/tmp/demo/SKILL.md", Reason: "yaml", Time: 456}}); err != nil {
		t.Fatalf("store.Sync failed: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM skills_warning`).Scan(&count); err != nil {
		t.Fatalf("query count failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("warning row count = %d, want 1", count)
	}
}
