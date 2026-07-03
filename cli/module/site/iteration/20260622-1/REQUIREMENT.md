### 第一性原则
+ 仅可以新增/更新/删除site（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Site介绍：../../REQUIREMENT.md
+ Site手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../REQUIREMENT.md
+ 所以设计/编译都需要遵守site页面的现有前端收口原则

### 需求介绍
+ 知识库WIKI修改为Agent维度，切换Agent时需要同时刷新知识库
+ 当前Agent的知识库WIKI默认首页固定为`/knowledge/<agentId>/index.md`
+ 知识库最后更新时间、过期判断、最近刷新展示都需要按Agent调用`/knowledge_lastUpdate?agentId=...`
+ 知识库整理浮层读取的目录需要按Agent调用`/knowledge_path?agentId=...`
    + 返回的真实目录需要符合`dirname(--agent-dir)/knowledge/<agentId>`
    + 如果当前Agent知识库目录不存在，点击时需要创建空目录后再继续后续流程
+ 知识库WIKI的自动整理开关需要改成Agent维度，本地按Agent单独保存
+ 发送`/v1/chat/completions`时，`knowledge_disable`需要从顶层`metadata.knowledge_disable`移动到`metadata.knowledge.knowledge_disable`
    + `metadata.knowledge.agentId`为当前Agent名称
    + 不再发送顶层`metadata.knowledge_disable`
+ 通过知识库WIKI小灯泡触发的手动整理请求仍然需要发送顶层`metadata.knowledge_commit=true`
+ 刷新按钮需要在收起状态下先刷新当前Agent知识库，再展开WIKI；如果当前已展开，则点击后收起
+ 小灯泡在知识库目录尚未预加载时也需要可以点击，点击时懒加载`/knowledge_path?agentId=...`
+ Integration需求：../../../integration/iteration/20260622-1/REQUIREMENT.md
+ Knowledge需求：../../../knowledge/iteration/20260622-01/REQUIREMENT.md
+ Proxy需求：../../../proxy/iteration/20260622-1/REQUIREMENT.md

### 编写代码
+ 以现有site页面技术栈编写以上代码，要求：
    + 在../../index.html中按现有HTML/CSS/JavaScript组织方式实现
    + 不引入新的构建流程和额外运行时依赖
    + 代码简洁，尽量复用现有待发送消息队列、消息编辑、提示动效和轮询状态同步逻辑
    + 能用现有开源包和浏览器能力的就不要重复造轮子
+ 最小范围更新

### 撰写手册
+ 如有必要同步更新USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
