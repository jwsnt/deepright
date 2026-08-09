本次迭代为 `integration token` 增加了本地 Token 用量查询能力，并保持原有模型密钥读取与消费写入模式兼容。

## 查询入口

- 兼容原需求中的顶层写法：

```bash
integration token --n 500
integration token --n 500 --start "2026-06-14 12:00:00" --close "2026-06-14 14:00:00"
```

- 新增独立子命令：

```bash
integration token get --n 500
integration token get --n 500 --start "2026-06-14 12:00:00" --close "2026-06-14 14:00:00"
integration token get --help
```

- 可选参数：
  - `--agentId` / `--agent`：仅查询指定 AgentId
  - `--n`：最近 N 条，默认 `500`
  - `--start`：开始时间
  - `--close`：结束时间，默认当前时间

## 数据来源

- 查询读取的是本地 SQLite 中 `token_consume_log` 表里的 Token 用量数据
- 聚合与过滤规则复用 `/api/consume` 的同一套底层查询逻辑
- CLI 在“最近 N 条”模式下会先取最新记录，再按时间升序输出，便于直接查看时间线

## 时间格式

- `--start` 与 `--close` 支持：
  - `yyyyMMdd-hhmmss`
  - `YYYY-MM-DD HH:MM:SS`

示例：

```bash
integration token get --n 100 --start "20260614-120000" --close "20260614-140000"
integration token get --n 100 --start "2026-06-14 12:00:00" --close "2026-06-14 14:00:00"
```

## 输出结构

- 查询输出与 HTTP `/api/consume` 一致：
  - `status`
  - `details`
  - `summary`

示例输出：

```json
{
  "status": 0,
  "details": [
    {
      "thinking": 12,
      "input": 24,
      "total": 36,
      "cache": 6,
      "model": "deepseek-chat",
      "agentId": "demo-agent",
      "function": "cli/get",
      "timestamp": 1781409720000
    }
  ],
  "summary": [
    {
      "model": "deepseek-chat",
      "thinking": 12,
      "input": 24,
      "total": 36,
      "cache": 6
    }
  ]
}
```

## 兼容性

- 以下旧命令保持不变：

```bash
integration token
integration token --provider deepseek
integration token --agentId demo-agent --model deepseek-chat --function cli/get --thinking 10 --input 20 --total 30 --cache 5
```

- 只有传入 `--n`、`--start`、`--close`，或显式使用 `token get` 时，才进入本地 token 用量查询模式
