# /api/kill 本地命令终止

## 功能说明

新增 `POST /api/kill`，用于终止指定 Agent、指定 ChatID 对应的活动系统命令。

## 请求约束

- 仅允许来自 `127.0.0.1`、`::1` 或 `localhost` 的请求执行
- 请求体必须包含 `agentId`
- `agentId` 对应的 Agent 必须仍然存在
- 请求体必须包含 `chatId`
- 请求体必须包含 `cmd`

请求体示例：

```json
{
  "agentId": "demo-agent",
  "chatId": "chat_001",
  "cmd": "sleep 10",
  "tid": "optional-task-id"
}
```

## 终止规则

- 仅终止当前进程内由 `/api/cmd` 启动且仍在运行的活动命令
- 优先按 `agentId + chatId + tid + cmd` 匹配
- 如果未提供 `tid`，则回退按 `agentId + chatId + cmd` 匹配
- 匹配不到活动命令时返回未找到
- 与 `cli-get` 一样，命令本身仍然是 Shell 子进程；`/api/kill` 仅负责终止该子进程

## 日志落库

终止请求会写入共享 `data` SQLite 的 `kill_log` 表：

```sql
CREATE TABLE kill_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  agent_id TEXT NOT NULL,
  chat_id TEXT NOT NULL,
  tid TEXT NOT NULL,
  cmd TEXT NOT NULL,
  received_at TEXT NOT NULL,
  completed_at TEXT NOT NULL DEFAULT ''
)
```

索引：

```sql
CREATE INDEX IF NOT EXISTS idx_kill_agent_chat_time
ON kill_log(agent_id, chat_id, received_at);
```

说明：

- `received_at`：收到 kill 请求的时间
- `completed_at`：kill 处理完成时间
- 即使未找到活动命令，也会记录 kill 请求日志
- `kill_log` 与 cron 共用 `data` SQLite 和连接池

## 返回结果

成功响应示例：

```json
{
  "status": 0,
  "agentId": "demo-agent",
  "chatId": "chat_001",
  "tid": "optional-task-id",
  "cmd": "sleep 10",
  "content": "killed",
  "receivedAt": "2026-05-01T16:00:00.000",
  "completedAt": "2026-05-01T16:00:00.020"
}
```

未命中活动命令示例：

```json
{
  "status": 1,
  "agentId": "demo-agent",
  "chatId": "chat_001",
  "tid": "optional-task-id",
  "cmd": "sleep 10",
  "content": "active command not found",
  "receivedAt": "2026-05-01T16:00:00.000",
  "completedAt": "2026-05-01T16:00:00.020"
}
```
