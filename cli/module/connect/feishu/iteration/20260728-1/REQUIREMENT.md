### 第一性原则
+ 仅可以新增/更新/删除feishu（../..）同目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../../DESIGN.md
+ 本模块设计文档：../../../DESIGN.md

### 迭代要求
+ Connect介绍：../../../REQUIREMENT.md
+ Connect手册：../../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 重要提示
+ 设计前需要仔细阅读Connect的设计
    + Connect介绍：../../../REQUIREMENT.md
+ 严格遵守原始报文JSON SCHEMA：../../REQUEST_SCHEMA.json）
+ 严格遵守测试必过集：../../TEST_CASE.md

### 需求介绍
+ 修复 Connect 完成任务回推飞书时，在飞书已经接受消息、但本地进程超时、崩溃或状态回写失败后，同一 `task_detail` 被后续扫描重复发送的问题。
+ 本迭代的优先级是严格的“不得重复发送相同结果”：对一次飞书结果回推，允许进入待人工确认状态，但不得在发送结果未知时自动以新的幂等标识再次发送。
+ `task_detail.id` 是一次具体执行结果的稳定主键；不得使用 `task_meta.id` 作为回推幂等边界。周期任务、重跑任务和聚合任务可共享一条 meta，但必须拥有不同的 detail。
+ `task_detail` 增加以下持久化字段：
    + `reply_state`：只能为 `pending`、`sending`、`sent`、`unknown` 或空字符串。非 Connect 任务保留空字符串。
    + `reply_uuid`：当前 detail 的固定飞书回推 UUID。创建后不可更换。
    + `reply_started_at`：进入 `sending` 的 Unix 秒时间戳，用于诊断，不得用它自动重发。
+ 新产生的非默认任务在 LLM 执行完成、结果非空后必须进入 `pending`；默认 cron 任务不参与第三方回推。
+ 回推前必须以单条带条件的数据库更新原子领取发送权：仅 `reply_state=pending` 的 detail 可以变更为 `sending`。并发扫描、多个 Integration/Proxy 进程或重复 tick 中，至多一个进程可以实际调用插件 `send`。
+ 固定 UUID 必须由 `task_detail.id`、目标插件 key 和发送内容类别确定性生成，并以 RFC 4122 UUID 文本格式持久化。文本、图片和文件属于不同的飞书 API 消息时，必须使用稳定但彼此不同的类别后缀；同一类别的重试必须复用原 UUID。
+ 插件回调 flags 必须把固定 UUID 透传给 `feishu send`；飞书插件必须将其传入飞书 API 的 `uuid` 字段。不得在每次 CLI 调用时重新生成 UUID。
+ 飞书插件返回成功后，系统必须把 Connect 原始请求从“已启动”更新为“已回复”，并把 detail 标记为 `sent`、写入 `replied_at`。状态更新失败也必须把 detail 保留为 `unknown`，不得回到 `pending`。
+ 插件发送失败、调用超时、进程在飞书调用后退出，或任一步的结果无法确定时，detail 必须原子地变为 `unknown`。`unknown` 不参与自动回推扫描，不得自动重试；日志必须包含 detail ID、目标 request ID、固定 UUID 和失败原因，便于人工确认。
+ 数据库升级时，所有历史 `started=3`、非默认任务、`replied_at=''` 且尚无 `reply_state` 的 detail 必须迁移为 `unknown`。升级不得把这些可能已经送达的历史结果重新发送。
+ 完成回推查询只允许选择 `started=3`、`reply_state=pending`、结果非空、`replied_at=''` 且目标 Connect request 仍为“已启动”的任务。
+ 飞书 API 对相同 UUID 的幂等范围和有效期必须在用户指南中说明。即使远端 UUID 去重不可用或已过期，本地仍必须遵循 `unknown` 不自动重发的规则，以满足不重复发送的硬约束。
+ 必须覆盖以下测试：
    + 同一 detail 连续扫描两次，仅调用一次插件 `send`。
    + 两个并发 dispatcher 扫描同一 detail，仅一个成功领取并调用 `send`。
    + send 成功但 Connect 状态更新失败时，detail 进入 `unknown`，后续扫描不再发送。
    + send 返回错误或超时时，detail 进入 `unknown`，后续扫描不再发送。
    + 同一 detail 的重复发送尝试（仅在受控测试中）携带相同 UUID；不同 detail 携带不同 UUID。
    + 旧库升级后，已完成但未标记回复的 detail 迁为 `unknown`，不会触发插件 `send`。

### 同步代码
+ ../../../feishu/REQUIREMENT.md
+ 所以设计/编译都需要遵守feishu的二进制和CLI收口原则

### 编写代码
+ 以Golang编写以上代码，要求：
    + 编译应用名：feishu
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 复制至Plugin：../../../../plugins/
