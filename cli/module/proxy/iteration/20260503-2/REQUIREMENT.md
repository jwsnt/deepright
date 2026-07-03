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
+ 启动HTTP服务时，参数所有启动参数的取值保存至应用启动目录下的runtime.json中，每次启动都覆盖更新
+ 案例：
```
../.. --agent-dir /agent/ --site ../../../site
```
    + 保存内容：
    ```
    {
        "agent-dir": "/agent/",
        "site": "../../../site"
    }
    ```

### 编写代码
+ 以Golang编写以上代码，要求：
    + 与Cron模块共享使用文件名为data的sqlite存储，并使用连接池，避免每次都新建连接
        + Cron模块需求：../../../cron/REQUIREMENT.md
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




