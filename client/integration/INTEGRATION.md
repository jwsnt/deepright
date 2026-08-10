# 集成日志

## 基线集成

整合以下模块为统一 HTTP 服务（端口 `--port`，默认 8080）：

| 模块 | 路径 | 说明 |
|------|------|------|
| proxy | `POST /v1/chat/completions` | SSE 流式代理，注入 Agent 元数据后转发至上游 |
| static | `GET /site/*` | 静态文件服务，映射 `--site` 目录 |
| cli-get | 后台线程 | 心跳上报 + 任务执行 + 结果回传 |

共享参数：`--host`、`--agent-dir`、`--device`、`--agent-cache`

关联模块：agent（Agent 元数据扫描）、skills（技能元数据扫描）

---

## 迭代集成记录

当前浏览器收口说明：

- 当前可执行浏览器入口以 `module/connect/browser` 为准，服务层回归以 `module/connect/browserplaywrightsvc` 为准
- `module/connect/browser/playwright` 当前不再作为可直接 `go build` / `go test` 的 Go 包使用，目录中保留的是 Playwright 运行时资产与相关文档
- 因此浏览器相关的当前校验命令统一使用 `go test ./browserplaywrightsvc ./browser`

### 20260605 — integration 启动浏览器自动打开规则收口

来源：`module/integration/iteration/20260605-1/REQUIREMENT.md`

集成结果：

- integration 服务启动后会在后台延迟约 200ms 异步打开 `http://localhost:<port>/site/#app`
- 自动打开浏览器优先按操作系统规则查找指定浏览器，并附带最大化参数启动
- Windows 场景补齐 WSL 浏览器查找与回退逻辑；未命中优先浏览器时统一回退到系统默认浏览器
- 浏览器打开失败只记录日志，不影响 integration 主服务继续运行

同步说明：

- 新增 `module/integration/browseropen.go`
- 补充 `module/integration/browseropen_test.go`
- 更新 `module/integration/main.go`
- 更新 `module/integration/USER_GUIDE.md`

### 20260511 — knowledge 静态目录映射收口

来源：

- `module/integration/iteration/20260511-1/REQUIREMENT.md`
- `module/proxy/iteration/20260511-1/REQUIREMENT.md`

集成结果：

- `integration` HTTP 服务新增 `/knowledge` 与 `/knowledge/*` 路由
- 根目录统一映射到当前应用启动目录下的 `knowledge/`
- 访问目录时返回树形结构，访问文件时直接返回原始内容
- 保持根目录边界检查，拒绝 `..` 等越界路径

同步说明：

- 更新 `module/integration/main.go`
- 补充 `module/integration/main_test.go` 回归测试，覆盖目录树输出、文件读取与路径逃逸拦截
- 更新 `module/integration/USER_GUIDE.md`

### 20260512 — connect/browser playwright CDP User-Agent 收口补齐

来源：

- `module/connect/browser/playwright/iteration/20260511-1/REQUIREMENT.md`
- `module/integration/iteration/20260511-1/REQUIREMENT.md`

集成结果：

- `browserplaywrightsvc` 统一收口默认 Chrome UA 常量，避免 `browser` 与 `browser_playwright` 入口各自漂移
- `attach --cdp=...`、`create` 与受管 `Agent@Chat` 会话在连接已有 CDP 浏览器后，会继续把默认或显式 `--user-agent` 实际覆盖到当前页面与后续新页面
- integration 交付的 `plugins/browser` 因复用同一服务层，默认不再暴露 Linux headless Chrome UA

同步说明：

- 更新 `module/connect/browserplaywrightsvc/service.go`、`types.go`
- 补充 `module/connect/browserplaywrightsvc/service_test.go`
- 更新 `module/connect/browser/USER_GUIDE.md`、`module/connect/browser/playwright/USER_GUIDE.md`、`module/integration/USER_GUIDE.md`

### 20260508 — connect/browser playwright 命令级超时

来源：`module/connect/browser/playwright/iteration/20260508-2/REQUIREMENT.md`

变更：

- `browser_playwright` 所有 CLI 命令新增 `--browser-timeout` 总超时参数，默认 `120s`
- 超时覆盖 daemon 启动等待、命令请求处理以及 `create` 命令内部代理的 `browser_instance create`
- 命令超时后会主动终止本地 `browser_playwright` daemon，避免调用方进程被长时间阻塞

集成操作：

- 同步共享服务层 `browserplaywrightsvc` 的超时控制到 release 产物
- 更新 `module/connect/browser/playwright/USER_GUIDE.md` 与本集成手册中的浏览器交付说明
- 按当前收口后的包结构，验证 `go test ./browserplaywrightsvc ./browser` 通过

### 20260508 — connect/browser instance 独立 CDP 实例工具

来源：`module/connect/browser/instance/REQUIREMENT.md`

变更：

- 新增独立工具 `browser_instance`，按 `agentId + chatId` 启动和复用 Obscura CDP 服务
- 运行状态写入 `browser_instance.json`，并增加后台周期检查清理失效实例
- release 构建同时补齐 `browser_instance` 与 Obscura 运行时目录，避免交付后再手动拼装依赖

集成操作：

- 在 `module/build.sh` 中追加 `browser_instance` 的编译步骤
- 将 `module/connect/browser/obscura/release` 拷贝到 `module/release/browser/obscura/release`
- 更新 `USER_GUIDE.md` 的 release 目录说明，补充浏览器辅助工具交付路径

### 20260508 — connect/browser playwright create 自动实例附加

来源：`module/connect/browser/playwright/iteration/20260508-1/REQUIREMENT.md`

变更：

- `browser_playwright` 新增 `create` 命令，先按 `agentId + chatId` 调用相对路径 `../instance/browser_instance create`
- 接收到实例端口后自动按 `ws://127.0.0.1:<port>/devtools/browser` 执行 attach
- release 构建补齐 `browser/playwright/browser_playwright`，保证与 `browser/instance/browser_instance` 的相对目录关系稳定

集成操作：

- 在 `module/build.sh` 中追加 `browser_playwright` 的编译步骤
- 更新 `USER_GUIDE.md` 的 release 目录说明，补充 `browser_playwright` 与 `browser_instance` 的协作关系

### 20260508 — connect/browser 统一 browser 二进制交付

来源：

- `module/connect/browser/playwright/iteration/20260508-1/REQUIREMENT.md`
- `module/connect/browser/REQUIREMENT.md`

变更：

- `browser_playwright create` 改为强制使用 `Agent@Chat` 作为 Playwright 会话名，并通过 `ws://127.0.0.1:<port>/devtools/browser` attach
- `browser create` 同步采用相同语义，顶层帮助和手册统一改为 `browser` 单二进制口径
- integration 的浏览器交付从 `browser_instance + browser_playwright` 双产物收口为单一 `browser/browser`，并保留同目录 `browser/obscura/release/...`

集成操作：

- 修改 `module/build.sh`，改为编译 `module/connect/browser` 到 `module/release/browser/browser`
- 复制完整 `module/connect/browser/obscura/release` 到 `module/release/browser/obscura/release`
- 更新 `module/integration/USER_GUIDE.md` 的 release 结构与使用说明，避免继续传播旧的双二进制目录

### 20260508 — connect/browser playwright 自动注入 Chrome Cookie

来源：`module/connect/browser/playwright/iteration/20260508-3/REQUIREMENT.md`

变更：

- `browser_playwright` 在 `open`、`attach/create` 导航、`goto`、`tab-new` 等导航类命令中自动读取本机 Chrome Cookie
- Cookie 注入复用 `browser_cookie` 服务层，并按当前页面域名筛选后注入到 Playwright context
- 用户执行页面命令时不再需要额外先手动执行 `cookie-set`

集成操作：

- 同步更新 `browserplaywrightsvc`，让上层 `browser` 统一继承自动 Cookie 注入能力
- 更新 `module/connect/browser/playwright/USER_GUIDE.md`

### 20260508 — connect/browser playwright 导航诊断日志

来源：`module/connect/browser/playwright/iteration/20260508-4/REQUIREMENT.md`

变更：

- `browser_playwright` 新增导航诊断日志，记录 Cookie 缓存是否命中、是否跳过 host 重复注入、以及 `goto` 真实耗时毫秒
- 诊断日志默认写入程序同目录 `log/browser_log`
- 日志改为分卷滚动：单文件最大 `10MB`，最多保留 `4` 卷

集成操作：

- 在共享服务层 `browserplaywrightsvc` 中统一实现日志路径、滚动策略和导航诊断，保证上层 `browser` 自动继承
- 更新 `module/connect/browser/playwright/USER_GUIDE.md` 与本集成日志
- 按当前收口后的包结构，验证 `go test ./browserplaywrightsvc ./browser` 通过

### 20260506 — integration 重新构建收口

来源：`module/integration/REQUIREMENT.md`

变更：

- 保持 `integration` 作为唯一主二进制入口，继续统一承载 proxy、connect、cli-get、site/static、cron 能力
- 启动时如果 `--agent-dir` 指向空目录，则自动补齐 `DEF_AGENT/skills`
- 重新明确交付物构建路径：`module/release/integration` 与 `module/release/plugins/{feishu,email}`

集成操作：

- 为 `resolveServeAgentDir` 增加空目录脚手架初始化逻辑，并补充测试覆盖默认目录、显式空目录、非空目录三种场景
- 更新 `USER_GUIDE.md`，补充空目录初始化行为与 release 构建步骤
- 验证 `go test ./...` 通过，作为本次重新构建的回归基线

### 20260419 — agent 迭代

来源：`module/agent/iteration/20260419_1/REQUIREMENT.md`

变更：新增 `GetAgentIDs` 和 `GetAgentByID` 两个子模块 API，从 Agents 数组中获取 AgentId 列表或指定 Agent 元数据。

集成影响：内部 API 变更，integration 模块的 `getAgentOutput` 已返回完整 Agents 数组，可直接提取 AgentId 列表。无需额外代码变更，但为 proxy 迭代提供了基础能力。

### 20260419 — proxy 迭代

来源：`module/proxy/iteration/20260419/REQUIREMENT.md`

变更：新增 `GET /api/agentId` 接口，返回所有 Agent ID 的 JSON 数组。

集成操作：
- 新增 `handleAgentIDs` handler，从共享的 Agent 元数据缓存提取 agentId 列表
- 注册路由 `mux.HandleFunc("/api/agentId", handleAgentIDs(&cfg))`
- 新增测试 `TestApiAgentId` 验证返回 `["a","b"]`

### 20260419 — proxy metadata 合并

来源：`module/proxy/REQUIREMENT.md`（如果参数metadata已存在，则使用追加覆盖逻辑而不是替换）

变更：`handleChatCompletions` 注入 metadata 时，从直接替换改为追加覆盖——保留前端传入的字段（如 `agentId`、`chat`），同时注入 Agent 扫描元数据。

集成操作：
- 同步 proxy 模块的 metadata 合并逻辑至 integration 的 `handleChatCompletions`
- 使用 `json.Marshal` → `json.Unmarshal` 方式提取前端 metadata，避免类型断言失败

### 20260419_2 — proxy 连接超时

来源：`module/proxy/iteration/20260419_2/REQUIREMENT.md`

变更：SSE 代理 HTTP 客户端仅设置连接超时（`--connect_timeout`，默认 15000ms），不设置读取超时和总超时。

集成操作：
- Config 新增 `ConnectTimeoutMs` 字段
- 新增 `--connect_timeout` CLI 参数
- proxy HTTP 客户端使用可配置的连接超时，无总超时

### cli-get — cli/pub 上报 metadata

来源：`module/cli-get/REQUIREMENT.md`

变更：`cli/pub` 回传结果时携带 Agent 元数据（`metadata`），与 `cli/get` 一致。

集成操作：
- `PubRequest` 新增 `Metadata *AgentOutput` 字段
- `publishResult` 函数签名新增 `metadata` 参数
- `startCliGet` 中将 metadata 传入 worker 闭包并传递给 `publishResult`

### 20260510 — knowledge 元数据收口

来源：

- `module/knowledge/REQUIREMENT.md`
- `module/agent/iteration/20260510-1/REQUIREMENT.md`
- `module/proxy/iteration/20260510-1/REQUIREMENT.md`

变更：`integration` 入口层不再自行拼装 knowledge 字段，而是直接复用共享的 Agent 元数据内核输出；`/v1/chat/completions`、`cli/get`、`cli/pub` 与内部 cron 执行链路统一获得 `knowledge` 字段。

集成操作：

- `getAgentOutput(...)` 改为调用 `agentcore.GetOutputWithPluginsForApp(...)`
- `app-dir` 直接复用主应用 `config/config.json` / 启动目录现有解析规则
- 补充测试覆盖 `/v1/chat/completions` 与 `cli/get` / `cli/pub` 的 `knowledge` 元数据透传

设计说明：

- knowledge 字段实现继续保持单一 authoritative implementation，避免在 integration 再复制一份逻辑
- integration 只负责入口适配与路由转发，符合 `DESIGN.md` 的单一实现原则与入口薄包装原则

### site 20260419_2 — 会话数量上限

来源：`module/site/iteration/20260419_2/REQUIREMENT.md`

变更：左侧边栏会话数量上限 10 个，超出时提示删除后才可新建。

集成影响：site 为静态 HTML 文件，由 static 模块提供服务，无需 integration 代码变更。

### 20260419_3 — proxy /api/folder

来源：`module/proxy/iteration/20260419_3/REQUIREMENT.md`

变更：新增 `GET /api/folder?agentId=xxx` 接口，根据 agentId 打开对应 Agent 的 workspace 目录。

集成操作：
- 新增 `openFolder` 函数，按系统调用 open/xdg-open/explorer
- 新增 `handleFolder` handler
- 注册路由 `mux.HandleFunc("/api/folder", handleFolder(&cfg))`

### 20260501_1 — agent git 元数据

来源：`module/agent/iteration/20260501_1/REQUIREMENT.md`

变更：Agent 顶层元数据新增 `git` 字段，返回当前设备已安装 git 可执行文件的绝对路径；若未安装或无法探测则返回空字符串。

集成操作：
- integration 共享的 `AgentOutput` 结构新增 `git`
- 集成与 proxy 对齐为按操作系统分别探测：macOS/Linux 使用 `PATH` 与 `command -v git`，Windows 使用 `PATHEXT` 与 `where git`
- 保持 CLI、proxy 转发 metadata、cli-get 心跳与发布链路中的 Agent 元数据一致

### 20260503 — cron / proxy 最新任务创建能力

来源：

- `module/cron/iteration/20260502-3/REQUIREMENT.md`
- `module/proxy/iteration/20260503-1/REQUIREMENT.md`

变更：

- cron 任务元数据支持 `chatId`
- 支持高频周期 `cycle=3/4/5`
- 支持 `cycle=-1 + cron` 的自定义 Cron 表达式
- proxy 模块新增本地 CLI 创建任务能力，但 integration 继续以统一 HTTP 服务形式暴露该能力

集成操作：

- `POST /api/cron/create` 已支持 `chatId` 字段
- `task_meta` 与 `task_detail` 已包含 `chat_id`
- 创建任务时已支持 `cycle=3/4/5`，并在写入时展开未来 5 天窗口内的高频任务明细
- 后台执行器执行任务时，优先复用 `task_detail.chat_id`；为空时才回退到 `metaId@detailId`
- integration 新增 `cron create` / `cron create-cron` / `cron find-meta` / `cron find-detail` CLI 命令
- integration 默认不带子命令时仍保持原有统一 HTTP 服务启动行为
- CLI 与 HTTP 现在共用同一套 cron 创建与查询逻辑

### 20260503-18 — proxy 备忘录执行 metadata 任务类型

来源：`module/proxy/iteration/20260503-18/REQUIREMENT.md`

变更：备忘录任务明细执行时，请求 `/v1/chat/completions` 的 metadata 需附带 `cron_type`，值为普通任务的 `cron` 或插件桥接任务对应的插件 `key`。

集成操作：
- 同步 proxy 的定时任务执行链路到 integration 的 `cronExecuteOnce`
- 执行请求 metadata 新增 `cron_type`
- `META_ID` 继续保留“取最后一条 request ID”的行为，避免把整串 `meta_ref` 直接透传给上游

### 20260503 — proxy / site 模型密钥服务端持久化

来源：

- `module/proxy/iteration/20260503-1/REQUIREMENT.md`
- `module/site/iteration/20260503-1/REQUIREMENT.md`

变更：

- proxy 新增 `GET /api/token` 与 `POST /api/token`
- site 设置页中的模型与密钥不再保存在页面本地存储，而改为通过 `/api/token` 读取和保存

集成操作：

- integration 新增同名接口 `/api/token`
- 统一使用共享 SQLite `data` 中的 `token_store` 表存储模型密钥与更新时间
- 每次 token 更新都会写入 `token_store_log`
- site 侧本地仅保留 `agentId`、会话数据与当前页面状态，模型密钥由服务端返回
- 聊天请求与备忘录任务创建继续复用同一份模型密钥数据

### 20260503 — cron / proxy 动态获取模型 Token

来源：

- `module/cron/iteration/20260502-4/REQUIREMENT.md`
- `module/proxy/iteration/20260503-2/REQUIREMENT.md`

变更：

- 创建备忘录（定时任务）时，不再在 `task_meta` / `task_detail` 中存储模型 Token
- 执行任务时，改为根据模型名称动态从 SQLite `token_store` 读取密钥

集成操作：

- integration 的 cron 创建逻辑移除了 `token` 持久化
- 后台执行器执行任务时，会按 `model` 查询 `token_store`
- cron/proxy/integration 三处创建入口都会先校验 Agent 是否存在、模型是否已在 `token_store` 注册
- site 创建备忘录时不再向 `/api/cron/create` 发送 `token`

### 20260503 — proxy / integration 启动参数落盘

来源：

- `module/proxy/iteration/20260503-2/REQUIREMENT.md`
- `module/integration/iteration/20260503-2/REQUIREMENT.md`

变更：

- 以 HTTP 服务模式启动 proxy 或 integration 时，会将本次启动参数写入启动目录下的主应用 `config/config.json`
- 每次启动都会覆盖更新

集成操作：

- proxy 在 `runServe(...)` 启动前写入 `config/config.json`
- integration 在 HTTP 服务模式解析完参数后写入 `config/config.json`
- `integration cron create` / `integration cron create-cron` / `integration cron find-meta` / `integration cron find-detail` 仍不会写启动配置文件
- `proxy create` / `integration cron create` 在未显式传入 `agent-dir` 等非 cron 模块参数时，会优先从 `config/config.json` 读取

### 20260503 — cron / proxy 查询任务元数据与明细

来源：

- `module/proxy/iteration/20260503-5/REQUIREMENT.md`
- `module/integration/iteration/20260503-3/REQUIREMENT.md`

变更：

- 新增 cron 任务元数据查询与任务明细查询 CLI
- HTTP 与 CLI 使用同一套筛选维度
- 任务明细未指定时间条件时，默认优先查询当前时间之后的数据；已保留的已完成明细也会返回

集成操作：

- integration 新增 `cron find-meta` / `cron find-detail`
- `/api/cron/detail/metadata` 支持 `agentId`、`chatId`、`model`、`content`、`cycle`、开始执行时间范围
- `/api/cron/detail/list` 支持 `metaId`、`agentId`、`chatId`、`model`、`content`、`cycle`、执行时间范围
- `metaId` 同时兼容数字格式与 `cron_1` 这种前缀格式

### 20260503 — cron / proxy 删除任务元数据与明细

来源：

- `module/proxy/iteration/20260503-6/REQUIREMENT.md`
- `module/integration/iteration/20260503-4/REQUIREMENT.md`

变更：

- 新增 cron 任务元数据删除与任务明细删除 CLI
- HTTP 与 CLI 使用同一套删除筛选维度
- 删除任务明细时，未指定时间条件不会默认限制未来数据

集成操作：

- integration 新增 `cron delete-meta` / `cron delete-detail`
- `/api/cron/delete` 支持 `id` / `metaId` / `meta` 或元数据过滤条件

### 20260504 — proxy provider 日志与 cron 删除清理联动

来源：

- `module/proxy/iteration/20260503-7/REQUIREMENT.md`
- `module/proxy/iteration/20260503-8/REQUIREMENT.md`

变更：

- 模型密钥日志表统一为 `proxy_agent_provider_log`
- 如果共享数据库中仍存在旧表 `token_store_log`，启动时自动改名
- 删除 Agent 时，同步删除该 Agent 关联的全部 cron 元数据与全部 cron 明细
- `/api/cron/delete` 改为只联动删除尚未完成的任务明细，保留已完成明细
- 每分钟执行前会清理“已删除 Agent”关联的 cron 元数据与未完成明细，并跳过这些 Agent 的待执行任务

集成操作：

- integration 与 proxy 共用 `proxy_agent_provider_log`
- integration 启动时使用 `ensureTokenStore` 初始化 provider 日志表和索引
- integration 的 cron 查询、删除、状态更新继续写入 `cron_meta_log` / `cron_detail_log`
- integration 的后台 cron 执行器增加已删除 Agent 的清理与跳过逻辑
- `/api/cron/detail/delete` 支持 `detailId` / `metaId` 或明细过滤条件
- `delete-meta` 删除元数据时会同步删除关联的全部任务明细

### 20260504 — cron 数据库操作审计日志

来源：

- `module/cron/iteration/20260502-5/REQUIREMENT.md`

变更：

- cron 的任务元数据与任务明细数据库操作增加统一日志表
- 日志表按 `Agent + Chat + 时间` 建立索引

集成操作：

- integration 内置 cron 同步新增 `cron_meta_log` / `cron_detail_log`
- 任务创建、查询和执行状态更新会写入对应日志表

### 20260505 — connect 三方请求状态字段

来源：

- `module/connect/REQUIREMENT.md`

变更：

- connect 的三方请求数据新增 `status` 字段，枚举为 `0=待处理`、`1=已启动`、`2=已完成`、`3=已过期`、`4=已回复`
- 需要继续遵守 integration 的二进制与 CLI 收口原则

集成操作：

- `connect_request` 表新增 `status` 列，兼容已有 SQLite 数据自动迁移
- `connect add-request` / `integration connect add-request` 支持可选 `--status`
- `connect request-list` / `integration connect request-list` 支持按 `status` 过滤
- `POST /api/connect/request` 与 `GET /api/connect/request` 同步支持 `status`
- 未显式传入时，新增请求默认写入 `status=0`
- `add-response` 成功后自动将对应请求更新为 `status=4`
- 若历史库中存在旧相关日志表，会自动改名到新表名

### 20260504 — integration 默认启动目录与自动打开浏览器

来源：

- `module/integration/iteration/20260503-5/REQUIREMENT.md`

变更：

- HTTP 服务模式下，`--agent-dir` 默认值改为当前目录下的 `./agent`，目录不存在时自动创建
- `--site` 默认值改为当前目录下的 `./site`
- 服务启动成功后自动打开浏览器，访问 `http://127.0.0.1:<port>/site/#app`

集成操作：

- integration 启动参数默认值已调整，并在启动前统一解析为绝对路径
- `agent-dir` 在默认或显式指定目录不存在时会自动创建
- `config/config.json` 写入的是解析后的实际目录值
- 启动监听成功后会异步调用系统默认浏览器打开前端页面；失败仅记录日志，不影响服务

### 20260505 — proxy 插件元数据接口

来源：

- `module/proxy/iteration/20260503-9/REQUIREMENT.md`

变更：

- proxy 新增 `GET /api/plugins/meta`
- 接口底层合并 `connect list-plugins` 与 `connect meta-list`
- 返回可用插件的 `name`、`param` 与已填参数 `meta`

集成操作：

- integration 已同步注册 `GET /api/plugins/meta`
- integration 侧使用共享的 `connect-cache` 控制插件元数据缓存时长
- FEISHU 等插件可通过该接口被前端或其他模块动态发现

### 20260505 — proxy 插件日志 SSE 接口

来源：

- `module/proxy/iteration/20260503-10/REQUIREMENT.md`

变更：

- proxy 新增 `GET /api/plugins/log`
- 按“插件同名 `.log` 文件”定位日志，例如 `plugins/feishu` 对应 `plugins/feishu.log`
- 接口使用 SSE 持续返回，行为等同 `tail -f`
- 文件不存在时会返回错误事件并断开连接

集成操作：

- integration 已同步注册 `GET /api/plugins/log`
- integration 与 proxy 共用同样的插件日志路径解析规则：支持插件名、插件路径或日志路径
- 启动连接时会先补发最近若干行，再持续输出新增日志
- 已补充集成测试，覆盖历史回放、增量追加、日志文件删除三种场景

### 20260505 — proxy 插件状态接口

来源：

- `module/proxy/iteration/20260503-16/REQUIREMENT.md`

变更：

- proxy 新增 `GET /api/plugins/status?name=xxx`
- 接口底层复用 `connect list-plugins` 同一套插件扫描逻辑解析插件
- 通过插件对应 PID 文件判断进程是否仍在运行，默认规则为 `plugins/<key>.pid`；为兼容旧版 `browser` 运行态，仍额外兼容 `plugins/.browser_playwright/browser_playwright.pid`
- 同步补充 `proxy plugins status --name xxx` CLI 能力

集成操作：

- integration 已同步注册 `GET /api/plugins/status`
- integration 顶层 CLI 已同步提供 `integration plugins status --name xxx`
- 已补充 proxy / integration / connect 三层测试，覆盖已启动和未启动两种状态判断

---

## 集成检查清单

| 模块 | 迭代 | 状态 |
|------|------|------|
| agent | 20260419 (GetAgentIDs/GetAgentByID) | ✅ 已集成 |
| proxy | 20260419 (GET /api/agentId) | ✅ 已集成 |
| proxy | 20260419 (metadata 追加覆盖) | ✅ 已集成 |
| proxy | 20260419_2 (连接超时配置) | ✅ 已集成 |
| proxy | 20260419_3 (/api/folder) | ✅ 已集成 |
| proxy | 20260419_4 (/api/skills) | ✅ 已集成 |
| proxy | 20260419_5 (/api/files) | ✅ 已集成 |
| proxy | 20260419_6 (/api/data) | ✅ 已集成 |
| proxy | 20260419_7 (/api/workspace) | ✅ 已集成 |
| proxy | 20260419_8 (/api/edit) | ✅ 已集成 |
| proxy | 20260419_9 (/api/del) | ✅ 已集成 |
| proxy | 20260419_10 (/api/raw) | ✅ 已集成 |
| proxy | 20260419_11 (/api/heartbeat) | ✅ 已集成 |
| cron | 20260502-3 (chatId + 高频周期 + create/create-cron) | ✅ HTTP 与 CLI 能力已集成 |
| integration | 20260503-1 (cron create CLI 与 HTTP 共存) | ✅ 已集成 |
| proxy | 20260503-1 (任务创建 CLI 与 HTTP 共存) | ✅ HTTP 能力已集成 |
| proxy | 20260503-1 (/api/token 模型密钥存储) | ✅ 已集成 |
| proxy | 20260503-2 (HTTP 启动参数写入主应用 config/config.json) | ✅ 已集成 |
| cron | 20260502-4 (任务元数据/明细禁止存储 token) | ✅ 已集成 |
| proxy | 20260503-2 (执行时动态获取模型 token) | ✅ 已集成 |
| proxy | 20260503-9 (/api/plugins/meta) | ✅ 已集成 |
| proxy | 20260503-10 (/api/plugins/log) | ✅ 已集成 |
| proxy | 20260503-16 (/api/plugins/status) | ✅ 已集成 |
| integration | 20260503-2 (HTTP 启动参数写入主应用 config/config.json) | ✅ 已集成 |
| integration | 20260503-5 (默认 agent/site + 自动打开浏览器) | ✅ 已集成 |
| site | 20260503-1 (设置页改为使用 /api/token) | ✅ 已集成 |
| agent | 20260501_1 (git 元数据) | ✅ 已集成 |
| cli-get | cli/pub 上报 metadata | ✅ 已集成 |
| static | 无迭代 | ✅ 无需变更 |

### 20260505 — integration 顶层 start/stop/restart 收口

来源：

- `module/integration/REQUIREMENT.md`

变更：

- integration 顶层新增 `serve` / `start` / `stop` / `restart`
- 默认前台服务模式收口到 `integration serve`，同时保留 `integration [serve options]` 兼容行为
- `start` 进入后台进程并写入 PID 文件，默认 `./integration.pid`
- `stop` 先释放 HTTP 服务、cli-get、cron 定时器、连接池和活跃子进程，再安全退出
- `restart` 复用同一套 PID 文件和启动参数，先停后起

集成操作：

- 用户只需要感知 `integration` 一个二进制，不再要求单独理解或启动独立 connect/cli 进程
- integration 顶层 CLI 已按收口原则直接暴露生命周期命令
- 手册已同步更新后台启动、停止和重启方式

### 20260508 — connect/browser instance 指定实例查询

来源：

- `module/connect/browser/instance/REQUIREMENT.md`
- `module/connect/browser/REQUIREMENT.md`

变更：

- `browser_instance` 新增 `get --agentId --chatId`，用于返回指定 Obscura CDP 实例
- 实例记录统一补充 `cdp` 字段，格式固定为 `ws://127.0.0.1:<port>/devtools/browser`
- `browser` 收口层同步新增 `browser instance get --agentId --chatId`

集成操作：

- 同步更新 `module/connect/browser/instance/main.go`、`module/connect/browser/main.go`
- 更新 `module/connect/browser/instance/USER_GUIDE.md` 与 `module/connect/browser/USER_GUIDE.md`
- `module/build.sh` 继续以 `browser` 单二进制口径编译浏览器交付物，无需向最终用户额外暴露 `browser_instance`

### 20260508 — connect/browser instance 10 分钟空闲自动释放

来源：

- `module/connect/browser/instance/iteration/20260508-1/REQUIREMENT.md`

集成结果：

- `browser_instance` 状态文件新增 `lastActiveAt`，用于记录每个 `agentId + chatId` 实例的最近受管访问时间
- 默认后台监控间隔调整为 1 分钟，当实例空闲超过 `--browser_expired` 分钟时自动关闭对应 Obscura 进程并清理状态，默认 10 分钟
- `browser` 收口层在 `create` 和受管 `--session agent@chat` 命令上同步刷新实例活跃时间，避免仍在使用的会话被误回收

同步说明：

- 更新 `module/connect/browser/instance/main.go`、`module/connect/browser/instance/main_test.go`
- 更新 `module/connect/browser/instance_runtime.go`、`module/connect/browser/main.go`、`module/connect/browser/main_test.go`
- 更新 `module/connect/browser/instance/USER_GUIDE.md`、`module/connect/browser/USER_GUIDE.md`、`module/integration/USER_GUIDE.md`

### 20260509 — connect/browser_playwright 自动创建 CDP 实例

来源：

- `module/connect/browser/playwright/iteration/20260508-6/REQUIREMENT.md`

集成结果：

- `browser_playwright` 的普通 Playwright 命令在未显式传入 `--cdp` 时，如果检测到 `--agentId + --chatId` 或 `--session agent@chat`，会先调用 `browser_instance get`
- 若指定实例不存在，则自动补调 `browser_instance create`，不再要求调用方先手动执行 `create`
- 当 `--session` 与 `--agentId` / `--chatId` 同时出现且不一致时，统一以 `--session` 拆解出的 `agentId + chatId` 为准
- 自动接管后统一补齐 `ws://127.0.0.1:<port>/devtools/browser`，并让共享服务层与最终 `browser` 收口二进制同时继承该行为

同步说明：

- 更新 `module/connect/browserplaywrightsvc/instance.go`、`module/connect/browserplaywrightsvc/service.go`
- 补充 `module/connect/browserplaywrightsvc/service_test.go` 回归测试，覆盖 `get -> create` 回退与 `--session` 优先级
- 更新 `module/connect/browser/playwright/USER_GUIDE.md`、`module/connect/browser/USER_GUIDE.md`

### 20260508 — integration SSE 取消竞态修复

来源：

- 本轮运行时问题排查：`http: panic serving ... close of closed channel`

变更：

- 修复 `/api/cancel` 与 SSE 代理自然收尾同时关闭同一聊天通道时的竞态
- 通过 `sync.Once` 收口活跃连接通道关闭逻辑，避免重复 `close(channel)` 导致 HTTP handler panic

集成操作：

- 更新 `module/integration/main.go`
- 补充 `TestCloseActiveConnIsIdempotent` 回归测试，覆盖重复关闭场景
- 定向验证 `go test -run TestCloseActiveConnIsIdempotent ./` 通过

### 20260508 — integration 最新 release 构建脚本收口

来源：

- `module/integration/REQUIREMENT.md`
- `module/connect/browser/REQUIREMENT.md`

变更：

- `module/build.sh` 现已在构建前清理自身负责的 release 产物，避免历史残留二进制混入最新交付
- 浏览器交付目录保持为 `release/browser/browser + release/browser/obscura/release/...`
- 不再保留旧的 `release/browser/instance`、`release/browser/playwright` 历史残留目录

集成操作：

- 更新 `module/build.sh`
- 实际在 `module/` 目录执行 `sh ./build.sh` 完成最新编译
- 确认产物时间戳已更新到 `2026-05-08 21:54`，release 浏览器目录仅保留收口后的 `browser` 与 `obscura/release`
