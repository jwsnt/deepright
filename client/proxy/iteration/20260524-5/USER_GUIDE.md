本次迭代将 Proxy 对外暴露的蜂群开关统一收口为 `router_disable`。

## 变更点
- `swarm` 对外参数改为 `router_disable`
- 语意改为相反：
  - `router_disable=true` 表示关闭 router
  - `router_disable=false` 表示开启 router
- 默认值改为 `true`

## HTTP
- `POST /api/plugins/config`
  - 新参数：`router_disable`
- `GET /api/plugins/meta`
  - 返回字段改为 `router_disable`

- `POST /api/cron/create?agentId=xxx`
  - 请求体字段改为 `router_disable`
  - 不传时默认 `true`

- `POST /api/cron/detail/metadata`
  - 返回字段改为 `router_disable`

## CLI
- `proxy create`
- `proxy create-cron`
  - 新参数：`--router_disable[=true|false]`
  - 默认值：`true`

## 插件配置 CLI
- `proxy plugins config`
  - 新参数：`--router_disable[=true|false]`
  - 默认值：`true`

## 兼容说明
- SQLite 内部仍复用历史 `swarm` 列做持久化
- 历史 `swarm` 不再作为新的 Proxy 入参
