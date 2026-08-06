### 第一性原则
+ 仅可以新增/更新/删除当前需求文档（REQUIREMENT.md）同目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../DESIGN.md
+ 本模块设计文档：../DESIGN.md

### 需求介绍
+ 二进制收口, 最终交付给用户的主程序必须是`browser`一个二进制文件（不包含插件obscura目录下文件），日志必须是与browser同目录下的browser.log
+ `browser`必须同时具备以下能力：
    + 作为`playwright`的操控浏览器能力：playwright/REQUIREMENT.md
    + 作为`instance`的创建CDP服务能力：instance/REQUIREMENT.md
    + 作为`cookie`的获取Cookie能力：cookie_runtime.go
+ 最终使用者只应感知`browser`，不应要求用户额外理解、安装、启动或调用其他模块
+ `playwright/driver`需要放在browser同目录下
+ `obscura/release`需要放在browser同目录下

### 执行案例
+ 假定编译后唯一应用程序为browser
``` 创建CDP服务
./browser create --agentId agent-a --chatId ctrip-home
```
``` GOTO携程
./browser --session agent-a@ctrip-home goto https://www.ctrip.com
``` 获取内容
./browser --session agent-a@ctrip-home eval 'document.body ? document.body.innerText.slice(0, 1000) : ""'
```

### 帮助命令
+ 增加help为所有browser支持、代理和集成的CLI命令提供使用手册和案例（User Guide/Usage）
+ browser help需要为代理的browser_playwright（playwright-cli）的所有命令提供完整的使用说明
+ browser start和stop命令仅由插件容器使用，不要出现在--help中

### 构建交付
+ 插件加载固定使用启动目录下的plugins目录，禁止任何候选回退
+ 编写最新的构建脚本build.sh

### 插件元信息
> 新增自 iteration/20260509-1/REQUIREMENT.md
+ `name`命令固定返回：
```JSON
{"key":"browser","name":"浏览器"}
```
+ `param`命令固定返回：
```JSON
["cookie_path"]
```

### 生命周期命令
> 新增自 iteration/20260509-1/REQUIREMENT.md
+ `start`命令启动前需要先清理受管CDP服务并写入`browser.pid`
+ `stop`命令关闭时需要同时清理browser插件拉起的所有obscura和monitor子进程，不留下残留后台进程

### WSL Chat 共享 Profile
+ 在 Windows WSL / WSL2 下，同一 `chatId` 的所有 Agent 共享一个受管 Chrome 实例和 `C:\\ProgramData\\deepright\\profiles\\chats\\<chatId>` Profile；不同 Chat 必须隔离。
+ Chat Profile 首次仅创建空目录，不复制系统 Chrome `User Data` 或 `chrome_def`；`stop` 与 `instance shutdown` 不得立即删除该目录。
+ WSL Profile 锁文件保持原样，不执行锁清理。
+ WSL 过期 Profile 由 `browser.clear` / `browser.scan` 后台任务按目录最后修改时间清理，扫描根目录仅限 `C:\\ProgramData\\deepright\\profiles\\chats`。
+ Browser 忽略 `app-dir`、`app` 和 `agent-dir` 路径覆盖，macOS、WSL 与原生 Linux/Windows 必须按各自固定运行时目录规则定位应用、插件与 Agent 目录。

### 代理Playwright兼容
> 新增自 iteration/20260509-2/REQUIREMENT.md
+ 为Browser代理的Playwright所有功能增加执行日志，尤其要单独记录当前域名下实际注入的Cookie
> 新增自 iteration/20260509-3/REQUIREMENT.md
+ 为Browser代理的Playwright所有功能增加超时重试控制：单次超时先报错，不立即杀daemon，连续失败达到`--browser_retry`后再回收
+ `/command`请求从固定短重试改为在`--browser-timeout`剩余窗口内持续重试
> 新增自 iteration/20260509-4/REQUIREMENT.md
+ `eval`同时兼容以下两种写法：
```
browser --session xxx eval 'document.title'
browser --session xxx eval --code 'document.title'
```
> 新增自 iteration/20260509-5/REQUIREMENT.md
+ `screenshot`同时兼容`--filename`和`--path`两种参数写法

### 同步代码
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 编写代码
+ 以Golang编写以上代码，要求：
    + 需要将browser后台daemon启动为真正独立于前台命令生命周期的进程，不能继承父进程的临时stdout/stderr管道，避免后续命令执行时因输出链路失效导致daemon断开
    + 指定驱动加载目录应用启动同目录的driver目录下，Install只在driver缺失时才调用，如果driver已存在，就直接playwright.Run()，不再重复打印下载日志
    + 确保browser goto在shell使用2>& 合并输出时，仍能稳定启动本地daemon并完成页面跳转，不再因为本地POST /command启动竞态返回EOF
    + 唯一应用程序browser需要将以下模块编译在内，而不需要额外的外部依赖
        + Cookie模块：cookie_runtime.go
        + instance模块：instance/REQUIREMENT.md
        + playwright模块：playwright/REQUIREMENT.md
    + 相同名称的命令行参数共享
    + 代码简洁，包体积越小越好
    + 能用开源包的就用开源包

### 验收测试
+ 不能只看端口通不通，必须校验daemon归属，避免误连旧版本
+ 后台daemon必须脱离父会话，不能出现start后几秒自灭
+ 所有后台进程功能都要补，父命令退出后仍存活的验收测试
+ 集成验证案例（必须通过）：CHECK.md

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 相关迭代：iteration/日期/REQUIREMENT.md
+ 同步代码：../../integration/REQUIREMENT.md（每次都要同步更新代码）
