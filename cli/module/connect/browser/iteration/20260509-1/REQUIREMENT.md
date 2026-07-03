### 第一性原则
+ 仅可以新增/更新/删除browser（ ../../..）目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../../DESIGN.md
+ 本模块设计文档：../../../DESIGN.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 需求介绍
+ 为Browser实现Plugin规范
    + Plugin规范：../../../PLUGIN.md

### name命令
``` 响应报文
{"key":"browser","name":"浏览器"}
```

### param命令
``` 响应报文, 非案例, 实际仅返回cookie_path
["cookie_path"]
```

### start命令
+ 使用browser代理的browser_instance list获取并关闭所有CDP服务，相当于重启
+ 启动后台进程，代理instance超时关闭功能：../../instance/iteration/20260508-1/REQUIREMENT.md
+ 启动后台进程后需要在应用启动目录下记录进程browser.pid文件
    + 无论browser从哪里触发，最终都需要把PID固定写到相对于integration/plugins目录下的browser.pid
    + 插件目录需求：../../../../integration/REQUIREMENT.md
+ 端口冲突：需要保证browser start只把当前release/plugins/目录下托管的daemon识别为已启动，不能把别的目录遗留的18333（Playwright daemon 的本地 HTTP 控制端口）监听误判成当前插件已启动
    + 如果端口冲突但不属于当前runtime的后台残留进程需要处理掉，然后在当前release/plugins/目录下重新拉起属于这份browser的daemon，并写出当前目录的 browser.pid

### stop命令
+ 使用browser代理的browser_instance list获取并关闭所有CDP服务，相当于清理
+ 关闭start命令启动的后台进程, 确保在关闭服务时也能正确关闭browser插件拉起的所有obscura和monitor子进程，不留下残留后台进程

#### 插件帮助
+ 必须实现help来提供完成的插件使用手册
``` 假设插件可执行程序为a
./a help
```

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
