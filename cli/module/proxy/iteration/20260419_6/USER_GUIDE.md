# Proxy 迭代 20260419_6 使用手册

## 变更说明

新增 `GET /api/data?path=xxx` 接口，返回指定文件内容，JSON 格式响应。

## 接口说明

| 方法 | 路径 | 参数 | 说明 |
|------|------|------|------|
| GET | `/api/data` | `path`（必填） | 返回指定文件内容 |

## path 支持

- 绝对路径、`~` 路径、含空格路径
- 不区分大小写（逐段匹配）

## 响应格式

成功（status=0）：
```json
{"path": "/absolute/path/file.md", "content": "文件内容", "status": 0}
```

失败（status=1）：
```json
{"path": "/absolute/path", "content": "错误提示", "status": 1}
```

## 访问限制

以下情况返回 status=1：
- path 指向目录
- path 指向二进制文件（图片、多媒体、压缩包、字体等）
- 文件不存在或无法读取
