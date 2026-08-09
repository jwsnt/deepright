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
+ 在左侧边栏设置按钮的水平右侧，增加一个关机按钮，点击后展示确认浮动（居中对话框水平垂直，模糊背景），有2个按钮：关机（绑定回车）和取消（绑定ESC）
+ 点击关机后开启关机动画，模糊背景，提示倒计时5秒，5秒后关闭当前窗口且不要提示是否需要关闭
+ 开始动画后调用/api/shutdown关闭主进程
    + Integration需求：../../../integration/iteration/20260608-2/REQUIREMENT.md

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
