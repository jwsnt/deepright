package agentarchive

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Result struct {
	AgentID   string `json:"agentId"`
	AgentPath string `json:"agentPath"`
	Source    string `json:"source"`
}

func ValidateAgentID(agentID string) error {
	name := strings.TrimSpace(agentID)
	if name == "" {
		return fmt.Errorf("agent_id is required")
	}
	if strings.ContainsAny(name, " /\\:*?\"<>|") {
		return fmt.Errorf("agent_id contains invalid characters")
	}
	return nil
}

func Export(agentRootDir, agentID string, writer io.Writer) error {
	agentPath, err := resolveAgentPath(agentRootDir, agentID)
	if err != nil {
		return err
	}
	if writer == nil {
		return fmt.Errorf("export writer is nil")
	}

	zw := zip.NewWriter(writer)
	defer zw.Close()

	rootName := filepath.Base(agentPath)
	if _, err := zw.Create(rootName + "/"); err != nil {
		return fmt.Errorf("create zip root: %w", err)
	}

	return filepath.WalkDir(agentPath, func(current string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(agentPath, current)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		if shouldSkipExportDir(rel, d) {
			return filepath.SkipDir
		}

		zipName := filepath.ToSlash(filepath.Join(rootName, rel))
		if d.IsDir() {
			_, err := zw.Create(zipName + "/")
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = zipName
		header.Method = zip.Deflate
		entryWriter, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		file, err := os.Open(current)
		if err != nil {
			return err
		}
		_, err = io.Copy(entryWriter, file)
		closeErr := file.Close()
		if err != nil {
			return err
		}
		return closeErr
	})
}

func ImportZipFile(agentRootDir, zipPath string) (Result, error) {
	zipPath = strings.TrimSpace(zipPath)
	if zipPath == "" {
		return Result{}, fmt.Errorf("zip input is required")
	}
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return Result{}, fmt.Errorf("open zip: %w", err)
	}
	defer reader.Close()

	rootName, err := detectZipRoot(reader.File)
	if err != nil {
		return Result{}, err
	}
	targetPath := filepath.Join(strings.TrimSpace(agentRootDir), rootName)
	if info, err := os.Stat(targetPath); err == nil && info.IsDir() {
		return Result{}, fmt.Errorf("agent already exists: %s; please delete the existing agent first", rootName)
	}

	stagingDir, err := os.MkdirTemp(strings.TrimSpace(agentRootDir), ".agent-import-*")
	if err != nil {
		return Result{}, fmt.Errorf("create staging dir: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	for _, file := range reader.File {
		cleanName, err := cleanArchivePath(file.Name)
		if err != nil {
			return Result{}, err
		}
		if cleanName == "" {
			continue
		}

		target := filepath.Join(stagingDir, filepath.FromSlash(cleanName))
		if !isSubPath(target, stagingDir) {
			return Result{}, fmt.Errorf("archive entry escapes staging dir: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return Result{}, fmt.Errorf("create dir %s: %w", cleanName, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return Result{}, fmt.Errorf("create parent dir for %s: %w", cleanName, err)
		}

		rc, err := file.Open()
		if err != nil {
			return Result{}, fmt.Errorf("open zip entry %s: %w", cleanName, err)
		}
		mode := file.Mode()
		if mode == 0 {
			mode = 0o644
		}
		writeErr := writeFileFromReader(target, mode, rc)
		rc.Close()
		if writeErr != nil {
			return Result{}, fmt.Errorf("extract %s: %w", cleanName, writeErr)
		}
	}

	return finalizeImportedRoot(agentRootDir, filepath.Join(stagingDir, rootName), "zip")
}

func ImportDirectory(agentRootDir, sourceDir string) (Result, error) {
	sourceDir = strings.TrimSpace(sourceDir)
	if sourceDir == "" {
		return Result{}, fmt.Errorf("directory input is required")
	}
	info, err := os.Stat(sourceDir)
	if err != nil {
		return Result{}, fmt.Errorf("stat input directory: %w", err)
	}
	if !info.IsDir() {
		return Result{}, fmt.Errorf("input is not a directory: %s", sourceDir)
	}

	rootName := filepath.Base(filepath.Clean(sourceDir))
	if err := ValidateAgentID(rootName); err != nil {
		return Result{}, err
	}
	targetPath := filepath.Join(strings.TrimSpace(agentRootDir), rootName)
	if existing, err := os.Stat(targetPath); err == nil && existing.IsDir() {
		return Result{}, fmt.Errorf("agent already exists: %s; please delete the existing agent first", rootName)
	}

	stagingDir, err := os.MkdirTemp(strings.TrimSpace(agentRootDir), ".agent-import-*")
	if err != nil {
		return Result{}, fmt.Errorf("create staging dir: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	stagingRoot := filepath.Join(stagingDir, rootName)
	if err := copyDir(sourceDir, stagingRoot); err != nil {
		return Result{}, err
	}
	return finalizeImportedRoot(agentRootDir, stagingRoot, "directory")
}

func finalizeImportedRoot(agentRootDir, stagedRoot, source string) (Result, error) {
	agentRootDir = strings.TrimSpace(agentRootDir)
	stagedRoot = strings.TrimSpace(stagedRoot)
	info, err := os.Stat(stagedRoot)
	if err != nil {
		return Result{}, fmt.Errorf("staged agent not found: %w", err)
	}
	if !info.IsDir() {
		return Result{}, fmt.Errorf("staged agent is not a directory: %s", stagedRoot)
	}

	agentID := filepath.Base(filepath.Clean(stagedRoot))
	if err := ValidateAgentID(agentID); err != nil {
		return Result{}, err
	}

	targetPath := filepath.Join(agentRootDir, agentID)
	if existing, err := os.Stat(targetPath); err == nil && existing.IsDir() {
		return Result{}, fmt.Errorf("agent already exists: %s; please delete the existing agent first", agentID)
	}
	if err := os.Rename(stagedRoot, targetPath); err != nil {
		return Result{}, fmt.Errorf("move imported agent into place: %w", err)
	}
	return Result{
		AgentID:   agentID,
		AgentPath: targetPath,
		Source:    source,
	}, nil
}

func resolveAgentPath(agentRootDir, agentID string) (string, error) {
	if err := ValidateAgentID(agentID); err != nil {
		return "", err
	}
	root := strings.TrimSpace(agentRootDir)
	if root == "" {
		return "", fmt.Errorf("agent root dir is required")
	}
	agentPath := filepath.Join(root, strings.TrimSpace(agentID))
	info, err := os.Stat(agentPath)
	if err != nil {
		return "", fmt.Errorf("agent not found: %s", strings.TrimSpace(agentID))
	}
	if !info.IsDir() {
		return "", fmt.Errorf("agent is not a directory: %s", strings.TrimSpace(agentID))
	}
	return agentPath, nil
}

func shouldSkipExportDir(rel string, entry fs.DirEntry) bool {
	if !entry.IsDir() {
		return false
	}
	top := filepath.ToSlash(rel)
	if strings.Contains(top, "/") {
		top = strings.Split(top, "/")[0]
	}
	name := strings.ToLower(strings.TrimSpace(top))
	return strings.HasPrefix(name, "chrome") || name == "data" || name == "tmp"
}

func detectZipRoot(files []*zip.File) (string, error) {
	root := ""
	for _, file := range files {
		cleanName, err := cleanArchivePath(file.Name)
		if err != nil {
			return "", err
		}
		if cleanName == "" {
			continue
		}
		parts := strings.Split(cleanName, "/")
		if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
			return "", fmt.Errorf("zip archive must contain a single agent root directory")
		}
		if err := ValidateAgentID(parts[0]); err != nil {
			return "", err
		}
		if root == "" {
			root = parts[0]
			continue
		}
		if root != parts[0] {
			return "", fmt.Errorf("zip archive must contain exactly one top-level agent directory")
		}
	}
	if root == "" {
		return "", fmt.Errorf("zip archive is empty")
	}
	return root, nil
}

func cleanArchivePath(name string) (string, error) {
	normalized := strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	normalized = strings.TrimPrefix(normalized, "./")
	normalized = strings.TrimPrefix(normalized, "/")
	if normalized == "" {
		return "", nil
	}
	cleaned := path.Clean(normalized)
	if cleaned == "." {
		return "", nil
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("archive entry escapes destination: %s", name)
	}
	return cleaned, nil
}

func writeFileFromReader(target string, mode fs.FileMode, reader io.Reader) error {
	file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, reader)
	return err
}

func copyDir(sourceDir, targetDir string) error {
	return filepath.WalkDir(sourceDir, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(sourceDir, current)
		if err != nil {
			return err
		}
		target := targetDir
		if rel != "." {
			target = filepath.Join(targetDir, rel)
		}
		if !isSubPath(target, targetDir) {
			return fmt.Errorf("copied path escapes destination: %s", current)
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		sourceFile, err := os.Open(current)
		if err != nil {
			return err
		}
		writeErr := writeFileFromReader(target, info.Mode(), sourceFile)
		closeErr := sourceFile.Close()
		if writeErr != nil {
			return writeErr
		}
		return closeErr
	})
}

func isSubPath(targetPath, rootPath string) bool {
	relative, err := filepath.Rel(rootPath, targetPath)
	if err != nil {
		return false
	}
	if relative == "." {
		return true
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
