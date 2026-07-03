### 第一性原则
+ 仅可以新增/更新/删除browser（ ../../..）目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../../../DESIGN.md
+ 本模块设计文档：../../../../DESIGN.md

### 需求介绍
+ 为browser_playwright代理的Playwright命令加入诊断日志，如：
    + 打印本次是否命中cookie缓存
    + 打印本次是否跳过host注入
    + 打印goto真正耗时毫秒
+ 日志保存在应用程序同目录下log目录, 以browser_log命名
+ 日志以分卷形式保存，单文件最多10M，最多保存4卷日志

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
