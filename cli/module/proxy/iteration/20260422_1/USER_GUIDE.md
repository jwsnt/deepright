# Proxy 迭代 20260422_1 使用手册

## 变更说明

新增 `POST /api/upload?agentId=xxx` 接口，上传文件或文件夹到 Agent 目录下的 `tmp`。

## 接口说明

| 方法 | 路径 | 参数 | 说明 |
|------|------|------|------|
| POST | `/api/upload` | `agentId`（query） | 上传文件到 agent/tmp |

## 请求格式

- Content-Type: `multipart/form-data`
- 支持单文件和文件夹上传（HTML `webkitdirectory` 属性）
- 文件夹上传时保留相对路径结构
- 最大 100MB

## 响应

成功：
```json
{"status": 0, "agentId": "a", "files": ["file1.txt", "dir/file2.txt"], "dest": "/path/to/agent/tmp"}
```

失败：
```json
{"status": 1, "content": "错误提示"}
```
