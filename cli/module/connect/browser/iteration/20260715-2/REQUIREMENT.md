### 第一性原则
+ 仅可以新增/更新/删除browser（ ../../..）目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../../DESIGN.md
+ 本模块设计文档：../../../DESIGN.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 需求介绍
+ 本次变更仅适用于WSL分支；非WSL分支的现有行为保持不变。
+ `browser start`必须先强制关闭系统Chrome进程，再删除并从系统Chrome User Data重新复制用户目录到`C:\\ProgramData\\deepright\\chrome_def`。
+ WSL复制系统Chrome用户目录时，必须保留`Network`目录（包括`Network/Cookies`）及全部与登录态相关的目录和文件，确保浏览器实例继承系统Chrome登录态。
+ `browser start`完成初始化后只启动后台daemon，不得重启集成、不创建页面、不打开或激活新的浏览器标签页/窗口。
+ 用户目录复制耗时较长时，WSL `browser instance init` 的等待时限必须覆盖复制及Chrome拉起过程；从 integration 的`config.json`读取`browser.init_timeout`（单位：秒，默认配置为`300`），避免超时提前终止启动器。
+ `browser instance init`必须先执行一次对应实例的`instance shutdown`尝试关闭旧CDP及Chrome进程，再以`headless=false`拉起有头Chrome。
+ `browser instance init`检查映射的`chrome_$变量`目录：目录不存在时，从`chrome_def`复制创建；目录存在时保留并复用该目录，但仍必须关闭旧实例后重新拉起有头Chrome。
+ `browser instance init`不得因探测到旧CDP、已存在profile或残留状态而直接返回成功；只有新的有头Chrome实际启动成功后才可完成初始化。
+ WSL的强制重建属于`browser instance init`内部实现，通过`DEEPRIGHT_BROWSER_WSL_FORCE_RECREATE`传递；不得新增或要求用户使用公开的`--force-recreate` CLI参数。
+ 页面中的“完成”按钮只能在`browser instance init`实际成功后展示，点击后调用对应实例的`instance shutdown`关闭有头Chrome；用户也可以先手动关闭有头Chrome窗口，再点击“完成”，此时关闭操作仍必须返回`OK`。
+ `browser stop`必须尽力关闭所有受管理的实例，然后只删除`chrome_def`；不得删除任何`chrome_xxx`目录，以便后续保留各实例的登录态和配置。
+ WSL实例关闭后的端口与状态清理由绝对路径的PowerShell执行，避免Chrome已退出后因PATH中找不到`powershell.exe`而使关闭状态残留。

#### 插件日志
+ 必须在插件同目录下提供以browser.log的日志文件

### 撰写手册
+ 编写USER_GUIDE.md

### 同步代码
+ ../../../browser/REQUIREMENT.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 编写代码
+ WSL的用户目录复制、实例profile创建、实例重启、完成按钮展示与实例关闭应围绕上述流程实现；其他既有browser逻辑不得改变。
+ `browser start`不应含有等待页面打开、重启集成或创建页面的逻辑；等待仅用于完成用户目录复制、daemon就绪及必要的Chrome启动检测。
+ `browser stop`关闭实例时允许忽略已手动关闭Chrome导致的“进程/端口不存在”结果，但必须清理该实例的运行状态。

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

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 同步代码：../../../../integration/REQUIREMENT.md（每次都要同步更新代码）
