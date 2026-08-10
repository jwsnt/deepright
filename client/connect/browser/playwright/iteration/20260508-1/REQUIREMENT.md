### 第一性原则
+ 仅可以新增/更新/删除browser（ ../../..）目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../../../DESIGN.md
+ 本模块设计文档：../../../../DESIGN.md

### 需求介绍
+ 新增create命令, 先通过Agent和Chat（会话ID）在instance中创建CDP服务, 然后对返回的端口attach并使用Agent@Chat做session
    + 首先归一化--agentId和--chatId参数为小写字母，防止因为大小写引起的匹配错误
    + instance需求，需要使用相对路径调用：../../../instance/REQUIREMENT.md
```
browser_playwright create --agentId xxx --chatId yyy
```
+ 案例：假定返回端口为10086, 那么调用browser_playwright attach
```
browser_playwright --session xxx@yyy attach --cdp=ws://127.0.0.1:10086/devtools/browser
```
    + 首先归一化--session参数为小写字母，防止因为大小写引起的匹配错误
    + 参数session为Agent@Chat的拼接字符串
    + 注意使用ws协议

### 同步代码
+ ../../../browser/REQUIREMENT.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 编写帮助
+ browser help需要为代理的browser_playwright（playwright-cli）的所有命令提供完整的使用说明

### 编写代码
+ 以Golang编写以上代码，要求：
    + 二进制收口, 最终交付给用户的主程序必须是`browser`一个二进制文件（不包含插件obscura目录下文件），日志必须是与browser同目录下的browser.log
    + 代码简洁，包体积越小越好
    + 能用开源包的就用开源包
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写
+ 严格遵守指纹需求：../../../CHECK.md

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 相关迭代：iteration/日期/REQUIREMENT.md
+ 同步代码：../../../../../integration/REQUIREMENT.md（每次都要同步更新代码）
