### 第一性原则
+ 仅可以新增/更新/删除build（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ build介绍：../../REQUIREMENT.md
+ build手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计与交付都需要遵守integration单二进制收口原则

### 需求介绍
+ 同步 `install.bat`、`install.ps1`、`start.bat` 的 Windows WSL2 安装与启动流程，和主构建产出的 WSL 发布行为保持一致。
+ WSL2 Ubuntu 实例别名与默认用户名固定为 `deepright`。
+ 安装流程必须幂等：已有环境不重复重装，只补齐缺失依赖并覆盖同步 release 文件。
+ 复制载荷时优先使用 `build/app/`，兼容手工调试目录；若 `app/` 不存在，则允许直接使用当前目录作为 release 包根目录。
+ 编写 `DESIGN.md` 与 `USER_GUIDE.md`，明确 build 模块职责、目录结构和使用方式。

### 编写代码
+ 以现有 Windows Batch + PowerShell 技术栈编写
+ 不引入新的构建流程和额外运行时依赖
+ 最小范围更新

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
