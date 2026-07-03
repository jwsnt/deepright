# Proxy 迭代 20260419_4 使用手册

## 变更说明

新增 `GET /api/skills?agentId=xxx` 接口，返回指定 Agent 的 Skill 名称列表。

## 接口说明

| 方法 | 路径 | 参数 | 说明 |
|------|------|------|------|
| GET | `/api/skills` | `agentId`（必填） | 返回指定 Agent 的 Skill 名称 JSON 数组 |

## 响应示例

```json
["__internal_A", "__internal_F"]
```

## 错误码

- 400：缺少 agentId 参数
- 404：agentId 不存在
- 500：Agent 元数据扫描失败
