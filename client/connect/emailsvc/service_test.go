package emailsvc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/mail"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"connect/connectsvc"
	pop3 "github.com/knadh/go-pop3"
)

func TestCLIParamAndName(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	if code := RunCLI([]string{"command"}, stdout, stderr); code != 0 {
		t.Fatalf("command exit code = %d, stderr=%s", code, stderr.String())
	}
	var commands []string
	if err := json.Unmarshal(stdout.Bytes(), &commands); err != nil {
		t.Fatalf("decode commands: %v raw=%s", err, stdout.String())
	}
	if strings.Join(commands, ",") != "command,help,name,param,scope,schema,init,send,start,stop" {
		t.Fatalf("unexpected commands: %+v", commands)
	}

	stdout.Reset()
	stderr.Reset()

	if code := RunCLI([]string{"scope"}, stdout, stderr); code != 0 {
		t.Fatalf("scope exit code = %d, stderr=%s", code, stderr.String())
	}
	var scopes []string
	if err := json.Unmarshal(stdout.Bytes(), &scopes); err != nil {
		t.Fatalf("decode scopes: %v raw=%s", err, stdout.String())
	}
	if !reflect.DeepEqual(scopes, []string{"reuse", "agent", "provider", "thinking", "swarm"}) {
		t.Fatalf("unexpected scopes: %+v", scopes)
	}

	stdout.Reset()
	stderr.Reset()

	if code := RunCLI([]string{"schema"}, stdout, stderr); code != 0 {
		t.Fatalf("schema exit code = %d, stderr=%s", code, stderr.String())
	}
	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatalf("decode schema: %v raw=%s", err, stdout.String())
	}
	if schema["type"] != "object" {
		t.Fatalf("unexpected schema output: %+v", schema)
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema properties type = %T", schema["properties"])
	}
	content, ok := properties["content"].(map[string]any)
	if !ok {
		t.Fatalf("schema content type = %T", properties["content"])
	}
	if content["description"] != "The content is presented in `HTML` format, and the file paths have been replaced with filenames." {
		t.Fatalf("content description = %#v", content["description"])
	}

	stdout.Reset()
	stderr.Reset()

	if code := RunCLI([]string{"param"}, stdout, stderr); code != 0 {
		t.Fatalf("param exit code = %d, stderr=%s", code, stderr.String())
	}
	var params []string
	if err := json.Unmarshal(stdout.Bytes(), &params); err != nil {
		t.Fatalf("decode params: %v raw=%s", err, stdout.String())
	}
	if len(params) != 5 || params[0] != "email" || params[1] != "email_pop3" || params[2] != "email_smtp" || params[3] != "email_password" || params[4] != "email_whitelist" {
		t.Fatalf("unexpected params: %+v", params)
	}

	stdout.Reset()
	stderr.Reset()
	if code := RunCLI([]string{"name"}, stdout, stderr); code != 0 {
		t.Fatalf("name exit code = %d, stderr=%s", code, stderr.String())
	}
	var name struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &name); err != nil {
		t.Fatalf("decode name: %v raw=%s", err, stdout.String())
	}
	if name.Key != DefaultName || name.Name != DefaultDisplayName {
		t.Fatalf("unexpected name output: %+v", name)
	}
}

func TestHelpHidesLifecycleCommands(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"help"}} {
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		if code := RunCLI(args, stdout, stderr); code != 0 {
			t.Fatalf("%v exit code = %d, stderr = %s", args, code, stderr.String())
		}
		for _, hidden := range []string{"email init", "email start", "email stop", "\"init\"", "\"start\"", "\"stop\""} {
			if strings.Contains(stdout.String(), hidden) {
				t.Fatalf("%v unexpectedly mentions %q\n%s", args, hidden, stdout.String())
			}
		}
	}
}

func TestCLIConnectClientSupportsPrefixedConnectCommands(t *testing.T) {
	runner := &stubRunner{}
	client := &CLIConnectClient{
		Binary: "/tmp/integration",
		Prefix: []string{"connect"},
		Runner: runner,
	}

	runner.outputs = append(runner.outputs, []byte(`{"id":1,"name":"email","meta":"{}","stream":true,"callback":"./email","agentId":"A","chatId":"","model":"OpenAI","thinking":false,"createdAt":"2026-05-06T00:00:00Z","updatedAt":"2026-05-06T00:00:00Z"}`))
	if _, err := client.GetMeta(context.Background(), "email"); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 runner call, got %d", len(runner.calls))
	}
	if got := strings.Join(runner.calls[0], " "); !strings.Contains(got, "/tmp/integration connect meta-get --key email") {
		t.Fatalf("unexpected call: %s", got)
	}
}

func TestCLIConnectClientAddRequestIncludesSchemaFlag(t *testing.T) {
	runner := &stubRunner{}
	client := &CLIConnectClient{
		Binary: "connect",
		Runner: runner,
	}

	runner.outputs = append(runner.outputs, []byte(`{"id":1,"name":"email","request":"hello","responseSchema":"{\"type\":\"object\"}"}`))
	_, err := client.AddRequest(context.Background(), connectsvc.RequestInput{
		Key:            "email",
		Content:        "hello",
		ResponseSchema: `{"type":"object"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 runner call, got %d", len(runner.calls))
	}
	got := strings.Join(runner.calls[0], " ")
	if !strings.Contains(got, "--schema {\"type\":\"object\"}") {
		t.Fatalf("schema flag missing from call: %s", got)
	}
}

func TestCLIConnectClientAddRequestErrorUsesDecodedLogValues(t *testing.T) {
	runner := &stubRunner{}
	client := &CLIConnectClient{
		Binary: "connect",
		Runner: runner,
	}

	_, err := client.AddRequest(context.Background(), connectsvc.RequestInput{
		Key:       "email",
		Content:   "=?GBK?B?KM7e1vfM4ik=?=\n今天天气如何？",
		Original:  `{"message":{"subject":"=?GBK?B?KM7e1vfM4ik=?=","content":"=?GBK?B?KM7e1vfM4ik=?=\n今天天气如何？"}}`,
		CreatedAt: "1779371360",
	})
	if err == nil {
		t.Fatal("expected add-request failure")
	}
	if strings.Contains(err.Error(), "=?GBK?B?KM7e1vfM4ik=?=") {
		t.Fatalf("error should not leak encoded subject: %v", err)
	}
	if !strings.Contains(err.Error(), "(无主题)") {
		t.Fatalf("error should contain decoded subject: %v", err)
	}
}

func TestCLIConnectClientGetMetaUsesKeyVerbatim(t *testing.T) {
	runner := &stubRunner{}
	client := &CLIConnectClient{
		Binary: "connect",
		Runner: runner,
	}

	runner.outputs = append(runner.outputs, []byte(`{"id":1,"name":"邮件","meta":"{}","stream":true,"callback":"./email","agentId":"A","chatId":"","model":"OpenAI","thinking":false,"createdAt":"2026-05-06T00:00:00Z","updatedAt":"2026-05-06T00:00:00Z"}`))
	if _, err := client.GetMeta(context.Background(), DefaultName); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 runner call, got %d", len(runner.calls))
	}
	if got := strings.Join(runner.calls[0], " "); !strings.Contains(got, "meta-get --key email") {
		t.Fatalf("unexpected call: %s", got)
	}
}

func TestCLIConnectClientGetMetaAcceptsObjectMeta(t *testing.T) {
	runner := &stubRunner{}
	client := &CLIConnectClient{
		Binary: "connect",
		Runner: runner,
	}

	runner.outputs = append(runner.outputs, []byte(`{"key":"email","name":"邮件","meta":{"email":"demo@example.com","email_pop3":"pop.example.com:995","email_smtp":"smtp.example.com:465","email_password":"secret"},"stream":true,"callback":"./email","agentId":"A","chatId":"","model":"OpenAI","thinking":false,"createdAt":"2026-05-06T00:00:00Z","updatedAt":"2026-05-06T00:00:00Z"}`))
	meta, err := client.GetMeta(context.Background(), DefaultName)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(meta.Meta, `"email":"demo@example.com"`) {
		t.Fatalf("meta.Meta = %q", meta.Meta)
	}
}

func TestCLIConnectClientGetMetaDoesNotFallbackToLegacyKey(t *testing.T) {
	runner := &stubRunner{}
	client := &CLIConnectClient{
		Binary: "connect",
		Runner: runner,
	}

	if _, err := client.GetMeta(context.Background(), "email_smtp"); err == nil {
		t.Fatal("expected get meta failure")
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 runner call, got %d", len(runner.calls))
	}
	if got := strings.Join(runner.calls[0], " "); !strings.Contains(got, "meta-get --key email_smtp") {
		t.Fatalf("unexpected call: %s", got)
	}
}

func TestBuildServiceAddsConnectPrefixForIntegrationBinary(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	svc, err := buildService(map[string]string{
		"connect-bin": "/tmp/integration",
	}, logger, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	client, ok := svc.client.(*CLIConnectClient)
	if !ok {
		t.Fatalf("unexpected client type: %T", svc.client)
	}
	if client.Binary != "/tmp/integration" {
		t.Fatalf("binary = %q", client.Binary)
	}
	if strings.Join(client.Prefix, " ") != "connect" {
		t.Fatalf("prefix = %v, want [connect]", client.Prefix)
	}
}

func TestServiceRunPushesUnreadMailToConnect(t *testing.T) {
	client := &stubConnectClient{
		meta: &connectsvc.Meta{
			Name: DefaultName,
			Meta: `{"email":"demo@example.com","email_pop3":"pop.example.com:995","email_smtp":"smtp.example.com:465","email_password":"secret","email_whitelist":"sender@example.com","scanSeconds":1}`,
		},
	}
	fetcher := &stubFetcher{
		mails: []FetchedMail{
			{
				UID:         "100",
				MessageID:   "<msg-1@example.com>",
				DateHeader:  "Wed, 06 May 2026 10:00:00 +0800",
				ReceivedAt:  time.Date(2026, 5, 6, 10, 0, 0, 0, time.FixedZone("CST", 8*3600)),
				Subject:     "测试邮件",
				From:        "sender@example.com",
				FromAddress: "sender@example.com",
				TextBody:    "你好，世界",
				Artifacts: []string{
					"/tmp/email_artifacts/image_key_inline-1.png",
					"/tmp/email_artifacts/file_key_attach-1.pdf",
				},
				RawJSON: `{"headers":[{"name":"Subject","value":"测试邮件"}],"content":"你好，世界"}`,
			},
		},
	}
	loggerOut := &bytes.Buffer{}
	messageLog := &bytes.Buffer{}
	stateFile := filepath.Join(t.TempDir(), "email.state.json")
	svc, err := NewService(Options{
		Client:     client,
		Fetcher:    fetcher,
		Logger:     log.New(loggerOut, "", 0),
		MessageLog: messageLog,
		StateFile:  stateFile,
		ScanEvery:  20 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	if err := svc.Run(ctx); err != nil {
		t.Fatal(err)
	}

	requests := client.Requests()
	if len(requests) == 0 {
		t.Fatal("expected pushed requests")
	}
	got := requests[0]
	if got.Name != DefaultName {
		t.Fatalf("request name = %q", got.Name)
	}
	if got.ExternalID != "<msg-1@example.com>" {
		t.Fatalf("external id = %q", got.ExternalID)
	}
	if strings.TrimSpace(got.ResponseSchema) == "" {
		t.Fatalf("response schema should not be empty: %+v", got)
	}
	var schema map[string]any
	if err := json.Unmarshal([]byte(got.ResponseSchema), &schema); err != nil {
		t.Fatalf("decode response schema: %v raw=%s", err, got.ResponseSchema)
	}
	if schema["type"] != "object" {
		t.Fatalf("schema type = %#v, want object", schema["type"])
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema properties type = %T", schema["properties"])
	}
	content, ok := properties["content"].(map[string]any)
	if !ok {
		t.Fatalf("schema content type = %T", properties["content"])
	}
	if content["description"] != "The content is presented in `HTML` format, and the file paths have been replaced with filenames." {
		t.Fatalf("content description = %#v", content["description"])
	}
	if !strings.Contains(got.Request, "测试邮件") {
		t.Fatalf("request should include subject = %q", got.Request)
	}
	if !strings.Contains(got.Request, "你好，世界") {
		t.Fatalf("request content = %q", got.Request)
	}
	if !strings.Contains(got.Request, "[image]/tmp/email_artifacts/image_key_inline-1.png") {
		t.Fatalf("request should include normalized image path: %q", got.Request)
	}
	if !strings.Contains(got.Request, "[file]/tmp/email_artifacts/file_key_attach-1.pdf") {
		t.Fatalf("request should include normalized file path: %q", got.Request)
	}
	if got.Artifacts != "/tmp/email_artifacts/image_key_inline-1.png,/tmp/email_artifacts/file_key_attach-1.pdf" {
		t.Fatalf("artifacts = %q", got.Artifacts)
	}
	if !strings.Contains(got.RawRequest, `"source":"email"`) {
		t.Fatalf("raw request = %q", got.RawRequest)
	}
	if !strings.Contains(got.RawRequest, `\"headers\"`) {
		t.Fatalf("raw request should contain encoded header payload: %q", got.RawRequest)
	}
	if !strings.Contains(got.RawRequest, `"subject":"测试邮件"`) {
		t.Fatalf("raw request should contain decoded subject field: %q", got.RawRequest)
	}
	if !strings.Contains(messageLog.String(), "你好，世界") {
		t.Fatalf("message log = %s", messageLog.String())
	}
	if !strings.Contains(messageLog.String(), `做了：收到了新邮件`) {
		t.Fatalf("message log should contain structured stage: %s", messageLog.String())
	}
	if !strings.Contains(messageLog.String(), `主题=测试邮件`) {
		t.Fatalf("message log should contain structured subject: %s", messageLog.String())
	}
	if !strings.Contains(messageLog.String(), `发件人=sender@example.com`) {
		t.Fatalf("message log should contain structured from: %s", messageLog.String())
	}
	if !strings.Contains(messageLog.String(), `邮件编号=<msg-1@example.com>`) {
		t.Fatalf("message log should contain structured message id: %s", messageLog.String())
	}
}

func TestServiceRunStopsAfterThreeConsecutiveScanFailures(t *testing.T) {
	client := &stubConnectClient{
		meta: &connectsvc.Meta{
			Name: DefaultName,
			Meta: `{"email":"demo@example.com","email_pop3":"pop.example.com:995","email_smtp":"smtp.example.com:465","email_password":"secret","email_whitelist":"","scanSeconds":1}`,
		},
	}
	messageLog := &bytes.Buffer{}
	loggerOut := &bytes.Buffer{}
	svc, err := NewService(Options{
		Client:     client,
		Fetcher:    &stubFetcher{err: errors.New("pop3 down")},
		Logger:     log.New(loggerOut, "", 0),
		MessageLog: messageLog,
		StateFile:  filepath.Join(t.TempDir(), "email.state.json"),
		ScanEvery:  10 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = svc.Run(context.Background())
	if err == nil {
		t.Fatal("expected service failure")
	}
	if !strings.Contains(err.Error(), "scan failed after 3 attempts") {
		t.Fatalf("unexpected err: %v", err)
	}
	logText := messageLog.String()
	if !strings.Contains(logText, `做了：邮件插件运行出错，正在自动重试`) {
		t.Fatalf("missing retry log: %s", logText)
	}
	if !strings.Contains(logText, `做了：邮件插件连续运行失败，已停止自动重试`) {
		t.Fatalf("missing terminate log: %s", logText)
	}
}

func TestServiceRunSkipsNonWhitelistedMailButAdvancesTimeline(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "email.state.json")
	client := &stubConnectClient{
		meta: &connectsvc.Meta{
			Name: DefaultName,
			Meta: `{"email":"demo@example.com","email_pop3":"pop.example.com:995","email_smtp":"smtp.example.com:465","email_password":"secret","email_whitelist":"allow@example.com","scanSeconds":1}`,
		},
	}
	fetcher := &stubFetcher{
		mails: []FetchedMail{
			{
				UID:         "100",
				MessageID:   "<skip-1@example.com>",
				DateHeader:  "Wed, 06 May 2026 10:00:00 +0800",
				ReceivedAt:  time.Date(2026, 5, 6, 10, 0, 0, 0, time.FixedZone("CST", 8*3600)),
				From:        "deny@example.com",
				FromAddress: "deny@example.com",
				TextBody:    "不会进入 connect",
			},
		},
	}
	svc, err := NewService(Options{
		Client:     client,
		Fetcher:    fetcher,
		Logger:     log.New(io.Discard, "", 0),
		MessageLog: &bytes.Buffer{},
		StateFile:  stateFile,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.scanOnce(context.Background(), Config{
		Email:          "demo@example.com",
		EmailPOP3:      "pop.example.com:995",
		EmailSMTP:      "smtp.example.com:465",
		EmailPassword:  "secret",
		EmailWhitelist: "allow@example.com",
	}); err != nil {
		t.Fatal(err)
	}
	if len(client.Requests()) != 0 {
		t.Fatalf("expected no pushed requests, got %+v", client.Requests())
	}
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"lastTimelineUnix"`) {
		t.Fatalf("state file should persist timeline: %s", string(data))
	}
}

func TestServiceRunAllowsAllSendersWhenWhitelistEmpty(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "email.state.json")
	client := &stubConnectClient{
		meta: &connectsvc.Meta{
			Name: DefaultName,
			Meta: `{"email":"demo@example.com","email_pop3":"pop.example.com:995","email_smtp":"smtp.example.com:465","email_password":"secret","email_whitelist":"","scanSeconds":1}`,
		},
	}
	fetcher := &stubFetcher{
		mails: []FetchedMail{
			{
				UID:         "200",
				MessageID:   "<allow-all@example.com>",
				DateHeader:  "Wed, 06 May 2026 11:00:00 +0800",
				ReceivedAt:  time.Date(2026, 5, 6, 11, 0, 0, 0, time.FixedZone("CST", 8*3600)),
				From:        "random@example.com",
				FromAddress: "random@example.com",
				TextBody:    "白名单为空也应放行",
				RawJSON:     `{"headers":[],"content":"白名单为空也应放行"}`,
			},
		},
	}
	svc, err := NewService(Options{
		Client:     client,
		Fetcher:    fetcher,
		Logger:     log.New(io.Discard, "", 0),
		MessageLog: &bytes.Buffer{},
		StateFile:  stateFile,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.scanOnce(context.Background(), Config{
		Email:          "demo@example.com",
		EmailPOP3:      "pop.example.com:995",
		EmailSMTP:      "smtp.example.com:465",
		EmailPassword:  "secret",
		EmailWhitelist: "",
	}); err != nil {
		t.Fatal(err)
	}
	if len(client.Requests()) != 1 {
		t.Fatalf("expected request to be pushed when whitelist empty, got %+v", client.Requests())
	}
	var snapshot connectsvc.MessageSnapshot
	if err := json.Unmarshal([]byte(client.Requests()[0].MessageSnapshot), &snapshot); err != nil {
		t.Fatalf("decode message snapshot: %v raw=%s", err, client.Requests()[0].MessageSnapshot)
	}
	if snapshot.Source != DefaultName || len(snapshot.Messages) != 1 {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	item := snapshot.Messages[0]
	if item.MessageID != "<allow-all@example.com>" || item.SenderID != "random@example.com" || item.MessageType != "text" || item.Content != "白名单为空也应放行" {
		t.Fatalf("snapshot item = %+v", item)
	}
}

func TestBuildEmailMessageSnapshotUsesDateAndNormalizesSender(t *testing.T) {
	date := "Wed, 06 May 2026 11:00:00 +0800"
	receivedAt := time.Date(2026, time.May, 6, 12, 0, 0, 0, time.FixedZone("CST", 8*3600))
	raw := buildEmailMessageSnapshot(FetchedMail{
		MessageID:  "<mail@example.com>",
		DateHeader: date,
		ReceivedAt: receivedAt,
		From:       "Alice <ALICE@Example.com>",
		Subject:    "退款申请",
		TextBody:   "已处理",
	}, receivedAt.Add(time.Hour))
	var snapshot connectsvc.MessageSnapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		t.Fatalf("decode snapshot: %v raw=%s", err, raw)
	}
	if len(snapshot.Messages) != 1 {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	item := snapshot.Messages[0]
	if item.SenderID != "alice@example.com" || item.Content != "退款申请\n已处理" || item.SentAt != time.Date(2026, time.May, 6, 11, 0, 0, 0, time.FixedZone("CST", 8*3600)).UnixMilli() {
		t.Fatalf("snapshot item = %+v", item)
	}
}

func TestBuildEmailMessageSnapshotFallsBackToReceivedAt(t *testing.T) {
	receivedAt := time.Date(2026, time.May, 6, 12, 0, 0, 0, time.UTC)
	raw := buildEmailMessageSnapshot(FetchedMail{
		MessageID:  "<mail@example.com>",
		DateHeader: "not-a-date",
		ReceivedAt: receivedAt,
		From:       "alice@example.com",
		TextBody:   "正文",
	}, time.Time{})
	var snapshot connectsvc.MessageSnapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		t.Fatalf("decode snapshot: %v raw=%s", err, raw)
	}
	if got := snapshot.Messages[0].SentAt; got != receivedAt.UnixMilli() {
		t.Fatalf("sentAt = %d, want %d", got, receivedAt.UnixMilli())
	}
}

func TestNewServiceInitializesTimelineToThirtyMinutesBeforeStartup(t *testing.T) {
	now := time.Date(2026, 5, 6, 15, 0, 0, 0, time.FixedZone("CST", 8*3600))
	stateFile := filepath.Join(t.TempDir(), "email.state.json")
	fetcher := &capturingFetcher{}

	svc, err := NewService(Options{
		Client: &stubConnectClient{
			meta: &connectsvc.Meta{
				Name: DefaultName,
				Meta: `{"email":"demo@example.com","email_pop3":"pop.example.com:995","email_smtp":"smtp.example.com:465","email_password":"secret","email_whitelist":""}`,
			},
		},
		Fetcher:    fetcher,
		Logger:     log.New(io.Discard, "", 0),
		MessageLog: &bytes.Buffer{},
		StateFile:  stateFile,
		Now:        func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}

	want := now.Add(-30 * time.Minute)
	got := svc.timeline()
	if !got.Equal(want) {
		t.Fatalf("timeline = %s, want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
	}

	if err := svc.scanOnce(context.Background(), Config{
		Email:          "demo@example.com",
		EmailPOP3:      "pop.example.com:995",
		EmailSMTP:      "smtp.example.com:465",
		EmailPassword:  "secret",
		EmailWhitelist: "",
	}); err != nil {
		t.Fatal(err)
	}
	if !fetcher.lastSince.Equal(want) {
		t.Fatalf("fetch since = %s, want %s", fetcher.lastSince.Format(time.RFC3339), want.Format(time.RFC3339))
	}

	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"lastTimelineUnix"`) {
		t.Fatalf("state file should persist initial timeline: %s", string(data))
	}
}

func TestServiceStateDeduplicatesProcessedMessageID(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "email.state.json")
	logger := log.New(io.Discard, "", 0)
	clientA := &stubConnectClient{
		meta: &connectsvc.Meta{
			Name: DefaultName,
			Meta: `{"email":"demo@example.com","email_pop3":"pop.example.com:995","email_smtp":"smtp.example.com:465","email_password":"secret","email_whitelist":"sender@example.com","scanSeconds":1}`,
		},
	}
	fetcher := &stubFetcher{
		mails: []FetchedMail{
			{
				UID:         "100",
				MessageID:   "<dup-1@example.com>",
				DateHeader:  "Wed, 06 May 2026 10:00:00 +0800",
				ReceivedAt:  time.Date(2026, 5, 6, 10, 0, 0, 0, time.FixedZone("CST", 8*3600)),
				From:        "sender@example.com",
				FromAddress: "sender@example.com",
				TextBody:    "第一次投递",
				RawJSON:     `{"headers":[],"content":"第一次投递"}`,
			},
		},
	}
	svcA, err := NewService(Options{
		Client:     clientA,
		Fetcher:    fetcher,
		Logger:     logger,
		MessageLog: &bytes.Buffer{},
		StateFile:  stateFile,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svcA.scanOnce(context.Background(), Config{
		Email:          "demo@example.com",
		EmailPOP3:      "pop.example.com:995",
		EmailSMTP:      "smtp.example.com:465",
		EmailPassword:  "secret",
		EmailWhitelist: "sender@example.com",
	}); err != nil {
		t.Fatal(err)
	}
	if len(clientA.Requests()) != 1 {
		t.Fatalf("expected first request to push once, got %d", len(clientA.Requests()))
	}

	clientB := &stubConnectClient{
		meta: clientA.meta,
	}
	svcB, err := NewService(Options{
		Client:     clientB,
		Fetcher:    fetcher,
		Logger:     logger,
		MessageLog: &bytes.Buffer{},
		StateFile:  stateFile,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svcB.scanOnce(context.Background(), Config{
		Email:          "demo@example.com",
		EmailPOP3:      "pop.example.com:995",
		EmailSMTP:      "smtp.example.com:465",
		EmailPassword:  "secret",
		EmailWhitelist: "sender@example.com",
	}); err != nil {
		t.Fatal(err)
	}
	if len(clientB.Requests()) != 0 {
		t.Fatalf("expected duplicate message to be skipped, got %+v", clientB.Requests())
	}
}

func TestServiceStateDeduplicatesProcessedUID(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "email.state.json")
	logger := log.New(io.Discard, "", 0)
	clientA := &stubConnectClient{
		meta: &connectsvc.Meta{
			Name: DefaultName,
			Meta: `{"email":"demo@example.com","email_pop3":"pop.example.com:995","email_smtp":"smtp.example.com:465","email_password":"secret","email_whitelist":"sender@example.com","scanSeconds":1}`,
		},
	}
	fetcher := &stubFetcher{
		mails: []FetchedMail{
			{
				UID:         "uid-dup-1",
				MessageID:   "",
				DateHeader:  "Wed, 06 May 2026 10:00:00 +0800",
				ReceivedAt:  time.Date(2026, 5, 6, 10, 0, 0, 0, time.FixedZone("CST", 8*3600)),
				From:        "sender@example.com",
				FromAddress: "sender@example.com",
				TextBody:    "第一次投递",
				RawJSON:     `{"headers":[],"content":"第一次投递"}`,
			},
		},
	}
	svcA, err := NewService(Options{
		Client:     clientA,
		Fetcher:    fetcher,
		Logger:     logger,
		MessageLog: &bytes.Buffer{},
		StateFile:  stateFile,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svcA.scanOnce(context.Background(), Config{
		Email:          "demo@example.com",
		EmailPOP3:      "pop.example.com:995",
		EmailSMTP:      "smtp.example.com:465",
		EmailPassword:  "secret",
		EmailWhitelist: "sender@example.com",
	}); err != nil {
		t.Fatal(err)
	}
	if len(clientA.Requests()) != 1 {
		t.Fatalf("expected first request to push once, got %d", len(clientA.Requests()))
	}

	clientB := &stubConnectClient{
		meta: clientA.meta,
	}
	svcB, err := NewService(Options{
		Client:     clientB,
		Fetcher:    fetcher,
		Logger:     logger,
		MessageLog: &bytes.Buffer{},
		StateFile:  stateFile,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svcB.scanOnce(context.Background(), Config{
		Email:          "demo@example.com",
		EmailPOP3:      "pop.example.com:995",
		EmailSMTP:      "smtp.example.com:465",
		EmailPassword:  "secret",
		EmailWhitelist: "sender@example.com",
	}); err != nil {
		t.Fatal(err)
	}
	if len(clientB.Requests()) != 0 {
		t.Fatalf("expected duplicate uid to be skipped, got %+v", clientB.Requests())
	}
}

func TestParseFetchedMailExtractsAttachmentsAndNestedMessage(t *testing.T) {
	dir := t.TempDir()
	raw := "From: sender@example.com\r\n" +
		"To: receiver@example.com\r\n" +
		"Subject: =?UTF-8?B?5rWL6K+V6YKu5Lu2?=\r\n" +
		"Message-ID: <root@example.com>\r\n" +
		"Date: Wed, 06 May 2026 10:00:00 +0800\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: multipart/mixed; boundary=\"root\"\r\n" +
		"\r\n" +
		"--root\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		"正文内容\r\n" +
		"--root\r\n" +
		"Content-Type: image/png\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"Content-ID: <inline-1>\r\n" +
		"\r\n" +
		"aGVsbG8=\r\n" +
		"--root\r\n" +
		"Content-Type: application/pdf\r\n" +
		"Content-Disposition: attachment; filename=\"report.pdf\"\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"\r\n" +
		"aGVsbG8=\r\n" +
		"--root\r\n" +
		"Content-Type: message/rfc822\r\n" +
		"\r\n" +
		"From: nested@example.com\r\n" +
		"To: receiver@example.com\r\n" +
		"Subject: nested\r\n" +
		"Message-ID: <nested@example.com>\r\n" +
		"Date: Wed, 06 May 2026 10:01:00 +0800\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		"附件邮件正文\r\n" +
		"--root--\r\n"

	item, err := parseFetchedMail(RawMail{UID: "7", Raw: []byte(raw)}, dir)
	if err != nil {
		t.Fatal(err)
	}
	if item.Subject != "测试邮件" {
		t.Fatalf("subject = %q", item.Subject)
	}
	if !strings.Contains(item.TextBody, "正文内容") {
		t.Fatalf("text body = %q", item.TextBody)
	}
	if !strings.Contains(item.TextBody, "附件邮件正文") {
		t.Fatalf("nested text body = %q", item.TextBody)
	}
	if len(item.Artifacts) != 2 {
		t.Fatalf("artifacts = %+v", item.Artifacts)
	}
	for _, path := range item.Artifacts {
		if !filepath.IsAbs(path) {
			t.Fatalf("artifact path should be absolute: %s", path)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("artifact missing: %s err=%v", path, err)
		}
	}
}

func TestNormalizeMailContentFallsBackToHTML(t *testing.T) {
	content := normalizeMailContent(FetchedMail{
		HTMLBody: `<div>hello<br/>world</div>`,
	})
	if content != "hello\nworld" {
		t.Fatalf("content = %q", content)
	}
}

func TestNormalizeMailContentIncludesSubjectBeforeBodyAndArtifacts(t *testing.T) {
	content := normalizeMailContent(FetchedMail{
		Subject:  "测试标题",
		TextBody: "正文内容",
		Artifacts: []string{
			"/tmp/email_artifacts/image_key_inline-1.png",
			"/tmp/email_artifacts/file_key_attach-1.pdf",
		},
	})
	want := "测试标题\n正文内容\n[image]/tmp/email_artifacts/image_key_inline-1.png\n[file]/tmp/email_artifacts/file_key_attach-1.pdf"
	if content != want {
		t.Fatalf("content = %q, want %q", content, want)
	}
}

func TestNormalizeMailContentDecodesEncodedSubject(t *testing.T) {
	content := normalizeMailContent(FetchedMail{
		Subject:  "=?GBK?B?KM7e1vfM4ik=?=",
		TextBody: "杭州景点",
	})
	want := "(无主题)\n杭州景点"
	if content != want {
		t.Fatalf("content = %q, want %q", content, want)
	}
}

func TestBuildMailMessageDecodesEncodedSubject(t *testing.T) {
	got := buildMailMessage(FetchedMail{
		UID:       "1",
		MessageID: "<msg-1@example.com>",
		Subject:   "=?GBK?B?KM7e1vfM4ik=?=",
		TextBody:  "今天天气如何？",
	})
	if got.Subject != "(无主题)" {
		t.Fatalf("subject = %q", got.Subject)
	}
	if got.Content != "(无主题)\n今天天气如何？" {
		t.Fatalf("content = %q", got.Content)
	}
}

func TestCompactMailLogPrefersDecodedSubject(t *testing.T) {
	got := compactMailLog(MailMessage{
		Subject: "=?GBK?B?vfHM7Mzsxvg=?=",
		Content: "=?GBK?B?vfHM7Mzsxvg=?=",
	})
	if got != "今天天气" {
		t.Fatalf("compact log = %q", got)
	}
}

func TestCompactMailLogFallsBackToDecodedSubjectFromRawJSON(t *testing.T) {
	got := compactMailLog(MailMessage{
		Raw: `{"headers":[{"name":"Subject","value":"=?GBK?B?vfHM7Mzsxvg=?="}],"content":"Subject: =?GBK?B?vfHM7Mzsxvg=?="}`,
	})
	if got != "今天天气" {
		t.Fatalf("compact log = %q", got)
	}
}

func TestStructuredMailLogIncludesDecodedFields(t *testing.T) {
	got := structuredMailLog("mail-received", MailMessage{
		Subject:   "=?GBK?B?vfHM7Mzsxvg=?=",
		From:      "=?UTF-8?B?5rWL6K+V?= <sender@example.com>",
		MessageID: "<msg-1@example.com>",
		Content:   "=?GBK?B?vfHM7Mzsxvg=?=",
	})
	if !strings.Contains(got, `stage=mail-received`) {
		t.Fatalf("structured log stage = %q", got)
	}
	if !strings.Contains(got, `subject="今天天气"`) {
		t.Fatalf("structured log subject = %q", got)
	}
	if !strings.Contains(got, `from="测试 <sender@example.com>"`) {
		t.Fatalf("structured log from = %q", got)
	}
	if !strings.Contains(got, `message_id="<msg-1@example.com>"`) {
		t.Fatalf("structured log message id = %q", got)
	}
}

func TestParseFetchedMailDecodesNonUTF8BodyToUTF8(t *testing.T) {
	dir := t.TempDir()
	raw := append([]byte("From: sender@example.com\r\n"+
		"To: receiver@example.com\r\n"+
		"Subject: =?GBK?B?suLK1A==?=\r\n"+
		"Message-ID: <gbk@example.com>\r\n"+
		"Date: Wed, 06 May 2026 10:00:00 +0800\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/plain; charset=GBK\r\n"+
		"\r\n"), []byte{0xC4, 0xE3, 0xBA, 0xC3}...)

	item, err := parseFetchedMail(RawMail{UID: "8", Raw: raw}, dir)
	if err != nil {
		t.Fatal(err)
	}
	if item.Subject != "测试" {
		t.Fatalf("subject = %q", item.Subject)
	}
	if item.TextBody != "你好" {
		t.Fatalf("text body = %q", item.TextBody)
	}
	if !strings.Contains(item.RawJSON, `"content":"你好"`) {
		t.Fatalf("raw json = %q", item.RawJSON)
	}
}

func TestParseFetchedMailPrefersReceivedHeaderTime(t *testing.T) {
	dir := t.TempDir()
	raw := "Received: from mx.example.com by server.example.com; Thu, 12 Jun 2026 15:57:00 +0800\r\n" +
		"From: sender@example.com\r\n" +
		"To: receiver@example.com\r\n" +
		"Subject: Delayed date\r\n" +
		"Message-ID: <received@example.com>\r\n" +
		"Date: Thu, 12 Jun 2026 09:00:00 +0800\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		"正文\r\n"

	item, err := parseFetchedMail(RawMail{UID: "9", Raw: []byte(raw)}, dir)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 6, 12, 15, 57, 0, 0, time.FixedZone("CST", 8*3600))
	if !item.ReceivedAt.Equal(want) {
		t.Fatalf("receivedAt = %s, want %s", item.ReceivedAt.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

func TestNormalizePOP3AddressDefaultsTo995(t *testing.T) {
	host, port, err := normalizePOP3Address("pop.126.com")
	if err != nil {
		t.Fatal(err)
	}
	if host != "pop.126.com" || port != 995 {
		t.Fatalf("addr = %s:%d", host, port)
	}
	host, port, err = normalizePOP3Address("pop.126.com:110")
	if err != nil {
		t.Fatal(err)
	}
	if host != "pop.126.com" || port != 110 {
		t.Fatalf("addr = %s:%d", host, port)
	}
}

func TestAuthenticatePOP3FallsBackToLocalPartOnInvalidUsername(t *testing.T) {
	conn := &stubPOP3Session{
		authErrs: map[string]error{
			"demo@example.com": errors.New("Username not valid"),
		},
	}

	err := authenticatePOP3(conn, "demo@example.com", "secret", "pop.example.com")
	if err != nil {
		t.Fatalf("authenticatePOP3 returned err = %v", err)
	}
	if got := strings.Join(conn.authUsers, ","); got != "demo@example.com,demo" {
		t.Fatalf("auth users = %q", got)
	}
}

func TestAuthenticatePOP3ReturnsWrappedErrorForPasswordFailure(t *testing.T) {
	conn := &stubPOP3Session{
		authErrs: map[string]error{
			"demo@example.com": errors.New("authentication failed"),
		},
	}

	err := authenticatePOP3(conn, "demo@example.com", "secret", "pop.example.com")
	if err == nil {
		t.Fatal("authenticatePOP3 unexpectedly succeeded")
	}
	if got := strings.Join(conn.authUsers, ","); got != "demo@example.com" {
		t.Fatalf("auth users = %q", got)
	}
	text := err.Error()
	if !strings.Contains(text, "server=pop.example.com") {
		t.Fatalf("wrapped err missing server: %v", err)
	}
	if !strings.Contains(text, "account=d***@example.com") {
		t.Fatalf("wrapped err missing masked account: %v", err)
	}
	if !strings.Contains(text, "authentication failed") {
		t.Fatalf("wrapped err missing original message: %v", err)
	}
}

func TestDecodePossiblyEncodedTextDecodesGBKPOP3Error(t *testing.T) {
	raw := []byte{
		0xb5, 0xc7, 0xc2, 0xbc, 0xcc, 0xab, 0xc6, 0xb5, 0xb7, 0xb1, 0x21,
		0xc7, 0xeb, 0xbc, 0xec, 0xb2, 0xe9, 0xc4, 0xfa, 0xb5, 0xc4,
		0x6f, 0x75, 0x74, 0x6c, 0x6f, 0x6f, 0x6b, 0x2c, 0x66, 0x6f, 0x78, 0x6d, 0x61, 0x69, 0x6c,
		0xbb, 0xf2, 0xd5, 0xdf, 0xc6, 0xe4, 0xcb, 0xfc, 0xd7, 0xd4, 0xb6, 0xaf, 0xbc, 0xec, 0xb2, 0xe2,
		0xd3, 0xca, 0xcf, 0xe4, 0xb5, 0xc4, 0xb9, 0xa4, 0xbe, 0xdf, 0x28, 0xc0, 0xfd, 0xc8, 0xe7, 0xcd, 0xf8, 0xd2, 0xd7,
		0x70, 0x6f, 0x70, 0x6f, 0xb5, 0xc8, 0xc1, 0xc4, 0xcc, 0xec, 0xb9, 0xa4, 0xbe, 0xdf, 0x29, 0x2c,
		0xbd, 0xab, 0xbc, 0xec, 0xb2, 0xe2, 0xb5, 0xc4, 0xca, 0xb1, 0xbc, 0xe4, 0xbc, 0xe4, 0xb8, 0xf4, 0xb5, 0xf7, 0xb4, 0xf3, 0xd2, 0xbb, 0xd0, 0xa9, 0xa3, 0xac,
		0xc0, 0xfd, 0xc8, 0xe7, 0x35, 0xb7, 0xd6, 0xd6, 0xd3, 0xb1, 0xe0, 0xb2, 0xe2, 0xd2, 0xbb, 0xb4, 0xce, 0x2e,
	}
	got := decodePossiblyEncodedText(string(raw))
	if !strings.Contains(got, "登录太频繁") {
		t.Fatalf("decoded text should contain login-rate-limit hint, got %q", got)
	}
	if !strings.Contains(got, "outlook,foxmail") {
		t.Fatalf("decoded text should keep ascii tool names, got %q", got)
	}
	if !strings.Contains(got, "例如5分钟") {
		t.Fatalf("decoded text should keep timing suggestion, got %q", got)
	}
	if strings.Contains(got, "��") {
		t.Fatalf("decoded text should not contain replacement mojibake, got %q", got)
	}
}

func TestDecodePossiblyEncodedTextKeepsEnglishText(t *testing.T) {
	got := decodePossiblyEncodedText("Username not valid")
	if got != "Username not valid" {
		t.Fatalf("decoded text = %q", got)
	}
}

func TestExtractReplyEnvelope(t *testing.T) {
	reply, err := extractReplyEnvelope(latestEmailConnectRequestMessageJSON("<origin@example.com>", "测试主题", "Sender <sender@example.com>", "<root@example.com>"))
	if err != nil {
		t.Fatal(err)
	}
	if len(reply.To) != 1 || reply.To[0] != "sender@example.com" {
		t.Fatalf("unexpected reply to: %+v", reply)
	}
	if reply.Subject != "Re: 测试主题" {
		t.Fatalf("unexpected subject: %q", reply.Subject)
	}
	if reply.MessageID != "<origin@example.com>" {
		t.Fatalf("unexpected message id: %q", reply.MessageID)
	}
	if got := strings.Join(reply.References, " "); got != "<root@example.com> <origin@example.com>" {
		t.Fatalf("unexpected references: %q", got)
	}
}

func TestExtractReplyEnvelopeFromConnectRawRequest(t *testing.T) {
	reply, err := extractReplyEnvelope(latestEmailConnectRequestMessageJSON("<origin@example.com>", "测试主题", "Sender <sender@example.com>", "<root@example.com>"))
	if err != nil {
		t.Fatal(err)
	}
	if len(reply.To) != 1 || reply.To[0] != "sender@example.com" {
		t.Fatalf("unexpected reply to: %+v", reply)
	}
	if reply.MessageID != "<origin@example.com>" {
		t.Fatalf("unexpected message id: %q", reply.MessageID)
	}
	if got := strings.Join(reply.References, " "); got != "<root@example.com> <origin@example.com>" {
		t.Fatalf("unexpected references: %q", got)
	}
}

func TestExtractReplyEnvelopeNormalizesInReplyToAndReferences(t *testing.T) {
	reply, err := extractReplyEnvelope(latestEmailConnectRequestMessageJSONWithRawHeaders("<origin@example.com>", "测试主题", "Sender <sender@example.com>", []map[string]string{
		{"name": "From", "value": "Sender <sender@example.com>"},
		{"name": "Subject", "value": "测试主题"},
		{"name": "Message-ID", "value": "thread parent <origin@example.com>"},
		{"name": "In-Reply-To", "value": "reply parent <mid@example.com>"},
		{"name": "References", "value": "refs <root@example.com> <mid@example.com>"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if reply.MessageID != "<origin@example.com>" {
		t.Fatalf("unexpected message id: %q", reply.MessageID)
	}
	if got := strings.Join(reply.References, " "); got != "<mid@example.com> <root@example.com> <origin@example.com>" {
		t.Fatalf("unexpected references: %q", got)
	}
}

func TestBuildSMTPMessageIncludesReplyHeadersAndAttachments(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "demo.txt")
	if err := os.WriteFile(filePath, []byte("demo attachment"), 0o644); err != nil {
		t.Fatal(err)
	}
	attachment, err := readSendAttachment("file", filePath)
	if err != nil {
		t.Fatal(err)
	}

	raw, err := buildSMTPMessage(OutgoingMail{
		From:       "demo@example.com",
		To:         []string{"sender@example.com"},
		Subject:    "Re: 测试主题",
		TextBody:   "hello",
		HTMLBody:   "<div>hello</div>",
		InReplyTo:  "<origin@example.com>",
		References: []string{"<root@example.com>", "<origin@example.com>"},
		MessageID:  "<reply@example.com>",
		Date:       time.Date(2026, 5, 6, 10, 0, 0, 0, time.FixedZone("CST", 8*3600)),
		Attachments: []SendAttachment{
			attachment,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if got := msg.Header.Get("In-Reply-To"); got != "<origin@example.com>" {
		t.Fatalf("In-Reply-To = %q", got)
	}
	if got := msg.Header.Get("References"); got != "<root@example.com> <origin@example.com>" {
		t.Fatalf("References = %q", got)
	}
	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}
	if mediaType != "multipart/mixed" {
		t.Fatalf("content type = %q", mediaType)
	}
	reader := multipart.NewReader(msg.Body, params["boundary"])
	partCount := 0
	foundAttachment := false
	for {
		part, partErr := reader.NextPart()
		if partErr == io.EOF {
			break
		}
		if partErr != nil {
			t.Fatal(partErr)
		}
		partCount++
		disposition := part.Header.Get("Content-Disposition")
		if strings.Contains(disposition, "attachment") && strings.Contains(disposition, "demo.txt") {
			foundAttachment = true
		}
		part.Close()
	}
	if partCount < 2 {
		t.Fatalf("expected multipart body with attachment, got %d parts", partCount)
	}
	if !foundAttachment {
		t.Fatalf("attachment part not found in raw message:\n%s", string(raw))
	}
}

func TestSenderRepliesWithTextImageAndFile(t *testing.T) {
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "demo.png")
	filePath := filepath.Join(tempDir, "demo.pdf")
	if err := os.WriteFile(imagePath, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("pdf"), 0o644); err != nil {
		t.Fatal(err)
	}

	loggerOut := &bytes.Buffer{}
	messageLog := &bytes.Buffer{}
	loader := stubMetaLoader{
		meta: &connectsvc.Meta{Name: DefaultName, Meta: `{"email":"demo@example.com","email_pop3":"pop.example.com:995","email_smtp":"smtp.example.com:465","email_password":"secret"}`},
		cfg: Config{
			Email:         "demo@example.com",
			EmailPOP3:     "pop.example.com:995",
			EmailSMTP:     "smtp.example.com:465",
			EmailPassword: "secret",
		},
	}
	transport := &stubSMTPTransport{}
	sender := NewSender(loader, log.New(loggerOut, "", 0), messageLog)
	sender.transport = transport
	sender.now = func() time.Time { return time.Unix(100, 0) }

	result, err := sender.Send(context.Background(), SendInput{
		Message: latestEmailConnectRequestMessageJSON("<origin@example.com>", "原始主题", "Sender <sender@example.com>", "<root@example.com>"),
		Content: "reply text",
		Images:  []string{imagePath},
		Files:   []string{filePath},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.To != "sender@example.com" {
		t.Fatalf("unexpected to: %+v", result)
	}
	if len(result.Sent) != 3 {
		t.Fatalf("expected 3 sent records, got %+v", result.Sent)
	}
	if got := result.Sent[0].Type; got != "text" {
		t.Fatalf("expected first sent type text, got %q", got)
	}
	if got := result.Sent[1].Type; got != "image" {
		t.Fatalf("expected second sent type image, got %q", got)
	}
	if got := result.Sent[2].Type; got != "file" {
		t.Fatalf("expected third sent type file, got %q", got)
	}
	if len(transport.calls) != 1 {
		t.Fatalf("expected one smtp send, got %d", len(transport.calls))
	}
	call := transport.calls[0]
	if call.mail.InReplyTo != "<origin@example.com>" {
		t.Fatalf("unexpected in-reply-to: %+v", call.mail)
	}
	if got := strings.Join(call.mail.References, " "); got != "<root@example.com> <origin@example.com>" {
		t.Fatalf("unexpected references: %q", got)
	}
	if call.mail.Subject != "Re: 原始主题" {
		t.Fatalf("unexpected subject: %q", call.mail.Subject)
	}
	if len(call.mail.Attachments) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(call.mail.Attachments))
	}
	logText := messageLog.String()
	if !strings.Contains(logText, `做了：开始准备发送邮件`) {
		t.Fatalf("expected request log: %s", logText)
	}
	if !strings.Contains(logText, `做了：确认了这封邮件要发给谁`) {
		t.Fatalf("expected parse log: %s", logText)
	}
	if !strings.Contains(logText, `做了：成功发出了邮件`) {
		t.Fatalf("unexpected send log: %s", logText)
	}
}

func TestRunCLISendUsesBuildServiceAndWritesLogs(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "email.log")

	originalBuilder := newServiceBuilder
	defer func() { newServiceBuilder = originalBuilder }()
	newServiceBuilder = func(flags map[string]string, logger *log.Logger, messageLog io.Writer) (*Service, error) {
		return NewService(Options{
			Client: &stubConnectClient{
				meta: &connectsvc.Meta{Name: DefaultName, Meta: `{"email":"demo@example.com","email_pop3":"pop.example.com:995","email_smtp":"smtp.example.com:465","email_password":"secret"}`},
			},
			Logger:     logger,
			MessageLog: messageLog,
		})
	}

	originalNewSender := newSender
	defer func() { newSender = originalNewSender }()
	newSender = func(loader MetaLoader, logger *log.Logger, messageLog io.Writer) *Sender {
		sender := NewSender(stubMetaLoader{
			meta: &connectsvc.Meta{Name: DefaultName},
			cfg: Config{
				Email:         "demo@example.com",
				EmailPOP3:     "pop.example.com:995",
				EmailSMTP:     "smtp.example.com:465",
				EmailPassword: "secret",
			},
		}, logger, messageLog)
		sender.transport = &stubSMTPTransport{}
		return sender
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := RunCLI([]string{
		"send",
		"--message", latestEmailConnectRequestMessageJSON("<origin@example.com>", "原始主题", "Sender <sender@example.com>", ""),
		"--content", "hello",
		"--log-file", logPath,
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("send exit code = %d, stderr = %s", code, stderr.String())
	}
	var result SendResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode send output: %v raw=%s", err, stdout.String())
	}
	if len(result.Sent) != 1 || result.Sent[0].Type != "text" {
		t.Fatalf("unexpected send output: %+v", result)
	}
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	logText := string(logData)
	if !strings.Contains(logText, `做了：开始准备发送邮件`) {
		t.Fatalf("expected send request log in email.log, got: %s", logText)
	}
	if !strings.Contains(logText, `做了：确认了这封邮件要发给谁`) {
		t.Fatalf("expected send parse log in email.log, got: %s", logText)
	}
	if !strings.Contains(logText, `做了：成功发出了邮件`) {
		t.Fatalf("expected send result log in email.log, got: %s", logText)
	}
}

func TestRunCLIInitUsesSameSendFlowAndWritesLogs(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "email.log")

	originalBuilder := newServiceBuilder
	defer func() { newServiceBuilder = originalBuilder }()
	newServiceBuilder = func(flags map[string]string, logger *log.Logger, messageLog io.Writer) (*Service, error) {
		return NewService(Options{
			Client: &stubConnectClient{
				meta: &connectsvc.Meta{Name: DefaultName, Meta: `{"email":"demo@example.com","email_pop3":"pop.example.com:995","email_smtp":"smtp.example.com:465","email_password":"secret"}`},
			},
			Logger:     logger,
			MessageLog: messageLog,
		})
	}

	originalNewSender := newSender
	defer func() { newSender = originalNewSender }()
	newSender = func(loader MetaLoader, logger *log.Logger, messageLog io.Writer) *Sender {
		sender := NewSender(stubMetaLoader{
			meta: &connectsvc.Meta{Name: DefaultName},
			cfg: Config{
				Email:         "demo@example.com",
				EmailPOP3:     "pop.example.com:995",
				EmailSMTP:     "smtp.example.com:465",
				EmailPassword: "secret",
			},
		}, logger, messageLog)
		sender.transport = &stubSMTPTransport{}
		return sender
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := RunCLI([]string{
		"init",
		"--message", latestEmailConnectRequestMessageJSON("<origin@example.com>", "原始主题", "Sender <sender@example.com>", ""),
		"--content", "hello",
		"--log-file", logPath,
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("init exit code = %d, stderr = %s", code, stderr.String())
	}
	var result SendResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode init output: %v raw=%s", err, stdout.String())
	}
	if len(result.Sent) != 1 || result.Sent[0].Type != "text" {
		t.Fatalf("unexpected init output: %+v", result)
	}
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	logText := string(logData)
	if !strings.Contains(logText, `动作=发送开始前通知`) {
		t.Fatalf("expected init request log in email.log, got: %s", logText)
	}
	if !strings.Contains(logText, `做了：确认了这封邮件要发给谁`) {
		t.Fatalf("expected init parse log in email.log, got: %s", logText)
	}
	if !strings.Contains(logText, `做了：成功发出了邮件`) {
		t.Fatalf("expected init result log in email.log, got: %s", logText)
	}
}

func TestSenderLogsSMTPHeaderWhenSMTPSendFails(t *testing.T) {
	messageLog := &bytes.Buffer{}
	transport := &stubSMTPTransport{err: errors.New("smtp rejected")}
	sender := NewSender(stubMetaLoader{
		meta: &connectsvc.Meta{Name: DefaultName},
		cfg: Config{
			Email:         "demo@example.com",
			EmailPOP3:     "pop.example.com:995",
			EmailSMTP:     "smtp.example.com:465",
			EmailPassword: "secret",
		},
	}, nil, messageLog)
	sender.transport = transport
	sender.now = func() time.Time { return time.Unix(100, 0) }

	_, err := sender.Send(context.Background(), SendInput{
		Action:  "send",
		Message: latestEmailConnectRequestMessageJSON("<origin@example.com>", "原始主题", "Sender <sender@example.com>", ""),
		Content: "hello",
	})
	if err == nil {
		t.Fatal("expected smtp send failure")
	}
	logText := messageLog.String()
	if !strings.Contains(logText, `做了：发送邮件失败`) {
		t.Fatalf("expected send failure log, got: %s", logText)
	}
	if !strings.Contains(logText, `原因：smtp rejected`) {
		t.Fatalf("expected smtp rejection reason in failure log, got: %s", logText)
	}
}

func TestRunCLISendInvalidMessageWritesFailureLog(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "email.log")

	originalBuilder := newServiceBuilder
	defer func() { newServiceBuilder = originalBuilder }()
	newServiceBuilder = func(flags map[string]string, logger *log.Logger, messageLog io.Writer) (*Service, error) {
		return NewService(Options{
			Client: &stubConnectClient{
				meta: &connectsvc.Meta{Name: DefaultName, Meta: `{"email":"demo@example.com","email_pop3":"pop.example.com:995","email_smtp":"smtp.example.com:465","email_password":"secret"}`},
			},
			Logger:     logger,
			MessageLog: messageLog,
		})
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := RunCLI([]string{
		"send",
		"--message", `{"id":1}`,
		"--content", "hello",
		"--log-file", logPath,
	}, stdout, stderr)
	if code != 1 {
		t.Fatalf("send exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "message.messageId not found") {
		t.Fatalf("expected parse error on stderr, got: %s", stderr.String())
	}
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	logText := string(logData)
	if !strings.Contains(logText, `做了：开始准备发送邮件`) {
		t.Fatalf("expected request log in email.log, got: %s", logText)
	}
	if !strings.Contains(logText, `原因：message.messageId not found`) {
		t.Fatalf("expected failure log in email.log, got: %s", logText)
	}
}

func TestRunCLIRejectsLegacyStartFlag(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := RunCLI([]string{"--start"}, stdout, stderr)
	if code == 0 {
		t.Fatalf("legacy --start unexpectedly succeeded, stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unknown command: --start") {
		t.Fatalf("stderr=%q, want unknown command", stderr.String())
	}
}

func TestStartBackgroundUsesPublicForegroundCommand(t *testing.T) {
	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "email.pid")
	logFile := filepath.Join(tempDir, "email.log")
	recorded := make([]string, 0, 8)

	originalBuilder := newServiceBuilder
	originalExecCommand := execCommand
	originalFindProcess := findProcess
	originalExecutable := osExecutableFn
	originalValidate := validateEmailConfigFn
	defer func() { newServiceBuilder = originalBuilder }()
	defer func() { execCommand = originalExecCommand }()
	defer func() { findProcess = originalFindProcess }()
	defer func() { osExecutableFn = originalExecutable }()
	defer func() { validateEmailConfigFn = originalValidate }()

	newServiceBuilder = func(flags map[string]string, logger *log.Logger, messageLog io.Writer) (*Service, error) {
		return &Service{
			name: DefaultName,
			client: &stubConnectClient{
				meta: &connectsvc.Meta{
					Name: DefaultName,
					Meta: `{"email":"demo@example.com","email_pop3":"ok.pop3.local","email_smtp":"ok.smtp.local","email_password":"secret"}`,
				},
			},
			logger:     logger,
			messageLog: messageLog,
		}, nil
	}
	osExecutableFn = func() (string, error) { return "/tmp/email", nil }
	validateEmailConfigFn = func(ctx context.Context, cfg Config) error { return nil }
	findProcess = func(pid int) (processHandle, error) {
		return stubProcessHandle{signalFn: func(sig os.Signal) error { return nil }}, nil
	}
	execCommand = func(name string, args ...string) *exec.Cmd {
		recorded = append([]string{name}, args...)
		cmd := exec.Command("sh", "-c", "echo $$ > \"$1\"; sleep 30", "sh", pidFile)
		return cmd
	}

	result, err := startBackground(map[string]string{
		"pid-file": pidFile,
		"log-file": logFile,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "started" {
		t.Fatalf("unexpected result: %+v", result)
	}
	got := strings.Join(recorded, " ")
	if !strings.Contains(got, "/tmp/email start") {
		t.Fatalf("expected public start command, got: %s", got)
	}
	if strings.Contains(got, " serve ") {
		t.Fatalf("unexpected internal serve command: %s", got)
	}
	if !strings.Contains(got, "--foreground true") {
		t.Fatalf("expected foreground flag, got: %s", got)
	}
}

func TestStartBackgroundFailsWhenStartupValidationFails(t *testing.T) {
	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "email.pid")
	logFile := filepath.Join(tempDir, "email.log")
	execCalled := false

	originalBuilder := newServiceBuilder
	originalExecCommand := execCommand
	originalValidate := validateEmailConfigFn
	defer func() { newServiceBuilder = originalBuilder }()
	defer func() { execCommand = originalExecCommand }()
	defer func() { validateEmailConfigFn = originalValidate }()

	newServiceBuilder = func(flags map[string]string, logger *log.Logger, messageLog io.Writer) (*Service, error) {
		return &Service{
			name: DefaultName,
			client: &stubConnectClient{
				meta: &connectsvc.Meta{
					Name: DefaultName,
					Meta: `{"email":"demo@example.com","email_pop3":"bad.pop3.local","email_smtp":"bad.smtp.local","email_password":"bad-secret"}`,
				},
			},
			logger:     logger,
			messageLog: messageLog,
		}, nil
	}
	validateEmailConfigFn = func(ctx context.Context, cfg Config) error {
		if cfg.EmailPOP3 != "bad.pop3.local" {
			t.Fatalf("cfg.EmailPOP3 = %q", cfg.EmailPOP3)
		}
		return errors.New("invalid email credentials")
	}
	execCommand = func(name string, args ...string) *exec.Cmd {
		execCalled = true
		return exec.Command("sh", "-c", "exit 0")
	}

	_, err := startBackground(map[string]string{
		"pid-file": pidFile,
		"log-file": logFile,
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "invalid email credentials") {
		t.Fatalf("err = %v, want invalid email credentials", err)
	}
	if execCalled {
		t.Fatalf("background process should not start after failed validation")
	}
}

func TestStopProcessTreatsZombiePIDAsStopped(t *testing.T) {
	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "email.pid")
	if err := os.WriteFile(pidFile, []byte("82675"), 0o644); err != nil {
		t.Fatal(err)
	}

	originalFindProcess := findProcess
	defer func() { findProcess = originalFindProcess }()
	findProcess = func(pid int) (processHandle, error) {
		return stubProcessHandle{signalFn: func(sig os.Signal) error { return nil }}, nil
	}

	originalExecCommand := execCommand
	defer func() { execCommand = originalExecCommand }()
	execCommand = func(name string, args ...string) *exec.Cmd {
		if name == "ps" {
			return exec.Command("sh", "-c", "printf 'Z\\n'")
		}
		return exec.Command(name, args...)
	}

	result, err := stopProcess(map[string]string{
		"pid-file": pidFile,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "stopped" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if _, err := os.Stat(pidFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("pid file should be removed, stat err=%v", err)
	}
}

type stubFetcher struct {
	mails []FetchedMail
	err   error
}

type stubPOP3Session struct {
	authUsers []string
	authErrs  map[string]error
}

func (s *stubPOP3Session) Auth(user, _ string) error {
	s.authUsers = append(s.authUsers, user)
	if s.authErrs == nil {
		return nil
	}
	return s.authErrs[user]
}

func (s *stubPOP3Session) Uidl(_ int) ([]pop3.MessageID, error) {
	return nil, nil
}

func (s *stubPOP3Session) RetrRaw(_ int) (*bytes.Buffer, error) {
	return bytes.NewBuffer(nil), nil
}

func (s *stubPOP3Session) Quit() error {
	return nil
}

type stubProcessHandle struct {
	signalFn func(sig os.Signal) error
}

func (s stubProcessHandle) Signal(sig os.Signal) error {
	if s.signalFn != nil {
		return s.signalFn(sig)
	}
	return nil
}

func (s *stubFetcher) Fetch(_ context.Context, _ Config, _ time.Time) ([]FetchedMail, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := make([]FetchedMail, len(s.mails))
	copy(out, s.mails)
	return out, nil
}

type capturingFetcher struct {
	lastSince time.Time
}

func (s *capturingFetcher) Fetch(_ context.Context, _ Config, since time.Time) ([]FetchedMail, error) {
	s.lastSince = since
	return nil, nil
}

type stubConnectClient struct {
	meta     *connectsvc.Meta
	requests []connectsvc.RequestInput
	mu       sync.Mutex
}

func (s *stubConnectClient) GetMeta(_ context.Context, _ string) (*connectsvc.Meta, error) {
	if s.meta == nil {
		return nil, io.EOF
	}
	copyMeta := *s.meta
	return &copyMeta, nil
}

func (s *stubConnectClient) AddRequest(_ context.Context, input connectsvc.RequestInput) (*connectsvc.Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, input)
	return &connectsvc.Request{ID: len(s.requests), Name: input.Name, Request: input.Request}, nil
}

func (s *stubConnectClient) Requests() []connectsvc.RequestInput {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]connectsvc.RequestInput, len(s.requests))
	copy(out, s.requests)
	return out
}

type stubRunner struct {
	calls   [][]string
	outputs [][]byte
}

func (s *stubRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	call := append([]string{name}, args...)
	s.calls = append(s.calls, call)
	if len(s.outputs) == 0 {
		return nil, io.EOF
	}
	out := s.outputs[0]
	s.outputs = s.outputs[1:]
	if out == nil {
		return nil, io.EOF
	}
	return out, nil
}

type stubMetaLoader struct {
	meta *connectsvc.Meta
	cfg  Config
	err  error
}

func (s stubMetaLoader) Load(_ context.Context) (*connectsvc.Meta, Config, error) {
	if s.err != nil {
		return nil, Config{}, s.err
	}
	if s.meta == nil {
		return nil, Config{}, io.EOF
	}
	copyMeta := *s.meta
	return &copyMeta, s.cfg, nil
}

type stubSMTPTransport struct {
	calls []stubSMTPSendCall
	err   error
}

type stubSMTPSendCall struct {
	cfg  Config
	mail OutgoingMail
}

func (s *stubSMTPTransport) Send(_ context.Context, cfg Config, mail OutgoingMail) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	s.calls = append(s.calls, stubSMTPSendCall{cfg: cfg, mail: mail})
	return firstNonEmpty(strings.TrimSpace(mail.MessageID), "<stub@example.com>"), nil
}

func latestEmailConnectRequestMessageJSON(messageID, subject, from, references string) string {
	headers := []map[string]string{
		{"name": "From", "value": from},
		{"name": "Subject", "value": subject},
		{"name": "Message-ID", "value": messageID},
	}
	if strings.TrimSpace(references) != "" {
		headers = append(headers, map[string]string{"name": "References", "value": references})
	}
	return latestEmailConnectRequestMessageJSONWithRawHeaders(messageID, subject, from, headers)
}

func latestEmailConnectRequestMessageJSONWithRawHeaders(messageID, subject, from string, headers []map[string]string) string {
	raw := map[string]any{
		"headers": headers,
		"content": "hello",
	}
	envelope := map[string]any{
		"source":     DefaultName,
		"receivedAt": "2026-05-22T00:00:00+08:00",
		"message": map[string]any{
			"uid":        "uid-1",
			"messageId":  messageID,
			"createTime": "2026-05-22T00:00:00+08:00",
			"subject":    subject,
			"from":       from,
			"content":    "hello",
			"artifacts":  []string{},
			"raw":        mustJSONText(raw),
		},
	}
	request := map[string]any{
		"id":         1,
		"name":       DefaultName,
		"request":    "hello",
		"rawRequest": mustJSONText(envelope),
	}
	return mustJSONText(request)
}

func mustJSONText(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(data)
}
