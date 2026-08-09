package browserplaywrightsvc

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const playwrightDriverProbeTimeout = 2 * time.Second

var (
	playwrightDriverCommandContextFn   = exec.CommandContext
	playwrightDriverRuntimeGOOSFn      = func() string { return runtime.GOOS }
	playwrightDriverExecutableUsableFn = playwrightDriverExecutableUsable
)

func PlaywrightDriverReady(driverDir string) (bool, error) {
	return playwrightDriverReady(driverDir)
}

func playwrightDriverReady(driverDir string) (bool, error) {
	driverDir = strings.TrimSpace(driverDir)
	if driverDir == "" {
		return false, nil
	}
	info, err := os.Stat(driverDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !info.IsDir() {
		return false, nil
	}
	requiredFiles := []string{
		filepath.Join(driverDir, "package", "cli.js"),
		filepath.Join(driverDir, "package", "package.json"),
	}
	for _, path := range requiredFiles {
		item, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, err
		}
		if item.IsDir() {
			return false, nil
		}
	}
	nodePath, err := resolvePlaywrightNodePath(driverDir)
	if err != nil {
		return false, nil
	}
	if !playwrightDriverExecutableUsableFn(nodePath) {
		return false, nil
	}
	return true, nil
}

func playwrightDriverNodeFileName() string {
	if strings.EqualFold(strings.TrimSpace(playwrightDriverRuntimeGOOSFn()), "windows") {
		return "node.exe"
	}
	return "node"
}

func playwrightDriverExecutableUsable(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), playwrightDriverProbeTimeout)
	defer cancel()

	cmd := playwrightDriverCommandContextFn(ctx, path, "--version")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}
