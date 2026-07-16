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
+ `cli/get` 执行任务超时时，不改变现有输入交互，不允许新增运行时用户交互、标准输入转发或 PTY。
+ 超时前已经采集到的 stdout/stderr 必须保留，并作为任务结果回传。
+ 超时结果格式：
    + 若采集输出经 `TrimSpace` 后为空，返回：`[Warning: Command execution timed out.]`
    + 若采集输出经 `TrimSpace` 后非空，返回：`<已收集的输出内容>[Warning: Command execution timed out, the returned content may be incomplete.]`
+ stdout 与 stderr 使用现有合并采集链路；不要求区分来源，也不额外插入换行。
+ 该规则需同时适用于本地 Shell、外部 `CLI_SANDBOX`、独立沙盒服务和 WSL helper 执行路径。
+ 页面自定义 CMD 的 `/api/cmd` 也必须应用同一规则；integration 与 proxy 的本地命令执行器需要同步实现。

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
