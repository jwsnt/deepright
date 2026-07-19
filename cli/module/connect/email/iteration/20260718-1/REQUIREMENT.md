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
+ 邮件插件的 `send`、`init` 命令新增可选参数 `--to` 和 `--subject`，用于在没有原始请求上下文时发送一封新的邮件
+ `--to` 支持以逗号分隔的多个收件人；每个收件人可使用标准邮件地址或带显示名的邮件地址
+ `--message` 改为可选参数：
    + 未传 `--message`，或 `--message` 中未包含 `rawRequest` 时，必须同时传入非空的 `--to` 和 `--subject`
    + 显式传入 `--message` 时，其值必须为合法 JSON；JSON 不合法时立即报错，不发送邮件，即使已传入 `--to` 和 `--subject`
    + 此模式不生成 `In-Reply-To` 和 `References`，作为新邮件发送
    + `--message` 仅用于承载 `connect add-request` 返回的请求 JSON 及其中的 `rawRequest`，不作为邮件正文
+ `rawRequest` 存在时，邮件必须按回复邮件处理：
    + 忽略 CLI 中传入的 `--to` 和 `--subject`
    + 只有 `rawRequest` 字段完全不存在时才可进入新邮件模式；字段存在但值为 `null`、空字符串、纯空白或非字符串时，均视为无效并立即报错，不发送邮件
    + `rawRequest` 必须是合法 JSON，且 `message.messageId`、`message.from`、`message.subject` 均为非空值；缺少任一字段或字段不合法时立即报错，不发送邮件
    + 收件人只从 `rawRequest.message.from` 解析，主题只从 `rawRequest.message.subject` 解析并按既有规则生成 `Re: 原主题`
    + `In-Reply-To` 使用 `rawRequest.message.messageId`；`References` 按既有规则保留原有引用并追加父邮件 ID
+ 发件人仍只使用邮件插件配置中的 `meta.email`，本次不新增指定发件人的 CLI 参数
+ `--to` / `--subject` 的新增参数、`--message` 的可选语义、回复邮件与新邮件的校验和发信头行为均需覆盖 `send`、`init` 的 CLI 帮助、用户手册和自动化测试

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
