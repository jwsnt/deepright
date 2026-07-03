### 第一性原则
+ 仅可以新增/更新/删除email（../..）同目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../../DESIGN.md
+ 本模块设计文档：../../../DESIGN.md

### 迭代要求
+ Connect介绍：../../../REQUIREMENT.md
+ Connect手册：../../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 重要提示
+ 设计前需要仔细阅读Connect的设计，需要兼容方案
    + Connect介绍：../../../REQUIREMENT.md
+ 严格遵守测试必过集：../../TEST_CASE.md

### 需求介绍
+ 插件mail启动start命令后，每个扫描周期都必须重新建立一条新的POP3连接，完成认证、UIDL拉取和邮件读取后立即主动断开，不能把上一轮扫描留下来的POP3会话复用到下一轮
+ 若当前轮扫描中的POP3连接被服务端主动断开并返回EOF，需要把这类可恢复断链当成"需要重连"，并在当前轮内自动重连后继续完成这一轮扫描
+ 邮件插件必须对POP3域名解析异常、TLS握手超时和单线路由抖动具备自动容错能力，不能因为网络侧偶发问题导致新邮件长时间无法收取
    + 邮件插件连接email_pop3时，若默认DNS解析结果不可用或握手超时，必须自动切换到备用解析结果继续建连
    + POP3建连、TLS 握手、首包读取都必须有明确超时时间，超时后要记录结构化错误日志，不能静默卡死
    + 单次扫描失败不能阻塞后续轮询；插件必须按周期自动重试，并在网络恢复后自动继续收信，无需人工重启
    + 只有在邮件成功读取并进入后续处理后，才允许推进email.state.json时间线，避免因为网络异常漏掉新邮件。
    + 邮件插件扫描邮件时，不能扫描全部邮箱历史邮件；只允许扫描最近 `email_pop3_interval * 2` 时间范围内的未处理邮件。
    + 若 `email_pop3_interval` 缺失、为空或非法，则扫描窗口按默认 `300 * 2 = 600` 秒处理。
    + 若 `email.state.json` 已存在，则仍必须继续沿用已持久化的时间线和去重状态，不能因为重启而回退为全量扫描。

### 同步代码
+ ../../../email/REQUIREMENT.md
+ 所以设计/编译都需要遵守email的二进制和CLI收口原则

### 编写代码
+ 以Golang编写以上代码，要求：
    + 编译应用名：email
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 验证测试
```
./integration connect meta-create --name email --meta '{"email":从系统环境变量获取$EMAIL,"email_pop3":从系统环境变量获取$EMAIL_POP3,"email_smtp":从系统环境变量获取$EMAIL_SMTP3,"email_password":从系统环境变量获取$EMAIL_PASSWORD,"email_whitelist":""}'
```
    + 其中 `--meta` 的字段集合必须与 `./feishu param` 返回值一致
```
./email send --message 原消息报文（json string） --content 消息文本内容 --image 以逗号分隔的图片附件 --file 以逗号分隔的文件附件
```

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 复制至Plugin：../../../../plugins/
