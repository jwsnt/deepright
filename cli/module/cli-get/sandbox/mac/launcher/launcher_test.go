package launcher

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildAppEntitlementsStaysEmpty(t *testing.T) {
	if items := buildAppEntitlements(RuntimeConfig{Mode: "filepick"}); len(items) != 0 {
		t.Fatalf("buildAppEntitlements should stay empty, got %+v", items)
	}
}

func TestBuildInheritEntitlementsStaysEmpty(t *testing.T) {
	if items := buildInheritEntitlements(); len(items) != 0 {
		t.Fatalf("buildInheritEntitlements should stay empty, got %+v", items)
	}
}

func TestNormalizeConfigDefaultsAndAutoIdentity(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only identity test")
	}
	tmp := t.TempDir()
	sandboxSrc := filepath.Join(tmp, "sandbox")
	if err := os.MkdirAll(sandboxSrc, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sandboxSrc, "go.mod"), []byte("module sandbox\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("WriteFile go.mod: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	cfg, err := NormalizeConfig(Config{SandboxSrc: sandboxSrc, Mode: "net"})
	if err != nil {
		t.Fatalf("NormalizeConfig: %v", err)
	}
	if cfg.AppName != "CLI_SANDBOX" {
		t.Fatalf("AppName = %q", cfg.AppName)
	}
	if cfg.BundleID != defaultBundleID {
		t.Fatalf("BundleID = %q", cfg.BundleID)
	}
	if cfg.Mode != "net" {
		t.Fatalf("Mode = %q", cfg.Mode)
	}
	if cfg.TargetGOOS != "darwin" {
		t.Fatalf("TargetGOOS = %q", cfg.TargetGOOS)
	}
	if cfg.TargetGOARCH == "" {
		t.Fatal("TargetGOARCH should default to the host architecture")
	}
	if cfg.Identity == "" {
		t.Fatal("Identity should auto-resolve")
	}
}

func TestWriteInfoPlistIncludesBundleMetadata(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "Info.plist")
	cfg := RuntimeConfig{
		AppName:     "CLI_SANDBOX",
		BundleID:    "cn.deepright.cli-sandbox",
		Mode:        "filepick",
		Version:     "1.2.3",
		BuildNumber: "7",
	}
	if err := writeInfoPlist(path, cfg); err != nil {
		t.Fatalf("writeInfoPlist: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "CFBundleIdentifier") || !strings.Contains(text, "cn.deepright.cli-sandbox") {
		t.Fatalf("missing bundle id: %s", text)
	}
	if !strings.Contains(text, "CFBundleExecutable") || !strings.Contains(text, "CLI_SANDBOX") {
		t.Fatalf("missing executable: %s", text)
	}
	if strings.Contains(text, "LSBackgroundOnly") {
		t.Fatalf("filepick plist should not be background-only: %s", text)
	}
}

func TestWriteInfoPlistNetModeStaysBackgroundOnly(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "Info.plist")
	cfg := RuntimeConfig{
		AppName:     "CLI_SANDBOX",
		BundleID:    "cn.deepright.cli-sandbox.net",
		Mode:        "net",
		Version:     "1.2.3",
		BuildNumber: "7",
	}
	if err := writeInfoPlist(path, cfg); err != nil {
		t.Fatalf("writeInfoPlist: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "LSBackgroundOnly") {
		t.Fatalf("net plist should stay background-only: %s", text)
	}
}

func TestNormalizeConfigFilepickKeepsUserSelectedDisabledByDefault(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only identity test")
	}
	tmp := t.TempDir()
	sandboxSrc := filepath.Join(tmp, "sandbox")
	if err := os.MkdirAll(sandboxSrc, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sandboxSrc, "go.mod"), []byte("module sandbox\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("WriteFile go.mod: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	cfg, err := NormalizeConfig(Config{SandboxSrc: sandboxSrc, Mode: "filepick"})
	if err != nil {
		t.Fatalf("NormalizeConfig: %v", err)
	}
	if cfg.UserSelectedReadWrite {
		t.Fatal("filepick mode should not auto-enable user-selected read-write entitlement")
	}
	if cfg.UserSelectedReadOnly {
		t.Fatal("filepick mode should not auto-enable read-only entitlement")
	}
}

func TestNormalizeConfigRespectsExplicitUserSelectedReadOnly(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only identity test")
	}
	tmp := t.TempDir()
	sandboxSrc := filepath.Join(tmp, "sandbox")
	if err := os.MkdirAll(sandboxSrc, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sandboxSrc, "go.mod"), []byte("module sandbox\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("WriteFile go.mod: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	cfg, err := NormalizeConfig(Config{
		SandboxSrc:           sandboxSrc,
		Mode:                 "filepick_net",
		UserSelectedReadOnly: true,
	})
	if err != nil {
		t.Fatalf("NormalizeConfig: %v", err)
	}
	if !cfg.UserSelectedReadOnly {
		t.Fatal("explicit read-only entitlement should be preserved")
	}
	if cfg.UserSelectedReadWrite {
		t.Fatal("explicit read-only entitlement should not be upgraded to read-write")
	}
}
