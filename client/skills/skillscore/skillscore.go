package skillscore

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type compatibilityValue string

func (c *compatibilityValue) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var value string
		if err := node.Decode(&value); err != nil {
			return err
		}
		*c = compatibilityValue(strings.TrimSpace(value))
		return nil
	case yaml.SequenceNode:
		var items []string
		if err := node.Decode(&items); err != nil {
			return err
		}
		normalized := make([]string, 0, len(items))
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			normalized = append(normalized, item)
		}
		*c = compatibilityValue(strings.Join(normalized, "; "))
		return nil
	default:
		return fmt.Errorf("compatibility 字段无效")
	}
}

type Skill struct {
	Name          string         `json:"name" yaml:"name"`
	Location      string         `json:"location" yaml:"-"`
	Description   string         `json:"description" yaml:"description"`
	License       string         `json:"license,omitempty" yaml:"license"`
	Compatibility compatibilityValue `json:"compatibility,omitempty" yaml:"compatibility"`
	Metadata      map[string]any `json:"metadata,omitempty" yaml:"metadata"`
	AllowedTools  string         `json:"allowed-tools,omitempty" yaml:"allowed-tools"`
}

type SkillWarning struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
	Time   int64  `json:"time"`
}

var (
	nameRe        = regexp.MustCompile(`^[a-zA-Z0-9_]([a-zA-Z0-9_-]*[a-zA-Z0-9_])?$`)
	frontMatterRe = regexp.MustCompile(`(?s)^-{3,}\n(.*?)\n-{3,}`)
)

func parseFrontMatter(content string) (string, bool) {
	m := frontMatterRe.FindStringSubmatch(content)
	if m == nil {
		return "", false
	}
	return m[1], true
}

func validateSkill(s *Skill) error {
	if s.Name == "" || len(s.Name) > 64 || !nameRe.MatchString(s.Name) {
		return fmt.Errorf("name 字段无效")
	}
	desc := strings.TrimSpace(s.Description)
	if desc == "" || len(desc) > 1024 {
		return fmt.Errorf("description 字段无效")
	}
	if len(s.Compatibility) > 500 {
		return fmt.Errorf("compatibility 字段无效")
	}
	return nil
}

func parseSkill(path string) (Skill, error) {
	var s Skill
	data, err := os.ReadFile(path)
	if err != nil {
		return s, fmt.Errorf("读取文件失败: %w", err)
	}
	yamlStr, ok := parseFrontMatter(string(data))
	if !ok {
		return s, fmt.Errorf("缺少 YAML front matter")
	}
	if err := yaml.Unmarshal([]byte(yamlStr), &s); err != nil {
		return s, fmt.Errorf("YAML 解析失败: %w", err)
	}
	if err := validateSkill(&s); err != nil {
		return s, err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return s, fmt.Errorf("解析绝对路径失败: %w", err)
	}
	s.Location = absPath
	return s, nil
}

func scan(root string, allowPlainSkill bool) ([]Skill, []SkillWarning, error) {
	skillMap := make(map[string]Skill)
	var order []string
	var warnings []SkillWarning

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() != "SKILL.md" && (!allowPlainSkill || d.Name() != "SKILL") {
			return nil
		}
		s, parseErr := parseSkill(path)
		if parseErr != nil {
			if d.Name() == "SKILL.md" {
				absPath, absErr := filepath.Abs(path)
				if absErr != nil {
					absPath = path
				}
				warnings = append(warnings, SkillWarning{
					Path:   absPath,
					Reason: parseErr.Error(),
					Time:   time.Now().Unix(),
				})
			}
			return nil
		}
		if _, exists := skillMap[s.Name]; !exists {
			order = append(order, s.Name)
		}
		skillMap[s.Name] = s
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	result := make([]Skill, 0, len(skillMap))
	for _, name := range order {
		result = append(result, skillMap[name])
	}
	return result, warnings, nil
}

func Scan(root string) ([]Skill, error) {
	skills, _, err := scan(root, false)
	return skills, err
}

func ScanAgentSkills(root string) ([]Skill, error) {
	skills, _, err := scan(root, true)
	return skills, err
}

func ScanWarnings(root string) ([]SkillWarning, error) {
	_, warnings, err := scan(root, false)
	return warnings, err
}

func ScanWithWarnings(root string) ([]Skill, []SkillWarning, error) {
	return scan(root, false)
}

func GetOutputJSON(root string, ttl time.Duration) ([]byte, error) {
	skills, err := Scan(root)
	if err != nil {
		return nil, err
	}
	out, err := json.MarshalIndent(skills, "", "  ")
	if err != nil {
		return nil, err
	}
	_ = ttl
	return out, nil
}

func FlushCache() {
}
