### 第一性原则
+ 仅可以新增/更新/删除当前需求文档（REQUIREMENT.md）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../DESIGN.md
+ 本模块设计文档：DESIGN.md

### 需求介绍
+ 参考install.bat（检测、安装WSL2和Ubuntu、安装apt依赖）和start.bat（启动应用）为../build.sh增加Windows WSL2安装执行包
    + WSL2 Ubuntu实例使用别名和用户名均deepright
+ 安装依赖后产生可执行包，执行可执行包后启动（同MAC版本）
    + 已经安装完依赖和环境再次点击时不要重复安装
    + 过程使用的目录与install.bat一致
+ 复制app的目录为构建release目录中不同的Linux/WSL版本目录
+ 使用MAC版本一样的Icon

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 相关迭代：iteration/日期/REQUIREMENT.md
+ 同步代码：../integration/REQUIREMENT.md
