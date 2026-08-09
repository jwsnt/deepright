# CLI 命令执行日志

## 功能说明

`cli/get` 收到的命令及执行结果以 AgentId+Chat 维度保存到共享的 `data` SQLite 数据库的 `cmd_log` 表中。

## 存储表结构

```sql
CREATE TABLE cmd_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  agent_id TEXT NOT NULL,
  chat_id TEXT NOT NULL,
  tid TEXT NOT NULL,
  cmd TEXT NOT NULL,           -- 原始命令
  result TEXT NOT NULL DEFAULT '',  -- 执行结果（GZIP+Base64）
  status INTEGER NOT NULL DEFAULT -1,  -- -1=执行中, 0=成功, 1=失败
  received_at TEXT NOT NULL,   -- 收到命令时间
  completed_at TEXT NOT NULL DEFAULT ''  -- 执行完成时间
)
```

## 写入时机

1. 收到 `cli/get` 响应中的命令时：写入 `cmd`、`received_at`，`status=-1`
2. 命令执行完成后：更新 `result`、`status`、`completed_at`

## 说明

- 使用全局 `cronDB` 连接池，不单独打开数据库
- 仅当 `AgentId`、`Chat`、`Cmd` 均非空时记录
- `result` 为 GZIP+Base64 编码的命令输出
