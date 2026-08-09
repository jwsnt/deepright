### 第一性原则
+ 仅可以新增/更新/删除connect（../..）同目录的文件和文件夹
+ 如非授权，禁止修改其他插件目录文件和文件夹
### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
    + browser、email、feishu等

### 迭代要求
+ Connect介绍：../../REQUIREMENT.md
+ Connect手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 关联需求
+ 飞书插件：../../feishu/iteration/日期/REQUIREMENT.md

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 重要提示
+ 设计前需要仔细阅读Connect的设计，需要兼容方案
    + Connect介绍：../../REQUIREMENT.md

### 需求介绍
+ 为connect模块增加新命令list-plugins，获取plugins目录（子孙目录不需要）所有二进制可执行文件的name命令和param命令的组合json
+ 案例
``` plugins/a
./a name 返回 IM
./a param 返回 ["key"]
```
``` plugins/b
./b name 返回 APP
./b param 返回 ["token","ticket"]
```
    + 以上案例
    ```
    ./connect list-plugins 返回 [{"name":"IM","param":["key"]},{"name":"APP","param":["token","ticket"]}]
    ```
+ 每次执行后缓存，缓存时间以命令行参数--connect-cache指定的毫秒数，默认10秒

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




