#### Summary节点
###### config
+ 总结上下文的继承配置，递归标记`summary`，避免嵌套总结。@see RequestRewriter

###### compress
+ 压缩上下文。@see RequestRewriter
+ 属性rag：
``` 替换占位符#schema为每个配置的response_schema，一般用于强化约束输出
"replace": "#schema",
"key": "rag_schema"
```
+ 属性response_schema：约束LLM返回

###### update
+ 继承自summary@update，主动触发，保存或更新用户画像到数据库（当用户要求保存画像、记录偏好、更新个人信息、持久化使用习惯等场景。@see ProfileUpdateFunction
