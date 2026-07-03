# 20260505-1 User Guide

## 目标

本次迭代为 `feishu` 插件增加主动发送能力，支持通过独立 CLI 命令回复飞书已有消息，并附带文本、图片、文件内容，同时保持现有 `connect` / `integration` 的二进制与 CLI 收口方式不变。

本轮完成点包括：

- 新增 `./feishu send` 命令
- `send` 与 `start` 共用同一套 `meta` 配置获取方式
- `param` 固定返回 `["appId","appSecret"]`
- `name` 固定返回 `{"key":"feishu","name":"飞书"}`
- 每次执行 `send` 时，都会向 `feishu.log` 追加一条调用记录

## 编译

在 `/path/to/deepright/cli/module/connect` 目录编译插件：

```bash
go build -o ../plugins/feishu ./feishu
```

## 固定输出命令

以下两个命令不依赖长连接服务是否启动，始终可以直接调用：

```bash
../plugins/feishu param
../plugins/feishu name
```

返回结果固定为：

```json
["appId","appSecret"]
```

```json
{"key":"feishu","name":"飞书"}
```

## 配置方式

`send` 与 `start` 一样，都会通过 `connect` / `integration` 中名为 `feishu` 的连接配置读取参数。

推荐按顶层 CLI 方式写入：

```bash
./integration connect meta-create --name feishu --meta '{"appId":"x","appSecret":"y"}' --callback ./feishu --agent a --model deepseek
```

## 主动发送

命令格式：

```bash
../plugins/feishu send --message 原消息报文JSON --content 消息文本内容 --image /tmp/a.png,/tmp/b.jpg --file /tmp/a.pdf,/tmp/b.txt
```

使用示例：

```bash
../plugins/feishu send --message '{"id":1,"rawRequest":"{\"schema\":\"2.0\",\"event\":{\"message\":{\"message_id\":\"om_xxx\",\"content\":\"{\\\"text\\\":\\\"你好\\\"}\",\"message_type\":\"text\"}}}"}' --content "收到"
../plugins/feishu send --message '{"message":{"raw":"{\"schema\":\"2.0\",\"event\":{\"message\":{\"message_id\":\"om_xxx\",\"content\":\"{\\\"text\\\":\\\"你好\\\"}\",\"message_type\":\"text\"}}}"}}' --image /tmp/a.png,/tmp/b.jpg
../plugins/feishu send --message '{"messageId":"om_xxx"}' --file /tmp/a.pdf,/tmp/b.txt
../plugins/feishu send --message '{"id":1,"rawRequest":"{\"schema\":\"2.0\",\"event\":{\"message\":{\"message_id\":\"om_xxx\",\"content\":\"{\\\"text\\\":\\\"你好\\\"}\",\"message_type\":\"text\"}}}"}' --content "附件如下" --image /tmp/a.png --file /tmp/a.pdf
```

说明：

- `--message` 为必填
- `--message` 应传入 `connect add-request` 的原始报文 JSON
- 飞书原消息 `messageId` 会从 `--message` 中自动提取
- `--image`、`--file` 可以为空
- `--content` 会以飞书 `interactive` 卡片消息发送，正文使用 Markdown 渲染
- `--image` 会先调用飞书图片上传接口拿到 `image_key`，再发送图片消息
- `--file` 会先调用飞书文件上传接口拿到 `file_key`，再发送文件消息
- `send` 只用于回复已有消息，不再支持“省略原消息后直接新发一条消息”
- 如果同时带文本和附件，插件会拆成多次飞书 API 调用，发送顺序为图片 -> 文件 -> 文本
- 文本、图片、文件至少要提供一种，否则命令会报错

## 日志

`send` 执行时会同时写入：

- `feishu.log`：消息流水日志
- `feishu.runtime.log`：运行诊断日志

其中 `feishu.log` 会追加一条类似记录：

```text
2026-05-05T16:30:00+08:00,send name=feishu target=oc_xxx types=text+image+file
```

## 验证

本轮已覆盖的关键验证点：

- `go test ./feishusvc/...` 通过
- 编译后的 `feishu` 二进制可正确执行 `param`
- 编译后的 `feishu` 二进制可正确执行 `name`
- `send` 文本消息已改为飞书富文本发送
- `send` 图片消息已改为先上传再发送
- `send` 文件消息已改为先上传再发送
- `feishusvc` 测试已覆盖 `send` 调用与 `feishu.log` 写入行为
