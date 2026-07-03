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
+ 为任务元数据和任务明细都增加router_disable字段（boolean）
    + 由任务元数据周期性创建的任务明细继承任务元数据的router_disable
    + 一次性任务明细在创建时可选的指定router_disable
+ 任务元数据和任务明细的router_disable字段不可为空，默认为true（关闭）
+ 定时器将任务元数据拆分为任务明细时需要继承router_disable
+ 任务元数据和任务明细原始需求：../../../cron/REQUIREMENT.md

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




