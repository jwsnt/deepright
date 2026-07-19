package main

import (
	"os/exec"
	"reflect"
	"testing"
)

func withIntegrationCaffeinateTestHooks(t *testing.T, goos string, commandFn func(string, ...string) *exec.Cmd) {
	t.Helper()
	oldGOOS := integrationRuntimeGOOS
	oldCommandFn := integrationCaffeinateCommandFn
	integrationRuntimeGOOS = goos
	integrationCaffeinateCommandFn = commandFn
	t.Cleanup(func() {
		integrationRuntimeGOOS = oldGOOS
		integrationCaffeinateCommandFn = oldCommandFn
	})
}

func TestStartIntegrationSleepAssertionUsesCaffeinateOnMacOS(t *testing.T) {
	var gotName string
	var gotArgs []string
	withIntegrationCaffeinateTestHooks(t, "darwin", func(name string, args ...string) *exec.Cmd {
		gotName = name
		gotArgs = append([]string(nil), args...)
		return exec.Command("sh", "-c", "exec sleep 30")
	})

	assertion, err := startIntegrationSleepAssertion()
	if err != nil {
		t.Fatalf("start sleep assertion: %v", err)
	}
	if assertion == nil {
		t.Fatal("expected macOS sleep assertion")
	}
	t.Cleanup(func() { _ = assertion.Stop() })

	if gotName != integrationCaffeinateBinary {
		t.Fatalf("command = %q, want %q", gotName, integrationCaffeinateBinary)
	}
	if want := []string{"-d", "-i", "-m", "-s"}; !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("arguments = %#v, want %#v", gotArgs, want)
	}
}

func TestStartIntegrationSleepAssertionSkipsNonMacOS(t *testing.T) {
	called := false
	withIntegrationCaffeinateTestHooks(t, "linux", func(string, ...string) *exec.Cmd {
		called = true
		return exec.Command("sh", "-c", "exit 1")
	})

	assertion, err := startIntegrationSleepAssertion()
	if err != nil {
		t.Fatalf("start sleep assertion: %v", err)
	}
	if assertion != nil {
		t.Fatal("unexpected non-macOS sleep assertion")
	}
	if called {
		t.Fatal("caffeinate command must not run outside macOS")
	}
}

func TestStartIntegrationSleepAssertionFailureDoesNotCreateAssertion(t *testing.T) {
	withIntegrationCaffeinateTestHooks(t, "darwin", func(string, ...string) *exec.Cmd {
		return exec.Command("integration-caffeinate-command-does-not-exist")
	})

	assertion, err := startIntegrationSleepAssertion()
	if err == nil {
		t.Fatal("expected command start error")
	}
	if assertion != nil {
		t.Fatal("unexpected assertion after start failure")
	}
}

func TestIntegrationCaffeinateProcessStopTerminatesProcessOnlyOnce(t *testing.T) {
	withIntegrationCaffeinateTestHooks(t, "darwin", func(string, ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "exec sleep 30")
	})

	assertion, err := startIntegrationSleepAssertion()
	if err != nil {
		t.Fatalf("start sleep assertion: %v", err)
	}
	process, ok := assertion.(*integrationCaffeinateProcess)
	if !ok {
		t.Fatalf("assertion type = %T, want *integrationCaffeinateProcess", assertion)
	}

	if err := process.Stop(); err != nil {
		t.Fatalf("first stop: %v", err)
	}
	select {
	case <-process.done:
	default:
		t.Fatal("process remains active after stop")
	}
	if err := process.Stop(); err != nil {
		t.Fatalf("second stop: %v", err)
	}
}
