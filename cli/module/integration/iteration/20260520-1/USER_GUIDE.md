本次迭代将 Integration 收口的 `/v1/chat/completions` 转发与 Proxy 保持一致，自动透传模型扩展配置到请求 `metadata`。

### 转发 metadata 自动补充

当请求体中的 `model` 命中 `token_store` 中保存的模型配置时，Integration 会把该模型下已配置且非空的以下字段写入转发请求的 `metadata`：

- `__url`
- `__model`
- `__model_fast`
- `__model_thinking`
- `__model_multi_input`
- `__model_multi_output`

示例：`deepseek` 只配置了 `__url` 与 `__model_fast`

```json
{
  "model": "deepseek",
  "messages": [
    {
      "role": "user",
      "content": "hello"
    }
  ],
  "metadata": {
    "hello": "world"
  }
}
```

转发到上游后的请求体片段：

```json
{
  "metadata": {
    "hello": "world",
    "__url": "https://provider.example/v1",
    "__model_fast": "deepseek-fast",
    "__model_multi_input": "deepseek-vision"
  }
}
```

说明：

- 只读取当前 `model` 命中的那一条模型配置
- 未配置或为空字符串的字段不会出现在转发 `metadata` 中
- 其他 metadata 注入逻辑继续保持原行为不变
