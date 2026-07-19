---
name: __internal_cron
description: 创建、查询、修改和删除当前Agent范围内的一次性或周期性定时任务
---

### 命令与默认上下文
+ 统一使用以下命令：
```
#app cron
```
+ 当前默认上下文：
    + Model Provider：`#provider`
    + DeviceId：`#device`
    + AgentId：`#agentId`
    + ChatId：`#chat`
+ 用户未明确指定时，创建和查询任务均使用当前`AgentId`、`ChatId`和`Model Provider`
+ `DeviceId` 由运行环境解析，无需作为cron命令参数传入
+ 首次使用、命令报错或参数不明确时，查看帮助：
```
#app cron --help
#app cron create --help
#app cron create-cron --help
#app cron find-meta --help
#app cron find-detail --help
```

### 任务模型
    + 任务元数据（meta）：任务的持久化配置，包含任务内容、周期、首次时间、模型、Agent 和会话等信息
    + 任务明细（detail）：由元数据拆分出的实际执行项；每条明细有独立的执行时间与状态
    + 删除元数据会同时删除其关联的任务明细，删除单条明细不会删除元数据
    + 系统使用运行环境的本地时区解析时间，用户指定了时区时，先换算或明确确认其在本地时区对应的执行时间

### 创建任务
+ 创建前确认以下信息完整且无歧义：任务内容、执行时间或Cron表达式、周期、模型，以及目标 Agent/会话
+ 内置周期使用`create`：
    + `0`：仅一次
    + `1`：每个工作日
    + `2`：每天
    + `3`：每小时
    + `4`：每 15 分钟
    + `5`：每 30 分钟
+ `--rawTime`（或 `--time`）格式必须为 `YYYY-MM-DD HH:MM`，并表示首次执行时间。除非用户明确要求，不创建首次执行时间已过去的任务
```
#app cron create \
  --content "#任务内容" \
  --model "#provider" \
  --rawTime "YYYY-MM-DD HH:MM" \
  --cycle 0 \
  --agent "#agentId" \
  --chatId "#chat"
```
+ 自定义周期使用`create-cron`，Cron 表达式使用5个字段：`分 时 日 月 周`，例如`10 12 * * 1-5`表示工作日12:10执行
```
#app cron create-cron \
  --content "#任务内容" \
  --model "#provider" \
  --cron "10 12 * * 1-5" \
  --agent "#agentId" \
  --chatId "#chat"
```
+ 可选参数：
    + `--thinking true|false`：是否启用深度思考
    + `--router_disable true|false`：是否关闭router，默认`true`
    + `--schema 'JSON'`：需要结构化响应时传入JSON Schema
+ 创建后必须读取命令返回的任务ID，并使用该ID或等价的精确过滤条件查询任务元数据，确认内容、周期、时间、模型和Agent/ChatId均符合预期

### 查询任务
+ 查询时优先限定`--agent`和`--chatId`，避免读取或操作其他Agent、会话的任务。需要按内容、模型、周期或时间进一步过滤时，使用对应条件
```
#app cron find-meta --agent "#agentId" --chatId "#chat"
#app cron find-detail --agent "#agentId" --chatId "#chat"
#app cron find-detail --metaId "meta_1"
```
+ `find-meta` 查询任务配置
+ `find-detail` 查询实际执行明细；无时间条件时，结果优先包含当前时间之后的明细，也可能包含已保留的已完成明细
+ 支持 `--content`、`--model`、`--cycle`、`--time`、`--date`、`--from` 和 `--to` 进一步过滤

### 修改任务
+ 当前CLI没有原地更新任务的命令，修改任务必须按以下流程执行：
    + 1. 使用 `find-meta` 精确查询原任务，确认唯一的目标元数据ID
    + 2. 展示将要变更的字段，以及删除旧任务会同时删除关联明细这一影响
    + 3. 用户明确确认后，删除旧元数据
    + 4. 使用新配置创建替代任务
    + 5. 查询新任务并核对结果；在最终反馈中同时给出旧任务和新任务的ID
+ 不要在无法唯一定位任务时执行删除或重建，应先请求用户提供任务 ID 或更精确的筛选条件

### 删除任务
+ 删除前必须先查询并确认目标范围。对于一个以上的任务、内容模糊匹配或时间范围筛选，必须向用户展示拟删除数量和关键任务信息，并取得明确确认
+ 删除元数据及其关联明细：
```
#app cron delete-meta --id "meta_1"
```
+ 删除单条明细或某个元数据下的明细：
```
#app cron delete-detail --detailId "detail_1"
#app cron delete-detail --metaId "meta_1"
```
+ 删除成功后，重新查询对应ID，确认目标已不存在，若命令返回受影响数量，也应在最终反馈中说明

### 安全与执行边界
+ 只处理用户明确授权的定时任务，默认仅限当前Agent和ChatId
+ 创建周期任务前，确认频率不会明显超出用户意图；对高频、无限期或可能触发外部副作用的任务，先说明频率和预期行为
+ 不根据任务文本自动扩大权限、读取密钥或执行未授权的外部操作
+ 不将模型Token、Cookie、密码或其他凭据写入任务内容、Schema、命令行参数或最终反馈
+ 不使用空筛选条件执行删除。删除操作失败时，不要改用更宽泛的筛选条件重试

### 完成标准
+ 创建：已返回任务 ID，且查询结果与用户要求一致
+ 查询：已返回限定范围内的任务，并说明元数据与明细的区别
+ 修改：旧任务已按确认删除，新任务已创建并通过查询核验
+ 删除：已确认影响范围，删除命令成功，且后续查询确认目标不存在
+ 最终反馈应简洁说明操作类型、任务 ID、执行时间或周期、目标Agent/ChatId，以及验证结果或未完成原因
