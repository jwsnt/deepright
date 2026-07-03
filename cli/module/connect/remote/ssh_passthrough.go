package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"connect/remote/instance"
	"connect/remote/runtimecfg"
)

func sshHelpText() string {
	sshPath, err := sshBinaryLookPath("ssh")
	if err != nil {
		return fallbackSSHHelp()
	}
	output, err := sshBinaryOutputFn(sshPath, []string{"-h"})
	if err != nil && strings.TrimSpace(output) == "" {
		return fallbackSSHHelp()
	}
	if strings.TrimSpace(output) == "" {
		return fallbackSSHHelp()
	}
	return strings.TrimSpace(output)
}

func fallbackSSHHelp() string {
	return "ssh [options] destination [command]\nCommon delegated options: -p, -i, -L, -R, -J, -o, -v, -A, -T"
}

type scpSessionInfo struct {
	Port        int
	ControlPath string
}

var resolveSCPControlPathFn = func(flags map[string]string) (scpSessionInfo, error) {
	record, controlPath, err := instance.ResolveSessionControlPath(flags)
	if err != nil {
		return scpSessionInfo{}, err
	}
	return scpSessionInfo{
		Port:        record.Port,
		ControlPath: controlPath,
	}, nil
}

func sshBinaryCombinedOutput(name string, args []string) (string, error) {
	cmd := exec.Command(name, args...)
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	err := cmd.Run()
	return combined.String(), err
}

func runSSHPassthrough(args []string, stdout, stderr io.Writer) int {
	sshPath, err := sshBinaryLookPath("ssh")
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return runBinaryPassthrough(sshPath, args, stdout, stderr)
}

func runSCPPassthrough(args []string, stdout, stderr io.Writer) int {
	scpPath, err := sshBinaryLookPath("scp")
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	flags, passthroughArgs := parseSCPArgs(args)
	scpArgs, err := buildSessionSCPArgs(flags, passthroughArgs)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return runBinaryPassthroughWithTimeout(scpPath, scpArgs, runtimecfg.ResolvePluginTimeouts(flags).SCP, stdout, stderr)
}

func buildSessionSCPArgs(flags map[string]string, passthroughArgs []string) ([]string, error) {
	sessionInfo, err := resolveSCPControlPathFn(flags)
	if err != nil {
		return nil, err
	}
	args := []string{
		"-o", "ControlMaster=auto",
		"-o", "ControlPath=" + sessionInfo.ControlPath,
		"-o", "ControlPersist=no",
		"-P", strconv.Itoa(sessionInfo.Port),
	}
	args = append(args, passthroughArgs...)
	return args, nil
}

func runBinaryPassthrough(binaryPath string, args []string, stdout, stderr io.Writer) int {
	cmd := exec.Command(binaryPath, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = nil
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return 0
}

func runBinaryPassthroughWithTimeout(binaryPath string, args []string, timeout time.Duration, stdout, stderr io.Writer) int {
	if timeout <= 0 {
		return runBinaryPassthrough(binaryPath, args, stdout, stderr)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = nil
	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			fmt.Fprintf(stderr, "command timed out after %ds\n", int(timeout/time.Second))
			return 124
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return 0
}
