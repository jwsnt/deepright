# Email Iteration 20260521-1 User Guide

## 需求目录

- 迭代需求文档：[REQUIREMENT.md](REQUIREMENT.md)
- 邮件插件主需求：[../REQUIREMENT.md](../../REQUIREMENT.md)
- 邮件插件主手册：[../USER_GUIDE.md](../../USER_GUIDE.md)

## 本次迭代内容

本次迭代只补 `email.log` 的收件日志表现：

- 日志里的主题统一输出为已解码后的可读文本
- 日志里增加结构化字段，至少包含 `subject`、`from`、`message_id`

## 日志示例

收到并成功推送的邮件：

```text
2026-05-21T10:00:00+08:00,stage=mail-received subject="今天天气" from="Sender <sender@example.com>" message_id="<origin@example.com>" summary="今天天气\n正文内容"
```

被白名单拒绝的邮件：

```text
2026-05-21T10:00:05+08:00,stage=mail-skip subject="今天天气" from="deny@example.com" message_id="<origin@example.com>" summary="今天天气\n正文内容" reason="sender not in whitelist"
```

命中重复 `Message-ID` 的邮件：

```text
2026-05-21T10:00:10+08:00,stage=mail-skip subject="今天天气" from="sender@example.com" message_id="<origin@example.com>" summary="今天天气\n正文内容" reason=duplicate-message-id
```

## 说明

- `subject` 永远优先输出解码后的主题
- `from` 会优先输出解码后的发件人显示名和地址
- `message_id` 会保留 RFC 5322 的尖括号格式
- `summary` 用于补充快速排查信息，通常是“主题 + 正文摘要”
