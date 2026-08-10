# 迭代说明

本次迭代将 `cli-get` 的沙盒状态从 `agentId + chatId` 联合命中改为仅按 `chatId` 命中。沙盒状态仍保存在当前工作目录的 SQLite `data` 文件中。

## 状态命中规则

- 只有非空 `chatId` 可以读取、写入、更新或删除沙盒状态。
- 任务执行前只使用任务的 `chatId` 选择沙盒模式；`agentId` 不参与状态查询。
- 因此，两个不同的 `agentId` 只要使用相同的 `chatId`，就会使用同一个沙盒模式。
- `chatId` 为空、全为空白，或没有对应状态记录时，不会启用沙盒，任务继续使用原来的本地 Shell 执行链路。
- `agentId` 仍会保留在任务执行、`cli/get`/`cli/pub` 日志与沙盒文本日志中，便于排查执行来源。

示例：

```text
设置：agentId=agent-a, chatId=chat-001, mode=net
执行：agentId=agent-b, chatId=chat-001
结果：命中 net 模式，并通过 net/CLI_SANDBOX 执行
```

## 状态存储与迁移

`cli_sandbox_state` 的当前逻辑结构为：

```sql
CREATE TABLE cli_sandbox_state (
  chat_id TEXT NOT NULL PRIMARY KEY,
  sandbox_exe TEXT NOT NULL DEFAULT '',
  allowed_dir TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL
);
```

- `chat_id` 是唯一状态键。
- `sandbox_exe` 只接受 `filepick`、`net`、`filepick_net`；没有记录或关闭状态均视为 `off`。
- 关闭沙盒会删除该 `chatId` 对应的整行记录。
- 旧的表结构和旧数据不兼容本次语义。程序检测到旧表结构时会删除旧表并按上述结构重建，不迁移旧记录。

## 沙盒模式与绕过规则

- `filepick`：执行前请求用户选择目录；未选择时按权限拒绝处理。
- `net`：以禁用网络的沙盒模式执行。
- `filepick_net`：同时限制目录访问与网络。
- 任务的 `subOps.exempted=true` 时，即使该 `chatId` 已配置沙盒，也会直接走本地 Shell。
- 若已命中模式但未找到该模式的 `CLI_SANDBOX` 可执行文件，任务会以失败结果回传，不会静默降级为直连执行。

## 文本日志

成功设置、变更或关闭沙盒状态时，`cli-get` 会向标准错误输出类似日志：

```text
cli-get: sandbox change agentId=agent-a chatId=chat-001 from=off to=net
cli-get: sandbox change agentId=agent-b chatId=chat-001 from=net to=off
```

命中沙盒状态并执行任务时，会输出当前命中的模式：

```text
cli-get: sandbox lookup agentId=agent-b chatId=chat-001 mode=net allowedDir=
```

这些日志仅用于文本可观测性；不会新增查询接口或额外日志表。

## 对使用方的影响

- 调用设置沙盒状态的接口时必须传入非空 `chatId`；空值会返回错误。
- 不要再以 `agentId` 隔离同一会话的沙盒模式。若需要独立模式，请使用不同的 `chatId`。
- 若升级后已有旧版状态表，首次初始化会清空旧表；需要的沙盒模式应按新的 `chatId` 维度重新设置。
