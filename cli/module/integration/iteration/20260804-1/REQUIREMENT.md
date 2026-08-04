### 第一性原则
+ 仅可以新增、更新或删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../site/index.html`、`../../../config/config.json` 与应用发布资源。

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
+ 不新增外部 Go 依赖，不改变既有 `/api/runtime_config`、普通消息发送、附件上传或 Agent 工作区访问协议。

### 需求介绍
+ 降低 Integration 待机时向上游 `POST /cli/get` 的访问频率，同时不得降低任一未终态 SSE 期间的任务拉取吞吐。
+ SSE 活跃范围是整个 Integration 进程，而非单个 Agent 或单个会话：
    + 普通会话的 `/v1/chat/completions` SSE。
    + 备忘录、飞书消息、邮件消息等定时/连接任务发起的 SSE。
    + Integration 内其他实际向上游建立 SSE 的链路也应纳入，避免同一进程出现未计数的活跃流。
+ “未终态 SSE”是指 SSE 请求已经发起，且尚未完成、异常结束、取消或超时的请求；请求已发出但尚未收到上游响应头的连接阶段同样属于未终态。

+ `config/config.json` 新增并使用如下配置：
    ```json
    {
      "get": {
        "sleep": 15000,
        "await": 30000,
        "check": 10
      }
    }
    ```
    + 单位均为毫秒。
    + `get.sleep` 是 cli-get 心跳失败、Agent 扫描失败或本地任务队列满时的等待基数。
    + `get.await` 是 Integration 没有任何未终态 SSE 时，成功 `/cli/get` 响应后的待机间隔。
    + `get.check` 是在收到过需执行 `cmd` 后、重新进入待机前允许连续快速检查的“无 cmd 成功响应”次数。
    + `get.sleep` 与 `get.await` 必须是非负整数；`get.check` 必须是正整数。配置缺失时保持现有兼容默认值，其中 `await` 默认 `30000`、`check` 默认 `10`。
    + 命令行显式传入 `--sleep` 时必须优先于 `config.json.get.sleep`；未传入时使用 `config.json.get.sleep`。
    + 不新增 `--await` 命令行参数，`await` 仅由 `config.json.get.await` 配置。

+ 新增 Integration 进程级、跨平台的 SSE 原子活跃计数：
    + 每个 SSE 在向上游发起请求前原子加一。
    + 每个 SSE 在完成、异常、取消、超时或请求建立失败的统一清理路径原子减一。
    + 单个 SSE 的清理必须幂等，计数不得因为重复清理变为负数。
    + 计数仅保存在进程内存，不写入 SQLite，不新增 HTTP 接口、数据库表、上游字段或报文 schema。
    + 现有 macOS 防休眠计数不等同于本需求计数：本需求计数必须在所有系统生效，并覆盖响应头到达前的连接阶段。

+ Integration 内置 cli-get 的成功心跳调度规则：
    + 若 SSE 原子计数大于零：
        + `/cli/get` 返回任务时，任务必须先进入现有本地 `taskQueue`；成功入队后立即发起下一次 `/cli/get`。
        + `/cli/get` 返回无任务时，立即发起下一次 `/cli/get`。
        + 不等待任务执行、`/cli/pub`、`get.await` 或 SSE 结束。
    + 若 SSE 原子计数等于零：
        + 若本次响应包含非空 `cmd`，必须原子重置连续无 cmd 计数，并立即进行下一次 `/cli/get`；不等待任务执行或 `/cli/pub`。
        + 若进程启动以来尚未收到过需执行 `cmd`，本次成功响应无 `cmd` 后直接等待 `get.await`。
        + 若此前已收到过需执行 `cmd`，每次成功响应无 `cmd` 都必须原子递增连续无 cmd 计数；计数小于 `get.check` 时立即进行下一次 `/cli/get`，计数达到 `get.check` 时才等待 `get.await`。
        + 快速检查期间任一次响应再次包含非空 `cmd` 时，必须原子重置连续无 cmd 计数，并重新开始快速检查周期。
    + SSE 原子计数与连续无 cmd 计数必须独立维护、共同决策：
        + SSE 原子计数大于零时始终立即请求，连续无 cmd 计数不得使其等待。
        + SSE 原子计数等于零时，才允许连续无 cmd 计数决定立即请求或 `await`。
    + 当 cli-get 正在等待 `get.await` 时，只要 SSE 计数从零变为正数：
        + 必须立即中断本次等待。
        + 必须立即进入下一次 `/cli/get` 上报，不得等待当前 `await` 计时结束。
    + 现有例外语义保持不变：
        + `/cli/get` 网络错误、超时、HTTP 非 200 或响应解析失败时，按 `--sleep` / `get.sleep` 进行既有指数退避，最大退避上限保持不变。
        + 本地 `taskQueue` 已满时，不发起 `/cli/get`，按 `--sleep` / `get.sleep` 等待后重新检查。
        + `/cli/pub` 的独立队列与重试不影响心跳等待决策。

+ 验收要求：
    + 覆盖 `get.sleep`、`get.await` 的配置读取、非负校验以及 `--sleep` 优先级。
    + 覆盖 SSE 计数在开始、正常完成、异常、取消/超时与重复清理时的增减和不为负语义。
    + 覆盖无 SSE 时成功心跳等待 `get.await`。
    + 覆盖等待 `get.await` 期间创建 SSE 会立即唤醒心跳。
    + 覆盖 SSE 活跃时，有任务和无任务两种成功响应都不会等待 `get.await`。
    + 覆盖启动后的首个无 cmd 响应直接进入 `await`、收到 cmd 后的连续空响应快速检查、达到 `get.check` 后进入 `await`，以及新 cmd 原子重置计数。
    + 覆盖心跳失败、队列满和 `/cli/pub` 重试仍使用既有策略。

### 编写代码
+ 以 Golang 完成最小范围更新，复用既有 cli-get 队列、心跳退避、上下文取消与 SSE 生命周期处理；不得新增外部 Go 依赖。
+ 不得改变 `/cli/get`、`/cli/pub`、`/v1/chat/completions`、`/api/heartbeat` 或其他既有 HTTP 接口的请求/响应 schema。
+ 不得改变任务队列、执行 worker、发布 worker、`ddl` 校验、沙盒、活跃命令、取消、日志和 `/cli/pub` 重试的既有语义。

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`，明确配置优先级、SSE 活跃/待机两种心跳节奏和立即唤醒语义。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求、边界和验收行为，不记录实现过程。
