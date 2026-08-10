# Feishu 迭代手册（20260518-1）

## 本次变更

本次迭代为 `feishu` 插件补齐了固定命令 `schema`，并把 `scope` 加入 `command` 返回值。

这样插件容器既可以直接读取飞书插件声明的结构化响应 Schema，也可以稳定读取该插件支持的容器配置范围。

## 命令格式

```bash
../plugins/feishu command
../plugins/feishu scope
../plugins/feishu schema
```

返回值示例：

```json
["command","help","name","param","scope","schema","init","send","start","stop"]
```

```json
["reuse","agent","provider","thinking"]
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

## 当前行为

1. `schema` 成为飞书插件固定输出命令，可直接通过 CLI 调用
2. `schema` 返回值为稳定 JSON object，可直接作为 `--schema` 的 Json String 使用
3. `command` 返回值中现在包含 `scope` 和 `schema`
4. `scope` 固定返回 `["reuse","agent","provider","thinking"]`
5. `schema` 输出保持稳定，便于 `integration`、`proxy` 或其他插件容器复用

## 适用场景

- 飞书插件自身对外声明结构化响应格式
- 飞书插件声明自身支持的容器配置范围
- `integration connect add-request --schema` 的参数来源
- 后续 cron / proxy 链路的 structured output 约束透传

## 验证方式

```bash
../plugins/feishu command
../plugins/feishu scope
../plugins/feishu schema
```

验证预期：

- `command` 返回值中包含 `scope`、`schema`
- `scope` 返回 `["reuse","agent","provider","thinking"]`
- `schema` 返回合法 JSON object
- 返回结果包含 `content`、`artifacts`、`why_do_this` 三个属性
- `content.description` 明确声明正文为 `MARKDOWN FORMAT`，且路径展示替换为文件名
- `required` 中固定包含 `content`、`why_do_this`

## 说明

- 本次迭代手册对应当前目录下的 `REQUIREMENT.md`
- 当前目录交付文件为：`REQUIREMENT.md`、`USER_GUIDE.md`
