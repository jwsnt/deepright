# Integration 迭代手册

## 本次收口

本次迭代将以下两项 `proxy` 能力收口到 `integration` 主二进制中，保持 HTTP 和 CLI 都可以直接使用，不要求调用方再切换到独立 `proxy`。

- `proxy/iteration/20260515-4`：文件或目录最后更新时间
- `proxy/iteration/20260516-1`：`knowledge_commit` 知识库时间回写

## HTTP 接口

```bash
GET /file/lastUpdate?file=...&agentId=...
```

- `file` 必填
- `agentId` / `agent` 仅在 `file` 为相对路径时必填
- 返回值为纯文本毫秒数，表示目标文件距离当前时间的更新时间差

路径规则：

- 绝对路径：直接解析，兼容大小写不一致
- 相对路径：基于 Agent workspace 解析
- 不支持 `~`
- 不支持通过 `..` 逃逸 workspace
- 文件和目录都支持

示例：

```bash
curl 'http://127.0.0.1:8080/file/lastUpdate?agentId=A&file=USER.md'
curl 'http://127.0.0.1:8080/file/lastUpdate?file=/abs/path/to/USER.md'
```

## CLI 命令

```bash
./integration file-last-update --file 路径 [--agent AgentId]
```

示例：

```bash
./integration file-last-update --agent A --file USER.md
./integration file-last-update --agentId A --file docs/USER.md
./integration file-last-update --file /abs/path/to/USER.md
```

## 说明

- 该能力遵守 integration 的二进制和 CLI 收口原则
- 行为与 proxy 版本保持一致，便于其他模块在 integration 下统一调用

## knowledge_commit 收口

- `integration` 会检查 `/v1/chat/completions` 请求体中的 `metadata.knowledge_commit`
- 当 `knowledge_commit` 明确为 `true` 时，这次请求转发会强制携带 `knowledge.lastUpdate`
- 该场景下不会再检查知识库最后申请更新时间锁（`knowledge_update_lock.last_requested_at`）
- 当 `knowledge_commit` 明确为 `true` 时，会在对应 SSE 响应完整结束后回写知识库时间
- 回写内容同时包括：
  - 知识库最后更新时间（`/knowledge_lastUpdate` 读取的来源）
  - 知识库最后申请更新时间（`knowledge_update_lock.last_requested_at`）

请求示例：

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

行为说明：

1. 只有当 `metadata.knowledge_commit` 明确存在且为 `true` 时，才会触发这次回写
2. 只要 `knowledge_commit = true`，这次转发就会强制保留 `knowledge.lastUpdate`
3. 该场景不会再检查 `knowledge_update_interval`，也不会检查 `knowledge_update_lock`
4. 回写时机不是转发前，而是上游 SSE 响应完整结束之后
5. 回写会同时更新知识库最后更新时间和知识库更新申请锁时间，二者都写成当前完成时间
6. 如果 `knowledge_commit` 不存在，或值为 `false`，则不会触发这次回写
