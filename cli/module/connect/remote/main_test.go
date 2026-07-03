package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"connect/remote/runtimecfg"
)

func containsString(items []string, target string) bool {
	for _, item := range items {
		if strings.TrimSpace(item) == strings.TrimSpace(target) {
			return true
		}
	}
	return false
}

func TestHelpIncludesManagedAndDelegatedSSHUsage(t *testing.T) {
	oldOutput := sshBinaryOutputFn
	oldLookPath := sshBinaryLookPath
	t.Cleanup(func() {
		sshBinaryOutputFn = oldOutput
		sshBinaryLookPath = oldLookPath
	})

	sshBinaryLookPath = func(file string) (string, error) {
		return "/usr/bin/ssh", nil
	}
	sshBinaryOutputFn = func(name string, args []string) (string, error) {
		return "usage: ssh [-46AaCfGgKkMNnqsTtVvXxYy] destination [command]", nil
	}

	stdout := &bytes.Buffer{}
	code := runCLI([]string{"help"}, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("help exit code = %d", code)
	}

	text := stdout.String()
	for _, want := range []string{
		"remote create --agentId Agent-A --chatId Chat-001",
		"--remote ubuntu@1.2.3.4",
		"--certificate /path/to/id.pem",
		"remote.json is always written beside the remote binary",
		"exec reuses the cached SSH connection",
		"remote scp ./artifact.txt",
		"Delegated SSH Manual:",
		"usage: ssh",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("help output missing %q\n%s", want, text)
		}
	}
	for _, unwanted := range []string{
		"  start       start the remote manager daemon and write remote.pid",
		"  stop        stop the remote manager daemon and clear all SSH sessions",
		"./remote start",
		"./remote stop",
		"start writes remote.pid under the plugin runtime root",
		"stop closes the started manager process",
	} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("help output should not contain %q\n%s", unwanted, text)
		}
	}
}

func TestCommandMetadataIncludesExecAndSCP(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := runCLI([]string{"command"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("command exit code = %d, stderr = %s", code, stderr.String())
	}

	var commands []string
	if err := json.Unmarshal(stdout.Bytes(), &commands); err != nil {
		t.Fatalf("decode commands: %v raw=%s", err, stdout.String())
	}
	for _, want := range []string{"command", "exec", "help", "name", "param", "scope", "start", "stop", "scp"} {
		if !containsString(commands, want) {
			t.Fatalf("command output missing %q: %+v", want, commands)
		}
	}
	for _, hidden := range []string{"__daemon", "__manager"} {
		if containsString(commands, hidden) {
			t.Fatalf("command output should hide %q: %+v", hidden, commands)
		}
	}
}

func TestNameReturnsRemoteDisplayName(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := runCLI([]string{"name"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("name exit code = %d, stderr = %s", code, stderr.String())
	}

	var item map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &item); err != nil {
		t.Fatalf("decode name: %v raw=%s", err, stdout.String())
	}
	if item["key"] != "remote" {
		t.Fatalf("unexpected key: %+v", item)
	}
	if item["name"] != "远程" {
		t.Fatalf("unexpected name: %+v", item)
	}
}

func TestParamReturnsRuntimeConfigKeys(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := runCLI([]string{"param"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("param exit code = %d, stderr = %s", code, stderr.String())
	}

	var params []remoteParamDescriptor
	if err := json.Unmarshal(stdout.Bytes(), &params); err != nil {
		t.Fatalf("decode params: %v raw=%s", err, stdout.String())
	}
	if len(params) != 1 {
		t.Fatalf("param output length = %d, want 1: %+v", len(params), params)
	}
	if got := params[0].ExecTimeout; got != "选填。SSH执行超时" {
		t.Fatalf("exec_timeout = %q", got)
	}
	if got := params[0].SCPTimeout; got != "选填。SCP执行超时" {
		t.Fatalf("scp_timeout = %q", got)
	}
}

func TestScopeReturnsEmptyArray(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := runCLI([]string{"scope"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("scope exit code = %d, stderr = %s", code, stderr.String())
	}

	var scopes []string
	if err := json.Unmarshal(stdout.Bytes(), &scopes); err != nil {
		t.Fatalf("decode scope: %v raw=%s", err, stdout.String())
	}
	if len(scopes) != 0 {
		t.Fatalf("unexpected scope output: %+v", scopes)
	}
}

func TestSCPPassthroughInvokesSystemBinary(t *testing.T) {
	tempDir := t.TempDir()
	fakeSCP := filepath.Join(tempDir, "scp")
	script := "#!/bin/sh\nprintf 'SCP:%s\\n' \"$*\"\n"
	if err := os.WriteFile(fakeSCP, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	oldLookPath := sshBinaryLookPath
	oldResolveSession := resolveSCPControlPathFn
	t.Cleanup(func() {
		sshBinaryLookPath = oldLookPath
		resolveSCPControlPathFn = oldResolveSession
	})
	sshBinaryLookPath = func(file string) (string, error) {
		switch file {
		case "scp":
			return fakeSCP, nil
		case "ssh":
			return "/usr/bin/ssh", nil
		default:
			return "", os.ErrNotExist
		}
	}
	resolveSCPControlPathFn = func(flags map[string]string) (scpSessionInfo, error) {
		if got := flags["remote"]; got != "ubuntu@1.2.3.4" {
			t.Fatalf("remote = %q", got)
		}
		return scpSessionInfo{
			Port:        22,
			ControlPath: "/tmp/remote-control.sock",
		}, nil
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := runCLI([]string{"scp", "--session", "agent-a@chat-001", "./a.txt", "ubuntu@1.2.3.4:/tmp/"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("scp exit code = %d, stderr = %s", code, stderr.String())
	}
	text := strings.TrimSpace(stdout.String())
	if !strings.Contains(text, "SCP:-o ControlMaster=auto -o ControlPath=/tmp/remote-control.sock -o ControlPersist=no -P 22 ./a.txt ubuntu@1.2.3.4:/tmp/") {
		t.Fatalf("unexpected scp output: %q", stdout.String())
	}
}

func TestParseSCPArgsSeparatesConnectBin(t *testing.T) {
	flags, passthrough := parseSCPArgs([]string{
		"--connect-bin", "/tmp/integration",
		"--session", "agent-a@chat-001",
		"--timeout", "30000",
		"-P", "2222",
		"./a.txt",
		"ubuntu@1.2.3.4:/tmp/",
	})
	if got := flags["connect-bin"]; got != "/tmp/integration" {
		t.Fatalf("connect-bin = %q", got)
	}
	if got := flags["session"]; got != "agent-a@chat-001" {
		t.Fatalf("session = %q", got)
	}
	if got := flags["timeout"]; got != "30000" {
		t.Fatalf("timeout = %q", got)
	}
	if got := flags["remote"]; got != "ubuntu@1.2.3.4" {
		t.Fatalf("remote = %q", got)
	}
	if strings.Join(passthrough, " ") != "-P 2222 ./a.txt ubuntu@1.2.3.4:/tmp/" {
		t.Fatalf("unexpected passthrough args: %+v", passthrough)
	}
}

func TestSCPPassthroughReusesManagedSession(t *testing.T) {
	tempDir := t.TempDir()
	fakeSCP := filepath.Join(tempDir, "scp")
	script := "#!/bin/sh\nprintf 'SCP:%s\\n' \"$*\"\n"
	if err := os.WriteFile(fakeSCP, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	oldLookPath := sshBinaryLookPath
	t.Cleanup(func() {
		sshBinaryLookPath = oldLookPath
	})
	sshBinaryLookPath = func(file string) (string, error) {
		switch file {
		case "scp":
			return fakeSCP, nil
		case "ssh":
			return "/usr/bin/ssh", nil
		default:
			return "", os.ErrNotExist
		}
	}

	oldResolveSession := resolveSCPControlPathFn
	t.Cleanup(func() {
		resolveSCPControlPathFn = oldResolveSession
	})
	resolveSCPControlPathFn = func(flags map[string]string) (scpSessionInfo, error) {
		if got := flags["session"]; got != "agent-a@chat-001" {
			t.Fatalf("session = %q", got)
		}
		if got := flags["remote"]; got != "ubuntu@1.2.3.4" {
			t.Fatalf("remote = %q", got)
		}
		return scpSessionInfo{
			Port:        10086,
			ControlPath: "/tmp/remote-control.sock",
		}, nil
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := runCLI([]string{"scp", "--session", "agent-a@chat-001", "./a.txt", "ubuntu@1.2.3.4:/tmp/"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("scp exit code = %d, stderr = %s", code, stderr.String())
	}
	text := strings.TrimSpace(stdout.String())
	for _, want := range []string{
		"-o ControlMaster=auto",
		"-o ControlPath=/tmp/remote-control.sock",
		"-o ControlPersist=no",
		"-P 10086",
		"./a.txt ubuntu@1.2.3.4:/tmp/",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("scp output missing %q: %q", want, text)
		}
	}
}

func TestResolvePluginTimeoutsFromMetaGet(t *testing.T) {
	oldExec := runtimecfg.ExecCommandContext
	oldSelf := runtimecfg.OSExecutable
	t.Cleanup(func() {
		runtimecfg.ExecCommandContext = oldExec
		runtimecfg.OSExecutable = oldSelf
	})

	runtimecfg.OSExecutable = func() (string, error) { return "/tmp/remote", nil }
	runtimecfg.ExecCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		script := "printf '%s' '{\"key\":\"remote\",\"meta\":{\"exec_timeout\":12,\"scp_timeout\":\"34\"}}'"
		return exec.CommandContext(ctx, "sh", "-c", script)
	}

	timeouts := runtimecfg.ResolvePluginTimeouts(map[string]string{"connect-bin": "/tmp/integration"})
	if timeouts.Exec != 12*time.Millisecond {
		t.Fatalf("exec timeout = %v", timeouts.Exec)
	}
	if timeouts.SCP != 34*time.Millisecond {
		t.Fatalf("scp timeout = %v", timeouts.SCP)
	}
}

func TestResolvePluginTimeoutsFallsBackToDefault(t *testing.T) {
	oldExec := runtimecfg.ExecCommandContext
	oldSelf := runtimecfg.OSExecutable
	t.Cleanup(func() {
		runtimecfg.ExecCommandContext = oldExec
		runtimecfg.OSExecutable = oldSelf
	})

	runtimecfg.OSExecutable = func() (string, error) { return "/tmp/remote", nil }
	runtimecfg.ExecCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		script := "printf '%s' '{\"key\":\"remote\",\"meta\":{\"exec_timeout\":\"\",\"scp_timeout\":\"0\"}}'"
		return exec.CommandContext(ctx, "sh", "-c", script)
	}

	timeouts := runtimecfg.ResolvePluginTimeouts(map[string]string{"connect-bin": "/tmp/integration"})
	if timeouts.Exec != runtimecfg.DefaultCommandTimeout {
		t.Fatalf("exec timeout = %v", timeouts.Exec)
	}
	if timeouts.SCP != runtimecfg.DefaultSCPTimeout {
		t.Fatalf("scp timeout = %v", timeouts.SCP)
	}
}

func TestResolvePluginTimeoutsPrefersCLIFlag(t *testing.T) {
	timeouts := runtimecfg.ResolvePluginTimeouts(map[string]string{
		"timeout":     "1234",
		"connect-bin": "/tmp/integration",
	})
	if timeouts.Exec != 1234*time.Millisecond {
		t.Fatalf("exec timeout = %v", timeouts.Exec)
	}
	if timeouts.SCP != 1234*time.Millisecond {
		t.Fatalf("scp timeout = %v", timeouts.SCP)
	}
}

func TestParseRemoteCommandArgsAcceptsTrailingCommand(t *testing.T) {
	flags, remoteCommand, err := parseRemoteCommandArgs([]string{"--session", "Agent@Chat", "ls", "-la", "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	if flags["session"] != "Agent@Chat" {
		t.Fatalf("unexpected flags: %+v", flags)
	}
	if remoteCommand != "ls -la /tmp" {
		t.Fatalf("remote exec command = %q", remoteCommand)
	}
}

func TestRemoteBinaryDetachedLifecycle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("detached lifecycle integration test is unix-oriented")
	}

	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "remote-test")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./remote")
	buildCmd.Dir = filepath.Clean(filepath.Join(".."))
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build remote binary: %v\n%s", err, string(output))
	}

	fakeSSH := filepath.Join(tempDir, "fake-ssh.sh")
	fakeScript := `#!/bin/sh
control=""
op=""
mode=""
remote=""
cmd=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -h|--help)
      echo "usage: ssh [options] destination [command]" >&2
      exit 0
      ;;
    -S)
      shift
      control="$1"
      ;;
    -O)
      shift
      op="$1"
      ;;
    -p)
      shift
      ;;
    -M)
      mode="master"
      ;;
    -N)
      ;;
    -o)
      shift
      ;;
    --)
      shift
      break
      ;;
    -*)
      ;;
    *)
      if [ -z "$remote" ]; then
        remote="$1"
      else
        if [ -n "$cmd" ]; then
          cmd="$cmd $1"
        else
          cmd="$1"
        fi
      fi
      ;;
  esac
  shift
done
if [ "$#" -gt 0 ]; then
  if [ -n "$cmd" ]; then
    cmd="$cmd $*"
  else
    cmd="$*"
  fi
fi
case "$op" in
  check)
    if [ -n "$control" ] && [ -e "$control" ]; then
      echo "Master running"
      exit 0
    fi
    echo "No master" >&2
    exit 255
    ;;
  exit)
    rm -f "$control"
    exit 0
    ;;
esac
if [ "$mode" = "master" ]; then
  mkdir -p "$(dirname "$control")"
  : > "$control"
  trap 'rm -f "$control"; exit 0' INT TERM EXIT
  while [ -e "$control" ]; do
    sleep 1
  done
  exit 0
fi
if [ -n "$control" ] && [ ! -e "$control" ]; then
  echo "No master" >&2
  exit 255
fi
case "$cmd" in
  *fail*)
    echo "remote failed" >&2
    exit 7
    ;;
  *)
    echo "EXEC:$cmd"
    exit 0
    ;;
esac
`
	if err := os.WriteFile(fakeSSH, []byte(fakeScript), 0o755); err != nil {
		t.Fatal(err)
	}

	create := exec.Command(binaryPath,
		"create",
		"--agentId", "Agent-A",
		"--chatId", "Chat-001",
		"--remote", "ubuntu@127.0.0.1",
		"--password", "secret",
		"--ssh-bin", fakeSSH,
	)
	create.Dir = tempDir
	createOutput, err := create.CombinedOutput()
	if err != nil {
		t.Fatalf("create failed: %v\n%s", err, string(createOutput))
	}

	var created map[string]any
	if err := json.Unmarshal(createOutput, &created); err != nil {
		t.Fatalf("decode create output: %v raw=%s", err, string(createOutput))
	}
	if created["agentId"] != "agent-a" || created["chatId"] != "chat-001" || created["ssh"] != "ubuntu@127.0.0.1" {
		t.Fatalf("unexpected create output: %+v", created)
	}

	time.Sleep(300 * time.Millisecond)

	getCmd := exec.Command(binaryPath, "get", "--agentId", "AGENT-A", "--chatId", "CHAT-001")
	getCmd.Args = append(getCmd.Args, "--remote", "ubuntu@127.0.0.1")
	getCmd.Dir = tempDir
	getOutput, err := getCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("get failed: %v\n%s", err, string(getOutput))
	}
	var fetched map[string]any
	if err := json.Unmarshal(getOutput, &fetched); err != nil {
		t.Fatalf("decode get output: %v raw=%s", err, string(getOutput))
	}
	if fetched["agentId"] != "agent-a" || fetched["chatId"] != "chat-001" {
		t.Fatalf("unexpected get output: %+v", fetched)
	}

	execCmd := exec.Command(binaryPath, "exec", "--session", "AGENT-A@CHAT-001", "--remote", "ubuntu@127.0.0.1", "echo hello")
	execCmd.Dir = tempDir
	commandOutput, err := execCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("exec failed: %v\n%s", err, string(commandOutput))
	}
	if strings.TrimSpace(string(commandOutput)) != "EXEC:echo hello" {
		t.Fatalf("unexpected exec output: %q", string(commandOutput))
	}

	logPath := filepath.Join(tempDir, "remote.log")
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("remote.log missing: %v", err)
	}

	statePath := filepath.Join(tempDir, "remote.json")
	stateData, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read remote.json: %v", err)
	}
	if !strings.Contains(string(stateData), "\"agentId\": \"agent-a\"") {
		t.Fatalf("unexpected state file: %s", string(stateData))
	}

	shutdownCmd := exec.Command(binaryPath, "shutdown", "--agentId", "agent-a", "--chatId", "chat-001", "--remote", "ubuntu@127.0.0.1")
	shutdownCmd.Dir = tempDir
	shutdownOutput, err := shutdownCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("shutdown failed: %v\n%s", err, string(shutdownOutput))
	}
	if strings.TrimSpace(string(shutdownOutput)) != "OK" {
		t.Fatalf("unexpected shutdown output: %q", string(shutdownOutput))
	}
}

func TestRemoteBinaryCreateWithCertificate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("detached lifecycle integration test is unix-oriented")
	}

	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "remote-test")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./remote")
	buildCmd.Dir = filepath.Clean(filepath.Join(".."))
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build remote binary: %v\n%s", err, string(output))
	}

	fakeSSH := filepath.Join(tempDir, "fake-ssh.sh")
	fakeScript := `#!/bin/sh
control=""
op=""
mode=""
remote=""
cmd=""
key=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -h|--help)
      echo "usage: ssh [options] destination [command]" >&2
      exit 0
      ;;
    -S)
      shift
      control="$1"
      ;;
    -O)
      shift
      op="$1"
      ;;
    -p|-o)
      shift
      ;;
    -i)
      shift
      key="$1"
      ;;
    -M)
      mode="master"
      ;;
    -N)
      ;;
    --)
      shift
      break
      ;;
    -*)
      ;;
    *)
      if [ -z "$remote" ]; then
        remote="$1"
      else
        if [ -n "$cmd" ]; then
          cmd="$cmd $1"
        else
          cmd="$1"
        fi
      fi
      ;;
  esac
  shift
done
case "$op" in
  check)
    if [ -n "$control" ] && [ -e "$control" ]; then
      exit 0
    fi
    exit 255
    ;;
  exit)
    rm -f "$control"
    exit 0
    ;;
esac
if [ "$mode" = "master" ]; then
  mkdir -p "$(dirname "$control")"
  echo "$key" > "$control"
  trap 'rm -f "$control"; exit 0' INT TERM EXIT
  while [ -e "$control" ]; do
    sleep 1
  done
  exit 0
fi
echo "EXEC:$cmd"
`
	if err := os.WriteFile(fakeSSH, []byte(fakeScript), 0o755); err != nil {
		t.Fatal(err)
	}

	certPath := filepath.Join(tempDir, "id.pem")
	if err := os.WriteFile(certPath, []byte("pem"), 0o600); err != nil {
		t.Fatal(err)
	}

	create := exec.Command(binaryPath,
		"create",
		"--agentId", "Agent-A",
		"--chatId", "Chat-002",
		"--remote", "ubuntu@127.0.0.1",
		"--certificate", certPath,
		"--ssh-bin", fakeSSH,
	)
	create.Dir = tempDir
	createOutput, err := create.CombinedOutput()
	if err != nil {
		t.Fatalf("create failed: %v\n%s", err, string(createOutput))
	}

	var created map[string]any
	if err := json.Unmarshal(createOutput, &created); err != nil {
		t.Fatalf("decode create output: %v raw=%s", err, string(createOutput))
	}
	if created["chatId"] != "chat-002" {
		t.Fatalf("unexpected create output: %+v", created)
	}
}
