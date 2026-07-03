### 第一性原则
+ 仅可以新增/更新/删除当前需求文档（REQUIREMENT.md）同目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../DESIGN.md
+ 本模块设计文档：../DESIGN.md

### 需求介绍
+ 二进制收口, 最终交付给用户的主程序必须是`remote`一个二进制文件，日志必须是与remote同目录下的remote.log
+ `remote`必须同时具备以下能力：
    + 创建ssh连接
    + 通过ssh连接，执行远程系统命令
+ 最终使用者只应感知`remote`，不应要求用户额外理解、安装、启动或调用其他模块

### name命令
``` 响应报文
{"key":"remote","name":"远程"}
```

### param命令
``` 响应报文, 非案例, 实际仅返回cookie_path
["exec_timeout","scp_timeout"]
```

### start命令
+ 启动后台进程，使用remote list获取并关闭所有ssh连接，相当于重启
+ 启动后台进程后需要在应用启动目录下记录进程remote.pid文件
    + 无论remote从哪里触发，最终都需要把PID固定写到相对于integration/plugins目录下的remote.pid
    + 插件目录需求：../../integration/REQUIREMENT.md

### stop命令
+ 使用remote list获取并关闭所有ssh连接，相当于清理
+ 关闭start命令启动的后台进程, 确保在关闭服务时也能正确关闭remote插件拉起的所有ssh子进程，不留下残留后台进程

### 新建实例
+ 新增命令create, 以start命令启动的主进程来启动ssh连接子进程
``` 假定应用程序为remote
./remote create --agentId xxx --chatId yyy --remote ubuntu@1.2.3.4 --password xxx --port 10086
```
    + 参数：
        + agentId：AgentId
        + chatId：ChatId（当前会话)
        + remote：远程主机用户名和地址
        + password：远程主机密码
        + port：远程主机端口，选填，默认22
    + 首先归一化--agentId和--chatId参数为小写字母，防止因为大小写引起的匹配错误
    + 检查对应agentId和chatId（当前会话）是否存在有效的ssh连接，如果有则直接返回成功
    + 如果没有，调用ssh命令连接远程主机，并缓存连接后返回成功
    + 使用后台线程启动
``` 返回响应
{"agentId":"xxx","chatId":"yyy","port":10086,"pid":9999,"ssh":"ubuntu@1.2.3.4"}
```
+ 新建完连接实例需要创建或更新应用程序同目录下的remote.json, 该文件负责记录当前创建所有连接实例的端口号、pid和SSH地址
``` remote.json
[{"agentId":"xxx","chatId":"yyy","port":10086,"pid":9999,"ssh":"ubuntu@1.2.3.4"},...]
```
+ 如果Agent和Chat（会话ID）已经存在SSH连接且pid依旧存在则不用重复创建, 否则创建并更新remote.json

### 关闭实例
+ 新增命令shutdown, 关闭remote的ssh连接子进程
``` 假定应用程序为remote
./remote shutdown --agentId xxx --chatId yyy
```
    + 从remote.json查找对应pid并关闭
``` 返回响应
OK或失败异常
```

### 实例列表
+ 新增命令list, 获取已启动remote的ssh连接列表
``` 假定应用程序为remote
./remote list
```
``` 返回响应
[{"agentId":"xxx","chatId":"yyy","port":10086,"pid":9999,"ssh":"ubuntu@1.2.3.4"},...]
```

### 指定实例
+ 新增命令get, 获取指定remote的ssh连接
    + 首先归一化--agentId和--chatId参数为小写字母，防止因为大小写引起的匹配错误
``` 假定应用程序为remote
./remote get --agentId xxx --chatId yyy
```
``` 返回响应
{"agentId":"xxx","chatId":"yyy","port":10086,"pid":9999,"ssh":"ubuntu@1.2.3.4"}
```

### 执行命令
+ 新增exec命令，执行远程系统命令
``` 假定编译后唯一应用程序为remote
./remote exec --session xxx@yyy "ls -l /a"
```
    + 参数：
        + session：对应--agentId和--chatId的组合，使用@连接
        ```
        "agentId":"xxx","chatId":"yyy"则session为xxx@yyy
        ```
    + 获取对应session（AgentId和ChatId）的ssh连接，并执行远程命令后返回结果

### 帮助命令
+ 增加help为所有remote支持、代理和集成的CLI命令提供使用手册和案例（User Guide/Usage）
+ remote help需要为代理的ssh（ssh）的所有命令提供完整的使用说明

### 构建交付
+ 插件加载固定使用启动目录下的plugins目录，禁止任何候选回退
+ 编写最新的构建脚本build.sh

### 同步代码
+ 所以设计/编译都需要遵守remote的二进制和CLI收口原则

### 编写代码
+ 以Golang编写以上代码，要求：
    + 需要将remote后台daemon启动为真正独立于前台命令生命周期的进程，不能继承父进程的临时stdout/stderr管道，避免后续命令执行时因输出链路失效导致daemon断开
    + 确保remote ssh在shell使用2>& 合并输出时，仍能稳定启动本地daemon并完成页面跳转，不再因为本地POST/command启动竞态返回EOF
    + 唯一应用程序remote需要将以下模块编译在内，而不需要额外的外部依赖
        + instance模块：instance/REQUIREMENT.md
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
+ 同步代码：../../integration/REQUIREMENT.md（每次都要同步更新代码）
