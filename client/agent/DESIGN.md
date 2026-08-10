# Agent 模块详细技术设计

## 设计目标

+ 将 Agent 元数据扫描能力收敛到唯一内核 `agentcore`，避免 `agent`、`integration`、`proxy`、`cli-get` 等入口各自维护一套扫描逻辑。
+ 对外同时支持两种使用方式：
    + 作为可独立运行的 CLI：扫描指定 Agent 根目录并输出 JSON。
    + 作为可复用的 Golang 子模块：为其他模块提供结构化查询能力。
+ 在保证输出结构稳定的前提下，引入“共享字段缓存 + 动态字段实时刷新”的混合策略，兼顾性能与实时性。
+ 保持与 `skills`、`knowledge`、`sandboxstate` 模块的边界清晰：Agent 只负责编排和汇总，不重复实现下游模块的解析规则。

## 模块边界

+ `main.go`
    + 负责 CLI 参数解析、调用 `agentcore`、格式化输出、错误退出。
    + 不承载扫描、缓存、插件探测、知识库探测等核心逻辑。

+ `agentcore/core.go`
    + 是 Agent 模块唯一的能力内核。
    + 负责目录扫描、配置读取、系统环境探测、插件状态探测、知识库元数据探测、缓存、结构化查询 API。

+ 外部依赖模块
    + `skill-scanner/skillscore`：递归扫描 `skills` 目录并解析技能 front matter。
    + `knowledge/knowledgecore`：解析知识库目录和 sqlite 数据文件位置。
    + `connect/sandboxstate`：按 `agentId + chatId` 查询实时沙盒模式。

## 目录模型

+ 输入根目录 `root` 下的每个直接子目录都视为一个 Agent。
+ 只扫描一层子目录作为 Agent，不把更深层目录识别为独立 Agent。
+ 每个 Agent 子目录本身就是其工作目录 `workspace`。
+ 目录命名规则：
    + `workspace`：Agent 目录绝对路径。
    + `agentId`：Agent 目录名。

## 数据模型

### Agent

`agentcore.Agent` 负责承载单个 Agent 的结构化元数据：

```json
{
  "description": "Agent 描述",
  "provider": "模型供应商",
  "router_disable": true,
  "thinking": false,
  "version": "版本号",
  "sandbox": "filepick | net | filepick_net | \"\"",
  "workspace": "/abs/path/to/agent",
  "agentId": "demo",
  "soul": "SOUL.md 内容",
  "user": "USER.md 或 user.md 内容",
  "skills": []
}
```

字段语义：

+ `description`、`provider`、`thinking`、`router_disable`：来自 Agent 目录下的 `config.json`。
+ `version`：来自 `config.json`，但只在缓存重建时更新。
+ `sandbox`：来自共享 sqlite 的实时查询结果，不参与缓存。
+ `skills`：来自 `skills` 目录实时扫描结果，不参与缓存。
+ `soul`：读取 `SOUL.md`，不存在时为空字符串。
+ `user`：优先读取 `USER.md`，若不存在则回退 `user.md`，仍不存在则为空字符串。

### Output

`agentcore.Output` 是完整输出模型：

```json
{
  "timezone": "Asia/Shanghai",
  "deviceId": "stable-device-id",
  "terminal": "/bin/zsh",
  "plugins": ["browser"],
  "knowledge": {
    "lastUpdate": 1710000000,
    "path": "/abs/path/to/knowledge"
  },
  "git": "/usr/bin/git",
  "gateway": "xx:xx:xx:xx:xx:xx",
  "sys": "Darwin 24.0.0",
  "app": "/abs/path/to/current/executable",
  "agents": []
}
```

字段分层：

+ 共享运行时字段：
    + `timezone`
    + `deviceId`
    + `terminal`
    + `plugins`
    + `knowledge`
    + `git`
    + `gateway`
    + `sys`
    + `app`
+ Agent 数组字段：
    + `agents[]`

### Knowledge

+ `path`：应用目录下知识库的绝对路径。
+ `lastUpdate`：来自共享 sqlite 的 `knowledge_runtime.last_update`，未命中时为 `0`。
+ 如果知识库目录不存在，则整个 `knowledge` 字段不输出。

### PluginRuntime

+ `key`：插件运行时主键。
+ `callback`：插件回调路径，用于推导 PID 文件目录。

该结构由当前可执行程序返回的插件元数据 JSON 反序列化得到，Agent 模块只消费，不负责定义插件 schema。

## 配置文件设计

### Agent `config.json`

当前读取字段：

```json
{
  "description": "string",
  "provider": "string",
  "router_disable": true,
  "swarm": false,
  "thinking": false,
  "version": "string"
}
```

兼容策略：

+ `router_disable` 是当前标准字段，默认值为 `true`。
+ 若未配置 `router_disable` 但配置了历史字段 `swarm`，则做反向映射：
    + `swarm=true` => `router_disable=false`
    + `swarm=false` => `router_disable=true`
+ `thinking` 默认值为 `false`。
+ `description`、`provider`、`version` 默认值为空字符串。
+ `sandbox` 不从 `config.json` 读取，始终走运行时查询。

### 技能文件

+ Agent 模块不直接解析 `SKILL.md`。
+ 技能解析完全委托 `skillscore.ScanAgentSkills()`。
+ 继承 `skills` 模块的兼容策略：
    + 支持 `SKILL.md`
    + Agent 场景额外兼容无后缀的 `SKILL`
    + `compatibility` 兼容字符串和字符串数组，最终统一为单个字符串输出

## 核心流程设计

### 1. 入口调用

CLI 与子模块 API 最终都会进入以下能力链路之一：

+ `GetOutput()` / `GetOutputForApp()`
+ `GetOutputForAppAndChat()`
+ `GetOutputJSON()` / `GetOutputJSONForApp()` / `GetOutputJSONForAppAndChat()`
+ `GetAgentIDs()`
+ `GetAgentByID()`
+ `GetSkillNames()`

其中：

+ `main.go` 的无 flag 场景调用 `GetAgentOutputForAppAndChat()`。
+ `--list` 走 `GetAgentIDs()`。
+ `--get` 先拿完整输出，再在内存中过滤指定 `agentId`。
+ `--skills` 走 `GetSkillNames()`。

### 2. App 目录解析

`resolveAppDir(root, appDir)` 负责统一应用目录语义：

+ 若显式传入 `appDir`，优先使用其绝对路径。
+ 若未传，则使用 `root` 的父目录。

这个目录用于：

+ 解析知识库目录位置。
+ 定位共享 sqlite 数据库。
+ 查询 `agentId + chatId` 维度的沙盒状态。

### 3. Agent 目录扫描

`scanAgents(root)` 的处理步骤：

1. 将 `root` 归一化为绝对路径。
2. 读取 `root` 的直接子项。
3. 仅处理目录项，跳过普通文件。
4. 对每个 Agent 目录读取：
    + `SOUL.md`
    + `USER.md`，不存在时回退 `user.md`
    + `skills/` 目录
    + `config.json`
5. 组装为 `[]Agent` 返回。

容错策略：

+ `SOUL.md` / `USER.md` / `user.md` 缺失：返回空字符串，不报错。
+ `skills/` 缺失：返回空数组，不报错。
+ `config.json` 缺失或 JSON 非法：回退默认配置，不报错。
+ `skills/` 存在但技能扫描发生错误：整体扫描失败并向上返回错误。
+ 根目录不可访问：整体扫描失败。

### 4. 共享运行时字段探测

`buildOutput()` 在缓存未命中时构建完整结构：

+ `timezone`
    + 优先读环境变量 `TZ`
    + 否则读 `time.Now().Location()`
    + 再回退 `/etc/localtime`
    + 最后回退 `time.Now().Zone()`

+ `deviceId`
    + 优先使用外部传入值
    + 否则自动生成
    + macOS 优先读硬件 UUID
    + Linux 优先读 `/etc/machine-id` 或 `/var/lib/dbus/machine-id`
    + 都不可得时回退为 `GOOS:GOARCH` 的 SHA-256 截断值

+ `terminal`
    + 读取环境变量 `SHELL`

+ `git`
    + macOS/Linux：优先 `exec.LookPath("git")`，再回退 `command -v git`
    + Windows：优先查找 `git.exe/git.cmd/git.bat/git`，再回退 `where git`
    + 统一输出绝对路径

+ `gateway`
    + macOS：通过 `route + arp` 获取默认网关 MAC
    + Linux：通过 `ip route + arp` 获取默认网关 MAC
    + Windows 当前返回空字符串

+ `sys`
    + 通过 `uname -sr` 获取

+ `app`
    + 通过 `os.Executable()` 获取当前可执行程序绝对路径

+ `plugins`
    + 基于 `app` 定位同级 `plugins` 目录
    + 再通过当前程序自身执行插件元数据查询命令

+ `knowledge`
    + 基于解析后的 `appDir` 判断知识库目录和数据表

### 5. 插件探测

插件探测分两步：

#### 插件元数据枚举

`detectPlugins(appPath)` 会执行当前程序自身：

+ 如果程序名是 `integration` / `proxy` 或其平台变体：
    + 执行 `connect meta-list`
+ 否则：
    + 执行 `list-meta`

返回值需满足：

+ 能成功反序列化为 `[]PluginRuntime`
+ 每项包含 `key`，可选 `callback`

#### 插件运行态过滤

`DetectRunningPluginKeys()` 会进一步筛选真正“已启动”的插件：

+ 根据 `key` 和 `callback` 推导 PID 文件路径
+ 当前默认 PID 文件为 `<plugin-dir>/<key>.pid`
+ 若提供 `callback`，则优先使用 `callback` 所在目录中的 `<key>.pid`
+ 读取 PID 后检查进程是否存活
+ 仅返回“已配置且进程仍在运行”的插件 key
+ 最终结果去重、排序后输出

### 6. 知识库元数据探测

`lookupKnowledge(appDir)` 的规则：

+ 先通过 `knowledgecore.KnowledgeDir(appDir)` 解析知识库目录。
+ 若目录不存在，直接返回 `nil`。
+ 若目录存在，则至少返回：
    + `path`
    + `lastUpdate=0`
+ 再通过 `knowledgecore.DBPath(appDir)` 定位 sqlite。
+ 若 sqlite 存在，则查询 `knowledge_runtime` 表的 `last_update`。
+ 查询失败不视为致命错误，保留 `lastUpdate=0`。

该设计保证 Agent 模块只读共享知识库状态，不承担初始化知识库的职责。

### 7. 沙盒状态探测

`refreshAgentSandboxFields(output, appDir, chatID)` 的规则：

+ 每次调用先把所有 `agents[i].sandbox` 清空。
+ 仅当 `chatID` 非空且 `appDir` 可解析且 sqlite 文件存在时才继续查询。
+ 对每个 Agent 调用 `sandboxstate.Get(db, agentId, chatID)`。
+ 如果记录存在，则用 `sandboxstate.NormalizeMode()` 归一化到：
    + `filepick`
    + `net`
    + `filepick_net`
+ 若无记录或值非法，则输出空字符串。

设计意图：

+ `sandbox` 是强实时字段，不能绑定到 `--agent-cache`。
+ `sandbox` 与会话相关，因此必须通过 `chatId` 显式注入，不能混入静态 Agent 配置。

## 缓存设计

### 缓存结构

Agent 模块使用进程内单槽缓存：

```go
type outputCache struct {
    mu      sync.Mutex
    root    string
    appDir  string
    device  string
    output  *Output
    expires time.Time
}
```

特征：

+ 整个进程只有一份缓存。
+ 缓存键由 `root + resolvedAppDir + deviceID` 共同决定。
+ 命中缓存时会先深拷贝，再做动态字段刷新，避免直接修改缓存内对象。

### 缓存命中路径

`GetOutputForApp()` 命中缓存后执行：

1. `cloneOutput(structCache.output)`
2. `refreshAgentSkills(cached)`
3. `refreshDynamicOutputFields(cached)`
4. 返回刷新后的副本

其中 `refreshDynamicOutputFields()` 当前包含：

+ `refreshAgentConfigFields()`
    + 实时刷新 `description`
    + 实时刷新 `provider`
    + 实时刷新 `router_disable`
    + 实时刷新 `thinking`
+ `detectGit()`
    + 实时刷新 `git`

`GetOutputForAppAndChat()` 则在此基础上额外执行：

+ `refreshAgentSandboxFields()`

### 缓存未命中路径

缓存未命中时：

1. 调用 `buildOutput()` 完整重建输出。
2. 将结果写入缓存。
3. 再对返回副本执行一次 `refreshDynamicOutputFields()`。

这样设计的原因：

+ 保证首次请求和命中缓存请求在动态字段行为上保持一致。
+ 避免首次构建与热刷新逻辑不一致。

### 动态字段与静态字段边界

当前设计中，以下字段是实时刷新的：

+ `agents[].skills`
+ `agents[].description`
+ `agents[].provider`
+ `agents[].router_disable`
+ `agents[].thinking`
+ `git`
+ `agents[].sandbox`（仅 `GetOutputForAppAndChat`）

以下字段跟随缓存生命周期刷新：

+ Agent 集合本身增删
+ `agents[].workspace`
+ `agents[].agentId`
+ `agents[].soul`
+ `agents[].user`
+ `agents[].version`
+ `timezone`
+ `deviceId`
+ `terminal`
+ `plugins`
+ `knowledge`
+ `gateway`
+ `sys`
+ `app`

其中 `version` 被刻意设计为缓存字段，满足“启动期读取一次”的需求。

## API 设计

### 结构化 API

+ `GetOutput(root, deviceID, ttl)`
+ `GetOutputForApp(root, appDir, deviceID, ttl)`
+ `GetOutputForAppAndChat(root, appDir, deviceID, ttl, chatID)`
+ `GetOutputWithPlugins(root, deviceID, ttl, pluginDir, items)`
+ `GetOutputWithPluginsForApp(root, appDir, deviceID, ttl, pluginDir, items)`
+ `GetOutputWithPluginsForAppAndChat(root, appDir, deviceID, ttl, chatID, pluginDir, items)`
+ `GetAgentIDs(root, deviceID, ttl)`
+ `GetAgentByID(root, deviceID, ttl, agentID)`
+ `GetSkillNames(root, deviceID, ttl, agentID)`
+ `FlushCache()`

设计原则：

+ 结构化 API 优先复用统一内核，避免 JSON 反序列化往返。
+ `GetOutputWithPlugins*` 提供对上游插件列表的显式注入能力，便于 `integration` / `proxy` 复用自身已知的插件运行态。

### JSON API

+ `GetOutputJSON(root, deviceID, ttl)`
+ `GetOutputJSONForApp(root, appDir, deviceID, ttl)`
+ `GetOutputJSONForAppAndChat(root, appDir, deviceID, ttl, chatID)`

这些 API 只是对结构化结果做 `json.MarshalIndent`，不再额外引入新行为。

### 查询 API 语义

+ `GetAgentIDs()`
    + 返回当前输出中的 `agentId` 列表

+ `GetAgentByID()`
    + 命中则返回 `*Agent`
    + 不存在时返回 `nil, nil`

+ `GetSkillNames()`
    + 从指定 Agent 的 `skills` 中提取 `name`
    + 若 Agent 不存在则返回错误

## CLI 设计

### 参数

+ `--agent-cache`
    + 缓存 TTL，单位毫秒，默认 `10000`

+ `--device`
    + 显式指定设备 ID

+ `--app-dir`
    + 显式指定应用目录，用于知识库与共享 sqlite 解析

+ `--chatId`
    + 指定会话 ID，用于实时补充 `sandbox`

+ `--list`
    + 输出所有 `agentId`

+ `--get <id>`
    + 输出指定 Agent 元数据

+ `--skills <id>`
    + 输出指定 Agent 的技能名称数组

### 行为

+ 无子命令时输出完整 JSON。
+ 参数错误、查询失败、Agent 不存在都会写 stderr 并以非 0 退出。
+ CLI 只是薄包装层，不维护额外状态。

## 错误处理设计

+ 可选资源缺失时，尽量降级为空值而不是失败：
    + `SOUL.md`
    + `USER.md`
    + `user.md`
    + `config.json`
    + `skills/`
    + `knowledge` 目录
    + sqlite 记录

+ 基础输入不可用时，直接失败：
    + 根目录不可读
    + Agent `skills` 目录存在但扫描函数返回硬错误
    + JSON 序列化失败

+ `sandbox`、`knowledge`、`plugins` 的运行时读取失败默认降级，不阻塞 Agent 主输出。

这种设计的目标是：Agent 模块优先保证“主元数据可用”，外部运行态补充信息尽量 best-effort。

## 并发与可变性设计

+ 通过 `structCache.mu` 保护缓存读写。
+ 返回前使用 `cloneOutput()` 深拷贝：
    + 拷贝 `Plugins`
    + 拷贝 `Knowledge`
    + 拷贝 `Agents`
    + 拷贝每个 Agent 的 `Skills`

这样可以保证：

+ 缓存对象不会因为动态刷新被就地修改。
+ 调用方拿到的结果可以安全读写，而不影响进程内缓存。

## 与整体架构的关系

+ Agent 模块是“能力内核 + CLI 壳”的典型实现。
+ `agentcore` 是 Agent 元数据能力的 authoritative implementation。
+ 其他模块必须复用 `agentcore`，不能再复制扫描逻辑。
+ 这符合整体设计文档中“单一实现原则”“入口薄包装原则”“integration 二进制/CLI 收口原则”。

## 测试设计

当前测试覆盖以下关键行为：

+ `git` 字段输出与实时刷新
+ `knowledge` 字段存在/不存在两种路径
+ `provider`、`description`、`thinking`、`router_disable` 的读取与热刷新
+ `version` 缓存、`sandbox` 实时刷新
+ `skills` 热刷新
+ `compatibility` 数组归一化为字符串
+ 插件探测仅返回“已配置且正在运行”的插件
+ 工具函数：
    + `ParseInstallApps`
    + `MergeInstallApps`
    + `ResolveExecutablePath`
    + `GenerateDeviceID`
    + `PluginPIDFiles`
    + `readAgentConfig`
    + `cloneOutput`

测试目标不是重复验证下游模块内部实现，而是确保 Agent 模块自己的编排策略、缓存边界和对外契约稳定。

## 已知约束

+ 当前缓存是单槽缓存，不支持多个不同 `root/appDir/deviceID` 结果并存。
+ Agent 集合的增删、`SOUL.md`、`USER.md`、`version` 变化需要等待缓存失效或显式 `FlushCache()` 才会反映。
+ `plugins` 与 `knowledge` 当前也跟随缓存刷新，不是强实时字段。
+ `gateway` 与 `sys` 的探测依赖平台命令，在部分环境中可能为空。
+ Windows 下 `gateway` 目前未实现。

## 后续演进原则

+ 若新增 Agent 维度字段，优先判断它属于：
    + 缓存字段
    + 热刷新字段
    + 会话字段

只有字段刷新语义明确后，才能放入 `buildOutput()` 或某个 `refresh*()` 流程。

+ 若新增运行时依赖模块，Agent 只做只读编排，不重复下沉实现。
+ 若下游入口需要新增展示形式，应扩展 `main.go` 或新增包装 API，不得破坏 `agentcore` 的单一事实源角色。
