#### CLI节点
###### verify
+ 复盘入口，包括文件检查、内容符合检查

###### task
+ 发布CLI任务到成员 @see CliTaskFunction
+ 属性funCall：对外发布FunCall的描述

###### sub
+ 发布CLI命令到队列 @see CliSubFunction
+ 属性funCall：对外发布FunCall的描述

###### pub
+ 接收CLI结果并合并至LLM @see CliPubFunction
+ 属性funCall：对外发布FunCall的描述

###### get
+ 客户端轮询CLI队列，并上报环境变量 @see CliGetFunction
+ 属性funCall：对外发布FunCall的描述
+ 属性rag：独立，不继承base@config
+ 属性response_schema：约束LLM返回

###### ops
+ CLI安全性校验，使用指定Prompt校验后是否允许执行
+ 属性rag：
``` 替换占位符#schema为每个配置的response_schema，一般用于强化约束输出
"replace": "#schema",
"key": "rag_schema"
```
+ 属性response_schema：约束LLM返回


