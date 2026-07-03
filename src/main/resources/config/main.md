#### Main节点
###### main
+ React主线程，继承自base@config和base@llm
+ base@close，转到base.json中的close节点（用于关闭通道）
+ FunCall：
    + 规划任务，模型关键词是[plan]，会转发到plan.json中的main配置
+ Rag：
    + 加载main/base.md作为模板
    + 替换规划 @see PlanRag

###### recall
+ 阅读外部记忆细节，自定义Function @See GitMemoryService

###### memory
+ 强制记录外部记忆，自定义Assistant @See MemoryAssistant

###### summary
+ LLM提取会话的配置，继承自base@llm

