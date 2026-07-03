# 20260525-5 使用说明

## 变更说明

- 设置中的蜂群开关展示名称保持不变，仍显示为 `蜂群`。
- 当前 Agent 的 `config.json` 持久化字段由 `swarm` 改为 `router_disable`。
- 字段语义与原来相反：`router_disable=true` 表示关闭蜂群，`router_disable=false` 表示开启蜂群。
- 页面读取配置时兼容旧的 `swarm` 字段，避免已有 Agent 配置在升级后失效。

## 验证方式

1. 打开左下角 `设置`，展开蜂群配置，分别保存一次开启和关闭状态。
2. 检查对应 Agent 的 `config.json`，确认写入的是 `router_disable`，且开启时为 `false`、关闭时为 `true`。
3. 准备一份仍只包含旧字段 `swarm` 的 `config.json`，重新打开设置后确认蜂群开关仍能正确回显。
