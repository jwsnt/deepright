### 第一性原则
+ 仅可以新增/更新/删除site（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ site介绍：../../REQUIREMENT.md
+ site手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 为右下角知识库WIKI添加小灯泡图标（在展开小图标右侧），点击后展示浮层，提示用户是否需要整理知识库WIKI
    + 提示文案：检测到知识库，是否重新整理？
    + 小灯泡图标在展开知识库图标的右侧
+ 浮层风格同：../20260515-3/REQUIREMENT.md
``` 提示内容
按知识库WIKI要求重新整理 [FILE:/...]
```
    + 其中[FILE:/...]是知识库在文件系统绝对路径（目录），需要用小气泡表示
        + 知识库的路径需求：
            + Integration需求：../../../integration/iteration/20260516-2/REQUIREMENT.md
            + Proxy需求：../../../proxy/iteration/20260516-2/REQUIREMENT.md
    + 点击确认后自动在当前Agent+Chat（当前对话框）发送提示内容
        + 通过知识库WIKI小灯泡浮层触发的SSE请求（/v1/chat/completions）需要在metadata中添加knowledge_commit
        ```
        {
            ...,
            "metadata": {
                ...
                "knowledge_commit": true
            }
        }
        ```
+ 知识库WIKI小灯泡图标默认为灰色，仅当最后更新时间距离当前时间超过8小时进行闪动
    + 判断时间需求（/knowledge_lastUpdate）：
        + Integration需求：../../../integration/iteration/20260516-2/REQUIREMENT.md
        + Proxy需求：../../../proxy/iteration/20260511-2/REQUIREMENT.md:15
+ 如果正在等待SSE响应（有请求中）时小灯泡图标不可点击（不展开浮层）
+ 完成后刷新知识库内容区和最近刷新时间
+ 实现方式参考：../20260516-1/REQUIREMENT.md

### 编写代码
    + 所有固定定位遮罩层、确认框、弹窗必须统一管理层级、显示状态和pointer-events，按左侧Sidebar、居中会话区、右侧Sidebar分域隔离，禁止未激活浮层跨区域遮挡或拦截其他可操作控件点击
    + 涉及弹窗、日志面板、确认框、遮罩层、透明点击层、复制按钮、全局选区逻辑，默认都必须遵守“交互域隔离 + 隐藏视图彻底失活”原则
    + 设计上先统一弹层挂哪、用哪套坐标系，并避免把会溢出的弹层放进overflow:hidden 容器里
    + 代码上把定位收口成一个公共函数或portal机制，不要在业务里各自手算left/top
    + 共享弹层承载多种业务时，不能只靠单一状态字段区分请求类型，必须在最终提交点用业务数据本身做幂等判定并显式写入目标metadata
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
