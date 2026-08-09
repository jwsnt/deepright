# WSL Chrome 实例技术方案

## 文档目的

本文档记录当前 `browser` 插件在 Windows WSL / WSL2 下管理 Chrome CDP 实例的真实实现。

WSL 版本已经不再沿用 macOS 的“稳定端口 + 克隆系统 Chrome User Data”方案，而是切换为：

- `browser instance` 公共 CLI 层
- `browser_launcher.sh` 启动器层
- `browser __wsl-instance acquire` 内置命令层
- `browser_data` SQLite 恢复层

这四层共同完成实例的复用、恢复、新建和 profile 持久化。

核心实现文件：

- `cli/module/connect/browser/main.go`
- `cli/module/connect/browser/instance_runtime.go`
- `cli/module/connect/browser/wsl_launcher_runtime.go`
- `cli/module/connect/browser/wsl_instance_runtime.go`
- `cli/module/connect/browser/wsl_runtime.go`
- `cli/module/connect/browser/instance/browser_launcher.sh`

## 为什么 WSL 方案与 macOS 不同

当前 WSL 方案的核心判断是：

- 真正运行的是 Windows 宿主机上的 Chrome
- `--user-data-dir` 使用 Windows 路径更稳定
- 直接复制 WSL 侧的系统 Chrome User Data 性价比低，而且容易引入路径和锁问题
- WSL 下更适合把 profile 持久化为 `C:\temp\chrome_xxxx`，并让 Chrome 自行分配首个可用调试端口

因此当前实现做了两个关键切换：

- 新建实例时不再克隆系统 Chrome User Data
- 首次新建实例时不再强制使用 macOS 那套稳定哈希端口，而是让 Chrome 用 `--remote-debugging-port=0` 动态分配

## 整体架构

### 1. 公共 CLI 层

外部仍通过统一命令访问：

- `browser instance create`
- `browser instance init`
- `browser instance get`
- `browser instance list`
- `browser instance stop`
- `browser instance shutdown`
- `browser instance restart`

这层仍然维护公共状态文件：

- `browser_instance.json`

它负责：

- 向外提供统一 JSON 输出
- 管理 `list/get` 视图
- 做空闲超时清理
- 记录当前活跃实例的 `agentId/chatId/port/pid/cdp/lastActiveAt`

### 2. Launcher 层

当系统检测为 WSL 时，`create` 和 `init` 不直接本地拉起 Chrome，而是调用：

- `browser_launcher.sh`

脚本本身非常薄，只做两件事：

1. 在脚本同级或上一级目录找到 `browser` 二进制
2. `exec "$BROWSER_BIN" __wsl-instance acquire ...`

因此 launcher 的职责是提供一个稳定的 shell 入口，方便 CLI 层统一调用。

### 3. 内置 WSL Acquire 层

`browser __wsl-instance acquire` 的实现位于：

- `wsl_instance_runtime.go`

这层负责真正的 WSL 实例管理，包括：

- 查询/更新 SQLite
- 探活已有实例
- 重启失活实例
- 创建新 profile
- 启动 Windows Chrome
- 返回标准 JSON

### 4. SQLite 恢复层

WSL 独有的持久化数据库是：

- `browser_data`

表名：

- `browser_instance_wsl`

它保存的是真正和 WSL 恢复逻辑强相关的信息：

- `agent_id`
- `chat_id`
- `pid`
- `port`
- `ws`
- `http`
- `user_data_dir`
- `updated_at`

和 `browser_instance.json` 的关系是：

- `browser_instance.json` 面向公共 CLI 视图和空闲清理
- `browser_data` 面向 WSL 的 profile 复用、失活恢复、`profileDir` 回填

在 WSL 上，`browser_data` 才是 profile 维度的长期恢复层。

## 路径与状态

### 1. `browser_instance.json`

默认位于 `browser` 二进制同目录，也可以通过：

- `--state`
- `--instance-state`

覆盖。

对外的 `get/list/create/init` 仍然围绕这个文件工作。

### 2. `browser_data`

SQLite 路径解析规则：

1. 如果当前工作目录“看起来像 browser 程序目录”，优先使用 `cwd/browser_data`
2. 否则退回到 `browser` 运行时根目录下的 `browser_data`

“看起来像 browser 程序目录”的判断依据包括目录中存在以下任意文件：

- `browser`
- `browser.exe`
- `browser_launcher.sh`
- `build.sh`
- `main.go`

### 3. 受管 profile 目录

WSL 版本当前使用固定 Windows 根目录：

- Windows 路径根：`C:\temp`
- WSL 路径根：`/mnt/c/temp`

新 profile 的命名规则：

- `chrome_<4位随机小写字母数字>`

例如：

- `C:\temp\chrome_ab12`
- `/mnt/c/temp/chrome_ab12`

和 macOS 最大的差异是：

- WSL 的 `profileDir` 不依赖 `<agent workspace>/chrome_<port>`
- WSL 的 profile 与端口不是先天绑定关系
- 首次启动的端口可以是动态分配的

## Chrome 路径与 headless 规则

### 1. Chrome 可执行文件

WSL 下 `browserResolveChromePath` 的解析优先级是：

1. Browser 元数据 `meta.chrome`
2. 默认路径 `/mnt/c/Program Files/Google/Chrome/Application/chrome.exe`
3. 如果前两者都不可用，再退回命令行 `--chrome`
4. 最后才走普通系统候选路径探测

这里需要注意一个实现细节：

- 只要系统检测为 WSL，就会优先尝试默认 Windows Chrome 路径
- `meta.chrome` 如果存在但不是绝对路径，会直接报错

### 2. headless

`create` 在 WSL 下仍然复用公共 headless 解析逻辑：

1. `--headless-force`
2. `meta.headless`
3. `--headless`
4. 默认 `new`

最终传给 launcher 的是布尔值：

- `headlessMode == "none"` 时传 `false`
- 其他情况都传 `true`

所以 WSL launcher 只区分：

- 有头
- 无头

而不区分更多 headless 模式字符串。

`init` 在 WSL 下固定走有头模式，即传 `headless=false`。

## `create` 的 WSL 实际流程

`browser instance create --agentId ... --chatId ...` 在 WSL 下分成两层复用。

### 第一层：公共状态复用

CLI 先读取 `browser_instance.json` 并做状态归一化。

如果已经有一个健康实例：

- 只刷新 `lastActiveAt`
- 直接返回
- 不调用 launcher

### 第二层：Launcher + SQLite 复用

如果公共状态里没有可用实例，则调用 launcher。

launcher 内部的 `acquire` 流程如下。

#### 1. 基础校验

- 必须运行在 WSL
- `agentId/chatId` 先 `trim + lower-case`
- 二者都不能为空
- Chrome 路径必须存在

#### 2. 查询 SQLite 旧记录

用归一化后的 `agentId + chatId` 查询 `browser_instance_wsl`。

如果找到了旧记录，先做“在线探活”。

#### 3. 探活旧记录

探活方式：

- 请求 `http://localhost:<port>/json/version`
- 解析 `webSocketDebuggerUrl`

探活成功时会刷新：

- `ws`
- `http`
- `pid`
- `user_data_dir`

其中：

- `pid` 优先通过 `netstat.exe -ano` 重新按端口反查
- `user_data_dir` 如果库里是空，会先尝试 `/proc/<pid>/cmdline`
- `/proc` 查不到时，再退回 PowerShell 读取 `Win32_Process.CommandLine`

然后 upsert 回 SQLite 并直接返回。

#### 4. 旧记录失活时的“原地重启”

如果旧记录存在但已失活，会尝试用旧 profile 和旧端口重启，而不是立刻丢弃。

前提条件：

- `port > 0`
- `user_data_dir` 非空
- `user_data_dir` 可转回 WSL 路径
- 对应 profile 目录仍存在

重启前会清理 profile 内的锁相关条目，包括：

- `LOCK`
- `LOCKFILE`
- `SINGLETONLOCK`
- `SINGLETONCOOKIE`
- `SINGLETONSOCKET`
- `DEVTOOLSACTIVEPORT`
- `*.lock`
- `*-journal`

之后会最多等待 5 秒确认旧端口释放，再使用：

- 相同端口
- 相同 `user-data-dir`

重新启动 Chrome。

如果重启成功并探活通过：

- upsert SQLite
- 把同一个 profile 和端口重新交给公共 CLI 层

#### 5. 创建全新实例

如果旧记录不存在，或者无法原地重启，则：

1. 删除旧 SQLite 记录
2. 在 `/mnt/c/temp` 预留一个新的 `chrome_<4位随机串>` 目录
3. 用对应的 Windows 路径 `C:\temp\chrome_xxxx` 启动 Chrome
4. 启动参数使用 `--remote-debugging-port=0`
5. 等待 profile 目录里出现 `DevToolsActivePort`
6. 再访问 `/json/version` 获取最终 `ws/http`
7. upsert SQLite

首次新建时，端口由 Chrome 动态分配。

## WSL Chrome 启动参数

`wsl_instance_runtime.go` 当前使用的核心参数为：

- `--remote-debugging-address=0.0.0.0`
- `--user-data-dir=<Windows路径>`
- `--no-first-run`

新建实例时：

- `--remote-debugging-port=0`

重启旧实例时：

- `--remote-debugging-port=<旧port>`

无头时额外追加：

- `--headless=new`

有头时不追加 headless 参数。

标准输入、标准输出、标准错误都会重定向到 `/dev/null`。

## 就绪判定

WSL acquire 的就绪判定分两步：

1. 读取 `<profile>/DevToolsActivePort`
2. 调 `curl -s http://localhost:<port>/json/version`

关键参数：

- 总等待超时：30 秒
- 轮询间隔：5 秒
- 单次 `curl` 超时：5 秒

成功后返回：

```json
{
  "status": 0,
  "pid": 1234,
  "port": 9222,
  "ws": "ws://localhost:9222/devtools/browser/...",
  "http": "http://localhost:9222",
  "user-data-dir": "C:\\temp\\chrome_ab12"
}
```

失败时统一返回：

```json
{
  "status": 1,
  "message": "错误原因"
}
```

`browser_launcher.sh` 必须返回 JSON；如果 stdout 没有 JSON，CLI 直接视为失败。

## CLI 层如何接入 WSL Launcher

### `create`

CLI 层在 WSL 下会：

1. 先看 `browser_instance.json` 是否已有健康实例
2. 没有则调用 launcher
3. 将 launcher 返回的 `port/pid/ws` 写入 `browser_instance.json`
4. 返回公开记录，并把 `profileDir` 填成 launcher 返回的 `user-data-dir`

### `init`

`init` 在 WSL 下的当前行为是：

1. 先尝试关掉旧实例
2. 调用 launcher，强制 `headless=false`
3. 把返回值写进 `browser_instance.json`
4. ready 后立即返回

这里要特别强调：

- 当前实现不再要求 WSL `init` 使用 macOS 同款稳定哈希端口
- `init` 最终拿到的是 launcher/acquire 返回的端口，可能是旧记录复用的端口，也可能是新实例动态端口

### `get` / `list`

`get/list` 依然基于 `browser_instance.json` 做公开视图。

但 `profileDir` 的补全优先来自 `browser_data`：

1. 按 `agentId + chatId` 查 SQLite
2. 如果查到 `user_data_dir`，就把它回填到 API 响应
3. 查不到时，才退回公共路径推导

因此在 WSL 上：

- `profileDir` 更接近“SQLite 真值”
- 不是简单由 `agent workspace + port` 推导出来的

## stop / shutdown / restart 的 WSL 语义

### stop / shutdown

二者都会走 Windows 宿主机侧的强制结束逻辑：

- 如果已知 PID，直接 PowerShell `Stop-Process -Id <pid> -Force`
- 如果只知道端口，会先通过端口反查 PID
- 结束后确认 PID 和端口都已经释放
- 最后做一次 best-effort 锁文件清理

但两者的入口能力不同。

`browser instance stop`：

- 只依赖当前 `browser_instance.json`
- 适合“当前公开状态还在”的直接停止
- 不负责 missing-state fallback

`browser instance shutdown`：

- 走更完整的 destroy 路径
- 当 `browser_instance.json` 缺失或对应记录已不在时，仍可尝试从 `browser_data` 做 fallback 恢复
- 因此更适合正式销毁和兜底清理

无论是 `stop` 还是 `shutdown`，标准关闭后都会：

- 从 `browser_instance.json` 删除公开状态
- 不删除 `C:\temp\chrome_*`
- 不删除 `browser_data` 里的 SQLite 记录

这正是 WSL 方案的关键设计之一：

- profile 和 SQLite 记录会保留下来，供后续 `acquire` 做原地重启和登录态复用

### restart

`browser instance restart` 仍然等价于：

1. `shutdown`
2. `create`

但在 WSL 上，第二步的 `create` 是否会回到旧端口，取决于 SQLite 旧记录能否被成功原地重启：

- 能重启则继续沿用旧端口和旧 profile
- 不能重启则创建新 profile，并拿到一个新的动态端口

## 空闲超时与清理

公共状态层仍然保留空闲超时机制。

默认：

- `browser_expired = 10` 分钟

触发时会：

- 结束对应 Chrome 进程
- 从 `browser_instance.json` 移除记录

但不会：

- 删除 `browser_data`
- 删除 `C:\temp\chrome_*`

所以 WSL 的“超时释放”更像是“结束当前活跃实例”，不是“彻底销毁会话痕迹”。

## WSL 的状态源差异

可以把 WSL 当前方案理解为“双状态源”：

### 面向公开 CLI 的状态源

- `browser_instance.json`

用于：

- `list/get`
- `lastActiveAt`
- 空闲超时
- 顶层插件统一管理

### 面向恢复逻辑的状态源

- `browser_data`

用于：

- profile 持久化
- `profileDir` 回填
- 失活后原地重启
- 状态文件缺失时的缺省恢复

这也是 WSL 与 macOS 的根本区别之一。

## 状态文件缺失时的行为

### macOS

状态文件丢失而端口仍被占用时，通常会直接遇到稳定端口冲突。

### WSL

WSL 有额外恢复路径：

- `shutdown` 的 missing-state fallback 会先查 `browser_data`
- 不再依赖“按稳定哈希端口猜测实例”

但也要注意：

- 如果 `browser_data` 本身丢失，就只能按全新 profile 重新创建，原登录态无法自动恢复

## 与旧设想相比，当前实现已经明确变化的点

以下内容已经不再是当前 WSL 实现事实：

- 不再复制系统 Chrome User Data
- 不再要求首次创建必须走稳定哈希端口
- `profileDir` 不再绑定到 `<agent workspace>/chrome_<port>`
- `shutdown/stop` 不会默认删除 `C:\temp\chrome_*`

## 额外说明：WSL 插件生命周期 Bootstrap CDP

除 `browser instance` 之外，WSL 里还保留了一套插件生命周期用的 bootstrap CDP 逻辑，位于：

- `wsl_runtime.go`

其特征包括：

- 固定端口 `29876`
- 使用 `C:\temp\chrome_29876`
- 主要用于插件 `start/stop` 生命周期兼容

它和本文档重点描述的“受管实例 acquire 方案”是两条并存路径，不应混淆。

## 风险与边界

### 1. profile 会长期保留

标准 `shutdown/stop/expired` 都不会删 `C:\temp\chrome_*`，如果长期运行且身份很多，目录会持续累积。

### 2. 首次新建端口不稳定

首次分配使用 `--remote-debugging-port=0`，因此不能假设第一次 `create` 就和 macOS 一样拥有固定端口。

### 3. `browser_instance.json` 与 `browser_data` 可能短时间不完全一致

公开状态和恢复状态分离是设计选择，但排查问题时必须同时看两份状态。

### 4. `profileDir` 返回的是 Windows 路径

WSL 对外返回的是 `C:\temp\chrome_xxxx`，调用方不能假设这是 Unix 路径。

## 排查建议

遇到 WSL 实例问题时，建议按这个顺序查：

1. `browser_instance.json` 是否仍有该实例记录
2. `browser_data` 是否存在该 `agentId + chatId`
3. `user_data_dir` 对应的 `C:\temp\chrome_*` 是否还在
4. `/json/version` 是否能返回有效 `webSocketDebuggerUrl`
5. `netstat.exe -ano` 是否还能按端口找到 PID
6. `profile` 目录里是否残留锁文件导致重启失败
7. `meta.chrome` 是否配置错误或指向了不存在的路径

## 建议验证用例

建议至少覆盖以下回归：

1. 首次 `create`，确认返回动态端口和新的 `C:\temp\chrome_*`
2. 再次 `create`，确认能直接复用活跃实例
3. `shutdown` 后再 `create`，确认能基于 SQLite 旧记录原地重启
4. 删除 `browser_instance.json` 后执行 `shutdown`，确认还能通过 `browser_data` 做缺省恢复
5. `init`，确认始终以有头模式启动
6. 检查 `profileDir`，确认 API 返回的是 SQLite 中的 Windows 路径
