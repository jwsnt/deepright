package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
)

type rotatingLocalFeishuLogWriter struct {
	maxBackups int
	maxBytes   int64
	path       string

	mu   sync.Mutex
	file *os.File
	size int64
}

func newRotatingLocalFeishuLogWriter(path string, maxBytes int64, maxBackups int) *rotatingLocalFeishuLogWriter {
	return &rotatingLocalFeishuLogWriter{
		path:       strings.TrimSpace(path),
		maxBytes:   maxBytes,
		maxBackups: maxBackups,
	}
}

func (w *rotatingLocalFeishuLogWriter) Write(p []byte) (int, error) {
	if w == nil {
		return 0, errors.New("log writer is required")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.ensureFile(); err != nil {
		return 0, err
	}
	if w.maxBytes > 0 && w.size+int64(len(p)) > w.maxBytes {
		if err := w.rotate(); err != nil {
			return 0, err
		}
		if err := w.ensureFile(); err != nil {
			return 0, err
		}
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *rotatingLocalFeishuLogWriter) Close() error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	w.size = 0
	return err
}

func (w *rotatingLocalFeishuLogWriter) ensureFile() error {
	if w.file != nil {
		return nil
	}
	if err := ensureParentDirLocalFeishu(w.path); err != nil {
		return err
	}

	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}

	w.file = file
	w.size = info.Size()
	return nil
}

func (w *rotatingLocalFeishuLogWriter) rotate() error {
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return err
		}
		w.file = nil
	}

	if w.maxBackups > 0 {
		last := w.backupPath(w.maxBackups)
		_ = os.Remove(last)
		for idx := w.maxBackups - 1; idx >= 1; idx-- {
			src := w.backupPath(idx)
			dst := w.backupPath(idx + 1)
			if _, err := os.Stat(src); err == nil {
				if err := os.Rename(src, dst); err != nil {
					return err
				}
			}
		}
		if _, err := os.Stat(w.path); err == nil {
			if err := os.Rename(w.path, w.backupPath(1)); err != nil {
				return err
			}
		}
	} else {
		if err := os.Remove(w.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	w.size = 0
	return nil
}

func (w *rotatingLocalFeishuLogWriter) backupPath(index int) string {
	return fmt.Sprintf("%s.%d", w.path, index)
}
