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
- 启动 `integration start`

### 3. 后续启动

安装完成后可直接双击：

```text
start.bat
```

该脚本会在 WSL 中执行：

```sh
~/.integration --start
```

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
