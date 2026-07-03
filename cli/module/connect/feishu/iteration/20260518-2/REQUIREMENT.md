### 第一性原则
+ 仅可以新增/更新/删除feishu（../..）同目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../../DESIGN.md
+ 本模块设计文档：../../../DESIGN.md

### 迭代要求
+ Connect介绍：../../../REQUIREMENT.md
+ Connect手册：../../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 重要提示
+ 设计前需要仔细阅读Connect的设计，需要兼容方案
    + Connect介绍：../../../REQUIREMENT.md
+ 严格遵守原始报文JSON SCHEMA：../../REQUEST_SCHEMA.json）
+ 严格遵守测试必过集：../../TEST_CASE.md

### 需求介绍
+ 飞书插件调用add-request命令时需要调用自身的schema命令获取response_schema, 并通过--schema参数传递
``` 案例
./integration connect add-request --key 三方Key --externalId 外部ID --content 请求内容 --artifacts 以,分隔的字符串附件路径 --original 原始请求 --status 状态 --created 创建时间的时间戳 --schema response_schema内容
```
    + --schema参数传递值需要与schema命令返回值完全一致（代码收口共享一份schema）
        + Schema命令需求：../20260518-1/REQUIREMENT.md
        + 插件规范： ../../../PLUGIN.md

### 同步代码
+ ../../../feishu/REQUIREMENT.md
+ 所以设计/编译都需要遵守feishu的二进制和CLI收口原则

### 编写代码
+ 以Golang编写以上代码，要求：
    + 编译应用名：feishu
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 复制至Plugin：../../../../plugins/

