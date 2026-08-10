本次迭代将 Integration 收口的 `/api/token` 与 `integration token` 命令同步升级到和 Proxy 一致的扩展模型配置结构。

### `/api/token`

Integration 现在会在共享 SQLite `data` 的 `token_store` 中保存以下字段：

- `token`
- `__url`
- `__model`
- `__model_fast`
- `__model_thinking`

`POST /api/token` 请求体支持两种格式：

```json
{
  "models": {
    "deepseek": "Bearer sk-deepseek"
  }
}
```

```json
{
  "models": {
    "deepseek": {
      "token": "Bearer sk-deepseek",
      "__url": "https://api.example.com/v1",
      "__model": "deepseek-chat",
      "__model_fast": "deepseek-fast",
      "__model_thinking": "deepseek-reasoner"
    }
  }
}
```

返回示例：

```json
{
  "status": 0,
  "models": {
    "deepseek": {
      "token": "Bearer sk-deepseek",
      "__url": "https://api.example.com/v1",
      "__model": "deepseek-chat",
      "__model_fast": "deepseek-fast",
      "__model_thinking": "deepseek-reasoner"
    }
  },
  "updatedAt": {
    "deepseek": "2026-05-19T20:30:00+08:00"
  }
}
```

说明：

- `token` 仍是运行链路必需字段
- 其余 4 个字段仅作为扩展配置保存和返回，可为空
- 旧数据会在启动时自动兼容到新表结构

### `integration token`

CLI 命令：

```bash
./integration token
./integration token --provider deepseek
```

输出现在与 Proxy 保持一致，模型值为对象而不是纯字符串。

示例：

```json
[
  {
    "deepseek": {
      "token": "Bearer sk-deepseek",
      "__url": "https://api.example.com/v1",
      "__model": "deepseek-chat",
      "__model_fast": "deepseek-fast",
      "__model_thinking": "deepseek-reasoner"
    }
  }
]
```
