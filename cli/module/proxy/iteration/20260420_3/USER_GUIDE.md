# Proxy 迭代 20260420_3 使用手册

## 变更说明

新增 `GET /api/agent/create?agentId=xxx&name=yyy&type=zzz` 接口，在指定 Agent 目录下创建文件或文件夹。

## 接口说明

| 方法 | 路径 | 参数 | 说明 |
|------|------|------|------|
| GET | `/api/agent/create` | `agentId`、`name`、`type` | 创建文件或文件夹 |

## 参数

- `agentId`：Agent ID
- `name`：文件或文件夹名称（无空格和特殊字符）
- `type`：`0` = 文件夹，`1` = 文件

## 响应

成功：
```json
{"status": 0, "agentId": "a", "name": "b", "type": "1"}
```

失败：
```json
{"status": 1, "content": "错误提示"}
```
