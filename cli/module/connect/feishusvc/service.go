package feishusvc

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"connect/connectsvc"
	lark "github.com/larksuite/oapi-sdk-go/v3"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

const (
	DefaultName              = "feishu"
	DefaultDisplayName       = "飞书"
	defaultAttachmentTimeout = 180 * time.Second
	defaultHeartbeatInterval = 60 * time.Second
	defaultHeartbeatTimeout  = 130 * time.Second
	defaultReconnectDelay    = 2 * time.Second
	defaultMockHeartbeat     = time.Second
	maxSendRetries           = 3
)

type ConnectClient interface {
	GetMeta(ctx context.Context, name string) (*connectsvc.Meta, error)
	AddRequest(ctx context.Context, input connectsvc.RequestInput) (*connectsvc.Request, error)
}

type MetaLoader interface {
	Load(ctx context.Context) (*connectsvc.Meta, Config, error)
}

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type ExecCommandRunner struct{}

func (ExecCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) == 0 {
			return nil, fmt.Errorf("%w", err)
		}
		return output, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

type CLIConnectClient struct {
	Binary   string
	Prefix   []string
	BaseArgs []string
	Runner   CommandRunner
}

func (c *CLIConnectClient) GetMeta(ctx context.Context, name string) (*connectsvc.Meta, error) {
	var out connectsvc.Meta
	key := normalizeMetaKey(name)
	if err := c.runJSON(ctx, &out, "meta-get", "--key", key); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *CLIConnectClient) AddRequest(ctx context.Context, input connectsvc.RequestInput) (*connectsvc.Request, error) {
	args := []string{
		"add-request",
		"--key", firstNonEmpty(input.Key, input.Name),
		"--content", firstNonEmpty(input.Content, input.Request),
	}
	if strings.TrimSpace(input.ExternalID) != "" {
		args = append(args, "--external-id", input.ExternalID)
	}
	if strings.TrimSpace(input.Artifacts) != "" {
		args = append(args, "--artifacts", input.Artifacts)
	}
	original := firstNonEmpty(input.Original, input.RawRequest)
	if strings.TrimSpace(original) != "" {
		args = append(args, "--original", original)
	}
	if strings.TrimSpace(input.CreatedAt) != "" {
		args = append(args, "--created", input.CreatedAt)
	}
	if strings.TrimSpace(input.ResponseSchema) != "" {
		args = append(args, "--schema", input.ResponseSchema)
	}
	var out connectsvc.Request
	if err := c.runJSON(ctx, &out, args...); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *CLIConnectClient) runJSON(ctx context.Context, dest any, args ...string) error {
	bin := strings.TrimSpace(c.Binary)
	if bin == "" {
		bin = "connect"
	}
	runner := c.Runner
	if runner == nil {
		runner = ExecCommandRunner{}
	}
	fullArgs := append([]string{}, c.Prefix...)
	fullArgs = append(fullArgs, args...)
	fullArgs = append(fullArgs, c.BaseArgs...)
	output, err := runner.Run(ctx, bin, fullArgs...)
	if err != nil {
		return fmt.Errorf("run %s %s failed: %w", bin, strings.Join(fullArgs, " "), err)
	}
	if err := json.Unmarshal(output, dest); err != nil {
		return fmt.Errorf("decode connect cli output: %w", err)
	}
	return nil
}

func normalizeMetaKey(name string) string {
	switch strings.TrimSpace(name) {
	case "", DefaultName:
		return DefaultName
	default:
		return strings.TrimSpace(name)
	}
}

type Options struct {
	Name              string
	Client            ConnectClient
	SessionFactory    SessionFactory
	Logger            *log.Logger
	MessageLog        io.Writer
	DownloadDir       string
	HeartbeatInterval time.Duration
	HeartbeatTimeout  time.Duration
	ReconnectDelay    time.Duration
}

type Service struct {
	name              string
	client            ConnectClient
	factory           SessionFactory
	logger            *log.Logger
	messageLog        io.Writer
	downloadDir       string
	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration
	reconnectDelay    time.Duration

	seenMu         sync.Mutex
	seenMessageIDs map[string]time.Time
}

type SessionFactory interface {
	New(meta *connectsvc.Meta, cfg Config, logger *log.Logger, downloadDir string) (Session, error)
}

type Session interface {
	Events() <-chan SessionEvent
	Run(ctx context.Context) error
}

type SessionEventType string

const (
	SessionEventHeartbeat SessionEventType = "heartbeat"
	SessionEventMessage   SessionEventType = "message"
	SessionEventConnected SessionEventType = "connected"
)

type SessionEvent struct {
	Type    SessionEventType
	Message IncomingMessage
	At      time.Time
}

type IncomingMessage struct {
	RequestID   string   `json:"requestId,omitempty"`
	EventID     string   `json:"eventId,omitempty"`
	MessageID   string   `json:"messageId,omitempty"`
	ExternalID  string   `json:"externalId,omitempty"`
	ChatID      string   `json:"chatId,omitempty"`
	MessageAtMS int64    `json:"messageAtMs,omitempty"`
	MessageType string   `json:"messageType,omitempty"`
	SenderID    string   `json:"senderId,omitempty"`
	SenderType  string   `json:"senderType,omitempty"`
	Content     string   `json:"content,omitempty"`
	RawContent  string   `json:"rawContent,omitempty"`
	Artifacts   []string `json:"artifacts,omitempty"`
	Raw         string   `json:"raw,omitempty"`
	Parsed      bool     `json:"parsed,omitempty"`
}

type PushPayload struct {
	Source     string          `json:"source"`
	ReceivedAt string          `json:"receivedAt"`
	Message    IncomingMessage `json:"message"`
}

type SendInput struct {
	Action  string
	Message string
	Content string
	Images  []string
	Files   []string
}

type normalizedSendPayload struct {
	Content string
	Images  []string
	Files   []string
}

type SendResult struct {
	Name      string       `json:"name"`
	Target    string       `json:"target"`
	MessageID string       `json:"messageId,omitempty"`
	Sent      []SentRecord `json:"sent"`
}

type SentRecord struct {
	Type        string `json:"type"`
	MessageID   string `json:"messageId,omitempty"`
	ParentID    string `json:"parentId,omitempty"`
	ThreadID    string `json:"threadId,omitempty"`
	ResourceKey string `json:"resourceKey,omitempty"`
	Path        string `json:"path,omitempty"`
}

type Config struct {
	AppID                 string        `json:"appId"`
	AppSecret             string        `json:"appSecret"`
	Mode                  string        `json:"mode,omitempty"`
	HeartbeatIntervalSec  int           `json:"heartbeatIntervalSec,omitempty"`
	HeartbeatTimeoutSec   int           `json:"heartbeatTimeoutSec,omitempty"`
	ReconnectDelayMS      int           `json:"reconnectDelayMs,omitempty"`
	MockHeartbeatMS       int           `json:"mockHeartbeatMs,omitempty"`
	MockDisconnectAfterMS int           `json:"mockDisconnectAfterMs,omitempty"`
	MockMessages          []MockMessage `json:"mockMessages,omitempty"`
}

var validateFeishuConfigFn = func(ctx context.Context, cfg Config) error {
	if strings.EqualFold(strings.TrimSpace(cfg.Mode), "mock") {
		return nil
	}
	client, err := newFeishuLarkClient(cfg)
	if err != nil {
		return err
	}
	resp, err := client.GetTenantAccessTokenBySelfBuiltApp(ctx, &larkcore.SelfBuiltTenantAccessTokenReq{
		AppID:     strings.TrimSpace(cfg.AppID),
		AppSecret: strings.TrimSpace(cfg.AppSecret),
	})
	if err != nil {
		return err
	}
	if resp == nil {
		return errors.New("empty feishu auth response")
	}
	if !resp.Success() {
		return fmt.Errorf("feishu auth failed code=%d msg=%s", resp.Code, strings.TrimSpace(resp.Msg))
	}
	return nil
}

func ValidateConfig(ctx context.Context, cfg Config) error {
	return validateFeishuConfigFn(ctx, cfg)
}

type MockMessage struct {
	DelayMS     int    `json:"delayMs,omitempty"`
	MessageID   string `json:"messageId,omitempty"`
	ChatID      string `json:"chatId,omitempty"`
	SenderID    string `json:"senderId,omitempty"`
	SenderType  string `json:"senderType,omitempty"`
	MessageType string `json:"messageType,omitempty"`
	MessageAtMS int64  `json:"messageAtMs,omitempty"`
	Content     string `json:"content"`
}

type MessageDownloader interface {
	DownloadImage(ctx context.Context, messageID, imageKey string) ([]byte, string, error)
	DownloadFile(ctx context.Context, messageID, fileKey string) ([]byte, string, error)
}

type attachmentRef struct {
	kind      string
	key       string
	messageID string
}

type parseResult struct {
	message     IncomingMessage
	attachments []attachmentRef
}

func NewService(opts Options) (*Service, error) {
	if opts.Client == nil {
		return nil, errors.New("connect client is required")
	}
	factory := opts.SessionFactory
	if factory == nil {
		factory = DefaultSessionFactory{}
	}
	logger := opts.Logger
	if logger == nil {
		logger = log.New(io.Discard, "", log.LstdFlags)
	}
	messageLog := opts.MessageLog
	if messageLog == nil {
		messageLog = io.Discard
	}
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		name = DefaultName
	}
	downloadDir := strings.TrimSpace(opts.DownloadDir)
	if downloadDir == "" {
		downloadDir = DefaultName
	}
	absDownloadDir, err := filepath.Abs(downloadDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absDownloadDir, 0o755); err != nil {
		return nil, fmt.Errorf("create feishu download dir: %w", err)
	}
	heartbeatInterval := opts.HeartbeatInterval
	if heartbeatInterval <= 0 {
		heartbeatInterval = defaultHeartbeatInterval
	}
	heartbeatTimeout := opts.HeartbeatTimeout
	if heartbeatTimeout <= 0 {
		heartbeatTimeout = defaultHeartbeatTimeout
	}
	reconnectDelay := opts.ReconnectDelay
	if reconnectDelay <= 0 {
		reconnectDelay = defaultReconnectDelay
	}
	return &Service{
		name:              name,
		client:            opts.Client,
		factory:           factory,
		logger:            logger,
		messageLog:        messageLog,
		downloadDir:       absDownloadDir,
		heartbeatInterval: heartbeatInterval,
		heartbeatTimeout:  heartbeatTimeout,
		reconnectDelay:    reconnectDelay,
		seenMessageIDs:    make(map[string]time.Time),
	}, nil
}

func (s *Service) Run(ctx context.Context) error {
	for {
		meta, cfg, err := s.loadConfig(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			s.logger.Printf("stage=load-config name=%s err=%v", s.name, err)
			if !sleepContext(ctx, s.reconnectDelay) {
				return nil
			}
			continue
		}
		if cfg.HeartbeatIntervalSec > 0 {
			s.heartbeatInterval = time.Duration(cfg.HeartbeatIntervalSec) * time.Second
		}
		if cfg.HeartbeatTimeoutSec > 0 {
			s.heartbeatTimeout = time.Duration(cfg.HeartbeatTimeoutSec) * time.Second
		}
		if cfg.ReconnectDelayMS > 0 {
			s.reconnectDelay = time.Duration(cfg.ReconnectDelayMS) * time.Millisecond
		}

		session, err := s.factory.New(meta, cfg, s.logger, s.downloadDir)
		if err != nil {
			s.logger.Printf("stage=create-session name=%s err=%v", s.name, err)
			if !sleepContext(ctx, s.reconnectDelay) {
				return nil
			}
			continue
		}
		s.logger.Printf("stage=connected name=%s callback=%s", meta.Name, meta.Callback)
		err = s.runSession(ctx, meta, session)
		if errors.Is(err, context.Canceled) {
			return nil
		}
		if err != nil {
			s.logger.Printf("stage=reconnect name=%s err=%v", s.name, err)
		}
		if !sleepContext(ctx, s.reconnectDelay) {
			return nil
		}
	}
}

func (s *Service) loadConfig(ctx context.Context) (*connectsvc.Meta, Config, error) {
	name := strings.TrimSpace(s.name)
	if name == "" {
		name = DefaultName
	}
	meta, err := s.client.GetMeta(ctx, name)
	if err != nil {
		return nil, Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal([]byte(meta.Meta), &cfg); err != nil {
		return nil, Config{}, fmt.Errorf("invalid feishu meta json: %w", err)
	}
	cfg.Mode = strings.ToLower(strings.TrimSpace(cfg.Mode))
	if cfg.Mode == "" {
		cfg.Mode = "feishu"
	}
	if cfg.Mode != "mock" {
		if strings.TrimSpace(cfg.AppID) == "" {
			return nil, Config{}, errors.New("meta.appId is required")
		}
		if strings.TrimSpace(cfg.AppSecret) == "" {
			return nil, Config{}, errors.New("meta.appSecret is required")
		}
	}
	return meta, cfg, nil
}

func (s *Service) Load(ctx context.Context) (*connectsvc.Meta, Config, error) {
	return s.loadConfig(ctx)
}

func (s *Service) ValidateStartup(ctx context.Context) error {
	_, cfg, err := s.loadConfig(ctx)
	if err != nil {
		return err
	}
	return ValidateConfig(ctx, cfg)
}

func (s *Service) runSession(ctx context.Context, meta *connectsvc.Meta, session Session) error {
	sessionCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- session.Run(sessionCtx)
	}()

	lastBeat := time.Now()
	ticker := time.NewTicker(s.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			cancel()
			<-errCh
			return nil
		case err := <-errCh:
			if err == nil || errors.Is(err, context.Canceled) {
				return err
			}
			return err
		case event, ok := <-session.Events():
			if !ok {
				cancel()
				if err := <-errCh; err != nil && !errors.Is(err, context.Canceled) {
					return err
				}
				return errors.New("session closed")
			}
			if event.At.IsZero() {
				event.At = time.Now()
			}
			if event.Type == SessionEventHeartbeat || event.Type == SessionEventConnected {
				lastBeat = event.At
			}
			if event.Type != SessionEventMessage {
				continue
			}
			lastBeat = event.At
			if s.shouldDropMessage(event.Message) {
				continue
			}
			if s.shouldSkip(event.Message.ExternalID) {
				continue
			}
			if err := s.pushRequest(ctx, meta.Name, event.Message); err != nil {
				s.logger.Printf("stage=push-request name=%s external_id=%s err=%v", meta.Name, event.Message.ExternalID, err)
			}
		case <-ticker.C:
			if time.Since(lastBeat) <= s.heartbeatTimeout {
				continue
			}
			cancel()
			<-errCh
			return fmt.Errorf("heartbeat timeout after %s", s.heartbeatTimeout)
		}
	}
}

func (s *Service) pushRequest(ctx context.Context, name string, message IncomingMessage) error {
	payload := PushPayload{
		Source:     DefaultName,
		ReceivedAt: time.Now().Format(time.RFC3339),
		Message:    message,
	}
	s.writeMessageLog(payload.ReceivedAt, compactMessageLog(message))
	_, err := s.client.AddRequest(ctx, connectsvc.RequestInput{
		Key:            name,
		ExternalID:     message.ExternalID,
		Name:           name,
		Content:        message.Content,
		Request:        message.Content,
		Artifacts:      strings.Join(message.Artifacts, ","),
		Original:       message.Raw,
		RawRequest:     message.Raw,
		ResponseSchema: ResponseSchemaJSON(),
		CreatedAt:      buildMessageCreatedAt(message),
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) shouldDropMessage(message IncomingMessage) bool {
	if !message.Parsed {
		s.logSkip(fmt.Sprintf("用参数：内容摘要=%s，做了：跳过了一条无法识别的飞书消息", compactMessageLog(message)))
		return true
	}
	if shouldIgnoreSenderType(message.SenderType) {
		s.logSkip(fmt.Sprintf("用参数：发送方类型=%s；外部编号=%s；内容摘要=%s，做了：跳过了一条插件自己发出的飞书消息", strings.TrimSpace(message.SenderType), message.ExternalID, compactMessageLog(message)))
		return true
	}
	if message.MessageAtMS > 0 {
		messageAt := time.UnixMilli(message.MessageAtMS)
		if time.Since(messageAt) > 30*time.Minute {
			s.logSkip(fmt.Sprintf("用参数：外部编号=%s；消息时间=%d；内容摘要=%s，做了：跳过了一条过期的飞书消息", message.ExternalID, message.MessageAtMS, compactMessageLog(message)))
			return true
		}
	}
	return false
}

func shouldIgnoreSenderType(senderType string) bool {
	switch strings.ToLower(strings.TrimSpace(senderType)) {
	case "bot", "app", "application", "assistant", "robot":
		return true
	default:
		return false
	}
}

func buildMessageCreatedAt(message IncomingMessage) string {
	if message.MessageAtMS <= 0 {
		return ""
	}
	return strconv.FormatInt(message.MessageAtMS, 10)
}

func compactMessageLog(message IncomingMessage) string {
	content := strings.TrimSpace(message.Content)
	if content == "" {
		content = strings.TrimSpace(message.RawContent)
	}
	if content == "" {
		content = strings.TrimSpace(message.Raw)
	}
	content = strings.ReplaceAll(content, "\n", "\\n")
	content = strings.ReplaceAll(content, "\r", "\\r")
	return content
}

func (s *Service) writeMessageLog(at, content string) {
	writeTimestampedMessageLog(s.messageLog, s.logger, at, content)
}

func (s *Service) logSkip(content string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return
	}
	s.writeMessageLog(time.Now().Format(time.RFC3339), content)
	if s != nil && s.logger != nil {
		s.logger.Print(content)
	}
}

func (s *Service) shouldSkip(messageID string) bool {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return false
	}
	s.seenMu.Lock()
	defer s.seenMu.Unlock()
	now := time.Now()
	for id, seenAt := range s.seenMessageIDs {
		if now.Sub(seenAt) > 10*time.Minute {
			delete(s.seenMessageIDs, id)
		}
	}
	if _, ok := s.seenMessageIDs[messageID]; ok {
		return true
	}
	s.seenMessageIDs[messageID] = now
	return false
}

type DefaultSessionFactory struct{}

func (DefaultSessionFactory) New(meta *connectsvc.Meta, cfg Config, logger *log.Logger, downloadDir string) (Session, error) {
	switch cfg.Mode {
	case "mock":
		return NewMockSession(cfg), nil
	case "", "feishu":
		session := NewFeishuSession(cfg, logger)
		session.downloadDir = downloadDir
		return session, nil
	default:
		return nil, fmt.Errorf("unsupported feishu mode: %s", cfg.Mode)
	}
}

type FeishuSession struct {
	cfg         Config
	logger      *log.Logger
	downloader  MessageDownloader
	downloadDir string
	events      chan SessionEvent
}

func NewFeishuSession(cfg Config, logger *log.Logger) *FeishuSession {
	return &FeishuSession{
		cfg:        cfg,
		logger:     logger,
		downloader: newLarkMessageDownloader(cfg),
		events:     make(chan SessionEvent, 32),
	}
}

func (s *FeishuSession) Events() <-chan SessionEvent {
	return s.events
}

func (s *FeishuSession) Run(ctx context.Context) error {
	defer close(s.events)

	if s.downloader == nil {
		s.downloader = newLarkMessageDownloader(s.cfg)
	}
	if err := installFeishuNetworkCompat(); err != nil {
		return err
	}
	wsLogger := newWSBridgeLogger(s.events, s.logger)
	handler := dispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(func(_ context.Context, event *larkim.P2MessageReceiveV1) error {
			raw, err := json.Marshal(event)
			if err != nil {
				return err
			}
			parsed := parseIncomingMessage(raw, "")
			if len(parsed.message.Raw) == 0 {
				parsed.message.Raw = string(raw)
			}
			if len(parsed.attachments) > 0 && s.downloader != nil {
				resolved, resolveErr := s.resolveAttachments(ctx, parsed.attachments)
				if resolveErr != nil {
					if s.logger != nil {
						s.logger.Printf("stage=resolve-attachments event_id=%s message_id=%s err=%v", parsed.message.EventID, parsed.message.MessageID, resolveErr)
					}
				} else {
					parsed.message.Artifacts = resolved
				}
			}
			select {
			case s.events <- SessionEvent{
				Type:    SessionEventMessage,
				At:      time.Now(),
				Message: parsed.message,
			}:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		})

	client := larkws.NewClient(
		s.cfg.AppID,
		s.cfg.AppSecret,
		larkws.WithEventHandler(handler),
		larkws.WithLogger(wsLogger),
		larkws.WithLogLevel(larkcore.LogLevelDebug),
	)

	errCh := make(chan error, 1)
	go func() {
		errCh <- client.Start(ctx)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errCh:
			if err == nil || errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
	}
}

func (s *FeishuSession) resolveAttachments(ctx context.Context, refs []attachmentRef) ([]string, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		path, err := s.downloadAttachment(ctx, ref)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(path) != "" {
			out = append(out, path)
		}
	}
	return out, nil
}

func (s *FeishuSession) downloadAttachment(ctx context.Context, ref attachmentRef) (string, error) {
	if s.downloader == nil {
		return "", errors.New("message downloader is required")
	}
	var (
		data     []byte
		fileName string
		err      error
	)
	switch ref.kind {
	case "image":
		data, fileName, err = s.downloader.DownloadImage(ctx, ref.messageID, ref.key)
	case "file":
		data, fileName, err = s.downloader.DownloadFile(ctx, ref.messageID, ref.key)
	default:
		return "", fmt.Errorf("unsupported attachment kind: %s", ref.kind)
	}
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", fmt.Errorf("empty attachment data: %s", ref.key)
	}
	if ref.kind == "image" {
		fileName = strings.TrimSpace(ref.key) + detectImageExtension(data)
	} else {
		ext := filepath.Ext(strings.TrimSpace(fileName))
		if ext == "" {
			ext = ".bin"
		}
		fileName = strings.TrimSpace(ref.key) + ext
	}
	fileName = sanitizeFileName(fileName)
	if fileName == "" {
		fileName = fmt.Sprintf("%s_%s.bin", ref.kind, ref.key)
	}
	downloadDir := strings.TrimSpace(s.downloadDir)
	if downloadDir == "" {
		downloadDir = filepath.Join(".", "feishu_artifacts")
	}
	target := filepath.Join(downloadDir, fileName)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return "", err
	}
	return target, nil
}

type MockSession struct {
	cfg    Config
	events chan SessionEvent
}

func NewMockSession(cfg Config) *MockSession {
	return &MockSession{
		cfg:    cfg,
		events: make(chan SessionEvent, 32),
	}
}

func (s *MockSession) Events() <-chan SessionEvent {
	return s.events
}

func (s *MockSession) Run(ctx context.Context) error {
	defer close(s.events)

	select {
	case s.events <- SessionEvent{Type: SessionEventConnected, At: time.Now()}:
	case <-ctx.Done():
		return nil
	}

	heartbeatEvery := time.Duration(s.cfg.MockHeartbeatMS) * time.Millisecond
	if heartbeatEvery <= 0 {
		heartbeatEvery = defaultMockHeartbeat
	}
	ticker := time.NewTicker(heartbeatEvery)
	defer ticker.Stop()

	errCh := make(chan error, 1)
	if s.cfg.MockDisconnectAfterMS > 0 {
		go func() {
			select {
			case <-ctx.Done():
				errCh <- nil
			case <-time.After(time.Duration(s.cfg.MockDisconnectAfterMS) * time.Millisecond):
				errCh <- errors.New("mock heartbeat disconnected")
			}
		}()
	}

	type timedMessage struct {
		at      time.Duration
		message MockMessage
	}
	queue := make([]timedMessage, 0, len(s.cfg.MockMessages))
	for _, item := range s.cfg.MockMessages {
		queue = append(queue, timedMessage{
			at:      time.Duration(item.DelayMS) * time.Millisecond,
			message: item,
		})
	}
	sort.Slice(queue, func(i, j int) bool { return queue[i].at < queue[j].at })
	start := time.Now()
	index := 0

	for {
		var next <-chan time.Time
		if index < len(queue) {
			waitFor := queue[index].at - time.Since(start)
			if waitFor < 0 {
				waitFor = 0
			}
			next = time.After(waitFor)
		}
		select {
		case <-ctx.Done():
			return nil
		case err := <-errCh:
			return err
		case t := <-ticker.C:
			select {
			case s.events <- SessionEvent{Type: SessionEventHeartbeat, At: t}:
			case <-ctx.Done():
				return nil
			}
		case <-next:
			item := queue[index].message
			index++
			select {
			case s.events <- SessionEvent{
				Type: SessionEventMessage,
				At:   time.Now(),
				Message: IncomingMessage{
					MessageID:   firstNonEmpty(item.MessageID, fmt.Sprintf("mock-%d", index)),
					ChatID:      firstNonEmpty(item.ChatID, "mock-chat"),
					MessageAtMS: firstInt64(item.MessageAtMS, time.Now().UnixMilli()),
					SenderID:    firstNonEmpty(item.SenderID, "mock-user"),
					SenderType:  firstNonEmpty(item.SenderType, "user"),
					MessageType: firstNonEmpty(item.MessageType, "text"),
					Content:     item.Content,
					RawContent:  item.Content,
					Raw:         item.Content,
					ExternalID:  md5Key(firstInt64(item.MessageAtMS, time.Now().UnixMilli()), item.Content),
					Parsed:      true,
				},
			}:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func parseIncomingMessage(raw []byte, requestID string) parseResult {
	message := IncomingMessage{
		RequestID: requestID,
		Raw:       string(raw),
	}
	result := parseResult{message: message}
	var envelope map[string]any
	if err := json.Unmarshal(raw, &envelope); err != nil {
		result.message.Content = string(raw)
		result.message.RawContent = string(raw)
		return result
	}
	result.message.EventID = stringPath(envelope, "header", "event_id")
	result.message.MessageID = stringPath(envelope, "event", "message", "message_id")
	result.message.ChatID = stringPath(envelope, "event", "message", "chat_id")
	result.message.MessageAtMS = int64Path(envelope, "header", "create_time")
	if result.message.MessageAtMS == 0 {
		result.message.MessageAtMS = int64Path(envelope, "event", "message", "create_time")
	}
	result.message.MessageType = stringPath(envelope, "event", "message", "message_type")
	result.message.SenderType = stringPath(envelope, "event", "sender", "sender_type")
	result.message.SenderID = firstNonEmpty(
		stringPath(envelope, "event", "sender", "sender_id", "open_id"),
		stringPath(envelope, "event", "sender", "sender_id", "user_id"),
		stringPath(envelope, "event", "sender", "sender_id", "union_id"),
	)
	result.message.RawContent = stringPath(envelope, "event", "message", "content")
	result.message.Content, result.attachments = normalizeMessageContent(result.message.MessageType, result.message.RawContent)
	for i := range result.attachments {
		if strings.TrimSpace(result.attachments[i].messageID) == "" {
			result.attachments[i].messageID = result.message.MessageID
		}
	}
	if result.message.Content == "" {
		result.message.Content = strings.TrimSpace(result.message.RawContent)
	}
	result.message.ExternalID = md5Key(result.message.MessageAtMS, result.message.Content)
	result.message.Parsed = result.message.ChatID != "" && result.message.Content != "" && result.message.ExternalID != ""
	return result
}

func normalizeMessageContent(messageType, raw string) (string, []attachmentRef) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	var textBody struct {
		Text string `json:"text"`
	}
	if json.Unmarshal([]byte(raw), &textBody) == nil && strings.TrimSpace(textBody.Text) != "" {
		return strings.TrimSpace(textBody.Text), nil
	}
	var mediaBody struct {
		ImageKey string `json:"image_key"`
		FileKey  string `json:"file_key"`
		FileName string `json:"file_name"`
	}
	if json.Unmarshal([]byte(raw), &mediaBody) == nil {
		switch strings.ToLower(strings.TrimSpace(messageType)) {
		case "image":
			if strings.TrimSpace(mediaBody.ImageKey) != "" {
				return "[image]", []attachmentRef{{kind: "image", key: strings.TrimSpace(mediaBody.ImageKey)}}
			}
		case "file":
			if strings.TrimSpace(mediaBody.FileKey) != "" {
				label := "[file]"
				if strings.TrimSpace(mediaBody.FileName) != "" {
					label = strings.TrimSpace(mediaBody.FileName)
				}
				return label, []attachmentRef{{kind: "file", key: strings.TrimSpace(mediaBody.FileKey)}}
			}
		}
	}
	return raw, nil
}

func stringPath(root map[string]any, parts ...string) string {
	var current any = root
	for _, part := range parts {
		asMap, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current, ok = asMap[part]
		if !ok {
			return ""
		}
	}
	switch value := current.(type) {
	case string:
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func int64Path(root map[string]any, parts ...string) int64 {
	var current any = root
	for _, part := range parts {
		asMap, ok := current.(map[string]any)
		if !ok {
			return 0
		}
		current, ok = asMap[part]
		if !ok {
			return 0
		}
	}
	switch value := current.(type) {
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		return n
	case float64:
		return int64(value)
	case int64:
		return value
	case int:
		return int64(value)
	default:
		return 0
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type MessageAPI interface {
	Create(ctx context.Context, req *larkim.CreateMessageReq, options ...larkcore.RequestOptionFunc) (*larkim.CreateMessageResp, error)
	Reply(ctx context.Context, req *larkim.ReplyMessageReq, options ...larkcore.RequestOptionFunc) (*larkim.ReplyMessageResp, error)
}

type ImageAPI interface {
	Create(ctx context.Context, req *larkim.CreateImageReq, options ...larkcore.RequestOptionFunc) (*larkim.CreateImageResp, error)
}

type FileAPI interface {
	Create(ctx context.Context, req *larkim.CreateFileReq, options ...larkcore.RequestOptionFunc) (*larkim.CreateFileResp, error)
}

type LarkAPISet struct {
	Message MessageAPI
	Image   ImageAPI
	File    FileAPI
}

type Sender struct {
	loader      MetaLoader
	apis        LarkAPISet
	logger      *log.Logger
	messageLog  io.Writer
	downloadDir string
	httpClient  *http.Client
	now         func() time.Time
}

type sendRetryExhaustedError struct {
	step string
	err  error
}

func (e *sendRetryExhaustedError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *sendRetryExhaustedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func NewSender(loader MetaLoader, logger *log.Logger, messageLog io.Writer) *Sender {
	sender := &Sender{
		loader:     loader,
		logger:     logger,
		messageLog: messageLog,
		httpClient: &http.Client{Timeout: defaultAttachmentTimeout},
		now:        time.Now,
	}
	return sender
}

var defaultLarkAPIs = func(cfg Config) (LarkAPISet, error) {
	client, err := newFeishuLarkClient(cfg)
	if err != nil {
		return LarkAPISet{}, err
	}
	return LarkAPISet{
		Message: client.Im.Message,
		Image:   client.Im.Image,
		File:    client.Im.File,
	}, nil
}

func (s *Sender) ensureAPIs(cfg Config) (LarkAPISet, error) {
	if s.apis.Message != nil && s.apis.Image != nil && s.apis.File != nil {
		return s.apis, nil
	}
	return defaultLarkAPIs(cfg)
}

func (s *Sender) Send(ctx context.Context, input SendInput) (*SendResult, error) {
	if s == nil || s.loader == nil {
		return nil, errors.New("sender loader is required")
	}
	action := strings.TrimSpace(strings.ToLower(input.Action))
	if action == "" {
		action = "send"
	}
	at := s.clock().Format(time.RFC3339)
	s.writeMessageLog(at, formatSendRequestLog(action, input))

	meta, cfg, err := s.loader.Load(ctx)
	if err != nil {
		s.writeMessageLog(at, formatSendFailureLog(action, "load-meta", nil, err))
		return nil, err
	}
	replyCtx, err := extractReplyMessageContext(input.Message)
	if err != nil {
		s.writeMessageLog(at, formatSendFailureLog(action, "parse-message", nil, err))
		return nil, err
	}
	s.writeMessageLog(at, formatSendParseLog(action, replyCtx))
	messageID := replyCtx.MessageID
	if messageID == "" {
		err := errors.New("message is required")
		s.writeMessageLog(at, formatSendFailureLog(action, "parse-message", nil, err))
		return nil, err
	}

	content := strings.TrimSpace(input.Content)
	images := cleanCSVItems(input.Images)
	files := cleanCSVItems(input.Files)
	if content == "" && len(images) == 0 && len(files) == 0 {
		err := errors.New("content, image or file is required")
		s.writeMessageLog(at, formatSendFailureLog(action, "validate-input", nil, err))
		return nil, err
	}

	apis, err := s.ensureAPIs(cfg)
	if err != nil {
		s.writeMessageLog(at, formatSendFailureLog(action, "build-api", nil, err))
		return nil, err
	}
	if apis.Message == nil || apis.Image == nil || apis.File == nil {
		err := errors.New("feishu api client is required")
		s.writeMessageLog(at, formatSendFailureLog(action, "build-api", nil, err))
		return nil, err
	}

	result := &SendResult{
		Name:      firstNonEmpty(meta.Name, DefaultName),
		Target:    strings.TrimSpace(meta.ChatID),
		MessageID: messageID,
		Sent:      make([]SentRecord, 0, 1+len(images)+len(files)),
	}
	fallbackTriggered := false

	if normalized, ok := s.normalizeSendInput(content); ok {
		content = normalized.Content
		images = uniqueStrings(append(images, normalized.Images...))
		files = uniqueStrings(append(files, normalized.Files...))
		s.writeSchemaLogs(normalized)
	}
	if len(images) > 0 && content != "" {
		images = filterImagesAlreadyEmbeddedInMarkdown(content, images)
	}

	if content != "" {
		rewrittenContent, rewritten, err := s.prepareMarkdownContent(ctx, apis.Image, content)
		if err != nil {
			content = strings.TrimSpace(input.Content)
			images = nil
			files = nil
			fallbackTriggered = true
		} else if rewritten {
			content = rewrittenContent
		}
	}

	for _, imagePath := range images {
		record, err := s.sendImage(ctx, apis, "", messageID, imagePath)
		if err != nil {
			s.writeMessageLog(at, formatSendFailureLog(action, "send-image", result, err))
			return nil, err
		}
		result.Sent = append(result.Sent, record)
	}

	for _, filePath := range files {
		record, err := s.sendFile(ctx, apis, "", messageID, filePath)
		if err != nil {
			s.writeMessageLog(at, formatSendFailureLog(action, "send-file", result, err))
			return nil, err
		}
		result.Sent = append(result.Sent, record)
	}

	if content != "" {
		record, err := s.sendText(ctx, apis, "", messageID, content)
		if err != nil {
			if fallbackTriggered && !isSendRetryExhaustedError(err) {
				fallbackRecord, fallbackErr := s.sendText(ctx, apis, "", messageID, "<消息异常>请登录客户端查看")
				if fallbackErr == nil {
					result.Sent = append(result.Sent, fallbackRecord)
					s.writeMessageLog(at, formatSendResultLog(action, result))
					return result, nil
				}
				s.writeMessageLog(at, formatSendFailureLog(action, "send-text-fallback", result, fmt.Errorf("send raw content failed: %v; send fallback content failed: %w", err, fallbackErr)))
				return nil, fallbackErr
			}
			s.writeMessageLog(at, formatSendFailureLog(action, "send-text", result, err))
			return nil, err
		}
		result.Sent = append(result.Sent, record)
	}

	s.writeMessageLog(at, formatSendResultLog(action, result))
	return result, nil
}

func (s *Sender) clock() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}

func (s *Sender) logf(format string, args ...any) {
	if s != nil && s.logger != nil {
		s.logger.Printf(format, args...)
	}
}

func (s *Sender) writeMessageLog(at, content string) {
	if s == nil {
		return
	}
	writeTimestampedMessageLog(s.messageLog, s.logger, at, content)
}

func attachmentTimeoutContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, defaultAttachmentTimeout)
}

func appendAttachmentFailureNotice(content string) string {
	const notice = "<附件发送失败>"
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return notice
	}
	if strings.Contains(trimmed, notice) {
		return content
	}
	return strings.TrimRight(content, "\n") + "\n" + notice
}

func (s *Sender) sendText(ctx context.Context, apis LarkAPISet, target, messageID, content string) (SentRecord, error) {
	var record SentRecord
	err := s.retrySendOperation(ctx, "send-text", func() error {
		payload, err := buildFeishuMarkdownCardContent(content)
		if err != nil {
			return err
		}
		if strings.TrimSpace(messageID) != "" {
			req := larkim.NewReplyMessageReqBuilder().
				MessageId(strings.TrimSpace(messageID)).
				Body(larkim.NewReplyMessageReqBodyBuilder().
					MsgType("interactive").
					Content(payload).
					ReplyInThread(false).
					Uuid(newSendUUID()).
					Build()).
				Build()
			resp, err := apis.Message.Reply(ctx, req)
			if err != nil {
				return err
			}
			if resp == nil || !resp.Success() {
				return apiError(resp)
			}
			record = sentRecordFromReply("text", strings.TrimSpace(messageID), resp)
			return nil
		}
		req := larkim.NewCreateMessageReqBuilder().
			ReceiveIdType(larkim.ReceiveIdTypeChatId).
			Body(larkim.NewCreateMessageReqBodyBuilder().
				ReceiveId(strings.TrimSpace(target)).
				MsgType("interactive").
				Content(payload).
				Uuid(newSendUUID()).
				Build()).
			Build()
		resp, err := apis.Message.Create(ctx, req)
		if err != nil {
			return err
		}
		if resp == nil || !resp.Success() {
			return apiError(resp)
		}
		record = sentRecordFromCreate("text", resp)
		return nil
	})
	if err != nil {
		return SentRecord{}, err
	}
	return record, nil
}

var markdownImagePattern = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)

func filterImagesAlreadyEmbeddedInMarkdown(content string, images []string) []string {
	if len(images) == 0 {
		return nil
	}
	embedded := extractLocalMarkdownImagePaths(content)
	if len(embedded) == 0 {
		return images
	}
	filtered := make([]string, 0, len(images))
	for _, imagePath := range images {
		normalized := strings.TrimSpace(imagePath)
		if normalized == "" {
			continue
		}
		if cleaned, err := filepath.Abs(filepath.Clean(normalized)); err == nil {
			normalized = cleaned
		} else {
			normalized = filepath.Clean(normalized)
		}
		if _, exists := embedded[normalized]; exists {
			continue
		}
		filtered = append(filtered, imagePath)
	}
	return filtered
}

func extractLocalMarkdownImagePaths(content string) map[string]struct{} {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	matches := markdownImagePattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return nil
	}
	paths := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		urlStart, urlEnd := match[4], match[5]
		imageURL := strings.TrimSpace(content[urlStart:urlEnd])
		if !isLocalMarkdownImagePath(imageURL) {
			continue
		}
		normalized, err := normalizeLocalMarkdownImagePath(imageURL)
		if err != nil {
			continue
		}
		if absPath, err := filepath.Abs(normalized); err == nil {
			normalized = absPath
		}
		paths[filepath.Clean(normalized)] = struct{}{}
	}
	if len(paths) == 0 {
		return nil
	}
	return paths
}

func (s *Sender) prepareMarkdownContent(ctx context.Context, imageAPI ImageAPI, content string) (string, bool, error) {
	content = strings.TrimSpace(content)
	matches := markdownImagePattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content, false, nil
	}
	cache := make(map[string]string, len(matches))
	var builder strings.Builder
	last := 0
	rewritten := false
	for _, match := range matches {
		start, end := match[0], match[1]
		urlStart, urlEnd := match[4], match[5]
		builder.WriteString(content[last:start])
		raw := content[start:end]
		imageURL := strings.TrimSpace(content[urlStart:urlEnd])
		localPath, kind, err := s.resolveMarkdownImageSource(ctx, imageURL)
		if err != nil {
			return "", false, err
		}
		if kind == markdownImageUnsupported {
			builder.WriteString(raw)
			last = end
			continue
		}
		imageKey, ok := cache[imageURL]
		if !ok {
			imageKey, err = s.uploadImage(ctx, imageAPI, localPath)
			if err != nil {
				return "", false, err
			}
			cache[imageURL] = imageKey
		}
		builder.WriteString(content[start:urlStart])
		builder.WriteString(imageKey)
		builder.WriteString(content[urlEnd:end])
		last = end
		rewritten = true
	}
	builder.WriteString(content[last:])
	return builder.String(), rewritten, nil
}

func buildFeishuMarkdownCardContent(content string) (string, error) {
	content = strings.TrimSpace(content)
	payload, err := json.Marshal(map[string]any{
		"schema": "2.0",
		"body": map[string]any{
			"elements": []map[string]string{
				{
					"tag":     "markdown",
					"content": content,
				},
			},
		},
	})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func (s *Sender) sendImage(ctx context.Context, apis LarkAPISet, target, messageID, imagePath string) (SentRecord, error) {
	var record SentRecord
	err := s.retrySendOperation(ctx, "send-image", func() error {
		attachmentCtx, cancel := attachmentTimeoutContext(ctx)
		defer cancel()

		imageKey, err := s.uploadImage(attachmentCtx, apis.Image, imagePath)
		if err != nil {
			return err
		}
		payload, err := json.Marshal(map[string]string{"image_key": imageKey})
		if err != nil {
			return err
		}
		record, err = s.sendByMessageIDOrTarget(attachmentCtx, apis.Message, target, messageID, "image", string(payload))
		if err != nil {
			return err
		}
		record.ResourceKey = imageKey
		record.Path = imagePath
		return nil
	})
	if err != nil {
		return SentRecord{}, err
	}
	return record, nil
}

func (s *Sender) uploadImage(ctx context.Context, api ImageAPI, imagePath string) (string, error) {
	body, err := larkim.NewCreateImagePathReqBodyBuilder().
		ImageType(larkim.ImageTypeMessage).
		ImagePath(imagePath).
		Build()
	if err != nil {
		return "", err
	}
	resp, err := api.Create(ctx, larkim.NewCreateImageReqBuilder().Body(body).Build())
	if err != nil {
		return "", err
	}
	if resp == nil || !resp.Success() || resp.Data == nil || resp.Data.ImageKey == nil || strings.TrimSpace(*resp.Data.ImageKey) == "" {
		return "", apiError(resp)
	}
	return strings.TrimSpace(*resp.Data.ImageKey), nil
}

func isRemoteHTTPURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return strings.TrimSpace(parsed.Host) != ""
	default:
		return false
	}
}

type markdownImageSourceKind string

const (
	markdownImageUnsupported markdownImageSourceKind = "unsupported"
	markdownImageRemote      markdownImageSourceKind = "remote"
	markdownImageLocal       markdownImageSourceKind = "local"
)

func (s *Sender) resolveMarkdownImageSource(ctx context.Context, imageURL string) (string, markdownImageSourceKind, error) {
	imageURL = strings.TrimSpace(imageURL)
	switch {
	case isRemoteHTTPURL(imageURL):
		path, err := s.downloadMarkdownImage(ctx, imageURL)
		return path, markdownImageRemote, err
	case isLocalMarkdownImagePath(imageURL):
		path, err := normalizeLocalMarkdownImagePath(imageURL)
		return path, markdownImageLocal, err
	default:
		return "", markdownImageUnsupported, nil
	}
}

func isLocalMarkdownImagePath(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "file://") {
		return true
	}
	return filepath.IsAbs(value)
}

func normalizeLocalMarkdownImagePath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "file://") {
		parsed, err := url.Parse(value)
		if err != nil {
			return "", err
		}
		if parsed.Scheme != "file" {
			return "", fmt.Errorf("unsupported local markdown image url: %s", value)
		}
		if strings.TrimSpace(parsed.Path) == "" {
			return "", fmt.Errorf("empty local markdown image path: %s", value)
		}
		return filepath.Clean(parsed.Path), nil
	}
	if !filepath.IsAbs(value) {
		return "", fmt.Errorf("markdown image path must be absolute: %s", value)
	}
	return filepath.Clean(value), nil
}

func (s *Sender) downloadMarkdownImage(ctx context.Context, imageURL string) (string, error) {
	attachmentCtx, cancel := attachmentTimeoutContext(ctx)
	defer cancel()

	client := s.httpClient
	if client == nil {
		client = &http.Client{Timeout: defaultAttachmentTimeout}
	}
	req, err := http.NewRequestWithContext(attachmentCtx, http.MethodGet, imageURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download markdown image failed status=%d url=%s", resp.StatusCode, imageURL)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", fmt.Errorf("download markdown image empty body: %s", imageURL)
	}
	targetDir := strings.TrimSpace(s.downloadDir)
	if targetDir == "" {
		targetDir = filepath.Join(".", "feishu_artifacts")
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("create feishu download dir: %w", err)
	}
	sum := md5.Sum([]byte(imageURL))
	fileName := fmt.Sprintf("remote_%x%s", sum, inferImageExtension(imageURL, resp.Header.Get("Content-Type")))
	target := filepath.Join(targetDir, sanitizeFileName(fileName))
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return "", err
	}
	return target, nil
}

func inferImageExtension(imageURL, contentType string) string {
	if parsed, err := url.Parse(strings.TrimSpace(imageURL)); err == nil {
		if ext := strings.ToLower(filepath.Ext(parsed.Path)); ext != "" {
			return ext
		}
	}
	if contentType != "" {
		if exts, err := mime.ExtensionsByType(strings.TrimSpace(strings.Split(contentType, ";")[0])); err == nil {
			for _, ext := range exts {
				if strings.TrimSpace(ext) != "" {
					return ext
				}
			}
		}
	}
	return ".png"
}

func (s *Sender) normalizeSendInput(content string) (normalizedSendPayload, bool) {
	raw := strings.TrimSpace(content)
	if raw == "" || !strings.HasPrefix(raw, "{") || !strings.HasSuffix(raw, "}") {
		return normalizedSendPayload{}, false
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return normalizedSendPayload{}, false
	}
	if !matchesSchema(value, ResponseSchemaDefinition()) {
		return normalizedSendPayload{}, false
	}
	object, ok := value.(map[string]any)
	if !ok {
		return normalizedSendPayload{}, false
	}
	body, ok := object["content"].(string)
	if !ok {
		return normalizedSendPayload{}, false
	}
	payload := normalizedSendPayload{Content: strings.TrimSpace(body)}
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

func (s *Sender) writeSchemaLogs(payload normalizedSendPayload) {
	if s == nil {
		return
	}
	s.logf("schema-send content=%s", strconv.Quote(payload.Content))
	for _, image := range payload.Images {
		s.logf("schema-send image=%s", strconv.Quote(image))
	}
	for _, file := range payload.Files {
		s.logf("schema-send file=%s", strconv.Quote(file))
	}
}

func (s *Sender) sendFile(ctx context.Context, apis LarkAPISet, target, messageID, filePath string) (SentRecord, error) {
	var record SentRecord
	err := s.retrySendOperation(ctx, "send-file", func() error {
		attachmentCtx, cancel := attachmentTimeoutContext(ctx)
		defer cancel()

		body, err := larkim.NewCreateFilePathReqBodyBuilder().
			FileType(detectFeishuFileType(filePath)).
			FileName(filepath.Base(filePath)).
			FilePath(filePath).
			Build()
		if err != nil {
			return err
		}
		resp, err := apis.File.Create(attachmentCtx, larkim.NewCreateFileReqBuilder().Body(body).Build())
		if err != nil {
			return err
		}
		if resp == nil || !resp.Success() || resp.Data == nil || resp.Data.FileKey == nil || strings.TrimSpace(*resp.Data.FileKey) == "" {
			return apiError(resp)
		}
		fileKey := strings.TrimSpace(*resp.Data.FileKey)
		payload, err := json.Marshal(map[string]string{"file_key": fileKey})
		if err != nil {
			return err
		}
		record, err = s.sendByMessageIDOrTarget(attachmentCtx, apis.Message, target, messageID, "file", string(payload))
		if err != nil {
			return err
		}
		record.ResourceKey = fileKey
		record.Path = filePath
		return nil
	})
	if err != nil {
		return SentRecord{}, err
	}
	return record, nil
}

func (s *Sender) retrySendOperation(ctx context.Context, step string, fn func() error) error {
	var lastErr error
	for attempt := 1; attempt <= maxSendRetries; attempt++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			s.writeRetryLog(step, attempt, maxSendRetries, err)
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
				break
			}
		}
	}
	s.writeTerminateLog(step, maxSendRetries, lastErr)
	return &sendRetryExhaustedError{step: step, err: lastErr}
}

func (s *Sender) writeRetryLog(step string, attempt, max int, err error) {
	if s == nil || err == nil {
		return
	}
	line := fmt.Sprintf(
		"stage=send-retry step=%s attempt=%d max=%d err=%s",
		quoteLogValue(strings.TrimSpace(step)),
		attempt,
		max,
		quoteLogValue(err.Error()),
	)
	s.writeMessageLog(s.clock().Format(time.RFC3339), line)
	s.logf("%s", line)
}

func (s *Sender) writeTerminateLog(step string, max int, err error) {
	if s == nil || err == nil {
		return
	}
	line := fmt.Sprintf(
		"stage=send-terminate step=%s max=%d err=%s",
		quoteLogValue(strings.TrimSpace(step)),
		max,
		quoteLogValue(err.Error()),
	)
	s.writeMessageLog(s.clock().Format(time.RFC3339), line)
	s.logf("%s", line)
}

func isSendRetryExhaustedError(err error) bool {
	var target *sendRetryExhaustedError
	return errors.As(err, &target)
}

func (s *Sender) sendByMessageIDOrTarget(ctx context.Context, api MessageAPI, target, messageID, msgType, content string) (SentRecord, error) {
	if strings.TrimSpace(messageID) != "" {
		req := larkim.NewReplyMessageReqBuilder().
			MessageId(strings.TrimSpace(messageID)).
			Body(larkim.NewReplyMessageReqBodyBuilder().
				MsgType(msgType).
				Content(content).
				ReplyInThread(false).
				Uuid(newSendUUID()).
				Build()).
			Build()
		resp, err := api.Reply(ctx, req)
		if err != nil {
			return SentRecord{}, err
		}
		if resp == nil || !resp.Success() {
			return SentRecord{}, apiError(resp)
		}
		return sentRecordFromReply(msgType, strings.TrimSpace(messageID), resp), nil
	}
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(strings.TrimSpace(target)).
			MsgType(msgType).
			Content(content).
			Uuid(newSendUUID()).
			Build()).
		Build()
	resp, err := api.Create(ctx, req)
	if err != nil {
		return SentRecord{}, err
	}
	if resp == nil || !resp.Success() {
		return SentRecord{}, apiError(resp)
	}
	return sentRecordFromCreate(msgType, resp), nil
}

func sentRecordFromCreate(kind string, resp *larkim.CreateMessageResp) SentRecord {
	record := SentRecord{Type: kind}
	if resp == nil || resp.Data == nil {
		return record
	}
	if resp.Data.MessageId != nil {
		record.MessageID = strings.TrimSpace(*resp.Data.MessageId)
	}
	if resp.Data.ParentId != nil {
		record.ParentID = strings.TrimSpace(*resp.Data.ParentId)
	}
	if resp.Data.ThreadId != nil {
		record.ThreadID = strings.TrimSpace(*resp.Data.ThreadId)
	}
	return record
}

func sentRecordFromReply(kind, parentID string, resp *larkim.ReplyMessageResp) SentRecord {
	record := SentRecord{Type: kind, ParentID: strings.TrimSpace(parentID)}
	if resp == nil || resp.Data == nil {
		return record
	}
	if resp.Data.MessageId != nil {
		record.MessageID = strings.TrimSpace(*resp.Data.MessageId)
	}
	if resp.Data.ParentId != nil && strings.TrimSpace(*resp.Data.ParentId) != "" {
		record.ParentID = strings.TrimSpace(*resp.Data.ParentId)
	}
	if resp.Data.ThreadId != nil {
		record.ThreadID = strings.TrimSpace(*resp.Data.ThreadId)
	}
	return record
}

func apiError(resp interface{}) error {
	switch value := resp.(type) {
	case *larkim.CreateMessageResp:
		if value != nil {
			return fmt.Errorf("feishu api error code=%d msg=%s", value.Code, value.Msg)
		}
	case *larkim.ReplyMessageResp:
		if value != nil {
			return fmt.Errorf("feishu api error code=%d msg=%s", value.Code, value.Msg)
		}
	case *larkim.CreateImageResp:
		if value != nil {
			return fmt.Errorf("feishu api error code=%d msg=%s", value.Code, value.Msg)
		}
	case *larkim.CreateFileResp:
		if value != nil {
			return fmt.Errorf("feishu api error code=%d msg=%s", value.Code, value.Msg)
		}
	}
	return errors.New("feishu api error")
}

func cleanCSVItems(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func splitCSVFlag(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return cleanCSVItems(strings.Split(value, ","))
}

type replyMessageContext struct {
	MessageID   string
	EnvelopeRaw string
}

func extractReplyMessageContext(raw string) (*replyMessageContext, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("message is required")
	}

	var requestPayload map[string]any
	if err := json.Unmarshal([]byte(raw), &requestPayload); err != nil {
		return nil, errors.New("message must be valid json")
	}

	envelopeRaw := strings.TrimSpace(firstNonEmpty(
		stringPath(requestPayload, "rawRequest"),
		stringPath(requestPayload, "raw_request"),
	))
	if envelopeRaw == "" {
		return nil, errors.New("message.messageId not found")
	}

	var envelope map[string]any
	if err := json.Unmarshal([]byte(envelopeRaw), &envelope); err != nil {
		return nil, errors.New("message.messageId not found")
	}

	messageID := strings.TrimSpace(stringPath(envelope, "message", "messageId"))
	if messageID == "" {
		return nil, errors.New("message.messageId not found")
	}
	return &replyMessageContext{
		MessageID:   messageID,
		EnvelopeRaw: envelopeRaw,
	}, nil
}

func extractReplyMessageID(raw string) (string, error) {
	ctx, err := extractReplyMessageContext(raw)
	if err != nil {
		return "", err
	}
	return ctx.MessageID, nil
}

func newSendUUID() string {
	return fmt.Sprintf("feishu-send-%d", time.Now().UnixNano())
}

func formatSendRequestLog(action string, input SendInput) string {
	action = strings.TrimSpace(strings.ToLower(action))
	if action == "" {
		action = "send"
	}
	parts := []string{
		"stage=send-request",
		"action=" + quoteLogValue(action),
		"message=" + quoteLogValue(input.Message),
		"content=" + quoteLogValue(input.Content),
	}
	if images := strings.Join(cleanCSVItems(input.Images), ","); images != "" {
		parts = append(parts, "images="+quoteLogValue(images))
	}
	if files := strings.Join(cleanCSVItems(input.Files), ","); files != "" {
		parts = append(parts, "files="+quoteLogValue(files))
	}
	return strings.Join(parts, " ")
}

func formatSendParseLog(action string, ctx *replyMessageContext) string {
	action = strings.TrimSpace(strings.ToLower(action))
	if action == "" {
		action = "send"
	}
	if ctx == nil {
		return strings.Join([]string{
			"stage=send-parse",
			"action=" + quoteLogValue(action),
		}, " ")
	}
	return strings.Join([]string{
		"stage=send-parse",
		"action=" + quoteLogValue(action),
		"reply_to=" + quoteLogValue(ctx.MessageID),
		"raw_request=" + quoteLogValue(ctx.EnvelopeRaw),
	}, " ")
}

func formatSendResultLog(action string, result *SendResult) string {
	action = strings.TrimSpace(strings.ToLower(action))
	if action == "" {
		action = "send"
	}
	if result == nil {
		return strings.Join([]string{
			"stage=send-result",
			"action=" + quoteLogValue(action),
		}, " ")
	}
	parts := []string{
		"stage=send-result",
		"action=" + quoteLogValue(action),
		"name=" + quoteLogValue(strings.TrimSpace(result.Name)),
		"target=" + quoteLogValue(strings.TrimSpace(result.Target)),
	}
	if strings.TrimSpace(result.MessageID) != "" {
		parts = append(parts, "reply_to="+quoteLogValue(strings.TrimSpace(result.MessageID)))
	}
	types := make([]string, 0, len(result.Sent))
	for _, item := range result.Sent {
		types = append(types, item.Type)
	}
	if len(types) > 0 {
		parts = append(parts, "types="+quoteLogValue(strings.Join(types, "+")))
	}
	parts = append(parts, "result="+quoteLogValue(mustJSONLogString(result)))
	return strings.Join(parts, " ")
}

func formatSendFailureLog(action, step string, result *SendResult, err error) string {
	action = strings.TrimSpace(strings.ToLower(action))
	if action == "" {
		action = "send"
	}
	parts := []string{
		"stage=send-failed",
		"action=" + quoteLogValue(action),
		"step=" + quoteLogValue(step),
	}
	if result != nil {
		if strings.TrimSpace(result.Name) != "" {
			parts = append(parts, "name="+quoteLogValue(strings.TrimSpace(result.Name)))
		}
		if strings.TrimSpace(result.Target) != "" {
			parts = append(parts, "target="+quoteLogValue(strings.TrimSpace(result.Target)))
		}
		if strings.TrimSpace(result.MessageID) != "" {
			parts = append(parts, "reply_to="+quoteLogValue(strings.TrimSpace(result.MessageID)))
		}
		if len(result.Sent) > 0 {
			types := make([]string, 0, len(result.Sent))
			for _, item := range result.Sent {
				types = append(types, item.Type)
			}
			parts = append(parts, "types="+quoteLogValue(strings.Join(types, "+")))
		}
		parts = append(parts, "result="+quoteLogValue(mustJSONLogString(result)))
	}
	if err != nil {
		parts = append(parts, "err="+quoteLogValue(err.Error()))
	}
	return strings.Join(parts, " ")
}

func quoteLogValue(value string) string {
	return strconv.Quote(strings.TrimSpace(value))
}

func mustJSONLogString(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf(`{"marshalError":%q}`, err.Error())
	}
	return string(data)
}

func detectFeishuFileType(path string) string {
	// The plugin always sends attachments as msg_type=file, so the uploaded file
	// must use Feishu's generic stream type. Specialized upload types such as
	// pdf/doc/xls/ppt/mp4/opus are intended for their corresponding message types.
	_ = path
	return larkim.FileTypeStream
}

func firstInt64(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func md5Key(messageAtMS int64, content string) string {
	content = strings.TrimSpace(content)
	if messageAtMS <= 0 || content == "" {
		return ""
	}
	sum := md5.Sum([]byte(fmt.Sprintf("%d:%s", messageAtMS, content)))
	return fmt.Sprintf("%x", sum)
}

func sanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	return name
}

type wsBridgeLogger struct {
	events chan<- SessionEvent
	logger *log.Logger
}

func newWSBridgeLogger(events chan<- SessionEvent, logger *log.Logger) larkcore.Logger {
	return &wsBridgeLogger{events: events, logger: logger}
}

func (l *wsBridgeLogger) Debug(_ context.Context, args ...interface{}) {
	l.capture("debug", args...)
}

func (l *wsBridgeLogger) Info(_ context.Context, args ...interface{}) {
	l.capture("info", args...)
}

func (l *wsBridgeLogger) Warn(_ context.Context, args ...interface{}) {
	l.capture("warn", args...)
}

func (l *wsBridgeLogger) Error(_ context.Context, args ...interface{}) {
	l.capture("error", args...)
}

func (l *wsBridgeLogger) capture(level string, args ...interface{}) {
	if len(args) == 0 || l == nil {
		return
	}
	text := strings.TrimSpace(fmt.Sprint(args...))
	if l.logger != nil && text != "" && (strings.EqualFold(strings.TrimSpace(level), "warn") || strings.EqualFold(strings.TrimSpace(level), "error")) {
		l.logger.Printf("stage=feishu-ws level=%s msg=%s", strings.TrimSpace(level), text)
	}
	if l.events == nil {
		return
	}
	if strings.Contains(text, "connected to ") {
		select {
		case l.events <- SessionEvent{Type: SessionEventConnected, At: time.Now()}:
		default:
		}
		return
	}
	if strings.Contains(strings.ToLower(text), "receive pong") {
		select {
		case l.events <- SessionEvent{Type: SessionEventHeartbeat, At: time.Now()}:
		default:
		}
	}
}

type larkMessageDownloader struct {
	client *lark.Client
	err    error
}

func newLarkMessageDownloader(cfg Config) MessageDownloader {
	if strings.TrimSpace(cfg.AppID) == "" || strings.TrimSpace(cfg.AppSecret) == "" {
		return nil
	}
	client, err := newFeishuLarkClient(cfg)
	if err != nil {
		return &larkMessageDownloader{err: err}
	}
	return &larkMessageDownloader{
		client: client,
	}
}

func (d *larkMessageDownloader) DownloadImage(ctx context.Context, messageID, imageKey string) ([]byte, string, error) {
	if d != nil && d.err != nil {
		return nil, "", d.err
	}
	if d == nil || d.client == nil {
		return nil, "", errors.New("lark client is required")
	}
	req := larkim.NewGetMessageResourceReqBuilder().
		MessageId(strings.TrimSpace(messageID)).
		FileKey(strings.TrimSpace(imageKey)).
		Type("image").
		Build()
	resp, err := d.client.Im.MessageResource.Get(ctx, req, larkcore.WithFileDownload())
	if err != nil {
		return nil, "", err
	}
	if resp == nil || resp.File == nil {
		return nil, "", fmt.Errorf("empty image response: message_id=%s image_key=%s", strings.TrimSpace(messageID), strings.TrimSpace(imageKey))
	}
	data, err := io.ReadAll(resp.File)
	if err != nil {
		return nil, "", err
	}
	fileName := sanitizeFileName(resp.FileName)
	if filepath.Ext(fileName) == "" {
		fileName += detectImageExtension(data)
	}
	if fileName == "" {
		fileName = fmt.Sprintf("image_%s%s", strings.TrimSpace(imageKey), detectImageExtension(data))
	}
	return data, fileName, nil
}

func (d *larkMessageDownloader) DownloadFile(ctx context.Context, messageID, fileKey string) ([]byte, string, error) {
	if d != nil && d.err != nil {
		return nil, "", d.err
	}
	if d == nil || d.client == nil {
		return nil, "", errors.New("lark client is required")
	}
	req := larkim.NewGetMessageResourceReqBuilder().
		MessageId(strings.TrimSpace(messageID)).
		FileKey(strings.TrimSpace(fileKey)).
		Type("file").
		Build()
	resp, err := d.client.Im.MessageResource.Get(ctx, req, larkcore.WithFileDownload())
	if err != nil {
		return nil, "", err
	}
	if resp == nil || resp.File == nil {
		return nil, "", fmt.Errorf("empty file response: message_id=%s file_key=%s", strings.TrimSpace(messageID), strings.TrimSpace(fileKey))
	}
	data, err := io.ReadAll(resp.File)
	if err != nil {
		return nil, "", err
	}
	fileName := sanitizeFileName(resp.FileName)
	if fileName == "" {
		fileName = fmt.Sprintf("file_%s.bin", strings.TrimSpace(fileKey))
	}
	return data, fileName, nil
}

func detectImageExtension(data []byte) string {
	contentType := http.DetectContentType(data)
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}

func sleepContext(ctx context.Context, wait time.Duration) bool {
	if wait <= 0 {
		wait = time.Millisecond
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
