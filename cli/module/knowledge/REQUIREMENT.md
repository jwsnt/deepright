### 第一性原则
+ 仅可以新增/更新/删除当前需求文档（REQUIREMENT.md）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../DESIGN.md
+ 本模块设计文档：DESIGN.md

### 同步代码
+ ../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 应用启动时（通常为integration）检查启动的二进制程序（通常为integration）所在目录是否存在knowledge目录，如果不存在则创建
+ 在数据库记录Knowledge以时间戳为单位的最后更新时间（毫秒），初始值为0
> 新增自 ./iteration/20260510-1/REQUIREMENT.md
+ 新增`update-time`命令，用于更新数据库中记录的Knowledge最后更新时间（毫秒时间戳）
``` 假定应用程序为knowledge, 实际启动时为integration代理
knowledge update-time 时间戳
```

### 编写代码
+ 以Golang编写以上代码，要求：
    + 与Proxy模块共享文件名为data的sqlite存储，使用连接池
    + 代码简洁，包体积越小越好
    + 能用开源包的就用开源包
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 相关迭代：iteration/日期/REQUIREMENT.md
+ 同步代码：../integration/REQUIREMENT.md
> 合并截止：./iteration/20260510-1/REQUIREMENT.md，下次合并从此之后的新迭代开始
