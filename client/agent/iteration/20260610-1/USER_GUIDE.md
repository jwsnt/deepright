# 20260610-1 迭代说明

本次迭代为 Agent 元数据新增了两个字段：

- `version`
  - 来自 `--agent-dir/<agentId>/config.json` 中的 `version`
  - 如果 `config.json` 不存在或未声明该字段，则输出空字符串
  - 该字段只在当前 Agent metadata 缓存周期首次扫描时读取一次；缓存未失效前不会因为 `--agent-dir/<agentId>/config.json` 中的 `version` 变化而立刻刷新
- `sandbox`
  - 按 `agentId + chatId` 实时读取共享 sqlite 的 `cli_sandbox_state`
  - 如果未传 `chatId`，或当前会话从未写入沙盒状态，则输出空字符串
  - 该字段不参与缓存

## CLI

新增可选参数：

```bash
./agent-scanner --chatId chat-001 ./agents
./agent-scanner --app-dir /path/to/app --chatId chat-001 ./agents
```

- `--chatId` 只影响 `sandbox` 字段
- `version` 仍然沿用 `--agent-cache` 的缓存周期

## 子模块调用

新增会话感知接口：

```go
data, err := GetAgentOutputForAppAndChat("/path/to/agents", "/path/to/app", "device-id", 120*time.Second, "chat-001")
```

- 返回值会同时包含：
  - 缓存版 `version`
  - 实时版 `sandbox`
