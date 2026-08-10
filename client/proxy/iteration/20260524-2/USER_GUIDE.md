# 20260524-2 使用说明

## 迭代目标

本次迭代让 `proxy` 的新建 Agent 行为改为“按默认模板目录初始化”，不再只补 `SOUL.md`、`USER.md` 和 `skills/`。

## 行为说明

- 新增服务启动参数 `--default-dir`
- 未显式传参时，默认使用应用启动目录下的 `config/`
- `GET /api/agent/init?name=xxx` 创建 Agent 后，会把 `default-dir` 目录中的全部内容复制到该 Agent 目录
- `default-dir` 缺失、不是目录或复制失败时，请求会直接失败
- 复制失败会回滚刚创建的 Agent 目录，避免残留半成品目录

## 验收建议

1. 启动 `proxy serve --agent-dir ./agent --default-dir ./config` 后调用 `/api/agent/init?name=test-agent`，确认新 Agent 内包含 `config/` 下的全部默认文件。
2. 删除或改错 `default-dir` 后再次调用 `/api/agent/init`，确认接口返回失败且 `agent/test-agent` 不会残留空目录。
3. 不传 `--default-dir` 时，从应用启动目录下准备 `config/` 并重复创建，确认默认目录生效。
