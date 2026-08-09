# Integration 迭代手册（20260516-4）

## 本次收口

本次迭代在 integration 主二进制下统一收口两类能力：

- `proxy/iteration/20260516-3` 中的 `token` 命令
- `proxy/iteration/20260516-4` 中的 Agent `provider` 元数据透传能力

这意味着：

- 不再要求调用方切换到独立 `proxy` 二进制读取模型密钥
- `integration` 可以直接通过顶层 CLI 读取设置页中已经保存的模型与密钥
- `integration` 下的 `/v1/chat/completions`、`cli/get`、`cli/pub`、内部 cron 执行链路，都会复用统一的 Agent `provider` 元数据输出

## Token CLI 用法

```bash
cd /path/to/deepright/cli/module/integration
./integration token
./integration token --provider deepseek
```

输出示例：

```json
[{"deepseek":"aaa"},{"kimi":"bbb"}]
```

```json
{"deepseek":"aaa"}
```

## Token 行为规则

- `integration token` 直接读取共享 SQLite `data` 中保存的模型与密钥
- 不带 `--provider` 时，输出按模型名排序后的 JSON 数组
- 带 `--provider MODEL` 时，只输出指定 provider 的单个 JSON 对象
- 如果指定模型不存在，则输出空对象 `{}`
- 该命令是只读 CLI，不会改写已有设置

## Token 路径规则

- `integration token` 优先按当前启动目录下的 `runtime.json` 解析共享 SQLite `data`
- 如果当前没有 `runtime.json`，则按 `integration` 既有规则回退解析数据库路径
- 因此在一体化部署场景下，应优先使用 `integration token`，避免和独立 `proxy` 的启动目录混淆

## Agent Provider 收口

- `integration` 会在统一 Agent metadata 中输出 `agents[].provider`
- 该字段来自对应 Agent 工作目录下的 `config.json.provider`
- 如果 `config.json` 不存在，或未声明 `provider`，则输出空字符串

同时，以下 `config.json` 字段也会一起透传：

- `description`
- `provider`
- `thinking`
- `swarm`

## 实时读取规则

- `provider` 与同一份 `config.json` 中的 `description`、`thinking`、`swarm` 一样，都会在每次 metadata 输出前实时重新读取
- 这些字段不受 `--agent-cache` 影响
- 即使 integration 进程使用了较长的 `--agent-cache`，只要 Agent 工作目录下的 `config.json` 被修改，下一次 `/v1/chat/completions`、`cli/get`、`cli/pub` 或内部 cron 请求生成的 metadata 都会立即反映最新结果

## 与 Proxy 的关系

- `integration token` 是对 `proxy token` 需求的主二进制收口实现
- `integration` 中的 Agent `provider` metadata 行为与 `proxy` 保持一致
- 已经依赖 `proxy token` 或 `proxy` metadata 结构的上层调用方，可以直接平移到 `integration`
