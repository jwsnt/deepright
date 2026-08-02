# Integration 迭代手册（20260802-2）

## 本次更新

macOS 防休眠改为按需运行：Integration 不再在服务全程无条件持有 `caffeinate`。

## 配置

在 `config/config.json` 设置必填的扫描间隔，单位为分钟：

```json
{
  "caffeinate": 15
}
```

该值必须是正整数。配置缺失、非整数或小于等于零时，Integration 会记录错误并拒绝启动；配置只在启动时读取，修改后需要重启服务。

## 触发条件

服务启动时立即检查，之后按 `caffeinate` 配置的周期复核。以下任一条件成立时，macOS 会运行 `caffeinate -d -i -m -s`：

- 飞书或邮件插件已成功启动；
- 有尚未处理、计划执行时间在未来 24 小时内的备忘录任务；
- 有尚未结束的 SSE 响应。

SSE 开始、完成及异常结束都会立即重新评估。上游错误、EOF、解析失败、写入失败、客户端断开和上下文取消都会被视为已结束，不会遗留防休眠状态。

当三项条件均不满足时，Integration 终止 `caffeinate` 并允许系统休眠。仅 macOS 启用；其他平台保持原有行为。

## 关闭行为

`integration stop`、`restart`、`SIGINT`、`SIGTERM` 和本机 `/api/shutdown` 会先停止条件检查并尽力终止 `caffeinate`，再关闭插件、任务和 HTTP 服务。子进程已退出、终止失败或等待超时只会写入日志，不会阻断应用退出。
