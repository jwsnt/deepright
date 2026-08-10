### 第一性原则
+ 仅可以新增/更新/删除当前需求文档（REQUIREMENT.md）同目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../DESIGN.md
+ 本模块设计文档：../DESIGN.md

### 需求介绍
+ 邮件插件模块，负责通过POP3/SMTP协议收发邮件，将准入邮件消息推送至Connect的add-request，并支持基于add-request回复邮件
+ 二进制收口, 最终交付给用户的主程序必须是`email`一个二进制文件，日志必须是与email同目录下的email.log

### 插件元数据
> 新增自 iteration/20260506-1/REQUIREMENT.md
+ 运行时主键固定为`email`，展示名为`邮件`
+ 需要提供的参数：
    + `email` — 邮箱账号
    + `email_pop3` — POP3服务器地址
    + `email_smtp` — SMTP服务器地址
    + `email_password` — 邮箱密码或授权码
    + `email_whitelist` — 以逗号分隔的白名单邮件地址，未填或为空表示不过滤

### CLI命令
#### 启动与停止
> 新增自 iteration/20260506-1/REQUIREMENT.md
+ 独立的CLI启动（`start`）和终止（`stop`）命令，执行错误需抛出异常
+ 启动和关闭规则参考Connect模块及飞书模块实现
+ `start` 若已存在运行中的进程，先执行 `stop`，整体行为等同 `restart`
+ 启动后自动使用代码登录邮箱，每60秒扫描一次未读邮件
+ 运行日志写入 `email.log`
+ 配置通过 Integration 代理的 Connect 能力获取 `name=email` 的元数据

#### 发送邮件
> 新增自 iteration/20260506-2/REQUIREMENT.md
+ CLI命令 `send`，向邮件推送消息：
```
./email send --message 原消息报文（json string） --content 消息文本内容 --image 以逗号分隔的图片附件 --file 以逗号分隔的文件附件
```
    + `--message` 为 add-request 的原始报文（JSON string），必填
    + `--image`、`--file` 可为空
+ 启动参数（密钥等）获取方式同 `./email start`
+ 文本消息生成富文本正文，图片和文件使用邮件附件发送
+ 若 `--message` 能解析出原始邮件头，则回复原发件人并跟踪会话（遵守RFC 5322）：
    + `In-Reply-To` = 父邮件的 Message-ID
    + `References` = 原有 References + 父邮件的 Message-ID
+ 历史邮件再次作为Query回复时，必须沿用原邮件会话链路，不能作为新邮件发送
+ 每次推送在 `email.log` 记录调用

### 邮件接收与处理
> 新增自 iteration/20260506-1/REQUIREMENT.md
+ 收到邮件后提取邮件消息并推送至 add-request，处理规则如下：

###### 准入名单
+ 发件人必须是 `email_whitelist` 中定义的白名单用户或 `email` 地址自身，否则跳过并记录在 `email.log`

###### 时间过滤
+ 应用启动时间前30分钟为起始时间线（如 20260506 15:00:00 启动，起始为 20260506 14:30:00）
+ 记录每次处理过的邮件最后时间（包括非准入邮件）为最新时间线，下次仅处理该时间线后的邮件

###### 去重
+ 记录准入后的邮件 `Message-ID`，已处理过的邮件不再处理

###### 持久配置
+ 时间过滤和去重均需持久化记录，保证多次重启后可恢复

###### 报文映射
+ `create_time`：使用邮件消息头中的 `Date` 字段（RFC 5322）
+ 原始请求（用于回溯数据）：邮件报文头、标题、内容组成JSON — `{"headers":[{}],"content":""}`
+ 请求内容（用于执行备忘录明细）：邮件标题、内容、下载附件拼接
+ 文字编码统一解析为UTF-8，避免乱码

###### 附件与资源下载
+ 图片/文件下载到应用启动目录下的 `email_artifacts` 目录，不存在则新建
+ 图片使用 `image_key` 命名，文件使用 `file_key` 命名
+ 下载后在 `artifacts` 属性上追加本地文件系统绝对路径
+ 内容归一化格式：
    + 图片：`[image]图片绝对链接`
    + 文件：`[file]文件绝对路径`
+ 多协议资源解析：
    + 嵌入邮件本身的图片或附件
    + 附件邮件中的资源

### 调试
> 新增自 iteration/20260506-1/REQUIREMENT.md
+ 解析eml文件：
    + 从eml文件中提取from、to、date、messageId、text、html、image、file、header
    + eml_file_path = 目标文件

### 链路整理
> 新增自 iteration/20260506-1/REQUIREMENT.md
+ 邮件模块，通过命令行启动
+ 邮件模块，从Integration代理的Connect能力获取 `name=email_smtp` 的启动配置
+ 邮件模块，管理连接并等待邮件
    + 接收文字邮件
    + 接收图片邮件
+ 邮件模块，收到消息并向Integration代理的Connect能力推送
> 新增自 iteration/20260506-2/REQUIREMENT.md
+ 邮件模块，通过CLI命令 `send` 回复邮件
+ 回复时从Integration代理的Connect能力获取配置，通过SMTP发送

### init命令
> 新增自 ./iteration/20260507-1/REQUIREMENT.md
+ 新增CLI命令 `init`，向邮件推送任务初始化消息，参数与send相同（代理方法模式）
+ 日志需要提示调用了init
```
./email init --message 原消息报文（json string） --content 消息文本内容 --image 以逗号分隔的图片附件 --file 以逗号分隔的文件附件
```
### 数据存储改用add-request
> 新增自 ./iteration/20260507-2/REQUIREMENT.md
+ 邮件保存三方请求数据改为使用add-request命令
+ --key需要与name命令返回的key相同（email）
+ 保持自身模块独立，不自行操作数据库或调用非模块内代码

### command命令
> 新增自 ./iteration/20260507-3/REQUIREMENT.md
+ 新增CLI命令 `command`，返回邮件插件的功能列表
```
./email command
```

### 邮件日志结构化
> 新增自 ./iteration/20260521-1/REQUIREMENT.md
+ 日志（email.log）里永远输出已解码主题
+ 日志（email.log）里输出结构化字段，如 subject=今天天气 from=... message_id=...

### 邮件立即触发
> 新增自 ./iteration/20260521-2/REQUIREMENT.md
+ 邮件插件收到邮件后立即触发add-request命令

### 邮件解码RFC 2047
> 新增自 ./iteration/20260521-3/REQUIREMENT.md
+ 如果邮件使用了RFC 2047/MIME encoded-word格式（如 ），需要解码转为任务明细、原始请求和排障日志

### 发送日志
> 新增自 ./iteration/20260522-1/REQUIREMENT.md
+ 邮件插件发送消息时需要记录详细内容，包括请求报文（报文头）、解析结果、发送结果、失败原因
+ 发送消息包括 init 命令和 send 命令

### 同步代码
+ 所以设计/编译都需要遵守email的二进制和CLI收口原则

### 插件参数标准化
> 新增自 ./iteration/20260610-1/REQUIREMENT.md
+ 修改命令param，固定返回：
```
[{"email":"邮箱地址，如hello_world@gmail.com","email_pop3":"邮箱的pop3地址，如pop.gmail.com","email_smtp":"邮箱的smtp地址，如smtp.gmail.com","email_password":"邮箱的密码","email_whitelist":"以逗号分隔的收件人白名单，如a@gmail.com,b@gmail.com。","email_pop3_interval":"每次扫描待处理邮件的间隔秒数，默认300"}]
```

### 会话复用
> 新增自 ./iteration/20260612-1/REQUIREMENT.md
+ 在插件页面点击复用当前会话，启动email时使用当前ChatID作为后续备忘录任务明细的ChatID
+ 如果未点击复用，则会话（ChatID）固定为email

### JSON提取兼容
> 新增自 ./iteration/20260612-3/REQUIREMENT.md
+ 如果响应报文无法解析成JSON，尝试使用正则从文本中提取JSON并应用到schema中
+ 典型案例：前缀干扰文本+JSON，需提取纯净JSON部分
+ 如果依旧提取失败则使用原逻辑，全部发送

### sender与search命令
> 新增自 ./iteration/20260719-1/REQUIREMENT.md
+ 新增 `sender` 命令，查询 Integration / Connect 已保存的本地通用消息快照；不得连接 POP3/SMTP、读取邮件日志或直接连接 SQLite。
```
./email sender
```
    + 查询窗口从 Integration 运行目录 `config/config.json` 的 `email.lastMessage` 读取，单位为小时，必须为正整数。
    + 配置文件不存在、JSON 非法、字段缺失或值非正整数时立即失败，不得回退默认值。
    + 从 `From` 头解析第一个有效邮箱、去除显示名并转小写；输出按最后发送时间倒序的 JSON 数组，元素固定为 `sender`、`lastMessageAt`；同一邮箱仅保留最后发送时间。
+ 新增 `search` 命令，查询同一窗口内归一化后的邮件主题和正文：
```
./email search --query "关键词"
```
    + `--query` 可省略；省略或为空时列出窗口内全部文本消息。多个空白分隔关键词为 AND；双引号内的连续内容是完整短语；匹配不区分大小写且为包含匹配。
    + 支持 `--sender` 对标准化后的发件人邮箱精确过滤，可与 `--query` 组合为 AND。
    + 不搜索附件二进制内容或附件路径。
    + 支持 `--limit`（默认 50、最大 200）与 `--offset`（默认 0）；非法参数必须立即失败。
    + 返回 JSON 对象，字段为 `total`、`limit`、`offset`、`items`；每个 item 固定为 `messageId`、`sender`、`content`、`sentAt`，并按 `sentAt` 倒序。
+ 邮件在调用 `connect add-request` 时将自身归一化消息作为 `source=email` 的通用消息快照一并提交。发送时间优先使用邮件 `Date`，无效时回退插件接收时间。
+ Integration / Connect 仅提供通用消息快照的存储、索引和查询能力，不得依赖邮件插件、解析邮件报文或包含邮件配置/发件人语义。
+ 通用能力为时间范围、发送者聚合建立索引，并使用 SQLite FTS5 trigram 优化文本包含查询；邮件插件只通过 Integration / Connect CLI 使用该能力。
+ `command` 必须列出 `sender`、`search`；全局帮助及两个子命令的 `--help` 必须包含参数、输出、错误条件和完整案例。

### 编写代码
+ 以Golang编写以上代码，要求：
    + 邮件内部调用Connect语义必须统一通过integration代理执行，遵循 `connect <subcommand> [options...]` 格式，子命令必须固定放在第一个位置
    + 邮件模块只负责从Integration代理的Connect能力获取配置来维护长连接、获取和推送消息，不直接连接db和指定agent目录
    + 用户发送的图片或文件，必须同时支持从附件解析和从邮件嵌入内容解析
    + 图片下载后落到 `email_artifacts`，用 `image_key` 命名
    + 文件下载后落到 `email_artifacts`，用 `file_key` 命名
    + 用标准文件名 `email` 启动时直接执行二进制
    + 下载失败时只能记日志，不能因空响应导致email进程崩溃
    + 编译应用名：`email`
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写
+ 编译后的二进制文件放在connect模块同目录的 `plugins` 目录

### 验证测试
+ 测试必过集：TEST_CASE.md
> 新增自 iteration/20260506-1/REQUIREMENT.md
+ 使用mock数据验证流程：
    + 模拟邮件模块启动，正确通过CLI从Integration代理的Connect能力获取配置，并建立独立扫描任务
    + 模拟邮件模块收到消息，并正确通过CLI向Integration代理的Connect能力推送消息
``` 启动 integration
./integration --agent-dir ../agent/test-case --site ../site
{
  "status": "started"
}
``` 通过integration注册邮件
./integration connect meta-create \
  --name email \
  --meta '{"email":使用环境变量EMAIL,"email_address":使用环境变量EMAIL_ADDRESS,"email_password":使用环境变量EMAIL_PASSWORD,"mode":"email_smtp"}' \
  --stream true \
  --callback ./email \
  --agent a \
  --model deepseek
```
``` 启动邮件
./integration plugins start --name email
```
``` 如需独立验证插件CLI
./email start --connect-bin ../integration/integration
```
``` 关闭邮件
./email stop --pid-file ./email.pid
```
``` 通过integration注销邮件
./integration meta-delete --name email
```
``` 关闭integration
停止当前integration进程
```

> 新增自 iteration/20260506-2/REQUIREMENT.md
```
./integration connect meta-create --name email --meta '{"email":从系统环境变量获取$EMAIL,"email_pop3":从系统环境变量获取$EMAIL_POP3,"email_smtp":从系统环境变量获取$EMAIL_SMTP,"email_password":从系统环境变量获取$EMAIL_PASSWORD,"email_whitelist":""}'
```
    + 其中 `--meta` 字段集合必须与 `./email param` 返回值一致
```
./email send --message 原消息报文（json string） --content 消息文本内容 --image 以逗号分隔的图片附件 --file 以逗号分隔的文件附件
```
> 新增自 iteration/20260518-1/REQUIREMENT.md
+ 新增email schema命令，返回邮件插件的Response Json Schema
+ 为mail的command命令添加schema命令
> 新增自 iteration/20260518-2/REQUIREMENT.md
+ email调用add-request时通过自身schema命令获取response_schema，通过--schema传递
> 新增自 iteration/20260518-3/REQUIREMENT.md
+ send命令检查--content是否为符合schema的JSON
+ 符合则归一化：content纯文本（图片替换为文件名）、artifacts作为附件
+ 不符合或异常降级为整体发送
+ init/send共用schema归一化与图片替换逻辑
+ 发送邮件超时180秒
> 新增自 iteration/20260518-4/REQUIREMENT.md
+ param命令新增email_pop3_interval参数
+ pop3使用email_pop3_seconds（默认300秒）间隔轮询
+ pop3/stmp日志记录到email.log，分卷最大10M最多4个
> 新增自 iteration/20260520-1/REQUIREMENT.md
+ mail启动start后仅连接一次保持Pop3连接，除非断开否则持续复用
> 新增自 iteration/20260518-1/REQUIREMENT.md
+ 新增email schema命令，返回邮件插件的Response Json Schema
+ 为mail的command命令添加schema命令
> 新增自 iteration/20260518-2/REQUIREMENT.md
+ email调用add-request时通过自身schema命令获取response_schema，通过--schema传递
> 新增自 iteration/20260518-3/REQUIREMENT.md
+ send命令处理发送消息时检查--content是否为符合schema的JSON
+ 符合schema则归一化处理：content转纯文本（图片链接替换为文件名）、artifacts作为附件发送
+ 不符合或异常则降级，整个--content作为整体发送
+ init命令与send命令共用相同的schema归一化与图片替换逻辑
+ 发送邮件超时180秒
> 新增自 iteration/20260518-4/REQUIREMENT.md
+ param命令新增email_pop3_interval参数
+ pop3使用email_pop3_seconds（默认300秒）间隔轮询扫描邮件
+ pop3/smtp明细日志（含失败）记录到email.log，分卷最大10M，最多4个分卷
> 新增自 iteration/20260520-1/REQUIREMENT.md
+ 插件mail启动start后仅连接一次并保持Pop3连接，除非断开否则持续复用
> 新增自 iteration/20260518-1/REQUIREMENT.md
+ 新增email schema命令，返回邮件插件的Response Json Schema
+ 为mail的command命令添加schema命令
> 新增自 iteration/20260518-2/REQUIREMENT.md
+ email调用add-request时通过自身schema命令获取response_schema，通过--schema传递
> 新增自 iteration/20260518-3/REQUIREMENT.md
+ send命令处理发送消息时检查--content是否为符合schema的JSON
+ 符合schema则归一化处理：content转纯文本（图片链接替换为文件名）、artifacts作为附件发送
+ 不符合或异常则降级，整个--content作为整体发送
+ init命令与send命令共用相同的schema归一化与图片替换逻辑
+ 发送邮件超时180秒
> 新增自 iteration/20260518-4/REQUIREMENT.md
+ param命令新增email_pop3_interval参数
+ pop3使用email_pop3_seconds（默认300秒）间隔轮询扫描邮件
+ pop3/smtp明细日志（含失败）记录到email.log，分卷最大10M，最多4个分卷
> 新增自 iteration/20260520-1/REQUIREMENT.md
+ 插件mail启动start后仅连接一次并保持Pop3连接，除非断开否则持续复用
> 新增自 iteration/20260518-1/REQUIREMENT.md
+ 新增email schema命令，返回邮件插件的Response Json Schema
+ 为mail的command命令添加schema命令
> 新增自 iteration/20260518-2/REQUIREMENT.md
+ email调用add-request时通过自身schema命令获取response_schema，通过--schema传递
> 新增自 iteration/20260518-3/REQUIREMENT.md
+ send命令处理发送消息时检查--content是否为符合schema的JSON
+ 符合schema则归一化处理：content转纯文本（图片链接替换为文件名）、artifacts作为附件发送
+ 不符合或异常则降级，整个--content作为整体发送
+ init命令与send命令共用相同的schema归一化与图片替换逻辑
+ 发送邮件超时180秒
> 新增自 iteration/20260518-4/REQUIREMENT.md
+ param命令新增email_pop3_interval参数
+ pop3使用email_pop3_seconds（默认300秒）间隔轮询扫描邮件
+ pop3/smtp明细日志（含失败）记录到email.log，分卷最大10M，最多4个分卷
> 新增自 iteration/20260520-1/REQUIREMENT.md
+ 插件mail启动start后仅连接一次并保持Pop3连接，除非断开否则持续复用


























### 撰写手册
+ 编写USER_GUIDE.md

### 关联需求
+ 邮件插件：email/iteration/日期/REQUIREMENT.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 相关迭代：iteration/日期/REQUIREMENT.md
+ 同步代码：../../integration/REQUIREMENT.md（每次都要同步更新代码）
+ 复制至Plugin：../../../../plugins/
+ Connect介绍：../REQUIREMENT.md
+ Connect手册：../USER_GUIDE.md
+ 不能破坏现有设计和功能
+ 设计和编译需遵守integration的二进制和CLI收口原则
+ 设计前需仔细阅读Connect的设计以保证兼容方案
> 合并截止：./iteration/20260612-3/REQUIREMENT.md，下次合并从此之后的新迭代开始
