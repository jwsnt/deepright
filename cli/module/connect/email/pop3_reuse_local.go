package main

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
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"net/textproto"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"connect/emailsvc"
	"github.com/emersion/go-message"
	pop3 "github.com/knadh/go-pop3"
	"github.com/saintfish/chardet"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/transform"
)

const (
	defaultPOP3PortLocal    = 995
	pop3CommandTimeoutLocal = 10 * time.Second
	initialTimelineLagLocal = 30 * time.Minute
	maxPOP3ReconnectsLocal  = 8
)

var fallbackDNSAddrLocal = []string{
	"223.5.5.5:53",
	"223.6.6.6:53",
}

type localRawMail struct {
	ID  int
	UID string
	Raw []byte
}

type localPOP3Session interface {
	Auth(user, password string) error
	Noop() error
	Uidl(msgID int) ([]pop3.MessageID, error)
	RetrRaw(msgID int) (*bytes.Buffer, error)
	Quit() error
}

type localPersistentPOP3Fetcher struct {
	downloadDir string
	nowFn       func() time.Time
	openConn    func(ctx context.Context, host string, port int) (localPOP3Session, error)

	mu                sync.Mutex
	session           localPOP3Session
	sessionKey        string
	processedUIDs     map[string]struct{}
	useTimelineFilter bool
}

func newLocalPersistentPOP3Fetcher(downloadDir string) *localPersistentPOP3Fetcher {
	return &localPersistentPOP3Fetcher{
		downloadDir:       strings.TrimSpace(downloadDir),
		nowFn:             time.Now,
		useTimelineFilter: true,
	}
}

func (f *localPersistentPOP3Fetcher) ConfigureIncremental(processedUIDs map[string]struct{}, useTimelineFilter bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(processedUIDs) == 0 {
		f.processedUIDs = nil
	} else {
		f.processedUIDs = make(map[string]struct{}, len(processedUIDs))
		for uid := range processedUIDs {
			uid = strings.TrimSpace(uid)
			if uid == "" {
				continue
			}
			f.processedUIDs[uid] = struct{}{}
		}
	}
	f.useTimelineFilter = useTimelineFilter
}

func (f *localPersistentPOP3Fetcher) Fetch(ctx context.Context, cfg emailsvc.Config, since time.Time) ([]emailsvc.FetchedMail, error) {
	rawMessages, err := f.fetchRaw(ctx, cfg)
	if err != nil {
		return nil, err
	}

	now := f.now()
	recentCutoff, useRecentCutoff := recentMailCutoffLocal(cfg.ScanSeconds, now)
	out := make([]emailsvc.FetchedMail, 0, len(rawMessages))
	for _, item := range rawMessages {
		parsed, err := parseFetchedMailLocal(item, f.downloadDir, now)
		if err != nil {
			parsed = fallbackFetchedMailLocal(item, now)
		}
		mailTime := effectiveMailTimeLocal(parsed, now)
		if useRecentCutoff && !mailTime.After(recentCutoff) {
			continue
		}
		if f.shouldApplyTimelineFilter() && !since.IsZero() && !mailTime.After(since) {
			continue
		}
		out = append(out, parsed)
	}
	return out, nil
}

func (f *localPersistentPOP3Fetcher) fetchRaw(ctx context.Context, cfg emailsvc.Config) ([]localRawMail, error) {
	host, port, err := normalizePOP3AddressLocal(strings.TrimSpace(cfg.EmailPOP3))
	if err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return f.fetchRawLocked(ctx, cfg, host, port)
}

func (f *localPersistentPOP3Fetcher) fetchRawLocked(ctx context.Context, cfg emailsvc.Config, host string, port int) ([]localRawMail, error) {
	defer f.closeLocked()

	var (
		conn       localPOP3Session
		uidls      []pop3.MessageID
		reconnects int
	)
	reconnect := func(err error) bool {
		if !errors.Is(err, io.EOF) || reconnects >= maxPOP3ReconnectsLocal {
			return false
		}
		reconnects++
		f.closeLocked()
		conn = nil
		return true
	}

	for {
		var err error
		if conn == nil {
			conn, err = f.ensureSessionLocked(ctx, cfg, host, port)
			if err != nil {
				if reconnect(err) {
					continue
				}
				return nil, err
			}
		}
		uidls, err = conn.Uidl(0)
		if err == nil {
			break
		}
		f.closeLocked()
		if reconnect(err) {
			continue
		}
		return nil, err
	}
	sort.Slice(uidls, func(i, j int) bool {
		return uidls[i].ID < uidls[j].ID
	})

	out := make([]localRawMail, 0, len(uidls))
	for _, item := range uidls {
		if f.isProcessedUIDLocked(item.UID) {
			continue
		}
		for {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			var err error
			if conn == nil {
				conn, err = f.ensureSessionLocked(ctx, cfg, host, port)
				if err != nil {
					if reconnect(err) {
						continue
					}
					return nil, err
				}
			}
			buf, err := conn.RetrRaw(item.ID)
			if err != nil {
				f.closeLocked()
				conn = nil
				if reconnect(err) {
					continue
				}
				return nil, err
			}
			out = append(out, localRawMail{
				ID:  item.ID,
				UID: strings.TrimSpace(item.UID),
				Raw: append([]byte(nil), buf.Bytes()...),
			})
			break
		}
	}
	return out, nil
}

func (f *localPersistentPOP3Fetcher) ensureSessionLocked(ctx context.Context, cfg emailsvc.Config, host string, port int) (localPOP3Session, error) {
	key := localPOP3SessionKey(cfg, host, port)
	if f.session != nil {
		f.closeLocked()
	}

	conn, err := f.newConn(ctx, host, port)
	if err != nil {
		return nil, err
	}
	if err := authenticatePOP3Local(conn, strings.TrimSpace(cfg.Email), strings.TrimSpace(cfg.EmailPassword), host); err != nil {
		_ = conn.Quit()
		return nil, err
	}
	f.session = conn
	f.sessionKey = key
	return conn, nil
}

func (f *localPersistentPOP3Fetcher) newConn(ctx context.Context, host string, port int) (localPOP3Session, error) {
	if f.openConn != nil {
		return f.openConn(ctx, host, port)
	}
	client := pop3.New(pop3.Opt{
		Host:          host,
		Port:          port,
		TLSEnabled:    false,
		TLSSkipVerify: false,
		DialTimeout:   10 * time.Second,
		Dialer:        &localTLSDialer{ctx: ctx, host: host},
	})
	return client.NewConn()
}

func (f *localPersistentPOP3Fetcher) closeLocked() {
	if f.session != nil {
		_ = f.session.Quit()
	}
	f.session = nil
	f.sessionKey = ""
}

func (f *localPersistentPOP3Fetcher) now() time.Time {
	if f != nil && f.nowFn != nil {
		return f.nowFn()
	}
	return time.Now()
}

func (f *localPersistentPOP3Fetcher) shouldApplyTimelineFilter() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.useTimelineFilter
}

func recentMailCutoffLocal(scanSeconds int, now time.Time) (time.Time, bool) {
	if scanSeconds <= 0 {
		return time.Time{}, false
	}
	return now.Add(-time.Duration(scanSeconds*initialTimelineScanMultiplierLocal) * time.Second), true
}

func (f *localPersistentPOP3Fetcher) isProcessedUIDLocked(uid string) bool {
	uid = strings.TrimSpace(uid)
	if uid == "" || len(f.processedUIDs) == 0 {
		return false
	}
	_, ok := f.processedUIDs[uid]
	return ok
}

func localPOP3SessionKey(cfg emailsvc.Config, host string, port int) string {
	return strings.Join([]string{
		strings.TrimSpace(cfg.Email),
		strings.TrimSpace(cfg.EmailPassword),
		strings.TrimSpace(host),
		strconv.Itoa(port),
	}, "\x00")
}

func normalizePOP3AddressLocal(value string) (string, int, error) {
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
	return value, defaultPOP3PortLocal, nil
}

func authenticatePOP3Local(conn localPOP3Session, email, password, host string) error {
	candidates := pop3UsernameCandidatesLocal(email)
	var errs []string
	for _, candidate := range candidates {
		if err := conn.Auth(candidate, password); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", candidate, err))
			if shouldRetryPOP3WithLocalPartLocal(err) {
				continue
			}
			return fmt.Errorf("pop3 auth failed server=%s account=%s username=%s err=%w", strings.TrimSpace(host), maskEmailAccountLocal(email), candidate, err)
		}
		return nil
	}
	if len(errs) > 0 {
		return fmt.Errorf("pop3 auth failed server=%s account=%s tried=%s", strings.TrimSpace(host), maskEmailAccountLocal(email), strings.Join(errs, "; "))
	}
	return errors.New("pop3 auth failed")
}

func pop3UsernameCandidatesLocal(email string) []string {
	email = strings.TrimSpace(email)
	if email == "" {
		return nil
	}
	out := []string{email}
	if at := strings.Index(email, "@"); at > 0 {
		localPart := strings.TrimSpace(email[:at])
		if localPart != "" && !strings.EqualFold(localPart, email) {
			out = append(out, localPart)
		}
	}
	return out
}

func shouldRetryPOP3WithLocalPartLocal(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(strings.TrimSpace(formatErrorForLogLocal(err)))
	return strings.Contains(text, "username not valid") ||
		strings.Contains(text, "invalid user") ||
		strings.Contains(text, "unknown user") ||
		strings.Contains(text, "unknown username") ||
		strings.Contains(text, "no such user") ||
		strings.Contains(text, "user unknown")
}

func maskEmailAccountLocal(email string) string {
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

func fallbackFetchedMailLocal(item localRawMail, now time.Time) emailsvc.FetchedMail {
	rawHeaders := rawHeadersLocal(item.Raw)
	headers := mailHeaderToJSONLocal(rawHeaders)
	rawContent := toUTF8StringLocal(item.Raw)
	result := emailsvc.FetchedMail{
		UID:         strings.TrimSpace(item.UID),
		MessageID:   normalizeMessageIDLocal(rawHeaders.Get("Message-ID"), rawHeaders.Get("Message-Id")),
		DateHeader:  strings.TrimSpace(rawHeaders.Get("Date")),
		Subject:     decodeHeaderValueLocal(rawHeaders.Get("Subject")),
		From:        decodeHeaderValueLocal(rawHeaders.Get("From")),
		FromAddress: extractMailAddressLocal(rawHeaders.Get("From")),
		Headers:     headers,
		RawHeaders:  rawHeaders,
		TextBody:    rawContent,
		RawContent:  rawContent,
		ProcessedAt: now,
	}
	result.ReceivedAt = parseBestReceivedAtLocal(result.DateHeader, rawHeaders, headers)
	if item.ID > 0 && result.UID == "" {
		result.UID = strconv.Itoa(item.ID)
	}
	result = finalizeFetchedMailLocal(result, now)
	rawJSON, err := json.Marshal(localMailRawPayload{
		Headers: result.Headers,
		Content: firstNonEmptyLocal(result.TextBody, result.RawContent),
	})
	if err == nil {
		result.RawJSON = string(rawJSON)
	}
	return result
}

func rawHeadersLocal(raw []byte) mail.Header {
	reader := textproto.NewReader(bufio.NewReader(bytes.NewReader(raw)))
	header, err := reader.ReadMIMEHeader()
	if err != nil {
		return mail.Header{}
	}
	return mail.Header(header)
}

func parseFetchedMailLocal(item localRawMail, downloadDir string, now time.Time) (emailsvc.FetchedMail, error) {
	message.CharsetReader = newCharsetReaderLocal
	entity, err := message.Read(bytes.NewReader(item.Raw))
	if err != nil && !message.IsUnknownCharset(err) {
		return emailsvc.FetchedMail{}, err
	}
	if entity == nil {
		msg, mailErr := mail.ReadMessage(bytes.NewReader(item.Raw))
		if mailErr != nil {
			return emailsvc.FetchedMail{}, mailErr
		}
		return parseFetchedMailFallbackLocal(item, msg, downloadDir, now)
	}

	result := emailsvc.FetchedMail{
		UID:         strings.TrimSpace(item.UID),
		RawContent:  toUTF8StringLocal(item.Raw),
		RawHeaders:  mail.Header{},
		ProcessedAt: now,
	}
	if item.ID > 0 && result.UID == "" {
		result.UID = strconv.Itoa(item.ID)
	}
	headers := collectEntityHeadersLocal(entity.Header)
	result.Headers = headers
	result.RawHeaders = headersToMailHeaderLocal(headers)
	result.DateHeader = headerValueLocal(headers, "Date")
	result.ReceivedAt = parseBestReceivedAtLocal(result.DateHeader, result.RawHeaders, headers)
	result.Subject = decodeHeaderValueLocal(headerValueLocal(headers, "Subject"))
	result.From = decodeHeaderValueLocal(headerValueLocal(headers, "From"))
	result.FromAddress = extractMailAddressLocal(result.From)
	result.MessageID = normalizeMessageIDLocal(headerValueLocal(headers, "Message-ID"), headerValueLocal(headers, "Message-Id"))
	if err := consumeEntityLocal(entity, downloadDir, &result, now); err != nil {
		return emailsvc.FetchedMail{}, err
	}
	result = finalizeFetchedMailLocal(result, now)
	rawJSON, err := json.Marshal(localMailRawPayload{
		Headers: result.Headers,
		Content: firstNonEmptyLocal(result.TextBody, htmlToTextLocal(result.HTMLBody), result.RawContent),
	})
	if err != nil {
		return emailsvc.FetchedMail{}, err
	}
	result.RawJSON = string(rawJSON)
	return result, nil
}

func parseFetchedMailFallbackLocal(item localRawMail, msg *mail.Message, downloadDir string, now time.Time) (emailsvc.FetchedMail, error) {
	header := msg.Header
	body, err := io.ReadAll(msg.Body)
	if err != nil {
		return emailsvc.FetchedMail{}, err
	}

	dateHeader := strings.TrimSpace(header.Get("Date"))
	subject := decodeHeaderValueLocal(header.Get("Subject"))
	from := decodeHeaderValueLocal(header.Get("From"))
	headers := mailHeaderToJSONLocal(header)

	result := emailsvc.FetchedMail{
		UID:         strings.TrimSpace(item.UID),
		MessageID:   normalizeMessageIDLocal(header.Get("Message-ID"), header.Get("Message-Id")),
		DateHeader:  dateHeader,
		Subject:     subject,
		From:        from,
		FromAddress: extractMailAddressLocal(from),
		RawContent:  toUTF8StringLocal(item.Raw),
		ProcessedAt: now,
		RawHeaders:  header,
		Headers:     headers,
	}
	result.ReceivedAt = parseBestReceivedAtLocal(dateHeader, header, headers)
	if item.ID > 0 && result.UID == "" {
		result.UID = strconv.Itoa(item.ID)
	}

	mediaType, params, _ := mime.ParseMediaType(header.Get("Content-Type"))
	if strings.HasPrefix(strings.ToLower(mediaType), "multipart/") {
		if err := walkMultipartLocal(bytes.NewReader(body), params["boundary"], downloadDir, &result, now); err != nil {
			return emailsvc.FetchedMail{}, err
		}
	} else if strings.Contains(strings.ToLower(mediaType), "text/html") {
		result.HTMLBody = decodeBodyWithCharsetLocal(body, params["charset"])
	} else {
		result.TextBody = strings.TrimSpace(decodeBodyWithCharsetLocal(body, params["charset"]))
	}

	result = finalizeFetchedMailLocal(result, now)
	rawJSON, err := json.Marshal(localMailRawPayload{
		Headers: result.Headers,
		Content: firstNonEmptyLocal(result.TextBody, htmlToTextLocal(result.HTMLBody), result.RawContent),
	})
	if err != nil {
		return emailsvc.FetchedMail{}, err
	}
	result.RawJSON = string(rawJSON)
	return result, nil
}

type localMailRawPayload struct {
	Headers []map[string]string `json:"headers"`
	Content string              `json:"content"`
}

func consumeEntityLocal(entity *message.Entity, downloadDir string, result *emailsvc.FetchedMail, now time.Time) error {
	if entity == nil || result == nil {
		return nil
	}
	mediaType, params, _ := entity.Header.ContentType()
	switch {
	case strings.HasPrefix(strings.ToLower(mediaType), "multipart/"):
		reader := entity.MultipartReader()
		if reader == nil {
			return walkMultipartLocal(entity.Body, params["boundary"], downloadDir, result, now)
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
			if err := consumeEntityLocal(part, downloadDir, result, now); err != nil {
				_ = closeEntityBodyLocal(part)
				return err
			}
			_ = closeEntityBodyLocal(part)
		}
	case strings.EqualFold(strings.ToLower(mediaType), "message/rfc822"):
		childBytes, err := io.ReadAll(entity.Body)
		if err != nil {
			return err
		}
		child, err := parseFetchedMailLocal(localRawMail{UID: result.UID, Raw: childBytes}, downloadDir, now)
		if err != nil {
			return err
		}
		mergeFetchedMailLocal(result, child)
		return nil
	case strings.HasPrefix(strings.ToLower(mediaType), "text/plain"):
		data, err := io.ReadAll(entity.Body)
		if err != nil {
			return err
		}
		result.TextBody = appendContentLocal(result.TextBody, strings.TrimSpace(decodeBodyWithCharsetLocal(data, params["charset"])))
		return nil
	case strings.HasPrefix(strings.ToLower(mediaType), "text/html"):
		data, err := io.ReadAll(entity.Body)
		if err != nil {
			return err
		}
		result.HTMLBody = appendContentLocal(result.HTMLBody, strings.TrimSpace(decodeBodyWithCharsetLocal(data, params["charset"])))
		return nil
	default:
		data, err := io.ReadAll(entity.Body)
		if err != nil {
			return err
		}
		filename := fileNameFromEntityLocal(entity)
		contentID := strings.Trim(strings.TrimSpace(headerFieldLocal(entity.Header, "Content-Id")), "<>")
		disposition := strings.ToLower(strings.TrimSpace(headerFieldLocal(entity.Header, "Content-Disposition")))
		if disposition == "" && !strings.HasPrefix(strings.ToLower(mediaType), "image/") && contentID == "" && filename == "" {
			return nil
		}
		path, kind, key, err := persistArtifactLocal(downloadDir, data, mediaType, filename, contentID)
		if err != nil {
			log.Printf("[email] stage=artifact-skip err=%s media_type=%s filename=%s content_id=%s", formatErrorForLogLocal(err), mediaType, filename, contentID)
			return nil
		}
		if strings.TrimSpace(path) == "" {
			return nil
		}
		result.Artifacts = append(result.Artifacts, path)
		result.Attachments = append(result.Attachments, emailsvc.MailArtifact{Kind: kind, Key: key, Path: path})
		return nil
	}
}

func closeEntityBodyLocal(entity *message.Entity) error {
	if entity == nil {
		return nil
	}
	if closer, ok := entity.Body.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func walkMultipartLocal(reader io.Reader, boundary, downloadDir string, result *emailsvc.FetchedMail, now time.Time) error {
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
		if err := consumeMimePartLocal(part, downloadDir, result, now); err != nil {
			part.Close()
			return err
		}
		part.Close()
	}
}

func consumeMimePartLocal(part *multipart.Part, downloadDir string, result *emailsvc.FetchedMail, now time.Time) error {
	contentType := part.Header.Get("Content-Type")
	mediaType, params, _ := mime.ParseMediaType(contentType)
	data, err := io.ReadAll(part)
	if err != nil {
		return err
	}
	switch {
	case strings.HasPrefix(strings.ToLower(mediaType), "multipart/"):
		return walkMultipartLocal(bytes.NewReader(data), params["boundary"], downloadDir, result, now)
	case strings.EqualFold(strings.ToLower(mediaType), "message/rfc822"):
		child, err := parseFetchedMailLocal(localRawMail{UID: result.UID, Raw: data}, downloadDir, now)
		if err != nil {
			return err
		}
		mergeFetchedMailLocal(result, child)
		return nil
	case strings.HasPrefix(strings.ToLower(mediaType), "text/plain"):
		result.TextBody = appendContentLocal(result.TextBody, strings.TrimSpace(decodeBodyWithCharsetLocal(data, params["charset"])))
		return nil
	case strings.HasPrefix(strings.ToLower(mediaType), "text/html"):
		result.HTMLBody = appendContentLocal(result.HTMLBody, strings.TrimSpace(decodeBodyWithCharsetLocal(data, params["charset"])))
		return nil
	default:
		filename := fileNameFromPartLocal(part)
		contentID := strings.Trim(strings.TrimSpace(part.Header.Get("Content-ID")), "<>")
		disposition, _, _ := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		if disposition == "" && !strings.HasPrefix(strings.ToLower(mediaType), "image/") && contentID == "" && filename == "" {
			return nil
		}
		path, kind, key, err := persistArtifactLocal(downloadDir, data, mediaType, filename, contentID)
		if err != nil {
			log.Printf("[email] stage=artifact-skip err=%s media_type=%s filename=%s content_id=%s", formatErrorForLogLocal(err), mediaType, filename, contentID)
			return nil
		}
		if strings.TrimSpace(path) == "" {
			return nil
		}
		result.Artifacts = append(result.Artifacts, path)
		result.Attachments = append(result.Attachments, emailsvc.MailArtifact{Kind: kind, Key: key, Path: path})
		return nil
	}
}

func mergeFetchedMailLocal(dst *emailsvc.FetchedMail, child emailsvc.FetchedMail) {
	if dst == nil {
		return
	}
	dst.TextBody = appendContentLocal(dst.TextBody, child.TextBody)
	dst.HTMLBody = appendContentLocal(dst.HTMLBody, child.HTMLBody)
	dst.Artifacts = append(dst.Artifacts, child.Artifacts...)
	dst.Attachments = append(dst.Attachments, child.Attachments...)
}

func finalizeFetchedMailLocal(item emailsvc.FetchedMail, now time.Time) emailsvc.FetchedMail {
	item.TextBody = strings.TrimSpace(item.TextBody)
	item.HTMLBody = strings.TrimSpace(item.HTMLBody)
	item.Artifacts = dedupeStringsLocal(item.Artifacts)
	if item.ReceivedAt.IsZero() {
		item.ReceivedAt = now
	}
	if strings.TrimSpace(item.FromAddress) == "" {
		item.FromAddress = extractMailAddressLocal(item.From)
	}
	return item
}

func effectiveMailTimeLocal(item emailsvc.FetchedMail, now time.Time) time.Time {
	if !item.ReceivedAt.IsZero() {
		return item.ReceivedAt
	}
	if !item.ProcessedAt.IsZero() {
		return item.ProcessedAt
	}
	return now
}

func parseBestReceivedAtLocal(dateHeader string, header mail.Header, headers []map[string]string) time.Time {
	best := parseMailDateLocal(dateHeader)
	for _, value := range receivedHeaderValuesLocal(header, headers) {
		if parsed := parseReceivedHeaderTimeLocal(value); parsed.After(best) {
			best = parsed
		}
	}
	return best
}

func parseMailDateLocal(value string) time.Time {
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

func receivedHeaderValuesLocal(header mail.Header, headers []map[string]string) []string {
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

func parseReceivedHeaderTimeLocal(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	if index := strings.LastIndex(value, ";"); index >= 0 {
		if parsed := parseMailDateLocal(value[index+1:]); !parsed.IsZero() {
			return parsed
		}
	}
	return parseMailDateLocal(value)
}

func appendContentLocal(existing, next string) string {
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

func persistArtifactLocal(downloadDir string, data []byte, mediaType, filename, keyHint string) (string, string, string, error) {
	if len(data) == 0 {
		return "", "", "", nil
	}
	if strings.TrimSpace(downloadDir) == "" {
		downloadDir = "email_artifacts"
	}
	if err := ensureParentDirLocal(filepath.Join(downloadDir, ".keep")); err != nil {
		return "", "", "", err
	}

	kind := "file"
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(mediaType)), "image/") {
		kind = "image"
	}
	key := sanitizeFileNameLocal(firstNonEmptyLocal(keyHint, filename))
	if key == "" {
		key = md5TextLocal(string(data))
	}
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = extensionByMediaTypeLocal(mediaType)
	}
	target := filepath.Join(downloadDir, sanitizeFileNameLocal(kind+"_key_"+key)+ext)
	if abs, err := filepath.Abs(target); err == nil {
		target = abs
	}
	if err := ensureParentDirLocal(target); err != nil {
		return "", "", "", err
	}
	if err := osWriteFileLocal(target, data, 0o644); err != nil {
		return "", "", "", err
	}
	return target, kind, key, nil
}

func osWriteFileLocal(name string, data []byte, perm uint32) error {
	return os.WriteFile(name, data, os.FileMode(perm))
}

func fileNameFromPartLocal(part *multipart.Part) string {
	if part == nil {
		return ""
	}
	if name := strings.TrimSpace(part.FileName()); name != "" {
		return sanitizeFileNameLocal(name)
	}
	if _, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition")); err == nil {
		if name := strings.TrimSpace(params["filename"]); name != "" {
			return sanitizeFileNameLocal(name)
		}
	}
	if _, params, err := mime.ParseMediaType(part.Header.Get("Content-Type")); err == nil {
		if name := strings.TrimSpace(params["name"]); name != "" {
			return sanitizeFileNameLocal(name)
		}
	}
	return ""
}

func fileNameFromEntityLocal(entity *message.Entity) string {
	if entity == nil {
		return ""
	}
	if disposition, params, err := entity.Header.ContentDisposition(); err == nil {
		if disposition != "" && strings.TrimSpace(params["filename"]) != "" {
			return sanitizeFileNameLocal(params["filename"])
		}
	}
	_, params, _ := entity.Header.ContentType()
	return sanitizeFileNameLocal(params["name"])
}

func extensionByMediaTypeLocal(mediaType string) string {
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

func sanitizeFileNameLocal(name string) string {
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

func md5TextLocal(input string) string {
	sum := md5.Sum([]byte(input))
	return hex.EncodeToString(sum[:])
}

func decodeBodyWithCharsetLocal(data []byte, charsetName string) string {
	if utf8ValidLocal(string(bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}))) {
		return toUTF8StringLocal(data)
	}
	charsetName = strings.TrimSpace(charsetName)
	if charsetName == "" {
		return toUTF8StringLocal(data)
	}
	reader, err := newCharsetReaderLocal(charsetName, bytes.NewReader(data))
	if err != nil {
		return toUTF8StringLocal(data)
	}
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return toUTF8StringLocal(data)
	}
	return toUTF8StringLocal(decoded)
}

func toUTF8StringLocal(data []byte) string {
	text := strings.TrimSpace(string(bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})))
	if utf8ValidLocal(text) {
		return text
	}
	return strings.ToValidUTF8(text, "")
}

func utf8ValidLocal(s string) bool {
	return strings.ToValidUTF8(s, "\uFFFD") == s
}

func newCharsetReaderLocal(charsetName string, input io.Reader) (io.Reader, error) {
	enc, err := htmlindex.Get(strings.TrimSpace(charsetName))
	if err != nil {
		return input, nil
	}
	return transform.NewReader(input, enc.NewDecoder()), nil
}

func formatErrorForLogLocal(err error) string {
	if err == nil {
		return ""
	}
	return decodePossiblyEncodedTextLocal(err.Error())
}

func decodePossiblyEncodedTextLocal(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if utf8ValidLocal(text) {
		return text
	}
	raw := []byte(text)
	charsetName, ok := detectTextCharsetLocal(raw)
	if ok {
		decoded, ok := decodeBytesWithCharsetNameLocal(raw, charsetName)
		if ok {
			return decoded
		}
	}
	return strings.TrimSpace(strings.ToValidUTF8(text, ""))
}

func detectTextCharsetLocal(data []byte) (string, bool) {
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

func decodeBytesWithCharsetNameLocal(data []byte, charsetName string) (string, bool) {
	charsetName = normalizeCharsetLabelLocal(charsetName)
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
	text := strings.TrimSpace(toUTF8StringLocal(decoded))
	return text, text != ""
}

func normalizeCharsetLabelLocal(charsetName string) string {
	charsetName = strings.TrimSpace(strings.ToLower(charsetName))
	if charsetName == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range charsetName {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func mailHeaderToJSONLocal(header mail.Header) []map[string]string {
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
				"value": decodeHeaderValueLocal(value),
			})
		}
	}
	return out
}

func collectEntityHeadersLocal(header message.Header) []map[string]string {
	out := make([]map[string]string, 0)
	fields := header.Fields()
	for fields.Next() {
		out = append(out, map[string]string{
			"name":  fields.Key(),
			"value": decodeHeaderValueLocal(fields.Value()),
		})
	}
	return out
}

func headersToMailHeaderLocal(headers []map[string]string) mail.Header {
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

func headerValueLocal(headers []map[string]string, key string) string {
	for _, item := range headers {
		if strings.EqualFold(strings.TrimSpace(item["name"]), strings.TrimSpace(key)) {
			return strings.TrimSpace(item["value"])
		}
	}
	return ""
}

func headerFieldLocal(header message.Header, key string) string {
	fields := header.FieldsByKey(key)
	if fields.Next() {
		return fields.Value()
	}
	return ""
}

func normalizeMessageIDLocal(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

type localTLSDialer struct {
	ctx  context.Context
	host string
}

func (d *localTLSDialer) Dial(network, address string) (net.Conn, error) {
	return dialTLSWithFallbackLocal(d.ctx, strings.TrimSpace(d.host), network, address)
}

func dialTLSWithFallbackLocal(ctx context.Context, host, network, address string) (net.Conn, error) {
	netDialer := &net.Dialer{Timeout: 10 * time.Second}
	tlsDialer := &tls.Dialer{
		NetDialer: netDialer,
		Config: &tls.Config{
			ServerName: host,
		},
	}
	conn, err := tlsDialer.DialContext(ctx, network, address)
	if err == nil {
		return wrapPOP3ConnTimeoutLocal(conn), nil
	}

	targets := fallbackDialTargetsLocal(ctx, host, address)
	if len(targets) == 0 {
		return nil, err
	}
	lastErr := err
	for _, target := range targets {
		conn, dialErr := tlsDialer.DialContext(ctx, network, target)
		if dialErr == nil {
			return wrapPOP3ConnTimeoutLocal(conn), nil
		}
		lastErr = dialErr
	}
	return nil, lastErr
}

type deadlineConnLocal struct {
	net.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func wrapPOP3ConnTimeoutLocal(conn net.Conn) net.Conn {
	if conn == nil {
		return nil
	}
	return &deadlineConnLocal{
		Conn:         conn,
		readTimeout:  pop3CommandTimeoutLocal,
		writeTimeout: pop3CommandTimeoutLocal,
	}
}

func (c *deadlineConnLocal) Read(p []byte) (int, error) {
	if c == nil || c.Conn == nil {
		return 0, net.ErrClosed
	}
	if c.readTimeout > 0 {
		if err := c.Conn.SetReadDeadline(time.Now().Add(c.readTimeout)); err != nil {
			return 0, err
		}
	}
	return c.Conn.Read(p)
}

func (c *deadlineConnLocal) Write(p []byte) (int, error) {
	if c == nil || c.Conn == nil {
		return 0, net.ErrClosed
	}
	if c.writeTimeout > 0 {
		if err := c.Conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
			return 0, err
		}
	}
	return c.Conn.Write(p)
}

func fallbackDialTargetsLocal(ctx context.Context, host, address string) []string {
	host = strings.TrimSpace(host)
	if host == "" || net.ParseIP(host) != nil {
		return nil
	}
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil
	}
	ips := lookupFallbackIPsLocal(ctx, host)
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		if !isUsableFallbackIPLocal(ip) {
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

func lookupFallbackIPsLocal(ctx context.Context, host string) []net.IP {
	var out []net.IP
	for _, dnsAddr := range fallbackDNSAddrLocal {
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

func isUsableFallbackIPLocal(ip net.IP) bool {
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
