package skillstate

import (
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"

	_ "github.com/glebarez/go-sqlite"
)

func TestApplyDisabledPaths(t *testing.T) {
	root := t.TempDir()
	alpha := filepath.Join(root, "skills", "alpha")
	beta := filepath.Join(alpha, "beta")
	gamma := filepath.Join(root, "skills", "gamma")

	got := ApplyDisabledPaths(nil, beta, true)
	if want := []string{NormalizePath(beta)}; !reflect.DeepEqual(got, want) {
		t.Fatalf("disable beta = %#v, want %#v", got, want)
	}

	got = ApplyDisabledPaths(got, alpha, true)
	if want := []string{NormalizePath(alpha)}; !reflect.DeepEqual(got, want) {
		t.Fatalf("disable alpha = %#v, want %#v", got, want)
	}

	got = ApplyDisabledPaths(got, gamma, true)
	if want := []string{NormalizePath(alpha), NormalizePath(gamma)}; !reflect.DeepEqual(got, want) {
		t.Fatalf("disable gamma = %#v, want %#v", got, want)
	}

	got = ApplyDisabledPaths(got, alpha, false)
	if want := []string{NormalizePath(gamma)}; !reflect.DeepEqual(got, want) {
		t.Fatalf("enable alpha = %#v, want %#v", got, want)
	}

	got = ApplyDisabledPaths([]string{NormalizePath(alpha)}, beta, true)
	if want := []string{NormalizePath(alpha)}; !reflect.DeepEqual(got, want) {
		t.Fatalf("disable beta with alpha ancestor = %#v, want %#v", got, want)
	}
}

func TestDisabledStatus(t *testing.T) {
	root := t.TempDir()
	alpha := filepath.Join(root, "skills", "alpha")
	beta := filepath.Join(alpha, "beta")

	self, inherited := DisabledStatus([]string{alpha}, alpha)
	if !self || inherited {
		t.Fatalf("alpha status = self:%v inherited:%v, want true/false", self, inherited)
	}

	self, inherited = DisabledStatus([]string{alpha}, beta)
	if self || !inherited {
		t.Fatalf("beta status = self:%v inherited:%v, want false/true", self, inherited)
	}
}

func TestSetDisabledPersistsNormalizedPaths(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	root := t.TempDir()
	alpha := filepath.Join(root, "skills", "alpha")
	beta := filepath.Join(alpha, "beta")

	next, err := SetDisabled(db, "chat-1", beta, true)
	if err != nil {
		t.Fatalf("SetDisabled beta: %v", err)
	}
	if want := []string{NormalizePath(beta)}; !reflect.DeepEqual(next, want) {
		t.Fatalf("stored beta = %#v, want %#v", next, want)
	}

	next, err = SetDisabled(db, "chat-1", alpha, true)
	if err != nil {
		t.Fatalf("SetDisabled alpha: %v", err)
	}
	if want := []string{NormalizePath(alpha)}; !reflect.DeepEqual(next, want) {
		t.Fatalf("stored alpha = %#v, want %#v", next, want)
	}

	stored, err := ListDisabledPaths(db, "chat-1")
	if err != nil {
		t.Fatalf("ListDisabledPaths: %v", err)
	}
	if want := []string{NormalizePath(alpha)}; !reflect.DeepEqual(stored, want) {
		t.Fatalf("stored paths = %#v, want %#v", stored, want)
	}

	next, err = SetDisabled(db, "chat-1", alpha, false)
	if err != nil {
		t.Fatalf("SetDisabled enable alpha: %v", err)
	}
	if len(next) != 0 {
		t.Fatalf("enabled alpha next = %#v, want empty", next)
	}
}
