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
+ send命令处理发送消息时，检查--content参数输入字符串是否为Json Object（{}），如果是检查JSON是否是否符合schema命令返回的SCHEMA要求
    + Schema命令：../20260518-1/REQUIREMENT.md
+ 如果符合schema命令的SCHEMA则按如下处理：
    + content：纯文本格式的内容，作为实际--content的参数
        + 会先扫描纯文本格式的远程图片或本地图片链接，如果遇到http/https/...下载图片到email_artifacts、本地绝对路径（/tmp/a.png、file:///tmp/a.png）直接作为附件
            + 本地图片案例：/Users/xxx/Documents/agent/A/tmp/disk_usage.png
            + 远程图片案例：https://www.xx.com/A/tmp/disk_usage.png
            + 以上作为没有替换成文件名的兜底策略
    + artifacts：将path对应的文件作为邮件附件发送（可能有多个），注意MIME类型（如果是图片则作为--image，如果是文件则作为--file）
        + send命令需求：../../../PLUGIN.md
    + 解析过程需要有日志标记content、image、file
+ 如果--content参数为不符合schema命令返回的SCHEMA、解析JSON时失败、下载失败、上传附件等异常时，整个--content参数作为整体发送消息，不处理图片和文件等附件
    + 如果降级后还失败，则推送：<消息异常>请登录客户端查看
+ init命令与send命令共用相同的schema归一化与图片替换逻辑
+ 发送邮件的超时为180秒

### 同步代码
+ ../../../email/REQUIREMENT.md
+ 所以设计/编译都需要遵守email的二进制和CLI收口原则

### 编写代码
+ 以Golang编写以上代码，要求：
    + 编译应用名：email
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 复制至Plugin：../../../../plugins/
