### 第一性原则
+ 仅可以新增/更新/删除browser（ ../../..）目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../../DESIGN.md
+ 本模块设计文档：../../../DESIGN.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 需求介绍
+ 仅在 Windows WSL / WSL2 版本的 browser 插件内增加“等待桌面小组件”能力，影响范围必须控制在 `cli/module/connect/browser` 目录及其子目录，不允许为了这个能力去修改 integration 或其他模块的既有启动链路
+ 仅在 `browser instance init --agentId ... --chatId ...` 这条会先关闭旧浏览器、再重新拉起有头 Chrome/CDP 的链路里生效；`create`、`get`、`list`、`shutdown`、`restart`、playwright 代理命令及其他非 init 命令都不能出现这个等待桌面小组件
+ 当 Windows WSL / WSL2 下执行 `browser instance init` 时，需要在真正关闭旧浏览器前，于 Windows 宿主机桌面左下角显示一个等待中的桌面小组件，用来明确提示“浏览器正在重新启动/连接中”
+ 小组件必须是轻量级、临时的、不可交互的等待提示，不允许抢占焦点、不允许进入任务栏主流程、不允许影响用户后续操作；即使显示失败，也不能阻断原有 `init` 流程
+ 小组件的显示时机必须早于旧浏览器关闭动作；只要 `init` 进入关闭旧实例并重新创建新实例的流程，小组件就应持续存在，直到本次 `init` 成功或失败后再收口
+ 当新的浏览器实例/CDP 启动成功并达到当前 `browser instance init` 的 ready 判定后，小组件必须自动消失，不允许残留在桌面
+ 当新的浏览器实例/CDP 启动失败、超时、无法探测到 ready、launcher 返回异常、PowerShell 执行异常或其他任何 init 失败场景发生时，小组件也必须被关闭，不允许因为异常而永远挂在桌面
+ 小组件相关能力只在检测到 Windows WSL / WSL2 时启用；macOS、Linux 原生、Windows 原生非 WSL 环境的 browser 插件行为必须保持原样，不允许出现新 UI、副作用或兼容性回归
+ 小组件的视觉样式可以后续实现时细化，但本次需求至少要求：固定出现在桌面左下角、表达“等待浏览器初始化”的语义、默认展示期间不打断用户
+ 如果宿主机缺少 PowerShell、桌面能力不可用、权限不足、脚本拉起失败或关闭失败，browser 插件只能记录日志并继续原有 `init` 主流程，不能因为小组件失败而让浏览器初始化失败
+ 小组件的显示、成功关闭、失败关闭、显示异常、关闭异常都必须具备明确的日志记录，且日志统一写入 browser 插件同目录下的 `browser.log`

#### 插件日志
+ 必须在插件同目录下提供以browser.log的日志文件
+ 本次需求新增的小组件生命周期日志必须全部写入 `browser.log`，至少包括：
+ `widget_show_begin`：开始尝试显示等待桌面小组件
+ `widget_show_success`：等待桌面小组件显示成功
+ `widget_show_error`：等待桌面小组件显示失败，需包含错误原因，但不能阻断 init
+ `widget_close_success`：在浏览器 init 成功后关闭小组件成功
+ `widget_close_error`：在成功或失败收口阶段关闭小组件失败，需包含错误原因
+ `widget_init_failed_cleanup`：浏览器 init 失败后已进入小组件清理逻辑
+ 浏览器 `init` 成功和失败本身也都需要在 `browser.log` 中保留明确记录，并且要能和本次小组件日志串联，方便按一次 `init` 流程排查问题

### 撰写手册
+ 编写USER_GUIDE.md

### 同步代码
+ ../../../browser/REQUIREMENT.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 编写代码
+ 以Golang编写以上代码，要求：
    + 需要将browser后台daemon启动为真正独立于前台命令生命周期的进程，不能继承父进程的临时stdout/stderr管道，避免后续命令执行时因输出链路失效导致daemon断开
    + browser在插件配置和命令链路中正确识别并校验cookie_path，确保fetch、store和start都按该 Cookie 文件工作，出错时终止启动并写明browser.log
    + 代码简洁，包体积越小越好
    + 能用开源包的就用开源包
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 验收测试
+ 不能只看端口通不通，必须校验daemon归属，避免误连旧版本
+ 后台daemon必须脱离父会话，不能出现start后几秒自灭
+ 所有后台进程功能都要补，父命令退出后仍存活的验收测试
+ 严格遵守指纹需求：../../CHECK.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 同步代码：../../../../integration/REQUIREMENT.md（每次都要同步更新代码）
