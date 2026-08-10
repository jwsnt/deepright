# 20260502-6 使用手册

## 完成状态

本次迭代需求已完成。

已落地内容：

- `task_meta` 与 `task_detail` 增加 `type` 字段，默认值为 `cron`
- 子模块调用、CLI 创建、HTTP 创建都会透传并保存任务类型
- `task_detail` 查询索引统一调整为 `agent_id + chat_id + exec_time + task_type`
- 历史库中旧索引存在时会在启动时自动删除并重建
- integration 侧已同步支持创建、查询、删除时按 `type` 传参与过滤

## 类型说明

- `cron`
  - 表示备忘录 cron 创建的周期任务或一次性任务
- `connect`
  - 表示由 connect 模块创建的任务
  - 实际落库值可写为具体模块名，例如 `FEISHU`

## Cron 模块使用

CLI 创建时可直接指定 `type`：

```bash
./cron create --agent-dir ../agent/test-case --cycle 0 --time "2026-05-02 09:30" --agent A --chat chat-001 --model OpenAI --content "提醒检查日报" --type cron
```

Connect 场景可传具体模块名：

```bash
./cron create-cron --agent-dir ../agent/test-case --cron "10 12 * * 1-5" --agent A --chat chat-001 --model OpenAI --content "同步飞书审批" --type FEISHU
```

行为说明：

- 未传 `--type` 时默认写入 `cron`
- 后续自动生成的 `task_detail.type` 会继承 `task_meta.type`
- 查询返回的元数据与明细都会包含 `type`

## Integration 对齐

HTTP 创建示例：

```json
{
  "content": "同步飞书审批",
  "model": "OpenAI",
  "thinking": true,
  "cycle": -1,
  "cron": "10 12 * * 1-5",
  "chatId": "chat-001",
  "type": "FEISHU"
}
```

查询过滤说明：

- `/api/cron/detail/metadata` 支持 `type`
- `/api/cron/detail/list` 支持 `type`
- `/api/cron/delete` 与 `/api/cron/detail/delete` 也支持复用同样的 `type` 过滤条件
- `integration cron find-meta`、`integration cron find-detail`、`integration cron delete-meta`、`integration cron delete-detail` 也都支持 `--type`

## 数据库变更

任务主表：

- `task_meta.task_type`
- `task_detail.task_type`

审计日志表：

- `cron_meta_log.task_type`
- `cron_detail_log.task_type`

索引调整：

```sql
DROP INDEX IF EXISTS idx_detail_agent_time;
DROP INDEX IF EXISTS idx_detail_agent_chat_time;
DROP INDEX IF EXISTS idx_detail_agent_chat_time_type;
CREATE INDEX IF NOT EXISTS idx_detail_agent_chat_time_type
ON task_detail(agent_id, chat_id, exec_time, task_type);
```

## 验证点

- 新建任务但不传 `type` 时，元数据和明细中的值应为 `cron`
- 传入 `type=FEISHU` 时，元数据和明细中的值都应为 `FEISHU`
- 历史库升级后，`task_detail` 存在 `task_type` 列
- 历史库升级后，旧索引被移除，新索引 `idx_detail_agent_chat_time_type` 存在
