package notification

import (
	"encoding/base64"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"unicode/utf16"
)

var (
	notificationLookPathFn = exec.LookPath
	notificationStartFn    = startNotificationProcess
)

func powerShellNotificationSupported() bool {
	_, ok := findPowerShellExecutable()
	return ok
}

func notifyViaPowerShell(opts Options) error {
	executable, ok := findPowerShellExecutable()
	if !ok {
		return nil
	}
	return notificationStartFn(executable, buildPowerShellArgs(opts)...)
}

func findPowerShellExecutable() (string, bool) {
	for _, candidate := range []string{"powershell.exe", "pwsh.exe", "powershell", "pwsh"} {
		path, err := notificationLookPathFn(candidate)
		if err == nil && strings.TrimSpace(path) != "" {
			return path, true
		}
	}
	return "", false
}

func buildPowerShellArgs(opts Options) []string {
	return []string{
		"-NoProfile",
		"-NonInteractive",
		"-WindowStyle",
		"Hidden",
		"-ExecutionPolicy",
		"Bypass",
		"-EncodedCommand",
		encodePowerShellCommand(buildPowerShellNotificationScript(opts)),
	}
}

func buildPowerShellNotificationScript(opts Options) string {
	title := escapePowerShellSingleQuotedLiteral(opts.Title)
	message := escapePowerShellSingleQuotedLiteral(opts.Message)
	return fmt.Sprintf(`$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Windows.Forms | Out-Null
Add-Type -AssemblyName System.Drawing | Out-Null
$notify = New-Object System.Windows.Forms.NotifyIcon
$notify.Icon = [System.Drawing.SystemIcons]::Information
$notify.BalloonTipIcon = [System.Windows.Forms.ToolTipIcon]::Info
$notify.BalloonTipTitle = '%s'
$notify.BalloonTipText = '%s'
$notify.Visible = $true
$notify.ShowBalloonTip(5000)
Start-Sleep -Milliseconds 5500
$notify.Dispose()
`, title, message)
}

func escapePowerShellSingleQuotedLiteral(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func looksLikeWSLKernel(data []byte) bool {
	text := strings.ToLower(string(data))
	return strings.Contains(text, "microsoft") || strings.Contains(text, "wsl")
}

func encodePowerShellCommand(script string) string {
	runes := utf16.Encode([]rune(script))
	encoded := make([]byte, 0, len(runes)*2)
	for _, r := range runes {
		encoded = append(encoded, byte(r), byte(r>>8))
	}
	return base64.StdEncoding.EncodeToString(encoded)
}

func startNotificationProcess(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
