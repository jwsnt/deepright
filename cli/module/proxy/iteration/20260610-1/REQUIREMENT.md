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
+ 转发/v1/chat/completions时需要带上metadata.agent.sandbox和metadata.agent.version
```
{
    "metadata": {
    ...
        "agent": {
            ... 其他属性,
            "version": string,
            "sandbox": string
        }
    }
}
+ Agent需求：../../../agent/iteration/20260610-1/REQUIREMENT.md

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



