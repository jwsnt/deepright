# CLI-Get 迭代手册（20260804-1）

## 本次更新

独立 `cli-get` 在成功心跳后不再立即高频轮询，而是使用 `config.json.get.await` 作为待机间隔，从而减少没有交互时的 `/cli/get` 访问。

## 配置

在 `config/config.json` 中配置：

```json
{
  "get": {
    "sleep": 15000,
    "await": 30000
  }
}
```

| 字段 | 单位 | 作用 |
| --- | --- | --- |
| `get.sleep` | 毫秒 | 心跳失败退避的基数，以及任务队列满时的等待时间 |
| `get.await` | 毫秒 | 每次成功 `/cli/get` 后到下一次上报前的等待时间 |

- 两个字段必须是非负整数。
- 显式传入 `--sleep` 时优先使用命令行值；未传时使用 `get.sleep`。
- `await` 没有命令行参数，始终读取 `get.await`。

## 心跳行为

- `/cli/get` 成功返回任务：任务先进入本地 `taskQueue`，不等待执行 worker 或 `/cli/pub`；随后等待 `get.await` 再上报。
- `/cli/get` 成功返回无任务：等待 `get.await` 再上报。
- `/cli/get` 网络错误、超时、HTTP 非 200 或响应解析失败：继续按 `--sleep` 指数退避。
- 本地 `taskQueue` 满：不请求 `/cli/get`，继续按 `--sleep` 等待后检查。

独立 `cli-get` 是单独进程，不能读取 Integration 进程内的 SSE 活跃计数。因此它始终使用上述成功后的 `await` 策略；Integration 内置 cli-get 会在有未终态 SSE 时跳过该等待。

## 保持不变的能力

- `taskQueue -> execute worker -> publishQueue -> cli/pub` 两段式流水线；
- `ddl` 出队执行前校验；
- 沙盒和 `subOps.exempted` 豁免；
- 活跃命令注册与 `/api/kill`；
- `/cli/pub` 重试、`message_insert` 上报与日志协议。
