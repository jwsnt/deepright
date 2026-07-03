本次迭代为 Proxy 的模型密钥接口和 `proxy token` 命令补充了 4 个可选字段，用于为每个模型保存更细的路由与模型分层配置。

### `/api/token`

`GET /api/token` 与 `POST /api/token` 现在都会读写以下字段：

- `token`：实际请求头里的鉴权值
- `__url`：模型 URL
- `__model`：基础模型
- `__model_fast`：快速响应模型
- `__model_thinking`：深度思考模型
- `__model_multi_input`：多模态输入模型
- `__model_multi_output`：多模态输出模型

响应示例：

```json
{
  "status": 0,
  "models": {
    "deepseek": {
      "token": "Bearer sk-deepseek",
      "__url": "https://api.example.com/v1",
      "__model": "deepseek-chat",
      "__model_fast": "deepseek-fast",
      "__model_thinking": "deepseek-reasoner",
      "__model_multi_input": "deepseek-vision",
      "__model_multi_output": "deepseek-image"
    }
  },
  "updatedAt": {
    "deepseek": "2026-05-19T20:30:00+08:00"
  }
}
```

请求体既兼容旧格式，也支持新格式：

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
      "__model_thinking": "deepseek-reasoner",
      "__model_multi_input": "deepseek-vision",
      "__model_multi_output": "deepseek-image"
    }
  }
}
```

说明：

- `token` 仍然是唯一必填项，其余扩展字段都允许为空字符串
- 历史数据库会自动补齐新增列，不需要手工迁移
- `/api/token` 的写入日志仍然记录到 `proxy_agent_provider_log`

### `proxy token`

CLI 输出也升级为对象格式：

```bash
./proxy token
./proxy token --provider deepseek
```

- 不带 `--provider` 时，仍按模型名排序输出 JSON 数组
- 带 `--provider` 时，输出单个 JSON 对象
- 每个模型值现在都是包含 `token`、`__url`、`__model`、`__model_fast`、`__model_thinking`、`__model_multi_input`、`__model_multi_output` 的对象

示例：

```json
{"deepseek":{"token":"Bearer sk-deepseek","__url":"https://api.example.com/v1","__model":"deepseek-chat","__model_fast":"deepseek-fast","__model_thinking":"deepseek-reasoner","__model_multi_input":"deepseek-vision","__model_multi_output":"deepseek-image"}}
```
