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
+ 新增/api/swarm_agent，获取当前启动了蜂群的Agent名称（router_disable=false）
+ 启动了蜂群是指在设置中的蜂群中启动了蜂群开关（上报Agent属性中的router_disable=false）
    + Agent需求：../../../agent/iteration/20260524-1/REQUIREMENT.md
    + Agent名单中不能包含当前Agent

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



