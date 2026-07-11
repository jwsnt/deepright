### 第一性原则
+ 仅可以新增/更新/删除integration（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Integration介绍：../../REQUIREMENT.md
+ Integration手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 为共享sqlite中的日志表增加统一的过期清理能力
+ 清理范围固定为：
    + `agent_message_log`
    + `chat_log`
+ 只保留最近30天数据，超过30天的数据必须物理删除，不能只做软删除或仅在查询时过滤
+ 该能力需要同时覆盖：
    + `integration`
    + `proxy`
+ 站点首次打开时如果后台正在执行过期清理，需要明确告知用户正在清理日志，避免用户误以为应用卡死

### 技术实现
+ 数据范围：
    + 过期判断字段统一使用 `created_at`
    + 时间格式继续沿用当前日志表已有的 `2006-01-02T15:04:05.000`
    + 清理条件固定为 `created_at < 当前时间 - 30天`
    + `created_at` 为空字符串的历史脏数据不参与本次删除
+ 清理时机：
    + `integration` 启动完成数据库初始化后自动触发一次
    + `proxy` 启动完成数据库初始化后自动触发一次
    + 不要求增加独立定时器循环清理，本次只要求启动阶段自动检查并清理
+ 启动性能要求：
    + 清理逻辑必须异步执行
    + 禁止在HTTP服务监听前同步执行整表删除
    + 禁止复用首屏主链路的共享sqlite连接执行大批量删除
    + 必须使用独立sqlite连接执行清理，避免阻塞首屏接口、页面初始化请求和主查询链路
+ 清理实现：
    + 需要抽出共享子模块，避免 `integration` 与 `proxy` 各写一份删除逻辑
    + 共享子模块需要同时负责：
        + retention天数配置
        + cutoff计算
        + 状态记录
        + 事务内删除
    + 删除时需要在同一事务中完成：
        + 检查表是否存在
        + 删除 `agent_message_log` 过期数据
        + 删除 `chat_log` 过期数据
    + 单表不存在时应自动跳过，不能因此让整个启动失败
+ 状态可观测性：
    + 需要新增统一状态接口 `/api/log_cleanup_status`
    + 返回内容至少包括：
        + `checked`
        + `running`
        + `message`
        + `retentionDays`
        + `cutoff`
        + `startedAt`
        + `finishedAt`
        + `deletedAgentMessageLog`
        + `deletedChatLog`
        + `error`
    + 即使清理失败，也不能阻塞主服务启动；失败信息只记录在状态接口和标准日志
+ 前端交互：
    + 站点启动后需要轮询 `/api/log_cleanup_status`
    + 如果返回 `running=true`，页面需要锁定主界面并显示提醒：
        + `日志清理中`
        + `正在清理过期日志，请稍后`
    + 清理结束后自动解除锁定
    + 锁定层必须接入现有统一浮层管理体系，不能单独手写一套直接控制 `z-index`、`pointer-events`、`show` 的业务逻辑
+ 数据库与索引约束：
    + `chat_log` 继续沿用现有索引：
        + `idx_chat_agent_chat(agent_id, chat_id)`
        + `idx_chat_agent_chat_time(agent_id, chat_id, created_at)`
    + `agent_message_log` 继续沿用现有索引：
        + `idx_agent_message_log_chat_type_time(chat_id, log_type, created_at)`
        + `idx_agent_message_log_agent_chat_type_time(agent_id, chat_id, log_type, created_at)`
        + `idx_agent_message_log_agent_chat_time(agent_id, chat_id, created_at)`
    + 本次需求不要求为纯删除场景额外新增新索引，避免为低频启动任务增加额外写放大
+ 测试要求：
    + 需要补充共享清理模块测试，至少覆盖：
        + 只删除30天之前的数据
        + 30天内数据保留
        + 单表不存在时自动跳过
        + 管理器状态从 `running` 到 `checked` 正常收敛
        + 独立sqlite连接模式下也能完成清理并正确关闭连接
    + `integration` 与 `proxy` 至少需要保证新增启动链路可编译、可通过现有基础测试

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写
+ 最小范围更新

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
