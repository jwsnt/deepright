# 20260507-1 使用手册

## 目标

本次迭代为 `connect` 增加 `add-request` 命令，用于保存三方请求数据。

最终用户验收入口优先使用 `integration` 顶层命令；`connect add-request` 主要用于内部实现或兼容说明。

## 推荐用法

在 `/path/to/deepright/cli/module/integration` 目录执行：

```bash
./integration connect add-request --key feishu --externalId msg-1 --content "HELLO WORLD"
./integration connect add-request --key feishu --content "HELLO WORLD" --artifacts "/tmp/a.txt,/tmp/b.txt" --original '{"text":"HELLO WORLD"}'
./integration connect add-request --key feishu --content "HELLO WORLD" --status 1 --created 1777852800
```

如果服务不在默认地址，也可以显式指定：

```bash
./integration connect add-request --addr http://127.0.0.1:18081 --key feishu --content "HELLO WORLD"
./integration connect add-request --port 18081 --key feishu --content "HELLO WORLD"
```

## 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--key` / `--name` | 是 | 连接主键；推荐优先使用 `--key` |
| `--externalId` / `--external-id` | 否 | 三方外部消息 ID；与 `key` 组成唯一键 |
| `--content` / `--request` | 是 | 请求内容；推荐优先使用 `--content` |
| `--artifacts` | 否 | 逗号分隔的附件路径 |
| `--original` / `--raw-request` | 否 | 原始请求内容；推荐优先使用 `--original` |
| `--status` | 否 | 请求状态：`0=待处理`、`1=已启动`、`2=已完成`、`3=已过期`、`4=已回复` |
| `--created` | 否 | 创建时间，支持 Unix 时间戳或 RFC3339 |

## 返回格式

示例：

```json
{
  "id": 1,
  "key": "feishu",
  "name": "feishu",
  "externalId": "msg-1",
  "content": "HELLO WORLD",
  "request": "HELLO WORLD",
  "artifacts": "/tmp/a.txt,/tmp/b.txt",
  "original": "{\"text\":\"HELLO WORLD\"}",
  "rawRequest": "{\"text\":\"HELLO WORLD\"}",
  "status": 0,
  "createdAt": "2026-05-02T00:00:00Z"
}
```

## 行为说明

- `integration connect add-request` 和 `connect add-request` 都会通过本地 HTTP 服务写入，不会直接操作数据库
- 如果 `connect` 服务未启动，命令会直接报错
- 请求会复用启动时初始化的全局数据库连接，不会为单次命令单独开关 SQLite
- 未传 `--status` 时默认写入 `0`
- 未传 `--created` 时默认写入当前时间
- 如果传入 `externalId`，同一 `key + externalId` 不允许重复写入

## 兼容说明

- `connect add-request` 同步支持相同参数
- `request-list` / `add-response` / `response-list` 可继续用于后续查询与回写链路
- 飞书、邮件等插件后续推送请求时，也统一走这套 `add-request` 入口
