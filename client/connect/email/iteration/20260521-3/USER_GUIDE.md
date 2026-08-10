# Email Iteration 20260521-3 User Guide

## 需求目录

- 迭代需求文档：[REQUIREMENT.md](REQUIREMENT.md)
- 邮件插件主需求：[../REQUIREMENT.md](../../REQUIREMENT.md)
- 邮件插件主手册：[../USER_GUIDE.md](../../USER_GUIDE.md)

## 本次迭代内容

本次迭代补齐 RFC 2047 / MIME encoded-word 主题的对外展示一致性。

- 任务明细正文使用解码后的主题
- `raw_request.message.subject` 使用解码后的主题
- `raw_request.message.content` 使用解码后的主题和正文
- `push-request` 等排障日志中的 `--content`、`subject`、`content` 展示值使用解码后的文本

## 当前行为

如果邮件主题原始头为：

```text
=?GBK?B?KM7e1vfM4ik=?=
```

则插件对外可见字段会统一展示为：

```text
(无主题)
```

也就是说，`connect_request.request`、`connect_request.raw_request.message.subject`、`connect_request.raw_request.message.content`、以及 `stage=push-request` 这类错误日志中的展示文本，都会优先显示解码后的可读内容，而不是直接显示 `=?...?=` 编码串。

历史上已经落库的旧记录不会因为这次迭代自动回写；本次保证的是修复后的新入库数据和新产生的排障日志都使用解码后的可读文本。

## 说明

- 本次迭代不改变 POP3 收件协议、SMTP 发送协议或邮件去重/时间线语义
- 本次迭代只修复“用户可见字段”的乱码问题，保证任务明细、原始请求与排障日志三处输出一致
