### 第一性原则
+ 仅可以新增、更新或删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../site/index.html` 与 Site 使用手册。

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
+ 不新增外部 Go 依赖，不改变既有请求正文、响应语义、SSE、任务队列或 `/cli/pub` 协议。

### 需求介绍
+ Integration 向上游发送请求时，必须在顶层 `metadata.version` 上报当前应用程序的客户端版本。该值唯一来自 Integration 实际生效的主 `config/config.json` 的字符串字段 `version`，不得使用 Agent 工作目录、HTTP 调用方、工作目录、环境变量或其他配置文件中的同名字段。
+ `version` 与主配置在启动时一并读取，可保存在进程内存中；当前进程运行期间主配置文件后续修改不会改变已缓存的客户端版本。
+ 仅对以下由 Integration 发往上游的请求注入字段：
    + 内置 cli-get 心跳的 `POST /cli/get`。
    + 普通会话、备忘录任务（包括飞书、邮件等 Connect 消息创建的即时任务）和设置页模型测试发出的 `POST /v1/chat/completions`。
    + 不修改 `/cli/pub`，也不扩展独立 `cli-get` 可执行程序的请求 schema。
+ `metadata.version` 是 Integration 运行时托管字段：在普通会话合并浏览器传入 metadata 后，必须删除并以缓存的主配置版本重写 `version`；cli-get、备忘录/Connect 任务和模型测试同样只能写入该值。除新增字段外，保持所有既有 metadata 字段及请求行为不变。
+ 页面直接发起的普通会话和设置页模型测试上游 `/v1/chat/completions` 请求，必须在顶层 `metadata.theme` 上报本次页面主题：冷色模式为 `cold`，暖色模式为 `warm`。页面每次请求都显式传入该值，Integration 不缓存、推测或改写它。
+ 模型测试还必须按普通会话相同的 metadata 层级传递非空 `metadata.device`；Integration 只接受 `cold` 或 `warm`，并在转发模型测试时保留 `device` 与 `theme`。备忘录、Connect 任务、内置 `/cli/get` 与 `/cli/pub` 不携带 `metadata.theme`。
+ 上游返回 HTTP 或 SSE 业务码 `400` 时，普通会话和模型测试必须统一显示“服务商请求无效（400），请检查模型地址、模型名称与请求参数”，不得直接展示服务商原始错误内容。
+ 验收要求：
    + 主配置 `version` 在启动后被缓存，覆盖的请求均携带完全一致的 `metadata.version`。
    + `/cli/get`、普通会话、备忘录/Connect 任务和模型测试均携带主配置版本。
    + 普通会话传入伪造 `metadata.version` 时，上游只收到本机主配置的缓存版本。
    + `/cli/pub` 与独立 `cli-get` 不新增 `metadata.version`。
    + 普通会话和模型测试均按页面当前模式携带正确的 `metadata.theme`；模型测试同时携带页面传入的 `metadata.device`。
    + `metadata.theme` 不写入缓存或持久化，也不出现在备忘录、Connect、`/cli/get`、`/cli/pub` 请求中。
    + 上游 400 的 HTTP 响应及 SSE 业务码均使用统一中文提示。

### 编写代码
+ 以 Golang 和现有页面代码完成最小范围更新，复用当前启动配置读取和 metadata 上游注入链路；不得新增外部依赖、配置副本、运行时配置文件、数据库表或 HTTP 接口。
+ 为主配置版本缓存、主题/设备参数校验及 `/cli/get`、普通会话、备忘录和模型测试的上游 metadata 补充测试。

### 撰写手册
+ 更新 Integration、Site 及本迭代目录的 `USER_GUIDE.md`，说明 `metadata.version` 的来源、缓存时机和适用请求范围，以及 `metadata.theme`、模型测试 `metadata.device` 的请求范围与取值。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求、边界和验收行为，不记录实现过程。
