# /api/cmd 本地命令执行

## 功能说明

新增 `POST /api/cmd`，用于执行指定 Agent、指定 ChatID 的系统命令。

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
  "cmd": "pwd && ls",
  "timeout": 60000,
  "tid": "optional-task-id"
}
```

## 执行方式

- 命令执行方式与 `cli-get` 保持一致
- 使用当前进程 Shell 执行 `shell -c <cmd>`
- 支持 `&&`、管道、绝对路径、相对路径和 `~`
- 超时优先使用请求中的 `timeout`（毫秒）
- 如果未提供 `timeout`，默认 `180000ms`

## 安全检查

- 执行前会进行恶意命令检查
- 只要命令中包含 `rm`，就直接拒绝执行
- 包括 `rm`、`&& rm` 等连续指令场景

## 返回结果

成功响应示例：

```json
{
  "status": 0,
  "agentId": "demo-agent",
  "chatId": "chat_001",
  "tid": "cmd_1746100000000000000",
  "cmd": "pwd && ls",
  "output": "/current/process/cwd\nREADME.md\n",
  "receivedAt": "2026-05-01T10:00:00.000",
  "completedAt": "2026-05-01T10:00:00.120"
}
```

- `status=0` 表示执行成功
- `status=1` 表示执行失败
- `output` 为命令原始输出

## 日志落库

执行过程会写入共享 `data` SQLite 的 `cmd_log` 表：

```sql
CREATE TABLE cmd_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  agent_id TEXT NOT NULL,
  chat_id TEXT NOT NULL,
  tid TEXT NOT NULL,
  cmd TEXT NOT NULL,
  result TEXT NOT NULL DEFAULT '',
  status INTEGER NOT NULL DEFAULT -1,
  received_at TEXT NOT NULL,
  completed_at TEXT NOT NULL DEFAULT ''
)
```

- 收到命令时写入 `cmd`、`received_at`
- 执行完成后更新 `result`、`status`、`completed_at`
- `result` 字段保存为 `GZIP+Base64`
