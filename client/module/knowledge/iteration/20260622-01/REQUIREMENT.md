### 第一性原则
+ 仅可以新增/更新/删除`../../目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md

### 迭代要求
+ Knowledge元数据介绍：../../REQUIREMENT.md
+ Knowledge元数据手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 修改last_update的维度从全局改成Agent维度，每个Agent都有一个last_update和knowledge_commit，独立保存在数据库，每个Agent独立提交自己的lastUpdate和knowledge_commit
+ 知识库真实目录需要改成`--agent-dir`的平级目录`knowledge`
    + 如`--agent-dir=/a/b`，则知识库根目录为`/a/knowledge`
    + 如Agent为`X`，则该Agent知识库目录为`/a/knowledge/X`
    + 目标目录不存在时需要自动创建空目录
+ 每个Agent知识库WIKI首页的路径改为原路径 + Agent名称
    + 如Agent为X，则路径为[原路径/knowledge/X/index.md]
+ `knowledge_runtime`表结构需要按`agent_id`维度维护`last_update`和`knowledge_commit`
+ 对外输出`metadata.knowledge`时需要按Agent返回：
    + `path = dirname(--agent-dir)/knowledge/<agentId>`的绝对路径
    + `lastUpdate = 对应Agent的last_update`
    + `knowledgeCommit = 对应Agent的knowledge_commit`
+ `KnowledgeDirForAgent`、`EnsureRuntimeForAgent`、`LookupRuntimeForAgent`、`MetadataForAgent`等能力需要统一支持Agent维度
    + `KnowledgeDirForAgent`返回的目录规则需要与`dirname(--agent-dir)/knowledge/<agentId>`保持一致
    + 对目标目录做读取或返回前，需要保证不存在时会创建空目录
+ `knowledge update-time`和`knowledge update-commit`命令需要支持通过`--agent-id`更新指定Agent的知识库运行时

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写


