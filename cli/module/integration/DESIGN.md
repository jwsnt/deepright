# Integration 模块详细技术设计

## 设计定位

`integration` 已经不是“把几个 HTTP handler 拼到一起”的轻量壳层，而是当前 CLI 体系对外的统一主程序。它同时承担以下职责：

- 统一 HTTP 服务入口
- `start/stop/restart/serve` 生命周期管理
- `connect` CLI 与插件管理 CLI 的顶层收口
- `cli-get` 心跳拉取与命令执行
- 定时任务存储、补齐、执行、回推插件回复
- Token、Sandbox、Round Log、Knowledge 状态中心
- 桌面 bundle 运行时目录迁移、插件同步、浏览器唤醒、启动 Splash

当前实现的权威核心在 [main.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/main.go)，并没有再拆出单独的 `integrationcore` 共享包。

## 代码边界

### 主文件

- [main.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/main.go)
  - 模块绝大部分核心逻辑都在这里
  - 包括启动参数、HTTP 路由、Token/Sandbox/Cron/Connect 桥接、文件接口、日志导出、Agent 管理、生命周期控制

### 辅助文件

- [browseropen.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/browseropen.go)
  - 跨平台浏览器探测与最大化打开
  - macOS 下优先尝试恢复现有 Chrome 标签页
  - Linux / WSL / Windows 下有不同的浏览器探测顺序

- [plugin_runtime_sync.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/plugin_runtime_sync.go)
  - bundle 运行时目录准备
  - 历史 sqlite 迁移到运行时目录
  - 内置插件同步到运行时插件目录
  - `integration plugins sync-bundled` CLI

- [plugins_local.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/plugins_local.go)
  - 解析本地插件目录
  - 本地插件列表与 metadata 扫描入口

### 子包

- [agentarchive/agentarchive.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/agentarchive/agentarchive.go)
  - Agent 导出 zip、zip 导入、目录导入

- [knowledge/lastupdate.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/knowledge/lastupdate.go)
  - 共享 knowledge 最后更新时间格式化

- [runtimehost/runtimehost.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/runtimehost/runtimehost.go)
  - 运行时 host 覆盖状态与 HTTP client

- [standalone/standalone.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/standalone/standalone.go)
  - 运行时 standalone 状态与 HTTP client

- [http11client/http11client.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/http11client/http11client.go)
  - 强制 HTTP/1.1 的 client / transport

- [launchsplash/launchsplash.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/launchsplash/launchsplash.go)
  - macOS bundle 启动 splash

## 总体架构

### 单进程多职责收口

当前架构不是多个独立子服务进程互相 RPC，而是一个主进程直接持有：

- HTTP Server
- `connectsvc.Service`
- 全局 sqlite 连接 `cronDB`
- `cli-get` worker pool
- cron 补齐 goroutine
- cron 执行 goroutine
- connect pending request 同步 goroutine
- 活跃 SSE 连接表
- 活跃命令表

### 分层关系

可以把当前实现理解为 6 层：

1. 启动与运行时目录层
   - bundle / runtime 路径解析
   - 启动配置读写
   - PID / 日志 / 浏览器 / splash

2. HTTP 接入层
   - `http.ServeMux`
   - 本地访问校验
   - standalone 保护包装

3. 统一状态层
   - sqlite `data`
   - token / cron / chat_log / cmd_log / kill_log / sandbox state

4. 元数据聚合层
   - Agent metadata
   - Knowledge metadata
   - 运行时 plugin skills
   - selected agent version / sandbox / model metadata

5. 自动任务层
   - `cli-get`
   - cron detail 补齐
   - cron detail 执行
   - connect pending request -> cron task 桥接

6. 本地桌面能力层
   - 浏览器打开
   - 本地命令执行
   - sandbox helper
   - 文件系统与 Agent 导入导出

## 运行时目录与配置模型

### 资源目录与运行时目录

当前实现区分两类路径：

- 资源目录
  - 普通二进制下通常是 `integration` 可执行文件所在目录
  - macOS `.app` 下是 `Contents/Resources`

- 运行时目录
  - 用于持久化 `data`、运行时插件、副本配置
  - 在 bundle 场景下会被固定迁移到应用外部

### 托管运行时根目录

`integrationManagedRuntimeBaseDir()` 的当前规则：

- macOS
  - `~/Library/Application Support/deepright`

- WSL
  - `~/deepright`

- 普通 Linux / Windows 非 WSL
  - 不启用固定托管运行时根目录

### 插件目录

- 资源内置插件目录
  - `<resources>/plugins`

- 运行时插件目录
  - `<runtime>/plugins`
  - 或环境变量 `DEEPRIGHT_PLUGIN_DIR`

`prepareIntegrationRuntimeLayout()` 会在启动前：

1. 准备运行时根目录
2. 必要时迁移历史 `data`
3. 将内置插件同步到运行时插件目录
4. 导出 `DEEPRIGHT_PLUGIN_DIR`

### 配置文件路径

当前实现统一把主应用配置与运行时记录写到：

```text
<resources>/config/config.json
```

而不是旧版方案中的独立启动配置文件。

这是当前实现里非常重要的事实：

- `config/config.json` 同时承担“启动默认配置”和“当前已解析运行时路径记录”两种职责
- `writeRuntimeConfig()` 会在 `serve` 启动阶段把最终生效配置重新写回这个文件
- `readRuntimeConfig()` / `readIntegrationRuntimeConfigRaw()` 本质也是在读同一个文件

### 配置读取优先级

当前优先级是：

1. CLI 显式参数
2. `config/config.json`
3. 内置默认值

### 支持的配置键

`readIntegrationStartupConfig()` 会把多种命名风格归一化，例如：

- `agentDir`
- `agent_dir`
- `agent-dir`

最终统一成内部 canonical key。

HTTP 相关配置当前只从嵌套 `http` 节读取：

```json
{
  "http": {
    "http_timeout": 45000,
    "http_connect_timeout": 15000,
    "http_socket_timeout": 45000,
    "debug": false
  }
}
```

旧的平铺 `http_timeout` / `http_debug` 不再作为启动配置读取来源。

### 路径重定位

因为 `config/config.json` 里会记录：

- `app`
- `app-dir`
- `resources-dir`
- `db`

所以 `integrationResolveStoredPath()` 能在应用被移动位置后，把旧记录的相对资源路径重新 rebasing 到新路径下。

这也是 bundle 交付场景里配置可迁移的关键机制。

## 启动参数与默认值

`Config` 结构体定义在 [main.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/main.go) 中，核心字段包括：

- 共享字段
  - `Port`
  - `Host`
  - `AgentDir`
  - `DefaultDir`
  - `Device`
  - `AgentCacheMs`
  - `ConnectCacheMs`
  - `Site`

- Proxy / Connect 相关
  - `ConnectTimeoutMs`
  - `KnowledgeUpdateIntervalMs`
  - `KnowledgeUpdateLockMs`
  - `InstallApp`
  - `Reply`

- cli-get 相关
  - `SleepMs`
  - `Thread`
  - `HTTPTimeout`
  - `HTTPConnTimeout`
  - `HTTPSocketTimeout`
  - `HTTPDebug`
  - `IdleTimeout`
  - `PluginExecTimeout`

当前默认值：

- `Port = 8080`
- `Host = https://www.deepright.cn`
- `AgentCacheMs = 120000`
- `ConnectCacheMs = 10000`
- `ConnectTimeoutMs = 15000`
- `KnowledgeUpdateIntervalMs = 7200000`
- `KnowledgeUpdateLockMs = 1800000`
- `SleepMs = 3000`
- `Thread = 3`
- `HTTPTimeout = 45000`
- `HTTPConnTimeout = 15000`
- `HTTPSocketTimeout = 45000`
- `IdleTimeout = 90`
- `PluginExecTimeout = 600000`

### 端口约束

当前实现不是“任意端口可配置”，而是强制：

```text
--port 只允许 8080
```

`validateIntegrationServicePort()` 会直接拒绝其他端口。

## 生命周期设计

### 命令分发

`main()` 当前支持：

- `start`
- `stop`
- `restart`
- `serve`
- `splash`
- `cron`
- `knowledge`
- `host`
- `standalone`
- `agent`
- `sandbox`
- `token`
- `connect`
- `plugins`
- `log-round`
- `skills-warning`
- `file-last-update`

另外，如果第一个参数直接是 connect 子命令，例如：

```bash
integration meta-list
integration add-request
```

当前也会直接转发到 `runIntegrationConnectCLI()`，作为兼容入口保留。

### `serve`

`runIntegrationForeground()` 是前台模式的真实服务启动入口：

1. 读取 `config/config.json`
2. 绑定 flag，并让 CLI 覆盖配置默认值
3. 校验端口只能是 `8080`
4. 解析 `default-dir` / `agent-dir` / `site`
5. 初始化 knowledge runtime
6. 把最终配置写回 `config/config.json`
7. 创建 upstream `proxyClient`
8. 创建 `connectsvc.Service`
9. 注册 HTTP 路由
10. 初始化 `cronDB`
11. 启动 3 个后台 goroutine
    - `startCliGet()`
    - `startCronCheck()`
    - `startCronExecutor()`
    - `startConnectPendingRequestSync()`
12. 写 PID 文件
13. 按需异步打开浏览器
14. 启动 HTTP server

### `start`

`startIntegrationProcess()` 的当前行为：

1. 解析顶层 flags
2. 清理启动目录中残留的 `*.pid`
3. 如果已有运行进程且服务 ready：
   - 直接复用
   - 尝试打开浏览器
4. 否则后台拉起 `integration serve`
5. 轮询：
   - `startup status file`
   - PID 文件
   - `/api/heartbeat` 就绪状态
6. 只有在 HTTP 真 ready 后才返回成功

生命周期日志只写 `integration.log`，不会把内部细节大量回显到 stdout。

### `stop`

`stopIntegrationProcess()` 的收口策略是“尽力关闭”：

1. 根据 PID 文件找进程
2. 先停止已配置且已启动的插件
3. 即使插件 stop 出错，也继续关闭 integration 主进程
4. 先发 `SIGTERM`
5. 超时后发 `SIGKILL`
6. 清理 PID 文件与残留 `*.pid`

### `restart`

当前实现就是：

1. `stop`
2. `start`

没有单独的热更新逻辑。

### `/api/shutdown`

HTTP 关闭接口不是立即退出，而是通过 `integrationShutdownController` 只调度一次延迟关闭：

- 先触发插件 stop
- 再 cancel 主上下文
- 最终走和 `integration stop` 相同的资源回收路径

## HTTP 路由总览

`serve` 当前注册的核心路由包括：

### 对话与自动执行

- `POST /v1/chat/completions`
- `GET /api/heartbeat`
- `POST /api/cancel`
- `POST /api/cmd`
- `POST /api/kill`

### Plugin / Connect

- `GET /api/plugins/meta`
- `GET /api/plugins/status`
- `POST /api/plugins/config`
- `POST /api/plugins/start`
- `POST /api/plugins/stop`
- `GET /api/plugins/exec`
- `GET /api/plugins/log`
- `/api/connect/meta`
- `/api/connect/request`
- `/api/connect/response`

### Cron

- `POST /api/cron/create`
- `POST /api/cron/detail/metadata`
- `POST /api/cron/detail/list`
- `POST /api/cron/delete`
- `POST /api/cron/detail/delete`
- `POST /api/cron/detail/status`

### Token / Config / Sandbox

- `POST /api/config`
- `GET/POST /api/token`
- `GET /api/consume`
- `GET/POST /api/sandbox`
- `GET/POST /api/sandbox=off`
- `GET/POST /api/sandbox=filepick`
- `GET/POST /api/sandbox=net`
- `GET/POST /api/sandbox=filepick_net`
- `GET /api/sandbox_status`

### Agent 与文件

- `GET /api/agentId`
- `GET /api/swarm_agent`
- `GET /api/workspace`
- `GET /api/files`
- `GET /api/data`
- `POST /api/edit`
- `GET /api/del`
- `GET /api/raw`
- `GET /file/lastUpdate`
- `GET /api/folder`
- `GET /api/agent/init`
- `GET /api/agent/delete`
- `GET /api/agent/export`
- `POST /api/agent/import`
- `GET /api/agent/create`
- `POST /api/upload`
- `GET /api/download`

### Knowledge / Log / Runtime

- `GET /knowledge`
- `GET /knowledge/<relative-path>`
- `GET /knowledge_lastUpdate`
- `GET /knowledge_path`
- `GET /log_round`
- `GET /log_skill`
- `GET /log_skill_status`
- `GET/POST/DELETE /api/host`
- `GET/POST/DELETE /api/standalone`
- `GET /api/site/access`
- `GET/POST /api/shutdown`

### 静态站点

- 通过 `static-server/server.Register(mux, cfg.Site)` 注册

## 本地访问保护与 Standalone 机制

### 本地管理请求判定

`isLocalManagementRequest()` 会同时校验：

- `RemoteAddr` 属于 loopback
- 请求 Host 也是 `localhost` / `127.0.0.1` / `::1`

因此“远程 Host + 本地转发地址”的伪本地请求不会被放行。

### 写接口保护

当前这些能力限制为本地管理请求：

- plugin 配置 / start / stop / exec
- `/api/cmd`
- `/api/kill`
- `/api/host`
- `/api/standalone`
- `/api/shutdown`

### Standalone

`standalone` 是运行时内存开关，不持久化到磁盘。

开启后：

- 所有非本地请求都会被 `withStandaloneAPIProtection()` 直接断开连接
- 不只是 `/api/...`
- 静态页面也会被阻断

## Proxy 转发设计

### `/v1/chat/completions` 的职责

当前实现不是简单转发，而是在发送给上游前做一轮完整的 metadata 组装。

### 元数据组装流程

`handleChatCompletions()` 的主流程：

1. 读取请求 JSON
2. 根据 `metadata.chat` 获取带 chat 语义的 AgentOutput
3. 将 AgentOutput 转为 `metaMap`
4. 把请求自带 `metadata` merge 到 `metaMap`
5. 按 `model` 从 `token_store` 读取配置
6. 把模型运行时 metadata 注入到 `metaMap`
   - `__url`
   - `__model`
   - `__model_fast`
   - `__model_thinking`
   - `__model_multi_input`
   - `__model_multi_output`
7. 处理 `knowledge.lastUpdate`
8. 把 `thinking/html/router_disable` 归一化到 `metadata`
9. 把 `metadata.type` 归一化成：
   - `page_session`
   - `scheduled_task`
10. 裁剪转发 metadata
    - 删除顶层冗余 `metadata.agent`
    - 当前 Agent 的版本、sandbox、media、knowledge 统一从 `metadata.agents[]` 读取

### Knowledge 元数据控制

`processKnowledgeMetadata()` 的当前规则：

- 若 `knowledge.lastUpdate` 距离当前时间未超过 `KnowledgeUpdateIntervalMs`
  - 则从 metadata 中删除该字段

- 若超出 interval，但 `knowledge_update_lock` 表中的 `last_requested_at` 仍在锁窗口内
  - 也删除该字段

- 只有超过 interval 且超过锁窗口时
  - 才保留 `lastUpdate`
  - 并更新 `knowledge_update_lock`

如果请求显式带了 `metadata.knowledge_commit=true`：

- 转发时不强制删除 `lastUpdate`
- SSE 完成后会调用 `updateKnowledgeManualTimestamps()`
  - 同时更新 knowledge last update
  - 以及 request lock 时间

### Token 回填

如果上游请求头里的 `Authorization`：

- 缺失
- 或是掩码值 `**********`

那么 integration 会自动回填 `token_store` 中该模型的真实 token。

### SSE 代理与日志

响应转发时：

- 头部基本保持透传
- 强制 `Accept: text/event-stream`
- 流式拷贝上游 SSE
- 同时把请求与 SSE 片段记入本地数据库

相关本地状态：

- `connMap`
  - 以 `agentId|chatId` 作为 key
  - 支持 `/api/cancel` 取消当前会话

- `chat_log`
  - 记录 `Q/A/X`

- `agent_message_log`
  - 记录结构化 event log

## cli-get 集成设计

### 基本模型

`startCliGet()` 在 integration 进程内启动后台 goroutine，复用：

- Agent metadata
- shared sqlite
- 统一 HTTP client
- 活跃命令表

不再依赖独立 `cli-get` 子进程。

### 心跳流程

1. 获取 AgentOutput
2. `POST <host>/cli/get`
3. 若拿到任务 JSON
   - 提交到 `ants` worker pool
4. 执行命令
5. `POST <host>/cli/pub`

### HTTP 行为

`createCliGetHTTPClient()` 使用 [http11client/http11client.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/http11client/http11client.go)：

- 强制 HTTP/1.1
- 禁止 ALPN 升级到 HTTP/2
- 独立 connect / total / response-header timeout

### 失败重试

心跳失败时使用指数退避：

- 初始值 `SleepMs`
- 每次翻倍
- 最大 15 秒

### 当前实现的一个细节

`shouldSleepAfterHeartbeat()` 只在“出错”时返回 `true`。

这意味着：

- 出错会 sleep + backoff
- 没有任务时不会 sleep
- 当前实现会持续紧凑轮询

这是现状，不是文档层的抽象优化。

### 命令执行

`executeTask()` 会：

1. 查询当前 `agentId + chatId` 的 sandbox mode
2. 若可用则通过 sandbox helper 执行
3. 否则直接走 shell `-c`
4. 输出 gzip+base64 编码
5. 结果回传 `/cli/pub`

### sandbox bypass 条件

`cli-get` 任务在以下场景会跳过 sandbox helper：

- `task.SubOps.Exempted = true`
- 命令本身是 integration 内部 token CLI
- 任务读写的绝对路径全部落在 integration 托管运行时目录内

## 本地命令执行与 Sandbox 设计

### `/api/cmd`

`/api/cmd` 与 `cli-get executeTask()` 不同，它没有复杂的子操作豁免逻辑，而是：

1. 校验本地请求
2. 校验 `agentId/chatId/cmd`
3. 拦截包含阻断 token 的命令
4. 查询当前 session sandbox
5. 有 sandbox 就走 helper
6. 否则直接 shell 执行

结果写入：

- `cmd_log`

### `/api/kill`

`/api/kill` 基于活跃命令表 `activeCmd`：

- 通过 `agentId/chatId/tid/cmd` 查找
- cancel context
- 直接 kill 子进程
- 懒创建并写入 `kill_log`

### sandbox 状态模型

sandbox 状态按会话持久化：

- 维度：`agentId + chatId`
- 存储：`sandboxstate` 对应 sqlite 表

能力入口：

- `/api/sandbox`
- `/api/sandbox=off`
- `/api/sandbox=filepick`
- `/api/sandbox=net`
- `/api/sandbox=filepick_net`
- `/api/sandbox_status`
- `integration sandbox ...`

### helper 路径解析

当前 helper 路径有两套：

- bundle 形态
  - `Helpers/<mode>/CLI_SANDBOX.app/Contents/MacOS/CLI_SANDBOX`

- 普通可执行文件形态
  - `<executable-base>/helpers/<mode>/CLI_SANDBOX`

### 白名单目录 priming

`filepick` / `filepick_net` 支持：

- 无 `--dir`
  - 通过 helper 弹出目录选择

- 有 `--dir`
  - 直接把目录传给 helper 作为白名单

`off` / `net` 不支持手动目录白名单。

## Plugin 与 Connect 集成设计

### CLI 收口

`integration connect ...` 当前直接调用 `connectsvc.RunCLIWithService()`。

它复用：

- shared sqlite
- integration 解析出的 `agent-dir`
- `connect-cache`

### 两类插件事实源

当前要特别区分两类列表：

- `integration connect list-plugins`
  - 扫描本地插件目录
  - 反映“当前有哪些插件二进制”

- `integration connect meta-list` / `list-meta`
  - 查询 connect 元数据配置
  - 反映“当前配置了哪些插件连接项”

二者不是同一份数据。

### `/api/plugins/meta`

当前 handler 的事实来源不是外部执行 `connect list-plugins`，而是：

1. 每次请求新建 `connectsvc.Service`
2. 通过 `listLocalPluginMeta()` 直接扫描本地插件目录

对远程请求会做脱敏：

- 清空 `param`
- 清空 `meta`
- 清空 `callback`
- 清空 agent/chat/model 等敏感运行字段

### 写操作权限

允许远程查看：

- plugin list/meta
- plugin status
- plugin log

仅允许本地执行：

- plugin config
- plugin start
- plugin stop
- plugin exec

### `--connect-bin` 自动补齐

以下入口会自动补齐 `--connect-bin` 指向当前 integration 可执行文件：

- `plugins start`
- `plugins stop`
- `plugins exec`

这保证插件生命周期命令回到当前 integration 的运行目录上下文。

### bundle 插件同步

`prepareIntegrationRuntimeLayout()` 与 `runIntegrationPluginSyncBundledCLI()` 都依赖：

- `buildIntegrationBundledPluginSyncPlan()`
- `applyIntegrationBundledPluginSyncPlan()`

同步粒度是：

- 按插件文件逐个比对 MD5
- 仅拷贝变化项

如果 bundle 启动时：

- 端口已被占用
- 且有待更新插件

则不会立即覆盖运行中的运行时插件，而是提示重启应用。

## Connect Pending Request -> Cron 桥接

### 后台同步周期

`startConnectPendingRequestSync()` 当前每 30 秒执行一次。

### 决策逻辑

它会把 connect pending request 交给 `connectsvc.SyncPendingPluginRuntime()`，并使用：

- `ExpireAge = 20m`
- `PendingDetailState = 0`
- `ExpiredDetailState = 2`

### 生成任务

一旦判定需要落任务，会通过 `createImmediateCronTaskFromConnect()`：

- 新建 `task_meta`
- 新建 `task_detail`
- 写 `cron_meta_log`
- 写 `cron_detail_log`

并额外保留：

- `meta_ref`
- `task_type`
- `router_disable`
- `response_schema`

### 自动通知与完成回推

桥接任务创建后：

- 会调用 plugin callback `init`
- 发送“开始执行”回复

cron detail 完成后：

- `notifyCompletedConnectTasks()` 会根据 `meta_ref` 找回原始 request
- 通过 plugin callback `send` 自动回复
- 并把 `task_detail.replied_at` 写入当前时间

## Cron 设计

### 实现边界

`integration` 没有直接复用 `cron` 模块的 service 对象。

当前是把 cron 的一套“集成版实现”直接写在 [main.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/main.go) 里，并做了额外扩展：

- `meta_ref`
- `result_content`
- `replied_at`
- chat / event / cmd / kill log
- connect pending request 桥接

### 主要表结构

当前至少会维护这些表：

- `task_meta`
- `task_detail`
- `cron_meta_log`
- `cron_detail_log`
- `chat_log`
- `agent_message_log`
- `cmd_log`
- `kill_log`
- `token_store`

其中 `task_detail` 相比独立 `cron` 模块额外包含：

- `meta_ref`
- `result_content`
- `replied_at`

### 创建任务

`createCronTask()` 支持两类任务：

- 内置周期 `cycle = 0..5`
- 自定义 cron `cycle = -1`

#### 内置周期

- `0`
  - 一次性
- `1`
  - 工作日
- `2`
  - 自然日
- `3`
  - 每小时
- `4`
  - 每 15 分钟
- `5`
  - 每 30 分钟

#### `router_disable`

当前 integration cron 的权威字段是：

- `router_disable`

而不是历史文档里的 `swarm`。旧别名在这里已不再参与解析，默认值是：

```text
true
```

### 首批 detail 生成

#### `cycle = 0`

- 创建单条 detail
- 且当前实现会拒绝“早于当前分钟”的一次性时间

#### `cycle = 1 / 2`

- 创建首条 detail
- 再补齐未来 5 天窗口

#### `cycle = 3 / 4 / 5`

当前实现会：

- 直接从 `time.Now().Truncate(time.Minute)` 开始
- 按固定间隔铺满未来 5 天窗口

注意这和“从 `rawTime` 开始滚动”不同，当前代码并不以 `rawTime` 作为高频 detail 的起点。

#### `cycle = -1`

- 只创建 meta
- 不立即生成 detail

### 周期补齐

`startCronCheck()` 每分钟触发一次 `cronCheckOnce()`。

补齐前会先清理：

- 已删除 Agent 的任务
- 已失效 model 的任务

#### 当前实际会补齐的任务类型

`cronCheckOnce()` 当前只补齐：

- 自定义 cron `cycle = -1`
- 工作日 `cycle = 1`
- 自然日 `cycle = 2`

也就是说，**当前实现不会在后台继续补齐 `cycle = 3 / 4 / 5` 的高频任务**；它们只有创建时铺出来的首个 5 天窗口。

这是当前实现非常关键的现状。

### 执行 detail

`startCronExecutor()` 每分钟触发 `cronExecuteOnce()`。

执行流程：

1. 查找 `started = 0` 且 `exec_time <= now` 的 detail
2. 只处理最近 1 小时内到期的 detail
3. 校验 Agent 仍存在
4. 校验 model 仍在 `token_store` 中有效
5. `started = 1`
6. 若 `chat_id` 为空则自动补一个 `metaId@detailId`
7. 组装 metadata
8. 向当前 runtime host 的 `/v1/chat/completions` 发起 SSE 请求
9. 成功后：
   - `started = 3`
   - 写入 `result_content`
10. 失败后：
   - 回滚为 `started = 0`

### 执行请求 metadata

cron 执行时会额外注入：

- `agentId`
- `chat`
- `type = scheduled_task`
- `cron_type`
- `thinking`
- `router_disable`
- `agent`
- `router_remote`
- `META_ID`

如果 detail 带有 `response_schema`：

- `metadata.response_schema` 也会透传
- 同时请求体会补 `response_format.type = json_schema`

### 查询与删除

当前提供：

- `find-meta`
- `find-detail`
- `delete-meta`
- `delete-detail`

HTTP 侧也有对应 API。

时间过滤支持：

- `YYYY-MM-DD`
- `YYYY-MM-DD HH:MM`
- `YYYY-MM-DD HH:MM:SS`
- `RFC3339`

并兼容：

- `meta_123`
- `cron_123`
- `detail_123`

### 当前一个重要约束

`delete-meta` / cleanup 类删除逻辑在删除元数据前，只会删除：

```sql
started != 3
```

的 detail。

这意味着：

- 已完成 detail 可能在 meta 删除后保留在 `task_detail`
- 当前实现可能留下孤立的历史完成记录

这是现状，设计文档需要明确记录。

## Token 与模型配置设计

### 存储

integration 使用共享 sqlite 中的 `token_store` 保存模型配置。

支持的核心字段包括：

- `token`
- `base_url`
- `model_base`
- `model_fast`
- `model_thinking`
- `model_multi_input`
- `model_multi_output`

### HTTP 行为

`/api/token`：

- `GET`
  - 读取模型配置
  - 远程请求返回掩码 token

- `POST`
  - 写入模型配置
  - 远程请求不能新增模型
  - 对已有模型，若传入掩码 token，会保留数据库中的真实 secret

### CLI 行为

`integration token` 既支持：

- 读 / 写 `token_store`
- 追加 token consume 记录
- 查询 token consume 历史

### `/api/config` 的现状

历史上叫 `swarm` 的配置入口现在有两种职责：

1. 给 Agent 工作区写 `config.json`
   - `description`
   - `thinking`
   - `router_disable`

2. 处理 `delete_model`
   - 删除 `token_store` 中某个模型

因此它已经不只是“swarm 开关”接口。

## Skills、Knowledge、文件与 Agent 管理

### `/api/skills`

返回的不是纯 Agent 声明 skills，而是三部分合并：

1. Agent 自身 skills
2. `config/config.json` 里的全局 `skills`
3. 运行中内部 plugin skills
   - `browser` -> `__internal_browser`
   - `remote` -> `__internal_remote`

### skills warning

`startCronExecutor()` 在执行 cron 前会先同步一次 skills warning，此后每分钟同步一次：

- 扫描 agent 目录下的 `SKILL.md`
- 结果写入共享 sqlite warning store

入口：

- `GET /skills_warning`
- `integration skills-warning [--refresh]`

### Knowledge

当前对外能力：

- `GET /knowledge`
- `GET /knowledge/<path>`
- `GET /knowledge_lastUpdate`
- `GET /knowledge_path`
- `integration knowledge update-time`
- `integration knowledge last-update`

特点：

- 目录树渲染与文件读取在 integration 内完成
- 对相对路径做 escape 防护
- 共享 knowledge runtime 与 shared sqlite

### 文件接口

当前文件能力分两类：

- 纯读取
  - `/api/data`
  - `/api/raw`
  - `/api/files`
  - `/file/lastUpdate`

- workspace 内写操作
  - `/api/edit`
  - `/api/del`
  - `/api/agent/create`
  - `/api/upload`

写操作的共同原则：

- 只允许相对路径
- 只允许在 Agent workspace 内
- 会阻止 `..` 逃逸

某些读接口支持绝对路径，例如：

- `/api/raw`
- `/file/lastUpdate`

### Agent 管理

当前支持：

- `agent init`
  - 从 `default-dir` 复制模板
  - 但新 Agent 的根 `config.json` 固定初始化为 `{}`，不会继承主应用配置

- `agent export`
  - zip 导出
  - 自动过滤 Agent 顶层受管目录，如 `chrome*`、`data`、`tmp`

- `agent import`
  - zip 或目录导入
  - 若同名 Agent 已存在则拒绝

- `agent delete`
  - 删除目录
  - 并清理该 Agent 对应 cron 数据

## Runtime Host、浏览器与桌面 bundle 设计

### Runtime Host

`runtimehost.State` 是纯内存覆盖层：

- startup host 来自 `--host` / `config/config.json`
- runtime override 只活在当前进程

入口：

- `GET/POST/DELETE /api/host`
- `integration host get|set|reset`

### 浏览器打开

`openIntegrationBrowserMaximized()` 的优先级：

- macOS
  - 优先尝试现有 Chrome 标签页激活
  - 否则 `open -a Chrome/Edge/Brave/Chromium`

- Linux
  - 优先 `google-chrome` / `chromium` / `edge` / `brave`
  - 否则 `xdg-open`

- WSL
  - 优先查 Windows 宿主机浏览器路径
  - 否则 `cmd.exe /c start /max`

- Windows
  - 优先常见安装目录
  - 否则 PATH

### bundle 启动

bundle 双击启动时：

1. 过滤 macOS 自动注入的 `-psn_*`
2. 先尝试 splash
3. 准备运行时目录
4. 如已有运行服务则直接打开浏览器
5. 否则走 `start`

当前代码里：

- bundle 启动位置不再限制必须放到 `/Applications`
- 虽然历史 alert 文案还保留着“必须放在应用程序目录”的文本函数，但现在不会触发这条限制

## 测试设计

当前测试覆盖面很宽，重点包括：

- `/v1/chat/completions` metadata 注入
- token 掩码回填与模型 metadata 注入
- knowledge interval / lock / `knowledge_commit`
- `cli-get` 心跳、退避、统一日志、HTTP debug
- sandbox helper 路径解析、session 持久化、命令执行
- plugin meta/status/log/start/stop/exec 与远程脱敏
- bundle 运行时目录、DB 迁移、插件同步、浏览器唤醒
- `start/stop/restart` 的 PID、ready、日志与插件停止语义
- cron 创建 / 查询 / 删除 / 执行 / `router_disable` 继承
- connect pending request -> cron 桥接与完成回推
- Agent 导入导出
- Knowledge 文件树
- file-last-update / raw / folder / workspace 路径安全
- `skills-warning`
- acceptance fixtures

对应测试文件主要包括：

- [main_test.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/main_test.go)
- [acceptance_test.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/acceptance_test.go)
- [plugin_runtime_sync_test.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/plugin_runtime_sync_test.go)
- [browseropen_test.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/browseropen_test.go)
- [agent_import_export_test.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/agent_import_export_test.go)
- [runtime_config_bundle_test.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/runtime_config_bundle_test.go)
- [http_client_config_test.go](/Users/shenjiawei/Documents/code/deepright/cli/module/integration/http_client_config_test.go)

## 已知约束

- 当前核心逻辑高度集中在 `main.go`，模块内聚度高但体量很大。
- `config/config.json` 同时承担启动配置与运行时记录，旧版单独启动配置文件的描述已不符合当前实现。
- 服务端口当前硬编码只允许 `8080`。
- 高频 cron `cycle = 3 / 4 / 5` 只在创建时铺 5 天窗口，后台不会继续补齐。
- `delete-meta` 与部分 cleanup 流程可能留下 `started = 3` 的孤立 detail 历史记录。
- `cli-get` 在“无任务”场景当前不会 sleep，会持续快速轮询。
- `runtime host` 与 `standalone` 覆盖都是纯内存状态，重启后丢失。
- plugin 远程访问当前允许读取列表、状态、日志，不允许远程执行管理动作。
- bundle 启动位置限制已经取消，但遗留文案函数仍在代码中保留。
