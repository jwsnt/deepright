# Proxy 迭代手册（20260516-1）

## 本次更新

- `proxy` 会检查 `/v1/chat/completions` 请求体中的 `metadata.knowledge_commit`
- 当 `knowledge_commit` 明确为 `true` 时，这次请求转发会强制携带 `knowledge.lastUpdate`
- 该场景下不会再检查知识库最后申请更新时间锁（`knowledge_update_lock.last_requested_at`）
- 当 `knowledge_commit` 明确为 `true` 时，会在对应 SSE 响应完整结束后回写知识库时间
- 回写内容同时包括：
  - 知识库最后更新时间（`/knowledge_lastUpdate` 读取的来源）
  - 知识库最后申请更新时间（`knowledge_update_lock.last_requested_at`）

## 请求示例

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
    "knowledge_commit": true
  }
}
```

## 当前行为

1. 只有当 `metadata.knowledge_commit` 明确存在且为 `true` 时，才会触发这次回写
2. 只要 `knowledge_commit = true`，这次转发就会强制保留 `knowledge.lastUpdate`
3. 该场景不会再检查 `knowledge_update_interval`，也不会检查 `knowledge_update_lock`
4. 回写时机不是转发前，而是上游 SSE 响应完整结束之后
5. 回写会同时更新知识库最后更新时间和知识库更新申请锁时间，二者都写成当前完成时间
6. 如果 `knowledge_commit` 不存在，或值为 `false`，则不会触发这次回写

## 说明

- 这个能力适合知识库“手动整理完成”后的显式确认回写
- 更完整说明请继续参考上级手册 `../../USER_GUIDE.md`
