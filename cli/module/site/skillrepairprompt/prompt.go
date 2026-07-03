package skillrepairprompt

import (
	"fmt"
	"path/filepath"
	"strings"
)

const SpecURL = "https://agentskills.io/specification"

func NormalizeSkillPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", fmt.Errorf("path is required")
	}
	cleaned := filepath.Clean(trimmed)
	if !filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("path must be absolute")
	}
	if filepath.Base(cleaned) != "SKILL.md" {
		return "", fmt.Errorf("path must point to SKILL.md")
	}
	return filepath.ToSlash(cleaned), nil
}

func Build(path string) (string, error) {
	normalized, err := NormalizeSkillPath(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("参考[%s]修复%s的错误。", SpecURL, normalized), nil
}
