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
+ restore 接口需要补齐右侧 CLI 子任务 `cmd` 的历史恢复数据来源，供 site 在重新打开会话或轮询续拉时重建 `cmd` 子任务展示
+ 继续沿用现有 `/api/restore` 收口，不新增新的恢复接口；返回结构仍保持当前统一 `data[]` 记录数组格式
+ `/api/restore` 在查询 `chat_log` 之外，还需要继续合并当前会话下的 CLI 事件日志，并把 `cli/get`、`cli/pub` 一并返回给前端：
    + 查询范围与消息 restore 保持一致，按 `agentId + chatId + timeline` 过滤
    + 当带有 `lastId` 时，需要支持增量续拉，避免前端重复消费已恢复记录
    + 返回给前端的 CLI 日志记录需保留 `id/agentId/chatId/content/logType/createdAt`，并用 `role=cli/get`、`role=cli/pub` 明确标识类型
+ 为了让前端能恢复 CLI 子任务 `cmd`，服务端返回的 `content` 必须保留原始日志内容，不在 restore 链路中提前裁剪成摘要：
    + `cli/get` 需要保留原始任务载荷，兼容直接 `cmd` 字段、嵌套 `message`、`messages[].content` 等既有格式
    + `cli/pub` 需要保留原始执行结果，兼容纯文本输出和 JSON 包裹的消息结构
    + 不额外发明新的 `cmd restore` 专用字段，避免打破既有日志写入和消费方兼容性
+ restore 返回的消息记录与 CLI 事件日志必须合并后统一排序：
    + 先按 `createdAt` 升序
    + `createdAt` 相同时按 `id` 升序
    + 最终保证前端按单一时间线即可把 `cli/get -> cli/pub` 配对为同一条 CLI 子任务 `cmd`
+ 若当前环境下 CLI 事件日志查询失败，不应影响原有 chat_log restore 主流程；可以按现有容错方式仅返回已拿到的消息记录，避免整个 restore 接口失败
+ 本次只补充 restore 对 CLI 子任务 `cmd` 的恢复能力，继续复用现有 CLI 日志写入链路和存储表，不新增新的日志表、消息表或额外后台任务

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
