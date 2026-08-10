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
+ 如果当前系统为MAC OS则（非MAC使用原逻辑）：
    + 所有数据库文件从~/Library/Application Support/deepright/data这个固定目录读取，写入
    + 所有插件二进制文件从当前应用同目录的plugins目录读取：
        + 案例：
            + APP路径：/Users/mac/arm/integration.app
            + 主二进制路径：/Users/mac/arm/integration.app/Contents/MacOS/integration
            + 二进制插件路径：/Users/mac/arm/integration.app/Contents/Resources/plugins
    + 参数--agent-dir默认指向~/Library/Application Support/deepright/agent
    + knowledge目录指向~/Library/Application Support/deepright/knowledge

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写