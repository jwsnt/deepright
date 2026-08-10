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
+ 增加构建CLI（命令行）工具来删除任务元数据和任务明细的功能，并编写详细的--help
    + 功能和Help可以参考：../20260503-3/REQUIREMENT.md
    + CLI到原始命令需要是proxy自己而不是cron
+ 需要HTTP服务和CLI命令兼容，不同命令既可以启动HTTP又可以执行CLI，模型Token由调用时从Sqlite动态获取
+ 任务元数据需要支持的查询条件：AgentId、Chat、模型、开始执行的时间（范围查询）、执行周期
    + 如果没有指定则表示该维度需要全部匹配
+ 任务明细需要支持的查询条件：元数据Id、AgentId、Chat、模型、执行周期、执行时间点（范围查询）
    + 如果没有指定则表示该维度需要全部匹配
    + 如果没有指定时间点，任务明细数据仅查询当前时间之后的数据
+ 案例：
``` 删除任务元数据
../.. cron delete-meta --id meta_1
```
``` 删除任务元数据所有任务明细
../.. cron delete-detail --metaId meta_1
```
``` 删除任务元数据所有指定明细
../.. cron delete-detail --detailId detail_1
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




