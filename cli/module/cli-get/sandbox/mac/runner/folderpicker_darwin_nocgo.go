//go:build darwin && !cgo

package main

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
)

func sandboxChooseFolderWithTimeout(timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		"/usr/bin/osascript",
		"-ss",
		"-e", `POSIX path of (choose folder with prompt "CLI_SANDBOX 请选择允许访问的目录")`,
	)
	out, err := cmd.CombinedOutput()
	text := stripWrappedQuotes(strings.TrimSpace(string(out)))
	if ctx.Err() == context.DeadlineExceeded {
		return text, errors.New("目录授权弹窗超时，请切回桌面确认选择窗口后重试")
	}
	if err != nil {
		switch {
		case strings.Contains(text, "(-128)"):
			return text, errors.New("已取消目录授权")
		case text != "":
			return text, errors.New(text)
		default:
			return "", err
		}
	}
	return text, nil
}
