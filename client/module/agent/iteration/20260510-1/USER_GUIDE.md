# Agent 迭代 20260510-1 使用手册

## 变更说明

本次迭代为 Agent 元数据新增了可选的 `knowledge` 字段，用于对外暴露当前应用知识库的绝对路径与最后更新时间。

字段结构如下：

```json
{
  "knowledge": {
    "lastUpdate": 0,
    "path": "/app/knowledge"
  }
}
```

## 字段规则

- `knowledge` 作为 Agent 顶层元数据的可选字段输出
- 如果知识库不存在，则不添加该字段
- `path`
  - 为知识库绝对路径
  - 固定解析为应用启动目录下的 `knowledge`
- `lastUpdate`
  - 为知识库最后更新时间
  - 来自同一应用目录下共享 `data` sqlite 中的 `knowledge_runtime.last_update`
  - 如果目录存在但没有有效记录，则为 `0`

## 应用启动目录判定

Agent 模块不会额外维护一套 knowledge 路径规则，而是按以下方式解析应用启动目录：

- 显式传入 `--app-dir` 时，优先使用 `--app-dir`
- 未显式传入时，默认使用扫描根目录的父目录

例如：

```bash
./agent-scanner --app-dir /srv/my-app /srv/my-app/agent
```

则：

- knowledge 路径按 `/srv/my-app/knowledge` 解析
- sqlite 路径按 `/srv/my-app/data` 解析

## 输出规则

- `<app-dir>/knowledge` 不存在：
  - 不输出 `knowledge`
- `<app-dir>/knowledge` 存在，但 `<app-dir>/data` 不存在：
  - 输出 `knowledge`
  - `lastUpdate = 0`
- 目录和数据库都存在：
  - 输出真实的 `path`
  - 输出真实的 `lastUpdate`

## 设计说明

本次迭代遵守整体设计文档：

- knowledge 字段不是在 Agent 入口层临时拼装一份新结构
- Agent 仅负责复用共享能力，并把结果暴露为统一 metadata
- 若 knowledge 不存在，不做自动创建，保持 Agent 模块只读探测职责

## 使用示例

默认按扫描目录父目录推断应用启动目录：

```bash
./agent-scanner ./agent
```

显式指定应用启动目录：

```bash
./agent-scanner --app-dir /srv/my-app ./agent
```

## 补充说明

- 本次迭代只扩展 Agent 元数据输出，不改变 Agent 扫描、Skill 扫描、插件探测等既有行为
- 更完整的参数说明、输出示例和子模块调用方式，请查看上级手册：
  - `../../USER_GUIDE.md`
