### 第一性原则
+ 仅可以新增、更新或删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../site/index.html`、`../../../config/config.json` 与应用发布资源。

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
+ 不新增外部 Go 依赖，不改变既有 `/api/runtime_config`、普通消息发送、附件上传或 Agent 工作区访问协议。

### 需求介绍
+ Integration 的主 `config/config.json` 是每个上游请求关联的本机配置来源。向上游发送请求时，必须把当前实际生效的主配置文件绝对路径写入顶层 `metadata.config`，使上游任务能够明确知道应关联的本机配置文件。
+ `metadata.config` 的来源必须统一为 Integration 已解析的主配置路径，不得从 HTTP 调用方、Agent 配置、工作目录、进程当前目录或环境变量猜测。路径规则沿用当前启动配置规则：
    + macOS `.app` 为 `<App>.app/Contents/Resources/config/config.json`。
    + macOS 直接运行二进制、Linux 与 WSL 为 `<integration 可执行文件所在目录>/config/config.json`；常规 WSL 安装位置为 `~/deepright/config/config.json`。
    + 路径必须为清理后的绝对路径；不得创建、复制、写回或迁移 `config.json`。
+ 主配置文件是启动必需资源：
    + 已解析路径不存在、不是普通文件、无法读取或 JSON 非法时，Integration 启动必须失败并输出包含该绝对路径的清晰错误。
    + 已启动后，如果该文件被删除、替换为非普通文件或变为不可读取，则不得继续向上游发起本需求覆盖的请求；请求必须在本机失败，且不得发送缺失、空值或过期的 `metadata.config`。
    + 配置读取失败不得回退到 Agent 工作目录中的 `config.json`、旧运行目录、进程当前目录或内置空配置。
+ 仅对以下由 Integration 发往上游的请求注入字段：
    + 内置 cli-get 心跳的 `POST /cli/get`。
    + 普通会话的 `POST /v1/chat/completions`。
    + 备忘录任务的 `POST /v1/chat/completions`，包括由飞书消息、邮件消息等 Connect 请求创建的即时任务。
    + 设置页模型测试的 `POST /v1/chat/completions`。
    + 不修改 `/cli/pub`，也不扩展独立 `cli-get` 可执行程序的请求 schema。
+ `metadata.config` 是 Integration 运行时托管字段：
    + 普通会话请求在合并浏览器传入 metadata 后，必须删除并以本机解析出的真实绝对路径重写 `config`；调用方不得伪造、覆盖或清空该值。
    + cli-get、备忘录/Connect 任务和模型测试同样只能写入本机解析出的真实绝对路径。
    + 除新增 `metadata.config` 外，保持所有既有 metadata 字段、请求正文、响应语义、SSE、任务队列、模型测试和 `/cli/pub` 行为不变。
+ 验收要求：
    + 覆盖 macOS `.app`、直接运行二进制、Linux/WSL 的主配置绝对路径解析规则。
    + 覆盖缺失、目录、不可读和非法 JSON 配置导致启动失败；已启动后配置失效时，覆盖的上游请求不会发送。
    + 覆盖 `/cli/get`、普通会话、备忘录任务（含飞书/邮件 Connect 任务）与模型测试均携带真实 `metadata.config`。
    + 覆盖普通会话调用方传入伪造 `metadata.config` 时，上游只收到真实本机绝对路径。
    + 覆盖 `/cli/pub` 不新增 `metadata.config`。

### 编写代码
+ 以 Golang 完成最小范围更新，复用当前启动配置路径解析、配置读取、metadata 组装和错误处理；不得新增外部 Go 依赖。
+ 不得新增配置副本、运行时配置文件、数据库表、HTTP 接口或请求参数；仅在既有上游请求的顶层 `metadata` 对象新增 `config` 字段。

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`，说明不同发布形态的主配置实际路径、配置文件为启动必需资源，以及哪些上游请求会携带 `metadata.config`。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求、边界和验收行为，不记录实现过程。
