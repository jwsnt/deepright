# Proxy 迭代 20260620-1 使用手册

## 变更说明

- `POST /api/config?agentId=xxx` 新增 `media` 字段，写入 Agent 工作目录下的 `config.json`
- `media` 是 Agent 维度的 JSON 对象；Site 侧会按 `模型服务商名 -> 多组参数` 的结构写入
- `POST /api/edit?agentId=xxx&path=config.json` 的请求体兼容额外的 `media` 字段，方便前端在回写完整 `config.json` 时带上同一份结构
- `proxy` 转发 `/v1/chat/completions` 时，如果某个 Agent 的 `config.json.media` 非空，则会在请求体 `metadata.agents[]` 中把该 Agent 对应项补上 `media`
- 同一次转发里，`metadata.agent` 也会补上当前选中 Agent 的 `media`
- `media` 每次转发前都会直接重新读取对应 Agent 的最新 `config.json`，不等待 Agent metadata cache 过期

## `/api/config`

请求示例：

```json
{
  "description": "demo",
  "thinking": true,
  "router_disable": false,
  "media": {
    "gemini": {
      "aspectRatio": "16:9",
      "imageSize": "2K"
    }
  }
}
```

保存后，Agent 工作目录下的 `config.json` 会包含同名 `media` 对象。

如果前端通过 `/api/edit` 直接回写完整 `config.json`，请求体可以写成：

```json
{
  "content": "{\n  \"media\": {\n    \"gemini\": {\n      \"aspectRatio\": \"16:9\",\n      \"imageSize\": \"2K\"\n    }\n  }\n}",
  "media": {
    "gemini": {
      "aspectRatio": "16:9",
      "imageSize": "2K"
    }
  }
}
```

## `/v1/chat/completions` 转发

当 Agent `A` 的 `config.json` 中存在：

```json
{
  "media": {
    "gemini": {
      "aspectRatio": "16:9",
      "imageSize": "2K"
    }
  }
}
```

则 `proxy` 转发时会在 `metadata` 中补充：

```json
{
  "agents": [
    {
      "agentId": "A",
      "media": {
        "gemini": {
          "aspectRatio": "16:9",
          "imageSize": "2K"
        }
      }
    }
  ],
  "agent": {
    "agentId": "A",
    "media": {
      "gemini": {
        "aspectRatio": "16:9",
        "imageSize": "2K"
      }
    }
  }
}
```

## 缓存说明

- `version`、`skills` 等仍沿用原有 Agent metadata cache 机制
- `media` 不走该缓存；每次真正转发前都会直接读取对应 Agent 的最新 `config.json`
