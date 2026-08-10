# Knowledge 迭代 20260622-01 使用手册

## 本次更新

- `knowledge_runtime` 从全局单条记录改为按 `agent_id` 独立保存
- 每个 Agent 都有独立的：
  - `last_update`
  - `knowledge_commit`
- `knowledge` 运行时路径改为 Agent 维度：
  - 未指定 Agent 时：`<app-dir>/knowledge`
  - 指定 Agent 时：`<app-dir>/knowledge/<agentId>`
- `metadata.knowledge` 现在会输出：
  - `path`
  - `lastUpdate`
  - `knowledgeCommit`

## 子模块接口

- `EnsureRuntimeForAgent(appDir, agentID)`：确保 `knowledge/<agentId>` 与 Agent 级运行时存在
- `LookupRuntimeForAgent(appDir, agentID)`：只读探测某个 Agent 的知识库运行时
- `MetadataForAgent(appDir, agentID)`：输出 Agent 级 `metadata.knowledge`
- `SetLastUpdateForAgent(db, agentID, ts)`：更新某个 Agent 的 `last_update`
- `SetKnowledgeCommitForAgent(db, agentID, value)`：更新某个 Agent 的 `knowledge_commit`

## CLI

```bash
knowledge ensure --app-dir /srv/app --agent-id agent-a
knowledge metadata --app-dir /srv/app --agent-id agent-a
knowledge update-time --app-dir /srv/app --agent-id agent-a --timestamp 1715337600000
knowledge update-commit --app-dir /srv/app --agent-id agent-a --value true
```

## 说明

- 旧的全局运行时记录会迁移到 `agent_id=''` 兼容行
- 未指定 `--agent-id` 时，CLI 仍兼容读取这条默认记录
- 更完整说明请参考上级手册 `../../USER_GUIDE.md`
