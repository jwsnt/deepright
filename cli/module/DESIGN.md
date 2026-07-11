### 设计目标
+ 将当前CLI系统的多个模块实现收敛为共享能力内核 + 明确边界适配层的分层结构

### 核心设计目标
+ 建立单一能力内核，同一种能力只能有一个authoritative implementation，避免多模块平行实现导致行为漂移。
+ 明确层次边界，区分“能力内核”“入口适配层”“插件实现层”“对外协议层”，禁止跨层直接穿透实现。
+ 固化运行时标识语义，系统 `Key` 作为唯一运行时主标识，展示名 `Name` 退出运行链路，仅保留在展示层。
+ 降低变更传播半径，任一模块内部变更应尽量停留在模块内或共享内核层，不应要求多个主程序、多个插件同时联动修改。

### 典型错误设计
+ 横向复制，多个模块各自维护同类能力实现，造成没有唯一真相，功能升级需要多处同步修改，极易出现行为漂移
    + 典型错误（knowledge、skills、agent属于同类型模块）：
        + `agent` 元数据扫描逻辑在 `agent`、`cli-get`、`integration`、`proxy` 中重复存在
        + `static` 文件服务在 `static` 和 `proxy` 中各维护一份
        + 插件生命周期调度逻辑在 `integration` 与 `proxy` 中分别实现

+ 纵向穿透，插件实现细节上升到主程序层，会造成变更会反向影响 `connect` 编译，造成插件独立边界单元的设计被破坏
    + 典型错误（插件分插件容器和插件实现）：
        + `connect` 顶层入口直接编译依赖具体插件实现，例如：
        + `connect/main.go` 直接依赖 `emailsvc`
        + `connect/main.go` 直接依赖 `feishusvc`

+ 语义混用，`Key`与`Name` 在链路中混杂使用：存储，查询，回调，插件通知，HTTP配置输出
    + 问题在于
        + 展示字段进入运行链路
        + 系统无法区分 主标识 和 展示别名
        + 多语言展示名会污染运行态行为

### 设计原则
##### 单一实现原则
+ 每项基础能力只保留一份实现，其他模块只能复用，不能复制。
    + 参考案例（knowledge、skills同理）
        + `agentcore` 成为Agent元数据唯一内核
        + `static/server` 成为静态服务唯一内核

##### 入口薄包装原则
+ `main.go` 只负责参数解析、启动和结果输出，不承担核心业务能力。
    + 意味着（knowledge、skills同理）：
        + `agent/main.go` 从“实现 + 入口”转为“内核包装 + CLI”
        + `static/main.go` 从“实现 + 入口”转为“server.Register 的启动壳”

##### 运行时解耦原则
+ 主程序与插件之间只通过文件系统可执行体和稳定 CLI 契约交互，而不是编译期依赖插件包。
    + 参考案例
        + `connect/main.go` 不再 import `emailsvc` / `feishusvc`
        + 改为运行时解析插件并转发命令

##### 标识分层原则
+ `Key`：系统主键，参与持久化、查找、回调、运行态通知, `Name`：展示名称，仅用于 UI、日志、帮助文档
    + 技术上要求
        + 任何运行路径都不得依赖`Name`作为主查找键
        + `Name` 可兼容输入，但不能成为运行主标识

##### 兼容收敛原则
+ 兼容逻辑只能存在于边界适配层，不能扩散进多个内核实现。
    + 参考案例
        + `connectsvc` 的查询与解析适配
        + `integration` / `connect` 的接入层修正
        + 测试层对新契约的适配

### 目标架构
+ 可抽象为四层：

##### 能力内核层
+ 负责提供唯一实现：
    + `agentcore`：让 `integration` / `proxy` 包装共享为 `agentcore`（knowledge、skills同理）
    + `static/server`
    + `connectsvc`

##### 入口适配层
+ 负责将能力内核暴露为 CLI 或 HTTP：
    + `agent/main.go`
    + `static/main.go`
    + `connect/main.go`
    + `knowledge/main.go`
    + `integration/main.go`
    + `proxy/main.go`

##### 插件实现层
+ 负责具体平台能力：
    + `email`
    + `feishu`
    + `browser`
    + `remote`

##### 沙箱执行层
+ 负责通过沙箱控制系统命令权限，分MAC OS和Windows WSL
+ 沙箱实现与主应用隔离，通过不同命令行方式进行路由区分
+ MAC沙盒技术规范：cli-get/sandbox/mac/DESIGN.md

##### 外部交互层
+ 对外协议边界：
    + CLI
    + HTTP API
    + 插件二进制协议

### 历史成功案例
##### Agent 元数据能力收敛
+ 将Agent 扫描、技能解析、系统信息探测、插件探测、缓存策略统一收进一个共享内核，解决 schema 漂移和实现复制。
+ 设计动作，新增共享内核：
    - [core.go](/Users/shenjiawei/DEV/code_gen/cli/module/agent/agentcore/core.go)
+ 下沉职责：
    - Skill扫描
    - Agent目录扫描
    - 系统字段探测
    - 插件探测
    - JSON缓存与结构化缓存
    - `GetAgentIDs`
    - `GetAgentByID`
    - `GetSkillNames`
+ 入口调整：
    - `agent/main.go` 退化为CLI包装层（knowledge、skills同理）
    - `cli-get` 改为复用 `agentcore.Output`
+ 技术收益
    - Agent schema只保留一份
    - 新字段变更只需改一处
    - 下游模块获得统一缓存行为与统一字段定义
    - `cli-get` 中缺失`git`字段的漂移问题被消除

##### Static能力收敛
+ 将静态文件服务改造成真正可复用的库能力，而不是只能独立运行的程序。
+ 设计动作，新增共享包：
    - [server.go](/Users/shenjiawei/DEV/code_gen/cli/module/static/server/server.go)
+ 改造方式：
    - `static/main.go`只负责启动
    - `proxy`改为依赖 `static-server/server`
    - 删除`proxy/static`副本实现
+ 技术收益
    - `static`变成可复用能力而不是孤立二进制
    - 消除一类重复代码
    - 保证静态服务行为一致

##### Connect与插件解耦
+ 恢复插件机制设计边界，使`connect`只依赖插件契约，而不是插件源码实现。
+ 设计动作
    - [connect/main.go](/Users/shenjiawei/DEV/code_gen/cli/module/connect/main.go)
+ 实现变化：
    - 移除对`emailsvc` / `feishusvc`的直接import
    - 引入运行时插件解析与转发
    - 新增统一插件CLI转发入口
    - 保留`connect feishu ...` / `connect email ...`等顶层兼容形式
+ 技术收益
    - 插件独立演化
    - `connect`与插件编译边界清晰
    - 更符合`PLUGIN.md`所定义的插件模型

##### 插件生命周期语义统一
+ 插件第一性原则：插件容器与插件交互只允许通过CLI命令：start、stop、init、send、name、param、command等（不允许直接调用插件代码）
+ 固化任务开始与任务结束通知语义，避免不同入口各自定义生命周期含义。
+ 将`integration`与`proxy`的插件通知、回调执行、命令支持探测，抽成共享runtime。
+ 设计动作，统一协议：
    - `init`：任务开始前通知
    - `send`：任务结束后通知
+ 技术规范
    + 插件运行状态必须以统一状态接口为唯一事实源，禁止各入口自行判断进程、PID 文件或运行态。
    + 插件包装层必须完整实现插件协议命令，包括 name、param、scope、command、start、stop、status；禁止未识别命令下钻到运行时daemon
    + start、stop、status、UI展示、批量停插件等生命周期操作，必须复用同一套状态判定与动作执行链路，禁止多套实现并存
    + 插件daemon只负责运行时能力，不承担协议适配职责；协议收口必须在插件包装层完成
    + 插件启动前必须先做残留实例探测与状态校正；存在历史进程时，应返回真实运行态，而不是依赖本地缓存或前端默认值
    + 生命周期相关接口必须可测试覆盖以下场景：未启动、已启动、残留 PID、状态漂移、协议命令误透传
+ 修正行为：
    - `integration`中任务开始通知从`send`改回`init`
+ 技术收益
    - 生命周期与插件执行逻辑彻底消除双实现
    - 生命周期语义变为协议级约束
    - 主入口之间不再出现行为漂移
    - 新插件只需遵循同一通知模型
    - 降低入口实现之间的行为偏差
+ 插件准入
    - 插件测试必过集在插件目录下的TEST_CASE.md

##### Key / Name语义治理
+ 从模型层切断展示语义对运行链路的污染。
+ 设计动作,明确分工：
    - `Key`：运行主标识
    - `Name`：展示标识
+ 具体治理：
    - `GetMetaConfig`支持兼容性展示名查询
    - 运行链路继续使用`Key`
    - `AddRequest`等操作通过 `config.Key`反查元数据
    - 保留旧调用兼容，但不让展示名成为运行主键
+ 技术收益
    - 消除“显示字段反向驱动业务逻辑”的设计缺陷
    - 回调、通知、数据库记录的语义更稳定
    - 多语言展示名不再影响运行态行为

##### 帮助命令
+ 为integration和插件的每个命令提供完整的帮助（--help）和使用案例

##### 统一浮层架构
- 建立统一浮层管理体系，统一管理页面内所有固定定位遮罩、引导层、确认框、弹窗、日志面板、透明点击层，彻底解决层级冲突、跨区域误拦截、隐藏视图仍可点击、遮罩显示但弹窗不可点、残留浮层拦截当前弹窗点击、业务代码各自控制显隐和层级的问题。
- 所有固定定位浮层必须接入统一浮层管理器，禁止业务代码直接控制 `show`、`z-index`、`pointer-events`。
- 新增浮层必须基于统一架构开发，旧浮层逐步迁移到统一管理体系。
- 所有浮层必须具备统一元数据：`id`、`domain`、`type`、`order`、`visible`、`payload`。
- 必须提供统一公共 API：`openOverlay`、`closeOverlay`、`toggleOverlay`、`getTopOverlay(domain)`。
- 所有浮层必须按三大交互域隔离：`left`、`center`、`right`。
- 左中右三域浮层互不串扰，不允许跨域竞争层级和点击控制。
- 同一交互域内任一时刻只允许一个前台浮层可交互。
- 同域内打开新浮层前必须先关闭当前可交互浮层，禁止多个浮层重叠争抢点击和层级。
- 同域内即使存在多个显示中的浮层，只有最前面的浮层可点击，其余浮层必须彻底失活。
- 隐藏浮层必须同步失活，并同时满足 `visibility:hidden`、`pointer-events:none`、`aria-hidden=true`。
- 隐藏浮层不得参与层级竞争，不得拦截点击，不得接收焦点，不得参与键盘导航。
- 隐藏浮层及其所有子元素必须完全不可点击、不可聚焦、不可交互。
- 非前台浮层即使显示，也必须按失活态处理，不能拦截当前前台浮层操作。
- 所有浮层必须通过统一 portal 或统一挂载机制渲染，禁止直接挂在受 `overflow:hidden` 或定位上下文影响的业务容器中。
- 浮层挂载位置必须与目标压制层保持一致，目标在弹窗、抽屉、设置面板等容器中时，浮层必须直接渲染到对应容器内部。
- 禁止依赖外层 `z-index` 强压解决浮层层级问题。
- 浮层定位、遮罩高亮、说明卡片位置必须统一收口到公共函数或 portal 机制。
- 禁止业务代码自行手算 `left`、`top`、`right`、`bottom`。
- 设计浮层结构时必须先确定挂载层级与坐标系，避免放入会导致裁切、错位和点击异常的容器。
- 浮层层级必须按“域内 `order` + 最近打开顺序”统一排序。
- 层级排序规则仅在各自交互域内生效，不允许跨域混排。
- backdrop 点击、`ESC`、取消、确认都必须走统一关闭链路，禁止业务各自分散处理。
- 共享确认框不能只靠单一状态字段区分业务，必须基于真实业务 `payload` 执行提交、校验和幂等控制。
- 确认框的打开、取消、确认、关闭都必须纳入统一浮层状态流转。
- 引导、确认、弹层等步骤切换不能只依赖理想时序，面对用户点击、浮层关闭、侧栏展开、动画结束、异步渲染等顺序变化时流程仍必须可恢复。
- 所有引导步骤都应能根据当前 DOM 状态重新同步并具备兜底恢复能力，不能只依赖内存状态机。
- 必须避免因中途时序变化导致步骤丢失、跳错或无法继续的问题。
- 选择器必须唯一且稳定，优先使用带作用域的 ID 或明确容器前缀。
- 禁止依赖 `.settings-panel` 这类可能重复命中的通用类名作为关键浮层选择器。
- 布局逻辑与 CSS 限制必须一致，运行时动态计算的 `width`、`height`、`maxWidth` 不得再被 CSS 写死更小上限覆盖。
- 所有前端依赖资源必须随站点页面一起可访问。
- 发布链路中必须强校验静态资源实际存在且 MIME 类型正确。
- 禁止出现页面已引用但静态服务未实际发布对应本地资源的情况。
- 若存在 `src/site` 与 `release/site` 双份静态资源，修改前必须先确认服务实际读取的运行目录。
- 修改静态资源时必须同步运行目录与源码目录，不能只改源码文件。
- 浮层问题排查顺序必须固定为：先确认运行文件，再确认挂载层级，再检查层级与 `pointer-events`，再检查 CSS 尺寸限制，最后确认选择器唯一性与状态恢复能力。
- 实施时优先复用成熟、轻量、可控的开源方案，在满足需求前提下保持代码简洁、依赖最少、包体积尽可能小。

##### 统一端口
+ integration/proxy所有服务对外使用统一且唯一的端口，有--port指定或使用默认值

##### 设计落地
1. Agent 能力内核化
2. Static 能力内核化
3. Knowledge 能力内核化
4. Connect 顶层对插件去编译期耦合
5. Integration 生命周期协议修正
6. Key / Name 运行语义重整

##### 模块设计
+ agent/DESIGN.md
+ cli-get/DESIGN.md
+ connect/DESIGN.md
+ cron/DESIGN.md
+ integration/DESIGN.md
+ knowledge/DESIGN.md
+ proxy/DESIGN.md
+ site/DESIGN.md
+ skills/DESIGN.md
+ static/DESIGN.md

##### 数据库设计
- integration 与 proxy 统一复用应用运行目录下的共享 sqlite；integration 通过 `resolveIntegrationDBPath()` 打开 `data`，proxy 通过 `getDataDBPath()` 打开同一份库，避免日志、会话、定时任务状态分裂为多份本地库。
- 日志时间统一写入文本格式 `2006-01-02T15:04:05.000`，所有按时间范围读取与过期删除都依赖该格式的字典序与时间序一致。
- `chat_log` 用于保存会话消息主链路，字段为 `agent_id / chat_id / chat_type / role / response_type / content / created_at`；其中 `chat_type` 当前覆盖 `page_session` 与 `scheduled_task`，用于区分普通页面会话与定时任务会话。
- `chat_log` 当前索引为 `idx_chat_agent_chat(agent_id, chat_id)` 与 `idx_chat_agent_chat_time(agent_id, chat_id, created_at)`，对应代码里按会话读取、按会话增量拉取、按时间续传的主查询路径。
- `agent_message_log` 用于保存 Agent 侧结构化消息日志，字段为 `agent_id / chat_id / content / log_type / created_at`；写入入口来自 `cli-get`、integration 以及 proxy 的 eventlog，读取入口以按 `chat_id`、`agent_id + chat_id`、`log_type`、时间范围查询为主。
- `agent_message_log` 当前索引为 `idx_agent_message_log_chat_type_time(chat_id, log_type, created_at)`、`idx_agent_message_log_agent_chat_type_time(agent_id, chat_id, log_type, created_at)` 与 `idx_agent_message_log_agent_chat_time(agent_id, chat_id, created_at)`，与现有查询条件保持一致，不额外引入只覆盖低频写路径的冗余索引。
- `connect/logretention` 是日志保留策略的共享实现；integration 与 proxy 在启动完成数据库初始化后都会使用独立 sqlite 连接异步执行一次清理，不阻塞首屏服务与主查询连接，并在同一事务内物理删除 `agent_message_log` 与 `chat_log` 中 `created_at < cutoff` 且超过最近 30 天的数据。
- 启动阶段的过期日志清理必须是非阻塞设计：禁止在 HTTP 服务监听、首屏接口响应或主共享 sqlite 连接上同步执行大批量删除，避免首次打开出现空白页或长时间无响应。
- 日志清理状态通过 `/api/log_cleanup_status` 暴露给站点页面；返回 `checked / running / message / cutoff / deletedAgentMessageLog / deletedChatLog / error` 等字段，便于前端轮询并在启动阶段给出确定反馈。
- 站点在检测到清理运行中时，使用统一中心域浮层锁定界面并提示“正在清理过期日志，请稍后”；清理结束后自动解除锁定，不要求人工确认。
