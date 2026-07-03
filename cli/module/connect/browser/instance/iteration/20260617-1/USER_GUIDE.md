# 20260617-1 使用手册

## 目标

本次迭代提供了一个独立的 WSL Browser Instance 管理程序 `browser_instance_wsl.go`，用于在 Windows WSL / WSL2 环境内按 `agentId + chatId` 维度复用、恢复或新建一个可用的 Chrome CDP 实例，并以 JSON 输出结果。

## 适用场景

- 在 WSL 内为 `browser instance create` 提供实例获取能力
- 在 WSL 内为 `browser instance init` 提供有头实例获取能力
- 需要按 `agentId + chatId` 复用已存在的 Chrome CDP
- 需要在旧实例失活时尝试原地重启，而不是直接丢弃旧 profile

## 运行方式

可直接运行同目录源码：

```bash
go run ./browser_instance_wsl.go --agentId agent-a --chatId chat-001
```

也可编译后运行：

```bash
./browser_instance_wsl --agentId agent-a --chatId chat-001
```

说明：

- 程序必须运行在 WSL 环境内
- 当前实现不提供 `help` 子命令
- 参数解析失败时不会输出默认 flag 帮助，而是统一输出 JSON 错误

## 参数

```bash
./browser_instance_wsl --agentId agent-a --chatId chat-001 --headless=true --chrome "/mnt/c/Program Files/Google/Chrome/Application/chrome.exe"
```

参数说明：

- `--agentId`：必填，会先做 `trim + lower-case`
- `--chatId`：必填，会先做 `trim + lower-case`
- `--headless`：选填，默认 `true`
- `--chrome`：选填，未传或为空白时回退到 `/mnt/c/Program Files/Google/Chrome/Application/chrome.exe`

常见错误：

- `agentId` 为空时返回 `{"status":1,"message":"agentId is required"}`
- `chatId` 为空时返回 `{"status":1,"message":"chatId is required"}`
- 不在 WSL 内运行时返回 `{"status":1,"message":"browser_instance_wsl must run inside WSL"}`
- Chrome 不存在时返回 `{"status":1,"message":"chrome not found: <实际路径>"}`

## 实例获取顺序

程序会按下面顺序获取实例。

### 1. 复用已登记且健康的实例

- 先到 SQLite 表 `browser_instance_wsl` 里按 `agentId + chatId` 查记录
- 如果记录存在且端口可用，就请求 `http://localhost:<port>/json/version`
- 当 `webSocketDebuggerUrl` 可解析时，直接复用该实例
- 成功复用后会刷新：
  - `pid`
  - `ws`
  - `http`
  - `user_data_dir`
  - `updated_at`

### 2. 原地重启失活实例

- 如果旧记录探活失败，程序会优先尝试复用旧端口和旧 profile
- 前提是旧记录里有合法 `port` 和 `user_data_dir`
- 会先清理 profile 下的锁文件，例如：
  - `LOCK`
  - `LOCKFILE`
  - `SINGLETONLOCK`
  - `SINGLETONCOOKIE`
  - `SINGLETONSOCKET`
  - `DEVTOOLSACTIVEPORT`
  - `*.lock`
  - `*-journal`
- 然后最多等待 5 秒，确认旧端口释放
- 端口释放后，用相同端口、相同 `user-data-dir` 重启 Chrome

### 3. 新建实例

- 如果没有可复用实例，也无法原地重启，就新建 profile 并启动 Chrome
- profile 根目录固定在：
  - WSL 路径：`/mnt/c/temp`
  - Windows 路径：`C:\temp`
- 新目录命名规则为 `chrome_<4位随机字符>`
- 随机字符集为 `a-z0-9`

## 数据存储

- 程序使用同目录下的 SQLite 文件 `browser_data`
- 如果当前工作目录本身就像 browser 程序目录，也会优先使用当前目录下的 `browser_data`
- 表名固定为 `browser_instance_wsl`
- 主键为 `agent_id + chat_id`

表里保存的核心字段包括：

- `pid`
- `port`
- `ws`
- `http`
- `user_data_dir`
- `updated_at`

## Chrome 启动规则

- Chrome 路径优先取 `--chrome`
- `--chrome` 为空时回退到 `/mnt/c/Program Files/Google/Chrome/Application/chrome.exe`
- 启动参数固定包含：
  - `--remote-debugging-address=0.0.0.0`
  - `--user-data-dir=<Windows路径>`
  - `--no-first-run`
- 新建实例时使用 `--remote-debugging-port=0`
- 原地重启旧实例时使用旧记录里的固定端口
- `--headless=true` 时追加 `--headless=new`
- `--headless=false` 时不追加 headless 参数

## 就绪判定

- 单次创建或重启的整体等待上限是 30 秒
- 轮询间隔固定为 5 秒
- 新建实例时，会先从 `<profileUnix>/DevToolsActivePort` 读取端口
- 拿到端口后，再请求 `http://localhost:<port>/json/version`
- 只有当返回里存在有效 `webSocketDebuggerUrl` 时，才认为 CDP 就绪
- 30 秒内仍未就绪时，返回最后一次错误；如果没有更具体错误，则返回 `cdp not ready after 30s`

## 输出格式

成功时输出：

```json
{"status":0,"pid":1234,"port":9222,"ws":"ws://localhost:9222/devtools/browser/...","http":"http://localhost:9222","user-data-dir":"C:\\temp\\chrome_ab12"}
```

失败时输出：

```json
{"status":1,"message":"错误原因"}
```

说明：

- `http` 字段固定是 `http://localhost:<port>`
- `user-data-dir` 返回的是 Windows 路径
- 如果脚本成功复用老实例，也会返回完整成功 JSON

## 清理语义

- 只有“新建实例失败”这一路径，才会尝试删除刚刚创建的新 profile
- 对于旧记录对应的 profile，程序不会先删除目录再启动
- 对于旧记录对应的数据库记录，程序不会先删除旧记录；成功后直接 upsert 覆盖
