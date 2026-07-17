package main

import (
	"strings"
	"testing"
)

func TestBrowserIntegrationPluginsRootBlocksWhenAppDirIsMissing(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	connectBin, runtimePath := writeBundledConnectBinFixture(t, homeDir, map[string]string{})
	_, ok, err := browserIntegrationPluginsRootFromConnectBin(connectBin)
	if err == nil {
		t.Fatal("expected missing app-dir to block Browser startup")
	}
	if ok {
		t.Fatal("missing app-dir must not resolve a plugins root")
	}
	for _, want := range []string{"runtime configuration is incomplete", runtimePath, "restart integration"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want %q", err, want)
		}
	}
}
