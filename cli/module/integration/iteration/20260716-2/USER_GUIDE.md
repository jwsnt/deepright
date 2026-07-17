# 迭代 20260716-2：Windows WSL 依赖安装超时与降级

本次更新增强了 Windows WSL 安装器的依赖安装容错能力。通过 `install.bat` 安装 Integration 时，单个软件包安装卡住不会再阻断后续依赖安装、应用文件复制或 Integration 启动。

## 超时规则

- `apt-get update` 与每个软件包在单个安装源的一次尝试，默认最多执行 `10` 分钟。
- 超时后会先向安装命令及其子进程发送终止信号；`30` 秒内未退出时会强制结束，避免遗留 `apt`、`dpkg` 或安装脚本继续占用锁。
- 每次尝试都会将软件包、安装源、超时阈值、结果和退出码写入安装包目录的 `install.log`。

## 安装与 fallback

- 以下基础软件包逐个从 Ubuntu apt 安装：`git`、`python3`、`python3-pip`、`curl`、`build-essential`、`bubblewrap`、`xdg-utils`。
- 基础软件包没有备用源；任一包命令失败或超时后，会记录红色错误并继续下一个包。
- Node.js 先尝试 NodeSource 的 Node.js `20.x` 安装源；该次尝试失败或超时时，控制台会显示黄色提示，随后立即回退到 Ubuntu apt 安装 `nodejs` 和 `npm`。
- Node.js 的两个安装源都失败或超时时，会记录红色错误并继续后续安装流程。

## 安装结果

- 安装器始终会继续执行既有工具校验，便于确认 `git`、`node`、`npm`、`python3` 与 `bwrap` 的实际可用状态。
- 缺失工具只作为可见的安装结果，不会阻碍应用文件复制、启动脚本生成与 Integration 启动。
- 如需排查失败，可查看安装包目录下的 `install.log`；其中会记录每次 apt 与 NodeSource 尝试的来源、超时阈值、结果和命令输出。
