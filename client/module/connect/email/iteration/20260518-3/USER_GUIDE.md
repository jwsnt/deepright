# Email 迭代手册（20260518-3）

## 本次变更

本次迭代补齐了邮件插件 `send` 命令对结构化 `--content` 的兼容处理。

当 `--content` 传入的是一个 JSON Object，并且该 JSON 符合 `email schema` 命令返回的 Schema 时，邮件插件会先解析该对象，再按邮件发送命令的既有参数语义执行发送。

如果 `--content` 不是 JSON Object，则保持原有逻辑不变。若它是 JSON Object 但不符合 `email schema`、JSON 解析失败、正文中的图片引用下载失败或附件处理失败，则会降级为“整个原始 `--content` 作为正文发送，并且不处理图片和文件附件”。

这里校验使用的 schema 与 `../plugins/email schema` 的 CLI 输出、以及邮件接收侧透传给 `integration connect add-request --schema` 的内容完全相同。

## 当前行为

1. `send` 执行时会先检查 `--content` 是否为 `{}` 形式的 JSON Object
2. 若 JSON Object 符合 `../plugins/email schema` 返回的 Schema，则按结构化字段拆解
3. `content` 字段会作为真实邮件正文
4. `artifacts` 数组中的每个 `path` 都会作为待发送附件
5. 附件会按 MIME 类型自动区分：
   - 图片进入 `--image`
   - 非图片文件进入 `--file`
6. 解析过程中会输出带有 `content`、`image`、`file` 标记的日志，便于排查实际发送内容
7. 还会扫描正文里的远程图片链接、本地绝对路径图片和 `file://` 图片链接，并自动补充到图片附件
8. 若结构不匹配 Schema 或归一化过程失败，则回退为原始正文整体发送，不再处理任何图片和文件附件
9. 若降级发送仍失败，则继续兜底发送 `<消息异常>请登录客户端查看`

## 结构化 content 格式

`--content` 支持的结构化 JSON 如下：

```json
{
  "content": "真正发送的邮件正文",
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

- `content`：实际发出的正文内容；schema 对外仍按纯文本约定，发送阶段会自动兼容普通文本或 HTML
- `artifacts[].path`：附件的绝对路径
- `artifacts[].desc`：附件用途说明，当前主要用于满足 Schema 约束
- `why_do_this`：过程说明字段，当前用于满足 Schema 约束

## 发送示例

直接按原逻辑发送文本：

```bash
../plugins/email send \
  --message '{"message":{"raw":"{\"headers\":[{\"name\":\"From\",\"value\":\"Sender <sender@example.com>\"},{\"name\":\"Subject\",\"value\":\"原始主题\"},{\"name\":\"Message-ID\",\"value\":\"<origin@example.com>\"}],\"content\":\"hello\"}"}}' \
  --content "普通文本正文"
```

使用结构化 JSON 发送正文和附件：

```bash
../plugins/email send \
  --message '{"message":{"raw":"{\"headers\":[{\"name\":\"From\",\"value\":\"Sender <sender@example.com>\"},{\"name\":\"Subject\",\"value\":\"原始主题\"},{\"name\":\"Message-ID\",\"value\":\"<origin@example.com>\"}],\"content\":\"hello\"}"}}' \
  --content '{"content":"请查收附件","artifacts":[{"path":"/tmp/demo.png","desc":"截图"},{"path":"/tmp/report.pdf","desc":"报告"}],"why_do_this":"补充执行结果"}'
```

在第二种情况下，预期行为为：

- 邮件正文使用 `请查收附件`
- `/tmp/demo.png` 作为图片附件发送
- `/tmp/report.pdf` 作为文件附件发送

## 回退规则

以下情况都会触发降级：

- `--content` 不是合法 JSON
- `--content` 不是 JSON Object
- 缺少 Schema 要求的必填字段
- `artifacts` 中元素结构不合法
- `artifacts[].path` 不是绝对路径
- `artifacts[].path` 对应文件不存在
- 正文中的远程图片下载失败
- 附件读取或发送失败

回退后行为：

- 原始 `--content` 不做拆解
- 仍直接作为邮件正文
- 不再处理图片和文件附件

## 验证方式

先确认插件公开的 Schema：

```bash
../plugins/email schema
```

再执行结构化发送：

```bash
../plugins/email send \
  --message '{"message":{"raw":"{\"headers\":[{\"name\":\"From\",\"value\":\"Sender <sender@example.com>\"},{\"name\":\"Subject\",\"value\":\"原始主题\"},{\"name\":\"Message-ID\",\"value\":\"<origin@example.com>\"}],\"content\":\"hello\"}"}}' \
  --content '{"content":"请查收附件","artifacts":[{"path":"/tmp/demo.png","desc":"截图"},{"path":"/tmp/report.pdf","desc":"报告"}],"why_do_this":"补充执行结果"}'
```

验证预期：

- `send` 能正常执行
- 图片被归入图片附件发送链路
- 文件被归入文件附件发送链路
- 日志中可看到 `content`、`image`、`file` 标记
- 不符合 Schema 或归一化失败的 JSON 输入不会直接中断，而是回退为“原始正文整体发送”

## 兼容性说明

- 本次变更只增强 `send` 对 `--content` 的解释能力
- 原有 `send --content "普通文本"` 用法保持不变
- 原有 `--image`、`--file` 显式传参方式保持不变
- 该能力对未使用结构化 JSON 的现有调用方无破坏

## 说明

- 本次迭代手册对应当前目录下的 `REQUIREMENT.md`
- 当前目录交付文件为：`REQUIREMENT.md`、`USER_GUIDE.md`
