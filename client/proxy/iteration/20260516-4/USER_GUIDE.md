# Proxy 迭代手册（20260516-4）

## 本次变更

本次迭代把 Agent 元数据中的 `provider` 收口到 `proxy` 对外转发链路。

覆盖范围：

- `/v1/chat/completions`
- `cli/get`
- `cli/pub`
- proxy 内部 cron 执行时构造的 metadata

## 字段来源

- `metadata.agents[].provider` 来自对应 Agent 工作目录下的 `config.json.provider`
- 如果 `config.json` 不存在，或未声明 `provider`，则输出空字符串

同时，`config.json` 中同一批 Agent 属性也会一起复用：

- `description`
- `provider`
- `thinking`
- `swarm`

## 实时读取规则

- `provider` 与同一份 `config.json` 中的 `description`、`thinking`、`swarm` 一样，都会在每次请求前实时重新读取
- 这些字段不受 `--agent-cache` 影响
- 即使 `proxy` 进程使用了较长的 `--agent-cache`，只要 Agent 工作目录下的 `config.json` 有更新，下一次 metadata 注入结果就会立刻生效

## 转发示例

当某个 Agent 的 `config.json` 为：

```json
{
  "description": "demo agent",
  "provider": "deepseek",
  "thinking": true,
  "swarm": false
}
```

则 `proxy` 转发时的 metadata 中会包含类似：

```json
{
  "agents": [
    {
      "agentId": "demo",
      "description": "demo agent",
      "provider": "deepseek",
      "thinking": true,
      "swarm": false
    }
  ]
}
```

## 兼容性说明

- 现有 metadata 合并规则不变，请求体中业务方自己传入的 `metadata` 仍会覆盖同名共享字段
- `provider` 只是新增字段，不会改变原有 `/v1/chat/completions`、`cli/get`、`cli/pub` 的协议结构
- 未配置 `provider` 的旧 Agent 仍可继续正常使用，只是该字段为空字符串
