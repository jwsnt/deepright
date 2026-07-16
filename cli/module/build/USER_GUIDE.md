# Build WSL2 使用手册

## 模块作用

`cli/module/build` 提供 Windows 上的 WSL2 安装入口脚本，用于把 Linux release 载荷安装到受管的 Ubuntu WSL 发行版中，并直接启动 `integration`。

## 文件说明

- `install.bat`
  - Windows 双击安装入口
- `install.ps1`
  - WSL2 安装、依赖补齐、文件复制、启动主流程
- `start.bat`
  - 已安装后的快速启动入口
- `DeepRight.ico`
  - Windows 快捷方式图标
- `app/`
  - 可选
  - 如果存在，则脚本优先复制该目录作为 Linux release 载荷

## 使用方式

### 1. 准备 release 载荷

任选一种方式：

- 把 Linux release 内容放到 `build/app/` 下
- 或者把 `install.bat`、`install.ps1`、`start.bat` 放到已经构建好的 Linux release 根目录旁边，直接以当前目录作为安装源

release 根目录至少需要包含：

- `integration`
- `config/`
- `site/`

如果发布物带有插件或 WSL 沙盒帮助程序，也应一并放在同级目录，例如：

- `plugins/`
- `helpers/`

### 2. 首次安装

在 Windows 上双击：

```text
install.bat
```

安装脚本会自动：

- 请求管理员权限
- 检查或安装 WSL2
- 创建 `deepright` Ubuntu 实例
- 创建默认用户 `deepright`
- 安装缺失依赖
- 复制 release 文件到 WSL 中的 `~/deepright`
- 在桌面和开始菜单创建带 `DeepRight.ico` 的启动快捷方式
- 以 root 身份启动 `integration start`

### 3. 后续启动

安装完成后可直接双击：

```text
start.bat
```

也可以直接使用安装时自动创建的 `DeepRight` 桌面/开始菜单快捷方式，图标与应用主站点资源保持一致。

该脚本会在 WSL 中以非交互式 sudo 执行：

```sh
sudo -n /usr/bin/env HOME=/home/deepright /app/integration start
```

安装器会为 `deepright` 配置免密 sudo；`-n` 确保启动过程不会等待密码输入。虽然进程以 root 运行，HOME 保持为 `/home/deepright`，因此会继续使用原有的 agent 和运行时数据目录。

每次启动会在以下日志文件记录包装脚本路径、完整启动命令、PID 文件路径、启动器 PID、退出码，以及最终 integration 进程的 PID、UID、用户名：

```text
/home/deepright/deepright/integration.log
```

启动完成行中的 `integration_uid=0 integration_user=root` 表示服务已实际以 root 身份运行。

### 4. apt 安装超时与 fallback

- `apt-get update` 及每个软件包的一次安装尝试最多执行 10 分钟；超时后会终止对应安装进程，安装器继续处理后续软件包。
- `git`、`python3`、`python3-pip`、`curl`、`build-essential`、`bubblewrap`、`xdg-utils` 按单个包安装，当前使用 Ubuntu apt 源。
- Node.js 依次尝试 NodeSource 20.x 与 Ubuntu apt（`nodejs`、`npm`）两个源。
- 最终无法安装的软件包会在控制台和 `install.log` 中以红色错误记录，但不会中断后续安装与应用启动。

## 幂等行为

- 已安装的 WSL2 不会重复安装
- 已存在且健康的 `deepright` 发行版不会重复导入
- apt 依赖只补齐缺失项
- release 目录会重新覆盖复制，确保 WSL 中的运行文件与当前包一致
- Ubuntu rootfs 会根据 Windows 主机架构自动选择 `amd64` 或 `arm64`

## 默认约定

- WSL 发行版别名：`deepright`
- WSL 默认用户：`deepright`
- Windows 受管目录：`C:\WSL`
- WSL 安装目录：`/home/deepright/deepright`

## 常见问题

### 双击后提示需要重启

这是 WSL2 组件首次启用后的正常行为。重启 Windows 后再次运行 `install.bat` 即可。

### 双击 `start.bat` 提示未安装

说明当前 `deepright` 发行版中还没有生成 `~/.integration`。请先执行 `install.bat` 完成安装。

### 想跳过自动启动

可以在 PowerShell 中手动执行：

```powershell
.\install.ps1 -SkipLaunch
```
