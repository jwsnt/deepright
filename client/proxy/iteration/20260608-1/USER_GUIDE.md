# 迭代说明

本次迭代为 `proxy` 增加了按 `AgentId + ChatId` 维度持久化的沙盒开关，与 `integration` 保持相同的会话级行为。

## 新增能力

- 新增会话级沙盒状态存储，键为 `agentId + chatId`
- 沙盒状态写入当前应用目录共享 sqlite 的 `cli_sandbox_state` 表
- 新增 HTTP 接口：
  - `GET /api/sandbox?agentId=A&chatId=chat-001`
  - `GET /api/sandbox_status?agentId=A&chatId=chat-001`
  - `GET /api/sandbox=true?agentId=A&chatId=chat-001`
  - `GET /api/sandbox=false?agentId=A&chatId=chat-001`
  - 也兼容 `GET/POST /api/sandbox?sandbox=true|false&agentId=...&chatId=...`
- 新增 CLI：
  - `./proxy sandbox --agentId A --chatId chat-001`
  - `./proxy sandbox --agentId A --chatId chat-001 --sandbox true`

## 返回说明

- 未写入过该会话时，返回 `sandbox=false`
- 未写入过该会话时，返回 `recorded=false`
- 写入后会返回 `updatedAt`
- `/api/sandbox_status` 只读，不会因为携带 `sandbox` 参数而改写状态
- `chat` 仍兼容作为 `chatId` 别名输入

## 测试

- 补充了 `/api/sandbox` 按 `agentId + chatId` 读写测试
- 补充了 `/api/sandbox_status` 只读查询测试
- 补充了 `proxy sandbox` CLI 读写共享 sqlite 的测试
- 覆盖了不同 `chatId` 间状态隔离的场景
