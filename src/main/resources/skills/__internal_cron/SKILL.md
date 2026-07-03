---
name: __internal_cron
description: 创建、查看、修改、删除定时任务
---

### 如何使用
+ 任务元数据：任务模板，定时拆解出明细
+ 任务明细：实际需要执行的任务
+ 查看--help
```
#app cron --help
```
+ 如果用户无特殊指定，使用默认配置创建定时任务
    + Model Provider：`#provider`
    + DeviceId：`#device`
    + AgentId：`#agentId`
    + ChatId：`#chat`