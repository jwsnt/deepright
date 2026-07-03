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
+ 制作一个MAC OS的沙箱，运行CLI_SANDBOX
+ 制作一个../build.sh，将MAC的sandbox代码构建为.app，并将构建物放到../../release/mac下
+ 区分arm和x86架构，需要放下
    + ../../release/mac/arm
    + ../../release/mac/x86

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