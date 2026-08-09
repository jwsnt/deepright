# Proxy 迭代手册（20260524-4）

## 本次变更

本次迭代为 Proxy 的插件配置链路补齐了 `router_disable` 字段透传、持久化与桥接继承能力。

## 接口变化

### 1. `POST /api/plugins/config`

- 新增可选布尔参数 `router_disable`
- 不传时默认 `true`
- 传入后会写入共享存储 `connect_meta`

示例：

```text
POST /api/plugins/config?key=feishu&agentId=A&chatId=chat-001&model=OpenAI&thinking=true&router_disable=false
```

### 2. `GET /api/plugins/meta`

- 返回结果中的每个插件对象新增 `router_disable`
- 已保存插件配置时会返回对应值；未设置时默认为 `true`

返回示例：

```json
{
  "status": 0,
  "data": [
    {
      "key": "feishu",
      "name": "飞书",
      "router_disable": false
    }
  ]
}
```

## 桥接行为

- 当插件配置 `router_disable=false` 时，命中 `add-request` 转备忘录明细的桥接链路后，生成的 `task_meta` 与 `task_detail` 会继承相同的 `router_disable`
- 未传或为 `true` 时，桥接出的任务保持默认关闭状态

## 兼容性说明

- `router_disable` 是唯一对外字段
- 历史 `swarm` 仅作为旧库迁移来源
- 主手册 `../../USER_GUIDE.md` 已同步更新，适合查看完整接口与示例
# 2026-05-24 修订

- 本迭代文档里提到的插件配置字段，现统一以 `router_disable` 为准。
- `router_disable=true` 表示关闭 SWARM，`router_disable=false` 表示开启 SWARM。
- `/api/plugins/config`、`/api/plugins/meta` 与桥接出来的 cron 任务都使用 `router_disable`。
- 页面仍显示 `SWARM`，但 Proxy CLI 与接口统一使用 `router_disable`。
