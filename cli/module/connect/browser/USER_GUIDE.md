# browser 使用手册

## 简介

`browser` 是 `connect` 体系下的浏览器插件统一入口，对外提供两类能力：

- Playwright 浏览器操控
- 本机 Chrome CDP 实例创建与管理

插件日志默认写入同目录 `browser.log`。

## 常用命令

```bash
./browser help
./browser name
./browser param
./browser command
./browser start --connect-bin /path/to/integration
./browser stop
./browser fetch --cookie_path ./cookies.json
./browser store --cookie_path ./cookies.json
./browser --session agent-a@ctrip-home goto https://www.ctrip.com
./browser --session agent-a@ctrip-home eval 'document.title'
./browser daemon start
./browser instance create --agentId agent-a --chatId ctrip-home
./browser instance init --agentId agent-a --chatId ctrip-home
./browser instance restart --agentId agent-a --chatId ctrip-home
./browser instance stop --agentId agent-a --chatId ctrip-home
./browser instance shutdown --agentId agent-a --chatId ctrip-home
./browser instance list
./browser instance get --agentId agent-a --chatId ctrip-home
```

## help

```bash
./browser help
./browser instance help
```

说明：

- `help` 会覆盖插件入口命令和 `instance` 子命令的完整使用手册
- 手册会明确说明 Windows WSL / WSL2 下，`instance create` / `instance init` 返回的 `profileDir` 固定落在 `C:\ProgramData\deepright\chrome_<随机后缀>`
- 手册会明确说明 Windows WSL / WSL2 下，`browser stop` 和 `browser instance stop` 在原有关闭流程结束后，都会继续处理 `C:\ProgramData\deepright` 下全部 `chrome*` 目录的清理语义

## 插件元信息

`browser name` 固定返回：

```json
{"key":"browser","name":"浏览器"}
```

`browser param` 固定返回：

```json
[{"headless":"选填。默认为true，使用无头浏览器静默访问，也可切换为false开启可视化访问"},{"chrome":"选填。Chrome浏览器地址，默认使用系统路径。"}]
```

说明：

- `headless` 用于控制插件默认浏览器是否无头启动
- `chrome` 用于指定 Chrome 可执行文件路径
- 插件运行日志固定写在 Browser 同目录的 `browser.log`

## 插件生命周期

### start

```bash
./browser start --connect-bin /path/to/integration
```

补充行为：

- `start` 是唯一允许传入 `--connect-bin` 的 Browser 命令
- `start` 成功后，才会把本次 `--connect-bin` 写入 Browser 同目录的 `browser_runtime.json`
- 如果 `start` 失败，不会新写也不会覆盖已有的 `browser_runtime.json`
- `start` 会先检查并准备 Playwright driver；失败只记日志，不中断后续启动
- 如果配置了 `cookie_path`，`start` 会先执行一次同路径的 `store + fetch` 校验
- `start` 在执行前后都会把一次 `browser instance list` 快照写入 `browser.log`
- `start` 会先清理当前已托管的 CDP 实例状态，再继续 Playwright daemon 的启动
- 在 Windows WSL / WSL2 下，`start` 会先强制关闭 Windows 宿主机当前所有 `chrome.exe` 进程，包括 `integration start` 打开的浏览器窗口
- 然后把 Windows 默认 Chrome 的 `User Data` 目录重新复制到 `C:\ProgramData\deepright\chrome_def`
- 如果 `C:\ProgramData\deepright\chrome_def` 已存在，会先删除后再复制
- 复制时沿用实例复制同一套精简规则：过滤 `CacheStorage`、`OptGuideOnDeviceModel`、`Default/Cache`、`Default/Code Cache`、`Default/GPUCache`、`Default/Network` 等易失缓存，同时保留登录态相关目录
- 复制结束后会继续清理 `SingletonLock`、`SingletonCookie`、`SingletonSocket`、`DevToolsActivePort`、`lockfile`、`*.lock`、`*-journal` 等锁文件
- 如果关闭 Chrome 或复制 `chrome_def` 失败，只写 `browser.log`，不会中断 `start`
- 如果 `chrome_def` 刷新成功，`start` 会再尝试执行一次 `integration start`，让主应用重新打开新的 Chrome 窗口

### stop

```bash
./browser stop
```

说明：

- `start` 和 `stop` 都会读取 `browser` 同目录下的 `browser_instance.json`
- Browser 后续命令需要的 integration 路径，统一优先从 `browser_runtime.json` 读取
- 如果 `browser_runtime.json` 不存在，才会按当前 `browser` 二进制路径推断 runtime config
- 除 `browser start` 外，其他 Browser 命令传入 `--connect-bin` 都会立即报错
- `start` 和 `stop` 在执行前后都会把一次 `browser instance list` 快照写入 `browser.log`
- `stop` 会先停止 Playwright daemon
- 然后通过 `instance list` 枚举所有受管 CDP，并逐个调用 `shutdown`
- 单个实例关闭失败只会写 `browser.log`，不会阻断其他实例继续关闭
- `stop` 结束前还会清理 `agent-dir` 下所有 `chrome_${port}` 结构的受管实例目录
- `stop` 结束时会 best-effort 删除 Browser 同目录的 `browser_runtime.json`；删除失败只写日志，不会导致 `stop` 失败
- 通过 `integration` 运行且未显式覆盖 `--agent-dir` 时，macOS 默认会清理 `~/Library/Application Support/deepright/agent` 下的这些目录
- 在 Windows WSL / WSL2 下，`stop` 会在原有关闭流程全部结束后，并发删除 `C:\ProgramData\deepright` 下所有 `chrome*` 目录，包括 `chrome_def`
- 每个目录的删除成功和失败都会写入 `browser.log`
- 任意目录删除失败都不会阻断 `stop` 返回 `OK`

## instance create

```bash
./browser instance create --agentId agent-a --chatId ctrip-home
./browser instance create --agentId agent-a --chatId ctrip-home --headless none
```

说明：

- 返回结果包含 `agentId`、`chatId`、`port`、`pid`、`cdp`、`profileDir`
- `--agentId`、`--chatId` 会先统一转成小写
- 会优先读取 Browser 插件配置里的 `meta.chrome` 和 `meta.headless`
- 如果存在 `browser_runtime.json`，Browser 插件配置读取会优先沿该文件记录的 integration/runtime 定位
- 在 Windows WSL / WSL2 下，当 `meta.chrome` 缺失时优先回退到默认路径 `/mnt/c/Program Files/Google/Chrome/Application/chrome.exe`
- 默认路径不可用时，再退回命令行 `--chrome` 和系统自动探测
- `meta.headless=false/FALSE/False` 时以有头模式启动
- `meta.headless=true/TRUE/True` 时以无头模式启动
- `meta.headless` 为空时退回命令行 `--headless`
- `meta.headless` 解析失败时默认按 `--headless=new`
- 在 Windows WSL / WSL2 下，`create` / `init` 会调用插件同目录的 `browser_launcher.sh`
- 新建实例前，会准备受管的 `--user-data-dir`；在 macOS / Linux / Windows 原生环境里默认是 `<agent workspace>/chrome_${port}`
- `profileDir` 就是本次实例实际使用的受管 `--user-data-dir` 目录
- 在 macOS / Linux / Windows 原生环境里，如果该目录不存在，会基于当前系统 Chrome 的 `User Data` 根目录复制一份精简副本；如果已存在则直接复用
- 复制时会过滤 `CacheStorage`、`OptGuideOnDeviceModel` 和其他易失缓存目录，同时保留 `Default/WebStorage`、`Default/IndexedDB`、`Default/Local Storage`
- 在 Windows WSL / WSL2 下，`profileDir` 仍会出现在原有响应报文里，但它的值以 `browser_instance_wsl` 实际返回的 `user-data-dir` 为准
- 在 Windows WSL / WSL2 下，新的 `profileDir` 固定落在 `C:\ProgramData\deepright\chrome_<随机后缀>`
- 如果这个 WSL 目录是首次创建，则会优先尝试从 `C:\ProgramData\deepright\chrome_def` 复制一份精简副本
- 如果 `chrome_def` 不存在，或复制过程中任意文件失败，则只写 `browser.log` 并回退为空目录，不阻断 `create` / `init`
- 通过 `integration` 运行且未显式覆盖 `--agent-dir` 时，macOS 默认会落在 `~/Library/Application Support/deepright/agent/<agentId>/chrome_${port}`

## instance init

```bash
./browser instance init --agentId agent-a --chatId ctrip-home
```

说明：

- 会先检查该 `AgentId + ChatId` 是否已有实例；如果有，则先走一次优雅 `shutdown`
- 然后重新创建一个新的有头 Chrome CDP
- `init` 会等到新 CDP 真正可用后才写入状态
- 在 macOS 下，写入状态后命令会继续阻塞，直到这个 Chrome/CDP 被关闭，再把状态删掉
- 在 Linux 原生环境里，写入状态后命令立即返回，不再等待这个 Chrome/CDP 被关闭
- 在 Windows WSL / WSL2 下，写入状态后命令会继续阻塞，直到这个 Chrome/CDP 被关闭，再删除状态
- 后续在 `get` / `list` / `create` / `restart` 重载状态时，会自动清理已经退出的 Chrome/CDP 状态
- 在 Windows WSL / WSL2 下，`init` 会通过插件同目录的 `browser_launcher.sh` 调用 `browser_instance_wsl` 逻辑，以有头模式启动或复用实例

## instance stop

```bash
./browser instance stop --agentId agent-a --chatId ctrip-home
```

说明：

- 需要显式传入 `--agentId` 和 `--chatId`
- 会从 `browser_instance.json` 中定位实例
- 按 `pid` 结束 Chrome 进程
- 删除对应状态记录
- 在 Windows WSL / WSL2 下，会在原有 stop 流程结束后，并发删除 `C:\ProgramData\deepright` 下全部 `chrome*` 目录，包括 `chrome_def`
- 每个目录的删除成功和失败都会写入 `browser.log`
- 任意目录删除失败都不会阻断 `stop` 返回 `OK`
- 成功时输出 `OK`

## instance shutdown

```bash
./browser instance shutdown --agentId agent-a --chatId ctrip-home
```

说明：

- 需要显式传入 `--agentId` 和 `--chatId`
- 会直接强制结束对应 Chrome 进程
- 在 Windows WSL / WSL2 下，会额外在 Windows 宿主机侧强制结束该进程并确认端口释放
- 进程关闭后会继续清理状态记录
- 在 Windows WSL / WSL2 下，`shutdown` 不会删除 `C:\ProgramData\deepright\chrome_*` 目录
- 如果删除后 `browser_instance.json` 已为空，会一并删除该状态文件
- 兼容旧版本残留时，不带 `--agentId` / `--chatId` 的 `shutdown` 仍会尝试清理历史固定端口 CDP

## Playwright 能力

除了 `instance` 生命周期子命令外，`browser` 也直接支持 `browser_playwright` 的核心命令，例如：

```bash
./browser open https://example.com --headed
./browser attach --cdp=chrome
./browser --session agent-a@ctrip-home goto https://www.ctrip.com
./browser --session agent-a@ctrip-home eval 'document.title'
./browser --session agent-a@ctrip-home click e12
./browser --session agent-a@ctrip-home snapshot
./browser --session agent-a@ctrip-home screenshot --filename page.png
./browser --session agent-a@ctrip-home requests
```

说明：

- `--session agent@chat` 会自动映射到受管实例
- 通过 `agentId + chatId` 自动建实例时，与 `browser instance create` 使用同一套 Chrome 路径与 Headless 规则
