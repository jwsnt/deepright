### 第一性原则
+ 仅可以新增/更新/删除当前需求文档（REQUIREMENT.md）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md

### CLI-GET
+ CLI-GET元数据介绍：../../REQUIREMENT.md
+ CLI-GET元数据手册：../../USER_GUIDE.md
+ CLI-GET元数据迭代：../../iteration/日期/REQUIREMENT.md

### CLI-CLI_SANDBOX
+ CLI-SANDBOX介绍：../REQUIREMENT.md
+ CLI-SANDBOX手册：../USER_GUIDE.md
+ CLI-SANDBOX迭代：../iteration/日期/REQUIREMENT.md

### Mac沙箱技术规范
+ 严格遵守技术规范：DESIGN.md

### 需求介绍
+ 按照WSL沙盒的DESIGN.md，实现功能等价替代的WSL环境沙盒方案：
    + 参考MAC方案，使用bubblewrap实现3种隔离能力：../mac/REQUIREMENT.md
        + 只能读写指定目录（含用户指定和默认目录，默认目录等价MAC方案）
        + 无网操作
        + 1+2的组合
    + 在proxy/integration/site模块中区分系统（MAC或是Windows/WSL）调用不同沙盒方案（包括目录选择）
+ 严格隔离MAC系统的实现路径，完全保持原样
+ 如果产生独立沙盒二进制代码，需要在主应用build.sh中linxu分支构建
    + 在当前目录创建build.sh进行独立构建（参考../mac/build.sh)

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
+ 同步代码：../../../integration/REQUIREMENT.md