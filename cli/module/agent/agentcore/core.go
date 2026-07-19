package agentcore

import (
	"connect/sandboxstate"
	"connect/sharedutil"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"knowledge/knowledgecore"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/glebarez/go-sqlite"
	"skill-scanner/skillscore"
)

type Skill = skillscore.Skill

const internalSkillPrefix = "__internal"

func alphabeticalNameLess(left, right string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	leftFolded := strings.ToLower(left)
	rightFolded := strings.ToLower(right)
	if leftFolded == rightFolded {
		return left < right
	}
	return leftFolded < rightFolded
}

func isInternalSkillName(name string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(name)), internalSkillPrefix)
}

// SortSkillNames orders regular skills before internal skills. Each group is
// ordered alphabetically so every API exposing skill names has a stable order.
func SortSkillNames(names []string) {
	sort.SliceStable(names, func(i, j int) bool {
		leftInternal := isInternalSkillName(names[i])
		rightInternal := isInternalSkillName(names[j])
		if leftInternal != rightInternal {
			return !leftInternal
		}
		return alphabeticalNameLess(names[i], names[j])
	})
}

// SortSkills orders skill metadata using the same order as SortSkillNames.
func SortSkills(skills []Skill) {
	sort.SliceStable(skills, func(i, j int) bool {
		leftInternal := isInternalSkillName(skills[i].Name)
		rightInternal := isInternalSkillName(skills[j].Name)
		if leftInternal != rightInternal {
			return !leftInternal
		}
		return alphabeticalNameLess(skills[i].Name, skills[j].Name)
	})
}

func sortAgents(agents []Agent) {
	sort.SliceStable(agents, func(i, j int) bool {
		return alphabeticalNameLess(agents[i].AgentID, agents[j].AgentID)
	})
}

func SkillNames(skills []Skill) []string {
	names := make([]string, 0, len(skills))
	for _, skill := range skills {
		name := strings.TrimSpace(skill.Name)
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	SortSkillNames(names)
	return names
}

type Agent struct {
	Description   string  `json:"description"`
	Provider      string  `json:"provider"`
	RouterDisable bool    `json:"router_disable"`
	Thinking      bool    `json:"thinking"`
	Version       string  `json:"version"`
	Sandbox       string  `json:"sandbox"`
	Workspace     string  `json:"workspace"`
	AgentID       string  `json:"agentId"`
	Soul          string  `json:"soul"`
	User          string  `json:"user"`
	Skills        []Skill `json:"skills"`
}

type Output struct {
	Timezone  string     `json:"timezone"`
	DeviceID  string     `json:"deviceId"`
	Terminal  string     `json:"terminal"`
	Plugins   []string   `json:"plugins,omitempty"`
	Knowledge *Knowledge `json:"knowledge,omitempty"`
	Git       string     `json:"git"`
	Gateway   string     `json:"gateway"`
	Sys       string     `json:"sys"`
	App       string     `json:"app"`
	Agents    []Agent    `json:"agents"`
}

type Knowledge struct {
	LastUpdate int64  `json:"lastUpdate"`
	Path       string `json:"path"`
}

type PluginRuntime struct {
	Key      string `json:"key"`
	Callback string `json:"callback"`
}

func init() {
	sharedutil.ApplySystemPath()
}

var structCache outputCache

type outputCache struct {
	mu      sync.Mutex
	root    string
	appDir  string
	device  string
	output  *Output
	expires time.Time
}

func normalizeAppDir(appDir string) string {
	appDir = strings.TrimSpace(appDir)
	if appDir == "" {
		return ""
	}
	absPath, err := filepath.Abs(appDir)
	if err != nil {
		return ""
	}
	return absPath
}

func inferAppDir(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return ""
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return ""
	}
	return filepath.Dir(absRoot)
}

func resolveAppDir(root, appDir string) string {
	if normalized := normalizeAppDir(appDir); normalized != "" {
		return normalized
	}
	return inferAppDir(root)
}

func lookupKnowledge(appDir string) *Knowledge {
	appDir = normalizeAppDir(appDir)
	if appDir == "" {
		return nil
	}
	knowledgeDir, err := knowledgecore.KnowledgeDir(appDir)
	if err != nil {
		return nil
	}
	info, err := os.Stat(knowledgeDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	state := &Knowledge{
		Path:       knowledgeDir,
		LastUpdate: 0,
	}
	dbPath, err := knowledgecore.DBPath(appDir)
	if err != nil {
		return state
	}
	if _, err := os.Stat(dbPath); err != nil {
		return state
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return state
	}
	defer db.Close()

	var lastUpdate int64
	if err := db.QueryRow(`SELECT last_update FROM knowledge_runtime WHERE id = 1`).Scan(&lastUpdate); err == nil {
		state.LastUpdate = lastUpdate
	}
	return state
}

func readFileContent(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

type agentConfig struct {
	Description   string
	Provider      string
	RouterDisable bool
	Thinking      bool
	Version       string
}

func readAgentConfig(agentDir string) agentConfig {
	cfgData, err := os.ReadFile(filepath.Join(agentDir, "config.json"))
	if err != nil {
		return agentConfig{RouterDisable: true}
	}
	var raw struct {
		Description   string `json:"description"`
		Provider      string `json:"provider"`
		RouterDisable *bool  `json:"router_disable"`
		Swarm         *bool  `json:"swarm"`
		Thinking      *bool  `json:"thinking"`
		Version       string `json:"version"`
	}
	if err := json.Unmarshal(cfgData, &raw); err != nil {
		return agentConfig{RouterDisable: true}
	}
	cfg := agentConfig{
		Description:   raw.Description,
		Provider:      raw.Provider,
		RouterDisable: true,
		Version:       raw.Version,
	}
	if raw.RouterDisable != nil {
		cfg.RouterDisable = *raw.RouterDisable
	} else if raw.Swarm != nil {
		cfg.RouterDisable = !*raw.Swarm
	}
	if raw.Thinking != nil {
		cfg.Thinking = *raw.Thinking
	}
	return cfg
}

func scanAgents(root string) ([]Agent, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve path: %w", err)
	}

	entries, err := os.ReadDir(absRoot)
	if err != nil {
		return nil, fmt.Errorf("cannot read directory: %w", err)
	}

	var agents []Agent
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		agentDir := filepath.Join(absRoot, entry.Name())

		soul := readFileContent(filepath.Join(agentDir, "SOUL.md"))
		user := readFileContent(filepath.Join(agentDir, "USER.md"))
		if user == "" {
			user = readFileContent(filepath.Join(agentDir, "user.md"))
		}

		var skills []Skill
		skillsDir := filepath.Join(agentDir, "skills")
		if info, err := os.Stat(skillsDir); err == nil && info.IsDir() {
			skills, err = skillscore.ScanAgentSkills(skillsDir)
			if err != nil {
				return nil, fmt.Errorf("scanning skills for agent %s: %w", entry.Name(), err)
			}
		}
		if skills == nil {
			skills = []Skill{}
		}
		SortSkills(skills)

		cfg := readAgentConfig(agentDir)

		agents = append(agents, Agent{
			Description:   cfg.Description,
			Provider:      cfg.Provider,
			RouterDisable: cfg.RouterDisable,
			Thinking:      cfg.Thinking,
			Version:       cfg.Version,
			Sandbox:       "",
			Workspace:     agentDir,
			AgentID:       entry.Name(),
			Soul:          soul,
			User:          user,
			Skills:        skills,
		})
	}
	sortAgents(agents)
	return agents, nil
}

func listAgentDirNames(root string) ([]string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve path: %w", err)
	}

	entries, err := os.ReadDir(absRoot)
	if err != nil {
		return nil, fmt.Errorf("cannot read directory: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		names = append(names, entry.Name())
	}
	return names, nil
}

// ListAgentIDs returns the immediate Agent directory names only. It avoids
// loading Agent metadata, skills, knowledge, plugins, or runtime state.
func ListAgentIDs(root string) ([]string, error) {
	names, err := listAgentDirNames(root)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(names, func(i, j int) bool {
		return alphabeticalNameLess(names[i], names[j])
	})
	return names, nil
}

func cachedAgentDirNames(output *Output) []string {
	if output == nil || len(output.Agents) == 0 {
		return []string{}
	}
	names := make([]string, 0, len(output.Agents))
	for _, agent := range output.Agents {
		agentID := strings.TrimSpace(agent.AgentID)
		if agentID == "" {
			continue
		}
		names = append(names, agentID)
	}
	return names
}

func agentDirectoryLayoutChanged(root string, output *Output) (bool, error) {
	currentNames, err := listAgentDirNames(root)
	if err != nil {
		return false, err
	}
	cachedNames := cachedAgentDirNames(output)
	if len(currentNames) != len(cachedNames) {
		return true, nil
	}
	for i := range currentNames {
		if currentNames[i] != cachedNames[i] {
			return true, nil
		}
	}
	return false, nil
}

func detectTerminal() string {
	shell := os.Getenv("SHELL")
	if shell != "" {
		return shell
	}
	return ""
}

func resolveExecutablePath(path string) string {
	path = strings.TrimSpace(strings.Trim(path, `"'`))
	if path == "" {
		return ""
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	return absPath
}

func detectGitUnix() string {
	if path, err := exec.LookPath("git"); err == nil {
		if absPath := resolveExecutablePath(path); absPath != "" {
			return absPath
		}
	}
	out, err := exec.Command("sh", "-lc", "command -v git").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if absPath := resolveExecutablePath(line); absPath != "" {
			return absPath
		}
	}
	return ""
}

func detectGitWindows() string {
	for _, name := range []string{"git.exe", "git.cmd", "git.bat", "git"} {
		if path, err := exec.LookPath(name); err == nil {
			if absPath := resolveExecutablePath(path); absPath != "" {
				return absPath
			}
		}
	}
	out, err := exec.Command("where", "git").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if absPath := resolveExecutablePath(line); absPath != "" {
			return absPath
		}
	}
	return ""
}

func detectGit() string {
	switch runtime.GOOS {
	case "windows":
		return detectGitWindows()
	case "darwin", "linux":
		return detectGitUnix()
	default:
		if path, err := exec.LookPath("git"); err == nil {
			return resolveExecutablePath(path)
		}
		return ""
	}
}

func DetectGit() string {
	return detectGit()
}

func DetectPython3() string {
	switch runtime.GOOS {
	case "windows":
		for _, name := range []string{"python3.exe", "python3.cmd", "python3.bat", "python3"} {
			if path, err := exec.LookPath(name); err == nil {
				if absPath := ResolveExecutablePath(path); absPath != "" {
					return absPath
				}
			}
		}
	default:
		if path, err := exec.LookPath("python3"); err == nil {
			return ResolveExecutablePath(path)
		}
	}
	return ""
}

func ResolveExecutablePath(path string) string {
	return resolveExecutablePath(path)
}

func DetectRequiredInstallApps() []string {
	apps := make([]string, 0, 1)
	if DetectGit() == "" {
		apps = append(apps, "git")
	}
	return apps
}

func ParseInstallApps(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	items := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		key := strings.ToLower(item)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, item)
	}
	return items
}

func MergeInstallApps(base []string, extras ...string) []string {
	merged := make([]string, 0, len(base)+len(extras))
	seen := make(map[string]struct{}, len(base)+len(extras))
	for _, item := range base {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, trimmed)
	}
	for _, extra := range extras {
		for _, item := range ParseInstallApps(extra) {
			key := strings.ToLower(item)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			merged = append(merged, item)
		}
	}
	return merged
}

func detectGateway() string {
	var cmdStr string
	switch runtime.GOOS {
	case "darwin":
		cmdStr = "arp -n $(route -n get default | grep gateway | awk '{print $2}') | awk '{print $4}'"
	case "linux":
		cmdStr = "ip route | grep default | awk '{print $3}' | head -1 | xargs -I{} arp -n {} | awk 'NR==2{print $3}'"
	default:
		return ""
	}
	out, err := exec.Command("sh", "-c", cmdStr).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func DetectGateway() string {
	return detectGateway()
}

func detectSys() string {
	out, err := exec.Command("uname", "-sr").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func DetectSys() string {
	return detectSys()
}

func detectHardwareUUID() string {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("sh", "-c",
			"ioreg -rd1 -c IOPlatformExpertDevice | awk '/IOPlatformUUID/{print $3}' | tr -d '\"'").Output()
		if err == nil {
			if id := strings.TrimSpace(string(out)); id != "" {
				return id
			}
		}
	case "linux":
		for _, path := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
			if data, err := os.ReadFile(path); err == nil {
				if id := strings.TrimSpace(string(data)); id != "" {
					return id
				}
			}
		}
	}
	return ""
}

func generateDeviceID() string {
	if id := detectHardwareUUID(); id != "" {
		return id
	}
	hash := sha256.Sum256([]byte(runtime.GOOS + ":" + runtime.GOARCH))
	return fmt.Sprintf("%x", hash[:16])
}

func GenerateDeviceID() string {
	return generateDeviceID()
}

func detectTimezone() string {
	if tz, ok := os.LookupEnv("TZ"); ok && tz != "" {
		return tz
	}
	loc := time.Now().Location().String()
	if loc != "" && loc != "Local" {
		return loc
	}
	if target, err := os.Readlink("/etc/localtime"); err == nil {
		if idx := strings.Index(target, "zoneinfo/"); idx >= 0 {
			return target[idx+9:]
		}
	}
	z, _ := time.Now().Zone()
	return z
}

func DetectTerminal() string {
	return detectTerminal()
}

func detectApp() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	abs, err := filepath.Abs(exe)
	if err != nil {
		return exe
	}
	return abs
}

func detectPlugins(appPath string) []string {
	appPath = strings.TrimSpace(appPath)
	if appPath == "" {
		return nil
	}
	pluginDir := filepath.Join(filepath.Dir(appPath), "plugins")
	info, err := os.Stat(pluginDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	args := pluginRuntimeListCommandArgs(appPath)
	cmd := exec.Command(appPath, args...)
	cmd.Dir = filepath.Dir(appPath)
	out, err := cmd.Output()
	if err == nil {
		var items []PluginRuntime
		if json.Unmarshal(out, &items) == nil {
			return DetectRunningPluginKeys(pluginDir, items)
		}
	}
	return nil
}

func pluginRuntimeListCommandArgs(appPath string) []string {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(appPath)))
	if base == "integration" || base == "proxy" || strings.HasPrefix(base, "integration.") || strings.HasPrefix(base, "proxy.") {
		return []string{"connect", "meta-list"}
	}
	return []string{"list-meta"}
}

func DetectPlugins(appPath string) []string {
	return detectPlugins(appPath)
}

func DetectRunningPluginKeys(defaultPluginDir string, items []PluginRuntime) []string {
	if len(items) == 0 {
		return nil
	}
	plugins := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		if !pluginStarted(defaultPluginDir, key, item.Callback) {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		plugins = append(plugins, key)
	}
	if len(plugins) == 0 {
		return nil
	}
	sort.Strings(plugins)
	return plugins
}

func PluginStarted(defaultPluginDir, key, callback string) bool {
	return pluginStarted(defaultPluginDir, key, callback)
}

func PluginPIDFiles(defaultPluginDir, key, callback string) []string {
	files := pluginPIDFiles(defaultPluginDir, key, callback)
	if len(files) == 0 {
		return nil
	}
	out := make([]string, 0, len(files))
	seen := make(map[string]struct{}, len(files))
	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		if _, ok := seen[file]; ok {
			continue
		}
		seen[file] = struct{}{}
		out = append(out, file)
	}
	return out
}

func pluginStarted(defaultPluginDir, key, callback string) bool {
	for _, pidPath := range PluginPIDFiles(defaultPluginDir, key, callback) {
		pid, err := readPluginPID(pidPath)
		if err != nil {
			continue
		}
		if isProcessRunning(pid) {
			return true
		}
	}
	return false
}

func pluginPIDFiles(defaultPluginDir, key, callback string) []string {
	dir := strings.TrimSpace(defaultPluginDir)
	key = strings.TrimSpace(key)
	if callback = strings.TrimSpace(callback); callback != "" {
		if abs, err := filepath.Abs(callback); err == nil {
			callback = abs
		}
		callbackDir := strings.TrimSpace(filepath.Dir(callback))
		if callbackDir != "" && callbackDir != "." {
			dir = callbackDir
		}
	}
	if dir == "" {
		dir = "."
	}
	return []string{filepath.Join(dir, key+".pid")}
}

func readPluginPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("invalid pid file: %s", path)
	}
	return pid, nil
}

func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	switch runtime.GOOS {
	case "windows":
		out, err := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH").Output()
		if err != nil {
			return false
		}
		line := strings.TrimSpace(string(out))
		if line == "" || strings.HasPrefix(strings.ToLower(line), "info:") {
			return false
		}
		record, err := csv.NewReader(strings.NewReader(line)).Read()
		return err == nil && len(record) > 1 && strings.TrimSpace(record[1]) == strconv.Itoa(pid)
	default:
		out, err := exec.Command("ps", "-o", "pid=", "-p", strconv.Itoa(pid)).Output()
		return err == nil && strings.TrimSpace(string(out)) != ""
	}
}

func buildOutput(root string, appDir string, deviceID string) (*Output, error) {
	agents, err := scanAgents(root)
	if err != nil {
		return nil, err
	}

	if deviceID == "" {
		deviceID = generateDeviceID()
	}
	appPath := detectApp()

	return &Output{
		Timezone:  detectTimezone(),
		DeviceID:  deviceID,
		Terminal:  detectTerminal(),
		Plugins:   detectPlugins(appPath),
		Knowledge: lookupKnowledge(resolveAppDir(root, appDir)),
		Git:       detectGit(),
		Gateway:   detectGateway(),
		Sys:       detectSys(),
		App:       appPath,
		Agents:    agents,
	}, nil
}

func refreshAgentSkills(output *Output) error {
	if output == nil {
		return nil
	}
	for i := range output.Agents {
		skillsDir := filepath.Join(output.Agents[i].Workspace, "skills")
		info, err := os.Stat(skillsDir)
		if err != nil || !info.IsDir() {
			output.Agents[i].Skills = []Skill{}
			continue
		}
		skills, err := skillscore.ScanAgentSkills(skillsDir)
		if err != nil {
			return fmt.Errorf("scanning skills for agent %s: %w", output.Agents[i].AgentID, err)
		}
		if skills == nil {
			skills = []Skill{}
		}
		SortSkills(skills)
		output.Agents[i].Skills = skills
	}
	return nil
}

func refreshAgentConfigFields(output *Output) {
	if output == nil {
		return
	}
	for i := range output.Agents {
		cfg := readAgentConfig(output.Agents[i].Workspace)
		output.Agents[i].Description = cfg.Description
		output.Agents[i].Provider = cfg.Provider
		output.Agents[i].RouterDisable = cfg.RouterDisable
		output.Agents[i].Thinking = cfg.Thinking
	}
}

func refreshAgentSandboxFields(output *Output, appDir string, chatID string) {
	if output == nil {
		return
	}
	for i := range output.Agents {
		output.Agents[i].Sandbox = ""
	}

	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return
	}
	appDir = normalizeAppDir(appDir)
	if appDir == "" {
		return
	}

	dbPath, err := knowledgecore.DBPath(appDir)
	if err != nil {
		return
	}
	if _, err := os.Stat(dbPath); err != nil {
		return
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return
	}
	defer db.Close()

	for i := range output.Agents {
		state, recorded, err := sandboxstate.Get(db, output.Agents[i].AgentID, chatID)
		if err != nil || !recorded {
			continue
		}
		output.Agents[i].Sandbox = sandboxstate.NormalizeMode(state.Sandbox)
	}
}

func refreshDynamicOutputFields(output *Output) {
	if output == nil {
		return
	}
	refreshAgentConfigFields(output)
	output.Git = detectGit()
}

func cloneOutput(src *Output) *Output {
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
		cloned.Agents = make([]Agent, len(src.Agents))
		for i := range src.Agents {
			cloned.Agents[i] = src.Agents[i]
			if src.Agents[i].Skills != nil {
				cloned.Agents[i].Skills = append([]Skill(nil), src.Agents[i].Skills...)
			}
		}
	}
	return &cloned
}

func GetOutput(root string, deviceID string, ttl time.Duration) (*Output, error) {
	return GetOutputForApp(root, "", deviceID, ttl)
}

func GetOutputForApp(root string, appDir string, deviceID string, ttl time.Duration) (*Output, error) {
	resolvedAppDir := resolveAppDir(root, appDir)
	structCache.mu.Lock()
	cacheHit := structCache.output != nil && structCache.root == root && structCache.appDir == resolvedAppDir && structCache.device == deviceID && time.Now().Before(structCache.expires)
	var cached *Output
	if cacheHit {
		cached = cloneOutput(structCache.output)
	}
	structCache.mu.Unlock()
	if cacheHit {
		layoutChanged, err := agentDirectoryLayoutChanged(root, cached)
		if err != nil {
			return nil, err
		}
		if !layoutChanged {
			if err := refreshAgentSkills(cached); err != nil {
				return nil, err
			}
			refreshDynamicOutputFields(cached)
			return cached, nil
		}
	}
	output, err := buildOutput(root, resolvedAppDir, deviceID)
	if err != nil {
		return nil, err
	}
	structCache.mu.Lock()
	structCache.root = root
	structCache.appDir = resolvedAppDir
	structCache.device = deviceID
	structCache.output = output
	structCache.expires = time.Now().Add(ttl)
	structCache.mu.Unlock()
	cloned := cloneOutput(output)
	refreshDynamicOutputFields(cloned)
	return cloned, nil
}

func GetOutputForAppAndChat(root string, appDir string, deviceID string, ttl time.Duration, chatID string) (*Output, error) {
	output, err := GetOutputForApp(root, appDir, deviceID, ttl)
	if err != nil {
		return nil, err
	}
	refreshAgentSandboxFields(output, resolveAppDir(root, appDir), chatID)
	return output, nil
}

func GetOutputWithPlugins(root string, deviceID string, ttl time.Duration, pluginDir string, items []PluginRuntime) (*Output, error) {
	return GetOutputWithPluginsForApp(root, "", deviceID, ttl, pluginDir, items)
}

func GetOutputWithPluginsForApp(root string, appDir string, deviceID string, ttl time.Duration, pluginDir string, items []PluginRuntime) (*Output, error) {
	output, err := GetOutputForApp(root, appDir, deviceID, ttl)
	if err != nil {
		return nil, err
	}
	cloned := *output
	cloned.Plugins = DetectRunningPluginKeys(pluginDir, items)
	return &cloned, nil
}

func GetOutputWithPluginsForAppAndChat(root string, appDir string, deviceID string, ttl time.Duration, chatID string, pluginDir string, items []PluginRuntime) (*Output, error) {
	output, err := GetOutputForAppAndChat(root, appDir, deviceID, ttl, chatID)
	if err != nil {
		return nil, err
	}
	cloned := *output
	cloned.Plugins = DetectRunningPluginKeys(pluginDir, items)
	return &cloned, nil
}

func GetOutputJSON(root string, deviceID string, ttl time.Duration) ([]byte, error) {
	return GetOutputJSONForApp(root, "", deviceID, ttl)
}

func GetOutputJSONForApp(root string, appDir string, deviceID string, ttl time.Duration) ([]byte, error) {
	output, err := GetOutputForApp(root, appDir, deviceID, ttl)
	if err != nil {
		return nil, err
	}
	out, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, err
	}
	return out, nil
}

func GetOutputJSONForAppAndChat(root string, appDir string, deviceID string, ttl time.Duration, chatID string) ([]byte, error) {
	output, err := GetOutputForAppAndChat(root, appDir, deviceID, ttl, chatID)
	if err != nil {
		return nil, err
	}
	out, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, err
	}
	return out, nil
}

func FlushCache() {
	structCache.mu.Lock()
	structCache.root = ""
	structCache.appDir = ""
	structCache.device = ""
	structCache.output = nil
	structCache.expires = time.Time{}
	structCache.mu.Unlock()
}

func GetAgentIDs(root string, deviceID string, ttl time.Duration) ([]string, error) {
	_ = deviceID
	_ = ttl
	return ListAgentIDs(root)
}

func GetAgentByID(root string, deviceID string, ttl time.Duration, agentID string) (*Agent, error) {
	output, err := GetOutput(root, deviceID, ttl)
	if err != nil {
		return nil, err
	}
	for _, a := range output.Agents {
		if a.AgentID == agentID {
			cp := a
			return &cp, nil
		}
	}
	return nil, nil
}

func GetSkillNames(root string, deviceID string, ttl time.Duration, agentID string) ([]string, error) {
	agent, err := GetAgentByID(root, deviceID, ttl, agentID)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, fmt.Errorf("agent %q not found", agentID)
	}
	names := make([]string, len(agent.Skills))
	for i, s := range agent.Skills {
		names[i] = s.Name
	}
	return names, nil
}
