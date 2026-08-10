package agentarchive

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportSkipsManagedTopLevelDirectories(t *testing.T) {
	agentRoot := t.TempDir()
	agentID := "DemoAgent"
	agentDir := filepath.Join(agentRoot, agentID)
	mustWriteFile(t, filepath.Join(agentDir, "config.json"), `{"hello":"world"}`)
	mustWriteFile(t, filepath.Join(agentDir, "notes", "README.md"), "keep")
	mustWriteFile(t, filepath.Join(agentDir, "data.txt"), "keep top-level file")
	mustWriteFile(t, filepath.Join(agentDir, "chrome_9222", "state.json"), "drop")
	mustWriteFile(t, filepath.Join(agentDir, "data", "cache.db"), "drop")
	mustWriteFile(t, filepath.Join(agentDir, "tmp", "draft.txt"), "drop")

	var buf bytes.Buffer
	if err := Export(agentRoot, agentID, &buf); err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("zip.NewReader() error = %v", err)
	}

	names := make([]string, 0, len(reader.File))
	for _, file := range reader.File {
		names = append(names, file.Name)
	}

	if !containsZipEntry(names, agentID+"/config.json") {
		t.Fatalf("config.json missing from export: %v", names)
	}
	if !containsZipEntry(names, agentID+"/notes/README.md") {
		t.Fatalf("nested file missing from export: %v", names)
	}
	if !containsZipEntry(names, agentID+"/data.txt") {
		t.Fatalf("top-level file missing from export: %v", names)
	}
	for _, banned := range []string{
		agentID + "/chrome_9222/state.json",
		agentID + "/data/cache.db",
		agentID + "/tmp/draft.txt",
	} {
		if containsZipEntry(names, banned) {
			t.Fatalf("unexpected exported entry %q in %v", banned, names)
		}
	}
}

func TestImportZipFileCreatesAgentDirectory(t *testing.T) {
	agentRoot := t.TempDir()
	zipPath := filepath.Join(t.TempDir(), "agent.zip")
	createZipFile(t, zipPath, map[string]string{
		"ImportedAgent/":                "",
		"ImportedAgent/config.json":     `{"provider":"openai"}`,
		"ImportedAgent/skills/SKILL.md": "# skill",
	})

	result, err := ImportZipFile(agentRoot, zipPath)
	if err != nil {
		t.Fatalf("ImportZipFile() error = %v", err)
	}

	if result.AgentID != "ImportedAgent" {
		t.Fatalf("result.AgentID = %q, want ImportedAgent", result.AgentID)
	}
	if result.Source != "zip" {
		t.Fatalf("result.Source = %q, want zip", result.Source)
	}
	if _, err := os.Stat(filepath.Join(agentRoot, "ImportedAgent", "config.json")); err != nil {
		t.Fatalf("expected imported config.json: %v", err)
	}
	if _, err := os.Stat(filepath.Join(agentRoot, "ImportedAgent", "skills", "SKILL.md")); err != nil {
		t.Fatalf("expected imported SKILL.md: %v", err)
	}
}

func TestImportDirectoryRejectsExistingAgent(t *testing.T) {
	agentRoot := t.TempDir()
	existing := filepath.Join(agentRoot, "ExistingAgent")
	if err := os.MkdirAll(existing, 0o755); err != nil {
		t.Fatalf("mkdir existing agent: %v", err)
	}

	sourceRoot := t.TempDir()
	sourceDir := filepath.Join(sourceRoot, "ExistingAgent")
	mustWriteFile(t, filepath.Join(sourceDir, "config.json"), `{"hello":"world"}`)

	_, err := ImportDirectory(agentRoot, sourceDir)
	if err == nil {
		t.Fatalf("ImportDirectory() error = nil, want conflict")
	}
	if !strings.Contains(err.Error(), "please delete the existing agent first") {
		t.Fatalf("ImportDirectory() error = %v", err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if strings.HasSuffix(path, "/") {
		return
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func createZipFile(t *testing.T, path string, files map[string]string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create(%q) error = %v", path, err)
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	for name, content := range files {
		writer, err := zw.Create(name)
		if err != nil {
			t.Fatalf("Create(%q) zip entry error = %v", name, err)
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			t.Fatalf("Write(%q) zip entry error = %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("Close zip writer error = %v", err)
	}
}

func containsZipEntry(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}
