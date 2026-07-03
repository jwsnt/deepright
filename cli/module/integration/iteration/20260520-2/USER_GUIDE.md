本次迭代为 Integration 收口的 `/api/config` 补充了“删除指定模型配置”的能力，和 Proxy 的行为保持一致。

### `/api/config`

Integration 原本通过 `POST /api/config?agentId=xxx` 更新 Agent 工作目录下的 `config.json`。本次新增另一种请求体，用于删除共享模型配置：

```json
{
  "action": "delete_model",
  "model": "deepseek"
}
```

说明：

- `action` 固定为 `delete_model`
- `model` 为待删除模型名，服务端会先做统一归一化
- 删除完成后，响应会直接返回最新的 `models` 与 `updatedAt`
- 如果该模型当前不存在，也会按成功返回，方便上层直接重复调用

响应示例：

```json
{
  "status": 0,
  "action": "delete_model",
  "model": "deepseek",
  "models": {
    "openai": {
      "token": "Bearer sk-openai",
      "__url": "https://api.openai.com/v1/chat/completions",
      "__model": "gpt-5.4",
      "__model_fast": "gpt-5.4",
      "__model_thinking": "gpt-5.4",
      "__model_multi_input": "gpt-5.4",
      "__model_multi_output": ""
    }
  },
  "updatedAt": {
    "openai": "2026-05-20T18:30:00+08:00"
  }
}
```

### 数据收口

- 删除动作直接作用于共享 SQLite `data` 中的 `token_store`
- 同时会向 `proxy_agent_provider_log` 写入一条 `action=delete` 日志
- 原有 `/api/token` 与 `integration token` 的新增、更新和读取行为保持不变
- 原有 `POST /api/config?agentId=xxx` 更新 Agent `config.json` 的能力也保持不变
