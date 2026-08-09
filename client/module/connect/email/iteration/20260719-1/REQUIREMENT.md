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
+ 邮件插件新增 `sender` 命令：
```
./email sender
```
    + 只能查询 Integration / Connect 已保存的本地通用消息快照，不得调用邮件服务器、扫描 `email.log` 或直接连接 SQLite。
    + 从 Integration 运行目录的 `config/config.json` 读取 `email.lastMessage`，该值为正整数，单位为小时。
    + 查询窗口为执行命令时刻向前回溯 `email.lastMessage` 小时，按邮件发送时间筛选。
    + `config/config.json` 不存在、JSON 无法解析、缺少 `email.lastMessage`、值不是正整数时，命令必须立即失败并输出清晰错误；不得静默使用默认值。
    + 发件人从邮件 `From` 头提取第一个有效邮箱地址，去除显示名、转为小写后存储和去重；没有有效邮箱地址的邮件不得写入快照。
    + 结果中的 `sender` 必须唯一；同一邮箱有多条邮件时，仅保留发送时间最新的一条。
    + 返回 JSON 数组，字段固定为 `sender`、`lastMessageAt`，按 `lastMessageAt` 倒序排列。
+ 邮件插件新增 `search` 命令：
```
./email search --query "关键词"
```
    + 查询时间窗口、配置读取和错误处理规则与 `sender` 命令完全一致。
    + 仅查询插件归一化后的邮件主题加正文；不得搜索附件二进制内容或附件路径。
    + 内容匹配为不区分大小写的包含匹配。
    + `--query` 可省略；省略或传入空字符串时，列出时间窗口内全部可搜索的文本消息。空白分隔的多个关键词为 AND 关系；双引号包裹的连续内容视为一个完整短语，例如：`--query '"退款申请" 已处理'`。
    + 支持可选 `--sender`，对标准化后的发件人邮箱作精确匹配；输入值先转小写。`--sender` 与 `--query` 同时提供时取 AND；仅提供 `--sender` 时，返回该邮箱窗口内全部可搜索的文本消息。
    + 支持 `--limit` 和 `--offset` 分页：`--limit` 默认 50、最大 200，`--offset` 默认 0；负数、0（仅 `limit`）或非整数必须立即报错。
    + 返回 JSON 对象，字段固定为 `total`、`limit`、`offset`、`items`；`total` 为过滤后窗口内全部文本消息数。`items` 中每项字段固定为 `messageId`、`sender`、`content`、`sentAt`，按 `sentAt` 倒序排列。
+ 邮件成功写入 `connect_request` 时，邮件插件必须将归一化后的消息快照作为通用能力输入一并提交。快照来源固定为 `email`，必须保存消息 ID、标准化发件人邮箱、主题加正文、消息类型和发送时间。
+ 邮件发送时间优先使用 RFC 5322 `Date` 头；该头缺失或无法解析时，回退为邮件插件的接收时间。
+ Integration / Connect 只能提供与来源无关的消息快照存储和查询能力：不得解析邮件报文、不得包含邮件发件人、配置字段或命令语义，也不得依赖邮件插件包。
+ 邮件插件负责把通用快照中的发送者 ID 映射为自身的 `sender` 输出，并负责读取 `email.lastMessage` 配置；Integration / Connect 只接收 `source`、消息属性、查询窗口和搜索参数。
+ 必须优化数据库查询：
    + 通用能力为时间范围和发送者聚合建立 `(source, sent_at DESC)`、`(source, sender_id, sent_at DESC)` 索引。
    + 通用能力使用 SQLite FTS5 trigram 优化文本包含检索；短关键词或 FTS 无法覆盖的情况仍必须保证包含匹配语义正确。
    + 所有查询必须使用参数绑定，不得拼接用户的搜索内容到 SQL 中。
+ `email command` 输出必须包含 `sender`、`search`；`email help` / `email --help` 以及两个子命令的 `--help` 必须说明参数、返回结果、错误条件和可直接复制的完整案例。

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
