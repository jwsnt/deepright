package feishusvc

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"connect/connectsvc"
	ws "github.com/gorilla/websocket"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

type fakeFileInfo struct{}

func (fakeFileInfo) Name() string       { return "keychain" }
func (fakeFileInfo) Size() int64        { return 0 }
func (fakeFileInfo) Mode() os.FileMode  { return 0 }
func (fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (fakeFileInfo) IsDir() bool        { return false }
func (fakeFileInfo) Sys() any           { return nil }

func pemEncodeCertificate(t *testing.T, der []byte) []byte {
	t.Helper()
	block := &pem.Block{Type: "CERTIFICATE", Bytes: der}
	return pem.EncodeToMemory(block)
}

func TestServicePushesMockMessagesAndReconnects(t *testing.T) {
	messageAtMS := time.Now().Add(-time.Minute).UnixMilli()
	client := &stubConnectClient{
		meta: &connectsvc.Meta{
			Name: DefaultName,
			Meta: fmt.Sprintf(`{
				"mode":"mock",
				"heartbeatIntervalSec":1,
				"heartbeatTimeoutSec":1,
				"reconnectDelayMs":10,
				"mockHeartbeatMs":100,
				"mockDisconnectAfterMs":250,
				"mockMessages":[{"delayMs":50,"messageId":"m1","chatId":"chat-1","messageAtMs":%d,"content":"hello from mock"}]
			}`, messageAtMS),
		},
	}
	loggerOut := &bytes.Buffer{}
	messageLog := &bytes.Buffer{}
	svc, err := NewService(Options{
		Client:            client,
		Logger:            log.New(loggerOut, "", 0),
		MessageLog:        messageLog,
		HeartbeatInterval: 100 * time.Millisecond,
		HeartbeatTimeout:  180 * time.Millisecond,
		ReconnectDelay:    10 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 450*time.Millisecond)
	defer cancel()
	if err := svc.Run(ctx); err != nil {
		t.Fatal(err)
	}

	requests := client.Requests()
	if len(requests) != 1 {
		t.Fatalf("expected 1 pushed request, got %d", len(requests))
	}
	if requests[0].ExternalID != md5Key(messageAtMS, "hello from mock") {
		t.Fatalf("expected external id to be generated, got %+v", requests[0])
	}
	if requests[0].Request != "hello from mock" {
		t.Fatalf("expected request content to be normalized text, got %+v", requests[0])
	}
	if requests[0].RawRequest != "hello from mock" {
		t.Fatalf("expected raw request to keep original mock payload, got %+v", requests[0])
	}
	if !strings.Contains(messageLog.String(), "做了：收到了一条飞书消息") {
		t.Fatalf("expected message log entry, got: %s", messageLog.String())
	}
	if client.GetMetaCalls() < 2 {
		t.Fatalf("expected reconnect to fetch meta again, getMetaCalls=%d logs=%s", client.GetMetaCalls(), loggerOut.String())
	}
	if !strings.Contains(loggerOut.String(), "stage=reconnect") {
		t.Fatalf("expected reconnect log, got: %s", loggerOut.String())
	}
}

func TestServiceLoadConfigUsesKeyOnly(t *testing.T) {
	client := &stubConnectClient{
		metaByName: map[string]*connectsvc.Meta{
			DefaultName: {
				Name: DefaultName,
				Meta: `{"mode":"mock"}`,
			},
		},
	}
	svc, err := NewService(Options{
		Client:         client,
		Logger:         log.New(io.Discard, "", 0),
		MessageLog:     io.Discard,
		ReconnectDelay: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	meta, cfg, err := svc.loadConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if meta.Name != DefaultName {
		t.Fatalf("meta.Name = %q", meta.Name)
	}
	if cfg.Mode != "mock" {
		t.Fatalf("cfg.Mode = %q", cfg.Mode)
	}
	if len(client.metaCalls) != 1 || client.metaCalls[0] != DefaultName {
		t.Fatalf("metaCalls = %v", client.metaCalls)
	}
}

func TestRunCLIReturnsFixedParamAndName(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	if code := RunCLI([]string{"command"}, stdout, stderr); code != 0 {
		t.Fatalf("command exit code = %d, stderr = %s", code, stderr.String())
	}
	var commands []string
	if err := json.Unmarshal(stdout.Bytes(), &commands); err != nil {
		t.Fatalf("decode command output: %v; raw=%s", err, stdout.String())
	}
	if !reflect.DeepEqual(commands, []string{"command", "help", "name", "param", "scope", "schema", "init", "send", "start", "stop"}) {
		t.Fatalf("unexpected command output: %+v", commands)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected command stderr: %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := RunCLI([]string{"param"}, stdout, stderr); code != 0 {
		t.Fatalf("param exit code = %d, stderr = %s", code, stderr.String())
	}
	var params []string
	if err := json.Unmarshal(stdout.Bytes(), &params); err != nil {
		t.Fatalf("decode param output: %v; raw=%s", err, stdout.String())
	}
	if len(params) != 2 || params[0] != "appId" || params[1] != "appSecret" {
		t.Fatalf("unexpected param output: %+v", params)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected param stderr: %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := RunCLI([]string{"scope"}, stdout, stderr); code != 0 {
		t.Fatalf("scope exit code = %d, stderr = %s", code, stderr.String())
	}
	var scopes []string
	if err := json.Unmarshal(stdout.Bytes(), &scopes); err != nil {
		t.Fatalf("decode scope output: %v; raw=%s", err, stdout.String())
	}
	if !reflect.DeepEqual(scopes, []string{"reuse", "agent", "provider", "thinking", "swarm"}) {
		t.Fatalf("unexpected scope output: %+v", scopes)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected scope stderr: %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := RunCLI([]string{"schema"}, stdout, stderr); code != 0 {
		t.Fatalf("schema exit code = %d, stderr = %s", code, stderr.String())
	}
	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatalf("decode schema output: %v; raw=%s", err, stdout.String())
	}
	if schema["type"] != "object" {
		t.Fatalf("unexpected schema output: %+v", schema)
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema properties type = %T", schema["properties"])
	}
	for _, key := range []string{"content", "artifacts", "why_do_this"} {
		if _, exists := properties[key]; !exists {
			t.Fatalf("schema missing property %q: %+v", key, properties)
		}
	}
	required, ok := schema["required"].([]any)
	if !ok {
		t.Fatalf("schema required type = %T", schema["required"])
	}
	if len(required) != 1 || required[0] != "content" {
		t.Fatalf("schema required = %#v", required)
	}

	stdout.Reset()
	stderr.Reset()
	if code := RunCLI([]string{"name"}, stdout, stderr); code != 0 {
		t.Fatalf("name exit code = %d, stderr = %s", code, stderr.String())
	}
	var name struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &name); err != nil {
		t.Fatalf("decode name output: %v; raw=%s", err, stdout.String())
	}
	if name.Key != "feishu" || name.Name != "飞书" {
		t.Fatalf("unexpected name output: %+v", name)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected name stderr: %s", stderr.String())
	}
}

func TestHelpHidesLifecycleCommands(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"help"}} {
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		if code := RunCLI(args, stdout, stderr); code != 0 {
			t.Fatalf("%v exit code = %d, stderr = %s", args, code, stderr.String())
		}
		for _, hidden := range []string{"feishu init", "feishu start", "feishu stop", "\"init\"", "\"start\"", "\"stop\""} {
			if strings.Contains(stdout.String(), hidden) {
				t.Fatalf("%v unexpectedly mentions %q\n%s", args, hidden, stdout.String())
			}
		}
	}
}

func TestValidateConfigReturnsCompatSetupError(t *testing.T) {
	originalBuilder := newFeishuLarkClient
	defer func() { newFeishuLarkClient = originalBuilder }()

	newFeishuLarkClient = func(cfg Config) (*lark.Client, error) {
		return nil, errors.New("tls compat failed")
	}

	err := ValidateConfig(context.Background(), Config{AppID: "app", AppSecret: "secret"})
	if err == nil || !strings.Contains(err.Error(), "tls compat failed") {
		t.Fatalf("ValidateConfig error = %v, want tls compat failure", err)
	}
}

func TestNewLarkMessageDownloaderReturnsCompatError(t *testing.T) {
	originalBuilder := newFeishuLarkClient
	defer func() { newFeishuLarkClient = originalBuilder }()

	newFeishuLarkClient = func(cfg Config) (*lark.Client, error) {
		return nil, errors.New("tls compat failed")
	}

	downloader := newLarkMessageDownloader(Config{AppID: "app", AppSecret: "secret"})
	if downloader == nil {
		t.Fatal("downloader is nil")
	}
	_, _, err := downloader.DownloadImage(context.Background(), "m", "img")
	if err == nil || !strings.Contains(err.Error(), "tls compat failed") {
		t.Fatalf("DownloadImage error = %v, want tls compat failure", err)
	}
}

func TestInstallFeishuNetworkCompatLoadsDarwinRoots(t *testing.T) {
	originalGOOS := currentGOOS
	originalGOARCH := currentGOARCH
	originalHomeDir := userHomeDirFn
	originalStat := keychainStatFn
	originalExport := exportKeychainCertificatesFn
	originalDefaultTransport := http.DefaultTransport
	originalDefaultDialer := ws.DefaultDialer
	defer func() {
		currentGOOS = originalGOOS
		currentGOARCH = originalGOARCH
		userHomeDirFn = originalHomeDir
		keychainStatFn = originalStat
		exportKeychainCertificatesFn = originalExport
		http.DefaultTransport = originalDefaultTransport
		ws.DefaultDialer = originalDefaultDialer
		feishuNetworkCompatOnce = sync.Once{}
		feishuNetworkCompatErr = nil
		feishuTLSConfigOnce = sync.Once{}
		feishuTLSConfigValue = nil
		feishuTLSConfigErr = nil
	}()

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()
	if len(ts.TLS.Certificates) == 0 || len(ts.TLS.Certificates[0].Certificate) == 0 {
		t.Fatal("missing test certificate")
	}
	testPEM := pemEncodeCertificate(t, ts.TLS.Certificates[0].Certificate[0])

	currentGOOS = "darwin"
	currentGOARCH = "amd64"
	userHomeDirFn = func() (string, error) { return "/Users/test", nil }
	keychainStatFn = func(path string) (os.FileInfo, error) { return fakeFileInfo{}, nil }
	exportKeychainCertificatesFn = func(ctx context.Context, keychainPath string) ([]byte, error) {
		return testPEM, nil
	}
	http.DefaultTransport = &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}}
	ws.DefaultDialer = &ws.Dialer{}
	feishuNetworkCompatOnce = sync.Once{}
	feishuNetworkCompatErr = nil
	feishuTLSConfigOnce = sync.Once{}
	feishuTLSConfigValue = nil
	feishuTLSConfigErr = nil

	if err := installFeishuNetworkCompat(); err != nil {
		t.Fatal(err)
	}

	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		t.Fatalf("default transport type = %T", http.DefaultTransport)
	}
	if transport.TLSClientConfig == nil || transport.TLSClientConfig.RootCAs == nil {
		t.Fatalf("default transport TLS config = %+v", transport.TLSClientConfig)
	}
	if ws.DefaultDialer == nil || ws.DefaultDialer.TLSClientConfig == nil || ws.DefaultDialer.TLSClientConfig.RootCAs == nil {
		t.Fatalf("websocket dialer TLS config = %+v", ws.DefaultDialer)
	}
}

func TestInstallFeishuNetworkCompatNoopOutsideDarwinAMD64(t *testing.T) {
	cases := []struct {
		name   string
		goos   string
		goarch string
	}{
		{name: "windows-amd64", goos: "windows", goarch: "amd64"},
		{name: "linux-amd64", goos: "linux", goarch: "amd64"},
		{name: "darwin-arm64", goos: "darwin", goarch: "arm64"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			originalGOOS := currentGOOS
			originalGOARCH := currentGOARCH
			originalDefaultTransport := http.DefaultTransport
			originalDefaultDialer := ws.DefaultDialer
			defer func() {
				currentGOOS = originalGOOS
				currentGOARCH = originalGOARCH
				http.DefaultTransport = originalDefaultTransport
				ws.DefaultDialer = originalDefaultDialer
				feishuNetworkCompatOnce = sync.Once{}
				feishuNetworkCompatErr = nil
				feishuTLSConfigOnce = sync.Once{}
				feishuTLSConfigValue = nil
				feishuTLSConfigErr = nil
			}()

			currentGOOS = tc.goos
			currentGOARCH = tc.goarch
			http.DefaultTransport = &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}}
			ws.DefaultDialer = &ws.Dialer{}
			feishuNetworkCompatOnce = sync.Once{}
			feishuNetworkCompatErr = nil
			feishuTLSConfigOnce = sync.Once{}
			feishuTLSConfigValue = nil
			feishuTLSConfigErr = nil

			transportBefore := http.DefaultTransport
			dialerBefore := ws.DefaultDialer
			if err := installFeishuNetworkCompat(); err != nil {
				t.Fatal(err)
			}
			if http.DefaultTransport != transportBefore {
				t.Fatal("default transport changed unexpectedly")
			}
			if ws.DefaultDialer != dialerBefore {
				t.Fatal("websocket dialer changed unexpectedly")
			}
		})
	}
}

func TestSenderSendTextToChat(t *testing.T) {
	loggerOut := &bytes.Buffer{}
	messageLog := &bytes.Buffer{}
	loader := stubMetaLoader{
		meta: &connectsvc.Meta{Name: "feishu", ChatID: "oc_test_chat", Meta: `{"appId":"app","appSecret":"secret"}`},
		cfg:  Config{AppID: "app", AppSecret: "secret"},
	}
	messageAPI := &stubMessageAPI{
		createResp: &larkim.CreateMessageResp{
			CodeError: larkcore.CodeError{Code: 0},
			Data:      &larkim.CreateMessageRespData{MessageId: stringPtr("om_text_1"), ChatId: stringPtr("oc_test_chat")},
		},
	}
	sender := NewSender(loader, log.New(loggerOut, "", 0), messageLog)
	sender.apis = LarkAPISet{
		Message: messageAPI,
		Image:   &stubImageAPI{},
		File:    &stubFileAPI{},
	}
	sender.now = func() time.Time { return time.Unix(100, 0) }

	_, err := sender.Send(context.Background(), SendInput{Content: "hello"})
	if err == nil || !strings.Contains(err.Error(), "message is required") {
		t.Fatalf("expected message required error, got %v", err)
	}
	if len(messageAPI.createReqs) != 0 {
		t.Fatalf("expected no create request, got %d", len(messageAPI.createReqs))
	}
	logText := messageLog.String()
	if !strings.Contains(logText, `做了：开始准备发送飞书消息`) {
		t.Fatalf("expected request log entry, got: %s", logText)
	}
	if !strings.Contains(logText, `原因：message is required`) {
		t.Fatalf("expected failure log entry, got: %s", logText)
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

	loader := stubMetaLoader{
		meta: &connectsvc.Meta{Name: "feishu", ChatID: "oc_test_chat", Meta: `{"appId":"app","appSecret":"secret"}`},
		cfg:  Config{AppID: "app", AppSecret: "secret"},
	}
	messageAPI := &stubMessageAPI{
		replyResps: []*larkim.ReplyMessageResp{
			{CodeError: larkcore.CodeError{Code: 0}, Data: &larkim.ReplyMessageRespData{MessageId: stringPtr("om_reply_text")}},
			{CodeError: larkcore.CodeError{Code: 0}, Data: &larkim.ReplyMessageRespData{MessageId: stringPtr("om_reply_image")}},
			{CodeError: larkcore.CodeError{Code: 0}, Data: &larkim.ReplyMessageRespData{MessageId: stringPtr("om_reply_file")}},
		},
	}
	imageAPI := &stubImageAPI{
		resp: &larkim.CreateImageResp{
			CodeError: larkcore.CodeError{Code: 0},
			Data:      &larkim.CreateImageRespData{ImageKey: stringPtr("img_key_1")},
		},
	}
	fileAPI := &stubFileAPI{
		resp: &larkim.CreateFileResp{
			CodeError: larkcore.CodeError{Code: 0},
			Data:      &larkim.CreateFileRespData{FileKey: stringPtr("file_key_1")},
		},
	}
	sender := NewSender(loader, log.New(io.Discard, "", 0), &bytes.Buffer{})
	sender.apis = LarkAPISet{Message: messageAPI, Image: imageAPI, File: fileAPI}

	result, err := sender.Send(context.Background(), SendInput{
		Message: latestConnectRequestMessageJSON("om_origin"),
		Content: "reply text",
		Images:  []string{imagePath},
		Files:   []string{filePath},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Sent) != 3 {
		t.Fatalf("expected 3 sent records, got %+v", result.Sent)
	}
	if len(messageAPI.replyReqs) != 3 {
		t.Fatalf("expected 3 reply requests, got %d", len(messageAPI.replyReqs))
	}
	if len(imageAPI.reqs) != 1 || len(fileAPI.reqs) != 1 {
		t.Fatalf("expected one upload per resource, image=%d file=%d", len(imageAPI.reqs), len(fileAPI.reqs))
	}
	assertUploadedFileType(t, fileAPI.reqs[0], larkim.FileTypeStream)
	if got := result.Sent[0].Type; got != "image" {
		t.Fatalf("expected first sent type image, got %q", got)
	}
	if got := result.Sent[1].Type; got != "file" {
		t.Fatalf("expected second sent type file, got %q", got)
	}
	if got := result.Sent[2].Type; got != "text" {
		t.Fatalf("expected third sent type text, got %q", got)
	}
	if got := result.Sent[0].ResourceKey; got != "img_key_1" {
		t.Fatalf("image resource key = %q", got)
	}
	if got := result.Sent[1].ResourceKey; got != "file_key_1" {
		t.Fatalf("file resource key = %q", got)
	}
	assertReplyMessageType(t, messageAPI.replyJSON[0], "image")
	assertReplyMessageType(t, messageAPI.replyJSON[1], "file")
	assertPostReply(t, messageAPI.replyJSON[2], "reply text")
}

func TestNewSenderUses180SecondAttachmentTimeout(t *testing.T) {
	sender := NewSender(stubMetaLoader{}, log.New(io.Discard, "", 0), io.Discard)
	if sender.httpClient == nil {
		t.Fatal("expected sender http client")
	}
	if sender.httpClient.Timeout != 180*time.Second {
		t.Fatalf("http client timeout = %s, want %s", sender.httpClient.Timeout, 180*time.Second)
	}
}

func TestSenderRepliesWithImageOnly(t *testing.T) {
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "demo.png")
	if err := os.WriteFile(imagePath, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}

	loader := stubMetaLoader{
		meta: &connectsvc.Meta{Name: "feishu", ChatID: "oc_test_chat", Meta: `{"appId":"app","appSecret":"secret"}`},
		cfg:  Config{AppID: "app", AppSecret: "secret"},
	}
	messageAPI := &stubMessageAPI{
		replyResps: []*larkim.ReplyMessageResp{
			{CodeError: larkcore.CodeError{Code: 0}, Data: &larkim.ReplyMessageRespData{MessageId: stringPtr("om_reply_image")}},
		},
	}
	imageAPI := &stubImageAPI{
		resp: &larkim.CreateImageResp{
			CodeError: larkcore.CodeError{Code: 0},
			Data:      &larkim.CreateImageRespData{ImageKey: stringPtr("img_key_only")},
		},
	}
	sender := NewSender(loader, log.New(io.Discard, "", 0), &bytes.Buffer{})
	sender.apis = LarkAPISet{Message: messageAPI, Image: imageAPI, File: &stubFileAPI{}}

	result, err := sender.Send(context.Background(), SendInput{
		Message: latestConnectRequestMessageJSON("om_origin"),
		Images:  []string{imagePath},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Sent) != 1 || result.Sent[0].Type != "image" {
		t.Fatalf("unexpected send result: %+v", result)
	}
	if len(imageAPI.reqs) != 1 {
		t.Fatalf("expected 1 image upload request, got %d", len(imageAPI.reqs))
	}
	if len(messageAPI.replyReqs) != 1 {
		t.Fatalf("expected 1 reply request, got %d", len(messageAPI.replyReqs))
	}
	assertReplyMessageType(t, messageAPI.replyJSON[0], "image")
}

func TestSenderRewritesMarkdownRemoteImagesToFeishuImageKeys(t *testing.T) {
	tempDir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/demo.png" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png"))
	}))
	defer server.Close()

	loader := stubMetaLoader{
		meta: &connectsvc.Meta{Name: "feishu", ChatID: "oc_test_chat", Meta: `{"appId":"app","appSecret":"secret"}`},
		cfg:  Config{AppID: "app", AppSecret: "secret"},
	}
	messageAPI := &stubMessageAPI{
		replyResps: []*larkim.ReplyMessageResp{
			{CodeError: larkcore.CodeError{Code: 0}, Data: &larkim.ReplyMessageRespData{MessageId: stringPtr("om_reply_text")}},
		},
	}
	imageAPI := &stubImageAPI{
		resp: &larkim.CreateImageResp{
			CodeError: larkcore.CodeError{Code: 0},
			Data:      &larkim.CreateImageRespData{ImageKey: stringPtr("img_key_remote")},
		},
	}
	sender := NewSender(loader, log.New(io.Discard, "", 0), &bytes.Buffer{})
	sender.downloadDir = tempDir
	sender.apis = LarkAPISet{Message: messageAPI, Image: imageAPI, File: &stubFileAPI{}}

	_, err := sender.Send(context.Background(), SendInput{
		Message: latestConnectRequestMessageJSON("om_origin"),
		Content: "图文\n\n![赛博哈利](" + server.URL + `/demo.png)`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(imageAPI.reqs) != 1 {
		t.Fatalf("expected 1 uploaded markdown image, got %d", len(imageAPI.reqs))
	}
	assertPostReply(t, messageAPI.replyJSON[0], "图文\n\n![赛博哈利](img_key_remote)")
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected downloaded markdown image artifact")
	}
}

func TestSenderRewritesMarkdownLocalImagesToFeishuImageKeys(t *testing.T) {
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "demo.png")
	if err := os.WriteFile(imagePath, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}

	loader := stubMetaLoader{
		meta: &connectsvc.Meta{Name: "feishu", ChatID: "oc_test_chat", Meta: `{"appId":"app","appSecret":"secret"}`},
		cfg:  Config{AppID: "app", AppSecret: "secret"},
	}
	messageAPI := &stubMessageAPI{
		replyResps: []*larkim.ReplyMessageResp{
			{CodeError: larkcore.CodeError{Code: 0}, Data: &larkim.ReplyMessageRespData{MessageId: stringPtr("om_reply_text")}},
		},
	}
	imageAPI := &stubImageAPI{
		resp: &larkim.CreateImageResp{
			CodeError: larkcore.CodeError{Code: 0},
			Data:      &larkim.CreateImageRespData{ImageKey: stringPtr("img_key_local")},
		},
	}
	sender := NewSender(loader, log.New(io.Discard, "", 0), &bytes.Buffer{})
	sender.apis = LarkAPISet{Message: messageAPI, Image: imageAPI, File: &stubFileAPI{}}

	_, err := sender.Send(context.Background(), SendInput{
		Message: latestConnectRequestMessageJSON("om_origin"),
		Content: "本地图\n\n![赛博哈利](" + imagePath + ")\n\n![赛博哈利2](file://" + imagePath + ")",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(imageAPI.reqs) != 2 {
		t.Fatalf("expected 2 uploaded markdown images, got %d", len(imageAPI.reqs))
	}
	assertPostReply(t, messageAPI.replyJSON[0], "本地图\n\n![赛博哈利](img_key_local)\n\n![赛博哈利2](img_key_local)")
}

func TestSenderSchemaContentMergesArtifactsAndLogs(t *testing.T) {
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
	loader := stubMetaLoader{
		meta: &connectsvc.Meta{Name: "feishu", ChatID: "oc_test_chat", Meta: `{"appId":"app","appSecret":"secret"}`},
		cfg:  Config{AppID: "app", AppSecret: "secret"},
	}
	messageAPI := &stubMessageAPI{
		replyResps: []*larkim.ReplyMessageResp{
			{CodeError: larkcore.CodeError{Code: 0}, Data: &larkim.ReplyMessageRespData{MessageId: stringPtr("om_reply_text")}},
			{CodeError: larkcore.CodeError{Code: 0}, Data: &larkim.ReplyMessageRespData{MessageId: stringPtr("om_reply_image")}},
			{CodeError: larkcore.CodeError{Code: 0}, Data: &larkim.ReplyMessageRespData{MessageId: stringPtr("om_reply_file")}},
		},
	}
	imageAPI := &stubImageAPI{
		resp: &larkim.CreateImageResp{
			CodeError: larkcore.CodeError{Code: 0},
			Data:      &larkim.CreateImageRespData{ImageKey: stringPtr("img_key_1")},
		},
	}
	fileAPI := &stubFileAPI{
		resp: &larkim.CreateFileResp{
			CodeError: larkcore.CodeError{Code: 0},
			Data:      &larkim.CreateFileRespData{FileKey: stringPtr("file_key_1")},
		},
	}
	sender := NewSender(loader, log.New(loggerOut, "", 0), &bytes.Buffer{})
	sender.apis = LarkAPISet{Message: messageAPI, Image: imageAPI, File: fileAPI}

	_, err := sender.Send(context.Background(), SendInput{
		Message: latestConnectRequestMessageJSON("om_origin"),
		Content: `{"content":"请查收附件","artifacts":[{"path":"` + imagePath + `","desc":"截图"},{"path":"` + filePath + `","desc":"报告"}],"why_do_this":"补充执行结果"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(messageAPI.replyReqs) != 3 {
		t.Fatalf("expected 3 reply requests, got %d", len(messageAPI.replyReqs))
	}
	assertReplyMessageType(t, messageAPI.replyJSON[0], "image")
	assertReplyMessageType(t, messageAPI.replyJSON[1], "file")
	assertPostReply(t, messageAPI.replyJSON[2], "请查收附件")
	if !strings.Contains(loggerOut.String(), `schema-send content="请查收附件"`) {
		t.Fatalf("expected schema content log, got %s", loggerOut.String())
	}
	if !strings.Contains(loggerOut.String(), strconv.Quote(imagePath)) || !strings.Contains(loggerOut.String(), strconv.Quote(filePath)) {
		t.Fatalf("expected schema artifact logs, got %s", loggerOut.String())
	}
}

func TestSenderSchemaContentSkipsArtifactImagesAlreadyEmbeddedInMarkdown(t *testing.T) {
	tempDir := t.TempDir()
	embeddedImagePath := filepath.Join(tempDir, "embedded.png")
	extraImagePath := filepath.Join(tempDir, "extra.png")
	filePath := filepath.Join(tempDir, "demo.pdf")
	if err := os.WriteFile(embeddedImagePath, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(extraImagePath, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("pdf"), 0o644); err != nil {
		t.Fatal(err)
	}

	loader := stubMetaLoader{
		meta: &connectsvc.Meta{Name: "feishu", ChatID: "oc_test_chat", Meta: `{"appId":"app","appSecret":"secret"}`},
		cfg:  Config{AppID: "app", AppSecret: "secret"},
	}
	messageAPI := &stubMessageAPI{
		replyResps: []*larkim.ReplyMessageResp{
			{CodeError: larkcore.CodeError{Code: 0}, Data: &larkim.ReplyMessageRespData{MessageId: stringPtr("om_reply_image")}},
			{CodeError: larkcore.CodeError{Code: 0}, Data: &larkim.ReplyMessageRespData{MessageId: stringPtr("om_reply_file")}},
			{CodeError: larkcore.CodeError{Code: 0}, Data: &larkim.ReplyMessageRespData{MessageId: stringPtr("om_reply_text")}},
		},
	}
	imageAPI := &stubImageAPI{
		resp: &larkim.CreateImageResp{
			CodeError: larkcore.CodeError{Code: 0},
			Data:      &larkim.CreateImageRespData{ImageKey: stringPtr("img_key_1")},
		},
	}
	fileAPI := &stubFileAPI{
		resp: &larkim.CreateFileResp{
			CodeError: larkcore.CodeError{Code: 0},
			Data:      &larkim.CreateFileRespData{FileKey: stringPtr("file_key_1")},
		},
	}
	sender := NewSender(loader, log.New(io.Discard, "", 0), &bytes.Buffer{})
	sender.apis = LarkAPISet{Message: messageAPI, Image: imageAPI, File: fileAPI}

	_, err := sender.Send(context.Background(), SendInput{
		Message: latestConnectRequestMessageJSON("om_origin"),
		Content: `{"content":"请查收结果\n\n![已嵌入图片](` + embeddedImagePath + `)","artifacts":[{"path":"` + embeddedImagePath + `","desc":"正文图片"},{"path":"` + extraImagePath + `","desc":"额外截图"},{"path":"` + filePath + `","desc":"报告"}],"why_do_this":"补充执行结果"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(imageAPI.reqs) != 2 {
		t.Fatalf("expected 2 image uploads (1 markdown + 1 extra artifact), got %d", len(imageAPI.reqs))
	}
	if len(messageAPI.replyReqs) != 3 {
		t.Fatalf("expected 3 reply requests, got %d", len(messageAPI.replyReqs))
	}
	assertReplyMessageType(t, messageAPI.replyJSON[0], "image")
	assertReplyMessageType(t, messageAPI.replyJSON[1], "file")
	assertPostReply(t, messageAPI.replyJSON[2], "请查收结果\n\n![已嵌入图片](img_key_1)")
}

func TestSenderSchemaContentMergesArtifactsWithoutWhyDoThis(t *testing.T) {
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "demo.png")
	filePath := filepath.Join(tempDir, "demo.pdf")
	if err := os.WriteFile(imagePath, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("%PDF"), 0o644); err != nil {
		t.Fatal(err)
	}

	loggerOut := &bytes.Buffer{}
	loader := stubMetaLoader{
		meta: &connectsvc.Meta{Name: "feishu", ChatID: "oc_test_chat", Meta: `{"appId":"app","appSecret":"secret"}`},
		cfg:  Config{AppID: "app", AppSecret: "secret"},
	}
	messageAPI := &stubMessageAPI{
		replyResps: []*larkim.ReplyMessageResp{
			{CodeError: larkcore.CodeError{Code: 0}, Data: &larkim.ReplyMessageRespData{MessageId: stringPtr("om_reply_text")}},
			{CodeError: larkcore.CodeError{Code: 0}, Data: &larkim.ReplyMessageRespData{MessageId: stringPtr("om_reply_image")}},
			{CodeError: larkcore.CodeError{Code: 0}, Data: &larkim.ReplyMessageRespData{MessageId: stringPtr("om_reply_file")}},
		},
	}
	imageAPI := &stubImageAPI{
		resp: &larkim.CreateImageResp{
			CodeError: larkcore.CodeError{Code: 0},
			Data:      &larkim.CreateImageRespData{ImageKey: stringPtr("img_key_1")},
		},
	}
	fileAPI := &stubFileAPI{
		resp: &larkim.CreateFileResp{
			CodeError: larkcore.CodeError{Code: 0},
			Data:      &larkim.CreateFileRespData{FileKey: stringPtr("file_key_1")},
		},
	}
	sender := NewSender(loader, log.New(loggerOut, "", 0), &bytes.Buffer{})
	sender.apis = LarkAPISet{Message: messageAPI, Image: imageAPI, File: fileAPI}

	_, err := sender.Send(context.Background(), SendInput{
		Message: latestConnectRequestMessageJSON("om_origin"),
		Content: `{"content":"请查收附件","artifacts":[{"path":"` + imagePath + `","desc":"截图"},{"path":"` + filePath + `","desc":"报告"}]}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(messageAPI.replyReqs) != 3 {
		t.Fatalf("expected 3 reply requests, got %d", len(messageAPI.replyReqs))
	}
	assertReplyMessageType(t, messageAPI.replyJSON[0], "image")
	assertReplyMessageType(t, messageAPI.replyJSON[1], "file")
	assertPostReply(t, messageAPI.replyJSON[2], "请查收附件")
	if !strings.Contains(loggerOut.String(), `schema-send content="请查收附件"`) {
		t.Fatalf("expected schema content log, got %s", loggerOut.String())
	}
}

func TestSenderUploadsFileMessagesAsStream(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "demo.pdf")
	if err := os.WriteFile(filePath, []byte("%PDF"), 0o644); err != nil {
		t.Fatal(err)
	}

	loader := stubMetaLoader{
		meta: &connectsvc.Meta{Name: "feishu", ChatID: "oc_test_chat", Meta: `{"appId":"app","appSecret":"secret"}`},
		cfg:  Config{AppID: "app", AppSecret: "secret"},
	}
	messageAPI := &stubMessageAPI{
		replyResps: []*larkim.ReplyMessageResp{
			{CodeError: larkcore.CodeError{Code: 0}, Data: &larkim.ReplyMessageRespData{MessageId: stringPtr("om_reply_file")}},
		},
	}
	fileAPI := &stubFileAPI{
		resp: &larkim.CreateFileResp{
			CodeError: larkcore.CodeError{Code: 0},
			Data:      &larkim.CreateFileRespData{FileKey: stringPtr("file_key_stream")},
		},
	}
	sender := NewSender(loader, log.New(io.Discard, "", 0), &bytes.Buffer{})
	sender.apis = LarkAPISet{Message: messageAPI, Image: &stubImageAPI{}, File: fileAPI}

	result, err := sender.Send(context.Background(), SendInput{
		Message: latestConnectRequestMessageJSON("om_origin"),
		Files:   []string{filePath},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Sent) != 1 || result.Sent[0].Type != "file" {
		t.Fatalf("unexpected send result: %+v", result)
	}
	if len(fileAPI.reqs) != 1 {
		t.Fatalf("expected 1 file upload request, got %d", len(fileAPI.reqs))
	}
	assertUploadedFileType(t, fileAPI.reqs[0], larkim.FileTypeStream)
	assertReplyMessageType(t, messageAPI.replyJSON[0], "file")
}

func TestSenderFallsBackToRawContentWhenMarkdownImageRewriteFails(t *testing.T) {
	loader := stubMetaLoader{
		meta: &connectsvc.Meta{Name: "feishu", ChatID: "oc_test_chat", Meta: `{"appId":"app","appSecret":"secret"}`},
		cfg:  Config{AppID: "app", AppSecret: "secret"},
	}
	messageAPI := &stubMessageAPI{
		replyResps: []*larkim.ReplyMessageResp{
			{CodeError: larkcore.CodeError{Code: 0}, Data: &larkim.ReplyMessageRespData{MessageId: stringPtr("om_reply_text")}},
		},
	}
	imageAPI := &stubImageAPI{err: errors.New("upload failed")}
	fileAPI := &stubFileAPI{
		resp: &larkim.CreateFileResp{
			CodeError: larkcore.CodeError{Code: 0},
			Data:      &larkim.CreateFileRespData{FileKey: stringPtr("file_key_1")},
		},
	}
	sender := NewSender(loader, log.New(io.Discard, "", 0), &bytes.Buffer{})
	sender.apis = LarkAPISet{Message: messageAPI, Image: imageAPI, File: fileAPI}

	_, err := sender.Send(context.Background(), SendInput{
		Message: latestConnectRequestMessageJSON("om_origin"),
		Content: `{"content":"![图](file:///tmp/demo.png)","artifacts":[{"path":"/tmp/demo.pdf","desc":"报告"}],"why_do_this":"补充执行结果"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(messageAPI.replyReqs) != 1 {
		t.Fatalf("expected only text reply after fallback, got %d", len(messageAPI.replyReqs))
	}
	if len(fileAPI.reqs) != 0 {
		t.Fatalf("expected file sends to be dropped after fallback, got %d", len(fileAPI.reqs))
	}
	assertPostReply(t, messageAPI.replyJSON[0], `{"content":"![图](file:///tmp/demo.png)","artifacts":[{"path":"/tmp/demo.pdf","desc":"报告"}],"why_do_this":"补充执行结果"}`)
}

func TestSenderTerminatesWhenFileAttachmentFailsAfterThreeRetries(t *testing.T) {
	loader := stubMetaLoader{
		meta: &connectsvc.Meta{Name: "feishu", ChatID: "oc_test_chat", Meta: `{"appId":"app","appSecret":"secret"}`},
		cfg:  Config{AppID: "app", AppSecret: "secret"},
	}
	messageLog := &bytes.Buffer{}
	messageAPI := &stubMessageAPI{}
	fileAPI := &stubFileAPI{}
	sender := NewSender(loader, log.New(io.Discard, "", 0), messageLog)
	sender.apis = LarkAPISet{Message: messageAPI, Image: &stubImageAPI{}, File: fileAPI}

	_, err := sender.Send(context.Background(), SendInput{
		Message: latestConnectRequestMessageJSON("om_origin"),
		Content: `{"content":"请查收附件","artifacts":[{"path":"/tmp/not-exists.pdf","desc":"报告"}],"why_do_this":"补充执行结果"}`,
	})
	if err == nil {
		t.Fatal("expected attachment failure")
	}
	if len(fileAPI.reqs) != 0 {
		t.Fatalf("expected file upload builder to fail before request, got %d", len(fileAPI.reqs))
	}
	if len(messageAPI.replyReqs) != 0 {
		t.Fatalf("expected send to terminate before text reply, got %d", len(messageAPI.replyReqs))
	}
	logText := messageLog.String()
	if !strings.Contains(logText, `重试进度=第1/3次`) {
		t.Fatalf("missing retry log: %s", logText)
	}
	if !strings.Contains(logText, `已停止继续重试`) {
		t.Fatalf("missing terminate log: %s", logText)
	}
}

func TestSenderTerminatesWhenRawContentSendFailsAfterThreeRetries(t *testing.T) {
	loader := stubMetaLoader{
		meta: &connectsvc.Meta{Name: "feishu", ChatID: "oc_test_chat", Meta: `{"appId":"app","appSecret":"secret"}`},
		cfg:  Config{AppID: "app", AppSecret: "secret"},
	}
	messageLog := &bytes.Buffer{}
	messageAPI := &stubMessageAPI{
		replyResps: []*larkim.ReplyMessageResp{
			{CodeError: larkcore.CodeError{Code: 230099}},
			{CodeError: larkcore.CodeError{Code: 230099}},
			{CodeError: larkcore.CodeError{Code: 230099}},
		},
	}
	imageAPI := &stubImageAPI{err: errors.New("upload failed")}
	sender := NewSender(loader, log.New(io.Discard, "", 0), messageLog)
	sender.apis = LarkAPISet{Message: messageAPI, Image: imageAPI, File: &stubFileAPI{}}

	_, err := sender.Send(context.Background(), SendInput{
		Message: latestConnectRequestMessageJSON("om_origin"),
		Content: `{"content":"![图](file:///tmp/demo.png)","artifacts":[{"path":"/tmp/demo.pdf","desc":"报告"}],"why_do_this":"补充执行结果"}`,
	})
	if err == nil {
		t.Fatal("expected send failure")
	}
	if len(messageAPI.replyReqs) != 3 {
		t.Fatalf("expected 3 reply attempts, got %d", len(messageAPI.replyReqs))
	}
	if strings.Contains(messageLog.String(), "<消息异常>请登录客户端查看") {
		t.Fatalf("unexpected exception fallback log: %s", messageLog.String())
	}
	if !strings.Contains(messageLog.String(), `已停止继续重试`) {
		t.Fatalf("missing terminate log: %s", messageLog.String())
	}
}

func TestSenderReplyWithoutChatID(t *testing.T) {
	loader := stubMetaLoader{
		meta: &connectsvc.Meta{Name: "feishu", ChatID: "", Meta: `{"appId":"app","appSecret":"secret"}`},
		cfg:  Config{AppID: "app", AppSecret: "secret"},
	}
	messageAPI := &stubMessageAPI{
		replyResps: []*larkim.ReplyMessageResp{
			{CodeError: larkcore.CodeError{Code: 0}, Data: &larkim.ReplyMessageRespData{MessageId: stringPtr("om_reply_text")}},
		},
	}
	sender := NewSender(loader, log.New(io.Discard, "", 0), &bytes.Buffer{})
	sender.apis = LarkAPISet{
		Message: messageAPI,
		Image:   &stubImageAPI{},
		File:    &stubFileAPI{},
	}

	result, err := sender.Send(context.Background(), SendInput{
		Message: latestConnectRequestMessageJSON("om_origin"),
		Content: "reply without chat id",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.MessageID != "om_origin" || len(result.Sent) != 1 || result.Sent[0].Type != "text" {
		t.Fatalf("unexpected send result: %+v", result)
	}
	if len(messageAPI.replyReqs) != 1 {
		t.Fatalf("expected one reply request, got %d", len(messageAPI.replyReqs))
	}
	assertPostReply(t, messageAPI.replyJSON[0], "reply without chat id")
	if len(messageAPI.createReqs) != 0 {
		t.Fatalf("expected no create request, got %d", len(messageAPI.createReqs))
	}
}

func TestRunCLISendUsesBuildServiceAndWritesLogs(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "feishu.log")

	originalBuilder := newServiceBuilder
	defer func() { newServiceBuilder = originalBuilder }()
	newServiceBuilder = func(flags map[string]string, logger *log.Logger, messageLog io.Writer) (*Service, error) {
		return NewService(Options{
			Client:     &stubConnectClient{meta: &connectsvc.Meta{Name: DefaultName, Meta: `{"appId":"app","appSecret":"secret"}`}},
			Logger:     logger,
			MessageLog: messageLog,
		})
	}

	originalNewSender := newSender
	defer func() { newSender = originalNewSender }()
	newSender = func(loader MetaLoader, logger *log.Logger, messageLog io.Writer) *Sender {
		sender := NewSender(stubMetaLoader{
			meta: &connectsvc.Meta{Name: DefaultName, ChatID: "oc_test_chat"},
			cfg:  Config{AppID: "app", AppSecret: "secret"},
		}, logger, messageLog)
		sender.apis = LarkAPISet{
			Message: &stubMessageAPI{
				replyResps: []*larkim.ReplyMessageResp{
					{
						CodeError: larkcore.CodeError{Code: 0},
						Data:      &larkim.ReplyMessageRespData{MessageId: stringPtr("om_text_1")},
					},
				},
			},
			Image: &stubImageAPI{},
			File:  &stubFileAPI{},
		}
		return sender
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := RunCLI([]string{
		"send",
		"--message", latestConnectRequestMessageJSON("om_origin"),
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
	if !strings.Contains(logText, `做了：开始准备发送飞书消息`) {
		t.Fatalf("expected send request log in feishu.log, got: %s", logText)
	}
	if !strings.Contains(logText, `做了：确认了这条飞书消息要回复到哪里`) {
		t.Fatalf("expected send parse log in feishu.log, got: %s", logText)
	}
	if !strings.Contains(logText, `做了：成功发出了飞书消息`) {
		t.Fatalf("expected send result log in feishu.log, got: %s", logText)
	}
}

func TestRunCLIInitUsesSameSendFlowAndWritesLogs(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "feishu.log")

	originalBuilder := newServiceBuilder
	defer func() { newServiceBuilder = originalBuilder }()
	newServiceBuilder = func(flags map[string]string, logger *log.Logger, messageLog io.Writer) (*Service, error) {
		return NewService(Options{
			Client:     &stubConnectClient{meta: &connectsvc.Meta{Name: DefaultName, Meta: `{"appId":"app","appSecret":"secret"}`}},
			Logger:     logger,
			MessageLog: messageLog,
		})
	}

	originalNewSender := newSender
	defer func() { newSender = originalNewSender }()
	newSender = func(loader MetaLoader, logger *log.Logger, messageLog io.Writer) *Sender {
		sender := NewSender(stubMetaLoader{
			meta: &connectsvc.Meta{Name: DefaultName, ChatID: "oc_test_chat"},
			cfg:  Config{AppID: "app", AppSecret: "secret"},
		}, logger, messageLog)
		sender.apis = LarkAPISet{
			Message: &stubMessageAPI{
				replyResps: []*larkim.ReplyMessageResp{
					{
						CodeError: larkcore.CodeError{Code: 0},
						Data:      &larkim.ReplyMessageRespData{MessageId: stringPtr("om_text_init")},
					},
				},
			},
			Image: &stubImageAPI{},
			File:  &stubFileAPI{},
		}
		return sender
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := RunCLI([]string{
		"init",
		"--message", latestConnectRequestMessageJSON("om_origin"),
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
		t.Fatalf("expected init request log in feishu.log, got: %s", logText)
	}
	if !strings.Contains(logText, `做了：确认了这条飞书消息要回复到哪里`) {
		t.Fatalf("expected init parse log in feishu.log, got: %s", logText)
	}
	if !strings.Contains(logText, `做了：成功发出了飞书消息`) {
		t.Fatalf("expected init result log in feishu.log, got: %s", logText)
	}
}

func TestRunCLISendInvalidMessageWritesFailureLog(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "feishu.log")

	originalBuilder := newServiceBuilder
	defer func() { newServiceBuilder = originalBuilder }()
	newServiceBuilder = func(flags map[string]string, logger *log.Logger, messageLog io.Writer) (*Service, error) {
		return NewService(Options{
			Client:     &stubConnectClient{meta: &connectsvc.Meta{Name: DefaultName, Meta: `{"appId":"app","appSecret":"secret"}`}},
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
	if !strings.Contains(logText, `做了：开始准备发送飞书消息`) {
		t.Fatalf("expected request log in feishu.log, got: %s", logText)
	}
	if !strings.Contains(logText, `原因：message.messageId not found`) {
		t.Fatalf("expected failure log in feishu.log, got: %s", logText)
	}
}

func TestBuildFeishuMarkdownCardContent(t *testing.T) {
	payload, err := buildFeishuMarkdownCardContent("**hello**\n[world](https://example.com)")
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Schema string `json:"schema"`
		Body   struct {
			Elements []map[string]string `json:"elements"`
		} `json:"body"`
	}
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("unmarshal markdown payload: %v raw=%s", err, payload)
	}
	if decoded.Schema != "2.0" {
		t.Fatalf("expected schema=2.0, got %q", decoded.Schema)
	}
	if len(decoded.Body.Elements) != 1 {
		t.Fatalf("unexpected elements: %+v", decoded.Body.Elements)
	}
	if decoded.Body.Elements[0]["tag"] != "markdown" || decoded.Body.Elements[0]["content"] != "**hello**\n[world](https://example.com)" {
		t.Fatalf("unexpected markdown element: %+v", decoded.Body.Elements[0])
	}
}

func TestRunCLISendPassesDownloadDirToSender(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "nested", "feishu.log")

	originalBuilder := newServiceBuilder
	defer func() { newServiceBuilder = originalBuilder }()
	newServiceBuilder = func(flags map[string]string, logger *log.Logger, messageLog io.Writer) (*Service, error) {
		return NewService(Options{
			Client:     &stubConnectClient{meta: &connectsvc.Meta{Name: DefaultName, Meta: `{"appId":"app","appSecret":"secret"}`}},
			Logger:     logger,
			MessageLog: messageLog,
		})
	}

	originalNewSender := newSender
	defer func() { newSender = originalNewSender }()
	var captured *Sender
	newSender = func(loader MetaLoader, logger *log.Logger, messageLog io.Writer) *Sender {
		sender := NewSender(stubMetaLoader{
			meta: &connectsvc.Meta{Name: DefaultName, ChatID: "oc_test_chat"},
			cfg:  Config{AppID: "app", AppSecret: "secret"},
		}, logger, messageLog)
		sender.apis = LarkAPISet{
			Message: &stubMessageAPI{
				replyResps: []*larkim.ReplyMessageResp{
					{
						CodeError: larkcore.CodeError{Code: 0},
						Data:      &larkim.ReplyMessageRespData{MessageId: stringPtr("om_text_1")},
					},
				},
			},
			Image: &stubImageAPI{},
			File:  &stubFileAPI{},
		}
		sender.now = func() time.Time { return time.Unix(100, 0) }
		captured = sender
		return sender
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := RunCLI([]string{
		"send",
		"--message", latestConnectRequestMessageJSON("om_origin"),
		"--content", "hello",
		"--log-file", logPath,
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("send exit code = %d, stderr = %s", code, stderr.String())
	}
	if captured == nil {
		t.Fatal("expected sender to be created")
	}
	if captured.downloadDir != downloadDirForLog(logPath) {
		t.Fatalf("download dir = %q, want %q", captured.downloadDir, downloadDirForLog(logPath))
	}
}

func TestExtractReplyMessageID(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
		wantErr string
	}{
		{
			name:    "latest connect request rawRequest",
			message: latestConnectRequestMessageJSON("om_latest"),
			want:    "om_latest",
		},
		{
			name:    "raw feishu event unsupported",
			message: `{"schema":"2.0","event":{"message":{"message_id":"om_evt","content":"{\"text\":\"hello\"}","message_type":"text"}}}`,
			wantErr: "message.messageId not found",
		},
		{
			name:    "invalid json",
			message: `not-json`,
			wantErr: "message must be valid json",
		},
		{
			name:    "missing latest protocol id",
			message: `{"id":1,"rawRequest":"{\"source\":\"feishu\",\"receivedAt\":\"2026-05-22T00:00:00+08:00\",\"message\":{\"chatId\":\"oc_test_chat\"},\"pending\":[],\"groupedBy\":\"chat_id\",\"windowSecs\":30,\"expireSecs\":600}"}`,
			wantErr: "message.messageId not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractReplyMessageID(tt.message)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("message id = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCLIConnectClientInvokesConnectCommands(t *testing.T) {
	runner := &stubRunner{}
	client := &CLIConnectClient{
		Binary:   "/tmp/connect",
		BaseArgs: []string{"--db", "/tmp/data", "--agent-dir", "/tmp/agents"},
		Runner:   runner,
	}
	ctx := context.Background()

	runner.outputs = append(runner.outputs,
		[]byte(`{"id":1,"name":"feishu","meta":"{\"mode\":\"mock\"}","stream":true,"callback":"./feishu","agentId":"A","chatId":"","model":"OpenAI","thinking":false,"createdAt":"2026-05-04T00:00:00Z","updatedAt":"2026-05-04T00:00:00Z"}`),
		[]byte(`{"id":9,"name":"feishu","request":"{}","status":0,"createdAt":"2026-05-04T00:00:01Z"}`),
	)

	meta, err := client.GetMeta(ctx, "feishu")
	if err != nil {
		t.Fatal(err)
	}
	if meta.Name != "feishu" {
		t.Fatalf("unexpected meta: %+v", meta)
	}

	request, err := client.AddRequest(ctx, connectsvc.RequestInput{
		Name:           "feishu",
		Request:        `{"x":1}`,
		Content:        `{"x":1}`,
		ExternalID:     "ext-1",
		RawRequest:     `{"raw":true}`,
		Original:       `{"raw":true}`,
		ResponseSchema: `{"type":"object"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if request.ID != 9 {
		t.Fatalf("unexpected request: %+v", request)
	}

	if len(runner.calls) != 2 {
		t.Fatalf("expected 2 runner calls, got %d", len(runner.calls))
	}
	if got := strings.Join(runner.calls[0], " "); !strings.Contains(got, "meta-get --key feishu") {
		t.Fatalf("unexpected first call: %s", got)
	}
	if got := strings.Join(runner.calls[1], " "); !strings.Contains(got, "add-request --key feishu --content {\"x\":1}") {
		t.Fatalf("unexpected second call: %s", got)
	}
	if got := strings.Join(runner.calls[1], " "); !strings.Contains(got, "--external-id ext-1") || !strings.Contains(got, "--original {\"raw\":true}") {
		t.Fatalf("expected external-id/original in second call: %s", got)
	}
	if got := strings.Join(runner.calls[1], " "); !strings.Contains(got, "--schema {\"type\":\"object\"}") {
		t.Fatalf("expected schema in second call: %s", got)
	}
}

func TestCLIConnectClientSupportsPrefixedConnectCommands(t *testing.T) {
	runner := &stubRunner{}
	client := &CLIConnectClient{
		Binary: "/tmp/integration",
		Prefix: []string{"connect"},
		Runner: runner,
	}

	runner.outputs = append(runner.outputs, []byte(`{"id":1,"name":"feishu","meta":"{}","stream":true,"callback":"./feishu","agentId":"A","chatId":"","model":"OpenAI","thinking":false,"createdAt":"2026-05-04T00:00:00Z","updatedAt":"2026-05-04T00:00:00Z"}`))

	if _, err := client.GetMeta(context.Background(), "feishu"); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 runner call, got %d", len(runner.calls))
	}
	if got := strings.Join(runner.calls[0], " "); !strings.Contains(got, "/tmp/integration connect meta-get --key feishu") {
		t.Fatalf("unexpected prefixed call: %s", got)
	}
}

func TestCLIConnectClientGetMetaDoesNotNormalizeDisplayNameToKey(t *testing.T) {
	runner := &stubRunner{}
	client := &CLIConnectClient{
		Binary: "connect",
		Runner: runner,
	}

	runner.outputs = append(runner.outputs, []byte(`{"id":1,"name":"飞书","meta":"{}","stream":true,"callback":"./feishu","agentId":"A","chatId":"","model":"OpenAI","thinking":false,"createdAt":"2026-05-04T00:00:00Z","updatedAt":"2026-05-04T00:00:00Z"}`))

	if _, err := client.GetMeta(context.Background(), DefaultDisplayName); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 runner call, got %d", len(runner.calls))
	}
	if got := strings.Join(runner.calls[0], " "); !strings.Contains(got, "meta-get --key 飞书") {
		t.Fatalf("unexpected call: %s", got)
	}
}

func TestCLIConnectClientGetMetaAcceptsObjectMeta(t *testing.T) {
	runner := &stubRunner{}
	client := &CLIConnectClient{
		Binary: "connect",
		Runner: runner,
	}

	runner.outputs = append(runner.outputs, []byte(`{"key":"feishu","name":"飞书","meta":{"appId":"app","appSecret":"secret"},"stream":true,"callback":"./feishu","agentId":"A","chatId":"","model":"OpenAI","thinking":false,"createdAt":"2026-05-04T00:00:00Z","updatedAt":"2026-05-04T00:00:00Z"}`))

	meta, err := client.GetMeta(context.Background(), DefaultName)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Meta != `{"appId":"app","appSecret":"secret"}` {
		t.Fatalf("meta.Meta = %q", meta.Meta)
	}
}

func TestBuildServiceDoesNotPassDBOrAgentDirToConnectCLI(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	svc, err := buildService(map[string]string{
		"connect-bin":   "/tmp/connect",
		"addr":          "http://127.0.0.1:18081",
		"db":            "/tmp/data",
		"agent-dir":     "/tmp/agents",
		"connect-cache": "10000",
	}, logger, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	client, ok := svc.client.(*CLIConnectClient)
	if !ok {
		t.Fatalf("unexpected client type: %T", svc.client)
	}
	got := strings.Join(client.BaseArgs, " ")
	if strings.Contains(got, "--db") {
		t.Fatalf("db should not be passed through to FEISHU connect client: %s", got)
	}
	if strings.Contains(got, "--agent-dir") {
		t.Fatalf("agent-dir should not be passed through to FEISHU connect client: %s", got)
	}
	if !strings.Contains(got, "--addr http://127.0.0.1:18081") {
		t.Fatalf("addr should still be passed through: %s", got)
	}
}

func TestBuildServiceAddsConnectPrefixForProxyBinary(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	svc, err := buildService(map[string]string{
		"connect-bin": "/tmp/proxy",
	}, logger, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	client, ok := svc.client.(*CLIConnectClient)
	if !ok {
		t.Fatalf("unexpected client type: %T", svc.client)
	}
	if client.Binary != "/tmp/proxy" {
		t.Fatalf("binary = %q", client.Binary)
	}
	if strings.Join(client.Prefix, " ") != "connect" {
		t.Fatalf("prefix = %v, want [connect]", client.Prefix)
	}
}

func TestResolveConnectBinaryFallsBackToSiblingConnect(t *testing.T) {
	tempDir := t.TempDir()
	connectPath := filepath.Join(tempDir, "connect")
	if err := os.WriteFile(connectPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	originalExecutable := osExecutableFn
	defer func() { osExecutableFn = originalExecutable }()
	osExecutableFn = func() (string, error) {
		return filepath.Join(tempDir, "feishu"), nil
	}

	got := resolveConnectBinary(map[string]string{})
	if got != connectPath {
		t.Fatalf("binary = %q, want %q", got, connectPath)
	}
	if prefix := resolveConnectPrefix(map[string]string{}); len(prefix) != 0 {
		t.Fatalf("prefix = %v, want empty", prefix)
	}
}

func TestRunCLIForegroundWithMockMeta(t *testing.T) {
	client := &stubConnectClient{
		meta: &connectsvc.Meta{
			Name: DefaultName,
			Meta: `{
				"mode":"mock",
				"heartbeatIntervalSec":1,
				"heartbeatTimeoutSec":1,
				"reconnectDelayMs":10,
				"mockHeartbeatMs":50,
				"mockMessages":[{"delayMs":10,"messageId":"msg-1","chatId":"chat-1","content":"ping"}]
			}`,
		},
	}
	originalBuilder := newServiceBuilder
	defer func() { newServiceBuilder = originalBuilder }()
	newServiceBuilder = func(flags map[string]string, logger *log.Logger, messageLog io.Writer) (*Service, error) {
		return NewService(Options{
			Client:            client,
			Logger:            logger,
			MessageLog:        messageLog,
			HeartbeatInterval: 50 * time.Millisecond,
			HeartbeatTimeout:  150 * time.Millisecond,
			ReconnectDelay:    10 * time.Millisecond,
		})
	}

	tempDir := t.TempDir()
	pidFile := tempDir + "/feishu.pid"
	logFile := tempDir + "/feishu.log"

	ctx, cancel := context.WithCancel(context.Background())
	originalSignalContext := signalContextFn
	defer func() { signalContextFn = originalSignalContext }()
	signalContextFn = func() (context.Context, context.CancelFunc) { return ctx, cancel }

	done := make(chan int, 1)
	go func() {
		done <- RunCLI([]string{"start", "--foreground", "true", "--pid-file", pidFile, "--log-file", logFile}, io.Discard, io.Discard)
	}()

	waitUntil(t, time.Second, func() bool {
		return len(client.Requests()) == 1
	})
	cancel()

	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("unexpected exit code: %d", code)
		}
	case <-time.After(time.Second):
		t.Fatal("foreground run did not stop")
	}

	logData, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	logText := string(logData)
	if !strings.Contains(logText, "做了：启动前检查了飞书插件连接配置") {
		t.Fatalf("expected startup-connect log, got: %s", logText)
	}
}

func TestLogStartupConnectParamsIncludesPrefixedCommand(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := log.New(buf, "", 0)

	logStartupConnectParams(logger, map[string]string{
		"name":        "feishu",
		"connect-bin": "/tmp/integration",
		"addr":        "http://127.0.0.1:18081",
		"port":        "18081",
	})

	text := buf.String()
	if !strings.Contains(text, "stage=startup-connect") {
		t.Fatalf("unexpected startup log: %s", text)
	}
	if !strings.Contains(text, "/tmp/integration connect meta-get --key feishu") {
		t.Fatalf("expected prefixed command in startup log: %s", text)
	}
	if !strings.Contains(text, "--addr http://127.0.0.1:18081") {
		t.Fatalf("expected addr in startup log: %s", text)
	}
	if !strings.Contains(text, "--port 18081") {
		t.Fatalf("expected port in startup log: %s", text)
	}
}

func TestStartBackgroundRestartsExistingProcess(t *testing.T) {
	tempDir := t.TempDir()
	pidFile := tempDir + "/feishu.pid"
	logFile := tempDir + "/feishu.log"
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0o644); err != nil {
		t.Fatal(err)
	}

	signaled := false
	originalBuilder := newServiceBuilder
	originalExecCommand := execCommand
	originalFindProcess := findProcess
	originalValidate := validateFeishuConfigFn
	defer func() { newServiceBuilder = originalBuilder }()
	defer func() { execCommand = originalExecCommand }()
	defer func() { findProcess = originalFindProcess }()
	defer func() { validateFeishuConfigFn = originalValidate }()
	newServiceBuilder = func(flags map[string]string, logger *log.Logger, messageLog io.Writer) (*Service, error) {
		return &Service{
			name: DefaultName,
			client: &stubConnectClient{
				meta: &connectsvc.Meta{
					Name: DefaultName,
					Meta: `{"appId":"ok-app","appSecret":"ok-secret"}`,
				},
			},
			logger:     logger,
			messageLog: messageLog,
		}, nil
	}
	validateFeishuConfigFn = func(ctx context.Context, cfg Config) error { return nil }
	findProcess = func(pid int) (processHandle, error) {
		return stubProcessHandle{
			signalFn: func(sig os.Signal) error {
				if sig == syscall.Signal(0) {
					return nil
				}
				if sig == syscall.SIGTERM {
					signaled = true
					_ = os.Remove(pidFile)
					return nil
				}
				return nil
			},
		}, nil
	}
	execCommand = func(name string, args ...string) *exec.Cmd {
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
	if !signaled {
		t.Fatalf("expected existing process to be stopped before restart")
	}
	if _, ok := runningPID(pidFile); !ok {
		t.Fatalf("expected restarted process recorded in pid file")
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
	pidFile := filepath.Join(tempDir, "feishu.pid")
	logFile := filepath.Join(tempDir, "feishu.log")
	recorded := make([]string, 0, 8)

	originalBuilder := newServiceBuilder
	originalExecCommand := execCommand
	originalFindProcess := findProcess
	originalExecutable := osExecutableFn
	originalValidate := validateFeishuConfigFn
	defer func() { newServiceBuilder = originalBuilder }()
	defer func() { execCommand = originalExecCommand }()
	defer func() { findProcess = originalFindProcess }()
	defer func() { osExecutableFn = originalExecutable }()
	defer func() { validateFeishuConfigFn = originalValidate }()

	newServiceBuilder = func(flags map[string]string, logger *log.Logger, messageLog io.Writer) (*Service, error) {
		return &Service{
			name: DefaultName,
			client: &stubConnectClient{
				meta: &connectsvc.Meta{
					Name: DefaultName,
					Meta: `{"appId":"ok-app","appSecret":"ok-secret"}`,
				},
			},
			logger:     logger,
			messageLog: messageLog,
		}, nil
	}
	osExecutableFn = func() (string, error) { return "/tmp/feishu", nil }
	validateFeishuConfigFn = func(ctx context.Context, cfg Config) error { return nil }
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
	if !strings.Contains(got, "/tmp/feishu start") {
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
	pidFile := filepath.Join(tempDir, "feishu.pid")
	logFile := filepath.Join(tempDir, "feishu.log")
	execCalled := false

	originalBuilder := newServiceBuilder
	originalExecCommand := execCommand
	originalValidate := validateFeishuConfigFn
	defer func() { newServiceBuilder = originalBuilder }()
	defer func() { execCommand = originalExecCommand }()
	defer func() { validateFeishuConfigFn = originalValidate }()

	newServiceBuilder = func(flags map[string]string, logger *log.Logger, messageLog io.Writer) (*Service, error) {
		return &Service{
			name: DefaultName,
			client: &stubConnectClient{
				meta: &connectsvc.Meta{
					Name: DefaultName,
					Meta: `{"appId":"bad-app","appSecret":"bad-secret"}`,
				},
			},
			logger:     logger,
			messageLog: messageLog,
		}, nil
	}
	validateFeishuConfigFn = func(ctx context.Context, cfg Config) error {
		if cfg.AppID != "bad-app" {
			t.Fatalf("cfg.AppID = %q", cfg.AppID)
		}
		return errors.New("invalid feishu credentials")
	}
	execCommand = func(name string, args ...string) *exec.Cmd {
		execCalled = true
		return exec.Command("sh", "-c", "exit 0")
	}

	_, err := startBackground(map[string]string{
		"pid-file": pidFile,
		"log-file": logFile,
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "invalid feishu credentials") {
		t.Fatalf("err = %v, want invalid feishu credentials", err)
	}
	if execCalled {
		t.Fatalf("background process should not start after failed validation")
	}
}

func TestParseIncomingMessageExtractsTextAndTimestamp(t *testing.T) {
	raw := []byte(`{"schema":"2.0","header":{"event_id":"evt_1","create_time":"1777890990994"},"event":{"message":{"chat_id":"chat_1","content":"{\"text\":\"你好\"}","message_id":"msg_1","message_type":"text","create_time":"1777890990639"},"sender":{"sender_id":{"open_id":"ou_1"},"sender_type":"user"}}}`)
	result := parseIncomingMessage(raw, "")
	msg := result.message
	if !msg.Parsed {
		t.Fatalf("message should be parsed: %+v", msg)
	}
	if msg.Content != "你好" {
		t.Fatalf("content = %q", msg.Content)
	}
	if msg.MessageAtMS != 1777890990994 {
		t.Fatalf("messageAtMS = %d", msg.MessageAtMS)
	}
	if msg.ExternalID != md5Key(1777890990994, "你好") {
		t.Fatalf("externalID = %q", msg.ExternalID)
	}
}

func TestParseIncomingMessageExtractsImageAttachment(t *testing.T) {
	raw := []byte(`{"schema":"2.0","header":{"event_id":"evt_2","create_time":"1777899530913"},"event":{"message":{"chat_id":"chat_1","content":"{\"image_key\":\"img_v3_0211c_xxx\"}","message_id":"msg_2","message_type":"image","create_time":"1777899530590"},"sender":{"sender_id":{"open_id":"ou_1"},"sender_type":"user"}}}`)
	result := parseIncomingMessage(raw, "")
	if !result.message.Parsed {
		t.Fatalf("image message should be parsed: %+v", result.message)
	}
	if result.message.Content != "[image]" {
		t.Fatalf("content = %q", result.message.Content)
	}
	if len(result.attachments) != 1 || result.attachments[0].kind != "image" || result.attachments[0].key != "img_v3_0211c_xxx" {
		t.Fatalf("attachments = %+v", result.attachments)
	}
}

func TestServiceSkipsExpiredMessage(t *testing.T) {
	client := &stubConnectClient{
		meta: &connectsvc.Meta{
			Name: DefaultName,
			Meta: fmt.Sprintf(`{
				"mode":"mock",
				"heartbeatIntervalSec":1,
				"heartbeatTimeoutSec":1,
				"reconnectDelayMs":10,
				"mockHeartbeatMs":50,
				"mockMessages":[{"delayMs":10,"messageId":"msg-old","chatId":"chat-1","content":"old message"}]
			}`),
		},
	}
	loggerOut := &bytes.Buffer{}
	messageLog := &bytes.Buffer{}
	svc, err := NewService(Options{
		Client:            client,
		Logger:            log.New(loggerOut, "", 0),
		MessageLog:        messageLog,
		HeartbeatInterval: 50 * time.Millisecond,
		HeartbeatTimeout:  150 * time.Millisecond,
		ReconnectDelay:    10 * time.Millisecond,
		SessionFactory: stubSessionFactory{session: &stubSession{
			events: []SessionEvent{{
				Type: SessionEventMessage,
				At:   time.Now(),
				Message: IncomingMessage{
					MessageID:   "msg-old",
					ExternalID:  "ext-old",
					ChatID:      "chat-1",
					MessageAtMS: time.Now().Add(-31 * time.Minute).UnixMilli(),
					Content:     "old message",
					Parsed:      true,
				},
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_ = svc.Run(ctx)

	if len(client.Requests()) != 0 {
		t.Fatalf("expired message should be skipped, got requests: %+v", client.Requests())
	}
	if !strings.Contains(loggerOut.String(), "跳过了一条过期的飞书消息") {
		t.Fatalf("expected expired log, got: %s", loggerOut.String())
	}
	if !strings.Contains(messageLog.String(), "跳过了一条过期的飞书消息") {
		t.Fatalf("expected expired message log, got: %s", messageLog.String())
	}
}

func TestServiceSkipsBotMessage(t *testing.T) {
	client := &stubConnectClient{
		meta: &connectsvc.Meta{
			Name: DefaultName,
			Meta: `{"mode":"mock","heartbeatIntervalSec":1,"heartbeatTimeoutSec":1,"reconnectDelayMs":10,"mockHeartbeatMs":50}`,
		},
	}
	loggerOut := &bytes.Buffer{}
	messageLog := &bytes.Buffer{}
	svc, err := NewService(Options{
		Client:            client,
		Logger:            log.New(loggerOut, "", 0),
		MessageLog:        messageLog,
		HeartbeatInterval: 50 * time.Millisecond,
		HeartbeatTimeout:  150 * time.Millisecond,
		ReconnectDelay:    10 * time.Millisecond,
		SessionFactory: stubSessionFactory{session: &stubSession{
			events: []SessionEvent{{
				Type: SessionEventMessage,
				At:   time.Now(),
				Message: IncomingMessage{
					MessageID:   "msg-bot",
					ChatID:      "chat-1",
					MessageAtMS: time.Now().UnixMilli(),
					SenderID:    "bot-1",
					SenderType:  "bot",
					MessageType: "text",
					Content:     "开始执行",
					RawContent:  `{"text":"开始执行"}`,
					Raw:         `{"text":"开始执行"}`,
					ExternalID:  md5Key(time.Now().UnixMilli(), "开始执行"),
					Parsed:      true,
				},
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
	defer cancel()
	if err := svc.Run(ctx); err != nil {
		t.Fatal(err)
	}

	if len(client.Requests()) != 0 {
		t.Fatalf("expected bot message to be skipped, got requests=%+v", client.Requests())
	}
	if !strings.Contains(messageLog.String(), "跳过了一条插件自己发出的飞书消息") {
		t.Fatalf("expected self-message log, got: %s", messageLog.String())
	}
}

func TestServiceSkipsInvalidMessage(t *testing.T) {
	client := &stubConnectClient{
		meta: &connectsvc.Meta{Name: DefaultName, Meta: `{"mode":"mock"}`},
	}
	loggerOut := &bytes.Buffer{}
	messageLog := &bytes.Buffer{}
	svc, err := NewService(Options{
		Client:            client,
		Logger:            log.New(loggerOut, "", 0),
		MessageLog:        messageLog,
		HeartbeatInterval: 50 * time.Millisecond,
		HeartbeatTimeout:  150 * time.Millisecond,
		ReconnectDelay:    10 * time.Millisecond,
		SessionFactory: stubSessionFactory{session: &stubSession{
			events: []SessionEvent{{
				Type: SessionEventMessage,
				At:   time.Now(),
				Message: IncomingMessage{
					Raw:    `{"broken":true}`,
					Parsed: false,
				},
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_ = svc.Run(ctx)

	if len(client.Requests()) != 0 {
		t.Fatalf("invalid message should be skipped, got requests: %+v", client.Requests())
	}
	if !strings.Contains(loggerOut.String(), "跳过了一条无法识别的飞书消息") {
		t.Fatalf("expected invalid payload log, got: %s", loggerOut.String())
	}
	if !strings.Contains(messageLog.String(), "跳过了一条无法识别的飞书消息") {
		t.Fatalf("expected invalid payload message log, got: %s", messageLog.String())
	}
}

type stubConnectClient struct {
	meta       *connectsvc.Meta
	metaByName map[string]*connectsvc.Meta
	metaCalls  []string
	getMetaN   int
	mu         sync.Mutex
	requests   []connectsvc.RequestInput
}

func (s *stubConnectClient) GetMeta(_ context.Context, name string) (*connectsvc.Meta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getMetaN++
	s.metaCalls = append(s.metaCalls, name)
	if s.metaByName != nil {
		meta, ok := s.metaByName[name]
		if !ok || meta == nil {
			return nil, errors.New("meta missing")
		}
		copyMeta := *meta
		return &copyMeta, nil
	}
	if s.meta == nil {
		return nil, errors.New("meta missing")
	}
	if s.meta.Name == "" {
		s.meta.Name = name
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

func (s *stubConnectClient) GetMetaCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getMetaN
}

type stubRunner struct {
	mu      sync.Mutex
	outputs [][]byte
	calls   [][]string
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
	return s.meta, s.cfg, nil
}

type stubMessageAPI struct {
	createReqs []*larkim.CreateMessageReq
	replyReqs  []*larkim.ReplyMessageReq
	createJSON []string
	replyJSON  []string
	createResp *larkim.CreateMessageResp
	replyResps []*larkim.ReplyMessageResp
	err        error
}

func (s *stubMessageAPI) Create(_ context.Context, req *larkim.CreateMessageReq, _ ...larkcore.RequestOptionFunc) (*larkim.CreateMessageResp, error) {
	s.createReqs = append(s.createReqs, req)
	s.createJSON = append(s.createJSON, marshalSDKRequest(takeCreateBody(req)))
	if s.err != nil {
		return nil, s.err
	}
	if s.createResp == nil {
		return nil, errors.New("missing create response")
	}
	return s.createResp, nil
}

func (s *stubMessageAPI) Reply(_ context.Context, req *larkim.ReplyMessageReq, _ ...larkcore.RequestOptionFunc) (*larkim.ReplyMessageResp, error) {
	s.replyReqs = append(s.replyReqs, req)
	s.replyJSON = append(s.replyJSON, marshalSDKRequest(takeReplyBody(req)))
	if s.err != nil {
		return nil, s.err
	}
	if len(s.replyResps) == 0 {
		return nil, errors.New("missing reply response")
	}
	resp := s.replyResps[0]
	s.replyResps = s.replyResps[1:]
	return resp, nil
}

type stubImageAPI struct {
	reqs []*larkim.CreateImageReq
	resp *larkim.CreateImageResp
	err  error
	errs []error
}

func (s *stubImageAPI) Create(_ context.Context, req *larkim.CreateImageReq, _ ...larkcore.RequestOptionFunc) (*larkim.CreateImageResp, error) {
	s.reqs = append(s.reqs, req)
	if len(s.errs) > 0 {
		err := s.errs[0]
		s.errs = s.errs[1:]
		if err != nil {
			return nil, err
		}
	}
	if s.err != nil {
		return nil, s.err
	}
	if s.resp == nil {
		return nil, errors.New("missing image response")
	}
	return s.resp, nil
}

type stubFileAPI struct {
	reqs []*larkim.CreateFileReq
	resp *larkim.CreateFileResp
	err  error
	errs []error
}

func (s *stubFileAPI) Create(_ context.Context, req *larkim.CreateFileReq, _ ...larkcore.RequestOptionFunc) (*larkim.CreateFileResp, error) {
	s.reqs = append(s.reqs, req)
	if len(s.errs) > 0 {
		err := s.errs[0]
		s.errs = s.errs[1:]
		if err != nil {
			return nil, err
		}
	}
	if s.err != nil {
		return nil, s.err
	}
	if s.resp == nil {
		return nil, errors.New("missing file response")
	}
	return s.resp, nil
}

func TestSenderRetriesTextSendBeforeSuccess(t *testing.T) {
	loader := stubMetaLoader{
		meta: &connectsvc.Meta{Name: "feishu", ChatID: "oc_test_chat", Meta: `{"appId":"app","appSecret":"secret"}`},
		cfg:  Config{AppID: "app", AppSecret: "secret"},
	}
	messageLog := &bytes.Buffer{}
	messageAPI := &stubMessageAPI{
		replyResps: []*larkim.ReplyMessageResp{
			{CodeError: larkcore.CodeError{Code: 230099}},
			{CodeError: larkcore.CodeError{Code: 230099}},
			{CodeError: larkcore.CodeError{Code: 0}, Data: &larkim.ReplyMessageRespData{MessageId: stringPtr("om_reply_text")}},
		},
	}
	sender := NewSender(loader, log.New(io.Discard, "", 0), messageLog)
	sender.apis = LarkAPISet{Message: messageAPI, Image: &stubImageAPI{}, File: &stubFileAPI{}}

	result, err := sender.Send(context.Background(), SendInput{
		Message: latestConnectRequestMessageJSON("om_origin"),
		Content: "重试文本",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || len(result.Sent) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(messageAPI.replyReqs) != 3 {
		t.Fatalf("reply attempts = %d, want 3", len(messageAPI.replyReqs))
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

type stubProcessHandle struct {
	signalFn func(sig os.Signal) error
}

func (p stubProcessHandle) Signal(sig os.Signal) error {
	if p.signalFn == nil {
		return nil
	}
	return p.signalFn(sig)
}

type stubSessionFactory struct {
	session Session
}

func (f stubSessionFactory) New(_ *connectsvc.Meta, _ Config, _ *log.Logger, _ string) (Session, error) {
	return f.session, nil
}

type stubSession struct {
	events []SessionEvent
}

func (s *stubSession) Events() <-chan SessionEvent {
	ch := make(chan SessionEvent, len(s.events))
	for _, event := range s.events {
		ch <- event
	}
	close(ch)
	return ch
}

func (s *stubSession) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (s *stubRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	call := append([]string{name}, args...)
	s.calls = append(s.calls, call)
	if len(s.outputs) == 0 {
		return nil, errors.New("no output configured")
	}
	out := s.outputs[0]
	s.outputs = s.outputs[1:]
	return out, nil
}

func waitUntil(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not reached")
}

func stringPtr(value string) *string {
	return &value
}

func latestConnectRequestMessageJSON(messageID string) string {
	envelope := map[string]any{
		"source":     DefaultName,
		"receivedAt": "2026-05-22T00:00:00+08:00",
		"message": map[string]any{
			"eventId":     "evt-" + messageID,
			"messageId":   messageID,
			"externalId":  "ext-" + messageID,
			"chatId":      "oc_test_chat",
			"messageAtMs": int64(1779377212951),
			"messageType": "text",
			"senderId":    "ou_test",
			"senderType":  "user",
			"content":     "hello",
			"rawContent":  `{"text":"hello"}`,
			"raw":         `{"schema":"2.0","event":{"message":{"message_id":"` + messageID + `","chat_id":"oc_test_chat","content":"{\"text\":\"hello\"}","message_type":"text"}}}`,
			"parsed":      true,
		},
		"pending": []map[string]any{
			{
				"externalId": "ext-" + messageID,
				"eventId":    "evt-" + messageID,
				"messageId":  messageID,
				"chatId":     "oc_test_chat",
				"senderId":   "ou_test",
				"senderType": "user",
				"raw":        `{"schema":"2.0","event":{"message":{"message_id":"` + messageID + `","chat_id":"oc_test_chat","content":"{\"text\":\"hello\"}","message_type":"text"}}}`,
				"rawContent": `{"text":"hello"}`,
				"type":       "text",
				"content":    "hello",
				"messageAt":  int64(1779377212951),
				"receivedAt": int64(1779377210954),
			},
		},
		"groupedBy":  "chat_id",
		"windowSecs": 30,
		"expireSecs": 600,
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

func assertPostReply(t *testing.T, raw string, wantText string) {
	t.Helper()
	var body struct {
		MsgType string `json:"msg_type"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatalf("unmarshal reply wrapper: %v raw=%s", err, raw)
	}
	if body.MsgType != "interactive" {
		t.Fatalf("expected msg_type=interactive, got %q raw=%s", body.MsgType, raw)
	}
	assertPostContent(t, body.Content, wantText)
}

func assertReplyMessageType(t *testing.T, raw, wantType string) {
	t.Helper()
	var body struct {
		MsgType string `json:"msg_type"`
	}
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatalf("unmarshal reply wrapper: %v raw=%s", err, raw)
	}
	if body.MsgType != wantType {
		t.Fatalf("expected msg_type=%s, got %q raw=%s", wantType, body.MsgType, raw)
	}
}

func assertPostContent(t *testing.T, raw, wantText string) {
	t.Helper()
	var decoded struct {
		Schema string `json:"schema"`
		Body   struct {
			Elements []map[string]string `json:"elements"`
		} `json:"body"`
	}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("unmarshal markdown content: %v raw=%s", err, raw)
	}
	if decoded.Schema != "2.0" {
		t.Fatalf("unexpected schema: %q", decoded.Schema)
	}
	if len(decoded.Body.Elements) != 1 {
		t.Fatalf("unexpected markdown elements: %+v", decoded.Body.Elements)
	}
	item := decoded.Body.Elements[0]
	if item["tag"] != "markdown" || item["content"] != wantText {
		t.Fatalf("unexpected markdown item: %+v wantText=%q", item, wantText)
	}
}

func takeReplyBody(req *larkim.ReplyMessageReq) any {
	if req == nil {
		return nil
	}
	return readUnexportedField(req, "apiReq", "Body")
}

func takeCreateBody(req *larkim.CreateMessageReq) any {
	if req == nil {
		return nil
	}
	return readUnexportedField(req, "apiReq", "Body")
}

func takeFileBody(req *larkim.CreateFileReq) any {
	if req == nil {
		return nil
	}
	return readUnexportedField(req, "apiReq", "Body")
}

func readUnexportedField(target any, fields ...string) any {
	value := reflect.ValueOf(target)
	for _, field := range fields {
		if !value.IsValid() {
			return nil
		}
		if value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return nil
			}
			value = value.Elem()
		}
		value = value.FieldByName(field)
	}
	if !value.IsValid() {
		return nil
	}
	return reflect.NewAt(value.Type(), unsafe.Pointer(value.UnsafeAddr())).Elem().Interface()
}

func marshalSDKRequest(body any) string {
	if body == nil {
		return ""
	}
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Sprintf(`{"marshal_error":%q}`, err.Error())
	}
	return string(data)
}

func assertUploadedFileType(t *testing.T, req *larkim.CreateFileReq, want string) {
	t.Helper()
	raw := marshalSDKRequest(takeFileBody(req))
	var body struct {
		FileType string `json:"file_type"`
	}
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatalf("unmarshal file body: %v raw=%s", err, raw)
	}
	if body.FileType != want {
		t.Fatalf("expected file_type=%s, got %q raw=%s", want, body.FileType, raw)
	}
}
