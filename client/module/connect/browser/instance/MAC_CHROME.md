# macOS Chrome 实例技术方案

## 文档目的

本文档记录当前 `browser` 插件在 macOS 下管理 Chrome CDP 实例的真实实现，而不是早期设想。

重点覆盖：

- `browser instance create/init/get/list/stop/shutdown/restart` 的当前行为
- macOS 下 Chrome 可执行文件、状态文件、受管 profile 的解析方式
- `chrome_<port>` 的复制、过滤、复用和清理策略
- 与 WSL 版本相比，macOS 方案为什么坚持“稳定端口 + 本地受管 profile”

核心实现文件：

- `cli/module/connect/browser/main.go`
- `cli/module/connect/browser/instance_runtime.go`
- `cli/module/connect/browser/wsl_runtime.go`

## 总体设计

macOS 版本采用“每个 `agentId + chatId` 对应一个稳定端口 + 一个受管 `chrome_<port>` 目录”的方案。

设计目标：

- 同一会话多次 `create`/`restart` 时端口稳定，便于外层依赖按身份定位实例
- 每个受管实例使用独立的 `--user-data-dir`，避免直接共享系统 Chrome 目录带来的锁冲突
- 首次创建时尽量继承系统 Chrome 登录态，但过滤掉大体积、易失缓存，降低创建耗时
- 标准 `shutdown/stop` 只结束进程和移除状态，不默认删除 `chrome_<port>`，以便后续继续复用登录态

## 状态与路径

### 1. 状态文件

实例状态文件默认是浏览器二进制同目录下的 `browser_instance.json`。

也可以通过以下 flag 覆盖：

- `--state`
- `--instance-state`

内部持久化记录字段为：

```json
[
  {
    "agentId": "agent-a",
    "chatId": "chat-001",
    "port": 28412,
    "pid": 12345,
    "cdp": "ws://127.0.0.1:28412/devtools/browser/...",
    "lastActiveAt": "2026-06-17T09:00:00.000000000Z"
  }
]
```

说明：

- `lastActiveAt` 只存在于内部状态文件，不在公开 API 响应里强调
- `profileDir` 不写入 `browser_instance.json`，对外返回时按当前规则动态推导

### 2. Agent Workspace

通过 Integration 运行时，macOS 下受管 profile 使用固定目录：

1. 如果存在 `browser_runtime.json`，优先按其中记录的 integration 读取集成运行时配置
2. 忽略配置中的 `agent-dir`、`app-dir` 和 `app`
3. 固定使用 `~/Library/Containers/cn.deepright.integration/Data/Library/Application Support/deepright/agent`
4. 未关联 Integration 运行时才退回到 `browser` 可执行文件所在目录

最终某个实例的 profile 路径为：

```text
<agent workspace>/chrome_<port>
```

在桌面运行时，这通常会落到类似：

```text
~/Library/Application Support/deepright/agent/<agentId>/chrome_<port>
```

但这只是常见结果，不是代码里的硬编码常量。

### 3. Chrome 可执行文件解析

macOS 下 `browserResolveChromePath` 的解析优先级如下：

1. Browser 元数据 `meta.chrome`
2. 命令行 `--chrome`
3. 命令行历史兼容别名 `--obscura`
4. 系统候选路径自动探测

macOS 候选路径当前为：

- `/Applications/Google Chrome.app/Contents/MacOS/Google Chrome`
- `/Applications/Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing`
- `/Applications/Chromium.app/Contents/MacOS/Chromium`

如果 `meta.chrome` 存在但不是绝对路径，命令会直接报错，而不是静默回退。

### 4. 系统 Chrome User Data 根目录

首次克隆 profile 时，系统 User Data 根目录按以下候选顺序查找：

- `~/Library/Application Support/Google/Chrome`
- `~/Library/Application Support/Google/Chrome for Testing`
- `~/Library/Application Support/Chromium`

找到第一个存在的目录后即作为克隆源。

## 端口策略

macOS 使用稳定哈希端口。

实现要点：

- 输入先做 `trim + lower-case` 归一化
- 使用 `sha256(agentId + "\n" + chatId)` 计算稳定值
- 端口范围固定在 `20000-65535`
- 同一个 `agentId + chatId` 每次计算结果一致

影响：

- `create` 和 `restart` 正常情况下会回到同一个端口
- 如果状态文件丢失，但旧 Chrome 仍占着这个稳定端口，新的 `create` 会因为端口不可用而失败
- macOS 不像 WSL 那样有额外的 SQLite 恢复层，`browser_instance.json` 是主要的公开状态源

## create 流程

`browser instance create --agentId ... --chatId ...` 在 macOS 下的主流程如下。

### 1. 归一化身份并加载状态

- `agentId`、`chatId` 统一转小写
- 读取 `browser_instance.json`
- 执行状态归一化：
  - 清理无效记录
  - 清理已死亡进程
  - 清理已不再可探活的 CDP 端口
  - 清理空闲超时实例

空闲超时默认 `10` 分钟，可通过：

- `--browser_expired`
- `--instance-browser_expired`

覆盖。

### 2. 直接复用健康实例

如果状态文件里已经存在该 `agentId + chatId` 的健康实例：

- 不重新启动 Chrome
- 刷新 `lastActiveAt`
- 直接返回当前 `port/pid/cdp/profileDir`

### 3. 分配稳定端口并解析 profile 目录

当没有可直接复用的健康实例时：

- 计算稳定端口
- 校验该端口没有被现有受管实例占用
- 校验本地端口当前可监听
- 解析实例专属目录 `<agent workspace>/chrome_<port>`

### 4. 准备 `--user-data-dir`

默认模式是 `clone`，也就是：

- 如果 `chrome_<port>` 已存在，直接复用
- 如果不存在，则从系统 Chrome User Data 根目录复制一份“过滤后的副本”

另有一个实现级开关：

- `--user-data-mode direct`

在 `direct` 模式下不会做克隆，只会确保目录存在并清理锁文件。当前主流程默认仍是 `clone`。

### 5. 复制过滤策略

复制入口：

- `browserPrepareChromeUserDataDir`
- `browserCopyDirectoryWithProgress`
- `browserShouldSkipClonedChromeUserDataPath`

复制时会主动跳过：

- 任意层级的 `CacheStorage`
- `OptGuideOnDeviceModel`
- `Default/Cache`
- `Default/Code Cache`
- `Default/GPUCache`
- `Default/GrShaderCache`
- `Default/GraphiteDawnCache`
- `Default/DawnGraphiteCache`
- `Default/DawnWebGPUCache`
- `Default/ShaderCache`
- `Default/Media Cache`
- `Default/Network`
- `Default/Safe Browsing Network`
- `Default/Shared Dictionary`
- `Default/GCM Store`
- `Default/blob_storage`
- `Default/Service Worker/ScriptCache`

因此下列登录相关数据通常会被保留：

- `Cookies`
- `Login Data`
- `Preferences`
- `Default/WebStorage`
- `Default/IndexedDB`
- `Default/Local Storage`

这就是当前 macOS 版本的平衡点：

- 不直接复用系统正在使用的 User Data
- 尽量保留登录态
- 尽量不复制会快速膨胀的缓存与本地模型目录

### 6. 复制后清理

不管是复用已有 `chrome_<port>`，还是新克隆出来的目录，都会做一次锁文件清理。

清理目标包括但不限于：

- `SingletonLock`
- `SingletonCookie`
- `SingletonSocket`
- `LOCK`
- `LOCKFILE`
- `DevToolsActivePort`
- 若干 `.lock` / `-journal` 类条目

### 7. 启动 Chrome

`create` 默认按 headless 模式启动。

headless 解析优先级：

1. `--headless-force`
2. Browser 元数据 `meta.headless`
3. 命令行 `--headless`
4. 默认值 `new`

可接受值：

- `new`
- `none`

Chrome 启动参数核心包含：

- `--remote-debugging-port=<stable port>`
- `--remote-debugging-address=127.0.0.1`
- `--user-data-dir=<profile dir>`
- `--no-first-run`
- `--no-default-browser-check`
- `--disable-background-networking`
- `--disable-sync`
- `--disable-component-update`
- `about:blank`

当模式不是 `none` 时，会额外追加：

- `--headless=<mode>`

### 8. 就绪探测与落盘

启动后会：

- 先等待端口 ready
- 再请求 `/json/version`
- 解析 `webSocketDebuggerUrl`
- 生成最终 CDP 地址
- 将实例写入 `browser_instance.json`

成功后对外返回：

```json
{
  "agentId": "agent-a",
  "chatId": "chat-001",
  "port": 28412,
  "pid": 12345,
  "cdp": "ws://127.0.0.1:28412/devtools/browser/...",
  "profileDir": "/abs/path/to/agent-a/chrome_28412"
}
```

## init 流程

`browser instance init` 与 `create` 的区别主要有两点。

### 1. 先关旧实例

`init` 会先尝试：

- `get`
- `shutdown`

如果旧实例存在，先强制结束；如果不存在，则继续创建。

### 2. 总是有头模式

macOS 下 `init` 不走 `create` 的 headless 解析逻辑，而是直接按有头模式启动，不追加 `--headless=*`。

它仍然复用同一个 `chrome_<port>` 目录：

- 已存在则直接复用
- 不存在则按 `create` 的克隆逻辑补齐

实现上使用 attached 方式拉起 Chrome，但当前 CLI 在实例 ready 后就返回，不会持续阻塞到窗口关闭。

## get / list / restart / shutdown / stop

### get / list

`get` 和 `list` 在返回前都会先重载状态并做归一化清理。

行为包括：

- 移除已死亡实例
- 移除不可探活的 CDP 记录
- 释放超时空闲实例
- 重新计算最新 CDP 地址
- 为返回结果补出 `profileDir`

其中 `get` 还会刷新该实例的 `lastActiveAt`。

### restart

`browser instance restart` 的当前实现等价于：

1. `shutdown`
2. `create`

由于 macOS 使用稳定哈希端口，所以通常会重新回到同一个端口和同一个 `chrome_<port>` 目录。

### shutdown / stop

两者都会结束当前实例，但实现层级不完全相同。

`browser instance stop`：

- 只依赖当前 `browser_instance.json`
- 找到记录后结束 Chrome 进程
- 删除该条公开状态
- 不默认删除 `chrome_<port>`

`browser instance shutdown` 和顶层 `browser shutdown --agentId ... --chatId ...`：

- 走更完整的 destroy 路径
- 除了正常结束实例，还支持缺失状态时的 fallback 清理
- 仍然不默认删除 `chrome_<port>`
- 只有在内部清理标记开启时，才会额外删除 profile 或整批受管目录

因此标准用户视角下两者都能“关实例”，但：

- `stop` 更偏向“按当前公开状态直接停”
- `shutdown` 更偏向“带兜底和内部清理能力的正式销毁入口”

## 返回值里的 `profileDir`

macOS 下 `profileDir` 来自规则推导，而不是状态文件直存：

1. 根据 `agentId + port` 重新解析 `<agent workspace>/chrome_<port>`
2. 对外返回绝对路径

因此：

- `browser_instance.json` 里不必存 `profileDir`
- 只要 `agentId` 和 `port` 还在，CLI 就能稳定推导出同一个目录

## 与 WSL 方案的关键差异

macOS 和 WSL 最大的差异在于状态与 profile 管理方式：

- macOS 使用稳定哈希端口
- macOS 首次创建时会克隆系统 Chrome User Data
- macOS 的 profile 路径由 `agent workspace + chrome_<port>` 规则稳定推导
- macOS 没有额外的 SQLite 作为恢复层
- macOS 的标准 stop/shutdown 不删除 profile，而是留给下次复用

## 风险与边界

### 1. 首次克隆后不会自动和系统 Chrome 再同步

`chrome_<port>` 一旦创建成功，后续默认直接复用。  
这意味着后续系统 Chrome 中新增的登录态，不会自动同步进已有受管目录。

### 2. 某些站点可能依赖被过滤掉的缓存

如果站点把关键状态写在 `CacheStorage` 一类目录里，首次进入时可能需要重新拉资源或重新生成局部状态。

### 3. 状态文件丢失但端口仍被占用时，create 会失败

因为 macOS 依赖稳定端口和公开状态文件，没有像 WSL 那样的额外恢复数据库。

### 4. `profileDir` 是推导值，不是持久化真值

Integration 运行时配置里的 `agent-dir`、`app-dir` 或 `app` 变化不会影响同一条记录对外展示的 `profileDir`。

## 排查建议

遇到 macOS 实例问题时，优先检查：

1. `browser_instance.json` 中的 `port/pid/cdp/lastActiveAt` 是否合理
2. 目标 `chrome_<port>` 是否存在且权限正常
3. 该目录下是否残留 `Singleton*` 或 `DevToolsActivePort`
4. 系统 Chrome User Data 根目录是否存在
5. `meta.chrome` 是否错误地配置成了不存在或非绝对路径
6. 当前稳定端口是否被别的进程占用

## 建议验证用例

建议至少做以下回归：

1. 首次 `create`，确认能从系统 Chrome 继承登录态
2. 二次 `create`，确认直接复用旧实例或旧 profile
3. `shutdown` 后再 `create`，确认端口和 `chrome_<port>` 保持稳定
4. `init`，确认旧实例会先关闭并以有头模式重建
5. 目录检查，确认缓存过滤仍生效且登录相关目录仍保留
