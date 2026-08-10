### 第一性原则
+ 仅可以新增/更新/删除proxy（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Proxy介绍：../../REQUIREMENT.md
+ Proxy手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 新建HTTP POST服务，路径为/api/plugins/config?key=xxx&&agentId=yyy，更新指定插件的配置
    + 使用Connect模块的meta-create，如果配置已存在则使用meta-update更新
    + Connect meta-creata需求：../../../connect/REQUIREMENT.md
    + 插件标识统一原则：展示名（name）可以是中文，系统主键（key）必须稳定唯一，所有运行时链路只能用主键，不能混用展示名
+ 属性映射：
        + 系统主键：key，必须校验是否传递，插件返回的稳定唯一标识，例如feishu
        + 三方名称：name，仅作为展示字段透出，例如飞书，禁止作为运行时主键写入/查询/拼接内部链路
        + 连接元数据：输入表单的json string
        + 是否支持Stream（流式回复）：默认false
        + 回调用地址：插件可执行文件的文件系统绝对路径
        + 绑定的AgentId：必须校验是否传递
        + 会话ID（CHAT_ID）：如果传了会话ID就用，不传就使用key（插件键，不是展示名）
        + 选择的模型：必须校验是否传递
        + 是否深度思考：默认值为false
+ 创建失败需要返回原因

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写


