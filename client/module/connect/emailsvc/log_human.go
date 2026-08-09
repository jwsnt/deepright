package emailsvc

import (
	"fmt"
	"strings"

	"connect/sharedutil"
)

func HumanizeLogEntry(at, content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return sharedutil.HumanLogLine(at, nil, "记录了一条邮件插件日志")
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
	case "load-config":
		params := []string{
			sharedutil.HumanParam("插件", fields["key"]),
			sharedutil.HumanParam("Agent", fields["agent_id"]),
			sharedutil.HumanParam("会话", fields["chat_id"]),
			sharedutil.HumanParam("扫描间隔", firstNonEmptyLog(fields["scan_seconds"], fields["email_pop3_interval"], fields["email_pop3_seconds"])),
		}
		return sharedutil.HumanLogLine(at, params, "读取了邮件插件配置")
	case "meta-update":
		params := []string{
			sharedutil.HumanParam("插件", fields["key"]),
			sharedutil.HumanParam("Agent", fields["agent_id"]),
			sharedutil.HumanParam("会话", fields["chat_id"]),
			sharedutil.HumanParam("原因", fields["reason"]),
		}
		return sharedutil.HumanLogLine(at, params, "更新了邮件插件配置")
	case "startup-connect":
		return sharedutil.HumanLogLine(at,
			[]string{sharedutil.HumanParam("插件", fields["name"])},
			"启动前检查了邮件插件连接配置")
	case "send-request":
		params := []string{
			sharedutil.HumanParam("动作", emailActionLabel(fields["action"])),
			sharedutil.HumanParam("内容", firstNonEmptyLog(fields["content"], fields["message"])),
			countCSVParam("图片", fields["images"], "张"),
			countCSVParam("文件", fields["files"], "个"),
		}
		return sharedutil.HumanLogLine(at, params, "开始准备发送邮件")
	case "send-parse":
		params := []string{
			sharedutil.HumanParam("动作", emailActionLabel(fields["action"])),
			sharedutil.HumanParam("回复对象", fields["reply_to"]),
			sharedutil.HumanParam("收件人", fields["to"]),
			sharedutil.HumanParam("主题", fields["subject"]),
		}
		return sharedutil.HumanLogLine(at, params, "确认了这封邮件要发给谁")
	case "send-result":
		params := []string{
			sharedutil.HumanParam("动作", emailActionLabel(fields["action"])),
			sharedutil.HumanParam("收件人", fields["to"]),
			sharedutil.HumanParam("回复对象", fields["reply_to"]),
			sharedutil.HumanParam("发送内容", humanTypes(fields["types"])),
		}
		return sharedutil.HumanLogLine(at, params, "成功发出了邮件")
	case "send-failed":
		params := []string{
			sharedutil.HumanParam("动作", emailActionLabel(fields["action"])),
			sharedutil.HumanParam("环节", emailStepLabel(fields["step"])),
			sharedutil.HumanParam("收件人", fields["to"]),
			sharedutil.HumanParam("回复对象", firstNonEmptyLog(fields["reply_to"], fields["message_id"])),
			sharedutil.HumanParam("发送内容", humanTypes(fields["types"])),
		}
		return sharedutil.HumanLogLine(at, params, "发送邮件失败，原因："+firstNonEmptyLog(fields["err"], "未返回具体原因"))
	case "send-retry":
		params := []string{
			sharedutil.HumanParam("动作", emailActionLabel(fields["action"])),
			sharedutil.HumanParam("环节", emailStepLabel(fields["step"])),
			retryAttemptParam(fields["attempt"], fields["max"]),
		}
		return sharedutil.HumanLogLine(at, params, "发送邮件失败后正在重试，原因："+firstNonEmptyLog(fields["err"], "未返回具体原因"))
	case "send-terminate":
		params := []string{
			sharedutil.HumanParam("动作", emailActionLabel(fields["action"])),
			sharedutil.HumanParam("环节", emailStepLabel(fields["step"])),
			sharedutil.HumanParam("最大重试次数", fields["max"]),
		}
		return sharedutil.HumanLogLine(at, params, "邮件连续失败，已停止继续重试，原因："+firstNonEmptyLog(fields["err"], "未返回具体原因"))
	case "mail-received":
		params := []string{
			sharedutil.HumanParam("发件人", fields["from"]),
			sharedutil.HumanParam("主题", fields["subject"]),
			sharedutil.HumanParam("邮件编号", fields["message_id"]),
		}
		return sharedutil.HumanLogLine(at, params, "收到了新邮件，内容摘要："+firstNonEmptyLog(fields["summary"], "无"))
	case "mail-skip":
		params := []string{
			sharedutil.HumanParam("发件人", fields["from"]),
			sharedutil.HumanParam("主题", fields["subject"]),
			sharedutil.HumanParam("邮件编号", fields["message_id"]),
			sharedutil.HumanParam("原因", emailSkipReason(fields["reason"])),
		}
		return sharedutil.HumanLogLine(at, params, "跳过了这封邮件")
	case "service-retry":
		params := []string{
			sharedutil.HumanParam("环节", emailStepLabel(fields["step"])),
			retryAttemptParam(fields["attempt"], fields["max"]),
		}
		return sharedutil.HumanLogLine(at, params, "邮件插件运行出错，正在自动重试，原因："+firstNonEmptyLog(fields["err"], "未返回具体原因"))
	case "service-terminate":
		params := []string{
			sharedutil.HumanParam("环节", emailStepLabel(fields["step"])),
			sharedutil.HumanParam("最大重试次数", fields["max"]),
		}
		return sharedutil.HumanLogLine(at, params, "邮件插件连续运行失败，已停止自动重试，原因："+firstNonEmptyLog(fields["err"], "未返回具体原因"))
	default:
		return sharedutil.HumanLogLine(at,
			[]string{sharedutil.HumanParam("原始内容", content)},
			"记录了一条邮件插件日志")
	}
}

func countCSVParam(label, raw, unit string) string {
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

func retryAttemptParam(attempt, max string) string {
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

func emailActionLabel(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "init":
		return "发送开始前通知"
	case "send":
		return "发送结果通知"
	default:
		return firstNonEmptyLog(strings.TrimSpace(action), "发送邮件")
	}
}

func emailStepLabel(step string) string {
	switch strings.ToLower(strings.TrimSpace(step)) {
	case "build-service":
		return "准备发送服务"
	case "load-meta":
		return "读取插件配置"
	case "parse-message":
		return "解析消息内容"
	case "send-structured":
		return "按标准方式发送"
	case "send-fallback":
		return "改用备用方式发送"
	case "send-exception":
		return "改发异常提醒"
	case "send-smtp":
		return "连接邮件服务器发送"
	default:
		return firstNonEmptyLog(strings.TrimSpace(step), "处理邮件")
	}
}

func emailSkipReason(reason string) string {
	switch strings.TrimSpace(reason) {
	case "duplicate-mail":
		return "这封邮件之前已经处理过了"
	default:
		return firstNonEmptyLog(strings.TrimSpace(reason), "不符合处理条件")
	}
}

func humanTypes(types string) string {
	types = strings.TrimSpace(types)
	if types == "" {
		return ""
	}
	return strings.NewReplacer(
		"text", "文字",
		"image", "图片",
		"file", "文件",
		"+", "+",
	).Replace(types)
}

func firstNonEmptyLog(values ...string) string {
	for _, value := range values {
		value = sharedutil.SummarizeLogText(value, 120)
		if value != "" {
			return value
		}
	}
	return ""
}
