# 20260502-5 User Guide

## 目标

本次迭代为 cron 模块的任务元数据与任务明细数据库操作补充了统一审计日志。

## 新增日志表

- `cron_meta_log`
- `cron_detail_log`

## 行为说明

- `task_meta` 的数据库操作会记录到 `cron_meta_log`
- `task_detail` 的数据库操作会记录到 `cron_detail_log`
- 当前日志覆盖创建、查询和任务执行过程中的状态更新
- 两张日志表都按 `Agent + Chat + 时间` 建立索引
- 如果数据库中存在历史相关日志表，会自动改名到新的表名

## 索引

```sql
CREATE INDEX IF NOT EXISTS idx_cron_meta_log_agent_chat_time
ON cron_meta_log(agent_id, chat_id, created_at);

CREATE INDEX IF NOT EXISTS idx_cron_detail_log_agent_chat_time
ON cron_detail_log(agent_id, chat_id, occurred_at);
```
