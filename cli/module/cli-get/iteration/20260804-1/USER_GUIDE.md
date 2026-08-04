# CLI-Get 迭代手册（20260804-1）

## 本次更新

独立 `cli-get` 在连续 `get.check` 次未收到 cmd 后使用 `config.json.get.await` 降低 `/cli/get` 访问。

## 配置

在 `config/config.json` 中配置：

```json
{
  "get": {
    "sleep": 15000,
    "await": 30000,
    "check": 10
  }
}
```

| 字段 | 单位 | 作用 |
| --- | --- | --- |
| `get.sleep` | 毫秒 | 心跳失败退避的基数，以及任务队列满时的等待时间 |
| `get.await` | 毫秒 | 连续 `get.check` 次未收到 cmd 后到下一次上报前的等待时间 |
| `get.check` | 次数 | 连续无 cmd 结果的阈值；成功与失败均计入 |

- `sleep`、`await` 必须是非负整数；`check` 必须是正整数。
- 显式传入 `--sleep` 时优先使用命令行值；未传时使用 `get.sleep`。
- `await` 没有命令行参数，始终读取 `get.await`。
- `check` 没有命令行参数，必须为正整数，始终读取 `get.check`。

## 心跳行为

- `/cli/get` 成功返回非空 cmd：任务先进入本地 `taskQueue`，不等待执行 worker 或 `/cli/pub`；连续无 cmd 计数原子归零，并立即再上报。
- 每次未收到 cmd 都会原子递增连续计数，包括成功无任务、网络错误、超时、HTTP 非 200 和响应解析失败。成功无 cmd 且计数小于 `get.check` 时立即再上报；达到或超过 `get.check` 时等待 `get.await`。
- 失败无 cmd 在计数未达到 `get.check` 前按 `--sleep` 指数退避；恰好达到阈值时直接等待 `get.await`，不再先 sleep。
- 任一新的 cmd 都会原子归零该计数并重新开始。
- 本地 `taskQueue` 满：不请求 `/cli/get`，继续按 `--sleep` 等待后检查。
- 当服务端 HTTP 状态码或 JSON 业务 `code` 等于主 `config/config.json.page.new_tab` 或 `page.iframe`（例如 `931`、`932`）时，独立 `cli-get` 把它当作成功但无任务的响应：不输出 heartbeat error、不计入失败或连续无 cmd 的异常路径，也不会影响页面的远程服务报警。

独立 `cli-get` 是单独进程，不能读取 Integration 进程内的 SSE 活跃计数。因此它只使用上述 `cmd -> check -> await` 策略；Integration 内置 cli-get 在有未终态 SSE 时只跳过 `await`，无 cmd 结果仍会计入 `check`。

## 保持不变的能力

- `taskQueue -> execute worker -> publishQueue -> cli/pub` 两段式流水线；
- `ddl` 出队执行前校验；
- 沙盒和 `subOps.exempted` 豁免；
- 活跃命令注册与 `/api/kill`；
- `/cli/pub` 重试、`message_insert` 上报与日志协议。
