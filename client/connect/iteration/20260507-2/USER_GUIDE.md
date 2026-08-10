# 20260507-2 使用手册

## 目标

本次迭代为 `connect` 增加按插件主键读取配置的 `meta-get` 能力。

最终用户验收入口优先使用 `integration` 顶层命令；`connect meta-get` 主要用于内部实现或兼容说明。

## 推荐用法

在 `/path/to/deepright/cli/module/integration` 目录执行：

```bash
./integration connect meta-get --key feishu
```

如果需要兼容旧的原始元数据读取方式，也可以继续使用：

```bash
./integration connect meta-get --name 飞书
```

## Connect 兼容入口

在 `/path/to/deepright/cli/module/connect` 目录执行：

```bash
./connect meta-get --key feishu
./connect meta-get --name 飞书
```

说明：

- `--key` 返回插件配置视图，适合插件按稳定主键读取运行时配置
- `--name` 返回原始 connect 元数据记录，主要用于调试和兼容旧流程

## 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--key` | 否 | 插件运行时主键；本次迭代新增，推荐优先使用 |
| `--name` | 否 | 连接名称；保留作兼容输入 |
| `--include-deleted` | 否 | 是否包含已删除数据，默认 `false` |

补充说明：

- `meta-get` 至少应提供 `--key` 或 `--name` 其中一个
- 当同时提供 `--key` 与 `--name` 时，优先按 `--key` 读取

## 返回示例

`--key` 读取插件配置时，返回示例：

```json
{
  "key": "feishu",
  "name": "飞书",
  "meta": {
    "appId": "cli-app",
    "appSecret": "cli-secret"
  },
  "stream": true,
  "callback": "/abs/path/plugins/feishu",
  "agentId": "A",
  "chatId": "chat-001",
  "model": "OpenAI",
  "thinking": true,
  "createdAt": "2026-05-05T10:00:00+08:00",
  "updatedAt": "2026-05-05T10:00:00+08:00"
}
```

`--name` 读取原始元数据时，返回示例：

```json
{
  "id": 1,
  "name": "feishu",
  "meta": "{\"appId\":\"cli-app\",\"appSecret\":\"cli-secret\"}",
  "stream": true,
  "callback": "/abs/path/plugins/feishu",
  "agentId": "A",
  "chatId": "chat-001",
  "model": "OpenAI",
  "thinking": true,
  "createdAt": "2026-05-05T10:00:00+08:00",
  "updatedAt": "2026-05-05T10:00:00+08:00"
}
```

## 行为说明

- `integration connect meta-get --key ...` 与 `connect meta-get --key ...` 都会返回单个插件的已配置运行时视图
- 插件配置视图中的 `meta` 会被解析为 JSON 对象，便于插件直接消费
- `integration` 进程会复用启动时初始化的共享 `connect` 服务与 SQLite 连接，不会为每次请求单独开关数据库
- `connect` 服务自身也会按数据库路径复用连接池，避免同一路径重复打开 SQLite
- `connect meta-get --name ...` 与旧链路保持兼容，不影响现有调试方式

## 关联命令

- `./integration connect meta-list`：查看全部已配置插件
- `./integration meta-list`：查看原始 connect 元数据列表
- `./integration plugins config`：写入或更新插件配置后，可再用 `meta-get --key` 校验结果
