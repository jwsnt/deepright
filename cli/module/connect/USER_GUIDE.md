# Connect 使用手册

## 简介

`connect` 是一个独立的 Golang 模块，用来维护三方连接元数据，并接收三方请求、推送处理结果。

当前设计下，`connect` 本身是一个本地 HTTP 服务：

- 服务通过 `./connect start` / `./connect stop` 管理
- 业务命令如 `meta-create`、`add-request`、`add-response` 不再直接访问数据库
- 所有业务命令都会通过 `http://127.0.0.1` 请求本地 `connect` 服务
- 如果服务没有启动，业务命令会直接报错
- SQLite 仍然使用 `data` 文件，并通过连接池复用数据库连接
- 当 `connect` 以子模块嵌入 `integration` 或 `proxy` 时，也会复用启动阶段初始化的同一份 `connectsvc.Service` 与数据库连接池，不会在每次 HTTP 请求时重新打开/关闭 SQLite

模块仍然支持作为子模块复用，底层核心逻辑保留在 `connectsvc` 包中。最终用户验收建议优先使用 `integration` 顶层入口，`connect help` 更适合内部实现联调与兼容排查。

## 编译

```bash
cd /path/to/deepright/cli/module/connect
/opt/homebrew/bin/go build -o connect ./
/opt/homebrew/bin/go build -o ../plugins/browser ./browser
/opt/homebrew/bin/go build -o ../plugins/feishu ./feishu
/opt/homebrew/bin/go build -o ../plugins/email ./email
```

## 启动与停止服务

启动本地服务：

```bash
./connect start --db ./data --agent-dir ../agent/test-case
```

前台调试模式：

```bash
./connect start --foreground true --db ./data --agent-dir ../agent/test-case
```

停止服务：

```bash
./connect stop
```

默认情况下，服务监听本地地址：

```text
http://127.0.0.1:18080
```

也可以显式指定地址或端口：

```bash
./connect start --addr 127.0.0.1:18081 --db ./data --agent-dir ../agent/test-case
./connect start --port 18081 --db ./data --agent-dir ../agent/test-case
```

### 启动后验证

服务启动后，可以用健康检查快速确认：

```bash
curl "http://127.0.0.1:18080/api/connect/health"
```

## 业务命令

在服务启动后，可以执行以下命令：

```bash
./connect meta-create --key "feishu" --meta "{\"token\":\"abc\"}" --stream true --callback "ignored-by-runtime" --agent A --chatId chat-001 --model OpenAI

./connect meta-update --key "feishu" --meta "{\"token\":\"xyz\"}" --stream false --callback "ignored-by-runtime" --agent A --chatId chat-002 --model OpenAI

./connect meta-delete --name "飞书"

./connect meta-get --key "feishu"
./connect meta-get --name "飞书"
./connect meta-list
./connect list-meta

./connect add-request --key "feishu" --externalId "msg-1" --content "HELLO WORLD"
./connect add-request --key "feishu" --content "HELLO WORLD" --status 1
./connect add-request --key "feishu" --content "HELLO WORLD" --artifacts "/tmp/a.txt,/tmp/b.txt" --original '{"text":"HELLO WORLD"}'
./connect add-request --key "feishu" --content "HELLO WORLD" --schema '{"type":"object","properties":{"reply":{"type":"string"}}}'
./connect request-list --name "飞书" --after-id 0 --limit 20
./connect request-list --name "飞书" --status 2 --limit 20

./connect add-response --name "飞书" --request-id 1 --response "HELLO BACK"
./connect add-response --name "飞书" --request-id 1 --response "HELLO BACK" --artifacts "/tmp/a.txt"
./connect response-list --name "飞书" --request-id 1 --after-id 0 --limit 20
```

如果服务不在默认地址，也可以对业务命令带上 `--addr` 或 `--port`：

```bash
./connect meta-list --port 18081
./connect add-request --addr http://127.0.0.1:18081 --key "feishu" --content "HELLO"
```

这些命令都会先访问本地 HTTP 服务，而不是直接打开 SQLite；如果服务没有启动，命令会直接返回错误。

## 已配置插件 Meta

`connect` 新增 `list-meta` 命令，用来返回当前所有已配置插件的运行时元数据视图。

```bash
./connect list-meta
./connect list-meta --addr http://127.0.0.1:18081
```

返回格式示例：

```json
[
  {
    "name": "飞书",
    "meta": {
      "appId": "cli-app",
      "appSecret": "cli-secret"
    },
    "stream": true,
    "callback": "./feishu",
    "agentId": "A",
    "chatId": "chat-001",
    "model": "OpenAI",
    "thinking": true,
    "createdAt": "2026-05-05T10:00:00+08:00",
    "updatedAt": "2026-05-05T10:00:00+08:00"
  }
]
```

说明：

- `meta-get --key feishu` 会直接返回单个插件的已配置运行时视图，适合插件按稳定主键读取配置
- `meta-get --name ...` 仍保留原始数据库记录视图，兼容已有调试和排查流程
- `list-meta` 只返回当前已配置的连接元数据
- 字段 `meta` 会从数据库中的 JSON 字符串解析为 JSON 对象，便于插件直接消费
- `meta-list` 仍然保留原始数据库记录视图，兼容已有调试和排查流程
- HTTP 接口也支持相同视图：`GET /api/connect/meta?view=config`

## 插件发现

`connect` 提供本地命令 `list-plugins`，用于扫描 `../plugins` 目录当前层的可执行文件，并组合每个插件的 `name`、`param`、`scope`、`command` 探测结果。

```bash
./connect list-plugins
./connect list-plugins --connect-cache 30000
```

返回格式示例：

```json
[
  {
    "key": "feishu",
    "name": "飞书",
    "param": [
      "appId",
      "appSecret"
    ],
    "scope": [
      "reuse",
      "agent",
      "provider",
      "thinking"
    ],
    "command": [
      "help",
      "name",
      "param",
      "scope",
      "command",
      "start",
      "stop"
    ]
  }
]
```

规则说明：

- 只扫描 `../plugins` 当前层文件，不扫描子目录
- 只读取二进制可执行文件，普通文件和目录会被跳过
- 每个插件会并发调用 `name`、`param`、`scope`、`command`、`help`，以缩短整体探测耗时
- `scope` 表示该插件支持哪些容器配置项，可选值为 `reuse`、`agent`、`provider`、`thinking`、`swarm`
- 如果插件没有实现 `scope` 命令，则默认回退为 `["reuse","agent","provider","thinking","swarm"]`
- 如果插件显式返回 `[]`，则表示完全不支持容器配置
- 结果会写入本地缓存文件，并按 `--connect-cache` 控制失效时间，默认 10000 毫秒
- `--connect-cache 0` 或负数时，不读取也不写入缓存
- `proxy` 与 `integration` 的 `GET /api/plugins/meta` 复用这套共享探测结果，再按插件名合并 `list-meta` 的已配置参数后对外返回

## Browser 插件

`browser` 插件遵循统一插件规范，可直接被 `connect list-plugins`、`integration plugins start|stop` 发现和调用。

```bash
./connect list-plugins
../plugins/browser name
../plugins/browser param
../plugins/browser start
../plugins/browser stop
```

说明：

- `../plugins/browser name` 固定返回 `{"key":"browser","name":"浏览器"}`
- `../plugins/browser param` 固定返回 `["headless","chrome"]`
- `../plugins/browser start` 和 `stop` 都会关闭 `browser_instance.json` 中记录的全部 CDP 服务
- `../plugins/browser start` 会继续启动同目录下 `browser.pid` 对应的后台 Playwright daemon
- `../plugins/browser stop` 会继续关闭同目录下 `browser.pid` 对应的后台 Playwright daemon
- `../plugins/browser` 的 `eval` 同时兼容 `eval '<js>'` 与 `eval --code '<js>'` 两种写法
- `integration` 判断 Browser 插件是否已启动时，读取的也是 `../plugins/browser.pid`
- Playwright daemon 生命周期如果需要独立调试，使用 `../plugins/browser daemon start|stop|serve`
- `browser.log`、`browser.pid`、`browser_instance.json` 位于 `../plugins/` 目录，与 `browser` 二进制同级
- `playwright/driver`、`obscura/release/...` 也需要放在 `../plugins/` 目录下供 `browser` 按相对路径加载

## Feishu 启动与停止

`feishu` 插件既可以通过顶层子命令启动，也可以直接运行独立二进制：

```bash
./connect feishu param
./connect feishu name

./connect feishu start --connect-bin ./connect
./connect feishu stop --pid-file ./feishu.pid

../plugins/feishu param
../plugins/feishu name

../plugins/feishu start --connect-bin ./connect
../plugins/feishu stop --pid-file ../plugins/feishu.pid
```

编译后的独立二进制默认放在 `../plugins/feishu`，因此从 `connect` 目录直接联调时，推荐优先使用这个路径。

`../plugins/feishu start --connect-bin ./connect` 具备幂等重启语义：如果当前 `feishu` 已经在运行，会先自动执行一次 `stop`，再重新 `start`。

在未启动 `../plugins/feishu start` 时，以下固定命令也始终可用：

```bash
../plugins/feishu param
# ["appId","appSecret"]

../plugins/feishu name
# {"key":"feishu","name":"飞书"}
```

如果 `connect` 服务不在默认地址，也可以传入：

```bash
../plugins/feishu start --connect-bin ./connect --addr http://127.0.0.1:18081
../plugins/feishu init --message '{"messageId":"om_xxx"}' --content "开始执行" --addr http://127.0.0.1:18081
```

`feishu` 只依赖 `connect` 服务中的稳定主键 `key=feishu` 对应元数据来工作，不需要也不会直接连接 SQLite 或访问 Agent 目录；为兼容旧数据，运行时仍会回退尝试 `FEISHU` 和 `飞书`。

`../plugins/feishu start` 默认会生成两类本地文件：

- `./feishu.log`：消息流水日志，每次收到消息且在调用 `connect add-request` 之前都会追加一行，格式固定为 `时间,内容`
- `./feishu.runtime.log`：运行诊断日志，用于排查建连、重连、推送失败等问题

## Feishu 主动发送

`feishu` 提供独立的 `send` / `init` / `command` 命令：
- `send`：回复飞书文本、图片、文件消息
- `init`：向飞书推送任务初始化消息，参数与处理方式与send相同，日志提示调用了init
- `command`：返回飞书插件的功能列表

飞书保存三方请求数据使用 `connect add-request` 命令，--key固定为feishu。

```bash
../plugins/feishu send --message '{"id":1,"original":"{\"schema\":\"2.0\",\"event\":{\"message\":{\"message_id\":\"om_xxx\",\"content\":\"{\\\"text\\\":\\\"你好\\\"}\",\"message_type\":\"text\"}}}"}' --content "收到"
../plugins/feishu init --message '{"id":1,"original":"{\"schema\":\"2.0\",\"event\":{\"message\":{\"message_id\":\"om_xxx\",\"content\":\"{\\\"text\\\":\\\"你好\\\"}\",\"message_type\":\"text\"}}}"}' --content "<开始执行>可通过新消息更新任务内容"
../plugins/feishu send --message '{"message":{"raw":"{\"schema\":\"2.0\",\"event\":{\"message\":{\"message_id\":\"om_xxx\",\"content\":\"{\\\"text\\\":\\\"你好\\\"}\",\"message_type\":\"text\"}}}"}}' --image /tmp/a.png,/tmp/b.jpg
../plugins/feishu send --message '{"messageId":"om_xxx"}' --content "附件如下" --image /tmp/a.png,/tmp/b.jpg --file /tmp/a.pdf
```

说明：

- `--message` 为必填，值应为 `connect add-request` 返回的请求 JSON；推荐使用其中的 `original` 字段，插件也兼容旧字段 `rawRequest`
- 飞书原消息 `messageId` 会从 `--message` 中自动提取
- `--image`、`--file` 都可以为空
- `send` / `init` 只支持回复已有消息
- 如果同时带文本和附件，插件会拆成多次飞书 API 调用分别发送
- 每次执行 `send` 或 `init` 时，都会在 `feishu.log` 追加一条与实际命令一致的调用记录，例如 `send name=feishu target=oc_xxx replyTo=om_xxx types=text` 或 `init name=feishu target=oc_xxx replyTo=om_xxx types=text`

## Email 启动与停止

`email` 插件用于定时扫描未读邮件，并通过 `connect add-request` 把邮件推入 Integration 代理。

```bash
./connect email param
./connect email name

./connect email start --connect-bin ./connect
./connect email stop --pid-file ./email.pid

../plugins/email param
../plugins/email name
../plugins/email start --connect-bin ./connect
../plugins/email stop --pid-file ../plugins/email.pid
```

固定输出：

```bash
../plugins/email param
# ["email","email_pop3","email_smtp","email_password","email_whitelist","email_pop3_interval"]

../plugins/email name
# {"key":"email","name":"邮件"}
```

运行所需 meta 配置：

```json
{
  "email": "demo@example.com",
  "email_pop3": "pop.example.com:995",
  "email_smtp": "smtp.example.com:465",
  "email_password": "secret",
  "email_whitelist": "alice@example.com,bob@example.com",
  "email_pop3_interval": "300",
  "mode": "email"
}
```

说明：

- `email` 插件不会直接访问 SQLite 或 Agent 目录
- 所有配置都通过 `connect` 或 `integration connect` 代理获取
- 默认每 300 秒扫描一次未读邮件
- 轮询秒数优先读取 `email_pop3_interval`，并兼容回退旧字段 `email_pop3_seconds`
- 发件人白名单使用 `email_whitelist` 控制；为空时默认不过滤
- 收到图片或文件后，会下载到 `email_artifacts` 并把绝对路径追加到 `artifacts`
- 邮件正文中的资源会被归一化为 `[image]绝对路径` 或 `[file]绝对路径`
- POP3 / SMTP 明细日志和失败信息会写入 `./email.log`
- `./email.log` 和 `./email.runtime.log` 都支持单文件 `10MB` 自动分卷，最多保留 `4` 个历史分卷
- 扫描时间线和已处理消息会持久化到 `./email.state.json`

## Email 主动发送

`email` 也支持独立的 `send` / `init` / `command` 命令：
- `send`：回复邮件正文、图片和文件附件
- `init`：向邮件推送任务初始化消息，参数与处理方式与send相同，日志提示调用了init
- `command`：返回邮件插件的功能列表

邮件保存三方请求数据使用 `connect add-request` 命令，--key固定为email。

```bash
../plugins/email send --message '{"message":{"raw":"{\"headers\":[{\"name\":\"From\",\"value\":\"Sender <sender@example.com>\"},{\"name\":\"Subject\",\"value\":\"原始主题\"},{\"name\":\"Message-ID\",\"value\":\"<origin@example.com>\"}],\"content\":\"hello\"}"}}' --content "收到"
../plugins/email send --message '{"message":{"raw":"{\"headers\":[{\"name\":\"From\",\"value\":\"Sender <sender@example.com>\"},{\"name\":\"Subject\",\"value\":\"原始主题\"},{\"name\":\"Message-ID\",\"value\":\"<origin@example.com>\"}],\"content\":\"hello\"}"}}' --content "附件如下" --image /tmp/a.png,/tmp/b.jpg --file /tmp/a.pdf
../plugins/email init --message '{"message":{"raw":"{\"headers\":[{\"name\":\"From\",\"value\":\"Sender <sender@example.com>\"},{\"name\":\"Subject\",\"value\":\"原始主题\"},{\"name\":\"Message-ID\",\"value\":\"<origin@example.com>\"}],\"content\":\"hello\"}"}}' --content "初始化完成"
```

说明：

- `--message` 为必填，值应为 `connect add-request` 返回的请求 JSON；推荐使用其中的 `original` 字段，插件也兼容旧字段 `rawRequest`
- `--content`、`--image`、`--file` 可以只传其中一种，也可以混合传
- `send` / `init` 会优先通过 `meta-get --key email` 读取 `email`、`email_smtp`、`email_password`
- 如果能从原始报文中解析到邮件头，会自动回复原发件人并带上 `In-Reply-To`
- `References` 会保留原始链路，并追加父消息和本次回复邮件的 `Message-ID`
- 每次执行 `send` 或 `init` 时，都会在 `email.log` 追加一条与实际命令一致的调用记录，例如 `send name=email to=sender@example.com replyTo=<origin@example.com> types=text` 或 `init name=email to=sender@example.com replyTo=<origin@example.com> types=text`

## 参数说明

### 服务参数

| 参数 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `--addr` | 否 | `127.0.0.1:18080` | 本地服务监听地址 |
| `--port` | 否 | `18080` | 本地服务端口简写 |
| `--db` | 否 | `../cron/data` 或 `./data` | SQLite 路径 |
| `--agent-dir` | 否 | `AGENT_DIR` 或 `./agent` | Agent 根目录 |
| `--connect-cache` | 否 | `10000` | 读缓存 TTL，单位毫秒 |
| `--pid-file` | 否 | `./connect.pid` | 服务 PID 文件 |
| `--log-file` | 否 | `./connect.log` | 服务日志文件 |
| `--foreground` | 否 | `false` | 是否前台运行 |

### `meta-create` / `meta-update`

| 参数 | `meta-create` | `meta-update` | 说明 |
|------|----------------|---------------|------|
| `--key` | 是 | 是 | 插件运行时主键；推荐主参数 |
| `--name` | 否 | 否 | 兼容旧输入；仅用于展示或排查 |
| `--meta` | 是 | 否 | JSON 字符串 |
| `--stream` | 否 | 否 | 是否支持流式响应 |
| `--callback` | 是 | 否 | 兼容占位参数；实际落库值始终固定解析为“应用启动目录/plugins/<plugin-key>” |
| `--agent` / `--agentId` | 是 | 否 | 绑定 AgentId |
| `--chat` / `--chatId` | 否 | 否 | 会话 ID |
| `--model` | 是 | 否 | 模型名称，必须已在 `token_store` 注册 |
| `--thinking` | 否 | 否 | 是否深度思考 |

### `help`

查看 connect 模块的完整命令使用手册。

```bash
./connect help
```

返回 connect 所有支持的命令及参数说明。

### `add-request`

| 参数 | 必填 | 说明 |
|------|------|------|
| `--key` / `--name` | 是 | 连接主键；推荐优先使用 `--key` |
| `--external-id` | 否 | 三方外部消息 ID；与 `key` 组成唯一键 |
| `--content` / `--request` | 是 | 请求内容；推荐优先使用 `--content` |
| `--artifacts` | 否 | 逗号分隔的附件路径 |
| `--original` / `--raw-request` | 否 | 原始请求内容；推荐优先使用 `--original` |
| `--status` | 否 | 请求状态：`0=待处理`、`1=已启动`、`2=已完成`、`3=已过期`、`4=已回复`；默认 `0` |
| `--created` | 否 | 创建时间，支持 Unix 时间戳或 RFC3339 |

### `add-response`

| 参数 | 必填 | 说明 |
|------|------|------|
| `--name` | 是 | 连接名称 |
| `--request-id` | 是 | 对应请求 ID |
| `--response` | 是 | 响应内容 |
| `--artifacts` | 否 | 逗号分隔的附件路径 |

### `request-list` / `response-list`

| 参数 | 必填 | 说明 |
|------|------|------|
| `--name` | 否 | 按连接名称过滤 |
| `--request-id` | 否，仅 `response-list` 支持 | 按请求 ID 过滤 |
| `--status` | 否，仅 `request-list` 支持 | 按请求状态过滤：`0/1/2/3` |
| `--after-id` | 否 | 只返回指定 ID 之后的数据 |
| `--before-id` | 否，仅 `request-list` 支持 | 只返回指定 ID 之前的数据；默认结果按 ID 倒序，分页读取更早记录时使用上一页最小 ID |
| `--limit` | 否 | 返回条数，默认 100 |

## HTTP 接口

本地服务当前暴露以下接口：

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/connect/health` | 健康检查 |
| POST / PUT / DELETE / GET | `/api/connect/meta` | 元数据创建、更新、删除、查询 |
| POST / GET | `/api/connect/request` | 请求创建、查询 |
| POST / GET | `/api/connect/response` | 响应创建、查询 |

## 校验规则

- `name` 必须存在且未删除，才能继续写入请求和响应
- `meta` 必须是合法 JSON
- `agentId` 必须在 `--agent-dir` 指向的 Agent 根目录下存在
- `model` 必须已经在共享数据库的 `token_store` 里注册，且 token 不能为空
- `artifacts` 会被解析成逗号分隔字符串；本地路径必须存在
- 如果传入 `externalId`，则同一个 `name + externalId` 不能重复写入
- 请求记录包含 `status` 字段，枚举固定为 `0=待处理`、`1=已启动`、`2=已完成`、`3=已过期`、`4=已回复`
- `add-request` 未传 `status` 时默认写入 `0`
- `add-request` 未传 `created` 时默认写入当前时间
- `add-request` 支持新可选参数 `--schema`，值为 Json String
- 该参数会先保存到 `connect_request.response_schema`
- 后续被代理为一次性 cron 任务时，会继续透传到任务明细的 `task_detail.response_schema`
- `request-list` 支持通过 `--status` 或 `GET /api/connect/request?status=` 按状态过滤
- `add-response` 的 `request-id` 必须存在，并且属于同一个连接名称
- `add-response` 成功后，会自动把对应请求更新为 `status=4`
- 如果服务没有启动，CLI 会提示先执行 `./connect start`

## Feishu 元数据格式

`name=feishu` 时，推荐把 `meta` 设置为以下 JSON：

```json
{
  "appId": "cli_xxx",
  "appSecret": "xxx",
  "mode": "feishu"
}
```

## Feishu 消息处理规则

- feishu 收到飞书事件后，会从原始消息 JSON 中提取文本内容
- 例如 `content` 为 `"{\"text\":\"你好\"}"` 时，最终提取出的消息内容为 `你好`
- 图片消息会提取 `image_key`，调用飞书下载接口拉取内容并落到本地 `feishu_artifacts` 目录，再把本地文件路径作为 `artifacts` 推送给 `connect`
- 文件消息会提取 `file_key`，调用飞书下载接口拉取内容并落到本地 `feishu_artifacts` 目录，再把本地文件路径作为 `artifacts` 推送给 `connect`
- feishu 推送到 `connect` 时：
  - `externalId` 使用 `create_time + content` 的 MD5，作为唯一键
  - `request` 使用提取后的文本内容
  - `rawRequest` 保留原始飞书事件 JSON
- feishu 启动后会基于飞书长连接的真实连接/心跳信号做健康检查；默认每 60 秒检查一次，如果超过超时时间仍未收到心跳或消息，会主动断开并重连
- 如果消息创建时间距离当前时间超过 30 分钟，会记录 `feishu.log` 并跳过，不再推送到 `connect`
- 如果飞书报文无法解析出有效消息，也会记录 `feishu.log` 并跳过
- 正常消息会先写一条 `时间 + 内容` 的日志，再调用 `connect add-request`

测试或联调时可以使用 mock 模式：

```json
{
  "mode": "mock",
  "heartbeatIntervalSec": 1,
  "heartbeatTimeoutSec": 1,
  "reconnectDelayMs": 50,
  "mockHeartbeatMs": 100,
  "mockDisconnectAfterMs": 300,
  "mockMessages": [
    {
      "delayMs": 50,
      "messageId": "msg-1",
      "chatId": "chat-1",
      "content": "hello from mock"
    }
  ]
}
```

## 推荐链路

1. 启动 `connect` 服务
2. 使用 `meta-create --key <plugin-key>` 创建连接元数据
3. 三方模块启动后，通过 `meta-get` 获取自己的配置
4. 三方模块收到消息后，通过 `add-request` 推送请求
5. 业务处理完成后，通过 `add-response` 写回响应

一个最小示例：

```bash
./connect start --db ./data --agent-dir ../agent/test-case

./connect meta-create \
  --key feishu \
  --meta '{"mode":"mock"}' \
  --stream true \
  --callback ignored \
  --agent A \
  --model OpenAI

./connect add-request --key feishu --content "HELLO WORLD"
./connect request-list --name feishu --status 0 --limit 10
./connect add-response --name feishu --request-id 1 --response "HELLO BACK"
./connect request-list --name feishu --status 2 --limit 10
./connect response-list --name feishu --request-id 1 --limit 10
./connect list-plugins
```

## 作为子模块调用

```go
package main

import (
    "fmt"
    "time"

    "connect/connectsvc"
)

func main() {
    svc, err := connectsvc.NewService(connectsvc.Options{
        DBPath:   "/path/to/data",
        AgentDir: "/path/to/agent",
        CacheTTL: 10 * time.Second,
    })
    if err != nil {
        panic(err)
    }
    defer svc.Close()

    meta, err := svc.CreateMeta(connectsvc.MetaInput{
        Name:     "feishu",
        Meta:     `{"mode":"mock"}`,
        Stream:   true,
        Callback: "../plugins/feishu",
        AgentID:  "A",
        Model:    "OpenAI",
    })
    if err != nil {
        panic(err)
    }
    fmt.Println(meta.Name)
}
```

---

## 迭代 20260716-1：插件开关与飞书收件确认

右上角插件开关统一控制关联插件的启停与配置，并支持 Remote 超时设置。飞书消息在可靠写入任务队列后会回复“已收到”，表示任务已保存、可继续用新消息更新内容；这不表示任务已经开始或已经完成，后续状态和结果仍按原流程发送。

---

## 迭代 20260716-1：插件开关与飞书入库回执

右上角插件扇形开关用于统一启停关联插件及管理其配置，并支持 Remote 超时配置。

飞书消息在成功可靠入库后，会在原会话收到一次“已收到”提示。该提示仅表示任务已经入库，任务开始、执行结果和最终回复仍按原有流程发送；重复投递、重连或重启不会重复发送此提示。

---

## 迭代 20260718-1：插件帮助精简

飞书、邮件、Browser 等插件的 `--help` 只展示面向任务处理的公开使用方式，不再介绍 `start`、`stop`、`init` 三个生命周期或初始化命令。Browser 的 `daemon --help` 与 `instance --help` 也遵循相同规则。

这不改变任何命令能力：Integration 仍可通过插件生命周期接口管理插件，已有脚本和内部调用仍可执行上述命令；插件的 `command` JSON 能力列表也保持不变。
