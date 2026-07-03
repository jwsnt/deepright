# 20260525-3 使用说明

## 变更说明

- 右上角创建备忘录区域在 `Thinking/Auto` 开关左侧新增 `SWARM` 开关，默认关闭。
- 打开保存确认浮层时，会额外展示当前备忘录的蜂群状态，区分 `开启` 和 `关闭`。
- 创建备忘录时，页面会把当前开关状态作为 `router_disable` 字段一并提交到 `POST /api/cron/create?agentId=xxx`。
- 周期备忘录列表 hover、当天任务明细 hover，以及已完成任务的“查看会话”确认浮层，都会展示当前任务的 `SWARM` 状态。
- 周期备忘录列表会直接显示每条任务当前的 `SWARM` 状态，方便核对接口返回的 `router_disable`。

## 验证方式

1. 在右上角备忘录区域开启 `SWARM`，输入内容并点击保存。
2. 在浏览器 Network 中检查 `/api/cron/create` 请求体，确认开启 SWARM 时包含 `router_disable: false`。
3. 重置备忘录或切换 Agent 后，确认 `SWARM` 会恢复为默认关闭状态。
# 2026-05-24 修订

- 创建备忘录时，请求体统一发送 `router_disable`。
- SWARM 开启时发送 `router_disable=false`，关闭时发送 `router_disable=true`。
- 旧文档中的 `swarm` 字段已不再作为当前规范写法。
