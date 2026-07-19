package main

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"connect/connectsvc"
	"connect/feishusvc"
)

const (
	defaultFeishuPIDFile = "feishu.pid"
	defaultFeishuLogFile = "feishu.log"
	defaultFeishuRunLog  = "feishu.runtime.log"
)

var localSupportedCommands = supportedCommands()
var localMetaParams = []localMetaParamDefinition{
	{
		AppID:     "飞书开放平台（https://open.feishu.cn/app）中应用凭证的App ID ",
		AppSecret: "App Secret",
	},
}
var runFeishuCLI = feishusvc.RunCLI
var feishuSendTimeout = 180 * time.Second
var runUpstreamCommand = func(name string, args []string, stdout, stderr io.Writer) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
var runUpstreamQueryCommand = func(name string, args []string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%w", err)
	}
	return output, nil
}

type localMetaParamDefinition struct {
	AppID     string `json:"appId"`
	AppSecret string `json:"appSecret"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && connectsvc.IsHelpCommand(args[0]) {
		printHelp(stdout)
		return 0
	}
	if code, handled := handleProxyCommand(args, stdout, stderr); handled {
		return code
	}
	if code, handled := handleSnapshotQueryCommand(args, stdout, stderr); handled {
		return code
	}
	if handleLocalCommand(args, stdout) {
		return 0
	}
	if handled, code := runLocalLifecycleCommandFeishu(args, stdout, stderr); handled {
		return code
	}
	normalizedArgs, upstream := normalizeArgs(args, stderr)
	restore := setProxyConnectBinaryEnv(upstream)
	defer restore()
	if isSendLikeCommand(normalizedArgs) {
		return runFeishuCLIWithTimeout(normalizedArgs, stdout, stderr, feishuSendTimeout)
	}
	return runFeishuCLI(normalizedArgs, stdout, stderr)
}

func handleLocalCommand(args []string, stdout io.Writer) bool {
	if len(args) == 0 {
		return false
	}
	switch strings.TrimSpace(args[0]) {
	case "command":
		connectsvc.WriteJSON(stdout, localSupportedCommands)
		return true
	case "help":
		printHelp(stdout)
		return true
	case "name":
		connectsvc.WriteJSON(stdout, map[string]string{
			"key":  feishusvc.DefaultName,
			"name": feishusvc.DefaultDisplayName,
		})
		return true
	case "param":
		connectsvc.WriteJSON(stdout, localMetaParams)
		return true
	case "scope":
		connectsvc.WriteJSON(stdout, supportedScopes())
		return true
	case "schema":
		connectsvc.WriteJSON(stdout, responseSchemaDefinition())
		return true
	default:
		return false
	}
}

func handleProxyCommand(args []string, stdout, stderr io.Writer) (int, bool) {
	if len(args) == 0 {
		return 0, false
	}
	command := strings.TrimSpace(args[0])
	switch command {
	case "meta-get", "add-request":
	default:
		return 0, false
	}

	flags, err := connectsvc.ParseFlags(args[1:])
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1, true
	}
	if command == "add-request" && strings.TrimSpace(connectsvc.FirstValue(flags, "schema")) == "" {
		flags["schema"] = responseSchemaJSON()
	}

	binary, prefix := resolveProxyTarget()
	if strings.TrimSpace(binary) == "" {
		fmt.Fprintln(stderr, "upstream connect/integration binary not found")
		return 1, true
	}
	proxyArgs := rebuildArgs(command, flags)
	proxyArgs = append(prefix, proxyArgs...)
	if err := runUpstreamCommand(binary, proxyArgs, stdout, stderr); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1, true
	}
	return 0, true
}

func handleSnapshotQueryCommand(args []string, stdout, stderr io.Writer) (int, bool) {
	if len(args) == 0 {
		return 0, false
	}
	command := strings.TrimSpace(args[0])
	if command != "openid" && command != "search" {
		return 0, false
	}
	if len(args) > 1 && connectsvc.IsHelpCommand(args[1]) {
		printSnapshotCommandHelp(stdout, command)
		return 0, true
	}
	flags, err := connectsvc.ParseFlags(args[1:])
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1, true
	}
	if connectsvc.BoolValue(flags, "help", false) {
		printSnapshotCommandHelp(stdout, command)
		return 0, true
	}
	if err := validateSnapshotQueryFlags(command, flags); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1, true
	}

	binary, prefix := resolveSnapshotQueryTarget(flags)
	if binary == "" {
		fmt.Fprintln(stderr, "upstream connect/integration binary not found")
		return 1, true
	}
	hours, err := readFeishuLastMessageHours(binary)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1, true
	}

	upstreamCommand := "message-snapshot-senders"
	if command == "search" {
		upstreamCommand = "message-snapshot-search"
	}
	upstreamArgs := append([]string{}, prefix...)
	upstreamArgs = append(upstreamArgs, upstreamCommand, "--source", feishusvc.DefaultName, "--window-hours", strconv.Itoa(hours))
	if command == "search" {
		if value, ok := flags["query"]; ok {
			upstreamArgs = append(upstreamArgs, "--query", value)
		}
		if value := connectsvc.FirstValue(flags, "openid"); value != "" {
			upstreamArgs = append(upstreamArgs, "--sender-id", value)
		}
		if value, ok := flags["limit"]; ok {
			upstreamArgs = append(upstreamArgs, "--limit", value)
		}
		if value, ok := flags["offset"]; ok {
			upstreamArgs = append(upstreamArgs, "--offset", value)
		}
	}
	output, err := runUpstreamQueryCommand(binary, upstreamArgs)
	if err != nil {
		if text := strings.TrimSpace(string(output)); text != "" {
			fmt.Fprintln(stderr, text)
		}
		fmt.Fprintln(stderr, err.Error())
		return 1, true
	}
	if err := writeSnapshotQueryResult(command, output, stdout); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1, true
	}
	return 0, true
}

func writeSnapshotQueryResult(command string, output []byte, stdout io.Writer) error {
	if command == "openid" {
		var upstream []struct {
			SenderID      string `json:"senderId"`
			LastMessageAt string `json:"lastMessageAt"`
		}
		if err := json.Unmarshal(output, &upstream); err != nil {
			return fmt.Errorf("decode Integration message snapshot result: %w", err)
		}
		items := make([]map[string]string, 0, len(upstream))
		for _, item := range upstream {
			items = append(items, map[string]string{"openid": item.SenderID, "lastMessageAt": item.LastMessageAt})
		}
		connectsvc.WriteJSON(stdout, items)
		return nil
	}
	var upstream struct {
		Total  int `json:"total"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
		Items  []struct {
			MessageID string `json:"messageId"`
			SenderID  string `json:"senderId"`
			Content   string `json:"content"`
			SentAt    string `json:"sentAt"`
		} `json:"items"`
	}
	if err := json.Unmarshal(output, &upstream); err != nil {
		return fmt.Errorf("decode Integration message snapshot result: %w", err)
	}
	items := make([]map[string]string, 0, len(upstream.Items))
	for _, item := range upstream.Items {
		items = append(items, map[string]string{
			"messageId": item.MessageID,
			"openid":    item.SenderID,
			"content":   item.Content,
			"sentAt":    item.SentAt,
		})
	}
	connectsvc.WriteJSON(stdout, map[string]any{
		"total":  upstream.Total,
		"limit":  upstream.Limit,
		"offset": upstream.Offset,
		"items":  items,
	})
	return nil
}

func validateSnapshotQueryFlags(command string, flags map[string]string) error {
	if command != "search" {
		return nil
	}
	if _, ok := flags["limit"]; ok {
		limit, err := connectsvc.IntValue(flags, "limit", 50)
		if err != nil {
			return err
		}
		if limit <= 0 {
			return fmt.Errorf("limit must be positive")
		}
		if limit > 200 {
			return fmt.Errorf("limit must not exceed 200")
		}
	}
	if _, ok := flags["offset"]; ok {
		offset, err := connectsvc.IntValue(flags, "offset", 0)
		if err != nil {
			return err
		}
		if offset < 0 {
			return fmt.Errorf("offset must not be negative")
		}
	}
	if raw := connectsvc.FirstValue(flags, "query"); raw != "" {
		if _, err := connectsvc.ParseMessageSnapshotSearchTerms(raw); err != nil {
			return err
		}
	}
	return nil
}

func resolveSnapshotQueryTarget(flags map[string]string) (string, []string) {
	if explicit := strings.TrimSpace(connectsvc.FirstValue(flags, "connect-bin")); explicit != "" {
		binary := normalizeBinaryPath(explicit)
		if binary != "" && !sameExecutable(binary) {
			return binary, prefixForBinary(binary)
		}
		return "", nil
	}
	return resolveProxyTarget()
}

func readFeishuLastMessageHours(connectBinary string) (int, error) {
	configPath, ok := connectsvc.ResolveRuntimeConfigPathFromConnectBin(connectBinary)
	if !ok {
		return 0, fmt.Errorf("integration config/config.json not found for feishu query: %s", connectBinary)
	}
	body, err := os.ReadFile(configPath)
	if err != nil {
		return 0, fmt.Errorf("read config/config.json: %w", err)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return 0, fmt.Errorf("parse config/config.json: %w", err)
	}
	feishuRaw, ok := root["feishu"]
	if !ok {
		return 0, fmt.Errorf("config/config.json feishu.lastMessage is required")
	}
	var feishu map[string]json.RawMessage
	if err := json.Unmarshal(feishuRaw, &feishu); err != nil {
		return 0, fmt.Errorf("config/config.json feishu.lastMessage must be a positive integer")
	}
	lastMessageRaw, ok := feishu["lastMessage"]
	if !ok {
		return 0, fmt.Errorf("config/config.json feishu.lastMessage is required")
	}
	var hours int
	if err := json.Unmarshal(lastMessageRaw, &hours); err != nil || hours <= 0 {
		return 0, fmt.Errorf("config/config.json feishu.lastMessage must be a positive integer")
	}
	return hours, nil
}

func normalizeArgs(args []string, logWriter io.Writer) ([]string, string) {
	if len(args) == 0 {
		return args, ""
	}

	command := strings.TrimSpace(args[0])
	flags, err := connectsvc.ParseFlags(args[1:])
	if err != nil {
		return args, ""
	}

	upstream := strings.TrimSpace(connectsvc.FirstValue(flags, "connect-bin"))
	switch command {
	case "send":
		flags = cloneFlags(flags)
		payload, ok := parseSchemaBackedContent(connectsvc.FirstValue(flags, "content"))
		if ok {
			flags["content"] = payload.Content
			delete(flags, "images")
			delete(flags, "files")
			if merged := mergeCSV(connectsvc.FirstValue(flags, "image", "images"), payload.Images...); len(merged) > 0 {
				flags["image"] = strings.Join(merged, ",")
			} else {
				delete(flags, "image")
			}
			if merged := mergeCSV(connectsvc.FirstValue(flags, "file", "files"), payload.Files...); len(merged) > 0 {
				flags["file"] = strings.Join(merged, ",")
			} else {
				delete(flags, "file")
			}
			writeNormalizationLogs(logWriter, payload)
		}
		ensureSelfProxy(flags)
		return rebuildArgs(command, flags), upstream
	case "init", "start":
		flags = cloneFlags(flags)
		ensureSelfProxy(flags)
		return rebuildArgs(command, flags), upstream
	default:
		return args, upstream
	}
}

func ensureSelfProxy(flags map[string]string) {
	if flags == nil {
		return
	}
	if explicit := strings.TrimSpace(connectsvc.FirstValue(flags, "connect-bin")); explicit != "" {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	if abs, err := filepath.Abs(exe); err == nil {
		exe = abs
	}
	flags["connect-bin"] = exe
}

func setProxyConnectBinaryEnv(value string) func() {
	previous, existed := os.LookupEnv(proxyConnectBinaryEnv)
	value = strings.TrimSpace(value)
	if value == "" {
		return func() {}
	}
	_ = os.Setenv(proxyConnectBinaryEnv, value)
	return func() {
		if existed {
			_ = os.Setenv(proxyConnectBinaryEnv, previous)
			return
		}
		_ = os.Unsetenv(proxyConnectBinaryEnv)
	}
}

func supportedCommands() []string {
	return []string{"command", "help", "name", "param", "scope", "schema", "openid", "search", "init", "send", "start", "stop"}
}

func supportedScopes() []string {
	return []string{"reuse", "agent", "provider", "thinking", "swarm"}
}

func isSendLikeCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch strings.TrimSpace(args[0]) {
	case "send", "init":
		return true
	default:
		return false
	}
}

func runFeishuCLIWithTimeout(args []string, stdout, stderr io.Writer, timeout time.Duration) int {
	if timeout <= 0 {
		return runFeishuCLI(args, stdout, stderr)
	}

	done := make(chan int, 1)
	go func() {
		done <- runFeishuCLI(args, stdout, stderr)
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case code := <-done:
		return code
	case <-timer.C:
		fmt.Fprintf(stderr, "feishu %s timeout after %s\n", firstCommand(args), timeout)
		return 1
	}
}

func firstCommand(args []string) string {
	if len(args) == 0 {
		return "send"
	}
	command := strings.TrimSpace(args[0])
	if command == "" {
		return "send"
	}
	return command
}

const responseSchemaRaw = "{\n" +
	"  \"type\": \"object\",\n" +
	"  \"properties\": {\n" +
	"    \"content\": {\n" +
	"      \"type\": \"string\",\n" +
	"      \"description\": \"The content is presented in MARKDOWN FORMAT text, and the file paths have been replaced with filenames.\"\n" +
	"    },\n" +
	"    \"artifacts\": {\n" +
	"      \"type\": \"array\",\n" +
	"      \"description\": \"All file paths found in the response.\",\n" +
	"      \"items\": {\n" +
	"        \"type\": \"object\",\n" +
	"        \"properties\": {\n" +
	"          \"path\": {\n" +
	"            \"type\": \"string\",\n" +
	"            \"description\": \"Absolute file path.\"\n" +
	"          },\n" +
	"          \"desc\": {\n" +
	"            \"type\": \"string\",\n" +
	"            \"description\": \"Purpose of this file.\"\n" +
	"          }\n" +
	"        },\n" +
	"        \"required\": [\n" +
	"          \"path\",\n" +
	"          \"desc\"\n" +
	"        ]\n" +
	"      }\n" +
	"    },\n" +
	"    \"why_do_this\": {\n" +
	"      \"type\": \"string\",\n" +
	"      \"description\": \"Brief reasoning for executing this command. Must retain granular data and operational logs for process summarization. The more granular, the better.\"\n" +
	"    }\n" +
	"  },\n" +
	"  \"required\": [\n" +
	"    \"content\"\n" +
	"  ]\n" +
	"}"

func responseSchemaJSON() string {
	return strings.TrimSpace(responseSchemaRaw)
}

func responseSchemaDefinition() map[string]any {
	var schema map[string]any
	_ = json.Unmarshal([]byte(responseSchemaJSON()), &schema)
	return schema
}

type normalizedSendPayload struct {
	Content string
	Images  []string
	Files   []string
}

func parseSchemaBackedContent(raw string) (normalizedSendPayload, bool) {
	candidate, ok := schemaJSONCandidateLocal(raw)
	if !ok {
		return normalizedSendPayload{}, false
	}

	var value any
	if err := json.Unmarshal([]byte(candidate), &value); err != nil {
		return normalizedSendPayload{}, false
	}
	if !matchesSchema(value, responseSchemaDefinition()) {
		return normalizedSendPayload{}, false
	}

	object, ok := value.(map[string]any)
	if !ok {
		return normalizedSendPayload{}, false
	}
	content, ok := object["content"].(string)
	if !ok {
		return normalizedSendPayload{}, false
	}

	payload := normalizedSendPayload{Content: content}
	artifacts, _ := object["artifacts"].([]any)
	for _, entry := range artifacts {
		item, ok := entry.(map[string]any)
		if !ok {
			return normalizedSendPayload{}, false
		}
		path, ok := item["path"].(string)
		if !ok || strings.TrimSpace(path) == "" || !filepath.IsAbs(strings.TrimSpace(path)) {
			return normalizedSendPayload{}, false
		}
		path = filepath.Clean(strings.TrimSpace(path))
		if detectArtifactKind(path) == "image" {
			payload.Images = append(payload.Images, path)
			continue
		}
		payload.Files = append(payload.Files, path)
	}
	payload.Images = uniqueStrings(payload.Images)
	payload.Files = uniqueStrings(payload.Files)
	return payload, true
}

func matchesSchema(value any, schema any) bool {
	definition, ok := schema.(map[string]any)
	if !ok {
		return false
	}
	typeName, _ := definition["type"].(string)
	switch strings.TrimSpace(typeName) {
	case "object":
		object, ok := value.(map[string]any)
		if !ok {
			return false
		}
		for _, required := range requiredKeys(definition["required"]) {
			if _, exists := object[required]; !exists {
				return false
			}
		}
		properties, _ := definition["properties"].(map[string]any)
		for key, entry := range object {
			propertySchema, ok := properties[key]
			if !ok {
				continue
			}
			if !matchesSchema(entry, propertySchema) {
				return false
			}
		}
		return true
	case "array":
		items, ok := value.([]any)
		if !ok {
			return false
		}
		for _, item := range items {
			if !matchesSchema(item, definition["items"]) {
				return false
			}
		}
		return true
	case "string":
		_, ok := value.(string)
		return ok
	case "integer":
		number, ok := value.(float64)
		return ok && number == float64(int64(number))
	case "number":
		_, ok := value.(float64)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "":
		return true
	default:
		return false
	}
}

func requiredKeys(value any) []string {
	items, _ := value.([]any)
	keys := make([]string, 0, len(items))
	for _, item := range items {
		key, ok := item.(string)
		if !ok || strings.TrimSpace(key) == "" {
			continue
		}
		keys = append(keys, strings.TrimSpace(key))
	}
	return keys
}

func detectArtifactKind(path string) string {
	if mimeType := strings.TrimSpace(mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))); strings.HasPrefix(mimeType, "image/") {
		return "image"
	}
	file, err := os.Open(path)
	if err != nil {
		return "file"
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "file"
	}
	if strings.HasPrefix(http.DetectContentType(buffer[:n]), "image/") {
		return "image"
	}
	return "file"
}

func mergeCSV(existing string, values ...string) []string {
	merged := append(splitCSV(existing), values...)
	return uniqueStrings(merged)
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		items = append(items, part)
	}
	return items
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func rebuildArgs(command string, flags map[string]string) []string {
	keys := make([]string, 0, len(flags))
	for key := range flags {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	args := []string{command}
	for _, key := range keys {
		args = append(args, "--"+key, flags[key])
	}
	return args
}

func cloneFlags(flags map[string]string) map[string]string {
	cloned := make(map[string]string, len(flags))
	for key, value := range flags {
		cloned[key] = value
	}
	return cloned
}

const proxyConnectBinaryEnv = "FEISHU_PROXY_CONNECT_BIN"

func resolveProxyTarget() (string, []string) {
	if explicit := strings.TrimSpace(os.Getenv(proxyConnectBinaryEnv)); explicit != "" {
		if binary := normalizeBinaryPath(explicit); strings.TrimSpace(binary) != "" && !sameExecutable(binary) {
			return binary, prefixForBinary(binary)
		}
	}
	for _, candidate := range proxyBinaryCandidates() {
		if sameExecutable(candidate) {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return candidate, prefixForBinary(candidate)
		}
	}
	return "", nil
}

func normalizeBinaryPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	if strings.HasPrefix(path, ".") || strings.Contains(path, "/") || strings.Contains(path, string(filepath.Separator)) {
		if abs, err := filepath.Abs(path); err == nil {
			return filepath.Clean(abs)
		}
	}
	return filepath.Clean(path)
}

func prefixForBinary(path string) []string {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(path)))
	if base == "proxy" || base == "integration" || strings.HasPrefix(base, "proxy.") || strings.HasPrefix(base, "integration.") {
		return []string{"connect"}
	}
	return nil
}

func sameExecutable(path string) bool {
	path = normalizeBinaryPath(path)
	if path == "" {
		return false
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	exe = normalizeBinaryPath(exe)
	return exe != "" && exe == path
}

func proxyBinaryCandidates() []string {
	seen := make(map[string]struct{})
	add := func(list *[]string, path string) {
		path = normalizeBinaryPath(path)
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		*list = append(*list, path)
	}

	var candidates []string
	if exe, err := os.Executable(); err == nil && strings.TrimSpace(exe) != "" {
		exeDir := filepath.Dir(exe)
		add(&candidates, filepath.Join(exeDir, "integration"))
		add(&candidates, filepath.Join(exeDir, "proxy"))
		add(&candidates, filepath.Join(exeDir, "connect"))
		add(&candidates, filepath.Join(exeDir, "..", "integration", "integration"))
		add(&candidates, filepath.Join(exeDir, "..", "proxy", "proxy"))
		add(&candidates, filepath.Join(exeDir, "..", "connect", "connect"))
		add(&candidates, filepath.Join(exeDir, "..", "module", "integration", "integration"))
		add(&candidates, filepath.Join(exeDir, "..", "module", "proxy", "proxy"))
		add(&candidates, filepath.Join(exeDir, "..", "module", "connect", "connect"))
	}
	if cwd, err := os.Getwd(); err == nil && strings.TrimSpace(cwd) != "" {
		add(&candidates, filepath.Join(cwd, "integration"))
		add(&candidates, filepath.Join(cwd, "proxy"))
		add(&candidates, filepath.Join(cwd, "connect"))
		add(&candidates, filepath.Join(cwd, "..", "integration", "integration"))
		add(&candidates, filepath.Join(cwd, "..", "proxy", "proxy"))
		add(&candidates, filepath.Join(cwd, "..", "connect", "connect"))
		add(&candidates, filepath.Join(cwd, "..", "..", "integration", "integration"))
		add(&candidates, filepath.Join(cwd, "..", "..", "proxy", "proxy"))
		add(&candidates, filepath.Join(cwd, "..", "..", "connect", "connect"))
	}
	return candidates
}

func writeNormalizationLogs(w io.Writer, payload normalizedSendPayload) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, "[feishu] schema-send content=%s\n", strconv.Quote(payload.Content))
	for _, image := range payload.Images {
		fmt.Fprintf(w, "[feishu] schema-send image=%s\n", strconv.Quote(image))
	}
	for _, file := range payload.Files {
		fmt.Fprintf(w, "[feishu] schema-send file=%s\n", strconv.Quote(file))
	}
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  feishu command")
	fmt.Fprintln(w, "  feishu help")
	fmt.Fprintln(w, "  feishu name")
	fmt.Fprintln(w, "  feishu param")
	fmt.Fprintln(w, "  feishu scope")
	fmt.Fprintln(w, "  feishu schema")
	fmt.Fprintln(w, "  feishu openid [options]")
	fmt.Fprintln(w, "  feishu search [--query TEXT] [--openid OPEN_ID] [options]")
	fmt.Fprintln(w, "  feishu send [options]")
	fmt.Fprintln(w, "  feishu init [options]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Fixed output commands:")
	fmt.Fprintln(w, "  command                         print supported commands")
	fmt.Fprintln(w, "  name                            print stable plugin key/name: {\"key\":\"feishu\",\"name\":\"飞书\"}")
	fmt.Fprintln(w, "  param                           print required meta fields with descriptions for appId/appSecret")
	fmt.Fprintln(w, "  scope                           print supported container scopes: [\"reuse\",\"agent\",\"provider\",\"thinking\",\"swarm\"]")
	fmt.Fprintln(w, "  schema                          print response json schema for schema-backed send content")
	fmt.Fprintln(w, "  openid                          list unique Feishu senders and their last message time in the configured window")
	fmt.Fprintln(w, "  search                          search normalized text messages in the configured window")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Message snapshot queries:")
	fmt.Fprintln(w, "  Both commands query the local Integration/Connect Feishu message snapshot only; they never call Feishu APIs.")
	fmt.Fprintln(w, "  The window comes from integration config/config.json: feishu.lastMessage (positive integer, hours).")
	fmt.Fprintln(w, "  Missing/invalid config causes an immediate error; no default window is used.")
	fmt.Fprintln(w, "  openid output: [{\"openid\":\"ou_xxx\",\"lastMessageAt\":\"2026-07-19T10:30:00Z\"}] (newest first)")
	fmt.Fprintln(w, "  search output: {\"total\":1,\"limit\":50,\"offset\":0,\"items\":[{\"messageId\":\"om_xxx\",\"openid\":\"ou_xxx\",\"content\":\"...\",\"sentAt\":\"2026-07-19T10:30:00Z\"}]}")
	fmt.Fprintln(w, "  search --query TEXT            optional; whitespace terms are AND, quoted text is one phrase, case-insensitive contains")
	fmt.Fprintln(w, "  search --openid OPEN_ID        optional exact sender filter; combines with --query using AND")
	fmt.Fprintln(w, "  search --limit N               result count, default 50, maximum 200")
	fmt.Fprintln(w, "  search --offset N              zero-based result offset, default 0")
	fmt.Fprintln(w, "  --connect-bin PATH             Integration/Connect binary used for the query and config/config.json lookup")
	fmt.Fprintln(w, "  Examples:")
	fmt.Fprintln(w, "    feishu openid --connect-bin ./integration")
	fmt.Fprintln(w, "    feishu search --limit 20 --offset 0 --connect-bin ./integration")
	fmt.Fprintln(w, "    feishu search --query \"退款 已处理\" --limit 20 --offset 0 --connect-bin ./integration")
	fmt.Fprintln(w, "    feishu search --query '\"退款申请\" 已处理' --connect-bin ./integration")
	fmt.Fprintln(w, "    feishu search --openid ou_xxx --limit 20 --connect-bin ./integration")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Message commands:")
	fmt.Fprintln(w, "  send                            reply text/image/file to the original Feishu message")
	fmt.Fprintln(w, "  init                            send the task initialization message with the same arguments as send")
	fmt.Fprintln(w, "  --message JSON                  required, add-request request json with latest rawRequest envelope")
	fmt.Fprintln(w, "  --content TEXT|JSON             text to send, or a schema object from `feishu schema`")
	fmt.Fprintln(w, "  --image PATHS                   comma separated image file paths")
	fmt.Fprintln(w, "  --file PATHS                    comma separated file paths")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Schema-backed send content:")
	fmt.Fprintln(w, "  {\"content\":\"markdown text\",\"artifacts\":[{\"path\":\"/abs/file\",\"desc\":\"purpose\"}],\"why_do_this\":\"reason\"}")
	fmt.Fprintln(w, "  When --content matches `feishu schema`, the plugin will:")
	fmt.Fprintln(w, "    1. send content as the real text body")
	fmt.Fprintln(w, "    2. merge image artifacts into --image")
	fmt.Fprintln(w, "    3. merge file artifacts into --file")
	fmt.Fprintln(w, "    4. write schema-send logs for content/image/file")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Connect proxy behavior:")
	fmt.Fprintln(w, "  meta-get                        forwarded to upstream connect/integration using the same binary contract")
	fmt.Fprintln(w, "  add-request                     forwarded with --schema automatically injected when missing")
}

func printSnapshotCommandHelp(w io.Writer, command string) {
	if command == "openid" {
		fmt.Fprintln(w, "Usage: feishu openid [--connect-bin PATH]")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "List unique openid values from the local Integration/Connect Feishu message snapshot.")
		fmt.Fprintln(w, "The lastMessageAt value is each sender's latest message time in config/config.json feishu.lastMessage hours.")
		fmt.Fprintln(w, "No Feishu API or plugin log is queried. Missing/invalid feishu.lastMessage fails immediately.")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "Example:")
		fmt.Fprintln(w, "  feishu openid --connect-bin ./integration")
		return
	}
	fmt.Fprintln(w, "Usage: feishu search [--query TEXT] [--openid OPEN_ID] [--limit N] [--offset N] [--connect-bin PATH]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Search normalized text messages in the local Integration/Connect Feishu message snapshot.")
	fmt.Fprintln(w, "Terms separated by whitespace are AND; quoted text is one phrase; matching is case-insensitive contains.")
	fmt.Fprintln(w, "Without --query, all text messages in the configured window are listed. --openid is an exact sender filter and combines with --query using AND.")
	fmt.Fprintln(w, "Pure image/file messages are excluded. --limit defaults to 50 and is capped at 200; --offset defaults to 0.")
	fmt.Fprintln(w, "The time window is config/config.json feishu.lastMessage positive integer hours; invalid config fails immediately.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  feishu search --limit 20 --offset 0 --connect-bin ./integration")
	fmt.Fprintln(w, "  feishu search --query \"退款 已处理\" --limit 20 --offset 0 --connect-bin ./integration")
	fmt.Fprintln(w, "  feishu search --query '\"退款申请\" 已处理' --connect-bin ./integration")
	fmt.Fprintln(w, "  feishu search --openid ou_xxx --limit 20 --connect-bin ./integration")
}
