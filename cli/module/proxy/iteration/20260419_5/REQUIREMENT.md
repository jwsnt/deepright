### 第一性原则
+ 仅可以新增/更新/删除proxy（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Proxy介绍：../../REQUIREMENT.md
+ Proxy手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 新建HTTP GET服务，路径为/api/files?path=xxx，模糊查找指定绝对路径下文件和目录列表（不递归子孙目录）
    + 模糊查找：在当前目录下不区分大小写的前缀匹配，即算匹配
+ path的value需要同时支持绝对路径和~路径
+ path的value需要标记结果是文件或是目录
+ path的value需要支持有空格的路径场景
+ 案例：
    + 请求'/api/files=/a/b/c，返回/a/b/c下的文件或目录
    ```
    [
        {
            "name": 文件1,
            "type": file
        },
        {
            "name": 目录1,
            "type": dir
        }
    ]
    ```
    + 请求'/api/files=~/a，返回/Users/用户名/a下的文件或目录

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 验证测试
+ 以`test-case`作为指定目录，验证代码生产内容是否符合：
    + path=a，目录：skills，文件：SOUL.md和user.md（区分大小写）
    + path=b，目录：skills

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




