package main

import (
	"connect/browserplaywrightsvc"
	"connect/connectsvc"
	"context"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	browserStateFileName         = "browser_instance.json"
	browserDefaultExpiredMinutes = 10
	browserDefaultHeadlessMode   = "new"
	browserUserDataModeClone     = "clone"
	browserUserDataModeDirect    = "direct"
	browserMinPort               = 20000
	browserMaxPort               = 65535
	browserPortRetryAttempts     = 8
	browserLogMaxSizeMB          = 10
	browserLogMaxFiles           = 4
)

type browserInstanceRecord struct {
	AgentID    string `json:"agentId"`
	ChatID     string `json:"chatId"`
	Port       int    `json:"port"`
	PID        int    `json:"pid"`
	CDP        string `json:"cdp"`
	ProfileDir string `json:"profileDir,omitempty"`
}

type browserInstanceStateRecord struct {
	AgentID      string `json:"agentId"`
	ChatID       string `json:"chatId"`
	Port         int    `json:"port"`
	PID          int    `json:"pid"`
	CDP          string `json:"cdp"`
	LastActiveAt string `json:"lastActiveAt,omitempty"`
}

var (
	browserExecutablePathFn                                = os.Executable
	browserRuntimeGOOSFn                                   = func() string { return runtime.GOOS }
	browserUserHomeDirFn                                   = os.UserHomeDir
	browserProcessExistsFn                                 = browserProcessExists
	browserTerminateProcessFn                              = browserTerminateProcess
	browserTerminateManagedInstanceFn                      = browserTerminateManagedInstance
	browserStartChromeFn                                   = browserStartChromeProcess
	browserWaitForPortFn                                   = browserWaitForPort
	browserWaitForCDPShutdownFn                            = browserWaitForCDPShutdown
	browserNowFn                                           = time.Now
	browserDialTimeoutFn                                   = net.DialTimeout
	browserPortAvailableFn                                 = browserIsPortAvailable
	browserPortCDPCheckFn                                  = browserIsCDPPort
	browserCDPVersionFn                                    = browserFetchCDPVersion
	browserPortPIDLookupFn                                 = browserLookupPIDByPort
	browserPrepareChromeUserDataDirFn                      = browserPrepareChromeUserDataDir
	browserResolveSystemChromeUserDataDirFn                = browserResolveSystemChromeUserDataDir
	browserStartChromeLogFilterFn                          = browserStartChromeLogFilter
	browserLogClosureEventFn                               = browserLogClosureEvent
	browserLoadAndPersistLiveInstancesWithExpiredMinutesFn = browserLoadAndPersistLiveInstancesWithExpiredMinutes
	browserAllocateCreatePortFn                            = browserAllocateCreatePort
	browserAllocateInitPortFn                              = browserAllocateInitPort
	browserAllocatePortFn                                  = browserAllocatePort
	browserRandomIntnFn                                    = browserRandomIntn
	browserAttachedCommandStartProcessFn                   = browserStartAttachedChromeProcessDirect
	browserAttachedCommandWaitForExitFn                    = browserWaitForChromeProcessExit
	browserResolveInstanceInitRuntimeConfigFn              = browserResolveInstanceInitRuntimeConfig
)

type browserCDPVersion struct {
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

type browserChromeProfilePaths struct {
	Local  string
	Launch string
}

func browserNormalizeIdentityPart(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func browserNormalizeIdentityFlags(flags map[string]string) map[string]string {
	if len(flags) == 0 {
		return map[string]string{}
	}
	next := make(map[string]string, len(flags))
	for key, value := range flags {
		next[key] = value
	}
	for _, key := range []string{"agentId", "agent", "chatId", "chat", "session"} {
		if value, ok := next[key]; ok {
			next[key] = browserNormalizeIdentityFlagValue(key, value)
		}
	}
	return next
}

func browserNormalizeIdentityFlagValue(key, value string) string {
	switch strings.TrimSpace(key) {
	case "agentId", "agent", "chatId", "chat":
		return browserNormalizeIdentityPart(value)
	case "session":
		return browserNormalizeManagedSession(value)
	default:
		return value
	}
}

func browserNormalizeManagedSession(session string) string {
	session = strings.TrimSpace(session)
	if session == "" {
		return ""
	}
	parts := strings.SplitN(session, "@", 2)
	if len(parts) != 2 {
		return strings.ToLower(session)
	}
	agentID := browserNormalizeIdentityPart(parts[0])
	chatID := browserNormalizeIdentityPart(parts[1])
	if agentID == "" || chatID == "" {
		return strings.ToLower(session)
	}
	return browserInstanceSessionName(agentID, chatID)
}

func browserCreateInstance(flags map[string]string) (browserInstanceRecord, error) {
	agentID, chatID, err := browserRequiredIdentity(flags)
	if err != nil {
		return browserInstanceRecord{}, err
	}
	if agentID == "" {
		return browserInstanceRecord{}, errors.New("agentId is required")
	}
	browserCreateTrace("instance.create.begin", map[string]any{
		"agentId": agentID,
		"chatId":  chatID,
	})

	statePath, err := browserResolveStatePath(flags)
	if err != nil {
		return browserInstanceRecord{}, err
	}
	browserCreateTrace("instance.create.state", map[string]any{
		"statePath": statePath,
	})
	items, err := browserLoadAndPersistLiveInstances(statePath)
	if err != nil {
		return browserInstanceRecord{}, err
	}
	browserCreateTrace("instance.create.loaded", map[string]any{
		"count": len(items),
	})
	if idx := browserFindInstanceIndex(items, agentID, chatID); idx >= 0 {
		items[idx].LastActiveAt = browserFormatActivityTime(browserNowFn())
		if err := browserSaveInstances(statePath, items); err != nil {
			return browserInstanceRecord{}, err
		}
		browserCreateTrace("instance.create.reuse", map[string]any{
			"agentId": agentID,
			"chatId":  chatID,
			"port":    items[idx].Port,
			"pid":     items[idx].PID,
			"cdp":     items[idx].CDP,
		})
		return browserInstanceAPIRecord(flags, items[idx]), nil
	}

	chromePath, err := browserResolveChromePath(flags)
	if err != nil {
		return browserInstanceRecord{}, err
	}
	headlessMode, err := browserResolveChromeHeadlessMode(flags)
	if err != nil {
		return browserInstanceRecord{}, err
	}
	isWSL, _ := browserWSLDetectFn()
	logPath := ""
	if !isWSL {
		logPath, err = browserDefaultLogPath()
		if err != nil {
			return browserInstanceRecord{}, err
		}
	}
	var (
		port         int
		pid          int
		cdp          string
		profilePaths browserChromeProfilePaths
	)
	if isWSL {
		browserCreateTrace("instance.create.new.wsl", map[string]any{
			"agentId":      agentID,
			"chatId":       chatID,
			"chromePath":   chromePath,
			"headlessMode": headlessMode,
		})
		wslRecord, err := browserAcquireWSLManagedInstanceWithLauncher(context.Background(), flags, agentID, chatID, headlessMode != "none", false, chromePath)
		if err != nil {
			return browserInstanceRecord{}, err
		}
		item := browserInstanceStateRecord{
			AgentID:      agentID,
			ChatID:       chatID,
			Port:         wslRecord.Port,
			PID:          wslRecord.PID,
			CDP:          wslRecord.WS,
			LastActiveAt: browserFormatActivityTime(browserNowFn()),
		}
		items = append(items, item)
		browserSortInstances(items)
		if err := browserSaveInstances(statePath, items); err != nil {
			return browserInstanceRecord{}, err
		}
		record := browserInstanceAPIRecord(flags, item)
		if strings.TrimSpace(wslRecord.UserDataDir) != "" {
			record.ProfileDir = wslRecord.UserDataDir
		}
		browserCreateTrace("instance.create.new.ready", map[string]any{
			"agentId":    item.AgentID,
			"chatId":     item.ChatID,
			"port":       item.Port,
			"pid":        item.PID,
			"cdp":        item.CDP,
			"profileDir": record.ProfileDir,
		})
		return record, nil
	}
	port, err = browserAllocateCreatePortFn(agentID, chatID, browserAPIRecords(items), nil)
	if err != nil {
		return browserInstanceRecord{}, err
	}
	browserCreateTrace("instance.create.new.port", map[string]any{
		"port":    port,
		"attempt": 1,
	})
	profilePaths, err = browserResolveChromeProfileDir(flags, agentID, port)
	if err != nil {
		return browserInstanceRecord{}, err
	}
	browserCreateTrace("instance.create.new.chrome", map[string]any{
		"chromePath":       chromePath,
		"headlessMode":     headlessMode,
		"profileDir":       profilePaths.Local,
		"launchProfileDir": profilePaths.Launch,
		"userDataMode":     browserResolveChromeUserDataMode(flags),
		"command":          browserFormatCommandLine(chromePath, browserChromeLaunchArgs(port, profilePaths.Launch, headlessMode)),
		"attempt":          1,
	})
	if err := browserPrepareChromeUserDataDirFn(flags, profilePaths.Local); err != nil {
		return browserInstanceRecord{}, err
	}
	pid, err = browserStartChromeFn(chromePath, port, profilePaths.Launch, headlessMode, logPath)
	if err != nil {
		return browserInstanceRecord{}, err
	}
	browserCreateTrace("instance.create.new.started", map[string]any{
		"pid":     pid,
		"port":    port,
		"attempt": 1,
	})
	if err = browserWaitForPortFn(pid, port, browserChromeStartupTimeout()); err != nil {
		_ = browserTerminateProcessFn(pid)
		return browserInstanceRecord{}, err
	}
	cdp, err = browserResolveLiveCDPEndpoint(port)
	if err != nil {
		_ = browserTerminateProcessFn(pid)
		return browserInstanceRecord{}, err
	}

	item := browserInstanceStateRecord{
		AgentID:      agentID,
		ChatID:       chatID,
		Port:         port,
		PID:          pid,
		CDP:          cdp,
		LastActiveAt: browserFormatActivityTime(browserNowFn()),
	}
	items = append(items, item)
	browserSortInstances(items)
	if err := browserSaveInstances(statePath, items); err != nil {
		_ = browserTerminateProcessFn(pid)
		return browserInstanceRecord{}, err
	}

	browserCreateTrace("instance.create.new.ready", map[string]any{
		"agentId": item.AgentID,
		"chatId":  item.ChatID,
		"port":    item.Port,
		"pid":     item.PID,
		"cdp":     item.CDP,
	})
	return browserInstanceAPIRecord(flags, item), nil
}

func browserShutdownInstance(flags map[string]string) error {
	agentID, chatID, err := browserRequiredIdentity(flags)
	if err != nil {
		return err
	}
	statePath, err := browserResolveStatePath(flags)
	if err != nil {
		return err
	}
	items, err := browserLoadAndPersistLiveInstances(statePath)
	if err != nil {
		return err
	}

	idx := browserFindInstanceIndex(items, agentID, chatID)
	if idx >= 0 {
		item := browserInstanceAPIRecord(flags, items[idx])
		if err := browserTerminateManagedInstanceFn(item); err != nil {
			return err
		}
		browserLogClosureEventFn("shutdown", map[string]any{
			"agentId": agentID,
			"chatId":  chatID,
			"pid":     item.PID,
			"port":    item.Port,
			"cdp":     item.CDP,
		})
		return browserRemoveInstance(statePath, agentID, chatID, 0)
	}
	return browserRemoveInstance(statePath, agentID, chatID, 0)
}

func browserInitInstance(flags map[string]string) (browserInstanceRecord, error) {
	next := normalizeBrowserIdentityFlags(flags)
	agentID, chatID, err := browserRequiredIdentity(next)
	if err != nil {
		return browserInstanceRecord{}, err
	}
	if agentID == "" {
		return browserInstanceRecord{}, errors.New("agentId is required")
	}
	next["agentId"] = agentID
	next["chatId"] = chatID
	runtimeConfig, err := browserResolveInstanceInitRuntimeConfigFn()
	if err != nil {
		return browserInstanceRecord{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), runtimeConfig.Timeout)
	defer cancel()
	browserCreateTrace("instance.init.config", map[string]any{
		"agentId":    agentID,
		"chatId":     chatID,
		"configPath": runtimeConfig.ConfigPath,
		"timeout":    runtimeConfig.Timeout.String(),
	})
	if err := browserInstanceInitContextError(ctx.Done()); err != nil {
		return browserInstanceRecord{}, err
	}
	if _, err := instanceGetFn(next); err != nil {
		if !browserIsInstanceNotFoundError(err) {
			return browserInstanceRecord{}, err
		}
	}
	browserLogInstanceShutdownRequest("instance_init_replace_existing", next, nil)
	if err := browserInvokeInstanceShutdown(next); err != nil && !browserIsInstanceNotFoundError(err) {
		return browserInstanceRecord{}, err
	}
	if err := browserInstanceInitContextError(ctx.Done()); err != nil {
		return browserInstanceRecord{}, err
	}
	return browserCreateAttachedInstance(ctx, next, agentID, chatID)
}

func browserDestroyInstance(flags map[string]string) error {
	if browserDestroyUsesDefaultStartPort(flags) {
		return browserShutdownDefaultBootstrapCDP(flags)
	}
	cleanupUserData := browserFlagEnabled(flags, "cleanup-user-data")
	cleanupAllAgentUserData := browserFlagEnabled(flags, "cleanup-all-agent-user-data")
	cleanupStateFile := browserFlagEnabled(flags, "cleanup-state-file")
	agentID, chatID, err := browserRequiredIdentity(flags)
	if err != nil {
		return err
	}
	statePath, err := browserResolveStatePath(flags)
	if err != nil {
		return err
	}
	if cleanupStateFile {
		defer browserDestroyCleanupStateFile(statePath)
	}
	browserShutdownTrace("instance.shutdown.begin", map[string]any{
		"agentId":   agentID,
		"chatId":    chatID,
		"statePath": statePath,
	})

	item, err := instanceGetFn(flags)
	if err != nil {
		browserShutdownTrace("instance.shutdown.error", map[string]any{
			"agentId": agentID,
			"chatId":  chatID,
			"stage":   "get",
			"error":   err.Error(),
		})
		fallbackItems := browserDestroyMissingStateFallbackItems(agentID, chatID)
		if cleanupUserData || cleanupAllAgentUserData {
			cleanupFields := map[string]any{
				"agentId":     agentID,
				"chatId":      chatID,
				"statePath":   statePath,
				"cleanupOnly": true,
			}
			removedProfileDirs := []string{}
			if cleanupUserData {
				for _, fallbackItem := range fallbackItems {
					profileDir, cleanupErr := browserDestroyCleanupUserDataDir(flags, fallbackItem)
					if cleanupErr != nil {
						browserShutdownTrace("instance.shutdown.cleanup.warn", map[string]any{
							"agentId": agentID,
							"chatId":  chatID,
							"port":    fallbackItem.Port,
							"stage":   "cleanup_user_data",
							"error":   cleanupErr.Error(),
						})
						continue
					}
					removedProfileDirs = browserAppendUniqueStrings(removedProfileDirs, profileDir)
				}
			}
			if len(removedProfileDirs) == 1 {
				cleanupFields["profileDir"] = removedProfileDirs[0]
			} else if len(removedProfileDirs) > 1 {
				cleanupFields["profileDirs"] = removedProfileDirs
			}
			if cleanupAllAgentUserData {
				removedDirs, cleanupErr := browserDestroyCleanupAllAgentUserDataDirs(flags)
				if cleanupErr != nil {
					browserShutdownTrace("instance.shutdown.cleanup.warn", map[string]any{
						"agentId": agentID,
						"chatId":  chatID,
						"stage":   "cleanup_all_agent_user_data",
						"error":   cleanupErr.Error(),
					})
				} else if len(removedDirs) > 0 {
					cleanupFields["agentProfileDirs"] = removedDirs
				}
			}
			browserShutdownTrace("instance.shutdown.cleanup.ok", cleanupFields)
			return nil
		}
		cleaned, releaseErr := browserDestroyReleaseMissingStateFallbackItems(fallbackItems)
		if releaseErr != nil {
			return releaseErr
		}
		if cleaned {
			browserShutdownTrace("instance.shutdown.cleanup.ok", map[string]any{
				"agentId":     agentID,
				"chatId":      chatID,
				"statePath":   statePath,
				"cleanupOnly": false,
				"fallback":    true,
			})
			return nil
		}
		return err
	}
	browserShutdownTrace("instance.shutdown.get.ok", map[string]any{
		"agentId": item.AgentID,
		"chatId":  item.ChatID,
		"pid":     item.PID,
		"port":    item.Port,
		"cdp":     item.CDP,
	})
	if err := browserTerminateManagedInstanceFn(item); err != nil {
		browserShutdownTrace("instance.shutdown.error", map[string]any{
			"agentId": item.AgentID,
			"chatId":  item.ChatID,
			"pid":     item.PID,
			"port":    item.Port,
			"cdp":     item.CDP,
			"stage":   "terminate",
			"error":   err.Error(),
		})
		return err
	}
	if err := browserRemoveInstance(statePath, agentID, chatID, item.PID); err != nil {
		browserShutdownTrace("instance.shutdown.error", map[string]any{
			"agentId": item.AgentID,
			"chatId":  item.ChatID,
			"pid":     item.PID,
			"port":    item.Port,
			"cdp":     item.CDP,
			"stage":   "cleanup_state",
			"error":   err.Error(),
		})
		return err
	}
	cleanupFields := map[string]any{
		"agentId": item.AgentID,
		"chatId":  item.ChatID,
		"pid":     item.PID,
		"port":    item.Port,
		"cdp":     item.CDP,
	}
	if cleanupUserData {
		profileDir, err := browserDestroyCleanupUserDataDir(flags, item)
		if err != nil {
			browserShutdownTrace("instance.shutdown.cleanup.warn", map[string]any{
				"agentId": item.AgentID,
				"chatId":  item.ChatID,
				"pid":     item.PID,
				"port":    item.Port,
				"cdp":     item.CDP,
				"stage":   "cleanup_user_data",
				"error":   err.Error(),
			})
		} else if strings.TrimSpace(profileDir) != "" {
			cleanupFields["profileDir"] = profileDir
		}
	}
	if cleanupAllAgentUserData {
		removedDirs, err := browserDestroyCleanupAllAgentUserDataDirs(flags)
		if err != nil {
			browserShutdownTrace("instance.shutdown.cleanup.warn", map[string]any{
				"agentId": item.AgentID,
				"chatId":  item.ChatID,
				"pid":     item.PID,
				"port":    item.Port,
				"cdp":     item.CDP,
				"stage":   "cleanup_all_agent_user_data",
				"error":   err.Error(),
			})
		} else if len(removedDirs) > 0 {
			cleanupFields["agentProfileDirs"] = removedDirs
		}
	}
	browserShutdownTrace("instance.shutdown.cleanup.ok", cleanupFields)
	browserLogClosureEventFn("shutdown", map[string]any{
		"agentId": item.AgentID,
		"chatId":  item.ChatID,
		"pid":     item.PID,
		"port":    item.Port,
		"cdp":     item.CDP,
	})
	return nil
}

func browserIsInstanceNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(err.Error())), "instance not found:")
}

func browserCreateAttachedInstance(ctx context.Context, flags map[string]string, agentID, chatID string) (browserInstanceRecord, error) {
	if err := browserInstanceInitContextError(ctx.Done()); err != nil {
		return browserInstanceRecord{}, err
	}
	browserCreateTrace("instance.init.begin", map[string]any{
		"agentId": agentID,
		"chatId":  chatID,
	})
	statePath, err := browserResolveStatePath(flags)
	if err != nil {
		return browserInstanceRecord{}, err
	}
	items, err := browserLoadAndPersistLiveInstances(statePath)
	if err != nil {
		return browserInstanceRecord{}, err
	}
	chromePath, err := browserResolveChromePath(flags)
	if err != nil {
		return browserInstanceRecord{}, err
	}
	isWSL, _ := browserWSLDetectFn()
	logPath := ""
	if !isWSL {
		logPath, err = browserDefaultLogPath()
		if err != nil {
			return browserInstanceRecord{}, err
		}
	}

	var (
		port         int
		pid          int
		cdp          string
		profilePaths browserChromeProfilePaths
		waitFn       func() error
	)
	if isWSL {
		wslRecord, err := browserAcquireWSLManagedInstanceWithLauncher(ctx, flags, agentID, chatID, false, true, chromePath)
		if err != nil {
			if timeoutErr := browserInstanceInitContextError(ctx.Done()); timeoutErr != nil {
				browserBestEffortCleanupTimedOutWSLInit(agentID, chatID)
				return browserInstanceRecord{}, timeoutErr
			}
			return browserInstanceRecord{}, err
		}
		if err := browserInstanceInitContextError(ctx.Done()); err != nil {
			_ = browserTerminateManagedInstanceFn(browserInstanceRecord{PID: wslRecord.PID, Port: wslRecord.Port})
			return browserInstanceRecord{}, err
		}
		browserCreateTrace("instance.init.launch", map[string]any{
			"agentId":    agentID,
			"chatId":     chatID,
			"chromePath": chromePath,
			"port":       wslRecord.Port,
			"attempt":    1,
		})
		item := browserInstanceStateRecord{
			AgentID:      agentID,
			ChatID:       chatID,
			Port:         wslRecord.Port,
			PID:          wslRecord.PID,
			CDP:          wslRecord.WS,
			LastActiveAt: browserFormatActivityTime(browserNowFn()),
		}
		if err := browserInstanceInitContextError(ctx.Done()); err != nil {
			_ = browserTerminateManagedInstanceFn(browserInstanceRecord{PID: wslRecord.PID, Port: wslRecord.Port})
			return browserInstanceRecord{}, err
		}
		items = browserUpsertInstanceState(items, item)
		if err := browserSaveInstances(statePath, items); err != nil {
			return browserInstanceRecord{}, err
		}
		apiRecord := browserInstanceAPIRecord(flags, item)
		if strings.TrimSpace(wslRecord.UserDataDir) != "" {
			apiRecord.ProfileDir = wslRecord.UserDataDir
		}
		browserCreateTrace("instance.init.ready", map[string]any{
			"agentId": item.AgentID,
			"chatId":  item.ChatID,
			"pid":     item.PID,
			"port":    item.Port,
			"cdp":     item.CDP,
		})
		return apiRecord, nil
	}
	port, err = browserAllocateInitPortFn(agentID, chatID, browserAPIRecords(items), nil)
	if err != nil {
		return browserInstanceRecord{}, err
	}
	profilePaths, err = browserResolveChromeProfileDir(flags, agentID, port)
	if err != nil {
		return browserInstanceRecord{}, err
	}
	if browserRuntimeGOOSFn() == "darwin" {
		if err := browserPrepareChromeUserDataDirWithContext(ctx, flags, profilePaths.Local); err != nil {
			return browserInstanceRecord{}, err
		}
		launchArgs := browserInitChromeLaunchArgs(port, profilePaths.Launch)
		browserCreateTrace("instance.init.launch.command", map[string]any{
			"agentId":          agentID,
			"chatId":           chatID,
			"chromePath":       chromePath,
			"profileDir":       profilePaths.Local,
			"launchProfileDir": profilePaths.Launch,
			"port":             port,
			"command":          browserFormatCommandLine(chromePath, launchArgs),
			"attempt":          1,
		})
		pid, waitFn, err = browserAttachedCommandStartProcessFn(chromePath, launchArgs, logPath)
		if err != nil {
			return browserInstanceRecord{}, err
		}
		if waitFn == nil {
			waitFn = func() error { return nil }
		}
		if err = browserWaitForPortFn(pid, port, browserInstanceInitRemainingTimeout(ctx)); err != nil {
			_ = browserTerminateProcessFn(pid)
			_ = waitFn()
			if timeoutErr := browserInstanceInitContextError(ctx.Done()); timeoutErr != nil {
				return browserInstanceRecord{}, timeoutErr
			}
			return browserInstanceRecord{}, err
		}
		if err := browserInstanceInitContextError(ctx.Done()); err != nil {
			_ = browserTerminateProcessFn(pid)
			_ = waitFn()
			return browserInstanceRecord{}, err
		}
		cdp, err = browserResolveLiveCDPEndpoint(port)
		if err != nil {
			_ = browserTerminateProcessFn(pid)
			_ = waitFn()
			return browserInstanceRecord{}, err
		}

		item := browserInstanceStateRecord{
			AgentID:      agentID,
			ChatID:       chatID,
			Port:         port,
			PID:          pid,
			CDP:          cdp,
			LastActiveAt: browserFormatActivityTime(browserNowFn()),
		}
		if err := browserInstanceInitContextError(ctx.Done()); err != nil {
			_ = browserTerminateProcessFn(pid)
			_ = waitFn()
			return browserInstanceRecord{}, err
		}
		items = browserUpsertInstanceState(items, item)
		if err := browserSaveInstances(statePath, items); err != nil {
			_ = browserTerminateProcessFn(pid)
			_ = waitFn()
			return browserInstanceRecord{}, err
		}

		apiRecord := browserInstanceAPIRecord(flags, item)
		browserCreateTrace("instance.init.ready", map[string]any{
			"agentId": item.AgentID,
			"chatId":  item.ChatID,
			"pid":     item.PID,
			"port":    item.Port,
			"cdp":     item.CDP,
		})
		return apiRecord, nil
	}

	browserCreateTrace("instance.init.launch", map[string]any{
		"agentId":          agentID,
		"chatId":           chatID,
		"chromePath":       chromePath,
		"profileDir":       profilePaths.Local,
		"launchProfileDir": profilePaths.Launch,
		"port":             port,
		"attempt":          1,
	})
	if err := browserPrepareChromeUserDataDirWithContext(ctx, flags, profilePaths.Local); err != nil {
		return browserInstanceRecord{}, err
	}
	launchArgs := browserInitChromeLaunchArgs(port, profilePaths.Launch)
	browserCreateTrace("instance.init.launch.command", map[string]any{
		"agentId":          agentID,
		"chatId":           chatID,
		"chromePath":       chromePath,
		"profileDir":       profilePaths.Local,
		"launchProfileDir": profilePaths.Launch,
		"port":             port,
		"command":          browserFormatCommandLine(chromePath, launchArgs),
		"attempt":          1,
	})
	pid, waitFn, err = browserAttachedCommandStartProcessFn(chromePath, launchArgs, logPath)
	if err != nil {
		return browserInstanceRecord{}, err
	}
	if waitFn == nil {
		waitFn = func() error { return nil }
	}
	if err = browserWaitForPortFn(pid, port, browserInstanceInitRemainingTimeout(ctx)); err != nil {
		_ = browserTerminateProcessFn(pid)
		_ = waitFn()
		if timeoutErr := browserInstanceInitContextError(ctx.Done()); timeoutErr != nil {
			return browserInstanceRecord{}, timeoutErr
		}
		return browserInstanceRecord{}, err
	}
	if err := browserInstanceInitContextError(ctx.Done()); err != nil {
		_ = browserTerminateProcessFn(pid)
		_ = waitFn()
		return browserInstanceRecord{}, err
	}
	cdp, err = browserResolveLiveCDPEndpoint(port)
	if err != nil {
		_ = browserTerminateProcessFn(pid)
		_ = waitFn()
		return browserInstanceRecord{}, err
	}

	item := browserInstanceStateRecord{
		AgentID:      agentID,
		ChatID:       chatID,
		Port:         port,
		PID:          pid,
		CDP:          cdp,
		LastActiveAt: browserFormatActivityTime(browserNowFn()),
	}
	if err := browserInstanceInitContextError(ctx.Done()); err != nil {
		_ = browserTerminateProcessFn(pid)
		_ = waitFn()
		return browserInstanceRecord{}, err
	}
	items = browserUpsertInstanceState(items, item)
	if err := browserSaveInstances(statePath, items); err != nil {
		_ = browserTerminateProcessFn(pid)
		if !isWSL {
			_ = waitFn()
		}
		return browserInstanceRecord{}, err
	}

	apiRecord := browserInstanceAPIRecord(flags, item)
	browserCreateTrace("instance.init.ready", map[string]any{
		"agentId": item.AgentID,
		"chatId":  item.ChatID,
		"port":    item.Port,
		"pid":     item.PID,
		"cdp":     item.CDP,
	})
	return apiRecord, nil
}

func browserBestEffortCleanupTimedOutWSLInit(agentID, chatID string) {
	item, found := browserWSLInstanceLookupRecordFn(agentID, chatID)
	if !found || item.PID <= 0 {
		return
	}
	_ = browserWSLTerminateProcessFn(item.PID, item.Port)
}

func browserDestroyInstanceAlreadyClosed(item browserInstanceRecord) bool {
	if item.Port > 0 {
		if err := browserWaitForCDPShutdownFn(item.Port, 2*time.Second); err != nil {
			return false
		}
	}
	if item.PID <= 0 {
		return true
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		if !browserProcessExistsFn(item.PID) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func browserDestroyCleanupUserDataDir(flags map[string]string, item browserInstanceRecord) (string, error) {
	profileDir, err := browserResolveDestroyChromeProfileDir(flags, item)
	if err != nil {
		return "", err
	}
	profileDir = strings.TrimSpace(profileDir)
	if profileDir == "" {
		return "", nil
	}
	if isWSL, err := browserWSLDetectFn(); err == nil && isWSL {
		return "", nil
	}
	browserShutdownTrace("instance.shutdown.user_data.begin", map[string]any{
		"agentId":    item.AgentID,
		"chatId":     item.ChatID,
		"port":       item.Port,
		"profileDir": profileDir,
	})
	if err := browserRemoveAllFn(profileDir); err != nil && !errors.Is(err, os.ErrNotExist) {
		return profileDir, err
	}
	browserShutdownTrace("instance.shutdown.user_data.ok", map[string]any{
		"agentId":    item.AgentID,
		"chatId":     item.ChatID,
		"port":       item.Port,
		"profileDir": profileDir,
	})
	return profileDir, nil
}

func browserDestroyCleanupAllAgentUserDataDirs(flags map[string]string) ([]string, error) {
	if isWSL, err := browserWSLDetectFn(); err == nil && isWSL {
		return nil, nil
	}
	cleanupRoots, err := browserResolveManagedChromeCleanupRoots(flags)
	if err != nil {
		return nil, err
	}
	if len(cleanupRoots) == 0 {
		return nil, nil
	}
	removed := []string{}
	for _, cleanupRoot := range cleanupRoots {
		rootRemoved, err := browserDestroyCleanupManagedChromeProfileDirs(cleanupRoot)
		if err != nil {
			return removed, err
		}
		removed = browserAppendUniqueStrings(removed, rootRemoved...)
	}
	return removed, nil
}

func browserResolveManagedChromeCleanupRoots(flags map[string]string) ([]string, error) {
	roots := []string{}
	agentRoot, err := browserResolveAgentRoot(flags)
	if err != nil {
		return nil, err
	}
	if agentRoot = strings.TrimSpace(agentRoot); agentRoot != "" {
		roots = append(roots, agentRoot)
	}
	if isWSL, err := browserWSLDetectFn(); err == nil && isWSL {
		wslRoot, wslErr := browserResolveWSLManagedProfileCleanupRoot()
		if wslErr != nil {
			return nil, wslErr
		}
		if wslRoot = strings.TrimSpace(wslRoot); wslRoot != "" {
			roots = append(roots, wslRoot)
		}
	}
	return browserAppendUniqueStrings(nil, roots...), nil
}

func browserDestroyCleanupManagedChromeProfileDirs(cleanupRoot string) ([]string, error) {
	cleanupRoot = strings.TrimSpace(cleanupRoot)
	if cleanupRoot == "" {
		return nil, nil
	}
	browserShutdownTrace("instance.shutdown.agent_root_user_data.begin", map[string]any{
		"cleanupRoot": cleanupRoot,
	})
	targets := []string{}
	walkErr := filepath.WalkDir(cleanupRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, os.ErrNotExist) {
				return nil
			}
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		if path != cleanupRoot && browserIsManagedChromeProfileDirName(d.Name()) {
			targets = append(targets, path)
			return filepath.SkipDir
		}
		return nil
	})
	if walkErr != nil {
		if errors.Is(walkErr, os.ErrNotExist) {
			browserShutdownTrace("instance.shutdown.agent_root_user_data.ok", map[string]any{
				"cleanupRoot":  cleanupRoot,
				"removedCount": 0,
			})
			return nil, nil
		}
		return nil, walkErr
	}
	removed := []string{}
	var removedMu sync.Mutex
	var wg sync.WaitGroup
	for _, profileDir := range targets {
		profileDir := profileDir
		wg.Add(1)
		go func() {
			defer wg.Done()
			parentDir := filepath.Dir(profileDir)
			profileName := filepath.Base(profileDir)
			browserShutdownTrace("instance.shutdown.agent_root_user_data.profile.begin", map[string]any{
				"parentDir":   parentDir,
				"profileDir":  profileDir,
				"profileName": profileName,
			})
			if err := browserRemoveAllFn(profileDir); err != nil && !errors.Is(err, os.ErrNotExist) {
				browserShutdownTrace("instance.shutdown.agent_root_user_data.profile.warn", map[string]any{
					"parentDir":   parentDir,
					"profileDir":  profileDir,
					"profileName": profileName,
					"error":       err.Error(),
				})
				return
			}
			removedMu.Lock()
			removed = append(removed, profileDir)
			removedMu.Unlock()
			browserShutdownTrace("instance.shutdown.agent_root_user_data.profile.ok", map[string]any{
				"parentDir":   parentDir,
				"profileDir":  profileDir,
				"profileName": profileName,
			})
		}()
	}
	wg.Wait()
	sort.Strings(removed)
	fields := map[string]any{
		"cleanupRoot":  cleanupRoot,
		"removedCount": len(removed),
	}
	if len(removed) > 0 {
		fields["removed"] = removed
	}
	browserShutdownTrace("instance.shutdown.agent_root_user_data.ok", fields)
	return removed, nil
}

func browserIsManagedChromeProfileDirName(name string) bool {
	if !strings.HasPrefix(name, "chrome_") {
		return false
	}
	suffix := strings.TrimPrefix(name, "chrome_")
	if suffix == "" {
		return false
	}
	allDigits := true
	allLowerAlphaNum := true
	for _, ch := range suffix {
		if ch < '0' || ch > '9' {
			allDigits = false
		}
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'z') {
			allLowerAlphaNum = false
		}
	}
	if allDigits {
		return true
	}
	return allLowerAlphaNum && len(suffix) == 4
}

func browserResolveDestroyChromeProfileDir(flags map[string]string, item browserInstanceRecord) (string, error) {
	if profileDir := strings.TrimSpace(item.ProfileDir); profileDir != "" {
		return profileDir, nil
	}
	profilePaths, err := browserResolveDestroyChromeProfilePaths(flags, item)
	if err != nil {
		return "", err
	}
	return profilePaths.Local, nil
}

func browserResolveDestroyChromeProfilePaths(flags map[string]string, item browserInstanceRecord) (browserChromeProfilePaths, error) {
	agentID := browserNormalizeIdentityPart(item.AgentID)
	if item.Port <= 0 {
		return browserChromeProfilePaths{}, fmt.Errorf("missing port for destroy profile cleanup: agentId=%s chatId=%s", agentID, browserNormalizeIdentityPart(item.ChatID))
	}
	return browserResolveChromeProfileDir(flags, agentID, item.Port)
}

func browserAppendUniqueInts(dst []int, values ...int) []int {
	seen := make(map[int]struct{}, len(dst)+len(values))
	for _, value := range dst {
		if value > 0 {
			seen[value] = struct{}{}
		}
	}
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		dst = append(dst, value)
	}
	return dst
}

func browserAppendUniqueStrings(dst []string, values ...string) []string {
	seen := make(map[string]struct{}, len(dst)+len(values))
	for _, value := range dst {
		value = strings.TrimSpace(value)
		if value != "" {
			seen[value] = struct{}{}
		}
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		dst = append(dst, value)
	}
	return dst
}

func browserDestroyMissingStateFallbackItems(agentID, chatID string) []browserInstanceRecord {
	agentID = browserNormalizeIdentityPart(agentID)
	chatID = browserNormalizeIdentityPart(chatID)
	if isWSL, err := browserWSLDetectFn(); err == nil && isWSL {
		return browserDestroyMissingStateWSLItems(agentID, chatID)
	}
	ports := browserDestroyCreateCandidatePorts(agentID, chatID)
	items := make([]browserInstanceRecord, 0, len(ports))
	for _, port := range ports {
		items = append(items, browserInstanceRecord{
			AgentID: agentID,
			ChatID:  chatID,
			Port:    port,
		})
	}
	return items
}

func browserDestroyMissingStateWSLItems(agentID, chatID string) []browserInstanceRecord {
	// WSL missing-state recovery uses browser_data only. We do not fall back to
	// hashed-port guessing here because WSL instances can be launched on dynamic
	// ports and browser_data is the source of truth for agentId + chatId lookup.
	record, ok := browserWSLInstanceLookupRecordFn(agentID, chatID)
	if !ok {
		return nil
	}
	return []browserInstanceRecord{{
		AgentID:    record.AgentID,
		ChatID:     record.ChatID,
		Port:       record.Port,
		PID:        record.PID,
		CDP:        record.WS,
		ProfileDir: record.UserDataDir,
	}}
}

func browserDestroyCreateCandidatePorts(agentID, chatID string) []int {
	port := browserHashedPort(agentID, chatID)
	if !browserManagedPortAllowed(port) {
		return nil
	}
	return []int{port}
}

func browserDestroyReleaseMissingStateFallbackItems(items []browserInstanceRecord) (bool, error) {
	cleaned := false
	for _, item := range items {
		if item.Port <= 0 {
			continue
		}
		if _, err := browserResolveLiveCDPEndpoint(item.Port); err != nil {
			continue
		}
		if err := browserTerminateManagedInstanceFn(item); err != nil {
			return cleaned, err
		}
		cleaned = true
	}
	return cleaned, nil
}

func browserDestroyCleanupStateFile(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	_ = os.Remove(path)
}

func browserFlagsWithoutIdentity(flags map[string]string) map[string]string {
	next := cloneFlags(flags)
	delete(next, "agentId")
	delete(next, "agent")
	delete(next, "chatId")
	delete(next, "chat")
	delete(next, "session")
	return next
}

func browserRestartInstances(flags map[string]string) error {
	statePath, err := browserResolveStatePath(flags)
	if err != nil {
		return err
	}
	items, err := browserLoadAndPersistLiveInstancesWithFlags(statePath, flags)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := browserTerminateManagedInstanceFn(browserInstanceAPIRecord(flags, item)); err != nil {
			return err
		}
		browserLogClosureEventFn("restart", map[string]any{
			"agentId": item.AgentID,
			"chatId":  item.ChatID,
			"pid":     item.PID,
			"port":    item.Port,
			"cdp":     item.CDP,
		})
	}
	return browserSaveInstances(statePath, []browserInstanceStateRecord{})
}

func browserGracefulStopPluginInstances(flags map[string]string) {
	items, err := instanceListFn(flags)
	if err != nil {
		browserLogAsyncLifecycleEvent("browser_stop_instances", "list_error", nil, 0, err)
		return
	}
	for _, item := range items {
		nextFlags := browserFlagsWithoutIdentity(flags)
		nextFlags["agentId"] = item.AgentID
		nextFlags["chatId"] = item.ChatID
		browserLogInstanceShutdownRequest("plugin_stop_cleanup", nextFlags, map[string]any{
			"pid":  item.PID,
			"port": item.Port,
			"cdp":  item.CDP,
		})
		if err := browserInvokeInstanceShutdown(nextFlags); err != nil {
			browserLogAsyncLifecycleEvent("browser_stop_instances", "shutdown_error", []string{item.AgentID, item.ChatID, strconv.Itoa(item.Port)}, item.PID, err)
			continue
		}
		browserLogClosureEventFn("stop_shutdown", map[string]any{
			"agentId": item.AgentID,
			"chatId":  item.ChatID,
			"pid":     item.PID,
			"port":    item.Port,
			"cdp":     item.CDP,
		})
	}
}

func browserInvokeInstanceShutdown(flags map[string]string) error {
	err := instanceDestroyFn(flags)
	if browserIsInstanceNotFoundError(err) {
		return nil
	}
	return err
}

func browserLogInstanceShutdownRequest(reason string, flags map[string]string, extra map[string]any) {
	next := browserNormalizeIdentityFlags(flags)
	payload := map[string]any{
		"event":     "browser_instance_shutdown_request",
		"reason":    strings.TrimSpace(reason),
		"timestamp": browserNowFn().Format(time.RFC3339Nano),
	}
	if session := strings.TrimSpace(next["session"]); session != "" {
		payload["session"] = session
	}
	if agentID := firstNonEmptyBrowser(strings.TrimSpace(next["agentId"]), strings.TrimSpace(next["agent"])); agentID != "" {
		payload["agentId"] = agentID
	}
	if chatID := firstNonEmptyBrowser(strings.TrimSpace(next["chatId"]), strings.TrimSpace(next["chat"])); chatID != "" {
		payload["chatId"] = chatID
	}
	for key, value := range extra {
		payload[key] = value
	}
	browserAppendLogJSON(payload)
}

func browserReleaseManagedInstance(item browserInstanceStateRecord) error {
	record := item.apiRecord()
	if strings.TrimSpace(record.CDP) == "" && record.Port > 0 {
		if cdp, err := browserResolveLiveCDPEndpoint(record.Port); err == nil {
			record.CDP = cdp
		}
	}
	return browserTerminateManagedInstanceFn(record)
}

func browserListInstances(flags map[string]string) ([]browserInstanceRecord, error) {
	statePath, err := browserResolveStatePath(flags)
	if err != nil {
		return nil, err
	}
	items, err := browserLoadAndPersistLiveInstances(statePath)
	if err != nil {
		return nil, err
	}
	return browserAPIRecordsWithProfile(flags, items), nil
}

func browserGetInstance(flags map[string]string) (browserInstanceRecord, error) {
	agentID, chatID, err := browserRequiredIdentity(flags)
	if err != nil {
		return browserInstanceRecord{}, err
	}
	statePath, err := browserResolveStatePath(flags)
	if err != nil {
		return browserInstanceRecord{}, err
	}
	items, err := browserLoadAndPersistLiveInstances(statePath)
	if err != nil {
		return browserInstanceRecord{}, err
	}
	idx := browserFindInstanceIndex(items, agentID, chatID)
	if idx < 0 {
		return browserInstanceRecord{}, fmt.Errorf("instance not found: agentId=%s chatId=%s", agentID, chatID)
	}
	items[idx].LastActiveAt = browserFormatActivityTime(browserNowFn())
	if err := browserSaveInstances(statePath, items); err != nil {
		return browserInstanceRecord{}, err
	}
	return browserInstanceAPIRecord(flags, items[idx]), nil
}

func browserRequiredIdentity(flags map[string]string) (string, string, error) {
	flags = browserNormalizeIdentityFlags(flags)
	agentID := connectsvc.FirstValue(flags, "agent", "agentId")
	chatID := connectsvc.FirstValue(flags, "chat", "chatId")
	if strings.TrimSpace(chatID) == "" {
		return "", "", errors.New("chatId is required")
	}
	return agentID, chatID, nil
}

func browserResolveStatePath(flags map[string]string) (string, error) {
	if raw := strings.TrimSpace(connectsvc.FirstValue(flags, "state")); raw != "" {
		return filepath.Abs(raw)
	}
	root, _, err := browserRuntimeRoot(flags)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, browserStateFileName), nil
}

func browserResolveSelfPath(flags map[string]string) (string, error) {
	_, browserPath, err := browserRuntimeRoot(flags)
	if err != nil {
		return "", err
	}
	return browserPath, nil
}

func browserResolveChromePath(flags map[string]string) (string, error) {
	if path, ok, err := browserResolveChromePathFromPluginMeta(flags); err != nil {
		return "", err
	} else if ok {
		return path, nil
	}
	if isWSL, err := browserWSLDetectFn(); err == nil && isWSL {
		if path, ensureErr := browserEnsureExecutablePath(browserWSLDefaultChromePath); ensureErr == nil {
			return path, nil
		}
	}
	if raw := strings.TrimSpace(connectsvc.FirstValue(flags, "chrome", "obscura")); raw != "" {
		return browserEnsureExecutablePath(raw)
	}

	for _, candidate := range browserChromePathCandidates() {
		if path, err := browserEnsureExecutablePath(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("chrome executable not found; use --chrome to specify a custom path")
}

func browserResolveChromePathFromPluginMeta(flags map[string]string) (string, bool, error) {
	raw, ok := browserLookupChromePathFromPluginMeta(flags)
	if !ok {
		return "", false, nil
	}
	if !filepath.IsAbs(raw) {
		return "", true, fmt.Errorf("browser meta chrome must be an absolute path: %q", raw)
	}
	path, err := browserEnsureExecutablePath(raw)
	if err != nil {
		return "", true, err
	}
	return path, true, nil
}

func browserChromePathCandidates() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
	case "linux":
		return []string{
			"google-chrome",
			"google-chrome-stable",
			"chromium-browser",
			"chromium",
			"chrome",
		}
	case "windows":
		programFiles := strings.TrimSpace(os.Getenv("PROGRAMFILES"))
		programFilesX86 := strings.TrimSpace(os.Getenv("PROGRAMFILES(X86)"))
		localAppData := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
		return []string{
			filepath.Join(programFiles, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(programFilesX86, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(localAppData, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(programFiles, "Chromium", "Application", "chrome.exe"),
			filepath.Join(programFilesX86, "Chromium", "Application", "chrome.exe"),
		}
	default:
		return []string{"google-chrome", "chromium", "chrome"}
	}
}

func browserResolveChromeHeadlessMode(flags map[string]string) (string, error) {
	if forced := strings.ToLower(strings.TrimSpace(connectsvc.FirstValue(flags, "headless-force"))); forced != "" {
		return browserValidateChromeHeadlessMode(forced)
	}
	if mode, ok, err := browserResolveChromeHeadlessModeFromPluginMeta(flags); err != nil {
		return "", err
	} else if ok {
		return mode, nil
	}
	return browserResolveChromeHeadlessModeFromFlags(flags)
}

func browserResolveChromeHeadlessModeFromFlags(flags map[string]string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(connectsvc.FirstValue(flags, "headless")))
	if mode == "" {
		return browserDefaultHeadlessMode, nil
	}
	return browserValidateChromeHeadlessMode(mode)
}

func browserValidateChromeHeadlessMode(mode string) (string, error) {
	switch mode {
	case "new", "none":
		return mode, nil
	default:
		return "", fmt.Errorf("headless must be one of: new, none")
	}
}

func browserResolveChromeHeadlessModeFromPluginMeta(flags map[string]string) (string, bool, error) {
	response, ok, err := browserLookupPluginMeta(flags, browserKey, true)
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}
	mode, ok := browserParsePluginMetaHeadlessMode(response.Meta["headless"])
	if !ok {
		return "", false, nil
	}
	return mode, true, nil
}

func browserParsePluginMetaHeadlessMode(value any) (string, bool) {
	switch v := value.(type) {
	case bool:
		if v {
			return browserDefaultHeadlessMode, true
		}
		return "none", true
	case string:
		normalized := strings.ToLower(strings.TrimSpace(v))
		switch normalized {
		case "":
			return "", false
		case "true":
			return browserDefaultHeadlessMode, true
		case "false":
			return "none", true
		default:
			return browserDefaultHeadlessMode, true
		}
	default:
		return browserDefaultHeadlessMode, true
	}
}

func browserResolveChromeProfileDir(flags map[string]string, agentID string, port int) (browserChromeProfilePaths, error) {
	if isWSL, err := browserWSLDetectFn(); err == nil && isWSL {
		return browserResolveWSLChromeProfileDir(port)
	}
	workspace, err := browserResolveAgentWorkspace(flags, agentID)
	if err != nil {
		return browserChromeProfilePaths{}, err
	}
	profileDir := filepath.Join(workspace, "chrome_"+strconv.Itoa(port))
	if absProfileDir, err := filepath.Abs(profileDir); err == nil {
		profileDir = absProfileDir
	}
	return browserChromeProfilePaths{Local: profileDir, Launch: profileDir}, nil
}

func browserLookupLiveInstance(flags map[string]string) (browserInstanceRecord, bool, error) {
	agentID, chatID, err := browserRequiredIdentity(flags)
	if err != nil {
		return browserInstanceRecord{}, false, err
	}
	statePath, err := browserResolveStatePath(flags)
	if err != nil {
		return browserInstanceRecord{}, false, err
	}
	items, err := browserLoadAndPersistLiveInstances(statePath)
	if err != nil {
		return browserInstanceRecord{}, false, err
	}
	idx := browserFindInstanceIndex(items, agentID, chatID)
	if idx < 0 {
		return browserInstanceRecord{}, false, nil
	}
	return items[idx].apiRecord(), true, nil
}

func browserResolveAgentRoot(flags map[string]string) (string, error) {
	if runtimePath, ok, err := browserResolveRuntimeConfigPath(flags); err != nil {
		return "", err
	} else if ok {
		cfg, err := browserReadRuntimeConfig(runtimePath)
		if err != nil {
			return "", err
		}
		agentRoot := browserResolveRuntimePathValue(runtimePath, cfg["agent-dir"])
		if agentRoot == "" {
			if appDir := browserResolveRuntimePathValue(runtimePath, cfg["app-dir"]); appDir != "" {
				agentRoot = filepath.Join(appDir, "agent")
			}
		}
		if agentRoot != "" {
			return agentRoot, nil
		}
	}
	root, _, err := browserRuntimeRoot(flags)
	if err != nil {
		return "", err
	}
	return root, nil
}

func browserResolveAgentWorkspace(flags map[string]string, agentID string) (string, error) {
	agentID = browserNormalizeIdentityPart(agentID)
	if agentID == "" {
		return "", errors.New("agentId is required")
	}
	agentRoot, err := browserResolveAgentRoot(flags)
	if err != nil {
		return "", err
	}
	return filepath.Join(agentRoot, agentID), nil
}

func browserResolveRuntimeConfigPath(flags map[string]string) (string, bool, error) {
	_ = flags
	if runtimePath, ok, err := browserResolveRecordedRuntimeConfigPath(); err != nil {
		return "", false, err
	} else if ok {
		return runtimePath, true, nil
	}
	execPath, err := browserExecutablePathFn()
	if err != nil {
		return "", false, err
	}
	if resolved, err := filepath.EvalSymlinks(execPath); err == nil {
		execPath = resolved
	}
	runtimePath, ok := connectsvc.ResolveRuntimeConfigPathNearBinary(execPath)
	return runtimePath, ok, nil
}

func browserResolveRuntimePathValue(runtimePath, raw string) string {
	return connectsvc.ResolveRuntimePathValue(runtimePath, raw)
}

func browserPrepareChromeUserDataDir(flags map[string]string, profileDir string) error {
	return browserPrepareChromeUserDataDirWithContext(context.Background(), flags, profileDir)
}

func browserPrepareChromeUserDataDirWithContext(ctx context.Context, flags map[string]string, profileDir string) error {
	if err := browserInstanceInitContextError(ctx.Done()); err != nil {
		return err
	}
	profileDir = strings.TrimSpace(profileDir)
	if profileDir == "" {
		return errors.New("chrome profile dir is required")
	}
	browserCreateTrace("instance.user_data.prepare.begin", map[string]any{
		"profileDir": profileDir,
	})
	if browserResolveChromeUserDataMode(flags) == browserUserDataModeDirect {
		if err := os.MkdirAll(filepath.Dir(profileDir), 0o755); err != nil {
			browserCreateTrace("instance.user_data.prepare.error", map[string]any{
				"profileDir": profileDir,
				"error":      err.Error(),
				"stage":      "mkdir_parent",
			})
			return err
		}
		browserCreateTrace("instance.user_data.target_prepare.begin", map[string]any{
			"profileDir": profileDir,
		})
		if err := os.MkdirAll(profileDir, 0o755); err != nil {
			browserCreateTrace("instance.user_data.prepare.error", map[string]any{
				"profileDir": profileDir,
				"error":      err.Error(),
				"stage":      "mkdir_profile",
			})
			return err
		}
		browserCreateTrace("instance.user_data.target_prepare.ok", map[string]any{
			"profileDir": profileDir,
		})
		browserCreateTrace("instance.user_data.copy.skip", map[string]any{
			"profileDir": profileDir,
			"sourceKind": browserUserDataModeDirect,
			"reason":     "reuse_existing_profile_dir",
		})
		if err := browserMaybeCleanupPreparedChromeUserData(profileDir); err != nil {
			return err
		}
		browserCreateTrace("instance.user_data.prepare.ok", map[string]any{
			"profileDir": profileDir,
			"sourceKind": browserUserDataModeDirect,
		})
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(profileDir), 0o755); err != nil {
		browserCreateTrace("instance.user_data.prepare.error", map[string]any{
			"profileDir": profileDir,
			"error":      err.Error(),
			"stage":      "mkdir_parent",
		})
		return err
	}
	browserCreateTrace("instance.user_data.target_prepare.begin", map[string]any{
		"profileDir": profileDir,
	})
	info, err := os.Stat(profileDir)
	switch {
	case err == nil && !info.IsDir():
		browserCreateTrace("instance.user_data.prepare.error", map[string]any{
			"profileDir": profileDir,
			"error":      "chrome profile path exists and is not a directory",
			"stage":      "profile_not_dir",
		})
		return fmt.Errorf("chrome profile path exists and is not a directory: %s", profileDir)
	case err != nil && !errors.Is(err, os.ErrNotExist):
		browserCreateTrace("instance.user_data.prepare.error", map[string]any{
			"profileDir": profileDir,
			"error":      err.Error(),
			"stage":      "stat_profile",
		})
		return err
	}
	browserCreateTrace("instance.user_data.target_prepare.ok", map[string]any{
		"profileDir": profileDir,
	})
	if err == nil {
		browserCreateTrace("instance.user_data.copy.skip", map[string]any{
			"profileDir": profileDir,
			"sourceKind": "existing_profile",
			"reason":     "reuse_existing_profile_dir",
		})
		if err := browserMaybeCleanupPreparedChromeUserData(profileDir); err != nil {
			return err
		}
		browserCreateTrace("instance.user_data.prepare.ok", map[string]any{
			"profileDir": profileDir,
			"sourceKind": "existing_profile",
		})
		return nil
	}
	sourceDir, sourceKind, err := browserResolveCreateChromeUserDataSourceDir(flags)
	if err != nil {
		browserCreateTrace("instance.user_data.prepare.error", map[string]any{
			"profileDir": profileDir,
			"error":      err.Error(),
		})
		return err
	}
	if strings.TrimSpace(sourceDir) == "" {
		browserCreateTrace("instance.user_data.copy.skip", map[string]any{
			"profileDir": profileDir,
			"sourceKind": sourceKind,
			"reason":     "empty_source_dir",
		})
		if err := browserMkdirAllFn(profileDir, 0o755); err != nil {
			browserCreateTrace("instance.user_data.prepare.error", map[string]any{
				"profileDir": profileDir,
				"sourceKind": sourceKind,
				"error":      err.Error(),
				"stage":      "mkdir_empty_profile",
			})
			return err
		}
		if err := browserMaybeCleanupPreparedChromeUserData(profileDir); err != nil {
			return err
		}
		browserCreateTrace("instance.user_data.prepare.ok", map[string]any{
			"profileDir": profileDir,
			"sourceKind": sourceKind,
		})
		return nil
	}
	browserCreateTrace("instance.user_data.source.ok", map[string]any{
		"profileDir": profileDir,
		"sourceDir":  sourceDir,
		"sourceKind": sourceKind,
	})
	browserCreateTrace("instance.user_data.copy.begin", map[string]any{
		"profileDir": profileDir,
		"sourceDir":  sourceDir,
		"sourceKind": sourceKind,
	})
	copyStats, err := browserCopyDirectoryWithProgressContext(ctx,
		sourceDir,
		profileDir,
		browserInstanceUserDataProgressTracer("instance.user_data.copy.progress", map[string]any{
			"profileDir": profileDir,
			"sourceDir":  sourceDir,
			"sourceKind": sourceKind,
		}),
		browserInstanceUserDataSkipTracer("instance.user_data.copy.skip", map[string]any{
			"profileDir": profileDir,
			"sourceDir":  sourceDir,
			"sourceKind": sourceKind,
		}),
	)
	if err != nil {
		browserCreateTrace("instance.user_data.prepare.error", map[string]any{
			"profileDir": profileDir,
			"sourceDir":  sourceDir,
			"sourceKind": sourceKind,
			"error":      err.Error(),
			"stage":      "copy",
		})
		return err
	}
	browserCreateTrace("instance.user_data.copy.ok", map[string]any{
		"profileDir": profileDir,
		"sourceDir":  sourceDir,
		"sourceKind": sourceKind,
		"files":      copyStats.Files,
		"dirs":       copyStats.Dirs,
		"bytes":      copyStats.Bytes,
	})
	browserCreateTrace("instance.user_data.cleanup.begin", map[string]any{
		"profileDir": profileDir,
	})
	if err := browserCleanupClonedChromeUserData(profileDir); err != nil {
		browserCreateTrace("instance.user_data.prepare.error", map[string]any{
			"profileDir": profileDir,
			"error":      err.Error(),
			"stage":      "cleanup_locks",
		})
		return err
	}
	browserCreateTrace("instance.user_data.cleanup.ok", map[string]any{
		"profileDir": profileDir,
	})
	browserCreateTrace("instance.user_data.prepare.ok", map[string]any{
		"profileDir": profileDir,
		"sourceDir":  sourceDir,
		"sourceKind": sourceKind,
	})
	return nil
}

func browserInstanceInitRemainingTimeout(ctx context.Context) time.Duration {
	deadline, ok := ctx.Deadline()
	if !ok {
		return browserChromeStartupTimeout()
	}
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return time.Nanosecond
	}
	return remaining
}

func browserMaybeCleanupPreparedChromeUserData(profileDir string) error {
	browserCreateTrace("instance.user_data.cleanup.begin", map[string]any{
		"profileDir": profileDir,
	})
	if isWSL, err := browserWSLDetectFn(); err == nil && isWSL {
		if err := browserWSLCleanupChromeUserDataFn(profileDir); err != nil {
			browserCreateTrace("instance.user_data.prepare.error", map[string]any{
				"profileDir": profileDir,
				"error":      err.Error(),
				"stage":      "cleanup_locks",
			})
			return err
		}
		browserCreateTrace("instance.user_data.cleanup.ok", map[string]any{
			"profileDir": profileDir,
		})
		return nil
	}
	if err := browserCleanupClonedChromeUserData(profileDir); err != nil {
		browserCreateTrace("instance.user_data.prepare.error", map[string]any{
			"profileDir": profileDir,
			"error":      err.Error(),
			"stage":      "cleanup_locks",
		})
		return err
	}
	browserCreateTrace("instance.user_data.cleanup.ok", map[string]any{
		"profileDir": profileDir,
	})
	return nil
}

func browserResolveCreateChromeUserDataSourceDir(flags map[string]string) (string, string, error) {
	if isWSL, err := browserWSLDetectFn(); err == nil && isWSL {
		return "", "wsl_empty", nil
	}
	sourceDir, err := browserResolveSystemChromeUserDataDirFn(flags)
	if err != nil {
		return "", "", err
	}
	return sourceDir, "system", nil
}

func browserResolveChromeUserDataMode(flags map[string]string) string {
	mode := strings.ToLower(strings.TrimSpace(flags["user-data-mode"]))
	switch mode {
	case "", browserUserDataModeClone:
		return browserUserDataModeClone
	case browserUserDataModeDirect:
		return browserUserDataModeDirect
	default:
		return browserUserDataModeClone
	}
}

func browserResolveSystemChromeUserDataDir(flags map[string]string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	for _, candidate := range browserSystemChromeUserDataCandidates(home) {
		info, statErr := os.Stat(candidate)
		if statErr == nil && info.IsDir() {
			if abs, absErr := filepath.Abs(candidate); absErr == nil {
				return abs, nil
			}
			return candidate, nil
		}
	}
	return "", fmt.Errorf("chrome user data root not found on %s", runtime.GOOS)
}

func browserSystemChromeUserDataCandidates(home string) []string {
	home = strings.TrimSpace(home)
	switch runtime.GOOS {
	case "darwin":
		return []string{
			filepath.Join(home, "Library", "Application Support", "Google", "Chrome"),
			filepath.Join(home, "Library", "Application Support", "Google", "Chrome for Testing"),
			filepath.Join(home, "Library", "Application Support", "Chromium"),
		}
	case "linux":
		return []string{
			filepath.Join(home, ".config", "google-chrome"),
			filepath.Join(home, ".config", "google-chrome-beta"),
			filepath.Join(home, ".config", "chromium"),
		}
	case "windows":
		localAppData := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
		return []string{
			filepath.Join(localAppData, "Google", "Chrome", "User Data"),
			filepath.Join(localAppData, "Chromium", "User Data"),
		}
	default:
		return []string{
			filepath.Join(home, ".config", "google-chrome"),
			filepath.Join(home, ".config", "chromium"),
		}
	}
}

func browserCopyDirectory(sourceDir, targetDir string) error {
	_, err := browserCopyDirectoryWithProgress(sourceDir, targetDir, nil, nil)
	return err
}

func browserCopyDirectoryWithProgress(sourceDir, targetDir string, progressFn func(browserArchiveProgress), skipFn func(string, error)) (browserArchiveProgress, error) {
	return browserCopyDirectoryWithProgressContext(context.Background(), sourceDir, targetDir, progressFn, skipFn)
}

func browserCopyDirectoryWithProgressContext(ctx context.Context, sourceDir, targetDir string, progressFn func(browserArchiveProgress), skipFn func(string, error)) (browserArchiveProgress, error) {
	return browserCopyDirectoryWithProgressUsingSkipRuleContext(ctx, sourceDir, targetDir, progressFn, skipFn, browserShouldSkipClonedChromeUserDataPath)
}

func browserCopyWSLChromeUserDataWithProgress(sourceDir, targetDir string, progressFn func(browserArchiveProgress), skipFn func(string, error)) (browserArchiveProgress, error) {
	return browserCopyDirectoryWithProgressUsingSkipRuleContext(context.Background(), sourceDir, targetDir, progressFn, skipFn, browserShouldSkipWSLChromeUserDataPath)
}

func browserCopyDirectoryWithProgressUsingSkipRule(sourceDir, targetDir string, progressFn func(browserArchiveProgress), skipFn func(string, error), shouldSkip func(string) (bool, string)) (browserArchiveProgress, error) {
	return browserCopyDirectoryWithProgressUsingSkipRuleContext(context.Background(), sourceDir, targetDir, progressFn, skipFn, shouldSkip)
}

func browserCopyDirectoryWithProgressUsingSkipRuleContext(ctx context.Context, sourceDir, targetDir string, progressFn func(browserArchiveProgress), skipFn func(string, error), shouldSkip func(string) (bool, string)) (browserArchiveProgress, error) {
	progress := browserArchiveProgress{}
	sourceDir = filepath.Clean(strings.TrimSpace(sourceDir))
	targetDir = filepath.Clean(strings.TrimSpace(targetDir))
	if sourceDir == "" || targetDir == "" {
		return progress, errors.New("source and target chrome dirs are required")
	}
	err := filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := browserInstanceInitContextError(ctx.Done()); err != nil {
			return err
		}
		if walkErr != nil {
			if browserShouldSkipArchivePathError(walkErr) {
				if skipFn != nil {
					skipFn(path, walkErr)
				}
				return nil
			}
			return walkErr
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		destination := targetDir
		if rel != "." {
			destination = filepath.Join(targetDir, rel)
		}
		if skip, reason := shouldSkip(rel); skip {
			if skipFn != nil {
				skipFn(path, errors.New(reason))
			}
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := browserArchiveDirEntryInfoFn(entry)
		if err != nil {
			if browserShouldSkipArchivePathError(err) {
				if skipFn != nil {
					skipFn(path, err)
				}
				return nil
			}
			return err
		}
		switch mode := info.Mode(); {
		case mode.IsDir():
			if err := os.MkdirAll(destination, mode.Perm()); err != nil {
				return err
			}
			progress.Dirs++
			progress.Path = path
			if progressFn != nil {
				progressFn(progress)
			}
			return nil
		case mode.IsRegular():
			if err := browserCopyFileWithContext(ctx, path, destination, mode.Perm()); err != nil {
				if browserShouldSkipArchivePathError(err) {
					if skipFn != nil {
						skipFn(path, err)
					}
					return nil
				}
				return err
			}
			progress.Files++
			progress.Bytes += info.Size()
			progress.Path = path
			if progressFn != nil {
				progressFn(progress)
			}
			return nil
		case mode&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
				return err
			}
			if err := os.Symlink(target, destination); err != nil {
				return err
			}
			progress.Path = path
			if progressFn != nil {
				progressFn(progress)
			}
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return progress, err
	}
	return progress, nil
}

func browserMeasureArchiveProgress(root string) (browserArchiveProgress, error) {
	progress := browserArchiveProgress{}
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" {
		return progress, errors.New("archive root is required")
	}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := browserArchiveDirEntryInfoFn(entry)
		if err != nil {
			return err
		}
		if info.IsDir() {
			progress.Dirs++
		} else {
			progress.Files++
			progress.Bytes += info.Size()
		}
		progress.Path = path
		return nil
	})
	return progress, err
}

func browserShouldSkipClonedChromeUserDataPath(rel string) (bool, string) {
	rel = filepath.Clean(strings.TrimSpace(rel))
	if rel == "" || rel == "." {
		return false, ""
	}
	segments := strings.Split(filepath.ToSlash(rel), "/")
	if len(segments) == 1 {
		name := strings.TrimSpace(segments[0])
		switch name {
		case "RunningChromeVersion":
			return true, "skip chrome runtime version marker"
		default:
			if browserShouldRemoveClonedChromeLockEntry(name) {
				return true, "skip chrome runtime lock path"
			}
		}
	}
	if browserPathSegmentsContain(segments, "CacheStorage") {
		return true, "skip chrome CacheStorage path"
	}
	if browserPathSegmentsHavePrefix(segments, []string{"OptGuideOnDeviceModel"}) {
		return true, "skip chrome on-device model path"
	}
	volatilePrefixes := [][]string{
		{"Default", "Cache"},
		{"Default", "Code Cache"},
		{"Default", "GPUCache"},
		{"Default", "GrShaderCache"},
		{"Default", "GraphiteDawnCache"},
		{"Default", "DawnGraphiteCache"},
		{"Default", "DawnWebGPUCache"},
		{"Default", "ShaderCache"},
		{"Default", "Media Cache"},
		{"Default", "Network"},
		{"Default", "Safe Browsing Network"},
		{"Default", "Shared Dictionary"},
		{"Default", "GCM Store"},
		{"Default", "blob_storage"},
		{"Default", "Service Worker", "ScriptCache"},
	}
	for _, prefix := range volatilePrefixes {
		if browserPathSegmentsHavePrefix(segments, prefix) {
			return true, "skip volatile chrome cache path"
		}
	}
	return false, ""
}

func browserShouldSkipWSLChromeUserDataPath(rel string) (bool, string) {
	segments := strings.Split(filepath.ToSlash(filepath.Clean(strings.TrimSpace(rel))), "/")
	if browserWSLChromeLoginStatePath(segments) {
		return false, ""
	}
	return browserShouldSkipClonedChromeUserDataPath(rel)
}

func browserWSLChromeLoginStatePath(segments []string) bool {
	if len(segments) < 2 {
		return false
	}
	profile := strings.TrimSpace(segments[0])
	if profile != "Default" && profile != "Guest Profile" && profile != "System Profile" && !strings.HasPrefix(profile, "Profile ") {
		return false
	}
	switch strings.TrimSpace(segments[1]) {
	case "Network", "Cookies", "Login Data", "Login Data For Account", "Web Data", "Preferences", "Secure Preferences", "Local Storage", "Session Storage", "IndexedDB", "WebStorage", "Extension State", "Extension Cookies", "Extension Rules", "Service Worker", "GCM Store", "Shared Dictionary", "blob_storage":
		return true
	default:
		return false
	}
}

func browserPathSegmentsHavePrefix(pathSegments, prefix []string) bool {
	if len(pathSegments) < len(prefix) {
		return false
	}
	for idx := range prefix {
		if pathSegments[idx] != prefix[idx] {
			return false
		}
	}
	return true
}

func browserPathSegmentsContain(pathSegments []string, needle string) bool {
	needle = strings.TrimSpace(needle)
	if needle == "" {
		return false
	}
	for _, segment := range pathSegments {
		if segment == needle {
			return true
		}
	}
	return false
}

func browserExtractRemoteDebuggingPort(commandLine string) int {
	commandLine = strings.TrimSpace(commandLine)
	marker := "--remote-debugging-port="
	idx := strings.Index(commandLine, marker)
	if idx < 0 {
		return 0
	}
	start := idx + len(marker)
	end := start
	for end < len(commandLine) && commandLine[end] >= '0' && commandLine[end] <= '9' {
		end++
	}
	if end == start {
		return 0
	}
	port, err := strconv.Atoi(commandLine[start:end])
	if err != nil {
		return 0
	}
	return port
}

func browserCopyFile(sourcePath, targetPath string, perm fs.FileMode) error {
	return browserCopyFileWithContext(context.Background(), sourcePath, targetPath, perm)
}

func browserCopyFileWithContext(ctx context.Context, sourcePath, targetPath string, perm fs.FileMode) error {
	if err := browserInstanceInitContextError(ctx.Done()); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	src, err := browserArchiveOpenFileFn(sourcePath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	buffer := make([]byte, 128*1024)
	for {
		if err := browserInstanceInitContextError(ctx.Done()); err != nil {
			_ = dst.Close()
			return err
		}
		readCount, readErr := src.Read(buffer)
		if readCount > 0 {
			if _, err := dst.Write(buffer[:readCount]); err != nil {
				_ = dst.Close()
				return err
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			_ = dst.Close()
			return readErr
		}
	}
	if err := dst.Close(); err != nil {
		return err
	}
	if err := browserArchiveChmodFn(targetPath, perm); err != nil && !browserShouldIgnoreCopyFileModeError(targetPath, err) {
		return err
	}
	return nil
}

func browserCleanupClonedChromeUserData(profileDir string) error {
	return browserCleanupClonedChromeUserDataForOS(profileDir, runtime.GOOS)
}

func browserCleanupClonedChromeUserDataForOS(profileDir, goos string) error {
	profileDir = strings.TrimSpace(profileDir)
	if profileDir == "" {
		return nil
	}
	for _, name := range browserChromeUserDataLockNames(goos) {
		if err := os.RemoveAll(filepath.Join(profileDir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return filepath.WalkDir(profileDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, os.ErrNotExist) {
				return nil
			}
			return walkErr
		}
		rel, err := filepath.Rel(profileDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if skip, _ := browserShouldSkipClonedChromeUserDataPath(rel); skip {
			if err := os.RemoveAll(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if browserShouldRemoveClonedChromeLockEntry(entry.Name()) {
			if err := os.RemoveAll(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			if entry.IsDir() {
				return filepath.SkipDir
			}
		}
		return nil
	})
}

func browserShouldRemoveClonedChromeLockEntry(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	switch strings.ToUpper(name) {
	case "LOCK", "LOCKFILE", "SINGLETONLOCK", "SINGLETONCOOKIE", "SINGLETONSOCKET", "DEVTOOLSACTIVEPORT":
		return true
	}
	lowerName := strings.ToLower(name)
	return strings.HasSuffix(lowerName, ".lock") || strings.HasSuffix(lowerName, "-journal")
}

func browserRemoveTopLevelChromeUserDataLockfile(profileDir string) error {
	profileDir = strings.TrimSpace(profileDir)
	if profileDir == "" {
		return nil
	}
	if err := os.Remove(filepath.Join(profileDir, "lockfile")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func browserChromeUserDataLockNames(goos string) []string {
	items := []string{
		"SingletonLock",
		"SingletonCookie",
		"SingletonSocket",
		"DevToolsActivePort",
	}
	switch strings.TrimSpace(goos) {
	case "windows":
		items = append(items, "lockfile")
	case "linux":
		items = append(items, "lockfile")
	case "darwin":
		items = append(items, "lockfile")
	}
	return items
}

func browserEnsureExecutablePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("path is empty")
	}
	if !filepath.IsAbs(path) && !strings.Contains(path, "/") && !strings.Contains(path, "\\") {
		resolved, err := browserLookPathFn(path)
		if err != nil {
			return "", fmt.Errorf("executable not found in PATH: %s", path)
		}
		path = resolved
	}
	absPath := path
	if !filepath.IsAbs(path) {
		var err error
		absPath, err = filepath.Abs(path)
		if err != nil {
			return "", err
		}
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("path is directory: %s", absPath)
	}
	return absPath, nil
}

func browserLoadAndPersistLiveInstances(statePath string) ([]browserInstanceStateRecord, error) {
	return browserLoadAndPersistLiveInstancesWithFlags(statePath, nil)
}

func browserLoadAndPersistLiveInstancesWithFlags(statePath string, flags map[string]string) ([]browserInstanceStateRecord, error) {
	expiredMinutes, err := browserInstanceExpiredMinutes(flags)
	if err != nil {
		return nil, err
	}
	return browserLoadAndPersistLiveInstancesWithExpiredMinutes(statePath, expiredMinutes)
}

func browserLoadAndPersistLiveInstancesWithExpiredMinutes(statePath string, expiredMinutes int) ([]browserInstanceStateRecord, error) {
	items, err := browserLoadInstances(statePath)
	if err != nil {
		return nil, err
	}
	liveItems, changed, err := browserNormalizeInstances(items, expiredMinutes)
	if err != nil {
		return nil, err
	}
	if changed {
		if err := browserSaveInstances(statePath, liveItems); err != nil {
			return nil, err
		}
	}
	return liveItems, nil
}

func browserLoadInstances(statePath string) ([]browserInstanceStateRecord, error) {
	data, err := os.ReadFile(statePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []browserInstanceStateRecord{}, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return []browserInstanceStateRecord{}, nil
	}
	var items []browserInstanceStateRecord
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func browserNormalizeInstances(items []browserInstanceStateRecord, expiredMinutes int) ([]browserInstanceStateRecord, bool, error) {
	normalized := make([]browserInstanceStateRecord, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	changed := false

	for _, item := range items {
		item.AgentID = browserNormalizeIdentityPart(item.AgentID)
		item.ChatID = browserNormalizeIdentityPart(item.ChatID)
		if item.AgentID == "" || item.ChatID == "" || !browserManagedPortAllowed(item.Port) {
			changed = true
			continue
		}
		isCDP, err := browserPortCDPCheckFn(item.Port)
		if err != nil {
			return nil, false, err
		}
		pidAlive := item.PID > 0 && browserProcessExistsFn(item.PID)
		if !isCDP {
			if pidAlive {
				if err := browserTerminateProcessFn(item.PID); err != nil {
					return nil, false, err
				}
			}
			browserLogClosureEventFn("normalize_unhealthy_cdp", map[string]any{
				"agentId": item.AgentID,
				"chatId":  item.ChatID,
				"pid":     item.PID,
				"port":    item.Port,
				"cdp":     item.CDP,
			})
			changed = true
			continue
		}
		lastActive := browserParseActivityTime(item.LastActiveAt)
		if browserNowFn().Sub(lastActive) > browserInstanceExpiredDuration(expiredMinutes) {
			if err := browserReleaseManagedInstance(item); err != nil {
				return nil, false, err
			}
			browserLogClosureEventFn("normalize_expired", map[string]any{
				"agentId":        item.AgentID,
				"chatId":         item.ChatID,
				"pid":            item.PID,
				"port":           item.Port,
				"cdp":            item.CDP,
				"lastActiveAt":   item.LastActiveAt,
				"expiredMinutes": expiredMinutes,
			})
			changed = true
			continue
		}
		expectedCDP, err := browserResolveLiveCDPEndpoint(item.Port)
		if err != nil {
			return nil, false, err
		}
		if strings.TrimSpace(item.CDP) != expectedCDP {
			item.CDP = expectedCDP
			changed = true
		}
		expectedLastActive := browserFormatActivityTime(lastActive)
		if item.LastActiveAt != expectedLastActive {
			item.LastActiveAt = expectedLastActive
			changed = true
		}
		key := browserInstanceKey(item.AgentID, item.ChatID)
		if _, ok := seen[key]; ok {
			changed = true
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, item)
	}

	before := append([]browserInstanceStateRecord(nil), normalized...)
	browserSortInstances(normalized)
	if !browserSameInstances(before, normalized) {
		changed = true
	}
	if len(normalized) != len(items) {
		changed = true
	}
	return normalized, changed, nil
}

func browserManagedPortAllowed(port int) bool {
	return port >= browserMinPort && port <= browserMaxPort
}

func browserAllocatePortWithBlocked(agentID, chatID string, items []browserInstanceRecord, blockedPorts map[int]struct{}) (int, error) {
	if len(blockedPorts) == 0 {
		return browserAllocatePortFn(agentID, chatID, items)
	}
	next := make([]browserInstanceRecord, 0, len(items)+len(blockedPorts))
	next = append(next, items...)
	for port := range blockedPorts {
		next = append(next, browserInstanceRecord{Port: port})
	}
	return browserAllocatePortFn(agentID, chatID, next)
}

func browserRandomIntn(max int) (int, error) {
	if max <= 0 {
		return 0, fmt.Errorf("random max must be positive: %d", max)
	}
	value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(value.Int64()), nil
}

func browserAllocateCreatePort(agentID, chatID string, items []browserInstanceRecord, blockedPorts map[int]struct{}) (int, error) {
	_ = blockedPorts
	port := browserHashedPort(agentID, chatID)
	if !browserManagedPortAllowed(port) {
		return 0, fmt.Errorf("invalid stable browser instance port for agentId=%s chatId=%s port=%d", agentID, chatID, port)
	}
	for _, item := range items {
		if item.Port == port {
			return 0, fmt.Errorf("stable browser instance port occupied: agentId=%s chatId=%s port=%d", agentID, chatID, port)
		}
	}
	if !browserPortAvailableFn(port) {
		return 0, fmt.Errorf("stable browser instance port unavailable: agentId=%s chatId=%s port=%d", agentID, chatID, port)
	}
	return port, nil
}

func browserAllocateInitPort(agentID, chatID string, items []browserInstanceRecord, blockedPorts map[int]struct{}) (int, error) {
	return browserAllocateCreatePort(agentID, chatID, items, blockedPorts)
}

func browserShouldRetryChromePortFailure(attempt int, stage string, port int, err error) bool {
	if err == nil || port <= 0 || attempt >= browserPortRetryAttempts {
		return false
	}
	switch strings.TrimSpace(stage) {
	case "port_check", "wait_port", "resolve_cdp":
		return true
	default:
		return false
	}
}

func browserSameInstances(a, b []browserInstanceStateRecord) bool {
	if len(a) != len(b) {
		return false
	}
	for idx := range a {
		if a[idx] != b[idx] {
			return false
		}
	}
	return true
}

func browserSaveInstances(statePath string, items []browserInstanceStateRecord) error {
	browserSortInstances(items)
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmpPath := statePath + "." + strconv.FormatInt(time.Now().UnixNano(), 10) + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, statePath)
}

func browserRemoveInstance(statePath, _ string, chatID string, pid int) error {
	chatID = browserNormalizeIdentityPart(chatID)
	items, err := browserLoadInstances(statePath)
	if err != nil {
		return err
	}
	filtered := browserFilterInstances(items, func(item browserInstanceStateRecord) bool {
		return !(browserNormalizeIdentityPart(item.ChatID) == chatID &&
			(pid == 0 || item.PID == pid))
	})
	return browserPersistInstancesOrRemoveStateFile(statePath, filtered)
}

func browserRemoveInstancesByPort(statePath string, port int) error {
	items, err := browserLoadInstances(statePath)
	if err != nil {
		return err
	}
	filtered := browserFilterInstances(items, func(item browserInstanceStateRecord) bool {
		return item.Port != port
	})
	return browserPersistInstancesOrRemoveStateFile(statePath, filtered)
}

func browserPersistInstancesOrRemoveStateFile(statePath string, items []browserInstanceStateRecord) error {
	if len(items) == 0 {
		if err := os.Remove(statePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	return browserSaveInstances(statePath, items)
}

func browserFindInstanceIndex(items []browserInstanceStateRecord, agentID, chatID string) int {
	agentID = browserNormalizeIdentityPart(agentID)
	chatID = browserNormalizeIdentityPart(chatID)
	if chatID == "" {
		return -1
	}
	if agentID != "" {
		for idx, item := range items {
			if browserNormalizeIdentityPart(item.AgentID) == agentID && browserNormalizeIdentityPart(item.ChatID) == chatID {
				return idx
			}
		}
	}
	for idx, item := range items {
		if browserNormalizeIdentityPart(item.ChatID) == chatID {
			return idx
		}
	}
	return -1
}

func browserSortInstances(items []browserInstanceStateRecord) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].AgentID != items[j].AgentID {
			return items[i].AgentID < items[j].AgentID
		}
		if items[i].ChatID != items[j].ChatID {
			return items[i].ChatID < items[j].ChatID
		}
		if items[i].Port != items[j].Port {
			return items[i].Port < items[j].Port
		}
		return items[i].PID < items[j].PID
	})
}

func browserFilterInstances(items []browserInstanceStateRecord, keepFn func(browserInstanceStateRecord) bool) []browserInstanceStateRecord {
	if keepFn == nil {
		return append([]browserInstanceStateRecord(nil), items...)
	}
	filtered := make([]browserInstanceStateRecord, 0, len(items))
	for _, item := range items {
		if keepFn(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func browserAllocatePort(agentID, chatID string, items []browserInstanceRecord) (int, error) {
	used := make(map[int]struct{}, len(items))
	for _, item := range items {
		if item.Port > 0 {
			used[item.Port] = struct{}{}
		}
	}
	for attempts := 0; attempts < 64; attempts++ {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return 0, err
		}
		port := listener.Addr().(*net.TCPAddr).Port
		_ = listener.Close()
		if !browserManagedPortAllowed(port) {
			continue
		}
		if _, ok := used[port]; ok {
			continue
		}
		if !browserPortAvailableFn(port) {
			continue
		}
		return port, nil
	}
	for port := browserMinPort; port <= browserMaxPort; port++ {
		if _, ok := used[port]; ok {
			continue
		}
		if browserPortAvailableFn(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("could not allocate browser instance port")
}

func browserHashedPort(agentID, chatID string) int {
	agentID = browserNormalizeIdentityPart(agentID)
	chatID = browserNormalizeIdentityPart(chatID)
	sum := sha256.Sum256([]byte(agentID + "\n" + chatID))
	value := binary.BigEndian.Uint32(sum[:4])
	span := browserMaxPort - browserMinPort + 1
	return browserMinPort + int(value%uint32(span))
}

func browserIsPortAvailable(port int) bool {
	listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		return false
	}
	_ = listener.Close()
	if isWSL, err := browserWSLDetectFn(); err == nil && isWSL {
		free, freeErr := browserWSLWindowsPortFreeFn(port)
		if freeErr == nil {
			return free
		}
	}
	return true
}

func browserChromeStartupTimeout() time.Duration {
	return 5 * time.Second
}

func browserChromeLaunchArgs(port int, profileDir, headlessMode string) []string {
	args := []string{
		"--remote-debugging-port=" + strconv.Itoa(port),
		"--remote-debugging-address=127.0.0.1",
		"--user-data-dir=" + profileDir,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-background-networking",
		"--disable-sync",
		"--disable-component-update",
		"about:blank",
	}
	if headlessMode == "" {
		headlessMode = browserDefaultHeadlessMode
	}
	if headlessMode != "none" {
		args = append([]string{"--headless=" + headlessMode}, args...)
	}
	return args
}

func browserStartChromeProcess(chromePath string, port int, profileDir, headlessMode, logPath string) (int, error) {
	args := browserChromeLaunchArgs(port, profileDir, headlessMode)
	cmd := exec.Command(chromePath, args...)
	cmd.Dir = filepath.Dir(chromePath)
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return 0, err
	}
	defer devNull.Close()
	logWriter, cleanup, err := browserStartChromeLogFilterFn(logPath)
	if err == nil {
		defer cleanup()
		cmd.Stdout = logWriter
		cmd.Stderr = logWriter
	} else {
		logFile, fileErr := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if fileErr != nil {
			return 0, fileErr
		}
		defer logFile.Close()
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}
	cmd.Stdin = devNull
	browserPrepareDetachedCommand(cmd)
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	return cmd.Process.Pid, nil
}

func browserInitChromeLaunchArgs(port int, profileDir string) []string {
	address := "0.0.0.0"
	if isWSL, err := browserWSLDetectFn(); err == nil && isWSL {
		address = "127.0.0.1"
	}
	return []string{
		"--remote-debugging-port=" + strconv.Itoa(port),
		"--remote-debugging-address=" + address,
		"--user-data-dir=" + profileDir,
		"--no-first-run",
	}
}

func browserWaitForPort(pid, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	hosts := browserCDPProbeHosts()
	for time.Now().Before(deadline) {
		if !browserProcessExistsFn(pid) {
			return errors.New("chrome process exited before port became ready")
		}
		for _, host := range hosts {
			conn, err := browserDialTimeoutFn("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 200*time.Millisecond)
			if err != nil {
				continue
			}
			_ = conn.Close()
			isCDP, cdpErr := browserPortCDPCheckFn(port)
			if cdpErr != nil {
				return cdpErr
			}
			if isCDP {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("chrome did not become ready as CDP on port %d", port)
}

func browserWaitForCDPShutdown(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		isCDP, err := browserPortCDPCheckFn(port)
		if err != nil {
			return err
		}
		if !isCDP {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("chrome did not stop exposing CDP on port %d", port)
}

func browserIsCDPPort(port int) (bool, error) {
	info, err := browserCDPVersionFn(port)
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(info.WebSocketDebuggerURL) != "", nil
}

func browserFetchCDPVersion(port int) (browserCDPVersion, error) {
	var lastErr error
	for _, host := range browserCDPProbeHosts() {
		payload, err := browserFetchCDPVersionFromHost(host, port)
		if err == nil {
			return payload, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return browserCDPVersion{}, lastErr
	}
	return browserCDPVersion{}, fmt.Errorf("could not resolve CDP version on port %d", port)
}

func browserFetchCDPVersionFromHost(host string, port int) (browserCDPVersion, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		host = "127.0.0.1"
	}
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Get(browserCDPHTTPURL(host, port, "/json/version"))
	if err != nil {
		return browserCDPVersion{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return browserCDPVersion{}, fmt.Errorf("unexpected /json/version status: %s", resp.Status)
	}
	var payload browserCDPVersion
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return browserCDPVersion{}, err
	}
	if normalized, err := browserNormalizeCDPWebSocketURL(payload.WebSocketDebuggerURL, host, port); err == nil {
		payload.WebSocketDebuggerURL = normalized
	}
	return payload, nil
}

func browserResolveLiveCDPEndpoint(port int) (string, error) {
	info, err := browserCDPVersionFn(port)
	if err != nil {
		return "", err
	}
	cdp := strings.TrimSpace(info.WebSocketDebuggerURL)
	if cdp == "" {
		return "", fmt.Errorf("chrome on port %d returned empty webSocketDebuggerUrl", port)
	}
	return cdp, nil
}

func browserCDPProbeHosts() []string {
	return []string{"127.0.0.1", "::1"}
}

func browserUniqueHosts(hosts []string) []string {
	unique := make([]string, 0, len(hosts))
	seen := make(map[string]struct{}, len(hosts))
	for _, host := range hosts {
		host = strings.TrimSpace(host)
		if host == "" {
			continue
		}
		if _, ok := seen[host]; ok {
			continue
		}
		seen[host] = struct{}{}
		unique = append(unique, host)
	}
	if len(unique) == 0 {
		return []string{"127.0.0.1", "::1"}
	}
	return unique
}

func browserCDPHTTPURL(host string, port int, path string) string {
	if strings.TrimSpace(path) == "" {
		path = "/"
	}
	return (&url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(strings.TrimSpace(host), strconv.Itoa(port)),
		Path:   path,
	}).String()
}

func browserNormalizeCDPWebSocketURL(rawURL, host string, port int) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if !browserHostNeedsRewrite(parsed.Hostname()) {
		return rawURL, nil
	}
	targetPort := parsed.Port()
	if targetPort == "" && port > 0 {
		targetPort = strconv.Itoa(port)
	}
	if targetPort == "" {
		return rawURL, nil
	}
	parsed.Host = net.JoinHostPort(strings.TrimSpace(host), targetPort)
	return parsed.String(), nil
}

func browserHostNeedsRewrite(host string) bool {
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	if host == "" || strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsUnspecified()
}

func browserLookupPIDByPort(port int) (int, error) {
	switch runtime.GOOS {
	case "windows":
		out, err := exec.Command("netstat", "-ano", "-p", "tcp").Output()
		if err != nil {
			return 0, nil
		}
		target := ":" + strconv.Itoa(port)
		for _, line := range strings.Split(string(out), "\n") {
			fields := strings.Fields(strings.TrimSpace(line))
			if len(fields) < 5 || !strings.EqualFold(fields[0], "TCP") {
				continue
			}
			if !strings.HasSuffix(fields[1], target) || !strings.EqualFold(fields[3], "LISTENING") {
				continue
			}
			pid, err := strconv.Atoi(fields[4])
			if err == nil && pid > 0 {
				return pid, nil
			}
		}
		return 0, nil
	default:
		out, err := exec.Command("lsof", "-nP", "-iTCP:"+strconv.Itoa(port), "-sTCP:LISTEN", "-t").Output()
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) && len(exitErr.Stderr) == 0 {
				return 0, nil
			}
			return 0, nil
		}
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			pid, err := strconv.Atoi(line)
			if err == nil && pid > 0 {
				return pid, nil
			}
		}
		return 0, nil
	}
}

func browserLogClosureEvent(reason string, fields map[string]any) {
	payload := map[string]any{
		"event":     "browser_instance_close",
		"reason":    strings.TrimSpace(reason),
		"timestamp": browserNowFn().Format(time.RFC3339Nano),
	}
	for key, value := range fields {
		payload[key] = value
	}
	browserAppendLogJSON(payload)
}

func browserLogWriter(path string) (io.WriteCloser, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return &lumberjack.Logger{
		Filename:   path,
		MaxSize:    browserLogMaxSizeMB,
		MaxBackups: browserLogMaxFiles - 1,
		LocalTime:  true,
		Compress:   false,
	}, nil
}

func browserEnsurePluginLogFile() error {
	logPath, err := browserDefaultLogPath()
	if err != nil {
		return err
	}
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	return file.Close()
}

func browserLogInstanceListEvent(command, stage string, items []browserInstanceRecord) {
	payload := map[string]any{
		"event":     "browser_instance_list",
		"command":   strings.TrimSpace(command),
		"stage":     strings.TrimSpace(stage),
		"timestamp": browserNowFn().Format(time.RFC3339Nano),
		"items":     items,
	}
	browserAppendLogJSON(payload)
}

func browserLogPluginDaemonEvent(command string, result *browserplaywrightsvc.StartResult) {
	if result == nil {
		return
	}
	payload := map[string]any{
		"event":     "browser_plugin_daemon",
		"command":   strings.TrimSpace(command),
		"status":    strings.TrimSpace(result.Status),
		"timestamp": browserNowFn().Format(time.RFC3339Nano),
		"pidFile":   strings.TrimSpace(result.PIDFile),
		"logFile":   strings.TrimSpace(result.LogFile),
		"addr":      strings.TrimSpace(result.Addr),
	}
	if result.PID > 0 {
		payload["pid"] = result.PID
	}
	if reason := strings.TrimSpace(result.Reason); reason != "" {
		payload["reason"] = reason
	}
	browserAppendLogJSON(payload)
}

func browserCreateTrace(stage string, fields map[string]any) {
	payload := map[string]any{
		"event":     "browser_create_trace",
		"stage":     strings.TrimSpace(stage),
		"timestamp": browserNowFn().Format(time.RFC3339Nano),
	}
	for key, value := range fields {
		payload[key] = value
	}
	browserAppendLogJSON(payload)
}

func browserShutdownTrace(stage string, fields map[string]any) {
	payload := map[string]any{
		"event":     "browser_shutdown_trace",
		"stage":     strings.TrimSpace(stage),
		"timestamp": browserNowFn().Format(time.RFC3339Nano),
	}
	for key, value := range fields {
		payload[key] = value
	}
	browserAppendLogJSON(payload)
}

func browserInstanceUserDataProgressTracer(stage string, fields map[string]any) func(browserArchiveProgress) {
	lastLoggedAt := time.Time{}
	lastFiles := -1
	var lastBytes int64 = -1
	return func(progress browserArchiveProgress) {
		now := time.Now()
		shouldLog := lastLoggedAt.IsZero() ||
			now.Sub(lastLoggedAt) >= 2*time.Second ||
			progress.Files-lastFiles >= 200 ||
			progress.Bytes-lastBytes >= 128*1024*1024
		if !shouldLog {
			return
		}
		payload := map[string]any{}
		for key, value := range fields {
			payload[key] = value
		}
		payload["files"] = progress.Files
		payload["dirs"] = progress.Dirs
		payload["bytes"] = progress.Bytes
		if strings.TrimSpace(progress.Path) != "" {
			payload["path"] = progress.Path
		}
		browserCreateTrace(strings.TrimSpace(stage), payload)
		lastLoggedAt = now
		lastFiles = progress.Files
		lastBytes = progress.Bytes
	}
}

func browserInstanceUserDataSkipTracer(stage string, fields map[string]any) func(string, error) {
	return func(path string, err error) {
		payload := map[string]any{}
		for key, value := range fields {
			payload[key] = value
		}
		if strings.TrimSpace(path) != "" {
			payload["path"] = path
		}
		if err != nil {
			payload["error"] = err.Error()
		}
		browserCreateTrace(strings.TrimSpace(stage), payload)
	}
}

func browserDefaultLogPath() (string, error) {
	execPath, err := browserExecutablePathFn()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(execPath); err == nil {
		execPath = resolved
	}
	dir := filepath.Dir(execPath)
	base := strings.TrimSpace(filepath.Base(execPath))
	if base == "" || base == "." || base == string(filepath.Separator) {
		base = "browser"
	}
	return filepath.Join(dir, base+".log"), nil
}

func browserUpsertInstanceState(items []browserInstanceStateRecord, next browserInstanceStateRecord) []browserInstanceStateRecord {
	next.AgentID = browserNormalizeIdentityPart(next.AgentID)
	next.ChatID = browserNormalizeIdentityPart(next.ChatID)
	for idx := range items {
		if browserNormalizeIdentityPart(items[idx].ChatID) == next.ChatID {
			items[idx] = next
			browserSortInstances(items)
			return items
		}
	}
	items = append(items, next)
	browserSortInstances(items)
	return items
}

func browserProcessExists(pid int) bool {
	return browserProcessExistsByPID(pid)
}

func browserTerminateProcess(pid int) error {
	return browserTerminateProcessByPID(pid)
}

func browserTerminateManagedInstance(item browserInstanceRecord) error {
	if isWSL, err := browserWSLDetectFn(); err == nil && isWSL {
		return browserTerminateWSLManagedInstance(item)
	}
	if item.PID > 0 {
		if err := browserTerminateProcessFn(item.PID); err != nil {
			return err
		}
		if item.Port > 0 {
			return browserTerminateManagedInstanceByPort(item.Port)
		}
		return nil
	}
	if item.Port > 0 {
		return browserTerminateManagedInstanceByPort(item.Port)
	}
	return nil
}

func browserTerminateManagedInstanceByPort(port int) error {
	if port <= 0 {
		return nil
	}
	if _, err := browserResolveLiveCDPEndpoint(port); err != nil {
		return nil
	}
	terminated := false
	seen := map[int]struct{}{}
	for attempts := 0; attempts < 8; attempts++ {
		pid, err := browserPortPIDLookupFn(port)
		if err != nil {
			return err
		}
		if pid <= 0 {
			break
		}
		if _, ok := seen[pid]; ok {
			break
		}
		seen[pid] = struct{}{}
		if err := browserTerminateProcessFn(pid); err != nil {
			return err
		}
		terminated = true
		if err := browserWaitForCDPShutdownFn(port, 500*time.Millisecond); err == nil {
			return nil
		}
	}
	if _, err := browserResolveLiveCDPEndpoint(port); err == nil {
		if terminated {
			return fmt.Errorf("managed chrome cdp still live on port %d after termination", port)
		}
		return fmt.Errorf("managed chrome cdp still live on port %d", port)
	}
	return nil
}

func browserInstanceKey(_ string, chatID string) string {
	return browserNormalizeIdentityPart(chatID)
}

func browserInstanceExpiredMinutes(flags map[string]string) (int, error) {
	expiredMinutes, err := connectsvc.IntValue(flags, "browser_expired", browserDefaultExpiredMinutes)
	if err != nil {
		return 0, err
	}
	if expiredMinutes <= 0 {
		return 0, errors.New("browser_expired must be positive")
	}
	return expiredMinutes, nil
}

func browserInstanceExpiredDuration(expiredMinutes int) time.Duration {
	if expiredMinutes <= 0 {
		expiredMinutes = browserDefaultExpiredMinutes
	}
	return time.Duration(expiredMinutes) * time.Minute
}

func (item browserInstanceStateRecord) apiRecord() browserInstanceRecord {
	return browserInstanceRecord{
		AgentID: browserNormalizeIdentityPart(item.AgentID),
		ChatID:  browserNormalizeIdentityPart(item.ChatID),
		Port:    item.Port,
		PID:     item.PID,
		CDP:     item.CDP,
	}
}

func browserAPIRecords(items []browserInstanceStateRecord) []browserInstanceRecord {
	out := make([]browserInstanceRecord, 0, len(items))
	for _, item := range items {
		out = append(out, item.apiRecord())
	}
	return out
}

func browserAPIRecordsWithProfile(flags map[string]string, items []browserInstanceStateRecord) []browserInstanceRecord {
	out := make([]browserInstanceRecord, 0, len(items))
	for _, item := range items {
		out = append(out, browserInstanceAPIRecord(flags, item))
	}
	return out
}

func browserInstanceAPIRecord(flags map[string]string, item browserInstanceStateRecord) browserInstanceRecord {
	record := item.apiRecord()
	if isWSL, err := browserWSLDetectFn(); err == nil && isWSL {
		if profileDir, ok := browserWSLInstanceLookupUserDataDir(record.AgentID, record.ChatID); ok {
			record.ProfileDir = profileDir
			return record
		}
	}
	profilePaths, err := browserResolveChromeProfileDir(flags, record.AgentID, item.Port)
	if err == nil {
		record.ProfileDir = browserChromeProfileDisplayDir(profilePaths)
	}
	return record
}

func browserChromeProfileDisplayDir(paths browserChromeProfilePaths) string {
	if launch := strings.TrimSpace(paths.Launch); launch != "" {
		return launch
	}
	return strings.TrimSpace(paths.Local)
}

func browserFormatActivityTime(ts time.Time) string {
	return ts.UTC().Format(time.RFC3339Nano)
}

func browserParseActivityTime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return browserNowFn()
	}
	ts, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return browserNowFn()
	}
	return ts
}

func instanceCDPEndpoint(port int) string {
	return fmt.Sprintf("ws://127.0.0.1:%d/devtools/browser", port)
}
