package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	integrationBackupMaximumFileSize = int64(10_000_000)
	integrationBackupCommitMessage   = "deepright backup"
)

// integrationBackupBinaryExtensions deliberately classifies by file extension,
// rather than inspecting file contents. The policy is stable across macOS,
// Windows/WSL, and files that are being written while a backup runs.
var integrationBackupBinaryExtensions = map[string]struct{}{
	".7z": {}, ".a": {}, ".aac": {}, ".ai": {}, ".apk": {}, ".app": {}, ".ar": {}, ".avi": {},
	".avif": {}, ".bin": {}, ".bmp": {}, ".bz2": {}, ".cab": {}, ".class": {}, ".crx": {}, ".db": {},
	".deb": {}, ".dmg": {}, ".dll": {}, ".doc": {}, ".docm": {}, ".docx": {}, ".dot": {}, ".dotm": {},
	".dotx": {}, ".dylib": {}, ".ear": {}, ".eot": {}, ".epub": {}, ".exe": {}, ".flac": {}, ".flv": {},
	".fon": {}, ".gif": {}, ".gz": {}, ".heic": {}, ".ico": {}, ".img": {}, ".ipa": {}, ".iso": {},
	".jar": {}, ".jpeg": {}, ".jpg": {}, ".lz": {}, ".lz4": {}, ".lzh": {}, ".m4a": {}, ".m4v": {},
	".mkv": {}, ".mov": {}, ".mp3": {}, ".mp4": {}, ".mpeg": {}, ".mpg": {}, ".msi": {}, ".msix": {},
	".node": {}, ".o": {}, ".obj": {}, ".odp": {}, ".ods": {}, ".odt": {}, ".oga": {}, ".ogg": {},
	".ogv": {}, ".opus": {}, ".otf": {}, ".pak": {}, ".pdf": {}, ".pkg": {}, ".png": {}, ".ppt": {},
	".pptm": {}, ".pptx": {}, ".psd": {}, ".pyc": {}, ".pyo": {}, ".rar": {}, ".rpm": {}, ".rtf": {},
	".so": {}, ".sqlite": {}, ".sqlite3": {}, ".svg": {}, ".tar": {}, ".tif": {}, ".tiff": {}, ".ttc": {},
	".ttf": {}, ".wasm": {}, ".wav": {}, ".webm": {}, ".webp": {}, ".wma": {}, ".wmv": {}, ".woff": {},
	".woff2": {}, ".xls": {}, ".xlsb": {}, ".xlsm": {}, ".xlsx": {}, ".xlt": {}, ".xltm": {}, ".xltx": {},
	".xz": {}, ".z": {}, ".zip": {}, ".zst": {},
}

type integrationBackupConfig struct {
	SkillsInterval time.Duration
	AppInterval    time.Duration
}

type integrationBackupResult struct {
	Agent            string
	Directory        string
	Repository       string
	Committed        bool
	Tracked          int
	ExcludedBinary   int
	ExcludedTooLarge int
}

type integrationBackupSummary struct {
	AgentCount int
	Results    []integrationBackupResult
}

type integrationBackupStatusResponse struct {
	Status       int    `json:"status"`
	Directory    string `json:"directory"`
	LastBackupAt int64  `json:"lastBackupAt"`
}

func readIntegrationBackupConfig() (integrationBackupConfig, error) {
	raw, _, err := readIntegrationStartupConfigRaw()
	if err != nil {
		return integrationBackupConfig{}, fmt.Errorf("read config/config.json.backup: %w", err)
	}
	backupRaw, ok := raw["backup"]
	if !ok {
		return integrationBackupConfig{}, fmt.Errorf("config/config.json.backup is required")
	}
	backup, ok := backupRaw.(map[string]interface{})
	if !ok || backup == nil {
		return integrationBackupConfig{}, fmt.Errorf("config/config.json.backup must be an object")
	}
	skillsMinutes, err := integrationBackupPositiveMinutes(backup["skills"], "config/config.json.backup.skills")
	if err != nil {
		return integrationBackupConfig{}, err
	}
	appMinutes, err := integrationBackupPositiveMinutes(backup["app"], "config/config.json.backup.app")
	if err != nil {
		return integrationBackupConfig{}, err
	}
	return integrationBackupConfig{
		SkillsInterval: time.Duration(skillsMinutes) * time.Minute,
		AppInterval:    time.Duration(appMinutes) * time.Minute,
	}, nil
}

func integrationBackupPositiveMinutes(raw interface{}, label string) (int64, error) {
	value, ok := raw.(json.Number)
	if !ok {
		return 0, fmt.Errorf("%s must be a positive integer in minutes", label)
	}
	minutes, err := value.Int64()
	if err != nil || minutes <= 0 || minutes > int64((1<<63-1)/int64(time.Minute)) {
		return 0, fmt.Errorf("%s must be a positive integer in minutes", label)
	}
	return minutes, nil
}

// startIntegrationAgentBackups starts two independent non-blocking loops. A
// long-running Git operation can only delay the next run of its own directory;
// it cannot delay HTTP handling or the other directory's backup loop.
func startIntegrationAgentBackups(ctx context.Context, agentRoot string) {
	if ctx == nil || strings.TrimSpace(agentRoot) == "" {
		return
	}
	go func() {
		config, err := readIntegrationBackupConfig()
		if err != nil {
			log.Printf("[backup] disabled: %v", err)
			return
		}
		if _, err := exec.LookPath("git"); err != nil {
			log.Printf("[backup] disabled: git is unavailable: %v", err)
			return
		}
		startIntegrationAgentBackupLoop(ctx, agentRoot, "skills", config.SkillsInterval)
		startIntegrationAgentBackupLoop(ctx, agentRoot, "app", config.AppInterval)
	}()
}

func startIntegrationAgentBackupLoop(ctx context.Context, agentRoot, directory string, interval time.Duration) {
	go func() {
		run := func() {
			summary, err := backupIntegrationAgentDirectory(agentRoot, directory)
			if err != nil {
				log.Printf("[backup] scan failed directory=%s agent-dir=%s reason=%v", directory, agentRoot, err)
				return
			}
			for _, result := range summary.Results {
				if result.Committed {
					log.Printf("[backup] committed agent=%s directory=%s repository=%s tracked=%d excluded_binary=%d excluded_too_large=%d", result.Agent, result.Directory, result.Repository, result.Tracked, result.ExcludedBinary, result.ExcludedTooLarge)
				}
			}
		}
		run()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}

func backupIntegrationAgentDirectory(agentRoot, directory string) (integrationBackupSummary, error) {
	entries, err := os.ReadDir(agentRoot)
	if err != nil {
		return integrationBackupSummary{}, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	summary := integrationBackupSummary{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		summary.AgentCount++
		result, exists, err := backupIntegrationDirectory(filepath.Join(agentRoot, entry.Name()), entry.Name(), directory)
		if !exists {
			continue
		}
		if err != nil {
			log.Printf("[backup] failed agent=%s directory=%s repository=%s reason=%v", entry.Name(), directory, filepath.Join(agentRoot, entry.Name(), directory), err)
			continue
		}
		summary.Results = append(summary.Results, result)
	}
	return summary, nil
}

func backupIntegrationDirectory(workspace, agent, directory string) (integrationBackupResult, bool, error) {
	repository := filepath.Join(workspace, directory)
	result := integrationBackupResult{Agent: agent, Directory: directory, Repository: repository}
	info, err := os.Lstat(repository)
	if errors.Is(err, os.ErrNotExist) {
		return result, false, nil
	}
	if err != nil {
		return result, true, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return result, true, fmt.Errorf("backup target is not a directory")
	}
	if err := integrationBackupGit(repository, "init", "-q"); err != nil {
		return result, true, err
	}
	if err := integrationBackupEnsureLocalIdentity(repository); err != nil {
		return result, true, err
	}

	eligible, excludedBinary, excludedTooLarge, err := integrationBackupEligibleFiles(repository)
	if err != nil {
		return result, true, err
	}
	result.Tracked = len(eligible)
	result.ExcludedBinary = excludedBinary
	result.ExcludedTooLarge = excludedTooLarge
	if err := integrationBackupSyncIndex(repository, eligible); err != nil {
		return result, true, err
	}
	changed, err := integrationBackupHasStagedChanges(repository)
	if err != nil {
		return result, true, err
	}
	if !changed {
		return result, true, nil
	}
	if err := integrationBackupGit(repository, "-c", "commit.gpgSign=false", "commit", "--no-verify", "-m", integrationBackupCommitMessage); err != nil {
		return result, true, err
	}
	result.Committed = true
	return result, true, nil
}

func integrationBackupEligibleFiles(repository string) ([]string, int, int, error) {
	files := make([]string, 0)
	excludedBinary := 0
	excludedTooLarge := 0
	err := filepath.WalkDir(repository, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == repository {
			return nil
		}
		relative, err := filepath.Rel(repository, path)
		if err != nil {
			return err
		}
		if relative == ".git" || strings.HasPrefix(relative, ".git"+string(filepath.Separator)) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() && entry.Type()&fs.ModeSymlink == 0 {
			return nil
		}
		if integrationBackupIsBinaryExtension(relative) {
			excludedBinary++
			return nil
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Size() > integrationBackupMaximumFileSize {
			excludedTooLarge++
			return nil
		}
		files = append(files, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return nil, 0, 0, err
	}
	sort.Strings(files)
	return files, excludedBinary, excludedTooLarge, nil
}

func integrationBackupIsBinaryExtension(path string) bool {
	_, ok := integrationBackupBinaryExtensions[strings.ToLower(filepath.Ext(path))]
	return ok
}

func integrationBackupSyncIndex(repository string, eligible []string) error {
	eligibleSet := make(map[string]struct{}, len(eligible))
	for _, file := range eligible {
		eligibleSet[file] = struct{}{}
	}
	tracked, err := integrationBackupTrackedFiles(repository)
	if err != nil {
		return err
	}
	remove := make([]string, 0)
	for _, file := range tracked {
		if _, ok := eligibleSet[file]; !ok {
			remove = append(remove, file)
		}
	}
	if err := integrationBackupRunGitBatches(repository, []string{"rm", "--cached", "--ignore-unmatch", "--"}, remove); err != nil {
		return err
	}
	return integrationBackupRunGitBatches(repository, []string{"add", "-f", "--"}, eligible)
}

func integrationBackupTrackedFiles(repository string) ([]string, error) {
	output, err := integrationBackupGitOutput(repository, "ls-files", "-z")
	if err != nil {
		return nil, err
	}
	parts := bytes.Split(output, []byte{0})
	files := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) != 0 {
			files = append(files, string(part))
		}
	}
	return files, nil
}

func integrationBackupRunGitBatches(repository string, prefix, paths []string) error {
	const batchSize = 100
	for start := 0; start < len(paths); start += batchSize {
		end := start + batchSize
		if end > len(paths) {
			end = len(paths)
		}
		args := make([]string, 0, len(prefix)+end-start)
		args = append(args, prefix...)
		args = append(args, paths[start:end]...)
		if err := integrationBackupGit(repository, args...); err != nil {
			return err
		}
	}
	return nil
}

func integrationBackupEnsureLocalIdentity(repository string) error {
	for _, item := range []struct {
		key   string
		value string
	}{
		{key: "user.email", value: integrationDefaultGitUserEmail},
		{key: "user.name", value: integrationDefaultGitUserName},
	} {
		value, err := integrationBackupGitOutput(repository, "config", "--get", item.key)
		if err == nil && strings.TrimSpace(string(value)) != "" {
			continue
		}
		if err != nil && !integrationBackupGitExitCode(err, 1) {
			return err
		}
		if err := integrationBackupGit(repository, "config", "--local", item.key, item.value); err != nil {
			return err
		}
	}
	return nil
}

func integrationBackupHasStagedChanges(repository string) (bool, error) {
	_, err := integrationBackupGitOutput(repository, "diff", "--cached", "--quiet")
	if err == nil {
		return false, nil
	}
	if integrationBackupGitExitCode(err, 1) {
		return true, nil
	}
	return false, err
}

func integrationBackupGit(repository string, args ...string) error {
	_, err := integrationBackupGitOutput(repository, args...)
	return err
}

func integrationBackupGitOutput(repository string, args ...string) ([]byte, error) {
	commandArgs := make([]string, 0, len(args)+2)
	commandArgs = append(commandArgs, "-C", repository)
	commandArgs = append(commandArgs, args...)
	command := exec.Command("git", commandArgs...)
	output, err := command.CombinedOutput()
	if err == nil {
		return output, nil
	}
	message := strings.TrimSpace(string(output))
	if message == "" {
		return output, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return output, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, message)
}

func integrationBackupGitExitCode(err error, code int) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr) && exitErr.ExitCode() == code
}

// handleBackupStatus exposes only the last successful backup commit for the
// selected Agent's two managed repositories. It does not accept filesystem
// paths, so the virtual file system cannot query arbitrary Git directories.
func handleBackupStatus(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		directory := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("directory")))
		if directory != "skills" && directory != "app" {
			http.Error(w, "directory must be skills or app", http.StatusBadRequest)
			return
		}
		workspace, err := getWorkspaceByAgentID(cfg, r.URL.Query().Get("agentId"))
		if err != nil {
			http.Error(w, "Agent not found: "+err.Error(), http.StatusNotFound)
			return
		}
		lastBackupAt, found, err := integrationBackupLastCommitUnixMilli(filepath.Join(workspace, directory))
		if err != nil {
			http.Error(w, "Read backup status failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if !found {
			lastBackupAt = 0
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(integrationBackupStatusResponse{
			Status:       0,
			Directory:    directory,
			LastBackupAt: lastBackupAt,
		})
	}
}

// integrationBackupLastCommitUnixMilli returns the timestamp of the most
// recent commit written by this backup feature. It intentionally ignores
// regular Git commits that may predate the local backup repository.
func integrationBackupLastCommitUnixMilli(repository string) (int64, bool, error) {
	gitDir, err := os.Lstat(filepath.Join(repository, ".git"))
	if errors.Is(err, os.ErrNotExist) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if !gitDir.IsDir() {
		return 0, false, nil
	}
	output, err := integrationBackupGitOutput(repository, "log", "-1", "--format=%ct", "--grep=^"+integrationBackupCommitMessage+"$")
	if err != nil {
		if integrationBackupGitExitCode(err, 1) {
			return 0, false, nil
		}
		return 0, false, err
	}
	value := strings.TrimSpace(string(output))
	if value == "" {
		return 0, false, nil
	}
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil || seconds < 0 || seconds > (1<<63-1)/1000 {
		return 0, false, fmt.Errorf("invalid backup commit timestamp %q", value)
	}
	return seconds * 1000, true, nil
}
