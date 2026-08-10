package notification

import (
	"encoding/base64"
	"os/exec"
	"strings"
	"testing"
)

func TestNormalizeOptions(t *testing.T) {
	opts, err := NormalizeOptions(Options{
		Title:   "  DeepRight通知  ",
		Message: "  普通对话已完成  ",
	})
	if err != nil {
		t.Fatalf("NormalizeOptions() error = %v", err)
	}
	if opts.Title != "DeepRight通知" {
		t.Fatalf("NormalizeOptions() title = %q, want %q", opts.Title, "DeepRight通知")
	}
	if opts.Message != "普通对话已完成" {
		t.Fatalf("NormalizeOptions() message = %q, want %q", opts.Message, "普通对话已完成")
	}
}

func TestNormalizeOptionsRejectsEmptyTitle(t *testing.T) {
	if _, err := NormalizeOptions(Options{Title: "   "}); err != ErrEmptyTitle {
		t.Fatalf("NormalizeOptions() error = %v, want %v", err, ErrEmptyTitle)
	}
}

func TestBuildPowerShellNotificationScriptEscapesSingleQuotes(t *testing.T) {
	script := buildPowerShellNotificationScript(Options{
		Title:   "DeepRight's 通知",
		Message: "It's done",
	})
	if !strings.Contains(script, "$notify.BalloonTipTitle = 'DeepRight''s 通知'") {
		t.Fatalf("script title escape missing: %q", script)
	}
	if !strings.Contains(script, "$notify.BalloonTipText = 'It''s done'") {
		t.Fatalf("script message escape missing: %q", script)
	}
}

func TestFindPowerShellExecutableFallsBackCandidates(t *testing.T) {
	oldLookPath := notificationLookPathFn
	t.Cleanup(func() {
		notificationLookPathFn = oldLookPath
	})

	var looked []string
	notificationLookPathFn = func(file string) (string, error) {
		looked = append(looked, file)
		if file == "pwsh.exe" {
			return "/mock/pwsh.exe", nil
		}
		return "", exec.ErrNotFound
	}

	got, ok := findPowerShellExecutable()
	if !ok {
		t.Fatal("findPowerShellExecutable() ok = false, want true")
	}
	if got != "/mock/pwsh.exe" {
		t.Fatalf("findPowerShellExecutable() = %q, want %q", got, "/mock/pwsh.exe")
	}
	if len(looked) < 2 || looked[0] != "powershell.exe" || looked[1] != "pwsh.exe" {
		t.Fatalf("lookup order = %v, want powershell.exe then pwsh.exe", looked)
	}
}

func TestNotifyViaPowerShellStartsDetachedProcess(t *testing.T) {
	oldLookPath := notificationLookPathFn
	oldStart := notificationStartFn
	t.Cleanup(func() {
		notificationLookPathFn = oldLookPath
		notificationStartFn = oldStart
	})

	notificationLookPathFn = func(file string) (string, error) {
		if file == "powershell.exe" {
			return "/mock/powershell.exe", nil
		}
		return "", exec.ErrNotFound
	}

	var gotName string
	var gotArgs []string
	notificationStartFn = func(name string, args ...string) error {
		gotName = name
		gotArgs = append([]string{}, args...)
		return nil
	}

	if err := notifyViaPowerShell(Options{Title: "DeepRight通知", Message: "普通对话已完成"}); err != nil {
		t.Fatalf("notifyViaPowerShell() error = %v", err)
	}
	if gotName != "/mock/powershell.exe" {
		t.Fatalf("process name = %q, want %q", gotName, "/mock/powershell.exe")
	}
	if len(gotArgs) < 8 {
		t.Fatalf("process args too short: %v", gotArgs)
	}
	encoded := gotArgs[len(gotArgs)-1]
	decoded := decodeUTF16LEBase64(t, encoded)
	if !strings.Contains(decoded, "$notify.BalloonTipTitle = 'DeepRight通知'") {
		t.Fatalf("decoded script missing title: %q", decoded)
	}
	if !strings.Contains(decoded, "$notify.BalloonTipText = '普通对话已完成'") {
		t.Fatalf("decoded script missing message: %q", decoded)
	}
}

func TestLooksLikeWSLKernel(t *testing.T) {
	cases := []struct {
		name string
		data string
		want bool
	}{
		{name: "wsl2 release", data: "5.15.153.1-microsoft-standard-WSL2", want: true},
		{name: "microsoft version", data: "Linux version 5.15.90.1-microsoft-standard-WSL2", want: true},
		{name: "plain linux", data: "6.10.0-arch1-1", want: false},
	}
	for _, tc := range cases {
		if got := looksLikeWSLKernel([]byte(tc.data)); got != tc.want {
			t.Fatalf("%s: looksLikeWSLKernel() = %t, want %t", tc.name, got, tc.want)
		}
	}
}

func decodeUTF16LEBase64(t *testing.T, value string) string {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		t.Fatalf("decode base64: %v", err)
	}
	if len(data)%2 != 0 {
		t.Fatalf("decoded data length = %d, want even", len(data))
	}
	var builder strings.Builder
	for i := 0; i < len(data); i += 2 {
		builder.WriteRune(rune(uint16(data[i]) | uint16(data[i+1])<<8))
	}
	return builder.String()
}
