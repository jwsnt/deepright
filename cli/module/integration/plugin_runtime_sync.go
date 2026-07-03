package main

import (
	"connect/connectsvc"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type integrationBundledPluginSyncItem struct {
	Name       string `json:"name"`
	SourcePath string `json:"sourcePath"`
	TargetPath string `json:"targetPath"`
	SourceMD5  string `json:"sourceMD5"`
	TargetMD5  string `json:"targetMD5,omitempty"`
}

type integrationBundledPluginSyncPlan struct {
	BundledPluginDir string                             `json:"bundledPluginDir"`
	RuntimePluginDir string                             `json:"runtimePluginDir"`
	Pending          []integrationBundledPluginSyncItem `json:"pending"`
}

var integrationPortOccupiedCheck = integrationPortOccupied
var integrationBundlePluginUpdateAlertFn = showIntegrationBundlePluginUpdateAlert

func prepareIntegrationRuntimeBaseDir() (string, error) {
	runtimeDir := integrationBundleRuntimeBaseDir()
	if runtimeDir == "" {
		return "", nil
	}
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		return "", fmt.Errorf("prepare runtime dir: %w", err)
	}
	if err := migrateLegacyIntegrationDB(runtimeDir); err != nil {
		return "", fmt.Errorf("migrate runtime db: %w", err)
	}
	return runtimeDir, nil
}

func resolveIntegrationRuntimePluginDirFromRuntimeBase(runtimeDir string) (string, error) {
	pluginDir := strings.TrimSpace(integrationRuntimePluginDir())
	if pluginDir == "" {
		pluginDir = strings.TrimSpace(os.Getenv(integrationPluginDirEnv))
		if pluginDir == "" && strings.TrimSpace(runtimeDir) != "" {
			pluginDir = filepath.Join(runtimeDir, "plugins")
		}
	}
	if pluginDir == "" {
		return "", fmt.Errorf("resolve application plugins dir")
	}
	info, err := os.Stat(pluginDir)
	if err == nil && !info.IsDir() {
		return "", fmt.Errorf("plugins runtime path is not a directory: %s", pluginDir)
	}
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("stat plugins dir: %w", err)
	}
	if os.IsNotExist(err) {
		if err := os.MkdirAll(pluginDir, 0o755); err != nil {
			return "", fmt.Errorf("prepare plugins runtime dir: %w", err)
		}
	}
	return pluginDir, nil
}

func buildIntegrationBundledPluginSyncPlan(srcDir, dstDir string) (integrationBundledPluginSyncPlan, error) {
	srcDir = strings.TrimSpace(srcDir)
	dstDir = strings.TrimSpace(dstDir)
	plan := integrationBundledPluginSyncPlan{
		BundledPluginDir: srcDir,
		RuntimePluginDir: dstDir,
		Pending:          []integrationBundledPluginSyncItem{},
	}
	if srcDir == "" {
		return plan, fmt.Errorf("bundled plugins dir is empty")
	}
	if dstDir == "" {
		return plan, fmt.Errorf("runtime plugins dir is empty")
	}
	srcInfo, err := os.Stat(srcDir)
	if err != nil {
		if os.IsNotExist(err) && integrationBundleLayout() != nil {
			return plan, fmt.Errorf("bundled plugins dir not found: %s", srcDir)
		}
		return plan, err
	}
	if !srcInfo.IsDir() {
		return plan, fmt.Errorf("bundled plugins path is not a directory: %s", srcDir)
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return plan, err
	}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return plan, err
	}
	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return plan, err
		}
		if !shouldSyncBundledPluginEntry(name, info) {
			continue
		}
		srcPath := filepath.Join(srcDir, name)
		dstPath := filepath.Join(dstDir, name)
		srcMD5, err := fileMD5Hex(srcPath)
		if err != nil {
			return plan, err
		}
		dstMD5, err := fileMD5Hex(dstPath)
		if err == nil && strings.EqualFold(srcMD5, dstMD5) {
			continue
		}
		if err != nil && !os.IsNotExist(err) {
			return plan, err
		}
		plan.Pending = append(plan.Pending, integrationBundledPluginSyncItem{
			Name:       name,
			SourcePath: srcPath,
			TargetPath: dstPath,
			SourceMD5:  srcMD5,
			TargetMD5:  strings.TrimSpace(dstMD5),
		})
	}
	sort.Slice(plan.Pending, func(i, j int) bool {
		return plan.Pending[i].Name < plan.Pending[j].Name
	})
	return plan, nil
}

func fileMD5Hex(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func applyIntegrationBundledPluginSyncPlan(plan integrationBundledPluginSyncPlan) error {
	for _, item := range plan.Pending {
		info, err := os.Stat(item.SourcePath)
		if err != nil {
			return err
		}
		if err := copyReleaseAssetWithPermissions(item.SourcePath, item.TargetPath, info.Mode()); err != nil {
			return err
		}
	}
	return nil
}

func prepareIntegrationRuntimeLayoutForBundleLaunch(port int, stderr io.Writer) (bool, error) {
	runtimeDir, err := prepareIntegrationRuntimeBaseDir()
	if err != nil {
		return false, err
	}
	pluginDir, err := resolveIntegrationRuntimePluginDirFromRuntimeBase(runtimeDir)
	if err != nil {
		return false, err
	}
	bundledPluginDir := strings.TrimSpace(integrationBundledPluginDir())
	if bundledPluginDir != "" {
		plan, err := buildIntegrationBundledPluginSyncPlan(bundledPluginDir, pluginDir)
		if err != nil {
			return false, err
		}
		if len(plan.Pending) > 0 && integrationPortOccupiedCheck(port) {
			if err := integrationBundlePluginUpdateAlertFn(integrationBundleLayout(), plan.Pending); err != nil {
				log.Printf("show plugin update alert failed: %v", err)
			}
			if stderr != nil {
				fmt.Fprintln(stderr, integrationBundlePluginUpdateMessage(plan.Pending))
			}
			if err := os.Setenv(integrationPluginDirEnv, pluginDir); err != nil {
				return false, fmt.Errorf("export plugin dir: %w", err)
			}
			return true, nil
		}
		if err := applyIntegrationBundledPluginSyncPlan(plan); err != nil {
			return false, fmt.Errorf("sync bundled plugins: %w", err)
		}
	}
	if err := os.Setenv(integrationPluginDirEnv, pluginDir); err != nil {
		return false, fmt.Errorf("export plugin dir: %w", err)
	}
	return false, nil
}

func integrationBundlePluginUpdateMessage(items []integrationBundledPluginSyncItem) string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return "有插件需要更新，请重启应用。"
	}
	return fmt.Sprintf("有插件需要更新，请重启应用。\n待更新插件：%s", strings.Join(names, ", "))
}

func showIntegrationBundlePluginUpdateAlert(layout *integrationBundlePaths, items []integrationBundledPluginSyncItem) error {
	osascriptPath, err := integrationBundleLaunchAlertLookPathFn("osascript")
	if err != nil {
		return fmt.Errorf("locate osascript: %w", err)
	}
	title := "DeepRight.app"
	if layout != nil {
		if name := strings.TrimSpace(filepath.Base(layout.BundleRoot)); name != "" {
			title = name
		}
	}
	cmd := integrationBundleLaunchAlertCommandFn(
		osascriptPath,
		"-e",
		`on run argv
display alert (item 1 of argv) message (item 2 of argv) buttons {"好"} default button "好"
end run`,
		title,
		integrationBundlePluginUpdateMessage(items),
	)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	text := strings.TrimSpace(string(output))
	if text == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, text)
}

func integrationPortOccupied(port int) bool {
	if port <= 0 {
		port = integrationServicePort
	}
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err == nil {
		_ = listener.Close()
		return false
	}
	return true
}

func runIntegrationPluginSyncBundledCLI(args []string, stdout, stderr io.Writer) int {
	flags, err := connectsvc.ParseFlags(args)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	checkOnly := queryBool(flags["check"])
	runtimeDir, err := prepareIntegrationRuntimeBaseDir()
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	pluginDir, err := resolveIntegrationRuntimePluginDirFromRuntimeBase(runtimeDir)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	plan, err := buildIntegrationBundledPluginSyncPlan(integrationBundledPluginDir(), pluginDir)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	updated := make([]string, 0, len(plan.Pending))
	if !checkOnly && len(plan.Pending) > 0 {
		for _, item := range plan.Pending {
			updated = append(updated, item.Name)
		}
		if err := applyIntegrationBundledPluginSyncPlan(plan); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
	}
	out, _ := json.MarshalIndent(map[string]any{
		"status": 0,
		"data": map[string]any{
			"bundledPluginDir": plan.BundledPluginDir,
			"runtimePluginDir": plan.RuntimePluginDir,
			"needsUpdate":      len(plan.Pending) > 0,
			"pending":          plan.Pending,
			"updated":          updated,
			"checkOnly":        checkOnly,
		},
	}, "", "  ")
	fmt.Fprintln(stdout, string(out))
	return 0
}
