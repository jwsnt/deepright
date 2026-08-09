### 第一性原则
+ 仅可以新增/更新/删除integration（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Integration介绍：../../REQUIREMENT.md
+ Integration手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ add-request命令新可选参数--schema（对应任务明细的response_schema），为Json String
``` 案例
./integration connect add-request --key 三方Key --externalId 外部ID --content 请求内容 --artifacts 以,分隔的字符串附件路径 --original 原始请求 --status 状态 --created 创建时间的时间戳 --schema JSON String表示的Response JSON Schema
```
    + add-request命令需求：../../../connect/iteration/20260518-1/REQUIREMENT.md
    + 技术规范的指导下收口：
        + Proxy需求：../../../proxy/iteration/20260518-1/REQUIREMENT.md

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




