# 20260605-1 使用手册

## 目标

本次迭代补齐了 `browser instance` 在 Windows WSL2 下的固定生命周期 CDP，以及新的 `init` 命令行为。

适用范围：

- `browser start`
- `browser stop`
- `browser instance create`
- `browser instance init`
- `browser instance shutdown`

## 新增参数

`browser param` 现在固定返回：

```json
["headless", "chrome"]
```

其中：

- `chrome` 用于覆盖 Chrome 可执行文件路径

## WSL2 固定生命周期 CDP

在 Windows WSL2 下，`browser start` 除了原有插件生命周期逻辑外，还会启动一个固定的 Chrome CDP：

```bash
./browser start
```

行为：

- 端口固定为 `29876`
- Chrome 路径优先取 `integration connect meta-get --key browser` 返回的 `meta.chrome`
- 若未配置 `meta.chrome`，退回 `/mnt/c/Program Files/Google/Chrome/Application/chrome.exe`
- User Data 根目录固定为 `/mnt/c/ProgramData/deepright/chrome_base`
- 命令会阻塞到该固定 CDP 退出或被关闭后才返回

关闭方式：

```bash
./browser stop
./browser instance shutdown
```

说明：

- `browser stop` 会先停止 Playwright daemon
- 然后通过 `instance list` 枚举所有受管 CDP，并逐个调用 `shutdown`
- 最后再关闭固定端口 `29876` 的 WSL 生命周期 CDP
- 在 WSL2 下还会清理 `/mnt/c/ProgramData/deepright/agent` 下的实例目录
- 任意单个关闭失败都只写 `browser.log`，不会阻断整体 stop

## instance create

```bash
./browser instance create --agentId agent-a --chatId chat-001
```

行为：

- 端口计算逻辑与历史版本一致，按 `agentId + chatId` 的稳定哈希生成
- WSL2 下实例目录固定为 `/mnt/c/ProgramData/deepright/agent/chrome_${port}`
- `user-data-dir` 已存在则直接复用
- `user-data-dir` 不存在时：
  - WSL2 下复制默认 `/mnt/c/ProgramData/deepright/chrome_base`
  - macOS / Linux / Windows 原生环境下复制当前系统 Chrome `User Data` 根目录
- `meta.headless=false/FALSE/False` 时创建有头实例
- `meta.headless=true/TRUE/True` 时创建无头实例
- `meta.headless` 为空时回退到命令行 `--headless`
- `meta.headless` 解析失败时默认无头 `--headless=new`

## instance init

```bash
./browser instance init --agentId agent-a --chatId chat-001
```

行为：

- 先执行一次 `instance get`
- 只有在已确认该 CDP 已存在时，才执行 `destroy`
- 然后按与 `create` 完全一致的 `chrome`、端口、`user-data-dir` 解析规则启动新的 CDP
- `user-data-dir` 已存在则直接复用；不存在才复制
- `init` 强制有头，启动参数里不会追加任何 `--headless=...`
- 启动参数中的调试地址固定为 `--remote-debugging-address=0.0.0.0`
- 命令会先等待新的 CDP 启动成功，再继续阻塞到该 Chrome/CDP 进程退出或被关闭

## instance shutdown

```bash
./browser instance shutdown --agentId agent-a --chatId chat-001
./browser instance shutdown
```

行为：

- 带 `agentId + chatId` 时，会对指定实例发送 WebSocket `Browser.close`
- 不带 `agentId + chatId` 时，会优先关闭 WSL2 固定端口 `29876` 的生命周期 CDP
- 关闭失败会返回错误，并写入 `browser.log`

## 日志

- 所有生命周期、复制、关闭、同步阻塞退出等事件，统一写到 `browser` 同目录下的 `browser.log`
