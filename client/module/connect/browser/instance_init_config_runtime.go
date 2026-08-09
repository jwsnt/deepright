package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const browserDefaultInstanceInitTimeout = 300 * time.Second

type browserInstanceInitRuntimeConfig struct {
	Timeout    time.Duration
	ConfigPath string
}

// browserResolveInstanceInitRuntimeConfig deliberately uses only the runtime
// record written after a successful browser start. Init must never infer a
// config location from its own binary or the current working directory.
func browserResolveInstanceInitRuntimeConfig() (browserInstanceInitRuntimeConfig, error) {
	connectBin, started, err := browserReadRecordedConnectBin()
	if err != nil {
		return browserInstanceInitRuntimeConfig{}, fmt.Errorf("browser instance init requires a successful browser start: %w", err)
	}
	if !started {
		return browserInstanceInitRuntimeConfig{}, errors.New("browser instance init requires a successful browser start; run `browser start` first")
	}
	runtimePath, ok := browserResolveIntegrationRuntimePath(connectBin)
	if !ok {
		return browserInstanceInitRuntimeConfig{}, fmt.Errorf("integration config/config.json not found for browser start runtime: %s", connectBin)
	}
	data, err := browserReadFileFn(runtimePath)
	if err != nil {
		return browserInstanceInitRuntimeConfig{}, fmt.Errorf("read integration config %s: %w", runtimePath, err)
	}
	timeout, err := browserParseInstanceInitTimeout(data)
	if err != nil {
		return browserInstanceInitRuntimeConfig{}, fmt.Errorf("parse integration config %s: %w", runtimePath, err)
	}
	return browserInstanceInitRuntimeConfig{Timeout: timeout, ConfigPath: runtimePath}, nil
}

func browserParseInstanceInitTimeout(data []byte) (time.Duration, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return 0, err
	}
	if root == nil {
		return 0, errors.New("config root must be an object")
	}
	browserRaw, ok := root["browser"]
	if !ok {
		return browserDefaultInstanceInitTimeout, nil
	}
	if strings.TrimSpace(string(browserRaw)) == "null" {
		return 0, errors.New("browser must be an object")
	}
	var browserConfig map[string]json.RawMessage
	if err := json.Unmarshal(browserRaw, &browserConfig); err != nil {
		return 0, errors.New("browser must be an object")
	}
	initTimeoutRaw, ok := browserConfig["init_timeout"]
	if !ok {
		return browserDefaultInstanceInitTimeout, nil
	}
	var seconds int64
	if err := json.Unmarshal(initTimeoutRaw, &seconds); err != nil || seconds <= 0 {
		return 0, errors.New("browser.init_timeout must be a positive integer number of seconds")
	}
	if seconds > int64((time.Duration(1<<63-1))/time.Second) {
		return 0, errors.New("browser.init_timeout is too large")
	}
	return time.Duration(seconds) * time.Second, nil
}

func browserInstanceInitContextError(ctxDeadline <-chan struct{}) error {
	select {
	case <-ctxDeadline:
		return errors.New("browser instance init timed out")
	default:
		return nil
	}
}

func browserInstanceInitTimeoutSeconds(deadline time.Time) int64 {
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return 1
	}
	seconds := int64(remaining / time.Second)
	if remaining%time.Second != 0 {
		seconds++
	}
	return seconds
}
