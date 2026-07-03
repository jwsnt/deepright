# Browser 迭代 20260509-3 使用手册

## 变更说明

本次迭代为 `browser` 代理的全部 Playwright 能力补齐新的超时恢复策略：

- 单次命令超时先只报错，不立刻杀 daemon
- 连续超时累计达到 `--browser_retry` 指定次数后才回收 daemon，默认 `3`
- `/command` 命中瞬时网络错误时，不再固定只做 3 次短重试，而是会在 `--browser-timeout` 窗口内持续重试

## 验收结果

当前实现已经与本轮需求一致，关键行为如下：

- `browser_playwright` 和 `browser` 两层 CLI 都支持 `--browser_retry`
- 未显式传入时，连续超时回收阈值默认为 `3`
- 命令只要成功一次，连续超时计数就会清零
- 非超时错误返回时，同样会清零连续超时计数，避免旧状态残留
- `/command` 对 `connection refused`、`EOF`、`connection reset by peer`、`broken pipe` 会在当前 `--browser-timeout` 剩余时间内持续重试

## 参数说明

新增参数：

```bash
--browser_retry N
```

含义：

- 控制连续超时多少次后才主动回收当前 Playwright daemon
- 默认值为 `3`
- 传入小于等于 `0` 或未传时，都会退回默认值

## 使用示例

```bash
./browser --browser-timeout 30s --browser_retry 5 --session agent-a@ctrip-home goto https://www.ctrip.com
./browser_playwright --browser-timeout 20s --browser_retry 2 --session demo goto https://example.com
```

## 行为说明

- 如果 daemon 正在重启、端口刚切换、或者命中短暂的本地连接抖动，CLI 会在 `--browser-timeout` 的时间窗口内继续重试，而不是很快失败
- 如果某次命令真的超时，当前 daemon 会先保留，方便下一条命令继续复用
- 如果连续超时达到阈值，CLI 才会认为 daemon 已进入异常状态并主动回收

## 同步文档

本次迭代变更已同步到：

- `module/connect/browser/USER_GUIDE.md`
- `module/connect/browser/playwright/USER_GUIDE.md`
