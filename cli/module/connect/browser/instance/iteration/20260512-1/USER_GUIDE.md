# browser_instance / browser 迭代 20260512-1 使用手册

## 变更说明

本次迭代聚焦 Browser 关闭链路的资源回收语义，要求在 `shutdown/stop` 后不只是发出停止信号，而是要同步等待并完成 runtime 级清理。

目标：

- 保证 Browser daemon 拉起的 `playwright driver` 被释放
- 保证 Browser daemon 内部持有的 Playwright CDP session 被关闭
- 保证受管实例对应的 `obscura` 进程被回收
- 保证 `integration stop` 通过执行 `browser stop` 时，也能得到同样的关闭语义

## 适用范围

这次迭代同时影响两层：

- `browser_instance shutdown`
- `browser stop`

它们的职责不同：

- `browser_instance shutdown`：负责实例级 `obscura` 进程关闭与状态删除
- `browser stop`：负责插件级 Playwright daemon 停止、同步等待和 runtime 级清理

## browser_instance shutdown

命令：

```bash
./browser_instance shutdown --agentId demo-agent --chatId chat-001
```

行为：

- 按 `agentId + chatId` 找到对应实例
- 优雅结束实例进程，必要时强制终止
- 删除对应状态记录
- 目标是同步回收该实例绑定的 `obscura` 进程

注意：

- 该命令不负责 Playwright daemon
- 该命令不负责 Playwright driver
- 该命令不负责 Browser 插件内部 session/runtime 的关闭

## browser stop

命令：

```bash
./browser stop
./browser stop --connect-bin /path/to/integration
```

新增语义：

- 先清理 `browser_instance.json` 中全部受管实例
- 再向 `browser.pid` 对应的 Playwright daemon 发出停止信号
- 不再是“发完信号立即返回”，而是会同步等待 daemon 真正退出
- daemon 退出阶段会主动关闭全部 Playwright session
- daemon 退出阶段会继续执行 Playwright runtime 的 stop，确保 `playwright driver` 被释放
- 必要时会升级为更强的终止方式，避免后台残留

结果要求：

- `browser stop` 返回成功时，不应只意味着 pid 文件消失
- 还应意味着 daemon 拉起的 `playwright driver`、CDP session、`obscura` 都已经完成清理

## integration stop

`integration stop` 关闭 Browser 插件时，调用方式仍然是执行插件 CLI：

```bash
plugins/browser stop --connect-bin /path/to/integration
```

也就是说：

- `integration` 不会直接调用 Browser 内部 Go 函数
- Browser 是否关闭干净，取决于 `browser stop` 这条 CLI 的实现
- 本次迭代补齐后，`integration stop` 也会继承同样的同步等待与 runtime 清理语义

## 推荐验收

```bash
ps aux | grep browser
./browser stop
ps aux | grep browser
```

或在统一插件场景下：

```bash
ps aux | grep browser
./integration stop
ps aux | grep browser
```

重点检查：

- `browser __daemon` 已退出
- `playwright/driver/node ... cli.js run-driver` 已退出
- 对应 `obscura serve --port ... --stealth` 已退出
- 不再残留旧的 `./browser create`、`./browser daemon stop` 等僵住进程

## 手册索引

完整说明请结合以下文档一起阅读：

- `/path/to/deepright/cli/module/connect/browser/USER_GUIDE.md`
- `/path/to/deepright/cli/module/connect/browser/instance/USER_GUIDE.md`
- `/path/to/deepright/cli/module/connect/browser/instance/iteration/20260512-1/REQUIREMENT.md`
