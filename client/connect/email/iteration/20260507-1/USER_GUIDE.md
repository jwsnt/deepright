# 20260507-1 User Guide

## 目标

本轮迭代为 `email` 插件新增 `init` 命令，用于向邮件发送任务初始化消息，并要求参数与处理流程完全复用 `send`，保持现有 CLI 收口和插件设计不变。

## 本轮完成点

- 新增 `./email init` 命令
- `init` 与 `send` 共用同一套参数解析、配置加载和发送实现
- `init` 支持正文、图片附件、文件附件组合发送
- `init` 执行时与 `send` 一样写入 `email.log`
- 同步更新主手册与 `email` 模块手册

## 命令格式

```bash
../plugins/email init --message 原消息报文JSON --content 消息文本内容 --image /tmp/a.png,/tmp/b.jpg --file /tmp/a.pdf,/tmp/b.txt
```

它与 `send` 的区别只在命令名，参数和处理流程完全一致。

## 示例

```bash
../plugins/email init --message '{"message":{"raw":"{\"headers\":[{\"name\":\"From\",\"value\":\"Sender <sender@example.com>\"},{\"name\":\"Subject\",\"value\":\"原始主题\"},{\"name\":\"Message-ID\",\"value\":\"<origin@example.com>\"}],\"content\":\"hello\"}"}}' --content "初始化完成"

../plugins/email init --message '{"message":{"raw":"{\"headers\":[{\"name\":\"From\",\"value\":\"Sender <sender@example.com>\"},{\"name\":\"Subject\",\"value\":\"原始主题\"},{\"name\":\"Message-ID\",\"value\":\"<origin@example.com>\"}],\"content\":\"hello\"}"}}' --content "开始处理" --image /tmp/a.png --file /tmp/a.pdf
```

## 行为说明

- `--message` 必填，值为 `connect add-request` 保存的原始报文 JSON
- `--content`、`--image`、`--file` 至少要提供一种
- `init` 内部直接复用 `send` 的发送逻辑
- 文本会同时生成邮件纯文本和 HTML 正文
- 图片和文件仍作为普通邮件附件发送
- 如果原始报文中能解析到邮件头，会自动回复原发件人并补齐 `In-Reply-To`、`References`
- 日志行为与 `send` 保持一致

## 编译

在 `/path/to/deepright/cli/module/connect` 目录执行：

```bash
go build -o ../plugins/email ./email
```

## 验证

- `go test ./...` in `module/connect/emailsvc`
- `./email init` 与 `./email send` 共享同一套发送实现
