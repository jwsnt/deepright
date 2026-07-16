# Proxy 使用手册

## 简介

Proxy 是一个可独立运行的 Golang 模块，既可以作为 HTTP 代理服务接收并转发 `/v1/chat/completions`，也可以作为 CLI 工具直接创建定时任务元数据。

本模块当前同时提供三类能力：

- OpenAI 兼容接口的 SSE 流式代理
- 内嵌 Connect HTTP 服务与 CLI 子命令
- Agent 文件、工作区、命令执行等本地接口
- SKILL 解析告警接口 `/skills_warning`
- 知识库静态映射接口 `/knowledge`
- 知识库最后更新时间接口 `/knowledge_lastUpdate`
- 知识库真实路径接口 `/knowledge_path`
- 文件最后更新时间接口 `/file/lastUpdate`
- 插件元数据接口 `/api/plugins/meta`
- 插件配置接口 `/api/plugins/config`
- 插件日志 SSE 接口 `/api/plugins/log`
- 插件状态接口 `/api/plugins/status`
- 插件启动接口 `/api/plugins/start`
- 插件停止接口 `/api/plugins/stop`
- 插件命令执行接口 `/api/plugins/exec`
- 模型密钥的本地持久化接口 `/api/token`
- Cron 任务创建、查询 CLI 与 HTTP API
- Connect 待处理消息到一次性任务的自动桥接执行（每 30 秒扫描一次 `add-request`，命中后立即转换为 `task_detail`）
- Agent 模型/密钥与 cron 数据库操作审计日志

## `/api/plugins/meta`

- `GET /api/plugins/meta` 每次都会实时读取插件目录，不使用旧的长缓存结果
- 当前只会把以下文件视为插件候选：
  - 无后缀且可执行的程序
  - 后缀为 `.py`、`.js`、`.go` 的脚本文件
- 会跳过目录、隐藏文件以及其他不符合条件的文件
- 如果某个候选文件读取信息或执行 `name` / `param` / `scope` / `command` / `help` 失败，接口会跳过该文件并输出日志，不会因为单个坏文件返回 500
- 每个插件会并发读取 `name`、`param`、`scope`、`command`、`help`，尽量缩短接口总耗时
- 返回值中的每个插件对象包含：
  - `key`
  - `name`
  - `param`
  - `scope`
  - `meta`
  - `router_disable`
- `param` 使用对象数组格式，例如 `[{"appId":""},{"appSecret":""}]`
- `param` 中每一项的 key 是参数名，value 是占位提示；插件未提供提示时返回空字符串
- 其中 `scope` 对应插件规范中的 `scope` 命令，表示该插件支持哪些容器配置项：
  - `reuse`
  - `agent`
  - `provider`
  - `thinking`
  - `swarm`
- 如果插件不实现 `scope` 命令，则默认视为支持全部容器配置，返回 `["reuse","agent","provider","thinking","swarm"]`
- 如果插件显式返回 `[]`，则表示完全不支持容器配置

## `/api/plugins/exec`

- `GET /api/plugins/exec?key=x&command=y&param1=value1&param2=value2...` 用于执行指定插件的指定命令
- `key` 是插件运行时主键
- `command` 是插件命令文本，支持多级子命令，例如 `instance init`
- 其余任意 query 参数都会被转换为 `--param value` 的 CLI 参数并原样透传给插件
- 如果某个参数值为空，则只会透传对应的 `--param`
- `command` 中的空格需要按 URL 规则转义，例如 `instance%20init`
- 如果请求里没有显式传 `connect-bin`，proxy 会自动补齐当前 `proxy` 二进制路径，便于 browser 这类插件直接读取当前运行时上下文
- 返回值与 `/api/plugins/start`、`/api/plugins/stop` 一致，包含实际执行的插件路径、命令参数数组，以及插件输出解析结果

示例：

```bash
curl 'http://127.0.0.1:8080/api/plugins/exec?key=browser&command=instance%20init&agentId=A&chatId=chat-001'
```

也可以通过 CLI 调用：

```bash
./proxy plugins exec --key browser --command 'instance init' --agentId A --chatId chat-001
```

## Metadata 补充字段

- `/v1/chat/completions` 转发时会复用统一的 Agent 元数据输出
- 请求体中显式传入的 `metadata` 会与共享 Agent 元数据合并后一起转发到上游
- 如果请求体里原本已经有同名 `metadata` 字段，则保留请求体传入值
- 当请求体中的 `model` 命中 `token_store` 里对应的模型配置时，还会把该模型下已配置且非空的 `__url`、`__model`、`__model_fast`、`__model_thinking`、`__model_multi_input`、`__model_multi_output` 一并写入转发 `metadata`
- 这些 `__*` 字段只读取当前请求 `model` 命中的那一条配置，不会跨模型取值
- 如果对应字段未配置或值为空字符串，则不会写入转发 `metadata`
- 如果某个 Agent 工作目录下的 `config.json.media` 非空，则会把同一份对象补充到 `metadata.agents[].media`
- 如果当前请求能确定 `metadata.agentId + metadata.chat`，则对应的 `metadata.agent.media` 也会一并补齐
- `media` 是 Agent 维度的 JSON 对象；当前 Site 侧会按 `模型服务商名 -> 多组参数` 的结构写入，例如 `"media":{"gemini":{"aspectRatio":"16:9","imageSize":"2K"}}`
- 其中 `agents[].skills` 会在每次请求时实时遍历 Agent 的 `skills` 目录，不跟随 `--agent-cache` 一起缓存
- 其中 `agents[].description`、`agents[].provider`、`agents[].thinking`、`agents[].router_disable` 也会在每次请求时实时重新读取对应 Agent 的 `config.json`
- 其中 `agents[].version` 来自 `--agent-dir/<agentId>/config.json` 中的 `version`，只在当前 Agent metadata 缓存周期首次扫描时读取一次
- 如果当前请求能确定 `metadata.agentId + metadata.chat`，则还会额外补出 `metadata.agent`
  - `metadata.agent.version`：当前 Agent 的缓存版本号
  - `metadata.agent.sandbox`：共享 sqlite 中该 `chatId` 的实时沙盒模式；未写入时为空字符串
- metadata 中的 `git` 字段也会在每次请求时实时重新探测，不跟随 `--agent-cache` 一起缓存
- 当当前应用启动目录下存在 `knowledge` 目录时，metadata 中会额外包含：

```json
{
  "knowledge": {
    "lastUpdate": 0,
    "path": "/app/knowledge"
  }
}
```

- `path` 为知识库绝对路径
- `lastUpdate` 来自当前启动目录下共享的 `data` sqlite
- `knowledgeCommit` 同样来自共享 `data` sqlite，并按 `agent_id` 独立保存
- 如果当前启动目录还没有知识库，则不会追加 `knowledge` 字段
- 转发 `/v1/chat/completions` 前还会额外检查 `knowledge.lastUpdate`
  - 如果 `lastUpdate` 距离当前请求时间未超过 `--knowledge_update_interval`（默认 `7200000` 毫秒，即 2 小时），则会在转发前删除 `knowledge.lastUpdate`，仅保留 `path`
  - 如果已经超过该时间，则会继续检查共享 sqlite 中的知识库更新申请锁时间
  - 如果最近一次申请更新时间距离当前请求未超过 `--knowledge_update_lock`（默认 `1800000` 毫秒，即 30 分钟），同样会删除 `knowledge.lastUpdate`
  - 只有当知识库确实过期，且最近没有其他请求触发过更新申请时，才会把 `lastUpdate` 原样转发给上游
- 如果请求体 `metadata` 中显式传入 `knowledge_commit`，则会按 `metadata.agentId` 维度把最新提交值写回共享 sqlite 的 `knowledge_runtime.knowledge_commit`
- 如果请求体 `metadata` 中显式传入 `knowledge_commit: true`，则在对应 SSE 响应完整结束后，会把当前时间同时写回该 Agent 的知识库最后更新时间和知识库更新申请锁时间
- 如果请求体 `metadata` 中显式传入 `knowledge_commit: true`，则这次请求转发时会强制保留 `knowledge.lastUpdate`，不再检查 `knowledge_update_interval` 和 `knowledge_update_lock`
- 这套逻辑用于避免并发请求同时触发多次知识库刷新；服务端仍通过是否收到 `lastUpdate` 来决定是否执行更新

例如，请求：

```bash
curl 'http://127.0.0.1:8080/v1/chat/completions' \
  -H 'Content-Type: application/json' \
  -d '{"model":"gpt-4","messages":[{"role":"user","content":"hi"}],"stream":true,"metadata":{"hello":"world","extract":"true"}}'
```

转发到上游时，请求体会补成类似：

```json
{
  "model": "gpt-4",
  "messages": [
    {
      "role": "user",
      "content": "hi"
    }
  ],
  "stream": true,
  "metadata": {
    "hello": "world",
    "extract": "true",
    "__url": "https://provider.example/v1",
    "__model_fast": "deepseek-fast",
    "__model_multi_input": "deepseek-vision"
  }
}
```

- 上面的 `__url`、`__model_fast`、`__model_multi_input` 仅在当前 `model` 对应配置中存在且非空时才会被补充
- 转发到上游的最终报文会统一收口为：
  - `/v1/chat/completions`：`{ "messages": [...], "stream": ..., "metadata": ..., "model": ... }`
  - proxy 内部 cron 执行请求：`{ "messages": [...], "stream": true, "metadata": ..., "model": ... }`
- `thinking`、`html`、`router_disable` 只会保留在 `metadata` 内，不再继续向上游发送旧的顶层布尔字段
- 两条转发链路都会附带 `metadata.agent.version` 与 `metadata.agent.sandbox`
- 如果外部入口仍传入简化的顶层 `message`，proxy 会在转发前归一成单条 `messages`
- `/cli/get`、`/cli/pub` 的协议收口由 `integration` / `cli-get` 模块负责，proxy 当前只覆盖这里列出的两条上游转发链路
- `media` 不跟随 Agent metadata cache；每次真正转发 `/v1/chat/completions` 或 proxy 内部 cron 执行请求前，都会重新读取对应 Agent 最新的 `config.json`

### 当前模块内已覆盖的 metadata 链路

- 已覆盖：
  - `/v1/chat/completions`
  - proxy 内部发起的 cron 执行请求 metadata
- 备忘录相关的 `router_disable` 默认值固定为 `true`
- 右上角 `SWARM` 开关与 `router_disable` 的映射固定为：`SWARM 开启 -> router_disable=false`、`SWARM 关闭 -> router_disable=true`
- cron 真正执行 `task_detail` 时，会显式把该条任务明细自己的 `router_disable` 写入转发 `metadata.router_disable`，不会回退到 Agent `config.json` 的默认值
- 说明：
  - 本模块目录内当前没有实际的 `/cli/get`、`/cli/pub` HTTP 路由实现
  - 如果后续这两条链路由 `integration` 或 `cli-get` 收口，需要在对应模块继续复用同一份 Agent 元数据输出

### `skills` 字段说明

- `metadata.agents[].skills` 直接复用共享 Agent 元数据内核输出
- 每次请求都会重新扫描每个 Agent 的 `skills` 目录及其子孙目录
- 即使 `proxy` 进程使用了较长的 `--agent-cache`，技能文件更新后也会立刻反映到下一次 metadata 注入结果中
- `proxy` 本身不单独维护一份技能缓存
- `metadata.agents[].skills[].compatibility` 最终固定为字符串；如果原始 `SKILL.md` 使用 YAML 列表，会被规范化为以 `; ` 连接的单个字符串

### `provider` 字段说明

- `metadata.agents[].provider` 直接复用共享 Agent 元数据输出
- 该字段来自对应 Agent 工作目录下的 `config.json.provider`
- 如果 `config.json` 不存在，或未声明 `provider`，则该字段输出空字符串
- 该字段会在每次请求时实时重新读取，不跟随 `--agent-cache` 一起缓存
- `/v1/chat/completions`、proxy 内部 cron 执行请求，以及 `/cli/get`、`/cli/pub` 相关的 Agent 元数据透传链路，都应复用同一份 `provider` 输出

### `version` 与 `sandbox` 字段说明

- `metadata.agents[].version` 与 `metadata.agent.version` 都来自 `--agent-dir/<agentId>/config.json` 中的 `version`
- `version` 只在当前 Agent metadata 缓存周期首次扫描时读取一次；缓存未失效前不会因为 `--agent-dir/<agentId>/config.json` 中的 `version` 变化而立刻刷新
- `version/provider` 不会持久化到 sqlite；`version` 只使用当前 `proxy` 进程内的 Agent metadata 内存缓存
- 对应用内置的 `DEF_AGENT`，`proxy` 启动时会把 bundled `default-dir/config.json.version` 同步写回 `--agent-dir/DEF_AGENT/config.json`，以便应用升级后默认 Agent 的 `version` 跟随更新
- `metadata.agents[].sandbox` 与 `metadata.agent.sandbox` 都按当前 `chatId` 实时读取共享 sqlite 的 `cli_sandbox_state`
- `agentId` 不参与沙盒状态命中，仅用于定位当前 Agent metadata 与运行日志
- 如果当前请求没有有效的 `chatId`，或该会话从未写入沙盒状态，则 `sandbox` 输出空字符串

### `git` 字段说明

- `metadata.git` 直接复用共享 Agent 元数据输出
- `/v1/chat/completions` 转发时会在每次请求前实时重新探测 git 安装路径
- 即使 `proxy` 进程使用了较长的 `--agent-cache`，git 路径变化后也会立刻反映到下一次 metadata 注入结果中
- 如果当前机器未安装 git，则该字段为空字符串

## 统一日志

- `proxy` 会把 `/v1/chat/completions` 的请求与响应异步写入当前应用目录下的 SQLite `data`
- 统一日志表为 `agent_message_log`
- 表字段：
  - `agent_id`
  - `chat_id`
  - `content`
  - `log_type`
  - `created_at`
- 索引为 `agent_id + chat_id + log_type + created_at`
- `log_type` 固定取值：
  - `0`：`/v1/chat/completions` 请求
  - `1`：`/v1/chat/completions` SSE 响应分段
  - `2`：`cli/get`
  - `3`：`cli/pub`
- `/v1/chat/completions` 的 SSE 响应仍保持“收到一段记录一段”的方式，不会聚合后再写
- 如果统一日志里存在 `cli/pub` 记录，则其 `content` 会按压缩前原始执行结果保存或兼容解码，导出时可直接查看
- 现有 `chat_log` 仍继续保留给 `/api/restore` 的历史页面/任务会话恢复使用
- `/api/restore` 现在会额外合并返回同一 `agentId + chat` 下的 `cli/get` 与 `cli/pub` 日志，返回记录中的 `role` 分别为 `cli/get`、`cli/pub`

## 最近轮次日志导出

- `proxy` 提供 `GET /log_skill?agentId=xxx&chatId=yyy&round=zzz&start=aaa&close=bbb`
- 作用是按最近 N 轮 `/v1/chat/completions` 请求为边界，并叠加时间范围过滤，导出该 `agentId + chatId` 下符合条件的会话请求、SSE 响应、`cli/get`、`cli/pub` 日志
- 数据源直接读取统一日志表 `agent_message_log`
- 单个请求下的多段 SSE 响应会在导出时合并为一条 Markdown 记录；日志表本身不会被改写
- 输出文件会写入对应 Agent 工作目录下的 `tmp/`
- 返回值包含导出文件的绝对路径和文件大小（`K`）
- 另外还提供 `GET /log_skill_status?agentId=xxx&chatId=yyy`
- 该接口会检查当前会话最近一轮完整 SSE 响应内的 `cli/get` 次数
- 默认阈值来自主应用 `config/config.json.skill_extract`；未配置时默认阈值为 `10`
- 如果显式传入 `round`，会把它当作本次检查使用的阈值覆盖默认值

### HTTP 示例

```bash
curl 'http://127.0.0.1:8080/log_skill?agentId=A&chatId=chat-001&round=3'
curl 'http://127.0.0.1:8080/log_skill?agentId=A&chatId=chat-001&start=2026-05-13%2012:00:00&close=2026-05-13%2012:10:00'
curl 'http://127.0.0.1:8080/log_skill_status?agentId=A&chatId=chat-001'
curl 'http://127.0.0.1:8080/log_skill_status?agentId=A&chatId=chat-001&round=5'
```

返回示例：

```json
{
  "status": 0,
  "path": "/abs/agents/A/tmp/A_chat-001_20260513121009.md",
  "sizeK": 1.23
}
```

### CLI 示例

```bash
./proxy log-skill --agent A --chat chat-001 --round 3
./proxy log-skill --agent A --chat chat-001 --start '2026-05-13 12:00:00' --close '2026-05-13 12:10:00'
```

### 查询条件规则

- `round`：以 `/v1/chat/completions` 请求日志为边界，默认 `1`
- `start`：日志开始时间，默认空
- `close`：日志结束时间，默认空
- 三个条件按 AND 同时生效
- 其中 `round` 和 `start` 至少要提供一个
- 仅传 `start` / `start + close` 时，会按纯时间范围查询，不会额外强制套最近一轮
- 只有传入 `round` 时，才会先按最近 N 轮收缩范围，再继续叠加时间过滤
- 如果实际轮次不足 N，则先从第一轮开始取，再继续叠加时间过滤

### 导出格式

- 导出文件名由 `agentId + chatId + 时间戳` 组合，例如 `agenta_chatb_20260513121009.md`
- 文件格式为 Markdown 表格
- 列顺序固定为：时间、类型、内容
- 时间格式为 `yyyy-MM-dd HH:mm:ss`
- 类型固定显示为：
  - `SSE请求`
  - `SSE响应`
  - `工具请求（cli/get）`
  - `工具响应（cli/pub）`
- 不同类型的“内容”会导出为可读结果，而不是原始报文：
  - `SSE请求`：优先提取 `/v1/chat/completions` 请求中的 `messages[].content`，兼容单条 `message`
  - `SSE响应`：提取 SSE `delta.content`，并在遇到下一条 `cli/get` 或 `cli/pub` 前持续合并
  - `工具请求（cli/get）`：从 `cli/get` 返回任务的 `content` JSON 中提取 `cmd`
  - `工具响应（cli/pub）`：提取 `messages[].content`；如果是历史原始执行结果日志，则直接输出原文

## 编译

```bash
cd cli/module/proxy
/opt/homebrew/bin/go build -o proxy ./
```

## CLI 使用

### 查看帮助

```bash
./proxy --help
```

### 读取已保存模型密钥

```bash
./proxy token
./proxy token --provider deepseek
./proxy token --agentId A --function 完成任务 --thinking 1 --input 2 --total 3 --cache 4 --model deepseek
```

输出示例：

```json
[
  {
    "deepseek": {
      "token": "aaa",
      "__url": "https://api.example.com/v1",
      "__model": "deepseek-chat",
      "__model_fast": "deepseek-fast",
      "__model_thinking": "deepseek-reasoner"
    }
  }
]
```

```json
{
  "deepseek": {
    "token": "aaa",
    "__url": "https://api.example.com/v1",
    "__model": "deepseek-chat",
    "__model_fast": "deepseek-fast",
    "__model_thinking": "deepseek-reasoner"
  }
}
```

说明：

- `proxy token` 会按模型名排序后输出当前模块目录下共享 SQLite `data` 中已保存的全部模型与密钥
- `proxy token --provider MODEL` 只输出指定 provider；如果 provider 不存在，则返回空对象 `{}`
- 当传入 `--agentId`、`--model`、`--function` 以及 token 消费字段时，会改为写入一条消费明细
- 例如：`./proxy token --agentId A --function 完成任务 --thinking 1 --input 2 --total 3 --cache 4 --model deepseek`
- `--timestamp` 可选，单位为毫秒；不传时默认使用当前时间
- 如果不写 `token` 子命令，`proxy` 会进入默认服务启动逻辑，因此这些参数必须写在 `proxy token` 后面

### 启动 HTTP 服务

默认情况下，`proxy` 不写子命令时会直接按服务模式启动：

```bash
./proxy --agent-dir ./agents
```

也支持显式写 `serve`：

```bash
./proxy serve --agent-dir ./agents --port 8080 --host http://127.0.0.1:9998
```

如果希望自定义三方插件开始执行时的推送文案，可以追加：

```bash
./proxy serve --agent-dir ./agents --reply "<开始执行>可通过新消息更新任务"
```

如果希望把新建 Agent 的默认模板改成其他目录，可以追加：

```bash
./proxy serve --agent-dir ./agents --default-dir ./config
```

如果希望调整知识库更新时间判断窗口，可以追加：

```bash
./proxy serve --agent-dir ./agents \
  --knowledge_update_interval 7200000 \
  --knowledge_update_lock 1800000
```

说明：

- `proxy` 进程启动后，会在同一个可执行文件内同时提供 Connect 能力，不需要额外再手工启动 `./connect`
- Connect 相关 HTTP 路径会直接复用当前 `proxy` 进程，包括 `/api/connect/health`、`/api/connect/meta`、`/api/connect/request`、`/api/connect/response`
- `GET /api/agent/init?name=...` 会在创建 Agent 后，把 `--default-dir` 指向目录中的全部内容复制到新 Agent 目录；未显式传参时默认使用应用启动目录下的 `./config`
- 如果 `--default-dir` 不存在、不是目录，或复制过程中出错，请求会直接失败，并回滚刚创建的空 Agent 目录
- 服务启动后还会每分钟执行两类后台任务：
  - 扫描周期任务，补齐未来待执行的 cron 明细
  - 扫描 `connect add-request` 产生的待处理消息，按插件聚合为一条立即执行的一次性任务
- 当插件配置里的 `router_disable=false` 时，这类由 `add-request` 桥接生成的 `task_meta` / `task_detail` 也会继承同一值
- 同时还会每分钟扫描一次 `skills` 根目录（默认使用 `--agent-dir/skills`），把解析失败的 `SKILL.md` 同步到共享 `data` sqlite 的 `skills_warning` 表
- 当前工作目录会被视为应用启动目录，因此知识库路径默认解析为 `./knowledge`，共享 sqlite 默认解析为 `./data`

### 查看 SKILL 解析告警

```bash
curl http://127.0.0.1:8080/skills_warning
curl http://127.0.0.1:8080/skills_warning?refresh=1
```

说明：

- `/skills_warning` 返回当前共享 sqlite 中保存的解析错误列表
- `?refresh=1` 会先立即重新扫描一次，再返回最新结果
- 返回结构为：

```json
{
  "status": 0,
  "data": [
    {
      "path": "/abs/path/SKILL.md",
      "reason": "name 字段无效",
      "time": 1747020000
    }
  ]
}
```

- `path` 为错误 `SKILL.md` 绝对路径
- `reason` 为解析失败原因
- `time` 为最近一次扫描到该错误的 Unix 时间戳（秒）
- 如果某个错误文件修复完成，则会在下一轮扫描后自动从列表中移除

### 查看待安装应用

```bash
curl http://127.0.0.1:8080/install_app
```

说明：

- 当前返回值是 JSON 字符串数组
- 当前已收口的自动检测项为 `git` 和 `python3`
- 主应用 `config/config.json` 可配置 `install_app`，并按当前操作系统读取 `linux`、`wsl`、`mac` 对应数组
- Linux 读取 `install_app.linux`，macOS 读取 `install_app.mac`，Windows/WSL 读取 `install_app.wsl`
- 启动时可通过 `--install_app a,b,c` 追加自定义待安装应用
- `install_app` 中的每个元素都表示一个本地应用名；接口会按当前操作系统检查是否已安装，已安装项不会出现在返回列表中
- 接口会把自动探测结果、`config/config.json` 当前系统对应配置、`--install_app` 指定值做去重合并，并对安装状态缓存 5 分钟
- `config/config.json` 示例：

```json
{
  "install_app": {
    "linux": ["node", "python"],
    "wsl": ["node", "python", "docker"],
    "mac": ["node", "python", "xcode-select"]
  }
}
```

- 如果当前机器未安装 `git` 和 `python3`，则返回：

```json
["git", "python3"]
```

- 如果启动参数为：

```bash
./proxy serve --agent-dir ./agents --install_app node,python,git,python3
```

则接口可能返回：

```json
["git", "node", "python", "python3"]
```

- 如果当前支持的应用都已安装，则返回空数组 `[]`

### 读取共享 `deviceId`

```bash
curl http://127.0.0.1:8080/api/deviceId
```

成功时返回：

```json
{
  "status": 0,
  "deviceId": "your-device-id"
}
```

说明：

- `/api/deviceId` 会返回当前共享 Agent 元数据中的 `deviceId`
- Site 设置页里的复制 `deviceId` 按钮会优先调用这个接口
- 如果 `proxy` 无法读取 Agent 元数据，则会返回 `status=1`

### 获取已开启蜂群的 Agent

```bash
curl http://127.0.0.1:8080/api/swarm_agent
```

成功时返回：

```json
["DEF_AGENT", "planner"]
```

说明：

- `/api/swarm_agent` 仅支持 `GET`；其他方法会返回 `405 Method Not Allowed`
- `/api/swarm_agent` 会实时扫描当前 Agent 元数据，只返回其中 `router_disable=false` 的 Agent ID
- 传入查询参数 `agentId=当前AgentId` 时，返回结果会额外过滤掉当前 Agent 自身
- 返回结果按 Agent 元数据扫描顺序输出
- 如果当前没有任何 Agent 开启蜂群，则返回空数组 `[]`
- Site 居中会话输入框在当前会话开启 `SWARM` 时，会直接复用这个接口填充 `@ Agent` 菜单

### 访问知识库静态映射

```bash
curl http://127.0.0.1:8080/knowledge
curl http://127.0.0.1:8080/knowledge/README.md
curl http://127.0.0.1:8080/knowledge_lastUpdate
curl http://127.0.0.1:8080/knowledge_path
curl 'http://127.0.0.1:8080/knowledge_lastUpdate?agentId=agent-a'
curl 'http://127.0.0.1:8080/knowledge_path?agentId=agent-a'
```

说明：

- `/knowledge` 会映射当前应用启动目录下的 `knowledge` 目录
- 访问目录时返回树形结构文本
- 访问文件时直接返回文件原始内容
- 路径不能跳出 `knowledge` 根目录
- `/knowledge_lastUpdate` 会返回当前知识库最后更新时间，格式为 `yyyy-MM-dd HH:mm`；传 `agentId` 时读取对应 Agent 的记录
- `/knowledge_path` 会返回当前知识库目录的真实文件系统绝对路径；根目录固定为 `--agent-dir/knowledge`，传 `agentId` 时返回 `--agent-dir/knowledge/<agentId>`

### 查看文件最后更新时间

```bash
curl 'http://127.0.0.1:8080/file/lastUpdate?agentId=b&file=USER.md'
curl 'http://127.0.0.1:8080/file/lastUpdate?file=/abs/path/to/USER.md'
```

说明：

- `/file/lastUpdate` 返回目标文件最后更新时间距离当前时间的毫秒数
- `file` 支持绝对路径和相对路径
- 绝对路径会直接按文件系统路径解析，并支持大小写不敏感匹配
- 相对路径会以对应 `agentId` 的 Agent workspace 为根目录解析
- 相对路径场景下，`agentId` 为必填，也兼容 `agent`
- 相对路径拒绝 `..` 越界
- `~` 路径不支持
- 文件和目录都支持
- 例如文件最后更新时间为 `2026-05-06 13:00:00`，当前时间为 `2026-05-06 13:00:05`，则返回：

```text
5000
```

### 使用内嵌 Connect 子命令

```bash
./proxy connect meta-create --key feishu --meta '{"token":"abc"}' --callback ignored --agent A --model OpenAI
```

```bash
./proxy connect meta-get --key feishu
```

```bash
./proxy connect meta-list
```

说明：

- 最终用户统一使用 `proxy connect meta-create` / `proxy connect meta-update` / `proxy connect meta-get` / `proxy connect meta-list`
- `proxy connect ...` 仍保留作内部实现和兼容入口
- `proxy connect ...` 会直接复用 Connect 模块的同一套数据结构与校验逻辑
- 默认使用当前目录下的 `data` 数据库，并自动继承 `--agent-dir` / runtime 中的 `connect-cache`
- 插件二进制仍然保持独立可执行文件，不会被编译进 `proxy`
- `--key` 是插件运行时主键；`--name` 仅保留兼容旧调用
- `--callback` 是兼容占位参数，实际保存时会固定规范化到应用启动目录下的 `plugins/<plugin-key>`

### 持续查看插件日志

```bash
./proxy plugins log --plugin feishu --last 20
```

### 查询插件是否已启动

```bash
./proxy plugins status --name feishu
```

### CLI 查询 SKILL 告警

```bash
./proxy skills-warning
./proxy skills-warning --agent-dir ./agent --refresh
./proxy skills-warning --refresh --root ./agent/custom-skills
```

### CLI 查询文件最后更新时间

```bash
./proxy file-last-update --agent b --file USER.md
./proxy file-last-update --file /abs/path/to/USER.md
```

说明：

- `file-last-update` 会输出毫秒数纯文本
- `--file` 支持绝对路径和相对路径
- 相对路径时需要传 `--agent` 或 `--agentId`
- 相对路径解析根目录与 HTTP 接口保持一致，都是对应 Agent 的 workspace

说明：

- 默认直接读取当前目录 `data` sqlite 中的 `skills_warning`
- `--refresh` 会先扫描再输出
- `--root` 仅在 `--refresh` 时生效，用于指定要扫描的根目录
- 未传 `--root` 时，默认扫描 `--agent-dir/skills`

说明：

- `--name` 支持插件展示名、插件二进制文件名和插件路径
- 底层会调用与 `./connect list-plugins` 相同的插件扫描逻辑解析插件信息
- 默认按插件目录下的同名 `.pid` 文件判断进程是否仍存活，例如 `plugins/feishu` 对应 `plugins/feishu.pid`
- 也支持通过 `--pid-file` 显式覆盖 PID 文件路径

说明：

- `--plugin` 支持插件名、插件路径或日志路径
- 如果传的是插件名 `feishu`，固定读取 `release/plugins/feishu.log`
- CLI 会以 SSE 文本格式持续输出，行为等同 `tail -f`

### 启动或停止插件

```bash
./proxy plugins start --name feishu --connect-bin ./connect
```

```bash
./proxy plugins stop --name feishu --pid-file ../plugins/feishu.pid
```

说明：

- `--name` 同时支持插件展示名、插件二进制文件名和插件路径
- 其余参数会原样透传给插件，例如 `--connect-bin`、`--pid-file`
- 启动时会优先尝试 `<plugin> start`，如果插件只支持旧式参数，则自动回退到 `<plugin> --start`
- 停止时同理，会兼容 `<plugin> stop` 与 `<plugin> --stop`

### 创建或更新插件配置

```bash
./proxy plugins config --key feishu --meta '{"appId":"cli-app","appSecret":"cli-secret"}' --agentId A --model OpenAI
```

说明：

- `--key` 使用插件运行时主键，例如 `feishu`
- `--name` 仅保留兼容旧调用，建议不要再用展示名驱动运行时链路
- `--callback` 即使传入任意值，也只作为兼容占位参数
- 如果该插件尚未配置，则创建 Connect 元数据
- 如果该插件已存在配置，则自动更新原记录
- `callback` 会自动写入插件可执行文件的绝对路径
- 未传 `--chat` / `--chatId` 时，会默认使用插件 `key` 作为 CHAT_ID
- `--stream` 与 `--thinking` 默认都是 `false`

### Connect 待处理消息自动转任务

说明：

- `connect add-request` 写入的待处理消息会在后台每分钟被扫描一次
- 同一插件当前所有待处理消息会按最早消息时间做聚合窗口判断
- 如果当前拼接内容不含文本，仅包含图片或文件，则本轮不处理，也不会修改消息状态
- 如果最早待处理消息距离当前不足 10 分钟，则本轮不处理，也不会修改消息状态
- 如果最早待处理消息已超过 20 分钟，则会转成一条“无需启动”的备忘录明细，`task_detail.started = 2`
- 如果最早待处理消息位于 10 到 20 分钟之间，则会转成一条立即执行的一次性任务内容
- 如果本轮为某个插件至少生成了一条 `started=0` 的待执行明细，则会使用该插件的 `init` 命令，仅回复一次开始通知
- 开始通知内容来自 `--reply`，默认值为 `<开始执行>可通过新消息更新任务`
- `init` 的插件程序路径不会写死；proxy 会先读取 `connect meta-list` 对应配置里的 `callback` 绝对路径，再执行 `<callback> init ...`
- 执行 `init` 或 `send` 前，proxy 会先调用 `<callback> command` 读取插件能力列表，确认插件声明支持对应命令后再执行
- 为兼容尚未升级的旧插件，如果 `command` 不可用，proxy 仍会回退到 `<callback> --help` 检查
- 开始通知同样会把目标原始消息 JSON 作为 `--message` 传给插件，便于插件按原消息上下文回复
- 任务执行会复用该插件 connect 元数据中的 `agentId`、`chatId`、`model`、`thinking`
- 执行前会校验 Agent 是否仍存在、模型是否已注册且 Token 非空
- 备忘录任务明细执行时，请求 `/v1/chat/completions` 的 metadata 会附带 `cron_type`
- 普通周期任务写入 `cron_type=cron`
- 插件桥接生成的任务写入对应插件 `key`，例如 `cron_type=feishu`
- 如果任务明细存在 `response_schema`，则还会把原始 Json String 追加到 `metadata.response_schema`
- 如果当前启动目录存在 `knowledge` 目录，cron 执行请求的 metadata 也会同时带上 `knowledge`
- 执行请求的 metadata 会附带 `META_ID`，内容为本轮聚合消息中的最后一条 request ID
- 后台还会每分钟检查最近 24 小时内已完成的非 `cron` 任务明细
- 如果该明细通过 `META_ID/meta_ref` 关联到的 connect 原始消息状态仍为“已启动”，则会使用对应插件的 `send --message {} --content {}` 命令把任务最终文本结果回推给三方
- 完成回推同样复用 `callback` 绝对路径，避免在 proxy 内写死任何插件二进制位置
- 自动回推使用 `task_detail.result_content` 作为 `--content`，并将目标原始消息 JSON 作为 `--message`
- 如果 `task_detail.result_content` 是 ```json ... ``` 或 ``` ... ``` 包裹的 JSON object / array，proxy 会先去掉 Markdown 外壳并标准化为紧凑 JSON，再传给插件 `send`
- 如果插件自身会记录消息流水日志，则开始通知通常会记为 `init ...`，完成回推会记为 `send ...`
- 回推成功后，当前原始消息状态会更新为“已回复”，并把更早且仍为“已启动”的同插件消息更新为“已完成”
- 同一条已完成明细只会自动回推一次；成功后会写入 `task_detail.replied_at`
- 开始通知要求插件 `command` 返回 `init`；完成回推要求插件 `command` 返回 `send`

### 创建周期任务

```bash
./proxy create --content "每15分钟检查一次上游接口健康" --model "OpenAI" --thinking true --rawTime "2026-05-03 10:00" --cycle 4 --chatId "chat-001" --agent "A"
./proxy create --content "整理日报" --schema '{"type":"object","properties":{"summary":{"type":"string"}},"required":["summary"]}' --model "OpenAI" --rawTime "2026-05-08 09:00" --cycle 0 --agent "A"
```

### 创建自定义 Cron 任务

```bash
./proxy create-cron --cron='10 12 * * 1-5' --agent=A --chat=chat-001 --model=OpenAI --thinking --content='查看天气'
./proxy create-cron --cron='0 18 * * 1-5' --agent=A --model=OpenAI --content='提取结构化日报' --schema '{"type":"object","properties":{"todo":{"type":"array"}},"required":["todo"]}'
```

### 查询任务元数据

```bash
./proxy cron find-meta --content "每15分钟检查一次上游接口健康" --model "OpenAI" --chatId "chat-001"
```

也支持顶层命令：

```bash
./proxy find-meta --agent A --cycle 4 --from "2026-05-03 00:00" --to "2026-05-04 00:00"
```

### 查询任务明细

```bash
./proxy cron find-detail --metaId cron_1 --content "每15分钟检查一次上游接口健康" --model "OpenAI" --cycle 4 --chatId "chat-001"
```

也支持按日期查询：

```bash
./proxy find-detail --agent A --date "2026-05-03"
```

### 删除任务元数据

```bash
./proxy cron delete-meta --id meta_1
```

说明：

- 删除任务元数据时，会同时删除该任务下所有“未完成”的任务明细
- `started = 3` 的已完成明细会保留，便于追溯历史执行结果

### 删除任务明细

删除某个元数据下的全部明细：

```bash
./proxy cron delete-detail --metaId meta_1
```

删除单条明细：

```bash
./proxy delete-detail --detailId detail_1
```

### 兼容命令

- `submit` 等同 `create`
- `submit-cron` 等同 `create-cron`
- `proxy cron ...` 与顶层 `proxy create` / `proxy create-cron` / `proxy find-meta` / `proxy find-detail` 可同时使用
- 删除同样支持 `proxy cron ...` 与顶层 `proxy delete-meta` / `proxy delete-detail`
- `create` / `create-cron` 额外支持 `--schema JSON`
- `--schema` 会写入 `task_meta.response_schema`，周期拆分出的每条 `task_detail` 都会继承它
- `cycle=0` 的一次性任务会把 `--schema` 直接继承到首条明细
- 执行 cron 明细时，如果 `response_schema` 非空，proxy 会把它透传成上游 LLM 请求里的 `response_format.type=json_schema`
- 同时也会把同一份 Json String 透传到上游请求的 `metadata.response_schema`

## Response Schema 透传

- `POST /api/connect/request` 支持 `schema` 参数，CLI `add-request` 也支持新可选参数 `--schema`
- `--schema` 的值为 Json String，最终对应桥接任务明细的 `task_detail.response_schema`
- Connect 待处理消息桥接为一次性 cron 任务时，会继承最后一条 request 的 `responseSchema`
- 桥接生成的 `task_meta`、`task_detail`、cron 查询结果以及日志都会返回 `responseSchema`
- 任务明细开始执行时，proxy 会把这份 Json String 同时透传到上游 `/v1/chat/completions` 请求的 `metadata.response_schema`
- `find-meta` / `find-detail` 的 JSON 结果中，新增字段名为 `responseSchema`

## knowledge 配合建议

- 第一次启动应用前，如果希望 proxy 立刻对外暴露 `metadata.knowledge`，先初始化知识库目录：

```bash
cd /path/to/app
knowledge ensure --app-dir .
proxy --agent-dir ./agent
```

- 如果没有预先初始化：
  - proxy 不会主动创建 `knowledge`
  - metadata 中也不会出现 `knowledge`

## CLI 参数

### 服务启动参数

| 参数 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `--agent-dir` | 是 | - | Agent 根目录 |
| `--default-dir` | 否 | 启动目录下的 `./config` | 新建 Agent 和空 `agent-dir` 启动补齐 `DEF_AGENT` 时使用的默认模板目录 |
| `--port` | 否 | `config/config.json.port`，未配置时为 `8080` | 代理监听端口；显式 `--port` 优先 |
| `--host` | 否 | `https://www.deepright.cn` | 上游服务地址 |
| `--device` | 否 | 自动探测 | 设备 ID |
| `--agent-cache` | 否 | `10000` | Agent 元数据缓存 TTL，单位毫秒 |
| `--site` | 否 | `./site` | 静态站点目录 |
| `--connect_timeout` | 否 | `15000` | 上游连接超时，单位毫秒 |
| `--reply` | 否 | `<开始执行>可通过新消息更新任务` | 三方插件开始执行时的推送文案 |

### `create` 参数

- `--cycle=INT`
  - `0=仅一次`
  - `1=工作日`
  - `2=自然日`
  - `3=每小时`
  - `4=每15分钟`
  - `5=每30分钟`
- `--time='YYYY-MM-DD HH:MM'`
  - 首次开始时间
  - `create` 必填
- `--agent=ID`
  - 绑定的 AgentId
  - 必填
- `--chat=ID` / `--chatId=ID`
  - 会话 ID
  - 可为空
- `--model=NAME`
  - 模型名称
  - 必填
- `--thinking`
  - 带上即表示深度思考为 `true`
- `--content='TEXT'`
  - 任务内容
  - 必填
- `--schema='JSON'`
  - 可选
  - 写入 `response_schema`，执行时透传为 `response_format.json_schema`

### `create-cron` 参数

- `--cron='EXPR'`
  - Cron 表达式
  - `create-cron` 必填
- `--agent=ID`
  - 绑定的 AgentId
  - 必填
- `--chat=ID` / `--chatId=ID`
  - 会话 ID
  - 可为空
- `--model=NAME`
  - 模型名称
  - 必填
- `--thinking`
  - 带上即表示深度思考为 `true`
- `--content='TEXT'`
  - 任务内容
  - 必填
- `--schema='JSON'`
  - 可选
  - 写入 `response_schema`，后续每条明细执行时透传为 `response_format.json_schema`

### `find-meta` 参数

- `--agent=ID` / `--agentId=ID`
  - 按 AgentId 精确匹配
- `--chat=ID` / `--chatId=ID`
  - 按 ChatId 精确匹配
- `--model=NAME`
  - 按模型精确匹配
- `--content='TEXT'`
  - 按内容模糊匹配
- `--cycle=INT`
  - 按执行周期精确匹配
- `--time='YYYY-MM-DD HH:MM'`
  - 查询指定开始执行时间点
- `--date='YYYY-MM-DD'`
  - 查询指定开始执行日期
- `--from='YYYY-MM-DD HH:MM'`
  - 开始执行时间范围起点
- `--to='YYYY-MM-DD HH:MM'`
  - 开始执行时间范围终点

### `find-detail` 参数

- `--meta=ID` / `--metaId=ID`
  - 按元数据 ID 精确匹配
  - 支持 `1` 或 `cron_1`
- `--agent=ID` / `--agentId=ID`
  - 按 AgentId 精确匹配
- `--chat=ID` / `--chatId=ID`
  - 按 ChatId 精确匹配
- `--model=NAME`
  - 按模型精确匹配
- `--content='TEXT'`
  - 按内容模糊匹配
- `--cycle=INT`
  - 按执行周期精确匹配
- `--time='YYYY-MM-DD HH:MM'`
  - 查询指定执行时间点
- `--date='YYYY-MM-DD'`
  - 查询指定执行日期
- `--from='YYYY-MM-DD HH:MM'`
  - 执行时间范围起点
- `--to='YYYY-MM-DD HH:MM'`
  - 执行时间范围终点

### `delete-meta` 参数

- `--id=ID`
  - 按元数据 ID 删除
  - 支持 `1`、`cron_1`、`meta_1`
- 其余过滤条件与 `find-meta` 相同

### `delete-detail` 参数

- `--detailId=ID`
  - 按明细 ID 删除
  - 支持 `1`、`detail_1`
- `--meta=ID` / `--metaId=ID`
  - 删除指定元数据下全部匹配明细
  - 支持 `1`、`cron_1`、`meta_1`
- 其余过滤条件与 `find-detail` 相同

## 定时任务行为

### 任务元数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | int | 自增主键 |
| `cycle` | int | `0/1/2/3/4/5` 或 `-1` |
| `rawTime` | string | 首次执行时间；自定义 Cron 时为空 |
| `agentId` | string | 绑定的 Agent |
| `chatId` | string | 绑定的会话 ID，可为空 |
| `model` | string | 模型名称 |
| `thinking` | bool | 是否深度思考 |
| `cron` | string | 最终保存的 Cron 表达式 |
| `content` | string | 任务内容 |

### 内置周期

- `0` 仅一次
- `1` 工作日
- `2` 自然日
- `3` 每小时
- `4` 每 15 分钟
- `5` 每 30 分钟
- `-1` 自定义 Cron 表达式

### 明细生成规则

- `仅一次`：提交时立即生成一条任务明细
- `工作日 / 自然日`：提交时立即补齐未来 5 天窗口内的明细
- `每小时 / 每15分钟 / 每30分钟`：提交时立即展开未来 5 天窗口内的全部明细
- `自定义 Cron`：创建元数据时不立即展开，由后续检查逻辑补齐
- 任务元数据和任务明细都不会存储模型 Token（密钥）
- 执行链路应在真正执行时按模型动态从 SQLite `token_store` 查询密钥
- 任务元数据查询支持 `AgentId / ChatId / model / content / cycle / 开始执行时间范围`
- 任务明细查询支持 `metaId / AgentId / ChatId / model / content / cycle / 执行时间范围`
- 任务元数据删除支持 `id` 或 `find-meta` 同款过滤条件
- 任务明细删除支持 `detailId`、`metaId` 或 `find-detail` 同款过滤条件
- 通过 `/api/cron/delete` 或 `delete-meta` 删除任务元数据时，只会联动删除尚未完成的任务明细；已完成明细会保留
- 删除 Agent 时，会联动删除该 Agent 关联的全部任务元数据和全部任务明细
- 未指定查询维度表示该维度全部匹配
- 任务明细未指定时间条件时，默认仅返回当前时间之后的数据
- 删除任务明细时，如果未指定时间条件，不会默认限制为未来数据

### 数据库日志

- 模型密钥写入日志表：`proxy_agent_provider_log`
- 任务元数据日志表：`cron_meta_log`
- 任务明细日志表：`cron_detail_log`
- cron 查询、创建、删除、状态更新都会写入审计日志
- 历史日志表若仍是旧名称，会在启动时自动改名

### CHAT_ID

- CLI 创建时支持 `--chat` 或 `--chatId`
- `chatId` 会写入 `task_meta`
- 自动生成的 `task_detail` 会继承相同的 `chatId`
- 创建时会检查指定 Agent 是否存在
- `--agent-dir` 既支持传 Agent 根目录，也支持直接传某个具体 Agent 目录
- 未显式传入的非 cron 模块必填参数会优先从主应用 `config/config.json` 读取，当前主要用于补全 `agent-dir` 与 `device`
- 创建时还会检查指定模型是否已在 `/api/token` 注册且 token 非空

## HTTP 服务

## 启动示例

```bash
./proxy --agent-dir ./agents --site ./site --host http://127.0.0.1:9998
```

### config/config.json

- 每次以 HTTP 服务模式启动 `proxy` 时，都会在当前启动目录下的 `config/config.json` 写入或更新一份启动配置
- 写入内容为本次启动参数的取值
- 每次启动都会覆盖更新

示例：

```json
{
  "app": "/path/to/proxy",
  "app-dir": "/path/to/release",
  "port": 8080,
  "host": "https://www.deepright.cn",
  "agent-dir": "/agent/",
  "default-dir": "/path/to/release/config",
  "device": "",
  "agent-cache": 10000,
  "connect-cache": 10000,
  "site": "../site",
  "connect_timeout": 15000
}
```

### 工作流程

1. 接收客户端的 `/v1/chat/completions` 请求
2. 扫描 Agent 目录并注入 `metadata`
3. 转发到上游 OpenAI 兼容服务
4. 将 SSE 响应原样流式回传

## Connect HTTP 接口

### 健康检查

`GET /api/connect/health`

响应示例：

```json
{
  "status": 0
}
```

### 连接元数据

- `POST /api/connect/meta`
- `PUT /api/connect/meta`
- `DELETE /api/connect/meta`
- `GET /api/connect/meta`

示例：

```text
POST /api/connect/meta?name=飞书&meta={"token":"abc"}&stream=true&callback=./feishu&agent=A&model=OpenAI
```

说明：

- `GET /api/connect/meta?view=config` 会返回配置视图
- 这些接口与 `proxy connect meta-create/meta-update/meta-delete/meta-get/meta-list` 使用同一套底层逻辑

### 请求与响应

- `POST /api/connect/request`
- `GET /api/connect/request`
- `POST /api/connect/response`
- `GET /api/connect/response`

说明：

- `request` 用于写入三方请求
- `response` 用于写入三方响应
- `proxy` 内嵌 Connect 后，插件配置与日志能力也会共用这套本地服务能力

## 插件元数据接口

### 获取可用插件

`GET /api/plugins/meta`

响应示例：

```json
{
  "status": 0,
  "data": [
    {
      "key": "feishu",
      "name": "飞书",
      "param": [
        {
          "appId": ""
        },
        {
          "appSecret": ""
        }
      ],
      "meta": {
        "appId": "cli-app",
        "appSecret": "cli-secret"
      }
    }
  ]
}
```

说明：

- 底层会把本地实时插件扫描结果与 `./connect meta-list` 的已保存配置合并返回
- 接口不会复用 `connect-cache` 的插件发现缓存；每次请求都会实时重新扫描插件并重新读取最新已保存 meta
- 默认扫描当前启动目录下的 `./plugins` 目录
- 只扫描插件目录当前层，不递归子目录
- 仅识别无后缀可执行程序，以及 `.py` / `.js` / `.go` 脚本文件
- 目录和不符合条件的文件会被直接跳过；单个候选文件探测失败时只记日志并继续处理其他插件
- 每个插件都会调用 `<plugin> name`、`param`、`scope`、`command`、`help`
- `key` 字段优先取插件 `name` 输出中的 `key`；未显式提供时回退为文件名，脚本文件会自动去掉扩展名
- `meta` 字段为已配置参数，未配置时返回空对象 `{}`
- `router_disable` 字段来自已保存的 `connect_meta.router_disable`，未配置时默认为 `true`
- 当前接口只支持 `GET`

## 插件配置接口

### 创建或更新插件配置

`POST /api/plugins/config?key=feishu&agentId=A&model=OpenAI`

请求示例：

```text
POST /api/plugins/config?key=feishu&meta=%7B%22appId%22%3A%22cli-app%22%2C%22appSecret%22%3A%22cli-secret%22%7D&stream=true&agentId=A&chatId=chat-001&model=OpenAI&thinking=true&router_disable=false
```

成功响应示例：

```json
{
  "status": 0,
  "data": {
    "id": 1,
    "name": "feishu",
    "meta": "{\"appId\":\"cli-app\",\"appSecret\":\"cli-secret\"}",
    "stream": true,
    "callback": "/abs/path/plugins/feishu",
    "agentId": "A",
    "chatId": "chat-001",
    "model": "OpenAI",
    "thinking": true,
    "router_disable": false,
    "createdAt": "2026-05-05T11:40:00+08:00",
    "updatedAt": "2026-05-05T11:40:00+08:00"
  }
}
```

- `router_disable` 为可选布尔参数，默认 `true`
- 该值会写入共享 `connect_meta.router_disable`

失败响应示例：

```json
{
  "status": 1,
  "content": "agentId is required"
}
```

参数说明：

- `key`：必填，插件运行时主键，必须能匹配到插件 `name` 命令返回的 `key`
- `name`：仅保留兼容旧调用，展示名不能再作为运行时主键
- `meta`：可选，配置表单 JSON 字符串；默认 `{}`，但必须是合法 JSON
- `stream`：可选，是否支持流式回复；默认 `false`
- `agentId`：必填，绑定的 AgentId
- `chatId`：可选，绑定的会话 ID
- `model`：必填，模型名称，且必须已经通过 `/api/token` 注册
- `thinking`：可选，是否深度思考；默认 `false`

行为说明：

- 当前接口只支持 `POST`
- 底层复用 `connectsvc.UpsertPluginConfig`
- 首次调用时使用 Connect 的 `meta-create` 创建配置
- 当同 key 插件配置已存在时，会自动切换为 `meta-update`
- `callback` 不接受外部传入，始终自动解析为插件可执行文件的绝对路径
- 未传 `chatId` 时，会默认使用插件 `key` 作为 CHAT_ID
- 创建或更新失败时，会直接返回失败原因，例如缺少参数、插件不存在、Agent 不存在、模型未注册、`meta` 不是合法 JSON
- CLI `proxy plugins config ...` 与 HTTP 接口复用同一套底层逻辑

## 插件日志接口

### 持续读取插件日志

`GET /api/plugins/log?name=feishu&last=10`

响应头：

```text
Content-Type: text/event-stream; charset=utf-8
```

响应示例：

```text
event: log
data: 2026-05-05 10:00:00,收到消息 A

event: log
data: 2026-05-05 10:00:03,收到消息 B

event: error
data: log file not found: release/plugins/feishu.log
```

参数说明：

- `name`：必填，支持插件名、插件路径或日志路径
- `last`：可选，启动时先补发最后 N 行，默认 `10`

行为说明：

- 插件日志文件路径固定为 `release/plugins/插件名.log`
- 例如 `feishu` 固定读取 `release/plugins/feishu.log`，`email` 固定读取 `release/plugins/email.log`
- 不允许再根据当前工作目录、上级目录或其他候选目录推断日志路径
- 服务会先回放最后 N 行，再继续流式输出新增内容
- 用户主动关闭连接后，服务立即停止读取
- 如果日志文件不存在或在读取过程中被删除，会先推送一条 `event: error`，随后关闭连接
- 当前接口只支持 `GET`
- 为兼容旧调用，当前也接受 `plugin` 查询参数，但新代码建议统一使用 `name`

### `/api/plugins/status`

`GET /api/plugins/status?name=feishu`

说明：

- 使用与 `./connect list-plugins` 一致的插件扫描逻辑定位插件
- 默认按插件目录下的同名 `.pid` 文件判断插件进程是否已启动
- 也支持传 `pid-file` 查询参数覆盖默认 PID 文件路径

示例：

```http
GET /api/plugins/status?name=feishu
```

响应示例：

```json
{
  "status": 0,
  "data": {
    "key": "feishu",
    "name": "飞书",
    "path": "/abs/path/plugins/feishu",
    "pid": 12345,
    "pidFile": "/abs/path/plugins/feishu.pid",
    "started": true
  }
}
```

## 插件启动/停止接口

### 启动插件

`POST /api/plugins/start?name=feishu`

示例：

```text
POST /api/plugins/start?name=feishu&connect-bin=./connect
```

响应示例：

```json
{
  "status": 0,
  "data": {
    "path": "/abs/path/plugins/feishu",
    "command": ["start", "--connect-bin", "./connect"],
    "output": {
      "status": "started",
      "pid": 56157
    }
  }
}
```

### 停止插件

`POST /api/plugins/stop?name=feishu`

示例：

```text
POST /api/plugins/stop?name=feishu&pid-file=../plugins/feishu.pid
```

说明：

- `name` 必填，支持插件展示名、插件文件名或插件路径
- 除 `name` 之外的查询参数都会透传给插件
- 接口会优先执行 `<plugin> start|stop`，失败后自动兼容 `<plugin> --start|--stop`
- 当前两个接口都只支持 `POST`

## 模型密钥接口

### 获取全部模型密钥

`GET /api/token`

响应示例：

```json
{
  "status": 0,
  "models": {
    "openai": {
      "token": "Bearer sk-openai",
      "__url": "https://api.openai.example/v1",
      "__model": "gpt-4.1",
      "__model_fast": "gpt-4.1-mini",
      "__model_thinking": "o3",
      "__model_multi_input": "gpt-4.1-vision",
      "__model_multi_output": "gpt-image-1"
    },
    "kimi": {
      "token": "Bearer sk-kimi",
      "__url": "",
      "__model": "",
      "__model_fast": "",
      "__model_thinking": "",
      "__model_multi_input": "",
      "__model_multi_output": ""
    }
  },
  "updatedAt": {
    "openai": "2026-05-03T14:30:00+08:00",
    "kimi": "2026-05-03T14:30:00+08:00"
  }
}
```

### 批量保存模型密钥

`POST /api/token`

请求体示例：

```json
{
  "models": {
    "openai": {
      "token": "Bearer sk-openai",
      "__url": "https://api.openai.example/v1",
      "__model": "gpt-4.1",
      "__model_fast": "gpt-4.1-mini",
      "__model_thinking": "o3",
      "__model_multi_input": "gpt-4.1-vision",
      "__model_multi_output": "gpt-image-1"
    },
    "kimi": "Bearer sk-kimi"
  }
}
```

说明：

- 模型名称作为唯一键保存
- `token` 为必填，`__url`、`__model`、`__model_fast`、`__model_thinking`、`__model_multi_input`、`__model_multi_output` 可为空
- 如果模型已存在，则更新 token、扩展字段和 `updated_at`
- 每次保存都会额外写一条更新日志到 `proxy_agent_provider_log`
- `POST /api/config` 也额外支持请求体 `{"action":"delete_model","model":"openai"}`，用于删除指定模型配置并写入一条 `delete` 审计日志
- `proxy_agent_provider_log` 会按 `agent_id + chat_id + 时间` 建索引；当前 `/api/token` 写入时 `agent_id` 与 `chat_id` 为空字符串
- 如果数据库里已有历史表 `token_store_log`，启动后会自动改名为 `proxy_agent_provider_log`
- 数据保存在当前模块目录下的 SQLite `data` 文件中的 `token_store` 和 `proxy_agent_provider_log` 表

也兼容单条写入格式：

```json
{
  "model": "openai",
  "token": "Bearer sk-openai"
}
```

### CLI 读取模型密钥

除了 HTTP 接口，也可以直接通过 CLI 读取当前启动目录下共享 SQLite `data` 中保存的模型密钥：

```bash
./proxy token
./proxy token --provider deepseek
```

返回格式：

- 不带 `--provider` 时，输出按模型名排序后的 JSON 数组；每个模型值都是对象，包含 `token`、`__url`、`__model`、`__model_fast`、`__model_thinking`、`__model_multi_input`、`__model_multi_output`
- 带 `--provider` 时，输出单个 JSON 对象；模型值同样为上述对象结构
- 如果指定模型不存在，则输出空对象 `{}`

### 请求转发示例

原始请求：

```json
{
  "model": "gpt-4",
  "messages": [
    {
      "role": "user",
      "content": "你好"
    }
  ],
  "stream": true
}
```

转发后：

```json
{
  "model": "gpt-4",
  "messages": [
    {
      "role": "user",
      "content": "你好"
    }
  ],
  "stream": true,
  "metadata": {
    "deviceId": "...",
    "terminal": "/bin/zsh",
    "git": "/usr/bin/git",
    "gateway": "aa:bb:cc:dd:ee:ff",
    "sys": "Darwin",
    "agents": []
  }
}
```

## 定时任务 HTTP API

### 创建任务

`POST /api/cron/create?agentId=xxx`

周期任务示例：

```json
{
  "content": "查看天气",
  "model": "OpenAI",
  "thinking": true,
  "router_disable": false,
  "rawTime": "2026-05-03 09:30",
  "cycle": 1,
  "chatId": "chat-001",
  "type": "cron"
}
```

自定义 Cron 示例：

```json
{
  "content": "查看天气",
  "model": "OpenAI",
  "thinking": true,
  "router_disable": false,
  "cycle": -1,
  "cron": "10 12 * * 1-5",
  "chatId": "chat-001",
  "type": "cron"
}
```

成功响应：

```json
{
  "status": 0,
  "id": 1,
  "cron": "10 12 * * 1-5",
  "agentId": "xxx",
  "type": "cron"
}
```

说明：

- `type` 默认为 `cron`
- 如需区分 Connect 创建的任务，可写具体模块名，例如 `feishu`
- `router_disable` 默认为 `true`；如果传入 `false`，创建出的 `task_meta` 以及首批 `task_detail` 都会保存该值
- 如果当前 Agent 工作目录下的 `config.json.router_remote` 已保存有效值，则任务明细在真正执行 `/v1/chat/completions` 时会额外注入 `metadata.router_remote`
- 该规则同时覆盖右侧备忘录创建的任务和 Connect/插件桥接生成的任务明细

### 查询任务元数据

`POST /api/cron/detail/metadata`

支持查询参数：

- `agentId` / `agent`
- `chatId` / `chat`
- `type`
- `model`
- `content`
- `cycle`
- `time` / `rawTime` / `date`
- `from` / `to`

返回示例：

```json
{
  "status": 0,
  "data": [
    {
      "id": 1,
      "cycle": 1,
      "rawTime": "2026-05-03 09:30",
      "agentId": "xxx",
      "type": "cron",
      "model": "OpenAI",
      "thinking": true,
      "router_disable": false,
      "cron": "0 30 9 * * 1-5",
      "content": "查看天气",
      "chatId": "chat-001"
    }
  ]
}
```

说明：

- 返回结果中的 `type` 字段表示任务类型，未设置时默认为 `cron`
- 返回结果中的 `router_disable` 字段表示该任务是否关闭 router，未设置时默认为 `true`
- 右上角备忘录列表在悬停查看明细浮层时，会额外展示该条备忘录的 `类型` 列，便于区分普通备忘录任务与 Connect 等模块写入的任务

### 删除任务

`POST /api/cron/delete`

支持查询参数：

- `id` / `metaId` / `meta`
- 或 `find-meta` 同款过滤条件

说明：

- 删除元数据时会同时删除其关联的全部明细

### 删除任务明细

`POST /api/cron/detail/delete`

支持查询参数：

- `detailId` / `detail`
- `metaId` / `meta`
- 或 `find-detail` 同款过滤条件

说明：

- 删除时未指定时间条件，不会默认限制为未来数据

### 查询任务明细

`POST /api/cron/detail/list`

支持查询参数：

- `metaId` / `meta`
- `agentId` / `agent`
- `chatId` / `chat`
- `type`
- `model`
- `content`
- `cycle`
- `time` / `date`
- `from` / `to`

说明：

- 未指定时间条件时，默认仅查询当前时间之后的明细
- `task_detail` 主表索引为 `idx_detail_agent_chat_time_type(agent_id, chat_id, exec_time, task_type)`

### 更新任务明细状态

`POST /api/cron/detail/status?agentId=xxx&detailId=yyy&status=zzz`

## 其他本地接口

### `/api/cmd`

- 仅允许本地请求
- 用于执行本地 Shell 命令
- 如果当前 `chatId` 已配置沙盒模式，则改走对应模式的 `CLI_SANDBOX` helper
  - macOS 使用 `.app/Contents/MacOS/CLI_SANDBOX`
  - WSL/Linux 使用 `helpers/<mode>/CLI_SANDBOX`
- 结果写入共享 `data` SQLite

### 会话沙盒

- 沙盒状态按 `chatId` 维度保存到共享 SQLite 的 `cli_sandbox_state`
- `chatId` 为空时，`/api/sandbox_status` 与 `/api/sandbox=*` 都会直接报错
- 仅支持 3 个有效模式：
  - `filepick`
  - `net`
  - `filepick_net`
- 读取当前会话沙盒模式：

```text
GET /api/sandbox_status?chatId=chat-001
```

- 读取接口只按 `chatId` 命中；即使请求里携带 `agentId`，也不会参与状态定位
- 写入当前会话沙盒模式：

```text
POST /api/sandbox=filepick?agentId=A&chatId=chat-001
POST /api/sandbox=net?agentId=A&chatId=chat-001
POST /api/sandbox=filepick_net?agentId=A&chatId=chat-001
POST /api/sandbox=filepick?agentId=A&chatId=chat-001&dir=%2FUsers%2Fme%2FDesktop
POST /api/sandbox=off?agentId=A&chatId=chat-001
```

- 写接口仍要求 `agentId` 与 `chatId`；其中 `agentId` 只用于日志，`chatId` 用于定位当前会话沙盒状态
- `filepick` / `filepick_net` 可选传入 `dir`，显式持久化当前 `chatId` 对应的 `allowed_dir`；未传时仍按当前系统走目录选择流程
- `off` 表示关闭沙盒，并直接删除该 `chatId` 的数据库记录
- CLI 也使用同一套协议：

```bash
./proxy sandbox --agentId A --chatId chat-001
./proxy sandbox --agentId A --chatId chat-001 --sandbox filepick_net
./proxy sandbox --agentId A --chatId chat-001 --sandbox filepick --dir /Users/me/Desktop
./proxy sandbox --agentId A --chatId chat-001 --sandbox off
```

### `/api/kill`

- 仅允许本地请求
- 用于终止 `/api/cmd` 启动的活动命令
- 日志写入 `kill_log`

### `/api/edit`

- 用于向 Agent workspace 写入或新建文件
- 拒绝越界路径和非法绝对路径
- 当前前端在回写 `config.json` 时，请求体也会兼容额外的 `media` 字段，便于与 `/api/config` 共用同一份 Agent 级多模态输出配置结构

### `media` 字段说明

- `POST /api/config?agentId=xxx` 可以直接持久化 Agent 级别的 `media`
- 当前前端会把 `media` 组织为 `模型服务商名 -> 多组参数`，用于描述多模态输出参数
- 典型 `config.json` 片段如下：

```json
{
  "media": {
    "gemini": {
      "aspectRatio": "16:9",
      "imageSize": "2K"
    }
  }
}
```

### `/api/raw`

- 用于读取文件二进制内容，并以 Base64 字符串返回
- 支持 Agent workspace 下的相对路径
- 支持文件系统绝对路径，并按大小写不敏感方式逐段解析
- 含空格路径可直接通过 URL 编码传入
- `~` 路径不支持

### `/api/agent/init`

- `GET /api/agent/init?name=xxx`
- 仅负责创建新的 Agent 目录
- 创建成功后，会把 `--default-dir` 指向目录中的内容完整复制到该 Agent 目录
- 未显式传 `--default-dir` 时，默认复制当前应用启动目录下的 `config/`
- 如果 `default-dir` 缺失、不是目录或复制失败，会返回错误且不会留下半成品 Agent 目录
- `name` 仍然必须是单段 Agent 名称，不能包含空格或 ` /\\:*?"<>|`

### `/api/agent/create`

- `GET /api/agent/create?agentId=xxx&name=yyy&type=zzz`
- `name` 表示 Agent workspace 内相对路径，不再只限制为单个名称
- 允许传入 `docs/data`、`tmp/a/b` 这类以 `/` 分隔的相对路径
- 每个路径段都必须非空，且不能为 `.`、`..`
- 路径段中不允许包含空格和 `\:*?"<>|`
- 禁止绝对路径、`~`、`../` 等越界写入
- `type=0` 创建目录，`type=1` 创建文件；父目录不存在时会自动补齐
- 已存在、Agent 不存在、参数缺失等既有错误语义保持不变

## 作为子模块调用

```go
proxy := &ProxyServer{
    Host:     "https://www.deepright.cn",
    AgentDir: "./agents",
    DefaultDir: "./config",
    DeviceID: "",
    CacheTTL: 120 * time.Second,
    Client:   NewProxyClient(15 * time.Second),
}

mux := http.NewServeMux()
mux.HandleFunc("/v1/chat/completions", proxy.HandleChatCompletions)
http.ListenAndServe(":8080", mux)
```

## 注意事项

- `proxy` 既能作为 HTTP 服务运行，也能作为 CLI 创建 cron 元数据
- 未写子命令时默认进入服务模式
- 服务启动时如果 `--agent-dir` 指向空目录，会把 `--default-dir` 内容复制到 `DEF_AGENT/`，再确保 `DEF_AGENT/skills` 存在
- 如果空 `agent-dir` 启动补齐时发现 `default-dir` 缺失、不是目录或复制失败，启动会直接报错
- CLI 与 HTTP 创建任务使用同一套 `createCronTask` 逻辑
- SSE 响应会原样流式转发，不做内容改写
- `deviceId` 未显式传入时会自动探测

## 20260524-1 更新

/api/agent/create的name语义升级为Agent workspace内相对路径，支持docs/data格式，每路径段非空、不能为.或..、不能含空格和\:*?"<>|，禁止绝对路径/~\/..\/越界写入，type=0创建目录/type=1创建文件，父目录不存在自动补齐。

## 20260524-2 更新

/api/agent/init 改为从 `--default-dir` 复制默认模板初始化新 Agent；未传参时默认使用应用启动目录下的 `config/`，复制失败会回滚空目录。

## 20260525-1 更新

启动时如果 `--agent-dir` 指向空目录，proxy 也会把 `--default-dir` 内容复制到 `DEF_AGENT/`，再补齐 `DEF_AGENT/skills`；这样服务启动补齐与 `/api/agent/init` 使用同一份默认模板来源。
# 2026-05-24 更新

- Proxy 侧所有和 SWARM 对应的对外参数、返回字段、落库字段统一改为 `router_disable`。
- `router_disable=true` 表示关闭，`router_disable=false` 表示开启。
- 页面上仍显示 `SWARM` 这个开关名，但 CLI 与接口统一使用 `router_disable`。
- 插件配置、插件元数据、即时创建 cron 任务、桥接生成的 `task_meta` / `task_detail` 都以 `router_disable` 为准。
- 历史 `swarm` 只作为旧库迁移来源，不再作为 Proxy 的新入参或规范字段。

---

## 迭代 20260603-1：插件文件类型识别

- `GET /api/plugins/meta` 改为在 `proxy` 模块内直接扫描插件目录，不再依赖外部 `connect list-plugins`
- 当前只把以下文件视为插件候选：
  - 无后缀且可执行的程序
  - 后缀为 `.py`、`.js`、`.go` 的脚本文件
- 目录、隐藏文件和其他无关文件会被直接跳过
- 单个候选文件读取失败或执行 `name`、`param`、`scope`、`command`、`help` 失败时，只输出日志并跳过，不会让接口报错或崩溃

请求：

```text
GET /api/plugins/meta
```

- 每次请求都会实时扫描当前启动目录下的 `./plugins`
- 只扫描当前层，不递归子目录
- 跳过文件时会输出类似 `plugins meta skip <filename>: ...` 的日志

## 迭代 20260606-1：插件远程执行接口

- 新增 `GET /api/plugins/exec`
- 新增 `proxy plugins exec` CLI 子命令
- `command` 支持多级子命令文本，例如 `instance init`
- 除 `key`、`command` 之外的 query / CLI 参数都会转成 `--flag value` 透传给插件

请求：

```text
GET /api/plugins/exec?key=browser&command=instance%20init&agentId=A&chatId=chat-001
```

- `key` 必填，`command` 必填
- `command` 里的空格需要按 URL 规则转义
- 其他参数数量不限，会按名字排序后透传给插件

CLI 用法：

```bash
./proxy plugins exec --key browser --command 'instance init' --agentId A --chatId chat-001
```

## 迭代 20260610-2：插件参数结构收口

- `GET /api/plugins/meta` 中每个插件的 `param` 已统一改为对象数组格式
- 返回示例从旧的 `["appId","appSecret"]` 调整为 `[{"appId":""},{"appSecret":""}]`
- 对象中的 key 是参数名，value 是占位提示；未提供提示时返回空字符串
- `proxy/USER_GUIDE.md` 与本次迭代手册已同步更新为新格式

---

## 迭代 20260614-1：/install_app 区分操作系统与已安装检测

本轮迭代把 `GET /install_app` 的配置来源升级为主应用 `config/config.json` 的按操作系统结构，同时保留 `--install_app` 作为额外追加项。

适用规则：

- Linux 读取 `install_app.linux`
- macOS 读取 `install_app.mac`
- Windows 和 WSL 读取 `install_app.wsl`
- `--install_app` 依旧使用逗号分隔字符串，并与当前系统配置、自动探测结果统一去重合并
- 每个 `install_app` 元素都表示一个本地应用名；当前系统如果已安装该应用，就不会出现在 `/install_app` 返回中
- 应用安装状态会缓存 5 分钟

`config/config.json` 示例：

```json
{
  "install_app": {
    "linux": ["node", "python"],
    "wsl": ["node", "python", "docker"],
    "mac": ["node", "python", "xcode-select"]
  }
}
```

示例：

```bash
./proxy serve --agent-dir ./agents --install_app git,python3
curl http://127.0.0.1:8080/install_app
```

接口返回会合并：

- 当前机器自动探测缺失的 `git`、`python3`
- `config/config.json` 中当前操作系统对应的数组
- `--install_app` 传入的额外条目

---

## 迭代 20260614-2：Token 消费记录查询

本次迭代为 `proxy token` 增加了本地 Token 用量查询能力，并保留原有模型密钥读取与消费写入行为不变。

## 查询入口

- 兼容顶层查询写法：

```bash
proxy token --n 500
proxy token --n 500 --start "2026-06-14 12:00:00" --close "2026-06-14 14:00:00"
```

- 新增独立子命令：

```bash
proxy token get --n 500
proxy token get --n 500 --start "2026-06-14 12:00:00" --close "2026-06-14 14:00:00"
proxy token get --help
```

- 可选参数：
  - `--agentId` / `--agent`：仅查询指定 AgentId
  - `--n`：最近 N 条，默认 `500`
  - `--start`：开始时间
  - `--close`：结束时间，默认当前时间

## 时间格式

- `--start` 与 `--close` 支持两种格式：
  - `yyyyMMdd-hhmmss`
  - `YYYY-MM-DD HH:MM:SS`

示例：

```bash
proxy token get --n 100 --start "20260614-120000" --close "20260614-140000"
proxy token get --n 100 --start "2026-06-14 12:00:00" --close "2026-06-14 14:00:00"
```

## 输出结构

- 查询输出与 `/api/consume` 保持同一套数据结构：
  - `status`
  - `details`
  - `summary`
- `details` 为命中的消费明细
- `summary` 为按 `model` 聚合后的 `thinking`、`input`、`total`、`cache`
- CLI 在“最近 N 条”模式下会先取最新记录，再按时间升序输出，便于阅读

示例输出：

```json
{
  "status": 0,
  "details": [
    {
      "thinking": 11,
      "input": 21,
      "total": 31,
      "cache": 2,
      "model": "deepseek-chat",
      "agentId": "demo-agent",
      "function": "cli/get",
      "timestamp": 1781409720000
    }
  ],
  "summary": [
    {
      "model": "deepseek-chat",
      "thinking": 11,
      "input": 21,
      "total": 31,
      "cache": 2
    }
  ]
}
```

## 兼容性

- 以下旧命令保持不变：

```bash
proxy token
proxy token --provider deepseek
proxy token --agentId demo-agent --model deepseek-chat --function cli/get --thinking 10 --input 20 --total 30 --cache 5
```

- 只有传入 `--n`、`--start`、`--close`，或显式使用 `token get` 时，才会进入本地 token 用量查询模式

---

## 迭代 20260614-3：技能动态注入

## 本次更新

- `GET /api/skills?agentId=xxx` 仍然先返回 Agent 自身的技能名
- 主应用 `config/config.json` 中新增的 `skills` 数组会按顺序追加到返回结果
- `__internal_cron` 不再由接口硬编码追加，改为完全由主应用 `config/config.json.skills` 控制
- 当 `browser` 插件处于开启状态时，接口会追加 `__internal_browser`
- 当 `remote` 插件处于开启状态时，接口会追加 `__internal_remote`
- 返回结果会自动去重，保留首次出现的顺序

## 配置方式

主应用 `config/config.json` 示例：

```json
{
  "skills": [
    "__internal_cron",
    "__internal_demo"
  ]
}
```

## HTTP 用法

请求：

```text
GET /api/skills?agentId=A
```

返回示例：

```json
[
  "__internal_F",
  "__internal_cron",
  "__internal_demo",
  "__internal_browser",
  "__internal_remote"
]
```

说明：

- `agentId` 仍为必填
- `config/config.json.skills` 中声明了什么，接口就追加什么
- `browser`、`remote` 两个内部技能只会在对应插件实际开启时追加
- 如果 `skills`、Agent 自身技能、插件内部技能之间出现重名，只保留一份

## 同步结果

- `proxy/main.go` 已改为本地组装 `/api/skills` 返回结果
- `proxy/main_test.go` 已改为覆盖 config 驱动技能和运行中插件技能场景
- 本迭代手册对应当前目录下的 `REQUIREMENT.md`

---

## 迭代 20260618-1：沙盒模式与系统隔离

## 本次更新

- `GET /api/skills?agentId=xxx` 会先返回 Agent 自身技能名
- 主应用 `config/config.json.skills` 会按顺序追加到结果
- `__internal_cron` 不再硬编码，改为完全由 `config/config.json.skills` 控制
- `browser` 插件处于已开启状态时，结果追加 `__internal_browser`
- `remote` 插件处于已开启状态时，结果追加 `__internal_remote`
- `/api/cmd` 的沙盒 helper 路径解析新增 WSL/Linux 产物支持，mac 路径保持不变

## 示例

主应用 `config/config.json`：

```json
{
  "skills": [
    "__internal_cron",
    "__internal_demo"
  ]
}
```

请求：

```text
GET /api/skills?agentId=A
```

当 `browser`、`remote` 插件均已启动时，返回示例：

```json
[
  "__internal_F",
  "__internal_cron",
  "__internal_demo",
  "__internal_browser",
  "__internal_remote"
]
```

## 同步结果

- `proxy/main.go` 继续保持按插件运行状态动态追加内部技能
- `proxy/main.go` 的沙盒 helper 路径解析同时支持：
  - mac `.app/Contents/MacOS/CLI_SANDBOX`
  - WSL/Linux `helpers/<mode>/CLI_SANDBOX`
- `proxy/main_test.go` 已补充 WSL/Linux helper 路径解析测试

---

## 迭代 20260709-1：会话沙盒改为按 `chatId` 命中

## 本次更新

- 会话沙盒状态从 `agentId + chatId` 改为仅按 `chatId` 保存与命中
- `/api/sandbox_status` 改为只依赖 `chatId`；即使请求里携带 `agentId`，也不会参与状态定位
- `/api/sandbox=*` 写接口仍要求 `agentId` 与 `chatId`；其中 `agentId` 仅用于日志，`chatId` 用于写入当前会话沙盒状态
- `metadata.agent.sandbox` 与 `metadata.agents[].sandbox` 都改为按当前 `chatId` 实时读取共享 sqlite
- `/api/cmd` 的沙盒命中改为只看 `chatId`
- 跨系统 helper 选择保持不变：macOS 继续走 `CLI_SANDBOX.app`，WSL/Linux 继续走 `helpers/<mode>/CLI_SANDBOX`

## 行为说明

- `chatId` 为空时，读写都会直接报错
- `off` 会删除当前 `chatId` 的记录；无记录视为 `off`
- `filepick` / `filepick_net` 如显式传入 `dir`，会把 `allowed_dir` 按当前 `chatId` 持久化；未传时继续按当前系统走目录选择流程
- 写入日志会输出 `agentId`、`chatId` 以及 `from -> to` 的文本变更信息
