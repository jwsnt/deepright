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
+ 增加构建CLI（命令行）工具来创建任务元数据的功能，并编写详细的--help
    + 功能和Help可以从Cron模块复制：../../../cron/iteration/20260502-3/REQUIREMENT.md
    + CLI到原始命令需要是proxy自己而不是cron
+ 创建时需要检查数据，指定Agent必须存在，指定模型是否在Proxy已注册（填写Token），及其他参数必须符合格式和要求
+ 需要HTTP服务和CLI命令兼容，不同命令既可以启动HTTP又可以执行CLI，模型Token由调用时从Sqlite动态获取
    + 非Cron模块必填参数但又需要使用的从runtime.json获取：../20260503-2/REQUIREMENT.md
    + 模型与密钥：../20260503-2/REQUIREMENT.md
+ 案例：
```
../.. cron create --content "每15分钟检查一次上游接口健康" --model "OpenAI" --thinking true --rawTime "2026-05-03 10:00" --cycle 4 --chatId "chat-001"
```

### 编写代码
+ 以Golang编写以上代码，要求：
    + proxy cron create与proxy cron create-cron在未启动HTTP服务时也必须先完成共享sqlite初始化并复用与HTTP一致的校验逻辑
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




