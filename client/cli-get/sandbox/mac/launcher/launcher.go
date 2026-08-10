package launcher

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const (
	defaultBundleID = "cn.deepright.cli-sandbox"
)

func normalizeMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "filepick":
		return "filepick"
	case "net":
		return "net"
	case "filepick_net":
		return "filepick_net"
	default:
		return ""
	}
}

type Config struct {
	SandboxSrc            string
	OutputDir             string
	AppName               string
	BundleID              string
	Mode                  string
	Identity              string
	KeychainPath          string
	TargetGOOS            string
	TargetGOARCH          string
	Version               string
	BuildNumber           string
	NetworkClient         bool
	NetworkServer         bool
	UserSelectedReadOnly  bool
	UserSelectedReadWrite bool
	DownloadsReadOnly     bool
	DownloadsReadWrite    bool
	HardenedRuntime       bool
	SkipSign              bool
	VerifyOnly            bool
	AppPath               string
}

type RuntimeConfig struct {
	ModuleDir             string
	SandboxSrc            string
	OutputDir             string
	AppName               string
	BundleID              string
	Mode                  string
	Identity              string
	KeychainPath          string
	TargetGOOS            string
	TargetGOARCH          string
	Version               string
	BuildNumber           string
	NetworkClient         bool
	NetworkServer         bool
	UserSelectedReadOnly  bool
	UserSelectedReadWrite bool
	DownloadsReadOnly     bool
	DownloadsReadWrite    bool
	HardenedRuntime       bool
	SkipSign              bool
	VerifyOnly            bool
	AppPath               string
}

type Result struct {
	AppPath                 string
	Identity                string
	AppEntitlementsPath     string
	InheritEntitlementsPath string
	InfoPlistPath           string
}

type runnerConfig struct {
	BundleID string `json:"bundleId"`
	Mode     string `json:"mode"`
}

type plistDict struct {
	Version string
	Dict    []plistKV
}

type plistKV struct {
	Key     string
	String  *string
	Boolean *bool
	Integer *string
}

func Run(cfg Config) (Result, error) {
	moduleDir, _ := os.Getwd()
	action := "build"
	logOutcome := func(status, appPath, identity, detail string) {
		baseDir := moduleDir
		if baseDir == "" {
			baseDir = "."
		}
		appendBuildLog(baseDir, action, status, appPath, identity, detail)
	}

	runCfg, err := NormalizeConfig(cfg)
	if err != nil {
		logOutcome("error", "", "", err.Error())
		return Result{}, err
	}
	moduleDir = runCfg.ModuleDir

	if runCfg.VerifyOnly {
		action = "verify"
		if err := Verify(runCfg.AppPath); err != nil {
			logOutcome("error", runCfg.AppPath, runCfg.Identity, err.Error())
			return Result{}, err
		}
		logOutcome("success", runCfg.AppPath, runCfg.Identity, "verified app bundle")
		return Result{AppPath: runCfg.AppPath, Identity: runCfg.Identity}, nil
	}

	if err := os.MkdirAll(runCfg.OutputDir, 0o755); err != nil {
		logOutcome("error", "", runCfg.Identity, err.Error())
		return Result{}, fmt.Errorf("create output directory: %w", err)
	}

	appPath := filepath.Join(runCfg.OutputDir, runCfg.AppName+".app")
	if err := os.RemoveAll(appPath); err != nil {
		logOutcome("error", appPath, runCfg.Identity, err.Error())
		return Result{}, fmt.Errorf("remove previous app: %w", err)
	}

	contentsDir := filepath.Join(appPath, "Contents")
	macOSDir := filepath.Join(contentsDir, "MacOS")
	helpersDir := filepath.Join(contentsDir, "Helpers")
	resourcesDir := filepath.Join(contentsDir, "Resources")
	buildDir := filepath.Join(runCfg.OutputDir, runCfg.AppName+"-build")

	for _, dir := range []string{macOSDir, helpersDir, resourcesDir, buildDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			logOutcome("error", appPath, runCfg.Identity, err.Error())
			return Result{}, fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	mainExecutable := filepath.Join(macOSDir, runCfg.AppName)
	helperExecutable := filepath.Join(helpersDir, "CLI_SANDBOX")
	infoPlistPath := filepath.Join(contentsDir, "Info.plist")
	appEntitlementsPath := filepath.Join(buildDir, "app.entitlements.plist")
	inheritEntitlementsPath := filepath.Join(buildDir, "inherit.entitlements.plist")
	runnerConfigPath := filepath.Join(resourcesDir, "runner-config.json")

	if err := buildGoBinary(runCfg.ModuleDir, "./runner", mainExecutable, runCfg.TargetGOOS, runCfg.TargetGOARCH); err != nil {
		logOutcome("error", appPath, runCfg.Identity, err.Error())
		return Result{}, err
	}
	if err := buildGoBinary(runCfg.SandboxSrc, ".", helperExecutable, runCfg.TargetGOOS, runCfg.TargetGOARCH); err != nil {
		logOutcome("error", appPath, runCfg.Identity, err.Error())
		return Result{}, err
	}

	if err := writeRunnerConfig(runnerConfigPath, runnerConfig{BundleID: runCfg.BundleID, Mode: runCfg.Mode}); err != nil {
		logOutcome("error", appPath, runCfg.Identity, err.Error())
		return Result{}, err
	}
	if err := writeInfoPlist(infoPlistPath, runCfg); err != nil {
		logOutcome("error", appPath, runCfg.Identity, err.Error())
		return Result{}, err
	}
	if err := writeEntitlements(appEntitlementsPath, buildAppEntitlements(runCfg)); err != nil {
		logOutcome("error", appPath, runCfg.Identity, err.Error())
		return Result{}, err
	}
	if err := writeEntitlements(inheritEntitlementsPath, buildInheritEntitlements()); err != nil {
		logOutcome("error", appPath, runCfg.Identity, err.Error())
		return Result{}, err
	}

	if !runCfg.SkipSign {
		if err := codesignHelper(helperExecutable, inheritEntitlementsPath, runCfg); err != nil {
			logOutcome("error", appPath, runCfg.Identity, err.Error())
			return Result{}, err
		}
		if err := codesignApp(appPath, appEntitlementsPath, runCfg); err != nil {
			logOutcome("error", appPath, runCfg.Identity, err.Error())
			return Result{}, err
		}
		if err := Verify(appPath); err != nil {
			logOutcome("error", appPath, runCfg.Identity, err.Error())
			return Result{}, err
		}
	}

	detail := "built app bundle without signing"
	if !runCfg.SkipSign {
		detail = "built and signed app bundle"
	}
	logOutcome("success", appPath, runCfg.Identity, detail)
	return Result{
		AppPath:                 appPath,
		Identity:                runCfg.Identity,
		AppEntitlementsPath:     appEntitlementsPath,
		InheritEntitlementsPath: inheritEntitlementsPath,
		InfoPlistPath:           infoPlistPath,
	}, nil
}

func NormalizeConfig(cfg Config) (RuntimeConfig, error) {
	if runtime.GOOS != "darwin" {
		return RuntimeConfig{}, errors.New("cli-get sandbox mac packaging only supports darwin")
	}

	moduleDir, err := os.Getwd()
	if err != nil {
		return RuntimeConfig{}, fmt.Errorf("get working directory: %w", err)
	}

	sandboxSrc := strings.TrimSpace(cfg.SandboxSrc)
	if sandboxSrc == "" {
		sandboxSrc = ".."
	}
	if !filepath.IsAbs(sandboxSrc) {
		sandboxSrc = filepath.Join(moduleDir, sandboxSrc)
	}
	sandboxSrc = filepath.Clean(sandboxSrc)
	if _, err := os.Stat(filepath.Join(sandboxSrc, "go.mod")); err != nil {
		return RuntimeConfig{}, fmt.Errorf("sandbox source does not look like a Go module: %s", sandboxSrc)
	}

	outputDir := strings.TrimSpace(cfg.OutputDir)
	if outputDir == "" {
		outputDir = "./dist"
	}
	if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(moduleDir, outputDir)
	}
	outputDir = filepath.Clean(outputDir)

	appName := strings.TrimSpace(cfg.AppName)
	if appName == "" {
		appName = "CLI_SANDBOX"
	}
	appName = strings.TrimSuffix(appName, ".app")

	bundleID := strings.TrimSpace(cfg.BundleID)
	if bundleID == "" {
		bundleID = defaultBundleID
	}
	mode := normalizeMode(cfg.Mode)
	if !cfg.VerifyOnly && mode == "" {
		return RuntimeConfig{}, errors.New("sandbox mode is required")
	}

	version := strings.TrimSpace(cfg.Version)
	if version == "" {
		version = "1.0.0"
	}
	buildNumber := strings.TrimSpace(cfg.BuildNumber)
	if buildNumber == "" {
		buildNumber = "1"
	}

	targetGOOS := strings.TrimSpace(cfg.TargetGOOS)
	if targetGOOS == "" {
		targetGOOS = "darwin"
	}
	targetGOARCH := strings.TrimSpace(cfg.TargetGOARCH)
	if targetGOARCH == "" {
		targetGOARCH = runtime.GOARCH
	}

	appPath := strings.TrimSpace(cfg.AppPath)
	if appPath != "" && !filepath.IsAbs(appPath) {
		appPath = filepath.Join(moduleDir, appPath)
	}
	appPath = filepath.Clean(appPath)

	keychainPath := strings.TrimSpace(cfg.KeychainPath)
	if keychainPath != "" && !filepath.IsAbs(keychainPath) {
		keychainPath = filepath.Join(moduleDir, keychainPath)
	}
	keychainPath = filepath.Clean(keychainPath)
	if strings.TrimSpace(cfg.KeychainPath) == "" {
		keychainPath = ""
	}

	identity := strings.TrimSpace(cfg.Identity)
	if !cfg.SkipSign {
		var err error
		identity, err = resolveIdentity(identity, keychainPath)
		if err != nil {
			return RuntimeConfig{}, err
		}
	}

	return RuntimeConfig{
		ModuleDir:             moduleDir,
		SandboxSrc:            sandboxSrc,
		OutputDir:             outputDir,
		AppName:               appName,
		BundleID:              bundleID,
		Mode:                  mode,
		Identity:              identity,
		KeychainPath:          keychainPath,
		TargetGOOS:            targetGOOS,
		TargetGOARCH:          targetGOARCH,
		Version:               version,
		BuildNumber:           buildNumber,
		NetworkClient:         cfg.NetworkClient,
		NetworkServer:         cfg.NetworkServer,
		UserSelectedReadOnly:  cfg.UserSelectedReadOnly,
		UserSelectedReadWrite: cfg.UserSelectedReadWrite,
		DownloadsReadOnly:     cfg.DownloadsReadOnly,
		DownloadsReadWrite:    cfg.DownloadsReadWrite,
		HardenedRuntime:       cfg.HardenedRuntime,
		SkipSign:              cfg.SkipSign,
		VerifyOnly:            cfg.VerifyOnly,
		AppPath:               appPath,
	}, nil
}

func resolveIdentity(identity, keychainPath string) (string, error) {
	if identity != "" {
		return identity, nil
	}

	args := []string{"find-identity", "-v", "-p", "codesigning"}
	if keychainPath != "" {
		args = append(args, keychainPath)
	}
	cmd := exec.Command("security", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("list codesign identities: %w", err)
	}
	lines := strings.Split(string(out), "\n")
	re := regexp.MustCompile(`"([^"]+)"`)
	var developerID string
	var appleDevelopment string
	for _, line := range lines {
		match := re.FindStringSubmatch(line)
		if len(match) != 2 {
			continue
		}
		name := match[1]
		switch {
		case strings.Contains(name, "Developer ID Application:"):
			if developerID == "" {
				developerID = name
			}
		case strings.Contains(name, "Apple Development:"):
			if appleDevelopment == "" {
				appleDevelopment = name
			}
		}
	}
	switch {
	case developerID != "":
		return developerID, nil
	case appleDevelopment != "":
		return appleDevelopment, nil
	default:
		return "", errors.New("no usable codesign identity found; pass --identity explicitly")
	}
}

func buildGoBinary(dir, pkg, out, targetGOOS, targetGOARCH string) error {
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return fmt.Errorf("create output directory for %s: %w", out, err)
	}
	cmd := exec.Command("go", "build", "-o", out, pkg)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GOOS="+targetGOOS,
		"GOARCH="+targetGOARCH,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed for %s in %s (%s/%s): %w", pkg, dir, targetGOOS, targetGOARCH, err)
	}
	return nil
}

func writeRunnerConfig(path string, cfg runnerConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal runner config: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write runner config: %w", err)
	}
	return nil
}

func writeInfoPlist(path string, cfg RuntimeConfig) error {
	appName := cfg.AppName
	executable := cfg.AppName
	version := cfg.Version
	buildNumber := cfg.BuildNumber
	bundleID := cfg.BundleID
	packageType := "APPL"
	dict := plistDict{
		Version: "1.0",
		Dict: []plistKV{
			{Key: "CFBundleDevelopmentRegion", String: stringPtr("en")},
			{Key: "CFBundleDisplayName", String: &appName},
			{Key: "CFBundleExecutable", String: &executable},
			{Key: "CFBundleIdentifier", String: &bundleID},
			{Key: "CFBundleInfoDictionaryVersion", String: stringPtr("6.0")},
			{Key: "CFBundleName", String: &appName},
			{Key: "CFBundlePackageType", String: &packageType},
			{Key: "CFBundleShortVersionString", String: &version},
			{Key: "CFBundleVersion", String: &buildNumber},
		},
	}
	if cfg.Mode == "net" {
		background := true
		dict.Dict = append(dict.Dict, plistKV{Key: "LSBackgroundOnly", Boolean: &background})
	}
	content, err := marshalPlist(dict)
	if err != nil {
		return fmt.Errorf("marshal Info.plist: %w", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write Info.plist: %w", err)
	}
	return nil
}

func buildAppEntitlements(cfg RuntimeConfig) []plistKV {
	return nil
}

func buildInheritEntitlements() []plistKV {
	return nil
}

func writeEntitlements(path string, items []plistKV) error {
	content, err := marshalPlist(plistDict{Version: "1.0", Dict: items})
	if err != nil {
		return fmt.Errorf("marshal entitlements: %w", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write entitlements: %w", err)
	}
	return nil
}

func marshalPlist(dict plistDict) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	buf.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	buf.WriteString(`<plist version="` + escapeXML(dict.Version) + `">` + "\n")
	buf.WriteString("  <dict>\n")
	for _, item := range dict.Dict {
		buf.WriteString("    <key>" + escapeXML(item.Key) + "</key>\n")
		switch {
		case item.String != nil:
			buf.WriteString("    <string>" + escapeXML(*item.String) + "</string>\n")
		case item.Boolean != nil:
			if *item.Boolean {
				buf.WriteString("    <true/>\n")
			} else {
				buf.WriteString("    <false/>\n")
			}
		case item.Integer != nil:
			buf.WriteString("    <integer>" + escapeXML(*item.Integer) + "</integer>\n")
		default:
			buf.WriteString("    <string></string>\n")
		}
	}
	buf.WriteString("  </dict>\n")
	buf.WriteString("</plist>\n")
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

func stringPtr(value string) *string {
	return &value
}

func escapeXML(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(value)
}

func codesignHelper(path, entitlementsPath string, cfg RuntimeConfig) error {
	args := []string{
		"--force",
		"--sign", cfg.Identity,
		"--timestamp=none",
		"--entitlements", entitlementsPath,
	}
	if cfg.KeychainPath != "" {
		args = append(args, "--keychain", cfg.KeychainPath)
	}
	args = append(args, path)
	cmd := exec.Command("codesign", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("codesign helper failed: %w", err)
	}
	return nil
}

func codesignApp(appPath, entitlementsPath string, cfg RuntimeConfig) error {
	args := []string{
		"--force",
		"--sign", cfg.Identity,
		"--timestamp=none",
		"--entitlements", entitlementsPath,
	}
	if cfg.KeychainPath != "" {
		args = append(args, "--keychain", cfg.KeychainPath)
	}
	if cfg.HardenedRuntime {
		args = append(args, "--options", "runtime")
	}
	args = append(args, appPath)
	cmd := exec.Command("codesign", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("codesign app failed: %w", err)
	}
	return nil
}

func Verify(appPath string) error {
	if strings.TrimSpace(appPath) == "" {
		return errors.New("--app-path is required with --verify-only")
	}
	cmd := exec.Command("codesign", "--verify", "--deep", "--strict", "--verbose=2", appPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("codesign verification failed: %w", err)
	}
	return nil
}
