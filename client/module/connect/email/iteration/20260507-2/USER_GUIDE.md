# 20260507-2 User Guide

## 目标

本次迭代补充 `email send` 与 `connect add-request` 的对接说明，明确邮件插件回复链路消费的是 `add-request` 保存下来的请求 JSON，而不是插件私有结构。

## 命令格式

```bash
../plugins/email send \
  --message 原消息报文JSON \
  --content 消息文本内容 \
  --image /tmp/a.png,/tmp/b.jpg \
  --file /tmp/a.pdf,/tmp/b.txt
```

说明：

- `--message` 为必填，值应为 `connect add-request` 返回的请求 JSON
- 推荐优先使用其中的 `original` 字段；插件同时兼容旧字段 `rawRequest`
- `--content`、`--image`、`--file` 至少需要提供一种

## 配置读取

`send` 与 `start` 使用同一套配置读取方式：

- 优先调用 `connect meta-get --key email`
- 若当前环境仍保留旧 key `email_smtp`，插件会自动回退兼容

读取的关键字段包括：

- `email`
- `email_smtp`
- `email_password`

## 回复行为

- 文本消息会同时生成纯文本正文和 HTML 正文
- 图片、文件会作为普通邮件附件发送
- 如果 `--message` 中能解析出原始邮件头，则会回复原发件人
- `In-Reply-To` 使用原始报文头中的 `Message-ID`
- `References` 保留原始链路，并追加父消息 `Message-ID`

## 验证示例

先创建符合插件固定参数输出的 meta：

```bash
./integration connect meta-create \
  --name email \
  --meta '{"email":"'$EMAIL'","email_pop3":"'$EMAIL_POP3'","email_smtp":"'$EMAIL_SMTP'","email_password":"'$EMAIL_PASSWORD'","email_whitelist":""}'
```

然后执行发送：

```bash
../plugins/email send \
  --message '{"id":1,"key":"email","original":"{\"source\":\"email\",\"message\":{\"raw\":\"{\\\"headers\\\":[{\\\"name\\\":\\\"From\\\",\\\"value\\\":\\\"Sender <sender@example.com>\\\"},{\\\"name\\\":\\\"Subject\\\",\\\"value\\\":\\\"原始主题\\\"},{\\\"name\\\":\\\"Message-ID\\\",\\\"value\\\":\\\"<origin@example.com>\\\"}],\\\"content\\\":\\\"hello\\\"}\"}}"}' \
  --content "收到，附件如下" \
  --image /tmp/a.png \
  --file /tmp/a.pdf
```

## 日志

每次执行 `send` 时，都会在 `email.log` 追加一行发送记录，例如：

```text
2026-05-07T10:00:00+08:00,send name=email to=sender@example.com replyTo=<origin@example.com> types=text+image+file
```
