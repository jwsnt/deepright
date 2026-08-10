本次迭代为 `proxy token` 增加了本地 Token 用量查询能力，并保留原有模型密钥读取与消费写入行为不变。

## 查询入口

- 兼容顶层查询写法：

```bash
proxy token --n 500
proxy token --n 500 --start "2026-06-14 12:00:00" --close "2026-06-14 14:00:00"
```

- 新增独立子命令：

```bash
proxy token get --n 500
proxy token get --n 500 --start "2026-06-14 12:00:00" --close "2026-06-14 14:00:00"
proxy token get --help
```

- 可选参数：
  - `--agentId` / `--agent`：仅查询指定 AgentId
  - `--n`：最近 N 条，默认 `500`
  - `--start`：开始时间
  - `--close`：结束时间，默认当前时间

## 时间格式

- `--start` 与 `--close` 支持两种格式：
  - `yyyyMMdd-hhmmss`
  - `YYYY-MM-DD HH:MM:SS`

示例：

```bash
proxy token get --n 100 --start "20260614-120000" --close "20260614-140000"
proxy token get --n 100 --start "2026-06-14 12:00:00" --close "2026-06-14 14:00:00"
```

## 输出结构

- 查询输出与 `/api/consume` 保持同一套数据结构：
  - `status`
  - `details`
  - `summary`
- `details` 为命中的消费明细
- `summary` 为按 `model` 聚合后的 `thinking`、`input`、`total`、`cache`
- CLI 在“最近 N 条”模式下会先取最新记录，再按时间升序输出，便于阅读

示例输出：

```json
{
  "status": 0,
  "details": [
    {
      "thinking": 11,
      "input": 21,
      "total": 31,
      "cache": 2,
      "model": "deepseek-chat",
      "agentId": "demo-agent",
      "function": "cli/get",
      "timestamp": 1781409720000
    }
  ],
  "summary": [
    {
      "model": "deepseek-chat",
      "thinking": 11,
      "input": 21,
      "total": 31,
      "cache": 2
    }
  ]
}
```

## 兼容性

- 以下旧命令保持不变：

```bash
proxy token
proxy token --provider deepseek
proxy token --agentId demo-agent --model deepseek-chat --function cli/get --thinking 10 --input 20 --total 30 --cache 5
```

- 只有传入 `--n`、`--start`、`--close`，或显式使用 `token get` 时，才会进入本地 token 用量查询模式
