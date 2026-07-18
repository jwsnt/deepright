# Feishu Plugin User Guide

## 简介

`feishu` 是 `connect` 体系下的飞书插件，既能作为独立 CLI 运行，也能被 `connect` / `integration` 顶层命令透传调用。

当前插件提供两类能力：

- `start` / `stop`：启动或停止飞书长连接接收服务，把收到的消息推入 `connect add-request`
- `send` / `init`：基于 `connect add-request` 的原始报文回复飞书文本、图片、文件消息

接收侧会把 10 分钟内尚未推送的同会话待处理消息放入本地待处理队列。只要队列中已经出现文本消息，就会把同批次中的图片/文件资源归一化后和文本一起推送；如果 10 分钟内始终只有图片或文件，则标记过期并丢弃。成功推送 `connect add-request` 后，插件会立即回复 `<已收到>任务将在30秒内批量执行，可通过新消息更新内容。`。

此外，插件始终提供六个固定元信息命令：

- `command` 固定返回 `["command","help","name","param","scope","schema","init","send","start","stop"]`
- `help` 打印插件使用手册
- `param` 固定返回带字段说明的示例对象数组，字段 key 固定为 `appId`、`appSecret`
- `name` 固定返回 `{"key":"feishu","name":"飞书"}`
- `scope` 固定返回 `["reuse","agent","provider","thinking","swarm"]`
- `schema` 固定返回飞书插件响应 JSON Schema

## 编译

在 `/path/to/deepright/cli/module/connect` 目录执行：

```bash
/opt/homebrew/bin/go build -o ../plugins/feishu ./feishu
```

## 需求目录

- 主需求文档：[REQUIREMENT.md](REQUIREMENT.md)
- 待处理聚合迭代需求：[iteration/20260521-1/REQUIREMENT.md](iteration/20260521-1/REQUIREMENT.md)
- 入库即时回执迭代需求：[iteration/20260717-1/REQUIREMENT.md](iteration/20260717-1/REQUIREMENT.md)
- 入库即时回执迭代手册：[iteration/20260717-1/USER_GUIDE.md](iteration/20260717-1/USER_GUIDE.md)
- 当前用户手册：[USER_GUIDE.md](USER_GUIDE.md)

## 固定输出命令

这些命令无需启动服务即可使用：

```bash
../plugins/feishu command
../plugins/feishu help
../plugins/feishu name
../plugins/feishu param
../plugins/feishu scope
../plugins/feishu schema
```

返回示例：

```json
["command","help","name","param","scope","schema","init","send","start","stop"]
```

```json
["reuse","agent","provider","thinking","swarm"]
```

```json
{
  "type": "object",
  "properties": {
    "content": {
      "type": "string",
      "description": "The content is presented in MARKDOWN FORMAT text, and the file paths have been replaced with filenames."
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
    "content",
    "why_do_this"
  ]
}
```

```json
[{
  "appId": "飞书开放平台（https://open.feishu.cn/app）中应用凭证的App ID ",
  "appSecret": "App Secret"
}]
```

```json
{"key":"feishu","name":"飞书"}
```

## 配置方式

`feishu` 的运行参数来自 `connect` 中 `key=feishu` 的 meta 配置，核心字段如下：

```json
{
  "appId": "cli_xxx",
  "appSecret": "yyy"
}
```

`param` 返回的是字段说明示例，不是运行时真实值；`meta-create` / `meta-update` 里仍然使用同名字段传入真实配置。

通常通过 `integration` 顶层命令写入：

```bash
./integration connect meta-create --key feishu --meta '{"appId":"x","appSecret":"y"}' --callback ignored --agent a --model deepseek
```

通过页面右上角即时通讯扇形菜单打开 `feishu` 插件配置时，运行时绑定规则如下：

- 勾选“复用当前会话”：
  - 启动时只会把当前会话的 `ChatId` 写入插件配置
  - `AgentId` 仍然使用插件页面当前选中的 Agent，不会自动切换，也不会被锁定
- 不勾选“复用当前会话”：
  - 仍然可以手动选择 Agent
  - 插件配置中的 `chatId` 会固定写成 `feishu`

以上规则只针对插件页面和 `POST /api/plugins/config?key=feishu...` 这条配置链路；如果直接手工调用 `meta-create` / `meta-update`，需要自行传入期望的 `chatId`。

## 接收模式

启动插件：

```bash
../plugins/feishu start --connect-bin ./connect
```

停止插件：

```bash
../plugins/feishu stop
```

常见参数：

- `--connect-bin`：`connect` 或 `integration` 二进制路径
- `--addr` / `--port`：`connect` 服务地址
- `--log-file`：消息日志，默认 `./feishu.log`
- `--pid-file`：进程 PID 文件，默认 `./feishu.pid`

### 入库即时回执

消息形成可处理的文本批次后，插件的处理顺序如下：

1. 按同会话聚合规则调用 `connect add-request`，将消息和原始飞书报文可靠入库。
2. 入库成功后立即回复原飞书消息：`<已收到>任务将在30秒内批量执行，可通过新消息更新内容。`。
3. Proxy/Integration 仍按自身周期创建、执行任务和发送最终结果；它们不会再为同一条飞书请求发送旧的 `<开始执行>可通过新消息更新任务内容`。

因此，“已收到”只表示消息已入库，不表示模型任务已经开始或完成。仅有图片/文件时，插件会继续等待同会话文本消息，不会先回复“已收到”。

回执使用入库请求中保存的原始飞书报文，因此始终回复到对应的原消息。回执状态会写入 `feishu.pending.json`：重复事件、服务重连或重启不会重复回执；若飞书发送回执失败，插件会保留待回执记录，并在下一次 30 秒扫描时重试，而不会再次写入 `add-request`。

## 主动发送消息

命令格式：

```bash
../plugins/feishu send --message 原消息报文JSON --content 消息文本内容 --image /tmp/a.png,/tmp/b.jpg --file /tmp/a.pdf,/tmp/b.txt
../plugins/feishu init --message 原消息报文JSON --content 消息文本内容 --image /tmp/a.png,/tmp/b.jpg --file /tmp/a.pdf,/tmp/b.txt
```

回复原消息：

```bash
../plugins/feishu send --message '{"id":1,"rawRequest":"{\"source\":\"feishu\",\"receivedAt\":\"2026-05-22T00:00:00+08:00\",\"message\":{\"messageId\":\"om_xxx\",\"chatId\":\"oc_xxx\",\"messageType\":\"text\",\"content\":\"你好\",\"rawContent\":\"{\\\"text\\\":\\\"你好\\\"}\",\"raw\":\"{\\\"schema\\\":\\\"2.0\\\",\\\"event\\\":{\\\"message\\\":{\\\"message_id\\\":\\\"om_xxx\\\"}}}\"},\"pending\":[],\"groupedBy\":\"chat_id\",\"windowSecs\":30,\"expireSecs\":600}"}' --content "收到"
../plugins/feishu init --message '{"id":1,"rawRequest":"{\"source\":\"feishu\",\"receivedAt\":\"2026-05-22T00:00:00+08:00\",\"message\":{\"messageId\":\"om_xxx\",\"chatId\":\"oc_xxx\",\"messageType\":\"text\",\"content\":\"你好\",\"rawContent\":\"{\\\"text\\\":\\\"你好\\\"}\",\"raw\":\"{\\\"schema\\\":\\\"2.0\\\",\\\"event\\\":{\\\"message\\\":{\\\"message_id\\\":\\\"om_xxx\\\"}}}\"},\"pending\":[],\"groupedBy\":\"chat_id\",\"windowSecs\":30,\"expireSecs\":600}"}' --content "<已收到>任务将在30秒内批量执行，可通过新消息更新内容。"
```

发送图片：

```bash
../plugins/feishu send --message '{"id":1,"rawRequest":"{\"source\":\"feishu\",\"receivedAt\":\"2026-05-22T00:00:00+08:00\",\"message\":{\"messageId\":\"om_xxx\",\"chatId\":\"oc_xxx\",\"messageType\":\"text\",\"content\":\"你好\",\"rawContent\":\"{\\\"text\\\":\\\"你好\\\"}\",\"raw\":\"{\\\"schema\\\":\\\"2.0\\\",\\\"event\\\":{\\\"message\\\":{\\\"message_id\\\":\\\"om_xxx\\\"}}}\"},\"pending\":[],\"groupedBy\":\"chat_id\",\"windowSecs\":30,\"expireSecs\":600}"}' --image /tmp/a.png,/tmp/b.jpg
```

发送文件：

```bash
../plugins/feishu send --message '{"id":1,"rawRequest":"{\"source\":\"feishu\",\"receivedAt\":\"2026-05-22T00:00:00+08:00\",\"message\":{\"messageId\":\"om_xxx\",\"chatId\":\"oc_xxx\",\"messageType\":\"text\",\"content\":\"你好\",\"rawContent\":\"{\\\"text\\\":\\\"你好\\\"}\",\"raw\":\"{\\\"schema\\\":\\\"2.0\\\",\\\"event\\\":{\\\"message\\\":{\\\"message_id\\\":\\\"om_xxx\\\"}}}\"},\"pending\":[],\"groupedBy\":\"chat_id\",\"windowSecs\":30,\"expireSecs\":600}"}' --file /tmp/a.pdf,/tmp/b.txt
```

图文或文档混发：

```bash
../plugins/feishu send --message '{"id":1,"rawRequest":"{\"source\":\"feishu\",\"receivedAt\":\"2026-05-22T00:00:00+08:00\",\"message\":{\"messageId\":\"om_xxx\",\"chatId\":\"oc_xxx\",\"messageType\":\"text\",\"content\":\"你好\",\"rawContent\":\"{\\\"text\\\":\\\"你好\\\"}\",\"raw\":\"{\\\"schema\\\":\\\"2.0\\\",\\\"event\\\":{\\\"message\\\":{\\\"message_id\\\":\\\"om_xxx\\\"}}}\"},\"pending\":[],\"groupedBy\":\"chat_id\",\"windowSecs\":30,\"expireSecs\":600}"}' --content "附件如下" --image /tmp/a.png --file /tmp/a.pdf
```

说明：

- `--message` 为必填，值必须为 `connect add-request` 返回的请求 JSON，且其中的 `rawRequest` 必须符合最新 envelope 协议
- 飞书原消息 `messageId` 会从 `--message` 中自动提取
- `--image`、`--file` 可以为空
- `--content` 默认会以飞书 `interactive` 卡片消息发送，正文按 `MARKDOWN FORMAT` 约定渲染
- `--image` 会先调用飞书图片上传接口拿到 `image_key`，再发送图片消息
- `--file` 会先调用飞书文件上传接口拿到 `file_key`，再发送文件消息
- 如果同时带文本和附件，插件会拆成多次飞书 API 调用，发送顺序为图片 -> 文件 -> 文本
- `send` 与 `init` 的参数和处理方式完全一致
- `send` / `init` 只用于回复已有消息，不再支持省略原消息后直接新发一条消息
- `send` / `init` 的整次发送执行超时为 180 秒；超时会直接返回失败

如果 `--content` 直接传入的是符合 `feishu schema` 的 JSON Object，或者报文里是“说明文字 + JSON Object”的混合文本，插件都会先尝试提取其中第一段合法 JSON Object，再做一次归一化：

```json
{
  "content": "真正发送的飞书文本",
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

- `content`：作为真正传给 `--content` 的文本内容
- `artifacts[].path`：必须是绝对路径
- 图片类型路径会自动并入 `--image`
- 非图片类型路径会自动并入 `--file`
- 如果报文前后夹带了解释性文本，插件会先从整段文本中提取第一段合法 JSON Object，再按同一份 schema 继续处理
- 如果原命令里已经显式传了 `--image` / `--file`，会与 schema 里的附件做合并去重
- `content` 里的 Markdown 图片会在发送前预处理：
  远程 `http/https` 图片会先下载到 `feishu_artifacts`，再上传飞书并替换成 `image_key`
  本地绝对路径图片和 `file://` 图片会直接上传飞书并替换成 `image_key`
- 解析过程会输出带 `content`、`image`、`file` 标记的日志
- 如果整段报文里找不到可提取、且符合 `feishu schema` 的 JSON Object，则完全按原逻辑处理，不做拆解
- 如果 schema 归一化、Markdown 图片下载、或 Markdown 图片上传失败，则会降级成“仅发送原始 `--content` 文本，不发送附件”
- 如果降级后原始 `--content` 文本仍然发送失败，则会再兜底发送 `"<消息异常>请登录客户端查看"`
- 飞书 `send` / `init` 中，文本卡片发送、图片上传+发送、文件上传+发送都最多自动重试 3 次；超过 3 次会直接终止本次发送
- 单独附件发送失败后不再继续静默降级为“仅发送文本”，而是按重试上限失败退出
- 对应日志会写入 `feishu.log`：重试阶段为 `stage=send-retry`，超过上限终止为 `stage=send-terminate`

对应日志示例：

```text
[feishu] schema-send content="请查收附件"
[feishu] schema-send image="/tmp/demo.png"
[feishu] schema-send file="/tmp/report.pdf"
```

例如：

```bash
../plugins/feishu send \
  --message '{"id":1,"original":"{\"schema\":\"2.0\",\"event\":{\"message\":{\"message_id\":\"om_xxx\",\"content\":\"{\\\"text\\\":\\\"你好\\\"}\",\"message_type\":\"text\"}}}"}' \
  --content '{"content":"请查收附件","artifacts":[{"path":"/tmp/demo.png","desc":"截图"},{"path":"/tmp/report.pdf","desc":"报告"}],"why_do_this":"补充执行结果"}'
```

带 Markdown 图片的 schema 内容示例：

```bash
../plugins/feishu send \
  --message '{"id":1,"original":"{\"schema\":\"2.0\",\"event\":{\"message\":{\"message_id\":\"om_xxx\",\"content\":\"{\\\"text\\\":\\\"你好\\\"}\",\"message_type\":\"text\"}}}"}' \
  --content '{"content":"请查看结果\n\n![本地图片](/tmp/demo.png)\n\n![远程图片](https://example.com/demo.png)","why_do_this":"补充执行结果"}'
```

带前缀说明文本的 schema 内容示例：

```bash
../plugins/feishu send \
  --message '{"id":1,"original":"{\"schema\":\"2.0\",\"event\":{\"message\":{\"message_id\":\"om_xxx\",\"content\":\"{\\\"text\\\":\\\"你好\\\"}\",\"message_type\":\"text\"}}}"}' \
  --content '先看看桌面文件情况，同时尝试截图。截图已生成。现在截图下载目录：{"content":"请查收附件","artifacts":[{"path":"/tmp/demo.png","desc":"截图"},{"path":"/tmp/report.pdf","desc":"报告"}],"why_do_this":"补充执行结果"}'
```

飞书插件接收到消息后，会通过 `add-request` 推送：

```bash
./integration connect add-request --key feishu --externalId <md5(create_time+content)> --content <归一化后的文本> --artifacts <附件路径> --original <原始事件JSON> --created <消息时间戳> --schema <feishu schema输出>
```

说明：

- 当前接收链路会先进入本地 pending 队列，再按 30 秒扫描规则决定是否推送
- 同一个 `chat_id` 下，只要 10 分钟窗口内已经出现文本消息，就会把这批消息中的图片/文件一并归一化到最终 `content` 和 `artifacts`
- 归一化文本格式固定为：图片 `"[image]绝对路径"`，文件 `"[file]绝对路径"`
- 仅包含图片/文件且超过 10 分钟的 pending 消息会直接过期，不会执行 `add-request`
- `--schema` 的值与 `../plugins/feishu schema` 返回值完全一致
- 飞书插件对外声明的 schema 和内部写入 `add-request` 的 schema 是同一份数据
- 这样 `connect_request.response_schema`、后续桥接出的 cron 任务明细、以及最终回复飞书时的 JSON 标准化可以保持统一语义

如果直接查看帮助，`../plugins/feishu help` 也会明确列出：

- `scope`
- `schema`
- `--content TEXT|JSON`
- `add-request` 会在缺省时自动补上 `--schema`
- 长驻飞书进程在 `load-config`、`create-session`、`run-session`、`flush-pending` 等链路若连续失败，也只会自动重试 3 次；超过后会写入 `stage=service-retry` / `stage=service-terminate` 并退出进程

## 日志

插件默认写两类日志：

- `./feishu.log`：消息流水日志；接收消息和主动 `send` / `init` 时都会追加一行
- `./feishu.runtime.log`：运行诊断日志

执行 `send` 或 `init` 时，`feishu.log` 会按阶段追加记录：

```text
2026-05-22T10:00:00+08:00,stage=send-request action="send" message="{\"id\":1,...}" content="执行完成"
2026-05-22T10:00:00+08:00,stage=send-parse action="send" reply_to="om_xxx" raw_request="{\"source\":\"feishu\",...}"
2026-05-22T10:00:00+08:00,stage=send-result action="send" name="feishu" target="oc_xxx" reply_to="om_xxx" types="text" result="{\"name\":\"feishu\",...}"
2026-05-22T10:01:00+08:00,stage=send-failed action="init" step="parse-message" err="message.messageId not found"
```

说明：

- `stage=send-request` 记录本次命令收到的原始请求报文和发送参数
- `stage=send-parse` 记录从 `rawRequest.message.messageId` 提取到的回复锚点
- `stage=send-result` 记录最终发送结果，包括目标会话、回复消息 ID、发送类型和返回结果
- `stage=send-failed` 记录失败阶段和失败原因；当 `rawRequest` 缺失、JSON 非法或 `message.messageId` 缺失时，会直接在这里体现

---

---

## 迭代 20260716-1 至 20260717-1：可靠入库回执

可处理的飞书消息会先按既有规则入队、与同会话文本合并并通过 `connect add-request` 成功入库。入库成功后，机器人立即在该消息的原会话回复：

```text
<已收到>任务将在30秒内批量执行，可通过新消息更新内容。
```

- 此回执不等待 Proxy 的 30 秒扫描，也不表示任务已经开始或完成；最终结果仍由原有流程回复。
- 回执使用本次请求保留的原始飞书报文，因此不会串发到其他会话。
- 状态按稳定外部消息标识持久化。重投、重连或重启不会重复回执；仅发送成功后才标记为已回执，发送失败会在后续扫描中重试。
- 仅附件消息会继续等待同会话文本。附件与文本合并并成功入库后只回执一次；被过滤、去重、过期或入库失败的消息不会收到回执。
- 飞书的入库回执不再使用 `<开始执行>可通过新消息更新任务内容`。任务创建、启动和执行状态不由这条回执表示。
