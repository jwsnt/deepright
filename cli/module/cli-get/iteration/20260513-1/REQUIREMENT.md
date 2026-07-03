### 第一性原则
+ 仅可以新增/更新/删除`../../目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Cli-Get介绍：../../REQUIREMENT.md
+ Cli-Get手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ cli/get和cli/pub需要记录日志，包括
    + AgentID、ChatID（会话ID）、内容、类型、时间
        + 类型：0=/v1/chat/completions请求、1=/v1/chat/completions的SSE响应、2=cli/get、3=cli/pub
    + 与Proxy需求采用同表结构：../../../proxy/iteration/20260513-1/REQUIREMENT.md
+ 日志均采用异步，不要影响渲染和执行速度
+ 如果原逻辑有日志记录，则合并到该需求
+ 如果原逻辑有查询需求，则适配到该需求（注意查询类型）

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
    + 使用文件名为data的sqlite存储，并使用连接池
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




