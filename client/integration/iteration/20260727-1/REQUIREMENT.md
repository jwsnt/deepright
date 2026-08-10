### 第一性原则
+ 仅可以新增/更新/删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../build.sh`、`../../../build/install.ps1`、`../../../build/USER_GUIDE.md`、`../../../build/USER_GUIDE.txt` 与 `../../../config/app/API.md`、`../../../config/app/CANVAS.md`、`../../../config/app/DESIGN.md`。

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Integration介绍：../../REQUIREMENT.md
+ Integration手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ Windows WSL 安装器在首次创建受管的 `deepright` 发行版前，必须提供两个明确选项：`1` 使用微软官方 Ubuntu 渠道，`2` 使用清华镜像。用户只负责选择其中一个选项；选择后的下载、导入、用户创建、依赖安装、应用复制和启动均须由安装器自动完成，不得要求手工下载 Rootfs、执行 `wsl --import`、编辑 APT 配置或运行额外命令。
+ 选择微软官方渠道时，安装器先使用官方 `wsl --install -d Ubuntu`，失败后可自动使用微软/Ubuntu 官方 WSL Rootfs 作为同一分支的兜底；选择清华镜像时，安装器直接下载与 Windows 发布架构匹配的 Ubuntu 24.04 Noble Base Rootfs（amd64 或 arm64），并通过 `wsl --import` 创建 `deepright`。清华分支不得依赖 Microsoft Store 或要求用户转到浏览器下载。
+ 清华镜像导入后，安装器必须自动把 Ubuntu 的 APT 源切换为清华 Ubuntu 镜像：amd64 使用 `https://mirrors.tuna.tsinghua.edu.cn/ubuntu/`，arm64 使用 `https://mirrors.tuna.tsinghua.edu.cn/ubuntu-ports/`。应保留原源文件备份，并同时兼容 Ubuntu 24.04 的 DEB822 `ubuntu.sources` 与传统 `sources.list`；后续 `apt-get update` 和依赖安装必须使用切换后的源。清华源的更新或任一依赖安装失败时，安装器必须自动恢复该备份的 Ubuntu 官方源、重试当前操作，且后续依赖继续使用官方源。
+ Ubuntu Base Rootfs 不保证预装 `sudo`。安装器必须以 root 自动完成 APT 操作并补齐 `sudo`，随后继续为 `deepright` 配置默认用户和免密 sudo；不得因精简 Rootfs 缺少 `sudo` 而中断。
+ Rootfs 下载的内置 .NET HTTPS 请求出现 TLS 或连接错误时，安装器必须自动依次尝试 Windows 自带的 `curl.exe` 与 BITS 下载。清华镜像的全部 HTTPS 下载器失败后，必须自动回退到 Ubuntu 官方 WSL Rootfs；不得使用 HTTP 下载、要求用户更换 URL、关闭证书校验、手工下载或重新执行导入步骤。官方回退同样失败时，返回两次尝试原因并保留安装日志。
+ 不得导出、注销、修改或删除用户已有的 `Ubuntu` 或其他非 `deepright` WSL 发行版。官方分支仅可注销本次安装器新建、用于临时导出的 Ubuntu 实例；`deepright` 之外的发行版始终视为用户数据。

### 编写代码
+ 最小范围更新，不新增外部依赖
+ Windows 安装器的选项、镜像 URL、Rootfs 导入和 APT 换源逻辑必须保留在发布源代码中；构建 `linux/x86` 与 `linux/arm` Windows 安装载荷时，必须分别写入对应架构的官方和清华 Rootfs URL。

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`，说明 Windows WSL 安装器的两种下载来源、一次选择后的全自动行为、清华分支的 APT 加速、适用范围与既有发行版保护。
+ 更新构建模块面向 Windows 用户的两份安装说明。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求，不记录实现过程。
