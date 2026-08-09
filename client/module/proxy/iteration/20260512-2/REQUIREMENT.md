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
+ 转发/v1/chat/completions及cli/get和cli/pub提交metadata的git路径每次实时获取，不要缓存
    + Agent需求：../../../agent/iteration/20260512-1/REQUIREMENT.md
+ 新建/install_app接口，返回一个[string]
    + 如果git没有安装则在返回中添加一个元素"git"
    ```
    [{"git"}]
    ```
+ 新增参数--install_app，以逗号分隔，如果指定了则合并至/install_app接口返回（需要去重），默认为空

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



