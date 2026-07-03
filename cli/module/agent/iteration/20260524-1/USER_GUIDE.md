# Agent 迭代手册（20260524-1）

## 本次变更

本次迭代将 Agent 配置里的蜂群开关从 `swarm` 改为 `router_disable`，字段类型仍为 `boolean`，但语义改为相反值：

- `router_disable=true`：关闭蜂群路由
- `router_disable=false`：开启蜂群路由

## 配置格式

每个 Agent 工作目录下可选放置一个 `config.json`：

```json
{
  "description": "agent description",
  "provider": "deepseek",
  "thinking": true,
  "router_disable": true
}
```

说明：

- `config.json` 不存在时，不报错
- `description` 未配置时，输出空字符串
- `provider` 未配置时，输出空字符串
- `thinking` 未配置时，输出 `false`
- `router_disable` 未配置时，输出 `true`

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
      "router_disable": true,
      "soul": "",
      "user": "",
      "skills": []
    }
  ]
}
```

## 实时读取规则

- `router_disable` 与同一份 `config.json` 中的 `description`、`provider`、`thinking` 一样，都会在每次读取 Agent metadata 时实时重新获取
- 这些字段不受 `--agent-cache` 影响
- 即使进程内其他共享 metadata 仍命中缓存，只要 `config.json` 被修改，下一次 Agent metadata 输出就会立即反映最新结果

## 兼容说明

- 当前对外输出统一使用 `router_disable`
- 为兼容历史 Agent 配置，读取 `config.json` 时仍接受旧字段 `swarm`
- 如果旧配置中存在 `swarm=true`，会自动转换为 `router_disable=false`
- 如果旧配置中存在 `swarm=false`，会自动转换为 `router_disable=true`
