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
+ 将`cli/get`响应中需要执行命令及命令响应以当前AgentId+Chat（会话ID）的维度保存在sqlite中
    + cli/get需求：../../../cli-get/REQUIREMENT.md
+ 以追加内容的形式进行更新，存储数据需要能区分是什么时候收到请求，什么时候执行完成
+ 属性Cmd不为空表示需要执行命令，同时执行结果即为命令响应
+ 属性AgentId和Chat在cli/get为响应属性

### 编写代码
+ 以Golang编写以上代码，要求：
    + sqlite使用连接池，避免每次都新建连接：../20260427-8/REQUIREMENT.md
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




