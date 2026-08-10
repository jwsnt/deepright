# Integration 迭代手册（20260718-1）

## 本次更新

- Integration 现在会在发往上游的请求顶层 metadata 中统一补充 `port`。
- 该字段覆盖普通 `/v1/chat/completions` 转发、`/cli/get` 心跳，以及备忘录、邮件、飞书等自动任务最终发往上游的 `/v1/chat/completions` 请求。

## 端口来源

`metadata.port` 是整数，取当前 Integration 进程实际使用的监听端口，优先级与服务启动保持一致：

1. 显式传入的 `--port`
2. `config/config.json` 中的 `port`
3. 内置默认值 `8080`

例如：

```bash
./integration --port 18080
```

上游收到的 metadata 会包含：

```json
{
  "metadata": {
    "port": 18080
  }
}
```

## 兼容性说明

- 这是 Integration 运行时拥有的字段。即使外部请求提供了 `metadata.port`，转发前也会以当前服务的实际端口覆盖。
- 字段位于顶层 `metadata`，不会写入 `metadata.agent`、`metadata.agents[]`、备忘录或 Connect 任务的持久化记录。
- 不新增接口、命令行参数或配置项。
