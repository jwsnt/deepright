package emailsvc

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"net/textproto"
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
	"github.com/emersion/go-message"
	pop3 "github.com/knadh/go-pop3"
	"github.com/saintfish/chardet"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/transform"
)

const (
	DefaultName        = "email"
	DefaultDisplayName = "邮件"
	defaultScanEvery   = 60 * time.Second
	defaultPOP3Port    = 995
	initialTimelineLag = 30 * time.Minute
	stateFileName      = "email.state.json"
	maxServiceRetries  = 3
)

var fallbackDNSAddr = []string{
	"223.5.5.5:53",
	"223.6.6.6:53",
}

type ConnectClient interface {
	GetMeta(ctx context.Context, name string) (*connectsvc.Meta, error)
	AddRequest(ctx context.Context, input connectsvc.RequestInput) (*connectsvc.Request, error)
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
	if err := c.runJSON(ctx, &out, "meta-get", "--key", normalizeMetaKey(name)); err != nil {
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
	if strings.TrimSpace(input.MessageSnapshot) != "" {
		args = append(args, "--message-snapshot", input.MessageSnapshot)
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
		return fmt.Errorf("run %s %s failed: %w", bin, formatCLIArgsForLog(fullArgs), err)
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
	Name        string
	Client      ConnectClient
	Fetcher     MailFetcher
	Logger      *log.Logger
	MessageLog  io.Writer
	DownloadDir string
	StateFile   string
	ScanEvery   time.Duration
	Now         func() time.Time
}

type Service struct {
	name        string
	client      ConnectClient
	fetcher     MailFetcher
	logger      *log.Logger
	messageLog  io.Writer
	downloadDir string
	stateFile   string
	scanEvery   time.Duration
	nowFn       func() time.Time

	stateMu sync.Mutex
	state   serviceState
}

type Config struct {
	Email          string `json:"email"`
	EmailPOP3      string `json:"email_pop3"`
	EmailSMTP      string `json:"email_smtp"`
	EmailPassword  string `json:"email_password"`
	EmailWhitelist string `json:"email_whitelist"`
	Mode           string `json:"mode,omitempty"`
	ScanSeconds    int    `json:"scanSeconds,omitempty"`
}

type MailFetcher interface {
	Fetch(ctx context.Context, cfg Config, since time.Time) ([]FetchedMail, error)
}

type FetchedMail struct {
	UID         string
	MessageID   string
	DateHeader  string
	ReceivedAt  time.Time
	Subject     string
	From        string
	FromAddress string
	Headers     []map[string]string
	TextBody    string
	HTMLBody    string
	RawJSON     string
	RawContent  string
	Artifacts   []string
	Attachments []MailArtifact
	Allowed     bool
	SkipReason  string
	RawHeaders  mail.Header
	ProcessedAt time.Time
}

type MailArtifact struct {
	Kind string
	Key  string
	Path string
}

type mailRawPayload struct {
	Headers []map[string]string `json:"headers"`
	Content string              `json:"content"`
}

type PushPayload struct {
	Source     string      `json:"source"`
	ReceivedAt string      `json:"receivedAt"`
	Message    MailMessage `json:"message"`
}

type MailMessage struct {
	UID              string   `json:"uid,omitempty"`
	MessageID        string   `json:"messageId,omitempty"`
	CreateTime       string   `json:"create_time,omitempty"`
	CreateTimeCompat string   `json:"createTime,omitempty"`
	Subject          string   `json:"subject,omitempty"`
	From             string   `json:"from,omitempty"`
	Content          string   `json:"content,omitempty"`
	Artifacts        []string `json:"artifacts,omitempty"`
	Raw              string   `json:"raw,omitempty"`
}

type serviceState struct {
	LastTimelineUnix int64            `json:"lastTimelineUnix"`
	Processed        map[string]int64 `json:"processed"`
	ProcessedUIDs    map[string]int64 `json:"processedUIDs,omitempty"`
}

var validateEmailConfigFn = func(ctx context.Context, cfg Config) error {
	if err := probePOP3Config(ctx, cfg); err != nil {
		return err
	}
	if err := probeSMTPConfig(ctx, cfg); err != nil {
		return err
	}
	return nil
}

var probePOP3Config = func(ctx context.Context, cfg Config) error {
	host, port, err := normalizePOP3Address(cfg.EmailPOP3)
	if err != nil {
		return err
	}
	conn, err := (realPOP3Client{}).openConn(ctx, host, port)
	if err != nil {
		return err
	}
	defer conn.Quit()
	return authenticatePOP3(conn, strings.TrimSpace(cfg.Email), strings.TrimSpace(cfg.EmailPassword), host)
}

type IncrementalMailFetcher interface {
	ConfigureIncremental(processedUIDs map[string]struct{}, useTimelineFilter bool)
}

func NewService(opts Options) (*Service, error) {
	if opts.Client == nil {
		return nil, errors.New("connect client is required")
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
		downloadDir = "email_artifacts"
	}
	absDownloadDir, err := filepath.Abs(downloadDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absDownloadDir, 0o755); err != nil {
		return nil, fmt.Errorf("create email download dir: %w", err)
	}
	scanEvery := opts.ScanEvery
	if scanEvery <= 0 {
		scanEvery = defaultScanEvery
	}
	stateFile := strings.TrimSpace(opts.StateFile)
	if stateFile == "" {
		stateFile = filepath.Join(filepath.Dir(absDownloadDir), stateFileName)
	}
	absStateFile, err := filepath.Abs(stateFile)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(absStateFile), 0o755); err != nil {
		return nil, err
	}

	svc := &Service{
		name:        name,
		client:      opts.Client,
		logger:      logger,
		messageLog:  messageLog,
		downloadDir: absDownloadDir,
		stateFile:   absStateFile,
		scanEvery:   scanEvery,
		nowFn:       opts.Now,
		state: serviceState{
			Processed: map[string]int64{},
		},
	}
	if svc.nowFn == nil {
		svc.nowFn = time.Now
	}
	if opts.Fetcher != nil {
		svc.fetcher = opts.Fetcher
	} else {
		svc.fetcher = NewPOP3Fetcher(absDownloadDir)
	}
	if err := svc.loadState(); err != nil {
		return nil, err
	}
	if err := svc.ensureInitialTimeline(); err != nil {
		return nil, err
	}
	return svc, nil
}

func (s *Service) Run(ctx context.Context) error {
	failures := 0
	if _, err := s.loadAndScanWithRetry(ctx, &failures); err != nil {
		return err
	}

	ticker := time.NewTicker(s.scanEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			_, nextCfg, err := s.loadConfig(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				if stopErr := s.handleServiceFailure("load-config", &failures, err); stopErr != nil {
					return stopErr
				}
				continue
			}
			if nextCfg.ScanSeconds > 0 && time.Duration(nextCfg.ScanSeconds)*time.Second != s.scanEvery {
				s.scanEvery = time.Duration(nextCfg.ScanSeconds) * time.Second
				ticker.Reset(s.scanEvery)
			}
			if err := s.scanOnce(ctx, nextCfg); err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				if stopErr := s.handleServiceFailure("scan", &failures, err); stopErr != nil {
					return stopErr
				}
				continue
			}
			failures = 0
		}
	}
}

func (s *Service) loadAndScanWithRetry(ctx context.Context, failures *int) (Config, error) {
	for {
		_, cfg, err := s.loadConfig(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return Config{}, nil
			}
			if stopErr := s.handleServiceFailure("load-config", failures, err); stopErr != nil {
				return Config{}, stopErr
			}
			if !sleepContextEmail(ctx, s.scanEvery) {
				return Config{}, nil
			}
			continue
		}
		if cfg.ScanSeconds > 0 {
			s.scanEvery = time.Duration(cfg.ScanSeconds) * time.Second
		}
		if err := s.scanOnce(ctx, cfg); err != nil {
			if errors.Is(err, context.Canceled) {
				return Config{}, nil
			}
			if stopErr := s.handleServiceFailure("scan", failures, err); stopErr != nil {
				return Config{}, stopErr
			}
			if !sleepContextEmail(ctx, s.scanEvery) {
				return Config{}, nil
			}
			continue
		}
		if failures != nil {
			*failures = 0
		}
		return cfg, nil
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
		return nil, Config{}, fmt.Errorf("invalid email meta json: %w", err)
	}
	cfg.Mode = strings.ToLower(strings.TrimSpace(cfg.Mode))
	if cfg.Mode == "" {
		cfg.Mode = DefaultName
	}
	cfg.Email = strings.TrimSpace(cfg.Email)
	cfg.EmailPOP3 = strings.TrimSpace(cfg.EmailPOP3)
	cfg.EmailSMTP = strings.TrimSpace(cfg.EmailSMTP)
	cfg.EmailPassword = strings.TrimSpace(cfg.EmailPassword)
	cfg.EmailWhitelist = strings.TrimSpace(cfg.EmailWhitelist)
	if cfg.Email == "" {
		return nil, Config{}, errors.New("meta.email is required")
	}
	if cfg.EmailPOP3 == "" {
		return nil, Config{}, errors.New("meta.email_pop3 is required")
	}
	if cfg.EmailSMTP == "" {
		return nil, Config{}, errors.New("meta.email_smtp is required")
	}
	if cfg.EmailPassword == "" {
		return nil, Config{}, errors.New("meta.email_password is required")
	}
	return meta, cfg, nil
}

func (s *Service) ValidateStartup(ctx context.Context) error {
	_, cfg, err := s.loadConfig(ctx)
	if err != nil {
		return err
	}
	return validateEmailConfigFn(ctx, cfg)
}

func (s *Service) scanOnce(ctx context.Context, cfg Config) error {
	since := s.timeline()
	if fetcher, ok := s.fetcher.(IncrementalMailFetcher); ok {
		fetcher.ConfigureIncremental(s.processedUIDsSnapshot(), !s.hasProcessedHistory())
	}
	mails, err := s.fetcher.Fetch(ctx, cfg, since)
	if err != nil {
		return err
	}
	sort.Slice(mails, func(i, j int) bool {
		left := effectiveMailTime(mails[i])
		right := effectiveMailTime(mails[j])
		return left.Before(right)
	})

	for _, item := range mails {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		item.Allowed = s.isAllowedSender(cfg, item)
		if !item.Allowed {
			item.SkipReason = "sender not in whitelist"
			s.logger.Printf("stage=mail-skip sender=%s message_id=%s reason=%s", strings.TrimSpace(item.FromAddress), strings.TrimSpace(item.MessageID), item.SkipReason)
			s.writeMessageLog(s.nowFn().Format(time.RFC3339), structuredMailLog("mail-skip", buildMailMessage(item), "reason="+quoteMailLogValue(item.SkipReason)))
			s.recordTimeline(item)
			if err := s.saveState(); err != nil {
				s.logger.Printf("stage=state-save err=%s", formatErrorForLog(err))
			}
			continue
		}
		if s.isProcessedMail(item) {
			s.logger.Printf("stage=mail-skip sender=%s message_id=%s uid=%s reason=duplicate-mail", strings.TrimSpace(item.FromAddress), strings.TrimSpace(item.MessageID), strings.TrimSpace(item.UID))
			s.writeMessageLog(s.nowFn().Format(time.RFC3339), structuredMailLog("mail-skip", buildMailMessage(item), "reason=duplicate-mail"))
			s.recordTimeline(item)
			if err := s.saveState(); err != nil {
				s.logger.Printf("stage=state-save err=%s", formatErrorForLog(err))
			}
			continue
		}
		if err := s.pushRequest(ctx, item); err != nil {
			s.logger.Printf("stage=push-request uid=%s message_id=%s err=%s", item.UID, item.MessageID, formatErrorForLog(err))
			return err
		}
		s.markProcessedMail(item)
		s.recordTimeline(item)
		if err := s.saveState(); err != nil {
			s.logger.Printf("stage=state-save err=%s", formatErrorForLog(err))
		}
	}
	return nil
}

func (s *Service) handleServiceFailure(step string, failures *int, err error) error {
	if err == nil {
		return nil
	}
	if failures == nil {
		return err
	}
	*failures++
	s.logServiceRetry(step, *failures, maxServiceRetries, err)
	if *failures < maxServiceRetries {
		return nil
	}
	s.logServiceTerminate(step, maxServiceRetries, err)
	return fmt.Errorf("%s failed after %d attempts: %w", strings.TrimSpace(step), maxServiceRetries, err)
}

func (s *Service) logServiceRetry(step string, attempt, max int, err error) {
	line := fmt.Sprintf(
		"stage=service-retry step=%s attempt=%d max=%d err=%s",
		quoteMailLogValue(strings.TrimSpace(step)),
		attempt,
		max,
		quoteMailLogValue(formatErrorForLog(err)),
	)
	if s.logger != nil {
		s.logger.Print(line)
	}
	s.writeMessageLog(s.nowFn().Format(time.RFC3339), line)
}

func (s *Service) logServiceTerminate(step string, max int, err error) {
	line := fmt.Sprintf(
		"stage=service-terminate step=%s max=%d err=%s",
		quoteMailLogValue(strings.TrimSpace(step)),
		max,
		quoteMailLogValue(formatErrorForLog(err)),
	)
	if s.logger != nil {
		s.logger.Print(line)
	}
	s.writeMessageLog(s.nowFn().Format(time.RFC3339), line)
}

func sleepContextEmail(ctx context.Context, d time.Duration) bool {
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

func (s *Service) pushRequest(ctx context.Context, mailItem FetchedMail) error {
	content := normalizeMailContent(mailItem)
	message := buildMailMessage(mailItem)
	message.Content = content
	message.Artifacts = append([]string{}, mailItem.Artifacts...)
	payload := PushPayload{
		Source:     DefaultName,
		ReceivedAt: s.nowFn().Format(time.RFC3339),
		Message:    message,
	}
	s.writeMessageLog(payload.ReceivedAt, structuredMailLog("mail-received", message))

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = s.client.AddRequest(ctx, connectsvc.RequestInput{
		Key:             DefaultName,
		ExternalID:      buildExternalID(mailItem),
		Name:            DefaultName,
		Content:         content,
		Request:         content,
		Artifacts:       strings.Join(mailItem.Artifacts, ","),
		Original:        string(rawPayload),
		RawRequest:      string(rawPayload),
		ResponseSchema:  ResponseSchemaJSON(),
		MessageSnapshot: buildEmailMessageSnapshot(mailItem, s.nowFn()),
		CreatedAt:       buildMailCreatedAt(mailItem),
	})
	if err != nil {
		return err
	}
	return nil
}

func buildEmailMessageSnapshot(mailItem FetchedMail, receivedAt time.Time) string {
	sender := strings.ToLower(strings.TrimSpace(firstNonEmpty(mailItem.FromAddress, extractMailAddress(mailItem.From))))
	messageID := strings.TrimSpace(buildExternalID(mailItem))
	if sender == "" || messageID == "" {
		return ""
	}
	sentAt := parseMailDate(mailItem.DateHeader)
	if sentAt.IsZero() {
		sentAt = mailItem.ReceivedAt
	}
	if sentAt.IsZero() {
		sentAt = receivedAt
	}
	if sentAt.IsZero() {
		return ""
	}
	content := normalizeMailSnapshotContent(mailItem)
	messageType := "attachment"
	if content != "" {
		messageType = "text"
	}
	body, err := json.Marshal(connectsvc.MessageSnapshot{
		Source: DefaultName,
		Messages: []connectsvc.MessageSnapshotRecord{{
			MessageID:   messageID,
			SenderID:    sender,
			Content:     content,
			MessageType: messageType,
			SentAt:      sentAt.UnixMilli(),
		}},
	})
	if err != nil {
		return ""
	}
	return string(body)
}

func normalizeMailSnapshotContent(mailItem FetchedMail) string {
	subject := strings.TrimSpace(decodeHeaderValue(mailItem.Subject))
	content := strings.TrimSpace(mailItem.TextBody)
	if content == "" {
		content = strings.TrimSpace(htmlToText(mailItem.HTMLBody))
	}
	return strings.TrimSpace(strings.Join(nonEmptyStrings(subject, content), "\n"))
}

func nonEmptyStrings(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}

func buildMailMessage(mailItem FetchedMail) MailMessage {
	createTime := firstNonEmpty(strings.TrimSpace(mailItem.DateHeader), effectiveMailTime(mailItem).Format(time.RFC3339))
	return MailMessage{
		UID:              strings.TrimSpace(mailItem.UID),
		MessageID:        strings.TrimSpace(mailItem.MessageID),
		CreateTime:       createTime,
		CreateTimeCompat: createTime,
		Subject:          strings.TrimSpace(decodeHeaderValue(mailItem.Subject)),
		From:             strings.TrimSpace(mailItem.From),
		Content:          normalizeMailContent(mailItem),
		Artifacts:        append([]string{}, mailItem.Artifacts...),
		Raw:              firstNonEmpty(mailItem.RawJSON, mailItem.RawContent),
	}
}

func formatCLIArgsForLog(args []string) string {
	if len(args) == 0 {
		return ""
	}
	formatted := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		current := strings.TrimSpace(args[index])
		if current == "" {
			continue
		}
		formatted = append(formatted, current)
		if !strings.HasPrefix(current, "--") || index+1 >= len(args) {
			continue
		}
		index++
		formatted = append(formatted, strconv.Quote(sanitizeCLIValueForLog(current, args[index])))
	}
	return strings.Join(formatted, " ")
}

func sanitizeCLIValueForLog(flag, value string) string {
	value = strings.TrimSpace(value)
	switch strings.TrimSpace(flag) {
	case "--content":
		return decodePossiblyEncodedText(value)
	case "--original", "--schema", "--message":
		return sanitizeJSONTextForLog(value)
	default:
		return value
	}
}

func sanitizeJSONTextForLog(text string) string {
	text = strings.TrimSpace(text)
	if text == "" || !strings.HasPrefix(text, "{") {
		return decodePossiblyEncodedText(text)
	}
	var value any
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		return decodePossiblyEncodedText(text)
	}
	sanitizeJSONValueForLog(&value)
	body, err := json.Marshal(value)
	if err != nil {
		return decodePossiblyEncodedText(text)
	}
	return string(body)
}

func sanitizeJSONValueForLog(value *any) {
	if value == nil || *value == nil {
		return
	}
	switch current := (*value).(type) {
	case string:
		*value = decodePossiblyEncodedText(current)
	case []any:
		for index := range current {
			item := current[index]
			sanitizeJSONValueForLog(&item)
			current[index] = item
		}
		*value = current
	case map[string]any:
		for key, item := range current {
			sanitizeJSONValueForLog(&item)
			current[key] = item
		}
		*value = current
	}
}

func buildMailCreatedAt(mailItem FetchedMail) string {
	at := effectiveMailTime(mailItem)
	if at.IsZero() {
		return ""
	}
	return strconv.FormatInt(at.Unix(), 10)
}

func responseSchemaJSON() string {
	return ResponseSchemaJSON()
}

func (s *Service) writeMessageLog(at, content string) {
	writeTimestampedMessageLog(s.messageLog, s.logger, at, content)
}

func compactMailLog(message MailMessage) string {
	content := toUTF8String([]byte(compactMailSummary(message)))
	content = strings.ReplaceAll(content, "\n", "\\n")
	content = strings.ReplaceAll(content, "\r", "\\r")
	return content
}

func compactMailSummary(message MailMessage) string {
	subject := mailLogSubject(message)
	content := toUTF8String([]byte(strings.TrimSpace(message.Content)))
	decodedContent := strings.TrimSpace(decodeHeaderValue(content))
	raw := toUTF8String([]byte(strings.TrimSpace(message.Raw)))

	lines := make([]string, 0, 3)
	if subject != "" {
		lines = append(lines, subject)
	}
	switch {
	case decodedContent != "" && decodedContent != subject:
		lines = append(lines, decodedContent)
	case decodedContent == "" && content != "" && content != subject:
		lines = append(lines, content)
	}
	if len(lines) == 0 && raw != "" {
		if subject == "" {
			subject = decodeHeaderValue(extractHeaderValueFromRawJSON(raw, "Subject"))
			if strings.TrimSpace(subject) != "" {
				lines = append(lines, strings.TrimSpace(subject))
			}
		}
		if len(lines) == 0 {
			lines = append(lines, raw)
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func structuredMailLog(stage string, message MailMessage, extras ...string) string {
	fields := []string{
		"stage=" + firstNonEmpty(strings.TrimSpace(stage), "mail"),
	}
	if subject := mailLogSubject(message); subject != "" {
		fields = append(fields, "subject="+quoteMailLogValue(subject))
	}
	if from := mailLogFrom(message); from != "" {
		fields = append(fields, "from="+quoteMailLogValue(from))
	}
	if messageID := mailLogMessageID(message); messageID != "" {
		fields = append(fields, "message_id="+quoteMailLogValue(messageID))
	}
	if summary := strings.TrimSpace(compactMailSummary(message)); summary != "" {
		fields = append(fields, "summary="+quoteMailLogValue(summary))
	}
	for _, extra := range extras {
		extra = strings.TrimSpace(extra)
		if extra == "" {
			continue
		}
		fields = append(fields, extra)
	}
	return strings.Join(fields, " ")
}

func mailLogSubject(message MailMessage) string {
	subject := strings.TrimSpace(decodeHeaderValue(message.Subject))
	if subject != "" {
		return subject
	}
	return strings.TrimSpace(decodeHeaderValue(firstNonEmpty(
		extractHeaderValueFromRawJSON(message.Raw, "Subject"),
		extractHeaderValueFromRawJSON(message.Raw, "subject"),
	)))
}

func mailLogFrom(message MailMessage) string {
	from := strings.TrimSpace(decodeHeaderValue(message.From))
	if from != "" {
		return from
	}
	return strings.TrimSpace(decodeHeaderValue(firstNonEmpty(
		extractHeaderValueFromRawJSON(message.Raw, "From"),
		extractHeaderValueFromRawJSON(message.Raw, "from"),
	)))
}

func mailLogMessageID(message MailMessage) string {
	return strings.TrimSpace(normalizeMessageID(firstNonEmpty(
		message.MessageID,
		extractHeaderValueFromRawJSON(message.Raw, "Message-ID"),
		extractHeaderValueFromRawJSON(message.Raw, "Message-Id"),
		extractHeaderValueFromRawJSON(message.Raw, "message_id"),
		extractHeaderValueFromRawJSON(message.Raw, "messageId"),
	)))
}

func quoteMailLogValue(value string) string {
	return strconv.Quote(toUTF8String([]byte(strings.TrimSpace(value))))
}

func extractHeaderValueFromRawJSON(raw, key string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || !strings.HasPrefix(raw, "{") {
		return ""
	}
	var payload mailRawPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return ""
	}
	return headerValue(payload.Headers, key)
}

func normalizeMailContent(mailItem FetchedMail) string {
	subject := strings.TrimSpace(decodeHeaderValue(mailItem.Subject))
	content := strings.TrimSpace(mailItem.TextBody)
	if content == "" {
		content = htmlToText(mailItem.HTMLBody)
	}
	lines := make([]string, 0, len(mailItem.Artifacts)+2)
	if subject != "" {
		lines = append(lines, subject)
	}
	if content != "" {
		lines = append(lines, content)
	}
	lines = append(lines, normalizeArtifactLines(mailItem.Artifacts)...)
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func normalizeArtifactLines(paths []string) []string {
	lines := make([]string, 0, len(paths))
	for _, path := range paths {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			continue
		}
		if isImagePath(trimmed) {
			lines = append(lines, "[image]"+trimmed)
			continue
		}
		lines = append(lines, "[file]"+trimmed)
	}
	return lines
}

func buildExternalID(mailItem FetchedMail) string {
	base := firstNonEmpty(strings.TrimSpace(mailItem.MessageID), strings.TrimSpace(mailItem.UID))
	if base != "" {
		return base
	}
	return md5Text(strings.TrimSpace(mailItem.DateHeader) + ":" + strings.TrimSpace(mailItem.RawContent))
}

func fallbackFetchedMail(item RawMail) FetchedMail {
	rawHeaders := rawMailHeader(item.Raw)
	headers := mailHeaderToJSON(rawHeaders)
	rawContent := toUTF8String(item.Raw)
	result := FetchedMail{
		UID:         strings.TrimSpace(item.UID),
		MessageID:   normalizeMessageID(rawHeaders.Get("Message-ID"), rawHeaders.Get("Message-Id")),
		DateHeader:  strings.TrimSpace(rawHeaders.Get("Date")),
		Subject:     decodeHeaderValue(rawHeaders.Get("Subject")),
		From:        decodeHeaderValue(rawHeaders.Get("From")),
		FromAddress: extractMailAddress(rawHeaders.Get("From")),
		Headers:     headers,
		RawHeaders:  rawHeaders,
		TextBody:    rawContent,
		RawContent:  rawContent,
		ProcessedAt: time.Now(),
	}
	result.ReceivedAt = parseBestReceivedAt(result.DateHeader, rawHeaders, headers)
	if item.ID > 0 && result.UID == "" {
		result.UID = strconv.Itoa(item.ID)
	}
	rawJSON, err := json.Marshal(mailRawPayload{
		Headers: result.Headers,
		Content: firstNonEmpty(result.TextBody, result.RawContent),
	})
	if err == nil {
		result.RawJSON = string(rawJSON)
	}
	return finalizeFetchedMail(result)
}

func rawMailHeader(raw []byte) mail.Header {
	reader := textproto.NewReader(bufio.NewReader(bytes.NewReader(raw)))
	header, err := reader.ReadMIMEHeader()
	if err != nil {
		return mail.Header{}
	}
	return mail.Header(header)
}

func effectiveMailTime(item FetchedMail) time.Time {
	if !item.ReceivedAt.IsZero() {
		return item.ReceivedAt
	}
	if !item.ProcessedAt.IsZero() {
		return item.ProcessedAt
	}
	return time.Now()
}

func parseBestReceivedAt(dateHeader string, header mail.Header, headers []map[string]string) time.Time {
	best := parseMailDate(dateHeader)
	for _, value := range receivedHeaderValues(header, headers) {
		if parsed := parseReceivedHeaderTime(value); parsed.After(best) {
			best = parsed
		}
	}
	return best
}

func parseMailDate(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	parsed, err := mail.ParseDate(value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func receivedHeaderValues(header mail.Header, headers []map[string]string) []string {
	values := append([]string{}, header[textproto.CanonicalMIMEHeaderKey("Received")]...)
	if len(values) > 0 {
		return values
	}
	for _, item := range headers {
		if strings.EqualFold(strings.TrimSpace(item["name"]), "Received") {
			values = append(values, strings.TrimSpace(item["value"]))
		}
	}
	return values
}

func parseReceivedHeaderTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	if index := strings.LastIndex(value, ";"); index >= 0 {
		if parsed := parseMailDate(value[index+1:]); !parsed.IsZero() {
			return parsed
		}
	}
	return parseMailDate(value)
}

func (s *Service) isAllowedSender(cfg Config, item FetchedMail) bool {
	address := strings.ToLower(strings.TrimSpace(item.FromAddress))
	if address == "" {
		address = strings.ToLower(strings.TrimSpace(extractMailAddress(item.From)))
	}
	if address == "" {
		return false
	}
	if strings.TrimSpace(cfg.EmailWhitelist) == "" {
		return true
	}
	if address == strings.ToLower(strings.TrimSpace(cfg.Email)) {
		return true
	}
	for _, allow := range strings.Split(cfg.EmailWhitelist, ",") {
		if address == strings.ToLower(strings.TrimSpace(allow)) {
			return true
		}
	}
	return false
}

func (s *Service) loadState() error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	data, err := os.ReadFile(s.stateFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.state = serviceState{Processed: map[string]int64{}}
			return nil
		}
		return err
	}
	var state serviceState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	if state.Processed == nil {
		state.Processed = map[string]int64{}
	}
	if state.ProcessedUIDs == nil {
		state.ProcessedUIDs = map[string]int64{}
	}
	s.state = state
	s.pruneProcessedLocked()
	return nil
}

func (s *Service) ensureInitialTimeline() error {
	s.stateMu.Lock()
	needsInit := s.state.LastTimelineUnix <= 0
	if needsInit {
		s.state.LastTimelineUnix = s.nowFn().Add(-initialTimelineLag).Unix()
	}
	s.stateMu.Unlock()
	if !needsInit {
		return nil
	}
	return s.saveState()
}

func (s *Service) saveState() error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	s.pruneProcessedLocked()
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

func (s *Service) pruneProcessedLocked() {
	if s.state.Processed == nil {
		s.state.Processed = map[string]int64{}
	}
	if s.state.ProcessedUIDs == nil {
		s.state.ProcessedUIDs = map[string]int64{}
	}
	cutoff := s.nowFn().Add(-30 * 24 * time.Hour).Unix()
	for key, unixTs := range s.state.Processed {
		if strings.TrimSpace(key) == "" || unixTs < cutoff {
			delete(s.state.Processed, key)
		}
	}
	for key, unixTs := range s.state.ProcessedUIDs {
		if strings.TrimSpace(key) == "" || unixTs < cutoff {
			delete(s.state.ProcessedUIDs, key)
		}
	}
}

func (s *Service) timeline() time.Time {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.state.LastTimelineUnix <= 0 {
		return time.Time{}
	}
	return time.Unix(s.state.LastTimelineUnix, 0)
}

func (s *Service) recordTimeline(item FetchedMail) {
	at := effectiveMailTime(item)
	if at.IsZero() {
		return
	}
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	unixTs := at.Unix()
	if unixTs > s.state.LastTimelineUnix {
		s.state.LastTimelineUnix = unixTs
	}
}

func (s *Service) isProcessed(messageID string) bool {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return false
	}
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	_, ok := s.state.Processed[messageID]
	return ok
}

func (s *Service) isProcessedUID(uid string) bool {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return false
	}
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	_, ok := s.state.ProcessedUIDs[uid]
	return ok
}

func (s *Service) isProcessedMail(item FetchedMail) bool {
	return s.isProcessed(item.MessageID) || s.isProcessedUID(item.UID)
}

func (s *Service) markProcessed(messageID string) {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return
	}
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.state.Processed == nil {
		s.state.Processed = map[string]int64{}
	}
	s.state.Processed[messageID] = s.nowFn().Unix()
}

func (s *Service) markProcessedUID(uid string) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return
	}
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.state.ProcessedUIDs == nil {
		s.state.ProcessedUIDs = map[string]int64{}
	}
	s.state.ProcessedUIDs[uid] = s.nowFn().Unix()
}

func (s *Service) markProcessedMail(item FetchedMail) {
	s.markProcessed(item.MessageID)
	s.markProcessedUID(item.UID)
}

func (s *Service) processedUIDsSnapshot() map[string]struct{} {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	out := make(map[string]struct{}, len(s.state.ProcessedUIDs))
	for uid := range s.state.ProcessedUIDs {
		uid = strings.TrimSpace(uid)
		if uid == "" {
			continue
		}
		out[uid] = struct{}{}
	}
	return out
}

func (s *Service) hasProcessedHistory() bool {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	return len(s.state.Processed) > 0 || len(s.state.ProcessedUIDs) > 0
}

type POP3Fetcher struct {
	downloadDir string
	client      POP3Client
}

type POP3Client interface {
	Fetch(ctx context.Context, cfg Config, since time.Time) ([]RawMail, error)
}

type RawMail struct {
	ID  int
	UID string
	Raw []byte
}

type pop3Session interface {
	Auth(user, password string) error
	Uidl(msgID int) ([]pop3.MessageID, error)
	RetrRaw(msgID int) (*bytes.Buffer, error)
	Quit() error
}

func NewPOP3Fetcher(downloadDir string) *POP3Fetcher {
	return &POP3Fetcher{
		downloadDir: strings.TrimSpace(downloadDir),
		client:      realPOP3Client{},
	}
}

func (f *POP3Fetcher) Fetch(ctx context.Context, cfg Config, since time.Time) ([]FetchedMail, error) {
	if f.client == nil {
		f.client = realPOP3Client{}
	}
	rawMessages, err := f.client.Fetch(ctx, cfg, since)
	if err != nil {
		return nil, err
	}
	out := make([]FetchedMail, 0, len(rawMessages))
	for _, item := range rawMessages {
		parsed, err := parseFetchedMail(item, f.downloadDir)
		if err != nil {
			parsed = fallbackFetchedMail(item)
		}
		if !since.IsZero() && !effectiveMailTime(parsed).After(since) {
			continue
		}
		out = append(out, parsed)
	}
	return out, nil
}

type realPOP3Client struct {
	newConn func(ctx context.Context, host string, port int) (pop3Session, error)
}

func (realPOP3Client) Fetch(ctx context.Context, cfg Config, since time.Time) ([]RawMail, error) {
	return realPOP3Client{}.fetch(ctx, cfg, since)
}

func (c realPOP3Client) fetch(ctx context.Context, cfg Config, since time.Time) ([]RawMail, error) {
	host, port, err := normalizePOP3Address(cfg.EmailPOP3)
	if err != nil {
		return nil, err
	}
	conn, err := c.openConn(ctx, host, port)
	if err != nil {
		return nil, err
	}
	defer conn.Quit()

	if err := authenticatePOP3(conn, strings.TrimSpace(cfg.Email), strings.TrimSpace(cfg.EmailPassword), host); err != nil {
		return nil, err
	}
	uidls, err := conn.Uidl(0)
	if err != nil {
		return nil, err
	}
	sort.Slice(uidls, func(i, j int) bool {
		return uidls[i].ID < uidls[j].ID
	})
	out := make([]RawMail, 0, len(uidls))
	for _, item := range uidls {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		buf, err := conn.RetrRaw(item.ID)
		if err != nil {
			return nil, err
		}
		raw := append([]byte(nil), buf.Bytes()...)
		out = append(out, RawMail{
			ID:  item.ID,
			UID: strings.TrimSpace(item.UID),
			Raw: raw,
		})
	}
	return out, nil
}

func (c realPOP3Client) openConn(ctx context.Context, host string, port int) (pop3Session, error) {
	if c.newConn != nil {
		return c.newConn(ctx, host, port)
	}
	client := pop3.New(pop3.Opt{
		Host:          host,
		Port:          port,
		TLSEnabled:    false,
		TLSSkipVerify: false,
		DialTimeout:   10 * time.Second,
		Dialer:        &tlsContextDialer{ctx: ctx, host: host},
	})
	return client.NewConn()
}

func authenticatePOP3(conn pop3Session, email, password, host string) error {
	candidates := pop3UsernameCandidates(email)
	var errs []string
	for index, candidate := range candidates {
		if err := conn.Auth(candidate, password); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", candidate, err))
			if index == 0 && shouldRetryPOP3WithLocalPart(err) && len(candidates) > 1 {
				continue
			}
			return fmt.Errorf("pop3 auth failed server=%s account=%s username=%s err=%w", strings.TrimSpace(host), maskEmailAccount(email), candidate, err)
		}
		return nil
	}
	if len(errs) > 0 {
		return fmt.Errorf("pop3 auth failed server=%s account=%s tried=%s", strings.TrimSpace(host), maskEmailAccount(email), strings.Join(errs, "; "))
	}
	return errors.New("pop3 auth failed")
}

func pop3UsernameCandidates(email string) []string {
	email = strings.TrimSpace(email)
	if email == "" {
		return nil
	}
	out := []string{email}
	at := strings.Index(email, "@")
	if at > 0 {
		localPart := strings.TrimSpace(email[:at])
		if localPart != "" && !strings.EqualFold(localPart, email) {
			out = append(out, localPart)
		}
	}
	return out
}

func shouldRetryPOP3WithLocalPart(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(strings.TrimSpace(formatErrorForLog(err)))
	if text == "" {
		return false
	}
	return strings.Contains(text, "username not valid") ||
		strings.Contains(text, "invalid user") ||
		strings.Contains(text, "unknown user") ||
		strings.Contains(text, "unknown username") ||
		strings.Contains(text, "no such user") ||
		strings.Contains(text, "user unknown")
}

func maskEmailAccount(email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return ""
	}
	at := strings.Index(email, "@")
	if at <= 1 || at == len(email)-1 {
		return email
	}
	return email[:1] + "***" + email[at:]
}

func formatErrorForLog(err error) string {
	if err == nil {
		return ""
	}
	return decodePossiblyEncodedText(err.Error())
}

func decodePossiblyEncodedText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	text = decodeRFC2047Words(text)
	if utf8Valid(text) {
		return text
	}
	raw := []byte(text)
	charsetName, ok := detectTextCharset(raw)
	if ok {
		decoded, ok := decodeBytesWithCharsetName(raw, charsetName)
		if ok {
			return decoded
		}
	}
	return strings.TrimSpace(strings.ToValidUTF8(text, ""))
}

func decodeRFC2047Words(text string) string {
	text = strings.TrimSpace(text)
	if text == "" || !strings.Contains(text, "=?") {
		return text
	}
	re := regexp.MustCompile(`=\?[^?\s]+\?[bBqQ]\?[^?]+\?=`)
	decoder := &mime.WordDecoder{CharsetReader: wordDecoderCharsetReader}
	return strings.TrimSpace(re.ReplaceAllStringFunc(text, func(part string) string {
		decoded, err := decoder.DecodeHeader(part)
		if err != nil {
			return part
		}
		return strings.TrimSpace(decoded)
	}))
}

func detectTextCharset(data []byte) (string, bool) {
	detector := chardet.NewTextDetector()
	result, err := detector.DetectBest(data)
	if err != nil || result == nil {
		return "", false
	}
	charsetName := strings.TrimSpace(result.Charset)
	if charsetName == "" {
		return "", false
	}
	return charsetName, true
}

func decodeBytesWithCharsetName(data []byte, charsetName string) (string, bool) {
	charsetName = normalizeCharsetLabel(charsetName)
	if charsetName == "" {
		return "", false
	}
	enc, err := htmlindex.Get(charsetName)
	if err != nil {
		return "", false
	}
	reader := transform.NewReader(bytes.NewReader(data), enc.NewDecoder())
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return "", false
	}
	text := strings.TrimSpace(toUTF8String(decoded))
	if text == "" {
		return "", false
	}
	return text, true
}

func normalizeCharsetLabel(charsetName string) string {
	charsetName = strings.TrimSpace(strings.ToLower(charsetName))
	if charsetName == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(charsetName))
	for _, r := range charsetName {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

type tlsContextDialer struct {
	ctx  context.Context
	host string
}

func (d *tlsContextDialer) Dial(network, address string) (net.Conn, error) {
	return dialTLSWithFallback(d.ctx, strings.TrimSpace(d.host), network, address)
}

func dialTLSWithFallback(ctx context.Context, host, network, address string) (net.Conn, error) {
	netDialer := &net.Dialer{Timeout: 10 * time.Second}
	tlsDialer := &tls.Dialer{
		NetDialer: netDialer,
		Config: &tls.Config{
			ServerName: host,
		},
	}
	conn, err := tlsDialer.DialContext(ctx, network, address)
	if err == nil {
		return conn, nil
	}

	targets := fallbackDialTargets(ctx, host, address)
	if len(targets) == 0 {
		return nil, err
	}
	lastErr := err
	for _, target := range targets {
		conn, dialErr := tlsDialer.DialContext(ctx, network, target)
		if dialErr == nil {
			return conn, nil
		}
		lastErr = dialErr
	}
	return nil, lastErr
}

func fallbackDialTargets(ctx context.Context, host, address string) []string {
	host = strings.TrimSpace(host)
	if host == "" || net.ParseIP(host) != nil {
		return nil
	}
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil
	}
	ips := lookupFallbackIPs(ctx, host)
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		if !isUsableFallbackIP(ip) {
			continue
		}
		target := net.JoinHostPort(ip.String(), port)
		if _, ok := seen[target]; ok {
			continue
		}
		seen[target] = struct{}{}
		out = append(out, target)
	}
	return out
}

func lookupFallbackIPs(ctx context.Context, host string) []net.IP {
	var out []net.IP
	for _, dnsAddr := range fallbackDNSAddr {
		resolver := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
				return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, network, dnsAddr)
			},
		}
		for _, family := range []string{"ip6", "ip4"} {
			ips, err := resolver.LookupIP(ctx, family, host)
			if err != nil {
				continue
			}
			out = append(out, ips...)
		}
		if len(out) > 0 {
			return out
		}
	}
	return nil
}

func isUsableFallbackIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 198 && (ip4[1] == 18 || ip4[1] == 19) {
			return false
		}
		return ip4.IsGlobalUnicast()
	}
	return ip.IsGlobalUnicast()
}

func normalizePOP3Address(value string) (string, int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", 0, errors.New("pop3 server is required")
	}
	host, portText, err := net.SplitHostPort(value)
	if err == nil {
		port, convErr := strconv.Atoi(portText)
		if convErr != nil {
			return "", 0, convErr
		}
		return host, port, nil
	}
	if strings.Contains(value, ":") && !strings.Contains(value, "]") {
		return "", 0, err
	}
	return value, defaultPOP3Port, nil
}

func parseFetchedMail(item RawMail, downloadDir string) (FetchedMail, error) {
	message.CharsetReader = newCharsetReader
	entity, err := message.Read(bytes.NewReader(item.Raw))
	if err != nil && !message.IsUnknownCharset(err) {
		return FetchedMail{}, err
	}
	if entity == nil {
		msg, mailErr := mail.ReadMessage(bytes.NewReader(item.Raw))
		if mailErr != nil {
			return FetchedMail{}, mailErr
		}
		return parseFetchedMailFallback(item, msg, downloadDir)
	}
	result := FetchedMail{
		UID:         strings.TrimSpace(item.UID),
		RawContent:  toUTF8String(item.Raw),
		RawHeaders:  mail.Header{},
		ProcessedAt: time.Now(),
	}
	if item.ID > 0 && result.UID == "" {
		result.UID = strconv.Itoa(item.ID)
	}
	headers := collectEntityHeaders(entity.Header)
	result.Headers = headers
	result.RawHeaders = headersToMailHeader(headers)
	result.DateHeader = headerValue(headers, "Date")
	result.ReceivedAt = parseBestReceivedAt(result.DateHeader, result.RawHeaders, headers)
	result.Subject = decodeHeaderValue(headerValue(headers, "Subject"))
	result.From = decodeHeaderValue(headerValue(headers, "From"))
	result.FromAddress = extractMailAddress(result.From)
	result.MessageID = normalizeMessageID(headerValue(headers, "Message-ID"), headerValue(headers, "Message-Id"))
	if err := consumeEntity(entity, downloadDir, &result); err != nil {
		return FetchedMail{}, err
	}
	result = finalizeFetchedMail(result)
	rawJSON, err := json.Marshal(mailRawPayload{
		Headers: result.Headers,
		Content: firstNonEmpty(result.TextBody, htmlToText(result.HTMLBody), result.RawContent),
	})
	if err != nil {
		return FetchedMail{}, err
	}
	result.RawJSON = string(rawJSON)
	return result, nil
}

func parseFetchedMailFallback(item RawMail, msg *mail.Message, downloadDir string) (FetchedMail, error) {
	header := msg.Header
	body, err := io.ReadAll(msg.Body)
	if err != nil {
		return FetchedMail{}, err
	}
	dateHeader := strings.TrimSpace(header.Get("Date"))
	subject := decodeHeaderValue(header.Get("Subject"))
	from := decodeHeaderValue(header.Get("From"))
	headers := mailHeaderToJSON(header)
	result := FetchedMail{
		UID:         strings.TrimSpace(item.UID),
		MessageID:   normalizeMessageID(header.Get("Message-ID"), header.Get("Message-Id")),
		DateHeader:  dateHeader,
		Subject:     subject,
		From:        from,
		FromAddress: extractMailAddress(from),
		RawContent:  toUTF8String(item.Raw),
		ProcessedAt: time.Now(),
		RawHeaders:  header,
		Headers:     headers,
	}
	result.ReceivedAt = parseBestReceivedAt(dateHeader, header, headers)
	if item.ID > 0 && result.UID == "" {
		result.UID = strconv.Itoa(item.ID)
	}
	mediaType, params, _ := mime.ParseMediaType(header.Get("Content-Type"))
	if strings.HasPrefix(strings.ToLower(mediaType), "multipart/") {
		if err := walkMultipart(bytes.NewReader(body), params["boundary"], downloadDir, &result); err != nil {
			return FetchedMail{}, err
		}
	} else if strings.Contains(strings.ToLower(mediaType), "text/html") {
		result.HTMLBody = decodeBodyWithCharset(body, params["charset"])
	} else {
		result.TextBody = strings.TrimSpace(decodeBodyWithCharset(body, params["charset"]))
	}
	result = finalizeFetchedMail(result)
	rawJSON, err := json.Marshal(mailRawPayload{
		Headers: result.Headers,
		Content: firstNonEmpty(result.TextBody, htmlToText(result.HTMLBody), result.RawContent),
	})
	if err != nil {
		return FetchedMail{}, err
	}
	result.RawJSON = string(rawJSON)
	return result, nil
}

func consumeEntity(entity *message.Entity, downloadDir string, result *FetchedMail) error {
	if entity == nil || result == nil {
		return nil
	}
	mediaType, params, _ := entity.Header.ContentType()
	switch {
	case strings.HasPrefix(strings.ToLower(mediaType), "multipart/"):
		reader := entity.MultipartReader()
		if reader == nil {
			return walkMultipart(entity.Body, params["boundary"], downloadDir, result)
		}
		defer reader.Close()
		for {
			part, err := reader.NextPart()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return err
			}
			if err := consumeEntity(part, downloadDir, result); err != nil {
				_ = closeEntityBody(part)
				return err
			}
			_ = closeEntityBody(part)
		}
	case strings.EqualFold(strings.ToLower(mediaType), "message/rfc822"):
		childBytes, err := io.ReadAll(entity.Body)
		if err != nil {
			return err
		}
		child, err := parseFetchedMail(RawMail{UID: result.UID, Raw: childBytes}, downloadDir)
		if err != nil {
			return err
		}
		mergeFetchedMail(result, child)
		return nil
	case strings.HasPrefix(strings.ToLower(mediaType), "text/plain"):
		data, err := io.ReadAll(entity.Body)
		if err != nil {
			return err
		}
		result.TextBody = appendContent(result.TextBody, strings.TrimSpace(decodeBodyWithCharset(data, params["charset"])))
		return nil
	case strings.HasPrefix(strings.ToLower(mediaType), "text/html"):
		data, err := io.ReadAll(entity.Body)
		if err != nil {
			return err
		}
		result.HTMLBody = appendContent(result.HTMLBody, strings.TrimSpace(decodeBodyWithCharset(data, params["charset"])))
		return nil
	default:
		data, err := io.ReadAll(entity.Body)
		if err != nil {
			return err
		}
		filename := fileNameFromEntity(entity)
		contentID := strings.Trim(strings.TrimSpace(headerField(entity.Header, "Content-Id")), "<>")
		disposition := strings.ToLower(strings.TrimSpace(headerField(entity.Header, "Content-Disposition")))
		if disposition == "" && !strings.HasPrefix(strings.ToLower(mediaType), "image/") && contentID == "" && filename == "" {
			return nil
		}
		path, kind, key, err := persistArtifact(downloadDir, data, mediaType, filename, contentID)
		if err != nil {
			log.Printf("[email] stage=artifact-skip err=%s media_type=%s filename=%s content_id=%s", formatErrorForLog(err), mediaType, filename, contentID)
			return nil
		}
		if strings.TrimSpace(path) == "" {
			return nil
		}
		result.Artifacts = append(result.Artifacts, path)
		result.Attachments = append(result.Attachments, MailArtifact{Kind: kind, Key: key, Path: path})
		return nil
	}
}

func closeEntityBody(entity *message.Entity) error {
	if entity == nil {
		return nil
	}
	if closer, ok := entity.Body.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func walkMultipart(reader io.Reader, boundary, downloadDir string, result *FetchedMail) error {
	if strings.TrimSpace(boundary) == "" {
		return errors.New("multipart boundary is required")
	}
	mr := multipart.NewReader(reader, boundary)
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := consumeMimePart(part, downloadDir, result); err != nil {
			part.Close()
			return err
		}
		part.Close()
	}
}

func consumeMimePart(part *multipart.Part, downloadDir string, result *FetchedMail) error {
	contentType := part.Header.Get("Content-Type")
	mediaType, params, _ := mime.ParseMediaType(contentType)
	data, err := io.ReadAll(part)
	if err != nil {
		return err
	}
	switch {
	case strings.HasPrefix(strings.ToLower(mediaType), "multipart/"):
		return walkMultipart(bytes.NewReader(data), params["boundary"], downloadDir, result)
	case strings.EqualFold(strings.ToLower(mediaType), "message/rfc822"):
		child, err := parseFetchedMail(RawMail{UID: result.UID, Raw: data}, downloadDir)
		if err != nil {
			return err
		}
		mergeFetchedMail(result, child)
		return nil
	case strings.HasPrefix(strings.ToLower(mediaType), "text/plain"):
		result.TextBody = appendContent(result.TextBody, strings.TrimSpace(decodeBodyWithCharset(data, params["charset"])))
		return nil
	case strings.HasPrefix(strings.ToLower(mediaType), "text/html"):
		result.HTMLBody = appendContent(result.HTMLBody, strings.TrimSpace(decodeBodyWithCharset(data, params["charset"])))
		return nil
	default:
		filename := fileNameFromPart(part)
		contentID := strings.Trim(strings.TrimSpace(part.Header.Get("Content-ID")), "<>")
		disposition, _, _ := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		if disposition == "" && !strings.HasPrefix(strings.ToLower(mediaType), "image/") && contentID == "" && filename == "" {
			return nil
		}
		path, kind, key, err := persistArtifact(downloadDir, data, mediaType, filename, contentID)
		if err != nil {
			log.Printf("[email] stage=artifact-skip err=%s media_type=%s filename=%s content_id=%s", formatErrorForLog(err), mediaType, filename, contentID)
			return nil
		}
		if strings.TrimSpace(path) == "" {
			return nil
		}
		result.Artifacts = append(result.Artifacts, path)
		result.Attachments = append(result.Attachments, MailArtifact{Kind: kind, Key: key, Path: path})
		return nil
	}
}

func mergeFetchedMail(dst *FetchedMail, child FetchedMail) {
	if dst == nil {
		return
	}
	dst.TextBody = appendContent(dst.TextBody, child.TextBody)
	dst.HTMLBody = appendContent(dst.HTMLBody, child.HTMLBody)
	dst.Artifacts = append(dst.Artifacts, child.Artifacts...)
	dst.Attachments = append(dst.Attachments, child.Attachments...)
}

func finalizeFetchedMail(item FetchedMail) FetchedMail {
	item.TextBody = strings.TrimSpace(item.TextBody)
	item.HTMLBody = strings.TrimSpace(item.HTMLBody)
	item.Artifacts = dedupeStrings(item.Artifacts)
	if item.ReceivedAt.IsZero() {
		item.ReceivedAt = time.Now()
	}
	if strings.TrimSpace(item.FromAddress) == "" {
		item.FromAddress = extractMailAddress(item.From)
	}
	return item
}

func appendContent(existing, next string) string {
	existing = strings.TrimSpace(existing)
	next = strings.TrimSpace(next)
	if existing == "" {
		return next
	}
	if next == "" {
		return existing
	}
	return existing + "\n" + next
}

func persistArtifact(downloadDir string, data []byte, mediaType, filename, keyHint string) (string, string, string, error) {
	if len(data) == 0 {
		return "", "", "", nil
	}
	if strings.TrimSpace(downloadDir) == "" {
		downloadDir = "email_artifacts"
	}
	if err := os.MkdirAll(downloadDir, 0o755); err != nil {
		return "", "", "", err
	}
	kind := "file"
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(mediaType)), "image/") {
		kind = "image"
	}
	key := sanitizeFileName(firstNonEmpty(keyHint, filename))
	if key == "" {
		key = md5Text(string(data))
	}
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = extensionByMediaType(mediaType)
	}
	base := kind + "_key_" + key
	target := filepath.Join(downloadDir, sanitizeFileName(base)+ext)
	abs, err := filepath.Abs(target)
	if err == nil {
		target = abs
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return "", "", "", err
	}
	return target, kind, key, nil
}

func fileNameFromPart(part *multipart.Part) string {
	if part == nil {
		return ""
	}
	if name := strings.TrimSpace(part.FileName()); name != "" {
		return sanitizeFileName(name)
	}
	if _, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition")); err == nil {
		if name := strings.TrimSpace(params["filename"]); name != "" {
			return sanitizeFileName(name)
		}
	}
	if _, params, err := mime.ParseMediaType(part.Header.Get("Content-Type")); err == nil {
		if name := strings.TrimSpace(params["name"]); name != "" {
			return sanitizeFileName(name)
		}
	}
	return ""
}

func fileNameFromEntity(entity *message.Entity) string {
	if entity == nil {
		return ""
	}
	if disposition, params, err := entity.Header.ContentDisposition(); err == nil {
		if disposition != "" && strings.TrimSpace(params["filename"]) != "" {
			return sanitizeFileName(params["filename"])
		}
	}
	_, params, _ := entity.Header.ContentType()
	return sanitizeFileName(params["name"])
}

func decodeHeaderValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	decoded, err := (&mime.WordDecoder{CharsetReader: wordDecoderCharsetReader}).DecodeHeader(value)
	if err != nil {
		return toUTF8String([]byte(value))
	}
	return strings.TrimSpace(decoded)
}

func htmlToText(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	reScript := regexp.MustCompile(`(?is)<script.*?>.*?</script>`)
	reStyle := regexp.MustCompile(`(?is)<style.*?>.*?</style>`)
	reTag := regexp.MustCompile(`(?s)<[^>]+>`)
	cleaned := reScript.ReplaceAllString(input, " ")
	cleaned = reStyle.ReplaceAllString(cleaned, " ")
	cleaned = strings.ReplaceAll(cleaned, "<br>", "\n")
	cleaned = strings.ReplaceAll(cleaned, "<br/>", "\n")
	cleaned = strings.ReplaceAll(cleaned, "<br />", "\n")
	cleaned = strings.ReplaceAll(cleaned, "</p>", "\n")
	cleaned = strings.ReplaceAll(cleaned, "</div>", "\n")
	cleaned = reTag.ReplaceAllString(cleaned, " ")
	cleaned = html.UnescapeString(cleaned)
	lines := strings.Split(cleaned, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func extensionByMediaType(mediaType string) string {
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "application/pdf":
		return ".pdf"
	default:
		exts, _ := mime.ExtensionsByType(mediaType)
		if len(exts) > 0 {
			return exts[0]
		}
		return ".bin"
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

func sanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = filepath.Base(name)
	replacer := strings.NewReplacer(
		" ", "_",
		"/", "_",
		"\\", "_",
		":", "_",
		"@", "_",
		"<", "_",
		">", "_",
		",", "_",
	)
	return replacer.Replace(name)
}

func dedupeStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func md5Text(input string) string {
	sum := md5.Sum([]byte(input))
	return hex.EncodeToString(sum[:])
}

func isImagePath(path string) bool {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(path))) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg":
		return true
	default:
		return false
	}
}

func extractMailAddress(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if addr, err := mail.ParseAddress(value); err == nil {
		return strings.ToLower(strings.TrimSpace(addr.Address))
	}
	if list, err := mail.ParseAddressList(value); err == nil && len(list) > 0 {
		return strings.ToLower(strings.TrimSpace(list[0].Address))
	}
	if strings.Contains(value, "<") && strings.Contains(value, ">") {
		parts := strings.SplitN(value, "<", 2)
		value = strings.TrimSuffix(parts[1], ">")
	}
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeMessageID(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func toUTF8String(data []byte) string {
	text := strings.TrimSpace(string(bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})))
	if utf8Valid(text) {
		return text
	}
	return strings.ToValidUTF8(text, "")
}

func decodeBodyWithCharset(data []byte, charsetName string) string {
	if utf8Valid(string(bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}))) {
		return toUTF8String(data)
	}
	charsetName = strings.TrimSpace(charsetName)
	if charsetName == "" {
		return toUTF8String(data)
	}
	reader, err := newCharsetReader(charsetName, bytes.NewReader(data))
	if err != nil {
		return toUTF8String(data)
	}
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return toUTF8String(data)
	}
	return toUTF8String(decoded)
}

func utf8Valid(s string) bool {
	return strings.ToValidUTF8(s, "\uFFFD") == s
}

func newCharsetReader(charsetName string, input io.Reader) (io.Reader, error) {
	enc, err := htmlindex.Get(strings.TrimSpace(charsetName))
	if err != nil {
		return input, nil
	}
	return transform.NewReader(input, enc.NewDecoder()), nil
}

func wordDecoderCharsetReader(charsetName string, input io.Reader) (io.Reader, error) {
	return newCharsetReader(charsetName, input)
}

func mailHeaderToJSON(header mail.Header) []map[string]string {
	keys := make([]string, 0, len(header))
	for key := range header {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]string, 0, len(keys))
	for _, key := range keys {
		for _, value := range header[key] {
			out = append(out, map[string]string{
				"name":  key,
				"value": decodeHeaderValue(value),
			})
		}
	}
	return out
}

func collectEntityHeaders(header message.Header) []map[string]string {
	out := make([]map[string]string, 0)
	fields := header.Fields()
	for fields.Next() {
		out = append(out, map[string]string{
			"name":  fields.Key(),
			"value": decodeHeaderValue(fields.Value()),
		})
	}
	return out
}

func headersToMailHeader(headers []map[string]string) mail.Header {
	out := mail.Header{}
	for _, item := range headers {
		key := strings.TrimSpace(item["name"])
		if key == "" {
			continue
		}
		out[key] = append(out[key], item["value"])
	}
	return out
}

func headerValue(headers []map[string]string, key string) string {
	for _, item := range headers {
		if strings.EqualFold(strings.TrimSpace(item["name"]), strings.TrimSpace(key)) {
			return strings.TrimSpace(item["value"])
		}
	}
	return ""
}

func headerField(header message.Header, key string) string {
	fields := header.FieldsByKey(key)
	if fields.Next() {
		return fields.Value()
	}
	return ""
}
