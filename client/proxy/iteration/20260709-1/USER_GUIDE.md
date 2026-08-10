# Proxy 迭代手册（20260709-1）

## 本次更新

- 会话沙盒状态从 `agentId + chatId` 改为仅按 `chatId` 保存与命中
- `/api/sandbox_status` 改为只依赖 `chatId`；即使请求里携带 `agentId`，也不会参与状态定位
- `/api/sandbox=*` 写接口仍要求 `agentId` 与 `chatId`；其中 `agentId` 只用于日志，`chatId` 用于写入当前会话沙盒状态
- `metadata.agent.sandbox` 与 `metadata.agents[].sandbox` 都改为按当前 `chatId` 实时读取共享 sqlite
- `/api/cmd` 的沙盒命中改为只看 `chatId`
- 跨系统 helper 选择保持不变：macOS 继续走 `CLI_SANDBOX.app`，WSL/Linux 继续走 `helpers/<mode>/CLI_SANDBOX`

## 接口示例

```text
GET /api/sandbox_status?chatId=chat-001
POST /api/sandbox=filepick?agentId=A&chatId=chat-001
POST /api/sandbox=filepick_net?agentId=A&chatId=chat-001&dir=%2FUsers%2Fme%2FDesktop
POST /api/sandbox=off?agentId=A&chatId=chat-001
```

## 行为说明

- `chatId` 为空时，读写都会直接报错
- 不同 `agentId` 访问同一 `chatId` 时，会命中同一份沙盒状态
- `off` 会删除当前 `chatId` 的记录；无记录视为 `off`
- `filepick` / `filepick_net` 如显式传入 `dir`，会把 `allowed_dir` 按当前 `chatId` 持久化；未传时继续按当前系统走目录选择流程
- 写入日志会输出 `agentId`、`chatId` 以及 `from -> to` 的文本变更信息
