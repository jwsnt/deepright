package feishusvc

import (
	"fmt"
	"strings"

	"connect/sharedutil"
)

func HumanizeLogEntry(at, content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return sharedutil.HumanLogLine(at, nil, "记录了一条飞书插件日志")
	}
	if strings.HasPrefix(content, "用参数：") || strings.Contains(content, "，做了：") {
		if strings.TrimSpace(at) == "" {
			return content
		}
		return strings.TrimSpace(at) + "，" + content
	}
	fields := sharedutil.ParseStructuredLogFields(content)
	stage := strings.TrimSpace(fields["stage"])
	switch stage {
	case "startup-connect":
		return sharedutil.HumanLogLine(at,
			[]string{sharedutil.HumanParam("插件", fields["name"])},
			"启动前检查了飞书插件连接配置")
	case "send-request":
		params := []string{
			sharedutil.HumanParam("动作", feishuActionLabel(fields["action"])),
			sharedutil.HumanParam("内容", firstNonEmptyFeishu(fields["content"], fields["message"])),
			countCSVParamFeishu("图片", fields["images"], "张"),
			countCSVParamFeishu("文件", fields["files"], "个"),
		}
		return sharedutil.HumanLogLine(at, params, "开始准备发送飞书消息")
	case "send-parse":
		params := []string{
			sharedutil.HumanParam("动作", feishuActionLabel(fields["action"])),
			sharedutil.HumanParam("回复对象", fields["reply_to"]),
		}
		return sharedutil.HumanLogLine(at, params, "确认了这条飞书消息要回复到哪里")
	case "send-result":
		params := []string{
			sharedutil.HumanParam("动作", feishuActionLabel(fields["action"])),
			sharedutil.HumanParam("会话", fields["target"]),
			sharedutil.HumanParam("回复对象", fields["reply_to"]),
			sharedutil.HumanParam("发送内容", humanTypesFeishu(fields["types"])),
		}
		return sharedutil.HumanLogLine(at, params, "成功发出了飞书消息")
	case "send-failed":
		params := []string{
			sharedutil.HumanParam("动作", feishuActionLabel(fields["action"])),
			sharedutil.HumanParam("环节", feishuStepLabel(fields["step"])),
			sharedutil.HumanParam("会话", fields["target"]),
			sharedutil.HumanParam("回复对象", fields["reply_to"]),
			sharedutil.HumanParam("发送内容", humanTypesFeishu(fields["types"])),
		}
		return sharedutil.HumanLogLine(at, params, "发送飞书消息失败，原因："+firstNonEmptyFeishu(fields["err"], "未返回具体原因"))
	case "send-retry":
		params := []string{
			sharedutil.HumanParam("环节", feishuStepLabel(fields["step"])),
			retryAttemptParamFeishu(fields["attempt"], fields["max"]),
		}
		return sharedutil.HumanLogLine(at, params, "发送飞书消息失败后正在重试，原因："+firstNonEmptyFeishu(fields["err"], "未返回具体原因"))
	case "send-terminate":
		params := []string{
			sharedutil.HumanParam("环节", feishuStepLabel(fields["step"])),
			sharedutil.HumanParam("最大重试次数", fields["max"]),
		}
		return sharedutil.HumanLogLine(at, params, "飞书消息连续发送失败，已停止继续重试，原因："+firstNonEmptyFeishu(fields["err"], "未返回具体原因"))
	case "push-request":
		params := []string{
			sharedutil.HumanParam("来源插件", fields["key"]),
			sharedutil.HumanParam("会话", firstNonEmptyFeishu(fields["chat_id"], fields["target"])),
			sharedutil.HumanParam("消息编号", fields["message_id"]),
			sharedutil.HumanParam("附件数量", fields["count"]),
			sharedutil.HumanParam("内容", fields["content"]),
		}
		return sharedutil.HumanLogLine(at, params, "把飞书消息整理成待处理任务")
	case "push-request-failed":
		params := []string{
			sharedutil.HumanParam("来源插件", fields["key"]),
			sharedutil.HumanParam("会话", firstNonEmptyFeishu(fields["chat_id"], fields["target"])),
			sharedutil.HumanParam("消息编号", fields["message_id"]),
			sharedutil.HumanParam("内容", fields["content"]),
		}
		return sharedutil.HumanLogLine(at, params, "整理飞书消息时失败，原因："+firstNonEmptyFeishu(fields["err"], "未返回具体原因"))
	case "service-retry":
		params := []string{
			sharedutil.HumanParam("环节", feishuStepLabel(fields["step"])),
			retryAttemptParamFeishu(fields["attempt"], fields["max"]),
		}
		return sharedutil.HumanLogLine(at, params, "飞书插件运行出错，正在自动重试，原因："+firstNonEmptyFeishu(fields["err"], "未返回具体原因"))
	case "service-terminate":
		params := []string{
			sharedutil.HumanParam("环节", feishuStepLabel(fields["step"])),
			sharedutil.HumanParam("最大重试次数", fields["max"]),
		}
		return sharedutil.HumanLogLine(at, params, "飞书插件连续运行失败，已停止自动重试，原因："+firstNonEmptyFeishu(fields["err"], "未返回具体原因"))
	case "message-expired":
		params := []string{
			sharedutil.HumanParam("外部编号", fields["external_id"]),
			sharedutil.HumanParam("消息编号", fields["message_id"]),
			sharedutil.HumanParam("消息类型", fields["type"]),
			sharedutil.HumanParam("内容", fields["content"]),
		}
		return sharedutil.HumanLogLine(at, params, "跳过了一条过期的飞书消息")
	default:
		return sharedutil.HumanLogLine(at,
			[]string{sharedutil.HumanParam("内容摘要", content)},
			"收到了一条飞书消息")
	}
}

func countCSVParamFeishu(label, raw, unit string) string {
	items := 0
	for _, part := range strings.Split(strings.TrimSpace(raw), ",") {
		if strings.TrimSpace(part) != "" {
			items++
		}
	}
	if items == 0 {
		return ""
	}
	return fmt.Sprintf("%s=%d%s", strings.TrimSpace(label), items, strings.TrimSpace(unit))
}

func retryAttemptParamFeishu(attempt, max string) string {
	attempt = strings.TrimSpace(attempt)
	max = strings.TrimSpace(max)
	if attempt == "" && max == "" {
		return ""
	}
	if attempt == "" {
		return "重试进度=共" + max + "次"
	}
	if max == "" {
		return "重试进度=第" + attempt + "次"
	}
	return "重试进度=第" + attempt + "/" + max + "次"
}

func feishuActionLabel(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "init":
		return "发送开始前通知"
	case "send":
		return "发送结果通知"
	default:
		return firstNonEmptyFeishu(strings.TrimSpace(action), "发送飞书消息")
	}
}

func feishuStepLabel(step string) string {
	switch strings.ToLower(strings.TrimSpace(step)) {
	case "load-meta":
		return "读取插件配置"
	case "parse-message":
		return "解析消息内容"
	case "validate-input":
		return "检查发送参数"
	case "build-api":
		return "准备飞书接口"
	case "send-image":
		return "发送图片"
	case "send-file":
		return "发送文件"
	case "send-text":
		return "发送文字"
	case "send-text-fallback":
		return "改用备用文字发送"
	default:
		return firstNonEmptyFeishu(strings.TrimSpace(step), "处理飞书消息")
	}
}

func humanTypesFeishu(types string) string {
	types = strings.TrimSpace(types)
	if types == "" {
		return ""
	}
	return strings.NewReplacer(
		"text", "文字",
		"image", "图片",
		"file", "文件",
	).Replace(types)
}

func firstNonEmptyFeishu(values ...string) string {
	for _, value := range values {
		value = sharedutil.SummarizeLogText(value, 120)
		if value != "" {
			return value
		}
	}
	return ""
}
