package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

const (
	defaultModelTaskConcurrence = 1
	defaultModelTaskDownload    = 10
	defaultModelTaskTimeout     = 90
	defaultModelTaskRetry       = 3
)

// modelTaskQueue is the single execution gate shared by all local model
// tasks. A permit is acquired before a task record is moved from queued to
// running, so tasks still waiting for capacity remain visibly queued.
type modelTaskQueue struct {
	mu      sync.Mutex
	limit   int
	active  int
	waiters []*modelTaskQueueWaiter
}

type modelTaskQueueWaiter struct {
	ready chan struct{}
}

func newModelTaskQueue(concurrence int) *modelTaskQueue {
	if concurrence < 1 {
		concurrence = defaultModelTaskConcurrence
	}
	return &modelTaskQueue{limit: concurrence}
}

func (queue *modelTaskQueue) acquire(ctx context.Context) error {
	if queue == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	waiter := &modelTaskQueueWaiter{ready: make(chan struct{})}
	queue.mu.Lock()
	if queue.active < queue.limit && len(queue.waiters) == 0 {
		queue.active++
		queue.mu.Unlock()
		return nil
	}
	queue.waiters = append(queue.waiters, waiter)
	queue.mu.Unlock()

	select {
	case <-waiter.ready:
		return nil
	case <-ctx.Done():
		queue.mu.Lock()
		for index, candidate := range queue.waiters {
			if candidate != waiter {
				continue
			}
			queue.waiters = append(queue.waiters[:index], queue.waiters[index+1:]...)
			queue.mu.Unlock()
			return ctx.Err()
		}
		// The queue handed this waiter a permit while the context was being
		// cancelled. Return that permit before propagating cancellation.
		queue.mu.Unlock()
		queue.release()
		return ctx.Err()
	}
}

func (queue *modelTaskQueue) release() {
	if queue == nil {
		return
	}
	queue.mu.Lock()
	defer queue.mu.Unlock()
	if len(queue.waiters) > 0 {
		waiter := queue.waiters[0]
		queue.waiters = queue.waiters[1:]
		close(waiter.ready)
		return
	}
	if queue.active > 0 {
		queue.active--
	}
}

func parseModelTaskConcurrence(raw map[string]interface{}) (int, error) {
	if raw == nil {
		return defaultModelTaskConcurrence, nil
	}
	modelTaskRaw, exists := raw["modelTask"]
	if !exists || modelTaskRaw == nil {
		return defaultModelTaskConcurrence, nil
	}
	modelTask, ok := modelTaskRaw.(map[string]interface{})
	if !ok {
		return 0, errors.New("config/config.json.modelTask 必须是对象")
	}
	concurrenceRaw, exists := modelTask["concurrence"]
	if !exists {
		return 0, errors.New("config/config.json.modelTask.concurrence 缺失")
	}
	var concurrence int64
	switch value := concurrenceRaw.(type) {
	case json.Number:
		var err error
		concurrence, err = value.Int64()
		if err != nil {
			return 0, errors.New("config/config.json.modelTask.concurrence 必须是正整数")
		}
	case int:
		concurrence = int64(value)
	case int64:
		concurrence = value
	default:
		return 0, errors.New("config/config.json.modelTask.concurrence 必须是正整数")
	}
	maxInt := int64(^uint(0) >> 1)
	if concurrence < 1 || concurrence > maxInt {
		return 0, errors.New("config/config.json.modelTask.concurrence 必须是正整数")
	}
	return int(concurrence), nil
}

func parseModelTaskDownload(raw map[string]interface{}) (int, error) {
	if raw == nil {
		return defaultModelTaskDownload, nil
	}
	modelTaskRaw, exists := raw["modelTask"]
	if !exists || modelTaskRaw == nil {
		return defaultModelTaskDownload, nil
	}
	modelTask, ok := modelTaskRaw.(map[string]interface{})
	if !ok {
		return 0, errors.New("config/config.json.modelTask 必须是对象")
	}
	downloadRaw, exists := modelTask["download"]
	if !exists {
		return defaultModelTaskDownload, nil
	}
	var download int64
	switch value := downloadRaw.(type) {
	case json.Number:
		var err error
		download, err = value.Int64()
		if err != nil {
			return 0, errors.New("config/config.json.modelTask.download 必须是正整数")
		}
	case int:
		download = int64(value)
	case int64:
		download = value
	default:
		return 0, errors.New("config/config.json.modelTask.download 必须是正整数")
	}
	maxInt := int64(^uint(0) >> 1)
	if download < 1 || download > maxInt {
		return 0, errors.New("config/config.json.modelTask.download 必须是正整数")
	}
	return int(download), nil
}

func parseModelTaskRetry(raw map[string]interface{}) (int, error) {
	if raw == nil {
		return defaultModelTaskRetry, nil
	}
	modelTaskRaw, exists := raw["modelTask"]
	if !exists || modelTaskRaw == nil {
		return defaultModelTaskRetry, nil
	}
	modelTask, ok := modelTaskRaw.(map[string]interface{})
	if !ok {
		return 0, errors.New("config/config.json.modelTask 必须是对象")
	}
	retryRaw, exists := modelTask["retry"]
	if !exists {
		return defaultModelTaskRetry, nil
	}
	var retry int64
	switch value := retryRaw.(type) {
	case json.Number:
		var err error
		retry, err = value.Int64()
		if err != nil {
			return 0, errors.New("config/config.json.modelTask.retry 必须是非负整数")
		}
	case int:
		retry = int64(value)
	case int64:
		retry = value
	default:
		return 0, errors.New("config/config.json.modelTask.retry 必须是非负整数")
	}
	maxInt := int64(^uint(0) >> 1)
	if retry < 0 || retry > maxInt {
		return 0, errors.New("config/config.json.modelTask.retry 必须是非负整数")
	}
	return int(retry), nil
}

func parseModelTaskTimeout(raw map[string]interface{}) (int, error) {
	if raw == nil {
		return defaultModelTaskTimeout, nil
	}
	modelTaskRaw, exists := raw["modelTask"]
	if !exists || modelTaskRaw == nil {
		return defaultModelTaskTimeout, nil
	}
	modelTask, ok := modelTaskRaw.(map[string]interface{})
	if !ok {
		return 0, errors.New("config/config.json.modelTask 必须是对象")
	}
	timeoutRaw, exists := modelTask["timeout"]
	if !exists {
		return defaultModelTaskTimeout, nil
	}
	var timeout int64
	switch value := timeoutRaw.(type) {
	case json.Number:
		var err error
		timeout, err = value.Int64()
		if err != nil {
			return 0, errors.New("config/config.json.modelTask.timeout 必须是正整数秒")
		}
	case int:
		timeout = int64(value)
	case int64:
		timeout = value
	default:
		return 0, errors.New("config/config.json.modelTask.timeout 必须是正整数秒")
	}
	maxInt := int64(^uint(0) >> 1)
	if timeout < 1 || timeout > maxInt {
		return 0, errors.New("config/config.json.modelTask.timeout 必须是正整数秒")
	}
	return int(timeout), nil
}

func configureModelTaskQueue(cfg *Config) {
	if cfg == nil {
		return
	}
	concurrence := defaultModelTaskConcurrence
	download := defaultModelTaskDownload
	timeout := defaultModelTaskTimeout
	retry := defaultModelTaskRetry
	raw, _, err := readIntegrationStartupConfigRaw()
	if err == nil {
		concurrence, err = parseModelTaskConcurrence(raw)
	}
	if err != nil {
		log.Printf("[model-task] invalid config/config.json.modelTask.concurrence: %v; using %d", err, defaultModelTaskConcurrence)
		concurrence = defaultModelTaskConcurrence
	}
	if raw != nil {
		configuredDownload, downloadErr := parseModelTaskDownload(raw)
		if downloadErr != nil {
			log.Printf("[model-task] invalid config/config.json.modelTask.download: %v; using %d", downloadErr, defaultModelTaskDownload)
		} else {
			download = configuredDownload
		}
		configuredTimeout, timeoutErr := parseModelTaskTimeout(raw)
		if timeoutErr != nil {
			log.Printf("[model-task] invalid config/config.json.modelTask.timeout: %v; using %d", timeoutErr, defaultModelTaskTimeout)
		} else {
			timeout = configuredTimeout
		}
		configuredRetry, retryErr := parseModelTaskRetry(raw)
		if retryErr != nil {
			log.Printf("[model-task] invalid config/config.json.modelTask.retry: %v; using %d", retryErr, defaultModelTaskRetry)
		} else {
			retry = configuredRetry
		}
	}
	cfg.ModelTaskConcurrence = concurrence
	cfg.ModelTaskDownload = download
	cfg.ModelTaskTimeout = timeout
	cfg.ModelTaskRetry = retry
	cfg.modelTaskQueue = newModelTaskQueue(concurrence)
	log.Printf("[model-task] shared queue concurrency: %d; download segments: %d; download read timeout: %ds; download retries: %d", concurrence, download, timeout, retry)
}

func modelTaskDownloadWorkers(cfg *Config) int {
	if cfg != nil && cfg.ModelTaskDownload > 0 {
		return cfg.ModelTaskDownload
	}
	return defaultModelTaskDownload
}

func modelTaskDownloadRetries(cfg *Config) int {
	if cfg != nil && cfg.ModelTaskRetry >= 0 {
		return cfg.ModelTaskRetry
	}
	return defaultModelTaskRetry
}

func modelTaskDownloadReadTimeout(cfg *Config) time.Duration {
	if cfg != nil && cfg.ModelTaskTimeout > 0 {
		return time.Duration(cfg.ModelTaskTimeout) * time.Second
	}
	return time.Duration(defaultModelTaskTimeout) * time.Second
}

// reserveModelTaskSlot keeps a queued task in its table until the shared
// queue has capacity. table is always one of the five compile-time task table
// names below, never caller input.
func reserveModelTaskSlot(ctx context.Context, cfg *Config, db *sql.DB, table string) (bool, error) {
	if db == nil {
		return false, errors.New("task manager is not ready")
	}
	var query string
	switch table {
	case "whisper_task", "rembg_task", "rvm_task", "wav2lip_task", "voxcpm_task":
		query = fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE status = ? AND cancel_requested = 0)", table)
	default:
		return false, errors.New("unknown model task table")
	}
	var queued bool
	if err := db.QueryRow(query, "queued").Scan(&queued); err != nil {
		return false, err
	}
	if !queued {
		return false, nil
	}
	if cfg == nil || cfg.modelTaskQueue == nil {
		return true, nil
	}
	if err := cfg.modelTaskQueue.acquire(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func releaseModelTaskSlot(cfg *Config) {
	if cfg != nil {
		cfg.modelTaskQueue.release()
	}
}
