package connectsvc

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMessageSnapshotQueriesMessagesAndSenders(t *testing.T) {
	svc := newMessageSnapshotTestService(t)
	base := time.Date(2026, time.July, 19, 10, 0, 0, 0, time.UTC)
	addMessageSnapshotTestRequest(t, svc, MessageSnapshot{Source: "feishu", Messages: []MessageSnapshotRecord{
		{MessageID: "om_1", SenderID: "ou_alpha", MessageType: "text", Content: "Refund application processed", SentAt: base.UnixMilli()},
		{MessageID: "om_img", SenderID: "ou_beta", MessageType: "image", Content: "[image]/tmp/a.png", SentAt: base.Add(time.Minute).UnixMilli()},
	}})
	addMessageSnapshotTestRequest(t, svc, MessageSnapshot{Source: "feishu", Messages: []MessageSnapshotRecord{
		{MessageID: "om_2", SenderID: "ou_alpha", MessageType: "text", Content: "Second REFUND update", SentAt: base.Add(2 * time.Minute).UnixMilli()},
	}})

	window := 72 * time.Hour
	senders, err := svc.ListMessageSenders("feishu", window, base.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(senders) != 2 {
		t.Fatalf("sender count = %d, want 2: %+v", len(senders), senders)
	}
	if senders[0].SenderID != "ou_alpha" || senders[0].LastMessageAt != "2026-07-19T10:02:00Z" {
		t.Fatalf("first sender = %+v", senders[0])
	}
	if senders[1].SenderID != "ou_beta" || senders[1].LastMessageAt != "2026-07-19T10:01:00Z" {
		t.Fatalf("second sender = %+v", senders[1])
	}

	page, err := svc.SearchMessageSnapshots(window, MessageSnapshotSearch{Source: "feishu", Query: "refund processed", Limit: 50}, base.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("search result = %+v", page)
	}
	if page.Items[0].MessageID != "om_1" || page.Items[0].SenderID != "ou_alpha" || page.Items[0].SentAt != "2026-07-19T10:00:00Z" {
		t.Fatalf("search item = %+v", page.Items[0])
	}

	imagePage, err := svc.SearchMessageSnapshots(window, MessageSnapshotSearch{Source: "feishu", Query: "image", Limit: 50}, base.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if imagePage.Total != 0 {
		t.Fatalf("pure image should not be searchable: %+v", imagePage)
	}

	allText, err := svc.SearchMessageSnapshots(window, MessageSnapshotSearch{Source: "feishu", Limit: 50}, base.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if allText.Total != 2 || len(allText.Items) != 2 || allText.Items[0].MessageID != "om_2" || allText.Items[1].MessageID != "om_1" {
		t.Fatalf("all text result = %+v", allText)
	}

	bySender, err := svc.SearchMessageSnapshots(window, MessageSnapshotSearch{Source: "feishu", SenderID: "ou_alpha", Limit: 50}, base.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if bySender.Total != 2 || len(bySender.Items) != 2 {
		t.Fatalf("sender result = %+v", bySender)
	}
}

func TestMessageSnapshotCLIRejectsInvalidArguments(t *testing.T) {
	svc := newMessageSnapshotTestService(t)
	for _, args := range [][]string{
		{"message-snapshot-search", "--source", "feishu", "--window-hours", "72", "--query", "refund", "--limit", "0"},
		{"message-snapshot-search", "--source", "feishu", "--window-hours", "72", "--query", "refund", "--offset", "-1"},
		{"message-snapshot-search", "--source", "feishu", "--window-hours", "0", "--query", "refund"},
	} {
		var stdout, stderr bytes.Buffer
		if code := RunCLIWithService(args[0], mustParseMessageSnapshotTestFlags(t, args[1:]), svc, &stdout, &stderr); code == 0 {
			t.Fatalf("%v unexpectedly succeeded: %s", args, stdout.String())
		}
	}
}

func TestParseMessageSnapshotSearchTerms(t *testing.T) {
	terms, err := ParseMessageSnapshotSearchTerms(`"refund application" processed`)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(terms); got != 2 || terms[0] != "refund application" || terms[1] != "processed" {
		t.Fatalf("terms = %#v", terms)
	}
	if _, err := ParseMessageSnapshotSearchTerms(`"refund`); err == nil {
		t.Fatal("expected unclosed quote error")
	}
}

func TestListRequestsReturnsNewestFirstAndPaginatesOlderRecords(t *testing.T) {
	svc := newMessageSnapshotTestService(t)
	for _, content := range []string{"first", "second", "third"} {
		if _, err := svc.AddRequest(RequestInput{Key: "feishu", ExternalID: "request-" + content, Content: content}); err != nil {
			t.Fatal(err)
		}
	}
	latest, err := svc.ListRequests(RequestFilter{Key: "feishu", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(latest) != 2 || latest[0].Content != "third" || latest[1].Content != "second" {
		t.Fatalf("latest = %+v", latest)
	}
	older, err := svc.ListRequests(RequestFilter{Key: "feishu", BeforeID: latest[1].ID, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(older) != 1 || older[0].Content != "first" {
		t.Fatalf("older = %+v", older)
	}
}

func newMessageSnapshotTestService(t *testing.T) *Service {
	t.Helper()
	root := t.TempDir()
	agentDir := filepath.Join(root, "agent")
	if err := os.MkdirAll(filepath.Join(agentDir, "A"), 0o755); err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(Options{DBPath: filepath.Join(root, "data"), AgentDir: agentDir})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Close() })
	if _, err := svc.db.Exec(`INSERT INTO token_store (model, token, updated_at) VALUES ('OpenAI', 'test-token', '2026-07-19T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateMeta(MetaInput{Key: "feishu", Meta: `{}`, Callback: "ignored", AgentID: "A", Model: "OpenAI"}); err != nil {
		t.Fatal(err)
	}
	return svc
}

func addMessageSnapshotTestRequest(t *testing.T, svc *Service, snapshot MessageSnapshot) {
	t.Helper()
	body, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddRequest(RequestInput{
		Key:             snapshot.Source,
		ExternalID:      "external-" + snapshot.Messages[0].MessageID,
		Content:         snapshot.Messages[0].Content,
		MessageSnapshot: string(body),
	}); err != nil {
		t.Fatal(err)
	}
}

func mustParseMessageSnapshotTestFlags(t *testing.T, args []string) map[string]string {
	t.Helper()
	flags, err := ParseFlags(args)
	if err != nil {
		t.Fatal(err)
	}
	return flags
}
