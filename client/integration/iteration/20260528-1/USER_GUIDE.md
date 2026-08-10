# 20260528-1

本次迭代把 `integration` 的字段命名与默认端口统一收口，并把转发到上游 `--host` 的报文统一为保留 `messages` 的格式；其中 `/v1/chat/completions` 与内部 cron 继续保留 `stream`，`/cli/get` 不再携带 `stream`。

## 变更点

- 统一变量名：
  - `SWARM` 对应 `router_disable`
  - 思考模式对应 `thinking`
  - HTML 开关对应 `html`
  - 模型选择对应 `model`
  - `metadata.agents[]` 中的模型字段统一为 `provider`
- `router_disable` 语义保持：
  - `SWARM` 开启时：`router_disable=false`
  - `SWARM` 关闭时：`router_disable=true`
- `integration` 默认监听端口统一为 `8080`
- `integration start` / `restart` 在未显式传入 `--port` 时，会回到默认 `8080`

## 上游转发协议

### `/v1/chat/completions`

发往 `--host` 的请求体统一为：

```json
{
  "messages": [
    {
      "role": "user",
      "content": "你好"
    }
  ],
  "stream": true,
  "metadata": {
    "thinking": true,
    "html": false,
    "router_disable": false,
    "agents": [
      {
        "provider": "OpenAI",
        "thinking": true,
        "router_disable": false
      }
    ]
  },
  "model": "gpt-4"
}
```

说明：

- `thinking`、`html`、`router_disable` 只保留在 `metadata` 内
- 不再向上游继续发送旧的顶层布尔字段
- `messages`、`stream` 保持原逻辑

### `/cli/get`

发往 `--host` 的心跳请求统一为：

```json
{
  "messages": [
    {
      "role": "user",
      "content": ""
    }
  ],
  "metadata": {
    "agents": [
      {
        "provider": "OpenAI",
        "thinking": true,
        "router_disable": false
      }
    ]
  }
}
```

说明：

- 心跳请求保持 `messages` 原逻辑，但不再向上游发送 `stream`

### 内部 cron 执行请求

integration 内部备忘录/周期任务真正执行并转发到上游时，也统一使用：

```json
{
  "messages": [
    {
      "role": "user",
      "content": "按时检查接口健康"
    }
  ],
  "stream": true,
  "metadata": {
    "thinking": false,
    "router_disable": true,
    "cron_type": "cron"
  },
  "model": "OpenAI"
}
```

说明：

- 执行阶段优先使用当前 `task_detail.router_disable`
- 不会回退到 Agent `config.json` 中的默认值

## 日志与兼容

- 对外入口如果传入简化的顶层 `message`，integration 会在内部转换成单条 `messages`
- 发往 `/v1/chat/completions` 和内部 cron 时继续保持 `messages` + `stream` 结构
- 发往 `/cli/get` 时保持 `messages`，但不再携带 `stream`
- 日志导出会优先读取 `messages[].content`，同时兼容单条 `message`

## 端口

- `integration --port` 默认值：`8080`
- 示例：

```bash
./integration --agent-dir ./agent --port 8080 --host http://127.0.0.1:9998
```
