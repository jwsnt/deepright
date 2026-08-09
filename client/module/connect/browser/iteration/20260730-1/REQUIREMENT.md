### 第一性原则
+ 仅可以新增/更新/删除browser（ ../../..）目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../../DESIGN.md
+ 本模块设计文档：../../../DESIGN.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 需求介绍
+ 本次变更仅适用于 Windows WSL / WSL2 分支；macOS、Linux 与 Windows 原生环境的 Chrome 用户目录复制、缓存过滤和锁文件清理行为不得改变。
+ WSL 下执行 `browser start` 时，不得枚举、结束或等待 Windows 主机上的任何 Chrome 进程；不得因 Browser 插件启动影响用户正在使用的系统 Chrome。
+ WSL 下执行 `browser start` 时，不得读取、解析或复制 `%LOCALAPPDATA%\\Google\\Chrome\\User Data`，也不得通过 PowerShell 查询该目录。
+ WSL 下不得创建、删除、刷新、复制或作为模板使用 `C:\\ProgramData\\deepright\\chrome_def`；`browser stop` 同样不得针对该目录执行清理。
+ WSL 下 `browser instance create` 与 `browser instance init` 为新实例分配 `C:\\ProgramData\\deepright\\chrome_<随机后缀>` 后，必须直接以空目录作为 `--user-data-dir` 启动，不得从系统 Chrome、`chrome_def` 或任何其他 profile 复制数据。
+ WSL 下不得在 `browser start`、`instance create`、`instance init`、实例重启或实例关闭过程中删除 `Singleton*`、`DevToolsActivePort`、`*.lock`、`*-journal` 或其他 profile 锁/运行态文件。
+ WSL 下已有实例 profile 的重启继续复用原目录，但不得为了重启而清理该目录中的锁文件。
+ Browser 在静态 `config/config.json` 缺少 `app-dir` 和 `app` 时，必须复用 Integration 的固定运行目录规则：macOS 为 `~/Library/Containers/cn.deepright.integration/Data/Library/Application Support/deepright`，WSL 为 `~/deepright`；不得要求重启 Integration 写回这些派生字段。

#### 插件日志
+ 必须在插件同目录下提供以browser.log的日志文件

### 撰写手册
+ 编写USER_GUIDE.md
+ 明确记录 WSL 下 `browser start` 不会操作系统 Chrome 或其用户目录，且 `instance create/init` 使用空的独立 profile、不复制登录态、不清理 profile 锁文件。

### 同步代码
+ ../../../browser/REQUIREMENT.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 编写代码
+ 移除 WSL `browser start` 中结束 Windows Chrome、复制系统 Chrome 用户目录、维护 `chrome_def` 及清理其锁文件的全部调用链和实现。
+ 移除 WSL 新建实例时从 `chrome_def` seed profile 的调用链；保留目录分配与 Chrome 启动逻辑，使新目录保持为空。
+ 移除仅为 WSL profile 删除锁/运行态文件的调用链；其他平台既有的复制后锁清理保持不变。
+ 将 Browser plugins 目录解析改为兼容 Integration 的动态 `app-dir` 规则；保留显式 `app-dir` / `app` 的旧配置兼容性。
+ `browser start`不应含有等待页面打开、重启集成或创建页面的逻辑；WSL 也不得加入任何系统 Chrome/profile 预处理。
+ `browser stop`关闭实例时允许忽略已手动关闭Chrome导致的“进程/端口不存在”结果，但必须清理该实例的运行状态，不得删除 WSL profile 或 `chrome_def`。
+ 将`browser instance init`的配置解析与总deadline收口为各平台共用逻辑；WSL启动器、macOS、Windows和Linux的实例初始化都必须使用同一个已校验的timeout。
+ 不得把`browser.init_timeout`当作顶层字符串读取；必须按嵌套JSON对象`browser.init_timeout`解析为正整数秒。
+ 当总deadline耗尽时，取消后续初始化操作，关闭本次新进程并删除本次新增状态；现有`chrome_xxx`目录一律保留。

### 验收测试
+ 不能只看端口通不通，必须校验daemon归属，避免误连旧版本
+ 后台daemon必须脱离父会话，不能出现start后几秒自灭
+ 所有后台进程功能都要补，父命令退出后仍存活的验收测试
+ 严格遵守指纹需求：../../CHECK.md
+ WSL下验证`browser start`不会调用 Windows Chrome 进程结束、系统 Chrome 用户目录查询/复制、`chrome_def` 删除/创建或锁清理；同时确认其不打开页面或有头Chrome窗口。
+ WSL下验证首次`browser instance create/init`创建缺失的`chrome_xxx` 空目录，先关闭旧CDP后启动新的`headless=false` Chrome，并确认CDP不复用旧进程。
+ WSL下验证已有`chrome_xxx`时，`browser instance init`仍会关闭旧实例并拉起新的有头Chrome，保留该profile的全部内容且不删除锁/运行态文件。
+ WSL下验证“完成”按钮仅在实例启动成功后出现；点击按钮关闭实例返回`OK`，手动关闭有头窗口后再点击也返回`OK`并完成状态清理。
+ WSL下验证`browser stop`关闭所有实例并完整保留所有`chrome_xxx`目录，且不访问或删除`chrome_def`。
+ 验证 macOS 与 WSL 静态 `config/config.json` 未包含 `app-dir` / `app` 时，`browser start` 仍能分别解析到固定运行目录下的 `plugins`。
+ 验证macOS、Windows、WSL/WSL2和Linux的`browser instance init`都读取integration的`config/config.json`中`browser.init_timeout`，并以秒作为单位。
+ 验证`browser.init_timeout`字段缺失时，所有平台的`instance init`总deadline为300秒；字段为正整数时按配置值生效。
+ 验证未执行或未成功执行`browser start`时，`instance init`因缺少`browser_runtime.json`立即失败，且不会从其他路径猜测配置文件。
+ 验证`config/config.json`缺失、JSON非法、`browser.init_timeout`非正整数时，`instance init`在关闭任何旧实例前失败，旧CDP、Chrome和状态保持不变。
+ 验证总deadline覆盖旧实例关闭、profile准备（非 WSL 的复制）、Chrome启动和CDP就绪；任一阶段超时都会关闭本次新Chrome、清理本次状态、保留`chrome_xxx`，且不会留下未登记CDP。

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 同步代码：../../../../integration/REQUIREMENT.md（每次都要同步更新代码）
