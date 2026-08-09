# 20260530-1 使用说明

## 变更说明

- 右侧 `知识库WIKI` 标题栏中，在 `整理知识库WIKI` 按钮左侧新增了一个全局 `自动` 开关。
- 这个开关不绑定 Agent，也不绑定当前会话；页面内任意会话发送 `/v1/chat/completions` 时，都会统一带上 `metadata.knowledge_disable`。
- 开关默认处于开启态，此时请求中会发送 `metadata.knowledge_disable = false`；关闭后会改为发送 `metadata.knowledge_disable = true`。
- 原有的 `metadata.agentId`、`metadata.chat`、`metadata.thinking`、`metadata.html`、`metadata.router_disable`、`metadata.knowledge_commit` 等字段保持原样，不会因为这个开关被删除或改名。

## 请求示例

```json
{
  "metadata": {
    "agentId": "当前Agent",
    "chat": "当前会话ID",
    "knowledge_disable": false
  }
}
```

关闭 `自动` 开关后，同一位置会改为发送 `metadata.knowledge_disable = true`。

## 验证方式

1. 打开页面，查看右侧 `知识库WIKI` 标题栏，确认 `整理知识库WIKI` 按钮左侧存在 `自动` 开关。
2. 保持开关为默认开启状态，发送一条普通对话消息。
3. 在浏览器开发者工具的 Network 中检查对应 `/v1/chat/completions` 请求体，确认存在 `metadata.knowledge_disable = false`。
4. 关闭该开关后再次发送消息，确认同一字段变为 `true`，且其它 metadata 字段仍然存在。
5. 切换 Agent、新建会话或切换会话后再次发送消息，确认该开关状态仍按页面全局生效，而不是随会话变化。
