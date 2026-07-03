# 飞书迭代手册（20260612-2）

## 本次变更

本次迭代补齐了飞书插件对“说明文字 + JSON Object”混合响应的兼容处理：

- `send` / `init` 的 `--content` 如果本身不是纯 JSON，但正文里包含一段合法 JSON Object，插件会先提取这段 JSON
- 提取出的 JSON 会继续按 `feishu schema` 做归一化
- `content` 会作为真正发送的飞书正文
- `artifacts[].path` 会继续拆分为图片附件和文件附件
- 如果整段文本里仍然提取不到合法且符合 schema 的 JSON，则维持原始发送逻辑，不做拆解

## 兼容示例

```bash
../plugins/feishu send \
  --message '{"id":1,"original":"{\"schema\":\"2.0\",\"event\":{\"message\":{\"message_id\":\"om_xxx\",\"content\":\"{\\\"text\\\":\\\"你好\\\"}\",\"message_type\":\"text\"}}}"}' \
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
- 提取失败不会破坏原有的日志和发送链路

## 追加说明

- `feishu send` / `feishu init` 中，文本、图片、文件三类实际发送步骤都最多自动重试 3 次
- 飞书单独推送图片或文件附件时，也遵守同样的 3 次重试上限
- 超过 3 次会直接终止本次发送，不再继续静默降级为仅发文本
- `feishu.log` 中会新增 `stage=send-retry` 和 `stage=send-terminate` 日志
- 长驻 `feishu start` 进程在 `load-config`、`create-session`、`run-session`、`flush-pending` 等链路连续失败 3 次后，也会写 `stage=service-retry` / `stage=service-terminate` 并自动退出
