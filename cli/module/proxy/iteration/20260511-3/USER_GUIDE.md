# Proxy 迭代 20260511-3 使用手册

## 目标

本次迭代让 `proxy` 注入到 metadata 中的 Agent `skills` 改为实时刷新。

覆盖链路：

- `/v1/chat/completions`
- proxy 内部发起的 cron 执行请求 metadata

## 行为说明

`proxy` 仍然复用统一的 Agent 元数据输出，不在入口层单独维护 skills 扫描逻辑。

本次变化后，即使 `proxy` 启动时配置了较长的 `--agent-cache`：

- 修改 `agent/<AgentId>/skills/**/SKILL.md`
- 再次请求 `/v1/chat/completions`

转发给上游的 `metadata.agents[].skills` 也会立即反映最新技能内容。

## 使用方式

按原方式启动 `proxy` 即可：

```bash
cd /path/to/deepright/cli/module/proxy
./proxy --agent-dir ./agent
```

然后正常调用：

```text
POST /v1/chat/completions
```

无需新增参数，也无需手工清理缓存。

## 兼容说明

- 不新增新的 HTTP 路由
- 不改变原有 metadata 追加覆盖规则
- 不改变 `proxy` 的 `--agent-cache` 参数语义；只是其中不再缓存 `skills` 字段
- 更完整说明请继续参考上级手册 `../../USER_GUIDE.md`
