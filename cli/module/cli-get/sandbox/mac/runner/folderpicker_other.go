//go:build !darwin

package main

import (
	"fmt"
	"time"
)

func sandboxChooseFolderWithTimeout(timeout time.Duration) (string, error) {
	return "", fmt.Errorf("sandbox directory picker only supports darwin")
}
