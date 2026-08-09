package main

import (
	"os"

	"connect/connectsvc"
)

func main() {
	os.Exit(connectsvc.RunCLI(os.Args[1:], os.Stdout, os.Stderr))
}
