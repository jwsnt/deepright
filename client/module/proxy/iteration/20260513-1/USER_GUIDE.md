# Proxy 迭代手册（20260513-1）

## 本次更新

- 为 `/v1/chat/completions` 新增统一日志落库
- 请求日志写入类型 `0`
- SSE 响应分段日志写入类型 `1`
- 日志异步写入当前应用目录下的 SQLite `data`
- 新增统一日志表 `agent_message_log`
- 表索引为 `agent_id + chat_id + log_type + created_at`
- 保留原有 `chat_log`，避免破坏 `/api/restore` 既有恢复能力
- `/api/restore` 现在会额外合并返回同一 `agentId + chat` 下的 `cli/get`、`cli/pub` 日志

## 日志表结构

```sql
CREATE TABLE agent_message_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  agent_id TEXT NOT NULL DEFAULT '',
  chat_id TEXT NOT NULL DEFAULT '',
  content TEXT NOT NULL,
  log_type INTEGER NOT NULL,
  created_at TEXT NOT NULL
);
```

日志类型说明：

- `0`：`/v1/chat/completions` 请求
- `1`：`/v1/chat/completions` SSE 响应分段
- `2`：`cli/get`
- `3`：`cli/pub`

## 使用说明

- `proxy` 转发 `/v1/chat/completions` 时，会在识别到 `metadata.agentId` 与 `metadata.chat` 后记录请求日志
- 上游 SSE 每收到一段就异步写一条类型 `1` 日志，不会聚合后再落库
- 如果需要恢复会话，可继续使用：

```bash
curl -X POST 'http://127.0.0.1:9876/api/restore?agentId=A&chat=chat-001&timeline=2026-05-13T12:00:00'
```

- 返回结果中，如果命中了统一日志表里的心跳记录，会出现：
  - `role=cli/get`
  - `role=cli/pub`

## 兼容性说明

- 本次迭代不删除旧表，不迁移旧 `chat_log`
- 页面会话恢复、定时任务会话恢复仍保持原行为
- 新增日志写入采用异步方式，不阻塞 SSE 转发
