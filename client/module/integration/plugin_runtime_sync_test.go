package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"runtimepaths"
	"strings"
	"testing"
)

func TestPrepareIntegrationRuntimeLayoutForBundleLaunchBlocksPendingUpdatesWhenPortOccupied(t *testing.T) {
	bundleRoot := filepath.Join(t.TempDir(), "DeepRight.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	bundledPluginsDir := filepath.Join(bundleRoot, "Contents", "Resources", "plugins")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bundledPluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundledPluginsDir, "browser"), []byte("new-plugin"), 0o755); err != nil {
		t.Fatal(err)
	}

	runtimeRoot := t.TempDir()
	pluginDir := filepath.Join(runtimeRoot, "plugins")
	if runtime.GOOS == "darwin" {
		pluginDir = filepath.Join(runtimepaths.MacAppRuntimeBaseDir(runtimeRoot, integrationBundleID, integrationRuntimeAppDir), "plugins")
	}
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(pluginDir, "browser")
	if err := os.WriteFile(targetPath, []byte("old-plugin"), 0o755); err != nil {
		t.Fatal(err)
	}

	oldExecutable := integrationExecutableFn
	oldHome := integrationUserHomeFn
	oldPortOccupied := integrationPortOccupiedCheck
	oldAlert := integrationBundlePluginUpdateAlertFn
	oldRuntimeEnv := os.Getenv(integrationRuntimeDirEnv)
	oldPluginEnv := os.Getenv(integrationPluginDirEnv)
	defer func() {
		integrationExecutableFn = oldExecutable
		integrationUserHomeFn = oldHome
		integrationPortOccupiedCheck = oldPortOccupied
		integrationBundlePluginUpdateAlertFn = oldAlert
		_ = os.Setenv(integrationRuntimeDirEnv, oldRuntimeEnv)
		_ = os.Setenv(integrationPluginDirEnv, oldPluginEnv)
	}()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(macOSDir, "integration"), nil
	}
	if runtime.GOOS == "darwin" {
		integrationUserHomeFn = func() (string, error) { return runtimeRoot, nil }
	} else {
		if err := os.Setenv(integrationRuntimeDirEnv, runtimeRoot); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Unsetenv(integrationPluginDirEnv); err != nil {
		t.Fatal(err)
	}
	integrationPortOccupiedCheck = func(int) bool { return true }
	alertCalled := false
	integrationBundlePluginUpdateAlertFn = func(layout *integrationBundlePaths, items []integrationBundledPluginSyncItem) error {
		alertCalled = true
		if layout == nil || layout.BundleRoot != bundleRoot {
			t.Fatalf("layout = %#v, want bundleRoot=%q", layout, bundleRoot)
		}
		if len(items) != 1 || items[0].Name != "browser" {
			t.Fatalf("items = %#v", items)
		}
		return nil
	}

	var stderr bytes.Buffer
	blocked, err := prepareIntegrationRuntimeLayoutForBundleLaunch(integrationServicePort, &stderr)
	if err != nil {
		t.Fatalf("prepareIntegrationRuntimeLayoutForBundleLaunch() error = %v", err)
	}
	if !blocked {
		t.Fatal("expected update to be blocked while port is occupied")
	}
	if !alertCalled {
		t.Fatal("expected plugin update alert to be shown")
	}
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old-plugin" {
		t.Fatalf("runtime plugin was overwritten: %q", string(data))
	}
	if got, want := strings.TrimSpace(stderr.String()), "有插件需要更新，请重启应用。"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
	if got := os.Getenv(integrationPluginDirEnv); got != pluginDir {
		t.Fatalf("plugin dir env = %q, want %q", got, pluginDir)
	}
}

func TestPrepareIntegrationRuntimeLayoutForBundleLaunchSyncsPendingUpdatesWhenPortFree(t *testing.T) {
	bundleRoot := filepath.Join(t.TempDir(), "DeepRight.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	bundledPluginsDir := filepath.Join(bundleRoot, "Contents", "Resources", "plugins")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bundledPluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundledPluginsDir, "browser"), []byte("new-plugin"), 0o755); err != nil {
		t.Fatal(err)
	}

	runtimeRoot := t.TempDir()
	pluginDir := filepath.Join(runtimeRoot, "plugins")
	if runtime.GOOS == "darwin" {
		pluginDir = filepath.Join(runtimepaths.MacAppRuntimeBaseDir(runtimeRoot, integrationBundleID, integrationRuntimeAppDir), "plugins")
	}
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(pluginDir, "browser")
	if err := os.WriteFile(targetPath, []byte("old-plugin"), 0o755); err != nil {
		t.Fatal(err)
	}

	oldExecutable := integrationExecutableFn
	oldHome := integrationUserHomeFn
	oldPortOccupied := integrationPortOccupiedCheck
	oldAlert := integrationBundlePluginUpdateAlertFn
	oldRuntimeEnv := os.Getenv(integrationRuntimeDirEnv)
	oldPluginEnv := os.Getenv(integrationPluginDirEnv)
	defer func() {
		integrationExecutableFn = oldExecutable
		integrationUserHomeFn = oldHome
		integrationPortOccupiedCheck = oldPortOccupied
		integrationBundlePluginUpdateAlertFn = oldAlert
		_ = os.Setenv(integrationRuntimeDirEnv, oldRuntimeEnv)
		_ = os.Setenv(integrationPluginDirEnv, oldPluginEnv)
	}()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(macOSDir, "integration"), nil
	}
	if runtime.GOOS == "darwin" {
		integrationUserHomeFn = func() (string, error) { return runtimeRoot, nil }
	} else {
		if err := os.Setenv(integrationRuntimeDirEnv, runtimeRoot); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Unsetenv(integrationPluginDirEnv); err != nil {
		t.Fatal(err)
	}
	integrationPortOccupiedCheck = func(int) bool { return false }
	integrationBundlePluginUpdateAlertFn = func(*integrationBundlePaths, []integrationBundledPluginSyncItem) error {
		t.Fatal("alert should not be shown when port is free")
		return nil
	}

	blocked, err := prepareIntegrationRuntimeLayoutForBundleLaunch(integrationServicePort, nil)
	if err != nil {
		t.Fatalf("prepareIntegrationRuntimeLayoutForBundleLaunch() error = %v", err)
	}
	if blocked {
		t.Fatal("did not expect update to be blocked")
	}
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new-plugin" {
		t.Fatalf("runtime plugin content = %q, want new-plugin", string(data))
	}
	if got := os.Getenv(integrationPluginDirEnv); got != pluginDir {
		t.Fatalf("plugin dir env = %q, want %q", got, pluginDir)
	}
}

func TestRunIntegrationPluginSyncBundledCLICheckOnly(t *testing.T) {
	bundleRoot := filepath.Join(t.TempDir(), "DeepRight.app")
	macOSDir := filepath.Join(bundleRoot, "Contents", "MacOS")
	bundledPluginsDir := filepath.Join(bundleRoot, "Contents", "Resources", "plugins")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bundledPluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundledPluginsDir, "browser"), []byte("new-plugin"), 0o755); err != nil {
		t.Fatal(err)
	}

	runtimeRoot := t.TempDir()
	pluginDir := filepath.Join(runtimeRoot, "plugins")
	if runtime.GOOS == "darwin" {
		pluginDir = filepath.Join(runtimepaths.MacAppRuntimeBaseDir(runtimeRoot, integrationBundleID, integrationRuntimeAppDir), "plugins")
	}
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(pluginDir, "browser")
	if err := os.WriteFile(targetPath, []byte("old-plugin"), 0o755); err != nil {
		t.Fatal(err)
	}

	oldExecutable := integrationExecutableFn
	oldHome := integrationUserHomeFn
	oldRuntimeEnv := os.Getenv(integrationRuntimeDirEnv)
	oldPluginEnv := os.Getenv(integrationPluginDirEnv)
	defer func() {
		integrationExecutableFn = oldExecutable
		integrationUserHomeFn = oldHome
		_ = os.Setenv(integrationRuntimeDirEnv, oldRuntimeEnv)
		_ = os.Setenv(integrationPluginDirEnv, oldPluginEnv)
	}()
	integrationExecutableFn = func() (string, error) {
		return filepath.Join(macOSDir, "integration"), nil
	}
	if runtime.GOOS == "darwin" {
		integrationUserHomeFn = func() (string, error) { return runtimeRoot, nil }
	} else {
		if err := os.Setenv(integrationRuntimeDirEnv, runtimeRoot); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Unsetenv(integrationPluginDirEnv); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runIntegrationPluginSyncBundledCLI([]string{"--check"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runIntegrationPluginSyncBundledCLI() code = %d, stderr=%s", code, stderr.String())
	}
	if got := strings.TrimSpace(stderr.String()); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}

	var resp struct {
		Status int `json:"status"`
		Data   struct {
			NeedsUpdate bool                               `json:"needsUpdate"`
			CheckOnly   bool                               `json:"checkOnly"`
			Updated     []string                           `json:"updated"`
			Pending     []integrationBundledPluginSyncItem `json:"pending"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		t.Fatalf("decode stdout: %v; raw=%s", err, stdout.String())
	}
	if resp.Status != 0 {
		t.Fatalf("status = %d; raw=%s", resp.Status, stdout.String())
	}
	if !resp.Data.CheckOnly {
		t.Fatal("expected checkOnly to be true")
	}
	if !resp.Data.NeedsUpdate {
		t.Fatal("expected needsUpdate to be true")
	}
	if len(resp.Data.Updated) != 0 {
		t.Fatalf("updated = %#v, want empty", resp.Data.Updated)
	}
	if len(resp.Data.Pending) != 1 || resp.Data.Pending[0].Name != "browser" {
		t.Fatalf("pending = %#v", resp.Data.Pending)
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old-plugin" {
		t.Fatalf("runtime plugin content = %q, want old-plugin", string(data))
	}
}
