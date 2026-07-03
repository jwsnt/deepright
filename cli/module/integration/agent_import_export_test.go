package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleAgentExportFiltersManagedDirectories(t *testing.T) {
	agentRoot := t.TempDir()
	agentID := "ExportAgent"
	writeTestFile(t, filepath.Join(agentRoot, agentID, "config.json"), `{"provider":"openai"}`)
	writeTestFile(t, filepath.Join(agentRoot, agentID, "notes", "README.md"), "keep")
	writeTestFile(t, filepath.Join(agentRoot, agentID, "chrome_9222", "state.json"), "drop")
	writeTestFile(t, filepath.Join(agentRoot, agentID, "data", "cache.db"), "drop")
	writeTestFile(t, filepath.Join(agentRoot, agentID, "tmp", "draft.txt"), "drop")

	req := httptest.NewRequest(http.MethodGet, "/api/agent/export?agent_id="+agentID, nil)
	rec := httptest.NewRecorder()
	handleAgentExport(&Config{AgentDir: agentRoot})(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, rec.Body.String())
	}

	reader, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("zip.NewReader() error = %v", err)
	}

	names := make([]string, 0, len(reader.File))
	for _, file := range reader.File {
		names = append(names, file.Name)
	}

	for _, expected := range []string{
		agentID + "/config.json",
		agentID + "/notes/README.md",
	} {
		if !containsString(names, expected) {
			t.Fatalf("missing export entry %q in %v", expected, names)
		}
	}
	for _, banned := range []string{
		agentID + "/chrome_9222/state.json",
		agentID + "/data/cache.db",
		agentID + "/tmp/draft.txt",
	} {
		if containsString(names, banned) {
			t.Fatalf("unexpected export entry %q in %v", banned, names)
		}
	}
}

func TestHandleAgentImportRejectsExistingAgent(t *testing.T) {
	agentRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(agentRoot, "ImportedAgent"), 0o755); err != nil {
		t.Fatalf("mkdir existing agent: %v", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("files", "ImportedAgent.zip")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	zipWriter := zip.NewWriter(part)
	fileWriter, err := zipWriter.Create("ImportedAgent/config.json")
	if err != nil {
		t.Fatalf("Create zip entry error = %v", err)
	}
	if _, err := fileWriter.Write([]byte(`{"provider":"openai"}`)); err != nil {
		t.Fatalf("Write zip entry error = %v", err)
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("Close zip writer error = %v", err)
	}
	if err := writer.WriteField("sourceKind", "zip"); err != nil {
		t.Fatalf("WriteField() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close multipart writer error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/agent/import", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	handleAgentImport(&Config{AgentDir: agentRoot})(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v, body = %s", err, rec.Body.String())
	}
	if payload["status"] != float64(1) {
		t.Fatalf("payload.status = %v, want 1", payload["status"])
	}
	if !strings.Contains(payload["content"].(string), "please delete the existing agent first") {
		t.Fatalf("payload.content = %q", payload["content"])
	}
}

func TestHandleAgentCopyCopiesManagedEntries(t *testing.T) {
	root := t.TempDir()
	agentRoot := filepath.Join(root, "agent")
	knowledgeRoot := filepath.Join(root, "knowledge")

	writeTestFile(t, filepath.Join(agentRoot, "SourceAgent", "app", "index.html"), "source app")
	writeTestFile(t, filepath.Join(agentRoot, "SourceAgent", "data", "cache.json"), "source data")
	writeTestFile(t, filepath.Join(agentRoot, "SourceAgent", "skills", "skill", "SKILL.md"), "source skill")
	writeTestFile(t, filepath.Join(agentRoot, "SourceAgent", "SOUL.md"), "source soul")
	writeTestFile(t, filepath.Join(agentRoot, "SourceAgent", "USER.md"), "source user")
	writeTestFile(t, filepath.Join(agentRoot, "SourceAgent", "Knowledge.md"), "source uppercase knowledge")
	writeTestFile(t, filepath.Join(agentRoot, "SourceAgent", "config.json"), `{"source":true}`)
	writeTestFile(t, filepath.Join(knowledgeRoot, "SourceAgent", "index.md"), "source knowledge")

	writeTestFile(t, filepath.Join(agentRoot, "TargetAgent", "app", "old.txt"), "old app")
	writeTestFile(t, filepath.Join(agentRoot, "TargetAgent", "knowledge.md"), "stale lowercase knowledge")
	writeTestFile(t, filepath.Join(agentRoot, "TargetAgent", "config.json"), `{"keep":true}`)
	writeTestFile(t, filepath.Join(knowledgeRoot, "TargetAgent", "stale.md"), "stale knowledge")

	req := httptest.NewRequest(http.MethodGet, "/api/copy?source_agentId=SourceAgent&target_agentId=TargetAgent", nil)
	rec := httptest.NewRecorder()
	handleAgentCopy(&Config{AgentDir: agentRoot})(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v, body = %s", err, rec.Body.String())
	}
	if payload["status"] != float64(0) {
		t.Fatalf("payload.status = %v, want 0", payload["status"])
	}
	if payload["sourceAgentId"] != "SourceAgent" {
		t.Fatalf("payload.sourceAgentId = %v", payload["sourceAgentId"])
	}
	if payload["targetAgentId"] != "TargetAgent" {
		t.Fatalf("payload.targetAgentId = %v", payload["targetAgentId"])
	}

	assertIntegrationTestFileContent(t, filepath.Join(agentRoot, "TargetAgent", "app", "index.html"), "source app")
	assertIntegrationTestFileContent(t, filepath.Join(agentRoot, "TargetAgent", "data", "cache.json"), "source data")
	assertIntegrationTestFileContent(t, filepath.Join(agentRoot, "TargetAgent", "skills", "skill", "SKILL.md"), "source skill")
	assertIntegrationTestFileContent(t, filepath.Join(agentRoot, "TargetAgent", "SOUL.md"), "source soul")
	assertIntegrationTestFileContent(t, filepath.Join(agentRoot, "TargetAgent", "USER.md"), "source user")
	assertIntegrationTestFileContent(t, filepath.Join(agentRoot, "TargetAgent", "Knowledge.md"), "source uppercase knowledge")
	assertIntegrationTestFileContent(t, filepath.Join(agentRoot, "TargetAgent", "config.json"), `{"keep":true}`)
	assertIntegrationTestFileContent(t, filepath.Join(knowledgeRoot, "TargetAgent", "index.md"), "source knowledge")
	assertIntegrationTestMissing(t, filepath.Join(agentRoot, "TargetAgent", "app", "old.txt"))
	assertIntegrationTestMissing(t, filepath.Join(knowledgeRoot, "TargetAgent", "stale.md"))
}

func TestRunIntegrationAgentCLIExportAndImport(t *testing.T) {
	exportRoot := t.TempDir()
	writeTestFile(t, filepath.Join(exportRoot, "CliAgent", "config.json"), `{"provider":"openai"}`)

	zipPath := filepath.Join(t.TempDir(), "CliAgent.zip")
	var exportStdout bytes.Buffer
	var exportStderr bytes.Buffer
	code := runIntegrationAgentCLI([]string{"export", "--agent", "CliAgent", "--output", zipPath, "--agent-dir", exportRoot}, &exportStdout, &exportStderr)
	if code != 0 {
		t.Fatalf("export code = %d, stderr = %s", code, exportStderr.String())
	}
	if _, err := os.Stat(zipPath); err != nil {
		t.Fatalf("exported zip missing: %v", err)
	}

	importRoot := filepath.Join(t.TempDir(), "imported-agents")
	var importStdout bytes.Buffer
	var importStderr bytes.Buffer
	code = runIntegrationAgentCLI([]string{"import", "--input", zipPath, "--agent-dir", importRoot}, &importStdout, &importStderr)
	if code != 0 {
		t.Fatalf("import code = %d, stderr = %s", code, importStderr.String())
	}
	if _, err := os.Stat(filepath.Join(importRoot, "CliAgent", "config.json")); err != nil {
		t.Fatalf("imported config missing: %v", err)
	}
	if !strings.Contains(importStdout.String(), `"agentId": "CliAgent"`) {
		t.Fatalf("import stdout = %s", importStdout.String())
	}
}

func TestRunIntegrationAgentCLICopy(t *testing.T) {
	root := t.TempDir()
	agentRoot := filepath.Join(root, "agent")
	knowledgeRoot := filepath.Join(root, "knowledge")

	writeTestFile(t, filepath.Join(agentRoot, "SourceAgent", "skills", "skill", "SKILL.md"), "source skill")
	writeTestFile(t, filepath.Join(agentRoot, "SourceAgent", "SOUL.md"), "source soul")
	writeTestFile(t, filepath.Join(agentRoot, "SourceAgent", "knowledge.md"), "source lowercase knowledge")
	writeTestFile(t, filepath.Join(agentRoot, "TargetAgent", "config.json"), `{"keep":true}`)
	writeTestFile(t, filepath.Join(knowledgeRoot, "SourceAgent", "index.md"), "source knowledge")
	writeTestFile(t, filepath.Join(knowledgeRoot, "TargetAgent", "stale.md"), "stale knowledge")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runIntegrationAgentCLI([]string{"copy", "--source", "SourceAgent", "--target", "TargetAgent", "--agent-dir", agentRoot}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("copy code = %d, stderr = %s", code, stderr.String())
	}
	assertIntegrationTestFileContent(t, filepath.Join(agentRoot, "TargetAgent", "skills", "skill", "SKILL.md"), "source skill")
	assertIntegrationTestFileContent(t, filepath.Join(agentRoot, "TargetAgent", "SOUL.md"), "source soul")
	assertIntegrationTestFileContent(t, filepath.Join(agentRoot, "TargetAgent", "knowledge.md"), "source lowercase knowledge")
	assertIntegrationTestFileContent(t, filepath.Join(agentRoot, "TargetAgent", "config.json"), `{"keep":true}`)
	assertIntegrationTestFileContent(t, filepath.Join(knowledgeRoot, "TargetAgent", "index.md"), "source knowledge")
	assertIntegrationTestMissing(t, filepath.Join(knowledgeRoot, "TargetAgent", "stale.md"))
	if !strings.Contains(stdout.String(), `"sourceAgentId": "SourceAgent"`) {
		t.Fatalf("copy stdout = %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"targetAgentId": "TargetAgent"`) {
		t.Fatalf("copy stdout = %s", stdout.String())
	}
}

func TestIntegrationCLIHelpIncludesAgentCommand(t *testing.T) {
	stdout := captureIntegrationStdout(t, func() {
		printCLIHelp()
	})
	if !strings.Contains(stdout, "integration agent export --agent ID [--output PATH] [--agent-dir DIR]") {
		t.Fatalf("missing agent export usage: %s", stdout)
	}
	if !strings.Contains(stdout, "integration agent copy --source ID --target ID [--agent-dir DIR]") {
		t.Fatalf("missing agent copy usage: %s", stdout)
	}
	if !strings.Contains(stdout, "agent             Export, import, or copy managed Agent files") {
		t.Fatalf("missing agent command description: %s", stdout)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func assertIntegrationTestFileContent(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if string(data) != want {
		t.Fatalf("file %q = %q, want %q", path, string(data), want)
	}
}

func assertIntegrationTestMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %q to be missing, stat err = %v", path, err)
	}
}
