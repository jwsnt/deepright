# Connect 模块详细技术设计

## 设计目标

+ 将“连接元数据管理”“三方请求落库”“三方响应落库”“插件发现与生命周期”“插件回调通知”收敛到统一模块，避免 `integration`、`proxy`、插件实现层各自维护一套连接逻辑。
+ 明确 `connect` 的两层职责：
    + `connectsvc` 作为唯一共享能力内核。
    + `connect/main.go` 作为极薄 CLI 包装层。
+ 固化插件标识语义：
    + `key` 是运行时唯一主标识。
    + `name` 仅用于展示，不参与运行态查找。
+ 满足 integration 的收口原则：
    + `connect` 可独立启动本地 HTTP 服务。
    + 也可被 `integration` / `proxy` 嵌入为动态 HTTP handler 或共享服务实例。

## 模块分层

当前 `connect` 模块可以分为四层：

### 1. 顶层入口层

+ [main.go](/Users/shenjiawei/Documents/code/deepright/cli/module/connect/main.go)
    + 只负责把 `os.Args` 交给 `connectsvc.RunCLI()`。
    + 不承载任何业务逻辑。

### 2. 共享服务内核层

+ `connectsvc`
    + 负责 CLI 命令解析、HTTP 客户端、HTTP 服务端、SQLite 读写、缓存、插件发现、插件生命周期调用、插件回调能力检测。
+ `sharedutil`
    + 负责系统环境、路径、执行器、token 规范化等通用能力。
+ `sharedschemasvc`
    + 负责统一 schema 能力。
+ `sandboxstate`
    + 负责 `AgentId + ChatId` 维度的沙盒状态存储。
+ `pluginlog`
    + 负责插件日志相关共享能力。

### 3. 插件包装层

这些目录提供符合 `PLUGIN.md` 协议的插件 CLI 包装：

+ `browser`
+ `email`
+ `feishu`
+ `remote`

包装层职责：

+ 实现 `name`、`param`、`scope`、`command`、`help` 等标准插件命令。
+ 暴露 `start`、`stop`、`status`、`init`、`send` 等插件生命周期命令。
+ 通过 CLI 协议与 `connect` 交互，而不是直接依赖数据库。

### 4. 插件实现层

部分插件又进一步拆出更底层的实现包：

+ `browserplaywrightsvc`
+ `emailsvc`
+ `feishusvc`
+ `remote/instance`

这些包负责具体平台实现，而不是对外协议收口。

## 总体架构

`connect` 的核心结构是“服务内核 + 本地 HTTP 协议 + 插件 CLI 协议”的双协议架构：

1. 连接元数据、请求、响应统一存储在共享 sqlite。
2. 本地命令默认通过 HTTP 请求访问 `connectsvc.Service`，而不是直接碰数据库。
3. 插件发现与生命周期管理通过扫描 `plugins/` 目录和执行插件 CLI 命令完成。
4. 插件事件回调仍然走 CLI，而不是进程内直接调用插件代码。
5. `integration` / `proxy` 可以直接复用 `connectsvc`，不必重复实现连接链路。

该设计满足三个核心目标：

+ 数据唯一事实源在 `connectsvc.Service`。
+ 插件运行时协议统一走 CLI，避免编译期耦合。
+ 同一套内核既支持独立进程模式，也支持嵌入模式。

## 关键边界

### `connect` 与插件的边界

+ `connect` 不能直接 import 插件实现代码来完成业务。
+ `connect` 与插件交互只允许通过插件二进制和稳定 CLI 命令。
+ 插件负责“接入三方平台”和“实现具体平台动作”。
+ `connect` 负责“存储统一连接态”和“把请求/响应抽象成统一模型”。

### `connect` 与 `integration/proxy` 的边界

+ `connectsvc.Service` 是 authoritative implementation。
+ `integration/proxy` 若需要连接能力，必须复用 `connectsvc` 或 `connect` 的 HTTP/CLI 能力。
+ 不允许在上层模块复制 `meta`、`request`、`response` 表的业务语义或 CRUD 逻辑。

## 数据模型

### Meta

`Meta` 是数据库视图，保留了历史字段结构：

```json
{
  "id": 1,
  "key": "feishu",
  "name": "feishu",
  "meta": "{\"appId\":\"...\"}",
  "stream": true,
  "callback": "/abs/path/to/plugins/feishu",
  "agentId": "A",
  "chatId": "chat-001",
  "model": "OpenAI",
  "thinking": true,
  "router_disable": false,
  "createdAt": "2026-06-01T10:00:00Z",
  "updatedAt": "2026-06-01T10:00:00Z",
  "deletedAt": ""
}
```

实现细节：

+ sqlite 真实列名仍是 `connect_meta.name`。
+ 当前运行时语义中，这个字段已经被当作插件 `key` 使用。
+ `Meta.Key` 主要是运行时填充字段，不是独立持久化列。
这是一个兼容历史结构后的现实设计：数据库层还保留旧列名，但运行态主语义已经切换为 `key`。

### MetaConfig

`MetaConfig` 是插件消费的运行时配置视图：

```json
{
  "key": "feishu",
  "name": "飞书",
  "meta": {
    "appId": "cli-app",
    "appSecret": "cli-secret"
  },
  "stream": true,
  "callback": "/abs/path/to/plugins/feishu",
  "agentId": "A",
  "chatId": "chat-001",
  "model": "OpenAI",
  "thinking": true,
  "router_disable": false,
  "createdAt": "...",
  "updatedAt": "..."
}
```

`MetaConfig` 与 `Meta` 的区别：

+ `meta` 从 JSON 字符串解码成 `map[string]any`。
+ `callback` 会被归一化为绝对路径。
+ `key` 和展示名 `name` 会优先通过插件二进制自检结果修正。

### Request

```json
{
  "id": 1,
  "key": "feishu",
  "name": "feishu",
  "externalId": "msg-1",
  "content": "HELLO WORLD",
  "request": "HELLO WORLD",
  "artifacts": "/abs/a.png,/abs/b.pdf",
  "original": "{\"text\":\"HELLO WORLD\"}",
  "rawRequest": "{\"text\":\"HELLO WORLD\"}",
  "responseSchema": "{\"type\":\"object\"}",
  "status": 0,
  "createdAt": "2026-06-01T10:00:00Z"
}
```

状态枚举：

+ `0` `RequestStatusPending`
+ `1` `RequestStatusStarted`
+ `2` `RequestStatusCompleted`
+ `3` `RequestStatusExpired`
+ `4` `RequestStatusReplied`

### Response

```json
{
  "id": 1,
  "name": "feishu",
  "requestId": 1,
  "response": "HELLO BACK",
  "artifacts": "/abs/a.png",
  "createdAt": "2026-06-01T10:01:00Z"
}
```

## SQLite 设计

### 数据库路径

`DefaultDBPath()` 的规则：

+ 若 `CONNECT_DB` 环境变量存在，优先使用。
+ 若存在 `../cron`，默认使用 `../cron/data`，与 cron 模块共享数据库。
+ 否则回退到当前目录 `data`。

### 共享连接池

`connectsvc.NewService()` 不会为每个 Service 实例都新建一个 sqlite 连接，而是通过全局引用计数池共享：

```go
map[string]*sharedDBEntry
```

每个 DB 路径维护：

+ `db *sql.DB`
+ `refs int`

连接池参数：

+ `SetMaxOpenConns(10)`
+ `SetMaxIdleConns(10)`
+ `SetConnMaxLifetime(30 * time.Minute)`

设计目标：

+ 同一路径下多个 `Service` 实例复用同一连接池。
+ 独立 CLI、HTTP handler、嵌入调用之间共享数据库连接，而不是反复开关 sqlite。

### 表结构

`initDB()` 会初始化以下表：

+ `token_store`
+ `token_store_log`
+ `connect_meta`
+ `connect_request`
+ `connect_response`

核心表结构：

#### `connect_meta`

```sql
CREATE TABLE IF NOT EXISTS connect_meta (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    meta TEXT NOT NULL,
    stream INTEGER NOT NULL DEFAULT 0,
    callback TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    chat_id TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL,
    thinking INTEGER NOT NULL DEFAULT 0,
    router_disable INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT NOT NULL DEFAULT ''
)
```

活跃配置唯一索引：

```sql
CREATE UNIQUE INDEX IF NOT EXISTS idx_connect_meta_name_active
ON connect_meta(name) WHERE deleted_at = ''
```

#### `connect_request`

```sql
CREATE TABLE IF NOT EXISTS connect_request (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    external_id TEXT NOT NULL DEFAULT '',
    request TEXT NOT NULL,
    artifacts TEXT NOT NULL DEFAULT '',
    raw_request TEXT NOT NULL DEFAULT '',
    response_schema TEXT NOT NULL DEFAULT '',
    status INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL
)
```

索引：

+ `(name, id)`
+ `(name, created_at)`
+ `(name, external_id)` 的条件唯一索引，`external_id != ''`

#### `connect_response`

```sql
CREATE TABLE IF NOT EXISTS connect_response (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    request_id INTEGER NOT NULL,
    response TEXT NOT NULL,
    artifacts TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
)
```

索引：

+ `(name, id)`
+ `(request_id, id)`

### 迁移策略

`initDB()` 采用轻量在线迁移：

+ `ALTER TABLE` 补齐缺失列。
+ 自动补齐 `router_disable` 列。
+ 将旧状态规范化。
+ 创建缺失索引。

设计目标：

+ 避免复杂迁移框架。
+ 保证已有本地数据库可以平滑升级。

## 服务内核设计

### `Service`

`connectsvc.Service` 封装：

+ `db`
+ `dbPath`
+ `agentDir`
+ `cacheTTL`
+ `closeFn`
+ 进程内 JSON cache

这是模块唯一数据库业务内核，所有 Meta/Request/Response 的合法操作都必须通过它。

### Agent 与模型校验

`Service` 在写路径上做两类硬校验：

+ Agent 校验
    + 插件 scope 允许 `agent` 时，必须存在对应 Agent 目录。
+ 模型校验
    + 插件 scope 允许 `provider` 时，模型必须已在 `token_store` 注册且 token 非空。

这两个校验在以下路径都会触发：

+ `CreateMeta`
+ `UpdateMeta`
+ `AddRequest`
+ `AddResponse`
同时，scope 为空列表时允许插件完全跳过容器配置要求。

### Meta 写路径

#### `CreateMeta`

处理流程：

1. `normalizeMetaInput()`
2. `validateMetaInput()`
3. 插入 `connect_meta`
4. 失效缓存
5. 回读新记录

注意点：

+ `normalizeMetaInput()` 会把 `input.Name = input.Key`。
+ `callback` 会被强制归一化到默认插件路径：
    + `<plugin-dir>/<key>`
也就是说，用户传入的 `callback` 在当前实现里只是兼容输入位，最终运行时路径由插件 key 和运行目录决定。

#### `UpdateMeta`

+ 先读取当前活跃 Meta。
+ 将 patch 合并回完整 `MetaInput`。
+ 再走和创建相同的 normalize + validate 流程。
这样保证更新路径不会绕过任何校验。

#### `DeleteMeta`

+ 采用软删除：
    + 写入 `deleted_at`
    + 更新 `updated_at`
被删除的 Meta 不会再接受新请求。

### Request 写路径

#### `AddRequest`

处理流程：

1. 通过 `key` 找到活跃 Meta。
2. 校验 Meta 仍可运行。
3. 归一化内容：
    + `content` 为空时回退 `request`
    + `original` 为空时回退 `rawRequest`
4. 归一化附件路径。
5. 校验 status。
6. 归一化创建时间。
7. 插入 `connect_request`。

实现特点：

+ `name` 列最终写的是 `meta.runtimeKey()`。
+ `external_id` 只在非空时参加唯一性约束。
+ `artifacts` 会被展开为绝对路径并校验存在。

#### `ListRequests`

+ 只支持按 `key` 过滤，不再支持 `name` 过滤。
+ 返回值中 `fillDerivedFields()` 会把：
    + `Key <- Name`
    + `Content <- Request`
    + `Original <- RawRequest`

#### `UpdateRequestStatus`

+ 支持按当前状态做 CAS 风格更新：
    + `From`
    + `To`
    + `Strict`
这个设计用于上层保证请求状态推进的幂等性。

### Response 写路径

#### `AddResponse`

处理流程：

1. 校验 `key`
2. 找到活跃 Meta
3. 校验 Meta 运行态
4. 校验 `request_id`
5. 校验请求归属插件与当前 Meta 一致
6. 插入 `connect_response`
7. 将对应 `connect_request.status` 更新为 `RequestStatusReplied`

这保证了 Response 与 Request 的归属一致，不允许跨插件回写。

## 进程内缓存设计

### Service 读缓存

`Service` 维护一个进程内 JSON cache：

+ 读路径命中：
    + `GetMeta`
    + `ListMeta`
    + `ListMetaConfig`
    + `ListRequests`
    + `ListResponses`
+ 写路径触发：
    + `CreateMeta`
    + `UpdateMeta`
    + `DeleteMeta`
    + `AddRequest`
    + `UpdateRequestStatus`
    + `AddResponse`
缓存特点：

+ 按 key 保存 JSON payload，而不是原始对象引用。
+ 读出时重新反序列化，避免调用方修改缓存对象。
+ TTL 来自 `Options.CacheTTL`，默认 10 秒。

### 插件发现缓存

`list-plugins` 还有一层独立文件缓存：

+ 文件名：`.connect-plugin-cache.json`
+ 位置：插件目录下
+ TTL：`--connect-cache`
该缓存只服务于插件扫描，不参与 Meta/Request/Response 的内存缓存。

## CLI 设计

### 总入口

`connectsvc.RunCLI()` 是唯一 CLI 入口。

命令分三类：

+ 服务生命周期命令
    + `start`
    + `serve`
    + `stop`
+ 本地插件发现命令
    + `list-plugins`
+ 业务命令
    + `meta-create`
    + `meta-update`
    + `meta-delete`
    + `meta-get`
    + `meta-list`
    + `list-meta`
    + `add-request`
    + `request-list`
    + `add-response`
    + `response-list`

### 命令执行模式

#### 直接本地执行

+ `start` / `serve` / `stop`
+ `list-plugins`

这些命令直接在本地处理，不经过 APIClient。

#### 通过本地 HTTP 服务执行

其他业务命令默认走：

+ `APIClient`
+ 本地 HTTP 协议
这样可以保证：

+ 命令行工具和 HTTP 服务共用同一业务内核。
+ 独立 CLI 不会绕开服务层直接写数据库。

### `RunCLIWithService`

该入口允许在已持有 `Service` 实例时直接复用同一内核执行命令。

用途：

+ 测试
+ 嵌入式场景
+ integration/proxy 内部直接调度

### `RunCLIWithClient`

该入口强制命令走 HTTP client，用于：

+ 独立进程模式下的业务命令
+ 连接到外部 `connect` 服务

## 本地 HTTP 服务设计

### 服务地址

默认监听：

```text
127.0.0.1:18080
```

基础 URL 解析规则：

+ `--host`
+ `--addr`
+ `--port`
+ `CONNECT_ADDR`
最终由 `ServiceBaseURLFromFlags()` 和 `ListenAddrFromFlags()` 统一解析。

### API 路径

+ `/api/connect/health`
+ `/api/connect/meta`
+ `/api/connect/request`
+ `/api/connect/response`

### 响应格式

成功：

```json
{
  "status": 0,
  "data": {}
}
```

失败：

```json
{
  "status": 1,
  "content": "error message"
}
```

### Handler 形态

#### `NewHTTPHandler`

+ 适用于已有固定 `Service` 实例的场景。

#### `NewDynamicHTTPHandler`

+ 适用于上层模块按请求动态提供 Service 的场景。
+ 通过 `ServiceProvider` 注入：
    + `svc`
    + `release`
这个设计使 `connect` 能被 `integration` / `proxy` 无缝嵌入，而不要求额外起一个独立进程。

## 服务生命周期设计

### `start`

+ 若未加 `--foreground`，走后台模式。
+ 后台模式会：
    + 解析可执行文件路径
    + 重新启动自身并执行 `serve`
    + 写日志文件
    + 写 PID 文件
    + 轮询健康检查直到服务可用

### `serve`

+ 前台启动 HTTP 服务。
+ 绑定 listener。
+ 写 PID 文件。
+ 监听系统信号并优雅关闭 `http.Server`。

### `stop`

+ 根据 PID 文件找到进程。
+ 发送 `SIGTERM`。
+ 轮询等待 PID 文件消失。

### 文件约定

+ PID 文件默认：`connect.pid`
+ 日志文件默认：`connect.log`

## 插件协议设计

### 基础协议

插件必须通过 CLI 提供这些命令：

+ `name`
+ `param`
+ `scope`
+ `command`
+ `help`

可选命令：

+ `schema`
+ `init`
+ `send`
+ `start`
+ `stop`
+ `status`
+ 其他插件自定义业务命令

### `key` / `name` 语义

当前设计强约束：

+ `key`
    + 稳定唯一
    + 参与 meta 查找、request 归属、response 归属、callback 路由、生命周期管理
+ `name`
    + 仅用于展示
    + 可来自插件 `name` 命令
    + 不允许作为运行时查找主键

### 插件 scope

插件 `scope` 决定容器层允许配置哪些运行时字段：

+ `reuse`
+ `agent`
+ `provider`
+ `thinking`
+ `swarm`

如果插件返回空数组：

+ 表示完全不需要容器通用配置。
+ `connect` 将跳过 Agent/Model 等校验。

### 插件命令能力发现

`list-plugins` / `inspectLocalPlugin` 会并发调用插件：

+ `name`
+ `param`
+ `scope`
+ `command`
+ `help`

如果插件未实现 `scope`：

+ 本地扫描场景会回退为 `nil`
而运行时 callback 检测则不会做模糊回退，`init` / `send` 是否支持必须以 `command` 输出为准。

## 插件发现与生命周期设计

### `list-plugins`

`ListPlugins()` 扫描运行时插件目录，产出：

+ `key`
+ `name`
+ `param`
+ `scope`
+ `command`

扫描规则：

+ 只看插件目录当前层。
+ 跳过隐藏文件、目录和运行时产物。
+ 支持：
    + 无扩展名可执行文件
    + `.py`
    + `.js`
    + `.go`
脚本插件会自动根据运行时选择解释器执行。

### 本地插件目录

插件目录解析有两条路径：

+ `defaultPluginDir()`
    + 面向运行时二进制目录和环境变量解析
+ `DefaultLocalPluginDir()`
    + 面向当前启动目录下的 `plugins/`
    + 主要给本地集成场景用

### 插件动作执行

`RunPluginAction(target, action, flags)` 负责：

1. 解析插件二进制。
2. 检查命令是否支持。
3. 以插件目录作为工作目录执行。
4. 返回路径、命令行和原始输出。

状态查询：

+ `GetPluginStatus`
+ `PluginStatusByKey`

通过 PID 文件判断插件是否已启动。

## 插件配置归一化设计

`UpsertPluginConfig()` / `UpsertPluginConfigWithService()` 是插件容器侧的高层写接口。

作用：

+ 基于插件 `key` 获取插件定义。
+ 归一化 scope 支持的字段。
+ 回写/创建对应 Meta 配置。
+ 统一 callback 路径。
这使上层模块不必自己理解不同插件的 scope 差异。

## 插件回调运行时设计

`callback_runtime.go` 和 `plugin_callback_runtime.go` 提供的是“连接层到插件层”的回调桥接能力。

### 回调映射

+ 先从 `MetaConfig` 列表中构建：
    + `plugin key -> callback path`
+ 仅使用 key 做映射。

### 能力检测

在真正调用 `init` 或 `send` 前，会先执行插件 `command`：

+ 只有 `command` 明确包含对应能力时才允许调用。
+ 不再靠隐式回退、失败吞掉或展示名兼容做推断。

### 启动通知

`NotifyPluginStarted()` 用于在任务开始前向插件发送 `init`。

### 完成回执

`DispatchCompletedPluginReplies()` 用于在任务结束后向插件发送 `send`。

### 待处理请求同步

`SyncPendingPluginRuntime()` 用于把还未进入任务系统的 connect request 转成上层任务，并在需要时通知插件。

这些 helper 的设计目的不是替代 `integration` / `proxy`，而是让上层可以直接复用连接态到插件通知的统一编排逻辑。

## Runtime Config 设计

`runtime_config.go` 用于从 `connect` 或 bundle 目录附近找到运行时 `config.json`。

支持：

+ 从 `connect-bin` 反推配置目录
+ 从 macOS `.app` bundle 推导用户目录下的 runtime config
+ 解析相对路径配置值为绝对路径

该能力主要服务于插件和上层容器在不同发布形态下的路径解析。

## HTTP Client 设计

`APIClient` 负责对 `connect` 本地 HTTP 服务做统一封装：

+ `Health`
+ `CreateMeta`
+ `UpdateMeta`
+ `DeleteMeta`
+ `GetMeta`
+ `GetMetaConfig`
+ `ListMeta`
+ `ListMetaConfig`
+ `AddRequest`
+ `ListRequests`
+ `UpdateRequestStatus`
+ `AddResponse`
+ `ListResponses`

超时固定为 3 秒。

设计目标：

+ 让 CLI 调用和嵌入式调用都能共享同一套请求编码/响应解码逻辑。

## 错误处理策略

+ 参数缺失：
    + 直接返回明确错误，例如 `key is required`
+ 资源不存在：
    + Meta 不存在：`connect meta not found`
    + Request 不存在：`request not found`
    + Agent 不存在：`agent not found`
    + 模型未注册：`model not registered`
+ 插件协议不满足：
    + 直接报错，不做模糊回退
+ 服务未启动：
    + 业务 CLI 走 HTTP 时直接返回错误
+ 插件生命周期命令失败：
    + 原样向上返回，不自动改走其它路径

## 测试设计

当前测试覆盖重点包括：

+ 帮助文档输出
+ Meta / Request / Response 全链路
+ Meta 删除后阻断新流量
+ `externalId` 唯一性
+ Request status 过滤与状态迁移
+ CLI 通过 HTTP 服务调用 CRUD
+ `list-meta` 与 `meta-get` 的运行时配置视图
+ 插件 key / 展示名语义隔离
+ callback 归一化到 runtime plugins 目录
+ Model 注册校验
+ 本地插件扫描：
    + 可执行脚本扩展名
    + scope 解析
    + 已保存配置合并
+ 插件动作执行：
    + key 解析
    + 命令兼容
    + PID 文件与 status 判断
+ runtime config 路径解析
+ callback runtime 能力检测

这些测试共同保证：

+ `connectsvc` 是稳定的唯一事实源。
+ 插件协议没有在不同入口之间产生漂移。

## 已知约束

+ 历史数据库列名 `name` 仍存在，短期内不会物理改名为 `key`。
+ `connect` 主模块本身不提供前端 UI，只提供 CLI 与 HTTP API。
+ 插件发现依赖插件二进制正确实现 `name/param/scope/command/help`。
+ 插件状态判断目前以 PID 文件为主，不读取更复杂的进程内健康状态。
+ `list-plugins` 与 `list-meta` 是两个不同事实源：
    + 前者是“目录里有哪些可用插件”
    + 后者是“当前已配置了哪些连接”

## 后续演进原则

+ 若继续扩展连接模型，优先在 `connectsvc.Service` 中落地，不要在上层模块复制逻辑。
+ 若要治理历史 `name`/`key` 技术债，应先保证兼容 API 语义，再考虑数据库列迁移。
+ 若增加新插件，必须先满足 `PLUGIN.md` 协议，再接入 `connect`，不能让容器层为单个插件做特判扩散。
