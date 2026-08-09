# 20260610-1 迭代说明

本次迭代把 `integration` 收口的 `/v1/chat/completions` 当前 Agent 会话元数据补齐为：

- 保留完整的 `metadata.agents[]`
- 额外补出 `metadata.agent`
  - `metadata.agent.version`
    - 来自 `--agent-dir/<agentId>/config.json` 中的 `version`
    - 走共享 Agent metadata 缓存周期
  - `metadata.agent.sandbox`
    - 按当前 `metadata.agentId + metadata.chat` 实时读取共享 sqlite 的 `cli_sandbox_state`
    - 未写入时返回空字符串

## 覆盖链路

- 外部 `POST /v1/chat/completions`
- integration 内部 cron 执行器真正转发到上游 `/v1/chat/completions` 的请求

## 结果

转发到上游的 `metadata` 现在会包含：

```json
{
  "agent": {
    "agentId": "A",
    "version": "2026.06.10",
    "sandbox": "filepick_net"
  }
}
```
