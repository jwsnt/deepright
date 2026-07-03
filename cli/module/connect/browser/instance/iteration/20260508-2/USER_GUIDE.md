# browser_instance 迭代 20260508-2 使用手册

## 变更说明

本次迭代为 `browser_instance` 增加 `restart` 命令，并要求它对同一组 `agentId + chatId` 复用同一个稳定端口。

命令：

```bash
./browser_instance restart --agentId demo-agent --chatId chat-001
```

行为：

- 先执行一次 `shutdown`
- 再执行一次 `create`
- 由于端口算法对同一组 `agentId + chatId` 是幂等的，因此正常情况下会回到同一个稳定端口
- 返回值不是 `OK`，而是最新实例配置 JSON

## restart 返回示例

```json
{
  "agentId": "demo-agent",
  "chatId": "chat-001",
  "port": 28412,
  "pid": 95232,
  "cdp": "ws://127.0.0.1:28412/devtools/browser"
}
```

说明：

- `pid` 通常会变成新的进程号
- `port` 在正常情况下会保持不变
- `cdp` 始终按最新端口回填为 `ws://127.0.0.1:<port>/devtools/browser`

## help

本次迭代同时要求 `help` 能输出完整手册：

```bash
./browser_instance help
```

至少应覆盖：

- `create`
- `restart`
- `shutdown`
- `list`
- `get`

## 推荐验收

```bash
./browser_instance create --agentId demo-agent --chatId chat-001
./browser_instance restart --agentId demo-agent --chatId chat-001
./browser_instance get --agentId demo-agent --chatId chat-001
```

重点检查：

- `restart` 前后 `port` 保持稳定
- `restart` 后 `pid` 为新值
- `help` 能看到 `restart` 的完整说明
