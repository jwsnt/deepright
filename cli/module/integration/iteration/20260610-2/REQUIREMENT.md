### 第一性原则
+ 仅可以新增/更新/删除integration（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Integration介绍：../../REQUIREMENT.md
+ Integration手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 技术规范的指导下收口：
    + 修改/api/plugins/meta中每个插件的param的参数结构为[{"key":"val"},{"key":"val"},{"key":"val"},...]
    + 插件规范：../../../connect/PLUGIN.md
        + 浏览器插件格式：../../../connect/browser/iteration/20260610-1/REQUIREMENT.md
        + 邮件插件格式：../../../connect/email/iteration/20260610-1/REQUIREMENT.md
        + 飞书插件格式：../../../connect/feishu/iteration/20260610-1/REQUIREMENT.md
        + SSH插件格式：../../../connect/remote/iteration/20260610-1/REQUIREMENT.md
    + Proxy需求：../../../proxy/iteration/20260610-2/REQUIREMENT.md

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写