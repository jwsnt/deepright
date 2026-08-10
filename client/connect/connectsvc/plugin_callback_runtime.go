package connectsvc

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

type PluginCallbackRuntimeOptions struct {
	ListMeta          func() ([]MetaConfig, error)
	NormalizeKey      func(string) string
	ResolveConnectBin func() string
	Init              func(callbackPath string, flags map[string]string) error
	Send              func(callbackPath string, flags map[string]string) error
	SkipInitError     func(error) bool
	SkipSendError     func(error) bool
	Logger            func(format string, args ...interface{})
}

type PendingPluginRuntimeOptions struct {
	Service                  *Service
	QueryDB                  *sql.DB
	BuildTextContent         func([]Request) string
	DecidePlan               func(meta Meta, reqs []Request, now time.Time, textContent string) (*PendingRequestPlan, error)
	CreateTask               func(meta *Meta, reqs []Request, detailStarted int) (any, string, error)
	AfterCreate              func(meta Meta, selected []Request, plan PendingRequestPlan, created any) error
	NotifyStarted            func(callbacks map[string]string, meta Meta, request Request) error
	DispatchCompletedReplies func(callbacks map[string]string, svc *Service)
	Logger                   func(format string, args ...interface{})
}

type PendingRequestPolicy struct {
	MinNotifyAge       time.Duration
	ExpireAge          time.Duration
	PendingDetailState int
	ExpiredDetailState int
}

func DecidePendingPluginPlan(meta Meta, reqs []Request, now time.Time, textContent string, policy PendingRequestPolicy) (*PendingRequestPlan, error) {
	if strings.TrimSpace(textContent) == "" {
		return nil, nil
	}
	oldestAt, err := OldestRequestTime(reqs)
	if err != nil {
		return nil, err
	}
	age := now.Sub(oldestAt)
	if age < 0 {
		age = 0
	}
	if policy.ExpireAge > 0 && age >= policy.ExpireAge {
		expiredState := policy.ExpiredDetailState
		if expiredState < 0 {
			expiredState = 2
		}
		return &PendingRequestPlan{
			DetailStarted: expiredState,
			NextStatus:    RequestStatusExpired,
			Notify:        false,
		}, nil
	}
	if policy.MinNotifyAge > 0 && age < policy.MinNotifyAge {
		return nil, nil
	}
	pendingState := policy.PendingDetailState
	if pendingState < 0 {
		pendingState = 0
	}
	return &PendingRequestPlan{
		DetailStarted: pendingState,
		NextStatus:    RequestStatusStarted,
		Notify:        true,
	}, nil
}

func PluginRuntimeKey(meta Meta, normalizeKey func(string) string) string {
	return NormalizePluginCallbackKey(firstNonEmpty(meta.Key, meta.Name), normalizeKey)
}

func ListPluginCallbacks(listMeta func() ([]MetaConfig, error), normalizeKey func(string) string) (map[string]string, error) {
	if listMeta == nil {
		return nil, nil
	}
	items, err := listMeta()
	if err != nil {
		return nil, err
	}
	return BuildPluginCallbackMapFromMeta(items, normalizeKey), nil
}

func ResolvePluginRuntimeCallback(callbacks map[string]string, pluginName string, normalizeKey func(string) string) (string, error) {
	return ResolvePluginCallback(callbacks, NormalizePluginCallbackKey(pluginName, normalizeKey))
}

func NotifyPluginStarted(callbacks map[string]string, meta Meta, request Request, reply string, opts PluginCallbackRuntimeOptions) error {
	pluginKey := PluginRuntimeKey(meta, opts.NormalizeKey)
	if pluginKey == "" {
		return nil
	}
	if callbacks == nil {
		var err error
		callbacks, err = ListPluginCallbacks(opts.ListMeta, opts.NormalizeKey)
		if err != nil {
			return err
		}
	}
	callbackPath, err := ResolvePluginRuntimeCallback(callbacks, pluginKey, opts.NormalizeKey)
	if err != nil {
		return err
	}
	connectBin := ""
	if opts.ResolveConnectBin != nil {
		connectBin = opts.ResolveConnectBin()
	}
	flags, err := BuildPluginCallbackFlags(pluginKey, request, ResolveAutoReply(reply), connectBin)
	if err != nil {
		return err
	}
	if opts.Init == nil {
		return nil
	}
	if err := opts.Init(callbackPath, flags); err != nil {
		return err
	}
	return nil
}

func DispatchCompletedPluginReplies(
	callbacks map[string]string,
	queryDB *sql.DB,
	svc *Service,
	defaultTaskType string,
	sinceUnix int64,
	markDetailReplied func(detailID int) error,
	opts PluginCallbackRuntimeOptions,
) {
	if queryDB == nil || svc == nil || markDetailReplied == nil {
		return
	}
	if callbacks == nil {
		var err error
		callbacks, err = ListPluginCallbacks(opts.ListMeta, opts.NormalizeKey)
		if err != nil {
			logDispatch(opts.Logger, "[connect-cron] list plugin callbacks failed: %v", err)
			return
		}
	}
	connectBinFn := opts.ResolveConnectBin
	DispatchCompletedReplies(CompletedReplyDispatchOptions{
		QueryDB:         queryDB,
		Service:         svc,
		DefaultTaskType: defaultTaskType,
		SinceUnix:       sinceUnix,
		ResolveCallback: func(pluginKey string) (string, error) {
			return ResolvePluginRuntimeCallback(callbacks, pluginKey, opts.NormalizeKey)
		},
		SendReply: opts.Send,
		BuildFlags: func(pluginKey string, request Request, replyText string) (map[string]string, error) {
			connectBin := ""
			if connectBinFn != nil {
				connectBin = connectBinFn()
			}
			return BuildPluginCallbackFlags(NormalizePluginCallbackKey(pluginKey, opts.NormalizeKey), request, replyText, connectBin)
		},
		MarkDetailReplied: markDetailReplied,
		Logger:            opts.Logger,
	})
}

func SyncPendingPluginRuntime(callbacks map[string]string, opts PendingPluginRuntimeOptions) []PendingRequestNotification {
	if opts.Service == nil {
		return nil
	}
	notifications := SyncPendingRequests(PendingRequestSyncOptions{
		Service:          opts.Service,
		Logger:           opts.Logger,
		RuntimeKey:       nil,
		BuildTextContent: opts.BuildTextContent,
		DecidePlan:       opts.DecidePlan,
		CreateTask:       opts.CreateTask,
		AfterCreate:      opts.AfterCreate,
	})
	for _, item := range notifications {
		if opts.NotifyStarted == nil {
			continue
		}
		if err := opts.NotifyStarted(callbacks, item.Meta, item.Request); err != nil {
			logDispatch(opts.Logger, "[connect-cron] notify plugin started failed: %s request=%d: %v", item.RuntimeKey, item.Request.ID, err)
		}
	}
	if opts.DispatchCompletedReplies != nil {
		opts.DispatchCompletedReplies(callbacks, opts.Service)
	}
	return notifications
}

func ExtractSSEAssistantText(payload []byte) string {
	if len(payload) == 0 {
		return ""
	}
	lines := strings.Split(string(payload), "\n")
	var builder strings.Builder
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		if data == "" || data == "[DONE]" {
			continue
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		for _, choice := range chunk.Choices {
			builder.WriteString(choice.Delta.Content)
		}
	}
	return strings.TrimSpace(builder.String())
}

func NormalizePluginReplyContent(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	candidate := raw
	if unwrapped, ok := unwrapJSONMarkdownFence(candidate); ok {
		candidate = unwrapped
	}
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return raw
	}
	if normalized, ok := canonicalizeJSONObjectOrArray(candidate); ok {
		return normalized
	}
	return raw
}

func unwrapJSONMarkdownFence(raw string) (string, bool) {
	text := strings.TrimSpace(raw)
	if !strings.HasPrefix(text, "```") || !strings.HasSuffix(text, "```") {
		return "", false
	}
	lines := strings.Split(text, "\n")
	if len(lines) < 2 {
		return "", false
	}
	first := strings.TrimSpace(lines[0])
	if first != "```" && !strings.EqualFold(first, "```json") {
		return "", false
	}
	last := strings.TrimSpace(lines[len(lines)-1])
	if last != "```" {
		return "", false
	}
	return strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n")), true
}

func canonicalizeJSONObjectOrArray(raw string) (string, bool) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return "", false
	}
	if !(strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[")) {
		return "", false
	}
	var value any
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		return "", false
	}
	switch value.(type) {
	case map[string]any, []any:
	default:
		return "", false
	}
	out, err := json.Marshal(value)
	if err != nil {
		return "", false
	}
	return string(out), true
}
