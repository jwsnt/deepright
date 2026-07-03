### 第一性原则
+ 仅可以新增/更新/删除feishu（../..）同目录及其子目录下`的文件和文件夹

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
+ 严格遵守原始报文JSON SCHEMA：../../REQUEST_SCHEMA.json）
+ 严格遵守测试必过集：../../TEST_CASE.md

### 需求介绍
+ send/init命令处理发送消息时，检查--content参数输入字符串是否为JSON Object（{}），如果是则检查JSON是否符合schema命令返回的SCHEMA要求
    + Schema命令：../20260518-1/REQUIREMENT.md
+ 如果符合schema命令的SCHEMA则按如下处理：
    + content：Markdown格式的内容，作为实际--content的参数
        + 会先扫描Markdown里的远程图片或本地图片链接：
            + 远程图片 `![alt](http/https/...)`：先下载图片到feishu_artifacts，再调用飞书图片上传接口拿image_key，并把Markdown里的图片地址替换成![alt](image_key)，再按原来的interactive card发送
            + 本地图片 `![alt](/tmp/a.png)`、`![alt](file:///tmp/a.png)`：直接调用飞书图片上传接口拿image_key，并把Markdown里的图片地址替换成![alt](image_key)，再按原来的interactive card发送
            + 本地图片案例：![磁盘使用率](/Users/xxx/Documents/agent/A/tmp/disk_usage.png)
            + 远程图片案例：![磁盘使用率](https://www.xx.com/A/tmp/disk_usage.png)
    + artifacts：将path对应的文件作为飞书附件发送（可能有多个），注意MIME类型（如果是图片则作为--image，如果是文件则作为--file）
        + send命令需求：../../../PLUGIN.md
    + 解析过程需要有日志标记content、image、file
+ 如果--content参数不是符合schema命令返回的SCHEMA的JSON，或在schema归一化、Markdown图片下载、Markdown图片上传过程中出现异常时，整个--content参数作为整体发送消息，不处理图片和文件等附件
    + 如果降级后还是发送失败，则推送：<消息异常>请登录客户端查看
+ init命令与send命令共用相同的schema归一化与Markdown图片替换逻辑
+ 发送飞书的超时为180秒

### 同步代码
+ ../../../feishu/REQUIREMENT.md
+ 所以设计/编译都需要遵守feishu的二进制和CLI收口原则

### 编写代码
+ 以Golang编写以上代码，要求：
    + 编译应用名：feishu
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 复制至Plugin：../../../../plugins/
