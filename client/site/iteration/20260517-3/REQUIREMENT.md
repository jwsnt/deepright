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
+ 居中对话框需要支持HTML片段渲染，同时兼容现有的Markdown和Latex公式
    + Latex公式需求：../20260517-2/REQUIREMENT.md
``` 支持结构
+ 文本结构：div section article p span br hr
+ 标题：h1 到 h6
+ 强调：strong em u del sub sup
+ 列表：ul ol li
+ 引用/折叠：blockquote details summary
+ 代码：pre code
+ 表格：table thead tbody tr th td
+ 媒体：img iframe video audio source
+ 语义容器：figure figcaption
+ 链接：a
```
``` 常用属性
+ 全局属性：class title aria-label aria-hidden role
+ 链接：href target rel
+ iframe：src loading allow allowfullscreen referrerpolicy
+ 图片：src alt loading width height
+ 视频：src controls preload poster muted playsinline loop autoplay
+ 音频：src controls preload
+ source：src type
+ 表格单元格：colspan rowspan，th还支持scope
```
``` 明确过滤
+ script
+ 所有行内事件属性，比如 onclick
+ style
+srcdoc
```
``` 危险协议
+ javascript:、vbscript:、非图片型 data:
+ iframe的src只接受 http(s) 或/开头的同源路径
```

### 编写代码
+ 所有前端依赖资源必须随站点页面一起可访问并在发布链路中强校验存在性与正确MIME，禁止出现页面已引用但静态服务未实际发布的本地资源
+ 所有固定定位遮罩层、确认框、弹窗必须统一管理层级、显示状态和pointer-events，按左侧Sidebar、居中会话区、右侧Sidebar分域隔离，禁止未激活浮层跨区域遮挡或拦截其他可操作控件点击
+ 涉及弹窗、日志面板、确认框、遮罩层、透明点击层、复制按钮、全局选区逻辑，默认都必须遵守“交互域隔离 + 隐藏视图彻底失活”原则
+ 设计上先统一弹层挂哪、用哪套坐标系，并避免把会溢出的弹层放进overflow:hidden 容器里
+ 代码上把定位收口成一个公共函数或portal机制，不要在业务里各自手算left/top
+ 能用开源包的就用开源包
+ 代码简洁，包体积越小越好

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
