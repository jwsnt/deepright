# Email 迭代手册（20260518-2）

## 本次变更

本次迭代补齐了邮件插件调用 `add-request` 时的 `schema` 透传。

邮件插件在接收到新邮件并调用 `integration connect add-request` 时，会复用自身 `email schema` 命令对应的同一份 Response JSON Schema，并通过 `--schema` 参数传递给 integration 顶层入口。

当前复用的 schema 内容如下：

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
    "content",
    "why_do_this"
  ]
}
```

## 当前行为

1. 邮件插件收到准入邮件后，仍然会先整理邮件正文、附件路径和原始报文
2. 调用 `integration connect add-request` 时，会额外附加 `--schema`
3. `--schema` 的值与 `email schema` 命令返回值语义完全一致
4. integration 持久化时会写入 `connect_request.response_schema`
5. 后续待处理消息桥接为一次性 cron 任务时，会继续透传到 `task_meta.response_schema` 和 `task_detail.response_schema`

## 设计说明

- 邮件插件不额外发明第二份 schema 协议
- 对外声明使用 `email schema`
- 对内提交请求时也使用同一份 schema 数据
- `content.description` 统一为纯文本正文语义，路径展示替换为文件名
- 这样可以保证插件声明、CLI 输出和后续执行链路的 `response_schema` 语义完全一致

## 验证方式

先创建 meta：

```bash
./integration connect meta-create \
  --name email \
  --meta '{"email":"'$EMAIL'","email_pop3":"'$EMAIL_POP3'","email_smtp":"'$EMAIL_SMTP'","email_password":"'$EMAIL_PASSWORD'","email_whitelist":""}'
```

然后启动或执行邮件插件，使其扫描并推送新邮件请求。

也可以先确认插件公开的 schema：

```bash
../plugins/email schema
```

随后检查邮件插件触发的 `integration connect add-request` 链路，预期行为为：

- 会向 `add-request` 传递 `--schema`
- 传递值与 `email schema` 的输出保持一致
- `content.description` 为 plain-text 文案，并与 `email schema` 输出完全一致
- request 持久化结果中可看到 `responseSchema`

## 兼容性说明

- 本次新增参数仅在邮件插件调用 `add-request` 时附加
- 未传 `schema` 的旧请求仍可继续处理
- 该值保持原始 Json String 语义，不在邮件插件侧重新解释

## 说明

- 本次迭代手册对应当前目录下的 `REQUIREMENT.md`
- 当前目录交付文件为：`REQUIREMENT.md`、`USER_GUIDE.md`
