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
+ 将转发/v1/chat/completions的连接按AgentId和Chat（会话ID）维度进行管理，一个Agent和一个Chat在同一时刻只能有一个活跃的转发连接
+ 新建HTTP POST服务，路径为/api/cancel?agentId=xxx&chat=yyy，主动终止关闭指定Agent和指定chat（会话ID）的转发连接
+ 如果转发连接不是主动终止，那么任何非预期断开（如刷新页面）都不会关闭转发连接，会话存储要继续直到SSE完成或转发服务异常终止
    + 会话存储需求：../20260427-8/REQUIREMENT.md
+ 如果主动断开连接（Agent+Chat维度），那么在会话存储中需要记录包括时间的主动断开标记，同时不在记录会话存储

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




