# 20260611-1 迭代说明

本次迭代为 `proxy` 新增了 `GET /api/swarm_agent`，供前端实时获取当前已开启 SWARM 的 Agent 列表。

- 接口仅支持 `GET`；其他方法返回 `405 Method Not Allowed`
- 接口会基于共享 Agent 元数据实时扫描 Agent 目录
- 只返回 `router_disable=false` 的 Agent ID，也就是设置中已开启蜂群开关的 Agent
- 传入 `agentId=当前AgentId` 时，会把当前 Agent 从结果里过滤掉
- 返回顺序与 Agent 元数据扫描顺序一致
- 如果当前没有任何 Agent 开启蜂群，则返回空数组 `[]`
- Site 居中输入框的 `@ Agent` 菜单会直接复用这个接口
