package sharedutil

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const DefaultAPICmdTimeout = 180 * time.Second

// GenerateCmdTID returns a random 8-char alphanumeric task ID.
func GenerateCmdTID() string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// GzipBase64String compresses a string as gzip then base64.
func GzipBase64String(text string) string {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte(text)); err != nil {
		return text
	}
	gz.Close()
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// ---------------------------------------------------------------------------
// ActiveCmd management
// ---------------------------------------------------------------------------

// ActiveCmd represents a running shell command.
type ActiveCmd struct {
	Cancel    context.CancelFunc
	Cmd       *exec.Cmd
	AgentID   string
	ChatID    string
	Tid       string
	RawCmd    string
	StartedAt string
}

var (
	activeCmdMu  sync.Mutex
	activeCmdMap = make(map[string]*ActiveCmd)
)

// ActiveCmdKey builds the unique key for an active command.
func ActiveCmdKey(agentID, chatID, tid, cmd string) string {
	return agentID + "|" + chatID + "|" + tid + "|" + cmd
}

// RegisterActiveCmd stores an ActiveCmd and returns its key.
func RegisterActiveCmd(cmd *ActiveCmd) string {
	key := ActiveCmdKey(cmd.AgentID, cmd.ChatID, cmd.Tid, cmd.RawCmd)
	activeCmdMu.Lock()
	activeCmdMap[key] = cmd
	activeCmdMu.Unlock()
	return key
}

// UnregisterActiveCmd removes an ActiveCmd by key.
func UnregisterActiveCmd(key string) {
	activeCmdMu.Lock()
	delete(activeCmdMap, key)
	activeCmdMu.Unlock()
}

// FindActiveCmd looks up an ActiveCmd by its identity fields.
// First tries exact (agentID, chatID, tid, rawCmd) match, then falls back
// to scanning for any command with matching (agentID, chatID, rawCmd).
func FindActiveCmd(agentID, chatID, tid, rawCmd string) (*ActiveCmd, string) {
	key := ActiveCmdKey(agentID, chatID, tid, rawCmd)
	activeCmdMu.Lock()
	cmd, ok := activeCmdMap[key]
	activeCmdMu.Unlock()
	if ok {
		return cmd, key
	}

	activeCmdMu.Lock()
	for _, ac := range activeCmdMap {
		if ac.AgentID == agentID && ac.ChatID == chatID && ac.RawCmd == rawCmd {
			activeCmdMu.Unlock()
			return ac, ActiveCmdKey(ac.AgentID, ac.ChatID, ac.Tid, ac.RawCmd)
		}
	}
	activeCmdMu.Unlock()
	return nil, ""
}

// ExecuteShellCommand runs a shell command with a timeout.
// onStart is called before execution begins with a partially-filled ActiveCmd.
func ExecuteShellCommand(shell, rawCmd string, timeoutMs int64, onStart func(*ActiveCmd)) (int, string) {
	ApplySystemPath()
	if timeoutMs <= 0 {
		timeoutMs = int64(DefaultAPICmdTimeout.Milliseconds())
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, shell, "-c", rawCmd)
	cmd.Env = CurrentEnvironmentWithSystemPath()
	cmd.Dir, _ = os.Getwd()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if onStart != nil {
		onStart(&ActiveCmd{
			Cancel: cancel,
			Cmd:    cmd,
			RawCmd: rawCmd,
		})
	}

	err := cmd.Run()

	var output string
	if stdout.Len() > 0 {
		output = strings.TrimRight(stdout.String(), "\n\r\t ")
	}
	if stderr.Len() > 0 {
		stderrStr := strings.TrimRight(stderr.String(), "\n\r\t ")
		if output != "" {
			output += "\n" + stderrStr
		} else {
			output = stderrStr
		}
	}

	if ctx.Err() != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return 124, output
		}
		return -1, output
	}
	if err != nil {
		if output == "" {
			output = err.Error()
		}
		return 1, output
	}
	return 0, output
}

// DefaultExecShell returns the user's shell, using SHELL env or /bin/sh.
func DefaultExecShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return shell
}

// IntPtrValue safely dereferences an int pointer.
func IntPtrValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
