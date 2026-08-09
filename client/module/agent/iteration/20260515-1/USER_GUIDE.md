# Agent 20260515-1

本轮迭代让 Agent 元数据中的 `skills[].compatibility` 与 skills 模块保持一致，兼容两种 `SKILL.md` 写法：

- 字符串
- 字符串列表

无论技能文件原本怎么写，`agent` 输出的 `compatibility` 都会被规范化成单个字符串。

例如，技能文件：

```yaml
compatibility:
  - macOS (Darwin)
  - zsh shell
```

Agent 输出中会变成：

```json
{
  "compatibility": "macOS (Darwin); zsh shell"
}
```

说明：

- 该能力直接复用 `skills` 共享解析内核
- 列表项会按顺序保留
- 每项会去掉首尾空白
- 空项会被忽略
- 最终使用 `; ` 连接

因此，上游使用 `agent` 输出时，可以始终把 `compatibility` 当成普通字符串处理。
