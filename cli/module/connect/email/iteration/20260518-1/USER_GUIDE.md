# Email 迭代手册（20260518-1）

## 本次变更

本次迭代为 `email` 插件新增固定命令 `schema`，用于返回邮件插件的 Response JSON Schema。

该命令用于让外部模块在不感知插件内部实现的情况下，直接获取邮件插件要求的结构化响应约束。

## 命令格式

```bash
../plugins/email schema
```

返回值固定为：

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

1. `schema` 作为邮件插件固定输出命令，和 `name`、`param`、`command` 一样可直接通过 CLI 调用
2. 返回值为合法 JSON object，可直接被其他模块作为 Json String 消费
3. 当前邮件插件的 `command` 输出中也包含 `schema`
4. `schema` 输出保持稳定，便于 `integration`、`proxy` 或其他插件容器复用
5. 后续 `add-request --schema` 和 `send` 的 schema 校验也会复用同一份定义，避免内容漂移

## 适用场景

- 邮件插件自身对外声明结构化响应格式
- `integration connect add-request --schema` 的参数来源
- 后续 cron / proxy 链路的 structured output 约束透传

## 验证方式

```bash
../plugins/email command
../plugins/email schema
```

验证预期：

- `command` 返回值中包含 `schema`
- `schema` 返回合法 JSON object
- 返回结果包含 `content`、`artifacts`、`why_do_this` 三个属性
- `content.description` 明确声明正文为纯文本格式，且路径展示替换为文件名
- `required` 中固定包含 `content`、`why_do_this`

## 说明

- 本次迭代手册对应当前目录下的 `REQUIREMENT.md`
- 当前目录交付文件为：`REQUIREMENT.md`、`USER_GUIDE.md`
