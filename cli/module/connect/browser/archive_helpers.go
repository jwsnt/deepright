package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type browserArchiveProgress struct {
	Files int
	Dirs  int
	Bytes int64
	Path  string
}

var (
	browserArchiveDirEntryInfoFn = func(entry fs.DirEntry) (fs.FileInfo, error) { return entry.Info() }
	browserArchiveOpenFileFn     = os.Open
	browserArchiveChmodFn        = os.Chmod
)

func browserShouldSkipArchivePathError(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, os.ErrNotExist)
}

func browserShouldIgnoreCopyFileModeError(targetPath string, err error) bool {
	if err == nil {
		return false
	}
	cleanPath := filepath.Clean(strings.TrimSpace(targetPath))
	if runtime.GOOS != "linux" || !strings.HasPrefix(cleanPath, "/mnt/") {
		return false
	}
	if errors.Is(err, fs.ErrPermission) || errors.Is(err, os.ErrPermission) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "operation not permitted")
}
