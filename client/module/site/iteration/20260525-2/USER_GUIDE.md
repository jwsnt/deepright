# 20260525-2 使用说明

## 变更说明

- 居中对话输入框在 `HTML` 开关左侧新增 `SWARM` 开关。
- `SWARM` 按 `Agent + Chat` 维度独立保存，切换会话后会自动恢复该会话自己的蜂群开关状态。
- 当前会话发送 `/v1/chat/completions` 请求时，会显式携带 `metadata.router_disable = true/false`。

## 请求示例

```json
{
  "metadata": {
    "agentId": "当前Agent",
    "chat": "当前会话ID",
    "router_disable": false
  }
}
```

关闭 `SWARM` 后，同一位置会改为发送 `metadata.router_disable = true`。

## 验证方式

1. 在任意会话中开启 `SWARM`，发送一条消息。
2. 切换到其他会话后确认 `SWARM` 状态随会话切换。
3. 在浏览器 Network 中检查 `/v1/chat/completions` 请求体，确认 `metadata.router_disable` 会随当前会话开关状态在 `false` 和 `true` 间切换。
# 2026-05-24 修订

- 当前会话请求元数据统一发送 `metadata.router_disable`。
- SWARM 开启时发送 `router_disable=false`，关闭时发送 `router_disable=true`。
- 旧文档中的 `metadata.swarm` 仅代表历史写法，不再是当前规范字段。
