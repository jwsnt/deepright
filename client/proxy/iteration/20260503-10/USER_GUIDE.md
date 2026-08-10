# 20260503-10 使用手册

## 目标

本次迭代为 `proxy` 新增了插件日志流接口：

- 新增 `GET /api/plugins/log`
- 新增 `proxy plugins log` CLI
- 底层按“插件同名 `.log` 文件”定位日志，并持续以 SSE 输出
- `integration` 已同步暴露同一路径

## HTTP 接口

请求：

```text
GET /api/plugins/log?name=feishu&last=10
```

参数：

- `name`：必填，支持插件名、插件路径或日志路径
- `last`：可选，启动时先回放最后 N 行，默认 `10`

响应示例：

```text
event: log
data: 2026-05-05 10:00:00,收到消息 A

event: log
data: 2026-05-05 10:00:03,收到消息 B

event: error
data: log file not found: /abs/path/plugins/feishu.log
```

## CLI

```bash
./proxy plugins log --plugin feishu --last 20
```

说明：

- `--plugin feishu` 默认解析到 `../plugins/feishu.log`
- 也可以直接传 `--plugin ../plugins/feishu` 或 `--plugin ../plugins/feishu.log`
- CLI 会直接把 SSE 事件输出到标准输出，适合其他模块复用

## 行为规则

- 插件日志文件命名规则固定为“插件同名 `.log` 文件”
- 启动连接时会先读取最后 N 行，再持续输出新增内容
- 行分隔规则与普通文本日志一致，每一行对应一个 `event: log`
- 用户主动断开连接时，服务端立即停止读取
- 文件不存在或读取过程中被删除时，会先推送 `event: error` 再关闭连接
- 为兼容旧调用，当前也接受 `plugin` 查询参数，但本次需求约定统一使用 `name`

## 同步结果

- `proxy` 主手册已补充 `/api/plugins/log` 与 `proxy plugins log`
- `integration` 主手册已补充 `/api/plugins/log`
- `integration` 集成日志已追加本次同步记录
