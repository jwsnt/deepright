### 第一性原则
+ 仅可以新增/更新/删除browser（ ../../..）目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../../DESIGN.md
+ 本模块设计文档：../../../DESIGN.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 需求介绍
+ 本次`browser instance init`超时规则适用于macOS、Windows、WSL/WSL2和Linux；其他`browser instance`命令的既有超时行为不变。
+ 所有`browser instance init`必须从integration运行时`config/config.json`读取嵌套配置`browser.init_timeout`，单位为秒。
+ `browser.init_timeout`字段不存在时，统一使用300秒；该默认值仅适用于字段缺失。
+ `config/config.json`文件找不到、JSON解析失败、`browser.init_timeout`已配置但不是正整数时，`instance init`必须立即报错终止，不得静默回退默认值。
+ `browser instance init`只能在`browser start`成功后运行；`browser start`写入的`browser_runtime.json`是定位integration二进制及其`config/config.json`的运行时依据。
+ `browser_runtime.json`不存在或无法解析时，`instance init`必须报错提示先执行`browser start`；不得通过Browser二进制目录、当前工作目录或其他猜测路径绕过该前置条件。
+ 配置文件定位与校验必须在关闭旧CDP、旧Chrome或修改实例状态之前完成；配置异常不得影响正在运行的旧实例。
+ 超时时间是整个`instance init`的总deadline，覆盖旧实例关闭、profile准备和复制、Chrome拉起、CDP就绪检测等所有阶段，不得只限制某一个等待步骤。
+ 超时后必须终止本次新启动的Chrome、清理本次运行状态，但保留`chrome_xxx` profile；不得遗留存活但未登记的CDP，也不得丢失用户登录态。
+ profile复制等长耗时操作必须响应总deadline，不能因单个大文件复制而绕过配置的超时上限。

#### 插件日志
+ 必须在插件同目录下提供以browser.log的日志文件

### 撰写手册
+ 编写USER_GUIDE.md
+ 明确记录`browser.init_timeout`的配置路径、单位、字段缺失时的300秒默认值、`browser start`前置条件及配置非法时的失败行为。

### 同步代码
+ ../../../browser/REQUIREMENT.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 编写代码
+ WSL的用户目录复制、实例profile创建、实例重启、完成按钮展示与实例关闭应围绕上述流程实现；其他既有browser逻辑不得改变。
+ `browser start`不应含有等待页面打开、重启集成或创建页面的逻辑；等待仅用于完成用户目录复制、daemon就绪及必要的Chrome启动检测。
+ `browser stop`关闭实例时允许忽略已手动关闭Chrome导致的“进程/端口不存在”结果，但必须清理该实例的运行状态。
+ 将`browser instance init`的配置解析与总deadline收口为各平台共用逻辑；WSL启动器、macOS、Windows和Linux的实例初始化都必须使用同一个已校验的timeout。
+ 不得把`browser.init_timeout`当作顶层字符串读取；必须按嵌套JSON对象`browser.init_timeout`解析为正整数秒。
+ 当总deadline耗尽时，取消后续初始化操作，关闭本次新进程并删除本次新增状态；现有`chrome_xxx`目录一律保留。

### 验收测试
+ 不能只看端口通不通，必须校验daemon归属，避免误连旧版本
+ 后台daemon必须脱离父会话，不能出现start后几秒自灭
+ 所有后台进程功能都要补，父命令退出后仍存活的验收测试
+ 严格遵守指纹需求：../../CHECK.md
+ WSL下验证`browser start`会重建`chrome_def`，并包含`Default/Network/Cookies`及登录态所需内容；同时确认其不打开页面或有头Chrome窗口。
+ WSL下验证首次`browser instance init`从`chrome_def`创建缺失的`chrome_xxx`，先关闭旧CDP后启动新的`headless=false`Chrome，并确认CDP不复用旧进程。
+ WSL下验证已有`chrome_xxx`时，`browser instance init`仍会关闭旧实例并拉起新的有头Chrome，且保留该profile内容。
+ WSL下验证“完成”按钮仅在实例启动成功后出现；点击按钮关闭实例返回`OK`，手动关闭有头窗口后再点击也返回`OK`并完成状态清理。
+ WSL下验证`browser stop`关闭所有实例、删除`chrome_def`，且完整保留所有`chrome_xxx`目录。
+ 验证macOS、Windows、WSL/WSL2和Linux的`browser instance init`都读取integration的`config/config.json`中`browser.init_timeout`，并以秒作为单位。
+ 验证`browser.init_timeout`字段缺失时，所有平台的`instance init`总deadline为300秒；字段为正整数时按配置值生效。
+ 验证未执行或未成功执行`browser start`时，`instance init`因缺少`browser_runtime.json`立即失败，且不会从其他路径猜测配置文件。
+ 验证`config/config.json`缺失、JSON非法、`browser.init_timeout`非正整数时，`instance init`在关闭任何旧实例前失败，旧CDP、Chrome和状态保持不变。
+ 验证总deadline覆盖旧实例关闭、profile复制、Chrome启动和CDP就绪；任一阶段超时都会关闭本次新Chrome、清理本次状态、保留`chrome_xxx`，且不会留下未登记CDP。

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 同步代码：../../../../integration/REQUIREMENT.md（每次都要同步更新代码）
