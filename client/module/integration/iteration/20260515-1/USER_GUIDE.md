# Integration 20260515-1

本轮迭代把 skills / agent / proxy 对 `compatibility` 的兼容能力统一收口到了最终交付入口 `integration`。

现在通过 `integration` 暴露出去的 Agent 元数据里，`skills[].compatibility` 具备以下规则：

- 兼容字符串写法
- 兼容字符串列表写法
- 最终统一输出为单个字符串

例如：

```yaml
compatibility:
  - macOS (Darwin)
  - zsh shell
```

会在 metadata 中输出为：

```json
{
  "compatibility": "macOS (Darwin); zsh shell"
}
```

生效链路包括：

- `/v1/chat/completions`
- `cli/get`
- `cli/pub`
- integration 内部 cron 执行请求

因此，上层调用方只需要按字符串读取 `skills[].compatibility`，不需要再为数组格式做额外兼容。
