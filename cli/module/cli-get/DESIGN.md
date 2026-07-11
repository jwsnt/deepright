# CLI-Get 模块详细技术设计

## 设计目标

+ 将 `cli/get -> 执行任务 -> cli/pub` 收敛为单一客户端实现，避免上游模块各自重复维护心跳、执行、回传链路。
+ 复用 `agentcore` 作为 Agent 元数据唯一事实源，`cli-get` 只负责上传、任务调度、执行和回传，不重复定义 Agent schema。
+ 将运行态状态拆分为三类并分别处理：
    + 请求链路状态：心跳、任务、回传。
    + 本地执行状态：线程池、活跃命令、Shell/沙盒执行。
    + 持久化状态：统一日志、会话沙盒模式。
+ 满足 integration 的二进制收口原则：`cli-get` 既可单独编译运行，也可被更上层统一 CLI 收口复用。

## 模块边界

+ `main.go`
    + 当前模块主编排入口。
    + 负责 HTTP 请求、任务解析、线程池调度、沙盒模式判定、统一日志、活跃命令注册、CLI 参数处理。

+ `taskexec/taskexec.go`
    + 负责本地 Shell 执行。
    + 处理超时、取消、权限拒绝早停、标准输出/错误合并采集。
    + 对上暴露通用执行抽象，便于 `cli-get` 主链路复用。

+ `sandbox/`
    + 是独立沙盒执行子模块，不是 `cli-get` 主链路内部实现细节。
    + `cli-get` 主模块只在需要时调用外部 `CLI_SANDBOX` 可执行体，不直接内嵌平台沙箱策略。

+ 复用外部模块
    + `agent-scanner/agentcore`
        + 提供 Agent 元数据、插件运行态过滤能力。
    + `knowledge/knowledgecore`
        + 提供统一 sqlite 路径解析规则。
    + `connect`
        + 只通过 replace 依赖提供其他共享约束；主执行链路不直接复刻 connect 内核。

## 总体架构

当前实现是“单主循环 + worker 池 + 外部状态存储”的结构：

1. 主线程循环获取 Agent 元数据。
2. 主线程调用 `/cli/get` 上报心跳。
3. 若无任务，立即进入下一次心跳。
4. 若有任务，把任务投递到 worker 池。
5. worker 决定走本地 Shell 还是 `CLI_SANDBOX`。
6. worker 执行任务，得到结果后调用 `/cli/pub`。
7. 心跳日志和回传日志异步写入 sqlite。
8. 正在执行的命令注册到进程内活跃命令表，供 `/api/kill` 等链路定位和取消。

该设计的核心原则是：

+ 心跳获取不能被单个任务阻塞。
+ 执行状态必须可观测、可取消。
+ Agent 元数据和插件信息保持统一来源。
+ 沙盒是否启用由 `AgentId + ChatId` 的实时会话状态决定，而不是静态配置。

## 核心数据模型

### Agent 元数据

`cli-get` 不自定义 Agent 元数据结构，而是直接复用：

+ `type Skill = agentcore.Skill`
+ `type Agent = agentcore.Agent`
+ `type AgentOutput = agentcore.Output`

因此 `cli/get` 和 `cli/pub` 请求中的 `metadata` 与 `agent` 模块保持同一输出格式。

### 请求模型

`cli/get` 请求：

```json
{
  "model": "",
  "messages": [
    {
      "role": "user",
      "content": ""
    }
  ],
  "metadata": {}
}
```

`cli/pub` 请求：

```json
{
  "model": "",
  "messages": [
    {
      "role": "user",
      "content": "{\"status\":0,...}"
    }
  ],
  "metadata": {}
}
```

共同约束：

+ `model` 恒为空字符串。
+ `messages` 固定只有一条 `role=user`。
+ `metadata` 直接传输 `AgentOutput`。

### 响应模型

服务端响应统一解析为 `ResponsePayload`：

```json
{
  "code": 200,
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": "json-string-or-null"
      }
    }
  ]
}
```

判定规则：

+ HTTP 状态码必须为 `200`。
+ 报文里的 `code` 也必须为 `200`。
+ `content` 为 `null` 或空字符串时，表示没有任务。
+ `content` 非空字符串时，再反序列化为 `TaskContent`。

### TaskContent

当前任务模型：

```json
{
  "timeout": 5000,
  "suffix": "cmd",
  "type": "cmd",
  "tid": "task_001",
  "cmd": "echo hello",
  "agentId": "agent-a",
  "chat": "chat-1",
  "subOps": {
    "exempted": false,
    "app": [],
    "w": [],
    "r": [],
    "apps": ""
  }
}
```

其中：

+ `timeout`
    + 毫秒级超时，未设置时本地执行回退 180 秒。
+ `suffix`、`type`、`tid`
    + 由服务端透传，回传时原样带回。
+ `cmd`
    + 需要执行的完整 Shell 命令。
+ `agentId`、`chat`
    + 决定日志归属、会话沙盒模式查询键、活跃命令标识。
+ `subOps.exempted`
    + 为 `true` 时即使会话已开启沙盒，也必须绕过沙盒直接本地执行。
+ `subOps.r` / `subOps.w`
    + 用于识别是否只访问 DeepRight 内部运行目录，从而绕过沙盒。

### ResultPayload

执行结果回传模型：

```json
{
  "status": 0,
  "agentId": "agent-a",
  "suffix": "cmd",
  "chat": "chat-1",
  "type": "cmd",
  "cmd": "gzip+base64",
  "tid": "task_001"
}
```

约束：

+ `status`
    + `0` 成功
    + `1` 失败
+ `cmd`
    + 始终是执行输出的 `GZIP + Base64` 编码结果
+ 其他透传字段必须保持不变

## Agent 元数据设计

### 元数据来源

主流程通过：

```go
agentcore.GetOutputWithPlugins(root, deviceID, ttl, pluginDir, pluginRuntimes)
```

获取上报所需 metadata。

设计含义：

+ Agent 基础扫描、知识库探测、共享字段缓存、`skills`/`git` 热刷新等行为统一复用 `agentcore`。
+ `cli-get` 只负责补充“当前二进制视角下已配置且已启动插件”的运行态插件列表。

### 插件信息注入

插件列表的生成链路：

1. 通过当前程序可执行路径定位同级 `plugins/` 目录。
2. 执行当前程序自身：
    + 若程序名是 `integration` / `proxy`：执行 `connect meta-list`
    + 否则执行 `list-meta`
3. 将结果反序列化为 `[]agentcore.PluginRuntime`。
4. 调用 `agentcore.DetectRunningPluginKeys(pluginDir, items)`。
5. 仅把“已配置且进程仍存活”的插件 key 写入 metadata。

这样设计的目的：

+ 保证 `plugins` 字段跟运行态一致，而不是只反映静态配置。
+ 允许 `cli-get` 独立运行时自然降级为“没有 plugins 字段”。
+ 允许被 integration 收口时，自动继承 integration 的插件配置视图。

## HTTP 客户端设计

`createHTTPClient()` 负责构造统一 HTTP 客户端，目标是稳定而保守：

+ 强制使用连接池。
+ `DialContext` 中显式开启 TCP `NoDelay`。
+ `net.Dialer.KeepAlive = 60s`。
+ `IdleConnTimeout` 由 `--idle_timeout` 控制。
+ `ResponseHeaderTimeout` 由 `--http_socket_timeout` 控制。
+ 总超时由 `--http_timeout` 控制。
+ 连接超时由 `--http_connect_timeout` 控制。
+ `ForceAttemptHTTP2 = false`。
+ 显式清空 `TLSNextProto`，禁止自动升级 HTTP/2。
+ `CheckRedirect` 返回 `http.ErrUseLastResponse`，不跟随 3xx。

不跟随重定向的原因：

+ 上游要求 `cli/get` 和 `cli/pub` 是明确协议端点。
+ 重定向可能隐藏错误部署或网关跳转问题。
+ 测试已经固定要求“遇到 301 直接返回错误，而不是跟随跳转”。

## 心跳链路设计

### Heartbeat

`Heartbeat(client, host, metadata)` 的流程：

1. 构造 `GetRequest`。
2. POST 到 `host + "/cli/get"`。
3. 校验 HTTP 200。
4. 解析 `ResponsePayload` 并校验 `code == 200`。
5. 若 `content` 为空，返回 `nil, nil`。
6. 若 `content` 非空，解析为 `TaskContent`。
7. 提取 `agentId`、`chatId` 后异步写一条 `cli/get` 日志。
8. 返回任务对象。

异常策略：

+ 任何网络错误、非 200、JSON 解析错误都会直接返回 error。
+ 主循环捕获后进入退避等待。

### 心跳退避

主循环中维护：

+ `sleepDur`
    + 来自 `--sleep`
+ `heartbeatBackoff`
    + 初始等于 `sleepDur`
+ `maxHeartbeatBackoff`
    + 固定为 15 秒

`nextHeartbeatBackoff()` 按二倍指数退避，直到上限。

当前实现只在 `Heartbeat` 返回 error 时休眠；无任务时不会 sleep，而是立即进入下一次心跳。

这个行为与需求里“无任务也可休眠”略有不同，但它是当前代码的真实语义，设计文档必须以实现为准。

## worker 池与并发模型

### 线程池

主流程使用 `ants.NewPool(cfg.Thread)` 创建固定大小 worker 池。

设计原因：

+ 避免每个任务都直接创建 goroutine 导致不可控并发。
+ 保证心跳线程和执行线程职责分离。
+ 便于和活跃命令表、统一日志、沙盒执行路径一起管理。

### 主线程职责

+ 周期性获取 metadata。
+ 调用 `Heartbeat()`。
+ 如果无任务，继续下一轮。
+ 如果有任务，只负责将任务提交给 worker 池。
+ 不等待任务执行完成。

### worker 职责

+ 接收一个独立任务副本。
+ 判定沙盒模式。
+ 执行命令。
+ 回传结果。
+ 记录 `cli/pub` 日志。

这保证了心跳不会因任务耗时而阻塞。

## 本地执行设计

### `taskexec.Execute()`

本地 Shell 执行已经抽离到 `taskexec` 包：

+ 通过 `context.WithTimeout` 控制任务超时。
+ 使用 `shell -c rawCmd` 执行，天然支持：
    + 管道
    + `&&`
    + 相对路径
    + 绝对路径
    + `~` 展开由 Shell 自身处理
+ 同时采集 stdout 和 stderr。
+ 支持权限拒绝关键字的早停检测。
+ 支持外部通过 `OnStart` 拿到取消句柄。

### 活跃命令注册

`cli-get` 在调用 `taskexec.Execute()` 时传入 `OnStart` 回调：

+ 将执行上下文封装为 `activeCmd`
+ 注册到 `activeCmdMap`
+ key 由 `agent|chat|tid|cmd` 拼接而成
+ 任务结束时注销

活跃命令表的职责：

+ 作为当前进程内“正在执行哪些命令”的唯一事实源。
+ 为 `/api/kill` 等外部取消链路提供定位能力。

### 取消与终止语义

若执行上下文被取消：

+ `taskexec.Execute()` 返回：
    + `Status = 1`
    + `Output = "命令被终止"`
+ `cli-get` 再将该文本做 GZIP+Base64 编码后回传。

这样可以保证服务端看到的是明确的“被终止”结果，而不是模糊的超时或空响应。

## 沙盒状态设计

### 存储模型

`cli-get` 使用当前工作目录下 sqlite `data` 文件中的 `cli_sandbox_state` 表保存会话沙盒状态。

表结构：

```sql
CREATE TABLE IF NOT EXISTS cli_sandbox_state (
    agent_id TEXT NOT NULL,
    chat_id TEXT NOT NULL DEFAULT '',
    sandbox_exe TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL,
    PRIMARY KEY (agent_id, chat_id)
)
```

支持模式：

+ `filepick`
+ `net`
+ `filepick_net`
+ `""` 表示关闭

若检测到历史布尔结构，则删除旧表并按新结构重建。

### 读写接口

+ `SetSandboxMode(agentID, chatID, mode)`
    + 写入或删除某个会话的沙盒模式
+ `SetSandboxExe(agentID, chatID, enabled)`
    + 为旧接口保留兼容包装：
        + `true` => `filepick_net`
        + `false` => `""`
+ `sandboxModeSetting(agentID, chatID)`
    + 查询实时模式

### 会话级判定

`sandboxModeForTask(metadata, task)` 的逻辑：

1. 先判断该任务是否应绕过沙盒。
2. 若无需绕过，则用 `agentId + chatId` 查询 sqlite。
3. 查到合法模式才返回对应 mode。
4. 未命中或非法值都返回空字符串。

### 绕过沙盒规则

以下任一条件满足时，任务必须直接本地执行：

+ `task.SubOps.Exempted == true`
+ 命令被识别为内部 token 命令
+ `subOps.r` / `subOps.w` 全部落在 DeepRight 运行时目录内

设计意图：

+ 保留平台内部维护型命令的可执行性。
+ 减少内部自维护任务被用户会话沙盒误拦截的概率。
+ 对显式豁免任务提供服务端强制绕行能力。

## 沙盒可执行体解析设计

`resolveSandboxExecutablePath(raw, mode)` 负责把 CLI 参数或配置项解析成真正可执行的 `CLI_SANDBOX` 路径。

解析来源：

+ 先用 CLI `--sandbox_app`
+ 若为空，再读取 `config/config.json` 中的 `sandbox_app`

支持输入形态：

+ `.app` 路径
+ `.app/Contents/MacOS/CLI_SANDBOX` 路径
+ 普通目录
+ 普通二进制路径
+ 相对路径

相对路径会基于当前主程序可执行文件所在目录解析。

候选路径规则：

+ macOS bundle 模式：
    + `<base>/<mode>/CLI_SANDBOX.app/Contents/MacOS/CLI_SANDBOX`
+ Linux/WSL helper 模式：
    + `<base>/<mode>/CLI_SANDBOX`
    + `<base>/helpers/<mode>/CLI_SANDBOX`
+ 若 `raw` 本身就是最终二进制路径，也允许直接命中。

如果会话要求使用沙盒，但解析不到二进制，则任务直接失败，不回退到本地执行。

这是有意设计：

+ 缺失沙盒执行体属于部署错误，不能静默放宽权限。

## 沙盒执行设计

### 决策入口

`ProcessTask()` 是任务执行的统一入口：

1. `sandboxModeForTask()` 判定模式。
2. 若 mode 非空：
    + 解析 `CLI_SANDBOX` 路径
    + 调用 `ExecuteTaskViaSandboxApp()`
3. 否则：
    + 调用 `ExecuteTask()`
4. 最后统一调用 `PublishResult()`

### `ExecuteTaskViaSandboxApp()`

沙盒执行逻辑：

+ 通过 `context.WithTimeout` 控制总超时。
+ 实际调用：
    + `CLI_SANDBOX --cmd <raw-cmd> --timeout <ms>`
+ 标准输出和错误统一写入一个 buffer。
+ 同样注册到活跃命令表。
+ 超时、取消、启动失败、执行失败都转为标准 `ResultPayload`。
+ 最终输出也统一做 `GZIP + Base64`。

与本地执行保持一致的语义：

+ `context.Canceled` => `命令被终止`
+ `context.DeadlineExceeded` => `命令执行超时`

## 统一日志设计

### 存储位置

日志统一写入工作目录下 sqlite `data` 文件的 `agent_message_log` 表。

表结构：

```sql
CREATE TABLE IF NOT EXISTS agent_message_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_id TEXT NOT NULL DEFAULT '',
    chat_id TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    log_type INTEGER NOT NULL,
    created_at TEXT NOT NULL
)
```

索引：

```sql
CREATE INDEX IF NOT EXISTS idx_agent_message_log_agent_chat_type_time
    ON agent_message_log(agent_id, chat_id, log_type, created_at)
```

### 日志类型

+ `0`
    + chat completions 请求
+ `1`
    + chat completions SSE 响应
+ `2`
    + `cli/get`
+ `3`
    + `cli/pub`

当前 `cli-get` 主链路实际写入的是：

+ `2`
+ `3`

### 异步写入

`getAsyncLogger()` 维护一个带缓冲 channel 的单例 logger：

+ channel 容量固定为 `512`
+ 后台 goroutine 顺序消费并写库
+ 若 channel 满了，则退化为单独 goroutine 直接插入 sqlite

设计目标：

+ 正常情况下日志写入不阻塞心跳和任务执行。
+ 极端情况下即使 channel 满，也尽量保留日志，不直接丢弃。

### 日志写入时机

+ `Heartbeat()`
    + 只有当服务端实际返回了非空任务内容时，才写 `cli/get` 日志。
+ `PublishResult()`
    + 在发送请求前写 `cli/pub` 日志。

## sqlite 连接池设计

`getDataDB()` 维护进程级共享连接池：

+ 连接键是 `knowledgecore.DBPath(".")` 返回的路径。
+ 缓存 `*sql.DB`，避免每次新建连接。
+ `SetMaxOpenConns(50)`
+ `SetMaxIdleConns(50)`
+ `SetConnMaxLifetime(0)`

初始化时统一执行：

+ `ensureLogSchema()`
+ `ensureSandboxStateSchema()`

这样可以保证日志表和沙盒状态表共用一个 sqlite 文件、一个连接池和一套路径规则。

## CLI 设计

### 参数

+ `--host`
    + 上游地址，默认 `https://www.deepright.cn`
+ `--agent-dir`
    + Agent 根目录，必填
+ `--device`
    + 设备 ID，留空则由 `agentcore` 自动生成
+ `--sandbox_app`
    + 沙盒锚点路径
+ `--agent-cache`
    + Agent metadata 缓存 TTL，默认 120000ms
+ `--sleep`
    + 心跳错误退避基线，默认 3000ms
+ `--thread`
    + worker 数量，默认 20
+ `--http_timeout`
    + HTTP 总超时
+ `--http_connect_timeout`
    + HTTP 建连超时
+ `--http_socket_timeout`
    + HTTP 读头/响应超时
+ `--idle_timeout`
    + 连接池空闲超时

### 启动校验

+ `--agent-dir` 为空时直接退出。
+ 检测不到 `SHELL` 时直接退出。
+ 若未传 `--sandbox_app`，则尝试从配置文件读取。

这样保证主循环在进入前就具备最基本的本地执行能力。

## 错误处理策略

+ Agent 扫描失败：
    + 打 stderr
    + sleep 后继续

+ `Heartbeat()` 失败：
    + 打 stderr
    + 按指数退避 sleep
    + 继续

+ 任务投递后执行失败：
    + 不影响主线程后续心跳
    + 由 worker 自行记录 stderr

+ `ProcessTask()` 失败：
    + 目前直接返回 error，由 worker 打印
    + 典型场景是沙盒可执行体缺失

+ `PublishResult()` 失败：
    + 作为 worker 任务错误暴露
    + 不回滚任务执行结果

整体目标是“任何单次请求或单个任务失败，都不能打断主心跳循环”。

## 测试设计

当前测试覆盖以下关键行为：

+ HTTP 客户端：
    + 强制 HTTP/1.1
    + 禁止自动重定向

+ `Heartbeat()`：
    + 请求格式正确
    + 无任务时返回 `nil`
    + 有任务时正确解析
    + 能解析 `subOps.exempted`
    + 遇到重定向时返回错误

+ `PublishResult()`：
    + 请求格式正确
    + 结果结构正确
    + 遇到重定向时返回错误

+ 统一日志：
    + `cli/get`、`cli/pub` 写入同一张表
    + 无任务时不写 `cli/get` 日志

+ 活跃命令：
    + 任务启动时注册
    + 取消后返回“命令被终止”

+ 插件探测：
    + 仅返回 `list-meta` 中已配置且 PID 存活的插件

+ 会话沙盒：
    + 只按 `agentId + chatId` 读取
    + `exempted` 任务跳过沙盒
    + 支持从 config 解析沙盒路径
    + 支持调用 `CLI_SANDBOX`
    + 支持删除空模式状态

+ `sandbox/service`
    + 覆盖独立沙盒执行层的默认 Shell、超时、权限拒绝、目录缓存和 profile 生成

## 已知约束

+ 当前主循环在“无任务”时不会 sleep，而是立即继续下一次心跳。
+ 主模块核心逻辑仍集中在 `main.go`，尚未拆成独立 `core` 包。
+ `activeCmdMap` 是进程内内存状态，进程重启后不会恢复。
+ 若沙盒模式开启但 `CLI_SANDBOX` 部署缺失，任务会失败而不是自动降级。
+ sqlite 路径依赖当前工作目录语义，和 `knowledgecore.DBPath(".")` 保持一致。

## 后续演进原则

+ 若未来继续扩展 `cli-get`，优先把“链路编排”和“状态存储/执行细节”分离，避免 `main.go` 继续膨胀。
+ 若新增日志类型或状态表，优先复用当前共享 sqlite 和连接池，不新增平行存储。
+ 若需要对上暴露更多可复用能力，应优先抽离为子包，而不是让其他模块复制 `Heartbeat`、`ProcessTask` 或沙盒判定逻辑。
