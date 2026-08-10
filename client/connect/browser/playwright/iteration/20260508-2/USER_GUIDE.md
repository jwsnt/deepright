# 20260508-2 使用手册

## 目标

本次迭代为所有 `browser_playwright` 命令增加统一的 `--browser-timeout` 总超时控制。

适用范围包括：

- 直接执行的 `browser_playwright` 命令
- 自动拉起 daemon 的过程
- 命令发送到 daemon 后的等待过程
- `create` 命令内部代理的 `browser_instance create`

超时后会立即终止当前流程，而不是让进程继续卡住。

## 参数说明

```bash
--browser-timeout VALUE
```

说明：

- 默认值为 `120s`
- 纯数字按秒解析，例如 `45` 等价于 `45s`
- 也支持 Go duration 格式，例如 `3s`、`45s`、`2m`
- 必须大于 `0`

示例：

```bash
./browser_playwright --browser-timeout 45s open https://example.com
./browser_playwright --browser-timeout 20 goto https://example.com
./browser_playwright --browser-timeout 15s create --agentId demo-agent --chatId chat-001
```

## 超时行为

### 普通命令

当一次 CLI 调用超过 `--browser-timeout` 时，会直接返回错误：

```text
browser_playwright <command> timed out after 45s
```

同时会尝试立即停止当前 `browser_playwright` daemon，避免请求已经超时但后台进程还继续阻塞。

### start

```bash
./browser_playwright --browser-timeout 10s start
```

如果 daemon 在限定时间内没有成功监听，会直接失败，并清理刚拉起的后台进程。

### create

```bash
./browser_playwright --browser-timeout 15s create --agentId demo-agent --chatId chat-001
```

`create` 内部会把总超时继续传递给实例创建流程；其中底层 `browser_instance create` 还会再做一次保护，避免实例创建长时间无响应。

## 推荐用法

对可能访问慢站点、登录态站点或远程 CDP 的命令，建议显式带上超时：

```bash
./browser_playwright --browser-timeout 30s attach --cdp=chrome
./browser_playwright --browser-timeout 30s goto https://www.ctrip.com
./browser_playwright --browser-timeout 60s pdf --filename page.pdf
```

## 验收建议

```bash
./browser_playwright --browser-timeout 1ms start
./browser_playwright --browser-timeout 1ms open https://example.com
```

重点检查：

- 命令会快速失败，而不是长时间卡住
- 错误信息中包含实际超时时长
- 超时后不会留下持续挂起的 `browser_playwright` 进程
