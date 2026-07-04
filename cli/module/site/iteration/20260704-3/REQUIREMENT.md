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
+ 右侧边栏手动 `CMD` 执行前，需要先判断当前输入内容在去除首尾空白后，是否仅为单个 `http/https` URL
+ 如果仅为单个 `http/https` URL，则本次运行不能再按系统命令执行，而是直接将该 URL 作为新页面打开
+ 只有“整段输入仅为 URL”时才走上述特判；其余普通命令、混合内容或非 `http/https` 文本仍保持原有系统命令执行逻辑

+ 以现有site页面技术栈编写以上代码，要求：
    + 在../../index.html中按现有HTML/CSS/JavaScript组织方式实现
    + 不引入新的构建流程和额外运行时依赖
    + 代码简洁，尽量复用现有SSE解析、assistant气泡渲染、思考片段展示、历史恢复、虚拟文件系统渲染和轮询状态同步逻辑
    + 能用现有开源包和浏览器能力的就不要重复造轮子
+ 最小范围更新

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
