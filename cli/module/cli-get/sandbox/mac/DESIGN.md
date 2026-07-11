# DeepRight Mac 沙盒技术白皮书

## 1. 文档目标

本文档描述 DeepRight 在 macOS 上的沙盒实现，覆盖以下内容：

- 当前代码的整体架构
- `.app` 打包、签名与发布方式
- 运行期的调用链
- 目录授权与会话状态管理
- 文件访问权限控制的底层技术手段
- `sandbox-exec` / Seatbelt 规则在本实现中的作用
- 当前实现的边界、风险与已知限制

本文档基于当前仓库实现整理，重点文件包括：

- `cli/module/cli-get/sandbox/mac/launcher/launcher.go`
- `cli/module/cli-get/sandbox/mac/runner/main.go`
- `cli/module/cli-get/sandbox/mac/runner/folderpicker_darwin.go`
- `cli/module/cli-get/sandbox/service/mode.go`
- `cli/module/cli-get/sandbox/service/service.go`
- `cli/module/integration/main.go`
- `cli/module/connect/sandboxstate/sandboxstate.go`
- `cli/module/cli-get/main.go`

## 2. 总体结论

当前 macOS 沙盒实现不是单一技术，而是两层机制叠加：

1. **应用封装层**
   - 把 `CLI_SANDBOX` 封装成可执行的 `.app`
   - 外层 `runner` 负责与桌面交互，例如弹出原生目录面板
   - 内层 `helper` 负责执行真实命令

2. **运行期约束层**
   - 真实的文件/网络权限限制主要依赖 `sandbox-exec` 加载的 **Seatbelt profile**
   - 当前版本中，helper/app 的签名 entitlements 文件为空字典，**没有依赖 App Sandbox entitlement 模型做文件目录白名单**
   - 换句话说，**目录授权 UI 是应用层逻辑，文件访问限制是 `sandbox-exec`/Seatbelt 规则**

因此，当前实现的核心不是：

- `com.apple.security.files.user-selected.read-write`
- security-scoped bookmark
- App Sandbox 原生目录授权持久化

而是：

- 外层 App 原生目录选择
- 上层按 chat 持久化 `mode + allowedDir`
- runner / helper 不维护“最后一次目录”的全局缓存
- 内层命令执行时动态生成 `sandbox-exec` profile

## 3. 术语说明

### 3.1 Seatbelt

Seatbelt 是 macOS 底层的沙盒机制。`sandbox-exec` 最终就是把 profile 交给 Seatbelt。

### 3.2 App Sandbox

App Sandbox 是 Apple 在开发者层暴露的一套 entitlement 模型。它最终也建立在 Seatbelt 之上，但表达方式不同，能力边界由 Apple 预定义。

### 3.3 本项目中的“沙盒”

本项目里“沙盒”一词具体指三件事：

- 会话维度的运行模式：`filepick` / `net` / `filepick_net`
- macOS 上以 `.app` 形式交付的 `CLI_SANDBOX`
- 使用 `sandbox-exec` 动态加载的 Seatbelt 规则

## 4. 运行模式

代码中定义了三种模式，位于 `cli/module/cli-get/sandbox/service/mode.go`：

- `filepick`
  - 需要目录授权
  - 不禁网
- `net`
  - 不需要目录授权
  - 禁网
- `filepick_net`
  - 需要目录授权
  - 禁网

对应常量：

```go
const (
    SandboxModeFilePick    = "filepick"
    SandboxModeNet         = "net"
    SandboxModeFilePickNet = "filepick_net"
)
```

## 5. 架构分层

### 5.1 逻辑分层

从外到内可以分成五层：

1. **integration 会话控制层**
   - 负责把某个 `chatId` 绑定到一个沙盒状态：`mode + allowedDir`
   - 负责触发目录授权预热或手动白名单注入

2. **cli-get 调度层**
   - 执行任务前，根据会话状态决定是否改走 sandbox helper

3. **mac app runner 层**
   - `.app/Contents/MacOS/CLI_SANDBOX`
   - 负责前台 UI、固定初始目录、日志路径、helper 参数透传

4. **sandbox helper 层**
   - `.app/Contents/Helpers/CLI_SANDBOX`
   - 负责实际构造 `sandbox-exec` profile 并启动 shell

5. **Seatbelt 执行层**
   - 由 `/usr/bin/sandbox-exec -p <profile> <shell> -c <cmd>` 生效

### 5.2 组件图

```text
integration / cli-get
        |
        v
CLI_SANDBOX.app/Contents/MacOS/CLI_SANDBOX   (runner)
        |
        v
CLI_SANDBOX.app/Contents/Helpers/CLI_SANDBOX (helper)
        |
        v
/usr/bin/sandbox-exec -p "<dynamic profile>" /bin/sh -c "<cmd>"
        |
        v
macOS Seatbelt
```

## 6. 打包与交付设计

### 6.1 打包入口

打包入口在：

- `cli/module/cli-get/sandbox/mac/main.go`
- `cli/module/cli-get/sandbox/mac/build.sh`

`go run .` 会调用 `launcher.Run(cfg)`。

### 6.2 产物目录结构

每个模式被单独打包为一个 `.app`：

```text
CLI_SANDBOX.app
└── Contents
    ├── MacOS
    │   └── CLI_SANDBOX            # 外层 runner
    ├── Helpers
    │   └── CLI_SANDBOX            # 内层 helper
    ├── Resources
    │   └── runner-config.json     # bundleId + mode
    └── Info.plist
```

这意味着：

- 外层二进制只负责“交互/包装”
- 内层二进制只负责“执行/限制”

`build.sh` 不是只打一个通用产物，而是会分别产出两套目录：

- `release/mac/arm/<mode>/CLI_SANDBOX.app`
- `release/mac/x86/<mode>/CLI_SANDBOX.app`

其中：

- `arm` 对应 `GOARCH=arm64`
- `x86` 对应 `GOARCH=amd64`

### 6.3 构建步骤

`launcher.Run()` 的关键步骤如下：

1. 归一化配置
2. 创建 `.app` 目录树
3. 构建 `./runner` 到 `Contents/MacOS/CLI_SANDBOX`
4. 构建 sandbox helper 到 `Contents/Helpers/CLI_SANDBOX`
5. 生成 `runner-config.json`
6. 生成 `Info.plist`
7. 生成 entitlement plist
8. 对 helper 和 app 分别 `codesign`
9. 使用 `codesign --verify --deep --strict --verbose=2` 做校验

关键代码位于 `launcher.go`：

- 目录布局创建
- `buildGoBinary(runCfg.ModuleDir, "./runner", ...)`
- `buildGoBinary(runCfg.SandboxSrc, ".", helperExecutable, ...)`
- `writeRunnerConfig(...)`
- `writeInfoPlist(...)`
- `codesignHelper(...)`
- `codesignApp(...)`

### 6.4 构建日志落库

`launcher/buildlog.go` 会把打包/验签结果写入模块目录下共享 `data` sqlite 的 `mac_build_log` 表。

- 仅用于记录构建动作与结果，不参与会话级沙盒状态读写
- 连接策略保持保守：
  - `SetMaxOpenConns(1)`
  - `SetMaxIdleConns(1)`
  - `SetConnMaxLifetime(0)`

### 6.3.1 双架构构建与 x86 交叉编译差异

`build.sh` 内部固定调用两次：

1. `build_one_arch "arm" "arm64"`
2. `build_one_arch "x86" "amd64"`

而 `launcher.go` 中的 `buildGoBinary(...)` 只显式传入：

- `GOOS=<target>`
- `GOARCH=<target>`

没有显式设置：

- `CGO_ENABLED=1`
- `CC=o64-clang` 一类的跨架构 C/Objective-C 编译器

这会带来一个很重要的现实效果：

- 在 arm64 macOS 主机上构建 `arm64` runner 时，更容易直接命中 `darwin && cgo` 分支
- 在同一台 arm64 主机上交叉构建 `amd64` runner/helper 时，如果没有额外配置可用的 x86_64 cgo 工具链，就很容易落到 `darwin && !cgo` 分支

因此，当前仓库里的 x86 技术差异，**本质上不是 Seatbelt 规则在 x86 上不同**，而是：

- 目录授权 UI 更容易从原生 Cocoa `NSOpenPanel` 退回到 `osascript choose folder`
- 进而带来前台激活、超时中断、系统授权弹窗呈现方式的差异

这一点是当前代码与打包链路共同决定的，不是文档推测。

### 6.4 runner-config 的作用

`runner-config.json` 只存两项：

- `bundleId`
- `mode`

runner 启动后读取该文件，决定：

- 当前 helper 对应哪种模式
- 默认日志容器目录
- 是否需要目录授权

### 6.5 签名与证书导入

`build.sh` 会：

- 从 `DEEPRIGHT_KEY` 指向的目录导入 `*.cer`、`*.p12`
- 尝试在临时 keychain 中找到可用 identity
- 优先选择 `Developer ID Application`
- 其次选择 `Apple Development`
- 必要时回退到系统 keychain

这是一个“构建期工具链能力”，不是运行期沙盒能力。

## 7. Entitlement 与 App Sandbox 现状

### 7.1 当前实现状态

当前代码中：

```go
func buildAppEntitlements(cfg RuntimeConfig) []plistKV {
    return nil
}

func buildInheritEntitlements() []plistKV {
    return nil
}
```

也就是说：

- app entitlement plist 仍然会生成
- helper entitlement plist 仍然会生成
- 但内容是空 `<dict></dict>`

### 7.2 这意味着什么

当前版本：

- **没有通过 entitlement 打开 App Sandbox 文件访问授权模型**
- **没有使用 `com.apple.security.files.user-selected.read-write` 控制白名单**
- **没有使用 security-scoped bookmark 持久化目录授权**

当前文件访问约束完全落在 `sandbox-exec` 上。

### 7.3 为什么会演进成这样

从仓库历史与当前代码痕迹可以看出，这套实现经历过一次重要调整：

- 早期设计显然预留了 App Sandbox entitlement 开关
  - `Config` 仍保留 `NetworkClient`、`UserSelectedReadWrite` 等字段
  - `USER_GUIDE.md` 仍描述会生成 App Sandbox entitlements
- 当前实现将 `buildAppEntitlements` / `buildInheritEntitlements` 置空
  - 实际原因是为了避免 helper 已在 App Sandbox 中时再调用 `sandbox-exec` 触发冲突
  - 这种冲突在实践中表现为 `sandbox-exec: sandbox_apply: Operation not permitted`

因此，**代码的现状与部分旧文档不再完全一致**。本白皮书以当前代码为准。

## 8. 外层 runner 设计

runner 位于：

- `cli/module/cli-get/sandbox/mac/runner/main.go`

### 8.1 runner 的职责

runner 不是权限控制器，而是：

- 读取模式配置
- 处理目录授权
- 设定固定初始目录
- 清洗 stderr 噪音
- 把参数透传给 helper

### 8.2 runner 启动流程

启动流程如下：

1. `loadRunnerConfig()` 读取 `runner-config.json`
2. 解析命令行参数：
   - `--shell`
   - `--log-file`
   - `--allowed-dir`
   - `--cmd`
   - `--timeout`
3. `embeddedHelperPath()` 解析 app 内 helper 路径
4. 计算默认日志目录
5. 如模式需要目录授权，则执行 `resolveAllowedDirectory(...)`
6. 构造 helper 参数
7. 运行 helper
8. 透传 stdout
9. 清洗 stderr 噪音
10. 继承 helper 退出码

### 8.3 日志目录策略

runner 通过 `containerHomeBase(bundleID)` 推导容器路径：

```text
~/Library/Containers/<bundle-id>/Data
```

默认日志路径为：

```text
~/Library/Containers/<bundle-id>/Data/Library/Logs/CLI_SANDBOX/sandbox.log
```

### 8.4 目录授权解析逻辑

`resolveAllowedDirectory(explicit)` 的行为：

1. 如果传入 `--allowed-dir`
   - 直接校验路径存在且为目录
   - 返回
2. 如果没有 `--allowed-dir`
   - 直接弹出目录选择器
   - 不读取任何 runner / helper 级“最后一次目录”缓存
   - 不预加载当前 chat 已持久化的 `allowedDir`
   - 首次未配置或重新授权时，都从固定初始目录开始
     - `darwin && cgo`：优先使用 `~/Documents`
     - `darwin && !cgo`：优先使用 `~/Documents`
     - 若 `~/Documents` 不存在，则退回系统默认起始位置
   - 目录选择窗口最长等待 **60 秒**
   - 校验路径
   - 返回

当前实现已经移除 runner / helper 本地 `selected-dir.txt` 作为执行回退。
真正驱动 chat 切换与任务执行的目录来源只剩两种：

- 上层针对当前 chat 显式传入的 `allowedDir`
- 当前这一次交互里用户刚刚选中的目录

### 8.5 runner 的目录缓存位置

当前实现不再维护 runner 级目录缓存文件。

### 8.6 环境变量转发策略

当 runner 已经解析到 `allowedDir` 后，会调用：

```go
forwardedEnvironment(strings.TrimSpace(allowedDir) != "")
```

其目的是在“已经完成目录授权”后，**剥离 `CLI_SANDBOX_FORCE_PICK`**，避免内层 helper 再次强制弹窗。

这点非常关键，因为当前目录选择应由外层 app 完成，不能再由内层命令 helper 重复执行。

另外，当前实现里“强制重新授权”已经改为 **pick-only** 设计：

- runner 先独立完成目录选择
- 当本次调用只是为了授权而不是执行命令时，runner 直接返回已选择目录
- 不再通过附带一个短超时的占位命令来间接触发目录弹窗

这样目录授权超时只由目录选择器自己的等待时间控制，当前统一为 **60 秒**。

### 8.7 stderr 噪音清洗

runner 会过滤两类系统噪音：

- `Failure on line 686 in function id scheduleApplicationNotification(`
- `_LSModifyNotification(`

这样做的原因是：

- 原生 Cocoa/LaunchServices 在某些情况下会往 stderr 打系统级提示
- 这些内容不应该污染用户可见的业务错误

## 9. 原生目录选择器实现

### 9.1 实现位置

runner 内有三套目录选择器实现：

- `folderpicker_darwin.go`
  - `darwin && cgo`
  - 原生 Cocoa `NSOpenPanel`
- `folderpicker_darwin_nocgo.go`
  - `darwin && !cgo`
  - 回退到 `osascript choose folder`
- `folderpicker_other.go`
  - 非 macOS 返回 unsupported

helper 的 `service` 包也保留了同名实现文件，用于直接执行 helper 时的目录选择逻辑。两者本质一致，但属于不同模块、不同二进制入口。

当前 runner 层对目录选择器统一施加 **60 秒** 超时：

- `darwin && cgo` 路径下由 `NSOpenPanel` 的关闭调度控制
- `darwin && !cgo` 路径下由 `osascript choose folder` 的 `context.WithTimeout(60s)` 控制

### 9.2 原生面板路径

在 `darwin && cgo` 分支中，调用链为：

1. `runtime.LockOSThread()`
2. `NSApplication sharedApplication`
3. `setActivationPolicy(NSApplicationActivationPolicyRegular)`
4. `finishLaunching()`
5. `activateIgnoringOtherApps:YES`
6. 构造 `NSOpenPanel`
7. `runModal()`

这说明：

- 目录选择在主线程执行
- 外层 `.app` 被显式激活为前台应用
- 这是当前修复“点选目录秒返回已取消”的关键

### 9.3 面板配置

当前面板配置是：

- 不能选文件
- 只能选目录
- 不允许多选
- 不允许面板内新建目录
- 解析 alias / symlink
- Prompt 为“允许”
- Message 为“CLI_SANDBOX 请选择允许访问的目录”
- 初始目录优先为 `~/Documents`
- 不会把当前 chat 已保存的 `allowedDir` 作为面板起点

### 9.4 超时机制

原生面板超时逻辑使用：

- `dispatch_after(...)`
- `NSApp abortModal()`
- `panel orderOut:nil`

达到超时时返回：

```text
目录授权弹窗超时，请切回桌面确认选择窗口后重试
```

### 9.5 取消逻辑

当 `runModal()` 返回 `NSModalResponseCancel` 时，统一返回：

```text
已取消目录授权
```

### 9.6 非 cgo 回退

当交叉编译为 `darwin && !cgo` 时，会退回到：

```applescript
POSIX path of (choose folder with prompt "CLI_SANDBOX 请选择允许访问的目录")
```

这意味着：

- Apple Silicon 本机可走原生 Cocoa 面板
- 某些无 cgo 的 darwin/amd64 构建会走 AppleScript
- 两种实现会存在一定行为差异

### 9.6.1 x86 上的具体技术差异

基于当前代码，x86 相关差异可以明确拆成两层：

1. **目录选择层存在实现差异**
   - `darwin && cgo`：
     - 直接链接 Cocoa
     - 使用 `NSApplication` + `NSOpenPanel`
     - 显式 `activateIgnoringOtherApps:YES`
     - 超时靠 `dispatch_after(...) + abortModal()`
   - `darwin && !cgo`：
     - 不链接 Cocoa
     - 通过 `/usr/bin/osascript` 执行 `choose folder`
     - 超时靠 Go 的 `context.WithTimeout(...)` 杀掉外部进程
     - 取消通过 AppleScript 输出中的 `(-128)` 判定

2. **文件权限限制层没有架构特化差异**
   - 无论外层授权 UI 最终走 Cocoa 还是 AppleScript
   - 一旦拿到 `allowedDir`
   - 后续都仍由同一套 helper/service 代码生成 Seatbelt profile
   - 同样调用 `/usr/bin/sandbox-exec`
   - 同样执行 `deny /Users /Volumes /private` 再对白名单子树放开

也就是说，**x86 上当前最核心的差异是“怎么拿到授权目录”，而不是“拿到目录后如何做文件沙盒”**。

### 9.6.2 为什么 x86 更容易出现“没弹窗/直接超时”

从实现上看，这类现象更容易发生在 `darwin && !cgo` 路径，原因包括：

- 弹窗宿主不再是当前 `.app` 的原生 `NSOpenPanel`，而是 `osascript`
- 超时不是终止一个本进程 modal loop，而是超时后直接取消外部 AppleScript 进程
- 错误面来源从 Cocoa 返回值变成 AppleScript 文本输出，诊断信号更弱

所以在当前代码下：

- arm64 原生构建通常更接近“应用自己拉起原生面板”
- x86 交叉构建更接近“runner 通过 AppleScript 间接请求系统弹窗”

这就是为什么前者与后者在稳定性、可见性、错误表征上会让人感觉不像同一套实现。

## 10. 内层 helper 设计

helper 逻辑入口在：

- `cli/module/cli-get/sandbox/main.go`
- `cli/module/cli-get/sandbox/service/service.go`
- `cli/module/cli-get/sandbox/service/mode.go`

### 10.1 helper 的职责

helper 才是最终的命令执行器，负责：

- 解析模式
- 解析选中目录
- 构造动态 profile
- 调用 `/usr/bin/sandbox-exec`
- 提前识别权限错误并快速返回

### 10.2 helper 的启动模式

helper 支持两类用法：

1. 只设置目录白名单，不执行命令
   - `--allowed-dir /path`
2. 执行命令
   - `--mode`
   - `--cmd`
   - `--timeout`
   - `--shell`

### 10.3 helper 的目录来源优先级

目录来源的优先级如下：

1. `CLI_SANDBOX_ALLOWED_DIR`
2. 否则直接报错

错误文案为：

```text
未找到当前命令的已授权目录，请显式传入 --allowed-dir
```

这意味着当前 helper **不会再读取本地目录缓存，也不会自行再次弹窗**；真正参与命令执行的目录必须由外层 runner / integration / cli-get 按当前 chat 显式重新注入。

### 10.4 helper 的本地目录缓存

当前实现中，helper 已不再维护 `selected-dir.txt`、`selected-dir.lock` 之类目录缓存或锁文件。

### 10.5 helper 的状态目录

`os.UserConfigDir()/CLI_SANDBOX` 仍可能作为运行时依赖路径被 profile 放行，但它不再承担“最后一次目录缓存”的职责。

## 11. 文件访问控制的核心技术手段

这是本文最重要的部分。

### 11.1 实现结论

当前实现的文件访问控制**主要靠 Seatbelt profile，不靠 App Sandbox entitlement 白名单**。

执行关键路径在 `service.go`：

```go
cmd = exec.CommandContext(
    ctx,
    "/usr/bin/sandbox-exec",
    "-p",
    buildSandboxProfile(mode, pickedDir),
    shell,
    "-c",
    task.Cmd,
)
```

### 11.2 profile 生成函数

profile 由 `buildSandboxProfile(mode, pickedDir)` 动态生成。

基础模板为：

```scheme
(version 1)
(allow default)
```

这意味着默认策略不是“全盘 deny”，而是“默认允许”。

### 11.3 `net` / `filepick_net` 的禁网实现

当模式为 `net` 或 `filepick_net` 时，会追加：

```scheme
(deny network*)
```

这是一个纯 Seatbelt 网络规则，不依赖 entitlement。

### 11.4 `filepick` / `filepick_net` 的目录限制实现

当模式需要目录授权时，会追加：

```scheme
(deny file-read* file-write* (subpath "/Users") (subpath "/Volumes") (subpath "/private"))
```

再通过：

```scheme
(allow file-read* file-write* ...)
```

重放开若干路径。

### 11.5 重放开的路径组成

当前实现会放开以下路径：

1. **用户选中的目录**
2. **用户选中目录解析 symlink 后的真实路径**
3. `~/Library/Application Support/deepright`
4. `os.UserConfigDir()/CLI_SANDBOX`
5. 一组运行时系统目录
   - `/private/etc`
   - `/private/dev`
   - `/private/var/select`
   - `/private/var/run`
   - `/private/var/db`
   - `/private/tmp`
   - `/tmp`

### 11.5.1 当前文件权限中的“默认授权目录列表”

这里需要特别区分两类“允许访问”：

1. **显式白名单目录**
   - 这些目录是代码在 profile 中明确追加的
2. **由于 `(allow default)` 仍然可访问的目录**
   - 这些目录不是代码逐条白名单写进去的
   - 而是因为当前 profile 采用了“默认允许，再局部 deny”的策略

#### A. 显式白名单目录

当前代码会明确放开的目录列表如下：

- 用户本次选中的目录
- 用户本次选中目录经 `filepath.EvalSymlinks()` 解析后的真实路径
- `~/Library/Application Support/deepright`
- `os.UserConfigDir()/CLI_SANDBOX`
- `/private/etc`
- `/private/dev`
- `/private/var/select`
- `/private/var/run`
- `/private/var/db`
- `/private/tmp`
- `/tmp`

其中前两项是**会话相关目录**，后面几项是**运行时依赖目录**。

#### D. 不会额外放开的祖先目录

当前实现**不会**为了改善某些命令表现而额外放开：

- 用户选中目录的父目录
- 更高层祖先目录
- 这些祖先目录的 `file-read-metadata`

因此只要命令在运行时显式或隐式触碰：

- `..`
- 祖先目录的条目枚举
- 祖先目录的元数据读取

就仍然会被 Seatbelt 拒绝。这是当前严格目录边界的一部分，而不是 bug。

#### B. 当前默认仍可访问的系统目录

由于 profile 顶部使用了：

```scheme
(allow default)
```

当前实现并没有把整个文件系统先全盘 deny 掉，因此除了显式 deny 的三大区域之外，很多系统路径仍然是默认可访问的。

在当前实现下，常见的默认可访问目录通常包括：

- `/Applications`
- `/Library`
- `/System`
- `/bin`
- `/cores`
- `/dev`
- `/etc`
- `/home`
- `/opt`
- `/sbin`
- `/tmp`
- `/usr`
- `/var`

这几个目录之所以还能访问，不是因为它们被逐条加入了白名单，而是因为：

- 当前只显式 deny 了 `/Users`
- 当前只显式 deny 了 `/Volumes`
- 当前只显式 deny 了 `/private`
- 其它路径仍继承 `(allow default)`

#### C. 当前默认被拒绝的目录根

当前 profile 中被整体拦住的根目录是：

- `/Users`
- `/Volumes`
- `/private`

然后再通过前面的显式白名单，对这些根目录下面的少数子树做重放开。

### 11.6 为什么要放开这些系统目录

因为如果完全只放开用户目录，shell 和常见命令往往无法正常工作。当前白名单中的系统目录用于保证：

- shell 启动
- 临时文件
- 某些系统查询
- 基础运行时依赖

### 11.7 `pickedDirectoryVariants()` 的作用

用户选中的路径会经过：

- `filepath.Clean`
- `filepath.EvalSymlinks`

这样做是为了解决：

- 用户选择 alias/symlink 目录
- 运行期访问真实路径时被误判 deny

### 11.8 这不是字符串级拦截

当前设计并不分析命令字符串，例如：

- 不会特判 `ls /`
- 不会特判 `find /`
- 不会特判 `python -c ...`

限制发生在 syscall/file 访问层。

所以无论命令是：

- `ls`
- `find`
- `node`
- `python`
- 任意 shell pipeline

只要最终触发文件读写，就要经过 Seatbelt 规则判断。

这也意味着：

- `ls -l <allowedDir>` 可能成功
- `ls -la <allowedDir>` 可能因为读取 `..` 而失败

系统不会因为某个命令习惯顺带访问祖先目录，就自动扩大白名单。

### 11.9 当前策略的实际语义

当前 profile 使用的是：

```scheme
(allow default)
```

因此它并不是“只允许访问选中目录，其他一律不可见”的强隔离模型。

它的真实语义更接近：

- 默认允许
- 重点阻断 `/Users`、`/Volumes`、`/private`
- 再对少量显式白名单目录做例外放开

这会带来一个重要结果：

- `ls /` 仍然可以看到 `/Applications`、`/Library`、`/System`、`/usr` 等目录名
- 但 `ls /Users`、`ls /Volumes`、访问 `private` 下多数路径会被拒绝

因此：

- 它能阻止越权访问大量用户数据
- 但不是“目录可见性最小化”的严格设计

### 11.10 当前策略对“穿透”的处理方式

以白名单目录 `/a/b` 为例：

- `ls /`
  - 通常可返回根目录条目
- `ls /Users`
  - 被拒绝
- `ls -R /`
  - 会在允许区域继续递归
  - 进入 `/Users`、`/Volumes`、`/private` 时报 `Operation not permitted`

也就是说，当前实现处理“穿透”的方式是：

- **在 syscall 层阻止进入被 deny 的子树**
- **而不是把整个文件系统做成 only-allow `/a/b`**

### 11.11 为什么没有使用“默认 deny”

当前代码没有把 profile 写成：

```scheme
(deny default)
```

或“默认 deny 文件读写，只放开少量路径”的形式。

原因可以从现有实现推断为：

- 需要优先让 shell 和常见命令稳定运行
- 避免把系统运行依赖白名单补得过于复杂
- 用较少规则先拦住最敏感的用户数据树

这是一种偏工程可用性的折中，而不是最强安全策略。

## 12. 命令执行环境控制

### 12.1 工作目录

当存在 `pickedDir` 时：

- `cmd.Dir = pickedDir`

因此子进程默认从用户授权目录启动。

### 12.2 `ZDOTDIR`

还会额外设置：

```go
extraEnv = append(extraEnv, "ZDOTDIR="+pickedDir)
```

这是一个很有针对性的细节：

- 对 `zsh` 而言，`ZDOTDIR` 会影响其配置文件搜索位置
- 如果不改，shell 启动时可能去访问用户 home 下的 dotfiles
- 在当前 deny `/Users` 的策略下，这些读取会很容易触发权限拒绝

把 `ZDOTDIR` 指到白名单目录，可以降低 shell 初始化阶段的越权访问概率。

### 12.3 shell 选择

默认 shell 取自：

- `$SHELL`
- 否则回退 `/bin/sh`

## 13. 权限错误的快速返回机制

### 13.1 问题背景

如果只是单纯等待子进程退出，权限拒绝可能表现为：

- 命令卡住
- pipeline 半阻塞
- 某些程序等待下游 I/O

### 13.2 当前实现

`runCommandWithEarlyPermissionDetection(...)` 会：

1. 同时读取 stdout/stderr
2. 持续把输出写入 buffer
3. 检查输出中是否出现权限拒绝关键词
4. 一旦命中：
   - `cancel()`
   - 向子进程发送 `SIGKILL`
   - 立即返回

### 13.3 命中的关键词

当前关键词包括：

- `permission denied`
- `operation not permitted`
- `sandbox denied`
- `forbidden by sandbox`
- `权限拒绝`
- `权限不足`
- `没有权限`
- `不允许的操作`

### 13.4 设计效果

这保证了当命令因沙盒规则失败时：

- 不会长时间挂住
- 错误内容能直接透传
- UI/上层 API 可以尽快反馈状态

## 14. 会话状态与 integration 集成

### 14.1 状态存储模型

当前实现里，沙盒状态已经是 **chat 维度的完整状态**，同时记录模式和目录。

SQLite 表：

```sql
CREATE TABLE cli_sandbox_state (
    chat_id TEXT NOT NULL DEFAULT '',
    sandbox_exe TEXT NOT NULL DEFAULT '',
    allowed_dir TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL,
    PRIMARY KEY (chat_id)
)
```

也就是说：

- `chatId` 决定一个会话是否启用沙盒
- `sandbox_exe` 表达当前模式
- `allowed_dir` 表达当前 chat 最近一次确认后的授权目录
- `updated_at` 用于排查切换时序问题

### 14.2 为什么目录要进库

根因是当前产品语义已经不是“helper 进程本地记住一个目录”这么简单，而是：

- 用户会在多个 chat 之间来回切换
- 每个 chat 可以选择不同目录
- 执行命令时必须恢复“这个 chat 自己的目录”，而不是继续沿用 helper 上一次缓存的全局目录

因此，目录如果只放在 `selected-dir.txt` 这类 helper 本地缓存里，就会出现“chat A 选了照片，chat B 选了 code，切回 A 却仍然看到 code”的串话问题。

当前修正后的设计是：

- `cli_sandbox_state` 保存 chat 维度的权威目录
- integration / proxy / cli-get 每次执行都必须把当前 chat 的 `allowedDir` 再次通过 `--allowed-dir` 传给 `CLI_SANDBOX`
- runner / helper 不再保留任何可跨 chat 复用的“最后一次目录”回退

### 14.3 integration 的模式设置

integration 侧有两种入口：

1. CLI
   - `integration sandbox --agentId ... --chatId ... --sandbox filepick`
   - 支持 `--dir`
2. HTTP
   - `/api/sandbox=filepick?...`
   - 支持 `?dir=...`

### 14.4 目录预热（prime）

当设置 `filepick` 或 `filepick_net` 模式时，integration 会先调用 helper 进行“预热”，并把选中的目录写回数据库：

- 无 `dir` 时：触发选目录
- 有 `dir` 时：直接校验并规范化该目录

具体行为：

- `primeIntegrationSandboxMode(mode)`
  - 通过 helper 走 pick-only 授权路径
  - 注入 `CLI_SANDBOX_FORCE_PICK=1`
  - 强制弹窗
  - 返回本次实际选中的目录
- `primeIntegrationSandboxDirectory(mode, dir)`
  - 直接执行 helper `--allowed-dir <dir>`
  - 返回规范化后的目录

只有预热成功之后，integration 才会把 `mode + allowedDir` 一起写入 SQLite。

因此，会话模式切换的设计是：

1. 先确认目录授权可用
2. 再记录该 chat 的完整沙盒状态

这样可以避免数据库里写着 `filepick`，但实际上本地还没拿到目录授权，或者切回旧 chat 时误用了别的 chat 刚刚写入的 helper 缓存。

### 14.5 执行时机

后续当 integration / cli-get 真正执行命令时：

- 先查当前 chat 的完整状态
- 解析对应 helper 路径
- 通过 app bundle 的 `Contents/MacOS/CLI_SANDBOX` 启动沙盒执行
- 如果状态里有 `allowedDir`，则显式追加 `--allowed-dir <dir>`

helper 路径解析支持：

- 直接给 `.app`
- 直接给 `Contents/MacOS/CLI_SANDBOX`
- 给包含不同 mode 目录的父路径

## 15. 目录授权状态机

从用户交互角度看，`filepick` 模式的状态机如下：

1. 请求启用模式
2. integration 调 helper prime
3. runner 判断本次是否已显式传入 `--allowed-dir`
4. 若没有，则弹原生目录面板
   - 面板从固定初始目录 `~/Documents` 开始
   - 不预加载当前 chat 已保存目录
5. 用户：
   - 允许
   - 取消
   - 超时未处理
6. integration 把 `mode + allowedDir` 一起写入 SQLite
7. 后续命令执行按当前 chat 重新取回该目录
8. helper 使用该目录生成 `sandbox-exec` profile

失败分支包括：

- 用户取消：`已取消目录授权`
- 面板超时：`目录授权弹窗超时，请切回桌面确认选择窗口后重试`
- 路径非法：`not a directory`

## 16. 当前实现已移除双路径目录缓存模型

当前实现不再保留：

1. runner 级“最后一次目录”缓存
2. helper/service 级“最后一次目录”缓存

目录的唯一权威来源是：

1. `cli_sandbox_state` 中当前 chat 的 `allowedDir`
2. 本次 prime / 显式 `--allowed-dir` 传入的目录

因此：

- chat 之间不会再通过本地缓存串话
- 没有配置沙盒的 chat 不会因为 helper 的历史选择而继承旧目录
- 重新授权时总是从固定初始目录重新开始

## 17. 测试覆盖说明

当前测试主要覆盖了：

- 普通命令执行
- 超时返回
- 权限拒绝的快速返回
- `filepick` 模式只允许访问已选目录
- profile 中包含 runtime 路径
- chat 维度目录恢复
- 缺失 `--allowed-dir` 时的显式报错
- 目录选择错误透传
- launcher 的构建配置、plist、entitlement 当前状态

这说明实现已经对以下行为有回归保护：

- 不阻塞
- 显式错误返回
- 模式归一化
- 基本文件访问隔离

## 18. 安全性评估

### 18.1 当前实现真正提供了什么

当前实现能够提供：

- 会话维度的模式开关
- 目录授权前置
- 对 `/Users`、`/Volumes`、`/private` 的重点限制
- 网络禁止
- 权限拒绝的快速失败

### 18.2 当前实现没有提供什么

当前实现并不提供：

- App Sandbox 原生目录授权白名单
- security-scoped bookmark 持久授权
- 全文件系统默认 deny
- “除了白名单目录以外什么都看不到”的强隔离语义

### 18.3 风险与取舍

最关键的取舍是：

- 当前 profile 使用 `(allow default)`
- 因而系统目录仍有较大可见面

这是一种：

- 对用户目录较严格
- 对系统目录较宽松
- 偏向工程可运行性

的折中模型。

如果未来要提升隔离强度，首要改造点应是：

1. 将 profile 改为“默认 deny 文件访问”
2. 精细列出 shell/动态库/临时目录的最小运行时依赖
3. 评估是否重新引入 App Sandbox + security-scoped bookmark 路线

## 19. 已知限制

### 19.1 文档与实现存在历史偏差

`USER_GUIDE.md` 中仍描述会生成 App Sandbox entitlements，但当前代码已不再依赖它们做文件权限控制。

### 19.2 x86 构建可能退回 AppleScript

`darwin && !cgo` 构建路径会退回 `osascript choose folder`，与 arm64 原生 Cocoa 面板存在行为差异。

更精确地说：

- 差异主要发生在“目录授权 UI”
- 不主要发生在“Seatbelt 文件访问规则”
- 如果 arm64 主机在没有额外 x86 cgo 工具链的情况下交叉构建 `amd64` 产物，这个差异最容易出现

### 19.3 目录路径未入业务数据库

当前实现中，会话模式和已授权目录都已进入业务数据库；没有任何“仅存在本机缓存文件中”的 chat 级目录状态。

### 19.4 当前 profile 不是最小可见面模型

`ls /`、系统目录遍历等仍可能暴露较多非用户数据路径信息。

## 20. 推荐阅读顺序

如果要继续维护这套实现，建议按以下顺序阅读：

1. `sandbox/service/mode.go`
   - 看模式、chat 状态约束、profile 生成
2. `sandbox/service/service.go`
   - 看实际执行与错误处理
3. `sandbox/mac/runner/main.go`
   - 看外层 app 如何做授权 UI 与参数透传
4. `sandbox/mac/runner/folderpicker_darwin.go`
   - 看原生目录面板
5. `sandbox/mac/runner/picker_defaults.go`
   - 看固定初始目录策略
6. `sandbox/mac/launcher/launcher.go`
   - 看打包与签名
7. `integration/main.go`
   - 看会话状态如何驱动沙盒
8. `connect/sandboxstate/sandboxstate.go`
   - 看模式持久化

## 21. 最终总结

DeepRight 当前 macOS 沙盒实现的本质可以概括为：

- **交互上**：通过 `.app` runner 弹原生目录选择器
- **状态上**：通过 SQLite 按 chat 保存 `mode + allowedDir`，不保留全局最后一次目录缓存
- **执行上**：通过 helper 动态生成 Seatbelt profile，并用 `sandbox-exec` 执行 shell
- **限制上**：重点阻断敏感用户目录与网络，而不是走 App Sandbox 原生目录授权白名单

因此，当前版本最准确的技术定义是：

> 它是一个“以 macOS `.app` 交互壳为入口、以 `sandbox-exec`/Seatbelt 为核心约束手段、以 chat 维度 `mode + allowedDir` 为控制面、且不依赖全局目录缓存的命令执行沙盒”。
