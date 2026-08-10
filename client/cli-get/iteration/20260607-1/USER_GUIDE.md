# 迭代说明

本次迭代为 `cli-get` 主模块增加了可选的沙盒代理执行模式，同时保持默认直连执行链路不变。

## 新增能力

- 新增内存开关 `sandbox`，默认 `false`
- `sandbox=false` 时，继续沿用原链路：
  - `cli/get -> 本地执行 -> cli/pub`
- `sandbox=true` 时，切换为沙盒代理链路：
  - `cli/get -> localhost:<sandbox_port>/cli/delegate/get`
  - `CLI_SANDBOX` 执行命令
  - `CLI_SANDBOX -> localhost:<port>/cli/delegate/pub`
  - `cli-get` 收到明文结果后再按原逻辑压缩 `cmd`，最后回传远端 `cli/pub`

## 新增参数

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--sandbox` | `false` | 是否启用本地 `CLI_SANDBOX` 代理执行 |
| `--port` | `8080` | 本地监听 `/cli/delegate/pub` 的端口 |
| `--sandbox_port` | `8180` | 本地 `CLI_SANDBOX` 的监听端口 |

## 兼容性

- 默认不启用沙盒，因此现有 `cli-get` 行为不变
- `cli/pub` 的远端回传格式不变，仍然是 GZIP+Base64 后的 `cmd`
- 统一日志仍然写入当前工作目录下的 SQLite `data`

## 测试

- 新增了沙盒转发请求格式测试
- 新增了 `/cli/delegate/pub` 明文结果转回 `cli/pub` 的测试
- 保留并通过原有本地执行、回传和日志测试
