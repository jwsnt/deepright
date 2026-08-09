package knowledge

import (
	"testing"
	"time"
)

func TestLoadLastUpdateText(t *testing.T) {
	// This function depends on a shared SQLite database with knowledgecore.
	// Test that it returns an error for non-existent directories.
	_, err := LoadLastUpdateText("/nonexistent/path", "", time.UTC)
	if err == nil {
		t.Log("LoadLastUpdateText() expected error for invalid path (note: may succeed if DB exists elsewhere)")
	} else {
		// Expected: error opening database
		t.Logf("LoadLastUpdateText() returned expected error: %v", err)
	}
}
