### 第一性原则
+ 仅可以新增/更新/删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../connect/skillstate/`、`../../../site/index.html`、`../../../build.sh`、`../../../build/install.ps1`、`../../../build/USER_GUIDE.md`、`../../../build/USER_GUIDE.txt` 与 `../../../config/app/API.md`、`../../../config/app/CANVAS.md`、`../../../config/app/DESIGN.md`。

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
+ 不新增外部依赖，不改变既有 `/api/files`、`/api/runtime_config` 或普通消息发送协议。

### 需求介绍

所有会话中的技能均改为默认关闭，只有用户对当前会话明确启用后才可参与 `@ 技能` 选择和 Agent 请求上下文。

- 技能状态继续以 `chatId` 隔离并持久化到共享 SQLite；首次读取某个会话的技能目录或技能元数据时，所有直属包含 `SKILL.md` 的目录均应自动登记为关闭状态。
- 状态存储必须区分“尚未发现的技能目录”和“用户已明确启用的技能目录”：新发现目录默认关闭，用户启用后的记录在刷新、重启和后续扫描中不能被重新覆盖为关闭。
- 现有会话也适用新默认值：已有的关闭记录保持关闭；此前没有记录的已存在技能目录在下一次读取时补齐为关闭。
- `GET /api/files?path=...&chatId=...` 继续返回 `hasSkill`、`skillDisabled`、`skillDisabledSelf` 和 `skillDisabledInherited`，但首次返回技能目录时应已反映默认关闭状态。
- `POST /api/skill_state` 的请求和响应字段保持兼容；`disabled: false` 表示用户显式启用当前目录，`disabled: true` 表示关闭当前目录。父子目录的继承关闭规则保持不变。
- 带 `chatId` 的 `GET /api/skills`、`@ 技能` 缓存、聊天元数据和 CLI 相关 Agent 元数据必须使用同一份状态过滤，默认不暴露任何尚未启用的技能；不传 `chatId` 的兼容读取链路保持现有完整返回。
- 新增、删除或重新扫描技能目录后，新发现的目录默认关闭；不覆盖当前会话已显式启用或关闭的目录选择。

### 编写代码
+ 最小范围更新，不新增外部依赖。

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`

### 其他要求
+ `REQUIREMENT.md` 仅描述需求，不记录实现过程。
