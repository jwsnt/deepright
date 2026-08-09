# 20260524-1 使用说明

## 迭代目标

本次迭代让 `integration` 收口的 `/api/agent/create` 与 `proxy` 保持一致，支持在 Agent workspace 内按相对路径创建子目录或文件。

## 行为说明

- `GET /api/agent/create?agentId=xxx&name=yyy&type=zzz`
- `name` 表示 workspace 内相对路径，支持 `docs/data`、`tmp/a/b` 这类多段路径
- 每个路径段必须非空，且不能为 `.`、`..`
- 路径段中不允许包含空格和 `\:*?"<>|`
- 禁止绝对路径、`~`、`../` 或其他越界写入
- `type=0` 创建目录，`type=1` 创建文件
- 父目录不存在时会自动补齐
- 已存在、Agent 不存在、参数缺失等原有返回语义保持不变

## 验收建议

1. 调用 `/api/agent/create?agentId=...&name=docs/data&type=0`，确认返回成功。
2. 调用 `/api/agent/create?agentId=...&name=docs/note.md&type=1`，确认文件落在目标 workspace 子目录。
3. 调用 `name=../escape`、`name=~/escape`、`name=/tmp/a`，确认被拒绝且没有越界写入。
