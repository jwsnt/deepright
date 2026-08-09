package agentcopy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyCopiesManagedEntriesAndKnowledge(t *testing.T) {
	root := t.TempDir()
	agentRoot := filepath.Join(root, "agent")
	knowledgeRoot := filepath.Join(root, "knowledge")
	sourceAgentID := "SourceAgent"
	targetAgentID := "TargetAgent"

	writeAgentCopyTestFile(t, filepath.Join(agentRoot, sourceAgentID, "app", "index.html"), "source app")
	writeAgentCopyTestFile(t, filepath.Join(agentRoot, sourceAgentID, "data", "cache.json"), "source data")
	writeAgentCopyTestFile(t, filepath.Join(agentRoot, sourceAgentID, "skills", "tool", "SKILL.md"), "source skill")
	writeAgentCopyTestFile(t, filepath.Join(agentRoot, sourceAgentID, "SOUL.md"), "source soul")
	writeAgentCopyTestFile(t, filepath.Join(agentRoot, sourceAgentID, "USER.md"), "source user")
	writeAgentCopyTestFile(t, filepath.Join(agentRoot, sourceAgentID, "Knowledge.md"), "source uppercase knowledge")
	writeAgentCopyTestFile(t, filepath.Join(agentRoot, sourceAgentID, "config.json"), `{"source":true}`)
	writeAgentCopyTestFile(t, filepath.Join(agentRoot, sourceAgentID, "workspace", "notes.md"), "keep source workspace untouched")
	writeAgentCopyTestFile(t, filepath.Join(knowledgeRoot, sourceAgentID, "wiki", "index.md"), "source knowledge")

	writeAgentCopyTestFile(t, filepath.Join(agentRoot, targetAgentID, "app", "old.txt"), "old app")
	writeAgentCopyTestFile(t, filepath.Join(agentRoot, targetAgentID, "skills", "old.txt"), "old skill")
	writeAgentCopyTestFile(t, filepath.Join(agentRoot, targetAgentID, "SOUL.md"), "old soul")
	writeAgentCopyTestFile(t, filepath.Join(agentRoot, targetAgentID, "knowledge.md"), "old lowercase knowledge")
	writeAgentCopyTestFile(t, filepath.Join(agentRoot, targetAgentID, "config.json"), `{"keep":true}`)
	writeAgentCopyTestFile(t, filepath.Join(agentRoot, targetAgentID, "workspace", "keep.md"), "keep target workspace")
	writeAgentCopyTestFile(t, filepath.Join(knowledgeRoot, targetAgentID, "stale.md"), "stale knowledge")

	result, err := Copy(agentRoot, knowledgeRoot, sourceAgentID, targetAgentID)
	if err != nil {
		t.Fatalf("Copy() error = %v", err)
	}

	if result.SourceAgentID != sourceAgentID {
		t.Fatalf("result.SourceAgentID = %q, want %q", result.SourceAgentID, sourceAgentID)
	}
	if result.TargetAgentID != targetAgentID {
		t.Fatalf("result.TargetAgentID = %q, want %q", result.TargetAgentID, targetAgentID)
	}
	if result.AgentPath != filepath.Join(agentRoot, targetAgentID) {
		t.Fatalf("result.AgentPath = %q", result.AgentPath)
	}

	assertAgentCopyTestFile(t, filepath.Join(agentRoot, targetAgentID, "app", "index.html"), "source app")
	assertAgentCopyTestFile(t, filepath.Join(agentRoot, targetAgentID, "data", "cache.json"), "source data")
	assertAgentCopyTestFile(t, filepath.Join(agentRoot, targetAgentID, "skills", "tool", "SKILL.md"), "source skill")
	assertAgentCopyTestFile(t, filepath.Join(agentRoot, targetAgentID, "SOUL.md"), "source soul")
	assertAgentCopyTestFile(t, filepath.Join(agentRoot, targetAgentID, "USER.md"), "source user")
	assertAgentCopyTestFile(t, filepath.Join(agentRoot, targetAgentID, "Knowledge.md"), "source uppercase knowledge")
	assertAgentCopyTestFile(t, filepath.Join(agentRoot, targetAgentID, "config.json"), `{"keep":true}`)
	assertAgentCopyTestFile(t, filepath.Join(agentRoot, targetAgentID, "workspace", "keep.md"), "keep target workspace")
	assertAgentCopyTestFile(t, filepath.Join(knowledgeRoot, targetAgentID, "wiki", "index.md"), "source knowledge")

	assertAgentCopyTestMissing(t, filepath.Join(agentRoot, targetAgentID, "app", "old.txt"))
	assertAgentCopyTestMissing(t, filepath.Join(agentRoot, targetAgentID, "skills", "old.txt"))
	assertAgentCopyTestMissing(t, filepath.Join(knowledgeRoot, targetAgentID, "stale.md"))
}

func TestCopyRemovesTargetManagedEntriesWhenSourceMissing(t *testing.T) {
	root := t.TempDir()
	agentRoot := filepath.Join(root, "agent")
	knowledgeRoot := filepath.Join(root, "knowledge")

	writeAgentCopyTestFile(t, filepath.Join(agentRoot, "SourceAgent", "config.json"), `{"source":true}`)
	writeAgentCopyTestFile(t, filepath.Join(agentRoot, "TargetAgent", "app", "old.txt"), "old app")
	writeAgentCopyTestFile(t, filepath.Join(agentRoot, "TargetAgent", "USER.md"), "old user")
	writeAgentCopyTestFile(t, filepath.Join(agentRoot, "TargetAgent", "Knowledge.md"), "old uppercase knowledge")
	writeAgentCopyTestFile(t, filepath.Join(agentRoot, "TargetAgent", "config.json"), `{"keep":true}`)
	writeAgentCopyTestFile(t, filepath.Join(knowledgeRoot, "TargetAgent", "stale.md"), "stale knowledge")

	if _, err := Copy(agentRoot, knowledgeRoot, "SourceAgent", "TargetAgent"); err != nil {
		t.Fatalf("Copy() error = %v", err)
	}

	assertAgentCopyTestMissing(t, filepath.Join(agentRoot, "TargetAgent", "app"))
	assertAgentCopyTestMissing(t, filepath.Join(agentRoot, "TargetAgent", "USER.md"))
	assertAgentCopyTestMissing(t, filepath.Join(agentRoot, "TargetAgent", "Knowledge.md"))
	assertAgentCopyTestMissing(t, filepath.Join(knowledgeRoot, "TargetAgent"))
	assertAgentCopyTestFile(t, filepath.Join(agentRoot, "TargetAgent", "config.json"), `{"keep":true}`)
}

func writeAgentCopyTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func assertAgentCopyTestFile(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if string(data) != want {
		t.Fatalf("file %q = %q, want %q", path, string(data), want)
	}
}

func assertAgentCopyTestMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %q to be missing, stat err = %v", path, err)
	}
}
