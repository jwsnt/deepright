### 第一性原则
+ 仅可以新增、更新或删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../proxy`、`../../../site/index.html` 与使用手册。

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
+ 保持既有 token 配置字段、聊天、`/cli/get`、飞书、邮件、备忘录任务及 SSE 协议兼容；不新增外部 Go 依赖。

### 需求介绍
+ 新增只读 `GET /api/session_recovery_candidates`，用于列出可恢复的普通页面会话；该接口不新增数据表、不补写日志，也不改变既有 `/api/restore` 的路径、参数、响应或恢复语义。
+ 接口使用 `page` 参数从 1 开始分页，每页固定返回最多 10 个会话，并返回当前页、页大小和是否还有下一页；非法页码必须返回现有风格的 JSON 错误。
+ 前端可通过重复的 `exclude` 查询参数传入当前左侧会话列表中的 `chatId`；服务端必须在数据库查询阶段排除这些会话，以确保每页返回的都是当前列表之外的候选会话。
+ 候选范围仅限 `chat_type = page_session` 的普通会话。每个 `chatId` 只返回最后一次正常完成的回答：回答记录必须为 `role = A`、`response_type = normal`，且包含 SSE 完成标记 `data: [DONE]`；异常、取消、未完成流以及定时备忘录会话均不得出现在列表中。
+ 每个候选返回 `agentId`、`chatId`、`lastPrompt`、`completedAt`：`lastPrompt` 必须是该成功回答前最近一条 `Q` 请求中的最后一条用户消息文本。无有效 `chatId`、提问或关联提问记录的候选不得返回。
+ 该查询必须直接分页读取 SQLite，禁止读出全部 `chat_log` 后在内存中去重、排序或分页。排序固定为最后成功完成时间倒序、同时间按日志 ID 倒序。
+ 为该访问路径增加可由 SQLite 使用的联合索引：
    + 以 `chat_type, role, response_type, chat_id, created_at DESC, id DESC` 支持按普通成功会话找每个 `chatId` 的最新完成记录。
    + 以 `chat_type, role, response_type, created_at DESC, id DESC, chat_id` 支持候选结果按完成时间与日志 ID 倒序分页。
    + 以 `chat_id, role, created_at DESC, id DESC` 支持回溯关联的最近 `Q` 提问。
+ 本需求必须同时覆盖 Integration 和仍提供相同聊天恢复能力的 Proxy 服务入口，保持两者接口、字段、分页、排除和索引语义一致。

### 编写代码
+ 最小范围更新，不新增外部依赖。
+ 复用现有 `chat_log`、SSE 完成标记识别、SQLite 错误响应与 `/api/restore` 的聊天恢复数据；不得把候选查询混入 `/api/restore` 或改变其兼容协议。
+ 为候选查询补充覆盖分页、排除当前会话、仅取正常完成会话、最后提问解析和索引初始化的自动化测试。

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求、边界和验收行为，不记录实现过程。
