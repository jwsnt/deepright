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
+ 新增/api/sandbox=枚举值?agentId=x&chatId=y，为指定AgentID+ChatID（当前会话）开启或关闭沙盒模式
    + 关闭则删除对应Agent+Chat的数据
+ 指定AgentID和ChatID是否开启了沙盒需要记录进数据库
+ 新增/api/sandbox_status?agentId=x&chatId=y，获取指定AgentID+ChatID的沙盒模式
+ 修改/api/cmd也需要参与沙盒执行判断
+ 沙盒模式需求：../../../cli-get/iteration/20260609-1/REQUIREMENT.md
+ 不需要兼容老逻辑，删除兼容代码，以最新需求编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



