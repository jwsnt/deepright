# Agent 迭代 20260511-1 使用手册

## 目标

本次迭代让 `agent` 元数据中的 `skills` 字段改为实时扫描：

- 每次读取 Agent 元数据时，都重新遍历对应 Agent 的 `skills` 目录及子孙目录
- 不再让 `skills` 字段跟随整份 Agent metadata 一起被 `--agent-cache` 缓存住

## 行为变化

本次变化后，以下链路中的 `skills` 都会实时刷新：

- `./agent-scanner <目录>`
- `./agent-scanner --get <agentId> <目录>`
- `./agent-scanner --skills <agentId> <目录>`
- 其他模块通过 `agentcore` 读取 Agent metadata 的所有调用

例如：

1. 先启动一个长期进程，使用较大的 `--agent-cache`
2. 修改某个 Agent 的 `skills/**/SKILL.md`
3. 再次查询 Agent metadata

返回结果中的 `agents[].skills` 会立即反映最新技能文件内容，不需要等待缓存过期。

## 兼容说明

- Agent 其他字段仍继续复用原有缓存策略
- `skills` 目录仍兼容历史上的 `SKILL.md` 与 `SKILL` 文件
- `agent` 对外 CLI 参数、JSON 结构、`GetAgentIDs` / `GetAgentByID` / `GetSkillNames` 接口签名均保持不变
- 更完整说明请继续参考上级手册 `../../USER_GUIDE.md`
