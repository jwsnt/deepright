package main

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"connect/connectsvc"
	"connect/feishusvc"
)

const (
	localFeishuScanInterval      = 30 * time.Second
	localFeishuExpireWindow      = 10 * time.Minute
	localFeishuMaxServiceRetries = 3
)

type localFeishuService struct {
	name        string
	client      feishusvc.ConnectClient
	factory     feishusvc.SessionFactory
	logger      *log.Logger
	messageLog  io.Writer
	downloadDir string
	stateFile   string

	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration
	reconnectDelay    time.Duration
	scanInterval      time.Duration
	expireWindow      time.Duration
	nowFn             func() time.Time

	stateMu sync.Mutex
	state   localFeishuState
}

type localFeishuState struct {
	Pending map[string]localFeishuPendingMessage `json:"pending"`
	Pushed  map[string]int64                     `json:"pushed"`
}

type localFeishuPendingMessage struct {
	ExternalID string   `json:"externalId"`
	EventID    string   `json:"eventId,omitempty"`
	MessageID  string   `json:"messageId,omitempty"`
	ChatID     string   `json:"chatId,omitempty"`
	SenderID   string   `json:"senderId,omitempty"`
	SenderType string   `json:"senderType,omitempty"`
	Raw        string   `json:"raw,omitempty"`
	RawContent string   `json:"rawContent,omitempty"`
	Type       string   `json:"type"`
	Content    string   `json:"content,omitempty"`
	Artifacts  []string `json:"artifacts,omitempty"`
	MessageAt  int64    `json:"messageAt"`
	ReceivedAt int64    `json:"receivedAt"`
}

type localFeishuPushEnvelope struct {
	Source     string                      `json:"source"`
	ReceivedAt string                      `json:"receivedAt"`
	Message    feishusvc.IncomingMessage   `json:"message"`
	Pending    []localFeishuPendingMessage `json:"pending,omitempty"`
	GroupedBy  string                      `json:"groupedBy"`
	WindowSecs int                         `json:"windowSecs"`
	ExpireSecs int                         `json:"expireSecs"`
}

func newLocalFeishuService(flags map[string]string, logger *log.Logger, messageLog io.Writer) (*localFeishuService, error) {
	baseArgs, err := buildConnectBaseArgsLocalFeishu(flags)
	if err != nil {
		return nil, err
	}

	heartbeatIntervalSec, err := connectsvc.IntValue(flags, "heartbeat-seconds", int((60*time.Second)/time.Second))
	if err != nil {
		return nil, err
	}
	heartbeatTimeoutSec, err := connectsvc.IntValue(flags, "heartbeat-timeout-seconds", int((130*time.Second)/time.Second))
	if err != nil {
		return nil, err
	}
	reconnectMS, err := connectsvc.IntValue(flags, "reconnect-delay-ms", int((2*time.Second)/time.Millisecond))
	if err != nil {
		return nil, err
	}

	downloadDir := downloadDirForLogLocalFeishu(connectsvc.FirstValue(flags, "log-file"))
	stateFile := stateFileForLogLocalFeishu(connectsvc.FirstValue(flags, "log-file"))

	svc := &localFeishuService{
		name: connectsvc.FirstValue(flags, "name"),
		client: &feishusvc.CLIConnectClient{
			Binary:   resolveConnectBinaryLocalFeishu(flags),
			Prefix:   resolveConnectPrefixLocalFeishu(flags),
			BaseArgs: baseArgs,
		},
		factory:           feishusvc.DefaultSessionFactory{},
		logger:            logger,
		messageLog:        messageLog,
		downloadDir:       downloadDir,
		stateFile:         stateFile,
		heartbeatInterval: time.Duration(heartbeatIntervalSec) * time.Second,
		heartbeatTimeout:  time.Duration(heartbeatTimeoutSec) * time.Second,
		reconnectDelay:    time.Duration(reconnectMS) * time.Millisecond,
		scanInterval:      localFeishuScanInterval,
		expireWindow:      localFeishuExpireWindow,
		nowFn:             time.Now,
		state: localFeishuState{
			Pending: map[string]localFeishuPendingMessage{},
			Pushed:  map[string]int64{},
		},
	}
	if err := svc.ensureDirs(); err != nil {
		return nil, err
	}
	if err := svc.loadState(); err != nil {
		return nil, err
	}
	return svc, nil
}

func (s *localFeishuService) ensureDirs() error {
	if err := os.MkdirAll(filepath.Dir(s.stateFile), 0o755); err != nil {
		return err
	}
	return os.MkdirAll(s.downloadDir, 0o755)
}

func (s *localFeishuService) Run(ctx context.Context) error {
	failures := 0
	for {
		meta, cfg, err := s.loadConfig(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			if stopErr := s.handleServiceFailure("load-config", &failures, err); stopErr != nil {
				return stopErr
			}
			if !sleepContextLocalFeishu(ctx, s.reconnectDelay) {
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
			if stopErr := s.handleServiceFailure("create-session", &failures, err); stopErr != nil {
				return stopErr
			}
			if !sleepContextLocalFeishu(ctx, s.reconnectDelay) {
				return nil
			}
			continue
		}
		s.logf("stage=connected name=%s callback=%s", meta.Name, meta.Callback)
		if err := s.runSession(ctx, meta, session); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			if stopErr := s.handleServiceFailure("run-session", &failures, err); stopErr != nil {
				return stopErr
			}
		} else {
			failures = 0
		}
		if !sleepContextLocalFeishu(ctx, s.reconnectDelay) {
			return nil
		}
	}
}

func (s *localFeishuService) loadConfig(ctx context.Context) (*connectsvc.Meta, feishusvc.Config, error) {
	name := s.serviceName()
	meta, err := s.client.GetMeta(ctx, name)
	if err != nil {
		return nil, feishusvc.Config{}, err
	}
	var cfg feishusvc.Config
	if err := json.Unmarshal([]byte(meta.Meta), &cfg); err != nil {
		return nil, feishusvc.Config{}, fmt.Errorf("invalid feishu meta json: %w", err)
	}
	cfg.Mode = strings.ToLower(strings.TrimSpace(cfg.Mode))
	if cfg.Mode == "" {
		cfg.Mode = "feishu"
	}
	if cfg.Mode != "mock" {
		if strings.TrimSpace(cfg.AppID) == "" {
			return nil, feishusvc.Config{}, errors.New("meta.appId is required")
		}
		if strings.TrimSpace(cfg.AppSecret) == "" {
			return nil, feishusvc.Config{}, errors.New("meta.appSecret is required")
		}
	}
	return meta, cfg, nil
}

func (s *localFeishuService) ValidateStartup(ctx context.Context) error {
	_, cfg, err := s.loadConfig(ctx)
	if err != nil {
		return err
	}
	return feishusvc.ValidateConfig(ctx, cfg)
}

func (s *localFeishuService) runSession(ctx context.Context, meta *connectsvc.Meta, session feishusvc.Session) error {
	sessionCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- session.Run(sessionCtx)
	}()

	lastBeat := s.now()
	heartbeatTicker := time.NewTicker(s.heartbeatInterval)
	defer heartbeatTicker.Stop()
	scanTicker := time.NewTicker(s.scanInterval)
	defer scanTicker.Stop()

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
				event.At = s.now()
			}
			if event.Type == feishusvc.SessionEventHeartbeat || event.Type == feishusvc.SessionEventConnected {
				lastBeat = event.At
			}
			if event.Type != feishusvc.SessionEventMessage {
				continue
			}
			lastBeat = event.At
			if s.shouldDropMessage(event.Message) {
				continue
			}
			if err := s.storePending(event.Message, event.At); err != nil {
				s.logf("stage=store-pending external_id=%s err=%v", strings.TrimSpace(event.Message.ExternalID), err)
				continue
			}
			runtimeKey := runtimeKeyFromMetaLocalFeishu(meta)
			if err := s.flushPending(ctx, runtimeKey); err != nil {
				return err
			}
		case <-scanTicker.C:
			runtimeKey := runtimeKeyFromMetaLocalFeishu(meta)
			if err := s.flushPending(ctx, runtimeKey); err != nil {
				return err
			}
		case <-heartbeatTicker.C:
			if s.now().Sub(lastBeat) <= s.heartbeatTimeout {
				continue
			}
			cancel()
			<-errCh
			return fmt.Errorf("heartbeat timeout after %s", s.heartbeatTimeout)
		}
	}
}

func (s *localFeishuService) handleServiceFailure(step string, failures *int, err error) error {
	if err == nil {
		return nil
	}
	if failures == nil {
		return err
	}
	*failures++
	s.logServiceRetry(step, *failures, localFeishuMaxServiceRetries, err)
	if *failures < localFeishuMaxServiceRetries {
		return nil
	}
	s.logServiceTerminate(step, localFeishuMaxServiceRetries, err)
	return fmt.Errorf("%s failed after %d attempts: %w", strings.TrimSpace(step), localFeishuMaxServiceRetries, err)
}

func (s *localFeishuService) logServiceRetry(step string, attempt, max int, err error) {
	line := fmt.Sprintf(
		"stage=service-retry step=%s attempt=%d max=%d err=%s",
		quoteLogValueLocalFeishu(strings.TrimSpace(step)),
		attempt,
		max,
		quoteLogValueLocalFeishu(err.Error()),
	)
	s.writeMessageLog(s.now().Format(time.RFC3339), line)
	s.logf("%s", line)
}

func (s *localFeishuService) logServiceTerminate(step string, max int, err error) {
	line := fmt.Sprintf(
		"stage=service-terminate step=%s max=%d err=%s",
		quoteLogValueLocalFeishu(strings.TrimSpace(step)),
		max,
		quoteLogValueLocalFeishu(err.Error()),
	)
	s.writeMessageLog(s.now().Format(time.RFC3339), line)
	s.logf("%s", line)
}

func (s *localFeishuService) shouldDropMessage(message feishusvc.IncomingMessage) bool {
	if !message.Parsed {
		s.logSkip(fmt.Sprintf("用参数：内容摘要=%s，做了：跳过了一条无法识别的飞书消息", compactMessageLogLocalFeishu(message)))
		return true
	}
	if shouldIgnoreSenderTypeLocalFeishu(message.SenderType) {
		s.logSkip(fmt.Sprintf("用参数：发送方类型=%s；外部编号=%s；内容摘要=%s，做了：跳过了一条插件自己发出的飞书消息", strings.TrimSpace(message.SenderType), message.ExternalID, compactMessageLogLocalFeishu(message)))
		return true
	}
	if message.MessageAtMS > 0 {
		messageAt := time.UnixMilli(message.MessageAtMS)
		if s.now().Sub(messageAt) > 30*time.Minute {
			s.logSkip(fmt.Sprintf("用参数：外部编号=%s；消息时间=%d；内容摘要=%s，做了：跳过了一条过期的飞书消息", message.ExternalID, message.MessageAtMS, compactMessageLogLocalFeishu(message)))
			return true
		}
	}
	return false
}

func (s *localFeishuService) storePending(message feishusvc.IncomingMessage, receivedAt time.Time) error {
	externalID := strings.TrimSpace(message.ExternalID)
	if externalID == "" {
		return errors.New("external id is required")
	}

	item := localFeishuPendingMessage{
		ExternalID: externalID,
		EventID:    strings.TrimSpace(message.EventID),
		MessageID:  strings.TrimSpace(message.MessageID),
		ChatID:     strings.TrimSpace(message.ChatID),
		SenderID:   strings.TrimSpace(message.SenderID),
		SenderType: strings.TrimSpace(message.SenderType),
		Raw:        strings.TrimSpace(message.Raw),
		RawContent: strings.TrimSpace(message.RawContent),
		Type:       normalizePendingTypeLocalFeishu(message.MessageType, message.Content, message.Artifacts),
		Content:    strings.TrimSpace(message.Content),
		Artifacts:  append([]string{}, message.Artifacts...),
		MessageAt:  normalizeMessageAtLocalFeishu(message, receivedAt, s.now()),
		ReceivedAt: receivedAt.UnixMilli(),
	}

	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	s.pruneStateLocked()
	if _, pushed := s.state.Pushed[externalID]; pushed {
		return nil
	}
	if existing, ok := s.state.Pending[externalID]; ok {
		item = mergePendingMessageLocalFeishu(existing, item)
	}
	s.state.Pending[externalID] = item
	return s.saveStateLocked()
}

func (s *localFeishuService) flushPending(ctx context.Context, name string) error {
	s.stateMu.Lock()
	s.pruneStateLocked()
	groups := s.buildReadyGroupsLocked()
	expired := s.collectExpiredLocked()
	if len(groups) == 0 && len(expired) == 0 {
		s.stateMu.Unlock()
		return nil
	}
	snapshotPending := make(map[string]localFeishuPendingMessage, len(s.state.Pending))
	for key, value := range s.state.Pending {
		snapshotPending[key] = value
	}
	s.stateMu.Unlock()

	for _, item := range expired {
		s.logSkip(fmt.Sprintf("用参数：外部编号=%s；消息编号=%s；消息类型=%s；内容摘要=%s，做了：跳过了一条等待太久的飞书消息", item.ExternalID, item.MessageID, item.Type, compactPendingSummaryLocalFeishu(item)))
	}

	for _, group := range groups {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		payload, err := buildPushInputLocalFeishu(group, s.now())
		if err != nil {
			s.logf("stage=build-push name=%s err=%v", name, err)
			continue
		}
		if err := s.pushRequest(ctx, name, payload, group); err != nil {
			s.logf("stage=push-request name=%s external_id=%s err=%v", name, payload.ExternalID, err)
			continue
		}
		s.stateMu.Lock()
		for _, item := range group {
			s.state.Pushed[item.ExternalID] = s.now().Unix()
			delete(s.state.Pending, item.ExternalID)
		}
		if err := s.saveStateLocked(); err != nil {
			s.stateMu.Unlock()
			return err
		}
		s.stateMu.Unlock()
	}

	if len(expired) > 0 {
		s.stateMu.Lock()
		for _, item := range expired {
			delete(s.state.Pending, item.ExternalID)
		}
		if err := s.saveStateLocked(); err != nil {
			s.stateMu.Unlock()
			return err
		}
		s.stateMu.Unlock()
	}
	_ = snapshotPending
	return nil
}

func (s *localFeishuService) pushRequest(ctx context.Context, name string, message feishusvc.IncomingMessage, group []localFeishuPendingMessage) error {
	envelope := localFeishuPushEnvelope{
		Source:     feishusvc.DefaultName,
		ReceivedAt: s.now().Format(time.RFC3339),
		Message:    message,
		Pending:    append([]localFeishuPendingMessage(nil), group...),
		GroupedBy:  "chat_id",
		WindowSecs: int(s.scanInterval / time.Second),
		ExpireSecs: int(s.expireWindow / time.Second),
	}
	rawPayload, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	_, err = s.client.AddRequest(ctx, connectsvc.RequestInput{
		Key:            name,
		ExternalID:     message.ExternalID,
		Name:           name,
		Content:        message.Content,
		Request:        message.Content,
		Artifacts:      strings.Join(message.Artifacts, ","),
		Original:       string(rawPayload),
		RawRequest:     string(rawPayload),
		ResponseSchema: feishusvc.ResponseSchemaJSON(),
		CreatedAt:      buildMessageCreatedAtLocalFeishu(message),
	})
	if err != nil {
		s.writeMessageLog(envelope.ReceivedAt, structuredPushFailureLogLocalFeishu(name, message, group, err))
		return err
	}
	s.writeMessageLog(envelope.ReceivedAt, structuredPushLogLocalFeishu(name, message, group))
	return err
}

func (s *localFeishuService) buildReadyGroupsLocked() [][]localFeishuPendingMessage {
	groupsByChat := make(map[string][]localFeishuPendingMessage)
	keys := make([]string, 0, len(s.state.Pending))
	for key := range s.state.Pending {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		item := s.state.Pending[key]
		chatKey := strings.TrimSpace(item.ChatID)
		if chatKey == "" {
			chatKey = "__default__"
		}
		groupsByChat[chatKey] = append(groupsByChat[chatKey], item)
	}

	chatIDs := make([]string, 0, len(groupsByChat))
	for chatID := range groupsByChat {
		chatIDs = append(chatIDs, chatID)
	}
	sort.Strings(chatIDs)

	out := make([][]localFeishuPendingMessage, 0, len(chatIDs))
	for _, chatID := range chatIDs {
		items := groupsByChat[chatID]
		sort.Slice(items, func(i, j int) bool {
			if items[i].MessageAt == items[j].MessageAt {
				return items[i].ExternalID < items[j].ExternalID
			}
			return items[i].MessageAt < items[j].MessageAt
		})
		if group := selectReadyGroupLocalFeishu(items); len(group) > 0 {
			out = append(out, group)
		}
	}
	return out
}

func (s *localFeishuService) collectExpiredLocked() []localFeishuPendingMessage {
	now := s.now()
	out := make([]localFeishuPendingMessage, 0)
	for _, item := range s.state.Pending {
		messageAt := time.UnixMilli(item.MessageAt)
		if messageAt.IsZero() {
			messageAt = now
		}
		if now.Sub(messageAt) > s.expireWindow {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].MessageAt == out[j].MessageAt {
			return out[i].ExternalID < out[j].ExternalID
		}
		return out[i].MessageAt < out[j].MessageAt
	})
	return out
}

func (s *localFeishuService) pruneStateLocked() {
	if s.state.Pending == nil {
		s.state.Pending = map[string]localFeishuPendingMessage{}
	}
	if s.state.Pushed == nil {
		s.state.Pushed = map[string]int64{}
	}
	cutoff := s.now().Add(-24 * time.Hour).Unix()
	for key, unixTs := range s.state.Pushed {
		if strings.TrimSpace(key) == "" || unixTs < cutoff {
			delete(s.state.Pushed, key)
		}
	}
}

func (s *localFeishuService) loadState() error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	body, err := os.ReadFile(s.stateFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.state = localFeishuState{
				Pending: map[string]localFeishuPendingMessage{},
				Pushed:  map[string]int64{},
			}
			return nil
		}
		return err
	}
	var state localFeishuState
	if err := json.Unmarshal(body, &state); err != nil {
		return err
	}
	if state.Pending == nil {
		state.Pending = map[string]localFeishuPendingMessage{}
	}
	if state.Pushed == nil {
		state.Pushed = map[string]int64{}
	}
	s.state = state
	s.pruneStateLocked()
	return nil
}

func (s *localFeishuService) saveStateLocked() error {
	body, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	tmpFile := s.stateFile + ".tmp"
	if err := os.WriteFile(tmpFile, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpFile, s.stateFile)
}

func (s *localFeishuService) serviceName() string {
	name := strings.TrimSpace(s.name)
	if name == "" {
		return feishusvc.DefaultName
	}
	return name
}

func (s *localFeishuService) now() time.Time {
	if s != nil && s.nowFn != nil {
		return s.nowFn()
	}
	return time.Now()
}

func (s *localFeishuService) writeMessageLog(at, content string) {
	writeTimestampedMessageLogLocalFeishu(s.messageLog, s.logger, at, content)
}

func (s *localFeishuService) logSkip(content string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return
	}
	s.writeMessageLog(s.now().Format(time.RFC3339), content)
	s.logf(content)
}

func (s *localFeishuService) logf(format string, args ...any) {
	if s != nil && s.logger != nil {
		s.logger.Printf(format, args...)
	}
}

func normalizePendingTypeLocalFeishu(messageType, content string, artifacts []string) string {
	switch strings.ToLower(strings.TrimSpace(messageType)) {
	case "text":
		return "text"
	case "image":
		return "image"
	case "file":
		return "file"
	}
	content = strings.TrimSpace(content)
	if content != "" && content != "[image]" && content != "[file]" {
		return "text"
	}
	for _, path := range artifacts {
		if strings.TrimSpace(path) == "" {
			continue
		}
		if detectArtifactKind(path) == "image" {
			return "image"
		}
		return "file"
	}
	return "text"
}

func normalizeMessageAtLocalFeishu(message feishusvc.IncomingMessage, receivedAt, now time.Time) int64 {
	if message.MessageAtMS > 0 {
		return message.MessageAtMS
	}
	if !receivedAt.IsZero() {
		return receivedAt.UnixMilli()
	}
	return now.UnixMilli()
}

func mergePendingMessageLocalFeishu(existing, incoming localFeishuPendingMessage) localFeishuPendingMessage {
	out := existing
	if strings.TrimSpace(out.EventID) == "" {
		out.EventID = incoming.EventID
	}
	if strings.TrimSpace(out.MessageID) == "" {
		out.MessageID = incoming.MessageID
	}
	if strings.TrimSpace(out.ChatID) == "" {
		out.ChatID = incoming.ChatID
	}
	if strings.TrimSpace(out.SenderID) == "" {
		out.SenderID = incoming.SenderID
	}
	if strings.TrimSpace(out.SenderType) == "" {
		out.SenderType = incoming.SenderType
	}
	if strings.TrimSpace(out.Raw) == "" {
		out.Raw = incoming.Raw
	}
	if strings.TrimSpace(out.RawContent) == "" {
		out.RawContent = incoming.RawContent
	}
	if strings.TrimSpace(out.Content) == "" {
		out.Content = incoming.Content
	}
	if len(out.Artifacts) == 0 {
		out.Artifacts = append([]string{}, incoming.Artifacts...)
	} else {
		out.Artifacts = uniqueStrings(append(append([]string{}, out.Artifacts...), incoming.Artifacts...))
	}
	if out.MessageAt == 0 || (incoming.MessageAt > 0 && incoming.MessageAt < out.MessageAt) {
		out.MessageAt = incoming.MessageAt
	}
	if incoming.ReceivedAt > out.ReceivedAt {
		out.ReceivedAt = incoming.ReceivedAt
	}
	if out.Type != "text" && incoming.Type == "text" {
		out.Type = "text"
	}
	return out
}

func selectReadyGroupLocalFeishu(items []localFeishuPendingMessage) []localFeishuPendingMessage {
	if len(items) == 0 {
		return nil
	}
	for _, item := range items {
		if item.Type == "text" && strings.TrimSpace(item.Content) != "" && strings.TrimSpace(item.Content) != "[image]" && strings.TrimSpace(item.Content) != "[file]" {
			return append([]localFeishuPendingMessage(nil), items...)
		}
	}
	return nil
}

func buildPushInputLocalFeishu(group []localFeishuPendingMessage, now time.Time) (feishusvc.IncomingMessage, error) {
	if len(group) == 0 {
		return feishusvc.IncomingMessage{}, errors.New("group is required")
	}
	textIndex := -1
	for idx, item := range group {
		if item.Type == "text" && strings.TrimSpace(item.Content) != "" && strings.TrimSpace(item.Content) != "[image]" && strings.TrimSpace(item.Content) != "[file]" {
			textIndex = idx
		}
	}
	if textIndex < 0 {
		return feishusvc.IncomingMessage{}, errors.New("text message not found")
	}

	textItem := group[textIndex]
	textLines := make([]string, 0, len(group))
	attachmentLines := make([]string, 0, len(group))
	artifacts := make([]string, 0, len(group))
	for _, item := range group {
		normalizedLines, normalizedArtifacts := normalizePendingLinesLocalFeishu(item)
		if item.Type == "text" && strings.TrimSpace(item.Content) != "" && strings.TrimSpace(item.Content) != "[image]" && strings.TrimSpace(item.Content) != "[file]" {
			if body := strings.TrimSpace(item.Content); body != "" {
				textLines = append(textLines, body)
			}
		}
		attachmentLines = append(attachmentLines, normalizedLines...)
		artifacts = append(artifacts, normalizedArtifacts...)
	}
	lines := append(textLines, attachmentLines...)
	content := strings.TrimSpace(strings.Join(lines, " "))
	if content == "" {
		content = strings.TrimSpace(textItem.Content)
	}
	if content == "" {
		return feishusvc.IncomingMessage{}, errors.New("normalized content is empty")
	}
	externalSeed := make([]string, 0, len(group))
	for _, item := range group {
		externalSeed = append(externalSeed, item.ExternalID)
	}
	baseAt := textItem.MessageAt
	if baseAt <= 0 {
		baseAt = now.UnixMilli()
	}
	return feishusvc.IncomingMessage{
		EventID:     firstNonEmptyLocalFeishu(textItem.EventID, group[0].EventID),
		MessageID:   firstNonEmptyLocalFeishu(textItem.MessageID, group[0].MessageID),
		ExternalID:  md5KeyLocalFeishu(baseAt, strings.Join(externalSeed, "|")+"|"+content),
		ChatID:      firstNonEmptyLocalFeishu(textItem.ChatID, group[0].ChatID),
		MessageAtMS: baseAt,
		MessageType: "text",
		SenderID:    firstNonEmptyLocalFeishu(textItem.SenderID, group[0].SenderID),
		SenderType:  firstNonEmptyLocalFeishu(textItem.SenderType, group[0].SenderType),
		Content:     content,
		RawContent:  firstNonEmptyLocalFeishu(textItem.RawContent, textItem.Content),
		Artifacts:   uniqueStrings(artifacts),
		Raw:         firstNonEmptyLocalFeishu(textItem.Raw, group[0].Raw),
		Parsed:      true,
	}, nil
}

func normalizePendingLinesLocalFeishu(item localFeishuPendingMessage) ([]string, []string) {
	lines := make([]string, 0, len(item.Artifacts))
	artifacts := make([]string, 0, len(item.Artifacts))
	for _, path := range item.Artifacts {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		artifacts = append(artifacts, path)
		if detectArtifactKind(path) == "image" {
			lines = append(lines, "[image]"+path)
			continue
		}
		lines = append(lines, "[file]"+path)
	}
	return lines, artifacts
}

func structuredPushLogLocalFeishu(key string, message feishusvc.IncomingMessage, group []localFeishuPendingMessage) string {
	parts := []string{
		"stage=push-request",
		"key=" + quoteLogValueLocalFeishu(key),
		"chat_id=" + quoteLogValueLocalFeishu(message.ChatID),
		"external_id=" + quoteLogValueLocalFeishu(message.ExternalID),
		"message_id=" + quoteLogValueLocalFeishu(message.MessageID),
		"count=" + strconv.Itoa(len(group)),
		"content=" + quoteLogValueLocalFeishu(message.Content),
	}
	if len(message.Artifacts) > 0 {
		parts = append(parts, "artifacts="+quoteLogValueLocalFeishu(strings.Join(message.Artifacts, ",")))
	}
	return strings.Join(parts, " ")
}

func structuredPushFailureLogLocalFeishu(key string, message feishusvc.IncomingMessage, group []localFeishuPendingMessage, err error) string {
	parts := []string{
		"stage=push-request-failed",
		"key=" + quoteLogValueLocalFeishu(key),
		"chat_id=" + quoteLogValueLocalFeishu(message.ChatID),
		"external_id=" + quoteLogValueLocalFeishu(message.ExternalID),
		"message_id=" + quoteLogValueLocalFeishu(message.MessageID),
		"count=" + strconv.Itoa(len(group)),
		"content=" + quoteLogValueLocalFeishu(message.Content),
		"err=" + quoteLogValueLocalFeishu(err.Error()),
	}
	if len(message.Artifacts) > 0 {
		parts = append(parts, "artifacts="+quoteLogValueLocalFeishu(strings.Join(message.Artifacts, ",")))
	}
	return strings.Join(parts, " ")
}

func compactMessageLogLocalFeishu(message feishusvc.IncomingMessage) string {
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

func compactPendingSummaryLocalFeishu(item localFeishuPendingMessage) string {
	body := strings.TrimSpace(item.Content)
	if body == "" {
		body = strings.TrimSpace(item.RawContent)
	}
	if body == "" {
		body = strings.TrimSpace(item.Raw)
	}
	if body != "" {
		return body
	}
	if len(item.Artifacts) > 0 {
		return strings.Join(item.Artifacts, ",")
	}
	return item.Type
}

func shouldIgnoreSenderTypeLocalFeishu(senderType string) bool {
	switch strings.ToLower(strings.TrimSpace(senderType)) {
	case "bot", "app", "application", "assistant", "robot":
		return true
	default:
		return false
	}
}

func buildMessageCreatedAtLocalFeishu(message feishusvc.IncomingMessage) string {
	if message.MessageAtMS <= 0 {
		return ""
	}
	return strconv.FormatInt(message.MessageAtMS, 10)
}

func quoteLogValueLocalFeishu(value string) string {
	return strconv.Quote(strings.TrimSpace(value))
}

func writeTimestampedMessageLogLocalFeishu(messageLog io.Writer, logger *log.Logger, at, content string) {
	if messageLog == nil {
		return
	}
	line := feishusvc.HumanizeLogEntry(strings.TrimSpace(at), strings.TrimSpace(content)) + "\n"
	if _, err := io.WriteString(messageLog, line); err != nil && logger != nil {
		logger.Printf("stage=message-log err=%v", err)
	}
}

func sleepContextLocalFeishu(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func runtimeLogPathForLocalFeishu(messageLogPath string) string {
	messageLogPath = strings.TrimSpace(messageLogPath)
	if messageLogPath == "" {
		messageLogPath = defaultFeishuLogFile
	}
	return filepath.Join(filepath.Dir(messageLogPath), defaultFeishuRunLog)
}

func downloadDirForLogLocalFeishu(messageLogPath string) string {
	messageLogPath = strings.TrimSpace(messageLogPath)
	if messageLogPath == "" {
		messageLogPath = defaultFeishuLogFile
	}
	return filepath.Join(filepath.Dir(messageLogPath), "feishu_artifacts")
}

func stateFileForLogLocalFeishu(messageLogPath string) string {
	messageLogPath = strings.TrimSpace(messageLogPath)
	if messageLogPath == "" {
		messageLogPath = defaultFeishuLogFile
	}
	return filepath.Join(filepath.Dir(messageLogPath), "feishu.pending.json")
}

func buildConnectBaseArgsLocalFeishu(flags map[string]string) ([]string, error) {
	cacheMS, err := connectsvc.IntValue(flags, "connect-cache", 10000)
	if err != nil {
		return nil, err
	}
	baseArgs := make([]string, 0, 6)
	if addr := connectsvc.FirstValue(flags, "addr"); addr != "" {
		baseArgs = append(baseArgs, "--addr", addr)
	}
	if port := connectsvc.FirstValue(flags, "port"); port != "" {
		baseArgs = append(baseArgs, "--port", port)
	}
	baseArgs = append(baseArgs, "--connect-cache", strconv.Itoa(cacheMS))
	return baseArgs, nil
}

func resolveConnectBinaryLocalFeishu(flags map[string]string) string {
	if explicit := strings.TrimSpace(connectsvc.FirstValue(flags, "connect-bin")); explicit != "" {
		if filepath.IsAbs(explicit) {
			return explicit
		}
		if strings.HasPrefix(explicit, ".") || strings.Contains(explicit, "/") || strings.Contains(explicit, string(filepath.Separator)) {
			if abs, err := filepath.Abs(explicit); err == nil {
				return abs
			}
		}
		return explicit
	}

	for _, candidate := range connectBinaryCandidatesLocalFeishu() {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return candidate
		}
	}
	return "connect"
}

func resolveConnectPrefixLocalFeishu(flags map[string]string) []string {
	explicit := strings.TrimSpace(connectsvc.FirstValue(flags, "connect-bin"))
	if explicit != "" {
		base := strings.ToLower(filepath.Base(explicit))
		if base == "proxy" || base == "integration" || strings.HasPrefix(base, "proxy.") || strings.HasPrefix(base, "integration.") {
			return []string{"connect"}
		}
		return nil
	}

	resolved := resolveConnectBinaryLocalFeishu(flags)
	base := strings.ToLower(filepath.Base(resolved))
	if base == "proxy" || base == "integration" || strings.HasPrefix(base, "proxy.") || strings.HasPrefix(base, "integration.") {
		return []string{"connect"}
	}
	return nil
}

func firstNonEmptyLocalFeishu(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func runtimeKeyFromMetaLocalFeishu(meta *connectsvc.Meta) string {
	if meta == nil {
		return feishusvc.DefaultName
	}
	if key := strings.TrimSpace(meta.Key); key != "" {
		return key
	}
	if key := strings.TrimSpace(meta.Name); key == feishusvc.DefaultName {
		return key
	}
	return feishusvc.DefaultName
}

func md5KeyLocalFeishu(createTime int64, content string) string {
	return fmt.Sprintf("%x", md5BytesLocalFeishu([]byte(strconv.FormatInt(createTime, 10)+"\n"+strings.TrimSpace(content))))
}

func md5BytesLocalFeishu(data []byte) [16]byte {
	return md5.Sum(data)
}

func connectBinaryCandidatesLocalFeishu() []string {
	seen := make(map[string]struct{})
	add := func(list *[]string, path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		clean := filepath.Clean(path)
		if _, ok := seen[clean]; ok {
			return
		}
		seen[clean] = struct{}{}
		*list = append(*list, clean)
	}

	var candidates []string
	if exe, err := os.Executable(); err == nil && strings.TrimSpace(exe) != "" {
		exeDir := filepath.Dir(exe)
		add(&candidates, filepath.Join(exeDir, "connect"))
		add(&candidates, filepath.Join(exeDir, "..", "connect", "connect"))
		add(&candidates, filepath.Join(exeDir, "..", "proxy", "proxy"))
		add(&candidates, filepath.Join(exeDir, "..", "integration", "integration"))
		add(&candidates, filepath.Join(exeDir, "..", "module", "connect", "connect"))
		add(&candidates, filepath.Join(exeDir, "..", "module", "proxy", "proxy"))
		add(&candidates, filepath.Join(exeDir, "..", "module", "integration", "integration"))
	}
	if cwd, err := os.Getwd(); err == nil && strings.TrimSpace(cwd) != "" {
		add(&candidates, filepath.Join(cwd, "connect"))
		add(&candidates, filepath.Join(cwd, "..", "connect", "connect"))
		add(&candidates, filepath.Join(cwd, "..", "proxy", "proxy"))
		add(&candidates, filepath.Join(cwd, "..", "integration", "integration"))
		add(&candidates, filepath.Join(cwd, "..", "..", "connect", "connect"))
		add(&candidates, filepath.Join(cwd, "..", "..", "proxy", "proxy"))
		add(&candidates, filepath.Join(cwd, "..", "..", "integration", "integration"))
	}
	return candidates
}
