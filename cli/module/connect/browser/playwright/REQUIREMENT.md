### 第一性原则
+ 仅可以新增/更新/删除browser（ ../..）目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 需求介绍
+ 为playwright-go创建go版本的playwright-cli
    + https://github.com/playwright-community/playwright-go
    + https://github.com/microsoft/playwright-cli
+ 最终编译可执行程序仅需要browser_playwright，同时包含cli和支持代码
+ playwright-cli所有的命令功能都需要支持

### CDP支持
+ 需要增加CDP服务连接的支持
```
browser_playwright attach --cdp=chrome
browser_playwright attach --cdp=<url>
```

### Session归一化
+ 归一化--session参数为小写字母，防止因为大小写引起的匹配错误

### 实例复用与导航
> 新增自 iteration/20260510-1/REQUIREMENT.md
+ `eval`执行需要增加页面内超时保护，避免把daemon卡死
+ 在instance复用前需要先检查CDP服务健康，不要盲目复用旧实例
+ 首跳导航需要先按目标域名自动注入Chrome Cookie，再用`domcontentloaded`作为完成条件，避免动态站点第一次`goto`卡超时

### UserAgent
> 新增自 iteration/20260511-1/REQUIREMENT.md
+ 为browser_playwright代理的Playwright命令自动添加`--user-agent`为Chrome标准UA：
```
Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36
```

#### 插件帮助
+ 必须实现help来提供完成的插件使用手册
``` 假设插件可执行程序为a
./a help
```

### 同步代码
+ ../../browser/REQUIREMENT.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 编写代码
+ 以Golang编写以上代码，要求：
    + 二进制收口, 最终交付给用户的主程序必须是`browser`一个二进制文件（不包含插件obscura目录下文件），日志必须是与browser同目录下的browser.log
        + browser_playwright.pid：插件内部Playwright daemon的PID
        + browser.pid：插件层PID
    + 指定驱动加载目录应用启动同目录的`playwright/driver`目录下，Install只在driver缺失时才调用，如果driver已存在，就直接playwright.Run()，不再重复打印下载日志
    + 首跳导航先按目标域名自动注入Chrome Cookie，再用`domcontentloaded`作为完成条件
    + 默认自动补齐Chrome标准UA；如果显式传入`--user-agent`则以显式值为准
    + 代码简洁，包体积越小越好
    + 能用开源包的就用开源包
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 相关迭代：iteration/日期/REQUIREMENT.md
+ 同步代码：../../../integration/REQUIREMENT.md（每次都要同步更新代码）
