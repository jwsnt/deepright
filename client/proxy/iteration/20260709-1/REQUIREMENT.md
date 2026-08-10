### 第一性原则
+ 仅可以新增/更新/删除proxy（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Proxy介绍：../../REQUIREMENT.md
+ Proxy手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 需求介绍
+ 调整proxy内部会话沙盒状态维度：从`agentId + chatId`改为仅按`chatId`存储和命中
    + 包括HTTP接口、共享sqlite读写、运行时命令执行判断、转发元数据中的沙盒状态读取
+ `/api/sandbox=枚举值`写接口仍要求传入`agentId`和`chatId`
    + 其中`chatId`用于沙盒状态定位
    + `agentId`仅用于日志，不参与沙盒状态命中
    + 写接口需要先按`chatId`读取旧状态，再执行写入或删除，并输出`from -> to`文本日志
+ `/api/sandbox_status`查询接口改为仅按`chatId`查询当前会话沙盒状态
    + 不需要兼容或回退到`agentId + chatId`
    + `chatId`查不到时，不需要继续尝试`chatId + agentId`
    + `agentId`对查询接口不再是必需参数
    + 即使请求中携带了`agentId`，服务端也必须忽略该字段的定位作用
+ `chatId`必须强校验为非空
    + 写接口和读接口都需要对空`chatId`直接报错
    + 不能再把空`chatId`当作默认会话或兜底key
+ 关闭沙盒时继续删除当前`chatId`对应的沙盒记录
+ proxy内部通过共享sqlite读取/写入`cli_sandbox_state`时，不允许再以`agentId`作为主定位条件
    + 共享状态结构需要与`cli-get`保持一致
    + 旧数据不需要迁移，允许直接按新结构重建
+ `/api/cmd`等运行时沙盒执行判断需要仅按`chatId`命中沙盒模式
+ 运行时执行日志仍需要保留`agentId`和`chatId`，用于说明当前命中了哪种沙盒模式
+ 需要补充文本日志：记录`agentId`、`chatId`以及沙盒模式从什么状态变更到了什么状态
    + 无记录视为`off`
    + 删除记录视为切换到`off`
    + 仅要求写入常规文本日志，不需要额外查询接口
+ 不需要围绕响应里的`agentId`调整服务端协议；本次需求中该字段不参与沙盒状态逻辑
    + 沙盒逻辑中可以忽略响应里的`agentId`
    + 但本次需求不要求删除或重构该响应字段
+ 沙盒模式需求：../../../cli-get/iteration/20260709-1/REQUIREMENT.md
+ 需要同步更新转发给上游的`metadata.agent.sandbox`与`metadata.agents[].sandbox`
    + 两者都需要按当前`chatId`读取共享沙盒状态
    + 其语义从“看起来像agent级”调整为“当前chat级”
+ 需要保持最新跨系统沙盒方案的执行语义不回退
    + macOS继续走现有 `CLI_SANDBOX.app`
    + Windows/WSL继续走 `bubblewrap` helper
    + 区分系统的 helper 选择、目录选择和参数透传逻辑都需要与 `cli-get` / WSL 沙盒方案保持一致
    + 严格隔离 macOS 现有实现路径，不允许因为本次`chatId`维度调整破坏既有行为
+ `filepick`与`filepick_net`模式继续保留目录白名单能力
    + 写接口若携带`dir`，仍需要按当前`chatId`写入/覆盖共享状态中的`allowed_dir`
    + 未显式传入`dir`时，仍按当前系统走对应的目录选择流程
    + `net`与`off`不允许复用目录白名单参数语义
+ 需要补充测试
    + `/api/sandbox_status`仅依赖`chatId`查询的测试
    + `/api/sandbox=...`写入后不同`agentId`查询同一`chatId`返回同一状态的测试
    + `/api/cmd`仅按`chatId`命中沙盒的测试
    + `metadata.agent.sandbox`与`metadata.agents[].sandbox`按`chatId`读取的测试
    + `filepick`/`filepick_net`在`dir`模式下按`chatId`持久化`allowed_dir`的测试
    + 不同系统继续选择既有沙盒 helper、且 macOS 路径不回退的测试

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
