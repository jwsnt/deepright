### 第一性原则
+ 仅可以新增/更新/删除当前需求文档（REQUIREMENT.md）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md

### CLI-GET
+ CLI-GET元数据介绍：../../../../REQUIREMENT.md
+ CLI-GET元数据手册：../../../../USER_GUIDE.md
+ CLI-GET元数据迭代：../../../../iteration/日期/REQUIREMENT.md

### CLI-CLI_SANDBOX
+ CLI-SANDBOX介绍：../../REQUIREMENT.md
+ CLI-SANDBOX手册：../../USER_GUIDE.md
+ CLI-SANDBOX迭代：../../iteration/日期/REQUIREMENT.md

### Mac沙箱技术规范
+ 严格遵守技术规范：../DESIGN.md

### 需求介绍
+ 修改沙盒模式为以下3种：
    + 用户选择目录（没有选择均认为没有权限）
        + key：filepick
    + 关闭网络
        + key：net
    + 两者都限
        + key：filepick_net
+ 制作3个独立沙盒，接收命令、执行命令、返回命令的方式不变（宿主使用命令行调用并等待返回)
+ 尽可能缩小3个沙盒的整个包体积
+ 更新sandbox模块的build.sh和主项目的build.sh
    +  ../../../build.sh
    + ../../../../../build.sh
+ CLI-GET调用沙盒需求：../../iteration/20260609-1/REQUIREMENT.md

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 相关迭代：iteration/日期/REQUIREMENT.md
+ 同步代码：../../../../../integration/REQUIREMENT.md