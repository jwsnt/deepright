# Proxy 迭代 20260419_7 使用手册

## 变更说明

新增 `GET /api/workspace?agentId=xxx` 接口，返回指定 Agent 的 workspace 绝对路径。

## 接口说明

| 方法 | 路径 | 参数 | 说明 |
|------|------|------|------|
| GET | `/api/workspace` | `agentId`（必填） | 返回 Agent 的 workspace 绝对路径（text/plain） |

## 响应示例

```
/Users/user/agents/a
```

## 错误码

- 400：缺少 agentId 参数
- 404：agentId 不存在
- 500：Agent 元数据扫描失败
