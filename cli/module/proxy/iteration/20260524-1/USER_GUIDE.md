# 20260524-1 使用说明

## 迭代目标

本次迭代将 `proxy` 侧 `/api/agent/create` 的 `name` 语义升级为“Agent workspace 内相对路径”，允许按当前浏览目录创建子目录或文件，不再因为路径中包含 `/` 就误报 `name contains invalid characters`。

## 行为说明

- `GET /api/agent/create?agentId=xxx&name=yyy&type=zzz`
- `name` 现在支持 `docs/data`、`tmp/a/b` 这类相对路径
- 每个路径段必须非空，且不能为 `.`、`..`
- 路径段中不允许包含空格和 `\:*?"<>|`
- 禁止绝对路径、`~`、`../` 等越界写入
- `type=0` 创建目录，`type=1` 创建文件
- 如果父目录不存在，会按相对路径自动创建
- 已存在、Agent 不存在、参数缺失等既有错误语义保持不变

## 验收建议

1. 调用 `/api/agent/create?agentId=...&name=docs/data&type=0`，确认创建成功。
2. 调用 `/api/agent/create?agentId=...&name=docs/note.md&type=1`，确认文件创建在 workspace 子目录内。
3. 调用 `/api/agent/create?agentId=...&name=../escape&type=0`、`name=~/escape`、`name=/tmp/a`，确认请求失败且 workspace 外没有产生文件。
