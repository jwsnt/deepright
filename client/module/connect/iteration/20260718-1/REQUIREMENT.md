### 第一性原则
+ 仅可以新增/更新/删除connect（../..）同目录的文件和文件夹
+ 如非授权，禁止修改其他插件目录文件和文件夹
### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
    + browser、email、feishu等

### 迭代要求
+ Connect介绍：../../REQUIREMENT.md
+ Connect手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 重要提示
+ 设计前需要仔细阅读Connect的设计，需要兼容方案
    + Connect介绍：../../REQUIREMENT.md

### 需求介绍
+ 插件`--help`帮助精简：
    + 所有Connect插件二进制的`--help`输出中，不得介绍、说明或展示`start`、`stop`、`init`命令
    + 删除内容包括但不限于：Usage命令行、命令说明、Notes说明、案例，以及`--help`中内嵌的命令列表示例
    + Browser插件除顶层`browser --help`外，`browser daemon --help`、`browser instance --help`等子命令帮助也必须遵守上述规则
    + 飞书、邮件等插件若`help`子命令与`--help`共用同一帮助输出，两个入口都必须保持一致，不得仅隐藏其中一个入口
    + `integration plugins --help`属于Integration管理CLI，不属于插件自身`--help`，本次不修改
+ 命令能力保持不变：
    + `start`、`stop`、`init`的命令解析、路由、实现、内部调用、命令参数和执行结果均不得修改
    + 插件的`command`子命令实际返回的JSON能力列表不得修改；仅`--help`中的静态文案不得再列出这三个命令
    + 不得影响Integration通过插件`start`、`stop`执行生命周期管理的既有流程
+ 验收：
```bash
./plugins/feishu --help
./plugins/email --help
./plugins/browser --help
./plugins/browser daemon --help
./plugins/browser instance --help
```
    + 上述帮助输出均不得出现对`start`、`stop`、`init`命令的介绍、说明或案例
    + 直接执行三个命令时，仍必须沿用修改前的命令行为；自动化测试应覆盖帮助输出精简和命令路由未变化两个方面

### 编写代码
+ 以Golang编写以上代码，要求：
    + 所有Connect模块复用启动时初始化的全局数据库连接，禁止每次请求单独打开和关闭数据库文件
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
