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
+ 新建HTTP GET服务，路径为/api/workspace?agentId=xxx，获取指定Agent的workspace绝对路径
    + Agent相关需求：../../../agent/iteration/20260419_2/REQUIREMENT.md
+ 案例：
``` 当前Agent
"agents": [
    {
        "workspace": /a/b/c,
        "agentId": A,
        "soul": string,
        "user": string,
        "skills": [
            {
                "name": A,
                ...
            },
            {
                "name": B,
                ...
            }
        ],
    }
]
```
+ 请求'/api/workspace?agentId=A'，返回响应`/a/b/c`

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 验证测试
+ 以`test-case`作为指定目录，验证代码生产内容是否符合：
    + agentId=a，workspace=xxx/a
    + agentId=b，workspace=xxx/b

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




