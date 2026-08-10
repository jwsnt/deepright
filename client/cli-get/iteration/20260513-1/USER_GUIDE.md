# CLI-Get 迭代手册（20260513-1）

## 本次更新

- 为 `cli/get` 与 `cli/pub` 新增统一日志落库
- `cli/get` 写入类型 `2`
- `cli/pub` 写入类型 `3`
- 与 `proxy` 共用同一张日志表 `agent_message_log`
- 使用当前工作目录下的 SQLite `data` 文件
- 日志写入改为异步，不阻塞心跳轮询和结果回传
- 执行命令前注册到进程内活跃命令列表，执行结束后自动注销
- 命令执行上下文被取消时，回传结果中的解压内容固定为 `命令被终止`

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

- `2`：`cli/get`
- `3`：`cli/pub`

## 行为说明

- `cli/get` 会在服务端返回任务后，使用该任务里的 `agentId` 与 `chat` 记录本次心跳日志
- 如果 `cli/get` 响应中的 `content` 为 `null` 或空字符串，表示当前没有待执行任务，本次不记录日志
- `cli/pub` 会使用结果中的 `agentId` 与 `chat` 记录本次回传日志
- 如果当前心跳没有返回任务，则本次不会生成带 `chat_id` 的心跳日志

## 命令终止说明

- Worker 执行命令时会先注册活跃命令
- 如果命令上下文被取消：
  - 返回 `status=1`
  - `cmd` 字段解压后的内容为 `命令被终止`

## 适用场景

- 供 `integration` 后续统一收口时直接复用
- 供 `proxy` 的 `/api/restore` 合并查询 `cli/get` / `cli/pub` 日志
