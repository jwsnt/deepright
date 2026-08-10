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
+ 技术收口：
    + 修改接口/install_app，从主应用的config.json中的"install_app"读取数据，需要区分操作系统：
    ```
    {
        ...
        "install_app": {
            "linux": [...],
            "wsl": [...],
            "mac": [...],
        }
    }
    ```
        + 其中Linux系统使用linux，Mac系统使用mac，Windows（WSL）使用wsl
        + 数据结构不变，依旧是string array
        + --install_app参数不变，如果存在所有操作系统结构都要追加
    + 所有install_app的元素表示一个本地应用名称，需要检查是否已安装，已安装则从返回列表中删除
        + 不同操作系统判断方式不同
        + 接口缓存5分钟
    + Proxy需求：../../../proxy/iteration/20260614-1/REQUIREMENT.md

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写