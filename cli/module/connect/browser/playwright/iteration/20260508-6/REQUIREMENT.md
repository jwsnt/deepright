### 第一性原则
+ 仅可以新增/更新/删除browser（ ../../..）目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../../../DESIGN.md
+ 本模块设计文档：../../../../DESIGN.md

### 需求介绍
+ 为browser_playwright代理的Playwright命令自动创建CDP实例, 而不需要通过命令行创建

### 案例参考
+ 如果没有指定--session但指定agentId和chatId则先检查通过`browser_instance get`检查是否存在CDP服务，存在则继续后续，不存在则先创建
    + 首先归一化--session参数为小写字母，防止因为大小写引起的匹配错误
    + CDP服务需求：../../../instance/REQUIREMENT.md
```
./browser_playwright --agentId agent-a --chatId ctrip-home eval 'document.body ? document.body.innerText.slice(0, 1000) : ""'
```
``` 最终编译版本为browser代理browser_instance, 检查对应CDP服务, 不存在则创建
./browser_instance get --agentId agent-a --chatId ctrip-home
```
+ 如果指定--session但又指定agentId和chatId则使用--session拆解成agentId和chatId
```
./browser_playwright --session agent-a@ctrip-home --agentId agent-b --chatId ctrip-home eval 'document.body ? document.body.innerText.slice(0, 1000) : ""'
```
```
./browser_instance get --agentId agent-a --chatId ctrip-home
```

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
