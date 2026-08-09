# 20260611-6 用户手册

本次迭代为居中对话输入框的 `@` 唤起菜单补充了 `Agent` 入口，用于在开启 SWARM 的会话里快速插入团队协作气泡。

- 只有当前 `Agent + Chat` 会话的 `SWARM` 处于开启状态时，居中输入框输入 `@` 后的一级菜单才会显示 `Agent`
- `Agent` 菜单数据来自 `GET /api/swarm_agent?agentId=当前AgentId`
- 该接口只会返回当前已开启蜂群的 Agent，也就是 `router_disable=false` 的 Agent ID
- 返回结果会额外过滤掉当前 Agent 自身
- 如果接口返回空数组或请求失败，一级菜单不会显示 `Agent`
- 选择某个 Agent 后，输入框会在当前光标位置插入 `[TEAM:AgentId]` 气泡
- 发送消息时会按同名文本 `[TEAM:AgentId]` 透传；例如 `[TEAM:DEF_AGENT]`
- 这项扩展只作用于居中对话输入框，右侧备忘录的 `@` 菜单不增加 `Agent`
