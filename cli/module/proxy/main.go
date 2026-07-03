package main

import (
	"agent-scanner/agentcore"
	"archive/zip"
	"bytes"

	"connect/connectsvc"
	"connect/sandboxstate"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"knowledge/knowledgecore"
	"log"

	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"

	"database/sql"

	"connect/pluginlog"
	"connect/sharedutil"
	_ "github.com/glebarez/go-sqlite"
	tokenconsume "proxy/consume"
	"proxy/eventlog"
	"skill-scanner/skillscore"
	"static-server/server"
)

type CmdRequest = sharedutil.CmdRequest
type KillRequest = sharedutil.KillRequest
type CmdResponse = sharedutil.CmdResponse
type KillResponse = sharedutil.KillResponse

const (
	defaultUpstreamHost = "https://www.deepright.cn"
	proxyServicePort    = 8080
)

// findAgent searches an Agent slice for one matching agentID.
func findAgent(agents []Agent, agentID string) *Agent {
	for i := range agents {
		if agents[i].AgentID == agentID {
			return &agents[i]
		}
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Skill & Agent metadata (inlined from agent module)
// ═══════════════════════════════════════════════════════════════════════════

type Skill = agentcore.Skill
type Agent = agentcore.Agent
type AgentOutput = agentcore.Output
type pluginActionResponse = connectsvc.PluginActionResponse
type pluginStatusResponse = connectsvc.PluginStatusResponse

type connectMetaConfigListItem = connectsvc.MetaConfig

func detectTerminal() string { return agentcore.DetectTerminal() }

func resolveExecutablePath(path string) string { return agentcore.ResolveExecutablePath(path) }

func detectGit() string { return agentcore.DetectGit() }

func detectPython3() string { return agentcore.DetectPython3() }

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

var proxyInstallAppPlatformKeyFn = proxyInstallAppPlatformKey

func proxyInstallAppPlatformKey() string {
	switch runtime.GOOS {
	case "darwin":
		return "mac"
	case "windows":
		return "wsl"
	case "linux":
		if proxyInstallAppIsWSL() {
			return "wsl"
		}
		return "linux"
	default:
		return ""
	}
}

func proxyInstallAppIsWSL() bool {
	if strings.TrimSpace(os.Getenv("WSL_DISTRO_NAME")) != "" {
		return true
	}
	if strings.TrimSpace(os.Getenv("WSL_INTEROP")) != "" {
		return true
	}
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(data)), "microsoft")
}

func proxyAppendInstallApps(base []string, raw interface{}) []string {
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
			base = proxyAppendInstallApps(base, value[key])
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

func proxyInstallAppsFromConfigValue(raw interface{}, platform string) []string {
	platform = strings.ToLower(strings.TrimSpace(platform))
	if value, ok := raw.(map[string]interface{}); ok {
		if platform != "" {
			return proxyAppendInstallApps(nil, value[platform])
		}
		return proxyAppendInstallApps(nil, value)
	}
	return proxyAppendInstallApps(nil, raw)
}

const proxyInstallAppCacheTTL = 5 * time.Minute

type proxyInstallAppCacheEntry struct {
	installed bool
	expiresAt time.Time
}

var proxyInstallAppAvailabilityCache struct {
	mu    sync.Mutex
	items map[string]proxyInstallAppCacheEntry
}

var proxyInstallAppNowFn = time.Now
var proxyInstallAppLookPathFn = exec.LookPath
var proxyInstallAppStatFn = os.Stat
var proxyInstallAppUserHomeDirFn = os.UserHomeDir
var proxyInstallAppDetectorFn = proxyDetectInstallAppInstalled

func proxyResetInstallAppAvailabilityCache() {
	proxyInstallAppAvailabilityCache.mu.Lock()
	proxyInstallAppAvailabilityCache.items = nil
	proxyInstallAppAvailabilityCache.mu.Unlock()
}

func proxyInstallAppCacheKey(name string) string {
	return proxyInstallAppPlatformKeyFn() + "|" + strings.ToLower(strings.TrimSpace(name))
}

func proxyInstallAppNameVariants(name string) []string {
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

func proxyInstallAppCommandCandidates(name string) []string {
	candidates := proxyInstallAppNameVariants(name)
	if proxyInstallAppPlatformKeyFn() == "wsl" || runtime.GOOS == "windows" {
		base := append([]string(nil), candidates...)
		for _, candidate := range base {
			if !strings.HasSuffix(strings.ToLower(candidate), ".exe") {
				candidates = append(candidates, candidate+".exe")
			}
		}
	}
	return candidates
}

func proxyResolveExecutableFromPaths(paths []string) string {
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

func proxyResolveExecutableFromGlobs(patterns []string) string {
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		if resolved := proxyResolveExecutableFromPaths(matches); resolved != "" {
			return resolved
		}
	}
	return ""
}

func proxyInstallAppCommonExecutablePaths(name string) []string {
	platform := proxyInstallAppPlatformKeyFn()
	if platform != "mac" && platform != "linux" {
		return nil
	}
	candidates := proxyInstallAppCommandCandidates(name)
	if len(candidates) == 0 {
		return nil
	}

	baseDirs := []string{"/opt/homebrew/bin", "/usr/local/bin", "/usr/bin"}
	homeDir, _ := proxyInstallAppUserHomeDirFn()
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

func proxyInstallAppCommonExecutableGlobs(name string) []string {
	platform := proxyInstallAppPlatformKeyFn()
	if platform != "mac" && platform != "linux" {
		return nil
	}
	candidates := proxyInstallAppCommandCandidates(name)
	if len(candidates) == 0 {
		return nil
	}

	homeDir, _ := proxyInstallAppUserHomeDirFn()
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

func proxyFindInstallAppExecutable(name string) string {
	for _, candidate := range proxyInstallAppCommandCandidates(name) {
		path, err := proxyInstallAppLookPathFn(candidate)
		if err != nil {
			continue
		}
		if resolved := resolveExecutablePath(path); resolved != "" {
			return resolved
		}
	}
	if resolved := proxyResolveExecutableFromPaths(proxyInstallAppCommonExecutablePaths(name)); resolved != "" {
		return resolved
	}
	return proxyResolveExecutableFromGlobs(proxyInstallAppCommonExecutableGlobs(name))
}

func proxyInstallAppExists(paths []string) bool {
	for _, candidate := range paths {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		info, err := proxyInstallAppStatFn(candidate)
		if err == nil && info != nil {
			return true
		}
	}
	return false
}

func proxyInstallAppMatchesAnyGlob(patterns []string) bool {
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		if proxyInstallAppExists(matches) {
			return true
		}
	}
	return false
}

func proxyDetectInstallAppInstalled(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return proxyInstallAppExists([]string{name})
	}
	if proxyFindInstallAppExecutable(name) != "" {
		return true
	}

	switch proxyInstallAppPlatformKeyFn() {
	case "mac":
		appNames := proxyInstallAppNameVariants(name)
		paths := make([]string, 0, len(appNames)*4)
		homeDir, _ := proxyInstallAppUserHomeDirFn()
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
		return proxyInstallAppExists(paths)
	case "wsl":
		executables := proxyInstallAppCommandCandidates(name)
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
		return proxyInstallAppExists(directPaths) || proxyInstallAppMatchesAnyGlob(globPatterns)
	default:
		return false
	}
}

func proxyIsInstallAppInstalled(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	now := proxyInstallAppNowFn()
	key := proxyInstallAppCacheKey(name)

	proxyInstallAppAvailabilityCache.mu.Lock()
	if entry, ok := proxyInstallAppAvailabilityCache.items[key]; ok && now.Before(entry.expiresAt) {
		proxyInstallAppAvailabilityCache.mu.Unlock()
		return entry.installed
	}
	proxyInstallAppAvailabilityCache.mu.Unlock()

	installed := proxyInstallAppDetectorFn(name)

	proxyInstallAppAvailabilityCache.mu.Lock()
	if proxyInstallAppAvailabilityCache.items == nil {
		proxyInstallAppAvailabilityCache.items = make(map[string]proxyInstallAppCacheEntry)
	}
	proxyInstallAppAvailabilityCache.items[key] = proxyInstallAppCacheEntry{
		installed: installed,
		expiresAt: now.Add(proxyInstallAppCacheTTL),
	}
	proxyInstallAppAvailabilityCache.mu.Unlock()
	return installed
}

func proxyFilterMissingInstallApps(apps []string) []string {
	out := make([]string, 0, len(apps))
	for _, app := range apps {
		app = strings.TrimSpace(app)
		if app == "" || proxyIsInstallAppInstalled(app) {
			continue
		}
		out = mergeInstallApps(out, app)
	}
	return out
}

type pooledDB struct {
	mu   sync.Mutex
	path string
	db   *sql.DB
}

var sharedDataDB pooledDB
var sharedEventLogger struct {
	mu     sync.Mutex
	path   string
	logger *eventlog.Logger
}

const defaultTaskType = sharedutil.DefaultTaskType

func getDataDB() (*sql.DB, error) {
	path, err := getDataDBPath()
	if err != nil {
		return nil, err
	}

	sharedDataDB.mu.Lock()
	defer sharedDataDB.mu.Unlock()

	if sharedDataDB.db != nil && sharedDataDB.path == path {
		return sharedDataDB.db, nil
	}
	if sharedDataDB.db != nil {
		_ = sharedDataDB.db.Close()
		sharedDataDB.db = nil
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	sharedDataDB.db = db
	sharedDataDB.path = path
	ensureCronSchema(db)
	return db, nil
}

func getDataDBPath() (string, error) {
	if dbPath := configuredProxyStringValue("db"); dbPath != "" {
		return dbPath, nil
	}
	return knowledgecore.DBPath(proxyAppDir())
}

func getEventLogger() (*eventlog.Logger, error) {
	path, err := getDataDBPath()
	if err != nil {
		return nil, err
	}

	sharedEventLogger.mu.Lock()
	defer sharedEventLogger.mu.Unlock()

	if sharedEventLogger.logger != nil && sharedEventLogger.path == path {
		return sharedEventLogger.logger, nil
	}
	if sharedEventLogger.logger != nil {
		_ = sharedEventLogger.logger.Close()
		sharedEventLogger.logger = nil
	}

	logger, err := eventlog.NewLogger(path, 512)
	if err != nil {
		return nil, err
	}
	sharedEventLogger.path = path
	sharedEventLogger.logger = logger
	return logger, nil
}

func agentExists(agentDir, deviceID, agentID string, ttl time.Duration) bool {
	metadata, err := getAgentOutput(agentDir, deviceID, ttl)
	if err != nil {
		return false
	}
	for _, agent := range metadata.Agents {
		if strings.TrimSpace(agent.AgentID) == strings.TrimSpace(agentID) {
			return true
		}
	}
	return false
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
			router_disable INTEGER NOT NULL DEFAULT 1,
			content TEXT NOT NULL,
			response_schema TEXT NOT NULL DEFAULT '',
			started INTEGER NOT NULL DEFAULT 0,
			occurred_at TEXT NOT NULL
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
		ensureCronRouterDisableColumn(db, tableName)
	}
}

func ensureCronRouterDisableColumn(db *sql.DB, tableName string) {
	if db == nil || hasTableColumn(db, tableName, "router_disable") {
		return
	}
	_, _ = db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN router_disable INTEGER NOT NULL DEFAULT 1`, tableName))
}

func cronRouterDisableSelect(db *sql.DB, tableName string) string {
	return cronRouterDisableSelectExpr(db, tableName, "router_disable")
}

func cronRouterDisableSelectExpr(db *sql.DB, tableName, expr string) string {
	if hasTableColumn(db, tableName, "router_disable") {
		return expr
	}
	return "1 AS router_disable"
}

func hasTableColumn(db *sql.DB, tableName, columnName string) bool {
	if db == nil {
		return false
	}
	rows, err := db.Query(`PRAGMA table_info(` + tableName + `)`)
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid        int
			name       string
			dataType   string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultVal, &pk); err != nil {
			continue
		}
		if strings.EqualFold(name, columnName) {
			return true
		}
	}
	return false
}

func getAgentOutput(root string, deviceID string, ttl time.Duration) (*AgentOutput, error) {
	output, err := agentcore.GetOutputForApp(root, ".", deviceID, ttl)
	if err != nil {
		return nil, err
	}
	cloned := *output
	hydrateAgentOutput(&cloned, "")
	if err := syncProxyKnowledgeOutput(&cloned, root); err != nil {
		return nil, err
	}
	cloned.Plugins = detectPluginKeys()
	return &cloned, nil
}

func getAgentOutputForChat(root string, deviceID string, ttl time.Duration, chatID string) (*AgentOutput, error) {
	output, err := agentcore.GetOutputForAppAndChat(root, ".", deviceID, ttl, chatID)
	if err != nil {
		return nil, err
	}
	cloned := *output
	hydrateAgentOutput(&cloned, chatID)
	if err := syncProxyKnowledgeOutput(&cloned, root); err != nil {
		return nil, err
	}
	cloned.Plugins = detectPluginKeys()
	return &cloned, nil
}

func proxyKnowledgeRoot(agentDir string) (string, error) {
	resolvedAgentDir, err := resolveAgentDir(agentDir)
	if err != nil {
		return "", fmt.Errorf("resolve agent-dir for knowledge: %w", err)
	}
	return filepath.Join(filepath.Dir(resolvedAgentDir), "knowledge"), nil
}

func proxyKnowledgeDirForAgent(agentDir, agentID string) (string, error) {
	root, err := proxyKnowledgeRoot(agentDir)
	if err != nil {
		return "", err
	}
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return root, nil
	}
	return filepath.Join(root, agentID), nil
}

func proxyKnowledgeLastUpdate(appDir string) int64 {
	db, err := knowledgecore.OpenExistingDB(appDir)
	if err != nil {
		return 0
	}
	defer db.Close()
	lastUpdate, err := knowledgecore.GetLastUpdate(db)
	if err != nil {
		return 0
	}
	return lastUpdate
}

func syncProxyKnowledgeOutput(output *AgentOutput, agentDir string) error {
	if output == nil {
		return nil
	}
	root, err := proxyKnowledgeRoot(agentDir)
	if err != nil {
		return err
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		output.Knowledge = nil
		return nil
	}
	lastUpdate := proxyKnowledgeLastUpdate(proxyAppDir())
	if output.Knowledge != nil && output.Knowledge.LastUpdate > lastUpdate {
		lastUpdate = output.Knowledge.LastUpdate
	}
	output.Knowledge = &agentcore.Knowledge{
		Path:       root,
		LastUpdate: lastUpdate,
	}
	return nil
}

func hydrateAgentOutput(output *AgentOutput, chatID string) {
	if output == nil {
		return
	}
	chatID = strings.TrimSpace(chatID)
	for i := range output.Agents {
		output.Agents[i].Sandbox = ""
		if chatID == "" {
			continue
		}
		mode, err := sandboxModeForSession(output.Agents[i].AgentID, chatID)
		if err == nil {
			output.Agents[i].Sandbox = mode
		}
	}
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

func workspaceForAgent(metadata *AgentOutput, agentID string) string {
	agentID = strings.TrimSpace(agentID)
	if metadata == nil || agentID == "" {
		return ""
	}
	for _, agent := range metadata.Agents {
		if strings.TrimSpace(agent.AgentID) == agentID {
			return agent.Workspace
		}
	}
	return ""
}

func attachLiveAgentMedia(agentMap map[string]interface{}, workspace string, fallback map[string]interface{}) {
	if agentMap == nil {
		return
	}
	media := cloneStringAnyMap(fallback)
	if len(media) == 0 {
		media = readLiveAgentMedia(workspace)
	}
	if len(media) == 0 {
		delete(agentMap, "media")
		return
	}
	agentMap["media"] = media
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

func selectedAgentMetadata(metadata *AgentOutput, agentID string) map[string]interface{} {
	out := map[string]interface{}{
		"agentId": strings.TrimSpace(agentID),
		"version": "",
		"sandbox": "",
	}
	agentID = strings.TrimSpace(agentID)
	if metadata == nil || agentID == "" {
		return out
	}
	for _, agent := range metadata.Agents {
		if strings.TrimSpace(agent.AgentID) != agentID {
			continue
		}
		raw, err := json.Marshal(agent)
		if err != nil {
			return out
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			return map[string]interface{}{
				"agentId": agentID,
				"version": agent.Version,
				"sandbox": agent.Sandbox,
			}
		}
		return out
	}
	return out
}

func detectPluginRuntimes() []agentcore.PluginRuntime {
	db, err := getDataDB()
	if err != nil {
		return nil
	}
	if err := ensureTokenStore(db); err != nil {
		return nil
	}
	rows, err := db.Query(`SELECT model, token, updated_at FROM token_store LIMIT 1`)
	if err != nil {
		return nil
	}
	rows.Close()

	svc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   "data",
		AgentDir: ".",
		CacheTTL: 0,
	})
	if err != nil {
		return nil
	}
	defer svc.Close()

	items, err := svc.ListMetaConfig(false)
	if err != nil {
		return nil
	}

	plugins := make([]agentcore.PluginRuntime, 0, len(items))
	for _, item := range items {
		plugins = append(plugins, agentcore.PluginRuntime{
			Key:      strings.TrimSpace(item.Key),
			Callback: strings.TrimSpace(item.Callback),
		})
	}
	return plugins
}

func detectPluginKeys() []string {
	return agentcore.DetectRunningPluginKeys("plugins", detectPluginRuntimes())
}

func pluginStarted(key, callback string) bool {
	return agentcore.PluginStarted("plugins", key, callback)
}

func pluginPIDFile(key, callback string) string {
	files := agentcore.PluginPIDFiles("plugins", key, callback)
	if len(files) == 0 {
		return filepath.Join("plugins", strings.TrimSpace(key)+".pid")
	}
	return files[0]
}

// ═══════════════════════════════════════════════════════════════════════════
// Proxy handler
// ═══════════════════════════════════════════════════════════════════════════

// ProxyServer holds the proxy configuration and state.
type ProxyServer struct {
	Host                    string
	AgentDir                string
	DefaultDir              string
	DeviceID                string
	CacheTTL                time.Duration
	SiteDir                 string
	ConnectTimeout          time.Duration
	KnowledgeUpdateInterval time.Duration
	KnowledgeUpdateLock     time.Duration
	InstallApps             []string
	Client                  *http.Client
	ConnectCacheTTL         time.Duration
	AutoReply               string
	ConnectService          *connectsvc.Service
	WarningScanRoot         string
	PluginExecTimeout       time.Duration
}

// sharedutil.NewProxyClient creates an HTTP client with only connect timeout, no read/total timeout.

func (p *ProxyServer) newConnectService() (*connectsvc.Service, error) {
	if p != nil && p.ConnectService != nil {
		return p.ConnectService, nil
	}
	cacheTTL := p.ConnectCacheTTL
	if cacheTTL <= 0 {
		cacheTTL = 10 * time.Second
	}
	return connectsvc.NewService(connectsvc.Options{
		DBPath:   "data",
		AgentDir: p.AgentDir,
		CacheTTL: cacheTTL,
	})
}

func (p *ProxyServer) borrowConnectService() (*connectsvc.Service, func(), error) {
	if p != nil && p.ConnectService != nil {
		return p.ConnectService, func() {}, nil
	}
	svc, err := p.newConnectService()
	if err != nil {
		return nil, nil, err
	}
	return svc, func() { _ = svc.Close() }, nil
}

func writeConnectError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":  1,
		"content": err.Error(),
	})
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

const (
	defaultAPICmdTimeout       = 180 * time.Second
	defaultPluginExecTimeoutMs = 600000
)

func initCmdLogTable(db *sql.DB) {
	db.Exec(`CREATE TABLE IF NOT EXISTS cmd_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		agent_id TEXT NOT NULL,
		chat_id TEXT NOT NULL,
		tid TEXT NOT NULL,
		cmd TEXT NOT NULL,
		result TEXT NOT NULL DEFAULT '',
		status INTEGER NOT NULL DEFAULT -1,
		received_at TEXT NOT NULL,
		completed_at TEXT NOT NULL DEFAULT ''
	)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_cmd_agent_chat ON cmd_log(agent_id, chat_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_cmd_agent_chat_time ON cmd_log(agent_id, chat_id, received_at)`)
}

func initKillLogTable(db *sql.DB) {
	db.Exec(`CREATE TABLE IF NOT EXISTS kill_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		agent_id TEXT NOT NULL,
		chat_id TEXT NOT NULL,
		tid TEXT NOT NULL,
		cmd TEXT NOT NULL,
		received_at TEXT NOT NULL,
		completed_at TEXT NOT NULL DEFAULT ''
	)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_kill_agent_chat_time ON kill_log(agent_id, chat_id, received_at)`)
}

func insertCmdLog(agentID, chatID, tid, cmd, receivedAt string) (int64, error) {
	db, err := getDataDB()
	if err != nil {
		return 0, err
	}
	initCmdLogTable(db)
	res, err := db.Exec(`INSERT INTO cmd_log (agent_id, chat_id, tid, cmd, received_at) VALUES (?,?,?,?,?)`,
		agentID, chatID, tid, cmd, receivedAt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func updateCmdLog(id int64, result string, status int, completedAt string) error {
	if id <= 0 {
		return nil
	}
	db, err := getDataDB()
	if err != nil {
		return err
	}
	initCmdLogTable(db)
	_, err = db.Exec(`UPDATE cmd_log SET result = ?, status = ?, completed_at = ? WHERE id = ?`,
		sharedutil.GzipBase64String(result), status, completedAt, id)
	return err
}

func insertKillLog(agentID, chatID, tid, cmd, receivedAt string) (int64, error) {
	db, err := getDataDB()
	if err != nil {
		return 0, err
	}
	initKillLogTable(db)
	res, err := db.Exec(`INSERT INTO kill_log (agent_id, chat_id, tid, cmd, received_at) VALUES (?,?,?,?,?)`,
		agentID, chatID, tid, cmd, receivedAt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func updateKillLog(id int64, completedAt string) error {
	if id <= 0 {
		return nil
	}
	db, err := getDataDB()
	if err != nil {
		return err
	}
	initKillLogTable(db)
	_, err = db.Exec(`UPDATE kill_log SET completed_at = ? WHERE id = ?`, completedAt, id)
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

func executeExternalCommand(binary string, args []string, rawCmd string, timeoutMs int64, onStart func(*activeCmd)) (int, string) {
	timeout := time.Duration(timeoutMs) * time.Millisecond
	if timeoutMs <= 0 {
		timeout = defaultAPICmdTimeout
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
		return 1, "命令执行超时"
	}
	if ctx.Err() == context.Canceled {
		return 1, "命令被终止"
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == -1 {
			return 1, "命令被终止"
		}
		if strings.TrimSpace(output) != "" {
			return 1, output
		}
		return 1, err.Error()
	}
	return 0, output
}

func executeShellCommand(shell, rawCmd string, timeoutMs int64, onStart func(*activeCmd)) (int, string) {
	return executeExternalCommand(shell, []string{"-c", rawCmd}, rawCmd, timeoutMs, onStart)
}

var proxyOsExecutable = os.Executable
var proxySandboxExecutablePathFn = resolveProxySandboxExecutablePath

func proxyConfigPaths() []string {
	paths := make([]string, 0, 2)
	if executable, err := proxyOsExecutable(); err == nil && strings.TrimSpace(executable) != "" {
		execDir := filepath.Dir(filepath.Clean(executable))
		paths = append(paths, filepath.Join(execDir, "config", "config.json"))
	}
	if cwd, err := os.Getwd(); err == nil {
		cwdPath := filepath.Join(cwd, "config", "config.json")
		if len(paths) == 0 || paths[len(paths)-1] != cwdPath {
			paths = append(paths, cwdPath)
		}
	}
	return paths
}

func proxyStartupConfigPath() string {
	if cwd, err := os.Getwd(); err == nil && strings.TrimSpace(cwd) != "" {
		return filepath.Join(cwd, "config", "config.json")
	}
	executable, err := proxyOsExecutable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(filepath.Clean(executable)), "config", "config.json")
}

func readProxyStartupConfigRaw() (map[string]interface{}, string, error) {
	for _, path := range proxyConfigPaths() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, path, err
		}
		if raw == nil {
			raw = map[string]interface{}{}
		}
		return raw, path, nil
	}
	path := proxyStartupConfigPath()
	if strings.TrimSpace(path) == "" {
		return nil, "", fmt.Errorf("cannot resolve config/config.json path")
	}
	return nil, path, nil
}

func readProxyStartupConfig() map[string]string {
	raw, _, err := readProxyStartupConfigRaw()
	if err != nil || raw == nil {
		return nil
	}
	out := make(map[string]string, len(raw))
	for key, value := range raw {
		out[key] = strings.TrimSpace(fmt.Sprint(value))
	}
	return out
}

func configuredProxyStringValue(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	for _, path := range proxyConfigPaths() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}
		if value := strings.TrimSpace(fmt.Sprint(raw[key])); value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func configuredProxySandboxApp() string {
	return configuredProxyStringValue("sandbox_app")
}

func configuredProxyInstallApps() []string {
	platform := proxyInstallAppPlatformKeyFn()
	for _, path := range proxyConfigPaths() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}
		value, ok := raw["install_app"]
		if !ok {
			continue
		}
		return proxyInstallAppsFromConfigValue(value, platform)
	}
	return nil
}

func configuredProxyIntValue(key string) (int, bool) {
	key = strings.TrimSpace(key)
	if key == "" {
		return 0, false
	}
	for _, path := range proxyConfigPaths() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}
		value, ok := raw[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case float64:
			if typed > 0 {
				return int(typed), true
			}
		case int:
			if typed > 0 {
				return typed, true
			}
		case string:
			parsed, err := strconv.Atoi(strings.TrimSpace(typed))
			if err == nil && parsed > 0 {
				return parsed, true
			}
		}
	}
	return 0, false
}

func resolveProxySkillExtractRound() int {
	if value, ok := configuredProxyIntValue("skill_extract"); ok && value > 0 {
		return value
	}
	return 10
}

func normalizeConfiguredSkillNames(input interface{}) []string {
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

func configuredProxySkillNames() []string {
	for _, path := range proxyConfigPaths() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var raw struct {
			Skills interface{} `json:"skills"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}
		return normalizeConfiguredSkillNames(raw.Skills)
	}
	return nil
}

func buildProxyRuntimeSkillNames(base []string) []string {
	out := append([]string(nil), base...)
	seen := make(map[string]struct{}, len(base)+4)
	appendIfMissing := func(name string, enabled bool) {
		name = strings.TrimSpace(name)
		if !enabled || name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		out = append(out, name)
		seen[name] = struct{}{}
	}

	for _, name := range out {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		seen[name] = struct{}{}
	}
	for _, name := range configuredProxySkillNames() {
		appendIfMissing(name, true)
	}
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

func proxySandboxExecutableCandidates(raw, mode string) []string {
	raw = filepath.Clean(raw)
	var candidates []string
	addCandidate := func(path string) {
		path = filepath.Clean(path)
		for _, existing := range candidates {
			if existing == path {
				return
			}
		}
		candidates = append(candidates, path)
	}

	baseDir := raw
	appDir := ""
	switch {
	case strings.HasSuffix(strings.ToLower(raw), filepath.Clean("/Contents/MacOS/CLI_SANDBOX")):
		appDir = filepath.Dir(filepath.Dir(filepath.Dir(raw)))
		baseDir = filepath.Dir(appDir)
	case strings.HasSuffix(strings.ToLower(raw), ".app"):
		appDir = raw
		baseDir = filepath.Dir(appDir)
	}
	if appDir != "" {
		addCandidate(filepath.Join(baseDir, mode, "CLI_SANDBOX.app", "Contents", "MacOS", "CLI_SANDBOX"))
		addCandidate(filepath.Join(appDir, "Contents", "MacOS", "CLI_SANDBOX"))
		return candidates
	}

	addCandidate(filepath.Join(raw, mode, "CLI_SANDBOX.app", "Contents", "MacOS", "CLI_SANDBOX"))
	addCandidate(filepath.Join(raw, mode, "CLI_SANDBOX"))
	addCandidate(filepath.Join(raw, "helpers", mode, "CLI_SANDBOX"))
	addCandidate(raw)
	return candidates
}

func resolveProxySandboxExecutablePath(mode string) (string, error) {
	mode = sandboxstate.NormalizeMode(mode)
	if mode == "" {
		return "", fmt.Errorf("sandbox mode is required")
	}
	raw := strings.TrimSpace(configuredProxySandboxApp())
	if raw == "" {
		return "", fmt.Errorf("sandbox_app is not configured")
	}
	executable, err := proxyOsExecutable()
	if err != nil {
		return "", fmt.Errorf("locate executable: %w", err)
	}
	baseDir := filepath.Dir(filepath.Clean(executable))
	if !filepath.IsAbs(raw) {
		raw = filepath.Join(baseDir, raw)
	}
	raw = filepath.Clean(raw)
	for _, candidate := range proxySandboxExecutableCandidates(raw, mode) {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("sandbox helper stat failed for mode=%s: %w", mode, err)
		}
	}
	return "", fmt.Errorf("sandbox helper not found for mode=%s", mode)
}

func executeSandboxCommandWithPath(path, shell, rawCmd string, timeoutMs int64, onStart func(*activeCmd)) (int, string) {
	args := []string{"--cmd", rawCmd}
	if shell = strings.TrimSpace(shell); shell != "" {
		args = append(args, "--shell", shell)
	}
	if timeoutMs > 0 {
		args = append(args, "--timeout", strconv.FormatInt(timeoutMs, 10))
	}
	return executeExternalCommand(path, args, rawCmd, timeoutMs, onStart)
}

func executeSandboxCommandLogged(scope, agentID, chatID, tid, mode, shell, rawCmd string, timeoutMs int64, onStart func(*activeCmd)) (int, string) {
	path, err := proxySandboxExecutablePathFn(mode)
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
	log.Printf("%s: sandbox exec start agentId=%s chatId=%s tid=%s helper=%s shell=%s timeoutMs=%d mode=%s cmd=%q",
		scope,
		strings.TrimSpace(agentID),
		strings.TrimSpace(chatID),
		strings.TrimSpace(tid),
		strings.TrimSpace(path),
		strings.TrimSpace(shell),
		timeoutMs,
		mode,
		rawCmd,
	)
	startedAt := time.Now()
	status, output := executeSandboxCommandWithPath(path, shell, rawCmd, timeoutMs, onStart)
	log.Printf("%s: sandbox exec done agentId=%s chatId=%s tid=%s status=%d durationMs=%d mode=%s output=%q",
		scope,
		strings.TrimSpace(agentID),
		strings.TrimSpace(chatID),
		strings.TrimSpace(tid),
		status,
		time.Since(startedAt).Milliseconds(),
		mode,
		output,
	)
	return status, output
}

func sandboxModeForSession(agentID, chatID string) (string, error) {
	db, err := getDataDB()
	if err != nil {
		return "", err
	}
	state, recorded, err := sandboxstate.Get(db, agentID, chatID)
	if err != nil {
		return "", err
	}
	if !recorded {
		return "", nil
	}
	return sandboxstate.NormalizeMode(state.Sandbox), nil
}

// HandleCmd serves POST /api/cmd, executing a command for an agent chat from localhost only.
func (p *ProxyServer) HandleCmd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !sharedutil.IsLocalExecutionRequest(r) {
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

	metadata, err := getAgentOutput(p.AgentDir, p.DeviceID, p.CacheTTL)
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
	cmdLogID, err := insertCmdLog(req.AgentID, req.ChatID, tid, req.Cmd, receivedAt)
	if err != nil {
		log.Printf("api/cmd: insert cmd_log failed: %v", err)
	}

	sandboxMode, err := sandboxModeForSession(req.AgentID, req.ChatID)
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

	var activeKey string
	var status int
	var output string
	if sandboxMode != "" {
		status, output = executeSandboxCommandLogged("api/cmd", req.AgentID, req.ChatID, tid, sandboxMode, sharedutil.DefaultExecShell(), req.Cmd, req.Timeout, func(ac *activeCmd) {
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
	if err := updateCmdLog(cmdLogID, output, status, completedAt); err != nil {
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

// HandleKill serves POST /api/kill, terminating an active command for an agent chat from localhost only.
func (p *ProxyServer) HandleKill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !sharedutil.IsLocalExecutionRequest(r) {
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

	metadata, err := getAgentOutput(p.AgentDir, p.DeviceID, p.CacheTTL)
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
	logID, err := insertKillLog(req.AgentID, req.ChatID, req.Tid, req.Cmd, receivedAt)
	if err != nil {
		log.Printf("api/kill: insert kill_log failed: %v", err)
	}

	ac, key := findActiveCmd(req.AgentID, req.ChatID, req.Tid, req.Cmd)
	if ac == nil {
		completedAt := time.Now().Format("2006-01-02T15:04:05.000")
		_ = updateKillLog(logID, completedAt)
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
	if err := updateKillLog(logID, completedAt); err != nil {
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

// HandleAgentIDs serves GET /api/agentId, returning all agent IDs as a JSON array.
func (p *ProxyServer) HandleAgentIDs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metadata, err := getAgentOutput(p.AgentDir, p.DeviceID, p.CacheTTL)
	if err != nil {
		log.Printf("agent scan error: %v", err)
		http.Error(w, "Failed to get agent metadata", http.StatusInternalServerError)
		return
	}

	ids := make([]string, len(metadata.Agents))
	for i, a := range metadata.Agents {
		ids[i] = a.AgentID
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ids)
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

// HandleSwarmAgents serves GET /api/swarm_agent, returning all Agent IDs with SWARM enabled.
func (p *ProxyServer) HandleSwarmAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metadata, err := getAgentOutput(p.AgentDir, p.DeviceID, p.CacheTTL)
	if err != nil {
		log.Printf("swarm agent metadata error: %v", err)
		http.Error(w, "Failed to get agent metadata", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(swarmAgentIDs(metadata, r.URL.Query().Get("agentId")))
}

// HandleDeviceID serves GET /api/deviceId, returning the shared deviceId from agent metadata.
func (p *ProxyServer) HandleDeviceID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metadata, err := getAgentOutput(p.AgentDir, p.DeviceID, p.CacheTTL)
	if err != nil {
		log.Printf("deviceId metadata error: %v", err)
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

// HandlePluginsMeta serves GET /api/plugins/meta, returning available plugin metadata.
func (p *ProxyServer) HandlePluginsMeta(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// This endpoint must reflect plugin definitions and saved meta changes immediately,
	// so it bypasses shared plugin discovery caches. When a service is injected,
	// reuse it for DB access; otherwise create a short-lived service on demand.
	var items []connectsvc.PluginMetaInfo
	var err error
	if p != nil && p.ConnectService != nil {
		items, err = listLocalPluginMeta(p.ConnectService)
	} else {
		svc, svcErr := connectsvc.NewService(connectsvc.Options{
			DBPath:   "data",
			AgentDir: p.AgentDir,
			CacheTTL: time.Millisecond,
		})
		if svcErr != nil {
			log.Printf("plugins meta error: %v", svcErr)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"status":  1,
				"content": svcErr.Error(),
			})
			return
		}
		defer svc.Close()
		items, err = listLocalPluginMeta(svc)
	}
	if err != nil {
		log.Printf("plugins meta error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"status":  1,
			"content": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": 0,
		"data":   items,
	})
}

// HandlePluginsLog serves GET /api/plugins/log?key=xxx, streaming plugin logs via SSE.
func (p *ProxyServer) HandlePluginsLog(w http.ResponseWriter, r *http.Request) {
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

	if err := pluginlog.ServeSSE(r.Context(), w, logPath, pluginlog.Options{LastLines: lastLines}); err != nil && r.Context().Err() == nil {
		log.Printf("plugins log stream ended: plugin=%s log=%s err=%v", plugin, logPath, err)
	}
}

// HandlePluginsStatus serves GET /api/plugins/status?key=xxx, returning plugin running state.
func (p *ProxyServer) HandlePluginsStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name, flags, err := parsePluginActionRequest(r)
	if err != nil {
		connectsvc.WritePluginActionError(w, http.StatusBadRequest, err)
		return
	}

	result, err := connectsvc.PluginStatusByKey(name, flags)
	if err != nil {
		log.Printf("plugins status error: %v", err)
		connectsvc.WritePluginActionError(w, connectsvc.PluginActionStatusCode(err), err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": 0,
		"data":   buildPluginStatusResponse(result),
	})
}

// HandlePluginsConfig serves POST /api/plugins/config, creating or updating plugin config.
func (p *ProxyServer) HandlePluginsConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	input, err := parsePluginConfigRequest(r)
	if err != nil {
		writePluginConfigError(w, http.StatusBadRequest, err)
		return
	}
	input.Name = firstNonEmpty(input.Key, input.Name)

	svc, release, err := p.borrowConnectService()
	if err != nil {
		log.Printf("plugins config error: %v", err)
		writePluginConfigError(w, http.StatusBadRequest, err)
		return
	}
	defer release()

	item, err := connectsvc.UpsertPluginConfigWithService(svc, input)
	if err != nil {
		log.Printf("plugins config error: %v", err)
		writePluginConfigError(w, http.StatusBadRequest, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": 0,
		"data":   item,
	})
}

func (p *ProxyServer) HandlePluginsStart(w http.ResponseWriter, r *http.Request) {
	p.handlePluginAction(w, r, "start")
}

func (p *ProxyServer) HandlePluginsStop(w http.ResponseWriter, r *http.Request) {
	p.handlePluginAction(w, r, "stop")
}

func (p *ProxyServer) HandlePluginsExec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name, command, flags, err := connectsvc.ParsePluginExecRequest(r)
	if err != nil {
		connectsvc.WritePluginActionError(w, http.StatusBadRequest, err)
		return
	}
	ensurePluginConnectBin(flags)

	result, err := connectsvc.RunPluginExecCommand(name, command, flags, p.PluginExecTimeout)
	if err != nil {
		log.Printf("plugins exec error: %v", err)
		connectsvc.WritePluginActionError(w, connectsvc.PluginActionStatusCode(err), err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": 0,
		"data":   buildPluginActionResponse(result),
	})
}

func (p *ProxyServer) handlePluginAction(w http.ResponseWriter, r *http.Request, action string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name, flags, err := parsePluginActionRequest(r)
	if err != nil {
		connectsvc.WritePluginActionError(w, http.StatusBadRequest, err)
		return
	}
	ensurePluginConnectBin(flags)

	result, err := connectsvc.RunPluginAction(name, action, flags)
	if err != nil {
		log.Printf("plugins %s error: %v", action, err)
		connectsvc.WritePluginActionError(w, connectsvc.PluginActionStatusCode(err), err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": 0,
		"data":   buildPluginActionResponse(result),
	})
}

func ensurePluginConnectBin(flags map[string]string) {
	exe, err := os.Executable()
	if err != nil {
		exe = ""
	}
	connectsvc.EnsureConnectBinary(flags, exe)
}

func resolveProxyPluginExecTimeout() time.Duration {
	if value, ok := configuredProxyIntValue("plugin_exec_timeout"); ok && value > 0 {
		return time.Duration(value) * time.Millisecond
	}
	return time.Duration(defaultPluginExecTimeoutMs) * time.Millisecond
}

// HandleFolder serves GET /api/folder?agentId=xxx, opens the agent's workspace folder.
func (p *ProxyServer) HandleFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pathParam := normalizeQuotedPathArg(r.URL.Query().Get("path"))
	if pathParam != "" {
		expandedPath, err := ExpandPath(pathParam)
		if err != nil {
			http.Error(w, "Failed to expand path: "+err.Error(), http.StatusBadRequest)
			return
		}
		if !filepath.IsAbs(expandedPath) {
			http.Error(w, "path must be absolute", http.StatusBadRequest)
			return
		}
		resolved, err := ResolveCaseInsensitive(expandedPath)
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
		if err := OpenFolder(resolved); err != nil {
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

	metadata, err := getAgentOutput(p.AgentDir, p.DeviceID, p.CacheTTL)
	if err != nil {
		log.Printf("agent scan error: %v", err)
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

	// Optional dir parameter to open a subdirectory
	dir := normalizeQuotedPathArg(r.URL.Query().Get("dir"))
	target := workspace
	if dir != "" {
		target = filepath.Join(workspace, dir)
	}

	if err := OpenFolder(target); err != nil {
		log.Printf("open folder error: %v", err)
		http.Error(w, "Failed to open folder: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "path": target})
}

// OpenFolder opens a folder using the system's file manager.
var openFolderFn = openFolderSystem

func OpenFolder(path string) error {
	return openFolderFn(path)
}

func openFolderSystem(path string) error {
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "linux":
		cmd = "xdg-open"
	case "windows":
		cmd = "explorer"
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	return exec.Command(cmd, path).Start()
}

func proxyBrowserURL(port int) string {
	if port <= 0 {
		port = 8080
	}
	return fmt.Sprintf("http://localhost:%d/site/#app", port)
}

// HandleSkills serves GET /api/skills?agentId=xxx, returning skill names for the agent.
func (p *ProxyServer) HandleSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := r.URL.Query().Get("agentId")
	if agentID == "" {
		http.Error(w, "agentId is required", http.StatusBadRequest)
		return
	}

	metadata, err := getAgentOutput(p.AgentDir, p.DeviceID, p.CacheTTL)
	if err != nil {
		log.Printf("agent scan error: %v", err)
		http.Error(w, "Failed to get agent metadata", http.StatusInternalServerError)
		return
	}

	for _, a := range metadata.Agents {
		if a.AgentID == agentID {
			names := buildProxyRuntimeSkillNames(agentcore.SkillNames(a.Skills))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(names)
			return
		}
	}
	http.Error(w, "Agent not found: "+agentID, http.StatusNotFound)
}

func resolveSkillsWarningRoot(p *ProxyServer) string {
	if p != nil && strings.TrimSpace(p.WarningScanRoot) != "" {
		return strings.TrimSpace(p.WarningScanRoot)
	}
	if p != nil && strings.TrimSpace(p.AgentDir) != "" {
		return filepath.Join(strings.TrimSpace(p.AgentDir), "skills")
	}
	return "skills"
}

func syncSkillsWarnings(p *ProxyServer) error {
	root := resolveSkillsWarningRoot(p)
	dbPath, err := getDataDBPath()
	if err != nil {
		return err
	}
	_, err = skillscore.ScanAndSyncWarnings(root, dbPath)
	return err
}

// HandleSkillsWarning serves GET /skills_warning, returning current SKILL parse warnings.
func (p *ProxyServer) HandleSkillsWarning(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if refresh := strings.TrimSpace(r.URL.Query().Get("refresh")); refresh != "" && refresh != "0" && strings.ToLower(refresh) != "false" {
		if err := syncSkillsWarnings(p); err != nil {
			log.Printf("skills warning refresh error: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": err.Error()})
			return
		}
	}

	dbPath, err := getDataDBPath()
	if err != nil {
		http.Error(w, "Failed to resolve data path", http.StatusInternalServerError)
		return
	}
	store, err := skillscore.OpenWarningStore(dbPath)
	if err != nil {
		log.Printf("skills warning open db error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": err.Error()})
		return
	}
	items, err := store.List()
	if err != nil {
		log.Printf("skills warning list error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": 0, "data": items})
}

// FileEntry represents a file or directory in a listing.
type FileEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type knowledgeTreeNode struct {
	Name     string
	Children []*knowledgeTreeNode
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

// ExpandPath expands ~ to the user's home directory.
func ExpandPath(path string) (string, error) {
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

// HandleFiles serves GET /api/files?path=xxx, listing files and dirs with prefix matching.
func (p *ProxyServer) HandleFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dirPath := r.URL.Query().Get("path")
	if dirPath == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	expanded, err := ExpandPath(dirPath)
	if err != nil {
		http.Error(w, "Failed to expand path: "+err.Error(), http.StatusBadRequest)
		return
	}

	// If path is a directory, list all entries; otherwise prefix-match in parent dir
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
	for _, e := range entries {
		if prefix != "" && !strings.HasPrefix(strings.ToLower(e.Name()), strings.ToLower(prefix)) {
			continue
		}
		t := "file"
		if e.IsDir() {
			t = "dir"
		}
		result = append(result, FileEntry{Name: e.Name(), Type: t})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ResolveCaseInsensitive resolves a path with case-insensitive matching for each segment.
func ResolveCaseInsensitive(target string) (string, error) {
	expanded, err := ExpandPath(target)
	if err != nil {
		return "", err
	}
	// If exact path exists, use it directly
	if _, err := os.Stat(expanded); err == nil {
		return expanded, nil
	}
	// Walk each segment case-insensitively
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

func knowledgeRelativePath(requestPath string) (string, error) {
	decoded, err := url.PathUnescape(requestPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	trimmed := strings.TrimPrefix(decoded, "/knowledge")
	trimmed = strings.TrimPrefix(trimmed, "/")
	if trimmed == "" {
		return ".", nil
	}
	cleaned := filepath.Clean(trimmed)
	if cleaned == "." {
		return ".", nil
	}
	if filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes knowledge root")
	}
	return cleaned, nil
}

// HandleKnowledge serves /knowledge and /knowledge/... from the configured agent knowledge directory.
func (p *ProxyServer) HandleKnowledge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	root, err := proxyKnowledgeRoot(p.AgentDir)
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

// HandleKnowledgeLastUpdate serves /knowledge_lastUpdate with the formatted knowledge last update time.
func (p *ProxyServer) HandleKnowledgeLastUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	agentID := strings.TrimSpace(r.URL.Query().Get("agentId"))
	db, err := knowledgecore.OpenSharedDB(proxyAppDir())
	if err != nil {
		http.Error(w, "Failed to open knowledge runtime: "+err.Error(), http.StatusInternalServerError)
		return
	}
	lastUpdate, err := knowledgecore.GetLastUpdateForAgent(db, agentID)
	if err != nil {
		http.Error(w, "Failed to load knowledge last update: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, time.UnixMilli(lastUpdate).In(time.Local).Format("2006-01-02 15:04"))
}

// HandleKnowledgePath serves /knowledge_path with the absolute knowledge directory path.
func (p *ProxyServer) HandleKnowledgePath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	root, err := proxyKnowledgeRoot(p.AgentDir)
	if err != nil {
		http.Error(w, "Failed to resolve knowledge dir: "+err.Error(), http.StatusInternalServerError)
		return
	}
	agentID := strings.TrimSpace(r.URL.Query().Get("agentId"))
	target := root
	if agentID != "" {
		target, err = proxyKnowledgeDirForAgent(p.AgentDir, agentID)
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

func (p *ProxyServer) resolveFileTarget(agentID, filePath string) (string, error) {
	rawPath := normalizeQuotedPathArg(filePath)
	if rawPath == "" {
		return "", fmt.Errorf("file is required")
	}
	expandedPath, err := ExpandPath(rawPath)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(expandedPath) {
		resolved, err := ResolveCaseInsensitive(expandedPath)
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
	workspace, err := p.getWorkspaceByAgentID(agentID)
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

func (p *ProxyServer) HandleFileLastUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	filePath := strings.TrimSpace(r.URL.Query().Get("file"))
	agentID := strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("agentId"), r.URL.Query().Get("agent")))
	resolved, err := p.resolveFileTarget(agentID, filePath)
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

// DataResponse is the JSON response for /api/data.
type DataResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Status  int    `json:"status"`
}

// binaryExts lists file extensions considered binary (images, multimedia, etc.)
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

func writeDataResponse(w http.ResponseWriter, resp DataResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleData serves GET /api/data?path=xxx, returning file content as JSON.
func (p *ProxyServer) HandleData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		writeDataResponse(w, DataResponse{Path: "", Content: "path is required", Status: 1})
		return
	}

	resolved, err := ResolveCaseInsensitive(filePath)
	if err != nil {
		writeDataResponse(w, DataResponse{Path: filePath, Content: "文件不存在: " + err.Error(), Status: 1})
		return
	}

	info, err := os.Stat(resolved)
	if err != nil {
		writeDataResponse(w, DataResponse{Path: resolved, Content: "无法访问: " + err.Error(), Status: 1})
		return
	}
	if info.IsDir() {
		writeDataResponse(w, DataResponse{Path: resolved, Content: "不支持读取目录", Status: 1})
		return
	}
	if isBinaryFile(resolved) {
		writeDataResponse(w, DataResponse{Path: resolved, Content: "不支持读取二进制文件", Status: 1})
		return
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		writeDataResponse(w, DataResponse{Path: resolved, Content: "读取失败: " + err.Error(), Status: 1})
		return
	}

	writeDataResponse(w, DataResponse{Path: resolved, Content: string(data), Status: 0})
}

// HandleWorkspace serves GET /api/workspace?agentId=xxx, returning the agent's workspace path.
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

// HandleURLPreviewProbe serves GET /api/url_preview_probe?url=xxx, checking whether the URL can be embedded safely.
func (p *ProxyServer) HandleURLPreviewProbe(w http.ResponseWriter, r *http.Request) {
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
	finalURL, reason, err := inspectURLPreviewTarget(ctx, p.Client, target, urlPreviewProbeAppOrigin(r))
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

func (p *ProxyServer) HandleWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	agentID := r.URL.Query().Get("agentId")
	if agentID == "" {
		http.Error(w, "agentId is required", http.StatusBadRequest)
		return
	}
	metadata, err := getAgentOutput(p.AgentDir, p.DeviceID, p.CacheTTL)
	if err != nil {
		log.Printf("agent scan error: %v", err)
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

// EditResponse is the JSON response for /api/edit.
type EditResponse struct {
	AgentID string `json:"agentId"`
	Path    string `json:"path"`
	SavedAs string `json:"savedAs,omitempty"`
	Content string `json:"content,omitempty"`
	Status  int    `json:"status"`
}

// EditRequest is the JSON body for /api/edit.
type EditRequest struct {
	Content string                 `json:"content"`
	Media   map[string]interface{} `json:"media,omitempty"`
}

func writeEditResponse(w http.ResponseWriter, resp EditResponse) {
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

// HandleEdit serves POST /api/edit?agentId=xxx&path=yyy, writing content to a file under the agent's workspace.
func (p *ProxyServer) HandleEdit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := r.URL.Query().Get("agentId")
	relPath := normalizeQuotedPathArg(r.URL.Query().Get("path"))
	if agentID == "" || relPath == "" {
		writeEditResponse(w, EditResponse{AgentID: agentID, Path: relPath, Content: "agentId and path are required", Status: 1})
		return
	}

	// Reject absolute paths
	if filepath.IsAbs(relPath) || strings.HasPrefix(relPath, "~") {
		writeEditResponse(w, EditResponse{AgentID: agentID, Path: relPath, Content: "只支持相对路径", Status: 1})
		return
	}
	cleanRelPath := filepath.Clean(relPath)
	if cleanRelPath == "." || cleanRelPath == "" || cleanRelPath == ".." || strings.HasPrefix(cleanRelPath, ".."+string(filepath.Separator)) {
		writeEditResponse(w, EditResponse{AgentID: agentID, Path: relPath, Content: "只支持 workspace 下的相对路径", Status: 1})
		return
	}
	saveAsNew := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("saveAsNew")), "true")

	// Find agent workspace
	metadata, err := getAgentOutput(p.AgentDir, p.DeviceID, p.CacheTTL)
	if err != nil {
		writeEditResponse(w, EditResponse{AgentID: agentID, Path: relPath, Content: "Agent 元数据获取失败: " + err.Error(), Status: 1})
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
		writeEditResponse(w, EditResponse{AgentID: agentID, Path: relPath, Content: "Agent 不存在: " + agentID, Status: 1})
		return
	}

	targetRelPath := cleanRelPath
	if saveAsNew {
		targetRelPath = appendTimestampToFilename(cleanRelPath, time.Now())
	}

	resolved, err := resolveCaseInsensitiveUnderRoot(workspace, targetRelPath)
	if err != nil {
		writeEditResponse(w, EditResponse{AgentID: agentID, Path: relPath, Content: "路径解析失败: " + err.Error(), Status: 1})
		return
	}
	if !ensurePathWithinRoot(workspace, resolved) {
		writeEditResponse(w, EditResponse{AgentID: agentID, Path: relPath, Content: "只支持 workspace 下的相对路径", Status: 1})
		return
	}

	// Check if target is a directory
	if info, err := os.Stat(resolved); err == nil && info.IsDir() {
		writeEditResponse(w, EditResponse{AgentID: agentID, Path: relPath, Content: "不支持写入目录", Status: 1})
		return
	}

	// Read request body
	var req EditRequest
	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		writeEditResponse(w, EditResponse{AgentID: agentID, Path: relPath, Content: "读取请求体失败", Status: 1})
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeEditResponse(w, EditResponse{AgentID: agentID, Path: relPath, Content: "请求体格式错误", Status: 1})
		return
	}

	// Ensure parent directory exists
	dir := filepath.Dir(resolved)
	if err := os.MkdirAll(dir, 0755); err != nil {
		writeEditResponse(w, EditResponse{AgentID: agentID, Path: relPath, Content: "创建目录失败: " + err.Error(), Status: 1})
		return
	}

	var writeData []byte
	if isBinaryFile(resolved) {
		decoded, err := base64.StdEncoding.DecodeString(req.Content)
		if err != nil {
			writeEditResponse(w, EditResponse{AgentID: agentID, Path: relPath, Content: "二进制内容不是合法的 base64", Status: 1})
			return
		}
		writeData = decoded
	} else {
		writeData = []byte(req.Content)
	}

	// Write file
	if err := os.WriteFile(resolved, writeData, 0644); err != nil {
		writeEditResponse(w, EditResponse{AgentID: agentID, Path: relPath, Content: "写入失败: " + err.Error(), Status: 1})
		return
	}

	writeEditResponse(w, EditResponse{AgentID: agentID, Path: relPath, SavedAs: resolved, Status: 0})
}

// DelResponse is the JSON response for /api/del.
type DelResponse struct {
	AgentID string `json:"agentId"`
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
	Status  int    `json:"status"`
}

func writeDelResponse(w http.ResponseWriter, resp DelResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleDel serves GET /api/del?agentId=xxx&path=yyy, deleting a file or directory under the agent's workspace.
func (p *ProxyServer) HandleDel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := r.URL.Query().Get("agentId")
	relPath := normalizeQuotedPathArg(r.URL.Query().Get("path"))
	if agentID == "" || relPath == "" {
		writeDelResponse(w, DelResponse{AgentID: agentID, Path: relPath, Content: "agentId and path are required", Status: 1})
		return
	}

	if filepath.IsAbs(relPath) || strings.HasPrefix(relPath, "~") {
		writeDelResponse(w, DelResponse{AgentID: agentID, Path: relPath, Content: "只支持相对路径", Status: 1})
		return
	}

	metadata, err := getAgentOutput(p.AgentDir, p.DeviceID, p.CacheTTL)
	if err != nil {
		writeDelResponse(w, DelResponse{AgentID: agentID, Path: relPath, Content: "Agent 元数据获取失败: " + err.Error(), Status: 1})
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
		writeDelResponse(w, DelResponse{AgentID: agentID, Path: relPath, Content: "Agent 不存在: " + agentID, Status: 1})
		return
	}

	fullPath := filepath.Join(workspace, relPath)
	resolved, err := ResolveCaseInsensitive(fullPath)
	if err != nil {
		writeDelResponse(w, DelResponse{AgentID: agentID, Path: relPath, Content: "不存在: " + err.Error(), Status: 1})
		return
	}

	if _, err := os.Stat(resolved); err != nil {
		writeDelResponse(w, DelResponse{AgentID: agentID, Path: relPath, Content: "不存在: " + err.Error(), Status: 1})
		return
	}

	// RemoveAll handles both files and directories (recursive)
	if err := os.RemoveAll(resolved); err != nil {
		writeDelResponse(w, DelResponse{AgentID: agentID, Path: relPath, Content: "删除失败: " + err.Error(), Status: 1})
		return
	}

	writeDelResponse(w, DelResponse{AgentID: agentID, Path: relPath, Status: 0})
}

// RawResponse is the JSON response for /api/raw.
type RawResponse struct {
	AgentID string `json:"agentId"`
	Path    string `json:"path"`
	Content string `json:"content"`
	Status  int    `json:"status"`
}

func writeRawResponse(w http.ResponseWriter, resp RawResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleRaw serves GET /api/raw?agentId=xxx&path=yyy, returning base64-encoded file content.
func (p *ProxyServer) HandleRaw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := r.URL.Query().Get("agentId")
	pathParam := normalizeQuotedPathArg(r.URL.Query().Get("path"))
	if agentID == "" || pathParam == "" {
		writeRawResponse(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "agentId and path are required", Status: 1})
		return
	}

	var resolved string
	expandedPath, err := ExpandPath(pathParam)
	if err != nil {
		writeRawResponse(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "路径解析失败: " + err.Error(), Status: 1})
		return
	}
	if filepath.IsAbs(expandedPath) {
		resolvedPath, resolveErr := ResolveCaseInsensitive(expandedPath)
		if resolveErr != nil {
			writeRawResponse(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "文件不存在: " + resolveErr.Error(), Status: 1})
			return
		}
		resolved = resolvedPath
	} else {
		cleanRelPath := filepath.Clean(pathParam)
		if cleanRelPath == "." || cleanRelPath == "" || cleanRelPath == ".." || strings.HasPrefix(cleanRelPath, ".."+string(filepath.Separator)) {
			writeRawResponse(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "只支持 workspace 下的相对路径", Status: 1})
			return
		}

		metadata, err := getAgentOutput(p.AgentDir, p.DeviceID, p.CacheTTL)
		if err != nil {
			writeRawResponse(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "Agent 元数据获取失败: " + err.Error(), Status: 1})
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
			writeRawResponse(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "Agent 不存在: " + agentID, Status: 1})
			return
		}

		resolvedPath, resolveErr := resolveCaseInsensitiveUnderRoot(workspace, cleanRelPath)
		if resolveErr != nil {
			writeRawResponse(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "文件不存在: " + resolveErr.Error(), Status: 1})
			return
		}
		resolved = resolvedPath
		if !ensurePathWithinRoot(workspace, resolved) {
			writeRawResponse(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "只支持 workspace 下的相对路径", Status: 1})
			return
		}
	}

	info, err := os.Stat(resolved)
	if err != nil {
		writeRawResponse(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "文件不存在: " + err.Error(), Status: 1})
		return
	}
	if info.IsDir() {
		writeRawResponse(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "不支持读取目录", Status: 1})
		return
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		writeRawResponse(w, RawResponse{AgentID: agentID, Path: pathParam, Content: "读取失败: " + err.Error(), Status: 1})
		return
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	writeRawResponse(w, RawResponse{AgentID: agentID, Path: pathParam, Content: encoded, Status: 0})
}

// ═══════════════════════════════════════════════════════════════════════════
// Heartbeat status tracking
// ═══════════════════════════════════════════════════════════════════════════

type HeartbeatStatus struct {
	mu            sync.Mutex
	LastTimestamp int64 `json:"lastTimestamp"`
	Status        int   `json:"status"` // 0=success, 1=failure
}

var hbStatus HeartbeatStatus

// RecordHeartbeat records the result of a heartbeat attempt.
// status: 0=success no task, 1=failure, 2=success with task
func RecordHeartbeat(status int) {
	hbStatus.mu.Lock()
	defer hbStatus.mu.Unlock()
	hbStatus.LastTimestamp = time.Now().UnixMilli()
	hbStatus.Status = status
}

// HandleHeartbeat serves GET /api/heartbeat, returning the last heartbeat time and status.
func (p *ProxyServer) HandleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	hbStatus.mu.Lock()
	resp := struct {
		LastTimestamp int64 `json:"lastTimestamp"`
		Status        int   `json:"status"`
	}{
		LastTimestamp: hbStatus.LastTimestamp,
		Status:        hbStatus.Status,
	}
	hbStatus.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// copyDir recursively copies a directory.
func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
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

func (p *ProxyServer) resolveDefaultDir() string {
	if p != nil {
		if value := strings.TrimSpace(p.DefaultDir); value != "" {
			return value
		}
	}
	if value := configuredProxyStringValue("default-dir"); value != "" {
		if abs, err := filepath.Abs(value); err == nil {
			return abs
		}
		return filepath.Clean(value)
	}
	appDir := proxyAppDir()
	if appDir == "" {
		return ""
	}
	return filepath.Join(appDir, "config")
}

// HandleAgentInit serves GET /api/agent/init?name=xxx, creating a new agent directory.
func (p *ProxyServer) HandleAgentInit(w http.ResponseWriter, r *http.Request) {
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
	// Validate name: no spaces, valid directory name
	if strings.ContainsAny(name, " /\\:*?\"<>|") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "name contains invalid characters"})
		return
	}

	agentDir := filepath.Join(p.AgentDir, name)
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
	defaultDir := p.resolveDefaultDir()
	info, err := os.Stat(defaultDir)
	if err != nil {
		_ = os.RemoveAll(agentDir)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "default-dir unavailable: " + err.Error()})
		return
	}
	if !info.IsDir() {
		_ = os.RemoveAll(agentDir)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "default-dir is not a directory: " + defaultDir})
		return
	}
	if err := copyDir(defaultDir, agentDir); err != nil {
		_ = os.RemoveAll(agentDir)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "failed to copy default-dir: " + err.Error()})
		return
	}

	// Flush agent cache
	flushAgentCache()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "name": name, "path": agentDir})
}

// HandleAgentDelete serves GET /api/agent/delete?name=xxx, deleting an agent directory.
func (p *ProxyServer) HandleAgentDelete(w http.ResponseWriter, r *http.Request) {
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
	agentDir := filepath.Join(p.AgentDir, name)
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
	db, err := getDataDB()
	if err == nil {
		if _, _, cleanupErr := deleteAgentCronData(db, name); cleanupErr != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "cleanup cron failed: " + cleanupErr.Error()})
			return
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		log.Printf("agent delete cleanup db skipped for %s: %v", name, err)
	}
	flushAgentCache()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "name": name})
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

// HandleAgentCreate serves GET /api/agent/create?agentId=xxx&name=yyy&type=zzz
func (p *ProxyServer) HandleAgentCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	agentID := r.URL.Query().Get("agentId")
	name := r.URL.Query().Get("name")
	createType := r.URL.Query().Get("type") // 0=dir, 1=file
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
	// Find agent workspace
	metadata, err := getAgentOutput(p.AgentDir, p.DeviceID, p.CacheTTL)
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
		// Create directory
		if err := os.MkdirAll(target, 0755); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "create dir failed: " + err.Error()})
			return
		}
	} else {
		// Create file
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

// HandleUpload serves POST /api/upload, uploading files/folders to agent tmp directory.
func (p *ProxyServer) HandleUpload(w http.ResponseWriter, r *http.Request) {
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

	metadata, err := getAgentOutput(p.AgentDir, p.DeviceID, p.CacheTTL)
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

	// Parse multipart form (max 200MB)
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

// AgentConfig represents the config.json structure.
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
			log.Printf("[proxy] rename provider log failed: %v", err)
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

func printProxyTokenHelp() {
	fmt.Println("Usage:")
	fmt.Println("  proxy token")
	fmt.Println("  proxy token --provider MODEL")
	fmt.Println("  proxy token --n N [--start 'YYYY-MM-DD HH:MM:SS'] [--close 'YYYY-MM-DD HH:MM:SS'] [--agentId ID]")
	fmt.Println("  proxy token get [--n N] [--start 'YYYY-MM-DD HH:MM:SS'] [--close 'YYYY-MM-DD HH:MM:SS'] [--agentId ID]")
	fmt.Println("  proxy token --agentId ID --model MODEL --function NAME --total N [--thinking N --input N --cache N --timestamp MS]")
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
	fmt.Println("  proxy token")
	fmt.Println("  proxy token --provider deepseek")
	fmt.Println("  proxy token --n 500")
	fmt.Println("  proxy token get --n 500 --start '2026-06-14 12:00:00' --close '2026-06-14 14:00:00'")
	fmt.Println("  proxy token --agentId demo-agent --model deepseek-chat --function cli/get --thinking 10 --input 20 --total 30 --cache 5")
}

func printProxyTokenGetHelp() {
	fmt.Println("Usage:")
	fmt.Println("  proxy token get [--n N] [--start 'YYYY-MM-DD HH:MM:SS'] [--close 'YYYY-MM-DD HH:MM:SS'] [--agentId ID]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --n N            查询最新 N 条本地 token 用量记录，默认 500")
	fmt.Println("  --start TIME     查询开始时间，支持 yyyyMMdd-hhmmss 或 YYYY-MM-DD HH:MM:SS")
	fmt.Println("  --close TIME     查询结束时间，支持 yyyyMMdd-hhmmss 或 YYYY-MM-DD HH:MM:SS，默认当前时间")
	fmt.Println("  --agentId ID     仅查询指定 AgentId 的 token 用量")
	fmt.Println("  --agent ID       AgentId 别名")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  proxy token get --n 500")
	fmt.Println("  proxy token get --n 500 --start '2026-06-14 12:00:00' --close '2026-06-14 14:00:00'")
}

func writeProxyTokenCLIOutput(w io.Writer, models map[string]tokenConfig, provider string) error {
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

func writeProxyTokenConsumeCLIOutput(w io.Writer, result tokenconsume.QueryResult) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(map[string]interface{}{
		"status":  0,
		"details": result.Details,
		"summary": result.Summary,
	})
}

func resolveProxyTokenQueryRange(startRaw, closeRaw string) (int64, int64, error) {
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

func runProxyTokenConsumeQuery(db *sql.DB, agentID, startRaw, closeRaw string, limit int) {
	if limit <= 0 {
		log.Fatal("n must be a positive integer")
	}
	startTimestamp, closeTimestamp, err := resolveProxyTokenQueryRange(startRaw, closeRaw)
	if err != nil {
		log.Fatal(err)
	}
	result, err := tokenconsume.QueryLatestRange(db, agentID, startTimestamp, closeTimestamp, limit)
	if err != nil {
		log.Fatal(err)
	}
	if err := writeProxyTokenConsumeCLIOutput(os.Stdout, result); err != nil {
		log.Fatal(err)
	}
}

func runProxyTokenGetCLI(args []string) {
	if hasHelpFlag(args) {
		printProxyTokenGetHelp()
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
	db, err := getDataDB()
	if err != nil {
		log.Fatal(err)
	}
	targetAgentID := strings.TrimSpace(*agentID)
	if targetAgentID == "" {
		targetAgentID = strings.TrimSpace(*agent)
	}
	runProxyTokenConsumeQuery(db, targetAgentID, *startRaw, *closeRaw, *limit)
}

func runProxyTokenCLI(args []string) {
	if len(args) > 0 && strings.EqualFold(strings.TrimSpace(args[0]), "get") {
		runProxyTokenGetCLI(args[1:])
		return
	}
	if hasHelpFlag(args) {
		printProxyTokenHelp()
		return
	}

	fs := flag.NewFlagSet("token", flag.ContinueOnError)
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

	db, err := getDataDB()
	if err != nil {
		log.Fatal(err)
	}
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
		runProxyTokenConsumeQuery(db, targetAgentID, *startRaw, *closeRaw, limit)
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
	if err := writeProxyTokenCLIOutput(os.Stdout, models, selectedProvider); err != nil {
		log.Fatal(err)
	}
}

func printProxySandboxHelp() {
	fmt.Println("Usage:")
	fmt.Println("  proxy sandbox --agentId ID --chatId ID")
	fmt.Println("  proxy sandbox --agentId ID --chatId ID --sandbox off|filepick|net|filepick_net")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --agentId ID      当前会话所属 AgentId")
	fmt.Println("  --chatId ID       当前会话所属 ChatId")
	fmt.Println("  --chat ID         ChatId 别名")
	fmt.Println("  --sandbox MODE    写入当前会话沙盒模式；off 表示关闭，不传则只查询")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  proxy sandbox --agentId A --chatId chat-001")
	fmt.Println("  proxy sandbox --agentId A --chatId chat-001 --sandbox filepick_net")
	fmt.Println("  proxy sandbox --agentId A --chatId chat-001 --sandbox off")
}

func runProxySandboxCLI(args []string) {
	if hasHelpFlag(args) {
		printProxySandboxHelp()
		return
	}

	fs := flag.NewFlagSet("proxy sandbox", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	agentID := fs.String("agentId", "", "agent id")
	chatID := fs.String("chatId", "", "chat id")
	chatAlias := fs.String("chat", "", "chat id")
	sandboxRaw := fs.String("sandbox", "", "sandbox mode")
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
	if targetAgentID == "" || targetChatID == "" {
		log.Fatal("agentId and chatId are required")
	}

	db, err := getDataDB()
	if err != nil {
		log.Fatal(err)
	}

	var (
		state    sandboxstate.State
		recorded bool
	)
	if strings.TrimSpace(*sandboxRaw) != "" {
		mode, err := parseSandboxUpdateValue(*sandboxRaw)
		if err != nil {
			log.Fatal(err)
		}
		state, err = sandboxstate.Set(db, targetAgentID, targetChatID, mode)
		if err != nil {
			log.Fatal(err)
		}
		recorded = strings.TrimSpace(state.Sandbox) != ""
	} else {
		var err error
		state, recorded, err = sandboxstate.Get(db, targetAgentID, targetChatID)
		if err != nil {
			log.Fatal(err)
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(map[string]interface{}{
		"status":    0,
		"agentId":   state.AgentID,
		"chatId":    state.ChatID,
		"sandbox":   state.Sandbox,
		"recorded":  recorded,
		"updatedAt": state.UpdatedAt,
	}); err != nil {
		log.Fatal(err)
	}
}

func (p *ProxyServer) HandleConsume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	db, err := getDataDB()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "db error: " + err.Error()})
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
	result, err := tokenconsume.QueryLatestRange(db, r.URL.Query().Get("agentId"), startTimestamp, closeTimestamp, limit)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "query error: " + err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "details": result.Details, "summary": result.Summary})
}

func proxySandboxTarget(r *http.Request) (string, string, error) {
	agentID := strings.TrimSpace(r.URL.Query().Get("agentId"))
	chatID := strings.TrimSpace(r.URL.Query().Get("chatId"))
	if chatID == "" {
		chatID = strings.TrimSpace(r.URL.Query().Get("chat"))
	}
	if agentID == "" || chatID == "" {
		return "", "", fmt.Errorf("agentId and chatId are required")
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

func proxySandboxModeFromRequestPath(r *http.Request) (string, bool, error) {
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

func writeProxySandboxState(w http.ResponseWriter, state sandboxstate.State, recorded bool) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    0,
		"agentId":   state.AgentID,
		"chatId":    state.ChatID,
		"sandbox":   state.Sandbox,
		"recorded":  recorded,
		"updatedAt": state.UpdatedAt,
	})
}

func (p *ProxyServer) handleSandboxQuery(w http.ResponseWriter, r *http.Request, allowSet bool) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	db, err := getDataDB()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "db error: " + err.Error()})
		return
	}

	agentID, chatID, err := proxySandboxTarget(r)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
		return
	}

	if allowSet {
		mode, shouldSet, err := proxySandboxModeFromRequestPath(r)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		if !shouldSet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "sandbox mode is required"})
			return
		}
		state, err := sandboxstate.Set(db, agentID, chatID, mode)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "save error: " + err.Error()})
			return
		}
		writeProxySandboxState(w, state, strings.TrimSpace(state.Sandbox) != "")
		return
	}

	state, found, err := sandboxstate.Get(db, agentID, chatID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "query error: " + err.Error()})
		return
	}
	writeProxySandboxState(w, state, found)
}

func (p *ProxyServer) HandleSandbox(w http.ResponseWriter, r *http.Request) {
	p.handleSandboxQuery(w, r, true)
}

func (p *ProxyServer) HandleSandboxStatus(w http.ResponseWriter, r *http.Request) {
	p.handleSandboxQuery(w, r, false)
}

func validateRegisteredModel(db *sql.DB, model string) error {
	model = sharedutil.NormalizeTokenModelName(model)
	if model == "" {
		return fmt.Errorf("model is required")
	}
	models, _, err := readTokenStore(db)
	if err != nil {
		return fmt.Errorf("query token failed: %w", err)
	}
	cfg, ok := models[model]
	if !ok {
		return fmt.Errorf("model not registered: %s", model)
	}
	if strings.TrimSpace(cfg.Token) == "" {
		return fmt.Errorf("model token is empty: %s", model)
	}
	return nil
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

// HandleSwarm serves POST /api/config?agentId=xxx, creating or updating config.json.
func (p *ProxyServer) HandleSwarm(w http.ResponseWriter, r *http.Request) {
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
			db, err := getDataDB()
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "db error: " + err.Error()})
				return
			}
			if err := deleteTokenStoreModel(db, actionPayload.Model); err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "delete model failed: " + err.Error()})
				return
			}
			models, updatedAt, err := readTokenStore(db)
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
	metadata, err := getAgentOutput(p.AgentDir, p.DeviceID, p.CacheTTL)
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

func (p *ProxyServer) HandleToken(w http.ResponseWriter, r *http.Request) {
	db, err := getDataDB()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "db error: " + err.Error()})
		return
	}

	switch r.Method {
	case http.MethodGet:
		models, updatedAt, err := readTokenStore(db)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "query error: " + err.Error()})
			return
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
		if err := writeTokenStoreChanges(db, models, removedModels); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "save error: " + err.Error()})
			return
		}
		updatedModels, updatedAt, err := readTokenStore(db)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "query error: " + err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "models": updatedModels, "updatedAt": updatedAt})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleDownload serves GET /api/download?path=xxx, downloading a file or directory (as zip).
func (p *ProxyServer) HandleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pathParam := r.URL.Query().Get("path")
	if pathParam == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}
	resolved, err := ResolveCaseInsensitive(pathParam)
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

// HandleCron serves POST /api/cron/create?agentId=xxx, creating a cron task.
func (p *ProxyServer) HandleCron(w http.ResponseWriter, r *http.Request) {
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
	if err := validateProxyAgentExists(p.AgentDir, p.DeviceID, agentID); err != nil {
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

type cronMetaFilter struct {
	AgentID   string
	ChatID    string
	TaskType  string
	Model     string
	Content   string
	Cycle     *int
	StartFrom *time.Time
	StartTo   *time.Time
}

type cronDetailFilter struct {
	MetaID   *int
	AgentID  string
	ChatID   string
	TaskType string
	Model    string
	Content  string
	Cycle    *int
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
	RouterDisable  bool   `json:"router_disable"`
	Cron           string `json:"cron"`
	Content        string `json:"content"`
	ResponseSchema string `json:"responseSchema"`
	ChatID         string `json:"chatId,omitempty"`
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
	RouterDisable  bool
	Content        string
	ResponseSchema string
	Started        int
	OccurredAt     string
}

type cronQueryOptions struct {
	MetaID   string
	DetailID string
	ID       string
	AgentID  string
	ChatID   string
	TaskType string
	Model    string
	Content  string
	Cycle    *int
	Time     string
	Date     string
	From     string
	To       string
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
			DetailID:      detail.ID,
			MetaID:        detail.MetaID,
			AgentID:       detail.AgentID,
			ChatID:        detail.ChatID,
			TaskType:      detail.Type,
			Action:        "delete",
			ExecTime:      detail.ExecTime,
			Model:         detail.Model,
			Thinking:      detail.Thinking,
			RouterDisable: detail.RouterDisable,
			Content:       detail.Content,
			Started:       detail.Started,
			OccurredAt:    cronLogTimestamp(),
		})
	}
	for _, metaID := range metaIDs {
		if item, ok := metaMap[metaID]; ok {
			appendCronMetaLog(db, cronMetaLogEntry{
				MetaID:        item.ID,
				AgentID:       item.AgentID,
				ChatID:        item.ChatID,
				TaskType:      item.Type,
				Action:        "delete",
				Cycle:         item.Cycle,
				RawTime:       item.RawTime,
				Model:         item.Model,
				Thinking:      item.Thinking,
				RouterDisable: item.RouterDisable,
				Cron:          item.Cron,
				Content:       item.Content,
				CreatedAt:     cronLogTimestamp(),
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
}, metaIDs []int) (map[int]cronMetaResult, error) {
	result := make(map[int]cronMetaResult, len(metaIDs))
	if len(metaIDs) == 0 {
		return result, nil
	}
	selectRouterDisable := `1 AS router_disable`
	if db, ok := queryer.(*sql.DB); ok {
		selectRouterDisable = cronRouterDisableSelect(db, "task_meta")
	}
	rows, err := queryer.Query(fmt.Sprintf(`SELECT id, cycle, raw_time, agent_id, model, thinking, %s, cron, content, response_schema, chat_id, task_type FROM task_meta WHERE id IN (%s)`, selectRouterDisable, placeholders(len(metaIDs))), intsToInterfaces(metaIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item cronMetaResult
		var th, routerDisable int
		if err := rows.Scan(&item.ID, &item.Cycle, &item.RawTime, &item.AgentID, &item.Model, &th, &routerDisable, &item.Cron, &item.Content, &item.ResponseSchema, &item.ChatID, &item.Type); err != nil {
			return nil, err
		}
		item.Type = sharedutil.NormalizeTaskType(item.Type)
		item.Thinking = th != 0
		item.RouterDisable = routerDisable != 0
		result[item.ID] = item
	}
	return result, rows.Err()
}

func loadCronDetailsByMetaIDs(queryer interface {
	Query(string, ...interface{}) (*sql.Rows, error)
}, metaIDs []int, unfinishedOnly bool) ([]cronDetailResult, error) {
	if len(metaIDs) == 0 {
		return []cronDetailResult{}, nil
	}
	selectRouterDisable := `1 AS router_disable`
	if db, ok := queryer.(*sql.DB); ok {
		selectRouterDisable = cronRouterDisableSelectExpr(db, "task_detail", "d.router_disable")
	}
	query := fmt.Sprintf(`SELECT d.id, d.meta_id, d.exec_time, d.agent_id, d.chat_id, d.task_type, d.model, d.thinking, %s, d.content, d.response_schema, d.started, m.cycle, m.raw_time, m.cron
		FROM task_detail d
		LEFT JOIN task_meta m ON m.id = d.meta_id
		WHERE d.meta_id IN (%s)`, selectRouterDisable, placeholders(len(metaIDs)))
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
		var th, routerDisable int
		if err := rows.Scan(&item.ID, &item.MetaID, &item.ExecTime, &item.AgentID, &item.ChatID, &item.Type, &item.Model, &th, &routerDisable, &item.Content, &item.ResponseSchema, &item.Started, &item.Cycle, &item.RawTime, &item.Cron); err != nil {
			return nil, err
		}
		item.Type = sharedutil.NormalizeTaskType(item.Type)
		item.Thinking = th != 0
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
	rows, err := db.Query(fmt.Sprintf(`SELECT d.id, d.meta_id, d.exec_time, d.agent_id, d.chat_id, d.task_type, d.model, d.thinking, %s, d.content, d.response_schema, d.started, m.cycle, m.raw_time, m.cron
		FROM task_detail d
		LEFT JOIN task_meta m ON m.id = d.meta_id
		WHERE d.id IN (%s)`, cronRouterDisableSelectExpr(db, "task_detail", "d.router_disable"), placeholders(len(detailIDs))), intsToInterfaces(detailIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item cronDetailResult
		var th, routerDisable int
		if err := rows.Scan(&item.ID, &item.MetaID, &item.ExecTime, &item.AgentID, &item.ChatID, &item.Type, &item.Model, &th, &routerDisable, &item.Content, &item.ResponseSchema, &item.Started, &item.Cycle, &item.RawTime, &item.Cron); err != nil {
			return nil, err
		}
		item.Type = sharedutil.NormalizeTaskType(item.Type)
		item.Thinking = th != 0
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
	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}

	for _, item := range detailItems {
		appendCronDetailLog(db, cronDetailLogEntry{
			DetailID:      item.ID,
			MetaID:        item.MetaID,
			AgentID:       item.AgentID,
			ChatID:        item.ChatID,
			TaskType:      item.Type,
			Action:        "delete_agent",
			ExecTime:      item.ExecTime,
			Model:         item.Model,
			Thinking:      item.Thinking,
			RouterDisable: item.RouterDisable,
			Content:       item.Content,
			Started:       item.Started,
			OccurredAt:    cronLogTimestamp(),
		})
	}
	for _, item := range metaItems {
		appendCronMetaLog(db, cronMetaLogEntry{
			MetaID:        item.ID,
			AgentID:       item.AgentID,
			ChatID:        item.ChatID,
			TaskType:      item.Type,
			Action:        "delete_agent",
			Cycle:         item.Cycle,
			RawTime:       item.RawTime,
			Model:         item.Model,
			Thinking:      item.Thinking,
			RouterDisable: item.RouterDisable,
			Cron:          item.Cron,
			Content:       item.Content,
			CreatedAt:     cronLogTimestamp(),
		})
	}
	return metaAffected, detailAffected, nil
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
	execFrom, execTo, err := resolveTimeRange(opts.Time, opts.Date, opts.From, opts.To, true)
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
	return buildCronMetaFilter(cronQueryOptions{
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
}

func parseCronDetailFilterFromRequest(r *http.Request) (cronDetailFilter, error) {
	q := r.URL.Query()
	cycle, err := parseOptionalIntValue(q.Get("cycle"))
	if err != nil {
		return cronDetailFilter{}, err
	}
	return buildCronDetailFilter(cronQueryOptions{
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

func queryCronMetas(db *sql.DB, filter cronMetaFilter) ([]cronMetaResult, error) {
	query := fmt.Sprintf(`SELECT id, cycle, raw_time, agent_id, model, thinking, %s, cron, content, response_schema, chat_id, task_type FROM task_meta WHERE 1=1`, cronRouterDisableSelect(db, "task_meta"))
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
		query += ` AND content LIKE ?`
		args = append(args, "%"+filter.Content+"%")
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
	query += ` ORDER BY id`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	metas := make([]cronMetaResult, 0)
	for rows.Next() {
		var (
			item          cronMetaResult
			th            int
			routerDisable int
		)
		if err := rows.Scan(&item.ID, &item.Cycle, &item.RawTime, &item.AgentID, &item.Model, &th, &routerDisable, &item.Cron, &item.Content, &item.ResponseSchema, &item.ChatID, &item.Type); err != nil {
			return nil, err
		}
		item.Type = sharedutil.NormalizeTaskType(item.Type)
		item.Thinking = th != 0
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
	query := fmt.Sprintf(`SELECT d.id, d.meta_id, d.exec_time, d.agent_id, d.chat_id, d.meta_ref, d.task_type, d.model, d.thinking, %s, d.content, %s, %s, %s, d.started, m.cycle, m.raw_time, m.cron
		FROM task_detail d
		INNER JOIN task_meta m ON m.id = d.meta_id
		WHERE 1=1`, cronRouterDisableSelectExpr(db, "task_detail", "d.router_disable"), selectResponseSchema, selectResultContent, selectRepliedAt)
	args := make([]interface{}, 0, 12)
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
		query += ` AND d.content LIKE ?`
		args = append(args, "%"+filter.Content+"%")
	}
	if filter.Cycle != nil {
		query += ` AND m.cycle = ?`
		args = append(args, *filter.Cycle)
	}
	if filter.ExecFrom != nil {
		query += ` AND d.exec_time >= ?`
		args = append(args, filter.ExecFrom.Unix())
	}
	if filter.ExecTo != nil {
		query += ` AND d.exec_time <= ?`
		args = append(args, filter.ExecTo.Unix())
	}
	query += ` ORDER BY d.exec_time, d.id`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	details := make([]cronDetailResult, 0)
	for rows.Next() {
		var (
			item          cronDetailResult
			th            int
			routerDisable int
		)
		if err := rows.Scan(&item.ID, &item.MetaID, &item.ExecTime, &item.AgentID, &item.ChatID, &item.MetaRef, &item.Type, &item.Model, &th, &routerDisable, &item.Content, &item.ResponseSchema, &item.ResultContent, &item.RepliedAt, &item.Started, &item.Cycle, &item.RawTime, &item.Cron); err != nil {
			return nil, err
		}
		item.Type = sharedutil.NormalizeTaskType(item.Type)
		item.Thinking = th != 0
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

func appendCronMetaLog(db *sql.DB, entry cronMetaLogEntry) {
	if db == nil {
		return
	}
	ensureCronRouterDisableColumn(db, "cron_meta_log")
	hasRouterDisable := hasTableColumn(db, "cron_meta_log", "router_disable")
	hasResponseSchema := hasTableColumn(db, "cron_meta_log", "response_schema")
	query := `INSERT INTO cron_meta_log (meta_id, agent_id, chat_id, action, cycle, raw_time, model, thinking, router_disable, cron, content, response_schema, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`
	args := []any{entry.MetaID, entry.AgentID, entry.ChatID, entry.Action, entry.Cycle, entry.RawTime, entry.Model, boolToInt(entry.Thinking), boolToInt(entry.RouterDisable), entry.Cron, entry.Content, entry.ResponseSchema, entry.CreatedAt}
	switch {
	case hasRouterDisable && hasResponseSchema:
	case hasRouterDisable:
		query = `INSERT INTO cron_meta_log (meta_id, agent_id, chat_id, action, cycle, raw_time, model, thinking, router_disable, cron, content, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`
		args = []any{entry.MetaID, entry.AgentID, entry.ChatID, entry.Action, entry.Cycle, entry.RawTime, entry.Model, boolToInt(entry.Thinking), boolToInt(entry.RouterDisable), entry.Cron, entry.Content, entry.CreatedAt}
	case hasResponseSchema:
		query = `INSERT INTO cron_meta_log (meta_id, agent_id, chat_id, action, cycle, raw_time, model, thinking, cron, content, response_schema, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`
		args = []any{entry.MetaID, entry.AgentID, entry.ChatID, entry.Action, entry.Cycle, entry.RawTime, entry.Model, boolToInt(entry.Thinking), entry.Cron, entry.Content, entry.ResponseSchema, entry.CreatedAt}
	default:
		query = `INSERT INTO cron_meta_log (meta_id, agent_id, chat_id, action, cycle, raw_time, model, thinking, cron, content, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`
		args = []any{entry.MetaID, entry.AgentID, entry.ChatID, entry.Action, entry.Cycle, entry.RawTime, entry.Model, boolToInt(entry.Thinking), entry.Cron, entry.Content, entry.CreatedAt}
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
	ensureCronRouterDisableColumn(db, "cron_detail_log")
	hasRouterDisable := hasTableColumn(db, "cron_detail_log", "router_disable")
	hasResponseSchema := hasTableColumn(db, "cron_detail_log", "response_schema")
	query := `INSERT INTO cron_detail_log (detail_id, meta_id, agent_id, chat_id, action, exec_time, model, thinking, router_disable, content, response_schema, started, occurred_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`
	args := []any{entry.DetailID, entry.MetaID, entry.AgentID, entry.ChatID, entry.Action, entry.ExecTime, entry.Model, boolToInt(entry.Thinking), boolToInt(entry.RouterDisable), entry.Content, entry.ResponseSchema, entry.Started, entry.OccurredAt}
	switch {
	case hasRouterDisable && hasResponseSchema:
	case hasRouterDisable:
		query = `INSERT INTO cron_detail_log (detail_id, meta_id, agent_id, chat_id, action, exec_time, model, thinking, router_disable, content, started, occurred_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`
		args = []any{entry.DetailID, entry.MetaID, entry.AgentID, entry.ChatID, entry.Action, entry.ExecTime, entry.Model, boolToInt(entry.Thinking), boolToInt(entry.RouterDisable), entry.Content, entry.Started, entry.OccurredAt}
	case hasResponseSchema:
		query = `INSERT INTO cron_detail_log (detail_id, meta_id, agent_id, chat_id, action, exec_time, model, thinking, content, response_schema, started, occurred_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`
		args = []any{entry.DetailID, entry.MetaID, entry.AgentID, entry.ChatID, entry.Action, entry.ExecTime, entry.Model, boolToInt(entry.Thinking), entry.Content, entry.ResponseSchema, entry.Started, entry.OccurredAt}
	default:
		query = `INSERT INTO cron_detail_log (detail_id, meta_id, agent_id, chat_id, action, exec_time, model, thinking, content, started, occurred_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`
		args = []any{entry.DetailID, entry.MetaID, entry.AgentID, entry.ChatID, entry.Action, entry.ExecTime, entry.Model, boolToInt(entry.Thinking), entry.Content, entry.Started, entry.OccurredAt}
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
	selectMetaRef := `'' AS meta_ref`
	if hasTableColumn(db, "task_detail", "meta_ref") {
		selectMetaRef = `meta_ref`
	}
	selectResponseSchema := `'' AS response_schema`
	if hasTableColumn(db, "task_detail", "response_schema") {
		selectResponseSchema = `response_schema`
	}
	row := db.QueryRow(fmt.Sprintf(`SELECT id, meta_id, exec_time, agent_id, chat_id, %s, model, thinking, %s, content, %s, started FROM task_detail WHERE id = ?`,
		selectMetaRef, cronRouterDisableSelect(db, "task_detail"), selectResponseSchema), detailID)
	var d cronDetailResult
	var th, routerDisable int
	if err := row.Scan(&d.ID, &d.MetaID, &d.ExecTime, &d.AgentID, &d.ChatID, &d.MetaRef, &d.Model, &th, &routerDisable, &d.Content, &d.ResponseSchema, &d.Started); err != nil {
		return
	}
	d.Thinking = th != 0
	d.RouterDisable = routerDisable != 0
	appendCronDetailLog(db, cronDetailLogEntry{
		DetailID:       d.ID,
		MetaID:         d.MetaID,
		AgentID:        d.AgentID,
		ChatID:         d.ChatID,
		Action:         action,
		ExecTime:       d.ExecTime,
		Model:          d.Model,
		Thinking:       d.Thinking,
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

// HandleCronMetadata serves POST /api/cron/detail/metadata with optional filters.
func (p *ProxyServer) HandleCronMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	db, err := getDataDB()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "db error: " + err.Error()})
		return
	}
	filter, err := parseCronMetaFilterFromRequest(r)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
		return
	}
	metas, err := queryCronMetas(db, filter)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "query error: " + err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "data": metas})
}

// HandleCronDelete serves POST /api/cron/delete with explicit id or optional filters.
func (p *ProxyServer) HandleCronDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	db, err := getDataDB()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "db error: " + err.Error()})
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
	affected, deletedIDs, err := deleteCronMetas(db, filter, explicitID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "delete error: " + err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "affected": affected, "ids": deletedIDs})
}

// HandleCronDetailDelete serves POST /api/cron/detail/delete with explicit id or optional filters.
func (p *ProxyServer) HandleCronDetailDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	db, err := getDataDB()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "db error: " + err.Error()})
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
	affected, deletedIDs, err := deleteCronDetails(db, filter, explicitDetailID, explicitMetaID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "delete error: " + err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "affected": affected, "ids": deletedIDs})
}

// HandleCronDetailList serves POST /api/cron/detail/list with optional filters.
func (p *ProxyServer) HandleCronDetailList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	db, err := getDataDB()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "db error: " + err.Error()})
		return
	}
	filter, err := parseCronDetailFilterFromRequest(r)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": err.Error()})
		return
	}
	details, err := queryCronDetails(db, filter)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "query error: " + err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "data": details})
}

// HandleCronDetailStatus serves POST /api/cron/detail/status?agentId=xxx&detailId=yyy&status=zzz
func (p *ProxyServer) HandleCronDetailStatus(w http.ResponseWriter, r *http.Request) {
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
	db, err := getDataDB()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "db error: " + err.Error()})
		return
	}
	res, err := db.Exec(`UPDATE task_detail SET started = ? WHERE id = ? AND agent_id = ?`, status, detailID, agentID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "update error: " + err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if detailInt, parseErr := strconv.Atoi(detailID); parseErr == nil && n > 0 {
		logCronDetailStatusByID(db, detailInt, "update_status")
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "affected": n})
}

// HandleRestore serves POST /api/restore?agentId=xxx&chat=yyy&timeline=zzz
func (p *ProxyServer) HandleRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	agentID := r.URL.Query().Get("agentId")
	chatID := r.URL.Query().Get("chat")
	timeline := r.URL.Query().Get("timeline")
	lastIDStr := r.URL.Query().Get("lastId")
	if agentID == "" || chatID == "" || timeline == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "agentId, chat and timeline are required"})
		return
	}
	lastID := 0
	if lastIDStr != "" {
		if parsed, err := strconv.Atoi(lastIDStr); err == nil && parsed > 0 {
			lastID = parsed
		}
	}
	db, err := getDataDB()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "db error: " + err.Error()})
		return
	}
	initChatTable(db)
	query := `SELECT id, agent_id, chat_id, chat_type, role, response_type, content, created_at FROM chat_log WHERE agent_id = ? AND chat_id = ? AND created_at > ? ORDER BY id`
	args := []interface{}{agentID, chatID, timeline}
	if lastID > 0 {
		query = `SELECT id, agent_id, chat_id, chat_type, role, response_type, content, created_at FROM chat_log WHERE agent_id = ? AND chat_id = ? AND (created_at > ? OR (created_at = ? AND id > ?)) ORDER BY id`
		args = []interface{}{agentID, chatID, timeline, timeline, lastID}
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "query error: " + err.Error()})
		return
	}
	defer rows.Close()
	type Record struct {
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
	var records []Record
	for rows.Next() {
		var rec Record
		rows.Scan(&rec.ID, &rec.AgentID, &rec.ChatID, &rec.ChatType, &rec.Role, &rec.ResponseType, &rec.Content, &rec.CreatedAt)
		rec.ChatType = normalizeChatType(rec.ChatType)
		rec.ResponseType = normalizeResponseType(rec.ResponseType)
		records = append(records, rec)
	}
	eventLogger, err := getEventLogger()
	if err == nil && eventLogger != nil {
		entries, queryErr := eventLogger.Query(agentID, chatID, timeline, eventlog.TypeCLIGet, eventlog.TypeCLIPub)
		if queryErr == nil {
			for _, item := range entries {
				rec := Record{
					ID:           item.ID,
					AgentID:      item.AgentID,
					ChatID:       item.ChatID,
					ChatType:     chatTypePageSession,
					ResponseType: "normal",
					Content:      item.Content,
					LogType:      item.Type,
					CreatedAt:    item.CreatedAt,
				}
				if item.Type == eventlog.TypeCLIGet {
					rec.Role = "cli/get"
				} else {
					rec.Role = "cli/pub"
				}
				records = append(records, rec)
			}
		}
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].CreatedAt == records[j].CreatedAt {
			return records[i].ID < records[j].ID
		}
		return records[i].CreatedAt < records[j].CreatedAt
	})
	if records == nil {
		records = []Record{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "data": records})
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

func (p *ProxyServer) getWorkspaceByAgentID(agentID string) (string, error) {
	metadata, err := getAgentOutput(p.AgentDir, p.DeviceID, p.CacheTTL)
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
	if value == "" {
		return ""
	}
	layouts := []string{
		roundLogStorageTimeLayout,
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
	if opts.Round == 0 && startRaw == "" {
		opts.Round = 1
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
	if opts.Start != "" && opts.Close != "" {
		if opts.Close < opts.Start {
			return opts, fmt.Errorf("close must be greater than or equal to start")
		}
	}
	return opts, nil
}

func filterRoundLogEntries(entries []eventlog.Entry, opts roundLogOptions) []eventlog.Entry {
	if len(entries) == 0 {
		return []eventlog.Entry{}
	}
	startIdx := 0
	if opts.Round > 0 {
		requestIndexes := make([]int, 0)
		for idx, item := range entries {
			if item.Type == eventlog.TypeChatCompletionRequest {
				requestIndexes = append(requestIndexes, idx)
			}
		}
		if len(requestIndexes) == 0 {
			return []eventlog.Entry{}
		}
		startReq := len(requestIndexes) - opts.Round
		if startReq < 0 {
			startReq = 0
		}
		startIdx = requestIndexes[startReq]
	}

	filtered := make([]eventlog.Entry, 0, len(entries)-startIdx)
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

func filterChatSessionLogEntries(entries []eventlog.Entry, startValue, closeValue string) []eventlog.Entry {
	if len(entries) == 0 {
		return []eventlog.Entry{}
	}
	filtered := make([]eventlog.Entry, 0, len(entries))
	for _, item := range entries {
		if item.Type != eventlog.TypeChatCompletionRequest && item.Type != eventlog.TypeChatCompletionResponse {
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

func buildChatSessionLogRecords(entries []eventlog.Entry, limit int) []chatSessionLogRecord {
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
			LogType:      item.Type,
			LogTypeLabel: buildRoundTypeLabel(item.Type),
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

func sanitizeLogRoundFilePart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "empty"
	}
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"?", "_",
		"&", "_",
		"=", "_",
		" ", "_",
	)
	return replacer.Replace(value)
}

func buildRoundTypeLabel(logType int) string {
	switch logType {
	case eventlog.TypeChatCompletionRequest:
		return "SSE请求"
	case eventlog.TypeChatCompletionResponse:
		return "SSE响应"
	case eventlog.TypeCLIGet:
		return "工具请求（cli/get）"
	case eventlog.TypeCLIPub:
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
		payload, ok := parseRoundLogJSON(data)
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
	case eventlog.TypeChatCompletionRequest:
		return extractRoundLogRequestContent(content)
	case eventlog.TypeChatCompletionResponse:
		return extractRoundLogResponseContent(content)
	case eventlog.TypeCLIGet:
		return extractRoundLogCLIGetContent(content)
	case eventlog.TypeCLIPub:
		return extractRoundLogCLIPubContent(content)
	default:
		return strings.TrimSpace(content)
	}
}

func mergeSSEContent(records []roundLogRecord) []roundLogRecord {
	if len(records) == 0 {
		return records
	}
	out := make([]roundLogRecord, 0, len(records))
	for _, item := range records {
		if item.LogType != eventlog.TypeChatCompletionResponse {
			out = append(out, item)
			continue
		}
		last := len(out) - 1
		if last >= 0 && out[last].LogType == eventlog.TypeChatCompletionResponse {
			out[last].Content += item.Content
			continue
		}
		out = append(out, item)
	}
	return out
}

func buildRoundMarkdown(records []roundLogRecord) string {
	var builder strings.Builder
	builder.WriteString("| 时间 | 类型 | 具体内容 |\n")
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

func exportRoundLog(p *ProxyServer, agentID, chatID string, opts roundLogOptions) (string, error) {
	agentID = strings.TrimSpace(agentID)
	chatID = strings.TrimSpace(chatID)
	if agentID == "" || chatID == "" {
		return "", fmt.Errorf("agentId and chatId are required")
	}
	normalizedOpts, err := normalizeRoundLogOptions(opts)
	if err != nil {
		return "", err
	}

	logger, err := getEventLogger()
	if err != nil {
		return "", err
	}
	entries, err := logger.QueryAll(agentID, chatID)
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
		formattedContent := formatRoundLogContent(item.Type, item.Content)
		if item.Type == eventlog.TypeChatCompletionResponse && strings.TrimSpace(formattedContent) == "" {
			continue
		}
		records = append(records, roundLogRecord{
			ID:        item.ID,
			AgentID:   item.AgentID,
			ChatID:    item.ChatID,
			LogType:   item.Type,
			TypeLabel: buildRoundTypeLabel(item.Type),
			Content:   formattedContent,
			CreatedAt: item.CreatedAt,
		})
	}
	records = mergeSSEContent(records)

	workspace, err := p.getWorkspaceByAgentID(agentID)
	if err != nil {
		return "", err
	}
	tmpDir := filepath.Join(workspace, "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%s_%s_%s.md",
		sanitizeLogRoundFilePart(agentID),
		sanitizeLogRoundFilePart(chatID),
		roundLogTimestamp(),
	)
	path := filepath.Join(tmpDir, filename)
	if err := os.WriteFile(path, []byte(buildRoundMarkdown(records)), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func collectRoundGroups(entries []eventlog.Entry) [][]eventlog.Entry {
	if len(entries) == 0 {
		return nil
	}
	rounds := make([][]eventlog.Entry, 0)
	var current []eventlog.Entry
	for _, item := range entries {
		if item.Type == eventlog.TypeChatCompletionRequest {
			if len(current) > 0 {
				rounds = append(rounds, current)
			}
			current = []eventlog.Entry{item}
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

func isCompleteSSERound(entries []eventlog.Entry) bool {
	for _, item := range entries {
		if item.Type == eventlog.TypeChatCompletionResponse {
			return true
		}
	}
	return false
}

func latestCompleteSSERound(entries []eventlog.Entry) []eventlog.Entry {
	rounds := collectRoundGroups(entries)
	for idx := len(rounds) - 1; idx >= 0; idx-- {
		if isCompleteSSERound(rounds[idx]) {
			return rounds[idx]
		}
	}
	return nil
}

func countCLIGetInEntries(entries []eventlog.Entry) int {
	count := 0
	for _, item := range entries {
		if item.Type == eventlog.TypeCLIGet {
			count++
		}
	}
	return count
}

func querySkillStatus(agentID, chatID string, threshold int) (bool, error) {
	agentID = strings.TrimSpace(agentID)
	chatID = strings.TrimSpace(chatID)
	if agentID == "" || chatID == "" {
		return false, fmt.Errorf("agentId and chatId are required")
	}
	if threshold <= 0 {
		threshold = 10
	}
	logger, err := getEventLogger()
	if err != nil {
		return false, err
	}
	entries, err := logger.QueryAll(agentID, chatID)
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

func (p *ProxyServer) HandleLogSkill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	agentID := strings.TrimSpace(r.URL.Query().Get("agentId"))
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

	path, err := exportRoundLog(p, agentID, chatID, opts)
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

func (p *ProxyServer) HandleLogSkillStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	agentID := strings.TrimSpace(r.URL.Query().Get("agentId"))
	chatID := strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("chatId"), r.URL.Query().Get("chat")))
	round := resolveProxySkillExtractRound()
	if roundStr := strings.TrimSpace(r.URL.Query().Get("round")); roundStr != "" {
		value, err := strconv.Atoi(roundStr)
		if err != nil || value <= 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": "round must be a positive integer"})
			return
		}
		round = value
	}
	hasSkill, err := querySkillStatus(agentID, chatID, round)
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

func (p *ProxyServer) HandleChatSessionLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	agentID := strings.TrimSpace(r.URL.Query().Get("agentId"))
	chatID := strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("chatId"), r.URL.Query().Get("chat")))
	if agentID == "" || chatID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": "agentId and chatId are required"})
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
	logger, err := getEventLogger()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": err.Error()})
		return
	}
	entries, err := logger.QueryAll(agentID, chatID)
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

	db, err := getDataDB()
	if err != nil {
		return nil, fmt.Errorf("db error: %w", err)
	}
	if err := validateRegisteredModel(db, req.Model); err != nil {
		return nil, err
	}

	var cronExpr string
	var rawTime string
	thinkInt := 0
	routerDisableInt := 0
	if req.Thinking {
		thinkInt = 1
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
		if req.Cycle == 0 {
			nowTrunc := time.Now().Truncate(time.Minute)
			if t.Before(nowTrunc) {
				return nil, fmt.Errorf("一次性任务执行时间不能早于当前时间")
			}
		}
	}

	res, err := db.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, model, thinking, router_disable, cron, content, response_schema, chat_id, task_type) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		req.Cycle, rawTime, agentID, req.Model, thinkInt, routerDisableInt, cronExpr, req.Content, req.ResponseSchema, req.ChatID, req.Type)
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
		RouterDisable:  req.RouterDisable,
		Cron:           cronExpr,
		Content:        req.Content,
		ResponseSchema: req.ResponseSchema,
		CreatedAt:      cronLogTimestamp(),
	})

	if req.Cycle >= 0 && req.Cycle <= 2 && rawTime != "" {
		t, _ := time.ParseInLocation("2006-01-02 15:04", rawTime, time.Local)
		detailRes, _ := db.Exec(`INSERT OR IGNORE INTO task_detail (meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, router_disable, content, response_schema, started) VALUES (?,?,?,?,?,?,?,?,?,?,0)`,
			metaID, t.Unix(), agentID, req.ChatID, req.Type, req.Model, thinkInt, routerDisableInt, req.Content, req.ResponseSchema)
		if detailRes != nil {
			if n, _ := detailRes.RowsAffected(); n > 0 {
				if detailID, err := detailRes.LastInsertId(); err == nil {
					appendCronDetailLog(db, cronDetailLogEntry{DetailID: int(detailID), MetaID: int(metaID), AgentID: agentID, ChatID: req.ChatID, TaskType: req.Type, Action: "insert", ExecTime: t.Unix(), Model: req.Model, Thinking: req.Thinking, RouterDisable: req.RouterDisable, Content: req.Content, ResponseSchema: req.ResponseSchema, Started: 0, OccurredAt: cronLogTimestamp()})
				}
			}
		}
		if req.Cycle == 1 || req.Cycle == 2 {
			end := t.Add(5 * 24 * time.Hour)
			day := t.AddDate(0, 0, 1)
			for !day.After(end) {
				if req.Cycle == 2 || (day.Weekday() >= time.Monday && day.Weekday() <= time.Friday) {
					detailRes, _ := db.Exec(`INSERT OR IGNORE INTO task_detail (meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, router_disable, content, response_schema, started) VALUES (?,?,?,?,?,?,?,?,?,?,0)`,
						metaID, day.Unix(), agentID, req.ChatID, req.Type, req.Model, thinkInt, routerDisableInt, req.Content, req.ResponseSchema)
					if detailRes != nil {
						if n, _ := detailRes.RowsAffected(); n > 0 {
							if detailID, err := detailRes.LastInsertId(); err == nil {
								appendCronDetailLog(db, cronDetailLogEntry{DetailID: int(detailID), MetaID: int(metaID), AgentID: agentID, ChatID: req.ChatID, TaskType: req.Type, Action: "insert", ExecTime: day.Unix(), Model: req.Model, Thinking: req.Thinking, RouterDisable: req.RouterDisable, Content: req.Content, ResponseSchema: req.ResponseSchema, Started: 0, OccurredAt: cronLogTimestamp()})
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
		now := time.Now().Truncate(time.Minute)
		end := now.Add(5 * 24 * time.Hour)
		tick := now
		for !tick.After(end) {
			detailRes, _ := db.Exec(`INSERT OR IGNORE INTO task_detail (meta_id, exec_time, agent_id, chat_id, task_type, model, thinking, router_disable, content, response_schema, started) VALUES (?,?,?,?,?,?,?,?,?,?,0)`,
				metaID, tick.Unix(), agentID, req.ChatID, req.Type, req.Model, thinkInt, routerDisableInt, req.Content, req.ResponseSchema)
			if detailRes != nil {
				if n, _ := detailRes.RowsAffected(); n > 0 {
					if detailID, err := detailRes.LastInsertId(); err == nil {
						appendCronDetailLog(db, cronDetailLogEntry{DetailID: int(detailID), MetaID: int(metaID), AgentID: agentID, ChatID: req.ChatID, TaskType: req.Type, Action: "insert", ExecTime: tick.Unix(), Model: req.Model, Thinking: req.Thinking, RouterDisable: req.RouterDisable, Content: req.Content, ResponseSchema: req.ResponseSchema, Started: 0, OccurredAt: cronLogTimestamp()})
					}
				}
			}
			tick = tick.Add(interval)
		}
	}

	return map[string]interface{}{"status": 0, "id": metaID, "cron": cronExpr, "agentId": agentID, "type": req.Type}, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Active connection manager (AgentId+Chat dimension)
// ═══════════════════════════════════════════════════════════════════════════

type chatMsg struct {
	role         string // Q, A, X
	content      string
	responseType string
}

type activeConn struct {
	cancel  context.CancelFunc
	ch      chan chatMsg
	agentID string
	chatID  string
}

var (
	connMu  sync.Mutex
	connMap = make(map[string]*activeConn) // key: agentId+"|"+chatId
)

func connKey(agentID, chatID string) string { return agentID + "|" + chatID }

const (
	chatTypePageSession   = "page_session"
	chatTypeScheduledTask = "scheduled_task"
)

func initChatTable(db *sql.DB) {
	db.Exec(`CREATE TABLE IF NOT EXISTS chat_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		agent_id TEXT NOT NULL,
		chat_id TEXT NOT NULL,
		chat_type TEXT NOT NULL DEFAULT 'page_session',
		role TEXT NOT NULL,
		response_type TEXT NOT NULL DEFAULT 'normal',
		content TEXT NOT NULL,
		created_at TEXT NOT NULL
	)`)
	db.Exec(`ALTER TABLE chat_log ADD COLUMN chat_type TEXT NOT NULL DEFAULT 'page_session'`)
	db.Exec(`ALTER TABLE chat_log ADD COLUMN response_type TEXT NOT NULL DEFAULT 'normal'`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_agent_chat ON chat_log(agent_id, chat_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_agent_chat_time ON chat_log(agent_id, chat_id, created_at)`)
	initCmdLogTable(db)
}

func appendEventLog(agentID, chatID, content string, logType int, createdAt string) {
	logger, err := getEventLogger()
	if err != nil {
		return
	}
	logger.Append(eventlog.Entry{
		AgentID:   agentID,
		ChatID:    chatID,
		Content:   content,
		Type:      logType,
		CreatedAt: createdAt,
	})
}

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

func appendChatLog(agentID, chatID, chatType, role, responseType, content string) {
	db, err := getDataDB()
	if err != nil {
		return
	}
	initChatTable(db)
	createdAt := time.Now().Format("2006-01-02T15:04:05.000")
	db.Exec(`INSERT INTO chat_log (agent_id, chat_id, chat_type, role, response_type, content, created_at) VALUES (?,?,?,?,?,?,?)`,
		agentID, chatID, normalizeChatType(chatType), role, normalizeResponseType(responseType), content, createdAt)
	switch role {
	case "Q":
		appendEventLog(agentID, chatID, content, eventlog.TypeChatCompletionRequest, createdAt)
	case "A":
		appendEventLog(agentID, chatID, content, eventlog.TypeChatCompletionResponse, createdAt)
	}
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
		if requestText == "" || requestText == "[image]" || requestText == "[file]" {
			continue
		}
		parts = append(parts, requestText)
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

func createImmediateCronTaskFromConnect(db *sql.DB, meta *connectsvc.Meta, reqs []connectsvc.Request, started int) (dueTask, string, error) {
	if db == nil {
		return dueTask{}, "", fmt.Errorf("db is required")
	}
	result, err := connectsvc.CreateImmediateCronTask(meta, reqs, started, func(items []connectsvc.Request) string {
		return buildConnectRequestTaskContent(items)
	}, connectsvc.ImmediateCronTaskPersistence{
		InsertMeta: func(seed *connectsvc.ImmediateCronTaskSeed) (int, error) {
			taskType := sharedutil.NormalizeTaskType(seed.TaskType)
			res, err := db.Exec(`INSERT INTO task_meta (cycle, raw_time, agent_id, chat_id, task_type, model, thinking, router_disable, cron, content, response_schema) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
				0, seed.RawTime, seed.AgentID, seed.ChatID, taskType, seed.Model, boolToInt(seed.Thinking), boolToInt(seed.RouterDisable), "", seed.Content, seed.ResponseSchema)
			if err != nil {
				return 0, err
			}
			id, _ := res.LastInsertId()
			return int(id), nil
		},
		AppendMetaLog: func(metaID int, seed *connectsvc.ImmediateCronTaskSeed) {
			appendCronMetaLog(db, cronMetaLogEntry{
				MetaID:         metaID,
				AgentID:        seed.AgentID,
				ChatID:         seed.ChatID,
				TaskType:       sharedutil.NormalizeTaskType(seed.TaskType),
				Action:         "insert",
				Cycle:          0,
				RawTime:        seed.RawTime,
				Model:          seed.Model,
				Thinking:       seed.Thinking,
				RouterDisable:  seed.RouterDisable,
				Cron:           "",
				Content:        seed.Content,
				ResponseSchema: seed.ResponseSchema,
				CreatedAt:      cronLogTimestamp(),
			})
		},
		InsertDetail: func(metaID int, chatID string, seed *connectsvc.ImmediateCronTaskSeed) (int, error) {
			taskType := sharedutil.NormalizeTaskType(seed.TaskType)
			res, err := db.Exec(`INSERT OR IGNORE INTO task_detail (meta_id, exec_time, agent_id, chat_id, meta_ref, task_type, model, thinking, router_disable, content, response_schema, started) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
				metaID, seed.ExecTime, seed.AgentID, chatID, seed.MetaRef, taskType, seed.Model, boolToInt(seed.Thinking), boolToInt(seed.RouterDisable), seed.Content, seed.ResponseSchema, seed.Started)
			if err != nil {
				return 0, err
			}
			id, _ := res.LastInsertId()
			return int(id), nil
		},
		AppendDetailLog: func(detailID, metaID int, chatID string, seed *connectsvc.ImmediateCronTaskSeed) {
			appendCronDetailLog(db, cronDetailLogEntry{
				DetailID:       detailID,
				MetaID:         metaID,
				AgentID:        seed.AgentID,
				ChatID:         chatID,
				TaskType:       sharedutil.NormalizeTaskType(seed.TaskType),
				Action:         "insert",
				ExecTime:       seed.ExecTime,
				Model:          seed.Model,
				Thinking:       seed.Thinking,
				RouterDisable:  seed.RouterDisable,
				Content:        seed.Content,
				ResponseSchema: seed.ResponseSchema,
				Started:        seed.Started,
				OccurredAt:     cronLogTimestamp(),
			})
		},
	})
	if err != nil || result == nil {
		return dueTask{}, "", err
	}
	return dueTask{
		ID:             result.DetailID,
		MetaID:         result.MetaID,
		ExecTime:       result.Task.ExecTime,
		AgentID:        result.Task.AgentID,
		ChatID:         result.ChatID,
		MetaRef:        result.Task.MetaRef,
		TaskType:       sharedutil.NormalizeTaskType(result.Task.TaskType),
		Model:          result.Task.Model,
		Thinking:       result.Task.Thinking,
		RouterDisable:  result.Task.RouterDisable,
		Content:        result.Task.Content,
		ResponseSchema: result.Task.ResponseSchema,
	}, result.Task.LastRequestID, nil
}

var runConnectListMeta = func() ([]connectMetaConfigListItem, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve proxy executable: %w", err)
	}
	if strings.HasSuffix(strings.TrimSpace(exe), ".test") {
		svc, err := connectsvc.NewService(connectsvc.Options{
			DBPath:   "data",
			AgentDir: resolveDefaultAgentDirArg(),
			CacheTTL: 10 * time.Second,
		})
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
	cmd := exec.Command(exe, "connect", "meta-list", "--db", "data", "--agent-dir", resolveDefaultAgentDirArg())
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) == 0 {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	var items []connectMetaConfigListItem
	if err := json.Unmarshal(output, &items); err != nil {
		return nil, fmt.Errorf("decode proxy connect meta-list output: %w", err)
	}
	return items, nil
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

func runConnectPluginAction(callbackPath, action string, flags map[string]string) error {
	return connectsvc.RunPluginCallbackAction(callbackPath, action, flags)
}

var runConnectPluginInit = func(callbackPath string, flags map[string]string) error {
	return runConnectPluginAction(callbackPath, "init", flags)
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
	return runConnectPluginAction(callbackPath, "send", flags)
}

func notifyConnectPluginStarted(callbacks map[string]string, meta connectsvc.Meta, request connectsvc.Request, reply string) error {
	return connectsvc.NotifyPluginStarted(callbacks, meta, request, reply, connectsvc.PluginCallbackRuntimeOptions{
		ListMeta:     runConnectListMeta,
		NormalizeKey: sharedutil.NormalizeTaskType,
		ResolveConnectBin: func() string {
			return resolveExecutablePath(os.Args[0])
		},
		Init:   runConnectPluginInit,
		Logger: log.Printf,
	})
}

func extractSSEAssistantText(payload []byte) string {
	return trimLeadingSSETextPunctuation(connectsvc.ExtractSSEAssistantText(payload))
}

func normalizePluginReplyContent(raw string) string {
	return connectsvc.NormalizePluginReplyContent(raw)
}

func trimLeadingSSETextPunctuation(text string) string {
	return strings.TrimLeft(strings.TrimSpace(text), "。.,，")
}

func notifyCompletedConnectTasks(p *ProxyServer, db *sql.DB, svc *connectsvc.Service, callbacks map[string]string) {
	if p == nil || db == nil || svc == nil {
		return
	}
	connectsvc.DispatchCompletedPluginReplies(callbacks, db, svc, sharedutil.DefaultTaskType, time.Now().Add(-24*time.Hour).Unix(), func(detailID int) error {
		_, err := db.Exec(`UPDATE task_detail SET replied_at = ? WHERE id = ?`, time.Now().Format(time.RFC3339), detailID)
		return err
	}, connectsvc.PluginCallbackRuntimeOptions{
		ListMeta:     runConnectListMeta,
		NormalizeKey: sharedutil.NormalizeTaskType,
		ResolveConnectBin: func() string {
			return resolveExecutablePath(os.Args[0])
		},
		Send:   runConnectPluginSend,
		Logger: log.Printf,
	})
}

func syncConnectPendingRequests(p *ProxyServer) {
	svc, err := p.newConnectService()
	if err != nil {
		log.Printf("[connect-cron] init service failed: %v", err)
		return
	}
	defer svc.Close()

	db, err := getDataDB()
	if err != nil {
		log.Printf("[connect-cron] open db failed: %v", err)
		return
	}

	callbacks, err := listConnectPluginCallbacks()
	if err != nil {
		log.Printf("[connect-cron] list meta config failed: %v", err)
		callbacks = map[string]string{}
	}
	policy := connectsvc.PendingRequestPolicy{
		MinNotifyAge:       0,
		ExpireAge:          20 * time.Minute,
		PendingDetailState: 1,
		ExpiredDetailState: 2,
	}
	connectsvc.SyncPendingPluginRuntime(callbacks, connectsvc.PendingPluginRuntimeOptions{
		Service:          svc,
		QueryDB:          db,
		Logger:           log.Printf,
		BuildTextContent: buildConnectRequestTextContent,
		DecidePlan: func(meta connectsvc.Meta, reqs []connectsvc.Request, now time.Time, textContent string) (*connectsvc.PendingRequestPlan, error) {
			return connectsvc.DecidePendingPluginPlan(meta, reqs, now, textContent, policy)
		},
		CreateTask: func(meta *connectsvc.Meta, reqs []connectsvc.Request, detailStarted int) (any, string, error) {
			return createImmediateCronTaskFromConnect(db, meta, reqs, detailStarted)
		},
		AfterCreate: func(meta connectsvc.Meta, selected []connectsvc.Request, plan connectsvc.PendingRequestPlan, created any) error {
			if !plan.Notify {
				return nil
			}
			task, ok := created.(dueTask)
			if !ok {
				return nil
			}
			if task.ID > 0 {
				runDueTask(p, db, task, true)
			}
			return nil
		},
		NotifyStarted: func(callbacks map[string]string, meta connectsvc.Meta, request connectsvc.Request) error {
			return notifyConnectPluginStarted(callbacks, meta, request, p.AutoReply)
		},
		DispatchCompletedReplies: func(callbacks map[string]string, svc *connectsvc.Service) {
			notifyCompletedConnectTasks(p, db, svc, callbacks)
		},
	})
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

type dueTask struct {
	ID             int
	MetaID         int
	ExecTime       int64
	AgentID        string
	ChatID         string
	MetaRef        string
	TaskType       string
	Model          string
	Thinking       bool
	RouterDisable  bool
	Content        string
	ResponseSchema string
}

func runDueTask(p *ProxyServer, db *sql.DB, task dueTask, alreadyStarted bool) bool {
	if p == nil || db == nil {
		return false
	}
	if strings.TrimSpace(p.Host) == "" || p.Client == nil {
		return false
	}

	task.TaskType = sharedutil.NormalizeTaskType(task.TaskType)
	if !agentExists(p.AgentDir, p.DeviceID, task.AgentID, p.CacheTTL) {
		appendCronDetailLog(db, cronDetailLogEntry{
			DetailID:   task.ID,
			MetaID:     task.MetaID,
			AgentID:    task.AgentID,
			ChatID:     task.ChatID,
			TaskType:   task.TaskType,
			Action:     "skip_deleted_agent",
			ExecTime:   task.ExecTime,
			Model:      task.Model,
			Thinking:   task.Thinking,
			Content:    task.Content,
			Started:    boolToInt(alreadyStarted),
			OccurredAt: cronLogTimestamp(),
		})
		return false
	}
	if err := validateRegisteredModel(db, task.Model); err != nil {
		appendCronDetailLog(db, cronDetailLogEntry{
			DetailID:   task.ID,
			MetaID:     task.MetaID,
			AgentID:    task.AgentID,
			ChatID:     task.ChatID,
			TaskType:   task.TaskType,
			Action:     "skip_invalid_model",
			ExecTime:   task.ExecTime,
			Model:      task.Model,
			Thinking:   task.Thinking,
			Content:    task.Content,
			Started:    boolToInt(alreadyStarted),
			OccurredAt: cronLogTimestamp(),
		})
		return false
	}

	if !alreadyStarted {
		res, err := db.Exec(`UPDATE task_detail SET started = 1 WHERE id = ? AND started = 0`, task.ID)
		if err != nil {
			return false
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return false
		}
		logCronDetailStatusByID(db, task.ID, "update_started")
	}

	chatID := strings.TrimSpace(task.ChatID)
	if chatID == "" {
		chatID = fmt.Sprintf("%d@%d", task.MetaID, task.ID)
		_, _ = db.Exec(`UPDATE task_detail SET chat_id = ? WHERE id = ?`, chatID, task.ID)
		logCronDetailStatusByID(db, task.ID, "update_chat_id")
	}

	metadata, err := getAgentOutputForChat(p.AgentDir, p.DeviceID, p.CacheTTL, chatID)
	if err != nil {
		if !alreadyStarted {
			_, _ = db.Exec(`UPDATE task_detail SET started = 0 WHERE id = ?`, task.ID)
			logCronDetailStatusByID(db, task.ID, "reset_started")
		}
		return false
	}
	metaBytes, _ := json.Marshal(metadata)
	var metaMap map[string]interface{}
	_ = json.Unmarshal(metaBytes, &metaMap)
	metaMap["agentId"] = task.AgentID
	metaMap["chat"] = chatID
	metaMap["type"] = chatTypeScheduledTask
	metaMap["cron_type"] = task.TaskType
	metaMap["thinking"] = task.Thinking
	metaMap["router_disable"] = task.RouterDisable
	mediaByAgentID := injectLiveAgentMediaIntoAgentList(metaMap)
	selectedAgent := selectedAgentMetadata(metadata, task.AgentID)
	attachLiveAgentMedia(selectedAgent, workspaceForAgent(metadata, task.AgentID), mediaByAgentID[strings.TrimSpace(task.AgentID)])
	metaMap["agent"] = selectedAgent
	if routerRemote := readAgentRouterRemote(p.AgentDir, p.DeviceID, task.AgentID, p.CacheTTL); len(routerRemote) > 0 {
		metaMap["router_remote"] = routerRemote
	}
	if metaID := lastMetaRefID(task.MetaRef); metaID != "" {
		metaMap["META_ID"] = metaID
	}
	if schema := strings.TrimSpace(task.ResponseSchema); schema != "" {
		metaMap["response_schema"] = schema
	}

	reqData := map[string]interface{}{
		"model":    task.Model,
		"messages": []map[string]string{{"role": "user", "content": task.Content}},
		"stream":   true,
		"metadata": metaMap,
	}
	if strings.TrimSpace(task.ResponseSchema) != "" {
		reqData["response_format"] = map[string]interface{}{
			"type": "json_schema",
			"json_schema": map[string]interface{}{
				"name":   "scheduled_task_response",
				"schema": json.RawMessage(task.ResponseSchema),
			},
		}
	}
	syncForwardedChatRequestFlags(reqData, metaMap)
	body, _ := json.Marshal(reqData)

	appendChatLog(task.AgentID, chatID, chatTypeScheduledTask, "Q", "normal", string(body))
	token, err := lookupTokenByModel(db, task.Model)
	if err != nil {
		if !alreadyStarted {
			_, _ = db.Exec(`UPDATE task_detail SET started = 0 WHERE id = ?`, task.ID)
			logCronDetailStatusByID(db, task.ID, "reset_started")
		}
		return false
	}

	detailID := task.ID
	agentID := task.AgentID
	go func() {
		targetURL := strings.TrimRight(p.Host, "/") + "/v1/chat/completions"
		req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewReader(body))
		if err != nil {
			if !alreadyStarted {
				_, _ = db.Exec(`UPDATE task_detail SET started = 0 WHERE id = ?`, detailID)
				logCronDetailStatusByID(db, detailID, "reset_started")
			}
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")
		if token != "" {
			req.Header.Set("Authorization", token)
		}

		resp, err := p.Client.Do(req)
		if err != nil {
			if !alreadyStarted {
				_, _ = db.Exec(`UPDATE task_detail SET started = 0 WHERE id = ?`, detailID)
				logCronDetailStatusByID(db, detailID, "reset_started")
			}
			return
		}
		defer resp.Body.Close()

		payload, readErr := io.ReadAll(resp.Body)
		if len(payload) > 0 {
			appendChatLog(agentID, chatID, chatTypeScheduledTask, "A", detectResponseType(string(payload)), string(payload))
		}
		if readErr != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
			if !alreadyStarted {
				_, _ = db.Exec(`UPDATE task_detail SET started = 0 WHERE id = ?`, detailID)
				logCronDetailStatusByID(db, detailID, "reset_started")
			}
			return
		}
		resultContent := extractSSEAssistantText(payload)
		_, _ = db.Exec(`UPDATE task_detail SET started = 3, result_content = ? WHERE id = ?`, resultContent, detailID)
		logCronDetailStatusByID(db, detailID, "complete")
	}()
	return true
}

func cronExecuteOnce(p *ProxyServer) {
	db, err := getDataDB()
	if err != nil {
		return
	}

	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	rows, err := db.Query(fmt.Sprintf(`SELECT id, meta_id, exec_time, agent_id, chat_id, meta_ref, task_type, model, thinking, %s, content, response_schema FROM task_detail WHERE started = 0 AND exec_time <= ? AND exec_time >= ? ORDER BY exec_time, id`,
		cronRouterDisableSelect(db, "task_detail")),
		now.Unix(), oneHourAgo.Unix())
	if err != nil {
		return
	}
	defer rows.Close()

	var tasks []dueTask
	for rows.Next() {
		var item dueTask
		var thinking, routerDisable int
		if err := rows.Scan(&item.ID, &item.MetaID, &item.ExecTime, &item.AgentID, &item.ChatID, &item.MetaRef, &item.TaskType, &item.Model, &thinking, &routerDisable, &item.Content, &item.ResponseSchema); err != nil {
			continue
		}
		item.TaskType = sharedutil.NormalizeTaskType(item.TaskType)
		item.Thinking = thinking != 0
		item.RouterDisable = routerDisable != 0
		tasks = append(tasks, item)
	}

	for _, task := range tasks {
		runDueTask(p, db, task, false)
	}
}

func startConnectPendingRequestSync(p *ProxyServer) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		syncConnectPendingRequests(p)
		for range ticker.C {
			syncConnectPendingRequests(p)
		}
	}()
	log.Println("[connect] pending request sync started (every 30 seconds)")
}

func startCronExecutor(p *ProxyServer) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		if err := syncSkillsWarnings(p); err != nil {
			log.Printf("[skills_warning] initial sync failed: %v", err)
		}
		cronExecuteOnce(p)
		for range ticker.C {
			if err := syncSkillsWarnings(p); err != nil {
				log.Printf("[skills_warning] periodic sync failed: %v", err)
			}
			cronExecuteOnce(p)
		}
	}()
	log.Println("[cron] task executor started (every 1 minute)")
}

// HandleCancel serves POST /api/cancel?agentId=xxx&chat=yyy
func (p *ProxyServer) HandleCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	agentID := r.URL.Query().Get("agentId")
	chatID := r.URL.Query().Get("chat")
	if agentID == "" || chatID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 1, "content": "agentId and chat are required"})
		return
	}
	key := connKey(agentID, chatID)
	connMu.Lock()
	ac, exists := connMap[key]
	if exists {
		// Send cancel marker then close channel
		if ac.ch != nil {
			ac.ch <- chatMsg{role: "X", content: ""}
			close(ac.ch)
			ac.ch = nil
		}
		ac.cancel()
		delete(connMap, key)
	}
	connMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": 0, "cancelled": exists})
}

func (p *ProxyServer) processKnowledgeMetadata(metaMap map[string]interface{}, now time.Time, forceLastUpdate bool) error {
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
	knowledgePath, pathErr := proxyKnowledgeDirForAgent(p.AgentDir, agentID)
	if pathErr != nil {
		return fmt.Errorf("resolve knowledge path: %w", pathErr)
	}
	knowledge["path"] = knowledgePath
	var (
		lastUpdate int64
		err        error
	)
	if agentID != "" {
		db, openErr := knowledgecore.OpenSharedDB(proxyAppDir())
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

	interval := p.KnowledgeUpdateInterval
	if interval <= 0 {
		interval = 2 * time.Hour
	}
	lockWindow := p.KnowledgeUpdateLock
	if lockWindow <= 0 {
		lockWindow = 30 * time.Minute
	}

	nowMs := now.UnixMilli()
	if nowMs-lastUpdate <= interval.Milliseconds() {
		delete(knowledge, "lastUpdate")
		return nil
	}

	db, err := getDataDB()
	if err != nil {
		return err
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
	db, err := knowledgecore.OpenSharedDB(proxyAppDir())
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
	db, err := knowledgecore.OpenSharedDB(proxyAppDir())
	if err != nil {
		return fmt.Errorf("open knowledge runtime: %w", err)
	}
	if err := knowledgecore.SetKnowledgeCommitForAgent(db, agentID, value); err != nil {
		return fmt.Errorf("set knowledge commit: %w", err)
	}
	return nil
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
	syncForwardedBoolField(reqData, metaMap, "html")
	syncForwardedBoolField(reqData, metaMap, "router_disable")
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

// HandleChatCompletions proxies /v1/chat/completions with metadata injection.
func (p *ProxyServer) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read original request body
	bodyBytes, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	// Parse as generic JSON map
	var reqData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &reqData); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// Inject metadata
	_, requestedChatID := requestChatContext(reqData["metadata"])
	metadata, err := getAgentOutputForChat(p.AgentDir, p.DeviceID, p.CacheTTL, requestedChatID)
	if err != nil {
		log.Printf("agent scan error: %v", err)
		http.Error(w, "Failed to get agent metadata", http.StatusInternalServerError)
		return
	}

	// Convert AgentOutput to map for injection
	metaBytes, _ := json.Marshal(metadata)
	var metaMap map[string]interface{}
	json.Unmarshal(metaBytes, &metaMap)

	// Merge request metadata onto the shared Agent metadata and forward the
	// merged body as-is without any query-based metadata expansion.
	sharedutil.MergeMetadataFields(metaMap, reqData["metadata"])
	if db, err := getDataDB(); err != nil {
		log.Printf("provider config lookup failed: %v", err)
		http.Error(w, "Failed to read provider config", http.StatusInternalServerError)
		return
	} else if modelName, _ := reqData["model"].(string); strings.TrimSpace(modelName) != "" {
		modelCfg, err := lookupTokenConfigByModel(db, modelName)
		if err != nil {
			log.Printf("provider config lookup failed: %v", err)
			http.Error(w, "Failed to read provider config", http.StatusInternalServerError)
			return
		}
		injectConfiguredModelMetadata(metaMap, modelCfg)
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
			log.Printf("knowledge commit persist error: %v", err)
			http.Error(w, "Failed to persist knowledge commit", http.StatusInternalServerError)
			return
		}
		if knowledgeMeta, ok := metaMap["knowledge"].(map[string]interface{}); ok {
			knowledgeMeta["knowledgeCommit"] = knowledgeCommit
		}
	}
	if err := p.processKnowledgeMetadata(metaMap, time.Now(), knowledgeCommit); err != nil {
		log.Printf("knowledge metadata process error: %v", err)
		http.Error(w, "Failed to process knowledge metadata", http.StatusInternalServerError)
		return
	}
	mediaByAgentID := injectLiveAgentMediaIntoAgentList(metaMap)
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
		selectedAgent := selectedAgentMetadata(metadata, chatAgentID)
		attachLiveAgentMedia(selectedAgent, workspaceForAgent(metadata, chatAgentID), mediaByAgentID[strings.TrimSpace(chatAgentID)])
		mm["agent"] = selectedAgent
	}

	if err := normalizeForwardedChatRequest(reqData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Cancel any existing connection for this agent+chat
	key := connKey(chatAgentID, chatID)
	if chatAgentID != "" && chatID != "" {
		connMu.Lock()
		if old, exists := connMap[key]; exists {
			old.cancel()
			delete(connMap, key)
		}
		connMu.Unlock()
	}

	// Create cancellable context (not tied to client disconnect)
	ctx, cancel := context.WithCancel(context.Background())

	// Marshal modified body
	newBody, err := json.Marshal(reqData)
	if err != nil {
		cancel()
		http.Error(w, "Failed to marshal request", http.StatusInternalServerError)
		return
	}

	targetURL := strings.TrimRight(p.Host, "/") + "/v1/chat/completions"
	proxyReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(newBody))
	if err != nil {
		cancel()
		http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
		return
	}

	// Start async writer goroutine and register active connection early so
	// /api/cancel can hit even before the first SSE chunk arrives.
	var ch chan chatMsg
	if chatAgentID != "" && chatID != "" {
		ch = make(chan chatMsg, 64)
		go func() {
			for msg := range ch {
				if msg.role == "X" {
					appendChatLog(chatAgentID, chatID, chatType, "X", msg.responseType, "")
					return
				}
				if msg.role == "A" && msg.content != "" {
					appendChatLog(chatAgentID, chatID, chatType, "A", msg.responseType, msg.content)
				}
			}
		}()

		connMu.Lock()
		connMap[key] = &activeConn{cancel: cancel, ch: ch, agentID: chatAgentID, chatID: chatID}
		connMu.Unlock()
	}

	// Copy original headers unchanged
	for k, vv := range r.Header {
		for _, v := range vv {
			proxyReq.Header.Add(k, v)
		}
	}
	proxyReq.Header.Set("Content-Length", fmt.Sprintf("%d", len(newBody)))
	proxyReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.Client.Do(proxyReq)
	if err != nil {
		if chatAgentID != "" && chatID != "" {
			connMu.Lock()
			if ac, exists := connMap[key]; exists {
				delete(connMap, key)
				if ac.ch != nil {
					close(ac.ch)
					ac.ch = nil
				}
			}
			connMu.Unlock()
		}
		cancel()
		if chatAgentID != "" && chatID != "" {
			appendChatLog(chatAgentID, chatID, chatType, "A", "abnormal", "Failed to forward request: "+err.Error())
		}
		http.Error(w, "Failed to forward request", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers unchanged
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	// Ensure SSE headers are set to prevent Go's ResponseWriter from
	// buffering the first 512 bytes for Content-Type sniffing.
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "text/event-stream")
	}
	// Disable any proxy/CDN buffering
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	// Save the normalized forwarded request body so request logs match the
	// metadata that actually reached the upstream proxy boundary.
	if chatAgentID != "" && chatID != "" {
		go appendChatLog(chatAgentID, chatID, chatType, "Q", "normal", string(newBody))
	}

	w.WriteHeader(resp.StatusCode)

	// Stream response: read and flush each chunk, async write to SQLite
	flusher, ok := w.(http.Flusher)
	if !ok {
		io.Copy(w, resp.Body)
		cancel()
		return
	}

	buf := make([]byte, 256)
	logBuf := make([]byte, 0, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
			flusher.Flush()
			if ch != nil {
				logBuf = append(logBuf, buf[:n]...)
				for _, event := range splitCompleteSSEEvents(&logBuf) {
					ch <- chatMsg{role: "A", content: event, responseType: detectResponseType(event)}
				}
			}
		}
		if err != nil {
			break
		}
	}
	if ch != nil {
		for _, event := range flushTrailingSSEBytes(&logBuf) {
			ch <- chatMsg{role: "A", content: event, responseType: detectResponseType(event)}
		}
		close(ch)
	}
	if knowledgeCommit {
		if err := updateKnowledgeManualTimestamps(time.Now(), chatAgentID); err != nil {
			log.Printf("knowledge manual update failed: %v", err)
		}
	}

	// Clean up connection
	if chatAgentID != "" && chatID != "" {
		connMu.Lock()
		delete(connMap, key)
		connMu.Unlock()
	}
	cancel()
}

// HandleInstallApp serves GET /install_app, returning required app install keys.
func (p *ProxyServer) HandleInstallApp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	configured := configuredProxyInstallApps()
	configured = mergeInstallApps(configured, strings.Join(p.InstallApps, ","))
	apps := proxyFilterMissingInstallApps(mergeInstallApps(detectRequiredInstallApps(), strings.Join(configured, ",")))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=300")
	json.NewEncoder(w).Encode(apps)
}

// ═══════════════════════════════════════════════════════════════════════════
// CLI entry point
// ═══════════════════════════════════════════════════════════════════════════

type serveOptions struct {
	port                      int
	host                      string
	agentDir                  string
	defaultDir                string
	device                    string
	agentCacheMs              int
	connectCacheMs            int
	siteDir                   string
	connectTimeoutMs          int
	knowledgeUpdateIntervalMs int
	knowledgeUpdateLockMs     int
	installApp                string
	reply                     string
	pluginExecTimeoutMs       int
}

const defaultConnectAutoReply = connectsvc.DefaultAutoReply

func resolveServeReply(reply string) string {
	return connectsvc.ResolveAutoReply(reply)
}

func validateProxyServicePort(port int) error {
	if port != proxyServicePort {
		return fmt.Errorf("--port only supports %d", proxyServicePort)
	}
	return nil
}

func validateServeOptions(opts serveOptions) error {
	return validateProxyServicePort(opts.port)
}

func flagProvided(fs *flag.FlagSet, name string) bool {
	provided := false
	fs.Visit(func(f *flag.Flag) {
		if f != nil && f.Name == name {
			provided = true
		}
	})
	return provided
}

func writeProxyStartupConfig(data map[string]interface{}) {
	raw, path, err := readProxyStartupConfigRaw()
	if err != nil {
		log.Printf("config/config.json: read failed: %v", err)
		return
	}
	if raw == nil {
		raw = map[string]interface{}{}
	}
	for key, value := range data {
		raw[key] = value
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Printf("config/config.json: prepare dir failed: %v", err)
		return
	}
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		log.Printf("config/config.json: marshal failed: %v", err)
		return
	}
	if err := os.WriteFile(path, append(out, '\n'), 0644); err != nil {
		log.Printf("config/config.json: write failed: %v", err)
	}
}

func proxyAppDir() string {
	if appDir := configuredProxyStringValue("app-dir"); appDir != "" {
		if abs, err := filepath.Abs(appDir); err == nil {
			return abs
		}
		return filepath.Clean(appDir)
	}
	if appPath := configuredProxyStringValue("app"); appPath != "" {
		dir := filepath.Dir(appPath)
		if abs, err := filepath.Abs(dir); err == nil {
			return abs
		}
		return filepath.Clean(dir)
	}
	if cwd, err := os.Getwd(); err == nil {
		if abs, err := filepath.Abs(cwd); err == nil {
			return abs
		}
		return filepath.Clean(cwd)
	}
	return "."
}

func resolveServeDefaultDir(defaultDir string) (string, error) {
	defaultDir = strings.TrimSpace(defaultDir)
	if defaultDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get current directory for default default-dir: %w", err)
		}
		defaultDir = filepath.Join(cwd, "config")
	}
	absPath, err := filepath.Abs(defaultDir)
	if err != nil {
		return "", fmt.Errorf("cannot resolve default-dir: %w", err)
	}
	return absPath, nil
}

func ensureServeDefaultAgentScaffold(agentDir, defaultDir string) error {
	entries, err := os.ReadDir(agentDir)
	if err != nil {
		return fmt.Errorf("read agent-dir: %w", err)
	}
	if len(entries) != 0 {
		return nil
	}
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
	defAgentDir := filepath.Join(agentDir, "DEF_AGENT")
	if err := copyDir(resolvedDefaultDir, defAgentDir); err != nil {
		_ = os.RemoveAll(defAgentDir)
		return fmt.Errorf("failed to copy default-dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(defAgentDir, "skills"), 0o755); err != nil {
		return fmt.Errorf("create default agent scaffold: %w", err)
	}
	return nil
}

func readConfigVersionFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var raw struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return ""
	}
	return strings.TrimSpace(raw.Version)
}

func syncDefaultAgentVersion(agentDir, defaultDir string) error {
	resolvedDefaultDir, err := resolveServeDefaultDir(defaultDir)
	if err != nil {
		return err
	}
	version := readConfigVersionFile(filepath.Join(resolvedDefaultDir, "config.json"))
	if version == "" {
		return nil
	}
	defAgentDir := filepath.Join(strings.TrimSpace(agentDir), "DEF_AGENT")
	info, err := os.Stat(defAgentDir)
	if err != nil || !info.IsDir() {
		return nil
	}
	configPath := filepath.Join(defAgentDir, "config.json")
	cfg := map[string]interface{}{}
	data, err := os.ReadFile(configPath)
	if err == nil {
		if len(bytes.TrimSpace(data)) != 0 {
			if err := json.Unmarshal(data, &cfg); err != nil {
				return fmt.Errorf("parse default agent config: %w", err)
			}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read default agent config: %w", err)
	}
	if current, _ := cfg["version"].(string); strings.TrimSpace(current) == version {
		return nil
	}
	cfg["version"] = version
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal default agent config: %w", err)
	}
	if err := os.WriteFile(configPath, append(out, '\n'), 0o644); err != nil {
		return fmt.Errorf("write default agent config: %w", err)
	}
	return nil
}

func newServeFlagSet(name string, opts *serveOptions) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.IntVar(&opts.port, "port", proxyServicePort, "proxy listen port")
	fs.StringVar(&opts.host, "host", defaultUpstreamHost, "upstream server URL")
	fs.StringVar(&opts.agentDir, "agent-dir", "", "agent directory (required)")
	fs.StringVar(&opts.defaultDir, "default-dir", "", "default agent template directory (default: ./config under startup dir)")
	fs.StringVar(&opts.device, "device", "", "device ID (auto-generated if empty)")
	fs.IntVar(&opts.agentCacheMs, "agent-cache", 10000, "agent metadata cache TTL in ms")
	fs.IntVar(&opts.connectCacheMs, "connect-cache", 10000, "connect metadata cache TTL in ms")
	fs.StringVar(&opts.siteDir, "site", "", "static site directory (default: ./site)")
	fs.IntVar(&opts.connectTimeoutMs, "connect_timeout", 15000, "upstream connect timeout in ms")
	fs.IntVar(&opts.knowledgeUpdateIntervalMs, "knowledge_update_interval", 7200000, "knowledge lastUpdate refresh interval in ms")
	fs.IntVar(&opts.knowledgeUpdateLockMs, "knowledge_update_lock", 1800000, "knowledge refresh request lock window in ms")
	fs.StringVar(&opts.installApp, "install_app", "", "extra install_app entries, comma separated")
	fs.StringVar(&opts.reply, "reply", "", "reply text sent once per plugin when scheduled details are created")
	fs.IntVar(&opts.pluginExecTimeoutMs, "plugin_exec_timeout", defaultPluginExecTimeoutMs, "plugin exec timeout in ms")
	return fs
}

func runServe(opts serveOptions) {
	opts.reply = resolveServeReply(opts.reply)
	if err := validateServeOptions(opts); err != nil {
		log.Fatal(err)
	}
	if strings.TrimSpace(opts.agentDir) == "" {
		fmt.Fprintln(os.Stderr, "error: --agent-dir is required")
		os.Exit(1)
	}
	startupDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("cannot get cwd: %v", err)
	}
	startupDir, err = filepath.Abs(startupDir)
	if err != nil {
		log.Fatalf("cannot resolve startup dir: %v", err)
	}
	defaultDir, err := resolveServeDefaultDir(opts.defaultDir)
	if err != nil {
		log.Fatalf("cannot resolve default-dir: %v", err)
	}
	opts.defaultDir = defaultDir
	if err := ensureServeDefaultAgentScaffold(opts.agentDir, opts.defaultDir); err != nil {
		log.Fatalf("cannot prepare agent-dir: %v", err)
	}
	if err := syncDefaultAgentVersion(opts.agentDir, opts.defaultDir); err != nil {
		log.Printf("sync default agent version skipped: %v", err)
	}

	writeProxyStartupConfig(map[string]interface{}{
		"app":                       firstNonEmpty(resolveExecutablePath(os.Args[0]), os.Args[0]),
		"app-dir":                   startupDir,
		"port":                      opts.port,
		"host":                      opts.host,
		"agent-dir":                 opts.agentDir,
		"default-dir":               opts.defaultDir,
		"device":                    opts.device,
		"agent-cache":               opts.agentCacheMs,
		"connect-cache":             opts.connectCacheMs,
		"site":                      opts.siteDir,
		"connect_timeout":           opts.connectTimeoutMs,
		"knowledge_update_interval": opts.knowledgeUpdateIntervalMs,
		"knowledge_update_lock":     opts.knowledgeUpdateLockMs,
		"install_app":               opts.installApp,
		"reply":                     opts.reply,
		"plugin_exec_timeout":       opts.pluginExecTimeoutMs,
	})

	site := strings.TrimSpace(opts.siteDir)
	if site == "" {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("cannot get cwd: %v", err)
		}
		site = filepath.Join(cwd, "site")
	} else {
		var err error
		site, err = filepath.Abs(site)
		if err != nil {
			log.Fatalf("cannot resolve site path: %v", err)
		}
	}

	connectTimeout := time.Duration(opts.connectTimeoutMs) * time.Millisecond
	client := sharedutil.NewProxyClient(connectTimeout)
	connectSvc, err := connectsvc.NewService(connectsvc.Options{
		DBPath:   "data",
		AgentDir: opts.agentDir,
		CacheTTL: time.Duration(opts.connectCacheMs) * time.Millisecond,
	})
	if err != nil {
		log.Fatalf("cannot init connect service: %v", err)
	}
	defer connectSvc.Close()

	proxy := &ProxyServer{
		Host:                    opts.host,
		AgentDir:                opts.agentDir,
		DefaultDir:              opts.defaultDir,
		DeviceID:                opts.device,
		CacheTTL:                time.Duration(opts.agentCacheMs) * time.Millisecond,
		SiteDir:                 site,
		ConnectTimeout:          connectTimeout,
		KnowledgeUpdateInterval: time.Duration(opts.knowledgeUpdateIntervalMs) * time.Millisecond,
		KnowledgeUpdateLock:     time.Duration(opts.knowledgeUpdateLockMs) * time.Millisecond,
		InstallApps:             parseInstallApps(opts.installApp),
		Client:                  client,
		ConnectCacheTTL:         time.Duration(opts.connectCacheMs) * time.Millisecond,
		AutoReply:               strings.TrimSpace(opts.reply),
		ConnectService:          connectSvc,
		PluginExecTimeout:       time.Duration(opts.pluginExecTimeoutMs) * time.Millisecond,
	}
	startCronExecutor(proxy)
	startConnectPendingRequestSync(proxy)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/connect/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": 0})
	})
	connectHandler := connectsvc.NewDynamicHTTPHandler(func(_ *http.Request) (*connectsvc.Service, func(), error) {
		return proxy.borrowConnectService()
	})
	mux.Handle("/api/connect/meta", connectHandler)
	mux.Handle("/api/connect/request", connectHandler)
	mux.Handle("/api/connect/response", connectHandler)
	mux.HandleFunc("/v1/chat/completions", proxy.HandleChatCompletions)
	mux.HandleFunc("/install_app", proxy.HandleInstallApp)
	mux.HandleFunc("/api/agentId", proxy.HandleAgentIDs)
	mux.HandleFunc("/api/swarm_agent", proxy.HandleSwarmAgents)
	mux.HandleFunc("/api/deviceId", proxy.HandleDeviceID)
	mux.HandleFunc("/api/plugins/meta", proxy.HandlePluginsMeta)
	mux.HandleFunc("/api/plugins/status", proxy.HandlePluginsStatus)
	mux.HandleFunc("/api/plugins/config", proxy.HandlePluginsConfig)
	mux.HandleFunc("/api/plugins/start", proxy.HandlePluginsStart)
	mux.HandleFunc("/api/plugins/stop", proxy.HandlePluginsStop)
	mux.HandleFunc("/api/plugins/exec", proxy.HandlePluginsExec)
	mux.HandleFunc("/api/plugins/log", proxy.HandlePluginsLog)
	mux.HandleFunc("/api/folder", proxy.HandleFolder)
	mux.HandleFunc("/api/skills", proxy.HandleSkills)
	mux.HandleFunc("/skills_warning", proxy.HandleSkillsWarning)
	mux.HandleFunc("/api/files", proxy.HandleFiles)
	mux.HandleFunc("/api/data", proxy.HandleData)
	mux.HandleFunc("/api/workspace", proxy.HandleWorkspace)
	mux.HandleFunc("/api/url_preview_probe", proxy.HandleURLPreviewProbe)
	mux.HandleFunc("/file/lastUpdate", proxy.HandleFileLastUpdate)
	mux.HandleFunc("/api/edit", proxy.HandleEdit)
	mux.HandleFunc("/api/del", proxy.HandleDel)
	mux.HandleFunc("/api/raw", proxy.HandleRaw)
	mux.HandleFunc("/api/heartbeat", proxy.HandleHeartbeat)
	mux.HandleFunc("/api/agent/init", proxy.HandleAgentInit)
	mux.HandleFunc("/api/agent/delete", proxy.HandleAgentDelete)
	mux.HandleFunc("/api/agent/create", proxy.HandleAgentCreate)
	mux.HandleFunc("/api/upload", proxy.HandleUpload)
	mux.HandleFunc("/api/config", proxy.HandleSwarm)
	mux.HandleFunc("/api/token", proxy.HandleToken)
	mux.HandleFunc("/api/consume", proxy.HandleConsume)
	mux.HandleFunc("/api/sandbox", proxy.HandleSandbox)
	mux.HandleFunc("/api/sandbox=off", proxy.HandleSandbox)
	mux.HandleFunc("/api/sandbox=filepick", proxy.HandleSandbox)
	mux.HandleFunc("/api/sandbox=net", proxy.HandleSandbox)
	mux.HandleFunc("/api/sandbox=filepick_net", proxy.HandleSandbox)
	mux.HandleFunc("/api/sandbox_status", proxy.HandleSandboxStatus)
	mux.HandleFunc("/api/cron/create", proxy.HandleCron)
	mux.HandleFunc("/api/cron/detail/metadata", proxy.HandleCronMetadata)
	mux.HandleFunc("/api/cron/delete", proxy.HandleCronDelete)
	mux.HandleFunc("/api/cron/detail/delete", proxy.HandleCronDetailDelete)
	mux.HandleFunc("/api/cron/detail/list", proxy.HandleCronDetailList)
	mux.HandleFunc("/api/cron/detail/status", proxy.HandleCronDetailStatus)
	mux.HandleFunc("/api/cancel", proxy.HandleCancel)
	mux.HandleFunc("/api/restore", proxy.HandleRestore)
	mux.HandleFunc("/api/chat_session_log", proxy.HandleChatSessionLog)
	mux.HandleFunc("/log_skill", proxy.HandleLogSkill)
	mux.HandleFunc("/log_skill_status", proxy.HandleLogSkillStatus)
	mux.HandleFunc("/api/download", proxy.HandleDownload)
	mux.HandleFunc("/api/cmd", proxy.HandleCmd)
	mux.HandleFunc("/api/kill", proxy.HandleKill)
	mux.HandleFunc("/knowledge", proxy.HandleKnowledge)
	mux.HandleFunc("/knowledge/", proxy.HandleKnowledge)
	mux.HandleFunc("/knowledge_lastUpdate", proxy.HandleKnowledgeLastUpdate)
	mux.HandleFunc("/knowledge_path", proxy.HandleKnowledgePath)

	if err := server.Register(mux, site); err != nil {
		log.Printf("static: %v (skipping)", err)
	}

	addr := fmt.Sprintf(":%d", opts.port)
	log.Printf("proxy started on %s → %s", addr, opts.host)
	log.Printf("site: %s", proxyBrowserURL(opts.port))
	log.Printf("agent-dir: %s, device: %s", opts.agentDir, opts.device)
	go func() {
		time.Sleep(200 * time.Millisecond)
		browserURL := proxyBrowserURL(opts.port)
		if err := sharedutil.OpenBrowserMaximized(browserURL); err != nil {
			log.Printf("open browser failed: %v", err)
			return
		}
		log.Printf("browser opened maximized: %s", browserURL)
	}()

	if err := http.ListenAndServe(addr, sharedutil.WithLocalCORS(mux)); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func parseBoolValue(flagName, value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "1", "true", "yes", "y", "on":
		return true, nil
	case "0", "false", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid %s value: %s", flagName, value)
	}
}

func parseArgs(args []string, cycle *int, rawTime, agentID, chatID, model, content *string, thinking, routerDisable *bool, cronExpr, agentDir *string) error {
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
			parsed, err := parseBoolValue("thinking", value)
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
			parsed, err := parseBoolValue("router_disable", value)
			if err != nil {
				return err
			}
			*routerDisable = parsed
		}
	}
	return nil
}

func resolveAgentDir(agentDir string) (string, error) {
	agentDir = strings.TrimSpace(agentDir)
	if agentDir == "" {
		agentDir = configuredProxyStringValue("agent-dir")
	}
	if agentDir == "" {
		agentDir = strings.TrimSpace(os.Getenv("AGENT_DIR"))
	}
	if agentDir == "" {
		if info, err := os.Stat("agents"); err == nil && info.IsDir() {
			agentDir = "agents"
		}
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

func resolveProxyDeviceID(deviceID string) string {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID != "" {
		return deviceID
	}
	return configuredProxyStringValue("device")
}

func validateAgentExists(agentDir, agentID string) error {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return fmt.Errorf("agent is required")
	}
	cleanDir := filepath.Clean(agentDir)
	if filepath.Base(cleanDir) == agentID {
		if info, err := os.Stat(cleanDir); err == nil && info.IsDir() {
			return nil
		}
	}
	info, err := os.Stat(filepath.Join(agentDir, agentID))
	if err != nil || !info.IsDir() {
		return fmt.Errorf("agent not found: %s", agentID)
	}
	return nil
}

func validateProxyAgentExists(agentDir, deviceID, agentID string) error {
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
	metadata, err := getAgentOutput(agentDir, deviceID, 10*time.Second)
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

func validateCreateArgs(cycle int, rawTime, agentID, model, content string) error {
	if cycle < 0 || cycle > 5 {
		return fmt.Errorf("cycle must be one of 0,1,2,3,4,5")
	}
	if strings.TrimSpace(rawTime) == "" {
		return fmt.Errorf("time is required, format: yyyy-MM-dd hh:mm")
	}
	if _, err := time.ParseInLocation("2006-01-02 15:04", rawTime, time.Local); err != nil {
		return fmt.Errorf("invalid time format: %w", err)
	}
	if strings.TrimSpace(agentID) == "" {
		return fmt.Errorf("agent is required")
	}
	if strings.TrimSpace(model) == "" {
		return fmt.Errorf("model is required")
	}
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content is required")
	}
	return nil
}

func validateCreateCronArgs(cronExpr, agentID, model, content string) error {
	if strings.TrimSpace(cronExpr) == "" {
		return fmt.Errorf("cron is required")
	}
	if strings.TrimSpace(agentID) == "" {
		return fmt.Errorf("agent is required")
	}
	if strings.TrimSpace(model) == "" {
		return fmt.Errorf("model is required")
	}
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content is required")
	}
	return nil
}

func printCronFindMetaHelp() {
	fmt.Println("Usage:")
	fmt.Println("  proxy cron find-meta [options]")
	fmt.Println("  proxy find-meta [options]")
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
	fmt.Println("  proxy cron find-meta --content '每15分钟检查一次上游接口健康' --model OpenAI --chatId chat-001")
	fmt.Println("  proxy find-meta --agent A --cycle 4 --from '2026-05-03 00:00' --to '2026-05-04 00:00'")
}

func printCronFindDetailHelp() {
	fmt.Println("Usage:")
	fmt.Println("  proxy cron find-detail [options]")
	fmt.Println("  proxy find-detail [options]")
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
	fmt.Println("  未指定时间条件时，默认只查询当前时间之后的任务明细。")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  proxy cron find-detail --metaId cron_1 --content '每15分钟检查一次上游接口健康' --model OpenAI --cycle 4 --chatId chat-001")
	fmt.Println("  proxy find-detail --agent A --date 2026-05-03")
}

func printCronDeleteMetaHelp() {
	fmt.Println("Usage:")
	fmt.Println("  proxy cron delete-meta [options]")
	fmt.Println("  proxy delete-meta [options]")
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
	fmt.Println("  proxy cron delete-meta --id meta_1")
	fmt.Println("  proxy delete-meta --agent A --cycle 4 --date 2026-05-03")
}

func printCronDeleteDetailHelp() {
	fmt.Println("Usage:")
	fmt.Println("  proxy cron delete-detail [options]")
	fmt.Println("  proxy delete-detail [options]")
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
	fmt.Println("  proxy cron delete-detail --metaId meta_1")
	fmt.Println("  proxy cron delete-detail --detailId detail_1")
}

func runProxyFindMeta(args []string) {
	if hasHelpFlag(args) {
		printCronFindMetaHelp()
		return
	}
	opts, err := parseCronQueryArgs(args)
	if err != nil {
		log.Fatal(err)
	}
	filter, err := buildCronMetaFilter(opts)
	if err != nil {
		log.Fatal(err)
	}
	db, err := getDataDB()
	if err != nil {
		log.Fatal(err)
	}
	items, err := queryCronMetas(db, filter)
	if err != nil {
		log.Fatal(err)
	}
	out, _ := json.MarshalIndent(map[string]interface{}{"status": 0, "data": items}, "", "  ")
	fmt.Println(string(out))
}

func runProxyFindDetail(args []string) {
	if hasHelpFlag(args) {
		printCronFindDetailHelp()
		return
	}
	opts, err := parseCronQueryArgs(args)
	if err != nil {
		log.Fatal(err)
	}
	filter, err := buildCronDetailFilter(opts)
	if err != nil {
		log.Fatal(err)
	}
	db, err := getDataDB()
	if err != nil {
		log.Fatal(err)
	}
	items, err := queryCronDetails(db, filter)
	if err != nil {
		log.Fatal(err)
	}
	out, _ := json.MarshalIndent(map[string]interface{}{"status": 0, "data": items}, "", "  ")
	fmt.Println(string(out))
}

func runProxyDeleteMeta(args []string) {
	if hasHelpFlag(args) {
		printCronDeleteMetaHelp()
		return
	}
	opts, err := parseCronQueryArgs(args)
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
	db, err := getDataDB()
	if err != nil {
		log.Fatal(err)
	}
	affected, ids, err := deleteCronMetas(db, filter, opts.ID)
	if err != nil {
		log.Fatal(err)
	}
	out, _ := json.MarshalIndent(map[string]interface{}{"status": 0, "affected": affected, "ids": ids}, "", "  ")
	fmt.Println(string(out))
}

func runProxyDeleteDetail(args []string) {
	if hasHelpFlag(args) {
		printCronDeleteDetailHelp()
		return
	}
	opts, err := parseCronQueryArgs(args)
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
	db, err := getDataDB()
	if err != nil {
		log.Fatal(err)
	}
	affected, ids, err := deleteCronDetails(db, filter, opts.DetailID, opts.MetaID)
	if err != nil {
		log.Fatal(err)
	}
	out, _ := json.MarshalIndent(map[string]interface{}{"status": 0, "affected": affected, "ids": ids}, "", "  ")
	fmt.Println(string(out))
}

func runProxyCronCLI(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		fmt.Println("Usage:")
		fmt.Println("  proxy cron create [options]")
		fmt.Println("  proxy cron create-cron [options]")
		fmt.Println("  proxy cron find-meta [options]")
		fmt.Println("  proxy cron find-detail [options]")
		fmt.Println("  proxy cron delete-meta [options]")
		fmt.Println("  proxy cron delete-detail [options]")
		fmt.Println("")
		fmt.Println("Subcommands:")
		fmt.Println("  create       Create task metadata from cycle + first start time")
		fmt.Println("  create-cron  Create task metadata from explicit cron expression")
		fmt.Println("  find-meta    Query task metadata")
		fmt.Println("  find-detail  Query task details")
		fmt.Println("  delete-meta  Delete task metadata and its details")
		fmt.Println("  delete-detail Delete task details")
		return
	}

	switch args[0] {
	case "create", "submit":
		cycle, rawTime, agentID, chatID, model, content, agentDir := 0, "", "", "", "", "", ""
		thinking, routerDisable := false, true
		cronExpr := ""
		if err := parseArgs(args[1:], &cycle, &rawTime, &agentID, &chatID, &model, &content, &thinking, &routerDisable, &cronExpr, &agentDir); err != nil {
			log.Fatal(err)
		}
		if err := validateCreateArgs(cycle, rawTime, agentID, model, content); err != nil {
			log.Fatal(err)
		}
		deviceID := resolveProxyDeviceID("")
		resolvedAgentDir, err := resolveAgentDir(agentDir)
		if err != nil {
			log.Fatal(err)
		}
		if err := validateProxyAgentExists(resolvedAgentDir, deviceID, agentID); err != nil {
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
		cronExpr, agentID, chatID, model, content, agentDir := "", "", "", "", "", ""
		thinking, routerDisable := false, true
		if err := parseArgs(args[1:], nil, nil, &agentID, &chatID, &model, &content, &thinking, &routerDisable, &cronExpr, &agentDir); err != nil {
			log.Fatal(err)
		}
		if err := validateCreateCronArgs(cronExpr, agentID, model, content); err != nil {
			log.Fatal(err)
		}
		deviceID := resolveProxyDeviceID("")
		resolvedAgentDir, err := resolveAgentDir(agentDir)
		if err != nil {
			log.Fatal(err)
		}
		if err := validateProxyAgentExists(resolvedAgentDir, deviceID, agentID); err != nil {
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
		runProxyFindMeta(args[1:])
	case "find-detail":
		runProxyFindDetail(args[1:])
	case "delete-meta":
		runProxyDeleteMeta(args[1:])
	case "delete-detail":
		runProxyDeleteDetail(args[1:])
	default:
		log.Fatalf("unknown cron command: %s", args[0])
	}
}

func runProxySkillsWarningCLI(args []string) {
	fs := flag.NewFlagSet("skills-warning", flag.ExitOnError)
	refresh := fs.Bool("refresh", false, "scan and sync warnings before reading")
	root := fs.String("root", "", "skills root directory for refresh")
	agentDir := fs.String("agent-dir", resolveDefaultAgentDirArg(), "agent directory")
	_ = fs.Parse(args)

	proxy := &ProxyServer{
		WarningScanRoot: strings.TrimSpace(*root),
		AgentDir:        strings.TrimSpace(*agentDir),
	}
	if *refresh {
		if err := syncSkillsWarnings(proxy); err != nil {
			log.Fatal(err)
		}
	}

	dbPath, err := getDataDBPath()
	if err != nil {
		log.Fatal(err)
	}
	store, err := skillscore.OpenWarningStore(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	out, err := store.ListJSON()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}

func runProxyLogSkillCLI(args []string) {
	fs := flag.NewFlagSet("log-skill", flag.ExitOnError)
	agentID := fs.String("agentId", "", "agent id")
	agent := fs.String("agent", "", "agent id")
	chatID := fs.String("chatId", "", "chat id")
	chat := fs.String("chat", "", "chat id")
	round := fs.Int("round", 0, "recent round count; omit to use pure time range")
	start := fs.String("start", "", "log start time")
	closeTime := fs.String("close", "", "log close time")
	agentDir := fs.String("agent-dir", "", "agent directory")
	device := fs.String("device", "", "device id")
	cacheMs := fs.Int("agent-cache", 10000, "agent metadata cache ttl in ms")
	_ = fs.Parse(args)

	targetAgentID := strings.TrimSpace(*agentID)
	if targetAgentID == "" {
		targetAgentID = strings.TrimSpace(*agent)
	}
	targetChatID := strings.TrimSpace(*chatID)
	if targetChatID == "" {
		targetChatID = strings.TrimSpace(*chat)
	}

	resolvedAgentDir, err := resolveAgentDir(strings.TrimSpace(*agentDir))
	if err != nil {
		log.Fatal(err)
	}
	proxy := &ProxyServer{
		AgentDir: resolvedAgentDir,
		DeviceID: resolveProxyDeviceID(strings.TrimSpace(*device)),
		CacheTTL: time.Duration(*cacheMs) * time.Millisecond,
	}
	path, err := exportRoundLog(proxy, targetAgentID, targetChatID, roundLogOptions{
		Round: *round,
		Start: *start,
		Close: *closeTime,
	})
	if err != nil {
		log.Fatal(err)
	}
	sizeK, err := fileSizeK(path)
	if err != nil {
		log.Fatal(err)
	}
	out, _ := json.MarshalIndent(map[string]any{"status": 0, "path": path, "sizeK": sizeK}, "", "  ")
	fmt.Println(string(out))
}

func runProxyFileLastUpdateCLI(args []string) {
	fs := flag.NewFlagSet("file-last-update", flag.ExitOnError)
	filePath := fs.String("file", "", "target file path")
	agentID := fs.String("agentId", "", "agent id for relative path")
	agent := fs.String("agent", "", "agent id for relative path")
	agentDir := fs.String("agent-dir", resolveDefaultAgentDirArg(), "agent directory")
	device := fs.String("device", "", "device id")
	cacheMs := fs.Int("agent-cache", 10000, "agent metadata cache ttl in ms")
	_ = fs.Parse(args)

	targetAgentID := strings.TrimSpace(*agentID)
	if targetAgentID == "" {
		targetAgentID = strings.TrimSpace(*agent)
	}
	resolvedAgentDir, err := resolveAgentDir(strings.TrimSpace(*agentDir))
	if err != nil {
		log.Fatal(err)
	}
	proxy := &ProxyServer{
		AgentDir: resolvedAgentDir,
		DeviceID: resolveProxyDeviceID(strings.TrimSpace(*device)),
		CacheTTL: time.Duration(*cacheMs) * time.Millisecond,
	}
	resolved, err := proxy.resolveFileTarget(targetAgentID, strings.TrimSpace(*filePath))
	if err != nil {
		log.Fatal(err)
	}
	value, err := fileLastUpdateDuration(resolved, time.Now())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(value)
}

func printHelp() {
	fmt.Println("Usage:")
	fmt.Println("  proxy [serve options]")
	fmt.Println("  proxy serve [serve options]")
	fmt.Println("  proxy connect <subcommand> [options]")
	fmt.Println("  proxy plugins config --key KEY --agentId ID --model MODEL [--meta JSON]")
	fmt.Println("  proxy plugins exec --key KEY --command 'SUBCOMMAND [ARGS...]' [plugin options]")
	fmt.Println("  proxy plugins log --key KEY [--last N]")
	fmt.Println("  proxy plugins status --key KEY [--pid-file PATH]")
	fmt.Println("  proxy plugins start --key KEY [plugin options]")
	fmt.Println("  proxy plugins stop --key KEY [plugin options]")
	fmt.Println("  proxy create [options]")
	fmt.Println("  proxy create-cron [options]")
	fmt.Println("  proxy find-meta [options]")
	fmt.Println("  proxy find-detail [options]")
	fmt.Println("  proxy delete-meta [options]")
	fmt.Println("  proxy delete-detail [options]")
	fmt.Println("  proxy log-skill [options]")
	fmt.Println("  proxy sandbox --agentId ID --chatId ID [--sandbox off|filepick|net|filepick_net]")
	fmt.Println("  proxy token [--provider MODEL]")
	fmt.Println("  proxy token get [--n N] [--start 'YYYY-MM-DD HH:MM:SS'] [--close 'YYYY-MM-DD HH:MM:SS'] [--agentId ID]")
	fmt.Println("  proxy token --agentId ID --model MODEL --function NAME --total N [--thinking N --input N --cache N --timestamp MS]")
	fmt.Println("  proxy file-last-update [options]")
	fmt.Println("  proxy skills-warning [--refresh] [--root DIR]")
	fmt.Println("  proxy cron <subcommand> [options]")
	fmt.Println("  proxy help")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  serve         Start HTTP proxy service")
	fmt.Println("  connect       Run embedded connect subcommands inside proxy")
	fmt.Println("  plugins       Plugin helper commands")
	fmt.Println("  create        Create task metadata from cycle + first start time")
	fmt.Println("  create-cron   Create task metadata from explicit cron expression")
	fmt.Println("  find-meta     Query task metadata")
	fmt.Println("  find-detail   Query task details")
	fmt.Println("  delete-meta   Delete task metadata and its details")
	fmt.Println("  delete-detail Delete task details")
	fmt.Println("  log-skill     Export recent chat and cli logs to agent tmp with round/start/close filters")
	fmt.Println("  sandbox       Read or update sandbox mode for one AgentId + ChatId")
	fmt.Println("  token         Print saved model tokens, query token consume details, or record token consume details")
	fmt.Println("  file-last-update Print milliseconds since the target file was last updated")
	fmt.Println("  skills-warning Read current SKILL parse warnings from shared sqlite")
	fmt.Println("  cron          Grouped cron subcommands: create / create-cron / find-meta / find-detail / delete-meta / delete-detail")
	fmt.Println("  help          Show this help")
	fmt.Println("")
	fmt.Println("Serve options:")
	fmt.Println("  --agent-dir=DIR       Agent 根目录，启动 HTTP 服务时必填")
	fmt.Println("  --default-dir=DIR     新建 Agent 默认模板目录，默认当前启动目录下的 ./config")
	fmt.Println("  --port=INT            监听端口，固定 8080")
	fmt.Println("  --host=URL            上游服务地址，默认 " + defaultUpstreamHost)
	fmt.Println("  --device=ID           设备 ID，可选")
	fmt.Println("  --agent-cache=MS      Agent 元数据缓存时长（毫秒）")
	fmt.Println("  --connect-cache=MS    Connect 元数据缓存时长（毫秒）")
	fmt.Println("  --site=DIR            静态站点目录，默认当前目录下的 ./site")
	fmt.Println("  --connect_timeout=MS  上游连接超时（毫秒）")
	fmt.Println("  --knowledge_update_interval=MS  knowledge.lastUpdate 透传阈值，默认 7200000")
	fmt.Println("  --knowledge_update_lock=MS      knowledge 更新申请锁窗口，默认 1800000")
	fmt.Println("  --install_app=CSV     额外 install_app 条目，逗号分隔，接口返回会自动去重合并")
	fmt.Println("  --reply=TEXT          三方插件开始执行时的推送文案")
	fmt.Println("")
	fmt.Println("Plugin log options:")
	fmt.Println("  proxy plugins log --key=KEY [--last=10]")
	fmt.Println("  --key=VALUE                  插件运行时主键")
	fmt.Println("  --last=INT                   启动时先补发最后 N 行，默认 10")
	fmt.Println("")
	fmt.Println("Plugin start/stop options:")
	fmt.Println("  proxy plugins start --key=KEY [plugin options]")
	fmt.Println("  proxy plugins stop --key=KEY [plugin options]")
	fmt.Println("  --key=VALUE                  插件运行时主键")
	fmt.Println("  其余参数会原样透传给插件，例如 --connect-bin ./connect")
	fmt.Println("")
	fmt.Println("Plugin exec options:")
	fmt.Println("  proxy plugins exec --key=KEY --command='SUBCOMMAND [ARGS...]' [plugin options]")
	fmt.Println("  --key=VALUE                  插件运行时主键")
	fmt.Println("  --command=TEXT               插件命令，支持多级子命令，例如 'instance init'")
	fmt.Println("  其余参数会转成 --flag value 形式透传给插件；未传 --connect-bin 时会自动补齐当前 proxy")
	fmt.Println("")
	fmt.Println("Plugin config options:")
	fmt.Println("  proxy plugins config --key=KEY --agentId=ID --model=MODEL [--meta='{}']")
	fmt.Println("  --key=VALUE                  插件运行时主键，例如 feishu")
	fmt.Println("  --meta=JSON                  配置表单 JSON 字符串，默认 {}")
	fmt.Println("  --stream[=true|false]        是否支持流式回复，默认 false")
	fmt.Println("  --agent=ID / --agentId=ID    绑定 AgentId")
	fmt.Println("  --chat=ID / --chatId=ID      会话 ID")
	fmt.Println("  --model=NAME                 已注册模型名")
	fmt.Println("  --thinking[=true|false]      是否深度思考，默认 false")
	fmt.Println("  --router_disable[=true|false] 是否关闭 router，默认 true")
	fmt.Println("")
	fmt.Println("Create options:")
	fmt.Println("  --cycle=INT                0=仅一次, 1=工作日, 2=自然日, 3=每小时, 4=每15分钟, 5=每30分钟")
	fmt.Println("  --time='YYYY-MM-DD HH:MM' / --rawTime 'YYYY-MM-DD HH:MM'  首次开始时间，create 必填")
	fmt.Println("  --agent=ID / --agentId ID  绑定的 AgentId，必填")
	fmt.Println("  --agent-dir=DIR            Agent 根目录；也可用 AGENT_DIR 或当前目录 ./agents")
	fmt.Println("  --chat=ID / --chatId ID    会话 ID（CHAT_ID），可选")
	fmt.Println("  --model=NAME / --model NAME  模型名称，必填")
	fmt.Println("  --thinking[=true|false]    是否深度思考；也支持 --thinking true")
	fmt.Println("  --router_disable[=true|false] 是否关闭 router，默认 true")
	fmt.Println("  --content='TEXT'           任务内容，必填")
	fmt.Println("")
	fmt.Println("Create-cron options:")
	fmt.Println("  --cron='EXPR' / --cron EXPR  Cron 表达式，必填")
	fmt.Println("  --agent=ID / --agentId ID   绑定的 AgentId，必填")
	fmt.Println("  --agent-dir=DIR             Agent 根目录；也可用 AGENT_DIR 或当前目录 ./agents")
	fmt.Println("  --chat=ID / --chatId ID     会话 ID（CHAT_ID），可选")
	fmt.Println("  --model=NAME / --model NAME  模型名称，必填")
	fmt.Println("  --thinking[=true|false]     是否深度思考；也支持 --thinking true")
	fmt.Println("  --router_disable[=true|false] 是否关闭 router，默认 true")
	fmt.Println("  --content='TEXT'            任务内容，必填")
	fmt.Println("")
	fmt.Println("Find-meta options:")
	fmt.Println("  --agent=ID / --agentId ID   按 AgentId 精确匹配")
	fmt.Println("  --chat=ID / --chatId ID     按 ChatId 精确匹配")
	fmt.Println("  --model=NAME                按模型精确匹配")
	fmt.Println("  --content='TEXT'            按内容模糊匹配")
	fmt.Println("  --cycle=INT                 按执行周期精确匹配")
	fmt.Println("  --time='YYYY-MM-DD HH:MM'   查询指定开始执行时间点")
	fmt.Println("  --date='YYYY-MM-DD'         查询指定开始执行日期")
	fmt.Println("  --from='YYYY-MM-DD HH:MM'   开始执行时间范围起点")
	fmt.Println("  --to='YYYY-MM-DD HH:MM'     开始执行时间范围终点")
	fmt.Println("")
	fmt.Println("Find-detail options:")
	fmt.Println("  --meta=ID / --metaId ID     按元数据 ID 精确匹配，支持 1 或 cron_1")
	fmt.Println("  --agent=ID / --agentId ID   按 AgentId 精确匹配")
	fmt.Println("  --chat=ID / --chatId ID     按 ChatId 精确匹配")
	fmt.Println("  --model=NAME                按模型精确匹配")
	fmt.Println("  --content='TEXT'            按内容模糊匹配")
	fmt.Println("  --cycle=INT                 按执行周期精确匹配")
	fmt.Println("  --time='YYYY-MM-DD HH:MM'   查询指定执行时间点")
	fmt.Println("  --date='YYYY-MM-DD'         查询指定执行日期")
	fmt.Println("  --from='YYYY-MM-DD HH:MM'   执行时间范围起点")
	fmt.Println("  --to='YYYY-MM-DD HH:MM'     执行时间范围终点")
	fmt.Println("  未指定时间条件时，默认只查询当前时间之后的任务明细")
	fmt.Println("")
	fmt.Println("Delete-meta options:")
	fmt.Println("  --id=ID                     按元数据 ID 删除，支持 1 / cron_1 / meta_1")
	fmt.Println("  其余过滤条件与 find-meta 相同")
	fmt.Println("")
	fmt.Println("Delete-detail options:")
	fmt.Println("  --detailId=ID               按明细 ID 删除，支持 1 / detail_1")
	fmt.Println("  --meta=ID / --metaId ID     删除指定元数据下全部匹配明细")
	fmt.Println("  其余过滤条件与 find-detail 相同；删除时未传时间不会默认限制未来数据")
	fmt.Println("")
	fmt.Println("Log-skill options:")
	fmt.Println("  --agent=ID / --agentId ID   绑定的 AgentId，必填")
	fmt.Println("  --chat=ID / --chatId ID     会话 ID，必填")
	fmt.Println("  --round=N                   最近 N 轮，必填且大于 0")
	fmt.Println("  --agent-dir=DIR             Agent 根目录；也可用 AGENT_DIR 或当前目录 ./agents")
	fmt.Println("  --device=ID                 设备 ID，可选")
	fmt.Println("  --agent-cache=MS            Agent 元数据缓存时长（毫秒）")
	fmt.Println("")
	fmt.Println("Skills-warning options:")
	fmt.Println("  --refresh                   读取前先扫描并同步 SKILL 解析告警")
	fmt.Println("  --root=DIR                  refresh 时扫描的根目录，显式覆盖默认值")
	fmt.Println("  --agent-dir=DIR             Agent 根目录；未传 --root 时默认扫描 DIR/skills")
	fmt.Println("")
	fmt.Println("File-last-update options:")
	fmt.Println("  --file=PATH                 目标文件路径，支持绝对路径和相对路径")
	fmt.Println("  --agent=ID / --agentId ID   相对路径时对应的 AgentId")
	fmt.Println("  --agent-dir=DIR             Agent 根目录；也可用 AGENT_DIR 或当前目录 ./agents")
	fmt.Println("  --device=ID                 设备 ID，可选")
	fmt.Println("  --agent-cache=MS            Agent 元数据缓存时长（毫秒）")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  proxy --agent-dir ./agents --port 8080")
	fmt.Println("  proxy serve --agent-dir ./agents --site ./site")
	fmt.Println("  proxy serve --agent-dir ./agents --knowledge_update_interval 7200000 --knowledge_update_lock 1800000")
	fmt.Println("  proxy serve --agent-dir ./agents --install_app git,node")
	fmt.Println("  proxy connect meta-create --key feishu --meta '{\"appId\":\"cli-app\"}' --callback ignored --agent A --model OpenAI")
	fmt.Println("  proxy connect meta-get --key feishu")
	fmt.Println("  proxy connect meta-list")
	fmt.Println("  proxy plugins config --key feishu --meta '{\"appId\":\"cli-app\"}' --agentId A --model OpenAI --router_disable=true")
	fmt.Println("  proxy plugins exec --key browser --command 'instance init' --agentId A --chatId chat-001")
	fmt.Println("  proxy plugins log --key feishu --last 20")
	fmt.Println("  proxy plugins status --key feishu")
	fmt.Println("  proxy plugins start --key feishu --connect-bin ./connect")
	fmt.Println("  proxy plugins stop --key feishu --pid-file ../plugins/feishu.pid")
	fmt.Println("  proxy create --agent-dir ../agent/test-case --content '每15分钟检查一次上游接口健康' --model OpenAI --thinking true --router_disable=false --rawTime '2026-05-03 10:00' --cycle 4 --chatId chat-001 --agent A")
	fmt.Println("  proxy create --cycle=1 --time='2026-05-03 09:30' --agent=A --chat=chat-001 --model=OpenAI --thinking --content='查看天气'")
	fmt.Println("  proxy create-cron --cron='10 12 * * 1-5' --agent=A --chat=chat-001 --model=OpenAI --router_disable=true --content='查看天气'")
	fmt.Println("  proxy cron find-meta --content '每15分钟检查一次上游接口健康' --model OpenAI --chatId chat-001")
	fmt.Println("  proxy find-detail --metaId cron_1 --cycle 4 --chatId chat-001")
	fmt.Println("  proxy cron delete-meta --id meta_1")
	fmt.Println("  proxy delete-detail --detailId detail_1")
	fmt.Println("  proxy log-skill --agent A --chat chat-001 --round 3")
	fmt.Println("  proxy log-skill --agent A --chat chat-001 --start '2026-05-13 12:00:00' --close '2026-05-13 12:10:00'")
	fmt.Println("  proxy token")
	fmt.Println("  proxy token --provider deepseek")
	fmt.Println("  proxy file-last-update --agent b --file USER.md")
	fmt.Println("  proxy file-last-update --file /abs/path/to/file.md")
	fmt.Println("  proxy skills-warning --agent-dir ./agent --refresh")
	fmt.Println("  proxy skills-warning --refresh --root ./agent/custom-skills")
}

func printProxyConnectHelp() {
	fmt.Println("Usage:")
	fmt.Println("  proxy connect <subcommand> [options]")
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
	fmt.Println("  推荐统一使用 proxy connect <subcommand>，例如 proxy connect meta-get。")
	fmt.Println("  proxy connect list-meta 仍作为兼容入口保留，并映射到新的 proxy connect meta-list。")
	fmt.Println("  --callback 仅作为输入占位，实际保存值固定为 plugins/<plugin-key>。")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  proxy connect meta-create --key feishu --meta '{\"appId\":\"cli-app\"}' --callback ignored --agent A --chatId chat-001 --model OpenAI")
	fmt.Println("  proxy connect meta-update --key feishu --meta '{\"appId\":\"new\"}' --callback ignored --agent A --chatId chat-002 --model OpenAI")
	fmt.Println("  proxy connect meta-get --key feishu")
	fmt.Println("  proxy connect meta-list")
	fmt.Println("  proxy connect add-request --key feishu --externalId ext-1 --content 'HELLO WORLD' --original '{\"text\":\"HELLO WORLD\"}'")
	fmt.Println("  proxy connect response-list --key feishu --request-id 1")
}

var proxyConnectSubcommands = map[string]struct{}{
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

func isProxyConnectSubcommand(command string) bool {
	_, ok := proxyConnectSubcommands[strings.TrimSpace(command)]
	return ok
}

func runProxyPluginCLI(args []string) {
	if len(args) == 0 || hasHelpFlag(args) {
		fmt.Println("Usage:")
		fmt.Println("  proxy plugins config --key KEY --agentId ID --model MODEL [--meta JSON]")
		fmt.Println("  proxy plugins exec --key KEY --command 'SUBCOMMAND [ARGS...]' [plugin options]")
		fmt.Println("  proxy plugins log --key KEY [--last N]")
		fmt.Println("  proxy plugins status --key KEY [--pid-file PATH]")
		fmt.Println("  proxy plugins start --key KEY [plugin options]")
		fmt.Println("  proxy plugins stop --key KEY [plugin options]")
		fmt.Println("")
		fmt.Println("Subcommands:")
		fmt.Println("  config        Create or update plugin config")
		fmt.Println("  exec          Execute one plugin command")
		fmt.Println("  log           Tail plugin log and print SSE events to stdout")
		fmt.Println("  status        Check whether a plugin process is running")
		fmt.Println("  start         Start a plugin executable")
		fmt.Println("  stop          Stop a plugin executable")
		return
	}

	switch args[0] {
	case "config":
		fs := flag.NewFlagSet("plugins config", flag.ExitOnError)
		key := fs.String("key", "", "plugin runtime key")
		meta := fs.String("meta", "{}", "plugin config json string")
		stream := fs.Bool("stream", false, "plugin supports stream")
		agent := fs.String("agent", "", "agent id")
		agentID := fs.String("agentId", "", "agent id")
		chat := fs.String("chat", "", "chat id")
		chatID := fs.String("chatId", "", "chat id")
		model := fs.String("model", "", "model name")
		thinking := fs.Bool("thinking", false, "deep thinking")
		routerDisable := fs.Bool("router_disable", true, "disable router")
		_ = fs.Parse(args[1:])

		targetAgentID := strings.TrimSpace(*agentID)
		if targetAgentID == "" {
			targetAgentID = strings.TrimSpace(*agent)
		}
		targetChatID := strings.TrimSpace(*chatID)
		if targetChatID == "" {
			targetChatID = strings.TrimSpace(*chat)
		}

		item, err := connectsvc.UpsertPluginConfig(map[string]string{
			"agent-dir": resolveDefaultAgentDirArg(),
			"db":        "data",
		}, connectsvc.MetaInput{
			Key:           strings.TrimSpace(*key),
			Meta:          strings.TrimSpace(*meta),
			Stream:        *stream,
			AgentID:       targetAgentID,
			ChatID:        targetChatID,
			Model:         strings.TrimSpace(*model),
			Thinking:      *thinking,
			RouterDisable: *routerDisable,
		})
		if err != nil {
			log.Fatal(err)
		}
		out, _ := json.MarshalIndent(map[string]any{"status": 0, "data": item}, "", "  ")
		fmt.Println(string(out))
	case "start", "stop":
		flags, err := connectsvc.ParseFlags(args[1:])
		if err != nil {
			log.Fatal(err)
		}
		target := connectsvc.FirstValue(flags, "key")
		delete(flags, "key")
		result, err := connectsvc.RunPluginAction(target, args[0], flags)
		if err != nil {
			log.Fatal(err)
		}
		out, _ := json.MarshalIndent(map[string]any{"status": 0, "data": buildPluginActionResponse(result)}, "", "  ")
		fmt.Println(string(out))
	case "exec":
		flags, err := connectsvc.ParseFlags(args[1:])
		if err != nil {
			log.Fatal(err)
		}
		target := connectsvc.FirstValue(flags, "key")
		command := connectsvc.FirstValue(flags, "command")
		delete(flags, "key")
		delete(flags, "command")
		ensurePluginConnectBin(flags)
		result, err := connectsvc.RunPluginExecCommand(target, command, flags, resolveProxyPluginExecTimeout())
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
	case "log":
		fs := flag.NewFlagSet("plugins log", flag.ExitOnError)
		key := fs.String("key", "", "plugin runtime key")
		last := fs.Int("last", pluginlog.DefaultLastLines, "initial number of tail lines")
		_ = fs.Parse(args[1:])

		target := strings.TrimSpace(*key)
		logPath, err := pluginlog.ResolveLogPath(target)
		if err != nil {
			log.Fatal(err)
		}
		err = pluginlog.TailFile(context.Background(), logPath, pluginlog.Options{LastLines: *last}, func(event pluginlog.Event) error {
			_, writeErr := fmt.Printf("event: %s\ndata: %s\n\n", event.Type, event.Data)
			return writeErr
		})
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unknown plugins command: %s", args[0])
	}
}

func runProxyConnectCLI(args []string) {
	if len(args) == 0 || hasHelpFlag(args) {
		printProxyConnectHelp()
		return
	}
	flags, err := connectsvc.ParseFlags(args[1:])
	if err != nil {
		log.Fatal(err)
	}
	if _, ok := flags["db"]; !ok {
		flags["db"] = "data"
	}
	if _, ok := flags["agent-dir"]; !ok {
		flags["agent-dir"] = resolveDefaultAgentDirArg()
	}
	if _, ok := flags["connect-cache"]; !ok {
		if value, ok := configuredProxyIntValue("connect-cache"); ok && value > 0 {
			flags["connect-cache"] = strconv.Itoa(value)
		}
		if _, ok := flags["connect-cache"]; !ok {
			flags["connect-cache"] = "10000"
		}
	}

	command := normalizeProxyConnectCommand(args[0])
	switch command {
	case "start", "serve", "stop":
		rebuildFlags := func(values map[string]string) []string {
			keys := make([]string, 0, len(values))
			for key := range values {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			out := make([]string, 0, len(keys)*2)
			for _, key := range keys {
				out = append(out, "--"+key, values[key])
			}
			return out
		}
		code := connectsvc.RunCLI(append([]string{command}, rebuildFlags(flags)...), os.Stdout, os.Stderr)
		if code != 0 {
			os.Exit(code)
		}
		return
	default:
		cacheMS, err := connectsvc.IntValue(flags, "connect-cache", 10000)
		if err != nil {
			log.Fatal(err)
		}
		svc, err := connectsvc.NewService(connectsvc.Options{
			DBPath:   firstNonEmpty(flags["db"], "data"),
			AgentDir: firstNonEmpty(flags["agent-dir"], resolveDefaultAgentDirArg()),
			CacheTTL: time.Duration(cacheMS) * time.Millisecond,
		})
		if err != nil {
			log.Fatal(err)
		}
		defer svc.Close()
		code := connectsvc.RunCLIWithService(command, flags, svc, os.Stdout, os.Stderr)
		if code != 0 {
			os.Exit(code)
		}
	}
}

func normalizeProxyConnectCommand(command string) string {
	switch strings.TrimSpace(command) {
	case "meta-list", "list-meta":
		// proxy 对外统一使用 meta-list，内部继续复用 connectsvc 的配置视图实现。
		return "list-meta"
	default:
		return strings.TrimSpace(command)
	}
}

func resolveRouterDisableFormValue(form url.Values, fallback bool) bool {
	return connectsvc.ResolveRouterDisableFormValue(form, fallback)
}

func parsePluginConfigRequest(r *http.Request) (connectsvc.MetaInput, error) {
	return connectsvc.ParsePluginConfigRequest(r)
}

func parsePluginActionRequest(r *http.Request) (string, map[string]string, error) {
	return connectsvc.ParsePluginActionRequest(r)
}

func writePluginConfigError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"status":  1,
		"content": err.Error(),
	})
}

func buildPluginActionResponse(result *connectsvc.PluginActionResult) pluginActionResponse {
	return connectsvc.BuildPluginActionResponse(result)
}

func buildPluginStatusResponse(result *connectsvc.PluginStatus) pluginStatusResponse {
	return connectsvc.BuildPluginStatusResponse(result)
}

func resolveDefaultAgentDirArg() string {
	if value := strings.TrimSpace(os.Getenv("AGENT_DIR")); value != "" {
		return value
	}
	if runtime.GOOS == "darwin" {
		if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
			return filepath.Join(home, "Library", "Application Support", "deepright", "agent")
		}
	}
	if info, err := os.Stat("agent"); err == nil && info.IsDir() {
		return "agent"
	}
	if info, err := os.Stat("../agent"); err == nil && info.IsDir() {
		return "../agent"
	}
	return "agent"
}

func main() {
	if len(os.Args) < 2 {
		var opts serveOptions
		fs := newServeFlagSet(os.Args[0], &opts)
		if err := fs.Parse(os.Args[1:]); err != nil {
			os.Exit(2)
		}
		if !flagProvided(fs, "reply") {
			opts.reply = resolveServeReply("")
		}
		runServe(opts)
		return
	}

	switch os.Args[1] {
	case "--help", "-h", "help":
		printHelp()
		return
	case "connect":
		runProxyConnectCLI(os.Args[2:])
		return
	case "plugins":
		runProxyPluginCLI(os.Args[2:])
		return
	case "cron":
		runProxyCronCLI(os.Args[2:])
		return
	case "serve":
		var opts serveOptions
		fs := newServeFlagSet("serve", &opts)
		if err := fs.Parse(os.Args[2:]); err != nil {
			os.Exit(2)
		}
		if !flagProvided(fs, "reply") {
			opts.reply = resolveServeReply("")
		}
		runServe(opts)
		return
	case "create", "submit":
		runProxyCronCLI(append([]string{"create"}, os.Args[2:]...))
		return
	case "create-cron", "submit-cron":
		runProxyCronCLI(append([]string{"create-cron"}, os.Args[2:]...))
		return
	case "find-meta":
		runProxyFindMeta(os.Args[2:])
		return
	case "find-detail":
		runProxyFindDetail(os.Args[2:])
		return
	case "delete-meta":
		runProxyDeleteMeta(os.Args[2:])
		return
	case "delete-detail":
		runProxyDeleteDetail(os.Args[2:])
		return
	case "log-skill":
		runProxyLogSkillCLI(os.Args[2:])
		return
	case "token":
		runProxyTokenCLI(os.Args[2:])
		return
	case "file-last-update":
		runProxyFileLastUpdateCLI(os.Args[2:])
		return
	case "skills-warning":
		runProxySkillsWarningCLI(os.Args[2:])
		return
	}

	if !strings.HasPrefix(os.Args[1], "-") && isProxyConnectSubcommand(os.Args[1]) {
		runProxyConnectCLI(os.Args[1:])
		return
	}

	var opts serveOptions
	fs := newServeFlagSet(os.Args[0], &opts)
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if !flagProvided(fs, "reply") {
		opts.reply = resolveServeReply("")
	}
	runServe(opts)
}
