# 20260617-3 使用手册

## 目标

本次迭代只调整 `browser start` 在 Windows WSL / WSL2 下的宿主机 Chrome 处理逻辑，并补充 `help` / 使用手册说明。

核心目标：

- `start` 时关闭 Windows 当前所有 Chrome 进程
- 刷新 `C:\ProgramData\deepright\chrome_def`
- 刷新成功后回调一次 `integration start`，重新打开新的 Chrome
- 任何 WSL 复制失败都只记录日志，不阻断插件启动

## help

```bash
./browser help
```

说明：

- `help` 需要继续覆盖完整插件手册
- 手册里需要明确 `browser` 同时提供 Playwright 能力和 Chrome CDP 实例管理能力
- 在 WSL 下要额外说明 `start` 会刷新 `C:\ProgramData\deepright\chrome_def`
- 还要说明这个刷新失败只写日志，不会让 `browser start` 退出失败

## start

```bash
./browser start
```

行为：

- 保留原有 `start` 逻辑，包括：
  - Playwright driver 预检查
  - `cookie_path` 的 `store + fetch` 校验
  - 启动前后实例快照写入 `browser.log`
  - 清理当前受管实例后再启动 Playwright daemon
- 如果系统是 Windows WSL / WSL2：
  - 先强制结束当前 Windows 宿主机上所有 `chrome.exe`
  - 包括 `integration start` 已经打开的 Chrome
  - 然后重新生成 `C:\ProgramData\deepright\chrome_def`

`chrome_def` 刷新规则：

- 源目录是 Windows 默认 Chrome `User Data`
- 目标目录固定为 `C:\ProgramData\deepright\chrome_def`
- 如果目标已存在，先删掉再整目录复制
- 复制时沿用现有 Chrome Profile 精简规则：
  - 跳过 `CacheStorage`
  - 跳过 `OptGuideOnDeviceModel`
  - 跳过 `Default/Cache`、`Default/Code Cache`、`Default/GPUCache`、`Default/Network` 等易失缓存目录
  - 保留登录态常用目录，例如 `WebStorage`、`IndexedDB`、`Local Storage`
- 复制后继续删除锁文件：
  - `SingletonLock`
  - `SingletonCookie`
  - `SingletonSocket`
  - `DevToolsActivePort`
  - `lockfile`
  - `*.lock`
  - `*-journal`

失败语义：

- 关闭全部 Chrome 失败：只记日志，不中断 `start`
- 复制 `chrome_def` 失败：只记日志，不中断 `start`
- 清理复制后的锁文件失败：只记日志，不中断 `start`

成功后的追加动作：

- 如果 `chrome_def` 刷新成功，`browser start` 会再尝试执行一次：

```bash
integration start
```

- 目的不是重复启动服务，而是复用主应用自己的 `start` 逻辑重新打开 Chrome
- 如果 `integration` 已经在运行，则由主应用自行判断并直接打开新的浏览器窗口
- 如果这一步失败，同样只写日志，不阻断 `browser start`

## 总结

- WSL 下的 `start` 现在会主动清理所有 Chrome
- 会把 Windows 默认 Chrome Profile 精简复制到 `C:\ProgramData\deepright\chrome_def`
- 成功后会 best-effort 触发一次 `integration start`
- 整个 WSL 刷新链路都是“记录日志但不阻断插件启动”
