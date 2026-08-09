# Integration 迭代手册（20260707-3）

## 本次更新

- `打开目录` 在 WSL 场景下不再只停留在“尝试打开”
- `integration` / `proxy` 的 `/api/folder` 会先尝试 `xdg-open`
- 如果 `xdg-open` 失败或无法满足置前需求，会进入 PowerShell 前台化分支
- 前台化分支会先拉起 Explorer，再按目标目录匹配真实窗口并尝试置前
- 如果前台化仍失败，最后再回退到既有 `explorer.exe` / `cmd.exe` 方案
- 页面端在 `打开目录` 成功后会补一个轻提示，提示用户目录已经尝试置前打开

## WSL 顺序

- WSL 下的目录打开链路按以下顺序执行：
  - `xdg-open <path>`
  - PowerShell 隐藏执行前台化脚本
  - 既有 `explorer.exe` / `cmd.exe` 回退
- 非 WSL 的 `linux` 仍保持 `xdg-open`
- `darwin` 仍保持 `open`
- `windows` 仍保持 `explorer`

## 前台化说明

- WSL 路径会先通过 `wslpath -w` 转成 Windows 路径
- 前台化脚本会：
  - `Start-Process explorer.exe -ArgumentList <windows-path>`
  - 通过 `Shell.Application` 枚举当前 Explorer 窗口
  - 使用 `Document.Folder.Self.Path` 与目标目录做路径匹配
  - 命中后取窗口 `HWND`
  - 组合调用 `ShowWindowAsync`、`SetForegroundWindow` 和 `SendKeys('%')` 尝试置前
- 如果脚本未找到对应窗口，或置前失败，会返回非零并继续走回退链路

## 页面提示

- 消息路径浮层的 `打开目录` 成功后，会提示：
  - `目录已尝试置前打开，若仍未看到请查看任务栏`
- 左侧虚拟文件系统的 `打开目录` 也复用同一提示
- 失败场景继续沿用原有错误提示，不会把失败静默吞掉

## 依赖补充

- WSL 安装脚本已把 `xdg-open` 纳入工具检查范围
- 实际通过安装 `xdg-utils` 提供 `xdg-open`
