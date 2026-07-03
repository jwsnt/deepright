### 第一性原则
+ 仅可以新增/更新/删除当前需求文档（REQUIREMENT.md）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../DESIGN.md
+ 本模块设计文档：DESIGN.md

### 需求介绍
+ 为静态资源文件做HTTP映射
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 伴生应用
+ 与Proxy模块使用共同的HTTP服务
    + Proxy介绍：proxy/REQUIREMENT.md
    + Proxy手册：proxy/USER_GUIDE.md
    + Proxy迭代：proxy/iteration/日期/REQUIREMENT.md
+ 映射`/site`路径为由命令行参数--site指定的绝对路径（默认为Proxy启动同目录下的`site`目录）下的资源内容
    + 包括HTML/JS/CSS/Image等静态资源，路径可能为多层

### 编写代码
+ 以Golang编写以上代码，要求：
    + 代码简洁，包体积越小越好
    + 能用开源包的就用开源包
+ 作为Proxy模块的子模块和可独立运行的CLI命令来编写

### 验证测试
+ 以`test-case`作为指定目录，验证静态资源加在是否符合：
    + hello.html的内容为HELLO WORLD
    + js/hello.js的内容为1+1

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 相关迭代：iteration/日期/REQUIREMENT.md
+ 同步代码：../integration/REQUIREMENT.md
