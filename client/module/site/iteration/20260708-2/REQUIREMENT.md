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
+ Seedream 需要作为特殊模型处理
    + 允许在现有模型配置入口中配置 `seedream`
    + 但不能在居中对话输入框的模型列表中直接选择
    + 也不能在备忘录输入框的模型列表中直接选择
    + 也不能在插件浮层中的模型列表中直接选择
+ Seedream 的技能上报也需要做特殊处理
    + 不再使用容易与本地技能重名的 `image-seedream`
    + 统一改用内部技能名 `__internal_seedream`
    + 该技能名仅用于系统内部和上报链路，不要求作为普通本地技能暴露给用户自行创建或复用
+ `__internal_seedream` 只有在 Seedream 已完成可用配置时才允许上报
    + 上报范围包括 `/api/skills`
    + 上报范围包括前端 `@技能` 列表
    + 上报范围包括转发 `/v1/chat/completions` 时 `metadata.agents[].skills`
    + 上报范围包括 `cli/get` 时 `metadata.agents[].skills`
    + 未完成配置时，上述所有链路都不能出现 `__internal_seedream`
+ Seedream 的可用配置判定需要遵守 integration/proxy 现有收口方式
    + 配置来源按 `integration token --provider seedream` 判定
    + 至少要能拿到可用 `token`、`__url`、`__model_multi_output`
    + 其中 `token` 必须真实已配置
    + `__url` 和 `__model_multi_output` 如果用户未显式填写，但系统存在默认值，也应视为可用
+ 本次需求不能只改前端展示
    + site 侧需要保证 Seedream 可配置但不可在上述模型列表中直接使用
    + integration 与 proxy 侧需要保证 `__internal_seedream` 的上报条件一致
    + 所有链路都需要遵守 integration 的二进制和 CLI 收口原则

### 编写代码
+ 以现有site页面技术栈编写以上代码，要求：
    + 在../../index.html中按现有HTML/CSS/JavaScript组织方式实现
    + 不引入新的构建流程和额外运行时依赖
    + 代码简洁，尽量复用现有SSE解析、assistant气泡渲染、思考片段展示、历史恢复和轮询状态同步逻辑
    + 能用现有开源包和浏览器能力的就不要重复造轮子
+ 最小范围更新

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
