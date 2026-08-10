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
+ 新建HTTP GET服务，路径为/api/edit?agentId=xxx&path=yyy，在指定Agent的绝对路径下写入文件内容
+ path的value只支持相对于Agent工作目录（workspace）的相对路径
    + Agent相关需求：../20260419_7/REQUIREMENT.md
+ path的value需要支持有空格的路径场景
+ path的value不区分大小写

### 访问限制
+ 如果path指定绝对路径、目录、二进制文件类型（图片、多媒体格式等）则抛出异常

+ 案例：
    + 请求'/api/edit?agentId=a&&path=/a/b/c，写入AgentId为a的工作目录下/a/b目录的文件c
    ``` 写入成功
    {
        "agentId": xxx
        "path": 文件相对路径,
        "status": 0
    }
    ```
    ``` 写入失败
    {
        "agentId": xxx
        "path": 文件相对路径,
        "content": 错误提示
        "status": 1
    }
    + 请求'/api/edit?agentId=a&&path=/a/b/d，其中d为目录
    ```
    {
        "agentId": xxx
        "path": 文件相对路径,
        "content": 错误提示,
        "status": 1
    }
    ```
    + 请求'/api/edit?agentId=a&&path=/a/b/e，其中e为图片
    ```
    {
        "agentId": xxx
        "path": 文件相对路径,
        "content": 错误提示,
        "status": 1
    }
    ```

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 验证测试
+ 以`test-case`作为指定目录，验证代码生产内容是否符合：
    + agentId=a&&path=`b/c/user.md`，写入内容：HELLO USER

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




