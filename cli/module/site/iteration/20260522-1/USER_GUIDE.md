# 20260522-1 使用说明

## 迭代目标

本次迭代补齐了居中对话框 `THINKING` 开关的请求协议：当当前会话开启思考模式并发送消息时，请求体会显式写入 `metadata.thinking = true`。

## 变更说明

- 影响范围仅为居中会话区主聊天发送链路。
- 当前会话打开 `THINKING` 后，后续发送到 `/v1/chat/completions` 的请求会自动补充：

```json
{
  "metadata": {
    "thinking": true
  }
}
```

- 关闭 `THINKING` 后，不会再额外附带 `metadata.thinking`。
- `THINKING` 仍按会话独立保存；切换会话时，会回到该会话上次保存的状态。
- 如果同一会话同时开启了 `HTML` 和 `THINKING`，请求体会同时包含 `metadata.html = true` 与 `metadata.thinking = true`。
- 用户气泡上的“重试”会沿用该次请求记录下来的 metadata，因此开启 `THINKING` 时发出的消息在重试后也会继续带上同样的 `metadata.thinking`。

## 验收建议

1. 打开页面并任选一个会话。
2. 打开输入区 `THINKING` 开关，发送一条消息。
3. 在浏览器开发者工具的 Network 中检查对应 `/v1/chat/completions` 请求体，确认存在 `metadata.thinking = true`。
4. 关闭 `THINKING` 后再次发送消息，确认请求体中不再携带该字段。
5. 如同时打开 `HTML`，确认同一请求的 `metadata` 中同时存在 `html` 与 `thinking`。

## 说明

- 本次迭代只收口居中会话区 `THINKING` 开关的 metadata 传递，不调整蜂群、插件或右侧备忘录的独立 thinking 参数结构。
- 更完整说明请继续参考上级手册 `../../USER_GUIDE.md`。
