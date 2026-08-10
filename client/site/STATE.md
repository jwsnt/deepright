# `index.html` 中 `agentId + chatId` / `chatId` 绑定梳理

以下内容只基于 `cli/module/site/index.html` 当前实现总结，重点看“前端在什么功能里把 `agentId` 和 `chatId` 视为一组上下文”，以及“什么功能只按 `chatId` 就能成立”。

## 1. 基础绑定模型

- 站内会话主键是 `chatId`。消息正文存在 `state.chats[chatId]`。
- Agent 与会话的绑定关系存在 `chatAgents[chatId]`，统一通过 `getChatAgentId(chatId)` 读取。
- `createNewChat()` 新建会话时，会默认把当前 `state.settings.agentId` 绑定到新 `chatId`。
- `switchChat()` 切换会话时，会恢复该 `chatId` 绑定的 Agent、模型、Thinking/SWARM/HTML 开关等上下文。
- `bindAgentToChat(chatId, agentId)` 是显式改绑入口，会把当前 chat 切到新 Agent，并刷新 VFS、技能状态、沙盒状态、待发送队列等依赖 Agent 的能力。
- 备忘录时间线查看历史会话 `tlViewDo()` 时，也会把时间线明细里的 `agentId` 重新绑定到对应 `chatId`。说明“`chatId` 挂一个 Agent”是整个页面的基础约定。

## 2. 明确按 `agentId + chatId` 绑定的功能

| 功能 | 入口 / 接口 | 为什么必须同时带 `agentId + chatId` |
| --- | --- | --- |
| 会话沙盒 | `refreshChatSandboxStatus()` / `setActiveChatSandboxMode()`；`/api/sandbox_status?agentId=...&chatId=...`；`/api/sandbox=<mode>?agentId=...&chatId=...` | 沙盒本质上是“某个 Agent 工作区下、某个会话”的执行限制。换 Agent 或换 chat 都应视为新上下文。 |
| 技能列表与会话级技能过滤 | `showAtMenu()`、`loadSkills()`；`/api/skills?agentId=...&chatId=...` | `agentId` 决定该 Agent 原本能看到哪些技能，`chatId` 决定当前会话里哪些技能目录已被禁用。两者缺一不可。 |
| VFS 中技能目录状态投影 | `vfsRefresh()`；`/api/files?path=...&chatId=...` | 文件树本身来自 Agent 工作区，但是否显示 `skillDisabled` / `skillDisabledInherited` 是当前 chat 的禁用态投影。 |
| 待发送消息队列作用域 | `getQueuedCenterScopeKey()`、`getQueuedCenterQueue()`、`scheduleQueuedCenterRequestDrain()` | 队列 key 明确是 `agentId::chatId`。这样同一个 `chatId` 如果改绑到别的 Agent，旧队列不会串到新 Agent 上下文。 |
| message-insert 创建 | `postQueuedCenterMessageInsertAdd()`；`/api/message_insert/add` | 插入消息创建时同时写入 `agentId` 和 `chatId`，因为它既属于某个聊天流，也属于某个 Agent 执行上下文。 |
| 浏览器实例初始化 | `execBrowserInstanceCommand('instance init', binding)`；`/api/plugins/exec?...agentId=...&chatId=...` | 初始化浏览器实例时，需要把实例挂到当前 Agent + 当前 chat 的组合上下文。 |
| 即时插件配置启动流 | `pluginModalBuildActionPayload()`；`/api/plugins/config?...agentId=...&chatId=...` | 插件配置查询参数里同时承载 Agent 和 chat。只有插件声明相关 scope 时这两个值才真正生效，但前端能力模型已经支持 pair 绑定。 |
| 手动 CMD 与 CLI 终止 | `/api/cmd`、`/api/kill` 请求体都带 `agentId` 和 `chatId` | 这类操作既要定位 Agent 工作区，又要把结果挂回当前 chat 的运行流。 |
| 备忘录复用当前会话 | `rsSubmitNote()`；`/api/cron/create?agentId=...` 请求体可带 `chatId` | 定时任务本身归属 Agent；若勾选“复用当前会话”，则该 cron 明细还会绑定到当前 chat。 |
| 前端内部 pair-scope 状态 | `getWorkingIndicatorScopeKey()`、`getChatRestoreSkipScopeKey()`、`getQueuedCenterScopeKey()` | 工作指示、恢复跳过标记、待发送队列都明确按 `agentId::chatId` 隔离。 |

## 3. 明确只按 `chatId` 绑定的功能

| 功能 | 入口 / 接口 | 为什么只需要 `chatId` |
| --- | --- | --- |
| SSE 取消 | `requestChatCancel()`；`/api/cancel?chat=...` | 取消的是“这条聊天流”的进行中请求，后端直接按 chatId 终止即可。 |
| SSE 恢复 / 轮询补拉 | `performReloadChatRecords()`、`performLiveCliEventPoll()`；`/api/restore?chat=...&timeline=...&lastId=...` | 恢复逻辑跟随 chat 的消息流水，不需要再额外传 Agent。 |
| Chat Session Log 查询 | `submitChatSessionLogQuery()`；`/api/chat_session_log?chatId=...` | 面板里会显示 `agentId + chatId`，但真正查询只传 `chatId`。这里 Agent 更多是展示当前绑定上下文，不是查询键。 |
| message-insert 查询 / 删除 | `/api/message_insert/list?chatId=...`、`/api/message_insert/del`、`/api/message_insert/delete` | 插入消息创建完成后，后续追踪和清理都是“当前 chat 的待处理会话消息”问题，只需 chatId。 |
| 技能目录禁用开关写入 | `vfsToggleSkillDir()`；`/api/skill_state` 请求体 `{ chatId, path, disabled }` | 这里写入的是“当前会话对某个技能目录的禁用态”。目录由 `path` 唯一标识，状态按 `chatId` 保存。 |
| 浏览器实例关闭 | `execBrowserInstanceCommand('instance shutdown', { chatId })` | 实例初始化后，关闭动作直接按 chat 维度回收即可，前端不再强制携带 Agent。 |
| Footer inline hint / task badge 匹配 | `getChatFooterInlineHintScopeKey()`、`getChatTaskBadgeScopeKey()` | 代码注释已写明：这些 UI 提示只按 `chatId` 匹配，`agentId` 和 `__TARGET__` 不参与匹配或清理。 |

## 4. 容易误判的边界点

### 4.1 `chatSandbox` 表面按 `chatId` 存，实际仍是 pair 语义

- `chatSandbox` 的 map key 是 `chatId`。
- 但 `ensureChatSandbox()` / `getChatSandbox()` 会额外校验 `existing.agentId === binding.agentId`，不一致就视为不同快照。
- 所以它不是纯 chat 级缓存，而是“用 `chatId` 做入口、再用 `agentId` 做二次判定”的逻辑 pair 绑定。

### 4.2 `Chat Session Log` 面板展示双绑定，但请求只按 `chatId`

- `openChatSessionLogOverlay()` 会先解析当前 `agentId + chatId`，并把这两个值都显示到面板里。
- 但 `submitChatSessionLogQuery()` 真正请求时只发 `chatId`。
- 这说明它的“展示上下文”是 pair，“后端查询键”是 chat-only。

### 4.3 `/api/files` 在不同场景下是否带 `chatId` 是刻意区分的

- 主 VFS 列表 `vfsRefresh()` 会带 `chatId`，因为要拿到当前会话下的技能目录禁用态。
- `@文件` 搜索 `atFileSearch()` 不带 `chatId`，因为这里只是做普通文件路径补全，不需要 chat 级技能禁用投影。

## 5. 当前页面里的判断规律

可以把 `index.html` 里的绑定规则概括成一句话：

- 只要功能同时涉及“Agent 工作区 / Agent 配置”和“当前会话执行上下文”，就会走 `agentId + chatId`。
- 只要功能本质上是在追踪“这条 chat 的消息流、恢复流、日志流或 UI 提示流”，就通常只需要 `chatId`。

这个规律基本覆盖了当前页面里沙盒、技能、消息恢复、待发送队列、插件、CLI、备忘录复用等主要场景。
