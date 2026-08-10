# Cron 定时任务模块

## 简介

管理本地一次性任务或周期任务的执行。支持六种内置周期（仅一次、工作日、自然日、每小时、每15分钟、每30分钟）和自定义 Cron 表达式。

## 存储

使用 SQLite 数据库文件 `data` 存储任务元数据和明细, 重启后数据不丢失。

注意：

- `cron` 只存任务元数据和任务明细，不存模型 Token（密钥）
- 如果调用方需要执行任务，应在执行时按模型动态提供或查询对应 Token
- 所有 `task_meta` / `task_detail` 的数据库操作都会额外写入审计日志表

## 子模块使用

```go
import "cron"

svc, err := cron.NewCronService("data")
svc.Start() // 启动每分钟周期检查

// 提交周期任务（自动生成 Cron 表达式）
meta, err := svc.Submit(
    0,                    // cycle: 0=仅一次, 1=工作日, 2=自然日, 3=每小时, 4=每15分钟, 5=每30分钟
    "2026-04-30 12:10",   // rawTime
    "AgentA",             // agentId
    "chat-001",           // chatId，可为空
    "OpenAI",             // model
    "查看天气",            // content
    true,                 // thinking
)

// 提交带 Response JSON Schema 的任务
meta, err = svc.SubmitWithTypeAndSchema(
    0,
    "2026-04-30 12:10",
    "AgentA",
    "chat-001",
    "cron",
    "OpenAI",
    "查看天气",
    "{\"type\":\"object\",\"properties\":{\"answer\":{\"type\":\"string\"}}}",
    true,
)

// 提交带 Response JSON Schema 和 swarm 的任务
meta, err = svc.SubmitWithTypeSchemaAndSwarm(
    0,
    "2026-04-30 12:10",
    "AgentA",
    "chat-001",
    "cron",
    "OpenAI",
    "查看天气",
    "{\"type\":\"object\",\"properties\":{\"answer\":{\"type\":\"string\"}}}",
    true,                 // swarm
    true,                 // thinking
)

// 提交自定义 Cron 表达式任务
meta, err := svc.SubmitCron(
    "10 12 * * 1-5",      // cron expression (5-field: min hour dom month dow)
    "AgentA",             // agentId
    "chat-001",           // chatId，可为空
    "OpenAI",             // model
    "查看天气",            // content
    true,                 // thinking
)

// 查询
metas := svc.GetMetas()
details := svc.GetDetails()

svc.Stop() // 停止周期检查
```

## CLI 使用

```bash
# 查看详细帮助
./cron --help

# 提交周期任务
./cron create --agent-dir ../agent/test-case --cycle=0 --time='2026-04-30 12:10' --agent=A --chat=chat-001 --model=OpenAI --thinking --content='查看天气'

# 提交带 Response JSON Schema 的任务
./cron create --agent-dir ../agent/test-case --cycle=0 --time='2026-04-30 12:10' --agent=A --chat=chat-001 --model=OpenAI --content='查看天气' --schema '{"type":"object","properties":{"answer":{"type":"string"}}}'

# 提交带 swarm 的任务
./cron create --agent-dir ../agent/test-case --cycle=0 --time='2026-04-30 12:10' --agent=A --chat=chat-001 --model=OpenAI --content='查看天气' --swarm

# 兼容空格分隔参数与 --rawTime / --thinking true 写法
./cron create --agent-dir ../agent/test-case --content "每15分钟检查一次上游接口健康" --model "OpenAI" --thinking true --swarm true --rawTime "2026-05-03 10:00" --cycle 4 --chatId "chat-001" --agent "A"

# 提交自定义 Cron 表达式任务
./cron create-cron --agent-dir ../agent/test-case --cron='10 12 * * 1-5' --agent=A --chat=chat-001 --model=OpenAI --thinking --content='查看天气'

# 查看所有任务
./cron list

# 手动执行一次周期检查
./cron check
```

## HTTP API

```
POST /api/cron/create?agentId=xxx
Content-Type: application/json

# 周期任务
{"content":"查看天气","model":"OpenAI","thinking":true,"swarm":true,"rawTime":"2026-04-30 12:10","cycle":0}

# 自定义 Cron 表达式
{"content":"查看天气","model":"OpenAI","thinking":true,"swarm":true,"cycle":-1,"cron":"10 12 * * 1-5"}
```

响应：`{"status":0,"id":1,"cron":"...","agentId":"xxx"}`

注意：

- `cron` 模块创建任务时不接收也不保存模型 token
- 调用方如果需要执行任务，应在执行链路里按 `model` 动态查询或注入 token

```
POST /api/cron/detail/metadata?agentId=xxx
```

响应：`{"status":0,"data":[{"id":1,"cycle":0,"rawTime":"...","agentId":"xxx","model":"...","thinking":false,"cron":"...","content":"..."}]}`

## 任务元数据

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | 自增 ID |
| cycle | int | 0=仅一次, 1=工作日, 2=自然日, 3=每小时, 4=每15分钟, 5=每30分钟, -1=自定义 Cron |
| rawTime | string | yyyy-MM-dd hh:mm（自定义 Cron 时为空） |
| agentId | string | 绑定的 Agent |
| chatId | string | 绑定的会话 ID，可为空 |
| model | string | 选择的模型 |
| thinking | boolean | 是否深度思考 |
| swarm | boolean | 是否启用 swarm，默认 `false` |
| cron | string | Cron 表达式（内置周期自动生成，自定义直接使用） |
| content | string | 任务内容 |
| responseSchema | string | 可选的 LLM Response JSON Schema，任务明细会继承该值 |

说明：

- `task_meta` 不包含 token / 密钥字段
- `task_detail` 也不包含 token / 密钥字段
- `model` 仅作为执行时查找 token 的键使用
- `swarm` 在元数据和明细中都不可为空，默认值为 `false`

## 数据库日志

新增两张审计日志表：

- `cron_meta_log`
- `cron_detail_log`

说明：

- 所有任务元数据与任务明细的数据库操作都会记录日志
- 当前至少覆盖创建、查询以及任务执行过程中的状态更新
- 日志按 `agent_id + chat_id + 时间` 建立索引，便于按 Agent、会话和时间追踪
- 如果历史库中已存在相关旧日志表，会自动改名为新的 `cron_meta_log` / `cron_detail_log`
- 如果历史库中已存在旧明细索引 `idx_detail_agent_time`、`idx_detail_agent_chat_time` 或 `idx_detail_agent_chat_time_type`，启动时会自动删除
- `task_detail` 主表索引已调整为以下两组：
  `idx_detail_agent_exec_status`：`agent_id + exec_time + started`
  `idx_detail_exec_status`：`exec_time + started`

索引：

```sql
CREATE INDEX IF NOT EXISTS idx_detail_agent_exec_status
ON task_detail(agent_id, exec_time, started);

CREATE INDEX IF NOT EXISTS idx_detail_exec_status
ON task_detail(exec_time, started);

CREATE INDEX IF NOT EXISTS idx_cron_meta_log_agent_chat_time
ON cron_meta_log(agent_id, chat_id, created_at);

CREATE INDEX IF NOT EXISTS idx_cron_detail_log_agent_chat_time
ON cron_detail_log(agent_id, chat_id, occurred_at);
```

## Cron 表达式规则

内置周期自动生成：
- 仅一次: `0 {分} {时} {日} {月} ? {年}`
- 工作日: `0 {分} {时} * * 1-5`
- 自然日: `0 {分} {时} * * ?`
- 每小时: `{分} * * * *`
- 每15分钟: `*/15 * * * *`
- 每30分钟: `*/30 * * * *`

自定义表达式使用 5 字段格式：`分 时 日 月 周`，例如 `10 12 * * 1-5` 表示工作日 12:10。

## 任务明细

- 周期任务：提交时立即创建首批明细
- 自定义 Cron：提交时不创建明细，由周期检查创建
- 工作日/自然日：提交时立即补齐后 5 天窗口内的明细
- 每小时/每15分钟/每30分钟：提交时立即补齐后 5 天窗口内按间隔展开的全部明细
- 周期检查（每分钟）会继续为所有非一次性任务补齐后续 5 天窗口内需要执行的明细，已存在则跳过
- 每条明细有启动状态：0=未启动、1=已启动、2=无需启动、3=已完成
- 每条明细包含 `chatId` 字段；如果创建元数据时显式传入 `chatId`，后续生成的任务明细会继承该值
- 每条明细包含 `responseSchema` 字段；周期拆分出的任务明细会继承任务元数据的 `responseSchema`
- 每条明细包含 `swarm` 字段；周期拆分出的任务明细会继承任务元数据的 `swarm`
- `task_meta` 与 `task_detail` 都包含 `type` 字段，默认值为 `cron`
- `type=cron` 表示备忘录 cron 创建的任务；如果未来由 Connect 创建，可写具体模块名，例如 `FEISHU`
- CLI 同时兼容 `--key=value` 与 `--key value` 两种传参方式
- `--time` 与 `--rawTime` 等价，`--thinking` 也支持显式写成 `--thinking true` 或 `--thinking false`
- `--swarm` 也支持显式写成 `--swarm true` 或 `--swarm false`
- CLI 支持 `--type` 指定任务类型；未传时默认 `cron`
- CLI 支持 `--schema` 指定 Response JSON Schema；未传时默认为空字符串
- 创建时会检查指定 `agentId` 是否存在；Agent 根目录优先取 `--agent-dir`，其次取环境变量 `AGENT_DIR`，最后回退当前目录下的 `./agents`
- `--agent-dir` 既支持传 Agent 根目录，也支持直接传某个具体 Agent 目录
- 创建时还会检查指定 `model` 是否已在共享 SQLite `token_store` 中注册，且对应 token 非空
- 上述校验只用于确认模型已注册；Cron 本身不会把 token 写入 `task_meta` 或 `task_detail`

## 编译

```bash
cd cli/module/cron
/opt/homebrew/bin/go build -o cron ./
```

已验证可编译产物：当前目录下已成功生成 `cron` 可执行文件。

## 测试

```bash
go test -v ./...
```


## Response Schema 规范与继承机制

在 `20260518-1` 迭代中，系统为任务元数据和任务明细全面引入了 `response_schema`（在代码/JSON中通常表示为 `responseSchema`）字段，用于支持结构化输出（Structured Outputs）。

### 1. 字段定义
- **任务元数据 (Task Metadata)**:
  - 字段名: `response_schema` (string)
  - 说明: 可选的 LLM Response JSON Schema。可为空字符串 `""`。
- **任务明细 (Task Detail)**:
  - 字段名: `response_schema` (string)
  - 说明: 可选的 LLM Response JSON Schema。可为空字符串 `""`。

### 2. 继承与指定规则
- **周期性任务**:
  - 由任务元数据周期性拆分/创建的任务明细，将**自动继承**其所属任务元数据的 `response_schema`。
- **一次性任务**:
  - 一次性任务明细在创建时，可以**可选地指定**其自身的 `response_schema`。

## Swarm 规范与继承机制

在 `20260524-1` 迭代中，系统为任务元数据和任务明细全面引入了 `swarm` 字段，用于标记任务是否启用 swarm 模式。

### 1. 字段定义
- **任务元数据 (Task Metadata)**:
  - 字段名: `swarm` (boolean)
  - 说明: 不可为空，默认值为 `false`
- **任务明细 (Task Detail)**:
  - 字段名: `swarm` (boolean)
  - 说明: 不可为空，默认值为 `false`

### 2. 继承与指定规则
- **周期性任务**:
  - 由任务元数据周期性拆分/创建的任务明细，将**自动继承**其所属任务元数据的 `swarm`
- **一次性任务**:
  - 一次性任务在创建时，如果显式传入 `--swarm` 或通过子模块调用指定 `swarm=true`，首条明细直接继承该值
- **默认值**:
  - 元数据和明细在未传时统一保存为 `false`
# 2026-05-24 更新

- 对外字段与库表字段统一使用 `router_disable`。
- `router_disable=true` 表示关闭路由/SWARM，`router_disable=false` 表示开启。
- CLI 仍保留 `--swarm` 作为开关别名，但语义与 `router_disable` 相反：
  - `--swarm true` 等价于 `router_disable=false`
  - `--swarm false` 等价于 `router_disable=true`
- 未显式传值时，默认按 `router_disable=true` 处理。
- 旧数据里的 `swarm` 仅作为兼容输入/迁移来源，新的请求、响应、持久化和示例都应以 `router_disable` 为准。
