### 第一性原则
+ 仅可以新增、更新或删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../proxy`、`../../../site/index.html` 与使用手册。

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
+ 保持既有 token 配置字段、聊天、`/cli/get`、飞书、邮件、备忘录任务及 SSE 协议兼容；不新增外部 Go 依赖。

### 需求介绍
+ `/api/restore` 必须继续按既有时间线返回 `cli/get` 与 `cli/pub` 原始记录，且每个记录保留可解析的 `chatId`、任务 `tid` 与 `subOps.echo` 信息；不得因 `echo=false` 删除、阻断或伪造任务执行和回传记录。
+ `cli/get` 任务协议中的 `subOps.echo` 兼容缺失值；只有显式布尔值 `false` 表示恢复态页面不展示该任务。该字段只服务前端恢复展示，不改变 Integration 的任务调度、执行、发布、日志写入或 `/cli/pub` 转发。
+ `cli/pub` 的原始结果必须保留 `chat` 与 `tid`，使页面能按 `chatId + tid` 将结果与同一任务关联；服务端不得将结果改写为“最近任务”或跨会话合并。

### 编写代码
+ 最小范围更新，不新增外部依赖。
+ Integration 内置 cli-get 任务解析必须保留 `subOps.echo`，同时保持现有执行语义不变。
+ 为任务解析与 `/api/restore` 记录补充测试，验证 `echo=false`、任务 `tid`、`cli/pub` 的会话字段均可原样用于前端恢复。

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`，说明恢复接口保留 `echo`、`chatId` 与 `tid` 仅供前端展示关联，不影响执行或发布。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求、边界和验收行为，不记录实现过程。
