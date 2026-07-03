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
+ 转发`/v1/chat/completions`时如果当前模型配置了：__url、 __model、__model_fast、__model_thinking、__model_multi_input、__model_multi_output则在转发请求的metadata中加入：
+ 案例：配置了__url
```
{
    ...
    "metadata": {
        ...
        "__url": aaa
    }
}
```
+ 案例：配置了__url和__model_fast
```
{
    ...
    "metadata": {
        ...
        "__url": aaa,
        "__model_fast": bbb
    }
}
```
    + 模型匹配，如使用deepseek则使用该模型对应的配置，如果没有配置或属性为空则不添加

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



