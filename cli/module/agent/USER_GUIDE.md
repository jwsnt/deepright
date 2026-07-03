# Agent Scanner 使用手册

## 简介

Agent Scanner 是一个命令行工具，用于遍历指定目录下的 Agent 子目录，提取每个 Agent 的元数据（包括身份信息和技能列表），并结合系统环境信息输出为 JSON 格式。

本次更新后，Agent 元数据还会按需补充 `knowledge` 字段，用于向上游统一暴露知识库路径与最后更新时间。

同时，每个 Agent 还会从 `config.json` 暴露以下配置字段：

- `description`
- `provider`
- `thinking`
- `router_disable`
- `version`
- `sandbox`

其中：

- `provider` 未配置时输出空字符串
- `thinking` 未配置时输出 `false`
- `router_disable` 未配置时输出 `true`
- `version` 未配置时输出空字符串
- `sandbox` 默认输出空字符串；传入 `--chatId` 时会实时读取共享 sqlite 里的 `agentId + chatId` 沙盒模式
- `router_disable=true` 表示关闭蜂群路由，`router_disable=false` 表示开启蜂群路由

同时，`agents[].skills` 已改为实时扫描：

- 每次读取 Agent metadata 时，都会重新遍历对应 Agent 的 `skills` 目录
- `--agent-cache` 仍然存在，但不再缓存 `skills` 字段本身

另外，`config.json` 中的 Agent 属性会分两类刷新：

- `description`、`provider`、`thinking`、`router_disable` 仍然每次实时读取
- `version` 只在首次扫描进入缓存时读取一次；缓存未失效前，即使 `--agent-dir/<agentId>/config.json` 里的 `version` 变化也不会立刻刷新
- 即使其他共享字段仍命中 `--agent-cache`
- `version/provider` 都不会写入 sqlite；`version` 仅存在当前进程的内存缓存里，进程重启后会重新从 `--agent-dir/<agentId>/config.json` 读取
- 如果历史 `config.json` 里仍保留旧字段 `swarm`，会自动按相反语义转换成 `router_disable`

另外，`git` 字段也会在每次读取 Agent metadata 时实时探测：

- 即使其他共享字段仍命中 `--agent-cache`
- 当前机器的 git 安装路径变化后，下一次输出也会立即反映最新结果

## 安装

```bash
go build -o agent-scanner .
```

## 使用方法

```bash
./agent-scanner [--agent-cache <毫秒数>] [--device <设备ID>] [--chatId <ChatId>] <目录路径>
./agent-scanner [--app-dir <应用启动目录>] [--chatId <ChatId>] <目录路径>
```

### 参数说明

| 参数 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `<目录路径>` | 是 | — | 包含 Agent 子目录的根目录 |
| `--agent-cache` | 否 | `10000` (10秒) | Agent 共享元数据缓存有效期；`skills`、`git`、`sandbox` 始终实时刷新；`version` 跟随缓存失效刷新 |
| `--device` | 否 | 自动生成 | 设备ID，默认基于系统硬件信息生成稳定唯一码 |
| `--app-dir` | 否 | `<目录路径>` 的父目录 | 应用启动目录；用于只读探测 `knowledge` 元数据 |
| `--chatId` | 否 | 空 | 当前会话 ChatId；传入后会实时补充每个 Agent 的 `sandbox` |

### 示例

```bash
# 扫描 test-case 目录，使用默认缓存
./agent-scanner test-case

# 指定设备ID，缓存60秒
./agent-scanner --device my-device-001 --agent-cache 60000 ./agents

# 不使用缓存
./agent-scanner --agent-cache 0 ./agents

# 显式指定应用启动目录，以便探测 knowledge 元数据
./agent-scanner --app-dir /srv/my-app ./agents

# 同时按指定 ChatId 实时查看沙盒状态
./agent-scanner --app-dir /srv/my-app --chatId chat-001 ./agents
```

## 目录结构要求

```
<根目录>/
├── agent-a/              # Agent 名称 = "agent-a"
│   ├── SOUL.md           # soul 内容（可选，不存在则为空）
│   ├── USER.md           # user 内容（可选，不存在则为空）
│   ├── config.json       # Agent 额外元数据（可选）
│   └── skills/           # 技能目录（可选）
│       ├── skill-1/
│       │   └── SKILL.md  # 技能元数据文件
│       └── skill-2/
│           └── SKILL.md
└── agent-b/
    ├── SOUL.md
    └── skills/
        └── ...
```

### 说明

- 根目录下的每个子目录代表一个 Agent，子目录名即为 Agent ID（agentId）
- 不扫描嵌套子孙目录作为 Agent（仅直接子目录）
- `SOUL.md` 和 `USER.md`（或 `user.md`）为可选文件，不存在时对应字段为空字符串
- `config.json` 为可选文件；当前会读取其中的 `description`、`provider`、`thinking`、`router_disable`、`version`
- `skills` 目录下递归扫描所有 `SKILL.md` 和 `SKILL` 文件，提取技能元数据
- `skills` 每次查询时都会重新遍历，不依赖 `--agent-cache` 中的历史扫描结果
- `config.json` 中的 `description`、`provider`、`thinking`、`router_disable` 每次查询时都会重新读取，不依赖 `--agent-cache` 中的历史结果
- `--agent-dir/<agentId>/config.json` 里的 `version` 只在当前缓存周期首次扫描时读取一次；缓存失效或显式 `FlushCache()` 后才会重新读取
- `sandbox` 不读 `config.json`；只有传入 `--chatId` 或等价子模块参数时，才会按 `agentId + chatId` 实时读取共享 sqlite
- `router_disable=true` 表示关闭蜂群；如遇到旧配置中的 `swarm=true/false`，系统会自动转换为 `router_disable=false/true`
- `skills[].compatibility` 同时兼容 YAML 字符串和字符串列表；如果源文件使用列表，会规范化为以 `; ` 连接的单个字符串

## 输出格式

```json
{
  "deviceId": "设备唯一标识",
  "terminal": "终端类型（如 /bin/zsh）",
  "plugins": ["插件Key"],
  "knowledge": {
    "lastUpdate": 0,
    "path": "/app/knowledge"
  },
  "git": "本机 git 可执行文件绝对路径，未安装时为空字符串",
  "gateway": "MAC 网关地址",
  "sys": "操作系统信息（如 Darwin 23.4.0）",
  "agents": [
    {
      "workspace": "Agent 工作目录绝对路径",
      "agentId": "Agent ID（目录名）",
      "description": "Agent 描述，未配置时为空字符串",
      "provider": "模型 provider，未配置时为空字符串",
      "thinking": false,
      "router_disable": true,
      "version": "Agent 版本，未配置时为空字符串",
      "sandbox": "当前 agentId + chatId 的沙盒模式；未传 chatId 或未写入时为空字符串",
      "soul": "SOUL.md 内容",
      "user": "USER.md 内容",
      "skills": [
        {
          "name": "技能名称",
          "description": "技能描述",
          "compatibility": "macOS (Darwin); zsh shell"
        }
      ]
    }
  ]
}
```

### `plugins` 字段

- `plugins` 表示当前应用里“已配置且已启动插件”的 `key` 列表
- 先复用 `list-meta` 返回的插件配置视图，并读取其中的 `key`
- 再校验对应插件进程是否已启动；只有存在 `<plugin-key>.pid` 且该 PID 仍存活时才会写入
- 返回结果会按 `key` 排序
- 如果没有任何同时满足“已配置且已启动”的插件，则输出中不包含该字段
- `knowledge` 与 `plugins` 一样属于可选字段；不存在时不会输出空对象

### `knowledge` 字段

- `knowledge` 仅在应用启动目录下已存在 `knowledge` 目录时输出
- `path` 为知识库绝对路径
- `lastUpdate` 来自同一应用目录下共享的 `data` sqlite；没有记录时为 `0`
- 独立运行时，默认把扫描根目录的父目录视为应用启动目录
- 需要显式指定时，可通过 `--app-dir` 传入

### `knowledge` 的判定规则

- 目录 `<app-dir>/knowledge` 不存在：
  - 不输出 `knowledge`
- 目录存在，但 `<app-dir>/data` 不存在：
  - 输出 `knowledge.path`
  - `lastUpdate = 0`
- 目录存在，`data` 也存在，但 `knowledge_runtime` 记录缺失：
  - 仍输出 `knowledge`
  - `lastUpdate = 0`
- 目录和记录都存在：
  - 输出真实的 `lastUpdate`

## 作为子模块调用

```go
import "agent-scanner"

data, err := GetAgentOutput("/path/to/agents", "device-id", 120*time.Second)
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(data))
```

`GetAgentOutput` 函数签名：

```go
func GetAgentOutput(root string, deviceID string, ttl time.Duration) ([]byte, error)
```

- `root`：Agent 根目录路径
- `deviceID`：设备ID，传空字符串则自动生成
- `ttl`：缓存有效期

如果需要显式指定应用启动目录，推荐使用：

```go
data, err := GetAgentOutputForApp("/path/to/agents", "/path/to/app", "device-id", 120*time.Second)
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(data))
```

函数签名：

```go
func GetAgentOutputForApp(root string, appDir string, deviceID string, ttl time.Duration) ([]byte, error)
```

如果还需要按某个会话实时补充 `sandbox`，可使用：

```go
data, err := GetAgentOutputForAppAndChat("/path/to/agents", "/path/to/app", "device-id", 120*time.Second, "chat-001")
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(data))
```

函数签名：

```go
func GetAgentOutputForAppAndChat(root string, appDir string, deviceID string, ttl time.Duration, chatID string) ([]byte, error)
```

## 注意事项

- `deviceId` 默认优先直接使用系统硬件UUID生成；仅在系统取不到硬件UUID时，才回退为基于操作系统与架构计算的稳定值
- `terminal` 取自 `SHELL` 环境变量
- `plugins` 取自当前程序 `list-meta` 暴露的插件 key，并额外过滤掉未启动的插件进程
- `knowledge` 使用只读探测逻辑；不会在 `agent` 模块里自动创建知识库目录或 sqlite
- `sandbox` 来自共享 sqlite 的 `cli_sandbox_state` 表，按 `agentId + chatId` 实时读取，不参与缓存
- Windows 仅识别 `.exe` 文件；macOS/Linux 仅识别无后缀且具备执行权限的文件
- `git` 为本机安装的 git 可执行文件绝对路径；macOS/Linux 优先通过 `PATH` 与 `command -v git` 获取，Windows 优先通过 `PATHEXT`/`where git` 获取，失败时返回空字符串
- `git` 字段每次查询都会实时重新探测，不受 `--agent-cache` 影响
- `router_disable` 是当前规范字段，默认值为 `true`
- `router_disable=true` 表示关闭蜂群，`router_disable=false` 表示开启蜂群
- 为兼容历史 Agent 配置，读取 `config.json` 时仍接受旧字段 `swarm`，并自动转换为新的 `router_disable`
- `skills[].compatibility` 最终固定输出为字符串，因此上游不需要再区分原始 `SKILL.md` 使用的是字符串还是数组写法
- `gateway` 通过系统命令获取 MAC 网关地址（支持 macOS 和 Linux）
- `sys` 通过 `uname -sr` 获取操作系统信息
- 同名技能按扫描顺序后覆盖前
- 除 `skills`、`config.json` 中的 Agent 属性与 `git` 外，其余 Agent 共享字段仍在同一进程生命周期内按 `root + appDir + deviceID` 共同决定缓存键
