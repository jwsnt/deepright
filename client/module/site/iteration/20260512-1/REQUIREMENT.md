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
+ 每30秒调用一次/skills_warning，如果有SKILLS异常则在虚拟文件系统高频高亮闪动，闪动级别需要贯穿整个错误SKILLS的文件系统路径
+ 在错误的SKILL.md后新增一个高频高闪的感叹号小图标，点击后展示错误原因的浮层
    + integration需求：../../../integration/iteration/20260511-3/REQUIREMENT.md
    + Proxy需求：../../../proxy/iteration/20260512-1/REQUIREMENT.md
    + Skills需求：../../../skills/iteration/20260512-1/REQUIREMENT.md

### 编写代码
    + 所有固定定位遮罩层、确认框、弹窗必须统一管理层级、显示状态和pointer-events，按左侧Sidebar、居中会话区、右侧Sidebar分域隔离，禁止未激活浮层跨区域遮挡或拦截其他可操作控件点击
    + 涉及弹窗、日志面板、确认框、遮罩层、透明点击层、复制按钮、全局选区逻辑，默认都必须遵守“交互域隔离 + 隐藏视图彻底失活”原则
    + 设计上先统一弹层挂哪、用哪套坐标系，并避免把会溢出的弹层放进overflow:hidden 容器里
    + 代码上把定位收口成一个公共函数或portal 机制，不要在业务里各自手算 left/top
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
