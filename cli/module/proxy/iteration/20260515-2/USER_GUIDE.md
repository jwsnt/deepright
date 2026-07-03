# Proxy 20260515-2

本轮迭代补齐了 `proxy` 注入 metadata 时对技能 `compatibility` 的兼容能力。

`proxy` 不单独解析技能文件，而是直接复用 `agent` / `skills` 的共享结果，所以现在以下链路都会统一得到字符串格式的 `compatibility`：

- `/v1/chat/completions`
- proxy 内部 cron 执行请求

兼容的技能声明方式：

```yaml
compatibility: macOS (Darwin)
```

或：

```yaml
compatibility:
  - macOS (Darwin)
  - zsh shell
```

无论输入格式如何，注入到 metadata 后都会变成：

```json
{
  "compatibility": "macOS (Darwin); zsh shell"
}
```

这样前端和上游服务都可以直接把该字段当成字符串读取，不需要自己兼容数组格式。
