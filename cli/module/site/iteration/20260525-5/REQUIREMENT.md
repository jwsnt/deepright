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
+ 设置中蜂群开关参数swarm改为router_disable，类型不变，语意相反（router_disable=true表示关闭）
+ 开启SWARM（蜂群）时router_disable=false，关闭时router_disable=true
+ 页面展示开关名称不变
    + HTTP /api/edit中swarm改为router_disable，默认为true，语意相反
        + Proxy需求：../../../proxy/iteration/20260524-5/REQUIREMENT.md

### 编写代码
+ 所有前端依赖资源必须随站点页面一起可访问并在发布链路中强校验存在性与正确MIME，禁止出现页面已引用但静态服务未实际发布的本地资源
+ 所有固定定位遮罩层、确认框、弹窗必须统一管理层级、显示状态和pointer-events，按左侧Sidebar、居中会话区、右侧Sidebar分域隔离，禁止未激活浮层跨区域遮挡或拦截其他可操作控件点击
+ 涉及弹窗、日志面板、确认框、遮罩层、透明点击层、复制按钮、全局选区逻辑，默认都必须遵守“交互域隔离 + 隐藏视图彻底失活”原则
+ 设计上先统一弹层挂哪、用哪套坐标系，并避免把会溢出的弹层放进overflow:hidden 容器里
+ 所有浮层都需要先关闭当前打开的浮层后再打开自己，不要产生重叠，不要同时存在多个浮层
+ 代码上把定位收口成一个公共函数或portal机制，不要在业务里各自手算left/top
+ 能用开源包的就用开源包
+ 代码简洁，包体积越小越好

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
