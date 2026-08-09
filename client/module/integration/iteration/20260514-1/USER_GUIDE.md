# Integration 迭代手册（20260514-1）

## 本次收口

- `integration` 复用 `proxy` 的 `/v1/chat/completions` metadata 收口行为
- 请求体中的 `metadata` 会与共享 Agent 元数据合并后一起转发到上游
- 这样最终交付给用户的 `integration` 单二进制与 `proxy` 行为保持一致

## 行为说明

请求：

```text
POST /v1/chat/completions
```

`integration` 转发到上游时使用：

```text
/v1/chat/completions
```

同时会把请求体中的 metadata 合并进转发请求：

```json
{
  "metadata": {
    "hello": "world",
    "extract": "true"
  }
}
```

## 兼容规则

- 请求体中已有同名 `metadata` 字段时，保留请求体传入值
- 该行为只影响 `integration` 对外收口的 `/v1/chat/completions`

## 示例

```bash
curl 'http://127.0.0.1:8080/v1/chat/completions' \
  -H 'Content-Type: application/json' \
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

最终上游收到的 URL 为：

```text
http://deepright.cn/v1/chat/completions
```

最终上游收到的请求体中包含：

```json
{
  "metadata": {
    "hello": "world"
  }
}
```

## 与 Site 的关系

- `site` 中由 `Skill` 小图标触发的请求会发起：

```json
{
  "metadata": {
    "extract": "true"
  }
}
```

- `integration` 会把它自动转换为：

```json
{
  "metadata": {
    "extract": "true"
  }
}
```

- 因此前端只需要直接填写请求体 `metadata`，不需要再依赖 Query 临时开关
