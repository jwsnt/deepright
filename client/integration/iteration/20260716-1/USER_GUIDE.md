# 迭代 20260716-1：Integration `device` 热更新

Integration 服务会将 `deviceId` 放入 Agent metadata，并用于 heartbeat、聊天转发、cron 与 `GET /api/deviceId`。

## 配置优先级

服务启动时按以下顺序确定 device：

1. 非空 `--device`
2. `config/config.json` 的非空字符串 `device`
3. 系统自动生成值

例如：

```json
{
  "device": "1234"
}
```

键缺失、`null`、数字、对象、空字符串和仅空白字符串都等同于未配置。系统自动生成值只在当前服务进程启动时生成一次。

当指定 `--device=my-device` 时，该值优先于配置文件，并沿用服务现有的运行时配置写回行为。

## 运行期刷新

服务启动后每 60 秒检查一次 `config/config.json`：

- 文件未变化：不重新读取，也不输出刷新日志；
- 配置可读取：更新内存缓存，后续新请求使用新的有效 `deviceId`；
- 配置读取或 JSON 解析失败：继续使用最近一次成功值，并记录一次失败日志；文件不再变化时不会重复刷日志；
- 命令行指定有效 `--device` 时，配置变化不会覆盖该值。

刷新采用最终一致性。刷新前已经开始的请求可以使用旧 device，新请求会读取最新缓存，不会中断正在处理的聊天、心跳或 cron。

可通过以下接口查看当前有效值：

```bash
curl http://127.0.0.1:8080/api/deviceId
```
