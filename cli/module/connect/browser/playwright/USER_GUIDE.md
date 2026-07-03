# browser_playwright 使用手册

## 简介

`browser_playwright` 是基于 `playwright-go` 实现的 Go 版 Playwright CLI。

它提供：

- 独立可执行的浏览器自动化 CLI
- 本地守护进程，保证多次命令调用之间共享同一个浏览器会话
- 可被其他 Go 模块复用的服务层 `browserplaywrightsvc`
- `snapshot -> ref -> action` 的交互流程
- 自动通过 Cookie 模块读取 Chrome Cookie，并在导航时按域名注入匹配 Cookie

最终交付二进制为 `browser_playwright`。

当前目录说明：

- `module/connect/browser/playwright` 当前主要保留 Playwright 运行时资产、历史文档与命令说明
- 当前仓库结构下，这个目录已经不是可直接 `go build ./browser/playwright` 或 `go test ./browser/playwright` 的 Go 包
- 当前实际浏览器入口收口到 `module/connect/browser`
- 当前服务层回归入口收口到 `module/connect/browserplaywrightsvc`

## 编译

在 `/path/to/deepright/cli/module/connect` 目录执行当前可用入口：

```bash
go build -o ./browser/browser ./browser
go test ./browserplaywrightsvc ./browser
```

说明：

- 旧的 `go build -o ./browser/playwright/browser_playwright ./browser/playwright` 已不再适用，因为 `./browser/playwright` 当前不再包含 Go 入口文件
- 如果只是查看 `browser_playwright` 的历史命令面与运行时约定，可继续参考本文后续命令说明

首次运行只有在 driver 缺失时才会自动安装 `playwright-go` 所需 driver 和浏览器。

当前默认会按下面顺序确定 driver 目录：

1. 显式传入 `--driver-dir`
2. 应用启动同目录下的 `./playwright/driver`

如果 `./playwright/driver` 已存在且内容完整，运行时会直接 `playwright.Run()`，不会重复调用 `Install()`，也不会再打印下载日志。

## 命令总览

```bash
./browser_playwright help
./browser_playwright start
./browser_playwright stop
./browser_playwright --browser-timeout 45s open [url]
./browser_playwright --timeout 15000 --browser-timeout 30s eval 'new Promise(resolve => setTimeout(resolve, 10000))'
./browser_playwright --session demo eval --code 'document.title'
./browser_playwright open [url] [--headed] [--persistent]
./browser_playwright create --agentId AGENT --chatId CHAT
./browser_playwright attach --cdp=chrome
./browser_playwright attach --cdp=http://127.0.0.1:9222
./browser_playwright goto <url>
./browser_playwright click <ref|selector|text>
./browser_playwright fill <ref|selector|text> <value>
./browser_playwright snapshot
./browser_playwright screenshot
./browser_playwright screenshot --filename page.png
./browser_playwright screenshot --path page.png
./browser_playwright pdf
./browser_playwright tab-list
./browser_playwright tab-new [url]
./browser_playwright cookie-list
./browser_playwright localstorage-list
./browser_playwright requests
```

## 元信息命令

### `name`

输出插件唯一键和展示名：

```bash
./browser_playwright name
```

返回示例：

```json
{
  "key": "browser_playwright",
  "name": "Playwright Browser"
}
```

### `param`

输出常用参数名：

```bash
./browser_playwright param
```

### `command`

输出当前支持的命令列表：

```bash
./browser_playwright command
```

## 守护进程

### 启动

```bash
./browser_playwright start
```

### 停止

```bash
./browser_playwright stop
```

说明：

- 普通业务命令会自动拉起本地 daemon
- daemon 默认监听 `127.0.0.1:18333`
- 状态目录默认在当前目录下的 `./.browser_playwright`
- 诊断日志默认写入当前程序同目录下的 `./browser.log`，按 10MB 单文件滚动，最多保留 4 卷
- daemon 固定优先使用应用启动同目录下的 `./playwright/driver`
- 只有当该目录缺失或内容不完整时，才会触发一次 `Install()`
- 后台 `start` 会把 daemon 进程真正脱离当前前台命令生命周期，不继承临时 `stdout/stderr` 管道
- 在 macOS / Linux 上，daemon 启动时会创建独立会话，避免出现 `start` 后前台命令退出、后台几秒内自灭的问题

## 会话

默认会话名是 `default`，也可以通过 `--session` 指定：

```bash
./browser_playwright --session demo open https://example.com
./browser_playwright --session demo snapshot
./browser_playwright --session demo click e12
```

如果没有显式传入 `--session`，但传入了 `--agentId` 和 `--chatId`，则会自动使用 `agentId@chatId` 作为会话名：

```bash
./browser_playwright --agentId agent-a --chatId ctrip-home eval 'document.body ? document.body.innerText.slice(0, 1000) : ""'
```

说明：

- 显式 `--session` 优先级最高
- 仅当 `--session` 缺失时，才会根据 `agentId + chatId` 自动生成会话名
- 如果 `--session` 本身就是 `agentId@chatId` 形式，则受管实例优先按这个 `session` 拆解得到真实 `agentId + chatId`
- 即使同时又传了不同的 `--agentId` / `--chatId`，也会以 `--session` 拆解结果为准
- `create` 同样遵循这一规则，最终固定使用 `agentId@chatId`
- `--agentId`、`--chatId`、`--session` 在匹配前都会先统一转换为小写
- 例如 `--session AGENT-A@CTRIP-HOME` 最终会按 `agent-a@ctrip-home` 作为真实会话名和实例身份

### 列出会话

```bash
./browser_playwright list
```

### 关闭全部会话

```bash
./browser_playwright close-all
```

## 打开浏览器

```bash
./browser_playwright open https://example.com
./browser_playwright open https://example.com --headed
./browser_playwright open https://example.com --persistent
./browser_playwright open https://example.com --browser firefox
```

常用参数：

- `--timeout`：单次 Playwright 动作超时，单位毫秒，默认 `5000`
- `--navigation-timeout`：导航超时，单位毫秒，默认 `60000`
- `--browser-timeout`：整个 `browser_playwright` 命令的总超时，默认 `120s`，纯数字按秒解析
- `--browser_retry`：连续超时多少次后才回收 daemon，默认 `3`
- `--headed`：有头模式
- `--persistent`：持久化 profile
- `--browser`：`chromium` / `firefox` / `webkit`
- `--cdp`：通过 CDP 连接到已有 Chromium / Chrome
- `--user-agent`：覆盖默认自动注入的 Chrome 标准 UA
- `--profile`：自定义 profile 目录
- `--width` / `--height`：设置视口，默认 `2560x1440`

User-Agent 说明：

- 所有由 `browser_playwright` 代理的 Playwright 命令，默认都会自动补齐标准 Chrome UA
- 默认值固定为 `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36`
- 对 `attach --cdp=...`、`create` 以及受管 `Agent@Chat` 会话，默认 UA 不只体现在 CLI 参数上，还会继续同步到请求头与页面内 `navigator.userAgent`
- 同一条覆盖链路还会把页面内 `navigator.platform` 固定为 Chrome on macOS 常见值 `MacIntel`
- 同一条指纹链会补齐 `navigator.userAgentData` 的 macOS / Chrome 信息，`navigator.maxTouchPoints=5`，`window.screen=2560x1440`，并模拟常见 Apple GPU 的 WebGL vendor / renderer
- Playwright context 会默认继承当前系统时区与语言；在 macOS 上优先读取系统偏好设置，其余环境回退到当前运行环境变量与系统时区链接
- 受管 CDP 会话不再使用 `NewCDPSession(page) + Emulation.setUserAgentOverride` 做逐页 UA 覆盖，避免在 Obscura 场景下触发 Playwright driver 崩溃并出现 `target closed: could not read protocol padding: EOF`
- 当前改为组合使用 `SetExtraHTTPHeaders({"User-Agent": ...})` 与 `AddInitScript` / `page.Evaluate`，既保持 UA 与 `navigator.platform` 兼容性，也避免 attach 后把会话打崩
- 如果显式传入 `--user-agent`，则以显式值为准，不再自动覆盖

`eval` 同时兼容两种写法：

- `./browser_playwright --session demo eval 'document.title'`
- `./browser_playwright --session demo eval --code 'document.title'`

两种写法最终都会发送同一条 Playwright `eval <js>` 指令，便于兼容上层代理调用。

`screenshot` 同时兼容两种输出参数：

- `./browser_playwright screenshot --filename page.png`
- `./browser_playwright screenshot --path page.png`

超时说明：

- `--browser-timeout` 会限制一次 CLI 调用的总耗时，包括自动拉起 daemon、HTTP 请求以及 `create` 命令内部代理的 `browser_instance create`
- `--timeout` 主要影响 `eval`、`click`、`fill` 等单次 Playwright 动作；如果脚本本身会等待较长时间，需要同时调大这个值
- `eval` 现在会在页面内额外包一层超时保护；像 `new Promise(() => {})` 这类永远不 resolve 的脚本，不会再把 daemon 长时间卡死
- `--navigation-timeout` 主要影响 `open`、`goto`、`reload` 等导航类命令
- `/command` 遇到 `connection refused`、`EOF`、`connection reset by peer`、`broken pipe` 这类瞬时错误时，不再固定只重试 3 次，而是会在本次 `--browser-timeout` 的剩余窗口内持续重试
- 单次命令超时时会先直接返回错误，不会立刻终止当前 daemon
- 只有连续超时累计达到 `--browser_retry` 指定次数后，才会回收当前 daemon；默认阈值为 `3`
- 若未指定，默认总超时为 `120s`

## 自动 Cookie 注入

- `open`、`attach/create` 后首次进入页面、`goto`、`tab-new` 等导航类命令会自动尝试注入本机 Chrome Cookie
- 注入逻辑直接复用 Cookie 模块，不需要额外传入命令行参数
- 只会注入与当前页面域名匹配的 Cookie
- 首跳导航会先注入 Cookie，并直接以 `domcontentloaded` 作为 `goto` 完成条件，避免动态站点首跳卡在 `load`
- Cookie 默认使用 120 秒缓存，避免每条命令都重复读取 Chrome 数据库
- 诊断日志会额外记录本次是否命中 Cookie 缓存、是否因 host 已注入而跳过重复注入

## 诊断日志

- 导航类命令会记录 `goto` 的真实耗时毫秒
- Cookie 注入相关日志会记录：
  - 本次是否命中 Cookie 缓存
  - 本次是否因 host 已注入而跳过重复注入
- `event=command` 会记录每次 Playwright 命令的 `start|finish|error` 三阶段，并包含 `session`、`command`、`args`、`flags`
- `event=command` 在完成或失败时还会附带 `elapsedMs`、`elapsed`，完成时会附带 `result` 摘要，失败时会附带 `error`
- `event=cookie` 会额外附带 `action=inject|skip`、`targetHost`、`cachePath`、`cookieCount`、`cookies`、`injected`
- `event=navigation` 会额外附带 `target`、`gotoCostMs`、`gotoCostExact`
- 默认日志文件为程序同目录下的 `browser.log`
- 日志采用分卷滚动：单文件最大 `10MB`，最多保留 `4` 卷

示例：

```bash
./browser_playwright --session agent-a@ctrip-home goto https://www.ctrip.com
./browser_playwright --session agent-a@ctrip-home eval 'document.cookie'
```

## 附加到已有 Chrome

如果目标 Chrome 已启用远程调试端口，可以直接通过 CDP 附加：

```bash
./browser_playwright attach --cdp=chrome
./browser_playwright attach --cdp=http://127.0.0.1:9222
./browser_playwright --session demo attach --cdp=ws://127.0.0.1:9222/devtools/browser/xxx
```

说明：

- `--cdp=chrome` 会默认连接到 `http://127.0.0.1:9222`
- `--cdp=<url>` 支持 `http://`、`https://`、`ws://`、`wss://`
- `attach` 会按当前 `--session` 重建会话并连接到目标 CDP 服务

## create: 先创建实例再 attach

如果希望按 `AgentId + ChatId` 自动创建或复用专属 CDP 服务，可以直接使用：

```bash
./browser_playwright create --agentId demo-agent --chatId chat-001
./browser_playwright --session demo-agent@chat-001 goto https://example.com
```

行为说明：

- `create` 会先调用当前程序同级相对路径的 `../instance/browser_instance get`
- 如果实例已存在，则直接复用
- 如果实例不存在，再自动调用 `../instance/browser_instance create`
- `browser_instance` 返回端口后，`browser_playwright` 会自动按 `ws://127.0.0.1:<port>/devtools/browser` 执行 attach
- `create` 固定使用 `agentId@chatId` 作为 Playwright 会话名，忽略外部传入的 `--session`
- 如果同一个 `AgentId + ChatId` 已经有存活实例，只有在对应端口仍是健康 CDP 服务时才会复用
- 如果旧记录里的进程还活着，但 `/json/version` 已不再返回预期的 CDP 地址，则会由 instance 侧先清理旧实例，再重新创建
- `create` 的后续页面操作与 `attach` 完全一致
- `create` 完成后，受管 session 应可直接继续执行 `eval`、`goto`、`snapshot` 等命令，不应再因 UA 覆盖流程报 `target closed: could not read protocol padding: EOF`
- `create` 在调用前也会先把 `--agentId` 和 `--chatId` 转成小写，因此 `AGENT-A + CHAT-001` 会落成 `agent-a@chat-001`

如需显式覆盖实例管理二进制路径，可附加：

```bash
./browser_playwright create --agentId demo-agent --chatId chat-001 --instance-bin /opt/browser_instance
```

说明：

- 上述 `browser_instance` 外部二进制依赖仅适用于独立 `browser_playwright` 工具
- 最终面向用户的统一入口 `browser` 已将 instance 模块编译在内，不再依赖外部 `browser_instance`

## 自动 CDP 实例接管

除显式 `create` 外，普通 Playwright 命令在满足以下任一条件时，也会自动检查并接管 `browser_instance`：

- 没有传 `--cdp`，但传了 `--agentId` 和 `--chatId`
- 没有传 `--cdp`，但传了形如 `--session agent-a@chat-001`

示例：

```bash
./browser_playwright --agentId agent-a --chatId ctrip-home eval 'document.body ? document.body.innerText.slice(0, 1000) : ""'
./browser_playwright --session agent-a@ctrip-home --agentId ignored --chatId ignored snapshot
```

行为说明：

- 会先执行 `browser_instance get`
- 若不存在则自动执行 `browser_instance create`
- 成功后会自动补齐 `--cdp=ws://127.0.0.1:<port>/devtools/browser`
- 会话名统一落到 `agentId@chatId`
- `get/create` 背后不会再盲目复用“只有进程活着、但 CDP 已失效”的旧实例
- 如果已经显式传入 `--cdp`，则不会触发这套自动接管逻辑
- `--agentId`、`--chatId`、`--session` 会先统一转换为小写，再执行这条接管链路

补充说明：

- `browser_playwright` 作为独立工具时，自动接管仍然依赖 `browser_instance`
- `browser` 收口运行时则会改为调用内嵌的 instance 模块，不走这里的外部进程依赖

## 页面操作

### 导航

```bash
./browser_playwright goto https://playwright.dev
./browser_playwright go-back
./browser_playwright go-forward
./browser_playwright reload
```

### 元素交互

```bash
./browser_playwright click e12
./browser_playwright dblclick "#submit"
./browser_playwright fill e20 "hello"
./browser_playwright hover "Login"
./browser_playwright select "#country" CN
./browser_playwright check e8
./browser_playwright uncheck e8
./browser_playwright drag e10 e18
```

目标支持三种形式：

- `snapshot` 生成的 `e12` 这类 ref
- CSS 选择器
- 文本匹配

## 快照

### 生成快照

```bash
./browser_playwright snapshot
```

输出会包含：

- 当前页面 URL
- 当前页面标题
- 可交互或有意义元素的 ref 列表

示例：

```text
- e1 <a> text="Docs"
- e2 <button> role=button text="Submit"
- e3 <input> name=email type=email
```

之后可直接：

```bash
./browser_playwright click e2
./browser_playwright fill e3 demo@example.com
```

### 指定快照文件

```bash
./browser_playwright snapshot --filename snapshot.json
```

## 截图与 PDF

```bash
./browser_playwright screenshot
./browser_playwright screenshot e2 --filename submit.png
./browser_playwright pdf --filename page.pdf
```

## 标签页

```bash
./browser_playwright tab-list
./browser_playwright tab-new https://example.com
./browser_playwright tab-select 1
./browser_playwright tab-close 1
```

## 键盘和鼠标

```bash
./browser_playwright press Enter
./browser_playwright keydown Shift
./browser_playwright keyup Shift
./browser_playwright mousemove 300 240
./browser_playwright mousedown
./browser_playwright mouseup
./browser_playwright mousewheel 0 600
```

## 存储状态

### 保存状态

```bash
./browser_playwright state-save
./browser_playwright state-save ./auth.json
```

### 载入状态

```bash
./browser_playwright state-load ./auth.json
```

说明：

- 当前实现会先保存状态文件供后续会话复用
- 推荐在固定 `--session` 和 `--persistent` 场景下组合使用

## Cookies

```bash
./browser_playwright cookie-list
./browser_playwright cookie-get sid
./browser_playwright cookie-set sid abc123
./browser_playwright cookie-delete sid
./browser_playwright cookie-clear
```

## LocalStorage / SessionStorage

```bash
./browser_playwright localstorage-list
./browser_playwright localstorage-get token
./browser_playwright localstorage-set token abc
./browser_playwright localstorage-delete token
./browser_playwright localstorage-clear

./browser_playwright sessionstorage-list
./browser_playwright sessionstorage-get token
./browser_playwright sessionstorage-set token abc
./browser_playwright sessionstorage-delete token
./browser_playwright sessionstorage-clear
```

## 控制台与请求日志

```bash
./browser_playwright console
./browser_playwright console error
./browser_playwright requests
./browser_playwright request 3
```

## 对话框

```bash
./browser_playwright dialog-accept
./browser_playwright dialog-accept "typed value"
./browser_playwright dialog-dismiss
```

## 常用示例

### 示例一：打开页面并点击按钮

```bash
./browser_playwright open https://example.com --headed
./browser_playwright snapshot
./browser_playwright click e2
./browser_playwright screenshot --filename after-click.png
```

### 示例二：保存站点状态

```bash
./browser_playwright open https://example.com --persistent
./browser_playwright state-save ./example-state.json
```

### 示例三：查看页面请求

```bash
./browser_playwright open https://playwright.dev
./browser_playwright requests
./browser_playwright request 1
```

## 目录结构

默认状态目录：

```text
./.browser_playwright/
  browser_playwright.pid
  daemon.json
  sessions/
    default/
      session.json
      snapshot.json
      screenshot.png
      page.pdf
      downloads/
      profile/
```

默认日志文件：

```text
./browser.log
```

## 验收建议

```bash
./browser_playwright help
./browser_playwright open https://example.com --headed
./browser_playwright create --agentId demo-agent --chatId chat-001
./browser_playwright snapshot
./browser_playwright click e1
./browser_playwright tab-new https://playwright.dev
./browser_playwright tab-list
./browser_playwright cookie-list
./browser_playwright localstorage-list
./browser_playwright requests
./browser_playwright stop
```
