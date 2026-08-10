# 20260617-2 使用手册

## 目标

本次迭代调整了 `browser` / `browser instance` 在 Windows WSL / WSL2 下的实例生命周期，重点是去掉 `start` 阶段的固定端口 CDP 启动，并统一改为通过 `browser_instance_wsl.go` 管理 WSL 里的 Chrome CDP。

适用范围：

- `browser help`
- `browser start`
- `browser stop`
- `browser instance create`
- `browser instance init`
- `browser instance shutdown`

## help

```bash
./browser help
```

说明：

- `help` 需要覆盖完整插件使用手册
- 手册中应明确 `browser` 同时提供 Playwright 能力和本机 Chrome CDP 实例管理能力
- 在 Windows WSL / WSL2 下，`start` 不再负责启动固定端口、固定目录的生命周期 CDP

## start

```bash
./browser start
```

行为：

- 保留原有插件启动逻辑，包括 Playwright driver 检查、cookie 校验、实例快照日志等
- 在 Windows WSL / WSL2 下，不再额外启动一个指定端口、指定目录的 Chrome CDP 进程
- WSL 下 Chrome 路径仍优先读取：

```bash
integration connect meta-get --key browser
```

返回中的 `meta.chrome`

- 如果 `meta.chrome` 未配置，则回退到默认路径 `/mnt/c/Program Files/Google/Chrome/Application/chrome.exe`

## instance create

```bash
./browser instance create --agentId agent-a --chatId chat-001
./browser instance create --agentId agent-a --chatId chat-001 --headless none
```

行为：

- 保持原有返回结构，成功时继续返回 `agentId`、`chatId`、`port`、`pid`、`cdp`、`profileDir`
- 在 Windows WSL / WSL2 下，不再复制系统 Chrome 的 `User Data` 到受管目录
- WSL 下改为调用 `../20260617-1/browser_instance_wsl.go` 启动或复用 CDP，并解析其 JSON 响应
- 如果脚本没有返回合法 JSON，则本次创建视为失败
- 是否无头启动，沿用与 macOS 版本一致的判断逻辑，优先读取：

```bash
integration connect meta-get --key browser
```

中的 `meta.headless`

- `meta.headless=false/FALSE/False` 时使用有头模式
- `meta.headless=true/TRUE/True` 时使用无头模式
- `meta.headless` 为空时回退到命令行 `--headless`
- WSL 下 Chrome 路径优先读取 `meta.chrome`，缺失时回退到 `/mnt/c/Program Files/Google/Chrome/Application/chrome.exe`
- `profileDir` 继续保留在原响应报文里，其值以 `browser_instance_wsl.go` 实际返回的 `user-data-dir` 为准

## instance init

```bash
./browser instance init --agentId agent-a --chatId chat-001
```

行为：

- 先执行一次 `instance get`
- 如果已存在实例，则先走一次 `shutdown` 优雅关闭旧实例；如果实例不存在，则直接继续
- 在 Windows WSL / WSL2 下，改为调用 `../20260617-1/browser_instance_wsl.go`
- 如果脚本没有返回合法 JSON，则初始化失败
- `init` 在 WSL 下强制使用有头模式
- 端口计算逻辑保持与原实现一致，和 macOS 版本口径相同
- Chrome 路径读取规则与 `start` / `create` 一致，优先取 `meta.chrome`，缺失时回退到 `/mnt/c/Program Files/Google/Chrome/Application/chrome.exe`
- 只有在端口和 CDP 真正可用后，才会把实例写入状态文件
- 写入状态后，命令继续阻塞，直到 Chrome 进程退出或被关闭，再删除状态记录

## stop

```bash
./browser stop
```

行为：

- 仍会先停止 Playwright daemon，再枚举受管实例并逐个关闭
- 在 Windows WSL / WSL2 下，不再删除 `chrome_*` 目录
- 特别是不会删除 `C:\\temp` 下由 WSL Chrome 实例产生的 `chrome_*` 目录

## shutdown

```bash
./browser instance shutdown --agentId agent-a --chatId chat-001
./browser instance shutdown
```

行为：

- 仍按原有流程关闭指定实例或兼容清理历史遗留实例
- 在 Windows WSL / WSL2 下，不再删除 `chrome_*` 目录
- 关闭后的状态清理、端口释放确认、错误返回语义保持原有逻辑不变

## WSL 总结

- `start` 不再创建固定生命周期 CDP
- `create` 和 `init` 统一通过 `browser_instance_wsl.go` 获取或启动实例
- `create` 不再复制系统 Chrome `User Data`
- `init` 在 WSL 下强制有头，并在实例退出后删除状态
- `stop` / `shutdown` 在 WSL 下都不再删除 `chrome_*` 目录
