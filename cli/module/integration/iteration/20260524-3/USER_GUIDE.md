# 20260524-3 User Guide

本次迭代把 Integration 的 cron 创建、查询和落库字段统一改为 `router_disable`。

- 请求字段：`router_disable`
- 返回字段：`router_disable`
- 落库字段：`task_meta.router_disable`、`task_detail.router_disable`
- 默认值：`true`
- 执行字段：真正转发 `/v1/chat/completions` 时固定写入 `metadata.router_disable`
- 优先级：执行阶段优先使用当前 `task_detail.router_disable`，不会回退到 Agent `config.json`
- 周期任务后续自动拆分出的新 `task_detail` 会继承所属 `task_meta.router_disable`

兼容规则：

- `router_disable` 是唯一对外字段
- 历史 `swarm` 仅作为旧库迁移来源
