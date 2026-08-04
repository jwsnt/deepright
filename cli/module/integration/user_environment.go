package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Integration can be started by a launcher or GUI session whose PATH does not
// contain the user's Python or Homebrew directories. Resolve commands from the
// user's current shell environment first, then fall back to the service PATH.
// A failed lookup is deliberately not cached so an install completed by a cmd
// task becomes visible to the next dependency check without a restart.
var (
	integrationUserEnvironmentPathFn = integrationUserEnvironmentPath
	integrationServiceLookPathFn     = exec.LookPath
)

func integrationCommandLookPath(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", exec.ErrNotFound
	}
	if filepath.Base(name) != name {
		return integrationServiceLookPathFn(name)
	}
	for _, directory := range filepath.SplitList(integrationUserEnvironmentPathFn()) {
		directory = strings.TrimSpace(directory)
		if directory == "" {
			continue
		}
		candidate := filepath.Join(directory, name)
		info, err := os.Stat(candidate)
		if err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
			return candidate, nil
		}
	}
	return integrationServiceLookPathFn(name)
}

func integrationUserEnvironmentPath() string {
	if path := strings.TrimSpace(os.Getenv("DEEPRIGHT_USER_PATH")); path != "" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ""
	}

	if runtime.GOOS == "linux" && os.Geteuid() == 0 {
		// The service is commonly root in WSL while HOME points at deepright.
		// Read the user's shell PATH as that user rather than executing a user
		// controlled shell configuration with root privileges.
		username := strings.TrimSpace(os.Getenv("DEEPRIGHT_RUNTIME_USER"))
		if username == "" {
			username = filepath.Base(filepath.Clean(home))
		}
		if username != "" {
			return integrationShellPath("runuser", "-u", username, "--", "env", "HOME="+home, "/bin/bash", "--noprofile", "--norc", "-ic", `source "$HOME/.bashrc" >/dev/null 2>&1; printf '%s' "$PATH"`)
		}
		return ""
	}

	if runtime.GOOS == "darwin" {
		return integrationShellPath("/bin/zsh", "-ic", `source "$HOME/.zshrc" >/dev/null 2>&1; printf '%s' "$PATH"`)
	}
	return integrationShellPath("/bin/bash", "--noprofile", "--norc", "-ic", `source "$HOME/.bashrc" >/dev/null 2>&1; printf '%s' "$PATH"`)
}

func integrationShellPath(command string, args ...string) string {
	output, err := exec.Command(command, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
