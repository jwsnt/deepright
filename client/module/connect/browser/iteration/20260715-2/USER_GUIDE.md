# 20260715-2 USER_GUIDE

## 适用范围

本次调整仅适用于 Windows WSL / WSL2。macOS、Linux 原生和 Windows 原生的既有 Browser 行为不变。

## browser start

```bash
./browser start --connect-bin /path/to/integration
```

在 WSL 下，`start` 的流程如下：

- 先强制关闭 Windows 宿主机的 Chrome 进程
- 删除 `C:\ProgramData\deepright\chrome_def`
- 从系统 Chrome User Data 重新复制到该目录
- 复制保留 `Network`（含 `Network/Cookies`）以及 `Login Data`、`Local Storage`、`Session Storage`、`IndexedDB`、`WebStorage`、`Service Worker` 等登录态内容；运行锁文件会被清理
- 强制重建的 `instance init` 启动等待上限读取 integration 的 `config.json` 中的 `browser.init_timeout`（单位：秒）；默认配置为 `300`
- 完成后只启动后台 daemon；不会重启 integration，也不会创建、打开或激活新的浏览器页面

## browser instance init

```bash
./browser instance init --agentId agent-a --chatId chat-001
```

在 WSL 下，`init` 始终以有头模式重新拉起对应实例：

- 先尝试执行该实例的 `instance shutdown`，关闭旧 CDP 和旧 Chrome
- 检查映射的 `C:\ProgramData\deepright\chrome_xxx` profile：不存在时从 `chrome_def` 复制；存在时保留并继续使用该 profile
- 无论 profile 是否已存在，都以 `headless=false` 启动新的有头 Chrome；不会因旧 CDP、残留状态或已有 profile 而直接返回成功
- 强制重新创建是内部实现，使用 `DEEPRIGHT_BROWSER_WSL_FORCE_RECREATE`；命令行没有 `--force-recreate` 参数
- 在集成页面中，只有 Chrome 实际启动成功才会显示“完成”按钮；点击按钮会调用 `instance shutdown`
- 也可先手动关闭有头 Chrome 窗口，再点击“完成”；关闭和状态清理仍会返回 `OK`

## browser stop

```bash
./browser stop
```

在 WSL 下，`stop` 会尽力关闭全部受管理实例，之后：

- 删除 `C:\ProgramData\deepright\chrome_def`
- 保留所有 `C:\ProgramData\deepright\chrome_xxx` 目录，保留各实例登录态和配置

## instance shutdown

```bash
./browser instance shutdown --agentId agent-a --chatId chat-001
```

WSL 下的关闭流程会在 Windows 宿主机结束对应 Chrome 并确认 CDP 端口释放。即使窗口已由用户手动关闭，也会清理实例状态并返回 `OK`；不会删除该实例的 `chrome_xxx` profile。

## 推荐验证流程

```bash
./browser start --connect-bin /path/to/integration
./browser instance init --agentId agent-a --chatId chat-001
# 确认有头 Chrome 已出现，并在集成页面点击“完成”
./browser stop
```

预期：`start` 重建包含登录态的 `chrome_def` 且不打开页面；`init` 启动新的有头 Chrome；`stop` 删除 `chrome_def`，保留全部 `chrome_xxx` profile。
