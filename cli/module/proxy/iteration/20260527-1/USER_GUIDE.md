本次迭代为 `proxy` 增加了 Token 消费明细记录与查询能力。

## CLI

- 继续保留原有 `proxy token` 查询模型密钥的能力
- 新增写入模式，传入消费字段后会直接写入：

```bash
proxy token \
  --agentId demo-agent \
  --model deepseek-chat \
  --function cli/get \
  --thinking 10 \
  --input 20 \
  --total 30 \
  --cache 5
```

- `--timestamp` 可选，单位为毫秒；不传时默认写入当前时间
- 历史 `--record` 仍兼容，但已不是必需参数
- 写入成功后会返回 JSON，包含刚保存的明细
- 如果是在最终交付的 `integration` 二进制中使用这项能力，需要写成 `integration token ...`；不带 `token` 子命令会进入默认服务模式

## HTTP

### `GET /api/consume`

- 参数：
  - `starTime`：必填，格式固定为 `yyyyMMdd-hhmmss`
  - `closeTime`：必填，格式固定为 `yyyyMMdd-hhmmss`
  - `agentId`：选填；不传时查询所有 Agent
  - `limit`：选填；不传默认 `500`，表示最多返回的明细条数
- 返回两部分：
  - `details`：所有明细记录
  - `summary`：按 `model` 聚合后的 `thinking`、`input`、`total`、`cache`

示例：

```bash
curl "http://127.0.0.1:9876/api/consume?agentId=demo-agent&starTime=20260527-090000&closeTime=20260527-180000&limit=500"
```

## 存储

- 使用共享 `data` sqlite
- 明细字段：
  - `thinking`
  - `input`
  - `total`
  - `cache`
  - `model`
  - `agentId`
  - `function`
  - `timestamp`
- 新增索引：时间戳 + AgentId
