# CLI-Get 迭代手册（20260804-1）

## 本次更新

独立 `cli-get` 在待机时使用 `config.json.get.await` 降低 `/cli/get` 访问；收到 cmd 后会先按 `get.check` 进行连续快速检查。

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
| `get.await` | 毫秒 | 每次成功 `/cli/get` 后到下一次上报前的等待时间 |
| `get.check` | 次数 | 收到 cmd 后重新进入 `await` 前允许的连续无 cmd 快速检查次数 |

- `sleep`、`await` 必须是非负整数；`check` 必须是正整数。
- 显式传入 `--sleep` 时优先使用命令行值；未传时使用 `get.sleep`。
- `await` 没有命令行参数，始终读取 `get.await`。
- `check` 没有命令行参数，必须为正整数，始终读取 `get.check`。

## 心跳行为

- 进程启动后先立即请求一次 `/cli/get`；首个成功无 cmd 响应等待 `get.await`。
- `/cli/get` 成功返回非空 cmd：任务先进入本地 `taskQueue`，不等待执行 worker 或 `/cli/pub`；连续无 cmd 计数原子归零，并立即再上报。
- 收到过 cmd 后，成功无 cmd 响应会原子递增连续计数；小于 `get.check` 时立即再上报，达到 `get.check` 时等待 `get.await`。
- 快速检查期间任一新的 cmd 都会原子归零该计数并重新开始。
- `/cli/get` 网络错误、超时、HTTP 非 200 或响应解析失败：继续按 `--sleep` 指数退避。
- 本地 `taskQueue` 满：不请求 `/cli/get`，继续按 `--sleep` 等待后检查。

独立 `cli-get` 是单独进程，不能读取 Integration 进程内的 SSE 活跃计数。因此它只使用上述 `cmd -> check -> await` 策略；Integration 内置 cli-get 会在有未终态 SSE 时优先跳过 `check` 和 `await`。

## 保持不变的能力

- `taskQueue -> execute worker -> publishQueue -> cli/pub` 两段式流水线；
- `ddl` 出队执行前校验；
- 沙盒和 `subOps.exempted` 豁免；
- 活跃命令注册与 `/api/kill`；
- `/cli/pub` 重试、`message_insert` 上报与日志协议。
