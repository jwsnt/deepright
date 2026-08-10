# Integration 迭代 20260622-1 使用手册

## 本次更新

- `integration` 收口的知识库运行时改为 Agent 维度
- `/knowledge_lastUpdate` 支持 `agentId` 查询某个 Agent 的最后更新时间
- `/knowledge_path` 支持 `agentId` 返回某个 Agent 的真实知识库目录
- 转发 `/v1/chat/completions` 时：
  - `metadata.knowledge.path` 会改成 `--agent-dir/knowledge/<agentId>`
  - `metadata.knowledge.lastUpdate` 会读取该 Agent 的 `last_update`
  - `metadata.knowledge.knowledgeCommit` 会读取该 Agent 的 `knowledge_commit`
- 当请求显式传入 `metadata.knowledge_commit` 时，`integration` 会按 `metadata.agentId` 持久化这次提交值

## 接口示例

```bash
curl 'http://127.0.0.1:8080/knowledge_lastUpdate?agentId=agent-a'
curl 'http://127.0.0.1:8080/knowledge_path?agentId=agent-a'
```

```json
{
  "model": "gpt-4",
  "stream": true,
  "messages": [
    {
      "role": "user",
      "content": "整理完成"
    }
  ],
  "metadata": {
    "agentId": "agent-a",
    "knowledge_commit": true
  }
}
```

## 行为说明

1. `knowledge_runtime` 现在按 `agent_id` 独立保存 `last_update` 与 `knowledge_commit`
2. `knowledge_commit = true` 时，仍然会强制保留 `knowledge.lastUpdate`
3. SSE 完整结束后，会回写该 Agent 的知识库最后更新时间和更新申请锁时间
4. Site 侧 WIKI 首页路径约定统一为 `knowledge/<agentId>/index.html`

## 说明

- 更完整说明请参考上级手册 `../../USER_GUIDE.md`
