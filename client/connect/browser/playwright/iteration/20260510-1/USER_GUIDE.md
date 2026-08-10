# browser_playwright 迭代 20260510-1 使用手册

## 变更说明

本次迭代解决两个问题：

- 为 `eval` 增加执行超时保护，避免把 Playwright daemon 卡死
- 在复用 `browser_instance` 前先校验 CDP 健康，不再盲目复用旧实例

## eval 超时保护

旧行为里，如果执行了类似下面这种永远不返回的脚本：

```bash
./browser_playwright --session demo --timeout 15000 eval 'new Promise(() => {})'
```

客户端虽然可能先报超时，但 daemon 内部的 `page.Evaluate(...)` 仍可能长时间挂住，导致后续 `attach`、`create`、`goto` 都只能排队等待。

现在的行为：

- `eval` 会在页面内包一层超时保护
- 到达 `--timeout` 后会直接抛出超时错误
- 不再让悬挂脚本长时间占住 daemon

推荐写法：

```bash
./browser_playwright --session demo --timeout 15000 eval 'new Promise(resolve => setTimeout(() => resolve("ok"), 1000))'
```

如果确实需要更长时间，显式调大 `--timeout`：

```bash
./browser_playwright --session demo --timeout 60000 eval 'longRunningTask()'
```

## instance 复用健康检查

旧行为里，如果 `browser_instance` 记录中的进程还活着、端口还在监听，就可能直接复用；但这并不保证这个端口仍然是健康的 CDP 服务。

现在的行为：

- 在复用前会检查对应端口的 `/json/version`
- 只有当返回值中的 `webSocketDebuggerUrl` 与当前实例应有地址一致时，才会复用
- 如果旧进程还活着，但端口已经不是健康 CDP 服务，则会先清理旧实例，再重新创建

## 受影响命令

以下命令都会受益于这次修复：

- `create --agentId ... --chatId ...`
- `attach --cdp=chrome|URL`
- 未显式传 `--cdp`、但通过 `--agentId + --chatId` 或 `--session agent@chat` 自动接管 `browser_instance` 的普通 Playwright 命令

示例：

```bash
./browser_playwright create --agentId agent-a --chatId chat-001
./browser_playwright --agentId agent-a --chatId chat-001 snapshot
./browser_playwright --session agent-a@chat-001 eval 'document.title'
```

## 推荐验收

```bash
./browser_playwright --session demo --timeout 5000 eval 'new Promise(() => {})'
./browser_playwright create --agentId agent-a --chatId chat-001
./browser_playwright --session agent-a@chat-001 goto https://example.com
```

重点检查：

- 永不返回的 `eval` 能在 `--timeout` 后结束，而不是把 daemon 卡住数分钟
- `create` 不会再盲目复用失效的旧 CDP 实例
- 自动实例接管链路在健康实例场景下仍保持原有兼容行为
