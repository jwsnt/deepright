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
+ 新增/log_skill?agentId=xxx&&chatId=yyy&&round=zzz&&start=aaa&&close=bbb获取指定Agent和Chat（会话）最近round次的会话和CLI数据，写入指定AgentID的tmp的目录，并返回文件系统绝对路径和大小（K）
+ 会话和CLI数据：从日志获取，按`时间排序`的包括会话的请求、所有SSE响应、cli/get、cli/pub的数据
    + 日志需求：../20260513-1/REQUIREMENT.md
+ 查询条件:
    + round：指最近N轮的对话，以每次转发/v1/chat/completions的请求为标记点，默认为1（最近一轮）
    + start：日志开始时间（yyyy-MM-dd hh:mm:ss），默认为空
    + close：日志结束时间（yyyy-MM-dd hh:mm:ss），默认为空
    + 以上3个条件为AND条件，其中round和start至少有一个，start和close需要转换为日志数据对应时间戳
    + 案例：
        + agentId=A&&chatId=B&&round=1
            + 取agentId=A&&chatId=B，以当前时间为节点，最后一个/v1/chat/completions请求日志到当前时间结束到所有会话请求、SSE响应、CLI/get、CLI/pub日志
        + agentId=A&&chatId=B&&round=3
            + 取agentId=A&&chatId=B，以当前时间为节点，倒数第三个/v1/chat/completions请求日志到当前时间结束到所有会话请求、SSE响应、CLI/get、CLI/pub日志
        + agentId=A&&chatId=B&&start=2026-05-13 21:25:00&&close=2026-05-13 22:25:00
        + agentId=A&&chatId=B&&round=2&&start=2026-05-13 21:25:00&&close=2026-05-13 22:25:00
+ 文件名称：加agentId+chatId+时间戳组合。如agenta_chatb_20260513121009.md
+ 文件路径：存放在指定AgentID的tmp的目录
+ 文件格式：Markdown表格
    + ｜以yyyy-MM-dd hh:mm:ss表示的时间|可读类型:SSE请求、SSE响应、工具请求（cli/get）、工具响应（cli/pub）｜具体内容|
        + 其中cli/get和cli/pub是提示，实际类型为工具请求和工具响应
+ 不通类型的具体内容不通：
    + 对于SSE请求，仅需要CONTENT
    ``` 仅需要`看下我的桌面`
    {"model":"deepseek","messages":[{"role":"user","content":"看下我的桌面"}],"stream":true,"metadata":{"agentId":"A","chat":"c045489b-dad3-4bd4-95b8-9846e7102ec4","type":"page_session"}}
    ```
    + 对于SSE响应，需要将下个CLI/GET或CLI/PUB出现时所有报文合并为一段
    ``` 合并为`正在加载长期记忆\n正在读取知识库\n查看桌面文件列表\n`
    data: {"choices":[{"index":0,"metadata":{},"delta":{"content":"正在加载长期记忆\n","role":"assistant"}}],"workflow":"main","object":"chat.completion","model":"right","created":1778768094547,"code":200,"biz":"__main","id":"4a8112e2-0527-4280-8361-7bb681a9ac13"}
    data: {"choices":[{"index":0,"metadata":{},"delta":{"content":"正在读取知识库\n","role":"assistant"}}],"workflow":"main","object":"chat.completion","model":"right","created":1778768094547,"code":200,"biz":"__main","id":"4a8112e2-0527-4280-8361-7bb681a9ac13"}
    data: {"choices":[{"index":0,"metadata":{},"delta":{"content":"查看桌面文件列表\n","role":"assistant"}}],"workflow":"sub","object":"chat.completion","model":"right","created":1778768094547,"code":200,"biz":"__cli","id":"4a8112e2-0527-4280-8361-7bb681a9ac13"}
    插入CLI/GET或CLI/PUB报文
    ```
        + 处理完CLI/GET和CLI/PUB后如果继续出现SSE响应则再次重新合并
    + 对于CLI/GET，参考CLI/GET需求，解析CONTENT后提取其中的cmd属性
        + CLI/GET需求：../../../cli-get/REQUIREMENT.md
    + 对于CLI/PUB，同SSE请求，仅需要CONTENT

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
    + 使用文件名为data的sqlite存储，并使用连接池
        + 索引采用AgentID + Chat ID + 类型 + 时间
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



