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
+ 每次上报心跳（cli/get）后记录最后一次成功（无网络异常）、失败（有网络异常）、执行任务，来判断与代理服务的互通性
    + 心跳需求（cli/get）：../../../cli-get/REQUIREMENT.md
    + 如果有待处理任务，即使执行失败也不归回心跳失败
+ 新建HTTP GET服务，路径为/api/heartbeat，获取最后一次心跳的时间和状态

+ 案例：
    + 请求'/api/heartbeat'
    ``` 心跳成功，无任务
    {
        "lastTimestamp": 时间戳
        "status": 0
    }
    ```
    ``` 心跳失败，有故障
    {
        "lastTimestamp": 时间戳
        "status": 1
    }
    ```
    ``` 心跳成功，有任务
    {
        "lastTimestamp": 时间戳
        "status": 2
    }
    ```

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




