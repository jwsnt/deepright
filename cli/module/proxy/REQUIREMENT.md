### 第一性原则
+ 仅可以新增/更新/删除当前需求文档（REQUIREMENT.md）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../DESIGN.md
+ 本模块设计文档：DESIGN.md

### 需求介绍
+ 使用Stream模式代理请求OPEN AI服务，并流式响应结果

### Agent元数据
+ Agent元数据介绍：../agent/REQUIREMENT.md
+ Agent元数据手册：../agent/USER_GUIDE.md
+ Agent元数据迭代：../agent/iteration/日期/REQUIREMENT.md

### 代理请求转发
+ 启动由命令行参数--port（默认8080）指定的统一HTTP服务，其中POST路径/v1/chat/completions用于代理转发
+ 该服务会接类似如下CURL命令的请求报文：
``` 参考请求案例
curl --location 'http://xxx/v1/chat/completions' \
--header 'Content-Type: application/json' \
--header 'Authorization: xxxxx' \
--data-raw '{
    "model": xxxx,
    "messages": [
        {
            "role": "user",
            "content": xxxxx
        }
    ],
    "stream": true
}'
```
###### 转发要求
+ 读取请求Body（内容为标准Open AI的SSE协议请求），现有参数保持不变，在参数messages平级处插入metadata，内容与cli/get保持一致：
    + 如果参数metadata已存在，则使用追加覆盖（append）逻辑而不是替换（replace）
    + cli/get参数metadata介绍：../cli-get/REQUIREMENT.md
    + cli/get参数metadata获取：../cli-get/USER_GUIDE.md
```
"metadata": Agent元数据
```
+ 原有的请求头Header保持不变，转发新增SSE请求头"Accept:text/event-stream"
+ 转发地址由命令行参数--host获取，默认为https://www.deepright.cn，与cli/get保持一致
    + 转发URL为地址加/v1/chat/completions，如：https://www.deepright.cn/v1/chat/completions
> 新增自 iteration/20260509-1/REQUIREMENT.md
+ metadata Agent 元数据需加上 plugins 信息
> 新增自 iteration/20260510-1/REQUIREMENT.md
+ 在转发`/v1/chat/completions`、`/cli/get`和`/cli/pub`的metadata中增加knowledge字段
```JSON
{
    "knowledge": {
        "lastUpdate": long,
        "path": string
    }
}
```
> 新增自 iteration/20260511-3/REQUIREMENT.md
+ metadata中的`agents[].skills`每次都需要实时遍历指定目录及其子孙目录后提取文件内容，不要缓存


> 新增自 ./iteration/20260515-3/REQUIREMENT.md
+ 转发`/v1/chat/completions`前检查metadata中knowledge属性的lastUpdate时间：
    + 如果lastUpdate距离当前未超过`--knowledge_update_interval`（默认2小时），则删除knowledge的lastUpdate属性
    + 如果已删除lastUpdate，但距离上次申请更新时间未超过`--knowledge_update_lock`（默认30分钟），则同样删除lastUpdate（防止并发更新）
    + 知识库需求：../knowledge/REQUIREMENT.md
> 新增自 ./iteration/20260516-1/REQUIREMENT.md
+ 如果metadata包含`knowledge_commit:true`，则必须保留lastUpdate而无需检查更新锁逻辑
+ 包含knowledge_commit的请求在SSE响应完全结束后更新知识库最后更新时间
    + knowledge_lastUpdate接口：../proxy/iteration/20260511-2/REQUIREMENT.md
###### 转发超时控制
> 补充自 iteration/20260419_2/REQUIREMENT.md
+ 代理转发的HTTP SSE请求/v1/chat/completions只设置连接超时，不设置读取超时：
    + 连接超时由命令行参数--connect_timeout指定的毫秒值，默认15000毫秒
    + 连上后等待直到成功或失败（无读超时）

###### 客户端断开处理
> 修正自 iteration/20260427-7/REQUIREMENT.md
+ 转发/v1/chat/completions的请求当客户端（如网页）断开时自动取消/关闭代理请求连接

### 代理响应转发
+ 被代理的服务响应报文协议为SSE，看起来如下：
```
data: {"id":"cmpl-1305b94c570f447fbde3180560736287","object":"chat.completion.chunk","created":1698999575,"model":"kimi-k2.5","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}
data: {"id":"cmpl-1305b94c570f447fbde3180560736287","object":"chat.completion.chunk","created":1698999575,"model":"kimi-k2.5","choices":[{"index":0,"delta":{"content":"你好"},"finish_reason":null}]}
data: {"id":"cmpl-1305b94c570f447fbde3180560736287","object":"chat.completion.chunk","created":1698999575,"model":"kimi-k2.5","choices":[{"index":0,"delta":{"content":"。"},"finish_reason":null}]}
data: {"id":"cmpl-1305b94c570f447fbde3180560736287","object":"chat.completion.chunk","created":1698999575,"model":"kimi-k2.5","choices":[{"index":0,"delta":{},"finish_reason":"stop","usage":{"prompt_tokens":19,"completion_tokens":13,"total_tokens":32}}]}
data: [DONE]
```
+ 原封不动转发，不要修改任何内容
+ 收到一段转发一段，不要聚合

### 链路整理
+ 接收请求，并追加报文
+ 转发请求，并转发响应

### 会话存储
> 新增自 iteration/20260427-8/REQUIREMENT.md
+ 转发/v1/chat/completions的请求和SSE响应以AgentId+Chat（会话ID）维度保存在sqlite中
    + 会话ID对应请求中"chat": "会话UUID"
+ 以追加内容形式更新，区分提问/回答、创建时间、类型（页面会话或定时任务）
+ 保存原始报文格式，时间前缀在响应到达后才生成，保证时间准确
+ 客户端断开时终止写入
+ 响应类型：正常记录SSE响应，异常记录异常原因，均可回溯展示
+ 异步写入不影响SSE延迟，收到一段写一段，禁止仅在结束时一次性保存
+ 与Cron模块共享文件名为data的sqlite存储，使用连接池

> 修正自 iteration/20260502-2/REQUIREMENT.md
+ 流式对话日志保存及/api/restore恢复回放时，必须按完整SSE event（data: {...}\n\n）为单位处理，禁止按固定字节分片直接转字符串，避免UTF-8中文截断乱码

> 补充自 iteration/20260430-1/REQUIREMENT.md
+ 会话存储数据按Agent、Chat、时间建索引

### 转发连接管理
> 新增自 iteration/20260427-9/REQUIREMENT.md
+ 按AgentId和Chat维度管理转发连接，同一Agent+Chat同时只能有一个活跃转发连接
+ 新建HTTP POST `/api/cancel?agentId=xxx&chat=yyy`：主动终止指定转发连接
+ 非主动断开（如刷新页面）不关闭转发连接，会话存储继续直到SSE完成或异常
+ 主动断开时在会话存储中记录主动断开标记（含时间），不再记录后续会话

### 会话恢复
> 新增自 iteration/20260427-10/REQUIREMENT.md
+ 新建HTTP POST `/api/restore?agentId=xxx&chat=yyy&timeline=zzz`：获取指定Agent+Chat在指定时间之后的所有会话记录（提问+回答），按时间顺序返回

### CLI/GET命令存储
> 新增自 iteration/20260427-11/REQUIREMENT.md
+ 将cli/get响应中需要执行的命令及命令响应以AgentId+Chat维度保存在sqlite中
+ 追加内容形式更新，区分收到请求时间和执行完成时间
+ Cmd不为空表示需要执行命令，执行结果即为命令响应

### Agent管理API
#### Agent列表
> 新增自 iteration/20260419_1/REQUIREMENT.md
+ HTTP GET `/api/agentId`：获取所有Agent的AgentId列表，返回`["A","B"]`

#### Workspace路径
> 新增自 iteration/20260419_7/REQUIREMENT.md
+ HTTP GET `/api/workspace?agentId=xxx`：获取指定Agent的workspace绝对路径

#### 打开文件夹
> 新增自 iteration/20260419_3/REQUIREMENT.md
+ HTTP GET `/api/folder?agentId=xxx&dir=yyy`：调用本地命令行工具打开指定Agent的Workspace路径（dir可选，拼接子目录），需区分操作系统

#### Skills列表
> 新增自 iteration/20260419_4/REQUIREMENT.md
+ HTTP GET `/api/skills?agentId=xxx`：获取指定Agent的Skill名称列表

#### 初始化Agent
> 新增自 iteration/20260420_1/REQUIREMENT.md
+ HTTP GET `/api/agent/init?name=xxx`：在Agent目录下创建新文件夹，复制SOUL.md、USER.md和skills目录（不存在则创建空文件/空目录），立即刷新Agent/Skills缓存。name禁止空格
#### 删除Agent
> 新增自 iteration/20260420_2/REQUIREMENT.md
+ HTTP GET `/api/agent/delete?name=xxx`：删除Agent目录下指定目录，立即刷新Agent/Skills缓存
> 修正自 iteration/20260503-8/REQUIREMENT.md
+ 删除Agent时同时删除关联的所有备忘录元数据和明细（已完成的不删），需记录日志

#### 创建文件/文件夹
> 新增自 iteration/20260420_3/REQUIREMENT.md
+ HTTP GET `/api/agent/create?agentId=xxx&name=yyy&type=zzz`：在Agent workspace下创建文件(type=1)或文件夹(type=0)，立即刷新缓存
> 修正自 ./iteration/20260524-1/REQUIREMENT.md
    + name 升级为 workspace 内相对路径，允许形如 `docs/data`、`tmp/a/b` 的相对路径，以 `/` 作为目录分隔
    + name 的每个路径段必须满足：非空、不能是 `.` 或 `..`、不能包含空格和 `\:*?"<>|`
    + name 必须限制在当前 Agent 的 workspace 内，禁止绝对路径、`~`、`../` 或其他越界写入
    + type=0 时创建目录，若父目录不存在则按相对路径自动补齐
    + type=1 时创建文件，若父目录不存在则按相对路径自动补齐
    + 现有"已存在""Agent不存在""参数缺失"等错误语义保持不变

#### Agent配置
> 新增自 iteration/20260425-1/REQUIREMENT.md
+ HTTP POST `/api/config?agentId=xxx`：创建或更新Agent的config.json（description不超过200字），保存在Agent工作目录下

### 文件管理API
#### 文件列表
> 新增自 iteration/20260419_5/REQUIREMENT.md
+ HTTP GET `/api/files?path=xxx`：模糊查找（不区分大小写前缀匹配）指定绝对路径下文件和目录列表，不递归子孙目录
+ path支持绝对路径和~路径，支持空格路径
+ 返回name和type（file/dir）

#### 读取文件内容
> 新增自 iteration/20260419_6/REQUIREMENT.md
+ HTTP GET `/api/data?path=xxx`：获取指定绝对路径下文件内容，支持绝对路径/~路径/空格路径，不区分大小写
+ path为目录或二进制文件时抛出异常，返回status=1

#### 读取二进制流
> 新增自 iteration/20260419_10/REQUIREMENT.md
+ HTTP GET `/api/raw?agentId=xxx&path=yyy`：读取文件二进制流（Base64编码）
+ path支持相对路径（相对于Agent workspace）或文件系统绝对路径，支持空格

#### 写入文件
> 新增自 iteration/20260419_8/REQUIREMENT.md
+ HTTP GET `/api/edit?agentId=xxx&path=yyy`：在指定Agent工作目录相对路径下写入文件内容
+ path仅支持相对路径，支持空格，不区分大小写
+ 写入目录或二进制文件时抛出异常
> 修正自 iteration/20260502-1/REQUIREMENT.md
+ /api/edit接口支持二进制文件类型（图片、多媒体等）编辑，文件不存在则新建并保存

#### 删除文件
> 新增自 iteration/20260419_9/REQUIREMENT.md
+ HTTP GET `/api/del?agentId=xxx&path=yyy`：删除指定Agent工作目录下的文件或目录（目录递归删除）
+ path仅支持相对路径，支持空格，不区分大小写

#### 文件下载
> 新增自 iteration/20260427-3/REQUIREMENT.md
+ HTTP GET `/api/download?path=xxx`：下载指定文件系统的文件或目录（目录先打包zip），标准流式下载

#### 文件上传
> 新增自 iteration/20260422_1/REQUIREMENT.md
+ HTTP POST `/api/upload?agentId=xxx`：上传文件或文件夹到Agent目录下的tmp，支持文件夹（保留相对路径结构），最大200M


#### 文件或目录最后时间
> 新增自 ./iteration/20260515-4/REQUIREMENT.md
+ HTTP GET `/file/lastUpdate?file=xxx`：获取指定文件或目录最后更新时间距离当前时间的毫秒数
    + file支持绝对路径和相对路径（相对当前Agent目录），支持文件或目录
### 知识库接口
> 新增自 iteration/20260511-1/REQUIREMENT.md
+ HTTP GET `/knowledge`：映射当前应用启动目录下的knowledge目录
+ 访问目录时返回目录树形结构，访问文件时返回文件原始内容（与Nginx映射一致）
> 新增自 iteration/20260511-2/REQUIREMENT.md
+ HTTP GET `/knowledge_lastUpdate`：获取知识库最后更新时间，返回`yyyy-MM-dd HH:mm`格式

> 新增自 ./iteration/20260516-2/REQUIREMENT.md
+ HTTP GET `/knowledge_path`：获取知识库的真实文件系统绝对路径
    + 知识库需求：../knowledge/REQUIREMENT.md
> 新增自 ./iteration/20260516-1/REQUIREMENT.md
+ 包含`knowledge_commit`的请求在SSE响应结束后更新知识库最后更新时间（对应`/knowledge_lastUpdate`）

### 心跳与状态
#### 心跳上报
> 新增自 iteration/20260419_11/REQUIREMENT.md
+ 每次上报心跳（cli/get）后记录最后成功/失败/执行任务状态
    + 有待处理任务时即使执行失败也不归为心跳失败
+ HTTP GET `/api/heartbeat`：获取最后一次心跳时间和状态（0成功无任务/1失败/2成功有任务）

### 模型与密钥管理
#### Token管理
> 新增自 iteration/20260503-1/REQUIREMENT.md
+ HTTP POST `/api/token`：保存/更新模型名称与密钥，模型名称为唯一键，记录最后更新时间
+ HTTP GET `/api/token`：获取所有模型与密钥
+ 模型密钥和更新日志均存储到sqlite
> 修正自 iteration/20260503-3/REQUIREMENT.md
+ 创建备忘录（定时任务）时禁止存储模型Token，执行时从sqlite动态获取密钥

### 启动参数持久化
> 新增自 iteration/20260503-2/REQUIREMENT.md
+ 启动HTTP服务时将所有启动参数保存至 `config/config.json`（应用启动目录下），每次启动覆盖更新

### 数据库日志
> 补充自 iteration/20260503-7/REQUIREMENT.md
+ 所有Proxy模块数据库操作（含Agent模型和密钥）均需记录日志，按Agent、Chat、时间建索引
+ 日志表：`proxy_agent_provider_log`

### 系统命令执行
#### 执行CMD
> 新增自 iteration/20260501-1/REQUIREMENT.md
+ HTTP POST `/api/cmd`：执行指定Agent+Chat的系统命令
+ 仅127.0.0.1或localhost可执行，Agent必须存在且未删除
+ 安全检查：禁止包含rm（含& rm等变体）
+ 记录日志到sqlite（Agent、Chat、指令、开始/结束时间），按Agent/Chat/开始时间建索引
+ CMD实现方式与CLI保持一致

#### 终止CMD
> 新增自 iteration/20260501-2/REQUIREMENT.md
+ HTTP POST `/api/kill`：终止指定CMD系统命令
+ 仅127.0.0.1或localhost可执行，Agent必须存在且未删除
+ 记录kill日志到sqlite

### Cron/备忘录
#### 创建与删除
> 新增自 iteration/20260427-2/REQUIREMENT.md
+ HTTP POST `/api/cron/create?agentId=xxx`：创建备忘录元数据
    + 属性：Cron表达式（可选）、任务内容、模型、Thinking模式、原始时间、使用周期（仅一次/工作日/自然日/每小时/每15分钟/每30分钟）、AgentId
    + 周期任务（工作日/自然日/高频周期）创建后立即生成后5天内所有任务明细
    + 一次性任务早于当前时间则禁止创建
+ HTTP POST `/api/cron/delete?id=xxx`：删除元数据和关联明细
> 修正自 iteration/20260502-3/REQUIREMENT.md
+ 创建任务支持CHAT_ID参数；定时任务执行时若明细指定了CHAT_ID则使用，否则沿用@连接逻辑

#### 元数据查询
> 新增自 iteration/20260427-4/REQUIREMENT.md
+ HTTP POST `/api/cron/detail/metadata?agentId=xxx`：获取指定Agent的备忘录元数据集合

#### 明细查询
> 新增自 iteration/20260427-5/REQUIREMENT.md
+ HTTP POST `/api/cron/detail/list?agentId=xxx&date=yyy`：获取指定Agent+日期的所有任务明细（含已执行/待执行/无需执行/一次性）

#### 明细状态
> 新增自 iteration/20260427-6/REQUIREMENT.md
+ HTTP POST `/api/cron/detail/status?agentId=xxx&detailId=yyy&status=zzz`：更新指定任务明细状态

#### 定时任务执行
> 新增自 iteration/20260429-1/REQUIREMENT.md
+ 每分钟执行一次定时任务，获取已超时但未执行（且超时不超过1小时）的任务明细
+ 将任务明细转为转发/v1/chat/completions请求（会话ID=元数据ID@明细ID，或使用CHAT_ID），附带Metadata
+ 立即更新状态为已启动，SSE完成后更新为已完成，失败回滚到待启动
> 修正自 iteration/20260503-8/REQUIREMENT.md
+ 每分钟检查任务明细对应Agent是否存在且未删除、模型是否存在且有密钥，不存在则不执行
+ 检查元数据Agent是否存在，不存在则删除元数据和未完成明细（已完成保留）
+ /api/cron/delete删除时同时删除未完成明细（已完成保留），需记录日志

#### 数据库索引
> 补充自 iteration/20260430-1/REQUIREMENT.md
+ 备忘录数据按Agent、Chat、时间建索引

### Cron CLI工具
#### 创建
> 新增自 iteration/20260503-4/REQUIREMENT.md
+ `./proxy cron create`：CLI创建任务元数据，功能和Help参考Cron模块，需校验Agent存在及模型已注册Token
+ HTTP和CLI兼容，模型Token动态获取；非Cron必填参数从主应用 `config/config.json` 获取

#### 查询
> 新增自 iteration/20260503-5/REQUIREMENT.md
+ `./proxy cron find-meta`：按AgentId/Chat/模型/时间范围/执行周期查询元数据（未指定则全匹配）
+ `./proxy cron find-detail`：按元数据Id/AgentId/Chat/模型/执行周期/时间范围查询明细（未指定时间仅查当前时间之后）

#### 删除
> 新增自 iteration/20260503-6/REQUIREMENT.md
+ `./proxy cron delete-meta --id`：删除元数据
+ `./proxy cron delete-detail --metaId/--detailId`：删除明细

### 插件管理
#### 插件元数据
> 新增自 iteration/20260503-9/REQUIREMENT.md
+ HTTP GET `/api/plugins/meta`：获取插件信息（调用 `connect list-plugins` + `connect meta-list`），以 name 合并，返回名称/可填参数/已填参数，不缓存，每次都实时读取
> 新增自 ./iteration/20260517-1/REQUIREMENT.md
> 新增自 ./iteration/20260517-2/REQUIREMENT.md
+ 接口/api/plugins/meta读取每个插件的scope命令，获取容器可配置项列表
+ 接口/api/plugins/meta获取每个插件的name、param、scope命令需要并发执行，使用最短时间方案
+ 如果插件没有实现scope命令，则默认返回["reuse","agent","provider","thinking"]
+ 如果插件显式返回空数组[]，则表示完全不支持容器配置

#### 插件配置
> 新增自 iteration/20260503-11/REQUIREMENT.md
+ HTTP POST `/api/plugins/config?key=xxx&agentId=yyy`：更新插件配置
    + 系统主键key（稳定唯一）、展示名name（仅展示）、连接元数据、是否Stream、回调地址、绑定AgentId、CHAT_ID、模型、深度思考
    + 插件标识原则：name可中文展示，key必须稳定唯一，所有运行时链路只能用key
> 修正自 iteration/20260507-3/REQUIREMENT.md
+ 注册和更新插件配置更新为采用插件标准命令和语义
    + Connect meta-create / meta-update 命令：../connect/iteration/20260507-4/REQUIREMENT.md

#### 插件启停
> 新增自 iteration/20260503-14/REQUIREMENT.md
+ HTTP POST `/api/plugins/start?name=xxx`：启动指定插件
+ HTTP POST `/api/plugins/stop?name=xxx`：关闭指定插件
+ Plugin保持独立可执行文件

#### 插件状态
> 新增自 iteration/20260503-16/REQUIREMENT.md
+ HTTP GET `/api/plugins/status?name=xxx`：判断插件是否已启动（调用connect list-plugins检查进程）

#### 插件日志
> 新增自 iteration/20260503-10/REQUIREMENT.md
+ HTTP GET `/api/plugins/log?name=xxx`：获取插件同名.log文件的日志，SSE流式返回（等同tail -f），连接关闭或文件不存在时停止

#### Proxy整合Connect
> 补充自 iteration/20260503-12/REQUIREMENT.md
+ Proxy模块整合Connect模块功能，启动时自动启动Connect，关闭时自动关闭；Plugin模块不整合，作为独立可执行文件

### 三方消息处理
#### 消息转任务
> 新增自 iteration/20260503-15/REQUIREMENT.md
+ 定时任务中将待处理的add-request消息转为一次性任务，标记已启动/已完成
    + 任务内容：同插件所有待处理消息的文本拼接（图片/文件仅标记不参与拼接）
    + 模型/AgentID/思考模式/CHAT_ID来自插件meta-create注册信息
    + 不含文本内容的消息不处理，等待后续累计
    + 超过5分钟未处理的add-request合并为一条（状态=无需启动），标记原消息为已过期
+ 生成的任务明细需检查Agent存在且模型已注册密钥
+ 正常转换的任务明细立即执行
+ 每次轮询单插件仅发送一次开始通知（通过插件init命令回调），按META_ID精确定位原始消息
> 修正自 iteration/20260507-1/REQUIREMENT.md
+ 插件的推送二进制程序通过 `meta-list` 获取 `--callback` 参数的应用程序绝对路径，并执行 `init` 命令，使用参数 `message` 带上原消息报文
    + 改动点：从send改为init
+ 执行init命令前先使用插件的command检查是否支持
> 修正自 iteration/20260521-1/REQUIREMENT.md
+ 修改每30秒扫描一次add-request的消息，立即转换为任务明细（task_detail）
+ 修改将add-request待处理消息的桥接逻辑修改为每30秒扫描一次，且对命中的可处理文本消息无需等待10分钟老化、应在扫描命中后立即转换为task_detail，仅无文本内容的消息继续按过期规则处理。

#### 任务完成回复
> 新增自 iteration/20260503-17/REQUIREMENT.md
+ 定时任务检查最近24小时已完成且类型非cron的任务明细
+ 通过META_ID找到原始add-request，使用对应插件send命令回复任务响应
+ 先检查插件是否支持send（通过--help），不支持则记录日志
+ 发送成功后标记三方消息为已回复，早于该消息的已启动消息标记已完成
+ 回推必须按META_ID精确定位原始add-request
> 修正自 iteration/20260507-2/REQUIREMENT.md
+ 使用插件（plugin）回复（send）任务明细响应前先通过插件command检查是否支持

### Cron_Type元数据
> 修正自 iteration/20260503-18/REQUIREMENT.md
+ 备忘录任务明细执行时在请求报文中附加cron_type metadata：cron类型为"cron"，插件类型为插件key

### 备忘录明细类型列
> 补充自 iteration/20260503-13/REQUIREMENT.md
+ 备忘录明细列表悬停浮动层增加类型列


### /skills_warning 接口
> 新增自 ./iteration/20260512-1/REQUIREMENT.md
+ 新增 `/skills_warning` 接口，获取当前解析错误的SKILLS信息
+ Skills需求：../skills/iteration/20260512-1/REQUIREMENT.md
+ skills目录为--agent-dir指定目录下的skills

### Git路径实时查询 + /install_app 接口
> 新增自 ./iteration/20260512-2/REQUIREMENT.md
+ 转发/v1/chat/completions及cli/get和cli/pub提交metadata的git路径每次实时获取，不要缓存
+ Agent需求：../agent/iteration/20260512-1/REQUIREMENT.md
+ 新增 `/install_app` 接口，返回 `[string]`
+ 如果git没有安装则在返回中添加一个元素"git"

### 日志记录
> 新增自 ./iteration/20260513-1/REQUIREMENT.md
+ 转发/v1/chat/completions及cli/get和cli/pub需要记录日志，包括AgentID、ChatID（会话ID）、内容、类型、时间
+ 类型定义：
    + 0：/v1/chat/completions请求
    + 1：/v1/chat/completions的SSE响应
    + 2：cli/get
    + 3：cli/pub
+ /v1/chat/completions的SSE响应保持原纪录方式，响应一段记录一段，可能存在多段
+ cli/pub需要记录GZIP+Base64前的原始执行结果作为日志
+ cli/get没有需要执行的任务时（content为null或空时），不记录日志
+ 日志均采用异步，不要影响渲染和执行速度
+ 如果原逻辑有日志记录则合并到该需求
+ 如果原逻辑有查询需求则适配到该需求（注意查询类型）
+ 使用文件名为data的sqlite存储，使用连接池，索引采用AgentID+ChatID+类型+时间

### /log_skill 接口
> 新增自 ./iteration/20260513-2/REQUIREMENT.md
+ 新增 `/log_skill?agentId=xxx&chatId=yyy&round=zzz&start=aaa&close=bbb` 接口
+ 获取指定Agent和Chat（会话）最近round次的会话和CLI数据，写入指定AgentID的tmp目录，返回文件系统绝对路径和大小（K）
+ 查询条件（AND条件）：
    + round：最近N轮对话，以每次/v1/chat/completions请求为标记点，默认1
    + start：日志开始时间（yyyy-MM-dd hh:mm:ss），默认空
    + close：日志结束时间（yyyy-MM-dd hh:mm:ss），默认空
    + round和start至少有一个
+ 文件名称：agentId+chatId+时间戳组合，如 agenta_chatb_20260513121009.md
+ 文件格式：Markdown表格，列：时间|可读类型|具体内容

### /log_skill_status 接口
> 新增自 ./iteration/20260513-3/REQUIREMENT.md
+ 新增 `/log_skill_status?agentId=xxx&chatId=yyy` 接口
+ 检查日志数据库，判断指定Agent和Chat（会话）最近一轮会话是否触发过CLI/PUB命令执行流程

### Metadata透传
> 新增自 ./iteration/20260516-2/REQUIREMENT.md，./iteration/20260515-1/REQUIREMENT.md
+ /v1/chat/completions增加metadata参数支持
+ 请求中的metadata在转发/v1/chat/completions时附带传递

### Skills兼容数组compatibility
> 新增自 ./iteration/20260515-2/REQUIREMENT.md
+ Agent的skill属性需要兼容数组形式的compatibility属性
+ Skill需求：../skills/iteration/20260515-1/REQUIREMENT.md
+ Agent需求：../agent/iteration/20260515-1/REQUIREMENT.md

### Skills Warning接口
> 新增自 ./iteration/20260512-1/REQUIREMENT.md
+ 新增 `/skills_warning` HTTP GET 接口，返回当前解析错误的 Skills 信息
+ Skills需求：../skills/iteration/20260511-1/REQUIREMENT.md
+ skills目录为 `--agent-dir` 指定目录下的 skills

### Git路径实时获取
> 新增自 ./iteration/20260512-2/REQUIREMENT.md
+ 转发 `/v1/chat/completions`、`/cli/get`、`/cli/pub` 提交 metadata 的 git 路径每次实时获取，不要缓存
+ Agent需求：../agent/iteration/20260512-1/REQUIREMENT.md
+ 新增 `/install_app` HTTP GET 接口，返回 `[string]`
+ 如果 git 未安装则返回 `[{"git"}]`

### 日志记录
> 新增自 ./iteration/20260513-1/REQUIREMENT.md
+ 转发 `/v1/chat/completions`、`/cli/get`、`/cli/pub` 需要记录日志
+ 日志包含：AgentID、ChatID（会话ID）、内容、类型、时间
+ 类型定义：
    + 0：`/v1/chat/completions` 请求
    + 1：`/v1/chat/completions` 的 SSE 响应
    + 2：`cli/get`
    + 3：`cli/pub`
+ `/v1/chat/completions` SSE 响应保持原记录方式，响应一段记录一段，可能存在多段
+ `cli/pub` 需要记录 GZIP+Base64 前的原始执行结果作为日志
+ `cli/get` 没有需要执行的任务时（content 为 null 或空），不记录日志
+ 日志均采用异步，不要影响渲染和执行速度
+ 如果原逻辑有日志记录，则合并到该需求
+ 如果原逻辑有查询需求，则适配到该需求（注意查询类型）
+ 使用文件名为 data 的 sqlite 存储，使用连接池
+ 索引采用 AgentID + Chat ID + 类型 + 时间

### 日志查询导出
> 新增自 ./iteration/20260513-2/REQUIREMENT.md
+ 新增 `/log_skill?agentId=xxx&chatId=yyy&round=zzz&start=aaa&close=bbb` HTTP GET 接口
+ 获取指定 Agent 和 Chat（会话）最近 round 次的会话和 CLI 数据
+ 写入指定 AgentID 的 tmp 目录，返回文件系统绝对路径和大小（K）
+ 查询条件（AND 条件）：
    + round：最近 N 轮对话，以每次 `/v1/chat/completions` 请求为标记点，默认为 1
    + start：日志开始时间（yyyy-MM-dd hh:mm:ss），默认为空
    + close：日志结束时间（yyyy-MM-dd hh:mm:ss），默认为空
    + round 和 start 至少有一个
+ 文件名称：agentId+chatId+时间戳组合，如 `agenta_chatb_20260513121009.md`
+ 文件路径：存放在指定 AgentID 的 tmp 目录
+ 文件格式：Markdown 表格
+ SSE 响应合并为一段（到下一个 CLI/GET 或 CLI/PUB 出现为止）
+ CLI/GET：解析 content 后提取 cmd 属性
+ CLI/PUB：仅需要 content

### 日志状态检查
> 新增自 ./iteration/20260513-3/REQUIREMENT.md
+ 新增 `/log_skill_status?agentId=xxx&chatId=yyy` HTTP GET 接口
+ 检查日志数据库，判断指定 Agent 和 Chat 最近一轮（round）是否触发过 CLI/PUB 命令执行流程

### Metadata透传
> 新增自 ./iteration/20260516-2/REQUIREMENT.md
+ `/v1/chat/completions` 请求中增加 metadata 参数，转发时附带在 metadata 中
+ 案例：请求时 metadata 为 `{"hello":"world"}`，转发时附加到 metadata

### Metadata透传（重复合并）
> 新增自 ./iteration/20260515-1/REQUIREMENT.md
+ 同 20260513-4：`/v1/chat/completions` 请求中增加 metadata 参数并透传

### Skills兼容数组compatibility
> 新增自 ./iteration/20260515-2/REQUIREMENT.md
+ Agent 的 skill 属性需要兼容数组形式的 compatibility 属性
+ Skill 需求：../skills/iteration/20260515-1/REQUIREMENT.md
+ Agent 需求：../agent/iteration/20260515-1/REQUIREMENT.md


### 新建Agent默认配置
> 新增自 ./iteration/20260524-2/REQUIREMENT.md
+ 新建 Agent 后，将由参数 --default-dir 指定的目录内容复制到 Agent 目录，默认为应用启动程序所在目录下的 config 目录

### 创建任务全链路router_disable
> 新增自 ./iteration/20260524-3/REQUIREMENT.md
+ HTTP POST `/api/cron/create?agentId=xxx`：创建备忘录元数据时增加 router_disable 参数（boolean，默认true关闭）
+ HTTP POST `/api/cron/detail/metadata?agentId=xxx`：查询元数据时也需要返回 router_disable 参数
+ 备忘录任务创建时，右上角SWARM开关与实际转发/v1/chat/completions的metadata.router_disable必须全链路一致
    + 映射规则：开启SWARM时 router_disable=false，关闭SWARM时 router_disable=true
+ 周期任务：创建时必须保存到 task_meta.router_disable，创建出的任务明细也必须保存到 task_detail.router_disable，后续由 task_meta 自动拆分出的新明细必须继承所属元数据的 router_disable
+ 一次性任务：创建时任务明细必须保存 task_detail.router_disable，实际执行时以该条任务明细的 router_disable 作为最终转发值
+ 执行时 metadata.router_disable 的取值来源必须优先使用当前执行中的 task_detail.router_disable
+ 禁止在执行链路中丢失该字段，禁止回退为 Agent config.json 中的 router_disable
+ 禁止用旧字段 swarm 替代最终执行字段

### 插件配置router_disable
> 新增自 ./iteration/20260524-4/REQUIREMENT.md
+ HTTP `/api/plugins/config` 增加 router_disable 参数传入，默认 true（关闭）
+ HTTP `/api/plugins/meta` 中返回每个插件的 router_disable 参数
+ 在对应存储 `connect_meta` 中增加对应字段
+ 在指定插件 add-request 转换为备忘录明细时，需要传递 router_disable 参数

### 蜂群开关参数名变更
> 新增自 ./iteration/20260524-5/REQUIREMENT.md
+ 蜂群开关参数 swarm 改为 router_disable，类型不变，语意相反（router_disable=true 表示关闭）
+ HTTP /api/edit 中 swarm 改为 router_disable，默认为 true，语意相反

### 启动自动补齐Agent配置
> 新增自 ./iteration/20260525-1/REQUIREMENT.md
+ 启动时如果 --agent-dir 指向空目录，自动补齐 DEF_AGENT 时需要复制由参数 --default-dir 指定的目录内容到 DEF_AGENT 目录，默认为应用启动程序所在目录下的 config 目录（同创建Agent流程一致）

### Token消费记录
> 新增自 ./iteration/20260527-1/REQUIREMENT.md
+ 新增命令 token，记录 Token 消费明细，保存至数据库：
    + thinking：int，思考耗费的 Token
    + input：int，输入耗费的 Token
    + total：int，总耗费的 Token
    + cache：int，缓存耗费的 Token
    + model：string，对应的模型
    + agentId：string，对应的 agent
    + function：string，用途
    + 记录产生时间的时间戳：int64，自动产生，使用当前时间
    + 索引为时间戳 + AgentId
+ 新增 /api/consume?agentId=xxx&starTime=yyy&closeTime=zzz&limit=aaa 获取指定Agent在指定时间范围内的Token消费数据
    + starTime 和 closeTime 格式为 yyyyMMdd-hhmmss，需转换为时间戳（必填）
    + agentId 为选填，不填则查询所有 Agent
    + limit 为必填，默认 500，最多查询条数
    + 返回结果：第一部分为所有明细，第二部分为按模型聚合的 thinking、input、total、cache

> 新增自 ./iteration/20260614-2/REQUIREMENT.md
+ 增加token get命令，获取最近N条，或指定时间段的token数据
    + 读取的数据为token在本地数据库存储的用量数据
    + 数据查询方式需要与接口/api/consume相同
+ 案例
``` 使用integration代理proxy，查询最新500条
integration token --n 500
```
``` 使用integration代理proxy，查询2026-06-14 12:00:00至2026-06-14 14:00:00最新500条
integration token --n 500 --start "2026-06-14 12:00:00" --close "2026-06-14 14:00:00"
```
+ 不能破坏现有token命令，如下
``` 使用integration代理proxy
integration token
integration token --provider deepseek
integration token --agentId demo-agent --model deepseek-chat --function cli/get --thinking 10 --input 20 --total 30 --cache 5
```
+ 增加--help
```
integration token get --help
```

### 统一变量名和转发报文
> 新增自 ./iteration/20260528-1/REQUIREMENT.md
+ 统一变量名：
    + Site 居中输入框、备忘录元数据、插件配置的蜂群开关、HTML开关、思考模式开关、模型选择
    + 蜂群（Swarm）开关对应参数名 router_disable，开关打开时 router_disable=false，关闭时 router_disable=true
    + 思考模式对应参数名 thinking
    + HTML开关对应参数名 html
    + 模型选择对应参数名 model
+ 转发 /v1/chat/completions 时统一报文格式，metadata 中包含 router_disable、thinking、html 和 agents 数组
+ 转发 /cli/get 时统一报文格式，metadata 的 agents 中包含 router_disable、thinking、provider
+ Site 或定时任务触发后按统一报文格式转发至 --host 指定的服务，只有一套协议，不需要老逻辑兼容
+ 所有功能均使用由 --port 指定的端口，默认为 8080

### 获取DeviceId
> 新增自 ./iteration/20260530-1/REQUIREMENT.md
+ 新增 /api/deviceId 获取 DeviceID

### /install_app 增加python3检测
> 新增自 ./iteration/20260530-2/REQUIREMENT.md
+ 修改 /install_app 接口，如果 python3 没有安装则在返回中添加一个元素 "python3"
+ 兼容现有 /install_app 的取值逻辑，仅新增

> 新增自 ./iteration/20260614-1/REQUIREMENT.md
+ 修改接口/install_app，从主应用的config.json中的"install_app"读取数据，需要区分操作系统：
```
{
    ...
    "install_app": {
        "linux": [...],
        "wsl": [...],
        "mac": [...],
    }
}
```
    + 其中Linux系统使用linux，Mac系统使用mac，Windows（WSL）使用wsl
    + 数据结构不变，依旧是string array
    + --install_app参数不变，如果存在所有操作系统结构都要追加
+ 所有install_app的元素表示一个本地应用名称，需要检查是否已安装，已安装则从返回列表中删除
    + 不同操作系统判断方式不同
    + 接口缓存5分钟

### 插件文件类型识别
> 新增自 ./iteration/20260603-1/REQUIREMENT.md
+ 接口/api/plugins/meta判断plugins目录文件是否为插件的条件：
    + 没有后缀名的应用程序，后缀名为py、js或go的脚本文件
    + 跳过目录
+ 如果判断出现错误，不要报错或崩溃，仅跳过该文件并输出日志

### 插件远程执行接口
> 新增自 ./iteration/20260606-1/REQUIREMENT.md
+ 新增/api/plugins/exec?key=x&command=y&param1=value1&param2=value2&...执行指定插件的指定命令
    + key：插件标识、command：该插件的命令
    + param1=value1&param2=value2&...：可以有任意组，表示插件参数
    ``` 案例：key=browser&command=instance init&agentId=a1&chatId=b1
    browser instance init --agentId a1 --chatId b1
    ```
        + 需要注意空格转义
+ /api/plugins/exec等待插件执行命令超时等待改为由integration/proxy参数--plugin_exec_timeout决定的毫秒数，默认600秒，如果超时或启动失败需要在integration.log保留日志
+ 如果插件完成了则立即返回

### 沙盒模式控制
> 新增自 ./iteration/20260607-1/REQUIREMENT.md
+ 新增/api/sandbox=true，变更内部cli/get使用的sandbox变量，该变量默认值由--sandbox参数控制，默认为false
> 新增自 ./iteration/20260608-1/REQUIREMENT.md
+ 新增/api/sandbox=枚举值?agentId=x&chatId=y，为指定AgentID+ChatID开启或关闭沙盒模式，关闭则删除对应Agent+Chat的数据
+ 新增/api/sandbox_status?agentId=x&chatId=y，获取指定AgentID+ChatID的沙盒模式
+ 修改/api/cmd也需要参与沙盒执行判断
+ 沙盒模式需求：../cli-get/iteration/20260609-1/REQUIREMENT.md
+ 不需要兼容老逻辑，删除兼容代码，以最新需求编写

### Agent元数据扩展
> 新增自 ./iteration/20260610-1/REQUIREMENT.md
+ 转发/v1/chat/completions时需要带上metadata.agent.sandbox和metadata.agent.version
+ Agent需求：../agent/iteration/20260610-1/REQUIREMENT.md

### 插件参数结构调整
> 新增自 ./iteration/20260610-2/REQUIREMENT.md
+ 修改/api/plugins/meta中每个插件的param的参数结构为[{"key":"val"},{"key":"val"},...]
+ 插件规范：../connect/PLUGIN.md
    + 浏览器插件格式：../connect/browser/iteration/20260610-1/REQUIREMENT.md
    + 邮件插件格式：../connect/email/iteration/20260610-1/REQUIREMENT.md
    + 飞书插件格式：../connect/feishu/iteration/20260610-1/REQUIREMENT.md

### 技能动态注入
> 新增自 ./iteration/20260610-3/REQUIREMENT.md
+ 修改/api/skills?agentId=xxx，在返回结果中加入：__internal_cron（无论如何都增加）
+ 如果开启了browser插件则增加：__internal_browser
+ 如果开启了remote插件则增加：__internal_remote

> 新增自 ./iteration/20260614-3/REQUIREMENT.md
+ 修改/api/skills?agentId=xxx：
    + 如果开启了browser插件（需要监测是否开启状态）则增加：__internal_browser
    + 如果开启了remote插件（需要监测是否开启状态）则增加：__internal_remote
    + 同时从主应用的config.json的将skills数据（string array）追加到结果
    + 原本__internal_cron的会改为从config.json读取
```
{
    ...
    "skills": [
        "__internal_cron",
        ...
    ]
}
```
+ 原需求：./iteration/20260610-3/REQUIREMENT.md
+ 不需要兼容，改为最新

> 新增自 ./iteration/20260618-1/REQUIREMENT.md
+ 修改/api/skills?agentId=xxx：
    + 如果开启了browser插件（需要监测是否开启状态）则增加：__internal_browser
    + 如果开启了remote插件（需要监测是否开启状态）则增加：__internal_remote
    + 同时从主应用的config.json的将skills数据（string array）追加到结果
    + 原本__internal_cron的会改为从config.json读取
```
{
    ...
    "skills": [
        "__internal_cron",
        ...
    ]
}
```
+ 原需求：./iteration/20260610-3/REQUIREMENT.md
+ 不需要兼容，改为最新

### 蜂群Agent查询
> 新增自 ./iteration/20260611-1/REQUIREMENT.md
+ 新增/api/swarm_agent，获取当前启动了蜂群的Agent名称（router_disable=false）
+ Agent名单中不能包含当前Agent
+ Agent需求：../agent/iteration/20260524-1/REQUIREMENT.md

### 编写代码
+ 以Golang编写以上代码，要求：
    + 所有API请求必须使用相对路径(如/v1/chat/completions), 禁止硬编码IP或域名, 确保非本地访问时请求自动指向当前Host
    + SSE流式代理时，必须设置Content-Type:text/event-stream，同时Read Buffer要小，避免多个SSE行被攒成一次写入
    + 代码简洁，包体积越小越好
    + 能用开源包的就用开源包
    + 所有Cron相关HTTP接口和定时任务必须复用启动时初始化的全局数据库连接，禁止每次请求单独打开和关闭数据库文件
    + sqlite使用连接池，避免每次都新建连接
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 验证测试
+ 使用mock数据，验证代码生产内容是否符合如下CURL：
###### 接收请求/v1/chat/completions
```
curl --location 'http://a/cli/get' \
--header 'Content-Type: application/json' \
--header 'Authorization: xxxxx' \
--data-raw '{
    "model": ,
    "messages": [
        {
            "role": "user",
            "content": "HELLO WORLD"
        }
    ],
    "stream": true
}'
```
+ 请求头Authorization不为空，参数model不为空，role=user，content不为空，stream=true
###### 转发请求/v1/chat/completions
```
curl --location 'http://b/cli/get' \
--header 'Content-Type: application/json' \
--header 'Authorization: xxxxx' \
--data-raw '{
    "model": ,
    "messages": [
        {
            "role": "user",
            "content": "HELLO WORLD"
        }
    ],
    "stream": true,
    "metadata": {xxx},
}'
```
+ 请求头未修改，参数model，role=user，content未修改，stream=true
+ 参数metadata符合追加要求
###### 转发响应/v1/chat/completions
```
data: {"id":"cmpl-1305b94c570f447fbde3180560736287","object":"chat.completion.chunk","created":1698999575,"model":"kimi-k2.5","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}
data: {"id":"cmpl-1305b94c570f447fbde3180560736287","object":"chat.completion.chunk","created":1698999575,"model":"kimi-k2.5","choices":[{"index":0,"delta":{"content":"你好"},"finish_reason":null}]}
data: {"id":"cmpl-1305b94c570f447fbde3180560736287","object":"chat.completion.chunk","created":1698999575,"model":"kimi-k2.5","choices":[{"index":0,"delta":{"content":"。"},"finish_reason":null}]}
data: {"id":"cmpl-1305b94c570f447fbde3180560736287","object":"chat.completion.chunk","created":1698999575,"model":"kimi-k2.5","choices":[{"index":0,"delta":{},"finish_reason":"stop","usage":{"prompt_tokens":19,"completion_tokens":13,"total_tokens":32}}]}
data: [DONE]
```
+ 响应内容为`你好。`，不改写
+ 收到一段转发一段，不要聚合
> 新增自 iteration/20260518-1/REQUIREMENT.md
+ add-request命令新增可选参数--schema（Json String）
> 新增自 iteration/20260518-2/REQUIREMENT.md
+ 定时器执行明细时有response_schema则转发时附加到metadata.response_schema
> 新增自 iteration/20260518-3/REQUIREMENT.md
+ 通过META_ID找到add-request后用插件send回复前对SSE响应进行JSON标准化
+ 检查是否为```json...```或```...```格式，去掉Markdown标记
+ 标准化失败则使用原始响应
> 新增自 iteration/20260519-1/REQUIREMENT.md
+ 插件日志统一路径为release/plugins/插件名.log，通过/api/plugins/log读取
+ 日志文件不存在时返回明确错误
> 新增自 iteration/20260519-2/REQUIREMENT.md
+ /api/token增加__url、__model、__model_fast、__model_thinking、__model_multi_input、__model_multi_output
+ token命令返回增加以上属性
> 新增自 iteration/20260520-1/REQUIREMENT.md
+ 转发时如模型配置了__url、__model等属性则加入metadata
> 新增自 iteration/20260520-2/REQUIREMENT.md
+ /api/config支持删除指定模型
> 新增自 iteration/20260518-1/REQUIREMENT.md
+ add-request命令新增可选参数--schema（Json String），对应任务明细的response_schema
> 新增自 iteration/20260518-2/REQUIREMENT.md
+ 定时器执行任务明细时，如有response_schema属性，则在转发/v1/chat/completions时附加到metadata.response_schema
> 新增自 iteration/20260518-3/REQUIREMENT.md
+ 通过META_ID找到原始add-request后，使用插件send回复前对SSE响应进行JSON标准化处理
+ 检查SSE响应是否为```json...```或```...```格式，如果是则去掉Markdown格式标记
+ 标准化失败则使用原始SSE响应
> 新增自 iteration/20260519-1/REQUIREMENT.md
+ 插件日志统一路径为release/plugins/插件名.log，通过/api/plugins/log?name=插件名读取
+ 日志文件不存在时返回明确错误信息
> 新增自 iteration/20260519-2/REQUIREMENT.md
+ /api/token接口增加__url、__model、__model_fast、__model_thinking、__model_multi_input、__model_multi_output属性（可为空）并持久化
+ token命令返回增加以上属性
> 新增自 iteration/20260520-1/REQUIREMENT.md
+ 转发/v1/chat/completions时，如当前模型配置了__url、__model等属性，则加入metadata中传递
> 新增自 iteration/20260520-2/REQUIREMENT.md
+ /api/config支持删除指定模型
> 新增自 iteration/20260518-1/REQUIREMENT.md
+ add-request命令新增可选参数--schema（Json String），对应任务明细的response_schema
> 新增自 iteration/20260518-2/REQUIREMENT.md
+ 定时器执行任务明细时，如有response_schema属性，则在转发/v1/chat/completions时附加到metadata.response_schema
> 新增自 iteration/20260518-3/REQUIREMENT.md
+ 通过META_ID找到原始add-request后，使用插件send回复前对SSE响应进行JSON标准化处理
+ 检查SSE响应是否为```json...```或```...```格式，如果是则去掉Markdown格式标记
+ 标准化失败则使用原始SSE响应
> 新增自 iteration/20260519-1/REQUIREMENT.md
+ 插件日志统一路径为release/plugins/插件名.log，通过/api/plugins/log?name=插件名读取
+ 日志文件不存在时返回明确错误信息
> 新增自 iteration/20260519-2/REQUIREMENT.md
+ /api/token接口增加__url、__model、__model_fast、__model_thinking、__model_multi_input、__model_multi_output属性（可为空）并持久化
+ token命令返回增加以上属性
> 新增自 iteration/20260520-1/REQUIREMENT.md
+ 转发/v1/chat/completions时，如当前模型配置了__url、__model等属性，则加入metadata中传递
> 新增自 iteration/20260520-2/REQUIREMENT.md
+ /api/config支持删除指定模型
> 新增自 iteration/20260518-1/REQUIREMENT.md
+ add-request命令新增可选参数--schema（Json String），对应任务明细的response_schema
> 新增自 iteration/20260518-2/REQUIREMENT.md
+ 定时器执行任务明细时，如有response_schema属性，则在转发/v1/chat/completions时附加到metadata.response_schema
> 新增自 iteration/20260518-3/REQUIREMENT.md
+ 通过META_ID找到原始add-request后，使用插件send回复前对SSE响应进行JSON标准化处理
+ 检查SSE响应是否为```json...```或```...```格式，如果是则去掉Markdown格式标记
+ 标准化失败则使用原始SSE响应
> 新增自 iteration/20260519-1/REQUIREMENT.md
+ 插件日志统一路径为release/plugins/插件名.log，通过/api/plugins/log?name=插件名读取
+ 日志文件不存在时返回明确错误信息
> 新增自 iteration/20260519-2/REQUIREMENT.md
+ /api/token接口增加__url、__model、__model_fast、__model_thinking、__model_multi_input、__model_multi_output属性（可为空）并持久化
+ token命令返回增加以上属性
> 新增自 iteration/20260520-1/REQUIREMENT.md
+ 转发/v1/chat/completions时，如当前模型配置了__url、__model等属性，则加入metadata中传递
> 新增自 iteration/20260520-2/REQUIREMENT.md
+ /api/config支持删除指定模型




































### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 相关迭代：iteration/日期/REQUIREMENT.md
+ 同步代码：../integration/REQUIREMENT.md（每次都要同步更新代码）
> 合并截止：./iteration/20260618-1/REQUIREMENT.md，下次合并从此之后的新迭代开始
