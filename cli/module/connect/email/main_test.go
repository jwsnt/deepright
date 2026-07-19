package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"connect/connectsvc"
	"connect/emailsvc"
	pop3 "github.com/knadh/go-pop3"
)

func TestEmailSchemaCommandOutput(t *testing.T) {
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
	contentSchema, ok := properties["content"].(map[string]any)
	if !ok {
		t.Fatalf("content schema type = %T", properties["content"])
	}
	if contentSchema["description"] != "The content is presented in plain text (not MARKDOWN), and the file paths have been replaced with filenames." {
		t.Fatalf("content description = %#v", contentSchema["description"])
	}

	required, ok := got["required"].([]any)
	if !ok {
		t.Fatalf("required type = %T", got["required"])
	}
	if len(required) != 1 || required[0] != "content" {
		t.Fatalf("required = %#v", required)
	}
}

func TestEmailSchemaCommandIgnoresOtherCommands(t *testing.T) {
	stdout := &bytes.Buffer{}
	if handleLocalCommand([]string{"start"}, stdout) {
		t.Fatal("handleLocalCommand should ignore non-local commands")
	}
	if stdout.Len() != 0 {
		t.Fatalf("unexpected stdout for non-schema command: %s", stdout.String())
	}
}

func TestEmailScopeCommandOutput(t *testing.T) {
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

func TestEmailCommandIncludesSchema(t *testing.T) {
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

func TestHandleLocalCommandParamReturnsDescriptiveSchema(t *testing.T) {
	var stdout bytes.Buffer
	if !handleLocalCommand([]string{"param"}, &stdout) {
		t.Fatal("handleLocalCommand should handle param")
	}

	var params []map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &params); err != nil {
		t.Fatalf("decode param output: %v raw=%s", err, stdout.String())
	}
	if len(params) != 1 {
		t.Fatalf("param len = %d, want 1; raw=%s", len(params), stdout.String())
	}
	got := params[0]
	for key, want := range map[string]string{
		"email":               "邮箱地址，如hello_world@gmail.com",
		"email_pop3":          "邮箱的pop3地址，如pop.gmail.com",
		"email_smtp":          "邮箱的smtp地址，如smtp.gmail.com",
		"email_password":      "邮箱的密码",
		"email_whitelist":     "以逗号分隔的收件人白名单，如a@gmail.com,b@gmail.com",
		"email_pop3_interval": "每次扫描待处理邮件的间隔秒数，默认300",
	} {
		if got[key] != want {
			t.Fatalf("param[%q] = %q, want %q; all=%v", key, got[key], want, got)
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

	rawContent := `{"content":"邮件正文","artifacts":[{"path":"` + imagePath + `","desc":"截图"},{"path":"` + filePath + `","desc":"说明"}],"why_do_this":"记录过程"}`
	var logs bytes.Buffer
	args := normalizeSendArgs([]string{
		"send",
		"--message", `{"id":1}`,
		"--content", rawContent,
	}, &logs)

	got, err := parseFlagMap(args[1:])
	if err != nil {
		t.Fatal(err)
	}
	if got["content"] != "邮件正文" {
		t.Fatalf("content = %q", got["content"])
	}
	if got["image"] != imagePath {
		t.Fatalf("image = %q", got["image"])
	}
	if got["file"] != filePath {
		t.Fatalf("file = %q", got["file"])
	}
	if !strings.Contains(logs.String(), `content="邮件正文"`) {
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

	rawContent := `先看看桌面文件情况，同时尝试截图。截图已生成。现在截图下载目录：{"content":"邮件正文","artifacts":[{"path":"` + imagePath + `","desc":"截图"},{"path":"` + filePath + `","desc":"说明"}],"why_do_this":"记录过程"}`
	var logs bytes.Buffer
	args := normalizeSendArgs([]string{
		"send",
		"--message", `{"id":1}`,
		"--content", rawContent,
	}, &logs)

	got, err := parseFlagMap(args[1:])
	if err != nil {
		t.Fatal(err)
	}
	if got["content"] != "邮件正文" {
		t.Fatalf("content = %q", got["content"])
	}
	if got["image"] != imagePath {
		t.Fatalf("image = %q", got["image"])
	}
	if got["file"] != filePath {
		t.Fatalf("file = %q", got["file"])
	}
	if !strings.Contains(logs.String(), `content="邮件正文"`) {
		t.Fatalf("logs missing content marker: %s", logs.String())
	}
}

func TestNormalizeSendArgsAcceptsContentOnlySchema(t *testing.T) {
	rawContent := `{"content":"only content"}`
	var logs bytes.Buffer
	args := normalizeSendArgs([]string{
		"send",
		"--content", rawContent,
		"--image", "/tmp/already.png",
	}, &logs)

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
	if !strings.Contains(logs.String(), `content="only content"`) {
		t.Fatalf("logs missing content marker: %s", logs.String())
	}
}

func TestNormalizeSendArgsKeepsOriginalWhenPrefixedTextHasNoExtractableJSON(t *testing.T) {
	rawContent := `先看看桌面文件情况，同时尝试截图。截图已生成。现在截图下载目录：{"content":}`
	args := normalizeSendArgs([]string{
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
}

func TestExtractStructuredContentFallbackLocalFromPrefixedText(t *testing.T) {
	rawContent := `先看看桌面文件情况，同时尝试截图。截图已生成。现在截图下载目录：{"content":"邮件正文","why_do_this":"记录过程"}`
	if got := extractStructuredContentFallbackLocal(rawContent); got != "邮件正文" {
		t.Fatalf("fallback content = %q", got)
	}
}

func TestNormalizeSendArgsKeepsStructuredContentWhenArtifactPathMissing(t *testing.T) {
	rawContent := `{"content":"only content","artifacts":[{"path":"/tmp/missing.txt","desc":"missing"}],"why_do_this":"test"}`
	args := normalizeSendArgs([]string{
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
	if got["file"] != "/tmp/missing.txt" {
		t.Fatalf("file = %q", got["file"])
	}
}

func TestNormalizeSendArgsLogsJSONParseReason(t *testing.T) {
	rawContent := `{"content":}`
	var logs bytes.Buffer
	args := normalizeSendArgs([]string{
		"send",
		"--content", rawContent,
	}, &logs)

	got, err := parseFlagMap(args[1:])
	if err != nil {
		t.Fatal(err)
	}
	if got["content"] != rawContent {
		t.Fatalf("content = %q, want original text", got["content"])
	}
	if !strings.Contains(logs.String(), `reason="json-parse-failed"`) {
		t.Fatalf("logs missing json parse reason: %s", logs.String())
	}
}

func TestNormalizeSendArgsKeepsOriginalWhenSchemaMismatch(t *testing.T) {
	rawContent := `{"hello":"world"}`
	args := normalizeSendArgs([]string{
		"send",
		"--content", rawContent,
		"--image", "/tmp/already.png",
	}, io.Discard)

	got, err := parseFlagMap(args[1:])
	if err != nil {
		t.Fatal(err)
	}
	if got["content"] != rawContent {
		t.Fatalf("content = %q, want original json", got["content"])
	}
	if _, ok := got["image"]; ok {
		t.Fatalf("image should be removed on schema fallback: %q", got["image"])
	}
}

func TestNormalizeSendArgsUsesContentFieldWhenImageReferenceScanFails(t *testing.T) {
	rawContent := `{"content":"好的，重新截取桌面。执行 screencapture -D 1 -x /tmp/desktop_20260612_1545.png","why_do_this":"重新获取桌面截图"}`
	args := normalizeSendArgs([]string{
		"send",
		"--content", rawContent,
	}, io.Discard)

	got, err := parseFlagMap(args[1:])
	if err != nil {
		t.Fatal(err)
	}
	if got["content"] != "好的，重新截取桌面。执行 screencapture -D 1 -x /tmp/desktop_20260612_1545.png" {
		t.Fatalf("content = %q", got["content"])
	}
	if _, ok := got["image"]; ok {
		t.Fatalf("plain command text should not be promoted to image attachment: %q", got["image"])
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

	rawContent := `{"content":"邮件正文","artifacts":[{"path":"` + imagePath + `","desc":"截图"},{"path":"` + filePath + `","desc":"文档"}],"why_do_this":"记录过程"}`
	args := normalizeSendArgs([]string{
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

func TestRunDispatchesSchemaAwareSend(t *testing.T) {
	originalLocalRunner := runLocalSendCommand
	defer func() { runLocalSendCommand = originalLocalRunner }()

	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "a.png")
	if err := os.WriteFile(imagePath, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}

	var capturedFlags map[string]string
	runLocalSendCommand = func(args []string, stdout, stderr io.Writer) (*localSendResult, bool, int) {
		normalized := normalizeSendArgs(args, io.Discard)
		flags, err := connectsvc.ParseFlags(normalized[1:])
		if err != nil {
			t.Fatal(err)
		}
		capturedFlags = flags
		return &localSendResult{
			Name:      emailsvc.DefaultName,
			To:        "sender@example.com",
			MessageID: "<reply@example.com>",
			Sent:      []localSentRecord{{Type: "text"}},
		}, true, 0
	}

	rawContent := `{"content":"邮件正文","artifacts":[{"path":"` + imagePath + `","desc":"截图"}],"why_do_this":"记录过程"}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{"send", "--content", rawContent}, &stdout, &stderr); code != 0 {
		t.Fatalf("run send exit code = %d, stderr=%s", code, stderr.String())
	}

	if got := capturedFlags["content"]; got != "邮件正文" {
		t.Fatalf("content = %q", got)
	}
	if got := capturedFlags["image"]; got != imagePath {
		t.Fatalf("image = %q", got)
	}
}

func TestRunDispatchesSchemaAwareInit(t *testing.T) {
	originalLocalRunner := runLocalSendCommand
	defer func() { runLocalSendCommand = originalLocalRunner }()

	var commandName string
	runLocalSendCommand = func(args []string, stdout, stderr io.Writer) (*localSendResult, bool, int) {
		commandName = args[0]
		return &localSendResult{
			Name:      emailsvc.DefaultName,
			To:        "sender@example.com",
			MessageID: "<reply@example.com>",
			Sent:      []localSentRecord{{Type: "text"}},
		}, true, 0
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{"init", "--content", "初始化完成", "--message", latestEmailConnectRequestMessageJSONLocal("<origin@example.com>", "原始主题", "Sender <sender@example.com>", "")}, &stdout, &stderr); code != 0 {
		t.Fatalf("run init exit code = %d, stderr=%s", code, stderr.String())
	}
	if commandName != "init" {
		t.Fatalf("command = %q", commandName)
	}
}

func TestHandleLocalCommandHelp(t *testing.T) {
	var stdout bytes.Buffer
	if !handleLocalCommand([]string{"help"}, &stdout) {
		t.Fatal("handleLocalCommand should handle help")
	}
	text := stdout.String()
	if !strings.Contains(text, "plain text body content") {
		t.Fatalf("help output should mention plain text content: %s", text)
	}
	for _, want := range []string{
		"email sender [options]",
		"email search [--query TEXT] [--sender EMAIL] [options]",
		"email search --query \"退款 已处理\" --limit 20 --offset 0 --connect-bin ./integration",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("help output missing %q: %s", want, text)
		}
	}
	if !strings.Contains(text, "default 300") {
		t.Fatalf("help output should mention 300 second default: %s", text)
	}
	if !strings.Contains(text, "邮箱的smtp地址，如smtp.gmail.com") {
		t.Fatalf("help output should mention descriptive param schema: %s", text)
	}
	if !strings.Contains(text, "--to ADDRESSES") || !strings.Contains(text, "--subject TEXT") {
		t.Fatalf("help output should mention new mail options: %s", text)
	}
	for _, unwanted := range []string{"email start [options]", "email stop [options]", "\"start\",\"stop\""} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("help output should not contain %q: %s", unwanted, text)
		}
	}
}

func TestEmailSenderQueriesGenericSnapshotCapability(t *testing.T) {
	root := t.TempDir()
	integration := filepath.Join(root, "integration")
	if err := os.WriteFile(integration, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "config.json"), []byte(`{"email":{"lastMessage":72}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	original := runEmailSnapshotQueryCommand
	defer func() { runEmailSnapshotQueryCommand = original }()
	var gotBinary string
	var gotArgs []string
	runEmailSnapshotQueryCommand = func(binary string, args []string) ([]byte, error) {
		gotBinary = binary
		gotArgs = append([]string(nil), args...)
		return []byte(`[{"senderId":"alice@example.com","lastMessageAt":"2026-07-19T10:30:00Z"}]`), nil
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"sender", "--connect-bin", integration}, &stdout, &stderr); code != 0 {
		t.Fatalf("sender code = %d, stderr = %s", code, stderr.String())
	}
	if gotBinary != integration {
		t.Fatalf("binary = %q, want %q", gotBinary, integration)
	}
	wantArgs := []string{"connect", "message-snapshot-senders", "--source", "email", "--window-hours", "72"}
	if !slices.Equal(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
	if !strings.Contains(stdout.String(), `"sender": "alice@example.com"`) {
		t.Fatalf("sender output = %s", stdout.String())
	}
}

func TestEmailSearchQueriesGenericSnapshotCapability(t *testing.T) {
	root := t.TempDir()
	integration := filepath.Join(root, "integration")
	if err := os.WriteFile(integration, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "config.json"), []byte(`{"email":{"lastMessage":24}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	original := runEmailSnapshotQueryCommand
	defer func() { runEmailSnapshotQueryCommand = original }()
	var gotArgs []string
	runEmailSnapshotQueryCommand = func(_ string, args []string) ([]byte, error) {
		gotArgs = append([]string(nil), args...)
		return []byte(`{"total":1,"limit":20,"offset":5,"items":[{"messageId":"<mail@example.com>","senderId":"alice@example.com","content":"退款已处理","sentAt":"2026-07-19T10:30:00Z"}]}`), nil
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"search", "--query", `"退款申请" 已处理`, "--limit", "20", "--offset", "5", "--connect-bin", integration}, &stdout, &stderr); code != 0 {
		t.Fatalf("search code = %d, stderr = %s", code, stderr.String())
	}
	wantArgs := []string{"connect", "message-snapshot-search", "--source", "email", "--window-hours", "24", "--query", `"退款申请" 已处理`, "--limit", "20", "--offset", "5"}
	if !slices.Equal(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
	if !strings.Contains(stdout.String(), `"sender": "alice@example.com"`) || strings.Contains(stdout.String(), "senderId") {
		t.Fatalf("search output = %s", stdout.String())
	}
}

func TestEmailSearchAllowsOmittedQueryAndFiltersSender(t *testing.T) {
	root := t.TempDir()
	integration := filepath.Join(root, "integration")
	if err := os.WriteFile(integration, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "config.json"), []byte(`{"email":{"lastMessage":24}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	original := runEmailSnapshotQueryCommand
	defer func() { runEmailSnapshotQueryCommand = original }()
	var gotArgs []string
	runEmailSnapshotQueryCommand = func(_ string, args []string) ([]byte, error) {
		gotArgs = append([]string(nil), args...)
		return []byte(`{"total":0,"limit":20,"offset":0,"items":[]}`), nil
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"search", "--sender", "Alice@Example.COM", "--limit", "20", "--connect-bin", integration}, &stdout, &stderr); code != 0 {
		t.Fatalf("search code = %d, stderr = %s", code, stderr.String())
	}
	wantArgs := []string{"connect", "message-snapshot-search", "--source", "email", "--window-hours", "24", "--sender-id", "alice@example.com", "--limit", "20"}
	if !slices.Equal(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
}

func TestEmailSenderRejectsInvalidLastMessageConfig(t *testing.T) {
	root := t.TempDir()
	integration := filepath.Join(root, "integration")
	if err := os.WriteFile(integration, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "config.json"), []byte(`{"email":{"lastMessage":0}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"sender", "--connect-bin", integration}, &stdout, &stderr); code == 0 {
		t.Fatal("sender unexpectedly succeeded")
	}
	if !strings.Contains(stderr.String(), "email.lastMessage must be a positive integer") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestLocalSchemaConnectClientInjectsSharedSchema(t *testing.T) {
	runner := stubRunnerFunc(func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if len(args) == 0 || args[0] != "add-request" {
			t.Fatalf("unexpected args: %v", args)
		}
		flags, err := connectsvc.ParseFlags(args[1:])
		if err != nil {
			t.Fatal(err)
		}
		if flags["schema"] != responseSchemaJSONLocal() {
			t.Fatalf("schema = %q", flags["schema"])
		}
		return []byte(`{"id":1}`), nil
	})

	client := localSchemaConnectClient{inner: &emailsvc.CLIConnectClient{
		Binary: "connect",
		Runner: runner,
	}}
	_, err := client.AddRequest(context.Background(), connectsvc.RequestInput{
		Key:            "email",
		Content:        "hello",
		ResponseSchema: `{"type":"object","properties":{"legacy":{"type":"string"}}}`,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestLocalConfigCompatClientPersistsDefaultEmailChatIDWithoutChangingSelectedAgent(t *testing.T) {
	runnerCalls := 0
	client := localConfigCompatClient{
		inner: stubLocalConfigConnectClient{
			meta: &connectsvc.Meta{
				Key:           "email",
				Name:          "邮件",
				Meta:          `{"email":"demo@example.com","email_pop3":"pop.example.com","email_smtp":"smtp.example.com","email_password":"secret","email_whitelist":""}`,
				Callback:      "/tmp/plugins/email",
				AgentID:       "agent-a",
				Model:         "OpenAI",
				RouterDisable: true,
			},
		},
		binary:   "integration",
		prefix:   []string{"connect"},
		baseArgs: []string{"--connect-cache", "10000"},
		runner: stubRunnerFunc(func(_ context.Context, name string, args ...string) ([]byte, error) {
			runnerCalls++
			if name != "integration" {
				t.Fatalf("binary = %q, want integration", name)
			}
			if len(args) < 2 || args[0] != "connect" || args[1] != "meta-update" {
				t.Fatalf("unexpected args: %v", args)
			}
			flags, err := connectsvc.ParseFlags(args[2:])
			if err != nil {
				t.Fatal(err)
			}
			if flags["key"] != "email" {
				t.Fatalf("key = %q", flags["key"])
			}
			if flags["chatId"] != "email" {
				t.Fatalf("chatId = %q", flags["chatId"])
			}
			if flags["agent"] != "agent-a" {
				t.Fatalf("agent = %q", flags["agent"])
			}
			if flags["model"] != "OpenAI" {
				t.Fatalf("model = %q", flags["model"])
			}
			if flags["connect-cache"] != "10000" {
				t.Fatalf("connect-cache = %q", flags["connect-cache"])
			}
			return []byte(`{"key":"email","name":"邮件","meta":{"email":"demo@example.com","email_pop3":"pop.example.com","email_smtp":"smtp.example.com","email_password":"secret","email_whitelist":""},"callback":"/tmp/plugins/email","agentId":"agent-a","chatId":"email","model":"OpenAI","router_disable":true}`), nil
		}),
	}

	meta, err := client.GetMeta(context.Background(), "email")
	if err != nil {
		t.Fatal(err)
	}
	if runnerCalls != 1 {
		t.Fatalf("runner calls = %d, want 1", runnerCalls)
	}
	if meta.AgentID != "agent-a" {
		t.Fatalf("agentId = %q, want agent-a", meta.AgentID)
	}
	if meta.ChatID != "email" {
		t.Fatalf("chatId = %q, want email", meta.ChatID)
	}
}

func TestLocalConfigCompatClientKeepsReusedChatIDAndSelectedAgent(t *testing.T) {
	client := localConfigCompatClient{
		inner: stubLocalConfigConnectClient{
			meta: &connectsvc.Meta{
				Key:           "email",
				Name:          "邮件",
				Meta:          `{"email":"demo@example.com","email_pop3":"pop.example.com","email_smtp":"smtp.example.com","email_password":"secret","email_whitelist":""}`,
				Callback:      "/tmp/plugins/email",
				AgentID:       "agent-a",
				ChatID:        "chat-reuse-1",
				Model:         "OpenAI",
				RouterDisable: true,
			},
		},
		runner: stubRunnerFunc(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			t.Fatal("runner should not be called when chatId already exists")
			return nil, nil
		}),
	}

	meta, err := client.GetMeta(context.Background(), "email")
	if err != nil {
		t.Fatal(err)
	}
	if meta.AgentID != "agent-a" {
		t.Fatalf("agentId = %q, want agent-a", meta.AgentID)
	}
	if meta.ChatID != "chat-reuse-1" {
		t.Fatalf("chatId = %q, want chat-reuse-1", meta.ChatID)
	}
}

func TestLocalConfigCompatClientLogsResolvedScanSeconds(t *testing.T) {
	var runtimeLog bytes.Buffer
	var messageLog bytes.Buffer
	client := localConfigCompatClient{
		inner: stubLocalConfigConnectClient{
			meta: &connectsvc.Meta{
				Key:           "email",
				Name:          "邮件",
				Meta:          `{"email":"demo@example.com","email_pop3":"pop.example.com","email_smtp":"smtp.example.com","email_password":"secret","email_whitelist":"","email_pop3_interval":"420"}`,
				Callback:      "/tmp/plugins/email",
				AgentID:       "agent-a",
				ChatID:        "chat-reuse-1",
				Model:         "OpenAI",
				RouterDisable: true,
			},
		},
		logger:     log.New(&runtimeLog, "", 0),
		messageLog: &messageLog,
	}

	meta, err := client.GetMeta(context.Background(), "email")
	if err != nil {
		t.Fatal(err)
	}
	if meta == nil {
		t.Fatal("meta should not be nil")
	}
	for _, text := range []string{runtimeLog.String(), messageLog.String()} {
		if !strings.Contains(text, `做了：读取了邮件插件配置`) && !strings.Contains(text, `stage=load-config`) {
			t.Fatalf("log missing load-config entry: %s", text)
		}
	}
}

type stubLocalMetaLoader struct {
	meta *connectsvc.Meta
	cfg  emailsvc.Config
	err  error
}

func (s stubLocalMetaLoader) Load(_ context.Context) (*connectsvc.Meta, emailsvc.Config, error) {
	if s.err != nil {
		return nil, emailsvc.Config{}, s.err
	}
	if s.meta == nil {
		return nil, emailsvc.Config{}, io.EOF
	}
	copyMeta := *s.meta
	return &copyMeta, s.cfg, nil
}

type stubLocalSMTPTransport struct {
	calls []stubLocalSMTPSendCall
	err   error
	errs  []error
}

type stubLocalSMTPSendCall struct {
	cfg  emailsvc.Config
	mail localOutgoingMail
}

func (s *stubLocalSMTPTransport) Send(_ context.Context, cfg emailsvc.Config, mail localOutgoingMail) (string, error) {
	s.calls = append(s.calls, stubLocalSMTPSendCall{cfg: cfg, mail: mail})
	if len(s.errs) > 0 {
		err := s.errs[0]
		s.errs = s.errs[1:]
		if err != nil {
			return "", err
		}
	}
	if s.err != nil {
		return "", s.err
	}
	if strings.TrimSpace(mail.MessageID) == "" {
		return "<stub@example.com>", nil
	}
	return mail.MessageID, nil
}

func newStubLocalSenderForTest(messageLog io.Writer, transport *stubLocalSMTPTransport) *localSender {
	if messageLog == nil {
		messageLog = io.Discard
	}
	sender := newLocalSender(stubLocalMetaLoader{
		meta: &connectsvc.Meta{Name: emailsvc.DefaultName},
		cfg: emailsvc.Config{
			Email:         "demo@example.com",
			EmailPOP3:     "pop.example.com:995",
			EmailSMTP:     "smtp.example.com:465",
			EmailPassword: "secret",
		},
	}, nil, messageLog)
	sender.transport = transport
	sender.now = func() time.Time { return time.Unix(100, 0) }
	return sender
}

func TestLocalSenderEmbedsHTMLImagesAsCID(t *testing.T) {
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "cid.png")
	filePath := filepath.Join(tempDir, "demo.pdf")
	if err := os.WriteFile(imagePath, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("%PDF"), 0o644); err != nil {
		t.Fatal(err)
	}

	messageLog := &bytes.Buffer{}
	transport := &stubLocalSMTPTransport{}
	sender := newLocalSender(stubLocalMetaLoader{
		meta: &connectsvc.Meta{Name: emailsvc.DefaultName},
		cfg: emailsvc.Config{
			Email:         "demo@example.com",
			EmailPOP3:     "pop.example.com:995",
			EmailSMTP:     "smtp.example.com:465",
			EmailPassword: "secret",
		},
	}, nil, messageLog)
	sender.transport = transport
	sender.now = func() time.Time { return time.Unix(100, 0) }

	result, err := sender.Send(context.Background(), emailsvc.SendInput{
		Action:  "send",
		Message: latestEmailConnectRequestMessageJSONLocal("<origin@example.com>", "原始主题", "Sender <sender@example.com>", ""),
		Content: `<div>正文<img src="` + imagePath + `" alt="demo"/></div>`,
		Files:   []string{filePath},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(transport.calls) != 1 {
		t.Fatalf("expected one smtp send, got %d", len(transport.calls))
	}
	call := transport.calls[0]
	if !strings.Contains(call.mail.HTMLBody, `src="cid:`) {
		t.Fatalf("html body missing cid image: %s", call.mail.HTMLBody)
	}
	if len(call.mail.Attachments) != 2 {
		t.Fatalf("expected inline image + file attachment, got %d", len(call.mail.Attachments))
	}
	if got := call.mail.Attachments[0].Disposition; got != "inline" {
		t.Fatalf("first attachment disposition = %q", got)
	}
	if strings.TrimSpace(call.mail.Attachments[0].ContentID) == "" {
		t.Fatalf("expected content id on inline attachment: %+v", call.mail.Attachments[0])
	}
	if got := call.mail.Attachments[1].Disposition; got != "attachment" {
		t.Fatalf("second attachment disposition = %q", got)
	}
	if len(result.Sent) != 3 {
		t.Fatalf("expected text+image+file sent records, got %+v", result.Sent)
	}
	logText := messageLog.String()
	if !strings.Contains(logText, `做了：开始准备发送邮件`) {
		t.Fatalf("expected request log, got: %s", logText)
	}
	if !strings.Contains(logText, `做了：确认了这封邮件要发给谁`) {
		t.Fatalf("expected parse log, got: %s", logText)
	}
	if !strings.Contains(logText, `做了：成功发出了邮件`) {
		t.Fatalf("expected result log, got: %s", logText)
	}
}

func TestLocalSenderLogsSMTPHeaderForInitAndFailure(t *testing.T) {
	messageLog := &bytes.Buffer{}
	transport := &stubLocalSMTPTransport{err: errors.New("smtp rejected")}
	sender := newLocalSender(stubLocalMetaLoader{
		meta: &connectsvc.Meta{Name: emailsvc.DefaultName},
		cfg: emailsvc.Config{
			Email:         "demo@example.com",
			EmailPOP3:     "pop.example.com:995",
			EmailSMTP:     "smtp.example.com:465",
			EmailPassword: "secret",
		},
	}, nil, messageLog)
	sender.transport = transport
	sender.now = func() time.Time { return time.Unix(100, 0) }

	_, err := sender.Send(context.Background(), emailsvc.SendInput{
		Action:  "init",
		Message: latestEmailConnectRequestMessageJSONLocal("<origin@example.com>", "原始主题", "Sender <sender@example.com>", ""),
		Content: "初始化完成",
	})
	if err == nil {
		t.Fatal("expected smtp send failure")
	}
	logText := messageLog.String()
	if !strings.Contains(logText, `动作=发送开始前通知`) {
		t.Fatalf("expected init request log, got: %s", logText)
	}
	if !strings.Contains(logText, `做了：发送邮件失败`) {
		t.Fatalf("expected init failure log, got: %s", logText)
	}
	if !strings.Contains(logText, `原因：smtp rejected`) {
		t.Fatalf("expected failure reason in init log, got: %s", logText)
	}
}

func TestLocalSenderRetriesSMTPFailuresBeforeSuccess(t *testing.T) {
	messageLog := &bytes.Buffer{}
	transport := &stubLocalSMTPTransport{
		errs: []error{errors.New("smtp retry 1"), errors.New("smtp retry 2"), nil},
	}
	sender := newStubLocalSenderForTest(messageLog, transport)

	result, err := sender.Send(context.Background(), emailsvc.SendInput{
		Action:  "send",
		Message: latestEmailConnectRequestMessageJSONLocal("<origin@example.com>", "原始主题", "Sender <sender@example.com>", ""),
		Content: "执行完成",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || len(result.Sent) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(transport.calls) != 3 {
		t.Fatalf("smtp calls = %d, want 3", len(transport.calls))
	}
	logText := messageLog.String()
	if !strings.Contains(logText, `重试进度=第1/3次`) {
		t.Fatalf("missing first retry log: %s", logText)
	}
	if !strings.Contains(logText, `重试进度=第2/3次`) {
		t.Fatalf("missing second retry log: %s", logText)
	}
	if strings.Contains(logText, `已停止继续重试`) {
		t.Fatalf("unexpected terminate log: %s", logText)
	}
}

func TestLocalSenderTerminatesAfterThreeSMTPFailures(t *testing.T) {
	messageLog := &bytes.Buffer{}
	transport := &stubLocalSMTPTransport{
		errs: []error{
			errors.New("smtp retry 1"),
			errors.New("smtp retry 2"),
			errors.New("smtp retry 3"),
		},
	}
	sender := newStubLocalSenderForTest(messageLog, transport)

	_, err := sender.Send(context.Background(), emailsvc.SendInput{
		Action:  "send",
		Message: latestEmailConnectRequestMessageJSONLocal("<origin@example.com>", "原始主题", "Sender <sender@example.com>", ""),
		Content: "执行完成",
	})
	if err == nil {
		t.Fatal("expected smtp failure")
	}
	if err.Error() != "smtp retry 3" {
		t.Fatalf("err = %q", err.Error())
	}
	if len(transport.calls) != 3 {
		t.Fatalf("smtp calls = %d, want 3", len(transport.calls))
	}
	logText := messageLog.String()
	if !strings.Contains(logText, `重试进度=第3/3次`) {
		t.Fatalf("missing third retry log: %s", logText)
	}
	if !strings.Contains(logText, `已停止继续重试`) {
		t.Fatalf("missing terminate log: %s", logText)
	}
}

func TestLocalSenderLogsSpecificParseFailureReason(t *testing.T) {
	messageLog := &bytes.Buffer{}
	sender := newStubLocalSenderForTest(messageLog, &stubLocalSMTPTransport{})

	_, err := sender.Send(context.Background(), emailsvc.SendInput{
		Action:  "send",
		Message: `{"id":1}`,
		Content: "执行完成",
	})
	if err == nil {
		t.Fatal("expected parse failure")
	}
	if err.Error() != "--to is required when message.rawRequest is not provided" {
		t.Fatalf("err = %q", err.Error())
	}
	if !strings.Contains(messageLog.String(), `原因：--to is required when message.rawRequest is not provided`) {
		t.Fatalf("expected parse failure log, got: %s", messageLog.String())
	}
}

func TestLocalSenderSendsNewMailWithoutMessage(t *testing.T) {
	transport := &stubLocalSMTPTransport{}
	sender := newStubLocalSenderForTest(io.Discard, transport)

	result, err := sender.SendWithOptions(context.Background(), localSendInput{
		SendInput: emailsvc.SendInput{
			Action:  "init",
			Content: "新的邮件正文",
		},
		To:      "Alice <alice@example.com>, bob@example.com",
		Subject: "新邮件主题",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.To != "alice@example.com,bob@example.com" {
		t.Fatalf("result to = %q", result.To)
	}
	if len(transport.calls) != 1 {
		t.Fatalf("smtp calls = %d, want 1", len(transport.calls))
	}
	mail := transport.calls[0].mail
	if got := strings.Join(mail.To, ","); got != "alice@example.com,bob@example.com" {
		t.Fatalf("mail to = %q", got)
	}
	if mail.Subject != "新邮件主题" {
		t.Fatalf("subject = %q", mail.Subject)
	}
	if mail.InReplyTo != "" || len(mail.References) != 0 {
		t.Fatalf("new mail must not have reply headers: %+v", mail)
	}
}

func TestLocalSenderSendsNewMailWithMessageWithoutRawRequest(t *testing.T) {
	transport := &stubLocalSMTPTransport{}
	sender := newStubLocalSenderForTest(io.Discard, transport)

	_, err := sender.SendWithOptions(context.Background(), localSendInput{
		SendInput: emailsvc.SendInput{
			Message: `{"id":1,"name":"email"}`,
			Content: "新的邮件正文",
		},
		To:      "recipient@example.com",
		Subject: "新邮件主题",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(transport.calls) != 1 {
		t.Fatalf("smtp calls = %d, want 1", len(transport.calls))
	}
}

func TestLocalSenderRequiresToAndSubjectForNewMail(t *testing.T) {
	cases := []struct {
		name    string
		to      string
		subject string
		wantErr string
	}{
		{name: "missing to", subject: "主题", wantErr: "--to is required when message.rawRequest is not provided"},
		{name: "missing subject", to: "recipient@example.com", wantErr: "--subject is required when message.rawRequest is not provided"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sender := newStubLocalSenderForTest(io.Discard, &stubLocalSMTPTransport{})
			_, err := sender.SendWithOptions(context.Background(), localSendInput{
				SendInput: emailsvc.SendInput{Content: "新的邮件正文"},
				To:        tc.to,
				Subject:   tc.subject,
			})
			if err == nil || err.Error() != tc.wantErr {
				t.Fatalf("err = %v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestLocalSenderRawRequestTakesPrecedenceOverToAndSubject(t *testing.T) {
	transport := &stubLocalSMTPTransport{}
	sender := newStubLocalSenderForTest(io.Discard, transport)

	_, err := sender.SendWithOptions(context.Background(), localSendInput{
		SendInput: emailsvc.SendInput{
			Action:  "send",
			Message: latestEmailConnectRequestMessageJSONLocal("<origin@example.com>", "原始主题", "Sender <sender@example.com>", "<root@example.com>"),
			Content: "回复正文",
		},
		To:      "other@example.com",
		Subject: "不应使用的主题",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(transport.calls) != 1 {
		t.Fatalf("smtp calls = %d, want 1", len(transport.calls))
	}
	mail := transport.calls[0].mail
	if got := strings.Join(mail.To, ","); got != "sender@example.com" {
		t.Fatalf("mail to = %q", got)
	}
	if mail.Subject != "Re: 原始主题" {
		t.Fatalf("subject = %q", mail.Subject)
	}
	if mail.InReplyTo != "<origin@example.com>" {
		t.Fatalf("in-reply-to = %q", mail.InReplyTo)
	}
	if got := strings.Join(mail.References, ","); got != "<root@example.com>,<origin@example.com>" {
		t.Fatalf("references = %q", got)
	}
}

func TestLocalSenderRejectsInvalidMessageAndRawRequest(t *testing.T) {
	cases := []struct {
		name    string
		message string
		wantErr string
	}{
		{
			name:    "invalid message json",
			message: "{",
			wantErr: "message must be valid json",
		},
		{
			name:    "empty raw request",
			message: `{"rawRequest":""}`,
			wantErr: "message.rawRequest must be a non-empty json string",
		},
		{
			name:    "null raw request",
			message: `{"rawRequest":null}`,
			wantErr: "message.rawRequest must be a non-empty json string",
		},
		{
			name:    "invalid raw request json",
			message: `{"rawRequest":"{"}`,
			wantErr: "message.rawRequest must be valid json",
		},
		{
			name: "missing raw request subject",
			message: mustJSONTextLocal(map[string]any{
				"rawRequest": mustJSONTextLocal(map[string]any{
					"message": map[string]any{
						"messageId": "<origin@example.com>",
						"from":      "sender@example.com",
					},
				}),
			}),
			wantErr: "message.rawRequest.message.subject not found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sender := newStubLocalSenderForTest(io.Discard, &stubLocalSMTPTransport{})
			_, err := sender.SendWithOptions(context.Background(), localSendInput{
				SendInput: emailsvc.SendInput{Message: tc.message, Content: "正文"},
				To:        "recipient@example.com",
				Subject:   "新主题",
			})
			if err == nil || err.Error() != tc.wantErr {
				t.Fatalf("err = %v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestLocalSenderValidatesMessageBeforeLoadingConfig(t *testing.T) {
	sender := newLocalSender(stubLocalMetaLoader{err: errors.New("load config should not run first")}, nil, io.Discard)
	_, err := sender.SendWithOptions(context.Background(), localSendInput{
		SendInput: emailsvc.SendInput{Message: "{", Content: "正文"},
		To:        "recipient@example.com",
		Subject:   "新主题",
	})
	if err == nil || err.Error() != "message must be valid json" {
		t.Fatalf("err = %v", err)
	}
}

func TestExtractLocalReplyEnvelopeReportsSpecificErrors(t *testing.T) {
	cases := []struct {
		name    string
		message string
		wantErr string
	}{
		{
			name:    "missing raw request",
			message: `{"id":1}`,
			wantErr: "message.rawRequest not found",
		},
		{
			name:    "invalid raw request json",
			message: mustJSONTextLocal(map[string]any{"id": 1, "rawRequest": "{"}),
			wantErr: "message.rawRequest must be valid json",
		},
		{
			name: "missing message id",
			message: mustJSONTextLocal(map[string]any{
				"id": 1,
				"rawRequest": mustJSONTextLocal(map[string]any{
					"source": "email",
					"message": map[string]any{
						"from": "Sender <sender@example.com>",
					},
				}),
			}),
			wantErr: "message.rawRequest.message.messageId not found",
		},
		{
			name: "missing from",
			message: mustJSONTextLocal(map[string]any{
				"id": 1,
				"rawRequest": mustJSONTextLocal(map[string]any{
					"source": "email",
					"message": map[string]any{
						"messageId": "<origin@example.com>",
					},
				}),
			}),
			wantErr: "message.rawRequest.message.from not found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := extractLocalReplyEnvelope(tc.message)
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != tc.wantErr {
				t.Fatalf("err = %q, want %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestLocalSenderCallbackSendCases(t *testing.T) {
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "reply.png")
	filePath := filepath.Join(tempDir, "reply.pdf")
	if err := os.WriteFile(imagePath, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("%PDF"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name                string
		content             string
		images              []string
		files               []string
		wantTypes           []string
		wantAttachmentCount int
	}{
		{
			name:                "text only",
			content:             "纯文字回复",
			wantTypes:           []string{"text"},
			wantAttachmentCount: 0,
		},
		{
			name:                "attachments only",
			images:              []string{imagePath},
			files:               []string{filePath},
			wantTypes:           []string{"image", "file"},
			wantAttachmentCount: 2,
		},
		{
			name:                "text and attachments",
			content:             "文字+附件回复",
			images:              []string{imagePath},
			files:               []string{filePath},
			wantTypes:           []string{"text", "image", "file"},
			wantAttachmentCount: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			transport := &stubLocalSMTPTransport{}
			sender := newStubLocalSenderForTest(io.Discard, transport)

			result, err := sender.Send(context.Background(), emailsvc.SendInput{
				Action:  "send",
				Message: latestEmailConnectRequestMessageJSONLocal("<origin@example.com>", "原始主题", "Sender <sender@example.com>", ""),
				Content: tc.content,
				Images:  tc.images,
				Files:   tc.files,
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(transport.calls) != 1 {
				t.Fatalf("smtp calls = %d, want 1", len(transport.calls))
			}
			if got := len(transport.calls[0].mail.Attachments); got != tc.wantAttachmentCount {
				t.Fatalf("attachments = %d, want %d", got, tc.wantAttachmentCount)
			}

			gotTypes := make([]string, 0, len(result.Sent))
			for _, item := range result.Sent {
				gotTypes = append(gotTypes, item.Type)
			}
			if strings.Join(gotTypes, ",") != strings.Join(tc.wantTypes, ",") {
				t.Fatalf("sent types = %v, want %v", gotTypes, tc.wantTypes)
			}
		})
	}
}

func TestLocalSenderPreservesCommonFileAttachmentContentTypes(t *testing.T) {
	tempDir := t.TempDir()
	files := []struct {
		name        string
		contentType string
	}{
		{name: "a.pdf", contentType: "application/pdf"},
		{name: "a.doc", contentType: "application/msword"},
		{name: "a.xls", contentType: "application/vnd.ms-excel"},
		{name: "a.ppt", contentType: "application/vnd.ms-powerpoint"},
		{name: "a.mp4", contentType: "video/mp4"},
		{name: "a.opus", contentType: "audio/ogg"},
	}

	paths := make([]string, 0, len(files))
	for _, file := range files {
		path := filepath.Join(tempDir, file.name)
		if err := os.WriteFile(path, []byte("attachment"), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}

	transport := &stubLocalSMTPTransport{}
	sender := newStubLocalSenderForTest(io.Discard, transport)
	result, err := sender.Send(context.Background(), emailsvc.SendInput{
		Action:  "send",
		Message: latestEmailConnectRequestMessageJSONLocal("<origin@example.com>", "原始主题", "Sender <sender@example.com>", ""),
		Content: "附件请查收",
		Files:   paths,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(transport.calls) != 1 {
		t.Fatalf("smtp calls = %d, want 1", len(transport.calls))
	}
	if len(result.Sent) != len(files)+1 {
		t.Fatalf("sent records = %d, want %d", len(result.Sent), len(files)+1)
	}

	gotTypes := make(map[string]string, len(transport.calls[0].mail.Attachments))
	for _, attachment := range transport.calls[0].mail.Attachments {
		gotTypes[attachment.FileName] = attachment.ContentType
	}
	for _, file := range files {
		if got := gotTypes[file.name]; got != file.contentType {
			t.Fatalf("content type for %s = %q, want %q", file.name, got, file.contentType)
		}
	}
}

func TestLocalSenderSchemaContentFailureFallsBackToRawContent(t *testing.T) {
	messageLog := &bytes.Buffer{}
	transport := &stubLocalSMTPTransport{}
	sender := newLocalSender(stubLocalMetaLoader{
		meta: &connectsvc.Meta{Name: emailsvc.DefaultName},
		cfg: emailsvc.Config{
			Email:         "demo@example.com",
			EmailPOP3:     "pop.example.com:995",
			EmailSMTP:     "smtp.example.com:465",
			EmailPassword: "secret",
		},
	}, nil, messageLog)
	sender.transport = transport

	rawJSON := `{"content":"请查看","artifacts":[{"path":"/tmp/not-exists.png","desc":"截图"}],"why_do_this":"记录过程"}`
	result, err := sender.Send(context.Background(), emailsvc.SendInput{
		Action:  "send",
		Message: latestEmailConnectRequestMessageJSONLocal("<origin@example.com>", "原始主题", "Sender <sender@example.com>", ""),
		Content: rawJSON,
		Images:  []string{"/tmp/existing.png"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(transport.calls) != 1 {
		t.Fatalf("expected one smtp send, got %d", len(transport.calls))
	}
	if transport.calls[0].mail.TextBody != "请查看" {
		t.Fatalf("fallback text body = %q", transport.calls[0].mail.TextBody)
	}
	if len(transport.calls[0].mail.Attachments) != 0 {
		t.Fatalf("fallback should not send attachments: %+v", transport.calls[0].mail.Attachments)
	}
	if len(result.Sent) != 1 || result.Sent[0].Type != "text" {
		t.Fatalf("fallback sent records = %+v", result.Sent)
	}
}

func TestCollectReferencedImagesLocalDownloadsRemoteImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{137, 80, 78, 71, 13, 10, 26, 10})
	}))
	defer server.Close()

	images, err := collectReferencedImagesLocal(context.Background(), `远程图片 `+server.URL+`/a.png`, nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 {
		t.Fatalf("images = %+v", images)
	}
	if _, err := os.Stat(images[0]); err != nil {
		t.Fatalf("downloaded image missing: %v", err)
	}
}

func TestCollectReferencedImagesLocalAcceptsFileScheme(t *testing.T) {
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "a.png")
	if err := os.WriteFile(imagePath, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}

	images, err := collectReferencedImagesLocal(context.Background(), `<img src="file://`+imagePath+`" />`, nil, tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 || images[0] != imagePath {
		t.Fatalf("images = %+v", images)
	}
}

func TestBuildSMTPMessageLocalIncludesInlineCIDPart(t *testing.T) {
	raw, err := buildSMTPMessageLocal(localOutgoingMail{
		From:       "demo@example.com",
		To:         []string{"sender@example.com"},
		Subject:    "Re: 测试主题",
		TextBody:   "hello",
		HTMLBody:   `<div>hello<img src="cid:img-1"/></div>`,
		InReplyTo:  "<origin@example.com>",
		References: []string{"<root@example.com>", "<origin@example.com>"},
		MessageID:  "<reply@example.com>",
		Attachments: []localAttachment{
			{
				Type:        "image",
				Path:        "/tmp/a.png",
				FileName:    "a.png",
				ContentType: "image/png",
				Data:        []byte("png"),
				Disposition: "inline",
				ContentID:   "img-1",
			},
			{
				Type:        "file",
				Path:        "/tmp/a.pdf",
				FileName:    "a.pdf",
				ContentType: "application/pdf",
				Data:        []byte("%PDF"),
				Disposition: "attachment",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	rawText := string(raw)
	if !strings.Contains(rawText, "Content-Id: <img-1>") {
		t.Fatalf("raw message missing content-id: %s", rawText)
	}
	if !strings.Contains(rawText, "Content-Disposition: inline; filename=\"a.png\"") {
		t.Fatalf("raw message missing inline disposition: %s", rawText)
	}
	if !strings.Contains(rawText, "multipart/related") {
		t.Fatalf("raw message missing multipart/related: %s", rawText)
	}
}

func TestNormalizeLocalMetaConfigUsesPOP3Interval(t *testing.T) {
	meta, err := normalizeLocalMetaConfig(&connectsvc.Meta{
		Name: "email",
		Meta: `{"email":"demo@example.com","email_pop3":"pop.example.com","email_smtp":"smtp.example.com","email_password":"secret","email_whitelist":"","email_pop3_interval":"420"}`,
	})
	if err != nil {
		t.Fatal(err)
	}

	var cfg map[string]any
	if err := json.Unmarshal([]byte(meta.Meta), &cfg); err != nil {
		t.Fatalf("decode normalized meta: %v raw=%s", err, meta.Meta)
	}
	if got := cfg["scanSeconds"]; got != float64(420) {
		t.Fatalf("scanSeconds = %#v, want 420", got)
	}
}

func TestNormalizeLocalMetaConfigFallsBackToDefaultScanSeconds(t *testing.T) {
	meta, err := normalizeLocalMetaConfig(&connectsvc.Meta{
		Name: "email",
		Meta: `{"email":"demo@example.com","email_pop3":"pop.example.com","email_smtp":"smtp.example.com","email_password":"secret","email_whitelist":""}`,
	})
	if err != nil {
		t.Fatal(err)
	}

	var cfg map[string]any
	if err := json.Unmarshal([]byte(meta.Meta), &cfg); err != nil {
		t.Fatalf("decode normalized meta: %v raw=%s", err, meta.Meta)
	}
	if got := cfg["scanSeconds"]; got != float64(defaultPOP3ScanSecsLocal) {
		t.Fatalf("scanSeconds = %#v, want %d", got, defaultPOP3ScanSecsLocal)
	}
}

func TestRotatingLocalLogWriterRotatesBackups(t *testing.T) {
	path := filepath.Join(t.TempDir(), "email.log")
	writer := newRotatingLocalLogWriter(path, 16, 4)
	defer writer.Close()

	chunk := []byte("1234567890\n")
	for i := 0; i < 8; i++ {
		if _, err := writer.Write(chunk); err != nil {
			t.Fatalf("write chunk %d: %v", i, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected active log file: %v", err)
	}
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("expected first backup file: %v", err)
	}
}

func TestLocalPersistentPOP3FetcherReconnectsAcrossFetches(t *testing.T) {
	first := &stubLocalPOP3Session{
		uidls: []pop3.MessageID{{ID: 1, UID: "uid-1"}},
		rawByID: map[int][]byte{
			1: []byte(testRawMailLocal()),
		},
	}
	second := &stubLocalPOP3Session{
		uidls: []pop3.MessageID{{ID: 1, UID: "uid-1"}},
		rawByID: map[int][]byte{
			1: []byte(testRawMailLocal()),
		},
	}
	sessions := []*stubLocalPOP3Session{first, second}
	openCalls := 0
	fetcher := newLocalPersistentPOP3Fetcher(t.TempDir())
	fetcher.nowFn = func() time.Time {
		return time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	}
	fetcher.openConn = func(_ context.Context, host string, port int) (localPOP3Session, error) {
		openCalls++
		if host != "pop.example.com" || port != 995 {
			t.Fatalf("unexpected connect target %s:%d", host, port)
		}
		if openCalls > len(sessions) {
			t.Fatalf("unexpected extra reconnect")
		}
		return sessions[openCalls-1], nil
	}

	cfg := emailsvc.Config{
		Email:         "demo@example.com",
		EmailPOP3:     "pop.example.com:995",
		EmailPassword: "secret",
	}
	for i := 0; i < 2; i++ {
		mails, err := fetcher.Fetch(context.Background(), cfg, time.Time{})
		if err != nil {
			t.Fatalf("fetch #%d failed: %v", i+1, err)
		}
		if len(mails) != 1 {
			t.Fatalf("fetch #%d mails = %d, want 1", i+1, len(mails))
		}
	}

	if openCalls != 2 {
		t.Fatalf("open calls = %d, want 2", openCalls)
	}
	if first.authCalls != 1 {
		t.Fatalf("first auth calls = %d, want 1", first.authCalls)
	}
	if second.authCalls != 1 {
		t.Fatalf("second auth calls = %d, want 1", second.authCalls)
	}
	if first.quitCalls != 1 {
		t.Fatalf("first quit calls = %d, want 1 after first scan", first.quitCalls)
	}
	if second.quitCalls != 1 {
		t.Fatalf("second quit calls = %d, want 1 after second scan", second.quitCalls)
	}
	if first.uidlCalls != 1 || second.uidlCalls != 1 {
		t.Fatalf("uidl calls = (%d,%d), want (1,1)", first.uidlCalls, second.uidlCalls)
	}
	if first.retrCalls != 1 || second.retrCalls != 1 {
		t.Fatalf("retr calls = (%d,%d), want (1,1)", first.retrCalls, second.retrCalls)
	}
}

func TestLocalPersistentPOP3FetcherReconnectsAfterSessionDrop(t *testing.T) {
	first := &stubLocalPOP3Session{
		uidls: []pop3.MessageID{{ID: 1, UID: "uid-1"}},
		rawByID: map[int][]byte{
			1: []byte(testRawMailLocal()),
		},
		noopErr: errors.New("connection closed"),
	}
	second := &stubLocalPOP3Session{
		uidls: []pop3.MessageID{{ID: 1, UID: "uid-1"}},
		rawByID: map[int][]byte{
			1: []byte(testRawMailLocal()),
		},
	}
	sessions := []*stubLocalPOP3Session{first, second}
	openCalls := 0
	fetcher := newLocalPersistentPOP3Fetcher(t.TempDir())
	fetcher.openConn = func(_ context.Context, _ string, _ int) (localPOP3Session, error) {
		if openCalls >= len(sessions) {
			t.Fatalf("unexpected extra reconnect")
		}
		session := sessions[openCalls]
		openCalls++
		return session, nil
	}

	cfg := emailsvc.Config{
		Email:         "demo@example.com",
		EmailPOP3:     "pop.example.com:995",
		EmailPassword: "secret",
	}
	if _, err := fetcher.Fetch(context.Background(), cfg, time.Time{}); err != nil {
		t.Fatalf("first fetch failed: %v", err)
	}
	if _, err := fetcher.Fetch(context.Background(), cfg, time.Time{}); err != nil {
		t.Fatalf("second fetch failed: %v", err)
	}

	if openCalls != 2 {
		t.Fatalf("open calls = %d, want 2", openCalls)
	}
	if first.quitCalls != 1 {
		t.Fatalf("first session quit calls = %d, want 1", first.quitCalls)
	}
	if second.authCalls != 1 {
		t.Fatalf("second session auth calls = %d, want 1", second.authCalls)
	}
}

func TestLocalPersistentPOP3FetcherReconnectsAfterEOFOnUIDL(t *testing.T) {
	first := &stubLocalPOP3Session{
		uidlErr: io.EOF,
	}
	second := &stubLocalPOP3Session{
		uidls: []pop3.MessageID{{ID: 1, UID: "uid-1"}},
		rawByID: map[int][]byte{
			1: []byte(testRawMailLocal()),
		},
	}
	sessions := []*stubLocalPOP3Session{first, second}
	openCalls := 0
	fetcher := newLocalPersistentPOP3Fetcher(t.TempDir())
	fetcher.openConn = func(_ context.Context, _ string, _ int) (localPOP3Session, error) {
		if openCalls >= len(sessions) {
			t.Fatalf("unexpected extra reconnect")
		}
		session := sessions[openCalls]
		openCalls++
		return session, nil
	}

	cfg := emailsvc.Config{
		Email:         "demo@example.com",
		EmailPOP3:     "pop.example.com:995",
		EmailPassword: "secret",
	}
	mails, err := fetcher.Fetch(context.Background(), cfg, time.Time{})
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(mails) != 1 {
		t.Fatalf("mails = %d, want 1", len(mails))
	}
	if openCalls != 2 {
		t.Fatalf("open calls = %d, want 2", openCalls)
	}
	if first.quitCalls != 1 {
		t.Fatalf("first session quit calls = %d, want 1", first.quitCalls)
	}
}

func TestLocalPersistentPOP3FetcherDropsTimedOutSessionBeforeNextScan(t *testing.T) {
	first := &stubLocalPOP3Session{
		uidlErr: timeoutNetErrorLocal{},
	}
	second := &stubLocalPOP3Session{
		uidls: []pop3.MessageID{{ID: 1, UID: "uid-1"}},
		rawByID: map[int][]byte{
			1: []byte(testRawMailLocal()),
		},
	}
	sessions := []*stubLocalPOP3Session{first, second}
	openCalls := 0
	fetcher := newLocalPersistentPOP3Fetcher(t.TempDir())
	fetcher.openConn = func(_ context.Context, _ string, _ int) (localPOP3Session, error) {
		if openCalls >= len(sessions) {
			t.Fatalf("unexpected extra reconnect")
		}
		session := sessions[openCalls]
		openCalls++
		return session, nil
	}

	cfg := emailsvc.Config{
		Email:         "demo@example.com",
		EmailPOP3:     "pop.example.com:995",
		EmailPassword: "secret",
	}
	if _, err := fetcher.Fetch(context.Background(), cfg, time.Time{}); err == nil {
		t.Fatal("first fetch unexpectedly succeeded")
	}
	if first.quitCalls != 1 {
		t.Fatalf("first session quit calls = %d, want 1 after timeout", first.quitCalls)
	}

	mails, err := fetcher.Fetch(context.Background(), cfg, time.Time{})
	if err != nil {
		t.Fatalf("second fetch failed: %v", err)
	}
	if len(mails) != 1 {
		t.Fatalf("mails = %d, want 1", len(mails))
	}
	if openCalls != 2 {
		t.Fatalf("open calls = %d, want 2", openCalls)
	}
}

func TestLocalPersistentPOP3FetcherUsesUIDIncrementalInsteadOfTimelineAfterHistoryExists(t *testing.T) {
	session := &stubLocalPOP3Session{
		uidls: []pop3.MessageID{
			{ID: 1, UID: "uid-old"},
			{ID: 2, UID: "uid-new"},
		},
		rawByID: map[int][]byte{
			1: []byte(testRawMailLocalWith("Old mail", "<old@example.com>", "already processed")),
			2: []byte(
				"From: sender@example.com\r\n" +
					"To: demo@example.com\r\n" +
					"Subject: New mail\r\n" +
					"Message-ID: <new@example.com>\r\n" +
					"Date: Wed, 20 May 2026 09:00:00 +0000\r\n" +
					"Content-Type: text/plain; charset=utf-8\r\n" +
					"\r\n" +
					"brand new\r\n",
			),
		},
	}
	fetcher := newLocalPersistentPOP3Fetcher(t.TempDir())
	fetcher.ConfigureIncremental(map[string]struct{}{"uid-old": {}}, false)
	fetcher.openConn = func(_ context.Context, _ string, _ int) (localPOP3Session, error) {
		return session, nil
	}

	cfg := emailsvc.Config{
		Email:         "demo@example.com",
		EmailPOP3:     "pop.example.com:995",
		EmailPassword: "secret",
	}
	since := time.Date(2026, 5, 20, 10, 30, 0, 0, time.UTC)
	mails, err := fetcher.Fetch(context.Background(), cfg, since)
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(mails) != 1 {
		t.Fatalf("mails = %d, want 1", len(mails))
	}
	if mails[0].UID != "uid-new" {
		t.Fatalf("uid = %q, want uid-new", mails[0].UID)
	}
	if session.retrCalls != 1 {
		t.Fatalf("retr calls = %d, want 1 because known uid should be skipped", session.retrCalls)
	}
}

func TestLocalPersistentPOP3FetcherOnlyReturnsRecentWindowMails(t *testing.T) {
	session := &stubLocalPOP3Session{
		uidls: []pop3.MessageID{
			{ID: 1, UID: "uid-too-old"},
			{ID: 2, UID: "uid-recent"},
		},
		rawByID: map[int][]byte{
			1: []byte(
				"From: sender@example.com\r\n" +
					"To: demo@example.com\r\n" +
					"Subject: Too old\r\n" +
					"Message-ID: <too-old@example.com>\r\n" +
					"Date: Wed, 20 May 2026 09:49:00 +0000\r\n" +
					"Content-Type: text/plain; charset=utf-8\r\n" +
					"\r\n" +
					"too old\r\n",
			),
			2: []byte(
				"From: sender@example.com\r\n" +
					"To: demo@example.com\r\n" +
					"Subject: Recent\r\n" +
					"Message-ID: <recent@example.com>\r\n" +
					"Date: Wed, 20 May 2026 09:57:00 +0000\r\n" +
					"Content-Type: text/plain; charset=utf-8\r\n" +
					"\r\n" +
					"recent\r\n",
			),
		},
	}
	fetcher := newLocalPersistentPOP3Fetcher(t.TempDir())
	fetcher.nowFn = func() time.Time {
		return time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	}
	fetcher.ConfigureIncremental(nil, false)
	fetcher.openConn = func(_ context.Context, _ string, _ int) (localPOP3Session, error) {
		return session, nil
	}

	cfg := emailsvc.Config{
		Email:         "demo@example.com",
		EmailPOP3:     "pop.example.com:995",
		EmailPassword: "secret",
		ScanSeconds:   300,
	}
	mails, err := fetcher.Fetch(context.Background(), cfg, time.Date(2026, 5, 20, 10, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(mails) != 1 {
		t.Fatalf("mails = %d, want 1", len(mails))
	}
	if mails[0].UID != "uid-recent" {
		t.Fatalf("uid = %q, want uid-recent", mails[0].UID)
	}
}

func TestParseFetchedMailLocalPrefersReceivedHeaderTime(t *testing.T) {
	raw := "Received: from mx.example.com by server.example.com; Thu, 12 Jun 2026 15:57:00 +0800\r\n" +
		"From: sender@example.com\r\n" +
		"To: demo@example.com\r\n" +
		"Subject: Delayed date\r\n" +
		"Message-ID: <received-local@example.com>\r\n" +
		"Date: Thu, 12 Jun 2026 09:00:00 +0800\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"\r\n" +
		"hello\r\n"

	item, err := parseFetchedMailLocal(localRawMail{UID: "uid-1", Raw: []byte(raw)}, t.TempDir(), time.Date(2026, 6, 12, 16, 0, 0, 0, time.FixedZone("CST", 8*3600)))
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 6, 12, 15, 57, 0, 0, time.FixedZone("CST", 8*3600))
	if !item.ReceivedAt.Equal(want) {
		t.Fatalf("receivedAt = %s, want %s", item.ReceivedAt.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

func TestDeadlineConnLocalReadTimesOut(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()

	conn := &deadlineConnLocal{
		Conn:        left,
		readTimeout: 20 * time.Millisecond,
	}
	start := time.Now()
	_, err := conn.Read(make([]byte, 1))
	if err == nil {
		t.Fatal("read unexpectedly succeeded")
	}
	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("read err = %v, want timeout", err)
	}
	if elapsed := time.Since(start); elapsed > 250*time.Millisecond {
		t.Fatalf("read timeout took too long: %s", elapsed)
	}
}

func TestLocalPersistentPOP3FetcherReconnectsAfterEOFOnRetr(t *testing.T) {
	first := &stubLocalPOP3Session{
		uidls:   []pop3.MessageID{{ID: 1, UID: "uid-1"}},
		retrErr: io.EOF,
	}
	second := &stubLocalPOP3Session{
		uidls: []pop3.MessageID{{ID: 1, UID: "uid-1"}},
		rawByID: map[int][]byte{
			1: []byte(testRawMailLocal()),
		},
	}
	sessions := []*stubLocalPOP3Session{first, second}
	openCalls := 0
	fetcher := newLocalPersistentPOP3Fetcher(t.TempDir())
	fetcher.openConn = func(_ context.Context, _ string, _ int) (localPOP3Session, error) {
		if openCalls >= len(sessions) {
			t.Fatalf("unexpected extra reconnect")
		}
		session := sessions[openCalls]
		openCalls++
		return session, nil
	}

	cfg := emailsvc.Config{
		Email:         "demo@example.com",
		EmailPOP3:     "pop.example.com:995",
		EmailPassword: "secret",
	}
	mails, err := fetcher.Fetch(context.Background(), cfg, time.Time{})
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(mails) != 1 {
		t.Fatalf("mails = %d, want 1", len(mails))
	}
	if openCalls != 2 {
		t.Fatalf("open calls = %d, want 2", openCalls)
	}
	if first.quitCalls != 1 {
		t.Fatalf("first session quit calls = %d, want 1", first.quitCalls)
	}
}

func TestLocalPersistentPOP3FetcherReconnectsFromInterruptedMessage(t *testing.T) {
	first := &stubLocalPOP3Session{
		uidls: []pop3.MessageID{{ID: 1, UID: "uid-1"}, {ID: 2, UID: "uid-2"}},
		rawByID: map[int][]byte{
			1: []byte(testRawMailLocalWith("First mail", "<test-1@example.com>", "hello one")),
		},
		retrErrByID: map[int][]error{
			2: {io.EOF},
		},
	}
	second := &stubLocalPOP3Session{
		uidls: []pop3.MessageID{{ID: 1, UID: "uid-1"}, {ID: 2, UID: "uid-2"}},
		retrErrByID: map[int][]error{
			2: {io.EOF},
		},
	}
	third := &stubLocalPOP3Session{
		uidls: []pop3.MessageID{{ID: 1, UID: "uid-1"}, {ID: 2, UID: "uid-2"}},
		rawByID: map[int][]byte{
			2: []byte(testRawMailLocalWith("Second mail", "<test-2@example.com>", "hello two")),
		},
	}
	sessions := []*stubLocalPOP3Session{first, second, third}
	openCalls := 0
	fetcher := newLocalPersistentPOP3Fetcher(t.TempDir())
	fetcher.openConn = func(_ context.Context, _ string, _ int) (localPOP3Session, error) {
		if openCalls >= len(sessions) {
			t.Fatalf("unexpected extra reconnect")
		}
		session := sessions[openCalls]
		openCalls++
		return session, nil
	}

	cfg := emailsvc.Config{
		Email:         "demo@example.com",
		EmailPOP3:     "pop.example.com:995",
		EmailPassword: "secret",
	}
	mails, err := fetcher.Fetch(context.Background(), cfg, time.Time{})
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(mails) != 2 {
		t.Fatalf("mails = %d, want 2", len(mails))
	}
	if mails[0].Subject != "First mail" {
		t.Fatalf("first subject = %q", mails[0].Subject)
	}
	if mails[1].Subject != "Second mail" {
		t.Fatalf("second subject = %q", mails[1].Subject)
	}
	if openCalls != 3 {
		t.Fatalf("open calls = %d, want 3", openCalls)
	}
}

func TestLocalPersistentPOP3FetcherFallsBackToRawMailOnParseError(t *testing.T) {
	session := &stubLocalPOP3Session{
		uidls: []pop3.MessageID{{ID: 1, UID: "uid-1"}},
		rawByID: map[int][]byte{
			1: []byte("From: sender@example.com\r\n" +
				"To: demo@example.com\r\n" +
				"Subject: Broken multipart\r\n" +
				"Message-ID: <broken@example.com>\r\n" +
				"Date: Wed, 20 May 2026 10:00:00 +0000\r\n" +
				"Content-Type: multipart/mixed\r\n" +
				"\r\n" +
				"broken body\r\n"),
		},
	}
	fetcher := newLocalPersistentPOP3Fetcher(t.TempDir())
	fetcher.openConn = func(_ context.Context, _ string, _ int) (localPOP3Session, error) {
		return session, nil
	}

	cfg := emailsvc.Config{
		Email:         "demo@example.com",
		EmailPOP3:     "pop.example.com:995",
		EmailPassword: "secret",
	}
	mails, err := fetcher.Fetch(context.Background(), cfg, time.Time{})
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(mails) != 1 {
		t.Fatalf("mails = %d, want 1", len(mails))
	}
	if mails[0].Subject != "Broken multipart" {
		t.Fatalf("subject = %q", mails[0].Subject)
	}
	if !strings.Contains(mails[0].TextBody, "broken body") {
		t.Fatalf("text body = %q", mails[0].TextBody)
	}
}

func TestLocalPersistentPOP3FetcherReconnectsWhenConfigChanges(t *testing.T) {
	first := &stubLocalPOP3Session{
		uidls: []pop3.MessageID{{ID: 1, UID: "uid-1"}},
		rawByID: map[int][]byte{
			1: []byte(testRawMailLocal()),
		},
	}
	second := &stubLocalPOP3Session{
		uidls: []pop3.MessageID{{ID: 1, UID: "uid-1"}},
		rawByID: map[int][]byte{
			1: []byte(testRawMailLocal()),
		},
	}
	sessions := []*stubLocalPOP3Session{first, second}
	openCalls := 0
	fetcher := newLocalPersistentPOP3Fetcher(t.TempDir())
	fetcher.openConn = func(_ context.Context, _ string, _ int) (localPOP3Session, error) {
		session := sessions[openCalls]
		openCalls++
		return session, nil
	}

	cfg := emailsvc.Config{
		Email:         "demo@example.com",
		EmailPOP3:     "pop.example.com:995",
		EmailPassword: "secret",
	}
	if _, err := fetcher.Fetch(context.Background(), cfg, time.Time{}); err != nil {
		t.Fatalf("first fetch failed: %v", err)
	}

	cfg.EmailPassword = "rotated-secret"
	if _, err := fetcher.Fetch(context.Background(), cfg, time.Time{}); err != nil {
		t.Fatalf("second fetch failed: %v", err)
	}

	if openCalls != 2 {
		t.Fatalf("open calls = %d, want 2", openCalls)
	}
	if first.quitCalls != 1 {
		t.Fatalf("first session quit calls = %d, want 1 after config change", first.quitCalls)
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

type stubRunnerFunc func(ctx context.Context, name string, args ...string) ([]byte, error)

func (f stubRunnerFunc) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return f(ctx, name, args...)
}

type stubLocalConfigConnectClient struct {
	meta       *connectsvc.Meta
	getMetaErr error
}

func (s stubLocalConfigConnectClient) GetMeta(_ context.Context, _ string) (*connectsvc.Meta, error) {
	if s.getMetaErr != nil {
		return nil, s.getMetaErr
	}
	if s.meta == nil {
		return nil, nil
	}
	copyMeta := *s.meta
	return &copyMeta, nil
}

func (s stubLocalConfigConnectClient) AddRequest(_ context.Context, _ connectsvc.RequestInput) (*connectsvc.Request, error) {
	return &connectsvc.Request{ID: 1}, nil
}

type stubLocalPOP3Session struct {
	authCalls int
	noopCalls int
	uidlCalls int
	retrCalls int
	quitCalls int

	authErr error
	noopErr error
	uidlErr error
	retrErr error

	uidls       []pop3.MessageID
	rawByID     map[int][]byte
	retrErrByID map[int][]error
}

func (s *stubLocalPOP3Session) Auth(_ string, _ string) error {
	s.authCalls++
	return s.authErr
}

func (s *stubLocalPOP3Session) Noop() error {
	s.noopCalls++
	return s.noopErr
}

func (s *stubLocalPOP3Session) Uidl(_ int) ([]pop3.MessageID, error) {
	s.uidlCalls++
	if s.uidlErr != nil {
		return nil, s.uidlErr
	}
	return append([]pop3.MessageID(nil), s.uidls...), nil
}

func (s *stubLocalPOP3Session) RetrRaw(msgID int) (*bytes.Buffer, error) {
	s.retrCalls++
	if len(s.retrErrByID[msgID]) > 0 {
		err := s.retrErrByID[msgID][0]
		s.retrErrByID[msgID] = s.retrErrByID[msgID][1:]
		return nil, err
	}
	if s.retrErr != nil {
		return nil, s.retrErr
	}
	return bytes.NewBuffer(append([]byte(nil), s.rawByID[msgID]...)), nil
}

func (s *stubLocalPOP3Session) Quit() error {
	s.quitCalls++
	return nil
}

type timeoutNetErrorLocal struct{}

func (timeoutNetErrorLocal) Error() string   { return "i/o timeout" }
func (timeoutNetErrorLocal) Timeout() bool   { return true }
func (timeoutNetErrorLocal) Temporary() bool { return true }

type stubLocalProcessHandle struct {
	signalFn func(os.Signal) error
}

func (s stubLocalProcessHandle) Signal(sig os.Signal) error {
	if s.signalFn != nil {
		return s.signalFn(sig)
	}
	return nil
}

func TestStopProcessLocalTreatsZombiePIDAsStopped(t *testing.T) {
	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "email.pid")
	if err := os.WriteFile(pidFile, []byte("82675"), 0o644); err != nil {
		t.Fatal(err)
	}

	originalFindProcessLocal := findProcessLocal
	originalExecCommandLocal := execCommandLocal
	defer func() { findProcessLocal = originalFindProcessLocal }()
	defer func() { execCommandLocal = originalExecCommandLocal }()

	findProcessLocal = func(pid int) (processHandleLocal, error) {
		return stubLocalProcessHandle{signalFn: func(sig os.Signal) error { return nil }}, nil
	}
	execCommandLocal = func(name string, args ...string) *exec.Cmd {
		if name == "ps" {
			return exec.Command("sh", "-c", "printf 'Z\\n'")
		}
		return exec.Command(name, args...)
	}

	result, err := stopProcessLocal(map[string]string{
		"pid-file": pidFile,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "stopped" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if _, statErr := os.Stat(pidFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("pid file should be removed, stat err=%v", statErr)
	}
}

func latestEmailConnectRequestMessageJSONLocal(messageID, subject, from, references string) string {
	headers := []map[string]string{
		{"name": "From", "value": from},
		{"name": "Subject", "value": subject},
		{"name": "Message-ID", "value": messageID},
	}
	if strings.TrimSpace(references) != "" {
		headers = append(headers, map[string]string{"name": "References", "value": references})
	}
	raw := map[string]any{
		"headers": headers,
		"content": "hello",
	}
	envelope := map[string]any{
		"source":     emailsvc.DefaultName,
		"receivedAt": "2026-05-22T00:00:00+08:00",
		"message": map[string]any{
			"uid":        "uid-1",
			"messageId":  messageID,
			"createTime": "2026-05-22T00:00:00+08:00",
			"subject":    subject,
			"from":       from,
			"content":    "hello",
			"artifacts":  []string{},
			"raw":        mustJSONTextLocal(raw),
		},
	}
	request := map[string]any{
		"id":         1,
		"name":       emailsvc.DefaultName,
		"request":    "hello",
		"rawRequest": mustJSONTextLocal(envelope),
	}
	return mustJSONTextLocal(request)
}

func mustJSONTextLocal(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func testRawMailLocal() string {
	return testRawMailLocalWith("Test mail", "<test-1@example.com>", "hello world")
}

func testRawMailLocalWith(subject, messageID, body string) string {
	return "From: sender@example.com\r\n" +
		"To: demo@example.com\r\n" +
		"Subject: " + subject + "\r\n" +
		"Message-ID: " + messageID + "\r\n" +
		"Date: Wed, 20 May 2026 10:00:00 +0000\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"\r\n" +
		body + "\r\n"
}
