# Integration 迭代手册（20260711-1）

## 本次更新

- 共享 sqlite 中的 `agent_message_log` 与 `chat_log` 新增了统一的 30 天保留策略
- `integration` 与 `proxy` 在启动完成数据库初始化后，都会自动检查并物理删除超过 30 天的日志
- 清理任务改为使用独立 sqlite 连接异步执行，不阻塞首屏服务、页面初始化请求和主查询链路
- 新增 `GET /api/log_cleanup_status`，用于返回当前启动阶段日志清理状态
- Site 在检测到清理正在进行时，会显示统一中心浮层并锁定界面，提示用户“正在清理过期日志，请稍后”

## 清理规则

- 清理范围固定为：
  - `agent_message_log`
  - `chat_log`
- 过期判断字段统一为 `created_at`
- 时间格式继续沿用日志表已有的 `2006-01-02T15:04:05.000`
- 删除条件固定为：

```text
created_at < 当前时间 - 30天
```

- `created_at` 为空字符串的历史数据不会被这次清理命中

## 启动行为

- `integration` 启动时，在数据库初始化完成后会异步触发一次清理
- `proxy` 启动时也会按同样方式异步触发一次清理
- 这次能力不是定时循环任务；当前只要求在每次启动时自动检查一次
- 清理失败不会阻塞主服务启动，但会写入标准日志与状态接口，便于排查

## 状态接口

请求：

```text
GET /api/log_cleanup_status
```

返回字段包括：

- `checked`
- `running`
- `message`
- `retentionDays`
- `cutoff`
- `startedAt`
- `finishedAt`
- `deletedAgentMessageLog`
- `deletedChatLog`
- `error`

## 页面表现

- Site 初始化后会轮询 `/api/log_cleanup_status`
- 当 `running=true` 时，页面会显示居中的统一浮层，并锁定界面交互
- 提示文案固定为：
  - `日志清理中`
  - `正在清理过期日志，请稍后`
- 清理结束后，浮层会自动关闭，不需要用户手动确认

## 兼容性说明

- 现有 `chat_log` 与 `agent_message_log` 的查询接口、写入链路和索引定义保持不变
- 本次迭代不新增日志表，也不把物理删除改成软删除
- 为避免首屏卡顿，清理任务不会复用主共享 sqlite 连接，而是使用独立连接异步完成
- 主手册 `../../USER_GUIDE.md` 已同步补充本次日志清理与状态接口说明
