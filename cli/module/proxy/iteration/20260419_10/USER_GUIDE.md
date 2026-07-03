# Proxy 迭代 20260419_10 使用手册

## 变更说明

新增 `GET /api/raw?agentId=xxx&path=yyy` 接口，读取文件二进制内容并以 base64 编码返回。

该接口同时支持两类路径：

- 相对于 Agent workspace 的相对路径
- 文件系统绝对路径（含大小写不敏感匹配）

## 接口说明

| 方法 | 路径 | 参数 | 说明 |
|------|------|------|------|
| GET | `/api/raw` | `agentId`（必填）、`path`（必填） | 读取文件二进制流，base64 返回 |

参数说明：

- `agentId`：目标 Agent 标识
- `path`：支持相对路径或绝对路径；相对路径以对应 Agent 的 workspace 为根目录解析

支持场景：

- `path=b/c/user.md`
- `path=/a/b/c`
- `path=tmp/My File.png`
- `path=/Users/name/Test Dir/demo.pdf`
- `path=~/demo.txt` 不支持

## 响应格式

成功：
```json
{"agentId": "a", "path": "b/c/user.md", "content": "SEVMTE8gV09STEQ=", "status": 0}
```

绝对路径成功：
```json
{"agentId": "a", "path": "/a/b/c", "content": "SEVMTE8gV09STEQ=", "status": 0}
```

失败：
```json
{"agentId": "a", "path": "b/c/user.md", "content": "错误提示", "status": 1}
```

## 限制

- 路径不区分大小写
- 相对路径仅允许落在 Agent workspace 内
- 相对路径拒绝 `..` 越界
- 绝对路径会直接按文件系统路径读取
- `~` 路径不支持
- 文件不存在或为目录时返回 status=1

## 常见失败

- 文件不存在：返回 `status=1`
- 路径指向目录：返回 `status=1`
- 相对路径越出 workspace：返回 `status=1`
