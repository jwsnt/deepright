# Integration 迭代 20260705-1 使用手册

本次迭代为 `integration` 的 `/api/restore` 补齐了右侧 CLI 子任务 `CMD` 的历史恢复数据来源，方便 site 在重新打开会话或轮询续拉时，按统一时间线重建 `cli/get -> cli/pub` 对应的命令执行记录。

## 变更说明

- `/api/restore` 继续沿用现有接口收口，不新增新的恢复接口
- 返回结构仍保持原来的统一 `data[]` 记录数组格式
- 在原有 `chat_log` 消息恢复结果之外，接口会继续合并同一 `agentId + chatId` 下的 CLI 事件日志
- 这次合并的 CLI 记录类型为 `cli/get` 和 `cli/pub`，并分别通过 `role=cli/get`、`role=cli/pub` 明确标识

## 返回内容

- 合并返回的 CLI 记录会保留 `id`、`agentId`、`chatId`、`content`、`logType`、`createdAt`
- `cli/get` 的 `content` 会保留原始任务载荷，兼容直接 `cmd` 字段、嵌套 `message`、`messages[].content` 等既有格式
- `cli/pub` 的 `content` 会保留原始执行结果，兼容纯文本输出和 JSON 包裹的消息结构
- restore 链路不会额外发明新的 `cmd restore` 专用字段，避免打破现有日志写入和消费兼容性

## 排序与增量恢复

- 消息记录与 CLI 事件日志会在服务端合并后统一排序
- 排序规则固定为先按 `createdAt` 升序，再按 `id` 升序
- 当前端请求带有 `lastId` 时，CLI 日志也会按同一增量边界续拉，避免重复消费已经恢复过的记录
- 这样前端只需要按单一时间线消费 `data[]`，就可以把 `cli/get -> cli/pub` 配对为同一条右侧 `CMD` 子任务

## 容错行为

- 如果当前环境下 CLI 事件日志查询失败，不会影响原有 `chat_log` restore 主流程
- 接口会继续按现有容错方式返回已拿到的消息记录，而不是让整个 `/api/restore` 失败
- 这次改动只补充 restore 对 CLI 子任务历史的恢复能力，继续复用现有 CLI 日志写入链路和存储表，不新增新的日志表、消息表或额外后台任务

## 同步结果

- 主手册 [../../USER_GUIDE.md](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/USER_GUIDE.md) 已同步补充 `/api/restore` 合并返回 `cli/get`、`cli/pub` 的总体说明
- 本次手册更新只描述 integration 的 restore 返回语义，不改变现有 CLI 执行链路和日志写入方式
