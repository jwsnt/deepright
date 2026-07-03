# Email Iteration 20260612-1 User Guide

本次迭代补齐了 `email` 插件页“复用当前会话”的运行时会话语义，重点只有两条：

- 勾选“复用当前会话”并保存后，`email start` 只会沿用当前页面保存的 `chatId`
- 未勾选“复用当前会话”时，`email start` 会把空的 `chatId` 自动补成固定值 `email`

`agentId` 不参与复用绑定，始终以页面当前选择值为准。

## 使用说明

通过插件页保存配置后，正常启动邮件插件即可：

```bash
./integration plugins start --name email
```

或直接执行插件：

```bash
../plugins/email start --connect-bin ../integration/integration
```

说明：

- 如果插件配置中的 `chatId` 已经是当前会话值，启动时会直接复用，不会覆盖
- 如果插件配置中的 `chatId` 为空，启动时会自动执行一次 `meta-update`，把 `chatId` 校正为 `email`
- 后续邮件插件调用 `add-request` 创建备忘录任务明细时，会使用校正后的 `chatId`
- `agentId` 始终保持页面选择值，不会因为勾选“复用当前会话”而自动切换或锁定
