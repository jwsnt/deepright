# DeepRight WSL 沙盒技术白皮书

## 1. 文档目标

本文档描述 DeepRight 在 WSL 上基于 `bubblewrap` 的沙盒目标实现，覆盖以下内容：

- 如何在 WSL 上完整复刻当前 mac 沙盒的三模式能力
- Windows 侧交互层与 WSL 侧执行层的分工
- `bubblewrap` 在文件系统和网络隔离中的作用
- 目录授权、chat 维度状态管理与执行约束方案
- 构建、分发、首次安装与运行时调用链
- 哪些能力可以等价实现，哪些只能近似实现，哪些无法 1:1 复制

与 mac 文档不同，本文档的 **runner/helper 落地部分仍是 WSL 目标设计**；但本文档约束的上层契约已经与当前仓库实现对齐，尤其是 chat 维度的 `mode + allowedDir` 状态模型、prime 语义，以及执行时必须重新注入 `--allowed-dir` 的要求。

当前仓库里，目录选择相关实现已经落到以下文件：

- `cli/module/cli-get/sandbox/wsl/picker/main_windows.go`
- `cli/module/cli-get/sandbox/wsl/picker-launcher/main.go`
- `cli/module/cli-get/sandbox/wsl/picker-launcher/picker_defaults.go`

同时，helper 落地与上层契约仍继续围绕以下文件演进：

- `cli/module/cli-get/sandbox/wsl/helper/main.go`
- `cli/module/cli-get/sandbox/wsl/build.sh`

同时会复用现有文件中的模式与集成设计：

- `cli/module/cli-get/sandbox/service/mode.go`
- `cli/module/cli-get/sandbox/service/service.go`
- `cli/module/integration/main.go`
- `cli/module/connect/sandboxstate/sandboxstate.go`
- `cli/module/cli-get/main.go`

## 2. 总体结论

如果目标是“在 WSL 上完整复制当前 mac 沙盒的用户能力”，推荐采用**双层架构**而不是把所有逻辑都塞进 WSL 内部：

1. **Windows runner 层**
   - 负责目录授权 UI
   - 负责首次未配置时的固定初始目录
   - 负责路径归一化
   - 不负责跨 chat 的“最后一次目录”缓存
   - 负责日志
   - 负责调用 `wsl.exe` 进入指定 WSL 发行版

2. **WSL helper 层**
   - 负责解析模式
   - 负责构造 `bubblewrap` 参数
   - 负责在 WSL Linux 内真正执行命令
   - 负责文件系统与网络隔离

3. **bubblewrap 执行层**
   - 通过 namespace + bind mount + tmpfs 提供约束
   - 用空根文件系统实现“默认不可见”
   - 用 `--unshare-net` 实现禁网

也就是说，WSL 版对 mac 版的最佳映射不是：

- “把 `sandbox-exec` 直接换成 `bwrap`”

而是：

- “把 mac 的 `.app runner + helper + Seatbelt`”
- 改成 “Windows runner + WSL helper + bubblewrap”

这样做的原因是：

- `bubblewrap` 只解决进程沙盒，不解决图形授权 UI
- WSL 本身支持 Windows 与 Linux 命令互操作
- WSL 2 使用真实 Linux 内核，更适合承载基于 namespace 的沙盒

因此，**从功能语义看**，当前 mac 的 3 种模式都可以在 WSL 上继续保留：

1. `filepick`
   - 只允许访问用户授权目录和默认目录
   - 保留网络
2. `net`
   - 不给业务目录访问权限
   - 禁网
3. `filepick_net`
   - 目录白名单 + 禁网

但如果要求“平台实现细节也 1:1 一模一样”，则做不到。做不到的点主要是：

- 没有 mac `.app`、`codesign`、App Sandbox entitlement 这一套
- 没有 `NSOpenPanel`
- 没有 security-scoped bookmark
- 无法在“纯 WSL、无 Windows interop、无 WSLg”的环境中保证图形目录选择器可用

## 3. 术语说明

### 3.1 Windows runner

指在 Windows 上运行的 `CLI_SANDBOX.exe`，职责类似当前 mac 版的 `.app/Contents/MacOS/CLI_SANDBOX`：

- 负责 UI
- 负责固定初始目录与参数透传
- 负责日志
- 负责桥接到 WSL

### 3.2 WSL helper

指运行在目标 WSL 发行版中的 Linux ELF 二进制，职责类似当前 mac 版的 helper：

- 负责真正执行命令
- 负责构造隔离策略
- 负责调用 `bubblewrap`

### 3.3 sandbox distro

指专门用于运行沙盒 helper 的 WSL 发行版。

推荐和日常开发发行版分离，例如：

- `deepright-dev`：用户日常开发
- `deepright-sandbox`：只用于沙盒执行

### 3.4 interop

指 WSL 的 Windows/Linux 互操作能力，例如：

- 在 Linux 命令行中执行 `powershell.exe`
- 在 Windows 命令行中执行 `wsl.exe`

### 3.5 WSLg

指 WSL 的 Linux GUI 应用支持层，用于在 Windows 上显示 Linux GUI 应用窗口。

### 3.6 本项目中的“沙盒”

在 WSL 设计里，“沙盒”一词具体指三件事：

- 会话维度的模式：`filepick` / `net` / `filepick_net`
- Windows runner + WSL helper 的双层执行链
- `bubblewrap` 提供的 Linux namespace / 挂载隔离

## 4. 运行模式

与当前 mac 版保持完全一致，仍定义 3 种模式：

- `filepick`
  - 需要目录授权
  - 不禁网
- `net`
  - 不需要目录授权
  - 禁网
- `filepick_net`
  - 需要目录授权
  - 禁网

推荐继续复用现有模式常量：

```go
const (
    SandboxModeFilePick    = "filepick"
    SandboxModeNet         = "net"
    SandboxModeFilePickNet = "filepick_net"
)
```

这能保证：

- `integration`
- `cli-get`
- `sandbox state`

在上层完全不需要因为 WSL 新方案而改业务语义。

## 5. 架构分层

### 5.1 逻辑分层

从外到内，WSL 版建议分成六层：

1. **integration 会话控制层**
   - 和 mac 版一致
   - 负责记录 `chatId -> sandbox mode + allowedDir`
   - 负责 prime 目录授权

2. **cli-get 调度层**
   - 和 mac 版一致
   - 根据会话状态决定是否走 sandbox runner

3. **Windows runner 层**
   - `CLI_SANDBOX.exe`
   - 负责目录选择 UI、固定初始目录、路径转换、日志、helper 安装、参数透传

4. **WSL bridge 层**
   - 通过 `wsl.exe -d <distro> -- ...`
   - 把命令、环境变量、授权目录传进目标发行版

5. **WSL helper 层**
   - Linux ELF `CLI_SANDBOX`
   - 负责构造 `bubblewrap` 参数并执行 shell

6. **bubblewrap 执行层**
   - 由 `bwrap ... <shell> -c <cmd>` 生效
   - 真正提供文件和网络隔离

### 5.2 组件图

```text
integration / cli-get
        |
        v
CLI_SANDBOX.exe                          (Windows runner)
        |
        v
wsl.exe -d <sandbox-distro> -- ...       (bridge)
        |
        v
/home/.../CLI_SANDBOX                    (WSL helper)
        |
        v
bwrap ... /bin/sh -c "<cmd>"             (bubblewrap)
        |
        v
Linux namespaces + bind mounts + tmpfs
```

## 6. 打包与交付设计

### 6.1 打包入口

建议新增：

- `cli/module/cli-get/sandbox/wsl/build.sh`
- `cli/module/cli-get/sandbox/wsl/main.go`

其中：

- `build.sh` 负责双架构构建与打包
- `main.go` 负责生成 runner / helper / config 产物

### 6.2 产物目录结构

WSL 版不再使用 `.app`，但建议保留“外层 runner + 内层 helper + resources”的分层结构：

```text
CLI_SANDBOX/
└── Contents
    ├── Windows
    │   └── CLI_SANDBOX.exe           # 外层 runner
    ├── Linux
    │   └── CLI_SANDBOX               # 内层 helper
    ├── Resources
    │   ├── runner-config.json
    │   └── helper.sha256
    └── VERSION
```

每个模式分别单独打包，目录建议为：

```text
release/wsl/x86/filepick/CLI_SANDBOX/...
release/wsl/x86/net/CLI_SANDBOX/...
release/wsl/x86/filepick_net/CLI_SANDBOX/...
release/wsl/arm/filepick/CLI_SANDBOX/...
release/wsl/arm/net/CLI_SANDBOX/...
release/wsl/arm/filepick_net/CLI_SANDBOX/...
```

其中：

- `x86` 对应：
  - Windows `GOARCH=amd64`
  - Linux `GOARCH=amd64`
- `arm` 对应：
  - Windows `GOARCH=arm64`
  - Linux `GOARCH=arm64`

### 6.3 构建步骤

建议 `build.sh` 做以下事情：

1. 归一化配置
2. 创建模式目录
3. 构建 Windows runner：
   - `GOOS=windows`
4. 构建 Linux helper：
   - `GOOS=linux`
5. 生成 `runner-config.json`
6. 生成 helper 校验文件
7. 生成最终分发目录

### 6.4 runner-config 的作用

建议 `runner-config.json` 至少存以下字段：

```json
{
  "mode": "filepick_net",
  "distro": "deepright-sandbox",
  "helper_install_dir": "/home/%USER%/.local/share/deepright/cli-sandbox",
  "picker_backend": "auto",
  "timeout_ms": 180000
}
```

作用：

- 指定当前模式
- 指定默认目标 WSL 发行版
- 指定 helper 在 WSL 中的安装目录
- 指定目录选择器优先级
- 指定默认执行超时

### 6.5 helper 的首次安装与升级

因为 runner 在 Windows，helper 在 WSL Linux，所以不能像 mac `.app` 那样直接走 bundle 内相对路径执行。

建议 runner 在启动时执行：

1. 通过 `wsl.exe -d <distro> -- sh -lc 'uname -m'` 探测目标架构
2. 通过 `wsl.exe -d <distro> -- sh -lc 'test -x <helper-path>'` 检查 helper 是否已安装
3. 比较本地 `helper.sha256` 与远端 helper 校验值
4. 若缺失或版本不一致：
   - 把 helper 上传到 WSL 侧安装目录
   - `chmod +x`
   - 写入版本文件

建议安装位置：

```text
~/.local/share/deepright/cli-sandbox/<version>/<mode>/CLI_SANDBOX
```

这样便于：

- 多版本共存
- mode 独立部署
- 升级回滚

## 7. 运行前探测与环境前提

### 7.1 必须满足的前提

WSL 沙盒要可靠工作，建议把以下前提视为硬要求：

1. 必须是 **WSL 2**
2. 目标发行版内必须安装 `bubblewrap`
3. 目标发行版必须能正常执行基础命令：
   - `/bin/sh`
   - `mkdir`
   - `chmod`
   - `cat`
4. helper 安装目录必须可写

推荐但不是绝对硬要求：

5. 使用独立的 `sandbox distro`
6. 在该发行版中关闭 interop
7. 默认不把 Windows 路径加入 sandbox 进程的 `PATH`

### 7.2 为什么必须要求 WSL 2

因为 `bubblewrap` 的核心依赖是 Linux namespace 和真实 Linux syscall 语义。

WSL 2 使用真实 Linux 内核，而不是 WSL 1 的 syscall translation 模型，因此 WSL 2 才是可支持 `bubblewrap` 的合理目标环境。

### 7.3 runner 启动时的 preflight

建议 Windows runner 每次启动时做如下探测：

1. `wsl.exe --status`
2. `wsl.exe -l -v`
3. 目标发行版是否存在
4. 目标发行版版本是否为 `2`
5. `wsl.exe -d <distro> -- sh -lc 'command -v bwrap'`
6. `wsl.exe -d <distro> -- sh -lc 'bwrap --version'`
7. `wsl.exe -d <distro> -- sh -lc 'bwrap --ro-bind / / true'`

如果任何一步失败，runner 直接返回明确错误，例如：

- `未检测到目标 WSL 发行版`
- `CLI_SANDBOX 仅支持 WSL 2`
- `目标 WSL 发行版未安装 bubblewrap`
- `bubblewrap 预检失败，请检查 user namespace / mount namespace 能力`

## 8. Windows runner 设计

### 8.1 runner 的职责

WSL 版 runner 的职责与 mac 版 runner 尽量保持一致：

- 读取模式配置
- 处理目录授权
- 设定固定初始目录
- 规范化路径
- 管理日志
- 安装/升级 helper
- 调用 `wsl.exe`
- 清洗桥接层噪音

### 8.2 runner 启动流程

建议启动流程如下：

1. 读取 `runner-config.json`
2. 解析命令行参数：
   - `--mode`
   - `--shell`
   - `--log-file`
   - `--allowed-dir`
   - `--cmd`
   - `--timeout`
3. preflight WSL 与 helper
4. 计算默认日志目录
5. 如模式需要目录授权，则执行 `resolveAllowedDirectory(...)`
6. 规范化成目标发行版内可用的 Linux 路径
7. 构造 helper 参数与环境变量
8. 通过 `wsl.exe` 启动 helper
9. 透传 stdout
10. 清洗 stderr 噪音
11. 继承 helper 退出码

### 8.3 日志目录策略

建议 Windows runner 默认日志路径为：

```text
%LOCALAPPDATA%\DeepRight\CLI_SANDBOX\Logs\<mode>\sandbox.log
```

helper 内部的次级日志建议位于 WSL：

```text
~/.local/state/deepright/cli-sandbox/<mode>/sandbox.log
```

### 8.4 目录授权解析逻辑

建议 `resolveAllowedDirectory(explicit)` 的行为如下：

1. 如果传入 `--allowed-dir`
   - 尝试识别它是：
     - Windows 路径
     - WSL Linux 路径
     - `\\wsl$\<distro>\...` UNC 路径
   - 归一化
   - 校验目录存在
   - 返回

2. 如果没有 `--allowed-dir`
   - 直接调用目录选择器
   - 不读取任何 runner / helper 级“最后一次目录”缓存
   - 不预加载当前 chat 已持久化的 `allowedDir`
   - 首次未配置或重新授权时，从固定初始目录开始
     - Windows 原生 picker：`C:\`
     - PowerShell `FolderBrowserDialog` fallback：`C:\`
   - 目录选择窗口最长等待 **60 秒**
   - 校验目录存在
   - 归一化
   - 返回

当前实现已经移除 helper 本地“最后一次目录”回退。
真正参与执行的目录只能来自：

- 当前 chat 显式透传的 `allowedDir`
- 当前这一次 picker 交互的返回值

### 8.5 目录选择器后端

为了尽量复制 mac 的“图形目录授权”，WSL 版建议提供三套 picker：

1. **Windows 原生目录选择器**
   - 推荐默认后端
   - 当前实现优先使用 `CLI_SANDBOX_PICKER.exe`
   - 通过 `SHBrowseForFolderW` + `BFFM_SETSELECTIONW` 把初始目录固定到 `C:\`

2. **PowerShell / .NET fallback**
   - 当前实现的后备方案
   - 使用 `System.Windows.Forms.FolderBrowserDialog`
   - 同样把 `SelectedPath` 设为 `C:\`

3. **纯命令行 fallback**
   - 如果没有 Windows picker 也没有 PowerShell 可用
   - 明确提示用户必须使用 `--allowed-dir`

推荐优先级：

```text
explicit -> windows picker -> powershell picker -> fail
```

当前实现建议把 Windows picker 与 WSLg picker 的单次等待时间统一限制为 **60 秒**，超时后返回目录授权失败，而不是无限阻塞。

### 8.5.1 取消语义与 fallback 终止规则

WSL 目录选择链路在实现上可能由多个进程串起来，例如：

- `CLI_SANDBOX`
- `CLI_SANDBOX_PICKER_LAUNCHER`
- `CLI_SANDBOX_PICKER.exe`

这条链路必须把“用户取消”定义成一个**明确且可透传的终止信号**，而不是普通失败。

推荐约定如下：

1. 最内层 picker
   - 用户点击取消时直接 `exit 1`
   - 不输出业务错误文本

2. launcher
   - 如果收到内层 picker 的“取消”，自己也应直接 `exit 1`
   - 不应把 `canceled`、`picker canceled` 之类文本写到 `stderr`
   - 不应在原生 picker 取消后再自动切到 PowerShell picker

3. helper
   - 调用 launcher 时，只要识别到“取消”，就直接返回 `CLI_SANDBOX权限拒绝`
   - 不再继续 fallback 到 PowerShell 或其他后端

这样做的原因是：

- 用户已经明确点了“取消”，语义上本次授权流程应立即结束
- 如果继续 fallback 到第二个后端，容易出现“先关闭第一个 picker，随后又弹出第二个 picker”的问题
- 第二个 picker 可能不在最前台，用户会感知为“窗口消失了但流程还卡着”

因此在 WSL 实现里，**“取消”必须短路整条 picker fallback 链**。

### 8.5.2 mixed-version 兼容

发布过程中可能出现以下短暂状态：

- 只更新了 `CLI_SANDBOX`
- `CLI_SANDBOX_PICKER_LAUNCHER` 还是旧版本
- 不同 mode 目录中的 helper / launcher / picker 版本不一致

因此 helper 侧建议额外兼容一层旧输出：

- 如果子进程是 `exit 1` 且输出为空，视为取消
- 如果子进程是 `exit 1` 且输出为 `canceled` 或 `picker canceled`，也视为取消

这不是为了长期保留多套协议，而是为了避免发布窗口内因为版本不一致，把“取消”误判成普通错误，继而错误触发第二个 fallback picker。

### 8.6 为什么建议默认用 Windows picker

因为 WSL 沙盒的控制面最好放在 Windows 上：

- 更容易复制 mac 的“外层交互壳”体验
- 不依赖 WSLg
- 不依赖目标 distro 是否启用了 interop
- 更适合和 `integration.app` 之类的 Windows 宿主集成

### 8.7 路径转换

runner 需要处理 3 类输入路径：

1. Windows 路径
   - `C:\Users\me\work`
2. WSL Linux 路径
   - `/home/me/work`
3. WSL UNC 路径
   - `\\wsl$\Ubuntu\home\me\work`

建议统一转换成：

- `windowsPath`
- `linuxPath`

双字段路径表示，例如：

```json
{
  "windows_path": "C:\\Users\\me\\work",
  "linux_path": "/mnt/c/Users/me/work",
  "source": "picker",
  "updated_at": "2026-06-18T10:00:00Z"
}
```

### 8.8 stderr 噪音清洗

Windows runner 需要过滤的噪音与 mac 不同，重点应是：

- `wsl.exe` 的非业务提示
- helper 安装过程中的无害输出
- Windows / PowerShell picker 的 UI 噪音

原则是：

- 不隐藏业务错误
- 只清洗平台噪音

## 9. 当前方案不使用本地目录缓存回退

### 9.1 权威目录来源

当前方案中，真正参与执行的目录只允许来自两处：

1. `cli_sandbox_state` 中当前 chat 的 `allowedDir`
2. 本次 prime / 显式 `--allowed-dir` 传入的目录

这意味着：

- 没有 Windows runner 级“最后一次目录”回退
- 没有 WSL helper 级“最后一次目录”回退
- 没有跨 chat 共享的目录记忆

### 9.2 为什么不保留本地目录缓存

原因与 mac 当前实现一致：

- 用户会在多个 chat 之间频繁切换
- 每个 chat 可能绑定完全不同的目录
- “上一次选中的目录”不是“当前 chat 应使用的目录”

如果保留本地目录缓存作为执行回退，就会重新引入 chat 串话问题。

### 9.3 并发语义

当前方案下，并发控制收敛为：

- prime 期间的一次 picker 交互
- 成功后把 `mode + allowedDir` 一起写入 SQLite
- 后续执行按 chat 再次显式注入 `--allowed-dir`

因此目录授权不再依赖本地缓存文件锁来维持正确性。

## 10. WSL helper 设计

### 10.1 helper 的职责

helper 是最终命令执行器，职责包括：

- 解析模式
- 解析授权目录
- 构造 `bubblewrap` 参数
- 调用 `bwrap`
- 快速识别权限拒绝
- 返回原始输出

### 10.2 helper 的启动模式

建议完全兼容当前 helper 的使用方式：

1. 只完成目录授权，不执行命令
   - `--allowed-dir /path`
   - 或者在 `filepick` / `filepick_net` 模式下仅设置 `CLI_SANDBOX_FORCE_PICK=1`，进入 **pick-only** 授权路径
2. 执行命令
   - `--mode`
   - `--cmd`
   - `--timeout`
   - `--shell`

### 10.3 helper 的目录来源优先级

建议优先级如下：

1. `CLI_SANDBOX_ALLOWED_DIR`
2. `--allowed-dir`
3. 若当前调用被设计成 picker 流程，则拉起目录选择器
4. 否则返回错误

错误文案建议为：

```text
未找到已授权目录，请先通过 CLI_SANDBOX 完成目录授权或显式传入 --allowed-dir
```

### 10.4 helper 的运行步骤

建议运行步骤如下：

1. 解析参数
2. 归一化模式
3. 解析 shell
4. 解析授权目录
   - 如需强制重新授权，则只执行目录选择，最长等待 **60 秒**
5. 构建 `bubblewrap` 参数
6. `exec.CommandContext(..., "bwrap", args...)`
7. 同步收集 stdout/stderr
8. 命中权限错误时快速终止
9. 返回退出码和输出

这里的设计重点是：

- 目录授权与命令执行是两条独立路径
- 强制目录授权时，不再依赖附带一个短超时的占位命令去间接触发 picker
- picker 的等待时间统一由目录选择层自己控制，当前为 **60 秒**

## 11. 文件访问控制的核心技术手段

### 11.1 实现结论

WSL 版真正的隔离能力应完全落在 `bubblewrap` 上，而不是落在字符串过滤、路径前缀拦截或 shell 黑名单上。

关键执行形态应为：

```text
bwrap ... <shell> -c <cmd>
```

### 11.2 为什么 WSL 版可以比 mac 版更强

mac 当前设计是：

- `(allow default)`
- 再 deny 某些敏感路径

WSL + `bubblewrap` 版则可以反过来做：

- 先创建空根文件系统
- 再显式挂入需要的目录

这意味着 WSL 版可以天然更接近：

- “默认不可见”
- “只开放明确挂进去的路径”

### 11.3 基础 sandbox 模板

建议所有模式的基础参数都从以下模板起步：

```text
bwrap
  --die-with-parent
  --new-session
  --clearenv
  --unshare-all
  --proc /proc
  --dev /dev
  --tmpfs /tmp
```

其含义是：

- 父进程死了，沙盒也一起退出
- 子进程独立 session
- 清空环境变量
- 默认创建新的 namespace
- 用新的 `/proc`
- 提供最小 `/dev`
- 用新的 `/tmp`

### 11.4 `filepick` 的目录限制实现

目标语义：

- 只允许访问选中的业务目录
- 允许少量默认目录
- 保留网络

推荐 `bubblewrap` 侧做法：

1. `--unshare-all`
2. 追加 `--share-net`
3. `--ro-bind` 运行时必须目录
4. `--bind` 用户授权目录
5. `--bind` deepright 状态目录
6. 创建私有 `/tmp` 与 `/var/tmp`
7. `--chdir` 到授权目录

### 11.5 `net` 的禁网实现

目标语义：

- 不给业务目录访问权限
- 命令可运行
- 禁网

推荐做法：

1. 保留 `--unshare-all`
2. 不追加 `--share-net`
3. 不挂用户业务目录
4. 仅只读挂 shell、运行库和基础配置目录
5. 使用私有 `/tmp` 与 `/var/tmp`
6. `cwd` 落到 `/tmp`

### 11.6 `filepick_net` 的组合实现

目标语义：

- 只允许访问授权目录和默认目录
- 禁网

做法就是：

- 文件系统挂载策略与 `filepick` 相同
- 网络策略与 `net` 相同

### 11.7 特殊处理目录与挂载方案

Bubblewrap 从空根文件系统启动；所有模式共用同一组系统只读根，以保证 shell 和常用命令可运行，但不会给工具目录写权限。

#### A. 系统 shell、工具、运行库和基础配置（只读）

以下目录在宿主存在时使用 `--ro-bind <path> <path>` 挂入：

- `/usr`
- `/bin`
- `/sbin`
- `/lib`
- `/lib64`
- `/etc`

`/usr` 覆盖大多数发行版的 `/usr/bin`、`/usr/sbin`、`/usr/lib` 和 `/usr/local`。沙箱内 `PATH` 固定为：

```text
/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
```

#### B. 按宿主存在情况挂入的包管理器运行时根（只读）

- `/run/current-system/sw`
- `/nix/store`

它们同样使用 `--ro-bind`。路径不存在时不生成 mount 参数，从而兼容 Debian/Ubuntu、NixOS 和其它 WSL 发行版。

#### C. 精确可读写目录

以下目录使用 `--bind`：

- `os.UserConfigDir()/CLI_SANDBOX`
- 已存在的 `~/deepright`
- 仅在 `filepick` / `filepick_net` 中：用户选择目录及其 `filepath.EvalSymlinks()` 解析后的真实路径

`net` 不挂入业务目录；即使内部调用误传 `pickedDir`，参数构造也必须忽略该目录。

#### D. 私有临时目录

- `/tmp` 使用 `--tmpfs /tmp` 创建，子进程设置 `TMPDIR=/tmp`。
- `/var/tmp` 使用 `--dir /var/tmp` 在沙箱内创建。
- 不再使用 `--bind /var/tmp /var/tmp`；宿主共享临时目录不得进入任一模式。
- scratch home、`HOME`、`XDG_*` 与 `ZDOTDIR` 均使用沙箱内目录，避免读取真实用户 home 的配置和缓存。

#### E. 明确不豁免的路径

- `/home`或任何用户 Home 根目录。
- `/mnt`、`/mnt/c`、`/mnt/c/Program Files`和Windows系统目录。
- 整棵`/opt`。

若用户选择 `/mnt/c/...` 目录，只精确挂入该选择子路径及其真实路径，不能因而暴露整块Windows磁盘。`filepick` / `filepick_net` 还必须拒绝与 `/usr`、`/bin`、`/sbin`、`/lib`、`/lib64`、`/etc` 重叠的选择目录，避免后续可写业务 bind 覆盖只读系统挂载。Linuxbrew、pyenv、nvm、Cargo、Conda等用户Home工具根不默认挂入；未来必须通过可信宿主配置精确选择并只读挂载，不能由环境变量扩权。

#### F. 三种模式的挂载行为

| 模式 | 只读系统/工具根 | 业务目录 | 状态目录 | 临时目录 | 网络 |
| --- | --- | --- | --- | --- | --- |
| `filepick` | 挂入 | 选择目录及其真实路径可读写 | 精确可读写挂入 | 私有 `/tmp`、`/var/tmp` | `--share-net` |
| `net` | 挂入 | 不挂入 | 精确可读写挂入 | 私有 `/tmp`、`/var/tmp` | 保持 `--unshare-all` 的禁网 |
| `filepick_net` | 挂入 | 选择目录及其真实路径可读写 | 精确可读写挂入 | 私有 `/tmp`、`/var/tmp` | 保持 `--unshare-all` 的禁网 |

### 11.8 Windows 驱动器路径的处理

如果授权目录位于：

```text
/mnt/c/...
```

则只 bind mount 该子路径，不允许整棵 `/mnt` 被带入沙盒。

例如：

- 允许：
  - `/mnt/c/Users/me/work`
- 不允许：
  - `/mnt`
  - `/mnt/c`

这样可以避免用户选了一个子目录，却把整块 Windows 盘暴露进来。

### 11.9 symlink / realpath 处理

为避免白名单目录因 symlink 失配导致误拒绝，建议对授权目录同时处理：

- `filepath.Clean`
- `filepath.EvalSymlinks`

必要时把：

- 逻辑路径
- 实际路径

都加入 bind mount 列表。

### 11.10 这不是字符串级命令审查

与 mac 一样，WSL 版不分析命令字符串本身，不做：

- `ls /` 特判
- `find /` 特判
- `python -c` 黑名单
- pipeline 黑名单

限制发生在文件访问和 namespace 层。

## 12. 网络隔离设计

### 12.1 基础做法

对于 `net` / `filepick_net`，推荐直接依赖 `bubblewrap` 的 network namespace 隔离：

- `--unshare-net`

而不是：

- 改 `/etc/hosts`
- 改代理环境变量
- 依赖 iptables

### 12.2 WSL 环境下的额外风险

WSL 和普通 Linux 不同，风险不只来自 Linux TCP/IP 网络，还来自 Windows 互操作能力。

特别是：

- WSL 支持从 Linux 调起 Windows 可执行文件
- WSL 支持访问 Windows 挂载路径

这意味着如果设计不严谨，就可能出现：

- Linux 网络被隔离了
- 但进程仍能借 Windows 互操作绕回宿主侧能力

### 12.3 严格禁网的推荐方案

如果想尽量贴近 mac 的“禁网”语义，推荐以下组合：

1. helper 运行在**独立 sandbox distro**
2. 该 distro 的 `/etc/wsl.conf` 配置：

```ini
[interop]
enabled=false
appendWindowsPath=false
```

3. sandbox 进程 `--clearenv`
4. 不把 `/mnt/c`、`/mnt/d` 等 Windows 盘挂入沙盒
5. `PATH` 只设置 Linux 侧最小路径

这样网络隔离就不只是不通 TCP/IP，而是把“从沙盒里再去借 Windows 能力”这条路也尽量堵住。

### 12.4 为什么仍然建议保留 Windows runner

即使 sandbox distro 关闭了 interop，也不影响 Windows runner 工作，因为：

- 目录选择器在 Windows 侧运行
- runner 再通过 `wsl.exe` 把结果传给 helper

这正是 WSL 方案比“把所有能力都塞进 WSL 内部”更稳的地方。

## 13. 命令执行环境控制

### 13.1 工作目录

建议规则如下：

- `filepick` / `filepick_net`
  - 默认 `cwd` = 授权目录
- `net`
  - 默认 `cwd` = `/tmp`

### 13.2 HOME / XDG / ZDOTDIR

与 mac 当前 `ZDOTDIR` 的考虑一致，WSL 版也必须避免 shell 去读真实 home 下的 dotfiles。

建议在 sandbox 中统一设置：

- `HOME=<scratch-home>`
- `XDG_CONFIG_HOME=<scratch-home>/.config`
- `XDG_CACHE_HOME=<scratch-home>/.cache`
- `XDG_STATE_HOME=<scratch-home>/.local/state`
- `ZDOTDIR=<scratch-home>`
- `TMPDIR=/tmp`

对于 `bash` / `sh` 还建议清空：

- `BASH_ENV`
- `ENV`

### 13.3 shell 选择

默认 shell 建议仍保持：

- 优先 `$SHELL`
- 否则 `/bin/sh`

### 13.4 标准输入 / 输出 / 错误输出

为了完整复制当前 `CLI_SANDBOX --cmd` 的使用语义，建议继续直接继承：

- stdin
- stdout
- stderr

因此以下形式都应成立：

```bash
CLI_SANDBOX.exe --mode net --cmd "echo hello"

printf "a\nb\n" | CLI_SANDBOX.exe --mode net --cmd "cat - | wc -l"
```

### 13.5 长命令与复杂引号

为避免 Windows -> WSL -> shell 的多层转义成本，建议 helper 支持两种执行输入：

1. 现有：
   - `--cmd "<shell command>"`
2. 增强：
   - `--cmd-file <path>`

其中 `--cmd-file` 推荐让 runner 把脚本文本写入临时文件，再挂入沙盒。

### 13.6 命令参数传递建议

为避免极长命令在 Windows 命令行长度、`wsl.exe` 参数转发、shell quoting 上出问题，建议 helper 与 runner 内部优先使用：

- 环境变量
- 临时文件
- `stdin`

而不是把所有内容硬塞进一条命令字符串。

## 14. 权限错误与快速失败机制

### 14.1 问题背景

和 mac 一样，如果只是等待子进程自然退出，权限拒绝可能表现为：

- 命令挂住
- pipeline 半阻塞
- 子进程等待下游 I/O

### 14.2 当前建议

继续沿用现有 `runCommandWithEarlyPermissionDetection(...)` 的思路：

1. 同时读取 stdout / stderr
2. 持续累积输出
3. 扫描权限拒绝关键词
4. 一旦命中：
   - `cancel()`
   - `SIGKILL`
   - 立即返回

### 14.3 关键词建议

在现有关键词基础上，WSL 版建议额外关注：

- `read-only file system`
- `permission denied`
- `operation not permitted`
- `access denied`
- `transport endpoint is not connected`
- `bwrap:`

其中：

- `bwrap:` 前缀更偏向 sandbox 启动失败
- 不能简单等同于业务权限拒绝

### 14.4 错误分类

建议把错误分成 5 类：

1. **runner preflight 错误**
   - WSL 不存在
   - 发行版不存在
   - WSL 1
   - 缺少 `bubblewrap`

2. **目录授权错误**
   - 用户取消
   - 弹窗超时
   - 路径非法

3. **sandbox 启动错误**
   - mount 参数错误
   - helper 安装失败
   - `bwrap` 启动失败

4. **sandbox 权限拒绝**
   - 越权读写
   - 越权遍历

5. **命令自身失败**
   - 命令返回非 0
   - 但不是 sandbox 问题

## 15. 与 integration 的会话状态集成

### 15.1 会话存储模型

上层契约应与当前 mac / integration / proxy / cli-get 的实现保持一致：`cli_sandbox_state` 直接保存 chat 维度的完整状态，而不是只记 mode。

```sql
CREATE TABLE cli_sandbox_state (
    chat_id TEXT NOT NULL DEFAULT '',
    sandbox_exe TEXT NOT NULL DEFAULT '',
    allowed_dir TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL,
    PRIMARY KEY (chat_id)
)
```

这意味着：

- `chatId` 是唯一键
- `sandbox_exe` 表示 `filepick` / `net` / `filepick_net`
- `allowed_dir` 表示当前 chat 最近一次成功确认后的目录
- runner / helper 不再维护可作为执行回退的本地目录缓存

### 15.2 为什么目录也必须入库

原因和 mac 完全一致，而且在 WSL 上更明显：

- 用户可能在多个 chat 之间来回切换
- 每个 chat 可以绑定不同目录
- “上一次选中的目录”无法代表“当前这个 chat 应该恢复哪个目录”

尤其在 WSL 下，目录还可能同时有：

- Windows 路径
- Linux 路径
- UNC 路径

如果不把 chat 维度的目录状态落库，串话风险只会更高。当前方案要求数据库保存上层权威目录，并在每次执行时再次显式注入 `--allowed-dir`。

### 15.3 prime 目录授权

当会话切到 `filepick` 或 `filepick_net` 时，integration 仍应先 prime：

- 无 `dir` 时：
  - 强制弹目录选择器
  - 从固定初始目录 `C:\` 开始
- 有 `dir` 时：
  - 直接校验并规范化该目录

只有 prime 成功，才把 `mode + allowedDir` 一起写入 SQLite。

### 15.4 执行时机

执行命令时：

1. `cli-get` 查询当前 chat 的完整状态
2. 解析 WSL runner 路径
3. 调用 Windows `CLI_SANDBOX.exe`
4. 如果状态里有 `allowedDir`，显式追加 `--allowed-dir <dir>`
5. 由它进入目标 WSL 发行版执行 helper

## 16. 目录授权状态机

从用户交互角度看，`filepick` 模式的状态机建议如下：

1. 请求启用模式
2. integration 调 runner prime
3. runner 判断本次是否已显式传入 `--allowed-dir`
4. 如没有，则弹 picker
   - 首次未配置或重新授权时，从 `C:\` 打开
   - 不预加载当前 chat 已保存目录
5. 用户：
   - 允许
   - 取消
   - 超时未处理
6. integration 把 `mode + allowedDir` 一起写入 SQLite
7. helper 使用该目录构造 `bubblewrap` 参数
8. 后续命令按当前 chat 重新取回并传入该目录

其中“取消”分支需要额外满足：

- 一旦用户在首个 picker 中点击取消，本次授权流程立即结束
- 不再继续 fallback 到 PowerShell picker、WSLg picker 或其他后端
- 返回语义按权限拒绝处理，而不是“无可用 picker”或普通执行错误

这样可以避免：

- 关闭第一个 picker 后又自动弹第二个 picker
- 第二个 picker 不在最前台，导致用户误以为窗口丢失
- 目录授权流程在用户已取消的情况下继续占用 60 秒等待时间

失败分支建议包括：

- 用户取消：`已取消目录授权`
- 弹窗超时：`目录授权弹窗超时，请切回 Windows 桌面确认选择窗口后重试`
- 无可用 picker：`当前 WSL 环境不支持目录授权弹窗，请使用 --allowed-dir`
- 锁繁忙：`等待10秒后仍未获得授权`
- 路径非法：`not a directory`

## 17. 测试覆盖建议

建议至少覆盖以下测试：

1. 模式归一化
2. WSL 发行版版本解析
3. helper 安装路径计算
4. Windows 路径 -> Linux 路径转换
5. UNC 路径 -> Linux 路径转换
6. picker fallback 顺序
7. `filepick` 的 bind mount 列表
8. `net` 的 `--unshare-net`
9. `filepick_net` 的组合参数
10. 不把整棵 `/mnt` 带入沙盒
11. `HOME` / `ZDOTDIR` / `XDG_*` 改写
12. 权限拒绝快速失败
13. stdin/stdout 原样透传
14. 首次未配置时从 `C:\` 打开 picker
15. 不依赖本地目录缓存回退
16. prime 成功后才写会话状态

## 18. 安全性评估

### 18.1 该设计真正能提供什么

在推荐架构下，WSL 沙盒可以提供：

- 会话维度的模式开关
- 目录授权前置
- 基于空根文件系统的显式目录开放
- 网络隔离
- 权限拒绝快速失败
- 对 Windows 挂载路径的最小暴露

### 18.2 该设计不提供什么

即使使用 `bubblewrap`，它本身也不是完整安全产品，仍不提供：

- 自动完备的安全策略
- seccomp 策略全集
- SELinux / AppArmor 策略
- cgroup 资源治理
- Windows 宿主侧的完整安全边界

更准确地说：

- `bubblewrap` 提供的是隔离装配能力
- 真正的安全边界强度取决于我们传入的参数

### 18.3 相比 mac 的安全取舍

相对当前 mac 实现，WSL 版的优点是：

- 更容易做到默认不可见
- 白名单语义更清晰

代价是：

- 跨 Windows / WSL 双系统边界更复杂
- interop、`/mnt/*`、UNC 路径都需要额外考虑

## 19. 功能对齐结论：哪些能完全复制，哪些不能

这一节是本文最关键的结论。

### 19.1 可以完整复制的能力

以下能力在推荐架构下可以完整复制：

1. **三模式语义**
   - `filepick`
   - `net`
   - `filepick_net`

2. **会话级开关**
   - `chatId -> sandbox mode + allowedDir`

3. **命令行执行接口**
   - `--cmd`
   - `--timeout`
   - `--shell`
   - `--allowed-dir`

4. **目录授权预热**
   - prime 成功后才写会话状态

5. **chat 独立目录状态**
   - 每次执行按当前 chat 重新注入 `allowedDir`

6. **标准输入/输出语义**
   - 命令输出直接透传

7. **权限拒绝快速失败**
   - 命中拒绝后及时终止

### 19.2 只能做功能等价、不能做平台 1:1 的能力

以下能力只能做到“功能等价”，不能做到“平台实现一模一样”：

1. **目录授权 UI**
   - mac：`NSOpenPanel`
   - WSL：Windows picker 或 WSLg picker

2. **打包产物**
   - mac：`.app`
   - WSL：`CLI_SANDBOX.exe + Linux helper bundle`

3. **日志与状态目录**
   - mac：容器路径 / `~/Library/...`
   - WSL：`%LOCALAPPDATA%` + `~/.config` / `~/.local/state`

### 19.3 明确无法 1:1 复制的能力

以下能力无法在 WSL 上原样复制：

1. **App Sandbox entitlement**
   - 这是 mac 专属能力

2. **security-scoped bookmark**
   - 这是 mac 专属目录授权持久化模型

3. **`.app` + `codesign` + `notarization` 这一整套交付模型**
   - WSL/Windows 不是这条技术栈

4. **在纯 WSL 无 GUI 条件下强制弹目录选择器**
   - 如果没有 Windows runner，也没有 WSLg，就做不到

### 19.4 需要额外前提才能“接近完整复制”的能力

以下能力可以实现，但必须满足额外前提：

1. **严格禁网**
   - 需要：
     - WSL 2
     - `bubblewrap`
     - 推荐独立 sandbox distro
     - 推荐关闭 interop

2. **图形目录选择**
   - 需要：
     - Windows runner 可用
     - 或 WSLg 可用

3. **选择 Linux 原生目录**
   - Windows picker 未必总比 Linux picker 更顺手
   - 实际上常常需要 WSLg picker 或显式 `--allowed-dir`

## 20. 三类不可 1:1 复制能力的详细替代方案

本节讨论的是**未来增强方案**，不属于当前已落地基线。当前基线仍以：

- chat 维度 `mode + allowedDir`
- 无本地目录缓存回退
- picker 首次未配置时从 `C:\` 打开

为准。

这一节不再只给结论，而是给出可直接落地实现的替代设计。

核心原则只有一条：

- **不追求复刻平台专属 API**
- **追求在 WSL 上重建同等级别的控制面、状态面和执行约束**

### 20.1 用“能力凭证 + runner 授权 + helper 验签”替代 App Sandbox / security-scoped bookmark

#### 20.1.1 要替代的到底是什么

mac 侧这两类能力本质上解决的是 3 个问题：

1. **谁批准了这个目录可访问**
2. **这个批准是否可以跨命令复用**
3. **命令执行时如何确认当前访问仍然合法**

WSL 上没有：

- App Sandbox entitlement
- security-scoped bookmark

但这 3 个问题本身仍然可以被重新实现。

因此，WSL 侧建议用以下三件东西替代：

1. **Windows runner 作为唯一授权入口**
2. **本地能力凭证（capability record）作为可复用授权载体**
3. **`bubblewrap` 作为最终强制执行器**

更准确地说：

- “谁批准”由 runner 解决
- “是否可复用”由本地凭证解决
- “执行时是否真的只访问获批目录”由 `bubblewrap` 解决

#### 20.1.2 替代方案的总调用链

建议的授权调用链如下：

1. 用户或 integration 请求启用 `filepick` / `filepick_net`
2. Windows runner 弹目录选择器，或接收显式 `--allowed-dir`
3. runner 做路径归一化：
   - Windows 路径
   - Linux 路径
   - realpath
4. runner 生成**能力凭证**
5. runner 用私钥对凭证签名
6. runner 把凭证写入本地缓存，并同步给 WSL helper 缓存
7. 之后每次执行命令：
   - runner 把凭证传给 helper
   - helper 验签
   - helper 校验模式、路径、过期时间
   - helper 把凭证中的目录挂进 `bubblewrap`

因此，授权状态不再是：

- “系统帮我记住这个目录”

而是：

- “DeepRight runner 为这个目录签发了一张可校验、可过期、可撤销的本地授权凭证”

#### 20.1.3 为什么必须引入“签名凭证”，而不是只缓存明文路径

如果只缓存：

- `selected-dir.txt`
- 或 `selected-dir.json`

会有三个问题：

1. 用户可以手改缓存文件，把路径改成更大的目录
2. helper 无法区分“真的是 runner 授权的”还是“手写塞进去的”
3. 目录缓存无法绑定 mode / distro / 过期时间

因此需要把缓存从“明文路径”升级成“带签名的授权对象”。

#### 20.1.4 能力凭证的数据模型

建议 Windows runner 与 helper 共享一个统一 JSON 结构：

```json
{
  "version": 1,
  "issued_by": "deepright-cli-sandbox",
  "subject": {
    "windows_user": "DOMAIN\\alice",
    "machine": "HOST-01"
  },
  "target": {
    "distro": "deepright-sandbox",
    "mode": "filepick_net"
  },
  "path": {
    "windows_path": "C:\\Users\\alice\\work\\proj",
    "linux_path": "/mnt/c/Users/alice/work/proj",
    "linux_realpath": "/mnt/c/Users/alice/work/proj"
  },
  "constraints": {
    "read_write": true,
    "network": false
  },
  "issued_at": "2026-06-18T10:00:00Z",
  "expires_at": "2026-06-25T10:00:00Z",
  "nonce": "base64-random",
  "signature": "base64-signature"
}
```

字段含义建议如下：

- `version`
  - 便于后续升级 schema
- `issued_by`
  - 固定签发者标识
- `subject`
  - 把凭证绑定到本机当前 Windows 用户和机器
- `target.distro`
  - 防止一个发行版签发的授权被另一发行版复用
- `target.mode`
  - 防止 `filepick` 授权被拿去跑别的高权限模式
- `path.windows_path`
  - 便于 UI 展示与调试
- `path.linux_path`
  - helper 实际使用的逻辑路径
- `path.linux_realpath`
  - 防止 symlink / bind / alias 偏差
- `constraints`
  - 让授权和运行模式语义一致
- `issued_at` / `expires_at`
  - 让凭证天然可过期
- `nonce`
  - 防止简单重放和缓存碰撞
- `signature`
  - 防篡改核心字段

#### 20.1.5 签名机制建议

推荐不要用“helper 也知道私钥”的方案，而是用：

1. runner 持有私钥
2. helper 只持有公钥

推荐实现：

- 首次启动时，runner 在 Windows 本地生成一对 Ed25519 密钥
- 私钥用 Windows 本机、当前用户作用域保护后落盘
- 公钥写入：
  - Windows runner 配置
  - WSL helper 配置

好处是：

1. helper 能独立验签
2. helper 不需要知道私钥
3. 即使 helper 缓存被看到，也不能伪造新的授权记录

#### 20.1.6 私钥保存策略

建议私钥只存在 Windows 侧，并满足以下要求：

1. 仅当前用户可读
2. 仅本机可用
3. 不同步到 WSL
4. 不写入 bundle

推荐落点：

```text
%LOCALAPPDATA%\DeepRight\CLI_SANDBOX\keys\signing.key
```

同时配套一个：

```text
%LOCALAPPDATA%\DeepRight\CLI_SANDBOX\keys\public.key
```

如果私钥丢失、重建或轮换，则之前签发的目录授权全部自然失效。

#### 20.1.7 helper 验签与使用流程

helper 收到凭证后，建议按以下顺序处理：

1. 反序列化 JSON
2. 去掉 `signature` 字段，重组 payload
3. 用本地公钥验签
4. 校验 `version`
5. 校验 `target.distro` 与当前 distro 一致
6. 校验 `target.mode` 与当前模式一致
7. 校验当前时间未超过 `expires_at`
8. 校验 `linux_path` 与 `linux_realpath`
9. 校验路径仍存在、仍是目录
10. 生成 `bubblewrap` mount 列表

任一步失败都直接拒绝执行。

#### 20.1.8 为什么还要校验 `realpath`

因为授权的是：

- 某个具体目录

而不是：

- 一个可任意重新指向的逻辑名称

如果只认逻辑路径，存在以下风险：

1. 用户先授权 `/mnt/c/work/current`
2. 之后把 `current` 改成指向别的目录
3. helper 误把新目录当成旧授权目录

因此建议：

- runner 签发时保存 `linux_realpath`
- helper 执行前重新求一次 realpath
- 两者不一致就要求重新授权

#### 20.1.9 过期与撤销策略

推荐把目录授权设计成：

- **默认可复用**
- **但有限期**

建议默认有效期：

- 7 天

同时提供 3 种撤销方式：

1. 用户主动清除缓存
2. mode 变化自动失效
3. 签名密钥轮换后自然失效

可选增强：

- 如果目录 inode / realpath 漂移，也自动失效

#### 20.1.10 runner 与 helper 的双缓存如何配合

建议：

1. Windows runner 缓存保存完整能力凭证
2. helper 缓存保存最近一次收到的能力凭证副本

执行时优先级：

1. 进程内显式传入凭证
2. helper 本地缓存
3. runner 重新同步

这样可以兼顾：

- runner 主路径
- helper 独立调试

#### 20.1.11 对应当前 mac 的“bookmark 持久授权”语义，WSL 版能接近到什么程度

可以接近到以下语义：

- 用户做过一次授权
- 后续命令可以复用
- 授权可以过期
- 授权可以撤销
- 授权和具体目录强绑定

但不能声称自己实现的是：

- 系统级目录 bookmark

更准确的说法应是：

- **应用级、本机级、可验签的目录授权凭证**

#### 20.1.12 这套替代方案仍然做不到什么

仍然做不到：

1. 让操作系统原生理解这份授权
2. 让别的程序天然继承这份授权
3. 让授权脱离 DeepRight runner 独立存在

所以它是：

- DeepRight 自己的授权模型

而不是：

- Windows/WSL 的系统级授权模型

### 20.2 用“Windows 签名分发链 + helper 摘要校验”替代 `.app` / `codesign` / `notarization`

#### 20.2.1 要替代的到底是什么

mac 的交付链本质上提供 4 类价值：

1. **产物身份可识别**
2. **产物内容未被篡改**
3. **系统能验证它来自谁**
4. **升级时有稳定产物结构**

WSL/Windows 上虽然没有完全同构的 `.app + codesign + notarization`，但这 4 类价值仍然可以被重新拆解后实现。

#### 20.2.2 推荐的交付对象拆分

建议把 WSL 版产物拆成两层：

1. **Windows 分发层**
   - `CLI_SANDBOX.exe`
   - `runner-config.json`
   - `helper-manifest.json`
   - `public.key`

2. **Linux helper 层**
   - `CLI_SANDBOX`
   - `helper.sha256`
   - `VERSION`

这样 runner 是主入口，helper 是被安装到 WSL 的受控执行载荷。

#### 20.2.3 helper manifest 建议结构

建议新增：

```json
{
  "version": "1.0.0",
  "mode": "filepick_net",
  "target_os": "linux",
  "target_arch": "amd64",
  "sha256": "hex-digest",
  "size": 1234567,
  "entrypoint": "CLI_SANDBOX"
}
```

用途：

- runner 在复制 helper 前做本地校验
- 复制到 WSL 后再次做远端校验
- 启动前再校验一次远端 helper 未漂移

#### 20.2.4 推荐的完整信任链

建议信任链按以下顺序建立：

1. Windows 用户启动 `CLI_SANDBOX.exe`
2. Windows 对 `CLI_SANDBOX.exe` 的签名做正常发布者校验
3. runner 读取 bundle 内 `helper-manifest.json`
4. runner 校验 bundle 内 Linux helper 的 `sha256`
5. runner 把 helper 安装到 WSL
6. runner 在 WSL 内再次计算 helper `sha256`
7. 只有摘要一致才允许后续执行

也就是说：

- Windows 可执行文件负责“谁在发这个工具”
- manifest + sha256 负责“发进去的 Linux helper 是不是原样”

#### 20.2.5 为什么 helper 还要额外做摘要校验

因为 helper 最终运行位置不在 Windows bundle 里，而在：

- WSL 文件系统

这带来两个风险：

1. helper 被本地替换
2. helper 被旧版本覆盖

因此摘要校验要至少出现 3 次：

1. 安装前
2. 安装后
3. 执行前

#### 20.2.6 首次安装流程

建议首次安装流程如下：

1. runner 判断目标 helper 不存在
2. runner 读取 manifest
3. runner 校验本地 helper 摘要
4. runner 在 WSL 创建安装目录
5. runner 复制 helper 到临时文件：
   - `CLI_SANDBOX.tmp`
6. runner 在 WSL 计算临时文件摘要
7. 摘要一致后原子 rename：
   - `CLI_SANDBOX.tmp -> CLI_SANDBOX`
8. `chmod +x`
9. 写入 `VERSION`

关键点是：

- 不要边复制边覆盖现有 helper
- 要先写临时文件，再原子替换

#### 20.2.7 升级流程

建议升级时也保持相同逻辑：

1. 对比版本
2. 对比 manifest 摘要
3. 若不同，则走“临时文件 + 校验 + 原子替换”
4. 替换成功后再切换 active version

可选增强：

- 保留最近 2 个 helper 版本
- 出现异常时自动回滚上一个可用版本

#### 20.2.8 如果没有 Windows 代码签名怎么办

如果短期内没有接 Windows 代码签名体系，也仍然建议至少做到：

1. runner 自身随主产品分发
2. helper 全流程 `sha256`
3. 关键配置与 manifest 只读部署

但要明确认识到：

- 这只能保证“包内一致性”
- 不能像已签名的 Windows 可执行文件那样提供明确发布者身份

因此，设计文档应把：

- “推荐”
- “最低可运行”

两档清楚区分。

#### 20.2.9 推荐的两档交付策略

建议分成两档：

1. **最低可运行档**
   - runner 未做平台签名
   - helper 做 sha256 校验
   - 适合内部联调

2. **正式交付档**
   - runner 做 Windows 代码签名
   - helper manifest 固化进发行包
   - 安装、升级、启动前都做摘要校验

#### 20.2.10 这套替代方案能替代到什么程度

可以替代到：

- 有稳定主入口
- 有发布身份
- 有载荷防篡改
- 有升级校验链

但不能说自己实现了：

- mac notarization
- mac `.app` bundle

更准确地说，它实现的是：

- **Windows/WSL 风格的可验证分发链**

### 20.3 用“Windows 强制授权入口 + headless 显式授权模式”替代“纯 WSL 一定能弹图形目录选择器”

#### 20.3.1 这个问题的本质

mac 版的暗含前提是：

- 外层 `.app` 天然处在桌面图形环境里

而 WSL 不具备这个前提。

WSL 可能处于以下任意环境：

1. 从 Windows 桌面应用触发
2. 从 Windows Terminal 触发
3. 从纯 WSL shell 触发
4. 从无 GUI 的远程会话触发
5. 从 CI / service 触发

因此 WSL 版真正要解决的问题不是：

- “如何保证总能弹窗”

而是：

- “如何保证 filepick 类模式总要先完成一次明确授权”

#### 20.3.2 推荐的设计原则

建议坚持以下原则：

1. `filepick` / `filepick_net` 必须有授权目录
2. 没有授权目录时绝不静默放大权限
3. GUI 只是授权手段之一，不是授权本身
4. 没 GUI 时必须要求显式输入目录

也就是说，WSL 版应该强制的是：

- **授权动作**

而不是：

- **弹窗这种具体交互形式**

#### 20.3.3 三种授权入口

建议明确支持 3 条授权入口：

1. **标准 GUI 授权**
   - 由 Windows runner 拉起 Windows picker
2. **Linux GUI 授权**
   - 如果显式启用 WSLg picker，则由 helper 拉起 Linux picker
3. **headless 显式授权**
   - 通过 `--allowed-dir` 或 prime API 显式指定

其中优先级建议为：

```text
explicit path > cached capability > windows picker > wslg picker > fail
```

#### 20.3.4 headless 模式下应该怎么做

在无 GUI 环境下，建议系统不再尝试“假装还能弹窗”，而是直接走：

1. 用户传 `--allowed-dir`
2. runner 或 helper 校验该目录
3. 生成并缓存能力凭证
4. 后续命令复用该凭证

例如：

```bash
CLI_SANDBOX.exe --mode filepick --allowed-dir C:\work\proj
CLI_SANDBOX.exe --mode filepick --cmd "cat hello.txt"
```

对于 integration，也建议支持：

- `primeDirectory(mode, dir)`

而不是在 headless 场景里继续要求 picker。

#### 20.3.5 纯 WSL shell 直跑时的严格策略

如果开发者直接在 WSL 中运行 helper，应采用更严格的策略：

1. 如果已有有效能力凭证：
   - 可以直接执行
2. 如果没有凭证，但传了 `--allowed-dir`：
   - 允许生成新凭证
3. 如果既没有凭证，也没有 `--allowed-dir`：
   - 直接失败

不要在纯 helper 直跑场景里：

- 偷偷放宽成“当前目录默认可访问”
- 或“没有 picker 就自动关闭 filepick 语义”

#### 20.3.6 如何在 Windows runner 中选择 Linux 原生目录

这是 WSL 版最容易让人卡住的点之一。

推荐支持两种方式：

1. 通过 `\\wsl$\<distro>\...` 选择
   - 用户在 Windows picker 中直接选 UNC 路径
2. 通过手工输入 Linux 路径
   - 例如 `/home/alice/work`

runner 收到后统一做：

1. 识别路径类型
2. 映射成 Linux 路径
3. 校验该路径在目标 distro 内存在

#### 20.3.7 为什么不建议把“无 GUI 必须弹窗”作为目标

因为这会把问题变成不可满足的硬约束：

- 无 GUI 本身就意味着没有窗口系统可用

正确目标应是：

- **无 GUI 时，仍强制一次显式授权**

这样既保留了安全语义，也符合 WSL 的实际运行环境。

#### 20.3.8 推荐的错误语义

为了让上层更容易判断，建议区分以下错误：

1. `已取消目录授权`
2. `目录授权弹窗超时，请切回 Windows 桌面确认选择窗口后重试`
3. `当前环境无可用图形目录选择器，请使用 --allowed-dir`
4. `未找到已授权目录，请先授权或显式传入 --allowed-dir`
5. `目录授权凭证已过期，请重新授权`

这样 integration 或 UI 层可以明确知道：

- 是用户取消
- 是 UI 不可用
- 还是授权状态失效

#### 20.3.9 这套替代方案能替代到什么程度

它可以替代到：

- filepick 类模式始终需要授权
- GUI 环境里可以有接近 mac 的点选体验
- headless 环境里也不会退化成无控制执行

但不能说它实现了：

- “无论什么环境都一定弹出图形窗口”

更准确地说，它实现的是：

- **图形优先、显式授权兜底的多入口授权模型**

### 20.4 三类替代方案组合后的整体调用链

当这三套替代方案组合起来后，完整链路应为：

1. 用户请求启用 `filepick` / `filepick_net`
2. Windows runner：
   - 选择目录
   - 归一化路径
   - 生成能力凭证
   - 签名
   - 缓存
3. integration 仅在授权成功后写会话模式
4. 执行命令时，Windows runner：
   - 校验本地签名状态
   - 校验 helper manifest
   - 校验 WSL helper 摘要
5. runner 通过 `wsl.exe` 启动 helper
6. helper：
   - 验签能力凭证
   - 检查过期和 realpath
   - 构造 `bubblewrap` 参数
7. `bubblewrap` 执行 shell 命令
8. stdout/stderr 原样透传

因此，3 个“不可 1:1 复制”的点在 WSL 上不应被理解成：

- “这功能没法做”

而应被理解成：

- “不能沿用 mac 平台专属机制，但可以用 WSL/Windows 本地机制重建等价控制链”

### 20.5 对实现优先级的建议

如果后续分阶段实现，建议顺序如下：

1. **先做第 3 类**
   - 无 GUI 时的显式授权兜底
   - 这是保证可用性的最低成本路径
2. **再做第 1 类**
   - 能力凭证签名与 helper 验签
   - 这是把“目录缓存”升级成“可验证授权”的关键
3. **最后做第 2 类**
   - 完整分发链与升级校验
   - 这是把系统从“能跑”升级到“可正式交付”

### 20.6 需要新增的测试项

在第 17 节测试建议基础上，至少再补以下测试：

1. 能力凭证签名成功
2. 能力凭证签名被篡改时验签失败
3. 能力凭证 mode 不匹配时拒绝执行
4. 能力凭证 distro 不匹配时拒绝执行
5. 能力凭证过期时拒绝执行
6. `linux_realpath` 漂移时拒绝执行
7. helper manifest 摘要校验失败时拒绝安装
8. helper 安装后摘要漂移时拒绝执行
9. 无 GUI 且无 `--allowed-dir` 时明确失败
10. headless `--allowed-dir` 成功生成能力凭证

## 21. 已知限制

### 21.1 WSL 1 不支持

本方案必须以 WSL 2 为目标，不支持 WSL 1。

### 21.2 依赖 `bubblewrap`

当前设计默认不把 `bubblewrap` 静态打包进项目，而是要求目标发行版安装它。

### 21.3 Windows 文件系统路径虽可支持，但不推荐默认作为工作目录

如果用户选的是：

```text
/mnt/c/...
```

则虽然可以 bind 进去，但：

- 性能较差
- 权限语义和 Linux 原生路径存在差异
- 更容易让人误以为整个 Windows 盘都可见

### 21.4 纯 helper 直跑时，图形目录选择能力不完整

如果开发者直接进入 WSL 调 helper，而不是从 Windows runner 进入，那么：

- 有 WSLg 时可以尝试 Linux picker
- 没有 WSLg 时只能依赖 `--allowed-dir`

### 21.5 `bubblewrap` 不是完整安全边界定义器

它能把“文件与网络隔离骨架”搭好，但不会替我们定义全部安全政策。

## 22. 推荐阅读顺序

如果后续要实现这套方案，建议按以下顺序推进：

1. `wsl/launcher/wsl.go`
   - 先把 `wsl.exe` bridge 打通
2. `wsl/launcher/picker_windows.go`
   - 先做 Windows 目录选择器
3. `wsl/helper/backend_bwrap_linux.go`
   - 实现三模式参数拼装
4. `wsl/helper/runtime_mounts_linux.go`
   - 实现最小运行时目录挂载
5. `wsl/helper/main.go`
   - 实现 helper 执行与错误映射
6. `integration/main.go`
   - 接 prime 流程
7. `cli-get/main.go`
   - 接执行入口

## 23. 最终总结

DeepRight 的 WSL 沙盒如果要完整复制当前 mac 的用户能力，最合适的实现不是“单纯在 WSL 里跑一个 `bwrap` 命令”，而是：

- **交互上**：用 Windows runner 复制 mac 外层交互壳
- **状态上**：用 SQLite 按 chat 保存 `mode + allowedDir`，不保留本地目录缓存回退
- **执行上**：在 WSL 发行版中由 helper 动态构造 `bubblewrap` 参数并执行 shell
- **限制上**：用空根文件系统 + bind mount + network namespace 实现目录白名单与禁网

因此，这套方案最准确的技术定义应是：

> 它是一个“以 Windows runner 为交互控制面、以 WSL helper 为执行入口、以 bubblewrap namespace + bind mount 为核心约束手段、以 chat 维度 `mode + allowedDir` 为状态控制面、且不依赖本地目录缓存回退的命令执行沙盒”。

如果只看用户可见功能，mac 当前三模式能力基本都可以在 WSL 上复刻。

如果看平台技术细节，则以下几项无法 1:1 复制：

- App Sandbox entitlement
- security-scoped bookmark
- mac `.app` 交付模型
- 在纯 WSL 无 GUI 条件下强制弹图形目录选择器
