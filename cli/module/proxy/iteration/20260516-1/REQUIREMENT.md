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
+ 检查在/v1/chat/completions的metadata参数是否包含：knowledge_commit
```
{
    ...,
    "metadata": {
        ...
        "knowledge_commit": true
    }
}
```
+ 包含knowledge_commit的请求必须包含知识库最后更新时间（lastUpdate）而不用检查最后申请更新的时间
    + 知识库最后申请更新的时间（用于防止并发更新的需求）：../20260515-3/REQUIREMENT.md
    + 如果不包含knowledge_commit，还是需要遵守最后申请更新的时间都检查逻辑
+ 如果存在则在SSE响应完全结束后更新
    + 知识库最后更新时间（对应/knowledge_lastUpdate）：../20260511-2/REQUIREMENT.md
    + 知识库需求：../../../knowledge/REQUIREMENT.md

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



