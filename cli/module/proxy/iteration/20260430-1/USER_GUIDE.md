# SQLite 索引优化

## 新增索引

| 表 | 索引名 | 列 |
|---|--------|-----|
| task_meta | idx_meta_agent | agent_id |
| task_detail | idx_detail_agent_time | agent_id, exec_time |
| chat_log | idx_chat_agent_chat | agent_id, chat_id |
| chat_log | idx_chat_agent_chat_time | agent_id, chat_id, created_at |
| cmd_log | idx_cmd_agent_chat | agent_id, chat_id |
| cmd_log | idx_cmd_agent_chat_time | agent_id, chat_id, received_at |

## 说明

- 使用 `CREATE INDEX IF NOT EXISTS`，兼容已有数据库
- 覆盖备忘录（cron）和会话存储（chat_log）的按 Agent、Chat、时间查询场景
- cron、proxy、integration 三个模块同步更新
