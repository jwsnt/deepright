### 第一性原则
+ 仅可以新增、更新或删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../site/index.html` 与 Site 使用手册。

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
+ 不新增外部 Go 依赖，不改变既有请求正文、响应语义、SSE、任务队列或 `/cli/pub` 协议。

### 需求介绍
+ Integration 新增同源只读 `GET /api/bubblewrap/check`，供 Site 在打开会话沙盒浮层前判断当前运行环境是否具备 Bubblewrap。接口只接受 `GET`；其它方法返回 `405` 并声明允许的方法。
+ 仅当 Integration 运行于 Linux（包括 WSL Linux）时依赖 Bubblewrap。非 Linux 运行环境必须返回 `status: 0`、`required: false`、`available: true`，不得读取 Bubblewrap 配置或查找可执行文件。
+ Linux/WSL 环境每次检查必须从主应用资源目录的 `config/config.json` 读取必填 `bubblewrap` 对象：`check` 为正整数小时，`install` 为非空字符串。不得读取 Agent 工作目录中的配置，不得提供默认值；配置缺失或无效时返回可直接展示的错误，且不得将沙盒视为可用。
+ 配置有效时，仅通过 Integration 受控命令搜索环境查找 `bwrap`。找到后才返回 `required: true`、`available: true`，并将成功时间仅缓存于当前 Integration 进程内存，缓存时长为 `bubblewrap.check`；进程重启后缓存自然失效。失败结果、查找异常和配置错误均不得写入成功缓存。
+ Linux/WSL 中缺少 `bwrap` 时，接口返回 `status: 0`、`required: true`、`available: false` 与经校验的 `bubblewrap.install` 原文。接口不得执行安装命令、修改配置、写入 Agent 文件、修改会话沙盒状态或代替实际沙盒 helper 的运行时校验。
+ 启动阶段可异步预热同一成功缓存，但不得阻塞服务启动；既有 `/api/sandbox*`、SSE、任务队列、`/cli/pub` 和实际沙盒执行协议保持不变。

### 编写代码
+ 以 Golang 和现有页面代码完成最小范围更新。
+ 复用现有 FFmpeg 依赖检查的配置校验、成功缓存和响应写入模式；不新增外部依赖。

### 撰写手册
+ 更新 Integration、Site 及本迭代目录的 `USER_GUIDE.md`，说明 Bubblewrap 配置、预检接口、Linux/WSL 范围、仅成功结果的进程内缓存、重启失效，以及缺少依赖时由页面发送安装请求的行为。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求、边界和验收行为，不记录实现过程。
