package launcher

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppendBuildLogWritesSQLiteDataFile(t *testing.T) {
	tmp := t.TempDir()
	resetBuildLogDBForTest()
	defer resetBuildLogDBForTest()

	if err := ensureBaseDir(tmp); err != nil {
		t.Fatalf("ensureBaseDir: %v", err)
	}
	appendBuildLog(tmp, "build", "success", "/tmp/a.app", "identity-1", "done")

	if _, err := os.Stat(filepath.Join(tmp, "data")); err != nil {
		t.Fatalf("expected sqlite data file: %v", err)
	}
	count, err := buildLogCount(tmp)
	if err != nil {
		t.Fatalf("buildLogCount: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	action, status, err := buildLogLatest(tmp)
	if err != nil {
		t.Fatalf("buildLogLatest: %v", err)
	}
	if action != "build" || status != "success" {
		t.Fatalf("latest = (%s,%s), want (build,success)", action, status)
	}
}
