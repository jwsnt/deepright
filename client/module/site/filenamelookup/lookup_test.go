package filenamelookup

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNormalizeCandidate(t *testing.T) {
	got, err := NormalizeCandidate("  `report.md`  ")
	if err != nil {
		t.Fatalf("NormalizeCandidate returned error: %v", err)
	}
	if got != "report.md" {
		t.Fatalf("NormalizeCandidate = %q, want %q", got, "report.md")
	}
	if _, err := NormalizeCandidate("tmp/report.md"); err == nil {
		t.Fatalf("NormalizeCandidate should reject paths")
	}
}

func TestLookupOrdersWorkspaceTmpAndImages(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "report.md"))
	mustWriteFile(t, filepath.Join(root, "docs", "report.md"))
	mustWriteFile(t, filepath.Join(root, "tmp", "report.md"))
	mustWriteFile(t, filepath.Join(root, "images", "report.md"))
	mustWriteFile(t, filepath.Join(root, "tmp", "nested", "report.md"))
	got, err := Lookup(root, "report.md")
	if err != nil {
		t.Fatalf("Lookup returned error: %v", err)
	}
	want := []Match{
		{Scope: ScopeWorkspace, Path: filepath.Join(root, "report.md")},
		{Scope: ScopeWorkspace, Path: filepath.Join(root, "docs", "report.md")},
		{Scope: ScopeTmp, Path: filepath.Join(root, "tmp", "report.md")},
		{Scope: ScopeTmp, Path: filepath.Join(root, "tmp", "nested", "report.md")},
		{Scope: ScopeImages, Path: filepath.Join(root, "images", "report.md")},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Lookup = %#v, want %#v", got, want)
	}
}

func TestLookupSkipsMissingOptionalScopes(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "note.txt"))
	got, err := Lookup(root, "note.txt")
	if err != nil {
		t.Fatalf("Lookup returned error: %v", err)
	}
	want := []Match{
		{Scope: ScopeWorkspace, Path: filepath.Join(root, "note.txt")},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Lookup = %#v, want %#v", got, want)
	}
}

func mustWriteFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) failed: %v", path, err)
	}
	if err := os.WriteFile(path, []byte("ok"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) failed: %v", path, err)
	}
}
