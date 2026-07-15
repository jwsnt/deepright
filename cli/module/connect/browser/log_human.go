package main

import (
	"fmt"
	"strings"
	"time"

	"connect/sharedutil"
)

func browserRenderLogLine(payload map[string]any) string {
	at := browserPayloadString(payload["timestamp"])
	if at == "" {
		at = browserNowFn().Format(time.RFC3339)
	}
	event := browserPayloadString(payload["event"])
	params := make([]string, 0, 12)
	appendParam := func(label string, value any) {
		if text := browserHumanParam(label, value); text != "" {
			params = append(params, text)
		}
	}
	if command := browserHumanExactParam("命令", payload["command"]); command != "" {
		params = append(params, command)
	}
	if request := browserHumanExactParam("请求", payload["request"]); request != "" {
		params = append(params, request)
	}
	if response := browserHumanExactParam("响应", payload["response"]); response != "" {
		params = append(params, response)
	}
	if stderr := browserHumanExactParam("标准错误", payload["stderr"]); stderr != "" {
		params = append(params, stderr)
	}
	appendParam("阶段", payload["stage"])
	appendParam("状态", payload["status"])
	appendParam("原因", payload["reason"])
	appendParam("会话", firstNonEmptyBrowser(browserPayloadString(payload["session"]), browserPayloadString(payload["chatId"])))
	appendParam("Agent", payload["agentId"])
	appendParam("端口", payload["port"])
	appendParam("进程", payload["pid"])
	appendParam("地址", payload["addr"])
	appendParam("页面", firstNonEmptyBrowser(browserPayloadString(payload["target"]), browserPayloadString(payload["url"]), browserPayloadString(payload["cdp"])))
	appendParam("Cookie 文件", payload["cookiePath"])
	appendParam("驱动目录", payload["driverDir"])
	if source := browserHumanExactParam("复制来源", payload["sourceDir"]); source != "" {
		params = append(params, source)
	}
	copyTarget := payload["targetDir"]
	if strings.TrimSpace(browserPayloadString(copyTarget)) == "" {
		copyTarget = payload["profileDir"]
	}
	if target := browserHumanExactParam("复制目标", copyTarget); target != "" {
		params = append(params, target)
	}
	appendParam("错误", payload["error"])
	appendParam("结果说明", payload["output"])
	if args, ok := payload["args"].([]string); ok && len(args) > 0 {
		params = append(params, fmt.Sprintf("参数数量=%d个", len(args)))
	}
	if items, ok := payload["items"].([]browserInstanceRecord); ok {
		params = append(params, fmt.Sprintf("实例数量=%d个", len(items)))
	}
	if rawItems, ok := payload["items"].([]any); ok {
		params = append(params, fmt.Sprintf("实例数量=%d个", len(rawItems)))
	}
	action := browserEventAction(event)
	return sharedutil.HumanLogLine(at, params, action)
}

func browserHumanParam(label string, value any) string {
	label = strings.TrimSpace(label)
	if label == "" || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return sharedutil.HumanParam(label, typed)
	case bool:
		if typed {
			return label + "=是"
		}
		return label + "=否"
	case int:
		if typed <= 0 {
			return ""
		}
		return fmt.Sprintf("%s=%d", label, typed)
	case int64:
		if typed <= 0 {
			return ""
		}
		return fmt.Sprintf("%s=%d", label, typed)
	case float64:
		if typed <= 0 {
			return ""
		}
		return fmt.Sprintf("%s=%d", label, int64(typed))
	default:
		return sharedutil.HumanParam(label, browserPayloadString(value))
	}
}

func browserHumanExactParam(label string, value any) string {
	label = strings.TrimSpace(label)
	if label == "" || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		typed = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(typed, "\r", " "), "\n", " "))
		typed = strings.Join(strings.Fields(typed), " ")
		if typed == "" {
			return ""
		}
		return label + "=" + typed
	case fmt.Stringer:
		return browserHumanExactParam(label, typed.String())
	default:
		return browserHumanParam(label, value)
	}
}

func browserPayloadString(value any) string {
	switch typed := value.(type) {
	case string:
		return sharedutil.SummarizeLogText(typed, 120)
	case fmt.Stringer:
		return sharedutil.SummarizeLogText(typed.String(), 120)
	case bool:
		if typed {
			return "是"
		}
		return "否"
	case int:
		return fmt.Sprintf("%d", typed)
	case int64:
		return fmt.Sprintf("%d", typed)
	case float64:
		return fmt.Sprintf("%d", int64(typed))
	default:
		return ""
	}
}

func browserEventAction(event string) string {
	switch strings.TrimSpace(event) {
	case "browser_instance_close":
		return "关闭了一个浏览器实例"
	case "browser_instance_shutdown_request":
		return "记录了浏览器实例关闭请求"
	case "browser_instance_list":
		return "查看了当前浏览器实例列表"
	case "browser_plugin_daemon":
		return "更新了浏览器插件后台状态"
	case "browser_create_trace":
		return "记录了浏览器实例的创建过程"
	case "browser_destroy_trace", "browser_shutdown_trace":
		return "记录了浏览器实例的关闭过程"
	case "browser_playwright_driver_preflight":
		return "检查了浏览器驱动是否可用"
	case "browser_cookie_preflight":
		return "检查了浏览器 Cookie 配置"
	default:
		return "记录了浏览器插件运行情况"
	}
}

func firstNonEmptyBrowser(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
