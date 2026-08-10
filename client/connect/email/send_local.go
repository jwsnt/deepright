package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"connect/connectsvc"
	"connect/emailsvc"
	"github.com/google/uuid"
)

const (
	defaultSMTPPort     = 465
	defaultLogFile      = "email.log"
	defaultRunLog       = "email.runtime.log"
	localMaxSendRetries = 3
)

var messageIDPattern = regexp.MustCompile(`<[^<>\r\n]+>`)
var imgTagPattern = regexp.MustCompile(`(?is)<img\b[^>]*\bsrc\s*=\s*(?:'([^']*)'|"([^"]*)"|([^'"\s>]+))[^>]*>`)
var imgSrcAttrPattern = regexp.MustCompile(`(?is)\bsrc\s*=\s*(?:'[^']*'|"[^"]*"|[^'"\s>]+)`)
var imageReferencePatternLocal = regexp.MustCompile(`(?i)(https?://[^\s"'<>]+|file://[^\s"'<>]+)`)

type localReplyEnvelope struct {
	To         []string
	Subject    string
	MessageID  string
	References []string
}

type localSendInput struct {
	emailsvc.SendInput
	To      string
	Subject string
}

type localMetaLoader interface {
	Load(ctx context.Context) (*connectsvc.Meta, emailsvc.Config, error)
}

type localSMTPTransport interface {
	Send(ctx context.Context, cfg emailsvc.Config, mail localOutgoingMail) (string, error)
}

type localSender struct {
	loader     localMetaLoader
	transport  localSMTPTransport
	logger     *log.Logger
	messageLog io.Writer
	now        func() time.Time
	baseDir    string
}

type localRetryExhaustedError struct {
	step string
	err  error
}

func (e *localRetryExhaustedError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *localRetryExhaustedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

type localSendResult struct {
	Name       string            `json:"name"`
	To         string            `json:"to"`
	MessageID  string            `json:"messageId,omitempty"`
	Sent       []localSentRecord `json:"sent"`
	SMTPHeader string            `json:"-"`
}

type localSentRecord struct {
	Type      string `json:"type"`
	MessageID string `json:"messageId,omitempty"`
	Path      string `json:"path,omitempty"`
}

type localAttachment struct {
	Type        string
	Path        string
	FileName    string
	ContentType string
	Data        []byte
	Disposition string
	ContentID   string
}

type localOutgoingMail struct {
	From        string
	To          []string
	Subject     string
	TextBody    string
	HTMLBody    string
	InReplyTo   string
	References  []string
	MessageID   string
	Date        time.Time
	Attachments []localAttachment
	Raw         []byte
}

type cidEmbeddedImage struct {
	Path      string
	ContentID string
}

type localSMTPTransportImpl struct{}

type emailMetaLoader struct {
	service *emailsvc.Service
}

type localSchemaConnectClient struct {
	inner *emailsvc.CLIConnectClient
}

func (l emailMetaLoader) Load(ctx context.Context) (*connectsvc.Meta, emailsvc.Config, error) {
	if l.service == nil {
		return nil, emailsvc.Config{}, errors.New("email service is required")
	}
	return l.service.Load(ctx)
}

func (c localSchemaConnectClient) GetMeta(ctx context.Context, name string) (*connectsvc.Meta, error) {
	if c.inner == nil {
		return nil, errors.New("connect client is required")
	}
	return c.inner.GetMeta(ctx, name)
}

func (c localSchemaConnectClient) AddRequest(ctx context.Context, input connectsvc.RequestInput) (*connectsvc.Request, error) {
	if c.inner == nil {
		return nil, errors.New("connect client is required")
	}
	input.ResponseSchema = responseSchemaJSONLocal()
	return c.inner.AddRequest(ctx, input)
}

func runSendCommand(args []string, stdout, stderr io.Writer) (*localSendResult, bool, int) {
	if len(args) == 0 {
		return nil, false, 0
	}
	command := strings.TrimSpace(strings.ToLower(args[0]))
	if command != "send" && command != "init" {
		return nil, false, 0
	}

	flags, err := connectsvc.ParseFlags(args[1:])
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return nil, true, 1
	}

	result, err := runLocalSend(command, flags, stderr)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return nil, true, 1
	}
	return result, true, 0
}

func runLocalSend(action string, flags map[string]string, stderr io.Writer) (*localSendResult, error) {
	messageFile, runtimeFile, logger, err := openLocalMessageAndRuntimeLogs(flags, stderr)
	if err != nil {
		return nil, err
	}
	defer messageFile.Close()
	defer runtimeFile.Close()

	service, err := buildLocalService(flags, logger, messageFile)
	if err != nil {
		writeTimestampedLocalMessageLog(messageFile, logger, time.Now().Format(time.RFC3339), formatSendRequestLogLocal(action, localSendInput{
			SendInput: emailsvc.SendInput{
				Action:  action,
				Message: connectsvc.FirstValue(flags, "message"),
				Content: connectsvc.FirstValue(flags, "content"),
				Images:  splitCSVFlag(connectsvc.FirstValue(flags, "image", "images")),
				Files:   splitCSVFlag(connectsvc.FirstValue(flags, "file", "files")),
			},
			To:      connectsvc.FirstValue(flags, "to"),
			Subject: connectsvc.FirstValue(flags, "subject"),
		}))
		writeTimestampedLocalMessageLog(messageFile, logger, time.Now().Format(time.RFC3339), formatSendFailureLogLocal(action, "build-service", nil, err))
		return nil, err
	}
	sender := newLocalSender(emailMetaLoader{service: service}, logger, messageFile)
	sender.baseDir = downloadDirForLogLocal(connectsvc.FirstValue(flags, "log-file"))
	result, err := sender.SendWithOptions(context.Background(), localSendInput{
		SendInput: emailsvc.SendInput{
			Action:  action,
			Message: connectsvc.FirstValue(flags, "message"),
			Content: connectsvc.FirstValue(flags, "content"),
			Images:  splitCSVFlag(connectsvc.FirstValue(flags, "image", "images")),
			Files:   splitCSVFlag(connectsvc.FirstValue(flags, "file", "files")),
		},
		To:      connectsvc.FirstValue(flags, "to"),
		Subject: connectsvc.FirstValue(flags, "subject"),
	})
	if err != nil {
		logger.Printf("stage=%s err=%v", strings.TrimSpace(action), err)
		return nil, err
	}
	return result, nil
}

func newLocalSender(loader localMetaLoader, logger *log.Logger, messageLog io.Writer) *localSender {
	return &localSender{
		loader:     loader,
		logger:     logger,
		messageLog: messageLog,
		now:        time.Now,
		baseDir:    downloadDirForLogLocal(""),
	}
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

func (s *localSender) Send(ctx context.Context, input emailsvc.SendInput) (*localSendResult, error) {
	return s.SendWithOptions(ctx, localSendInput{SendInput: input})
}

func (s *localSender) SendWithOptions(ctx context.Context, input localSendInput) (*localSendResult, error) {
	if s == nil || s.loader == nil {
		return nil, errors.New("sender loader is required")
	}
	action := strings.TrimSpace(strings.ToLower(input.Action))
	if action == "" {
		action = "send"
	}
	at := s.clock().Format(time.RFC3339)
	writeTimestampedLocalMessageLog(s.messageLog, s.logger, at, formatSendRequestLogLocal(action, input))
	reply, err := resolveLocalReplyEnvelope(input.Message, input.To, input.Subject)
	if err != nil {
		writeTimestampedLocalMessageLog(s.messageLog, s.logger, at, formatSendFailureLogLocal(action, "parse-message", nil, err))
		return nil, err
	}
	writeTimestampedLocalMessageLog(s.messageLog, s.logger, at, formatSendParseLogLocal(action, &reply))
	meta, cfg, err := s.loader.Load(ctx)
	if err != nil {
		writeTimestampedLocalMessageLog(s.messageLog, s.logger, at, formatSendFailureLogLocal(action, "load-meta", nil, err))
		return nil, err
	}
	normalized, normalizationErr := normalizeSendInputForDeliveryLocal(ctx, localSendNormalizeInput{
		Action:      action,
		Content:     input.Content,
		Images:      input.Images,
		Files:       input.Files,
		DownloadDir: s.downloadDir(),
		Logger:      s.logger,
		LogWriter:   s.messageLog,
	})

	if normalizationErr != nil {
		s.logf("stage=%s mode=structured-normalize err=%v", action, normalizationErr)
		fallback := normalized.fallbackPayload()
		fallbackResult, fallbackErr := s.deliverLocalMail(ctx, meta, cfg, reply, action, fallback)
		if fallbackErr == nil {
			writeTimestampedLocalMessageLog(s.messageLog, s.logger, at, formatSendResultLogLocal(action, fallbackResult, reply.MessageID))
			return fallbackResult, nil
		}
		writeTimestampedLocalMessageLog(s.messageLog, s.logger, at, formatSendFailureLogLocal(action, "send-fallback", fallbackResult, fallbackErr))
		if isLocalRetryExhaustedError(fallbackErr) {
			return nil, fallbackErr
		}
		s.logf("stage=%s mode=fallback err=%v", action, fallbackErr)
		exception := normalized.exceptionPayload()
		exceptionResult, exceptionErr := s.deliverLocalMail(ctx, meta, cfg, reply, action, exception)
		if exceptionErr == nil {
			writeTimestampedLocalMessageLog(s.messageLog, s.logger, at, formatSendResultLogLocal(action, exceptionResult, reply.MessageID))
			return exceptionResult, nil
		}
		writeTimestampedLocalMessageLog(s.messageLog, s.logger, at, formatSendFailureLogLocal(action, "send-exception", exceptionResult, exceptionErr))
		return nil, fallbackErr
	}

	result, sendErr := s.deliverLocalMail(ctx, meta, cfg, reply, action, normalized)
	if sendErr == nil {
		writeTimestampedLocalMessageLog(s.messageLog, s.logger, at, formatSendResultLogLocal(action, result, reply.MessageID))
		return result, nil
	}
	writeTimestampedLocalMessageLog(s.messageLog, s.logger, at, formatSendFailureLogLocal(action, "send-structured", result, sendErr))
	if isLocalRetryExhaustedError(sendErr) {
		return nil, sendErr
	}

	s.logf("stage=%s mode=structured-fallback err=%v", action, sendErr)
	fallback := normalized.fallbackPayload()
	fallbackResult, fallbackErr := s.deliverLocalMail(ctx, meta, cfg, reply, action, fallback)
	if fallbackErr == nil {
		writeTimestampedLocalMessageLog(s.messageLog, s.logger, at, formatSendResultLogLocal(action, fallbackResult, reply.MessageID))
		return fallbackResult, nil
	}
	writeTimestampedLocalMessageLog(s.messageLog, s.logger, at, formatSendFailureLogLocal(action, "send-fallback", fallbackResult, fallbackErr))
	if isLocalRetryExhaustedError(fallbackErr) {
		return nil, fallbackErr
	}

	s.logf("stage=%s mode=fallback err=%v", action, fallbackErr)
	exception := normalized.exceptionPayload()
	exceptionResult, exceptionErr := s.deliverLocalMail(ctx, meta, cfg, reply, action, exception)
	if exceptionErr == nil {
		writeTimestampedLocalMessageLog(s.messageLog, s.logger, at, formatSendResultLogLocal(action, exceptionResult, reply.MessageID))
		return exceptionResult, nil
	}
	writeTimestampedLocalMessageLog(s.messageLog, s.logger, at, formatSendFailureLogLocal(action, "send-exception", exceptionResult, exceptionErr))
	return nil, fallbackErr
}

func (s *localSender) ensureTransport() localSMTPTransport {
	if s != nil && s.transport != nil {
		return s.transport
	}
	return localSMTPTransportImpl{}
}

func (s *localSender) clock() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}

func (s *localSender) logf(format string, args ...any) {
	if s != nil && s.logger != nil {
		s.logger.Printf(format, args...)
	}
}

func (s *localSender) downloadDir() string {
	if s != nil && strings.TrimSpace(s.baseDir) != "" {
		return strings.TrimSpace(s.baseDir)
	}
	return downloadDirForLogLocal("")
}

func openLocalMessageAndRuntimeLogs(flags map[string]string, stderr io.Writer) (io.WriteCloser, io.WriteCloser, *log.Logger, error) {
	messageLogPath := strings.TrimSpace(connectsvc.FirstValue(flags, "log-file"))
	if messageLogPath == "" {
		messageLogPath = defaultLogFile
	}
	if err := ensureParentDirLocal(messageLogPath); err != nil {
		return nil, nil, nil, err
	}
	messageFile := newRotatingLocalLogWriter(messageLogPath, localLogMaxBytes, localLogMaxBackups)

	runtimeLogPath := runtimeLogPathForLocal(messageLogPath)
	if err := ensureParentDirLocal(runtimeLogPath); err != nil {
		return nil, nil, nil, err
	}
	runtimeFile := newRotatingLocalLogWriter(runtimeLogPath, localLogMaxBytes, localLogMaxBackups)
	logger := log.New(io.MultiWriter(stderr, runtimeFile, messageFile), "[email] ", log.LstdFlags)
	return messageFile, runtimeFile, logger, nil
}

type localSendNormalizeInput struct {
	Action      string
	Content     string
	Images      []string
	Files       []string
	DownloadDir string
	Logger      *log.Logger
	LogWriter   io.Writer
}

type localNormalizedDelivery struct {
	Content         string
	Images          []string
	Files           []string
	OriginalContent string
	OriginalImages  []string
	OriginalFiles   []string
}

func normalizeSendFlagsLocal(command string, flags map[string]string, downloadDir string, logWriter io.Writer) (map[string]string, error) {
	if flags == nil {
		return nil, errors.New("flags are required")
	}
	input := localSendNormalizeInput{
		Action:      command,
		Content:     connectsvc.FirstValue(flags, "content"),
		Images:      splitCSVFlag(connectsvc.FirstValue(flags, "image", "images")),
		Files:       splitCSVFlag(connectsvc.FirstValue(flags, "file", "files")),
		DownloadDir: downloadDir,
		LogWriter:   logWriter,
	}
	normalized, err := normalizeSendInputForDeliveryLocal(context.Background(), input)
	if err != nil {
		normalized = normalized.fallbackPayload()
	}

	next := make(map[string]string, len(flags))
	for key, value := range flags {
		next[key] = value
	}
	next["content"] = normalized.Content
	delete(next, "images")
	delete(next, "files")
	if len(normalized.Images) > 0 {
		next["image"] = strings.Join(normalized.Images, ",")
	} else {
		delete(next, "image")
	}
	if len(normalized.Files) > 0 {
		next["file"] = strings.Join(normalized.Files, ",")
	} else {
		delete(next, "file")
	}
	return next, err
}

func normalizeSendInputForDeliveryLocal(ctx context.Context, input localSendNormalizeInput) (localNormalizedDelivery, error) {
	original := localNormalizedDelivery{
		Content:         strings.TrimSpace(input.Content),
		Images:          uniqueStrings(cleanCSVItems(input.Images)),
		Files:           uniqueStrings(cleanCSVItems(input.Files)),
		OriginalContent: input.Content,
		OriginalImages:  append([]string(nil), cleanCSVItems(input.Images)...),
		OriginalFiles:   append([]string(nil), cleanCSVItems(input.Files)...),
	}

	payload, matchedSchemaLike, err := parseSchemaBackedContent(input.Content)
	if !matchedSchemaLike {
		writeNormalizationLogs(input.LogWriter, normalizedSendPayload{
			Content: original.Content,
			Images:  original.Images,
			Files:   original.Files,
		})
		return original, nil
	}
	if err != nil {
		writeNormalizationErrorLocal(input.LogWriter, input.Action, err)
		return original, err
	}

	normalized := localNormalizedDelivery{
		Content:         strings.TrimSpace(payload.Content),
		Images:          uniqueStrings(cleanCSVItems(input.Images)),
		Files:           uniqueStrings(cleanCSVItems(input.Files)),
		OriginalContent: input.Content,
		OriginalImages:  append([]string(nil), cleanCSVItems(input.Images)...),
		OriginalFiles:   append([]string(nil), cleanCSVItems(input.Files)...),
	}
	normalized.Images = mergeCSVItemsLocal(normalized.Images, payload.Images)
	normalized.Files = mergeCSVItemsLocal(normalized.Files, payload.Files)

	scanned, scanErr := collectReferencedImagesLocal(ctx, normalized.Content, normalized.Images, firstNonEmptyLocal(strings.TrimSpace(input.DownloadDir), downloadDirForLogLocal("")))
	if scanErr != nil {
		writeNormalizationErrorLocal(input.LogWriter, input.Action, scanErr)
		return original, scanErr
	}
	normalized.Images = mergeCSVItemsLocal(normalized.Images, scanned)
	writeNormalizationLogs(input.LogWriter, normalizedSendPayload{
		Content: normalized.Content,
		Images:  normalized.Images,
		Files:   normalized.Files,
	})
	return normalized, nil
}

func (d localNormalizedDelivery) fallbackPayload() localNormalizedDelivery {
	content := extractStructuredContentFallbackLocal(d.OriginalContent)
	if content == "" {
		content = strings.TrimSpace(d.OriginalContent)
	}
	if content == "" {
		content = strings.TrimSpace(d.Content)
	}
	return localNormalizedDelivery{
		Content:         content,
		Images:          nil,
		Files:           nil,
		OriginalContent: d.OriginalContent,
		OriginalImages:  append([]string(nil), d.OriginalImages...),
		OriginalFiles:   append([]string(nil), d.OriginalFiles...),
	}
}

func extractStructuredContentFallbackLocal(raw string) string {
	candidate, ok := schemaJSONCandidateLocal(raw)
	if !ok {
		return ""
	}
	var value any
	if err := json.Unmarshal([]byte(candidate), &value); err != nil {
		return ""
	}
	object, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	content, _ := object["content"].(string)
	return strings.TrimSpace(content)
}

func (d localNormalizedDelivery) exceptionPayload() localNormalizedDelivery {
	return localNormalizedDelivery{
		Content:         "<消息异常>请登录客户端查看",
		Images:          nil,
		Files:           nil,
		OriginalContent: d.OriginalContent,
		OriginalImages:  append([]string(nil), d.OriginalImages...),
		OriginalFiles:   append([]string(nil), d.OriginalFiles...),
	}
}

func (s *localSender) deliverLocalMail(ctx context.Context, meta *connectsvc.Meta, cfg emailsvc.Config, reply localReplyParseContext, action string, payload localNormalizedDelivery) (*localSendResult, error) {
	content := strings.TrimSpace(payload.Content)
	images := uniqueStrings(cleanCSVItems(payload.Images))
	files := uniqueStrings(cleanCSVItems(payload.Files))
	if content == "" && len(images) == 0 && len(files) == 0 {
		return nil, errors.New("content, image or file is required")
	}

	htmlBody, inlineImages, remainingImages, err := transformHTMLImagesToCID(content, images)
	if err != nil {
		return nil, err
	}
	textBody := buildPlainTextBodyLocal(content)

	attachments, records, err := buildLocalAttachments(inlineImages, remainingImages, files)
	if err != nil {
		return nil, err
	}

	outgoing := localOutgoingMail{
		From:        strings.TrimSpace(cfg.Email),
		To:          append([]string(nil), reply.To...),
		Subject:     strings.TrimSpace(reply.Subject),
		TextBody:    textBody,
		HTMLBody:    htmlBody,
		MessageID:   buildOutgoingMessageIDLocal(cfg.Email),
		Date:        s.clock(),
		Attachments: attachments,
	}
	if reply.IsReply {
		outgoing.InReplyTo = strings.TrimSpace(reply.MessageID)
		outgoing.References = append([]string(nil), reply.References...)
	}
	if outgoing.From == "" {
		return nil, errors.New("meta.email is required")
	}
	rawMessage, err := buildSMTPMessageLocal(outgoing)
	if err != nil {
		return nil, err
	}
	outgoing.Raw = rawMessage
	smtpHeader := extractSMTPHeaderBlockLocal(rawMessage)

	messageID, err := s.sendSMTPWithRetry(ctx, action, cfg, outgoing)
	if err != nil {
		return &localSendResult{
			Name:       firstNonEmptyLocal(meta.Name, emailsvc.DefaultName),
			To:         strings.Join(reply.To, ","),
			MessageID:  outgoing.MessageID,
			SMTPHeader: smtpHeader,
		}, err
	}
	if strings.TrimSpace(messageID) == "" {
		messageID = outgoing.MessageID
	}

	sent := make([]localSentRecord, 0, 1+len(records))
	if content != "" {
		sent = append(sent, localSentRecord{
			Type:      "text",
			MessageID: messageID,
		})
	}
	for _, record := range records {
		record.MessageID = messageID
		sent = append(sent, record)
	}

	result := &localSendResult{
		Name:       firstNonEmptyLocal(meta.Name, emailsvc.DefaultName),
		To:         strings.Join(reply.To, ","),
		MessageID:  messageID,
		Sent:       sent,
		SMTPHeader: smtpHeader,
	}
	return result, nil
}

func (s *localSender) sendSMTPWithRetry(ctx context.Context, action string, cfg emailsvc.Config, outgoing localOutgoingMail) (string, error) {
	transport := s.ensureTransport()
	var lastErr error
	for attempt := 1; attempt <= localMaxSendRetries; attempt++ {
		messageID, err := transport.Send(ctx, cfg, outgoing)
		if err == nil {
			return messageID, nil
		}
		lastErr = err
		s.writeRetryLog(action, "send-smtp", attempt, localMaxSendRetries, err)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			break
		}
	}
	s.writeTerminateLog(action, "send-smtp", localMaxSendRetries, lastErr)
	return "", &localRetryExhaustedError{step: "send-smtp", err: lastErr}
}

func (s *localSender) writeRetryLog(action, step string, attempt, max int, err error) {
	if s == nil || err == nil {
		return
	}
	writeTimestampedLocalMessageLog(
		s.messageLog,
		s.logger,
		s.clock().Format(time.RFC3339),
		formatSendRetryLogLocal(action, step, attempt, max, err),
	)
}

func (s *localSender) writeTerminateLog(action, step string, max int, err error) {
	if s == nil || err == nil {
		return
	}
	writeTimestampedLocalMessageLog(
		s.messageLog,
		s.logger,
		s.clock().Format(time.RFC3339),
		formatSendTerminateLogLocal(action, step, max, err),
	)
}

func isLocalRetryExhaustedError(err error) bool {
	var target *localRetryExhaustedError
	return errors.As(err, &target)
}

func mergeCSVItemsLocal(left, right []string) []string {
	return uniqueStrings(append(append([]string{}, left...), right...))
}

func writeNormalizationErrorLocal(w io.Writer, action string, err error) {
	if w == nil || err == nil {
		return
	}
	reason := classifyNormalizationErrorLocal(err)
	fmt.Fprintf(
		w,
		"[email] schema-send action=%s reason=%s error=%s fallback=%q\n",
		strconv.Quote(strings.TrimSpace(action)),
		strconv.Quote(reason),
		strconv.Quote(err.Error()),
		"raw-content",
	)
}

func classifyNormalizationErrorLocal(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(err.Error())
	switch {
	case strings.Contains(message, "content json parse failed"):
		return "json-parse-failed"
	case strings.Contains(message, "content schema mismatch"):
		return "schema-mismatch"
	case strings.Contains(message, "content must be a json object"):
		return "json-object-required"
	case strings.Contains(message, "content.content must be a string"):
		return "content-invalid"
	case strings.Contains(message, "content.artifacts item must be an object"):
		return "artifact-invalid"
	case strings.Contains(message, "content.artifacts.path must be an absolute path"):
		return "artifact-path-not-absolute"
	case strings.Contains(message, "content.artifacts.path not found"):
		return "artifact-path-not-found"
	case strings.Contains(message, "content.artifacts.path stat failed"):
		return "artifact-path-stat-failed"
	case strings.Contains(message, "content.artifacts.path must be a file"):
		return "artifact-path-is-directory"
	case strings.Contains(message, "download image status="):
		return "content-image-download-status"
	case strings.Contains(message, "downloaded resource is not an image"):
		return "content-image-not-image"
	case strings.Contains(message, "unsupported image source"):
		return "content-image-unsupported-source"
	case strings.Contains(message, "image source is a directory"):
		return "content-image-is-directory"
	case strings.Contains(message, "image source is not an image"):
		return "content-image-not-image"
	case strings.Contains(message, "empty file path"):
		return "content-image-empty-file-path"
	default:
		return "normalize-failed"
	}
}

func collectReferencedImagesLocal(ctx context.Context, content string, existingImages []string, downloadDir string) ([]string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, nil
	}
	downloadDir = strings.TrimSpace(downloadDir)
	if downloadDir == "" {
		downloadDir = downloadDirForLogLocal("")
	}
	if err := os.MkdirAll(downloadDir, 0o755); err != nil {
		return nil, err
	}

	candidates := make([]string, 0)
	for _, match := range imageReferencePatternLocal.FindAllString(content, -1) {
		match = strings.TrimSpace(html.UnescapeString(match))
		if match != "" {
			candidates = append(candidates, match)
		}
	}
	if looksLikeHTMLLocal(content) {
		matches := imgTagPattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			for _, candidate := range match[1:] {
				candidate = strings.TrimSpace(html.UnescapeString(candidate))
				if candidate != "" {
					candidates = append(candidates, candidate)
					break
				}
			}
		}
	}

	found := make([]string, 0)
	var firstErr error
	for _, src := range uniqueStrings(candidates) {
		if src == "" || strings.HasPrefix(strings.ToLower(src), "cid:") {
			continue
		}
		if resolved := resolveExistingImageSourceLocal(src, existingImages); resolved != "" {
			found = append(found, resolved)
			continue
		}
		path, err := downloadImageReferenceLocal(ctx, src, downloadDir)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		found = append(found, path)
	}
	return uniqueStrings(found), firstErr
}

func resolveExistingImageSourceLocal(src string, imageFlags []string) string {
	src = strings.TrimSpace(src)
	switch {
	case src == "":
		return ""
	case strings.HasPrefix(strings.ToLower(src), "http://"), strings.HasPrefix(strings.ToLower(src), "https://"), strings.HasPrefix(strings.ToLower(src), "data:"):
		return ""
	case strings.HasPrefix(strings.ToLower(src), "file://"):
		u, err := url.Parse(src)
		if err != nil {
			return ""
		}
		src = strings.TrimSpace(u.Path)
	}
	if filepath.IsAbs(src) {
		clean := filepath.Clean(src)
		if detectArtifactKind(clean) == "image" {
			return clean
		}
	}
	baseName := filepath.Base(src)
	for _, candidate := range imageFlags {
		clean := strings.TrimSpace(candidate)
		if clean == "" || !filepath.IsAbs(clean) {
			continue
		}
		clean = filepath.Clean(clean)
		if baseName != "" && filepath.Base(clean) != baseName && clean != src {
			continue
		}
		if detectArtifactKind(clean) == "image" {
			return clean
		}
	}
	return ""
}

func downloadImageReferenceLocal(ctx context.Context, src, downloadDir string) (string, error) {
	lower := strings.ToLower(strings.TrimSpace(src))
	switch {
	case strings.HasPrefix(lower, "http://"), strings.HasPrefix(lower, "https://"):
		return downloadRemoteImageLocal(ctx, src, downloadDir)
	case strings.HasPrefix(lower, "file://"):
		u, err := url.Parse(src)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(u.Path) == "" {
			return "", fmt.Errorf("empty file path: %s", src)
		}
		return normalizeDownloadedLocalFilePath(u.Path)
	default:
		if filepath.IsAbs(src) {
			return normalizeDownloadedLocalFilePath(src)
		}
	}
	return "", fmt.Errorf("unsupported image source: %s", src)
}

func normalizeDownloadedLocalFilePath(src string) (string, error) {
	src = filepath.Clean(strings.TrimSpace(src))
	info, err := os.Stat(src)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("image source is a directory: %s", src)
	}
	if detectArtifactKind(src) != "image" {
		return "", fmt.Errorf("image source is not an image: %s", src)
	}
	return src, nil
}

func downloadRemoteImageLocal(ctx context.Context, src, downloadDir string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download image status=%d source=%s", resp.StatusCode, src)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return "", err
	}
	contentType := http.DetectContentType(body)
	if !strings.HasPrefix(contentType, "image/") {
		return "", fmt.Errorf("downloaded resource is not an image: %s", src)
	}
	ext := imageExtensionForContentTypeLocal(contentType)
	if ext == "" {
		if u, parseErr := url.Parse(src); parseErr == nil {
			ext = filepath.Ext(u.Path)
		}
	}
	if ext == "" {
		ext = ".img"
	}
	fileName := "image_key_" + uuid.NewString() + ext
	path := filepath.Join(downloadDir, fileName)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func imageExtensionForContentTypeLocal(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "image/bmp":
		return ".bmp"
	default:
		return ""
	}
}

func transformHTMLImagesToCID(content string, imageFlags []string) (string, []cidEmbeddedImage, []string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", nil, uniqueStrings(imageFlags), nil
	}
	if !looksLikeHTMLLocal(content) {
		return buildHTMLBodyLocal(content), nil, uniqueStrings(imageFlags), nil
	}

	remaining := uniqueStrings(imageFlags)
	inlineImages := make([]cidEmbeddedImage, 0)
	htmlBody := imgTagPattern.ReplaceAllStringFunc(content, func(tag string) string {
		matches := imgTagPattern.FindStringSubmatch(tag)
		if len(matches) < 4 {
			return tag
		}
		src := ""
		for _, candidate := range matches[1:] {
			if strings.TrimSpace(candidate) != "" {
				src = candidate
				break
			}
		}
		src = strings.TrimSpace(html.UnescapeString(src))
		if src == "" || strings.HasPrefix(strings.ToLower(src), "cid:") {
			return tag
		}

		path, ok := resolveInlineImageSource(src, remaining)
		if !ok {
			return tag
		}
		contentID := "img-" + uuid.NewString()
		inlineImages = append(inlineImages, cidEmbeddedImage{
			Path:      path,
			ContentID: contentID,
		})
		remaining = removeFirstString(remaining, path)
		return replaceImageSrcWithCID(tag, contentID)
	})
	return htmlBody, dedupeInlineImages(inlineImages), uniqueStrings(remaining), nil
}

func resolveInlineImageSource(src string, imageFlags []string) (string, bool) {
	switch {
	case strings.HasPrefix(strings.ToLower(src), "http://"), strings.HasPrefix(strings.ToLower(src), "https://"), strings.HasPrefix(strings.ToLower(src), "data:"):
		return "", false
	}

	if filepath.IsAbs(src) {
		clean := filepath.Clean(src)
		if detectArtifactKind(clean) == "image" {
			return clean, true
		}
	}

	candidates := make([]string, 0, len(imageFlags)+1)
	candidates = append(candidates, imageFlags...)
	if src != "" {
		candidates = append(candidates, src)
	}

	baseName := filepath.Base(src)
	for _, candidate := range candidates {
		clean := strings.TrimSpace(candidate)
		if clean == "" {
			continue
		}
		if !filepath.IsAbs(clean) {
			continue
		}
		clean = filepath.Clean(clean)
		if baseName != "" && filepath.Base(clean) != baseName && clean != src {
			continue
		}
		if detectArtifactKind(clean) == "image" {
			return clean, true
		}
	}
	return "", false
}

func replaceImageSrcWithCID(tag, contentID string) string {
	return imgSrcAttrPattern.ReplaceAllString(tag, `src="cid:`+contentID+`"`)
}

func dedupeInlineImages(images []cidEmbeddedImage) []cidEmbeddedImage {
	if len(images) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(images))
	out := make([]cidEmbeddedImage, 0, len(images))
	for _, item := range images {
		key := strings.TrimSpace(item.Path) + "|" + strings.TrimSpace(item.ContentID)
		if strings.TrimSpace(item.Path) == "" || strings.TrimSpace(item.ContentID) == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func buildLocalAttachments(inlineImages []cidEmbeddedImage, images, files []string) ([]localAttachment, []localSentRecord, error) {
	attachments := make([]localAttachment, 0, len(inlineImages)+len(images)+len(files))
	records := make([]localSentRecord, 0, len(inlineImages)+len(images)+len(files))
	for _, inlineImage := range inlineImages {
		attachment, err := readLocalAttachment("image", inlineImage.Path)
		if err != nil {
			return nil, nil, err
		}
		attachment.Disposition = "inline"
		attachment.ContentID = inlineImage.ContentID
		attachments = append(attachments, attachment)
		records = append(records, localSentRecord{Type: "image", Path: attachment.Path})
	}
	for _, imagePath := range images {
		attachment, err := readLocalAttachment("image", imagePath)
		if err != nil {
			return nil, nil, err
		}
		attachment.Disposition = "attachment"
		attachments = append(attachments, attachment)
		records = append(records, localSentRecord{Type: "image", Path: attachment.Path})
	}
	for _, filePath := range files {
		attachment, err := readLocalAttachment("file", filePath)
		if err != nil {
			return nil, nil, err
		}
		attachment.Disposition = "attachment"
		attachments = append(attachments, attachment)
		records = append(records, localSentRecord{Type: "file", Path: attachment.Path})
	}
	return attachments, records, nil
}

func readLocalAttachment(kind, rawPath string) (localAttachment, error) {
	path := strings.TrimSpace(rawPath)
	if path == "" {
		return localAttachment{}, errors.New("attachment path is required")
	}
	absPath, err := filepath.Abs(path)
	if err == nil {
		path = absPath
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return localAttachment{}, err
	}
	return localAttachment{
		Type:        kind,
		Path:        path,
		FileName:    filepath.Base(path),
		ContentType: detectAttachmentContentTypeLocal(path, data),
		Data:        data,
	}, nil
}

func (localSMTPTransportImpl) Send(ctx context.Context, cfg emailsvc.Config, outgoing localOutgoingMail) (string, error) {
	host, port, err := normalizeSMTPAddressLocal(cfg.EmailSMTP)
	if err != nil {
		return "", err
	}
	raw := outgoing.Raw
	if len(raw) == 0 {
		raw, err = buildSMTPMessageLocal(outgoing)
		if err != nil {
			return "", err
		}
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	client, conn, err := dialSMTPContextLocal(ctx, host, addr, port)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	defer client.Close()

	if port != defaultSMTPPort {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
				return "", err
			}
		}
	}

	if strings.TrimSpace(cfg.EmailPassword) != "" {
		if ok, _ := client.Extension("AUTH"); !ok {
			return "", errors.New("smtp server does not advertise AUTH")
		}
		auth := smtp.PlainAuth("", strings.TrimSpace(cfg.Email), strings.TrimSpace(cfg.EmailPassword), host)
		if err := client.Auth(auth); err != nil {
			return "", err
		}
	}

	if err := client.Mail(strings.TrimSpace(outgoing.From)); err != nil {
		return "", err
	}
	for _, rcpt := range outgoing.To {
		if err := client.Rcpt(strings.TrimSpace(rcpt)); err != nil {
			return "", err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return "", err
	}
	if _, err := writer.Write(raw); err != nil {
		writer.Close()
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	if err := client.Quit(); err != nil {
		return "", err
	}
	return outgoing.MessageID, nil
}

func buildSMTPMessageLocal(outgoing localOutgoingMail) ([]byte, error) {
	if strings.TrimSpace(outgoing.From) == "" {
		return nil, errors.New("from is required")
	}
	if len(outgoing.To) == 0 {
		return nil, errors.New("to is required")
	}
	if strings.TrimSpace(outgoing.MessageID) == "" {
		outgoing.MessageID = buildOutgoingMessageIDLocal(outgoing.From)
	}
	if outgoing.Date.IsZero() {
		outgoing.Date = time.Now()
	}
	if strings.TrimSpace(outgoing.TextBody) == "" && strings.TrimSpace(outgoing.HTMLBody) == "" && len(outgoing.Attachments) == 0 {
		return nil, errors.New("mail body is required")
	}

	var body bytes.Buffer
	contentType, transferEncoding, err := writeMessageBodyLocal(&body, outgoing)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	writeHeaderLineLocal(&buf, "From", strings.TrimSpace(outgoing.From))
	writeHeaderLineLocal(&buf, "To", strings.Join(cleanCSVItems(outgoing.To), ", "))
	if subject := strings.TrimSpace(outgoing.Subject); subject != "" {
		writeHeaderLineLocal(&buf, "Subject", encodeMIMEHeaderLocal(subject))
	}
	writeHeaderLineLocal(&buf, "Date", outgoing.Date.Format(time.RFC1123Z))
	writeHeaderLineLocal(&buf, "Message-ID", strings.TrimSpace(outgoing.MessageID))
	if value := strings.TrimSpace(outgoing.InReplyTo); value != "" {
		writeHeaderLineLocal(&buf, "In-Reply-To", value)
	}
	if len(outgoing.References) > 0 {
		writeHeaderLineLocal(&buf, "References", strings.Join(outgoing.References, " "))
	}
	writeHeaderLineLocal(&buf, "MIME-Version", "1.0")
	writeHeaderLineLocal(&buf, "Content-Type", contentType)
	if strings.TrimSpace(transferEncoding) != "" {
		writeHeaderLineLocal(&buf, "Content-Transfer-Encoding", transferEncoding)
	}
	buf.WriteString("\r\n")
	buf.Write(body.Bytes())
	return buf.Bytes(), nil
}

func writeMessageBodyLocal(w io.Writer, outgoing localOutgoingMail) (string, string, error) {
	hasText := strings.TrimSpace(outgoing.TextBody) != ""
	hasHTML := strings.TrimSpace(outgoing.HTMLBody) != ""
	inlineAttachments, regularAttachments := splitInlineAttachments(outgoing.Attachments)
	if len(inlineAttachments) == 0 && len(regularAttachments) == 0 {
		if hasText && hasHTML {
			writer := multipart.NewWriter(w)
			if err := writeAlternativePartsLocal(writer, outgoing.TextBody, outgoing.HTMLBody); err != nil {
				return "", "", err
			}
			if err := writer.Close(); err != nil {
				return "", "", err
			}
			return fmt.Sprintf(`multipart/alternative; boundary="%s"`, writer.Boundary()), "", nil
		}
		if hasHTML {
			if err := writeSingleTextBodyLocal(w, outgoing.HTMLBody); err != nil {
				return "", "", err
			}
			return `text/html; charset=UTF-8`, "quoted-printable", nil
		}
		if err := writeSingleTextBodyLocal(w, outgoing.TextBody); err != nil {
			return "", "", err
		}
		return `text/plain; charset=UTF-8`, "quoted-printable", nil
	}

	writer := multipart.NewWriter(w)
	if err := writeBodyPartsLocal(writer, outgoing.TextBody, outgoing.HTMLBody, inlineAttachments); err != nil {
		return "", "", err
	}
	for _, attachment := range regularAttachments {
		if err := writeAttachmentPartLocal(writer, attachment); err != nil {
			return "", "", err
		}
	}
	if err := writer.Close(); err != nil {
		return "", "", err
	}
	return fmt.Sprintf(`multipart/mixed; boundary="%s"`, writer.Boundary()), "", nil
}

func writeBodyPartsLocal(writer *multipart.Writer, textBody, htmlBody string, inlineAttachments []localAttachment) error {
	hasText := strings.TrimSpace(textBody) != ""
	hasHTML := strings.TrimSpace(htmlBody) != ""
	if len(inlineAttachments) == 0 {
		if hasText && hasHTML {
			altBoundary := "alt-" + uuid.NewString()
			header := textproto.MIMEHeader{}
			header.Set("Content-Type", fmt.Sprintf(`multipart/alternative; boundary="%s"`, altBoundary))
			part, err := writer.CreatePart(header)
			if err != nil {
				return err
			}
			altWriter := multipart.NewWriter(part)
			if err := altWriter.SetBoundary(altBoundary); err != nil {
				return err
			}
			if err := writeAlternativePartsLocal(altWriter, textBody, htmlBody); err != nil {
				return err
			}
			return altWriter.Close()
		}
		if hasHTML {
			return writeMultipartTextPartLocal(writer, "text/html", htmlBody)
		}
		if hasText {
			return writeMultipartTextPartLocal(writer, "text/plain", textBody)
		}
		return nil
	}

	relatedBoundary := "rel-" + uuid.NewString()
	header := textproto.MIMEHeader{}
	header.Set("Content-Type", fmt.Sprintf(`multipart/related; boundary="%s"`, relatedBoundary))
	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}
	relatedWriter := multipart.NewWriter(part)
	if err := relatedWriter.SetBoundary(relatedBoundary); err != nil {
		return err
	}
	if hasText && hasHTML {
		altBoundary := "alt-" + uuid.NewString()
		altHeader := textproto.MIMEHeader{}
		altHeader.Set("Content-Type", fmt.Sprintf(`multipart/alternative; boundary="%s"`, altBoundary))
		altPart, err := relatedWriter.CreatePart(altHeader)
		if err != nil {
			return err
		}
		altWriter := multipart.NewWriter(altPart)
		if err := altWriter.SetBoundary(altBoundary); err != nil {
			return err
		}
		if err := writeAlternativePartsLocal(altWriter, textBody, htmlBody); err != nil {
			return err
		}
		if err := altWriter.Close(); err != nil {
			return err
		}
	} else if hasHTML {
		if err := writeMultipartTextPartLocal(relatedWriter, "text/html", htmlBody); err != nil {
			return err
		}
	} else if hasText {
		if err := writeMultipartTextPartLocal(relatedWriter, "text/plain", textBody); err != nil {
			return err
		}
	}
	for _, attachment := range inlineAttachments {
		if err := writeAttachmentPartLocal(relatedWriter, attachment); err != nil {
			return err
		}
	}
	return relatedWriter.Close()
}

func splitInlineAttachments(attachments []localAttachment) ([]localAttachment, []localAttachment) {
	inlineItems := make([]localAttachment, 0, len(attachments))
	regularItems := make([]localAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		if strings.EqualFold(strings.TrimSpace(attachment.Disposition), "inline") && strings.TrimSpace(attachment.ContentID) != "" {
			inlineItems = append(inlineItems, attachment)
			continue
		}
		regularItems = append(regularItems, attachment)
	}
	return inlineItems, regularItems
}

func writeAlternativePartsLocal(writer *multipart.Writer, textBody, htmlBody string) error {
	if strings.TrimSpace(textBody) != "" {
		if err := writeMultipartTextPartLocal(writer, "text/plain", textBody); err != nil {
			return err
		}
	}
	if strings.TrimSpace(htmlBody) != "" {
		if err := writeMultipartTextPartLocal(writer, "text/html", htmlBody); err != nil {
			return err
		}
	}
	return nil
}

func writeSingleTextBodyLocal(w io.Writer, body string) error {
	qp := quotedprintable.NewWriter(w)
	_, err := io.WriteString(qp, body)
	if closeErr := qp.Close(); err == nil {
		err = closeErr
	}
	return err
}

func writeMultipartTextPartLocal(writer *multipart.Writer, mediaType, body string) error {
	header := textproto.MIMEHeader{}
	header.Set("Content-Type", mediaType+`; charset=UTF-8`)
	header.Set("Content-Transfer-Encoding", "quoted-printable")
	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}
	return writeSingleTextBodyLocal(part, body)
}

func writeAttachmentPartLocal(writer *multipart.Writer, attachment localAttachment) error {
	contentType := strings.TrimSpace(attachment.ContentType)
	if contentType == "" {
		contentType = detectAttachmentContentTypeLocal(attachment.Path, attachment.Data)
	}
	fileName := strings.TrimSpace(attachment.FileName)
	if fileName == "" {
		fileName = filepath.Base(strings.TrimSpace(attachment.Path))
	}
	if fileName == "" {
		fileName = attachment.Type + ".bin"
	}
	disposition := strings.TrimSpace(attachment.Disposition)
	if disposition == "" {
		disposition = "attachment"
	}

	header := textproto.MIMEHeader{}
	header.Set("Content-Type", contentType+`; name="`+escapeHeaderParamLocal(fileName)+`"`)
	header.Set("Content-Transfer-Encoding", "base64")
	header.Set("Content-Disposition", disposition+`; filename="`+escapeHeaderParamLocal(fileName)+`"`)
	if strings.TrimSpace(attachment.ContentID) != "" {
		header.Set("Content-ID", "<"+strings.TrimSpace(attachment.ContentID)+">")
	}
	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}
	encoder := base64.NewEncoder(base64.StdEncoding, newBase64LineWriterLocal(part))
	if _, err := encoder.Write(attachment.Data); err != nil {
		encoder.Close()
		return err
	}
	return encoder.Close()
}

func dialSMTPContextLocal(ctx context.Context, host, addr string, port int) (*smtp.Client, net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	if port == defaultSMTPPort {
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
			ServerName: host,
			MinVersion: tls.VersionTLS12,
		})
		if err != nil {
			return nil, nil, err
		}
		client, err := smtp.NewClient(conn, host)
		if err != nil {
			conn.Close()
			return nil, nil, err
		}
		return client, conn, nil
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, nil, err
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	return client, conn, nil
}

func normalizeSMTPAddressLocal(value string) (string, int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", 0, errors.New("smtp server is required")
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
	return value, defaultSMTPPort, nil
}

type localReplyParseContext struct {
	localReplyEnvelope
	EnvelopeRaw string
	IsReply     bool
}

func resolveLocalReplyEnvelope(raw, to, subject string) (localReplyParseContext, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return newLocalOutgoingEnvelope(to, subject)
	}

	var requestPayload any
	if err := json.Unmarshal([]byte(raw), &requestPayload); err != nil {
		return localReplyParseContext{}, errors.New("message must be valid json")
	}
	request, _ := requestPayload.(map[string]any)
	_, hasRawRequest := localRawRequestValue(request)
	if !hasRawRequest {
		return newLocalOutgoingEnvelope(to, subject)
	}
	return extractLocalReplyEnvelope(raw)
}

func newLocalOutgoingEnvelope(to, subject string) (localReplyParseContext, error) {
	recipients := dedupeStringsLocal(parseAddressCandidatesLocal(to))
	if len(recipients) == 0 {
		return localReplyParseContext{}, errors.New("--to is required when message.rawRequest is not provided")
	}
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return localReplyParseContext{}, errors.New("--subject is required when message.rawRequest is not provided")
	}
	return localReplyParseContext{
		localReplyEnvelope: localReplyEnvelope{
			To:      recipients,
			Subject: subject,
		},
	}, nil
}

func extractLocalReplyEnvelope(raw string) (localReplyParseContext, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return localReplyParseContext{}, errors.New("message is required")
	}

	var requestPayload any
	if err := json.Unmarshal([]byte(raw), &requestPayload); err != nil {
		return localReplyParseContext{}, errors.New("message must be valid json")
	}
	request, _ := requestPayload.(map[string]any)
	envelopeRaw, hasRawRequest := localRawRequestValue(request)
	if !hasRawRequest {
		return localReplyParseContext{}, errors.New("message.rawRequest not found")
	}
	if strings.TrimSpace(envelopeRaw) == "" {
		return localReplyParseContext{}, errors.New("message.rawRequest must be a non-empty json string")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(envelopeRaw), &payload); err != nil {
		return localReplyParseContext{}, errors.New("message.rawRequest must be valid json")
	}

	message, _ := payload["message"].(map[string]any)
	messageID := normalizeReplyMessageIDLocal(stringPathLocal(message, "messageId"))
	if messageID == "" {
		return localReplyParseContext{}, errors.New("message.rawRequest.message.messageId not found")
	}
	from := strings.TrimSpace(stringPathLocal(message, "from"))
	if from == "" {
		return localReplyParseContext{}, errors.New("message.rawRequest.message.from not found")
	}
	subject := strings.TrimSpace(stringPathLocal(message, "subject"))
	if subject == "" {
		return localReplyParseContext{}, errors.New("message.rawRequest.message.subject not found")
	}

	envelope := localReplyEnvelope{
		To:        dedupeStringsLocal(parseAddressCandidatesLocal(from)),
		Subject:   replySubjectLocal(subject),
		MessageID: messageID,
	}
	if len(envelope.To) == 0 {
		return localReplyParseContext{}, errors.New("message.rawRequest.message.from not found")
	}
	collectLocalReplyEnvelope(&envelope, payload)
	envelope.To = dedupeStringsLocal(cleanCSVItems(envelope.To))
	envelope.References = appendReferenceIDsLocal(normalizeReplyMessageIDsLocal(envelope.References), envelope.MessageID)
	return localReplyParseContext{
		localReplyEnvelope: envelope,
		EnvelopeRaw:        envelopeRaw,
		IsReply:            true,
	}, nil
}

func localRawRequestValue(request map[string]any) (string, bool) {
	if request == nil {
		return "", false
	}
	if value, ok := request["rawRequest"]; ok {
		raw, ok := value.(string)
		if !ok {
			return "", true
		}
		return strings.TrimSpace(raw), true
	}
	if value, ok := request["raw_request"]; ok {
		raw, ok := value.(string)
		if !ok {
			return "", true
		}
		return strings.TrimSpace(raw), true
	}
	return "", false
}

func collectLocalReplyEnvelope(envelope *localReplyEnvelope, payload map[string]any) {
	if envelope == nil || payload == nil {
		return
	}

	if strings.TrimSpace(envelope.MessageID) == "" {
		envelope.MessageID = normalizeReplyMessageIDLocal(stringPathLocal(payload, "message", "messageId"))
	}
	if strings.TrimSpace(envelope.Subject) == "" {
		envelope.Subject = stringPathLocal(payload, "message", "subject")
	}
	envelope.To = append(envelope.To, parseAddressCandidatesLocal(
		stringPathLocal(payload, "message", "from"),
	)...)

	if messageValue, ok := payload["message"]; ok {
		if messageMap, ok := messageValue.(map[string]any); ok {
			if raw := strings.TrimSpace(stringPathLocal(messageMap, "raw")); raw != "" {
				var rawPayload map[string]any
				if err := json.Unmarshal([]byte(raw), &rawPayload); err == nil {
					if headers, ok := rawPayload["headers"]; ok {
						mergeReplyEnvelopeHeadersLocal(envelope, headers)
					}
				}
			}
		}
	}
}

func mergeReplyEnvelopeHeadersLocal(envelope *localReplyEnvelope, raw any) {
	headers, ok := raw.([]any)
	if !ok {
		return
	}
	for _, item := range headers {
		header, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := strings.TrimSpace(fmt.Sprint(header["name"]))
		value := strings.TrimSpace(fmt.Sprint(header["value"]))
		if name == "" || value == "" {
			continue
		}
		switch {
		case strings.EqualFold(name, "Message-ID"), strings.EqualFold(name, "Message-Id"):
			if strings.TrimSpace(envelope.MessageID) == "" {
				envelope.MessageID = normalizeReplyMessageIDLocal(value)
			}
		case strings.EqualFold(name, "Subject"):
			if strings.TrimSpace(envelope.Subject) == "" {
				envelope.Subject = decodeHeaderValueLocal(value)
			}
		case strings.EqualFold(name, "References"):
			envelope.References = append(envelope.References, extractReplyMessageIDsLocal(value)...)
		case strings.EqualFold(name, "In-Reply-To"):
			envelope.References = append(envelope.References, extractReplyMessageIDsLocal(value)...)
		}
	}
}

func parseAddressCandidatesLocal(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if list, err := mail.ParseAddressList(value); err == nil {
			for _, addr := range list {
				address := strings.ToLower(strings.TrimSpace(addr.Address))
				if address != "" {
					out = append(out, address)
				}
			}
			continue
		}
		address := extractMailAddressLocal(value)
		if address != "" {
			out = append(out, address)
		}
	}
	return out
}

func extractMailAddressLocal(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.Contains(value, "<") && strings.Contains(value, ">") {
		start := strings.LastIndex(value, "<")
		end := strings.Index(value[start:], ">")
		if start >= 0 && end > 0 {
			candidate := strings.ToLower(strings.TrimSpace(value[start+1 : start+end]))
			if strings.Contains(candidate, "@") {
				return candidate
			}
		}
	}
	for _, field := range strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	}) {
		field = strings.Trim(field, "<>()[]{}\"'")
		field = strings.ToLower(strings.TrimSpace(field))
		if strings.Contains(field, "@") {
			return field
		}
	}
	return ""
}

func normalizeReplyMessageIDLocal(value string) string {
	ids := extractReplyMessageIDsLocal(value)
	if len(ids) > 0 {
		return ids[0]
	}
	return strings.TrimSpace(value)
}

func normalizeReplyMessageIDsLocal(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, extractReplyMessageIDsLocal(value)...)
	}
	return dedupeStringsLocal(out)
}

func extractReplyMessageIDsLocal(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	matches := messageIDPattern.FindAllString(value, -1)
	if len(matches) > 0 {
		return dedupeStringsLocal(matches)
	}
	return dedupeStringsLocal(strings.Fields(value))
}

func appendReferenceIDsLocal(existing []string, messageID string) []string {
	out := make([]string, 0, len(existing)+1)
	seen := make(map[string]struct{}, len(existing)+1)
	for _, value := range existing {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	messageID = strings.TrimSpace(messageID)
	if messageID != "" {
		if _, ok := seen[messageID]; !ok {
			out = append(out, messageID)
		}
	}
	return out
}

func replySubjectLocal(subject string) string {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "Re:"
	}
	lower := strings.ToLower(subject)
	if strings.HasPrefix(lower, "re:") {
		return subject
	}
	return "Re: " + subject
}

func buildPlainTextBodyLocal(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	if looksLikeHTMLLocal(content) {
		text := htmlToTextLocal(content)
		if text != "" {
			return text
		}
	}
	return content
}

func buildHTMLBodyLocal(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	if looksLikeHTMLLocal(content) {
		return content
	}
	lines := strings.Split(content, "\n")
	escaped := make([]string, 0, len(lines))
	for _, line := range lines {
		escaped = append(escaped, html.EscapeString(line))
	}
	return "<div>" + strings.Join(escaped, "<br/>") + "</div>"
}

func looksLikeHTMLLocal(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	return strings.Contains(content, "<") && strings.Contains(content, ">")
}

func htmlToTextLocal(input string) string {
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

func buildOutgoingMessageIDLocal(email string) string {
	domain := "localhost"
	if at := strings.LastIndex(strings.TrimSpace(email), "@"); at >= 0 && at+1 < len(strings.TrimSpace(email)) {
		domain = strings.TrimSpace(email[at+1:])
	}
	return "<" + uuid.NewString() + "@" + domain + ">"
}

func formatSendRequestLogLocal(action string, input localSendInput) string {
	action = strings.TrimSpace(strings.ToLower(action))
	if action == "" {
		action = "send"
	}
	parts := []string{
		"stage=send-request",
		"action=" + quoteLogValueLocal(strings.TrimSpace(action)),
		"message=" + quoteLogValueLocal(input.Message),
		"content=" + quoteLogValueLocal(input.Content),
	}
	if images := strings.Join(cleanCSVItems(input.Images), ","); images != "" {
		parts = append(parts, "images="+quoteLogValueLocal(images))
	}
	if files := strings.Join(cleanCSVItems(input.Files), ","); files != "" {
		parts = append(parts, "files="+quoteLogValueLocal(files))
	}
	if to := strings.TrimSpace(input.To); to != "" {
		parts = append(parts, "to="+quoteLogValueLocal(to))
	}
	if subject := strings.TrimSpace(input.Subject); subject != "" {
		parts = append(parts, "subject="+quoteLogValueLocal(subject))
	}
	return strings.Join(parts, " ")
}

func formatSendParseLogLocal(action string, reply *localReplyParseContext) string {
	action = strings.TrimSpace(strings.ToLower(action))
	if action == "" {
		action = "send"
	}
	parts := []string{
		"stage=send-parse",
		"action=" + quoteLogValueLocal(strings.TrimSpace(action)),
	}
	if reply != nil {
		if strings.TrimSpace(reply.MessageID) != "" {
			parts = append(parts, "reply_to="+quoteLogValueLocal(reply.MessageID))
		}
		if len(reply.To) > 0 {
			parts = append(parts, "to="+quoteLogValueLocal(strings.Join(reply.To, ",")))
		}
		if strings.TrimSpace(reply.Subject) != "" {
			parts = append(parts, "subject="+quoteLogValueLocal(reply.Subject))
		}
		if strings.TrimSpace(reply.EnvelopeRaw) != "" {
			parts = append(parts, "raw_request="+quoteLogValueLocal(reply.EnvelopeRaw))
		}
	}
	return strings.Join(parts, " ")
}

func formatSendResultLogLocal(action string, result *localSendResult, replyTo string) string {
	action = strings.TrimSpace(strings.ToLower(action))
	if action == "" {
		action = "send"
	}
	if result == nil {
		return strings.Join([]string{
			"stage=send-result",
			"action=" + quoteLogValueLocal(strings.TrimSpace(action)),
		}, " ")
	}
	parts := []string{
		"stage=send-result",
		"action=" + quoteLogValueLocal(strings.TrimSpace(action)),
		"name=" + quoteLogValueLocal(strings.TrimSpace(result.Name)),
		"to=" + quoteLogValueLocal(strings.TrimSpace(result.To)),
	}
	if strings.TrimSpace(replyTo) != "" {
		parts = append(parts, "reply_to="+quoteLogValueLocal(strings.TrimSpace(replyTo)))
	}
	types := make([]string, 0, len(result.Sent))
	for _, item := range result.Sent {
		if strings.TrimSpace(item.Type) != "" {
			types = append(types, item.Type)
		}
	}
	if len(types) > 0 {
		parts = append(parts, "types="+quoteLogValueLocal(strings.Join(types, "+")))
	}
	if strings.TrimSpace(result.SMTPHeader) != "" {
		parts = append(parts, "smtp_header="+quoteLogValueLocal(result.SMTPHeader))
	}
	parts = append(parts, "result="+quoteLogValueLocal(mustJSONLogStringLocal(result)))
	return strings.Join(parts, " ")
}

func formatSendFailureLogLocal(action, step string, result *localSendResult, err error) string {
	action = strings.TrimSpace(strings.ToLower(action))
	if action == "" {
		action = "send"
	}
	parts := []string{
		"stage=send-failed",
		"action=" + quoteLogValueLocal(strings.TrimSpace(action)),
		"step=" + quoteLogValueLocal(strings.TrimSpace(step)),
	}
	if result != nil {
		if strings.TrimSpace(result.Name) != "" {
			parts = append(parts, "name="+quoteLogValueLocal(strings.TrimSpace(result.Name)))
		}
		if strings.TrimSpace(result.To) != "" {
			parts = append(parts, "to="+quoteLogValueLocal(strings.TrimSpace(result.To)))
		}
		if strings.TrimSpace(result.MessageID) != "" {
			parts = append(parts, "message_id="+quoteLogValueLocal(strings.TrimSpace(result.MessageID)))
		}
		if len(result.Sent) > 0 {
			types := make([]string, 0, len(result.Sent))
			for _, item := range result.Sent {
				if strings.TrimSpace(item.Type) != "" {
					types = append(types, item.Type)
				}
			}
			if len(types) > 0 {
				parts = append(parts, "types="+quoteLogValueLocal(strings.Join(types, "+")))
			}
		}
		if strings.TrimSpace(result.SMTPHeader) != "" {
			parts = append(parts, "smtp_header="+quoteLogValueLocal(result.SMTPHeader))
		}
		parts = append(parts, "result="+quoteLogValueLocal(mustJSONLogStringLocal(result)))
	}
	if err != nil {
		parts = append(parts, "err="+quoteLogValueLocal(err.Error()))
	}
	return strings.Join(parts, " ")
}

func formatSendRetryLogLocal(action, step string, attempt, max int, err error) string {
	action = strings.TrimSpace(strings.ToLower(action))
	if action == "" {
		action = "send"
	}
	parts := []string{
		"stage=send-retry",
		"action=" + quoteLogValueLocal(strings.TrimSpace(action)),
		"step=" + quoteLogValueLocal(strings.TrimSpace(step)),
		"attempt=" + strconv.Itoa(attempt),
		"max=" + strconv.Itoa(max),
	}
	if err != nil {
		parts = append(parts, "err="+quoteLogValueLocal(err.Error()))
	}
	return strings.Join(parts, " ")
}

func formatSendTerminateLogLocal(action, step string, max int, err error) string {
	action = strings.TrimSpace(strings.ToLower(action))
	if action == "" {
		action = "send"
	}
	parts := []string{
		"stage=send-terminate",
		"action=" + quoteLogValueLocal(strings.TrimSpace(action)),
		"step=" + quoteLogValueLocal(strings.TrimSpace(step)),
		"max=" + strconv.Itoa(max),
	}
	if err != nil {
		parts = append(parts, "err="+quoteLogValueLocal(err.Error()))
	}
	return strings.Join(parts, " ")
}

func mustJSONLogStringLocal(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf(`{"marshalError":%q}`, err.Error())
	}
	return string(data)
}

func quoteLogValueLocal(value string) string {
	return strconv.Quote(strings.TrimSpace(value))
}

func extractSMTPHeaderBlockLocal(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	separator := []byte("\r\n\r\n")
	index := bytes.Index(raw, separator)
	if index < 0 {
		return strings.TrimSpace(string(raw))
	}
	return strings.TrimSpace(string(raw[:index]))
}

func writeTimestampedLocalMessageLog(messageLog io.Writer, logger *log.Logger, at, content string) {
	if messageLog == nil {
		return
	}
	line := emailsvc.HumanizeLogEntry(strings.TrimSpace(at), strings.TrimSpace(content)) + "\n"
	if _, err := io.WriteString(messageLog, line); err != nil && logger != nil {
		logger.Printf("stage=message-log err=%v", err)
	}
}

func runtimeLogPathForLocal(messageLogPath string) string {
	messageLogPath = strings.TrimSpace(messageLogPath)
	if messageLogPath == "" {
		messageLogPath = defaultLogFile
	}
	dir := filepath.Dir(messageLogPath)
	return filepath.Join(dir, defaultRunLog)
}

func downloadDirForLogLocal(messageLogPath string) string {
	messageLogPath = strings.TrimSpace(messageLogPath)
	if messageLogPath == "" {
		messageLogPath = defaultLogFile
	}
	dir := filepath.Dir(messageLogPath)
	return filepath.Join(dir, "email_artifacts")
}

func stateFileForLogLocal(messageLogPath string) string {
	messageLogPath = strings.TrimSpace(messageLogPath)
	if messageLogPath == "" {
		messageLogPath = defaultLogFile
	}
	dir := filepath.Dir(messageLogPath)
	return filepath.Join(dir, "email.state.json")
}

func buildConnectBaseArgsLocal(flags map[string]string) ([]string, error) {
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

func resolveConnectBinaryLocal(flags map[string]string) string {
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

	for _, candidate := range connectBinaryCandidatesLocal() {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return candidate
		}
	}
	return "connect"
}

func resolveConnectPrefixLocal(flags map[string]string) []string {
	explicit := strings.TrimSpace(connectsvc.FirstValue(flags, "connect-bin"))
	if explicit != "" {
		base := strings.ToLower(filepath.Base(explicit))
		if base == "proxy" || base == "integration" || strings.HasPrefix(base, "proxy.") || strings.HasPrefix(base, "integration.") {
			return []string{"connect"}
		}
		return nil
	}

	resolved := resolveConnectBinaryLocal(flags)
	base := strings.ToLower(filepath.Base(resolved))
	if base == "proxy" || base == "integration" || strings.HasPrefix(base, "proxy.") || strings.HasPrefix(base, "integration.") {
		return []string{"connect"}
	}
	return nil
}

func connectBinaryCandidatesLocal() []string {
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

func detectAttachmentContentTypeLocal(path string, data []byte) string {
	if ext := strings.TrimSpace(strings.ToLower(filepath.Ext(path))); ext != "" {
		if kind, ok := localAttachmentContentTypeByExt[ext]; ok {
			return kind
		}
		if kind := mime.TypeByExtension(ext); kind != "" {
			return kind
		}
	}
	if len(data) > 0 {
		return http.DetectContentType(data)
	}
	return "application/octet-stream"
}

var localAttachmentContentTypeByExt = map[string]string{
	".pdf":  "application/pdf",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".mp4":  "video/mp4",
	".opus": "audio/ogg",
}

func writeHeaderLineLocal(buf *bytes.Buffer, key, value string) {
	buf.WriteString(key)
	buf.WriteString(": ")
	buf.WriteString(value)
	buf.WriteString("\r\n")
}

func encodeMIMEHeaderLocal(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, r := range value {
		if r > 127 {
			return mime.QEncoding.Encode("utf-8", value)
		}
	}
	return value
}

func escapeHeaderParamLocal(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, `"`, `'`)
	return value
}

type base64LineWriterLocal struct {
	w     io.Writer
	lineN int
}

func newBase64LineWriterLocal(w io.Writer) io.Writer {
	return &base64LineWriterLocal{w: w}
}

func (w *base64LineWriterLocal) Write(p []byte) (int, error) {
	total := len(p)
	for len(p) > 0 {
		remaining := 76 - w.lineN
		if remaining <= 0 {
			if _, err := io.WriteString(w.w, "\r\n"); err != nil {
				return total - len(p), err
			}
			w.lineN = 0
			remaining = 76
		}
		if remaining > len(p) {
			remaining = len(p)
		}
		if _, err := w.w.Write(p[:remaining]); err != nil {
			return total - len(p), err
		}
		w.lineN += remaining
		p = p[remaining:]
	}
	return total, nil
}

func stringPathLocal(payload map[string]any, keys ...string) string {
	var current any = payload
	for _, key := range keys {
		next, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current, ok = next[key]
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

func decodeHeaderValueLocal(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	decoded, err := (&mime.WordDecoder{}).DecodeHeader(value)
	if err != nil {
		return value
	}
	return strings.TrimSpace(decoded)
}

func firstNonEmptyLocal(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func dedupeStringsLocal(items []string) []string {
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

func removeFirstString(items []string, target string) []string {
	target = strings.TrimSpace(target)
	if target == "" {
		return items
	}
	out := make([]string, 0, len(items))
	removed := false
	for _, item := range items {
		if !removed && strings.TrimSpace(item) == target {
			removed = true
			continue
		}
		out = append(out, item)
	}
	return out
}

func ensureParentDirLocal(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}
