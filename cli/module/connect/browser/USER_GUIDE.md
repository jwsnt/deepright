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
- 手册会明确说明 Windows WSL / WSL2 下，`instance create` / `instance init` 返回的 `profileDir` 固定落在 `C:\ProgramData\deepright\profiles\chats\<chatId>`，同一 Chat 的 Agent 共享该目录
- 手册会明确说明 Windows WSL / WSL2 下，`browser start` 与 `browser stop` 都不会操作 `C:\ProgramData\deepright\chrome_def`，也不会删除 Chat Profile；过期 Profile 仅由后台清理任务删除

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
- 通过 `integration plugins exec` 或 `/api/plugins/exec` 转发 Browser 时，也只有 `browser start` 会被自动补 `--connect-bin`
- `browser instance init`、`browser instance create`、`goto`、`eval` 等非 `start` 命令不会再由 `integration` 偷偷补 `--connect-bin`
- `start` 会先检查并准备 Playwright driver；失败只记日志，不中断后续启动
- 如果配置了 `cookie_path`，`start` 会先执行一次同路径的 `store + fetch` 校验
- `start` 在执行前后都会把一次 `browser instance list` 快照写入 `browser.log`
- `start` 会先清理当前已托管的 CDP 实例状态，再继续 Playwright daemon 的启动
- 在 Windows WSL / WSL2 下，`start` 不会枚举、结束或等待 Windows 宿主机的 Chrome 进程，也不会读取系统 Chrome 的 `User Data` 目录
- `start` 不会创建、删除、刷新或复制 `C:\ProgramData\deepright\chrome_def`，也不会清理任何 Chrome profile 锁或运行态文件
- Browser 忽略 Integration 运行时 `config/config.json` 的 `app-dir`、`app` 和 `agent-dir` 路径覆盖，统一使用固定目录：macOS 使用容器运行目录，WSL 使用 `~/deepright`，原生 Linux/Windows 使用 `config/config.json` 的上级发布目录；插件和 Agent 分别位于该目录的 `plugins`、`agent` 子目录

### 过期 profile 后台清理

主应用 `config/config.json` 可配置 Browser profile 的保留时间与扫描间隔，单位均为小时：

```json
{
  "browser": {
    "init_timeout": 300,
    "clear": 72,
    "scan": 2
  }
}
```

- `browser start` 成功后会启动独立后台清理任务：立即扫描一次，之后每 `browser.scan` 小时扫描一次；不会阻塞插件启动。
- 目录最后修改时间严格早于当前时间减去 `browser.clear` 小时，才会被删除。
- macOS 只检查固定 Agent 目录 `~/Library/Containers/cn.deepright.integration/Data/Library/Application Support/deepright/agent` 下各 Agent 工作目录的直接子目录，并只处理名称为 `chrome_` 加非空后缀的真实目录；WSL 只检查 Windows 宿主机 `C:\ProgramData\deepright\profiles\chats` 下的直接 Chat Profile 目录。
- WSL 中每个直接子目录都代表一个 Chat Profile；普通文件和符号链接均会保留。
- 原生 Linux 与 Windows 原生环境不执行这项后台清理。
- `browser.clear` 或 `browser.scan` 缺失、无效，或主应用配置无法读取时，清理任务会跳过并将原因写入同目录 `browser.log`，不会导致 `browser start` 失败。
- `browser stop` 会停止后台清理任务；`stop` 本身不会额外立即删除 Profile。

### stop

```bash
./browser stop
```

说明：

- `start` 和 `stop` 都会读取 `browser` 同目录下的 `browser_instance.json`
- 除 `instance init` 外的 Browser 后续命令需要的 integration 路径，统一优先从 `browser_runtime.json` 读取
- 除 `browser start` 外，其他 Browser 命令传入 `--connect-bin` 都会立即报错
- `start` 和 `stop` 在执行前后都会把一次 `browser instance list` 快照写入 `browser.log`
- `stop` 会先停止 Playwright daemon
- 然后通过 `instance list` 枚举所有受管 CDP，并逐个调用 `shutdown`
- 单个实例关闭失败只会写 `browser.log`，不会阻断其他实例继续关闭
- 在非 WSL 环境中，`stop` 结束前会清理固定 Agent 目录下所有 `chrome_${port}` 结构的受管实例目录
- `stop` 结束时会 best-effort 删除 Browser 同目录的 `browser_runtime.json`；删除失败只写日志，不会导致 `stop` 失败
- 通过 `integration` 运行时，macOS 会清理 `~/Library/Containers/cn.deepright.integration/Data/Library/Application Support/deepright/agent` 下的这些目录
- `chrome_xxx` 实例目录（macOS）和 Chat Profile（WSL）不会因 `browser stop` 被立即删除；但过期目录仍可能由 `browser start` 启动的后台清理任务按 `browser.clear` 规则删除
- 在 Windows WSL / WSL2 下，`stop` 不会删除 `chrome_def`、Chat Profile 或其内部的锁/运行态文件

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
- 复制时也会跳过纯运行态文件：`RunningChromeVersion`、`SingletonLock`、`SingletonCookie`、`SingletonSocket`、`DevToolsActivePort`
- 在 Windows WSL / WSL2 下，`profileDir` 仍会出现在原有响应报文里，但它的值以 `browser_instance_wsl` 实际返回的 `user-data-dir` 为准
- 在 Windows WSL / WSL2 下，新的 `profileDir` 固定落在 `C:\ProgramData\deepright\profiles\chats\<chatId>`；同一 `chatId` 的所有 Agent 复用同一个实例和 Profile，不同 Chat 相互隔离
- WSL 下 Chat Profile 首次创建为空目录，不会复制系统 Chrome、`chrome_def` 或其他 Profile 的登录态和配置；后续创建会直接复用该目录
- WSL 下不会清理该目录中的 `Singleton*`、`DevToolsActivePort`、`*.lock`、`*-journal` 等锁/运行态文件；已有 profile 重启时也保持原样
- 通过 `integration` 运行时，macOS 固定落在 `~/Library/Containers/cn.deepright.integration/Data/Library/Application Support/deepright/agent/<agentId>/chrome_${port}`

## instance init

```bash
./browser instance init --agentId agent-a --chatId ctrip-home
```

说明：

- 会先检查该 `AgentId + ChatId` 是否已有实例；如果有，则先走一次优雅 `shutdown`
- 然后重新创建一个新的有头 Chrome CDP
- `init` 不接受 `--connect-bin`，且必须先成功执行过 `browser start`；它只使用 Browser 同目录 `browser_runtime.json` 中记录的 integration 路径定位 `config/config.json`，不会从当前目录或 Browser 二进制目录猜测配置位置
- 所有平台的 `init` 都读取 integration `config/config.json` 的 `browser.init_timeout`，单位为秒；字段缺失时默认 `300` 秒
- `config/config.json` 不存在、JSON 解析失败、`browser.init_timeout` 不是正整数或 `browser_runtime.json` 不存在/非法时，`init` 会立即报错；配置校验在关闭旧CDP前完成，因此不会影响既有实例
- `browser.init_timeout` 是整个初始化的总时限，覆盖旧实例关闭、profile准备（非 WSL 包含复制）、Chrome启动和CDP就绪；超时时会终止本次新Chrome并清理本次运行状态，同时保留profile目录
- `init` 会等到新 CDP 真正可用后才写入状态
- 非 WSL 平台初始化时复制系统 Chrome 登录态目录会跳过 `RunningChromeVersion`、`Singleton*`、`DevToolsActivePort` 等运行态文件；WSL 不执行该复制或锁清理
- 在 macOS、Windows、WSL / WSL2 和Linux下，命令会在新的有头 Chrome/CDP 就绪并写入状态后返回；关闭由集成页面的“完成”按钮或 `instance shutdown` 执行
- 后续在 `get` / `list` / `create` / `restart` 重载状态时，会自动清理已经退出的 Chrome/CDP 状态
- 在 Windows WSL / WSL2 下，`init` 会通过插件同目录的 `browser_launcher.sh` 调用 `browser_instance_wsl` 逻辑，以有头模式启动实例
- 在 Windows WSL / WSL2 下，`init` 会先对同一 `chatId` 的受管实例执行一次 `shutdown`，再以 `headless=false` 强制重新拉起有头 Chrome；不会直接复用旧的存活 CDP
- WSL 下如果对应的 Chat Profile 不存在，`init` 会直接创建空目录；Profile 已存在时保留其全部内容（包括锁/运行态文件），但仍会关闭旧实例并重新启动 Chrome
- WSL强制重新创建是 `init` 内部行为，通过 `DEEPRIGHT_BROWSER_WSL_FORCE_RECREATE` 传递；用户无需、也不能使用 `--force-recreate` 参数
- 通过集成页面初始化时，“完成”按钮只会在新的有头 Chrome 启动成功后出现；点击该按钮会调用 `instance shutdown`。用户先手动关闭窗口后再点击，关闭操作仍会返回 `OK`

## instance stop

```bash
./browser instance stop --agentId agent-a --chatId ctrip-home
```

说明：

- 需要显式传入 `--agentId` 和 `--chatId`
- 会从 `browser_instance.json` 中定位实例
- 按 `pid` 结束 Chrome 进程
- 删除对应状态记录
- 在 Windows WSL / WSL2 下，`instance stop` 只会关闭对应 Chat 的共享实例并更新状态；不会清理 Chat Profile 或 profile 锁文件
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
- Windows WSL / WSL2 下，端口确认使用绝对路径的 PowerShell 执行；若 Chrome 已被手动关闭，也会完成状态清理并返回 `OK`
- 在 Windows WSL / WSL2 下，`shutdown` 不会删除 `C:\ProgramData\deepright\profiles\chats\<chatId>`
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
