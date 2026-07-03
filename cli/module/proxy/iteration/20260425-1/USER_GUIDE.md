# Proxy 迭代 20260425-1 使用手册

## 变更说明

新增 `POST /api/config?agentId=xxx` 接口，创建或更新 Agent 的 `config.json`。

## 接口说明

| 方法 | 路径 | 参数 | 说明 |
|------|------|------|------|
| POST | `/api/config` | `agentId`（query） | 创建或更新 config.json |

## 请求体

```json
{"description": "Agent 描述", "thinking": true, "swarm": true}
```

- `description`：不超过 200 字
- `thinking`：布尔值
- `swarm`：布尔值

## 响应

成功：`{"status": 0, "agentId": "a"}`
失败：`{"status": 1, "content": "错误提示"}`
