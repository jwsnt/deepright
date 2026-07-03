# Proxy 迭代 20260419_9 使用手册

## 变更说明

新增 `GET /api/del?agentId=xxx&path=yyy` 接口，删除指定 Agent workspace 下的文件或目录。

## 接口说明

| 方法 | 路径 | 参数 | 说明 |
|------|------|------|------|
| GET | `/api/del` | `agentId`（必填）、`path`（必填） | 删除文件或目录 |

## 删除行为

- 文件：直接删除
- 目录：递归删除子孙目录及文件后删除该目录

## 响应格式

成功：
```json
{"agentId": "a", "path": "b/c/user.md", "status": 0}
```

失败：
```json
{"agentId": "a", "path": "b/c", "content": "错误提示", "status": 1}
```

## 访问限制

- 只支持相对路径（拒绝绝对路径和 `~` 路径）
- 路径不区分大小写
