# Integration 迭代 20260511-2 使用手册

## 目标

本次迭代将 `skills` 实时刷新能力收口到 `integration` 统一入口。

覆盖链路：

- `/v1/chat/completions`
- `cli/get`
- `cli/pub`
- integration 内部发起的 cron 执行请求 metadata

## 行为说明

`integration` 继续复用统一的 Agent 元数据内核，不在入口层重复实现技能扫描逻辑。

本次变化后，即使 `integration` 启动时使用了较长的 `--agent-cache`：

- 修改 `agent/<AgentId>/skills/**/SKILL.md`
- 再次调用 `/v1/chat/completions`、等待下一轮 `cli/get` / `cli/pub`、或触发 cron 执行

这些链路里的 `metadata.agents[].skills` 也会立即反映最新技能文件内容。

## 使用方式

按原方式启动即可：

```bash
cd /path/to/deepright/cli/module/integration
./integration --agent-dir ./agent
```

本次迭代没有新增 CLI 参数或 HTTP 路由。

## 兼容说明

- `integration` 仍然是最终对外唯一主二进制
- `--agent-cache` 仍保留，用于 Agent 其他共享字段缓存
- `skills` 字段改为每次实时遍历 Agent 的 `skills` 目录，因此不会受缓存 TTL 限制
- 更完整说明请继续参考上级手册 `../../USER_GUIDE.md`
