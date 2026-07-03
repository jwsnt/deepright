# 20260508-1 使用手册

## 目标

本次迭代为 `browser_instance` 增加按 `AgentId + ChatId` 维度的空闲自动释放能力。

核心行为：

- 为每个 `agentId + chatId` 独立记录最近活跃时间 `lastActiveAt`
- 默认每分钟检查一次当前实例状态
- 当实例连续空闲超过 `--browser_expired` 分钟时自动关闭对应 Obscura 进程
- 默认空闲超时时间为 `10` 分钟
- `agentId + chatId` 在实例匹配和端口计算前会先统一转换为小写

## 触发活跃时间刷新

以下受管命令会刷新 `lastActiveAt`：

- `./browser_instance create --agentId ... --chatId ...`
- `./browser_instance get --agentId ... --chatId ...`

上层统一入口 `browser` 也会在以下场景同步刷新：

- `./browser create --agentId ... --chatId ...`
- `./browser --session agent@chat ...`

这样可以避免仍在使用中的 `Agent@Chat` 会话被后台监控误回收。

## 参数说明

### `--browser_expired`

空闲自动释放时间，单位分钟，默认 `10`：

```bash
./browser_instance create --agentId demo-agent --chatId chat-001 --browser_expired 15
```

要求：

- 必须为正整数
- 未指定时默认 `10`

### `--monitor-ms`

后台检查间隔，单位毫秒，默认 `60000`：

```bash
./browser_instance create --agentId demo-agent --chatId chat-001 --monitor-ms 10000
```

## 状态文件

状态文件仍然位于应用程序同目录下的 `browser_instance.json`，本次迭代新增 `lastActiveAt` 字段：

```json
[
  {
    "agentId": "demo-agent",
    "chatId": "chat-001",
    "port": 28412,
    "pid": 95231,
    "cdp": "ws://127.0.0.1:28412/devtools/browser",
    "lastActiveAt": "2026-05-08T12:00:00Z"
  }
]
```

说明：

- `list` 会清理已失效 `pid`
- `list` 也会顺便清理已超时空闲的实例
- `shutdown` 会删除对应状态记录

## 验收建议

```bash
./browser_instance create --agentId demo-agent --chatId chat-001 --browser_expired 1 --monitor-ms 1000
./browser_instance list
sleep 70
./browser_instance list
```

重点检查：

- 创建后 `browser_instance.json` 中存在 `lastActiveAt`
- 空闲超时后实例会自动从列表中消失
- 对同一实例执行 `create` 或 `get` 后，`lastActiveAt` 会被刷新
