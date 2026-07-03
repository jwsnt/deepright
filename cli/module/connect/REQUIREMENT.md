### 第一性原则
+ 仅可以新增/更新/删除当前需求文档（REQUIREMENT.md）同目录的文件和文件夹
+ 如非授权，禁止修改其他插件目录文件和文件夹
### 技术规范
+ 严格遵守整体设计文档：../DESIGN.md
    + browser、email、feishu等
+ 本模块设计文档：DESIGN.md

### 需求介绍
+ 与三方工具进行连接的模块：接收消息、发送消息

### Connect
+ 由连接模块元数据、三方请求（Request）和三方响应（Response）组成
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

###### 连接模块元数据
+ 基本数据
    + 三方名称：string（唯一）
    + 连接元数据：每家连接需要提供的参数不也一样，使用Json String
    + callback：string，插件应用程序本地回调用地址的绝对路径，与插件运行时主键同名（meta-create、meta-update时必须提供）
    + 是否支持Stream（流式回复）：boolean
    + 绑定的AgentId：string
    + 会话ID（CHAT_ID）：string
    + 选择的模型：string
    + 是否深度思考：boolean
    + 创建和最后更新时间
+ 创建时需要检查数据，指定Agent必须存在，指定模型是否在Proxy已注册（填写Token），及其他参数必须符合格式和要求
+ 案例：
    + 使用connect meta-create创建元数据
    + 使用connect meta-update更新元数据（name用于定位，不可更新）
    + 使用connect meta-delete删除元数据
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 插件标识统一原则：展示名（name）可以是中文，系统主键（key）必须稳定唯一，所有运行时链路只能用主键，不能混用展示名
+ 最终用户验收与文档应优先写integration顶层入口，connect list-meta仅作为内部实现或兼容说明

###### 三方请求数据
+ 基本数据
    + 三方名称：string
    + 外部ID：string（三方名称+外部ID构成唯一键）
    + 请求内容：string
    + 附件内容：以,分隔的字符串附件路径
    + 原始请求：string
    + 状态：0，待处理、已启动、已过期、已回复
    + 创建时间
+ 以三方名称和创建时间构建索引
+ 接收时需要检查数据，指定name在连接模块元数据必须存在且未删除，指定Agent必须存在，指定模型是否在Proxy已注册（填写Token），及其他参数必须符合格式和要求

###### 三方响应数据
+ 基本数据
    + 三方名称：string
    + 三方请求数据ID
    + 响应内容
    + 附件内容：以,分隔的字符串
    + 创建时间
+ 推送时需要检查数据，指定name在连接模块元数据必须存在且未删除，指定Agent必须存在，指定模型是否在Proxy已注册（填写Token），及其他参数必须符合格式和要求

###### 进程服务
+ connect模块是一个HTTP服务，需要具备独立的CLI启动（start），终止（stop）命令，执行错误需要抛出异常
+ connect模块命令（如meta-create、add-request等）通过http://127.0.0.1向服务发送请求，而不能直接调用数据库
    + 如果服务没有启动，那么以上命令应该抛出异常

###### list-plugins
> 新增自 iteration/20260504-1/REQUIREMENT.md
+ 新增list-plugins命令：获取plugins目录（子孙目录不需要）所有二进制可执行文件的name和param组合JSON
+ 每次执行后缓存，缓存时间以命令行参数--connect-cache指定的毫秒数，默认10秒

###### list-meta
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 新增list-meta命令：获取当前所有已配置的插件meta
+ 插件标识统一原则参考上方「连接模块元数据」

###### add-request
> 新增自 iteration/20260507-1/REQUIREMENT.md
+ 新增add-request命令：保存三方请求数据进数据库
+ 参数：--key（三方Key）、--externalId（外部ID）、--content（请求内容）、--artifacts（以,分隔的字符串附件路径）、--original（原始请求）、--status（状态）、--created（创建时间的时间戳）

###### meta-get
> 新增自 iteration/20260507-2/REQUIREMENT.md
+ 新增meta-get命令：获取指定插件的配置
+ 案例：`./integration connect meta-get --key 三方Key` 返回 `{...}`

###### meta-create / meta-update
> 新增自 iteration/20260507-4/REQUIREMENT.md
+ 新增meta-create命令：创建指定插件的配置
    + 案例：`./integration connect meta-create --key 三方Key --meta {} --callback ...` 响应 OK
+ 新增meta-update命令：更新指定插件的配置
    + 案例：`./integration connect meta-update --key 三方Key --meta {} --callback ...` 响应 OK
+ meta-create/meta-update的--callback固定为应用启动路径下plugins目录下与插件key同名的可执行程序
    + 例如应用路径为/home/integration, 插件key=a, 则--callback为/home/integration/plugins/a

###### help
> 新增自 iteration/20260507-3/REQUIREMENT.md
+ 新增help命令：提供完整的插件使用手册
+ 案例：`./connect help`

### 链路整理
+ 创建连接模块元数据
+ 三方连接模块：启动应用后通过CLI命令, 从连接模块获取元数据，并开始读取消息
+ 三方连接模块：读取消息后通过CLI命令, 向连接模块推送消息
+ 连接模块: 处理消息后通过CLI命令, 向三方连接模块推送响应

### 插件机制
+ 所有插件内部识别、回调映射和通知发送必须统一使 key（规范化插件名），name仅用于界面和日志展示

#### 飞书插件
> 引用自 feishu/REQUIREMENT.md（含 feishu/iteration/20260504-2、20260505-1）
###### 启动与停止
+ 独立CLI启动（start）、终止（stop）命令，运行时主键固定为"feishu"，展示名为"飞书"
+ 每60秒检查一次心跳，心跳不存在或无法接收则自动断开重连
+ 每次收到消息后追加到feishu.log（单日志最大10M，最多5个）
+ 启动/停止均通过PID文件管理进程
###### param命令
> 新增自 feishu/iteration/20260504-2/REQUIREMENT.md
+ 返回启动飞书时需提供的参数SCHEMA，固定返回：["appId","appSecret"]
###### name命令
> 新增自 feishu/iteration/20260504-2/REQUIREMENT.md
+ 返回飞书插件的系统主键和展示名，固定返回：{"key":"feishu","name":"飞书"}
###### send命令
> 新增自 feishu/iteration/20260505-1/REQUIREMENT.md
+ 向飞书推送消息（回复已有消息）
    + message为add-request原始报文（必填），image/file可为空
    + 文本、图片、文件至少提供一种，否则报错
+ 文本消息以飞书interactive卡片发送，Markdown渲染
+ 图片/文件：先上传再发送；同时带附件时顺序为图片→文件→文本
+ 每次推送在feishu.log记录调用
###### 报文映射
+ create_time + content 的 MD5 构成唯一键
+ 图片通过 message_resource.get 下载到 feishu_artifacts
+ 文件下载到 feishu_artifacts，file_key 命名
+ 下载失败只记日志，不崩溃
+ 内容归一化：图片→[image]路径，文件→[file]路径
###### init命令
> 新增自 feishu/iteration/20260507-1/REQUIREMENT.md
+ 新增init命令：向飞书推送任务初始化消息，参数和处理方式与send命令相同，日志提示调用了init
+ 案例：`./feishu init --message 原消息报文 --content 消息文本内容 --image 图片附件 --file 文件附件`
###### command命令
> 新增自 feishu/iteration/20260507-3/REQUIREMENT.md
+ 新增command命令：返回飞书插件的功能列表
+ 案例：`./feishu command`
###### 数据存储
> 新增自 feishu/iteration/20260507-2/REQUIREMENT.md
+ 飞书保存三方请求数据改为使用add-request命令
    + --key固定为name命令返回的key（feishu）
+ 保持模块独立，不自行操作数据库或调用非模块内代码

####  邮件插件
> 引用自 email/REQUIREMENT.md（含 email/iteration/20260506-1、20260506-2）
###### 启动与停止
+ 独立CLI启动（start）、终止（stop）命令，运行时主键固定为"email"，展示名为"邮件"
+ 启动后自动登录邮箱，每60秒扫描一次未读邮件
+ 运行日志写入email.log
###### send命令（回复邮件）
> 新增自 email/iteration/20260506-2/REQUIREMENT.md
+ CLI命令send，向邮件推送回复消息
    + message为add-request原始报文（必填），image/file可为空
+ 文本生成富文本正文，图片/文件使用邮件附件发送
+ 回复时遵守RFC 5322：In-Reply-To / References 跟踪会话
+ 历史邮件回复时沿用原会话链路

###### 邮件接收与处理
> 新增自 email/iteration/20260506-1/REQUIREMENT.md
+ 准入名单：仅处理email_whitelist中的发件人或email自身
+ 时间过滤：启动时间前30分钟为起始线，更新为最新处理时间
+ 去重：记录已处理的Message-ID，不再重复
+ 持久配置：时间过滤和去重持久化，重启可恢复
+ 报文映射：Date字段为create_time；headers+content为原始请求
+ 附件下载到email_artifacts，image_key/file_key命名
+ 多协议解析：支持嵌入和附件两种方式
###### init命令
> 新增自 email/iteration/20260507-1/REQUIREMENT.md
+ 新增init命令：向邮件推送任务初始化消息，参数和处理方式与send命令相同，日志提示调用了init
+ 案例：`./email init --message 原消息报文 --content 消息文本内容 --image 图片附件 --file 文件附件`
###### command命令
> 新增自 email/iteration/20260507-3/REQUIREMENT.md
+ 新增command命令：返回邮件插件的功能列表
+ 案例：`./email command`
###### 数据存储
> 新增自 email/iteration/20260507-2/REQUIREMENT.md
+ 邮件保存三方请求数据改为使用add-request命令
    + --key固定为name命令返回的key（email）
+ 保持模块独立，不自行操作数据库或调用非模块内代码

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
    + 使用文件名为data的sqlite存储，并使用连接池
        + 与Cron模块公用一个数据库：../cron/REQUIREMENT.md
    + 每次生成后缓存，缓存时间以命令行参数--connect-cache指定的毫秒数，默认10秒
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写
> 补充自 iteration/20260504-1/REQUIREMENT.md
+ list-plugins 每次执行后缓存，时间由 --connect-cache 控制，默认10秒
> 补充自 feishu/REQUIREMENT.md
+ 飞书：编译应用名feishu，二进制放connect/plugins目录
> 补充自 email/REQUIREMENT.md
+ 邮件：编译应用名email，二进制放connect/plugins目录
> 新增自 iteration/20260518-1/REQUIREMENT.md
+ add-request命令新增可选参数--schema（Json String），对应任务明细的response_schema
> 新增自 iteration/20260518-1/REQUIREMENT.md
+ add-request命令新增可选参数--schema（Json String），对应任务明细的response_schema
> 新增自 iteration/20260518-1/REQUIREMENT.md
+ add-request命令新增可选参数--schema（Json String），对应任务明细的response_schema
> 新增自 iteration/20260518-1/REQUIREMENT.md
+ add-request命令新增可选参数--schema（Json String），对应任务明细的response_schema






### 撰写手册
+ 编写USER_GUIDE.md

### 关联需求
+ 飞书插件：feishu/REQUIREMENT.md
+ 邮件插件：email/REQUIREMENT.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 相关迭代：iteration/日期/REQUIREMENT.md
+ 同步代码：../integration/REQUIREMENT.md

> 合并截止：
    - Connect模块迭代：./iteration/20260518-1/REQUIREMENT.md
    - 飞书插件迭代：feishu/iteration/20260612-2/REQUIREMENT.md
    - 邮件插件迭代：email/iteration/20260612-3/REQUIREMENT.md
下次合并从此之后的新迭代开始