### 第一性原则
+ 仅可以新增/更新/删除cron（../..）同目录的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Cron介绍：../../REQUIREMENT.md
+ Cron手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 为定时任务明细增加类型属性（string，默认为cron）
    + cron：备忘录任务（由备忘录cron创建的周期任务或一次性任务）
    + connect：connect模块创建的任务，值为具体模块名（如飞书）
        + Connect模块：../../../connect/REQUIREMENT.md
+ 任务明细按Agent、Chat（会话ID）、时间、类型建立索引（如果已经有索引则删除）

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




