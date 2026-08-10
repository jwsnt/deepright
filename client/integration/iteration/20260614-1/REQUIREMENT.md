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
+ 增加/api/agent/export?agent_id=xxx，用于导出指定Agent目录（文件、子孙目录）
    + 文件过滤规则（一级目录）：
        + 去掉chrome开头的目录
        + 去掉data目录
        + 去掉tmp目录
    + 导出结构为.zip包
+ 增加/api/agent/import，用于导入Agent目录（输入数据为export zip或目录）
    + 导入前检查是否已经存在重名目录，如果存在需要拒绝导入，并提示先删除同名Agent
    + 需要支持export导出的zip结构，并解压后删除zip
    + 需要支持直接导入目录结构

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写