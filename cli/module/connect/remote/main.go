package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"connect/connectsvc"
	"connect/remote/instance"
)

const (
	remoteKey         = "remote"
	remoteDisplayName = "远程"
)

var (
	runCreateFn        = instance.CreateViaManager
	runShutdownFn      = instance.ShutdownViaManager
	runListFn          = instance.ListViaManager
	runGetFn           = instance.GetViaManager
	runExecFn          = instance.ExecViaManager
	runManagerStartFn  = instance.StartManager
	runManagerStopFn   = instance.StopManager
	runSessionDaemonFn = instance.RunSessionDaemon
	runManagerDaemonFn = instance.RunManagerDaemon
	sshBinaryOutputFn  = sshBinaryCombinedOutput
	sshBinaryLookPath  = exec.LookPath
	sshBinaryRunFn     = runSSHPassthrough
	scpBinaryRunFn     = runSCPPassthrough
)

type remoteParamDescriptor struct {
	ExecTimeout string `json:"exec_timeout"`
	SCPTimeout  string `json:"scp_timeout"`
}

func remoteParamDescriptors() []remoteParamDescriptor {
	return []remoteParamDescriptor{
		{
			ExecTimeout: "选填。SSH执行超时",
			SCPTimeout:  "选填。SCP执行超时",
		},
	}
}

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}

func runCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || connectsvc.IsHelpCommand(args[0]) {
		printHelp(stdout)
		return 0
	}

	command := strings.TrimSpace(args[0])
	switch command {
	case "help":
		printHelp(stdout)
		return 0
	case "name":
		connectsvc.WriteJSON(stdout, map[string]string{"key": remoteKey, "name": remoteDisplayName})
		return 0
	case "param":
		connectsvc.WriteJSON(stdout, remoteParamDescriptors())
		return 0
	case "scope":
		connectsvc.WriteJSON(stdout, []string{})
		return 0
	case "command":
		connectsvc.WriteJSON(stdout, remotePublicCommands())
		return 0
	case "exec":
		flags, remoteCommand, err := parseRemoteCommandArgs(args[1:])
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		result, err := runExecFn(flags, remoteCommand)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		if result.Stdout != "" {
			fmt.Fprint(stdout, result.Stdout)
		}
		if result.Stderr != "" {
			fmt.Fprint(stderr, result.Stderr)
		}
		return result.ExitCode
	case "start":
		flags, err := connectsvc.ParseFlags(args[1:])
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		if err := runManagerStartFn(flags); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		fmt.Fprintln(stdout, "OK")
		return 0
	case "stop":
		flags, err := connectsvc.ParseFlags(args[1:])
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		if err := runManagerStopFn(flags); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		fmt.Fprintln(stdout, "OK")
		return 0
	case "__daemon":
		flags, err := connectsvc.ParseFlags(args[1:])
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		if err := runSessionDaemonFn(flags); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		return 0
	case "__manager":
		flags, err := connectsvc.ParseFlags(args[1:])
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		if err := runManagerDaemonFn(flags); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		return 0
	case "create":
		flags, err := connectsvc.ParseFlags(args[1:])
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		item, err := runCreateFn(flags)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		connectsvc.WriteJSON(stdout, item)
		return 0
	case "shutdown":
		flags, err := connectsvc.ParseFlags(args[1:])
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		if err := runShutdownFn(flags); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		fmt.Fprintln(stdout, "OK")
		return 0
	case "list":
		flags, err := connectsvc.ParseFlags(args[1:])
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		items, err := runListFn(flags)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		connectsvc.WriteJSON(stdout, items)
		return 0
	case "get":
		flags, err := connectsvc.ParseFlags(args[1:])
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		item, err := runGetFn(flags)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		connectsvc.WriteJSON(stdout, item)
		return 0
	case "ssh":
		return sshBinaryRunFn(args[1:], stdout, stderr)
	case "scp":
		return scpBinaryRunFn(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", command)
		fmt.Fprintln(stderr, "run `remote help` for usage")
		return 1
	}
}

func remotePublicCommands() []string {
	items := []string{
		"command",
		"create",
		"exec",
		"get",
		"help",
		"list",
		"name",
		"param",
		"scp",
		"scope",
		"shutdown",
		"ssh",
		"start",
		"stop",
	}
	sort.Strings(items)
	return items
}
