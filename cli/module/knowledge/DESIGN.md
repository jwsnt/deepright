# Knowledge 模块详细技术设计

## 1. 设计定位

`knowledge` 模块不是知识内容管理系统，也不负责 Markdown 解析、向量检索、全文索引或知识生成。它当前承担的是一层非常明确的共享运行时职责：

- 统一知识库目录结构
- 统一知识库运行时状态持久化
- 向上游请求注入 `metadata.knowledge`
- 为 `site / proxy / integration` 提供一致的 Agent 级知识库视图

当前权威实现位于：

- [knowledgecore/core.go](/Users/shenjiawei/Documents/code/deepright/cli/module/knowledge/knowledgecore/core.go)

CLI 只是薄封装：

- [knowledgecli/cli.go](/Users/shenjiawei/Documents/code/deepright/cli/module/knowledge/knowledgecli/cli.go)
- [main.go](/Users/shenjiawei/Documents/code/deepright/cli/module/knowledge/main.go)

前后端联动的实际消费方主要是：

- `site`：右侧知识库 WIKI 面板、自动整理开关、小灯泡整理入口
- `proxy`：对外 `/v1/chat/completions` 转发与知识库 metadata 注入
- `integration`：站点收口、HTTP 接口、CLI 收口、知识库 metadata 注入

## 2. 当前总体架构

当前知识库链路可以按四层理解：

1. 前端展示层
   - `site/index.html`
   - 负责知识库 WIKI 展示、Agent 切换、手动整理、自动整理开关

2. 收口服务层
   - `proxy/main.go`
   - `integration/main.go`
   - 负责 HTTP 接口、请求 metadata 注入、手动整理完成后的回写

3. 共享运行时层
   - `knowledgecore`
   - 负责目录、sqlite、状态读写、metadata 组装

4. 持久化层
   - 磁盘知识库目录 `knowledge/`
   - 共享 sqlite `data`
   - 表 `knowledge_runtime`
   - 表 `knowledge_update_lock`（由 `connect/sharedutil` 维护，但服务于知识库刷新节流）

## 3. 端到端设计

### 3.1 Site 前端

`site` 侧当前有三类知识库相关能力：

- WIKI 浏览
  - HTTP 文档根路径：`/knowledge`
  - 当前 Agent 默认首页：`/knowledge/<agentId>/index.md`
  - 实际磁盘目录：`--agent-dir/knowledge/<agentId>`

- 状态展示
  - 通过 `/knowledge_lastUpdate?agentId=...` 获取当前 Agent 的最后更新时间文本
  - 通过 `/knowledge_path?agentId=...` 获取当前 Agent 知识库真实目录
  - 每个 Agent 单独缓存 `最近刷新`、过期提示关闭状态、当前文档位置

- 操作入口
  - 小灯泡手动整理知识库 WIKI
  - Agent 级“自动整理知识库 WIKI”开关
  - 刷新并展开 WIKI

### 3.2 Proxy / Integration 后端

`proxy` 与 `integration` 当前采用同一套知识库处理思路：

- 先合并基础 Agent metadata 与请求体 metadata
- 以顶层 `metadata.agentId` 作为当前知识库维度
- 把 `metadata.knowledge.path / lastUpdate / knowledgeCommit` 重写为当前 Agent 的真实值
- 如果请求带 `metadata.knowledge_commit=true`
  - 强制保留 `knowledge.lastUpdate`
  - 在 SSE 完整结束后回写该 Agent 的 `last_update`
  - 同时更新该 Agent 的 `knowledge_update_lock.last_requested_at`
- 如果不是手动提交请求
  - 会根据 `knowledge_update_interval` 与 `knowledge_update_lock` 决定是否删除本次转发中的 `knowledge.lastUpdate`

### 3.3 Knowledge Core

`knowledgecore` 是共享运行时中心，不感知浏览器 UI，也不感知 SSE，只提供：

- Agent 级知识库目录解析与创建
- Agent 级 `last_update / knowledge_commit` 读写
- `metadata.knowledge` 组装
- macOS 托管运行时目录适配

## 4. 模块边界

### 4.1 `main.go`

只做 CLI 转发与少量测试辅助，不承载核心业务。

### 4.2 `knowledgecli`

负责：

- 命令解析
- `--app-dir` / `--agent-id` 处理
- JSON 输出
- 错误格式输出

当前支持命令：

- `ensure`
- `get`
- `metadata`
- `update-time`
- `update-commit`
- `help`

### 4.3 `knowledgecore`

负责：

- 目录路径解析
- sqlite 路径解析
- sqlite schema 初始化
- Agent 级 `lastUpdate / knowledgeCommit`
- `State` / `metadata.knowledge`

### 4.4 `connect/sharedutil`

虽然不属于 `knowledge` 目录，但当前知识库整体设计依赖它维护：

- `knowledge_update_lock`
- `EnsureKnowledgeRequestLockRow(...)`
- `EnsureKnowledgeRequestLockRowTx(...)`

它解决的是“知识库更新申请节流”问题，而不是知识库内容存储问题。

## 5. 数据模型

### 5.1 核心状态结构

当前 `knowledgecore.State`：

```go
type State struct {
    Path            string `json:"path"`
    LastUpdate      int64  `json:"lastUpdate"`
    KnowledgeCommit bool   `json:"knowledgeCommit"`
}
```

对应语义：

- `path`
  - 当前 Agent 知识库目录的绝对路径
  - 不是 HTTP 路径，是磁盘路径

- `lastUpdate`
  - 当前 Agent 的知识库最后更新时间
  - Unix 毫秒时间戳
  - `0` 表示该 Agent 已初始化知识库状态，但尚未发生过一次成功的知识库更新时间回写

- `knowledgeCommit`
  - 当前 Agent 最近一次显式知识库提交开关值
  - 由请求里的顶层 `metadata.knowledge_commit` 持久化而来
  - 它表示“最近一次显式提交值”，不是“本次请求是否完成”

### 5.2 `metadata.knowledge` 形态

`knowledgecore.MetadataForAgent(...)` 输出：

```json
{
  "knowledge": {
    "path": "/abs/path/to/agent-dir/knowledge/agent-a",
    "lastUpdate": 1715337600000,
    "knowledgeCommit": false
  }
}
```

这是当前后端最终希望挂在请求 metadata 中的标准知识库片段。

## 6. 字段与使用场景

下面区分“持久化字段”和“请求时字段”。

### 6.1 持久化字段

| 字段 | 所在位置 | 作用 | 典型使用场景 |
| --- | --- | --- | --- |
| `knowledge_runtime.agent_id` | sqlite `knowledge_runtime` | 知识库状态维度键 | 每个 Agent 独立保存状态 |
| `knowledge_runtime.last_update` | sqlite `knowledge_runtime` | 当前 Agent 知识库最后更新时间 | 决定是否需要向上游声明知识库可能已过期 |
| `knowledge_runtime.knowledge_commit` | sqlite `knowledge_runtime` | 当前 Agent 最近一次显式提交值 | 让后端和调用方知道该 Agent 最近一次是否显式把请求标记为知识库提交 |
| `knowledge_update_lock.agent_id` | sqlite `knowledge_update_lock` | 更新申请节流维度键 | 防止同一个 Agent 短时间内重复申请知识库刷新 |
| `knowledge_update_lock.last_requested_at` | sqlite `knowledge_update_lock` | 最近一次“申请刷新知识库”的时间 | 普通请求转发时判断是否删除 `knowledge.lastUpdate` |

### 6.2 请求时字段

| 字段 | 位置 | 是否持久化 | 作用 | 当前主要来源 |
| --- | --- | --- | --- | --- |
| `metadata.agentId` | 顶层 metadata | 否 | 当前请求所属 Agent，后端 Agent 级知识库路由的权威字段 | `site` / 调用方 |
| `metadata.knowledge` | 顶层 metadata 内对象 | 否 | 当前请求知识库快照 | `site` 先带出，`proxy/integration` 再补全 |
| `metadata.knowledge.path` | `metadata.knowledge` 内 | 间接来自持久化与目录 | 当前 Agent 的真实知识库目录 | `proxy/integration` 注入 |
| `metadata.knowledge.lastUpdate` | `metadata.knowledge` 内 | 来自 `knowledge_runtime.last_update` | 告诉上游“当前知识库最后更新时间” | `proxy/integration` 注入 |
| `metadata.knowledge.knowledgeCommit` | `metadata.knowledge` 内 | 来自 `knowledge_runtime.knowledge_commit` | 暴露当前 Agent 最近一次显式提交值 | `proxy/integration` 注入 |
| `metadata.knowledge.agentId` | `metadata.knowledge` 内 | 否 | 让 `knowledge` 子对象自描述当前 Agent | `site` 注入 |
| `metadata.knowledge.knowledge_disable` | `metadata.knowledge` 内 | 否 | 前端 Agent 级“自动整理开关”状态 | `site` 注入 |
| `metadata.knowledge_commit` | 顶层 metadata | 是，持久化到 `knowledge_runtime.knowledge_commit` | 声明“本次请求是否为显式知识库提交” | `site` 手动整理时注入 |

### 6.3 字段关系说明

最容易混淆的几个字段关系如下：

- 权威 Agent 维度字段是顶层 `metadata.agentId`
  - `proxy` / `integration` 当前真正依赖它来读写知识库状态
  - `metadata.knowledge.agentId` 目前是一个“子对象自描述字段”，不是后端主路由键

- 顶层 `metadata.knowledge_commit` 与 `metadata.knowledge.knowledgeCommit` 不是一回事
  - 前者：本次请求的操作意图
  - 后者：当前 Agent 最近一次已持久化的显式提交状态

- `metadata.knowledge.knowledge_disable` 不属于 `knowledgecore` 持久化字段
  - 它是 `site` 前端的 Agent 级本地偏好
  - 当前通过请求 metadata 透传
  - `knowledge` 模块本身不写入 sqlite

## 7. 路径设计

### 7.1 标准目录

核心常量：

- `defaultKnowledgeDirName = "knowledge"`
- `defaultSQLiteFileName = "data"`
- `defaultRuntimeAppName = "deepright"`

因此基础资源固定为：

- `<runtime-app-dir>/knowledge`
- `<runtime-app-dir>/data`

### 7.2 Agent 级知识库目录

当前规则：

- 未指定 Agent：`<runtime-app-dir>/knowledge`
- 指定 Agent：`<runtime-app-dir>/knowledge/<agentId>`

对应 API：

- `KnowledgeDir(appDir)`
- `KnowledgeDirForAgent(appDir, agentID)`
- `EnsureKnowledgeDirForAgent(appDir, agentID)`

### 7.3 Site 视角的默认首页

当前页面约定的默认首页文档是：

- HTTP 路径：`/knowledge/<agentId>/index.md`
- 磁盘文件：`--agent-dir/knowledge/<agentId>/index.md`

注意：

- 这里的 `index.md` 是站点默认首页文档约定
- 目录根本身仍然是 `knowledge/<agentId>/`
- `knowledgecore` 只负责目录，不关心首页文件名

### 7.4 macOS 托管运行时目录

`resolveRuntimeAppDir(appDir)` 在 macOS 下会在以下场景切换到：

- `~/Library/Application Support/deepright`

触发条件包括：

- `appDir` 等于当前工作目录
- `appDir` 等于当前可执行文件所在目录
- 路径中包含 `.app/Contents/`

设计目的：

- 让 bundle 运行、双击启动、当前工作目录启动等场景共享同一套知识库运行时目录和 sqlite

## 8. SQLite 设计

### 8.1 `knowledge_runtime`

当前 schema：

```sql
CREATE TABLE knowledge_runtime (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_id TEXT NOT NULL UNIQUE,
    last_update INTEGER NOT NULL DEFAULT 0,
    knowledge_commit INTEGER NOT NULL DEFAULT 0
)
```

并保留一条默认兼容记录：

```sql
INSERT OR IGNORE INTO knowledge_runtime(agent_id, last_update, knowledge_commit)
VALUES ('', 0, 0)
```

设计含义：

- 状态已经从“全局单例”升级到“Agent 级多行”
- `agent_id=''` 仍然作为兼容默认行保留
- 每个 Agent 独立维护：
  - `last_update`
  - `knowledge_commit`

### 8.2 `knowledge_update_lock`

虽然它不在 `knowledgecore` 中创建，但当前整体知识库设计必须把它视为配套表：

```sql
CREATE TABLE knowledge_update_lock (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_id TEXT NOT NULL UNIQUE,
    last_requested_at INTEGER NOT NULL DEFAULT 0
)
```

设计含义：

- 它不是知识库状态表
- 它只负责记录“最近一次请求想触发知识库更新”的时间
- 用于请求节流，避免一个 Agent 在短时间内被多次重复申请刷新

### 8.3 PRAGMA 与连接池

`EnsureSchema(db)` 每次都会先执行：

```sql
PRAGMA journal_mode=WAL
```

`OpenSharedDB(appDir)` 使用包级共享连接池：

- 同一路径复用同一份 `*sql.DB`
- 新路径关闭旧连接后再打开新连接
- `MaxOpenConns = 4`
- `MaxIdleConns = 4`

`OpenExistingDB(appDir)` 用于只读探测：

- 不复用共享单例
- 只有 sqlite 文件真实存在时才打开
- `MaxOpenConns = 2`
- `MaxIdleConns = 2`

## 9. 核心 API 设计

### 9.1 目录与 DB

- `KnowledgeDir(appDir)`
- `KnowledgeDirForAgent(appDir, agentID)`
- `EnsureKnowledgeDir(appDir)`
- `EnsureKnowledgeDirForAgent(appDir, agentID)`
- `DBPath(appDir)`
- `OpenSharedDB(appDir)`
- `OpenExistingDB(appDir)`
- `CloseSharedDB()`

### 9.2 状态读写

- `GetLastUpdateForAgent(db, agentID)`
- `SetLastUpdateForAgent(db, agentID, ts)`
- `GetKnowledgeCommitForAgent(db, agentID)`
- `SetKnowledgeCommitForAgent(db, agentID, value)`

语义说明：

- `SetLastUpdateForAgent`
  - 只更新该 Agent 的 `last_update`
  - 不更新 `knowledge_commit`
  - 不接受负数时间戳

- `SetKnowledgeCommitForAgent`
  - 只更新该 Agent 的 `knowledge_commit`
  - 不表示本次一定已完成知识库更新时间回写

### 9.3 运行时读取

- `EnsureRuntimeForAgent(appDir, agentID)`
  - 创建目录
  - 初始化 sqlite/schema/默认行
  - 返回 Agent 级 `State`

- `LookupRuntimeForAgent(appDir, agentID)`
  - 只读探测
  - 如果目录不存在，返回 `nil`
  - 如果目录存在但 sqlite 不存在，返回 `lastUpdate=0, knowledgeCommit=false`

### 9.4 Metadata 组装

- `MetadataForAgent(appDir, agentID)`
- `MetadataIfExistsForAgent(appDir, agentID)`
- `MergeMetadata(base, appDir)`
- `MergeMetadataIfExists(base, appDir)`

这里的 `MergeMetadata(...)` 会从已有 `base["agentId"]` 中自动推断当前 Agent 维度。

## 10. CLI 设计

### 10.1 Knowledge 独立 CLI

当前 `knowledgecli` 支持：

- `ensure`
- `get`
- `metadata`
- `update-time`
- `update-commit`

公共参数：

- `--app-dir`
- `--agent-id`

说明：

- `get` 不是严格只读；它会走 `EnsureRuntimeForAgent(...)`
- 严格只读要使用：
  - `LookupRuntimeForAgent(...)`
  - `MetadataIfExistsForAgent(...)`
  - 或 CLI 的 `metadata --read-only`

### 10.2 Integration 收口 CLI

`integration` 对知识库有自己的 CLI 收口行为：

- `integration knowledge update-time --timestamp N --agentId ID`
- `integration knowledge last-update [--agentId ID]`

当前意图是：

- `update-time`
  - 明确要求同时传时间戳和 Agent
  - 用于更新指定 Agent 的知识库最后更新时间

- `last-update`
  - 用于读取指定 Agent（或默认兼容行）的格式化时间文本

## 11. HTTP 接口设计

### 11.1 `/knowledge`

由 `proxy` / `integration` 暴露：

- `GET /knowledge`
  - 返回知识库树形结构
- `GET /knowledge/<rel-path>`
  - 访问文件或目录

当前 `site` 会把它作为 WIKI 内容源使用。

### 11.2 `/knowledge_path`

支持：

- `GET /knowledge_path`
  - 返回知识库根目录
- `GET /knowledge_path?agentId=agent-a`
  - 返回 `--agent-dir/knowledge/agent-a`

用途：

- `site` 小灯泡整理浮层展示真实知识库目录
- 便于用户定位当前 Agent 的知识库磁盘位置

### 11.3 `/knowledge_lastUpdate`

支持：

- `GET /knowledge_lastUpdate`
  - 读取默认兼容记录
- `GET /knowledge_lastUpdate?agentId=agent-a`
  - 读取指定 Agent 的格式化最后更新时间

输出格式：

- `yyyy-MM-dd HH:mm`

用途：

- `site` 展示“最近刷新”
- `site` 判断是否超过 8 小时需要提醒整理

## 12. 前端设计细节

### 12.1 WIKI 面板状态

`site` 当前按 Agent 维度维护：

- 最近刷新时间文本
- 过期提示关闭状态
- 当前浏览文档
- 是否展开
- 自动整理开关本地状态

### 12.2 自动整理开关

前端本地存储形态是 Agent 级 map：

- `knowledge_disable_by_agent`

它会在发送消息时写到：

```json
{
  "metadata": {
    "knowledge": {
      "agentId": "当前Agent",
      "knowledge_disable": true
    }
  }
}
```

当前语义：

- 它是前端偏好，不是 `knowledgecore` 持久化字段
- 切换 Agent 时重新读取该 Agent 的本地开关状态
- 手动点小灯泡整理不受这个开关直接限制

### 12.3 手动整理请求

当用户通过 WIKI 小灯泡发起整理时，前端会发送：

- `requestSource = knowledge_tidy`
- 顶层 `metadata.knowledge_commit = true`

设计目的：

- 明确告诉后端这是一次显式知识库提交
- 强制保留本次请求里的 `knowledge.lastUpdate`
- 在流式响应完整结束后回写该 Agent 的 `last_update`

## 13. 后端请求处理设计

### 13.1 普通请求

普通 `/v1/chat/completions` 请求的知识库处理流程：

1. 读取顶层 `metadata.agentId`
2. 按 Agent 读取 `knowledge_runtime`
3. 把 `metadata.knowledge.path / lastUpdate / knowledgeCommit` 写成当前 Agent 的真实值
4. 若不是 `knowledge_commit=true`
   - 若 `now - lastUpdate <= knowledge_update_interval`
     - 删除本次转发中的 `knowledge.lastUpdate`
   - 否则继续检查 `knowledge_update_lock.last_requested_at`
   - 若仍在锁窗口内
     - 删除本次转发中的 `knowledge.lastUpdate`
   - 若已过锁窗口
     - 保留 `knowledge.lastUpdate`
     - 更新 `knowledge_update_lock.last_requested_at = now`

### 13.2 手动整理请求

如果请求显式带：

```json
{
  "metadata": {
    "knowledge_commit": true
  }
}
```

当前后端行为是：

1. 持久化该 Agent 的 `knowledge_runtime.knowledge_commit = true`
2. 处理 `metadata.knowledge` 时强制保留 `lastUpdate`
3. SSE 响应完整结束后：
   - `knowledge_runtime.last_update = now`
   - `knowledge_update_lock.last_requested_at = now`

### 13.3 为什么 `knowledgeCommit` 和 `lastUpdate` 要分开

这是当前设计里非常重要的分工：

- `knowledgeCommit`
  - 记录“用户是否显式把这次动作当成知识库提交”
  - 更像一个操作语义标记

- `lastUpdate`
  - 记录“知识库最后一次成功更新时间”
  - 更像一个事实状态

因此可能出现以下组合：

- `knowledgeCommit=false, lastUpdate=0`
  - 新 Agent，尚未发生知识库显式提交

- `knowledgeCommit=true, lastUpdate=0`
  - 已经收到显式提交请求，但这次流未完成或尚未完成回写

- `knowledgeCommit=true, lastUpdate>0`
  - 该 Agent 至少完成过一次显式知识库更新时间回写

## 14. 典型使用场景

### 场景 A：页面只是想展示当前 Agent 的知识库

适合读取：

- `/knowledge/<agentId>/index.md`
- `/knowledge_lastUpdate?agentId=<agentId>`
- `/knowledge_path?agentId=<agentId>`

此时不需要直接写 sqlite。

### 场景 B：服务端要给上游请求补知识库快照

适合调用：

- `MetadataForAgent(appDir, agentID)`
- 或在 `proxy / integration` 中执行 `processKnowledgeMetadata(...)`

此时会补齐：

- `knowledge.path`
- `knowledge.lastUpdate`
- `knowledge.knowledgeCommit`

### 场景 C：用户手动整理知识库完成

适合：

- 请求时带 `metadata.knowledge_commit=true`
- 响应完整结束后执行 `updateKnowledgeManualTimestamps(...)`

结果：

- `last_update` 更新为当前时间
- `knowledge_update_lock.last_requested_at` 同步更新

### 场景 D：只想记录“这次用户把提交开关设成了什么”

适合：

- 持久化顶层 `metadata.knowledge_commit`

结果：

- 只更新 `knowledge_runtime.knowledge_commit`
- 不代表 `last_update` 一定已经成功更新

## 15. 已知约束

- `knowledgecore` 只管理目录与运行时状态，不管理知识内容本身。
- `knowledge_disable` 目前不是 `knowledgecore` 的持久化字段，只是前端 Agent 级本地偏好加请求透传字段。
- 当前后端路由知识库维度时，真正依赖的是顶层 `metadata.agentId`；`metadata.knowledge.agentId` 不是权威字段。
- 普通请求里 `knowledge.lastUpdate` 当前不是每次都透传，仍受 `knowledge_update_interval` 与 `knowledge_update_lock` 双重约束。
- `knowledge_update_lock` 不在 `knowledge` 模块目录内实现，但当前知识库整体设计依赖它。
- `get` / `MarshalState` / `Metadata` 都属于“可能创建目录和 sqlite 状态”的创建型入口。

## 16. 测试关注点

当前知识库相关测试应至少覆盖：

- Agent 级目录解析与创建
- Agent 级 `last_update / knowledge_commit` 读写
- `MetadataForAgent(...)` 输出结构
- `MetadataIfExistsForAgent(...)` 的只读语义
- macOS 托管运行时目录切换
- `proxy / integration` 对 `metadata.knowledge` 的 Agent 级注入
- `/knowledge_lastUpdate?agentId=...`
- `/knowledge_path?agentId=...`
- 手动整理完成后的 `last_update + knowledge_update_lock` 联动回写

当前主要参考测试文件：

- [knowledgecore/core_test.go](/Users/shenjiawei/Documents/code/deepright/cli/module/knowledge/knowledgecore/core_test.go)
- [main_test.go](/Users/shenjiawei/Documents/code/deepright/cli/module/knowledge/main_test.go)
- [proxy/main_test.go](/Users/shenjiawei/Documents/code/deepright/cli/module/proxy/main_test.go)
- [integration/main_test.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/main_test.go)
