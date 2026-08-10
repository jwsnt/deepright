package filenamelookup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Scope string

const (
	ScopeWorkspace Scope = "workspace"
	ScopeTmp       Scope = "tmp"
	ScopeImages    Scope = "images"
)

type Match struct {
	Scope Scope  `json:"scope"`
	Path  string `json:"path"`
}

type searchScope struct {
	id           Scope
	root         string
	excludeRoots []string
	required     bool
}

func NormalizeCandidate(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.Trim(trimmed, "`\"'()[]{}<>（【】《》“”‘’")
	if trimmed == "" {
		return "", fmt.Errorf("candidate is required")
	}
	if strings.ContainsAny(trimmed, "/\\\r\n") {
		return "", fmt.Errorf("candidate must be a file name, not a path")
	}
	return trimmed, nil
}

func Lookup(workspaceRoot, rawCandidate string) ([]Match, error) {
	root := filepath.Clean(strings.TrimSpace(workspaceRoot))
	if root == "" {
		return nil, fmt.Errorf("workspace root is required")
	}
	if !filepath.IsAbs(root) {
		return nil, fmt.Errorf("workspace root must be absolute")
	}
	name, err := NormalizeCandidate(rawCandidate)
	if err != nil {
		return nil, err
	}
	tmpRoot := filepath.Join(root, "tmp")
	imagesRoot := filepath.Join(root, "images")
	scopes := []searchScope{
		{
			id:           ScopeWorkspace,
			root:         root,
			excludeRoots: []string{tmpRoot, imagesRoot},
			required:     true,
		},
		{
			id:   ScopeTmp,
			root: tmpRoot,
		},
		{
			id:   ScopeImages,
			root: imagesRoot,
		},
	}
	var matches []Match
	seen := make(map[string]struct{})
	for _, scope := range scopes {
		scopeMatches, err := lookupScope(name, scope)
		if err != nil {
			if !scope.required && os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, match := range scopeMatches {
			if _, exists := seen[match.Path]; exists {
				continue
			}
			seen[match.Path] = struct{}{}
			matches = append(matches, match)
		}
	}
	return matches, nil
}

func lookupScope(fileName string, scope searchScope) ([]Match, error) {
	root := filepath.Clean(scope.root)
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}
	var matches []Match
	queue := []string{root}
	visited := make(map[string]struct{})
	for len(queue) > 0 {
		dir := filepath.Clean(queue[0])
		queue = queue[1:]
		if _, exists := visited[dir]; exists {
			continue
		}
		visited[dir] = struct{}{}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			fullPath := filepath.Join(dir, entry.Name())
			if entry.IsDir() {
				if isExcludedScopeRoot(fullPath, scope.excludeRoots) {
					continue
				}
				queue = append(queue, fullPath)
				continue
			}
			if !strings.EqualFold(entry.Name(), fileName) {
				continue
			}
			matches = append(matches, Match{
				Scope: scope.id,
				Path:  filepath.Clean(fullPath),
			})
		}
	}
	return matches, nil
}

func isExcludedScopeRoot(path string, excludedRoots []string) bool {
	cleanPath := filepath.Clean(path)
	for _, excludedRoot := range excludedRoots {
		if cleanPath == filepath.Clean(excludedRoot) {
			return true
		}
	}
	return false
}
