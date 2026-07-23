package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"connect/connectsvc"
	"connect/feishusvc"
)

func TestFeishuSchemaCommandOutput(t *testing.T) {
	var stdout bytes.Buffer
	if !handleLocalCommand([]string{"schema"}, &stdout) {
		t.Fatal("handleLocalCommand should handle schema")
	}

	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode schema output: %v; raw=%s", err, stdout.String())
	}
	if got["type"] != "object" {
		t.Fatalf("schema type = %#v, want object", got["type"])
	}

	properties, ok := got["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties type = %T", got["properties"])
	}
	for _, key := range []string{"content", "artifacts", "why_do_this"} {
		if _, ok := properties[key]; !ok {
			t.Fatalf("schema missing property %q: %+v", key, properties)
		}
	}

	required, ok := got["required"].([]any)
	if !ok {
		t.Fatalf("required type = %T", got["required"])
	}
	if len(required) != 1 || required[0] != "content" {
		t.Fatalf("required = %#v", required)
	}
}

func TestFeishuSchemaCommandIgnoresOtherCommands(t *testing.T) {
	stdout := &bytes.Buffer{}
	if handleLocalCommand([]string{"start"}, stdout) {
		t.Fatal("handleLocalCommand should ignore non-local commands")
	}
	if stdout.Len() != 0 {
		t.Fatalf("unexpected stdout for non-schema command: %s", stdout.String())
	}
}

func TestFeishuHelpCommandOutput(t *testing.T) {
	var stdout bytes.Buffer
	if !handleLocalCommand([]string{"help"}, &stdout) {
		t.Fatal("handleLocalCommand should handle help")
	}
	text := stdout.String()
	for _, want := range []string{
		"feishu scope",
		"feishu schema",
		"feishu openid [options]",
		"feishu search [--query TEXT] [--openid OPEN_ID] [options]",
		"feishu search --query \"退款 已处理\" --limit 20 --offset 0 --connect-bin ./integration",
		"--content TEXT|JSON",
		"add-request                     forwarded with --schema automatically injected when missing",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("help output missing %q:\n%s", want, text)
		}
	}
	for _, unwanted := range []string{"feishu start [options]", "feishu stop [options]"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("help output should not contain %q:\n%s", unwanted, text)
		}
	}
}

func TestFeishuOpenIDQueriesGenericSnapshotCapability(t *testing.T) {
	root := t.TempDir()
	integration := filepath.Join(root, "integration")
	if err := os.WriteFile(integration, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "config.json"), []byte(`{"feishu":{"lastMessage":72}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	original := runUpstreamQueryCommand
	defer func() { runUpstreamQueryCommand = original }()
	var gotBinary string
	var gotArgs []string
	runUpstreamQueryCommand = func(binary string, args []string) ([]byte, error) {
		gotBinary = binary
		gotArgs = append([]string(nil), args...)
		return []byte(`[{"senderId":"ou_1","lastMessageAt":"2026-07-19T10:30:00Z"}]`), nil
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"openid", "--connect-bin", integration}, &stdout, &stderr); code != 0 {
		t.Fatalf("openid code = %d, stderr = %s", code, stderr.String())
	}
	if gotBinary != integration {
		t.Fatalf("binary = %q, want %q", gotBinary, integration)
	}
	wantArgs := []string{"connect", "message-snapshot-senders", "--source", "feishu", "--window-hours", "72"}
	if !slices.Equal(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
	if !strings.Contains(stdout.String(), `"openid": "ou_1"`) {
		t.Fatalf("openid output = %s", stdout.String())
	}
}

func TestFeishuSearchQueriesGenericSnapshotCapability(t *testing.T) {
	root := t.TempDir()
	integration := filepath.Join(root, "integration")
	if err := os.WriteFile(integration, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "config.json"), []byte(`{"feishu":{"lastMessage":24}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	original := runUpstreamQueryCommand
	defer func() { runUpstreamQueryCommand = original }()
	var gotArgs []string
	runUpstreamQueryCommand = func(_ string, args []string) ([]byte, error) {
		gotArgs = append([]string(nil), args...)
		return []byte(`{"total":1,"limit":20,"offset":5,"items":[{"messageId":"om_1","senderId":"ou_1","content":"退款已处理","sentAt":"2026-07-19T10:30:00Z"}]}`), nil
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"search", "--query", `"退款申请" 已处理`, "--limit", "20", "--offset", "5", "--connect-bin", integration}, &stdout, &stderr); code != 0 {
		t.Fatalf("search code = %d, stderr = %s", code, stderr.String())
	}
	wantArgs := []string{"connect", "message-snapshot-search", "--source", "feishu", "--window-hours", "24", "--query", `"退款申请" 已处理`, "--limit", "20", "--offset", "5"}
	if !slices.Equal(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
	if !strings.Contains(stdout.String(), `"openid": "ou_1"`) || strings.Contains(stdout.String(), "senderId") {
		t.Fatalf("search output = %s", stdout.String())
	}
}

func TestFeishuSearchAllowsOmittedQueryAndFiltersOpenID(t *testing.T) {
	root := t.TempDir()
	integration := filepath.Join(root, "integration")
	if err := os.WriteFile(integration, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "config.json"), []byte(`{"feishu":{"lastMessage":24}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	original := runUpstreamQueryCommand
	defer func() { runUpstreamQueryCommand = original }()
	var gotArgs []string
	runUpstreamQueryCommand = func(_ string, args []string) ([]byte, error) {
		gotArgs = append([]string(nil), args...)
		return []byte(`{"total":0,"limit":20,"offset":0,"items":[]}`), nil
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"search", "--openid", "ou_1", "--limit", "20", "--connect-bin", integration}, &stdout, &stderr); code != 0 {
		t.Fatalf("search code = %d, stderr = %s", code, stderr.String())
	}
	wantArgs := []string{"connect", "message-snapshot-search", "--source", "feishu", "--window-hours", "24", "--sender-id", "ou_1", "--limit", "20"}
	if !slices.Equal(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
}

func TestFeishuOpenIDRejectsInvalidLastMessageConfig(t *testing.T) {
	root := t.TempDir()
	integration := filepath.Join(root, "integration")
	if err := os.WriteFile(integration, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "config.json"), []byte(`{"feishu":{"lastMessage":0}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"openid", "--connect-bin", integration}, &stdout, &stderr); code == 0 {
		t.Fatal("openid unexpectedly succeeded")
	}
	if !strings.Contains(stderr.String(), "feishu.lastMessage must be a positive integer") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestFeishuNameCommandOutput(t *testing.T) {
	var stdout bytes.Buffer
	if !handleLocalCommand([]string{"name"}, &stdout) {
		t.Fatal("handleLocalCommand should handle name")
	}

	var got struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode name output: %v; raw=%s", err, stdout.String())
	}
	if got.Key != "feishu" || got.Name != "飞书" {
		t.Fatalf("unexpected name output: %+v", got)
	}
}

func TestFeishuParamCommandOutput(t *testing.T) {
	var stdout bytes.Buffer
	if !handleLocalCommand([]string{"param"}, &stdout) {
		t.Fatal("handleLocalCommand should handle param")
	}

	var got []map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode param output: %v; raw=%s", err, stdout.String())
	}
	if len(got) != 1 {
		t.Fatalf("param len = %d, want 1; got=%v", len(got), got)
	}
	want := map[string]string{
		"appId":     "飞书开放平台（https://open.feishu.cn/app）中应用凭证的App ID ",
		"appSecret": "App Secret",
		"mcp_url":   "飞书MCP地址",
	}
	if len(got[0]) != len(want) {
		t.Fatalf("param field len = %d, want %d; got=%v", len(got[0]), len(want), got[0])
	}
	for key, value := range want {
		if got[0][key] != value {
			t.Fatalf("param[%q] = %q, want %q; all=%v", key, got[0][key], value, got[0])
		}
	}
}

func TestFeishuScopeCommandOutput(t *testing.T) {
	var stdout bytes.Buffer
	if !handleLocalCommand([]string{"scope"}, &stdout) {
		t.Fatal("handleLocalCommand should handle scope")
	}

	var got []string
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode scope output: %v; raw=%s", err, stdout.String())
	}
	want := []string{"reuse", "agent", "provider", "thinking", "swarm"}
	if len(got) != len(want) {
		t.Fatalf("scope len = %d, want %d; got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("scope[%d] = %q, want %q; all=%v", i, got[i], want[i], got)
		}
	}
}

func TestFeishuCommandIncludesSchema(t *testing.T) {
	var stdout bytes.Buffer
	if !handleLocalCommand([]string{"command"}, &stdout) {
		t.Fatal("handleLocalCommand should handle command")
	}

	var commands []string
	if err := json.Unmarshal(stdout.Bytes(), &commands); err != nil {
		t.Fatalf("decode command output: %v; raw=%s", err, stdout.String())
	}
	if len(commands) != len(localSupportedCommands) {
		t.Fatalf("command len = %d, want %d", len(commands), len(localSupportedCommands))
	}
	for i, want := range localSupportedCommands {
		if commands[i] != want {
			t.Fatalf("command[%d] = %q, want %q; all=%v", i, commands[i], want, commands)
		}
	}
}

func TestRunDispatchesSchemaCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{"schema"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run schema exit code = %d, stderr = %s", code, stderr.String())
	}

	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode run schema output: %v; raw=%s", err, stdout.String())
	}
	if got["type"] != "object" {
		t.Fatalf("schema type = %#v, want object", got["type"])
	}
	properties, _ := got["properties"].(map[string]any)
	content, _ := properties["content"].(map[string]any)
	if content["description"] != "The content is presented in MARKDOWN FORMAT text, and the file paths have been replaced with filenames." {
		t.Fatalf("content description = %#v", content["description"])
	}
}

func TestNormalizeSendArgsTransformsSchemaContent(t *testing.T) {
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "a.png")
	filePath := filepath.Join(tempDir, "b.txt")
	if err := os.WriteFile(imagePath, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("plain"), 0o644); err != nil {
		t.Fatal(err)
	}

	rawContent := `{"content":"飞书正文","artifacts":[{"path":"` + imagePath + `","desc":"截图"},{"path":"` + filePath + `","desc":"说明"}],"why_do_this":"记录过程"}`
	var logs bytes.Buffer
	args, _ := normalizeArgs([]string{
		"send",
		"--message", `{"id":1}`,
		"--content", rawContent,
	}, &logs)

	got, err := parseFlagMap(args[1:])
	if err != nil {
		t.Fatal(err)
	}
	if got["content"] != "飞书正文" {
		t.Fatalf("content = %q", got["content"])
	}
	if got["image"] != imagePath {
		t.Fatalf("image = %q", got["image"])
	}
	if got["file"] != filePath {
		t.Fatalf("file = %q", got["file"])
	}
	if strings.TrimSpace(got["connect-bin"]) == "" {
		t.Fatal("connect-bin should be injected")
	}
	if !strings.Contains(logs.String(), `content="飞书正文"`) {
		t.Fatalf("logs missing content marker: %s", logs.String())
	}
	if !strings.Contains(logs.String(), `image="`+imagePath+`"`) {
		t.Fatalf("logs missing image marker: %s", logs.String())
	}
	if !strings.Contains(logs.String(), `file="`+filePath+`"`) {
		t.Fatalf("logs missing file marker: %s", logs.String())
	}
}

func TestNormalizeSendArgsExtractsSchemaFromPrefixedText(t *testing.T) {
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "a.png")
	filePath := filepath.Join(tempDir, "b.txt")
	if err := os.WriteFile(imagePath, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("plain"), 0o644); err != nil {
		t.Fatal(err)
	}

	rawContent := `先看看桌面文件情况，同时尝试截图。截图已生成。现在截图下载目录：{"content":"飞书正文","artifacts":[{"path":"` + imagePath + `","desc":"截图"},{"path":"` + filePath + `","desc":"说明"}],"why_do_this":"记录过程"}`
	var logs bytes.Buffer
	args, _ := normalizeArgs([]string{
		"send",
		"--message", `{"id":1}`,
		"--content", rawContent,
	}, &logs)

	got, err := parseFlagMap(args[1:])
	if err != nil {
		t.Fatal(err)
	}
	if got["content"] != "飞书正文" {
		t.Fatalf("content = %q", got["content"])
	}
	if got["image"] != imagePath {
		t.Fatalf("image = %q", got["image"])
	}
	if got["file"] != filePath {
		t.Fatalf("file = %q", got["file"])
	}
	if !strings.Contains(logs.String(), `content="飞书正文"`) {
		t.Fatalf("logs missing content marker: %s", logs.String())
	}
}

func TestNormalizeSendArgsAcceptsContentOnlySchema(t *testing.T) {
	rawContent := `{"content":"only content"}`
	args, _ := normalizeArgs([]string{
		"send",
		"--content", rawContent,
		"--image", "/tmp/already.png",
	}, io.Discard)

	got, err := parseFlagMap(args[1:])
	if err != nil {
		t.Fatal(err)
	}
	if got["content"] != "only content" {
		t.Fatalf("content = %q, want normalized text", got["content"])
	}
	if got["image"] != "/tmp/already.png" {
		t.Fatalf("image = %q", got["image"])
	}
	if strings.TrimSpace(got["connect-bin"]) == "" {
		t.Fatal("connect-bin should be injected")
	}
}

func TestNormalizeSendArgsKeepsOriginalWhenPrefixedTextHasNoExtractableJSON(t *testing.T) {
	rawContent := `先看看桌面文件情况，同时尝试截图。截图已生成。现在截图下载目录：{"content":}`
	args, _ := normalizeArgs([]string{
		"send",
		"--content", rawContent,
		"--image", "/tmp/already.png",
	}, io.Discard)

	got, err := parseFlagMap(args[1:])
	if err != nil {
		t.Fatal(err)
	}
	if got["content"] != rawContent {
		t.Fatalf("content = %q, want original text", got["content"])
	}
	if got["image"] != "/tmp/already.png" {
		t.Fatalf("image = %q", got["image"])
	}
	if strings.TrimSpace(got["connect-bin"]) == "" {
		t.Fatal("connect-bin should be injected")
	}
}

func TestNormalizeSendArgsMergesExistingArtifacts(t *testing.T) {
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "a.jpg")
	filePath := filepath.Join(tempDir, "b.pdf")
	if err := os.WriteFile(imagePath, []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("%PDF"), 0o644); err != nil {
		t.Fatal(err)
	}

	rawContent := `{"content":"飞书正文","artifacts":[{"path":"` + imagePath + `","desc":"截图"},{"path":"` + filePath + `","desc":"文档"}],"why_do_this":"记录过程"}`
	args, _ := normalizeArgs([]string{
		"send",
		"--content", rawContent,
		"--image", "/tmp/existing.png",
		"--file", "/tmp/existing.txt",
	}, io.Discard)

	got, err := parseFlagMap(args[1:])
	if err != nil {
		t.Fatal(err)
	}
	if got["image"] != "/tmp/existing.png,"+imagePath {
		t.Fatalf("image = %q", got["image"])
	}
	if got["file"] != "/tmp/existing.txt,"+filePath {
		t.Fatalf("file = %q", got["file"])
	}
}

func TestNormalizeStartInjectsConnectBin(t *testing.T) {
	args, _ := normalizeArgs([]string{"start", "--name", "feishu"}, io.Discard)
	got, err := parseFlagMap(args[1:])
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got["connect-bin"]) == "" {
		t.Fatal("connect-bin should be injected for start")
	}
}

func TestNormalizeInitInjectsConnectBin(t *testing.T) {
	args, _ := normalizeArgs([]string{"init", "--message", `{"id":1}`, "--content", "hello"}, io.Discard)
	got, err := parseFlagMap(args[1:])
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got["connect-bin"]) == "" {
		t.Fatal("connect-bin should be injected for init")
	}
}

func TestRunDispatchesSchemaAwareSend(t *testing.T) {
	originalRunner := runFeishuCLI
	defer func() { runFeishuCLI = originalRunner }()

	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "a.png")
	if err := os.WriteFile(imagePath, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}

	var captured []string
	runFeishuCLI = func(args []string, stdout, stderr io.Writer) int {
		captured = append([]string{}, args...)
		_, _ = stdout.Write([]byte("{}\n"))
		return 0
	}

	rawContent := `{"content":"飞书正文","artifacts":[{"path":"` + imagePath + `","desc":"截图"}],"why_do_this":"记录过程"}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{"send", "--content", rawContent}, &stdout, &stderr); code != 0 {
		t.Fatalf("run send exit code = %d, stderr=%s", code, stderr.String())
	}

	got, err := parseFlagMap(captured[1:])
	if err != nil {
		t.Fatal(err)
	}
	if captured[0] != "send" {
		t.Fatalf("command = %q", captured[0])
	}
	if got["content"] != "飞书正文" {
		t.Fatalf("content = %q", got["content"])
	}
	if got["image"] != imagePath {
		t.Fatalf("image = %q", got["image"])
	}
	if strings.TrimSpace(got["connect-bin"]) == "" {
		t.Fatal("connect-bin should be injected")
	}
}

func TestRunSendUsesTimeoutWrapper(t *testing.T) {
	originalRunner := runFeishuCLI
	originalTimeout := feishuSendTimeout
	defer func() {
		runFeishuCLI = originalRunner
		feishuSendTimeout = originalTimeout
	}()

	feishuSendTimeout = 50 * time.Millisecond
	started := make(chan struct{}, 1)
	runFeishuCLI = func(args []string, stdout, stderr io.Writer) int {
		started <- struct{}{}
		time.Sleep(200 * time.Millisecond)
		return 0
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"send", "--content", "hello"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("send should time out")
	}
	select {
	case <-started:
	default:
		t.Fatal("send runner should be invoked")
	}
	if !strings.Contains(stderr.String(), "feishu send timeout after 50ms") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunServeUsesLocalLifecycleInsteadOfRunner(t *testing.T) {
	originalRunner := runFeishuCLI
	defer func() { runFeishuCLI = originalRunner }()
	runFeishuCLI = func(args []string, stdout, stderr io.Writer) int {
		t.Fatal("local lifecycle command should not call shared feishu runner")
		return 1
	}

	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "feishu.pid")
	logFile := filepath.Join(tempDir, "feishu.log")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	originalSignalContext := signalContextLocalFeishuFn
	defer func() { signalContextLocalFeishuFn = originalSignalContext }()
	signalContextLocalFeishuFn = func() (context.Context, context.CancelFunc) { return ctx, cancel }

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"serve", "--pid-file", pidFile, "--log-file", logFile}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("serve exit code = %d, stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(logFile); err != nil {
		t.Fatalf("expected local lifecycle log file: %v", err)
	}
}

func TestRunInitBypassesLocalLifecycleAndUsesRunner(t *testing.T) {
	originalRunner := runFeishuCLI
	originalTimeout := feishuSendTimeout
	defer func() {
		runFeishuCLI = originalRunner
		feishuSendTimeout = originalTimeout
	}()

	feishuSendTimeout = time.Second
	runFeishuCLI = func(args []string, stdout, stderr io.Writer) int {
		_, _ = stdout.Write([]byte("ok"))
		return 0
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"init", "--message", `{"id":1}`, "--content", "hello"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("init exit code = %d, stderr=%s", code, stderr.String())
	}
	if stdout.String() != "ok" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestHandleProxyAddRequestInjectsSchema(t *testing.T) {
	original := runUpstreamCommand
	defer func() { runUpstreamCommand = original }()

	var calledName string
	var calledArgs []string
	runUpstreamCommand = func(name string, args []string, stdout, stderr io.Writer) error {
		calledName = name
		calledArgs = append([]string{}, args...)
		return nil
	}

	tempDir := t.TempDir()
	upstream := filepath.Join(tempDir, "integration")
	if err := os.WriteFile(upstream, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	restore := withEnv(proxyConnectBinaryEnv, upstream)
	defer restore()

	code, handled := handleProxyCommand([]string{"add-request", "--key", "feishu", "--content", "hello"}, io.Discard, io.Discard)
	if !handled {
		t.Fatal("proxy add-request should be handled")
	}
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if calledName != upstream {
		t.Fatalf("calledName = %q, want %q", calledName, upstream)
	}
	if len(calledArgs) < 2 {
		t.Fatalf("calledArgs too short: %+v", calledArgs)
	}
	got, err := parseFlagMap(calledArgs[2:])
	if err != nil {
		t.Fatal(err)
	}
	if calledArgs[0] != "connect" || calledArgs[1] != "add-request" {
		t.Fatalf("unexpected proxy args: %+v", calledArgs)
	}
	if got["schema"] != responseSchemaJSON() {
		t.Fatalf("schema = %q", got["schema"])
	}
}

func TestHandleProxyAddRequestPreservesExplicitSchema(t *testing.T) {
	original := runUpstreamCommand
	defer func() { runUpstreamCommand = original }()

	var calledArgs []string
	runUpstreamCommand = func(name string, args []string, stdout, stderr io.Writer) error {
		calledArgs = append([]string{}, args...)
		return nil
	}

	tempDir := t.TempDir()
	upstream := filepath.Join(tempDir, "connect")
	if err := os.WriteFile(upstream, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	restore := withEnv(proxyConnectBinaryEnv, upstream)
	defer restore()

	wantSchema := `{"type":"object","properties":{"x":{"type":"string"}}}`
	code, handled := handleProxyCommand([]string{"add-request", "--key", "feishu", "--content", "hello", "--schema", wantSchema}, io.Discard, io.Discard)
	if !handled {
		t.Fatal("proxy add-request should be handled")
	}
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if len(calledArgs) < 1 || calledArgs[0] != "add-request" {
		t.Fatalf("unexpected proxy args: %+v", calledArgs)
	}
	got, err := parseFlagMap(calledArgs[1:])
	if err != nil {
		t.Fatal(err)
	}
	if got["schema"] != wantSchema {
		t.Fatalf("schema = %q, want %q", got["schema"], wantSchema)
	}
}

func TestHandleProxyMetaGetPassThrough(t *testing.T) {
	original := runUpstreamCommand
	defer func() { runUpstreamCommand = original }()

	var calledArgs []string
	runUpstreamCommand = func(name string, args []string, stdout, stderr io.Writer) error {
		calledArgs = append([]string{}, args...)
		return nil
	}

	tempDir := t.TempDir()
	upstream := filepath.Join(tempDir, "connect")
	if err := os.WriteFile(upstream, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	restore := withEnv(proxyConnectBinaryEnv, upstream)
	defer restore()

	code, handled := handleProxyCommand([]string{"meta-get", "--key", "feishu"}, io.Discard, io.Discard)
	if !handled {
		t.Fatal("proxy meta-get should be handled")
	}
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if len(calledArgs) < 1 || calledArgs[0] != "meta-get" {
		t.Fatalf("unexpected proxy args: %+v", calledArgs)
	}
	got, err := parseFlagMap(calledArgs[1:])
	if err != nil {
		t.Fatal(err)
	}
	if got["key"] != "feishu" {
		t.Fatalf("key = %q", got["key"])
	}
	if _, ok := got["schema"]; ok {
		t.Fatalf("meta-get should not inject schema: %+v", got)
	}
}

func TestNormalizeArgsInjectsProxyEnvForSendRunner(t *testing.T) {
	originalRunner := runFeishuCLI
	defer func() { runFeishuCLI = originalRunner }()

	tempDir := t.TempDir()
	upstream := filepath.Join(tempDir, "integration")
	var seenEnv string
	runFeishuCLI = func(args []string, stdout, stderr io.Writer) int {
		seenEnv = os.Getenv(proxyConnectBinaryEnv)
		_, _ = stdout.Write([]byte("{}\n"))
		return 0
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{"send", "--connect-bin", upstream, "--content", "hello"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run send exit code = %d, stderr=%s", code, stderr.String())
	}
	if seenEnv != upstream {
		t.Fatalf("proxy env = %q, want %q", seenEnv, upstream)
	}
	if got := os.Getenv(proxyConnectBinaryEnv); got != "" {
		t.Fatalf("proxy env should be restored, got %q", got)
	}
}

func TestLocalFeishuFlushPendingPushesTextWithPriorArtifacts(t *testing.T) {
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "img.png")
	if err := os.WriteFile(imagePath, []byte{0x89, 'P', 'N', 'G'}, 0o644); err != nil {
		t.Fatal(err)
	}

	client := &stubLocalFeishuConnectClient{}
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	svc := &localFeishuService{
		name:         "feishu",
		client:       client,
		logger:       log.New(io.Discard, "", 0),
		messageLog:   io.Discard,
		downloadDir:  tempDir,
		stateFile:    filepath.Join(tempDir, "feishu.pending.json"),
		scanInterval: localFeishuScanInterval,
		expireWindow: localFeishuExpireWindow,
		nowFn:        func() time.Time { return now },
		state: localFeishuState{
			Pending: map[string]localFeishuPendingMessage{
				"image-1": {
					ExternalID: "image-1",
					MessageID:  "om-image-1",
					ChatID:     "chat-1",
					Type:       "image",
					Artifacts:  []string{imagePath},
					MessageAt:  now.Add(-20 * time.Second).UnixMilli(),
				},
				"text-1": {
					ExternalID: "text-1",
					MessageID:  "om-text-1",
					ChatID:     "chat-1",
					Type:       "text",
					Content:    "文本消息",
					Raw:        `{"schema":"2.0","event":{"message":{"message_id":"om-text-1"}}}`,
					MessageAt:  now.Add(-10 * time.Second).UnixMilli(),
				},
			},
			Pushed: map[string]int64{},
		},
	}

	if err := svc.flushPending(context.Background(), "feishu"); err != nil {
		t.Fatal(err)
	}
	requests := client.Requests()
	if len(requests) != 1 {
		t.Fatalf("requests len = %d, want 1", len(requests))
	}
	got := requests[0]
	if got.Name != "feishu" {
		t.Fatalf("name = %q", got.Name)
	}
	if got.Content != "文本消息 [image]"+imagePath {
		t.Fatalf("content = %q", got.Content)
	}
	if got.Artifacts != imagePath {
		t.Fatalf("artifacts = %q", got.Artifacts)
	}
	if strings.TrimSpace(got.ResponseSchema) == "" {
		t.Fatal("response schema should not be empty")
	}
	if len(svc.state.Pending) != 0 {
		t.Fatalf("pending should be empty after push: %+v", svc.state.Pending)
	}
}

func TestBuildMessageSnapshotLocalFeishuUsesNormalizedSourceMessages(t *testing.T) {
	now := time.Date(2026, time.July, 19, 10, 0, 0, 0, time.UTC)
	raw := buildMessageSnapshotLocalFeishu([]localFeishuPendingMessage{
		{
			ExternalID: "ext-text",
			MessageID:  "om-text",
			SenderID:   "ou_text",
			SenderType: "user",
			Type:       "text",
			Content:    "归一化文本",
			MessageAt:  now.UnixMilli(),
		},
		{
			ExternalID: "ext-image",
			MessageID:  "om-image",
			SenderID:   "ou_image",
			SenderType: "user",
			Type:       "image",
			Content:    "[image]/tmp/a.png",
			MessageAt:  now.Add(time.Minute).UnixMilli(),
		},
	}, feishusvc.IncomingMessage{})
	if raw == "" {
		t.Fatal("snapshot is empty")
	}
	var snapshot connectsvc.MessageSnapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Source != "feishu" || len(snapshot.Messages) != 2 {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	if snapshot.Messages[0].MessageID != "om-text" || snapshot.Messages[0].SenderID != "ou_text" || snapshot.Messages[0].MessageType != "text" {
		t.Fatalf("text snapshot = %+v", snapshot.Messages[0])
	}
	if snapshot.Messages[1].MessageID != "om-image" || snapshot.Messages[1].MessageType != "image" {
		t.Fatalf("image snapshot = %+v", snapshot.Messages[1])
	}
}

func TestLocalFeishuFlushPendingPushesPureTextMessage(t *testing.T) {
	tempDir := t.TempDir()
	client := &stubLocalFeishuConnectClient{}
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	svc := &localFeishuService{
		name:         "feishu",
		client:       client,
		logger:       log.New(io.Discard, "", 0),
		messageLog:   io.Discard,
		downloadDir:  tempDir,
		stateFile:    filepath.Join(tempDir, "feishu.pending.json"),
		scanInterval: localFeishuScanInterval,
		expireWindow: localFeishuExpireWindow,
		nowFn:        func() time.Time { return now },
		state: localFeishuState{
			Pending: map[string]localFeishuPendingMessage{
				"text-1": {
					ExternalID: "text-1",
					MessageID:  "om-text-1",
					ChatID:     "chat-1",
					Type:       "text",
					Content:    "纯文本消息",
					Raw:        `{"schema":"2.0","event":{"message":{"message_id":"om-text-1"}}}`,
					MessageAt:  now.Add(-10 * time.Second).UnixMilli(),
				},
			},
			Pushed: map[string]int64{},
		},
	}

	if err := svc.flushPending(context.Background(), "feishu"); err != nil {
		t.Fatal(err)
	}
	requests := client.Requests()
	if len(requests) != 1 {
		t.Fatalf("requests len = %d, want 1", len(requests))
	}
	got := requests[0]
	if got.Content != "纯文本消息" {
		t.Fatalf("content = %q", got.Content)
	}
	if got.Artifacts != "" {
		t.Fatalf("artifacts = %q, want empty", got.Artifacts)
	}
	if len(svc.state.Pending) != 0 {
		t.Fatalf("pending should be empty after push: %+v", svc.state.Pending)
	}
}

func TestLocalFeishuFlushPendingAcknowledgesImmediatelyAfterPersisting(t *testing.T) {
	tempDir := t.TempDir()
	client := &stubLocalFeishuConnectClient{}
	acknowledged := make([]connectsvc.Request, 0, 1)
	acknowledgementContent := ""
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	svc := &localFeishuService{
		name:         "feishu",
		client:       client,
		logger:       log.New(io.Discard, "", 0),
		messageLog:   io.Discard,
		downloadDir:  tempDir,
		stateFile:    filepath.Join(tempDir, "feishu.pending.json"),
		scanInterval: localFeishuScanInterval,
		expireWindow: localFeishuExpireWindow,
		nowFn:        func() time.Time { return now },
		acknowledger: localFeishuAcknowledgementSenderFunc(func(_ context.Context, request connectsvc.Request, content string) error {
			acknowledged = append(acknowledged, request)
			acknowledgementContent = content
			return nil
		}),
		state: localFeishuState{
			Pending: map[string]localFeishuPendingMessage{
				"text-1": {
					ExternalID: "text-1",
					MessageID:  "om-text-1",
					ChatID:     "chat-1",
					Type:       "text",
					Content:    "请处理这条消息",
					Raw:        `{"schema":"2.0","event":{"message":{"message_id":"om-text-1"}}}`,
					MessageAt:  now.UnixMilli(),
				},
			},
			Pushed: map[string]int64{},
		},
	}

	if err := svc.flushPending(context.Background(), "feishu"); err != nil {
		t.Fatal(err)
	}
	if len(client.Requests()) != 1 {
		t.Fatalf("add-request calls = %d, want 1", len(client.Requests()))
	}
	if len(acknowledged) != 1 {
		t.Fatalf("acknowledgements = %d, want 1", len(acknowledged))
	}
	if acknowledgementContent != localFeishuReceivedReply {
		t.Fatalf("acknowledgement content = %q, want %q", acknowledgementContent, localFeishuReceivedReply)
	}
	if !strings.Contains(acknowledged[0].RawRequest, `"messageId":"om-text-1"`) {
		t.Fatalf("acknowledgement did not retain original message target: %s", acknowledged[0].RawRequest)
	}
	if len(svc.state.Acknowledging) != 0 {
		t.Fatalf("pending acknowledgements = %+v, want empty", svc.state.Acknowledging)
	}
	if _, ok := svc.state.Acknowledged["text-1"]; !ok {
		t.Fatalf("missing persisted acknowledgement state: %+v", svc.state.Acknowledged)
	}
}

func TestLocalFeishuFlushPendingRetriesAcknowledgementWithoutRepersistingRequest(t *testing.T) {
	tempDir := t.TempDir()
	client := &stubLocalFeishuConnectClient{}
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	ackAttempts := 0
	svc := &localFeishuService{
		name:         "feishu",
		client:       client,
		logger:       log.New(io.Discard, "", 0),
		messageLog:   io.Discard,
		downloadDir:  tempDir,
		stateFile:    filepath.Join(tempDir, "feishu.pending.json"),
		scanInterval: localFeishuScanInterval,
		expireWindow: localFeishuExpireWindow,
		nowFn:        func() time.Time { return now },
		acknowledger: localFeishuAcknowledgementSenderFunc(func(_ context.Context, _ connectsvc.Request, _ string) error {
			ackAttempts++
			if ackAttempts == 1 {
				return errors.New("feishu unavailable")
			}
			return nil
		}),
		state: localFeishuState{
			Pending: map[string]localFeishuPendingMessage{
				"text-1": {
					ExternalID: "text-1",
					MessageID:  "om-text-1",
					ChatID:     "chat-1",
					Type:       "text",
					Content:    "请处理这条消息",
					Raw:        `{"schema":"2.0","event":{"message":{"message_id":"om-text-1"}}}`,
					MessageAt:  now.UnixMilli(),
				},
			},
			Pushed: map[string]int64{},
		},
	}

	if err := svc.flushPending(context.Background(), "feishu"); err != nil {
		t.Fatal(err)
	}
	if len(client.Requests()) != 1 {
		t.Fatalf("add-request calls after failed acknowledgement = %d, want 1", len(client.Requests()))
	}
	if len(svc.state.Acknowledging) != 1 {
		t.Fatalf("pending acknowledgements = %+v, want one", svc.state.Acknowledging)
	}

	if err := svc.flushPending(context.Background(), "feishu"); err != nil {
		t.Fatal(err)
	}
	if len(client.Requests()) != 1 {
		t.Fatalf("add-request calls after acknowledgement retry = %d, want 1", len(client.Requests()))
	}
	if ackAttempts != 2 {
		t.Fatalf("acknowledgement attempts = %d, want 2", ackAttempts)
	}
	if len(svc.state.Acknowledging) != 0 {
		t.Fatalf("pending acknowledgements = %+v, want empty", svc.state.Acknowledging)
	}
}

func TestLocalFeishuFlushPendingDoesNotAcknowledgeWhenPersistingFails(t *testing.T) {
	tempDir := t.TempDir()
	client := &stubLocalFeishuConnectClient{addErr: errors.New("connect unavailable")}
	acknowledged := false
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	svc := &localFeishuService{
		name:         "feishu",
		client:       client,
		logger:       log.New(io.Discard, "", 0),
		messageLog:   io.Discard,
		downloadDir:  tempDir,
		stateFile:    filepath.Join(tempDir, "feishu.pending.json"),
		scanInterval: localFeishuScanInterval,
		expireWindow: localFeishuExpireWindow,
		nowFn:        func() time.Time { return now },
		acknowledger: localFeishuAcknowledgementSenderFunc(func(_ context.Context, _ connectsvc.Request, _ string) error {
			acknowledged = true
			return nil
		}),
		state: localFeishuState{
			Pending: map[string]localFeishuPendingMessage{
				"text-1": {ExternalID: "text-1", MessageID: "om-text-1", ChatID: "chat-1", Type: "text", Content: "请处理这条消息", MessageAt: now.UnixMilli()},
			},
			Pushed: map[string]int64{},
		},
	}

	if err := svc.flushPending(context.Background(), "feishu"); err != nil {
		t.Fatal(err)
	}
	if acknowledged {
		t.Fatal("acknowledgement should not be sent when add-request fails")
	}
	if len(svc.state.Acknowledging) != 0 {
		t.Fatalf("pending acknowledgements = %+v, want empty", svc.state.Acknowledging)
	}
}

func TestLocalFeishuFlushPendingWaitsForText(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "a.txt")
	if err := os.WriteFile(filePath, []byte("demo"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := &stubLocalFeishuConnectClient{}
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	svc := &localFeishuService{
		name:         "feishu",
		client:       client,
		logger:       log.New(io.Discard, "", 0),
		messageLog:   io.Discard,
		downloadDir:  tempDir,
		stateFile:    filepath.Join(tempDir, "feishu.pending.json"),
		scanInterval: localFeishuScanInterval,
		expireWindow: localFeishuExpireWindow,
		nowFn:        func() time.Time { return now },
		state: localFeishuState{
			Pending: map[string]localFeishuPendingMessage{
				"file-1": {
					ExternalID: "file-1",
					MessageID:  "om-file-1",
					ChatID:     "chat-1",
					Type:       "file",
					Artifacts:  []string{filePath},
					MessageAt:  now.Add(-20 * time.Second).UnixMilli(),
				},
			},
			Pushed: map[string]int64{},
		},
	}

	if err := svc.flushPending(context.Background(), "feishu"); err != nil {
		t.Fatal(err)
	}
	if len(client.Requests()) != 0 {
		t.Fatalf("file-only messages should wait, got %+v", client.Requests())
	}
	if len(svc.state.Pending) != 1 {
		t.Fatalf("pending should remain: %+v", svc.state.Pending)
	}
}

func TestLocalFeishuFlushPendingWaitsForOnlyArtifactsAcrossMessages(t *testing.T) {
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "img.png")
	filePath := filepath.Join(tempDir, "a.txt")
	if err := os.WriteFile(imagePath, []byte{0x89, 'P', 'N', 'G'}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("demo"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := &stubLocalFeishuConnectClient{}
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	svc := &localFeishuService{
		name:         "feishu",
		client:       client,
		logger:       log.New(io.Discard, "", 0),
		messageLog:   io.Discard,
		downloadDir:  tempDir,
		stateFile:    filepath.Join(tempDir, "feishu.pending.json"),
		scanInterval: localFeishuScanInterval,
		expireWindow: localFeishuExpireWindow,
		nowFn:        func() time.Time { return now },
		state: localFeishuState{
			Pending: map[string]localFeishuPendingMessage{
				"image-1": {
					ExternalID: "image-1",
					MessageID:  "om-image-1",
					ChatID:     "chat-1",
					Type:       "image",
					Artifacts:  []string{imagePath},
					MessageAt:  now.Add(-20 * time.Second).UnixMilli(),
				},
				"file-1": {
					ExternalID: "file-1",
					MessageID:  "om-file-1",
					ChatID:     "chat-1",
					Type:       "file",
					Artifacts:  []string{filePath},
					MessageAt:  now.Add(-10 * time.Second).UnixMilli(),
				},
			},
			Pushed: map[string]int64{},
		},
	}

	if err := svc.flushPending(context.Background(), "feishu"); err != nil {
		t.Fatal(err)
	}
	if len(client.Requests()) != 0 {
		t.Fatalf("artifact-only messages should wait, got %+v", client.Requests())
	}
	if len(svc.state.Pending) != 2 {
		t.Fatalf("pending should remain: %+v", svc.state.Pending)
	}
}

func TestLocalFeishuFlushPendingPushesTextWithImageAndFileAfterWaiting(t *testing.T) {
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "img.png")
	filePath := filepath.Join(tempDir, "a.txt")
	if err := os.WriteFile(imagePath, []byte{0x89, 'P', 'N', 'G'}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("demo"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := &stubLocalFeishuConnectClient{}
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	svc := &localFeishuService{
		name:         "feishu",
		client:       client,
		logger:       log.New(io.Discard, "", 0),
		messageLog:   io.Discard,
		downloadDir:  tempDir,
		stateFile:    filepath.Join(tempDir, "feishu.pending.json"),
		scanInterval: localFeishuScanInterval,
		expireWindow: localFeishuExpireWindow,
		nowFn:        func() time.Time { return now },
		state: localFeishuState{
			Pending: map[string]localFeishuPendingMessage{
				"image-1": {
					ExternalID: "image-1",
					MessageID:  "om-image-1",
					ChatID:     "chat-1",
					Type:       "image",
					Artifacts:  []string{imagePath},
					MessageAt:  now.Add(-30 * time.Second).UnixMilli(),
				},
				"file-1": {
					ExternalID: "file-1",
					MessageID:  "om-file-1",
					ChatID:     "chat-1",
					Type:       "file",
					Artifacts:  []string{filePath},
					MessageAt:  now.Add(-20 * time.Second).UnixMilli(),
				},
				"text-1": {
					ExternalID: "text-1",
					MessageID:  "om-text-1",
					ChatID:     "chat-1",
					Type:       "text",
					Content:    "文本消息",
					Raw:        `{"schema":"2.0","event":{"message":{"message_id":"om-text-1"}}}`,
					MessageAt:  now.Add(-10 * time.Second).UnixMilli(),
				},
			},
			Pushed: map[string]int64{},
		},
	}

	if err := svc.flushPending(context.Background(), "feishu"); err != nil {
		t.Fatal(err)
	}
	requests := client.Requests()
	if len(requests) != 1 {
		t.Fatalf("requests len = %d, want 1", len(requests))
	}
	got := requests[0]
	wantContent := "文本消息 [image]" + imagePath + " [file]" + filePath
	if got.Content != wantContent {
		t.Fatalf("content = %q, want %q", got.Content, wantContent)
	}
	if got.Artifacts != imagePath+","+filePath {
		t.Fatalf("artifacts = %q", got.Artifacts)
	}

	var rawEnvelope struct {
		Pending []localFeishuPendingMessage `json:"pending"`
	}
	if err := json.Unmarshal([]byte(got.RawRequest), &rawEnvelope); err != nil {
		t.Fatalf("decode rawRequest: %v raw=%s", err, got.RawRequest)
	}
	if len(rawEnvelope.Pending) != 3 {
		t.Fatalf("pending len in rawRequest = %d, want 3", len(rawEnvelope.Pending))
	}
	if len(svc.state.Pending) != 0 {
		t.Fatalf("pending should be empty after push: %+v", svc.state.Pending)
	}
}

func TestLocalFeishuFlushPendingExpiresOldPendingMessages(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "a.txt")
	if err := os.WriteFile(filePath, []byte("demo"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := &stubLocalFeishuConnectClient{}
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	logBuf := &bytes.Buffer{}
	svc := &localFeishuService{
		name:         "feishu",
		client:       client,
		logger:       log.New(io.Discard, "", 0),
		messageLog:   logBuf,
		downloadDir:  tempDir,
		stateFile:    filepath.Join(tempDir, "feishu.pending.json"),
		scanInterval: localFeishuScanInterval,
		expireWindow: localFeishuExpireWindow,
		nowFn:        func() time.Time { return now },
		state: localFeishuState{
			Pending: map[string]localFeishuPendingMessage{
				"file-1": {
					ExternalID: "file-1",
					MessageID:  "om-file-1",
					ChatID:     "chat-1",
					Type:       "file",
					Artifacts:  []string{filePath},
					MessageAt:  now.Add(-11 * time.Minute).UnixMilli(),
				},
			},
			Pushed: map[string]int64{},
		},
	}

	if err := svc.flushPending(context.Background(), "feishu"); err != nil {
		t.Fatal(err)
	}
	if len(client.Requests()) != 0 {
		t.Fatalf("expired file-only message should not push, got %+v", client.Requests())
	}
	if len(svc.state.Pending) != 0 {
		t.Fatalf("expired pending should be removed: %+v", svc.state.Pending)
	}
	if !strings.Contains(logBuf.String(), "跳过了一条等待太久的飞书消息") {
		t.Fatalf("expected expiry log, got: %s", logBuf.String())
	}
}

func parseFlagMap(args []string) (map[string]string, error) {
	flags := make(map[string]string)
	for i := 0; i < len(args); i += 2 {
		if i+1 >= len(args) {
			return nil, io.ErrUnexpectedEOF
		}
		flags[strings.TrimPrefix(args[i], "--")] = args[i+1]
	}
	return flags, nil
}

type stubLocalFeishuConnectClient struct {
	requests []connectsvc.RequestInput
	metaErr  error
	addErr   error
}

func (s *stubLocalFeishuConnectClient) GetMeta(_ context.Context, name string) (*connectsvc.Meta, error) {
	if s.metaErr != nil {
		return nil, s.metaErr
	}
	return &connectsvc.Meta{Name: name, Meta: `{"appId":"demo","appSecret":"secret"}`}, nil
}

func (s *stubLocalFeishuConnectClient) AddRequest(_ context.Context, input connectsvc.RequestInput) (*connectsvc.Request, error) {
	if s.addErr != nil {
		return nil, s.addErr
	}
	s.requests = append(s.requests, input)
	return &connectsvc.Request{
		ID:             len(s.requests),
		Key:            input.Key,
		Name:           input.Name,
		ExternalID:     input.ExternalID,
		Content:        input.Content,
		Request:        input.Request,
		Artifacts:      input.Artifacts,
		Original:       input.Original,
		RawRequest:     input.RawRequest,
		ResponseSchema: input.ResponseSchema,
		CreatedAt:      input.CreatedAt,
	}, nil
}

func (s *stubLocalFeishuConnectClient) Requests() []connectsvc.RequestInput {
	out := make([]connectsvc.RequestInput, len(s.requests))
	copy(out, s.requests)
	return out
}

func TestLocalFeishuServiceStopsAfterThreeLoadConfigFailures(t *testing.T) {
	messageLog := &bytes.Buffer{}
	loggerOut := &bytes.Buffer{}
	svc := &localFeishuService{
		name:           "feishu",
		client:         &stubLocalFeishuConnectClient{metaErr: errors.New("meta unavailable")},
		logger:         log.New(loggerOut, "", 0),
		messageLog:     messageLog,
		reconnectDelay: time.Millisecond,
		nowFn: func() time.Time {
			return time.Unix(100, 0)
		},
	}

	err := svc.Run(context.Background())
	if err == nil {
		t.Fatal("expected service failure")
	}
	if !strings.Contains(err.Error(), "load-config failed after 3 attempts") {
		t.Fatalf("unexpected err: %v", err)
	}
	logText := messageLog.String()
	if !strings.Contains(logText, `重试进度=第1/3次`) {
		t.Fatalf("missing retry log: %s", logText)
	}
	if !strings.Contains(logText, `已停止自动重试`) {
		t.Fatalf("missing terminate log: %s", logText)
	}
}

func withEnv(key, value string) func() {
	old, existed := os.LookupEnv(key)
	_ = os.Setenv(key, value)
	return func() {
		if existed {
			_ = os.Setenv(key, old)
			return
		}
		_ = os.Unsetenv(key)
	}
}
