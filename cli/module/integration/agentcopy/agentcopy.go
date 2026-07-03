package agentcopy

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	integrationagentarchive "integration/agentarchive"
)

type Result struct {
	SourceAgentID string   `json:"sourceAgentId"`
	TargetAgentID string   `json:"targetAgentId"`
	AgentPath     string   `json:"agentPath"`
	KnowledgePath string   `json:"knowledgePath,omitempty"`
	Copied        []string `json:"copied,omitempty"`
}

type managedEntry struct {
	RelPath string
	IsDir   bool
}

var managedAgentEntries = []managedEntry{
	{RelPath: "app", IsDir: true},
	{RelPath: "data", IsDir: true},
	{RelPath: "skills", IsDir: true},
	{RelPath: "SOUL.md", IsDir: false},
	{RelPath: "USER.md", IsDir: false},
}

var managedKnowledgeEntries = []string{"Knowledge.md", "knowledge.md"}

func Copy(agentRootDir, knowledgeRootDir, sourceAgentID, targetAgentID string) (Result, error) {
	sourceAgentPath, err := resolveAgentDir(agentRootDir, sourceAgentID, "source")
	if err != nil {
		return Result{}, err
	}
	targetAgentPath, err := resolveAgentDir(agentRootDir, targetAgentID, "target")
	if err != nil {
		return Result{}, err
	}
	if samePath(sourceAgentPath, targetAgentPath) {
		return Result{}, fmt.Errorf("source and target agent must be different")
	}

	result := Result{
		SourceAgentID: strings.TrimSpace(sourceAgentID),
		TargetAgentID: strings.TrimSpace(targetAgentID),
		AgentPath:     targetAgentPath,
	}
	for _, entry := range managedAgentEntries {
		changed, err := syncManagedPath(
			filepath.Join(sourceAgentPath, entry.RelPath),
			filepath.Join(targetAgentPath, entry.RelPath),
			entry.IsDir,
		)
		if err != nil {
			return Result{}, fmt.Errorf("sync %s: %w", entry.RelPath, err)
		}
		if changed {
			result.Copied = append(result.Copied, entry.RelPath)
		}
	}
	knowledgeCopied, err := syncManagedKnowledgeFiles(sourceAgentPath, targetAgentPath)
	if err != nil {
		return Result{}, err
	}
	result.Copied = append(result.Copied, knowledgeCopied...)

	knowledgeRootDir = strings.TrimSpace(knowledgeRootDir)
	if knowledgeRootDir != "" {
		result.KnowledgePath = filepath.Join(knowledgeRootDir, result.TargetAgentID)
		changed, err := syncManagedPath(
			filepath.Join(knowledgeRootDir, result.SourceAgentID),
			result.KnowledgePath,
			true,
		)
		if err != nil {
			return Result{}, fmt.Errorf("sync knowledge: %w", err)
		}
		if changed {
			result.Copied = append(result.Copied, "knowledge")
		}
	}

	return result, nil
}

func syncManagedKnowledgeFiles(sourceAgentPath, targetAgentPath string) ([]string, error) {
	selectedName, err := selectManagedKnowledgeFile(sourceAgentPath)
	if err != nil {
		return nil, err
	}

	copied := make([]string, 0, len(managedKnowledgeEntries))
	if selectedName != "" {
		changed, err := syncManagedPath(
			filepath.Join(sourceAgentPath, selectedName),
			filepath.Join(targetAgentPath, selectedName),
			false,
		)
		if err != nil {
			return nil, fmt.Errorf("sync %s: %w", selectedName, err)
		}
		if changed {
			copied = append(copied, selectedName)
		}
	}

	selectedTargetPath := ""
	if selectedName != "" {
		selectedTargetPath = filepath.Join(targetAgentPath, selectedName)
	}
	for _, name := range managedKnowledgeEntries {
		if name == selectedName {
			continue
		}
		targetPath := filepath.Join(targetAgentPath, name)
		if selectedTargetPath != "" && sameExistingFile(selectedTargetPath, targetPath) {
			continue
		}
		removed, err := removePathIfExists(targetPath)
		if err != nil {
			return nil, fmt.Errorf("cleanup %s: %w", name, err)
		}
		if removed {
			copied = append(copied, name)
		}
	}
	return copied, nil
}

func selectManagedKnowledgeFile(sourceAgentPath string) (string, error) {
	for _, name := range managedKnowledgeEntries {
		info, err := os.Stat(filepath.Join(sourceAgentPath, name))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("stat %s: %w", name, err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("%s: source is not a file", name)
		}
		return name, nil
	}
	return "", nil
}

func resolveAgentDir(agentRootDir, agentID, label string) (string, error) {
	agentRootDir = strings.TrimSpace(agentRootDir)
	if agentRootDir == "" {
		return "", fmt.Errorf("%s agent root is required", label)
	}
	if err := integrationagentarchive.ValidateAgentID(agentID); err != nil {
		return "", fmt.Errorf("invalid %s agentId: %w", label, err)
	}
	agentPath := filepath.Join(agentRootDir, strings.TrimSpace(agentID))
	info, err := os.Stat(agentPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%s agent not found: %s", label, strings.TrimSpace(agentID))
		}
		return "", fmt.Errorf("stat %s agent: %w", label, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s agent is not a directory: %s", label, strings.TrimSpace(agentID))
	}
	return agentPath, nil
}

func syncManagedPath(src, dst string, expectDir bool) (bool, error) {
	srcInfo, err := os.Stat(src)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}
		dstExists, statErr := pathExists(dst)
		if statErr != nil {
			return false, statErr
		}
		if !dstExists {
			return false, nil
		}
		if err := os.RemoveAll(dst); err != nil && !os.IsNotExist(err) {
			return false, err
		}
		return true, nil
	}

	if expectDir && !srcInfo.IsDir() {
		return false, fmt.Errorf("source is not a directory")
	}
	if !expectDir && srcInfo.IsDir() {
		return false, fmt.Errorf("source is not a file")
	}

	if err := os.RemoveAll(dst); err != nil && !os.IsNotExist(err) {
		return false, err
	}
	if srcInfo.IsDir() {
		return true, copyDir(src, dst)
	}
	return true, copyFile(src, dst, srcInfo.Mode().Perm())
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func removePathIfExists(path string) (bool, error) {
	exists, err := pathExists(path)
	if err != nil || !exists {
		return false, err
	}
	if err := os.RemoveAll(path); err != nil && !os.IsNotExist(err) {
		return false, err
	}
	return true, nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(current string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, current)
		if err != nil {
			return err
		}
		target := dst
		if rel != "." {
			target = filepath.Join(dst, rel)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		return copyFile(current, target, info.Mode().Perm())
	})
}

func copyFile(src, dst string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func samePath(left, right string) bool {
	left = filepath.Clean(strings.TrimSpace(left))
	right = filepath.Clean(strings.TrimSpace(right))
	if left == "" || right == "" {
		return false
	}
	return left == right
}

func sameExistingFile(left, right string) bool {
	leftInfo, err := os.Stat(left)
	if err != nil {
		return false
	}
	rightInfo, err := os.Stat(right)
	if err != nil {
		return false
	}
	return os.SameFile(leftInfo, rightInfo)
}
