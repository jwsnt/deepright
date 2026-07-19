### 第一性原则
+ 仅可以新增/更新/删除integration（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Integration介绍：../../REQUIREMENT.md
+ Integration手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ Integration 在 macOS 上以前台或后台方式启动服务时，必须创建并持有一个独立的 `caffeinate` 子进程，避免系统因空闲关闭显示器或进入睡眠而中断正在执行的任务。
+ 子进程必须使用 `caffeinate -d -i -m -s`，并持续运行到 Integration 结束；不采用固定时长租约、定时续租或周期性重启方案。
+ `caffeinate` 仅在 macOS 上启动。Linux、WSL、Windows 和其他平台不得尝试执行该命令，且原有启动行为保持不变。
+ Integration 的关闭路径（正常退出、`SIGINT`、`SIGTERM`、`integration stop`、重启前的停止及本机 `/api/shutdown`）必须在服务资源释放前终止由当前进程启动的 `caffeinate` 子进程，避免留下残留防睡眠进程；重复关闭必须安全。
+ 若 macOS 中无法找到或启动 `caffeinate`，Integration 记录清晰日志后继续运行，不得使服务启动失败。
+ 不修改用户的屏幕保护、自动锁屏或其他系统偏好；不支持合上笔记本盖子后继续运行。
+ 覆盖测试：macOS 启动时使用预期参数创建子进程；非 macOS 不创建；关闭时终止已创建的子进程；启动失败不阻断服务；重复关闭安全。

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写
+ 最小范围更新

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
