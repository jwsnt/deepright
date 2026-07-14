### 第一性原则
+ 仅可以新增/更新/删除site（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Site介绍：../../REQUIREMENT.md
+ Site手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../REQUIREMENT.md
+ 所以设计/编译都需要遵守site页面的现有前端收口原则

### 需求介绍
+ 本次需要补写 site 前端关于本地存储压缩与丢弃策略的技术收口，明确哪些数据允许只保留运行时、哪些数据仍需要持久化：
    + 当前 `deepright_chats` 仍会整包写入浏览器 `localStorage`
    + 现网暴露出的主要体积风险并不是右侧 URL 预览 iframe 本身，而是 chat payload 中被持久化的超大字段
    + 其中已确认的高风险来源是 `cliCommandHistory.output`：单条命令输出即可达到数百 KB，远大于普通聊天正文
+ 右侧 URL 预览相关状态需要明确边界：
    + 右侧 iframe 只负责当前页面内展示远端页面，不应把 iframe 页面内容、DOM、响应正文或截图副本写入 `localStorage`
    + chat 级别若需要保留 URL 预览状态，也只能保留轻量元数据，例如 `active / href / title / subtitle`
    + 不允许把“右侧能看到的网页内容”误当作需要本地持久化的聊天历史
+ 右侧 CLI 子任务历史需要重新定义持久化原则：
    + 右侧 CLI 子任务的目标是“当前页展示 + 刷新后可由 `/api/restore` / live poll 重建”，而不是“在浏览器本地长期缓存完整命令输出”
    + 因此 `cliCommandHistory` 应视为运行时状态，而不是 chat 的长期本地持久化内容
    + 页面刷新前，右侧面板仍可继续使用运行时内存中的 `cliCommandHistory` 立即展示
    + 页面刷新后，右侧面板允许在 `/api/restore` 返回前短暂为空，再由恢复结果重建 `cli/get`、`cli/pub` 对应展示
    + 本次需求需要明确：`cliCommandHistory` 不再进入 `deepright_chats` 的本地持久化 payload
+ 本地存储压缩与丢弃需要显式区分“持久化裁剪”和“运行时展示”：
    + 运行时内存中的右侧 CLI 子任务可继续保留最近窗口，用于当前页即时展示
    + 写入 `localStorage` 时，必须允许主动丢弃 `cliCommandHistory`
    + 这类丢弃只影响浏览器本地副本，不影响当前页已展示内容，也不影响服务端 `/api/restore` 的历史恢复能力
+ 本次压缩/丢弃顺序需要写清楚，禁止继续采用“只限条数、不限单条体积”的半收口策略：
    + 第一优先级：明确不再持久化 `cliCommandHistory`，从根源上切断超大 `output` 进入 `localStorage`
    + 第二优先级：继续清理 assistant 消息中仅用于恢复/回放的冗余字段，例如已完成消息的 `rawSse`
    + 第三优先级：继续压缩非活跃 chat 的低价值本地副本，再视预算压力决定是否裁剪更多已完成正文窗口
    + 核心目标仍是“尽可能保住中心会话可直接阅读的正文”，而不是保住右侧 CLI 子任务的完整本地镜像
+ 对 `cliCommandHistory` 的技术要求需要写实：
    + 即使运行时仍保留 `cliCommandHistory`，也不能只做“最近 30 条”条数限制而不做字段级风险判断
    + 类似 `cat png`、二进制转文本、超长命令输出等明显高体积内容，不应再作为本地持久化候选
    + 本次需求的正式收口不是“把超大 output 截短后继续持久化”，而是“右侧 CLI 历史默认不进入页面本地持久化”
+ 刷新恢复语义需要保持稳定：
    + 去掉 `cliCommandHistory` 的本地持久化后，不能破坏现有 `/api/restore` 对 `cli/get`、`cli/pub` 的恢复展示
    + 正在执行中的会话刷新后，右侧 CLI 子任务允许先空窗、再逐步恢复，但不能因此丢失服务端已存在的 CLI 历史
    + 不得因为去持久化 `cliCommandHistory` 而破坏聊天主区正文恢复、任务 badge 恢复、footer inline hint 恢复等现有链路
+ 本次需求的验收重点如下：
    + 单个 chat 不得再因为 `cliCommandHistory.output` 而把 `localStorage` 顶到数 MB
    + 右侧 iframe 预览不应成为本地存储爆满的主因
    + 右侧 CLI 子任务在当前页仍可正常展示；刷新后可通过 `/api/restore` / live poll 重建
    + `deepright_chats` 的长期本地持久化应优先保留聊天正文和必要会话元数据，而不是完整 CLI 输出镜像

### 编写代码
+ 以现有site页面技术栈编写以上代码，要求：
    + 在../../index.html中按现有HTML/CSS/JavaScript组织方式实现
    + 不引入新的构建流程和额外运行时依赖
    + 代码简洁，尽量复用现有 onboarding 模板、设置面板底部 action 区、状态持久化和 queue/pending 清理逻辑
    + 总开关的“草稿态”和“真实生效态”必须显式分离，禁止在设置内点击时直接改动真实运行态
    + onboarding 右上角 `×` 的显隐规则必须模板统一，禁止为单个引导逐个手写特殊 DOM 分支
    + 各类 onboarding 启动入口必须统一尊重总开关，不能只拦一部分 start 逻辑而遗漏 queue/pending 或异步分支
    + 引导整条跳过后，不得留下会反复补播的 pending 状态
    + 本次本地存储收口必须优先复用现有 chat payload 构建、`saveChats()`、`/api/restore`、live cli poll 与右侧 CLI 子面板重建逻辑，禁止新造一套独立的 CLI 本地缓存体系
    + 去掉 `cliCommandHistory` 的本地持久化后，运行时展示与刷新恢复链路必须显式分流，禁止把“当前页内存态”和“浏览器长期持久化态”继续混为一谈
    + 若需要兼容旧版 `localStorage` 中已落盘的 `cliCommandHistory`，必须在加载时做最小侵入清理或忽略，避免旧大对象继续长期占用空间
    + 能用现有开源包和浏览器能力的就不要重复造轮子
+ 最小范围更新

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
