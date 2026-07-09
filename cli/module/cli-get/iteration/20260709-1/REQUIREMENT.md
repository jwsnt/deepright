### 第一性原则
+ 仅可以新增/更新/删除`../../目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Cli-Get介绍：../../REQUIREMENT.md
+ Cli-Get手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 调整`cli-get`内部沙盒状态命中维度：从`agentId + chatId`改为仅按`chatId`命中
+ `chatId`必须强校验为非空；为空时不允许命中沙盒状态，也不允许写入沙盒状态
+ `agentId`不再参与沙盒状态定位，仅用于日志和执行观测
+ `cli-get`在构建任务结果前判断是否启用沙盒时，只允许使用`chatId`选择沙盒模式
+ `cli-get`执行沙盒命令时保留`agentId`和`chatId`日志，日志需要明确体现当前命中的沙盒模式
+ 不考虑兼容旧数据，可直接清空/重建原有`cli_sandbox_state`相关存储
+ 需要同步更新`cli-get`内部sqlite表结构和对应读写逻辑，避免继续依赖`agentId`作为主命中条件
    + `cli_sandbox_state`需要改为以`chatId`作为唯一命中维度
    + 不允许继续使用`agentId + chatId`联合主键或联合查询作为运行时命中条件
    + 旧表结构、旧索引、旧测试用例可以直接按新语义重建
+ 需要补充文本日志：记录`agentId`、`chatId`以及沙盒模式从什么状态变更到了什么状态
    + 无记录视为`off`
    + 删除记录视为切换到`off`
    + 日志仅要求文本可观测，不需要提供单独查询接口或额外落库表
+ 需要同步更新 integration / proxy 依赖的共享沙盒状态语义与测试
    + `cli-get`自身测试需要覆盖仅按`chatId`命中的新逻辑
    + 需要覆盖`chatId`为空时不命中沙盒的场景
    + 需要覆盖不同`agentId`但相同`chatId`命中同一沙盒模式的场景
    + 需要覆盖关闭沙盒后删除`chatId`记录的场景

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
    + 使用文件名为data的sqlite存储，并使用连接池
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写
+ 最小范围更新
+ 需要为新增参数补充命令行解析、默认值和帮助文案：
    + `--queue`
    + `--retry_interval`
    + `--retry_times`
+ 需要为本地任务队列、发布队列、`ddl` 丢弃、发布重试分别补充测试
+ 需要同步更新 integration 中对应实现与测试

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
