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
+ 当SSE响应报文里的`choices[0].metadata.__TARGET__`为非空标记时对__PROCESS__和__RETRY__展示逻辑做特殊处理:
    + 在当前等待响应的SSE气泡内侧的左下（与右侧的日期水平对齐，自身左对齐）展示__PROCESS__和__RETRY__
    + 展示时间和样式与气泡外侧左下保持一致
    + `__TARGET__`只影响展示位置，不参与提示作用域匹配
+ 如果为空，则逻辑保持不变
+ 无论是否携带`__TARGET__`，__PROCESS__和__RETRY__的提示作用域都只按当前ChatId匹配
+ __PROCESS__需求：../20260618-2/REQUIREMENT.md
+ __RETRY__需求：../20260619-1/REQUIREMENT.md

### 编写代码
+ 以现有site页面技术栈编写以上代码，要求：
    + 在../../index.html中按现有HTML/CSS/JavaScript组织方式实现
    + 不引入新的构建流程和额外运行时依赖
    + 代码简洁，尽量复用现有SSE解析、历史恢复、消息渲染和动画逻辑

### 撰写手册
+ 如有必要同步更新USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
