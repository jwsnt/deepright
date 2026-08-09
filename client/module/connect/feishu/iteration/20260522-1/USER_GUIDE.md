# Feishu Iteration 20260522-1 User Guide

## 需求目录

- 迭代需求文档：[REQUIREMENT.md](REQUIREMENT.md)
- 飞书插件主需求：[../REQUIREMENT.md](../../REQUIREMENT.md)
- 飞书插件主手册：[../USER_GUIDE.md](../../USER_GUIDE.md)

## 本次迭代内容

本次迭代把飞书回复链路收紧为单一协议：

- `send` / `init` 只识别 `connect add-request` 请求中的 `rawRequest`
- `rawRequest` 只接受最新 envelope JSON
- 回复锚点只从 `rawRequest.message.messageId` 提取
- 不再兼容裸飞书事件报文和其他历史字段路径

## 原始报文结构

`rawRequest` 必须是以下结构：

```json
{
  "source": "feishu",
  "receivedAt": "2026-05-22T00:00:00+08:00",
  "message": {
    "messageId": "om_xxx"
  },
  "pending": [],
  "groupedBy": "chat_id",
  "windowSecs": 30,
  "expireSecs": 600
}
```

其中 `message.messageId` 为必填；缺失时本次回复直接失败，不会推送。

## 当前行为

执行 `send` / `init` 时，插件会按以下顺序处理：

1. 记录本次命令收到的请求报文和发送参数
2. 从 `rawRequest.message.messageId` 解析原消息 ID
3. 解析成功后执行图片、文件、文本发送
4. 把最终发送结果或失败原因写入 `feishu.log`

## 日志

`feishu.log` 会追加以下阶段日志：

- `stage=send-request`：原始请求报文、文本、图片、文件参数
- `stage=send-parse`：解析出的 `reply_to` 和 `rawRequest`
- `stage=send-result`：发送结果
- `stage=send-failed`：失败阶段和失败原因

失败场景例如：

- `rawRequest` 缺失
- `rawRequest` 不是合法 JSON
- `rawRequest.message.messageId` 缺失

这些场景都会直接失败，并在 `feishu.log` 中留下明确错误记录。
