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
+ 技术规范的指导下收口：
    + 当当前会话开启了沙盒模式，且该会话已存在非空沙盒目录时：
        + 在转发 `/v1/chat/completions` 时，需要在最外层 `metadata` 中补充 `sandbox_path`
        + 在上报 `cli/get` 时，也需要在最外层 `metadata` 中补充 `sandbox_path`
    + `sandbox_path` 需要与 `knowledge` 平级，不能挂在 `metadata.agent`、`metadata.agents[]`、`metadata.agentId` 或其他 Agent 维度字段下
    + `sandbox_path` 必须按当前会话 `chatId` 维度解析，不与 `agent` 绑定
    + `/v1/chat/completions` 的覆盖范围包括：
        + 居中对话框直接转发
        + 备忘录任务最终转发到上游的 `/v1/chat/completions`
        + 邮件任务最终转发到上游的 `/v1/chat/completions`
        + 飞书任务最终转发到上游的 `/v1/chat/completions`
    + 只有在“当前会话已开启沙盒”且“当前会话存在非空沙盒目录”时，才允许携带 `metadata.sandbox_path`
    + 若当前会话沙盒模式为 `net`、`off`、未开启，或当前会话没有已选目录，则不得携带空字符串，不得回退为 `""`，而是直接不传 `sandbox_path`
    + `sandbox_path` 的值需要直接取当前会话沙盒状态中的目录白名单路径，即当前会话 `allowedDir`
    + 外部请求体若自行传入 `metadata.sandbox_path`，最终也必须由 integration 按当前 `chatId` 重新计算并覆盖，不能信任外部入参
    + 本次迭代只新增顶层 `metadata.sandbox_path`，不要求删除现有 `metadata.agents[].sandbox`、`metadata.agent.sandbox` 或其他既有沙盒字段，避免破坏已有兼容性
    + 目标上游请求示例：
    ```
    {
        "metadata": {
            "knowledge": {
                "path": "/tmp/knowledge"
            },
            "sandbox_path": "/Users/demo/Desktop"
        }
    }
    ```

### 技术实现
+ 注入层级：
    + 统一由 `integration` 收口层负责注入顶层 `metadata.sandbox_path`
    + 不要求改动调用方页面、备忘录页面、邮件插件、飞书插件各自的前端请求拼装逻辑
    + 不要求为了本次需求去修改 `agent-scanner` 的结构定义，把 `sandbox_path` 放进 `metadata.agents[]`
+ 会话维度：
    + 当前会话沙盒状态已经按 `chatId` 存储和恢复，本次需求继续沿用这一原则
    + `agentId` 在本需求中只保留现有链路的兼容语义，不作为 `sandbox_path` 的绑定键
    + 同一 `chatId` 下即使切换不同 `agentId`，只要会话沙盒目录未变，最终上报的 `metadata.sandbox_path` 也必须保持一致
+ 数据来源：
    + 顶层 `metadata.sandbox_path` 的值需要读取当前会话沙盒状态表中的 `allowed_dir`
    + 实现时只认共享沙盒状态中的真实持久化值，不能从前端 query、请求 body、自定义 header 或其他临时字段拼接
    + 读取后需要做 `trim`
    + `trim` 后为空，则视为不存在，不上报
+ 注入时机：
    + 外部 `POST /v1/chat/completions`：
        + 在共享 Agent metadata 与请求体 metadata merge 完成后
        + 必须基于 merge 后最终生效的 `chatId` 再注入 `sandbox_path`
        + 注入完成后再继续走后续的 metadata prune、请求规范化、转发上游逻辑
    + `cli/get`：
        + 在构建心跳上报 metadata 时注入
        + 需要沿用现有 `cli/get` 的当前会话解析逻辑拿到本轮上报对应的 `chatId`
        + 同一轮 `cli/get` 的 `lastResponse`、`knowledge`、`plugins` 等已有 metadata 注入逻辑不能被破坏
    + integration 内部 cron/chat 转发：
        + 在 `memo`、`email`、`feishu` 等任务真正转发到上游 `/v1/chat/completions` 前注入
        + 因为这些链路不是前端直发，而是 integration 内部拼装 metadata 后再发起上游请求，所以不能只修外部 HTTP 入口
+ 字段行为：
    + 当存在有效目录时：
        + `metadata.sandbox_path = 当前会话沙盒目录`
    + 当不存在有效目录时：
        + 如果 `metadata` 上已有旧的 `sandbox_path`，必须显式删除，避免脏字段残留
        + 不允许把字段保留为 `null`
        + 不允许把字段保留为 `""`
+ 与现有字段的关系：
    + `metadata.sandbox_path` 是当前会话级字段，与 `knowledge` 同层
    + 现有 `metadata.agents[].sandbox`、`metadata.agent.sandbox` 若已存在，继续保留，不作为本次需求的迁移目标
    + 本次新增字段只表达“当前会话可访问的沙盒目录路径”，不替代已有“沙盒模式”语义
    + 也就是说：
        + `sandbox` 仍表示模式，如 `filepick` / `filepick_net` / `net`
        + `sandbox_path` 只表示目录白名单路径
+ 边界约束：
    + `net` 模式没有目录白名单，因此即使沙盒开启，也不得上报 `sandbox_path`
    + `off` 模式不得上报 `sandbox_path`
    + `filepick` / `filepick_net` 只有在目录白名单最终存在且非空时才允许上报
    + 若会话从 `filepick` / `filepick_net` 切到 `net` / `off`，后续转发与上报必须立即不再携带 `sandbox_path`
+ 复用设计：
    + 需要抽出统一的 `sandbox_path` metadata 注入函数，供 `/v1/chat/completions`、`cli/get`、内部 cron/chat 转发共用
    + 避免三处各自拼接，减少后续字段不一致
    + 该函数的职责应明确为：
        + 输入：顶层 `metadata map` + `chatId`
        + 输出：按当前会话状态对 `metadata.sandbox_path` 执行补充、覆盖或删除
+ 测试要求：
    + 需要补充自动化测试，至少覆盖以下情况：
        + 当前会话为 `filepick`，且 `allowed_dir` 非空时，`/v1/chat/completions` 转发包含顶层 `metadata.sandbox_path`
        + 当前会话为 `filepick_net`，且 `allowed_dir` 非空时，`cli/get` 上报包含顶层 `metadata.sandbox_path`
        + 当前会话为 `net` 时，不包含 `sandbox_path`
        + 当前会话为 `off` 或无记录时，不包含 `sandbox_path`
        + 当前会话 `allowed_dir` 为空串时，不包含 `sandbox_path`
        + 外部请求体手工传入错误的 `metadata.sandbox_path` 时，最终上游收到的值仍以当前会话真实目录为准
        + `memo`、`email`、`feishu` 这类 integration 内部转发链路至少覆盖一条，验证不是只有页面直发会带字段

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
