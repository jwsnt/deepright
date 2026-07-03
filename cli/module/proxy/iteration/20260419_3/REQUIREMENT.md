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
+ 新建HTTP GET服务，路径为/api/folder?agentId=xxx&&dir=yyy，调用本地命令行工具打开指定Agent的Workspace绝对路径
    + Agent相关需求：../../../agent/iteration/20260419_1/REQUIREMENT.md
    + 本地命令行工具需要区分当前系统
+ 案例：
``` 当前Agent
"agents": [
    {
        "workspace": /a/b/c,
        "agentId": A,
        "soul": string,
        "user": string,
        "skills": [
            技能元数据列表
        ],
    },
    {
        "workspace": /d/e/f,
        "agentId": B,
        "soul": string,
        "user": string,
        "skills": [
            技能元数据列表
        ],
    }
]
```
+ 请求'/api/folder?agentId=A'，打开/a/b/c
+ 请求'/api/folder?agentId=B'，打开/d/e/f
+ 请求'/api/folder?agentId=B&&dir=g'，打开/d/e/f/g

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 验证测试
+ 以`test-case`作为指定目录，验证代码生产内容是否符合：
    + agentId=a，路径为：绝对路径/test-case/a
    + agentId=b，路径为：绝对路径/test-case/b

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




