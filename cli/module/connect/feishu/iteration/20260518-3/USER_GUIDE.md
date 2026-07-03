# Feishu 迭代手册（20260518-2）

## 本次变更

本次迭代补齐了飞书插件 `send` 命令对结构化 `--content` 的兼容处理。

当 `--content` 传入的是一个 JSON Object，并且该 JSON 符合 `feishu schema` 命令返回的 Schema 时，飞书插件会先解析该对象，再按飞书发送命令既有参数语义执行发送。

如果 `--content` 不是 JSON Object，或者虽然是 JSON Object 但不符合 `feishu schema` 的要求，则保持原有逻辑不变，直接把原始 `--content` 字符串作为飞书文本内容。

## 当前行为

1. `send` 执行时会先检查 `--content` 是否为 `{}` 形式的 JSON Object
2. 若 JSON Object 符合 `../plugins/feishu schema` 返回的 Schema，则按结构化字段拆解
3. `content` 字段会作为真实飞书文本内容
4. `artifacts` 数组中的每个 `path` 都会作为待发送附件
5. 附件会按 MIME 类型自动区分：
   - 图片进入 `--image`
   - 非图片文件进入 `--file`
6. 如果原命令里已经显式传了 `--image`、`--file`，会和结构化 JSON 中的附件一起合并去重
7. 解析过程中会输出带有 `content`、`image`、`file` 标记的日志，便于排查实际发送内容
8. 若结构不匹配 Schema，则不拆解 JSON，继续按旧逻辑发送

## 结构化 content 格式

`--content` 支持的结构化 JSON 如下：

```json
{
  "content": "真正发送的飞书正文",
  "artifacts": [
    {
      "path": "/abs/path/demo.png",
      "desc": "截图说明"
    },
    {
      "path": "/abs/path/report.pdf",
      "desc": "报告附件"
    }
  ],
  "why_do_this": "保留过程说明"
}
```

字段说明：

- `content`：实际发出的飞书文本内容；schema 描述约定其为 `MARKDOWN FORMAT`
- `artifacts[].path`：附件的绝对路径
- `artifacts[].desc`：附件用途说明，当前主要用于满足 Schema 约束
- `why_do_this`：过程说明字段，当前用于满足 Schema 约束

## 发送示例

直接按原逻辑发送文本：

```bash
../plugins/feishu send \
  --message '{"id":1,"original":"{\"schema\":\"2.0\",\"event\":{\"message\":{\"message_id\":\"om_xxx\",\"content\":\"{\\\"text\\\":\\\"你好\\\"}\",\"message_type\":\"text\"}}}"}' \
  --content "普通文本正文"
```

使用结构化 JSON 发送正文和附件：

```bash
../plugins/feishu send \
  --message '{"id":1,"original":"{\"schema\":\"2.0\",\"event\":{\"message\":{\"message_id\":\"om_xxx\",\"content\":\"{\\\"text\\\":\\\"你好\\\"}\",\"message_type\":\"text\"}}}"}' \
  --content '{"content":"请查收附件","artifacts":[{"path":"/tmp/demo.png","desc":"截图"},{"path":"/tmp/report.pdf","desc":"报告"}],"why_do_this":"补充执行结果"}'
```

在第二种情况下，预期行为为：

- 飞书文本使用 `请查收附件`
- `/tmp/demo.png` 作为图片发送链路处理
- `/tmp/report.pdf` 作为文件发送链路处理

## 回退规则

以下情况都会回退到旧逻辑：

- `--content` 不是合法 JSON
- `--content` 不是 JSON Object
- 缺少 Schema 要求的必填字段
- `artifacts` 中元素结构不合法
- `artifacts[].path` 不是绝对路径

回退后行为：

- 原始 `--content` 不做拆解
- 仍直接作为飞书正文
- 原有 `--image`、`--file` 参数语义保持不变

## 验证方式

先确认插件公开的 Schema：

```bash
../plugins/feishu schema
```

再执行结构化发送：

```bash
../plugins/feishu send \
  --message '{"id":1,"original":"{\"schema\":\"2.0\",\"event\":{\"message\":{\"message_id\":\"om_xxx\",\"content\":\"{\\\"text\\\":\\\"你好\\\"}\",\"message_type\":\"text\"}}}"}' \
  --content '{"content":"请查收附件","artifacts":[{"path":"/tmp/demo.png","desc":"截图"},{"path":"/tmp/report.pdf","desc":"报告"}],"why_do_this":"补充执行结果"}'
```

验证预期：

- `send` 能正常执行
- 图片被归入图片发送链路
- 文件被归入文件发送链路
- 日志中可看到 `content`、`image`、`file` 标记
- 不符合 Schema 的 JSON 输入不会导致发送失败，而是按旧逻辑继续处理

## 兼容性说明

- 本次变更只增强 `send` 对 `--content` 的解释能力
- 原有 `send --content "普通文本"` 用法保持不变
- 原有 `--image`、`--file` 显式传参方式保持不变
- 该能力对未使用结构化 JSON 的现有调用方无破坏

## 说明

- 本次迭代手册对应当前目录下的 `REQUIREMENT.md`
- 当前目录交付文件为：`REQUIREMENT.md`、`USER_GUIDE.md`
