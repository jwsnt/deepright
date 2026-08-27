### 第一性原则
+ 仅可以新增、更新或删除 integration（../..）同目录及其子目录与使用手册。

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
+ 保持既有 Agent 工作区边界、聊天、备忘录、Connect 任务队列与 SSE 协议兼容；不新增外部 Go 依赖或 HTTP 接口。

### 需求介绍
+ Integration 必须按 `chatId` 持久化可复用的会话历史。每个成功完成的模型轮次只记录两条 OpenAI Chat 格式的消息：当前输入的 `user` 消息和该轮聚合后的 `assistant` 文本消息；两者按写入顺序组成完整轮次。
+ 每条持久化历史必须包含毫秒级 `created`：`user` 使用 Integration 创建本次上游请求的本地时间，`assistant` 使用本地确认 SSE 完整结束的时间；下次注入上游 `messages` 时必须原样携带该字段。
+ 会话历史与现有用于恢复页面 SSE 的 `chat_log`、`agent_message_log` 分离保存。原有日志继续保存规范化请求和原始 SSE 事件，不改变已有恢复、导出、取消和诊断行为。
+ 对同一非空 `chatId`，Integration 在发送上游 `POST /v1/chat/completions` 前读取此前已经成功完成的历史轮次，并以 OpenAI `messages` 结构注入请求：
    + 消息按时间从旧到新排列。
    + 原请求开头连续的 `system`、`developer` 消息保持在最前。
    + 历史 `user`、`assistant` 消息位于系统指令之后。
    + 本次请求原有消息保留在最后；通常最新 `user` 消息为数组最后一项。
    + 只按 `chatId` 读取历史，Agent 切换不切断同一个会话的上下文。
    + 注入的历史 `user`、`assistant` 按单条消息计数，最多保留 `config/config.json` 中 `chat.restore` 指定的最新条数；超过时截断更旧消息。
+ 历史注入必须覆盖以下由 Integration 发往上游的最终聊天请求：
    + 页面普通会话的 `/v1/chat/completions` 转发。
    + 备忘录及其他定时任务的模型请求。
    + 飞书、邮件等 Connect 消息创建的即时任务；飞书请求通过定时任务执行器发送时与备忘录使用同一规则。
+ 只有 HTTP 成功、SSE 未标记异常且收到 `data: [DONE]` 的轮次可以写入可复用历史；请求失败、取消、响应中断、异常 SSE 或没有可提取 assistant 文本的轮次不得形成历史，避免下一次请求带入不完整上下文。
+ 用户消息保留原 OpenAI `content` JSON 值，支持文本和多模态数组；assistant 消息保存从 SSE 聚合出的文本内容。没有 `chatId`、没有当前 `user` 消息或历史库暂不可用时，保持原请求行为，不阻断上游转发。
+ `integration` 与 `proxy` 的共享 sqlite 日志清理由各自 `config/config.json.chat.clean` 控制，单位为小时；清理范围包含 `agent_message_log`、`chat_log`、`cmd_log` 与 `chat_history_message`，且不得在启动时硬编码保留天数。
+ 验收要求：
    + 同一 `chatId` 的第二轮普通会话上游请求按“系统指令、第一轮 user、第一轮 assistant、第二轮 user”顺序携带 `messages`。
    + 备忘录和飞书 Connect 即时任务发送上游请求时也携带同一会话的既有历史。
    + 当前轮在正常 SSE 完成后仅新增一对 user/assistant 历史记录；失败、取消和不完整 SSE 不新增记录。
    + 注入历史中的每一条消息均包含正确的 `created`，并且历史条数不超过 `chat.restore`。
    + `chat.clean=168` 时，Integration 与 Proxy 都按 168 小时阈值清理四张日志/历史表。
    + 历史注入不改变原始 SSE 转发、`/api/restore`、`/log_round` 与既有 chat 日志格式。

### 编写代码
+ 以 Go 在 Integration 内实现共享的历史读取、消息组装和完成轮次写入逻辑；普通转发与定时/Connect 执行器必须复用该逻辑。
+ 为普通会话、备忘录和飞书 Connect 路径补充针对历史顺序、完成写入和异常不写入的测试。

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求、边界和验收行为，不记录实现过程。
