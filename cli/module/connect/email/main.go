package main

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"connect/connectsvc"
	"connect/emailsvc"
)

var localSupportedCommands = emailsvc.SupportedCommands()
var runEmailCLI = emailsvc.RunCLI
var runLocalSendCommand = runSendCommand
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
