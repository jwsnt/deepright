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
+ 新建HTTP GET服务，路径为/api/agent/create?agentId=xxx&name=yyt&type=zzz，在Agent目录下创建文件或文件夹
    + name需要符合操作系统的目录的命名规则，禁止包含空格
    + type用以区分文件或文件夹
    + 立即刷新Agent/Skills缓存
+ 案例：
    + 请求'/api/agent/create?agentId=a&name=b&type=1'
    ``` 为AgentId=a的目录创建一个名称为b的文件
    + 请求'/api/agent/create?agentId=a&name=c&type=0'
    ``` 为AgentId=a的目录创建一个名称为c的目录

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




