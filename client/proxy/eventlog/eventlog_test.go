package eventlog

import (
	"path/filepath"
	"testing"
)

func TestNormalizeContentPassthrough(t *testing.T) {
	tests := []struct {
		name    string
		logType int
		content string
		want    string
	}{
		{"non-pub type", TypeChatCompletionRequest, "hello", "hello"},
		{"empty content", TypeCLIPub, "", ""},
		{"not json", TypeCLIPub, "plain text", "plain text"},
		{"not a cliPubPayload", TypeCLIPub, `{"foo":"bar"}`, `{"foo":"bar"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeContent(tt.logType, tt.content)
			if got != tt.want {
				t.Errorf("NormalizeContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNowString(t *testing.T) {
	result := nowString()
	if len(result) != 23 {
		t.Errorf("nowString() length = %d, want 23 (ISO format with ms)", len(result))
	}
}

func TestOpenAndCloseStore(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer store.Close()

	if store.db == nil {
		t.Fatal("Open() returned store with nil db")
	}
}

func TestStoreInsertAndQuery(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer store.Close()

	entry := Entry{
		AgentID:   "test-agent",
		ChatID:    "test-chat",
		Content:   "test content",
		Type:      TypeChatCompletionRequest,
		CreatedAt: nowString(),
	}

	if err := store.Insert(entry); err != nil {
		t.Fatalf("Insert() error: %v", err)
	}

	entries, err := store.Query("test-agent", "test-chat", "1970-01-01", TypeChatCompletionRequest)
	if err != nil {
		t.Fatalf("Query() error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("Query() returned %d entries, want 1", len(entries))
	}
}

func TestStoreQueryWithTypeFilter(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer store.Close()

	entries := []Entry{
		{AgentID: "a1", ChatID: "c1", Content: "req", Type: TypeChatCompletionRequest, CreatedAt: nowString()},
		{AgentID: "a1", ChatID: "c1", Content: "resp", Type: TypeChatCompletionResponse, CreatedAt: nowString()},
	}
	for _, e := range entries {
		if err := store.Insert(e); err != nil {
			t.Fatal(err)
		}
	}

	// Query only request type
	results, err := store.Query("a1", "c1", "1970-01-01", TypeChatCompletionRequest)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results, want 1", len(results))
	}
}

func TestStoreQueryAll(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer store.Close()

	entries := []Entry{
		{AgentID: "a1", ChatID: "c1", Content: "first", Type: TypeCLIGet, CreatedAt: nowString()},
		{AgentID: "a1", ChatID: "c1", Content: "second", Type: TypeCLIPub, CreatedAt: nowString()},
	}
	for _, e := range entries {
		if err := store.Insert(e); err != nil {
			t.Fatal(err)
		}
	}

	results, err := store.QueryAll("a1", "c1")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("QueryAll() returned %d entries, want 2", len(results))
	}
}

func TestStoreNilSafety(t *testing.T) {
	var nilStore *Store

	// These should not panic
	nilStore.Close()
	nilStore.Insert(Entry{})
	_, err := nilStore.Query("", "", "")
	if err != nil {
		t.Errorf("nil store Query() error: %v", err)
	}
	_, err = nilStore.QueryAll("", "")
	if err != nil {
		t.Errorf("nil store QueryAll() error: %v", err)
	}
}

func TestNewLogger(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	logger, err := NewLogger(dbPath, 10)
	if err != nil {
		t.Fatalf("NewLogger() error: %v", err)
	}

	// Append some entries
	logger.Append(Entry{AgentID: "a1", ChatID: "c1", Content: "test", Type: TypeChatCompletionRequest})

	// Close first to flush the async goroutine
	if err := logger.Close(); err != nil {
		t.Fatal(err)
	}

	// Query from the underlying store directly
	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	entries, err := store.Query("a1", "c1", "1970-01-01", TypeChatCompletionRequest)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("got %d entries, want 1", len(entries))
	}
}

func TestLoggerNilSafety(t *testing.T) {
	var nilLogger *Logger

	// These should not panic
	nilLogger.Append(Entry{})
	nilLogger.Close()
	_, err := nilLogger.Query("", "", "")
	if err != nil {
		t.Errorf("nil logger Query() error: %v", err)
	}
	_, err = nilLogger.QueryAll("", "")
	if err != nil {
		t.Errorf("nil logger QueryAll() error: %v", err)
	}
}

func TestOpenInvalidPath(t *testing.T) {
	// Try opening in a non-writable location (simulate by empty path)
	store, err := Open("/nonexistent/dir/test.db")
	if err == nil {
		if store != nil {
			store.Close()
		}
	} else {
		// Expected error
		t.Logf("Open() with invalid path returned expected error: %v", err)
	}
}

func TestStoreQueryNoTypes(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	entries, err := store.Query("a1", "c1", "1970-01-01")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("Query() with no types returned %d entries, want 0", len(entries))
	}
}
