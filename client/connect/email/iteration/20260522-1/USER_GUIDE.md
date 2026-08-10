# Email Iteration 20260522-1 User Guide

## 需求目录

- 迭代需求文档：[REQUIREMENT.md](REQUIREMENT.md)
- 邮件插件主需求：[../REQUIREMENT.md](../../REQUIREMENT.md)
- 邮件插件主手册：[../USER_GUIDE.md](../../USER_GUIDE.md)

## 本次迭代内容

本次迭代把邮件回复链路收紧为单一协议：

- `send` / `init` 只识别 `connect add-request` 请求中的 `rawRequest`
- `rawRequest` 只接受最新 envelope JSON
- 回复锚点只从 `rawRequest.message.messageId` 提取
- 回复目标邮箱只从 `rawRequest.message.from` 提取
- 不再兼容 `original`、裸邮件原文和其他历史字段路径

## 原始报文结构

`rawRequest` 必须是以下结构：

```json
{
  "source": "email",
  "receivedAt": "2026-05-22T00:00:00+08:00",
  "message": {
    "uid": "uid-1",
    "messageId": "<origin@example.com>",
    "subject": "原始主题",
    "from": "Sender <sender@example.com>",
    "content": "hello",
    "artifacts": [],
    "raw": "{\"headers\":[...],\"content\":\"hello\"}"
  }
}
```

其中：

- `message.messageId` 必填
- `message.from` 必填
- `message.subject` 可选
- `message.raw` 可选；如果存在，会用于补充 `References` / `In-Reply-To`

## 日志

执行 `send` / `init` 时，`email.log` 会按阶段追加日志：

- `stage=send-request`：原始请求报文、文本、图片、文件参数
- `stage=send-parse`：解析出的父消息 ID、目标邮箱、主题、`rawRequest`
- `stage=send-result`：最终发送结果，以及实际报送的顶层邮件报文头 `smtp_header`
- `stage=send-failed`：失败阶段和失败原因；如果已经组装出待发送邮件，也会附带 `smtp_header`

失败场景例如：

- `rawRequest` 缺失
- `rawRequest` 不是合法 JSON
- `rawRequest.message.messageId` 缺失
- `rawRequest.message.from` 缺失

这些场景都会直接失败，并在 `email.log` 中留下明确错误记录。
