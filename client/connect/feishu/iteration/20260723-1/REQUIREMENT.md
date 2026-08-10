### 第一性原则
+ 仅可以新增/更新/删除feishu（../..）同目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../../DESIGN.md
+ 本模块设计文档：../../../DESIGN.md

### 迭代要求
+ Connect介绍：../../../REQUIREMENT.md
+ Connect手册：../../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 重要提示
+ 设计前需要仔细阅读Connect的设计
    + Connect介绍：../../../REQUIREMENT.md
+ 严格遵守原始报文JSON SCHEMA：../../REQUEST_SCHEMA.json）
+ 严格遵守测试必过集：../../TEST_CASE.md

### 需求介绍
+ 飞书插件新增可选元数据参数 `mcp_url`，用于保存用户手动填写的飞书 MCP 服务地址；未填写时必须保持为空，不得阻止插件配置保存、启动或现有飞书收发链路。
+ `./feishu param` 必须在现有 `appId`、`appSecret` 后新增字段 `mcp_url`，固定描述为 `飞书MCP地址`；参数描述中不得包含文档 URL。
+ `./feishu command` 保持现有插件命令列表与顺序不变，且 `param` 命令继续包含在命令列表中；该字段属于 `param` 的元数据契约，不新增独立 CLI 子命令。
+ 插件配置浮层根据 `param` 返回值展示 `mcp_url` 输入框，并与其他插件参数一起提交为 `meta`；重新打开浮层时必须回填已保存的 `mcp_url`。
+ 仅当插件运行时主键为 `feishu` 且参数字段为 `mcp_url` 时，在该输入框右侧展示闪烁的 URL 小图标；图标必须具备 hover、focus 与无障碍名称，点击后使用新页面打开 `https://open.feishu.cn/page/mcp/7616252843057548502`，不得影响输入、保存或启动插件。
+ 不得修改 `appId`、`appSecret` 的必填校验；`mcp_url` 不参与飞书应用凭证校验，也不得作为敏感字段输出、记录或要求用户填写。

### 同步代码
+ ../../../feishu/REQUIREMENT.md
+ 所以设计/编译都需要遵守feishu的二进制和CLI收口原则

### 编写代码
+ 以Golang编写以上代码，要求：
    + 编译应用名：feishu
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 复制至Plugin：../../../../plugins/
