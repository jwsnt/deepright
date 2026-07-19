# 自定义收件人与主题

本迭代为邮件插件的 `send` 和 `init` 增加了新邮件发送模式。未提供原始邮件上下文时，可以通过 `--to` 与 `--subject` 指定收件人和主题；提供原始邮件上下文时，插件继续严格按原邮件回复。

## 新邮件模式

不传 `--message` 时，必须同时提供非空的 `--to` 和 `--subject`：

```bash
../plugins/email send \
  --to 'Alice <alice@example.com>,bob@example.com' \
  --subject '任务完成通知' \
  --content '任务已完成，请查收附件。' \
  --file /tmp/report.pdf
```

`--to` 使用 `,` 分隔多个收件人，支持普通邮箱地址和带显示名的地址。新邮件使用插件配置中的 `email` 作为发件人，不生成 `In-Reply-To` 或 `References`。

`init` 使用相同规则：

```bash
../plugins/email init \
  --to 'ops@example.com,owner@example.com' \
  --subject '任务已开始执行' \
  --content '初始化完成。'
```

## 回复模式

当 `--message` 中有 `rawRequest` 时，插件始终按回复模式发送。此时传入的 `--to` 和 `--subject` 会被忽略；收件人和主题只取自：

- `rawRequest.message.from`
- `rawRequest.message.subject`

插件会生成 `Re: 原主题`，并使用 `rawRequest.message.messageId` 写入 `In-Reply-To` 和 `References`。

## 校验规则

- 未传 `--message`，或合法的 `--message` 未包含 `rawRequest`：必须同时传入 `--to` 与 `--subject`。
- 显式传入 `--message` 时，它必须是合法 JSON；否则立即失败。
- `rawRequest` 字段存在时必须为非空 JSON 字符串。`null`、空字符串、纯空白、非字符串或非法 JSON 均立即失败，不会改为新邮件发送。
- 回复用的 `rawRequest` 必须包含非空的 `message.messageId`、`message.from` 和 `message.subject`；缺少任一字段立即失败。

完整说明见：[邮件插件主手册](../../USER_GUIDE.md)。
