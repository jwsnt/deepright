package main

import (
	"fmt"
	"strings"
)

func parseRemoteCommandArgs(args []string) (map[string]string, string, error) {
	flags := map[string]string{}
	commandArgs := make([]string, 0)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if len(commandArgs) > 0 {
			commandArgs = append(commandArgs, arg)
			continue
		}
		if arg == "--" {
			commandArgs = append(commandArgs, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "--") {
			commandArgs = append(commandArgs, arg)
			continue
		}
		key := strings.TrimPrefix(arg, "--")
		value := "true"
		if parts := strings.SplitN(key, "=", 2); len(parts) == 2 {
			key = parts[0]
			value = parts[1]
		} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			value = args[i+1]
			i++
		}
		flags[key] = value
	}
	remoteCommand := strings.TrimSpace(strings.Join(commandArgs, " "))
	if remoteCommand == "" {
		return nil, "", fmt.Errorf("remote exec command is required")
	}
	return flags, remoteCommand, nil
}
