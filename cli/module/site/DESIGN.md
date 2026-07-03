# Site 模块详细技术设计

## 1. 模块定位

`site` 模块当前并不是一个独立的 Go Web 服务，而是一个以前端页面为中心的站点模块：

- 核心实现集中在单文件前端 [`index.html`](./index.html) 中，HTML、CSS、JavaScript 全部内嵌。
- 静态资源只有少量辅助文件，其中最关键的是 [`sw.js`](./sw.js)。
- Go 代码只承担两个很小的辅助能力：
  - [`filenamelookup`](./filenamelookup/)：按文件名在工作区内查找候选路径。
  - [`skillrepairprompt`](./skillrepairprompt/)：生成修复 `SKILL.md` 的标准提示语。

这意味着 `site` 的真实职责不是“提供一套前后端分离的 Web 服务”，而是：

- 提供 DeepRight 的主交互页面。
- 把聊天、Agent、VFS、沙盒、知识库、Cron、Quick Plugin、安装提示、引导流程等能力统一编排到一个浏览器页面里。
- 直接对接同源后端接口完成所有关键动作。

当前实现没有前端构建系统、没有模块化打包、没有独立 API SDK 层；页面逻辑通过全局函数直接访问大量 `/api/*` 与 `/v1/*` 接口。

## 2. 代码边界

### 2.1 前端主页面

- [`index.html`](./index.html)
  - 模块核心。
  - 同时包含页面结构、主题样式、聊天渲染、右侧面板、VFS、引导、缓存、轮询调度、模型设置、插件状态、知识库面板等所有前端逻辑。
  - 代码规模很大，属于单体脚本架构。

### 2.2 Service Worker

- [`sw.js`](./sw.js)
  - 当前仅负责消息图片缓存。
  - 缓存名固定为 `deepright-message-images-v1`。
  - 安装后 `skipWaiting()`，激活时清理同前缀旧缓存并 `clients.claim()`。
  - 只拦截满足以下条件的请求：
    - `GET`
    - `request.destination === 'image'`
    - URL 协议为 `http:` 或 `https:`
  - 不承担离线 App Shell 缓存，也不缓存 HTML、CSS、脚本或接口响应。

### 2.3 Go 辅助包

- [`filenamelookup/lookup.go`](./filenamelookup/lookup.go)
  - 对外暴露 `NormalizeCandidate` 与 `Lookup`。
  - 只接受“文件名”，不接受路径。
  - 搜索顺序固定为：
    - `workspace`
    - `tmp`
    - `images`
  - 搜索 `workspace` 时会排除 `tmp/` 和 `images/`，避免重复结果。

- [`skillrepairprompt/prompt.go`](./skillrepairprompt/prompt.go)
  - 对外暴露 `NormalizeSkillPath` 与 `Build`。
  - 只接受绝对路径，且目标文件必须名为 `SKILL.md`。
  - 固定产出格式：
    - `参考[https://agentskills.io/specification]修复<path>的错误。`

### 2.4 CLI 包装

- [`cmd/filenamelookup/main.go`](./cmd/filenamelookup/main.go)
  - 参数：
    - `-root`
    - `-name`
  - 调用 `filenamelookup.Lookup`，输出 JSON。

- [`cmd/skillrepairprompt/main.go`](./cmd/skillrepairprompt/main.go)
  - 参数：
    - `-path`
  - 也支持读取第一个位置参数。
  - 调用 `skillrepairprompt.Build`，输出单行提示语。

## 3. 前端总体架构

`site` 当前采用“单页应用外观 + 单文件脚本实现”的组织方式。实现上没有组件化边界，主要通过全局状态、全局函数和 DOM 约定协作。

### 3.1 状态分层

页面状态主要分成四层：

- 运行时内存状态
  - 主状态对象 `state` 保存当前会话、设置、主题、浏览器性能档位、知识库面板状态等。
  - 配套映射保存每个 chat 的 Agent、模型、Thinking、HTML 输出、Swarm、Sandbox、流式状态、恢复轮询状态等。

- `localStorage`
  - 设置通过 `loadSettings()` / `saveSettingsToStorage()` 维护。
  - 当前只持久化：
    - `agentId`
    - `knowledge_disable`
  - 模型 token 不再落本地存储，而是运行时从 `/api/token` 获取。

- 会话持久化
  - 使用 `CHATS_KEY = 'deepright_chats'` 保存：
    - `chats`
    - `order`
    - `agents`
    - `models`
    - `thinking`
    - `htmlOutput`
    - `swarm`
  - 单个聊天消息数受常量限制：
    - `MAX_STORE = 2000`
    - `MAX_SHOW = 500`
    - `LAZY_PAGE = 20`
    - `MAX_CLI_SUB_SHOW = 100`

- 浏览器缓存层
  - `IndexedDB`
    - DB：`deepright-message-images-db`
    - store：`images`
  - `Cache Storage`
    - cache：`deepright-message-images-v1`
  - 二者配合用于消息图片缓存与失效恢复。

此外，页面还在 `localStorage` 中保存大量引导状态 key，用于控制首次引导、特定功能引导是否重复展示。

### 3.2 初始化时序

页面主入口是 `async function init()`。当前初始化流程大致为：

1. 注册消息图片缓存相关 Service Worker。
2. 注册各类 overlay、屏保、consume 弹层、引导事件、删除确认事件。
3. 启动技能告警轮询与待安装应用轮询。
4. 渲染 Quick Fan Menu，更新 standalone 按钮。
5. 加载本地设置，再通过 `/api/token` 补全模型配置。
6. 读取 standalone 状态。
7. 识别浏览器性能档位，决定是否启用节能模式。
8. 加载 quick plugins 及其状态。
9. 从 `localStorage` 恢复 chats。
10. 应用主题、侧栏状态、布局观察器与滚动懒加载。
11. 初始化 VFS、模型指示器、右侧 wiki 面板。
12. 启动页面级调度器与可见性相关刷新逻辑。

这说明 `init()` 同时承担“页面装配”“状态恢复”“定时任务启动”“多个子系统注册”四类职责，是整个前端的启动协调中心。

### 3.3 页面级调度器

页面内存在统一调度器：

- `startPageTaskScheduler()`
- `pageTaskSchedulerStep()`

调度器按固定周期执行页面任务，并根据可见性、低功耗渲染模式、节能模式动态调整间隔。当前纳入调度的任务包括：

- `heartbeat`
- `installApp`
- `chatAutoReload`
- `vfs`
- `skillsWarnings`
- `cron`
- `quickPlugins`
- `wiki`
- `theme`

因此 `site` 不是“只在用户点击时触发动作”的纯事件页面，而是带有明显前端编排器特征的长生命周期控制台。

## 4. 聊天子系统设计

聊天链路是 `site` 的核心。其实现重点不在 UI，而在如何把浏览器页面状态、模型配置、Agent 绑定、Quick Plugin 状态和流式返回整合为一次请求。

### 4.1 会话模型

聊天通过 `state.chats` 和 `state.chatOrder` 维护，支持：

- 新建会话
- 指定自定义 chatId
- 删除会话
- 切换会话
- 按会话绑定 Agent 与模型
- 按会话保存 Thinking / HTML / Swarm 开关
- 恢复输入草稿、CLI+Sub 面板、Sandbox 状态

`uuid()` 使用的是基于 `Math.random()` 的自定义 UUID 生成器，而不是 `crypto.randomUUID()`。

### 4.2 发送链路

发送入口主要是：

- `handleSend()`
- `sendCenterMessage()`

发送前会完成以下准备动作：

- 判断当前 chat 是否忙碌，必要时转为排队或触发取消。
- 解析 request source 与附加 metadata。
- 根据当前 chat 或 UI 开关整理：
  - `thinking`
  - `html`
  - `router_disable`
  - `knowledge_disable`
- 校验当前 chat 绑定的 Agent 是否有效。
- 读取 Agent 的 router remote 配置。
- 解析当前所选模型，并从 `/api/token` 获取到的模型配置中取出 token。

构造请求时，发送体顶层字段为：

- `model`
- `messages`
- `stream: true`
- `thinking`
- `html`
- `router_disable`
- `metadata`

其中 `metadata` 至少包含：

- `agentId`
- `chat`
- `type: 'page_session'`

如果存在特殊来源或增强能力，还会附加：

- `requestSource`
- `knowledge_commit`
- `profile_commit`
- `router_remote`
- `plugins`
- 以及其他 request metadata

Quick Plugin 一旦处于运行中状态，会把 plugin key 列表附加到 `metadata.plugins`。

### 4.3 流式响应

`site` 没有使用 `EventSource`，而是通过：

- `fetch('/v1/chat/completions', ...)`
- `resp.body.getReader()`
- `TextDecoder`

手动读取并解析 SSE 文本流。处理方式是：

1. 逐块读取响应字节流。
2. 按换行切分。
3. 只处理以 `data:` 开头的行。
4. 遇到 `[DONE]` 结束流式过程。
5. 解析 JSON payload，并按 `choice.delta.content` 增量写入当前 assistant 消息。

这种实现方式比 `EventSource` 更灵活，允许前端：

- 自己控制 `Authorization` 头。
- 统一处理错误响应、retry 提示与中断逻辑。
- 在同一条 SSE 流里识别额外业务字段。

### 4.4 `cli + sub` 特殊流转

当前流式解析逻辑里内置了特殊分支：

- 当 SSE 记录满足 `parsed.biz === 'cli' && parsed.workflow === 'sub'` 时，
- 不把输出渲染在聊天中间区域，
- 而是写入右侧 `cliSub` 面板。

同时，消息对象会保留：

- `cliSubContent`
- `cliSubGroupId`

这说明 `site` 的聊天消息并不只是“纯文本助手回复”，还承担一部分多通道结果汇聚职责。

### 4.5 取消与恢复

取消流程由 `stopStreaming()` 驱动，特点如下：

- 本地先把当前 chat 标记为取消中。
- 中断当前 `AbortController`。
- 进入短暂 preflight/cancel 过渡态。
- 之后调用 `/api/cancel?agentId=...&chat=...` 重试取消请求。

页面同时具备恢复逻辑，当前实现会结合：

- `/api/restore`
- `restore polling`
- `lastRestoreAt`
- `lastRestoreId`

在页面切换、自动轮询或恢复场景下继续拉取未完成响应。聊天 UI 因此具备“流式 + 断点恢复”的双态能力。

## 5. 本地持久化与缓存

### 5.1 设置与模型

- `loadSettings()` 只从 `localStorage` 恢复：
  - `agentId`
  - `knowledge_disable`
- `loadTokenSettings()` 再从 `/api/token` 读取模型配置。
- 页面不把模型 token 重新写回 `localStorage`。

这说明“本地持久化设置”和“后端下发模型密钥”已经分离，页面只缓存轻量偏好，不缓存敏感 token。

### 5.2 会话历史

聊天历史完全保存在 `localStorage` 中，并在发送、结束、删除、切换时频繁调用 `saveChats()`。

当前保存内容不仅包含消息，还包含：

- 会话顺序
- 每个会话所绑定的 Agent
- 每个会话所绑定的模型
- Thinking / HTML / Swarm 开关

因此 `site` 的会话在语义上更接近“带完整运行上下文的页面会话”，而不是单纯消息列表。

### 5.3 消息图片缓存

消息图片缓存由页面脚本和 Service Worker 共同完成：

- 页面侧负责：
  - 识别可缓存图片 URL
  - 主动拉取图片
  - 把图片 blob 写入 IndexedDB
  - 在图片加载失败时从 IndexedDB / Cache Storage 回填
  - 在消息渲染后预热图片缓存

- Service Worker 负责：
  - 拦截消息图片请求
  - 优先返回 Cache Storage 中的命中结果
  - 回源抓取后写入缓存

这套设计的目标不是做全站离线化，而是尽量保证历史消息中的远程图片在失效、跨域或再次打开页面时仍可恢复。

## 6. 功能分区

虽然所有逻辑都塞在一个 `index.html` 里，但页面已经演化出多个稳定的功能域。

### 6.1 中央聊天区

中央区负责：

- 消息渲染
- Markdown / HTML 输出切换
- 流式状态显示
- 重试、复制、编辑、取消
- SSE 错误与 retry 提示
- CLI+Sub 结果桥接

### 6.2 左侧会话与配置区

左侧区域负责：

- chat 列表与会话切换
- Agent 绑定
- 模型选择
- Thinking / Swarm / HTML 等开关
- Quick Fan Menu
- 新手引导入口承载

### 6.3 右侧面板

右侧承载的是多个“附属工作区”：

- memo / note
- cron 视图
- wiki / knowledge 面板
- `cli+sub` 输出
- 若干状态提示与附属操作

其中 knowledge/wiki 并不是独立前端模块，而是内嵌在 `site` 中的右侧知识库面板。

### 6.4 VFS 与 `@` 能力

页面内建虚拟文件系统能力，支持：

- 工作区文件浏览
- 目录打开
- 文件读取
- 文件编辑
- 删除
- 上传

输入框里的 `@` 菜单也直接依赖这些能力：

- `@文件` 通过 `/api/files?path=...` 获取候选文件
- `@技能` 通过 `/api/skills?agentId=...` 获取技能列表

### 6.5 Standalone / Shutdown / Consume

页面并不只是聊天前端，还内置运行时控制能力：

- `loadStandaloneModeStatus()`
- `toggleStandaloneMode()`
- consume overlay 读取 `/api/consume`
- 关闭应用调用 `/api/shutdown`

这说明 `site` 实际上还是运行时控制台的一部分。

### 6.6 引导与体验层

页面中存在大量 onboarding / tour 状态与流程控制逻辑，覆盖：

- 首次使用
- 模型切换
- 技能提取
- 插件
- Wiki
- 输入区能力
- 其他高级功能入口

这些引导不是独立脚本，而是直接和真实 DOM、真实按钮、真实状态变量耦合。

## 7. 与后端 API 的耦合边界

`site` 当前对后端接口高度耦合，而且这种耦合是“前端直接 fetch 后端路径”的硬耦合，没有中间抽象层。

### 7.1 聊天与会话

- `/v1/chat/completions`
- `/api/restore`
- `/api/cancel`
- `/api/consume`

### 7.2 Agent 与运行时

- `/api/token`
- `/api/standalone`
- `/api/standalone=true|false`
- `/api/shutdown`
- `/api/agentId`
- `/api/agent/init`
- `/api/agent/delete`
- `/api/agent/export`
- `/api/agent/import`
- `/api/heartbeat`

### 7.3 工作区与 VFS

- `/api/workspace`
- `/api/data`
- `/api/raw`
- `/api/edit`
- `/api/del`
- `/api/files`
- `/api/folder`
- `/api/upload`

### 7.4 Cron / Memo

- `/api/cron/create`
- `/api/cron/delete`
- `/api/cron/detail/list`
- `/api/cron/detail/status`
- `/api/cron/detail/metadata`

### 7.5 Sandbox / Skills / Plugins

- `/api/sandbox=*`
- `/api/sandbox_status`
- `/api/skills`
- `/skills_warning`
- `/install_app`
- `/log_skill_status`

### 7.6 Knowledge / Wiki

- `/knowledge`
- `/knowledge_lastUpdate`
- `/knowledge_path`

另外，知识整理并不只靠专门接口，还会通过聊天请求触发 `knowledge_tidy` 类型的 page session 请求。

## 8. Go 辅助能力设计

### 8.1 `filenamelookup`

该包的目标是为“只知道文件名、不知道完整路径”的场景提供确定性查找。

实现特点：

- 输入必须是绝对工作区根目录。
- 文件候选名经过 `NormalizeCandidate()` 清洗：
  - 去空格
  - 去常见包裹符号
  - 禁止出现路径分隔符
- 查找逻辑采用广度优先遍历目录树。
- 结果按 scope 分组并保持稳定顺序：
  - 先 workspace
  - 再 tmp
  - 再 images
- 使用 `seen` 去重，避免同一路径重复输出。

输出结构：

```go
type Match struct {
    Scope Scope  `json:"scope"`
    Path  string `json:"path"`
}
```

### 8.2 `skillrepairprompt`

该包的目标是把无效 skill 文档路径转换为统一修复提示。

实现特点：

- 只接受绝对路径。
- 文件名必须精确为 `SKILL.md`。
- 输出是固定模板文本，而不是可配置 prompt。
- 返回值使用 `/` 风格路径，便于跨平台呈现。

这两个包都很小，但都刻意做了输入约束，避免上层调用传入歧义参数。

## 9. 测试现状

当前 Go 层测试只覆盖两个辅助包，没有覆盖前端主页面逻辑。

### 9.1 `filenamelookup` 测试

[`filenamelookup/lookup_test.go`](./filenamelookup/lookup_test.go) 当前覆盖：

- `NormalizeCandidate` 的清洗与路径拒绝逻辑。
- `Lookup` 的搜索顺序：
  - workspace
  - tmp
  - images
- 当 `tmp/`、`images/` 不存在时跳过可选 scope。

### 9.2 `skillrepairprompt` 测试

[`skillrepairprompt/prompt_test.go`](./skillrepairprompt/prompt_test.go) 当前覆盖：

- `Build("/tmp/demo/SKILL.md")` 的固定输出。
- 非绝对路径、空路径、非 `SKILL.md` 文件名时的报错。

### 9.3 缺失部分

当前没有以下自动化保障：

- `index.html` 的单元测试
- 聊天 SSE 解析测试
- localStorage / IndexedDB 行为测试
- 右侧面板、VFS、Cron、Wiki、Plugin 流程测试
- Service Worker 自动化测试

因此 `site` 的主要风险仍集中在前端回归与接口契约漂移。

## 10. 当前实现约束

### 10.1 单文件前端单体

`index.html` 已经承担过多职责：

- 页面结构
- 样式
- 状态管理
- 接口调用
- 流式协议解析
- 各类面板与引导

这种结构降低了初始接入成本，但会持续提高维护与局部演进成本。

### 10.2 后端契约强耦合

页面直接拼装 `/api/*` 与 `/v1/*` 路径，且大量逻辑依赖返回 JSON 结构中的约定字段。后端一旦修改字段名、状态码语义或流式记录格式，前端容易同步失效。

### 10.3 无构建与类型系统保护

当前没有：

- TypeScript
- 前端 bundler
- lint / test 的前端发布门禁

因此很多问题只能在运行时暴露。

### 10.4 缓存能力有限

虽然引入了 Service Worker，但它只缓存消息图片，并不提供完整离线站点能力。聊天、Agent、VFS、Cron、Knowledge 等核心能力仍依赖同源后端接口在线可用。

### 10.5 前端与运行时控制面板混合

`site` 同时承担：

- 聊天产品界面
- 运行时控制台
- 文件管理器
- 知识面板
- Cron 管理台

这让页面能力很强，但也使功能边界持续扩张，模块内聚度偏低。

## 11. 演进建议

在不改变当前功能的前提下，后续如果继续演进，建议优先按以下方向拆分：

1. 先把 `index.html` 中的 API 访问封装成统一数据访问层。
2. 再把聊天流式链路、VFS、Wiki、Cron、Quick Plugin 拆成独立脚本模块。
3. 为 SSE 解析、消息持久化、图片缓存恢复补齐最基本的前端测试。
4. 明确区分“聊天 UI”与“运行时控制台”边界，减少页面继续膨胀。

在当前版本中，`site` 的设计重点不是“模块化优雅”，而是“以单页面快速承载尽可能多的控制能力”。本设计文档因此以真实实现边界为准，而不以理想架构为准。
