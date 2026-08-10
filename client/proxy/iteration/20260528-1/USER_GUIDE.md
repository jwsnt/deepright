# 20260528-1

本次迭代把 `proxy` 的字段命名与默认端口统一收口，并把本模块实际负责的上游报文统一为保留 `messages` 的格式；其中 `/v1/chat/completions` 和内部 cron 继续保留 `stream`。

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
- `proxy` 默认监听端口改为 `8080`

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
- 不再向上游继续发送旧的顶层 `thinking` / `html` / `router_disable`
- `messages`、`stream` 保持原逻辑

### 内部 cron 执行请求

proxy 内部备忘录/周期任务真正执行并转发到上游时，也统一使用同一份报文结构：

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

## 兼容说明

- 对外入口如果传入简化的顶层 `message`，proxy 会在内部转换成单条 `messages`
- 发往上游时继续保持 `messages` + `stream` 结构
- `/cli/get`、`/cli/pub` 不在 proxy 当前模块的 HTTP 路由范围内，由对应模块单独收口
- 日志导出会优先读取 `messages[].content`，同时兼容单条 `message`

## 端口

- `proxy --port` 默认值：`8080`
- 示例：

```bash
./proxy serve --agent-dir ./agents --port 8080 --host http://127.0.0.1:9998
```
