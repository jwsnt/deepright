# 迭代说明

本次迭代为 `integration` 增加了内部 `cli-get` 的沙盒开关 API，并把沙盒执行链路收口到 `integration` 主服务。

## 新增能力

- 新增启动参数 `--sandbox`，默认 `false`
- 新增 `GET/POST /api/sandbox?sandbox=true|false`
  - 不带 `sandbox` 参数时返回当前状态
  - 带参数时会立即切换内部 `cli-get` 的沙盒执行开关
- 当沙盒开启时，内部 `cli-get` 会改走：
  - `cli/get -> localhost:<port+100>/cli/delegate/get`
  - `CLI_SANDBOX -> localhost:<port>/cli/delegate/pub`
  - `integration -> 远端 cli/pub`

## 兼容性

- 默认仍为 `sandbox=false`，原有本地执行链路不变
- 远端 `cli/pub` 报文仍然保持原来的 GZIP+Base64 结果格式
- 统一日志仍然按原逻辑写入共享 `data`

## 测试

- 新增了内部沙盒转发请求测试
- 新增了 `/cli/delegate/pub` 收口并转回 `cli/pub` 的测试
- 新增了 `/api/sandbox` 状态切换测试
