# 20260610-1 迭代说明

本次迭代补齐了 `proxy` 转发 `/v1/chat/completions` 时的当前 Agent 会话元数据：

- 会继续保留完整的 `metadata.agents[]`
- 另外新增 `metadata.agent`
  - `metadata.agent.version`
    - 来自 `--agent-dir/<agentId>/config.json` 中的 `version`
    - 走共享 Agent metadata 缓存周期
  - `metadata.agent.sandbox`
    - 按当前 `metadata.agentId + metadata.chat` 实时读取共享 sqlite 的 `cli_sandbox_state`
    - 未写入时返回空字符串

## 覆盖链路

- 页面或外部客户端发起的 `POST /v1/chat/completions`
- `proxy` 内部 cron 真正执行 `task_detail` 时发起的上游 `/v1/chat/completions`

## 结果

转发到上游的 `metadata` 现在同时具备：

```json
{
  "agent": {
    "agentId": "A",
    "version": "2026.06.10",
    "sandbox": "filepick_net"
  }
}
```
