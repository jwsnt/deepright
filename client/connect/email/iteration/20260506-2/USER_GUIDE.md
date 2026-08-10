# 20260506-2 User Guide

## 目标

本次迭代为 `email` 插件新增 `send` 命令，用于基于原始 `add-request` 报文回复邮件正文，并按需附带图片、文件附件。

## 命令格式

```bash
../plugins/email send \
  --message 原消息报文JSON \
  --content 消息文本内容 \
  --image /tmp/a.png,/tmp/b.jpg \
  --file /tmp/a.pdf,/tmp/b.txt
```

说明：

- `--message` 为必填，值应为 `connect add-request` 保存下来的原始报文 JSON
- `--content`、`--image`、`--file` 至少需要提供一种
- `--image`、`--file` 都是逗号分隔路径，可以为空

## 参数来源

`send` 与 `start` 使用相同的参数获取方式，会先通过 `connect meta-get --name email` 读取邮件配置：

```json
{
  "email": "demo@example.com",
  "email_pop3": "pop.example.com:995",
  "email_smtp": "smtp.example.com:465",
  "email_password": "secret",
  "email_whitelist": ""
}
```

其中：

- `email`：发件人地址
- `email_smtp`：SMTP 服务器地址
- `email_password`：SMTP 登录密码或授权码

## 回复行为

- 文本消息会同时生成纯文本正文和 HTML 正文
- `--image`、`--file` 会作为普通邮件附件发送
- 如果 `--message` 能解析出原始邮件头，则会回复原发件人
- `In-Reply-To` 使用原始报文头中的 `Message-ID`
- `References` 会保留原始 `References` / `In-Reply-To`，并追加父消息的 `Message-ID`
- `Subject` 会自动补齐为 `Re: 原主题`

## 日志

每次执行 `send` 时，都会在 `email.log` 追加一行发送记录，例如：

```text
2026-05-06T10:00:00+08:00,send name=email to=sender@example.com replyTo=<origin@example.com> types=text+image+file
```

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
  --message '{"message":{"raw":"{\"headers\":[{\"name\":\"From\",\"value\":\"Sender <sender@example.com>\"},{\"name\":\"Subject\",\"value\":\"原始主题\"},{\"name\":\"Message-ID\",\"value\":\"<origin@example.com>\"}],\"content\":\"hello\"}"}}' \
  --content "收到，附件如下" \
  --image /tmp/a.png \
  --file /tmp/a.pdf
```
