# 20260525-4 User Guide

本次迭代为插件浮层补齐了 SWARM 配置，并统一改为 `router_disable` 出参。

- 插件浮层里新增 `SWARM` 开关
- 开关名称仍显示为 `SWARM`
- 保存插件配置时，请求参数改为 `router_disable`
- 再次打开插件浮层时，会优先回显已保存的 `router_disable`

规则：

- SWARM 开启 -> `router_disable=false`
- SWARM 关闭 -> `router_disable=true`
- 默认值为 `true`

