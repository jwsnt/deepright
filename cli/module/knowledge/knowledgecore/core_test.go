package knowledgecore

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"runtimepaths"
	"testing"
)

func openTestDB(t *testing.T, appDir string) *sql.DB {
	t.Helper()
	db, err := OpenSharedDB(appDir)
	if err != nil {
		t.Fatalf("OpenSharedDB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = CloseSharedDB()
	})
	return db
}

func TestEnsureRuntimeCreatesKnowledgeDirAndDBState(t *testing.T) {
	appDir := t.TempDir()

	state, err := EnsureRuntime(appDir)
	if err != nil {
		t.Fatalf("EnsureRuntime() error = %v", err)
	}
	t.Cleanup(func() {
		_ = CloseSharedDB()
	})

	wantKnowledge := filepath.Join(appDir, defaultKnowledgeDirName)
	if state.Path != wantKnowledge {
		t.Fatalf("Path = %q, want %q", state.Path, wantKnowledge)
	}
	if state.LastUpdate != 0 {
		t.Fatalf("LastUpdate = %d, want 0", state.LastUpdate)
	}
	if state.KnowledgeCommit {
		t.Fatal("KnowledgeCommit = true, want false")
	}
	if _, err := filepath.Abs(filepath.Join(appDir, defaultSQLiteFileName)); err != nil {
		t.Fatalf("resolve db path: %v", err)
	}
}

func TestEnsureRuntimeForAgentCreatesAgentKnowledgeDirAndDBState(t *testing.T) {
	appDir := t.TempDir()

	state, err := EnsureRuntimeForAgent(appDir, "agent-a")
	if err != nil {
		t.Fatalf("EnsureRuntimeForAgent() error = %v", err)
	}
	t.Cleanup(func() {
		_ = CloseSharedDB()
	})

	wantKnowledge := filepath.Join(appDir, defaultKnowledgeDirName, "agent-a")
	if state.Path != wantKnowledge {
		t.Fatalf("Path = %q, want %q", state.Path, wantKnowledge)
	}
	if state.LastUpdate != 0 {
		t.Fatalf("LastUpdate = %d, want 0", state.LastUpdate)
	}
	if state.KnowledgeCommit {
		t.Fatal("KnowledgeCommit = true, want false")
	}
}

func TestSetLastUpdatePersistsValue(t *testing.T) {
	appDir := t.TempDir()
	db := openTestDB(t, appDir)

	if err := SetLastUpdate(db, 123456789); err != nil {
		t.Fatalf("SetLastUpdate() error = %v", err)
	}
	state, err := EnsureRuntime(appDir)
	if err != nil {
		t.Fatalf("EnsureRuntime() error = %v", err)
	}
	if state.LastUpdate != 123456789 {
		t.Fatalf("LastUpdate = %d, want 123456789", state.LastUpdate)
	}
}

func TestSetKnowledgeCommitForAgentPersistsValue(t *testing.T) {
	appDir := t.TempDir()
	db := openTestDB(t, appDir)

	if err := SetKnowledgeCommitForAgent(db, "agent-a", true); err != nil {
		t.Fatalf("SetKnowledgeCommitForAgent() error = %v", err)
	}
	state, err := EnsureRuntimeForAgent(appDir, "agent-a")
	if err != nil {
		t.Fatalf("EnsureRuntimeForAgent() error = %v", err)
	}
	if !state.KnowledgeCommit {
		t.Fatal("KnowledgeCommit = false, want true")
	}
}

func TestMergeMetadataAddsKnowledgeField(t *testing.T) {
	appDir := t.TempDir()

	merged, err := MergeMetadata(map[string]any{
		"agentId": "A1",
		"chat":    "C1",
	}, appDir)
	if err != nil {
		t.Fatalf("MergeMetadata() error = %v", err)
	}
	t.Cleanup(func() {
		_ = CloseSharedDB()
	})

	if merged["agentId"] != "A1" {
		t.Fatalf("agentId = %v, want A1", merged["agentId"])
	}
	knowledge, ok := merged["knowledge"].(*State)
	if !ok {
		t.Fatalf("knowledge type = %T, want *State", merged["knowledge"])
	}
	wantKnowledge := filepath.Join(appDir, defaultKnowledgeDirName, "A1")
	if knowledge.Path != wantKnowledge {
		t.Fatalf("knowledge.path = %q, want %q", knowledge.Path, wantKnowledge)
	}
	if knowledge.KnowledgeCommit {
		t.Fatal("knowledge.knowledgeCommit = true, want false")
	}
}

func TestLookupRuntimeReturnsNilWhenKnowledgeMissing(t *testing.T) {
	appDir := t.TempDir()

	state, err := LookupRuntime(appDir)
	if err != nil {
		t.Fatalf("LookupRuntime() error = %v", err)
	}
	if state != nil {
		t.Fatalf("LookupRuntime() = %#v, want nil", state)
	}
}

func TestLookupRuntimeReadsExistingStateWithoutCreating(t *testing.T) {
	appDir := t.TempDir()
	knowledgeDir := filepath.Join(appDir, defaultKnowledgeDirName)
	if err := os.MkdirAll(knowledgeDir, 0o755); err != nil {
		t.Fatalf("mkdir knowledge dir: %v", err)
	}
	db := openTestDB(t, appDir)
	if err := SetLastUpdate(db, 42); err != nil {
		t.Fatalf("SetLastUpdate() error = %v", err)
	}
	_ = CloseSharedDB()

	state, err := LookupRuntime(appDir)
	if err != nil {
		t.Fatalf("LookupRuntime() error = %v", err)
	}
	if state == nil {
		t.Fatal("LookupRuntime() = nil, want state")
	}
	if state.Path != knowledgeDir {
		t.Fatalf("Path = %q, want %q", state.Path, knowledgeDir)
	}
	if state.LastUpdate != 42 {
		t.Fatalf("LastUpdate = %d, want 42", state.LastUpdate)
	}
	if state.KnowledgeCommit {
		t.Fatal("KnowledgeCommit = true, want false")
	}
}

func TestMetadataIfExistsOmitsKnowledgeWhenMissing(t *testing.T) {
	appDir := t.TempDir()

	metadata, err := MetadataIfExists(appDir)
	if err != nil {
		t.Fatalf("MetadataIfExists() error = %v", err)
	}
	if metadata != nil {
		t.Fatalf("MetadataIfExists() = %#v, want nil", metadata)
	}
}

func TestMarshalStateMatchesKnowledgeSchema(t *testing.T) {
	appDir := t.TempDir()

	data, err := MarshalState(appDir)
	if err != nil {
		t.Fatalf("MarshalState() error = %v", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("unmarshal state: %v", err)
	}
	if state.Path == "" {
		t.Fatal("Path is empty")
	}
	if state.LastUpdate != 0 {
		t.Fatalf("LastUpdate = %d, want 0", state.LastUpdate)
	}
	if state.KnowledgeCommit {
		t.Fatal("KnowledgeCommit = true, want false")
	}
}

func TestDBPathUsesMacRuntimeDirectoryForCurrentWorkingDir(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("mac-specific runtime path")
	}

	home := t.TempDir()
	cwd := t.TempDir()

	restoreExec := knowledgeExecutableFn
	restoreHome := knowledgeUserHomeFn
	knowledgeExecutableFn = func() (string, error) {
		return filepath.Join(home, "Apps", "integration.app", "Contents", "MacOS", "integration"), nil
	}
	knowledgeUserHomeFn = func() (string, error) { return home, nil }
	defer func() {
		knowledgeExecutableFn = restoreExec
		knowledgeUserHomeFn = restoreHome
	}()

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() {
		_ = os.Chdir(prevWD)
	}()

	got, err := DBPath(".")
	if err != nil {
		t.Fatalf("DBPath(.) error = %v", err)
	}
	want := filepath.Join(runtimepaths.MacAppRuntimeBaseDir(home, defaultRuntimeBundleID, defaultRuntimeAppName), defaultSQLiteFileName)
	if got != want {
		t.Fatalf("DBPath(.) = %q, want %q", got, want)
	}
}

func TestDBPathKeepsExplicitAppDirOnMac(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("mac-specific runtime path")
	}

	home := t.TempDir()
	appDir := t.TempDir()

	restoreExec := knowledgeExecutableFn
	restoreHome := knowledgeUserHomeFn
	knowledgeExecutableFn = func() (string, error) {
		return filepath.Join(home, "Apps", "integration.app", "Contents", "MacOS", "integration"), nil
	}
	knowledgeUserHomeFn = func() (string, error) { return home, nil }
	defer func() {
		knowledgeExecutableFn = restoreExec
		knowledgeUserHomeFn = restoreHome
	}()

	got, err := DBPath(appDir)
	if err != nil {
		t.Fatalf("DBPath(appDir) error = %v", err)
	}
	want := filepath.Join(appDir, defaultSQLiteFileName)
	if got != want {
		t.Fatalf("DBPath(appDir) = %q, want %q", got, want)
	}
}
