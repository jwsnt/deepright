# Proxy 迭代 20260510-1 使用手册

## 目标

本次迭代为 `proxy` 转发链路中的 Agent 元数据补充可选的 `knowledge` 字段。

目标结构如下：

```json
{
  "knowledge": {
    "lastUpdate": 0,
    "path": "/app/knowledge"
  }
}
```

## 行为说明

`proxy` 在处理 `/v1/chat/completions` 请求时，仍然先读取统一的 Agent 元数据，再把结果注入到请求体的 `metadata` 中。

本次变化后，如果当前应用启动目录下存在知识库，则转发到上游的 `metadata` 会额外包含：

```json
{
  "knowledge": {
    "path": "/abs/path/knowledge",
    "lastUpdate": 0
  }
}
```

同时，proxy 内部发起的 cron 执行请求 metadata 也会复用同一份结构。

## 字段来源

- `proxy` 不单独实现 knowledge 逻辑
- `knowledge` 直接复用 Agent 模块最新元数据输出
- `path`
  - 固定解析为当前应用启动目录下的 `knowledge` 绝对路径
- `lastUpdate`
  - 来自当前应用启动目录下共享 `data` sqlite 中的 `knowledge_runtime.last_update`
  - 若目录已存在但尚未写入更新时间，则为 `0`

## 输出规则

- 当前应用启动目录下不存在 `knowledge` 目录：
  - metadata 中不包含 `knowledge`
- 存在 `knowledge` 目录，但没有 `data` 数据库：
  - 仍输出 `knowledge`
  - `lastUpdate = 0`
- 目录和数据库都存在：
  - 输出真实的 `path`
  - 输出真实的 `lastUpdate`

## 当前模块内已覆盖链路

本次迭代在 `proxy` 模块目录内实际覆盖的是：

- `/v1/chat/completions`
- proxy 内部 cron 执行请求

补充说明：

- 当前 `proxy` 模块目录内没有实际的 `/cli/get`、`/cli/pub` 路由实现
- 如果这些链路由 `integration` 收口，则应在 `integration` 中继续复用同一份 Agent 元数据输出

## 使用方式

本次迭代没有新增独立 CLI 命令或新的 HTTP 路径。

只要按原方式启动 `proxy`，并确保当前应用启动目录下已初始化知识库，即可自动获得带 `knowledge` 的 metadata 转发能力。

例如：

```bash
cd /path/to/app
knowledge ensure --app-dir .
proxy --agent-dir ./agent
```

之后调用：

```text
POST /v1/chat/completions
```

即可在转发 metadata 中看到 `knowledge`。

## 兼容说明

- 不修改 `/v1/chat/completions` 的路径、SSE 转发方式和响应格式
- 不改变原有 `metadata` 追加覆盖规则
- 不新增新的 HTTP 路由
- 不在 proxy 入口层单独维护一份 knowledge 实现，继续遵守共享内核 + 薄包装设计

更完整的启动参数和模块说明，请查看上级手册：

- `../../USER_GUIDE.md`
