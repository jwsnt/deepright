# 20260527-1 使用说明

本次迭代把 Token 消费明细能力收口到 `integration`，既可以通过 CLI 写入，也可以通过 HTTP 查询。

## CLI

- 原有 `integration token` 查询模型密钥的行为保持不变
- 新增写入模式，传入消费字段后会直接写入：

```bash
integration token \
  --agentId demo-agent \
  --model deepseek-chat \
  --function cli/get \
  --thinking 10 \
  --input 20 \
  --total 30 \
  --cache 5
```

- `--timestamp` 可选，单位为毫秒；不传时默认使用当前时间
- 历史 `--record` 仍兼容，但已不是必需参数
- 写入成功后会输出包含 `record` 的 JSON
- 最终交付的二进制必须写成 `integration token ...`
- 像 `/path/to/integration --agentId A --function main ...` 这种不带 `token` 子命令的写法，会进入默认服务启动参数解析并报 `flag provided but not defined`

## HTTP

### `GET /api/consume`

- 参数：
  - `starTime`：必填，格式为 `yyyyMMdd-hhmmss`
  - `closeTime`：必填，格式为 `yyyyMMdd-hhmmss`
  - `agentId`：选填；不传时查询所有 Agent
  - `limit`：选填；不传默认 `500`，表示最多返回的明细条数
- 返回：
  - `details`：命中的全部消费明细
  - `summary`：按模型聚合后的 `thinking`、`input`、`total`、`cache`

示例：

```bash
curl "http://127.0.0.1:8080/api/consume?agentId=demo-agent&starTime=20260527-090000&closeTime=20260527-180000&limit=500"
```

## 存储说明

- 与其他能力共用应用启动目录下的 `data` sqlite
- 明细记录保存以下字段：
  - `thinking`
  - `input`
  - `total`
  - `cache`
  - `model`
  - `agentId`
  - `function`
  - `timestamp`
- 新增索引：时间戳 + AgentId
