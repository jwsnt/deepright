# Integration 迭代手册（20260513-1）

## 本次收口

- 收口 Proxy 写入日志需求：
  - `/v1/chat/completions` 请求写类型 `0`
  - `/v1/chat/completions` SSE 响应分段写类型 `1`
- 收口 CLI-Get 写入日志需求：
  - `cli/get` 写类型 `2`
  - `cli/pub` 写类型 `3`
- 收口 Proxy 读取日志需求：
  - 新增 `GET /log_round?agentId=...&chatId=...&round=...`
  - 新增 `integration log-round --agent ... --chat ... --round ...`

## 统一日志表

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

索引：

- `agent_id + chat_id + log_type + created_at`

类型说明：

- `0`：会话请求
- `1`：会话响应
- `2`：工具请求（cli/get）
- `3`：工具响应（cli/pub）

## 行为说明

- `cli/get` 只有在服务端返回可执行任务时才会写统一日志
- 当 `cli/get` 响应中的 `content` 为 `null` 或空字符串时，不写日志
- `/api/restore` 现在会合并 `cli/get` 与 `cli/pub` 记录一起返回

## 最近轮次导出

- 轮次以 `/v1/chat/completions` 请求为边界
- 导出数据按时间排序
- 多段 SSE 在导出时会合并成一条记录
- 文件写入对应 Agent 工作目录下的 `tmp/`
- 文件格式为 Markdown 表格：

```md
| 时间 | 类型 | 内容 |
| --- | --- | --- |
```

类型显示：

- `请求`
- `响应`
- `工具请求（cli/get）`
- `工具响应（cli/pub）`
