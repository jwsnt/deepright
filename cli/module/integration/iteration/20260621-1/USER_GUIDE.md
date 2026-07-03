# Integration 迭代 20260621-1 使用手册

## 变更说明

- 新增 `POST /api/message_insert/add`
- 新增 `POST /api/message_insert/del`
- 新增 `POST /api/message_insert/status`
- 新增共享 SQLite `data` 中的 `message_insert` 表，状态固定为：
  - `0`：待上传
  - `1`：已上传
  - `2`：取消
- `integration` 内部 `cli/get -> exec -> cli/pub` 链路会在提交 `/cli/pub` 前，自动读取当前 `chatId` 下最多 `5` 条状态为 `0` 的插入消息，并在成功回传后把这些消息标记为 `1`

## 接口示例

### `/api/message_insert/add`

```json
{
  "agentId": "agent-a",
  "chatId": "chat-001",
  "mid": 1718966400000,
  "message": "这是一条待插入的排队消息"
}
```

### `/api/message_insert/del`

```json
{
  "chatId": "chat-001",
  "mid": 1718966400000
}
```

### `/api/message_insert/status`

```json
{
  "chatId": "chat-001",
  "mid": [1718966400000, 1718966405000]
}
```

## CLI

也支持直接通过 CLI 读写：

```bash
./integration message-insert add --agentId agent-a --chatId chat-001 --mid 1718966400000 --message 'HELLO'
./integration message-insert del --chatId chat-001 --mid 1718966400000
./integration message-insert status --chatId chat-001 --mid 1718966400000,1718966405000
```
