package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

const (
	integrationCaffeinateFeishuPlugin = "feishu"
	integrationCaffeinateEmailPlugin  = "email"
)

type integrationSleepConditions struct {
	Plugin bool
	Memo   bool
	SSE    bool
	Task   bool
}

func (conditions integrationSleepConditions) Active() bool {
	return conditions.Plugin || conditions.Memo || conditions.SSE || conditions.Task
}

type integrationSleepManager struct {
	interval time.Duration

	mu         sync.Mutex
	closed     bool
	started    bool
	assertion  integrationSleepAssertion
	stopFailed bool
	activeSSE  int
	cancel     context.CancelFunc

	evaluateConditions func(activeSSE int) (integrationSleepConditions, error)
	startAssertion     func() (integrationSleepAssertion, error)
	logf               func(string, ...interface{})
	evaluateNow        func()
	evaluateSignal     chan struct{}
}

func newIntegrationSleepManager(interval time.Duration) *integrationSleepManager {
	manager := &integrationSleepManager{
		interval:           interval,
		evaluateConditions: integrationCurrentSleepConditions,
		startAssertion:     startIntegrationSleepAssertion,
		logf:               log.Printf,
		evaluateSignal:     make(chan struct{}, 1),
	}
	manager.evaluateNow = manager.evaluate
	return manager
}

func (m *integrationSleepManager) Start(ctx context.Context) {
	if m == nil || integrationRuntimeGOOS != "darwin" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	m.mu.Lock()
	if m.closed || m.started {
		m.mu.Unlock()
		return
	}
	managerCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.started = true
	m.mu.Unlock()

	go func() {
		m.evaluateNow()
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()
		for {
			select {
			case <-managerCtx.Done():
				return
			case <-ticker.C:
				m.evaluateNow()
			case <-m.evaluateSignal:
				m.evaluateNow()
			}
		}
	}()
}

func (m *integrationSleepManager) TrackSSE() func() {
	if m == nil {
		return func() {}
	}
	m.mu.Lock()
	if m.closed || !m.started {
		m.mu.Unlock()
		return func() {}
	}
	m.activeSSE++
	m.mu.Unlock()
	m.requestEvaluation()

	var done sync.Once
	return func() {
		done.Do(func() {
			m.mu.Lock()
			if m.activeSSE > 0 {
				m.activeSSE--
			}
			m.mu.Unlock()
			m.requestEvaluation()
		})
	}
}

func trackIntegrationSSE(cfg *Config) func() {
	if cfg == nil || cfg.sleepManager == nil {
		return func() {}
	}
	return cfg.sleepManager.TrackSSE()
}

func notifyIntegrationSleepConditionChanged(cfg *Config) {
	if cfg == nil || cfg.sleepManager == nil {
		return
	}
	cfg.sleepManager.requestEvaluation()
}

func (m *integrationSleepManager) Close() {
	if m == nil {
		return
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.closed = true
	cancel := m.cancel
	assertion := m.assertion
	m.assertion = nil
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if assertion != nil {
		if err := assertion.Stop(); err != nil {
			m.logf("integration: stop macOS sleep assertion failed: %v", err)
		}
	}
}

func (m *integrationSleepManager) requestEvaluation() {
	if m == nil {
		return
	}
	select {
	case m.evaluateSignal <- struct{}{}:
	default:
	}
}

func (m *integrationSleepManager) evaluate() {
	if m == nil {
		return
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	activeSSE := m.activeSSE
	m.mu.Unlock()

	conditions, err := m.evaluateConditions(activeSSE)
	if err != nil {
		m.logf("integration: evaluate macOS sleep assertion conditions failed: %v", err)
		return
	}
	if conditions.Active() {
		m.ensureAssertion()
		return
	}
	m.releaseAssertion()
}

func (m *integrationSleepManager) ensureAssertion() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	if m.stopFailed || integrationSleepAssertionRunning(m.assertion) {
		m.mu.Unlock()
		return
	}
	m.assertion = nil
	m.mu.Unlock()

	assertion, err := m.startAssertion()
	if err != nil {
		m.logf("integration: start macOS sleep assertion failed: %v", err)
		return
	}
	if assertion == nil {
		return
	}

	m.mu.Lock()
	if m.closed || integrationSleepAssertionRunning(m.assertion) {
		m.mu.Unlock()
		if err := assertion.Stop(); err != nil {
			m.logf("integration: stop macOS sleep assertion failed: %v", err)
		}
		return
	}
	m.assertion = assertion
	m.stopFailed = false
	m.mu.Unlock()
	m.logf("integration: macOS sleep assertion started")
}

func (m *integrationSleepManager) releaseAssertion() {
	m.mu.Lock()
	assertion := m.assertion
	m.assertion = nil
	m.stopFailed = false
	m.mu.Unlock()
	if assertion == nil {
		return
	}
	if err := assertion.Stop(); err != nil {
		m.logf("integration: stop macOS sleep assertion failed: %v", err)
		// Keep the assertion registered after a failed stop. It may still be
		// running, and retaining it prevents a later condition transition from
		// starting a second caffeinate process.
		m.mu.Lock()
		if !m.closed && m.assertion == nil {
			m.assertion = assertion
			m.stopFailed = true
		}
		m.mu.Unlock()
		return
	}
	m.logf("integration: macOS sleep assertion stopped")
}

func integrationSleepAssertionRunning(assertion integrationSleepAssertion) bool {
	if assertion == nil {
		return false
	}
	if status, ok := assertion.(interface{ Running() bool }); ok {
		return status.Running()
	}
	return true
}

func integrationCurrentSleepConditions(activeSSE int) (integrationSleepConditions, error) {
	conditions := integrationSleepConditions{SSE: activeSSE > 0}
	if conditions.SSE {
		return conditions, nil
	}

	pluginStarted, err := integrationCaffeinateMessagingPluginStarted()
	if err != nil {
		return conditions, err
	}
	conditions.Plugin = pluginStarted
	if conditions.Plugin {
		return conditions, nil
	}

	pendingMemo, err := integrationCaffeinatePendingMemo(time.Now())
	if err != nil {
		return conditions, err
	}
	conditions.Memo = pendingMemo
	if conditions.Memo {
		return conditions, nil
	}

	pendingTask, err := integrationCaffeinatePendingTask()
	if err != nil {
		return conditions, err
	}
	conditions.Task = pendingTask
	return conditions, nil
}

func integrationCaffeinateMessagingPluginStarted() (bool, error) {
	flags := integrationAssociatedPluginFlags()
	var firstErr error
	for _, plugin := range []string{integrationCaffeinateFeishuPlugin, integrationCaffeinateEmailPlugin} {
		status, err := runConnectPluginStatus(plugin, flags)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "plugin not found") {
				continue
			}
			if firstErr == nil {
				firstErr = fmt.Errorf("read %s plugin status: %w", plugin, err)
			}
			continue
		}
		if status != nil && status.Started {
			return true, nil
		}
	}
	return false, firstErr
}

func integrationCaffeinatePendingMemo(now time.Time) (bool, error) {
	if cronDB == nil {
		return false, nil
	}
	windowStart := now.Unix()
	windowEnd := now.Add(24 * time.Hour).Unix()
	var exists int
	err := cronDB.QueryRow(`SELECT EXISTS(
		SELECT 1 FROM task_detail
		WHERE task_type = 'cron'
		  AND started = 0
		  AND exec_time >= ?
		  AND exec_time < ?
	)`, windowStart, windowEnd).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("query pending memo in next 24 hours: %w", err)
	}
	return exists != 0, nil
}

func integrationCaffeinatePendingTask() (bool, error) {
	if cronDB == nil {
		return false, nil
	}
	var exists int
	err := cronDB.QueryRow(`SELECT EXISTS(
		SELECT 1 FROM wav2lip_task WHERE status IN (?, ?)
		UNION ALL SELECT 1 FROM rembg_task WHERE status IN (?, ?)
		UNION ALL SELECT 1 FROM voxcpm_task WHERE status IN (?, ?)
		UNION ALL SELECT 1 FROM rvm_task WHERE status IN (?, ?)
	)`,
		wav2lipTaskQueued, wav2lipTaskRunning,
		rembgTaskQueued, rembgTaskRunning,
		voxcpmTaskQueued, voxcpmTaskRunning,
		rvmTaskQueued, rvmTaskRunning,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("query queued or running media tasks: %w", err)
	}
	return exists != 0, nil
}

func parseIntegrationCaffeinateInterval(raw map[string]interface{}) (time.Duration, error) {
	if raw == nil {
		return 0, fmt.Errorf("config/config.json.caffeinate 配置缺失")
	}
	value, ok := raw["caffeinate"]
	if !ok {
		return 0, fmt.Errorf("config/config.json.caffeinate 配置缺失")
	}
	minutes, ok := value.(interface{ Int64() (int64, error) })
	if !ok {
		return 0, fmt.Errorf("config/config.json.caffeinate 必须是正整数（分钟）")
	}
	parsed, err := minutes.Int64()
	if err != nil || parsed <= 0 || parsed > int64(time.Duration(1<<63-1)/time.Minute) {
		return 0, fmt.Errorf("config/config.json.caffeinate 必须是正整数（分钟）")
	}
	return time.Duration(parsed) * time.Minute, nil
}

func readIntegrationCaffeinateInterval() (time.Duration, string, error) {
	raw, path, err := readIntegrationStartupConfigRaw()
	if err != nil {
		return 0, path, err
	}
	interval, err := parseIntegrationCaffeinateInterval(raw)
	if err != nil {
		if strings.TrimSpace(path) == "" {
			path = "config/config.json"
		}
		return 0, path, err
	}
	return interval, path, nil
}
