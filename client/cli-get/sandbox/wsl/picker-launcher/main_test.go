package main

import (
	"strings"
	"testing"
)

func TestWindowsPowerShellPickerScriptStartsFromCRoot(t *testing.T) {
	script := windowsPowerShellPickerScript(defaultWindowsPickerDirectory())
	if !strings.Contains(script, "$dialog.SelectedPath = 'C:\\'") {
		t.Fatalf("script = %q, want C root selected path", script)
	}
	if !strings.Contains(script, "System.Windows.Forms.FolderBrowserDialog") {
		t.Fatalf("script = %q, want FolderBrowserDialog", script)
	}
}
