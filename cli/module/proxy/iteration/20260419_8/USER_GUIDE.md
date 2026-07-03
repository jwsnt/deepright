# Proxy 迭代 20260419_8 使用手册

## 变更说明

新增 `POST /api/edit?agentId=xxx&path=yyy` 接口，在指定 Agent 的 workspace 下写入文件内容。

## 接口说明

| 方法 | 路径 | 参数 | 说明 |
|------|------|------|------|
| POST | `/api/edit` | `agentId`（必填）、`path`（必填） | 写入文件内容 |

## 请求

- `agentId`：Agent ID（query 参数）
- `path`：相对于 Agent workspace 的相对路径（query 参数，不区分大小写）
- `path`：支持包含空格的路径
- Body：`{"content": "文件内容"}`

## 响应格式

成功：
```json
{"agentId": "a", "path": "b/c/user.md", "status": 0}
```

失败：
```json
{"agentId": "a", "path": "b/c/user.md", "content": "错误提示", "status": 1}
```

## 访问限制

- 只支持相对路径（拒绝绝对路径和 `~` 路径）
- 拒绝使用 `..` 跳出 workspace
- 拒绝写入目录
- 目标文件不存在时自动创建
- 二进制文件（图片、多媒体等）使用 base64 内容写入
- 父目录不存在时自动创建
