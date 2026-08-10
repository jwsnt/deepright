package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

const browserLogFilterCommand = "__log-filter"

func browserStartChromeLogFilter(logPath string) (_ *os.File, cleanup func(), err error) {
	logPath = strings.TrimSpace(logPath)
	if logPath == "" {
		return nil, nil, fmt.Errorf("browser log path is empty")
	}

	exe, err := browserExecutablePathFn()
	if err != nil {
		return nil, nil, err
	}

	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		_ = pipeReader.Close()
		_ = pipeWriter.Close()
		return nil, nil, err
	}

	cmd := exec.Command(exe, browserLogFilterCommand, "--log-file", logPath)
	cmd.Stdin = pipeReader
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	browserPrepareDetachedCommand(cmd)
	if err := cmd.Start(); err != nil {
		_ = pipeReader.Close()
		_ = pipeWriter.Close()
		_ = devNull.Close()
		return nil, nil, err
	}
	_ = cmd.Process.Release()
	_ = pipeReader.Close()

	cleanup = func() {
		_ = pipeWriter.Close()
		_ = devNull.Close()
	}
	return pipeWriter, cleanup, nil
}

func browserRunLogFilter(input io.Reader, logPath string) error {
	logPath = strings.TrimSpace(logPath)
	if logPath == "" {
		return fmt.Errorf("log-file is required")
	}

	writer, err := browserLogWriter(logPath)
	if err != nil {
		return err
	}
	defer writer.Close()

	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if browserShouldFilterChromeLogLine(line) {
			continue
		}
		if _, err := io.WriteString(writer, line+"\n"); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func browserShouldFilterChromeLogLine(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	switch {
	case strings.Contains(line, "chrome/updater/"):
		return true
	case strings.Contains(line, "third_party/crashpad/crashpad/"):
		return true
	case strings.Contains(line, "google_apis/gcm/engine/registration_request.cc") && strings.Contains(line, "DEPRECATED_ENDPOINT"):
		return true
	default:
		return false
	}
}
