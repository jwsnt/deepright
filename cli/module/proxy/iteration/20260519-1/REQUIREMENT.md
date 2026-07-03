### 第一性原则
+ 仅可以新增/更新/删除proxy（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Proxy介绍：../../REQUIREMENT.md
+ Proxy手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 插件日志统一处理，通过 `/api/plugins/log?name=插件key`读取
+ 所有插件日志文件路径固定为：`release/plugins/插件名.log`
    + 不允许根据当前工作目录、上级目录或其他候选目录推断日志路径
+ 当日志文件不存在时，返回明确错误：`log file not found: release/plugins/插件名.log`

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 验收标准
+ `feishu`日志读取固定为`release/plugins/feishu.log`
+ `email`日志读取固定为`release/plugins/email.log`
+ 其他插件同理，统一按`release/plugins/插件名.log` 规则处理

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



