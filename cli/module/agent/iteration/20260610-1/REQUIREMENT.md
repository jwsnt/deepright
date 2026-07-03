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
+ 为每个Agent添加version属性，从--agent-dir下的`config.json`获取`version`，如果json文件不存在则使用空字符串
```
{
    "description": string,
    "thinking": boolean,
    "swarm": boolean,
    "version": string
}
```
    + config.json中属性version仅在启动时获取一次并缓存
+ 为每个Agent添加sandbox属性，从存储沙盒的数据表实时获取Agent+ChatId维度的沙盒模式（不要缓存），不存在则使用""
    + Proxy需求：../../../proxy/iteration/20260608-1/REQUIREMENT.md
    + 沙盒模式需求：../../../cli-get/iteration/20260609-1/REQUIREMENT.md
```
{
    "description": string,
    "thinking": boolean,
    "swarm": boolean,
    "version": string,
    "sandbox": string
}
```

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




