# 20260507-1 User Guide

## 目标

本轮迭代为 `feishu` 插件新增 `init` 命令，用于向飞书发送任务初始化消息，并要求参数与处理流程完全复用 `send`，保持现有 CLI 收口和插件设计不变。

## 本轮完成点

- 新增 `./feishu init` 命令
- `init` 与 `send` 共用同一套参数解析、配置加载和发送实现
- `init` 支持文本、图片、文件三类内容组合发送
- `init` 执行时与 `send` 一样写入 `feishu.log`
- 同步更新主手册并复制编译产物到 `module/plugins/feishu`

## 命令格式

```bash
../plugins/feishu init --message 原消息报文JSON --content 消息文本内容 --image /tmp/a.png,/tmp/b.jpg --file /tmp/a.pdf,/tmp/b.txt
```

它与 `send` 的区别只在命令名，参数和处理流程完全一致。

## 示例

```bash
../plugins/feishu init --message '{"id":1,"rawRequest":"{\"schema\":\"2.0\",\"event\":{\"message\":{\"message_id\":\"om_xxx\",\"content\":\"{\\\"text\\\":\\\"任务\\\"}\",\"message_type\":\"text\"}}}"}' --content "<开始执行>可通过新消息更新任务"

../plugins/feishu init --message '{"messageId":"om_xxx"}' --content "开始处理" --image /tmp/a.png --file /tmp/a.pdf
```

## 行为说明

- `--message` 必填，值为 `connect add-request` 保存的原始报文 JSON
- `--content`、`--image`、`--file` 至少要提供一种
- `init` 内部直接复用 `send` 的发送逻辑
- 文本仍以飞书 `interactive` 卡片发送
- 图片和文件仍然是先上传资源，再发送消息
- 如果同时带文本和附件，发送顺序仍为图片 -> 文件 -> 文本
- 日志行为与 `send` 保持一致

## 编译

在 `/path/to/deepright/cli/module/connect` 目录执行：

```bash
go build -o ../plugins/feishu ./feishu
```

## 验证

- `go test ./...` in `module/connect/feishusvc`
- `./feishu init` 与 `./feishu send` 共享同一套发送实现
