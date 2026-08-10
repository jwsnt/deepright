### 第一性原则
+ 仅可以新增/更新/删除`../../目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Agent元数据介绍：../../REQUIREMENT.md
+ Agent元数据手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 从Agents Array获取所有AgentId的Skills名称
+ 案例：
```
"agents": [
    {
        "workspace": string,
        "agentId": A,
        "soul": string,
        "user": string,
        "skills": [
            {
                "name": SKILL_A,
                "location": string,
                "description": string,
                "license": string,
                "compatibility": string,
                "metadata": {},
                "allowed-tools": string
            },
            {
                "name": SKILL_B,
                "location": string,
                "description": string,
                "license": string,
                "compatibility": string,
                "metadata": {},
                "allowed-tools": string
            }
        ],
    }
]
```
    + AgentId=A的Skills名称为：[SKILL_A,SKILL_B]

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 验证测试
+ 以`test-case`作为指定目录，验证代码生产内容是否符合：
    + AgentId=a的Skills名称：[__internal_F]
    + AgentId=b的Skills名称：[__internal_A，__internal_F]

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




