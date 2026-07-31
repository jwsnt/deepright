package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestReadIntegrationBackupConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		want      integrationBackupConfig
		wantError string
	}{
		{
			name:   "valid",
			config: `{"backup":{"skills":30,"app":45}}`,
			want: integrationBackupConfig{
				SkillsInterval: 30 * time.Minute,
				AppInterval:    45 * time.Minute,
			},
		},
		{name: "missing backup", config: `{}`, wantError: "config/config.json.backup is required"},
		{name: "backup is not object", config: `{"backup":30}`, wantError: "config/config.json.backup must be an object"},
		{name: "missing skills", config: `{"backup":{"app":30}}`, wantError: "config/config.json.backup.skills"},
		{name: "zero app", config: `{"backup":{"skills":30,"app":0}}`, wantError: "config/config.json.backup.app"},
		{name: "fractional skills", config: `{"backup":{"skills":1.5,"app":30}}`, wantError: "config/config.json.backup.skills"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appDir := t.TempDir()
			useIntegrationExecutableDir(t, appDir)
			configDir := filepath.Join(appDir, "config")
			if err := os.MkdirAll(configDir, 0o755); err != nil {
				t.Fatalf("mkdir config: %v", err)
			}
			if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(tt.config), 0o644); err != nil {
				t.Fatalf("write config: %v", err)
			}

			got, err := readIntegrationBackupConfig()
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("error = %v, want %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("readIntegrationBackupConfig: %v", err)
			}
			if got != tt.want {
				t.Fatalf("config = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestBackupIntegrationAgentDirectoriesUseSeparateFilteredRepositories(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for backup test")
	}

	root := t.TempDir()
	workspace := filepath.Join(root, "agent-a")
	skillsDir := filepath.Join(workspace, "skills")
	appDir := filepath.Join(workspace, "app")
	if err := os.MkdirAll(filepath.Join(skillsDir, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir skills: %v", err)
	}
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("mkdir app: %v", err)
	}
	writeIntegrationBackupTestFile(t, filepath.Join(skillsDir, "nested", "SKILL.md"), "# demo\n")
	writeIntegrationBackupTestFile(t, filepath.Join(skillsDir, ".gitignore"), "ignored.md\n")
	writeIntegrationBackupTestFile(t, filepath.Join(skillsDir, "ignored.md"), "this remains in the backup\n")
	writeIntegrationBackupTestFile(t, filepath.Join(skillsDir, "image.PNG"), "not staged")
	if err := os.WriteFile(filepath.Join(skillsDir, "large.md"), make([]byte, integrationBackupMaximumFileSize+1), 0o644); err != nil {
		t.Fatalf("write large file: %v", err)
	}
	writeIntegrationBackupTestFile(t, filepath.Join(appDir, "index.html"), "<main>app</main>\n")

	skillsSummary, err := backupIntegrationAgentDirectory(root, "skills")
	if err != nil {
		t.Fatalf("backup skills: %v", err)
	}
	if len(skillsSummary.Results) != 1 || !skillsSummary.Results[0].Committed {
		t.Fatalf("skills result = %#v, want one committed repository", skillsSummary.Results)
	}
	if _, err := os.Stat(filepath.Join(skillsDir, ".git")); err != nil {
		t.Fatalf("skills repository missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(appDir, ".git")); !os.IsNotExist(err) {
		t.Fatalf("app repository should not be initialized by skills backup, stat err=%v", err)
	}
	skillsFiles := integrationBackupTestGit(t, skillsDir, "ls-files")
	for _, want := range []string{".gitignore", "ignored.md", "nested/SKILL.md"} {
		if !strings.Contains(skillsFiles, want+"\n") {
			t.Fatalf("skills files = %q, missing %q", skillsFiles, want)
		}
	}
	for _, unwanted := range []string{"image.PNG", "large.md"} {
		if strings.Contains(skillsFiles, unwanted) {
			t.Fatalf("skills files = %q, unexpectedly contains %q", skillsFiles, unwanted)
		}
	}
	if got := integrationBackupTestCommitCount(t, skillsDir); got != 1 {
		t.Fatalf("skills commits = %d, want 1", got)
	}

	if _, err := backupIntegrationAgentDirectory(root, "skills"); err != nil {
		t.Fatalf("backup unchanged skills: %v", err)
	}
	if got := integrationBackupTestCommitCount(t, skillsDir); got != 1 {
		t.Fatalf("skills commits after unchanged backup = %d, want 1", got)
	}

	if err := os.Rename(filepath.Join(skillsDir, "nested", "SKILL.md"), filepath.Join(skillsDir, "nested", "SKILL.exe")); err != nil {
		t.Fatalf("make tracked file binary: %v", err)
	}
	filteredSummary, err := backupIntegrationAgentDirectory(root, "skills")
	if err != nil {
		t.Fatalf("backup filtered skills: %v", err)
	}
	if len(filteredSummary.Results) != 1 || !filteredSummary.Results[0].Committed {
		t.Fatalf("filtered skills result = %#v, want a commit removing tracked file", filteredSummary.Results)
	}
	skillsFiles = integrationBackupTestGit(t, skillsDir, "ls-files")
	if strings.Contains(skillsFiles, "nested/SKILL.md") || strings.Contains(skillsFiles, "nested/SKILL.exe") {
		t.Fatalf("filtered skills files = %q, binary replacement must not be tracked", skillsFiles)
	}

	appSummary, err := backupIntegrationAgentDirectory(root, "app")
	if err != nil {
		t.Fatalf("backup app: %v", err)
	}
	if len(appSummary.Results) != 1 || !appSummary.Results[0].Committed {
		t.Fatalf("app result = %#v, want one committed repository", appSummary.Results)
	}
	if _, err := os.Stat(filepath.Join(appDir, ".git")); err != nil {
		t.Fatalf("app repository missing: %v", err)
	}
	if files := integrationBackupTestGit(t, appDir, "ls-files"); files != "index.html\n" {
		t.Fatalf("app files = %q, want only app content", files)
	}
}

func TestBackupIntegrationAgentDirectorySkipsMissingTarget(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "agent-a"), 0o755); err != nil {
		t.Fatalf("mkdir agent: %v", err)
	}
	summary, err := backupIntegrationAgentDirectory(root, "app")
	if err != nil {
		t.Fatalf("backup missing app: %v", err)
	}
	if len(summary.Results) != 0 {
		t.Fatalf("results = %#v, want missing target skipped", summary.Results)
	}
}

func TestStartIntegrationAgentBackupsRunsBothDirectoriesImmediately(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for backup test")
	}

	appRoot := t.TempDir()
	useIntegrationExecutableDir(t, appRoot)
	configDir := filepath.Join(appRoot, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"backup":{"skills":30,"app":30}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	agentRoot := filepath.Join(appRoot, "agent")
	for directory, content := range map[string]string{"skills/SKILL.md": "# skill\n", "app/index.html": "<main>app</main>\n"} {
		writeIntegrationBackupTestFile(t, filepath.Join(agentRoot, "agent-a", filepath.FromSlash(directory)), content)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startIntegrationAgentBackups(ctx, agentRoot)

	for _, repository := range []string{
		filepath.Join(agentRoot, "agent-a", "skills", ".git"),
		filepath.Join(agentRoot, "agent-a", "app", ".git"),
	} {
		deadline := time.Now().Add(2 * time.Second)
		for {
			if _, err := os.Stat(repository); err == nil {
				break
			} else if !os.IsNotExist(err) {
				t.Fatalf("stat %s: %v", repository, err)
			}
			if time.Now().After(deadline) {
				t.Fatalf("immediate backup did not initialize %s", repository)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestHandleBackupStatusReportsOnlyBackupCommits(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for backup test")
	}

	root := t.TempDir()
	skillsDir := filepath.Join(root, "agent-a", "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatalf("mkdir skills: %v", err)
	}
	if err := integrationBackupGit(skillsDir, "init", "-q"); err != nil {
		t.Fatalf("init skills repository: %v", err)
	}
	if err := integrationBackupEnsureLocalIdentity(skillsDir); err != nil {
		t.Fatalf("configure identity: %v", err)
	}
	writeIntegrationBackupTestFile(t, filepath.Join(skillsDir, "SKILL.md"), "# original\n")
	if err := integrationBackupGit(skillsDir, "add", "-f", "--", "SKILL.md"); err != nil {
		t.Fatalf("stage ordinary commit: %v", err)
	}
	if err := integrationBackupGit(skillsDir, "-c", "commit.gpgSign=false", "commit", "--no-verify", "-m", "ordinary commit"); err != nil {
		t.Fatalf("commit ordinary change: %v", err)
	}

	cfg := &Config{AgentDir: root}
	request := httptest.NewRequest(http.MethodGet, "/api/backup/status?agentId=agent-a&directory=skills", nil)
	response := httptest.NewRecorder()
	handleBackupStatus(cfg).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status before backup = %d, body = %s", response.Code, response.Body.String())
	}
	var status integrationBackupStatusResponse
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatalf("decode status before backup: %v", err)
	}
	if status.LastBackupAt != 0 {
		t.Fatalf("ordinary commit timestamp = %d, want no backup timestamp", status.LastBackupAt)
	}

	writeIntegrationBackupTestFile(t, filepath.Join(skillsDir, "SKILL.md"), "# backed up\n")
	if _, err := backupIntegrationAgentDirectory(root, "skills"); err != nil {
		t.Fatalf("create backup commit: %v", err)
	}
	response = httptest.NewRecorder()
	handleBackupStatus(cfg).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status after backup = %d, body = %s", response.Code, response.Body.String())
	}
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatalf("decode status after backup: %v", err)
	}
	if status.Directory != "skills" || status.LastBackupAt <= 0 {
		t.Fatalf("backup status = %#v, want skills timestamp", status)
	}

	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/backup/status?agentId=agent-a&directory=tmp", nil),
		httptest.NewRequest(http.MethodPost, "/api/backup/status?agentId=agent-a&directory=skills", nil),
	} {
		response = httptest.NewRecorder()
		handleBackupStatus(cfg).ServeHTTP(response, request)
		if response.Code < http.StatusBadRequest {
			t.Fatalf("request %s %s status = %d, want client error", request.Method, request.URL.String(), response.Code)
		}
	}
}

func writeIntegrationBackupTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func integrationBackupTestGit(t *testing.T, repository string, args ...string) string {
	t.Helper()
	commandArgs := append([]string{"-C", repository}, args...)
	output, err := exec.Command("git", commandArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func integrationBackupTestCommitCount(t *testing.T, repository string) int {
	t.Helper()
	value := strings.TrimSpace(integrationBackupTestGit(t, repository, "rev-list", "--count", "HEAD"))
	count, err := strconv.Atoi(value)
	if err != nil {
		t.Fatalf("parse commit count %q: %v", value, err)
	}
	return count
}
