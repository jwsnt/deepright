# Build Windows WSL2 Design

## 目标

`build` 模块负责维护 Windows 侧 WSL2 安装入口的参考实现，使 Linux 发布物可以通过双击安装包完成：

- 检测并安装 WSL2
- 创建受管 Ubuntu 发行版 `deepright`
- 创建默认用户 `deepright`
- 安装运行期所需 apt 依赖
- 复制 Linux release 载荷到 WSL 用户目录
- 启动 `integration start`

## 设计原则

- `build` 目录只负责 Windows WSL2 安装入口与文档，不直接承担 Linux 业务二进制的构建。
- WSL 发行版别名和默认用户名都固定为 `deepright`，避免多个入口产生语义漂移。
- 安装过程必须幂等：
  - 已安装 WSL2 时不重复安装
  - 已存在健康的 `deepright` 发行版时不重复导入
  - 已安装的 apt 依赖只补齐缺失项
  - 每次安装都覆盖同步最新 release 文件
- rootfs 架构随 Windows 主机架构自动选择：
  - x86_64 Windows -> amd64 Ubuntu rootfs
  - ARM64 Windows -> arm64 Ubuntu rootfs
- 载荷目录优先读取 `build/app/`；若不存在，则允许直接把当前目录当作 release 包根目录，兼容手工调试与打包后的目录结构。

## 目录职责

- `install.bat`
  - Windows 双击入口
  - 负责调用 `install.ps1`
  - 失败时保留窗口，便于读取错误
- `install.ps1`
  - 安装主流程
  - 负责 WSL2、Ubuntu、用户、依赖、文件复制、启动
- `start.bat`
  - 安装完成后的二次启动入口
  - 通过 WSL 用户家目录下的 `~/.integration` 启动主程序
- `DeepRight.ico`
  - Windows 启动快捷方式图标

## 运行流程

1. `install.bat` 启动 `install.ps1`
2. `install.ps1` 自动提权
3. 检查 Windows 版本与 WSL2 能力
4. 写入 `.wslconfig` 并启用 `networkingMode=mirrored`
5. 创建或修复受管 Ubuntu 发行版 `deepright`
6. 创建用户 `deepright` 并设为默认用户
7. 安装缺失依赖：`git`、`npm`、`python3`、`bubblewrap`
8. 复制 release 目录到 `~/<TargetDirName>`
9. 生成 `~/.integration` 与 `~/start-deepright.sh`
10. 生成桌面与开始菜单快捷方式，并统一使用 `DeepRight.ico`
11. 默认执行 `~/.integration --start`

## 与主构建的关系

- `../build.sh` 负责生成不同 Linux/WSL 架构的发布目录。
- `build` 目录中的脚本定义了 Windows 侧安装与启动的期望行为。
- 图标仍复用主站点图标资源，由 `../build.sh` 在正式发布时转换并打包为 Windows `.ico` 到目标产物。
