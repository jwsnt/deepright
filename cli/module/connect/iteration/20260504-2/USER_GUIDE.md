# 20260504-2 使用手册

## 目标

本次迭代为 `connect` 新增 `list-meta` 命令，用于读取当前所有已配置插件的 meta 配置。

## 命令

```bash
./integration connect meta-list
```

如果服务不在默认地址，也可以指定：

```bash
./integration connect meta-list --addr http://127.0.0.1:18081
./integration connect meta-list --port 18081
```

## 返回格式

示例：

```json
[
  {
    "name": "feishu",
    "meta": {
      "appId": "cli-app",
      "appSecret": "cli-secret"
    },
    "stream": true,
    "callback": "./feishu",
    "agentId": "A",
    "chatId": "chat-001",
    "model": "OpenAI",
    "thinking": true,
    "createdAt": "2026-05-05T10:00:00+08:00",
    "updatedAt": "2026-05-05T10:00:00+08:00"
  }
]
```

## 行为说明

- 最终用户主流程优先通过 `integration connect meta-list` 读取数据
- 如果服务没有启动，命令会直接报错
- 只返回当前有效的已配置插件元数据
- 返回中的 `meta` 字段不是字符串，而是已经解析后的 JSON 对象
- 原有 `connect list-meta` / `meta-list` 命令继续保留，用于兼容或查看原始数据库记录

## 兼容说明

- HTTP 接口 `GET /api/connect/meta?view=config` 返回与 `list-meta` 相同的结构
- `integration` 中内嵌的 `connect` 子命令也同步支持 `list-meta`
