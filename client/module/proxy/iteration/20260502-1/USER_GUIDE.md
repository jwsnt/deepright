# Proxy 迭代 20260502-1 使用手册

## 变更说明

本迭代扩展了 `POST /api/edit?agentId=xxx&path=yyy` 的写入能力：

- `/api/edit` 现在支持二进制文件类型（图片、多媒体等）编辑
- 如果目标文件不存在，则自动创建并保存

## 接口说明

| 方法 | 路径 | 参数 | 说明 |
|------|------|------|------|
| POST | `/api/edit` | `agentId`（必填）、`path`（必填） | 在指定 Agent 的 workspace 下写入文件 |

## 请求

- `agentId`：Agent ID（query 参数）
- `path`：相对于 Agent workspace 的相对路径（query 参数）
- Body：
  - 文本文件：`{"content": "文件内容"}`
  - 二进制文件：`{"content": "base64编码后的内容"}`

## 写入规则

- 只支持 Agent workspace 下的相对路径
- 目标父目录不存在时自动创建
- 目标文件不存在时自动创建
- `saveAsNew=true` 时，会在原目录基于原文件名追加时间戳后写入新文件，不覆盖原文件
- 二进制文件（图片、多媒体等）按 base64 内容写入
- 文本文件按普通字符串写入

## 访问限制

- 拒绝绝对路径和 `~` 路径
- 拒绝使用 `..` 跳出 workspace
- 拒绝写入目录

## 响应格式

成功：

```json
{"agentId":"a","path":"media/movie.mp4","savedAs":"/abs/workspace/media/movie_20260502_101530.mp4","status":0}
```

失败：

```json
{"agentId":"a","path":"../escape.txt","content":"只支持 workspace 下的相对路径","status":1}
```
