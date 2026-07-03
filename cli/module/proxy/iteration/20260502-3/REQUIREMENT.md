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
+ 每分钟执行一次定时任务获取任务明细时，如果任务明细指定了CHAT_ID则直接用于请求的会话ID，如果没指定或为空则使用原逻辑（用@连接的任务元数据ID@任务明细ID）
    + 定时任务需求：../20260429-1/REQUIREMENT.md
    + 任务明细需求：../../../cron/iteration/20260502-1/REQUIREMENT.md
+ 为创建任务（/api/cron/create）也增加CHAT_ID
    + 创建任务需求：../20260427-2/REQUIREMENT.md

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




