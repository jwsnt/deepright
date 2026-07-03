# 20260524-3 User Guide

本次迭代把 Proxy 的 cron 创建、查询与返回字段统一切到 `router_disable`。

- 请求字段：`router_disable`
- 返回字段：`router_disable`
- 默认值：`true`
- 语义：`true` 表示关闭 SWARM，`false` 表示开启 SWARM
- 执行字段：真正转发 `/v1/chat/completions` 时固定写入 `metadata.router_disable`
- 优先级：执行阶段优先使用当前 `task_detail.router_disable`，不会回退到 Agent `config.json`

兼容说明：

- `router_disable` 是唯一对外字段
- 历史 `swarm` 仅作为旧库迁移来源
