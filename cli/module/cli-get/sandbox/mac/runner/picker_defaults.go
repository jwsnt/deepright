package main

import (
	"os"
	"path/filepath"
	"strings"
)

func defaultMacPickerDirectory() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	home = filepath.Clean(strings.TrimSpace(home))
	if home == "" {
		return ""
	}
	documentsDir := filepath.Join(home, "Documents")
	info, err := os.Stat(documentsDir)
	if err != nil || !info.IsDir() {
		return ""
	}
	return documentsDir
}
