package main

import "strings"

func parseSCPArgs(args []string) (map[string]string, []string) {
	flags := map[string]string{}
	passthrough := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			passthrough = append(passthrough, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "--") {
			passthrough = append(passthrough, arg)
			continue
		}
		key := strings.TrimPrefix(arg, "--")
		value := ""
		if parts := strings.SplitN(key, "=", 2); len(parts) == 2 {
			key = parts[0]
			value = parts[1]
		} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			value = args[i+1]
			i++
		}
		switch key {
		case "connect-bin", "timeout", "session", "remote":
			flags[key] = value
		default:
			passthrough = append(passthrough, arg)
			if value != "" && !strings.Contains(arg, "=") {
				passthrough = append(passthrough, value)
			}
		}
	}
	if strings.TrimSpace(flags["remote"]) == "" {
		if remote := extractSCPRemote(passthrough); remote != "" {
			flags["remote"] = remote
		}
	}
	return flags, passthrough
}

func extractSCPRemote(args []string) string {
	for _, arg := range args {
		if remote := parseSCPRemoteEndpoint(arg); remote != "" {
			return remote
		}
	}
	return ""
}

func parseSCPRemoteEndpoint(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") {
		return ""
	}
	colon := strings.Index(value, ":")
	if colon <= 0 {
		return ""
	}
	prefix := value[:colon]
	if strings.Contains(prefix, "/") || strings.Contains(prefix, "\\") || strings.HasPrefix(prefix, ".") || strings.HasPrefix(prefix, "~") {
		return ""
	}
	if !strings.Contains(prefix, "@") {
		return ""
	}
	return prefix
}
