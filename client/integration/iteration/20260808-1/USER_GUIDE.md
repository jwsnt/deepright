# 历史会话候选查询使用手册

`GET /api/session_recovery_candidates` 提供普通页面会话的恢复候选列表。它只读取现有 `chat_log`，不创建新表、不补写日志，也不改变 `/api/restore` 的恢复协议。

请求使用 `page` 指定从 1 开始的页码，每页固定最多 10 条。响应包含 `page`、`pageSize`、`hasMore` 和 `sessions`；每个候选包含 `agentId`、`chatId`、`lastPrompt`、`completedAt`。

可重复提供 `exclude` 参数排除当前页面已展示的会话，例如：

```text
/api/session_recovery_candidates?page=1&exclude=chat-a&exclude=chat-b
```

接口仅返回普通页面会话中最后一次正常完成的回答：该回答必须具有 `data: [DONE]` 完成标记。异常、取消、未完成流和定时备忘录会话不会返回。`lastPrompt` 是这次成功回答对应请求中的最后一条用户提问。

查询在 SQLite 中按完成时间、日志 ID 倒序分页，并由聊天恢复相关联合索引支持；Integration 与 Proxy 保持相同的接口和查询语义。
