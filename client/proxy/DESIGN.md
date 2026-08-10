# Proxy 模块详细技术设计

## 设计定位

`proxy` 当前不是一个“只负责转发 `/v1/chat/completions` 的轻量反向代理”，而是一个独立可运行的本地控制面进程。它同时承担以下职责：

- OpenAI 兼容的流式聊天代理
- Agent 元数据聚合与请求注入
- 本地 Agent/工作区/文件/上传/目录打开接口
- 插件元数据、配置、启停、执行、日志流接口
- 内嵌 `connect` HTTP 服务与 CLI 收口
- 定时任务存储、执行、日志与 Connect pending request 桥接
- Token 配置、Token consume 记录与查询
- 会话级 Sandbox 状态中心
- 统一事件日志、聊天恢复、最近轮次导出与 skill 活跃判定
- Knowledge 目录映射与更新时间协调

当前权威实现集中在：

- `main.go`

这里没有再拆出独立的 `proxycore` 或服务层包；`main.go` 直接同时承载启动参数、HTTP 路由、SQLite schema、后台协程和大部分业务逻辑。

## 代码边界

### 主文件

- `main.go`
  - 模块绝大部分实现都在这里
  - 定义 `ProxyServer`
  - 管理 HTTP 路由、CLI 子命令、共享 sqlite、SSE 代理、cron 执行器、connect 桥接、文件接口、命令执行、knowledge 协调、event log 导出

### 辅助文件

- `plugins_local.go`
  - 只是对 `connectsvc` 本地插件扫描能力的薄封装
  - `proxy` 自己不维护另一套插件解析逻辑，主要通过 `connectsvc.ListLocalPluginMetaWithService(...)` / `connectsvc.ListLocalPlugins(...)` 获取结果

- `consume/consume.go`
  - 负责 `token_consume_log` 的 schema、写入、时间范围查询和按模型汇总
  - HTTP `/api/consume` 与 CLI `proxy token get` 都复用这里的查询能力

- `eventlog/eventlog.go`
  - 负责 `agent_message_log` 的异步写入与查询
  - 当前统一日志类型只有四种：
    - `0`：chat completion request
    - `1`：chat completion response
    - `2`：cli/get
    - `3`：cli/pub

### 外部依赖模块

`proxy` 当前强依赖以下共享模块：

- `agentcore`
  - Agent metadata 扫描、缓存、设备号生成、默认浏览器依赖检测、运行中插件检测

- `connectsvc`
  - 插件配置与 Connect 元数据/请求/回复存储
  - Connect HTTP handler
  - Connect CLI
  - pending request 决策与桥接辅助

- `knowledgecore`
  - `knowledge/` 路径与共享 sqlite 路径解析
  - knowledge `lastUpdate` 读写

- `sharedutil`
  - proxy client、metadata merge、布尔值标准化、本地请求校验、gzip/base64、knowledge lock 表、sandbox/path 等公共能力

- `sandboxstate`
  - 会话级 sandbox 状态读写

- `skillscore`
  - SKILL 解析告警扫描与持久化

## 总体架构

当前实现可以分成 6 层：

1. 启动与运行时层
   - 解析 `serve` 参数
   - 写入并读取主应用 `config/config.json`
   - 准备默认 Agent 模板

2. 共享状态层
   - 共享 sqlite `data`
   - 异步事件日志 `agent_message_log`
   - 连接表、活跃命令表、Agent metadata 缓存

3. 元数据聚合层
   - `agentcore.GetOutputForApp(...)`
   - selected agent metadata
   - model token 配置注入
   - knowledge lastUpdate 透传控制

4. HTTP 接入层
   - `http.ServeMux`
   - `/v1/chat/completions`
   - Connect/Plugin/File/Knowledge/Cron/Token/Sandbox/Restore 等接口

5. CLI 收口层
   - `proxy serve`
   - `proxy connect ...`
   - `proxy plugins ...`
   - `proxy cron ...`
   - 顶层兼容的 `create/find/delete/...`

6. 后台自动任务层
   - 每分钟 cron 执行器
   - 每 30 秒 connect pending request 同步器
   - 启动时与周期性的 skills warning 同步

## 启动模型与运行时配置

### 默认启动行为

`main()` 的入口规则是：

- 无子命令时，直接按 `serve` 处理
- `serve` 显式启动 HTTP 服务
- `connect`、`plugins`、`cron`、`log-skill`、`token`、`file-last-update`、`skills-warning` 走 CLI 分支
- 顶层 `create` / `submit` / `create-cron` / `submit-cron` / `find-meta` / `find-detail` / `delete-meta` / `delete-detail` 是 `cron` 的兼容快捷入口
- 如果第一个参数本身就是 Connect 子命令，也会被自动转给 `runProxyConnectCLI(...)`

### 端口约束

本地 HTTP 服务端口按以下优先级确定：

```text
--port > config/config.json.port > 8080
```

`validateProxyServicePort()` 会拒绝 `1` 至 `65535` 之外的端口。

### `config/config.json`

`proxy` 与 `integration` 一样，都会把启动期配置收口到当前启动目录下的：

```text
config/config.json
```

`runServe(...)` 启动时会把最终生效配置写回这个文件，字段包括：

- `app`
- `app-dir`
- `port`
- `host`
- `agent-dir`
- `default-dir`
- `device`
- `agent-cache`
- `connect-cache`
- `site`
- `connect_timeout`
- `knowledge_update_interval`
- `knowledge_update_lock`
- `install_app`
- `reply`
- `plugin_exec_timeout`

`proxyAppDir()` 会优先从 `config/config.json` 读取：

- `app-dir`
- 否则退回 `app` 所在目录
- 再退回当前工作目录

后续 knowledge 路径、共享 sqlite 路径都会基于这个 app dir 解析。

### Agent 根目录默认值

`resolveDefaultAgentDirArg()` 的优先级为：

1. 环境变量 `AGENT_DIR`
2. macOS 固定目录 `~/Library/Application Support/deepright/agent`
3. 当前目录 `agent`
4. 上级目录 `../agent`

这意味着 `proxy` 的 CLI 默认 Agent 根目录与 `integration` 当前实现并不完全相同。

### 默认 Agent 模板

`serve` 启动时会执行两步初始化：

- `ensureServeDefaultAgentScaffold(...)`
  - 如果 `agent-dir` 为空目录，就把 `default-dir` 整体复制成 `DEF_AGENT`
  - 同时确保 `DEF_AGENT/skills` 存在

- `syncDefaultAgentVersion(...)`
  - 如果 `default-dir/config.json` 含 `version`
  - 会把该版本号同步回 `agent-dir/DEF_AGENT/config.json`
  - 只更新 `version`，保留已有其它字段

## 共享状态与数据库设计

### 共享 sqlite

`getDataDB()` 使用包级单例 `pooledDB` 复用 sqlite 连接：

```go
type pooledDB struct {
    mu   sync.Mutex
    path string
    db   *sql.DB
}
```

当前连接池策略非常保守：

- `MaxOpenConns = 1`
- `MaxIdleConns = 1`
- `ConnMaxLifetime = 0`

这说明 `proxy` 仍按“单进程串行访问同一 sqlite 文件”的模型设计，而不是高并发 DB 层。

### sqlite 路径规则

`getDataDBPath()` 的优先级为：

1. `config/config.json["db"]`，如果手工存在
2. `knowledgecore.DBPath(proxyAppDir())`

默认情况下，`proxy` 与 knowledge 共享同一个 `data` sqlite。

### 异步事件日志

`getEventLogger()` 维护第二个包级单例，返回 `eventlog.Logger`：

- 同一路径复用同一份 logger
- buffer 默认 `512`
- 后台 goroutine 异步刷入 `agent_message_log`

注意：

- 聊天请求/响应会同时写 `chat_log` 与 `agent_message_log`
- `agent_message_log` 主要服务于 round log 导出、skill status 判断以及合并 `cli/get` / `cli/pub`

### 本模块实际使用的主要表

`proxy` 当前把大量运行时状态都收进同一个 sqlite 文件。主要表包括：

- `task_meta`
- `task_detail`
- `cron_meta_log`
- `cron_detail_log`
- `chat_log`
- `cmd_log`
- `kill_log`
- `token_store`
- `proxy_agent_provider_log`
- `token_consume_log`
- `agent_message_log`
- `knowledge_update_lock`
- `cli_sandbox_state`（由 `sandboxstate` 维护）
- Connect 相关表（由 `connectsvc` 维护）
- skills warning 表（由 `skillscore` 维护）

其中：

- `ensureCronSchema(db)` 负责 cron 相关 schema 以及一些迁移列补齐
- 其它表大多按功能第一次访问时懒初始化

## `ProxyServer` 与运行时对象

`ProxyServer` 不是纯配置对象，而是 HTTP handler 聚合中心。核心字段包括：

- `Host`
- `AgentDir`
- `DefaultDir`
- `DeviceID`
- `CacheTTL`
- `SiteDir`
- `ConnectTimeout`
- `KnowledgeUpdateInterval`
- `KnowledgeUpdateLock`
- `Client`
- `ConnectCacheTTL`
- `AutoReply`
- `ConnectService`
- `WarningScanRoot`
- `PluginExecTimeout`

它依赖两类共享运行时状态：

- `connMap`
  - 管理活跃 SSE 连接
  - key 为 `agentId + chatId`
  - `/api/cancel` 通过这里取消正在进行的聊天转发

- `activeCmdMap`
  - 管理活跃 shell / sandbox 命令
  - `/api/kill` 通过这里终止命令

## Agent 元数据聚合设计

### 共享 metadata 来源

`proxy` 不自己重新实现 Agent 扫描，而是复用：

- `agentcore.GetOutputForApp(root, ".", deviceID, ttl)`
- `agentcore.GetOutputForAppAndChat(root, ".", deviceID, ttl, chatID)`

注意这里第二个参数当前固定是 `"."`，表示 app dir 一律取当前进程工作目录。

### 额外 hydration

在拿到 `AgentOutput` 后，`proxy` 会做两层补充：

1. `hydrateAgentOutput(...)`
   - 为每个 Agent 补充当前 `chatId` 下的 `sandbox`

2. `cloned.Plugins = detectPluginKeys()`
   - 把当前运行中的插件 key 列表写回 metadata

### `selected agent` 片段

当请求 metadata 里能确定 `agentId` 时，`proxy` 会额外补一段：

```json
{
  "agent": {
    "agentId": "...",
    "version": "...",
    "sandbox": "..."
  }
}
```

这段数据来自当前选中 Agent 的完整 metadata，而不是单独查询另一张表。

## 聊天代理与 SSE 转发

### 请求标准化流程

`HandleChatCompletions(...)` 的核心流程如下：

1. 读取客户端原始 JSON body
2. 提取请求里的 `metadata.chat`
3. 基于 chat 上下文获取共享 Agent metadata
4. 把共享 metadata 与请求自带 `metadata` 合并
5. 若当前 `model` 在 `token_store` 中有扩展配置，则注入：
   - `__url`
   - `__model`
   - `__model_fast`
   - `__model_thinking`
   - `__model_multi_input`
   - `__model_multi_output`
6. 处理 knowledge metadata 的 `lastUpdate` 透传策略
7. 把顶层布尔字段归并到 `metadata`：
   - `thinking`
   - `html`
   - `router_disable`
8. 若外部仍传简化 `message`，归一成单条 `messages`
9. 为选中 Agent 补 `metadata.agent`
10. 转发到上游 `/v1/chat/completions`

### 布尔字段收口

`proxy` 当前会主动清理旧的顶层布尔字段：

- `thinking`
- `html`
- `router_disable`

这些字段最终只保留在 `metadata` 内，不继续以顶层字段形式发给上游。

### 同 chat 的连接互斥

若同一个 `agentId + chatId` 已有活跃 SSE 连接，新请求进入时会：

- 先取消旧连接
- 再注册新连接

因此 `proxy` 的聊天流默认是“每个 Agent 会话只保留一个当前活跃上游连接”。

### 日志写入方式

转发过程中：

- 归一化后的请求体作为 `Q` 写入 `chat_log`
- SSE 响应按完整 event chunk 逐段写入 `chat_log`
- 同时把 request/response 镜像写入 `agent_message_log`

这保证了：

- `/api/restore` 可以回放原始会话片段
- `/log_skill` 可以基于统一事件日志导出最近轮次

### `/api/cancel`

`POST /api/cancel?agentId=...&chat=...` 会：

- 向活跃连接写入一个取消标记 `X`
- 关闭日志 channel
- 调用连接的 `cancel()`
- 从 `connMap` 移除该连接

它只接受查询参数 `chat`，不是 `chatId`。

## Knowledge 协调设计

### knowledge 接口

`proxy` 当前暴露 3 个 knowledge 相关接口：

- `GET /knowledge`
- `GET /knowledge_lastUpdate`
- `GET /knowledge_path`

实现特点：

- `/knowledge`
  - 当前 app dir 下的 `knowledge/` 作为根目录
  - 访问目录时返回文本树
  - 访问文件时直接 `http.ServeFile(...)`
  - 使用 `knowledgeRelativePath(...)` 与 `ensurePathWithinRoot(...)` 防止路径逃逸

- `/knowledge_lastUpdate`
  - 直接读 knowledge shared db 的 `lastUpdate`
  - 返回本地时区格式化字符串 `YYYY-MM-DD HH:MM`

- `/knowledge_path`
  - 返回 knowledge 根目录绝对路径

### `knowledge.lastUpdate` 透传策略

`processKnowledgeMetadata(...)` 的逻辑是当前模块非常重要的一层协议控制：

- 若 metadata 里没有 `knowledge.lastUpdate`，不处理
- 若距离当前时间未超过 `KnowledgeUpdateInterval`，删除 `lastUpdate`
- 若已过期，则进一步检查 `knowledge_update_lock`
- 如果最近一次申请时间未超过 `KnowledgeUpdateLock`，仍删除 `lastUpdate`
- 只有真正过期且最近没有其它请求申请刷新时，才把 `lastUpdate` 原样转发给上游

默认值：

- `KnowledgeUpdateInterval = 2h`
- `KnowledgeUpdateLock = 30min`

### `knowledge_commit`

若请求 metadata 中显式携带：

```json
{
  "knowledge_commit": true
}
```

则：

- 这次请求会强制保留 `knowledge.lastUpdate`
- 等 SSE 完整结束后，`updateKnowledgeManualTimestamps(...)` 会把当前时间写回：
  - knowledge `lastUpdate`
  - `knowledge_update_lock.last_requested_at`

## 插件与 Connect 设计

### Connect Service 的持有方式

`runServe(...)` 启动时会预先创建一个：

```go
connectsvc.NewService(connectsvc.Options{
    DBPath:   "data",
    AgentDir: opts.agentDir,
    CacheTTL: ...
})
```

并把它挂到 `ProxyServer.ConnectService`。

后续 HTTP handler 优先复用这份 service；若没有注入，再临时创建短生命周期 service。

### 内嵌 Connect HTTP 服务

`runServe(...)` 通过 `connectsvc.NewDynamicHTTPHandler(...)` 注册：

- `/api/connect/meta`
- `/api/connect/request`
- `/api/connect/response`

另外暴露一个轻量健康检查：

- `/api/connect/health`

也就是说 `proxy` 直接把 Connect 模块内嵌进自身 HTTP 服务，而不是额外拉起独立进程。

### 插件接口

当前插件相关 HTTP 接口包括：

- `GET /api/plugins/meta`
- `GET /api/plugins/status`
- `POST /api/plugins/config`
- `POST /api/plugins/start`
- `POST /api/plugins/stop`
- `GET /api/plugins/exec`
- `GET /api/plugins/log`

实现要点如下：

- `/api/plugins/meta`
  - 必须实时反映本地插件定义和已保存 meta
  - 因此不会依赖长缓存扫描结果
  - 若没有共享 `ConnectService`，会临时创建 `CacheTTL = 1ms` 的 service

- `/api/plugins/config`
  - 通过 `connectsvc.UpsertPluginConfigWithService(...)` 落库
  - `router_disable` 等配置直接写到 Connect meta

- `/api/plugins/start` / `/api/plugins/stop`
  - 通过 `connectsvc.RunPluginAction(...)` 执行
  - 若未显式给出 `connect-bin`，会自动把当前 `proxy` 可执行文件路径补进去

- `/api/plugins/exec`
  - 支持多级插件子命令
  - 超时时间来自 `PluginExecTimeout`

- `/api/plugins/log`
  - 通过 `pluginlog.ServeSSE(...)` 直接把插件日志以 SSE 方式输出

### Connect pending request 桥接

`startConnectPendingRequestSync(...)` 会在后台每 30 秒执行一次：

- `syncConnectPendingRequests(p)`

桥接逻辑是当前 `proxy` 的一个核心自动化流程：

1. 从 Connect 元数据里找出已配置 callback 的插件
2. 读取尚未消费的 pending request
3. 基于 `connectsvc.DecidePendingPluginPlan(...)` 决定：
   - 是否创建一次性 cron task
   - 是否立即 notify plugin started
   - 是否分发 completed reply
4. 将待处理请求转成 `task_meta + task_detail`
5. 对某些立即执行的 detail，直接调用 `runDueTask(...)`

当前策略里有两个重要时间值：

- 超过 20 分钟的 pending request 会按过期策略处理
- 同步周期固定为 30 秒

## Cron 设计

### 数据模型

`ensureCronSchema(db)` 负责维护：

- `task_meta`
- `task_detail`
- `cron_meta_log`
- `cron_detail_log`

并补齐当前实现已经使用到的列，例如：

- `task_type`
- `router_disable`
- `response_schema`

### HTTP 接口

当前暴露的 cron HTTP 接口包括：

- `POST /api/cron/create`
- `GET /api/cron/detail/metadata`
- `GET /api/cron/delete`
- `GET /api/cron/detail/delete`
- `GET /api/cron/detail/list`
- `GET /api/cron/detail/status`

### CLI 收口

CLI 同时支持：

- `proxy cron create`
- `proxy cron create-cron`
- `proxy cron find-meta`
- `proxy cron find-detail`
- `proxy cron delete-meta`
- `proxy cron delete-detail`

以及顶层兼容别名：

- `proxy create`
- `proxy submit`
- `proxy create-cron`
- `proxy submit-cron`
- `proxy find-meta`
- `proxy find-detail`
- `proxy delete-meta`
- `proxy delete-detail`

### 执行器

`startCronExecutor(...)` 每分钟执行一次：

1. `syncSkillsWarnings(p)`
2. `cronExecuteOnce(p)`

`cronExecuteOnce(...)` 会查询最近 1 小时内到期且 `started = 0` 的 `task_detail`，再逐条执行 `runDueTask(...)`。

### `runDueTask(...)` 的转发协议

执行 detail 时，`proxy` 会构造一个新的 `/v1/chat/completions` 请求，并注入：

- `agentId`
- `chat`
- `type = scheduled_task`
- `cron_type`
- `thinking`
- `router_disable`
- `agent`
- `router_remote`
- `META_ID`
- `response_schema`

其中：

- 若 detail 没有 `chat_id`，会自动生成 `${metaID}@${detailID}`
- 若 Agent 已删除，直接记日志并跳过
- 若模型未注册或 token 为空，直接记日志并跳过
- 若 `response_schema` 不为空，还会组装 `response_format.json_schema`

执行结果：

- 请求与响应都会写入 `chat_log`
- 完成后把 `task_detail.started` 更新为 `3`
- 并把抽取出的 SSE 文本写入 `result_content`

## Token、Consume 与 Provider 配置

### `token_store`

`HandleToken(...)` 与 `runProxyTokenCLI(...)` 共用同一套 token 存储：

- `GET /api/token`
  - 返回当前全部模型配置

- `POST /api/token`
  - 写入或删除模型配置
  - 自动做 model 名标准化
  - 自动清理旧别名记录

除 token 外，单模型还可带 provider 扩展元数据：

- `__url`
- `__model`
- `__model_fast`
- `__model_thinking`
- `__model_multi_input`
- `__model_multi_output`

这些字段不会直接走独立 API，而是作为 `token_store` 配置的一部分存在，并在聊天代理阶段注入 metadata。

### provider 审计日志

token 配置修改时，`writeTokenStoreChanges(...)` 还会写入：

- `proxy_agent_provider_log`

用于记录模型配置的更新与删除。

### Token consume

`proxy` 同时支持 token consume 的记录与查询：

- CLI
  - `proxy token --record ...`
  - `proxy token get ...`
  - `proxy token --n ...` 也可直接进入查询模式

- HTTP
  - `GET /api/consume`

`/api/consume` 当前要求：

- `starTime`
- `closeTime`

时间格式固定为：

```text
yyyyMMdd-hhmmss
```

查询结果既返回原始明细，也返回按 model 汇总后的 summary。

## Sandbox 设计

### HTTP 侧

`proxy` 暴露两组 sandbox 接口：

- 写入：
  - `/api/sandbox=off`
  - `/api/sandbox=filepick`
  - `/api/sandbox=net`
  - `/api/sandbox=filepick_net`

- 只读：
  - `/api/sandbox_status`

两者都要求：

- `agentId`
- `chatId` 或 `chat`

写入后状态落到共享 sqlite 的 `cli_sandbox_state`。

### 命令执行联动

`HandleCmd(...)` 执行命令前会先查当前会话的 sandbox mode：

- 若为空，直接走本地 shell
- 若非空，调用外部 sandbox helper 执行

sandbox helper 路径来源于 `config/config.json` 里的：

- `sandbox_app`

当前会按不同 mode 组合多个候选路径进行探测。

### CLI 现状

代码里已经实现了：

- `runProxySandboxCLI(...)`

但 `main()` 当前并没有把 `sandbox` 挂成正式顶层子命令。因此当前状态是：

- 帮助文案里已经出现 `proxy sandbox ...`
- HTTP 也已完整可用
- 但正式 CLI 入口尚未在 `main()` 中接线

这是当前实现与文档/帮助文案不完全一致的一处现状。

## 文件、工作区与 Agent 管理

### Agent 基础接口

当前接口包括：

- `GET /api/agentId`
- `GET /api/swarm_agent`
- `GET /api/deviceId`
- `GET /api/agent/init`
- `GET /api/agent/delete`
- `GET /api/agent/create`

实现特点：

- `/api/swarm_agent`
  - 实际上是返回 `router_disable = false` 的 Agent ID 列表
  - 并可通过 `agentId` 查询参数排除当前 Agent

- `/api/agent/init`
  - 从 `default-dir` 复制一整份 Agent 模板

- `/api/agent/delete`
  - 删除 Agent 目录
  - 同时清理该 Agent 的 cron 数据

- `/api/agent/create`
  - 在 Agent workspace 下创建目录或空文件
  - 支持多层相对路径
  - 明确禁止路径逃逸

### 文件读取与写入

`proxy` 当前文件接口并不完全一致，分成两类：

- 受 workspace 约束的接口
  - `POST /api/edit`
  - `GET /api/del`
  - 相对路径必须落在对应 Agent workspace 下

- 可接受绝对路径的接口
  - `GET /api/files`
  - `GET /api/data`
  - `GET /api/raw`
  - `GET /api/folder?path=...`

其中：

- `/api/edit`
  - 文本文件直接写入
  - 二进制后缀文件要求 body 里的 `content` 为 base64
  - `saveAsNew=true` 时会自动追加时间戳另存

- `/api/raw`
  - 返回文件内容的 base64
  - 相对路径时受 workspace 约束
  - 绝对路径时可直接读取系统路径

- `/api/data`
  - 用于读取文本文件
  - 明确拒绝目录与二进制后缀
  - 当前实现支持任意绝对路径大小写不敏感解析，不要求 `agentId`

- `/api/files`
  - 支持目录列举或按前缀列举同级条目

- `/api/workspace`
  - 返回指定 Agent workspace 路径

- `/api/folder`
  - 可打开 Agent workspace
  - 也可直接打开绝对路径目录

### 上传与下载

当前还提供：

- `POST /api/upload`
- `GET /api/download`

前者把上传内容写到 Agent `tmp` 目录，后者支持文件或目录下载。

## 本地命令执行设计

### `/api/cmd`

`HandleCmd(...)` 只允许 localhost 请求执行命令。执行前会依次校验：

- 请求方法必须为 `POST`
- 远端必须是本地请求
- `agentId` / `chatId` / `cmd` 必填
- 命令不能包含被阻断 token
- Agent 必须存在

命令执行流程：

1. 生成或接收 `tid`
2. 写入 `cmd_log`
3. 根据会话 sandbox 状态选择执行器
4. 把活跃命令注册到 `activeCmdMap`
5. 完成后把输出 gzip+base64 写回 `cmd_log.result`

### `/api/kill`

`HandleKill(...)` 也只允许 localhost 调用。它会：

- 按 `agentId + chatId + tid + cmd` 查活跃命令
- 如果找不到，退化为“同 chat 下唯一活跃命令”的兜底匹配
- 成功后调用对应 cancel
- 并记录 `kill_log`

## 聊天恢复、统一日志与最近轮次导出

### `/api/restore`

`HandleRestore(...)` 从两类来源拼接结果：

1. `chat_log`
2. `agent_message_log` 中的 `cli/get` / `cli/pub`

这样页面恢复不仅能拿到普通 Q/A，还能拿到同一会话里的 CLI 活动记录。

### `agent_message_log`

统一日志服务于三个能力：

- 最近轮次导出
- skill status 检查
- restore 时补齐 CLI 活动

其中 `eventlog.NormalizeContent(...)` 会对 `cli/pub` 内容做规范化，尽量把真实输出恢复成可读文本。

### `/log_skill` 与 `/log_skill_status`

- `GET /log_skill`
  - 读取统一日志
  - 根据最近 N 轮 request 边界和可选 `start` / `close` 过滤
  - 合并同一轮里的 SSE response 分片
  - 导出 Markdown 到 Agent workspace 的 `tmp/`

- `GET /log_skill_status`
  - 在同一轮次定义下判断是否存在 `cli/get` 或 `cli/pub`

CLI `proxy log-skill ...` 直接复用同一导出函数 `exportRoundLog(...)`。

## Skills 与 install_app

### `/api/skills`

`HandleSkills(...)` 返回指定 Agent 的技能名列表，但返回值并不是单纯的 Agent 本地 `skills/`：

- 先取 Agent metadata 内已有 skills
- 再叠加 `config/config.json` 中声明的 app 级技能
- 若 `browser` 或 `remote` 插件当前处于启动状态，还会额外注入内部 skill：
  - `__internal_browser`
  - `__internal_remote`

因此这是“运行时技能视图”，不是纯文件系统枚举。

### `/skills_warning`

`HandleSkillsWarning(...)` 读取共享 sqlite 中的 SKILL 解析告警。

若传 `refresh=1`，会先执行扫描同步：

- 优先使用 `ProxyServer.WarningScanRoot`
- 否则默认扫描 `AgentDir/skills`
- 再否则退回当前目录 `skills`

### `/install_app`

`HandleInstallApp(...)` 的返回值由两部分合并而来：

1. 代码静态探测的必需依赖
2. `config/config.json` 里的 `install_app` 当前平台数组

应用清单不支持 CLI 追加，Proxy 启动也不会写回或覆盖 `install_app` 配置对象；然后再过滤掉当前机器已经安装的项。

## HTTP 路由收口

`runServe(...)` 当前注册的核心路由分组如下：

- Chat
  - `/v1/chat/completions`
  - `/api/cancel`
  - `/api/restore`

- Connect / Plugin
  - `/api/connect/*`
  - `/api/plugins/*`

- Agent / Workspace / File
  - `/api/agentId`
  - `/api/swarm_agent`
  - `/api/deviceId`
  - `/api/agent/init`
  - `/api/agent/delete`
  - `/api/agent/create`
  - `/api/files`
  - `/api/data`
  - `/api/workspace`
  - `/api/edit`
  - `/api/del`
  - `/api/raw`
  - `/api/upload`
  - `/api/download`
  - `/api/folder`
  - `/file/lastUpdate`

- Knowledge / Skill
  - `/knowledge`
  - `/knowledge_lastUpdate`
  - `/knowledge_path`
  - `/api/skills`
  - `/skills_warning`
  - `/log_skill`
  - `/log_skill_status`

- Runtime / Command / Sandbox
  - `/api/config`
  - `/api/cmd`
  - `/api/kill`
  - `/api/sandbox=*`
  - `/api/sandbox_status`

- Token / Consume / Cron
  - `/api/token`
  - `/api/consume`
  - `/api/cron/*`

此外还会调用 `server.Register(mux, site)` 挂载静态站点。

## 测试体现出的当前实现事实

从 `main_test.go`、`acceptance_test.go`、`eventlog/eventlog_test.go` 可以确认以下行为已经被测试覆盖：

- `/v1/chat/completions` 会把顶层布尔字段只保留到 `metadata`
- 共享 Agent metadata 会随 skills、provider、sandbox、selected agent version 实时更新
- `knowledge_commit` 会在 SSE 完整结束后回写 knowledge 时间戳
- `/api/plugins/meta` 会立即反映本地插件变化
- plugin config 会持久化 `router_disable`
- `/api/plugins/log` 走 SSE 流
- `cmd_log.result` 以 gzip+base64 保存
- `/api/restore` 会把 `cli/get` 与 `cli/pub` 合并返回
- `log_skill` 以最近 request 轮次为边界导出 Markdown
- `skills_warning` 支持 refresh 扫描
- cron 执行会注入 `META_ID`、`cron_type`、`response_schema`、detail 级 `router_disable`
- connect pending request 会桥接成即时 cron detail，并在可行时直接执行
- 服务端口校验会拒绝 `1` 至 `65535` 之外的值

## 当前实现约束

当前 `proxy` 有几条在设计上必须明确记录的现实约束：

- `main.go` 体量非常大，HTTP、CLI、schema、后台任务高度耦合，后续维护成本较高
- `proxy` 与 `integration` 已统一使用主应用 `config/config.json` 记录运行态启动配置
- `serve` 的 `--port` 参数优先于 `config/config.json.port`，未配置时回退至 `8080`
- 帮助文案已暴露 `proxy sandbox ...`，但 `main()` 尚未把它注册成正式顶层 CLI 子命令
- 文件接口的安全边界并不完全统一：`/api/edit`、`/api/del` 严格限制在 workspace，下游的 `/api/data`、`/api/files`、`/api/raw`、`/api/folder?path=...` 则仍允许绝对路径能力
- 许多实现细节直接依赖共享 sqlite 同文件协作，而不是明确的服务边界；这让它与 `knowledge`、`connect`、`agentcore` 的运行时耦合较深

## 总结

当前 `proxy` 的真实角色更接近“桌面本地控制面 + SSE 代理 + 运行时状态中心”，而不是单一网络代理。它的设计重点不在协议纯转发，而在于：

- 把 Agent metadata、knowledge、token、sandbox、plugin、connect、cron 等运行时状态统一收口到本地 HTTP/CLI
- 用共享 sqlite 作为跨能力协作总线
- 用后台同步器把 Connect pending request 与 cron 执行链路粘合起来

因此后续若继续演进 `proxy`，最优先的设计工作通常不是再加 handler，而是拆分 `main.go` 中已经稳定的几块核心能力边界。
