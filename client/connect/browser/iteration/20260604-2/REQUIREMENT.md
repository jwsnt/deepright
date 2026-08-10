### 第一性原则
+ 仅可以新增/更新/删除browser（ ../../..）目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../../DESIGN.md
+ 本模块设计文档：../../../DESIGN.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 需求介绍
+ 修改命令start，在原有逻辑前（代码最开始部分）检查当前系统是否安装了playwright（包括driver），如果已安装则继续原逻辑，如果未安装则安装当前系统的playwright（包括driver）
+ 需要包括Mac和Windows WSL，驱动放在插件目录的playwright/driver下，比如browser位于/plugins/browser那么驱动位于/plugins/playwright/driver
+ 无论安装成功、失败还是超时，都需要继续原有流程，不能因为下载而导致终止、崩溃或异常
+ 插件browser的playwright驱动均先从playwright/driver查找
+ 仅增加安全可靠的下载逻辑，不要修改原启动逻辑
+ 构建脚本（包括整个工程的构建脚本 ../../../../build.sh）需要去掉原本playwright/driver的编译和打包

### 最终效果
+ 在start命令安装playwright（包括driver）成功后，后续通过browser代理使用playwright不需要下载驱动

#### 插件日志
+ 必须在插件同目录下提供以browser.log的日志文件

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
