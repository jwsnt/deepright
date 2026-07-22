# 迭代 20260722-1：备忘录预览运行接口

Integration 提供 `POST /api/cron/detail/run?agentId=xxx`，用于从已有任务明细或任务元数据创建一条当前时间的待执行明细。

请求体：

```json
{
  "sourceType": "detail",
  "detailId": 123,
  "reuseChat": false
}
```

`sourceType` 可为 `detail` 或 `meta`：使用 `meta` 时改传 `metaId`。`reuseChat` 必填；为 `false` 时新明细不继承会话，为 `true` 时继承来源会话。

接口只创建 `started = 0` 的新明细，不直接执行任务。明细来源仅允许待执行、无需启动或已完成状态；已启动任务会被拒绝。创建与审计日志在同一事务中提交；如同一元数据下当前秒已存在明细，服务端会顺延秒级执行时间后创建。
