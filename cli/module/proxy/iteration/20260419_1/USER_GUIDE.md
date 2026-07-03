# Proxy 迭代 20260419 — AgentId 列表接口

## 新增功能

新增 HTTP GET 接口 `/api/agentId`，返回所有 Agent ID 的 JSON 数组。

## 接口说明

```
GET /api/agentId
```

### 响应

```json
["a", "b"]
```

Content-Type: `application/json`

### 示例

```bash
curl http://127.0.0.1:9876/api/agentId
# ["a","b"]
```

## 子模块调用

```go
mux.HandleFunc("/api/agentId", proxy.HandleAgentIDs)
```

`HandleAgentIDs` 从缓存的 Agent 元数据中提取所有 agentId，返回 JSON 数组。

## 注意事项

- 仅支持 GET 方法，其他方法返回 405
- 共享现有的 Agent 元数据缓存，不额外扫描
- 原有 `/v1/chat/completions` 和 `/site/` 功能不受影响
