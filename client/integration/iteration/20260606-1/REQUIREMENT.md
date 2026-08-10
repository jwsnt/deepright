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
+ 技术规范的指导下收口：
    + 新增/api/plugins/exec?key=x&command=y&param1=value1&param2=value2&...执行指定插件的指定命令
        + key：插件标识、command：改插件的命令
        + param1=value1&param2=value2&...：可以有任意组，表示插件参数
        + Proxy需求：../../../proxy/iteration/20260606-1/REQUIREMENT.md
    + /api/plugins/exec等待插件执行命令超时等待改为由integration/proxy参数--plugin_exec_timeout决定的毫秒数，默认600秒，如果超时或启动失败需要在integration.log保留日志
    + 如果插件完成了则立即返回

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写