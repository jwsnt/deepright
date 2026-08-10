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

	"connect/connectsvc"
	"connect/emailsvc"
)

var localSupportedCommands = []string{"command", "help", "name", "param", "scope", "schema", "sender", "search", "init", "send", "start", "stop"}
var runEmailCLI = emailsvc.RunCLI
var runLocalSendCommand = runSendCommand
var runEmailSnapshotQueryCommand = func(name string, args []string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%w", err)
	}
	return output, nil
}
var localMetaParams = []localMetaParamDefinition{
	{
		Email:             "邮箱地址，如hello_world@gmail.com",
		EmailPOP3:         "邮箱的pop3地址，如pop.gmail.com",
		EmailSMTP:         "邮箱的smtp地址，如smtp.gmail.com",
		EmailPassword:     "邮箱的密码",
		EmailWhitelist:    "以逗号分隔的收件人白名单，如a@gmail.com,b@gmail.com",
		EmailPOP3Interval: "每次扫描待处理邮件的间隔秒数，默认300",
	},
}

type localMetaParamDefinition struct {
	Email             string `json:"email"`
	EmailPOP3         string `json:"email_pop3"`
	EmailSMTP         string `json:"email_smtp"`
	EmailPassword     string `json:"email_password"`
	EmailWhitelist    string `json:"email_whitelist"`
	EmailPOP3Interval string `json:"email_pop3_interval"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && connectsvc.IsHelpCommand(args[0]) {
		printHelpLocal(stdout)
		return 0
	}
	if code, handled := handleSnapshotQueryCommandLocal(args, stdout, stderr); handled {
		return code
	}
	if handleLocalCommand(args, stdout) {
		return 0
	}
	if result, handled, code := runLocalSendCommand(args, stdout, stderr); handled {
		if result != nil {
			connectsvc.WriteJSON(stdout, result)
		}
		return code
	}
	if handled, code := runLocalLifecycleCommand(args, stdout, stderr); handled {
		return code
	}
	return runEmailCLI(normalizeSendArgs(args, stderr), stdout, stderr)
}

func handleSnapshotQueryCommandLocal(args []string, stdout, stderr io.Writer) (int, bool) {
	if len(args) == 0 {
		return 0, false
	}
	command := strings.TrimSpace(args[0])
	if command != "sender" && command != "search" {
		return 0, false
	}
	if len(args) > 1 && connectsvc.IsHelpCommand(args[1]) {
		printSnapshotCommandHelpLocal(stdout, command)
		return 0, true
	}
	flags, err := connectsvc.ParseFlags(args[1:])
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1, true
	}
	if connectsvc.BoolValue(flags, "help", false) {
		printSnapshotCommandHelpLocal(stdout, command)
		return 0, true
	}
	if err := validateSnapshotQueryFlagsLocal(command, flags); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1, true
	}

	binary := resolveConnectBinaryLocal(flags)
	if strings.TrimSpace(binary) == "" {
		fmt.Fprintln(stderr, "upstream connect/integration binary not found")
		return 1, true
	}
	hours, err := readEmailLastMessageHours(binary)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1, true
	}

	upstreamCommand := "message-snapshot-senders"
	if command == "search" {
		upstreamCommand = "message-snapshot-search"
	}
	upstreamArgs := append([]string{}, resolveConnectPrefixLocal(flags)...)
	upstreamArgs = append(upstreamArgs, upstreamCommand, "--source", emailsvc.DefaultName, "--window-hours", strconv.Itoa(hours))
	if command == "search" {
		if value, ok := flags["query"]; ok {
			upstreamArgs = append(upstreamArgs, "--query", value)
		}
		if value := strings.ToLower(connectsvc.FirstValue(flags, "sender")); value != "" {
			upstreamArgs = append(upstreamArgs, "--sender-id", value)
		}
		if value, ok := flags["limit"]; ok {
			upstreamArgs = append(upstreamArgs, "--limit", value)
		}
		if value, ok := flags["offset"]; ok {
			upstreamArgs = append(upstreamArgs, "--offset", value)
		}
	}
	output, err := runEmailSnapshotQueryCommand(binary, upstreamArgs)
	if err != nil {
		if text := strings.TrimSpace(string(output)); text != "" {
			fmt.Fprintln(stderr, text)
		}
		fmt.Fprintln(stderr, err.Error())
		return 1, true
	}
	if err := writeSnapshotQueryResultLocal(command, output, stdout); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1, true
	}
	return 0, true
}

func validateSnapshotQueryFlagsLocal(command string, flags map[string]string) error {
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
		_, err := connectsvc.ParseMessageSnapshotSearchTerms(raw)
		return err
	}
	return nil
}

func readEmailLastMessageHours(connectBinary string) (int, error) {
	configPath, ok := connectsvc.ResolveRuntimeConfigPathFromConnectBin(connectBinary)
	if !ok {
		return 0, fmt.Errorf("integration config/config.json not found for email query: %s", connectBinary)
	}
	body, err := os.ReadFile(configPath)
	if err != nil {
		return 0, fmt.Errorf("read config/config.json: %w", err)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return 0, fmt.Errorf("parse config/config.json: %w", err)
	}
	emailRaw, ok := root["email"]
	if !ok {
		return 0, fmt.Errorf("config/config.json email.lastMessage is required")
	}
	var email map[string]json.RawMessage
	if err := json.Unmarshal(emailRaw, &email); err != nil {
		return 0, fmt.Errorf("config/config.json email.lastMessage must be a positive integer")
	}
	lastMessageRaw, ok := email["lastMessage"]
	if !ok {
		return 0, fmt.Errorf("config/config.json email.lastMessage is required")
	}
	var hours int
	if err := json.Unmarshal(lastMessageRaw, &hours); err != nil || hours <= 0 {
		return 0, fmt.Errorf("config/config.json email.lastMessage must be a positive integer")
	}
	return hours, nil
}

func writeSnapshotQueryResultLocal(command string, output []byte, stdout io.Writer) error {
	if command == "sender" {
		var upstream []struct {
			SenderID      string `json:"senderId"`
			LastMessageAt string `json:"lastMessageAt"`
		}
		if err := json.Unmarshal(output, &upstream); err != nil {
			return fmt.Errorf("decode Integration message snapshot result: %w", err)
		}
		items := make([]map[string]string, 0, len(upstream))
		for _, item := range upstream {
			items = append(items, map[string]string{"sender": item.SenderID, "lastMessageAt": item.LastMessageAt})
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
			"sender":    item.SenderID,
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

func handleLocalCommand(args []string, stdout io.Writer) bool {
	if len(args) == 0 {
		return false
	}
	switch strings.TrimSpace(args[0]) {
	case "command":
		connectsvc.WriteJSON(stdout, localSupportedCommands)
		return true
	case "help":
		printHelpLocal(stdout)
		return true
	case "param":
		connectsvc.WriteJSON(stdout, localMetaParams)
		return true
	case "scope":
		connectsvc.WriteJSON(stdout, emailsvc.SupportedScopes())
		return true
	case "schema":
		connectsvc.WriteJSON(stdout, responseSchemaDefinitionLocal())
		return true
	default:
		return false
	}
}

type normalizedSendPayload struct {
	Content string
	Images  []string
	Files   []string
}

func normalizeSendArgs(args []string, logWriter io.Writer) []string {
	if len(args) == 0 {
		return args
	}
	command := strings.TrimSpace(args[0])
	if command != "send" && command != "init" {
		return args
	}
	flags, err := connectsvc.ParseFlags(args[1:])
	if err != nil {
		return args
	}
	normalizedFlags, _ := normalizeSendFlagsLocal(command, flags, downloadDirForLogLocal(connectsvc.FirstValue(flags, "log-file")), logWriter)
	return rebuildArgs(command, normalizedFlags)
}

func parseSchemaBackedContent(raw string) (normalizedSendPayload, bool, error) {
	candidate, ok := schemaJSONCandidateLocal(raw)
	if !ok {
		return normalizedSendPayload{}, false, nil
	}

	var value any
	if err := json.Unmarshal([]byte(candidate), &value); err != nil {
		return normalizedSendPayload{}, true, fmt.Errorf("content json parse failed: %w", err)
	}
	if !matchesSchema(value, responseSchemaDefinitionLocal()) {
		return normalizedSendPayload{}, true, fmt.Errorf("content schema mismatch")
	}

	object, ok := value.(map[string]any)
	if !ok {
		return normalizedSendPayload{}, true, fmt.Errorf("content must be a json object")
	}
	content, ok := object["content"].(string)
	if !ok {
		return normalizedSendPayload{}, true, fmt.Errorf("content.content must be a string")
	}

	payload := normalizedSendPayload{Content: content}
	artifacts, _ := object["artifacts"].([]any)
	for _, entry := range artifacts {
		item, ok := entry.(map[string]any)
		if !ok {
			return normalizedSendPayload{}, true, fmt.Errorf("content.artifacts item must be an object")
		}
		path, ok := item["path"].(string)
		if !ok || strings.TrimSpace(path) == "" || !filepath.IsAbs(strings.TrimSpace(path)) {
			return normalizedSendPayload{}, true, fmt.Errorf("content.artifacts.path must be an absolute path")
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
	return payload, true, nil
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
	switch items := value.(type) {
	case []any:
		keys := make([]string, 0, len(items))
		for _, item := range items {
			key, ok := item.(string)
			if !ok || strings.TrimSpace(key) == "" {
				continue
			}
			keys = append(keys, strings.TrimSpace(key))
		}
		return keys
	case []string:
		keys := make([]string, 0, len(items))
		for _, item := range items {
			if strings.TrimSpace(item) == "" {
				continue
			}
			keys = append(keys, strings.TrimSpace(item))
		}
		return keys
	default:
		return nil
	}
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

func writeNormalizationLogs(w io.Writer, payload normalizedSendPayload) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, "[email] schema-send content=%s\n", strconv.Quote(payload.Content))
	for _, image := range payload.Images {
		fmt.Fprintf(w, "[email] schema-send image=%s\n", strconv.Quote(image))
	}
	for _, file := range payload.Files {
		fmt.Fprintf(w, "[email] schema-send file=%s\n", strconv.Quote(file))
	}
}
