# 迭代 20260727-1：Windows WSL Ubuntu 下载源选择

Windows 安装包首次创建 `deepright` WSL 发行版时，会显示两个安装来源：

1. Microsoft 官方（推荐）
2. 清华镜像

用户只需选择一次。之后安装器会自动完成 Ubuntu Rootfs 下载、WSL 导入、`deepright` 用户创建、APT 依赖安装、应用文件复制和启动；不需要手工下载、解压、运行 `wsl --import` 或修改 Linux 软件源。

## Microsoft 官方

该分支首先执行官方 `wsl --install -d Ubuntu`。如官方 WSL Ubuntu 下载未成功，安装器会自动下载 Ubuntu 官方 WSL Rootfs 并继续导入，不会要求用户切换到手工安装流程。

## 清华镜像

该分支自动下载清华大学镜像站的 Ubuntu 24.04 Noble Base Rootfs，并导入为受管的 `deepright` 发行版。导入后会自动调整 APT 软件源：

- amd64：`https://mirrors.tuna.tsinghua.edu.cn/ubuntu/`
- arm64：`https://mirrors.tuna.tsinghua.edu.cn/ubuntu-ports/`

安装器同时兼容 Ubuntu 24.04 的 `/etc/apt/sources.list.d/ubuntu.sources` 和传统 `/etc/apt/sources.list`，并在源文件旁保留 `.deepright.bak` 备份。后续 `apt-get update` 和运行依赖安装都会使用清华源；若更新或任一依赖安装失败，安装器会自动恢复该备份的 Ubuntu 官方源、重试当前操作，并让后续依赖继续使用官方源。

Base Rootfs 可能未附带 `sudo`。安装器会用 root 自动执行 APT 操作、安装 `sudo`，再设置 `deepright` 的默认用户与免密 sudo，因此不会因这个精简差异要求额外人工操作。

若 PowerShell 内置 HTTPS 下载因 TLS 或网络连接失败，安装器会自动依次改用 Windows 自带的 `curl.exe` 和 BITS 下载。清华镜像的全部 HTTPS 下载器都失败时，安装器会自动回退到 Ubuntu 官方 WSL Rootfs；始终不使用 HTTP 下载。用户不需要更换链接、关闭证书校验或手工导入。

## 安全与重复执行

安装器不会导出、注销、修改或删除已有的 `Ubuntu` 或其他非 `deepright` WSL 发行版。健康的 `deepright` 已存在时，重复运行安装包只刷新应用文件，不会重新下载或导入系统，也不会重新要求选择下载来源。

双击安装包会显示选择菜单。自动化部署如需跳过菜单，可使用管理员 PowerShell：

```powershell
.\install.ps1 -UbuntuSource official
.\install.ps1 -UbuntuSource tsinghua
```
