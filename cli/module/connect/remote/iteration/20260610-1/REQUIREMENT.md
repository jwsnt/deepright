### 第一性原则
+ 仅可以新增/更新/删除connect（../..）同目录的文件和文件夹
+ 如非授权，禁止修改其他插件目录文件和文件夹
### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../../DESIGN.md
    + browser、email、feishu、remote等

### 迭代要求
+ Remote介绍：../../REQUIREMENT.md
+ Remote手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 关联需求
+ SSH插件：../日期/REQUIREMENT.md

### 同步代码
+ ../../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 重要提示
+ 设计前需要仔细阅读Remote的设计，需要兼容方案
    + Remote介绍：../../REQUIREMENT.md

### 需求介绍
+ 修改命令param，固定返回：
```
[{"exec_timeout":"选填。SSH执行超时。","scp_timeout":"选填。SCP执行超时。"}]
```

### 编写代码
+ 以Golang编写以上代码，要求：
    + 所有Remote模块复用启动时初始化的全局数据库连接，禁止每次请求单独打开和关闭数据库文件
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



