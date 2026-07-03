### 第一性原则
+ 仅可以新增/更新/删除integration（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Integration介绍：../../REQUIREMENT.md
+ Integration手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 技术收口：
    + 修改last_update的维度从全局改成Agent维度，每个Agent都有一个last_update和knowledge_commit，独立保存在数据库，每个Agent独立提交自己的lastUpdate和knowledge_commit
    + 知识库真实目录需要改成`--agent-dir`的平级目录`knowledge`
        + 如`--agent-dir=/a/b`，则知识库根目录为`/a/knowledge`
        + 如Agent为`X`，则该Agent知识库目录为`/a/knowledge/X`
        + 目标目录不存在时需要自动创建空目录
    + 每个Agent知识库WIKI首页的路径改为原路径 + Agent名称
        + 如Agent为X，则路径为[原路径/knowledge/X/index.md]
    + `/knowledge_lastUpdate`需要支持通过`agentId`读取指定Agent的知识库最后更新时间
    + `/knowledge_path`需要支持通过`agentId`返回指定Agent的知识库目录
        + 返回路径需要符合`dirname(--agent-dir)/knowledge/<agentId>`
        + 当目标目录不存在时，需要创建后再返回
    + 转发`/v1/chat/completions`时需要按`metadata.agentId`重写`metadata.knowledge.path`、`metadata.knowledge.lastUpdate`、`metadata.knowledge.knowledgeCommit`
        + 其中`metadata.knowledge.path`需要改成`dirname(--agent-dir)/knowledge/<agentId>`的绝对路径
    + 如果请求体中已经存在`metadata.knowledge.agentId`和`metadata.knowledge.knowledge_disable`，在补充`metadata.knowledge.path`、`metadata.knowledge.lastUpdate`、`metadata.knowledge.knowledgeCommit`时不能丢失原有字段
    + 当请求显式传入`metadata.knowledge_commit`时，需要按`metadata.agentId`持久化本次提交值
        + 当`metadata.knowledge_commit=true`且SSE完整结束后，还需要回写该Agent的`last_update`和`knowledge_update_lock.last_requested_at`
    + `integration knowledge update-time`命令需要同时传`--timestamp`和`--agentId`，用于更新指定Agent的知识库最后更新时间
    + Knowledge需求：../../../knowledge/iteration/20260622-01/REQUIREMENT.md
    + Proxy需求：../../../proxy/iteration/20260622-1/REQUIREMENT.md

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写
+ 最小范围更新

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
