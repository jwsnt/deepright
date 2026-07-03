# Proxy 迭代 20260419_3 使用手册

## 变更说明

新增 `GET /api/folder?agentId=xxx` 接口，根据 agentId 查找对应 Agent 的 workspace 绝对路径，调用系统命令打开目录。

## 接口说明

| 方法 | 路径 | 参数 | 说明 |
|------|------|------|------|
| GET | `/api/folder` | `agentId`（必填） | 打开指定 Agent 的 workspace 目录 |

## 系统命令

| 系统 | 命令 |
|------|------|
| macOS | `open <path>` |
| Linux | `xdg-open <path>` |
| Windows | `explorer <path>` |

## 响应

成功：
```json
{"status": "ok", "path": "/absolute/path/to/workspace"}
```

错误：
- 400：缺少 agentId 参数
- 404：agentId 不存在
- 500：打开目录失败

## 子模块调用

```go
err := OpenFolder("/path/to/workspace")
```
