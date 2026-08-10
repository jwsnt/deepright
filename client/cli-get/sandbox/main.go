package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"cli-get-sandbox/service"
)

func main() {
	var directCmd string
	var directTimeout int
	var shell string
	var logFile string
	var mode string
	var allowedDir string

	flag.StringVar(&shell, "shell", service.DefaultShell(), "shell used to execute delegated commands")
	flag.StringVar(&logFile, "log-file", "sandbox.log", "sandbox log file path")
	flag.StringVar(&mode, "mode", "", "sandbox mode: filepick, net, filepick_net")
	flag.StringVar(&allowedDir, "allowed-dir", "", "provide an allowed directory for filepick-based modes")
	flag.StringVar(&directCmd, "cmd", "", "execute a single command and print its output")
	flag.IntVar(&directTimeout, "timeout", 0, "command timeout in ms; 0 uses the default timeout")
	flag.Parse()

	logger := log.New(ioDiscard{}, "", 0)
	if path := strings.TrimSpace(logFile); path != "" {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sandbox init error: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		logger = log.New(file, "[cli-sandbox] ", log.LstdFlags|log.Lmicroseconds)
	}

	mode = service.NormalizeSandboxMode(mode)
	service.Debugf = logger.Printf
	if strings.TrimSpace(allowedDir) != "" {
		normalized, err := service.SetPickedDirectory(allowedDir)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			os.Exit(1)
		}
		if err := os.Setenv(service.SandboxAllowedDirEnv, normalized); err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			os.Exit(1)
		}
		logger.Printf("allowed directory set path=%q", normalized)
		if strings.TrimSpace(directCmd) == "" {
			fmt.Fprint(os.Stdout, normalized)
			return
		}
	}
	if strings.TrimSpace(directCmd) == "" {
		fmt.Fprintln(os.Stderr, "sandbox requires --cmd or --allowed-dir")
		os.Exit(1)
	}
	logger.Printf("command start mode=%s shell=%s timeoutMs=%d cmd=%q", mode, shell, directTimeout, directCmd)
	result := service.RunCommandWithMode(directCmd, shell, directTimeout, mode)
	logger.Printf("command finish status=%d output=%q", result.Status, result.Output)
	if result.Output != "" {
		if result.Status == 0 {
			fmt.Fprint(os.Stdout, result.Output)
		} else {
			fmt.Fprint(os.Stderr, result.Output)
		}
	}
	os.Exit(result.Status)
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}
