### 第一性原则
+ 仅可以新增/更新/删除当前需求文档（REQUIREMENT.md）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../DESIGN.md
+ 本模块设计文档：DESIGN.md

### 需求介绍
+ 将Agent元数据以心跳的形式上报，并获取任务在执行后回传（pub）
+ cli/get执行的命令必须注册到活跃命令列表（开始时注册、结束时注销），确保/api/kill能定位并终止正在执行的命令 > 引用自 ./iteration/20260501-1/REQUIREMENT.md
+ 命令被/api/kill终止时需捕捉异常，返回"命令被终止"而非等待超时无响应 > 引用自 ./iteration/20260501-1/REQUIREMENT.md

### Agent元数据
+ Agent元数据介绍：../agent/REQUIREMENT.md
+ Agent元数据手册：../agent/USER_GUIDE.md
+ Agent元数据迭代：../agent/iteration/日期/REQUIREMENT.md

### 沙箱设计
+ 沙箱设计：sandbox/REQUIREMENT.md
+ 沙箱手册：sandbox/REQUIREMENT.md
+ 沙箱迭代：sandbox/iteration/日期/REQUIREMENT.md

### 上报心跳
+ 参考如下CURL命令的结构，上报Agent元数据
```
curl --location 'http://xxx/cli/get' \
--header 'Content-Type: application/json' \
--data-raw '{
    "model": "",
    "messages": [
        {
            "role": "user",
            "content": ""
        }
    ],
    "metadata": Agent元数据
}'
```
###### 参数解释
+ 请求的URL（含端口）为必填，从命令行参数--host获取，默认为https://www.deepright.cn
+ 参数metadata为Agent元数据（agent/REQUIREMENT.md），包含：deviceId（string）、terminal（string）、gateway（string）、sys（string）和agents（array）
+ 参数model固定为""
> 新增自 iteration/20260509-1/REQUIREMENT.md
+ metadata Agent 元数据需加上 plugins 信息
> 新增自 iteration/20260509-2/REQUIREMENT.md
+ 导出 updatePluginsMetadata 方法，供外部模块获取 plugins metadata 并注入到 metadata 中

###### HTTP配置
+ 使用连接池，固定开启TCP NoDelay，TCP Keepalive，从命令行参数--idle_timeout获取空闲探测，默认90秒
+ 整体超时从命令行参数--http_timeout获取（建连+发送+读响应体总上限），默认60000毫秒
+ 连接超时从命令行参数--http_connect_timeout获取，默认15000毫秒
+ 读取超时从命令行参数--http_socket_timeout获取，默认45000毫秒

###### 心跳响应
+ 心跳响应可能包含一个需要执行的任务，也可能为空
``` 响应报文案例
{
  "code": 200,
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": "{\"subOps\":{\"exempted\":false,\"app\":[\"ls\"],\"w\":[],\"r\":[\"/home/user/project\"]},\"conversation\":\"conv_001\",\"workspace\":\"/home/user/project\",\"suffix\":\"cmd\",\"router\":\"device123node1\",\"chat\":\"chat_001\",\"type\":\"cmd\",\"tid\":\"task_20260328_001\",\"why\":\"列出项目目录文件\",\"cmd\":\"ls -la /home/user/project\",\"ddl\":1743177600000}"
      }
    }
  ]
}
```
+ 当HTTP Status Code和报文的code都为200时表示成功，否则表示上报异常
+ 响应中除属性`content`外均为固定内容，content是一个json string或null（不存在），schema为：
``` JSON SCHEMA
{
    "type": "object",
    "properties": {
    "subOps": {
        "exempted": {
            "type": boolean,
            "description": "是否豁免"
        }
    },
    "agentId": {
        "type": string,
        "description": "AgentID"
    },
    "chat": {
        "type": string,
        "description": "ChatID，会话ID"
    },
    "timeout": {
        "type": integer,
        "description": "毫秒表示的，执行任务的超时"
    },
    "suffix": {
        "type": string,
        "description": "无需理解，pub回传时原样带上"
    },
    "type": {
        "type": string,
        "description": "无需理解，pub回传时原样带上"
    },
    "ddl": {
        "type": long,
        "description": "执行超时时间戳，Dead line"
    },
    "tid": {
        "type": string,
        "description": "任务 ID"
    },
    "cmd": {
        "type": "string",
        "description": "需要执行的完整 Shell 命令"
    }
  }
}
```
+ 属性content为null或空时表示当前没有待执行任务，进入下一次心跳上报

### 执行任务
+ 属性content解析json string后的属性cmd，即本次需要执行的任务，启动与当前进程同环境（Shell）的子进程，执行对应系统命令
+ 执行任务的子进程需要指定超时，取值为属性timeout，如果服务端未返回则默认180秒
+ 系统命令需要支持 && ｜等管道符，支持绝对路径、相对路径和~路径
+ 命令被/api/kill终止时需捕捉异常信号，返回status=1且cmd内容为"命令被终止" > 引用自 ./iteration/20260501-1/REQUIREMENT.md
+ 将执行后的结果整理为如下JSON格式的请求报文
``` JSON SCHEMA
{
    "type": "object",
    "required": ["status", "suffix", "type", "cmd", "tid"],
    "properties": {
    "status": {
        "type": "integer",
        "description": "执行状态，0=成功，1=失败",
        "enum": [0, 1]
    },
    "agentId": {
       "type": "string",
       "description": "AgentId，原样带上"
    },
    "suffix": {
        "type": "string",
        "description": "原样带上"
    },
    "chat": {
        "type": "string",
        "description": "会话ID，原样带上"
    },
    "type": {
        "type": "string",
        "description": "原样带上"
    },
    "tid": {
        "type": "string",
        "description": "原样带上"
    },
    "cmd": {
        "type": "string",
        "description": "命令执行结果内容，需要GZIP+Base64"
    },
  }
}
```
+ 其中cmd作为执行结果，需要进行GZIP+Base64

### 回传结果
+ 参考如下CURL命令的结构，提交任务结果
```
curl --location 'http://127.0.0.1:9998/cli/pub' \
--header 'Content-Type: application/json' \
--data-raw '{
    "model": "",
    "messages": [
        {
            "role": "user",
            "content": JSON String
        }
    ],
    "metadata": Agent元数据
}'
```
###### 接口解释
+ 请求的URL（含端口）和参数`model`同上报心跳的获取方式
+ 参数messages.content，为执行结果的JSON string
+ 参数metadata同cli/get
###### 等待响应
``` 响应报文结构案例
{
  "code": 200,
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": null
      }
    }
  ]
}
```
###### 响应解释
+ 当HTTP Status Code和报文的code都为200时表示成功，否则表示提交异常

### 链路整理
+ 上报心跳
    + 无任务，进入下一次心跳上报
    + 有任务，执行任务并提交结果
        + 进入下一次心跳上报
+ 任何异常，均进入下一次心跳上报


### 日志记录
> 新增自 ./iteration/20260513-1/REQUIREMENT.md
+ cli/get和cli/pub需要记录日志，包括AgentID、ChatID（会话ID）、内容、类型、时间
+ 类型定义：
    + 0：/v1/chat/completions请求
    + 1：/v1/chat/completions的SSE响应
    + 2：cli/get
    + 3：cli/pub
+ 与Proxy需求采用同表结构：../proxy/iteration/20260513-1/REQUIREMENT.md
+ 日志均采用异步，不要影响渲染和执行速度
+ 如果原逻辑有日志记录则合并到该需求
+ 如果原逻辑有查询需求则适配到该需求（注意查询类型）
+ 使用文件名为data的sqlite存储，使用连接池避免每次新建连接

### 同步代码
+ ../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 沙盒执行
> 新增自 ./iteration/20260607-1/REQUIREMENT.md
+ 为每个Agent+Chat保存一个sandbox_exe的枚举值，默认为空（无值）
    + key：filepick — 用户选择目录（没有选择均认为没有权限）
    + key：net — 关闭网络
    + key：filepick_net — 两者都限
> 新增自 ./iteration/20260609-1/REQUIREMENT.md
+ 如果Agent+Chat的sandbox_exe存在枚举值且为以上3个，则cli/get获取待执行命令后通过指定key的CLI_SANDBOX执行命令→cli/pub提交
    + cli/get只在有待处理任务时才需要执行沙盒
+ 如果Agent+Chat的sandbox_exe不存在枚举值或不为以上3个，则cli/get和cli/pub保持原逻辑
+ 沙盒应用程序相对于主应用程序路径由--sandbox_app指定（或从config/config.json读取）
    + 格式：`$sandbox_app -cmd "cat hello.txt | wc -l"`
+ MAC Sandbox执行逻辑：.app/Contents/MacOS/CLI_SANDBOX，不要执行源码版或普通二进制
+ MAC沙盒需求：../../sandbox/mac/iteration/20260609-1/REQUIREMENT.md
+ 如果cli/get响应报文开启了豁免（subOps.exempted=true）则不使用沙盒
    + 响应JSON SCHEMA中新增 `"subOps":{"exempted": boolean}`

### 编写代码
+ 以Golang编写以上代码，要求：
    + 启动一根master线程上报cli/get，如果有待执行任务则交由命令行参数--thread指定的work线程池执行（默认为20）
    + 如果cli/get没有待执行任务或任何异常，则休眠由命令行参数--sleep指定的毫秒时间（默认为3000）
    + 如果cli/get返回待执行任务，转交对应work线程后立即进入下一次心跳上报
    + 代码简洁，包体积越小越好，能用开源包的就用开源包
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写
+ 命令执行前注册到全局活跃命令列表，执行完成后注销，确保/api/kill可查询到正在执行的命令 > 引用自 ./iteration/20260501-1/REQUIREMENT.md

### 验证测试
+ 使用mock数据，验证代码生产内容是否符合如下CURL：
###### cli/get
```
curl --location 'http://xxx/cli/get' \
--header 'Content-Type: application/json' \
--data-raw '{
    "model": "",
    "messages": [
        {
            "role": "user",
            "content": ""
        }
    ],
    "metadata": Agent元数据
}'
```
+ 参数model为空，role=user，content为空
+ 参数metadata符合如下格式：
```
{
    "timezone": string,
    "deviceId": string,
    "terminal": string,
    "gateway": string,
    "sys": string,
    "app": string,
    "agents": [
        {
            "workspace": string,
            "name": string,
            "soul": string,
            "user": string,
            "skills": [
                技能元数据列表
            ],
        }
    ]
}
```
###### cli/pub
```
curl --location 'http://127.0.0.1:9998/cli/pub' \
--header 'Content-Type: application/json' \
--data-raw '{
    "model": "",
    "messages": [
        {
            "role": "user",
            "content": JSON String
        }
    ],
    "metadata": Agent元数据
}'
```
+ 参数model为空，role=user，content为Json string
+ 参数content的Json string解析为JSON后符合：
``` JSON SCHEMA
{
    "type": "object",
    "required": ["status", "suffix", "type", "cmd", "tid"],
    "properties": {
    "status": {
        "type": "integer",
        "description": "执行状态，0=成功，1=失败",
        "enum": [0, 1]
    },
    "suffix": {
        "type": "string",
        "description": "原样带上"
    },
    "type": {
        "type": "string",
        "description": "原样带上"
    },
    "tid": {
        "type": "string",
        "description": "原样带上"
    },
    "cmd": {
        "type": "string",
        "description": "命令执行结果内容，需要GZIP+Base64"
    },
  }
}
```
+ 参数metadata同cli/get
+ 命令终止测试：mock被/api/kill终止的场景，验证返回status=1且cmd="命令被终止" > 引用自 ./iteration/20260501-1/REQUIREMENT.md

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 相关迭代：iteration/日期/REQUIREMENT.md
+ 同步代码：../integration/REQUIREMENT.md
> 合并截止：./iteration/20260609-1/REQUIREMENT.md，下次合并从此之后的新迭代开始
