### 第一性原则
+ 仅可以新增/更新/删除integration（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Integration介绍：../../REQUIREMENT.md
+ Integration手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 调整“打开目录”链路在 WSL 场景下的行为：
    + 页面触发 `打开目录` 后，后端不能只做到“尝试打开”
    + 当目录窗口已经被创建但未进入前台时，需要尽量把对应的 Windows Explorer 窗口置前
    + 若强制置前失败，仍需保留原有回退能力，避免把“能打开但不置前”的场景改坏成“完全打不开”
+ 覆盖范围：
    + `integration` 的 `/api/folder`
    + 站点中消息路径浮层的 `打开目录`
    + 左侧虚拟文件系统的 `打开目录`
+ 兼容要求：
    + 普通 macOS / Linux / Windows 的既有打开目录行为不能回退
    + WSL 下仍需先兼容系统默认打开能力；当默认能力不满足“置前”目标时，再进入增强分支
    + 页面端仍保持当前交互方式，不新增重型弹窗
+ 用户感知：
    + 当后端已成功触发打开目录时，前端需要补充轻提示，提示用户目录已尝试置前打开、若未看到可检查任务栏
    + 失败场景继续沿用现有错误提示，不把失败静默吞掉

### 技术实现
+ 收口原则：
    + 目录打开能力继续由 `integration` / `proxy` 的 `/api/folder` 收口
    + 站点前端不直接实现宿主系统置前逻辑，只负责调用接口与展示轻提示
    + 不要求改动消息动作浮层和 VFS 的路径解析规则，只增强打开成功后的宿主行为
+ WSL 打开顺序：
    + `open folder` 在 `linux + WSL` 场景下按以下优先级执行：
        + 先尝试 `xdg-open <path>`
        + 若 `xdg-open` 执行失败，则进入 WSL 前台化分支
        + 若前台化分支也失败，最后再回退到既有 `explorer.exe` / `cmd.exe` 方案
    + 非 WSL 的 `linux` 仍保持 `xdg-open`
    + `darwin` 仍保持 `open`
    + `windows` 仍保持 `explorer`
+ WSL 前台化分支：
    + 目录路径先通过 `wslpath -w` 转成 Windows 路径
    + 通过 PowerShell 隐藏执行前台化脚本：
        + `-NoProfile`
        + `-NonInteractive`
        + `-WindowStyle Hidden`
        + `-ExecutionPolicy Bypass`
        + `-EncodedCommand`
    + PowerShell 可执行文件查找顺序：
        + 优先固定路径
            + `/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe`
            + `/mnt/c/Program Files/PowerShell/7/pwsh.exe`
        + 再回退 `PATH` 中的 `powershell.exe` / `pwsh.exe` / `powershell` / `pwsh`
+ PowerShell 置前脚本要求：
    + 不能只依赖窗口标题做 `AppActivate`
    + 需要先 `Start-Process explorer.exe -ArgumentList <windows-path>`
    + 通过 `Shell.Application` 枚举当前 Explorer 窗口
    + 使用 `Document.Folder.Self.Path` 与目标目录做路径级匹配，找到真正对应的窗口
    + 命中窗口后取其 `HWND`
    + 通过 Win32 API 组合置前：
        + `ShowWindowAsync(hwnd, 9)` 用于恢复并展示窗口
        + `SetForegroundWindow(hwnd)` 用于尝试抢前台
        + `SendKeys('%')` 作为前台切换辅助，降低被系统前台策略拦截的概率
    + 需要保留轮询和短暂重试，避免 Explorer 窗口尚未创建完成时误判失败
    + 若最终未找到对应窗口或置前失败，PowerShell 分支返回非零，交给回退链路继续处理
+ 兼容与回退：
    + 即使增强置前能力失败，也不能影响“目录仍然可被打开”的既有行为
    + 因此 WSL 分支必须保留最终 `explorer.exe` / `cmd.exe` 回退
    + 现有 `/api/folder?path=...` 与 `/api/folder?agentId=...&dir=...` 的请求协议不变
+ 前端提示：
    + 消息路径浮层的 `打开目录` 成功后，补一个轻提示：
        + `目录已尝试置前打开，若仍未看到请查看任务栏`
    + 左侧 VFS 的 `打开目录` 成功后也复用同一提示
    + 轻提示继续使用现有 toast 体系，不新增新组件
+ 安装依赖：
    + Windows / WSL 安装脚本需要补齐 `xdg-open` 所在依赖
    + 实际通过安装 `xdg-utils` 提供 `xdg-open`
    + 安装文案和检测逻辑需要把 `xdg-open` 纳入工具检查集合
+ 测试要求：
    + 需要补充自动化测试，至少覆盖以下情况：
        + WSL 下 `xdg-open` 成功时，不进入前台化与最终回退
        + WSL 下 `xdg-open` 失败时，会先尝试前台化
        + WSL 下前台化成功时，不再进入最终 `explorer.exe` / `cmd.exe` 回退
        + WSL 下前台化失败时，仍会进入最终回退
        + PowerShell 前台化脚本中必须包含 `Shell.Application`、`Document.Folder.Self.Path`、`ShowWindowAsync`、`SetForegroundWindow`
        + 站点中消息路径与 VFS 的成功提示都能走到同一轻提示函数

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写
+ 最小范围更新

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
