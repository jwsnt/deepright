# Integration 迭代 20260509-3 使用手册

## 变更说明

本次迭代补充了 `integration start` 的启动失败透传能力，并把 `integration stop` 调整为幂等停止。

当后台启动阶段因为端口占用或其他异常导致服务无法真正启动时，CLI 不再只表现为笼统的等待超时，而会直接输出明确失败原因。

同时，当执行 `integration stop` 时如果 integration 进程本身已经不存在，或者某个已配置插件当前并未真正启动，命令也会继续按成功处理，不再因为“没有可关闭进程”把停止流程判为失败。

## 行为规则

- `integration start` 会在拉起后台 `serve` 子进程后，持续等待服务就绪
- 如果子进程在就绪前主动上报启动失败原因，父进程会立即结束等待，并把该原因输出到控制台
- 同一份失败原因也会写入 `integration.log`
- `integration stop` 在 integration 主进程已经不存在时默认成功，并清理残留 PID 文件
- `integration stop` 只会对“已配置且当前已启动”的插件执行 `stop`
- 如果插件仅存在配置、但没有有效 PID 或进程已退出，会直接跳过，不影响 integration 主进程关闭结果
- 典型场景包括：
  - 端口被占用
  - 监听地址失败
  - 启动前初始化异常
  - 重复执行 `integration stop`
  - 插件 PID 文件残留，但插件进程已经退出

## 输出位置

- 控制台：当前执行 `integration start` 的 stderr
- 日志：`integration.log` 中的 `[integration lifecycle] start failed: ...`

对于 `stop` 场景，相关跳过与清理信息会写入 `integration.log`，例如插件未启动时的 `plugin stop skipped`。

## 示例

例如端口 `8080` 已被其他进程占用时，执行：

```bash
./integration start --port 8080
```

可能直接看到类似输出：

```text
server listen failed: listen tcp :8080: bind: address already in use
```

同时 `integration.log` 中也会追加同样的失败原因，便于后续排查。

如果此时直接执行：

```bash
./integration stop
```

即使当前没有可关闭的 integration 进程，或者某个已配置插件并没有真实运行，命令也会默认成功，只在日志里记录跳过或清理信息。
