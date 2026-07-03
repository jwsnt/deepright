# 20260525-1 使用说明

## 迭代目标

本次迭代把 `integration` 启动时的空 `--agent-dir` 自动补齐行为，收口为和 `/api/agent/init` 一致的默认模板初始化流程。

## 行为说明

- 启动 `integration` 服务时，如果 `--agent-dir` 指向空目录（包括刚自动创建完成的场景），会自动创建 `DEF_AGENT/`
- 创建 `DEF_AGENT/` 时，会把 `--default-dir` 指向目录中的全部内容复制进去
- 未显式传 `--default-dir` 时，默认使用应用启动目录下的 `config/`
- 复制完成后，仍会确保 `DEF_AGENT/skills` 存在
- 如果 `default-dir` 不存在、不是目录或复制失败，服务启动会直接报错，避免留下半初始化的 `DEF_AGENT`

## 验收建议

1. 启动 `./integration --agent-dir ./agent --default-dir ./config`，并确保 `./agent` 启动前为空目录，确认启动后 `agent/DEF_AGENT` 内包含 `config/` 下的默认模板文件。
2. 检查 `agent/DEF_AGENT/skills` 目录仍然存在，确认模板复制没有破坏既有目录约定。
3. 把 `--default-dir` 改成不存在的目录后重新启动，确认启动直接失败，且 `agent/DEF_AGENT` 不会残留半成品内容。
