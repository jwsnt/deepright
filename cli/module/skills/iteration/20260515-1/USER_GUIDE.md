# Skills 20260515-1

本轮迭代让 `skills` 模块兼容 `compatibility` 的两种 YAML 声明格式：

- 字符串
- 字符串列表

支持示例：

```yaml
compatibility: macOS (Darwin)
```

```yaml
compatibility:
  - macOS (Darwin)
  - zsh shell
```

统一输出规则：

- 保持字段类型为字符串
- 列表项按原顺序保留
- 去除每项首尾空白
- 忽略空字符串项
- 使用 `; ` 连接

例如：

```yaml
compatibility:
  - macOS (Darwin)
  - zsh shell
```

输出：

```json
{
  "compatibility": "macOS (Darwin); zsh shell"
}
```

这样可以避免因为技能作者使用数组写法，导致整个 `SKILL.md` 被当成 YAML 类型不匹配而解析失败。
