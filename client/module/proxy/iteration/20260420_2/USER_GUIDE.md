# Proxy 迭代 20260420_2 使用手册

## 变更说明

新增 `GET /api/agent/delete?name=xxx` 接口，删除 Agent 目录下指定文件夹。

## 接口说明

| 方法 | 路径 | 参数 | 说明 |
|------|------|------|------|
| GET | `/api/agent/delete` | `name`（必填） | 删除指定 Agent 目录（递归） |

## 响应

成功：
```json
{"status": 0, "name": "my-agent"}
```

失败：
```json
{"status": 1, "content": "错误提示"}
```

## 行为

- 验证 name 合法性（无空格和特殊字符）
- 递归删除 Agent 目录及所有内容
- 删除后立即刷新 Agent 缓存
