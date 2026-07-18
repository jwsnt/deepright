# CLI-Get 使用手册

## 简介

CLI-Get 是一个心跳上报与任务执行客户端。它以心跳方式将 Agent 元数据上报至服务端（`cli/get`），获取待执行任务后执行，并将结果回传至服务端（`cli/pub`）。

- 默认模式：`cli/get -> 本地 Shell 执行 -> cli/pub`
- 当前版本实际为两段式流水线：`cli/get -> 本地任务队列 -> 执行 -> 本地发布队列 -> cli/pub`
- 当任务的非空 `chat` 命中有效 `sandbox_exe` 时：`cli/get -> 对应模式的 CLI_SANDBOX -cmd -> cli/pub`
- 如果 `cli/get` 响应任务中带有 `subOps.exempted=true`，则即使当前会话已开启沙盒，也仍然走原始链路：`cli/get -> 本地 Shell 执行 -> cli/pub`
- 如果当前工作目录共享 SQLite `data` 的 `message_insert` 表中存在当前 `chat` 的待上传插入消息，则会在 `cli/pub` 前自动附带最多 `5` 条 `insert` 记录；发布成功后自动改为已上传

## 安装

```bash
go build -o cli-get .
```

## 使用方法

```bash
./cli-get --agent-dir <Agent目录> [选项...]
```

### 参数说明

| 参数 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `--agent-dir` | 是 | — | Agent 根目录路径 |
| `--host` | 否 | `https://www.deepright.cn` | 服务端 URL（含端口） |
| `--device` | 否 | 自动生成 | 设备ID |
| `--sandbox_app` | 否 | 从 `config/config.json` 读取 | 沙盒根路径或锚点路径；支持 `.app`、目录或二进制路径，相对路径按主程序可执行文件目录解析 |
| `--agent-cache` | 否 | `120000` | Agent 元数据缓存 TTL（毫秒） |
| `--sleep` | 否 | `3000` | 心跳请求失败或非 200 时的重试等待时间（毫秒） |
| `--thread` | 否 | `20` | 执行 Worker 数量 |
| `--queue` | 否 | `1000` | 本地任务队列容量；队列满时会暂停发 `/cli/get` |
| `--retry_interval` | 否 | `10000` | `/cli/pub` 失败后的重试等待时间（毫秒） |
| `--retry_times` | 否 | `1` | `/cli/pub` 首次失败后允许额外重试的次数 |
| `--http_timeout` | 否 | `60000` | HTTP 总超时（毫秒） |
| `--http_connect_timeout` | 否 | `15000` | HTTP 连接超时（毫秒） |
| `--http_socket_timeout` | 否 | `45000` | HTTP 读取超时（毫秒） |
| `--idle_timeout` | 否 | `90` | 连接池空闲超时（秒） |

### 示例

```bash
# 基本用法
./cli-get --agent-dir ./agents

# 指定服务端和线程数
./cli-get --agent-dir ./agents --host http://192.168.1.100:9998 --thread 5

# 增大本地任务队列，并开启一次发布重试
./cli-get --agent-dir ./agents --queue 2000 --retry_interval 5000 --retry_times 1

# 自定义超时和心跳间隔
./cli-get --agent-dir ./agents --sleep 1000 --http_timeout 120000

# 指定 macOS 沙盒锚点路径
./cli-get --agent-dir ./agents --sandbox_app ../Helpers/CLI_SANDBOX.app

# 指定 WSL/Linux 沙盒锚点路径
./cli-get --agent-dir ./agents --sandbox_app ./helpers
```

## 工作流程

```
┌─────────────────────────────────────────┐
│            Heartbeat 线程                │
│                                         │
│  ┌──► 检查本地任务队列是否有空位        │
│  │       │                              │
│  │       ├── 队列满 -> sleep ───┐       │
│  │       │                      │       │
│  │       └── 有空位 -> cli/get ─┐       │
│  │                    │         │       │
│  │    ┌───────────────▼────────┐│       │
│  │    │  本地任务队列 taskQueue ││       │
│  │    └───────────────┬────────┘│       │
│  │                    │          │       │
│  │    ┌───────────────▼────────┐ │       │
│  │    │ 执行 Worker 池          │ │       │
│  │    │ 出队前检查 ddl          │ │       │
│  │    │ 执行命令 / 沙盒命令     │ │       │
│  │    └───────────────┬────────┘ │       │
│  │                    │          │       │
│  │    ┌───────────────▼────────┐ │       │
│  │    │ 本地发布队列 publishQueue│ │      │
│  │    └───────────────┬────────┘ │       │
│  │                    │          │       │
│  │    ┌───────────────▼────────┐ │       │
│  │    │ 发布 Worker             │ │       │
│  │    │ cli/pub + 重试          │ │       │
│  │    └────────────────────────┘ │       │
│  │                              │       │
│  └──────────────────────────────┘       │
└─────────────────────────────────────────┘
```

1. 心跳线程先检查 `taskQueue` 是否还有空位；只有有空位时才会发 `/cli/get`
2. 如果本地任务队列已满，不会拉新任务，而是按 `--sleep` 等待后再次检查
3. 如果 `/cli/get` 返回任务，任务会立刻进入本地 `taskQueue`，心跳线程马上继续下一轮，不等待执行完成
4. 执行 Worker 从 `taskQueue` 取任务时先检查 `ddl`；如果当前时间已超过 `ddl`，会打印日志并直接丢弃
5. 未过期的任务按现有逻辑执行，本地 Shell 或 `CLI_SANDBOX` 均会复用原有执行链路
6. 执行结果会进入 `publishQueue`，由独立发布 Worker 提交 `/cli/pub`
7. 如果 `/cli/pub` 返回明确错误、超时、HTTP 非 `200`、或响应解析失败，会按 `--retry_interval` 与 `--retry_times` 重试
8. 如果心跳请求本身失败、HTTP 非 200，或响应解析异常，则按指数退避等待后重试

## 本地队列与重试

- `cli-get` 现在不会因为执行 Worker 不够而阻塞下一次 `/cli/get` 心跳
- 本地 `taskQueue` 是纯内存队列，不做持久化恢复
- 当进程退出、崩溃、重启时，队列中尚未执行或尚未发布的任务允许丢失
- `publishQueue` 也是纯内存队列；这样 `/cli/pub` 的重试等待不会占住执行 Worker
- `/cli/pub` 重试接受“同一 `tid` 可能重复推送”的现实语义，不额外做客户端去重

## DDL 过期

- 服务端在 `cli/get` 任务里可以返回 `ddl`，含义是任务执行截止时间戳（毫秒）
- `cli-get` 不会只在收到响应时校验一次 `ddl`，而是会在执行 Worker 真正出队执行前再次检查
- 当当前时间大于 `ddl` 时：
  - 任务不会执行
  - 任务不会进入 `/cli/pub`
  - 会打印包含 `agentId`、`chat`、`tid`、`ddl`、当前时间和 `cmd` 的日志
- `ddl` 为空或 `0` 时，按“不过期”处理

## 插入消息上报

- `cli-get` 会复用当前工作目录共享 SQLite `data`
- 当 `message_insert` 表中存在与本次任务相同 `chat` 的 `status=0` 记录时，会在 `cli/pub` 请求体里追加：

```json
{
  "insert": [
    { "tid": "1718966400000", "message": "..." }
  ]
}
```

- 单次最多附带 `5` 条，按 `created_at` 顺序读取
- `/cli/pub` 成功后，这批 `tid` 只会标记为“已上报一次”，后续 `cli/get` 不会再次重复上报
- 只有当 integration 后续收到响应报文中 `metadata.__PROCESS__ = rag_insert` 且 `metadata.__TID__` 相同，这条消息才会最终更新为 `status=1`
- 如果读取或状态更新失败，会输出错误日志，但不会中断原有 `cli/pub` 主链路

## Sandbox 模式

- `cli-get` 按 `chatId` 保存和查找 `sandbox_exe`；`agentId` 不参与沙盒状态定位，只用于执行与日志观测
- `chatId` 必须为非空字符串。空白或缺失的 `chatId` 不会命中已有沙盒状态，也不能写入或删除沙盒状态；这类任务始终按非沙盒链路处理
- 只有以下 3 个枚举值会启用沙盒：
  - `filepick`：执行前弹出目录选择器；未选择则返回权限拒绝
  - `net`：通过 `sandbox-exec` 关闭网络
  - `filepick_net`：同时执行目录选择和网络关闭
- `sandbox_exe` 为空、缺失或不是以上 3 个值时，继续走原来的本地 Shell 执行链路
- 如果 `cli/get` 响应报文中的 `subOps.exempted=true`，也会直接跳过沙盒并走原始本地执行链路
- `sandbox_exe` 状态保存在当前工作目录下 SQLite `data` 的 `cli_sandbox_state` 表
- 表以 `chat_id` 为唯一键，包含 `sandbox_exe`、`allowed_dir` 与 `updated_at`；不会使用 `agent_id + chat_id` 联合主键或联合查询作为运行时命中条件
- 新版本首次运行时，如检测到旧版表结构（包括以 `agent_id + chat_id` 为键的结构），会删除旧表并重建；旧状态不会迁移
- 关闭沙盒时会直接删除该 `chatId` 的状态行，不保留关闭记录。无记录等价于模式 `off`
- 不同 `agentId` 使用同一非空 `chatId` 时，会命中同一条沙盒状态；同一 `agentId` 的不同 `chatId` 则彼此独立
- 每次成功变更状态会向标准错误输出文本日志，包含 `agentId`、`chatId`、旧模式和新模式；无记录与删除后的状态均记为 `off`。命中沙盒执行时也会输出包含 `agentId`、`chatId` 和当前模式的查找日志
- `sandbox_app` 优先来自 `--sandbox_app`；未传时读取主程序侧的 `config/config.json` 中的 `sandbox_app`
- 如果当前会话启用了沙盒模式，但没有找到对应模式的 `CLI_SANDBOX` 可执行文件，本次任务会直接按失败结果回传，不会静默回退到本地直连执行
- macOS 下会按 `sandbox_exe` 解析对应 bundle：
  - `filepick -> <anchor-parent>/filepick/CLI_SANDBOX.app/Contents/MacOS/CLI_SANDBOX`
  - `net -> <anchor-parent>/net/CLI_SANDBOX.app/Contents/MacOS/CLI_SANDBOX`
  - `filepick_net -> <anchor-parent>/filepick_net/CLI_SANDBOX.app/Contents/MacOS/CLI_SANDBOX`
- WSL/Linux 下会按 `sandbox_exe` 解析对应 helper：
  - `filepick -> <anchor-parent>/filepick/CLI_SANDBOX`
  - `net -> <anchor-parent>/net/CLI_SANDBOX`
  - `filepick_net -> <anchor-parent>/filepick_net/CLI_SANDBOX`
  - 如果 `sandbox_app` 指向主程序同级 `helpers` 目录，也支持 `helpers/<mode>/CLI_SANDBOX`

示例：

```json
{
  "sandbox_exe": "filepick_net"
}
```

## 统一日志

- `cli/get` 与 `cli/pub` 都会异步写入当前工作目录下的 SQLite `data`
- 与 `proxy` 需求共用同一张日志表：`agent_message_log`
- 表字段：
  - `agent_id`
  - `chat_id`
  - `content`
  - `log_type`
  - `created_at`
- 索引为 `agent_id + chat_id + log_type + created_at`
- `log_type` 固定取值：
  - `2`：`cli/get`
  - `3`：`cli/pub`
- `cli/get` 会在解析到服务端返回任务后，按该任务中的 `agentId` 与 `chat` 记录本次心跳日志
- 如果 `cli/get` 响应中的 `content` 为 `null` 或空字符串，表示当前没有待执行任务，本次不记录 `cli/get` 日志
- `cli/pub` 会按结果中的 `agentId` 与 `chat` 记录回传日志
- 异步写入不会阻塞心跳轮询和命令执行

## 活跃命令

- Worker 在执行命令前会先注册到进程内活跃命令列表，执行完成后注销
- 如果命令执行上下文被取消，回传结果中的 `status=1`，且解压后的 `cmd` 内容为 `命令被终止`

## Agent 元数据中的 `plugins`

- `cli/get` 与 `cli/pub` 共用同一份 Agent 元数据，只有当插件同时满足“已配置且已启动”时才会带上 `plugins`
- `plugins` 固定来自当前 `cli-get` 可执行文件同级目录下的 `plugins/`
- 探测方式优先为执行当前二进制自身的 `list-meta` 命令，并读取返回结果中的插件 `key`
- 只有 `list-meta` 成功返回、存在非空 `key`，且对应 `<plugin-key>.pid` 指向的进程仍存活时，才会把这些 key 排序后写入 `metadata.plugins`
- 如果当前二进制未收口 `list-meta`、命令执行失败或返回 JSON 解析失败，则会直接省略 `plugins`
- 当前仓库下独立编译出的 `cli-get` 只负责心跳与执行，不提供 `list-meta` 子命令，因此默认情况下通常不会携带 `plugins`
- 当 `cli-get` 能力被收口到支持 `list-meta` 的上层二进制时，例如 integration 统一 CLI，`plugins` 会按该命令输出的已配置插件列表继续过滤出已启动插件后再上报

示意：

```json
{
  "metadata": {
    "timezone": "Asia/Shanghai",
    "deviceId": "device-001",
    "terminal": "/bin/zsh",
    "plugins": ["browser", "feishu"],
    "gateway": "aa:bb:cc:dd:ee:ff",
    "sys": "Darwin 24.5.0",
    "app": "/path/to/cli-get",
    "agents": []
  }
}
```

如果插件探测失败或为空，实际请求中会直接省略 `plugins` 字段，而不是返回空数组。

## 请求格式

### cli/get（心跳上报）

```json
{
  "model": "",
  "messages": [{"role": "user", "content": ""}],
  "metadata": {
    "timezone": "Asia/Shanghai",
    "deviceId": "...",
    "terminal": "/bin/zsh",
    "plugins": ["browser", "feishu"],
    "gateway": "aa:bb:cc:dd:ee:ff",
    "sys": "Darwin 23.4.0",
    "app": "/path/to/cli-get",
    "agents": [...]
  }
}
```

如果服务端返回待执行任务，并且希望该任务豁免会话沙盒，可在响应 `content` 中携带：

```json
{
  "timeout": 5000,
  "suffix": "cmd",
  "type": "cmd",
  "ddl": 1743177600000,
  "tid": "task_001",
  "cmd": "echo hi",
  "agentId": "A",
  "chat": "chat_001",
  "subOps": {
    "exempted": true
  }
}
```

此时 `cli-get` 会直接执行命令并回传，不会调用 `CLI_SANDBOX`。

### cli/pub（结果回传）

```json
{
  "model": "",
  "messages": [
    {
      "role": "user",
      "content": "{\"status\":0,\"agentId\":\"A\",\"suffix\":\"cmd\",\"chat\":\"chat_001\",\"type\":\"cmd\",\"tid\":\"task_001\",\"cmd\":\"H4sI...Base64...\"}"
    }
  ],
  "metadata": {
    "timezone": "Asia/Shanghai",
    "deviceId": "...",
    "terminal": "/bin/zsh",
    "plugins": ["browser", "feishu"],
    "gateway": "aa:bb:cc:dd:ee:ff",
    "sys": "Darwin 23.4.0",
    "app": "/path/to/cli-get",
    "agents": [...]
  }
}
```

## 作为子模块调用

核心函数可独立使用：

```go
// 发送心跳，获取任务
task, err := Heartbeat(httpClient, "http://host:9998", agentMetadata)

// 执行任务
result := ExecuteTask(task, "/bin/zsh")

// 回传结果
err = PublishResult(httpClient, "http://host:9998", result, agentMetadata)
```

## 注意事项

- 任务命令支持管道符（`&&`、`|`）、绝对路径、相对路径和 `~` 路径
- 任务超时默认 180 秒，由服务端 `timeout` 字段指定
- 命令执行结果经 GZIP 压缩后 Base64 编码
- `cli/get` 与 `cli/pub` 都会带上相同的 Agent 元数据；若插件探测成功，其中会额外包含 `plugins` 插件 key 列表
- 任何异常均不会中断主循环，自动进入下一次心跳

---

## 迭代 20260621-1 至 20260716-1：消息、队列与超时结果

- 待上传插入消息会随任务可靠上报，服务端确认后才标记完成。
- 可通过 `--queue` 调整本地待执行队列上限（默认 `1000`）；超过执行截止时间的任务会被跳过并记录日志。
- 同一会话的沙盒状态以 `chatId` 为准，切换 Agent 不会改变该会话的沙盒选择。
- 命令超时时，若已经产生输出，结果会保留该输出并标注不完整；无输出时会返回明确的超时提示。

---

## 迭代 20260621-1 至 20260716-1：任务可靠性与超时结果

- 客户端会随任务安全上报当前会话中最多 5 条待插入消息，只有服务端确认后才从待上传列表移除。
- 可通过 `--queue` 调整内存任务队列容量，默认值为 1000；队列满时客户端不会继续拉取任务，过期任务会被跳过并记录日志。
- 会话沙盒以 `chatId` 为唯一状态键；切换 Agent 不会改变同一会话的沙盒状态。
- 命令超时时，已产生的输出仍会返回，并标明结果可能不完整；没有任何输出时会返回明确的超时提示。此规则对本地、沙盒、WSL 和页面命令执行一致。
