本次迭代为 Proxy 的 `/v1/chat/completions` 转发补充了模型路由配置透传能力。

### 转发 metadata 自动补充

当请求体中的 `model` 命中 `token_store` 里对应的模型配置时，Proxy 会在转发给上游的请求体 `metadata` 中自动补充以下非空字段：

- `__url`
- `__model`
- `__model_fast`
- `__model_thinking`
- `__model_multi_input`
- `__model_multi_output`

这些字段来源于当前命中模型本身的配置，不会跨模型读取。

示例：模型 `deepseek` 只配置了 `__url` 与 `__model_fast`

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

- 只有当前 `model` 对应配置里的非空字段才会写入 `metadata`
- 字段未配置或值为空字符串时，不会追加到 `metadata`
- 现有 Agent 元数据注入、请求体 `metadata` 合并、知识库字段处理逻辑保持不变
