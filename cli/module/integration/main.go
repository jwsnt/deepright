package main

import (
	"agent-scanner/agentcore"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"connect/connectsvc"
	"connect/logretention"
	"connect/sandboxstate"
	"connect/skillstate"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"knowledge/knowledgecore"
	"log"
	"maps"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"database/sql"

	"connect/pluginlog"
	"connect/sharedutil"
	_ "github.com/glebarez/go-sqlite"
	"github.com/robfig/cron/v3"
	integrationagentarchive "integration/agentarchive"
	integrationagentcopy "integration/agentcopy"
	tokenconsume "integration/consume"
	"integration/http11client"
	integrationknowledge "integration/knowledge"
	"integration/launchsplash"
	integrationnotification "integration/notification"
	"integration/runtimehost"
	integrationstandalone "integration/standalone"
	"runtimepaths"
	"skill-scanner/skillscore"
	"static-server/server"
)

type CmdRequest = sharedutil.CmdRequest
type KillRequest = sharedutil.KillRequest
type CmdResponse = sharedutil.CmdResponse
type KillResponse = sharedutil.KillResponse

const (
	seedreamProviderName            = "seedream"
	seedreamInternalSkillName       = "__internal_seedream"
	legacySeedreamSkillName         = "image-seedream"
	seedreamDefaultURL              = "https://ark.cn-beijing.volces.com/api/v3/images/generations"
	seedreamDefaultModelMultiOutput = "doubao-seedream-4.5"
	seedreamSkillDescription        = "文生图、单图/多图生图。凭据自动获取，支持自定义尺寸/数量/返回格式"
)

func findAgent(agents []agentcore.Agent, agentID string) *agentcore.Agent {
	for i := range agents {
		if agents[i].AgentID == agentID {
			return &agents[i]
		}
	}
	return nil
}

func normalizeRouterRemoteList(input interface{}) []string {
	switch value := input.(type) {
	case []string:
		out := make([]string, 0, len(value))
		for _, item := range value {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []interface{}:
		out := make([]string, 0, len(value))
		for _, item := range value {
			text := strings.TrimSpace(fmt.Sprint(item))
			if text != "" && text != "<nil>" {
				out = append(out, text)
			}
		}
		return out
	case string:
		value = strings.TrimSpace(value)
		if value == "" {
			return nil
		}
		return []string{value}
	default:
		return nil
	}
}

var integrationInstallAppPlatformKeyFn = integrationInstallAppPlatformKey

func integrationInstallAppPlatformKey() string {
	switch runtime.GOOS {
	case "darwin":
		return "mac"
	case "windows":
		return "wsl"
	case "linux":
		if integrationBrowserIsWSL() {
			return "wsl"
		}
		return "linux"
	default:
		return ""
	}
}

func integrationAppendInstallApps(base []string, raw interface{}) []string {
	switch value := raw.(type) {
	case nil:
		return base
	case string:
		return mergeInstallApps(base, value)
	case []string:
		for _, item := range value {
			base = mergeInstallApps(base, item)
		}
	case []interface{}:
		for _, item := range value {
			text := strings.TrimSpace(fmt.Sprint(item))
			if text == "" || text == "<nil>" {
				continue
			}
			base = mergeInstallApps(base, text)
		}
	case map[string]interface{}:
		for _, key := range []string{"linux", "wsl", "mac"} {
			base = integrationAppendInstallApps(base, value[key])
		}
	default:
		text := strings.TrimSpace(fmt.Sprint(value))
		if text == "" || text == "<nil>" {
			return base
		}
		base = mergeInstallApps(base, text)
	}
	return base
}

func integrationInstallAppsFromConfigValue(raw interface{}, platform string) []string {
	platform = strings.ToLower(strings.TrimSpace(platform))
	if value, ok := raw.(map[string]interface{}); ok {
		if platform != "" {
			return integrationAppendInstallApps(nil, value[platform])
		}
		return integrationAppendInstallApps(nil, value)
	}
	return integrationAppendInstallApps(nil, raw)
}

const integrationInstallAppCacheTTL = 5 * time.Minute

type integrationInstallAppCacheEntry struct {
	installed bool
	expiresAt time.Time
}

var integrationInstallAppAvailabilityCache struct {
	mu    sync.Mutex
	items map[string]integrationInstallAppCacheEntry
}

var integrationInstallAppNowFn = time.Now
var integrationInstallAppLookPathFn = exec.LookPath
var integrationInstallAppStatFn = os.Stat
var integrationInstallAppUserHomeDirFn = os.UserHomeDir
var integrationInstallAppDetectorFn = integrationDetectInstallAppInstalled

func integrationResetInstallAppAvailabilityCache() {
	integrationInstallAppAvailabilityCache.mu.Lock()
	integrationInstallAppAvailabilityCache.items = nil
	integrationInstallAppAvailabilityCache.mu.Unlock()
}

func integrationInstallAppCacheKey(name string) string {
	return integrationInstallAppPlatformKeyFn() + "|" + strings.ToLower(strings.TrimSpace(name))
}

func integrationInstallAppNameVariants(name string) []string {
	variants := make([]string, 0, 8)
	seen := make(map[string]struct{})
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		variants = append(variants, value)
	}

	name = strings.TrimSpace(name)
	add(name)
	lower := strings.ToLower(name)
	add(lower)
	add(strings.ReplaceAll(lower, " ", ""))
	add(strings.ReplaceAll(lower, " ", "-"))
	add(strings.ReplaceAll(lower, " ", "_"))
	return variants
}

func integrationInstallAppCommandCandidates(name string) []string {
	candidates := integrationInstallAppNameVariants(name)
	if integrationInstallAppPlatformKeyFn() == "wsl" || runtime.GOOS == "windows" {
		base := append([]string(nil), candidates...)
		for _, candidate := range base {
			if !strings.HasSuffix(strings.ToLower(candidate), ".exe") {
				candidates = append(candidates, candidate+".exe")
			}
		}
	}
	return candidates
}

func integrationResolveExecutableFromPaths(paths []string) string {
	for _, candidate := range paths {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if resolved := resolveExecutablePath(candidate); resolved != "" {
			return resolved
		}
	}
	return ""
}

func integrationResolveExecutableFromGlobs(patterns []string) string {
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		if resolved := integrationResolveExecutableFromPaths(matches); resolved != "" {
			return resolved
		}
	}
	return ""
}

func integrationInstallAppCommonExecutablePaths(name string) []string {
	platform := integrationInstallAppPlatformKeyFn()
	if platform != "mac" && platform != "linux" {
		return nil
	}
	candidates := integrationInstallAppCommandCandidates(name)
	if len(candidates) == 0 {
		return nil
	}

	baseDirs := []string{"/opt/homebrew/bin", "/usr/local/bin", "/usr/bin"}
	homeDir, _ := integrationInstallAppUserHomeDirFn()
	homeDir = strings.TrimSpace(homeDir)
	if homeDir != "" {
		baseDirs = append(baseDirs,
			filepath.Join(homeDir, ".local", "bin"),
			filepath.Join(homeDir, ".local", "node", "bin"),
			filepath.Join(homeDir, "bin"),
			filepath.Join(homeDir, ".npm-global", "bin"),
			filepath.Join(homeDir, ".volta", "bin"),
			filepath.Join(homeDir, ".fnm", "current", "bin"),
			filepath.Join(homeDir, ".n", "bin"),
		)
	}

	paths := make([]string, 0, len(baseDirs)*len(candidates))
	for _, dir := range baseDirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		for _, candidate := range candidates {
			paths = append(paths, filepath.Join(dir, candidate))
		}
	}
	return paths
}

func integrationInstallAppCommonExecutableGlobs(name string) []string {
	platform := integrationInstallAppPlatformKeyFn()
	if platform != "mac" && platform != "linux" {
		return nil
	}
	candidates := integrationInstallAppCommandCandidates(name)
	if len(candidates) == 0 {
		return nil
	}

	homeDir, _ := integrationInstallAppUserHomeDirFn()
	homeDir = strings.TrimSpace(homeDir)
	if homeDir == "" {
		return nil
	}

	patterns := make([]string, 0, len(candidates)*4)
	for _, candidate := range candidates {
		patterns = append(patterns,
			filepath.Join(homeDir, ".nvm", "versions", "node", "*", "bin", candidate),
			filepath.Join(homeDir, ".asdf", "installs", "nodejs", "*", "bin", candidate),
			filepath.Join(homeDir, ".mise", "installs", "node", "*", "bin", candidate),
			filepath.Join(homeDir, ".local", "share", "fnm", "node-versions", "*", "installation", "bin", candidate),
		)
		if platform == "mac" {
			patterns = append(patterns,
				filepath.Join(homeDir, "Library", "Application Support", "fnm", "node-versions", "*", "installation", "bin", candidate),
			)
		}
	}
	return patterns
}

func integrationFindInstallAppExecutable(name string) string {
	for _, candidate := range integrationInstallAppCommandCandidates(name) {
		path, err := integrationInstallAppLookPathFn(candidate)
		if err != nil {
			continue
		}
		if resolved := resolveExecutablePath(path); resolved != "" {
			return resolved
		}
	}
	if resolved := integrationResolveExecutableFromPaths(integrationInstallAppCommonExecutablePaths(name)); resolved != "" {
		return resolved
	}
	return integrationResolveExecutableFromGlobs(integrationInstallAppCommonExecutableGlobs(name))
}

func integrationInstallAppExists(paths []string) bool {
	for _, candidate := range paths {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		info, err := integrationInstallAppStatFn(candidate)
		if err == nil && info != nil {
			return true
		}
	}
	return false
}

func integrationInstallAppMatchesAnyGlob(patterns []string) bool {
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		if integrationInstallAppExists(matches) {
			return true
		}
	}
	return false
}

func integrationDetectInstallAppInstalled(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return integrationInstallAppExists([]string{name})
	}
	if integrationFindInstallAppExecutable(name) != "" {
		return true
	}

	switch integrationInstallAppPlatformKeyFn() {
	case "mac":
		appNames := integrationInstallAppNameVariants(name)
		paths := make([]string, 0, len(appNames)*4)
		homeDir, _ := integrationInstallAppUserHomeDirFn()
		for _, appName := range appNames {
			paths = append(paths,
				filepath.Join("/Applications", appName),
				filepath.Join("/Applications", appName+".app"),
			)
			if strings.TrimSpace(homeDir) != "" {
				paths = append(paths,
					filepath.Join(homeDir, "Applications", appName),
					filepath.Join(homeDir, "Applications", appName+".app"),
				)
			}
		}
		return integrationInstallAppExists(paths)
	case "wsl":
		executables := integrationInstallAppCommandCandidates(name)
		baseDirs := []string{}
		if runtime.GOOS == "windows" {
			for _, dir := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("LocalAppData")} {
				dir = strings.TrimSpace(dir)
				if dir != "" {
					baseDirs = append(baseDirs, dir)
				}
			}
		} else {
			baseDirs = append(baseDirs,
				"/mnt/c/Program Files",
				"/mnt/c/Program Files (x86)",
				"/mnt/c/Users/*/AppData/Local",
			)
		}
		directPaths := make([]string, 0, len(baseDirs)*len(executables)*2)
		globPatterns := make([]string, 0, len(baseDirs)*len(executables)*3)
		for _, baseDir := range baseDirs {
			for _, executable := range executables {
				directPaths = append(directPaths,
					filepath.Join(baseDir, executable),
					filepath.Join(baseDir, "*", executable),
				)
				globPatterns = append(globPatterns,
					filepath.Join(baseDir, "*", "Application", executable),
					filepath.Join(baseDir, "*", "*", "Application", executable),
					filepath.Join(baseDir, "*", "*", executable),
				)
			}
		}
		return integrationInstallAppExists(directPaths) || integrationInstallAppMatchesAnyGlob(globPatterns)
	default:
		return false
	}
}

func integrationIsInstallAppInstalled(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	now := integrationInstallAppNowFn()
	key := integrationInstallAppCacheKey(name)

	integrationInstallAppAvailabilityCache.mu.Lock()
	if entry, ok := integrationInstallAppAvailabilityCache.items[key]; ok && now.Before(entry.expiresAt) {
		integrationInstallAppAvailabilityCache.mu.Unlock()
		return entry.installed
	}
	integrationInstallAppAvailabilityCache.mu.Unlock()

	installed := integrationInstallAppDetectorFn(name)

	integrationInstallAppAvailabilityCache.mu.Lock()
	if integrationInstallAppAvailabilityCache.items == nil {
		integrationInstallAppAvailabilityCache.items = make(map[string]integrationInstallAppCacheEntry)
	}
	integrationInstallAppAvailabilityCache.items[key] = integrationInstallAppCacheEntry{
		installed: installed,
		expiresAt: now.Add(integrationInstallAppCacheTTL),
	}
	integrationInstallAppAvailabilityCache.mu.Unlock()
	return installed
}

func integrationFilterMissingInstallApps(apps []string) []string {
	out := make([]string, 0, len(apps))
	for _, app := range apps {
		app = strings.TrimSpace(app)
		if app == "" || integrationIsInstallAppInstalled(app) {
			continue
		}
		out = mergeInstallApps(out, app)
	}
	return out
}

func readRouterRemoteFromWorkspace(workspace string) []string {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(workspace, "config.json"))
	if err != nil {
		return nil
	}
	var raw struct {
		RouterRemote interface{} `json:"router_remote"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	return normalizeRouterRemoteList(raw.RouterRemote)
}

func readAgentRouterRemote(agentDir, deviceID, agentID string, ttl time.Duration) []string {
	metadata, err := getAgentOutput(agentDir, deviceID, ttl)
	if err != nil {
		return nil
	}
	agent := findAgent(metadata.Agents, agentID)
	if agent == nil {
		return nil
	}
	return readRouterRemoteFromWorkspace(agent.Workspace)
}

// ═══════════════════════════════════════════════════════════════════════════
// Shared: Skill & Agent metadata
// ═══════════════════════════════════════════════════════════════════════════

type Skill = agentcore.Skill
type Agent = agentcore.Agent
type AgentOutput = agentcore.Output

type connectMetaConfigListItem = connectsvc.MetaConfig
type pluginActionResponse = connectsvc.PluginActionResponse
type pluginStatusResponse = connectsvc.PluginStatusResponse

func detectTerminal() string { return agentcore.DetectTerminal() }

func resolveExecutablePath(path string) string { return agentcore.ResolveExecutablePath(path) }

func detectGit() string { return agentcore.DetectGit() }

func detectPython3() string { return agentcore.DetectPython3() }

const (
	integrationDefaultGitUserEmail = "deepright@deepright.cn"
	integrationDefaultGitUserName  = "deepright"
)

var integrationGitLookPathFn = exec.LookPath
var integrationGitReadConfigFn = func(key string) (string, error) {
	out, err := exec.Command("git", "config", "--get", key).CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return "", nil
		}
		text := strings.TrimSpace(string(out))
		if text == "" {
			return "", fmt.Errorf("git config --get %s: %w", key, err)
		}
		return "", fmt.Errorf("git config --get %s: %w: %s", key, err, text)
	}
	return strings.TrimSpace(string(out)), nil
}
var integrationGitWriteConfigFn = func(key, value string) error {
	out, err := exec.Command("git", "config", "--global", key, value).CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(out))
		if text == "" {
			return fmt.Errorf("git config --global %s: %w", key, err)
		}
		return fmt.Errorf("git config --global %s: %w: %s", key, err, text)
	}
	return nil
}
var integrationEnsureGitIdentityConfiguredFn = ensureIntegrationGitIdentityConfigured

func ensureIntegrationGitIdentityConfigured() error {
	if _, err := integrationGitLookPathFn("git"); err != nil {
		return nil
	}

	userName, err := integrationGitReadConfigFn("user.name")
	if err != nil {
		return err
	}
	userEmail, err := integrationGitReadConfigFn("user.email")
	if err != nil {
		return err
	}

	missingName := strings.TrimSpace(userName) == ""
	missingEmail := strings.TrimSpace(userEmail) == ""
	if !missingName && !missingEmail {
		return nil
	}

	if missingEmail {
		if err := integrationGitWriteConfigFn("user.email", integrationDefaultGitUserEmail); err != nil {
			return err
		}
		log.Printf("integration: git user.email missing, set default global value %q", integrationDefaultGitUserEmail)
	}
	if missingName {
		if err := integrationGitWriteConfigFn("user.name", integrationDefaultGitUserName); err != nil {
			return err
		}
		log.Printf("integration: git user.name missing, set default global value %q", integrationDefaultGitUserName)
	}
	return nil
}

func detectRequiredInstallApps() []string {
	apps := agentcore.DetectRequiredInstallApps()
	if detectPython3() == "" {
		return mergeInstallApps(apps, "python3")
	}
	return apps
}

func parseInstallApps(raw string) []string { return agentcore.ParseInstallApps(raw) }

func mergeInstallApps(base []string, extras ...string) []string {
	return agentcore.MergeInstallApps(base, extras...)
}

func detectGateway() string { return agentcore.DetectGateway() }

func detectSys() string { return agentcore.DetectSys() }

func generateDeviceID() string { return agentcore.GenerateDeviceID() }

func flushAgentCache() { agentcore.FlushCache() }

func processKnowledgeMetadata(cfg *Config, metaMap map[string]interface{}, now time.Time, forceLastUpdate bool) error {
	if metaMap == nil {
		return nil
	}
	agentID, _ := metaMap["agentId"].(string)
	agentID = strings.TrimSpace(agentID)
	rawKnowledge, ok := metaMap["knowledge"]
	if !ok || rawKnowledge == nil {
		return nil
	}
	knowledge, ok := rawKnowledge.(map[string]interface{})
	if !ok {
		return nil
	}
	knowledgePath, pathErr := integrationKnowledgeDirForAgent(cfg.AgentDir, agentID)
	if pathErr != nil {
		return fmt.Errorf("resolve knowledge path: %w", pathErr)
	}
	knowledge["path"] = knowledgePath
	var (
		lastUpdate int64
		err        error
	)
	if agentID != "" {
		db, openErr := knowledgecore.OpenSharedDB(integrationAppDir())
		if openErr != nil {
			return fmt.Errorf("open knowledge runtime: %w", openErr)
		}
		lastUpdate, err = knowledgecore.GetLastUpdateForAgent(db, agentID)
		if err != nil {
			return fmt.Errorf("load knowledge last update: %w", err)
		}
		knowledge["lastUpdate"] = lastUpdate
		knowledgeCommit, err := knowledgecore.GetKnowledgeCommitForAgent(db, agentID)
		if err != nil {
			return fmt.Errorf("load knowledge commit: %w", err)
		}
		knowledge["knowledgeCommit"] = knowledgeCommit
	} else {
		lastUpdateRaw, exists := knowledge["lastUpdate"]
		if !exists {
			return nil
		}
		parsedLastUpdate, ok := sharedutil.ToInt64Value(lastUpdateRaw)
		if !ok || parsedLastUpdate < 0 {
			return nil
		}
		lastUpdate = parsedLastUpdate
	}
	if forceLastUpdate {
		return nil
	}

	interval := time.Duration(cfg.KnowledgeUpdateIntervalMs) * time.Millisecond
	if interval <= 0 {
		interval = 2 * time.Hour
	}
	lockWindow := time.Duration(cfg.KnowledgeUpdateLockMs) * time.Millisecond
	if lockWindow <= 0 {
		lockWindow = 30 * time.Minute
	}

	nowMs := now.UnixMilli()
	if nowMs-lastUpdate <= interval.Milliseconds() {
		delete(knowledge, "lastUpdate")
		return nil
	}

	db, err := knowledgecore.OpenSharedDB(integrationAppDir())
	if err != nil {
		return fmt.Errorf("open knowledge runtime: %w", err)
	}
	if err := sharedutil.EnsureKnowledgeRequestLockRow(db, agentID); err != nil {
		return err
	}

	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin knowledge_update_lock tx: %w", err)
	}
	defer tx.Rollback()

	if err := sharedutil.EnsureKnowledgeRequestLockRowTx(tx, agentID); err != nil {
		return err
	}

	var lastRequestedAt int64
	if err := tx.QueryRow(`SELECT last_requested_at FROM knowledge_update_lock WHERE agent_id = ?`, agentID).Scan(&lastRequestedAt); err != nil {
		return fmt.Errorf("query knowledge_update_lock: %w", err)
	}
	if nowMs-lastRequestedAt <= lockWindow.Milliseconds() {
		delete(knowledge, "lastUpdate")
		return tx.Commit()
	}
	if _, err := tx.Exec(`UPDATE knowledge_update_lock SET last_requested_at = ? WHERE agent_id = ?`, nowMs, agentID); err != nil {
		return fmt.Errorf("update knowledge_update_lock: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit knowledge_update_lock: %w", err)
	}
	return nil
}

func updateKnowledgeManualTimestamps(now time.Time, agentID string) error {
	db, err := knowledgecore.OpenSharedDB(integrationAppDir())
	if err != nil {
		return fmt.Errorf("open knowledge runtime: %w", err)
	}
	agentID = strings.TrimSpace(agentID)
	if err := sharedutil.EnsureKnowledgeRequestLockRow(db, agentID); err != nil {
		return err
	}
	nowMs := now.UnixMilli()
	if err := knowledgecore.SetLastUpdateForAgent(db, agentID, nowMs); err != nil {
		return fmt.Errorf("set knowledge last update: %w", err)
	}
	if _, err := db.Exec(`UPDATE knowledge_update_lock SET last_requested_at = ? WHERE agent_id = ?`, nowMs, agentID); err != nil {
		return fmt.Errorf("update knowledge_update_lock: %w", err)
	}
	return nil
}

func persistKnowledgeCommitSelection(agentID string, value bool) error {
	db, err := knowledgecore.OpenSharedDB(integrationAppDir())
	if err != nil {
		return fmt.Errorf("open knowledge runtime: %w", err)
	}
	if err := knowledgecore.SetKnowledgeCommitForAgent(db, agentID, value); err != nil {
		return fmt.Errorf("set knowledge commit: %w", err)
	}
	return nil
}

func integrationAppDir() string {
	if runtimeDir := integrationBundleRuntimeBaseDir(); runtimeDir != "" {
		return runtimeDir
	}
	return integrationCurrentAppDir()
}

func integrationExecutableBaseDir() string {
	if layout := integrationBundleLayout(); layout != nil {
		if abs, err := filepath.Abs(layout.ResourcesDir); err == nil {
			return abs
		}
		return filepath.Clean(layout.ResourcesDir)
	}
	if exeDir := integrationExecutableDir(); exeDir != "" {
		return exeDir
	}
	if cwd, err := os.Getwd(); err == nil && strings.TrimSpace(cwd) != "" {
		if abs, err := filepath.Abs(cwd); err == nil {
			return abs
		}
		return filepath.Clean(cwd)
	}
	return ""
}

func integrationCurrentAppDir() string {
	if startupCfg := readRuntimeConfig(); startupCfg != nil {
		if resourcesDir := strings.TrimSpace(startupCfg["resources-dir"]); resourcesDir != "" {
			if abs, err := filepath.Abs(resourcesDir); err == nil {
				return abs
			}
			return filepath.Clean(resourcesDir)
		}
		if appDir := strings.TrimSpace(startupCfg["app-dir"]); appDir != "" {
			if abs, err := filepath.Abs(appDir); err == nil {
				return abs
			}
			return filepath.Clean(appDir)
		}
		if appPath := strings.TrimSpace(startupCfg["app"]); appPath != "" {
			if abs, err := filepath.Abs(filepath.Dir(appPath)); err == nil {
				return abs
			}
			return filepath.Clean(filepath.Dir(appPath))
		}
	}
	return integrationExecutableBaseDir()
}

type integrationBundlePaths struct {
	BundleRoot   string
	ContentsDir  string
	MacOSDir     string
	ResourcesDir string
	HelpersDir   string
}

const (
	integrationPluginDirEnv  = "DEEPRIGHT_PLUGIN_DIR"
	integrationRuntimeDirEnv = "DEEPRIGHT_INTEGRATION_RUNTIME_DIR"
	integrationRuntimeAppDir = "deepright"
	integrationBundleID      = "cn.deepright.integration"
)

var integrationUserHomeFn = os.UserHomeDir
var integrationEvalSymlinksFn = filepath.EvalSymlinks
var integrationBundleLaunchAlertFn = showIntegrationBundleLaunchLocationAlert
var integrationBundleLaunchAlertLookPathFn = exec.LookPath
var integrationBundleLaunchAlertCommandFn = exec.Command
var integrationRuntimeGOOS = runtime.GOOS

func integrationManagedRuntimeBaseDir() string {
	switch integrationRuntimeGOOS {
	case "darwin":
		home, err := integrationUserHomeFn()
		if err != nil || strings.TrimSpace(home) == "" {
			return ""
		}
		return runtimepaths.MacAppRuntimeBaseDir(home, integrationBundleID, integrationRuntimeAppDir)
	case "linux":
		if !integrationBrowserIsWSL() {
			return ""
		}
		home, err := integrationUserHomeFn()
		if err != nil || strings.TrimSpace(home) == "" {
			return ""
		}
		return filepath.Join(home, integrationRuntimeAppDir)
	default:
		return ""
	}
}

func integrationExecutablePath() string {
	exe, err := integrationExecutableFn()
	if err != nil {
		return ""
	}
	exe = strings.TrimSpace(exe)
	if exe == "" || strings.HasSuffix(exe, ".test") {
		return ""
	}
	if abs, err := filepath.Abs(exe); err == nil {
		return abs
	}
	return filepath.Clean(exe)
}

func integrationBundleLayout() *integrationBundlePaths {
	exe := integrationExecutablePath()
	if exe == "" {
		return nil
	}
	macOSDir := filepath.Dir(exe)
	if filepath.Base(macOSDir) != "MacOS" {
		return nil
	}
	contentsDir := filepath.Dir(macOSDir)
	if filepath.Base(contentsDir) != "Contents" {
		return nil
	}
	bundleRoot := filepath.Dir(contentsDir)
	if !strings.HasSuffix(strings.ToLower(bundleRoot), ".app") {
		return nil
	}
	return &integrationBundlePaths{
		BundleRoot:   bundleRoot,
		ContentsDir:  contentsDir,
		MacOSDir:     macOSDir,
		ResourcesDir: filepath.Join(contentsDir, "Resources"),
		HelpersDir:   filepath.Join(contentsDir, "Helpers"),
	}
}

func integrationBundleRuntimeBaseDir() string {
	if managed := strings.TrimSpace(integrationManagedRuntimeBaseDir()); managed != "" {
		return managed
	}
	if explicit := strings.TrimSpace(os.Getenv(integrationRuntimeDirEnv)); explicit != "" {
		if abs, err := filepath.Abs(explicit); err == nil {
			return abs
		}
		return filepath.Clean(explicit)
	}
	return ""
}

func integrationBundleBrowserReuseMarkerPath(layout *integrationBundlePaths) (string, error) {
	layout = layoutOrCurrentIntegrationBundle(layout)
	if layout == nil {
		return "", errors.New("bundle layout is unavailable")
	}
	runtimeBase := strings.TrimSpace(integrationBundleRuntimeBaseDir())
	if runtimeBase == "" {
		return "", errors.New("bundle runtime dir is unavailable")
	}
	fingerprint, err := integrationBundleLaunchFingerprint(layout)
	if err != nil {
		return "", err
	}
	return filepath.Join(runtimeBase, "state", "browser-reuse", fingerprint+".seen"), nil
}

func integrationBundleLaunchFingerprint(layout *integrationBundlePaths) (string, error) {
	layout = layoutOrCurrentIntegrationBundle(layout)
	if layout == nil {
		return "", errors.New("bundle layout is unavailable")
	}
	parts := make([]string, 0, 3)
	if part, ok := integrationBundleLaunchFingerprintPart(filepath.Join(layout.ContentsDir, "Info.plist")); ok {
		parts = append(parts, "plist:"+part)
	}
	if part, ok := integrationBundleLaunchFingerprintPart(filepath.Join(layout.MacOSDir, "integration")); ok {
		parts = append(parts, "exe:"+part)
	}
	if part, ok := integrationBundleLaunchFingerprintPart(layout.BundleRoot); ok {
		parts = append(parts, "bundle:"+part)
	}
	if len(parts) == 0 {
		return "", errors.New("bundle fingerprint inputs are unavailable")
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:]), nil
}

func integrationBundleLaunchFingerprintPart(path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", false
	}
	info, err := os.Stat(path)
	if err != nil || info == nil {
		return "", false
	}
	return fmt.Sprintf("%d:%d", info.Size(), info.ModTime().UnixNano()), true
}

func integrationShouldBypassExistingBrowserReuse(layout *integrationBundlePaths) (bool, func() error) {
	markerPath, err := integrationBundleBrowserReuseMarkerPath(layout)
	if err != nil {
		return false, func() error { return nil }
	}
	if _, err := os.Stat(markerPath); err == nil {
		return false, func() error { return nil }
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, func() error { return nil }
	}
	return true, func() error {
		if err := os.MkdirAll(filepath.Dir(markerPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(markerPath, []byte(time.Now().Format(time.RFC3339Nano)+"\n"), 0o644)
	}
}

func layoutOrCurrentIntegrationBundle(layout *integrationBundlePaths) *integrationBundlePaths {
	if layout != nil {
		return layout
	}
	return integrationBundleLayout()
}

func integrationAllowedApplicationDirs() []string {
	out := []string{"/Applications"}
	if home, err := integrationUserHomeFn(); err == nil && strings.TrimSpace(home) != "" {
		out = append(out, filepath.Join(home, "Applications"))
	}
	return out
}

func integrationResolveBundleLaunchPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	} else {
		path = filepath.Clean(path)
	}
	if resolved, err := integrationEvalSymlinksFn(path); err == nil && strings.TrimSpace(resolved) != "" {
		return filepath.Clean(resolved)
	}
	return filepath.Clean(path)
}

func integrationBundleLaunchAllowed(layout *integrationBundlePaths) bool {
	// Bundled app launches are no longer restricted to /Applications or
	// ~/Applications. Keep the helper so existing launch flow stays intact.
	return true
}

func integrationBundleLaunchBlockedMessage(layout *integrationBundlePaths) string {
	appName := "DeepRight.app"
	if layout != nil {
		if name := strings.TrimSpace(filepath.Base(layout.BundleRoot)); name != "" {
			appName = name
		}
	}
	return fmt.Sprintf("%s 必须放在“应用程序”文件夹中才能启动。请将它移动到 /Applications 或 ~/Applications 后再打开。", appName)
}

func showIntegrationBundleLaunchLocationAlert(layout *integrationBundlePaths) error {
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
display alert (item 1 of argv) message (item 2 of argv) as critical buttons {"好"} default button "好"
end run`,
		title,
		integrationBundleLaunchBlockedMessage(layout),
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

func ensureIntegrationBundleLaunchAllowed(layout *integrationBundlePaths, stderr io.Writer) bool {
	if integrationBundleLaunchAllowed(layout) {
		return true
	}
	if err := integrationBundleLaunchAlertFn(layout); err != nil {
		log.Printf("show bundle launch alert failed: %v", err)
	}
	if stderr != nil {
		location := ""
		if layout != nil {
			location = integrationResolveBundleLaunchPath(layout.BundleRoot)
		}
		if location == "" && layout != nil {
			location = strings.TrimSpace(layout.BundleRoot)
		}
		if location != "" {
			fmt.Fprintf(stderr, "%s\n当前路径: %s\n", integrationBundleLaunchBlockedMessage(layout), location)
		} else {
			fmt.Fprintln(stderr, integrationBundleLaunchBlockedMessage(layout))
		}
	}
	return false
}

func resolveIntegrationSandboxCommandPath(mode string) (string, error) {
	mode = sandboxstate.NormalizeMode(mode)
	if mode == "" {
		return "", fmt.Errorf("sandbox mode is required")
	}
	layout := integrationBundleLayout()
	if layout != nil {
		path := filepath.Join(layout.HelpersDir, mode, "CLI_SANDBOX.app", "Contents", "MacOS", "CLI_SANDBOX")
		info, err := os.Stat(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("sandbox helper not found for mode=%s: %s", mode, path)
			}
			return "", fmt.Errorf("sandbox helper stat failed for mode=%s: %w", mode, err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("sandbox helper path is a directory for mode=%s: %s", mode, path)
		}
		return path, nil
	}
	path := filepath.Join(integrationExecutableBaseDir(), "helpers", mode, "CLI_SANDBOX")
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("sandbox helper not found for mode=%s: %s", mode, path)
		}
		return "", fmt.Errorf("sandbox helper stat failed for mode=%s: %w", mode, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("sandbox helper path is a directory for mode=%s: %s", mode, path)
	}
	return path, nil
}

func integrationResourcesDir() string {
	return integrationExecutableBaseDir()
}

func integrationBundledPluginDir() string {
	resourcesDir := strings.TrimSpace(integrationResourcesDir())
	if resourcesDir == "" {
		return ""
	}
	return filepath.Join(resourcesDir, "plugins")
}

func integrationRuntimePluginDir() string {
	runtimeDir := strings.TrimSpace(integrationBundleRuntimeBaseDir())
	if runtimeDir == "" {
		return ""
	}
	return filepath.Join(runtimeDir, "plugins")
}

func integrationExecutableDir() string {
	exe := integrationExecutablePath()
	if exe == "" {
		return ""
	}
	dir := filepath.Dir(exe)
	if abs, err := filepath.Abs(dir); err == nil {
		return abs
	}
	return filepath.Clean(dir)
}

func integrationRuntimeRecordedAppDir(runtimeCfg map[string]string) string {
	if runtimeCfg == nil {
		return ""
	}
	if appDir := strings.TrimSpace(runtimeCfg["app-dir"]); appDir != "" {
		if abs, err := filepath.Abs(appDir); err == nil {
			return abs
		}
		return filepath.Clean(appDir)
	}
	if appPath := strings.TrimSpace(runtimeCfg["app"]); appPath != "" {
		dir := filepath.Dir(appPath)
		if abs, err := filepath.Abs(dir); err == nil {
			return abs
		}
		return filepath.Clean(dir)
	}
	return ""
}

func integrationRuntimeRecordedResourcesDir(runtimeCfg map[string]string) string {
	if runtimeCfg == nil {
		return ""
	}
	if resourcesDir := strings.TrimSpace(runtimeCfg["resources-dir"]); resourcesDir != "" {
		if abs, err := filepath.Abs(resourcesDir); err == nil {
			return abs
		}
		return filepath.Clean(resourcesDir)
	}
	return integrationRuntimeRecordedAppDir(runtimeCfg)
}

func isRelativeToBase(path, base string) bool {
	path = strings.TrimSpace(path)
	base = strings.TrimSpace(base)
	if path == "" || base == "" {
		return false
	}
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func integrationResolveAppPath(path string) string {
	return integrationResolveRuntimePath(path, nil)
}

func integrationResolveRuntimePath(path string, runtimeCfg map[string]string) string {
	return integrationResolveStoredPath(path, integrationAppDir(), integrationRuntimeRecordedAppDir(runtimeCfg))
}

func integrationResolveResourcePath(path string) string {
	return integrationResolveResourceRuntimePath(path, nil)
}

func integrationResolveResourceRuntimePath(path string, runtimeCfg map[string]string) string {
	return integrationResolveStoredPath(path, integrationResourcesDir(), integrationRuntimeRecordedResourcesDir(runtimeCfg))
}

func integrationResolveStoredPath(path, base, recordedBase string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !filepath.IsAbs(path) {
		if base == "" {
			return path
		}
		if abs, err := filepath.Abs(filepath.Join(base, path)); err == nil {
			return abs
		}
		return filepath.Clean(filepath.Join(base, path))
	}
	if recordedBase != "" && base != "" && !samePath(recordedBase, base) && isRelativeToBase(path, recordedBase) {
		rel, err := filepath.Rel(recordedBase, path)
		if err == nil {
			if abs, joinErr := filepath.Abs(filepath.Join(base, rel)); joinErr == nil {
				return abs
			}
			return filepath.Clean(filepath.Join(base, rel))
		}
	}
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return filepath.Clean(path)
}

func integrationRuntimePathArg(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	appDir := integrationAppDir()
	if appDir == "" || !filepath.IsAbs(path) || !isRelativeToBase(path, appDir) {
		return path
	}
	rel, err := filepath.Rel(appDir, path)
	if err != nil || strings.TrimSpace(rel) == "" {
		return path
	}
	return rel
}

func integrationResourcePathArg(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	resourcesDir := integrationResourcesDir()
	if resourcesDir == "" || !filepath.IsAbs(path) || !isRelativeToBase(path, resourcesDir) {
		return path
	}
	rel, err := filepath.Rel(resourcesDir, path)
	if err != nil || strings.TrimSpace(rel) == "" {
		return path
	}
	return rel
}

func integrationShouldIgnoreRuntimeConfig(runtimeCfg map[string]string) bool {
	if runtimeCfg == nil || integrationBundleLayout() == nil {
		return false
	}
	appPath := strings.TrimSpace(runtimeCfg["app"])
	if appPath == "" {
		return false
	}
	lowerApp := strings.ToLower(appPath)
	return strings.HasSuffix(lowerApp, ".test") || strings.Contains(lowerApp, string(filepath.Separator)+"go-build")
}

func samePath(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	cleanA := filepath.Clean(a)
	cleanB := filepath.Clean(b)
	if cleanA == cleanB {
		return true
	}
	absA, errA := filepath.Abs(cleanA)
	absB, errB := filepath.Abs(cleanB)
	if errA == nil && errB == nil {
		return absA == absB
	}
	return false
}

func getAgentOutput(root string, deviceID string, ttl time.Duration) (*AgentOutput, error) {
	output, err := agentcore.GetOutputForApp(root, integrationAppDir(), deviceID, ttl)
	if err != nil {
		return nil, err
	}
	cloned := *output
	hydrateAgentOutput(&cloned, "")
	if err := syncIntegrationKnowledgeOutput(&cloned, root); err != nil {
		return nil, err
	}
	cloned.Plugins = detectPluginKeys()
	return &cloned, nil
}

func getAgentOutputForChat(root string, deviceID string, ttl time.Duration, chatID string) (*AgentOutput, error) {
	output, err := agentcore.GetOutputForAppAndChat(root, integrationAppDir(), deviceID, ttl, chatID)
	if err != nil {
		return nil, err
	}
	cloned := *output
	hydrateAgentOutput(&cloned, chatID)
	if err := applyChatSkillState(&cloned, chatID); err != nil {
		return nil, err
	}
	if err := syncIntegrationKnowledgeOutput(&cloned, root); err != nil {
		return nil, err
	}
	cloned.Plugins = detectPluginKeys()
	return &cloned, nil
}

func integrationKnowledgeRoot(agentDir string) (string, error) {
	resolvedAgentDir, err := resolveIntegrationAgentDir(agentDir)
	if err != nil {
		return "", fmt.Errorf("resolve agent-dir for knowledge: %w", err)
	}
	return filepath.Join(filepath.Dir(resolvedAgentDir), "knowledge"), nil
}

func integrationAppStaticRoot(agentDir, agentID string) (string, error) {
	resolvedAgentDir, err := resolveIntegrationAgentDir(agentDir)
	if err != nil {
		return "", fmt.Errorf("resolve agent-dir for app static: %w", err)
	}
	agentID = strings.TrimSpace(agentID)
	if agentID == "" || agentID == "." || agentID == ".." {
		return "", fmt.Errorf("agentId is required")
	}
	if err := integrationagentarchive.ValidateAgentID(agentID); err != nil {
		return "", fmt.Errorf("invalid agentId: %w", err)
	}
	agentRoot := filepath.Join(resolvedAgentDir, agentID)
	if !ensurePathWithinRoot(resolvedAgentDir, agentRoot) {
		return "", fmt.Errorf("agentId escapes agent-dir")
	}
	return filepath.Join(agentRoot, "app"), nil
}

func integrationKnowledgeDirForAgent(agentDir, agentID string) (string, error) {
	root, err := integrationKnowledgeRoot(agentDir)
	if err != nil {
		return "", err
	}
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return root, nil
	}
	return filepath.Join(root, agentID), nil
}

func integrationKnowledgeLastUpdate(appDir string) int64 {
	db, err := knowledgecore.OpenExistingDB(appDir)
	if err != nil || db == nil {
		return 0
	}
	defer db.Close()
	lastUpdate, err := knowledgecore.GetLastUpdate(db)
	if err != nil {
		return 0
	}
	return lastUpdate
}

func syncIntegrationKnowledgeOutput(output *AgentOutput, agentDir string) error {
	if output == nil {
		return nil
	}
	root, err := integrationKnowledgeRoot(agentDir)
	if err != nil {
		return err
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		output.Knowledge = nil
		return nil
	}
	lastUpdate := integrationKnowledgeLastUpdate(integrationAppDir())
	if output.Knowledge != nil && output.Knowledge.LastUpdate > lastUpdate {
		lastUpdate = output.Knowledge.LastUpdate
	}
	output.Knowledge = &agentcore.Knowledge{
		Path:       root,
		LastUpdate: lastUpdate,
	}
	return nil
}

func readIntegrationSandboxMode(chatID string) string {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return ""
	}
	paths := []string{resolveIntegrationDBPath(), "data"}
	seen := map[string]struct{}{}
	for _, dbPath := range paths {
		dbPath = strings.TrimSpace(dbPath)
		if dbPath == "" {
			continue
		}
		if _, ok := seen[dbPath]; ok {
			continue
		}
		seen[dbPath] = struct{}{}
		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			continue
		}
		state, recorded, err := sandboxstate.Get(db, "", chatID)
		_ = db.Close()
		if err != nil || !recorded {
			continue
		}
		return sandboxstate.NormalizeMode(state.Sandbox)
	}
	return ""
}

func hydrateAgentOutput(output *AgentOutput, chatID string) {
	if output == nil {
		return
	}
	chatID = strings.TrimSpace(chatID)
	seedreamSkill, seedreamEnabled := integrationSeedreamSkillDefinition()
	mode := readIntegrationSandboxMode(chatID)
	for i := range output.Agents {
		output.Agents[i].Skills = integrationApplySeedreamSkill(output.Agents[i].Skills, seedreamSkill, seedreamEnabled)
		output.Agents[i].Sandbox = mode
	}
}

func disabledSkillPathsForChat(chatID string) ([]string, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return nil, nil
	}
	if cronDB != nil {
		return skillstate.ListDisabledPaths(cronDB, chatID)
	}
	db, err := sql.Open("sqlite", resolveIntegrationDBPath())
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return skillstate.ListDisabledPaths(db, chatID)
}

func filterSkillsByDisabledPaths(skills []agentcore.Skill, disabledPaths []string) []agentcore.Skill {
	if len(skills) == 0 || len(disabledPaths) == 0 {
		return skills
	}
	filtered := make([]agentcore.Skill, 0, len(skills))
	for _, skill := range skills {
		if self, inherited := skillstate.DisabledStatus(disabledPaths, skill.Location); self || inherited {
			continue
		}
		filtered = append(filtered, skill)
	}
	return filtered
}

func applyChatSkillState(output *AgentOutput, chatID string) error {
	if output == nil {
		return nil
	}
	disabledPaths, err := disabledSkillPathsForChat(chatID)
	if err != nil || len(disabledPaths) == 0 {
		return err
	}
	for i := range output.Agents {
		output.Agents[i].Skills = filterSkillsByDisabledPaths(output.Agents[i].Skills, disabledPaths)
	}
	return nil
}

func requestChatContext(raw interface{}) (string, string) {
	meta, ok := raw.(map[string]interface{})
	if !ok {
		return "", ""
	}
	agentID, _ := meta["agentId"].(string)
	chatID, _ := meta["chat"].(string)
	return strings.TrimSpace(agentID), strings.TrimSpace(chatID)
}

func cloneStringAnyMap(src map[string]interface{}) map[string]interface{} {
	if len(src) == 0 {
		return nil
	}
	cloned := make(map[string]interface{}, len(src))
	for k, v := range src {
		cloned[k] = v
	}
	return cloned
}

func normalizeConfiguredIntegrationSkillName(name string) string {
	name = strings.TrimSpace(name)
	switch name {
	case "", legacySeedreamSkillName:
		if name == legacySeedreamSkillName {
			return seedreamInternalSkillName
		}
		return ""
	default:
		return name
	}
}

func applySeedreamTokenDefaults(cfg tokenConfig) tokenConfig {
	cfg = normalizeTokenConfig(cfg)
	if cfg.BaseURL == "" {
		cfg.BaseURL = seedreamDefaultURL
	}
	if cfg.ModelMultiOutput == "" {
		cfg.ModelMultiOutput = seedreamDefaultModelMultiOutput
	}
	return cfg
}

func integrationSeedreamTokenConfig() (tokenConfig, bool, error) {
	if cronDB != nil {
		return integrationSeedreamTokenConfigFromDB(cronDB)
	}
	db, err := sql.Open("sqlite", resolveIntegrationDBPath())
	if err != nil {
		return tokenConfig{}, false, err
	}
	defer db.Close()
	return integrationSeedreamTokenConfigFromDB(db)
}

func integrationSeedreamTokenConfigFromDB(db *sql.DB) (tokenConfig, bool, error) {
	cfg, err := lookupTokenConfigByModel(db, seedreamProviderName)
	if err != nil {
		return tokenConfig{}, false, err
	}
	cfg = applySeedreamTokenDefaults(cfg)
	if cfg.Token == "" || cfg.BaseURL == "" || cfg.ModelMultiOutput == "" {
		return cfg, false, nil
	}
	return cfg, true, nil
}

func integrationSeedreamSkillEnabled() bool {
	_, enabled, err := integrationSeedreamTokenConfig()
	return err == nil && enabled
}

func integrationInternalSkillPath(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	return integrationResolveResourcePath(filepath.Join("config", "skills", name, "SKILL.md"))
}

func integrationSeedreamSkillDefinition() (agentcore.Skill, bool) {
	_, enabled, err := integrationSeedreamTokenConfig()
	if err != nil || !enabled {
		return agentcore.Skill{}, false
	}
	return agentcore.Skill{
		Name:        seedreamInternalSkillName,
		Description: seedreamSkillDescription,
		Location:    integrationInternalSkillPath(seedreamInternalSkillName),
	}, true
}

func integrationApplySeedreamSkill(skills []agentcore.Skill, seedream agentcore.Skill, enabled bool) []agentcore.Skill {
	filtered := make([]agentcore.Skill, 0, len(skills)+1)
	for _, skill := range skills {
		if strings.TrimSpace(skill.Name) == seedreamInternalSkillName {
			continue
		}
		filtered = append(filtered, skill)
	}
	if enabled {
		filtered = append(filtered, seedream)
	}
	return filtered
}

func readLiveAgentMedia(workspace string) map[string]interface{} {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(workspace, "config.json"))
	if err != nil {
		return nil
	}
	var raw struct {
		Media map[string]interface{} `json:"media"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	return cloneStringAnyMap(raw.Media)
}

func readLiveAgentKnowledge(workspace string) (string, bool) {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return "", false
	}
	for _, name := range []string{"Knowledge.md", "knowledge.md"} {
		data, err := os.ReadFile(filepath.Join(workspace, name))
		if err != nil {
			continue
		}
		return string(data), true
	}
	return "", false
}

func injectLiveAgentMediaIntoAgentList(metaMap map[string]interface{}) map[string]map[string]interface{} {
	agents, ok := metaMap["agents"].([]interface{})
	if !ok || len(agents) == 0 {
		return nil
	}
	mediaByAgentID := make(map[string]map[string]interface{}, len(agents))
	for _, item := range agents {
		agentMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		workspace, _ := agentMap["workspace"].(string)
		media := readLiveAgentMedia(workspace)
		if len(media) == 0 {
			delete(agentMap, "media")
			continue
		}
		agentMap["media"] = media
		if agentID, _ := agentMap["agentId"].(string); strings.TrimSpace(agentID) != "" {
			mediaByAgentID[strings.TrimSpace(agentID)] = cloneStringAnyMap(media)
		}
	}
	if len(mediaByAgentID) == 0 {
		return nil
	}
	return mediaByAgentID
}

func injectLiveAgentKnowledgeIntoAgentList(metaMap map[string]interface{}) {
	agents, ok := metaMap["agents"].([]interface{})
	if !ok || len(agents) == 0 {
		return
	}
	for _, item := range agents {
		agentMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		workspace, _ := agentMap["workspace"].(string)
		knowledge, exists := readLiveAgentKnowledge(workspace)
		if !exists {
			delete(agentMap, "knowledge")
			continue
		}
		agentMap["knowledge"] = knowledge
	}
}

func buildCLIRequestMetadataMap(metadata *AgentOutput) map[string]interface{} {
	if metadata == nil {
		return nil
	}
	metaBytes, _ := json.Marshal(metadata)
	var metaMap map[string]interface{}
	_ = json.Unmarshal(metaBytes, &metaMap)
	injectLiveAgentMediaIntoAgentList(metaMap)
	return metaMap
}

func resolveCurrentPageSessionChatID() string {
	if cronDB == nil {
		return ""
	}
	var chatID string
	if err := cronDB.QueryRow(`SELECT chat_id FROM chat_log WHERE chat_type = ? AND chat_id != '' ORDER BY created_at DESC, id DESC LIMIT 1`, chatTypePageSession).Scan(&chatID); err != nil {
		return ""
	}
	return strings.TrimSpace(chatID)
}

func lookupLastSSEResponseTimestamp(chatID string) (int64, bool) {
	if cronDB == nil {
		return 0, false
	}
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return 0, false
	}
	var createdAt string
	if err := cronDB.QueryRow(`SELECT created_at FROM agent_message_log WHERE chat_id = ? AND log_type = ? ORDER BY created_at DESC, id DESC LIMIT 1`, chatID, logTypeChatCompletionResponse).Scan(&createdAt); err != nil {
		return 0, false
	}
	parsed, err := parseRoundLogTimeValue(createdAt)
	if err != nil {
		return 0, false
	}
	return parsed.UnixMilli(), true
}

func injectLastResponseMetadata(metaMap map[string]interface{}, chatID string) {
	if metaMap == nil {
		return
	}
	delete(metaMap, "lastResponse")
	if lastResponse, ok := lookupLastSSEResponseTimestamp(chatID); ok {
		metaMap["lastResponse"] = lastResponse
	}
}

func lookupSandboxPath(chatID string) (string, bool) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return "", false
	}
	db := cronDB
	shouldClose := false
	if db == nil {
		dbPath := resolveIntegrationDBPath()
		if _, err := os.Stat(dbPath); err != nil {
			return "", false
		}
		var err error
		db, err = sql.Open("sqlite", dbPath)
		if err != nil {
			return "", false
		}
		shouldClose = true
	}
	if shouldClose {
		defer db.Close()
	}
	state, recorded, err := sandboxstate.Get(db, "", chatID)
	if err != nil || !recorded {
		return "", false
	}
	allowedDir := strings.TrimSpace(state.AllowedDir)
	if allowedDir == "" {
		return "", false
	}
	return allowedDir, true
}

func injectSandboxPathMetadata(metaMap map[string]interface{}, chatID string) {
	if metaMap == nil {
		return
	}
	delete(metaMap, "sandbox_path")
	if allowedDir, ok := lookupSandboxPath(chatID); ok {
		metaMap["sandbox_path"] = allowedDir
	}
}

func lookupForwardedPluginsDir() (string, bool) {
	pluginDir := strings.TrimSpace(integrationRuntimePluginDir())
	if pluginDir == "" {
		return "", false
	}
	if !filepath.IsAbs(pluginDir) {
		absPath, err := filepath.Abs(pluginDir)
		if err == nil {
			pluginDir = absPath
		}
	}
	pluginDir = filepath.Clean(strings.TrimSpace(pluginDir))
	if pluginDir == "" {
		return "", false
	}
	return pluginDir, true
}

func injectPluginsDirMetadata(metaMap map[string]interface{}) {
	if metaMap == nil {
		return
	}
	delete(metaMap, "plugins_dir")
	if pluginDir, ok := lookupForwardedPluginsDir(); ok {
		metaMap["plugins_dir"] = pluginDir
	}
}

func pruneForwardedKnowledgeCommit(metaMap map[string]interface{}) {
	if metaMap == nil {
		return
	}
	knowledge, ok := metaMap["knowledge"].(map[string]interface{})
	if !ok || knowledge == nil {
		return
	}
	delete(knowledge, "knowledgeCommit")
}

func integrationDefaultAgentDir() string {
	if value := strings.TrimSpace(os.Getenv("AGENT_DIR")); value != "" {
		return value
	}
	if runtimeBase := strings.TrimSpace(integrationManagedRuntimeBaseDir()); runtimeBase != "" {
		return filepath.Join(runtimeBase, "agent")
	}
	return "agent"
}

func detectPluginRuntimes() []agentcore.PluginRuntime {
	plugins := make([]agentcore.PluginRuntime, 0)
	if svc, err := newIntegrationConnectService(integrationDefaultAgentDir()); err == nil {
		defer svc.Close()
		if items, err := listLocalPluginMeta(svc); err == nil {
			plugins = integrationPluginRuntimesFromLocalMeta(items)
		}
	}
	return mergeIntegrationAssociatedPluginRuntimes(plugins)
}

func integrationPluginRuntimesFromLocalMeta(items []connectsvc.PluginMetaInfo) []agentcore.PluginRuntime {
	pluginDir, dirErr := localPluginDirResolver()
	seen := make(map[string]struct{}, len(items))
	plugins := make([]agentcore.PluginRuntime, 0, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		lookupKey := strings.ToLower(key)
		if _, exists := seen[lookupKey]; exists {
			continue
		}
		callback := strings.TrimSpace(item.Callback)
		if dirErr == nil && strings.TrimSpace(pluginDir) != "" {
			if callback == "" {
				callback = filepath.Join(pluginDir, key)
			} else if !filepath.IsAbs(callback) {
				// Meta callbacks such as ./browser are relative to the local
				// plugin directory, rather than Integration's current directory.
				callback = filepath.Join(pluginDir, callback)
			}
		}
		plugins = append(plugins, agentcore.PluginRuntime{Key: key, Callback: callback})
		seen[lookupKey] = struct{}{}
	}
	return plugins
}

func mergeIntegrationAssociatedPluginRuntimes(plugins []agentcore.PluginRuntime) []agentcore.PluginRuntime {
	associated, err := readIntegrationAssociatedPlugins()
	if err != nil || len(associated) == 0 {
		return plugins
	}

	seen := make(map[string]struct{}, len(plugins)+len(associated))
	for _, plugin := range plugins {
		if key := strings.ToLower(strings.TrimSpace(plugin.Key)); key != "" {
			seen[key] = struct{}{}
		}
	}
	pluginDir, dirErr := localPluginDirResolver()
	for _, key := range associated {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			continue
		}
		lookupKey := strings.ToLower(normalizedKey)
		if _, exists := seen[lookupKey]; exists {
			continue
		}
		callback := ""
		if dirErr == nil && strings.TrimSpace(pluginDir) != "" {
			callback = filepath.Join(pluginDir, normalizedKey)
		}
		plugins = append(plugins, agentcore.PluginRuntime{Key: normalizedKey, Callback: callback})
		seen[lookupKey] = struct{}{}
	}
	return plugins
}

func prepareIntegrationRuntimeLayout() error {
	runtimeDir, err := prepareIntegrationRuntimeBaseDir()
	if err != nil {
		return err
	}
	pluginDir, err := resolveIntegrationRuntimePluginDirFromRuntimeBase(runtimeDir)
	if err != nil {
		if runtimeDir == "" {
			return nil
		}
		return err
	}
	bundledPluginDir := strings.TrimSpace(integrationBundledPluginDir())
	if bundledPluginDir != "" {
		info, statErr := os.Stat(bundledPluginDir)
		if statErr == nil && info != nil && info.IsDir() {
			if err := syncBundledPluginsToRuntime(bundledPluginDir, pluginDir); err != nil {
				return fmt.Errorf("sync bundled plugins: %w", err)
			}
		} else if statErr != nil && !os.IsNotExist(statErr) {
			return fmt.Errorf("stat bundled plugins: %w", statErr)
		}
	}
	if err := os.Setenv(integrationPluginDirEnv, pluginDir); err != nil {
		return fmt.Errorf("export plugin dir: %w", err)
	}
	return nil
}

func shouldSyncBundledPluginEntry(name string, info os.FileInfo) bool {
	if info == nil || info.IsDir() {
		return false
	}
	name = strings.TrimSpace(name)
	if name == "" || strings.HasPrefix(name, ".") {
		return false
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case "":
		return info.Mode()&0o111 != 0
	case ".py", ".js", ".go":
		return true
	default:
		return false
	}
}

func syncBundledPluginsToRuntime(srcDir, dstDir string) error {
	plan, err := buildIntegrationBundledPluginSyncPlan(srcDir, dstDir)
	if err != nil {
		return err
	}
	return applyIntegrationBundledPluginSyncPlan(plan)
}

func copyReleaseAssetWithPermissions(srcPath, dstPath string, mode os.FileMode) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}
	if existing, err := os.ReadFile(dstPath); err == nil && bytes.Equal(existing, data) {
		if info, statErr := os.Stat(dstPath); statErr == nil && info.Mode().Perm() == mode.Perm() {
			return nil
		}
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(dstPath), filepath.Base(dstPath)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Chmod(mode.Perm()); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, dstPath); err != nil {
		if runtime.GOOS == "windows" {
			if removeErr := os.Remove(dstPath); removeErr != nil && !os.IsNotExist(removeErr) {
				return removeErr
			}
			if retryErr := os.Rename(tmpPath, dstPath); retryErr == nil {
				return nil
			}
		}
		return err
	}
	return nil
}

func detectPluginKeys() []string {
	return mergeDefaultRemotePluginKey(
		agentcore.DetectRunningPluginKeys("plugins", detectPluginRuntimes()),
	)
}

// mergeDefaultRemotePluginKey adds the built-in remote capability to the
// runtime-discovered plugin list. remote is advertised even before its
// associated process has created a PID file, while all other entries remain
// strictly realtime discoveries.
func mergeDefaultRemotePluginKey(running []string) []string {
	plugins := make([]string, 0, len(running)+1)
	seen := make(map[string]struct{}, len(running)+1)
	for _, plugin := range running {
		plugin = strings.TrimSpace(plugin)
		if plugin == "" {
			continue
		}
		lookupKey := strings.ToLower(plugin)
		if _, exists := seen[lookupKey]; exists {
			continue
		}
		seen[lookupKey] = struct{}{}
		if lookupKey == "remote" {
			plugins = append(plugins, "remote")
			continue
		}
		plugins = append(plugins, plugin)
	}
	if _, exists := seen["remote"]; !exists {
		plugins = append(plugins, "remote")
	}
	sort.Strings(plugins)
	return plugins
}

func pluginStarted(key, callback string) bool {
	return agentcore.PluginStarted("plugins", key, callback)
}

func pluginPIDFiles(key, callback string) []string {
	return agentcore.PluginPIDFiles("plugins", key, callback)
}

// ═══════════════════════════════════════════════════════════════════════════
// cli-get: heartbeat, execute, publish
// ═══════════════════════════════════════════════════════════════════════════

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GetRequest struct {
	Messages []Message    `json:"messages"`
	Metadata *AgentOutput `json:"metadata"`
}

type PubRequest struct {
	Model    string       `json:"model"`
	Messages []Message    `json:"messages"`
	Metadata *AgentOutput `json:"metadata"`
}

type ResponsePayload struct {
	Code    int `json:"code"`
	Choices []struct {
		Message struct {
			Role    string      `json:"role"`
			Content interface{} `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type TaskContent struct {
	Timeout int    `json:"timeout"`
	Suffix  string `json:"suffix"`
	Type    string `json:"type"`
	Ddl     int64  `json:"ddl"`
	Tid     string `json:"tid"`
	Cmd     string `json:"cmd"`
	AgentId string `json:"agentId"`
	Chat    string `json:"chat"`
	SubOps  struct {
		Exempted bool     `json:"exempted"`
		App      []string `json:"app"`
		W        []string `json:"w"`
		R        []string `json:"r"`
		Apps     string   `json:"apps"`
	} `json:"subOps"`
}

type ResultPayload struct {
	Status  int                        `json:"status"`
	AgentId string                     `json:"agentId,omitempty"`
	Suffix  string                     `json:"suffix"`
	Chat    string                     `json:"chat,omitempty"`
	Type    string                     `json:"type"`
	Cmd     string                     `json:"cmd"`
	Tid     string                     `json:"tid"`
	Insert  []messageInsertPublishItem `json:"insert,omitempty"`
}

type queuedCliGetTask struct {
	task     TaskContent
	metadata *AgentOutput
}

type queuedCliGetPublish struct {
	result   *ResultPayload
	metadata *AgentOutput
}

const (
	logTypeChatCompletionRequest  = 0
	logTypeChatCompletionResponse = 1
	logTypeCLIGet                 = 2
	logTypeCLIPub                 = 3
)

type eventLogEntry struct {
	ID        int
	AgentID   string
	ChatID    string
	Content   string
	LogType   int
	CreatedAt string
}

func integrationHTTPDebugTimestamp() string {
	return time.Now().Format(roundLogStorageTimeLayout)
}

func integrationHTTPDebugEnabled(cfg *Config) bool {
	return cfg != nil && cfg.HTTPDebug
}

func integrationHTTPDebugLog(cfg *Config, format string, args ...interface{}) {
	if !integrationHTTPDebugEnabled(cfg) {
		return
	}
	log.Printf("cli-get debug: "+format, args...)
}

func integrationHTTPTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if os.IsTimeout(err) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func integrationMetadataPort(cfg *Config) int {
	if cfg != nil && cfg.Port > 0 {
		return cfg.Port
	}
	return integrationServicePort
}

func heartbeat(client *http.Client, host string, metadata *AgentOutput, cfg *Config) (*TaskContent, error) {
	metaMap := buildCLIRequestMetadataMap(metadata)
	metaMap["port"] = integrationMetadataPort(cfg)
	currentChatID := resolveCurrentPageSessionChatID()
	injectLastResponseMetadata(metaMap, currentChatID)
	injectSandboxPathMetadata(metaMap, currentChatID)
	injectPluginsDirMetadata(metaMap)
	injectLiveAgentKnowledgeIntoAgentList(metaMap)
	pruneForwardedKnowledgeCommit(metaMap)
	reqBody, _ := json.Marshal(map[string]interface{}{
		"messages": []Message{{Role: "user", Content: ""}},
		"metadata": metaMap,
	})
	url := strings.TrimRight(host, "/") + "/cli/get"
	startedAt := time.Now()
	resp, err := client.Post(url, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		if integrationHTTPTimeoutError(err) {
			httpTimeout := 0
			httpConnectTimeout := 0
			httpSocketTimeout := 0
			if cfg != nil {
				httpTimeout = cfg.HTTPTimeout
				httpConnectTimeout = cfg.HTTPConnTimeout
				httpSocketTimeout = cfg.HTTPSocketTimeout
			}
			integrationHTTPDebugLog(cfg,
				"/cli/get timeout time=%s elapsed_ms=%d host=%s http_timeout=%d http_connect_timeout=%d http_socket_timeout=%d err=%v",
				integrationHTTPDebugTimestamp(),
				time.Since(startedAt).Milliseconds(),
				host,
				httpTimeout,
				httpConnectTimeout,
				httpSocketTimeout,
				err,
			)
		}
		return nil, fmt.Errorf("heartbeat failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("heartbeat HTTP %d", resp.StatusCode)
	}
	var rp ResponsePayload
	if err := json.Unmarshal(body, &rp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if rp.Code != 200 {
		return nil, fmt.Errorf("heartbeat code %d", rp.Code)
	}
	if len(rp.Choices) == 0 || rp.Choices[0].Message.Content == nil {
		return nil, nil
	}
	contentStr, ok := rp.Choices[0].Message.Content.(string)
	if !ok || contentStr == "" {
		return nil, nil
	}
	var task TaskContent
	if err := json.Unmarshal([]byte(contentStr), &task); err != nil {
		return nil, fmt.Errorf("parse task: %w", err)
	}
	appendEventLogDB(strings.TrimSpace(task.AgentId), strings.TrimSpace(task.Chat), contentStr, logTypeCLIGet, "")
	integrationHTTPDebugLog(cfg, "/cli/get task time=%s host=%s agentId=%s chat=%s tid=%s payload=%s",
		integrationHTTPDebugTimestamp(),
		host,
		strings.TrimSpace(task.AgentId),
		strings.TrimSpace(task.Chat),
		strings.TrimSpace(task.Tid),
		contentStr,
	)
	return &task, nil
}

func executeTask(task *TaskContent, terminal string) *ResultPayload {
	sandboxMode := ""
	sandboxAllowedDir := ""
	var err error
	if task != nil && !integrationTaskShouldBypassSandbox(task) {
		var sandboxState sandboxstate.State
		var recorded bool
		sandboxState, recorded, err = sandboxStateForSession(task.Chat)
		if recorded {
			sandboxMode = sandboxstate.NormalizeMode(sandboxState.Sandbox)
			sandboxAllowedDir = sandboxState.AllowedDir
		}
	}
	if err != nil {
		return &ResultPayload{
			Status:  1,
			AgentId: task.AgentId,
			Suffix:  task.Suffix,
			Chat:    task.Chat,
			Type:    task.Type,
			Cmd:     sharedutil.GzipBase64String("查询沙盒状态失败"),
			Tid:     task.Tid,
		}
	}

	changeCliGetRunningWorkers(1)
	defer changeCliGetRunningWorkers(-1)

	var activeKey string
	var status int
	var output string
	if sandboxMode != "" {
		status, output = executeSandboxCommandLogged("cli-get", task.AgentId, task.Chat, task.Tid, sandboxMode, sandboxAllowedDir, terminal, task.Cmd, int64(task.Timeout), func(ac *activeCmd) {
			ac.agentID = task.AgentId
			ac.chatID = task.Chat
			ac.tid = task.Tid
			activeKey = registerActiveCmd(ac)
		})
	} else {
		status, output = executeShellCommand(terminal, task.Cmd, int64(task.Timeout), func(ac *activeCmd) {
			ac.agentID = task.AgentId
			ac.chatID = task.Chat
			ac.tid = task.Tid
			activeKey = registerActiveCmd(ac)
		})
	}
	if activeKey != "" {
		unregisterActiveCmd(activeKey)
	}
	return &ResultPayload{
		Status:  status,
		AgentId: task.AgentId,
		Suffix:  task.Suffix,
		Chat:    task.Chat,
		Type:    task.Type,
		Cmd:     sharedutil.GzipBase64String(output),
		Tid:     task.Tid,
	}
}

func integrationTaskShouldBypassSandbox(task *TaskContent) bool {
	if task == nil {
		return true
	}
	if task.SubOps.Exempted {
		return true
	}
	if integrationTaskIsInternalTokenCommand(task) {
		return true
	}
	return integrationTaskUsesInternalRuntimePaths(task)
}

func integrationTaskIsInternalTokenCommand(task *TaskContent) bool {
	if task == nil {
		return false
	}
	cmd := strings.TrimSpace(task.Cmd)
	if cmd == "" {
		return false
	}
	exe := strings.TrimSpace(integrationExecutablePath())
	if exe == "" {
		return false
	}
	for _, candidate := range []string{
		exe + " token",
		"'" + exe + "' token",
		"\"" + exe + "\" token",
	} {
		if strings.Contains(cmd, candidate) {
			return true
		}
	}
	return false
}

func nextHeartbeatBackoff(current, base, max time.Duration) time.Duration {
	if base <= 0 {
		base = time.Second
	}
	if max < base {
		max = base
	}
	if current < base {
		return base
	}
	next := current * 2
	if next < current || next > max {
		return max
	}
	return next
}

func shouldSleepAfterHeartbeat(task *TaskContent, err error) bool {
	return err != nil
}

func integrationTaskUsesInternalRuntimePaths(task *TaskContent) bool {
	if task == nil {
		return false
	}
	runtimeBase := strings.TrimSpace(integrationBundleRuntimeBaseDir())
	if runtimeBase == "" {
		return false
	}
	paths := make([]string, 0, len(task.SubOps.R)+len(task.SubOps.W))
	paths = append(paths, task.SubOps.R...)
	paths = append(paths, task.SubOps.W...)
	internalCount := 0
	for _, raw := range paths {
		path := normalizeIntegrationTaskPath(raw)
		if path == "" {
			continue
		}
		if !filepath.IsAbs(path) {
			return false
		}
		if !isRelativeToBase(path, runtimeBase) {
			return false
		}
		internalCount++
	}
	return internalCount > 0
}

func normalizeIntegrationTaskPath(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) >= 2 {
		first := raw[0]
		last := raw[len(raw)-1]
		if (first == '\'' && last == '\'') || (first == '"' && last == '"') {
			raw = strings.TrimSpace(raw[1 : len(raw)-1])
		}
	}
	if raw == "" {
		return ""
	}
	if abs, err := filepath.Abs(raw); err == nil {
		raw = abs
	}
	return filepath.Clean(raw)
}

func sandboxModeRequiresDirectoryPick(mode string) bool {
	switch sandboxstate.NormalizeMode(mode) {
	case sandboxstate.ModeFilePick, sandboxstate.ModeFilePickNet:
		return true
	default:
		return false
	}
}

func primeIntegrationSandboxMode(mode string) (string, error) {
	mode = sandboxstate.NormalizeMode(mode)
	if !sandboxModeRequiresDirectoryPick(mode) {
		return "", nil
	}
	helperPath, err := integrationSandboxCommandPathFn(mode)
	if err != nil {
		return "", err
	}
	log.Printf("sandbox prime start mode=%s helper=%s", mode, helperPath)
	cmd := exec.Command(helperPath)
	cmd.Env = append(os.Environ(), "CLI_SANDBOX_FORCE_PICK=1")
	output, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if err != nil {
		if text != "" {
			log.Printf("sandbox prime failed mode=%s helper=%s err=%v output=%q", mode, helperPath, err, text)
			return "", errors.New(text)
		}
		log.Printf("sandbox prime failed mode=%s helper=%s err=%v", mode, helperPath, err)
		return "", err
	}
	log.Printf("sandbox prime done mode=%s helper=%s", mode, helperPath)
	return text, nil
}

func primeIntegrationSandboxDirectory(mode, dir string) (string, error) {
	mode = sandboxstate.NormalizeMode(mode)
	dir = strings.TrimSpace(dir)
	if !sandboxModeRequiresDirectoryPick(mode) {
		if dir == "" {
			return "", nil
		}
		if mode == "" {
			return "", fmt.Errorf("sandbox off does not support manual directory whitelist")
		}
		return "", fmt.Errorf("sandbox mode %s does not support manual directory whitelist", mode)
	}
	if dir == "" {
		return integrationPrimeSandboxModeFn(mode)
	}
	helperPath, err := integrationSandboxCommandPathFn(mode)
	if err != nil {
		return "", err
	}
	log.Printf("sandbox prime set-dir start mode=%s helper=%s dir=%s", mode, helperPath, dir)
	cmd := exec.Command(helperPath, "--allowed-dir", dir)
	output, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if err != nil {
		if text != "" {
			log.Printf("sandbox prime set-dir failed mode=%s helper=%s dir=%s err=%v output=%q", mode, helperPath, dir, err, text)
			return "", errors.New(text)
		}
		log.Printf("sandbox prime set-dir failed mode=%s helper=%s dir=%s err=%v", mode, helperPath, dir, err)
		return "", err
	}
	log.Printf("sandbox prime set-dir done mode=%s helper=%s dir=%s", mode, helperPath, text)
	if text == "" {
		text = dir
	}
	return text, nil
}

func publishResult(client *http.Client, host string, result *ResultPayload, metadata *AgentOutput, cfg *Config) error {
	payloadResult := result
	pendingInsertItems := result.Insert
	if len(pendingInsertItems) == 0 {
		items, err := loadPendingMessageInsertPublishItems(strings.TrimSpace(result.Chat), messageInsertPublishLimit)
		if err != nil {
			log.Printf("message_insert load pending failed chatId=%s: %v", strings.TrimSpace(result.Chat), err)
		} else if len(items) > 0 {
			result.Insert = items
			payloadResult = result
			pendingInsertItems = items
		}
	}
	resultJSON, _ := json.Marshal(payloadResult)
	rawResult := normalizeCLIPubLogContent(string(resultJSON))
	appendEventLogDB(strings.TrimSpace(result.AgentId), strings.TrimSpace(result.Chat), rawResult, logTypeCLIPub, "")
	integrationHTTPDebugLog(cfg, "/cli/pub result time=%s host=%s agentId=%s chat=%s tid=%s status=%d result=%s",
		integrationHTTPDebugTimestamp(),
		host,
		strings.TrimSpace(result.AgentId),
		strings.TrimSpace(result.Chat),
		strings.TrimSpace(result.Tid),
		result.Status,
		rawResult,
	)
	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":    "",
		"messages": []Message{{Role: "user", Content: string(resultJSON)}},
		"metadata": buildCLIRequestMetadataMap(metadata),
	})
	url := strings.TrimRight(host, "/") + "/cli/pub"
	resp, err := client.Post(url, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("publish failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("publish HTTP %d", resp.StatusCode)
	}
	var rp ResponsePayload
	if err := json.Unmarshal(body, &rp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	if rp.Code != 200 {
		return fmt.Errorf("publish code %d", rp.Code)
	}
	if len(pendingInsertItems) > 0 {
		if err := markPublishedMessageInsertPublishItems(strings.TrimSpace(result.Chat), pendingInsertItems); err != nil {
			log.Printf("message_insert mark published failed chatId=%s tids=%d: %v", strings.TrimSpace(result.Chat), len(pendingInsertItems), err)
		} else {
			log.Printf(
				"message_insert mark published success chatId=%s reportedAt=%s tids=%s",
				strings.TrimSpace(result.Chat),
				time.Now().Format(time.RFC3339),
				strings.Join(messageInsertPublishItemTIDs(pendingInsertItems), ","),
			)
		}
	}
	markConfirmedMessageInsertUploads(strings.TrimSpace(result.Chat), string(body))
	return nil
}

func messageInsertPublishItemTIDs(items []messageInsertPublishItem) []string {
	tids := make([]string, 0, len(items))
	for _, item := range items {
		if tid := strings.TrimSpace(item.Tid); tid != "" {
			tids = append(tids, tid)
		}
	}
	return tids
}

func isQueuedCliGetTaskExpired(task *TaskContent, now time.Time) bool {
	if task == nil || task.Ddl <= 0 {
		return false
	}
	return now.UnixMilli() > task.Ddl
}

func publishResultWithRetry(ctx context.Context, client *http.Client, hostFn func() string, result *ResultPayload, metadata *AgentOutput, retryInterval time.Duration, retryTimes int, cfg *Config) error {
	for attempt := 0; ; attempt++ {
		err := publishResult(client, hostFn(), result, metadata, cfg)
		if err == nil {
			if attempt > 0 {
				log.Printf("cli-get: publish retry success tid=%s retries=%d", strings.TrimSpace(result.Tid), attempt)
			}
			return nil
		}
		if attempt >= retryTimes {
			log.Printf("cli-get: publish give up tid=%s retries=%d err=%v", strings.TrimSpace(result.Tid), attempt, err)
			return err
		}
		nextRetry := attempt + 1
		log.Printf("cli-get: publish retry scheduled tid=%s retry=%d max=%d wait_ms=%d err=%v",
			strings.TrimSpace(result.Tid), nextRetry, retryTimes, retryInterval.Milliseconds(), err)
		if !sleepContext(ctx, retryInterval) {
			return ctx.Err()
		}
	}
}

func runCliGetExecuteWorker(ctx context.Context, taskQueue <-chan queuedCliGetTask, publishQueue chan<- queuedCliGetPublish, terminal string, cfg *Config) {
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-taskQueue:
			task := item.task
			now := time.Now()
			if isQueuedCliGetTaskExpired(&task, now) {
				log.Printf("cli-get: drop expired task agentId=%s chat=%s tid=%s ddl=%d now=%d cmd=%q",
					strings.TrimSpace(task.AgentId),
					strings.TrimSpace(task.Chat),
					strings.TrimSpace(task.Tid),
					task.Ddl,
					now.UnixMilli(),
					task.Cmd,
				)
				continue
			}

			receivedAt := now.Format("2006-01-02T15:04:05")
			var cmdLogID int64
			if cronDB != nil && task.AgentId != "" && task.Chat != "" && task.Cmd != "" {
				res, err := cronDB.Exec(`INSERT INTO cmd_log (agent_id, chat_id, tid, cmd, received_at) VALUES (?,?,?,?,?)`,
					task.AgentId, task.Chat, task.Tid, task.Cmd, receivedAt)
				if err == nil {
					cmdLogID, _ = res.LastInsertId()
				}
			}

			term := terminal
			if term == "" {
				term = detectTerminal()
			}
			if term == "" {
				log.Printf("cli-get: no SHELL available for tid=%s", task.Tid)
				continue
			}

			result := executeTask(&task, term)
			if cronDB != nil && cmdLogID > 0 && result != nil {
				completedAt := time.Now().Format("2006-01-02T15:04:05")
				cronDB.Exec(`UPDATE cmd_log SET result = ?, status = ?, completed_at = ? WHERE id = ?`,
					result.Cmd, result.Status, completedAt, cmdLogID)
			}

			select {
			case <-ctx.Done():
				return
			case publishQueue <- queuedCliGetPublish{result: result, metadata: item.metadata}:
				recordCliGetRuntimeEvent(cliGetRuntimeSnapshot{PendingPublishes: 1})
				if integrationHTTPDebugEnabled(cfg) {
					log.Printf("cli-get: publish queued tid=%s pending=%d", strings.TrimSpace(task.Tid), len(publishQueue))
				}
			}
		}
	}
}

func runCliGetPublishWorker(ctx context.Context, client *http.Client, publishQueue <-chan queuedCliGetPublish, retryInterval time.Duration, retryTimes int, cfg *Config) {
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-publishQueue:
			if item.result == nil {
				continue
			}
			if err := publishResultWithRetry(ctx, client, cfg.currentHost, item.result, item.metadata, retryInterval, retryTimes, cfg); err != nil && ctx.Err() == nil {
				log.Printf("cli-get: publish error tid=%s: %v", strings.TrimSpace(item.result.Tid), err)
			}
		}
	}
}

// startCliGet runs the cli-get heartbeat loop as a background goroutine.
func startCliGet(ctx context.Context, cfg *Config) {
	terminal := detectTerminal()
	if terminal == "" {
		log.Println("cli-get: cannot detect SHELL, disabled")
		return
	}

	client := createCliGetHTTPClient(cfg)
	if cfg.Thread <= 0 {
		log.Printf("cli-get: invalid thread=%d", cfg.Thread)
		return
	}
	if cfg.Queue <= 0 {
		log.Printf("cli-get: invalid queue=%d", cfg.Queue)
		return
	}
	if cfg.RetryIntervalMs < 0 {
		log.Printf("cli-get: invalid retry_interval=%d", cfg.RetryIntervalMs)
		return
	}
	if cfg.RetryTimes < 0 {
		log.Printf("cli-get: invalid retry_times=%d", cfg.RetryTimes)
		return
	}

	agentTTL := time.Duration(cfg.AgentCacheMs) * time.Millisecond
	sleepDur := time.Duration(cfg.SleepMs) * time.Millisecond
	retryInterval := time.Duration(cfg.RetryIntervalMs) * time.Millisecond
	heartbeatBackoff := sleepDur
	maxHeartbeatBackoff := 15 * time.Second
	taskQueue := make(chan queuedCliGetTask, cfg.Queue)
	publishQueue := make(chan queuedCliGetPublish, cfg.Queue)
	setCliGetRuntimeQueues(taskQueue, publishQueue)
	setCliGetRunningWorkers(0)

	for i := 0; i < cfg.Thread; i++ {
		go runCliGetExecuteWorker(ctx, taskQueue, publishQueue, terminal, cfg)
		go runCliGetPublishWorker(ctx, client, publishQueue, retryInterval, cfg.RetryTimes, cfg)
	}

	log.Printf("cli-get: started (host=%s, threads=%d, queue=%d, sleep=%dms, retry_interval=%dms, retry_times=%d)",
		cfg.currentHost(), cfg.Thread, cfg.Queue, cfg.SleepMs, cfg.RetryIntervalMs, cfg.RetryTimes)

	go func() {
		defer closeHTTPClientIdleConnections(client)
		defer resetCliGetRuntimeStatus()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if len(taskQueue) >= cap(taskQueue) {
				log.Printf("cli-get: task queue full pending=%d capacity=%d, skip /cli/get", len(taskQueue), cap(taskQueue))
				if !sleepContext(ctx, sleepDur) {
					return
				}
				continue
			}
			currentChatID := resolveCurrentPageSessionChatID()
			var (
				metadata *AgentOutput
				err      error
			)
			if currentChatID != "" {
				metadata, err = getAgentOutputForChat(cfg.AgentDir, cfg.effectiveDeviceID(), agentTTL, currentChatID)
			} else {
				metadata, err = getAgentOutput(cfg.AgentDir, cfg.effectiveDeviceID(), agentTTL)
			}
			if err != nil {
				log.Printf("cli-get: agent scan error: %v", err)
				if !sleepContext(ctx, sleepDur) {
					return
				}
				continue
			}
			host := cfg.currentHost()
			task, err := heartbeat(client, host, metadata, cfg)
			if shouldSleepAfterHeartbeat(task, err) {
				errText := formatCliGetHeartbeatError(host, err)
				log.Printf("%s", errText)
				recordHeartbeat(1, errText) // failure
				if !sleepContext(ctx, heartbeatBackoff) {
					return
				}
				heartbeatBackoff = nextHeartbeatBackoff(heartbeatBackoff, sleepDur, maxHeartbeatBackoff)
				continue
			}
			heartbeatBackoff = sleepDur
			if task == nil {
				recordHeartbeat(0, "") // success, no task
				continue
			}
			recordHeartbeat(2, "") // success, has task
			t := *task
			select {
			case <-ctx.Done():
				return
			case taskQueue <- queuedCliGetTask{task: t, metadata: metadata}:
				recordCliGetRuntimeEvent(cliGetRuntimeSnapshot{PendingTasks: 1})
				if integrationHTTPDebugEnabled(cfg) {
					log.Printf("cli-get: task queued tid=%s pending=%d capacity=%d", strings.TrimSpace(t.Tid), len(taskQueue), cap(taskQueue))
				}
			}
		}
	}()
}

func createCliGetHTTPClient(cfg *Config) *http.Client {
	return http11client.NewClient(http11client.Options{
		Timeout:               time.Duration(cfg.HTTPTimeout) * time.Millisecond,
		ConnectTimeout:        time.Duration(cfg.HTTPConnTimeout) * time.Millisecond,
		KeepAlive:             60 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       time.Duration(cfg.IdleTimeout) * time.Second,
		ResponseHeaderTimeout: time.Duration(cfg.HTTPSocketTimeout) * time.Millisecond,
		TLSHandshakeTimeout:   10 * time.Second,
	})
}

func closeHTTPClientIdleConnections(client *http.Client) {
	if client == nil || client.Transport == nil {
		return
	}
	if transport, ok := client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
}

func sleepContext(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		select {
		case <-ctx.Done():
			return false
		default:
			return true
		}
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func normalizeCLIPubLogContent(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
		return content
	}
	var payload struct {
		Cmd string `json:"cmd"`
	}
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return content
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(payload.Cmd))
	if err != nil {
		return content
	}
	gzr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return content
	}
	defer gzr.Close()
	decoded, err := io.ReadAll(gzr)
	if err != nil {
		return content
	}
	return string(decoded)
}

func formatSandboxLogValue(value string, limit int) string {
	normalized := strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(value), "\r", "\\r"), "\n", "\\n")
	if limit > 0 && len(normalized) > limit {
		return normalized[:limit] + "...(truncated)"
	}
	return normalized
}

func logSandboxExecutionStart(scope, agentID, chatID, tid, helperPath, shell, rawCmd string, timeoutMs int64) {
	log.Printf("%s: sandbox exec start agentId=%s chatId=%s tid=%s helper=%s shell=%s timeoutMs=%d cmd=%q",
		scope,
		strings.TrimSpace(agentID),
		strings.TrimSpace(chatID),
		strings.TrimSpace(tid),
		strings.TrimSpace(helperPath),
		strings.TrimSpace(shell),
		timeoutMs,
		rawCmd,
	)
}

func logSandboxExecutionDone(scope, agentID, chatID, tid string, status int, duration time.Duration, output string) {
	log.Printf("%s: sandbox exec done agentId=%s chatId=%s tid=%s status=%d durationMs=%d output=%s",
		scope,
		strings.TrimSpace(agentID),
		strings.TrimSpace(chatID),
		strings.TrimSpace(tid),
		status,
		duration.Milliseconds(),
		formatSandboxLogValue(output, 600),
	)
}

func executeExternalCommand(binary string, args []string, rawCmd string, timeoutMs int64, onStart func(*activeCmd)) (int, string) {
	timeout := time.Duration(timeoutMs) * time.Millisecond
	if timeoutMs <= 0 {
		timeout = sharedutil.DefaultAPICmdTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	c := exec.CommandContext(ctx, binary, args...)
	c.Env = os.Environ()
	if onStart != nil {
		onStart(&activeCmd{cancel: cancel, cmd: c, rawCmd: rawCmd})
	}
	out, err := c.CombinedOutput()
	output := string(out)
	if ctx.Err() == context.DeadlineExceeded {
		return 1, formatCommandTimeoutOutput(output)
	}
	if ctx.Err() == context.Canceled {
		return 1, "命令被终止"
	}
	if err != nil {
		// Check if killed by signal
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == -1 {
				return 1, "命令被终止"
			}
		}
		if strings.TrimSpace(output) != "" {
			return 1, output
		}
		return 1, err.Error()
	}
	return 0, output
}

func formatCommandTimeoutOutput(output string) string {
	if strings.TrimSpace(output) == "" {
		return "[Warning: Command execution timed out.]"
	}
	return output + "[Warning: Command execution timed out, the returned content may be incomplete.]"
}

func executeShellCommand(shell, rawCmd string, timeoutMs int64, onStart func(*activeCmd)) (int, string) {
	return executeExternalCommand(shell, []string{"-c", rawCmd}, rawCmd, timeoutMs, onStart)
}

var integrationSandboxCommandPathFn = resolveIntegrationSandboxCommandPath

func executeSandboxCommandWithPath(path, shell, rawCmd string, timeoutMs int64, allowedDir string, onStart func(*activeCmd)) (int, string) {
	args := []string{"--cmd", rawCmd}
	if allowedDir = strings.TrimSpace(allowedDir); allowedDir != "" {
		args = append(args, "--allowed-dir", allowedDir)
	}
	shell = strings.TrimSpace(shell)
	if shell != "" {
		args = append(args, "--shell", shell)
	}
	if timeoutMs > 0 {
		args = append(args, "--timeout", strconv.FormatInt(timeoutMs, 10))
	}
	return executeExternalCommand(path, args, rawCmd, timeoutMs, onStart)
}

func executeSandboxCommand(mode, shell, rawCmd string, timeoutMs int64, allowedDir string, onStart func(*activeCmd)) (int, string) {
	path, err := integrationSandboxCommandPathFn(mode)
	if err != nil {
		return 1, err.Error()
	}
	return executeSandboxCommandWithPath(path, shell, rawCmd, timeoutMs, allowedDir, onStart)
}

func executeSandboxCommandLogged(scope, agentID, chatID, tid, mode, allowedDir, shell, rawCmd string, timeoutMs int64, onStart func(*activeCmd)) (int, string) {
	path, err := integrationSandboxCommandPathFn(mode)
	if err != nil {
		log.Printf("%s: sandbox exec prepare failed agentId=%s chatId=%s tid=%s err=%v cmd=%q",
			scope,
			strings.TrimSpace(agentID),
			strings.TrimSpace(chatID),
			strings.TrimSpace(tid),
			err,
			rawCmd,
		)
		return 1, err.Error()
	}
	logSandboxExecutionStart(scope, agentID, chatID, tid, path, shell, rawCmd, timeoutMs)
	startedAt := time.Now()
	status, output := executeSandboxCommandWithPath(path, shell, rawCmd, timeoutMs, allowedDir, onStart)
	logSandboxExecutionDone(scope, agentID, chatID, tid, status, time.Since(startedAt), output)
	return status, output
}

func sandboxStateForSession(chatID string) (sandboxstate.State, bool, error) {
	if cronDB == nil {
		return sandboxstate.State{}, false, nil
	}
	state, recorded, err := sandboxstate.Get(cronDB, "", chatID)
	if err != nil {
		return sandboxstate.State{}, false, err
	}
	if !recorded {
		return sandboxstate.State{}, false, nil
	}
	return state, true, nil
}

func sandboxModeForSession(chatID string) (string, error) {
	state, recorded, err := sandboxStateForSession(chatID)
	if err != nil {
		return "", err
	}
	if !recorded {
		return "", nil
	}
	return sandboxstate.NormalizeMode(state.Sandbox), nil
}

func sandboxModeForLog(value string) string {
	mode := sandboxstate.NormalizeMode(value)
	if mode == "" {
		return "off"
	}
	return mode
}

func logIntegrationSandboxChange(agentID, chatID, fromMode, toMode string) {
	log.Printf("sandbox change agentId=%s chatId=%s from=%s to=%s",
		strings.TrimSpace(agentID),
		strings.TrimSpace(chatID),
		sandboxModeForLog(fromMode),
		sandboxModeForLog(toMode),
	)
}

func insertCmdLogRecord(agentID, chatID, tid, cmd, receivedAt string) (int64, error) {
	if cronDB == nil {
		return 0, nil
	}
	res, err := cronDB.Exec(`INSERT INTO cmd_log (agent_id, chat_id, tid, cmd, received_at) VALUES (?,?,?,?,?)`,
		agentID, chatID, tid, cmd, receivedAt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func updateCmdLogRecord(id int64, result string, status int, completedAt string) error {
	if cronDB == nil || id <= 0 {
		return nil
	}
	_, err := cronDB.Exec(`UPDATE cmd_log SET result = ?, status = ?, completed_at = ? WHERE id = ?`,
		sharedutil.GzipBase64String(result), status, completedAt, id)
	return err
}

func updateLatestCmdLogRecord(agentID, chatID, tid, result string, status int, completedAt string) error {
	if cronDB == nil {
		return nil
	}
	var id int64
	err := cronDB.QueryRow(`SELECT id FROM cmd_log WHERE agent_id = ? AND chat_id = ? AND tid = ? ORDER BY id DESC LIMIT 1`,
		agentID, chatID, tid).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	return updateCmdLogRecord(id, result, status, completedAt)
}

func initKillLogTable() {
	if cronDB == nil {
		return
	}
	cronDB.Exec(`CREATE TABLE IF NOT EXISTS kill_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		agent_id TEXT NOT NULL,
		chat_id TEXT NOT NULL,
		tid TEXT NOT NULL,
		cmd TEXT NOT NULL,
		received_at TEXT NOT NULL,
		completed_at TEXT NOT NULL DEFAULT ''
	)`)
	cronDB.Exec(`CREATE INDEX IF NOT EXISTS idx_kill_agent_chat_time ON kill_log(agent_id, chat_id, received_at)`)
}

func insertKillLogRecord(agentID, chatID, tid, cmd, receivedAt string) (int64, error) {
	if cronDB == nil {
		return 0, nil
	}
	initKillLogTable()
	res, err := cronDB.Exec(`INSERT INTO kill_log (agent_id, chat_id, tid, cmd, received_at) VALUES (?,?,?,?,?)`,
		agentID, chatID, tid, cmd, receivedAt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func updateKillLogRecord(id int64, completedAt string) error {
	if cronDB == nil || id <= 0 {
		return nil
	}
	initKillLogTable()
	_, err := cronDB.Exec(`UPDATE kill_log SET completed_at = ? WHERE id = ?`, completedAt, id)
	return err
}

type activeCmd struct {
	cancel    context.CancelFunc
	cmd       *exec.Cmd
	agentID   string
	chatID    string
	tid       string
	rawCmd    string
	startedAt string
}

var (
	activeCmdMu  sync.Mutex
	activeCmdMap = make(map[string]*activeCmd)
)

func activeCmdKey(agentID, chatID, tid, cmd string) string {
	return agentID + "|" + chatID + "|" + tid + "|" + cmd
}

func registerActiveCmd(cmd *activeCmd) string {
	key := activeCmdKey(cmd.agentID, cmd.chatID, cmd.tid, cmd.rawCmd)
	activeCmdMu.Lock()
	activeCmdMap[key] = cmd
	activeCmdMu.Unlock()
	return key
}

func unregisterActiveCmd(key string) {
	activeCmdMu.Lock()
	delete(activeCmdMap, key)
	activeCmdMu.Unlock()
}

func cancelAllActiveCmds() {
	activeCmdMu.Lock()
	items := make([]*activeCmd, 0, len(activeCmdMap))
	for key, cmd := range activeCmdMap {
		items = append(items, cmd)
		delete(activeCmdMap, key)
	}
	activeCmdMu.Unlock()

	for _, cmd := range items {
		if cmd == nil {
			continue
		}
		if cmd.cancel != nil {
			cmd.cancel()
		}
		if cmd.cmd != nil && cmd.cmd.Process != nil {
			if runtime.GOOS == "windows" {
				_ = cmd.cmd.Process.Kill()
			} else {
				_ = cmd.cmd.Process.Signal(syscall.SIGKILL)
			}
		}
	}
}

func findActiveCmd(agentID, chatID, tid, rawCmd string) (*activeCmd, string) {
	activeCmdMu.Lock()
	defer activeCmdMu.Unlock()
	if tid != "" {
		key := activeCmdKey(agentID, chatID, tid, rawCmd)
		if ac, ok := activeCmdMap[key]; ok {
			return ac, key
		}
	}
	for key, ac := range activeCmdMap {
		if ac.agentID == agentID && ac.chatID == chatID && ac.rawCmd == rawCmd {
			return ac, key
		}
	}
	var fallback *activeCmd
	var fallbackKey string
	for key, ac := range activeCmdMap {
		if ac.agentID != agentID || ac.chatID != chatID {
			continue
		}
		if fallback != nil {
			return nil, ""
		}
		fallback = ac
		fallbackKey = key
	}
	if fallback != nil {
		return fallback, fallbackKey
	}
	return nil, ""
}

func handleCmd(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !isLocalManagementRequest(r) {
			sharedutil.WriteCmdResponse(w, http.StatusForbidden, CmdResponse{Status: 1, Content: "only localhost requests can execute commands"})
			return
		}

		var req CmdRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sharedutil.WriteCmdResponse(w, http.StatusBadRequest, CmdResponse{Status: 1, Content: "invalid JSON body"})
			return
		}
		sharedutil.NormalizeCmdRequest(&req)
		if req.AgentID == "" {
			sharedutil.WriteCmdResponse(w, http.StatusBadRequest, CmdResponse{Status: 1, Content: "agentId is required"})
			return
		}
		if req.ChatID == "" {
			sharedutil.WriteCmdResponse(w, http.StatusBadRequest, CmdResponse{Status: 1, AgentID: req.AgentID, Content: "chatId is required"})
			return
		}
		if req.Cmd == "" {
			sharedutil.WriteCmdResponse(w, http.StatusBadRequest, CmdResponse{Status: 1, AgentID: req.AgentID, ChatID: req.ChatID, Content: "cmd is required"})
			return
		}
		if sharedutil.ContainsBlockedCommand(req.Cmd) {
			sharedutil.WriteCmdResponse(w, http.StatusBadRequest, CmdResponse{
				Status:  1,
				AgentID: req.AgentID,
				ChatID:  req.ChatID,
				Cmd:     req.Cmd,
				Content: "command contains blocked token rm",
			})
			return
		}

		agentTTL := time.Duration(cfg.AgentCacheMs) * time.Millisecond
		metadata, err := getAgentOutput(cfg.AgentDir, cfg.effectiveDeviceID(), agentTTL)
		if err != nil {
			log.Printf("api/cmd: agent scan error: %v", err)
			sharedutil.WriteCmdResponse(w, http.StatusInternalServerError, CmdResponse{
				Status:  1,
				AgentID: req.AgentID,
				ChatID:  req.ChatID,
				Content: "failed to get agent metadata",
			})
			return
		}
		agent := findAgent(metadata.Agents, req.AgentID)
		if agent == nil {
			sharedutil.WriteCmdResponse(w, http.StatusNotFound, CmdResponse{
				Status:  1,
				AgentID: req.AgentID,
				ChatID:  req.ChatID,
				Content: "agent not found or deleted",
			})
			return
		}

		tid := req.Tid
		if tid == "" {
			tid = sharedutil.GenerateCmdTID()
		}
		receivedAt := time.Now().Format("2006-01-02T15:04:05.000")
		cmdLogID, err := insertCmdLogRecord(req.AgentID, req.ChatID, tid, req.Cmd, receivedAt)
		if err != nil {
			log.Printf("api/cmd: insert cmd_log failed: %v", err)
		}

		sandboxState, recorded, err := sandboxStateForSession(req.ChatID)
		if err != nil {
			log.Printf("api/cmd: sandbox lookup failed: agentId=%s chatId=%s err=%v", req.AgentID, req.ChatID, err)
			sharedutil.WriteCmdResponse(w, http.StatusInternalServerError, CmdResponse{
				Status:  1,
				AgentID: req.AgentID,
				ChatID:  req.ChatID,
				Tid:     tid,
				Cmd:     req.Cmd,
				Content: "failed to query sandbox state",
			})
			return
		}
		sandboxMode := ""
		sandboxAllowedDir := ""
		if recorded {
			sandboxMode = sandboxstate.NormalizeMode(sandboxState.Sandbox)
			sandboxAllowedDir = sandboxState.AllowedDir
		}

		var activeKey string
		var status int
		var output string
		if sandboxMode != "" {
			status, output = executeSandboxCommandLogged("api/cmd", req.AgentID, req.ChatID, tid, sandboxMode, sandboxAllowedDir, sharedutil.DefaultExecShell(), req.Cmd, req.Timeout, func(ac *activeCmd) {
				ac.agentID = req.AgentID
				ac.chatID = req.ChatID
				ac.tid = tid
				ac.startedAt = receivedAt
				activeKey = registerActiveCmd(ac)
			})
		} else {
			status, output = executeShellCommand(sharedutil.DefaultExecShell(), req.Cmd, req.Timeout, func(ac *activeCmd) {
				ac.agentID = req.AgentID
				ac.chatID = req.ChatID
				ac.tid = tid
				ac.startedAt = receivedAt
				activeKey = registerActiveCmd(ac)
			})
		}
		if activeKey != "" {
			unregisterActiveCmd(activeKey)
		}
		completedAt := time.Now().Format("2006-01-02T15:04:05.000")
		if err := updateCmdLogRecord(cmdLogID, output, status, completedAt); err != nil {
			log.Printf("api/cmd: update cmd_log failed: %v", err)
		}

		sharedutil.WriteCmdResponse(w, http.StatusOK, CmdResponse{
			Status:      status,
			AgentID:     req.AgentID,
			ChatID:      req.ChatID,
			Tid:         tid,
			Cmd:         req.Cmd,
			Output:      output,
			ReceivedAt:  receivedAt,
			CompletedAt: completedAt,
		})
	}
}

func handleKill(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !isLocalManagementRequest(r) {
			sharedutil.WriteKillResponse(w, http.StatusForbidden, KillResponse{Status: 1, Content: "only localhost requests can execute commands"})
			return
		}

		var req KillRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sharedutil.WriteKillResponse(w, http.StatusBadRequest, KillResponse{Status: 1, Content: "invalid JSON body"})
			return
		}
		sharedutil.NormalizeKillRequest(&req)
		if req.AgentID == "" {
			sharedutil.WriteKillResponse(w, http.StatusBadRequest, KillResponse{Status: 1, Content: "agentId is required"})
			return
		}
		if req.ChatID == "" {
			sharedutil.WriteKillResponse(w, http.StatusBadRequest, KillResponse{Status: 1, AgentID: req.AgentID, Content: "chatId is required"})
			return
		}
		if req.Cmd == "" {
			sharedutil.WriteKillResponse(w, http.StatusBadRequest, KillResponse{Status: 1, AgentID: req.AgentID, ChatID: req.ChatID, Content: "cmd is required"})
			return
		}

		agentTTL := time.Duration(cfg.AgentCacheMs) * time.Millisecond
		metadata, err := getAgentOutput(cfg.AgentDir, cfg.effectiveDeviceID(), agentTTL)
		if err != nil {
			log.Printf("api/kill: agent scan error: %v", err)
			sharedutil.WriteKillResponse(w, http.StatusInternalServerError, KillResponse{
				Status:  1,
				AgentID: req.AgentID,
				ChatID:  req.ChatID,
				Content: "failed to get agent metadata",
			})
			return
		}
		agent := findAgent(metadata.Agents, req.AgentID)
		if agent == nil {
			sharedutil.WriteKillResponse(w, http.StatusNotFound, KillResponse{
				Status:  1,
				AgentID: req.AgentID,
				ChatID:  req.ChatID,
				Content: "agent not found or deleted",
			})
			return
		}

		receivedAt := time.Now().Format("2006-01-02T15:04:05.000")
		logID, err := insertKillLogRecord(req.AgentID, req.ChatID, req.Tid, req.Cmd, receivedAt)
		if err != nil {
			log.Printf("api/kill: insert kill_log failed: %v", err)
		}

		ac, key := findActiveCmd(req.AgentID, req.ChatID, req.Tid, req.Cmd)
		if ac == nil {
			completedAt := time.Now().Format("2006-01-02T15:04:05.000")
			_ = updateKillLogRecord(logID, completedAt)
			sharedutil.WriteKillResponse(w, http.StatusNotFound, KillResponse{
				Status:      1,
				AgentID:     req.AgentID,
				ChatID:      req.ChatID,
				Tid:         req.Tid,
				Cmd:         req.Cmd,
				Content:     "active command not found",
				ReceivedAt:  receivedAt,
				CompletedAt: completedAt,
			})
			return
		}

		if ac.cancel != nil {
			ac.cancel()
		}
		if ac.cmd != nil && ac.cmd.Process != nil {
			if runtime.GOOS == "windows" {
				_ = ac.cmd.Process.Kill()
			} else {
				_ = ac.cmd.Process.Signal(syscall.SIGKILL)
			}
		}
		unregisterActiveCmd(key)

		completedAt := time.Now().Format("2006-01-02T15:04:05.000")
		if err := updateKillLogRecord(logID, completedAt); err != nil {
			log.Printf("api/kill: update kill_log failed: %v", err)
		}

		sharedutil.WriteKillResponse(w, http.StatusOK, KillResponse{
			Status:      0,
			AgentID:     req.AgentID,
			ChatID:      req.ChatID,
			Tid:         ac.tid,
			Cmd:         ac.rawCmd,
			Content:     "killed",
			ReceivedAt:  receivedAt,
			CompletedAt: completedAt,
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Proxy: /api/agentId — list all agent IDs
// ═══════════════════════════════════════════════════════════════════════════

func handleAgentIDs(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ids, err := agentcore.ListAgentIDs(cfg.AgentDir)
		if err != nil {
			log.Printf("api/agentId: agent directory read error: %v", err)
			http.Error(w, "Failed to list agent IDs", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ids)
	}
}

func swarmAgentIDs(metadata *AgentOutput, excludeAgentID string) []string {
	if metadata == nil || len(metadata.Agents) == 0 {
		return []string{}
	}
	exclude := strings.TrimSpace(excludeAgentID)
	ids := make([]string, 0, len(metadata.Agents))
	for _, agent := range metadata.Agents {
		if agent.RouterDisable {
			continue
		}
		if exclude != "" && strings.TrimSpace(agent.AgentID) == exclude {
			continue
		}
		ids = append(ids, agent.AgentID)
	}
	return ids
}

func handleSwarmAgents(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentTTL := time.Duration(cfg.AgentCacheMs) * time.Millisecond
		metadata, err := getAgentOutput(cfg.AgentDir, cfg.effectiveDeviceID(), agentTTL)
		if err != nil {
			log.Printf("api/swarm_agent: agent scan error: %v", err)
			http.Error(w, "Failed to get agent metadata", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(swarmAgentIDs(metadata, r.URL.Query().Get("agentId")))
	}
}

func handleDeviceID(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentTTL := time.Duration(cfg.AgentCacheMs) * time.Millisecond
		metadata, err := getAgentOutput(cfg.AgentDir, cfg.effectiveDeviceID(), agentTTL)
		if err != nil {
			log.Printf("api/deviceId: agent scan error: %v", err)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "failed to get deviceId"})
			return
		}
		deviceID := strings.TrimSpace(metadata.DeviceID)
		if deviceID == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "deviceId is empty"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "deviceId": deviceID})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Proxy: /api/folder — open agent workspace folder
// ═══════════════════════════════════════════════════════════════════════════

var openFolderFn = openSystemTarget
var openSystemTargetGOOS = runtime.GOOS
var openSystemTargetIsWSLFn = integrationBrowserIsWSL
var openSystemTargetExecCommandFn = exec.Command
var openSystemTargetForegroundWSLFn = openSystemTargetForegroundWSL
var openSystemTargetStartCommandFn = func(name string, args ...string) error {
	return openSystemTargetExecCommandFn(name, args...).Start()
}
var openSystemTargetRunCommandFn = func(name string, args ...string) error {
	return openSystemTargetExecCommandFn(name, args...).Run()
}
var openSystemTargetLookPathFn = exec.LookPath
var openSystemTargetStatFn = os.Stat
var openSystemTargetOutputFn = func(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

func openFolder(path string) error {
	return openFolderFn(path)
}

func openSystemTarget(target string) error {
	if openSystemTargetGOOS == "linux" && openSystemTargetIsWSLFn() {
		if err := openSystemTargetRunCommandFn("xdg-open", target); err == nil {
			return nil
		}
		if err := openSystemTargetForegroundWSLFn(target); err == nil {
			return nil
		}
	}
	cmd, args, err := openSystemTargetCommand(target)
	if err != nil {
		return err
	}
	return openSystemTargetStartCommandFn(cmd, args...)
}

func openSystemTargetCommand(target string) (string, []string, error) {
	switch openSystemTargetGOOS {
	case "darwin":
		return "open", []string{target}, nil
	case "linux":
		if openSystemTargetIsWSLFn() {
			return openSystemTargetCommandWSL(target)
		}
		return "xdg-open", []string{target}, nil
	case "windows":
		return "explorer", []string{target}, nil
	default:
		return "", nil, fmt.Errorf("unsupported OS: %s", openSystemTargetGOOS)
	}
}

func openSystemTargetCommandWSL(target string) (string, []string, error) {
	winTarget, err := openSystemTargetWSLPath(target)
	if err != nil {
		return "", nil, err
	}
	if path, ok := openSystemTargetFirstExistingFile("/mnt/c/Windows/explorer.exe"); ok {
		return path, []string{winTarget}, nil
	}
	if path, ok := openSystemTargetFirstLookPath("explorer.exe"); ok {
		return path, []string{winTarget}, nil
	}
	if path, ok := openSystemTargetFirstExistingFile("/mnt/c/Windows/System32/cmd.exe"); ok {
		return path, []string{"/c", "start", "", winTarget}, nil
	}
	if path, ok := openSystemTargetFirstLookPath("cmd.exe"); ok {
		return path, []string{"/c", "start", "", winTarget}, nil
	}
	return "", nil, errors.New("WSL open folder requires explorer.exe or cmd.exe")
}

func openSystemTargetWSLPath(target string) (string, error) {
	out, err := openSystemTargetOutputFn("wslpath", "-w", target)
	if err != nil {
		return "", fmt.Errorf("convert WSL path: %w", err)
	}
	winTarget := strings.TrimSpace(string(out))
	if winTarget == "" {
		return "", errors.New("convert WSL path: empty result")
	}
	return winTarget, nil
}

func openSystemTargetForegroundWSL(target string) error {
	winTarget, err := openSystemTargetWSLPath(target)
	if err != nil {
		return err
	}
	executable, ok := openSystemTargetPowerShellExecutable()
	if !ok {
		return errors.New("WSL foreground open requires PowerShell")
	}
	return openSystemTargetRunCommandFn(executable, openSystemTargetPowerShellArgs(winTarget)...)
}

func openSystemTargetPowerShellExecutable() (string, bool) {
	if path, ok := openSystemTargetFirstExistingFile(
		"/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe",
		"/mnt/c/Program Files/PowerShell/7/pwsh.exe",
	); ok {
		return path, true
	}
	if path, ok := openSystemTargetFirstLookPath("powershell.exe", "pwsh.exe", "powershell", "pwsh"); ok {
		return path, true
	}
	return "", false
}

func openSystemTargetPowerShellArgs(winTarget string) []string {
	return []string{
		"-NoProfile",
		"-NonInteractive",
		"-WindowStyle",
		"Hidden",
		"-ExecutionPolicy",
		"Bypass",
		"-EncodedCommand",
		openSystemTargetEncodePowerShellCommand(openSystemTargetBuildForegroundScript(winTarget)),
	}
}

func openSystemTargetBuildForegroundScript(winTarget string) string {
	escapedTarget := strings.ReplaceAll(winTarget, "'", "''")
	return fmt.Sprintf(`$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Windows.Forms | Out-Null
Add-Type @"
using System;
using System.Runtime.InteropServices;
public static class DeepRightWin32 {
  [DllImport("user32.dll")]
  public static extern bool ShowWindowAsync(IntPtr hWnd, int nCmdShow);
  [DllImport("user32.dll")]
  public static extern bool SetForegroundWindow(IntPtr hWnd);
}
"@ | Out-Null
$path = '%s'
$normalizedPath = [System.IO.Path]::GetFullPath($path)
$shell = New-Object -ComObject Shell.Application
$ws = New-Object -ComObject WScript.Shell
Start-Process explorer.exe -ArgumentList $path | Out-Null
$windowRef = $null
for ($i = 0; $i -lt 40 -and -not $windowRef; $i++) {
  Start-Sleep -Milliseconds 250
  foreach ($window in @($shell.Windows())) {
    try {
      $folderPath = $window.Document.Folder.Self.Path
      if ([string]::IsNullOrWhiteSpace($folderPath)) { continue }
      $candidatePath = [System.IO.Path]::GetFullPath($folderPath)
      if ([string]::Equals($candidatePath, $normalizedPath, [System.StringComparison]::OrdinalIgnoreCase)) {
        $windowRef = $window
        break
      }
    } catch {}
  }
}
if (-not $windowRef) { exit 7 }
$hwnd = [IntPtr]::new([int64]$windowRef.HWND)
if ($hwnd -eq [IntPtr]::Zero) { exit 8 }
[System.Windows.Forms.SendKeys]::SendWait('%%')
Start-Sleep -Milliseconds 100
[DeepRightWin32]::ShowWindowAsync($hwnd, 9) | Out-Null
Start-Sleep -Milliseconds 100
$activated = $false
for ($i = 0; $i -lt 10 -and -not $activated; $i++) {
  try { $null = $ws.AppActivate($windowRef.LocationName) } catch {}
  $activated = [DeepRightWin32]::SetForegroundWindow($hwnd)
  if (-not $activated) {
    Start-Sleep -Milliseconds 150
  }
}
if (-not $activated) { exit 9 }
`, escapedTarget)
}

func openSystemTargetEncodePowerShellCommand(script string) string {
	runes := utf16.Encode([]rune(script))
	encoded := make([]byte, 0, len(runes)*2)
	for _, r := range runes {
		encoded = append(encoded, byte(r), byte(r>>8))
	}
	return base64.StdEncoding.EncodeToString(encoded)
}

func openSystemTargetFirstExistingFile(paths ...string) (string, bool) {
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		info, err := openSystemTargetStatFn(path)
		if err == nil && info != nil && !info.IsDir() {
			return path, true
		}
	}
	return "", false
}

func openSystemTargetFirstLookPath(names ...string) (string, bool) {
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		path, err := openSystemTargetLookPathFn(name)
		if err == nil && strings.TrimSpace(path) != "" {
			return path, true
		}
	}
	return "", false
}

func integrationBrowserURL(port int) string {
	return integrationBuildBrowserURL("localhost", port)
}

func integrationBrowserOpenURL(port int) string {
	return integrationBuildBrowserOpenURL("localhost", port)
}

var integrationSiteVersionTokenFn = integrationSiteVersionToken

func integrationSiteVersionToken() string {
	siteIndexPath := strings.TrimSpace(integrationResolveResourcePath(filepath.Join("site", "index.html")))
	if siteIndexPath == "" {
		return ""
	}
	info, err := os.Stat(siteIndexPath)
	if err != nil || info == nil || info.IsDir() {
		return ""
	}
	modTime := info.ModTime().UTC().Unix()
	size := info.Size()
	if modTime <= 0 && size <= 0 {
		return ""
	}
	return fmt.Sprintf("%x-%x", modTime, size)
}

func integrationBuildBrowserURL(host string, port int) string {
	host = strings.TrimSpace(host)
	if host == "" {
		host = "localhost"
	}
	if port <= 0 {
		port = 8080
	}
	base := fmt.Sprintf("http://%s:%d/site/", host, port)
	versionToken := strings.TrimSpace(integrationSiteVersionTokenFn())
	if versionToken == "" {
		return base + "#app"
	}
	return base + "?v=" + url.QueryEscape(versionToken) + "#app"
}

func integrationBuildBrowserOpenURL(host string, port int) string {
	host = strings.TrimSpace(host)
	if host == "" {
		host = "localhost"
	}
	if port <= 0 {
		port = 8080
	}
	return fmt.Sprintf("http://%s:%d/launch", host, port)
}

type lanAccessCandidate struct {
	Host           string
	InterfaceName  string
	InterfaceIndex int
	Private        bool
}

var detectLANAccessHost = func() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("list interfaces: %w", err)
	}
	candidates := make([]lanAccessCandidate, 0)
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagPointToPoint != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip := extractIPv4FromAddr(addr)
			if ip == nil || !ip.IsGlobalUnicast() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			candidates = append(candidates, lanAccessCandidate{
				Host:           ip.String(),
				InterfaceName:  iface.Name,
				InterfaceIndex: iface.Index,
				Private:        ip.IsPrivate(),
			})
		}
	}
	host := selectPreferredLANAccessHost(candidates)
	if host == "" {
		return "", fmt.Errorf("no LAN-accessible IPv4 address found")
	}
	return host, nil
}

func extractIPv4FromAddr(addr net.Addr) net.IP {
	switch value := addr.(type) {
	case *net.IPNet:
		if value == nil {
			return nil
		}
		return value.IP.To4()
	case *net.IPAddr:
		if value == nil {
			return nil
		}
		return value.IP.To4()
	default:
		return nil
	}
}

func lanInterfacePriority(name string) int {
	lower := strings.ToLower(strings.TrimSpace(name))
	switch {
	case lower == "":
		return 0
	case strings.HasPrefix(lower, "en"),
		strings.HasPrefix(lower, "eth"),
		strings.HasPrefix(lower, "wlan"),
		strings.HasPrefix(lower, "wl"),
		strings.HasPrefix(lower, "wifi"),
		strings.HasPrefix(lower, "lan"):
		return 30
	case strings.HasPrefix(lower, "bridge"),
		strings.HasPrefix(lower, "br"):
		return 6
	case strings.HasPrefix(lower, "docker"),
		strings.HasPrefix(lower, "veth"),
		strings.HasPrefix(lower, "utun"),
		strings.HasPrefix(lower, "awdl"),
		strings.HasPrefix(lower, "llw"),
		strings.HasPrefix(lower, "tailscale"),
		strings.HasPrefix(lower, "zt"),
		strings.HasPrefix(lower, "tun"),
		strings.HasPrefix(lower, "tap"),
		strings.HasPrefix(lower, "vmnet"),
		strings.HasPrefix(lower, "vboxnet"):
		return -20
	default:
		return 0
	}
}

func selectPreferredLANAccessHost(candidates []lanAccessCandidate) string {
	if len(candidates) == 0 {
		return ""
	}
	best := candidates[0]
	for i := 1; i < len(candidates); i++ {
		current := candidates[i]
		if current.Private != best.Private {
			if current.Private {
				best = current
			}
			continue
		}
		currentPriority := lanInterfacePriority(current.InterfaceName)
		bestPriority := lanInterfacePriority(best.InterfaceName)
		if currentPriority != bestPriority {
			if currentPriority > bestPriority {
				best = current
			}
			continue
		}
		if current.InterfaceIndex != best.InterfaceIndex {
			if current.InterfaceIndex < best.InterfaceIndex {
				best = current
			}
			continue
		}
		if strings.Compare(current.Host, best.Host) < 0 {
			best = current
		}
	}
	return strings.TrimSpace(best.Host)
}

func buildLANAccessURL(host string, port int) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	return integrationBuildBrowserURL(host, port)
}

func integrationBrowserEntryURL(port int) string {
	if port <= 0 {
		port = 8080
	}
	return fmt.Sprintf("http://127.0.0.1:%d/site/", port)
}

func integrationLaunchSplashLogoPath() string {
	candidates := []string{
		integrationResolveResourcePath(filepath.Join("site", "icon.png")),
		filepath.Join(integrationResourcesDir(), "site", "icon.png"),
		filepath.Join(integrationExecutableBaseDir(), "site", "icon.png"),
	}
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if abs, err := filepath.Abs(candidate); err == nil {
			candidate = abs
		} else {
			candidate = filepath.Clean(candidate)
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func integrationDefaultLaunchSplashConfig() (launchsplash.Config, error) {
	logoPath := integrationLaunchSplashLogoPath()
	if strings.TrimSpace(logoPath) == "" {
		return launchsplash.Config{}, fmt.Errorf("launch splash logo not found under %s", integrationResourcesDir())
	}
	return launchsplash.Config{
		LogoPath: logoPath,
		Duration: 5 * time.Second,
	}, nil
}

func startIntegrationLaunchSplash() error {
	cfg, err := integrationDefaultLaunchSplashConfig()
	if err != nil {
		return err
	}
	return launchsplash.Start(cfg)
}

func integrationShouldOpenBrowser() bool {
	return !queryBool(os.Getenv(integrationSkipBrowserEnv))
}

func openIntegrationBrowserAfterEntryReady(port int, logf func(string, ...interface{})) {
	if !waitIntegrationBrowserEntryReady(port) && logf != nil {
		logf("browser entry not ready yet, opening browser anyway addr=%s", fmt.Sprintf("127.0.0.1:%d", port))
	}
	browserURL := integrationBrowserOpenURL(port)
	if err := integrationOpenBrowserFn(browserURL); err != nil {
		if logf != nil {
			logf("open browser failed: %v", err)
		}
		return
	}
	if logf != nil {
		logf("browser opened maximized: %s", browserURL)
	}
}

func handleFolder(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		pathParam := normalizeQuotedPathArg(r.URL.Query().Get("path"))
		if pathParam != "" {
			expandedPath, err := expandPath(pathParam)
			if err != nil {
				http.Error(w, "Failed to expand path: "+err.Error(), http.StatusBadRequest)
				return
			}
			if !filepath.IsAbs(expandedPath) {
				http.Error(w, "path must be absolute", http.StatusBadRequest)
				return
			}
			resolved, err := resolveCaseInsensitive(expandedPath)
			if err != nil {
				http.Error(w, "Failed to resolve path: "+err.Error(), http.StatusNotFound)
				return
			}
			info, err := os.Stat(resolved)
			if err != nil {
				http.Error(w, "Failed to access path: "+err.Error(), http.StatusNotFound)
				return
			}
			if !info.IsDir() {
				http.Error(w, "path is not a directory", http.StatusBadRequest)
				return
			}
			if err := openFolder(resolved); err != nil {
				log.Printf("open folder error: %v", err)
				http.Error(w, "Failed to open folder: "+err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "ok", "path": resolved})
			return
		}
		agentID := r.URL.Query().Get("agentId")
		if agentID == "" {
			http.Error(w, "agentId is required", http.StatusBadRequest)
			return
		}
		agentTTL := time.Duration(cfg.AgentCacheMs) * time.Millisecond
		metadata, err := getAgentOutput(cfg.AgentDir, cfg.effectiveDeviceID(), agentTTL)
		if err != nil {
			log.Printf("api/folder: agent scan error: %v", err)
			http.Error(w, "Failed to get agent metadata", http.StatusInternalServerError)
			return
		}
		var workspace string
		for _, a := range metadata.Agents {
			if a.AgentID == agentID {
				workspace = a.Workspace
				break
			}
		}
		if workspace == "" {
			http.Error(w, "Agent not found: "+agentID, http.StatusNotFound)
			return
		}
		dir := normalizeQuotedPathArg(r.URL.Query().Get("dir"))
		target := workspace
		if dir != "" {
			target = filepath.Join(workspace, dir)
		}
		if err := openFolder(target); err != nil {
			log.Printf("open folder error: %v", err)
			http.Error(w, "Failed to open folder: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "path": target})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Proxy: /api/skills — list skill names for an agent
// ═══════════════════════════════════════════════════════════════════════════

func handleSkills(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentID := r.URL.Query().Get("agentId")
		if agentID == "" {
			http.Error(w, "agentId is required", http.StatusBadRequest)
			return
		}
		agentTTL := time.Duration(cfg.AgentCacheMs) * time.Millisecond
		chatID := strings.TrimSpace(r.URL.Query().Get("chatId"))
		var (
			metadata *AgentOutput
			err      error
		)
		if chatID != "" {
			metadata, err = getAgentOutputForChat(cfg.AgentDir, cfg.effectiveDeviceID(), agentTTL, chatID)
		} else {
			metadata, err = getAgentOutput(cfg.AgentDir, cfg.effectiveDeviceID(), agentTTL)
		}
		if err != nil {
			log.Printf("api/skills: agent scan error: %v", err)
			http.Error(w, "Failed to get agent metadata", http.StatusInternalServerError)
			return
		}
		for _, a := range metadata.Agents {
			if a.AgentID == agentID {
				names := buildIntegrationRuntimeSkillNames(agentcore.SkillNames(a.Skills))
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(names)
				return
			}
		}
		http.Error(w, "Agent not found: "+agentID, http.StatusNotFound)
	}
}

func normalizeIntegrationConfiguredSkillNames(input interface{}) []string {
	switch value := input.(type) {
	case []string:
		out := make([]string, 0, len(value))
		for _, item := range value {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []interface{}:
		out := make([]string, 0, len(value))
		for _, item := range value {
			text := strings.TrimSpace(fmt.Sprint(item))
			if text == "" || text == "<nil>" {
				continue
			}
			out = append(out, text)
		}
		return out
	case string:
		value = strings.TrimSpace(value)
		if value == "" {
			return nil
		}
		return []string{value}
	default:
		return nil
	}
}

func configuredIntegrationSkillNames() []string {
	path := strings.TrimSpace(integrationStartupConfigPath())
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var raw struct {
		Skills interface{} `json:"skills"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	return normalizeIntegrationConfiguredSkillNames(raw.Skills)
}

func buildIntegrationRuntimeSkillNames(base []string) []string {
	out := make([]string, 0, len(base)+5)
	seen := make(map[string]struct{}, len(base)+4)
	seedreamEnabled := integrationSeedreamSkillEnabled()
	appendIfMissing := func(name string, enabled bool) {
		name = strings.TrimSpace(name)
		if !enabled || name == "" {
			return
		}
		if name == seedreamInternalSkillName && !seedreamEnabled {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		out = append(out, name)
		seen[name] = struct{}{}
	}

	for _, name := range base {
		appendIfMissing(name, true)
	}
	for _, name := range configuredIntegrationSkillNames() {
		appendIfMissing(normalizeConfiguredIntegrationSkillName(name), true)
	}
	appendIfMissing(seedreamInternalSkillName, seedreamEnabled)
	for _, item := range []struct {
		key   string
		skill string
	}{
		{key: "browser", skill: connectsvc.InternalSkillBrowser},
		{key: "remote", skill: connectsvc.InternalSkillRemote},
	} {
		status, err := connectsvc.PluginStatusByKey(item.key, nil)
		appendIfMissing(item.skill, err == nil && status != nil && status.Started)
	}
	return out
}

// ═══════════════════════════════════════════════════════════════════════════
// Proxy: /api/files — list files in a directory
// ═══════════════════════════════════════════════════════════════════════════

type FileEntry struct {
	Name                   string `json:"name"`
	Type                   string `json:"type"`
	HasSkill               bool   `json:"hasSkill,omitempty"`
	SkillDisabled          bool   `json:"skillDisabled,omitempty"`
	SkillDisabledSelf      bool   `json:"skillDisabledSelf,omitempty"`
	SkillDisabledInherited bool   `json:"skillDisabledInherited,omitempty"`
}

func normalizeQuotedPathArg(path string) string {
	trimmed := strings.TrimSpace(path)
	if len(trimmed) >= 2 {
		if (trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"') || (trimmed[0] == '\'' && trimmed[len(trimmed)-1] == '\'') {
			return strings.TrimSpace(trimmed[1 : len(trimmed)-1])
		}
	}
	return trimmed
}

func expandPath(path string) (string, error) {
	path = normalizeQuotedPathArg(path)
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[1:]), nil
	}
	return path, nil
}

func handleFiles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		dirPath := r.URL.Query().Get("path")
		if dirPath == "" {
			http.Error(w, "path is required", http.StatusBadRequest)
			return
		}
		expanded, err := expandPath(dirPath)
		if err != nil {
			http.Error(w, "Failed to expand path: "+err.Error(), http.StatusBadRequest)
			return
		}
		// If path is a directory, list all; otherwise prefix-match in parent
		var searchDir, prefix string
		info, statErr := os.Stat(expanded)
		if statErr == nil && info.IsDir() {
			searchDir = expanded
			prefix = ""
		} else {
			searchDir = filepath.Dir(expanded)
			prefix = filepath.Base(expanded)
		}
		entries, err := os.ReadDir(searchDir)
		if err != nil {
			http.Error(w, "Failed to read directory: "+err.Error(), http.StatusNotFound)
			return
		}
		disabledPaths, err := disabledSkillPathsForChat(r.URL.Query().Get("chatId"))
		if err != nil {
			http.Error(w, "Failed to read skill state: "+err.Error(), http.StatusInternalServerError)
			return
		}
		result := make([]FileEntry, 0, len(entries))
		for _, e := range entries {
			if prefix != "" && !strings.HasPrefix(strings.ToLower(e.Name()), strings.ToLower(prefix)) {
				continue
			}
			t := "file"
			if e.IsDir() {
				t = "dir"
			}
			item := FileEntry{Name: e.Name(), Type: t}
			if e.IsDir() {
				entryPath := filepath.Join(searchDir, e.Name())
				item.HasSkill = hasDirectSkillMarkdown(entryPath)
				if item.HasSkill {
					item.SkillDisabledSelf, item.SkillDisabledInherited = skillstate.DisabledStatus(disabledPaths, entryPath)
					item.SkillDisabled = item.SkillDisabledSelf || item.SkillDisabledInherited
				}
			}
			result = append(result, item)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

type skillStateRequest struct {
	ChatID   string `json:"chatId"`
	Path     string `json:"path"`
	Disabled bool   `json:"disabled"`
}

func hasDirectSkillMarkdown(dir string) bool {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(dir, "SKILL.md"))
	return err == nil && !info.IsDir()
}

func handleSkillState() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req skillStateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": "invalid request body"})
			return
		}
		req.ChatID = strings.TrimSpace(req.ChatID)
		req.Path = skillstate.NormalizePath(req.Path)
		if req.ChatID == "" || req.Path == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": "chatId and path are required"})
			return
		}
		info, err := os.Stat(req.Path)
		if err != nil || !info.IsDir() || !hasDirectSkillMarkdown(req.Path) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": "path must be a skill directory"})
			return
		}
		db := cronDB
		opened := false
		if db == nil {
			db, err = sql.Open("sqlite", resolveIntegrationDBPath())
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": err.Error()})
				return
			}
			opened = true
			defer db.Close()
		}
		paths, err := skillstate.SetDisabled(db, req.ChatID, req.Path, req.Disabled)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": err.Error()})
			return
		}
		if opened && cronDB == nil {
			_ = skillstate.EnsureSchema(db)
		}
		selfDisabled, inheritedDisabled := skillstate.DisabledStatus(paths, req.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":            0,
			"chatId":            req.ChatID,
			"path":              req.Path,
			"disabled":          selfDisabled || inheritedDisabled,
			"disabledSelf":      selfDisabled,
			"disabledInherited": inheritedDisabled,
			"disabledPaths":     paths,
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Proxy: /api/data — read file content
// ═══════════════════════════════════════════════════════════════════════════

func resolveCaseInsensitive(target string) (string, error) {
	expanded, err := expandPath(target)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(expanded); err == nil {
		return expanded, nil
	}
	parts := strings.Split(filepath.Clean(expanded), string(filepath.Separator))
	resolved := string(filepath.Separator)
	for _, part := range parts {
		if part == "" {
			continue
		}
		entries, err := os.ReadDir(resolved)
		if err != nil {
			return "", fmt.Errorf("cannot read %s: %w", resolved, err)
		}
		found := false
		for _, e := range entries {
			if strings.EqualFold(e.Name(), part) {
				resolved = filepath.Join(resolved, e.Name())
				found = true
				break
			}
		}
		if !found {
			return "", fmt.Errorf("not found: %s in %s", part, resolved)
		}
	}
	return resolved, nil
}

func resolveCaseInsensitiveUnderRoot(root, relPath string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	cleanRel := filepath.Clean(normalizeQuotedPathArg(relPath))
	if cleanRel == "." || cleanRel == "" {
		return rootAbs, nil
	}
	if filepath.IsAbs(cleanRel) {
		return "", fmt.Errorf("absolute path is not allowed")
	}
	parts := strings.Split(cleanRel, string(filepath.Separator))
	resolved := rootAbs
	for idx, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			return "", fmt.Errorf("path escapes workspace")
		}
		exact := filepath.Join(resolved, part)
		if _, err := os.Stat(exact); err == nil {
			resolved = exact
			continue
		}
		entries, err := os.ReadDir(resolved)
		if err != nil {
			if os.IsNotExist(err) {
				return exact, nil
			}
			return "", err
		}
		matched := false
		for _, entry := range entries {
			if strings.EqualFold(entry.Name(), part) {
				resolved = filepath.Join(resolved, entry.Name())
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		remaining := filepath.Join(parts[idx:]...)
		return filepath.Join(resolved, remaining), nil
	}
	return resolved, nil
}

func ensurePathWithinRoot(root, target string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

type knowledgeTreeNode struct {
	Name     string
	Children []*knowledgeTreeNode
}

func buildKnowledgeTree(root string) (*knowledgeTreeNode, error) {
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("knowledge path is not a directory: %s", root)
	}

	var walk func(string) (*knowledgeTreeNode, error)
	walk = func(path string) (*knowledgeTreeNode, error) {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		node := &knowledgeTreeNode{Name: info.Name()}
		if !info.IsDir() {
			return node, nil
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		sort.Slice(entries, func(i, j int) bool {
			left, right := entries[i], entries[j]
			if left.IsDir() != right.IsDir() {
				return left.IsDir()
			}
			return strings.ToLower(left.Name()) < strings.ToLower(right.Name())
		})
		for _, entry := range entries {
			child, err := walk(filepath.Join(path, entry.Name()))
			if err != nil {
				return nil, err
			}
			node.Children = append(node.Children, child)
		}
		return node, nil
	}

	return walk(root)
}

func renderKnowledgeTree(node *knowledgeTreeNode) string {
	if node == nil {
		return ""
	}
	var lines []string
	lines = append(lines, node.Name+"/")
	var writeChildren func(children []*knowledgeTreeNode, prefix string)
	writeChildren = func(children []*knowledgeTreeNode, prefix string) {
		for i, child := range children {
			last := i == len(children)-1
			branch := "|-- "
			nextPrefix := prefix + "|   "
			if last {
				branch = "`-- "
				nextPrefix = prefix + "    "
			}
			label := child.Name
			if len(child.Children) > 0 {
				label += "/"
			}
			lines = append(lines, prefix+branch+label)
			if len(child.Children) > 0 {
				writeChildren(child.Children, nextPrefix)
			}
		}
	}
	writeChildren(node.Children, "")
	return strings.Join(lines, "\n") + "\n"
}

func mappedStaticRelativePath(requestPath, prefix, rootName string) (string, error) {
	decoded, err := url.PathUnescape(requestPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	if decoded != prefix && !strings.HasPrefix(decoded, prefix+"/") {
		return "", fmt.Errorf("path does not match %s", prefix)
	}
	trimmed := strings.TrimPrefix(decoded, prefix)
	trimmed = strings.TrimPrefix(trimmed, "/")
	if trimmed == "" {
		return ".", nil
	}
	cleaned := filepath.Clean(trimmed)
	if cleaned == "." {
		return ".", nil
	}
	if filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes %s root", rootName)
	}
	return cleaned, nil
}

func knowledgeRelativePath(requestPath string) (string, error) {
	return mappedStaticRelativePath(requestPath, "/knowledge", "knowledge")
}

func appStaticRequestTarget(requestPath string) (string, string, error) {
	decoded, err := url.PathUnescape(requestPath)
	if err != nil {
		return "", "", fmt.Errorf("invalid path: %w", err)
	}
	if decoded == "/mapping" || decoded == "/mapping/" {
		return "", "", fmt.Errorf("agentId is required")
	}
	if !strings.HasPrefix(decoded, "/mapping/") {
		return "", "", fmt.Errorf("path does not match /mapping")
	}
	trimmed := strings.TrimPrefix(decoded, "/mapping/")
	parts := strings.SplitN(trimmed, "/", 2)
	agentID := strings.TrimSpace(parts[0])
	if agentID == "" || agentID == "." || agentID == ".." {
		return "", "", fmt.Errorf("agentId is required")
	}
	if err := integrationagentarchive.ValidateAgentID(agentID); err != nil {
		return "", "", fmt.Errorf("invalid agentId: %w", err)
	}
	if len(parts) == 1 || parts[1] == "" {
		return agentID, ".", nil
	}
	relPath := strings.TrimPrefix(parts[1], "/")
	if relPath == "" {
		return agentID, ".", nil
	}
	if strings.Contains(relPath, "\\") {
		return "", "", fmt.Errorf("path escapes app root")
	}
	relPath = path.Clean(relPath)
	if relPath == "." {
		return agentID, ".", nil
	}
	if relPath == ".." || strings.HasPrefix(relPath, "../") {
		return "", "", fmt.Errorf("path escapes app root")
	}
	return agentID, relPath, nil
}

func handleKnowledge(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentDir := ""
		if cfg != nil {
			agentDir = cfg.AgentDir
		}
		root, err := integrationKnowledgeRoot(agentDir)
		if err != nil {
			http.Error(w, "Failed to resolve knowledge dir: "+err.Error(), http.StatusInternalServerError)
			return
		}
		info, err := os.Stat(root)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "Knowledge not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to access knowledge dir: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if !info.IsDir() {
			http.Error(w, "Knowledge path is not a directory", http.StatusInternalServerError)
			return
		}

		relPath, err := knowledgeRelativePath(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		target := filepath.Join(root, relPath)
		if !ensurePathWithinRoot(root, target) {
			http.Error(w, "path escapes knowledge root", http.StatusBadRequest)
			return
		}

		targetInfo, err := os.Stat(target)
		if err != nil {
			if os.IsNotExist(err) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "Failed to access knowledge target: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if targetInfo.IsDir() {
			tree, err := buildKnowledgeTree(target)
			if err != nil {
				http.Error(w, "Failed to build knowledge tree: "+err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = io.WriteString(w, renderKnowledgeTree(tree))
			return
		}

		http.ServeFile(w, r, target)
	}
}

func handleAppStatic(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		agentDir := ""
		if cfg != nil {
			agentDir = cfg.AgentDir
		}
		agentID, relPath, err := appStaticRequestTarget(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		root, err := integrationAppStaticRoot(agentDir, agentID)
		if err != nil {
			http.Error(w, "Failed to resolve app dir: "+err.Error(), http.StatusInternalServerError)
			return
		}
		target := filepath.Join(root, filepath.FromSlash(relPath))
		if !ensurePathWithinRoot(root, target) {
			http.Error(w, "path escapes app root", http.StatusBadRequest)
			return
		}
		if info, statErr := os.Stat(target); statErr == nil && !info.IsDir() {
			file, openErr := os.Open(target)
			if openErr != nil {
				http.Error(w, "Failed to open app file: "+openErr.Error(), http.StatusInternalServerError)
				return
			}
			defer file.Close()
			http.ServeContent(w, r, info.Name(), info.ModTime(), file)
			return
		}

		served := r.Clone(r.Context())
		if r.URL != nil {
			clonedURL := *r.URL
			served.URL = &clonedURL
		}
		if relPath == "." {
			served.URL.Path = "/"
		} else {
			served.URL.Path = "/" + filepath.ToSlash(relPath)
			// Preserve a caller-provided trailing slash for directories so
			// FileServer does not emit a relative redirect against the outer
			// /mapping/{agentId}/ URL and accidentally duplicate the subdirectory.
			if strings.HasSuffix(r.URL.Path, "/") {
				if info, statErr := os.Stat(target); statErr == nil && info.IsDir() {
					served.URL.Path += "/"
				}
			}
		}
		served.URL.RawPath = ""
		http.FileServer(http.Dir(root)).ServeHTTP(w, served)
	}
}

func handleKnowledgeLastUpdate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentID := strings.TrimSpace(r.URL.Query().Get("agentId"))
		text, err := integrationknowledge.LoadLastUpdateText(integrationAppDir(), agentID, time.Local)
		if err != nil {
			http.Error(w, "Failed to load knowledge last update: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, text)
	}
}

func handleKnowledgePath(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentDir := ""
		if cfg != nil {
			agentDir = cfg.AgentDir
		}
		root, err := integrationKnowledgeRoot(agentDir)
		if err != nil {
			http.Error(w, "Failed to resolve knowledge dir: "+err.Error(), http.StatusInternalServerError)
			return
		}
		agentID := strings.TrimSpace(r.URL.Query().Get("agentId"))
		target := root
		if agentID != "" {
			target, err = integrationKnowledgeDirForAgent(agentDir, agentID)
			if err != nil {
				http.Error(w, "Failed to resolve agent knowledge dir: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if err := os.MkdirAll(target, 0o755); err != nil {
			http.Error(w, "Failed to create knowledge dir: "+err.Error(), http.StatusInternalServerError)
			return
		}
		info, err := os.Stat(target)
		if err != nil {
			http.Error(w, "Failed to access knowledge dir: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if !info.IsDir() {
			http.Error(w, "Knowledge path is not a directory", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, target)
	}
}

func resolveFileTarget(cfg *Config, agentID, filePath string) (string, error) {
	rawPath := normalizeQuotedPathArg(filePath)
	if rawPath == "" {
		return "", fmt.Errorf("file is required")
	}
	expandedPath, err := expandPath(rawPath)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(expandedPath) {
		resolved, err := resolveCaseInsensitive(expandedPath)
		if err != nil {
			return "", err
		}
		return resolved, nil
	}
	cleanRel := filepath.Clean(rawPath)
	if cleanRel == "." || cleanRel == "" {
		return "", fmt.Errorf("file is required")
	}
	if cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes workspace")
	}
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return "", fmt.Errorf("agentId is required for relative path")
	}
	workspace, err := getWorkspaceByAgentID(cfg, agentID)
	if err != nil {
		return "", err
	}
	resolved, err := resolveCaseInsensitiveUnderRoot(workspace, cleanRel)
	if err != nil {
		return "", err
	}
	if !ensurePathWithinRoot(workspace, resolved) {
		return "", fmt.Errorf("path escapes workspace")
	}
	return resolved, nil
}

func fileLastUpdateDuration(path string, now time.Time) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	diff := now.Sub(info.ModTime())
	return diff.Milliseconds(), nil
}

func handleFileLastUpdate(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		filePath := strings.TrimSpace(r.URL.Query().Get("file"))
		agentID := strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("agentId"), r.URL.Query().Get("agent")))
		resolved, err := resolveFileTarget(cfg, agentID, filePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		lastUpdate, err := fileLastUpdateDuration(resolved, time.Now())
		if err != nil {
			status := http.StatusInternalServerError
			if os.IsNotExist(err) {
				status = http.StatusNotFound
			}
			http.Error(w, err.Error(), status)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, strconv.FormatInt(lastUpdate, 10))
	}
}

type DataResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Status  int    `json:"status"`
}

var binaryExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true, ".ico": true, ".webp": true, ".svg": true, ".tiff": true, ".tif": true,
	".mp3": true, ".mp4": true, ".avi": true, ".mov": true, ".wmv": true, ".flv": true, ".mkv": true, ".webm": true, ".wav": true, ".flac": true, ".aac": true, ".ogg": true,
	".zip": true, ".tar": true, ".gz": true, ".rar": true, ".7z": true, ".bz2": true,
	".exe": true, ".dll": true, ".so": true, ".dylib": true, ".bin": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
	".ttf": true, ".otf": true, ".woff": true, ".woff2": true, ".eot": true,
}

func isBinaryFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return binaryExts[ext]
}

func writeDataResp(w http.ResponseWriter, resp DataResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleData() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		filePath := r.URL.Query().Get("path")
		if filePath == "" {
			writeDataResp(w, DataResponse{Path: "", Content: "path is required", Status: 1})
			return
		}
		resolved, err := resolveCaseInsensitive(filePath)
		if err != nil {
			writeDataResp(w, DataResponse{Path: filePath, Content: "文件不存在: " + err.Error(), Status: 1})
			return
		}
		info, err := os.Stat(resolved)
		if err != nil {
			writeDataResp(w, DataResponse{Path: resolved, Content: "无法访问: " + err.Error(), Status: 1})
			return
		}
		if info.IsDir() {
			writeDataResp(w, DataResponse{Path: resolved, Content: "不支持读取目录", Status: 1})
			return
		}
		if isBinaryFile(resolved) {
			writeDataResp(w, DataResponse{Path: resolved, Content: "不支持读取二进制文件", Status: 1})
			return
		}
		data, err := os.ReadFile(resolved)
		if err != nil {
			writeDataResp(w, DataResponse{Path: resolved, Content: "读取失败: " + err.Error(), Status: 1})
			return
		}
		writeDataResp(w, DataResponse{Path: resolved, Content: string(data), Status: 0})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Proxy: /api/workspace — get agent workspace path
// ═══════════════════════════════════════════════════════════════════════════

type urlPreviewProbeResponse struct {
	OK      bool   `json:"ok"`
	Content string `json:"content,omitempty"`
	URL     string `json:"url,omitempty"`
	Status  int    `json:"status"`
}

func writeURLPreviewProbeResponse(w http.ResponseWriter, resp urlPreviewProbeResponse) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func normalizeURLPreviewProbeTarget(raw string) (*url.URL, error) {
	target := strings.TrimSpace(raw)
	if target == "" {
		return nil, fmt.Errorf("url is required")
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("url is invalid: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("only http(s) url is supported")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return nil, fmt.Errorf("url host is required")
	}
	return parsed, nil
}

func urlPreviewProbeAppOrigin(r *http.Request) string {
	if r == nil {
		return ""
	}
	proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if proto == "" {
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := strings.TrimSpace(r.Host)
	if host == "" {
		return ""
	}
	return strings.ToLower(proto) + "://" + strings.ToLower(host)
}

func urlPreviewProbeOrigin(raw *url.URL) string {
	if raw == nil || strings.TrimSpace(raw.Scheme) == "" || strings.TrimSpace(raw.Host) == "" {
		return ""
	}
	return strings.ToLower(raw.Scheme) + "://" + strings.ToLower(raw.Host)
}

func urlPreviewProbeBlockedByXFrameOptions(value, appOrigin, targetOrigin string) bool {
	for _, part := range strings.Split(value, ",") {
		fields := strings.Fields(strings.ToLower(strings.TrimSpace(part)))
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "deny":
			return true
		case "sameorigin":
			return appOrigin == "" || appOrigin != targetOrigin
		case "allow-from":
			if len(fields) < 2 {
				return true
			}
			return appOrigin == "" || fields[1] != strings.ToLower(appOrigin)
		}
	}
	return false
}

func urlPreviewProbeFrameAncestorsValue(headers []string) string {
	for _, header := range headers {
		for _, directive := range strings.Split(header, ";") {
			directive = strings.TrimSpace(directive)
			if directive == "" {
				continue
			}
			fields := strings.Fields(directive)
			if len(fields) == 0 || strings.ToLower(fields[0]) != "frame-ancestors" {
				continue
			}
			if len(fields) == 1 {
				return "'none'"
			}
			return strings.Join(fields[1:], " ")
		}
	}
	return ""
}

func urlPreviewProbeFrameAncestorsAllows(value, appOrigin, targetOrigin string) bool {
	if strings.TrimSpace(value) == "" {
		return true
	}
	appOrigin = strings.ToLower(strings.TrimSpace(appOrigin))
	targetOrigin = strings.ToLower(strings.TrimSpace(targetOrigin))
	appURL, _ := url.Parse(appOrigin)
	appScheme := ""
	appHost := ""
	if appURL != nil {
		appScheme = strings.ToLower(strings.TrimSpace(appURL.Scheme))
		appHost = strings.ToLower(strings.TrimSpace(appURL.Hostname()))
	}
	for _, token := range strings.Fields(strings.ToLower(value)) {
		switch token {
		case "*":
			return true
		case "'self'":
			if appOrigin != "" && appOrigin == targetOrigin {
				return true
			}
		case "'none'":
			continue
		default:
			if strings.HasSuffix(token, ":") {
				if appScheme != "" && token == appScheme+":" {
					return true
				}
				continue
			}
			if appOrigin != "" && token == appOrigin {
				return true
			}
			if appHost != "" && strings.HasPrefix(token, "*.") {
				suffix := strings.TrimPrefix(token, "*.")
				if appHost == suffix || strings.HasSuffix(appHost, "."+suffix) {
					return true
				}
			}
		}
	}
	return false
}

func inspectURLPreviewTarget(ctx context.Context, client *http.Client, target *url.URL, appOrigin string) (string, string, error) {
	if target == nil {
		return "", "", fmt.Errorf("url is required")
	}
	if client == nil {
		client = &http.Client{}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	_, _ = io.CopyN(io.Discard, resp.Body, 1024)
	finalURL := target.String()
	if resp.Request != nil && resp.Request.URL != nil && strings.TrimSpace(resp.Request.URL.String()) != "" {
		finalURL = resp.Request.URL.String()
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return finalURL, fmt.Sprintf("目标网站返回 HTTP %d，无法预览", resp.StatusCode), nil
	}
	targetOrigin := ""
	if resp.Request != nil {
		targetOrigin = urlPreviewProbeOrigin(resp.Request.URL)
	}
	xfo := strings.TrimSpace(resp.Header.Get("X-Frame-Options"))
	if xfo != "" && urlPreviewProbeBlockedByXFrameOptions(xfo, appOrigin, targetOrigin) {
		return finalURL, "该站点禁止 iframe 预览（X-Frame-Options）", nil
	}
	frameAncestors := urlPreviewProbeFrameAncestorsValue(resp.Header.Values("Content-Security-Policy"))
	if frameAncestors != "" && !urlPreviewProbeFrameAncestorsAllows(frameAncestors, appOrigin, targetOrigin) {
		return finalURL, "该站点禁止 iframe 预览（CSP frame-ancestors）", nil
	}
	return finalURL, "", nil
}

func handleURLPreviewProbe(client *http.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		target, err := normalizeURLPreviewProbeTarget(r.URL.Query().Get("url"))
		if err != nil {
			writeURLPreviewProbeResponse(w, urlPreviewProbeResponse{Status: 1, OK: false, Content: err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		finalURL, reason, err := inspectURLPreviewTarget(ctx, client, target, urlPreviewProbeAppOrigin(r))
		if err != nil {
			writeURLPreviewProbeResponse(w, urlPreviewProbeResponse{
				Status:  1,
				OK:      false,
				Content: "目标网站无法访问：" + err.Error(),
				URL:     finalURL,
			})
			return
		}
		if strings.TrimSpace(reason) != "" {
			writeURLPreviewProbeResponse(w, urlPreviewProbeResponse{Status: 1, OK: false, Content: reason, URL: finalURL})
			return
		}
		writeURLPreviewProbeResponse(w, urlPreviewProbeResponse{Status: 0, OK: true, URL: finalURL})
	}
}

func handleWorkspace(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentID := r.URL.Query().Get("agentId")
		if agentID == "" {
			http.Error(w, "agentId is required", http.StatusBadRequest)
			return
		}
		agentTTL := time.Duration(cfg.AgentCacheMs) * time.Millisecond
		metadata, err := getAgentOutput(cfg.AgentDir, cfg.effectiveDeviceID(), agentTTL)
		if err != nil {
			log.Printf("api/workspace: agent scan error: %v", err)
			http.Error(w, "Failed to get agent metadata", http.StatusInternalServerError)
			return
		}
		for _, a := range metadata.Agents {
			if a.AgentID == agentID {
				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte(a.Workspace))
				return
			}
		}
		http.Error(w, "Agent not found: "+agentID, http.StatusNotFound)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Proxy: /api/edit — write file content under agent workspace
// ═══════════════════════════════════════════════════════════════════════════

type EditResponse struct {
	AgentID string `json:"agentId"`
	Path    string `json:"path"`
	SavedAs string `json:"savedAs,omitempty"`
	Content string `json:"content,omitempty"`
	Status  int    `json:"status"`
}

type EditRequest struct {
	Content string                 `json:"content"`
	Media   map[string]interface{} `json:"media,omitempty"`
}

func writeEditResp(w http.ResponseWriter, resp EditResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func appendTimestampToFilename(path string, ts time.Time) string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	stamp := ts.Format("20060102_150405")
	if ext == "" {
		return base + "_" + stamp
	}
	return base + "_" + stamp + ext
}

func handleEdit(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentID := r.URL.Query().Get("agentId")
		relPath := normalizeQuotedPathArg(r.URL.Query().Get("path"))
		if agentID == "" || relPath == "" {
			writeEditResp(w, EditResponse{AgentID: agentID, Path: relPath, Content: "agentId and path are required", Status: 1})
			return
		}
		if filepath.IsAbs(relPath) || strings.HasPrefix(relPath, "~") {
			writeEditResp(w, EditResponse{AgentID: agentID, Path: relPath, Content: "只支持相对路径", Status: 1})
			return
		}
		cleanRelPath := filepath.Clean(relPath)
		if cleanRelPath == "." || cleanRelPath == "" || cleanRelPath == ".." || strings.HasPrefix(cleanRelPath, ".."+string(filepath.Separator)) {
			writeEditResp(w, EditResponse{AgentID: agentID, Path: relPath, Content: "只支持 workspace 下的相对路径", Status: 1})
			return
		}
		saveAsNew := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("saveAsNew")), "true")
		agentTTL := time.Duration(cfg.AgentCacheMs) * time.Millisecond
		metadata, err := getAgentOutput(cfg.AgentDir, cfg.effectiveDeviceID(), agentTTL)
		if err != nil {
			writeEditResp(w, EditResponse{AgentID: agentID, Path: relPath, Content: "Agent 元数据获取失败: " + err.Error(), Status: 1})
			return
		}
		var workspace string
		for _, a := range metadata.Agents {
			if a.AgentID == agentID {
				workspace = a.Workspace
				break
			}
		}
		if workspace == "" {
			writeEditResp(w, EditResponse{AgentID: agentID, Path: relPath, Content: "Agent 不存在: " + agentID, Status: 1})
			return
		}
		targetRelPath := cleanRelPath
		if saveAsNew {
			targetRelPath = appendTimestampToFilename(cleanRelPath, time.Now())
		}
		resolved, err := resolveCaseInsensitiveUnderRoot(workspace, targetRelPath)
		if err != nil {
			writeEditResp(w, EditResponse{AgentID: agentID, Path: relPath, Content: "路径解析失败: " + err.Error(), Status: 1})
			return
		}
		if !ensurePathWithinRoot(workspace, resolved) {
			writeEditResp(w, EditResponse{AgentID: agentID, Path: relPath, Content: "只支持 workspace 下的相对路径", Status: 1})
			return
		}
		if info, statErr := os.Stat(resolved); statErr == nil && info.IsDir() {
			writeEditResp(w, EditResponse{AgentID: agentID, Path: relPath, Content: "不支持写入目录", Status: 1})
			return
		}
		body, err := io.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			writeEditResp(w, EditResponse{AgentID: agentID, Path: relPath, Content: "读取请求体失败", Status: 1})
			return
		}
		var req EditRequest
		if err := json.Unmarshal(body, &req); err != nil {
			writeEditResp(w, EditResponse{AgentID: agentID, Path: relPath, Content: "请求体格式错误", Status: 1})
			return
		}
		if err := os.MkdirAll(filepath.Dir(resolved), 0755); err != nil {
			writeEditResp(w, EditResponse{AgentID: agentID, Path: relPath, Content: "创建目录失败: " + err.Error(), Status: 1})
			return
		}
		var writeData []byte
		if isBinaryFile(resolved) {
			decoded, err := base64.StdEncoding.DecodeString(req.Content)
			if err != nil {
				writeEditResp(w, EditResponse{AgentID: agentID, Path: relPath, Content: "二进制内容不是合法的 base64", Status: 1})
				return
			}
			writeData = decoded
		} else {
			writeData = []byte(req.Content)
		}
		if err := os.WriteFile(resolved, writeData, 0644); err != nil {
			writeEditResp(w, EditResponse{AgentID: agentID, Path: relPath, Content: "写入失败: " + err.Error(), Status: 1})
			return
		}
		writeEditResp(w, EditResponse{AgentID: agentID, Path: relPath, SavedAs: resolved, Status: 0})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Proxy: /api/del — delete file under agent workspace
// ═══════════════════════════════════════════════════════════════════════════

type DelResponse struct {
	AgentID string `json:"agentId"`
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
	Status  int    `json:"status"`
}

func writeDelResp(w http.ResponseWriter, resp DelResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleDel(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentID := r.URL.Query().Get("agentId")
		relPath := normalizeQuotedPathArg(r.URL.Query().Get("path"))
		if agentID == "" || relPath == "" {
			writeDelResp(w, DelResponse{AgentID: agentID, Path: relPath, Content: "agentId and path are required", Status: 1})
			return
		}
		if filepath.IsAbs(relPath) || strings.HasPrefix(relPath, "~") {
			writeDelResp(w, DelResponse{AgentID: agentID, Path: relPath, Content: "只支持相对路径", Status: 1})
			return
		}
		agentTTL := time.Duration(cfg.AgentCacheMs) * time.Millisecond
		metadata, err := getAgentOutput(cfg.AgentDir, cfg.effectiveDeviceID(), agentTTL)
		if err != nil {
			writeDelResp(w, DelResponse{AgentID: agentID, Path: relPath, Content: "Agent 元数据获取失败: " + err.Error(), Status: 1})
			return
		}
		var workspace string
		for _, a := range metadata.Agents {
			if a.AgentID == agentID {
				workspace = a.Workspace
				break
			}
		}
		if workspace == "" {
			writeDelResp(w, DelResponse{AgentID: agentID, Path: relPath, Content: "Agent 不存在: " + agentID, Status: 1})
			return
		}
		fullPath := filepath.Join(workspace, relPath)
		resolved, err := resolveCaseInsensitive(fullPath)
		if err != nil {
			writeDelResp(w, DelResponse{AgentID: agentID, Path: relPath, Content: "不存在: " + err.Error(), Status: 1})
			return
		}
		if _, err := os.Stat(resolved); err != nil {
			writeDelResp(w, DelResponse{AgentID: agentID, Path: relPath, Content: "不存在: " + err.Error(), Status: 1})
			return
		}
		if err := os.RemoveAll(resolved); err != nil {
			writeDelResp(w, DelResponse{AgentID: agentID, Path: relPath, Content: "删除失败: " + err.Error(), Status: 1})
			return
		}
		writeDelResp(w, DelResponse{AgentID: agentID, Path: relPath, Status: 0})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Proxy: /api/raw — read file binary as base64
// ═══════════════════════════════════════════════════════════════════════════

type RawResponse struct {
	AgentID string `json:"agentId"`
	Path    string `json:"path"`
	Content string `json:"content"`
	Status  int    `json:"status"`
}

func writeRawResp(w http.ResponseWriter, resp RawResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleRaw(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentID := r.URL.Query().Get("agentId")
		pathParam := normalizeQuotedPathArg(r.URL.Query().Get("path"))
		if agentID == "" || pathParam == "" {
			writeRawResp(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "agentId and path are required", Status: 1})
			return
		}
		var resolved string
		expandedPath, err := expandPath(pathParam)
		if err != nil {
			writeRawResp(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "路径解析失败: " + err.Error(), Status: 1})
			return
		}
		if filepath.IsAbs(expandedPath) {
			resolvedPath, resolveErr := resolveCaseInsensitive(expandedPath)
			if resolveErr != nil {
				writeRawResp(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "文件不存在: " + resolveErr.Error(), Status: 1})
				return
			}
			resolved = resolvedPath
		} else {
			cleanRelPath := filepath.Clean(pathParam)
			if cleanRelPath == "." || cleanRelPath == "" || cleanRelPath == ".." || strings.HasPrefix(cleanRelPath, ".."+string(filepath.Separator)) {
				writeRawResp(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "只支持 workspace 下的相对路径", Status: 1})
				return
			}
			agentTTL := time.Duration(cfg.AgentCacheMs) * time.Millisecond
			metadata, err := getAgentOutput(cfg.AgentDir, cfg.effectiveDeviceID(), agentTTL)
			if err != nil {
				writeRawResp(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "Agent 元数据获取失败: " + err.Error(), Status: 1})
				return
			}
			var workspace string
			for _, a := range metadata.Agents {
				if a.AgentID == agentID {
					workspace = a.Workspace
					break
				}
			}
			if workspace == "" {
				writeRawResp(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "Agent 不存在: " + agentID, Status: 1})
				return
			}
			resolved, err = resolveCaseInsensitiveUnderRoot(workspace, cleanRelPath)
			if err != nil {
				writeRawResp(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "文件不存在: " + err.Error(), Status: 1})
				return
			}
			if !ensurePathWithinRoot(workspace, resolved) {
				writeRawResp(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "只支持 workspace 下的相对路径", Status: 1})
				return
			}
		}
		info, err := os.Stat(resolved)
		if err != nil {
			writeRawResp(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "文件不存在: " + err.Error(), Status: 1})
			return
		}
		if info.IsDir() {
			writeRawResp(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "不支持读取目录", Status: 1})
			return
		}
		data, err := os.ReadFile(resolved)
		if err != nil {
			writeRawResp(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "读取失败: " + err.Error(), Status: 1})
			return
		}
		encoded := base64.StdEncoding.EncodeToString(data)
		writeRawResp(w, RawResponse{AgentID: agentID, Path: pathParam, Content: encoded, Status: 0})
	}
}

func syncIntegrationSkillsWarnings(cfg *Config) error {
	agentDir, err := resolveIntegrationAgentDir(cfg.AgentDir)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(agentDir)
	if err != nil {
		return err
	}
	var warnings []skillscore.SkillWarning
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillsDir := filepath.Join(agentDir, entry.Name(), "skills")
		info, err := os.Stat(skillsDir)
		if err != nil || !info.IsDir() {
			continue
		}
		items, err := skillscore.ScanWarnings(skillsDir)
		if err != nil {
			return err
		}
		warnings = append(warnings, items...)
	}
	store, err := skillscore.OpenWarningStore(resolveIntegrationDBPath())
	if err != nil {
		return err
	}
	return store.Sync(warnings)
}

func handleSkillsWarning(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		refresh := strings.TrimSpace(r.URL.Query().Get("refresh"))
		if refresh != "" && refresh != "0" && strings.ToLower(refresh) != "false" {
			if err := syncIntegrationSkillsWarnings(cfg); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": err.Error()})
				return
			}
		}

		store, err := skillscore.OpenWarningStore(resolveIntegrationDBPath())
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": err.Error()})
			return
		}
		items, err := store.List()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": 0, "data": items})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Heartbeat status tracking
// ═══════════════════════════════════════════════════════════════════════════

var hbMu sync.Mutex
var hbLastTimestamp int64
var hbLastStatus int // 0=success no task, 1=failure, 2=success with task
var hbLastError string

var cliGetRuntimeMu sync.RWMutex
var cliGetTaskQueue chan queuedCliGetTask
var cliGetPublishQueue chan queuedCliGetPublish
var cliGetRunningWorkers int64
var cliGetActivityMu sync.Mutex

const cliGetRuntimeAccumulationWindow = 5 * time.Second

type cliGetRuntimeSnapshot struct {
	PendingPublishes int
	RunningWorkers   int
	PendingTasks     int
}

type cliGetRuntimeEvent struct {
	At time.Time
	cliGetRuntimeSnapshot
}

func setCliGetRuntimeQueues(taskQueue chan queuedCliGetTask, publishQueue chan queuedCliGetPublish) {
	cliGetRuntimeMu.Lock()
	cliGetTaskQueue = taskQueue
	cliGetPublishQueue = publishQueue
	cliGetRuntimeMu.Unlock()
}

func snapshotCliGetRuntimeQueues() (pendingPublishes int, pendingTasks int) {
	cliGetRuntimeMu.RLock()
	taskQueue := cliGetTaskQueue
	publishQueue := cliGetPublishQueue
	cliGetRuntimeMu.RUnlock()
	if publishQueue != nil {
		pendingPublishes = len(publishQueue)
	}
	if taskQueue != nil {
		pendingTasks = len(taskQueue)
	}
	return pendingPublishes, pendingTasks
}

var cliGetRecentActivity []cliGetRuntimeEvent

func trimCliGetRuntimeEventsLocked(now time.Time) {
	if len(cliGetRecentActivity) == 0 {
		return
	}
	cutoff := now.Add(-cliGetRuntimeAccumulationWindow)
	trimIndex := 0
	for trimIndex < len(cliGetRecentActivity) && cliGetRecentActivity[trimIndex].At.Before(cutoff) {
		trimIndex++
	}
	if trimIndex <= 0 {
		return
	}
	cliGetRecentActivity = append([]cliGetRuntimeEvent(nil), cliGetRecentActivity[trimIndex:]...)
}

func recordCliGetRuntimeEvent(snapshot cliGetRuntimeSnapshot) {
	if snapshot.PendingPublishes <= 0 && snapshot.RunningWorkers <= 0 && snapshot.PendingTasks <= 0 {
		return
	}
	cliGetActivityMu.Lock()
	now := time.Now()
	trimCliGetRuntimeEventsLocked(now)
	cliGetRecentActivity = append(cliGetRecentActivity, cliGetRuntimeEvent{
		At:                    now,
		cliGetRuntimeSnapshot: snapshot,
	})
	cliGetActivityMu.Unlock()
}

func heartbeatCliGetRuntimeSnapshot() cliGetRuntimeSnapshot {
	cliGetActivityMu.Lock()
	now := time.Now()
	trimCliGetRuntimeEventsLocked(now)
	var total cliGetRuntimeSnapshot
	for _, item := range cliGetRecentActivity {
		total.PendingPublishes += item.PendingPublishes
		total.RunningWorkers += item.RunningWorkers
		total.PendingTasks += item.PendingTasks
	}
	defer cliGetActivityMu.Unlock()
	return total
}

func setCliGetRunningWorkers(count int64) {
	atomic.StoreInt64(&cliGetRunningWorkers, count)
}

func changeCliGetRunningWorkers(delta int64) int64 {
	count := atomic.AddInt64(&cliGetRunningWorkers, delta)
	if delta > 0 {
		recordCliGetRuntimeEvent(cliGetRuntimeSnapshot{RunningWorkers: int(delta)})
	}
	return count
}

func cliGetRunningWorkerCount() int {
	count := atomic.LoadInt64(&cliGetRunningWorkers)
	if count < 0 {
		return 0
	}
	return int(count)
}

func resetCliGetRuntimeStatus() {
	setCliGetRuntimeQueues(nil, nil)
	setCliGetRunningWorkers(0)
	cliGetActivityMu.Lock()
	cliGetRecentActivity = nil
	cliGetActivityMu.Unlock()
}

// recordHeartbeat records status: 0=success no task, 1=failure, 2=success with task
func recordHeartbeat(status int, lastError string) {
	hbMu.Lock()
	defer hbMu.Unlock()
	hbLastTimestamp = time.Now().UnixMilli()
	hbLastStatus = status
	hbLastError = strings.TrimSpace(lastError)
}

func handleHeartbeat() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		snapshot := heartbeatCliGetRuntimeSnapshot()
		hbMu.Lock()
		resp := struct {
			LastTimestamp    int64  `json:"lastTimestamp"`
			Status           int    `json:"status"`
			LastError        string `json:"lastError"`
			PendingPublishes int    `json:"pendingPublishes"`
			RunningWorkers   int    `json:"runningWorkers"`
			PendingTasks     int    `json:"pendingTasks"`
		}{
			LastTimestamp:    hbLastTimestamp,
			Status:           hbLastStatus,
			LastError:        hbLastError,
			PendingPublishes: snapshot.PendingPublishes,
			RunningWorkers:   snapshot.RunningWorkers,
			PendingTasks:     snapshot.PendingTasks,
		}
		hbMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func handleLogCleanupStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		snapshot := integrationLogRetentionManager.Snapshot()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":                 0,
			"checked":                snapshot.Checked,
			"running":                snapshot.Running,
			"message":                firstNonEmpty(snapshot.Message, logretention.DefaultMessage),
			"retentionDays":          snapshot.RetentionDays,
			"cutoff":                 snapshot.Cutoff,
			"startedAt":              snapshot.StartedAt,
			"finishedAt":             snapshot.FinishedAt,
			"deletedAgentMessageLog": snapshot.DeletedAgentMessageLog,
			"deletedChatLog":         snapshot.DeletedChatLog,
			"error":                  snapshot.Error,
		})
	}
}

func handleRuntimeHost(cfg *Config) http.HandlerFunc {
	type hostRequest struct {
		Host  string `json:"host"`
		Reset bool   `json:"reset"`
	}
	writeJSON := func(w http.ResponseWriter, statusCode int, payload map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(payload)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !isLocalManagementRequest(r) {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"status":  1,
				"content": "only localhost requests can modify runtime host",
			})
			return
		}

		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{"status": 0, "data": cfg.runtimeHostSnapshot()})
			return
		case http.MethodDelete:
			snapshot := cfg.resetRuntimeHost()
			log.Printf("runtime host reset to startup value: %s", snapshot.StartupHost)
			writeJSON(w, http.StatusOK, map[string]any{"status": 0, "data": snapshot})
			return
		case http.MethodPost, http.MethodPut:
			var payload hostRequest
			if r.Body != nil {
				defer r.Body.Close()
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
					writeJSON(w, http.StatusBadRequest, map[string]any{
						"status":  1,
						"content": "invalid JSON body",
					})
					return
				}
			}
			if raw := strings.TrimSpace(r.URL.Query().Get("host")); raw != "" {
				payload.Host = raw
			}
			if raw := strings.TrimSpace(r.URL.Query().Get("reset")); raw != "" {
				if value, ok := sharedutil.ToBoolValue(raw); ok {
					payload.Reset = value
				}
			}
			if payload.Reset {
				snapshot := cfg.resetRuntimeHost()
				log.Printf("runtime host reset to startup value: %s", snapshot.StartupHost)
				writeJSON(w, http.StatusOK, map[string]any{"status": 0, "data": snapshot})
				return
			}
			snapshot, err := cfg.setRuntimeHost(payload.Host)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"status":  1,
					"content": err.Error(),
				})
				return
			}
			log.Printf("runtime host updated: %s", snapshot.Host)
			writeJSON(w, http.StatusOK, map[string]any{"status": 0, "data": snapshot})
			return
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}
}

func parseStandaloneUpdateValue(value string) (bool, error) {
	enabled, ok := sharedutil.ToBoolValue(value)
	if !ok {
		return false, fmt.Errorf("standalone must be true or false")
	}
	return enabled, nil
}

func standaloneValueFromRequestPath(r *http.Request) (bool, bool, error) {
	if r == nil || r.URL == nil {
		return false, false, fmt.Errorf("standalone request is invalid")
	}
	path := strings.TrimSpace(r.URL.Path)
	switch {
	case path == integrationstandalone.Endpoint+"=true":
		return true, true, nil
	case path == integrationstandalone.Endpoint+"=false":
		return false, true, nil
	case strings.HasPrefix(path, integrationstandalone.Endpoint+"="):
		enabled, err := parseStandaloneUpdateValue(strings.TrimPrefix(path, integrationstandalone.Endpoint+"="))
		if err != nil {
			return false, false, err
		}
		return enabled, true, nil
	case path == integrationstandalone.Endpoint:
		return false, false, nil
	default:
		return false, false, fmt.Errorf("unsupported standalone path: %s", path)
	}
}

func normalizeIntegrationRequestHost(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.Contains(value, ",") {
		value = strings.TrimSpace(strings.Split(value, ",")[0])
	}
	if strings.Contains(value, "://") {
		if parsed, err := url.Parse(value); err == nil {
			value = strings.TrimSpace(parsed.Host)
		}
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	return strings.ToLower(strings.TrimSpace(value))
}

func integrationRequestAccessHost(r *http.Request) string {
	if r == nil {
		return ""
	}
	for _, candidate := range []string{
		r.Header.Get("X-Forwarded-Host"),
		r.Header.Get("X-Original-Host"),
		r.Host,
		func() string {
			if r.URL == nil {
				return ""
			}
			return r.URL.Host
		}(),
	} {
		if host := normalizeIntegrationRequestHost(candidate); host != "" {
			return host
		}
	}
	return ""
}

func isIntegrationLocalAccessHost(host string) bool {
	switch normalizeIntegrationRequestHost(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func isLocalManagementRequest(r *http.Request) bool {
	if !sharedutil.IsLocalExecutionRequest(r) {
		return false
	}
	host := integrationRequestAccessHost(r)
	if host == "" {
		return true
	}
	return isIntegrationLocalAccessHost(host)
}

func closeStandaloneConnection(w http.ResponseWriter) {
	if w == nil {
		return
	}
	conn, _, err := http.NewResponseController(w).Hijack()
	if err != nil {
		log.Printf("standalone close connection failed: %v", err)
		return
	}
	_ = conn.Close()
}

func isStandaloneProtectedPath(path string) bool {
	return strings.TrimSpace(path) != ""
}

func withStandaloneAPIProtection(next http.Handler, cfg *Config) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg != nil && cfg.standaloneEnabled() && isStandaloneProtectedPath(r.URL.Path) && !isLocalManagementRequest(r) {
			closeStandaloneConnection(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleRuntimeStandalone(cfg *Config) http.HandlerFunc {
	type standaloneRequest struct {
		Enabled *bool `json:"enabled"`
		Value   *bool `json:"value"`
		Reset   bool  `json:"reset"`
	}
	writeJSON := func(w http.ResponseWriter, statusCode int, payload map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(payload)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !isLocalManagementRequest(r) {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"status":  1,
				"content": "only localhost/127.0.0.1/::1 requests can modify standalone mode",
			})
			return
		}

		if enabled, shouldSet, err := standaloneValueFromRequestPath(r); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"status":  1,
				"content": err.Error(),
			})
			return
		} else if shouldSet {
			snapshot := cfg.setStandaloneEnabled(enabled)
			log.Printf("runtime standalone updated: enabled=%t", snapshot.Enabled)
			writeJSON(w, http.StatusOK, map[string]any{"status": 0, "data": snapshot})
			return
		}

		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{"status": 0, "data": cfg.standaloneSnapshot()})
			return
		case http.MethodDelete:
			snapshot := cfg.resetStandalone()
			log.Printf("runtime standalone reset to startup value: enabled=%t", snapshot.StartupEnabled)
			writeJSON(w, http.StatusOK, map[string]any{"status": 0, "data": snapshot})
			return
		case http.MethodPost, http.MethodPut:
			var payload standaloneRequest
			if r.Body != nil {
				defer r.Body.Close()
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
					writeJSON(w, http.StatusBadRequest, map[string]any{
						"status":  1,
						"content": "invalid JSON body",
					})
					return
				}
			}
			if payload.Enabled == nil {
				payload.Enabled = payload.Value
			}
			if raw := strings.TrimSpace(r.URL.Query().Get("enabled")); raw != "" {
				enabled, err := parseStandaloneUpdateValue(raw)
				if err != nil {
					writeJSON(w, http.StatusBadRequest, map[string]any{
						"status":  1,
						"content": err.Error(),
					})
					return
				}
				payload.Enabled = &enabled
			}
			if payload.Enabled == nil {
				if raw := strings.TrimSpace(r.URL.Query().Get("value")); raw != "" {
					enabled, err := parseStandaloneUpdateValue(raw)
					if err != nil {
						writeJSON(w, http.StatusBadRequest, map[string]any{
							"status":  1,
							"content": err.Error(),
						})
						return
					}
					payload.Enabled = &enabled
				}
			}
			if raw := strings.TrimSpace(r.URL.Query().Get("reset")); raw != "" {
				if value, ok := sharedutil.ToBoolValue(raw); ok {
					payload.Reset = value
				}
			}
			if payload.Reset {
				snapshot := cfg.resetStandalone()
				log.Printf("runtime standalone reset to startup value: enabled=%t", snapshot.StartupEnabled)
				writeJSON(w, http.StatusOK, map[string]any{"status": 0, "data": snapshot})
				return
			}
			if payload.Enabled == nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"status":  1,
					"content": "standalone enabled is required",
				})
				return
			}
			snapshot := cfg.setStandaloneEnabled(*payload.Enabled)
			log.Printf("runtime standalone updated: enabled=%t", snapshot.Enabled)
			writeJSON(w, http.StatusOK, map[string]any{"status": 0, "data": snapshot})
			return
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}
}

func handleLANAccessURL(cfg *Config) http.HandlerFunc {
	writeJSON := func(w http.ResponseWriter, statusCode int, payload map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(payload)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !isLocalManagementRequest(r) {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"status":  1,
				"content": "only localhost requests can query LAN access URL",
			})
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		host, err := detectLANAccessHost()
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"status":  1,
				"content": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": 0,
			"data": map[string]any{
				"host": host,
				"url":  buildLANAccessURL(host, cfg.Port),
			},
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Proxy: /api/agent/init — create new agent directory
// ═══════════════════════════════════════════════════════════════════════════

func copyDirRecursiveFiltered(src, dst string, keep func(rel string, d fs.DirEntry) bool) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if keep != nil && !keep(rel, d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			info, infoErr := d.Info()
			if infoErr != nil {
				return infoErr
			}
			return os.MkdirAll(target, info.Mode().Perm())
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			return infoErr
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}

func copyDirRecursive(src, dst string) error {
	return copyDirRecursiveFiltered(src, dst, nil)
}

func shouldCopyDefaultAgentTemplateEntry(rel string, d fs.DirEntry) bool {
	if rel == "." {
		return true
	}
	if d.IsDir() {
		return true
	}
	return filepath.Clean(rel) != "config.json"
}

func writeEmptyAgentConfig(dst string) error {
	configPath := filepath.Join(dst, "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(configPath, []byte("{}\n"), 0o644)
}

func writeAgentJSONResponse(w http.ResponseWriter, statusCode int, payload map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func normalizeImportedRelativePath(name string) (string, error) {
	normalized := strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	normalized = strings.TrimPrefix(normalized, "./")
	normalized = strings.TrimPrefix(normalized, "/")
	if normalized == "" {
		return "", fmt.Errorf("import path is required")
	}
	cleaned := path.Clean(normalized)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("import path escapes destination: %s", name)
	}
	return cleaned, nil
}

func importRootNameFromRelativePath(relPath string) (string, error) {
	parts := strings.Split(strings.TrimSpace(relPath), "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return "", fmt.Errorf("imported archive must contain a top-level agent directory")
	}
	rootName := strings.TrimSpace(parts[0])
	if err := integrationagentarchive.ValidateAgentID(rootName); err != nil {
		return "", err
	}
	return rootName, nil
}

func handleAgentExport(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentID := firstNonEmpty(r.URL.Query().Get("agent_id"), r.URL.Query().Get("agentId"))
		if err := integrationagentarchive.ValidateAgentID(agentID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="`+strings.TrimSpace(agentID)+`.zip"`)
		if err := integrationagentarchive.Export(cfg.AgentDir, agentID, w); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
}

func importUploadedZip(agentRootDir string, fileHeader *multipart.FileHeader) (integrationagentarchive.Result, error) {
	if fileHeader == nil {
		return integrationagentarchive.Result{}, fmt.Errorf("zip file is required")
	}
	stagingDir, err := os.MkdirTemp(agentRootDir, ".agent-import-zip-*")
	if err != nil {
		return integrationagentarchive.Result{}, fmt.Errorf("create zip staging dir: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	src, err := fileHeader.Open()
	if err != nil {
		return integrationagentarchive.Result{}, fmt.Errorf("open uploaded zip: %w", err)
	}
	defer src.Close()

	zipPath := filepath.Join(stagingDir, "agent.zip")
	dst, err := os.Create(zipPath)
	if err != nil {
		return integrationagentarchive.Result{}, fmt.Errorf("create zip staging file: %w", err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return integrationagentarchive.Result{}, fmt.Errorf("save uploaded zip: %w", err)
	}
	if err := dst.Close(); err != nil {
		return integrationagentarchive.Result{}, fmt.Errorf("close uploaded zip: %w", err)
	}
	return integrationagentarchive.ImportZipFile(agentRootDir, zipPath)
}

func importUploadedDirectory(agentRootDir string, fileHeaders []*multipart.FileHeader, paths []string, emptyDirs []string) (integrationagentarchive.Result, error) {
	if len(fileHeaders) == 0 && len(emptyDirs) == 0 {
		return integrationagentarchive.Result{}, fmt.Errorf("directory upload is empty")
	}

	stagingDir, err := os.MkdirTemp(agentRootDir, ".agent-import-dir-*")
	if err != nil {
		return integrationagentarchive.Result{}, fmt.Errorf("create directory staging dir: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	rootName := ""
	recordRoot := func(relPath string) error {
		currentRoot, err := importRootNameFromRelativePath(relPath)
		if err != nil {
			return err
		}
		if rootName == "" {
			rootName = currentRoot
			return nil
		}
		if rootName != currentRoot {
			return fmt.Errorf("directory import must contain exactly one top-level agent directory")
		}
		return nil
	}

	for i, fileHeader := range fileHeaders {
		rawPath := fileHeader.Filename
		if i < len(paths) && strings.TrimSpace(paths[i]) != "" {
			rawPath = paths[i]
		}
		relPath, err := normalizeImportedRelativePath(rawPath)
		if err != nil {
			return integrationagentarchive.Result{}, err
		}
		if err := recordRoot(relPath); err != nil {
			return integrationagentarchive.Result{}, err
		}

		target := filepath.Join(stagingDir, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return integrationagentarchive.Result{}, fmt.Errorf("create staging parent: %w", err)
		}
		src, err := fileHeader.Open()
		if err != nil {
			return integrationagentarchive.Result{}, fmt.Errorf("open uploaded file %s: %w", relPath, err)
		}
		dst, err := os.Create(target)
		if err != nil {
			src.Close()
			return integrationagentarchive.Result{}, fmt.Errorf("create staging file %s: %w", relPath, err)
		}
		if _, err := io.Copy(dst, src); err != nil {
			dst.Close()
			src.Close()
			return integrationagentarchive.Result{}, fmt.Errorf("save uploaded file %s: %w", relPath, err)
		}
		if err := dst.Close(); err != nil {
			src.Close()
			return integrationagentarchive.Result{}, fmt.Errorf("close staging file %s: %w", relPath, err)
		}
		if err := src.Close(); err != nil {
			return integrationagentarchive.Result{}, fmt.Errorf("close uploaded file %s: %w", relPath, err)
		}
	}

	for _, rawPath := range emptyDirs {
		relPath, err := normalizeImportedRelativePath(rawPath)
		if err != nil {
			return integrationagentarchive.Result{}, err
		}
		if err := recordRoot(relPath); err != nil {
			return integrationagentarchive.Result{}, err
		}
		target := filepath.Join(stagingDir, filepath.FromSlash(relPath))
		if err := os.MkdirAll(target, 0755); err != nil {
			return integrationagentarchive.Result{}, fmt.Errorf("create staging directory %s: %w", relPath, err)
		}
	}

	if rootName == "" {
		return integrationagentarchive.Result{}, fmt.Errorf("directory import must contain a top-level agent directory")
	}
	return integrationagentarchive.ImportDirectory(agentRootDir, filepath.Join(stagingDir, rootName))
}

func handleAgentImport(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := os.MkdirAll(cfg.AgentDir, 0755); err != nil {
			writeAgentJSONResponse(w, http.StatusInternalServerError, map[string]interface{}{
				"status":  1,
				"content": "failed to prepare agent-dir: " + err.Error(),
			})
			return
		}
		if err := r.ParseMultipartForm(512 << 20); err != nil {
			writeAgentJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
				"status":  1,
				"content": "parse form failed: " + err.Error(),
			})
			return
		}

		fileHeaders := r.MultipartForm.File["files"]
		if len(fileHeaders) == 0 {
			writeAgentJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
				"status":  1,
				"content": "files are required",
			})
			return
		}

		var paths []string
		if raw := strings.TrimSpace(r.FormValue("pathsJson")); raw != "" {
			_ = json.Unmarshal([]byte(raw), &paths)
		}
		var emptyDirs []string
		if raw := strings.TrimSpace(r.FormValue("emptyDirsJson")); raw != "" {
			_ = json.Unmarshal([]byte(raw), &emptyDirs)
		}

		sourceKind := strings.ToLower(strings.TrimSpace(r.FormValue("sourceKind")))
		useZipImport := sourceKind == "zip"
		if sourceKind == "" && len(fileHeaders) == 1 {
			name := strings.TrimSpace(fileHeaders[0].Filename)
			if len(paths) > 0 && strings.TrimSpace(paths[0]) != "" {
				name = strings.TrimSpace(paths[0])
			}
			useZipImport = strings.EqualFold(filepath.Ext(name), ".zip")
		}

		var (
			result integrationagentarchive.Result
			err    error
		)
		if useZipImport {
			if len(fileHeaders) != 1 {
				writeAgentJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
					"status":  1,
					"content": "zip import expects exactly one zip file",
				})
				return
			}
			result, err = importUploadedZip(cfg.AgentDir, fileHeaders[0])
		} else {
			result, err = importUploadedDirectory(cfg.AgentDir, fileHeaders, paths, emptyDirs)
		}
		if err != nil {
			writeAgentJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
				"status":  1,
				"content": err.Error(),
			})
			return
		}

		flushAgentCache()
		writeAgentJSONResponse(w, http.StatusOK, map[string]interface{}{
			"status":  0,
			"agentId": result.AgentID,
			"path":    result.AgentPath,
			"source":  result.Source,
		})
	}
}

func resolveAgentDefaultDir(defaultDir string) string {
	defaultDir = strings.TrimSpace(defaultDir)
	if defaultDir != "" {
		return defaultDir
	}
	if value, ok := readIntegrationStartupConfigValue("default-dir"); ok {
		return value
	}
	appDir := integrationResourcesDir()
	if appDir == "" {
		return ""
	}
	return filepath.Join(appDir, "config")
}

func handleAgentInit(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := r.URL.Query().Get("name")
		if name == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "name is required"})
			return
		}
		if strings.ContainsAny(name, " /\\:*?\"<>|") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "name contains invalid characters"})
			return
		}
		agentDir := filepath.Join(cfg.AgentDir, name)
		if _, err := os.Stat(agentDir); err == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "agent already exists: " + name})
			return
		}
		if err := os.MkdirAll(agentDir, 0755); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "failed to create directory: " + err.Error()})
			return
		}
		defaultDir := resolveAgentDefaultDir(cfg.DefaultDir)
		if err := copyDefaultAgentTemplate(defaultDir, agentDir); err != nil {
			_ = os.RemoveAll(agentDir)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		flushAgentCache()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "name": name, "path": agentDir})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Proxy: /api/copy — copy managed agent files into an existing target agent
// ═══════════════════════════════════════════════════════════════════════════

func handleAgentCopy(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		sourceAgentID := firstNonEmpty(
			r.URL.Query().Get("source_agentId"),
			r.URL.Query().Get("sourceAgentId"),
			r.URL.Query().Get("source_agent_id"),
		)
		targetAgentID := firstNonEmpty(
			r.URL.Query().Get("target_agentId"),
			r.URL.Query().Get("targetAgentId"),
			r.URL.Query().Get("target_agent_id"),
		)
		knowledgeRoot, err := integrationKnowledgeRoot(cfg.AgentDir)
		if err != nil {
			writeAgentJSONResponse(w, http.StatusInternalServerError, map[string]interface{}{
				"status":  1,
				"content": "failed to resolve knowledge dir: " + err.Error(),
			})
			return
		}
		result, err := integrationagentcopy.Copy(cfg.AgentDir, knowledgeRoot, sourceAgentID, targetAgentID)
		if err != nil {
			writeAgentJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
				"status":  1,
				"content": err.Error(),
			})
			return
		}
		flushAgentCache()
		payload := map[string]interface{}{
			"status":        0,
			"sourceAgentId": result.SourceAgentID,
			"targetAgentId": result.TargetAgentID,
			"path":          result.AgentPath,
		}
		if strings.TrimSpace(result.KnowledgePath) != "" {
			payload["knowledgePath"] = result.KnowledgePath
		}
		if len(result.Copied) > 0 {
			payload["copied"] = result.Copied
		}
		writeAgentJSONResponse(w, http.StatusOK, payload)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Proxy: /api/agent/delete — delete agent directory
// ═══════════════════════════════════════════════════════════════════════════

func handleAgentDelete(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := r.URL.Query().Get("name")
		if name == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "name is required"})
			return
		}
		if strings.ContainsAny(name, " /\\:*?\"<>|") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "name contains invalid characters"})
			return
		}
		agentDir := filepath.Join(cfg.AgentDir, name)
		info, err := os.Stat(agentDir)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "stat agent failed: " + err.Error()})
			return
		}
		if err == nil && !info.IsDir() {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "agent not found: " + name})
			return
		}
		if err == nil {
			if err := os.RemoveAll(agentDir); err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "delete failed: " + err.Error()})
				return
			}
		}
		if cronDB != nil {
			if _, _, cleanupErr := deleteAgentCronData(cronDB, name); cleanupErr != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "cleanup cron failed: " + cleanupErr.Error()})
				return
			}
		}
		flushAgentCache()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "name": name})
	}
}

func normalizeAgentCreatePath(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("name is required")
	}
	if strings.HasPrefix(trimmed, "~") {
		return "", fmt.Errorf("name escapes workspace")
	}
	parts := strings.Split(trimmed, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("name escapes workspace")
		}
		if strings.ContainsAny(part, " \\:*?\"<>|") {
			return "", fmt.Errorf("name contains invalid characters")
		}
	}
	cleaned := filepath.Clean(trimmed)
	if cleaned == "." || cleaned == "" || filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("name escapes workspace")
	}
	return cleaned, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Proxy: /api/agent/create — create file or dir under agent workspace
// ═══════════════════════════════════════════════════════════════════════════

func handleAgentCreate(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentID := r.URL.Query().Get("agentId")
		name := r.URL.Query().Get("name")
		createType := r.URL.Query().Get("type")
		if agentID == "" || name == "" || createType == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "agentId, name and type are required"})
			return
		}
		cleanName, err := normalizeAgentCreatePath(name)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		agentTTL := time.Duration(cfg.AgentCacheMs) * time.Millisecond
		metadata, err := getAgentOutput(cfg.AgentDir, cfg.effectiveDeviceID(), agentTTL)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "agent metadata error: " + err.Error()})
			return
		}
		var workspace string
		for _, a := range metadata.Agents {
			if a.AgentID == agentID {
				workspace = a.Workspace
				break
			}
		}
		if workspace == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "agent not found: " + agentID})
			return
		}
		target := filepath.Join(workspace, cleanName)
		if _, err := os.Stat(target); err == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "already exists: " + cleanName})
			return
		}
		if createType == "0" {
			if err := os.MkdirAll(target, 0755); err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "create dir failed: " + err.Error()})
				return
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "create parent dir failed: " + err.Error()})
				return
			}
			if err := os.WriteFile(target, []byte(""), 0644); err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "create file failed: " + err.Error()})
				return
			}
		}
		flushAgentCache()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "agentId": agentID, "name": cleanName, "type": createType})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Proxy: /api/upload — upload files to agent tmp
// ═══════════════════════════════════════════════════════════════════════════

func handleUpload(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentID := r.URL.Query().Get("agentId")
		if agentID == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "agentId is required"})
			return
		}
		agentTTL := time.Duration(cfg.AgentCacheMs) * time.Millisecond
		metadata, err := getAgentOutput(cfg.AgentDir, cfg.effectiveDeviceID(), agentTTL)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "agent metadata error: " + err.Error()})
			return
		}
		var workspace string
		for _, a := range metadata.Agents {
			if a.AgentID == agentID {
				workspace = a.Workspace
				break
			}
		}
		if workspace == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "agent not found: " + agentID})
			return
		}
		tmpDir := filepath.Join(workspace, "tmp")
		os.MkdirAll(tmpDir, 0755)
		if err := r.ParseMultipartForm(200 << 20); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "parse form failed: " + err.Error()})
			return
		}
		var uploaded []string
		var paths []string
		if pj := r.MultipartForm.Value["pathsJson"]; len(pj) > 0 {
			json.Unmarshal([]byte(pj[0]), &paths)
		}
		fileHeaders := r.MultipartForm.File["files"]
		for i, fh := range fileHeaders {
			relPath := fh.Filename
			if i < len(paths) && paths[i] != "" {
				relPath = paths[i]
			}
			destPath := filepath.Join(tmpDir, relPath)
			os.MkdirAll(filepath.Dir(destPath), 0755)
			src, err := fh.Open()
			if err != nil {
				continue
			}
			data, err := io.ReadAll(src)
			src.Close()
			if err != nil {
				continue
			}
			os.WriteFile(destPath, data, 0644)
			uploaded = append(uploaded, relPath)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "agentId": agentID, "files": uploaded, "dest": tmpDir})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Proxy: /api/config — create or update config.json
// ═══════════════════════════════════════════════════════════════════════════

type AgentConfig struct {
	Description   string                 `json:"description"`
	Thinking      bool                   `json:"thinking"`
	RouterDisable bool                   `json:"router_disable"`
	Media         map[string]interface{} `json:"media,omitempty"`
}

type tokenConfig struct {
	Token            string `json:"token"`
	BaseURL          string `json:"__url"`
	ModelBase        string `json:"__model"`
	ModelFast        string `json:"__model_fast"`
	ModelThinking    string `json:"__model_thinking"`
	ModelMultiInput  string `json:"__model_multi_input"`
	ModelMultiOutput string `json:"__model_multi_output"`
	CustomCleared    bool   `json:"__custom_cleared"`
}

const maskedTokenValue = "**********"

type tokenPayload struct {
	Model            string                 `json:"model"`
	Token            string                 `json:"token"`
	BaseURL          string                 `json:"__url"`
	ModelBase        string                 `json:"__model"`
	ModelFast        string                 `json:"__model_fast"`
	ModelThinking    string                 `json:"__model_thinking"`
	ModelMultiInput  string                 `json:"__model_multi_input"`
	ModelMultiOutput string                 `json:"__model_multi_output"`
	CustomCleared    bool                   `json:"__custom_cleared"`
	Models           map[string]interface{} `json:"models"`
	RemovedModels    []string               `json:"removedModels"`
}

type configActionPayload struct {
	Action string `json:"action"`
	Model  string `json:"model"`
}

func renameProviderLogTable(db *sql.DB) {
	if db == nil {
		return
	}
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table'`)
	if err != nil {
		return
	}
	defer rows.Close()

	foundNew := false
	foundOld := false
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		switch name {
		case "proxy_agent_provider_log":
			foundNew = true
		case "token_store_log":
			foundOld = true
		}
	}
	if foundOld && !foundNew {
		if _, err := db.Exec(`ALTER TABLE token_store_log RENAME TO proxy_agent_provider_log`); err != nil {
			log.Printf("[integration] rename provider log failed: %v", err)
		}
	}
}

func ensureTokenStore(db *sql.DB) error {
	renameProviderLogTable(db)
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS token_store (model TEXT PRIMARY KEY, token TEXT NOT NULL DEFAULT '', __url TEXT NOT NULL DEFAULT '', __model TEXT NOT NULL DEFAULT '', __model_fast TEXT NOT NULL DEFAULT '', __model_thinking TEXT NOT NULL DEFAULT '', __model_multi_input TEXT NOT NULL DEFAULT '', __model_multi_output TEXT NOT NULL DEFAULT '', __custom_cleared INTEGER NOT NULL DEFAULT 0, updated_at TEXT NOT NULL)`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS proxy_agent_provider_log (id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL DEFAULT '', chat_id TEXT NOT NULL DEFAULT '', model TEXT NOT NULL, token TEXT NOT NULL DEFAULT '', action TEXT NOT NULL, updated_at TEXT NOT NULL)`); err != nil {
		return err
	}
	_, _ = db.Exec(`ALTER TABLE token_store ADD COLUMN __url TEXT NOT NULL DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE token_store ADD COLUMN __model TEXT NOT NULL DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE token_store ADD COLUMN __model_fast TEXT NOT NULL DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE token_store ADD COLUMN __model_thinking TEXT NOT NULL DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE token_store ADD COLUMN __model_multi_input TEXT NOT NULL DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE token_store ADD COLUMN __model_multi_output TEXT NOT NULL DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE token_store ADD COLUMN __custom_cleared INTEGER NOT NULL DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE proxy_agent_provider_log ADD COLUMN agent_id TEXT NOT NULL DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE proxy_agent_provider_log ADD COLUMN chat_id TEXT NOT NULL DEFAULT ''`)
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_proxy_agent_provider_log_agent_chat_time ON proxy_agent_provider_log(agent_id, chat_id, updated_at)`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_proxy_agent_provider_log_model_time ON proxy_agent_provider_log(model, updated_at)`); err != nil {
		return err
	}
	return nil
}

func normalizeTokenConfig(cfg tokenConfig) tokenConfig {
	return tokenConfig{
		Token:            strings.TrimSpace(cfg.Token),
		BaseURL:          strings.TrimSpace(cfg.BaseURL),
		ModelBase:        strings.TrimSpace(cfg.ModelBase),
		ModelFast:        strings.TrimSpace(cfg.ModelFast),
		ModelThinking:    strings.TrimSpace(cfg.ModelThinking),
		ModelMultiInput:  strings.TrimSpace(cfg.ModelMultiInput),
		ModelMultiOutput: strings.TrimSpace(cfg.ModelMultiOutput),
		CustomCleared:    cfg.CustomCleared,
	}
}

func maskTokenConfig(cfg tokenConfig) tokenConfig {
	cfg = normalizeTokenConfig(cfg)
	cfg.Token = maskedTokenValue
	return cfg
}

func maskedTokenConfigs(models map[string]tokenConfig) map[string]tokenConfig {
	if len(models) == 0 {
		return map[string]tokenConfig{}
	}
	masked := make(map[string]tokenConfig, len(models))
	for model, cfg := range models {
		masked[model] = maskTokenConfig(cfg)
	}
	return masked
}

func tokenConfigFromMap(raw map[string]interface{}) tokenConfig {
	cfg := tokenConfig{}
	if raw == nil {
		return cfg
	}
	if value, ok := raw["token"].(string); ok {
		cfg.Token = value
	}
	if value, ok := raw["__url"].(string); ok {
		cfg.BaseURL = value
	}
	if value, ok := raw["__model"].(string); ok {
		cfg.ModelBase = value
	}
	if value, ok := raw["__model_fast"].(string); ok {
		cfg.ModelFast = value
	}
	if value, ok := raw["__model_thinking"].(string); ok {
		cfg.ModelThinking = value
	}
	if value, ok := raw["__model_multi_input"].(string); ok {
		cfg.ModelMultiInput = value
	}
	if value, ok := raw["__model_multi_output"].(string); ok {
		cfg.ModelMultiOutput = value
	}
	if value, ok := raw["__custom_cleared"].(bool); ok {
		cfg.CustomCleared = value
	}
	return normalizeTokenConfig(cfg)
}

func tokenConfigFromValue(raw interface{}) tokenConfig {
	switch value := raw.(type) {
	case string:
		return normalizeTokenConfig(tokenConfig{Token: value})
	case map[string]interface{}:
		return tokenConfigFromMap(value)
	default:
		return tokenConfig{}
	}
}

func readTokenStore(db *sql.DB) (map[string]tokenConfig, map[string]string, error) {
	if err := ensureTokenStore(db); err != nil {
		return nil, nil, err
	}
	rows, err := db.Query(`SELECT model, token, __url, __model, __model_fast, __model_thinking, __model_multi_input, __model_multi_output, __custom_cleared, updated_at FROM token_store ORDER BY model`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	models := make(map[string]tokenConfig)
	updatedAt := make(map[string]string)
	for rows.Next() {
		var model string
		var cfg tokenConfig
		var ts string
		if err := rows.Scan(&model, &cfg.Token, &cfg.BaseURL, &cfg.ModelBase, &cfg.ModelFast, &cfg.ModelThinking, &cfg.ModelMultiInput, &cfg.ModelMultiOutput, &cfg.CustomCleared, &ts); err != nil {
			return nil, nil, err
		}
		model = sharedutil.NormalizeTokenModelName(model)
		if model == "" {
			continue
		}
		models[model] = normalizeTokenConfig(cfg)
		updatedAt[model] = ts
	}
	return models, updatedAt, rows.Err()
}

func normalizeTokenPayload(payload tokenPayload) map[string]tokenConfig {
	models := make(map[string]tokenConfig)
	for model, raw := range payload.Models {
		model = sharedutil.NormalizeTokenModelName(model)
		cfg := tokenConfigFromValue(raw)
		if model == "" || cfg.Token == "" {
			continue
		}
		models[model] = cfg
	}
	model := sharedutil.NormalizeTokenModelName(payload.Model)
	cfg := normalizeTokenConfig(tokenConfig{
		Token:            payload.Token,
		BaseURL:          payload.BaseURL,
		ModelBase:        payload.ModelBase,
		ModelFast:        payload.ModelFast,
		ModelThinking:    payload.ModelThinking,
		ModelMultiInput:  payload.ModelMultiInput,
		ModelMultiOutput: payload.ModelMultiOutput,
		CustomCleared:    payload.CustomCleared,
	})
	if model != "" && cfg.Token != "" {
		models[model] = cfg
	}
	return models
}

func restoreMaskedRemoteTokenConfigs(db *sql.DB, models map[string]tokenConfig) (map[string]tokenConfig, error) {
	if len(models) == 0 {
		return models, nil
	}
	existingModels, _, err := readTokenStore(db)
	if err != nil {
		return nil, err
	}
	for model, cfg := range models {
		model = sharedutil.NormalizeTokenModelName(model)
		cfg = normalizeTokenConfig(cfg)
		if model == "" || cfg.Token != maskedTokenValue {
			continue
		}
		existingCfg, ok := existingModels[model]
		if !ok || strings.TrimSpace(existingCfg.Token) == "" {
			continue
		}
		cfg.Token = existingCfg.Token
		models[model] = cfg
	}
	return models, nil
}

func printIntegrationTokenHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration token")
	fmt.Println("  integration token --provider MODEL")
	fmt.Println("  integration token --n N [--start 'YYYY-MM-DD HH:MM:SS'] [--close 'YYYY-MM-DD HH:MM:SS'] [--agentId ID]")
	fmt.Println("  integration token get [--n N] [--start 'YYYY-MM-DD HH:MM:SS'] [--close 'YYYY-MM-DD HH:MM:SS'] [--agentId ID]")
	fmt.Println("  integration token --agentId ID --model MODEL --function NAME --total N [--thinking N --input N --cache N --timestamp MS]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --provider MODEL   仅输出指定 provider 对应的 token")
	fmt.Println("  --n N              查询最新 N 条本地 token 用量记录；支持与 --start/--close 组合")
	fmt.Println("  --start TIME       查询开始时间，支持 yyyyMMdd-hhmmss 或 YYYY-MM-DD HH:MM:SS")
	fmt.Println("  --close TIME       查询结束时间，支持 yyyyMMdd-hhmmss 或 YYYY-MM-DD HH:MM:SS，默认当前时间")
	fmt.Println("  --agentId ID       Token 消费所属 AgentId")
	fmt.Println("  --model MODEL      Token 消费对应模型")
	fmt.Println("  --function NAME    Token 消费用途")
	fmt.Println("  --thinking N       thinking token 数")
	fmt.Println("  --input N          input token 数")
	fmt.Println("  --total N          total token 数")
	fmt.Println("  --cache N          cache token 数")
	fmt.Println("  --timestamp MS     记录时间戳（毫秒），不传默认当前时间")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  integration token")
	fmt.Println("  integration token --provider deepseek")
	fmt.Println("  integration token --n 500")
	fmt.Println("  integration token get --n 500 --start '2026-06-14 12:00:00' --close '2026-06-14 14:00:00'")
	fmt.Println("  integration token --agentId demo-agent --model deepseek-chat --function cli/get --thinking 10 --input 20 --total 30 --cache 5")
}

func printIntegrationTokenGetHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration token get [--n N] [--start 'YYYY-MM-DD HH:MM:SS'] [--close 'YYYY-MM-DD HH:MM:SS'] [--agentId ID]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --n N            查询最新 N 条本地 token 用量记录，默认 500")
	fmt.Println("  --start TIME     查询开始时间，支持 yyyyMMdd-hhmmss 或 YYYY-MM-DD HH:MM:SS")
	fmt.Println("  --close TIME     查询结束时间，支持 yyyyMMdd-hhmmss 或 YYYY-MM-DD HH:MM:SS，默认当前时间")
	fmt.Println("  --agentId ID     仅查询指定 AgentId 的 token 用量")
	fmt.Println("  --agent ID       AgentId 别名")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  integration token get --n 500")
	fmt.Println("  integration token get --n 500 --start '2026-06-14 12:00:00' --close '2026-06-14 14:00:00'")
}

func writeIntegrationTokenCLIOutput(w io.Writer, models map[string]tokenConfig, provider string) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)

	provider = sharedutil.NormalizeTokenModelName(provider)
	if provider != "" {
		single := map[string]tokenConfig{}
		if cfg, ok := models[provider]; ok {
			single[provider] = cfg
		}
		return encoder.Encode(single)
	}

	keys := make([]string, 0, len(models))
	for key := range models {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	items := make([]map[string]tokenConfig, 0, len(keys))
	for _, key := range keys {
		items = append(items, map[string]tokenConfig{key: models[key]})
	}
	return encoder.Encode(items)
}

func writeIntegrationTokenConsumeCLIOutput(w io.Writer, result tokenconsume.QueryResult) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(map[string]interface{}{
		"status":  0,
		"details": result.Details,
		"summary": result.Summary,
	})
}

func resolveIntegrationTokenQueryRange(startRaw, closeRaw string) (int64, int64, error) {
	startTimestamp := int64(0)
	closeTimestamp := time.Now().UnixMilli()
	var err error
	if strings.TrimSpace(startRaw) != "" {
		startTimestamp, err = tokenconsume.ParseFlexibleQueryTime(startRaw, time.Local)
		if err != nil {
			return 0, 0, err
		}
	}
	if strings.TrimSpace(closeRaw) != "" {
		closeTimestamp, err = tokenconsume.ParseFlexibleQueryTime(closeRaw, time.Local)
		if err != nil {
			return 0, 0, err
		}
	}
	if closeTimestamp < startTimestamp {
		return 0, 0, fmt.Errorf("closeTime must be greater than or equal to starTime")
	}
	return startTimestamp, closeTimestamp, nil
}

func runIntegrationTokenConsumeQuery(db *sql.DB, agentID, startRaw, closeRaw string, limit int) {
	if limit <= 0 {
		log.Fatal("n must be a positive integer")
	}
	startTimestamp, closeTimestamp, err := resolveIntegrationTokenQueryRange(startRaw, closeRaw)
	if err != nil {
		log.Fatal(err)
	}
	result, err := tokenconsume.QueryLatestRange(db, agentID, startTimestamp, closeTimestamp, limit)
	if err != nil {
		log.Fatal(err)
	}
	if err := writeIntegrationTokenConsumeCLIOutput(os.Stdout, result); err != nil {
		log.Fatal(err)
	}
}

func runIntegrationTokenGetCLI(args []string) {
	if hasHelpFlag(args) {
		printIntegrationTokenGetHelp()
		return
	}
	fs := flag.NewFlagSet("token get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	limit := fs.Int("n", 500, "query latest N consume records")
	startRaw := fs.String("start", "", "consume query start time")
	closeRaw := fs.String("close", "", "consume query close time")
	agentID := fs.String("agentId", "", "consume agent id")
	agent := fs.String("agent", "", "consume agent id (deprecated)")
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}
	if fs.NArg() != 0 {
		log.Fatalf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	db, err := sql.Open("sqlite", resolveIntegrationDBPath())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	targetAgentID := strings.TrimSpace(*agentID)
	if targetAgentID == "" {
		targetAgentID = strings.TrimSpace(*agent)
	}
	runIntegrationTokenConsumeQuery(db, targetAgentID, *startRaw, *closeRaw, *limit)
}

func runIntegrationTokenCLI(args []string) {
	if len(args) > 0 && strings.EqualFold(strings.TrimSpace(args[0]), "get") {
		runIntegrationTokenGetCLI(args[1:])
		return
	}
	if hasHelpFlag(args) {
		printIntegrationTokenHelp()
		return
	}

	fs := flag.NewFlagSet("integration token", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	provider := fs.String("provider", "", "provider name")
	name := fs.String("name", "", "provider name (deprecated)")
	queryLimit := fs.Int("n", 0, "query latest N consume records")
	startRaw := fs.String("start", "", "consume query start time")
	closeRaw := fs.String("close", "", "consume query close time")
	recordMode := fs.Bool("record", false, "record token consume entry")
	thinking := fs.Int("thinking", 0, "thinking token count")
	inputTokens := fs.Int("input", 0, "input token count")
	totalTokens := fs.Int("total", 0, "total token count")
	cacheTokens := fs.Int("cache", 0, "cache token count")
	model := fs.String("model", "", "consume model")
	agentID := fs.String("agentId", "", "consume agent id")
	agent := fs.String("agent", "", "consume agent id (deprecated)")
	functionName := fs.String("function", "", "consume usage")
	timestamp := fs.Int64("timestamp", 0, "consume timestamp in unix milliseconds")
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}
	if fs.NArg() != 0 {
		log.Fatalf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	db, err := sql.Open("sqlite", resolveIntegrationDBPath())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	targetAgentID := strings.TrimSpace(*agentID)
	if targetAgentID == "" {
		targetAgentID = strings.TrimSpace(*agent)
	}
	queryMode := *queryLimit > 0 || strings.TrimSpace(*startRaw) != "" || strings.TrimSpace(*closeRaw) != ""
	if queryMode {
		selectedProvider := strings.TrimSpace(*provider)
		if selectedProvider == "" {
			selectedProvider = strings.TrimSpace(*name)
		}
		if selectedProvider != "" {
			log.Fatal("provider filter is not supported in token consume query mode")
		}
		if *recordMode || strings.TrimSpace(*model) != "" || strings.TrimSpace(*functionName) != "" || *thinking != 0 || *inputTokens != 0 || *totalTokens != 0 || *cacheTokens != 0 || *timestamp != 0 {
			log.Fatal("token consume query flags cannot be combined with record flags")
		}
		limit := *queryLimit
		if limit == 0 {
			limit = 500
		}
		runIntegrationTokenConsumeQuery(db, targetAgentID, *startRaw, *closeRaw, limit)
		return
	}
	consumeWriteMode := *recordMode ||
		targetAgentID != "" ||
		strings.TrimSpace(*model) != "" ||
		strings.TrimSpace(*functionName) != "" ||
		*thinking != 0 ||
		*inputTokens != 0 ||
		*totalTokens != 0 ||
		*cacheTokens != 0 ||
		*timestamp != 0
	if consumeWriteMode {
		record, err := tokenconsume.Insert(db, tokenconsume.Record{
			Thinking:  *thinking,
			Input:     *inputTokens,
			Total:     *totalTokens,
			Cache:     *cacheTokens,
			Model:     strings.TrimSpace(*model),
			AgentID:   targetAgentID,
			Function:  strings.TrimSpace(*functionName),
			Timestamp: *timestamp,
		})
		if err != nil {
			log.Fatal(err)
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(map[string]interface{}{"status": 0, "record": record}); err != nil {
			log.Fatal(err)
		}
		return
	}

	models, _, err := readTokenStore(db)
	if err != nil {
		log.Fatal(err)
	}
	selectedProvider := strings.TrimSpace(*provider)
	if selectedProvider == "" {
		selectedProvider = strings.TrimSpace(*name)
	}
	if err := writeIntegrationTokenCLIOutput(os.Stdout, models, selectedProvider); err != nil {
		log.Fatal(err)
	}
}

func printIntegrationSandboxHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration sandbox --chatId ID")
	fmt.Println("  integration sandbox --agentId ID --chatId ID --sandbox off|filepick|net|filepick_net [--dir DIR]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --agentId ID      当前会话所属 AgentId；查询可省略，写入必填")
	fmt.Println("  --chatId ID       当前会话所属 ChatId")
	fmt.Println("  --chat ID         ChatId 别名")
	fmt.Println("  --sandbox MODE    写入当前会话沙盒模式；off 表示关闭，不传则只查询")
	fmt.Println("  --dir DIR         filepick/filepick_net 手动指定白名单目录，不弹窗")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  integration sandbox --chatId chat-001")
	fmt.Println("  integration sandbox --agentId A --chatId chat-001 --sandbox filepick_net")
	fmt.Println("  integration sandbox --agentId A --chatId chat-001 --sandbox filepick --dir /Users/me/Desktop")
	fmt.Println("  integration sandbox --agentId A --chatId chat-001 --sandbox off")
}

func runIntegrationSandboxCLI(args []string) {
	if hasHelpFlag(args) {
		printIntegrationSandboxHelp()
		return
	}

	fs := flag.NewFlagSet("integration sandbox", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	agentID := fs.String("agentId", "", "agent id")
	chatID := fs.String("chatId", "", "chat id")
	chatAlias := fs.String("chat", "", "chat id")
	sandboxRaw := fs.String("sandbox", "", "sandbox mode")
	allowedDir := fs.String("dir", "", "manual whitelist directory for filepick modes")
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}
	if fs.NArg() != 0 {
		log.Fatalf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	targetAgentID := strings.TrimSpace(*agentID)
	targetChatID := strings.TrimSpace(*chatID)
	if targetChatID == "" {
		targetChatID = strings.TrimSpace(*chatAlias)
	}
	if targetChatID == "" {
		log.Fatal("chatId is required")
	}

	dbPath := resolveIntegrationDBPath()
	if err := ensureParentDir(dbPath); err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var (
		state    sandboxstate.State
		recorded bool
	)
	if strings.TrimSpace(*sandboxRaw) != "" {
		if targetAgentID == "" {
			log.Fatal("agentId is required when setting sandbox")
		}
		mode, err := parseSandboxUpdateValue(*sandboxRaw)
		if err != nil {
			log.Fatal(err)
		}
		resolvedAllowedDir, err := integrationPrimeSandboxDirectoryFn(mode, *allowedDir)
		if err != nil {
			log.Fatal(err)
		}
		previous, existed, err := sandboxstate.Get(db, "", targetChatID)
		if err != nil {
			log.Fatal(err)
		}
		beforeMode := ""
		if existed {
			beforeMode = previous.Sandbox
		}
		state, err = sandboxstate.SetWithDirectory(db, targetAgentID, targetChatID, mode, resolvedAllowedDir)
		if err != nil {
			log.Fatal(err)
		}
		logIntegrationSandboxChange(targetAgentID, targetChatID, beforeMode, state.Sandbox)
		recorded = strings.TrimSpace(state.Sandbox) != ""
	} else {
		if strings.TrimSpace(*allowedDir) != "" {
			log.Fatal("--dir requires --sandbox filepick or --sandbox filepick_net")
		}
		var err error
		state, recorded, err = sandboxstate.Get(db, targetAgentID, targetChatID)
		if err != nil {
			log.Fatal(err)
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(map[string]interface{}{
		"status":     0,
		"agentId":    state.AgentID,
		"chatId":     state.ChatID,
		"sandbox":    state.Sandbox,
		"allowedDir": state.AllowedDir,
		"recorded":   recorded,
		"updatedAt":  state.UpdatedAt,
	}); err != nil {
		log.Fatal(err)
	}
}

func lookupTokenConfigByModel(db *sql.DB, model string) (tokenConfig, error) {
	model = sharedutil.NormalizeTokenModelName(model)
	if model == "" {
		return tokenConfig{}, nil
	}
	models, _, err := readTokenStore(db)
	if err != nil {
		return tokenConfig{}, err
	}
	return normalizeTokenConfig(models[model]), nil
}

func injectConfiguredModelMetadata(metaMap map[string]interface{}, cfg tokenConfig) {
	if metaMap == nil {
		return
	}
	if cfg.BaseURL != "" {
		metaMap["__url"] = cfg.BaseURL
	}
	if cfg.ModelBase != "" {
		metaMap["__model"] = cfg.ModelBase
	}
	if cfg.ModelFast != "" {
		metaMap["__model_fast"] = cfg.ModelFast
	}
	if cfg.ModelThinking != "" {
		metaMap["__model_thinking"] = cfg.ModelThinking
	}
	if cfg.ModelMultiInput != "" {
		metaMap["__model_multi_input"] = cfg.ModelMultiInput
	}
	if cfg.ModelMultiOutput != "" {
		metaMap["__model_multi_output"] = cfg.ModelMultiOutput
	}
}

func writeTokenStore(db *sql.DB, models map[string]string) error {
	configs := make(map[string]tokenConfig, len(models))
	for model, token := range models {
		configs[model] = tokenConfig{Token: token}
	}
	return writeTokenStoreConfigs(db, configs)
}

func writeTokenStoreConfigs(db *sql.DB, models map[string]tokenConfig) error {
	return writeTokenStoreChanges(db, models, nil)
}

type tokenStoreVariant struct {
	Model string
	Token string
}

func listCanonicalTokenStoreVariants(q interface {
	Query(string, ...any) (*sql.Rows, error)
}, canonical string) ([]tokenStoreVariant, error) {
	canonical = sharedutil.NormalizeTokenModelName(canonical)
	if canonical == "" {
		return nil, nil
	}
	rows, err := q.Query(`SELECT model, token FROM token_store`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	variants := make([]tokenStoreVariant, 0, 1)
	for rows.Next() {
		var stored tokenStoreVariant
		if err := rows.Scan(&stored.Model, &stored.Token); err != nil {
			return nil, err
		}
		if sharedutil.NormalizeTokenModelName(stored.Model) == canonical {
			variants = append(variants, stored)
		}
	}
	return variants, rows.Err()
}

func writeTokenStoreChanges(db *sql.DB, models map[string]tokenConfig, removedModels []string) error {
	if err := ensureTokenStore(db); err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	upsertStmt, err := tx.Prepare(`INSERT INTO token_store (model, token, __url, __model, __model_fast, __model_thinking, __model_multi_input, __model_multi_output, __custom_cleared, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?) ON CONFLICT(model) DO UPDATE SET token=excluded.token, __url=excluded.__url, __model=excluded.__model, __model_fast=excluded.__model_fast, __model_thinking=excluded.__model_thinking, __model_multi_input=excluded.__model_multi_input, __model_multi_output=excluded.__model_multi_output, __custom_cleared=excluded.__custom_cleared, updated_at=excluded.updated_at`)
	if err != nil {
		return err
	}
	defer upsertStmt.Close()

	deleteStmt, err := tx.Prepare(`DELETE FROM token_store WHERE model = ?`)
	if err != nil {
		return err
	}
	defer deleteStmt.Close()

	logStmt, err := tx.Prepare(`INSERT INTO proxy_agent_provider_log (agent_id, chat_id, model, token, action, updated_at) VALUES (?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer logStmt.Close()

	now := time.Now().Format(time.RFC3339)
	for _, removedModel := range removedModels {
		model := sharedutil.NormalizeTokenModelName(removedModel)
		if model == "" {
			continue
		}
		if _, keep := models[model]; keep {
			continue
		}
		variants, err := listCanonicalTokenStoreVariants(tx, model)
		if err != nil {
			return err
		}
		for _, variant := range variants {
			if _, err := deleteStmt.Exec(variant.Model); err != nil {
				return err
			}
			if _, err := logStmt.Exec("", "", variant.Model, variant.Token, "delete", now); err != nil {
				return err
			}
		}
	}

	for model, cfg := range models {
		model = sharedutil.NormalizeTokenModelName(model)
		cfg = normalizeTokenConfig(cfg)
		if model == "" || cfg.Token == "" {
			continue
		}

		action := "insert"
		variants, err := listCanonicalTokenStoreVariants(tx, model)
		if err != nil {
			return err
		}
		for _, variant := range variants {
			if variant.Model == model {
				action = "update"
				continue
			}
			if _, err := deleteStmt.Exec(variant.Model); err != nil {
				return err
			}
			if _, err := logStmt.Exec("", "", variant.Model, variant.Token, "delete", now); err != nil {
				return err
			}
		}

		if _, err := upsertStmt.Exec(model, cfg.Token, cfg.BaseURL, cfg.ModelBase, cfg.ModelFast, cfg.ModelThinking, cfg.ModelMultiInput, cfg.ModelMultiOutput, cfg.CustomCleared, now); err != nil {
			return err
		}
		if _, err := logStmt.Exec("", "", model, cfg.Token, action, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func normalizeRemovedTokenModels(payload tokenPayload, models map[string]tokenConfig) []string {
	if len(payload.RemovedModels) == 0 {
		return nil
	}
	removed := make([]string, 0, len(payload.RemovedModels))
	seen := make(map[string]struct{}, len(payload.RemovedModels))
	for _, raw := range payload.RemovedModels {
		model := sharedutil.NormalizeTokenModelName(raw)
		if model == "" {
			continue
		}
		if _, keep := models[model]; keep {
			continue
		}
		if _, exists := seen[model]; exists {
			continue
		}
		seen[model] = struct{}{}
		removed = append(removed, model)
	}
	return removed
}

func listNewTokenModels(db *sql.DB, models map[string]tokenConfig) ([]string, error) {
	if len(models) == 0 {
		return nil, nil
	}
	existingModels, _, err := readTokenStore(db)
	if err != nil {
		return nil, err
	}
	added := make([]string, 0, len(models))
	for model := range models {
		model = sharedutil.NormalizeTokenModelName(model)
		if model == "" {
			continue
		}
		if _, exists := existingModels[model]; exists {
			continue
		}
		added = append(added, model)
	}
	sort.Strings(added)
	return added, nil
}

func deleteTokenStoreModel(db *sql.DB, model string) error {
	if err := ensureTokenStore(db); err != nil {
		return err
	}
	model = sharedutil.NormalizeTokenModelName(model)
	if model == "" {
		return fmt.Errorf("model is required")
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	variants, err := listCanonicalTokenStoreVariants(tx, model)
	if err != nil {
		return err
	}
	if len(variants) == 0 {
		return tx.Commit()
	}

	now := time.Now().Format(time.RFC3339)
	for _, variant := range variants {
		if _, err := tx.Exec(`DELETE FROM token_store WHERE model = ?`, variant.Model); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO proxy_agent_provider_log (agent_id, chat_id, model, token, action, updated_at) VALUES (?,?,?,?,?,?)`, "", "", variant.Model, variant.Token, "delete", now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func handleSwarm(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "read body failed"})
			return
		}

		var actionPayload configActionPayload
		if err := json.Unmarshal(body, &actionPayload); err == nil {
			if strings.EqualFold(strings.TrimSpace(actionPayload.Action), "delete_model") {
				if cronDB == nil {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "db not ready"})
					return
				}
				if err := deleteTokenStoreModel(cronDB, actionPayload.Model); err != nil {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "delete model failed: " + err.Error()})
					return
				}
				models, updatedAt, err := readTokenStore(cronDB)
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "query error: " + err.Error()})
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":    0,
					"action":    "delete_model",
					"model":     sharedutil.NormalizeTokenModelName(actionPayload.Model),
					"models":    models,
					"updatedAt": updatedAt,
				})
				return
			}
		}

		agentID := r.URL.Query().Get("agentId")
		if agentID == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "agentId is required"})
			return
		}
		agentTTL := time.Duration(cfg.AgentCacheMs) * time.Millisecond
		metadata, err := getAgentOutput(cfg.AgentDir, cfg.effectiveDeviceID(), agentTTL)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "agent metadata error: " + err.Error()})
			return
		}
		var workspace string
		for _, a := range metadata.Agents {
			if a.AgentID == agentID {
				workspace = a.Workspace
				break
			}
		}
		if workspace == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "agent not found: " + agentID})
			return
		}
		var sc AgentConfig
		if err := json.Unmarshal(body, &sc); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "invalid JSON"})
			return
		}
		if len([]rune(sc.Description)) > 200 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "description exceeds 200 characters"})
			return
		}
		out, _ := json.MarshalIndent(sc, "", "  ")
		if err := os.WriteFile(filepath.Join(workspace, "config.json"), out, 0644); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "write failed: " + err.Error()})
			return
		}
		flushAgentCache()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "agentId": agentID})
	}
}

func handleToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cronDB == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "db not ready"})
			return
		}
		localRequest := isLocalManagementRequest(r)
		switch r.Method {
		case http.MethodGet:
			models, updatedAt, err := readTokenStore(cronDB)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "query error: " + err.Error()})
				return
			}
			if !localRequest {
				models = maskedTokenConfigs(models)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "models": models, "updatedAt": updatedAt})
		case http.MethodPost:
			var payload tokenPayload
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "invalid request: " + err.Error()})
				return
			}
			models := normalizeTokenPayload(payload)
			removedModels := normalizeRemovedTokenModels(payload, models)
			if !localRequest {
				models, err := restoreMaskedRemoteTokenConfigs(cronDB, models)
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "query error: " + err.Error()})
					return
				}
				addedModels, err := listNewTokenModels(cronDB, models)
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "query error: " + err.Error()})
					return
				}
				if len(addedModels) > 0 {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "only localhost/127.0.0.1/::1 requests can add models"})
					return
				}
			}
			if err := writeTokenStoreChanges(cronDB, models, removedModels); err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "save error: " + err.Error()})
				return
			}
			updatedModels, updatedAt, err := readTokenStore(cronDB)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "query error: " + err.Error()})
				return
			}
			if !localRequest {
				updatedModels = maskedTokenConfigs(updatedModels)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "models": updatedModels, "updatedAt": updatedAt})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleModelProviderCatalog(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		providers := map[string]map[string]string{}
		if cfg != nil && cfg.Provider != nil {
			providers = cfg.Provider
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   0,
			"provider": providers,
		})
	}
}

var integrationClientRuntimeConfigFields = []string{
	"agent-dir",
	"app-dir",
	"app",
	"default-dir",
	"site",
	"resources-dir",
	"knowledge",
	"skills_git_install",
}

func integrationClientRuntimeConfig(raw map[string]interface{}) map[string]interface{} {
	config := make(map[string]interface{})
	for _, key := range integrationClientRuntimeConfigFields {
		if value, ok := raw[key]; ok {
			config[key] = value
		}
	}
	return config
}

func handleRuntimeConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		raw, _, err := readIntegrationStartupConfigRaw()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "读取 config/config.json 失败: " + err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": 0,
			"config": integrationClientRuntimeConfig(raw),
		})
	}
}

func handleConsume() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if cronDB == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "db not ready"})
			return
		}
		if r.Method == http.MethodDelete {
			deleted, err := tokenconsume.Clear(cronDB)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "clear error: " + err.Error()})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "deleted": deleted})
			return
		}
		startRaw := strings.TrimSpace(r.URL.Query().Get("starTime"))
		if startRaw == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "starTime is required"})
			return
		}
		startTimestamp, err := tokenconsume.ParseQueryTime(startRaw, time.Local)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "starTime must use format yyyyMMdd-hhmmss"})
			return
		}
		closeRaw := strings.TrimSpace(r.URL.Query().Get("closeTime"))
		if closeRaw == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "closeTime is required"})
			return
		}
		closeTimestamp, err := tokenconsume.ParseQueryTime(closeRaw, time.Local)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "closeTime must use format yyyyMMdd-hhmmss"})
			return
		}
		limit := 500
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			parsed, parseErr := strconv.Atoi(raw)
			if parseErr != nil || parsed <= 0 {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "limit must be a positive integer"})
				return
			}
			limit = parsed
		}
		result, err := tokenconsume.QueryLatestRange(cronDB, r.URL.Query().Get("agentId"), startTimestamp, closeTimestamp, limit)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "query error: " + err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "details": result.Details, "summary": result.Summary})
	}
}

func sandboxTarget(r *http.Request) (string, string, error) {
	agentID := strings.TrimSpace(r.URL.Query().Get("agentId"))
	chatID := strings.TrimSpace(r.URL.Query().Get("chatId"))
	if chatID == "" {
		chatID = strings.TrimSpace(r.URL.Query().Get("chat"))
	}
	if chatID == "" {
		return "", "", fmt.Errorf("chatId is required")
	}
	return agentID, chatID, nil
}

func parseSandboxUpdateValue(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "off", "disable", "disabled", "none", "close", "closed":
		return "", nil
	case sandboxstate.ModeFilePick, sandboxstate.ModeNet, sandboxstate.ModeFilePickNet:
		return sandboxstate.NormalizeMode(value), nil
	default:
		return "", fmt.Errorf("sandbox must be one of off,filepick,net,filepick_net")
	}
}

func sandboxModeFromRequestPath(r *http.Request) (string, bool, error) {
	if r == nil || r.URL == nil {
		return "", false, fmt.Errorf("sandbox request is invalid")
	}
	path := strings.TrimSpace(r.URL.Path)
	switch {
	case path == "/api/sandbox=off":
		return "", true, nil
	case strings.HasPrefix(path, "/api/sandbox="):
		mode, err := parseSandboxUpdateValue(strings.TrimPrefix(path, "/api/sandbox="))
		if err != nil {
			return "", false, err
		}
		return mode, true, nil
	case path == "/api/sandbox":
		return "", false, nil
	default:
		return "", false, fmt.Errorf("unsupported sandbox path: %s", path)
	}
}

func writeSandboxState(w http.ResponseWriter, state sandboxstate.State, recorded bool) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     0,
		"agentId":    state.AgentID,
		"chatId":     state.ChatID,
		"sandbox":    state.Sandbox,
		"allowedDir": state.AllowedDir,
		"recorded":   recorded,
		"updatedAt":  state.UpdatedAt,
	})
}

func handleSandboxQuery(allowSet bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if cronDB == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "db not ready"})
			return
		}

		agentID, chatID, err := sandboxTarget(r)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}

		if allowSet {
			if agentID == "" {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "agentId is required"})
				return
			}
			mode, shouldSet, err := sandboxModeFromRequestPath(r)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
				return
			}
			if !shouldSet {
				if strings.TrimSpace(r.URL.Query().Get("dir")) != "" {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "dir requires sandbox mode filepick or filepick_net"})
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "sandbox mode is required"})
				return
			}
			previous, existed, err := sandboxstate.Get(cronDB, "", chatID)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "query error: " + err.Error()})
				return
			}
			beforeMode := ""
			if existed {
				beforeMode = previous.Sandbox
			}
			allowedDir, err := integrationPrimeSandboxDirectoryFn(mode, r.URL.Query().Get("dir"))
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
				return
			}
			state, err := sandboxstate.SetWithDirectory(cronDB, agentID, chatID, mode, allowedDir)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "save error: " + err.Error()})
				return
			}
			logIntegrationSandboxChange(agentID, chatID, beforeMode, state.Sandbox)
			writeSandboxState(w, state, strings.TrimSpace(state.Sandbox) != "")
			return
		}

		state, found, err := sandboxstate.Get(cronDB, agentID, chatID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "query error: " + err.Error()})
			return
		}
		writeSandboxState(w, state, found)
	}
}

func handleSandbox() http.HandlerFunc {
	return handleSandboxQuery(true)
}

func handleSandboxStatus() http.HandlerFunc {
	return handleSandboxQuery(false)
}

// ═══════════════════════════════════════════════════════════════════════════
// Proxy: /api/download — download file or directory
// ═══════════════════════════════════════════════════════════════════════════

func handleDownload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		pathParam := r.URL.Query().Get("path")
		if pathParam == "" {
			http.Error(w, "path is required", http.StatusBadRequest)
			return
		}
		resolved, err := resolveCaseInsensitive(pathParam)
		if err != nil {
			http.Error(w, "文件不存在: "+err.Error(), http.StatusNotFound)
			return
		}
		info, err := os.Stat(resolved)
		if err != nil {
			http.Error(w, "无法访问: "+err.Error(), http.StatusNotFound)
			return
		}
		if info.IsDir() {
			w.Header().Set("Content-Type", "application/zip")
			w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(resolved)+`.zip"`)
			zw := zip.NewWriter(w)
			defer zw.Close()
			filepath.WalkDir(resolved, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				rel, _ := filepath.Rel(resolved, path)
				if d.IsDir() {
					if rel != "." {
						zw.Create(rel + "/")
					}
					return nil
				}
				fw, err := zw.Create(rel)
				if err != nil {
					return err
				}
				f, err := os.Open(path)
				if err != nil {
					return err
				}
				defer f.Close()
				io.Copy(fw, f)
				return nil
			})
			return
		}
		// Single file: stream download
		f, err := os.Open(resolved)
		if err != nil {
			http.Error(w, "读取失败: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(resolved)+`"`)
		http.ServeContent(w, r, filepath.Base(resolved), info.ModTime(), f)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Proxy: /api/cron/create — create cron task
// ═══════════════════════════════════════════════════════════════════════════

func handleCron(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentID := r.URL.Query().Get("agentId")
		if agentID == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "agentId is required"})
			return
		}
		var req cronCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "invalid request: " + err.Error()})
			return
		}
		if err := validateIntegrationAgentExists(cfg.AgentDir, cfg.effectiveDeviceID(), agentID); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		if err := validateIntegrationRegisteredModel(req.Model); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		resp, err := createCronTask(agentID, req)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

type cronAgentStopRequest struct {
	Enabled *bool `json:"enabled"`
}

func writeCronAgentStopResponse(w http.ResponseWriter, state cronAgentStopState) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     0,
		"agentId":    state.AgentID,
		"enabled":    state.Enabled,
		"updatedAt":  state.UpdatedAt,
		"stoppedAt":  state.StoppedAt,
		"resumeFrom": state.ResumeFrom,
	})
}

func handleCronAgentStop(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agentID := strings.TrimSpace(r.URL.Query().Get("agentId"))
		if agentID == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "agentId is required"})
			return
		}
		if err := validateIntegrationAgentExists(cfg.AgentDir, cfg.effectiveDeviceID(), agentID); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		if cronDB == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "db not ready"})
			return
		}
		switch r.Method {
		case http.MethodGet:
			state, err := getCronAgentStopState(cronDB, agentID)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
				return
			}
			writeCronAgentStopResponse(w, state)
		case http.MethodPatch:
			var req cronAgentStopRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Enabled == nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "enabled is required"})
				return
			}
			state, err := setCronAgentStopState(cronDB, agentID, *req.Enabled, time.Now())
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
				return
			}
			writeCronAgentStopResponse(w, state)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

type cronMetaFilter struct {
	AgentID       string
	ChatID        string
	TaskType      string
	Model         string
	Content       string
	Cycle         *int
	Status        *int
	RecurringOnly bool
	StartFrom     *time.Time
	StartTo       *time.Time
}

type cronDetailFilter struct {
	DetailID *int
	MetaID   *int
	AgentID  string
	ChatID   string
	TaskType string
	Model    string
	Content  string
	Cycle    *int
	Status   *int
	ExecFrom *time.Time
	ExecTo   *time.Time
}

type cronMetaResult struct {
	ID             int    `json:"id"`
	Cycle          int    `json:"cycle"`
	RawTime        string `json:"rawTime"`
	AgentID        string `json:"agentId"`
	Type           string `json:"type"`
	Model          string `json:"model"`
	Thinking       bool   `json:"thinking"`
	Verify         bool   `json:"verify"`
	RouterDisable  bool   `json:"router_disable"`
	Cron           string `json:"cron"`
	Content        string `json:"content"`
	ResponseSchema string `json:"responseSchema"`
	ChatID         string `json:"chatId,omitempty"`
	DetailStatuses string `json:"detailStatuses,omitempty"`
}

type cronDetailResult struct {
	ID             int    `json:"id"`
	MetaID         int    `json:"metaId"`
	ExecTime       int64  `json:"execTime"`
	AgentID        string `json:"agentId"`
	ChatID         string `json:"chatId,omitempty"`
	MetaRef        string `json:"metaRef,omitempty"`
	Type           string `json:"type"`
	Model          string `json:"model"`
	Thinking       bool   `json:"thinking"`
	Verify         bool   `json:"verify"`
	RouterDisable  bool   `json:"router_disable"`
	Content        string `json:"content"`
	ResponseSchema string `json:"responseSchema"`
	ResultContent  string `json:"resultContent,omitempty"`
	RepliedAt      string `json:"repliedAt,omitempty"`
	Started        int    `json:"started"`
	Cycle          int    `json:"cycle"`
	RawTime        string `json:"rawTime"`
	Cron           string `json:"cron"`
}

type cronMetaLogEntry struct {
	MetaID         int
	AgentID        string
	ChatID         string
	TaskType       string
	Action         string
	Cycle          int
	RawTime        string
	Model          string
	Thinking       bool
	Verify         bool
	RouterDisable  bool
	Cron           string
	Content        string
	ResponseSchema string
	CreatedAt      string
}

type cronDetailLogEntry struct {
	DetailID       int
	MetaID         int
	AgentID        string
	ChatID         string
	TaskType       string
	Action         string
	ExecTime       int64
	Model          string
	Thinking       bool
	Verify         bool
	RouterDisable  bool
	Content        string
	ResponseSchema string
	Started        int
	OccurredAt     string
}

type cronQueryOptions struct {
	MetaID         string
	DetailID       string
	ID             string
	AgentID        string
	ChatID         string
	TaskType       string
	Model          string
	Content        string
	Cycle          *int
	Time           string
	Date           string
	From           string
	To             string
	IncludeHistory bool
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func parseOptionalIntValue(value string) (*int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil, fmt.Errorf("invalid integer value: %s", value)
	}
	return &parsed, nil
}

func parseCronStatusFilter(value string) (*int, error) {
	status, err := parseOptionalIntValue(value)
	if err != nil || status == nil {
		return status, err
	}
	if *status < 0 || *status > 3 {
		return nil, fmt.Errorf("status must be one of 0,1,2,3")
	}
	return status, nil
}

func parseCronMetaIDValue(value string) (*int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	lower := strings.ToLower(value)
	for _, prefix := range []string{"cron_", "meta_"} {
		if strings.HasPrefix(lower, prefix) {
			value = value[len(prefix):]
			break
		}
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil, fmt.Errorf("invalid metaId: %s", value)
	}
	return &parsed, nil
}

func parseCronDetailIDValue(value string) (*int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "detail_") {
		value = value[7:]
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil, fmt.Errorf("invalid detailId: %s", value)
	}
	return &parsed, nil
}

func parseFlexibleLocalTime(value string) (time.Time, bool, error) {
	value = strings.TrimSpace(value)
	layouts := []struct {
		layout   string
		dateOnly bool
		rfc3339  bool
	}{
		{layout: "2006-01-02 15:04:05"},
		{layout: "2006-01-02 15:04"},
		{layout: "2006-01-02T15:04:05"},
		{layout: "2006-01-02T15:04"},
		{layout: time.RFC3339, rfc3339: true},
		{layout: "2006-01-02", dateOnly: true},
	}
	for _, item := range layouts {
		var (
			t   time.Time
			err error
		)
		if item.rfc3339 {
			t, err = time.Parse(item.layout, value)
			if err == nil {
				return t.In(time.Local), item.dateOnly, nil
			}
			continue
		}
		t, err = time.ParseInLocation(item.layout, value, time.Local)
		if err == nil {
			return t, item.dateOnly, nil
		}
	}
	return time.Time{}, false, fmt.Errorf("invalid time format: %s", value)
}

func resolveTimeRange(exactValue, dateValue, fromValue, toValue string, defaultFuture bool) (*time.Time, *time.Time, error) {
	if exact := firstNonEmpty(exactValue, dateValue); exact != "" {
		t, dateOnly, err := parseFlexibleLocalTime(exact)
		if err != nil {
			return nil, nil, err
		}
		if dateOnly {
			start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
			end := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.Local)
			return &start, &end, nil
		}
		return &t, &t, nil
	}

	var (
		from *time.Time
		to   *time.Time
	)
	if strings.TrimSpace(fromValue) != "" {
		t, dateOnly, err := parseFlexibleLocalTime(fromValue)
		if err != nil {
			return nil, nil, err
		}
		if dateOnly {
			t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
		}
		from = &t
	}
	if strings.TrimSpace(toValue) != "" {
		t, dateOnly, err := parseFlexibleLocalTime(toValue)
		if err != nil {
			return nil, nil, err
		}
		if dateOnly {
			t = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.Local)
		}
		to = &t
	}
	if from == nil && to == nil && defaultFuture {
		now := time.Now()
		from = &now
	}
	return from, to, nil
}

func parseCronQueryArgs(args []string) (cronQueryOptions, error) {
	var opts cronQueryOptions
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		key := strings.TrimPrefix(arg, "--")
		value := ""
		hasValue := false
		if parts := strings.SplitN(key, "=", 2); len(parts) == 2 {
			key = parts[0]
			value = parts[1]
			hasValue = true
		} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			value = args[i+1]
			hasValue = true
			i++
		}

		switch key {
		case "id":
			if hasValue {
				opts.ID = value
			}
		case "meta", "metaId", "meta-id":
			if hasValue {
				opts.MetaID = value
			}
		case "detail", "detailId", "detail-id":
			if hasValue {
				opts.DetailID = value
			}
		case "agent", "agentId":
			if hasValue {
				opts.AgentID = value
			}
		case "chat", "chatId":
			if hasValue {
				opts.ChatID = value
			}
		case "type":
			if hasValue {
				opts.TaskType = value
			}
		case "model":
			if hasValue {
				opts.Model = value
			}
		case "content":
			if hasValue {
				opts.Content = value
			}
		case "cycle":
			if !hasValue {
				return opts, fmt.Errorf("cycle requires a value")
			}
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return opts, fmt.Errorf("invalid cycle: %s", value)
			}
			opts.Cycle = &parsed
		case "time", "rawTime", "raw-time", "start", "startTime", "start-time", "execTime", "exec-time":
			if hasValue {
				opts.Time = value
			}
		case "date":
			if hasValue {
				opts.Date = value
			}
		case "from", "startFrom", "start-from", "execFrom", "exec-from":
			if hasValue {
				opts.From = value
			}
		case "to", "startTo", "start-to", "execTo", "exec-to":
			if hasValue {
				opts.To = value
			}
		}
	}
	return opts, nil
}

func deleteCronMetas(db *sql.DB, filter cronMetaFilter, explicitID string) (int64, []int, error) {
	var metaIDs []int
	if strings.TrimSpace(explicitID) != "" {
		metaID, err := parseCronMetaIDValue(explicitID)
		if err != nil {
			return 0, nil, err
		}
		if metaID == nil {
			return 0, nil, fmt.Errorf("id is required")
		}
		metaIDs = append(metaIDs, *metaID)
	} else {
		items, err := queryCronMetas(db, filter)
		if err != nil {
			return 0, nil, err
		}
		for _, item := range items {
			metaIDs = append(metaIDs, item.ID)
		}
	}
	if len(metaIDs) == 0 {
		return 0, []int{}, nil
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, nil, err
	}
	defer tx.Rollback()

	metaMap, err := loadCronMetaLogMap(tx, metaIDs)
	if err != nil {
		return 0, nil, err
	}
	detailItems, err := loadCronDetailsByMetaIDs(tx, metaIDs, true)
	if err != nil {
		return 0, nil, err
	}
	for _, metaID := range metaIDs {
		if _, err := tx.Exec(`DELETE FROM task_detail WHERE meta_id = ? AND started != 3`, metaID); err != nil {
			return 0, nil, err
		}
	}
	res, err := tx.Exec(fmt.Sprintf(`DELETE FROM task_meta WHERE id IN (%s)`, placeholders(len(metaIDs))), intsToInterfaces(metaIDs)...)
	if err != nil {
		return 0, nil, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, nil, err
	}
	if err := tx.Commit(); err != nil {
		return 0, nil, err
	}
	for _, detail := range detailItems {
		appendCronDetailLog(db, cronDetailLogEntry{
			DetailID:   detail.ID,
			MetaID:     detail.MetaID,
			AgentID:    detail.AgentID,
			ChatID:     detail.ChatID,
			TaskType:   detail.Type,
			Action:     "delete",
			ExecTime:   detail.ExecTime,
			Model:      detail.Model,
			Thinking:   detail.Thinking,
			Content:    detail.Content,
			Started:    detail.Started,
			OccurredAt: cronLogTimestamp(),
		})
	}
	for _, metaID := range metaIDs {
		if item, ok := metaMap[metaID]; ok {
			appendCronMetaLog(db, cronMetaLogEntry{
				MetaID:    item.ID,
				AgentID:   item.AgentID,
				ChatID:    item.ChatID,
				TaskType:  item.Type,
				Action:    "delete",
				Cycle:     item.Cycle,
				RawTime:   item.RawTime,
				Model:     item.Model,
				Thinking:  item.Thinking,
				Cron:      item.Cron,
				Content:   item.Content,
				CreatedAt: cronLogTimestamp(),
			})
		}
	}
	return affected, metaIDs, nil
}

func deleteCronDetails(db *sql.DB, filter cronDetailFilter, explicitDetailID, explicitMetaID string) (int64, []int, error) {
	var detailIDs []int
	if strings.TrimSpace(explicitDetailID) != "" {
		detailID, err := parseCronDetailIDValue(explicitDetailID)
		if err != nil {
			return 0, nil, err
		}
		if detailID == nil {
			return 0, []int{}, nil
		}
		detailIDs = append(detailIDs, *detailID)
	} else {
		if strings.TrimSpace(explicitMetaID) != "" {
			metaID, err := parseCronMetaIDValue(explicitMetaID)
			if err != nil {
				return 0, nil, err
			}
			filter.MetaID = metaID
		}
		items, err := queryCronDetails(db, filter)
		if err != nil {
			return 0, nil, err
		}
		for _, item := range items {
			detailIDs = append(detailIDs, item.ID)
		}
	}
	if len(detailIDs) == 0 {
		return 0, []int{}, nil
	}

	detailMap, err := loadCronDetailLogMap(db, detailIDs)
	if err != nil {
		return 0, nil, err
	}
	res, err := db.Exec(fmt.Sprintf(`DELETE FROM task_detail WHERE id IN (%s)`, placeholders(len(detailIDs))), intsToInterfaces(detailIDs)...)
	if err != nil {
		return 0, nil, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, nil, err
	}
	for _, detailID := range detailIDs {
		if item, ok := detailMap[detailID]; ok {
			appendCronDetailLog(db, cronDetailLogEntry{
				DetailID:   item.ID,
				MetaID:     item.MetaID,
				AgentID:    item.AgentID,
				ChatID:     item.ChatID,
				TaskType:   item.Type,
				Action:     "delete",
				ExecTime:   item.ExecTime,
				Model:      item.Model,
				Thinking:   item.Thinking,
				Content:    item.Content,
				Started:    item.Started,
				OccurredAt: cronLogTimestamp(),
			})
		}
	}
	return affected, detailIDs, nil
}

func loadCronMetaLogMap(queryer interface {
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryRow(string, ...interface{}) *sql.Row
}, metaIDs []int) (map[int]cronMetaResult, error) {
	result := make(map[int]cronMetaResult, len(metaIDs))
	if len(metaIDs) == 0 {
		return result, nil
	}
	rows, err := queryer.Query(fmt.Sprintf(`SELECT id, cycle, raw_time, agent_id, model, thinking, %s, %s, cron, content, chat_id, task_type FROM task_meta WHERE id IN (%s)`, cronVerifySelect(queryer, "task_meta"), cronRouterDisableSelect(queryer, "task_meta"), placeholders(len(metaIDs))), intsToInterfaces(metaIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item cronMetaResult
		var th, verify, routerDisable int
		if err := rows.Scan(&item.ID, &item.Cycle, &item.RawTime, &item.AgentID, &item.Model, &th, &verify, &routerDisable, &item.Cron, &item.Content, &item.ChatID, &item.Type); err != nil {
			return nil, err
		}
		item.Type = sharedutil.NormalizeTaskType(item.Type)
		item.Thinking = th != 0
		item.Verify = verify != 0
		item.RouterDisable = routerDisable != 0
		result[item.ID] = item
	}
	return result, rows.Err()
}

func loadCronDetailsByMetaIDs(queryer interface {
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryRow(string, ...interface{}) *sql.Row
}, metaIDs []int, unfinishedOnly bool) ([]cronDetailResult, error) {
	if len(metaIDs) == 0 {
		return []cronDetailResult{}, nil
	}
	selectMetaRef := `''`
	if hasTableColumn(queryer, "task_detail", "meta_ref") {
		selectMetaRef = `d.meta_ref`
	}
	query := fmt.Sprintf(`SELECT d.id, d.meta_id, d.exec_time, d.agent_id, d.chat_id, %s, d.task_type, d.model, d.thinking, %s, %s, d.content, d.started, m.cycle, m.raw_time, m.cron
		FROM task_detail d
		LEFT JOIN task_meta m ON m.id = d.meta_id
		WHERE d.meta_id IN (%s)`, selectMetaRef, cronVerifySelectExpr(queryer, "task_detail", "d.verify"), cronRouterDisableSelectExpr(queryer, "task_detail", "d.router_disable"), placeholders(len(metaIDs)))
	if unfinishedOnly {
		query += ` AND d.started != 3`
	}
	query += ` ORDER BY d.id`
	rows, err := queryer.Query(query, intsToInterfaces(metaIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []cronDetailResult
	for rows.Next() {
		var item cronDetailResult
		var th, verify, routerDisable int
		if err := rows.Scan(&item.ID, &item.MetaID, &item.ExecTime, &item.AgentID, &item.ChatID, &item.MetaRef, &item.Type, &item.Model, &th, &verify, &routerDisable, &item.Content, &item.Started, &item.Cycle, &item.RawTime, &item.Cron); err != nil {
			return nil, err
		}
		item.Type = sharedutil.NormalizeTaskType(item.Type)
		item.Thinking = th != 0
		item.Verify = verify != 0
		item.RouterDisable = routerDisable != 0
		items = append(items, item)
	}
	if items == nil {
		items = []cronDetailResult{}
	}
	return items, rows.Err()
}

func loadCronDetailLogMap(db *sql.DB, detailIDs []int) (map[int]cronDetailResult, error) {
	result := make(map[int]cronDetailResult, len(detailIDs))
	if len(detailIDs) == 0 {
		return result, nil
	}
	selectMetaRef := `''`
	if hasTableColumn(db, "task_detail", "meta_ref") {
		selectMetaRef = `d.meta_ref`
	}
	rows, err := db.Query(fmt.Sprintf(`SELECT d.id, d.meta_id, d.exec_time, d.agent_id, d.chat_id, %s, d.task_type, d.model, d.thinking, %s, %s, d.content, d.started, m.cycle, m.raw_time, m.cron
		FROM task_detail d
		LEFT JOIN task_meta m ON m.id = d.meta_id
		WHERE d.id IN (%s)`, selectMetaRef, cronVerifySelectExpr(db, "task_detail", "d.verify"), cronRouterDisableSelectExpr(db, "task_detail", "d.router_disable"), placeholders(len(detailIDs))), intsToInterfaces(detailIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item cronDetailResult
		var th, verify, routerDisable int
		if err := rows.Scan(&item.ID, &item.MetaID, &item.ExecTime, &item.AgentID, &item.ChatID, &item.MetaRef, &item.Type, &item.Model, &th, &verify, &routerDisable, &item.Content, &item.Started, &item.Cycle, &item.RawTime, &item.Cron); err != nil {
			return nil, err
		}
		item.Type = sharedutil.NormalizeTaskType(item.Type)
		item.Thinking = th != 0
		item.Verify = verify != 0
		item.RouterDisable = routerDisable != 0
		result[item.ID] = item
	}
	return result, rows.Err()
}

func deleteAgentCronData(db *sql.DB, agentID string) (int64, int64, error) {
	agentID = strings.TrimSpace(agentID)
	if db == nil || agentID == "" {
		return 0, 0, nil
	}
	metaItems, err := queryCronMetas(db, cronMetaFilter{AgentID: agentID})
	if err != nil {
		return 0, 0, err
	}
	detailItems, err := queryCronDetails(db, cronDetailFilter{AgentID: agentID})
	if err != nil {
		return 0, 0, err
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	detailRes, err := tx.Exec(`DELETE FROM task_detail WHERE agent_id = ?`, agentID)
	if err != nil {
		return 0, 0, err
	}
	detailAffected, err := detailRes.RowsAffected()
	if err != nil {
		return 0, 0, err
	}
	metaRes, err := tx.Exec(`DELETE FROM task_meta WHERE agent_id = ?`, agentID)
	if err != nil {
		return 0, 0, err
	}
	metaAffected, err := metaRes.RowsAffected()
	if err != nil {
		return 0, 0, err
	}
	if _, err := tx.Exec(`DELETE FROM agent_cron_stop WHERE agent_id = ?`, agentID); err != nil {
		return 0, 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}

	for _, item := range detailItems {
		appendCronDetailLog(db, cronDetailLogEntry{
			DetailID:   item.ID,
			MetaID:     item.MetaID,
			AgentID:    item.AgentID,
			ChatID:     item.ChatID,
			TaskType:   item.Type,
			Action:     "delete_agent",
			ExecTime:   item.ExecTime,
			Model:      item.Model,
			Thinking:   item.Thinking,
			Content:    item.Content,
			Started:    item.Started,
			OccurredAt: cronLogTimestamp(),
		})
	}
	for _, item := range metaItems {
		appendCronMetaLog(db, cronMetaLogEntry{
			MetaID:    item.ID,
			AgentID:   item.AgentID,
			ChatID:    item.ChatID,
			TaskType:  item.Type,
			Action:    "delete_agent",
			Cycle:     item.Cycle,
			RawTime:   item.RawTime,
			Model:     item.Model,
			Thinking:  item.Thinking,
			Cron:      item.Cron,
			Content:   item.Content,
			CreatedAt: cronLogTimestamp(),
		})
	}
	return metaAffected, detailAffected, nil
}

func isCronModelConfigured(db *sql.DB, model string) (bool, error) {
	token, err := lookupTokenByModel(db, model)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(token) != "", nil
}

func agentExists(agentDir, deviceID, agentID string) bool {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return false
	}
	agentTTL := 120 * time.Second
	metadata, err := getAgentOutput(agentDir, deviceID, agentTTL)
	if err != nil {
		return false
	}
	for _, agent := range metadata.Agents {
		if strings.TrimSpace(agent.AgentID) == agentID {
			return true
		}
	}
	return false
}

func cleanupCronMetaIDsWithAction(db *sql.DB, metaIDs []int, action string) error {
	if db == nil || len(metaIDs) == 0 {
		return nil
	}

	metaMap, err := loadCronMetaLogMap(db, metaIDs)
	if err != nil {
		return err
	}
	detailItems, err := loadCronDetailsByMetaIDs(db, metaIDs, true)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, metaID := range metaIDs {
		if _, err := tx.Exec(`DELETE FROM task_detail WHERE meta_id = ? AND started != 3`, metaID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(fmt.Sprintf(`DELETE FROM task_meta WHERE id IN (%s)`, placeholders(len(metaIDs))), intsToInterfaces(metaIDs)...); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	for _, detail := range detailItems {
		appendCronDetailLog(db, cronDetailLogEntry{
			DetailID:   detail.ID,
			MetaID:     detail.MetaID,
			AgentID:    detail.AgentID,
			ChatID:     detail.ChatID,
			Action:     action,
			ExecTime:   detail.ExecTime,
			Model:      detail.Model,
			Thinking:   detail.Thinking,
			Content:    detail.Content,
			Started:    detail.Started,
			OccurredAt: cronLogTimestamp(),
		})
	}
	for _, metaID := range metaIDs {
		if item, ok := metaMap[metaID]; ok {
			appendCronMetaLog(db, cronMetaLogEntry{
				MetaID:    item.ID,
				AgentID:   item.AgentID,
				ChatID:    item.ChatID,
				Action:    action,
				Cycle:     item.Cycle,
				RawTime:   item.RawTime,
				Model:     item.Model,
				Thinking:  item.Thinking,
				Cron:      item.Cron,
				Content:   item.Content,
				CreatedAt: cronLogTimestamp(),
			})
		}
	}
	return nil
}

func cleanupDeletedAgentCronData(cfg *Config) {
	if cronDB == nil || cfg == nil {
		return
	}
	metaItems, err := queryCronMetas(cronDB, cronMetaFilter{})
	if err != nil {
		return
	}

	agentExistsCache := make(map[string]bool)
	var deletedAgentMetaIDs []int
	for _, item := range metaItems {
		agentID := strings.TrimSpace(item.AgentID)
		exists, ok := agentExistsCache[agentID]
		if !ok {
			exists = agentExists(cfg.AgentDir, cfg.effectiveDeviceID(), agentID)
			agentExistsCache[agentID] = exists
		}
		if !exists {
			deletedAgentMetaIDs = append(deletedAgentMetaIDs, item.ID)
		}
	}
	if err := cleanupCronMetaIDsWithAction(cronDB, deletedAgentMetaIDs, "cleanup_deleted_agent"); err != nil {
		log.Printf("[cron] cleanup deleted agent task meta failed: %v", err)
	}
}

func cleanupInvalidCronData(cfg *Config) {
	if cronDB == nil || cfg == nil {
		return
	}
	metaItems, err := queryCronMetas(cronDB, cronMetaFilter{})
	if err != nil {
		return
	}

	modelReadyCache := make(map[string]bool)
	var invalidModelMetaIDs []int

	for _, item := range metaItems {
		agentID := strings.TrimSpace(item.AgentID)
		if !agentExists(cfg.AgentDir, cfg.effectiveDeviceID(), agentID) {
			continue
		}

		model := strings.TrimSpace(item.Model)
		ready, ok := modelReadyCache[model]
		if !ok {
			ready, err = isCronModelConfigured(cronDB, model)
			if err != nil {
				log.Printf("[cron] validate model failed: %s: %v", model, err)
				continue
			}
			modelReadyCache[model] = ready
		}
		if !ready {
			invalidModelMetaIDs = append(invalidModelMetaIDs, item.ID)
		}
	}

	cleanupDeletedAgentCronData(cfg)
	if err := cleanupCronMetaIDsWithAction(cronDB, invalidModelMetaIDs, "cleanup_invalid_model"); err != nil {
		log.Printf("[cron] cleanup invalid model task meta failed: %v", err)
	}
}

func placeholders(n int) string {
	items := make([]string, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, "?")
	}
	return strings.Join(items, ",")
}

func intsToInterfaces(items []int) []interface{} {
	values := make([]interface{}, 0, len(items))
	for _, item := range items {
		values = append(values, item)
	}
	return values
}

func buildCronMetaFilter(opts cronQueryOptions) (cronMetaFilter, error) {
	startFrom, startTo, err := resolveTimeRange(opts.Time, opts.Date, opts.From, opts.To, false)
	if err != nil {
		return cronMetaFilter{}, err
	}
	return cronMetaFilter{
		AgentID:   strings.TrimSpace(opts.AgentID),
		ChatID:    strings.TrimSpace(opts.ChatID),
		TaskType:  strings.TrimSpace(opts.TaskType),
		Model:     strings.TrimSpace(opts.Model),
		Content:   strings.TrimSpace(opts.Content),
		Cycle:     opts.Cycle,
		StartFrom: startFrom,
		StartTo:   startTo,
	}, nil
}

func buildCronDetailFilter(opts cronQueryOptions) (cronDetailFilter, error) {
	metaID, err := parseCronMetaIDValue(opts.MetaID)
	if err != nil {
		return cronDetailFilter{}, err
	}
	execFrom, execTo, err := resolveTimeRange(opts.Time, opts.Date, opts.From, opts.To, !opts.IncludeHistory)
	if err != nil {
		return cronDetailFilter{}, err
	}
	return cronDetailFilter{
		MetaID:   metaID,
		AgentID:  strings.TrimSpace(opts.AgentID),
		ChatID:   strings.TrimSpace(opts.ChatID),
		TaskType: strings.TrimSpace(opts.TaskType),
		Model:    strings.TrimSpace(opts.Model),
		Content:  strings.TrimSpace(opts.Content),
		Cycle:    opts.Cycle,
		ExecFrom: execFrom,
		ExecTo:   execTo,
	}, nil
}

func buildCronDetailDeleteFilter(opts cronQueryOptions) (cronDetailFilter, error) {
	metaID, err := parseCronMetaIDValue(opts.MetaID)
	if err != nil {
		return cronDetailFilter{}, err
	}
	execFrom, execTo, err := resolveTimeRange(opts.Time, opts.Date, opts.From, opts.To, false)
	if err != nil {
		return cronDetailFilter{}, err
	}
	return cronDetailFilter{
		MetaID:   metaID,
		AgentID:  strings.TrimSpace(opts.AgentID),
		ChatID:   strings.TrimSpace(opts.ChatID),
		TaskType: strings.TrimSpace(opts.TaskType),
		Model:    strings.TrimSpace(opts.Model),
		Content:  strings.TrimSpace(opts.Content),
		Cycle:    opts.Cycle,
		ExecFrom: execFrom,
		ExecTo:   execTo,
	}, nil
}

func parseCronMetaFilterFromRequest(r *http.Request) (cronMetaFilter, error) {
	q := r.URL.Query()
	cycle, err := parseOptionalIntValue(q.Get("cycle"))
	if err != nil {
		return cronMetaFilter{}, err
	}
	status, err := parseCronStatusFilter(q.Get("status"))
	if err != nil {
		return cronMetaFilter{}, err
	}
	filter, err := buildCronMetaFilter(cronQueryOptions{
		AgentID:  firstNonEmpty(q.Get("agentId"), q.Get("agent")),
		ChatID:   firstNonEmpty(q.Get("chatId"), q.Get("chat")),
		TaskType: q.Get("type"),
		Model:    q.Get("model"),
		Content:  q.Get("content"),
		Cycle:    cycle,
		Time:     firstNonEmpty(q.Get("time"), q.Get("rawTime"), q.Get("raw-time"), q.Get("start"), q.Get("startTime"), q.Get("start-time")),
		Date:     q.Get("date"),
		From:     firstNonEmpty(q.Get("from"), q.Get("startFrom"), q.Get("start-from")),
		To:       firstNonEmpty(q.Get("to"), q.Get("startTo"), q.Get("start-to")),
	})
	if err != nil {
		return cronMetaFilter{}, err
	}
	filter.RecurringOnly = q.Get("recurringOnly") == "1"
	filter.Status = status
	return filter, nil
}

func parseCronDetailFilterFromRequest(r *http.Request) (cronDetailFilter, error) {
	q := r.URL.Query()
	cycle, err := parseOptionalIntValue(q.Get("cycle"))
	if err != nil {
		return cronDetailFilter{}, err
	}
	status, err := parseCronStatusFilter(q.Get("status"))
	if err != nil {
		return cronDetailFilter{}, err
	}
	filter, err := buildCronDetailFilter(cronQueryOptions{
		MetaID:         firstNonEmpty(q.Get("metaId"), q.Get("meta"), q.Get("id")),
		AgentID:        firstNonEmpty(q.Get("agentId"), q.Get("agent")),
		ChatID:         firstNonEmpty(q.Get("chatId"), q.Get("chat")),
		TaskType:       q.Get("type"),
		Model:          q.Get("model"),
		Content:        q.Get("content"),
		Cycle:          cycle,
		Time:           firstNonEmpty(q.Get("time"), q.Get("execTime"), q.Get("exec-time")),
		Date:           q.Get("date"),
		From:           firstNonEmpty(q.Get("from"), q.Get("execFrom"), q.Get("exec-from")),
		To:             firstNonEmpty(q.Get("to"), q.Get("execTo"), q.Get("exec-to")),
		IncludeHistory: q.Get("history") == "1",
	})
	if err != nil {
		return cronDetailFilter{}, err
	}
	filter.Status = status
	return filter, nil
}

func parseCronDetailDeleteFilterFromRequest(r *http.Request) (cronDetailFilter, error) {
	q := r.URL.Query()
	cycle, err := parseOptionalIntValue(q.Get("cycle"))
	if err != nil {
		return cronDetailFilter{}, err
	}
	return buildCronDetailDeleteFilter(cronQueryOptions{
		MetaID:   firstNonEmpty(q.Get("metaId"), q.Get("meta"), q.Get("id")),
		AgentID:  firstNonEmpty(q.Get("agentId"), q.Get("agent")),
		ChatID:   firstNonEmpty(q.Get("chatId"), q.Get("chat")),
		TaskType: q.Get("type"),
		Model:    q.Get("model"),
		Content:  q.Get("content"),
		Cycle:    cycle,
		Time:     firstNonEmpty(q.Get("time"), q.Get("execTime"), q.Get("exec-time")),
		Date:     q.Get("date"),
		From:     firstNonEmpty(q.Get("from"), q.Get("execFrom"), q.Get("exec-from")),
		To:       firstNonEmpty(q.Get("to"), q.Get("execTo"), q.Get("exec-to")),
	})
}

func escapeCronLikeContains(value string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
}

func queryCronMetas(db *sql.DB, filter cronMetaFilter) ([]cronMetaResult, error) {
	selectResponseSchema := `response_schema`
	if !hasTableColumn(db, "task_meta", "response_schema") {
		selectResponseSchema = `'' AS response_schema`
	}
	query := fmt.Sprintf(`SELECT id, cycle, raw_time, agent_id, model, thinking, %s, %s, cron, content, %s, chat_id, task_type,
		COALESCE((SELECT GROUP_CONCAT(DISTINCT status_detail.started) FROM task_detail status_detail WHERE status_detail.meta_id = task_meta.id), '')
		FROM task_meta WHERE 1=1`, cronVerifySelect(db, "task_meta"), cronRouterDisableSelect(db, "task_meta"), selectResponseSchema)
	args := make([]interface{}, 0, 10)
	if filter.AgentID != "" {
		query += ` AND agent_id = ?`
		args = append(args, filter.AgentID)
	}
	if filter.ChatID != "" {
		query += ` AND chat_id = ?`
		args = append(args, filter.ChatID)
	}
	if filter.TaskType != "" {
		query += ` AND task_type = ?`
		args = append(args, filter.TaskType)
	}
	if filter.Model != "" {
		query += ` AND model = ?`
		args = append(args, filter.Model)
	}
	if filter.Content != "" {
		query += ` AND content LIKE ? ESCAPE '\'`
		args = append(args, "%"+escapeCronLikeContains(filter.Content)+"%")
	}
	if filter.RecurringOnly {
		query += ` AND cycle <> 0`
	}
	if filter.Status != nil {
		query += ` AND EXISTS (SELECT 1 FROM task_detail status_detail WHERE status_detail.meta_id = task_meta.id AND status_detail.started = ?)`
		args = append(args, *filter.Status)
	}
	if filter.Cycle != nil {
		query += ` AND cycle = ?`
		args = append(args, *filter.Cycle)
	}
	if filter.StartFrom != nil || filter.StartTo != nil {
		query += ` AND raw_time <> ''`
	}
	if filter.StartFrom != nil {
		query += ` AND raw_time >= ?`
		args = append(args, filter.StartFrom.Format("2006-01-02 15:04"))
	}
	if filter.StartTo != nil {
		query += ` AND raw_time <= ?`
		args = append(args, filter.StartTo.Format("2006-01-02 15:04"))
	}
	query += ` ORDER BY raw_time DESC, id DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	metas := make([]cronMetaResult, 0)
	for rows.Next() {
		var (
			item          cronMetaResult
			th            int
			verify        int
			routerDisable int
		)
		if err := rows.Scan(&item.ID, &item.Cycle, &item.RawTime, &item.AgentID, &item.Model, &th, &verify, &routerDisable, &item.Cron, &item.Content, &item.ResponseSchema, &item.ChatID, &item.Type, &item.DetailStatuses); err != nil {
			return nil, err
		}
		item.Type = sharedutil.NormalizeTaskType(item.Type)
		item.Thinking = th != 0
		item.Verify = verify != 0
		item.RouterDisable = routerDisable != 0
		metas = append(metas, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if metas == nil {
		metas = []cronMetaResult{}
	}
	for _, item := range metas {
		appendCronMetaLog(db, cronMetaLogEntry{
			MetaID:         item.ID,
			AgentID:        item.AgentID,
			ChatID:         item.ChatID,
			TaskType:       item.Type,
			Action:         "select",
			Cycle:          item.Cycle,
			RawTime:        item.RawTime,
			Model:          item.Model,
			Thinking:       item.Thinking,
			Verify:         item.Verify,
			Cron:           item.Cron,
			Content:        item.Content,
			ResponseSchema: item.ResponseSchema,
			CreatedAt:      cronLogTimestamp(),
		})
	}
	return metas, nil
}

func queryCronDetails(db *sql.DB, filter cronDetailFilter) ([]cronDetailResult, error) {
	selectResponseSchema := `d.response_schema`
	if !hasTableColumn(db, "task_detail", "response_schema") {
		selectResponseSchema = `'' AS response_schema`
	}
	selectResultContent := `d.result_content`
	if !hasTableColumn(db, "task_detail", "result_content") {
		selectResultContent = `'' AS result_content`
	}
	selectRepliedAt := `d.replied_at`
	if !hasTableColumn(db, "task_detail", "replied_at") {
		selectRepliedAt = `'' AS replied_at`
	}
	query := fmt.Sprintf(`SELECT d.id, d.meta_id, d.exec_time, d.agent_id, d.chat_id, d.task_type, d.model, d.thinking, %s, %s, d.content, %s, %s, %s, d.started,
			COALESCE(m.cycle, 0), COALESCE(m.raw_time, ''), COALESCE(m.cron, '')
		FROM task_detail d
		LEFT JOIN task_meta m ON m.id = d.meta_id
		WHERE 1=1`, cronVerifySelectExpr(db, "task_detail", "d.verify"), cronRouterDisableSelectExpr(db, "task_detail", "d.router_disable"), selectResponseSchema, selectResultContent, selectRepliedAt)
	args := make([]interface{}, 0, 12)
	if filter.DetailID != nil {
		query += ` AND d.id = ?`
		args = append(args, *filter.DetailID)
	}
	if filter.MetaID != nil {
		query += ` AND d.meta_id = ?`
		args = append(args, *filter.MetaID)
	}
	if filter.AgentID != "" {
		query += ` AND d.agent_id = ?`
		args = append(args, filter.AgentID)
	}
	if filter.ChatID != "" {
		query += ` AND d.chat_id = ?`
		args = append(args, filter.ChatID)
	}
	if filter.TaskType != "" {
		query += ` AND d.task_type = ?`
		args = append(args, filter.TaskType)
	}
	if filter.Model != "" {
		query += ` AND d.model = ?`
		args = append(args, filter.Model)
	}
	if filter.Content != "" {
		query += ` AND d.content LIKE ? ESCAPE '\'`
		args = append(args, "%"+escapeCronLikeContains(filter.Content)+"%")
	}
	if filter.Cycle != nil {
		query += ` AND m.cycle = ?`
		args = append(args, *filter.Cycle)
	}
	if filter.Status != nil {
		query += ` AND d.started = ?`
		args = append(args, *filter.Status)
	}
	if filter.ExecFrom != nil {
		query += ` AND d.exec_time >= ?`
		args = append(args, filter.ExecFrom.Unix())
	}
	if filter.ExecTo != nil {
		query += ` AND d.exec_time <= ?`
		args = append(args, filter.ExecTo.Unix())
	}
	query += ` ORDER BY d.exec_time DESC, d.id DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	details := make([]cronDetailResult, 0)
	for rows.Next() {
		var (
			item          cronDetailResult
			th            int
			verify        int
			routerDisable int
		)
		if err := rows.Scan(&item.ID, &item.MetaID, &item.ExecTime, &item.AgentID, &item.ChatID, &item.Type, &item.Model, &th, &verify, &routerDisable, &item.Content, &item.ResponseSchema, &item.ResultContent, &item.RepliedAt, &item.Started, &item.Cycle, &item.RawTime, &item.Cron); err != nil {
			return nil, err
		}
		item.Type = sharedutil.NormalizeTaskType(item.Type)
		if item.Cycle == 0 && item.RawTime == "" && item.Cron == "" && item.Started == 3 {
			// Completed details can outlive deleted task_meta rows; keep them queryable.
			item.RawTime = time.Unix(item.ExecTime, 0).In(time.Local).Format("2006-01-02 15:04")
		}
		item.Thinking = th != 0
		item.Verify = verify != 0
		item.RouterDisable = routerDisable != 0
		details = append(details, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if details == nil {
		details = []cronDetailResult{}
	}
	for _, item := range details {
		appendCronDetailLog(db, cronDetailLogEntry{
			DetailID:       item.ID,
			MetaID:         item.MetaID,
			AgentID:        item.AgentID,
			ChatID:         item.ChatID,
			TaskType:       item.Type,
			Action:         "select",
			ExecTime:       item.ExecTime,
			Model:          item.Model,
			Thinking:       item.Thinking,
			Verify:         item.Verify,
			RouterDisable:  item.RouterDisable,
			Content:        item.Content,
			ResponseSchema: item.ResponseSchema,
			Started:        item.Started,
			OccurredAt:     cronLogTimestamp(),
		})
	}
	return details, nil
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func cronLogTimestamp() string {
	return time.Now().Format("2006-01-02T15:04:05.000")
}

func renameCronLogTables(db *sql.DB) {
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table'`)
	if err != nil {
		return
	}
	defer rows.Close()

	foundMeta := false
	foundDetail := false
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		switch name {
		case "cron_meta_log":
			foundMeta = true
		case "cron_detail_log":
			foundDetail = true
		case "task_meta_log":
			if !foundMeta {
				db.Exec(`ALTER TABLE task_meta_log RENAME TO cron_meta_log`)
				foundMeta = true
			}
		case "task_detail_log":
			if !foundDetail {
				db.Exec(`ALTER TABLE task_detail_log RENAME TO cron_detail_log`)
				foundDetail = true
			}
		}
	}
}

type pragmaRowQueryer interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}

const defaultTaskType = sharedutil.DefaultTaskType

func hasTableColumn(queryer pragmaRowQueryer, tableName, columnName string) bool {
	if queryer == nil {
		return false
	}
	row := queryer.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name = ?`, tableName), columnName)
	var count int
	if err := row.Scan(&count); err != nil {
		return false
	}
	return count > 0
}

func cronRouterDisableSelectExpr(queryer pragmaRowQueryer, tableName, expr string) string {
	if hasTableColumn(queryer, tableName, "router_disable") {
		return expr
	}
	return "1 AS router_disable"
}

func ensureCronRouterDisableColumn(db *sql.DB, tableName string) {
	if db == nil || hasTableColumn(db, tableName, "router_disable") {
		return
	}
	_, _ = db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN router_disable INTEGER NOT NULL DEFAULT 1`, tableName))
}

func ensureCronVerifyColumn(db *sql.DB, tableName string) {
	if db == nil || hasTableColumn(db, tableName, "verify") {
		return
	}
	_, _ = db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN verify INTEGER NOT NULL DEFAULT 0`, tableName))
}

func cronRouterDisableSelect(queryer pragmaRowQueryer, tableName string) string {
	return cronRouterDisableSelectExpr(queryer, tableName, "router_disable")
}

func dropCronLegacySessionDisableColumn(db *sql.DB, tableName string) {
	if db == nil || !hasTableColumn(db, tableName, "session_disable") {
		return
	}
	if _, err := db.Exec(fmt.Sprintf(`ALTER TABLE %s DROP COLUMN session_disable`, tableName)); err != nil {
		log.Printf("[cron] remove legacy session_disable column failed: table=%s err=%v", tableName, err)
	}
}

func cronVerifySelectExpr(queryer pragmaRowQueryer, tableName, expr string) string {
	if hasTableColumn(queryer, tableName, "verify") {
		return expr
	}
	return "0 AS verify"
}

func cronVerifySelect(queryer pragmaRowQueryer, tableName string) string {
	return cronVerifySelectExpr(queryer, tableName, "verify")
}

func ensureCronSchema(db *sql.DB) {
	if db == nil {
		return
	}
	renameCronLogTables(db)
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS task_meta (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cycle INTEGER NOT NULL,
			raw_time TEXT NOT NULL DEFAULT '',
			agent_id TEXT NOT NULL,
			chat_id TEXT NOT NULL DEFAULT '',
			task_type TEXT NOT NULL DEFAULT 'cron',
			model TEXT NOT NULL,
			thinking INTEGER NOT NULL DEFAULT 0,
			verify INTEGER NOT NULL DEFAULT 0,
			router_disable INTEGER NOT NULL DEFAULT 1,
			cron TEXT NOT NULL,
			content TEXT NOT NULL,
			response_schema TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE IF NOT EXISTS task_detail (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			meta_id INTEGER NOT NULL,
			exec_time INTEGER NOT NULL,
			agent_id TEXT NOT NULL,
			chat_id TEXT NOT NULL DEFAULT '',
			meta_ref TEXT NOT NULL DEFAULT '',
			task_type TEXT NOT NULL DEFAULT 'cron',
			model TEXT NOT NULL,
			thinking INTEGER NOT NULL DEFAULT 0,
			verify INTEGER NOT NULL DEFAULT 0,
			router_disable INTEGER NOT NULL DEFAULT 1,
			content TEXT NOT NULL,
			response_schema TEXT NOT NULL DEFAULT '',
			result_content TEXT NOT NULL DEFAULT '',
			replied_at TEXT NOT NULL DEFAULT '',
			started INTEGER NOT NULL DEFAULT 0,
			UNIQUE(meta_id, exec_time)
		);
		CREATE TABLE IF NOT EXISTS cron_meta_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			meta_id INTEGER NOT NULL,
			agent_id TEXT NOT NULL,
			chat_id TEXT NOT NULL DEFAULT '',
			task_type TEXT NOT NULL DEFAULT 'cron',
			action TEXT NOT NULL,
			cycle INTEGER NOT NULL,
			raw_time TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL,
			thinking INTEGER NOT NULL DEFAULT 0,
			verify INTEGER NOT NULL DEFAULT 0,
			router_disable INTEGER NOT NULL DEFAULT 1,
			cron TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			response_schema TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS cron_detail_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			detail_id INTEGER NOT NULL,
			meta_id INTEGER NOT NULL,
			agent_id TEXT NOT NULL,
			chat_id TEXT NOT NULL DEFAULT '',
			task_type TEXT NOT NULL DEFAULT 'cron',
			action TEXT NOT NULL,
			exec_time INTEGER NOT NULL,
			model TEXT NOT NULL,
			thinking INTEGER NOT NULL DEFAULT 0,
			verify INTEGER NOT NULL DEFAULT 0,
			router_disable INTEGER NOT NULL DEFAULT 1,
			content TEXT NOT NULL,
			response_schema TEXT NOT NULL DEFAULT '',
			started INTEGER NOT NULL DEFAULT 0,
			occurred_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS agent_cron_stop (
			agent_id TEXT PRIMARY KEY,
			enabled INTEGER NOT NULL DEFAULT 0,
			stopped_at INTEGER NOT NULL DEFAULT 0,
			resume_from INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		);
	`); err != nil {
		log.Printf("[cron] ensure schema create failed: %v", err)
	}
	for _, stmt := range []string{
		`ALTER TABLE task_meta ADD COLUMN chat_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE task_meta ADD COLUMN task_type TEXT NOT NULL DEFAULT 'cron'`,
		`ALTER TABLE task_meta ADD COLUMN response_schema TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE task_meta ADD COLUMN router_disable INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE task_detail ADD COLUMN chat_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE task_detail ADD COLUMN meta_ref TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE task_detail ADD COLUMN started INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE task_detail ADD COLUMN task_type TEXT NOT NULL DEFAULT 'cron'`,
		`ALTER TABLE task_detail ADD COLUMN response_schema TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE task_detail ADD COLUMN router_disable INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE task_detail ADD COLUMN result_content TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE task_detail ADD COLUMN replied_at TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE cron_meta_log ADD COLUMN chat_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE cron_meta_log ADD COLUMN task_type TEXT NOT NULL DEFAULT 'cron'`,
		`ALTER TABLE cron_meta_log ADD COLUMN response_schema TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE cron_meta_log ADD COLUMN router_disable INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE cron_detail_log ADD COLUMN chat_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE cron_detail_log ADD COLUMN started INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE cron_detail_log ADD COLUMN task_type TEXT NOT NULL DEFAULT 'cron'`,
		`ALTER TABLE cron_detail_log ADD COLUMN response_schema TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE cron_detail_log ADD COLUMN router_disable INTEGER NOT NULL DEFAULT 1`,
		`DROP INDEX IF EXISTS idx_detail_agent_time`,
		`DROP INDEX IF EXISTS idx_detail_agent_chat_time`,
		`DROP INDEX IF EXISTS idx_detail_agent_chat_time_type`,
		`CREATE INDEX IF NOT EXISTS idx_detail_agent_chat_time_type ON task_detail(agent_id, chat_id, exec_time, task_type)`,
		`CREATE INDEX IF NOT EXISTS idx_meta_agent_raw_time_id ON task_meta(agent_id, raw_time DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_meta_model_raw_time_id ON task_meta(model, raw_time DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_meta_cycle_raw_time_id ON task_meta(cycle, raw_time DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_meta_agent_model_cycle_raw_time_id ON task_meta(agent_id, model, cycle, raw_time DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_detail_agent_exec_time_id ON task_detail(agent_id, exec_time DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_detail_model_exec_time_id ON task_detail(model, exec_time DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_detail_status_exec_time_id ON task_detail(started, exec_time DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_detail_agent_model_status_exec_time_id ON task_detail(agent_id, model, started, exec_time DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_detail_meta_status ON task_detail(meta_id, started)`,
		`CREATE INDEX IF NOT EXISTS idx_meta_agent ON task_meta(agent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cron_meta_log_agent_chat_time ON cron_meta_log(agent_id, chat_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_cron_meta_log_meta_time ON cron_meta_log(meta_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_cron_detail_log_agent_chat_time ON cron_detail_log(agent_id, chat_id, occurred_at)`,
		`CREATE INDEX IF NOT EXISTS idx_cron_detail_log_meta_time ON cron_detail_log(meta_id, occurred_at)`,
		`CREATE INDEX IF NOT EXISTS idx_cron_detail_log_detail_time ON cron_detail_log(detail_id, occurred_at)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			lower := strings.ToLower(err.Error())
			if !strings.Contains(lower, "duplicate column name") {
				log.Printf("[cron] ensure schema stmt failed: %s: %v", stmt, err)
			}
		}
	}
	for _, tableName := range []string{"task_meta", "task_detail", "cron_meta_log", "cron_detail_log"} {
		ensureCronVerifyColumn(db, tableName)
		ensureCronRouterDisableColumn(db, tableName)
	}
	dropCronLegacySessionDisableColumn(db, "task_meta")
	dropCronLegacySessionDisableColumn(db, "task_detail")
}

type cronAgentStopState struct {
	AgentID    string
	Enabled    bool
	StoppedAt  int64
	ResumeFrom int64
	UpdatedAt  int64
}

func getCronAgentStopState(db *sql.DB, agentID string) (cronAgentStopState, error) {
	state := cronAgentStopState{AgentID: strings.TrimSpace(agentID)}
	if db == nil || state.AgentID == "" {
		return state, nil
	}
	var enabled int
	err := db.QueryRow(`SELECT enabled, stopped_at, resume_from, updated_at FROM agent_cron_stop WHERE agent_id = ?`, state.AgentID).Scan(&enabled, &state.StoppedAt, &state.ResumeFrom, &state.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	state.Enabled = enabled != 0
	return state, nil
}

func getCronAgentStopStateMap(db *sql.DB, agentIDs []string) (map[string]cronAgentStopState, error) {
	states := make(map[string]cronAgentStopState)
	unique := make([]string, 0, len(agentIDs))
	seen := make(map[string]struct{}, len(agentIDs))
	for _, rawAgentID := range agentIDs {
		agentID := strings.TrimSpace(rawAgentID)
		if agentID == "" {
			continue
		}
		if _, ok := seen[agentID]; ok {
			continue
		}
		seen[agentID] = struct{}{}
		unique = append(unique, agentID)
		states[agentID] = cronAgentStopState{AgentID: agentID}
	}
	if db == nil || len(unique) == 0 {
		return states, nil
	}
	args := make([]interface{}, len(unique))
	for i, agentID := range unique {
		args[i] = agentID
	}
	rows, err := db.Query(fmt.Sprintf(`SELECT agent_id, enabled, stopped_at, resume_from, updated_at FROM agent_cron_stop WHERE agent_id IN (%s)`, placeholders(len(unique))), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var state cronAgentStopState
		var enabled int
		if err := rows.Scan(&state.AgentID, &enabled, &state.StoppedAt, &state.ResumeFrom, &state.UpdatedAt); err != nil {
			return nil, err
		}
		state.Enabled = enabled != 0
		states[state.AgentID] = state
	}
	return states, rows.Err()
}

func setCronAgentStopState(db *sql.DB, agentID string, enabled bool, now time.Time) (cronAgentStopState, error) {
	resolvedAgentID := strings.TrimSpace(agentID)
	if db == nil || resolvedAgentID == "" {
		return cronAgentStopState{}, fmt.Errorf("agent stop state is unavailable")
	}
	tx, err := db.Begin()
	if err != nil {
		return cronAgentStopState{}, err
	}
	defer tx.Rollback()
	state := cronAgentStopState{AgentID: resolvedAgentID}
	var storedEnabled int
	err = tx.QueryRow(`SELECT enabled, stopped_at, resume_from, updated_at FROM agent_cron_stop WHERE agent_id = ?`, resolvedAgentID).Scan(&storedEnabled, &state.StoppedAt, &state.ResumeFrom, &state.UpdatedAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return cronAgentStopState{}, err
	}
	state.Enabled = storedEnabled != 0
	nowUnix := now.Unix()
	if enabled {
		if !state.Enabled {
			state.StoppedAt = nowUnix
		}
		state.Enabled = true
	} else {
		if state.Enabled && state.StoppedAt > 0 {
			if state.ResumeFrom == 0 || state.StoppedAt < state.ResumeFrom {
				state.ResumeFrom = state.StoppedAt
			}
		}
		state.Enabled = false
	}
	state.UpdatedAt = nowUnix
	if _, err := tx.Exec(`INSERT INTO agent_cron_stop (agent_id, enabled, stopped_at, resume_from, updated_at) VALUES (?,?,?,?,?)
		ON CONFLICT(agent_id) DO UPDATE SET enabled = excluded.enabled, stopped_at = excluded.stopped_at, resume_from = excluded.resume_from, updated_at = excluded.updated_at`,
		state.AgentID, boolToInt(state.Enabled), state.StoppedAt, state.ResumeFrom, state.UpdatedAt); err != nil {
		return cronAgentStopState{}, err
	}
	if err := tx.Commit(); err != nil {
		return cronAgentStopState{}, err
	}
	return state, nil
}

func clearDrainedCronAgentStopResumeWindows(db *sql.DB, nowUnix int64) {
	if db == nil {
		return
	}
	_, _ = db.Exec(`UPDATE agent_cron_stop
		SET resume_from = 0
		WHERE enabled = 0
		  AND resume_from > 0
		  AND NOT EXISTS (
			SELECT 1 FROM task_detail d
			WHERE d.agent_id = agent_cron_stop.agent_id
			  AND d.task_type = 'cron'
			  AND d.started = 0
			  AND d.exec_time >= agent_cron_stop.resume_from
			  AND d.exec_time <= ?
		  )`, nowUnix)
}

func appendCronMetaLog(db *sql.DB, entry cronMetaLogEntry) {
	if db == nil {
		return
	}
	ensureCronVerifyColumn(db, "cron_meta_log")
	ensureCronRouterDisableColumn(db, "cron_meta_log")
	hasVerify := hasTableColumn(db, "cron_meta_log", "verify")
	hasRouterDisable := hasTableColumn(db, "cron_meta_log", "router_disable")
	hasResponseSchema := hasTableColumn(db, "cron_meta_log", "response_schema")
	query := `INSERT INTO cron_meta_log (meta_id, agent_id, chat_id, task_type, action, cycle, raw_time, model, thinking, verify, router_disable, cron, content, response_schema, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
	args := []interface{}{entry.MetaID, entry.AgentID, entry.ChatID, entry.TaskType, entry.Action, entry.Cycle, entry.RawTime, entry.Model, boolToInt(entry.Thinking), boolToInt(entry.Verify), boolToInt(entry.RouterDisable), entry.Cron, entry.Content, entry.ResponseSchema, entry.CreatedAt}
	switch {
	case hasVerify && hasRouterDisable && hasResponseSchema:
	case hasVerify && hasRouterDisable:
		query = `INSERT INTO cron_meta_log (meta_id, agent_id, chat_id, task_type, action, cycle, raw_time, model, thinking, verify, router_disable, cron, content, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
		args = []interface{}{entry.MetaID, entry.AgentID, entry.ChatID, entry.TaskType, entry.Action, entry.Cycle, entry.RawTime, entry.Model, boolToInt(entry.Thinking), boolToInt(entry.Verify), boolToInt(entry.RouterDisable), entry.Cron, entry.Content, entry.CreatedAt}
	case hasVerify && hasResponseSchema:
		query = `INSERT INTO cron_meta_log (meta_id, agent_id, chat_id, task_type, action, cycle, raw_time, model, thinking, verify, cron, content, response_schema, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
		args = []interface{}{entry.MetaID, entry.AgentID, entry.ChatID, entry.TaskType, entry.Action, entry.Cycle, entry.RawTime, entry.Model, boolToInt(entry.Thinking), boolToInt(entry.Verify), entry.Cron, entry.Content, entry.ResponseSchema, entry.CreatedAt}
	case hasVerify:
		query = `INSERT INTO cron_meta_log (meta_id, agent_id, chat_id, task_type, action, cycle, raw_time, model, thinking, verify, cron, content, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`
		args = []interface{}{entry.MetaID, entry.AgentID, entry.ChatID, entry.TaskType, entry.Action, entry.Cycle, entry.RawTime, entry.Model, boolToInt(entry.Thinking), boolToInt(entry.Verify), entry.Cron, entry.Content, entry.CreatedAt}
	case hasRouterDisable && hasResponseSchema:
		query = `INSERT INTO cron_meta_log (meta_id, agent_id, chat_id, task_type, action, cycle, raw_time, model, thinking, router_disable, cron, content, response_schema, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
		args = []interface{}{entry.MetaID, entry.AgentID, entry.ChatID, entry.TaskType, entry.Action, entry.Cycle, entry.RawTime, entry.Model, boolToInt(entry.Thinking), boolToInt(entry.RouterDisable), entry.Cron, entry.Content, entry.ResponseSchema, entry.CreatedAt}
	case hasRouterDisable:
		query = `INSERT INTO cron_meta_log (meta_id, agent_id, chat_id, task_type, action, cycle, raw_time, model, thinking, router_disable, cron, content, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`
		args = []interface{}{entry.MetaID, entry.AgentID, entry.ChatID, entry.TaskType, entry.Action, entry.Cycle, entry.RawTime, entry.Model, boolToInt(entry.Thinking), boolToInt(entry.RouterDisable), entry.Cron, entry.Content, entry.CreatedAt}
	case hasResponseSchema:
		query = `INSERT INTO cron_meta_log (meta_id, agent_id, chat_id, task_type, action, cycle, raw_time, model, thinking, cron, content, response_schema, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`
		args = []interface{}{entry.MetaID, entry.AgentID, entry.ChatID, entry.TaskType, entry.Action, entry.Cycle, entry.RawTime, entry.Model, boolToInt(entry.Thinking), entry.Cron, entry.Content, entry.ResponseSchema, entry.CreatedAt}
	default:
		query = `INSERT INTO cron_meta_log (meta_id, agent_id, chat_id, task_type, action, cycle, raw_time, model, thinking, cron, content, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`
		args = []interface{}{entry.MetaID, entry.AgentID, entry.ChatID, entry.TaskType, entry.Action, entry.Cycle, entry.RawTime, entry.Model, boolToInt(entry.Thinking), entry.Cron, entry.Content, entry.CreatedAt}
	}
	_, err := db.Exec(query, args...)
	if err != nil {
		log.Printf("[cron] append meta log failed: %v", err)
	}
}

func appendCronDetailLog(db *sql.DB, entry cronDetailLogEntry) {
	if db == nil {
		return
	}
	ensureCronVerifyColumn(db, "cron_detail_log")
	ensureCronRouterDisableColumn(db, "cron_detail_log")
	hasVerify := hasTableColumn(db, "cron_detail_log", "verify")
	hasRouterDisable := hasTableColumn(db, "cron_detail_log", "router_disable")
	hasResponseSchema := hasTableColumn(db, "cron_detail_log", "response_schema")
	query := `INSERT INTO cron_detail_log (detail_id, meta_id, agent_id, chat_id, task_type, action, exec_time, model, thinking, verify, router_disable, content, response_schema, started, occurred_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
	args := []interface{}{entry.DetailID, entry.MetaID, entry.AgentID, entry.ChatID, entry.TaskType, entry.Action, entry.ExecTime, entry.Model, boolToInt(entry.Thinking), boolToInt(entry.Verify), boolToInt(entry.RouterDisable), entry.Content, entry.ResponseSchema, entry.Started, entry.OccurredAt}
	switch {
	case hasVerify && hasRouterDisable && hasResponseSchema:
	case hasVerify && hasRouterDisable:
		query = `INSERT INTO cron_detail_log (detail_id, meta_id, agent_id, chat_id, task_type, action, exec_time, model, thinking, verify, router_disable, content, started, occurred_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
		args = []interface{}{entry.DetailID, entry.MetaID, entry.AgentID, entry.ChatID, entry.TaskType, entry.Action, entry.ExecTime, entry.Model, boolToInt(entry.Thinking), boolToInt(entry.Verify), boolToInt(entry.RouterDisable), entry.Content, entry.Started, entry.OccurredAt}
	case hasVerify && hasResponseSchema:
		query = `INSERT INTO cron_detail_log (detail_id, meta_id, agent_id, chat_id, task_type, action, exec_time, model, thinking, verify, content, response_schema, started, occurred_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
		args = []interface{}{entry.DetailID, entry.MetaID, entry.AgentID, entry.ChatID, entry.TaskType, entry.Action, entry.ExecTime, entry.Model, boolToInt(entry.Thinking), boolToInt(entry.Verify), entry.Content, entry.ResponseSchema, entry.Started, entry.OccurredAt}
	case hasVerify:
		query = `INSERT INTO cron_detail_log (detail_id, meta_id, agent_id, chat_id, task_type, action, exec_time, model, thinking, verify, content, started, occurred_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`
		args = []interface{}{entry.DetailID, entry.MetaID, entry.AgentID, entry.ChatID, entry.TaskType, entry.Action, entry.ExecTime, entry.Model, boolToInt(entry.Thinking), boolToInt(entry.Verify), entry.Content, entry.Started, entry.OccurredAt}
	case hasRouterDisable && hasResponseSchema:
	case hasRouterDisable:
		query = `INSERT INTO cron_detail_log (detail_id, meta_id, agent_id, chat_id, task_type, action, exec_time, model, thinking, router_disable, content, started, occurred_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`
		args = []interface{}{entry.DetailID, entry.MetaID, entry.AgentID, entry.ChatID, entry.TaskType, entry.Action, entry.ExecTime, entry.Model, boolToInt(entry.Thinking), boolToInt(entry.RouterDisable), entry.Content, entry.Started, entry.OccurredAt}
	case hasResponseSchema:
		query = `INSERT INTO cron_detail_log (detail_id, meta_id, agent_id, chat_id, task_type, action, exec_time, model, thinking, content, response_schema, started, occurred_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`
		args = []interface{}{entry.DetailID, entry.MetaID, entry.AgentID, entry.ChatID, entry.TaskType, entry.Action, entry.ExecTime, entry.Model, boolToInt(entry.Thinking), entry.Content, entry.ResponseSchema, entry.Started, entry.OccurredAt}
	default:
		query = `INSERT INTO cron_detail_log (detail_id, meta_id, agent_id, chat_id, task_type, action, exec_time, model, thinking, content, started, occurred_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`
		args = []interface{}{entry.DetailID, entry.MetaID, entry.AgentID, entry.ChatID, entry.TaskType, entry.Action, entry.ExecTime, entry.Model, boolToInt(entry.Thinking), entry.Content, entry.Started, entry.OccurredAt}
	}
	_, err := db.Exec(query, args...)
	if err != nil {
		log.Printf("[cron] append detail log failed: %v", err)
	}
}

func logCronDetailStatusByID(db *sql.DB, detailID int, action string) {
	if db == nil {
		return
	}
	selectResponseSchema := `response_schema`
	if !hasTableColumn(db, "task_detail", "response_schema") {
		selectResponseSchema = `'' AS response_schema`
	}
	query := fmt.Sprintf(`SELECT id, meta_id, exec_time, agent_id, chat_id, '' AS meta_ref, task_type, model, thinking, %s, %s, content, %s, started FROM task_detail WHERE id = ?`, cronVerifySelect(db, "task_detail"), cronRouterDisableSelect(db, "task_detail"), selectResponseSchema)
	if hasTableColumn(db, "task_detail", "meta_ref") {
		query = fmt.Sprintf(`SELECT id, meta_id, exec_time, agent_id, chat_id, meta_ref, task_type, model, thinking, %s, %s, content, %s, started FROM task_detail WHERE id = ?`, cronVerifySelect(db, "task_detail"), cronRouterDisableSelect(db, "task_detail"), selectResponseSchema)
	}
	row := db.QueryRow(query, detailID)
	var d cronDetailResult
	var th, verify, routerDisable int
	if err := row.Scan(&d.ID, &d.MetaID, &d.ExecTime, &d.AgentID, &d.ChatID, &d.MetaRef, &d.Type, &d.Model, &th, &verify, &routerDisable, &d.Content, &d.ResponseSchema, &d.Started); err != nil {
		return
	}
	d.Type = sharedutil.NormalizeTaskType(d.Type)
	d.Thinking = th != 0
	d.Verify = verify != 0
	d.RouterDisable = routerDisable != 0
	appendCronDetailLog(db, cronDetailLogEntry{
		DetailID:       d.ID,
		MetaID:         d.MetaID,
		AgentID:        d.AgentID,
		ChatID:         d.ChatID,
		TaskType:       d.Type,
		Action:         action,
		ExecTime:       d.ExecTime,
		Model:          d.Model,
		Thinking:       d.Thinking,
		Verify:         d.Verify,
		RouterDisable:  d.RouterDisable,
		Content:        d.Content,
		ResponseSchema: d.ResponseSchema,
		Started:        d.Started,
		OccurredAt:     cronLogTimestamp(),
	})
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func parseDeleteMetaIDFromRequest(r *http.Request) string {
	q := r.URL.Query()
	return firstNonEmpty(q.Get("id"), q.Get("metaId"), q.Get("meta"), q.Get("meta-id"))
}

func parseDeleteDetailIDFromRequest(r *http.Request) string {
	q := r.URL.Query()
	return firstNonEmpty(q.Get("detailId"), q.Get("detail"), q.Get("detail-id"))
}

func handleCronMetadata() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if cronDB == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "db not ready"})
			return
		}
		filter, err := parseCronMetaFilterFromRequest(r)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		metas, err := queryCronMetas(cronDB, filter)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "query error: " + err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "data": metas})
	}
}

func handleCronDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if cronDB == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "db not ready"})
			return
		}
		filter, err := parseCronMetaFilterFromRequest(r)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		explicitID := parseDeleteMetaIDFromRequest(r)
		if strings.TrimSpace(explicitID) == "" && filter.AgentID == "" && filter.ChatID == "" && filter.Model == "" && filter.Content == "" && filter.Cycle == nil && filter.StartFrom == nil && filter.StartTo == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "id or filter is required"})
			return
		}
		affected, deletedIDs, err := deleteCronMetas(cronDB, filter, explicitID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "delete error: " + err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "affected": affected, "ids": deletedIDs})
	}
}

func handleCronDetailDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if cronDB == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "db not ready"})
			return
		}
		filter, err := parseCronDetailDeleteFilterFromRequest(r)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		explicitMetaID := firstNonEmpty(r.URL.Query().Get("metaId"), r.URL.Query().Get("meta"))
		explicitDetailID := parseDeleteDetailIDFromRequest(r)
		if strings.TrimSpace(explicitMetaID) == "" && strings.TrimSpace(explicitDetailID) == "" &&
			filter.AgentID == "" && filter.ChatID == "" && filter.Model == "" && filter.Content == "" && filter.Cycle == nil && filter.ExecFrom == nil && filter.ExecTo == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "detailId, metaId or filter is required"})
			return
		}
		affected, deletedIDs, err := deleteCronDetails(cronDB, filter, explicitDetailID, explicitMetaID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "delete error: " + err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "affected": affected, "ids": deletedIDs})
	}
}

func handleCronDetailList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if cronDB == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "db not ready"})
			return
		}
		filter, err := parseCronDetailFilterFromRequest(r)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		details, err := queryCronDetails(cronDB, filter)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "query error: " + err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "data": details})
	}
}

type cronDetailUpdateRequest struct {
	DetailID      int    `json:"detailId"`
	AgentID       string `json:"agentId"`
	ExecTime      string `json:"execTime"`
	Content       string `json:"content"`
	Model         string `json:"model"`
	Thinking      bool   `json:"thinking"`
	Verify        bool   `json:"verify"`
	RouterDisable bool   `json:"router_disable"`
}

func (r *cronDetailUpdateRequest) UnmarshalJSON(data []byte) error {
	type rawCronDetailUpdateRequest struct {
		DetailID      int    `json:"detailId"`
		AgentID       string `json:"agentId"`
		ExecTime      string `json:"execTime"`
		Content       string `json:"content"`
		Model         string `json:"model"`
		Thinking      bool   `json:"thinking"`
		Verify        bool   `json:"verify"`
		RouterDisable *bool  `json:"router_disable"`
	}
	var raw rawCronDetailUpdateRequest
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.DetailID = raw.DetailID
	r.AgentID = raw.AgentID
	r.ExecTime = raw.ExecTime
	r.Content = raw.Content
	r.Model = raw.Model
	r.Thinking = raw.Thinking
	r.Verify = raw.Verify
	r.RouterDisable = true
	if raw.RouterDisable != nil {
		r.RouterDisable = *raw.RouterDisable
	}
	return nil
}

// handleCronDetailUpdate updates one pending cron detail. The started = 0
// predicate is deliberately part of the UPDATE, so a task that begins while
// its editor is open cannot be overwritten by a stale browser view.
func handleCronDetailUpdate(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentID := strings.TrimSpace(r.URL.Query().Get("agentId"))
		if agentID == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "agentId is required"})
			return
		}
		var req cronDetailUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "invalid request: " + err.Error()})
			return
		}
		if err := validateIntegrationAgentExists(cfg.AgentDir, cfg.effectiveDeviceID(), agentID); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		targetAgentID := strings.TrimSpace(req.AgentID)
		if targetAgentID == "" {
			targetAgentID = agentID
		}
		if err := validateIntegrationAgentExists(cfg.AgentDir, cfg.effectiveDeviceID(), targetAgentID); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		req.AgentID = targetAgentID
		if err := validateIntegrationRegisteredModel(req.Model); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		detail, err := updateCronDetail(agentID, req)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "data": detail})
	}
}

type cronMetaUpdateRequest struct {
	MetaID        int    `json:"metaId"`
	AgentID       string `json:"agentId"`
	RawTime       string `json:"rawTime"`
	Cycle         int    `json:"cycle"`
	Content       string `json:"content"`
	Model         string `json:"model"`
	Thinking      bool   `json:"thinking"`
	Verify        bool   `json:"verify"`
	RouterDisable bool   `json:"router_disable"`
}

func (r *cronMetaUpdateRequest) UnmarshalJSON(data []byte) error {
	type rawCronMetaUpdateRequest struct {
		MetaID        int    `json:"metaId"`
		AgentID       string `json:"agentId"`
		RawTime       string `json:"rawTime"`
		Cycle         int    `json:"cycle"`
		Content       string `json:"content"`
		Model         string `json:"model"`
		Thinking      bool   `json:"thinking"`
		Verify        bool   `json:"verify"`
		RouterDisable *bool  `json:"router_disable"`
	}
	var raw rawCronMetaUpdateRequest
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.MetaID = raw.MetaID
	r.AgentID = raw.AgentID
	r.RawTime = raw.RawTime
	r.Cycle = raw.Cycle
	r.Content = raw.Content
	r.Model = raw.Model
	r.Thinking = raw.Thinking
	r.Verify = raw.Verify
	r.RouterDisable = true
	if raw.RouterDisable != nil {
		r.RouterDisable = *raw.RouterDisable
	}
	return nil
}

func handleCronMetaUpdate(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentID := strings.TrimSpace(r.URL.Query().Get("agentId"))
		if agentID == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "agentId is required"})
			return
		}
		var req cronMetaUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "invalid request: " + err.Error()})
			return
		}
		if err := validateIntegrationAgentExists(cfg.AgentDir, cfg.effectiveDeviceID(), agentID); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		targetAgentID := strings.TrimSpace(req.AgentID)
		if targetAgentID == "" {
			targetAgentID = agentID
		}
		if err := validateIntegrationAgentExists(cfg.AgentDir, cfg.effectiveDeviceID(), targetAgentID); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		req.AgentID = targetAgentID
		detail, err := updateCronMeta(agentID, req)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "data": detail})
	}
}

func loadCronMetaForUpdate(tx *sql.Tx, metaID int, agentID string) (cronMetaResult, error) {
	var result cronMetaResult
	var thinking, verify, routerDisable int
	err := tx.QueryRow(`SELECT id, cycle, raw_time, agent_id, model, thinking, verify, router_disable, cron, content, response_schema, chat_id, task_type
		FROM task_meta WHERE id = ? AND agent_id = ?`, metaID, agentID).Scan(
		&result.ID, &result.Cycle, &result.RawTime, &result.AgentID, &result.Model, &thinking, &verify, &routerDisable,
		&result.Cron, &result.Content, &result.ResponseSchema, &result.ChatID, &result.Type,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return cronMetaResult{}, fmt.Errorf("task metadata not found")
	}
	if err != nil {
		return cronMetaResult{}, err
	}
	result.Thinking = thinking != 0
	result.Verify = verify != 0
	result.RouterDisable = routerDisable != 0
	result.Type = sharedutil.NormalizeTaskType(result.Type)
	return result, nil
}

func cronDetailTimesForMeta(cycle int, rawTime string) ([]time.Time, error) {
	if cycle == -1 {
		return nil, nil
	}
	start, err := time.ParseInLocation("2006-01-02 15:04", rawTime, time.Local)
	if err != nil {
		return nil, fmt.Errorf("invalid rawTime: %w", err)
	}
	switch cycle {
	case 0:
		return []time.Time{start}, nil
	case 1, 2:
		times := []time.Time{start}
		end := start.Add(5 * 24 * time.Hour)
		for day := start.AddDate(0, 0, 1); !day.After(end); day = day.AddDate(0, 0, 1) {
			if cycle == 2 || (day.Weekday() >= time.Monday && day.Weekday() <= time.Friday) {
				times = append(times, day)
			}
		}
		return times, nil
	case 3, 4, 5:
		interval := cronCycleInterval(cycle)
		times := make([]time.Time, 0, 5*24+1)
		end := start.Add(5 * 24 * time.Hour)
		for tick := start; !tick.After(end); tick = tick.Add(interval) {
			times = append(times, tick)
		}
		return times, nil
	default:
		return nil, fmt.Errorf("cycle must be one of -1,0,1,2,3,4,5")
	}
}

func loadPendingCronDetailLogs(tx *sql.Tx, metaID int) ([]cronDetailLogEntry, error) {
	rows, err := tx.Query(`SELECT id, meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, verify, router_disable, content, response_schema, started
		FROM task_detail WHERE meta_id = ? AND started = 0 ORDER BY id`, metaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := make([]cronDetailLogEntry, 0)
	for rows.Next() {
		var entry cronDetailLogEntry
		var thinking, verify, routerDisable int
		if err := rows.Scan(&entry.DetailID, &entry.MetaID, &entry.ExecTime, &entry.AgentID, &entry.ChatID, &entry.TaskType, &entry.Model, &thinking, &verify, &routerDisable, &entry.Content, &entry.ResponseSchema, &entry.Started); err != nil {
			return nil, err
		}
		entry.Thinking = thinking != 0
		entry.Verify = verify != 0
		entry.RouterDisable = routerDisable != 0
		entry.TaskType = sharedutil.NormalizeTaskType(entry.TaskType)
		entry.Action = "delete"
		entry.OccurredAt = cronLogTimestamp()
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func rebuildPendingCronDetails(tx *sql.Tx, meta cronMetaResult) ([]cronDetailLogEntry, error) {
	times, err := cronDetailTimesForMeta(meta.Cycle, meta.RawTime)
	if err != nil {
		return nil, err
	}
	entries := make([]cronDetailLogEntry, 0, len(times))
	for _, execAt := range times {
		res, err := tx.Exec(`INSERT OR IGNORE INTO task_detail (meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, verify, router_disable, content, response_schema, started)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,0)`,
			meta.ID, execAt.Unix(), meta.AgentID, meta.ChatID, meta.Type, meta.Model, boolToInt(meta.Thinking), boolToInt(meta.Verify), boolToInt(meta.RouterDisable), meta.Content, meta.ResponseSchema)
		if err != nil {
			return nil, err
		}
		if affected, _ := res.RowsAffected(); affected != 1 {
			continue
		}
		detailID, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		entries = append(entries, cronDetailLogEntry{
			DetailID:       int(detailID),
			MetaID:         meta.ID,
			AgentID:        meta.AgentID,
			ChatID:         meta.ChatID,
			TaskType:       meta.Type,
			Action:         "insert",
			ExecTime:       execAt.Unix(),
			Model:          meta.Model,
			Thinking:       meta.Thinking,
			Verify:         meta.Verify,
			RouterDisable:  meta.RouterDisable,
			Content:        meta.Content,
			ResponseSchema: meta.ResponseSchema,
			Started:        0,
			OccurredAt:     cronLogTimestamp(),
		})
	}
	return entries, nil
}

func updateCronMeta(agentID string, req cronMetaUpdateRequest) (cronMetaResult, error) {
	var empty cronMetaResult
	if req.MetaID <= 0 {
		return empty, fmt.Errorf("metaId is required")
	}
	agentID = strings.TrimSpace(agentID)
	req.AgentID = strings.TrimSpace(req.AgentID)
	if req.AgentID == "" {
		req.AgentID = agentID
	}
	req.Content = strings.TrimSpace(req.Content)
	req.Model = strings.TrimSpace(req.Model)
	req.RawTime = strings.TrimSpace(req.RawTime)
	if req.Content == "" {
		return empty, fmt.Errorf("content is required")
	}
	if utf8.RuneCountInString(req.Content) > 5000 {
		return empty, fmt.Errorf("content exceeds 5000 characters")
	}
	if req.Model == "" {
		return empty, fmt.Errorf("model is required")
	}
	if req.Cycle < -1 || req.Cycle > 5 || req.Cycle == 0 {
		return empty, fmt.Errorf("metadata cycle must be one of -1,1,2,3,4,5")
	}
	if !req.Thinking {
		req.Verify = false
	}
	if err := validateIntegrationRegisteredModel(req.Model); err != nil {
		return empty, err
	}
	if cronDB == nil {
		return empty, fmt.Errorf("db not ready")
	}
	ensureCronSchema(cronDB)
	tx, err := cronDB.Begin()
	if err != nil {
		return empty, err
	}
	defer tx.Rollback()
	current, err := loadCronMetaForUpdate(tx, req.MetaID, agentID)
	if err != nil {
		return empty, err
	}
	rawTime := req.RawTime
	cronExpr := current.Cron
	if req.Cycle == -1 {
		if strings.TrimSpace(cronExpr) == "" {
			return empty, fmt.Errorf("custom cron expression is missing")
		}
		rawTime = current.RawTime
	} else {
		execAt, err := time.ParseInLocation("2006-01-02 15:04", rawTime, time.Local)
		if err != nil {
			return empty, fmt.Errorf("invalid rawTime: %w", err)
		}
		if !execAt.After(time.Now()) {
			return empty, fmt.Errorf("执行时间需要晚于当前时间")
		}
		cronExpr = cronBuildExpr(req.Cycle, execAt)
	}
	deletedEntries, err := loadPendingCronDetailLogs(tx, current.ID)
	if err != nil {
		return empty, fmt.Errorf("load pending details: %w", err)
	}
	if _, err := tx.Exec(`UPDATE task_meta SET agent_id = ?, cycle = ?, raw_time = ?, model = ?, thinking = ?, verify = ?, router_disable = ?, cron = ?, content = ?
		WHERE id = ? AND agent_id = ?`,
		req.AgentID, req.Cycle, rawTime, req.Model, boolToInt(req.Thinking), boolToInt(req.Verify), boolToInt(req.RouterDisable), cronExpr, req.Content, current.ID, agentID); err != nil {
		return empty, fmt.Errorf("update metadata: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM task_detail WHERE meta_id = ? AND started = 0`, current.ID); err != nil {
		return empty, fmt.Errorf("delete pending details: %w", err)
	}
	updated := current
	updated.AgentID = req.AgentID
	updated.Cycle = req.Cycle
	updated.RawTime = rawTime
	updated.Model = req.Model
	updated.Thinking = req.Thinking
	updated.Verify = req.Verify
	updated.RouterDisable = req.RouterDisable
	updated.Cron = cronExpr
	updated.Content = req.Content
	createdEntries, err := rebuildPendingCronDetails(tx, updated)
	if err != nil {
		return empty, fmt.Errorf("rebuild pending details: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return empty, err
	}
	appendCronMetaLog(cronDB, cronMetaLogEntry{
		MetaID:         updated.ID,
		AgentID:        updated.AgentID,
		ChatID:         updated.ChatID,
		TaskType:       updated.Type,
		Action:         "update",
		Cycle:          updated.Cycle,
		RawTime:        updated.RawTime,
		Model:          updated.Model,
		Thinking:       updated.Thinking,
		Verify:         updated.Verify,
		RouterDisable:  updated.RouterDisable,
		Cron:           updated.Cron,
		Content:        updated.Content,
		ResponseSchema: updated.ResponseSchema,
		CreatedAt:      cronLogTimestamp(),
	})
	for _, entry := range deletedEntries {
		appendCronDetailLog(cronDB, entry)
	}
	for _, entry := range createdEntries {
		appendCronDetailLog(cronDB, entry)
	}
	return updated, nil
}

func updateCronDetail(agentID string, req cronDetailUpdateRequest) (cronDetailResult, error) {
	var empty cronDetailResult
	if req.DetailID <= 0 {
		return empty, fmt.Errorf("detailId is required")
	}
	agentID = strings.TrimSpace(agentID)
	req.AgentID = strings.TrimSpace(req.AgentID)
	if req.AgentID == "" {
		req.AgentID = agentID
	}
	req.Content = strings.TrimSpace(req.Content)
	req.Model = strings.TrimSpace(req.Model)
	req.ExecTime = strings.TrimSpace(req.ExecTime)
	if req.Content == "" {
		return empty, fmt.Errorf("content is required")
	}
	if req.Model == "" {
		return empty, fmt.Errorf("model is required")
	}
	if utf8.RuneCountInString(req.Content) > 5000 {
		return empty, fmt.Errorf("content exceeds 5000 characters")
	}
	execAt, err := time.ParseInLocation("2006-01-02 15:04", req.ExecTime, time.Local)
	if err != nil {
		return empty, fmt.Errorf("invalid execTime: %w", err)
	}
	if !execAt.After(time.Now()) {
		return empty, fmt.Errorf("执行时间需要晚于当前时间")
	}
	if !req.Thinking {
		req.Verify = false
	}
	if err := validateIntegrationRegisteredModel(req.Model); err != nil {
		return empty, err
	}
	if cronDB == nil {
		return empty, fmt.Errorf("db not ready")
	}
	ensureCronSchema(cronDB)
	res, err := cronDB.Exec(`UPDATE task_detail
		SET agent_id = ?, exec_time = ?, content = ?, model = ?, thinking = ?, verify = ?, router_disable = ?
		WHERE id = ? AND agent_id = ? AND started = 0`,
		req.AgentID, execAt.Unix(), req.Content, req.Model, boolToInt(req.Thinking), boolToInt(req.Verify), boolToInt(req.RouterDisable), req.DetailID, agentID)
	if err != nil {
		return empty, fmt.Errorf("update error: %w", err)
	}
	if affected, _ := res.RowsAffected(); affected != 1 {
		var started int
		err := cronDB.QueryRow(`SELECT started FROM task_detail WHERE id = ? AND agent_id = ?`, req.DetailID, agentID).Scan(&started)
		if errors.Is(err, sql.ErrNoRows) {
			return empty, fmt.Errorf("task detail not found")
		}
		if err != nil {
			return empty, fmt.Errorf("query detail state: %w", err)
		}
		return empty, fmt.Errorf("保存失败：任务已启动，无法修改")
	}
	detailID := req.DetailID
	details, err := queryCronDetails(cronDB, cronDetailFilter{DetailID: &detailID, AgentID: req.AgentID})
	if err != nil {
		return empty, fmt.Errorf("load updated detail: %w", err)
	}
	if len(details) != 1 {
		return empty, fmt.Errorf("updated detail not found")
	}
	detail := details[0]
	appendCronDetailLog(cronDB, cronDetailLogEntry{
		DetailID:       detail.ID,
		MetaID:         detail.MetaID,
		AgentID:        detail.AgentID,
		ChatID:         detail.ChatID,
		TaskType:       detail.Type,
		Action:         "update",
		ExecTime:       detail.ExecTime,
		Model:          detail.Model,
		Thinking:       detail.Thinking,
		Verify:         detail.Verify,
		RouterDisable:  detail.RouterDisable,
		Content:        detail.Content,
		ResponseSchema: detail.ResponseSchema,
		Started:        detail.Started,
		OccurredAt:     cronLogTimestamp(),
	})
	return detail, nil
}

func handleCronDetailStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentID := r.URL.Query().Get("agentId")
		detailID := r.URL.Query().Get("detailId")
		status := r.URL.Query().Get("status")
		if agentID == "" || detailID == "" || status == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "agentId, detailId and status are required"})
			return
		}
		if cronDB == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "db not ready"})
			return
		}
		res, err := cronDB.Exec(`UPDATE task_detail SET started = ? WHERE id = ? AND agent_id = ?`, status, detailID, agentID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "update error: " + err.Error()})
			return
		}
		n, _ := res.RowsAffected()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "affected": n})
	}
}

type restoreRecord struct {
	ID           int    `json:"id"`
	AgentID      string `json:"agentId"`
	ChatID       string `json:"chatId"`
	ChatType     string `json:"chatType"`
	Role         string `json:"role"`
	ResponseType string `json:"responseType"`
	Content      string `json:"content"`
	LogType      int    `json:"logType,omitempty"`
	CreatedAt    string `json:"createdAt"`
}

type restoreHistoryMeta struct {
	HasMore        bool   `json:"hasMore"`
	BeforeTimeline string `json:"beforeTimeline,omitempty"`
	BeforeID       int    `json:"beforeId,omitempty"`
}

func normalizeRestoreRecord(rec *restoreRecord) {
	if rec == nil {
		return
	}
	rec.ChatType = normalizeChatType(rec.ChatType)
	rec.ResponseType = normalizeResponseType(rec.ResponseType)
}

func restoreRecordBeforeCursor(rec restoreRecord, beforeTimeline string, beforeID int) bool {
	if strings.TrimSpace(beforeTimeline) == "" {
		return true
	}
	if rec.CreatedAt < beforeTimeline {
		return true
	}
	if rec.CreatedAt > beforeTimeline {
		return false
	}
	if beforeID > 0 {
		return rec.ID < beforeID
	}
	return false
}

func restoreRecordAfterCursor(rec restoreRecord, timeline string, lastID int) bool {
	if strings.TrimSpace(timeline) == "" {
		return true
	}
	if rec.CreatedAt > timeline {
		return true
	}
	if rec.CreatedAt < timeline {
		return false
	}
	if lastID > 0 {
		return rec.ID > lastID
	}
	return false
}

func queryRestoreForwardRecords(agentID, chatID, timeline string, lastID int) ([]restoreRecord, error) {
	if cronDB == nil {
		return nil, fmt.Errorf("db not ready")
	}
	query := `SELECT id, agent_id, chat_id, chat_type, role, response_type, content, created_at FROM chat_log WHERE chat_id = ? AND created_at > ? ORDER BY id`
	args := []interface{}{chatID, timeline}
	if strings.TrimSpace(agentID) != "" {
		query = `SELECT id, agent_id, chat_id, chat_type, role, response_type, content, created_at FROM chat_log WHERE agent_id = ? AND chat_id = ? AND created_at > ? ORDER BY id`
		args = []interface{}{agentID, chatID, timeline}
	}
	if lastID > 0 {
		query = `SELECT id, agent_id, chat_id, chat_type, role, response_type, content, created_at FROM chat_log WHERE chat_id = ? AND (created_at > ? OR (created_at = ? AND id > ?)) ORDER BY id`
		args = []interface{}{chatID, timeline, timeline, lastID}
		if strings.TrimSpace(agentID) != "" {
			query = `SELECT id, agent_id, chat_id, chat_type, role, response_type, content, created_at FROM chat_log WHERE agent_id = ? AND chat_id = ? AND (created_at > ? OR (created_at = ? AND id > ?)) ORDER BY id`
			args = []interface{}{agentID, chatID, timeline, timeline, lastID}
		}
	}
	rows, err := cronDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]restoreRecord, 0)
	for rows.Next() {
		var rec restoreRecord
		if err := rows.Scan(&rec.ID, &rec.AgentID, &rec.ChatID, &rec.ChatType, &rec.Role, &rec.ResponseType, &rec.Content, &rec.CreatedAt); err != nil {
			return nil, err
		}
		normalizeRestoreRecord(&rec)
		records = append(records, rec)
	}
	return records, rows.Err()
}

func appendRestoreCLIRecords(records []restoreRecord, entries []eventLogEntry) []restoreRecord {
	if len(entries) == 0 {
		return records
	}
	for _, item := range entries {
		rec := restoreRecord{
			ID:           item.ID,
			AgentID:      item.AgentID,
			ChatID:       item.ChatID,
			ChatType:     chatTypePageSession,
			ResponseType: "normal",
			Content:      item.Content,
			LogType:      item.LogType,
			CreatedAt:    item.CreatedAt,
		}
		if item.LogType == logTypeCLIGet {
			rec.Role = "cli/get"
		} else {
			rec.Role = "cli/pub"
		}
		records = append(records, rec)
	}
	return records
}

func sortRestoreRecords(records []restoreRecord) {
	sort.Slice(records, func(i, j int) bool {
		if records[i].CreatedAt == records[j].CreatedAt {
			return records[i].ID < records[j].ID
		}
		return records[i].CreatedAt < records[j].CreatedAt
	})
}

func queryRestoreHistoryRecords(agentID, chatID, beforeTimeline string, beforeID, limit int) ([]restoreRecord, *restoreHistoryMeta, error) {
	if cronDB == nil {
		return nil, nil, fmt.Errorf("db not ready")
	}
	if limit <= 0 {
		limit = 120
	}
	trimmedAgentID := strings.TrimSpace(agentID)
	trimmedChatID := strings.TrimSpace(chatID)
	trimmedBeforeTimeline := strings.TrimSpace(beforeTimeline)

	query := `SELECT id, agent_id, chat_id, chat_type, role, response_type, content, created_at FROM chat_log WHERE chat_id = ?`
	args := []interface{}{trimmedChatID}
	if trimmedAgentID != "" {
		query = `SELECT id, agent_id, chat_id, chat_type, role, response_type, content, created_at FROM chat_log WHERE agent_id = ? AND chat_id = ?`
		args = []interface{}{trimmedAgentID, trimmedChatID}
	}
	if trimmedBeforeTimeline != "" {
		if beforeID > 0 {
			query += ` AND (created_at < ? OR (created_at = ? AND id < ?))`
			args = append(args, trimmedBeforeTimeline, trimmedBeforeTimeline, beforeID)
		} else {
			query += ` AND created_at < ?`
			args = append(args, trimmedBeforeTimeline)
		}
	}
	query += ` ORDER BY created_at, id`

	rows, err := cronDB.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	allRecords := make([]restoreRecord, 0)
	for rows.Next() {
		var rec restoreRecord
		if err := rows.Scan(&rec.ID, &rec.AgentID, &rec.ChatID, &rec.ChatType, &rec.Role, &rec.ResponseType, &rec.Content, &rec.CreatedAt); err != nil {
			return nil, nil, err
		}
		normalizeRestoreRecord(&rec)
		allRecords = append(allRecords, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	entries, err := queryEventLogs(trimmedAgentID, trimmedChatID, logTypeCLIGet, logTypeCLIPub)
	if err != nil {
		return nil, nil, err
	}
	filtered := make([]eventLogEntry, 0, len(entries))
	for _, item := range entries {
		rec := restoreRecord{ID: item.ID, CreatedAt: item.CreatedAt}
		if !restoreRecordBeforeCursor(rec, trimmedBeforeTimeline, beforeID) {
			continue
		}
		filtered = append(filtered, item)
	}
	allRecords = appendRestoreCLIRecords(allRecords, filtered)

	sortRestoreRecords(allRecords)
	if len(allRecords) == 0 {
		return []restoreRecord{}, &restoreHistoryMeta{HasMore: false}, nil
	}

	start := 0
	if len(allRecords) > limit {
		start = len(allRecords) - limit
	}
	for start > 0 && allRecords[start].Role != "Q" {
		start--
	}
	records := append([]restoreRecord(nil), allRecords[start:]...)
	meta := &restoreHistoryMeta{HasMore: start > 0}
	if meta.HasMore && len(records) > 0 {
		meta.BeforeTimeline = records[0].CreatedAt
		meta.BeforeID = records[0].ID
	}
	return records, meta, nil
}

func handleRestore() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentID := r.URL.Query().Get("agentId")
		chatID := r.URL.Query().Get("chat")
		timeline := r.URL.Query().Get("timeline")
		lastIDStr := r.URL.Query().Get("lastId")
		historyMode := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("history")), "1") || strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("history")), "true")
		if chatID == "" || (!historyMode && timeline == "") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "chat and timeline are required"})
			return
		}
		lastID := 0
		if lastIDStr != "" {
			if parsed, err := strconv.Atoi(lastIDStr); err == nil && parsed > 0 {
				lastID = parsed
			}
		}
		if cronDB == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "db not ready"})
			return
		}

		if historyMode {
			beforeTimeline := r.URL.Query().Get("beforeTimeline")
			beforeID := 0
			if raw := strings.TrimSpace(r.URL.Query().Get("beforeId")); raw != "" {
				if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
					beforeID = parsed
				}
			}
			limit := 120
			if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
				if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
					limit = parsed
				}
			}
			records, history, err := queryRestoreHistoryRecords(agentID, chatID, beforeTimeline, beforeID, limit)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "query error: " + err.Error()})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "data": records, "history": history})
			return
		}

		records, err := queryRestoreForwardRecords(agentID, chatID, timeline, lastID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "query error: " + err.Error()})
			return
		}
		entries, err := queryEventLogs(agentID, chatID, logTypeCLIGet, logTypeCLIPub)
		if err == nil {
			filtered := make([]eventLogEntry, 0, len(entries))
			for _, item := range entries {
				rec := restoreRecord{ID: item.ID, CreatedAt: item.CreatedAt}
				if !restoreRecordAfterCursor(rec, timeline, lastID) {
					continue
				}
				filtered = append(filtered, item)
			}
			records = appendRestoreCLIRecords(records, filtered)
		}
		sortRestoreRecords(records)
		if records == nil {
			records = []restoreRecord{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "data": records})
	}
}

type roundLogRecord struct {
	ID        int
	AgentID   string
	ChatID    string
	LogType   int
	TypeLabel string
	Content   string
	CreatedAt string
}

type roundLogOptions struct {
	Round int
	Start string
	Close string
}

const roundLogStorageTimeLayout = "2006-01-02T15:04:05.000"

func getWorkspaceByAgentID(cfg *Config, agentID string) (string, error) {
	metadata, err := getAgentOutput(cfg.AgentDir, cfg.effectiveDeviceID(), time.Duration(cfg.AgentCacheMs)*time.Millisecond)
	if err != nil {
		return "", err
	}
	for _, agent := range metadata.Agents {
		if agent.AgentID == agentID {
			return agent.Workspace, nil
		}
	}
	return "", fmt.Errorf("agent not found: %s", agentID)
}

func formatRoundLogTime(value string) string {
	value = strings.TrimSpace(value)
	layouts := []string{
		"2006-01-02T15:04:05.000",
		"2006-01-02T15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Format("2006-01-02 15:04:05")
		}
	}
	return value
}

func markdownEscapeCell(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", "<br>")
	return value
}

func parseRoundLogTimeValue(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	layouts := []struct {
		layout      string
		withLocalTZ bool
	}{
		{layout: "2006-01-02 15:04:05", withLocalTZ: true},
		{layout: roundLogStorageTimeLayout, withLocalTZ: true},
		{layout: "2006-01-02T15:04:05", withLocalTZ: true},
		{layout: time.RFC3339Nano},
		{layout: time.RFC3339},
	}
	for _, layout := range layouts {
		var (
			parsed time.Time
			err    error
		)
		if layout.withLocalTZ {
			parsed, err = time.ParseInLocation(layout.layout, value, time.Local)
		} else {
			parsed, err = time.Parse(layout.layout, value)
		}
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time format: %s", value)
}

func normalizeRoundLogBoundary(value string, isClose bool) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	parsed, err := parseRoundLogTimeValue(value)
	if err != nil {
		return "", err
	}
	if isClose {
		switch {
		case len(value) == len("2006-01-02 15:04:05"):
			parsed = parsed.Add(999 * time.Millisecond)
		case len(value) == len("2006-01-02T15:04:05"):
			parsed = parsed.Add(999 * time.Millisecond)
		}
	}
	return parsed.Format(roundLogStorageTimeLayout), nil
}

func normalizeRoundLogOptions(opts roundLogOptions) (roundLogOptions, error) {
	if opts.Round < 0 {
		return opts, fmt.Errorf("round must be a positive integer")
	}
	startRaw := strings.TrimSpace(opts.Start)
	closeRaw := strings.TrimSpace(opts.Close)
	if opts.Round == 0 && startRaw == "" {
		return opts, fmt.Errorf("round or start is required")
	}
	if startRaw != "" {
		startValue, err := normalizeRoundLogBoundary(startRaw, false)
		if err != nil {
			return opts, fmt.Errorf("start %w", err)
		}
		opts.Start = startValue
	} else {
		opts.Start = ""
	}
	if closeRaw != "" {
		closeValue, err := normalizeRoundLogBoundary(closeRaw, true)
		if err != nil {
			return opts, fmt.Errorf("close %w", err)
		}
		opts.Close = closeValue
	} else {
		opts.Close = ""
	}
	if opts.Start != "" && opts.Close != "" && opts.Close < opts.Start {
		return opts, fmt.Errorf("close must be greater than or equal to start")
	}
	return opts, nil
}

func filterRoundLogEntries(entries []eventLogEntry, opts roundLogOptions) []eventLogEntry {
	if len(entries) == 0 {
		return []eventLogEntry{}
	}
	startIdx := 0
	if opts.Round > 0 {
		requestIndexes := make([]int, 0)
		for idx, item := range entries {
			if item.LogType == logTypeChatCompletionRequest {
				requestIndexes = append(requestIndexes, idx)
			}
		}
		if len(requestIndexes) == 0 {
			return []eventLogEntry{}
		}
		startReq := len(requestIndexes) - opts.Round
		if startReq < 0 {
			startReq = 0
		}
		startIdx = requestIndexes[startReq]
	}

	filtered := make([]eventLogEntry, 0, len(entries)-startIdx)
	for _, item := range entries[startIdx:] {
		createdAt := strings.TrimSpace(item.CreatedAt)
		if opts.Start != "" && createdAt < opts.Start {
			continue
		}
		if opts.Close != "" && createdAt > opts.Close {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

const defaultChatSessionLogLimit = 500

type chatSessionLogRecord struct {
	ID           int    `json:"id"`
	AgentID      string `json:"agent_id"`
	ChatID       string `json:"chat_id"`
	LogType      int    `json:"log_type"`
	LogTypeLabel string `json:"log_type_label"`
	CreatedAt    string `json:"created_at"`
	Content      string `json:"content"`
}

func normalizeChatSessionLogLimit(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return defaultChatSessionLogLimit, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit <= 0 {
		return 0, fmt.Errorf("limit must be a positive integer")
	}
	if limit > defaultChatSessionLogLimit {
		limit = defaultChatSessionLogLimit
	}
	return limit, nil
}

func normalizeChatSessionLogRange(startRaw, closeRaw string) (string, string, error) {
	startValue := ""
	closeValue := ""
	var err error
	if strings.TrimSpace(startRaw) != "" {
		startValue, err = normalizeRoundLogBoundary(startRaw, false)
		if err != nil {
			return "", "", fmt.Errorf("start %w", err)
		}
	}
	if strings.TrimSpace(closeRaw) != "" {
		closeValue, err = normalizeRoundLogBoundary(closeRaw, true)
		if err != nil {
			return "", "", fmt.Errorf("close %w", err)
		}
	}
	if startValue != "" && closeValue != "" && closeValue < startValue {
		return "", "", fmt.Errorf("close must be greater than or equal to start")
	}
	return startValue, closeValue, nil
}

func buildChatSessionLogTypeLabel(logType int) string {
	switch logType {
	case logTypeChatCompletionRequest:
		return "SSE请求"
	case logTypeChatCompletionResponse:
		return "SSE响应"
	default:
		return buildRoundTypeLabel(logType)
	}
}

func filterChatSessionLogEntries(entries []eventLogEntry, startValue, closeValue string) []eventLogEntry {
	if len(entries) == 0 {
		return []eventLogEntry{}
	}
	filtered := make([]eventLogEntry, 0, len(entries))
	for _, item := range entries {
		if item.LogType != logTypeChatCompletionRequest && item.LogType != logTypeChatCompletionResponse {
			continue
		}
		createdAt := strings.TrimSpace(item.CreatedAt)
		if startValue != "" && createdAt < startValue {
			continue
		}
		if closeValue != "" && createdAt > closeValue {
			continue
		}
		filtered = append(filtered, item)
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].CreatedAt == filtered[j].CreatedAt {
			return filtered[i].ID > filtered[j].ID
		}
		return filtered[i].CreatedAt > filtered[j].CreatedAt
	})
	return filtered
}

func buildChatSessionLogRecords(entries []eventLogEntry, limit int) []chatSessionLogRecord {
	if limit <= 0 {
		limit = defaultChatSessionLogLimit
	}
	if len(entries) > limit {
		entries = entries[:limit]
	}
	records := make([]chatSessionLogRecord, 0, len(entries))
	for _, item := range entries {
		records = append(records, chatSessionLogRecord{
			ID:           item.ID,
			AgentID:      item.AgentID,
			ChatID:       item.ChatID,
			LogType:      item.LogType,
			LogTypeLabel: buildChatSessionLogTypeLabel(item.LogType),
			CreatedAt:    item.CreatedAt,
			Content:      item.Content,
		})
	}
	return records
}

func roundLogTimestamp() string {
	return time.Now().Format("20060102150405")
}

func fileSizeK(path string) (float64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return float64(info.Size()) / 1024.0, nil
}

func buildRoundTypeLabel(logType int) string {
	switch logType {
	case logTypeChatCompletionRequest:
		return "请求"
	case logTypeChatCompletionResponse:
		return "响应"
	case logTypeCLIGet:
		return "工具请求（cli/get）"
	case logTypeCLIPub:
		return "工具响应（cli/pub）"
	default:
		return "未知"
	}
}

func collectRoundLogTextParts(value any) []string {
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{typed}
	case []any:
		out := make([]string, 0)
		for _, item := range typed {
			out = append(out, collectRoundLogTextParts(item)...)
		}
		return out
	case map[string]any:
		if text, ok := typed["text"]; ok {
			return collectRoundLogTextParts(text)
		}
		if text, ok := typed["content"]; ok {
			return collectRoundLogTextParts(text)
		}
	}
	return nil
}

func parseRoundLogJSON(content string) (map[string]any, bool) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &payload); err != nil {
		return nil, false
	}
	return payload, true
}

func extractRoundLogJSONObject(content string) string {
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end < start {
		return ""
	}
	return strings.TrimSpace(content[start : end+1])
}

func extractRoundLogMessageContents(content string) []string {
	payload, ok := parseRoundLogJSON(content)
	if !ok {
		return nil
	}
	if rawMessages, ok := payload["messages"].([]any); ok {
		out := make([]string, 0)
		for _, item := range rawMessages {
			message, ok := item.(map[string]any)
			if !ok {
				continue
			}
			out = append(out, collectRoundLogTextParts(message["content"])...)
		}
		return out
	}
	if message, ok := payload["message"]; ok {
		return collectRoundLogTextParts(message)
	}
	return nil
}

func extractRoundLogResponseContent(content string) string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var builder strings.Builder
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		payload, ok := parseRoundLogJSON(extractRoundLogJSONObject(data))
		if !ok {
			continue
		}
		rawChoices, ok := payload["choices"].([]any)
		if !ok {
			continue
		}
		for _, item := range rawChoices {
			choice, ok := item.(map[string]any)
			if !ok {
				continue
			}
			delta, ok := choice["delta"].(map[string]any)
			if !ok {
				continue
			}
			for _, part := range collectRoundLogTextParts(delta["content"]) {
				builder.WriteString(part)
			}
		}
	}
	return builder.String()
}

func extractRoundLogNestedCmdValue(value any) string {
	switch typed := value.(type) {
	case string:
		return extractRoundLogCLIGetContent(typed)
	case map[string]any:
		if cmd, ok := typed["cmd"].(string); ok && strings.TrimSpace(cmd) != "" {
			return strings.TrimSpace(cmd)
		}
		if content, ok := typed["content"]; ok {
			if nested := extractRoundLogNestedCmdValue(content); nested != "" {
				return nested
			}
		}
		if rawChoices, ok := typed["choices"].([]any); ok {
			for _, item := range rawChoices {
				if nested := extractRoundLogNestedCmdValue(item); nested != "" {
					return nested
				}
			}
		}
		if message, ok := typed["message"]; ok {
			if nested := extractRoundLogNestedCmdValue(message); nested != "" {
				return nested
			}
		}
	case []any:
		for _, item := range typed {
			if nested := extractRoundLogNestedCmdValue(item); nested != "" {
				return nested
			}
		}
	}
	return ""
}

func extractRoundLogCLIGetContent(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}
	if payload, ok := parseRoundLogJSON(trimmed); ok {
		if cmd, ok := payload["cmd"].(string); ok && strings.TrimSpace(cmd) != "" {
			return strings.TrimSpace(cmd)
		}
		if nested := extractRoundLogNestedCmdValue(payload); nested != "" {
			return nested
		}
		for _, item := range extractRoundLogMessageContents(trimmed) {
			if nested := extractRoundLogCLIGetContent(item); nested != "" {
				return nested
			}
		}
		return ""
	}
	return trimmed
}

func extractRoundLogCLIPubContent(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}
	if items := extractRoundLogMessageContents(trimmed); len(items) > 0 {
		return strings.Join(items, "\n")
	}
	if payload, ok := parseRoundLogJSON(trimmed); ok {
		if contentValue, ok := payload["content"]; ok {
			parts := collectRoundLogTextParts(contentValue)
			if len(parts) > 0 {
				return strings.Join(parts, "\n")
			}
		}
	}
	return trimmed
}

func extractRoundLogRequestContent(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}
	if items := extractRoundLogMessageContents(trimmed); len(items) > 0 {
		return strings.Join(items, "\n")
	}
	return trimmed
}

func formatRoundLogContent(logType int, content string) string {
	switch logType {
	case logTypeChatCompletionRequest:
		return extractRoundLogRequestContent(content)
	case logTypeChatCompletionResponse:
		return extractRoundLogResponseContent(content)
	case logTypeCLIGet:
		return extractRoundLogCLIGetContent(content)
	case logTypeCLIPub:
		return extractRoundLogCLIPubContent(content)
	default:
		return strings.TrimSpace(content)
	}
}

func mergeSSEContent(records []roundLogRecord) []roundLogRecord {
	out := make([]roundLogRecord, 0, len(records))
	for _, item := range records {
		if item.LogType != logTypeChatCompletionResponse {
			out = append(out, item)
			continue
		}
		last := len(out) - 1
		if last >= 0 && out[last].LogType == logTypeChatCompletionResponse {
			out[last].Content += item.Content
			continue
		}
		out = append(out, item)
	}
	return out
}

func buildRoundMarkdown(records []roundLogRecord) string {
	var builder strings.Builder
	builder.WriteString("| 时间 | 类型 | 内容 |\n")
	builder.WriteString("| --- | --- | --- |\n")
	for _, item := range records {
		builder.WriteString("| ")
		builder.WriteString(markdownEscapeCell(formatRoundLogTime(item.CreatedAt)))
		builder.WriteString(" | ")
		builder.WriteString(markdownEscapeCell(item.TypeLabel))
		builder.WriteString(" | ")
		builder.WriteString(markdownEscapeCell(item.Content))
		builder.WriteString(" |\n")
	}
	return builder.String()
}

func sanitizeLogRoundFilePart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "empty"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "?", "_", "&", "_", "=", "_", " ", "_")
	return replacer.Replace(value)
}

func resolveRoundLogAgentID(entries []eventLogEntry) string {
	for idx := len(entries) - 1; idx >= 0; idx-- {
		if agentID := strings.TrimSpace(entries[idx].AgentID); agentID != "" {
			return agentID
		}
	}
	return ""
}

func exportRoundLog(cfg *Config, chatID string, opts roundLogOptions) (string, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return "", fmt.Errorf("chatId is required")
	}
	normalizedOpts, err := normalizeRoundLogOptions(opts)
	if err != nil {
		return "", err
	}
	entries, err := queryEventLogs("", chatID)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("no log data found")
	}

	filteredEntries := filterRoundLogEntries(entries, normalizedOpts)
	if len(filteredEntries) == 0 {
		return "", fmt.Errorf("no log data found")
	}

	records := make([]roundLogRecord, 0, len(filteredEntries))
	for _, item := range filteredEntries {
		formattedContent := formatRoundLogContent(item.LogType, item.Content)
		if item.LogType == logTypeChatCompletionResponse && strings.TrimSpace(formattedContent) == "" {
			continue
		}
		records = append(records, roundLogRecord{
			ID:        item.ID,
			AgentID:   item.AgentID,
			ChatID:    item.ChatID,
			LogType:   item.LogType,
			TypeLabel: buildRoundTypeLabel(item.LogType),
			Content:   formattedContent,
			CreatedAt: item.CreatedAt,
		})
	}
	records = mergeSSEContent(records)

	exportAgentID := resolveRoundLogAgentID(filteredEntries)
	if exportAgentID == "" {
		exportAgentID = resolveRoundLogAgentID(entries)
	}
	if exportAgentID == "" {
		return "", fmt.Errorf("agent not found for chatId: %s", chatID)
	}
	workspace, err := getWorkspaceByAgentID(cfg, exportAgentID)
	if err != nil {
		return "", err
	}
	tmpDir := filepath.Join(workspace, "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%s_%s_%s.log",
		sanitizeLogRoundFilePart(exportAgentID),
		sanitizeLogRoundFilePart(chatID),
		roundLogTimestamp(),
	)
	path := filepath.Join(tmpDir, filename)
	if err := os.WriteFile(path, []byte(buildRoundMarkdown(records)), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func collectRoundGroups(entries []eventLogEntry) [][]eventLogEntry {
	if len(entries) == 0 {
		return nil
	}
	rounds := make([][]eventLogEntry, 0)
	var current []eventLogEntry
	for _, item := range entries {
		if item.LogType == logTypeChatCompletionRequest {
			if len(current) > 0 {
				rounds = append(rounds, current)
			}
			current = []eventLogEntry{item}
			continue
		}
		if len(current) == 0 {
			continue
		}
		current = append(current, item)
	}
	if len(current) > 0 {
		rounds = append(rounds, current)
	}
	return rounds
}

func isCompleteSSERound(entries []eventLogEntry) bool {
	for _, item := range entries {
		if item.LogType == logTypeChatCompletionResponse {
			return true
		}
	}
	return false
}

func latestCompleteSSERound(entries []eventLogEntry) []eventLogEntry {
	rounds := collectRoundGroups(entries)
	for idx := len(rounds) - 1; idx >= 0; idx-- {
		if isCompleteSSERound(rounds[idx]) {
			return rounds[idx]
		}
	}
	return nil
}

func countCLIGetInEntries(entries []eventLogEntry) int {
	count := 0
	for _, item := range entries {
		if item.LogType == logTypeCLIGet {
			count++
		}
	}
	return count
}

func querySkillStatus(chatID string, threshold int) (bool, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return false, fmt.Errorf("chatId is required")
	}
	if threshold <= 0 {
		threshold = 10
	}
	entries, err := queryEventLogs("", chatID)
	if err != nil {
		return false, err
	}
	if len(entries) == 0 {
		return false, nil
	}
	latestRound := latestCompleteSSERound(entries)
	if len(latestRound) == 0 {
		return false, nil
	}
	return countCLIGetInEntries(latestRound) >= threshold, nil
}

func resolveSkillExtractRound(cfg *Config) int {
	if cfg != nil && cfg.SkillExtractRound > 0 {
		return cfg.SkillExtractRound
	}
	return 10
}

func handleLogSkill(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		chatID := strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("chatId"), r.URL.Query().Get("chat")))
		roundStr := strings.TrimSpace(r.URL.Query().Get("round"))
		round := 0
		var err error
		if roundStr != "" {
			round, err = strconv.Atoi(roundStr)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": "round must be a positive integer"})
				return
			}
		}
		opts, err := normalizeRoundLogOptions(roundLogOptions{
			Round: round,
			Start: r.URL.Query().Get("start"),
			Close: r.URL.Query().Get("close"),
		})
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": err.Error()})
			return
		}

		path, err := exportRoundLog(cfg, chatID, opts)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": err.Error()})
			return
		}
		sizeK, err := fileSizeK(path)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": 0, "path": path, "sizeK": sizeK})
	}
}

func handleLogSkillStatus(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentID := strings.TrimSpace(r.URL.Query().Get("agentId"))
		chatID := strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("chatId"), r.URL.Query().Get("chat")))
		round := resolveSkillExtractRound(cfg)
		if roundStr := strings.TrimSpace(r.URL.Query().Get("round")); roundStr != "" {
			value, err := strconv.Atoi(roundStr)
			if err != nil || value <= 0 {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": "round must be a positive integer"})
				return
			}
			round = value
		}
		hasSkill, err := querySkillStatus(chatID, round)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":    0,
			"agentId":   agentID,
			"chatId":    chatID,
			"round":     round,
			"hasSkill":  hasSkill,
			"hasCLIPub": hasSkill,
			"source":    "db",
		})
	}
}

func handleChatSessionLog() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentID := strings.TrimSpace(r.URL.Query().Get("agentId"))
		chatID := strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("chatId"), r.URL.Query().Get("chat")))
		if chatID == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": "chatId is required"})
			return
		}
		limit, err := normalizeChatSessionLogLimit(r.URL.Query().Get("limit"))
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": err.Error()})
			return
		}
		startValue, closeValue, err := normalizeChatSessionLogRange(r.URL.Query().Get("start"), r.URL.Query().Get("close"))
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": err.Error()})
			return
		}
		entries, err := queryEventLogs("", chatID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": err.Error()})
			return
		}
		records := buildChatSessionLogRecords(filterChatSessionLogEntries(entries, startValue, closeValue), limit)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":   0,
			"agent_id": agentID,
			"chat_id":  chatID,
			"limit":    limit,
			"count":    len(records),
			"records":  records,
		})
	}
}

func handleLogRound(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		chatID := strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("chatId"), r.URL.Query().Get("chat")))
		round, err := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("round")))
		if err != nil || round <= 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": "round must be a positive integer"})
			return
		}
		path, err := exportRoundLog(cfg, chatID, roundLogOptions{Round: round})
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": 0, "path": path})
	}
}

func cronBuildExpr(cycle int, t time.Time) string {
	min := t.Minute()
	hour := t.Hour()
	switch cycle {
	case 0:
		return fmt.Sprintf("0 %d %d %d %d ? %d", min, hour, t.Day(), t.Month(), t.Year())
	case 1:
		return fmt.Sprintf("0 %d %d * * 1-5", min, hour)
	case 2:
		return fmt.Sprintf("0 %d %d * * ?", min, hour)
	case 3:
		return fmt.Sprintf("%d * * * *", min)
	case 4:
		return "*/15 * * * *"
	case 5:
		return "*/30 * * * *"
	}
	return ""
}

func cronCycleInterval(cycle int) time.Duration {
	switch cycle {
	case 3:
		return time.Hour
	case 4:
		return 15 * time.Minute
	case 5:
		return 30 * time.Minute
	default:
		return 0
	}
}

type cronCreateRequest struct {
	Content        string `json:"content"`
	Model          string `json:"model"`
	Thinking       bool   `json:"thinking"`
	Verify         bool   `json:"verify"`
	RouterDisable  bool   `json:"router_disable"`
	RawTime        string `json:"rawTime"`
	Cycle          int    `json:"cycle"`
	Cron           string `json:"cron"`
	ChatID         string `json:"chatId"`
	Type           string `json:"type"`
	ResponseSchema string `json:"responseSchema"`
}

func (r *cronCreateRequest) UnmarshalJSON(data []byte) error {
	type rawCronCreateRequest struct {
		Content        string `json:"content"`
		Model          string `json:"model"`
		Thinking       bool   `json:"thinking"`
		Verify         bool   `json:"verify"`
		RouterDisable  *bool  `json:"router_disable"`
		RawTime        string `json:"rawTime"`
		Cycle          int    `json:"cycle"`
		Cron           string `json:"cron"`
		ChatID         string `json:"chatId"`
		Type           string `json:"type"`
		ResponseSchema string `json:"responseSchema"`
	}

	var raw rawCronCreateRequest
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	r.Content = raw.Content
	r.Model = raw.Model
	r.Thinking = raw.Thinking
	r.Verify = raw.Verify
	r.RawTime = raw.RawTime
	r.Cycle = raw.Cycle
	r.Cron = raw.Cron
	r.ChatID = raw.ChatID
	r.Type = raw.Type
	r.ResponseSchema = raw.ResponseSchema
	r.RouterDisable = true
	if raw.RouterDisable != nil {
		r.RouterDisable = *raw.RouterDisable
	}
	return nil
}

func createCronTask(agentID string, req cronCreateRequest) (map[string]interface{}, error) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return nil, fmt.Errorf("agentId is required")
	}
	req.Model = strings.TrimSpace(req.Model)
	req.Content = strings.TrimSpace(req.Content)
	req.Cron = strings.TrimSpace(req.Cron)
	req.RawTime = strings.TrimSpace(req.RawTime)
	req.ChatID = strings.TrimSpace(req.ChatID)
	req.Type = sharedutil.NormalizeTaskType(req.Type)
	req.ResponseSchema = strings.TrimSpace(req.ResponseSchema)
	if req.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if req.Content == "" {
		return nil, fmt.Errorf("content is required")
	}

	var cronExpr string
	var rawTime string
	thinkInt := 0
	routerDisableInt := 0
	if req.Thinking {
		thinkInt = 1
	}
	verifyInt := 0
	if req.Verify {
		verifyInt = 1
	}
	if req.RouterDisable {
		routerDisableInt = 1
	}
	if req.Cycle == -1 && req.Cron != "" {
		cronExpr = req.Cron
		rawTime = ""
	} else {
		if req.Cycle < 0 || req.Cycle > 5 {
			return nil, fmt.Errorf("cycle must be one of 0,1,2,3,4,5")
		}
		t, err := time.ParseInLocation("2006-01-02 15:04", req.RawTime, time.Local)
		if err != nil {
			return nil, fmt.Errorf("invalid time: %w", err)
		}
		cronExpr = cronBuildExpr(req.Cycle, t)
		rawTime = req.RawTime
	}

	db := cronDB
	if db == nil {
		return nil, fmt.Errorf("db not ready")
	}

	ensureCronSchema(db)
	res, err := db.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, model, thinking, verify, router_disable, cron, content, response_schema, chat_id, task_type) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		req.Cycle, rawTime, agentID, req.Model, thinkInt, verifyInt, routerDisableInt, cronExpr, req.Content, req.ResponseSchema, req.ChatID, req.Type)
	if err != nil {
		return nil, fmt.Errorf("insert error: %w", err)
	}
	metaID, _ := res.LastInsertId()
	appendCronMetaLog(db, cronMetaLogEntry{
		MetaID:         int(metaID),
		AgentID:        agentID,
		ChatID:         req.ChatID,
		TaskType:       req.Type,
		Action:         "insert",
		Cycle:          req.Cycle,
		RawTime:        rawTime,
		Model:          req.Model,
		Thinking:       req.Thinking,
		Verify:         req.Verify,
		RouterDisable:  req.RouterDisable,
		Cron:           cronExpr,
		Content:        req.Content,
		ResponseSchema: req.ResponseSchema,
		CreatedAt:      cronLogTimestamp(),
	})

	if req.Cycle >= 0 && req.Cycle <= 2 && rawTime != "" {
		t, _ := time.ParseInLocation("2006-01-02 15:04", rawTime, time.Local)
		detailRes, _ := db.Exec(`INSERT OR IGNORE INTO task_detail (meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, verify, router_disable, content, response_schema, started) VALUES (?,?,?,?,?,?,?,?,?,?,?,0)`,
			metaID, t.Unix(), agentID, req.ChatID, req.Type, req.Model, thinkInt, verifyInt, routerDisableInt, req.Content, req.ResponseSchema)
		if detailRes != nil {
			if n, _ := detailRes.RowsAffected(); n > 0 {
				if detailID, err := detailRes.LastInsertId(); err == nil {
					appendCronDetailLog(db, cronDetailLogEntry{DetailID: int(detailID), MetaID: int(metaID), AgentID: agentID, ChatID: req.ChatID, TaskType: req.Type, Action: "insert", ExecTime: t.Unix(), Model: req.Model, Thinking: req.Thinking, Verify: req.Verify, RouterDisable: req.RouterDisable, Content: req.Content, ResponseSchema: req.ResponseSchema, Started: 0, OccurredAt: cronLogTimestamp()})
				}
			}
		}
		if req.Cycle == 1 || req.Cycle == 2 {
			end := t.Add(5 * 24 * time.Hour)
			day := t.AddDate(0, 0, 1)
			for !day.After(end) {
				if req.Cycle == 2 || (day.Weekday() >= time.Monday && day.Weekday() <= time.Friday) {
					detailRes, _ := db.Exec(`INSERT OR IGNORE INTO task_detail (meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, verify, router_disable, content, response_schema, started) VALUES (?,?,?,?,?,?,?,?,?,?,?,0)`,
						metaID, day.Unix(), agentID, req.ChatID, req.Type, req.Model, thinkInt, verifyInt, routerDisableInt, req.Content, req.ResponseSchema)
					if detailRes != nil {
						if n, _ := detailRes.RowsAffected(); n > 0 {
							if detailID, err := detailRes.LastInsertId(); err == nil {
								appendCronDetailLog(db, cronDetailLogEntry{DetailID: int(detailID), MetaID: int(metaID), AgentID: agentID, ChatID: req.ChatID, TaskType: req.Type, Action: "insert", ExecTime: day.Unix(), Model: req.Model, Thinking: req.Thinking, Verify: req.Verify, RouterDisable: req.RouterDisable, Content: req.Content, ResponseSchema: req.ResponseSchema, Started: 0, OccurredAt: cronLogTimestamp()})
							}
						}
					}
				}
				day = day.AddDate(0, 0, 1)
			}
		}
	}

	if req.Cycle >= 3 && req.Cycle <= 5 {
		interval := cronCycleInterval(req.Cycle)
		start, _ := time.ParseInLocation("2006-01-02 15:04", rawTime, time.Local)
		end := start.Add(5 * 24 * time.Hour)
		tick := start
		for !tick.After(end) {
			detailRes, _ := db.Exec(`INSERT OR IGNORE INTO task_detail (meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, verify, router_disable, content, response_schema, started) VALUES (?,?,?,?,?,?,?,?,?,?,?,0)`,
				metaID, tick.Unix(), agentID, req.ChatID, req.Type, req.Model, thinkInt, verifyInt, routerDisableInt, req.Content, req.ResponseSchema)
			if detailRes != nil {
				if n, _ := detailRes.RowsAffected(); n > 0 {
					if detailID, err := detailRes.LastInsertId(); err == nil {
						appendCronDetailLog(db, cronDetailLogEntry{DetailID: int(detailID), MetaID: int(metaID), AgentID: agentID, ChatID: req.ChatID, TaskType: req.Type, Action: "insert", ExecTime: tick.Unix(), Model: req.Model, Thinking: req.Thinking, Verify: req.Verify, RouterDisable: req.RouterDisable, Content: req.Content, ResponseSchema: req.ResponseSchema, Started: 0, OccurredAt: cronLogTimestamp()})
					}
				}
			}
			tick = tick.Add(interval)
		}
	}

	return map[string]interface{}{"status": 0, "id": metaID, "cron": cronExpr, "agentId": agentID, "type": req.Type}, nil
}

func validateIntegrationAgentExists(agentDir, deviceID, agentID string) error {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return fmt.Errorf("agentId is required")
	}
	cleanDir := filepath.Clean(agentDir)
	if filepath.Base(cleanDir) == agentID {
		if info, err := os.Stat(cleanDir); err == nil && info.IsDir() {
			return nil
		}
	}
	agentTTL := 120 * time.Second
	metadata, err := getAgentOutput(agentDir, deviceID, agentTTL)
	if err != nil {
		return fmt.Errorf("agent scan error: %w", err)
	}
	for _, agent := range metadata.Agents {
		if agent.AgentID == agentID {
			return nil
		}
	}
	return fmt.Errorf("agent not found: %s", agentID)
}

func validateIntegrationRegisteredModel(model string) error {
	if cronDB == nil {
		return fmt.Errorf("db not ready")
	}
	token, err := lookupTokenByModel(cronDB, model)
	if err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("model not registered: %s", strings.TrimSpace(model))
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Active connection manager (Chat dimension)
// ═══════════════════════════════════════════════════════════════════════════

type chatMsg struct {
	role         string
	content      string
	responseType string
}

type activeConn struct {
	cancel  context.CancelFunc
	ch      chan chatMsg
	agentID string
	chatID  string
	once    sync.Once
}

var (
	connMu  sync.Mutex
	connMap = make(map[string]*activeConn)
)

func connKey(chatID string) string { return strings.TrimSpace(chatID) }

const (
	chatTypePageSession   = "page_session"
	chatTypeScheduledTask = "scheduled_task"
)

const integrationCompletionNotificationTitle = "DeepRight通知"

func normalizeChatType(chatType string) string {
	switch strings.TrimSpace(strings.ToLower(chatType)) {
	case "", "user", "chat", "session", "page", "page_chat", "page_session":
		return chatTypePageSession
	case "cron", "task", "scheduled", "schedule", "scheduled_task":
		return chatTypeScheduledTask
	default:
		return chatType
	}
}

func normalizeResponseType(responseType string) string {
	switch strings.TrimSpace(strings.ToLower(responseType)) {
	case "abnormal", "error", "failed":
		return "abnormal"
	default:
		return "normal"
	}
}

func buildSSEStreamInterruptedContent(err error) string {
	if err == nil {
		return "SSE stream interrupted"
	}
	detail := strings.TrimSpace(err.Error())
	if detail == "" {
		return "SSE stream interrupted"
	}
	return "SSE stream interrupted: " + detail
}

func queueSSEStreamInterruptedLog(ch chan chatMsg, err error) bool {
	if ch == nil || err == nil {
		return false
	}
	ch <- chatMsg{
		role:         "A",
		responseType: "abnormal",
		content:      buildSSEStreamInterruptedContent(err),
	}
	return true
}

func sseEventHasDoneMarker(event string) bool {
	for _, line := range strings.Split(event, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "data:") {
			continue
		}
		if strings.TrimSpace(strings.TrimPrefix(trimmed, "data:")) == "[DONE]" {
			return true
		}
	}
	return false
}

func compactNotificationPrompt(input string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(input)), " ")
}

func truncateNotificationPrompt(input string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(compactNotificationPrompt(input))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "..."
}

func extractCompletionNotificationPrompt(reqData map[string]interface{}) string {
	if reqData == nil {
		return ""
	}
	rawMessages, ok := reqData["messages"].([]interface{})
	if !ok {
		return ""
	}
	for i := len(rawMessages) - 1; i >= 0; i-- {
		message, ok := rawMessages[i].(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := message["role"].(string)
		if strings.TrimSpace(role) != "user" {
			continue
		}
		content, _ := message["content"].(string)
		return truncateNotificationPrompt(content, 10)
	}
	return ""
}

var (
	integrationNotificationSupportedFn          = integrationnotification.Supported
	integrationSendNotificationFn               = integrationnotification.Notify
	integrationShouldOpenNotificationSettingsFn = integrationnotification.ShouldOpenSettingsForError
	integrationOpenNotificationSettingsFn       = integrationnotification.OpenSettings
)

func buildSSECompletionNotificationMessage(chatType, taskType, prompt string, abnormal bool) (string, bool) {
	switch normalizeChatType(chatType) {
	case chatTypePageSession:
		prompt = truncateNotificationPrompt(prompt, 10)
		if prompt != "" {
			return prompt, true
		}
		if abnormal {
			return "普通对话出现异常", true
		}
		return "已完成", true
	case chatTypeScheduledTask:
		if sharedutil.NormalizeTaskType(taskType) != sharedutil.DefaultTaskType {
			return "", false
		}
		if abnormal {
			return "备忘录任务出现异常", true
		}
		return "备忘录任务已完成", true
	default:
		return "", false
	}
}

func sendSSECompletionNotification(chatType, taskType, prompt string, abnormal bool) {
	if !integrationNotificationSupportedFn() {
		return
	}
	message, ok := buildSSECompletionNotificationMessage(chatType, taskType, prompt, abnormal)
	if !ok {
		return
	}
	if err := integrationSendNotificationFn(integrationnotification.Options{
		Title:   integrationCompletionNotificationTitle,
		Message: message,
	}); err != nil {
		log.Printf("[notification] send failed: chatType=%s taskType=%s abnormal=%t err=%v", chatType, taskType, abnormal, err)
	}
}

func maybeOpenIntegrationNotificationSettings(err error) {
	if !integrationShouldOpenNotificationSettingsFn(err) {
		return
	}
	if openErr := integrationOpenNotificationSettingsFn(); openErr != nil {
		log.Printf("[notification] open settings failed: %v", openErr)
	}
}

func detectResponseType(content string) string {
	for _, payload := range extractResponsePayloads(content) {
		if responsePayloadIsAbnormal(payload) {
			return "abnormal"
		}
	}
	return "normal"
}

func extractResponsePayloads(content string) []string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return []string{trimmed}
	}

	lines := strings.Split(content, "\n")
	payloads := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		payloads = append(payloads, payload)
	}
	return payloads
}

func responsePayloadIsAbnormal(payload string) bool {
	var raw interface{}
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return false
	}
	return responseValueIsAbnormal(raw)
}

func responseValueIsAbnormal(raw interface{}) bool {
	switch value := raw.(type) {
	case map[string]interface{}:
		if code, ok := parseResponseStatusCode(value["code"]); ok && (code < 200 || code >= 300) {
			return true
		}
		if reason, ok := value["finish_reason"].(string); ok && strings.EqualFold(strings.TrimSpace(reason), "error") {
			return true
		}
		if nested, ok := value["error"]; ok && responseValueIsAbnormal(nested) {
			return true
		}
		if choices, ok := value["choices"].([]interface{}); ok {
			for _, choice := range choices {
				if responseValueIsAbnormal(choice) {
					return true
				}
			}
		}
	case []interface{}:
		for _, item := range value {
			if responseValueIsAbnormal(item) {
				return true
			}
		}
	}
	return false
}

func parseResponseStatusCode(raw interface{}) (int, bool) {
	switch value := raw.(type) {
	case int:
		return value, true
	case int32:
		return int(value), true
	case int64:
		return int(value), true
	case float32:
		return int(value), true
	case float64:
		return int(value), true
	case string:
		code, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return 0, false
		}
		return code, true
	default:
		return 0, false
	}
}

func markConfirmedMessageInsertUploads(chatID, content string) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return
	}
	tids := extractConfirmedMessageInsertTIDs(content)
	if len(tids) == 0 {
		return
	}
	if err := markUploadedMessageInsertTIDs(chatID, tids); err != nil {
		log.Printf("message_insert mark uploaded failed chatId=%s tids=%d: %v", chatID, len(tids), err)
	}
}

func extractConfirmedMessageInsertTIDs(content string) []string {
	payloads := extractResponsePayloads(content)
	if len(payloads) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(payloads))
	for _, payload := range payloads {
		var raw interface{}
		if err := json.Unmarshal([]byte(payload), &raw); err != nil {
			continue
		}
		collectConfirmedMessageInsertTIDs(raw, seen, &result)
	}
	return result
}

func collectConfirmedMessageInsertTIDs(raw interface{}, seen map[string]struct{}, result *[]string) {
	switch value := raw.(type) {
	case map[string]interface{}:
		if metadata, ok := value["metadata"].(map[string]interface{}); ok {
			processKey := strings.TrimSpace(normalizeMessageInsertValue(metadata["__PROCESS__"]))
			tid := strings.TrimSpace(normalizeMessageInsertValue(metadata["__TID__"]))
			if strings.EqualFold(processKey, "rag_insert") && tid != "" {
				if _, exists := seen[tid]; !exists {
					seen[tid] = struct{}{}
					*result = append(*result, tid)
				}
			}
		}
		for _, nested := range value {
			collectConfirmedMessageInsertTIDs(nested, seen, result)
		}
	case []interface{}:
		for _, nested := range value {
			collectConfirmedMessageInsertTIDs(nested, seen, result)
		}
	}
}

func splitCompleteSSEEvents(buf *[]byte) []string {
	data := *buf
	var events []string
	for {
		end := bytes.Index(data, []byte("\n\n"))
		delimLen := 2
		if end < 0 {
			end = bytes.Index(data, []byte("\r\n\r\n"))
			delimLen = 4
		}
		if end < 0 {
			break
		}
		eventBytes := data[:end+delimLen]
		if len(eventBytes) > 0 {
			events = append(events, string(eventBytes))
		}
		data = data[end+delimLen:]
	}
	*buf = data
	return events
}

func flushTrailingSSEBytes(buf *[]byte) []string {
	if len(*buf) == 0 {
		return nil
	}
	if !utf8.Valid(*buf) {
		return nil
	}
	event := string(*buf)
	*buf = (*buf)[:0]
	return []string{event}
}

func appendChatLogDB(agentID, chatID, chatType, role, responseType, content string) {
	if cronDB == nil {
		return
	}
	createdAt := time.Now().Format("2006-01-02T15:04:05.000")
	cronDB.Exec(`INSERT INTO chat_log (agent_id, chat_id, chat_type, role, response_type, content, created_at) VALUES (?,?,?,?,?,?,?)`,
		agentID, chatID, normalizeChatType(chatType), role, normalizeResponseType(responseType), content, createdAt)
	switch role {
	case "Q":
		appendEventLogDB(agentID, chatID, content, logTypeChatCompletionRequest, createdAt)
	case "A":
		appendEventLogDB(agentID, chatID, content, logTypeChatCompletionResponse, createdAt)
	}
}

func appendEventLogDB(agentID, chatID, content string, logType int, createdAt string) {
	if cronDB == nil {
		return
	}
	if strings.TrimSpace(createdAt) == "" {
		createdAt = time.Now().Format("2006-01-02T15:04:05.000")
	}
	cronDB.Exec(`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES (?,?,?,?,?)`,
		strings.TrimSpace(agentID), strings.TrimSpace(chatID), content, logType, createdAt)
}

func queryEventLogs(agentID, chatID string, types ...int) ([]eventLogEntry, error) {
	if cronDB == nil {
		return nil, fmt.Errorf("db not ready")
	}
	trimmedAgentID := strings.TrimSpace(agentID)
	trimmedChatID := strings.TrimSpace(chatID)
	args := make([]any, 0, 2+len(types))
	args = append(args, trimmedChatID)
	query := `SELECT id, agent_id, chat_id, content, log_type, created_at FROM agent_message_log WHERE chat_id = ?`
	if trimmedAgentID != "" {
		args = append(args[:0], trimmedAgentID, trimmedChatID)
		query = `SELECT id, agent_id, chat_id, content, log_type, created_at FROM agent_message_log WHERE agent_id = ? AND chat_id = ?`
	}
	if len(types) > 0 {
		holders := make([]string, 0, len(types))
		for _, item := range types {
			holders = append(holders, "?")
			args = append(args, item)
		}
		query += ` AND log_type IN (` + strings.Join(holders, ",") + `)`
	}
	query += ` ORDER BY created_at, id`
	rows, err := cronDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]eventLogEntry, 0)
	for rows.Next() {
		var item eventLogEntry
		if err := rows.Scan(&item.ID, &item.AgentID, &item.ChatID, &item.Content, &item.LogType, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func handleCancel(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		chatID := r.URL.Query().Get("chat")
		if chatID == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "chat is required"})
			return
		}
		key := connKey(chatID)
		connMu.Lock()
		ac, exists := connMap[key]
		if exists {
			if ac.ch != nil {
				select {
				case ac.ch <- chatMsg{role: "X", content: ""}:
				default:
				}
			}
			ac.cancel()
			delete(connMap, key)
		}
		connMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "cancelled": exists})
	}
}

func syncForwardedBoolField(reqData map[string]interface{}, metaMap map[string]interface{}, key string) {
	if reqData == nil || metaMap == nil {
		return
	}
	if value, ok := sharedutil.ToBoolValue(metaMap[key]); ok {
		metaMap[key] = value
		delete(reqData, key)
		return
	}
	if value, ok := sharedutil.ToBoolValue(reqData[key]); ok {
		metaMap[key] = value
	}
	delete(reqData, key)
}

func syncForwardedChatRequestFlags(reqData map[string]interface{}, metaMap map[string]interface{}) {
	syncForwardedBoolField(reqData, metaMap, "thinking")
	syncForwardedBoolField(reqData, metaMap, "verify")
	syncForwardedBoolField(reqData, metaMap, "html")
	syncForwardedBoolField(reqData, metaMap, "router_disable")
}

func pruneForwardedChatMetadata(metaMap map[string]interface{}) {
	if metaMap == nil {
		return
	}
	pruneForwardedKnowledgeCommit(metaMap)
	for _, key := range []string{
		"agent",
		"workspace",
		"skills",
		"media",
		"soul",
		"user",
		"dir",
	} {
		delete(metaMap, key)
	}
}

func normalizeForwardedChatRequest(reqData map[string]interface{}) error {
	if reqData == nil {
		return fmt.Errorf("request body is required")
	}
	if rawMessages, ok := reqData["messages"].([]interface{}); ok && len(rawMessages) > 0 {
		delete(reqData, "message")
		return nil
	}
	if message, ok := reqData["message"]; ok && message != nil {
		reqData["messages"] = []map[string]interface{}{
			{"role": "user", "content": message},
		}
		delete(reqData, "message")
		return nil
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Proxy: SSE stream proxy handler
// ═══════════════════════════════════════════════════════════════════════════

func handleChatCompletions(cfg *Config, proxyClient *http.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		bodyBytes, err := io.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusInternalServerError)
			return
		}

		var reqData map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &reqData); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Inject metadata
		agentTTL := time.Duration(cfg.AgentCacheMs) * time.Millisecond
		_, requestedChatID := requestChatContext(reqData["metadata"])
		metadata, err := getAgentOutputForChat(cfg.AgentDir, cfg.effectiveDeviceID(), agentTTL, requestedChatID)
		if err != nil {
			log.Printf("proxy: agent scan error: %v", err)
			http.Error(w, "Failed to get metadata", http.StatusInternalServerError)
			return
		}
		metaBytes, _ := json.Marshal(metadata)
		var metaMap map[string]interface{}
		json.Unmarshal(metaBytes, &metaMap)

		// Merge request metadata onto the shared Agent metadata and forward the
		// merged body as-is without any query-based metadata expansion.
		sharedutil.MergeMetadataFields(metaMap, reqData["metadata"])
		metaMap["port"] = integrationMetadataPort(cfg)
		// plugins is runtime-owned metadata. A client request can contain a stale
		// plugin list, but it must not replace the built-in remote capability.
		metaMap["plugins"] = detectPluginKeys()
		var selectedModelCfg tokenConfig
		if db, err := sql.Open("sqlite", resolveIntegrationDBPath()); err != nil {
			log.Printf("integration: provider config lookup failed: %v", err)
			http.Error(w, "Failed to read provider config", http.StatusInternalServerError)
			return
		} else {
			if modelName, _ := reqData["model"].(string); strings.TrimSpace(modelName) != "" {
				modelCfg, err := lookupTokenConfigByModel(db, modelName)
				_ = db.Close()
				if err != nil {
					log.Printf("integration: provider config lookup failed: %v", err)
					http.Error(w, "Failed to read provider config", http.StatusInternalServerError)
					return
				}
				selectedModelCfg = modelCfg
				injectConfiguredModelMetadata(metaMap, modelCfg)
			} else {
				_ = db.Close()
			}
		}
		knowledgeCommit := false
		knowledgeCommitProvided := false
		if mm, ok := reqData["metadata"].(map[string]interface{}); ok {
			if value, ok := sharedutil.ToBoolValue(mm["knowledge_commit"]); ok {
				knowledgeCommit = value
				knowledgeCommitProvided = true
			}
		}
		if knowledgeCommitProvided {
			agentID, _ := metaMap["agentId"].(string)
			if err := persistKnowledgeCommitSelection(strings.TrimSpace(agentID), knowledgeCommit); err != nil {
				log.Printf("integration: knowledge commit persist error: %v", err)
				http.Error(w, "Failed to persist knowledge commit", http.StatusInternalServerError)
				return
			}
			if knowledgeMeta, ok := metaMap["knowledge"].(map[string]interface{}); ok {
				knowledgeMeta["knowledgeCommit"] = knowledgeCommit
			}
		}
		if err := processKnowledgeMetadata(cfg, metaMap, time.Now(), knowledgeCommit); err != nil {
			log.Printf("proxy: knowledge metadata process error: %v", err)
			http.Error(w, "Failed to process knowledge metadata", http.StatusInternalServerError)
			return
		}
		injectLiveAgentMediaIntoAgentList(metaMap)
		injectLiveAgentKnowledgeIntoAgentList(metaMap)
		_, mergedChatID := requestChatContext(metaMap)
		injectLastResponseMetadata(metaMap, mergedChatID)
		injectSandboxPathMetadata(metaMap, mergedChatID)
		injectPluginsDirMetadata(metaMap)
		reqData["metadata"] = metaMap
		syncForwardedChatRequestFlags(reqData, metaMap)

		// Extract agentId, chatId and type for chat logging
		var chatAgentID, chatID, chatType string
		if mm, ok := reqData["metadata"].(map[string]interface{}); ok {
			if v, ok := mm["agentId"].(string); ok {
				chatAgentID = v
			}
			if v, ok := mm["chat"].(string); ok {
				chatID = v
			}
			if v, ok := mm["type"].(string); ok {
				chatType = v
			}
		}
		chatType = normalizeChatType(chatType)
		if mm, ok := reqData["metadata"].(map[string]interface{}); ok {
			mm["type"] = chatType
			pruneForwardedChatMetadata(mm)
		}

		if err := normalizeForwardedChatRequest(reqData); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Cancel any existing connection for this chat, even if the bound agent changed.
		key := connKey(chatID)
		if chatID != "" {
			connMu.Lock()
			if old, exists := connMap[key]; exists {
				old.cancel()
				delete(connMap, key)
			}
			connMu.Unlock()
		}

		ctx, cancel := context.WithCancel(context.Background())

		newBody, err := json.Marshal(reqData)
		if err != nil {
			cancel()
			http.Error(w, "Failed to marshal request", http.StatusInternalServerError)
			return
		}

		targetURL := strings.TrimRight(cfg.currentHost(), "/") + "/v1/chat/completions"
		proxyReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(newBody))
		if err != nil {
			cancel()
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}

		// Start async writer goroutine and register active connection early so
		// /api/cancel can hit even before the first SSE chunk arrives.
		var ch chan chatMsg
		if chatID != "" {
			ch = make(chan chatMsg, 64)
			go func() {
				for msg := range ch {
					if msg.role == "X" {
						appendChatLogDB(chatAgentID, chatID, chatType, "X", msg.responseType, "")
						return
					}
					if msg.role == "A" && msg.content != "" {
						appendChatLogDB(chatAgentID, chatID, chatType, "A", msg.responseType, msg.content)
					}
				}
			}()

			connMu.Lock()
			connMap[key] = &activeConn{cancel: cancel, ch: ch, agentID: chatAgentID, chatID: chatID}
			connMu.Unlock()
		}

		// Copy headers unchanged
		for k, vv := range r.Header {
			for _, v := range vv {
				proxyReq.Header.Add(k, v)
			}
		}
		if auth := strings.TrimSpace(proxyReq.Header.Get("Authorization")); (auth == "" || auth == maskedTokenValue) && strings.TrimSpace(selectedModelCfg.Token) != "" {
			proxyReq.Header.Set("Authorization", selectedModelCfg.Token)
		}
		proxyReq.Header.Set("Content-Length", fmt.Sprintf("%d", len(newBody)))
		proxyReq.Header.Set("Accept", "text/event-stream")

		resp, err := proxyClient.Do(proxyReq)
		if err != nil {
			if chatID != "" {
				connMu.Lock()
				if ac, exists := connMap[key]; exists {
					delete(connMap, key)
					closeActiveConn(ac)
				}
				connMu.Unlock()
			}
			cancel()
			if chatID != "" {
				appendChatLogDB(chatAgentID, chatID, chatType, "A", "abnormal", "Failed to forward: "+err.Error())
			}
			http.Error(w, "Failed to forward", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "text/event-stream")
		}
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Accel-Buffering", "no")
		// Save the normalized forwarded request body so request logs match the
		// metadata that actually reached the upstream integration boundary.
		if chatID != "" {
			go appendChatLogDB(chatAgentID, chatID, chatType, "Q", "normal", string(newBody))
		}

		w.WriteHeader(resp.StatusCode)
		abnormalStream := resp.StatusCode < 200 || resp.StatusCode >= 300
		sawAbnormalPacket := false
		sawDoneMarker := false

		flusher, ok := w.(http.Flusher)
		if !ok {
			payload, copyErr := io.ReadAll(resp.Body)
			if len(payload) > 0 {
				_, _ = w.Write(payload)
				if detectResponseType(string(payload)) == "abnormal" {
					abnormalStream = true
				}
				if ch != nil {
					payloadBuf := append([]byte(nil), payload...)
					for _, event := range splitCompleteSSEEvents(&payloadBuf) {
						if sseEventHasDoneMarker(event) {
							sawDoneMarker = true
						}
						responseType := detectResponseType(event)
						if responseType == "abnormal" {
							sawAbnormalPacket = true
							abnormalStream = true
						}
						markConfirmedMessageInsertUploads(chatID, event)
						ch <- chatMsg{role: "A", content: event, responseType: responseType}
					}
					for _, event := range flushTrailingSSEBytes(&payloadBuf) {
						if sseEventHasDoneMarker(event) {
							sawDoneMarker = true
						}
						responseType := detectResponseType(event)
						if responseType == "abnormal" {
							sawAbnormalPacket = true
							abnormalStream = true
						}
						markConfirmedMessageInsertUploads(chatID, event)
						ch <- chatMsg{role: "A", content: event, responseType: responseType}
					}
				}
			}
			if copyErr != nil && !errors.Is(copyErr, io.EOF) && !errors.Is(ctx.Err(), context.Canceled) {
				abnormalStream = true
				if !sawDoneMarker && !sawAbnormalPacket {
					sawAbnormalPacket = queueSSEStreamInterruptedLog(ch, copyErr) || sawAbnormalPacket
				}
			} else if !errors.Is(ctx.Err(), context.Canceled) && !sawDoneMarker && !sawAbnormalPacket {
				abnormalStream = true
				sawAbnormalPacket = queueSSEStreamInterruptedLog(ch, io.EOF) || sawAbnormalPacket
			}
			if ch != nil {
				connMu.Lock()
				if ac, exists := connMap[key]; exists {
					closeActiveConn(ac)
				} else {
					closeActiveConn(&activeConn{ch: ch})
				}
				connMu.Unlock()
			}
			if knowledgeCommit {
				if err := updateKnowledgeManualTimestamps(time.Now(), chatAgentID); err != nil {
					log.Printf("proxy: knowledge manual update failed: %v", err)
				}
			}
			if !errors.Is(ctx.Err(), context.Canceled) {
				sendSSECompletionNotification(chatType, "", extractCompletionNotificationPrompt(reqData), abnormalStream)
			}
			if chatID != "" {
				connMu.Lock()
				delete(connMap, key)
				connMu.Unlock()
			}
			cancel()
			return
		}

		buf := make([]byte, 256)
		logBuf := make([]byte, 0, 1024)
		var streamErr error
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				w.Write(buf[:n])
				flusher.Flush()
				logBuf = append(logBuf, buf[:n]...)
				for _, event := range splitCompleteSSEEvents(&logBuf) {
					if sseEventHasDoneMarker(event) {
						sawDoneMarker = true
					}
					responseType := detectResponseType(event)
					if responseType == "abnormal" {
						sawAbnormalPacket = true
						abnormalStream = true
					}
					markConfirmedMessageInsertUploads(chatID, event)
					if ch != nil {
						ch <- chatMsg{role: "A", content: event, responseType: responseType}
					}
				}
			}
			if err != nil {
				streamErr = err
				break
			}
		}
		if len(logBuf) > 0 {
			for _, event := range flushTrailingSSEBytes(&logBuf) {
				if sseEventHasDoneMarker(event) {
					sawDoneMarker = true
				}
				responseType := detectResponseType(event)
				if responseType == "abnormal" {
					sawAbnormalPacket = true
					abnormalStream = true
				}
				markConfirmedMessageInsertUploads(chatID, event)
				if ch != nil {
					ch <- chatMsg{role: "A", content: event, responseType: responseType}
				}
			}
		}
		if streamErr != nil && !errors.Is(ctx.Err(), context.Canceled) {
			if !errors.Is(streamErr, io.EOF) || !sawDoneMarker {
				abnormalStream = true
			}
			if !sawDoneMarker && !sawAbnormalPacket {
				sawAbnormalPacket = queueSSEStreamInterruptedLog(ch, streamErr) || sawAbnormalPacket
			}
		}
		if ch != nil {
			connMu.Lock()
			if ac, exists := connMap[key]; exists {
				closeActiveConn(ac)
			} else {
				closeActiveConn(&activeConn{ch: ch})
			}
			connMu.Unlock()
		}
		if knowledgeCommit {
			if err := updateKnowledgeManualTimestamps(time.Now(), chatAgentID); err != nil {
				log.Printf("proxy: knowledge manual update failed: %v", err)
			}
		}
		if !errors.Is(ctx.Err(), context.Canceled) {
			sendSSECompletionNotification(chatType, "", extractCompletionNotificationPrompt(reqData), abnormalStream)
		}

		// Clean up connection
		if chatID != "" {
			connMu.Lock()
			delete(connMap, key)
			connMu.Unlock()
		}
		cancel()
	}
}

func handleInstallApp() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		apps := integrationFilterMissingInstallApps(mergeInstallApps(detectRequiredInstallApps(), cfgRuntimeInstallApps()))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "max-age=300")
		json.NewEncoder(w).Encode(apps)
	}
}

func handleIntegrationBrowserLaunch() http.HandlerFunc {
	launchHTML := func(target string) string {
		return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>DeepRight</title>
<style>
html,body{height:100%%;margin:0;font-family:"SF Pro Display","PingFang SC","Segoe UI",sans-serif;background:radial-gradient(circle at top,#eff6ff,#dbeafe 42%%,#bfdbfe 100%%);color:#10203f}
body{display:flex;align-items:center;justify-content:center;padding:24px}
.card{width:min(460px,calc(100vw - 32px));padding:30px 26px;border-radius:24px;background:rgba(255,255,255,.86);border:1px solid rgba(148,163,184,.28);box-shadow:0 24px 80px rgba(15,23,42,.12);backdrop-filter:blur(18px);display:flex;flex-direction:column;align-items:center;justify-content:center;text-align:center}
.badge{width:48px;height:48px;border-radius:16px;display:flex;align-items:center;justify-content:center;background:linear-gradient(180deg,#fff7d6,#ffe28a);box-shadow:0 12px 24px rgba(255,191,0,.18)}
.spinner{width:20px;height:20px;border-radius:50%%;border:2px solid rgba(184,134,11,.22);border-top-color:#c78600;border-right-color:#e0a125;animation:spin .8s linear infinite}
h1{margin:16px 0 0;font-size:20px;line-height:1.2;white-space:nowrap}
p{margin:8px 0 0;font-size:13px;line-height:1.4;color:#465b84;white-space:nowrap}
.retry{margin-top:18px;padding:10px 14px;border:none;border-radius:999px;background:#2563eb;color:#fff;font-size:13px;cursor:pointer;box-shadow:0 12px 22px rgba(37,99,235,.22)}
@media (max-width:520px){.card{width:min(460px,calc(100vw - 24px));padding:26px 18px}h1,p{white-space:normal}}
@keyframes spin{to{transform:rotate(360deg)}}
</style>
</head>
<body>
<main class="card" role="status" aria-live="polite">
  <div class="badge"><div class="spinner" aria-hidden="true"></div></div>
  <h1>正在启动</h1>
  <p id="launchDesc">等待完成</p>
  <button id="launchRetry" class="retry" type="button">立即重试</button>
</main>
<script>
const target=%s;
const entry='/site/';
const desc=document.getElementById('launchDesc');
const retry=document.getElementById('launchRetry');
let tries=0;
let done=false;
function openTarget(){if(done)return;done=true;window.location.replace(target);}
function schedule(delay){window.setTimeout(probe,delay);}
async function probe(){
  if(done)return;
  tries+=1;
  if(desc&&tries>=8)desc.textContent='首次打开新安装包时，页面资源可能还在准备';
  try{
    const resp=await fetch(entry,{cache:'no-store',credentials:'same-origin'});
    if(resp&&resp.ok){openTarget();return;}
  }catch(_){}
  schedule(Math.min(1200,150+tries*120));
}
retry&&retry.addEventListener('click',()=>{done=false;probe();});
probe();
</script>
</body>
</html>`, strconv.Quote(target))
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		host := "localhost"
		port := integrationServicePort
		if requestHost := strings.TrimSpace(r.Host); requestHost != "" {
			if parsedHost, parsedPort, err := net.SplitHostPort(requestHost); err == nil {
				if strings.TrimSpace(parsedHost) != "" {
					host = parsedHost
				}
				if value, err := strconv.Atoi(parsedPort); err == nil && value > 0 {
					port = value
				}
			} else {
				host = requestHost
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		if r.Method == http.MethodHead {
			return
		}
		_, _ = io.WriteString(w, launchHTML(integrationBuildBrowserURL(host, port)))
	}
}

func cfgRuntimeInstallApps() string {
	apps := integrationConfiguredInstallApps()
	return strings.Join(apps, ",")
}

func integrationConfiguredInstallApps() []string {
	values, _, err := readIntegrationStartupConfig()
	if err != nil || values == nil {
		return nil
	}
	return integrationInstallAppsFromConfigValue(values["install_app"], integrationInstallAppPlatformKeyFn())
}

func closeActiveConn(ac *activeConn) {
	if ac == nil {
		return
	}
	ac.once.Do(func() {
		if ac.ch != nil {
			close(ac.ch)
			ac.ch = nil
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// Static: file server
// ═══════════════════════════════════════════════════════════════════════════

func registerStatic(mux *http.ServeMux, siteDir string) {
	if err := server.Register(mux, siteDir); err != nil {
		log.Printf("static register failed: %v", err)
	}
}

func registerAppStatic(mux *http.ServeMux, cfg *Config) {
	if mux == nil {
		return
	}
	mux.HandleFunc("/mapping", handleAppStatic(cfg))
	mux.HandleFunc("/mapping/", handleAppStatic(cfg))
	agentDir := ""
	if cfg != nil {
		agentDir = cfg.AgentDir
	}
	root, err := resolveIntegrationAgentDir(agentDir)
	if err != nil {
		log.Printf("app static register failed: %v", err)
		return
	}
	log.Printf("static: serving /mapping/{agentId}/ from %s/*/app", root)
}

// ═══════════════════════════════════════════════════════════════════════════
// Config & main
// ═══════════════════════════════════════════════════════════════════════════

const integrationDeviceRefreshInterval = time.Minute

const (
	integrationDeviceSourceFlag   = "flag"
	integrationDeviceSourceConfig = "config"
	integrationDeviceSourceSystem = "system"
)

var integrationGenerateDeviceID = agentcore.GenerateDeviceID

type integrationDeviceSnapshot struct {
	ID     string
	Source string
}

type integrationDeviceConfigVersion struct {
	exists      bool
	modifiedAt  time.Time
	size        int64
	statErrText string
}

func (v integrationDeviceConfigVersion) equal(other integrationDeviceConfigVersion) bool {
	return v.exists == other.exists && v.modifiedAt.Equal(other.modifiedAt) && v.size == other.size && v.statErrText == other.statErrText
}

type integrationDeviceState struct {
	flagDevice   string
	configDevice string
	fallbackID   string
	current      atomic.Value // integrationDeviceSnapshot
	lastVersion  integrationDeviceConfigVersion
}

func newIntegrationDeviceState(flagDevice, configDevice, fallbackID string) *integrationDeviceState {
	state := &integrationDeviceState{
		flagDevice:   strings.TrimSpace(flagDevice),
		configDevice: strings.TrimSpace(configDevice),
		fallbackID:   strings.TrimSpace(fallbackID),
	}
	state.current.Store(state.nextSnapshot())
	return state
}

func (s *integrationDeviceState) nextSnapshot() integrationDeviceSnapshot {
	if s == nil {
		return integrationDeviceSnapshot{}
	}
	if s.flagDevice != "" {
		return integrationDeviceSnapshot{ID: s.flagDevice, Source: integrationDeviceSourceFlag}
	}
	if s.configDevice != "" {
		return integrationDeviceSnapshot{ID: s.configDevice, Source: integrationDeviceSourceConfig}
	}
	return integrationDeviceSnapshot{ID: s.fallbackID, Source: integrationDeviceSourceSystem}
}

func (s *integrationDeviceState) snapshot() integrationDeviceSnapshot {
	if s == nil {
		return integrationDeviceSnapshot{}
	}
	if value := s.current.Load(); value != nil {
		if snapshot, ok := value.(integrationDeviceSnapshot); ok {
			return snapshot
		}
	}
	return s.nextSnapshot()
}

func (s *integrationDeviceState) effectiveDeviceID() string {
	return s.snapshot().ID
}

func integrationDeviceConfigVersionForPath(path string) integrationDeviceConfigVersion {
	info, err := os.Stat(path)
	if err == nil {
		return integrationDeviceConfigVersion{exists: true, modifiedAt: info.ModTime(), size: info.Size()}
	}
	if errors.Is(err, os.ErrNotExist) {
		return integrationDeviceConfigVersion{}
	}
	return integrationDeviceConfigVersion{statErrText: err.Error()}
}

func integrationConfiguredDeviceValue(raw interface{}) string {
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func readIntegrationConfiguredDevice() (string, error) {
	raw, _, err := readIntegrationStartupConfigRaw()
	if err != nil {
		return "", err
	}
	if raw == nil {
		return "", nil
	}
	return integrationConfiguredDeviceValue(raw["device"]), nil
}

func (s *integrationDeviceState) captureConfigVersion() {
	if s == nil {
		return
	}
	s.lastVersion = integrationDeviceConfigVersionForPath(integrationStartupConfigPath())
}

func (s *integrationDeviceState) refreshConfig() {
	if s == nil {
		return
	}
	version := integrationDeviceConfigVersionForPath(integrationStartupConfigPath())
	if version.equal(s.lastVersion) {
		return
	}
	s.lastVersion = version
	if version.statErrText != "" {
		current := s.snapshot()
		log.Printf("device refresh failed source=%s err=%s", current.Source, version.statErrText)
		return
	}

	configDevice, err := readIntegrationConfiguredDevice()
	if err != nil {
		current := s.snapshot()
		log.Printf("device refresh failed source=%s err=%v", current.Source, err)
		return
	}
	previous := s.snapshot()
	s.configDevice = configDevice
	current := s.nextSnapshot()
	s.current.Store(current)
	log.Printf("device refresh succeeded source=%s changed=%t", current.Source, previous != current)
}

func startIntegrationDeviceConfigRefresh(ctx context.Context, state *integrationDeviceState) {
	if ctx == nil || state == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(integrationDeviceRefreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				state.refreshConfig()
			}
		}
	}()
}

func initializeIntegrationDeviceState(cfg *Config, explicitDevice string) error {
	if cfg == nil {
		return fmt.Errorf("device configuration is required")
	}
	configDevice, err := readIntegrationConfiguredDevice()
	if err != nil {
		return err
	}
	explicitDevice = strings.TrimSpace(explicitDevice)
	state := newIntegrationDeviceState(explicitDevice, configDevice, integrationGenerateDeviceID())
	cfg.DeviceState = state
	if explicitDevice != "" {
		cfg.Device = explicitDevice
	} else {
		cfg.Device = configDevice
	}
	return nil
}

func (cfg *Config) effectiveDeviceID() string {
	if cfg != nil && cfg.DeviceState != nil {
		return cfg.DeviceState.effectiveDeviceID()
	}
	if cfg == nil {
		return ""
	}
	return strings.TrimSpace(cfg.Device)
}

type Config struct {
	// Shared
	Port           int
	Host           string
	RuntimeHost    *runtimehost.State
	Standalone     *integrationstandalone.State
	AgentDir       string
	DefaultDir     string
	Device         string
	DeviceState    *integrationDeviceState
	AgentCacheMs   int
	ConnectCacheMs int
	Site           string
	Provider       map[string]map[string]string

	// Proxy specific
	ConnectTimeoutMs          int
	KnowledgeUpdateIntervalMs int
	KnowledgeUpdateLockMs     int
	InstallApp                string
	Reply                     string

	// cli-get specific
	SleepMs           int
	Thread            int
	Queue             int
	RetryIntervalMs   int
	RetryTimes        int
	HTTPTimeout       int
	HTTPConnTimeout   int
	HTTPSocketTimeout int
	HTTPDebug         bool
	IdleTimeout       int
	PluginExecTimeout int
	SkillExtractRound int
}

type integrationStartupOptions struct {
	Config  Config
	PIDFile string
	LogFile string
}

const (
	integrationServicePort      = 8080
	integrationDefaultPIDFile   = "integration.pid"
	integrationDefaultLogFile   = "integration.log"
	integrationLogRetentionDays = 3
	integrationStopWait         = 15 * time.Second
	integrationShutdownTimeout  = 5 * time.Second
	integrationReadyPath        = "/api/heartbeat"
	defaultPluginExecTimeoutMs  = 600000
	integrationSkipBrowserEnv   = "DEEPRIGHT_INTEGRATION_SKIP_BROWSER"
)

var integrationStartWait = 5 * time.Second
var integrationShutdownDelay = 5 * time.Second
var integrationBrowserEntryWait = 8 * time.Second
var integrationBrowserEntryPollInterval = 100 * time.Millisecond
var integrationBrowserEntryProbeTimeout = 1500 * time.Millisecond
var integrationReadyCheck = integrationReady
var integrationBrowserEntryReadyCheck = integrationBrowserEntryReady
var integrationOpenBrowserFn = openIntegrationBrowserMaximized
var integrationOpenBrowserWithoutActivationFn = openIntegrationBrowserWithoutActivation
var integrationStartLaunchSplashFn = startIntegrationLaunchSplash
var integrationStartProcessFn = startIntegrationProcess
var integrationPrimeSandboxModeFn = primeIntegrationSandboxMode
var integrationPrimeSandboxDirectoryFn = primeIntegrationSandboxDirectory

type serviceCommandResult struct {
	Status  string `json:"status"`
	PID     int    `json:"pid,omitempty"`
	PIDFile string `json:"pidFile,omitempty"`
	LogFile string `json:"logFile,omitempty"`
	Addr    string `json:"addr,omitempty"`
}

type integrationStartupStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

const defaultUpstreamHost = "https://www.deepright.cn"

func (cfg *Config) ensureRuntimeHostState() *runtimehost.State {
	if cfg == nil {
		return runtimehost.New(defaultUpstreamHost)
	}
	if cfg.RuntimeHost == nil {
		cfg.RuntimeHost = runtimehost.New(firstNonEmpty(strings.TrimSpace(cfg.Host), defaultUpstreamHost))
	}
	return cfg.RuntimeHost
}

func (cfg *Config) currentHost() string {
	if cfg == nil {
		return defaultUpstreamHost
	}
	host := strings.TrimSpace(cfg.ensureRuntimeHostState().Current())
	if host == "" {
		return defaultUpstreamHost
	}
	return host
}

func (cfg *Config) runtimeHostSnapshot() runtimehost.Snapshot {
	if cfg == nil {
		return runtimehost.New(defaultUpstreamHost).Snapshot()
	}
	return cfg.ensureRuntimeHostState().Snapshot()
}

func (cfg *Config) setRuntimeHost(host string) (runtimehost.Snapshot, error) {
	if cfg == nil {
		return runtimehost.Snapshot{}, fmt.Errorf("runtime host is not initialized")
	}
	return cfg.ensureRuntimeHostState().Set(host)
}

func (cfg *Config) resetRuntimeHost() runtimehost.Snapshot {
	if cfg == nil {
		return runtimehost.New(defaultUpstreamHost).Snapshot()
	}
	return cfg.ensureRuntimeHostState().Reset()
}

func (cfg *Config) ensureStandaloneState() *integrationstandalone.State {
	if cfg == nil {
		return integrationstandalone.New(false)
	}
	if cfg.Standalone == nil {
		cfg.Standalone = integrationstandalone.New(false)
	}
	return cfg.Standalone
}

func (cfg *Config) standaloneEnabled() bool {
	if cfg == nil {
		return false
	}
	return cfg.ensureStandaloneState().Enabled()
}

func (cfg *Config) standaloneSnapshot() integrationstandalone.Snapshot {
	if cfg == nil {
		return integrationstandalone.New(false).Snapshot()
	}
	return cfg.ensureStandaloneState().Snapshot()
}

func (cfg *Config) setStandaloneEnabled(enabled bool) integrationstandalone.Snapshot {
	if cfg == nil {
		return integrationstandalone.New(false).Set(enabled)
	}
	return cfg.ensureStandaloneState().Set(enabled)
}

func (cfg *Config) resetStandalone() integrationstandalone.Snapshot {
	if cfg == nil {
		return integrationstandalone.New(false).Reset()
	}
	return cfg.ensureStandaloneState().Reset()
}

func defaultIntegrationStartupOptions() integrationStartupOptions {
	return integrationStartupOptions{
		Config: Config{
			Port:                      integrationServicePort,
			Host:                      defaultUpstreamHost,
			AgentDir:                  integrationDefaultAgentDir(),
			DefaultDir:                "config",
			Device:                    "",
			AgentCacheMs:              120000,
			ConnectCacheMs:            10000,
			Site:                      "site",
			ConnectTimeoutMs:          15000,
			KnowledgeUpdateIntervalMs: 7200000,
			KnowledgeUpdateLockMs:     1800000,
			InstallApp:                "",
			Reply:                     "",
			SleepMs:                   3000,
			Thread:                    20,
			Queue:                     1000,
			RetryIntervalMs:           10000,
			RetryTimes:                1,
			HTTPTimeout:               45000,
			HTTPConnTimeout:           15000,
			HTTPSocketTimeout:         45000,
			HTTPDebug:                 false,
			IdleTimeout:               90,
			PluginExecTimeout:         defaultPluginExecTimeoutMs,
			SkillExtractRound:         10,
		},
		PIDFile: integrationDefaultPIDFile,
		LogFile: integrationDefaultLogFile,
	}
}

var integrationStartupConfigKeys = map[string]string{
	"port":                    "port",
	"host":                    "host",
	"app":                     "app",
	"appdir":                  "app-dir",
	"agentdir":                "agent-dir",
	"defaultdir":              "default-dir",
	"device":                  "device",
	"agentcache":              "agent-cache",
	"connectcache":            "connect-cache",
	"resourcesdir":            "resources-dir",
	"db":                      "db",
	"site":                    "site",
	"provider":                "provider",
	"connecttimeout":          "connect_timeout",
	"knowledgeupdateinterval": "knowledge_update_interval",
	"knowledgeupdatelock":     "knowledge_update_lock",
	"installapp":              "install_app",
	"reply":                   "reply",
	"sleep":                   "sleep",
	"thread":                  "thread",
	"queue":                   "queue",
	"retryinterval":           "retry_interval",
	"retrytimes":              "retry_times",
	"idletimeout":             "idle_timeout",
	"pluginexectimeout":       "plugin_exec_timeout",
	"skillextract":            "skill_extract",
	"skillcreate":             "skill_extract",
	"pidfile":                 "pid-file",
	"logfile":                 "log-file",
}

var integrationStartupHTTPConfigKeys = map[string]string{
	"httptimeout":        "http_timeout",
	"httpconnecttimeout": "http_connect_timeout",
	"httpsockettimeout":  "http_socket_timeout",
	"debug":              "http_debug",
}

func normalizeIntegrationStartupConfigKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, "-", "")
	key = strings.ReplaceAll(key, "_", "")
	key = strings.ReplaceAll(key, " ", "")
	return key
}

func integrationStartupConfigPath() string {
	resourcesDir := strings.TrimSpace(integrationResourcesDir())
	if resourcesDir == "" {
		return ""
	}
	return filepath.Join(resourcesDir, "config", "config.json")
}

func readIntegrationStartupConfigRaw() (map[string]interface{}, string, error) {
	path := integrationStartupConfigPath()
	if strings.TrimSpace(path) == "" {
		return map[string]interface{}{}, "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]interface{}{}, path, nil
		}
		return nil, path, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var raw map[string]interface{}
	if err := decoder.Decode(&raw); err != nil {
		return nil, path, fmt.Errorf("parse config/config.json: %w", err)
	}
	if raw == nil {
		raw = map[string]interface{}{}
	}
	return raw, path, nil
}

func readIntegrationStartupConfig() (map[string]interface{}, string, error) {
	raw, path, err := readIntegrationStartupConfigRaw()
	if err != nil {
		return nil, path, err
	}
	out := make(map[string]interface{})
	for key, value := range raw {
		canonical, ok := integrationStartupConfigKeys[normalizeIntegrationStartupConfigKey(key)]
		if !ok {
			if normalizeIntegrationStartupConfigKey(key) != "http" {
				continue
			}
		}
		if canonical != "" {
			out[canonical] = value
		}
	}
	if httpRaw, ok := raw["http"]; ok {
		if httpValues, ok := httpRaw.(map[string]interface{}); ok {
			for key, value := range httpValues {
				canonical, ok := integrationStartupHTTPConfigKeys[normalizeIntegrationStartupConfigKey(key)]
				if !ok {
					continue
				}
				out[canonical] = value
			}
		}
	}
	return out, path, nil
}

var integrationProviderConfigKeys = []string{
	"__url",
	"__model",
	"__model_fast",
	"__model_thinking",
	"__model_multi_input",
	"__model_multi_output",
}

func normalizeIntegrationProviderCatalog(raw interface{}) map[string]map[string]string {
	providers, ok := raw.(map[string]interface{})
	if !ok {
		return map[string]map[string]string{}
	}
	result := make(map[string]map[string]string, len(providers))
	for name, value := range providers {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		config := make(map[string]string)
		if source, ok := value.(map[string]interface{}); ok {
			for _, key := range integrationProviderConfigKeys {
				if field, ok := source[key].(string); ok {
					config[key] = field
				}
			}
		}
		result[name] = config
	}
	return result
}

func normalizeIntegrationAssociatedPlugins(raw interface{}) []string {
	values, ok := raw.([]interface{})
	if !ok {
		if stringValues, ok := raw.([]string); ok {
			values = make([]interface{}, len(stringValues))
			for index, value := range stringValues {
				values[index] = value
			}
		} else {
			return nil
		}
	}

	seen := make(map[string]struct{}, len(values))
	plugins := make([]string, 0, len(values))
	for _, value := range values {
		plugin, ok := value.(string)
		if !ok {
			continue
		}
		plugin = strings.TrimSpace(plugin)
		if plugin == "" {
			continue
		}
		key := strings.ToLower(plugin)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		plugins = append(plugins, plugin)
	}
	return plugins
}

func readIntegrationAssociatedPlugins() ([]string, error) {
	raw, _, err := readIntegrationStartupConfigRaw()
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}
	for key, value := range raw {
		if normalizeIntegrationStartupConfigKey(key) == "associated" {
			return normalizeIntegrationAssociatedPlugins(value), nil
		}
	}
	return nil, nil
}

func readIntegrationStartupConfigValue(key string) (string, bool) {
	values, _, err := readIntegrationStartupConfig()
	if err != nil || values == nil {
		return "", false
	}
	return integrationStartupConfigValue(values, key)
}

func integrationStartupConfigStringMap(values map[string]interface{}) map[string]string {
	if values == nil {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" || value == nil {
			continue
		}
		text := strings.TrimSpace(fmt.Sprint(value))
		if text == "" {
			continue
		}
		out[key] = text
	}
	return out
}

func formatCliGetHeartbeatError(host string, err error) string {
	host = strings.TrimSpace(host)
	if host == "" {
		host = "<empty>"
	}
	if err == nil {
		return fmt.Sprintf("cli-get: heartbeat error (host=%s)", host)
	}
	return fmt.Sprintf("cli-get: heartbeat error (host=%s): %v", host, err)
}

func integrationStartupConfigValue(values map[string]interface{}, key string) (string, bool) {
	if values == nil {
		return "", false
	}
	raw, ok := values[strings.TrimSpace(key)]
	if !ok || raw == nil {
		return "", false
	}
	if strings.TrimSpace(key) == "device" {
		value := integrationConfiguredDeviceValue(raw)
		return value, value != ""
	}
	value := strings.TrimSpace(fmt.Sprint(raw))
	if value == "" {
		return "", false
	}
	runtimeCfg := integrationStartupConfigStringMap(values)
	switch strings.TrimSpace(key) {
	case "default-dir", "site":
		value = integrationResolveResourceRuntimePath(value, runtimeCfg)
	case "agent-dir", "pid-file", "log-file":
		value = integrationResolveRuntimePath(value, runtimeCfg)
	}
	return value, true
}

func assignIntegrationStartupConfigInt(raw interface{}, name string, target *int) error {
	value, ok := sharedutil.ToInt64Value(raw)
	if !ok {
		return fmt.Errorf("%s must be integer", name)
	}
	*target = int(value)
	return nil
}

func assignIntegrationStartupConfigBool(raw interface{}, name string, target *bool) error {
	value, ok := sharedutil.ToBoolValue(raw)
	if !ok {
		return fmt.Errorf("%s must be boolean", name)
	}
	*target = value
	return nil
}

func applyIntegrationStartupConfig(opts *integrationStartupOptions, values map[string]interface{}) error {
	if opts == nil || values == nil {
		return nil
	}
	for key, raw := range values {
		switch key {
		case "port":
			if err := assignIntegrationStartupConfigInt(raw, key, &opts.Config.Port); err != nil {
				return err
			}
			if err := validateIntegrationServicePort(opts.Config.Port); err != nil {
				return err
			}
		case "host":
			if value := strings.TrimSpace(fmt.Sprint(raw)); value != "" {
				opts.Config.Host = value
			}
		case "agent-dir":
			if value := strings.TrimSpace(fmt.Sprint(raw)); value != "" {
				opts.Config.AgentDir = value
			}
		case "default-dir":
			if value := strings.TrimSpace(fmt.Sprint(raw)); value != "" {
				opts.Config.DefaultDir = value
			}
		case "device":
			opts.Config.Device = integrationConfiguredDeviceValue(raw)
		case "agent-cache":
			if err := assignIntegrationStartupConfigInt(raw, key, &opts.Config.AgentCacheMs); err != nil {
				return err
			}
		case "connect-cache":
			if err := assignIntegrationStartupConfigInt(raw, key, &opts.Config.ConnectCacheMs); err != nil {
				return err
			}
		case "site":
			if value := strings.TrimSpace(fmt.Sprint(raw)); value != "" {
				opts.Config.Site = value
			}
		case "provider":
			opts.Config.Provider = normalizeIntegrationProviderCatalog(raw)
		case "connect_timeout":
			if err := assignIntegrationStartupConfigInt(raw, key, &opts.Config.ConnectTimeoutMs); err != nil {
				return err
			}
		case "knowledge_update_interval":
			if err := assignIntegrationStartupConfigInt(raw, key, &opts.Config.KnowledgeUpdateIntervalMs); err != nil {
				return err
			}
		case "knowledge_update_lock":
			if err := assignIntegrationStartupConfigInt(raw, key, &opts.Config.KnowledgeUpdateLockMs); err != nil {
				return err
			}
		case "install_app":
			if value, ok := raw.(string); ok {
				opts.Config.InstallApp = strings.TrimSpace(value)
			}
		case "reply":
			opts.Config.Reply = strings.TrimSpace(fmt.Sprint(raw))
		case "sleep":
			if err := assignIntegrationStartupConfigInt(raw, key, &opts.Config.SleepMs); err != nil {
				return err
			}
		case "thread":
			if err := assignIntegrationStartupConfigInt(raw, key, &opts.Config.Thread); err != nil {
				return err
			}
		case "queue":
			if err := assignIntegrationStartupConfigInt(raw, key, &opts.Config.Queue); err != nil {
				return err
			}
		case "retry_interval":
			if err := assignIntegrationStartupConfigInt(raw, key, &opts.Config.RetryIntervalMs); err != nil {
				return err
			}
		case "retry_times":
			if err := assignIntegrationStartupConfigInt(raw, key, &opts.Config.RetryTimes); err != nil {
				return err
			}
		case "http_timeout":
			if err := assignIntegrationStartupConfigInt(raw, key, &opts.Config.HTTPTimeout); err != nil {
				return err
			}
		case "http_connect_timeout":
			if err := assignIntegrationStartupConfigInt(raw, key, &opts.Config.HTTPConnTimeout); err != nil {
				return err
			}
		case "http_socket_timeout":
			if err := assignIntegrationStartupConfigInt(raw, key, &opts.Config.HTTPSocketTimeout); err != nil {
				return err
			}
		case "http_debug":
			if err := assignIntegrationStartupConfigBool(raw, key, &opts.Config.HTTPDebug); err != nil {
				return err
			}
		case "idle_timeout":
			if err := assignIntegrationStartupConfigInt(raw, key, &opts.Config.IdleTimeout); err != nil {
				return err
			}
		case "plugin_exec_timeout":
			if err := assignIntegrationStartupConfigInt(raw, key, &opts.Config.PluginExecTimeout); err != nil {
				return err
			}
		case "skill_extract":
			if err := assignIntegrationStartupConfigInt(raw, key, &opts.Config.SkillExtractRound); err != nil {
				return err
			}
		case "pid-file":
			if value := strings.TrimSpace(fmt.Sprint(raw)); value != "" {
				opts.PIDFile = value
			}
		case "log-file":
			if value := strings.TrimSpace(fmt.Sprint(raw)); value != "" {
				opts.LogFile = value
			}
		}
	}
	return nil
}

func validateIntegrationServicePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("--port must be between 1 and 65535")
	}
	return nil
}

func validateOptionalIntegrationServicePort(port int) error {
	if port == 0 {
		return nil
	}
	return validateIntegrationServicePort(port)
}

func validateIntegrationTopLevelPortFlag(flags map[string]string) error {
	rawPort := strings.TrimSpace(flags["port"])
	if rawPort == "" {
		return nil
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil {
		return fmt.Errorf("--port must be between 1 and 65535")
	}
	return validateIntegrationServicePort(port)
}

func loadIntegrationStartupOptions() (integrationStartupOptions, string, error) {
	return loadIntegrationStartupOptionsWithPortOverride(false)
}

func loadIntegrationStartupOptionsWithPortOverride(portOverridden bool) (integrationStartupOptions, string, error) {
	opts := defaultIntegrationStartupOptions()
	values, path, err := readIntegrationStartupConfig()
	if err != nil {
		return opts, path, err
	}
	if portOverridden {
		values = maps.Clone(values)
		delete(values, "port")
	}
	if err := applyIntegrationStartupConfig(&opts, values); err != nil {
		return opts, path, err
	}
	return opts, path, nil
}

func bindIntegrationServeFlags(fs *flag.FlagSet, cfg *Config, pidFileFlag, logFileFlag, startupStatusFile *string, defaults integrationStartupOptions) {
	if fs == nil || cfg == nil {
		return
	}
	*cfg = defaults.Config
	if pidFileFlag != nil {
		*pidFileFlag = defaults.PIDFile
	}
	if logFileFlag != nil {
		*logFileFlag = defaults.LogFile
	}
	if startupStatusFile != nil {
		*startupStatusFile = ""
	}
	fs.IntVar(&cfg.Port, "port", defaults.Config.Port, "HTTP service port")
	fs.StringVar(&cfg.Host, "host", defaults.Config.Host, "upstream server URL")
	fs.StringVar(&cfg.AgentDir, "agent-dir", defaults.Config.AgentDir, "agent directory (auto-created if missing)")
	fs.StringVar(&cfg.DefaultDir, "default-dir", defaults.Config.DefaultDir, "default agent template directory (default: ./config)")
	fs.StringVar(&cfg.Device, "device", defaults.Config.Device, "device ID (auto-generated if empty)")
	fs.IntVar(&cfg.AgentCacheMs, "agent-cache", defaults.Config.AgentCacheMs, "agent metadata cache TTL in ms")
	fs.IntVar(&cfg.ConnectCacheMs, "connect-cache", defaults.Config.ConnectCacheMs, "connect metadata cache TTL in ms")
	fs.StringVar(&cfg.Site, "site", defaults.Config.Site, "static site directory (default: ./site)")
	fs.IntVar(&cfg.ConnectTimeoutMs, "connect_timeout", defaults.Config.ConnectTimeoutMs, "upstream connect timeout in ms")
	fs.IntVar(&cfg.KnowledgeUpdateIntervalMs, "knowledge_update_interval", defaults.Config.KnowledgeUpdateIntervalMs, "knowledge lastUpdate refresh interval in ms")
	fs.IntVar(&cfg.KnowledgeUpdateLockMs, "knowledge_update_lock", defaults.Config.KnowledgeUpdateLockMs, "knowledge refresh request lock window in ms")
	fs.StringVar(&cfg.InstallApp, "install_app", defaults.Config.InstallApp, "extra install_app entries, comma separated")
	fs.StringVar(&cfg.Reply, "reply", defaults.Config.Reply, "reply text sent once per plugin when scheduled details are created")
	fs.IntVar(&cfg.SleepMs, "sleep", defaults.Config.SleepMs, "cli-get heartbeat error sleep in ms")
	fs.IntVar(&cfg.Thread, "thread", defaults.Config.Thread, "cli-get worker pool size")
	fs.IntVar(&cfg.Queue, "queue", defaults.Config.Queue, "cli-get local task queue size")
	fs.IntVar(&cfg.RetryIntervalMs, "retry_interval", defaults.Config.RetryIntervalMs, "cli-get publish retry interval in ms")
	fs.IntVar(&cfg.RetryTimes, "retry_times", defaults.Config.RetryTimes, "cli-get publish retry count after first failure")
	fs.IntVar(&cfg.HTTPTimeout, "http_timeout", defaults.Config.HTTPTimeout, "cli-get HTTP total timeout in ms")
	fs.IntVar(&cfg.HTTPConnTimeout, "http_connect_timeout", defaults.Config.HTTPConnTimeout, "cli-get HTTP connect timeout in ms")
	fs.IntVar(&cfg.HTTPSocketTimeout, "http_socket_timeout", defaults.Config.HTTPSocketTimeout, "cli-get HTTP read timeout in ms")
	fs.IntVar(&cfg.IdleTimeout, "idle_timeout", defaults.Config.IdleTimeout, "cli-get idle timeout in seconds")
	fs.IntVar(&cfg.PluginExecTimeout, "plugin_exec_timeout", defaults.Config.PluginExecTimeout, "plugin exec timeout in ms")
	if pidFileFlag != nil {
		fs.StringVar(pidFileFlag, "pid-file", defaults.PIDFile, "integration pid file path")
	}
	if logFileFlag != nil {
		fs.StringVar(logFileFlag, "log-file", defaults.LogFile, "integration log file path")
	}
	if startupStatusFile != nil {
		fs.StringVar(startupStatusFile, "startup-status-file", "", "integration startup status file path")
	}
}

func integrationFlagValue(fs *flag.FlagSet, name string) (string, bool) {
	if fs == nil {
		return "", false
	}
	provided := false
	fs.Visit(func(item *flag.Flag) {
		if item.Name == name {
			provided = true
		}
	})
	if !provided {
		return "", false
	}
	item := fs.Lookup(name)
	if item == nil {
		return "", false
	}
	return strings.TrimSpace(item.Value.String()), true
}

func writeRuntimeConfig(data map[string]interface{}) {
	if runtimeDir := strings.TrimSpace(integrationBundleRuntimeBaseDir()); runtimeDir != "" {
		data["app-dir"] = runtimeDir
		if resourcesDir := strings.TrimSpace(integrationResourcesDir()); resourcesDir != "" {
			data["resources-dir"] = resourcesDir
		}
	}
	if raw, ok := data["default-dir"].(string); ok {
		data["default-dir"] = integrationResourcePathArg(raw)
	}
	if raw, ok := data["site"].(string); ok {
		data["site"] = integrationResourcePathArg(raw)
	}
	appDir := ""
	if raw, ok := data["app-dir"].(string); ok {
		appDir = strings.TrimSpace(raw)
	}
	if appDir == "" {
		appDir = integrationResourcesDir()
	}
	raw, path, err := readIntegrationStartupConfigRaw()
	if err != nil {
		log.Printf("config/config.json: read failed: %v", err)
		return
	}
	if raw == nil {
		raw = map[string]interface{}{}
	}
	for _, key := range []string{"http_timeout", "http_connect_timeout", "http_socket_timeout", "http_debug"} {
		delete(raw, key)
	}
	httpSection := map[string]interface{}{}
	if existing, ok := raw["http"].(map[string]interface{}); ok && existing != nil {
		for key, value := range existing {
			httpSection[key] = value
		}
	}
	for key, value := range data {
		switch key {
		case "http_timeout":
			httpSection["http_timeout"] = value
		case "http_connect_timeout":
			httpSection["http_connect_timeout"] = value
		case "http_socket_timeout":
			httpSection["http_socket_timeout"] = value
		case "http_debug":
			httpSection["debug"] = value
		default:
			raw[key] = value
		}
	}
	if len(httpSection) > 0 {
		raw["http"] = httpSection
	}
	if strings.TrimSpace(path) == "" {
		path = integrationStartupConfigPath()
	}
	if strings.TrimSpace(path) == "" {
		log.Printf("config/config.json: resolve path failed")
		return
	}
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		log.Printf("config/config.json: marshal failed: %v", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Printf("config/config.json: prepare dir failed: %v", err)
		return
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		log.Printf("config/config.json: write failed: %v", err)
	}
}

func removeIntegrationRuntimeConfigFiles(runtimeCfg map[string]string) ([]string, error) {
	return nil, nil
}

func copyDefaultAgentTemplate(defaultDir, dst string) error {
	resolvedDefaultDir, err := resolveServeDefaultDir(defaultDir)
	if err != nil {
		return err
	}
	info, err := os.Stat(resolvedDefaultDir)
	if err != nil {
		return fmt.Errorf("default-dir unavailable: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("default-dir is not a directory: %s", resolvedDefaultDir)
	}
	if err := copyDirRecursiveFiltered(resolvedDefaultDir, dst, shouldCopyDefaultAgentTemplateEntry); err != nil {
		_ = os.RemoveAll(dst)
		return fmt.Errorf("failed to copy default-dir: %w", err)
	}
	if err := writeEmptyAgentConfig(dst); err != nil {
		_ = os.RemoveAll(dst)
		return fmt.Errorf("failed to initialize agent config.json: %w", err)
	}
	return nil
}

func ensureDefaultAgentScaffold(agentDir, defaultDir string) error {
	entries, err := os.ReadDir(agentDir)
	if err != nil {
		return fmt.Errorf("read agent-dir: %w", err)
	}
	if len(entries) != 0 {
		return nil
	}
	defAgentDir := filepath.Join(agentDir, "DEF_AGENT")
	if err := copyDefaultAgentTemplate(defaultDir, defAgentDir); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(defAgentDir, "skills"), 0o755); err != nil {
		return fmt.Errorf("create default agent scaffold: %w", err)
	}
	return nil
}

func resolveServeAgentDir(agentDir, defaultDir string) (string, error) {
	agentDir = strings.TrimSpace(agentDir)
	if agentDir == "" {
		agentDir = integrationDefaultAgentDir()
	}
	absPath := integrationResolveAppPath(agentDir)
	if absPath == "" {
		return "", fmt.Errorf("cannot resolve agent-dir")
	}
	info, err := os.Stat(absPath)
	if err == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("agent-dir is not a directory: %s", absPath)
		}
		if err := ensureDefaultAgentScaffold(absPath, defaultDir); err != nil {
			return "", err
		}
		return absPath, nil
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat agent-dir: %w", err)
	}
	if err := os.MkdirAll(absPath, 0o755); err != nil {
		return "", fmt.Errorf("create agent-dir: %w", err)
	}
	if err := ensureDefaultAgentScaffold(absPath, defaultDir); err != nil {
		return "", err
	}
	return absPath, nil
}

func resolveServeDefaultDir(defaultDir string) (string, error) {
	defaultDir = strings.TrimSpace(defaultDir)
	if defaultDir == "" {
		defaultDir = "config"
	}
	absPath := integrationResolveResourcePath(defaultDir)
	if absPath == "" {
		return "", fmt.Errorf("cannot resolve default-dir")
	}
	return absPath, nil
}

func resolveServeSiteDir(siteDir string) (string, error) {
	siteDir = strings.TrimSpace(siteDir)
	if siteDir == "" {
		siteDir = "site"
	}
	absPath := integrationResolveResourcePath(siteDir)
	if absPath == "" {
		return "", fmt.Errorf("cannot resolve site")
	}
	return absPath, nil
}

func readRuntimeConfig() map[string]string {
	values, _, err := readIntegrationStartupConfig()
	if err != nil || values == nil {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = strings.TrimSpace(fmt.Sprint(value))
	}
	return out
}

func integrationRuntimeConfigPath() string {
	return integrationStartupConfigPath()
}

func readIntegrationRuntimeConfigRaw() (map[string]string, string) {
	return readRuntimeConfig(), integrationStartupConfigPath()
}

func readIntegrationRuntimeConfigFile(path string) (map[string]string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, false
	}
	out := make(map[string]string)
	for k, v := range raw {
		out[k] = strings.TrimSpace(fmt.Sprint(v))
	}
	return out, true
}

func integrationRuntimeConfigCandidates() []string {
	path := integrationStartupConfigPath()
	if strings.TrimSpace(path) == "" {
		return nil
	}
	return []string{path}
}

func resolveIntegrationDBPath() string {
	if runtimeDir := integrationBundleRuntimeBaseDir(); runtimeDir != "" {
		return filepath.Join(runtimeDir, "data")
	}
	if dbPath, ok := readIntegrationStartupConfigValue("db"); ok {
		return dbPath
	}
	if appDir, ok := readIntegrationStartupConfigValue("app-dir"); ok {
		return filepath.Join(appDir, "data")
	}
	if appPath, ok := readIntegrationStartupConfigValue("app"); ok {
		appDir := filepath.Dir(appPath)
		if strings.TrimSpace(appDir) != "" {
			return filepath.Join(appDir, "data")
		}
	}
	if exe, err := integrationExecutableFn(); err == nil && strings.TrimSpace(exe) != "" {
		if strings.HasSuffix(strings.TrimSpace(exe), ".test") {
			return "data"
		}
		return filepath.Join(filepath.Dir(exe), "data")
	}
	return "data"
}

func parseCLIArgs(args []string, cycle *int, rawTime, agentID, chatID, model, content *string, thinking, routerDisable *bool, cronExpr, agentDir *string) error {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		key := strings.TrimPrefix(arg, "--")
		value := ""
		hasValue := false
		if parts := strings.SplitN(key, "=", 2); len(parts) == 2 {
			key = parts[0]
			value = parts[1]
			hasValue = true
		} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			value = args[i+1]
			hasValue = true
			i++
		}

		switch key {
		case "cycle":
			if cycle == nil {
				continue
			}
			if !hasValue {
				return fmt.Errorf("cycle requires a value")
			}
			if _, err := fmt.Sscanf(value, "%d", cycle); err != nil {
				return fmt.Errorf("invalid cycle: %s", value)
			}
		case "time", "rawTime":
			if rawTime != nil && hasValue {
				*rawTime = value
			}
		case "agent", "agentId":
			if agentID != nil && hasValue {
				*agentID = value
			}
		case "agent-dir":
			if agentDir != nil && hasValue {
				*agentDir = value
			}
		case "chat", "chatId":
			if chatID != nil && hasValue {
				*chatID = value
			}
		case "model":
			if model != nil && hasValue {
				*model = value
			}
		case "content":
			if content != nil && hasValue {
				*content = value
			}
		case "cron":
			if cronExpr != nil && hasValue {
				*cronExpr = value
			}
		case "thinking":
			if thinking == nil {
				continue
			}
			if !hasValue {
				*thinking = true
				continue
			}
			parsed, err := parseCLIBoolValue("thinking", value)
			if err != nil {
				return err
			}
			*thinking = parsed
		case "router_disable":
			if routerDisable == nil {
				continue
			}
			if !hasValue {
				*routerDisable = true
				continue
			}
			parsed, err := parseCLIBoolValue("router_disable", value)
			if err != nil {
				return err
			}
			*routerDisable = parsed
		}
	}
	return nil
}

func resolveIntegrationAgentDir(agentDir string) (string, error) {
	agentDir = strings.TrimSpace(agentDir)
	if agentDir == "" {
		if value, ok := readIntegrationStartupConfigValue("agent-dir"); ok {
			agentDir = value
		}
	}
	if agentDir == "" {
		agentDir = strings.TrimSpace(os.Getenv("AGENT_DIR"))
	}
	if agentDir == "" {
		agentDir = integrationDefaultAgentDir()
	}
	if agentDir == "" {
		return "", fmt.Errorf("agent-dir is required for agent validation")
	}
	absPath, err := filepath.Abs(agentDir)
	if err != nil {
		return "", fmt.Errorf("cannot resolve agent-dir: %w", err)
	}
	return absPath, nil
}

func resolveIntegrationDeviceID(deviceID string) string {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID != "" {
		return deviceID
	}
	if value, ok := readIntegrationStartupConfigValue("device"); ok {
		deviceID = value
	}
	return deviceID
}

func parseCLIBoolValue(flagName, value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "1", "true", "yes", "y", "on":
		return true, nil
	case "0", "false", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid %s value: %s", flagName, value)
	}
}

func validateCLICreateArgs(cycle int, rawTime, agentID, model, content string) error {
	if cycle < 0 || cycle > 5 {
		return fmt.Errorf("cycle must be one of 0,1,2,3,4,5")
	}
	if strings.TrimSpace(rawTime) == "" {
		return fmt.Errorf("rawTime is required, format: yyyy-MM-dd hh:mm")
	}
	if _, err := time.ParseInLocation("2006-01-02 15:04", rawTime, time.Local); err != nil {
		return fmt.Errorf("invalid rawTime format: %w", err)
	}
	if strings.TrimSpace(agentID) == "" {
		return fmt.Errorf("agentId is required")
	}
	if strings.TrimSpace(model) == "" {
		return fmt.Errorf("model is required")
	}
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content is required")
	}
	return nil
}

func validateCLICreateCronArgs(cronExpr, agentID, model, content string) error {
	if strings.TrimSpace(cronExpr) == "" {
		return fmt.Errorf("cron is required")
	}
	if strings.TrimSpace(agentID) == "" {
		return fmt.Errorf("agentId is required")
	}
	if strings.TrimSpace(model) == "" {
		return fmt.Errorf("model is required")
	}
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content is required")
	}
	return nil
}

func printCronCreateHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration cron create [options]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --content TEXT                 任务内容，必填")
	fmt.Println("  --schema JSON                  可选；写入 response_schema，并在执行时透传为 response_format.json_schema")
	fmt.Println("  --model NAME                   模型名称，必填")
	fmt.Println("  --thinking[=true|false]        是否深度思考；也支持 --thinking true")
	fmt.Println("  --router_disable[=true|false]  是否关闭 router，默认 true")
	fmt.Println("  --rawTime 'YYYY-MM-DD HH:MM'   首次开始时间，必填；也支持 --time")
	fmt.Println("  --cycle INT                    0=仅一次, 1=工作日, 2=自然日, 3=每小时, 4=每15分钟, 5=每30分钟")
	fmt.Println("  --chatId ID                    会话 ID（CHAT_ID），可选；也支持 --chat")
	fmt.Println("  --agent ID                     绑定的 AgentId，必填；也支持 --agentId")
	fmt.Println("  --agent-dir DIR                Agent 根目录；也可用 AGENT_DIR 或当前目录 ./agent")
	fmt.Println("")
	fmt.Println("Notes:")
	fmt.Println("  create 在未启动 HTTP 服务时也会先初始化共享 sqlite，并复用与 HTTP 一致的校验逻辑。")
	fmt.Println("  执行前会校验 Agent 是否存在、模型是否已在 Proxy 注册 Token。")
	fmt.Println("  cycle=0 的一次性任务会把 --schema 直接继承到首条 task_detail。")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  integration cron create --content '每15分钟检查一次上游接口健康' --model OpenAI --thinking true --router_disable false --rawTime '2026-05-03 10:00' --cycle 4 --chatId chat-001 --agent demo-agent")
	fmt.Println("  integration cron create --content '整理日报' --schema '{\"type\":\"object\",\"properties\":{\"summary\":{\"type\":\"string\"}},\"required\":[\"summary\"]}' --model OpenAI --rawTime '2026-05-08 09:00' --cycle 0 --agent demo-agent")
	fmt.Println("  integration cron create --content '明早整理日报' --model OpenAI --router_disable --rawTime '2026-05-08 09:00' --cycle 0 --agent demo-agent")
}

func printCronCreateCronHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration cron create-cron [options]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --content TEXT                 任务内容，必填")
	fmt.Println("  --schema JSON                  可选；写入 response_schema，并在后续每条明细执行时透传为 response_format.json_schema")
	fmt.Println("  --model NAME                   模型名称，必填")
	fmt.Println("  --thinking[=true|false]        是否深度思考；也支持 --thinking true")
	fmt.Println("  --router_disable[=true|false]  是否关闭 router，默认 true")
	fmt.Println("  --cron EXPR                    Cron 表达式，必填")
	fmt.Println("  --chatId ID                    会话 ID（CHAT_ID），可选；也支持 --chat")
	fmt.Println("  --agent ID                     绑定的 AgentId，必填；也支持 --agentId")
	fmt.Println("  --agent-dir DIR                Agent 根目录；也可用 AGENT_DIR 或当前目录 ./agent")
	fmt.Println("")
	fmt.Println("Notes:")
	fmt.Println("  create-cron 在未启动 HTTP 服务时也会先初始化共享 sqlite，并复用与 HTTP 一致的校验逻辑。")
	fmt.Println("  执行前会校验 Agent 是否存在、模型是否已在 Proxy 注册 Token。")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  integration cron create-cron --content '工作日中午整理异常日志' --model OpenAI --thinking true --router_disable false --cron '10 12 * * 1-5' --chatId chat-001 --agent demo-agent")
	fmt.Println("  integration cron create-cron --content '提取结构化日报' --schema '{\"type\":\"object\",\"properties\":{\"todo\":{\"type\":\"array\"}},\"required\":[\"todo\"]}' --model OpenAI --cron '0 18 * * 1-5' --agent demo-agent")
	fmt.Println("  integration cron create-cron --content '每小时检查一次 API 健康' --model OpenAI --router_disable --cron '0 * * * *' --agent demo-agent")
}

func printCronFindMetaHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration cron find-meta [options]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --agent=ID / --agentId ID      按 AgentId 精确匹配")
	fmt.Println("  --chat=ID / --chatId ID        按 ChatId 精确匹配")
	fmt.Println("  --model=NAME                   按模型精确匹配")
	fmt.Println("  --content=TEXT                 按内容模糊匹配")
	fmt.Println("  --cycle=INT                    按执行周期精确匹配")
	fmt.Println("  --time='YYYY-MM-DD HH:MM'      查询指定开始执行时间点")
	fmt.Println("  --date='YYYY-MM-DD'            查询指定开始执行日期")
	fmt.Println("  --from='YYYY-MM-DD HH:MM'      开始执行时间范围起点")
	fmt.Println("  --to='YYYY-MM-DD HH:MM'        开始执行时间范围终点")
	fmt.Println("")
	fmt.Println("Notes:")
	fmt.Println("  未指定的维度表示全部匹配。")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  integration cron find-meta --content '每15分钟检查一次上游接口健康' --model OpenAI --chatId chat-001")
	fmt.Println("  integration cron find-meta --agent demo-agent --cycle 4 --from '2026-05-03 00:00' --to '2026-05-04 00:00'")
}

func printCronFindDetailHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration cron find-detail [options]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --meta=ID / --metaId ID        按元数据 ID 精确匹配，支持 1 或 cron_1")
	fmt.Println("  --agent=ID / --agentId ID      按 AgentId 精确匹配")
	fmt.Println("  --chat=ID / --chatId ID        按 ChatId 精确匹配")
	fmt.Println("  --model=NAME                   按模型精确匹配")
	fmt.Println("  --content=TEXT                 按内容模糊匹配")
	fmt.Println("  --cycle=INT                    按执行周期精确匹配")
	fmt.Println("  --time='YYYY-MM-DD HH:MM'      查询指定执行时间点")
	fmt.Println("  --date='YYYY-MM-DD'            查询指定执行日期")
	fmt.Println("  --from='YYYY-MM-DD HH:MM'      执行时间范围起点")
	fmt.Println("  --to='YYYY-MM-DD HH:MM'        执行时间范围终点")
	fmt.Println("")
	fmt.Println("Notes:")
	fmt.Println("  未指定的维度表示全部匹配。")
	fmt.Println("  未指定时间条件时，默认优先查询当前时间之后的任务明细；已保留的已完成明细也会返回。")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  integration cron find-detail --metaId cron_1 --content '每15分钟检查一次上游接口健康' --model OpenAI --cycle 4 --chatId chat-001")
	fmt.Println("  integration cron find-detail --agent demo-agent --date 2026-05-03")
}

func printCronDeleteMetaHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration cron delete-meta [options]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --id=ID                        按元数据 ID 删除，支持 1 / cron_1 / meta_1")
	fmt.Println("  --agent=ID / --agentId ID      按 AgentId 精确匹配")
	fmt.Println("  --chat=ID / --chatId ID        按 ChatId 精确匹配")
	fmt.Println("  --model=NAME                   按模型精确匹配")
	fmt.Println("  --content=TEXT                 按内容模糊匹配")
	fmt.Println("  --cycle=INT                    按执行周期精确匹配")
	fmt.Println("  --time='YYYY-MM-DD HH:MM'      删除指定开始执行时间点的元数据")
	fmt.Println("  --date='YYYY-MM-DD'            删除指定开始执行日期的元数据")
	fmt.Println("  --from='YYYY-MM-DD HH:MM'      开始执行时间范围起点")
	fmt.Println("  --to='YYYY-MM-DD HH:MM'        开始执行时间范围终点")
	fmt.Println("")
	fmt.Println("Notes:")
	fmt.Println("  delete-meta 会同时删除匹配元数据下的全部任务明细。")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  integration cron delete-meta --id meta_1")
	fmt.Println("  integration cron delete-meta --agent demo-agent --cycle 4 --date 2026-05-03")
}

func printCronDeleteDetailHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration cron delete-detail [options]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --detailId=ID                  按明细 ID 删除，支持 1 / detail_1")
	fmt.Println("  --meta=ID / --metaId ID        删除指定元数据下全部匹配明细，支持 1 / cron_1 / meta_1")
	fmt.Println("  --agent=ID / --agentId ID      按 AgentId 精确匹配")
	fmt.Println("  --chat=ID / --chatId ID        按 ChatId 精确匹配")
	fmt.Println("  --model=NAME                   按模型精确匹配")
	fmt.Println("  --content=TEXT                 按内容模糊匹配")
	fmt.Println("  --cycle=INT                    按执行周期精确匹配")
	fmt.Println("  --time='YYYY-MM-DD HH:MM'      删除指定执行时间点的明细")
	fmt.Println("  --date='YYYY-MM-DD'            删除指定执行日期的明细")
	fmt.Println("  --from='YYYY-MM-DD HH:MM'      执行时间范围起点")
	fmt.Println("  --to='YYYY-MM-DD HH:MM'        执行时间范围终点")
	fmt.Println("")
	fmt.Println("Notes:")
	fmt.Println("  未传时间条件时，不会默认限制为未来数据；删除会按你给出的条件全量匹配。")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  integration cron delete-detail --metaId meta_1")
	fmt.Println("  integration cron delete-detail --detailId detail_1")
}

func printCLIHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration [serve options]")
	fmt.Println("  integration serve [serve options]")
	fmt.Println("  integration start [serve options]")
	fmt.Println("  integration stop [--pid-file PATH]")
	fmt.Println("  integration restart [serve options]")
	fmt.Println("  integration splash [--logo PATH] [--duration 5s] [--async]")
	fmt.Println("  integration cron create [options]")
	fmt.Println("  integration cron create-cron [options]")
	fmt.Println("  integration cron find-meta [options]")
	fmt.Println("  integration cron find-detail [options]")
	fmt.Println("  integration cron delete-meta [options]")
	fmt.Println("  integration cron delete-detail [options]")
	fmt.Println("  integration knowledge update-time --timestamp N --agentId ID")
	fmt.Println("  integration host get [--addr URL]")
	fmt.Println("  integration host set --value URL [--addr URL]")
	fmt.Println("  integration host reset [--addr URL]")
	fmt.Println("  integration standalone get [--addr URL]")
	fmt.Println("  integration standalone set --value true|false [--addr URL]")
	fmt.Println("  integration standalone reset [--addr URL]")
	fmt.Println("  integration agent export --agent ID [--output PATH] [--agent-dir DIR]")
	fmt.Println("  integration agent import --input PATH [--agent-dir DIR]")
	fmt.Println("  integration agent copy --source ID --target ID [--agent-dir DIR]")
	fmt.Println("  integration sandbox --agentId ID --chatId ID [--sandbox off|filepick|net|filepick_net] [--dir DIR]")
	fmt.Println("  integration token [--provider MODEL]")
	fmt.Println("  integration token get [--n N] [--start 'YYYY-MM-DD HH:MM:SS'] [--close 'YYYY-MM-DD HH:MM:SS'] [--agentId ID]")
	fmt.Println("  integration token --agentId ID --model MODEL --function NAME --total N [--thinking N --input N --cache N --timestamp MS]")
	fmt.Println("  integration plugins exec --key KEY --command 'SUBCOMMAND [ARGS...]' [plugin flags...]")
	fmt.Println("  integration plugins status --key KEY [--pid-file PATH]")
	fmt.Println("  integration plugins sync-bundled [--check]")
	fmt.Println("  integration notify [--title TEXT] [--message TEXT]")
	fmt.Println("  integration log-round --agent A --chat chat-001 --round 3")
	fmt.Println("  integration skills-warning [--refresh]")
	fmt.Println("  integration file-last-update [options]")
	fmt.Println("  integration backup-clean [options]")
	fmt.Println("  integration connect <subcommand> [options]")
	fmt.Println("  integration help")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  serve              Run integration in foreground")
	fmt.Println("  start              Start integration in background and write pid file")
	fmt.Println("  stop               Stop the background integration process")
	fmt.Println("  restart            Restart the background integration process")
	fmt.Println("  splash             Show the macOS launch splash logo")
	fmt.Println("  cron create        Create task metadata from cycle + first start time")
	fmt.Println("  cron create-cron   Create task metadata from explicit cron expression")
	fmt.Println("  cron find-meta     Query task metadata")
	fmt.Println("  cron find-detail   Query task details")
	fmt.Println("  cron delete-meta   Delete task metadata and its details")
	fmt.Println("  cron delete-detail Delete task details")
	fmt.Println("  knowledge         Knowledge runtime helper commands")
	fmt.Println("  host              Read or update the in-memory upstream host of a running integration service")
	fmt.Println("  standalone        Read or update the in-memory localhost-only mode of a running integration service")
	fmt.Println("  agent             Export, import, or copy managed Agent files")
	fmt.Println("  sandbox           Read or update sandbox mode for one AgentId + ChatId")
	fmt.Println("  token             Print saved model tokens, query token consume details, or record token consume details")
	fmt.Println("  connect            Manage third-party connect metadata, requests and responses")
	fmt.Println("  plugins            Plugin helper commands")
	fmt.Println("  notify             Trigger one local macOS notification")
	fmt.Println("  log-round          Export the latest N rounds of chat and cli logs to agent tmp")
	fmt.Println("  skills-warning     Read current SKILL parse warnings from shared sqlite")
	fmt.Println("  file-last-update   Print milliseconds since the target file was last updated")
	fmt.Println("  backup-clean       Move stale User/Soul backup files into bak and delete stale bak files")
	fmt.Println("  help               Show this help")
	fmt.Println("")
	fmt.Println("Help shortcuts:")
	fmt.Println("  integration splash --help")
	fmt.Println("  integration cron create --help")
	fmt.Println("  integration cron create-cron --help")
	fmt.Println("  integration cron find-meta --help")
	fmt.Println("  integration cron find-detail --help")
	fmt.Println("  integration cron delete-meta --help")
	fmt.Println("  integration cron delete-detail --help")
	fmt.Println("  integration knowledge --help")
	fmt.Println("  integration sandbox --help")
	fmt.Println("  integration token --help")
	fmt.Println("  integration token get --help")
	fmt.Println("  integration plugins --help")
	fmt.Println("  integration connect --help")
	fmt.Println("  integration api --help")
	fmt.Println("  integration service --help")
	fmt.Println("  integration notify --help")
	fmt.Println("  integration skills-warning --refresh")
	fmt.Println("  integration file-last-update --agent A --file USER.md")
	fmt.Println("  integration file-last-update --file /abs/path/to/file.md")
	fmt.Println("  integration backup-clean --agent-dir ./agent")
	fmt.Println("")
	fmt.Println("Serve options:")
	fmt.Println("  --agent-dir=DIR    Agent 根目录；macOS 默认 app container 下的 deepright/agent，WSL 默认 ~/deepright/agent，不存在时自动创建")
	fmt.Println("  --default-dir=DIR  新建 Agent 默认模板目录，默认当前目录下的 ./config")
	fmt.Println("  --port=INT         HTTP 服务端口；未指定时读取 config/config.json.port，默认 8080")
	fmt.Println("  --host=URL         上游服务地址，默认 " + defaultUpstreamHost)
	fmt.Println("  --device=ID        设备 ID，可选")
	fmt.Println("  --agent-cache=MS   Agent 元数据缓存时长（毫秒）")
	fmt.Println("  --connect-cache=MS Connect 读缓存时长（毫秒）")
	fmt.Println("  --site=DIR         静态站点目录，默认当前目录下的 ./site")
	fmt.Println("  --connect_timeout=MS            上游连接超时（毫秒）")
	fmt.Println("  --knowledge_update_interval=MS  knowledge.lastUpdate 透传阈值，默认 7200000")
	fmt.Println("  --knowledge_update_lock=MS      knowledge 更新申请锁窗口，默认 1800000")
	fmt.Println("  --install_app=CSV               额外 install_app 条目，逗号分隔，接口返回会自动去重合并")
	fmt.Println("  --pid-file=PATH    后台进程 pid 文件，默认当前应用目录下的 integration.pid（WSL 为 ~/deepright/integration.pid）")
	fmt.Println("  --log-file=PATH    后台进程日志文件，默认当前应用目录下的 integration.log（WSL 为 ~/deepright/integration.log）")
	fmt.Println("")
	fmt.Println("Backup-clean options:")
	fmt.Println("  --agent-dir DIR               Agent 根目录；未传时会复用主应用 config/config.json / AGENT_DIR / 默认目录")
	fmt.Println("  --archive-after DURATION      备份文件在工作目录停留多久后移入 bak，默认 24h")
	fmt.Println("  --delete-after DURATION       bak 目录中的文件保留多久后删除，默认 72h")
	fmt.Println("")
	fmt.Println("Splash options:")
	fmt.Println("  --logo=PATH        启动 Logo 图片路径，默认自动读取当前 integration 资源目录下的 site/icon.png")
	fmt.Println("  --duration=5s      动画总时长，默认 5 秒")
	fmt.Println("  --async            异步拉起浮层后立即返回")
	fmt.Println("")
	fmt.Println("Cron create options:")
	fmt.Println("  --content TEXT                 任务内容，必填")
	fmt.Println("  --model NAME                  模型名称，必填")
	fmt.Println("  --thinking[=true|false]       是否深度思考；也支持 --thinking true")
	fmt.Println("  --router_disable[=true|false] 是否关闭 router，默认 true")
	fmt.Println("  --rawTime 'YYYY-MM-DD HH:MM'  首次开始时间，必填；也支持 --time")
	fmt.Println("  --cycle INT                   0=仅一次, 1=工作日, 2=自然日, 3=每小时, 4=每15分钟, 5=每30分钟")
	fmt.Println("  --chatId ID                   会话 ID（CHAT_ID），可选；也支持 --chat")
	fmt.Println("  --agent ID                    绑定的 AgentId，必填；也支持 --agentId")
	fmt.Println("  --agent-dir DIR               Agent 根目录；也可用 AGENT_DIR 或当前目录 ./agent")
	fmt.Println("")
	fmt.Println("Cron create-cron options:")
	fmt.Println("  --content TEXT                 任务内容，必填")
	fmt.Println("  --model NAME                  模型名称，必填")
	fmt.Println("  --thinking[=true|false]       是否深度思考；也支持 --thinking true")
	fmt.Println("  --router_disable[=true|false] 是否关闭 router，默认 true")
	fmt.Println("  --cron EXPR                   Cron 表达式，必填")
	fmt.Println("  --chatId ID                   会话 ID（CHAT_ID），可选；也支持 --chat")
	fmt.Println("  --agent ID                    绑定的 AgentId，必填；也支持 --agentId")
	fmt.Println("  --agent-dir DIR               Agent 根目录；也可用 AGENT_DIR 或当前目录 ./agent")
	fmt.Println("")
	fmt.Println("Cron find-meta options:")
	fmt.Println("  --agent=ID / --agentId ID      按 AgentId 精确匹配")
	fmt.Println("  --chat=ID / --chatId ID        按 ChatId 精确匹配")
	fmt.Println("  --model=NAME                   按模型精确匹配")
	fmt.Println("  --content=TEXT                 按内容模糊匹配")
	fmt.Println("  --cycle=INT                    按执行周期精确匹配")
	fmt.Println("  --time='YYYY-MM-DD HH:MM'      查询指定开始执行时间点")
	fmt.Println("  --date='YYYY-MM-DD'            查询指定开始执行日期")
	fmt.Println("  --from='YYYY-MM-DD HH:MM'      开始执行时间范围起点")
	fmt.Println("  --to='YYYY-MM-DD HH:MM'        开始执行时间范围终点")
	fmt.Println("")
	fmt.Println("Cron find-detail options:")
	fmt.Println("  --meta=ID / --metaId ID        按元数据 ID 精确匹配，支持 1 或 cron_1")
	fmt.Println("  --agent=ID / --agentId ID      按 AgentId 精确匹配")
	fmt.Println("  --chat=ID / --chatId ID        按 ChatId 精确匹配")
	fmt.Println("  --model=NAME                   按模型精确匹配")
	fmt.Println("  --content=TEXT                 按内容模糊匹配")
	fmt.Println("  --cycle=INT                    按执行周期精确匹配")
	fmt.Println("  --time='YYYY-MM-DD HH:MM'      查询指定执行时间点")
	fmt.Println("  --date='YYYY-MM-DD'            查询指定执行日期")
	fmt.Println("  --from='YYYY-MM-DD HH:MM'      执行时间范围起点")
	fmt.Println("  --to='YYYY-MM-DD HH:MM'        执行时间范围终点")
	fmt.Println("  未指定时间条件时，默认优先查询当前时间之后的任务明细；已保留的已完成明细也会返回")
	fmt.Println("")
	fmt.Println("Cron delete-meta options:")
	fmt.Println("  --id=ID                        按元数据 ID 删除，支持 1 / cron_1 / meta_1")
	fmt.Println("  其余过滤条件与 find-meta 相同")
	fmt.Println("")
	fmt.Println("Cron delete-detail options:")
	fmt.Println("  --detailId=ID                  按明细 ID 删除，支持 1 / detail_1")
	fmt.Println("  --meta=ID / --metaId ID        删除指定元数据下全部匹配明细")
	fmt.Println("  其余过滤条件与 find-detail 相同；删除时未传时间不会默认限制未来数据")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  integration")
	fmt.Println("  integration start --agent-dir ./agent --site ./site --port 8080")
	fmt.Println("  integration stop")
	fmt.Println("  integration restart --port 8080")
	fmt.Println("  integration --agent-dir ./agent --site ./site --port 8080")
	fmt.Println("  integration cron create --agent-dir ../agent/test-case --content '每15分钟检查一次上游接口健康' --model OpenAI --thinking true --router_disable false --rawTime '2026-05-03 10:00' --cycle 4 --chatId chat-001 --agent demo-agent")
	fmt.Println("  integration cron create-cron --agent-dir ../agent/test-case --content '工作日中午整理异常日志' --model OpenAI --router_disable --cron '10 12 * * 1-5' --chatId chat-001 --agent demo-agent")
	fmt.Println("  integration cron find-meta --content '每15分钟检查一次上游接口健康' --model OpenAI --chatId chat-001")
	fmt.Println("  integration cron find-detail --metaId cron_1 --cycle 4 --chatId chat-001")
	fmt.Println("  integration cron delete-meta --id meta_1")
	fmt.Println("  integration cron delete-detail --detailId detail_1")
	fmt.Println("  integration knowledge update-time --timestamp 1715337600000 --agentId demo-agent")
	fmt.Println("  integration host get")
	fmt.Println("  integration host set --value https://staging.deepright.cn")
	fmt.Println("  integration host reset")
	fmt.Println("  integration standalone get")
	fmt.Println("  integration standalone set --value true")
	fmt.Println("  integration standalone reset")
	fmt.Println("  integration message-insert add --agentId demo --chatId chat-001 --tid 1718966400000 --message 'HELLO'")
	fmt.Println("  integration agent export --agent DEF_AGENT --output ./DEF_AGENT.zip")
	fmt.Println("  integration agent import --input ./DEF_AGENT.zip")
	fmt.Println("  integration connect meta-create --key feishu --meta '{\"token\":\"abc\"}' --callback ignored --agent A --model OpenAI")
	fmt.Println("  integration connect meta-update --key feishu --meta '{\"token\":\"def\"}' --callback ignored --agent A --model OpenAI")
	fmt.Println("  integration connect meta-get --key feishu")
	fmt.Println("  integration connect meta-list")
	fmt.Println("  integration plugins start --key feishu --connect-bin ./integration")
	fmt.Println("  integration plugins exec --key browser --command 'instance init' --agentId A --chatId chat-001")
	fmt.Println("  integration plugins status --key feishu")
	fmt.Println("  integration plugins sync-bundled --check")
	fmt.Println("  integration log-round --agent A --chat chat-001 --round 3")
}

func printConnectHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration connect <subcommand> [options]")
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  meta-create")
	fmt.Println("  meta-update")
	fmt.Println("  meta-delete")
	fmt.Println("  meta-get")
	fmt.Println("  meta-list")
	fmt.Println("  list-plugins")
	fmt.Println("  add-request")
	fmt.Println("  request-list")
	fmt.Println("  add-response")
	fmt.Println("  response-list")
	fmt.Println("")
	fmt.Println("Common options:")
	fmt.Println("  --key NAME")
	fmt.Println("  --meta JSON")
	fmt.Println("  --stream true|false")
	fmt.Println("  --callback VALUE (stored callback is fixed to plugins/<plugin-key>)")
	fmt.Println("  --agent ID / --agentId ID")
	fmt.Println("  --chat ID / --chatId ID")
	fmt.Println("  --model NAME")
	fmt.Println("  --thinking true|false")
	fmt.Println("  --request TEXT")
	fmt.Println("  --schema JSON")
	fmt.Println("  --response TEXT")
	fmt.Println("  --request-id N")
	fmt.Println("  --artifacts PATHS")
	fmt.Println("  --after-id N")
	fmt.Println("  --limit N")
	fmt.Println("  --connect-cache MS")
	fmt.Println("")
	fmt.Println("Notes:")
	fmt.Println("  推荐统一使用 integration connect <subcommand>，例如 integration connect meta-get。")
	fmt.Println("  integration list-meta 仍作为兼容入口保留，并映射到新的 integration connect meta-list。")
	fmt.Println("  --callback 仅作为输入占位，实际保存值固定为 plugins/<plugin-key>。")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  integration connect meta-create --key feishu --meta '{\"appId\":\"cli-app\",\"appSecret\":\"cli-secret\"}' --callback ignored --agent A --chatId chat-001 --model OpenAI")
	fmt.Println("  integration connect meta-update --key feishu --meta '{\"appId\":\"new\",\"appSecret\":\"secret\"}' --callback ignored --agent A --chatId chat-002 --model OpenAI")
	fmt.Println("  integration connect meta-get --key feishu")
	fmt.Println("  integration connect meta-list")
	fmt.Println("  integration connect add-request --key feishu --externalId ext-1 --content 'HELLO WORLD' --original '{\"text\":\"HELLO WORLD\"}'")
	fmt.Println("  integration connect add-request --key feishu --content '提取结构化消息' --schema '{\"type\":\"object\",\"properties\":{\"title\":{\"type\":\"string\"}},\"required\":[\"title\"]}'")
	fmt.Println("  integration connect response-list --key feishu --request-id 1")
}

func printKnowledgeHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration knowledge update-time --timestamp N --agentId ID")
	fmt.Println("  integration knowledge last-update [--agentId ID]")
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  update-time       Update knowledge last_update in shared sqlite")
	fmt.Println("  last-update       Print formatted knowledge last_update from shared sqlite")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --timestamp N     Unix timestamp in milliseconds; required by update-time")
	fmt.Println("  --agentId ID      Agent 名称；update-time 必填，last-update 选填；也兼容 --agent-id")
	fmt.Println("")
	fmt.Println("Notes:")
	fmt.Println("  Knowledge 数据仍由共享 knowledge 模块维护，integration 仅做 CLI 收口。")
	fmt.Println("  默认使用当前应用启动目录对应的共享 data sqlite 与 knowledge 目录。")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  integration knowledge update-time --timestamp 1715337600000 --agentId demo-agent")
	fmt.Println("  integration knowledge last-update --agentId demo-agent")
}

func printHostHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration host get [--addr URL]")
	fmt.Println("  integration host set --value URL [--addr URL]")
	fmt.Println("  integration host reset [--addr URL]")
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  get               Print the current runtime host of the running integration service")
	fmt.Println("  set               Override the runtime host in memory until the service restarts")
	fmt.Println("  reset             Clear the runtime override and fall back to the startup host")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --value URL       New upstream host URL, required by set")
	fmt.Println("  --addr URL        Integration HTTP base address, default http://127.0.0.1:<runtime port>")
	fmt.Println("  --port INT        Integration HTTP 端口；仅支持 8080")
	fmt.Println("")
	fmt.Println("Notes:")
	fmt.Println("  该命令通过本机 integration 的 /api/host 接口工作，仅修改当前运行进程内存，不会写入 config.json。")
	fmt.Println("  integration 重启后，会恢复为 --host / config/config.json / 内置默认值决定的启动 Host。")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  integration host get")
	fmt.Println("  integration host set --value https://staging.deepright.cn")
	fmt.Println("  integration host reset")
}

func printStandaloneHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration standalone get [--addr URL]")
	fmt.Println("  integration standalone set --value true|false [--addr URL]")
	fmt.Println("  integration standalone reset [--addr URL]")
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  get               Print the current runtime standalone mode of the running integration service")
	fmt.Println("  set               Enable or disable localhost-only protection until the service restarts")
	fmt.Println("  reset             Clear the runtime override and fall back to the startup standalone mode")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --value BOOL      New standalone value, required by set")
	fmt.Println("  --addr URL        Integration HTTP base address, default http://127.0.0.1:<runtime port>")
	fmt.Println("  --port INT        Integration HTTP 端口；仅支持 8080")
	fmt.Println("")
	fmt.Println("Notes:")
	fmt.Println("  该命令通过本机 integration 的 /api/standalone 接口工作，仅修改当前运行进程内存，不会写入 config.json。")
	fmt.Println("  standalone=true 后，这个 --port 端口上的所有 HTTP 服务都只允许 localhost / 127.0.0.1 / ::1 访问，包括静态页面。")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  integration standalone get")
	fmt.Println("  integration standalone set --value true")
	fmt.Println("  integration standalone set --value false")
	fmt.Println("  integration standalone reset")
}

func printAgentHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration agent export --agent ID [--output PATH] [--agent-dir DIR]")
	fmt.Println("  integration agent import --input PATH [--agent-dir DIR]")
	fmt.Println("  integration agent copy --source ID --target ID [--agent-dir DIR]")
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  export            Export one Agent directory to a zip file")
	fmt.Println("  import            Import one Agent zip or directory into agent-dir")
	fmt.Println("  copy              Copy managed app/data/skills/SOUL.md/USER.md/Knowledge.md/knowledge.md and knowledge to target Agent")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --agent ID        AgentId to export; also accepts --agentId")
	fmt.Println("  --output PATH     Export zip output path; defaults to ./<agent>.zip")
	fmt.Println("  --input PATH      Zip file or directory to import")
	fmt.Println("  --source ID       Source AgentId to copy from; also accepts --sourceAgentId")
	fmt.Println("  --target ID       Existing target AgentId to copy into; also accepts --targetAgentId")
	fmt.Println("  --agent-dir DIR   Agent 根目录；未传时会复用主应用 config/config.json / AGENT_DIR / 默认目录")
	fmt.Println("")
	fmt.Println("Notes:")
	fmt.Println("  export 生成的 zip 会保留顶层 Agent 目录，并自动过滤该 Agent 一级目录中的 chrome*、data、tmp。")
	fmt.Println("  import 在发现同名 Agent 已存在时会直接拒绝，避免覆盖现有目录。")
	fmt.Println("  copy 只同步 app、data、skills、SOUL.md、USER.md、Knowledge.md、knowledge.md，以及同级 knowledge/<targetAgentId>；不会覆盖 target 的 config.json。")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  integration agent export --agent DEF_AGENT --output ./DEF_AGENT.zip")
	fmt.Println("  integration agent import --input ./DEF_AGENT.zip")
	fmt.Println("  integration agent import --input /path/to/DEF_AGENT")
	fmt.Println("  integration agent copy --source DEF_AGENT --target DEF_AGENT_COPY")
}

var integrationConnectSubcommands = map[string]struct{}{
	"meta-create":   {},
	"meta-update":   {},
	"meta-delete":   {},
	"meta-get":      {},
	"meta-list":     {},
	"list-meta":     {},
	"list-plugins":  {},
	"add-request":   {},
	"request-list":  {},
	"add-response":  {},
	"response-list": {},
}

func isIntegrationConnectSubcommand(command string) bool {
	_, ok := integrationConnectSubcommands[strings.TrimSpace(command)]
	return ok
}

func newIntegrationConnectService(agentDir string) (*connectsvc.Service, error) {
	resolvedAgentDir, err := resolveIntegrationAgentDir(agentDir)
	if err != nil {
		return nil, err
	}
	cacheMS := 10000
	if raw, ok := readIntegrationStartupConfigValue("connect-cache"); ok {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			cacheMS = parsed
		}
	}
	return connectsvc.NewService(connectsvc.Options{
		DBPath:   resolveIntegrationDBPath(),
		AgentDir: resolvedAgentDir,
		CacheTTL: time.Duration(cacheMS) * time.Millisecond,
	})
}

func runIntegrationConnectCLI(args []string) {
	if len(args) == 0 || hasHelpFlag(args) {
		printConnectHelp()
		return
	}
	command := normalizeIntegrationConnectCommand(args[0])
	flags, err := connectsvc.ParseFlags(args[1:])
	if err != nil {
		log.Fatal(err)
	}

	if command == "list-plugins" {
		items, err := listLocalPlugins()
		if err != nil {
			log.Fatal(err)
		}
		out, _ := json.MarshalIndent(items, "", "  ")
		fmt.Println(string(out))
		return
	}

	if cronDB == nil {
		initCronDB()
	}
	svc, err := newIntegrationConnectService(firstNonEmpty(flags["agent-dir"]))
	if err != nil {
		log.Fatal(err)
	}
	defer svc.Close()
	code := connectsvc.RunCLIWithService(command, flags, svc, os.Stdout, os.Stderr)
	if code != 0 {
		os.Exit(code)
	}
}

func normalizeIntegrationConnectCommand(command string) string {
	switch strings.TrimSpace(command) {
	case "meta-list", "list-meta":
		// integration/proxy 统一把 meta-list 暴露为“已配置插件配置视图”，兼容旧的 list-meta 入口。
		return "list-meta"
	default:
		return strings.TrimSpace(command)
	}
}

func runIntegrationSplashCLI(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("integration splash", flag.ContinueOnError)
	fs.SetOutput(stderr)
	defaultCfg, defaultErr := integrationDefaultLaunchSplashConfig()
	defaultLogo := ""
	defaultDuration := 5 * time.Second
	if defaultErr == nil {
		defaultLogo = defaultCfg.LogoPath
		defaultDuration = defaultCfg.Duration
	}
	logoPath := fs.String("logo", defaultLogo, "path to splash logo image")
	duration := fs.Duration("duration", defaultDuration, "splash animation duration")
	async := fs.Bool("async", false, "start splash asynchronously and return immediately")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: integration splash [--logo PATH] [--duration 5s] [--async]")
		fmt.Fprintln(fs.Output(), "")
		fmt.Fprintln(fs.Output(), "Show the centered macOS launch splash logo animation used by integration.app.")
		fmt.Fprintln(fs.Output(), "")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}
	cfg := launchsplash.Config{
		LogoPath: strings.TrimSpace(*logoPath),
		Duration: *duration,
	}
	if cfg.LogoPath == "" && defaultErr != nil {
		fmt.Fprintln(stderr, defaultErr.Error())
		return 1
	}
	var err error
	if *async {
		err = launchsplash.Start(cfg)
	} else {
		err = launchsplash.Run(cfg)
	}
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return 0
}

func runIntegrationKnowledgeCLI(args []string) {
	if len(args) == 0 || hasHelpFlag(args) {
		printKnowledgeHelp()
		return
	}

	switch strings.TrimSpace(args[0]) {
	case "update-time":
		code := runIntegrationKnowledgeUpdateTimeCLI(args[1:], os.Stdout, os.Stderr)
		if code != 0 {
			os.Exit(code)
		}
	case "last-update":
		code := runIntegrationKnowledgeLastUpdateCLI(args[1:], os.Stdout, os.Stderr)
		if code != 0 {
			os.Exit(code)
		}
	default:
		fmt.Println("Unknown knowledge command:", args[0])
		printKnowledgeHelp()
		os.Exit(1)
	}
}

func runIntegrationKnowledgeUpdateTimeCLI(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("integration knowledge update-time", flag.ContinueOnError)
	fs.SetOutput(stderr)
	timestamp := fs.String("timestamp", "", "unix timestamp in milliseconds")
	agentID := fs.String("agentId", "", "agent id")
	agentIDLegacy := fs.String("agent-id", "", "agent id")
	fs.Usage = func() { printKnowledgeHelp() }
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(stderr, "error: positional TIMESTAMP is not supported; use --timestamp")
		return 1
	}
	resolvedTimestamp := strings.TrimSpace(*timestamp)
	if resolvedTimestamp == "" {
		fmt.Fprintln(stderr, "error: --timestamp is required")
		return 1
	}
	resolvedAgentID := strings.TrimSpace(*agentID)
	if resolvedAgentID == "" {
		resolvedAgentID = strings.TrimSpace(*agentIDLegacy)
	}
	if resolvedAgentID == "" {
		fmt.Fprintln(stderr, "error: --agentId is required")
		return 1
	}
	parsedTimestamp, err := strconv.ParseInt(resolvedTimestamp, 10, 64)
	if err != nil {
		fmt.Fprintf(stderr, "error: parse timestamp: %v\n", err)
		return 1
	}
	db, err := knowledgecore.OpenSharedDB(integrationAppDir())
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if err := knowledgecore.SetLastUpdateForAgent(db, resolvedAgentID, parsedTimestamp); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	data, err := knowledgecore.MarshalStateForAgent(integrationAppDir(), resolvedAgentID)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, string(data))
	return 0
}

func runIntegrationKnowledgeLastUpdateCLI(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("integration knowledge last-update", flag.ContinueOnError)
	fs.SetOutput(stderr)
	agentID := fs.String("agentId", "", "agent id")
	agentIDLegacy := fs.String("agent-id", "", "agent id")
	fs.Usage = func() { printKnowledgeHelp() }
	if err := fs.Parse(args); err != nil {
		return 1
	}
	resolvedAgentID := strings.TrimSpace(*agentID)
	if resolvedAgentID == "" {
		resolvedAgentID = strings.TrimSpace(*agentIDLegacy)
	}
	text, err := integrationknowledge.LoadLastUpdateText(integrationAppDir(), resolvedAgentID, time.Local)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, text)
	return 0
}

func defaultIntegrationHostAPIBase() string {
	return fmt.Sprintf("http://127.0.0.1:%d", resolveIntegrationLaunchPort(nil))
}

func resolveIntegrationHostAPIBase(addr string, port int) string {
	if value := strings.TrimSpace(addr); value != "" {
		return strings.TrimRight(value, "/")
	}
	if port > 0 {
		return fmt.Sprintf("http://127.0.0.1:%d", port)
	}
	return defaultIntegrationHostAPIBase()
}

func runIntegrationHostCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || hasHelpFlag(args) {
		printHostHelp()
		return 0
	}

	command := strings.TrimSpace(args[0])
	fs := flag.NewFlagSet("integration host", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addr := fs.String("addr", "", "integration HTTP base address")
	port := fs.Int("port", 0, "integration HTTP port")
	value := fs.String("value", "", "runtime upstream host URL")
	fs.Usage = func() { printHostHelp() }
	if err := fs.Parse(args[1:]); err != nil {
		return 1
	}
	if err := validateOptionalIntegrationServicePort(*port); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	client := runtimehost.NewClient(resolveIntegrationHostAPIBase(*addr, *port), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var (
		snapshot runtimehost.Snapshot
		err      error
	)
	switch command {
	case "get":
		snapshot, err = client.Get(ctx)
	case "set":
		host := strings.TrimSpace(*value)
		if host == "" && fs.NArg() > 0 {
			host = strings.TrimSpace(fs.Arg(0))
		}
		if host == "" {
			fmt.Fprintln(stderr, "host set requires --value URL")
			return 1
		}
		snapshot, err = client.Set(ctx, host)
	case "reset":
		snapshot, err = client.Reset(ctx)
	default:
		fmt.Fprintf(stderr, "unknown host command: %s\n", command)
		printHostHelp()
		return 1
	}
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	out, marshalErr := json.MarshalIndent(map[string]any{
		"host":        snapshot.Host,
		"startupHost": snapshot.StartupHost,
		"overridden":  snapshot.Overridden,
		"runtimeOnly": snapshot.RuntimeOnly,
	}, "", "  ")
	if marshalErr != nil {
		fmt.Fprintln(stderr, marshalErr.Error())
		return 1
	}
	fmt.Fprintln(stdout, string(out))
	return 0
}

func runIntegrationStandaloneCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || hasHelpFlag(args) {
		printStandaloneHelp()
		return 0
	}

	command := strings.TrimSpace(args[0])
	fs := flag.NewFlagSet("integration standalone", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addr := fs.String("addr", "", "integration HTTP base address")
	port := fs.Int("port", 0, "integration HTTP port")
	value := fs.String("value", "", "runtime standalone value")
	fs.Usage = func() { printStandaloneHelp() }
	if err := fs.Parse(args[1:]); err != nil {
		return 1
	}
	if err := validateOptionalIntegrationServicePort(*port); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	client := integrationstandalone.NewClient(resolveIntegrationHostAPIBase(*addr, *port), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var (
		snapshot integrationstandalone.Snapshot
		err      error
	)
	switch command {
	case "get":
		snapshot, err = client.Get(ctx)
	case "set":
		raw := strings.TrimSpace(*value)
		if raw == "" && fs.NArg() > 0 {
			raw = strings.TrimSpace(fs.Arg(0))
		}
		if raw == "" {
			fmt.Fprintln(stderr, "standalone set requires --value true|false")
			return 1
		}
		enabled, parseErr := parseStandaloneUpdateValue(raw)
		if parseErr != nil {
			fmt.Fprintln(stderr, parseErr.Error())
			return 1
		}
		snapshot, err = client.Set(ctx, enabled)
	case "reset":
		snapshot, err = client.Reset(ctx)
	default:
		fmt.Fprintf(stderr, "unknown standalone command: %s\n", command)
		printStandaloneHelp()
		return 1
	}
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	out, marshalErr := json.MarshalIndent(map[string]any{
		"enabled":        snapshot.Enabled,
		"startupEnabled": snapshot.StartupEnabled,
		"overridden":     snapshot.Overridden,
		"runtimeOnly":    snapshot.RuntimeOnly,
	}, "", "  ")
	if marshalErr != nil {
		fmt.Fprintln(stderr, marshalErr.Error())
		return 1
	}
	fmt.Fprintln(stdout, string(out))
	return 0
}

func runIntegrationAgentCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || hasHelpFlag(args) {
		printAgentHelp()
		return 0
	}

	command := strings.TrimSpace(args[0])
	fs := flag.NewFlagSet("integration agent", flag.ContinueOnError)
	fs.SetOutput(stderr)
	agent := fs.String("agent", "", "agent id")
	agentAlias := fs.String("agentId", "", "agent id")
	output := fs.String("output", "", "export zip output path")
	input := fs.String("input", "", "zip file or directory to import")
	source := fs.String("source", "", "source agent id")
	sourceAlias := fs.String("sourceAgentId", "", "source agent id")
	target := fs.String("target", "", "target agent id")
	targetAlias := fs.String("targetAgentId", "", "target agent id")
	agentDir := fs.String("agent-dir", "", "agent root dir")
	fs.Usage = func() { printAgentHelp() }
	if err := fs.Parse(args[1:]); err != nil {
		return 1
	}

	resolvedAgentDir, err := resolveIntegrationAgentDir(*agentDir)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if err := os.MkdirAll(resolvedAgentDir, 0755); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	var result map[string]any
	switch command {
	case "export":
		agentID := firstNonEmpty(strings.TrimSpace(*agent), strings.TrimSpace(*agentAlias))
		if agentID == "" && fs.NArg() > 0 {
			agentID = strings.TrimSpace(fs.Arg(0))
		}
		if err := integrationagentarchive.ValidateAgentID(agentID); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		outputPath := strings.TrimSpace(*output)
		if outputPath == "" {
			outputPath = agentID + ".zip"
		}
		file, err := os.Create(outputPath)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		exportErr := integrationagentarchive.Export(resolvedAgentDir, agentID, file)
		closeErr := file.Close()
		if exportErr != nil {
			_ = os.Remove(outputPath)
			fmt.Fprintln(stderr, exportErr.Error())
			return 1
		}
		if closeErr != nil {
			_ = os.Remove(outputPath)
			fmt.Fprintln(stderr, closeErr.Error())
			return 1
		}
		absOutputPath, err := filepath.Abs(outputPath)
		if err != nil {
			absOutputPath = filepath.Clean(outputPath)
		}
		result = map[string]any{
			"agentId": agentID,
			"output":  absOutputPath,
		}
	case "import":
		inputPath := strings.TrimSpace(*input)
		if inputPath == "" && fs.NArg() > 0 {
			inputPath = strings.TrimSpace(fs.Arg(0))
		}
		if inputPath == "" {
			fmt.Fprintln(stderr, "agent import requires --input PATH")
			return 1
		}
		info, err := os.Stat(inputPath)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		var imported integrationagentarchive.Result
		if info.IsDir() {
			imported, err = integrationagentarchive.ImportDirectory(resolvedAgentDir, inputPath)
		} else {
			if !strings.EqualFold(filepath.Ext(inputPath), ".zip") {
				fmt.Fprintln(stderr, "agent import only supports zip files or directories")
				return 1
			}
			imported, err = integrationagentarchive.ImportZipFile(resolvedAgentDir, inputPath)
		}
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		flushAgentCache()
		result = map[string]any{
			"agentId": imported.AgentID,
			"path":    imported.AgentPath,
			"source":  imported.Source,
		}
	case "copy":
		sourceAgentID := firstNonEmpty(strings.TrimSpace(*source), strings.TrimSpace(*sourceAlias))
		targetAgentID := firstNonEmpty(strings.TrimSpace(*target), strings.TrimSpace(*targetAlias))
		if sourceAgentID == "" && fs.NArg() > 0 {
			sourceAgentID = strings.TrimSpace(fs.Arg(0))
		}
		if targetAgentID == "" && fs.NArg() > 1 {
			targetAgentID = strings.TrimSpace(fs.Arg(1))
		}
		knowledgeRoot, err := integrationKnowledgeRoot(resolvedAgentDir)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		copied, err := integrationagentcopy.Copy(resolvedAgentDir, knowledgeRoot, sourceAgentID, targetAgentID)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		flushAgentCache()
		result = map[string]any{
			"sourceAgentId": copied.SourceAgentID,
			"targetAgentId": copied.TargetAgentID,
			"path":          copied.AgentPath,
			"knowledgePath": copied.KnowledgePath,
			"copied":        copied.Copied,
		}
	default:
		fmt.Fprintf(stderr, "unknown agent command: %s\n", command)
		printAgentHelp()
		return 1
	}

	out, marshalErr := json.MarshalIndent(result, "", "  ")
	if marshalErr != nil {
		fmt.Fprintln(stderr, marshalErr.Error())
		return 1
	}
	fmt.Fprintln(stdout, string(out))
	return 0
}

func requireLocalPluginManagement(w http.ResponseWriter, r *http.Request) bool {
	if isLocalManagementRequest(r) {
		return true
	}
	connectsvc.WritePluginActionError(w, http.StatusForbidden, fmt.Errorf("only localhost/127.0.0.1/::1 requests can manage plugins; remote access is limited to viewing plugin list, status, and logs"))
	return false
}

func sanitizeRemotePluginMetaItems(items []connectsvc.PluginMetaInfo) []connectsvc.PluginMetaInfo {
	if len(items) == 0 {
		return nil
	}
	sanitized := make([]connectsvc.PluginMetaInfo, 0, len(items))
	for _, item := range items {
		sanitized = append(sanitized, connectsvc.PluginMetaInfo{
			Key:           item.Key,
			Name:          item.Name,
			Param:         nil,
			Scope:         append([]string{}, item.Scope...),
			Meta:          map[string]any{},
			Stream:        false,
			Callback:      "",
			AgentID:       "",
			ChatID:        "",
			Model:         "",
			Thinking:      false,
			RouterDisable: true,
			CreatedAt:     "",
			UpdatedAt:     "",
		})
	}
	return sanitized
}

func handlePluginsMeta(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		svc, err := connectsvc.NewService(connectsvc.Options{
			DBPath:   resolveIntegrationDBPath(),
			AgentDir: firstNonEmpty(cfg.AgentDir),
			CacheTTL: time.Duration(cfg.ConnectCacheMs) * time.Millisecond,
		})
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": err.Error()})
			return
		}
		defer svc.Close()

		items, err := listLocalPluginMeta(svc)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": err.Error()})
			return
		}
		if !isLocalManagementRequest(r) {
			items = sanitizeRemotePluginMetaItems(items)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": 0, "data": items})
	}
}

func handlePluginsConfig(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !requireLocalPluginManagement(w, r) {
			return
		}

		input, err := parseIntegrationPluginConfigRequest(r)
		if err != nil {
			connectsvc.WritePluginActionError(w, http.StatusBadRequest, err)
			return
		}

		svc, err := connectsvc.NewService(connectsvc.Options{
			DBPath:   resolveIntegrationDBPath(),
			AgentDir: firstNonEmpty(cfg.AgentDir),
			CacheTTL: time.Duration(cfg.ConnectCacheMs) * time.Millisecond,
		})
		if err != nil {
			log.Printf("plugins config error: %v", err)
			connectsvc.WritePluginActionError(w, http.StatusBadRequest, err)
			return
		}
		defer svc.Close()

		item, err := connectsvc.UpsertPluginConfigWithService(svc, input)
		if err != nil {
			log.Printf("plugins config error: %v", err)
			connectsvc.WritePluginActionError(w, http.StatusBadRequest, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": 0, "data": item})
	}
}

func handlePluginsLog(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		plugin := strings.TrimSpace(r.URL.Query().Get("key"))
		if plugin == "" {
			http.Error(w, "key is required", http.StatusBadRequest)
			return
		}

		logPath, err := pluginlog.ResolveLogPath(plugin)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		lastLines := pluginlog.DefaultLastLines
		if raw := strings.TrimSpace(r.URL.Query().Get("last")); raw != "" {
			value, convErr := strconv.Atoi(raw)
			if convErr != nil || value < 0 {
				http.Error(w, "last must be a non-negative integer", http.StatusBadRequest)
				return
			}
			lastLines = value
		}

		if err := pluginlog.ServeSSE(r.Context(), w, logPath, pluginlog.Options{
			LastLines:     lastLines,
			WaitForCreate: 2 * time.Second,
		}); err != nil && r.Context().Err() == nil {
			log.Printf("plugins log stream ended: plugin=%s log=%s err=%v", plugin, logPath, err)
		}
	}
}

func handlePluginsStatus(_ *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		name, flags, err := parsePluginActionRequest(r)
		if err != nil {
			connectsvc.WritePluginActionError(w, http.StatusBadRequest, err)
			return
		}

		result, err := runConnectPluginStatus(name, flags)
		if err != nil {
			log.Printf("plugins status error: %v", err)
			connectsvc.WritePluginActionError(w, connectsvc.PluginActionStatusCode(err), err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": 0, "data": buildPluginStatusResponse(result)})
	}
}

func handlePluginsStart(_ *Config) http.HandlerFunc {
	return handlePluginAction("start")
}

func handlePluginsStop(_ *Config) http.HandlerFunc {
	return handlePluginAction("stop")
}

func handlePluginsExec(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !requireLocalPluginManagement(w, r) {
			return
		}

		name, command, flags, err := connectsvc.ParsePluginExecRequest(r)
		if err != nil {
			connectsvc.WritePluginActionError(w, http.StatusBadRequest, err)
			return
		}
		ensureIntegrationPluginConnectBinForExec(name, command, flags)

		result, err := connectsvc.RunPluginExecCommand(name, command, flags, resolveIntegrationPluginExecTimeout(cfg))
		if err != nil {
			log.Printf("plugins exec error: %v", err)
			connectsvc.WritePluginActionError(w, connectsvc.PluginActionStatusCode(err), err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": 0, "data": buildPluginActionResponse(result)})
	}
}

func resolveIntegrationPluginExecTimeout(cfg *Config) time.Duration {
	if cfg != nil && cfg.PluginExecTimeout > 0 {
		return time.Duration(cfg.PluginExecTimeout) * time.Millisecond
	}
	if raw, ok := readIntegrationStartupConfigValue("plugin_exec_timeout"); ok {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			return time.Duration(value) * time.Millisecond
		}
	}
	return time.Duration(defaultPluginExecTimeoutMs) * time.Millisecond
}

func handlePluginAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !requireLocalPluginManagement(w, r) {
			return
		}

		name, flags, err := parsePluginActionRequest(r)
		if err != nil {
			connectsvc.WritePluginActionError(w, http.StatusBadRequest, err)
			return
		}
		if action == "start" {
			ensureIntegrationPluginConnectBin(flags)
		}

		result, err := runConnectPluginAction(name, action, flags)
		if err != nil {
			if action == "start" {
				if logErr := appendPluginActionFailureLog(name, action, err); logErr != nil {
					log.Printf("plugins %s append log error: plugin=%s err=%v", action, name, logErr)
				}
			}
			log.Printf("plugins %s error: %v", action, err)
			connectsvc.WritePluginActionError(w, connectsvc.PluginActionStatusCode(err), err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": 0, "data": buildPluginActionResponse(result)})
	}
}

func ensureIntegrationPluginConnectBin(flags map[string]string) {
	exe, err := integrationExecutableFn()
	if err != nil {
		exe = ""
	}
	connectsvc.EnsureConnectBinary(flags, exe)
}

func ensureIntegrationPluginConnectBinForExec(name, command string, flags map[string]string) {
	if !shouldEnsureIntegrationPluginConnectBinForExec(name, command) {
		return
	}
	ensureIntegrationPluginConnectBin(flags)
}

func shouldEnsureIntegrationPluginConnectBinForExec(name, command string) bool {
	if !strings.EqualFold(strings.TrimSpace(name), "browser") {
		return true
	}
	args, err := connectsvc.SplitPluginExecCommand(command)
	if err != nil || len(args) == 0 {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(args[0]), "start")
}

func appendPluginActionFailureLog(plugin, action string, actionErr error) error {
	if strings.TrimSpace(plugin) == "" || actionErr == nil {
		return nil
	}

	logPath, err := pluginlog.ResolveLogPath(plugin)
	if err != nil {
		return err
	}
	if err := ensureParentDir(logPath); err != nil {
		return err
	}

	writer, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = fmt.Fprintf(
		writer,
		"%s，用参数：插件=%s；操作=%s，做了：插件操作失败，原因：%s\n",
		time.Now().Format(time.RFC3339),
		strings.TrimSpace(plugin),
		strings.TrimSpace(action),
		strings.TrimSpace(actionErr.Error()),
	)
	return err
}

func buildPluginStatusResponse(result *connectsvc.PluginStatus) pluginStatusResponse {
	return connectsvc.BuildPluginStatusResponse(result)
}

func writeConnectError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": err.Error()})
}

var integrationExecutableFn = os.Executable

func resolveRouterDisableFormValue(form url.Values, fallback bool) bool {
	return connectsvc.ResolveRouterDisableFormValue(form, fallback)
}

func parseIntegrationPluginConfigRequest(r *http.Request) (connectsvc.MetaInput, error) {
	return connectsvc.ParsePluginConfigRequest(r)
}

func parsePluginActionRequest(r *http.Request) (string, map[string]string, error) {
	return connectsvc.ParsePluginActionRequest(r)
}

type integrationShutdownController struct {
	cancel      context.CancelFunc
	delay       time.Duration
	logWriter   io.Writer
	stopPlugins func(io.Writer) []string
	once        sync.Once
}

func newIntegrationShutdownController(cancel context.CancelFunc, logWriter io.Writer) *integrationShutdownController {
	return &integrationShutdownController{
		cancel:      cancel,
		delay:       integrationShutdownDelay,
		logWriter:   logWriter,
		stopPlugins: stopIntegrationPlugins,
	}
}

func (c *integrationShutdownController) schedule(reason string) bool {
	if c == nil || c.cancel == nil {
		return false
	}

	scheduled := false
	c.once.Do(func() {
		scheduled = true
		delay := c.delay
		if delay < 0 {
			delay = 0
		}
		reason = strings.TrimSpace(reason)
		if reason == "" {
			reason = "api shutdown"
		}
		integrationStopLog(c.logWriter, "shutdown scheduled delay=%s reason=%s", delay, reason)
		go func() {
			if delay > 0 {
				timer := time.NewTimer(delay)
				defer timer.Stop()
				<-timer.C
			}
			integrationStopLog(c.logWriter, "shutdown started reason=%s", reason)
			if c.stopPlugins != nil {
				c.stopPlugins(c.logWriter)
			}
			c.cancel()
		}()
	})
	return scheduled
}

func handleShutdown(controller *integrationShutdownController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !isLocalManagementRequest(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":  1,
				"content": "only localhost/127.0.0.1/::1 requests can shut down integration",
			})
			return
		}
		if controller == nil {
			connectsvc.WritePluginActionError(w, http.StatusInternalServerError, fmt.Errorf("shutdown controller is not initialized"))
			return
		}

		scheduled := controller.schedule("api shutdown")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": 0,
			"data": map[string]any{
				"scheduled": scheduled,
				"delayMs":   controller.delay.Milliseconds(),
			},
		})
	}
}

func runIntegrationForeground(args []string, stderr io.Writer) int {
	flags := parseTopLevelFlags(args)
	startupOpts, configPath, err := loadIntegrationStartupOptionsWithPortOverride(strings.TrimSpace(flags["port"]) != "")
	if err != nil {
		if strings.TrimSpace(configPath) != "" {
			fmt.Fprintf(stderr, "%s: %v\n", configPath, err)
		} else {
			fmt.Fprintf(stderr, "%v\n", err)
		}
		return 1
	}

	fs := flag.NewFlagSet("integration-serve", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var cfg Config
	pidFileFlag := ""
	logFileFlag := ""
	startupStatusFile := ""
	bindIntegrationServeFlags(fs, &cfg, &pidFileFlag, &logFileFlag, &startupStatusFile, startupOpts)

	if err := fs.Parse(args); err != nil {
		return 1
	}
	explicitDevice, _ := integrationFlagValue(fs, "device")

	logFilePath := integrationLogFilePath(map[string]string{"log-file": logFileFlag})
	serveLogWriter, err := newIntegrationDailyLogWriter(logFilePath, integrationLogRetentionDays)
	if err != nil {
		message := fmt.Sprintf("error: %v", err)
		if statusErr := writeIntegrationStartupStatus(startupStatusFile, integrationStartupStatus{
			Status:  "failed",
			Message: message,
		}); statusErr != nil {
			fmt.Fprintf(stderr, "startup status write failed: %v\n", statusErr)
		}
		fmt.Fprintln(stderr, message)
		return 1
	}
	defer serveLogWriter.Close()

	originalLogWriter := log.Writer()
	logOutput := io.Writer(serveLogWriter)
	if strings.TrimSpace(startupStatusFile) == "" {
		logOutput = io.MultiWriter(stderr, serveLogWriter)
	}
	log.SetOutput(logOutput)
	defer log.SetOutput(originalLogWriter)

	reportStartupFailure := func(format string, args ...interface{}) int {
		message := fmt.Sprintf(format, args...)
		if err := writeIntegrationStartupStatus(startupStatusFile, integrationStartupStatus{
			Status:  "failed",
			Message: message,
		}); err != nil {
			fmt.Fprintf(stderr, "startup status write failed: %v\n", err)
		}
		fmt.Fprintln(stderr, message)
		return 1
	}
	if err := integrationEnsureGitIdentityConfiguredFn(); err != nil {
		log.Printf("integration: git identity auto-config skipped: %v", err)
	}
	if err := validateIntegrationServicePort(cfg.Port); err != nil {
		return reportStartupFailure("error: %v", err)
	}
	if err := initializeIntegrationDeviceState(&cfg, explicitDevice); err != nil {
		return reportStartupFailure("error: initialize device: %v", err)
	}

	cfg.Reply = resolveServeReply(cfg.Reply)

	resolvedDefaultDir, err := resolveServeDefaultDir(cfg.DefaultDir)
	if err != nil {
		return reportStartupFailure("error: %v", err)
	}
	cfg.DefaultDir = resolvedDefaultDir
	resolvedAgentDir, err := resolveServeAgentDir(cfg.AgentDir, cfg.DefaultDir)
	if err != nil {
		return reportStartupFailure("error: %v", err)
	}
	cfg.AgentDir = resolvedAgentDir

	resolvedSiteDir, err := resolveServeSiteDir(cfg.Site)
	if err != nil {
		return reportStartupFailure("error: %v", err)
	}
	cfg.Site = resolvedSiteDir
	cfg.RuntimeHost = runtimehost.New(cfg.Host)

	pidFile := integrationPIDFilePath(map[string]string{"pid-file": pidFileFlag})
	if err := ensureParentDir(pidFile); err != nil {
		return reportStartupFailure("error: %v", err)
	}

	startupDir := integrationAppDir()
	if strings.TrimSpace(startupDir) == "" {
		return reportStartupFailure("error: resolve startup dir failed")
	}
	currentAppDir := integrationResourcesDir()
	if strings.TrimSpace(currentAppDir) == "" {
		currentAppDir = startupDir
	}
	if _, err := knowledgecore.OpenSharedDB(startupDir); err != nil {
		return reportStartupFailure("error: init knowledge runtime: %v", err)
	}
	knowledgeRoot, err := integrationKnowledgeRoot(cfg.AgentDir)
	if err != nil {
		return reportStartupFailure("error: resolve knowledge dir: %v", err)
	}
	if err := os.MkdirAll(knowledgeRoot, 0o755); err != nil {
		return reportStartupFailure("error: init knowledge dir: %v", err)
	}

	writeRuntimeConfig(map[string]interface{}{
		"app":                       firstNonEmpty(resolveExecutablePath(os.Args[0]), os.Args[0]),
		"app-dir":                   currentAppDir,
		"resources-dir":             integrationResourcesDir(),
		"db":                        resolveIntegrationDBPath(),
		"port":                      cfg.Port,
		"host":                      cfg.Host,
		"agent-dir":                 cfg.AgentDir,
		"default-dir":               cfg.DefaultDir,
		"device":                    cfg.Device,
		"agent-cache":               cfg.AgentCacheMs,
		"connect-cache":             cfg.ConnectCacheMs,
		"site":                      cfg.Site,
		"connect_timeout":           cfg.ConnectTimeoutMs,
		"knowledge_update_interval": cfg.KnowledgeUpdateIntervalMs,
		"knowledge_update_lock":     cfg.KnowledgeUpdateLockMs,
		"install_app":               cfg.InstallApp,
		"reply":                     cfg.Reply,
		"sleep":                     cfg.SleepMs,
		"thread":                    cfg.Thread,
		"queue":                     cfg.Queue,
		"retry_interval":            cfg.RetryIntervalMs,
		"retry_times":               cfg.RetryTimes,
		"http_timeout":              cfg.HTTPTimeout,
		"http_connect_timeout":      cfg.HTTPConnTimeout,
		"http_socket_timeout":       cfg.HTTPSocketTimeout,
		"http_debug":                cfg.HTTPDebug,
		"idle_timeout":              cfg.IdleTimeout,
		"plugin_exec_timeout":       cfg.PluginExecTimeout,
		"pid-file":                  pidFile,
		"log-file":                  integrationLogFilePath(map[string]string{"log-file": logFileFlag}),
	})
	if cfg.DeviceState != nil {
		cfg.DeviceState.captureConfigVersion()
	}

	connectTimeout := time.Duration(cfg.ConnectTimeoutMs) * time.Millisecond
	proxyClient := http11client.NewClient(http11client.Options{
		ConnectTimeout:      connectTimeout,
		KeepAlive:           60 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	})
	defer closeHTTPClientIdleConnections(proxyClient)

	connectSvc, err := newIntegrationConnectService(cfg.AgentDir)
	if err != nil {
		return reportStartupFailure("connect init failed: %v", err)
	}
	defer connectSvc.Close()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", handleChatCompletions(&cfg, proxyClient))
	mux.HandleFunc("/install_app", handleInstallApp())
	mux.HandleFunc("/api/agentId", handleAgentIDs(&cfg))
	mux.HandleFunc("/api/swarm_agent", handleSwarmAgents(&cfg))
	mux.HandleFunc("/api/deviceId", handleDeviceID(&cfg))
	mux.HandleFunc("/api/plugins/meta", handlePluginsMeta(&cfg))
	mux.HandleFunc("/api/plugins/status", handlePluginsStatus(&cfg))
	mux.HandleFunc("/api/plugins/config", handlePluginsConfig(&cfg))
	mux.HandleFunc("/api/plugins/start", handlePluginsStart(&cfg))
	mux.HandleFunc("/api/plugins/stop", handlePluginsStop(&cfg))
	mux.HandleFunc("/api/plugins/exec", handlePluginsExec(&cfg))
	mux.HandleFunc("/api/plugins/log", handlePluginsLog(&cfg))
	mux.HandleFunc("/api/folder", handleFolder(&cfg))
	mux.HandleFunc("/api/skills", handleSkills(&cfg))
	mux.HandleFunc("/api/skill_state", handleSkillState())
	mux.HandleFunc("/skills_warning", handleSkillsWarning(&cfg))
	mux.HandleFunc("/api/files", handleFiles())
	mux.HandleFunc("/api/data", handleData())
	mux.HandleFunc("/api/workspace", handleWorkspace(&cfg))
	mux.HandleFunc("/api/url_preview_probe", handleURLPreviewProbe(proxyClient))
	mux.HandleFunc("/api/edit", handleEdit(&cfg))
	mux.HandleFunc("/api/del", handleDel(&cfg))
	mux.HandleFunc("/api/raw", handleRaw(&cfg))
	mux.HandleFunc("/file/lastUpdate", handleFileLastUpdate(&cfg))
	mux.HandleFunc("/api/heartbeat", handleHeartbeat())
	mux.HandleFunc("/api/log_cleanup_status", handleLogCleanupStatus())
	mux.HandleFunc("/api/host", handleRuntimeHost(&cfg))
	mux.HandleFunc("/api/standalone", handleRuntimeStandalone(&cfg))
	mux.HandleFunc("/api/standalone=true", handleRuntimeStandalone(&cfg))
	mux.HandleFunc("/api/standalone=false", handleRuntimeStandalone(&cfg))
	mux.HandleFunc("/api/site/access", handleLANAccessURL(&cfg))
	mux.HandleFunc("/api/agent/init", handleAgentInit(&cfg))
	mux.HandleFunc("/api/copy", handleAgentCopy(&cfg))
	mux.HandleFunc("/api/agent/delete", handleAgentDelete(&cfg))
	mux.HandleFunc("/api/agent/export", handleAgentExport(&cfg))
	mux.HandleFunc("/api/agent/import", handleAgentImport(&cfg))
	mux.HandleFunc("/api/agent/create", handleAgentCreate(&cfg))
	mux.HandleFunc("/api/upload", handleUpload(&cfg))
	mux.HandleFunc("/api/config", handleSwarm(&cfg))
	mux.HandleFunc("/api/token", handleToken())
	mux.HandleFunc("/api/model_provider", handleModelProviderCatalog(&cfg))
	mux.HandleFunc("/api/runtime_config", handleRuntimeConfig())
	mux.HandleFunc("/api/consume", handleConsume())
	mux.HandleFunc("/api/message_insert/add", handleMessageInsertAdd())
	mux.HandleFunc("/api/message_insert/del", handleMessageInsertDel())
	mux.HandleFunc("/api/message_insert/delete", handleMessageInsertDelete())
	mux.HandleFunc("/api/message_insert/list", handleMessageInsertList())
	mux.HandleFunc("/api/sandbox", handleSandbox())
	mux.HandleFunc("/api/sandbox=off", handleSandbox())
	mux.HandleFunc("/api/sandbox=filepick", handleSandbox())
	mux.HandleFunc("/api/sandbox=net", handleSandbox())
	mux.HandleFunc("/api/sandbox=filepick_net", handleSandbox())
	mux.HandleFunc("/api/sandbox_status", handleSandboxStatus())
	connectHandler := connectsvc.NewDynamicHTTPHandler(func(_ *http.Request) (*connectsvc.Service, func(), error) {
		if connectSvc == nil {
			return nil, nil, fmt.Errorf("connect service is not initialized")
		}
		return connectSvc, nil, nil
	})
	mux.Handle("/api/connect/meta", connectHandler)
	mux.Handle("/api/connect/request", connectHandler)
	mux.Handle("/api/connect/response", connectHandler)
	mux.HandleFunc("/api/cron/create", handleCron(&cfg))
	mux.HandleFunc("/api/cron/agent/stop", handleCronAgentStop(&cfg))
	mux.HandleFunc("/api/cron/detail/metadata", handleCronMetadata())
	mux.HandleFunc("/api/cron/delete", handleCronDelete())
	mux.HandleFunc("/api/cron/detail/delete", handleCronDetailDelete())
	mux.HandleFunc("/api/cron/detail/list", handleCronDetailList())
	mux.HandleFunc("/api/cron/detail/update", handleCronDetailUpdate(&cfg))
	mux.HandleFunc("/api/cron/meta/update", handleCronMetaUpdate(&cfg))
	mux.HandleFunc("/api/cron/detail/status", handleCronDetailStatus())
	mux.HandleFunc("/api/cancel", handleCancel(&cfg))
	mux.HandleFunc("/api/restore", handleRestore())
	mux.HandleFunc("/api/chat_session_log", handleChatSessionLog())
	mux.HandleFunc("/log_round", handleLogRound(&cfg))
	mux.HandleFunc("/log_skill", handleLogSkill(&cfg))
	mux.HandleFunc("/log_skill_status", handleLogSkillStatus(&cfg))
	mux.HandleFunc("/api/download", handleDownload())
	mux.HandleFunc("/api/cmd", handleCmd(&cfg))
	mux.HandleFunc("/api/kill", handleKill(&cfg))
	registerAppStatic(mux, &cfg)
	mux.HandleFunc("/knowledge", handleKnowledge(&cfg))
	mux.HandleFunc("/knowledge/", handleKnowledge(&cfg))
	mux.HandleFunc("/knowledge_lastUpdate", handleKnowledgeLastUpdate())
	mux.HandleFunc("/knowledge_path", handleKnowledgePath(&cfg))
	mux.HandleFunc("/launch", handleIntegrationBrowserLaunch())
	if err := server.Register(mux, cfg.Site); err != nil {
		log.Printf("static register failed: %v", err)
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return reportStartupFailure("server listen failed: %v", err)
	}
	defer listener.Close()
	if err := writePIDFile(pidFile); err != nil {
		return reportStartupFailure("error: %v", err)
	}
	defer os.Remove(pidFile)
	_ = os.Remove(startupStatusFile)
	associatedPlugins, associatedErr := readIntegrationAssociatedPlugins()
	if associatedErr != nil {
		log.Printf("associated plugins config read failed: %v", associatedErr)
	}

	ctx, stop := signalContext()
	defer stop()
	associatedStartCtx, cancelAssociatedStart := context.WithCancel(context.Background())
	defer cancelAssociatedStart()
	associatedStartDone := make(chan struct{})
	var managedPluginStopOnce sync.Once
	managedPluginStopWarnings := []string(nil)
	stopManagedPlugins := func(logWriter io.Writer) []string {
		managedPluginStopOnce.Do(func() {
			cancelAssociatedStart()
			<-associatedStartDone
			managedPluginStopWarnings = stopIntegrationApplicationPlugins(associatedPlugins, logWriter)
		})
		return managedPluginStopWarnings
	}
	shutdownController := newIntegrationShutdownController(stop, nil)
	shutdownController.stopPlugins = stopManagedPlugins
	mux.HandleFunc("/api/shutdown", handleShutdown(shutdownController))
	server := &http.Server{Handler: withStandaloneAPIProtection(mux, &cfg)}
	defer func() {
		if removed, err := cleanupStartupPIDFiles(pidFile); err != nil {
			log.Printf("serve cleanup startup pid files failed: %v", err)
		} else if len(removed) > 0 {
			log.Printf("serve cleanup removed pid files: %s", strings.Join(removed, ", "))
		}
	}()

	startCliGet(ctx, &cfg)
	startIntegrationDeviceConfigRefresh(ctx, cfg.DeviceState)
	initCronDB()
	startIntegrationLogRetentionCleanup(ctx)
	startCronCheck(ctx, &cfg)
	startCronExecutor(ctx, &cfg, proxyClient, connectSvc)
	startConnectPendingRequestSync(ctx, &cfg, connectSvc)
	startIntegrationAssociatedPluginsAsync(associatedStartCtx, associatedPlugins, nil, associatedStartDone)

	go func() {
		<-ctx.Done()
		for _, warning := range stopManagedPlugins(nil) {
			log.Printf("plugin shutdown warning: %s", warning)
		}
		cancelAllActiveCmds()
		closeHTTPClientIdleConnections(proxyClient)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), integrationShutdownTimeout)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	log.Printf("integration started on %s → %s", addr, cfg.currentHost())
	log.Printf("agent-dir: %s", cfg.AgentDir)
	log.Printf("site-dir: %s", cfg.Site)
	log.Printf("plugin-dir: %s", strings.TrimSpace(os.Getenv(integrationPluginDirEnv)))
	log.Printf("pid-file: %s", pidFile)

	if integrationShouldOpenBrowser() {
		go func() {
			openIntegrationBrowserAfterEntryReady(cfg.Port, log.Printf)
		}()
	}

	err = server.Serve(listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return reportStartupFailure("server failed: %v", err)
	}

	cancelAllActiveCmds()
	if cronDB != nil {
		_ = cronDB.Close()
		cronDB = nil
	}
	return 0
}

func startIntegrationProcess(flags map[string]string, stdout, stderr io.Writer) int {
	_ = stdout
	flags = mergeIntegrationRuntimeFlags(flags)
	if err := validateIntegrationTopLevelPortFlag(flags); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	pidFile := integrationPIDFilePath(flags)
	logWriter, logFile, err := openIntegrationLifecycleLog(flags)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	defer logWriter.Close()
	integrationLifecycleLog(logWriter, "start requested pid-file=%s log-file=%s", pidFile, logFile)
	if _, configPath, err := loadIntegrationStartupOptionsWithPortOverride(strings.TrimSpace(flags["port"]) != ""); err != nil {
		message := err.Error()
		if strings.TrimSpace(configPath) != "" {
			message = fmt.Sprintf("%s: %v", configPath, err)
		}
		integrationLifecycleLog(logWriter, "start failed: %s", message)
		fmt.Fprintln(stderr, message)
		return 1
	}
	if removed, err := cleanupStartupPIDFiles(pidFile); err != nil {
		integrationLifecycleLog(logWriter, "start warning: cleanup startup pid files failed: %v", err)
	} else if len(removed) > 0 {
		integrationLifecycleLog(logWriter, "start cleanup removed pid files: %s", strings.Join(removed, ", "))
	}

	port := resolveIntegrationLaunchPort(flags)
	if pid, ok := readRunningPID(pidFile); ok {
		if opened, openErr := openExistingIntegrationBrowserIfReady(port, logWriter, stderr, "start reused running service pid=%d pid-file=%s addr=%s", pid, pidFile, fmt.Sprintf("127.0.0.1:%d", port)); opened {
			if openErr != nil {
				return 1
			}
			return 0
		}
		integrationLifecycleLog(logWriter, "start failed: already running pid=%d pid-file=%s", pid, pidFile)
		fmt.Fprintf(stderr, "integration already running: pid=%d\n", pid)
		return 1
	}
	if opened, openErr := openExistingIntegrationBrowserIfReady(port, logWriter, stderr, "start reused running service without pid-file=%s addr=%s", pidFile, fmt.Sprintf("127.0.0.1:%d", port)); opened {
		if openErr != nil {
			return 1
		}
		return 0
	}
	if err := ensureParentDir(pidFile); err != nil {
		integrationLifecycleLog(logWriter, "start failed: ensure pid-file dir error: %v", err)
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	exe, err := integrationExecutableFn()
	if err != nil {
		integrationLifecycleLog(logWriter, "start failed: resolve executable error: %v", err)
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	args := []string{"serve"}
	args = append(args, rebuildTopLevelFlags(flags, map[string]struct{}{
		"log-file":            {},
		"pid-file":            {},
		"startup-status-file": {},
	})...)
	args = append(args, "--pid-file", pidFile, "--log-file", logFile)
	startupStatusFile := integrationStartupStatusFilePath(pidFile)
	if err := os.Remove(startupStatusFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		integrationLifecycleLog(logWriter, "start warning: remove startup status file failed: %v", err)
	}
	args = append(args, "--startup-status-file", startupStatusFile)

	childPID, err := startDetachedIntegrationProcess(exe, args, logFile, true)
	if err != nil {
		integrationLifecycleLog(logWriter, "start failed: spawn error: %v", err)
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	integrationLifecycleLog(logWriter, "start spawned child pid=%d", childPID)

	deadline := time.Now().Add(integrationStartWait)
	for time.Now().Before(deadline) {
		if status, err := readIntegrationStartupStatus(startupStatusFile); err == nil && strings.EqualFold(status.Status, "failed") && strings.TrimSpace(status.Message) != "" {
			message := strings.TrimSpace(status.Message)
			integrationLifecycleLog(logWriter, "start failed: %s", message)
			fmt.Fprintln(stderr, message)
			return 1
		}
		if pid, ok := readRunningPID(pidFile); ok && integrationReadyCheck(strconv.Itoa(port)) {
			_ = os.Remove(startupStatusFile)
			integrationLifecycleLog(logWriter, "service started pid=%d pid-file=%s log-file=%s addr=%s", pid, pidFile, logFile, fmt.Sprintf("127.0.0.1:%d", port))
			if _, openErr := openExistingIntegrationBrowserIfReady(port, logWriter, stderr, ""); openErr != nil {
				return 1
			}
			return 0
		}
		if !integrationProcessAlive(childPID) {
			if status, err := readIntegrationStartupStatus(startupStatusFile); err == nil && strings.TrimSpace(status.Message) != "" {
				message := strings.TrimSpace(status.Message)
				integrationLifecycleLog(logWriter, "start failed: %s", message)
				fmt.Fprintln(stderr, message)
				return 1
			}
			message := fmt.Sprintf("integration start failed before readiness: child pid=%d exited", childPID)
			integrationLifecycleLog(logWriter, "start failed: %s", message)
			fmt.Fprintln(stderr, message)
			return 1
		}
		time.Sleep(100 * time.Millisecond)
	}

	if pid, ok := readRunningPID(pidFile); ok {
		if proc, err := os.FindProcess(pid); err == nil {
			_ = proc.Signal(syscall.SIGTERM)
			time.Sleep(300 * time.Millisecond)
			if _, ok := readRunningPID(pidFile); ok {
				_ = proc.Signal(syscall.SIGKILL)
			}
		}
	}
	_ = os.Remove(pidFile)
	_ = os.Remove(startupStatusFile)
	integrationLifecycleLog(logWriter, "start failed: timeout waiting for pid-file=%s", pidFile)
	fmt.Fprintln(stderr, "start timeout waiting for integration service")
	return 1
}

func stopIntegrationProcess(flags map[string]string, stdout, stderr io.Writer) int {
	_ = stdout
	flags = mergeIntegrationRuntimeFlags(flags)
	if err := validateIntegrationTopLevelPortFlag(flags); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	pidFile := integrationPIDFilePath(flags)
	logWriter, logFile, err := openIntegrationLifecycleLog(flags)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	defer logWriter.Close()
	integrationLifecycleLog(logWriter, "stop requested pid-file=%s log-file=%s", pidFile, logFile)
	pid, ok := readRunningPID(pidFile)
	if !ok {
		if err := os.Remove(pidFile); err == nil || errors.Is(err, os.ErrNotExist) {
			integrationLifecycleLog(logWriter, "service already stopped pid-file=%s", pidFile)
			return 0
		}
		integrationLifecycleLog(logWriter, "stop failed: integration not running pid-file=%s", pidFile)
		fmt.Fprintf(stderr, "integration not running: %s\n", pidFile)
		return 1
	}
	associatedPlugins, associatedErr := readIntegrationAssociatedPlugins()
	if associatedErr != nil {
		warning := fmt.Sprintf("stop associated plugins config read failed: %v", associatedErr)
		integrationLifecycleLog(logWriter, "stop warning: %s", warning)
		fmt.Fprintln(stderr, warning)
	}
	for _, warning := range stopIntegrationApplicationPlugins(associatedPlugins, logWriter) {
		integrationLifecycleLog(logWriter, "stop warning: %s", warning)
		fmt.Fprintln(stderr, warning)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		integrationLifecycleLog(logWriter, "stop failed: find process pid=%d error: %v", pid, err)
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		integrationLifecycleLog(logWriter, "stop failed: signal SIGTERM pid=%d error: %v", pid, err)
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	integrationLifecycleLog(logWriter, "stop signaled pid=%d with SIGTERM", pid)

	deadline := time.Now().Add(integrationStopWait)
	for time.Now().Before(deadline) {
		if _, ok := readRunningPID(pidFile); !ok {
			_ = os.Remove(pidFile)
			if removed, err := cleanupStartupPIDFiles(pidFile); err != nil {
				integrationLifecycleLog(logWriter, "stop warning: cleanup startup pid files failed: %v", err)
			} else if len(removed) > 0 {
				integrationLifecycleLog(logWriter, "stop cleanup removed pid files: %s", strings.Join(removed, ", "))
			}
			integrationLifecycleLog(logWriter, "service stopped pid=%d pid-file=%s", pid, pidFile)
			return 0
		}
		time.Sleep(100 * time.Millisecond)
	}

	_ = proc.Signal(syscall.SIGKILL)
	integrationLifecycleLog(logWriter, "stop timeout: sent SIGKILL to pid=%d", pid)
	time.Sleep(500 * time.Millisecond)
	if _, ok := readRunningPID(pidFile); ok {
		integrationLifecycleLog(logWriter, "stop failed: timeout waiting pid=%d", pid)
		fmt.Fprintf(stderr, "stop timeout waiting pid=%d\n", pid)
		return 1
	}
	_ = os.Remove(pidFile)
	if removed, err := cleanupStartupPIDFiles(pidFile); err != nil {
		integrationLifecycleLog(logWriter, "stop warning: cleanup startup pid files failed: %v", err)
	} else if len(removed) > 0 {
		integrationLifecycleLog(logWriter, "stop cleanup removed pid files: %s", strings.Join(removed, ", "))
	}
	integrationLifecycleLog(logWriter, "service stopped pid=%d pid-file=%s after SIGKILL", pid, pidFile)
	return 0
}

func restartIntegrationProcess(flags map[string]string, stdout, stderr io.Writer) int {
	flags = mergeIntegrationRuntimeFlags(flags)
	if err := validateIntegrationTopLevelPortFlag(flags); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if logWriter, _, err := openIntegrationLifecycleLog(flags); err == nil {
		integrationLifecycleLog(logWriter, "restart requested pid-file=%s log-file=%s", integrationPIDFilePath(flags), integrationLogFilePath(flags))
		_ = logWriter.Close()
	}
	if code := stopIntegrationProcess(flags, stdout, stderr); code != 0 {
		return code
	}
	return startIntegrationProcess(flags, stdout, stderr)
}

func buildPluginActionResponse(result *connectsvc.PluginActionResult) pluginActionResponse {
	return connectsvc.BuildPluginActionResponse(result)
}

func queryBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func parseRequiredIntValue(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("integer value is required")
	}
	return strconv.Atoi(raw)
}

func runIntegrationCronCLI(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		printCLIHelp()
		os.Exit(1)
	}
	if cronDB == nil {
		initCronDB()
	}
	if cronDB == nil {
		log.Fatal("db not ready")
	}
	switch args[0] {
	case "create":
		if hasHelpFlag(args[1:]) {
			printCronCreateHelp()
			return
		}
		cycle, rawTime, agentID, chatID, model, content, agentDir := 0, "", "", "", "", "", ""
		thinking, routerDisable := false, true
		cronExpr := ""
		if err := parseCLIArgs(args[1:], &cycle, &rawTime, &agentID, &chatID, &model, &content, &thinking, &routerDisable, &cronExpr, &agentDir); err != nil {
			log.Fatal(err)
		}
		if err := validateCLICreateArgs(cycle, rawTime, agentID, model, content); err != nil {
			log.Fatal(err)
		}
		deviceID := resolveIntegrationDeviceID("")
		resolvedAgentDir, err := resolveIntegrationAgentDir(agentDir)
		if err != nil {
			log.Fatal(err)
		}
		if err := validateIntegrationAgentExists(resolvedAgentDir, deviceID, agentID); err != nil {
			log.Fatal(err)
		}
		if err := validateIntegrationRegisteredModel(model); err != nil {
			log.Fatal(err)
		}
		resp, err := createCronTask(agentID, cronCreateRequest{
			Content:       content,
			Model:         model,
			Thinking:      thinking,
			RouterDisable: routerDisable,
			RawTime:       rawTime,
			Cycle:         cycle,
			ChatID:        chatID,
		})
		if err != nil {
			log.Fatal(err)
		}
		out, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(out))
	case "create-cron", "submit-cron":
		if hasHelpFlag(args[1:]) {
			printCronCreateCronHelp()
			return
		}
		cronExpr, agentID, chatID, model, content, agentDir := "", "", "", "", "", ""
		thinking, routerDisable := false, true
		if err := parseCLIArgs(args[1:], nil, nil, &agentID, &chatID, &model, &content, &thinking, &routerDisable, &cronExpr, &agentDir); err != nil {
			log.Fatal(err)
		}
		if err := validateCLICreateCronArgs(cronExpr, agentID, model, content); err != nil {
			log.Fatal(err)
		}
		deviceID := resolveIntegrationDeviceID("")
		resolvedAgentDir, err := resolveIntegrationAgentDir(agentDir)
		if err != nil {
			log.Fatal(err)
		}
		if err := validateIntegrationAgentExists(resolvedAgentDir, deviceID, agentID); err != nil {
			log.Fatal(err)
		}
		if err := validateIntegrationRegisteredModel(model); err != nil {
			log.Fatal(err)
		}
		resp, err := createCronTask(agentID, cronCreateRequest{
			Content:       content,
			Model:         model,
			Thinking:      thinking,
			RouterDisable: routerDisable,
			Cycle:         -1,
			Cron:          cronExpr,
			ChatID:        chatID,
		})
		if err != nil {
			log.Fatal(err)
		}
		out, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(out))
	case "find-meta":
		if hasHelpFlag(args[1:]) {
			printCronFindMetaHelp()
			return
		}
		opts, err := parseCronQueryArgs(args[1:])
		if err != nil {
			log.Fatal(err)
		}
		filter, err := buildCronMetaFilter(opts)
		if err != nil {
			log.Fatal(err)
		}
		items, err := queryCronMetas(cronDB, filter)
		if err != nil {
			log.Fatal(err)
		}
		out, _ := json.MarshalIndent(map[string]interface{}{"status": 0, "data": items}, "", "  ")
		fmt.Println(string(out))
	case "find-detail":
		if hasHelpFlag(args[1:]) {
			printCronFindDetailHelp()
			return
		}
		opts, err := parseCronQueryArgs(args[1:])
		if err != nil {
			log.Fatal(err)
		}
		filter, err := buildCronDetailFilter(opts)
		if err != nil {
			log.Fatal(err)
		}
		items, err := queryCronDetails(cronDB, filter)
		if err != nil {
			log.Fatal(err)
		}
		out, _ := json.MarshalIndent(map[string]interface{}{"status": 0, "data": items}, "", "  ")
		fmt.Println(string(out))
	case "delete-meta":
		if hasHelpFlag(args[1:]) {
			printCronDeleteMetaHelp()
			return
		}
		opts, err := parseCronQueryArgs(args[1:])
		if err != nil {
			log.Fatal(err)
		}
		filter, err := buildCronMetaFilter(opts)
		if err != nil {
			log.Fatal(err)
		}
		if strings.TrimSpace(opts.ID) == "" && filter.AgentID == "" && filter.ChatID == "" && filter.Model == "" && filter.Content == "" && filter.Cycle == nil && filter.StartFrom == nil && filter.StartTo == nil {
			log.Fatal("id or filter is required")
		}
		affected, ids, err := deleteCronMetas(cronDB, filter, opts.ID)
		if err != nil {
			log.Fatal(err)
		}
		out, _ := json.MarshalIndent(map[string]interface{}{"status": 0, "affected": affected, "ids": ids}, "", "  ")
		fmt.Println(string(out))
	case "delete-detail":
		if hasHelpFlag(args[1:]) {
			printCronDeleteDetailHelp()
			return
		}
		opts, err := parseCronQueryArgs(args[1:])
		if err != nil {
			log.Fatal(err)
		}
		filter, err := buildCronDetailDeleteFilter(opts)
		if err != nil {
			log.Fatal(err)
		}
		if strings.TrimSpace(opts.DetailID) == "" && strings.TrimSpace(opts.MetaID) == "" &&
			filter.AgentID == "" && filter.ChatID == "" && filter.Model == "" && filter.Content == "" && filter.Cycle == nil && filter.ExecFrom == nil && filter.ExecTo == nil {
			log.Fatal("detailId, metaId or filter is required")
		}
		affected, ids, err := deleteCronDetails(cronDB, filter, opts.DetailID, opts.MetaID)
		if err != nil {
			log.Fatal(err)
		}
		out, _ := json.MarshalIndent(map[string]interface{}{"status": 0, "affected": affected, "ids": ids}, "", "  ")
		fmt.Println(string(out))
	default:
		fmt.Println("Unknown cron command:", args[0])
		printCLIHelp()
		os.Exit(1)
	}
}

func runIntegrationPluginCLI(args []string) {
	if len(args) == 0 || hasHelpFlag(args) {
		fmt.Println("Usage:")
		fmt.Println("  integration plugins start --key KEY [plugin flags...]")
		fmt.Println("  integration plugins stop --key KEY [plugin flags...]")
		fmt.Println("  integration plugins exec --key KEY --command 'SUBCOMMAND [ARGS...]' [plugin flags...]")
		fmt.Println("  integration plugins status --key KEY [--pid-file PATH]")
		fmt.Println("  integration plugins sync-bundled [--check]")
		fmt.Println("")
		fmt.Println("Subcommands:")
		fmt.Println("  exec          Execute one plugin command")
		fmt.Println("  start         Start a plugin process")
		fmt.Println("  stop          Stop a plugin process")
		fmt.Println("  status        Check whether a plugin process is running")
		fmt.Println("  sync-bundled  Check or sync bundled plugins into the runtime plugins directory")
		fmt.Println("")
		fmt.Println("Notes:")
		fmt.Println("  start 会自动补齐 --connect-bin；exec 仅在 browser start 之外的插件命令需要时补齐。")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  integration plugins start --key feishu --connect-bin ./integration")
		fmt.Println("  integration plugins stop --key feishu --pid-file ./plugins/feishu.pid")
		fmt.Println("  integration plugins exec --key browser --command 'instance init' --agentId A --chatId B")
		fmt.Println("  integration plugins status --key feishu")
		fmt.Println("  integration plugins sync-bundled --check")
		return
	}

	switch args[0] {
	case "exec":
		flags, err := connectsvc.ParseFlags(args[1:])
		if err != nil {
			log.Fatal(err)
		}
		target := connectsvc.FirstValue(flags, "key")
		command := connectsvc.FirstValue(flags, "command")
		delete(flags, "key")
		delete(flags, "command")
		ensureIntegrationPluginConnectBinForExec(target, command, flags)
		result, err := connectsvc.RunPluginExecCommand(target, command, flags, resolveIntegrationPluginExecTimeout(nil))
		if err != nil {
			log.Fatal(err)
		}
		out, _ := json.MarshalIndent(map[string]any{"status": 0, "data": buildPluginActionResponse(result)}, "", "  ")
		fmt.Println(string(out))
	case "start", "stop":
		flags, err := connectsvc.ParseFlags(args[1:])
		if err != nil {
			log.Fatal(err)
		}
		target := connectsvc.FirstValue(flags, "key")
		delete(flags, "key")
		if args[0] == "start" {
			ensureIntegrationPluginConnectBin(flags)
		}
		result, err := runConnectPluginAction(target, args[0], flags)
		if err != nil {
			log.Fatal(err)
		}
		out, _ := json.MarshalIndent(map[string]any{"status": 0, "data": buildPluginActionResponse(result)}, "", "  ")
		fmt.Println(string(out))
	case "status":
		flags, err := connectsvc.ParseFlags(args[1:])
		if err != nil {
			log.Fatal(err)
		}
		target := connectsvc.FirstValue(flags, "key")
		delete(flags, "key")
		result, err := connectsvc.PluginStatusByKey(target, flags)
		if err != nil {
			log.Fatal(err)
		}
		out, _ := json.MarshalIndent(map[string]any{"status": 0, "data": buildPluginStatusResponse(result)}, "", "  ")
		fmt.Println(string(out))
	case "sync-bundled":
		if code := runIntegrationPluginSyncBundledCLI(args[1:], os.Stdout, os.Stderr); code != 0 {
			os.Exit(code)
		}
	default:
		log.Fatalf("unknown plugins command: %s", args[0])
	}
}

func runIntegrationSkillsWarningCLI(args []string, cfg *Config) {
	fs := flag.NewFlagSet("skills-warning", flag.ExitOnError)
	refresh := fs.Bool("refresh", false, "scan and sync warnings before reading")
	_ = fs.Parse(args)

	if *refresh {
		if err := syncIntegrationSkillsWarnings(cfg); err != nil {
			log.Fatal(err)
		}
	}

	store, err := skillscore.OpenWarningStore(resolveIntegrationDBPath())
	if err != nil {
		log.Fatal(err)
	}
	out, err := store.ListJSON()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}

func runIntegrationFileLastUpdateCLI(args []string, cfg *Config) {
	fs := flag.NewFlagSet("file-last-update", flag.ExitOnError)
	agentID := fs.String("agentId", "", "agent id for relative file paths")
	agent := fs.String("agent", "", "agent id for relative file paths")
	filePath := fs.String("file", "", "target file path")
	_ = fs.Parse(args)

	targetAgentID := strings.TrimSpace(*agentID)
	if targetAgentID == "" {
		targetAgentID = strings.TrimSpace(*agent)
	}
	resolved, err := resolveFileTarget(cfg, targetAgentID, strings.TrimSpace(*filePath))
	if err != nil {
		log.Fatal(err)
	}
	value, err := fileLastUpdateDuration(resolved, time.Now())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(strconv.FormatInt(value, 10))
}

type backupCleanupMove struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type backupCleanupSummary struct {
	AgentCount int                 `json:"agentCount"`
	Archived   []backupCleanupMove `json:"archived,omitempty"`
	Deleted    []string            `json:"deleted,omitempty"`
}

const (
	defaultBackupArchiveAfter = 24 * time.Hour
	defaultBackupDeleteAfter  = 72 * time.Hour
)

func printBackupCleanHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration backup-clean [options]")
	fmt.Println("")
	fmt.Println("Description:")
	fmt.Println("  Scan every Agent workspace for stale User/Soul backup files.")
	fmt.Println("  Files in the workspace root that look like USER.md / SOUL.md backups and")
	fmt.Println("  are older than --archive-after are moved into workspace/bak.")
	fmt.Println("  Files already under workspace/bak and older than --delete-after are deleted.")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --agent-dir DIR          Agent 根目录；未传时会复用主应用 config/config.json / AGENT_DIR / 默认目录")
	fmt.Println("  --archive-after DURATION 归档阈值，默认 24h")
	fmt.Println("  --delete-after DURATION  删除阈值，默认 72h")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  integration backup-clean")
	fmt.Println("  integration backup-clean --agent-dir ./agent")
	fmt.Println("  integration backup-clean --archive-after 24h --delete-after 72h")
}

func runIntegrationBackupCleanCLI(args []string, stdout, stderr io.Writer) int {
	if hasHelpFlag(args) {
		printBackupCleanHelp()
		return 0
	}

	fs := flag.NewFlagSet("integration backup-clean", flag.ContinueOnError)
	fs.SetOutput(stderr)
	agentDir := fs.String("agent-dir", "", "agent root dir")
	archiveAfter := fs.Duration("archive-after", defaultBackupArchiveAfter, "move stale workspace backups into bak after this duration")
	deleteAfter := fs.Duration("delete-after", defaultBackupDeleteAfter, "delete stale bak files after this duration")
	fs.Usage = func() { printBackupCleanHelp() }
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *archiveAfter < 0 {
		fmt.Fprintln(stderr, "archive-after must be >= 0")
		return 1
	}
	if *deleteAfter < 0 {
		fmt.Fprintln(stderr, "delete-after must be >= 0")
		return 1
	}

	resolvedAgentDir, err := resolveIntegrationAgentDir(*agentDir)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	summary, err := cleanupIntegrationAgentBackups(resolvedAgentDir, time.Now(), *archiveAfter, *deleteAfter)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	out, err := json.MarshalIndent(map[string]any{
		"status":      0,
		"agentDir":    resolvedAgentDir,
		"agentCount":  summary.AgentCount,
		"archived":    summary.Archived,
		"archivedCnt": len(summary.Archived),
		"deleted":     summary.Deleted,
		"deletedCnt":  len(summary.Deleted),
	}, "", "  ")
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	fmt.Fprintln(stdout, string(out))
	return 0
}

func cleanupIntegrationAgentBackups(agentRoot string, now time.Time, archiveAfter, deleteAfter time.Duration) (backupCleanupSummary, error) {
	entries, err := os.ReadDir(agentRoot)
	if err != nil {
		return backupCleanupSummary{}, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	summary := backupCleanupSummary{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		agentID := strings.TrimSpace(entry.Name())
		if agentID == "" || strings.HasPrefix(agentID, ".") {
			continue
		}
		if err := integrationagentarchive.ValidateAgentID(agentID); err != nil {
			continue
		}
		summary.AgentCount++
		if err := cleanupAgentWorkspaceBackups(filepath.Join(agentRoot, entry.Name()), now, archiveAfter, deleteAfter, &summary); err != nil {
			return backupCleanupSummary{}, fmt.Errorf("cleanup agent %s: %w", agentID, err)
		}
	}
	return summary, nil
}

func cleanupAgentWorkspaceBackups(workspace string, now time.Time, archiveAfter, deleteAfter time.Duration, summary *backupCleanupSummary) error {
	entries, err := os.ReadDir(workspace)
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	bakDir := filepath.Join(workspace, "bak")
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !isAgentBackupFileName(name) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if now.Sub(info.ModTime()) < archiveAfter {
			continue
		}
		if err := os.MkdirAll(bakDir, 0o755); err != nil {
			return err
		}
		src := filepath.Join(workspace, name)
		dst, err := nextAvailableBackupPath(bakDir, name)
		if err != nil {
			return err
		}
		if err := os.Rename(src, dst); err != nil {
			return err
		}
		summary.Archived = append(summary.Archived, backupCleanupMove{From: src, To: dst})
	}

	info, err := os.Stat(bakDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("bak exists but is not a directory: %s", bakDir)
	}
	return filepath.WalkDir(bakDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == bakDir || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if now.Sub(info.ModTime()) < deleteAfter {
			return nil
		}
		if err := os.Remove(path); err != nil {
			return err
		}
		summary.Deleted = append(summary.Deleted, path)
		return nil
	})
}

func isAgentBackupFileName(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return false
	}
	if lower == "user.md" || lower == "soul.md" {
		return false
	}
	if !hasAgentBackupStem(lower) {
		return false
	}
	return strings.Contains(lower, "bak") || hasTimestampLike(lower)
}

func hasAgentBackupStem(lower string) bool {
	for _, stem := range []string{"user.md", "soul.md"} {
		if strings.Contains(lower, stem) {
			return true
		}
	}
	return hasBackupStemPrefix(lower, "user") || hasBackupStemPrefix(lower, "soul")
}

func hasBackupStemPrefix(lower, stem string) bool {
	if !strings.HasPrefix(lower, stem) || len(lower) <= len(stem) {
		return false
	}
	next := lower[len(stem)]
	if next >= '0' && next <= '9' {
		return true
	}
	switch next {
	case '.', '_', '-', ' ', '(', '[':
		return true
	default:
		return false
	}
}

func hasTimestampLike(name string) bool {
	digits := 0
	for i := 0; i < len(name); i++ {
		if name[i] >= '0' && name[i] <= '9' {
			digits++
			if digits >= 8 {
				return true
			}
		}
	}
	return false
}

func nextAvailableBackupPath(dir, name string) (string, error) {
	target := filepath.Join(dir, name)
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return target, nil
	} else if err != nil {
		return "", err
	}

	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for idx := 1; ; idx++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s.%d%s", base, idx, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate, nil
		} else if err != nil {
			return "", err
		}
	}
}

func runIntegrationNotifyCLI(args []string) {
	fs := flag.NewFlagSet("notify", flag.ExitOnError)
	title := fs.String("title", integrationCompletionNotificationTitle, "notification title")
	message := fs.String("message", "", "notification message")
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}
	if err := integrationSendNotificationFn(integrationnotification.Options{
		Title:   *title,
		Message: *message,
	}); err != nil {
		maybeOpenIntegrationNotificationSettings(err)
		log.Fatal(err)
	}
	out, _ := json.MarshalIndent(map[string]any{
		"status":    0,
		"supported": integrationNotificationSupportedFn(),
		"title":     strings.TrimSpace(*title),
		"message":   strings.TrimSpace(*message),
	}, "", "  ")
	fmt.Println(string(out))
}

func runIntegrationLogRoundCLI(args []string, cfg *Config) {
	fs := flag.NewFlagSet("log-round", flag.ExitOnError)
	agentID := fs.String("agentId", "", "agent id")
	agent := fs.String("agent", "", "agent id")
	chatID := fs.String("chatId", "", "chat id")
	chat := fs.String("chat", "", "chat id")
	round := fs.Int("round", 0, "recent round count")
	_ = fs.Parse(args)

	targetAgentID := strings.TrimSpace(*agentID)
	if targetAgentID == "" {
		targetAgentID = strings.TrimSpace(*agent)
	}
	_ = targetAgentID
	targetChatID := strings.TrimSpace(*chatID)
	if targetChatID == "" {
		targetChatID = strings.TrimSpace(*chat)
	}
	path, err := exportRoundLog(cfg, targetChatID, roundLogOptions{Round: *round})
	if err != nil {
		log.Fatal(err)
	}
	out, _ := json.MarshalIndent(map[string]any{"status": 0, "path": path}, "", "  ")
	fmt.Println(string(out))
}

// startCronCheck runs periodic cron detail creation as a background goroutine.
func startCronCheck(ctx context.Context, cfg *Config) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		cronCheckOnce(cfg)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cronCheckOnce(cfg)
			}
		}
	}()
	log.Println("[cron] periodic check started (every 1 minute)")
}

// startCronExecutor runs periodic execution of due cron task details.
func startCronExecutor(ctx context.Context, cfg *Config, proxyClient *http.Client, connectSvc *connectsvc.Service) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		if err := syncIntegrationSkillsWarnings(cfg); err != nil {
			log.Printf("[skills_warning] initial sync failed: %v", err)
		}
		cronExecuteOnce(cfg, proxyClient, connectSvc)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := syncIntegrationSkillsWarnings(cfg); err != nil {
					log.Printf("[skills_warning] periodic sync failed: %v", err)
				}
				cronExecuteOnce(cfg, proxyClient, connectSvc)
			}
		}
	}()
	log.Println("[cron] task executor started (every 1 minute)")
}

func startConnectPendingRequestSync(ctx context.Context, cfg *Config, connectSvc *connectsvc.Service) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		syncConnectPendingRequests(cfg, connectSvc)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				syncConnectPendingRequests(cfg, connectSvc)
			}
		}
	}()
	log.Println("[connect] pending request sync started (every 30 seconds)")
}

func lookupTokenByModel(db *sql.DB, model string) (string, error) {
	model = sharedutil.NormalizeTokenModelName(model)
	if model == "" {
		return "", nil
	}
	cfg, err := lookupTokenConfigByModel(db, model)
	if err != nil {
		return "", err
	}
	return cfg.Token, nil
}

func buildConnectRequestTaskContent(reqs []connectsvc.Request) string {
	parts := make([]string, 0, len(reqs))
	for _, req := range reqs {
		requestText := strings.TrimSpace(req.Request)
		artifactOnly := requestText == "[image]" || requestText == "[file]"
		if requestText != "" && !artifactOnly {
			parts = append(parts, requestText)
		}
		for _, artifact := range strings.Split(strings.TrimSpace(req.Artifacts), ",") {
			artifact = strings.TrimSpace(artifact)
			if artifact == "" {
				continue
			}
			label := "[file]"
			lower := strings.ToLower(artifact)
			switch {
			case strings.HasPrefix(requestText, "[image]"):
				label = "[image]"
			case strings.HasPrefix(requestText, "[file]"):
				label = "[file]"
			case strings.HasSuffix(lower, ".png"), strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"), strings.HasSuffix(lower, ".gif"), strings.HasSuffix(lower, ".webp"), strings.HasSuffix(lower, ".bmp"), strings.HasSuffix(lower, ".svg"):
				label = "[image]"
			}
			parts = append(parts, label+artifact)
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func buildConnectRequestTextContent(reqs []connectsvc.Request) string {
	parts := make([]string, 0, len(reqs))
	for _, req := range reqs {
		requestText := strings.TrimSpace(req.Request)
		artifactOnly := requestText == "[image]" || requestText == "[file]"
		if requestText != "" && !artifactOnly {
			parts = append(parts, requestText)
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func lastConnectRequest(reqs []connectsvc.Request) *connectsvc.Request {
	return connectsvc.LastRequest(reqs)
}

func lastConnectRequestID(reqs []connectsvc.Request) string {
	return connectsvc.LastRequestID(reqs)
}

func lastMetaRefID(metaRef string) string {
	return connectsvc.LastMetaRefID(metaRef)
}

func findConnectRequestByID(svc *connectsvc.Service, name string, requestID int) (*connectsvc.Request, error) {
	return connectsvc.FindRequestByID(svc, name, requestID)
}

func connectRequestCreatedAt(req connectsvc.Request) (time.Time, error) {
	return connectsvc.RequestCreatedAt(req)
}

func oldestConnectRequestTime(reqs []connectsvc.Request) (time.Time, error) {
	return connectsvc.OldestRequestTime(reqs)
}

func createImmediateCronTaskFromConnect(db *sql.DB, meta *connectsvc.Meta, reqs []connectsvc.Request, started int) (string, error) {
	if db == nil {
		return "", fmt.Errorf("db is required")
	}
	result, err := connectsvc.CreateImmediateCronTask(meta, reqs, started, func(items []connectsvc.Request) string {
		return buildConnectRequestTaskContent(items)
	}, connectsvc.ImmediateCronTaskPersistence{
		InsertMeta: func(seed *connectsvc.ImmediateCronTaskSeed) (int, error) {
			taskType := sharedutil.NormalizeTaskType(seed.TaskType)
			res, err := db.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, model, thinking, verify, router_disable, cron, content, response_schema, chat_id, task_type) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
				0, seed.RawTime, seed.AgentID, seed.Model, boolToInt(seed.Thinking), boolToInt(seed.Verify), boolToInt(seed.RouterDisable), "", seed.Content, seed.ResponseSchema, seed.ChatID, taskType)
			if err != nil {
				return 0, err
			}
			id, _ := res.LastInsertId()
			return int(id), nil
		},
		AppendMetaLog: func(metaID int, seed *connectsvc.ImmediateCronTaskSeed) {
			appendCronMetaLog(cronDB, cronMetaLogEntry{
				MetaID:         metaID,
				AgentID:        seed.AgentID,
				ChatID:         seed.ChatID,
				TaskType:       sharedutil.NormalizeTaskType(seed.TaskType),
				Action:         "insert",
				Cycle:          0,
				RawTime:        seed.RawTime,
				Model:          seed.Model,
				Thinking:       seed.Thinking,
				Verify:         seed.Verify,
				RouterDisable:  seed.RouterDisable,
				Cron:           "",
				Content:        seed.Content,
				ResponseSchema: seed.ResponseSchema,
				CreatedAt:      cronLogTimestamp(),
			})
		},
		InsertDetail: func(metaID int, chatID string, seed *connectsvc.ImmediateCronTaskSeed) (int, error) {
			taskType := sharedutil.NormalizeTaskType(seed.TaskType)
			res, err := db.Exec(`INSERT OR IGNORE INTO task_detail (meta_id, exec_time, agent_id, chat_id, meta_ref, task_type, model, thinking, verify, router_disable, content, response_schema, started) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				metaID, seed.ExecTime, seed.AgentID, chatID, seed.MetaRef, taskType, seed.Model, boolToInt(seed.Thinking), boolToInt(seed.Verify), boolToInt(seed.RouterDisable), seed.Content, seed.ResponseSchema, seed.Started)
			if err != nil {
				return 0, err
			}
			id, _ := res.LastInsertId()
			return int(id), nil
		},
		AppendDetailLog: func(detailID, metaID int, chatID string, seed *connectsvc.ImmediateCronTaskSeed) {
			appendCronDetailLog(cronDB, cronDetailLogEntry{
				DetailID:       detailID,
				MetaID:         metaID,
				AgentID:        seed.AgentID,
				ChatID:         chatID,
				TaskType:       sharedutil.NormalizeTaskType(seed.TaskType),
				Action:         "insert",
				ExecTime:       seed.ExecTime,
				Model:          seed.Model,
				Thinking:       seed.Thinking,
				Verify:         seed.Verify,
				RouterDisable:  seed.RouterDisable,
				Content:        seed.Content,
				ResponseSchema: seed.ResponseSchema,
				Started:        seed.Started,
				OccurredAt:     cronLogTimestamp(),
			})
		},
	})
	if err != nil || result == nil {
		return "", err
	}
	return result.Task.LastRequestID, nil
}

var runConnectListMeta = func() ([]connectMetaConfigListItem, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve integration executable: %w", err)
	}
	if strings.HasSuffix(strings.TrimSpace(exe), ".test") {
		svc, err := newIntegrationConnectService(integrationDefaultAgentDir())
		if err != nil {
			return nil, err
		}
		defer svc.Close()
		items, err := svc.ListMetaConfig(false)
		if err != nil {
			return nil, err
		}
		out := make([]connectMetaConfigListItem, 0, len(items))
		for _, item := range items {
			out = append(out, connectMetaConfigListItem{
				Key:      item.Key,
				Name:     item.Name,
				Callback: item.Callback,
			})
		}
		return out, nil
	}
	cmd := exec.Command(exe, "connect", "meta-list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) == 0 {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	var items []connectMetaConfigListItem
	if err := json.Unmarshal(output, &items); err != nil {
		return nil, fmt.Errorf("decode integration connect meta-list output: %w", err)
	}
	return items, nil
}

var runConnectPluginAction = func(target, action string, flags map[string]string) (*connectsvc.PluginActionResult, error) {
	return connectsvc.RunPluginAction(target, action, flags)
}

var runConnectPluginStatus = func(target string, flags map[string]string) (*connectsvc.PluginStatus, error) {
	return connectsvc.GetPluginStatus(target, flags)
}

func integrationAssociatedPluginFlags() map[string]string {
	flags := map[string]string{}
	ensureIntegrationPluginConnectBin(flags)
	return flags
}

func startIntegrationAssociatedPluginsAsync(ctx context.Context, plugins []string, logWriter io.Writer, done chan<- struct{}) {
	plugins = append([]string(nil), plugins...)
	if ctx == nil {
		ctx = context.Background()
	}
	go func() {
		if done != nil {
			defer close(done)
		}
		if len(plugins) == 0 {
			integrationStopLog(logWriter, "associated plugins start skipped: none configured")
			return
		}
		startIntegrationAssociatedPluginsWithContext(ctx, plugins, logWriter)
	}()
}

func startIntegrationAssociatedPlugins(plugins []string, logWriter io.Writer) {
	startIntegrationAssociatedPluginsWithContext(context.Background(), plugins, logWriter)
}

func startIntegrationAssociatedPluginsWithContext(ctx context.Context, plugins []string, logWriter io.Writer) {
	for _, plugin := range normalizeIntegrationAssociatedPlugins(plugins) {
		select {
		case <-ctx.Done():
			integrationStopLog(logWriter, "associated plugins start canceled")
			return
		default:
		}
		flags := integrationAssociatedPluginFlags()
		status, err := runConnectPluginStatus(plugin, flags)
		if err == nil && status != nil && status.Started {
			integrationStopLog(logWriter, "associated plugin start skipped: already started target=%s", plugin)
			continue
		}
		if err != nil {
			integrationStopLog(logWriter, "associated plugin status unavailable before start target=%s error=%v", plugin, err)
		}
		select {
		case <-ctx.Done():
			integrationStopLog(logWriter, "associated plugin start canceled target=%s", plugin)
			return
		default:
		}
		result, err := runConnectPluginAction(plugin, "start", flags)
		if err != nil {
			integrationStopLog(logWriter, "associated plugin start failed target=%s error=%v", plugin, err)
			continue
		}
		commandText := ""
		if result != nil {
			commandText = strings.Join(result.Command, " ")
		}
		integrationStopLog(logWriter, "associated plugin started target=%s command=%s", plugin, strings.TrimSpace(commandText))
	}
}

func stopIntegrationAssociatedPlugins(plugins []string, logWriter io.Writer) []string {
	warnings := make([]string, 0)
	for _, plugin := range normalizeIntegrationAssociatedPlugins(plugins) {
		flags := integrationAssociatedPluginFlags()
		status, err := runConnectPluginStatus(plugin, flags)
		if err != nil {
			warning := fmt.Sprintf("associated plugin %s status failed: %v", plugin, err)
			integrationStopLog(logWriter, warning)
			warnings = append(warnings, warning)
			continue
		}
		if status == nil || !status.Started {
			integrationStopLog(logWriter, "associated plugin stop skipped: not started target=%s", plugin)
			continue
		}
		result, err := runConnectPluginAction(plugin, "stop", flags)
		if err != nil {
			warning := fmt.Sprintf("associated plugin %s stop failed: %v", plugin, err)
			integrationStopLog(logWriter, warning)
			warnings = append(warnings, warning)
			continue
		}
		commandText := ""
		if result != nil {
			commandText = strings.Join(result.Command, " ")
		}
		integrationStopLog(logWriter, "associated plugin stopped target=%s command=%s", plugin, strings.TrimSpace(commandText))
	}
	return warnings
}

func stopIntegrationApplicationPlugins(associatedPlugins []string, logWriter io.Writer) []string {
	warnings := stopIntegrationAssociatedPlugins(associatedPlugins, logWriter)
	excluded := make(map[string]struct{}, len(associatedPlugins))
	for _, plugin := range normalizeIntegrationAssociatedPlugins(associatedPlugins) {
		excluded[strings.ToLower(plugin)] = struct{}{}
	}
	return append(warnings, stopIntegrationPluginsExcluding(logWriter, excluded)...)
}

func stopIntegrationPlugins(logWriter io.Writer) []string {
	return stopIntegrationPluginsExcluding(logWriter, nil)
}

func stopIntegrationPluginsExcluding(logWriter io.Writer, excluded map[string]struct{}) []string {
	var warnings []string
	items, err := runConnectListMeta()
	if err != nil {
		warning := fmt.Sprintf("stop plugins warning: connect meta-list failed: %v", err)
		integrationStopLog(logWriter, warning)
		return []string{warning}
	}
	if len(items) == 0 {
		integrationStopLog(logWriter, "plugins stop skipped: no configured plugins")
		return nil
	}

	seen := make(map[string]struct{}, len(items))
	stopped := 0
	failed := 0
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		target := strings.TrimSpace(item.Key)
		if target == "" {
			continue
		}
		if _, skip := excluded[strings.ToLower(target)]; skip {
			integrationStopLog(logWriter, "plugin stop skipped: associated lifecycle manages target=%s", target)
			continue
		}
		dedupeKey := strings.ToLower(target)
		if _, ok := seen[dedupeKey]; ok {
			continue
		}
		seen[dedupeKey] = struct{}{}

		label := name
		if label == "" {
			label = target
		}
		statusFlags := map[string]string{}
		ensureIntegrationPluginConnectBin(statusFlags)
		status, err := runConnectPluginStatus(target, statusFlags)
		if err != nil {
			failed++
			warning := fmt.Sprintf("stop plugin %s status failed: %v", label, err)
			integrationStopLog(logWriter, warning)
			warnings = append(warnings, warning)
			continue
		} else if status == nil || !status.Started {
			integrationStopLog(logWriter, "plugin stop skipped: plugin not started name=%s target=%s", label, target)
			continue
		}
		integrationStopLog(logWriter, "stopping plugin name=%s target=%s", label, target)
		result, err := runConnectPluginAction(target, "stop", map[string]string{})
		if err != nil {
			failed++
			warning := fmt.Sprintf("stop plugin %s failed: %v", label, err)
			integrationStopLog(logWriter, warning)
			warnings = append(warnings, warning)
			continue
		}
		stopped++
		commandText := ""
		if result != nil {
			commandText = strings.Join(result.Command, " ")
		}
		integrationStopLog(logWriter, "plugin stopped name=%s target=%s command=%s", label, target, strings.TrimSpace(commandText))
	}
	integrationStopLog(logWriter, "plugins stop completed count=%d failed=%d", stopped, failed)
	return warnings
}

func integrationStopLog(logWriter io.Writer, format string, args ...interface{}) {
	if logWriter != nil {
		integrationLifecycleLog(logWriter, format, args...)
		return
	}
	log.Printf(format, args...)
}

func listConnectPluginCallbacks() (map[string]string, error) {
	return connectsvc.ListPluginCallbacks(runConnectListMeta, sharedutil.NormalizeTaskType)
}

func connectMetaRuntimeKey(meta connectsvc.Meta) string {
	return connectsvc.PluginRuntimeKey(meta, sharedutil.NormalizeTaskType)
}

func resolveConnectPluginCallback(callbacks map[string]string, pluginName string) (string, error) {
	return connectsvc.ResolvePluginRuntimeCallback(callbacks, pluginName, sharedutil.NormalizeTaskType)
}

var runConnectPluginSend = func(callbackPath string, flags map[string]string) error {
	if flags != nil {
		cloned := make(map[string]string, len(flags))
		for k, v := range flags {
			cloned[k] = v
		}
		cloned["content"] = connectsvc.NormalizePluginReplyContent(flags["content"])
		flags = cloned
	}
	return connectsvc.RunPluginCallbackAction(callbackPath, "send", flags)
}

var runConnectPluginInit = func(callbackPath string, flags map[string]string) error {
	return connectsvc.RunPluginCallbackAction(callbackPath, "init", flags)
}

const defaultConnectAutoReply = connectsvc.DefaultAutoReply

func resolveServeReply(reply string) string {
	return connectsvc.ResolveAutoReply(reply)
}

func parseTopLevelFlags(args []string) map[string]string {
	flags := make(map[string]string)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		key := strings.TrimPrefix(arg, "--")
		value := "true"
		if parts := strings.SplitN(key, "=", 2); len(parts) == 2 {
			key = parts[0]
			value = parts[1]
		} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			value = args[i+1]
			i++
		}
		flags[key] = value
	}
	return flags
}

func mergeIntegrationRuntimeFlags(flags map[string]string) map[string]string {
	merged := make(map[string]string, len(flags))
	for key, value := range flags {
		merged[key] = value
	}
	startupCfg, _, startupErr := readIntegrationStartupConfig()
	if startupCfg == nil {
		return merged
	}
	if startupErr != nil {
		return merged
	}
	for _, key := range []string{
		"host",
		"agent-dir",
		"default-dir",
		"device",
		"agent-cache",
		"connect-cache",
		"site",
		"connect_timeout",
		"reply",
		"sleep",
		"thread",
		"queue",
		"retry_interval",
		"retry_times",
		"idle_timeout",
		"pid-file",
		"log-file",
	} {
		if strings.TrimSpace(merged[key]) != "" {
			continue
		}
		if value, ok := integrationStartupConfigValue(startupCfg, key); ok {
			merged[key] = value
		}
	}
	return merged
}

func integrationDefaultFileInCurrentDir(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	return integrationResolveAppPath(name)
}

func integrationPIDFilePath(flags map[string]string) string {
	path := strings.TrimSpace(firstNonEmpty(flags["pid-file"]))
	if path == "" {
		if value, ok := readIntegrationStartupConfigValue("pid-file"); ok {
			path = integrationResolveAppPath(value)
		}
	}
	if path == "" {
		path = integrationDefaultFileInCurrentDir(integrationDefaultPIDFile)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func integrationLogFilePath(flags map[string]string) string {
	path := strings.TrimSpace(firstNonEmpty(flags["log-file"]))
	if path == "" {
		if value, ok := readIntegrationStartupConfigValue("log-file"); ok {
			path = integrationResolveAppPath(value)
		}
	}
	if path == "" {
		path = integrationDefaultFileInCurrentDir(integrationDefaultLogFile)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func openIntegrationLifecycleLog(flags map[string]string) (io.WriteCloser, string, error) {
	path := integrationLogFilePath(flags)
	writer, err := newIntegrationDailyLogWriter(path, integrationLogRetentionDays)
	if err != nil {
		return nil, path, err
	}
	return writer, path, nil
}

func integrationLifecycleLog(writer io.Writer, format string, args ...interface{}) {
	if writer == nil {
		return
	}
	_, _ = fmt.Fprintf(writer, "%s [integration lifecycle] %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(format, args...))
}

type integrationDailyLogWriter struct {
	mu         sync.Mutex
	basePath   string
	keepDays   int
	now        func() time.Time
	currentDay string
	file       *os.File
}

func newIntegrationDailyLogWriter(path string, keepDays int) (*integrationDailyLogWriter, error) {
	return newIntegrationDailyLogWriterWithClock(path, keepDays, time.Now)
}

func newIntegrationDailyLogWriterWithClock(path string, keepDays int, now func() time.Time) (*integrationDailyLogWriter, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("log path is required")
	}
	if keepDays <= 0 {
		keepDays = 1
	}
	if now == nil {
		now = time.Now
	}
	writer := &integrationDailyLogWriter{
		basePath: path,
		keepDays: keepDays,
		now:      now,
	}
	if err := writer.ensureFileLocked(now()); err != nil {
		return nil, err
	}
	return writer, nil
}

func (w *integrationDailyLogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.ensureFileLocked(w.now()); err != nil {
		return 0, err
	}
	return w.file.Write(p)
}

func (w *integrationDailyLogWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *integrationDailyLogWriter) ensureFileLocked(now time.Time) error {
	today := now.Format("2006-01-02")
	if w.file != nil && w.currentDay == today {
		return nil
	}
	if err := ensureParentDir(w.basePath); err != nil {
		return err
	}
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return err
		}
		w.file = nil
		if w.currentDay != "" && w.currentDay != today {
			if err := archiveIntegrationBaseLog(w.basePath, w.currentDay); err != nil {
				return err
			}
		}
		w.currentDay = ""
	}
	if err := rotateStaleIntegrationBaseLog(w.basePath, today); err != nil {
		return err
	}
	file, err := os.OpenFile(w.basePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	w.file = file
	w.currentDay = today
	return cleanupExpiredIntegrationLogArchives(w.basePath, today, w.keepDays)
}

func rotateStaleIntegrationBaseLog(basePath, today string) error {
	info, err := os.Stat(basePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("log path is a directory: %s", basePath)
	}
	day := info.ModTime().Format("2006-01-02")
	if day == today {
		return nil
	}
	return archiveIntegrationBaseLog(basePath, day)
}

func archiveIntegrationBaseLog(basePath, day string) error {
	basePath = strings.TrimSpace(basePath)
	day = strings.TrimSpace(day)
	if basePath == "" || day == "" {
		return nil
	}
	if _, err := os.Stat(basePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	archivePath := integrationArchivedLogPath(basePath, day)
	if _, err := os.Stat(archivePath); err == nil {
		if err := appendFileContents(archivePath, basePath); err != nil {
			return err
		}
		return os.Remove(basePath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(basePath, archivePath)
}

func appendFileContents(targetPath, sourcePath string) error {
	src, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(targetPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func cleanupExpiredIntegrationLogArchives(basePath, today string, keepDays int) error {
	if keepDays <= 0 {
		keepDays = 1
	}
	todayTime, err := time.Parse("2006-01-02", today)
	if err != nil {
		return err
	}
	cutoff := todayTime.AddDate(0, 0, -(keepDays - 1))
	dir := filepath.Dir(basePath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		archiveDay, ok := integrationArchivedLogDay(basePath, entry.Name())
		if !ok {
			continue
		}
		archiveTime, err := time.Parse("2006-01-02", archiveDay)
		if err != nil {
			continue
		}
		if archiveTime.Before(cutoff) {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	return nil
}

func integrationArchivedLogPath(basePath, day string) string {
	ext := filepath.Ext(basePath)
	base := strings.TrimSuffix(basePath, ext)
	return base + "-" + day + ext
}

func integrationArchivedLogDay(basePath, entryName string) (string, bool) {
	baseName := filepath.Base(basePath)
	ext := filepath.Ext(baseName)
	stem := strings.TrimSuffix(baseName, ext)
	prefix := stem + "-"
	suffix := ext
	if !strings.HasPrefix(entryName, prefix) {
		return "", false
	}
	if suffix != "" {
		if !strings.HasSuffix(entryName, suffix) {
			return "", false
		}
		entryName = strings.TrimSuffix(entryName, suffix)
	}
	day := strings.TrimPrefix(entryName, prefix)
	if len(day) != len("2006-01-02") {
		return "", false
	}
	if _, err := time.Parse("2006-01-02", day); err != nil {
		return "", false
	}
	return day, true
}

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0o755)
}

func integrationStartupDir(pidFile string) string {
	if appDir := integrationAppDir(); appDir != "" {
		return appDir
	}
	pidFile = strings.TrimSpace(pidFile)
	if pidFile != "" {
		if abs, err := filepath.Abs(filepath.Dir(pidFile)); err == nil {
			return abs
		}
		return filepath.Clean(filepath.Dir(pidFile))
	}
	if cwd, err := os.Getwd(); err == nil && strings.TrimSpace(cwd) != "" {
		return filepath.Clean(cwd)
	}
	return ""
}

func cleanupStartupPIDFiles(pidFile string) ([]string, error) {
	startupDir := strings.TrimSpace(integrationStartupDir(pidFile))
	if startupDir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(startupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	targetPID := strings.TrimSpace(filepath.Base(pidFile))
	removed := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if !strings.HasSuffix(strings.ToLower(name), ".pid") {
			continue
		}
		if name == "" || name == targetPID {
			continue
		}
		path := filepath.Join(startupDir, name)
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return removed, err
		}
		removed = append(removed, path)
	}
	sort.Strings(removed)
	return removed, nil
}

func writePIDFile(path string) error {
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o644)
}

func integrationStartupStatusFilePath(pidFile string) string {
	pidFile = strings.TrimSpace(pidFile)
	if pidFile == "" {
		pidFile = integrationDefaultPIDFile
	}
	return pidFile + ".startup.json"
}

func writeIntegrationStartupStatus(path string, status integrationStartupStatus) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if err := ensureParentDir(path); err != nil {
		return err
	}
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func readIntegrationStartupStatus(path string) (integrationStartupStatus, error) {
	var status integrationStartupStatus
	path = strings.TrimSpace(path)
	if path == "" {
		return status, os.ErrNotExist
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return status, err
	}
	if err := json.Unmarshal(data, &status); err != nil {
		return status, err
	}
	return status, nil
}

func integrationReady(port string) bool {
	port = strings.TrimSpace(port)
	if port == "" {
		port = "8080"
	}
	client := http11client.NewClient(http11client.Options{Timeout: 300 * time.Millisecond})
	resp, err := client.Get("http://127.0.0.1:" + port + integrationReadyPath)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func integrationBrowserEntryReady(port int) bool {
	client := http11client.NewClient(http11client.Options{Timeout: integrationBrowserEntryProbeTimeout})
	resp, err := client.Get(integrationBrowserEntryURL(port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func readRunningPID(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return 0, false
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return 0, false
	}
	return pid, true
}

func integrationProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func rebuildTopLevelFlags(flags map[string]string, skip map[string]struct{}) []string {
	keys := make([]string, 0, len(flags))
	for key := range flags {
		if _, ok := skip[key]; ok {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys)*2)
	for _, key := range keys {
		value := flags[key]
		switch key {
		case "default-dir", "site", "pid-file", "log-file":
			value = integrationRuntimePathArg(value)
		}
		out = append(out, "--"+key, value)
	}
	return out
}

func signalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-signals:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, func() {
		signal.Stop(signals)
		cancel()
	}
}

func notifyConnectPluginStarted(meta connectsvc.Meta, request connectsvc.Request) error {
	return connectsvc.NotifyPluginStarted(nil, meta, request, resolveServeReply(""), connectsvc.PluginCallbackRuntimeOptions{
		ListMeta:     runConnectListMeta,
		NormalizeKey: sharedutil.NormalizeTaskType,
		ResolveConnectBin: func() string {
			return firstNonEmpty(resolveExecutablePath(os.Args[0]), "integration")
		},
		Init:   runConnectPluginInit,
		Logger: log.Printf,
	})
}

func notifyConnectPluginStartedWithReply(meta connectsvc.Meta, request connectsvc.Request, reply string) error {
	return connectsvc.NotifyPluginStarted(nil, meta, request, resolveServeReply(reply), connectsvc.PluginCallbackRuntimeOptions{
		ListMeta:     runConnectListMeta,
		NormalizeKey: sharedutil.NormalizeTaskType,
		ResolveConnectBin: func() string {
			return firstNonEmpty(resolveExecutablePath(os.Args[0]), "integration")
		},
		Init:   runConnectPluginInit,
		Logger: log.Printf,
	})
}

func extractSSEAssistantText(payload []byte) string {
	return connectsvc.ExtractSSEAssistantText(payload)
}

func normalizePluginReplyContent(raw string) string {
	return connectsvc.NormalizePluginReplyContent(raw)
}

func notifyCompletedConnectTasks(cfg *Config, svc *connectsvc.Service) {
	if cfg == nil || cronDB == nil || svc == nil {
		return
	}
	connectsvc.DispatchCompletedPluginReplies(nil, cronDB, svc, sharedutil.DefaultTaskType, time.Now().Add(-24*time.Hour).Unix(), func(detailID int) error {
		_, err := cronDB.Exec(`UPDATE task_detail SET replied_at = ? WHERE id = ?`, time.Now().Format(time.RFC3339), detailID)
		return err
	}, connectsvc.PluginCallbackRuntimeOptions{
		ListMeta:     runConnectListMeta,
		NormalizeKey: sharedutil.NormalizeTaskType,
		ResolveConnectBin: func() string {
			return firstNonEmpty(resolveExecutablePath(os.Args[0]), "integration")
		},
		Send:   runConnectPluginSend,
		Logger: log.Printf,
	})
}

func syncConnectPendingRequests(cfg *Config, svc *connectsvc.Service) {
	if cronDB == nil || svc == nil {
		return
	}
	policy := connectsvc.PendingRequestPolicy{
		MinNotifyAge:       0,
		ExpireAge:          20 * time.Minute,
		PendingDetailState: 0,
		ExpiredDetailState: 2,
	}
	connectsvc.SyncPendingPluginRuntime(nil, connectsvc.PendingPluginRuntimeOptions{
		Service:          svc,
		QueryDB:          cronDB,
		Logger:           log.Printf,
		BuildTextContent: buildConnectRequestTextContent,
		DecidePlan: func(meta connectsvc.Meta, reqs []connectsvc.Request, now time.Time, textContent string) (*connectsvc.PendingRequestPlan, error) {
			return connectsvc.DecidePendingPluginPlan(meta, reqs, now, textContent, policy)
		},
		CreateTask: func(meta *connectsvc.Meta, reqs []connectsvc.Request, detailStarted int) (any, string, error) {
			lastRequestID, err := createImmediateCronTaskFromConnect(cronDB, meta, reqs, detailStarted)
			return nil, lastRequestID, err
		},
		NotifyStarted: func(_ map[string]string, meta connectsvc.Meta, request connectsvc.Request) error {
			if sharedutil.NormalizeTaskType(firstNonEmpty(meta.Key, meta.Name)) == "feishu" {
				return nil
			}
			return notifyConnectPluginStartedWithReply(meta, request, cfg.Reply)
		},
		DispatchCompletedReplies: func(_ map[string]string, svc *connectsvc.Service) {
			notifyCompletedConnectTasks(cfg, svc)
		},
	})
}

func cronExecuteOnce(cfg *Config, proxyClient *http.Client, connectSvc *connectsvc.Service) {
	if cronDB == nil || connectSvc == nil {
		return
	}
	cleanupDeletedAgentCronData(cfg)
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	clearDrainedCronAgentStopResumeWindows(cronDB, now.Unix())

	selectMetaRef := `''`
	if hasTableColumn(cronDB, "task_detail", "meta_ref") {
		selectMetaRef = `d.meta_ref`
	}
	rows, err := cronDB.Query(fmt.Sprintf(`SELECT d.id, d.meta_id, d.exec_time, d.agent_id, d.chat_id, %s, d.task_type, d.model, d.thinking, %s, %s, d.content, d.response_schema
		FROM task_detail d
		WHERE d.started = 0
		  AND d.exec_time <= ?
		  AND (
			d.exec_time >= ?
			OR EXISTS (
				SELECT 1 FROM agent_cron_stop stop
				WHERE stop.agent_id = d.agent_id
				  AND stop.enabled = 0
				  AND stop.resume_from > 0
				  AND d.task_type = 'cron'
				  AND d.exec_time >= stop.resume_from
			)
		  )`, selectMetaRef, cronVerifySelect(cronDB, "task_detail"), cronRouterDisableSelect(cronDB, "task_detail")), now.Unix(), oneHourAgo.Unix())
	if err != nil {
		return
	}
	defer rows.Close()

	type dueTask struct {
		ID             int
		MetaID         int
		AgentID        string
		ChatID         string
		MetaRef        string
		TaskType       string
		Model          string
		Thinking       bool
		Verify         bool
		RouterDisable  bool
		Content        string
		ResponseSchema string
	}
	var tasks []dueTask
	for rows.Next() {
		var t dueTask
		var execTime int64
		var th, verify, routerDisable int
		rows.Scan(&t.ID, &t.MetaID, &execTime, &t.AgentID, &t.ChatID, &t.MetaRef, &t.TaskType, &t.Model, &th, &verify, &routerDisable, &t.Content, &t.ResponseSchema)
		t.TaskType = sharedutil.NormalizeTaskType(t.TaskType)
		t.Thinking = th != 0
		t.Verify = verify != 0
		t.RouterDisable = routerDisable != 0
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return
	}
	if err := rows.Close(); err != nil {
		return
	}
	agents := make([]string, 0, len(tasks))
	for _, task := range tasks {
		if task.TaskType == "cron" {
			agents = append(agents, task.AgentID)
		}
	}
	stopStates, err := getCronAgentStopStateMap(cronDB, agents)
	if err != nil {
		return
	}

	for _, t := range tasks {
		if t.TaskType == "cron" && stopStates[t.AgentID].Enabled {
			// STOP is an Agent-level runtime gate. Keep task metadata and detail
			// untouched so the pending task can run after the Agent is resumed.
			continue
		}
		if !agentExists(cfg.AgentDir, cfg.effectiveDeviceID(), t.AgentID) {
			appendCronDetailLog(cronDB, cronDetailLogEntry{
				DetailID:       t.ID,
				MetaID:         t.MetaID,
				AgentID:        t.AgentID,
				ChatID:         t.ChatID,
				TaskType:       t.TaskType,
				Action:         "skip_deleted_agent",
				ExecTime:       now.Unix(),
				Model:          t.Model,
				Thinking:       t.Thinking,
				Verify:         t.Verify,
				RouterDisable:  t.RouterDisable,
				Content:        t.Content,
				ResponseSchema: t.ResponseSchema,
				Started:        0,
				OccurredAt:     cronLogTimestamp(),
			})
			continue
		}
		modelReady, err := isCronModelConfigured(cronDB, t.Model)
		if err != nil {
			log.Printf("[cron] validate due task model failed: detail=%d model=%s err=%v", t.ID, t.Model, err)
			continue
		}
		if !modelReady {
			appendCronDetailLog(cronDB, cronDetailLogEntry{
				DetailID:       t.ID,
				MetaID:         t.MetaID,
				AgentID:        t.AgentID,
				ChatID:         t.ChatID,
				TaskType:       t.TaskType,
				Action:         "skip_invalid_model",
				ExecTime:       now.Unix(),
				Model:          t.Model,
				Thinking:       t.Thinking,
				Verify:         t.Verify,
				RouterDisable:  t.RouterDisable,
				Content:        t.Content,
				ResponseSchema: t.ResponseSchema,
				Started:        0,
				OccurredAt:     cronLogTimestamp(),
			})
			continue
		}
		// Mark as started
		cronDB.Exec(`UPDATE task_detail SET started = 1 WHERE id = ?`, t.ID)
		logCronDetailStatusByID(cronDB, t.ID, "update_started")

		chatID := strings.TrimSpace(t.ChatID)
		if chatID == "" {
			chatID = fmt.Sprintf("%d@%d", t.MetaID, t.ID)
			cronDB.Exec(`UPDATE task_detail SET chat_id = ? WHERE id = ?`, chatID, t.ID)
			logCronDetailStatusByID(cronDB, t.ID, "update_chat_id")
		}

		// Build metadata
		agentTTL := time.Duration(cfg.AgentCacheMs) * time.Millisecond
		metadata, err := getAgentOutputForChat(cfg.AgentDir, cfg.effectiveDeviceID(), agentTTL, chatID)
		if err != nil {
			cronDB.Exec(`UPDATE task_detail SET started = 0 WHERE id = ?`, t.ID)
			logCronDetailStatusByID(cronDB, t.ID, "reset_started")
			continue
		}
		metaBytes, _ := json.Marshal(metadata)
		var metaMap map[string]interface{}
		json.Unmarshal(metaBytes, &metaMap)
		metaMap["port"] = integrationMetadataPort(cfg)
		metaMap["agentId"] = t.AgentID
		metaMap["chat"] = chatID
		metaMap["type"] = chatTypeScheduledTask
		metaMap["cron_type"] = sharedutil.NormalizeTaskType(t.TaskType)
		metaMap["thinking"] = t.Thinking
		metaMap["verify"] = t.Verify
		metaMap["router_disable"] = t.RouterDisable
		injectLiveAgentMediaIntoAgentList(metaMap)
		injectLiveAgentKnowledgeIntoAgentList(metaMap)
		injectLastResponseMetadata(metaMap, chatID)
		injectSandboxPathMetadata(metaMap, chatID)
		injectPluginsDirMetadata(metaMap)
		if routerRemote := readAgentRouterRemote(cfg.AgentDir, cfg.effectiveDeviceID(), t.AgentID, agentTTL); len(routerRemote) > 0 {
			metaMap["router_remote"] = routerRemote
		}
		if metaID := lastMetaRefID(t.MetaRef); metaID != "" {
			metaMap["META_ID"] = metaID
		}
		pruneForwardedKnowledgeCommit(metaMap)
		modelCfg, err := lookupTokenConfigByModel(cronDB, t.Model)
		if err != nil {
			log.Printf("[cron] read task model config failed: detail=%d model=%s err=%v", t.ID, t.Model, err)
			cronDB.Exec(`UPDATE task_detail SET started = 0 WHERE id = ?`, t.ID)
			logCronDetailStatusByID(cronDB, t.ID, "reset_started")
			continue
		}
		// Scheduled and Connect-triggered tasks bypass handleChatCompletions, so
		// apply the same current model configuration used by centered chat here.
		injectConfiguredModelMetadata(metaMap, modelCfg)

		reqData := map[string]interface{}{
			"model":    t.Model,
			"messages": []map[string]string{{"role": "user", "content": t.Content}},
			"stream":   true,
			"metadata": metaMap,
		}
		if strings.TrimSpace(t.ResponseSchema) != "" {
			metaMap["response_schema"] = t.ResponseSchema
			reqData["response_format"] = map[string]interface{}{
				"type": "json_schema",
				"json_schema": map[string]interface{}{
					"name":   "scheduled_task_response",
					"schema": json.RawMessage(t.ResponseSchema),
				},
			}
		}
		syncForwardedChatRequestFlags(reqData, metaMap)
		body, _ := json.Marshal(reqData)

		// Log Q to chat_log
		appendChatLogDB(t.AgentID, chatID, chatTypeScheduledTask, "Q", "normal", string(body))

		token := modelCfg.Token

		// Send to upstream
		detailID := t.ID
		go func(agentID, cID string, reqBody []byte, dID int, token, taskType string) {
			targetURL := strings.TrimRight(cfg.currentHost(), "/") + "/v1/chat/completions"
			req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewReader(reqBody))
			if err != nil {
				cronDB.Exec(`UPDATE task_detail SET started = 0 WHERE id = ?`, dID)
				logCronDetailStatusByID(cronDB, dID, "reset_started")
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "text/event-stream")
			if token != "" {
				req.Header.Set("Authorization", token)
			}

			resp, err := proxyClient.Do(req)
			if err != nil {
				cronDB.Exec(`UPDATE task_detail SET started = 0 WHERE id = ?`, dID)
				logCronDetailStatusByID(cronDB, dID, "reset_started")
				return
			}
			defer resp.Body.Close()

			payload, readErr := io.ReadAll(resp.Body)
			if len(payload) > 0 {
				appendChatLogDB(agentID, cID, chatTypeScheduledTask, "A", detectResponseType(string(payload)), string(payload))
			}
			abnormalStream := readErr != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 || detectResponseType(string(payload)) == "abnormal"
			if readErr != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
				cronDB.Exec(`UPDATE task_detail SET started = 0 WHERE id = ?`, dID)
				logCronDetailStatusByID(cronDB, dID, "reset_started")
				sendSSECompletionNotification(chatTypeScheduledTask, taskType, "", abnormalStream)
				return
			}
			resultContent := extractSSEAssistantText(payload)
			cronDB.Exec(`UPDATE task_detail SET started = 3, result_content = ? WHERE id = ?`, resultContent, dID)
			logCronDetailStatusByID(cronDB, dID, "complete")
			sendSSECompletionNotification(chatTypeScheduledTask, taskType, "", abnormalStream)
		}(t.AgentID, chatID, body, detailID, token, t.TaskType)
	}
}

var cronLastCheck time.Time
var cronDB *sql.DB
var integrationLogRetentionManager = logretention.NewManager()

func initCronDB() {
	var err error
	dbPath := resolveIntegrationDBPath()
	if err := ensureParentDir(dbPath); err != nil {
		log.Printf("[cron] failed to prepare db dir: %v", err)
		return
	}
	cronDB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Printf("[cron] failed to open data: %v", err)
		return
	}
	cronDB.SetMaxOpenConns(10)
	cronDB.SetMaxIdleConns(10)
	cronDB.SetConnMaxLifetime(0)
	cronDB.Exec(`PRAGMA journal_mode=WAL`)
	ensureCronSchema(cronDB)
	if err := ensureTokenStore(cronDB); err != nil {
		log.Printf("[cron] ensure token store failed: %v", err)
	}
	cronDB.Exec(`CREATE TABLE IF NOT EXISTS chat_log (id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL, chat_type TEXT NOT NULL DEFAULT 'page_session', role TEXT NOT NULL, response_type TEXT NOT NULL DEFAULT 'normal', content TEXT NOT NULL, created_at TEXT NOT NULL)`)
	cronDB.Exec(`ALTER TABLE chat_log ADD COLUMN chat_type TEXT NOT NULL DEFAULT 'page_session'`)
	cronDB.Exec(`ALTER TABLE chat_log ADD COLUMN response_type TEXT NOT NULL DEFAULT 'normal'`)
	cronDB.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_agent_chat ON chat_log(agent_id, chat_id)`)
	cronDB.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_agent_chat_time ON chat_log(agent_id, chat_id, created_at)`)
	cronDB.Exec(`CREATE TABLE IF NOT EXISTS agent_message_log (id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL DEFAULT '', chat_id TEXT NOT NULL DEFAULT '', content TEXT NOT NULL, log_type INTEGER NOT NULL, created_at TEXT NOT NULL)`)
	cronDB.Exec(`CREATE INDEX IF NOT EXISTS idx_agent_message_log_chat_type_time ON agent_message_log(chat_id, log_type, created_at)`)
	cronDB.Exec(`CREATE INDEX IF NOT EXISTS idx_agent_message_log_agent_chat_type_time ON agent_message_log(agent_id, chat_id, log_type, created_at)`)
	cronDB.Exec(`CREATE INDEX IF NOT EXISTS idx_agent_message_log_agent_chat_time ON agent_message_log(agent_id, chat_id, created_at)`)
	cronDB.Exec(`CREATE TABLE IF NOT EXISTS cmd_log (id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL, tid TEXT NOT NULL, cmd TEXT NOT NULL, result TEXT NOT NULL DEFAULT '', status INTEGER NOT NULL DEFAULT -1, received_at TEXT NOT NULL, completed_at TEXT NOT NULL DEFAULT '')`)
	cronDB.Exec(`CREATE INDEX IF NOT EXISTS idx_cmd_agent_chat ON cmd_log(agent_id, chat_id)`)
	cronDB.Exec(`CREATE INDEX IF NOT EXISTS idx_cmd_agent_chat_time ON cmd_log(agent_id, chat_id, received_at)`)
}

func startIntegrationLogRetentionCleanup(ctx context.Context) {
	if cronDB == nil {
		integrationLogRetentionManager.MarkChecked(fmt.Errorf("log retention db not ready"))
		return
	}
	dbPath := resolveIntegrationDBPath()
	integrationLogRetentionManager.StartAsyncWithDBOpener(ctx, logretention.DefaultRetentionDays, func() (*sql.DB, error) {
		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			return nil, err
		}
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		db.SetConnMaxLifetime(0)
		if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
			_ = db.Close()
			return nil, err
		}
		return db, nil
	}, log.Printf)
}

func cronCheckOnce(cfg *Config) {
	if cronDB == nil || cfg == nil {
		return
	}
	cleanupInvalidCronData(cfg)
	now := time.Now()
	end := now.Add(5 * 24 * time.Hour)
	cronLastCheck = now

	rows, err := cronDB.Query(fmt.Sprintf(`SELECT id, cycle, raw_time, agent_id, model, thinking, %s, %s, cron, content, chat_id, task_type FROM task_meta WHERE cycle != 0`, cronVerifySelect(cronDB, "task_meta"), cronRouterDisableSelect(cronDB, "task_meta")))
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, cycle, th, verify, routerDisable int
		var rawTime, agentID, model, cronExpr, content, chatID, taskType string
		rows.Scan(&id, &cycle, &rawTime, &agentID, &model, &th, &verify, &routerDisable, &cronExpr, &content, &chatID, &taskType)
		taskType = sharedutil.NormalizeTaskType(taskType)

		if cycle == -1 || rawTime == "" {
			// Custom cron expression
			expr := strings.ReplaceAll(cronExpr, "?", "*")
			parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
			sched, err := parser.Parse(expr)
			if err != nil {
				continue
			}
			next := sched.Next(now.Add(-1 * time.Second))
			for !next.After(end) && !next.IsZero() {
				res, _ := cronDB.Exec(`INSERT OR IGNORE INTO task_detail (meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, verify, router_disable, content, started) VALUES (?,?,?,?,?,?,?,?,?,?,0)`,
					id, next.Unix(), agentID, chatID, taskType, model, th, verify, routerDisable, content)
				if res != nil {
					if n, _ := res.RowsAffected(); n > 0 {
						if detailID, err := res.LastInsertId(); err == nil {
							appendCronDetailLog(cronDB, cronDetailLogEntry{DetailID: int(detailID), MetaID: id, AgentID: agentID, ChatID: chatID, TaskType: taskType, Action: "insert", ExecTime: next.Unix(), Model: model, Thinking: th != 0, Verify: verify != 0, RouterDisable: routerDisable != 0, Content: content, Started: 0, OccurredAt: cronLogTimestamp()})
						}
					}
				}
				next = sched.Next(next)
			}
		} else {
			t, err := time.ParseInLocation("2006-01-02 15:04", rawTime, time.Local)
			if err != nil {
				continue
			}
			switch cycle {
			case 1: // workday
				day := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, time.Local)
				for !day.After(end) {
					if !day.Before(now) {
						wd := day.Weekday()
						if wd >= time.Monday && wd <= time.Friday {
							res, _ := cronDB.Exec(`INSERT OR IGNORE INTO task_detail (meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, verify, router_disable, content, started) VALUES (?,?,?,?,?,?,?,?,?,?,0)`,
								id, day.Unix(), agentID, chatID, taskType, model, th, verify, routerDisable, content)
							if res != nil {
								if n, _ := res.RowsAffected(); n > 0 {
									if detailID, err := res.LastInsertId(); err == nil {
										appendCronDetailLog(cronDB, cronDetailLogEntry{DetailID: int(detailID), MetaID: id, AgentID: agentID, ChatID: chatID, TaskType: taskType, Action: "insert", ExecTime: day.Unix(), Model: model, Thinking: th != 0, Verify: verify != 0, RouterDisable: routerDisable != 0, Content: content, Started: 0, OccurredAt: cronLogTimestamp()})
									}
								}
							}
						}
					}
					day = day.AddDate(0, 0, 1)
				}
			case 2: // daily
				day := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, time.Local)
				for !day.After(end) {
					if !day.Before(now) {
						res, _ := cronDB.Exec(`INSERT OR IGNORE INTO task_detail (meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, verify, router_disable, content, started) VALUES (?,?,?,?,?,?,?,?,?,?,0)`,
							id, day.Unix(), agentID, chatID, taskType, model, th, verify, routerDisable, content)
						if res != nil {
							if n, _ := res.RowsAffected(); n > 0 {
								if detailID, err := res.LastInsertId(); err == nil {
									appendCronDetailLog(cronDB, cronDetailLogEntry{DetailID: int(detailID), MetaID: id, AgentID: agentID, ChatID: chatID, TaskType: taskType, Action: "insert", ExecTime: day.Unix(), Model: model, Thinking: th != 0, Verify: verify != 0, RouterDisable: routerDisable != 0, Content: content, Started: 0, OccurredAt: cronLogTimestamp()})
								}
							}
						}
					}
					day = day.AddDate(0, 0, 1)
				}
			case 3, 4, 5: // hourly / every 15 minutes / every 30 minutes
				interval := cronCycleInterval(cycle)
				tick := t
				if tick.Before(now) {
					steps := now.Sub(t) / interval
					tick = t.Add(steps * interval)
					if tick.Before(now) {
						tick = tick.Add(interval)
					}
				}
				for !tick.After(end) {
					res, _ := cronDB.Exec(`INSERT OR IGNORE INTO task_detail (meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, verify, router_disable, content, started) VALUES (?,?,?,?,?,?,?,?,?,?,0)`,
						id, tick.Unix(), agentID, chatID, taskType, model, th, verify, routerDisable, content)
					if res != nil {
						if n, _ := res.RowsAffected(); n > 0 {
							if detailID, err := res.LastInsertId(); err == nil {
								appendCronDetailLog(cronDB, cronDetailLogEntry{DetailID: int(detailID), MetaID: id, AgentID: agentID, ChatID: chatID, TaskType: taskType, Action: "insert", ExecTime: tick.Unix(), Model: model, Thinking: th != 0, Verify: verify != 0, RouterDisable: routerDisable != 0, Content: content, Started: 0, OccurredAt: cronLogTimestamp()})
							}
						}
					}
					tick = tick.Add(interval)
				}
			}
		}
	}
}

func main() {
	args := normalizeIntegrationLaunchArgs(os.Args[1:])
	if shouldHandleIntegrationBundleAppLaunch(args) {
		os.Exit(handleIntegrationBundleAppLaunch(os.Stderr))
	}
	if shouldPrepareIntegrationRuntimeLayout(args) {
		if err := prepareIntegrationRuntimeLayout(); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}
	if len(args) > 0 {
		switch args[0] {
		case "--help", "-h", "help":
			printCLIHelp()
			return
		case "start":
			os.Exit(startIntegrationProcess(parseTopLevelFlags(args[1:]), os.Stdout, os.Stderr))
		case "stop":
			os.Exit(stopIntegrationProcess(parseTopLevelFlags(args[1:]), os.Stdout, os.Stderr))
		case "restart":
			os.Exit(restartIntegrationProcess(parseTopLevelFlags(args[1:]), os.Stdout, os.Stderr))
		case "splash":
			os.Exit(runIntegrationSplashCLI(args[1:], os.Stderr))
		case "serve":
			os.Exit(runIntegrationForeground(args[1:], os.Stderr))
		case "cron":
			runIntegrationCronCLI(args[1:])
			return
		case "knowledge":
			runIntegrationKnowledgeCLI(args[1:])
			return
		case "host":
			os.Exit(runIntegrationHostCLI(args[1:], os.Stdout, os.Stderr))
		case "standalone":
			os.Exit(runIntegrationStandaloneCLI(args[1:], os.Stdout, os.Stderr))
		case "agent":
			os.Exit(runIntegrationAgentCLI(args[1:], os.Stdout, os.Stderr))
		case "message-insert":
			runIntegrationMessageInsertCLI(args[1:])
		case "sandbox":
			runIntegrationSandboxCLI(args[1:])
			return
		case "token":
			runIntegrationTokenCLI(args[1:])
			return
		case "connect":
			runIntegrationConnectCLI(args[1:])
			return
		case "api", "service":
			os.Exit(runIntegrationAPICLI(args[1:], os.Stdout, os.Stderr))
		case "plugins":
			runIntegrationPluginCLI(args[1:])
			return
		case "notify":
			runIntegrationNotifyCLI(args[1:])
			return
		case "log-round":
			cfg := &Config{AgentDir: integrationDefaultAgentDir(), AgentCacheMs: 120000}
			if value, ok := readIntegrationStartupConfigValue("agent-dir"); ok {
				cfg.AgentDir = firstNonEmpty(value, cfg.AgentDir)
			}
			if value, ok := readIntegrationStartupConfigValue("device"); ok {
				cfg.Device = value
			}
			if raw, ok := readIntegrationStartupConfigValue("agent-cache"); ok {
				if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
					cfg.AgentCacheMs = parsed
				}
			}
			runIntegrationLogRoundCLI(args[1:], cfg)
			return
		case "skills-warning":
			agentDir := integrationDefaultAgentDir()
			if value, ok := readIntegrationStartupConfigValue("agent-dir"); ok {
				agentDir = firstNonEmpty(value, agentDir)
			}
			runIntegrationSkillsWarningCLI(args[1:], &Config{AgentDir: agentDir})
			return
		case "file-last-update":
			cfg := &Config{AgentDir: integrationDefaultAgentDir(), AgentCacheMs: 120000}
			if value, ok := readIntegrationStartupConfigValue("agent-dir"); ok {
				cfg.AgentDir = firstNonEmpty(value, cfg.AgentDir)
			}
			if value, ok := readIntegrationStartupConfigValue("device"); ok {
				cfg.Device = value
			}
			if raw, ok := readIntegrationStartupConfigValue("agent-cache"); ok {
				if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
					cfg.AgentCacheMs = parsed
				}
			}
			runIntegrationFileLastUpdateCLI(args[1:], cfg)
			return
		case "backup-clean":
			os.Exit(runIntegrationBackupCleanCLI(args[1:], os.Stdout, os.Stderr))
		}
		if !strings.HasPrefix(args[0], "-") && isIntegrationConnectSubcommand(args[0]) {
			runIntegrationConnectCLI(args)
			return
		}
	}
	os.Exit(runIntegrationForeground(args, os.Stderr))
}

func shouldPrepareIntegrationRuntimeLayout(args []string) bool {
	if len(args) == 0 {
		return true
	}
	switch strings.TrimSpace(args[0]) {
	case "--help", "-h", "help", "backup-clean", "file-last-update":
		return false
	default:
		return true
	}
}

func normalizeIntegrationLaunchArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "-psn_") {
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered
}

func shouldHandleIntegrationBundleAppLaunch(args []string) bool {
	return shouldHandleIntegrationBundleAppLaunchWithContext(args, strings.TrimSpace(os.Getenv("__CFBundleIdentifier")), integrationBundleLayout())
}

func shouldHandleIntegrationBundleAppLaunchWithContext(args []string, bundleIdentifier string, layout *integrationBundlePaths) bool {
	return len(args) == 0 && strings.TrimSpace(bundleIdentifier) != "" && layout != nil
}

func handleIntegrationBundleAppLaunch(stderr io.Writer) int {
	layout := integrationBundleLayout()
	if !ensureIntegrationBundleLaunchAllowed(layout, stderr) {
		return 1
	}
	if err := integrationStartLaunchSplashFn(); err != nil {
		log.Printf("start launch splash failed: %v", err)
	}
	flags := mergeIntegrationRuntimeFlags(map[string]string{})
	port := resolveIntegrationLaunchPort(flags)
	updateBlocked, err := prepareIntegrationRuntimeLayoutForBundleLaunch(port, stderr)
	if err != nil {
		if stderr != nil {
			fmt.Fprintln(stderr, err.Error())
		}
		return 1
	}
	openBrowserFn := integrationOpenBrowserFn
	markBundleLaunchSeen := func() {}
	bypassBrowserReuse := false
	if bypassReuse, markSeen := integrationShouldBypassExistingBrowserReuse(layout); bypassReuse {
		bypassBrowserReuse = true
		openBrowserFn = integrationOpenBrowserWithoutActivationFn
		markBundleLaunchSeen = func() {
			if err := markSeen(); err != nil {
				log.Printf("mark bundle launch browser reuse state failed: %v", err)
			}
		}
	}
	if updateBlocked {
		if opened, openErr := openExistingIntegrationBrowserIfReadyWithOpener(port, nil, stderr, openBrowserFn, ""); opened {
			if openErr != nil {
				return 1
			}
			markBundleLaunchSeen()
			return 0
		}
		if err := openBrowserFn(integrationBrowserURL(port)); err != nil {
			if stderr != nil {
				fmt.Fprintf(stderr, "open browser failed: %v\n", err)
			}
			return 1
		}
		markBundleLaunchSeen()
		return 0
	}
	if opened, openErr := openExistingIntegrationBrowserIfReadyWithOpener(port, nil, stderr, openBrowserFn, ""); opened {
		if openErr != nil {
			return 1
		}
		markBundleLaunchSeen()
		return 0
	}
	if !bypassBrowserReuse {
		code := integrationStartProcessFn(flags, os.Stdout, stderr)
		if code == 0 {
			markBundleLaunchSeen()
		}
		return code
	}
	originalOpenBrowserFn := integrationOpenBrowserFn
	integrationOpenBrowserFn = openBrowserFn
	defer func() {
		integrationOpenBrowserFn = originalOpenBrowserFn
	}()
	code := integrationStartProcessFn(flags, os.Stdout, stderr)
	if code == 0 {
		markBundleLaunchSeen()
	}
	return code
}

func resolveIntegrationLaunchPort(flags map[string]string) int {
	portValue := strings.TrimSpace(flags["port"])
	if portValue == "" {
		if value, ok := readIntegrationStartupConfigValue("port"); ok {
			portValue = value
		}
	}
	if value, err := strconv.Atoi(portValue); err == nil && validateIntegrationServicePort(value) == nil {
		return value
	}
	return integrationServicePort
}

func openExistingIntegrationBrowserIfReady(port int, logWriter io.Writer, stderr io.Writer, successLogFormat string, args ...interface{}) (bool, error) {
	return openExistingIntegrationBrowserIfReadyWithOpener(port, logWriter, stderr, integrationOpenBrowserFn, successLogFormat, args...)
}

func openExistingIntegrationBrowserIfReadyWithOpener(port int, logWriter io.Writer, stderr io.Writer, openBrowserFn func(string) error, successLogFormat string, args ...interface{}) (bool, error) {
	if !integrationReadyCheck(strconv.Itoa(port)) {
		return false, nil
	}
	if !waitIntegrationBrowserEntryReady(port) {
		if logWriter != nil {
			integrationLifecycleLog(logWriter, "browser entry not ready yet, opening browser anyway addr=%s", fmt.Sprintf("127.0.0.1:%d", port))
		}
	}
	if logWriter != nil && strings.TrimSpace(successLogFormat) != "" {
		integrationLifecycleLog(logWriter, successLogFormat, args...)
	}
	if openBrowserFn == nil {
		openBrowserFn = integrationOpenBrowserFn
	}
	if err := openBrowserFn(integrationBrowserOpenURL(port)); err != nil {
		if logWriter != nil {
			integrationLifecycleLog(logWriter, "open browser failed: %v", err)
		}
		if stderr != nil {
			fmt.Fprintf(stderr, "open browser failed: %v\n", err)
		}
		return true, err
	}
	return true, nil
}

func waitIntegrationBrowserEntryReady(port int) bool {
	if integrationBrowserEntryReadyCheck(port) {
		return true
	}
	if integrationBrowserEntryWait <= 0 {
		return false
	}
	interval := integrationBrowserEntryPollInterval
	if interval <= 0 {
		interval = 50 * time.Millisecond
	}
	deadline := time.Now().Add(integrationBrowserEntryWait)
	for time.Now().Before(deadline) {
		time.Sleep(interval)
		if integrationBrowserEntryReadyCheck(port) {
			return true
		}
	}
	return integrationBrowserEntryReadyCheck(port)
}
