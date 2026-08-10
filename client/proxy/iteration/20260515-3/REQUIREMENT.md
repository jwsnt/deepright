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
+ 转发/v1/chat/completions前检查metadata的knowledge属性的最后更新时间与当前时间是否超过了由参数--knowledge_update_interval指定的毫秒数（默认为2小时）
    + Knowledge需求：../../../knowledge/REQUIREMENT.md
+ 如果没超过，则删除knowledge的lastUpdate属性（其他属性不变）
+ 如果超过了，则当前请求的时间记录为Knowledge最后申请更新的时间，需要保存进数据库
    + 同时对比数据库已经存在的最后申请更新的时间距离当前请求时间是否超过由参数--knowledge_update_lock指定的毫秒数（默认为30分钟），如果没超过则同样删除knowledge的lastUpdate属性（其他属性不变）
        + 如果数据库中不存在最后申请更新的时间则按超过处理
    + 如果超过了，则更新最后申请更新的时间，其他逻辑不变
+ 以上逻辑是为了防止，同时并发有多个请求触发了知识库更新，导致并发更新。
    + 服务端会通过判断是否有lastUpdate来执行是否更新

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



