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
+ 如果当前启动了浏览器插件，则在左侧边栏插件展开按钮的右下方（水平向下5px，垂直向下5px）展示一个浏览器的小图标，图标3x3像素，实心荧光青绿色（按当前深色和浅色模式区分饱和度）慢速闪烁
+ 点击图标后调用/api/plugins/exec?key=browser&command=instance init&agentId=当前AgentID&chatId=当前会话ID，来加载浏览器CDP
```
browser instance init --agentId 当前AgentID --chatId 当前会话ID
```
+ 加载后在锁定整个界面（左右侧边栏和居中对话框），提示正在进行浏览器插件登录，关闭后解锁，直到接口返回或报错后解锁。
    + Integration需求：../../../integration/iteration/20260606-1/REQUIREMENT.md
    + Proxy需求：../../../proxy/iteration/20260606-1/REQUIREMENT.md
+ 在锁定界面的浮层上增加一个完成按钮，点击后调用/api/plugins/exec?key=browser&command=instance shutdown&agentId=当前AgentID&chatId=当前会话ID，来销毁浏览器CDP，解锁界面
```
browser instance shutdown --agentId 当前AgentID --chatId 当前会话ID
```
+ 锁定界面的/api/plugins/exec不设置超时，持续等待直到用户点击完成
+ 完成按钮需要在锁定界面后30秒才出现

### 编写代码
+ 能用开源包的就用开源包
+ 代码简洁，包体积越小越好

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写