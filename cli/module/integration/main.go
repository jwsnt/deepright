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
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
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
	"math"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
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
	if runtime.GOOS == "windows" {
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

const integrationMCPParseWarning = "MCP 配置必须是有内容的 JSON 对象或数组"

type integrationMCPConfig struct {
	interval time.Duration
	file     string
}

type integrationMCPWarning struct {
	AgentID string `json:"agentId"`
	Path    string `json:"path"`
	Reason  string `json:"reason"`
	Time    int64  `json:"time"`
}

type integrationMCPAgentResult struct {
	metadataPath string
	warning      *integrationMCPWarning
}

type integrationMCPState struct {
	mu          sync.Mutex
	config      integrationMCPConfig
	lastRefresh time.Time
	results     map[string]integrationMCPAgentResult
}

func normalizeIntegrationMCPFilePath(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" || filepath.IsAbs(value) || strings.Contains(value, "\\") {
		return "", false
	}
	value = path.Clean(value)
	if value == "." || value == ".." || strings.HasPrefix(value, "../") || path.IsAbs(value) {
		return "", false
	}
	return value, true
}

func readIntegrationMCPConfig() (integrationMCPConfig, bool) {
	raw, _, err := readIntegrationStartupConfigRaw()
	if err != nil || raw == nil {
		return integrationMCPConfig{}, false
	}
	mcp, ok := raw["mcp"].(map[string]interface{})
	if !ok || mcp == nil {
		return integrationMCPConfig{}, false
	}
	intervalValue, ok := mcp["interval"].(json.Number)
	if !ok {
		return integrationMCPConfig{}, false
	}
	seconds, err := intervalValue.Int64()
	if err != nil || seconds <= 0 || seconds > int64((1<<63-1)/int64(time.Second)) {
		return integrationMCPConfig{}, false
	}
	fileValue, ok := mcp["file"].(string)
	if !ok {
		return integrationMCPConfig{}, false
	}
	file, ok := normalizeIntegrationMCPFilePath(fileValue)
	if !ok {
		return integrationMCPConfig{}, false
	}
	return integrationMCPConfig{interval: time.Duration(seconds) * time.Second, file: file}, true
}

func integrationMCPFileResult(agentID, workspace, relativePath string) integrationMCPAgentResult {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" || relativePath == "" {
		return integrationMCPAgentResult{}
	}
	absoluteWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return integrationMCPAgentResult{}
	}
	filePath := filepath.Join(absoluteWorkspace, filepath.FromSlash(relativePath))
	data, err := os.ReadFile(filePath)
	if err != nil || len(bytes.TrimSpace(data)) == 0 {
		return integrationMCPAgentResult{}
	}
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err == nil {
		switch value := parsed.(type) {
		case map[string]interface{}:
			if len(value) > 0 {
				return integrationMCPAgentResult{metadataPath: filePath}
			}
			return integrationMCPAgentResult{}
		case []interface{}:
			if len(value) > 0 {
				return integrationMCPAgentResult{metadataPath: filePath}
			}
			return integrationMCPAgentResult{}
		}
	}
	return integrationMCPAgentResult{warning: &integrationMCPWarning{
		AgentID: strings.TrimSpace(agentID),
		Path:    relativePath,
		Reason:  integrationMCPParseWarning,
		Time:    time.Now().Unix(),
	}}
}

func (state *integrationMCPState) refresh(config integrationMCPConfig, agents []Agent, force bool) bool {
	if state == nil {
		return false
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if !force && state.results != nil && state.config == config && time.Since(state.lastRefresh) < config.interval {
		return false
	}
	results := make(map[string]integrationMCPAgentResult, len(agents))
	for _, agent := range agents {
		agentID := strings.TrimSpace(agent.AgentID)
		if agentID == "" {
			continue
		}
		results[agentID] = integrationMCPFileResult(agentID, agent.Workspace, config.file)
	}
	state.config = config
	state.results = results
	state.lastRefresh = time.Now()
	return true
}

func (state *integrationMCPState) clear() bool {
	if state == nil {
		return false
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.results == nil && state.config == (integrationMCPConfig{}) {
		return false
	}
	state.config = integrationMCPConfig{}
	state.results = nil
	state.lastRefresh = time.Now()
	return true
}

func (state *integrationMCPState) pathForAgent(agentID string) string {
	if state == nil {
		return ""
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.results[strings.TrimSpace(agentID)].metadataPath
}

func (state *integrationMCPState) warnings() []integrationMCPWarning {
	if state == nil {
		return nil
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	warnings := make([]integrationMCPWarning, 0)
	for _, result := range state.results {
		if result.warning != nil {
			warnings = append(warnings, *result.warning)
		}
	}
	return warnings
}

func (cfg *Config) mcpState() *integrationMCPState {
	if cfg == nil {
		return nil
	}
	cfg.mcpStateMu.Lock()
	defer cfg.mcpStateMu.Unlock()
	if cfg.MCPState == nil {
		cfg.MCPState = &integrationMCPState{}
	}
	return cfg.MCPState
}

func injectMCPMetadata(cfg *Config, metadata *AgentOutput, metaMap map[string]interface{}) {
	if metaMap == nil {
		return
	}
	agentID, _ := metaMap["agentId"].(string)
	injectMCPMetadataForAgent(cfg, metadata, metaMap, agentID)
}

func injectMCPMetadataForAgent(cfg *Config, metadata *AgentOutput, metaMap map[string]interface{}, agentID string) {
	if metaMap == nil {
		return
	}
	delete(metaMap, "mcp")
	if cfg == nil || metadata == nil {
		return
	}
	state := cfg.mcpState()
	config, ok := readIntegrationMCPConfig()
	if !ok {
		state.clear()
		return
	}
	state.refresh(config, metadata.Agents, false)
	agentID = strings.TrimSpace(agentID)
	if agentID == "" && len(metadata.Agents) == 1 {
		agentID = metadata.Agents[0].AgentID
	}
	if mcpPath := state.pathForAgent(agentID); mcpPath != "" {
		metaMap["mcp"] = mcpPath
	}
}

func integrationMCPConfiguredAgents(cfg *Config) ([]Agent, error) {
	if cfg == nil {
		return nil, nil
	}
	agentIDs, err := agentcore.ListAgentIDs(cfg.AgentDir)
	if err != nil {
		return nil, err
	}
	agents := make([]Agent, 0, len(agentIDs))
	for _, agentID := range agentIDs {
		workspace, err := resolveAgentWorkspace(cfg.AgentDir, agentID)
		if err != nil {
			continue
		}
		agents = append(agents, Agent{AgentID: agentID, Workspace: workspace})
	}
	return agents, nil
}

func refreshIntegrationMCPMetadata(cfg *Config, force bool) (bool, error) {
	if cfg == nil {
		return false, nil
	}
	state := cfg.mcpState()
	config, ok := readIntegrationMCPConfig()
	if !ok {
		return state.clear(), nil
	}
	agents, err := integrationMCPConfiguredAgents(cfg)
	if err != nil {
		return false, err
	}
	return state.refresh(config, agents, force), nil
}

func integrationMCPRefreshInterval() time.Duration {
	config, ok := readIntegrationMCPConfig()
	if !ok {
		return time.Minute
	}
	return config.interval
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

// cleanupIntegrationBrowserReuseState removes the per-bundle browser launch
// markers when the integration service exits. They only influence the first
// browser-open behavior for a running app and do not need to persist after it
// has been closed.
func cleanupIntegrationBrowserReuseState() error {
	runtimeBase := strings.TrimSpace(integrationBundleRuntimeBaseDir())
	if runtimeBase == "" {
		return nil
	}
	return os.RemoveAll(filepath.Join(runtimeBase, "state", "browser-reuse"))
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
	if _, err := ensureDefaultDisabledSkillsForChat(chatID, skillDirectoriesFromOutput(&cloned)); err != nil {
		return nil, err
	}
	if err := applyChatSkillState(&cloned, chatID); err != nil {
		return nil, err
	}
	if err := syncIntegrationKnowledgeOutput(&cloned, root); err != nil {
		return nil, err
	}
	cloned.Plugins = detectPluginKeys()
	return &cloned, nil
}

type integrationAtMenuConfig struct {
	RefreshInterval time.Duration
	CacheTTL        time.Duration
}

type integrationAtMenuRefreshCall struct {
	done chan struct{}
	err  error
}

type integrationAtMenuSnapshot struct {
	output            *AgentOutput
	runtimeSkillNames []string
}

// integrationAtMenuCache holds the expensive Agent/Skill snapshot used only
// by the @ menu APIs. It deliberately does not replace the live metadata path
// used for chat, CLI, or scheduled requests.
type integrationAtMenuCache struct {
	mu        sync.Mutex
	config    integrationAtMenuConfig
	load      func() (*integrationAtMenuSnapshot, error)
	now       func() time.Time
	snapshot  *integrationAtMenuSnapshot
	chat      map[string]*AgentOutput
	expiresAt time.Time
	refresh   *integrationAtMenuRefreshCall
}

func newIntegrationAtMenuCache(config integrationAtMenuConfig, load func() (*integrationAtMenuSnapshot, error)) *integrationAtMenuCache {
	return &integrationAtMenuCache{
		config: config,
		load:   load,
		now:    time.Now,
		chat:   make(map[string]*AgentOutput),
	}
}

func cloneIntegrationAtMenuOutput(src *AgentOutput) *AgentOutput {
	if src == nil {
		return nil
	}
	cloned := *src
	if src.Plugins != nil {
		cloned.Plugins = append([]string(nil), src.Plugins...)
	}
	if src.Knowledge != nil {
		knowledge := *src.Knowledge
		cloned.Knowledge = &knowledge
	}
	if src.Agents != nil {
		cloned.Agents = make([]agentcore.Agent, len(src.Agents))
		for index := range src.Agents {
			cloned.Agents[index] = src.Agents[index]
			if src.Agents[index].Skills != nil {
				cloned.Agents[index].Skills = append([]agentcore.Skill(nil), src.Agents[index].Skills...)
			}
		}
	}
	return &cloned
}

func cloneIntegrationAtMenuSnapshot(src *integrationAtMenuSnapshot) *integrationAtMenuSnapshot {
	if src == nil {
		return nil
	}
	return &integrationAtMenuSnapshot{
		output:            cloneIntegrationAtMenuOutput(src.output),
		runtimeSkillNames: append([]string(nil), src.runtimeSkillNames...),
	}
}

func integrationAtMenuOutputForDisabledPaths(snapshot *integrationAtMenuSnapshot, disabledPaths []string) *AgentOutput {
	if snapshot == nil || snapshot.output == nil {
		return nil
	}
	output := cloneIntegrationAtMenuOutput(snapshot.output)
	for index := range output.Agents {
		baseSkills := output.Agents[index].Skills
		filteredSkills := filterSkillsByDisabledPaths(baseSkills, disabledPaths)
		present := make(map[string]struct{}, len(filteredSkills))
		allBaseNames := make(map[string]struct{}, len(baseSkills))
		for _, skill := range filteredSkills {
			if name := strings.TrimSpace(skill.Name); name != "" {
				present[name] = struct{}{}
			}
		}
		for _, skill := range baseSkills {
			if name := strings.TrimSpace(skill.Name); name != "" {
				allBaseNames[name] = struct{}{}
			}
		}
		runtime := make([]string, 0, len(snapshot.runtimeSkillNames))
		for _, name := range snapshot.runtimeSkillNames {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			// A configured runtime name must not restore a skill directory that
			// this chat has just disabled. It remains available when another
			// non-disabled directory exposes the same name.
			if _, wasBaseSkill := allBaseNames[name]; wasBaseSkill {
				if _, remainsAvailable := present[name]; !remainsAvailable {
					continue
				}
			}
			runtime = append(runtime, name)
		}
		names := mergeIntegrationRuntimeSkillNames(agentcore.SkillNames(filteredSkills), runtime)
		output.Agents[index].Skills = make([]agentcore.Skill, len(names))
		for skillIndex, name := range names {
			output.Agents[index].Skills[skillIndex] = agentcore.Skill{Name: name}
		}
	}
	return output
}

func (cache *integrationAtMenuCache) refreshSnapshot() error {
	return cache.refreshSnapshotWithMode(false)
}

// refreshSnapshotAfterMutation always performs a scan after the caller's
// mutation. If a scheduled scan was already running, waiting for that scan is
// not enough: it may have read the Agent directory before the mutation.
func (cache *integrationAtMenuCache) refreshSnapshotAfterMutation() error {
	return cache.refreshSnapshotWithMode(true)
}

func (cache *integrationAtMenuCache) refreshSnapshotWithMode(force bool) error {
	if cache == nil || cache.load == nil {
		return fmt.Errorf("@ cache is not initialized")
	}
	for {
		cache.mu.Lock()
		if pending := cache.refresh; pending != nil {
			cache.mu.Unlock()
			<-pending.done
			if !force {
				return pending.err
			}
			continue
		}
		pending := &integrationAtMenuRefreshCall{done: make(chan struct{})}
		cache.refresh = pending
		cache.mu.Unlock()

		snapshot, err := cache.load()
		if err == nil && (snapshot == nil || snapshot.output == nil) {
			err = fmt.Errorf("@ cache loader returned no Agent metadata")
		}

		cache.mu.Lock()
		pending.err = err
		if err == nil {
			cache.snapshot = cloneIntegrationAtMenuSnapshot(snapshot)
			cache.chat = make(map[string]*AgentOutput)
			cache.expiresAt = cache.now().Add(cache.config.CacheTTL)
		}
		cache.refresh = nil
		close(pending.done)
		cache.mu.Unlock()
		return err
	}
}

func (cache *integrationAtMenuCache) snapshotForRequest() (*integrationAtMenuSnapshot, error) {
	if cache == nil {
		return nil, fmt.Errorf("@ cache is not initialized")
	}
	cache.mu.Lock()
	if cache.snapshot != nil && cache.now().Before(cache.expiresAt) {
		snapshot := cloneIntegrationAtMenuSnapshot(cache.snapshot)
		cache.mu.Unlock()
		return snapshot, nil
	}
	cache.mu.Unlock()

	if err := cache.refreshSnapshot(); err != nil {
		return nil, fmt.Errorf("@ cache expired and refresh failed: %w", err)
	}

	cache.mu.Lock()
	snapshot := cloneIntegrationAtMenuSnapshot(cache.snapshot)
	cache.mu.Unlock()
	if snapshot == nil {
		return nil, fmt.Errorf("@ cache refresh completed without Agent metadata")
	}
	return snapshot, nil
}

func (cache *integrationAtMenuCache) outputForRequest(chatID string, disabledPaths func() ([]string, error)) (*AgentOutput, error) {
	snapshot, err := cache.snapshotForRequest()
	if err != nil {
		return nil, err
	}
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return integrationAtMenuOutputForDisabledPaths(snapshot, nil), nil
	}

	cache.mu.Lock()
	if output := cache.chat[chatID]; output != nil {
		cloned := cloneIntegrationAtMenuOutput(output)
		cache.mu.Unlock()
		return cloned, nil
	}
	cache.mu.Unlock()

	paths, err := disabledPaths()
	if err != nil {
		return nil, err
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if output := cache.chat[chatID]; output != nil {
		return cloneIntegrationAtMenuOutput(output), nil
	}
	if cache.snapshot == nil {
		return nil, fmt.Errorf("@ cache refresh completed without Agent metadata")
	}
	output := integrationAtMenuOutputForDisabledPaths(cache.snapshot, paths)
	cache.chat[chatID] = cloneIntegrationAtMenuOutput(output)
	return output, nil
}

func (cache *integrationAtMenuCache) updateChatSkillState(chatID string, disabledPaths []string) {
	if cache == nil || strings.TrimSpace(chatID) == "" {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.snapshot == nil {
		return
	}
	cache.chat[chatID] = integrationAtMenuOutputForDisabledPaths(cache.snapshot, disabledPaths)
	cache.expiresAt = cache.now().Add(cache.config.CacheTTL)
}

func readIntegrationAtMenuConfig() (integrationAtMenuConfig, error) {
	raw, _, err := readIntegrationStartupConfigRaw()
	if err != nil {
		return integrationAtMenuConfig{}, fmt.Errorf("read config/config.json.at: %w", err)
	}
	atRaw, ok := raw["at"]
	if !ok {
		return integrationAtMenuConfig{}, fmt.Errorf("config/config.json.at is required")
	}
	at, ok := atRaw.(map[string]interface{})
	if !ok || at == nil {
		return integrationAtMenuConfig{}, fmt.Errorf("config/config.json.at must be an object")
	}
	refreshSeconds, err := integrationAtMenuPositiveSeconds(at["refresh"], "config/config.json.at.refresh")
	if err != nil {
		return integrationAtMenuConfig{}, err
	}
	cacheSeconds, err := integrationAtMenuPositiveSeconds(at["cache"], "config/config.json.at.cache")
	if err != nil {
		return integrationAtMenuConfig{}, err
	}
	return integrationAtMenuConfig{
		RefreshInterval: time.Duration(refreshSeconds) * time.Second,
		CacheTTL:        time.Duration(cacheSeconds) * time.Second,
	}, nil
}

func integrationAtMenuPositiveSeconds(raw interface{}, label string) (int64, error) {
	value, ok := raw.(json.Number)
	if !ok {
		return 0, fmt.Errorf("%s must be a positive integer in seconds", label)
	}
	seconds, err := value.Int64()
	if err != nil || seconds <= 0 || seconds > int64((1<<63-1)/int64(time.Second)) {
		return 0, fmt.Errorf("%s must be a positive integer in seconds", label)
	}
	return seconds, nil
}

func initializeIntegrationAtMenuCache(cfg *Config, config integrationAtMenuConfig) error {
	if cfg == nil {
		return fmt.Errorf("@ cache configuration is required")
	}
	if config.RefreshInterval <= 0 || config.CacheTTL <= 0 {
		return fmt.Errorf("@ cache refresh and TTL must be positive")
	}
	cfg.AtMenuCache = newIntegrationAtMenuCache(config, func() (*integrationAtMenuSnapshot, error) {
		agentTTL := time.Duration(cfg.AgentCacheMs) * time.Millisecond
		output, err := agentcore.GetOutputForApp(cfg.AgentDir, integrationAppDir(), cfg.effectiveDeviceID(), agentTTL)
		if err != nil {
			return nil, err
		}
		cloned := *output
		hydrateAgentOutput(&cloned, "")
		return &integrationAtMenuSnapshot{
			output:            &cloned,
			runtimeSkillNames: buildIntegrationRuntimeSkillNames(nil),
		}, nil
	})
	return nil
}

func atMenuOutputForRequest(cfg *Config, chatID string) (*AgentOutput, error) {
	if cfg == nil || cfg.AtMenuCache == nil {
		return nil, fmt.Errorf("@ cache is not configured")
	}
	chatID = strings.TrimSpace(chatID)
	if chatID != "" {
		snapshot, err := cfg.AtMenuCache.snapshotForRequest()
		if err != nil {
			return nil, err
		}
		if _, err := ensureDefaultDisabledSkillsForChat(chatID, skillDirectoriesFromOutput(snapshot.output)); err != nil {
			return nil, err
		}
	}
	return cfg.AtMenuCache.outputForRequest(chatID, func() ([]string, error) {
		return disabledSkillPathsForChat(chatID)
	})
}

func startIntegrationAtMenuRefresh(ctx context.Context, cfg *Config) {
	if ctx == nil || cfg == nil || cfg.AtMenuCache == nil {
		return
	}
	cache := cfg.AtMenuCache
	go func() {
		if err := cache.refreshSnapshot(); err != nil {
			log.Printf("@ cache initial refresh failed: %v", err)
		}
		ticker := time.NewTicker(cache.config.RefreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := cache.refreshSnapshot(); err != nil {
					log.Printf("@ cache scheduled refresh failed: %v", err)
				}
			}
		}
	}()
}

func refreshIntegrationAtMenuCacheAfterAgentChange(cfg *Config, action string) {
	if cfg == nil || cfg.AtMenuCache == nil {
		return
	}
	if err := cfg.AtMenuCache.refreshSnapshotAfterMutation(); err != nil {
		log.Printf("%s: refresh @ cache failed: %v", action, err)
	}
}

// integrationAtMenuPathIsAgentConfig reports whether a workspace mutation
// changes the Agent configuration used by the cached @ Agent menu.
func integrationAtMenuPathIsAgentConfig(workspace, candidate string) bool {
	rel, err := filepath.Rel(workspace, candidate)
	return err == nil && strings.EqualFold(filepath.Clean(rel), "config.json")
}

// integrationAtMenuPathIsSkillDefinition reports whether a workspace file is
// one of the skill definitions scanned for the cached @ Skill menu. Agent
// skills retain compatibility with both SKILL.md and SKILL.
func integrationAtMenuPathIsSkillDefinition(workspace, candidate string) bool {
	skillsRoot := filepath.Join(workspace, "skills")
	rel, err := filepath.Rel(skillsRoot, candidate)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	name := filepath.Base(candidate)
	return strings.EqualFold(name, "SKILL.md") || strings.EqualFold(name, "SKILL")
}

// integrationAtMenuDeletedPathAffectsSnapshot determines before deletion
// whether the removed item contributed to the cached @ menu. Ordinary files
// and directories are intentionally excluded because /api/files is not
// cached by the @ menu cache.
func integrationAtMenuDeletedPathAffectsSnapshot(workspace, candidate string, info os.FileInfo) bool {
	if integrationAtMenuPathIsAgentConfig(workspace, candidate) || integrationAtMenuPathIsSkillDefinition(workspace, candidate) {
		return true
	}
	if info == nil || !info.IsDir() {
		return false
	}
	skillsRoot := filepath.Join(workspace, "skills")
	rel, err := filepath.Rel(skillsRoot, candidate)
	if err != nil || (rel != "." && (rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)))) {
		return false
	}
	skills, err := skillscore.ScanAgentSkills(candidate)
	return err == nil && len(skills) > 0
}

func refreshIntegrationAtMenuCacheAfterWorkspaceMutation(cfg *Config, action string) {
	if cfg == nil || cfg.AtMenuCache == nil {
		return
	}
	flushAgentCache()
	refreshIntegrationAtMenuCacheAfterAgentChange(cfg, action)
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
	paths := []string{resolveIntegrationDBPath()}
	// Keep compatibility with a legacy working-directory database only when it
	// already exists. Opening a missing relative SQLite path creates it, which
	// would otherwise write an unsigned data file into a macOS app bundle.
	if _, err := os.Stat("data"); err == nil {
		paths = append(paths, "data")
	}
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
		skills := integrationApplySeedreamSkill(output.Agents[i].Skills, seedreamSkill, seedreamEnabled)
		agentcore.SortSkills(skills)
		output.Agents[i].Skills = skills
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

func skillDirectoriesFromOutput(output *AgentOutput) []string {
	if output == nil {
		return nil
	}
	directories := make([]string, 0)
	for _, agent := range output.Agents {
		for _, skill := range agent.Skills {
			location := strings.TrimSpace(skill.Location)
			if location == "" {
				continue
			}
			directories = append(directories, filepath.Dir(location))
		}
	}
	return directories
}

func ensureDefaultDisabledSkillsForChat(chatID string, directories []string) ([]string, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return nil, nil
	}
	if cronDB != nil {
		return skillstate.EnsureDefaultDisabled(cronDB, chatID, directories)
	}
	db, err := sql.Open("sqlite", resolveIntegrationDBPath())
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return skillstate.EnsureDefaultDisabled(db, chatID, directories)
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
			// Prefer the Agent-local copy when present so the VFS folder switch
			// and chat metadata use the same persisted skill path.
			if location := strings.TrimSpace(skill.Location); location != "" {
				seedream.Location = location
			}
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

// integrationApplicationDataDir resolves the application data root from the
// configured Agent root. Agent workspaces are <dir>/agent/<agentId>, making
// this value exactly two levels above a workspace.
func integrationApplicationDataDir(agentDir string) string {
	agentDir = strings.TrimSpace(agentDir)
	if agentDir == "" {
		return ""
	}
	absAgentDir, err := filepath.Abs(agentDir)
	if err != nil {
		return ""
	}
	parent := filepath.Dir(filepath.Clean(absAgentDir))
	if parent == "." || parent == absAgentDir {
		return ""
	}
	return parent
}

// injectApplicationDataDirMetadata makes dir runtime-owned metadata. It must
// be called after merging request metadata so callers cannot forge the path.
func injectApplicationDataDirMetadata(metaMap map[string]interface{}, agentDir string) {
	if metaMap == nil {
		return
	}
	delete(metaMap, "dir")
	if dir := integrationApplicationDataDir(agentDir); dir != "" {
		metaMap["dir"] = dir
	}
}

// injectIntegrationConfigPathMetadata makes config and version runtime-owned
// upstream metadata fields. The path is checked immediately before sending so
// a deleted or unreadable main configuration cannot be advertised upstream;
// version is cached from that configuration at process startup.
func injectIntegrationConfigPathMetadata(metaMap map[string]interface{}, cfg *Config) error {
	if metaMap == nil || cfg == nil {
		return nil
	}
	configPath := strings.TrimSpace(cfg.StartupConfigPath)
	if configPath == "" {
		return nil
	}
	configPath, err := validateIntegrationStartupConfigFile(configPath)
	if err != nil {
		return err
	}
	delete(metaMap, "config")
	metaMap["config"] = configPath
	delete(metaMap, "version")
	if version := strings.TrimSpace(cfg.ClientVersion); version != "" {
		metaMap["version"] = version
	}
	return nil
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

func resolveCurrentPageSessionAgentID(chatID string) string {
	if cronDB == nil {
		return ""
	}
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return ""
	}
	var agentID string
	if err := cronDB.QueryRow(`SELECT agent_id FROM chat_log WHERE chat_id = ? AND agent_id != '' ORDER BY created_at DESC, id DESC LIMIT 1`, chatID).Scan(&agentID); err != nil {
		return ""
	}
	return strings.TrimSpace(agentID)
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

// setDeviceIDHeader preserves the public Header spelling expected by upstream.
// Header.Set canonicalizes this as Deviceid, which does not retain camel case.
func setDeviceIDHeader(req *http.Request, deviceID string) {
	req.Header["DeviceId"] = []string{deviceID}
}

func heartbeat(client *http.Client, host string, metadata *AgentOutput, cfg *Config) (*TaskContent, error) {
	metaMap := buildCLIRequestMetadataMap(metadata)
	if cfg != nil {
		injectApplicationDataDirMetadata(metaMap, cfg.AgentDir)
	}
	if err := injectIntegrationConfigPathMetadata(metaMap, cfg); err != nil {
		return nil, fmt.Errorf("validate integration config: %w", err)
	}
	currentChatID := resolveCurrentPageSessionChatID()
	injectMCPMetadataForAgent(cfg, metadata, metaMap, resolveCurrentPageSessionAgentID(currentChatID))
	metaMap["port"] = integrationMetadataPort(cfg)
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
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create heartbeat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	deviceID, _ := metaMap["deviceId"].(string)
	setDeviceIDHeader(req, deviceID)
	resp, err := client.Do(req)
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
		// Some upstream gateways put the configured page protocol code in the
		// HTTP status instead of the JSON response body. It has the same
		// control-packet meaning, and cli/get must not turn it into an alert.
		if readConfiguredPageResponseCodes().matches(resp.StatusCode) {
			return nil, nil
		}
		return nil, fmt.Errorf("heartbeat HTTP %d", resp.StatusCode)
	}
	var rp ResponsePayload
	if err := json.Unmarshal(body, &rp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if rp.Code != 200 {
		// Page protocol responses are intentional control packets rather than
		// failed heartbeats. cli/get does not execute them, so acknowledge them
		// as an idle, successful poll and do not contribute to the alert count.
		if readConfiguredPageResponseCodes().matches(rp.Code) {
			return nil, nil
		}
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

func shouldAwaitAfterCliGetFailure(sseActive, immediatelyRetryByCheck bool) bool {
	return !sseActive && !immediatelyRetryByCheck
}

// cliGetCommandCheck tracks consecutive cli/get results without a command.
// A failed heartbeat is also a no-command result. The single atomic state
// makes command reset and no-command increment visible atomically.
type cliGetCommandCheck struct {
	emptySinceCommand atomic.Int64
}

func newCliGetCommandCheck() cliGetCommandCheck {
	state := cliGetCommandCheck{}
	state.emptySinceCommand.Store(0)
	return state
}

func (state *cliGetCommandCheck) resetForCommand() {
	state.emptySinceCommand.Store(0)
}

func (state *cliGetCommandCheck) shouldImmediatelyRetryWithoutSSE(hasCommand bool, check int) bool {
	if hasCommand {
		state.resetForCommand()
		return true
	}
	for {
		current := state.emptySinceCommand.Load()
		if state.emptySinceCommand.CompareAndSwap(current, current+1) {
			return current+1 < int64(check)
		}
	}
}

func taskHasExecutableCommand(task *TaskContent) bool {
	return task != nil && strings.TrimSpace(task.Cmd) != ""
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
	metaMap := buildCLIRequestMetadataMap(metadata)
	if cfg != nil {
		injectApplicationDataDirMetadata(metaMap, cfg.AgentDir)
	}
	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":    "",
		"messages": []Message{{Role: "user", Content: string(resultJSON)}},
		"metadata": metaMap,
	})
	url := strings.TrimRight(host, "/") + "/cli/pub"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("create publish request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	deviceID, _ := metaMap["deviceId"].(string)
	setDeviceIDHeader(req, deviceID)
	resp, err := client.Do(req)
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
	if cfg.SleepMs < 0 {
		log.Printf("cli-get: invalid sleep=%d", cfg.SleepMs)
		return
	}
	if cfg.AwaitMs < 0 {
		log.Printf("cli-get: invalid get.await=%d", cfg.AwaitMs)
		return
	}
	if cfg.Check <= 0 {
		log.Printf("cli-get: invalid get.check=%d", cfg.Check)
		return
	}

	agentTTL := time.Duration(cfg.AgentCacheMs) * time.Millisecond
	sleepDur := time.Duration(cfg.SleepMs) * time.Millisecond
	awaitDur := time.Duration(cfg.AwaitMs) * time.Millisecond
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

	log.Printf("cli-get: started (host=%s, threads=%d, queue=%d, sleep=%dms, await=%dms, check=%d, retry_interval=%dms, retry_times=%d)",
		cfg.currentHost(), cfg.Thread, cfg.Queue, cfg.SleepMs, cfg.AwaitMs, cfg.Check, cfg.RetryIntervalMs, cfg.RetryTimes)
	commandCheck := newCliGetCommandCheck()

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
				immediatelyRetryByCheck := commandCheck.shouldImmediatelyRetryWithoutSSE(false, cfg.Check)
				if shouldAwaitAfterCliGetFailure(integrationHasActiveSSE(), immediatelyRetryByCheck) {
					heartbeatBackoff = sleepDur
					if !waitForCliGetHeartbeat(ctx, awaitDur) {
						return
					}
					continue
				}
				if !sleepContext(ctx, heartbeatBackoff) {
					return
				}
				heartbeatBackoff = nextHeartbeatBackoff(heartbeatBackoff, sleepDur, maxHeartbeatBackoff)
				continue
			}
			heartbeatBackoff = sleepDur
			hasCommand := taskHasExecutableCommand(task)
			sseActive := integrationHasActiveSSE()
			// Every result without cmd advances the idle check, including results
			// observed while an SSE is active. SSE activity only suppresses await.
			immediatelyRetryByCheck := commandCheck.shouldImmediatelyRetryWithoutSSE(hasCommand, cfg.Check)
			if task == nil {
				recordHeartbeat(0, "") // success, no task
			} else {
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
			if sseActive || immediatelyRetryByCheck {
				continue
			}
			// While any session, memo, Feishu, or mail SSE is in flight, retain
			// immediate heartbeats. When SSE is idle, the atomic command check
			// decides whether this successful heartbeat can enter standby.
			if !waitForCliGetHeartbeat(ctx, awaitDur) {
				return
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
	// Complete the one-time environment probe before the requested command
	// timeout starts, so dependency discovery cannot consume task runtime.
	env := sharedutil.CurrentEnvironmentWithSystemPath()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	c := exec.CommandContext(ctx, binary, args...)
	// Every command, including CLI_SANDBOX.app, receives the same normalized
	// PATH as the Integration service. This includes dynamically discovered
	// macOS user-level Python console-script directories.
	c.Env = env
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
		metadata, err := atMenuOutputForRequest(cfg, "")
		if err != nil {
			log.Printf("api/swarm_agent: @ cache error: %v", err)
			http.Error(w, "Failed to load @ menu Agent data", http.StatusInternalServerError)
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
var openBrowserURLFn = openSystemTarget
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

func openBrowserURL(target string) error {
	return openBrowserURLFn(target)
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

// handleBrowserOpen opens an HTTP(S) URL in the system browser. It is kept
// separate from the page UI because the UI deliberately blocks window.open.
func handleBrowserOpen() http.HandlerFunc {
	type browserOpenRequest struct {
		URL string `json:"url"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !isLocalManagementRequest(r) {
			http.Error(w, "only localhost requests can open browser URLs", http.StatusForbidden)
			return
		}
		var request browserOpenRequest
		decoder := json.NewDecoder(io.LimitReader(r.Body, 16*1024))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		target, err := normalizeURLPreviewProbeTarget(request.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := openBrowserURL(target.String()); err != nil {
			log.Printf("open browser URL error: %v", err)
			http.Error(w, "Failed to open browser URL: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "url": target.String()})
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
		chatID := strings.TrimSpace(r.URL.Query().Get("chatId"))
		metadata, err := atMenuOutputForRequest(cfg, chatID)
		if err != nil {
			log.Printf("api/skills: @ cache error: %v", err)
			http.Error(w, "Failed to load @ menu skill data", http.StatusInternalServerError)
			return
		}
		for _, a := range metadata.Agents {
			if a.AgentID == agentID {
				names := agentcore.SkillNames(a.Skills)
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
	runtime := make([]string, 0, 5)
	seen := make(map[string]struct{}, 4)
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
		runtime = append(runtime, name)
		seen[name] = struct{}{}
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
		{key: "email", skill: connectsvc.InternalSkillEmail},
		{key: "feishu", skill: connectsvc.InternalSkillFeishu},
	} {
		status, err := connectsvc.PluginStatusByKey(item.key, nil)
		appendIfMissing(item.skill, err == nil && status != nil && status.Started)
	}
	return mergeIntegrationRuntimeSkillNames(base, runtime)
}

func mergeIntegrationRuntimeSkillNames(base, runtime []string) []string {
	out := make([]string, 0, len(base)+len(runtime))
	seen := make(map[string]struct{}, len(base)+len(runtime))
	for _, names := range [][]string{base, runtime} {
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if _, exists := seen[name]; exists {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	agentcore.SortSkillNames(out)
	return out
}

// ═══════════════════════════════════════════════════════════════════════════
// Proxy: /api/files — list files in a directory
// ═══════════════════════════════════════════════════════════════════════════

type FileEntry struct {
	Name                   string `json:"name"`
	Type                   string `json:"type"`
	ModifiedAt             int64  `json:"modifiedAt"`
	HasSkill               bool   `json:"hasSkill,omitempty"`
	SkillDisabled          bool   `json:"skillDisabled,omitempty"`
	SkillDisabledSelf      bool   `json:"skillDisabledSelf,omitempty"`
	SkillDisabledInherited bool   `json:"skillDisabledInherited,omitempty"`
	HasSkills              bool   `json:"hasSkills,omitempty"`
	SkillCount             int    `json:"skillCount,omitempty"`
}

// skillDirectoryEntry keeps the valid Skill locations that belong to a
// top-level skills directory in one VFS response. The locations come from
// the same scanner used by Agent metadata, so the summary never counts an
// invalid SKILL.md as an available Skill.
type skillDirectoryEntry struct {
	resultIndex int
	locations   []string
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
		result := make([]FileEntry, 0, len(entries))
		skillDirectories := make([]string, 0)
		skillsDirectories := make([]skillDirectoryEntry, 0, 1)
		for _, e := range entries {
			if prefix != "" && !strings.HasPrefix(strings.ToLower(e.Name()), strings.ToLower(prefix)) {
				continue
			}
			t := "file"
			if e.IsDir() {
				t = "dir"
			}
			item := FileEntry{Name: e.Name(), Type: t}
			if entryInfo, infoErr := e.Info(); infoErr == nil {
				item.ModifiedAt = entryInfo.ModTime().UnixMilli()
			}
			if e.IsDir() {
				entryPath := filepath.Join(searchDir, e.Name())
				item.HasSkill = hasDirectSkillMarkdown(entryPath)
				if item.HasSkill {
					skillDirectories = append(skillDirectories, entryPath)
				}
				if strings.EqualFold(e.Name(), "skills") {
					skills, scanErr := skillscore.ScanAgentSkills(entryPath)
					if scanErr == nil && len(skills) > 0 {
						locations := make([]string, 0, len(skills))
						for _, skill := range skills {
							location := strings.TrimSpace(skill.Location)
							if location == "" {
								continue
							}
							locations = append(locations, location)
							skillDirectories = append(skillDirectories, filepath.Dir(location))
						}
						if len(locations) > 0 {
							item.HasSkills = true
							skillsDirectories = append(skillsDirectories, skillDirectoryEntry{
								resultIndex: len(result),
								locations:   locations,
							})
						}
					}
				}
			}
			result = append(result, item)
		}
		chatID := strings.TrimSpace(r.URL.Query().Get("chatId"))
		disabledPaths, err := ensureDefaultDisabledSkillsForChat(chatID, skillDirectories)
		if err != nil {
			http.Error(w, "Failed to read skill state: "+err.Error(), http.StatusInternalServerError)
			return
		}
		for index := range result {
			if !result[index].HasSkill {
				continue
			}
			entryPath := filepath.Join(searchDir, result[index].Name)
			result[index].SkillDisabledSelf, result[index].SkillDisabledInherited = skillstate.DisabledStatus(disabledPaths, entryPath)
			result[index].SkillDisabled = result[index].SkillDisabledSelf || result[index].SkillDisabledInherited
		}
		for _, skillsDirectory := range skillsDirectories {
			for _, location := range skillsDirectory.locations {
				selfDisabled, inheritedDisabled := skillstate.DisabledStatus(disabledPaths, location)
				if !selfDisabled && !inheritedDisabled {
					result[skillsDirectory.resultIndex].SkillCount++
				}
			}
		}
		// Keep the API order aligned with the virtual file system.  The Site also
		// applies this ordering defensively, but returning a deterministic order
		// prevents a raw directory enumeration from bypassing the user-visible
		// "folders first, newest files first" rule.
		sort.SliceStable(result, func(i, j int) bool {
			left, right := result[i], result[j]
			if left.Type != right.Type {
				return left.Type == "dir"
			}
			if left.Type == "file" && left.ModifiedAt != right.ModifiedAt {
				return left.ModifiedAt > right.ModifiedAt
			}
			return strings.ToLower(left.Name) < strings.ToLower(right.Name)
		})
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
	if dir == "" || !isWithinSkillsDirectory(dir) {
		return false
	}
	info, err := os.Stat(filepath.Join(dir, "SKILL.md"))
	return err == nil && !info.IsDir()
}

// isWithinSkillsDirectory reports whether path is the skills root itself or a
// descendant of one. SKILL.md files elsewhere are ordinary files and must not
// be exposed as loadable skills.
func isWithinSkillsDirectory(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	for {
		if filepath.Base(absPath) == "skills" {
			return true
		}
		parent := filepath.Dir(absPath)
		if parent == absPath {
			return false
		}
		absPath = parent
	}
}

func handleSkillState(cfg *Config) http.HandlerFunc {
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
		if cfg != nil && cfg.AtMenuCache != nil {
			cfg.AtMenuCache.updateChatSkillState(req.ChatID, paths)
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

// ensureWritablePathWithinRoot additionally resolves the closest existing
// parent directory before a write. A lexical path below workspace can still
// escape through a symlinked parent, so writes must validate the real parent
// as well as reject an existing target symlink.
func ensureWritablePathWithinRoot(root, target string) bool {
	if !ensurePathWithinRoot(root, target) {
		return false
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return false
	}
	parent := filepath.Dir(filepath.Clean(target))
	for {
		resolvedParent, err := filepath.EvalSymlinks(parent)
		if err == nil {
			if !ensurePathWithinRoot(resolvedRoot, resolvedParent) {
				return false
			}
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return false
		}
		nextParent := filepath.Dir(parent)
		if nextParent == parent {
			return false
		}
		parent = nextParent
	}
	if info, err := os.Lstat(target); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return false
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
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
		} else if os.IsNotExist(statErr) && path.Base(relPath) == "index.html" {
			// net/http's FileServer redirects any request ending in index.html to
			// its containing directory before it checks whether the file exists.
			// An explicit missing app page must remain a 404 rather than exposing
			// that directory's listing.
			http.NotFound(w, r)
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
	".mp3": true, ".m4a": true, ".mp4": true, ".m4v": true, ".avi": true, ".mov": true, ".wmv": true, ".flv": true, ".mkv": true, ".webm": true, ".ogv": true, ".mpg": true, ".mpeg": true, ".wav": true, ".flac": true, ".aac": true, ".ogg": true, ".opus": true,
	".zip": true, ".tar": true, ".gz": true, ".rar": true, ".7z": true, ".bz2": true,
	".exe": true, ".dll": true, ".so": true, ".dylib": true, ".bin": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
	".ttf": true, ".otf": true, ".woff": true, ".woff2": true, ".eot": true,
}

func isBinaryFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	// Canvas documents are versioned UTF-8 JSON. Keep their /api/edit writes on
	// the text path even if the binary extension allow-list changes later.
	if ext == ".canvas" {
		return false
	}
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
		workspace, err := getWorkspaceByAgentID(cfg, agentID)
		if err != nil {
			http.Error(w, "Agent not found: "+agentID, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(workspace))
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
	stamp := timestampWithNanoseconds(ts)
	if ext == "" {
		return base + "_" + stamp
	}
	return base + "_" + stamp + ext
}

func timestampWithNanoseconds(ts time.Time) string {
	return ts.Format("20060102_150405") + "_" + fmt.Sprintf("%09d", ts.Nanosecond())
}

func isSafeSaveAsNewFilename(name string) bool {
	name = strings.TrimSpace(name)
	return name != "" && name != "." && name != ".." && len(name) <= 180 && filepath.Base(name) == name && !strings.ContainsAny(name, `/\\`+"\x00")
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
		saveAsNewName := strings.TrimSpace(normalizeQuotedPathArg(r.URL.Query().Get("saveAsNewName")))
		if saveAsNewName != "" && (!saveAsNew || !isSafeSaveAsNewFilename(saveAsNewName)) {
			writeEditResp(w, EditResponse{AgentID: agentID, Path: relPath, Content: "新文件名无效", Status: 1})
			return
		}
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
		if saveAsNewName != "" {
			dir := filepath.Dir(cleanRelPath)
			if dir == "." {
				targetRelPath = saveAsNewName
			} else {
				targetRelPath = filepath.Join(dir, saveAsNewName)
			}
		} else if saveAsNew {
			targetRelPath = appendTimestampToFilename(cleanRelPath, time.Now())
		}
		resolved, err := resolveCaseInsensitiveUnderRoot(workspace, targetRelPath)
		if err != nil {
			writeEditResp(w, EditResponse{AgentID: agentID, Path: relPath, Content: "路径解析失败: " + err.Error(), Status: 1})
			return
		}
		if !ensureWritablePathWithinRoot(workspace, resolved) {
			writeEditResp(w, EditResponse{AgentID: agentID, Path: relPath, Content: "只支持 workspace 下的相对路径", Status: 1})
			return
		}
		if info, statErr := os.Stat(resolved); statErr == nil {
			if info.IsDir() {
				writeEditResp(w, EditResponse{AgentID: agentID, Path: relPath, Content: "不支持写入目录", Status: 1})
				return
			}
			if saveAsNew {
				writeEditResp(w, EditResponse{AgentID: agentID, Path: relPath, Content: "新文件名已存在，请换一个名称", Status: 1})
				return
			}
		} else if !os.IsNotExist(statErr) {
			writeEditResp(w, EditResponse{AgentID: agentID, Path: relPath, Content: "无法检查目标文件", Status: 1})
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
		if integrationAtMenuPathIsAgentConfig(workspace, resolved) {
			refreshIntegrationAtMenuCacheAfterWorkspaceMutation(cfg, "agent config edit")
		} else if integrationAtMenuPathIsSkillDefinition(workspace, resolved) {
			refreshIntegrationAtMenuCacheAfterWorkspaceMutation(cfg, "agent skill definition edit")
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
		info, err := os.Stat(resolved)
		if err != nil {
			writeDelResp(w, DelResponse{AgentID: agentID, Path: relPath, Content: "不存在: " + err.Error(), Status: 1})
			return
		}
		atMenuChanged := integrationAtMenuDeletedPathAffectsSnapshot(workspace, resolved, info)
		if err := os.RemoveAll(resolved); err != nil {
			writeDelResp(w, DelResponse{AgentID: agentID, Path: relPath, Content: "删除失败: " + err.Error(), Status: 1})
			return
		}
		if atMenuChanged {
			refreshIntegrationAtMenuCacheAfterWorkspaceMutation(cfg, "agent skill or config delete")
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

// mediaPreviewMIMETypes is the allow-list for streamed, browser-facing media
// previews. Keep this list in sync with the Site preview type detection.
var mediaPreviewMIMETypes = map[string]string{
	".mp4":  "video/mp4",
	".m4v":  "video/mp4",
	".webm": "video/webm",
	".ogv":  "video/ogg",
	".mov":  "video/quicktime",
	".mkv":  "video/x-matroska",
	".avi":  "video/x-msvideo",
	".mpg":  "video/mpeg",
	".mpeg": "video/mpeg",
	".mp3":  "audio/mpeg",
	".m4a":  "audio/mp4",
	".aac":  "audio/aac",
	".wav":  "audio/wav",
	".flac": "audio/flac",
	".ogg":  "audio/ogg",
	".opus": "audio/ogg",
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".bmp":  "image/bmp",
	".tif":  "image/tiff",
	".tiff": "image/tiff",
}

func mediaPreviewMIMEType(filePath string) (string, bool) {
	mimeType, ok := mediaPreviewMIMETypes[strings.ToLower(filepath.Ext(filePath))]
	return mimeType, ok
}

func isValidMediaPreviewAgentID(agentID string) bool {
	agentID = strings.TrimSpace(agentID)
	return agentID != "" && agentID != "." && agentID != ".." && filepath.Base(agentID) == agentID && !strings.ContainsAny(agentID, `/\\`)
}

func resolveMediaPreviewFile(cfg *Config, agentID, rawPath string) (string, os.FileInfo, int, string) {
	agentID = strings.TrimSpace(agentID)
	pathParam := normalizeQuotedPathArg(rawPath)
	if !isValidMediaPreviewAgentID(agentID) || pathParam == "" {
		return "", nil, http.StatusBadRequest, "agentId and relative path are required"
	}
	cleanPath := filepath.Clean(pathParam)
	if filepath.IsAbs(cleanPath) || strings.HasPrefix(pathParam, "~") || cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return "", nil, http.StatusBadRequest, "path must be a workspace-relative file path"
	}
	workspace, err := getWorkspaceByAgentID(cfg, agentID)
	if err != nil {
		return "", nil, http.StatusNotFound, "agent not found"
	}
	resolved, err := resolveCaseInsensitiveUnderRoot(workspace, cleanPath)
	if err != nil || !ensurePathWithinRoot(workspace, resolved) {
		return "", nil, http.StatusBadRequest, "path must be a workspace-relative file path"
	}
	resolved, err = filepath.EvalSymlinks(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, http.StatusNotFound, "media file not found"
		}
		return "", nil, http.StatusInternalServerError, "failed to resolve media file"
	}
	if !ensurePathWithinRoot(workspace, resolved) {
		return "", nil, http.StatusBadRequest, "path must be a workspace-relative file path"
	}
	info, err := os.Stat(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, http.StatusNotFound, "media file not found"
		}
		return "", nil, http.StatusInternalServerError, "failed to access media file"
	}
	if info.IsDir() {
		return "", nil, http.StatusBadRequest, "path must reference a media file"
	}
	if _, ok := mediaPreviewMIMEType(resolved); !ok {
		return "", nil, http.StatusBadRequest, "file type is not supported for media preview"
	}
	return resolved, info, 0, ""
}

// handleMediaPreview streams one media file from the requesting Agent's
// workspace. http.ServeContent handles byte ranges, which lets the player seek
// without loading the entire file into browser memory.
func handleMediaPreview(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		resolved, info, status, message := resolveMediaPreviewFile(cfg, r.URL.Query().Get("agentId"), r.URL.Query().Get("path"))
		if status != 0 {
			http.Error(w, message, status)
			return
		}
		file, err := os.Open(resolved)
		if err != nil {
			http.Error(w, "failed to open media file", http.StatusInternalServerError)
			return
		}
		defer file.Close()
		mimeType, _ := mediaPreviewMIMEType(resolved)
		w.Header().Set("Content-Type", mimeType)
		w.Header().Set("Content-Disposition", "inline")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Accept-Ranges", "bytes")
		http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	}
}

type VideoTrimRequest struct {
	AgentID    string                `json:"agentId"`
	Path       string                `json:"path"`
	Start      float64               `json:"start"`
	End        float64               `json:"end"`
	OutputName string                `json:"outputName,omitempty"`
	Crop       *VideoTrimCropRequest `json:"crop,omitempty"`
}

// VideoTrimCropRequest deliberately contains only normalized geometry. It is
// converted to a fixed, server-generated FFmpeg graph after validation; Site
// never gets to supply a filter expression or command argument.
type VideoTrimCropRequest struct {
	Shape      string               `json:"shape"`
	Background string               `json:"background,omitempty"`
	X          float64              `json:"x,omitempty"`
	Y          float64              `json:"y,omitempty"`
	Width      float64              `json:"width,omitempty"`
	Height     float64              `json:"height,omitempty"`
	Points     []VideoTrimCropPoint `json:"points,omitempty"`
}

type VideoTrimCropPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type validatedVideoTrimCrop struct {
	Shape      string
	Background string
	X          float64
	Y          float64
	Width      float64
	Height     float64
	Points     []VideoTrimCropPoint
}

type VideoTrimResponse struct {
	AgentID string `json:"agentId"`
	Path    string `json:"path"`
	SavedAs string `json:"savedAs,omitempty"`
	Content string `json:"content,omitempty"`
	Status  int    `json:"status"`
}

// VideoAudioExtractRequest deliberately carries no output path or command
// options. The server derives the MP3 name and always writes it below the
// requesting Agent's videos directory.
type VideoAudioExtractRequest struct {
	AgentID string `json:"agentId"`
	Path    string `json:"path"`
}

type VideoAudioExtractResponse struct {
	AgentID string `json:"agentId"`
	Path    string `json:"path"`
	SavedAs string `json:"savedAs,omitempty"`
	Content string `json:"content,omitempty"`
	Status  int    `json:"status"`
}

// VideoAudioEditRequest is the narrow project format used to replace a
// video's soundtrack. Source video pixels are never supplied by the client;
// the server resolves Path and uses it as the only video input.
type VideoAudioEditRequest struct {
	AgentID    string                `json:"agentId"`
	Path       string                `json:"path"`
	OutputName string                `json:"outputName,omitempty"`
	Tracks     []VideoAudioEditTrack `json:"tracks"`
}

// Original marks the optional first audio stream of the request video. Other
// tracks must name ordinary audio-containing files inside the same workspace.
type VideoAudioEditTrack struct {
	Original bool `json:"original,omitempty"`
	AudioMixTrack
}

type VideoAudioEditResponse struct {
	AgentID string `json:"agentId"`
	Path    string `json:"path"`
	SavedAs string `json:"savedAs,omitempty"`
	Renamed bool   `json:"renamed,omitempty"`
	Content string `json:"content,omitempty"`
	Status  int    `json:"status"`
}

// VideoRecordConvertRequest refers only to a browser-recorded WebM that Site
// staged under the requesting Agent's temporary workspace directory.
type VideoRecordConvertRequest struct {
	AgentID    string `json:"agentId"`
	Path       string `json:"path"`
	OutputName string `json:"outputName"`
}

type VideoRecordConvertResponse struct {
	AgentID string `json:"agentId"`
	Path    string `json:"path"`
	SavedAs string `json:"savedAs,omitempty"`
	Content string `json:"content,omitempty"`
	Status  int    `json:"status"`
}

const screenRecordingTemporaryDirectory = "tmp"

var screenRecordingTemporaryName = regexp.MustCompile(`^\.screen-recording-[a-z0-9-]{12,96}\.webm$`)

var videoTrimLookPathFn = integrationCommandLookPath
var videoTrimCommandContextFn = exec.CommandContext
var ffmpegEncoderProbeCommandContextFn = exec.CommandContext
var videoTrimLastOutputTimestamp int64

// ffmpegVideoEncoder describes one encoder in a small, ordered fallback list.
// Hardware encoders deliberately precede the software encoder, but their
// availability is verified against the FFmpeg binary that runs the job rather
// than inferred solely from the host OS.
type ffmpegVideoEncoder struct {
	Name     string
	Hardware bool
}

var ffmpegHardwareEncoderCache struct {
	sync.Mutex
	entries map[string]bool
}

func ffmpegHardwareEncoderAvailable(ffmpegPath, encoder string) bool {
	if strings.TrimSpace(ffmpegPath) == "" {
		return false
	}
	key := ffmpegPath + "\x00" + encoder
	ffmpegHardwareEncoderCache.Lock()
	available, found := ffmpegHardwareEncoderCache.entries[key]
	ffmpegHardwareEncoderCache.Unlock()
	if found {
		return available
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	command := ffmpegEncoderProbeCommandContextFn(ctx, ffmpegPath, "-hide_banner", "-loglevel", "error", "-h", "encoder="+encoder)
	available = command.Run() == nil && ctx.Err() == nil
	ffmpegHardwareEncoderCache.Lock()
	if ffmpegHardwareEncoderCache.entries == nil {
		ffmpegHardwareEncoderCache.entries = make(map[string]bool)
	}
	ffmpegHardwareEncoderCache.entries[key] = available
	ffmpegHardwareEncoderCache.Unlock()
	return available
}

func ffmpegH264EncoderCandidates(ffmpegPath string) []ffmpegVideoEncoder {
	encoders := []string{"h264_nvenc", "h264_qsv", "h264_amf"}
	if runtime.GOOS == "darwin" {
		encoders = append([]string{"h264_videotoolbox"}, encoders...)
	}
	candidates := make([]ffmpegVideoEncoder, 0, len(encoders)+1)
	for _, encoder := range encoders {
		if ffmpegHardwareEncoderAvailable(ffmpegPath, encoder) {
			candidates = append(candidates, ffmpegVideoEncoder{Name: encoder, Hardware: true})
		}
	}
	return append(candidates, ffmpegVideoEncoder{Name: "libx264"})
}

// runFFmpegVideoEncode prefers an FFmpeg-supported hardware encoder. It
// removes the partial temporary file a failed hardware encoder may leave
// behind before invoking the software fallback, since all of these callers
// use -n and must not mistake that partial output for a name collision.
func runFFmpegVideoEncode(ctx context.Context, operation, ffmpegPath, temporaryPath string, candidates []ffmpegVideoEncoder, argsFor func(ffmpegVideoEncoder) []string) ([]byte, error) {
	var output []byte
	var err error
	for index, encoder := range candidates {
		if encoder.Hardware {
			log.Printf("[ffmpeg] %s: using GPU encoder %s", operation, encoder.Name)
		} else {
			log.Printf("[ffmpeg] %s: using CPU encoder %s", operation, encoder.Name)
		}
		output, err = videoTrimCommandContextFn(ctx, ffmpegPath, argsFor(encoder)...).CombinedOutput()
		if err == nil {
			return output, nil
		}
		if encoder.Hardware && index+1 < len(candidates) {
			log.Printf("[ffmpeg] %s: GPU encoder %s failed: %s; removed partial output and falling back to CPU encoder %s", operation, encoder.Name, videoTrimCommandError(err, output), candidates[index+1].Name)
			if removeErr := os.Remove(temporaryPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				log.Printf("[ffmpeg] %s: remove failed GPU output %s: %v", operation, temporaryPath, removeErr)
			}
		}
	}
	return output, err
}

// FFmpegCheckResponse is intentionally narrow: the Site only needs to know
// whether it may enter video editing and, when unavailable, which configured
// request to send to the current chat. It never receives arbitrary runtime
// configuration.
type FFmpegCheckResponse struct {
	Available bool   `json:"available"`
	Install   string `json:"install,omitempty"`
	Content   string `json:"content,omitempty"`
	Status    int    `json:"status"`
}

type ffmpegCheckConfig struct {
	CacheFor time.Duration
	Install  string
}

var ffmpegCheckLookPathFn = integrationCommandLookPath
var ffmpegCheckNowFn = time.Now
var ffmpegCheckCache struct {
	sync.Mutex
	availableAt time.Time
}

func writeFFmpegCheckResp(w http.ResponseWriter, httpStatus int, resp FFmpegCheckResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(resp)
}

func ffmpegCheckPositiveHours(value interface{}) (int64, bool) {
	var hours int64
	maxHours := int64(math.MaxInt64 / int64(time.Hour))
	switch number := value.(type) {
	case float64:
		if math.IsNaN(number) || math.IsInf(number, 0) || number != math.Trunc(number) || number < 1 || number > float64(maxHours) {
			return 0, false
		}
		hours = int64(number)
	case float32:
		float := float64(number)
		if math.IsNaN(float) || math.IsInf(float, 0) || float != math.Trunc(float) || float < 1 || float > float64(maxHours) {
			return 0, false
		}
		hours = int64(number)
	case int:
		hours = int64(number)
	case int64:
		hours = number
	case int32:
		hours = int64(number)
	case json.Number:
		parsed, err := strconv.ParseInt(string(number), 10, 64)
		if err != nil {
			return 0, false
		}
		hours = parsed
	default:
		return 0, false
	}
	return hours, hours > 0 && hours <= maxHours
}

func parseFFmpegCheckConfig(raw map[string]interface{}) (ffmpegCheckConfig, error) {
	if raw == nil {
		return ffmpegCheckConfig{}, errors.New("config/config.json.ffmpeg 配置缺失")
	}
	value, ok := raw["ffmpeg"]
	if !ok {
		return ffmpegCheckConfig{}, errors.New("config/config.json.ffmpeg 配置缺失")
	}
	section, ok := value.(map[string]interface{})
	if !ok || section == nil {
		return ffmpegCheckConfig{}, errors.New("config/config.json.ffmpeg 必须是对象")
	}
	hours, ok := ffmpegCheckPositiveHours(section["check"])
	if !ok {
		return ffmpegCheckConfig{}, errors.New("config/config.json.ffmpeg.check 必须是正整数（小时）")
	}
	install, ok := section["install"].(string)
	install = strings.TrimSpace(install)
	if !ok || install == "" {
		return ffmpegCheckConfig{}, errors.New("config/config.json.ffmpeg.install 必须是非空字符串")
	}
	return ffmpegCheckConfig{CacheFor: time.Duration(hours) * time.Hour, Install: install}, nil
}

func ffmpegDependenciesAvailable(cacheFor time.Duration) bool {
	now := ffmpegCheckNowFn()
	ffmpegCheckCache.Lock()
	availableAt := ffmpegCheckCache.availableAt
	ffmpegCheckCache.Unlock()
	if !availableAt.IsZero() && now.Before(availableAt.Add(cacheFor)) {
		return true
	}

	if _, err := ffmpegCheckLookPathFn("ffmpeg"); err != nil {
		return false
	}
	if _, err := ffmpegCheckLookPathFn("ffprobe"); err != nil {
		return false
	}

	ffmpegCheckCache.Lock()
	ffmpegCheckCache.availableAt = now
	ffmpegCheckCache.Unlock()
	return true
}

// handleFFmpegCheck checks only executable availability. It must not execute
// an installer or create media, and unsuccessful lookups deliberately do not
// update the in-memory success cache.
func handleFFmpegCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeFFmpegCheckResp(w, http.StatusMethodNotAllowed, FFmpegCheckResponse{Content: "仅支持 GET 请求", Status: 1})
			return
		}
		status, response := checkFFmpegDependency()
		writeFFmpegCheckResp(w, status, response)
	}
}

// checkFFmpegDependency contains the complete availability rule shared by the
// HTTP endpoint and the asynchronous startup preflight.
func checkFFmpegDependency() (int, FFmpegCheckResponse) {
	raw, _, err := readIntegrationStartupConfigRaw()
	if err != nil {
		return http.StatusInternalServerError, FFmpegCheckResponse{Content: "读取 config/config.json 失败: " + err.Error(), Status: 1}
	}
	config, err := parseFFmpegCheckConfig(raw)
	if err != nil {
		return http.StatusBadRequest, FFmpegCheckResponse{Content: err.Error(), Status: 1}
	}
	if !ffmpegDependenciesAvailable(config.CacheFor) {
		return http.StatusOK, FFmpegCheckResponse{Available: false, Install: config.Install, Content: "未检测到 FFmpeg 或 FFprobe", Status: 0}
	}
	return http.StatusOK, FFmpegCheckResponse{Available: true, Status: 0}
}

// BubblewrapCheckResponse tells the Site whether the current runtime needs
// bubblewrap before it can enable a session sandbox. Bubblewrap is only the
// Linux/WSL sandbox backend, so non-Linux runtimes deliberately remain
// available without requiring this optional dependency.
type BubblewrapCheckResponse struct {
	Available bool   `json:"available"`
	Required  bool   `json:"required"`
	Install   string `json:"install,omitempty"`
	Content   string `json:"content,omitempty"`
	Status    int    `json:"status"`
}

type bubblewrapCheckConfig struct {
	CacheFor time.Duration
	Install  string
}

var bubblewrapCheckLookPathFn = integrationCommandLookPath
var bubblewrapCheckNowFn = time.Now
var bubblewrapCheckCache struct {
	sync.Mutex
	availableAt time.Time
}

func writeBubblewrapCheckResp(w http.ResponseWriter, httpStatus int, resp BubblewrapCheckResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(resp)
}

func parseBubblewrapCheckConfig(raw map[string]interface{}) (bubblewrapCheckConfig, error) {
	if raw == nil {
		return bubblewrapCheckConfig{}, errors.New("config/config.json.bubblewrap 配置缺失")
	}
	value, ok := raw["bubblewrap"]
	if !ok {
		return bubblewrapCheckConfig{}, errors.New("config/config.json.bubblewrap 配置缺失")
	}
	section, ok := value.(map[string]interface{})
	if !ok || section == nil {
		return bubblewrapCheckConfig{}, errors.New("config/config.json.bubblewrap 必须是对象")
	}
	hours, ok := ffmpegCheckPositiveHours(section["check"])
	if !ok {
		return bubblewrapCheckConfig{}, errors.New("config/config.json.bubblewrap.check 必须是正整数（小时）")
	}
	install, ok := section["install"].(string)
	install = strings.TrimSpace(install)
	if !ok || install == "" {
		return bubblewrapCheckConfig{}, errors.New("config/config.json.bubblewrap.install 必须是非空字符串")
	}
	return bubblewrapCheckConfig{CacheFor: time.Duration(hours) * time.Hour, Install: install}, nil
}

func bubblewrapRequired() bool {
	return integrationRuntimeGOOS == "linux"
}

func bubblewrapAvailable(cacheFor time.Duration) bool {
	now := bubblewrapCheckNowFn()
	bubblewrapCheckCache.Lock()
	availableAt := bubblewrapCheckCache.availableAt
	bubblewrapCheckCache.Unlock()
	if !availableAt.IsZero() && now.Before(availableAt.Add(cacheFor)) {
		return true
	}

	if _, err := bubblewrapCheckLookPathFn("bwrap"); err != nil {
		return false
	}

	bubblewrapCheckCache.Lock()
	bubblewrapCheckCache.availableAt = now
	bubblewrapCheckCache.Unlock()
	return true
}

func handleBubblewrapCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeBubblewrapCheckResp(w, http.StatusMethodNotAllowed, BubblewrapCheckResponse{Content: "仅支持 GET 请求", Status: 1})
			return
		}
		status, response := checkBubblewrapDependency()
		writeBubblewrapCheckResp(w, status, response)
	}
}

// checkBubblewrapDependency is intentionally limited to availability. It
// never installs or executes bwrap; the configured installation request is
// returned to the Site so that the user can choose to send it to the current
// chat, matching the FFmpeg dependency flow.
func checkBubblewrapDependency() (int, BubblewrapCheckResponse) {
	if !bubblewrapRequired() {
		return http.StatusOK, BubblewrapCheckResponse{Available: true, Required: false, Status: 0}
	}
	raw, _, err := readIntegrationStartupConfigRaw()
	if err != nil {
		return http.StatusInternalServerError, BubblewrapCheckResponse{Required: true, Content: "读取 config/config.json 失败: " + err.Error(), Status: 1}
	}
	config, err := parseBubblewrapCheckConfig(raw)
	if err != nil {
		return http.StatusBadRequest, BubblewrapCheckResponse{Required: true, Content: err.Error(), Status: 1}
	}
	if !bubblewrapAvailable(config.CacheFor) {
		return http.StatusOK, BubblewrapCheckResponse{Available: false, Required: true, Install: config.Install, Content: "未检测到 Bubblewrap（bwrap）", Status: 0}
	}
	return http.StatusOK, BubblewrapCheckResponse{Available: true, Required: true, Status: 0}
}

func writeVideoTrimResp(w http.ResponseWriter, httpStatus int, resp VideoTrimResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(resp)
}

func writeVideoAudioExtractResp(w http.ResponseWriter, httpStatus int, resp VideoAudioExtractResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(resp)
}

func writeVideoAudioEditResp(w http.ResponseWriter, httpStatus int, resp VideoAudioEditResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(resp)
}

func writeVideoRecordConvertResp(w http.ResponseWriter, httpStatus int, resp VideoRecordConvertResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(resp)
}

func isScreenRecordingTemporaryRelativePath(relPath string) bool {
	cleanPath := filepath.ToSlash(filepath.Clean(relPath))
	return path.Dir(cleanPath) == screenRecordingTemporaryDirectory && screenRecordingTemporaryName.MatchString(path.Base(cleanPath))
}

// videoRecordAvailableOutputPath returns a currently unused final MP4 path.
// A save still creates it atomically with os.Link below, so this preflight is
// only a friendly collision avoidance step rather than a race-prone guarantee.
func videoRecordAvailableOutputPath(workspace, outputName string, now time.Time) (string, error) {
	for attempt := 0; attempt < 64; attempt++ {
		candidateName := outputName
		if attempt > 0 {
			candidateName = appendTimestampToFilename(outputName, now.Add(time.Duration(attempt)*time.Nanosecond))
		}
		outputPath, err := resolveCaseInsensitiveUnderRoot(workspace, filepath.Join("videos", candidateName))
		if err != nil || !ensureWritablePathWithinRoot(workspace, outputPath) {
			return "", errors.New("无法在工作目录中创建 MP4")
		}
		if _, err := os.Lstat(outputPath); errors.Is(err, os.ErrNotExist) {
			return outputPath, nil
		} else if err != nil {
			return "", fmt.Errorf("无法检查 MP4 输出路径: %w", err)
		}
	}
	return "", errors.New("无法生成不重复的 MP4 文件名")
}

// handleVideoRecordConvert turns one browser-staged WebM into the final MP4.
// Its narrow input namespace deliberately prevents this endpoint from becoming
// a generic workspace transcoder.
func handleVideoRecordConvert(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeVideoRecordConvertResp(w, http.StatusMethodNotAllowed, VideoRecordConvertResponse{Content: "method not allowed", Status: 1})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		defer r.Body.Close()
		var request VideoRecordConvertRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeVideoRecordConvertResp(w, http.StatusBadRequest, VideoRecordConvertResponse{Content: "请求体格式错误", Status: 1})
			return
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			writeVideoRecordConvertResp(w, http.StatusBadRequest, VideoRecordConvertResponse{AgentID: request.AgentID, Path: request.Path, Content: "请求体只能包含一个 JSON 对象", Status: 1})
			return
		}
		request.AgentID = strings.TrimSpace(request.AgentID)
		request.Path = normalizeQuotedPathArg(request.Path)
		request.OutputName = strings.TrimSpace(normalizeQuotedPathArg(request.OutputName))
		if !isValidMediaPreviewAgentID(request.AgentID) || request.Path == "" {
			writeVideoRecordConvertResp(w, http.StatusBadRequest, VideoRecordConvertResponse{AgentID: request.AgentID, Path: request.Path, Content: "agentId 和临时录制路径不能为空", Status: 1})
			return
		}
		if !isScreenRecordingTemporaryRelativePath(request.Path) {
			writeVideoRecordConvertResp(w, http.StatusBadRequest, VideoRecordConvertResponse{AgentID: request.AgentID, Path: request.Path, Content: "仅支持本次录屏产生的临时 WebM 文件", Status: 1})
			return
		}
		if !isSafeVideoTrimOutputName(request.OutputName) {
			writeVideoRecordConvertResp(w, http.StatusBadRequest, VideoRecordConvertResponse{AgentID: request.AgentID, Path: request.Path, Content: "输出文件名必须是有效的 .mp4 文件名", Status: 1})
			return
		}

		workspace, err := getWorkspaceByAgentID(cfg, request.AgentID)
		if err != nil {
			writeVideoRecordConvertResp(w, http.StatusNotFound, VideoRecordConvertResponse{AgentID: request.AgentID, Path: request.Path, Content: "agent not found", Status: 1})
			return
		}
		cleanSourcePath := filepath.Clean(request.Path)
		sourcePath, err := resolveCaseInsensitiveUnderRoot(workspace, cleanSourcePath)
		if err != nil || !ensurePathWithinRoot(workspace, sourcePath) {
			writeVideoRecordConvertResp(w, http.StatusBadRequest, VideoRecordConvertResponse{AgentID: request.AgentID, Path: request.Path, Content: "临时录制路径无效", Status: 1})
			return
		}
		sourceInfo, err := os.Lstat(sourcePath)
		if err != nil {
			status := http.StatusInternalServerError
			message := "无法读取临时录制文件"
			if errors.Is(err, os.ErrNotExist) {
				status = http.StatusNotFound
				message = "临时录制文件不存在"
			}
			writeVideoRecordConvertResp(w, status, VideoRecordConvertResponse{AgentID: request.AgentID, Path: request.Path, Content: message, Status: 1})
			return
		}
		if sourceInfo.IsDir() || sourceInfo.Mode()&os.ModeSymlink != 0 {
			writeVideoRecordConvertResp(w, http.StatusBadRequest, VideoRecordConvertResponse{AgentID: request.AgentID, Path: request.Path, Content: "临时录制文件无效", Status: 1})
			return
		}
		// Every path after this point has passed the narrow temporary namespace
		// validation, so clean it up on both successful and failed conversions.
		defer func() {
			if err := os.Remove(sourcePath); err != nil && !errors.Is(err, os.ErrNotExist) {
				log.Printf("remove screen-recording temporary source %s: %v", sourcePath, err)
			}
		}()

		outputPath, err := videoRecordAvailableOutputPath(workspace, request.OutputName, time.Now())
		if err != nil {
			writeVideoRecordConvertResp(w, http.StatusInternalServerError, VideoRecordConvertResponse{AgentID: request.AgentID, Path: request.Path, Content: err.Error(), Status: 1})
			return
		}
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil || !ensureWritablePathWithinRoot(workspace, outputPath) {
			writeVideoRecordConvertResp(w, http.StatusInternalServerError, VideoRecordConvertResponse{AgentID: request.AgentID, Path: request.Path, Content: "无法创建 videos 目录", Status: 1})
			return
		}

		ffmpegPath, ffmpegErr := videoTrimLookPathFn("ffmpeg")
		ffprobePath, ffprobeErr := videoTrimLookPathFn("ffprobe")
		if ffmpegErr != nil || ffprobeErr != nil {
			writeVideoRecordConvertResp(w, http.StatusServiceUnavailable, VideoRecordConvertResponse{AgentID: request.AgentID, Path: request.Path, Content: "FFmpeg 或 FFprobe 未安装，无法生成 MP4", Status: 1})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
		defer cancel()
		bitrate := videoTrimProbeBitrate(ctx, ffprobePath, sourcePath)
		temporaryPath := strings.TrimSuffix(outputPath, ".mp4") + "." + strconv.FormatInt(time.Now().UnixNano(), 10) + ".tmp.mp4"
		if !ensureWritablePathWithinRoot(workspace, temporaryPath) {
			writeVideoRecordConvertResp(w, http.StatusBadRequest, VideoRecordConvertResponse{AgentID: request.AgentID, Path: request.Path, Content: "无法创建临时 MP4", Status: 1})
			return
		}
		defer os.Remove(temporaryPath)
		argsForEncoder := func(encoder ffmpegVideoEncoder) []string {
			args := []string{
				"-hide_banner", "-loglevel", "error", "-nostdin",
				"-i", sourcePath,
				"-map", "0:v:0", "-map", "0:a?",
				// H.264 with 4:2:0 chroma requires even dimensions. Browser display
				// capture may legitimately produce odd widths or heights (for example
				// 943x1235). Pad only the extra edge pixel instead of cropping source
				// content so full-frame recording always remains complete.
				"-vf", "pad=ceil(iw/2)*2:ceil(ih/2)*2:0:0",
				"-pix_fmt", "yuv420p",
				"-c:v", encoder.Name, "-c:a", "aac",
			}
			if bitrate > 0 {
				args = append(args, "-b:v", strconv.FormatInt(bitrate, 10))
			}
			return append(args, "-movflags", "+faststart", "-f", "mp4", "-n", temporaryPath)
		}
		commandOutput, err := runFFmpegVideoEncode(ctx, "screen recording conversion", ffmpegPath, temporaryPath, ffmpegH264EncoderCandidates(ffmpegPath), argsForEncoder)
		if err != nil {
			message := "视频转码失败：" + videoTrimCommandError(err, commandOutput)
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				message = "视频转码超时"
			}
			writeVideoRecordConvertResp(w, http.StatusInternalServerError, VideoRecordConvertResponse{AgentID: request.AgentID, Path: request.Path, Content: message, Status: 1})
			return
		}
		info, err := os.Stat(temporaryPath)
		if err != nil || info.Size() == 0 {
			writeVideoRecordConvertResp(w, http.StatusInternalServerError, VideoRecordConvertResponse{AgentID: request.AgentID, Path: request.Path, Content: "MP4 转码未生成有效输出", Status: 1})
			return
		}
		if err := os.Remove(sourcePath); err != nil {
			writeVideoRecordConvertResp(w, http.StatusInternalServerError, VideoRecordConvertResponse{AgentID: request.AgentID, Path: request.Path, Content: "清理临时 WebM 失败", Status: 1})
			return
		}
		var linkErr error
		for attempt := 0; attempt < 64; attempt++ {
			linkErr = os.Link(temporaryPath, outputPath)
			if linkErr == nil {
				break
			}
			if !errors.Is(linkErr, fs.ErrExist) {
				writeVideoRecordConvertResp(w, http.StatusInternalServerError, VideoRecordConvertResponse{AgentID: request.AgentID, Path: request.Path, Content: "保存 MP4 失败", Status: 1})
				return
			}
			outputPath, err = videoRecordAvailableOutputPath(workspace, request.OutputName, time.Now().Add(time.Duration(attempt+1)*time.Nanosecond))
			if err != nil {
				writeVideoRecordConvertResp(w, http.StatusInternalServerError, VideoRecordConvertResponse{AgentID: request.AgentID, Path: request.Path, Content: err.Error(), Status: 1})
				return
			}
		}
		if linkErr != nil {
			writeVideoRecordConvertResp(w, http.StatusInternalServerError, VideoRecordConvertResponse{AgentID: request.AgentID, Path: request.Path, Content: "无法生成不重复的 MP4 文件名", Status: 1})
			return
		}
		if err := os.Remove(temporaryPath); err != nil {
			_ = os.Remove(outputPath)
			writeVideoRecordConvertResp(w, http.StatusInternalServerError, VideoRecordConvertResponse{AgentID: request.AgentID, Path: request.Path, Content: "清理临时 MP4 失败", Status: 1})
			return
		}
		flushAgentCache()
		writeVideoRecordConvertResp(w, http.StatusOK, VideoRecordConvertResponse{AgentID: request.AgentID, Path: request.Path, SavedAs: outputPath, Status: 0})
	}
}

func formatVideoTrimSeconds(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func nextVideoTrimOutputTime(now time.Time) time.Time {
	requested := now.UnixNano()
	for {
		previous := atomic.LoadInt64(&videoTrimLastOutputTimestamp)
		next := requested
		if next <= previous {
			next = previous + 1
		}
		if atomic.CompareAndSwapInt64(&videoTrimLastOutputTimestamp, previous, next) {
			return time.Unix(0, next).In(now.Location())
		}
	}
}

func videoTrimOutputRelativePath(sourceRelativePath string, now time.Time) string {
	name := filepath.Base(sourceRelativePath)
	base := strings.TrimSuffix(name, filepath.Ext(name))
	if base == "" {
		base = "video"
	}
	outputName := base + "_trim_" + timestampWithNanoseconds(now) + ".mp4"
	return filepath.Join("videos", outputName)
}

func isSafeVideoTrimOutputName(name string) bool {
	return isSafeSaveAsNewFilename(name) && strings.EqualFold(filepath.Ext(name), ".mp4") && len(strings.TrimSuffix(name, filepath.Ext(name))) > 0
}

func videoAudioExtractOutputName(sourceRelativePath string) string {
	name := filepath.Base(sourceRelativePath)
	base := strings.TrimSuffix(name, filepath.Ext(name))
	if base == "" || base == "." {
		base = "video"
	}
	return base + ".mp3"
}

func videoAudioEditOutputName(sourceRelativePath string) string {
	name := filepath.Base(sourceRelativePath)
	base := strings.TrimSuffix(name, filepath.Ext(name))
	if base == "" || base == "." {
		base = "video"
	}
	return base + "_edit.mp4"
}

// videoAudioEditAvailableOutputPath picks a name without replacing any prior
// export. The final link in the handler is still the collision authority.
func videoAudioEditAvailableOutputPath(workspace, outputName string, now time.Time) (string, bool, error) {
	for attempt := 0; attempt < 64; attempt++ {
		candidateName := outputName
		if attempt > 0 {
			candidateName = appendTimestampToFilename(outputName, now.Add(time.Duration(attempt)*time.Nanosecond))
		}
		outputPath, err := resolveCaseInsensitiveUnderRoot(workspace, filepath.Join("videos", candidateName))
		if err != nil || !ensureWritablePathWithinRoot(workspace, outputPath) {
			return "", false, errors.New("无法在工作目录中创建 MP4")
		}
		if _, err := os.Lstat(outputPath); errors.Is(err, os.ErrNotExist) {
			return outputPath, attempt > 0, nil
		} else if err != nil {
			return "", false, fmt.Errorf("无法检查 MP4 输出路径: %w", err)
		}
	}
	return "", false, errors.New("无法生成不重复的 MP4 文件名")
}

// videoAudioExtractAvailableOutputPath finds an unused MP3 name. The final
// os.Link in handleVideoAudioExtract remains the authoritative collision
// guard, so a competing extraction can never overwrite an existing file.
func videoAudioExtractAvailableOutputPath(workspace, outputDirectory, outputName string, now time.Time) (string, error) {
	for attempt := 0; attempt < 64; attempt++ {
		candidateName := outputName
		if attempt > 0 {
			candidateName = appendTimestampToFilename(outputName, now.Add(time.Duration(attempt)*time.Nanosecond))
		}
		outputPath, err := resolveCaseInsensitiveUnderRoot(workspace, filepath.Join(outputDirectory, candidateName))
		if err != nil || !ensureWritablePathWithinRoot(workspace, outputPath) {
			return "", errors.New("无法在工作目录中创建 MP3")
		}
		if _, err := os.Lstat(outputPath); errors.Is(err, os.ErrNotExist) {
			return outputPath, nil
		} else if err != nil {
			return "", fmt.Errorf("无法检查 MP3 输出路径: %w", err)
		}
	}
	return "", errors.New("无法生成不重复的 MP3 文件名")
}

const (
	videoTrimCropMinimumSize   = 0.001
	videoTrimCropMaximumPoints = 48
	// Keep source-canvas output for high-resolution pen crops. Such filters can
	// legitimately take longer than simple time-only trimming, so they get a
	// bounded but more practical conversion window.
	videoTrimConversionTimeout = 15 * time.Minute
)

func isValidVideoTrimCropCoordinate(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1
}

func videoTrimCropNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', 8, 64)
}

func videoTrimCropRoundedNumber(value float64) float64 {
	rounded, err := strconv.ParseFloat(videoTrimCropNumber(value), 64)
	if err != nil {
		return 0
	}
	return rounded
}

func validateVideoTrimCrop(request *VideoTrimCropRequest) (*validatedVideoTrimCrop, error) {
	if request == nil {
		return nil, nil
	}
	crop := &validatedVideoTrimCrop{
		Shape:      strings.ToLower(strings.TrimSpace(request.Shape)),
		Background: strings.ToLower(strings.TrimSpace(request.Background)),
	}
	if crop.Background == "" {
		crop.Background = "black"
	}
	if crop.Background != "black" && crop.Background != "white" {
		return nil, errors.New("补边色仅支持 black 或 white")
	}
	switch crop.Shape {
	case "rect", "ellipse":
		if !isValidVideoTrimCropCoordinate(request.X) || !isValidVideoTrimCropCoordinate(request.Y) ||
			!isValidVideoTrimCropCoordinate(request.Width) || !isValidVideoTrimCropCoordinate(request.Height) ||
			request.Width < videoTrimCropMinimumSize || request.Height < videoTrimCropMinimumSize ||
			request.X+request.Width > 1+1e-9 || request.Y+request.Height > 1+1e-9 {
			return nil, errors.New("裁剪范围必须位于源画面内且具有有效面积")
		}
		crop.X = request.X
		crop.Y = request.Y
		crop.Width = request.Width
		crop.Height = request.Height
		return crop, nil
	case "polygon":
		if len(request.Points) < 3 || len(request.Points) > videoTrimCropMaximumPoints {
			return nil, fmt.Errorf("钢笔闭合区域必须包含 3 到 %d 个点", videoTrimCropMaximumPoints)
		}
		minX, minY, maxX, maxY := 1.0, 1.0, 0.0, 0.0
		areaTwice := 0.0
		crop.Points = make([]VideoTrimCropPoint, len(request.Points))
		for index, point := range request.Points {
			if !isValidVideoTrimCropCoordinate(point.X) || !isValidVideoTrimCropCoordinate(point.Y) {
				return nil, errors.New("钢笔闭合区域的坐标必须位于源画面内")
			}
			crop.Points[index] = point
			minX = math.Min(minX, point.X)
			minY = math.Min(minY, point.Y)
			maxX = math.Max(maxX, point.X)
			maxY = math.Max(maxY, point.Y)
			next := request.Points[(index+1)%len(request.Points)]
			areaTwice += point.X*next.Y - next.X*point.Y
		}
		if maxX-minX < videoTrimCropMinimumSize || maxY-minY < videoTrimCropMinimumSize || math.Abs(areaTwice) < videoTrimCropMinimumSize*videoTrimCropMinimumSize {
			return nil, errors.New("钢笔闭合区域必须具有有效面积")
		}
		crop.X = minX
		crop.Y = minY
		crop.Width = maxX - minX
		crop.Height = maxY - minY
		return crop, nil
	default:
		return nil, errors.New("裁剪形状仅支持 rect、ellipse 或 polygon")
	}
}

func videoTrimCropInsideExpression(crop *validatedVideoTrimCrop) string {
	if crop.Shape == "ellipse" {
		return "lte((X-W/2)*(X-W/2)/(W*W/4)+(Y-H/2)*(Y-H/2)/(H*H/4)\\,1)"
	}
	terms := make([]string, 0, len(crop.Points))
	for index, first := range crop.Points {
		second := crop.Points[(index+1)%len(crop.Points)]
		// Round before comparing because the same eight-decimal representation is
		// later embedded in FFmpeg's expression. Without this, two nearly-level
		// pen points could serialize to the same Y and leave a zero denominator.
		firstX := videoTrimCropRoundedNumber((first.X - crop.X) / crop.Width)
		firstY := videoTrimCropRoundedNumber((first.Y - crop.Y) / crop.Height)
		secondX := videoTrimCropRoundedNumber((second.X - crop.X) / crop.Width)
		secondY := videoTrimCropRoundedNumber((second.Y - crop.Y) / crop.Height)
		if firstY == secondY {
			continue
		}
		fx, fy := videoTrimCropNumber(firstX), videoTrimCropNumber(firstY)
		sx, sy := videoTrimCropNumber(secondX), videoTrimCropNumber(secondY)
		terms = append(terms, "gt(0\\,(Y-"+fy+"*H)*(Y-"+sy+"*H))*gt("+fx+"*W+(Y-"+fy+"*H)*("+sx+"*W-"+fx+"*W)/("+sy+"*H-"+fy+"*H)\\,X)")
	}
	return "mod(" + strings.Join(terms, "+") + "\\,2)"
}

func videoTrimCropFilter(crop *validatedVideoTrimCrop) string {
	width, height := videoTrimCropNumber(crop.Width), videoTrimCropNumber(crop.Height)
	x, y := videoTrimCropNumber(crop.X), videoTrimCropNumber(crop.Y)
	selection := "crop=w=ceil(iw*" + width + "):h=ceil(ih*" + height + "):x=floor(iw*" + x + "):y=floor(ih*" + y + ")"
	backgroundLuma := "0"
	if crop.Background == "white" {
		backgroundLuma = "255"
	}
	selectionFilter := "[trimCropInput]" + selection + "[trimCropSelection];"
	if crop.Shape != "rect" {
		inside := videoTrimCropInsideExpression(crop)
		// Generate the shape in the luma channel of a neutral YUV mask. A gray
		// mask makes some FFmpeg builds negotiate maskedmerge down to grayscale,
		// which removes the selected video's color. Keeping all three inputs in
		// yuv444p preserves its chroma while evaluating the shape only once.
		// Composite with the mask before scale/overlay. maskedmerge selects its
		// second input wherever the mask is white, so the fill must be first and
		// the source second. This
		// avoids alpha-pixel-format negotiation in
		// older FFmpeg builds, which caused ellipse selections to fail despite
		// valid source dimensions.
		selectionFilter = "[trimCropInput]" + selection + ",split=3[trimCropSourceRaw][trimCropMaskSource][trimCropFillSource];" +
			"[trimCropSourceRaw]format=yuv444p[trimCropSource];" +
			"[trimCropMaskSource]format=yuv444p,geq=lum='if(" + inside + "\\,255\\,0)':cb=128:cr=128[trimCropMask];" +
			"[trimCropFillSource]format=yuv444p,lut=y=" + backgroundLuma + ":u=128:v=128[trimCropFill];" +
			"[trimCropFill][trimCropSource][trimCropMask]maskedmerge[trimCropSelection];"
	}
	return "[0:v]split=2[trimCropBase][trimCropInput];" +
		selectionFilter +
		"[trimCropSelection][trimCropBase]scale2ref=w=ref_w:h=ref_h:force_original_aspect_ratio=decrease[trimCropScaled][trimCropBaseRef];" +
		"[trimCropBaseRef]format=yuv444p,lut=y=" + backgroundLuma + ":u=128:v=128[trimCropBackground];" +
		"[trimCropBackground][trimCropScaled]overlay=(W-w)/2:(H-h)/2:shortest=1,pad=ceil(iw/2)*2:ceil(ih/2)*2:0:0:" + crop.Background + ",format=yuv420p[trimCropOutput]"
}

func videoTrimProbeBitrate(ctx context.Context, ffprobePath, sourcePath string) int64 {
	cmd := videoTrimCommandContextFn(ctx, ffprobePath,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=bit_rate",
		"-of", "default=noprint_wrappers=1:nokey=1",
		sourcePath,
	)
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	bitrate, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil || bitrate <= 0 {
		return 0
	}
	return bitrate
}

// videoAudioExtractHasAudioStream keeps the user-facing no-audio case
// distinct from a failed conversion. It uses only server-defined FFprobe
// arguments and returns no stream details to the browser.
func videoAudioExtractHasAudioStream(ctx context.Context, ffprobePath, sourcePath string) (bool, error) {
	cmd := videoTrimCommandContextFn(ctx, ffprobePath,
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "stream=index",
		"-of", "default=noprint_wrappers=1:nokey=1",
		sourcePath,
	)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(output)) != "", nil
}

func videoTrimCommandError(err error, output []byte) string {
	message := strings.TrimSpace(string(output))
	if len(message) > 800 {
		message = message[:800] + "…"
	}
	if message == "" {
		return err.Error()
	}
	return message
}

// handleVideoTrim creates a new MP4 from a video in the requesting Agent's
// workspace. It never edits the source and deliberately fixes the codecs to
// H.264/AAC so the selected interval can be re-encoded accurately.
func handleVideoTrim(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeVideoTrimResp(w, http.StatusMethodNotAllowed, VideoTrimResponse{Content: "method not allowed", Status: 1})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		defer r.Body.Close()
		var request VideoTrimRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeVideoTrimResp(w, http.StatusBadRequest, VideoTrimResponse{Content: "请求体格式错误", Status: 1})
			return
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			writeVideoTrimResp(w, http.StatusBadRequest, VideoTrimResponse{AgentID: request.AgentID, Path: request.Path, Content: "请求体只能包含一个 JSON 对象", Status: 1})
			return
		}
		request.AgentID = strings.TrimSpace(request.AgentID)
		request.Path = normalizeQuotedPathArg(request.Path)
		request.OutputName = strings.TrimSpace(normalizeQuotedPathArg(request.OutputName))
		if !isValidMediaPreviewAgentID(request.AgentID) || request.Path == "" {
			writeVideoTrimResp(w, http.StatusBadRequest, VideoTrimResponse{AgentID: request.AgentID, Path: request.Path, Content: "agentId 和相对路径不能为空", Status: 1})
			return
		}
		// The range sliders use 0.5-second steps, but the initial end value is the
		// media's real duration. That duration commonly has a fractional tail (for
		// example 12.4s), and represents the user's "whole video" selection.
		// Accept any finite non-negative time rather than rejecting a valid full
		// duration merely because it is not exactly on a slider tick.
		if math.IsNaN(request.Start) || math.IsNaN(request.End) || math.IsInf(request.Start, 0) || math.IsInf(request.End, 0) || request.Start < 0 || request.End <= request.Start {
			writeVideoTrimResp(w, http.StatusBadRequest, VideoTrimResponse{AgentID: request.AgentID, Path: request.Path, Content: "开始和结束时间必须为有效的非负秒数，且结束时间大于开始时间", Status: 1})
			return
		}
		if request.OutputName != "" && !isSafeVideoTrimOutputName(request.OutputName) {
			writeVideoTrimResp(w, http.StatusBadRequest, VideoTrimResponse{AgentID: request.AgentID, Path: request.Path, Content: "新视频文件名必须是有效的 .mp4 文件名", Status: 1})
			return
		}
		crop, err := validateVideoTrimCrop(request.Crop)
		if err != nil {
			writeVideoTrimResp(w, http.StatusBadRequest, VideoTrimResponse{AgentID: request.AgentID, Path: request.Path, Content: err.Error(), Status: 1})
			return
		}

		sourcePath, _, sourceStatus, sourceMessage := resolveMediaPreviewFile(cfg, request.AgentID, request.Path)
		if sourceStatus != 0 {
			writeVideoTrimResp(w, sourceStatus, VideoTrimResponse{AgentID: request.AgentID, Path: request.Path, Content: sourceMessage, Status: 1})
			return
		}
		mimeType, _ := mediaPreviewMIMEType(sourcePath)
		if !strings.HasPrefix(mimeType, "video/") {
			writeVideoTrimResp(w, http.StatusBadRequest, VideoTrimResponse{AgentID: request.AgentID, Path: request.Path, Content: "仅支持切分视频文件", Status: 1})
			return
		}
		workspace, err := getWorkspaceByAgentID(cfg, request.AgentID)
		if err != nil {
			writeVideoTrimResp(w, http.StatusNotFound, VideoTrimResponse{AgentID: request.AgentID, Path: request.Path, Content: "agent not found", Status: 1})
			return
		}
		sourceRelativePath, err := filepath.Rel(workspace, sourcePath)
		if err != nil || sourceRelativePath == "." || strings.HasPrefix(sourceRelativePath, ".."+string(filepath.Separator)) {
			writeVideoTrimResp(w, http.StatusBadRequest, VideoTrimResponse{AgentID: request.AgentID, Path: request.Path, Content: "path must be a workspace-relative video path", Status: 1})
			return
		}
		outputRelativePath := videoTrimOutputRelativePath(sourceRelativePath, nextVideoTrimOutputTime(time.Now()))
		if request.OutputName != "" {
			outputRelativePath = filepath.Join("videos", request.OutputName)
		}
		outputPath, err := resolveCaseInsensitiveUnderRoot(workspace, outputRelativePath)
		if err != nil || !ensureWritablePathWithinRoot(workspace, outputPath) {
			writeVideoTrimResp(w, http.StatusBadRequest, VideoTrimResponse{AgentID: request.AgentID, Path: request.Path, Content: "无法在工作目录中创建新视频", Status: 1})
			return
		}
		if _, err := os.Lstat(outputPath); err == nil {
			writeVideoTrimResp(w, http.StatusConflict, VideoTrimResponse{AgentID: request.AgentID, Path: request.Path, Content: "目标视频已存在，请重试", Status: 1})
			return
		} else if !os.IsNotExist(err) {
			writeVideoTrimResp(w, http.StatusInternalServerError, VideoTrimResponse{AgentID: request.AgentID, Path: request.Path, Content: "无法检查目标视频", Status: 1})
			return
		}
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			writeVideoTrimResp(w, http.StatusInternalServerError, VideoTrimResponse{AgentID: request.AgentID, Path: request.Path, Content: "无法创建目标目录", Status: 1})
			return
		}

		ffmpegPath, ffmpegErr := videoTrimLookPathFn("ffmpeg")
		ffprobePath, ffprobeErr := videoTrimLookPathFn("ffprobe")
		if ffmpegErr != nil || ffprobeErr != nil {
			writeVideoTrimResp(w, http.StatusServiceUnavailable, VideoTrimResponse{AgentID: request.AgentID, Path: request.Path, Content: "FFmpeg 或 FFprobe 未安装，无法生成新视频", Status: 1})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), videoTrimConversionTimeout)
		defer cancel()
		bitrate := videoTrimProbeBitrate(ctx, ffprobePath, sourcePath)
		temporaryPath := strings.TrimSuffix(outputPath, ".mp4") + "." + strconv.FormatInt(time.Now().UnixNano(), 10) + ".tmp.mp4"
		if !ensureWritablePathWithinRoot(workspace, temporaryPath) {
			writeVideoTrimResp(w, http.StatusBadRequest, VideoTrimResponse{AgentID: request.AgentID, Path: request.Path, Content: "无法在工作目录中创建临时视频", Status: 1})
			return
		}
		defer os.Remove(temporaryPath)
		argsForEncoder := func(encoder ffmpegVideoEncoder) []string {
			args := []string{
				"-hide_banner", "-loglevel", "error", "-nostdin",
				"-ss", formatVideoTrimSeconds(request.Start),
				"-i", sourcePath,
				"-t", formatVideoTrimSeconds(request.End - request.Start),
			}
			if crop != nil {
				args = append(args, "-filter_complex", videoTrimCropFilter(crop), "-map", "[trimCropOutput]")
			} else {
				args = append(args, "-map", "0:v:0?")
			}
			args = append(args, "-map", "0:a?", "-c:v", encoder.Name, "-c:a", "aac")
			if bitrate > 0 {
				args = append(args, "-b:v", strconv.FormatInt(bitrate, 10))
			}
			return append(args, "-movflags", "+faststart", "-f", "mp4", "-n", temporaryPath)
		}
		output, err := runFFmpegVideoEncode(ctx, "video trim", ffmpegPath, temporaryPath, ffmpegH264EncoderCandidates(ffmpegPath), argsForEncoder)
		if err != nil {
			message := "视频转码失败：" + videoTrimCommandError(err, output)
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				message = "视频转码超时"
			}
			writeVideoTrimResp(w, http.StatusInternalServerError, VideoTrimResponse{AgentID: request.AgentID, Path: request.Path, Content: message, Status: 1})
			return
		}
		info, err := os.Stat(temporaryPath)
		if err != nil || info.Size() == 0 {
			writeVideoTrimResp(w, http.StatusInternalServerError, VideoTrimResponse{AgentID: request.AgentID, Path: request.Path, Content: "视频转码未生成有效输出", Status: 1})
			return
		}
		if err := os.Rename(temporaryPath, outputPath); err != nil {
			writeVideoTrimResp(w, http.StatusInternalServerError, VideoTrimResponse{AgentID: request.AgentID, Path: request.Path, Content: "保存新视频失败", Status: 1})
			return
		}
		writeVideoTrimResp(w, http.StatusOK, VideoTrimResponse{AgentID: request.AgentID, Path: request.Path, SavedAs: outputPath, Status: 0})
	}
}

// handleVideoAudioExtractToAudio is used by the original-track control in
// video audio editing and persists the MP3 under audios/ for later reuse.
func handleVideoAudioExtractToAudio(cfg *Config) http.HandlerFunc {
	return handleVideoAudioExtractToDirectory(cfg, "audios")
}

// handleVideoAudioExtractToDirectory converts only the first source audio
// stream and writes it beneath a server-selected workspace directory.
func handleVideoAudioExtractToDirectory(cfg *Config, outputDirectory string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeVideoAudioExtractResp(w, http.StatusMethodNotAllowed, VideoAudioExtractResponse{Content: "method not allowed", Status: 1})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		defer r.Body.Close()
		var request VideoAudioExtractRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeVideoAudioExtractResp(w, http.StatusBadRequest, VideoAudioExtractResponse{Content: "请求体格式错误", Status: 1})
			return
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			writeVideoAudioExtractResp(w, http.StatusBadRequest, VideoAudioExtractResponse{AgentID: request.AgentID, Path: request.Path, Content: "请求体只能包含一个 JSON 对象", Status: 1})
			return
		}
		request.AgentID = strings.TrimSpace(request.AgentID)
		request.Path = normalizeQuotedPathArg(request.Path)
		if !isValidMediaPreviewAgentID(request.AgentID) || request.Path == "" {
			writeVideoAudioExtractResp(w, http.StatusBadRequest, VideoAudioExtractResponse{AgentID: request.AgentID, Path: request.Path, Content: "agentId 和相对路径不能为空", Status: 1})
			return
		}

		sourcePath, _, sourceStatus, sourceMessage := resolveMediaPreviewFile(cfg, request.AgentID, request.Path)
		if sourceStatus != 0 {
			writeVideoAudioExtractResp(w, sourceStatus, VideoAudioExtractResponse{AgentID: request.AgentID, Path: request.Path, Content: sourceMessage, Status: 1})
			return
		}
		mimeType, _ := mediaPreviewMIMEType(sourcePath)
		if !strings.HasPrefix(mimeType, "video/") {
			writeVideoAudioExtractResp(w, http.StatusBadRequest, VideoAudioExtractResponse{AgentID: request.AgentID, Path: request.Path, Content: "仅支持从视频文件提取音频", Status: 1})
			return
		}
		workspace, err := getWorkspaceByAgentID(cfg, request.AgentID)
		if err != nil {
			writeVideoAudioExtractResp(w, http.StatusNotFound, VideoAudioExtractResponse{AgentID: request.AgentID, Path: request.Path, Content: "agent not found", Status: 1})
			return
		}
		sourceRelativePath, err := filepath.Rel(workspace, sourcePath)
		if err != nil || sourceRelativePath == "." || strings.HasPrefix(sourceRelativePath, ".."+string(filepath.Separator)) {
			writeVideoAudioExtractResp(w, http.StatusBadRequest, VideoAudioExtractResponse{AgentID: request.AgentID, Path: request.Path, Content: "path must be a workspace-relative video path", Status: 1})
			return
		}

		ffmpegPath, ffmpegErr := videoTrimLookPathFn("ffmpeg")
		ffprobePath, ffprobeErr := videoTrimLookPathFn("ffprobe")
		if ffmpegErr != nil || ffprobeErr != nil {
			writeVideoAudioExtractResp(w, http.StatusServiceUnavailable, VideoAudioExtractResponse{AgentID: request.AgentID, Path: request.Path, Content: "FFmpeg 或 FFprobe 未安装，无法提取音频", Status: 1})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), videoTrimConversionTimeout)
		defer cancel()
		hasAudio, err := videoAudioExtractHasAudioStream(ctx, ffprobePath, sourcePath)
		if err != nil {
			writeVideoAudioExtractResp(w, http.StatusInternalServerError, VideoAudioExtractResponse{AgentID: request.AgentID, Path: request.Path, Content: "无法检测视频音轨", Status: 1})
			return
		}
		if !hasAudio {
			writeVideoAudioExtractResp(w, http.StatusBadRequest, VideoAudioExtractResponse{AgentID: request.AgentID, Path: request.Path, Content: "视频没有可提取的音轨", Status: 1})
			return
		}

		outputName := videoAudioExtractOutputName(sourceRelativePath)
		outputPath, err := videoAudioExtractAvailableOutputPath(workspace, outputDirectory, outputName, time.Now())
		if err != nil {
			writeVideoAudioExtractResp(w, http.StatusInternalServerError, VideoAudioExtractResponse{AgentID: request.AgentID, Path: request.Path, Content: err.Error(), Status: 1})
			return
		}
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil || !ensureWritablePathWithinRoot(workspace, outputPath) {
			writeVideoAudioExtractResp(w, http.StatusInternalServerError, VideoAudioExtractResponse{AgentID: request.AgentID, Path: request.Path, Content: "无法创建 " + outputDirectory + " 目录", Status: 1})
			return
		}
		temporaryPath := strings.TrimSuffix(outputPath, ".mp3") + "." + strconv.FormatInt(time.Now().UnixNano(), 10) + ".tmp.mp3"
		if !ensureWritablePathWithinRoot(workspace, temporaryPath) {
			writeVideoAudioExtractResp(w, http.StatusBadRequest, VideoAudioExtractResponse{AgentID: request.AgentID, Path: request.Path, Content: "无法在工作目录中创建临时 MP3", Status: 1})
			return
		}
		defer os.Remove(temporaryPath)
		args := []string{
			"-hide_banner", "-loglevel", "error", "-nostdin",
			"-i", sourcePath,
			"-map", "0:a:0", "-vn",
			"-c:a", "libmp3lame", "-q:a", "2",
			"-f", "mp3", "-n", temporaryPath,
		}
		output, err := videoTrimCommandContextFn(ctx, ffmpegPath, args...).CombinedOutput()
		if err != nil {
			message := "音频提取失败：" + videoTrimCommandError(err, output)
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				message = "音频提取超时"
			}
			writeVideoAudioExtractResp(w, http.StatusInternalServerError, VideoAudioExtractResponse{AgentID: request.AgentID, Path: request.Path, Content: message, Status: 1})
			return
		}
		info, err := os.Stat(temporaryPath)
		if err != nil || info.Size() == 0 {
			writeVideoAudioExtractResp(w, http.StatusInternalServerError, VideoAudioExtractResponse{AgentID: request.AgentID, Path: request.Path, Content: "音频提取未生成有效 MP3", Status: 1})
			return
		}
		var linkErr error
		for attempt := 0; attempt < 64; attempt++ {
			linkErr = os.Link(temporaryPath, outputPath)
			if linkErr == nil {
				break
			}
			if !errors.Is(linkErr, fs.ErrExist) {
				writeVideoAudioExtractResp(w, http.StatusInternalServerError, VideoAudioExtractResponse{AgentID: request.AgentID, Path: request.Path, Content: "保存提取的音频失败", Status: 1})
				return
			}
			outputPath, err = videoAudioExtractAvailableOutputPath(workspace, outputDirectory, outputName, time.Now().Add(time.Duration(attempt+1)*time.Nanosecond))
			if err != nil {
				writeVideoAudioExtractResp(w, http.StatusInternalServerError, VideoAudioExtractResponse{AgentID: request.AgentID, Path: request.Path, Content: err.Error(), Status: 1})
				return
			}
		}
		if linkErr != nil {
			writeVideoAudioExtractResp(w, http.StatusInternalServerError, VideoAudioExtractResponse{AgentID: request.AgentID, Path: request.Path, Content: "无法生成不重复的 MP3 文件名", Status: 1})
			return
		}
		if err := os.Remove(temporaryPath); err != nil {
			_ = os.Remove(outputPath)
			writeVideoAudioExtractResp(w, http.StatusInternalServerError, VideoAudioExtractResponse{AgentID: request.AgentID, Path: request.Path, Content: "清理临时 MP3 失败", Status: 1})
			return
		}
		writeVideoAudioExtractResp(w, http.StatusOK, VideoAudioExtractResponse{AgentID: request.AgentID, Path: request.Path, SavedAs: outputPath, Status: 0})
	}
}

func videoAudioEditProbeVideoDuration(ctx context.Context, ffprobePath, sourcePath string) (float64, error) {
	output, err := videoTrimCommandContextFn(ctx, ffprobePath,
		"-v", "error", "-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1", sourcePath,
	).Output()
	if err != nil {
		return 0, errors.New("无法读取视频时长")
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil || !audioMixFiniteNonNegative(duration) || duration <= 0 || duration > audioMixMaxTimelineSeconds {
		return 0, errors.New("无法读取视频时长")
	}
	return duration, nil
}

func videoAudioEditTrackFilter(inputIndex, index int, track AudioMixTrack, duration float64, copies int) (string, error) {
	label := "vma" + strconv.Itoa(index)
	filters := []string{
		"[" + strconv.Itoa(inputIndex) + ":a]atrim=start=" + audioMixFormatFloat(track.TrimStart) + ":end=" + audioMixFormatFloat(track.TrimEnd),
		"asetpts=PTS-STARTPTS",
	}
	filters = append(filters, audioMixTempoFilters(track.Speed)...)
	if track.Loop && copies > 1 {
		outputs := make([]string, 0, copies)
		for copyIndex := 0; copyIndex < copies; copyIndex++ {
			outputs = append(outputs, "["+label+"_copy"+strconv.Itoa(copyIndex)+"]")
		}
		filters = append(filters, "asplit="+strconv.Itoa(copies)+strings.Join(outputs, ""))
		filters = append(filters, strings.Join(outputs, "")+"concat=n="+strconv.Itoa(copies)+":v=0:a=1")
	}
	filters = append(filters, "atrim=duration="+audioMixFormatFloat(duration), "aformat=channel_layouts=stereo")
	filters = append(filters,
		"volume="+audioMixFormatFloat(track.Volume),
		"pan=stereo|c0="+audioMixFormatFloat(track.LeftVolume)+"*c0|c1="+audioMixFormatFloat(track.RightVolume)+"*c1",
	)
	if track.FadeIn > 0 {
		filters = append(filters, "afade=t=in:st=0:d="+audioMixFormatFloat(track.FadeIn))
	}
	if track.FadeOut > 0 {
		filters = append(filters, "afade=t=out:st="+audioMixFormatFloat(math.Max(0, duration-track.FadeOut))+":d="+audioMixFormatFloat(track.FadeOut))
	}
	for _, effect := range track.Effects {
		filter, err := audioMixEffectFilter(effect)
		if err != nil {
			return "", err
		}
		filters = append(filters, filter)
	}
	filters = append(filters, "adelay="+strconv.FormatInt(int64(math.Round(track.Start*1000)), 10)+":all=1")
	return strings.Join(filters, ",") + "[" + label + "]", nil
}

// handleVideoAudioEdit keeps source pixels and replaces the soundtrack with a
// constrained multitrack mix. It accepts no client-supplied FFmpeg syntax or
// absolute paths and never overwrites an existing export.
func handleVideoAudioEdit(cfg *Config) http.HandlerFunc {
	return handleVideoAudioEditWithMode(cfg, false)
}

func handleVideoAudioEditPreview(cfg *Config) http.HandlerFunc {
	return handleVideoAudioEditWithMode(cfg, true)
}

func handleVideoAudioEditWithMode(cfg *Config, preview bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeVideoAudioEditResp(w, http.StatusMethodNotAllowed, VideoAudioEditResponse{Content: "method not allowed", Status: 1})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		defer r.Body.Close()
		var request VideoAudioEditRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeVideoAudioEditResp(w, http.StatusBadRequest, VideoAudioEditResponse{Content: "请求体格式错误", Status: 1})
			return
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			writeVideoAudioEditResp(w, http.StatusBadRequest, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "请求体只能包含一个 JSON 对象", Status: 1})
			return
		}
		request.AgentID = strings.TrimSpace(request.AgentID)
		request.Path = normalizeQuotedPathArg(request.Path)
		request.OutputName = strings.TrimSpace(normalizeQuotedPathArg(request.OutputName))
		if !isValidMediaPreviewAgentID(request.AgentID) || request.Path == "" {
			writeVideoAudioEditResp(w, http.StatusBadRequest, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "agentId 和相对路径不能为空", Status: 1})
			return
		}
		if len(request.Tracks) == 0 || len(request.Tracks) > audioMixMaxTracks {
			writeVideoAudioEditResp(w, http.StatusBadRequest, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "音轨数量必须在 1 到 64 之间", Status: 1})
			return
		}
		if request.OutputName != "" && !isSafeVideoTrimOutputName(request.OutputName) {
			writeVideoAudioEditResp(w, http.StatusBadRequest, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "新视频文件名必须是有效的 .mp4 文件名", Status: 1})
			return
		}

		sourcePath, _, sourceStatus, sourceMessage := resolveMediaPreviewFile(cfg, request.AgentID, request.Path)
		if sourceStatus != 0 {
			writeVideoAudioEditResp(w, sourceStatus, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: sourceMessage, Status: 1})
			return
		}
		mimeType, _ := mediaPreviewMIMEType(sourcePath)
		if !strings.HasPrefix(mimeType, "video/") {
			writeVideoAudioEditResp(w, http.StatusBadRequest, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "仅支持编辑视频音轨", Status: 1})
			return
		}
		workspace, err := getWorkspaceByAgentID(cfg, request.AgentID)
		if err != nil {
			writeVideoAudioEditResp(w, http.StatusNotFound, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "agent not found", Status: 1})
			return
		}
		sourceRelativePath, err := filepath.Rel(workspace, sourcePath)
		if err != nil || sourceRelativePath == "." || strings.HasPrefix(sourceRelativePath, ".."+string(filepath.Separator)) {
			writeVideoAudioEditResp(w, http.StatusBadRequest, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "path must be a workspace-relative video path", Status: 1})
			return
		}

		ffmpegPath, ffmpegErr := videoTrimLookPathFn("ffmpeg")
		ffprobePath, ffprobeErr := videoTrimLookPathFn("ffprobe")
		if ffmpegErr != nil || ffprobeErr != nil {
			writeVideoAudioEditResp(w, http.StatusServiceUnavailable, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "FFmpeg 或 FFprobe 未安装，无法剪辑音频", Status: 1})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), videoTrimConversionTimeout)
		defer cancel()
		videoDuration, err := videoAudioEditProbeVideoDuration(ctx, ffprobePath, sourcePath)
		if err != nil {
			writeVideoAudioEditResp(w, http.StatusBadRequest, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: err.Error(), Status: 1})
			return
		}

		inputArgs := []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-i", sourcePath}
		inputCount := 1
		filterParts := make([]string, 0, len(request.Tracks)+1)
		labels := make([]string, 0, len(request.Tracks))
		hasOriginal := false
		for index, item := range request.Tracks {
			track := item.AudioMixTrack
			inputIndex := 0
			sourceDuration := 0.0
			if item.Original {
				if hasOriginal || strings.TrimSpace(track.Path) != "" {
					writeVideoAudioEditResp(w, http.StatusBadRequest, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "原视频音轨只能添加一次且不能指定路径", Status: 1})
					return
				}
				hasOriginal = true
				hasAudio, probeErr := videoAudioExtractHasAudioStream(ctx, ffprobePath, sourcePath)
				if probeErr != nil {
					writeVideoAudioEditResp(w, http.StatusBadRequest, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "无法检测视频原始音轨", Status: 1})
					return
				}
				if !hasAudio {
					continue
				}
				sourceDuration = videoDuration
			} else {
				resolvedPath, resolveErr := audioMixResolveSource(workspace, track.Path)
				if resolveErr != nil {
					writeVideoAudioEditResp(w, http.StatusBadRequest, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "第 " + strconv.Itoa(index+1) + " 段音轨无效：" + resolveErr.Error(), Status: 1})
					return
				}
				probeDuration, probeErr := audioMixProbeSource(ctx, ffprobePath, resolvedPath)
				if probeErr != nil {
					writeVideoAudioEditResp(w, http.StatusBadRequest, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "第 " + strconv.Itoa(index+1) + " 段音轨无效：" + probeErr.Error(), Status: 1})
					return
				}
				sourceDuration = probeDuration
				inputIndex = inputCount
				inputArgs = append(inputArgs, "-i", resolvedPath)
				inputCount++
			}
			if track.TrimEnd == 0 {
				track.TrimEnd = math.Min(sourceDuration, math.Max(0, videoDuration-track.Start))
			}
			if track.TrimEnd > sourceDuration+0.001 {
				writeVideoAudioEditResp(w, http.StatusBadRequest, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "第 " + strconv.Itoa(index+1) + " 段音轨无效：裁剪范围超出源文件时长", Status: 1})
				return
			}
			duration, copies, durationErr := audioMixTrackDuration(track)
			if durationErr != nil || track.Start+duration > videoDuration+0.001 {
				message := "时间轴范围超出视频时长"
				if durationErr != nil {
					message = durationErr.Error()
				}
				writeVideoAudioEditResp(w, http.StatusBadRequest, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "第 " + strconv.Itoa(index+1) + " 段音轨无效：" + message, Status: 1})
				return
			}
			filter, filterErr := videoAudioEditTrackFilter(inputIndex, len(labels), track, duration, copies)
			if filterErr != nil {
				writeVideoAudioEditResp(w, http.StatusBadRequest, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "第 " + strconv.Itoa(index+1) + " 段音轨无效：" + filterErr.Error(), Status: 1})
				return
			}
			filterParts = append(filterParts, filter)
			labels = append(labels, "[vma"+strconv.Itoa(len(labels))+"]")
		}
		if len(labels) == 0 {
			writeVideoAudioEditResp(w, http.StatusBadRequest, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "视频没有可用音轨，请添加音频后再保存", Status: 1})
			return
		}
		filterParts = append(filterParts, strings.Join(labels, "")+"amix=inputs="+strconv.Itoa(len(labels))+":duration=longest:dropout_transition=0,alimiter=limit=0.99,apad,atrim=duration="+audioMixFormatFloat(videoDuration)+",aformat=channel_layouts=stereo[videoAudioEditMix]")
		if preview {
			temporaryFile, tempErr := os.CreateTemp("", "deepright-video-audio-preview-*.wav")
			if tempErr != nil {
				writeVideoAudioEditResp(w, http.StatusInternalServerError, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "无法创建试听临时音频", Status: 1})
				return
			}
			temporaryPath := temporaryFile.Name()
			if err := temporaryFile.Close(); err != nil {
				_ = os.Remove(temporaryPath)
				writeVideoAudioEditResp(w, http.StatusInternalServerError, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "无法创建试听临时音频", Status: 1})
				return
			}
			defer os.Remove(temporaryPath)
			args := append(inputArgs, "-filter_complex", strings.Join(filterParts, ";"), "-map", "[videoAudioEditMix]", "-ar", "48000", "-ac", "2", "-c:a", "pcm_s16le", "-f", "wav", "-y", temporaryPath)
			commandOutput, commandErr := videoTrimCommandContextFn(ctx, ffmpegPath, args...).CombinedOutput()
			if commandErr != nil {
				message := "生成试听失败：" + videoTrimCommandError(commandErr, commandOutput)
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					message = "生成试听超时"
				}
				writeVideoAudioEditResp(w, http.StatusInternalServerError, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: message, Status: 1})
				return
			}
			info, statErr := os.Stat(temporaryPath)
			if statErr != nil || info.Size() == 0 {
				writeVideoAudioEditResp(w, http.StatusInternalServerError, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "试听未生成有效音频", Status: 1})
				return
			}
			file, openErr := os.Open(temporaryPath)
			if openErr != nil {
				writeVideoAudioEditResp(w, http.StatusInternalServerError, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "无法读取试听音频", Status: 1})
				return
			}
			defer file.Close()
			w.Header().Set("Content-Type", "audio/wav")
			w.Header().Set("Content-Disposition", "inline")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			http.ServeContent(w, r, "video-audio-preview.wav", info.ModTime(), file)
			return
		}

		outputName := request.OutputName
		if outputName == "" {
			outputName = videoAudioEditOutputName(sourceRelativePath)
		}
		outputPath, renamed, err := videoAudioEditAvailableOutputPath(workspace, outputName, time.Now())
		if err != nil {
			writeVideoAudioEditResp(w, http.StatusInternalServerError, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: err.Error(), Status: 1})
			return
		}
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil || !ensureWritablePathWithinRoot(workspace, outputPath) {
			writeVideoAudioEditResp(w, http.StatusInternalServerError, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "无法创建 videos 目录", Status: 1})
			return
		}
		temporaryPath := strings.TrimSuffix(outputPath, ".mp4") + "." + strconv.FormatInt(time.Now().UnixNano(), 10) + ".tmp.mp4"
		if !ensureWritablePathWithinRoot(workspace, temporaryPath) {
			writeVideoAudioEditResp(w, http.StatusBadRequest, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "无法创建临时 MP4", Status: 1})
			return
		}
		defer os.Remove(temporaryPath)
		args := append(inputArgs, "-filter_complex", strings.Join(filterParts, ";"), "-map", "0:v:0?", "-map", "[videoAudioEditMix]", "-c:v", "copy", "-c:a", "aac", "-movflags", "+faststart", "-f", "mp4", "-n", temporaryPath)
		commandOutput, commandErr := videoTrimCommandContextFn(ctx, ffmpegPath, args...).CombinedOutput()
		if commandErr != nil {
			message := "视频音频合并失败：" + videoTrimCommandError(commandErr, commandOutput)
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				message = "视频音频合并超时"
			}
			writeVideoAudioEditResp(w, http.StatusInternalServerError, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: message, Status: 1})
			return
		}
		info, statErr := os.Stat(temporaryPath)
		if statErr != nil || info.Size() == 0 {
			writeVideoAudioEditResp(w, http.StatusInternalServerError, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "视频音频合并未生成有效 MP4", Status: 1})
			return
		}
		for attempt := 0; attempt < 64; attempt++ {
			if linkErr := os.Link(temporaryPath, outputPath); linkErr == nil {
				if err := os.Remove(temporaryPath); err != nil {
					_ = os.Remove(outputPath)
					writeVideoAudioEditResp(w, http.StatusInternalServerError, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "清理临时 MP4 失败", Status: 1})
					return
				}
				flushAgentCache()
				writeVideoAudioEditResp(w, http.StatusOK, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, SavedAs: outputPath, Renamed: renamed, Status: 0})
				return
			} else if !errors.Is(linkErr, fs.ErrExist) {
				writeVideoAudioEditResp(w, http.StatusInternalServerError, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "保存 MP4 失败", Status: 1})
				return
			}
			outputPath, renamed, err = videoAudioEditAvailableOutputPath(workspace, outputName, time.Now().Add(time.Duration(attempt+1)*time.Nanosecond))
			if err != nil {
				writeVideoAudioEditResp(w, http.StatusInternalServerError, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: err.Error(), Status: 1})
				return
			}
		}
		writeVideoAudioEditResp(w, http.StatusInternalServerError, VideoAudioEditResponse{AgentID: request.AgentID, Path: request.Path, Content: "无法生成不重复的 MP4 文件名", Status: 1})
	}
}

// AudioSaveRequest carries a browser-recorded WAV as standard Base64. Audio
// recording deliberately has its own narrow endpoint instead of broadening
// the generic editor write contract.
type AudioSaveRequest struct {
	Content string `json:"content"`
}

type AudioSaveResponse struct {
	AgentID string `json:"agentId"`
	Path    string `json:"path"`
	SavedAs string `json:"savedAs,omitempty"`
	Content string `json:"content,omitempty"`
	Status  int    `json:"status"`
}

func writeAudioSaveResp(w http.ResponseWriter, httpStatus int, resp AudioSaveResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(resp)
}

// isValidPCMContainerWAV validates the RIFF envelope rather than trusting a
// .wav filename. The browser can choose the sample rate and channel count,
// while this endpoint deliberately accepts only 16-bit PCM output.
func isValidPCMContainerWAV(data []byte) bool {
	if len(data) < 44 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return false
	}
	riffEnd := uint64(binary.LittleEndian.Uint32(data[4:8])) + 8
	if riffEnd > uint64(len(data)) || riffEnd < 12 {
		return false
	}
	var hasFormat, hasData bool
	for offset := uint64(12); offset < riffEnd; {
		if riffEnd-offset < 8 || offset+8 > uint64(len(data)) {
			return false
		}
		chunkID := string(data[offset : offset+4])
		chunkSize := uint64(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		chunkDataStart := offset + 8
		chunkEnd := chunkDataStart + chunkSize
		if chunkEnd < chunkDataStart || chunkEnd > riffEnd || chunkEnd > uint64(len(data)) {
			return false
		}
		switch chunkID {
		case "fmt ":
			if chunkSize < 16 {
				return false
			}
			format := binary.LittleEndian.Uint16(data[chunkDataStart : chunkDataStart+2])
			channels := binary.LittleEndian.Uint16(data[chunkDataStart+2 : chunkDataStart+4])
			sampleRate := binary.LittleEndian.Uint32(data[chunkDataStart+4 : chunkDataStart+8])
			bitsPerSample := binary.LittleEndian.Uint16(data[chunkDataStart+14 : chunkDataStart+16])
			if format != 1 || channels == 0 || sampleRate == 0 || bitsPerSample != 16 {
				return false
			}
			hasFormat = true
		case "data":
			if chunkSize == 0 {
				return false
			}
			hasData = true
		}
		offset = chunkEnd
		if chunkSize%2 != 0 {
			offset++ // RIFF chunks are word-aligned.
			if offset > riffEnd {
				return false
			}
		}
	}
	return hasFormat && hasData
}

func isAgentAudioRelativePath(relPath string) bool {
	pathValue := filepath.ToSlash(filepath.Clean(relPath))
	return path.Dir(pathValue) == "audios" && path.Base(pathValue) != "."
}

func handleAgentAudioSave(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeAudioSaveResp(w, http.StatusMethodNotAllowed, AudioSaveResponse{Content: "method not allowed", Status: 1})
			return
		}
		agentID := strings.TrimSpace(r.URL.Query().Get("agentId"))
		relPath := normalizeQuotedPathArg(r.URL.Query().Get("path"))
		if agentID == "" || relPath == "" {
			writeAudioSaveResp(w, http.StatusBadRequest, AudioSaveResponse{AgentID: agentID, Path: relPath, Content: "agentId 和相对路径不能为空", Status: 1})
			return
		}
		if filepath.IsAbs(relPath) || strings.HasPrefix(relPath, "~") {
			writeAudioSaveResp(w, http.StatusBadRequest, AudioSaveResponse{AgentID: agentID, Path: relPath, Content: "只支持 workspace 下的相对路径", Status: 1})
			return
		}
		cleanRelPath := filepath.Clean(relPath)
		if cleanRelPath == "." || cleanRelPath == "" || cleanRelPath == ".." || strings.HasPrefix(cleanRelPath, ".."+string(filepath.Separator)) {
			writeAudioSaveResp(w, http.StatusBadRequest, AudioSaveResponse{AgentID: agentID, Path: relPath, Content: "只支持 workspace 下的相对路径", Status: 1})
			return
		}
		if !strings.EqualFold(filepath.Ext(cleanRelPath), ".wav") {
			writeAudioSaveResp(w, http.StatusBadRequest, AudioSaveResponse{AgentID: agentID, Path: relPath, Content: "录音仅支持 .wav 后缀", Status: 1})
			return
		}
		if !isAgentAudioRelativePath(cleanRelPath) {
			writeAudioSaveResp(w, http.StatusBadRequest, AudioSaveResponse{AgentID: agentID, Path: relPath, Content: "录音仅可保存到 audios 目录", Status: 1})
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 256<<20)
		defer r.Body.Close()
		var request AudioSaveRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeAudioSaveResp(w, http.StatusBadRequest, AudioSaveResponse{AgentID: agentID, Path: relPath, Content: "请求体格式错误", Status: 1})
			return
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			writeAudioSaveResp(w, http.StatusBadRequest, AudioSaveResponse{AgentID: agentID, Path: relPath, Content: "请求体只能包含一个 JSON 对象", Status: 1})
			return
		}
		wavData, err := base64.StdEncoding.DecodeString(request.Content)
		if err != nil {
			writeAudioSaveResp(w, http.StatusBadRequest, AudioSaveResponse{AgentID: agentID, Path: relPath, Content: "WAV 内容不是合法的 base64", Status: 1})
			return
		}
		if !isValidPCMContainerWAV(wavData) {
			writeAudioSaveResp(w, http.StatusBadRequest, AudioSaveResponse{AgentID: agentID, Path: relPath, Content: "录音内容不是有效的 WAV 文件", Status: 1})
			return
		}

		workspace, err := getWorkspaceByAgentID(cfg, agentID)
		if err != nil {
			writeAudioSaveResp(w, http.StatusNotFound, AudioSaveResponse{AgentID: agentID, Path: relPath, Content: "agent not found", Status: 1})
			return
		}
		targetPath, err := resolveCaseInsensitiveUnderRoot(workspace, cleanRelPath)
		if err != nil || !ensureWritablePathWithinRoot(workspace, targetPath) {
			writeAudioSaveResp(w, http.StatusBadRequest, AudioSaveResponse{AgentID: agentID, Path: relPath, Content: "无法在工作目录中保存录音", Status: 1})
			return
		}
		if _, err := os.Lstat(targetPath); err == nil {
			writeAudioSaveResp(w, http.StatusConflict, AudioSaveResponse{AgentID: agentID, Path: relPath, Content: "同名文件已存在，请更换名称", Status: 1})
			return
		} else if !errors.Is(err, os.ErrNotExist) {
			writeAudioSaveResp(w, http.StatusInternalServerError, AudioSaveResponse{AgentID: agentID, Path: relPath, Content: "无法检查录音目标", Status: 1})
			return
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil || !ensureWritablePathWithinRoot(workspace, targetPath) {
			writeAudioSaveResp(w, http.StatusInternalServerError, AudioSaveResponse{AgentID: agentID, Path: relPath, Content: "无法创建录音目录", Status: 1})
			return
		}
		temporary, err := os.CreateTemp(filepath.Dir(targetPath), "."+filepath.Base(targetPath)+".tmp-*")
		if err != nil {
			writeAudioSaveResp(w, http.StatusInternalServerError, AudioSaveResponse{AgentID: agentID, Path: relPath, Content: "无法创建录音临时文件", Status: 1})
			return
		}
		temporaryPath := temporary.Name()
		defer os.Remove(temporaryPath)
		if _, err := temporary.Write(wavData); err != nil || temporary.Sync() != nil || temporary.Close() != nil {
			_ = temporary.Close()
			writeAudioSaveResp(w, http.StatusInternalServerError, AudioSaveResponse{AgentID: agentID, Path: relPath, Content: "保存录音失败", Status: 1})
			return
		}
		// Linking is an atomic create: unlike Rename it can never replace a
		// file that appeared between the conflict check and this write.
		if err := os.Link(temporaryPath, targetPath); err != nil {
			status := http.StatusInternalServerError
			message := "保存录音失败"
			if errors.Is(err, fs.ErrExist) {
				status = http.StatusConflict
				message = "同名文件已存在，请更换名称"
			}
			writeAudioSaveResp(w, status, AudioSaveResponse{AgentID: agentID, Path: relPath, Content: message, Status: 1})
			return
		}
		if err := os.Remove(temporaryPath); err != nil {
			log.Printf("remove recorded-audio temporary file %s: %v", temporaryPath, err)
		}
		flushAgentCache()
		writeAudioSaveResp(w, http.StatusOK, AudioSaveResponse{AgentID: agentID, Path: filepath.ToSlash(cleanRelPath), SavedAs: targetPath, Status: 0})
	}
}

// AudioMixRequest is deliberately a small, declarative audio project. It is
// not a transport for an FFmpeg command line: every value is independently
// validated before the server constructs its fixed filter graph.
type AudioMixRequest struct {
	AgentID    string          `json:"agentId"`
	OutputName string          `json:"outputName"`
	SampleRate int             `json:"sampleRate"`
	BitDepth   int             `json:"bitDepth"`
	Tracks     []AudioMixTrack `json:"tracks"`
}

type AudioMixTrack struct {
	Path        string           `json:"path"`
	TrimStart   float64          `json:"trimStart"`
	TrimEnd     float64          `json:"trimEnd"`
	Start       float64          `json:"start"`
	Speed       float64          `json:"speed"`
	Loop        bool             `json:"loop"`
	Duration    float64          `json:"duration"`
	Volume      float64          `json:"volume"`
	LeftVolume  float64          `json:"leftVolume"`
	RightVolume float64          `json:"rightVolume"`
	FadeIn      float64          `json:"fadeIn"`
	FadeOut     float64          `json:"fadeOut"`
	Effects     []AudioMixEffect `json:"effects"`
}

type AudioMixEffect struct {
	Type   string  `json:"type"`
	Amount float64 `json:"amount"`
}

type AudioMixResponse struct {
	AgentID string `json:"agentId"`
	Path    string `json:"path,omitempty"`
	SavedAs string `json:"savedAs,omitempty"`
	Content string `json:"content,omitempty"`
	Status  int    `json:"status"`
}

const (
	audioMixMaxTracks          = 64
	audioMixMaxEffects         = 16
	audioMixMaxTimelineSeconds = 2 * 60 * 60
	audioMixMaxLoopCopies      = 128
)

func writeAudioMixResp(w http.ResponseWriter, httpStatus int, resp AudioMixResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(resp)
}

func isSafeAudioMixOutputName(name string) bool {
	return isSafeSaveAsNewFilename(name) && strings.EqualFold(filepath.Ext(name), ".wav") && len(strings.TrimSuffix(name, filepath.Ext(name))) > 0
}

func audioMixFiniteNonNegative(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}

func audioMixFormatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func audioMixTempoFilters(speed float64) []string {
	filters := make([]string, 0, 3)
	for speed > 2 {
		filters = append(filters, "atempo=2")
		speed /= 2
	}
	for speed < 0.5 {
		filters = append(filters, "atempo=0.5")
		speed /= 0.5
	}
	return append(filters, "atempo="+audioMixFormatFloat(speed))
}

func audioMixEffectFilter(effect AudioMixEffect) (string, error) {
	typ := strings.ToLower(strings.TrimSpace(effect.Type))
	if math.IsNaN(effect.Amount) || math.IsInf(effect.Amount, 0) {
		return "", errors.New("效果器参数必须是有限数字")
	}
	switch typ {
	case "equalizer":
		if effect.Amount < -24 || effect.Amount > 24 {
			return "", errors.New("均衡器范围必须在 -24 到 24 dB")
		}
		return "equalizer=f=1000:t=q:w=1:g=" + audioMixFormatFloat(effect.Amount), nil
	case "compressor":
		if effect.Amount < 0 || effect.Amount > 1 {
			return "", errors.New("压缩器强度必须在 0 到 1")
		}
		ratio := 1 + effect.Amount*19
		return "acompressor=threshold=0.125:ratio=" + audioMixFormatFloat(ratio) + ":attack=20:release=250", nil
	case "reverb":
		if effect.Amount < 0 || effect.Amount > 1 {
			return "", errors.New("混响强度必须在 0 到 1")
		}
		return "aecho=0.8:0.9:1000:" + audioMixFormatFloat(effect.Amount*0.5), nil
	case "delay":
		if effect.Amount < 0 || effect.Amount > 1 {
			return "", errors.New("延迟强度必须在 0 到 1")
		}
		return "aecho=0.8:0.88:400:" + audioMixFormatFloat(effect.Amount*0.55), nil
	default:
		return "", errors.New("不支持的效果器")
	}
}

func audioMixTrackDuration(track AudioMixTrack) (float64, int, error) {
	if !audioMixFiniteNonNegative(track.TrimStart) || !audioMixFiniteNonNegative(track.TrimEnd) || !audioMixFiniteNonNegative(track.Start) || !audioMixFiniteNonNegative(track.FadeIn) || !audioMixFiniteNonNegative(track.FadeOut) {
		return 0, 0, errors.New("片段时间必须是非负有限数字")
	}
	if track.TrimEnd <= track.TrimStart {
		return 0, 0, errors.New("裁剪结束时间必须大于开始时间")
	}
	if math.IsNaN(track.Speed) || math.IsInf(track.Speed, 0) || track.Speed < 0.25 || track.Speed > 4 {
		return 0, 0, errors.New("播放速率必须在 0.25× 到 4×")
	}
	for _, value := range []float64{track.Volume, track.LeftVolume, track.RightVolume} {
		if !audioMixFiniteNonNegative(value) || value > 4 {
			return 0, 0, errors.New("音量必须在 0 到 4")
		}
	}
	baseDuration := (track.TrimEnd - track.TrimStart) / track.Speed
	duration := baseDuration
	copies := 1
	if track.Loop {
		if !audioMixFiniteNonNegative(track.Duration) || track.Duration <= 0 || track.Duration > audioMixMaxTimelineSeconds {
			return 0, 0, errors.New("循环片段总时长必须在有效范围内")
		}
		duration = track.Duration
		copies = int(math.Ceil(duration / baseDuration))
		if copies < 1 || copies > audioMixMaxLoopCopies {
			return 0, 0, errors.New("循环次数超出允许范围")
		}
	}
	if duration <= 0 || duration > audioMixMaxTimelineSeconds || track.Start+duration > audioMixMaxTimelineSeconds {
		return 0, 0, errors.New("片段时间轴范围超出允许范围")
	}
	if track.FadeIn+track.FadeOut > duration+1e-9 {
		return 0, 0, errors.New("淡入淡出时长不能超过片段时长")
	}
	if len(track.Effects) > audioMixMaxEffects {
		return 0, 0, errors.New("单个片段的效果器数量过多")
	}
	for _, effect := range track.Effects {
		if _, err := audioMixEffectFilter(effect); err != nil {
			return 0, 0, err
		}
	}
	return duration, copies, nil
}

func audioMixResolveSource(workspace, relativePath string) (string, error) {
	relativePath = normalizeQuotedPathArg(relativePath)
	if relativePath == "" || filepath.IsAbs(relativePath) || strings.HasPrefix(relativePath, "~") {
		return "", errors.New("源文件路径必须是当前工作区相对路径")
	}
	cleanPath := filepath.Clean(relativePath)
	if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return "", errors.New("源文件路径必须是当前工作区相对路径")
	}
	resolved, err := resolveCaseInsensitiveUnderRoot(workspace, cleanPath)
	if err != nil || !ensurePathWithinRoot(workspace, resolved) {
		return "", errors.New("源文件路径无效")
	}
	info, err := os.Lstat(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", errors.New("源文件不存在")
		}
		return "", errors.New("无法读取源文件")
	}
	if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("源文件必须是普通文件")
	}
	return resolved, nil
}

func audioMixProbeSource(ctx context.Context, ffprobePath, sourcePath string) (float64, error) {
	output, err := videoTrimCommandContextFn(ctx, ffprobePath,
		"-v", "error", "-select_streams", "a:0",
		"-show_entries", "stream=codec_type:format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1", sourcePath,
	).Output()
	if err != nil {
		return 0, errors.New("无法解析源文件中的音频流")
	}
	lines := strings.Fields(string(output))
	if len(lines) < 2 || lines[0] != "audio" {
		return 0, errors.New("源文件不含可解析音频流")
	}
	duration, err := strconv.ParseFloat(lines[len(lines)-1], 64)
	if err != nil || !audioMixFiniteNonNegative(duration) || duration <= 0 {
		return 0, errors.New("无法读取源文件时长")
	}
	return duration, nil
}

func audioMixTrackFilter(index int, track AudioMixTrack, duration float64, copies int) (string, error) {
	label := "m" + strconv.Itoa(index)
	filters := []string{
		"[" + strconv.Itoa(index) + ":a]atrim=start=" + audioMixFormatFloat(track.TrimStart) + ":end=" + audioMixFormatFloat(track.TrimEnd),
		"asetpts=PTS-STARTPTS",
	}
	filters = append(filters, audioMixTempoFilters(track.Speed)...)
	if track.Loop && copies > 1 {
		outputs := make([]string, 0, copies)
		for copyIndex := 0; copyIndex < copies; copyIndex++ {
			outputs = append(outputs, "["+label+"_copy"+strconv.Itoa(copyIndex)+"]")
		}
		filters = append(filters, "asplit="+strconv.Itoa(copies)+strings.Join(outputs, ""))
		filters = append(filters, strings.Join(outputs, "")+"concat=n="+strconv.Itoa(copies)+":v=0:a=1")
	}
	filters = append(filters, "atrim=duration="+audioMixFormatFloat(duration), "aformat=channel_layouts=stereo")
	filters = append(filters,
		"volume="+audioMixFormatFloat(track.Volume),
		"pan=stereo|c0="+audioMixFormatFloat(track.LeftVolume)+"*c0|c1="+audioMixFormatFloat(track.RightVolume)+"*c1",
	)
	if track.FadeIn > 0 {
		filters = append(filters, "afade=t=in:st=0:d="+audioMixFormatFloat(track.FadeIn))
	}
	if track.FadeOut > 0 {
		filters = append(filters, "afade=t=out:st="+audioMixFormatFloat(math.Max(0, duration-track.FadeOut))+":d="+audioMixFormatFloat(track.FadeOut))
	}
	for _, effect := range track.Effects {
		filter, err := audioMixEffectFilter(effect)
		if err != nil {
			return "", err
		}
		filters = append(filters, filter)
	}
	filters = append(filters, "adelay="+strconv.FormatInt(int64(math.Round(track.Start*1000)), 10)+":all=1")
	return strings.Join(filters, ",") + "[" + label + "]", nil
}

func handleAudioMix(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeAudioMixResp(w, http.StatusMethodNotAllowed, AudioMixResponse{Content: "method not allowed", Status: 1})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		defer r.Body.Close()
		var request AudioMixRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{Content: "请求体格式错误", Status: 1})
			return
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{AgentID: request.AgentID, Content: "请求体只能包含一个 JSON 对象", Status: 1})
			return
		}
		request.AgentID = strings.TrimSpace(request.AgentID)
		request.OutputName = strings.TrimSpace(normalizeQuotedPathArg(request.OutputName))
		if !isValidMediaPreviewAgentID(request.AgentID) || !isSafeAudioMixOutputName(request.OutputName) {
			writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{AgentID: request.AgentID, Content: "agentId 和输出文件名必须有效，且输出必须为 .wav", Status: 1})
			return
		}
		if request.SampleRate != 44100 && request.SampleRate != 48000 {
			writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{AgentID: request.AgentID, Content: "采样率只支持 44100 或 48000", Status: 1})
			return
		}
		if request.BitDepth != 16 && request.BitDepth != 24 {
			writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{AgentID: request.AgentID, Content: "位深只支持 16 或 24", Status: 1})
			return
		}
		if len(request.Tracks) == 0 || len(request.Tracks) > audioMixMaxTracks {
			writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{AgentID: request.AgentID, Content: "音轨数量必须在 1 到 64 之间", Status: 1})
			return
		}
		workspace, err := getWorkspaceByAgentID(cfg, request.AgentID)
		if err != nil {
			writeAudioMixResp(w, http.StatusNotFound, AudioMixResponse{AgentID: request.AgentID, Content: "agent not found", Status: 1})
			return
		}
		outputRelativePath := filepath.Join("audios", request.OutputName)
		outputPath, err := resolveCaseInsensitiveUnderRoot(workspace, outputRelativePath)
		if err != nil || !ensureWritablePathWithinRoot(workspace, outputPath) {
			writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{AgentID: request.AgentID, Content: "无法在工作目录中创建 WAV", Status: 1})
			return
		}
		if _, err := os.Lstat(outputPath); err == nil {
			writeAudioMixResp(w, http.StatusConflict, AudioMixResponse{AgentID: request.AgentID, Path: filepath.ToSlash(outputRelativePath), Content: "同名 WAV 已存在，请更换名称", Status: 1})
			return
		} else if !errors.Is(err, os.ErrNotExist) {
			writeAudioMixResp(w, http.StatusInternalServerError, AudioMixResponse{AgentID: request.AgentID, Content: "无法检查 WAV 输出路径", Status: 1})
			return
		}
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil || !ensureWritablePathWithinRoot(workspace, outputPath) {
			writeAudioMixResp(w, http.StatusInternalServerError, AudioMixResponse{AgentID: request.AgentID, Content: "无法创建 audios 目录", Status: 1})
			return
		}
		ffmpegPath, ffmpegErr := videoTrimLookPathFn("ffmpeg")
		ffprobePath, ffprobeErr := videoTrimLookPathFn("ffprobe")
		if ffmpegErr != nil || ffprobeErr != nil {
			writeAudioMixResp(w, http.StatusServiceUnavailable, AudioMixResponse{AgentID: request.AgentID, Content: "FFmpeg 或 FFprobe 未安装，无法生成 WAV", Status: 1})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
		defer cancel()
		inputArgs := []string{"-hide_banner", "-loglevel", "error", "-nostdin"}
		filterParts := make([]string, 0, len(request.Tracks)+1)
		labels := make([]string, 0, len(request.Tracks))
		for index, track := range request.Tracks {
			sourcePath, err := audioMixResolveSource(workspace, track.Path)
			if err != nil {
				writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{AgentID: request.AgentID, Content: "第 " + strconv.Itoa(index+1) + " 段音轨无效：" + err.Error(), Status: 1})
				return
			}
			sourceDuration, err := audioMixProbeSource(ctx, ffprobePath, sourcePath)
			if err != nil || track.TrimEnd > sourceDuration+0.001 {
				message := "源文件不含可解析音频流"
				if err != nil {
					message = err.Error()
				} else {
					message = "裁剪范围超出源文件时长"
				}
				writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{AgentID: request.AgentID, Content: "第 " + strconv.Itoa(index+1) + " 段音轨无效：" + message, Status: 1})
				return
			}
			if track.TrimEnd == 0 {
				track.TrimEnd = sourceDuration
			}
			duration, copies, err := audioMixTrackDuration(track)
			if err != nil {
				writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{AgentID: request.AgentID, Content: "第 " + strconv.Itoa(index+1) + " 段音轨无效：" + err.Error(), Status: 1})
				return
			}
			inputArgs = append(inputArgs, "-i", sourcePath)
			filter, err := audioMixTrackFilter(index, track, duration, copies)
			if err != nil {
				writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{AgentID: request.AgentID, Content: "第 " + strconv.Itoa(index+1) + " 段音轨无效：" + err.Error(), Status: 1})
				return
			}
			filterParts = append(filterParts, filter)
			labels = append(labels, "[m"+strconv.Itoa(index)+"]")
		}
		filterParts = append(filterParts, strings.Join(labels, "")+"amix=inputs="+strconv.Itoa(len(labels))+":duration=longest:dropout_transition=0,alimiter=limit=0.99,aformat=channel_layouts=stereo[mixout]")
		temporaryPath := strings.TrimSuffix(outputPath, ".wav") + "." + strconv.FormatInt(time.Now().UnixNano(), 10) + ".tmp.wav"
		if !ensureWritablePathWithinRoot(workspace, temporaryPath) {
			writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{AgentID: request.AgentID, Content: "无法创建临时 WAV", Status: 1})
			return
		}
		defer os.Remove(temporaryPath)
		args := append(inputArgs, "-filter_complex", strings.Join(filterParts, ";"), "-map", "[mixout]", "-ar", strconv.Itoa(request.SampleRate), "-ac", "2")
		if request.BitDepth == 24 {
			args = append(args, "-c:a", "pcm_s24le")
		} else {
			args = append(args, "-c:a", "pcm_s16le")
		}
		args = append(args, "-f", "wav", "-n", temporaryPath)
		output, err := videoTrimCommandContextFn(ctx, ffmpegPath, args...).CombinedOutput()
		if err != nil {
			message := "音频混音失败：" + videoTrimCommandError(err, output)
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				message = "音频混音超时"
			}
			writeAudioMixResp(w, http.StatusInternalServerError, AudioMixResponse{AgentID: request.AgentID, Content: message, Status: 1})
			return
		}
		info, err := os.Stat(temporaryPath)
		if err != nil || info.Size() == 0 {
			writeAudioMixResp(w, http.StatusInternalServerError, AudioMixResponse{AgentID: request.AgentID, Content: "WAV 混音未生成有效输出", Status: 1})
			return
		}
		if err := os.Link(temporaryPath, outputPath); err != nil {
			status, message := http.StatusInternalServerError, "保存 WAV 失败"
			if errors.Is(err, fs.ErrExist) {
				status, message = http.StatusConflict, "同名 WAV 已存在，请更换名称"
			}
			writeAudioMixResp(w, status, AudioMixResponse{AgentID: request.AgentID, Path: filepath.ToSlash(outputRelativePath), Content: message, Status: 1})
			return
		}
		if err := os.Remove(temporaryPath); err != nil {
			_ = os.Remove(outputPath)
			writeAudioMixResp(w, http.StatusInternalServerError, AudioMixResponse{AgentID: request.AgentID, Content: "清理临时 WAV 失败", Status: 1})
			return
		}
		flushAgentCache()
		writeAudioMixResp(w, http.StatusOK, AudioMixResponse{AgentID: request.AgentID, Path: filepath.ToSlash(outputRelativePath), SavedAs: outputPath, Content: "混音已保存", Status: 0})
	}
}

// handleAudioMixPreview renders the same constrained project as audio_mix, but
// streams a temporary WAV back to the browser and never writes an audios/ file.
func handleAudioMixPreview(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeAudioMixResp(w, http.StatusMethodNotAllowed, AudioMixResponse{Content: "method not allowed", Status: 1})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		defer r.Body.Close()
		var request AudioMixRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{Content: "请求体格式错误", Status: 1})
			return
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{AgentID: request.AgentID, Content: "请求体只能包含一个 JSON 对象", Status: 1})
			return
		}
		request.AgentID = strings.TrimSpace(request.AgentID)
		request.OutputName = strings.TrimSpace(normalizeQuotedPathArg(request.OutputName))
		if !isValidMediaPreviewAgentID(request.AgentID) || !isSafeAudioMixOutputName(request.OutputName) {
			writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{AgentID: request.AgentID, Content: "agentId 和输出文件名必须有效，且输出必须为 .wav", Status: 1})
			return
		}
		if request.SampleRate != 44100 && request.SampleRate != 48000 {
			writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{AgentID: request.AgentID, Content: "采样率只支持 44100 或 48000", Status: 1})
			return
		}
		if request.BitDepth != 16 && request.BitDepth != 24 {
			writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{AgentID: request.AgentID, Content: "位深只支持 16 或 24", Status: 1})
			return
		}
		if len(request.Tracks) == 0 || len(request.Tracks) > audioMixMaxTracks {
			writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{AgentID: request.AgentID, Content: "音轨数量必须在 1 到 64 之间", Status: 1})
			return
		}
		workspace, err := getWorkspaceByAgentID(cfg, request.AgentID)
		if err != nil {
			writeAudioMixResp(w, http.StatusNotFound, AudioMixResponse{AgentID: request.AgentID, Content: "agent not found", Status: 1})
			return
		}
		ffmpegPath, ffmpegErr := videoTrimLookPathFn("ffmpeg")
		ffprobePath, ffprobeErr := videoTrimLookPathFn("ffprobe")
		if ffmpegErr != nil || ffprobeErr != nil {
			writeAudioMixResp(w, http.StatusServiceUnavailable, AudioMixResponse{AgentID: request.AgentID, Content: "FFmpeg 或 FFprobe 未安装，无法试听混音", Status: 1})
			return
		}
		previewDir, err := os.MkdirTemp("", "deepright-audio-mix-preview-"+strconv.FormatInt(time.Now().UnixNano(), 10)+"-")
		if err != nil {
			writeAudioMixResp(w, http.StatusInternalServerError, AudioMixResponse{AgentID: request.AgentID, Content: "无法创建试听临时 WAV", Status: 1})
			return
		}
		defer os.RemoveAll(previewDir)
		temporaryPath := filepath.Join(previewDir, "mix-preview-"+strconv.FormatInt(time.Now().UnixNano(), 10)+".wav")

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
		defer cancel()
		inputArgs := []string{"-hide_banner", "-loglevel", "error", "-nostdin"}
		filterParts := make([]string, 0, len(request.Tracks)+1)
		labels := make([]string, 0, len(request.Tracks))
		for index, track := range request.Tracks {
			sourcePath, err := audioMixResolveSource(workspace, track.Path)
			if err != nil {
				writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{AgentID: request.AgentID, Content: "第 " + strconv.Itoa(index+1) + " 段音轨无效：" + err.Error(), Status: 1})
				return
			}
			sourceDuration, err := audioMixProbeSource(ctx, ffprobePath, sourcePath)
			if err != nil || track.TrimEnd > sourceDuration+0.001 {
				message := "源文件不含可解析音频流"
				if err != nil {
					message = err.Error()
				} else {
					message = "裁剪范围超出源文件时长"
				}
				writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{AgentID: request.AgentID, Content: "第 " + strconv.Itoa(index+1) + " 段音轨无效：" + message, Status: 1})
				return
			}
			if track.TrimEnd == 0 {
				track.TrimEnd = sourceDuration
			}
			duration, copies, err := audioMixTrackDuration(track)
			if err != nil {
				writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{AgentID: request.AgentID, Content: "第 " + strconv.Itoa(index+1) + " 段音轨无效：" + err.Error(), Status: 1})
				return
			}
			inputArgs = append(inputArgs, "-i", sourcePath)
			filter, err := audioMixTrackFilter(index, track, duration, copies)
			if err != nil {
				writeAudioMixResp(w, http.StatusBadRequest, AudioMixResponse{AgentID: request.AgentID, Content: "第 " + strconv.Itoa(index+1) + " 段音轨无效：" + err.Error(), Status: 1})
				return
			}
			filterParts = append(filterParts, filter)
			labels = append(labels, "[m"+strconv.Itoa(index)+"]")
		}
		filterParts = append(filterParts, strings.Join(labels, "")+"amix=inputs="+strconv.Itoa(len(labels))+":duration=longest:dropout_transition=0,alimiter=limit=0.99,aformat=channel_layouts=stereo[mixout]")
		args := append(inputArgs, "-filter_complex", strings.Join(filterParts, ";"), "-map", "[mixout]", "-ar", strconv.Itoa(request.SampleRate), "-ac", "2")
		if request.BitDepth == 24 {
			args = append(args, "-c:a", "pcm_s24le")
		} else {
			args = append(args, "-c:a", "pcm_s16le")
		}
		args = append(args, "-f", "wav", "-n", temporaryPath)
		output, err := videoTrimCommandContextFn(ctx, ffmpegPath, args...).CombinedOutput()
		if err != nil {
			message := "音频试听失败：" + videoTrimCommandError(err, output)
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				message = "音频试听超时"
			}
			writeAudioMixResp(w, http.StatusInternalServerError, AudioMixResponse{AgentID: request.AgentID, Content: message, Status: 1})
			return
		}
		info, err := os.Stat(temporaryPath)
		if err != nil || info.Size() == 0 {
			writeAudioMixResp(w, http.StatusInternalServerError, AudioMixResponse{AgentID: request.AgentID, Content: "试听 WAV 未生成有效输出", Status: 1})
			return
		}
		preview, err := os.Open(temporaryPath)
		if err != nil {
			writeAudioMixResp(w, http.StatusInternalServerError, AudioMixResponse{AgentID: request.AgentID, Content: "无法读取试听 WAV", Status: 1})
			return
		}
		defer preview.Close()
		w.Header().Set("Content-Type", "audio/wav")
		w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Disposition", "inline; filename=\"mix-preview.wav\"")
		_, _ = io.Copy(w, preview)
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

// handleMCPWarning serves MCP parse errors independently from SKILL.md
// warnings. The payload deliberately contains only Agent ID, relative path,
// and a stable validation message; MCP file contents never leave the process.
func handleMCPWarning(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		refresh := strings.TrimSpace(r.URL.Query().Get("refresh"))
		force := refresh != "" && refresh != "0" && strings.ToLower(refresh) != "false"
		if _, err := refreshIntegrationMCPMetadata(cfg, force); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": "读取 MCP 配置失败"})
			return
		}
		warnings := []integrationMCPWarning(nil)
		if cfg != nil {
			warnings = cfg.mcpState().warnings()
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": 0, "data": warnings})
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
			"deletedCmdLog":          snapshot.DeletedCmdLog,
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
	writeSnapshot := func(w http.ResponseWriter, snapshot runtimehost.Snapshot) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status": 0,
			"data":   map[string]string{"host": snapshot.Host},
		})
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !isLocalManagementRequest(r) {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"status":  1,
				"content": "only localhost requests can modify service address",
			})
			return
		}

		switch r.Method {
		case http.MethodGet:
			writeSnapshot(w, cfg.runtimeHostSnapshot())
			return
		case http.MethodDelete:
			snapshot, err := cfg.resetPersistentHost()
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"status": 1, "content": err.Error()})
				return
			}
			log.Printf("service address restored to default: %s", snapshot.Host)
			writeSnapshot(w, snapshot)
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
				snapshot, err := cfg.resetPersistentHost()
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{"status": 1, "content": err.Error()})
					return
				}
				log.Printf("service address restored to default: %s", snapshot.Host)
				writeSnapshot(w, snapshot)
				return
			}
			if _, err := runtimehost.Validate(payload.Host); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"status":  1,
					"content": err.Error(),
				})
				return
			}
			snapshot, err := cfg.setPersistentHost(payload.Host)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"status":  1,
					"content": err.Error(),
				})
				return
			}
			log.Printf("service address updated: %s", snapshot.Host)
			writeSnapshot(w, snapshot)
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
		refreshIntegrationAtMenuCacheAfterAgentChange(cfg, "agent import")
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
		refreshIntegrationAtMenuCacheAfterAgentChange(cfg, "agent init")
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
		refreshIntegrationAtMenuCacheAfterAgentChange(cfg, "agent copy")
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
		flushAgentCache()
		refreshIntegrationAtMenuCacheAfterAgentChange(cfg, "agent delete")
		if cronDB != nil {
			if _, _, cleanupErr := deleteAgentCronData(cronDB, name); cleanupErr != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "cleanup cron failed: " + cleanupErr.Error()})
				return
			}
		}
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
// Proxy: /api/upload — upload files to an agent workspace directory
// ═══════════════════════════════════════════════════════════════════════════

// resolveUploadDestination keeps uploads inside an Agent workspace.  A caller
// may explicitly name an existing relative directory with the dest query
// parameter. When preferTmp is set, that directory's existing tmp child takes
// precedence. The legacy/default destination is the workspace tmp when it
// already exists; a workspace without tmp receives the upload at its root.
func resolveUploadDestination(workspace, requestedDest string, hasRequestedDest, preferTmp bool) (string, error) {
	workspaceAbs, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("resolve workspace directory failed: %w", err)
	}
	if hasRequestedDest {
		cleanDest := filepath.Clean(filepath.FromSlash(requestedDest))
		if filepath.IsAbs(cleanDest) || cleanDest == ".." || strings.HasPrefix(cleanDest, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("invalid upload destination")
		}
		destination := workspaceAbs
		if cleanDest != "." && cleanDest != "" {
			destination = filepath.Join(workspaceAbs, cleanDest)
		}
		info, err := os.Stat(destination)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("upload destination does not exist")
			}
			return "", fmt.Errorf("read upload destination failed: %w", err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("upload destination is not a directory")
		}
		if destination != workspaceAbs && !ensureWritablePathWithinRoot(workspaceAbs, destination) {
			return "", fmt.Errorf("invalid upload destination")
		}
		if preferTmp {
			tmpDir := filepath.Join(destination, "tmp")
			if info, err := os.Stat(tmpDir); err == nil && info.IsDir() && ensureWritablePathWithinRoot(workspaceAbs, tmpDir) {
				return tmpDir, nil
			}
		}
		return destination, nil
	}

	tmpDir := filepath.Join(workspaceAbs, "tmp")
	if info, err := os.Stat(tmpDir); err == nil && info.IsDir() && ensureWritablePathWithinRoot(workspaceAbs, tmpDir) {
		return tmpDir, nil
	}
	return workspaceAbs, nil
}

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
		workspace, err := getWorkspaceByAgentID(cfg, agentID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "agent not found: " + agentID})
			return
		}
		_, hasRequestedDest := r.URL.Query()["dest"]
		preferTmp := r.URL.Query().Get("preferTmp") == "1"
		uploadDir, err := resolveUploadDestination(workspace, r.URL.Query().Get("dest"), hasRequestedDest, preferTmp)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
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
			cleanRelPath := filepath.Clean(filepath.FromSlash(relPath))
			if cleanRelPath == "." || cleanRelPath == "" || cleanRelPath == ".." || filepath.IsAbs(cleanRelPath) || strings.HasPrefix(cleanRelPath, ".."+string(filepath.Separator)) {
				continue
			}
			destPath := filepath.Join(uploadDir, cleanRelPath)
			if !ensureWritablePathWithinRoot(uploadDir, destPath) {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				continue
			}
			src, err := fh.Open()
			if err != nil {
				continue
			}
			data, err := io.ReadAll(src)
			src.Close()
			if err != nil {
				continue
			}
			if err := os.WriteFile(destPath, data, 0644); err != nil {
				continue
			}
			uploaded = append(uploaded, filepath.ToSlash(cleanRelPath))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "agentId": agentID, "files": uploaded, "dest": uploadDir})
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
	if err := clearRemovedMultimodalProviderReferences(tx, removedModels); err != nil {
		return err
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

func multimodalProviderReference(value string) string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "@") {
		return ""
	}
	return sharedutil.NormalizeTokenModelName(strings.TrimSpace(strings.TrimPrefix(value, "@")))
}

func clearRemovedMultimodalProviderReferences(tx *sql.Tx, removedModels []string) error {
	removed := make(map[string]struct{}, len(removedModels))
	for _, model := range removedModels {
		model = sharedutil.NormalizeTokenModelName(model)
		if model != "" {
			removed[model] = struct{}{}
		}
	}
	if len(removed) == 0 {
		return nil
	}
	now := time.Now().Format(time.RFC3339)

	rows, err := tx.Query(`SELECT model, __model_multi_input, __model_multi_output FROM token_store`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type referenceUpdate struct {
		model            string
		clearMultiInput  bool
		clearMultiOutput bool
	}
	updates := make([]referenceUpdate, 0)
	for rows.Next() {
		var model, multiInput, multiOutput string
		if err := rows.Scan(&model, &multiInput, &multiOutput); err != nil {
			return err
		}
		_, clearMultiInput := removed[multimodalProviderReference(multiInput)]
		_, clearMultiOutput := removed[multimodalProviderReference(multiOutput)]
		if clearMultiInput || clearMultiOutput {
			updates = append(updates, referenceUpdate{model: model, clearMultiInput: clearMultiInput, clearMultiOutput: clearMultiOutput})
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, update := range updates {
		if update.clearMultiInput && update.clearMultiOutput {
			if _, err := tx.Exec(`UPDATE token_store SET __model_multi_input = '', __model_multi_output = '', updated_at = ? WHERE model = ?`, now, update.model); err != nil {
				return err
			}
			continue
		}
		if update.clearMultiInput {
			if _, err := tx.Exec(`UPDATE token_store SET __model_multi_input = '', updated_at = ? WHERE model = ?`, now, update.model); err != nil {
				return err
			}
			continue
		}
		if _, err := tx.Exec(`UPDATE token_store SET __model_multi_output = '', updated_at = ? WHERE model = ?`, now, update.model); err != nil {
			return err
		}
	}
	return nil
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

	now := time.Now().Format(time.RFC3339)
	for _, variant := range variants {
		if _, err := tx.Exec(`DELETE FROM token_store WHERE model = ?`, variant.Model); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO proxy_agent_provider_log (agent_id, chat_id, model, token, action, updated_at) VALUES (?,?,?,?,?,?)`, "", "", variant.Model, variant.Token, "delete", now); err != nil {
			return err
		}
	}
	if err := clearRemovedMultimodalProviderReferences(tx, []string{model}); err != nil {
		return err
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
		refreshIntegrationAtMenuCacheAfterAgentChange(cfg, "agent swarm config")
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

const (
	modelTestMaxSSEEventBytes = 64 * 1024
	modelTestMaxErrorRunes    = 500
)

var (
	modelTestSensitiveValuePattern   = regexp.MustCompile(`(?i)(["']?(?:authorization|api[-_ ]?key|token|secret)["']?\s*[:=]\s*["']?)([^\s,"';&}]+)`)
	modelTestInlineStatusCodePattern = regexp.MustCompile(`(?i)\b(?:http(?:\s+status)?|code|status)\s*(?:=|:|\s)\s*([1-5][0-9]{2})\b`)
	modelTestURLAllowedCharacters    = regexp.MustCompile(`^[A-Za-z0-9\-._~:/?#[\]@!$&'()*+,;=%]+$`)
)

// modelTestRequest contains transient provider material plus the page's
// request-scoped metadata. It is never written to token_store or any of the
// chat/agent persistence paths.
type modelTestRequest struct {
	Model            string                   `json:"model"`
	Token            string                   `json:"token"`
	BaseURL          string                   `json:"__url"`
	ModelBase        string                   `json:"__model"`
	ModelFast        string                   `json:"__model_fast"`
	ModelThinking    string                   `json:"__model_thinking"`
	ModelMultiInput  string                   `json:"__model_multi_input"`
	ModelMultiOutput string                   `json:"__model_multi_output"`
	Metadata         modelTestRequestMetadata `json:"metadata"`
}

type modelTestRequestMetadata struct {
	Device string `json:"device"`
	Theme  string `json:"theme"`
}

type modelTestConfig struct {
	Content           string
	Timeout           time.Duration
	PageResponseCodes configuredPageResponseCodes
}

// configuredPageResponseCodes is the small, public part of the static page
// protocol shared by chat SSE, model tests, and cli/get heartbeats.
type configuredPageResponseCodes struct {
	NewTab int
	Iframe int
}

func (codes configuredPageResponseCodes) matches(code int) bool {
	return code > 0 && (code == codes.NewTab || code == codes.Iframe)
}

func configuredPageResponseCodesFromRaw(raw map[string]interface{}) configuredPageResponseCodes {
	page, ok := raw["page"].(map[string]interface{})
	if !ok || page == nil {
		return configuredPageResponseCodes{}
	}
	return configuredPageResponseCodes{
		NewTab: configuredPageResponseCode(page["new_tab"]),
		Iframe: configuredPageResponseCode(page["iframe"]),
	}
}

func configuredPageResponseCode(raw interface{}) int {
	var code int64
	switch value := raw.(type) {
	case json.Number:
		parsed, err := value.Int64()
		if err != nil {
			return 0
		}
		code = parsed
	default:
		parsed, ok := parseResponseStatusCode(value)
		if !ok {
			return 0
		}
		code = int64(parsed)
	}
	if code <= 0 || code > int64(^uint(0)>>1) {
		return 0
	}
	return int(code)
}

func readConfiguredPageResponseCodes() configuredPageResponseCodes {
	raw, _, err := readIntegrationStartupConfigRaw()
	if err != nil || raw == nil {
		return configuredPageResponseCodes{}
	}
	return configuredPageResponseCodesFromRaw(raw)
}

// handleModelTest runs an isolated, non-persistent provider probe. It does
// not use handleChatCompletions: that path expands Agent metadata and stores
// request/response history, which is expressly not valid for configuration
// tests.
func handleModelTest(cfg *Config, proxyClient *http.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !isLocalManagementRequest(r) {
			writeModelTestJSONError(w, http.StatusForbidden, "仅允许本机测试模型配置")
			return
		}

		request, err := decodeModelTestRequest(r.Body)
		if err != nil {
			writeModelTestJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		testConfig, err := readModelTestConfig()
		if err != nil {
			writeModelTestJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		sessionID, err := newModelTestUUID()
		if err != nil {
			writeModelTestJSONError(w, http.StatusInternalServerError, "创建测试会话失败")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), testConfig.Timeout)
		defer cancel()
		upstreamPayload := buildModelTestUpstreamRequest(cfg, request, sessionID, testConfig.Content)
		metadata, ok := upstreamPayload["metadata"].(map[string]interface{})
		if !ok {
			writeModelTestJSONError(w, http.StatusInternalServerError, "创建测试请求失败")
			return
		}
		if err := injectIntegrationConfigPathMetadata(metadata, cfg); err != nil {
			writeModelTestJSONError(w, http.StatusInternalServerError, "配置错误：Integration 主配置不可用")
			return
		}
		requestBody, err := json.Marshal(upstreamPayload)
		if err != nil {
			writeModelTestJSONError(w, http.StatusInternalServerError, "创建测试请求失败")
			return
		}
		targetURL := strings.TrimRight(cfg.currentHost(), "/") + "/v1/chat/completions"
		upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(requestBody))
		if err != nil {
			writeModelTestJSONError(w, http.StatusInternalServerError, "创建测试请求失败")
			return
		}
		upstreamReq.Header.Set("Content-Type", "application/json")
		upstreamReq.Header.Set("Accept", "text/event-stream")
		upstreamReq.Header.Set("Authorization", request.Token)
		deviceID, _ := metadata["deviceId"].(string)
		setDeviceIDHeader(upstreamReq, deviceID)

		client := proxyClient
		if client == nil {
			client = http.DefaultClient
		}
		finishSSE := trackIntegrationSSE(cfg)
		defer finishSSE()
		resp, err := client.Do(upstreamReq)
		if err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				writeModelTestJSONError(w, http.StatusGatewayTimeout, "配置错误：测试超时")
				return
			}
			if errors.Is(ctx.Err(), context.Canceled) {
				return
			}
			writeModelTestJSONError(w, http.StatusBadGateway, "配置错误：无法连接测试服务："+sanitizeModelTestError(err.Error(), request.Token))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, modelTestMaxSSEEventBytes))
			message := modelTestHTTPError(resp.StatusCode, string(body), request.Token)
			writeModelTestJSONError(w, http.StatusBadGateway, message)
			return
		}
		if !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, modelTestMaxSSEEventBytes))
			message := "配置错误：服务商未返回 SSE 响应"
			if detail := modelTestProviderErrorDetail(string(body), request.Token, resp.StatusCode); detail != "" {
				message += "：" + detail
			}
			writeModelTestJSONError(w, http.StatusBadGateway, message)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Accel-Buffering", "no")
		var pagePacket *modelTestPagePacket
		if err := consumeModelTestSSE(ctx, resp.Body, request.Token, modelTestSSEOptions{
			PageResponseCodes: testConfig.PageResponseCodes,
			OnPagePacket: func(packet modelTestPagePacket) {
				pagePacket = &packet
				writeModelTestSSEPagePacket(w, packet)
			},
		}); err != nil {
			if errors.Is(ctx.Err(), context.Canceled) && !errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return
			}
			writeModelTestSSEResult(w, false, modelTestErrorMessage(err, request.Token))
			return
		}
		if pagePacket != nil {
			writeModelTestSSEResult(w, false, pagePacket.Message)
			return
		}
		writeModelTestSSEResult(w, true, "配置成功")
	}
}

func decodeModelTestRequest(body io.ReadCloser) (modelTestRequest, error) {
	var request modelTestRequest
	if body == nil {
		return request, fmt.Errorf("配置错误：测试请求为空")
	}
	defer body.Close()
	decoder := json.NewDecoder(io.LimitReader(body, modelTestMaxSSEEventBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return request, fmt.Errorf("配置错误：测试请求无效")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return request, fmt.Errorf("配置错误：测试请求无效")
	}
	request.Model = sharedutil.NormalizeTokenModelName(request.Model)
	request.Token = strings.TrimSpace(request.Token)
	request.BaseURL = strings.TrimSpace(request.BaseURL)
	request.ModelBase = strings.TrimSpace(request.ModelBase)
	request.ModelFast = strings.TrimSpace(request.ModelFast)
	request.ModelThinking = strings.TrimSpace(request.ModelThinking)
	request.ModelMultiInput = strings.TrimSpace(request.ModelMultiInput)
	request.ModelMultiOutput = strings.TrimSpace(request.ModelMultiOutput)
	request.Metadata.Device = strings.TrimSpace(request.Metadata.Device)
	request.Metadata.Theme = strings.ToLower(strings.TrimSpace(request.Metadata.Theme))
	if request.Model == "" {
		return request, fmt.Errorf("配置错误：未选择模型服务商")
	}
	if request.Token == "" || request.Token == maskedTokenValue {
		return request, fmt.Errorf("配置错误：模型密钥不能为空")
	}
	if !isValidModelTestBaseURL(request.BaseURL) {
		return request, fmt.Errorf("配置错误：模型 URL 格式不正确，请填写 http:// 或 https:// 开头的完整 URL")
	}
	if request.ModelBase == "" {
		return request, fmt.Errorf("配置错误：基础模型 __model 不能为空")
	}
	if request.Metadata.Device == "" {
		return request, fmt.Errorf("配置错误：metadata.device 不能为空")
	}
	if request.Metadata.Theme != "cold" && request.Metadata.Theme != "warm" {
		return request, fmt.Errorf("配置错误：metadata.theme 必须为 cold 或 warm")
	}
	return request, nil
}

func isValidModelTestBaseURL(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	if !modelTestURLAllowedCharacters.MatchString(value) || !hasValidModelTestURLPercentEncoding(value) {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed == nil || parsed.User != nil || parsed.Hostname() == "" {
		return false
	}
	return strings.EqualFold(parsed.Scheme, "http") || strings.EqualFold(parsed.Scheme, "https")
}

func hasValidModelTestURLPercentEncoding(value string) bool {
	for index := 0; index < len(value); index++ {
		if value[index] != '%' {
			continue
		}
		if index+2 >= len(value) || !isModelTestURLHexDigit(value[index+1]) || !isModelTestURLHexDigit(value[index+2]) {
			return false
		}
		index += 2
	}
	return true
}

func isModelTestURLHexDigit(value byte) bool {
	return (value >= '0' && value <= '9') || (value >= 'a' && value <= 'f') || (value >= 'A' && value <= 'F')
}

func readModelTestConfig() (modelTestConfig, error) {
	raw, _, err := readIntegrationStartupConfigRaw()
	if err != nil {
		return modelTestConfig{}, fmt.Errorf("配置错误：读取 config.json.test 失败")
	}
	if raw == nil {
		return modelTestConfig{}, fmt.Errorf("配置错误：config.json.test 缺失")
	}
	testRaw, ok := raw["test"].(map[string]interface{})
	if !ok || testRaw == nil {
		return modelTestConfig{}, fmt.Errorf("配置错误：config.json.test 缺失")
	}
	content, ok := testRaw["content"].(string)
	content = strings.TrimSpace(content)
	if !ok || content == "" {
		return modelTestConfig{}, fmt.Errorf("配置错误：config.json.test.content 缺失")
	}
	timeoutSeconds, ok := modelTestTimeoutSeconds(testRaw["timeout"])
	if !ok || timeoutSeconds <= 0 || timeoutSeconds > int64((1<<63-1)/int64(time.Second)) {
		return modelTestConfig{}, fmt.Errorf("配置错误：config.json.test.timeout 必须为正整数秒")
	}
	return modelTestConfig{
		Content:           content,
		Timeout:           time.Duration(timeoutSeconds) * time.Second,
		PageResponseCodes: configuredPageResponseCodesFromRaw(raw),
	}, nil
}

func modelTestTimeoutSeconds(raw interface{}) (int64, bool) {
	switch value := raw.(type) {
	case json.Number:
		parsed, err := value.Int64()
		return parsed, err == nil
	case int:
		return int64(value), true
	case int64:
		return value, true
	case float64:
		if value != float64(int64(value)) {
			return 0, false
		}
		return int64(value), true
	default:
		return 0, false
	}
}

func buildModelTestUpstreamRequest(cfg *Config, request modelTestRequest, sessionID, content string) map[string]interface{} {
	metadata := map[string]interface{}{
		"test":     true,
		"type":     "test",
		"chat":     sessionID,
		"device":   request.Metadata.Device,
		"deviceId": cfg.effectiveDeviceID(),
		"theme":    request.Metadata.Theme,
		"port":     integrationMetadataPort(cfg),
		// Keep the runtime fields required by the upstream /v1/chat/completions
		// pipeline. They describe this Integration process only; unlike ordinary
		// chat requests, a model test still does not merge Agent, skills, memory,
		// knowledge, router, task, or media context.
		"app":       firstNonEmpty(resolveExecutablePath(os.Args[0]), os.Args[0]),
		"workspace": modelTestWorkspace(cfg),
		"terminal":  detectTerminal(),
		"Origin":    modelTestOrigin(cfg),
		"sys":       agentcore.DetectSys(),
		"provider":  request.Model,
		"__model":   request.ModelBase,
	}
	if request.BaseURL != "" {
		metadata["__url"] = request.BaseURL
	}
	if request.ModelFast != "" {
		metadata["__model_fast"] = request.ModelFast
	}
	if request.ModelThinking != "" {
		metadata["__model_thinking"] = request.ModelThinking
	}
	if request.ModelMultiInput != "" {
		metadata["__model_multi_input"] = request.ModelMultiInput
	}
	if request.ModelMultiOutput != "" {
		metadata["__model_multi_output"] = request.ModelMultiOutput
	}
	return map[string]interface{}{
		"model":    request.Model,
		"messages": []map[string]string{{"role": "user", "content": content}},
		"stream":   true,
		"metadata": metadata,
	}
}

func modelTestWorkspace(cfg *Config) string {
	if cfg == nil {
		return ""
	}
	workspace := strings.TrimSpace(cfg.AgentDir)
	if workspace == "" {
		return ""
	}
	if absolute, err := filepath.Abs(workspace); err == nil {
		return absolute
	}
	return filepath.Clean(workspace)
}

func modelTestOrigin(cfg *Config) string {
	return fmt.Sprintf("http://localhost:%d", integrationMetadataPort(cfg))
}

func newModelTestUUID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		value[0:4], value[4:6], value[6:8], value[8:10], value[10:16]), nil
}

type modelTestSSEOptions struct {
	PageResponseCodes configuredPageResponseCodes
	OnPagePacket      func(modelTestPagePacket)
}

type modelTestPagePacket struct {
	Code    int
	Content string
	Message string
}

func consumeModelTestSSE(ctx context.Context, body io.Reader, token string, options ...modelTestSSEOptions) error {
	var option modelTestSSEOptions
	if len(options) > 0 {
		option = options[0]
	}
	buffer := make([]byte, 0, 4096)
	chunk := make([]byte, 2048)
	sawBusinessData := false
	sawDone := false
	for {
		count, readErr := body.Read(chunk)
		if count > 0 {
			buffer = append(buffer, chunk[:count]...)
			if len(buffer) > modelTestMaxSSEEventBytes {
				return fmt.Errorf("配置错误：SSE 事件过大或未正常分帧")
			}
			for _, event := range splitCompleteSSEEvents(&buffer) {
				business, done, err := inspectModelTestSSEEvent(event, token, option)
				if err != nil {
					return err
				}
				sawBusinessData = sawBusinessData || business
				sawDone = sawDone || done
			}
		}
		if readErr == nil {
			continue
		}
		if !errors.Is(readErr, io.EOF) {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("配置错误：测试超时")
			}
			return fmt.Errorf("配置错误：SSE 响应中断：%s", sanitizeModelTestError(readErr.Error(), token))
		}
		break
	}
	for _, event := range flushTrailingSSEBytes(&buffer) {
		business, done, err := inspectModelTestSSEEvent(event, token, option)
		if err != nil {
			return err
		}
		sawBusinessData = sawBusinessData || business
		sawDone = sawDone || done
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("配置错误：测试超时")
	}
	if !sawBusinessData {
		return fmt.Errorf("配置错误：SSE 响应未包含有效业务数据")
	}
	if !sawDone {
		return fmt.Errorf("配置错误：SSE 响应未正常结束（缺少 [DONE]）")
	}
	return nil
}

func inspectModelTestSSEEvent(event, token string, option modelTestSSEOptions) (bool, bool, error) {
	if strings.Contains(strings.ToLower(event), "event: error") {
		return false, false, fmt.Errorf("配置错误：%s", modelTestSSEErrorDetail(event, token))
	}
	business := false
	done := false
	for _, line := range strings.Split(event, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}
		if payload == "[DONE]" {
			done = true
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(payload), &raw); err != nil {
			return false, done, fmt.Errorf("配置错误：SSE 响应包含无效数据")
		}
		if packet, ok := modelTestPagePacketFromPayload(raw, option.PageResponseCodes); ok {
			business = true
			if option.OnPagePacket != nil {
				option.OnPagePacket(packet)
			}
			continue
		}
		if modelTestPayloadIsError(raw) {
			return false, done, fmt.Errorf("配置错误：%s", modelTestPayloadErrorDetail(raw, token))
		}
		if choices, ok := raw["choices"].([]interface{}); ok && len(choices) > 0 {
			business = true
		}
	}
	return business, done, nil
}

func modelTestPagePacketFromPayload(raw map[string]interface{}, codes configuredPageResponseCodes) (modelTestPagePacket, bool) {
	if raw == nil || !codes.matches(modelTestPayloadTopLevelCode(raw)) {
		return modelTestPagePacket{}, false
	}
	choices, ok := raw["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return modelTestPagePacket{}, false
	}
	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return modelTestPagePacket{}, false
	}
	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		return modelTestPagePacket{}, false
	}
	content, ok := delta["content"].(string)
	if !ok {
		return modelTestPagePacket{}, false
	}
	var page map[string]interface{}
	if json.Unmarshal([]byte(content), &page) != nil || page == nil {
		return modelTestPagePacket{}, false
	}
	if _, ok := page["url"].(string); !ok {
		return modelTestPagePacket{}, false
	}
	message, ok := page["message"].(string)
	if !ok {
		return modelTestPagePacket{}, false
	}
	return modelTestPagePacket{Code: modelTestPayloadTopLevelCode(raw), Content: content, Message: message}, true
}

func modelTestPayloadTopLevelCode(raw map[string]interface{}) int {
	if raw == nil {
		return 0
	}
	code, ok := parseResponseStatusCode(raw["code"])
	if !ok {
		return 0
	}
	return code
}

func modelTestPayloadIsError(raw map[string]interface{}) bool {
	if modelTestPayloadStatusCode(raw) != 0 {
		return true
	}
	if responseValueIsAbnormal(raw) {
		return true
	}
	if value, exists := raw["error"]; exists && value != nil {
		switch typed := value.(type) {
		case string:
			return strings.TrimSpace(typed) != ""
		case map[string]interface{}:
			return len(typed) > 0
		}
	}
	return false
}

func modelTestSSEErrorDetail(event, token string) string {
	for _, payload := range extractResponsePayloads(event) {
		var raw map[string]interface{}
		if json.Unmarshal([]byte(payload), &raw) == nil {
			return modelTestPayloadErrorDetail(raw, token)
		}
	}
	return sanitizeModelTestError(event, token)
}

func modelTestPayloadErrorDetail(raw map[string]interface{}, token string) string {
	return formatModelTestProviderFailure(modelTestPayloadStatusCode(raw), modelTestPayloadErrorContent(raw, token))
}

func modelTestPayloadErrorContent(raw map[string]interface{}, token string) string {
	// Some compatible providers encode an error as an assistant delta instead
	// of the conventional top-level error object. Prefer its content so the UI
	// never has to show the complete provider payload (which may include
	// transport metadata).
	if text := modelTestNestedContentText(raw); text != "" {
		return sanitizeModelTestError(text, token)
	}
	for _, key := range []string{"error", "message", "detail", "content"} {
		if value, ok := raw[key]; ok && value != nil {
			if text := modelTestErrorValueText(value); text != "" {
				return sanitizeModelTestError(text, token)
			}
		}
	}
	encoded, _ := json.Marshal(raw)
	return sanitizeModelTestError(string(encoded), token)
}

func modelTestPayloadStatusCode(raw map[string]interface{}) int {
	if raw == nil {
		return 0
	}
	for _, key := range []string{"code", "status"} {
		if code, ok := parseResponseStatusCode(raw[key]); ok && (code < 200 || code >= 300) {
			return code
		}
	}
	for _, key := range []string{"error", "data", "result"} {
		if nested, ok := raw[key].(map[string]interface{}); ok {
			if code := modelTestPayloadStatusCode(nested); code != 0 {
				return code
			}
		}
	}
	if code := modelTestStatusCodeFromText(modelTestNestedContentText(raw)); code != 0 {
		return code
	}
	return 0
}

func modelTestStatusCodeFromText(value string) int {
	match := modelTestInlineStatusCodePattern.FindStringSubmatch(value)
	if len(match) < 2 {
		return 0
	}
	code, err := strconv.Atoi(match[1])
	if err != nil || (code >= 200 && code < 300) {
		return 0
	}
	return code
}

func formatModelTestProviderFailure(status int, detail string) string {
	detail = strings.TrimSpace(detail)
	standard := modelTestStandardStatusMessage(status)
	if standard != "" {
		return standard
	}
	if detail != "" {
		return detail
	}
	if status > 0 {
		return fmt.Sprintf("服务商返回 HTTP %d", status)
	}
	return "测试服务返回了未知错误"
}

func modelTestStandardStatusMessage(status int) string {
	switch {
	case status == http.StatusBadRequest:
		return "服务商请求无效（400），请稍后重试。"
	case status == http.StatusUnauthorized:
		return "服务商身份认证失败（401），请检查 API Key/Token 是否正确、有效且未过期"
	case status == http.StatusForbidden:
		return "服务商拒绝访问（403），请确认密钥已获该模型或接口的访问权限"
	case status == http.StatusNotFound:
		return "服务商资源不存在（404），请检查模型 URL 和基础模型"
	case status == http.StatusTooManyRequests:
		return "服务商请求受限（429），请稍后重试并检查账户额度或速率限制"
	case status == http.StatusServiceUnavailable:
		return "服务商暂不可用（503），请稍后重试"
	case status >= 500 && status <= 599:
		return fmt.Sprintf("服务商服务异常（%d），请稍后重试", status)
	default:
		return ""
	}
}

func modelTestNestedContentText(value interface{}) string {
	return modelTestNestedContentTextAtDepth(value, 0)
}

func modelTestNestedContentTextAtDepth(value interface{}, depth int) string {
	if depth > 16 || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []interface{}:
		for _, item := range typed {
			if text := modelTestNestedContentTextAtDepth(item, depth+1); text != "" {
				return text
			}
		}
	case map[string]interface{}:
		if content, ok := typed["content"]; ok {
			if text := modelTestNestedContentTextAtDepth(content, depth+1); text != "" {
				return text
			}
		}
		// Follow the provider response structure deterministically. This covers
		// choices[].delta.content as well as common wrapper objects.
		for _, key := range []string{"choices", "choice", "delta", "error", "data", "result"} {
			if nested, ok := typed[key]; ok {
				if text := modelTestNestedContentTextAtDepth(nested, depth+1); text != "" {
					return text
				}
			}
		}
	}
	return ""
}

func modelTestErrorValueText(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	case map[string]interface{}:
		for _, key := range []string{"message", "detail", "content", "type", "code"} {
			if nested, ok := typed[key]; ok {
				if text := modelTestErrorValueText(nested); text != "" {
					return text
				}
			}
		}
	}
	return ""
}

func modelTestHTTPError(status int, body, token string) string {
	return "配置错误：" + modelTestProviderErrorDetail(body, token, status)
}

func modelTestProviderErrorDetail(body, token string, fallbackStatus int) string {
	var raw map[string]interface{}
	if json.Unmarshal([]byte(body), &raw) == nil && raw != nil {
		status := modelTestPayloadStatusCode(raw)
		if status == 0 {
			status = fallbackStatus
		}
		return formatModelTestProviderFailure(status, modelTestPayloadErrorContent(raw, token))
	}
	return formatModelTestProviderFailure(fallbackStatus, sanitizeModelTestError(body, token))
}

func modelTestErrorMessage(err error, token string) string {
	if err == nil {
		return "配置错误：测试失败"
	}
	return sanitizeModelTestError(err.Error(), token)
}

func sanitizeModelTestError(raw, token string) string {
	message := strings.TrimSpace(raw)
	if token = strings.TrimSpace(token); token != "" {
		message = strings.ReplaceAll(message, token, "[REDACTED]")
		message = strings.ReplaceAll(message, "Bearer "+token, "Bearer [REDACTED]")
	}
	message = modelTestSensitiveValuePattern.ReplaceAllString(message, "${1}[REDACTED]")
	message = strings.Join(strings.Fields(message), " ")
	if message == "" {
		return "测试服务返回了未知错误"
	}
	runes := []rune(message)
	if len(runes) > modelTestMaxErrorRunes {
		return string(runes[:modelTestMaxErrorRunes]) + "…"
	}
	return message
}

func writeModelTestJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": message})
}

func writeModelTestSSEResult(w http.ResponseWriter, success bool, message string) {
	payload, _ := json.Marshal(map[string]interface{}{
		"status":  map[bool]int{true: 0, false: 1}[success],
		"content": message,
	})
	_, _ = fmt.Fprintf(w, "event: result\ndata: %s\n\n", payload)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// writeModelTestSSEPagePacket deliberately forwards only the fields Site needs
// for the configured page protocol, never ordinary model-test SSE payloads.
func writeModelTestSSEPagePacket(w http.ResponseWriter, packet modelTestPagePacket) {
	payload, _ := json.Marshal(map[string]interface{}{
		"code": packet.Code,
		"choices": []map[string]interface{}{{
			"delta": map[string]string{"content": packet.Content},
		}},
	})
	_, _ = fmt.Fprintf(w, "event: page\ndata: %s\n\n", payload)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

var integrationClientRuntimeConfigFields = []string{
	"agent-dir",
	"app-dir",
	"app",
	"default-dir",
	"site",
	"resources-dir",
	"version",
	"knowledge",
	"skills_git_install",
	"shortcut",
	"miniapp",
	"page",
}

func integrationClientRuntimeConfig(raw map[string]interface{}) map[string]interface{} {
	config := make(map[string]interface{})
	for _, key := range integrationClientRuntimeConfigFields {
		if value, ok := raw[key]; ok {
			config[key] = value
		}
	}
	config["runtime-platform"] = integrationClientRuntimePlatform()
	return config
}

func integrationClientRuntimePlatform() string {
	switch integrationRuntimeGOOS {
	case "linux":
		if integrationBrowserIsWSL() {
			return "wsl"
		}
		return "linux"
	case "darwin":
		return "macos"
	default:
		return integrationRuntimeGOOS
	}
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
		if _, err := tx.Exec(`DELETE FROM task_detail WHERE meta_id = ? AND started NOT IN (3, 4)`, metaID); err != nil {
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
		query += ` AND d.started NOT IN (3, 4)`
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
		if _, err := tx.Exec(`DELETE FROM task_detail WHERE meta_id = ? AND started NOT IN (3, 4)`, metaID); err != nil {
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
			reply_state TEXT NOT NULL DEFAULT '',
			reply_uuid TEXT NOT NULL DEFAULT '',
			reply_started_at INTEGER NOT NULL DEFAULT 0,
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
		`ALTER TABLE task_detail ADD COLUMN reply_state TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE task_detail ADD COLUMN reply_uuid TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE task_detail ADD COLUMN reply_started_at INTEGER NOT NULL DEFAULT 0`,
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
		`CREATE INDEX IF NOT EXISTS idx_detail_reply_dispatch ON task_detail(reply_state, started, exec_time DESC, id DESC)`,
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
	// A legacy completed third-party task may have already reached its remote
	// plugin before an older runtime crashed. Quarantine it instead of guessing
	// and replaying a possibly delivered result after this upgrade.
	if _, err := db.Exec(`
		UPDATE task_detail
		SET reply_state = 'unknown'
		WHERE task_type != 'cron' AND started = 3 AND replied_at = '' AND reply_state = ''
	`); err != nil {
		log.Printf("[cron] quarantine legacy completion replies failed: %v", err)
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

// appendCronDetailLogTx is used by mutations that require the detail and its
// audit entry to commit or roll back together. ensureCronSchema has already
// materialized every selected column before the transaction starts.
func appendCronDetailLogTx(tx *sql.Tx, entry cronDetailLogEntry) error {
	if tx == nil {
		return fmt.Errorf("transaction is required")
	}
	_, err := tx.Exec(`INSERT INTO cron_detail_log (detail_id, meta_id, agent_id, chat_id, task_type, action, exec_time, model, thinking, verify, router_disable, content, response_schema, started, occurred_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		entry.DetailID, entry.MetaID, entry.AgentID, entry.ChatID, entry.TaskType, entry.Action, entry.ExecTime,
		entry.Model, boolToInt(entry.Thinking), boolToInt(entry.Verify), boolToInt(entry.RouterDisable), entry.Content,
		entry.ResponseSchema, entry.Started, entry.OccurredAt)
	return err
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

type cronDetailRunRequest struct {
	SourceType string `json:"sourceType"`
	DetailID   int    `json:"detailId"`
	MetaID     int    `json:"metaId"`
	ReuseChat  *bool  `json:"reuseChat"`
}

// handleCronDetailRun creates a pending detail from an existing detail or
// metadata record. Execution remains owned by cronExecuteOnce, so this handler
// must never mark the new task as started or call the model directly.
func handleCronDetailRun(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentID := strings.TrimSpace(r.URL.Query().Get("agentId"))
		if agentID == "" {
			writeCronDetailRunError(w, "agentId is required")
			return
		}
		if cronDB == nil {
			writeCronDetailRunError(w, "db not ready")
			return
		}
		var req cronDetailRunRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			writeCronDetailRunError(w, "invalid request: "+err.Error())
			return
		}
		if req.ReuseChat == nil {
			writeCronDetailRunError(w, "reuseChat is required")
			return
		}
		if err := validateIntegrationAgentExists(cfg.AgentDir, cfg.effectiveDeviceID(), agentID); err != nil {
			writeCronDetailRunError(w, err.Error())
			return
		}
		detail, err := createCronDetailRun(agentID, req, time.Now())
		if err != nil {
			writeCronDetailRunError(w, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "data": detail})
	}
}

func writeCronDetailRunError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": message})
}

func createCronDetailRun(agentID string, req cronDetailRunRequest, now time.Time) (cronDetailResult, error) {
	var empty cronDetailResult
	if cronDB == nil {
		return empty, fmt.Errorf("db not ready")
	}
	agentID = strings.TrimSpace(agentID)
	req.SourceType = strings.ToLower(strings.TrimSpace(req.SourceType))
	if req.ReuseChat == nil {
		return empty, fmt.Errorf("reuseChat is required")
	}
	if req.SourceType != "detail" && req.SourceType != "meta" {
		return empty, fmt.Errorf("sourceType must be detail or meta")
	}
	if (req.SourceType == "detail" && (req.DetailID <= 0 || req.MetaID != 0)) ||
		(req.SourceType == "meta" && (req.MetaID <= 0 || req.DetailID != 0)) {
		return empty, fmt.Errorf("source ID is invalid")
	}

	ensureCronSchema(cronDB)
	tx, err := cronDB.Begin()
	if err != nil {
		return empty, err
	}
	defer tx.Rollback()

	var source cronDetailResult
	if req.SourceType == "detail" {
		source, err = loadCronDetailRunSource(tx, req.DetailID, agentID)
		if err == nil && source.Started == 1 {
			err = fmt.Errorf("已启动的任务不能重跑")
		}
		if err == nil && source.Started != 0 && source.Started != 2 && source.Started != 3 && source.Started != cronDetailStartedFailed {
			err = fmt.Errorf("task detail cannot be run")
		}
	} else {
		source, err = loadCronMetaRunSource(tx, req.MetaID, agentID)
	}
	if err != nil {
		return empty, err
	}
	if err := validateCronDetailRunSource(source); err != nil {
		return empty, err
	}
	if !*req.ReuseChat {
		source.ChatID = ""
	}
	source.Type = sharedutil.NormalizeTaskType(source.Type)

	const maxExecTimeCollisionRetries = 60
	baseExecTime := now.Unix()
	var detailID int64
	for offset := int64(0); offset < maxExecTimeCollisionRetries; offset++ {
		source.ExecTime = baseExecTime + offset
		res, insertErr := tx.Exec(`INSERT INTO task_detail (meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, verify, router_disable, content, response_schema, started)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,0)`,
			source.MetaID, source.ExecTime, source.AgentID, source.ChatID, source.Type, source.Model,
			boolToInt(source.Thinking), boolToInt(source.Verify), boolToInt(source.RouterDisable), source.Content, source.ResponseSchema)
		if insertErr == nil {
			detailID, insertErr = res.LastInsertId()
			if insertErr != nil {
				return empty, fmt.Errorf("read created task detail: %w", insertErr)
			}
			break
		}
		if !isCronDetailExecTimeConflict(insertErr) {
			return empty, fmt.Errorf("create task detail: %w", insertErr)
		}
	}
	if detailID == 0 {
		return empty, fmt.Errorf("create task detail: execution time is busy")
	}
	source.ID = int(detailID)
	source.Started = 0
	source.ResultContent = ""
	source.RepliedAt = ""
	if err := appendCronDetailLogTx(tx, cronDetailLogEntry{
		DetailID:       source.ID,
		MetaID:         source.MetaID,
		AgentID:        source.AgentID,
		ChatID:         source.ChatID,
		TaskType:       source.Type,
		Action:         "run",
		ExecTime:       source.ExecTime,
		Model:          source.Model,
		Thinking:       source.Thinking,
		Verify:         source.Verify,
		RouterDisable:  source.RouterDisable,
		Content:        source.Content,
		ResponseSchema: source.ResponseSchema,
		Started:        0,
		OccurredAt:     cronLogTimestamp(),
	}); err != nil {
		return empty, fmt.Errorf("write task detail audit log: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return empty, err
	}
	return source, nil
}

func loadCronDetailRunSource(tx *sql.Tx, detailID int, agentID string) (cronDetailResult, error) {
	var source cronDetailResult
	var thinking, verify, routerDisable int
	err := tx.QueryRow(`SELECT d.id, d.meta_id, d.exec_time, d.agent_id, d.chat_id, d.task_type, d.model, d.thinking, d.verify, d.router_disable, d.content, d.response_schema, d.started,
		COALESCE(m.cycle, 0), COALESCE(m.raw_time, ''), COALESCE(m.cron, '')
		FROM task_detail d
		LEFT JOIN task_meta m ON m.id = d.meta_id
		WHERE d.id = ? AND d.agent_id = ?`, detailID, agentID).Scan(
		&source.ID, &source.MetaID, &source.ExecTime, &source.AgentID, &source.ChatID, &source.Type, &source.Model,
		&thinking, &verify, &routerDisable, &source.Content, &source.ResponseSchema, &source.Started,
		&source.Cycle, &source.RawTime, &source.Cron,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return cronDetailResult{}, fmt.Errorf("task detail or metadata not found")
	}
	if err != nil {
		return cronDetailResult{}, err
	}
	source.Thinking = thinking != 0
	source.Verify = verify != 0
	source.RouterDisable = routerDisable != 0
	return source, nil
}

func loadCronMetaRunSource(tx *sql.Tx, metaID int, agentID string) (cronDetailResult, error) {
	var source cronDetailResult
	var thinking, verify, routerDisable int
	err := tx.QueryRow(`SELECT id, cycle, raw_time, agent_id, chat_id, task_type, model, thinking, verify, router_disable, cron, content, response_schema
		FROM task_meta WHERE id = ? AND agent_id = ?`, metaID, agentID).Scan(
		&source.MetaID, &source.Cycle, &source.RawTime, &source.AgentID, &source.ChatID, &source.Type, &source.Model,
		&thinking, &verify, &routerDisable, &source.Cron, &source.Content, &source.ResponseSchema,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return cronDetailResult{}, fmt.Errorf("task metadata not found")
	}
	if err != nil {
		return cronDetailResult{}, err
	}
	source.Thinking = thinking != 0
	source.Verify = verify != 0
	source.RouterDisable = routerDisable != 0
	return source, nil
}

func validateCronDetailRunSource(source cronDetailResult) error {
	if source.MetaID <= 0 || strings.TrimSpace(source.AgentID) == "" || strings.TrimSpace(source.Model) == "" || strings.TrimSpace(source.Content) == "" {
		return fmt.Errorf("task source is incomplete")
	}
	return nil
}

func isCronDetailExecTimeConflict(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint failed") && strings.Contains(message, "task_detail.meta_id") && strings.Contains(message, "task_detail.exec_time")
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
		if current.Cycle != -1 {
			return empty, fmt.Errorf("cannot convert a built-in cycle to custom cron")
		}
		if _, err := validateCronCreateSchedule(-1, rawTime, cronExpr); err != nil {
			return empty, err
		}
		rawTime = ""
	} else {
		execAt, err := validateCronCreateSchedule(req.Cycle, rawTime, "")
		if err != nil {
			return empty, err
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

// sessionRecoveryCandidate is the small amount of information the site needs
// to let a user reopen a page session that is no longer in localStorage.
// The complete transcript remains in chat_log and is loaded through
// /api/restore only after the user explicitly selects a candidate.
type sessionRecoveryCandidate struct {
	AgentID     string `json:"agentId"`
	ChatID      string `json:"chatId"`
	LastPrompt  string `json:"lastPrompt"`
	CompletedAt string `json:"completedAt"`
}

const (
	sessionRecoveryCandidatePageSize = 10
	maxSessionRecoveryCandidatePage  = 100000
)

func normalizeSessionRecoveryCandidatePage(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 1, nil
	}
	page, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || page <= 0 || page > maxSessionRecoveryCandidatePage {
		return 0, fmt.Errorf("page must be between 1 and %d", maxSessionRecoveryCandidatePage)
	}
	return page, nil
}

func extractSessionRecoveryPrompt(raw string) string {
	var request map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &request); err != nil {
		return ""
	}
	messages, ok := request["messages"].([]interface{})
	if !ok {
		return ""
	}
	for i := len(messages) - 1; i >= 0; i-- {
		message, ok := messages[i].(map[string]interface{})
		if !ok || strings.TrimSpace(fmt.Sprint(message["role"])) != "user" {
			continue
		}
		switch content := message["content"].(type) {
		case string:
			return strings.TrimSpace(content)
		case []interface{}:
			parts := make([]string, 0, len(content))
			for _, item := range content {
				switch value := item.(type) {
				case string:
					parts = append(parts, value)
				case map[string]interface{}:
					if text, ok := value["text"].(string); ok {
						parts = append(parts, text)
					}
				}
			}
			return strings.TrimSpace(strings.Join(parts, ""))
		default:
			return strings.TrimSpace(fmt.Sprint(content))
		}
	}
	return ""
}

func normalizeSessionRecoveryExcludedChatIDs(values []string) []string {
	seen := make(map[string]struct{})
	chatIDs := make([]string, 0, len(values))
	for _, value := range values {
		chatID := strings.TrimSpace(value)
		if chatID == "" {
			continue
		}
		if _, exists := seen[chatID]; exists {
			continue
		}
		seen[chatID] = struct{}{}
		chatIDs = append(chatIDs, chatID)
	}
	return chatIDs
}

func querySessionRecoveryCandidates(page int, excludedChatIDs []string) ([]sessionRecoveryCandidate, bool, error) {
	if cronDB == nil {
		return nil, false, fmt.Errorf("db not ready")
	}
	page = max(page, 1)
	offset := (page - 1) * sessionRecoveryCandidatePageSize
	args := []interface{}{chatTypePageSession}
	query := `
		SELECT a.agent_id, a.chat_id, q.content, a.created_at
		FROM chat_log AS a
		JOIN chat_log AS q ON q.id = (
			SELECT previous.id
			FROM chat_log AS previous
			WHERE previous.chat_id = a.chat_id
			  AND previous.role = 'Q'
			  AND (previous.created_at < a.created_at OR (previous.created_at = a.created_at AND previous.id < a.id))
			ORDER BY previous.created_at DESC, previous.id DESC
			LIMIT 1
		)
		WHERE a.chat_type = ?
		  AND a.role = 'A'
		  AND a.response_type = 'normal'
		  AND instr(a.content, '[DONE]') > 0
		  AND a.id = (
			SELECT latest.id
			FROM chat_log AS latest
			WHERE latest.chat_id = a.chat_id
			  AND latest.chat_type = ?
			  AND latest.role = 'A'
			  AND latest.response_type = 'normal'
			  AND instr(latest.content, '[DONE]') > 0
			ORDER BY latest.created_at DESC, latest.id DESC
			LIMIT 1
		)`
	args = append(args, chatTypePageSession)
	excludedChatIDs = normalizeSessionRecoveryExcludedChatIDs(excludedChatIDs)
	if len(excludedChatIDs) > 0 {
		placeholders := make([]string, 0, len(excludedChatIDs))
		for _, chatID := range excludedChatIDs {
			placeholders = append(placeholders, "?")
			args = append(args, chatID)
		}
		query += ` AND a.chat_id NOT IN (` + strings.Join(placeholders, ",") + `)`
	}
	query += ` ORDER BY a.created_at DESC, a.id DESC LIMIT ? OFFSET ?`
	args = append(args, sessionRecoveryCandidatePageSize+1, offset)
	rows, err := cronDB.Query(query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	candidates := make([]sessionRecoveryCandidate, 0, sessionRecoveryCandidatePageSize+1)
	for rows.Next() {
		var candidate sessionRecoveryCandidate
		var rawPrompt string
		if err := rows.Scan(&candidate.AgentID, &candidate.ChatID, &rawPrompt, &candidate.CompletedAt); err != nil {
			return nil, false, err
		}
		candidate.ChatID = strings.TrimSpace(candidate.ChatID)
		candidate.LastPrompt = extractSessionRecoveryPrompt(rawPrompt)
		if candidate.ChatID == "" || candidate.LastPrompt == "" {
			continue
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(candidates) > sessionRecoveryCandidatePageSize
	if hasMore {
		candidates = candidates[:sessionRecoveryCandidatePageSize]
	}
	return candidates, hasMore, nil
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

func restoreHistoryQueryBase(agentID, chatID, beforeTimeline string, beforeID int) (string, []interface{}) {
	trimmedAgentID := strings.TrimSpace(agentID)
	trimmedChatID := strings.TrimSpace(chatID)
	trimmedBeforeTimeline := strings.TrimSpace(beforeTimeline)
	query := ` FROM chat_log WHERE chat_id = ?`
	args := []interface{}{trimmedChatID}
	if trimmedAgentID != "" {
		query = ` FROM chat_log WHERE agent_id = ? AND chat_id = ?`
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
	return query, args
}

func queryRestoreRecords(query string, args ...interface{}) ([]restoreRecord, error) {
	if cronDB == nil {
		return nil, fmt.Errorf("db not ready")
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

func queryRestoreHistoryRecords(agentID, chatID, beforeTimeline string, beforeID, limit int) ([]restoreRecord, *restoreHistoryMeta, error) {
	if cronDB == nil {
		return nil, nil, fmt.Errorf("db not ready")
	}
	if limit <= 0 {
		limit = 120
	}
	baseQuery, baseArgs := restoreHistoryQueryBase(agentID, chatID, beforeTimeline, beforeID)
	tailArgs := append(append([]interface{}{}, baseArgs...), limit)
	tail, err := queryRestoreRecords(
		`SELECT id, agent_id, chat_id, chat_type, role, response_type, content, created_at`+baseQuery+` ORDER BY created_at DESC, id DESC LIMIT ?`,
		tailArgs...,
	)
	if err != nil {
		return nil, nil, err
	}
	if len(tail) == 0 {
		return []restoreRecord{}, &restoreHistoryMeta{HasMore: false}, nil
	}

	oldestTail := tail[len(tail)-1]
	firstRecord := oldestTail
	if oldestTail.Role != "Q" {
		anchorArgs := append(append([]interface{}{}, baseArgs...), "Q", oldestTail.CreatedAt, oldestTail.CreatedAt, oldestTail.ID)
		anchor, err := queryRestoreRecords(
			`SELECT id, agent_id, chat_id, chat_type, role, response_type, content, created_at`+baseQuery+` AND role = ? AND (created_at < ? OR (created_at = ? AND id < ?)) ORDER BY created_at DESC, id DESC LIMIT 1`,
			anchorArgs...,
		)
		if err != nil {
			return nil, nil, err
		}
		if len(anchor) > 0 {
			firstRecord = anchor[0]
		}
	}

	recordArgs := append(append([]interface{}{}, baseArgs...), firstRecord.CreatedAt, firstRecord.CreatedAt, firstRecord.ID)
	records, err := queryRestoreRecords(
		`SELECT id, agent_id, chat_id, chat_type, role, response_type, content, created_at`+baseQuery+` AND (created_at > ? OR (created_at = ? AND id >= ?)) ORDER BY created_at, id`,
		recordArgs...,
	)
	if err != nil {
		return nil, nil, err
	}

	hasMoreArgs := append(append([]interface{}{}, baseArgs...), firstRecord.CreatedAt, firstRecord.CreatedAt, firstRecord.ID)
	var hasMore bool
	if err := cronDB.QueryRow(`SELECT EXISTS(SELECT 1`+baseQuery+` AND (created_at < ? OR (created_at = ? AND id < ?)) LIMIT 1)`, hasMoreArgs...).Scan(&hasMore); err != nil {
		return nil, nil, err
	}
	meta := &restoreHistoryMeta{HasMore: hasMore}
	if hasMore {
		meta.BeforeTimeline = firstRecord.CreatedAt
		meta.BeforeID = firstRecord.ID
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

func handleSessionRecoveryCandidates() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		page, err := normalizeSessionRecoveryCandidatePage(r.URL.Query().Get("page"))
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		candidates, hasMore, err := querySessionRecoveryCandidates(page, r.URL.Query()["exclude"])
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "query error: " + err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   0,
			"page":     page,
			"pageSize": sessionRecoveryCandidatePageSize,
			"hasMore":  hasMore,
			"sessions": candidates,
		})
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
	if cfg == nil {
		return "", fmt.Errorf("config is required")
	}
	return resolveAgentWorkspace(cfg.AgentDir, agentID)
}

// resolveAgentWorkspace resolves one direct child of the configured Agent root
// without building Agent metadata. Workspace lookup is used by the file browser
// and must not walk every Agent's skills directory.
func resolveAgentWorkspace(agentDir, agentID string) (string, error) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" || agentID == "." || agentID == ".." || filepath.Base(agentID) != agentID || strings.ContainsAny(agentID, `/\\`) {
		return "", fmt.Errorf("invalid agentId")
	}

	root, err := resolveIntegrationAgentDir(agentDir)
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve agent root: %w", err)
	}
	rootInfo, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("stat agent root: %w", err)
	}
	if !rootInfo.IsDir() {
		return "", fmt.Errorf("agent root is not a directory")
	}

	workspace, err := filepath.EvalSymlinks(filepath.Join(root, agentID))
	if err != nil {
		return "", fmt.Errorf("resolve agent workspace: %w", err)
	}
	workspaceInfo, err := os.Stat(workspace)
	if err != nil {
		return "", fmt.Errorf("stat agent workspace: %w", err)
	}
	if !workspaceInfo.IsDir() || !ensurePathWithinRoot(root, workspace) {
		return "", fmt.Errorf("agent workspace is invalid")
	}
	return workspace, nil
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

// validateCronCreateSchedule keeps the persisted scheduling fields in one
// canonical shape. A task is either a built-in cycle with a first execution
// time, or a custom cron expression; mixing the two produces ambiguous UI
// values such as "Cron" for both cycle and time.
func validateCronCreateSchedule(cycle int, rawTime, cronExpr string) (time.Time, error) {
	rawTime = strings.TrimSpace(rawTime)
	cronExpr = strings.TrimSpace(cronExpr)
	if cycle == -1 {
		if rawTime != "" {
			return time.Time{}, fmt.Errorf("custom cron must not include rawTime")
		}
		if cronExpr == "" {
			return time.Time{}, fmt.Errorf("custom cron expression is required")
		}
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		if _, err := parser.Parse(strings.ReplaceAll(cronExpr, "?", "*")); err != nil {
			return time.Time{}, fmt.Errorf("invalid custom cron expression: %w", err)
		}
		return time.Time{}, nil
	}
	if cycle < 0 || cycle > 5 {
		return time.Time{}, fmt.Errorf("cycle must be one of -1,0,1,2,3,4,5")
	}
	if cronExpr != "" {
		return time.Time{}, fmt.Errorf("built-in cycle must not include cron")
	}
	if rawTime == "" {
		return time.Time{}, fmt.Errorf("rawTime is required for built-in cycle")
	}
	execAt, err := time.ParseInLocation("2006-01-02 15:04", rawTime, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time: %w", err)
	}
	return execAt, nil
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
	execAt, err := validateCronCreateSchedule(req.Cycle, req.RawTime, req.Cron)
	if err != nil {
		return nil, err
	}
	if req.Cycle == -1 {
		cronExpr = req.Cron
		rawTime = ""
	} else {
		cronExpr = cronBuildExpr(req.Cycle, execAt)
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
	return "连接已中断，请重试"
}

func queueSSEStreamInterruptedLog(ch chan chatMsg, err error) bool {
	if ch == nil || err == nil {
		return false
	}
	log.Printf("SSE stream interrupted: %v", err)
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
		// `test` is reserved for the isolated /api/model/test forwarding
		// path. Ordinary /v1/chat/completions requests must never be able to
		// opt into the test-only execution behavior.
		"test",
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
		injectApplicationDataDirMetadata(metaMap, cfg.AgentDir)
		injectMCPMetadata(cfg, metadata, metaMap)
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
			injectApplicationDataDirMetadata(mm, cfg.AgentDir)
			if err := injectIntegrationConfigPathMetadata(mm, cfg); err != nil {
				log.Printf("proxy: integration config unavailable: %v", err)
				http.Error(w, "Integration configuration is unavailable", http.StatusInternalServerError)
				return
			}
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

		// The browser's SSE connection is only a live view of the request.  A
		// refresh cancels r.Context(), but the upstream stream must keep running
		// so /api/restore can replay the remaining persisted events.
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
		var active *activeConn
		if chatID != "" {
			ch = make(chan chatMsg, 64)
			active = &activeConn{cancel: cancel, ch: ch, agentID: chatAgentID, chatID: chatID}
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
			connMap[key] = active
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
		deviceID, _ := metaMap["deviceId"].(string)
		setDeviceIDHeader(proxyReq, deviceID)

		finishSSE := trackIntegrationSSE(cfg)
		defer finishSSE()
		resp, err := proxyClient.Do(proxyReq)
		if err != nil {
			releaseActiveConn(key, active)
			cancel()
			// A user cancellation can race the initial upstream connection.
			// The cancel handler has already stored the X marker, so avoid
			// appending a misleading "Failed to forward" error to the chat.
			if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
				return
			}
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
				if _, writeErr := w.Write(payload); writeErr != nil {
					abnormalStream = true
					if !sawDoneMarker && !sawAbnormalPacket {
						sawAbnormalPacket = queueSSEStreamInterruptedLog(ch, writeErr) || sawAbnormalPacket
					}
				}
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
			if knowledgeCommit {
				if err := updateKnowledgeManualTimestamps(time.Now(), chatAgentID); err != nil {
					log.Printf("proxy: knowledge manual update failed: %v", err)
				}
			}
			if !errors.Is(ctx.Err(), context.Canceled) {
				sendSSECompletionNotification(chatType, "", extractCompletionNotificationPrompt(reqData), abnormalStream)
			}
			releaseActiveConn(key, active)
			cancel()
			return
		}

		buf := make([]byte, 256)
		logBuf := make([]byte, 0, 1024)
		var streamErr error
		clientDisconnected := false
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				if !clientDisconnected {
					if _, writeErr := w.Write(buf[:n]); writeErr != nil {
						// The page may have refreshed. Keep consuming and logging the
						// upstream SSE so a new page can resume it through /api/restore.
						clientDisconnected = true
					} else {
						flusher.Flush()
					}
				}
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
		if knowledgeCommit {
			if err := updateKnowledgeManualTimestamps(time.Now(), chatAgentID); err != nil {
				log.Printf("proxy: knowledge manual update failed: %v", err)
			}
		}
		if !errors.Is(ctx.Err(), context.Canceled) {
			sendSSECompletionNotification(chatType, "", extractCompletionNotificationPrompt(reqData), abnormalStream)
		}

		// Only the handler that registered a connection can remove or close it.
		// A newer request for the same chat may already have replaced connMap[key].
		releaseActiveConn(key, active)
		cancel()
	}
}

func handleInstallApp() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		config := integrationConfiguredInstallAppConfig()
		if r.URL.Query().Get("details") == "1" {
			// The browser calls this endpoint on its configured scan interval.
			// Clear the availability cache so a just-installed app is observed on
			// the next scan instead of up to five minutes later.
			integrationResetInstallAppAvailabilityCache()
		}
		apps := integrationFilterMissingInstallApps(mergeInstallApps(detectRequiredInstallApps(), strings.Join(config.Apps, ",")))
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("details") == "1" {
			w.Header().Set("Cache-Control", "no-store")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"apps":     apps,
				"interval": config.Interval,
				"content":  config.Content,
			})
			return
		}
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
  <button id="launchRetry" class="retry" type="button">手动重试</button>
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

const defaultInstallAppIntervalMinutes = 60

type integrationInstallAppConfig struct {
	Apps     []string
	Interval int
	Content  string
}

func integrationInstallAppInterval(value interface{}) int {
	interval := defaultInstallAppIntervalMinutes
	switch typed := value.(type) {
	case float64:
		if typed >= 1 && math.Trunc(typed) == typed {
			interval = int(typed)
		}
	case int:
		if typed > 0 {
			interval = typed
		}
	case json.Number:
		if parsed, err := typed.Int64(); err == nil && parsed > 0 {
			interval = int(parsed)
		}
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(typed)); err == nil && parsed > 0 {
			interval = parsed
		}
	}
	return interval
}

func integrationInstallAppConfigFromValue(raw interface{}, platform string) integrationInstallAppConfig {
	config := integrationInstallAppConfig{
		Apps:     integrationInstallAppsFromConfigValue(raw, platform),
		Interval: defaultInstallAppIntervalMinutes,
		Content:  "请安装 $namelist",
	}
	value, ok := raw.(map[string]interface{})
	if !ok {
		return config
	}
	config.Interval = integrationInstallAppInterval(value["interval"])
	if content := strings.TrimSpace(fmt.Sprint(value["content"])); content != "" && content != "<nil>" {
		config.Content = content
	}
	return config
}

func integrationConfiguredInstallAppConfig() integrationInstallAppConfig {
	values, _, err := readIntegrationStartupConfigRaw()
	if err != nil || values == nil {
		return integrationInstallAppConfig{
			Interval: defaultInstallAppIntervalMinutes,
			Content:  "请安装 $namelist",
		}
	}
	return integrationInstallAppConfigFromValue(values["install_app"], integrationInstallAppPlatformKeyFn())
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

// releaseActiveConn closes a handler's own log channel and removes its map
// entry only when that handler still owns it. This prevents a finishing,
// cancelled request from closing the channel installed by a newer request for
// the same chat.
func releaseActiveConn(key string, ac *activeConn) {
	if ac == nil {
		return
	}
	connMu.Lock()
	if connMap[key] == ac {
		delete(connMap, key)
	}
	connMu.Unlock()
	closeActiveConn(ac)
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
	Port int
	Host string
	// StartupConfigPath is the verified absolute path of the static main
	// config/config.json used to start this Integration process.
	StartupConfigPath string
	// ClientVersion is read from the main config/config.json during startup and
	// remains stable for the lifetime of this Integration process.
	ClientVersion  string
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
	Reply                     string

	// cli-get specific
	SleepMs           int
	AwaitMs           int
	Check             int
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
	// ModelTaskConcurrence limits all local media/model tasks together. It is
	// loaded from config/config.json.modelTask.concurrence at service startup.
	ModelTaskConcurrence int
	// ModelTaskDownload controls the number of HTTP byte-range download parts
	// used for one runtime model artifact. It is loaded from
	// config/config.json.modelTask.download at service startup.
	ModelTaskDownload int
	// ModelTaskTimeout controls the no-progress read timeout in seconds for
	// each runtime-model range part and ordinary download. It is loaded from
	// config/config.json.modelTask.timeout at service startup.
	ModelTaskTimeout int
	// ModelTaskRetry controls retry attempts for each runtime-model download
	// range part or ordinary download. It is loaded from
	// config/config.json.modelTask.retry at service startup.
	ModelTaskRetry int
	// ScheduledTaskReadTimeout is the SSE body idle timeout used only by
	// scheduled memo execution. It is intentionally distinct from the cli-get
	// HTTP timeout fields above, which must not impose a total duration on a
	// long-running streamed response.
	ScheduledTaskReadTimeout time.Duration
	modelTaskQueue           *modelTaskQueue

	// Per-Agent MCP metadata refresh state.
	mcpStateMu sync.Mutex
	MCPState   *integrationMCPState

	// Cached @ menu Agent/Skill metadata. Initialized only after config.json.at
	// is validated during Integration service startup.
	AtMenuCache *integrationAtMenuCache

	// sleepManager owns the optional macOS caffeinate process for this service.
	sleepManager *integrationSleepManager
}

type integrationStartupOptions struct {
	Config  Config
	PIDFile string
	LogFile string
}

const (
	integrationServicePort          = 8080
	integrationDefaultPIDFile       = "integration.pid"
	integrationDefaultLogFile       = "integration.log"
	integrationLogRetentionDays     = 3
	integrationStopWait             = 15 * time.Second
	integrationShutdownTimeout      = 5 * time.Second
	integrationReadyPath            = "/api/heartbeat"
	defaultPluginExecTimeoutMs      = 600000
	defaultScheduledTaskReadTimeout = 120 * time.Second
	integrationSkipBrowserEnv       = "DEEPRIGHT_INTEGRATION_SKIP_BROWSER"
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

const integrationPersistentSettingsTable = "integration_persistent_settings"

// integrationPersistentHostDBOpenFn is replaceable by tests so host settings
// can be verified without touching the user's shared application database.
var integrationPersistentHostDBOpenFn = func() (*sql.DB, error) {
	appDir := strings.TrimSpace(integrationAppDir())
	if appDir == "" {
		return nil, errors.New("resolve integration settings directory failed")
	}
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return nil, err
	}
	return knowledgecore.OpenSharedDB(appDir)
}

func ensureIntegrationPersistentSettingsTable(db *sql.DB) error {
	if db == nil {
		return errors.New("integration settings database is not initialized")
	}
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS ` + integrationPersistentSettingsTable + ` (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`)
	return err
}

func readIntegrationPersistentHost() (string, bool, error) {
	db, err := integrationPersistentHostDBOpenFn()
	if err != nil {
		return "", false, fmt.Errorf("open integration settings: %w", err)
	}
	if err := ensureIntegrationPersistentSettingsTable(db); err != nil {
		return "", false, fmt.Errorf("initialize integration settings: %w", err)
	}

	var host string
	err = db.QueryRow(`SELECT value FROM `+integrationPersistentSettingsTable+` WHERE key = ?`, "host").Scan(&host)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read persisted service address: %w", err)
	}
	normalized, err := runtimehost.Validate(host)
	if err != nil {
		return "", false, fmt.Errorf("validate persisted service address: %w", err)
	}
	return normalized, true, nil
}

func writeIntegrationPersistentHost(host string) error {
	db, err := integrationPersistentHostDBOpenFn()
	if err != nil {
		return fmt.Errorf("open integration settings: %w", err)
	}
	if err := ensureIntegrationPersistentSettingsTable(db); err != nil {
		return fmt.Errorf("initialize integration settings: %w", err)
	}
	if _, err := db.Exec(`INSERT INTO `+integrationPersistentSettingsTable+`(key, value) VALUES(?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, "host", host); err != nil {
		return fmt.Errorf("save persisted service address: %w", err)
	}
	return nil
}

func deleteIntegrationPersistentHost() error {
	db, err := integrationPersistentHostDBOpenFn()
	if err != nil {
		return fmt.Errorf("open integration settings: %w", err)
	}
	if err := ensureIntegrationPersistentSettingsTable(db); err != nil {
		return fmt.Errorf("initialize integration settings: %w", err)
	}
	if _, err := db.Exec(`DELETE FROM `+integrationPersistentSettingsTable+` WHERE key = ?`, "host"); err != nil {
		return fmt.Errorf("delete persisted service address: %w", err)
	}
	return nil
}

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

func (cfg *Config) setPersistentHost(host string) (runtimehost.Snapshot, error) {
	if cfg == nil {
		return runtimehost.Snapshot{}, fmt.Errorf("service address is not initialized")
	}
	normalized, err := runtimehost.Validate(host)
	if err != nil {
		return runtimehost.Snapshot{}, err
	}
	if err := writeIntegrationPersistentHost(normalized); err != nil {
		return runtimehost.Snapshot{}, err
	}
	cfg.Host = normalized
	return cfg.ensureRuntimeHostState().Set(normalized)
}

func (cfg *Config) resetPersistentHost() (runtimehost.Snapshot, error) {
	if cfg == nil {
		return runtimehost.Snapshot{}, fmt.Errorf("service address is not initialized")
	}
	host, err := integrationConfiguredDefaultHost()
	if err != nil {
		return runtimehost.Snapshot{}, err
	}
	if err := deleteIntegrationPersistentHost(); err != nil {
		return runtimehost.Snapshot{}, err
	}
	cfg.Host = host
	return cfg.ensureRuntimeHostState().Set(host)
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
			Reply:                     "",
			SleepMs:                   3000,
			AwaitMs:                   30000,
			Check:                     10,
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
			ModelTaskConcurrence:      defaultModelTaskConcurrence,
			ModelTaskDownload:         defaultModelTaskDownload,
			ModelTaskTimeout:          defaultModelTaskTimeout,
			ModelTaskRetry:            defaultModelTaskRetry,
			ScheduledTaskReadTimeout:  defaultScheduledTaskReadTimeout,
		},
		PIDFile: integrationDefaultPIDFile,
		LogFile: integrationDefaultLogFile,
	}
}

var integrationStartupConfigKeys = map[string]string{
	"port":                    "port",
	"host":                    "host",
	"version":                 "version",
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
	"reply":                   "reply",
	"get":                     "get",
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

// integrationBundledStartupConfigPath identifies the static configuration
// shipped with the application. In a macOS app bundle it is a signed resource
// and must never be changed after distribution.
func integrationBundledStartupConfigPath() string {
	resourcesDir := strings.TrimSpace(integrationResourcesDir())
	if resourcesDir == "" {
		return ""
	}
	return filepath.Join(resourcesDir, "config", "config.json")
}

// integrationStartupConfigPath is always the shipped static configuration.
// User-selected settings are kept separately in the shared SQLite database so
// a newer application package can supply new configuration fields on upgrade.
func integrationStartupConfigPath() string {
	return integrationBundledStartupConfigPath()
}

func validateIntegrationStartupConfigFile(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("config/config.json path is unavailable")
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve absolute config path: %w", err)
	}
	absolutePath = filepath.Clean(absolutePath)
	info, err := os.Stat(absolutePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return absolutePath, fmt.Errorf("config/config.json does not exist")
		}
		return absolutePath, fmt.Errorf("stat config/config.json: %w", err)
	}
	if !info.Mode().IsRegular() {
		return absolutePath, fmt.Errorf("config/config.json is not a regular file")
	}
	data, err := os.ReadFile(absolutePath)
	if err != nil {
		return absolutePath, fmt.Errorf("read config/config.json: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var raw map[string]interface{}
	if err := decoder.Decode(&raw); err != nil {
		return absolutePath, fmt.Errorf("parse config/config.json: %w", err)
	}
	return absolutePath, nil
}

func validateCurrentIntegrationStartupConfigFile() (string, error) {
	return validateIntegrationStartupConfigFile(integrationStartupConfigPath())
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

// readScheduledTaskReadTimeout reads the only timeout that applies to the
// scheduled memo SSE body. Keep it outside the flattened HTTP configuration:
// the existing HTTP fields configure other clients and must not turn this
// stream's idle timeout into a request-wide deadline.
func readScheduledTaskReadTimeout() time.Duration {
	raw, path, err := readIntegrationStartupConfigRaw()
	if err != nil {
		log.Printf("[cron] invalid config/config.json http.timeout.read: read %s failed: %v; using default %s", path, err, defaultScheduledTaskReadTimeout)
		return defaultScheduledTaskReadTimeout
	}

	invalid := func(reason string) time.Duration {
		if strings.TrimSpace(path) == "" {
			path = "config/config.json"
		}
		log.Printf("[cron] invalid config/config.json http.timeout.read in %s: %s; using default %s", path, reason, defaultScheduledTaskReadTimeout)
		return defaultScheduledTaskReadTimeout
	}
	httpRaw, ok := raw["http"]
	if !ok {
		return invalid("value is missing")
	}
	httpValues, ok := httpRaw.(map[string]interface{})
	if !ok || httpValues == nil {
		return invalid("http must be an object")
	}
	timeoutRaw, ok := httpValues["timeout"]
	if !ok {
		return invalid("timeout is missing")
	}
	timeoutValues, ok := timeoutRaw.(map[string]interface{})
	if !ok || timeoutValues == nil {
		return invalid("timeout must be an object")
	}
	readRaw, ok := timeoutValues["read"]
	if !ok {
		return invalid("value is missing")
	}
	secondsNumber, ok := readRaw.(json.Number)
	if !ok {
		return invalid("must be a positive integer in seconds")
	}
	seconds, err := secondsNumber.Int64()
	if err != nil || seconds <= 0 {
		return invalid("must be a positive integer in seconds")
	}
	if seconds > int64(^uint64(0)>>1)/int64(time.Second) {
		return invalid("exceeds time.Duration range")
	}
	return time.Duration(seconds) * time.Second
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

func applyIntegrationHeartbeatConfig(raw interface{}, cfg *Config) error {
	if cfg == nil {
		return nil
	}
	values, ok := raw.(map[string]interface{})
	if !ok || values == nil {
		return fmt.Errorf("get must be an object")
	}
	if value, ok := values["sleep"]; ok {
		if err := assignIntegrationHeartbeatMilliseconds(value, "get.sleep", &cfg.SleepMs); err != nil {
			return err
		}
	}
	if value, ok := values["await"]; ok {
		if err := assignIntegrationHeartbeatMilliseconds(value, "get.await", &cfg.AwaitMs); err != nil {
			return err
		}
	}
	if value, ok := values["check"]; ok {
		if err := assignIntegrationHeartbeatPositiveInteger(value, "get.check", &cfg.Check); err != nil {
			return err
		}
	}
	if cfg.SleepMs < 0 {
		return fmt.Errorf("get.sleep must be >= 0")
	}
	if cfg.AwaitMs < 0 {
		return fmt.Errorf("get.await must be >= 0")
	}
	return nil
}

func assignIntegrationHeartbeatMilliseconds(raw interface{}, name string, target *int) error {
	value, ok := raw.(json.Number)
	if !ok {
		return fmt.Errorf("%s must be a non-negative integer", name)
	}
	milliseconds, err := value.Int64()
	if err != nil || milliseconds < 0 || milliseconds > int64(^uint(0)>>1) {
		return fmt.Errorf("%s must be a non-negative integer", name)
	}
	*target = int(milliseconds)
	return nil
}

func assignIntegrationHeartbeatPositiveInteger(raw interface{}, name string, target *int) error {
	if err := assignIntegrationHeartbeatMilliseconds(raw, name, target); err != nil {
		return err
	}
	if *target <= 0 {
		return fmt.Errorf("%s must be a positive integer", name)
	}
	return nil
}

func applyIntegrationStartupConfig(opts *integrationStartupOptions, values map[string]interface{}) error {
	if opts == nil || values == nil {
		return nil
	}
	for key, raw := range values {
		switch key {
		case "version":
			if value, ok := raw.(string); ok {
				opts.Config.ClientVersion = strings.TrimSpace(value)
			}
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
	// The nested get block is the current heartbeat configuration. Apply it
	// last so its values consistently take precedence over legacy flat keys.
	if raw, ok := values["get"]; ok {
		if err := applyIntegrationHeartbeatConfig(raw, &opts.Config); err != nil {
			return err
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

// integrationConfiguredDefaultHost returns the host from the static packaged
// configuration without applying a user's SQLite override.
func integrationConfiguredDefaultHost() (string, error) {
	opts := defaultIntegrationStartupOptions()
	values, _, err := readIntegrationStartupConfig()
	if err != nil {
		return "", err
	}
	if err := applyIntegrationStartupConfig(&opts, values); err != nil {
		return "", err
	}
	host := strings.TrimSpace(opts.Config.Host)
	if host == "" {
		host = defaultUpstreamHost
	}
	normalized, err := runtimehost.Validate(host)
	if err != nil {
		return "", fmt.Errorf("validate configured service address: %w", err)
	}
	return normalized, nil
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
	// This has a fallback-and-log contract instead of making a malformed
	// optional timeout prevent Integration from starting.
	opts.Config.ScheduledTaskReadTimeout = readScheduledTaskReadTimeout()
	if host, found, err := readIntegrationPersistentHost(); err != nil {
		return opts, path, err
	} else if found {
		opts.Config.Host = host
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
	fmt.Println("  integration api whisper <check|list|create|cancel|restart|delete|log> [options]")
	fmt.Println("  integration api rembg <check|list|create|cancel|restart|delete|log> [options]")
	fmt.Println("  integration api voxcpm <check|list|create|cancel|restart|delete|log> [options]")
	fmt.Println("  integration api wav2lip <check|list|create|cancel|restart|delete|log> [options]")
	fmt.Println("  integration api rvm <check|list|create|cancel|restart|delete|log> [options]")
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
	fmt.Println("  host              Read or update the persisted service address of a running integration service")
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
	fmt.Println("  api                Call integration HTTP APIs; use `integration api --help`")
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
	fmt.Println("  integration api whisper --help")
	fmt.Println("  integration api rembg --help")
	fmt.Println("  integration api voxcpm --help")
	fmt.Println("  integration api wav2lip --help")
	fmt.Println("  integration api rvm --help")
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
	fmt.Println("  get               Print the current service address")
	fmt.Println("  set               Update and persist the service address")
	fmt.Println("  reset             Restore and persist the default service address")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --value URL       New upstream host URL, required by set")
	fmt.Println("  --addr URL        Integration HTTP base address, default http://127.0.0.1:<runtime port>")
	fmt.Println("  --port INT        Integration HTTP 端口；仅支持 8080")
	fmt.Println("")
	fmt.Println("Notes:")
	fmt.Println("  该命令通过本机 integration 的 /api/host 接口工作，并持久化写入 config/config.json.host。")
	fmt.Println("  reset 会恢复默认服务地址 https://www.deepright.cn。")
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
	value := fs.String("value", "", "service address URL")
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

	out, marshalErr := json.MarshalIndent(map[string]any{"host": snapshot.Host}, "", "  ")
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

func handlePluginsStart(cfg *Config) http.HandlerFunc {
	return handlePluginAction(cfg, "start")
}

func handlePluginsStop(cfg *Config) http.HandlerFunc {
	return handlePluginAction(cfg, "stop")
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

func handlePluginAction(cfg *Config, action string) http.HandlerFunc {
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
		refreshIntegrationAtMenuCacheAfterPluginAction(cfg, action, name)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": 0, "data": buildPluginActionResponse(result)})
	}
}

// refreshIntegrationAtMenuCacheAfterPluginAction makes plugin-backed skills
// visible (or hidden) in the @ menu as soon as the plugin lifecycle call
// succeeds, rather than waiting for the regular snapshot refresh interval.
func refreshIntegrationAtMenuCacheAfterPluginAction(cfg *Config, action, plugin string) {
	if cfg == nil || cfg.AtMenuCache == nil {
		return
	}
	if err := cfg.AtMenuCache.refreshSnapshotAfterMutation(); err != nil {
		log.Printf("plugin %s refresh @ cache failed: plugin=%s err=%v", action, plugin, err)
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
	configPath, err := validateCurrentIntegrationStartupConfigFile()
	if err != nil {
		if strings.TrimSpace(configPath) != "" {
			fmt.Fprintf(stderr, "%s: %v\n", configPath, err)
		} else {
			fmt.Fprintf(stderr, "%v\n", err)
		}
		return 1
	}
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
	cfg.StartupConfigPath = configPath
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
	defer func() {
		if err := cleanupIntegrationBrowserReuseState(); err != nil {
			log.Printf("integration: clear browser reuse state failed: %v", err)
		}
	}()

	reportStartupFailure := func(format string, args ...interface{}) int {
		message := fmt.Sprintf(format, args...)
		log.Printf("integration startup failed: %s", message)
		if err := writeIntegrationStartupStatus(startupStatusFile, integrationStartupStatus{
			Status:  "failed",
			Message: message,
		}); err != nil {
			fmt.Fprintf(stderr, "startup status write failed: %v\n", err)
		}
		fmt.Fprintln(stderr, message)
		return 1
	}
	caffeinateInterval, _, err := readIntegrationCaffeinateInterval()
	if err != nil {
		return reportStartupFailure("error: %v", err)
	}
	if err := integrationEnsureGitIdentityConfiguredFn(); err != nil {
		log.Printf("integration: git identity auto-config skipped: %v", err)
	}
	if err := validateIntegrationServicePort(cfg.Port); err != nil {
		return reportStartupFailure("error: %v", err)
	}
	atMenuConfig, err := readIntegrationAtMenuConfig()
	if err != nil {
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
	if err := initializeIntegrationAtMenuCache(&cfg, atMenuConfig); err != nil {
		return reportStartupFailure("error: initialize @ cache: %v", err)
	}

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
	mux.HandleFunc("/api/browser/open", handleBrowserOpen())
	mux.HandleFunc("/api/skills", handleSkills(&cfg))
	mux.HandleFunc("/api/skill_state", handleSkillState(&cfg))
	mux.HandleFunc("/skills_warning", handleSkillsWarning(&cfg))
	mux.HandleFunc("/mcp_warning", handleMCPWarning(&cfg))
	mux.HandleFunc("/api/files", handleFiles())
	mux.HandleFunc("/api/backup/status", handleBackupStatus(&cfg))
	mux.HandleFunc("/api/data", handleData())
	mux.HandleFunc("/api/workspace", handleWorkspace(&cfg))
	mux.HandleFunc("/api/url_preview_probe", handleURLPreviewProbe(proxyClient))
	mux.HandleFunc("/api/edit", handleEdit(&cfg))
	mux.HandleFunc("/api/del", handleDel(&cfg))
	mux.HandleFunc("/api/raw", handleRaw(&cfg))
	mux.HandleFunc("/api/media_preview", handleMediaPreview(&cfg))
	mux.HandleFunc("/api/ffmpeg/check", handleFFmpegCheck())
	mux.HandleFunc("/api/bubblewrap/check", handleBubblewrapCheck())
	mux.HandleFunc("/api/whisper/check", handleWhisperCheck())
	mux.HandleFunc("/api/whisper/tasks", handleWhisperTasks())
	mux.HandleFunc("/api/whisper/tasks/cancel", handleWhisperTaskCancel())
	mux.HandleFunc("/api/whisper/tasks/restart", handleWhisperTaskRestart())
	mux.HandleFunc("/api/whisper/tasks/delete", handleWhisperTaskDelete())
	mux.HandleFunc("/api/whisper/tasks/log", handleWhisperTaskLog())
	mux.HandleFunc("/api/rembg/check", handleRembgCheck())
	mux.HandleFunc("/api/rembg/tasks", handleRembgTasks())
	mux.HandleFunc("/api/rembg/tasks/cancel", handleRembgTaskCancel())
	mux.HandleFunc("/api/rembg/tasks/restart", handleRembgTaskRestart())
	mux.HandleFunc("/api/rembg/tasks/delete", handleRembgTaskDelete())
	mux.HandleFunc("/api/rembg/tasks/log", handleRembgTaskLog())
	mux.HandleFunc("/api/rvm/check", handleRVMCheck(&cfg))
	mux.HandleFunc("/api/rvm/tasks", handleRVMTasks())
	mux.HandleFunc("/api/rvm/tasks/cancel", handleRVMTaskCancel())
	mux.HandleFunc("/api/rvm/tasks/restart", handleRVMTaskRestart())
	mux.HandleFunc("/api/rvm/tasks/delete", handleRVMTaskDelete())
	mux.HandleFunc("/api/rvm/tasks/log", handleRVMTaskLog())
	mux.HandleFunc("/api/wav2lip/check", handleWav2LipCheck(&cfg))
	mux.HandleFunc("/api/wav2lip/tasks", handleWav2LipTasks())
	mux.HandleFunc("/api/wav2lip/tasks/cancel", handleWav2LipTaskCancel())
	mux.HandleFunc("/api/wav2lip/tasks/restart", handleWav2LipTaskRestart())
	mux.HandleFunc("/api/wav2lip/tasks/delete", handleWav2LipTaskDelete())
	mux.HandleFunc("/api/wav2lip/tasks/log", handleWav2LipTaskLog())
	mux.HandleFunc("/api/voxcpm/check", handleVoxCPMCheck())
	mux.HandleFunc("/api/voxcpm/tasks", handleVoxCPMTasks())
	mux.HandleFunc("/api/voxcpm/tasks/cancel", handleVoxCPMTaskCancel())
	mux.HandleFunc("/api/voxcpm/tasks/restart", handleVoxCPMTaskRestart())
	mux.HandleFunc("/api/voxcpm/tasks/delete", handleVoxCPMTaskDelete())
	mux.HandleFunc("/api/voxcpm/tasks/log", handleVoxCPMTaskLog())
	mux.HandleFunc("/api/task_help", handleTaskHelp(&cfg))
	mux.HandleFunc("/api/video_trim", handleVideoTrim(&cfg))
	mux.HandleFunc("/api/video_audio_extract_to_audio", handleVideoAudioExtractToAudio(&cfg))
	mux.HandleFunc("/api/video_audio_edit", handleVideoAudioEdit(&cfg))
	mux.HandleFunc("/api/video_audio_edit_preview", handleVideoAudioEditPreview(&cfg))
	mux.HandleFunc("/api/video_record_convert", handleVideoRecordConvert(&cfg))
	mux.HandleFunc("/api/agent/audio", handleAgentAudioSave(&cfg))
	mux.HandleFunc("/api/audio_mix", handleAudioMix(&cfg))
	mux.HandleFunc("/api/audio_mix_preview", handleAudioMixPreview(&cfg))
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
	mux.HandleFunc("/api/model/test", handleModelTest(&cfg, proxyClient))
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
	mux.HandleFunc("/api/cron/detail/run", handleCronDetailRun(&cfg))
	mux.HandleFunc("/api/cron/meta/update", handleCronMetaUpdate(&cfg))
	mux.HandleFunc("/api/cron/detail/status", handleCronDetailStatus())
	mux.HandleFunc("/api/cancel", handleCancel(&cfg))
	mux.HandleFunc("/api/restore", handleRestore())
	mux.HandleFunc("/api/session_recovery_candidates", handleSessionRecoveryCandidates())
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
	startIntegrationDependencyChecks(ctx, &cfg)
	defer func() {
		if removed, err := cleanupStartupPIDFiles(pidFile); err != nil {
			log.Printf("serve cleanup startup pid files failed: %v", err)
		} else if len(removed) > 0 {
			log.Printf("serve cleanup removed pid files: %s", strings.Join(removed, ", "))
		}
	}()

	startCliGet(ctx, &cfg)
	startIntegrationDeviceConfigRefresh(ctx, cfg.DeviceState)
	startIntegrationAtMenuRefresh(ctx, &cfg)
	initCronDB()
	configureModelTaskQueue(&cfg)
	startWhisperTaskManager(ctx, &cfg)
	startRembgTaskManager(ctx, &cfg)
	startRVMTaskManager(ctx, &cfg)
	startWav2LipTaskManager(ctx, &cfg)
	startVoxCPMTaskManager(ctx, &cfg)
	sleepManager := newIntegrationSleepManager(caffeinateInterval)
	cfg.sleepManager = sleepManager
	sleepManager.Start(ctx)
	defer sleepManager.Close()
	startIntegrationLogRetentionCleanup(ctx)
	startIntegrationTempCleanup(ctx, cfg.AgentDir)
	startIntegrationAgentBackups(ctx, cfg.AgentDir)
	startIntegrationMiniappDocumentRecovery(ctx, cfg.AgentDir, cfg.DefaultDir)
	startCronCheck(ctx, &cfg)
	startIntegrationMCPRefresh(ctx, &cfg)
	startCronExecutor(ctx, &cfg, proxyClient, connectSvc)
	startConnectPendingRequestSync(ctx, &cfg, connectSvc)
	startIntegrationAssociatedPluginsAsync(associatedStartCtx, associatedPlugins, nil, associatedStartDone)

	go func() {
		<-ctx.Done()
		sleepManager.Close()
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

	sleepManager.Close()
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
	if configPath, err := validateCurrentIntegrationStartupConfigFile(); err != nil {
		if strings.TrimSpace(configPath) != "" {
			fmt.Fprintf(stderr, "%s: %v\n", configPath, err)
		} else {
			fmt.Fprintln(stderr, err)
		}
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

// integrationMiniappRecoveryConfig controls the periodic restoration of the
// three static documents which are copied into every Agent's app directory.
// The interval deliberately has no fallback: an operator must opt in through
// config/config.json.miniapp.recover.
type integrationMiniappRecoveryConfig struct {
	ScanInterval time.Duration
}

type integrationMiniappDocumentSource struct {
	Content []byte
	Mode    fs.FileMode
	Path    string
}

type integrationMiniappRecoveryEvent struct {
	Action   string
	Agent    string
	Document string
	Source   string
	Target   string
	Reason   string
}

type integrationMiniappRecoverySummary struct {
	AgentCount int
	Events     []integrationMiniappRecoveryEvent
}

var integrationMiniappProtectedDocuments = []string{"API.md", "CANVAS.md", "DESIGN.md"}

func readIntegrationMiniappRecoveryConfig() (integrationMiniappRecoveryConfig, error) {
	raw, _, err := readIntegrationStartupConfigRaw()
	if err != nil {
		return integrationMiniappRecoveryConfig{}, fmt.Errorf("read config/config.json.miniapp: %w", err)
	}
	miniappRaw, ok := raw["miniapp"]
	if !ok {
		return integrationMiniappRecoveryConfig{}, fmt.Errorf("config/config.json.miniapp is required")
	}
	miniapp, ok := miniappRaw.(map[string]interface{})
	if !ok || miniapp == nil {
		return integrationMiniappRecoveryConfig{}, fmt.Errorf("config/config.json.miniapp must be an object")
	}
	minutes, err := integrationMiniappPositiveMinutes(miniapp["recover"], "config/config.json.miniapp.recover")
	if err != nil {
		return integrationMiniappRecoveryConfig{}, err
	}
	return integrationMiniappRecoveryConfig{ScanInterval: time.Duration(minutes) * time.Minute}, nil
}

func integrationMiniappPositiveMinutes(raw interface{}, label string) (int64, error) {
	value, ok := raw.(json.Number)
	if !ok {
		return 0, fmt.Errorf("%s must be a positive integer in minutes", label)
	}
	minutes, err := value.Int64()
	if err != nil || minutes <= 0 || minutes > int64((1<<63-1)/int64(time.Minute)) {
		return 0, fmt.Errorf("%s must be a positive integer in minutes", label)
	}
	return minutes, nil
}

func readIntegrationMiniappDocumentSources(defaultDir string) (map[string]integrationMiniappDocumentSource, error) {
	resolvedDefaultDir, err := resolveServeDefaultDir(defaultDir)
	if err != nil {
		return nil, err
	}
	sourceDir := filepath.Join(resolvedDefaultDir, "app")
	sources := make(map[string]integrationMiniappDocumentSource, len(integrationMiniappProtectedDocuments))
	for _, document := range integrationMiniappProtectedDocuments {
		sourcePath := filepath.Join(sourceDir, document)
		info, err := os.Stat(sourcePath)
		if err != nil {
			return nil, fmt.Errorf("read miniapp source %s: %w", sourcePath, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("miniapp source is not a regular file: %s", sourcePath)
		}
		content, err := os.ReadFile(sourcePath)
		if err != nil {
			return nil, fmt.Errorf("read miniapp source %s: %w", sourcePath, err)
		}
		sources[document] = integrationMiniappDocumentSource{
			Content: content,
			Mode:    info.Mode().Perm(),
			Path:    sourcePath,
		}
	}
	return sources, nil
}

func integrationMiniappDocumentNeedsRecovery(target string, source integrationMiniappDocumentSource) (bool, error) {
	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != source.Mode {
		return true, nil
	}
	content, err := os.ReadFile(target)
	if err != nil {
		return false, err
	}
	return !bytes.Equal(content, source.Content), nil
}

func restoreIntegrationMiniappDocument(workspace, document string, source integrationMiniappDocumentSource) error {
	appDir := filepath.Join(workspace, "app")
	if !ensureWritablePathWithinRoot(workspace, appDir) {
		return fmt.Errorf("app directory escapes agent workspace")
	}
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return fmt.Errorf("create app directory: %w", err)
	}
	target := filepath.Join(appDir, document)
	if !ensurePathWithinRoot(workspace, target) {
		return fmt.Errorf("target escapes agent workspace")
	}
	if _, err := os.Lstat(target); err == nil {
		// Do not recursively delete a non-empty directory placed at a protected
		// document path.  It may contain unrelated user data; report the single
		// document recovery failure instead.
		if err := os.Remove(target); err != nil {
			return fmt.Errorf("replace protected document: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect protected document: %w", err)
	}
	if !ensureWritablePathWithinRoot(workspace, target) {
		return fmt.Errorf("target escapes agent workspace")
	}
	if err := os.WriteFile(target, source.Content, source.Mode); err != nil {
		return fmt.Errorf("write protected document: %w", err)
	}
	if err := os.Chmod(target, source.Mode); err != nil {
		return fmt.Errorf("set protected document permissions: %w", err)
	}
	return nil
}

func recoverIntegrationMiniappDocuments(agentRoot, defaultDir string) (integrationMiniappRecoverySummary, error) {
	sources, err := readIntegrationMiniappDocumentSources(defaultDir)
	if err != nil {
		return integrationMiniappRecoverySummary{}, err
	}
	entries, err := os.ReadDir(agentRoot)
	if err != nil {
		return integrationMiniappRecoverySummary{}, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	summary := integrationMiniappRecoverySummary{}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		agentID := strings.TrimSpace(entry.Name())
		if err := integrationagentarchive.ValidateAgentID(agentID); err != nil {
			continue
		}
		summary.AgentCount++
		workspace := filepath.Join(agentRoot, entry.Name())
		appDir := filepath.Join(workspace, "app")
		// Validate the app directory before comparing individual documents.  If
		// it escapes through a symlink, an identical external file must not turn
		// the scan into a silent no-op.
		if !ensureWritablePathWithinRoot(workspace, appDir) {
			for _, document := range integrationMiniappProtectedDocuments {
				source := sources[document]
				summary.Events = append(summary.Events, integrationMiniappRecoveryEvent{
					Action: "error", Agent: agentID, Document: document, Source: source.Path,
					Target: filepath.Join(appDir, document), Reason: "app directory escapes agent workspace",
				})
			}
			continue
		}
		for _, document := range integrationMiniappProtectedDocuments {
			source := sources[document]
			target := filepath.Join(workspace, "app", document)
			needsRecovery, err := integrationMiniappDocumentNeedsRecovery(target, source)
			if err != nil {
				summary.Events = append(summary.Events, integrationMiniappRecoveryEvent{
					Action: "error", Agent: agentID, Document: document, Source: source.Path, Target: target, Reason: err.Error(),
				})
				continue
			}
			if !needsRecovery {
				continue
			}
			if err := restoreIntegrationMiniappDocument(workspace, document, source); err != nil {
				summary.Events = append(summary.Events, integrationMiniappRecoveryEvent{
					Action: "error", Agent: agentID, Document: document, Source: source.Path, Target: target, Reason: err.Error(),
				})
				continue
			}
			summary.Events = append(summary.Events, integrationMiniappRecoveryEvent{
				Action: "restore", Agent: agentID, Document: document, Source: source.Path, Target: target,
			})
		}
	}
	return summary, nil
}

// startIntegrationMiniappDocumentRecovery performs an immediate scan and then
// restores only changed protected miniapp documents at the configured cadence.
func startIntegrationMiniappDocumentRecovery(ctx context.Context, agentRoot, defaultDir string) {
	if ctx == nil || strings.TrimSpace(agentRoot) == "" {
		return
	}
	go func() {
		config, err := readIntegrationMiniappRecoveryConfig()
		if err != nil {
			log.Printf("[miniapp] document recovery disabled: %v", err)
			return
		}
		run := func() {
			summary, err := recoverIntegrationMiniappDocuments(agentRoot, defaultDir)
			if err != nil {
				log.Printf("[miniapp] document recovery scan failed: %v", err)
				return
			}
			for _, event := range summary.Events {
				if event.Action == "restore" {
					log.Printf("[miniapp] restored agent=%s document=%s source=%s target=%s", event.Agent, event.Document, event.Source, event.Target)
				} else {
					log.Printf("[miniapp] recovery failed agent=%s document=%s source=%s target=%s reason=%s", event.Agent, event.Document, event.Source, event.Target, event.Reason)
				}
			}
		}
		run()

		ticker := time.NewTicker(config.ScanInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}

// integrationTempCleanupConfig is read from config/config.json once for each
// Integration process. It intentionally has no source-code defaults: operators
// must set all three values in the runtime configuration file.
type integrationTempCleanupConfig struct {
	ClearAfter   time.Duration
	PackAfter    time.Duration
	ScanInterval time.Duration
}

type integrationTempCleanupEvent struct {
	Action    string
	Agent     string
	Directory string
	Source    string
	Target    string
	Reason    string
}

type integrationTempCleanupSummary struct {
	AgentCount int
	Events     []integrationTempCleanupEvent
}

// integrationTempCleanupDirectories are agent-owned output directories that
// share the configured temp retention policy. Each directory archives into
// its own bak/ child, so files from different media types never mingle.
var integrationTempCleanupDirectories = []string{"tmp", "images", "videos", "audios", "canvas"}

func readIntegrationTempCleanupConfig() (integrationTempCleanupConfig, error) {
	raw, _, err := readIntegrationStartupConfigRaw()
	if err != nil {
		return integrationTempCleanupConfig{}, fmt.Errorf("read config/config.json.temp: %w", err)
	}
	tempRaw, ok := raw["temp"]
	if !ok {
		return integrationTempCleanupConfig{}, fmt.Errorf("config/config.json.temp is required")
	}
	temp, ok := tempRaw.(map[string]interface{})
	if !ok || temp == nil {
		return integrationTempCleanupConfig{}, fmt.Errorf("config/config.json.temp must be an object")
	}
	clearHours, err := integrationTempPositiveHours(temp["clear"], "config/config.json.temp.clear")
	if err != nil {
		return integrationTempCleanupConfig{}, err
	}
	packHours, err := integrationTempPositiveHours(temp["pack"], "config/config.json.temp.pack")
	if err != nil {
		return integrationTempCleanupConfig{}, err
	}
	scanHours, err := integrationTempPositiveHours(temp["scan"], "config/config.json.temp.scan")
	if err != nil {
		return integrationTempCleanupConfig{}, err
	}
	return integrationTempCleanupConfig{
		ClearAfter:   time.Duration(clearHours) * time.Hour,
		PackAfter:    time.Duration(packHours) * time.Hour,
		ScanInterval: time.Duration(scanHours) * time.Hour,
	}, nil
}

func integrationTempPositiveHours(raw interface{}, label string) (int64, error) {
	value, ok := raw.(json.Number)
	if !ok {
		return 0, fmt.Errorf("%s must be a positive integer in hours", label)
	}
	hours, err := value.Int64()
	if err != nil || hours <= 0 || hours > int64((1<<63-1)/int64(time.Hour)) {
		return 0, fmt.Errorf("%s must be a positive integer in hours", label)
	}
	return hours, nil
}

// startIntegrationTempCleanup schedules cleanup in a separate goroutine so
// filesystem work never delays service startup or HTTP request handling.
func startIntegrationTempCleanup(ctx context.Context, agentRoot string) {
	if ctx == nil || strings.TrimSpace(agentRoot) == "" {
		return
	}
	go func() {
		config, err := readIntegrationTempCleanupConfig()
		if err != nil {
			log.Printf("[temp] cleanup disabled: %v", err)
			return
		}
		run := func() {
			summary, err := cleanupIntegrationAgentTmp(agentRoot, time.Now(), config)
			if err != nil {
				log.Printf("[temp] cleanup scan failed: %v", err)
				return
			}
			logIntegrationTempCleanupEvents(summary.Events)
		}
		run()

		ticker := time.NewTicker(config.ScanInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}

func logIntegrationTempCleanupEvents(events []integrationTempCleanupEvent) {
	for _, event := range events {
		switch event.Action {
		case "pack":
			log.Printf("[%s pack] agent=%s archived %s -> %s", event.Directory, event.Agent, event.Source, event.Target)
		case "pack-skip":
			log.Printf("[%s pack] agent=%s skipped %s because target exists at %s", event.Directory, event.Agent, event.Source, event.Target)
		case "clear":
			log.Printf("[%s clear] agent=%s deleted %s", event.Directory, event.Agent, event.Source)
		case "error":
			log.Printf("[%s] agent=%s failed path=%s reason=%s", event.Directory, event.Agent, event.Source, event.Reason)
		}
	}
}

func cleanupIntegrationAgentTmp(agentRoot string, now time.Time, config integrationTempCleanupConfig) (integrationTempCleanupSummary, error) {
	entries, err := os.ReadDir(agentRoot)
	if err != nil {
		return integrationTempCleanupSummary{}, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	summary := integrationTempCleanupSummary{}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		agentID := strings.TrimSpace(entry.Name())
		if err := integrationagentarchive.ValidateAgentID(agentID); err != nil {
			continue
		}
		summary.AgentCount++
		workspace := filepath.Join(agentRoot, entry.Name())
		for _, directory := range integrationTempCleanupDirectories {
			if err := cleanupIntegrationTempDirectory(workspace, directory, agentID, now, config, &summary); err != nil {
				summary.Events = append(summary.Events, integrationTempCleanupEvent{
					Action:    "error",
					Agent:     agentID,
					Directory: directory,
					Source:    filepath.Join(workspace, directory),
					Reason:    err.Error(),
				})
			}
		}
	}
	return summary, nil
}

func cleanupIntegrationTempDirectory(workspace, directory, agentID string, now time.Time, config integrationTempCleanupConfig, summary *integrationTempCleanupSummary) error {
	rootDir := filepath.Join(workspace, directory)
	info, err := os.Lstat(rootDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s exists but is not a directory", directory)
	}

	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})
	bakDir := filepath.Join(rootDir, "bak")
	for _, entry := range entries {
		if entry.Name() == "bak" {
			continue
		}
		source := filepath.Join(rootDir, entry.Name())
		expired, err := integrationTempEntryExpired(source, now, config.PackAfter)
		if err != nil {
			return err
		}
		if !expired {
			continue
		}
		if err := os.MkdirAll(bakDir, 0o755); err != nil {
			return err
		}
		if err := mergeIntegrationTempArchiveEntry(source, filepath.Join(bakDir, entry.Name()), agentID, directory, summary); err != nil {
			return err
		}
	}

	bakInfo, err := os.Lstat(bakDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if !bakInfo.IsDir() {
		return fmt.Errorf("%s/bak exists but is not a directory", directory)
	}
	bakEntries, err := os.ReadDir(bakDir)
	if err != nil {
		return err
	}
	sort.Slice(bakEntries, func(i, j int) bool {
		return strings.ToLower(bakEntries[i].Name()) < strings.ToLower(bakEntries[j].Name())
	})
	for _, entry := range bakEntries {
		path := filepath.Join(bakDir, entry.Name())
		expired, err := integrationTempEntryExpired(path, now, config.ClearAfter)
		if err != nil {
			return err
		}
		if !expired {
			continue
		}
		if err := os.RemoveAll(path); err != nil {
			return err
		}
		summary.Events = append(summary.Events, integrationTempCleanupEvent{
			Action:    "clear",
			Agent:     agentID,
			Directory: directory,
			Source:    path,
		})
	}
	return nil
}

// integrationTempEntryExpired checks a top-level temp item. A directory is
// eligible only when every descendant file is old enough; an empty directory
// uses its own mtime. WalkDir does not follow symbolic links.
func integrationTempEntryExpired(path string, now time.Time, age time.Duration) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return !info.ModTime().After(now.Add(-age)), nil
	}
	containsFile := false
	expired := true
	err = filepath.WalkDir(path, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		containsFile = true
		entryInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if entryInfo.ModTime().After(now.Add(-age)) {
			expired = false
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	if !containsFile {
		return !info.ModTime().After(now.Add(-age)), nil
	}
	return expired, nil
}

// mergeIntegrationTempArchiveEntry recursively joins a stale temp entry into
// bak. Existing targets win; skipped sources remain in tmp for a later scan.
func mergeIntegrationTempArchiveEntry(source, target, agentID, directory string, summary *integrationTempCleanupSummary) error {
	sourceInfo, err := os.Lstat(source)
	if err != nil {
		return err
	}
	targetInfo, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.Rename(source, target); err != nil {
			return err
		}
		summary.Events = append(summary.Events, integrationTempCleanupEvent{Action: "pack", Agent: agentID, Directory: directory, Source: source, Target: target})
		return nil
	}
	if err != nil {
		return err
	}
	if !sourceInfo.IsDir() || !targetInfo.IsDir() {
		summary.Events = append(summary.Events, integrationTempCleanupEvent{Action: "pack-skip", Agent: agentID, Directory: directory, Source: source, Target: target})
		return nil
	}

	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})
	for _, entry := range entries {
		if err := mergeIntegrationTempArchiveEntry(filepath.Join(source, entry.Name()), filepath.Join(target, entry.Name()), agentID, directory, summary); err != nil {
			return err
		}
	}
	// A skipped collision makes this removal fail, intentionally preserving the
	// source directory and its skipped entries for a later scan.
	if err := os.Remove(source); err != nil && !errors.Is(err, os.ErrNotExist) && !errors.Is(err, syscall.ENOTEMPTY) {
		return err
	}
	return nil
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

func startIntegrationMCPRefresh(ctx context.Context, cfg *Config) {
	if ctx == nil || cfg == nil {
		return
	}
	go func() {
		for {
			if _, err := refreshIntegrationMCPMetadata(cfg, false); err != nil {
				log.Printf("[mcp_warning] refresh failed: %v", err)
			}
			interval := integrationMCPRefreshInterval()
			timer := time.NewTimer(interval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}()
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
		_, err := cronDB.Exec(`UPDATE task_detail SET replied_at = ?, reply_state = 'sent' WHERE id = ? AND reply_state = 'sending'`, time.Now().Format(time.RFC3339), detailID)
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

const cronDetailStartedFailed = 4

var errScheduledTaskSSEReadIdleTimeout = errors.New("scheduled task SSE read idle timeout")

type scheduledTaskSSEReadChunk struct {
	data []byte
	err  error
}

func scheduledTaskReadTimeout(cfg *Config) time.Duration {
	if cfg != nil && cfg.ScheduledTaskReadTimeout > 0 {
		return cfg.ScheduledTaskReadTimeout
	}
	return defaultScheduledTaskReadTimeout
}

// readScheduledTaskSSEBody applies an idle timeout to a single scheduled memo
// SSE response. It deliberately does not use http.Client.Timeout: that value
// is a total request deadline and would incorrectly stop healthy long-running
// streams. Closing the body interrupts the blocked transport read on timeout.
func readScheduledTaskSSEBody(body io.ReadCloser, idleTimeout time.Duration) ([]byte, error) {
	if body == nil {
		return nil, fmt.Errorf("scheduled task SSE response body is nil")
	}
	if idleTimeout <= 0 {
		idleTimeout = defaultScheduledTaskReadTimeout
	}

	chunks := make(chan scheduledTaskSSEReadChunk, 1)
	stop := make(chan struct{})
	go func() {
		defer close(chunks)
		buffer := make([]byte, 32*1024)
		emit := func(chunk scheduledTaskSSEReadChunk) bool {
			select {
			case chunks <- chunk:
				return true
			case <-stop:
				return false
			}
		}
		for {
			n, err := body.Read(buffer)
			if n > 0 {
				data := append([]byte(nil), buffer[:n]...)
				if !emit(scheduledTaskSSEReadChunk{data: data}) {
					return
				}
			}
			if err != nil {
				emit(scheduledTaskSSEReadChunk{err: err})
				return
			}
		}
	}()
	defer close(stop)

	timer := time.NewTimer(idleTimeout)
	defer timer.Stop()
	resetTimer := func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(idleTimeout)
	}

	payload := make([]byte, 0, 1024)
	for {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				return payload, io.ErrUnexpectedEOF
			}
			if len(chunk.data) > 0 {
				payload = append(payload, chunk.data...)
				resetTimer()
			}
			if chunk.err != nil {
				if errors.Is(chunk.err, io.EOF) {
					return payload, nil
				}
				return payload, chunk.err
			}
		case <-timer.C:
			_ = body.Close()
			return payload, errScheduledTaskSSEReadIdleTimeout
		}
	}
}

func failScheduledTaskDetail(detailID int, agentID, chatID string, cause error) {
	if cause == nil {
		cause = errors.New("scheduled task stream failed")
	}
	log.Printf("[cron] scheduled task failed: detail=%d agent=%s chat=%s err=%v", detailID, agentID, chatID, cause)
	appendChatLogDB(agentID, chatID, chatTypeScheduledTask, "A", "abnormal", buildSSEStreamInterruptedContent(cause))
	if cronDB == nil {
		return
	}
	if _, err := cronDB.Exec(`UPDATE task_detail SET started = ? WHERE id = ?`, cronDetailStartedFailed, detailID); err != nil {
		log.Printf("[cron] mark scheduled task failed: detail=%d err=%v", detailID, err)
		return
	}
	logCronDetailStatusByID(cronDB, detailID, "failed")
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
			failScheduledTaskDetail(t.ID, t.AgentID, chatID, fmt.Errorf("build task metadata: %w", err))
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
		injectApplicationDataDirMetadata(metaMap, cfg.AgentDir)
		if err := injectIntegrationConfigPathMetadata(metaMap, cfg); err != nil {
			failScheduledTaskDetail(t.ID, t.AgentID, chatID, fmt.Errorf("validate integration config: %w", err))
			continue
		}
		injectMCPMetadata(cfg, metadata, metaMap)
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
			failScheduledTaskDetail(t.ID, t.AgentID, chatID, fmt.Errorf("read task model config: %w", err))
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
		deviceID, _ := metaMap["deviceId"].(string)

		// Send to upstream
		detailID := t.ID
		go func(agentID, cID string, reqBody []byte, dID int, token, taskType, deviceID string) {
			targetURL := strings.TrimRight(cfg.currentHost(), "/") + "/v1/chat/completions"
			req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewReader(reqBody))
			if err != nil {
				failScheduledTaskDetail(dID, agentID, cID, fmt.Errorf("create upstream request: %w", err))
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "text/event-stream")
			setDeviceIDHeader(req, deviceID)
			if token != "" {
				req.Header.Set("Authorization", token)
			}

			finishSSE := trackIntegrationSSE(cfg)
			defer finishSSE()
			resp, err := proxyClient.Do(req)
			if err != nil {
				failScheduledTaskDetail(dID, agentID, cID, fmt.Errorf("forward upstream request: %w", err))
				sendSSECompletionNotification(chatTypeScheduledTask, taskType, "", true)
				return
			}
			defer resp.Body.Close()

			payload, readErr := readScheduledTaskSSEBody(resp.Body, scheduledTaskReadTimeout(cfg))
			if len(payload) > 0 {
				appendChatLogDB(agentID, cID, chatTypeScheduledTask, "A", detectResponseType(string(payload)), string(payload))
			}
			responseType := detectResponseType(string(payload))
			abnormalStream := readErr != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 || responseType == "abnormal"
			var failure error
			switch {
			case readErr != nil:
				failure = fmt.Errorf("read upstream SSE response: %w", readErr)
			case resp.StatusCode < 200 || resp.StatusCode >= 300:
				failure = fmt.Errorf("upstream returned HTTP %d", resp.StatusCode)
			case responseType == "abnormal":
				failure = errors.New("upstream returned an abnormal SSE response")
			case !sseEventHasDoneMarker(string(payload)):
				failure = errors.New("upstream SSE response ended without data: [DONE]")
			}
			if failure != nil {
				failScheduledTaskDetail(dID, agentID, cID, failure)
				sendSSECompletionNotification(chatTypeScheduledTask, taskType, "", true)
				return
			}
			resultContent := extractSSEAssistantText(payload)
			cronDB.Exec(`UPDATE task_detail
				SET started = 3,
					result_content = ?,
					reply_state = CASE WHEN task_type != 'cron' AND ? != '' THEN 'pending' ELSE reply_state END,
					reply_uuid = CASE WHEN task_type != 'cron' AND ? != '' THEN '' ELSE reply_uuid END,
					reply_started_at = CASE WHEN task_type != 'cron' AND ? != '' THEN 0 ELSE reply_started_at END
				WHERE id = ?`, resultContent, resultContent, resultContent, resultContent, dID)
			logCronDetailStatusByID(cronDB, dID, "complete")
			sendSSECompletionNotification(chatTypeScheduledTask, taskType, "", abnormalStream)
		}(t.AgentID, chatID, body, detailID, token, t.TaskType, deviceID)
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
	cronDB.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_chat_time_id ON chat_log(chat_id, created_at, id)`)
	ensureChatRecoveryIndexes(cronDB)
	cronDB.Exec(`CREATE TABLE IF NOT EXISTS agent_message_log (id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL DEFAULT '', chat_id TEXT NOT NULL DEFAULT '', content TEXT NOT NULL, log_type INTEGER NOT NULL, created_at TEXT NOT NULL)`)
	cronDB.Exec(`CREATE INDEX IF NOT EXISTS idx_agent_message_log_chat_type_time ON agent_message_log(chat_id, log_type, created_at)`)
	cronDB.Exec(`CREATE INDEX IF NOT EXISTS idx_agent_message_log_chat_time_id ON agent_message_log(chat_id, created_at, id)`)
	cronDB.Exec(`CREATE INDEX IF NOT EXISTS idx_agent_message_log_agent_chat_type_time ON agent_message_log(agent_id, chat_id, log_type, created_at)`)
	cronDB.Exec(`CREATE INDEX IF NOT EXISTS idx_agent_message_log_agent_chat_time ON agent_message_log(agent_id, chat_id, created_at)`)
	cronDB.Exec(`CREATE TABLE IF NOT EXISTS cmd_log (id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL, chat_id TEXT NOT NULL, tid TEXT NOT NULL, cmd TEXT NOT NULL, result TEXT NOT NULL DEFAULT '', status INTEGER NOT NULL DEFAULT -1, received_at TEXT NOT NULL, completed_at TEXT NOT NULL DEFAULT '')`)
	cronDB.Exec(`CREATE INDEX IF NOT EXISTS idx_cmd_agent_chat ON cmd_log(agent_id, chat_id)`)
	cronDB.Exec(`CREATE INDEX IF NOT EXISTS idx_cmd_agent_chat_time ON cmd_log(agent_id, chat_id, received_at)`)
}

func ensureChatRecoveryIndexes(db *sql.DB) {
	if db == nil {
		return
	}
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_recovery_latest ON chat_log(chat_type, role, response_type, chat_id, created_at DESC, id DESC)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_recovery_order ON chat_log(chat_type, role, response_type, created_at DESC, id DESC, chat_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_recovery_prompt ON chat_log(chat_id, role, created_at DESC, id DESC)`)
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
	// Normalize PATH before dispatching any command so subprocesses and the
	// long-running service share one executable search environment.
	sharedutil.ApplySystemPath()
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
