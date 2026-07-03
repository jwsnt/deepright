#### 长期记忆
###### summary
+ 不在React中，LLM当前会话提取长文记忆
+ 替换占位符#schema为每个配置的response_schema，一般用于强化约束输出

###### refresh
+ 不在React中，LLM历史会话提取Soul/User（printReason=false，强制关闭Reason，保证JSON格式）
+ 替换占位符#schema为每个配置的response_schema，一般用于强化约束输出
+ 迷你版本的main@main，但仅Rag 长期记忆（rag_memory）和使用工具[cli]
+ Rag：加载main/base.md作为模板