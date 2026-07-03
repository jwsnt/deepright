# Agent 迭代手册（20260516-1）

## 本次变更

本次迭代为每个 Agent 增加了 `provider` 元数据字段，并统一从 Agent 工作目录下的 `config.json` 读取。

当前支持读取的 `config.json` 字段包括：

- `description`
- `provider`
- `thinking`
- `swarm`

## 配置格式

每个 Agent 工作目录下可选放置一个 `config.json`：

```json
{
  "description": "agent description",
  "provider": "deepseek",
  "thinking": true,
  "swarm": false
}
```

说明：

- `config.json` 不存在时，不报错
- `provider` 未配置时，输出空字符串
- `thinking`、`swarm` 未配置时，输出 `false`
- `description` 未配置时，输出空字符串

## 输出示例

```json
{
  "agents": [
    {
      "agentId": "demo",
      "workspace": "/abs/path/demo",
      "description": "agent description",
      "provider": "deepseek",
      "thinking": true,
      "swarm": false,
      "soul": "",
      "user": "",
      "skills": []
    }
  ]
}
```

## 实时读取规则

- `provider` 与同一份 `config.json` 中的 `description`、`thinking`、`swarm` 一样，都会在每次读取 Agent metadata 时实时重新获取
- 这些字段不受 `--agent-cache` 影响
- 即使进程内其他共享 metadata 仍命中缓存，只要 `config.json` 被修改，下一次 Agent metadata 输出就会立即反映最新结果

## 对外影响

- `agent-scanner` CLI 输出会包含 `agents[].provider`
- 作为共享内核的 `agentcore` 会把该字段同步提供给 `proxy`、`integration`、`cli-get` 等复用方
- 上游如果已经消费整份 Agent metadata，无需额外切换接口，只需按新字段读取即可
