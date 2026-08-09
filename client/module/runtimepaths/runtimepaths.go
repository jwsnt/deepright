package runtimepaths

import (
	"path/filepath"
	"strings"
)

const (
	DeepRightAppName             = "deepright"
	DeepRightMacBundleIdentifier = "cn.deepright.integration"
)

func MacAppContainerRoot(home, bundleID string) string {
	home = strings.TrimSpace(home)
	if home == "" {
		return ""
	}
	bundleID = strings.TrimSpace(bundleID)
	if bundleID == "" {
		bundleID = DeepRightMacBundleIdentifier
	}
	return filepath.Join(home, "Library", "Containers", bundleID, "Data")
}

func MacAppRuntimeBaseDir(home, bundleID, appName string) string {
	root := MacAppContainerRoot(home, bundleID)
	if root == "" {
		return ""
	}
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = DeepRightAppName
	}
	return filepath.Join(root, "Library", "Application Support", appName)
}

func MacLegacyRuntimeBaseDir(home, appName string) string {
	home = strings.TrimSpace(home)
	if home == "" {
		return ""
	}
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = DeepRightAppName
	}
	return filepath.Join(home, "Library", "Application Support", appName)
}

func MacRuntimeConfigCandidates(home, bundleID, appName string) []string {
	out := make([]string, 0, 2)
	add := func(base string) {
		base = strings.TrimSpace(base)
		if base == "" {
			return
		}
		out = append(out, filepath.Join(base, "config", "config.json"))
	}
	add(MacAppRuntimeBaseDir(home, bundleID, appName))
	add(MacLegacyRuntimeBaseDir(home, appName))
	return out
}
