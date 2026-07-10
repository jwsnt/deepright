#### config
+ 用于继承，基础的Rag和FunCall

``` 自定义Rag，用于加载CLI上报信息，并替换自身System Prompt中的#占位符（必须放第二位）@see CliRag
"global": {
    "file_system": "#file_system",
    "sandbox_path": "#sandbox_path",
    "workspace": "#workspace",
    "sys": "#sys"
},
"key": "rag_env"
```

``` 用于加载CLI上报信息，并替换自身System Prompt中的#占位符（必须放第二位）
{
    "global": {
        "file_system": "#file_system",
        "sandbox_path": "#sandbox_path",
        "workspace": "#workspace",
        "terminal": "#terminal",
        "memory": "#memory",
        "user": "#user",
        "soul": "#soul",
        "sys": "#sys"
    },
    "key": "rag_env"
},
```

+ `file_system`用于注入工作环境说明模板，普通会话使用`workspace_def.md`，沙箱会话使用`workspace_sandbox.md`
+ `sandbox_path`仅在沙箱会话存在时替换，普通会话可以在`global`中保留该占位符而不报错

``` 替换占位符#schema为每个配置的response_schema，一般用于强化约束输出
"replace": "#schema",
"key": "rag_schema"
```

``` 自定义Rag，加载Skills目录下的技能 @see SkillsSchemaRag
"key": "rag_skills"
```

``` 自定义Rag，禁止披露的内容 @see SafetyRag
"key": "rag_safety"
```

``` 自定义Rag，禁止披露的内容 @see RouterRag
"key": "rag_router"
```

``` 自定义Rag，加载长期记忆和提示词 @see MemoryRag
"key": "rag_memory"
```

``` 自定义Rag，提示Skills是否重加载 @see SKillsReloadRag
"key": "rag_skills_reload"
```

``` 替换占位符${}内容为配置文件信息（系统配置文件），需要放最后
"key": "rag_placeholder"
```

``` Debug模式下的检查 @See RequestChecker
"key": "rag_check"
```

###### FunCall
+ FunCall的具体描述和调用方法会在*.json对应配置中

``` 校验过程，模型关键词是[verify]，会转发到verify.json中的main配置
"prefix": "[verify]",
"name": "cli@verify"
```

``` 加载技能，模型关键词是[skills]，会转发到skills.json中的main配置
"prefix": "[skills]",
"name": "skills@main"
```

``` 图片生成，模型关键词是[image]，会转发到media.json中的image配置
"prefix": "[image]"
"name": "media@image"
```

``` 文件生成，模型关键词是[file]，会转发到media.json中的file配置
"prefix": "[file]",
"name": "media@file"
```

``` 启动任务，模型关键词是[task]，会转发到cli.json中的task配置
"prefix": "[task]",
"name": "task@main"
```

``` 调用终端，模型关键词是[cli]，会转发到cli.json中的sub配置
"prefix": "[cli]",
"name": "cli@sub"
```

``` 内容识别，模型关键词是[ocr]，会转发到media.json中的ocr配置
"prefix": "[ocr]",
"name": "media@ocr"
```

#### llm
+ 共享LLM配置，可以用占位符从系统配置替换
###### scene
+ 对应模型读取上下文的场景Key（用于Redis存储），从配置文件读取",
###### clientHistories
+ 客户端提交的上下文是否加入会话，如果加入就不会使用服务端上下文
###### recallOffset
+ 召回N毫秒内（前）的上下文，例如-900000表示15分钟内
###### recallNums
+ 召回N条上下文进会话，不配置就默认使用histories
###### storeCompleted
+ 请求和回答是否分开存储，为true时仅当获得Answer时才会原子化存储
###### storeFunCall
+ 保存FunCall过程进持久化上下文
###### histories
+ 持久化上下文的条数

###### memory
+ 清除持久化的上下文，下次会话生效 @see CustomHistoryAssistant
+ 继承base_llm的LLM配置
+ 属性funCall：对外发布FunCall的描述
+ 属性history：没有额外配置表示全部清理
+ 属性chain：base@close，转到base.json中的close节点（用于关闭通道）

###### upload
+ 调用终端上传文件到云 @see UploadFunction
+ 属性funCall：对外发布FunCall的描述

###### token
+ 查询Token用量消耗 @see CustomTokenAssistant
+ 属性funCall：对外发布FunCall的描述

#### 基础节点
###### close
+ 如果链接到该节点（chain）就主动关闭通道（HTTP STREAMING）
