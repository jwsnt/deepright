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
+ 从Agents Array获取所有AgentId的列表
+ 从Agents Array获取指定AgentId元数据
+ 案例：
```
"agents": [
    {
        "workspace": string,
        "agentId": A,
        "soul": string,
        "user": string,
        "skills": [
            技能元数据列表
        ],
    },
    {
        "workspace": string,
        "agentId": B,
        "soul": string,
        "user": string,
        "skills": [
            技能元数据列表
        ],
    }
]
```
    + 以上AgentId列表为：[A,B]
    + AgentId=A的元数据为：
    ```
    {
        "workspace": string,
        "agentId": A,
        "soul": string,
        "user": string,
        "skills": [
            技能元数据列表
        ],
    }
    ```

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 验证测试
+ 以`test-case`作为指定目录，验证代码生产内容是否符合：
    + AgentId列表为：[a,b]
    + AgentId=a的元数据：
    ```
    {
        "workspace": 一个绝对路径,
        "agentId": a,
        "soul": HELLO SOUL,
        "user": HELLO USER,
        "skills": [
            1个技能__internal_F
        ],
    }
    ```
    + AgentId=b的元数据：
    ```
    {
        "workspace": 一个绝对路径,
        "agentId": b,
        "soul": 没有,
        "user": 没有,
        "skills": [
            2个技能__internal_A和__internal_F
        ],
    }
    ```

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




