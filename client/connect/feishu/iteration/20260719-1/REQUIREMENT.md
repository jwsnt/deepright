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
+ 设计前需要仔细阅读Connect的设计
    + Connect介绍：../../../REQUIREMENT.md
+ 严格遵守原始报文JSON SCHEMA：../../REQUEST_SCHEMA.json）
+ 严格遵守测试必过集：../../TEST_CASE.md

### 需求介绍
+ 飞书插件新增 `openid` 命令：
```
./feishu openid
```
    + 只能查询 Integration / Connect 已保存的本地飞书消息快照，不得调用飞书 Open API，也不得扫描 `feishu.log` 或飞书插件临时状态文件。
    + 从 Integration 运行目录的 `config/config.json` 读取 `feishu.lastMessage`，该值为正整数，单位为小时。
    + 查询窗口为执行命令时刻向前回溯 `feishu.lastMessage` 小时，按飞书消息发送时间筛选。
    + `config/config.json` 不存在、JSON 无法解析、缺少 `feishu.lastMessage`、值不是正整数时，命令必须立即失败并输出清晰错误；不得静默使用默认值。
    + 结果中的 `openid` 必须唯一；同一 `openid` 有多条消息时，仅保留发送时间最新的一条。
    + 返回 JSON 数组，字段固定为 `openid`、`lastMessageAt`，按 `lastMessageAt` 倒序排列。
+ 飞书插件新增 `search` 命令：
```
./feishu search --query "关键词"
```
    + 查询时间窗口、配置读取和错误处理规则与 `openid` 命令完全一致。
    + 仅查询插件归一化后的文本消息内容；纯图片、纯文件消息不得参与搜索。
    + 内容匹配为不区分大小写的包含匹配。
    + `--query` 可省略；省略或传入空字符串时，列出时间窗口内全部可搜索的文本消息。空白分隔的多个关键词为 AND 关系；双引号包裹的连续内容视为一个完整短语，例如：`--query '"退款申请" 已处理'`。
    + 支持可选 `--openid`，对发送者 Open ID 作精确匹配。`--openid` 与 `--query` 同时提供时取 AND；仅提供 `--openid` 时，返回该发送者窗口内全部可搜索的文本消息。
    + 支持 `--limit` 和 `--offset` 分页：`--limit` 默认 50、最大 200，`--offset` 默认 0；负数、0（仅 `limit`）或非整数必须立即报错。
    + 返回 JSON 对象，字段固定为 `total`、`limit`、`offset`、`items`；`total` 为过滤后窗口内全部文本消息数。`items` 中每项字段固定为 `messageId`、`openid`、`content`、`sentAt`，按 `sentAt` 倒序排列。
+ 飞书消息成功写入 `connect_request` 时，飞书插件必须将归一化后的消息快照作为通用能力输入一并提交；快照必须保留来源、消息 ID、发送者 ID、归一化内容、消息类型和发送时间。消息 ID 在同一来源内唯一，重复入库不得产生重复快照。
+ Integration / Connect 只能提供与来源无关的消息快照存储和查询能力：不得解析飞书 envelope、不得包含飞书 `openid`、配置字段或命令语义，也不得依赖飞书插件包。
+ 飞书插件负责把通用快照中的发送者 ID 映射为自身的 `openid` 输出，并负责读取 `feishu.lastMessage` 配置；Integration / Connect 只接收 `source`、消息属性、查询窗口和搜索参数。
+ 必须优化数据库查询：
    + 为时间范围和发送者聚合建立 `(source, sent_at DESC)`、`(source, sender_id, sent_at DESC)` 索引。
    + 为文本的任意包含检索建立 SQLite FTS5 trigram 索引；短关键词或 FTS 无法覆盖的情况仍必须保证包含匹配语义正确。
    + 所有查询必须使用参数绑定，不得拼接用户的搜索内容到 SQL 中。
+ 飞书插件查询数据库必须统一经 Integration / Connect CLI 代理执行；飞书模块不得直接连接 SQLite、不得自行定位数据库文件。
+ `feishu command` 输出必须包含 `openid`、`search`；`feishu help` / `feishu --help` 必须说明参数、返回结果、错误条件和可直接复制的完整案例。

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
