package main

import (
	"os"

	"knowledge/knowledgecli"
)

func main() {
	os.Exit(knowledgecli.Run(os.Args[1:], knowledgecli.Options{}, os.Stdout, os.Stderr))
}

func printUsage() {
	knowledgecli.PrintUsage(os.Stderr)
}

func runEnsure(args []string) {
	_ = knowledgecli.Run(append([]string{"ensure"}, args...), knowledgecli.Options{}, os.Stdout, os.Stderr)
}

func runGet(args []string) {
	_ = knowledgecli.Run(append([]string{"get"}, args...), knowledgecli.Options{}, os.Stdout, os.Stderr)
}

func runMetadata(args []string) {
	_ = knowledgecli.Run(append([]string{"metadata"}, args...), knowledgecli.Options{}, os.Stdout, os.Stderr)
}

func runUpdateTime(args []string) {
	_ = knowledgecli.Run(append([]string{"update-time"}, args...), knowledgecli.Options{}, os.Stdout, os.Stderr)
}
