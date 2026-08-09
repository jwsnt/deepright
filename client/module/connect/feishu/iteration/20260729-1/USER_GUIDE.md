# 飞书迭代手册（20260729-1）

## 需求目录

- 本迭代需求：[REQUIREMENT.md](REQUIREMENT.md)
- 飞书插件主需求：[../../REQUIREMENT.md](../../REQUIREMENT.md)
- 飞书插件主手册：[../../USER_GUIDE.md](../../USER_GUIDE.md)
- 前端需求：[../../../site/iteration/20260729-1/REQUIREMENT.md](../../../site/iteration/20260729-1/REQUIREMENT.md)

## 合法配置

飞书插件仍通过 `key=feishu` 读取配置，字段 key 保持为 `appId`、`appSecret`、`mcp_url`；展示名 `name` 不参与运行态查找。

可以使用以下任意一种配置：

```json
{"appId":"cli_xxx","appSecret":"secret"}
```

```json
{"mcp_url":"https://your-feishu-mcp-server.example.com"}
```

也可以同时填写三项。`appId` 与 `appSecret` 必须成对出现；只填写其中一个、或三项都为空时，启动和重启都会失败。

## 启动行为与日志

完整凭证组会先通过原有凭证校验，再建立飞书长连接。仅填写 `mcp_url` 时，插件仍显示为已启动并可由 `stop` 正常关闭，但不会进行凭证校验、创建飞书会话、建立或重试长连接，也不会收发飞书消息。

`feishu.runtime.log` 会记录凭证组长连接、仅 MCP 地址启动和配置校验失败三类状态。仅 MCP 地址启动会带有 `mode=mcp-only` 与 `long_connection=false`。日志不会写入 `appSecret` 明文或完整敏感配置。
