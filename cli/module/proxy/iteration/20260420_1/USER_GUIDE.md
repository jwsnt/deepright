# Proxy 迭代 20260420_1 使用手册

## 变更说明

新增 `GET /api/agent/init?name=xxx` 接口，在 Agent 目录下创建新 Agent。

## 接口说明

| 方法 | 路径 | 参数 | 说明 |
|------|------|------|------|
| GET | `/api/agent/init` | `name`（必填） | 创建新 Agent 目录 |

## 行为

1. 验证 name（不含空格和特殊字符）
2. 在 Agent 根目录下创建 `name` 文件夹
3. 自动创建空的 `SOUL.md` 和 `USER.md`
4. 立即刷新 Agent 缓存

## 响应

成功：
```json
{"status": 0, "name": "my-agent", "path": "/absolute/path/my-agent"}
```

失败：
```json
{"status": 1, "content": "错误提示"}
```
