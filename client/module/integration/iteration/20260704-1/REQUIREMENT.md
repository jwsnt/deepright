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
+ 新增虚拟文件系统中技能目录的会话级禁用能力，并由 `integration` 统一收口服务与接口
+ 技能目录定义为：工作区中直属包含 `SKILL.md` 的目录；禁用后，该目录及其全部子孙技能在当前 `chatId` 下都视为不可用
+ 禁用状态必须按会话维度持久化保存；刷新页面、重载同一会话后仍然生效，不影响其他会话
+ `GET /api/skills?agentId=xxx&chatId=yyy` 需要按当前会话禁用状态过滤返回结果；未传 `chatId` 时保持现有行为不变
+ 所有读取指定 `chatId` Agent metadata 的 integration 链路都需要复用同一过滤逻辑，避免被禁用的技能继续通过 `metadata.agents[].skills` 暴露给上游
+ `GET /api/files?path=xxx&chatId=yyy` 在目录项返回中新增以下字段，用于标识虚拟文件系统中的技能目录状态：
    + `hasSkill`：当前目录直属是否存在 `SKILL.md`
    + `skillDisabled`：当前目录是否处于禁用态（包含自身禁用和继承禁用）
    + `skillDisabledSelf`：当前目录是否被当前目录自身禁用
    + `skillDisabledInherited`：当前目录是否因父级技能目录被禁用而继承禁用
+ 新增 `POST /api/skill_state` 接口，用于切换技能目录禁用状态：
    + 请求体为 `chatId`、`path`、`disabled`
    + `path` 必须是存在的目录，且直属包含 `SKILL.md`
    + 返回 `status`、`chatId`、`path`、`disabled`、`disabledSelf`、`disabledInherited`、`disabledPaths`
+ 技能目录禁用列表需要统一做绝对路径规范化、去重和父子路径压缩：
    + 如果父级目录已禁用，则子级目录不再重复写入禁用列表
    + 如果恢复父级目录，则原本仅继承禁用的子级目录需要自动恢复为可再次单独禁用
+ 持久化存储需要复用共享 sqlite，并新增独立的技能目录状态表保存 `chatId -> disabled skill dir paths`
+ 需要补充 integration 侧测试，至少覆盖：
    + `/api/skills` 在传入和不传入 `chatId` 时的返回差异
    + `/api/files` 返回技能目录禁用字段
    + `/api/skill_state` 的禁用、恢复、父子目录压缩与继承状态

### 技术实现
+ 新增独立的会话级技能目录状态子模块，提供：
    + 表结构初始化
    + 路径标准化
    + 父子目录命中判断
    + 禁用列表合并/裁剪
+ `integration` 的 `/api/skills`、`/api/files` 与 chat metadata 读取链路统一通过同一套会话技能状态能力取数
+ `integration` 在无共享运行态实例时，也需要能直接打开当前 data sqlite 读取或写入技能目录状态，保证接口可独立工作
+ 对外协议保持最小新增：
    + 仅在 `/api/files` 的目录项中追加状态字段
    + 仅新增一个 `/api/skill_state` POST 接口
    + 现有 `/api/skills` 在不传 `chatId` 时继续兼容旧调用

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
