//go:build !windows

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprint(os.Stderr, "CLI_SANDBOX_PICKER only supports Windows")
	os.Exit(2)
}
