# Cron 模块详细技术设计

## 设计目标

+ 将本地一次性任务与周期任务的持久化、拆分、补齐逻辑收敛到统一模块，避免上层模块分别维护“任务元数据”和“未来待执行明细”的生成规则。
+ 把 `cron` 设计成“可复用 service + CLI 包装层”的结构，使 `integration` 或其他模块既可以直接复用服务能力，也可以通过命令行统一收口。
+ 在不存储模型 token 的前提下，提供足够的运行时校验：
    + Agent 必须存在
    + Model 必须已注册且 token 非空
+ 固化“任务元数据 -> 未来 5 天任务明细”的窗口化预生成策略，保证长期运行不会漏生成任务。

## 模块边界

+ [main.go](/Users/shenjiawei/Documents/code/deepright/cli/module/cron/main.go)
    + 同时包含 `CronService` 核心实现和 CLI 入口。
    + 当前还没有独立 `croncore` 子包，内核和入口仍在同一文件中。

+ 依赖模块
    + `knowledge/knowledgecore`
        + 用于解析共享 sqlite `data` 的路径。
    + `github.com/robfig/cron/v3`
        + 用于解析自定义 cron 表达式。
    + `github.com/glebarez/go-sqlite`
        + SQLite 驱动。

## 总体架构

当前 `cron` 模块是“单服务对象 + SQLite 持久化 + 周期性补齐协程”的结构：

1. `CronService` 持有一份 sqlite 连接和互斥锁。
2. 调用 `Submit*()` 或 `SubmitCron*()` 创建任务元数据。
3. 对于需要立即展开的周期任务，提交时直接生成首批任务明细。
4. `Start()` 启动一个每分钟执行一次的 ticker。
5. ticker 调用 `CheckAndCreate()`，为所有非一次性任务补齐未来 5 天窗口内缺失的明细。
6. 所有元数据和明细的读写动作都写入审计日志表。

该设计的核心原则：

+ 元数据是长期事实源。
+ 明细是未来一段时间的可执行展开视图。
+ 周期任务不能只靠“即时生成一次”，必须持续滚动补齐。
+ 任务执行所需 token 不属于 `cron` 的持久化边界。

## 核心数据模型

### TaskMeta

`TaskMeta` 表示任务元数据：

```json
{
  "id": 1,
  "cycle": 0,
  "rawTime": "2026-04-30 12:10",
  "agentId": "A",
  "chatId": "chat-001",
  "type": "cron",
  "model": "OpenAI",
  "thinking": true,
  "router_disable": true,
  "cron": "0 10 12 30 4 ? 2026",
  "content": "查看天气",
  "responseSchema": "{\"type\":\"object\"}"
}
```

字段语义：

+ `cycle`
    + `0` 仅一次
    + `1` 工作日
    + `2` 自然日
    + `3` 每小时
    + `4` 每 15 分钟
    + `5` 每 30 分钟
    + `-1` 自定义 cron
+ `rawTime`
    + 内置周期任务的起始时间；自定义 cron 时为空字符串。
+ `type`
    + 默认 `cron`
    + 也允许上层传入 `FEISHU` 等外部来源类型
+ `router_disable`
    + 当前标准字段，`true` 表示关闭路由
+ `responseSchema`
    + 可选 LLM Response JSON Schema

### TaskDetail

`TaskDetail` 表示某个具体执行时刻的任务明细：

```json
{
  "id": 1,
  "metaId": 1,
  "execTime": 1777522200,
  "agentId": "A",
  "chatId": "chat-001",
  "type": "cron",
  "model": "OpenAI",
  "thinking": true,
  "router_disable": true,
  "content": "查看天气",
  "responseSchema": "{\"type\":\"object\"}",
  "started": 0
}
```

字段语义：

+ `metaId`
    + 指向所属 `TaskMeta`
+ `execTime`
    + Unix 秒级时间戳
+ `started`
    + `0` 未启动
    + `1` 已启动
    + `2` 无需启动
    + `3` 已完成
`TaskDetail` 继承自元数据的字段包括：

+ `agentId`
+ `chatId`
+ `type`
+ `model`
+ `thinking`
+ `router_disable`
+ `content`
+ `responseSchema`

### 日志模型

模块内部还维护两类审计日志结构体：

+ `cronMetaLogEntry`
+ `cronDetailLogEntry`

它们不是外部 API 模型，而是用于把每次 select / insert 等数据库动作记录进日志表。

## 任务周期模型

### 内置周期

`buildCron(cycle, t)` 的转换规则：

+ `0`
    + `0 分 时 日 月 ? 年`
+ `1`
    + `0 分 时 * * 1-5`
+ `2`
    + `0 分 时 * * ?`
+ `3`
    + `分 * * * *`
+ `4`
    + `*/15 * * * *`
+ `5`
    + `*/30 * * * *`

注意：

+ 仅一次、工作日、自然日生成的是带秒或带 `?` / 年份的扩展表达式字符串。
+ 高频周期和自定义表达式使用的是 5 字段风格。
+ 当前模块自己只在“展示/持久化”层保存这个 cron 字符串；实际展开逻辑并不统一依赖它。

### 自定义 Cron

+ 自定义 cron 对应 `cycle = -1`
+ `rawTime` 为空
+ `cron` 直接保存调用方传入表达式
+ 提交时不立即创建明细
+ 由 `CheckAndCreate()` 统一用 `robfig/cron` 解析并生成未来窗口内的执行点

## SQLite 设计

### 数据库路径

作为子模块使用时，调用方可自行传入任意 DB 路径：

```go
NewCronService(dbPath)
```

CLI 模式下，当前实现固定通过：

```go
knowledgecore.DBPath(".")
```

解析数据库路径。

这意味着：

+ CLI 默认和当前工作目录对应的共享 `data` sqlite 协同工作。
+ 它并没有自己定义独立的 DB 路径解析规则。

### 服务对象与连接

`NewCronService(dbPath)` 会：

1. `sql.Open("sqlite", dbPath)`
2. 创建 `CronService`
3. 调用 `initDB()`

当前实现特点：

+ 每个 `CronService` 自己持有一份 `*sql.DB`
+ 没有像 `connect` 那样做跨 service 的共享连接池
+ 并发安全主要依赖 `CronService.mu`

### 表结构

#### `task_meta`

```sql
CREATE TABLE IF NOT EXISTS task_meta (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cycle INTEGER NOT NULL,
    raw_time TEXT NOT NULL DEFAULT '',
    agent_id TEXT NOT NULL,
    chat_id TEXT NOT NULL DEFAULT '',
    task_type TEXT NOT NULL DEFAULT 'cron',
    model TEXT NOT NULL,
    thinking INTEGER NOT NULL DEFAULT 0,
    router_disable INTEGER NOT NULL DEFAULT 1,
    cron TEXT NOT NULL,
    content TEXT NOT NULL,
    response_schema TEXT NOT NULL DEFAULT ''
)
```

#### `task_detail`

```sql
CREATE TABLE IF NOT EXISTS task_detail (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    meta_id INTEGER NOT NULL,
    exec_time INTEGER NOT NULL,
    agent_id TEXT NOT NULL,
    chat_id TEXT NOT NULL DEFAULT '',
    task_type TEXT NOT NULL DEFAULT 'cron',
    model TEXT NOT NULL,
    thinking INTEGER NOT NULL DEFAULT 0,
    router_disable INTEGER NOT NULL DEFAULT 1,
    content TEXT NOT NULL,
    response_schema TEXT NOT NULL DEFAULT '',
    started INTEGER NOT NULL DEFAULT 0,
    UNIQUE(meta_id, exec_time)
)
```

这个唯一键是“防重复创建明细”的核心约束。

#### `cron_meta_log`

```sql
CREATE TABLE IF NOT EXISTS cron_meta_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    meta_id INTEGER NOT NULL,
    agent_id TEXT NOT NULL,
    chat_id TEXT NOT NULL DEFAULT '',
    task_type TEXT NOT NULL DEFAULT 'cron',
    action TEXT NOT NULL,
    cycle INTEGER NOT NULL,
    raw_time TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL,
    thinking INTEGER NOT NULL DEFAULT 0,
    router_disable INTEGER NOT NULL DEFAULT 1,
    cron TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    response_schema TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
)
```

#### `cron_detail_log`

```sql
CREATE TABLE IF NOT EXISTS cron_detail_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    detail_id INTEGER NOT NULL,
    meta_id INTEGER NOT NULL,
    agent_id TEXT NOT NULL,
    chat_id TEXT NOT NULL DEFAULT '',
    task_type TEXT NOT NULL DEFAULT 'cron',
    action TEXT NOT NULL,
    exec_time INTEGER NOT NULL,
    model TEXT NOT NULL,
    thinking INTEGER NOT NULL DEFAULT 0,
    router_disable INTEGER NOT NULL DEFAULT 1,
    content TEXT NOT NULL,
    response_schema TEXT NOT NULL DEFAULT '',
    started INTEGER NOT NULL DEFAULT 0,
    occurred_at TEXT NOT NULL
)
```

### 索引设计

当前实现会创建：

+ `idx_detail_agent_exec_status`
    + `task_detail(agent_id, exec_time, started)`
+ `idx_detail_exec_status`
    + `task_detail(exec_time, started)`
+ `idx_meta_agent`
    + `task_meta(agent_id)`
+ `idx_cron_meta_log_agent_chat_time`
    + `cron_meta_log(agent_id, chat_id, created_at)`
+ `idx_cron_meta_log_meta_time`
    + `cron_meta_log(meta_id, created_at)`
+ `idx_cron_detail_log_agent_chat_time`
    + `cron_detail_log(agent_id, chat_id, occurred_at)`
+ `idx_cron_detail_log_meta_time`
    + `cron_detail_log(meta_id, occurred_at)`
+ `idx_cron_detail_log_detail_time`
    + `cron_detail_log(detail_id, occurred_at)`

### 迁移策略

`ensureSchema()` 采用轻量迁移策略：

+ 若旧日志表存在：
    + `task_meta_log -> cron_meta_log`
    + `task_detail_log -> cron_detail_log`
+ 通过一组 `ALTER TABLE` 补齐新增列：
    + `chat_id`
    + `task_type`
    + `response_schema`
    + `router_disable`
    + `started`
+ 删除旧索引并重建新索引
+ 用 `routerDisableSelect()` 兼容缺少 `router_disable` 列的历史库

设计目标：

+ 保持本地 sqlite 数据可原地升级
+ 不引入复杂迁移框架

## 审计日志设计

### 记录范围

当前实现会对以下操作写日志：

+ 元数据 `insert`
+ 元数据 `select`
+ 明细 `insert`
+ 明细 `select`

并非只有“写操作”才记录；读取行为同样会进入审计表。

### 记录时机

+ `insert()`
    + 插入 `task_meta` 后立即写一条 meta log
+ `createDetail()`
    + 明细插入成功后写一条 detail log
+ `CheckAndCreate()`
    + 每次扫描到某条 meta 时会先写一条 `select` meta log
+ `GetMetas()` / `GetDetails()`
    + 每返回一条记录前都写一条对应的 `select` 日志

这个设计比较重，但它换来的好处是：

+ 可以完整追踪明细何时被创建、何时被查询。
+ 适合和其他模块联动排查任务链路。

## 核心流程设计

### 提交周期任务

以 `SubmitWithTypeSchemaAndRouterDisable()` 为统一入口：

1. 解析 `rawTime`
2. 调用 `buildCron()`
3. 调用 `insert(..., createFirst=true)`
4. `insert()` 写入 `task_meta`
5. `createInitialDetails()` 根据周期类型生成首批明细

### 提交自定义 Cron 任务

以 `SubmitCronWithTypeSchemaAndRouterDisable()` 为统一入口：

1. 校验 cron 字符串非空
2. 调用 `insert(..., cycle=-1, createFirst=false)`
3. 只写元数据，不立刻生成明细
4. 等待后续 `CheckAndCreate()` 补齐

### 明细创建

`createDetail(meta, execTime)` 负责单点创建：

+ 插入语句使用 `INSERT OR IGNORE`
+ 由 `UNIQUE(meta_id, exec_time)` 保证幂等
+ 只有 `RowsAffected > 0` 时才认定为新建成功并写日志

这意味着：

+ 同一个元数据在同一执行时间不会重复生成 detail
+ 周期补齐可以放心反复执行

## 首批明细生成策略

`createInitialDetails(meta, base)` 的策略：

+ `cycle=0`
    + 只创建基准时间这一条
+ `cycle=1`
    + 创建从 `base` 起未来 5 天窗口中的工作日明细
+ `cycle=2`
    + 创建从 `base` 起未来 5 天窗口中的自然日明细
+ `cycle=3/4/5`
    + 根据间隔立即补齐未来 5 天内的所有高频明细
+ `cycle=-1`
    + 不走这里

这里的设计目的是：

+ 高频任务不能只依赖每分钟补齐器“慢慢补”
+ 创建后就要立刻具备足够长的未来执行窗口

## 周期补齐设计

### `Start()`

`Start()` 启动一个 goroutine：

1. 先立即执行一次 `CheckAndCreate()`
2. 再创建 `time.NewTicker(1 * time.Minute)`
3. 每分钟执行一次 `CheckAndCreate()`
4. 收到 `stopCh` 后退出

### `CheckAndCreate()`

`CheckAndCreate()` 是整个模块的核心维护动作：

1. 加锁
2. 记录 `LastCheckTime = now`
3. 查询全部 `task_meta`
4. 对每条元数据做日志记录
5. 根据其周期类型生成未来 5 天窗口内缺失的 detail

窗口边界：

+ `now = time.Now()`
+ `end = now + 5*24h`

### 不同周期的补齐策略

#### 一次性任务

+ 若原始时间 `t` 满足：
    + `t >= now`
    + `t < end`
  则尝试创建该 detail
否则不补。

#### 工作日 / 自然日任务

通过 `createRange()` 处理：

+ 起点取 `max(baseTime, now)`
+ 每天在原始时分上生成一个候选执行时间
+ 工作日模式会额外经过 `isWorkday()`

#### 高频周期任务

通过 `createIntervalRange()` 处理：

+ 起点为 `now.Truncate(time.Minute)`
+ 每隔 `1h / 15m / 30m` 生成一个候选点
+ 直到超过窗口结束

#### 自定义 Cron 任务

通过 `createFromCron()` 处理：

+ 使用 `robfig/cron` 解析 cron
+ 先把 Quartz 风格 `?` 替换成 `*`
+ 从 `sched.Next(now - 1s)` 开始向后滚动
+ 直到超过窗口结束

## 并发模型

### 锁设计

`CronService` 只有一把 `mu sync.Mutex`，保护：

+ 元数据插入
+ 明细生成
+ 周期检查
+ 元数据查询
+ 明细查询

设计含义：

+ 逻辑简单，避免同一实例内部竞争导致重复生成 detail
+ 代价是所有读写都串行

### 停止语义

`Stop()` 当前做两件事：

+ `close(stopCh)`
+ `db.Close()`

注意：

+ 这是一次性关闭语义
+ 同一个 `CronService` 不能安全重复 `Start/Stop/Start`

## CLI 设计

### 命令

当前实现支持：

+ `start`
+ `create`
+ `submit`
+ `create-cron`
+ `submit-cron`
+ `list`
+ `check`
+ `help`

### 参数解析

`parseArgs()` 兼容：

+ `--key=value`
+ `--key value`
支持的关键参数：

+ `--cycle`
+ `--time` / `--rawTime`
+ `--agent` / `--agentId`
+ `--agent-dir`
+ `--chat` / `--chatId`
+ `--type`
+ `--model`
+ `--content`
+ `--schema`
+ `--cron`
+ `--thinking`
+ `--router_disable`

布尔值行为：

+ `--thinking` 不带值时视为 `true`
+ `--router_disable` 不带值时视为 `true`
当前代码没有实现 `--swarm` 兼容解析，外部统一应以 `--router_disable` 为准。

### CLI 校验

创建任务前会做三类校验：

#### 1. 参数校验

+ `validateCreateArgs()`
+ `validateCreateCronArgs()`

#### 2. Agent 校验

`resolveAgentDir()` 的查找顺序：

+ 显式 `--agent-dir`
+ 环境变量 `AGENT_DIR`
+ 当前目录下 `./agents`

`validateAgentExists()` 支持两种情况：

+ 传的是 Agent 根目录
+ 传的已经是某个具体 Agent 目录

#### 3. Model 校验

`validateRegisteredModel(model)` 会：

+ 通过 `knowledgecore.DBPath(".")` 找到共享 sqlite
+ 确保 `token_store` 表存在
+ 查询指定 model 的 token
+ 要求模型存在且 token 非空

设计目标：

+ `cron` 自己不存 token
+ 但要确保引用的模型在系统里是可执行的

## 子模块 API 设计

当前对上层最重要的 API 是：

+ `NewCronService(dbPath)`
+ `Submit(...)`
+ `SubmitWithType(...)`
+ `SubmitWithTypeAndSchema(...)`
+ `SubmitWithTypeSchemaAndRouterDisable(...)`
+ `SubmitCron(...)`
+ `SubmitCronWithType(...)`
+ `SubmitCronWithTypeAndSchema(...)`
+ `SubmitCronWithTypeSchemaAndRouterDisable(...)`
+ `CheckAndCreate()`
+ `Start()`
+ `Stop()`
+ `GetMetas()`
+ `GetDetails()`

设计特点：

+ API 通过不同函数重载暴露“默认 type / schema / router_disable”与“显式指定”两类能力。
+ 最终都收敛到两个底层入口：
    + `insert()`
    + `createInitialDetails() / CheckAndCreate()`

## 与其他模块的关系

### 与 Agent 模块

+ `cron` 不直接读取 Agent metadata
+ 只校验某个 `agentId` 对应目录是否存在
因此它依赖的是 Agent 目录约定，而不是 `agentcore` 的完整元数据输出。

### 与 Knowledge 模块

+ 通过 `knowledgecore.DBPath(".")` 获取共享 sqlite 路径
这让 `cron` 可以和系统其他模块共享：

+ `token_store`
+ 以及默认 `data` 文件位置

### 与 Integration / Connect

+ `type` 字段允许写 `cron` 之外的来源，例如 `FEISHU`
+ `responseSchema` 和 `router_disable` 的设计本身也是为了让上层任务系统复用
但当前模块本身只负责“存储与拆分”，不负责真正执行这些 detail。

## 错误处理策略

+ 时间解析失败：
    + 直接返回 `invalid time format`
+ 不支持的周期：
    + 直接返回 `unsupported cycle`
+ Agent 不存在：
    + 返回 `agent not found`
+ Model 未注册或 token 为空：
    + 返回 `model not registered` / `model token is empty`
+ 自定义 cron 解析失败：
    + `createFromCron()` 当前静默跳过，不抛出外部错误
+ 明细重复：
    + 通过 `INSERT OR IGNORE` 静默处理

整体目标是：

+ 提交路径尽量严格校验
+ 周期补齐路径尽量幂等、容错

## 测试设计

当前测试覆盖以下关键行为：

+ `buildCron()` 的内置周期表达式生成
+ 一次性、工作日、自然日、高频周期、自定义 cron 的提交与首批明细生成
+ 自定义 cron 在 `CheckAndCreate()` 后补齐 detail
+ 重复补齐不产生重复 detail
+ `chatId` 持久化与继承
+ 自定义 `type` 持久化与继承
+ `responseSchema` 持久化与继承
+ `router_disable` 持久化与继承
+ 高频任务生成大窗口 detail
+ CLI 布尔参数解析
+ schema 迁移补列与重建索引
这些测试共同验证：

+ 窗口化补齐逻辑是稳定且幂等的
+ 历史数据库可以平滑升级
+ 新引入字段确实会从元数据继承到明细

## 已知约束

+ 当前核心实现和 CLI 入口都集中在 `main.go`，还未拆成共享子包。
+ `CronService` 使用单互斥锁，读写完全串行。
+ `Stop()` 会直接关闭 DB，实例不可安全复用。
+ 自定义 cron 的 `CheckAndCreate()` 解析失败时只会跳过，不会把错误暴露给调用方。
+ 当前代码没有实现真正的 HTTP 服务端接口，尽管旧手册中仍有 HTTP API 描述。
+ CLI 当前只支持 `router_disable`，不支持旧文档里提到的 `--swarm` 别名。

## 后续演进原则

+ 若后续需要被更多模块复用，优先把 `CronService` 抽到独立子包，保持 `main.go` 薄包装。
+ 若未来要把 detail 执行链路也纳入 `cron`，应继续保持“元数据/明细生成”和“任务执行”两层解耦。
+ 若需要扩展更多周期类型，优先统一到：
    + `buildCron()`
    + `cycleInterval()`
    + `createInitialDetails()`
    + `CheckAndCreate()`
  这四个核心策略点中，而不是在多个入口分散加特判。
