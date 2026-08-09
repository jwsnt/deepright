# 20260508-1 使用手册

## 目标

本次迭代为 `browser_playwright` 增加 `create` 命令。

`create` 的行为是：

- 先调用相对路径的 `../instance/browser_instance create`
- 用 `agentId + chatId` 创建或复用一个独立的 CDP 服务
- 再自动以 `Agent@Chat` 作为会话名执行 `attach`

这样上层不需要手动先查端口再拼接 `attach` 命令。

## 推荐用法

在 [/path/to/deepright/cli/module/connect/browser/playwright](../..) 目录执行：

```bash
./browser_playwright create --agentId demo-agent --chatId chat-001
./browser_playwright --session demo-agent@chat-001 goto https://example.com
./browser_playwright --session demo-agent@chat-001 snapshot
```

## create 行为说明

```bash
./browser_playwright create --agentId demo-agent --chatId chat-001
```

说明：

- `create` 必须同时提供 `--agentId` 和 `--chatId`
- 会先调用同级相对路径的 `../instance/browser_instance`
- 底层实例创建成功后，会自动 attach 到 `ws://127.0.0.1:<port>/devtools/browser`
- `create` 固定把会话名设置为 `demo-agent@chat-001`
- 外部传入的 `--session` 不会覆盖 `Agent@Chat` 这条规则
- 如果相同 `AgentId + ChatId` 已经存在活跃实例，则直接复用
- `--agentId` 和 `--chatId` 会在这条链路开始时先统一转换为小写，再参与实例匹配和会话名生成

等价理解如下：

```bash
./browser_playwright create --agentId demo-agent --chatId chat-001
```

约等于：

```bash
../instance/browser_instance create --agentId demo-agent --chatId chat-001
./browser_playwright --session demo-agent@chat-001 attach --cdp=ws://127.0.0.1:<port>/devtools/browser
```

其中 `<port>` 由 `browser_instance create` 返回。

## 可选参数

如需覆盖默认实例程序路径，可显式指定：

```bash
./browser_playwright create --agentId demo-agent --chatId chat-001 --instance-bin /opt/browser_instance
```

如果需要把底层实例参数继续传给 `browser_instance`，可使用：

```bash
./browser_playwright create --agentId demo-agent --chatId chat-001 \
  --instance-state /tmp/browser_instance.json \
  --instance-obscura /opt/obscura/obscura \
  --instance-monitor=false
```

## 验收建议

```bash
./browser_playwright create --agentId demo-agent --chatId chat-001
./browser_playwright --session demo-agent@chat-001 goto https://example.com
./browser_playwright --session demo-agent@chat-001 snapshot
```

重点检查：

- `create` 能成功返回，不再需要手工执行 `attach`
- 会话名固定为 `demo-agent@chat-001`
- 后续 `goto`、`snapshot` 等命令能继续复用同一会话
