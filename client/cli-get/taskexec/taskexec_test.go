package taskexec

import (
	"runtime"
	"testing"
)

func TestTimeoutOutput(t *testing.T) {
	if got := timeoutOutput(""); got != "[Warning: Command execution timed out.]" {
		t.Fatalf("timeoutOutput(empty) = %q", got)
	}
	if got := timeoutOutput("partial output"); got != "partial output[Warning: Command execution timed out, the returned content may be incomplete.]" {
		t.Fatalf("timeoutOutput(partial) = %q", got)
	}
	if got := timeoutOutput("\n\t "); got != "[Warning: Command execution timed out.]" {
		t.Fatalf("timeoutOutput(whitespace) = %q", got)
	}
}

func TestExecutePreservesStderrOnTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell command is POSIX-specific")
	}

	result := Execute("printf 'stderr output' >&2; sleep 1", Options{
		Shell:     "/bin/sh",
		TimeoutMs: 50,
	})
	if result.Status != 1 || !result.TimedOut {
		t.Fatalf("result = %+v, want timeout failure", result)
	}
	want := "stderr output[Warning: Command execution timed out, the returned content may be incomplete.]"
	if result.Output != want {
		t.Fatalf("output = %q, want %q", result.Output, want)
	}
}

func TestExecutePreservesPartialOutputOnTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell command is POSIX-specific")
	}

	result := Execute("printf 'partial output'; sleep 1", Options{
		Shell:     "/bin/sh",
		TimeoutMs: 50,
	})
	if result.Status != 1 || !result.TimedOut {
		t.Fatalf("result = %+v, want timeout failure", result)
	}
	want := "partial output[Warning: Command execution timed out, the returned content may be incomplete.]"
	if result.Output != want {
		t.Fatalf("output = %q, want %q", result.Output, want)
	}
}
