### 第一性原则
+ 仅可以新增/更新/删除proxy（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Proxy介绍：../../REQUIREMENT.md
+ Proxy手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 新建HTTP POST服务，路径为/api/kill，终止指定CMD（系统命令）
    + 仅请求IP为127.0.0.1或localhost时可执行
    + 入参必须包含AgentID，且Agent必须未删除
    + CLI Get需求：../../../cli-get/iteration/20260501-1/REQUIREMENT.md
+ 执行Kill后需要记录日志到sqlite，包括Agent、会话（ChatID）、指令、执行开始时间和结束时间
    + 按Agent、会话（ChatID）、执行开始时间建立索引

### 验证案例
+ 正在执行
```
find /Users/abc/DEV -name ".codex_write_check" -not -path "/.git/" 2>/dev/null
```
+ 终止命令（以下仅为案例，具体需要兼容当前操作系统）
```
bashpgrep -f "find /Users/abc/DEV -name \.codex_write_check" | xargs kill
```

### 编写代码
+ 以Golang编写以上代码，要求：
    + 与Cron模块共享使用文件名为data的sqlite存储，并使用连接池，避免每次都新建连接
        + Cron模块需求：../../../cron/REQUIREMENT.md
    + CMD的实现方式需要与CLI保持一致：
        + CLI需求：../../../cli-get/REQUIREMENT.md
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




