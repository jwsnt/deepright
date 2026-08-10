# Proxy 迭代手册（20260513-4）

## 本次更新

- `/v1/chat/completions` 现在明确支持直接接收请求体中的 `metadata`
- 这些 `metadata` 字段会与共享 Agent 元数据合并后一起转发到上游
- 上游 `/v1/chat/completions` URL 不需要额外拼接 Query 开关

## 行为说明

请求：

```text
POST /v1/chat/completions
```

Proxy 实际转发到上游时使用：

```text
/v1/chat/completions
```

同时会把请求体补成类似：

```json
{
  "model": "gpt-4",
  "messages": [
    {
      "role": "user",
      "content": "hi"
    }
  ],
  "metadata": {
    "hello": "world",
    "extract": "true"
  }
}
```

## 字段规则

- 请求体中的 `metadata` 会直接写入转发请求
- 共享 Agent 元数据与请求体 `metadata` 合并时，请求体传入值优先
- `extract` 这类业务标记应直接由调用方写进请求体 `metadata`

## 使用示例

```bash
curl 'http://127.0.0.1:9876/v1/chat/completions' \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer sk-demo' \
  -d '{
    "model": "gpt-4",
    "messages": [
      {
        "role": "user",
        "content": "hello"
      }
    ],
    "metadata": {
      "hello": "world"
    }
  }'
```

## 典型场景

- 页面侧希望通过请求体 `metadata` 传递临时开关或来源标记
- `site` 中由 `Skill` 小图标触发的请求可以直接携带：

```json
{
  "metadata": {
    "extract": "true"
  }
}
```

最终由 Proxy 合并共享 Agent 元数据后继续转发。
