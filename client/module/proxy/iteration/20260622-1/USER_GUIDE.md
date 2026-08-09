# Proxy 迭代手册（20260622-1）

## 本次更新

- `/knowledge_lastUpdate` 改为支持 `agentId`
  - 未传 `agentId`：读取默认兼容记录
  - 传 `agentId`：读取对应 Agent 的 `last_update`
- `/knowledge_path` 改为支持 `agentId`
  - 未传 `agentId`：返回 `--agent-dir/knowledge`
  - 传 `agentId`：返回 `--agent-dir/knowledge/<agentId>`
- `metadata.knowledge.path` 改为 Agent 维度
- `metadata.knowledge.knowledgeCommit` 现在来自共享 sqlite 的 Agent 级 `knowledge_runtime.knowledge_commit`
- 当请求体显式传入 `metadata.knowledge_commit` 时，`proxy` 会按 `metadata.agentId` 把最新值写回共享 sqlite

## 请求示例

```bash
curl 'http://127.0.0.1:8080/knowledge_lastUpdate?agentId=agent-a'
curl 'http://127.0.0.1:8080/knowledge_path?agentId=agent-a'
```

```json
{
  "model": "gpt-4",
  "messages": [
    {
      "role": "user",
      "content": "知识库已经整理完成"
    }
  ],
  "metadata": {
    "agentId": "agent-a",
    "knowledge_commit": true
  }
}
```

## 当前行为

1. `knowledge_commit` 现在会按 Agent 独立持久化
2. 当 `knowledge_commit = true` 时，仍然会强制保留 `knowledge.lastUpdate`
3. SSE 完整结束后，会回写该 Agent 的 `last_update` 与 `knowledge_update_lock.last_requested_at`
4. WIKI 首页路径约定统一为 `knowledge/<agentId>/index.html`

## 说明

- 更完整说明请继续参考上级手册 `../../USER_GUIDE.md`
