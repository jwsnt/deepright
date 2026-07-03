本次迭代为 Proxy 的 `/api/config` 补充了“删除指定模型配置”的能力，用于配合设置页在删除模型时立即持久化到服务端。

### `/api/config`

除了原有的 `POST /api/config?agentId=xxx` 写入 Agent `config.json` 之外，现在还支持以下请求体：

```json
{
  "action": "delete_model",
  "model": "deepseek"
}
```

说明：

- `action` 固定为 `delete_model`
- `model` 为要删除的模型名，会先按现有别名规则归一化后再删除
- 删除成功后，会返回最新的 `models` 和 `updatedAt`
- 如果模型原本不存在，接口也会返回成功，便于前端幂等调用

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

### 持久化与日志

- 模型配置会从 SQLite `data` 的 `token_store` 表中直接删除
- 同时会向 `proxy_agent_provider_log` 额外写入一条 `action=delete` 的审计日志
- 原有 `/api/token` 的新增和更新逻辑不受影响，仍继续负责批量保存模型配置
