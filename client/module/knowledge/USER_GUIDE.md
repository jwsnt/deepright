# Knowledge 使用手册

## 简介

`knowledge` 模块负责提供知识库目录与其运行时元数据的统一能力。

- 应用启动目录下不存在 `knowledge` 目录时，会自动创建
- 与 `proxy` 共用同一个 `data` sqlite 文件
- sqlite 中会按 `agent_id` 维度维护 `last_update` 与 `knowledge_commit`
- 对外可输出 `metadata.knowledge={path,lastUpdate,knowledgeCommit}`，供 `/v1/chat/completions`、`/cli/get`、`/cli/pub` 等链路复用

## 核心规则

- 应用启动目录指的是业务二进制的启动目录，例如 `integration` 或 `proxy` 的当前工作目录
- 默认知识库根目录固定为 `<app-dir>/knowledge`
- 当指定 `agentId` 时，当前 Agent 的知识库目录固定为 `<app-dir>/knowledge/<agentId>`
- 共享 sqlite 固定为 `<app-dir>/data`
- `knowledge_runtime` 表按 `agent_id` 保存运行时：
  - `last_update` 初始值为 `0`
  - `knowledge_commit` 初始值为 `false`
- 创建型接口会自动补齐目录和表结构
- 只读型接口不会创建任何目录或数据库文件；若知识库不存在，则直接返回空结果

## 目录与数据

- 知识库根目录固定为应用启动目录下的 `knowledge`
- Agent 知识库目录固定为 `knowledge/<agentId>`
- sqlite 固定为应用启动目录下的 `data`
- 表 `knowledge_runtime` 按 `agent_id` 保存多条记录：
  - `last_update=0` 初始值
  - `knowledge_commit=false` 初始值

## 作为子模块调用

推荐由其他模块直接调用 `knowledgecore`：

- `EnsureRuntime(appDir)` / `EnsureRuntimeForAgent(appDir, agentID)`：确保知识库目录和 sqlite 状态存在
- `LookupRuntime(appDir)` / `LookupRuntimeForAgent(appDir, agentID)`：只读探测现有知识库；不存在时返回 `nil`
- `Metadata(appDir)` / `MetadataForAgent(appDir, agentID)`：返回可注入到请求 metadata 的 `knowledge` 字段，并确保底层目录/状态存在
- `MetadataIfExists(appDir)` / `MetadataIfExistsForAgent(appDir, agentID)`：仅在知识库已存在时返回 `knowledge` 字段
- `MergeMetadata(base, appDir)`：在已有 metadata 上补充 `knowledge`
- `MergeMetadataIfExists(base, appDir)`：仅在知识库已存在时补充 `knowledge`
- `OpenSharedDB(appDir)`：打开共享 `data` sqlite，并复用连接池
- `SetLastUpdate(db, ts)` / `SetLastUpdateForAgent(db, agentID, ts)`：更新最后更新时间
- `SetKnowledgeCommit(db, value)` / `SetKnowledgeCommitForAgent(db, agentID, value)`：更新知识库提交标记

### 推荐调用方式

- 应用启动阶段：
  - 调用 `EnsureRuntime(appDir)` 或 `EnsureRuntimeForAgent(appDir, agentID)`，确保知识库目录和默认运行时记录存在
- 元数据注入阶段：
  - 如果当前链路允许“自动初始化知识库”，调用 `Metadata(appDir)` 或 `MergeMetadata(...)`
  - 如果当前链路只允许“已有则带上，没有就忽略”，调用 `MetadataIfExists(appDir)` 或 `MergeMetadataIfExists(...)`
- 知识库内容发生更新后：
  - 复用 `OpenSharedDB(appDir)` 返回的共享连接池
  - 调用 `SetLastUpdate(db, ts)` 或 `SetLastUpdateForAgent(db, agentID, ts)` 更新最后更新时间
- 如果链路需要记录显式知识库提交：
  - 调用 `SetKnowledgeCommit(db, value)` 或 `SetKnowledgeCommitForAgent(db, agentID, value)` 更新提交标记

## CLI 用法

```bash
knowledge ensure --app-dir /path/to/app
knowledge get --app-dir /path/to/app
knowledge metadata --app-dir /path/to/app
knowledge metadata --app-dir /path/to/app --read-only
knowledge metadata --app-dir /path/to/app --agent-id agent-a
knowledge update-time --app-dir /path/to/app --timestamp 1715337600000
knowledge update-time --app-dir /path/to/app --agent-id agent-a --timestamp 1715337600000
knowledge update-commit --app-dir /path/to/app --agent-id agent-a --value true
knowledge update-time 1715337600000
```

### 命令说明

- `ensure`
  - 确保 `knowledge` 目录与 sqlite 记录存在
  - 输出当前状态 JSON
- `get`
  - 输出当前状态 JSON
- `metadata`
  - 输出可供上游注入的 metadata 片段
  - 默认会确保 `knowledge` 目录与 sqlite 记录存在
  - 传 `--read-only` 时只读探测；若知识库不存在则输出 `{}`
- `update-time`
  - 更新 `last_update`
  - 支持 `knowledge update-time 时间戳`
  - 未传 `--timestamp` 时默认使用当前 Unix 毫秒时间戳
- `update-commit`
  - 更新 `knowledge_commit`
  - 支持 `knowledge update-commit true|false`

### 常见用法

初始化知识库目录与状态：

```bash
knowledge ensure --app-dir /srv/my-app
```

初始化某个 Agent 的知识库目录与状态：

```bash
knowledge ensure --app-dir /srv/my-app --agent-id agent-a
```

只在知识库已存在时输出 metadata 片段：

```bash
knowledge metadata --app-dir /srv/my-app --read-only
```

知识库更新完成后写入最新更新时间：

```bash
knowledge update-time --app-dir /srv/my-app --timestamp 1715337600000
```

知识库整理完成后写入 Agent 级提交标记：

```bash
knowledge update-commit --app-dir /srv/my-app --agent-id agent-a --value true
```

使用位置参数更新时间：

```bash
knowledge update-time 1715337600000
```

## 输出示例

```json
{
  "path": "/app/knowledge/agent-a",
  "lastUpdate": 0,
  "knowledgeCommit": false
}
```

metadata 片段：

```json
{
  "knowledge": {
    "path": "/app/knowledge/agent-a",
    "lastUpdate": 0,
    "knowledgeCommit": false
  }
}
```

## 注意事项

- `Metadata(appDir)` 与 `ensure` 一样，都会自动创建缺失的 `knowledge` 目录和 sqlite 状态
- `MetadataIfExists(appDir)` 与 `metadata --read-only` 不会做任何创建动作
- `lastUpdate` 允许为 `0`，表示知识库已初始化但尚未发生过实际更新
- `SetLastUpdate` 不接受负数时间戳
- `update-time` 写入和展示的 `lastUpdate` 统一使用 Unix 毫秒时间戳
- `knowledgeCommit` 会按 Agent 独立保存最近一次显式提交值

---

## 迭代 20260622-01：按 Agent 隔离知识库

知识库不再在多个 Agent 间共享：每个 Agent 使用其自身的 `knowledge/<agentId>` 目录、WIKI 首页和运行时状态。查询 metadata 时会返回当前 Agent 对应的路径、最后更新时间及提交状态；缺失目录会自动初始化。

---

## 迭代 20260622-01：按 Agent 隔离知识库

知识库目录和运行状态现在按 Agent 独立维护。使用同一套运行目录的不同 Agent 不会共享 `lastUpdate`、`knowledgeCommit` 或 WIKI 首页；首次访问时会自动创建该 Agent 的知识库目录和 `index.md`。

更新指定 Agent 的知识库状态时，可在 `knowledge update-time` 或 `knowledge update-commit` 命令中传入 `--agent-id`。
