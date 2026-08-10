package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// taskLogDiagnosticSummary keeps a readable tail of a child process's output
// in the terminal failure message. The complete output has already been
// appended to the task log line-by-line; this summary makes the actual cause
// visible even when a client only inspects the final failure entry.
func taskLogDiagnosticSummary(diagnostics string) string {
	value := strings.TrimSpace(diagnostics)
	if value == "" {
		return ""
	}
	const limit = 3000
	if len(value) > limit {
		value = "…" + value[len(value)-limit:]
	}
	return value
}

func taskExecutionError(name string, cause error, diagnostics string) error {
	if detail := taskLogDiagnosticSummary(diagnostics); detail != "" {
		return fmt.Errorf("%s执行失败: %w；详细错误输出：%s", name, cause, detail)
	}
	return fmt.Errorf("%s执行失败: %w", name, cause)
}

// taskPythonExecutionEnvironment exposes the interpreter selected by a Python
// console-script wrapper without invoking a second process. The task log must
// identify the exact entry point even when a wrapper has no readable shebang.
func taskPythonExecutionEnvironment(entry string) string {
	entry = strings.TrimSpace(entry)
	interpreter := ""
	if file, err := os.Open(entry); err == nil {
		var header [1024]byte
		n, _ := file.Read(header[:])
		_ = file.Close()
		firstLine := strings.TrimSpace(strings.SplitN(string(header[:n]), "\n", 2)[0])
		if strings.HasPrefix(firstLine, "#!") {
			parts := strings.Fields(strings.TrimSpace(strings.TrimPrefix(firstLine, "#!")))
			if len(parts) > 0 {
				candidate := parts[0]
				if filepath.Base(candidate) == "env" {
					for _, value := range parts[1:] {
						if value != "-S" && !strings.HasPrefix(value, "-") {
							candidate = value
						}
					}
				}
				if filepath.Base(candidate) != "env" && candidate != "" {
					if filepath.IsAbs(candidate) {
						interpreter = candidate
					} else if resolved, lookErr := exec.LookPath(candidate); lookErr == nil {
						interpreter = resolved
					} else {
						interpreter = candidate + "（未在 PATH 中解析）"
					}
				}
			}
		}
	}
	if interpreter == "" {
		return "Python/CPython 运行环境：未能从执行入口解析解释器；将直接执行：" + entry
	}
	return "Python/CPython 运行环境：解释器：" + interpreter + "；执行入口：" + entry
}

func taskKnownPythonEnvironment(python, script string) string {
	return "Python/CPython 运行环境：解释器：" + strings.TrimSpace(python) + "；执行脚本：" + strings.TrimSpace(script)
}

func taskLogAbsolutePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if absolute, err := filepath.Abs(path); err == nil {
		return absolute
	}
	return path
}
