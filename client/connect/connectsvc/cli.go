package connectsvc

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

func RunCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || IsHelpCommand(args[0]) {
		PrintHelp(stdout, "connect")
		return 0
	}

	command := args[0]
	flags, err := ParseFlags(args[1:])
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	switch command {
	case "start", "serve", "stop":
		return runServiceCommand(command, flags, stdout, stderr)
	case "list-plugins":
		items, err := ListPlugins(flags)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, items)
		return 0
	case "list-meta":
		return RunCLIWithClient(command, flags, NewAPIClient(ServiceBaseURLFromFlags(flags)), stdout, stderr)
	default:
		return RunCLIWithClient(command, flags, NewAPIClient(ServiceBaseURLFromFlags(flags)), stdout, stderr)
	}
}

func RunCLIWithService(command string, flags map[string]string, svc *Service, stdout, stderr io.Writer) int {
	switch command {
	case "list-plugins":
		items, err := ListPlugins(flags)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, items)
		return 0
	case "meta-create":
		item, err := svc.CreateMeta(MetaInput{
			Key:           FirstValue(flags, "key"),
			Meta:          FirstValue(flags, "meta"),
			Stream:        BoolValue(flags, "stream", false),
			Callback:      FirstValue(flags, "callback"),
			AgentID:       FirstValue(flags, "agent", "agentId"),
			ChatID:        FirstValue(flags, "chat", "chatId"),
			Model:         FirstValue(flags, "model"),
			Thinking:      BoolValue(flags, "thinking", false),
			RouterDisable: ResolveRouterDisableFlag(flags, true),
		})
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, item)
		return 0

	case "meta-update":
		key := FirstValue(flags, "key")
		if strings.TrimSpace(key) == "" {
			fmt.Fprintln(stderr, "key is required")
			return 1
		}
		patch := MetaUpdate{}
		if value, ok := flags["meta"]; ok {
			patch.Meta = &value
		}
		if _, ok := flags["stream"]; ok {
			stream := BoolValue(flags, "stream", false)
			patch.Stream = &stream
		}
		if value, ok := flags["callback"]; ok {
			patch.Callback = &value
		}
		if value := FirstValue(flags, "agent", "agentId"); value != "" {
			patch.AgentID = &value
		}
		if value := FirstValue(flags, "chat", "chatId"); value != "" {
			patch.ChatID = &value
		}
		if value, ok := flags["model"]; ok {
			patch.Model = &value
		}
		if _, ok := flags["thinking"]; ok {
			thinking := BoolValue(flags, "thinking", false)
			patch.Thinking = &thinking
		}
		if value, ok := ResolveRouterDisablePatch(flags); ok {
			patch.RouterDisable = &value
		}
		item, err := svc.UpdateMeta(key, patch)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, item)
		return 0

	case "meta-delete":
		item, err := svc.DeleteMeta(FirstValue(flags, "key"))
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, item)
		return 0

	case "meta-get":
		key := FirstValue(flags, "key")
		if strings.TrimSpace(key) == "" {
			fmt.Fprintln(stderr, "key is required")
			return 1
		}
		item, err := svc.GetMetaConfig(key, BoolValue(flags, "include-deleted", false))
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, item)
		return 0

	case "meta-list":
		items, err := svc.ListMeta(BoolValue(flags, "include-deleted", false))
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, items)
		return 0

	case "list-meta":
		items, err := svc.ListMetaConfig(BoolValue(flags, "include-deleted", false))
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, items)
		return 0

	case "add-request":
		status, err := IntValue(flags, "status", -1)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		var requestStatus *int
		if _, ok := flags["status"]; ok {
			requestStatus = &status
		}
		item, err := svc.AddRequest(RequestInput{
			Key:             FirstValue(flags, "key"),
			ExternalID:      FirstValue(flags, "external-id", "externalId"),
			Content:         FirstValue(flags, "content"),
			Request:         FirstValue(flags, "request"),
			Artifacts:       FirstValue(flags, "artifacts"),
			Original:        FirstValue(flags, "original"),
			RawRequest:      FirstValue(flags, "raw-request", "rawRequest"),
			ResponseSchema:  FirstValue(flags, "schema"),
			MessageSnapshot: FirstValue(flags, "message-snapshot"),
			Status:          requestStatus,
			CreatedAt:       FirstValue(flags, "created"),
		})
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, item)
		return 0

	case "request-list":
		afterID, err := IntValue(flags, "after-id", 0)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		beforeID, err := IntValue(flags, "before-id", 0)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		limit, err := IntValue(flags, "limit", 100)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		status, err := IntValue(flags, "status", -1)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		var requestStatus *int
		if _, ok := flags["status"]; ok {
			requestStatus = &status
		}
		items, err := svc.ListRequests(RequestFilter{
			Key:      FirstValue(flags, "key"),
			AfterID:  afterID,
			BeforeID: beforeID,
			Limit:    limit,
			Status:   requestStatus,
		})
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, items)
		return 0

	case "message-snapshot-senders":
		window, err := ParseMessageSnapshotWindowHours(FirstValue(flags, "window-hours"))
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		items, err := svc.ListMessageSenders(FirstValue(flags, "source"), window, time.Now())
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, items)
		return 0

	case "message-snapshot-search":
		window, err := ParseMessageSnapshotWindowHours(FirstValue(flags, "window-hours"))
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		limit, err := IntValue(flags, "limit", messageSearchDefaultLimit)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		offset, err := IntValue(flags, "offset", 0)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		item, err := svc.SearchMessageSnapshots(window, MessageSnapshotSearch{
			Source:   FirstValue(flags, "source"),
			SenderID: FirstValue(flags, "sender-id"),
			Query:    FirstValue(flags, "query"),
			Limit:    limit,
			Offset:   offset,
		}, time.Now())
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, item)
		return 0

	case "add-response":
		requestID, err := IntValue(flags, "request-id", 0)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		item, err := svc.AddResponse(ResponseInput{
			Key:       FirstValue(flags, "key"),
			RequestID: requestID,
			Response:  FirstValue(flags, "response"),
			Artifacts: FirstValue(flags, "artifacts"),
		})
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, item)
		return 0

	case "response-list":
		requestID, err := IntValue(flags, "request-id", 0)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		afterID, err := IntValue(flags, "after-id", 0)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		limit, err := IntValue(flags, "limit", 100)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		items, err := svc.ListResponses(ResponseFilter{
			Key:       FirstValue(flags, "key"),
			RequestID: requestID,
			AfterID:   afterID,
			Limit:     limit,
		})
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, items)
		return 0
	}

	fmt.Fprintf(stderr, "unknown command: %s\n", command)
	PrintHelp(stderr, "connect")
	return 1
}

func RunCLIWithClient(command string, flags map[string]string, client *APIClient, stdout, stderr io.Writer) int {
	switch command {
	case "list-plugins":
		items, err := ListPlugins(flags)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, items)
		return 0
	case "meta-create":
		item, err := client.CreateMeta(MetaInput{
			Key:           FirstValue(flags, "key"),
			Meta:          FirstValue(flags, "meta"),
			Stream:        BoolValue(flags, "stream", false),
			Callback:      FirstValue(flags, "callback"),
			AgentID:       FirstValue(flags, "agent", "agentId"),
			ChatID:        FirstValue(flags, "chat", "chatId"),
			Model:         FirstValue(flags, "model"),
			Thinking:      BoolValue(flags, "thinking", false),
			RouterDisable: ResolveRouterDisableFlag(flags, true),
		})
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, item)
		return 0

	case "meta-update":
		key := FirstValue(flags, "key")
		if strings.TrimSpace(key) == "" {
			fmt.Fprintln(stderr, "key is required")
			return 1
		}
		patch := MetaUpdate{}
		if value, ok := flags["meta"]; ok {
			patch.Meta = &value
		}
		if _, ok := flags["stream"]; ok {
			stream := BoolValue(flags, "stream", false)
			patch.Stream = &stream
		}
		if value, ok := flags["callback"]; ok {
			patch.Callback = &value
		}
		if value := FirstValue(flags, "agent", "agentId"); value != "" {
			patch.AgentID = &value
		}
		if value := FirstValue(flags, "chat", "chatId"); value != "" {
			patch.ChatID = &value
		}
		if value, ok := flags["model"]; ok {
			patch.Model = &value
		}
		if _, ok := flags["thinking"]; ok {
			thinking := BoolValue(flags, "thinking", false)
			patch.Thinking = &thinking
		}
		if value, ok := ResolveRouterDisablePatch(flags); ok {
			patch.RouterDisable = &value
		}
		item, err := client.UpdateMeta(key, patch)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, item)
		return 0

	case "meta-delete":
		item, err := client.DeleteMeta(FirstValue(flags, "key"))
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, item)
		return 0

	case "meta-get":
		key := FirstValue(flags, "key")
		if strings.TrimSpace(key) == "" {
			fmt.Fprintln(stderr, "key is required")
			return 1
		}
		item, err := client.GetMetaConfig(key, BoolValue(flags, "include-deleted", false))
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, item)
		return 0

	case "meta-list":
		items, err := client.ListMeta(BoolValue(flags, "include-deleted", false))
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, items)
		return 0

	case "list-meta":
		items, err := client.ListMetaConfig(BoolValue(flags, "include-deleted", false))
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, items)
		return 0

	case "add-request":
		status, err := IntValue(flags, "status", -1)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		var requestStatus *int
		if _, ok := flags["status"]; ok {
			requestStatus = &status
		}
		item, err := client.AddRequest(RequestInput{
			Key:             FirstValue(flags, "key"),
			ExternalID:      FirstValue(flags, "external-id", "externalId"),
			Content:         FirstValue(flags, "content"),
			Request:         FirstValue(flags, "request"),
			Artifacts:       FirstValue(flags, "artifacts"),
			Original:        FirstValue(flags, "original"),
			RawRequest:      FirstValue(flags, "raw-request", "rawRequest"),
			ResponseSchema:  FirstValue(flags, "schema"),
			MessageSnapshot: FirstValue(flags, "message-snapshot"),
			Status:          requestStatus,
			CreatedAt:       FirstValue(flags, "created"),
		})
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, item)
		return 0

	case "request-list":
		afterID, err := IntValue(flags, "after-id", 0)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		beforeID, err := IntValue(flags, "before-id", 0)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		limit, err := IntValue(flags, "limit", 100)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		status, err := IntValue(flags, "status", -1)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		var requestStatus *int
		if _, ok := flags["status"]; ok {
			requestStatus = &status
		}
		items, err := client.ListRequests(RequestFilter{
			Key:      FirstValue(flags, "key"),
			AfterID:  afterID,
			BeforeID: beforeID,
			Limit:    limit,
			Status:   requestStatus,
		})
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, items)
		return 0

	case "add-response":
		requestID, err := IntValue(flags, "request-id", 0)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		item, err := client.AddResponse(ResponseInput{
			Key:       FirstValue(flags, "key"),
			RequestID: requestID,
			Response:  FirstValue(flags, "response"),
			Artifacts: FirstValue(flags, "artifacts"),
		})
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, item)
		return 0

	case "response-list":
		requestID, err := IntValue(flags, "request-id", 0)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		afterID, err := IntValue(flags, "after-id", 0)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		limit, err := IntValue(flags, "limit", 100)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		items, err := client.ListResponses(ResponseFilter{
			Key:       FirstValue(flags, "key"),
			RequestID: requestID,
			AfterID:   afterID,
			Limit:     limit,
		})
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		WriteJSON(stdout, items)
		return 0
	}

	fmt.Fprintf(stderr, "unknown command: %s\n", command)
	PrintHelp(stderr, "connect")
	return 1
}

func ParseFlags(args []string) (map[string]string, error) {
	flags := make(map[string]string)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			return nil, fmt.Errorf("invalid argument: %s", arg)
		}
		key := strings.TrimPrefix(arg, "--")
		value := "true"
		if parts := strings.SplitN(key, "=", 2); len(parts) == 2 {
			key = parts[0]
			value = parts[1]
		} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			value = args[i+1]
			i++
		}
		flags[key] = value
	}
	return flags, nil
}

func FirstValue(flags map[string]string, keys ...string) string {
	for _, key := range keys {
		if value, ok := flags[key]; ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func BoolValue(flags map[string]string, key string, fallback bool) bool {
	raw, ok := flags[key]
	if !ok {
		return fallback
	}
	parsed, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(raw)))
	if err == nil {
		return parsed
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "1", "yes", "y", "on":
		return true
	case "0", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func ResolveRouterDisableFlag(flags map[string]string, fallback bool) bool {
	if _, ok := flags["router_disable"]; ok {
		return BoolValue(flags, "router_disable", fallback)
	}
	return fallback
}

func ResolveRouterDisablePatch(flags map[string]string) (bool, bool) {
	if _, ok := flags["router_disable"]; ok {
		return BoolValue(flags, "router_disable", false), true
	}
	return false, false
}

func IntValue(flags map[string]string, key string, fallback int) (int, error) {
	raw, ok := flags[key]
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("%s must be integer", key)
	}
	return value, nil
}

func WriteJSON(w io.Writer, value any) {
	out, _ := json.MarshalIndent(value, "", "  ")
	fmt.Fprintln(w, string(out))
}

func IsHelpCommand(arg string) bool {
	switch strings.TrimSpace(arg) {
	case "help", "--help", "-h":
		return true
	default:
		return false
	}
}

func PrintHelp(w io.Writer, binary string) {
	if strings.TrimSpace(binary) == "" {
		binary = "connect"
	}
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintf(w, "  %s <command> [options]\n", binary)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Manual:")
	fmt.Fprintln(w, "  Prefer integration top-level commands for end-user operations; use connect help for")
	fmt.Fprintln(w, "  internal implementation details and standalone plugin bring-up.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  start           Start local connect HTTP service")
	fmt.Fprintln(w, "  stop            Stop local connect HTTP service")
	fmt.Fprintln(w, "  serve           Run local connect HTTP service in foreground")
	fmt.Fprintln(w, "  meta-create     Create connect metadata by plugin key")
	fmt.Fprintln(w, "  meta-update     Update connect metadata by plugin key")
	fmt.Fprintln(w, "  meta-delete     Soft-delete connect metadata by key")
	fmt.Fprintln(w, "  meta-get        Read one connect metadata record or plugin config by key")
	fmt.Fprintln(w, "  meta-list       List connect metadata")
	fmt.Fprintln(w, "  list-meta       List configured plugin meta payloads")
	fmt.Fprintln(w, "  list-plugins    Read plugin display metadata from ../plugins")
	fmt.Fprintln(w, "  add-request     Push one third-party request")
	fmt.Fprintln(w, "  request-list    Read request records")
	fmt.Fprintln(w, "  add-response    Push one processed response")
	fmt.Fprintln(w, "  response-list   Read response records")
	fmt.Fprintln(w, "  feishu          Manage FEISHU long-connection runtime")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Plugin workflow:")
	fmt.Fprintf(w, "  1. %s start --db ./data --agent-dir ../agent/test-case\n", binary)
	fmt.Fprintf(w, "  2. %s list-plugins --connect-cache 10000\n", binary)
	fmt.Fprintf(w, "  3. %s meta-create --key feishu --meta \"{\\\"appId\\\":\\\"cli-app\\\",\\\"appSecret\\\":\\\"cli-secret\\\"}\" --stream true --callback ignored --agent A --chatId chat-001 --model OpenAI\n", binary)
	fmt.Fprintf(w, "  4. %s meta-get --key feishu\n", binary)
	fmt.Fprintf(w, "  5. ../plugins/feishu start --connect-bin ./%s\n", binary)
	fmt.Fprintf(w, "  6. %s add-request --key feishu --externalId msg-1 --content \"HELLO WORLD\" --original \"{\\\"text\\\":\\\"HELLO WORLD\\\"}\"\n", binary)
	fmt.Fprintf(w, "  7. %s add-response --key feishu --request-id 1 --response \"HELLO BACK\"\n", binary)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Plugin notes:")
	fmt.Fprintln(w, "  - meta-get --key returns the normalized plugin runtime config view for plugins.")
	fmt.Fprintln(w, "  - connect start initializes and reuses one shared SQLite connection pool; embedded")
	fmt.Fprintln(w, "    integration/proxy mode reuses the same connectsvc.Service instead of reopening DB")
	fmt.Fprintln(w, "    files per request.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Common options:")
	fmt.Fprintln(w, "  --addr HOST:PORT           local service address; default 127.0.0.1:18080")
	fmt.Fprintln(w, "  --port PORT                local service port shorthand")
	fmt.Fprintln(w, "  --db PATH                  sqlite path; default CONNECT_DB or ../cron/data")
	fmt.Fprintln(w, "  --agent-dir DIR            agent root; also supports AGENT_DIR or ./agent")
	fmt.Fprintln(w, "  --connect-cache MS         cache ttl in ms, default 10000")
	fmt.Fprintln(w, "  --pid-file PATH            pid file for start/stop, default ./connect.pid")
	fmt.Fprintln(w, "  --log-file PATH            log file for start/serve, default ./connect.log")
	fmt.Fprintln(w, "  --foreground true          run start in foreground")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Meta create/update options:")
	fmt.Fprintln(w, "  --key NAME                 stable plugin key")
	fmt.Fprintln(w, "  --meta JSON                connect metadata json string")
	fmt.Fprintln(w, "  --stream true|false        whether stream is enabled")
	fmt.Fprintln(w, "  --callback VALUE           stored callback is fixed to plugins/<plugin-key>")
	fmt.Fprintln(w, "  --agent ID / --agentId ID  bound agent id")
	fmt.Fprintln(w, "  --chat ID / --chatId ID    chat id")
	fmt.Fprintln(w, "  --model NAME               registered model name")
	fmt.Fprintln(w, "  --thinking true|false      deep thinking")
	fmt.Fprintln(w, "  --router_disable true|false")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Request options:")
	fmt.Fprintln(w, "  --key NAME")
	fmt.Fprintln(w, "  --external-id ID")
	fmt.Fprintln(w, "  --content TEXT / --request TEXT")
	fmt.Fprintln(w, "  --artifacts PATHS          comma-separated artifact paths")
	fmt.Fprintln(w, "  --original TEXT / --raw-request TEXT")
	fmt.Fprintln(w, "  --schema JSON              optional json string; mapped to bridged task_detail.response_schema")
	fmt.Fprintln(w, "  --status N                 request status: 0 pending, 1 started, 2 completed, 3 expired, 4 replied")
	fmt.Fprintln(w, "  --created VALUE            unix timestamp or RFC3339 time")
	fmt.Fprintln(w, "  --after-id N")
	fmt.Fprintln(w, "  --before-id N              request-list: return records older than this id")
	fmt.Fprintln(w, "  --limit N")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Response options:")
	fmt.Fprintln(w, "  --key NAME")
	fmt.Fprintln(w, "  --request-id N")
	fmt.Fprintln(w, "  --response TEXT")
	fmt.Fprintln(w, "  --artifacts PATHS")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintf(w, "  %s start --port 18080 --db ./data --agent-dir ../agent/test-case\n", binary)
	fmt.Fprintf(w, "  %s list-plugins --connect-cache 10000\n", binary)
	fmt.Fprintf(w, "  %s list-meta\n", binary)
	fmt.Fprintf(w, "  %s meta-create --key feishu --meta \"{\\\"token\\\":\\\"...\\\"}\" --stream true --callback ignored --agent A --chatId chat-001 --model OpenAI\n", binary)
	fmt.Fprintf(w, "  %s add-request --key feishu --externalId ext-1 --content \"HELLO WORLD\" --artifacts \"/tmp/a.txt,/tmp/b.txt\" --original \"{\\\"text\\\":\\\"HELLO WORLD\\\"}\" --status 0 --created 1777852800\n", binary)
	fmt.Fprintf(w, "  %s add-request --key feishu --content \"提取结构化消息\" --schema \"{\\\"type\\\":\\\"object\\\",\\\"properties\\\":{\\\"reply\\\":{\\\"type\\\":\\\"string\\\"}}}\"\n", binary)
	fmt.Fprintf(w, "  %s add-response --key feishu --request-id 1 --response \"HELLO WORLD\"\n", binary)
}
