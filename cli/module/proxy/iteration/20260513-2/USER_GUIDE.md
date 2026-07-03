# Proxy 迭代手册（20260513-2）

## 本次更新

- 新增 `GET /log_skill?agentId=xxx&chatId=yyy&round=zzz&start=aaa&close=bbb`
- 新增 `proxy log-skill --agent ... --chat ... --round ... --start ... --close ...`
- 按最近 N 轮 `/v1/chat/completions` 请求为边界，并叠加时间范围过滤导出日志
- 导出数据包含：
  - `/v1/chat/completions` 请求
  - 所有 SSE 响应
  - `cli/get`
  - `cli/pub`
- 导出时会把同一轮中的多段 SSE 响应合并为一条 Markdown 记录
- 导出文件写入对应 Agent 工作目录下的 `tmp/`
- 返回结果包含导出文件绝对路径与文件大小（`K`）

## HTTP 用法

```bash
curl 'http://127.0.0.1:9876/log_skill?agentId=A&chatId=chat-001&round=1'
curl 'http://127.0.0.1:9876/log_skill?agentId=A&chatId=chat-001&round=3'
curl 'http://127.0.0.1:9876/log_skill?agentId=A&chatId=chat-001&start=2026-05-13%2012:00:00&close=2026-05-13%2012:10:00'
```

返回示例：

```json
{
  "status": 0,
  "path": "/abs/agents/A/tmp/A_chat-001_20260513121009.md",
  "sizeK": 1.23
}
```

## CLI 用法

```bash
./proxy log-skill --agent A --chat chat-001 --round 3
./proxy log-skill --agent A --chat chat-001 --start '2026-05-13 12:00:00' --close '2026-05-13 12:10:00'
```

可选参数：

- `--agent-dir`
- `--device`
- `--agent-cache`

## 查询条件说明

- `round`：最近 N 轮，以 `/v1/chat/completions` 请求为标记点，默认 `1`
- `start`：日志开始时间，默认空
- `close`：日志结束时间，默认空
- 三个条件按 AND 同时生效
- `round` 和 `start` 至少提供一个
- 仅传 `start` / `start + close` 时，会按纯时间范围查询，不会额外强制套最近一轮
- 只有传入 `round` 时，才会先按最近 N 轮收缩范围，再继续叠加时间过滤
- 如果轮次不足，则先从第一轮开始，再继续叠加时间过滤

## 文件格式

- 导出文件名由 `agentId + chatId + 时间戳` 组合，例如 `agenta_chatb_20260513121009.md`
- 文件内容为 Markdown 表格
- 表头固定为：

```md
| 时间 | 类型 | 具体内容 |
| --- | --- | --- |
```

- 时间格式：`yyyy-MM-dd HH:mm:ss`
- 类型显示：
  - `SSE请求`
  - `SSE响应`
  - `工具请求（cli/get）`
  - `工具响应（cli/pub）`
