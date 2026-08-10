# Email 迭代手册（20260612-3）

## 本次变更

本次迭代补齐了邮件插件对“说明文字 + JSON Object”混合响应的兼容处理：

- `send` / `init` 的 `--content` 如果本身不是纯 JSON，但正文里包含一段合法 JSON Object，插件会先提取这段 JSON
- 提取出的 JSON 会继续按 `email schema` 做归一化
- `content` 会作为真正发送的邮件正文
- `artifacts[].path` 会继续拆分为图片附件和文件附件
- 如果整段文本里仍然提取不到合法且符合 schema 的 JSON，则维持原始发送逻辑，不做拆解

## 兼容示例

```bash
../plugins/email send \
  --message '{"id":1,"rawRequest":"{\"source\":\"email\",\"receivedAt\":\"2026-05-22T00:00:00+08:00\",\"message\":{\"uid\":\"uid-1\",\"messageId\":\"<origin@example.com>\",\"subject\":\"原始主题\",\"from\":\"Sender <sender@example.com>\",\"content\":\"hello\",\"artifacts\":[],\"raw\":\"{\\\"headers\\\":[{\\\"name\\\":\\\"From\\\",\\\"value\\\":\\\"Sender <sender@example.com>\\\"},{\\\"name\\\":\\\"Subject\\\",\\\"value\\\":\\\"原始主题\\\"},{\\\"name\\\":\\\"Message-ID\\\",\\\"value\\\":\\\"<origin@example.com>\\\"}],\\\"content\\\":\\\"hello\\\"}\"}}"}' \
  --content '先看看桌面文件情况，同时尝试截图。截图已生成。现在截图下载目录：{"content":"请查收附件","artifacts":[{"path":"/tmp/demo.png","desc":"截图"},{"path":"/tmp/report.pdf","desc":"报告"}],"why_do_this":"补充执行结果"}'
```

上面这段命令最终会被归一化成：

- 正文：`请查收附件`
- 图片附件：`/tmp/demo.png`
- 文件附件：`/tmp/report.pdf`

## 验证点

- 纯 JSON Object 输入保持原有行为
- 前缀文本加 JSON Object 的输入现在会自动提取并归一化
- 没有可提取 JSON 的普通文本仍按原逻辑发送
- 如果后续发送链路再发生附件处理失败，插件也能继续从这类混合文本里回退出结构化 `content`

## 追加说明

- `email send` / `email init` 真正进入 SMTP 发送后，失败最多自动重试 3 次
- 超过 3 次会直接终止本次发送，不再继续切换其他发送兜底
- `email.log` 中会新增 `stage=send-retry` 和 `stage=send-terminate` 日志
- 长驻 `email start` 进程在 `load-config`、`scan`、`push-request` 等链路连续失败 3 次后，也会写 `stage=service-retry` / `stage=service-terminate` 并自动退出
