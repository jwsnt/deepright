package emailsvc

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
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"connect/connectsvc"
	"github.com/google/uuid"
)

const defaultSMTPPort = 465

var messageIDPattern = regexp.MustCompile(`<[^<>\r\n]+>`)
var probeSMTPConfig = func(ctx context.Context, cfg Config) error {
	host, port, err := normalizeSMTPAddress(cfg.EmailSMTP)
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	client, conn, err := dialSMTPContext(ctx, host, addr, port)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer client.Close()

	if port != defaultSMTPPort {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
				return err
			}
		}
	}
	if strings.TrimSpace(cfg.EmailPassword) != "" {
		if ok, _ := client.Extension("AUTH"); !ok {
			return errors.New("smtp server does not advertise AUTH")
		}
		auth := smtp.PlainAuth("", strings.TrimSpace(cfg.Email), strings.TrimSpace(cfg.EmailPassword), host)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	return client.Quit()
}

type MetaLoader interface {
	Load(ctx context.Context) (*connectsvc.Meta, Config, error)
}

type SendInput struct {
	Action  string
	Message string
	Content string
	Images  []string
	Files   []string
}

type SendResult struct {
	Name       string       `json:"name"`
	To         string       `json:"to"`
	MessageID  string       `json:"messageId,omitempty"`
	Sent       []SentRecord `json:"sent"`
	SMTPHeader string       `json:"-"`
}

type SentRecord struct {
	Type      string `json:"type"`
	MessageID string `json:"messageId,omitempty"`
	Path      string `json:"path,omitempty"`
}

type SendAttachment struct {
	Type        string
	Path        string
	FileName    string
	ContentType string
	Data        []byte
}

type OutgoingMail struct {
	From        string
	To          []string
	Subject     string
	TextBody    string
	HTMLBody    string
	InReplyTo   string
	References  []string
	MessageID   string
	Date        time.Time
	Attachments []SendAttachment
	Raw         []byte
}

type ReplyEnvelope struct {
	To         []string
	Subject    string
	MessageID  string
	References []string
}

type SMTPTransport interface {
	Send(ctx context.Context, cfg Config, mail OutgoingMail) (string, error)
}

type smtpTransport struct{}

type Sender struct {
	loader     MetaLoader
	transport  SMTPTransport
	logger     *log.Logger
	messageLog io.Writer
	now        func() time.Time
}

func (s *Service) Load(ctx context.Context) (*connectsvc.Meta, Config, error) {
	return s.loadConfig(ctx)
}

func NewSender(loader MetaLoader, logger *log.Logger, messageLog io.Writer) *Sender {
	return &Sender{
		loader:     loader,
		logger:     logger,
		messageLog: messageLog,
		now:        time.Now,
	}
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

	reply, err := extractReplyEnvelope(input.Message)
	if err != nil {
		s.writeMessageLog(at, formatSendFailureLog(action, "parse-message", nil, err))
		return nil, err
	}
	s.writeMessageLog(at, formatSendParseLog(action, &reply))
	if len(reply.To) == 0 {
		err := errors.New("message.replyTo not found")
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

	attachments, records, err := buildSendAttachments(images, files)
	if err != nil {
		s.writeMessageLog(at, formatSendFailureLog(action, "build-attachments", nil, err))
		return nil, err
	}

	outgoing := OutgoingMail{
		From:        strings.TrimSpace(cfg.Email),
		To:          append([]string(nil), reply.To...),
		Subject:     replySubject(reply.Subject),
		TextBody:    buildPlainTextBody(content),
		HTMLBody:    buildHTMLBody(content),
		InReplyTo:   strings.TrimSpace(reply.MessageID),
		MessageID:   buildOutgoingMessageID(cfg.Email),
		Date:        s.clock(),
		Attachments: attachments,
	}
	outgoing.References = append([]string(nil), reply.References...)
	if outgoing.From == "" {
		err := errors.New("meta.email is required")
		s.writeMessageLog(at, formatSendFailureLog(action, "build-outgoing", nil, err))
		return nil, err
	}
	rawMessage, err := buildSMTPMessage(outgoing)
	if err != nil {
		s.writeMessageLog(at, formatSendFailureLog(action, "build-smtp", nil, err))
		return nil, err
	}
	outgoing.Raw = rawMessage
	smtpHeader := extractSMTPHeaderBlock(rawMessage)

	messageID, err := s.ensureTransport().Send(ctx, cfg, outgoing)
	if err != nil {
		partial := &SendResult{
			Name:       firstNonEmpty(meta.Name, DefaultName),
			To:         strings.Join(reply.To, ","),
			MessageID:  outgoing.MessageID,
			SMTPHeader: smtpHeader,
		}
		s.writeMessageLog(at, formatSendFailureLog(action, "send-smtp", partial, err))
		return nil, err
	}
	if strings.TrimSpace(messageID) == "" {
		messageID = outgoing.MessageID
	}

	sent := make([]SentRecord, 0, 1+len(records))
	if content != "" {
		sent = append(sent, SentRecord{
			Type:      "text",
			MessageID: messageID,
		})
	}
	for _, record := range records {
		record.MessageID = messageID
		sent = append(sent, record)
	}

	result := &SendResult{
		Name:       firstNonEmpty(meta.Name, DefaultName),
		To:         strings.Join(reply.To, ","),
		MessageID:  messageID,
		Sent:       sent,
		SMTPHeader: smtpHeader,
	}
	s.writeMessageLog(at, formatSendResultLog(action, result, reply.MessageID))
	return result, nil
}

func (s *Sender) ensureTransport() SMTPTransport {
	if s != nil && s.transport != nil {
		return s.transport
	}
	return smtpTransport{}
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

func (smtpTransport) Send(ctx context.Context, cfg Config, outgoing OutgoingMail) (string, error) {
	host, port, err := normalizeSMTPAddress(cfg.EmailSMTP)
	if err != nil {
		return "", err
	}
	raw := outgoing.Raw
	if len(raw) == 0 {
		raw, err = buildSMTPMessage(outgoing)
		if err != nil {
			return "", err
		}
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	client, conn, err := dialSMTPContext(ctx, host, addr, port)
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

func dialSMTPContext(ctx context.Context, host, addr string, port int) (*smtp.Client, net.Conn, error) {
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

func normalizeSMTPAddress(value string) (string, int, error) {
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

func buildSMTPMessage(outgoing OutgoingMail) ([]byte, error) {
	if strings.TrimSpace(outgoing.From) == "" {
		return nil, errors.New("from is required")
	}
	if len(outgoing.To) == 0 {
		return nil, errors.New("to is required")
	}
	if strings.TrimSpace(outgoing.MessageID) == "" {
		outgoing.MessageID = buildOutgoingMessageID(outgoing.From)
	}
	if outgoing.Date.IsZero() {
		outgoing.Date = time.Now()
	}
	if strings.TrimSpace(outgoing.TextBody) == "" && strings.TrimSpace(outgoing.HTMLBody) == "" && len(outgoing.Attachments) == 0 {
		return nil, errors.New("mail body is required")
	}

	var body bytes.Buffer
	contentType, transferEncoding, err := writeMessageBody(&body, outgoing)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	writeHeaderLine(&buf, "From", strings.TrimSpace(outgoing.From))
	writeHeaderLine(&buf, "To", strings.Join(cleanCSVItems(outgoing.To), ", "))
	if subject := strings.TrimSpace(outgoing.Subject); subject != "" {
		writeHeaderLine(&buf, "Subject", encodeMIMEHeader(subject))
	}
	writeHeaderLine(&buf, "Date", outgoing.Date.Format(time.RFC1123Z))
	writeHeaderLine(&buf, "Message-ID", strings.TrimSpace(outgoing.MessageID))
	if value := strings.TrimSpace(outgoing.InReplyTo); value != "" {
		writeHeaderLine(&buf, "In-Reply-To", value)
	}
	if len(outgoing.References) > 0 {
		writeHeaderLine(&buf, "References", strings.Join(outgoing.References, " "))
	}
	writeHeaderLine(&buf, "MIME-Version", "1.0")
	writeHeaderLine(&buf, "Content-Type", contentType)
	if strings.TrimSpace(transferEncoding) != "" {
		writeHeaderLine(&buf, "Content-Transfer-Encoding", transferEncoding)
	}
	buf.WriteString("\r\n")
	buf.Write(body.Bytes())
	return buf.Bytes(), nil
}

func writeMessageBody(w io.Writer, outgoing OutgoingMail) (string, string, error) {
	hasText := strings.TrimSpace(outgoing.TextBody) != ""
	hasHTML := strings.TrimSpace(outgoing.HTMLBody) != ""
	if len(outgoing.Attachments) == 0 {
		if hasText && hasHTML {
			writer := multipart.NewWriter(w)
			if err := writeAlternativeParts(writer, outgoing.TextBody, outgoing.HTMLBody); err != nil {
				return "", "", err
			}
			if err := writer.Close(); err != nil {
				return "", "", err
			}
			return fmt.Sprintf(`multipart/alternative; boundary="%s"`, writer.Boundary()), "", nil
		}
		if hasHTML {
			if err := writeSingleTextBody(w, "text/html", outgoing.HTMLBody); err != nil {
				return "", "", err
			}
			return `text/html; charset=UTF-8`, "quoted-printable", nil
		}
		if err := writeSingleTextBody(w, "text/plain", outgoing.TextBody); err != nil {
			return "", "", err
		}
		return `text/plain; charset=UTF-8`, "quoted-printable", nil
	}

	writer := multipart.NewWriter(w)
	if hasText || hasHTML {
		if hasText && hasHTML {
			altBoundary := "alt-" + uuid.NewString()
			header := textproto.MIMEHeader{}
			header.Set("Content-Type", fmt.Sprintf(`multipart/alternative; boundary="%s"`, altBoundary))
			part, err := writer.CreatePart(header)
			if err != nil {
				return "", "", err
			}
			altWriter := multipart.NewWriter(part)
			if err := altWriter.SetBoundary(altBoundary); err != nil {
				return "", "", err
			}
			if err := writeAlternativeParts(altWriter, outgoing.TextBody, outgoing.HTMLBody); err != nil {
				return "", "", err
			}
			if err := altWriter.Close(); err != nil {
				return "", "", err
			}
		} else if hasHTML {
			if err := writeMultipartTextPart(writer, "text/html", outgoing.HTMLBody); err != nil {
				return "", "", err
			}
		} else {
			if err := writeMultipartTextPart(writer, "text/plain", outgoing.TextBody); err != nil {
				return "", "", err
			}
		}
	}

	for _, attachment := range outgoing.Attachments {
		if err := writeAttachmentPart(writer, attachment); err != nil {
			return "", "", err
		}
	}
	if err := writer.Close(); err != nil {
		return "", "", err
	}
	return fmt.Sprintf(`multipart/mixed; boundary="%s"`, writer.Boundary()), "", nil
}

func writeAlternativeParts(writer *multipart.Writer, textBody, htmlBody string) error {
	if strings.TrimSpace(textBody) != "" {
		if err := writeMultipartTextPart(writer, "text/plain", textBody); err != nil {
			return err
		}
	}
	if strings.TrimSpace(htmlBody) != "" {
		if err := writeMultipartTextPart(writer, "text/html", htmlBody); err != nil {
			return err
		}
	}
	return nil
}

func writeSingleTextBody(w io.Writer, mediaType, body string) error {
	qp := quotedprintable.NewWriter(w)
	_, err := io.WriteString(qp, body)
	if closeErr := qp.Close(); err == nil {
		err = closeErr
	}
	return err
}

func writeMultipartTextPart(writer *multipart.Writer, mediaType, body string) error {
	header := textproto.MIMEHeader{}
	header.Set("Content-Type", mediaType+`; charset=UTF-8`)
	header.Set("Content-Transfer-Encoding", "quoted-printable")
	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}
	return writeSingleTextBody(part, mediaType, body)
}

func writeAttachmentPart(writer *multipart.Writer, attachment SendAttachment) error {
	contentType := strings.TrimSpace(attachment.ContentType)
	if contentType == "" {
		contentType = detectAttachmentContentType(attachment.Path, attachment.Data)
	}
	fileName := strings.TrimSpace(attachment.FileName)
	if fileName == "" {
		fileName = filepath.Base(strings.TrimSpace(attachment.Path))
	}
	if fileName == "" {
		fileName = attachment.Type + ".bin"
	}
	header := textproto.MIMEHeader{}
	header.Set("Content-Type", contentType+`; name="`+escapeHeaderParam(fileName)+`"`)
	header.Set("Content-Transfer-Encoding", "base64")
	header.Set("Content-Disposition", `attachment; filename="`+escapeHeaderParam(fileName)+`"`)
	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}
	encoder := base64.NewEncoder(base64.StdEncoding, newBase64LineWriter(part))
	if _, err := encoder.Write(attachment.Data); err != nil {
		encoder.Close()
		return err
	}
	return encoder.Close()
}

func buildSendAttachments(images, files []string) ([]SendAttachment, []SentRecord, error) {
	attachments := make([]SendAttachment, 0, len(images)+len(files))
	records := make([]SentRecord, 0, len(images)+len(files))
	for _, imagePath := range images {
		attachment, err := readSendAttachment("image", imagePath)
		if err != nil {
			return nil, nil, err
		}
		attachments = append(attachments, attachment)
		records = append(records, SentRecord{Type: "image", Path: attachment.Path})
	}
	for _, filePath := range files {
		attachment, err := readSendAttachment("file", filePath)
		if err != nil {
			return nil, nil, err
		}
		attachments = append(attachments, attachment)
		records = append(records, SentRecord{Type: "file", Path: attachment.Path})
	}
	return attachments, records, nil
}

func readSendAttachment(kind, rawPath string) (SendAttachment, error) {
	path := strings.TrimSpace(rawPath)
	if path == "" {
		return SendAttachment{}, errors.New("attachment path is required")
	}
	absPath, err := filepath.Abs(path)
	if err == nil {
		path = absPath
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return SendAttachment{}, err
	}
	return SendAttachment{
		Type:        kind,
		Path:        path,
		FileName:    filepath.Base(path),
		ContentType: detectAttachmentContentType(path, data),
		Data:        data,
	}, nil
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

type replyParseContext struct {
	ReplyEnvelope
	EnvelopeRaw string
}

func extractReplyEnvelope(raw string) (replyParseContext, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return replyParseContext{}, errors.New("message is required")
	}

	var requestPayload map[string]any
	if err := json.Unmarshal([]byte(raw), &requestPayload); err != nil {
		return replyParseContext{}, errors.New("message must be valid json")
	}

	envelopeRaw := strings.TrimSpace(firstNonEmpty(
		stringPath(requestPayload, "rawRequest"),
		stringPath(requestPayload, "raw_request"),
	))
	if envelopeRaw == "" {
		return replyParseContext{}, errors.New("message.messageId not found")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(envelopeRaw), &payload); err != nil {
		return replyParseContext{}, errors.New("message.messageId not found")
	}

	var envelope ReplyEnvelope
	collectReplyEnvelope(&envelope, payload)
	envelope.MessageID = normalizeReplyMessageID(envelope.MessageID)
	if strings.TrimSpace(envelope.MessageID) == "" {
		return replyParseContext{}, errors.New("message.messageId not found")
	}
	envelope.To = dedupeStrings(cleanCSVItems(envelope.To))
	if len(envelope.To) == 0 {
		return replyParseContext{}, errors.New("message.replyTo not found")
	}
	envelope.Subject = replySubject(envelope.Subject)
	envelope.References = appendReferenceIDs(normalizeReplyMessageIDs(envelope.References), envelope.MessageID)
	return replyParseContext{
		ReplyEnvelope: envelope,
		EnvelopeRaw:   envelopeRaw,
	}, nil
}

func collectReplyEnvelope(envelope *ReplyEnvelope, payload map[string]any) {
	if envelope == nil || payload == nil {
		return
	}

	if strings.TrimSpace(envelope.MessageID) == "" {
		envelope.MessageID = normalizeReplyMessageID(stringPath(payload, "message", "messageId"))
	}
	if strings.TrimSpace(envelope.Subject) == "" {
		envelope.Subject = stringPath(payload, "message", "subject")
	}
	envelope.To = append(envelope.To, parseAddressCandidates(
		stringPath(payload, "message", "from"),
	)...)

	if messageValue, ok := payload["message"]; ok {
		if messageMap, ok := messageValue.(map[string]any); ok {
			if raw := strings.TrimSpace(stringPath(messageMap, "raw")); raw != "" {
				var rawPayload map[string]any
				if err := json.Unmarshal([]byte(raw), &rawPayload); err == nil {
					if headers, ok := rawPayload["headers"]; ok {
						mergeReplyEnvelopeHeaders(envelope, headers)
					}
				}
			}
		}
	}
}

func mergeReplyEnvelopeHeaders(envelope *ReplyEnvelope, raw any) {
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
				envelope.MessageID = normalizeReplyMessageID(value)
			}
		case strings.EqualFold(name, "Subject"):
			if strings.TrimSpace(envelope.Subject) == "" {
				envelope.Subject = decodeHeaderValue(value)
			}
		case strings.EqualFold(name, "Reply-To"):
			envelope.To = append(envelope.To, parseAddressCandidates(value)...)
		case strings.EqualFold(name, "From"):
			envelope.To = append(envelope.To, parseAddressCandidates(value)...)
		case strings.EqualFold(name, "References"):
			envelope.References = append(envelope.References, extractReplyMessageIDs(value)...)
		case strings.EqualFold(name, "In-Reply-To"):
			envelope.References = append(envelope.References, extractReplyMessageIDs(value)...)
		}
	}
}

func normalizeReplyMessageID(value string) string {
	ids := extractReplyMessageIDs(value)
	if len(ids) > 0 {
		return ids[0]
	}
	return strings.TrimSpace(value)
}

func normalizeReplyMessageIDs(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, extractReplyMessageIDs(value)...)
	}
	return dedupeStrings(out)
}

func extractReplyMessageIDs(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	matches := messageIDPattern.FindAllString(value, -1)
	if len(matches) > 0 {
		return dedupeStrings(matches)
	}
	return dedupeStrings(strings.Fields(value))
}

func parseAddressCandidates(values ...string) []string {
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
		address := extractMailAddress(value)
		if address != "" {
			out = append(out, address)
		}
	}
	return out
}

func extractReferenceIDs(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	fields := strings.Fields(value)
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			out = append(out, field)
		}
	}
	return out
}

func appendReferenceIDs(existing []string, messageID string) []string {
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

func replySubject(subject string) string {
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

func buildPlainTextBody(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	if strings.Contains(content, "<") && strings.Contains(content, ">") {
		text := htmlToText(content)
		if text != "" {
			return text
		}
	}
	return content
}

func buildHTMLBody(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	if strings.Contains(content, "<") && strings.Contains(content, ">") {
		return content
	}
	lines := strings.Split(content, "\n")
	escaped := make([]string, 0, len(lines))
	for _, line := range lines {
		escaped = append(escaped, html.EscapeString(line))
	}
	return "<div>" + strings.Join(escaped, "<br/>") + "</div>"
}

func buildOutgoingMessageID(email string) string {
	domain := "localhost"
	if at := strings.LastIndex(strings.TrimSpace(email), "@"); at >= 0 && at+1 < len(strings.TrimSpace(email)) {
		domain = strings.TrimSpace(email[at+1:])
	}
	return "<" + uuid.NewString() + "@" + domain + ">"
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

func formatSendParseLog(action string, reply *replyParseContext) string {
	action = strings.TrimSpace(strings.ToLower(action))
	if action == "" {
		action = "send"
	}
	parts := []string{
		"stage=send-parse",
		"action=" + quoteLogValue(action),
	}
	if reply != nil {
		if strings.TrimSpace(reply.MessageID) != "" {
			parts = append(parts, "reply_to="+quoteLogValue(reply.MessageID))
		}
		if len(reply.To) > 0 {
			parts = append(parts, "to="+quoteLogValue(strings.Join(reply.To, ",")))
		}
		if strings.TrimSpace(reply.Subject) != "" {
			parts = append(parts, "subject="+quoteLogValue(reply.Subject))
		}
		if strings.TrimSpace(reply.EnvelopeRaw) != "" {
			parts = append(parts, "raw_request="+quoteLogValue(reply.EnvelopeRaw))
		}
	}
	return strings.Join(parts, " ")
}

func formatSendResultLog(action string, result *SendResult, replyTo string) string {
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
		"to=" + quoteLogValue(strings.TrimSpace(result.To)),
	}
	if strings.TrimSpace(replyTo) != "" {
		parts = append(parts, "reply_to="+quoteLogValue(strings.TrimSpace(replyTo)))
	}
	types := make([]string, 0, len(result.Sent))
	for _, item := range result.Sent {
		if strings.TrimSpace(item.Type) != "" {
			types = append(types, item.Type)
		}
	}
	if len(types) > 0 {
		parts = append(parts, "types="+quoteLogValue(strings.Join(types, "+")))
	}
	if strings.TrimSpace(result.SMTPHeader) != "" {
		parts = append(parts, "smtp_header="+quoteLogValue(result.SMTPHeader))
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
		if strings.TrimSpace(result.To) != "" {
			parts = append(parts, "to="+quoteLogValue(strings.TrimSpace(result.To)))
		}
		if strings.TrimSpace(result.MessageID) != "" {
			parts = append(parts, "message_id="+quoteLogValue(strings.TrimSpace(result.MessageID)))
		}
		if len(result.Sent) > 0 {
			types := make([]string, 0, len(result.Sent))
			for _, item := range result.Sent {
				if strings.TrimSpace(item.Type) != "" {
					types = append(types, item.Type)
				}
			}
			if len(types) > 0 {
				parts = append(parts, "types="+quoteLogValue(strings.Join(types, "+")))
			}
		}
		if strings.TrimSpace(result.SMTPHeader) != "" {
			parts = append(parts, "smtp_header="+quoteLogValue(result.SMTPHeader))
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

func extractSMTPHeaderBlock(raw []byte) string {
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

func stringPath(payload map[string]any, keys ...string) string {
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

func encodeMIMEHeader(value string) string {
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

func detectAttachmentContentType(path string, data []byte) string {
	if ext := strings.TrimSpace(filepath.Ext(path)); ext != "" {
		if kind := mime.TypeByExtension(ext); kind != "" {
			return kind
		}
	}
	if len(data) > 0 {
		return http.DetectContentType(data)
	}
	return "application/octet-stream"
}

func writeHeaderLine(buf *bytes.Buffer, key, value string) {
	buf.WriteString(key)
	buf.WriteString(": ")
	buf.WriteString(value)
	buf.WriteString("\r\n")
}

func escapeHeaderParam(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, `"`, `'`)
	return value
}

type base64LineWriter struct {
	w     io.Writer
	lineN int
}

func newBase64LineWriter(w io.Writer) io.Writer {
	return &base64LineWriter{w: w}
}

func (w *base64LineWriter) Write(p []byte) (int, error) {
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
