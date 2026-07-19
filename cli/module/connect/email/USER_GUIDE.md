# Email Plugin User Guide

## 简介

`email` 是 `connect` 体系下的邮件插件，运行时主键固定为 `email`，展示名为 `邮件`。它既可以独立通过 CLI 启动，也可以被 `integration plugins start --name email` 拉起。

当前插件提供固定元信息命令、邮件接收能力和邮件发送能力：

- `command` 固定返回 `["command","help","name","param","scope","schema","sender","search","init","send","start","stop"]`
- `schema` 固定返回邮件插件响应 JSON Schema
- `param` 固定返回带字段说明的示例对象数组，字段 key 固定为 `email`、`email_pop3`、`email_smtp`、`email_password`、`email_whitelist`、`email_pop3_interval`
- `name` 固定返回 `{"key":"email","name":"邮件"}`
- `send` 可根据原始 `add-request` 报文回复邮件，也可指定收件人与主题发送新邮件，并附带图片/文件附件
- `init` 使用与 `send` 完全相同的参数和处理流程发送初始化消息
- `sender` 返回配置时间窗口内、按最后发送时间去重的发件人邮箱
- `search` 搜索配置时间窗口内归一化后的邮件主题与正文
- `start` / `stop` 负责启动或停止邮件扫描进程

邮件插件启动后，每个扫描周期都会重新建立一条新的 POP3 连接，完成认证、拉取 UIDL/邮件内容后立即主动断开，不再把上一轮轮询留下来的会话复用到下一轮。扫描期间仍会按周期读取 `connect` 中稳定主键 `key=email` 对应的元数据配置，通过 POP3 拉取邮件，并把符合准入规则的消息通过 `connect add-request` 推入 Integration 代理。为兼容旧环境，读取配置时也会回退尝试旧 key `email_smtp`。如果当前轮连接在 `UIDL`、`RETR` 等扫描步骤中被服务端主动断开并返回 `EOF`，插件会把它当成可恢复断链，在当前轮内自动重连并从中断位置继续；只有邮件真正被读取进入后续处理后，才会推进 `email.state.json` 的时间线。若个别邮件 MIME 结构异常导致标准解析失败，插件会退化为原始报文兜底读取，避免单封坏邮件拖垮整轮扫描。若 `email_pop3` 域名默认解析结果不可用、TLS 握手超时或单线路由抖动导致建连失败，插件会先按 10 秒超时中止当前建连，再自动切换到备用 DNS 解析结果继续建连，并在下一轮扫描继续自动重试，无需人工重启。若插件配置未启用“复用当前会话”，启动阶段会自动把空的 `chatId` 校正为固定值 `email`；若页面已经通过“复用当前会话”保存了当前 `chatId`，插件则直接沿用该 `chatId`。整个过程中 `agentId` 都保持页面选择值，不会因为复用当前会话而自动切换或锁定。

## 需求目录

- 主需求文档：[REQUIREMENT.md](REQUIREMENT.md)
- POP3 长连接与网络容错迭代需求：[iteration/20260520-1/REQUIREMENT.md](iteration/20260520-1/REQUIREMENT.md)
- 日志增强迭代需求：[iteration/20260521-1/REQUIREMENT.md](iteration/20260521-1/REQUIREMENT.md)
- 收件立即推送迭代需求：[iteration/20260521-2/REQUIREMENT.md](iteration/20260521-2/REQUIREMENT.md)
- MIME 主题解码迭代需求：[iteration/20260521-3/REQUIREMENT.md](iteration/20260521-3/REQUIREMENT.md)
- 发送日志增强迭代需求：[iteration/20260522-1/REQUIREMENT.md](iteration/20260522-1/REQUIREMENT.md)
- 会话复用迭代需求：[iteration/20260612-1/REQUIREMENT.md](iteration/20260612-1/REQUIREMENT.md)
- 自定义收件人与主题迭代需求：[iteration/20260718-1/REQUIREMENT.md](iteration/20260718-1/REQUIREMENT.md)
- 自定义收件人与主题迭代手册：[iteration/20260718-1/USER_GUIDE.md](iteration/20260718-1/USER_GUIDE.md)
- 邮件快照查询迭代需求：[iteration/20260719-1/REQUIREMENT.md](iteration/20260719-1/REQUIREMENT.md)
- 邮件快照查询迭代手册：[iteration/20260719-1/USER_GUIDE.md](iteration/20260719-1/USER_GUIDE.md)
- 当前用户手册：[USER_GUIDE.md](USER_GUIDE.md)

`20260520-1` 迭代补齐了 POP3 建连与网络容错约束：

- `start` 后默认只保持一条 POP3 长连接，轮询优先复用
- `EOF`、默认 DNS 解析异常、TLS 握手超时或单线路由抖动都会触发自动重连或备用解析回退
- 单次扫描失败不会阻塞后续轮询；网络恢复后会自动继续收信
- 只有邮件成功读取并进入后续处理后，才允许推进 `email.state.json`

`20260521-1` 迭代的目标只有两项：

- `email.log` 里始终输出已解码后的主题，避免 MIME 编码主题直接落日志
- `email.log` 里补充结构化字段，至少包含 `subject=...`、`from=...`、`message_id=...`

`20260521-2` 迭代补充了一条接收侧约束：

- POP3 当前轮扫描一旦拿到准入邮件，就在本轮内立刻调用 `add-request`
- 不再为了等待额外邮件而增加插件内部延迟

`20260521-3` 迭代补齐了 RFC 2047 / MIME encoded-word 解码范围：

- 任务明细正文中的主题和正文组合使用解码后的可读文本
- `raw_request.message.subject` / `raw_request.message.content` 使用解码后的可读文本
- `push-request` 等排障日志中展示的 `--content`、`subject`、`content` 也会先解码再输出

`20260522-1` 迭代把 `send` / `init` 的发送日志补齐为完整链路：

- `stage=send-request` 记录原始请求报文和附件参数
- `stage=send-parse` 记录回复上下文或新邮件模式解析出的目标邮箱、主题及原始 envelope
- `stage=send-result` 记录最终发送结果，以及实际 SMTP 顶层报文头
- `stage=send-failed` 记录明确失败原因；非法 `--message`、无效 `rawRequest`、缺少回复字段或新邮件模式缺少 `--to` / `--subject` 都会输出可直接定位的问题描述

`20260612-1` 迭代补齐了插件页“复用当前会话”的会话语义：

- 复用当前会话时，启动后的邮件插件只沿用页面保存的当前 `chatId`
- 未复用当前会话时，邮件插件会在启动阶段把空 `chatId` 自动固定为 `email`
- `agentId` 始终使用页面当前选择值，不会因为复用当前会话而自动绑定

## 编译

在 `/path/to/deepright/cli/module/connect` 目录执行：

```bash
/opt/homebrew/bin/go build -o ../plugins/email ./email
```

## 固定输出命令

```bash
../plugins/email command
../plugins/email schema
../plugins/email param
../plugins/email scope
../plugins/email name
../plugins/email sender --connect-bin ./integration
../plugins/email search --query "退款 已处理" --limit 20 --offset 0 --connect-bin ./integration
```

返回示例：

```json
["command","help","name","param","scope","schema","sender","search","init","send","start","stop"]
```

```json
{
  "type": "object",
  "properties": {
    "content": {
      "type": "string",
      "description": "The content is presented in plain text, and the file paths have been replaced with filenames."
    },
    "artifacts": {
      "type": "array",
      "description": "All file paths found in the response.",
      "items": {
        "type": "object",
        "properties": {
          "path": {
            "type": "string",
            "description": "Absolute file path."
          },
          "desc": {
            "type": "string",
            "description": "Purpose of this file."
          }
        },
        "required": [
          "path",
          "desc"
        ]
      }
    },
    "why_do_this": {
      "type": "string",
      "description": "Brief reasoning for executing this command. Must retain granular data and operational logs for process summarization. The more granular, the better."
    }
  },
  "required": [
    "content"
  ]
}
```

```json
[{
  "email": "邮箱地址，如hello_world@gmail.com",
  "email_pop3": "邮箱的pop3地址，如pop.gmail.com",
  "email_smtp": "邮箱的smtp地址，如smtp.gmail.com",
  "email_password": "邮箱的密码",
  "email_whitelist": "以逗号分隔的收件人白名单，如a@gmail.com,b@gmail.com。",
  "email_pop3_interval": "每次扫描待处理邮件的间隔秒数，默认300"
}]
```

```json
{"key":"email","name":"邮件"}
```

说明：

- `../plugins/email schema`、`send` 对结构化 `--content` 的校验、以及邮件插件调用 `integration connect add-request --schema` 时透传的值，三者复用同一份共享 schema 定义
- `schema` 中的 `content` 描述以“纯文本正文”作为对外约定；发送阶段如果传入的是 HTML，插件仍会自动生成 HTML 正文并尽量保留结构
- `param` 返回的是字段说明示例，真正写入 `meta-create` / `meta-update` 的仍然是同名字段对应的真实值

## 邮件快照查询

`sender` 和 `search` 只查询 Integration / Connect 的本地通用消息快照：邮件插件以 `source=email` 写入快照，Integration 不解析邮件报文或依赖邮件插件。命令不会连接 POP3/SMTP，也不会读取 `email.log`。

查询时间窗口来自 Integration 运行目录的 `config/config.json`：

```json
{
  "email": {
    "lastMessage": 72
  }
}
```

`email.lastMessage` 必须是大于 0 的整数，单位为小时。缺少配置文件、JSON 非法、字段缺失或值无效时，命令立即失败，不会默认使用 72 小时。

查询最近窗口内每个发件人的最后一次邮件：

```bash
../plugins/email sender --connect-bin ./integration
```

```json
[
  {
    "sender": "alice@example.com",
    "lastMessageAt": "2026-07-19T10:30:00Z"
  }
]
```

`From` 会被解析成第一个有效邮箱地址、去除显示名并转小写。结果中 `sender` 唯一，按 `lastMessageAt` 从新到旧排序。邮件时间优先使用 `Date` 头；该头无效时使用插件接收时间。

搜索归一化后的主题和正文：

```bash
../plugins/email search --query "退款 已处理" --limit 20 --offset 0 --connect-bin ./integration
../plugins/email search --query '"退款申请" 已处理' --connect-bin ./integration
../plugins/email search --limit 20 --offset 0 --connect-bin ./integration
../plugins/email search --sender alice@example.com --limit 20 --connect-bin ./integration
```

空白分隔的关键词采用 AND；双引号中的连续内容为一个短语。搜索不区分大小写，采用包含匹配；不会搜索附件内容或附件路径。省略或传入空 `--query` 时列出窗口内全部文本邮件；`--sender` 先转小写后作精确过滤，可与关键词取 AND。`--limit` 默认 50、最大 200，`--offset` 默认 0。

```json
{
  "total": 1,
  "limit": 20,
  "offset": 0,
  "items": [
    {
      "messageId": "<mail@example.com>",
      "sender": "alice@example.com",
      "content": "退款申请\n已处理",
      "sentAt": "2026-07-19T10:30:00Z"
    }
  ]
}
```

## 配置方式

`email` 插件的运行参数来自 `connect` / `integration` 中 `key=email` 的 meta 配置：

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

- `email` 为邮箱登录账号
- `email_pop3` 为 POP3 地址，推荐显式写成 `host:995`
- `email_smtp` 为 SMTP 地址
- `email_password` 为邮箱授权码或登录密码
- `email_whitelist` 为以 `,` 分隔的发件人白名单；留空表示不过滤
- `email_pop3_interval` 为当前推荐的 POP3 轮询秒数配置；留空时默认 `300`
- 为兼容旧配置，运行时也会回退读取 `email_pop3_seconds`；读取顺序为 `email_pop3_interval` -> `email_pop3_seconds` -> 默认 `300`
- POP3 建连会优先使用系统 DNS 解析 `email_pop3`；若默认解析不可用或 TLS 握手超时，插件会自动切换到备用解析结果继续建连
- 插件页未勾选“复用当前会话”时，可单独选择 `agentId`，而 `chatId` 会在首次启动时自动补成固定值 `email`
- 插件页勾选“复用当前会话”并保存后，插件启动时只会保留当前页面传入的 `chatId`
- 不论是否复用当前会话，`agentId` 都按页面选择值保存和使用

通常通过 Integration 顶层命令注册：

```bash
./integration connect meta-create \
  --name email \
  --meta '{"email":"'$EMAIL'","email_pop3":"'$EMAIL_POP3'","email_smtp":"'$EMAIL_SMTP'","email_password":"'$EMAIL_PASSWORD'","email_whitelist":"'$EMAIL_WHITELIST'","email_pop3_interval":"300","mode":"email"}' \
  --stream true \
  --callback ./email \
  --agent a \
  --model deepseek
```

也可以按需求文档里的验证方式，直接从系统环境变量创建：

```bash
./integration connect meta-create \
  --name email \
  --meta '{"email":"'$EMAIL'","email_pop3":"'$EMAIL_POP3'","email_smtp":"'$EMAIL_SMTP'","email_password":"'$EMAIL_PASSWORD'","email_whitelist":"","email_pop3_interval":"300"}'
```

这里 `--meta` 的字段集合需要与 `../plugins/email param` 返回对象中的 key 保持一致。

## 启动与停止

通过 Integration 拉起：

```bash
./integration plugins start --name email
```

直接执行插件：

```bash
../plugins/email start --connect-bin ../integration/integration
../plugins/email stop --pid-file ../plugins/email.pid
```

单次发送回复：

```bash
../plugins/email send \
  --message '{"id":1,"rawRequest":"{\"source\":\"email\",\"receivedAt\":\"2026-05-22T00:00:00+08:00\",\"message\":{\"uid\":\"uid-1\",\"messageId\":\"<origin@example.com>\",\"subject\":\"原始主题\",\"from\":\"Sender <sender@example.com>\",\"content\":\"hello\",\"artifacts\":[],\"raw\":\"{\\\"headers\\\":[{\\\"name\\\":\\\"From\\\",\\\"value\\\":\\\"Sender <sender@example.com>\\\"},{\\\"name\\\":\\\"Subject\\\",\\\"value\\\":\\\"原始主题\\\"},{\\\"name\\\":\\\"Message-ID\\\",\\\"value\\\":\\\"<origin@example.com>\\\"}],\\\"content\\\":\\\"hello\\\"}\"}}"}' \
  --content "收到，附件如下" \
  --image /tmp/a.png,/tmp/b.jpg \
  --file /tmp/a.pdf,/tmp/b.txt
```

发送新邮件（不传 `--message`）：

```bash
../plugins/email send \
  --to 'Alice <alice@example.com>,bob@example.com' \
  --subject '任务完成通知' \
  --content '任务已完成，请查收附件。' \
  --file /tmp/report.pdf
```

最小示例：

```bash
../plugins/email send \
  --message '{"id":1,"rawRequest":"{\"source\":\"email\",\"receivedAt\":\"2026-05-22T00:00:00+08:00\",\"message\":{\"uid\":\"uid-1\",\"messageId\":\"<origin@example.com>\",\"subject\":\"原始主题\",\"from\":\"Sender <sender@example.com>\",\"content\":\"hello\",\"artifacts\":[],\"raw\":\"{\\\"headers\\\":[{\\\"name\\\":\\\"From\\\",\\\"value\\\":\\\"Sender <sender@example.com>\\\"},{\\\"name\\\":\\\"Subject\\\",\\\"value\\\":\\\"原始主题\\\"},{\\\"name\\\":\\\"Message-ID\\\",\\\"value\\\":\\\"<origin@example.com>\\\"}],\\\"content\\\":\\\"hello\\\"}\"}}"}' \
  --content "收到"
```

初始化消息示例：

```bash
../plugins/email init \
  --message '{"id":1,"rawRequest":"{\"source\":\"email\",\"receivedAt\":\"2026-05-22T00:00:00+08:00\",\"message\":{\"uid\":\"uid-1\",\"messageId\":\"<origin@example.com>\",\"subject\":\"原始主题\",\"from\":\"Sender <sender@example.com>\",\"content\":\"hello\",\"artifacts\":[],\"raw\":\"{\\\"headers\\\":[{\\\"name\\\":\\\"From\\\",\\\"value\\\":\\\"Sender <sender@example.com>\\\"},{\\\"name\\\":\\\"Subject\\\",\\\"value\\\":\\\"原始主题\\\"},{\\\"name\\\":\\\"Message-ID\\\",\\\"value\\\":\\\"<origin@example.com>\\\"}],\\\"content\\\":\\\"hello\\\"}\"}}"}' \
  --content "初始化完成"
```

常见参数：

- `--connect-bin`：`connect` 或 `integration` 二进制路径
- `--addr` / `--port`：`connect` 服务地址
- `--scan-seconds`：扫描周期，默认 `300`
- `--log-file`：消息日志，默认 `./email.log`
- `--pid-file`：PID 文件，默认 `./email.pid`
- `--message`：`send` / `init` 可选；用于承载 `add-request` 返回的请求 JSON 和其中的 `rawRequest`，不作为邮件正文
- `--to`：未提供 `rawRequest` 时必填，以 `,` 分隔多个收件人
- `--subject`：未提供 `rawRequest` 时必填，作为新邮件主题
- `--content`：消息正文内容；普通文本会自动生成 HTML 正文，如果内容本身看起来像 HTML，则会保留 HTML 并同时生成纯文本副本
- `--content`：如果传入的是一个 JSON Object，或者整段文本里包含一段可提取的 JSON Object，并且结构符合 `email schema` 返回的 Schema，则会先解析再发送
- `email schema` 中的 `content` 字段仍按纯文本正文对外声明；发送实现同时兼容普通文本和 HTML
- `--image`：以 `,` 分隔的图片附件路径
- `--file`：以 `,` 分隔的文件附件路径

`start` 具备 restart 语义：若检测到已有 `email.pid` 且进程仍在运行，会先自动执行一次 `stop` 再重新启动。新进程启动后会建立一条长驻 POP3 会话，除非连接失效或配置变更，否则不会在每次扫描时重复登录。POP3 建连、TLS 握手、`NOOP` / `UIDL` / `RETR` 等命令读写都按明确超时执行；若默认解析结果不可用，插件会自动回退到备用解析结果并在后续轮询持续重试。

`send` 和 `init` 使用与 `start` 相同的参数获取方式，会优先通过 `connect meta-get --key email` 读取：

- `email`：发件人地址
- `email_smtp`：SMTP 服务器地址，支持 `host` 或 `host:port`
- `email_password`：SMTP 登录密码或授权码

如果当前环境还保留旧 key `email_smtp`，插件也会自动回退兼容。

## 发送行为

`send` 和 `init` 都会把 `--content` 作为正文，把 `--image` 和 `--file` 作为邮件附件发送。

如果 `--content` 直接传入的是符合 `email schema` 的 JSON Object，或者报文里是“说明文字 + JSON Object”的混合文本，插件都会先尝试提取其中第一段合法 JSON Object，再做一次归一化：

```json
{
  "content": "真正发送的邮件正文",
  "artifacts": [
    {
      "path": "/abs/path/a.png",
      "desc": "截图"
    },
    {
      "path": "/abs/path/b.pdf",
      "desc": "报告"
    }
  ],
  "why_do_this": "过程说明"
}
```

归一化规则：

- `content` 会替换原始 `--content`，作为真正发送的邮件正文
- `artifacts` 中每个 `path` 都必须是绝对路径
- 图片会自动并入 `--image`
- 非图片文件会自动并入 `--file`
- 如果报文前后夹带了解释性文本，插件会先从整段文本中提取第一段合法 JSON Object，再按同一份 schema 继续处理
- 会继续扫描正文里的远程图片链接、`file:///abs/path` 图片链接，以及 HTML `<img src="...">` 中的图片来源，并自动补充到图片附件
- POP3 轮询间隔会优先读取 meta 中的 `email_pop3_interval`；如果该字段为空，再回退读取 `email_pop3_seconds`
- `start` 建立的 POP3 连接会在轮询之间持续复用；只有探活失败、连接中断或 `email` / `email_pop3` / `email_password` 等关键配置变化时才会重连
- 如果 `content` 的 HTML 中存在可解析到本地图片文件的 `<img src="...">`，插件会自动改写为 `cid:` 引用，并把对应图片以内嵌资源发送
- 被改写为 `cid:` 的图片仍会作为邮件资源一起发送，因此收件端可以直接在正文中显示图片
- 解析过程会输出 `content`、`image`、`file` 标记日志，方便追踪实际发送内容
- 如果整段报文里找不到可提取、且符合 `email schema` 的 JSON Object，则完全按原逻辑处理，不做拆解
- 如果 `--content` 是 JSON Object 但不符合 `email schema`、解析 JSON 失败、引用图片下载失败或附件处理失败，则会优先回退到其中的 `content` 字段作为正文；若取不到 `content`，再回退为整个 `--content` 原文，且不处理图片/文件附件
- 如果上述降级发送仍失败，则会继续兜底发送 `<消息异常>请登录客户端查看`
- `send` / `init` 真正进入 SMTP 发送后，单次失败最多自动重试 3 次；超过 3 次会直接终止本次发送，不再继续切换其他发送兜底
- 对应日志会写入 `email.log`：重试阶段为 `stage=send-retry`，超过上限终止为 `stage=send-terminate`

- `rawRequest` 存在时始终进入回复模式，CLI 中的 `--to`、`--subject` 会被忽略
- 回复目标邮箱只从 `rawRequest.message.from` 识别
- `In-Reply-To` 使用 `rawRequest.message.messageId`
- `References` 会保留原有 `References` / `In-Reply-To`，并追加父消息 `Message-ID`
- `Subject` 只从 `rawRequest.message.subject` 识别，并自动补齐为 `Re: 原主题`
- 未提供 `rawRequest` 时进入新邮件模式，必须同时指定非空 `--to` 与 `--subject`；此模式不生成 `In-Reply-To` 和 `References`
- `--image` 指定的图片会作为普通附件发送
- 如果 HTML 正文中的 `<img>` 命中本地图片路径，则对应图片会以内嵌 `CID (Content-ID)` 资源发送；其他文件仍作为普通附件发送

`--message` 的校验规则：

- 未传 `--message` 时可直接使用 `--to` 和 `--subject` 发送新邮件
- 显式传入 `--message` 时必须是合法 JSON；JSON 非法会立即失败，即使 `--to`、`--subject` 已完整传入
- `--message` 为合法 JSON 但未包含 `rawRequest` 时可发送新邮件，仍必须同时传入 `--to` 与 `--subject`
- `rawRequest` 字段一旦存在，必须是非空 JSON 字符串；`null`、空字符串、纯空白、非字符串、JSON 非法或缺少字段都会立即失败，不会回退为新邮件
- 回复用 `rawRequest` 必须有非空的 `message.messageId`、`message.from`、`message.subject`

## 消息映射

邮件插件会把收到的邮件映射为 `connect add-request`：

- `create_time`：使用邮件消息头 `Date`
- `content`：邮件标题、正文内容、附件归一化资源行按顺序拼接
- `original`：完整的插件推送 JSON，其中 `message.subject`、`message.content` 使用解码后的可读文本，`message.raw` 为归一化后的 `{"headers":[{}],"content":""}` 结构
- `artifacts`：下载后的本地绝对路径，使用逗号拼接
- `key`：固定写入 `email`
- `schema`：复用 `email schema` 返回的同一份 Response JSON Schema，并透传给 `integration connect add-request --schema`
- `schema` 的具体内容与 `../plugins/email schema` 命令输出完全一致，不额外维护第二份定义

准入和去重规则：

- `email_whitelist` 非空时，发件人必须命中白名单，或等于 `email` 自身地址
- `email_whitelist` 为空时，默认不过滤发件人
- 首次启动时，会以“当前启动时间前 30 分钟”作为初始时间线
- 每次扫描都会持久化最后处理时间线；即使邮件不准入，也会推进时间线
- 重启后会继续复用已持久化的时间线，不会重新从全量邮件开始扫描
- 对准入邮件会按 `Message-ID` 持久化去重，避免重复推送

附件、内嵌图片和附件邮件中的资源会下载到启动目录下的 `email_artifacts`：

- 图片：`[image]绝对路径`
- 文件：`[file]绝对路径`

文件命名规则：

- 图片使用 `image_key_*`
- 文件使用 `file_key_*`

下载失败时只记日志，不会让 `email` 进程崩溃。

## 日志与状态

默认会写三类文件：

- `./email.log`：主日志文件，包含 POP3/SMTP 明细、失败信息、发送记录和启动连接记录
- `./email.log.1` 到 `./email.log.4`：按大小自动分卷的历史日志
- `./email.runtime.log`：运行诊断日志副本，同样按大小分卷
- `./email.state.json`：时间线和已处理 `Message-ID` 状态

`email.log` 与 `email.runtime.log` 都会在单文件达到 10MB 后自动轮转，最多保留 4 个历史分卷。
POP3 / SMTP 的明细日志和失败信息都会落到 `email.log`，同时复制到 `email.runtime.log` 便于排查。
每次启动时，`email.log` 与 `email.runtime.log` 都会追加实际调用 Integration 代理时使用的 `connect meta-get` 命令参数。
每次执行 `send` 或 `init` 时，`email.log` 会按阶段追加发送日志，例如：

```text
2026-05-22T10:00:00+08:00,stage=send-request action="send" message="{\"id\":1,...}" content="执行完成"
2026-05-22T10:00:00+08:00,stage=send-parse action="send" reply_to="<origin@example.com>" to="sender@example.com" subject="Re: 原始主题" raw_request="{\"source\":\"email\",...}"
2026-05-22T10:00:00+08:00,stage=send-result action="send" name="email" to="sender@example.com" reply_to="<origin@example.com>" types="text+image+file" smtp_header="From: demo@example.com\r\nTo: sender@example.com\r\nSubject: =?utf-8?... \r\n..." result="{\"name\":\"email\",...}"
2026-05-22T10:01:00+08:00,stage=send-failed action="init" step="parse-message" err="message.rawRequest.message.messageId not found"
```

其中：

- `stage=send-request` 记录原始请求报文和发送参数
- `stage=send-parse` 记录回复上下文或新邮件模式确定的目标邮箱、主题；回复模式额外记录回复锚点
- `stage=send-result` 记录最终发送结果，以及实际报送的顶层邮件报文头 `smtp_header`
- `stage=send-failed` 记录失败阶段、失败原因；如果已经组装出待发送邮件，也会附带 `smtp_header`
- `stage=send-retry` 记录 SMTP 发送失败后的第 N 次自动重试
- `stage=send-terminate` 记录 SMTP 连续失败超过 3 次后，本次发送被自动终止
收到邮件并准备推送到 `connect add-request` 时，`email.log` 会输出结构化收件日志，主题始终使用已解码值，例如 `stage=mail-received subject="今天天气" from="Sender <sender@example.com>" message_id="<abc@example.com>" summary="今天天气\\n正文"`。
白名单拒绝或命中重复 `Message-ID` 时，也会写同样风格的结构化日志，例如 `stage=mail-skip subject="今天天气" from="deny@example.com" message_id="<abc@example.com>" reason="sender not in whitelist"`。
如果 POP3 建连、TLS 握手或首包读取超时，日志会记录类似 `stage=scan err=context deadline exceeded` 的结构化错误；插件不会静默卡死，而是继续按轮询周期自动重试。
`stage=push-request` 这类失败日志中，如果 `--content`、`original`、`subject` 或 `message.content` 包含 RFC 2047 / MIME encoded-word，日志展示值会先解码成可读文本后再输出。
邮件插件长驻进程在 `load-config`、`scan`、推送 `add-request` 等链路如果连续失败，也只会自动重试 3 次；超过后会写入 `stage=service-retry` / `stage=service-terminate` 并退出进程。
