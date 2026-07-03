package launchsplash

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildCommandUsesOsaScriptWithEnv(t *testing.T) {
	oldLookPath := launchSplashLookPathFn
	oldGOOS := launchSplashGOOSFn
	defer func() {
		launchSplashLookPathFn = oldLookPath
		launchSplashGOOSFn = oldGOOS
	}()

	logoPath := filepath.Join(t.TempDir(), "logo.png")
	if err := os.WriteFile(logoPath, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}

	launchSplashGOOSFn = func() string { return "darwin" }
	launchSplashLookPathFn = func(name string) (string, error) {
		if name != "osascript" {
			t.Fatalf("look path name = %q", name)
		}
		return "/usr/bin/osascript", nil
	}

	cmd, err := buildCommand(Config{
		LogoPath: logoPath,
		Duration: 7 * time.Second,
	})
	if err != nil {
		t.Fatalf("buildCommand() error = %v", err)
	}
	if cmd.Path != "/usr/bin/osascript" {
		t.Fatalf("cmd.Path = %q", cmd.Path)
	}
	if len(cmd.Args) != 5 || cmd.Args[1] != "-l" || cmd.Args[2] != "JavaScript" || cmd.Args[3] != "-e" {
		t.Fatalf("cmd.Args = %#v", cmd.Args)
	}
	envText := strings.Join(cmd.Env, "\n")
	if !strings.Contains(envText, "DEEPRIGHT_SPLASH_LOGO="+logoPath) {
		t.Fatalf("missing splash logo env: %s", envText)
	}
	if !strings.Contains(envText, "DEEPRIGHT_SPLASH_DURATION_MS=7000") {
		t.Fatalf("missing splash duration env: %s", envText)
	}
}

func TestBuildCommandRejectsUnsupportedPlatform(t *testing.T) {
	oldGOOS := launchSplashGOOSFn
	defer func() { launchSplashGOOSFn = oldGOOS }()
	launchSplashGOOSFn = func() string { return "linux" }

	_, err := buildCommand(Config{LogoPath: "/tmp/logo.png"})
	if !errors.Is(err, ErrUnsupportedPlatform) {
		t.Fatalf("err = %v", err)
	}
}

func TestNormalizeConfigDefaultsDuration(t *testing.T) {
	logoPath := filepath.Join(t.TempDir(), "logo.png")
	if err := os.WriteFile(logoPath, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := normalizeConfig(Config{LogoPath: logoPath})
	if err != nil {
		t.Fatalf("normalizeConfig() error = %v", err)
	}
	if cfg.Duration != DefaultDuration {
		t.Fatalf("cfg.Duration = %v, want %v", cfg.Duration, DefaultDuration)
	}
	if !filepath.IsAbs(cfg.LogoPath) {
		t.Fatalf("cfg.LogoPath = %q, want absolute path", cfg.LogoPath)
	}
}
