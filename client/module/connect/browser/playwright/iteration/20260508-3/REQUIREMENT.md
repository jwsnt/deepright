### 第一性原则
+ 仅可以新增/更新/删除browser（ ../../..）目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../../../DESIGN.md
+ 本模块设计文档：../../../../DESIGN.md

### 需求介绍
+ 为browser_playwright代理的Playwright命令自动增加Cookie, 而不需要通过命令行添加
+ 通过Cookie模块获取并注入
    + Cookie需求，需要使用相对路径调用：../../../REQUIREMENT.md

### 案例参考
``` 假设应用程序为browser_playwright
./browser_playwright --session agent-a@ctrip-home eval 'document.body ? document.body.innerText.slice(0, 1000) : ""'
```
+ 首先归一化--session参数为小写字母，防止因为大小写引起的匹配错误

### 同步代码
+ ../../../browser/REQUIREMENT.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 编写帮助
+ browser help需要为代理的browser_playwright（playwright-cli）的所有命令提供完整的使用说明

### 编写代码
+ 以Golang编写以上代码，要求：
    + 二进制收口, 最终交付给用户的主程序必须是`browser`一个二进制文件（不包含插件obscura目录下文件），日志必须是与browser同目录下的browser.log
    + 首跳导航先按目标域名自动注入Chrome Cookie，再用domcontentloaded作为完成条件，避免动态站点第一次goto卡到超时
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
