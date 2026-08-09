# Site 迭代 20260419 使用手册

## 变更说明

调整 `/v1/chat/completions` 请求报文中的 `messages` 字段：仅包含用户最后（最新）的一条请求，不再发送历史消息记录。

## 请求报文示例

```json
{
  "model": "kimi",
  "messages": [
    {
      "role": "user",
      "content": "用户最新输入的内容"
    }
  ],
  "stream": true,
  "metadata": {
    "agentId": "a",
    "chat": "会话UUID"
  }
}
```

## 注意事项

- 历史消息仍然在本地存储和页面中正常展示，仅请求报文不再携带历史
- 其他功能（会话管理、SSE 流式响应、设置等）不受影响
