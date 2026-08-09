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
+ `/cli/get` 返回的任务 `subOps` 必须兼容 `echo` 布尔字段，并将其随原始任务内容写入既有 `cli/get` 日志；缺失该字段时保持兼容默认语义。
+ `subOps.echo` 仅供页面在 `/api/restore` 场景决定是否显示 CLI 子面板记录。独立 cli-get 不得因其为 `false` 而跳过入队、执行、`cli/pub` 回传、重试、日志写入、沙盒或其他任务行为。
+ 每次 `cli/pub` 必须继续携带原任务的 `chat` 与 `tid`，作为页面按 `chatId + tid` 关联恢复展示的稳定标识。

### 编写代码
+ 以Golang编写以上代码，要求：
    + 复用既有配置读取、心跳循环和队列实现，不新增外部 Go 依赖
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写
+ 最小范围更新
+ 需要同步更新 integration 中对应实现与测试
+ `TaskContent` 的 `subOps` 解析结构必须保留 `echo`；不得为展示字段引入新的执行分支。

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 并编写本迭代目录 `USER_GUIDE.md`，说明 `subOps.echo` 是恢复展示元数据，不影响独立 cli-get 的执行和 `cli/pub` 发布。

### 其他要求
+ REQUIREMENT.md 为需求文档，禁止记录实现过程。
