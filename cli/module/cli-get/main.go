package main

import (
	"agent-scanner/agentcore"
	"bytes"
	messageinsert "cli-get/messageinsert"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"knowledge/knowledgecore"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	taskexec "cli-get/taskexec"

	_ "github.com/glebarez/go-sqlite"
)

// ═══════════════════════════════════════════════════════════════════════════
// Skill & Agent metadata (shared from agent module)
// ═══════════════════════════════════════════════════════════════════════════

type Skill = agentcore.Skill
type Agent = agentcore.Agent
type AgentOutput = agentcore.Output

var (
	osExecutable = os.Executable
)

const (
	logTypeChatCompletionRequest  = 0
	logTypeChatCompletionResponse = 1
	logTypeCLIGet                 = 2
	logTypeCLIPub                 = 3
	defaultTaskTimeout            = 180 * time.Second
	sandboxModeFilePick           = "filepick"
	sandboxModeNet                = "net"
	sandboxModeFilePickNet        = "filepick_net"
)

type logEntry struct {
	AgentID   string
	ChatID    string
	Content   string
	LogType   int
	CreatedAt string
}

type pooledDB struct {
	mu   sync.Mutex
	path string
	db   *sql.DB
}

type asyncLogger struct {
	ch chan logEntry
}

type activeCmd struct {
	cancel context.CancelFunc
	agent  string
	chat   string
	tid    string
	cmd    string
}

var (
	sharedDataDB = pooledDB{}
	loggerState  struct {
		mu     sync.Mutex
		path   string
		logger *asyncLogger
	}
	activeCmdMu  sync.Mutex
	activeCmdMap = make(map[string]*activeCmd)
)

// ═══════════════════════════════════════════════════════════════════════════
// System info detection
// ═══════════════════════════════════════════════════════════════════════════

func detectTerminal() string {
	return os.Getenv("SHELL")
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

func detectSys() string {
	out, err := exec.Command("uname", "-sr").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func generateDeviceID() string {
	var parts []string
	if h, err := os.Hostname(); err == nil {
		parts = append(parts, h)
	}
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("sh", "-c",
			"ioreg -rd1 -c IOPlatformExpertDevice | awk '/IOPlatformUUID/{print $3}' | tr -d '\"'").Output()
		if err == nil && len(strings.TrimSpace(string(out))) > 0 {
			parts = append(parts, strings.TrimSpace(string(out)))
		}
	case "linux":
		if data, err := os.ReadFile("/etc/machine-id"); err == nil {
			parts = append(parts, strings.TrimSpace(string(data)))
		}
	}
	if len(parts) == 0 {
		parts = append(parts, runtime.GOOS, runtime.GOARCH)
	}
	hash := sha256.Sum256([]byte(strings.Join(parts, ":")))
	return fmt.Sprintf("%x", hash[:16])
}

func getAgentOutput(root string, deviceID string, ttl time.Duration) (*AgentOutput, error) {
	return agentcore.GetOutputWithPlugins(root, deviceID, ttl, detectPluginDir(), detectPluginRuntimes())
}

func detectPluginRuntimes() []agentcore.PluginRuntime {
	appPath, err := osExecutable()
	if err != nil {
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
	if err != nil {
		return nil
	}
	var items []agentcore.PluginRuntime
	if err := json.Unmarshal(out, &items); err != nil {
		return nil
	}
	return items
}

func pluginRuntimeListCommandArgs(appPath string) []string {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(appPath)))
	if base == "integration" || base == "proxy" || strings.HasPrefix(base, "integration.") || strings.HasPrefix(base, "proxy.") {
		return []string{"connect", "meta-list"}
	}
	return []string{"list-meta"}
}

func detectPluginDir() string {
	appPath, err := osExecutable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(appPath), "plugins")
}

func detectPluginKeys() []string {
	return agentcore.DetectRunningPluginKeys(detectPluginDir(), detectPluginRuntimes())
}

func getDataDBPath() (string, error) {
	return knowledgecore.DBPath(".")
}

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
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(0)
	if err := ensureLogSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	sharedDataDB.path = path
	sharedDataDB.db = db
	return db, nil
}

func ensureLogSchema(db *sql.DB) error {
	if db == nil {
		return nil
	}
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS agent_message_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id TEXT NOT NULL DEFAULT '',
			chat_id TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			log_type INTEGER NOT NULL,
			created_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_agent_message_log_agent_chat_type_time
			ON agent_message_log(agent_id, chat_id, log_type, created_at);
	`)
	if err != nil {
		return err
	}
	if err := messageinsert.EnsureSchema(db); err != nil {
		return err
	}
	return ensureSandboxStateSchema(db)
}

func ensureSandboxStateSchema(db *sql.DB) error {
	if db == nil {
		return nil
	}
	rows, err := db.Query(`PRAGMA table_info(cli_sandbox_state)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var cols []sandboxStateColumnInfo
	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultV   sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultV, &pk); err != nil {
			return err
		}
		cols = append(cols, sandboxStateColumnInfo{
			Name: strings.TrimSpace(name),
			Type: strings.ToUpper(strings.TrimSpace(columnType)),
			PK:   pk,
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if needsSandboxStateRebuild(cols) {
		if _, err := db.Exec(`DROP TABLE IF EXISTS cli_sandbox_state`); err != nil {
			return err
		}
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS cli_sandbox_state (
			chat_id TEXT NOT NULL DEFAULT '',
			sandbox_exe TEXT NOT NULL DEFAULT '',
			allowed_dir TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL,
			PRIMARY KEY (chat_id)
		);
		CREATE INDEX IF NOT EXISTS idx_cli_sandbox_state_updated_at
			ON cli_sandbox_state(updated_at);
	`)
	return err
}

type sandboxStateColumnInfo struct {
	Name string
	Type string
	PK   int
}

type sandboxStateRecord struct {
	Mode       string
	AllowedDir string
	UpdatedAt  string
}

func needsSandboxStateRebuild(cols []sandboxStateColumnInfo) bool {
	if len(cols) == 0 {
		return false
	}
	if len(cols) != 4 {
		return true
	}
	expected := map[string]string{
		"chat_id":     "TEXT",
		"sandbox_exe": "TEXT",
		"allowed_dir": "TEXT",
		"updated_at":  "TEXT",
	}
	for _, col := range cols {
		wantType, ok := expected[col.Name]
		if !ok || col.Type != wantType {
			return true
		}
		if col.Name == "chat_id" && col.PK != 1 {
			return true
		}
	}
	return false
}

func normalizeSandboxMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case sandboxModeFilePick:
		return sandboxModeFilePick
	case sandboxModeNet:
		return sandboxModeNet
	case sandboxModeFilePickNet:
		return sandboxModeFilePickNet
	default:
		return ""
	}
}

func sandboxModeForLog(value string) string {
	mode := normalizeSandboxMode(value)
	if mode == "" {
		return "off"
	}
	return mode
}

func getAsyncLogger() (*asyncLogger, error) {
	path, err := getDataDBPath()
	if err != nil {
		return nil, err
	}

	loggerState.mu.Lock()
	defer loggerState.mu.Unlock()

	if loggerState.logger != nil && loggerState.path == path {
		return loggerState.logger, nil
	}
	logger := &asyncLogger{ch: make(chan logEntry, 512)}
	go func() {
		for item := range logger.ch {
			db, err := getDataDB()
			if err != nil {
				continue
			}
			_, _ = db.Exec(
				`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES (?,?,?,?,?)`,
				strings.TrimSpace(item.AgentID),
				strings.TrimSpace(item.ChatID),
				item.Content,
				item.LogType,
				item.CreatedAt,
			)
		}
	}()
	loggerState.path = path
	loggerState.logger = logger
	return logger, nil
}

func appendAsyncLog(agentID, chatID, content string, logType int) {
	logger, err := getAsyncLogger()
	if err != nil {
		return
	}
	entry := logEntry{
		AgentID:   strings.TrimSpace(agentID),
		ChatID:    strings.TrimSpace(chatID),
		Content:   content,
		LogType:   logType,
		CreatedAt: time.Now().Format("2006-01-02T15:04:05.000"),
	}
	select {
	case logger.ch <- entry:
	default:
		go func() {
			db, err := getDataDB()
			if err != nil {
				return
			}
			_, _ = db.Exec(
				`INSERT INTO agent_message_log (agent_id, chat_id, content, log_type, created_at) VALUES (?,?,?,?,?)`,
				entry.AgentID, entry.ChatID, entry.Content, entry.LogType, entry.CreatedAt,
			)
		}()
	}
}

func activeCmdKey(agentID, chatID, tid, cmd string) string {
	return agentID + "|" + chatID + "|" + tid + "|" + cmd
}

func registerActiveCmd(item *activeCmd) string {
	key := activeCmdKey(item.agent, item.chat, item.tid, item.cmd)
	activeCmdMu.Lock()
	activeCmdMap[key] = item
	activeCmdMu.Unlock()
	return key
}

func unregisterActiveCmd(key string) {
	activeCmdMu.Lock()
	delete(activeCmdMap, key)
	activeCmdMu.Unlock()
}

func SetSandboxMode(agentID, chatID, mode string) error {
	return SetSandboxModeWithDirectory(agentID, chatID, mode, "")
}

func SetSandboxModeWithDirectory(agentID, chatID, mode, allowedDir string) error {
	agentID = strings.TrimSpace(agentID)
	chatID = strings.TrimSpace(chatID)
	allowedDir = strings.TrimSpace(allowedDir)
	if chatID == "" {
		return fmt.Errorf("chatID is required")
	}
	rawMode := strings.TrimSpace(mode)
	mode = normalizeSandboxMode(mode)
	if rawMode != "" && mode == "" {
		return fmt.Errorf("sandbox mode must be one of filepick,net,filepick_net")
	}
	db, err := getDataDB()
	if err != nil {
		return err
	}
	currentState, _, err := sandboxStateSetting(chatID)
	if err != nil {
		return err
	}
	if mode != sandboxModeFilePick && mode != sandboxModeFilePickNet {
		allowedDir = ""
	}
	if mode == "" {
		_, err = db.Exec(`DELETE FROM cli_sandbox_state WHERE chat_id = ?`, chatID)
		if err == nil {
			fmt.Fprintf(os.Stderr, "cli-get: sandbox change agentId=%s chatId=%s from=%s to=off\n",
				agentID, chatID, sandboxModeForLog(currentState.Mode))
		}
		return err
	}
	_, err = db.Exec(
		`INSERT INTO cli_sandbox_state (chat_id, sandbox_exe, allowed_dir, updated_at)
		 VALUES (?,?,?,?)
		 ON CONFLICT(chat_id) DO UPDATE SET sandbox_exe = excluded.sandbox_exe, allowed_dir = excluded.allowed_dir, updated_at = excluded.updated_at`,
		chatID,
		mode,
		allowedDir,
		time.Now().Format("2006-01-02T15:04:05.000"),
	)
	if err == nil {
		fmt.Fprintf(os.Stderr, "cli-get: sandbox change agentId=%s chatId=%s from=%s to=%s\n",
			agentID, chatID, sandboxModeForLog(currentState.Mode), sandboxModeForLog(mode))
	}
	return err
}

func SetSandboxExe(agentID, chatID string, enabled bool) error {
	if enabled {
		return SetSandboxModeWithDirectory(agentID, chatID, sandboxModeFilePickNet, "")
	}
	return SetSandboxModeWithDirectory(agentID, chatID, "", "")
}

func sandboxStateSetting(chatID string) (sandboxStateRecord, bool, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return sandboxStateRecord{}, false, nil
	}
	db, err := getDataDB()
	if err != nil {
		return sandboxStateRecord{}, false, err
	}
	var state sandboxStateRecord
	err = db.QueryRow(
		`SELECT sandbox_exe, allowed_dir, updated_at FROM cli_sandbox_state WHERE chat_id = ?`,
		chatID,
	).Scan(&state.Mode, &state.AllowedDir, &state.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return sandboxStateRecord{}, false, nil
	}
	if err != nil {
		return sandboxStateRecord{}, false, err
	}
	state.Mode = normalizeSandboxMode(state.Mode)
	if state.Mode == "" {
		return sandboxStateRecord{}, false, nil
	}
	state.AllowedDir = strings.TrimSpace(state.AllowedDir)
	return state, true, nil
}

func sandboxExeSetting(agentID, chatID string) (bool, bool, error) {
	state, found, err := sandboxStateSetting(chatID)
	if err != nil {
		return false, false, err
	}
	return state.Mode != "", found, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// HTTP data structures
// ═══════════════════════════════════════════════════════════════════════════

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GetRequest struct {
	Model    string       `json:"model"`
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

type queuedTask struct {
	task     TaskContent
	metadata *AgentOutput
}

type queuedPublish struct {
	result   *ResultPayload
	metadata *AgentOutput
}

// ═══════════════════════════════════════════════════════════════════════════
// HTTP client
// ═══════════════════════════════════════════════════════════════════════════

func createHTTPClient(connectTimeout, socketTimeout, totalTimeout, idleTimeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout:   connectTimeout,
		KeepAlive: 60 * time.Second,
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := dialer.DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			if tc, ok := conn.(*net.TCPConn); ok {
				_ = tc.SetNoDelay(true)
			}
			return conn, nil
		},
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       idleTimeout,
		ResponseHeaderTimeout: socketTimeout,
		TLSHandshakeTimeout:   10 * time.Second,
		ForceAttemptHTTP2:     false,
		TLSNextProto:          map[string]func(string, *tls.Conn) http.RoundTripper{},
	}
	return &http.Client{
		Transport: transport,
		Timeout:   totalTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Core logic: heartbeat, execute, publish
// ═══════════════════════════════════════════════════════════════════════════

// Heartbeat sends agent metadata to cli/get and returns a task if available.
func Heartbeat(client *http.Client, host string, metadata *AgentOutput) (*TaskContent, error) {
	reqBody, _ := json.Marshal(GetRequest{
		Model:    "",
		Messages: []Message{{Role: "user", Content: ""}},
		Metadata: metadata,
	})

	url := strings.TrimRight(host, "/") + "/cli/get"
	resp, err := client.Post(url, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("heartbeat request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read heartbeat response: %w", err)
	}

	if resp.StatusCode != 200 {
		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			location := strings.TrimSpace(resp.Header.Get("Location"))
			if location != "" {
				return nil, fmt.Errorf("heartbeat redirect HTTP %d to %s", resp.StatusCode, location)
			}
		}
		return nil, fmt.Errorf("heartbeat HTTP %d", resp.StatusCode)
	}

	var rp ResponsePayload
	if err := json.Unmarshal(body, &rp); err != nil {
		return nil, fmt.Errorf("parse heartbeat response: %w", err)
	}
	if rp.Code != 200 {
		return nil, fmt.Errorf("heartbeat code %d", rp.Code)
	}

	if len(rp.Choices) == 0 || rp.Choices[0].Message.Content == nil {
		return nil, nil // no task
	}

	contentStr, ok := rp.Choices[0].Message.Content.(string)
	if !ok || contentStr == "" {
		return nil, nil
	}

	var task TaskContent
	if err := json.Unmarshal([]byte(contentStr), &task); err != nil {
		return nil, fmt.Errorf("parse task content: %w", err)
	}
	agentID, chatID := extractAgentAndChatIDs(metadata, &task)
	appendAsyncLog(agentID, chatID, contentStr, logTypeCLIGet)
	return &task, nil
}

// ExecuteTask runs the shell command and returns the result payload.
func ExecuteTask(task *TaskContent, terminal string) *ResultPayload {
	var activeKey string
	result := taskexec.Execute(task.Cmd, taskexec.Options{
		Shell:     terminal,
		TimeoutMs: task.Timeout,
		AgentID:   task.AgentId,
		ChatID:    task.Chat,
		Tid:       task.Tid,
		OnStart: func(cmd *taskexec.ActiveCmd) {
			activeKey = registerActiveCmd(&activeCmd{
				cancel: cmd.Cancel,
				agent:  cmd.AgentID,
				chat:   cmd.ChatID,
				tid:    cmd.Tid,
				cmd:    cmd.RawCmd,
			})
		},
	})
	if activeKey != "" {
		defer unregisterActiveCmd(activeKey)
	}

	// GZIP + Base64
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write([]byte(result.Output))
	_ = gz.Close()
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())

	return &ResultPayload{
		Status:  result.Status,
		AgentId: task.AgentId,
		Suffix:  task.Suffix,
		Chat:    task.Chat,
		Type:    task.Type,
		Cmd:     encoded,
		Tid:     task.Tid,
	}
}

// PublishResult sends the execution result to cli/pub.
func PublishResult(client *http.Client, host string, result *ResultPayload, metadata *AgentOutput) error {
	payloadResult := result
	pendingInsertItems := result.Insert
	if len(pendingInsertItems) == 0 {
		items, err := loadPendingMessageInsertPublishItems(strings.TrimSpace(result.Chat), messageInsertPublishLimit)
		if err != nil {
			fmt.Fprintf(os.Stderr, "message_insert load pending error chat=%s: %v\n", strings.TrimSpace(result.Chat), err)
		} else if len(items) > 0 {
			result.Insert = items
			payloadResult = result
			pendingInsertItems = items
		}
	}
	resultJSON, _ := json.Marshal(payloadResult)
	appendAsyncLog(strings.TrimSpace(result.AgentId), strings.TrimSpace(result.Chat), string(resultJSON), logTypeCLIPub)

	reqBody, _ := json.Marshal(PubRequest{
		Model:    "",
		Messages: []Message{{Role: "user", Content: string(resultJSON)}},
		Metadata: metadata,
	})

	url := strings.TrimRight(host, "/") + "/cli/pub"
	resp, err := client.Post(url, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("publish request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read publish response: %w", err)
	}

	if resp.StatusCode != 200 {
		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			location := strings.TrimSpace(resp.Header.Get("Location"))
			if location != "" {
				return fmt.Errorf("publish redirect HTTP %d to %s", resp.StatusCode, location)
			}
		}
		return fmt.Errorf("publish HTTP %d", resp.StatusCode)
	}

	var rp ResponsePayload
	if err := json.Unmarshal(body, &rp); err != nil {
		return fmt.Errorf("parse publish response: %w", err)
	}
	if rp.Code != 200 {
		return fmt.Errorf("publish code %d", rp.Code)
	}
	if len(pendingInsertItems) > 0 {
		if err := markPublishedMessageInsertPublishItems(strings.TrimSpace(result.Chat), pendingInsertItems); err != nil {
			fmt.Fprintf(os.Stderr, "message_insert mark published error chat=%s tids=%d: %v\n", strings.TrimSpace(result.Chat), len(pendingInsertItems), err)
		}
	}
	return nil
}

func gzipBase64String(raw string) string {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write([]byte(raw))
	_ = gz.Close()
	return base64.StdEncoding.EncodeToString(buf.Bytes())
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

func sandboxStateForTask(metadata *AgentOutput, task *TaskContent) (sandboxStateRecord, error) {
	if taskShouldBypassSandbox(task) {
		return sandboxStateRecord{}, nil
	}
	agentID, chatID := extractAgentAndChatIDs(metadata, task)
	state, found, err := sandboxStateSetting(chatID)
	if err != nil {
		return sandboxStateRecord{}, err
	}
	if found {
		fmt.Fprintf(
			os.Stderr,
			"cli-get: sandbox lookup agentId=%s chatId=%s mode=%s allowedDir=%s\n",
			strings.TrimSpace(agentID),
			strings.TrimSpace(chatID),
			sandboxModeForLog(state.Mode),
			strings.TrimSpace(state.AllowedDir),
		)
		return state, nil
	}
	return sandboxStateRecord{}, nil
}

func taskShouldBypassSandbox(task *TaskContent) bool {
	if task == nil {
		return true
	}
	if task.SubOps.Exempted {
		return true
	}
	if taskIsInternalTokenCommand(task) {
		return true
	}
	return taskUsesInternalRuntimePaths(task)
}

func taskIsInternalTokenCommand(task *TaskContent) bool {
	if task == nil {
		return false
	}
	cmd := strings.TrimSpace(task.Cmd)
	if cmd == "" {
		return false
	}
	exe, err := osExecutable()
	if err != nil {
		return false
	}
	exe = strings.TrimSpace(exe)
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

func taskUsesInternalRuntimePaths(task *TaskContent) bool {
	if task == nil {
		return false
	}
	runtimeBase := cliGetRuntimeBaseDir()
	if runtimeBase == "" {
		return false
	}
	paths := make([]string, 0, len(task.SubOps.R)+len(task.SubOps.W))
	paths = append(paths, task.SubOps.R...)
	paths = append(paths, task.SubOps.W...)
	internalCount := 0
	for _, raw := range paths {
		path := normalizeTaskPath(raw)
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

func normalizeTaskPath(raw string) string {
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

func cliGetRuntimeBaseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	home = strings.TrimSpace(home)
	if home == "" {
		return ""
	}
	return filepath.Join(home, "Library", "Application Support", "deepright")
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

func cliGetConfigPaths() []string {
	executable, err := osExecutable()
	if err != nil {
		return nil
	}
	executable = filepath.Clean(executable)
	execDir := filepath.Dir(executable)

	paths := []string{
		filepath.Join(execDir, "config", "config.json"),
	}
	if filepath.Base(execDir) == "MacOS" && filepath.Base(filepath.Dir(execDir)) == "Contents" {
		contentsDir := filepath.Dir(execDir)
		paths = append(paths, filepath.Join(contentsDir, "Resources", "config", "config.json"))
	}
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, "config", "config.json"))
	}
	return paths
}

func configuredSandboxApp() string {
	for _, path := range cliGetConfigPaths() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}
		if value := strings.TrimSpace(fmt.Sprint(raw["sandbox_app"])); value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func sandboxExecutableCandidates(raw, mode string) []string {
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

func resolveSandboxExecutablePath(raw, mode string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = configuredSandboxApp()
	}
	if raw == "" {
		return "", nil
	}
	mode = normalizeSandboxMode(mode)
	if mode == "" {
		return "", nil
	}

	executable, err := osExecutable()
	if err != nil {
		return "", fmt.Errorf("locate executable: %w", err)
	}
	baseDir := filepath.Dir(filepath.Clean(executable))
	if !filepath.IsAbs(raw) {
		raw = filepath.Join(baseDir, raw)
	}
	raw = filepath.Clean(raw)
	for _, candidate := range sandboxExecutableCandidates(raw, mode) {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("sandbox_app not found for mode=%s: %s", mode, raw)
}

func ExecuteTaskViaSandboxApp(task *TaskContent, sandboxExecutable, allowedDir string) *ResultPayload {
	timeout := defaultTaskTimeout + 5*time.Second
	if task != nil && task.Timeout > 0 {
		timeout = time.Duration(task.Timeout)*time.Millisecond + 5*time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := []string{"--cmd", task.Cmd}
	if strings.TrimSpace(allowedDir) != "" {
		args = append(args, "--allowed-dir", strings.TrimSpace(allowedDir))
	}
	args = append(args, "--timeout", strconv.Itoa(task.Timeout))
	cmd := exec.CommandContext(ctx, sandboxExecutable, args...)
	cmd.Env = os.Environ()

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	activeKey := ""
	if err := cmd.Start(); err != nil {
		text := strings.TrimSpace(output.String())
		if text == "" {
			text = err.Error()
		}
		return &ResultPayload{
			Status:  1,
			AgentId: task.AgentId,
			Suffix:  task.Suffix,
			Chat:    task.Chat,
			Type:    task.Type,
			Cmd:     gzipBase64String(text),
			Tid:     task.Tid,
		}
	}
	activeKey = registerActiveCmd(&activeCmd{
		cancel: cancel,
		agent:  task.AgentId,
		chat:   task.Chat,
		tid:    task.Tid,
		cmd:    task.Cmd,
	})
	defer unregisterActiveCmd(activeKey)

	err := cmd.Wait()
	text := output.String()
	status := 0

	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		status = 1
		text = "命令被终止"
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		status = 1
		if strings.TrimSpace(text) == "" {
			text = "命令执行超时"
		}
	case err != nil:
		status = 1
		if strings.TrimSpace(text) == "" {
			text = err.Error()
		}
	}

	return &ResultPayload{
		Status:  status,
		AgentId: task.AgentId,
		Suffix:  task.Suffix,
		Chat:    task.Chat,
		Type:    task.Type,
		Cmd:     gzipBase64String(text),
		Tid:     task.Tid,
	}
}

func ProcessTask(client *http.Client, host string, task *TaskContent, terminal string, metadata *AgentOutput, sandboxExecutable string) error {
	result, err := BuildTaskResult(task, terminal, metadata, sandboxExecutable)
	if err != nil {
		return err
	}
	return PublishResult(client, host, result, metadata)
}

func BuildTaskResult(task *TaskContent, terminal string, metadata *AgentOutput, sandboxExecutable string) (*ResultPayload, error) {
	state, err := sandboxStateForTask(metadata, task)
	if err != nil {
		return nil, err
	}
	if state.Mode != "" {
		resolvedSandboxExecutable, err := resolveSandboxExecutablePath(sandboxExecutable, state.Mode)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(resolvedSandboxExecutable) == "" {
			return nil, fmt.Errorf("sandbox enabled for agent=%s chat=%s but sandbox_app is not configured", strings.TrimSpace(task.AgentId), strings.TrimSpace(task.Chat))
		}
		return ExecuteTaskViaSandboxApp(task, resolvedSandboxExecutable, state.AllowedDir), nil
	}
	return ExecuteTask(task, terminal), nil
}

func isTaskExpired(task *TaskContent, now time.Time) bool {
	if task == nil || task.Ddl <= 0 {
		return false
	}
	return now.UnixMilli() > task.Ddl
}

func publishWithRetry(client *http.Client, host string, result *ResultPayload, metadata *AgentOutput, retryInterval time.Duration, retryTimes int) error {
	for attempt := 0; ; attempt++ {
		err := PublishResult(client, host, result, metadata)
		if err == nil {
			if attempt > 0 {
				fmt.Printf("cli-get: publish retry success tid=%s retries=%d\n", strings.TrimSpace(result.Tid), attempt)
			}
			return nil
		}
		if attempt >= retryTimes {
			fmt.Fprintf(os.Stderr, "cli-get: publish give up tid=%s retries=%d err=%v\n", strings.TrimSpace(result.Tid), attempt, err)
			return err
		}
		nextRetry := attempt + 1
		fmt.Fprintf(os.Stderr, "cli-get: publish retry scheduled tid=%s retry=%d max=%d wait_ms=%d err=%v\n",
			strings.TrimSpace(result.Tid), nextRetry, retryTimes, retryInterval.Milliseconds(), err)
		time.Sleep(retryInterval)
	}
}

func runExecuteWorker(taskQueue <-chan queuedTask, publishQueue chan<- queuedPublish, terminal string, sandboxExecutable string) {
	for item := range taskQueue {
		task := item.task
		if isTaskExpired(&task, time.Now()) {
			fmt.Fprintf(os.Stderr, "cli-get: drop expired task agentId=%s chat=%s tid=%s ddl=%d now=%d cmd=%q\n",
				strings.TrimSpace(task.AgentId),
				strings.TrimSpace(task.Chat),
				strings.TrimSpace(task.Tid),
				task.Ddl,
				time.Now().UnixMilli(),
				task.Cmd,
			)
			continue
		}
		result, err := BuildTaskResult(&task, terminal, item.metadata, sandboxExecutable)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cli-get: execute prepare error tid=%s: %v\n", strings.TrimSpace(task.Tid), err)
			continue
		}
		publishQueue <- queuedPublish{result: result, metadata: item.metadata}
		fmt.Printf("cli-get: publish queued tid=%s pending=%d\n", strings.TrimSpace(task.Tid), len(publishQueue))
	}
}

func runPublishWorker(client *http.Client, host string, publishQueue <-chan queuedPublish, retryInterval time.Duration, retryTimes int) {
	for item := range publishQueue {
		if item.result == nil {
			continue
		}
		if err := publishWithRetry(client, host, item.result, item.metadata, retryInterval, retryTimes); err != nil {
			fmt.Fprintf(os.Stderr, "cli-get: publish error tid=%s: %v\n", strings.TrimSpace(item.result.Tid), err)
		}
	}
}

func extractAgentAndChatIDs(metadata *AgentOutput, task *TaskContent) (string, string) {
	agentID := ""
	if task != nil {
		agentID = strings.TrimSpace(task.AgentId)
	}
	if agentID == "" && metadata != nil && len(metadata.Agents) == 1 {
		agentID = strings.TrimSpace(metadata.Agents[0].AgentID)
	}
	chatID := ""
	if task != nil {
		chatID = strings.TrimSpace(task.Chat)
	}
	return agentID, chatID
}

// ═══════════════════════════════════════════════════════════════════════════
// CLI flags & main
// ═══════════════════════════════════════════════════════════════════════════

// Config holds all CLI configuration.
type Config struct {
	Host              string
	AgentDir          string
	Device            string
	SandboxApp        string
	AgentCacheMs      int
	SleepMs           int
	Thread            int
	Queue             int
	RetryIntervalMs   int
	RetryTimes        int
	HTTPTimeout       int
	HTTPConnTimeout   int
	HTTPSocketTimeout int
	IdleTimeout       int
}

const defaultUpstreamHost = "https://www.deepright.cn"

func main() {
	var cfg Config

	flag.StringVar(&cfg.Host, "host", defaultUpstreamHost, "server URL")
	flag.StringVar(&cfg.AgentDir, "agent-dir", "", "agent directory (required)")
	flag.StringVar(&cfg.Device, "device", "", "device ID (auto-generated if empty)")
	flag.StringVar(&cfg.SandboxApp, "sandbox_app", "", "relative path to CLI_SANDBOX(.app); defaults to config/config.json sandbox_app")
	flag.IntVar(&cfg.AgentCacheMs, "agent-cache", 120000, "agent metadata cache TTL in ms")
	flag.IntVar(&cfg.SleepMs, "sleep", 3000, "sleep after heartbeat errors in ms")
	flag.IntVar(&cfg.Thread, "thread", 20, "worker pool size")
	flag.IntVar(&cfg.Queue, "queue", 1000, "local task queue size")
	flag.IntVar(&cfg.RetryIntervalMs, "retry_interval", 10000, "cli/pub retry interval in ms")
	flag.IntVar(&cfg.RetryTimes, "retry_times", 1, "cli/pub retry count after first failure")
	flag.IntVar(&cfg.HTTPTimeout, "http_timeout", 60000, "HTTP total timeout in ms")
	flag.IntVar(&cfg.HTTPConnTimeout, "http_connect_timeout", 15000, "HTTP connect timeout in ms")
	flag.IntVar(&cfg.HTTPSocketTimeout, "http_socket_timeout", 45000, "HTTP read timeout in ms")
	flag.IntVar(&cfg.IdleTimeout, "idle_timeout", 90, "connection pool idle timeout in seconds")
	flag.Parse()

	if cfg.AgentDir == "" {
		fmt.Fprintln(os.Stderr, "error: --agent-dir is required")
		os.Exit(1)
	}
	if strings.TrimSpace(cfg.SandboxApp) == "" {
		cfg.SandboxApp = configuredSandboxApp()
	}

	terminal := detectTerminal()
	if terminal == "" {
		fmt.Fprintln(os.Stderr, "error: cannot detect SHELL")
		os.Exit(1)
	}

	client := createHTTPClient(
		time.Duration(cfg.HTTPConnTimeout)*time.Millisecond,
		time.Duration(cfg.HTTPSocketTimeout)*time.Millisecond,
		time.Duration(cfg.HTTPTimeout)*time.Millisecond,
		time.Duration(cfg.IdleTimeout)*time.Second,
	)
	if cfg.Thread <= 0 {
		fmt.Fprintln(os.Stderr, "error: --thread must be greater than 0")
		os.Exit(1)
	}
	if cfg.Queue <= 0 {
		fmt.Fprintln(os.Stderr, "error: --queue must be greater than 0")
		os.Exit(1)
	}
	if cfg.RetryIntervalMs < 0 {
		fmt.Fprintln(os.Stderr, "error: --retry_interval must be >= 0")
		os.Exit(1)
	}
	if cfg.RetryTimes < 0 {
		fmt.Fprintln(os.Stderr, "error: --retry_times must be >= 0")
		os.Exit(1)
	}

	agentTTL := time.Duration(cfg.AgentCacheMs) * time.Millisecond
	sleepDur := time.Duration(cfg.SleepMs) * time.Millisecond
	retryInterval := time.Duration(cfg.RetryIntervalMs) * time.Millisecond
	heartbeatBackoff := sleepDur
	maxHeartbeatBackoff := 15 * time.Second
	taskQueue := make(chan queuedTask, cfg.Queue)
	publishQueue := make(chan queuedPublish, cfg.Queue)

	for i := 0; i < cfg.Thread; i++ {
		go runExecuteWorker(taskQueue, publishQueue, terminal, cfg.SandboxApp)
		go runPublishWorker(client, cfg.Host, publishQueue, retryInterval, cfg.RetryTimes)
	}

	fmt.Printf("cli-get started: host=%s, agent-dir=%s, threads=%d, queue=%d, sleep=%dms, retry_interval=%dms, retry_times=%d, sandbox_app=%s\n",
		cfg.Host, cfg.AgentDir, cfg.Thread, cfg.Queue, cfg.SleepMs, cfg.RetryIntervalMs, cfg.RetryTimes, cfg.SandboxApp)

	// Master heartbeat loop
	for {
		if len(taskQueue) >= cap(taskQueue) {
			fmt.Printf("cli-get: task queue full pending=%d capacity=%d, skip /cli/get\n", len(taskQueue), cap(taskQueue))
			time.Sleep(sleepDur)
			continue
		}
		metadata, err := getAgentOutput(cfg.AgentDir, cfg.Device, agentTTL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "agent scan error: %v\n", err)
			time.Sleep(sleepDur)
			continue
		}

		task, err := Heartbeat(client, cfg.Host, metadata)
		if shouldSleepAfterHeartbeat(task, err) {
			fmt.Fprintf(os.Stderr, "heartbeat error: %v\n", err)
			time.Sleep(heartbeatBackoff)
			heartbeatBackoff = nextHeartbeatBackoff(heartbeatBackoff, sleepDur, maxHeartbeatBackoff)
			continue
		}
		heartbeatBackoff = sleepDur

		if task == nil {
			// No task, immediately retry heartbeat
			continue
		}

		// Dispatch task to worker pool, then immediately loop for next heartbeat
		t := *task
		taskQueue <- queuedTask{task: t, metadata: metadata}
		fmt.Printf("cli-get: task queued tid=%s pending=%d capacity=%d\n", strings.TrimSpace(t.Tid), len(taskQueue), cap(taskQueue))
	}
}
