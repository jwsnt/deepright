package main

import (
	"flag"
	"fmt"
	"os"

	"cli-get-sandbox-mac/launcher"
)

func main() {
	cfg := launcher.Config{}

	flag.StringVar(&cfg.SandboxSrc, "sandbox-src", "..", "path to the CLI_SANDBOX Go module")
	flag.StringVar(&cfg.OutputDir, "output-dir", "./dist", "directory used to write the signed .app")
	flag.StringVar(&cfg.AppName, "app-name", "CLI_SANDBOX", "app bundle name without .app suffix")
	flag.StringVar(&cfg.BundleID, "bundle-id", "cn.deepright.cli-sandbox", "bundle identifier used for the .app and sandbox container")
	flag.StringVar(&cfg.Mode, "mode", "", "sandbox mode: filepick, net, filepick_net")
	flag.StringVar(&cfg.Identity, "identity", "", "codesign identity; empty means auto-select Developer ID Application then Apple Development")
	flag.StringVar(&cfg.KeychainPath, "keychain", "", "optional keychain path used for identity lookup and codesign")
	flag.StringVar(&cfg.TargetGOOS, "target-goos", "darwin", "target GOOS used to build the app bundle")
	flag.StringVar(&cfg.TargetGOARCH, "target-goarch", "", "target GOARCH used to build the app bundle; default is the host architecture")
	flag.StringVar(&cfg.Version, "version", "1.0.0", "CFBundleShortVersionString")
	flag.StringVar(&cfg.BuildNumber, "build-number", "1", "CFBundleVersion")
	flag.BoolVar(&cfg.NetworkClient, "network-client", true, "enable com.apple.security.network.client entitlement")
	flag.BoolVar(&cfg.NetworkServer, "network-server", true, "enable com.apple.security.network.server entitlement")
	flag.BoolVar(&cfg.UserSelectedReadOnly, "user-selected-read-only", false, "enable com.apple.security.files.user-selected.read-only entitlement")
	flag.BoolVar(&cfg.UserSelectedReadWrite, "user-selected-read-write", false, "enable com.apple.security.files.user-selected.read-write entitlement")
	flag.BoolVar(&cfg.DownloadsReadOnly, "downloads-read-only", false, "enable com.apple.security.files.downloads.read-only entitlement")
	flag.BoolVar(&cfg.DownloadsReadWrite, "downloads-read-write", false, "enable com.apple.security.files.downloads.read-write entitlement")
	flag.BoolVar(&cfg.HardenedRuntime, "hardened-runtime", false, "sign the app with hardened runtime")
	flag.BoolVar(&cfg.SkipSign, "skip-sign", false, "build the .app bundle without codesign")
	flag.BoolVar(&cfg.VerifyOnly, "verify-only", false, "verify an existing app at --app-path instead of rebuilding")
	flag.StringVar(&cfg.AppPath, "app-path", "", "existing .app path used with --verify-only")
	flag.Parse()

	result, err := launcher.Run(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if cfg.VerifyOnly {
		fmt.Printf("verified=%s\n", result.AppPath)
		fmt.Printf("identity=%s\n", result.Identity)
		return
	}

	fmt.Printf("app=%s\n", result.AppPath)
	fmt.Printf("identity=%s\n", result.Identity)
	fmt.Printf("entitlements=%s\n", result.AppEntitlementsPath)
	fmt.Printf("inherit-entitlements=%s\n", result.InheritEntitlementsPath)
	fmt.Printf("info-plist=%s\n", result.InfoPlistPath)
}
